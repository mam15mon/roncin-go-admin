package main

import (
	"encoding/json"
	"testing"
)

func TestFlexibleIntAcceptsNumberAndString(t *testing.T) {
	for _, input := range []string{`2`, `"2"`} {
		var value flexibleInt
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			t.Fatalf("Unmarshal(%s) error = %v", input, err)
		}
		if value != 2 {
			t.Fatalf("Unmarshal(%s) = %d, want 2", input, value)
		}
	}
}

func TestAddRegionKeepsOnlyValidAdministrativeCodes(t *testing.T) {
	rows := make(map[string]regionRow)
	addRegion(rows, regionNode{Code: "310000000000", Name: " 上海市 ", Level: 1}, nil, defaultTable)
	addRegion(rows, regionNode{Code: "资料暂缺", Name: "台湾省", Level: 1}, nil, defaultTable)
	if len(rows) != 1 || rows["310000000000"].Name != "上海市" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestAddRegionRejectsInvalidLevel(t *testing.T) {
	rows := make(map[string]regionRow)
	addRegion(rows, regionNode{Code: "310000000000", Name: "上海市", Level: 4}, nil, defaultTable)
	if len(rows) != 0 {
		t.Fatalf("rows = %#v, want empty", rows)
	}
}
