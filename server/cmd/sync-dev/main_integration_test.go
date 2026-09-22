package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

// 正式迁移在随机Schema内运行，完整迁移先建表，public仅解析已安装的pg_trgm扩展。
func TestDevelopmentSeedPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("需要专用PostgreSQL连接")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "seed_" + uuid.New().String()[:8]
	if _, err = admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
	u, err := url.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migration.Apply(ctx, db, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err = data.SyncPermissionManifest(ctx, db); err != nil {
		t.Fatal(err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	defer client.Close()
	first, err := runSeed(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
	counts := func() string {
		var result string
		err := db.QueryRowContext(ctx, `SELECT concat((SELECT count(*) FROM organizations),',',(SELECT count(*) FROM memberships),',',(SELECT count(*) FROM orders),',',(SELECT count(*) FROM order_fees),',',(SELECT count(*) FROM finance_bills),',',(SELECT count(*) FROM sea_house_bill_versions))`).Scan(&result)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	before := counts()
	if _, err = runSeed(ctx, client); err != nil {
		t.Fatal(err)
	}
	if after := counts(); after != before {
		t.Fatalf("重复注入产生额外记录: %s -> %s", before, after)
	}
	checks := map[string]string{
		"员工无系统身份":  `SELECT count(*) FROM memberships m JOIN organizations o ON o.id=m.organization_id WHERE o.kind='system'`,
		"角色公司一致":   `SELECT count(*) FROM role_assignments a JOIN memberships m ON m.id=a.membership_id JOIN roles r ON r.id=a.role_id WHERE m.organization_id<>r.organization_id`,
		"公司必须为独立根": `SELECT count(*) FROM organizations WHERE kind='company' AND parent_id IS NOT NULL`,
		"系统无经营资料":  `SELECT (SELECT count(*) FROM orders o JOIN organizations g ON g.id=o.organization_id WHERE g.kind='system')+(SELECT count(*) FROM partners p JOIN organizations g ON g.id=p.organization_id WHERE g.kind='system')+(SELECT count(*) FROM fee_settings f JOIN organizations g ON g.id=f.organization_id WHERE g.kind='system')`,
		"费用公司关联":   `SELECT count(*) FROM order_fees f JOIN orders o ON o.id=f.order_id JOIN fee_settings s ON s.id=f.fee_setting_id WHERE o.organization_id<>s.organization_id`,
		"科目开票关联":   `SELECT count(*) FROM fee_settings f JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.organization_id<>t.organization_id`,
		"创建人公司成员":  `SELECT count(*) FROM sea_document_void_events o WHERE NOT EXISTS(SELECT 1 FROM memberships m WHERE m.user_id=o.created_by AND m.organization_id=o.organization_id AND m.enabled)`,
		"港口模式":     `SELECT count(*) FROM ports WHERE NOT transport_modes @> '["SEA"]'::jsonb`,
	}
	for name, query := range checks {
		var n int
		if err = db.QueryRowContext(ctx, query).Scan(&n); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if n != 0 {
			t.Fatalf("%s: %d", name, n)
		}
	}
	// 已有成员角色丢失时，重复注入必须补齐。
	if _, err = db.ExecContext(ctx, `DELETE FROM role_assignments WHERE membership_id IN (SELECT id FROM memberships WHERE user_id=$1)`, first.adminUser.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = runSeed(ctx, client); err != nil {
		t.Fatal(err)
	}
	var roles int
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM role_assignments r JOIN memberships m ON m.id=r.membership_id WHERE m.user_id=$1`, first.adminUser.ID).Scan(&roles); err != nil || roles != 1 {
		t.Fatalf("成员角色未补齐: %d %v", roles, err)
	}
	// 在后半程强制失败，前面新增员工必须随整组回滚。
	if _, err = db.ExecContext(ctx, `UPDATE users SET username='previous_liumin' WHERE username='liumin'`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `ALTER TABLE ports ADD CONSTRAINT seed_port_failure CHECK (un_locode <> 'CNSHA') NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE ports SET un_locode='CNVFY' WHERE un_locode='CNSHA'`); err != nil {
		t.Fatal(err)
	}
	if _, err = runSeed(ctx, client); err == nil {
		t.Fatal("预期种子失败")
	}
	var users int
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE username='liumin'`).Scan(&users); err != nil || users != 0 {
		t.Fatal(fmt.Sprintf("失败未回滚新增员工: %d %v", users, err))
	}
	t.Logf("真实PG种子两次幂等、公司边界、角色补齐和失败回滚通过；核心计数 %s", before)
}
