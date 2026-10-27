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

// attachChineseNames 将完整中文快照绑定到同码、同英文身份的本版船公司集合。
func attachChineseNames(rows []data.ShippingLineSyncRecord, raw []byte, summary *shippingLineParseSummary) ([]data.ShippingLineSyncRecord, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("船公司中文表必须使用 UTF-8 编码")
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	columns := []string{"smdg_code", "name_en", "name_zh", "name_kind", "source_ref"}
	reader.FieldsPerRecord = len(columns)
	header, err := reader.Read()
	if err != nil || !slices.Equal(header, columns) {
		return nil, fmt.Errorf("船公司中文表表头必须为 %s", strings.Join(columns, ","))
	}
	byCode := make(map[string]int, len(rows))
	for i, row := range rows {
		byCode[row.SCACCode] = i
	}
	result := slices.Clone(rows)
	seen := make(map[string]bool, len(rows))
	existing, translated := 0, 0
	for {
		fields, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取船公司中文表失败: %w", err)
		}
		line, _ := reader.FieldPos(0)
		code, nameEN, nameZH, kind, source := fields[0], fields[1], fields[2], fields[3], fields[4]
		index, found := byCode[code]
		switch {
		case !smdgLinerCodePattern.MatchString(code) || !found:
			return nil, fmt.Errorf("船公司中文表第 %d 行代码 %q 不属于当前底表", line, code)
		case seen[code]:
			return nil, fmt.Errorf("船公司中文表第 %d 行代码 %s 重复", line, code)
		case strings.Join(strings.Fields(nameEN), " ") != strings.Join(strings.Fields(rows[index].NameEN), " "):
			return nil, fmt.Errorf("船公司中文表第 %d 行 %s 英文身份不一致：%q 与 %q", line, code, nameEN, rows[index].NameEN)
		case !validChineseName(nameZH):
			return nil, fmt.Errorf("船公司中文表第 %d 行 %s 中文名无效", line, code)
		case strings.TrimSpace(source) == "":
			return nil, fmt.Errorf("船公司中文表第 %d 行 %s 来源说明为空", line, code)
		}
		switch kind {
		case "existing":
			existing++
		case "translated":
			translated++
		default:
			return nil, fmt.Errorf("船公司中文表第 %d 行 %s 名称类别必须为 existing 或 translated", line, code)
		}
		seen[code] = true
		result[index].NameZH = nameZH
	}
	for _, row := range rows {
		if !seen[row.SCACCode] {
			return nil, fmt.Errorf("船公司中文表缺少 %s（%s），已覆盖 %d/%d", row.SCACCode, row.NameEN, len(seen), len(rows))
		}
	}
	summary.ChineseHash = fmt.Sprintf("%x", sha256.Sum256(raw))
	summary.ChineseNames, summary.ExistingNames, summary.TranslatedNames = len(seen), existing, translated
	return result, nil
}

func validChineseName(name string) bool {
	if name == "" || name != strings.TrimSpace(name) || utf8.RuneCountInString(name) > 200 {
		return false
	}
	switch name {
	case "未知", "未知船公司", "待翻译", "待补充", "暂无", "无中文名", "未命名", "船公司", "航运", "船务":
		return false
	}
	return strings.ContainsFunc(name, func(r rune) bool { return unicode.Is(unicode.Han, r) })
}
