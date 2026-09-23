package migration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// 费用科目种子迁移（20260923120000）测试：先应用此前全部迁移并预置公司，
// 再应用完整目录触发种子迁移，最后断言模板、公司副本与主数据。
func feeCatalogSeedFixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("需要 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	schema := "fee_seed_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	mustFeeSeedSQL(t, db, `CREATE SCHEMA "`+schema+`"`)
	t.Cleanup(func() {
		if _, err := db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); err != nil {
			t.Error(err)
		}
	})
	mustFeeSeedSQL(t, db, `SET search_path TO "`+schema+`",public`)
	dir := filepath.Join("..", "..", "..", "migrations")
	old := t.TempDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "20260923120000" {
			continue
		}
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(old, filepath.Base(file)), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Apply(context.Background(), db, old); err != nil {
		t.Fatal(err)
	}
	// 预置一家公司，并故意预置同码本地科目 OF（校验追加同码跳过、不覆盖本地配置）。
	// 全新链在种子迁移前没有任何费用类别/计费单位，先手工落 BOOKING/BL 供本地科目引用，
	// 种子迁移随后按 kind+code 幂等跳过这两行。
	mustFeeSeedSQL(t, db, `INSERT INTO organizations(id,created_at,updated_at,code,name,kind,base_currency) VALUES ('20000000-0000-0000-0000-000000000001',now(),now(),'fee-seed-co','种子测试公司','company','CNY');
 INSERT INTO master_data_items(id,created_at,updated_at,kind,code,name,source,sort_order,enabled,attributes,search_keywords) VALUES ('20000000-0000-0000-0000-000000000010',now(),now(),'charge_category','BOOKING','订舱','system',10,true,'{}','订舱 DINGCANG DING CANG DC');
 INSERT INTO billing_units(id,created_at,updated_at,code,name,sort_order,enabled,is_container_unit,search_keywords) VALUES ('20000000-0000-0000-0000-000000000011',now(),now(),'BL','票',20,true,false,'票 PIAO P');
 INSERT INTO taxable_services(id,created_at,updated_at,organization_id,name,default_tax_rate,enabled,search_keywords) VALUES ('20000000-0000-0000-0000-000000000002',now(),now(),'20000000-0000-0000-0000-000000000001','本地税务',0,true,'本地税务');
 INSERT INTO fee_settings(id,created_at,updated_at,organization_id,fee_code,name_zh,charge_category_id,default_currency,billing_unit_id,tax_rate,taxable_service_id,enabled,sort_order,search_keywords)
 VALUES ('20000000-0000-0000-0000-000000000003',now(),now(),'20000000-0000-0000-0000-000000000001','OF','本地海运费','20000000-0000-0000-0000-000000000010','CNY','20000000-0000-0000-0000-000000000011',0,'20000000-0000-0000-0000-000000000002',true,5,'本地海运费');`)
	return db, dir
}

func mustFeeSeedSQL(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatal(err)
	}
}

func assertFeeSeedCount(t *testing.T, db *sql.DB, q string, want int) {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("数量得到 %d，期望 %d: %s", n, want, q)
	}
}

func TestFeeCatalogSeedSeedsTemplatesAndCompanyCopies(t *testing.T) {
	db, dir := feeCatalogSeedFixture(t)

	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatalf("应用种子迁移失败: %v", err)
	}

	// 模板 122 条且 fee_code 无重复；类别/单位/异常主数据齐备。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates`, 122)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM (SELECT fee_code FROM fee_setting_templates GROUP BY fee_code HAVING count(*)>1) d`, 0)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM master_data_items WHERE kind='charge_category'`, 24)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM master_data_items WHERE kind='abnormal_case'`, 5)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM billing_units`, 7)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates WHERE search_keywords=''`, 0)

	// THC/STORAGE 模板类别改挂港口操作。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates t JOIN master_data_items m ON m.id=t.charge_category_id WHERE t.fee_code IN ('THC','STORAGE') AND m.code='PORT_OPS'`, 2)

	// 公司副本：122 条（预置本地 OF 跳过后由模板 OF 顶上同码计数），应税劳务 7 个名称按公司创建。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_settings WHERE organization_id='20000000-0000-0000-0000-000000000001'`, 122)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM taxable_services WHERE organization_id='20000000-0000-0000-0000-000000000001'`, 8)

	// 同码本地科目未被改写：名称仍为「本地海运费」，税务仍指向本地税务。
	var nameZh, serviceName string
	if err := db.QueryRow(`SELECT f.name_zh, t.name FROM fee_settings f JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.organization_id='20000000-0000-0000-0000-000000000001' AND f.fee_code='OF'`).Scan(&nameZh, &serviceName); err != nil {
		t.Fatal(err)
	}
	if nameZh != "本地海运费" || serviceName != "本地税务" {
		t.Fatalf("同码本地科目被覆盖: name_zh=%s, service=%s", nameZh, serviceName)
	}

	// 异常挂接与检索键抽查。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates WHERE fee_code='KC' AND abnormal_case_id=(SELECT id FROM master_data_items WHERE kind='abnormal_case' AND code='ABN-CUSTOMER-CANCEL')`, 1)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates WHERE search_keywords LIKE '%KONGCANGFEI%' AND fee_code='KC'`, 1)
}
