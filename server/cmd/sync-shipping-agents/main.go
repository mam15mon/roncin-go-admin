package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/roncin/roncin-go-admin/server/cmd/internal/syncrunner"
	"github.com/roncin/roncin-go-admin/server/internal/data"
)

const snapshotFile = "shipping-agents.hgj.20261027.json"

func main() { syncrunner.Run(run) }

func run(ctx context.Context) error {
	apply := flag.Bool("apply", false, "将缺失的船代写入数据库，保留已有记录")
	source := flag.String("source", filepath.Join("seeds", snapshotFile), "船代来源快照 JSON 路径")
	flag.Parse()
	raw, err := os.ReadFile(*source)
	if err != nil {
		return fmt.Errorf("读取船代快照失败: %w", err)
	}
	rows, err := parseSnapshot(raw)
	if err != nil {
		return err
	}
	fmt.Printf("船代来源快照：%s，记录 %d 条，SHA-256：%x\n", *source, len(rows), sha256.Sum256(raw))
	if !*apply {
		fmt.Println("当前为离线预览；使用 -apply 补齐数据库中缺失的船代")
		return nil
	}
	store, cleanup, err := syncrunner.OpenStore()
	if err != nil {
		return fmt.Errorf("初始化船代同步存储失败: %w", err)
	}
	defer cleanup()
	created, err := store.ApplyShippingAgents(ctx, rows)
	if err != nil {
		return err
	}
	fmt.Printf("船代同步完成：新增 %d，已有保留 %d\n", created, len(rows)-created)
	return nil
}

func parseSnapshot(raw []byte) ([]data.ShippingAgentSyncRecord, error) {
	var snapshot struct {
		Success bool                           `json:"success"`
		Code    int                            `json:"code"`
		Data    []data.ShippingAgentSyncRecord `json:"data"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("解析船代快照失败: %w", err)
	}
	if !snapshot.Success || snapshot.Code != 0 || len(snapshot.Data) != 29 {
		return nil, fmt.Errorf("船代快照须为成功响应且正好包含已核对的 29 条记录")
	}
	seen := make(map[int64]bool, len(snapshot.Data))
	for _, row := range snapshot.Data {
		if row.SourceID <= 0 || seen[row.SourceID] || row.Name == "" || strings.TrimSpace(row.Name) != row.Name || utf8.RuneCountInString(row.Name) > 200 || utf8.RuneCountInString(row.NameEN) > 200 || strings.TrimSpace(row.NameEN) != row.NameEN || row.SortOrder < 0 {
			return nil, fmt.Errorf("船代快照含非法字段或重复来源 ID：%d", row.SourceID)
		}
		seen[row.SourceID] = true
	}
	return snapshot.Data, nil
}
