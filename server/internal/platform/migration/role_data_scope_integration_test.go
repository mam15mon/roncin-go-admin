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

// 覆盖范围退役升级：self 不得自动激活，其他启停状态、角色与成员权限关联完整保留。
func TestPostgresRetireRoleDataScopePreservesAuthorization(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" || os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("需设置 PostgreSQL 迁移测试开关与隔离数据库连接")
	}
	db, err := sql.Open("pgx", os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	schemaName := "roncin_role_scope_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("清理隔离Schema失败: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("..", "..", "..", "migrations")
	if err := Apply(ctx, db, postgresMigrationDirThrough(t, dir, "20261014100000_remove_membership_primary.sql")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
 INSERT INTO organizations(id,created_at,updated_at,code,name,kind,base_currency)
 VALUES ('10000000-0000-0000-0000-000000000001',now(),now(),'SCOPE','范围退役测试公司','company','CNY');
 INSERT INTO users(id,created_at,updated_at,username,display_name)
 VALUES ('20000000-0000-0000-0000-000000000001',now(),now(),'scope.user','范围退役测试员工');
 INSERT INTO memberships(id,created_at,updated_at,organization_id,user_id,enabled)
 VALUES ('30000000-0000-0000-0000-000000000001',now(),now(),'10000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001',true);
 INSERT INTO permissions(id,created_at,updated_at,key,name,"group",description)
 VALUES ('60000000-0000-0000-0000-000000000001',now(),now(),'migration.role_scope.test','迁移测试权限','系统管理','迁移测试');
 INSERT INTO roles(id,created_at,updated_at,organization_id,code,name,data_scope,enabled)
 SELECT ('40000000-0000-0000-0000-' || lpad(n::text,12,'0'))::uuid,now(),now(),'10000000-0000-0000-0000-000000000001',
 'role_' || n,'测试角色' || n,scope,enabled FROM (VALUES (1,'organization',true),(2,'organization_tree',true),(3,'all',true),(4,'self',true),(5,'all',false),(6,'self',false)) AS fixture(n,scope,enabled);
 INSERT INTO role_assignments(id,created_at,updated_at,membership_id,role_id)
 SELECT ('50000000-0000-0000-0000-' || lpad(row_number() OVER (ORDER BY id)::text,12,'0'))::uuid,now(),now(),'30000000-0000-0000-0000-000000000001',id FROM roles;
 INSERT INTO role_permissions(role_id,permission_id)
 SELECT id,'60000000-0000-0000-0000-000000000001' FROM roles;
 `); err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		t.Helper()
		var value string
		if err := db.QueryRowContext(ctx, `SELECT jsonb_build_object(
   'roles',(SELECT jsonb_agg(to_jsonb(r)-'data_scope'-'enabled'-'updated_at' ORDER BY id) FROM roles r),
   'assignments',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM role_assignments a),
   'permissions',(SELECT jsonb_agg(to_jsonb(p) ORDER BY role_id) FROM role_permissions p),
   'memberships',(SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM memberships m),
   'users',(SELECT jsonb_agg(to_jsonb(u) ORDER BY id) FROM users u))::text`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	before := snapshot()
	for attempt := 0; attempt < 2; attempt++ {
		if err := Apply(ctx, db, dir); err != nil {
			t.Fatal(err)
		}
		if after := snapshot(); after != before {
			t.Fatal("范围退役修改了角色内容、成员或权限关联")
		}
	}
	var enabledCodes string
	if err := db.QueryRowContext(ctx, `SELECT string_agg(code,',' ORDER BY code) FROM roles WHERE enabled`).Scan(&enabledCodes); err != nil {
		t.Fatal(err)
	}
	if enabledCodes != "role_1,role_2,role_3" {
		t.Fatalf("启用角色不符，self不得自动激活，停用角色不得恢复: %s", enabledCodes)
	}
	var remainingColumns int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=$1 AND table_name='roles' AND column_name='data_scope'`, schemaName).Scan(&remainingColumns); err != nil {
		t.Fatal(err)
	}
	if remainingColumns != 0 {
		t.Fatal("角色范围列仍然存在")
	}
}
