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
// 再仅应用被测种子迁移断言目标集合，最后验证后续追加与重复执行。
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
	mustFeeSeedSQL(t, db, `INSERT INTO organizations(id,created_at,updated_at,code,name,kind,base_currency) VALUES
 ('20000000-0000-0000-0000-000000000001',now(),now(),'fee-seed-co','种子测试公司','company','CNY'),
 ('20000000-0000-0000-0000-000000000004',now(),now(),'fee-seed-other','种子测试另一公司','company','CNY');
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
	const localFeeSnapshotQuery = `SELECT jsonb_build_object('fee',to_jsonb(f),'tax',to_jsonb(t))::text FROM fee_settings f JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.id='20000000-0000-0000-0000-000000000003'`
	var localFeeBefore string
	if err := db.QueryRow(localFeeSnapshotQuery).Scan(&localFeeBefore); err != nil {
		t.Fatal(err)
	}
	seedOnly := t.TempDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "20260923121000" {
			continue
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(seedOnly, filepath.Base(file)), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}

	if err := Apply(context.Background(), db, seedOnly); err != nil {
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

	assertFeeSeedCount(t, db, `SELECT count(*) FROM master_data_items WHERE kind='abnormal_case' AND code IN ('ABN-WAITING','ABN-OVERNIGHT','ABN-INSPECTION','ABN-CUSTOMER-CANCEL','ABN-ROLL-REASSIGN')`, 5)
	// 公司副本：121 个模板副本 + 1 个保留的本地 OF；应税劳务 7 个名称按公司创建。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_settings WHERE organization_id='20000000-0000-0000-0000-000000000001'`, 122)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM taxable_services WHERE organization_id='20000000-0000-0000-0000-000000000001'`, 8)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_settings WHERE organization_id='20000000-0000-0000-0000-000000000004'`, 122)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM taxable_services WHERE organization_id='20000000-0000-0000-0000-000000000004'`, 7)
	// 数量与 INNER JOIN 内容比较不能单独识别缺失代码；逐公司校验全部模板代码均有副本。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates t CROSS JOIN organizations o
 LEFT JOIN fee_settings f ON f.fee_code=t.fee_code AND f.organization_id=o.id
 WHERE o.id IN ('20000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000004') AND f.id IS NULL`, 0)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_settings f JOIN taxable_services tax ON tax.id=f.taxable_service_id WHERE tax.organization_id<>f.organization_id`, 0)
	// 每个新增副本完整保留科目配置与税务文本；仅本地 OF 允许与模板不同。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_settings f JOIN fee_setting_templates t USING(fee_code) JOIN taxable_services tax ON tax.id=f.taxable_service_id
 WHERE NOT(f.organization_id='20000000-0000-0000-0000-000000000001' AND f.fee_code='OF') AND
 (f.id=t.id OR (to_jsonb(f)-ARRAY['id','created_at','updated_at','organization_id','taxable_service_id']) IS DISTINCT FROM
 (to_jsonb(t)-ARRAY['id','created_at','updated_at','taxable_service_name','taxable_service_short_name','taxable_service_goods_code','taxable_service_default_tax_rate']) OR
 ROW(tax.name,tax.short_name,tax.goods_code,tax.default_tax_rate) IS DISTINCT FROM ROW(t.taxable_service_name,t.taxable_service_short_name,t.taxable_service_goods_code,t.taxable_service_default_tax_rate))`, 0)

	// 同码本地科目未被改写：名称仍为「本地海运费」，税务仍指向本地税务。
	var nameZh, serviceName string
	if err := db.QueryRow(`SELECT f.name_zh, t.name FROM fee_settings f JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.organization_id='20000000-0000-0000-0000-000000000001' AND f.fee_code='OF'`).Scan(&nameZh, &serviceName); err != nil {
		t.Fatal(err)
	}
	if nameZh != "本地海运费" || serviceName != "本地税务" {
		t.Fatalf("同码本地科目被覆盖: name_zh=%s, service=%s", nameZh, serviceName)
	}
	var localFeeAfter string
	if err := db.QueryRow(localFeeSnapshotQuery).Scan(&localFeeAfter); err != nil {
		t.Fatal(err)
	}
	if localFeeAfter != localFeeBefore {
		t.Fatalf("同码本地科目或税务被修改: %s", boundarySnapshotDiff(t, localFeeBefore, localFeeAfter))
	}

	// 异常挂接与检索键抽查。
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates WHERE fee_code='KC' AND abnormal_case_id=(SELECT id FROM master_data_items WHERE kind='abnormal_case' AND code='ABN-CUSTOMER-CANCEL')`, 1)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM fee_setting_templates WHERE search_keywords LIKE '%KONGCANGFEI%' AND fee_code='KC'`, 1)

	// 后续独立迁移新增 13 个异常类型，不应改变费用种子的 5 个目标项。
	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	assertFeeSeedCount(t, db, `SELECT count(*) FROM master_data_items WHERE kind='abnormal_case'`, 18)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM master_data_items WHERE kind='abnormal_case' AND code IN ('ABN-WAITING','ABN-OVERNIGHT','ABN-INSPECTION','ABN-CUSTOMER-CANCEL','ABN-ROLL-REASSIGN')`, 5)
	assertFeeSeedCount(t, db, `SELECT count(*) FROM master_data_items WHERE kind='abnormal_case' AND code IN ('ABN-MISSED-LOAD','ABN-ROLLED','ABN-DETENTION','ABN-DEMURRAGE','ABN-CARGO-HELD','ABN-40NOR','ABN-VESSEL-DELAY','ABN-SHUT-OUT','ABN-DAMAGED-BOX','ABN-CARGO-DAMAGE','ABN-CARGO-SHORTAGE','ABN-WET-DAMAGE','ABN-SEAL-ISSUE')`, 13)

	// 不仅验证迁移 ledger 的重复 Apply：直接重放幂等种子 SQL，校验业务值与 ID 无变化。
	tables := []string{"master_data_items", "billing_units", "fee_setting_templates", "fee_settings", "taxable_services"}
	before := make(map[string]string, len(tables))
	for _, table := range tables {
		before[table] = feeSeedTableSnapshot(t, db, table)
	}
	seed, err := os.ReadFile(filepath.Join(dir, "20260923120000_fee_catalog_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	mustFeeSeedSQL(t, db, string(seed))
	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		if after := feeSeedTableSnapshot(t, db, table); after != before[table] {
			t.Fatalf("重复费用种子改变 %s 的业务值或 ID", table)
		}
	}
}

func feeSeedTableSnapshot(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	var snapshot string
	// 种子会显式更新 THC/STORAGE 的 updated_at；其他列（含 ID/created_at）必须保持。
	if err := db.QueryRow(`SELECT jsonb_agg(to_jsonb(t)-'updated_at' ORDER BY id)::text FROM ` + table + ` t`).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
