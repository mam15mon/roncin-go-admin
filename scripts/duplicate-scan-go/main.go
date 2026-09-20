// duplicate-scan 只解析源码，不加载业务依赖或执行产品代码。
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
)

type Record struct {
	Language          string  `json:"language"`
	Path              string  `json:"path"`
	Symbol            string  `json:"symbol"`
	StartLine         int     `json:"startLine"`
	EndLine           int     `json:"endLine"`
	Start             int     `json:"start"`
	End               int     `json:"end"`
	Nodes             int     `json:"nodes"`
	Exact             string  `json:"exact"`
	Renamed           *string `json:"renamed"`
	RenameUnsupported *string `json:"renameUnsupported"`
}
type Input struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}
type Output struct {
	Records []Record    `json:"records"`
	Errors  []ScanError `json:"errors"`
}
type ScanError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

var posType = reflect.TypeOf(token.Pos(0))

// AST 对象绑定由 go/parser 解析；不序列化 Obj/Scope，避免引用环与位置信息。
func canonical(value reflect.Value, rename map[*ast.Object]string, protected map[*ast.Ident]bool) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return canonical(value.Elem(), rename, protected)
	}
	if value.Type() == posType {
		return nil
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if id, ok := value.Interface().(*ast.Ident); ok {
			name := id.Name
			if !protected[id] {
				if replacement, exists := rename[id.Obj]; exists {
					return map[string]any{"type": "Ident", "Binding": replacement}
				}
			}
			return map[string]any{"type": "Ident", "Name": name}
		}
		return canonical(value.Elem(), rename, protected)
	}
	switch value.Kind() {
	case reflect.Struct:
		out := map[string]any{"type": value.Type().Name()}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.Type == posType {
				if field.Name == "Ellipsis" || field.Name == "Assign" {
					out[field.Name] = value.Field(i).Int() != 0
				}
				continue
			}
			if field.Name == "Obj" || field.Name == "Scope" || field.Name == "Doc" || field.Name == "Comment" || field.Name == "Comments" {
				continue
			}
			// 分号位置影响 EmptyStmt 的隐式标记，语法结构仍保留该标记。
			out[field.Name] = canonical(value.Field(i), rename, protected)
		}
		return out
	case reflect.Slice:
		out := make([]any, value.Len())
		for i := range out {
			out[i] = canonical(value.Index(i), rename, protected)
		}
		return out
	default:
		return value.Interface()
	}
}

func extract(source, path string) ([]Record, error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}
	records := []Record{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		names := map[*ast.Object]string{}
		self := map[*ast.Object]string{}
		if fn.Name.Obj != nil {
			self[fn.Name.Obj] = "$self"
			names[fn.Name.Obj] = "$self"
		}
		protected := map[*ast.Ident]bool{}
		reason := ""
		nodes := 0
		ast.Inspect(fn, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			switch node.(type) {
			case *ast.CommentGroup, *ast.Comment:
				return false
			}
			nodes++
			switch value := node.(type) {
			case *ast.TypeSpec:
				reason = "局部类型声明暂不规范化"
			case *ast.StructType:
				reason = "匿名结构体字段暂不规范化"
			case *ast.KeyValueExpr:
				if key, ok := value.Key.(*ast.Ident); ok {
					protected[key] = true
				}
			case *ast.Ident:
				if value.Obj != nil && value.Obj.Kind == ast.Var && value.Obj.Pos() >= fn.Pos() && value.Obj.Pos() <= fn.End() {
					if _, exists := names[value.Obj]; !exists {
						names[value.Obj] = fmt.Sprintf("$local%d", len(names))
					}
				}
			}
			return true
		})
		// 方法接收者类型、函数签名与函数体均参与比较；函数声明名仅用于定位。
		shape := struct {
			Recv *ast.FieldList
			Type *ast.FuncType
			Body *ast.BlockStmt
		}{fn.Recv, fn.Type, fn.Body}
		exact, _ := json.Marshal(canonical(reflect.ValueOf(shape), self, nil))
		record := Record{Language: "go", Path: path, Symbol: fn.Name.Name, StartLine: set.Position(fn.Pos()).Line, EndLine: set.Position(fn.End()).Line, Start: set.Position(fn.Pos()).Offset, End: set.Position(fn.End()).Offset, Nodes: nodes, Exact: string(exact)}
		if fn.Recv != nil {
			record.Symbol = fmt.Sprintf("%s.%s", source[set.Position(fn.Recv.Pos()).Offset:set.Position(fn.Recv.End()).Offset], fn.Name.Name)
		}
		if reason != "" {
			record.RenameUnsupported = &reason
		} else {
			normalized, _ := json.Marshal(canonical(reflect.ValueOf(shape), names, protected))
			normalizedString := string(normalized)
			record.Renamed = &normalizedString
		}
		records = append(records, record)
	}
	return records, nil
}
func main() {
	var inputs []Input
	if err := json.NewDecoder(os.Stdin).Decode(&inputs); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output := Output{Records: []Record{}, Errors: []ScanError{}}
	for _, input := range inputs {
		records, err := extract(input.Source, input.Path)
		if err != nil {
			output.Errors = append(output.Errors, ScanError{input.Path, err.Error()})
			continue
		}
		output.Records = append(output.Records, records...)
	}
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(output.Errors) > 0 {
		os.Exit(1)
	}
}
