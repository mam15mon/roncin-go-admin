package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

const billingUnitCatalogMigration = "20261020100000_billing_unit_catalog.sql"

func TestBillingUnitCatalogColdStartPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	if n, err := d.db.Organization.Query().Count(ctx); err != nil || n != 0 {
		t.Fatalf("冷启动目录不应依赖组织: count=%d err=%v", n, err)
	}
	assertDefaultBillingUnitCatalog(t, d)

	// 完整迁移重复执行不会再新增单位，也不会重复记录 revision。
	before := billingUnitCatalogSnapshot(t, d, `SELECT COALESCE(jsonb_agg(to_jsonb(u) ORDER BY code), '[]')::text FROM billing_units u`)
	if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("重复执行迁移: %v", err)
	}
	if after := billingUnitCatalogSnapshot(t, d, `SELECT COALESCE(jsonb_agg(to_jsonb(u) ORDER BY code), '[]')::text FROM billing_units u`); before != after {
		t.Fatal("重复迁移改变了单位记录")
	}
	var revisions int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261020100000_billing_unit_catalog'`).Scan(&revisions); err != nil || revisions != 1 {
		t.Fatalf("补缺迁移应记录一次: count=%d err=%v", revisions, err)
	}
}

func TestBillingUnitCatalogUpgradeAndSeedPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, billingUnitCatalogMigration)
	t.Cleanup(cleanup)
	ctx := context.Background()
	fixture := newFinanceBillPostgresFixture(t, d)
	if err := d.WithTx(ctx, func(tx *ent.Tx) error {
		return initializeCompanyFeeSettings(ctx, tx.Client(), fixture.organizationID)
	}); err != nil {
		t.Fatalf("构造公司默认费用科目: %v", err)
	}
	t.Cleanup(func() {
		if _, err := d.sqlDB.ExecContext(ctx, `DELETE FROM fee_settings WHERE organization_id=$1`, fixture.organizationID); err != nil {
			t.Error(err)
		}
		if _, err := d.sqlDB.ExecContext(ctx, `DELETE FROM taxable_services WHERE organization_id=$1`, fixture.organizationID); err != nil {
			t.Error(err)
		}
	})
	// 同码自定义名称、停用、箱型和整数规则均保留；未知单位也保持原样。
	if _, err := d.sqlDB.ExecContext(ctx, `UPDATE billing_units SET name='管理员票单位', enabled=false, sort_order=999, is_container_unit=true, quantity_must_be_integer=false, search_keywords=$1 WHERE code='BL'`, searchtext.Build("管理员票单位")); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.BillingUnit.Create().SetCode("CUSTOM").SetName("自定义单位").SetEnabled(false).SetQuantityMustBeInteger(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	feeID := fixture.createUnbilledFee("catalog-upgrade")
	unit, err := d.db.BillingUnit.Query().Where(billingunitent.CodeEQ("BL")).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.OrderFee.UpdateOneID(feeID).SetBillingUnitID(unit.ID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	unitBefore := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(u) ORDER BY code)::text FROM billing_units u`)
	feeBefore := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM order_fees f`)
	settingBefore := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM fee_settings f`)
	if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("执行升级迁移: %v", err)
	}
	oldCodes := `'CONT','BL','CBM','TON','SET','DOC','DAY','CUSTOM'`
	unitAfter := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(u) ORDER BY code)::text FROM billing_units u WHERE code IN (`+oldCodes+`)`)
	if unitBefore != unitAfter {
		t.Fatal("升级迁移覆盖了已有单位配置")
	}
	if after := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM order_fees f`); after != feeBefore {
		t.Fatal("升级迁移改写了已有费用")
	}
	if after := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM fee_settings f`); after != settingBefore {
		t.Fatal("升级迁移改写了公司费用科目")
	}
	// 直接重放本次补缺 SQL，验证冲突跳过的实际行为。
	content, err := os.ReadFile(filepath.Join("..", "..", "migrations", billingUnitCatalogMigration))
	if err != nil {
		t.Fatal(err)
	}
	allBefore := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(u) ORDER BY code)::text FROM billing_units u`)
	if _, err := d.sqlDB.ExecContext(ctx, string(content)); err != nil {
		t.Fatal(err)
	}
	if after := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(u) ORDER BY code)::text FROM billing_units u`); after != allBefore {
		t.Fatal("补缺迁移重复执行改变了记录")
	}

	// 开发同步补回缺失单位，已有配置与引用继续保留。
	if _, err := d.db.BillingUnit.Delete().Where(billingunitent.CodeIn("KG", "RT")).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := d.WithTx(ctx, func(tx *ent.Tx) error {
			units, err := CreateDefaultBillingUnits(ctx, tx)
			if err == nil && len(units) != 38 {
				t.Fatalf("公共单位目录应有38项，实际%d", len(units))
			}
			return err
		}); err != nil {
			t.Fatalf("开发单位同步第%d次失败: %v", i+1, err)
		}
	}
	unitAfter = billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(u) ORDER BY code)::text FROM billing_units u WHERE code IN (`+oldCodes+`)`)
	if unitBefore != unitAfter {
		t.Fatal("开发同步覆盖了已有单位配置")
	}
	if after := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM order_fees f`); after != feeBefore {
		t.Fatal("开发同步改写了已有费用")
	}
	if after := billingUnitCatalogSnapshot(t, d, `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM fee_settings f`); after != settingBefore {
		t.Fatal("开发同步改写了公司费用科目")
	}
	for _, code := range []string{"KG", "RT", "PCS", "PLT", "HOUR"} {
		unit, err := d.db.BillingUnit.Query().Where(billingunitent.CodeEQ(code)).Only(ctx)
		if err != nil || unit.SearchKeywords != searchtext.Build(unit.Name) {
			t.Fatalf("单位 %s 未正确补齐检索键: unit=%v err=%v", code, unit, err)
		}
	}
}

func assertDefaultBillingUnitCatalog(t *testing.T, d *Data) {
	t.Helper()
	ctx := context.Background()
	units, err := d.db.BillingUnit.Query().All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 38 {
		t.Fatalf("冷启动公共目录应有38项，实际%d", len(units))
	}
	byCode := make(map[string]*ent.BillingUnit, len(units))
	for _, unit := range units {
		byCode[unit.Code] = unit
	}
	for _, seed := range defaultBillingUnitSeeds {
		unit := byCode[seed.code]
		if unit == nil {
			t.Fatalf("正式迁移缺失单位 %s", seed.code)
		}
		if unit.Name != seed.name || unit.IsContainerUnit != seed.isContainer || unit.QuantityMustBeInteger != seed.quantityMustBeInteger || unit.SortOrder != seed.sortOrder || !unit.Enabled || unit.SearchKeywords != searchtext.Build(seed.name) {
			t.Errorf("单位 %s 与开发初始化目录不一致: %+v", seed.code, unit)
		}
	}
	for _, keyword := range []string{"公斤", "GONGJIN", "GJ", "托盘", "TUOPAN", "TP", "PKG"} {
		n, err := d.db.BillingUnit.Query().Where(billingunitent.Or(billingunitent.CodeContainsFold(keyword), billingunitent.NameContainsFold(keyword), billingunitent.SearchKeywordsContainsFold(keyword))).Count(ctx)
		if err != nil || n == 0 {
			t.Errorf("公共单位关键字 %s 应可检索: count=%d err=%v", keyword, n, err)
		}
	}
}

func billingUnitCatalogSnapshot(t *testing.T, d *Data, query string) string {
	t.Helper()
	var snapshot string
	if err := d.sqlDB.QueryRowContext(context.Background(), query).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}
