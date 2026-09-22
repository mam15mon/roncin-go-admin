package main

import (
	"context"
	"database/sql"
	"github.com/google/uuid"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if companies != 4 || memberships != 0 || systemRules != 0 || companyRoles != 4 {
		t.Fatalf("初始化边界不符 company=%d membership=%d systemRules=%d roles=%d", companies, memberships, systemRules, companyRoles)
	}
	if err := bootstrapWithMigrations(context.Background(), cfg, filepath.Join("..", "..", "migrations")); err == nil {
		t.Fatal("重复初始化应拒绝")
	}
}
