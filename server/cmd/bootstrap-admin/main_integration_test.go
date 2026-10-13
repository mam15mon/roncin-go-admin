package main

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

func TestBootstrapFormalMigrationsAndWorkspaceBoundaries(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("需要真实PostgreSQL测试连接")
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema := "bootstrap_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = db.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
	parsed, err := url.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	cfg := &bootstrapConfig{databaseSource: parsed.String(), username: "initial.admin", displayName: "系统管理员", plainPassword: "test-password-only-12345", organizationCode: "SYSTEM", organizationName: "系统管理"}
	if err := bootstrapWithMigrations(context.Background(), cfg, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	isolated, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer isolated.Close()
	var companies, memberships, systemRules, companyRoles int
	if err = isolated.QueryRow(`SELECT count(*) FROM organizations WHERE kind='company' AND parent_id IS NULL`).Scan(&companies); err != nil {
		t.Fatal(err)
	}
	if err = isolated.QueryRow(`SELECT count(*) FROM memberships m JOIN organizations o ON o.id=m.organization_id WHERE o.kind='company'`).Scan(&memberships); err != nil {
		t.Fatal(err)
	}
	if err = isolated.QueryRow(`SELECT count(*) FROM number_rules n JOIN organizations o ON o.id=n.organization_id WHERE o.kind='system'`).Scan(&systemRules); err != nil {
		t.Fatal(err)
	}
	if err = isolated.QueryRow(`SELECT count(*) FROM roles r JOIN organizations o ON o.id=r.organization_id WHERE o.kind='company' AND r.code='administrator'`).Scan(&companyRoles); err != nil {
		t.Fatal(err)
	}
	if companies != 2 || memberships != 2 || systemRules != 0 || companyRoles != 2 {
		t.Fatalf("初始化边界不符 company=%d membership=%d systemRules=%d roles=%d", companies, memberships, systemRules, companyRoles)
	}
	if err := bootstrapWithMigrations(context.Background(), cfg, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("重复执行应补齐且保持幂等: %v", err)
	}
	var organizationID, roleID string
	if err := isolated.QueryRow(`SELECT o.id,r.id FROM organizations o JOIN roles r ON r.organization_id=o.id
		WHERE o.code='SYSTEM' AND r.code='administrator'`).Scan(&organizationID, &roleID); err != nil {
		t.Fatal(err)
	}
	clearAccounts := func() {
		t.Helper()
		if _, err := isolated.Exec(`DELETE FROM role_assignments; DELETE FROM memberships; DELETE FROM users`); err != nil {
			t.Fatalf("清空初始化账号失败: %v", err)
		}
	}
	clearAccounts()
	if _, err := isolated.Exec(`UPDATE organizations SET name='保留的系统工作台' WHERE id=$1`, organizationID); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapWithMigrations(context.Background(), cfg, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("保留组织和角色后重新初始化失败: %v", err)
	}
	var organizationName, reusedRoleID string
	if err := isolated.QueryRow(`SELECT o.name,r.id FROM organizations o JOIN roles r ON r.organization_id=o.id
		WHERE o.id=$1 AND r.code='administrator'`, organizationID).Scan(&organizationName, &reusedRoleID); err != nil {
		t.Fatal(err)
	}
	if organizationName != "保留的系统工作台" || reusedRoleID != roleID {
		t.Fatalf("重新初始化必须复用原组织和角色，实际名称=%s，角色=%s", organizationName, reusedRoleID)
	}
	var organizations, roles, users, rolePermissions, companyMemberships int
	for query, result := range map[string]*int{
		`SELECT count(*) FROM organizations`: &organizations,
		`SELECT count(*) FROM roles`:         &roles,
		`SELECT count(*) FROM users`:         &users,
		`SELECT count(*) FROM memberships m JOIN organizations o ON o.id=m.organization_id WHERE o.kind='company'`: &companyMemberships,
	} {
		if err := isolated.QueryRow(query).Scan(result); err != nil {
			t.Fatal(err)
		}
	}
	if err := isolated.QueryRow(`SELECT count(*) FROM role_permissions WHERE role_id=$1`, roleID).Scan(&rolePermissions); err != nil {
		t.Fatal(err)
	}
	expectedPermissions := 0
	for _, permission := range access.Manifest() {
		if access.PermissionAllowedInWorkspace(permission.Key, true) {
			expectedPermissions++
		}
	}
	if organizations != 3 || roles != 3 || users != 1 || companyMemberships != 2 || rolePermissions != expectedPermissions {
		t.Fatalf("重新初始化边界不符：组织=%d，角色=%d，账号=%d，公司成员=%d，管理员权限=%d", organizations, roles, users, companyMemberships, rolePermissions)
	}
	clearAccounts()
	for _, invalid := range []struct {
		name    string
		kind    string
		enabled bool
	}{
		{name: "同码公司不能作为系统工作台", kind: "company", enabled: true},
		{name: "停用系统工作台不能初始化", kind: "system", enabled: false},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			if _, err := isolated.Exec(`UPDATE organizations SET kind=$1,enabled=$2 WHERE id=$3`, invalid.kind, invalid.enabled, organizationID); err != nil {
				t.Fatal(err)
			}
			if err := bootstrapWithMigrations(context.Background(), cfg, filepath.Join("..", "..", "migrations")); err == nil {
				t.Fatal("无效系统工作台应拒绝初始化")
			}
			if err := isolated.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil || users != 0 {
				t.Fatalf("初始化失败不得写入账号：数量=%d，错误=%v", users, err)
			}
		})
	}
	if _, err := isolated.Exec(`UPDATE organizations SET kind='system',enabled=true WHERE id=$1;`, organizationID); err != nil {
		t.Fatal(err)
	}
	if _, err := isolated.Exec(`UPDATE roles SET enabled=false WHERE id=$1`, roleID); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapWithMigrations(context.Background(), cfg, filepath.Join("..", "..", "migrations")); err == nil {
		t.Fatal("停用管理员角色应拒绝初始化")
	}
	if err := isolated.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil || users != 0 {
		t.Fatalf("停用角色初始化失败不得写入账号：数量=%d，错误=%v", users, err)
	}
}
