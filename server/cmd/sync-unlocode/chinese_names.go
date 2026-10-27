package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/roncin/roncin-go-admin/server/internal/data"
)

var chineseColumns = []string{"un_locode", "name_en", "name_zh", "name_kind", "source_ref"}

func sourceFingerprint(raw []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// attachChineseNames 要求中文对照与本版海港集合完全一致，不复用错版名称或在线补译。
func attachChineseNames(ports []data.PortSyncRecord, raw, zipRaw []byte, summary *unlocodeParseSummary) ([]data.PortSyncRecord, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("中文 CSV 必须使用 UTF-8 编码")
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = len(chineseColumns)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("读取中文 CSV 表头: %w", err)
	}
	if !slices.Equal(header, chineseColumns) {
		return nil, fmt.Errorf("中文 CSV 表头必须为 %s", strings.Join(chineseColumns, ","))
	}
	indices := make(map[string]int, len(ports))
	for index, row := range ports {
		indices[row.UNLocode] = index
	}
	result := slices.Clone(ports)
	seen := make(map[string]struct{}, len(ports))
	existing, translated := 0, 0
	for line := 2; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取中文 CSV 第 %d 行: %w", line, err)
		}
		code, nameEN, nameZH, kind, source := record[0], record[1], record[2], record[3], record[4]
		if !unlocodePattern.MatchString(code) {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %q 不符合 UN/LOCODE 格式", line, code)
		}
		if _, duplicate := seen[code]; duplicate {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 重复", line, code)
		}
		index, exists := indices[code]
		if !exists {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 不属于当前海港集合", line, code)
		}
		if nameEN != ports[index].NameEN {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 英文身份不一致：对照 %q，底表 %q", line, code, nameEN, ports[index].NameEN)
		}
		if !validChineseName(nameZH, code) {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 中文名无效：必须为 1—200 字、包含汉字且不是占位值", line, code)
		}
		if strings.TrimSpace(source) == "" {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 来源说明为空", line, code)
		}
		switch kind {
		case "existing":
			existing++
		case "translated":
			translated++
		default:
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 名称类别 %q 非法，仅允许 existing/translated", line, code, kind)
		}
		seen[code] = struct{}{}
		result[index].NameZH = nameZH
	}
	for _, row := range ports {
		if _, exists := seen[row.UNLocode]; !exists {
			return nil, fmt.Errorf("中文 CSV 缺少海港 %s（%s），已覆盖 %d/%d", row.UNLocode, row.NameEN, len(seen), len(ports))
		}
	}
	summary.ZIPHash = sourceFingerprint(zipRaw)
	summary.ChineseHash = sourceFingerprint(raw)
	// 固定版本标记和两个定长摘要，避免拼接边界歧义并让任一源文件修订改变指纹。
	summary.SourceHash = sourceFingerprint([]byte("unlocode-zh-v1\n" + summary.ZIPHash + "\n" + summary.ChineseHash + "\n"))
	summary.ChineseNames, summary.ExistingNames, summary.TranslatedNames = len(seen), existing, translated
	return result, nil
}

func validChineseName(name, code string) bool {
	if name == "" || name != strings.TrimSpace(name) || utf8.RuneCountInString(name) > 200 || strings.Contains(name, code) {
		return false
	}
	switch name {
	case "未知", "未知港口", "待翻译", "待补充", "暂无", "无中文名", "未命名", "港", "港口", "码头", "港区":
		return false
	}
	// 原文仅加“港”不能构成中文地名；真实码头编号（如 B12码头）仍可保留。
	chinese := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Han, r) {
			return r
		}
		return -1
	}, name)
	return chinese != "" && chinese != "港" && chinese != "港口"
}
