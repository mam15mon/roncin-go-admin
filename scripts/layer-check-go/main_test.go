package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const fixtureModule = "example.com/test/server"

// buildServer 在临时目录构造一个 mini server 模块；files 的键是相对路径。
func buildServer(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		absolute := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, []byte(files[path]), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func violationsOf(t *testing.T, files map[string]string) []Violation {
	t.Helper()
	violations, _, err := CheckServer(buildServer(t, files))
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	return violations
}

func hasViolation(violations []Violation, rule, substring string) bool {
	for _, v := range violations {
		if v.Rule == rule && strings.Contains(v.Detail, substring) {
			return true
		}
	}
	return false
}

func TestLayerBoundaries(t *testing.T) {
	files := map[string]string{
		"go.mod": "module " + fixtureModule + "\n\ngo 1.23\n",
		// service 禁 data 与 server，允许 biz/access/platform/security/api。
		"internal/service/a.go":    "package service\nimport (\n_ \"" + fixtureModule + "/internal/biz\"\n_ \"" + fixtureModule + "/internal/access\"\norderv1 \"" + fixtureModule + "/api/order/v1\"\n)\nvar _ = orderv1.Order{}\n",
		"internal/service/bad.go":  "package service\nimport _ \"" + fixtureModule + "/internal/data\"\n",
		"internal/service/bad2.go": "package service\nimport _ \"" + fixtureModule + "/internal/server\"\n",
		// biz 禁 service/data/server/platform，api 仅 ErrorReason。
		"internal/biz/a.go":     "package biz\nimport (\n_ \"" + fixtureModule + "/internal/access\"\nfinancev1 \"" + fixtureModule + "/api/finance/v1\"\n)\nvar _ = financev1.ErrorReason_ERROR_REASON_X\nvar _ = reason(financev1.ErrorReason)\n",
		"internal/biz/bad.go":   "package biz\nimport _ \"" + fixtureModule + "/internal/data\"\n",
		"internal/biz/bad2.go":  "package biz\nimport _ \"" + fixtureModule + "/internal/platform\"\n",
		"internal/biz/leak.go":  "package biz\nimport financev1 \"" + fixtureModule + "/api/finance/v1\"\nvar _ = financev1.Bill{}\n",
		"internal/biz/leak2.go": "package biz\nimport \"" + fixtureModule + "/api/order/v1\"\nvar _ = v1.Order{}\n",
		// data 禁 service/server，允许 biz、自身 ent 与 access/conf/platform 工具层。
		"internal/data/a.go":   "package data\nimport (\n_ \"" + fixtureModule + "/internal/biz\"\n_ \"" + fixtureModule + "/internal/data/ent\"\n_ \"" + fixtureModule + "/internal/access\"\n_ \"" + fixtureModule + "/internal/conf\"\n_ \"" + fixtureModule + "/internal/platform/searchtext\"\n)\n",
		"internal/data/bad.go": "package data\nimport _ \"" + fixtureModule + "/internal/service\"\n",
		// server 禁 data（含 data/ent 前缀匹配），允许 service/biz/conf/api。
		"internal/server/a.go":   "package server\nimport (\n_ \"" + fixtureModule + "/internal/service\"\n_ \"" + fixtureModule + "/internal/conf\"\n)\n",
		"internal/server/bad.go": "package server\nimport _ \"" + fixtureModule + "/internal/data/ent\"\n",
		// platform 禁 data；_test.go 豁免。
		"internal/platform/a.go":      "package platform\nimport _ \"" + fixtureModule + "/internal/conf\"\n",
		"internal/platform/bad.go":    "package platform\nimport _ \"" + fixtureModule + "/internal/data\"\n",
		"internal/platform/a_test.go": "package platform\nimport _ \"" + fixtureModule + "/internal/data\"\n",
		// 叶子层禁止任何 internal 上层。
		"internal/access/bad.go":  "package access\nimport _ \"" + fixtureModule + "/internal/biz\"\n",
		"internal/security/a.go":  "package security\n",
		"internal/conf/a.go":      "package conf\n",
		"internal/webassets/a.go": "package webassets\n",
		// cmd 组装根豁免：随意导入不产生违规。
		"cmd/server/main.go": "package main\nimport (\n_ \"" + fixtureModule + "/internal/data\"\n_ \"" + fixtureModule + "/internal/biz\"\n)\nfunc main() {}\n",
		// api 包本体不扫描。
		"api/order/v1/order.go":     "package v1\n",
		"api/finance/v1/finance.go": "package v1\n",
	}
	violations := violationsOf(t, files)
	cases := []struct{ rule, detail string }{
		{"internal/service 越界依赖", "internal/data"},
		{"internal/service 越界依赖", "internal/server"},
		{"internal/biz 越界依赖", "internal/data"},
		{"internal/biz 越界依赖", "internal/platform"},
		{"internal/biz 泄漏契约类型", "financev1.Bill"},
		{"internal/biz 泄漏契约类型", "v1.Order"},
		{"internal/data 越界依赖", "internal/service"},
		{"internal/server 越界依赖", "internal/data/ent"},
		{"internal/platform 越界依赖", "internal/data"},
		{"internal/access 越界依赖", "internal/biz"},
	}
	for _, c := range cases {
		if !hasViolation(violations, c.rule, c.detail) {
			t.Errorf("缺少预期违规 %s: %s；实际: %v", c.rule, c.detail, violations)
		}
	}
	for _, unexpected := range []struct{ rule, absent string }{
		{"internal/biz 泄漏契约类型", "financev1.ErrorReason"},
		{"internal/service 越界依赖", "禁止导入 internal/biz"},
	} {
		if hasViolation(violations, unexpected.rule, unexpected.absent) {
			t.Errorf("出现不该有的违规 %s 包含 %s", unexpected.rule, unexpected.absent)
		}
	}
	for _, v := range violations {
		if strings.HasPrefix(v.Path, "cmd/") || strings.HasPrefix(v.Path, "api/") {
			t.Errorf("cmd/api 不应被扫描: %v", v)
		}
		if strings.HasSuffix(v.Path, "_test.go") {
			t.Errorf("测试文件应豁免: %v", v)
		}
	}
}

func TestUnregisteredLayerAndParseError(t *testing.T) {
	violations := violationsOf(t, map[string]string{
		"go.mod":                "module " + fixtureModule + "\n\ngo 1.23\n",
		"internal/mystery/a.go": "package mystery\n",
	})
	if !hasViolation(violations, "未登记层级", "规则表") {
		t.Fatalf("新增 internal 顶层包未报未登记层级: %v", violations)
	}
	// 解析失败必须产生违规并以错误退出（CheckServer 返回 err）。
	if _, _, err := CheckServer(buildServer(t, map[string]string{
		"go.mod":              "module " + fixtureModule + "\n\ngo 1.23\n",
		"internal/biz/bad.go": "package biz\nfunc broken(",
	})); err == nil {
		t.Fatal("解析失败被吞掉")
	}
}

func TestGreenFixturePasses(t *testing.T) {
	if violations := violationsOf(t, map[string]string{
		"go.mod":                "module " + fixtureModule + "\n\ngo 1.23\n",
		"internal/service/a.go": "package service\nimport (\n_ \"" + fixtureModule + "/internal/biz\"\norderv1 \"" + fixtureModule + "/api/order/v1\"\n)\nvar _ = orderv1.Order{}\n",
		"internal/biz/a.go":     "package biz\nimport financev1 \"" + fixtureModule + "/api/finance/v1\"\nvar _ = financev1.ErrorReason_ERROR_REASON_X\n",
		"internal/data/a.go":    "package data\nimport _ \"" + fixtureModule + "/internal/data/ent\"\n",
		"api/order/v1/a.go":     "package v1\n",
		"api/finance/v1/a.go":   "package v1\n",
	}); len(violations) != 0 {
		t.Fatalf("合法分层不应有违规: %v", violations)
	}
}
