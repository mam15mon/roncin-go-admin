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

var chineseColumns = []string{"iata_code", "name_en", "name_zh", "name_kind", "source_ref"}

func sourceFingerprint(raw []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// attachChineseNames 要求中文对照与解析后的航司集合及英文身份完全一致，拒绝错版与在线补译。
func attachChineseNames(airlines []data.AirlineSyncRecord, raw, sourceRaw []byte, summary *airlineParseSummary) ([]data.AirlineSyncRecord, error) {
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
	indices := make(map[string]int, len(airlines))
	for index, row := range airlines {
		indices[row.IATACode] = index
	}
	result := slices.Clone(airlines)
	seen := make(map[string]struct{}, len(airlines))
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
		if !iataPattern.MatchString(code) {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %q 不符合 IATA 格式", line, code)
		}
		if _, duplicate := seen[code]; duplicate {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 重复", line, code)
		}
		index, exists := indices[code]
		if !exists {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 不属于当前航司集合", line, code)
		}
		if nameEN != airlines[index].NameEN {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 英文身份不一致：对照 %q，底表 %q", line, code, nameEN, airlines[index].NameEN)
		}
		if !validChineseName(nameZH, code) {
			return nil, fmt.Errorf("中文 CSV 第 %d 行代码 %s 中文名无效：必须为 1—200 字、包含有效汉字且不是代码或英文占位值", line, code)
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
		result[index].NameZH = &nameZH
	}
	for _, row := range airlines {
		if _, exists := seen[row.IATACode]; !exists {
			return nil, fmt.Errorf("中文 CSV 缺少航司 %s（%s），已覆盖 %d/%d", row.IATACode, row.NameEN, len(seen), len(airlines))
		}
	}
	summary.RawHash = sourceFingerprint(sourceRaw)
	summary.ChineseHash = sourceFingerprint(raw)
	summary.SourceHash = sourceFingerprint([]byte("airlines-zh-v1\n" + summary.RawHash + "\n" + summary.ChineseHash + "\n"))
	summary.EnrichedChinese, summary.ExistingNames, summary.TranslatedNames = len(seen), existing, translated
	return result, nil
}

func validChineseName(name, code string) bool {
	if name == "" || name != strings.TrimSpace(name) || utf8.RuneCountInString(name) > 200 || strings.Contains(name, code) {
		return false
	}
	switch name {
	case "未知", "未知航空公司", "待翻译", "待补充", "暂无", "无中文名", "未命名":
		return false
	}
	chinese := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Han, r) {
			return r
		}
		return -1
	}, name)
	switch chinese {
	case "", "航空", "航空公司", "航空集团", "航空运输", "航空服务", "货运", "货运航空", "货运航空公司":
		return false
	}
	return true
}
