package data

import (
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	cashflow "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// supplementCommissionSourceFixture 用真实账单行和核销替代原无来源提成夹具。
func (f *feeSupplementPostgresFixture) supplementCommissionSourceFixture(order *ent.Order) (uuid.UUID, string) {
	f.t.Helper()
	lines, err := f.data.db.FinanceBillLine.Query().Where(billline.OrderIDEQ(order.ID), billline.HasBillWith(bill.DirectionEQ(bill.DirectionRECEIVABLE))).All(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	if len(lines) > 0 {
		v, err := f.data.db.FinanceVerification.Query().Where(verification.HasAllocationsWith(allocation.BillIDEQ(lines[0].BillID))).Only(f.ctx)
		if err != nil && !ent.IsNotFound(err) {
			f.t.Fatal(err)
		}
		if err == nil {
			count, err := f.data.db.FinanceCommission.Query().Where(financecommission.VerificationIDEQ(v.ID)).Count(f.ctx)
			if err != nil {
				f.t.Fatal(err)
			}
			role := "SALES"
			if count > 0 {
				role = "OPERATOR"
			}
			return v.ID, role
		}
	}
	original, err := f.data.db.OrderFee.Query().Where(fee.OrderIDEQ(order.ID), fee.DirectionEQ(fee.DirectionRECEIVABLE), fee.EconomicEffectEQ(fee.EconomicEffectINCREASE)).First(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	no := uuid.NewString()
	var b *ent.FinanceBill
	if len(lines) > 0 {
		// 结案夹具已有正常账单但尚未收款，为冻结提成建立真实收款来源。
		b, err = f.data.db.FinanceBill.Get(f.ctx, lines[0].BillID)
	} else {
		b, err = withTestFinanceBillSettlementAccountSnapshot(f.data.db.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo("B-"+no).SetIdempotencyKey(no).SetDirection(bill.DirectionRECEIVABLE).SetStatus(bill.StatusCONFIRMED).SetSettlementPartyID(f.customerID).SetSettlementPartyName("客户").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(bill.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-01").SetTotalAmount("1000").SetNetAmount("1000").SetTaxAmount("0").SetBaseCurrencyAmount("1000").SetFeeCount(1).SetBillDate("2026-09-01"), uuid.New(), "CNY").Save(f.ctx)
		if err != nil {
			f.t.Fatal(err)
		}
		if _, err = f.data.db.FinanceBillLine.Create().SetBillID(b.ID).SetOrderID(order.ID).SetOrderFeeID(original.ID).SetOrderNo(order.OrderNo).SetFeeCode("BASE").SetFeeName("原应收").SetQuantity("1").SetUnitPrice("1000").SetTotalAmount("1000").SetNetAmount("1000").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrencyAmount("1000").SetBaseCurrency("CNY").SetActive(true).Save(f.ctx); err != nil {
			f.t.Fatal(err)
		}
	}
	if err != nil {
		f.t.Fatal(err)
	}
	c, err := f.data.db.FinanceCashflow.Create().SetOrganizationID(f.organizationID).SetFlowNo("C-" + no).SetIdempotencyKey(no).SetDirection(cashflow.DirectionRECEIVABLE).SetStatus(cashflow.StatusCONFIRMED).SetSettlementPartyID(f.customerID).SetSettlementPartyName("客户").SetCurrency("CNY").SetAmount("1000").SetExchangeRate("1").SetExchangeRateSource(cashflow.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-01").SetBaseCurrency("CNY").SetBaseAmount("1000").SetTransactionDate("2026-09-01").SetOurAccount("测试").SetPaymentMethod("BANK_TRANSFER").Save(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	v, err := f.data.db.FinanceVerification.Create().SetOrganizationID(f.organizationID).SetVerificationNo("V-" + no).SetIdempotencyKey(no).SetDirection(verification.DirectionRECEIVABLE).SetStatus(verification.StatusACTIVE).SetSettlementPartyID(f.customerID).SetSettlementPartyName("客户").SetCurrency("CNY").SetAmount("1000").SetBaseCurrency("CNY").SetBaseAmount("1000").SetBillBaseAmount("1000").SetCashflowBaseAmount("1000").SetExchangeGainLoss("0").SetVerificationDate("2026-09-01").Save(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().SetVerificationID(v.ID).SetCashflowID(c.ID).SetBillID(b.ID).SetCashflowNo(c.FlowNo).SetBillNo(b.BillNo).SetAmount("1000").SetBillBaseAmount("1000").SetCashflowBaseAmount("1000").SetExchangeGainLoss("0").SetActive(true).Save(f.ctx); err != nil {
		f.t.Fatal(err)
	}
	return v.ID, "SALES"
}
