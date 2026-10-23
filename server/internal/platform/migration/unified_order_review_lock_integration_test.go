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

// 正式迁移必须明确拒绝无法判定人工审核依据的存量状态，且不得改写历史数据。
func TestPostgresUnifiedOrderReviewLockRejectsAmbiguousHistory(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" || os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("需设置 PostgreSQL 迁移测试开关与隔离数据库连接")
	}
	db, err := sql.Open("pgx", os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	schema := `"unified_lock_preflight_` + strings.ReplaceAll(uuid.NewString(), "-", "") + `"`
	if _, err = db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	if _, err = db.ExecContext(ctx, "SET search_path TO "+schema+",public"); err != nil {
		t.Fatal(err)
	}
	// 仅建立前置检查涉及的旧契约，执行的脚本仍是完整正式迁移文件。
	if _, err = db.ExecContext(ctx, `CREATE TABLE orders(order_no text,locked_at timestamptz,fee_review_status text);
 CREATE TABLE order_fee_supplement_requests(id uuid,status text);
 CREATE TABLE order_fee_correction_requests(id uuid,status text);`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "20261023100000_unified_order_review_lock.sql"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, fixture, want string }{
		{"旧人工或自动锁", `INSERT INTO orders VALUES ('AMBIGUOUS-LOCK',now(),'UNSUBMITTED')`, "AMBIGUOUS-LOCK"},
		{"独立费用批准", `INSERT INTO orders VALUES ('OLD-APPROVAL',NULL,'APPROVED')`, "OLD-APPROVAL"},
		{"整票待审", `INSERT INTO orders VALUES ('OLD-PENDING',NULL,'PENDING')`, "OLD-PENDING"},
		{"旧补录待审", `INSERT INTO order_fee_supplement_requests VALUES ('11111111-1111-1111-1111-111111111111','PENDING')`, "旧待审补录"},
		{"旧更正待审", `INSERT INTO order_fee_correction_requests VALUES ('22222222-2222-2222-2222-222222222222','PENDING')`, "旧待审费用更正"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, tc.fixture); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, string(migration)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("迁移必须给出具体阻断证据，got %v", err)
			}
			var count int
			if err := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM orders)+(SELECT count(*) FROM order_fee_supplement_requests)+(SELECT count(*) FROM order_fee_correction_requests)`).Scan(&count); err != nil || count != 1 {
				t.Fatalf("迁移不应修改旧事实：count=%d err=%v", count, err)
			}
			if _, err := db.ExecContext(ctx, `TRUNCATE orders,order_fee_supplement_requests,order_fee_correction_requests`); err != nil {
				t.Fatal(err)
			}
		})
	}
}
