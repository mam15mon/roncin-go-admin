package searchtext

import (
	"strings"
	"testing"
)

func TestBuildIncludesFullPinyinAndInitials(t *testing.T) {
	value := Build("上海港", "浦东国际机场")
	for _, expected := range []string{"SHANGHAIGANG", "SHANG HAI GANG", "SHG", "PUDONGGUOJIJICHANG", "PDGJJC"} {
		if !strings.Contains(value, expected) {
			t.Fatalf("Build() = %q, missing %q", value, expected)
		}
	}
}

func TestBuildPreservesNormalizedNonChineseValue(t *testing.T) {
	if value := Build(" pvg ", ""); value != "PVG" {
		t.Fatalf("Build() = %q, want PVG", value)
	}
}

func TestBuildUsesPhraseReadings(t *testing.T) {
	for _, test := range []struct{ name, full, initials string }{
		{"重庆银行", "CHONGQINGYINHANG", "CQYH"},
		{"闵行新区", "MINHANGXINQU", "MHXQ"},
		{"长春港", "CHANGCHUNGANG", "CCG"},
		{"重量", "ZHONGLIANG", "ZL"},
		{"还箱费", "HUANXIANGFEI", "HXF"},
		{"还柜费", "HUANGUIFEI", "HGF"},
	} {
		got := Build(test.name)
		if !strings.Contains(got, test.full) || !strings.Contains(got, test.initials) {
			t.Fatalf("%s 的词组读音错误: %s", test.name, got)
		}
	}
}
