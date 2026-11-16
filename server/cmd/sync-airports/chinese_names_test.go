package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/cmd/internal/syncrunner"
)

const airportFixture = "iata_code,gps_code,name,municipality,iso_country,type\nPVG,ZSPD,Shanghai Pudong International Airport,Shanghai,CN,large_airport\n"

func chineseFixture(t *testing.T, records ...[]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.WriteAll(append([][]string{chineseColumns}, records...)); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func validChineseRecord() []string {
	return []string{"PVG", "ZSPD", "CN", "Shanghai Pudong International Airport", "Shanghai", "上海浦东国际机场", "existing", "https://www.wikidata.org/wiki/Q36420"}
}

func TestAttachChineseNamesAndFingerprint(t *testing.T) {
	t.Parallel()
	raw := []byte(airportFixture)
	rows, summary, err := parseAirports(raw)
	if err != nil {
		t.Fatal(err)
	}
	chineseRaw := chineseFixture(t, validChineseRecord())
	attached, err := attachChineseNames(rows, chineseRaw, raw, &summary)
	if err != nil {
		t.Fatal(err)
	}
	if attached[0].NameZH != "上海浦东国际机场" || attached[0].IATACode != "PVG" || attached[0].NameEN != rows[0].NameEN || *attached[0].CityNameEN != "Shanghai" {
		t.Fatalf("中文附加不应改变代码、英文名或城市身份: %+v", attached)
	}
	if rows[0].NameZH != "" || summary.ChineseNames != 1 || summary.ExistingNames != 1 || summary.TranslatedNames != 0 {
		t.Fatalf("原输入不应修改且覆盖统计应准确: rows=%+v summary=%+v", rows, summary)
	}
	if summary.AirportsHash != sourceFingerprint(raw) || summary.ChineseHash != sourceFingerprint(chineseRaw) || summary.SourceHash == summary.AirportsHash {
		t.Fatalf("必须保留双源摘要并生成联合指纹: %+v", summary)
	}
	before := summary.SourceHash
	changed := validChineseRecord()
	changed[5], changed[6] = "上海浦东机场", "translated"
	if _, err := attachChineseNames(rows, chineseFixture(t, changed), raw, &summary); err != nil {
		t.Fatal(err)
	}
	if before == summary.SourceHash || summary.TranslatedNames != 1 {
		t.Fatal("仅修订中文名称也必须改变联合指纹与类别统计")
	}
	before = summary.SourceHash
	if _, err := attachChineseNames(rows, chineseFixture(t, changed), append(raw, '\n'), &summary); err != nil {
		t.Fatal(err)
	}
	if before == summary.SourceHash {
		t.Fatal("底表内容变化必须改变联合指纹")
	}
}

func TestAttachChineseNamesRejectsMismatches(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		edit func([]string) []string
	}{
		{"代码格式", func(r []string) []string { r[0] = "pvg"; return r }},
		{"错码", func(r []string) []string { r[0] = "SHA"; return r }},
		{"ICAO错配", func(r []string) []string { r[1] = "ZSSS"; return r }},
		{"国家错配", func(r []string) []string { r[2] = "US"; return r }},
		{"英文机场更名", func(r []string) []string { r[3] = "Old Airport"; return r }},
		{"英文城市错配", func(r []string) []string { r[4] = "Beijing"; return r }},
		{"空名称", func(r []string) []string { r[5] = ""; return r }},
		{"英文加机场", func(r []string) []string { r[5] = "Shanghai Airport机场"; return r }},
		{"英文加水上飞机基地", func(r []string) []string { r[5] = "Paris水上飞机基地"; return r }},
		{"英文加水上飞机航站", func(r []string) []string { r[5] = "Paris水上飞机航站"; return r }},
		{"英文加直升机停机坪", func(r []string) []string { r[5] = "Paris直升机停机坪"; return r }},
		{"英文加简易机场", func(r []string) []string { r[5] = "English简易机场"; return r }},
		{"英文加地区机场", func(r []string) []string { r[5] = "English地区机场"; return r }},
		{"英文加市立机场", func(r []string) []string { r[5] = "English市立机场"; return r }},
		{"英文加海军陆战队航空站", func(r []string) []string { r[5] = "English海军陆战队航空站"; return r }},
		{"代码加机场", func(r []string) []string { r[5] = "PVG机场"; return r }},
		{"纯类型", func(r []string) []string { r[5] = "国际机场"; return r }},
		{"未知占位", func(r []string) []string { r[5] = "未知机场"; return r }},
		{"首尾空格", func(r []string) []string { r[5] = " 上海浦东国际机场"; return r }},
		{"超长名称", func(r []string) []string { r[5] = strings.Repeat("沪", 201); return r }},
		{"类别错误", func(r []string) []string { r[6] = "unknown"; return r }},
		{"来源为空", func(r []string) []string { r[7] = " "; return r }},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows, summary, err := parseAirports([]byte(airportFixture))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := attachChineseNames(rows, chineseFixture(t, test.edit(validChineseRecord())), []byte(airportFixture), &summary); err == nil {
				t.Fatal("非法中文对照应显式失败")
			}
		})
	}
	rows, summary, err := parseAirports([]byte(airportFixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{
		chineseFixture(t),
		chineseFixture(t, validChineseRecord(), validChineseRecord()),
		[]byte("iata_code,name_zh\nPVG,上海机场\n"),
		[]byte{0xff, 0xfe},
	} {
		if _, err := attachChineseNames(rows, raw, []byte(airportFixture), &summary); err == nil {
			t.Fatal("缺行、重复、表头不符及非 UTF-8 输入应显式失败")
		}
	}
}

func TestValidChineseAirportNamePreservesNames(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"巴黎水上飞机基地", "巴黎水上飞机航站", "巴黎直升机停机坪",
		"阿福雷简易机场", "普雷斯科特地区机场", "圣莫尼卡市立机场",
		"上海浦东国际机场", "英国皇家空军阿克罗蒂里基地", "中山客运港",
	} {
		if !validChineseAirportName(name, "PVG") {
			t.Errorf("完整中文专名应保留: %q", name)
		}
	}
}

func TestAirportSeedChineseCoverage(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join("..", "..", "seeds", "airports.csv"))
	if err != nil {
		t.Fatal(err)
	}
	chineseRaw, err := os.ReadFile(filepath.Join("..", "..", "seeds", "airports.zh-CN.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, summary, err := parseAirports(raw)
	if err != nil || len(summary.FatalProblems) != 0 {
		t.Fatalf("归档底表应完整解析: %v %+v", err, summary)
	}
	rows, err = attachChineseNames(rows, chineseRaw, raw, &summary)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 9053 || summary.ChineseNames != len(rows) || summary.ExistingNames == 0 || summary.TranslatedNames == 0 {
		t.Fatalf("机场中文必须覆盖归档全部 IATA 集合: %+v", summary)
	}
	want := map[string]string{"PVG": "上海浦东国际机场", "PEK": "北京首都国际机场", "PKX": "北京大兴国际机场", "CAN": "广州白云国际机场", "LAX": "洛杉矶国际机场", "JFK": "约翰·肯尼迪国际机场", "LHR": "伦敦希思罗机场", "NRT": "成田国际机场", "HND": "东京羽田机场", "ICN": "仁川国际机场", "ZGN": "中山客运港"}
	for _, row := range rows {
		if name, ok := want[row.IATACode]; ok {
			if row.NameZH != name {
				t.Errorf("%s 中文名称=%q，期望 %q", row.IATACode, row.NameZH, name)
			}
			delete(want, row.IATACode)
		}
	}
	if len(want) != 0 {
		t.Fatalf("归档缺少关键机场: %v", want)
	}
}

func TestAirportOptionsAndOfflinePreview(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "")
	options, err := parseOptions(nil)
	if err != nil || options.Release != archivedRelease || filepath.Base(options.Source) != "airports.csv" || filepath.Base(options.ChineseSource) != "airports.zh-CN.csv" {
		t.Fatalf("默认应读取配套归档与版本: %+v %v", options, err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "airports.csv")
	chineseSource := filepath.Join(dir, "airports.zh-CN.csv")
	if err := os.WriteFile(source, []byte(airportFixture), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chineseSource, chineseFixture(t, validChineseRecord()), 0o600); err != nil {
		t.Fatal(err)
	}
	options, err = parseOptions([]string{"-source", source, "-release", "2026-test"})
	if err != nil || options.Source != source || options.ChineseSource != chineseSource || options.Release != "2026-test" {
		t.Fatalf("自定义底表默认使用同目录中文对照: %+v %v", options, err)
	}
	if err := runWithOptions(context.Background(), options); err != nil {
		t.Fatalf("无数据库配置时完整离线验源应成功: %v", err)
	}
	options.Apply = true
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "DATABASE_SOURCE") {
		t.Fatalf("显式写入缺少数据库配置时应失败: %v", err)
	}
	options = airportOptions{Options: syncrunner.Options{Source: source}, ChineseSource: chineseSource}
	if err := runWithOptions(context.Background(), options); err != nil {
		t.Fatalf("自定义版本缺省时应使用稳定内容摘要: %v", err)
	}
}
