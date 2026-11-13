package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// TestCommissionNetBasePostgres 验证提成基数为不含税口径：
// 已建账账单行（税率与费用行不同）与未建账费用并存，部分核销，创建后经济重算一致。
func TestCommissionNetBasePostgres(t *testing.T) {
	f := newCommissionPostgresFixture(t)
	ctx := context.Background()
	db := f.data.db

	// 应收：含税 1060 / 未税 1000，账单与账单行同口径，核销 1000（部分核销）。
	recFee, err := db.OrderFee.Query().Where(fee.OrderIDEQ(f.orderID), fee.DirectionEQ(fee.DirectionRECEIVABLE)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.OrderFee.UpdateOneID(recFee.ID).SetTotalAmount("1060.00000000").SetNetAmount("1000.00000000").SetTaxAmount("60.00000000").SetBaseCurrencyAmount("1060.00000000").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = db.FinanceBillLine.Update().Where(billline.OrderFeeIDEQ(recFee.ID)).SetTotalAmount("1060.00000000").SetNetAmount("1000.00000000").SetTaxAmount("60.00000000").SetBaseCurrencyAmount("1060.00000000").Save(ctx); err != nil {
		t.Fatal(err)
	}
	if err = db.FinanceBill.UpdateOneID(f.billID).SetTotalAmount("1060.00000000").SetNetAmount("1000.00000000").SetTaxAmount("60.00000000").SetBaseCurrencyAmount("1060.00000000").Exec(ctx); err != nil {
		t.Fatal(err)
	}

	// 应付已建账：费用行 6%（106/100），账单行 13%（113/100），两者税率不同。
	payFee, err := db.OrderFee.Query().Where(fee.OrderIDEQ(f.orderID), fee.DirectionEQ(fee.DirectionPAYABLE)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.OrderFee.UpdateOneID(payFee.ID).SetTotalAmount("106.00000000").SetNetAmount("100.00000000").SetTaxAmount("6.00000000").SetBaseCurrencyAmount("106.00000000").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	payLine, err := db.FinanceBillLine.Query().Where(billline.OrderFeeIDEQ(payFee.ID)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.FinanceBillLine.UpdateOneID(payLine.ID).SetTotalAmount("113.00000000").SetNetAmount("100.00000000").SetTaxAmount("13.00000000").SetBaseCurrencyAmount("113.00000000").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err = db.FinanceBill.UpdateOneID(payLine.BillID).SetTotalAmount("113.00000000").SetNetAmount("100.00000000").SetTaxAmount("13.00000000").SetBaseCurrencyAmount("113.00000000").Exec(ctx); err != nil {
		t.Fatal(err)
	}

	// 应付未建账：含税 53 / 未税 50。
	unbilled, err := db.OrderFee.Create().SetOrderID(f.orderID).SetIdempotencyKey("fee-unbilled-" + f.suffix).
		SetDirection(fee.DirectionPAYABLE).SetFeeCode("UNBILLED").SetFeeName("未建账成本").SetSettlementPartyID(f.customerID).
		SetBillingUnit("票").SetQuantity("1.0000").SetUnitPrice("50.0000").
		SetTotalAmount("53.00000000").SetNetAmount("50.00000000").SetTaxAmount("3.00000000").
		SetCurrency("CNY").SetExchangeRate("1.00000000").SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).SetBaseCurrency("CNY").SetBaseCurrencyAmount("53.00000000").
		SetVersion(1).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	unbilledID := unbilled.ID

	requests := []uuid.UUID{}
	commissionCorrectionTestCleanup(t, f, &requests)
	uc := f.newUsecase(NewCommissionRepo(f.data))
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("net-base"))
	if err != nil {
		t.Fatal(err)
	}
	// 应收未税 1000、应付未税 100+50；已实现收入 = 1000 × 1000/1060。
	want := map[string]string{
		"revenue": "943.39622642", "cost": "141.50943396", "profit": "801.88679246", "amount": "80.18867925",
	}
	if len(created.Lines) != 1 {
		t.Fatalf("明细行数 %d", len(created.Lines))
	}
	line := created.Lines[0]
	if line.RealizedRevenue.StringFixed(8) != want["revenue"] || line.AllocatedCost.StringFixed(8) != want["cost"] ||
		line.RealizedProfit.StringFixed(8) != want["profit"] || line.CommissionAmount.StringFixed(8) != want["amount"] ||
		created.CommissionAmount.StringFixed(8) != want["amount"] {
		t.Fatalf("创建结果非不含税口径: rev=%s cost=%s profit=%s amount=%s", line.RealizedRevenue, line.AllocatedCost, line.RealizedProfit, line.CommissionAmount)
	}
	if !line.TotalReceivableSnapshot.Equal(decimal.NewFromInt(1000)) || !line.TotalPayableSnapshot.Equal(decimal.NewFromInt(150)) {
		t.Fatalf("分母快照应为不含税 1000/150: %s/%s", line.TotalReceivableSnapshot, line.TotalPayableSnapshot)
	}
	if created.CalculationVersion != biz.CommissionCalculationVersion {
		t.Fatalf("计算版本不符: %s", created.CalculationVersion)
	}
	// 含未建账费用时不允许确认：取消该草稿，删除未建账费用后重新生成并确认，再验证经济重算。
	if _, err = uc.Cancel(ctx, f.organizationID, f.actorID, created.ID, created.Version, "回归取消"); err != nil {
		t.Fatal(err)
	}
	if err = db.OrderFee.DeleteOneID(unbilledID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	created, err = uc.Create(ctx, f.organizationID, f.actorID, f.input("net-base-2"))
	if err != nil {
		t.Fatal(err)
	}
	// 应付未税 100：成本 943.39622642×100/1000=94.33962264，利润 849.05660378，提成 84.90566038。
	if created.CommissionAmount.StringFixed(8) != "84.90566038" {
		t.Fatalf("删除未建账费用后提成应 84.90566038: %s", created.CommissionAmount)
	}
	parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}

	// 经济重算：删除未建账费用后应付未税 100（提成已变），再追加一笔含税 106 / 未税 100 的应付，目标应付未税 200。
	fees, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	var base *biz.OrderFee
	for _, item := range fees {
		if item.Direction == biz.OrderFeePayable {
			base = item
			break
		}
	}
	after := *base
	after.ID = uuid.New()
	after.Quantity = decimal.NewFromInt(1)
	after.UnitPrice = decimal.NewFromInt(100)
	after.TotalAmount = decimal.NewFromInt(106)
	after.NetAmount = decimal.NewFromInt(100)
	after.TaxAmount = decimal.NewFromInt(6)
	after.BaseCurrencyAmount = decimal.NewFromInt(106)
	impact := applyCommissionCorrectionTest(t, f, &requests, nil, &after)
	// 目标：成本 943.39622642×200/1000=188.67924528，利润 754.71698114，提成 75.47169811。
	if len(impact.Lines) != 1 || impact.Lines[0].TargetAmount.StringFixed(8) != "75.47169811" || impact.Lines[0].Difference.StringFixed(8) != "-9.43396227" {
		t.Fatalf("重算目标非不含税口径: %+v", impact.Lines)
	}
	result, err := uc.Get(ctx, f.organizationID, parent.ID)
	if err != nil || result.EffectiveCommissionAmount.StringFixed(8) != "75.47169811" {
		t.Fatalf("有效权益应 75.47169811: %+v %v", result, err)
	}
}
