package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestShippingAgentSnapshot(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "seeds", snapshotFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "93dca1753a3db49847b6396853b12beac20d5f6772e6b16cd7ebbd86d193069c" {
		t.Fatalf("已核对的来源快照被改写: %s", got)
	}
	rows, err := parseSnapshot(raw)
	if err != nil || len(rows) != 29 {
		t.Fatalf("来源解析错误: %d %v", len(rows), err)
	}
	if rows[0].Code() != "HGJ-1269566" || rows[0].Name != "中联运通" || rows[0].NameEN != "中联运通" || rows[0].SortOrder != 29 {
		t.Fatalf("来源字段映射错误: %+v", rows[0])
	}
	var original map[string]any
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"响应失败", "条数缺失", "重复ID", "空名称", "非法ID", "负排序", "无效JSON"} {
		t.Run(name, func(t *testing.T) {
			var value map[string]any
			_ = json.Unmarshal(raw, &value)
			items := value["data"].([]any)
			first := items[0].(map[string]any)
			switch name {
			case "响应失败":
				value["success"] = false
			case "条数缺失":
				value["data"] = items[:28]
			case "重复ID":
				items[1].(map[string]any)["id"] = first["id"]
			case "空名称":
				first["value"] = ""
			case "非法ID":
				first["id"] = -1
			case "负排序":
				first["order"] = -1
			}
			invalid, _ := json.Marshal(value)
			if name == "无效JSON" {
				invalid = []byte("{")
			}
			if _, err := parseSnapshot(invalid); err == nil {
				t.Fatal("非法来源应在连接数据库前失败")
			}
		})
	}
}
