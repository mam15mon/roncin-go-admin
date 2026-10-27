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
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

func TestArchivedChineseNames(t *testing.T) {
	rows, _, summary, err := parseShippingLines(filepath.Join("..", "..", "seeds", "SMDG_Liner-codes-list-20260903.xlsx"), "test-hash")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "seeds", "shipping-lines.zh-CN.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err = attachChineseNames(rows, raw, &summary)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ChineseNames != 360 || summary.ExistingNames+summary.TranslatedNames != 360 || len(summary.ChineseHash) != 64 {
		t.Fatalf("归档中文覆盖或摘要不完整：%+v", summary)
	}
	want := map[string]string{
		"COS": "中远海运集运", "MSK": "马士基航运", "DJS": "东进商船", "PAN": "泛洲海运",
		"CNC": "东穗航运", "CLN": "正利航业", "SML": "海滨海运", "SLC": "森罗商船",
		"SMM": "长锦商船", "SNL": "中国外运集运", "G2O": "古德航运", "TI2": "TI2集装箱航运",
	}
	for _, row := range rows {
		if !validChineseName(row.NameZH) || row.NameZH == row.NameEN {
			t.Fatalf("%s 没有有效中文：%q", row.SCACCode, row.NameZH)
		}
		if name, found := want[row.SCACCode]; found && row.NameZH != name {
			t.Fatalf("%s 中文身份错配：%s，期望 %s", row.SCACCode, row.NameZH, name)
		}
	}
}

func TestChineseNamesRejectInvalidSnapshot(t *testing.T) {
	rows := []data.ShippingLineSyncRecord{{SCACCode: "MSK", NameEN: "Maersk"}, {SCACCode: "COS", NameEN: "COSCO"}}
	base := [][]string{
		{"smdg_code", "name_en", "name_zh", "name_kind", "source_ref"},
		{"MSK", "Maersk", "马士基航运", "existing", "仓库原有中文"},
		{"COS", "COSCO", "中远海运集运", "translated", "本次编制"},
	}
	for _, tc := range []struct {
		name string
		edit func([][]string) [][]string
		want string
	}{
		{"表头不符", func(v [][]string) [][]string { v[0][0] = "code"; return v }, "表头"},
		{"缺行", func(v [][]string) [][]string { return v[:2] }, "缺少 COS"},
		{"多行", func(v [][]string) [][]string {
			return append(v, []string{"XYZ", "Other", "其他航运", "translated", "编制"})
		}, "不属于当前底表"},
		{"重复代码", func(v [][]string) [][]string { return append(v, v[1]) }, "重复"},
		{"身份错配", func(v [][]string) [][]string { v[1][1] = "Different company"; return v }, "英文身份不一致"},
		{"纯英文", func(v [][]string) [][]string { v[1][2] = "Maersk"; return v }, "中文名无效"},
		{"空中文", func(v [][]string) [][]string { v[1][2] = ""; return v }, "中文名无效"},
		{"占位中文", func(v [][]string) [][]string { v[1][2] = "待补充"; return v }, "中文名无效"},
		{"首尾空格", func(v [][]string) [][]string { v[1][2] += " "; return v }, "中文名无效"},
		{"超长中文", func(v [][]string) [][]string { v[1][2] = strings.Repeat("船", 201); return v }, "中文名无效"},
		{"非法类别", func(v [][]string) [][]string { v[1][3] = "official"; return v }, "名称类别"},
		{"空来源", func(v [][]string) [][]string { v[1][4] = " "; return v }, "来源说明为空"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := make([][]string, len(base))
			for i := range base {
				values[i] = append([]string(nil), base[i]...)
			}
			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			if err := w.WriteAll(tc.edit(values)); err != nil {
				t.Fatal(err)
			}
			if _, err := attachChineseNames(rows, buf.Bytes(), &shippingLineParseSummary{}); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("应拒绝 %s，得到 %v", tc.name, err)
			}
			if rows[0].NameZH != "" || rows[1].NameZH != "" {
				t.Fatal("校验失败不得部分修改输入")
			}
		})
	}
	for _, raw := range [][]byte{{0xff}, []byte("smdg_code,name_en,name_zh,name_kind,source_ref\nMSK,Maersk\n")} {
		if _, err := attachChineseNames(rows, raw, &shippingLineParseSummary{}); err == nil {
			t.Fatal("无效编码或列数必须失败")
		}
	}
}

func TestChineseNamesCompareEnglishIdentityWithoutLayoutWhitespace(t *testing.T) {
	rows := []data.ShippingLineSyncRecord{{SCACCode: "YQN", NameEN: "YQN LINK International Logistics (SG) \nPte. Ltd."}}
	raw := []byte("smdg_code,name_en,name_zh,name_kind,source_ref\nYQN,YQN LINK International Logistics (SG) Pte. Ltd.,运去哪国际物流,translated,本次编制\n")
	result, err := attachChineseNames(rows, raw, &shippingLineParseSummary{})
	if err != nil || result[0].NameEN != rows[0].NameEN || result[0].NameZH != "运去哪国际物流" {
		t.Fatalf("仅忽略排版空白核对身份，仍须保留底表原文：%+v err=%v", result, err)
	}
}

func TestRunChineseSeedValidation(t *testing.T) {
	options := shippingLineOptions{
		Options:       syncrunner.Options{Source: filepath.Join("..", "..", "seeds", "SMDG_Liner-codes-list-20260903.xlsx")},
		ChineseSource: filepath.Join("..", "..", "seeds", "shipping-lines.zh-CN.csv"),
	}
	t.Setenv("DATABASE_SOURCE", "")
	if err := runWithOptions(context.Background(), options); err != nil {
		t.Fatalf("完整中文数据应支持离线预览：%v", err)
	}
	options.Apply = true
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "DATABASE_SOURCE") {
		t.Fatalf("写库必须提供数据库连接：%v", err)
	}
	options.ChineseSource = filepath.Join(t.TempDir(), "incomplete.csv")
	if err := os.WriteFile(options.ChineseSource, []byte("smdg_code,name_en,name_zh,name_kind,source_ref\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_SOURCE", "postgres://invalid-connection")
	if err := runWithOptions(context.Background(), options); err == nil || !strings.Contains(err.Error(), "中文表缺少") {
		t.Fatalf("中文缺行必须在连接数据库前失败：%v", err)
	}
}

func TestSeedContainerPrefixes(t *testing.T) {
	for code, want := range map[string]string{"EMC": "EMCU", "ONE": "ONEU", "KMT": "KMTU", "MAT": "MATU"} {
		if got := resolveContainerPrefixes(code); len(got) != 1 || got[0] != want {
			t.Fatalf("%s 必须使用箱主前缀 %s，得到 %v", code, want, got)
		}
	}
	seen := make(map[string]string)
	for code, prefixes := range topContainerPrefixes {
		for _, prefix := range prefixes {
			if !prefixPattern.MatchString(prefix) || seen[prefix] != "" {
				t.Fatalf("%s 箱号前缀 %s 格式无效或重复归属于 %s", code, prefix, seen[prefix])
			}
			seen[prefix] = code
		}
	}
	for _, carrierCode := range []string{"EGLV", "ONEY", "KMTC", "MATS"} {
		if prefixPattern.MatchString(carrierCode) {
			t.Fatalf("承运人代码 %s 不应通过箱主前缀校验", carrierCode)
		}
	}
}
