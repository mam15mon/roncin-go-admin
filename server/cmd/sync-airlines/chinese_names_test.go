package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestAttachChineseNamesValidatesCompleteAirlineIdentity(t *testing.T) {
	t.Parallel()
	previousName := "旧的中文名"
	airlines := []data.AirlineSyncRecord{{IATACode: "CA", NameEN: "Air China", NameZH: &previousName, CountryCode: "CN", Enabled: true}}
	valid := []string{"CA", "Air China", "中国国际航空", "existing", "项目中文词典"}
	for _, test := range []struct {
		name string
		edit func([]string) [][]string
		want string
	}{
		{"缺行", func(row []string) [][]string { return nil }, "缺少航司"},
		{"多行", func(row []string) [][]string {
			extra := append([]string(nil), row...)
			extra[0] = "ZZ"
			return [][]string{row, extra}
		}, "不属于当前航司集合"},
		{"重复", func(row []string) [][]string { return [][]string{row, row} }, "重复"},
		{"英文错配", func(row []string) [][]string { row[1] = "Air China Cargo"; return [][]string{row} }, "英文身份不一致"},
		{"空中文", func(row []string) [][]string { row[2] = ""; return [][]string{row} }, "中文名无效"},
		{"英文占位", func(row []string) [][]string { row[2] = "Air China"; return [][]string{row} }, "中文名无效"},
		{"英文加航空占位", func(row []string) [][]string { row[2] = "Air China航空公司"; return [][]string{row} }, "中文名无效"},
		{"代码占位", func(row []string) [][]string { row[2] = "CA航空"; return [][]string{row} }, "中文名无效"},
		{"未知占位", func(row []string) [][]string { row[2] = "未知航空公司"; return [][]string{row} }, "中文名无效"},
		{"中文超长", func(row []string) [][]string { row[2] = strings.Repeat("中", 201); return [][]string{row} }, "中文名无效"},
		{"首尾空白", func(row []string) [][]string { row[2] = " 中国国际航空 "; return [][]string{row} }, "中文名无效"},
		{"非法类别", func(row []string) [][]string { row[3] = "official"; return [][]string{row} }, "名称类别"},
		{"空来源", func(row []string) [][]string { row[4] = " "; return [][]string{row} }, "来源说明为空"},
		{"非法代码", func(row []string) [][]string { row[0] = "ca"; return [][]string{row} }, "不符合 IATA 格式"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := attachChineseNames(airlines, chineseFixture(t, test.edit(append([]string(nil), valid...))), []byte("source"), &airlineParseSummary{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("应拒绝 %s 并提示 %q，实际 %v", test.name, test.want, err)
			}
			if *airlines[0].NameZH != previousName {
				t.Fatal("失败校验不得修改调用方中文")
			}
		})
	}
	for _, raw := range [][]byte{[]byte("invalid,header\n"), {0xff}, []byte("iata_code,name_en,name_zh,name_kind,source_ref\nCA,Air China,中国国际航空\n")} {
		if _, err := attachChineseNames(airlines, raw, nil, &airlineParseSummary{}); err == nil {
			t.Fatal("应拒绝错误表头、编码或列数")
		}
	}
}

func TestAttachChineseAirlinesTracksBothSources(t *testing.T) {
	t.Parallel()
	airlines := []data.AirlineSyncRecord{{IATACode: "CA", NameEN: "Air China", CountryCode: "CN"}, {IATACode: "ZZ", NameEN: "Example Air", CountryCode: "US"}}
	raw := chineseFixture(t, [][]string{{"ZZ", "Example Air", "示例航空", "translated", "编制译名"}, {"CA", "Air China", "中国国际航空", "existing", "项目中文词典"}})
	summary := airlineParseSummary{ValidAirlines: 2}
	rows, err := attachChineseNames(airlines, raw, []byte("source"), &summary)
	if err != nil {
		t.Fatal(err)
	}
	if *rows[0].NameZH != "中国国际航空" || *rows[1].NameZH != "示例航空" || rows[0].CountryCode != "CN" || airlines[0].NameZH != nil || summary.EnrichedChinese != 2 || summary.ExistingNames != 1 || summary.TranslatedNames != 1 {
		t.Fatalf("中文关联或摘要错误：rows=%+v summary=%+v", rows, summary)
	}
	for _, change := range []struct{ source, csv []byte }{{[]byte("source2"), raw}, {[]byte("source"), append(append([]byte(nil), raw...), '\n')}} {
		changed := airlineParseSummary{}
		if _, err := attachChineseNames(airlines, change.csv, change.source, &changed); err != nil {
			t.Fatal(err)
		}
		if changed.SourceHash == summary.SourceHash || len(changed.SourceHash) != 64 {
			t.Fatal("任一源文件变化必须改变联合指纹")
		}
	}
}

func TestAirlineOfflinePreviewAndExplicitApply(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "")
	dir := t.TempDir()
	options := airlineOptions{Options: syncrunner.Options{Source: filepath.Join(dir, "airlines.dat")}, ChineseSource: filepath.Join(dir, "custom.csv")}
	raw := []byte("24,American Airlines,\\N,AA,AAL,AMERICAN,United States,Y\n")
	rows, _, err := parseAirlines(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	records := make([][]string, 0, len(rows))
	for _, row := range rows {
		records = append(records, []string{row.IATACode, row.NameEN, *row.NameZH, "existing", "项目中文词典"})
	}
	if err := os.WriteFile(options.Source, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.ChineseSource, chineseFixture(t, records), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runWithOptions(context.Background(), options); err != nil {
		t.Fatalf("无数据库配置也应完成离线预览：%v", err)
	}
	options.Apply = true
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "DATABASE_SOURCE") {
		t.Fatalf("显式写入必须要求数据库配置：%v", err)
	}
	t.Setenv("DATABASE_SOURCE", "invalid")
	if err := os.WriteFile(options.ChineseSource, chineseFixture(t, nil), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "缺少航司") {
		t.Fatalf("必须先校验源文件：%v", err)
	}
}

func TestBundledChineseAirlinesCoverParsedRelease(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "seeds", "airlines.dat"))
	if err != nil {
		t.Fatal(err)
	}
	airlines, summary, err := parseAirlines(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	chinese, err := os.ReadFile(filepath.Join("..", "..", "seeds", "airlines.zh-CN.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := attachChineseNames(airlines, chinese, raw, &summary)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 988 || summary.EnrichedChinese != 988 || summary.ExistingNames+summary.TranslatedNames != 988 {
		t.Fatalf("配套中文必须覆盖当前全部 988 条解析航司：%+v", summary)
	}
	for _, row := range rows {
		if known, exists := knownAirlines[row.IATACode]; exists && *row.NameZH != known.NameZH {
			t.Fatalf("重点航司 %s 应保留已有中文 %q，实际 %q", row.IATACode, known.NameZH, *row.NameZH)
		}
	}
}

func TestAirlineSourceResolutionRequiresBundledArchive(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	archiveDir := filepath.Join(work, "server", "seeds")
	cacheDir := filepath.Join(root, ".cache", "master-data")
	for _, directory := range []string{work, archiveDir, cacheDir} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "airlines.dat"), []byte("obsolete cache"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	if got := resolveAirlineSource(""); got != "" {
		t.Fatalf("缺少归档不得使用其他缓存：%s", got)
	}
	if _, _, err := loadAirlineSource(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "归档") {
		t.Fatalf("未找到归档必须报错，不得自动下载：%v", err)
	}
	if err := runWithOptions(context.Background(), airlineOptions{}); err == nil || !strings.Contains(err.Error(), "归档") {
		t.Fatalf("默认路径缺档必须先报错：%v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "airlines.dat"), []byte("bundled"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := resolveAirlineSource(""); got != filepath.Join("server", "seeds", "airlines.dat") {
		t.Fatalf("根目录运行应定位仓库固定归档：%s", got)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		_, _ = writer.Write([]byte("explicit source"))
	}))
	defer server.Close()
	if err := runWithOptions(context.Background(), airlineOptions{Options: syncrunner.Options{Source: server.URL}}); err == nil || !strings.Contains(err.Error(), "chinese-source") || requests.Load() != 0 {
		t.Fatalf("远程源缺配套中文表应在联网前拒绝：requests=%d err=%v", requests.Load(), err)
	}
	raw, label, err := loadAirlineSource(context.Background(), server.URL)
	if err != nil || string(raw) != "explicit source" || label != server.URL || requests.Load() != 1 {
		t.Fatalf("只有显式 URL 才能下载：raw=%q label=%s requests=%d err=%v", raw, label, requests.Load(), err)
	}
}
