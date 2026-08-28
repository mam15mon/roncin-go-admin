package data

import (
	"context"
	"fmt"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// DefaultOrderOptionsSyncSummary 汇总默认订单主数据种子的补齐数量。
type DefaultOrderOptionsSyncSummary struct {
	Created int
}

// SyncDefaultOrderOptions 为所有已有组织补齐缺失的系统订单主数据种子。已存在的
// 主数据保持原名称、启停状态和排序，避免覆盖业务人员的显式维护结果。
func SyncDefaultOrderOptions(ctx context.Context, database transactionStarter) (*DefaultOrderOptionsSyncSummary, error) {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin default order options sync: %w", err)
	}
	defer tx.Rollback()

	summary := &DefaultOrderOptionsSyncSummary{}
	for _, item := range biz.DefaultOrderOptions() {
		result, err := tx.ExecContext(ctx, `
INSERT INTO "master_data_items" (
  "id", "created_at", "updated_at", "kind", "code", "name",
  "teu_factor", "source", "sort_order", "enabled", "organization_id"
)
SELECT gen_random_uuid(), NOW(), NOW(), $1, $2, $3, $4, $5, $6, true, "id"
FROM "organizations"
ON CONFLICT ("organization_id", "kind", "code") DO NOTHING`,
			item.Kind, item.Code, item.Name, item.TEUFactor, item.Source, item.SortOrder,
		)
		if err != nil {
			return nil, fmt.Errorf("sync default order option %s/%s: %w", item.Kind, item.Code, err)
		}
		created, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("count default order option %s/%s: %w", item.Kind, item.Code, err)
		}
		summary.Created += int(created)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit default order options sync: %w", err)
	}
	return summary, nil
}
