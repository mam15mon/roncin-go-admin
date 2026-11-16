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

var chineseColumns = []string{"iata_code", "icao_code", "country_code", "name_en", "city_name_en", "name_zh", "name_kind", "source_ref"}

func sourceFingerprint(raw []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// attachChineseNames 核对本版机场的代码、国别与英文身份，防止错版或历史代码复用串名。
func attachChineseNames(airports []data.AirportSyncRecord, raw, airportRaw []byte, summary *airportParseSummary) ([]data.AirportSyncRecord, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("机场中文 CSV 必须使用 UTF-8 编码")
	}
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.FieldsPerRecord = len(chineseColumns)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("读取机场中文 CSV 表头: %w", err)
	}
	if !slices.Equal(header, chineseColumns) {
		return nil, fmt.Errorf("机场中文 CSV 表头必须为 %s", strings.Join(chineseColumns, ","))
	}
	indices := make(map[string]int, len(airports))
	for index, row := range airports {
		indices[row.IATACode] = index
	}
	result := slices.Clone(airports)
	seen := make(map[string]struct{}, len(airports))
	existing, translated := 0, 0
	for line := 2; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取机场中文 CSV 第 %d 行: %w", line, err)
		}
		code, icao, country, nameEN, cityEN, nameZH, kind, source := record[0], record[1], record[2], record[3], record[4], record[5], record[6], record[7]
		if !iataPattern.MatchString(code) {
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %q 不符合 IATA 格式", line, code)
		}
		if _, exists := seen[code]; exists {
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %s 重复", line, code)
		}
		index, exists := indices[code]
		if !exists {
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %s 不属于当前机场集合", line, code)
		}
		airport := airports[index]
		var expectedICAO, expectedCity string
		if airport.ICAOCode != nil {
			expectedICAO = *airport.ICAOCode
		}
		if airport.CityNameEN != nil {
			expectedCity = *airport.CityNameEN
		}
		if icao != expectedICAO || country != airport.CountryCode || nameEN != airport.NameEN || cityEN != expectedCity {
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %s 的 ICAO、国家、英文名称或城市身份与底表不一致", line, code)
		}
		if !validChineseAirportName(nameZH, code) {
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %s 中文名无效：必须为 1—200 字、包含有意义的汉字名称且不是占位值", line, code)
		}
		if strings.TrimSpace(source) == "" {
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %s 来源说明为空", line, code)
		}
		switch kind {
		case "existing":
			existing++
		case "translated":
			translated++
		default:
			return nil, fmt.Errorf("机场中文 CSV 第 %d 行代码 %s 名称类别 %q 非法，仅允许 existing/translated", line, code, kind)
		}
		seen[code] = struct{}{}
		result[index].NameZH = nameZH
	}
	for _, airport := range airports {
		if _, exists := seen[airport.IATACode]; !exists {
			return nil, fmt.Errorf("机场中文 CSV 缺少机场 %s（%s），已覆盖 %d/%d", airport.IATACode, airport.NameEN, len(seen), len(airports))
		}
	}
	summary.AirportsHash = sourceFingerprint(airportRaw)
	summary.ChineseHash = sourceFingerprint(raw)
	summary.SourceHash = sourceFingerprint([]byte("ourairports-zh-v1\n" + summary.AirportsHash + "\n" + summary.ChineseHash + "\n"))
	summary.ChineseNames, summary.ExistingNames, summary.TranslatedNames = len(seen), existing, translated
	return result, nil
}

func validChineseAirportName(name, code string) bool {
	if name == "" || name != strings.TrimSpace(name) || utf8.RuneCountInString(name) > 200 || strings.Contains(name, code) {
		return false
	}
	switch name {
	case "未知", "未知机场", "待翻译", "待补充", "暂无", "无中文名", "未命名":
		return false
	}
	chinese := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Han, r) {
			return r
		}
		return -1
	}, name)
	// 设施后缀按长到短剔除；仅有英文专名加类型不算中文名，名称内部的专名不删除。
	for _, generic := range []string{
		"海军陆战队航空站",
		"陆军直升机场", "水上飞机航站", "水上飞机基地", "直升机停机坪", "军民合用机场",
		"海军航空站", "国际航空站",
		"国际机场", "直升机场", "水上机场", "地区机场", "市立机场", "简易机场", "军用机场", "空军基地", "海军基地", "航空基地",
		"航空站", "停机坪", "飞机场",
		"机场", "基地",
	} {
		chinese = strings.TrimSuffix(chinese, generic)
	}
	return chinese != ""
}
