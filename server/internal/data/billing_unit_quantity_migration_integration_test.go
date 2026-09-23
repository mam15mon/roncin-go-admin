package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBillingUnitQuantityRuleMigrationReplay(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	unit, err := data.db.BillingUnit.Create().SetCode("QTY_UNKNOWN").SetName("未知单位").Save(ctx)
	if err != nil {
		t.Fatalf("创建未知单位: %v", err)
	}
	fixture := newFinanceBillPostgresFixture(t, data)
	feeID := fixture.createUnbilledFee("quantity-migration")
	if _, err := data.db.OrderFee.UpdateOneID(feeID).SetBillingUnitID(unit.ID).SetQuantity("4.4000").Save(ctx); err != nil {
		t.Fatalf("关联已有费用: %v", err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, `ALTER TABLE billing_units DROP COLUMN quantity_must_be_integer`); err != nil {
		t.Fatalf("恢复迁移前列状态: %v", err)
	}
	path := filepath.Join("..", "..", "migrations", "20260923115000_billing_unit_quantity_rule.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取正式迁移: %v", err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, string(content)); err != nil {
		t.Fatalf("重放正式迁移: %v", err)
	}
	for _, code := range []string{"BL", "CONT", "SET", "DOC"} {
		var mustBeInteger bool
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT quantity_must_be_integer FROM billing_units WHERE code=$1`, code).Scan(&mustBeInteger); err != nil || !mustBeInteger {
			t.Fatalf("离散单位 %s 应预设整数: value=%v err=%v", code, mustBeInteger, err)
		}
	}
	for _, code := range []string{"CBM", "TON", "DAY", "QTY_UNKNOWN"} {
		var mustBeInteger bool
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT quantity_must_be_integer FROM billing_units WHERE code=$1`, code).Scan(&mustBeInteger); err != nil || mustBeInteger {
			t.Fatalf("单位 %s 应允许小数: value=%v err=%v", code, mustBeInteger, err)
		}
	}
	if _, err := data.sqlDB.ExecContext(ctx, `UPDATE billing_units SET quantity_must_be_integer=true WHERE code IN ('DAY','QTY_UNKNOWN')`); err != nil {
		t.Fatalf("构造已应用旧费用种子的数量规则: %v", err)
	}
	correctionPath := filepath.Join("..", "..", "migrations", "20260923121000_billing_unit_day_quantity_rule.sql")
	correction, err := os.ReadFile(correctionPath)
	if err != nil {
		t.Fatalf("读取正式修正迁移: %v", err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, string(correction)); err != nil {
		t.Fatalf("重放 DAY 数量规则修正迁移: %v", err)
	}
	for code, expected := range map[string]bool{"DAY": false, "QTY_UNKNOWN": true} {
		var mustBeInteger bool
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT quantity_must_be_integer FROM billing_units WHERE code=$1`, code).Scan(&mustBeInteger); err != nil || mustBeInteger != expected {
			t.Fatalf("修正迁移后单位 %s 数量规则不符: value=%v want=%v err=%v", code, mustBeInteger, expected, err)
		}
	}
	var quantity string
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT quantity FROM order_fees WHERE id=$1`, feeID).Scan(&quantity); err != nil || quantity != "4.4000" {
		t.Fatalf("迁移改写了已有费用数量: quantity=%s err=%v", quantity, err)
	}
}
