// layer-check 按 .trellis/spec/server/backend/directory-structure.md 的分层规则
// 检查 server/internal 各层的 import 边界；只读源码，不执行产品代码。
// 违规或解析失败输出明细并退出非 0，零违规退出 0。
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Violation struct {
	Path   string
	Line   int
	Rule   string
	Detail string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", v.Path, v.Line, v.Rule, v.Detail)
}

// apiSelectorAllowed 为空表示该层可自由使用 server/api 标识符；
// 非空时仅放行返回 true 的选择符（biz 的 ErrorReason 错误码契约）。
type layerRule struct {
	prefix             string
	allowInternal      []string
	allowAPI           bool
	apiSelectorAllowed func(selector string) bool
}

func errorReasonOnly(selector string) bool {
	return selector == "ErrorReason" || strings.HasPrefix(selector, "ErrorReason_")
}

// 规则以「现有仓库全绿」为准：新禁令先验证存量零违规再上（见任务 design.md）。
// 依赖方向自上而下：server/service → biz/data → access/conf/platform/security/webassets。
var layerRules = []layerRule{
	{prefix: "internal/service", allowInternal: []string{"internal/biz", "internal/access", "internal/platform", "internal/security"}, allowAPI: true},
	{prefix: "internal/biz", allowInternal: []string{"internal/access", "internal/security"}, allowAPI: true, apiSelectorAllowed: errorReasonOnly},
	{prefix: "internal/data", allowInternal: []string{"internal/biz", "internal/data", "internal/access", "internal/conf", "internal/platform"}},
	{prefix: "internal/server", allowInternal: []string{"internal/service", "internal/biz", "internal/access", "internal/platform", "internal/conf", "internal/webassets"}, allowAPI: true},
	{prefix: "internal/platform", allowInternal: []string{"internal/conf"}},
	{prefix: "internal/access"},
	{prefix: "internal/security"},
	{prefix: "internal/conf"},
	{prefix: "internal/webassets"},
}

type Stats struct {
	Files        int
	Imports      int
	SkippedTests int
}

// moduleName 从 go.mod 读取模块路径；读不到即报错，不猜测默认值。
func moduleName(serverRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(serverRoot, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod 缺少 module 声明")
}

func ruleFor(rel string) *layerRule {
	for i := range layerRules {
		if layerRules[i].prefix == rel || strings.HasPrefix(rel, layerRules[i].prefix+"/") {
			return &layerRules[i]
		}
	}
	return nil
}

func isAllowedInternal(allowed []string, target string) bool {
	for _, prefix := range allowed {
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return true
		}
	}
	return false
}

// localName 解析 import 在本文件内的可用标识符：别名优先，否则读目标包的 package 声明。
func localName(fset *token.FileSet, spec *ast.ImportSpec, serverRoot, module string, cache map[string]string) string {
	path := strings.Trim(spec.Path.Value, `"`)
	if spec.Name != nil {
		return spec.Name.Name
	}
	if name, ok := cache[path]; ok {
		return name
	}
	name := ""
	pkgs, err := parser.ParseDir(fset, packageDir(serverRoot, module, path), nil, parser.PackageClauseOnly)
	if err == nil {
		for _, pkg := range pkgs {
			name = pkg.Name
			break
		}
	}
	cache[path] = name
	return name
}

func packageDir(serverRoot, module, importPath string) string {
	if strings.HasPrefix(importPath, module+"/") {
		return filepath.Join(serverRoot, filepath.FromSlash(strings.TrimPrefix(importPath, module+"/")))
	}
	return ""
}

type checker struct {
	serverRoot string
	module     string
	fset       *token.FileSet
	pkgNames   map[string]string
}

func (c *checker) checkFile(rel string, src []byte) ([]Violation, int, error) {
	rule := ruleFor(rel)
	if rule == nil {
		return []Violation{{Path: rel, Rule: "未登记层级",
			Detail: "internal 下新增顶层包必须先在 layer-check 规则表登记层级与允许依赖"}}, 0, nil
	}
	file, err := parser.ParseFile(c.fset, rel, src, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("解析失败: %w", err)
	}
	var violations []Violation
	apiLocals := map[string]string{} // 本文件内 api 包标识符 -> import 路径
	importCount := 0
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, `"`)
		importCount++
		switch {
		case strings.HasPrefix(path, c.module+"/api/"):
			if !rule.allowAPI {
				violations = append(violations, Violation{Path: rel, Line: c.fset.Position(spec.Pos()).Line,
					Rule: rule.prefix + " 禁止 api", Detail: "传输层契约类型只允许 service/server 消费: " + path})
			} else {
				apiLocals[c.localName(spec)] = path
			}
		case strings.HasPrefix(path, c.module+"/internal/"):
			target := strings.TrimPrefix(path, c.module+"/")
			if target == rule.prefix || strings.HasPrefix(target, rule.prefix+"/") {
				continue // 层内自引用
			}
			if !isAllowedInternal(rule.allowInternal, target) {
				violations = append(violations, Violation{Path: rel, Line: c.fset.Position(spec.Pos()).Line,
					Rule: rule.prefix + " 越界依赖", Detail: "禁止导入 " + target + "；该层允许: " + strings.Join(rule.allowInternal, ", ")})
			}
		}
	}
	if rule.apiSelectorAllowed != nil {
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || apiLocals[ident.Name] == "" {
				return true
			}
			if !rule.apiSelectorAllowed(selector.Sel.Name) {
				violations = append(violations, Violation{Path: rel,
					Line:   c.fset.Position(selector.Sel.Pos()).Line,
					Rule:   rule.prefix + " 泄漏契约类型",
					Detail: ident.Name + "." + selector.Sel.Name + " 不是 ErrorReason；biz 只允许错误码关联，领域对象不得使用 Protobuf 类型"})
			}
			return true
		})
	}
	return violations, importCount, nil
}

func (c *checker) localName(spec *ast.ImportSpec) string {
	return localName(c.fset, spec, c.serverRoot, c.module, c.pkgNames)
}

// CheckServer 扫描 serverRoot 下 internal/ 的全部非测试 Go 文件。
// cmd/ 是组装根、api/ 与 configs/ 是契约与配置输入，不在扫描范围。
func CheckServer(serverRoot string) ([]Violation, Stats, error) {
	module, err := moduleName(serverRoot)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("读取 %s 的模块名: %w", serverRoot, err)
	}
	c := &checker{serverRoot: serverRoot, module: module, fset: token.NewFileSet(), pkgNames: map[string]string{}}
	var violations []Violation
	stats := Stats{}
	internalRoot := filepath.Join(serverRoot, "internal")
	if _, err := os.Stat(internalRoot); err != nil {
		return nil, stats, fmt.Errorf("缺少 internal 目录: %w", err)
	}
	var files []string
	err = filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "vendor" || entry.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, stats, err
	}
	sort.Strings(files)
	var parseErr error
	for _, path := range files {
		rel, err := filepath.Rel(serverRoot, path)
		if err != nil {
			return nil, stats, err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, "_test.go") {
			stats.SkippedTests++
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, stats, err
		}
		fileViolations, imports, err := c.checkFile(rel, src)
		if err != nil {
			violations = append(violations, Violation{Path: rel, Rule: "解析失败", Detail: err.Error()})
			parseErr = fmt.Errorf("%s: %w", rel, err)
			continue
		}
		violations = append(violations, fileViolations...)
		stats.Files++
		stats.Imports += imports
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		return violations[i].Line < violations[j].Line
	})
	return violations, stats, parseErr
}

// resolveServerRoot 显式参数优先；否则从当前目录向上查找 server/go.mod，
// 不依赖调用方的 cwd（go -C 不把目录切换传给被运行程序）。
func resolveServerRoot(arg string) (string, error) {
	if arg != "" {
		if _, err := os.Stat(filepath.Join(arg, "go.mod")); err == nil {
			return arg, nil
		}
		return "", fmt.Errorf("%s 下没有 go.mod", arg)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "server", "go.mod")); err == nil {
			return filepath.Join(dir, "server"), nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			return "", fmt.Errorf("从 %s 向上未找到 server/go.mod，请显式传入 server 根目录", cwd)
		}
	}
}

func main() {
	serverRoot, err := resolveServerRoot("")
	if len(os.Args) > 1 {
		serverRoot, err = resolveServerRoot(os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "分层检查失败: %v\n", err)
		os.Exit(1)
	}
	violations, stats, err := CheckServer(serverRoot)
	for _, v := range violations {
		fmt.Fprintln(os.Stderr, v.String())
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "分层检查失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已检查 %d 个 internal 文件、%d 条 import（跳过 %d 个测试文件）；违规 %d 处。\n",
		stats.Files, stats.Imports, stats.SkippedTests, len(violations))
	if len(violations) > 0 {
		os.Exit(1)
	}
}
