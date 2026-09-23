package data

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

func TestOrderFeeQuantityRuleUpdatePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	fixture := newFinanceBillPostgresFixture(t, data)
	ctx := context.Background()
	actor, err := data.db.User.Create().SetDisplayName("费用数量规则测试用户").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试操作人: %v", err)
	}
	unit, err := data.db.BillingUnit.Create().SetCode("QTY-" + fixture.suffix).SetName("测试票").SetQuantityMustBeInteger(true).Save(ctx)
	if err != nil {
		t.Fatalf("创建整数单位: %v", err)
	}
	feeID := fixture.createUnbilledFee("quantity-rule")
	if _, err := data.db.OrderFee.UpdateOneID(feeID).SetBillingUnitID(unit.ID).SetQuantity("4.4000").SetTotalAmount("440.00000000").SetNetAmount("440.00000000").SetBaseCurrencyAmount("440.00000000").Save(ctx); err != nil {
		t.Fatalf("构造旧小数费用: %v", err)
	}
	repo := NewOrderFeeRepo(data)
	update := func(quantity string) error {
		current, getErr := repo.Get(ctx, fixture.organizationID, fixture.orderID, feeID)
		if getErr != nil {
			return getErr
		}
		current.Quantity = decimal.RequireFromString(quantity)
		current.TotalAmount = current.Quantity.Mul(current.UnitPrice)
		current.NetAmount = current.TotalAmount
		current.BaseCurrencyAmount = current.TotalAmount
		_, updateErr := repo.Update(ctx, fixture.organizationID, fixture.orderID, feeID, current, nil, &biz.AuditEvent{
			OrganizationID: &fixture.organizationID, UserID: &actor.ID,
			Action: "order.fee.update", Result: "success", ResourceType: "order_fee", ResourceID: feeID.String(),
		})
		return updateErr
	}
	if err := update("4.4"); err != nil {
		t.Fatalf("旧小数数量数值未变，应允许修改其他字段: %v", err)
	}
	if err := update("4.5"); err != biz.ErrOrderFeeQuantityMustBeInteger {
		t.Fatalf("变更为小数应拒绝，得到 %v", err)
	}
	if err := update("5"); err != nil {
		t.Fatalf("变更为正整数应允许: %v", err)
	}
	current, err := repo.Get(ctx, fixture.organizationID, fixture.orderID, feeID)
	if err != nil {
		t.Fatalf("读取更新后的费用: %v", err)
	}
	current.ID = uuid.New()
	current.IdempotencyKey = uuid.NewString()
	current.Quantity = decimal.RequireFromString("4.4")
	current.TotalAmount = current.Quantity.Mul(current.UnitPrice)
	current.NetAmount = current.TotalAmount
	current.BaseCurrencyAmount = current.TotalAmount
	if _, err := repo.Add(ctx, fixture.organizationID, fixture.orderID, current, nil); err != biz.ErrOrderFeeQuantityMustBeInteger {
		t.Fatalf("整数单位新增小数费用应拒绝，得到 %v", err)
	}
}

func TestOrderFeeResolveCatalogForDisabledCurrentUnitPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)
	order := fixture.createOrder("QTY-DISABLED-" + fixture.suffix)
	if _, err := fixture.data.db.BillingUnit.UpdateOneID(fixture.billingUnitID).SetEnabled(false).Save(fixture.ctx); err != nil {
		t.Fatalf("停用已有计费单位: %v", err)
	}
	repo := NewOrderFeeRepo(fixture.data)
	if _, err := repo.ResolveCatalog(fixture.ctx, fixture.organizationID, order.ID, fixture.feeSettingID, fixture.billingUnitID, true); err != nil {
		t.Fatalf("维护原计费单位的费用应允许解析目录: %v", err)
	}
	if _, err := repo.ResolveCatalog(fixture.ctx, fixture.organizationID, order.ID, fixture.feeSettingID, fixture.billingUnitID, false); err != biz.ErrOrderFeeSettingInvalid {
		t.Fatalf("新增费用仍应拒绝已停用的默认单位，得到 %v", err)
	}
}
