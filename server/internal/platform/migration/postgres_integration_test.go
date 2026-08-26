package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	entmigrate "github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresColdStartMigration(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		t.Fatal("DATABASE_SOURCE 不能为空")
	}

	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("连接 PostgreSQL 失败: %v", err)
	}

	schemaName := "roncin_migration_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("创建临时 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("删除临时 Schema 失败: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema); err != nil {
		t.Fatalf("切换临时 Schema 失败: %v", err)
	}

	dir := filepath.Join("..", "..", "..", "migrations")
	if err := Apply(ctx, db, dir); err != nil {
		t.Fatalf("冷启动迁移失败: %v", err)
	}
	if err := Apply(ctx, db, dir); err != nil {
		t.Fatalf("重复执行迁移失败: %v", err)
	}

	for _, table := range entmigrate.Tables {
		var exists bool
		query := fmt.Sprintf(`SELECT to_regclass('%s.%s') IS NOT NULL`, schemaName, table.Name)
		if err := db.QueryRowContext(ctx, query).Scan(&exists); err != nil {
			t.Fatalf("检查表 %s 失败: %v", table.Name, err)
		}
		if !exists {
			t.Errorf("迁移后缺少表 %s", table.Name)
		}
	}

	expectedIndexes := map[string][]string{
		"financeinvoice_org_tax_invoice_no":    {"UNIQUE", "organization_id", "tax_invoice_no", "tax_invoice_no IS NOT NULL"},
		"financeinvoice_org_red_invoice_no":    {"UNIQUE", "organization_id", "red_invoice_no", "red_invoice_no IS NOT NULL"},
		"finance_bill_lines_active_fee_unique": {"UNIQUE", "order_fee_id", "active = true"},
		"verification_allocation_pair_unique":  {"UNIQUE", "verification_id", "cashflow_id", "bill_id"},
	}
	for name, fragments := range expectedIndexes {
		var definition string
		if err := db.QueryRowContext(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname = $1 AND indexname = $2`, schemaName, name).Scan(&definition); err != nil {
			t.Errorf("迁移后缺少索引 %s: %v", name, err)
			continue
		}
		for _, fragment := range fragments {
			if !strings.Contains(definition, fragment) {
				t.Errorf("索引 %s 定义缺少 %q: %s", name, fragment, definition)
			}
		}
	}
}
