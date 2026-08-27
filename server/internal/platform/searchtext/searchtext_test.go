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
