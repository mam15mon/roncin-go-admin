package data

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

// 批次头结构变更须从正式前态升级，不能只用空 Schema 验证删列语法。
func TestFinanceBillBatchHeadMigrationPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261114110000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	org, partner, creator, _, _ := seedAutoConfirmMigrationBase(t, data, "BATCH-HEAD")
	batchID := uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO finance_bill_batches
		(id, created_at, updated_at, organization_id, batch_no, idempotency_key, request_hash,
		 fee_count, bill_count, total_base_amount, base_currency, created_by)
		VALUES ($1, now(), now(), $2, 'BATCH-HEAD-UPGRADE', 'batch-head-upgrade', $3, 1, 1, 100, 'CNY', $4)`,
		batchID, org, strings.Repeat("a", 64), creator); err != nil {
		t.Fatal(err)
	}
	bill := createAutoConfirmMigrationBill(t, data, org, partner, "HEAD-UPGRADE", financebillent.StatusCONFIRMED, &batchID)
	for attempt := 0; attempt < 2; attempt++ {
		if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatalf("第 %d 次正式升级失败: %v", attempt+1, err)
		}
	}
	var removedColumns int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'finance_bill_batches'
		AND column_name IN ('fee_count','bill_count','total_base_amount','base_currency')`).Scan(&removedColumns); err != nil || removedColumns != 0 {
		t.Fatalf("旧汇总列应全部删除: count=%d err=%v", removedColumns, err)
	}
	batch, err := NewFinanceBillRepo(data).GetBatch(ctx, []uuid.UUID{org}, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if batch.BatchNo != "BATCH-HEAD-UPGRADE" || batch.CreatedBy != creator || batch.BillCount != 1 || len(batch.Bills) != 1 || batch.Bills[0].ID != bill.ID || len(batch.BaseTotals) != 1 || batch.BaseTotals[0].TotalBaseAmount.StringFixed(8) != "100.00000000" {
		t.Fatalf("升级须保留批次身份和账单事实，汇总改由账单读取: %#v", batch)
	}
}
