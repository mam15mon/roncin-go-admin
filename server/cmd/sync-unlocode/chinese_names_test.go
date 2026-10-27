package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/cmd/internal/syncrunner"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

func chineseFixture(t *testing.T, records [][]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.WriteAll(append([][]string{chineseColumns}, records...)); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestAttachChineseNamesValidatesCompleteIdentity(t *testing.T) {
	t.Parallel()
	ports := []data.PortSyncRecord{{UNLocode: "CNSHG", NameEN: "Shanghai", CountryCode: "CN", TransportModes: []string{"SEA"}, Enabled: true}}
	valid := []string{"CNSHG", "Shanghai", "上海", "existing", "编制来源"}
	cases := []struct {
		name string
		edit func([]string) [][]string
		want string
	}{
		{"缺行", func(row []string) [][]string { return nil }, "缺少海港"},
		{"多行", func(row []string) [][]string {
			extra := append([]string(nil), row...)
			extra[0] = "USLAX"
			return [][]string{row, extra}
		}, "不属于当前海港集合"},
		{"重复", func(row []string) [][]string { return [][]string{row, row} }, "重复"},
		{"英文错配", func(row []string) [][]string { row[1] = "Shanghai Terminal"; return [][]string{row} }, "英文身份不一致"},
		{"国家错配", func(row []string) [][]string { row[0] = "USSHG"; return [][]string{row} }, "不属于当前海港集合"},
		{"空中文", func(row []string) [][]string { row[2] = ""; return [][]string{row} }, "中文名无效"},
		{"原文占位", func(row []string) [][]string { row[2] = "Shanghai"; return [][]string{row} }, "中文名无效"},
		{"原文加港占位", func(row []string) [][]string { row[2] = "Shanghai港"; return [][]string{row} }, "中文名无效"},
		{"原文加港口占位", func(row []string) [][]string { row[2] = "港口Shanghai"; return [][]string{row} }, "中文名无效"},
		{"单字港占位", func(row []string) [][]string { row[2] = "港"; return [][]string{row} }, "中文名无效"},
		{"国家代码占位", func(row []string) [][]string { row[2] = "中国CNSHG"; return [][]string{row} }, "中文名无效"},
		{"未知占位", func(row []string) [][]string { row[2] = "未知港口"; return [][]string{row} }, "中文名无效"},
		{"中文超长", func(row []string) [][]string { row[2] = strings.Repeat("港", 201); return [][]string{row} }, "中文名无效"},
		{"中文空白", func(row []string) [][]string { row[2] = " 上海 "; return [][]string{row} }, "中文名无效"},
		{"非法类别", func(row []string) [][]string { row[3] = "official"; return [][]string{row} }, "名称类别"},
		{"空来源", func(row []string) [][]string { row[4] = " "; return [][]string{row} }, "来源说明为空"},
		{"代码格式", func(row []string) [][]string { row[0] = "cnshg"; return [][]string{row} }, "不符合 UN/LOCODE 格式"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := unlocodeParseSummary{}
			_, err := attachChineseNames(ports, chineseFixture(t, tc.edit(append([]string(nil), valid...))), []byte("zip"), &summary)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("应拒绝 %s 并提示 %q，实际 %v", tc.name, tc.want, err)
			}
			if ports[0].NameZH != "" {
				t.Fatal("失败校验不得修改调用方数据")
			}
		})
	}
	for _, raw := range [][]byte{[]byte("invalid,header\n"), []byte{0xff}, []byte("un_locode,name_en,name_zh,name_kind,source_ref\nCNSHG,Shanghai,上海\n")} {
		if _, err := attachChineseNames(ports, raw, nil, &unlocodeParseSummary{}); err == nil {
			t.Fatal("应拒绝错误表头、编码或列数")
		}
	}
}

func TestAttachChineseNamesTracksBothSources(t *testing.T) {
	t.Parallel()
	ports := []data.PortSyncRecord{{UNLocode: "CNSHG", NameEN: "Shanghai", CountryCode: "CN"}, {UNLocode: "USLAX", NameEN: "Los Angeles", CountryCode: "US"}}
	records := [][]string{{"USLAX", "Los Angeles", "洛杉矶", "translated", "按英文音译"}, {"CNSHG", "Shanghai", "上海", "existing", "中文原名"}}
	raw := chineseFixture(t, records)
	summary := unlocodeParseSummary{ValidPorts: 2}
	rows, err := attachChineseNames(ports, raw, []byte("zip"), &summary)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].NameZH != "上海" || rows[1].NameZH != "洛杉矶" || rows[0].CountryCode != "CN" || ports[0].NameZH != "" || summary.ChineseNames != 2 || summary.ExistingNames != 1 || summary.TranslatedNames != 1 {
		t.Fatalf("中文关联或摘要错误：rows=%+v summary=%+v", rows, summary)
	}
	for _, change := range []struct{ zip, csv []byte }{{[]byte("zip2"), raw}, {[]byte("zip"), append(append([]byte(nil), raw...), '\n')}} {
		changed := unlocodeParseSummary{}
		if _, err := attachChineseNames(ports, change.csv, change.zip, &changed); err != nil {
			t.Fatal(err)
		}
		if changed.SourceHash == summary.SourceHash || len(changed.SourceHash) != 64 {
			t.Fatal("任一源文件变化必须改变联合指纹")
		}
	}
}

func TestUNLocodeOfflinePreviewAndExplicitApply(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "")
	dir := t.TempDir()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range partNames {
		entry, err := writer.Create("2025-1 " + name)
		if err != nil {
			t.Fatal(err)
		}
		if name == partNames[0] {
			if _, err := entry.Write([]byte(",CN,SHG,Shanghai,Shanghai,SH,1-------,AI,9501,,,\n")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	options := unlocodeOptions{Options: syncrunner.Options{Source: filepath.Join(dir, "ports.zip")}, ChineseSource: filepath.Join(dir, "custom.csv")}
	if err := os.WriteFile(options.Source, buffer.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.ChineseSource, chineseFixture(t, [][]string{{"CNSHG", "Shanghai", "上海", "existing", "中文原名"}}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runWithOptions(context.Background(), options); err != nil {
		t.Fatalf("无数据库配置也应完成离线预览：%v", err)
	}
	options.Apply = true
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "DATABASE_SOURCE") {
		t.Fatalf("显式写入必须要求数据库配置：%v", err)
	}
	// 校验必须先于数据库初始化，错误配置不能掩盖源文件错配。
	t.Setenv("DATABASE_SOURCE", "invalid")
	if err := os.WriteFile(options.ChineseSource, chineseFixture(t, nil), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "缺少海港") {
		t.Fatalf("必须先校验源文件：%v", err)
	}
}

func TestBundledChinesePortsCoverUNLocodeRelease(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "seeds", "loc251csv.zip"))
	if err != nil {
		t.Fatal(err)
	}
	files, release, err := readCodeListFiles(raw)
	if err != nil {
		t.Fatal(err)
	}
	ports, summary, err := parseUNLocode(files, raw)
	if err != nil {
		t.Fatal(err)
	}
	chinese, err := os.ReadFile(filepath.Join("..", "..", "seeds", "ports.zh-CN.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := attachChineseNames(ports, chinese, raw, &summary)
	if err != nil {
		t.Fatal(err)
	}
	if release != "2025-1" || len(rows) != 17524 || summary.ChineseNames != 17524 || len(summary.FatalProblems) != 0 {
		t.Fatalf("配套文件必须覆盖 2025-1 全部 17,524 个海港：release=%s summary=%+v", release, summary)
	}
}
