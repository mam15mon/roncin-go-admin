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

// 只在随机隔离 schema 执行正式迁移，绝不修改开发库业务表。
func companyBoundaryFixture(t *testing.T) (*sql.DB, string) {
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
	schema := "boundary_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	mustBoundarySQL(t, db, `CREATE SCHEMA "`+schema+`"`)
	t.Cleanup(func() {
		if _, err := db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); err != nil {
			t.Error(err)
		}
	})
	mustBoundarySQL(t, db, `SET search_path TO "`+schema+`",public`)
	dir := filepath.Join("..", "..", "..", "migrations")
	old := t.TempDir()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "20260922100000" {
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
	mustBoundarySQL(t, db, `INSERT INTO organizations(id,created_at,updated_at,code,name,kind,base_currency) VALUES ('10000000-0000-0000-0000-000000000001',now(),now(),'migration-hq','总部','headquarters','CNY');
 INSERT INTO organizations(id,created_at,updated_at,code,name,kind,base_currency,parent_id) VALUES
 ('10000000-0000-0000-0000-000000000002',now(),now(),'migration-a','A','company','CNY','10000000-0000-0000-0000-000000000001'),
 ('10000000-0000-0000-0000-000000000003',now(),now(),'migration-b','B','company','CNY','10000000-0000-0000-0000-000000000001');
 INSERT INTO organizations(id,created_at,updated_at,code,name,kind,parent_id) VALUES ('10000000-0000-0000-0000-000000000004',now(),now(),'migration-dept','部门','department','10000000-0000-0000-0000-000000000002');`)
	return db, dir
}
func mustBoundarySQL(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatal(err)
	}
}
func assertBoundaryCount(t *testing.T, db *sql.DB, q string, want int) {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("数量得到 %d，期望 %d: %s", n, want, q)
	}
}

func TestCompanyBoundaryMigrationCopiesPoliciesAndPreservesRoots(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	mustBoundarySQL(t, db, `INSERT INTO users(id,created_at,updated_at,username,display_name,password_hash) VALUES('20000000-0000-0000-0000-000000000001',now(),now(),'migration-user','管理员','unused');
 INSERT INTO finance_custom_settings(id,created_at,updated_at,organization_id,billed_fee_edit_enabled,billed_fee_name_editable,credit_limit_selection_allowed,version,updated_by) VALUES(gen_random_uuid(),'2026-09-01T00:00:00Z','2026-09-02T00:00:00Z','10000000-0000-0000-0000-000000000001',true,true,false,7,'20000000-0000-0000-0000-000000000001');
 INSERT INTO ports(id,created_at,updated_at,un_locode,name_en,country_code,transport_modes,organization_id) VALUES('30000000-0000-0000-0000-000000000001',now(),now(),'CNXYZ','Test','CN','["SEA"]','10000000-0000-0000-0000-000000000002');`)
	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	assertBoundaryCount(t, db, `SELECT count(*) FROM organizations WHERE id='10000000-0000-0000-0000-000000000001' AND kind='system' AND name='系统管理' AND parent_id IS NULL`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM organizations WHERE code IN ('migration-a','migration-b') AND kind='company' AND parent_id IS NULL`, 2)
	assertBoundaryCount(t, db, `SELECT count(*) FROM organizations WHERE code='migration-dept' AND parent_id='10000000-0000-0000-0000-000000000002'`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM finance_custom_settings WHERE billed_fee_edit_enabled AND billed_fee_name_editable AND NOT credit_limit_selection_allowed AND version=7 AND created_at='2026-09-01T00:00:00Z' AND updated_at='2026-09-02T00:00:00Z'`, 2)
	assertBoundaryCount(t, db, `SELECT count(*) FROM ports WHERE id='30000000-0000-0000-0000-000000000001'`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name IN ('ports','airports') AND column_name='organization_id'`, 0)
	mustBoundarySQL(t, db, `UPDATE finance_custom_settings SET credit_limit_selection_allowed=true WHERE organization_id='10000000-0000-0000-0000-000000000002'`)
	assertBoundaryCount(t, db, `SELECT count(*) FROM finance_custom_settings WHERE NOT credit_limit_selection_allowed`, 1)
}

func TestCompanyBoundaryMigrationRejectsDuplicatePortsAtomically(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	mustBoundarySQL(t, db, `INSERT INTO ports(id,created_at,updated_at,un_locode,name_en,country_code,transport_modes,organization_id) VALUES(gen_random_uuid(),now(),now(),'CNXYZ','Shared','CN','["SEA"]',NULL),(gen_random_uuid(),now(),now(),'CNXYZ','Local differs','CN','["SEA"]','10000000-0000-0000-0000-000000000002')`)
	err := Apply(context.Background(), db, dir)
	if err == nil || !strings.Contains(err.Error(), "港口存在重复代码") {
		t.Fatalf("期望地点冲突，得到 %v", err)
	}
	assertBoundaryCount(t, db, `SELECT count(*) FROM ports WHERE un_locode='CNXYZ'`, 2)
	assertBoundaryCount(t, db, `SELECT count(*) FROM organizations WHERE code='migration-hq' AND kind='headquarters'`, 1)
}

func seedBoundarySharedFee(t *testing.T, db *sql.DB) {
	t.Helper()
	mustBoundarySQL(t, db, `INSERT INTO master_data_items(id,created_at,updated_at,kind,code,name) VALUES('40000000-0000-0000-0000-000000000001',now(),now(),'charge_category','MIGRATION_FEE','迁移费用');
 INSERT INTO billing_units(id,created_at,updated_at,code,name) VALUES('40000000-0000-0000-0000-000000000002',now(),now(),'MIGRATION_UNIT','票');
 INSERT INTO taxable_services(id,created_at,updated_at,organization_id,name,short_name,goods_code,default_tax_rate) VALUES('40000000-0000-0000-0000-000000000003',now(),now(),'10000000-0000-0000-0000-000000000001','迁移税务','税务','123456',6);
 INSERT INTO fee_settings(id,created_at,updated_at,organization_id,fee_code,name_zh,charge_category_id,default_currency,billing_unit_id,tax_rate,taxable_service_id) VALUES('40000000-0000-0000-0000-000000000004',now(),now(),NULL,'MIGRATION_FEE','迁移费用','40000000-0000-0000-0000-000000000001','CNY','40000000-0000-0000-0000-000000000002',6,'40000000-0000-0000-0000-000000000003');`)
}
func TestCompanyBoundaryMigrationCopiesSharedFeeAndTax(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	seedBoundarySharedFee(t, db)
	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_setting_templates WHERE fee_code='MIGRATION_FEE' AND taxable_service_name='迁移税务' AND taxable_service_default_tax_rate=6`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_settings f JOIN taxable_services t ON t.id=f.taxable_service_id WHERE f.fee_code='MIGRATION_FEE' AND f.organization_id=t.organization_id AND f.organization_id IN ('10000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000003')`, 2)
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_settings WHERE organization_id IS NULL`, 0)
	mustBoundarySQL(t, db, `UPDATE fee_setting_templates SET name_zh='模板新名称' WHERE fee_code='MIGRATION_FEE'`)
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_settings WHERE fee_code='MIGRATION_FEE' AND name_zh='迁移费用'`, 2)
}
func TestCompanyBoundaryMigrationRejectsTaxConflictAtomically(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	seedBoundarySharedFee(t, db)
	mustBoundarySQL(t, db, `INSERT INTO taxable_services(id,created_at,updated_at,organization_id,name,default_tax_rate) VALUES(gen_random_uuid(),now(),now(),'10000000-0000-0000-0000-000000000002','迁移税务',13)`)
	err := Apply(context.Background(), db, dir)
	if err == nil || !strings.Contains(err.Error(), "初始化费用税务同名配置冲突") {
		t.Fatalf("期望税務冲突，得到 %v", err)
	}
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_settings WHERE fee_code='MIGRATION_FEE' AND organization_id IS NULL`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_settings WHERE fee_code='MIGRATION_FEE' AND organization_id IS NOT NULL`, 0)
	assertBoundaryCount(t, db, `SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name='fee_setting_templates'`, 0)
}

func seedBoundaryFeeReferences(t *testing.T, db *sql.DB) {
	t.Helper()
	mustBoundarySQL(t, db, `INSERT INTO users(id,created_at,updated_at,username,display_name,password_hash) VALUES('20000000-0000-0000-0000-000000000001',now(),now(),'reference-user','测试','unused');
 INSERT INTO partners(id,created_at,updated_at,organization_id,legal_name,normalized_name) VALUES('50000000-0000-0000-0000-000000000001',now(),now(),'10000000-0000-0000-0000-000000000002','迁移客商','迁移客商');
 INSERT INTO orders(id,created_at,updated_at,organization_id,order_no,idempotency_key,customer_id,business_type,trade_direction,payment_term) VALUES('50000000-0000-0000-0000-000000000002',now(),now(),'10000000-0000-0000-0000-000000000002','migration-order','migration-order','50000000-0000-0000-0000-000000000001','SE','export','PREPAID');
 INSERT INTO order_fees(id,created_at,updated_at,order_id,idempotency_key,direction,fee_setting_id,fee_code,fee_name,settlement_party_id,billing_unit,quantity,unit_price,total_amount,net_amount,tax_amount,currency,exchange_rate,exchange_rate_source,exchange_rate_date,base_currency,base_currency_amount,expense_date,tax_rate,taxable_service_name) VALUES('50000000-0000-0000-0000-000000000003',now(),now(),'50000000-0000-0000-0000-000000000002','migration-fee','PAYABLE','40000000-0000-0000-0000-000000000004','OLD','历史费用快照','50000000-0000-0000-0000-000000000001','票',1,106,106,100,6,'CNY',1,'MANUAL','2026-09-22','CNY',106,'2026-09-22',6,'历史税务快照');
 INSERT INTO order_fee_supplement_requests(id,created_at,updated_at,organization_id,order_id,lock_basis,business_lock_generation,idempotency_key,request_fingerprint,direction,fee_setting_id,fee_code,fee_name,settlement_party_id,billing_unit,quantity,unit_price,total_amount,net_amount,tax_amount,currency,exchange_rate,exchange_rate_source,exchange_rate_date,base_currency,base_currency_amount,expense_date,tax_rate,taxable_service_name,reason,requested_by,requested_at) SELECT '50000000-0000-0000-0000-000000000004',created_at,updated_at,'10000000-0000-0000-0000-000000000002',order_id,'BUSINESS',1,'migration-supplement','fingerprint',direction,fee_setting_id,fee_code,fee_name,settlement_party_id,billing_unit,quantity,unit_price,total_amount,net_amount,tax_amount,currency,exchange_rate,exchange_rate_source,exchange_rate_date,base_currency,base_currency_amount,expense_date,tax_rate,taxable_service_name,'补录','20000000-0000-0000-0000-000000000001',now() FROM order_fees WHERE id='50000000-0000-0000-0000-000000000003';`)
}
func TestCompanyBoundaryMigrationRewritesFeeReferencesWithoutSnapshots(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	seedBoundarySharedFee(t, db)
	seedBoundaryFeeReferences(t, db)
	var beforeFee, beforeRequest string
	if err := db.QueryRow(`SELECT (to_jsonb(f)-'fee_setting_id')::text FROM order_fees f WHERE id='50000000-0000-0000-0000-000000000003'`).Scan(&beforeFee); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT (to_jsonb(f)-'fee_setting_id')::text FROM order_fee_supplement_requests f WHERE id='50000000-0000-0000-0000-000000000004'`).Scan(&beforeRequest); err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ table, id, before string }{{"order_fees", "50000000-0000-0000-0000-000000000003", beforeFee}, {"order_fee_supplement_requests", "50000000-0000-0000-0000-000000000004", beforeRequest}} {
		var after string
		if err := db.QueryRow(`SELECT (to_jsonb(f)-'fee_setting_id')::text FROM `+test.table+` f WHERE id=$1`, test.id).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if after != test.before {
			t.Fatalf("%s 费用快照或其他字段被修改", test.table)
		}
		assertBoundaryCount(t, db, `SELECT count(*) FROM `+test.table+` f JOIN fee_settings s ON s.id=f.fee_setting_id WHERE f.id='`+test.id+`' AND s.organization_id='10000000-0000-0000-0000-000000000002'`, 1)
	}
}

func TestCompanyBoundaryMigrationRejectsReferencedLocalOverride(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	seedBoundarySharedFee(t, db)
	seedBoundaryFeeReferences(t, db)
	mustBoundarySQL(t, db, `INSERT INTO taxable_services(id,created_at,updated_at,organization_id,name,short_name,goods_code,default_tax_rate) VALUES('60000000-0000-0000-0000-000000000001',now(),now(),'10000000-0000-0000-0000-000000000002','迁移税务','税务','123456',6);
 INSERT INTO fee_settings(id,created_at,updated_at,organization_id,fee_code,name_zh,charge_category_id,default_currency,billing_unit_id,tax_rate,taxable_service_id) VALUES(gen_random_uuid(),now(),now(),'10000000-0000-0000-0000-000000000002','MIGRATION_FEE','不同费用配置','40000000-0000-0000-0000-000000000001','CNY','40000000-0000-0000-0000-000000000002',6,'60000000-0000-0000-0000-000000000001');`)
	err := Apply(context.Background(), db, dir)
	if err == nil || !strings.Contains(err.Error(), "共享费用引用与本地同码配置冲突") {
		t.Fatalf("期望引用冲突，得到 %v", err)
	}
	assertBoundaryCount(t, db, `SELECT count(*) FROM orders WHERE id='50000000-0000-0000-0000-000000000002'`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM order_fees WHERE fee_setting_id='40000000-0000-0000-0000-000000000004'`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM organizations WHERE code='migration-hq' AND kind='headquarters'`, 1)
}
func TestCompanyBoundaryMigrationRejectsSystemPrivateFee(t *testing.T) {
	db, dir := companyBoundaryFixture(t)
	seedBoundarySharedFee(t, db)
	mustBoundarySQL(t, db, `UPDATE fee_settings SET organization_id='10000000-0000-0000-0000-000000000001' WHERE id='40000000-0000-0000-0000-000000000004'`)
	err := Apply(context.Background(), db, dir)
	if err == nil || !strings.Contains(err.Error(), "费用科目公司或税务归属不合法") {
		t.Fatalf("期望未决归属阻断，得到 %v", err)
	}
	assertBoundaryCount(t, db, `SELECT count(*) FROM fee_settings WHERE id='40000000-0000-0000-0000-000000000004' AND organization_id='10000000-0000-0000-0000-000000000001'`, 1)
	assertBoundaryCount(t, db, `SELECT count(*) FROM organizations WHERE code='migration-hq' AND kind='headquarters'`, 1)
}
