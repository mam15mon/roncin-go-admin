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

// 覆盖费用更正已应用后取消主要组织的真实升级，确保成员及角色绑定完整保留。
func TestPostgresRemoveMembershipPrimaryPreservesAuthorization(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("设置 RONCIN_INTEGRATION_DATABASE_SOURCE 后运行真实 PostgreSQL 迁移测试")
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	schemaName := "roncin_membership_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("清理隔离 Schema 失败: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("..", "..", "..", "migrations")
	beforeDir := postgresMigrationDirThrough(t, dir, "20261013120000_order_fee_correction.sql")
	if err := Apply(ctx, db, beforeDir); err != nil {
		t.Fatalf("构建费用更正已应用的迁移前态失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO organizations(id,created_at,updated_at,code,name,kind,base_currency)
		VALUES ('10000000-0000-0000-0000-000000000001',now(),now(),'TEST','迁移测试公司','company','CNY');
		INSERT INTO users(id,created_at,updated_at,username,display_name)
		VALUES ('20000000-0000-0000-0000-000000000001',now(),now(),'migration.user','迁移测试员工');
		INSERT INTO memberships(id,created_at,updated_at,organization_id,user_id,"primary",enabled)
		VALUES ('30000000-0000-0000-0000-000000000001',now(),now(),'10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001',true,false);
		INSERT INTO roles(id,created_at,updated_at,organization_id,code,name,data_scope)
		VALUES ('40000000-0000-0000-0000-000000000001',now(),now(),'10000000-0000-0000-0000-000000000001','operator','操作员','organization');
		INSERT INTO role_assignments(id,created_at,updated_at,membership_id,role_id)
		VALUES ('50000000-0000-0000-0000-000000000001',now(),now(),'30000000-0000-0000-0000-000000000001','40000000-0000-0000-0000-000000000001');`); err != nil {
		t.Fatalf("准备主要组织及角色绑定失败: %v", err)
	}
	snapshot := func() string {
		t.Helper()
		var value string
		if err := db.QueryRowContext(ctx, `SELECT jsonb_build_object(
			'memberships', (SELECT jsonb_agg(to_jsonb(m) - 'primary' ORDER BY id) FROM memberships m),
			'assignments', (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM role_assignments a),
			'roles', (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM roles r),
			'users', (SELECT jsonb_agg(to_jsonb(u) ORDER BY id) FROM users u))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	for attempt := 1; attempt <= 2; attempt++ {
		if err := Apply(ctx, db, dir); err != nil {
			t.Fatalf("第 %d 次取消主要组织迁移失败: %v", attempt, err)
		}
		if after := snapshot(); after != before {
			t.Fatal("取消主要组织不得修改用户、成员启停状态、角色及绑定")
		}
	}
	var remainingColumns, revisions int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns
		WHERE table_schema=$1 AND table_name='memberships' AND column_name='primary'`, schemaName).Scan(&remainingColumns); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations
		WHERE version='20261014100000_remove_membership_primary'`).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if remainingColumns != 0 || revisions != 1 {
		t.Fatalf("迁移结果不符: primary 列=%d，迁移记录=%d", remainingColumns, revisions)
	}
}
