package main

import "testing"

func fingerprint(t *testing.T, source string) Record {
	t.Helper()
	records, err := extract("package example\n"+source, "sample.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("函数数量 %d", len(records))
	}
	return records[0]
}
func TestExactAndLocalRename(t *testing.T) {
	a := fingerprint(t, "func first(x int) int { y := x + 1; { x := 2; use(x) }; return y }")
	b := fingerprint(t, "func second(input int) int { next := input + 1; { inner := 2; use(inner) }; return next }")
	if a.Exact == b.Exact || a.Renamed == nil || b.Renamed == nil || *a.Renamed != *b.Renamed {
		t.Fatal("绑定规范化失败")
	}
	c := fingerprint(t, "func other(x int) int { /*注释*/ y := x + 1; { x := 2; use(x) }; return y }")
	if a.Exact != c.Exact {
		t.Fatal("注释/声明名称影响 exact")
	}
}
func TestRangeAndClosure(t *testing.T) {
	a := fingerprint(t, "func a(xs []int) int { total:=0; for _, x := range xs { total += x }; f:=func() int { return total }; return f() }")
	b := fingerprint(t, "func b(items []int) int { sum:=0; for _, item := range items { sum += item }; run:=func() int { return sum }; return run() }")
	if *a.Renamed != *b.Renamed {
		t.Fatal("range/闭包未按绑定规范化")
	}
}
func TestMeaningfulDifferences(t *testing.T) {
	cases := [][2]string{
		{"func a(x int) int { return x+1 }", "func b(y int) int { return y+2 }"},
		{"func a(x int) int { return x+1 }", "func b(y int) int { return y-1 }"},
		{"func a(x int) int { return api(x) }", "func b(y int) int { return other(y) }"},
		{"func a(x Item) int { return x.Name }", "func b(y Item) int { return y.Code }"},
		{"func a(x int) Item { return Item{Name:x} }", "func b(y int) Item { return Item{Code:y} }"},
		{"func a(x int) int { f:=func() int {return x};return f() }", "func b(y int) int { f:=func() int {return x};return f() }"},
		{"func a(x int) int { { x:=1; return x } }", "func b(y int) int { { x:=1; return y } }"},
	}
	for _, pair := range cases {
		a := fingerprint(t, pair[0])
		b := fingerprint(t, pair[1])
		if *a.Renamed == *b.Renamed {
			t.Fatalf("关键差异被抹除: %s", pair[0])
		}
	}
}
func TestMethodsTypesAndErrors(t *testing.T) {
	r := fingerprint(t, "func (s Schema) Fields() []Field { return []Field{field.String(\"name\")} }")
	if r.Symbol != "(s Schema).Fields" {
		t.Fatal(r.Symbol)
	}
	r = fingerprint(t, "func a() { type Local struct { Name string }; use(Local{}) }")
	if r.Renamed != nil || r.RenameUnsupported == nil {
		t.Fatal("复杂绑定必须 exact-only")
	}
	if _, err := extract("package example\nfunc broken(", "bad.go"); err == nil {
		t.Fatal("解析错误被吞掉")
	}
	records, err := extract("package example\nfunc external()", "decl.go")
	if err != nil || len(records) != 0 {
		t.Fatal("无函数体应排除")
	}
}

func TestSemanticPositions(t *testing.T) {
	a := fingerprint(t, "func a(xs []int) { f(xs...) }")
	b := fingerprint(t, "func b(xs []int) { f(xs) }")
	if a.Exact == b.Exact || *a.Renamed == *b.Renamed {
		t.Fatal("实参展开语义被抹除")
	}
	a = fingerprint(t, "func a() { type A = B; use(A{}) }")
	b = fingerprint(t, "func b() { type A B; use(A{}) }")
	if a.Exact == b.Exact {
		t.Fatal("类型别名语义被抹除")
	}
}

func TestRecursiveAndExternalCall(t *testing.T) {
	records, err := extract("package example\nfunc f() { f() };func g() { f() }", "sample.go")
	if err != nil {
		t.Fatal(err)
	}
	if records[0].Exact == records[1].Exact || *records[0].Renamed == *records[1].Renamed {
		t.Fatal("自递归与外部调用被合并")
	}
}

func TestCommentDoesNotAffectThreshold(t *testing.T) {
	a := fingerprint(t, "// 注释\n// 更多注释\nfunc a(x int) int { return x+1 }")
	b := fingerprint(t, "func b(x int) int { return x+1 }")
	if a.Nodes != b.Nodes || a.Exact != b.Exact {
		t.Fatal("注释影响阈值或指纹")
	}
}
