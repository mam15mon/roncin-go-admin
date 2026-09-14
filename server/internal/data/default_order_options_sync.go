package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

// DefaultOrderOptionsSyncSummary 汇总默认订单主数据种子的补齐数量。
type DefaultOrderOptionsSyncSummary struct {
	Created int
}

// SyncDefaultOrderOptions 幂等补齐缺失的系统订单主数据种子。A 型主数据全局唯一，
// 不再按组织重复落行；已存在的种子保持原名称、启停状态和排序，避免覆盖业务人员
// 的显式维护结果。
func SyncDefaultOrderOptions(ctx context.Context, database transactionStarter) (*DefaultOrderOptionsSyncSummary, error) {
	summary := &DefaultOrderOptionsSyncSummary{}
	operationCompleted := false
	err := runTransaction(func() (*sql.Tx, error) {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin default order options sync: %w", err)
		}
		return tx, nil
	}, func(tx *sql.Tx) error {
		for _, item := range biz.DefaultOrderOptions() {
			result, err := tx.ExecContext(ctx, `
INSERT INTO "master_data_items" (
  "id", "created_at", "updated_at", "kind", "code", "name",
  "teu_factor", "source", "sort_order", "enabled", "attributes",
  "search_keywords"
)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2, $3, $4, $5, $6, true,
        '{}'::jsonb, $7)
ON CONFLICT ("kind", "code") DO NOTHING`,
				item.Kind, item.Code, item.Name, item.TEUFactor, item.Source, item.SortOrder,
				searchtext.Build(item.Name),
			)
			if err != nil {
				return fmt.Errorf("sync default order option %s/%s: %w", item.Kind, item.Code, err)
			}
			created, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("count default order option %s/%s: %w", item.Kind, item.Code, err)
			}
			summary.Created += int(created)
		}
		operationCompleted = true
		return nil
	})
	if err != nil {
		if operationCompleted {
			return nil, fmt.Errorf("commit default order options sync: %w", err)
		}
		return nil, err
	}
	return summary, nil
}
