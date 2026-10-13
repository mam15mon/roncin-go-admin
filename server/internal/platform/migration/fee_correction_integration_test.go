package migration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPostgresFeeCorrectionReplacesEntIndex 覆盖正式旧索引与 Ent 同列索引并存的开发库。
func TestPostgresFeeCorrectionReplacesEntIndex(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("设置 RONCIN_INTEGRATION_DATABASE_SOURCE 后运行真实 PostgreSQL 迁移测试")
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	schemaName := "roncin_correction_index_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("创建隔离 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("清理隔离 Schema 失败: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
		t.Fatalf("切换隔离 Schema 失败: %v", err)
	}
	dir := filepath.Join("..", "..", "..", "migrations")
	beforeDir := postgresMigrationDirThrough(t, dir, "20261012110000_fee_supplement_decision_reason.sql")
	if err := Apply(ctx, db, beforeDir); err != nil {
		t.Fatalf("构建费用更正前态失败: %v", err)
	}
	// 直接构造已确认的索引差异，禁止调用 Ent Schema.Create 污染迁移前态。
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX "financecommissionadjustment_co_36fb58015f1b066e73dc271e344c49a4"
		ON "finance_commission_adjustments" ("commission_id", "order_id", "source_type", "source_verification_id")`); err != nil {
		t.Fatalf("构造 Ent 同列索引失败: %v", err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if err := Apply(ctx, db, dir); err != nil {
			t.Fatalf("第 %d 次应用费用更正迁移失败: %v", attempt, err)
		}
	}
	var fullIndexes, partialIndexes int
	if err := db.QueryRowContext(ctx, `SELECT
		count(*) FILTER (WHERE indexdef NOT LIKE '%WHERE%'),
		count(*) FILTER (WHERE indexdef LIKE '%WHERE%ECONOMIC_REALLOCATION%')
		FROM pg_indexes WHERE schemaname = $1 AND tablename = 'finance_commission_adjustments'
		AND indexdef LIKE '%(commission_id, order_id, source_type, source_verification_id)%'`,
		schemaName).Scan(&fullIndexes, &partialIndexes); err != nil {
		t.Fatalf("检查提成调整索引失败: %v", err)
	}
	if fullIndexes != 0 || partialIndexes != 1 {
		t.Fatalf("应仅保留一个排除经济重分配的部分唯一索引，实际无条件索引=%d，部分索引=%d", fullIndexes, partialIndexes)
	}
	var revisions int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations
		WHERE version = '20261013120000_order_fee_correction'`).Scan(&revisions); err != nil {
		t.Fatalf("检查迁移记录失败: %v", err)
	}
	if revisions != 1 {
		t.Fatalf("费用更正迁移应只记录一次，实际为 %d", revisions)
	}
}
