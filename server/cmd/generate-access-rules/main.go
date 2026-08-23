package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"sort"
	"strings"

	accessv1 "github.com/roncin/roncin-go-admin/server/api/access/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type generatedRule struct {
	operation      string
	mode           string
	permission     string
	scope          string
	orderOperation string
}

func main() {
	output := flag.String("output", "access_rules_gen.go", "生成文件路径")
	protoRoot := flag.String("proto-root", ".", "Buf 工作区根目录")
	flag.Parse()

	rules, err := collectRules(*protoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	content, err := render(rules)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, content, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func collectRules(protoRoot string) ([]generatedRule, error) {
	permissions := make(map[string]struct{})
	for _, permission := range access.Manifest() {
		permissions[permission.Key] = struct{}{}
	}

	command := exec.Command("buf", "build", "--as-file-descriptor-set", "-o", "-")
	command.Dir = protoRoot
	var stderr bytes.Buffer
	command.Stderr = &stderr
	descriptorBytes, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("构建 Proto 描述符: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	descriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descriptorBytes, descriptorSet); err != nil {
		return nil, fmt.Errorf("解析 Proto 描述符: %w", err)
	}

	var rules []generatedRule
	for _, file := range descriptorSet.GetFile() {
		for _, service := range file.GetService() {
			for _, method := range service.GetMethod() {
				operation := "/" + file.GetPackage() + "." + service.GetName() + "/" + method.GetName()
				rule, err := parseRule(operation, method.GetOptions(), permissions)
				if err != nil {
					return nil, err
				}
				rules = append(rules, rule)
			}
		}
	}
	sort.Slice(rules, func(i, j int) bool { return rules[i].operation < rules[j].operation })
	return rules, nil
}

func parseRule(operation string, options *descriptorpb.MethodOptions, permissions map[string]struct{}) (generatedRule, error) {
	if options == nil || !proto.HasExtension(options, accessv1.E_Rule) {
		return generatedRule{}, fmt.Errorf("RPC %s 缺少 (access.v1.rule) 声明", operation)
	}
	rule, ok := proto.GetExtension(options, accessv1.E_Rule).(*accessv1.Rule)
	if !ok || rule == nil {
		return generatedRule{}, fmt.Errorf("RPC %s 的 (access.v1.rule) 声明无效", operation)
	}

	generated := generatedRule{operation: operation}
	switch rule.GetMode() {
	case accessv1.AccessMode_ACCESS_MODE_PUBLIC:
		generated.mode = "accessModePublic"
		if rule.GetPermission() != "" || rule.GetOrderOperation() != "" || rule.GetScope() != accessv1.DataScope_DATA_SCOPE_UNSPECIFIED {
			return generatedRule{}, fmt.Errorf("RPC %s 的公开规则不能声明权限或数据范围", operation)
		}
	case accessv1.AccessMode_ACCESS_MODE_AUTHENTICATED:
		generated.mode = "accessModeAuthenticated"
		if rule.GetPermission() != "" || rule.GetOrderOperation() != "" || rule.GetScope() != accessv1.DataScope_DATA_SCOPE_UNSPECIFIED {
			return generatedRule{}, fmt.Errorf("RPC %s 的仅登录规则不能声明权限或数据范围", operation)
		}
	case accessv1.AccessMode_ACCESS_MODE_PERMISSION:
		generated.mode = "accessModePermission"
		generated.permission = rule.GetPermission()
		if _, ok := permissions[generated.permission]; !ok {
			return generatedRule{}, fmt.Errorf("RPC %s 声明了权限清单中不存在的权限 %q", operation, generated.permission)
		}
		if rule.GetOrderOperation() != "" {
			return generatedRule{}, fmt.Errorf("RPC %s 的普通权限规则不能声明订单操作", operation)
		}
	case accessv1.AccessMode_ACCESS_MODE_ORDER_PERMISSION:
		generated.mode = "accessModeOrderPermission"
		generated.orderOperation = rule.GetOrderOperation()
		if !access.OrderOperation(generated.orderOperation).Valid() {
			return generatedRule{}, fmt.Errorf("RPC %s 声明了无效的订单操作 %q", operation, generated.orderOperation)
		}
		if rule.GetPermission() != "" {
			return generatedRule{}, fmt.Errorf("RPC %s 的订单权限规则不能声明普通权限", operation)
		}
	default:
		return generatedRule{}, fmt.Errorf("RPC %s 未声明有效的访问控制模式", operation)
	}

	if generated.mode == "accessModePermission" || generated.mode == "accessModeOrderPermission" {
		scope, err := dataScopeName(rule.GetScope())
		if err != nil {
			return generatedRule{}, fmt.Errorf("RPC %s: %w", operation, err)
		}
		generated.scope = scope
	}
	return generated, nil
}

func dataScopeName(scope accessv1.DataScope) (string, error) {
	switch scope {
	case accessv1.DataScope_DATA_SCOPE_ALL:
		return "biz.DataScopeAll", nil
	case accessv1.DataScope_DATA_SCOPE_ORGANIZATION:
		return "biz.DataScopeOrganization", nil
	case accessv1.DataScope_DATA_SCOPE_ORGANIZATION_TREE:
		return "biz.DataScopeOrganizationTree", nil
	case accessv1.DataScope_DATA_SCOPE_SELF:
		return "biz.DataScopeSelf", nil
	default:
		return "", fmt.Errorf("权限规则必须声明有效的数据范围")
	}
}

func render(rules []generatedRule) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("// Code generated by generate-access-rules. DO NOT EDIT.\n\n")
	buffer.WriteString("package server\n\n")
	buffer.WriteString("import (\n")
	buffer.WriteString("\t\"github.com/roncin/roncin-go-admin/server/internal/access\"\n")
	buffer.WriteString("\t\"github.com/roncin/roncin-go-admin/server/internal/biz\"\n")
	buffer.WriteString(")\n\n")
	buffer.WriteString("var operationAccessRules = map[string]accessRule{\n")
	for _, rule := range rules {
		fmt.Fprintf(&buffer, "\t%q: {mode: %s", rule.operation, rule.mode)
		if rule.permission != "" {
			fmt.Fprintf(&buffer, ", permission: %q", rule.permission)
		}
		if rule.scope != "" {
			fmt.Fprintf(&buffer, ", scope: %s", rule.scope)
		}
		if rule.orderOperation != "" {
			fmt.Fprintf(&buffer, ", orderOperation: access.OrderOperation(%q)", rule.orderOperation)
		}
		buffer.WriteString("},\n")
	}
	buffer.WriteString("}\n")

	content, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("格式化生成代码: %w\n%s", err, strings.TrimSpace(buffer.String()))
	}
	return content, nil
}
