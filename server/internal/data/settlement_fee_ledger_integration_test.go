package data

// 集成测试覆盖费用台账「有效结清金额 = 有效核销分摊 + 有效对冲分摊」口径：
// 列表筛选 SQL、列表行投影与订单详情投影三处一致（FIN-01/FIN-02），
// 以及核销候选的未结清过滤发生在 LIMIT 之前、不被大量已结清记录截断（FIN-03）。

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financecommissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	financecommissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	financecommissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	verificationallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

type feeLedgerPostgresFixture struct {
	*financeBillPostgresFixture
	orderNo string
}

func newFeeLedgerPostgresFixture(t *testing.T, data *Data) *feeLedgerPostgresFixture {
	t.Helper()
	base := newFinanceBillPostgresFixture(t, data)
	order, err := data.db.Order.Get(context.Background(), base.orderID)
	if err != nil {
		t.Fatalf("读取测试订单失败: %v", err)
	}
	fixture := &feeLedgerPostgresFixture{financeBillPostgresFixture: base, orderNo: order.OrderNo}
	// 台账夹具引入对冲、核销、流水和提成数据；这些记录引用账单与往来单位，
	// 必须在基础夹具删除账单/组织之前清理（t.Cleanup 按注册逆序执行）。
	t.Cleanup(fixture.cleanupFeeLedger)
	return fixture
}

func (f *feeLedgerPostgresFixture) cleanupFeeLedger() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{"提成线", func() error {
			_, err := f.data.db.FinanceCommissionLine.Delete().Where(financecommissionlineent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{"提成", func() error {
			_, err := f.data.db.FinanceCommission.Delete().Where(financecommissionent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{"提成规则", func() error {
			_, err := f.data.db.FinanceCommissionRule.Delete().Where(financecommissionruleent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{"对冲分摊", func() error {
			_, err := f.data.db.FinanceNettingAllocation.Delete().Where(nettingallocent.HasNettingWith(financenettingent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{"对冲单", func() error {
			_, err := f.data.db.FinanceNetting.Delete().Where(financenettingent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{"核销分摊", func() error {
			_, err := f.data.db.FinanceVerificationAllocation.Delete().Where(verificationallocent.HasVerificationWith(financeverificationent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{"核销单", func() error {
			_, err := f.data.db.FinanceVerification.Delete().Where(financeverificationent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{"资金流水", func() error {
			_, err := f.data.db.FinanceCashflow.Delete().Where(financecashflowent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理台账测试%s: %v", step.name, err)
		}
	}
}

func (f *feeLedgerPostgresFixture) createLedgerFeeWithStatus(key, total string, status orderfeeent.Status) uuid.UUID {
	f.t.Helper()
	total8 := decimal.RequireFromString(total).StringFixed(8)
	create := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("fee-ledger-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(status).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice(decimal.RequireFromString(total).StringFixed(4)).
		SetTotalAmount(total8).
		SetNetAmount(total8).
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(total8).
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1)
	if status == orderfeeent.StatusCANCELLED {
		// 数据库 CHECK 要求取消费用必须携带取消事实（时间、操作者、原因）。
		actor, err := f.data.db.User.Create().SetDisplayName("台账取消费用用户-" + f.suffix).Save(context.Background())
		if err != nil {
			f.t.Fatalf("创建台账取消费用用户失败: %v", err)
		}
		create = create.
			SetCancelledAt(time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)).
			SetCancelledBy(actor.ID).
			SetCancellationReason("台账集成测试取消")
	}
	fee, err := create.Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建台账测试费用 %s: %v", key, err)
	}
	return fee.ID
}

func (f *feeLedgerPostgresFixture) createLedgerFee(key, total string) uuid.UUID {
	f.t.Helper()
	return f.createLedgerFeeWithStatus(key, total, orderfeeent.StatusCONFIRMED)
}

func (f *feeLedgerPostgresFixture) createConfirmedBill(key, total, billDate string) *ent.FinanceBill {
	f.t.Helper()
	total8 := decimal.RequireFromString(total).StringFixed(8)
	create := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo("BILL-LEDGER-" + key + "-" + f.suffix).
		SetIdempotencyKey("bill-ledger-" + key + "-" + f.suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("账单事务测试客户-" + f.suffix).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount(total8).
		SetNetAmount(total8).
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount(total8).
		SetFeeCount(1).
		SetBillDate(billDate).
		SetVersion(1)
	bill, err := withTestFinanceBillSettlementAccountSnapshot(create, f.accountID, "CNY").Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建台账测试账单 %s: %v", key, err)
	}
	return bill
}

func (f *feeLedgerPostgresFixture) linkBillFee(key string, bill *ent.FinanceBill, feeID uuid.UUID, total string) {
	f.t.Helper()
	total8 := decimal.RequireFromString(total).StringFixed(8)
	_, err := f.data.db.FinanceBillLine.Create().
		SetBillID(bill.ID).
		SetOrderFeeID(feeID).
		SetOrderID(f.orderID).
		SetOrderNo(f.orderNo).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetQuantity("1.0000").
		SetUnitPrice(decimal.RequireFromString(total).StringFixed(4)).
		SetTotalAmount(total8).
		SetNetAmount(total8).
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(total8).
		SetActive(true).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("关联台账测试账单行 %s: %v", key, err)
	}
}

func (f *feeLedgerPostgresFixture) createNetting(key string, status financenettingent.Status, bill *ent.FinanceBill, total string, allocationActive bool) (*ent.FinanceNetting, *ent.FinanceNettingAllocation) {
	f.t.Helper()
	total8 := decimal.RequireFromString(total).StringFixed(8)
	netting, err := f.data.db.FinanceNetting.Create().
		SetOrganizationID(f.organizationID).
		SetNettingNo("NT-" + key + "-" + f.suffix).
		SetIdempotencyKey("netting-ledger-" + key + "-" + f.suffix).
		SetRequestHash("ledger-" + key + "-" + f.suffix).
		SetStatus(status).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("账单事务测试客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount(total8).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(total8).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建台账测试对冲 %s: %v", key, err)
	}
	allocation, err := f.data.db.FinanceNettingAllocation.Create().
		SetNettingID(netting.ID).
		SetBillID(bill.ID).
		SetBillNo(bill.BillNo).
		SetDirection(nettingallocent.DirectionRECEIVABLE).
		SetAmount(total8).
		SetBaseCurrencyAmount(total8).
		SetActive(allocationActive).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建台账测试对冲分摊 %s: %v", key, err)
	}
	return netting, allocation
}

func (f *feeLedgerPostgresFixture) createConfirmedCashflow(key, total, transactionDate string) *ent.FinanceCashflow {
	f.t.Helper()
	total8 := decimal.RequireFromString(total).StringFixed(8)
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(f.organizationID).
		SetFlowNo("FLOW-" + key + "-" + f.suffix).
		SetIdempotencyKey("cashflow-ledger-" + key + "-" + f.suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("账单事务测试客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount(total8).
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount(total8).
		SetTransactionDate(transactionDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建台账测试资金流水 %s: %v", key, err)
	}
	return cashflow
}

func (f *feeLedgerPostgresFixture) createActiveVerification(key string, amount string, buildAllocations func(verificationID uuid.UUID) []*ent.FinanceVerificationAllocationCreate) *ent.FinanceVerification {
	f.t.Helper()
	amount8 := decimal.RequireFromString(amount).StringFixed(8)
	verification, err := f.data.db.FinanceVerification.Create().
		SetOrganizationID(f.organizationID).
		SetVerificationNo("WO-" + key + "-" + f.suffix).
		SetIdempotencyKey("verification-ledger-" + key + "-" + f.suffix).
		SetStatus(financeverificationent.StatusACTIVE).
		SetDirection(financeverificationent.DirectionRECEIVABLE).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("账单事务测试客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount(amount8).
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financeverificationent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseAmount(amount8).
		SetBillBaseAmount(amount8).
		SetCashflowBaseAmount(amount8).
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建台账测试核销 %s: %v", key, err)
	}
	if buildAllocations == nil {
		return verification
	}
	allocations := buildAllocations(verification.ID)
	if len(allocations) == 0 {
		return verification
	}
	if _, err = f.data.db.FinanceVerificationAllocation.CreateBulk(allocations...).Save(context.Background()); err != nil {
		f.t.Fatalf("创建台账测试核销分摊 %s: %v", key, err)
	}
	return verification
}

func (f *feeLedgerPostgresFixture) verificationAllocationCreate(verificationID uuid.UUID, cashflow *ent.FinanceCashflow, bill *ent.FinanceBill, amount string) *ent.FinanceVerificationAllocationCreate {
	amount8 := decimal.RequireFromString(amount).StringFixed(8)
	return f.data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationID).
		SetCashflowID(cashflow.ID).
		SetBillID(bill.ID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo(bill.BillNo).
		SetAmount(amount8).
		SetBillBaseAmount(amount8).
		SetCashflowBaseAmount(amount8).
		SetWriteOffBaseAmount(amount8).
		SetExchangeGainLoss("0.00000000").
		SetActive(true)
}

// createConfirmedCommission 直接落一条 CONFIRMED 提成及其订单提成线，用于财务锁定口径。
func (f *feeLedgerPostgresFixture) createConfirmedCommission(key string, verification *ent.FinanceVerification) {
	f.t.Helper()
	ctx := context.Background()
	employee, err := f.data.db.User.Create().SetDisplayName("台账集成测试用户-" + f.suffix).Save(ctx)
	if err != nil {
		f.t.Fatalf("创建提成测试用户失败: %v", err)
	}
	rule, err := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName("台账锁定测试提成规则-" + f.suffix).
		SetPersonnelRole(financecommissionruleent.PersonnelRoleSALES).
		SetCalculationBasis(financecommissionruleent.CalculationBasisREALIZED_PROFIT).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建台账测试提成规则失败: %v", err)
	}
	zero := "0.00000000"
	base := "100.00000000"
	commission, err := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo("TC-" + key + "-" + f.suffix).
		SetIdempotencyKey("commission-ledger-" + key + "-" + f.suffix).
		SetVerificationID(verification.ID).
		SetVerificationNo(verification.VerificationNo).
		SetEmployeeID(employee.ID).
		SetEmployeeName("台账集成测试用户").
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName(rule.Name).
		SetPersonnelRole(string(rule.PersonnelRole)).
		SetCalculationBasis(string(rule.CalculationBasis)).
		SetStatus(financecommissionent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetRealizedRevenue(base).
		SetAllocatedCost(zero).
		SetRealizedProfit(base).
		SetCommissionBaseAmount(base).
		SetRatePercent("10.0000").
		SetCommissionAmount("10.00000000").
		SetCommissionDate(financeBillIntegrationDate).
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(financecommissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate(financeBillIntegrationDate).
		SetCnyCommissionAmount("10.00000000").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建台账测试提成失败: %v", err)
	}
	_, err = f.data.db.FinanceCommissionLine.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(commission.ID).
		SetOrderID(f.orderID).
		SetOrderNo(f.orderNo).
		SetOrderDate(financeBillIntegrationDate).
		SetCustomerID(f.partnerID).
		SetCustomerCode("CUSTOMER-" + f.suffix).
		SetCustomerName("账单事务测试客户-" + f.suffix).
		SetPersonnelAssignmentID(uuid.Must(uuid.NewV7())).
		SetPersonnelOrganizationID(f.organizationID).
		SetPersonnelAssignedAt(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)).
		SetFeeCount(1).
		SetEmployeeID(employee.ID).
		SetEmployeeName("台账集成测试用户").
		SetPersonnelRole("SALES").
		SetCalculationBasis("REALIZED_PROFIT").
		SetBaseCurrency("CNY").
		SetRealizedRevenue(base).
		SetAllocatedCost(zero).
		SetRealizedProfit(base).
		SetCommissionBaseAmount(base).
		SetRatePercent("10.0000").
		SetCommissionAmount("10.00000000").
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建台账测试提成线失败: %v", err)
	}
}

func findFeeLedgerItem(items []*biz.FeeLedgerItem, feeID uuid.UUID) *biz.FeeLedgerItem {
	for _, item := range items {
		if item.Fee != nil && item.Fee.ID == feeID {
			return item
		}
	}
	return nil
}

func requireFeeLedgerProjection(t *testing.T, scope string, item *biz.FeeLedgerItem, billNo string, progress biz.FeeLedgerFinancialProgress, financeLocked bool) {
	t.Helper()
	if item == nil {
		t.Fatalf("%s：未找到台账条目", scope)
	}
	if item.BillNo != billNo {
		t.Fatalf("%s：bill_no = %q，期望 %q", scope, item.BillNo, billNo)
	}
	if item.FinancialProgress != progress {
		t.Fatalf("%s：financial_progress = %q，期望 %q", scope, item.FinancialProgress, progress)
	}
	if item.FinanceLocked != financeLocked {
		t.Fatalf("%s：finance_locked = %v，期望 %v", scope, item.FinanceLocked, financeLocked)
	}
}

func listFeeLedgerByProgress(t *testing.T, repo biz.SettlementRepo, organizationIDs []uuid.UUID, progress biz.FeeLedgerFinancialProgress) *biz.FeeLedgerResult {
	t.Helper()
	result, err := repo.ListFeeLedger(context.Background(), organizationIDs, biz.FeeLedgerFilter{
		Page: 1, PageSize: 200, FinancialProgress: progress,
	})
	if err != nil {
		t.Fatalf("按进度 %s 查询费用台账失败: %v", progress, err)
	}
	return result
}

func TestFeeLedgerNettingSettlementConsistencyPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	// 每个子测试自建夹具：子测试内的 t.Cleanup 在外层 defer 关闭数据库前执行，
	// 同时让各场景的费用数据互不干扰，断言可以使用精确计数。
	newScopedFixture := func(t *testing.T) (*feeLedgerPostgresFixture, biz.SettlementRepo, context.Context, []uuid.UUID) {
		fixture := newFeeLedgerPostgresFixture(t, data)
		return fixture, NewSettlementRepo(data), context.Background(), []uuid.UUID{fixture.organizationID}
	}

	t.Run("确认对冲全额结清与列表详情一致", func(t *testing.T) {
		fixture, repo, ctx, organizationIDs := newScopedFixture(t)
		feeID := fixture.createLedgerFee("netting-full", "100")
		bill := fixture.createConfirmedBill("netting-full", "100", financeBillIntegrationDate)
		fixture.linkBillFee("netting-full", bill, feeID, "100")
		netting, allocation := fixture.createNetting("netting-full", financenettingent.StatusCONFIRMED, bill, "100", true)

		verified := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerVerifiedUninvoiced)
		if verified.Total != 1 || len(verified.Items) != 1 {
			t.Fatalf("有效结清金额应包含确认对冲：total=%d items=%d", verified.Total, len(verified.Items))
		}
		requireFeeLedgerProjection(t, "列表", verified.Items[0], bill.BillNo, biz.FeeLedgerVerifiedUninvoiced, false)
		if unverified := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerUnverifiedUninvoiced); unverified.Total != 0 {
			t.Fatalf("全额结清后不应再命中未结清筛选：total=%d", unverified.Total)
		}

		detail, err := repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, fixture.orderID)
		if err != nil {
			t.Fatalf("查询台账订单详情失败: %v", err)
		}
		requireFeeLedgerProjection(t, "详情", findFeeLedgerItem(detail.Items, feeID), bill.BillNo, biz.FeeLedgerVerifiedUninvoiced, false)

		// 已取消费用与列表共用同一投影：活动账单行仍保留账单号与推导进度，两处字段必须一致。
		cancelledFeeID := fixture.createLedgerFeeWithStatus("cancelled", "100", orderfeeent.StatusCANCELLED)
		cancelledBill := fixture.createConfirmedBill("cancelled", "100", financeBillIntegrationDate)
		fixture.linkBillFee("cancelled", cancelledBill, cancelledFeeID, "100")
		unfiltered, err := repo.ListFeeLedger(ctx, organizationIDs, biz.FeeLedgerFilter{Page: 1, PageSize: 200})
		if err != nil {
			t.Fatalf("查询费用台账失败: %v", err)
		}
		requireFeeLedgerProjection(t, "取消费用列表", findFeeLedgerItem(unfiltered.Items, cancelledFeeID), cancelledBill.BillNo, biz.FeeLedgerUnverifiedUninvoiced, false)
		detailWithCancelled, err := repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, fixture.orderID)
		if err != nil {
			t.Fatalf("重取台账订单详情失败: %v", err)
		}
		requireFeeLedgerProjection(t, "取消费用详情", findFeeLedgerItem(detailWithCancelled.Items, cancelledFeeID), cancelledBill.BillNo, biz.FeeLedgerUnverifiedUninvoiced, false)

		if _, err = data.db.FinanceNetting.UpdateOneID(netting.ID).SetStatus(financenettingent.StatusREVERSED).Save(ctx); err != nil {
			t.Fatalf("反转对冲失败: %v", err)
		}
		if _, err = data.db.FinanceNettingAllocation.UpdateOneID(allocation.ID).SetActive(false).Save(ctx); err != nil {
			t.Fatalf("失效对冲分摊失败: %v", err)
		}

		if verifiedAfter := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerVerifiedUninvoiced); verifiedAfter.Total != 0 {
			t.Fatalf("反对冲后不应保持已结清：total=%d", verifiedAfter.Total)
		}
		unverifiedAfter := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerUnverifiedUninvoiced)
		if unverifiedAfter.Total != 1 {
			t.Fatalf("反对冲后应回到未结清：total=%d", unverifiedAfter.Total)
		}
		detailAfter, err := repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, fixture.orderID)
		if err != nil {
			t.Fatalf("重新查询台账订单详情失败: %v", err)
		}
		requireFeeLedgerProjection(t, "反对冲后详情", findFeeLedgerItem(detailAfter.Items, feeID), bill.BillNo, biz.FeeLedgerUnverifiedUninvoiced, false)
	})

	t.Run("草稿或已反转父单据的对冲分摊不计入", func(t *testing.T) {
		fixture, repo, ctx, organizationIDs := newScopedFixture(t)
		draftFeeID := fixture.createLedgerFee("netting-draft", "100")
		draftBill := fixture.createConfirmedBill("netting-draft", "100", financeBillIntegrationDate)
		fixture.linkBillFee("netting-draft", draftBill, draftFeeID, "100")
		fixture.createNetting("netting-draft", financenettingent.StatusDRAFT, draftBill, "100", true)

		reversedFeeID := fixture.createLedgerFee("netting-reversed", "100")
		reversedBill := fixture.createConfirmedBill("netting-reversed", "100", financeBillIntegrationDate)
		fixture.linkBillFee("netting-reversed", reversedBill, reversedFeeID, "100")
		fixture.createNetting("netting-reversed", financenettingent.StatusREVERSED, reversedBill, "100", true)

		if settled := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerVerifiedUninvoiced); settled.Total != 0 {
			t.Fatalf("草稿/已反转对冲不应计入有效结清金额：total=%d", settled.Total)
		}
		unverified := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerUnverifiedUninvoiced)
		if unverified.Total != 2 {
			t.Fatalf("草稿/已反转对冲后费用应为未结清：total=%d", unverified.Total)
		}
		detail, err := repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, fixture.orderID)
		if err != nil {
			t.Fatalf("查询台账订单详情失败: %v", err)
		}
		requireFeeLedgerProjection(t, "草稿对冲详情", findFeeLedgerItem(detail.Items, draftFeeID), draftBill.BillNo, biz.FeeLedgerUnverifiedUninvoiced, false)
		requireFeeLedgerProjection(t, "已反转对冲详情", findFeeLedgerItem(detail.Items, reversedFeeID), reversedBill.BillNo, biz.FeeLedgerUnverifiedUninvoiced, false)
	})

	t.Run("核销与对冲混合计入部分结清", func(t *testing.T) {
		fixture, repo, ctx, organizationIDs := newScopedFixture(t)
		feeID := fixture.createLedgerFee("mixed", "200")
		bill := fixture.createConfirmedBill("mixed", "200", financeBillIntegrationDate)
		fixture.linkBillFee("mixed", bill, feeID, "200")
		fixture.createNetting("mixed", financenettingent.StatusCONFIRMED, bill, "80", true)
		cashflow := fixture.createConfirmedCashflow("mixed", "50", financeBillIntegrationDate)
		fixture.createActiveVerification("mixed", "50", func(verificationID uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			return []*ent.FinanceVerificationAllocationCreate{fixture.verificationAllocationCreate(verificationID, cashflow, bill, "50")}
		})

		if settled := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerVerifiedUninvoiced); settled.Total != 0 {
			t.Fatalf("混合 130/200 不应命中已结清：total=%d", settled.Total)
		}
		if unverified := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerUnverifiedUninvoiced); unverified.Total != 0 {
			t.Fatalf("混合 130/200 不应命中未结清：total=%d", unverified.Total)
		}
		partial := listFeeLedgerByProgress(t, repo, organizationIDs, biz.FeeLedgerPartiallyVerifiedUninvoiced)
		if partial.Total != 1 || len(partial.Items) != 1 {
			t.Fatalf("核销+对冲混合应命中部分结清：total=%d items=%d", partial.Total, len(partial.Items))
		}
		requireFeeLedgerProjection(t, "混合列表", partial.Items[0], bill.BillNo, biz.FeeLedgerPartiallyVerifiedUninvoiced, false)
		detail, err := repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, fixture.orderID)
		if err != nil {
			t.Fatalf("查询台账订单详情失败: %v", err)
		}
		requireFeeLedgerProjection(t, "混合详情", findFeeLedgerItem(detail.Items, feeID), bill.BillNo, biz.FeeLedgerPartiallyVerifiedUninvoiced, false)
	})

	t.Run("确认提成线使列表与详情同时财务锁定", func(t *testing.T) {
		fixture, repo, ctx, organizationIDs := newScopedFixture(t)
		feeID := fixture.createLedgerFee("lock", "100")
		standalone := fixture.createActiveVerification("lock", "10", nil)
		fixture.createConfirmedCommission("lock", standalone)

		result, err := repo.ListFeeLedger(ctx, organizationIDs, biz.FeeLedgerFilter{Page: 1, PageSize: 200})
		if err != nil {
			t.Fatalf("查询费用台账失败: %v", err)
		}
		detail, err := repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, fixture.orderID)
		if err != nil {
			t.Fatalf("查询台账订单详情失败: %v", err)
		}
		if findFeeLedgerItem(result.Items, feeID) == nil || findFeeLedgerItem(detail.Items, feeID) == nil {
			t.Fatal("台账列表或详情没有返回费用")
		}
		for _, item := range result.Items {
			requireFeeLedgerProjection(t, "锁定列表", item, item.BillNo, item.FinancialProgress, true)
		}
		for _, item := range detail.Items {
			requireFeeLedgerProjection(t, "锁定详情", item, item.BillNo, item.FinancialProgress, true)
		}
	})
}

func TestVerificationCreationCandidatesBeyondSettledLimitPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	// 夹具在子测试内创建，保证其 t.Cleanup 在外层 defer 关闭数据库之前执行。
	t.Run("超过200条已结清记录后未结清候选仍可见", func(t *testing.T) {
		fixture := newFeeLedgerPostgresFixture(t, data)
		repo := NewVerificationRepo(data)
		ctx := context.Background()

		const settledCount = 201
		settledDate := "2026-09-01"
		openDate := "2026-08-01"
		settledAmount := "10"

		// 201 张已确认且全额结清的账单，加上 1 张日期更早（排序在已结清之后）的未结清账单。
		settledBills := make([]*ent.FinanceBill, 0, settledCount)
		billBulk := make([]*ent.FinanceBillCreate, 0, settledCount)
		for i := 0; i < settledCount; i++ {
			create := data.db.FinanceBill.Create().
				SetOrganizationID(fixture.organizationID).
				SetBillNo(fmt.Sprintf("BILL-CAND-SETTLED-%03d-%s", i, fixture.suffix)).
				SetIdempotencyKey(fmt.Sprintf("bill-cand-settled-%03d-%s", i, fixture.suffix)).
				SetDirection(financebillent.DirectionRECEIVABLE).
				SetStatus(financebillent.StatusCONFIRMED).
				SetSettlementPartyID(fixture.partnerID).
				SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).
				SetCurrency("CNY").
				SetBaseCurrency("CNY").
				SetExchangeRate("1.00000000").
				SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
				SetExchangeRateDate(financeBillIntegrationDate).
				SetTotalAmount("10.00000000").
				SetNetAmount("10.00000000").
				SetTaxAmount("0.00000000").
				SetBaseCurrencyAmount("10.00000000").
				SetFeeCount(1).
				SetBillDate(settledDate).
				SetVersion(1)
			billBulk = append(billBulk, withTestFinanceBillSettlementAccountSnapshot(create, fixture.accountID, "CNY"))
		}
		settledBills, err := data.db.FinanceBill.CreateBulk(billBulk...).Save(ctx)
		if err != nil {
			t.Fatalf("创建已结清候选账单失败: %v", err)
		}
		openBill := fixture.createConfirmedBill("cand-open", settledAmount, openDate)

		// 单张支撑流水的 ACTIVE 核销单把 201 张账单全部结清。
		billSupportCashflow := fixture.createConfirmedCashflow("cand-bill-support", "2010", settledDate)
		fixture.createActiveVerification("cand-bills", "2010", func(verificationID uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			allocations := make([]*ent.FinanceVerificationAllocationCreate, 0, settledCount)
			for _, bill := range settledBills {
				allocations = append(allocations, fixture.verificationAllocationCreate(verificationID, billSupportCashflow, bill, settledAmount))
			}
			return allocations
		})

		// 201 条已确认且全额核销的流水，加上 1 条日期更早的未核销流水。
		settledCashflows := make([]*ent.FinanceCashflow, 0, settledCount)
		cashflowBulk := make([]*ent.FinanceCashflowCreate, 0, settledCount)
		for i := 0; i < settledCount; i++ {
			cashflowBulk = append(cashflowBulk, data.db.FinanceCashflow.Create().
				SetOrganizationID(fixture.organizationID).
				SetFlowNo(fmt.Sprintf("FLOW-CAND-SETTLED-%03d-%s", i, fixture.suffix)).
				SetIdempotencyKey(fmt.Sprintf("cashflow-cand-settled-%03d-%s", i, fixture.suffix)).
				SetDirection(financecashflowent.DirectionRECEIVABLE).
				SetStatus(financecashflowent.StatusCONFIRMED).
				SetSettlementPartyID(fixture.partnerID).
				SetSettlementPartyName("账单事务测试客户-"+fixture.suffix).
				SetCurrency("CNY").
				SetAmount("10.00000000").
				SetExchangeRate("1.00000000").
				SetExchangeRateSource(financecashflowent.ExchangeRateSourceBASE_CURRENCY).
				SetExchangeRateDate(financeBillIntegrationDate).
				SetBaseCurrency("CNY").
				SetBaseAmount("10.00000000").
				SetTransactionDate(settledDate).
				SetOurAccount("测试账户").
				SetPaymentMethod("BANK_TRANSFER").
				SetVersion(1))
		}
		settledCashflows, err = data.db.FinanceCashflow.CreateBulk(cashflowBulk...).Save(ctx)
		if err != nil {
			t.Fatalf("创建已核销候选流水失败: %v", err)
		}
		cashflowSupportBill := fixture.createConfirmedBill("cand-cash-support", "2010", settledDate)
		fixture.createActiveVerification("cand-cashflows", "2010", func(verificationID uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			allocations := make([]*ent.FinanceVerificationAllocationCreate, 0, settledCount)
			for _, cashflow := range settledCashflows {
				allocations = append(allocations, fixture.verificationAllocationCreate(verificationID, cashflow, cashflowSupportBill, settledAmount))
			}
			return allocations
		})
		openCashflow := fixture.createConfirmedCashflow("cand-open", settledAmount, openDate)

		candidates, err := repo.ListCreationCandidates(ctx, fixture.organizationID, biz.VerificationCreationCandidateFilter{
			Direction:         biz.OrderFeeReceivable,
			SettlementPartyID: fixture.partnerID,
			Currency:          "CNY",
		})
		if err != nil {
			t.Fatalf("查询核销创建候选失败: %v", err)
		}
		if len(candidates.Bills) != 1 || candidates.Bills[0].ID != openBill.ID {
			t.Fatalf("超过 200 条已结清账单后未结清候选应可见：bills=%d", len(candidates.Bills))
		}
		if !candidates.Bills[0].UnverifiedAmount.Equal(decimal.RequireFromString("10")) {
			t.Fatalf("未结清账单候选余额 = %s，期望 10", candidates.Bills[0].UnverifiedAmount)
		}
		if len(candidates.Cashflows) != 1 || candidates.Cashflows[0].ID != openCashflow.ID {
			t.Fatalf("超过 200 条已核销流水后未核销候选应可见：cashflows=%d", len(candidates.Cashflows))
		}
		if !candidates.Cashflows[0].UnverifiedAmount.Equal(decimal.RequireFromString("10")) {
			t.Fatalf("未核销流水候选余额 = %s，期望 10", candidates.Cashflows[0].UnverifiedAmount)
		}
	})
}
