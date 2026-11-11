package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	netalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	verifyalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/shopspring/decimal"
)

func nettingPreviewTestBill(f *financeBillPostgresFixture, label string, direction billent.Direction, currency string, partyID uuid.UUID) *ent.FinanceBillCreate {
	return withTestFinanceBillSettlementAccountSnapshot(f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo("PREVIEW-"+label+"-"+f.suffix).
		SetIdempotencyKey("preview-"+label+"-"+f.suffix).
		SetDirection(direction).
		SetStatus(billent.StatusCONFIRMED).
		SetSettlementPartyID(partyID).
		SetSettlementPartyName("预览候选测试单位").
		SetCurrency(currency).
		SetBaseCurrency("CNY").
		SetExchangeRate("1").
		SetExchangeRateSource(billent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").
		SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1), f.accountID, currency)
}

func TestFinanceNettingPreviewUnsettledCandidatesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	t.Run("五百张核销对冲及混合结清历史不占候选且撤销后重新入选", func(t *testing.T) {
		f := newNettingPostgresFixture(t, data)
		base := f.financeBillPostgresFixture
		builders := make([]*ent.FinanceBillCreate, 0, biz.MaxFinanceNettingBills)
		for i := 0; i < biz.MaxFinanceNettingBills; i++ {
			builders = append(builders, nettingPreviewTestBill(base, fmt.Sprintf("history-%d", i), billent.DirectionRECEIVABLE, "CNY", f.partnerID))
		}
		history, err := data.db.FinanceBill.CreateBulk(builders...).Save(ctx)
		if err != nil {
			t.Fatalf("创建已清历史账单: %v", err)
		}
		// 预览只消费有效分摊，不依赖父单与账单的冗余结清状态。最小账务夹具
		// 分别覆盖全额核销、全额对冲，以及两种有效分摊之和恰好结清。
		cashflow := data.db.FinanceCashflow.Create().SetOrganizationID(f.organizationID).
			SetFlowNo("PV-FLOW-" + f.suffix).SetIdempotencyKey("pv-flow-" + f.suffix).
			SetDirection(cashflowent.DirectionRECEIVABLE).SetStatus(cashflowent.StatusCONFIRMED).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("预览候选测试单位").
			SetCurrency("CNY").SetAmount("50000").SetBaseCurrency("CNY").SetBaseAmount("50000").
			SetExchangeRate("1").SetExchangeRateSource(cashflowent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate(financeBillIntegrationDate).SetTransactionDate(financeBillIntegrationDate).
			SetOurAccount("测试账户").SetPaymentMethod("BANK_TRANSFER").SaveX(ctx)
		verification := data.db.FinanceVerification.Create().SetOrganizationID(f.organizationID).
			SetVerificationNo("PV-VERIFY-" + f.suffix).SetIdempotencyKey("pv-verify-" + f.suffix).
			SetDirection(verificationent.DirectionRECEIVABLE).SetStatus(verificationent.StatusACTIVE).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("预览候选测试单位").
			SetCurrency("CNY").SetAmount("50000").SetBaseCurrency("CNY").SetBaseAmount("50000").
			SetBillBaseAmount("50000").SetCashflowBaseAmount("50000").SetExchangeGainLoss("0").
			SetVerificationDate(financeBillIntegrationDate).SaveX(ctx)
		netting := data.db.FinanceNetting.Create().SetOrganizationID(f.organizationID).
			SetNettingNo("PV-NET-" + f.suffix).SetIdempotencyKey("pv-net-" + f.suffix).
			SetRequestHash(strings.Repeat("a", 64)).SetStatus(nettingent.StatusCONFIRMED).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("预览候选测试单位").
			SetCurrency("CNY").SetAmount("50000").SetBaseCurrency("CNY").
			SetBaseCurrencyAmount("50000").SetPayableBaseAmount("50000").SetExchangeGainLoss("0").SaveX(ctx)
		t.Cleanup(func() {
			data.db.FinanceVerificationAllocation.Delete().Where(verifyalloc.VerificationIDEQ(verification.ID)).ExecX(ctx)
			data.db.FinanceVerification.DeleteOne(verification).ExecX(ctx)
			data.db.FinanceCashflow.DeleteOne(cashflow).ExecX(ctx)
		})
		verificationRows := make([]*ent.FinanceVerificationAllocationCreate, 0, len(history))
		nettingRows := make([]*ent.FinanceNettingAllocationCreate, 0, len(history))
		for i, bill := range history {
			verifyAmount, netAmount := "100", "100"
			if i%3 == 2 {
				verifyAmount, netAmount = "40", "60"
			}
			if i%3 != 1 {
				verificationRows = append(verificationRows, data.db.FinanceVerificationAllocation.Create().
					SetVerificationID(verification.ID).SetCashflowID(cashflow.ID).SetBillID(bill.ID).
					SetCashflowNo(cashflow.FlowNo).SetBillNo(bill.BillNo).SetAmount(verifyAmount).
					SetBillBaseAmount(verifyAmount).SetCashflowBaseAmount(verifyAmount).SetExchangeGainLoss("0").SetActive(true))
			}
			if i%3 != 0 {
				nettingRows = append(nettingRows, data.db.FinanceNettingAllocation.Create().
					SetNettingID(netting.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).
					SetDirection(netalloc.DirectionRECEIVABLE).SetAmount(netAmount).SetBaseCurrencyAmount(netAmount).SetActive(true))
			}
		}
		data.db.FinanceVerificationAllocation.CreateBulk(verificationRows...).ExecX(ctx)
		data.db.FinanceNettingAllocation.CreateBulk(nettingRows...).ExecX(ctx)

		preview, err := f.newUsecase().Preview(ctx, f.organizationID, f.partnerID, "CNY")
		if err != nil {
			t.Fatalf("五百张已清历史不应阻断两张未清账单预览: %v", err)
		}
		if len(preview.ReceivableBills) != 1 || len(preview.PayableBills) != 1 ||
			preview.ReceivableBills[0].BillID != f.receivableBillID || preview.PayableBills[0].BillID != f.payableBillID ||
			!preview.ReceivableAvailableAmount.Equal(decimal.NewFromInt(100)) || !preview.PayableAvailableAmount.Equal(decimal.NewFromInt(60)) || !preview.OffsetAmount.Equal(decimal.NewFromInt(60)) {
			t.Fatalf("未清候选及汇总不符: %+v", preview)
		}

		for _, tc := range []struct {
			name   string
			billID uuid.UUID
			amount int64
			undo   func()
		}{
			{name: "撤销核销", billID: history[0].ID, amount: 100, undo: func() {
				data.db.FinanceVerificationAllocation.Update().Where(verifyalloc.BillIDEQ(history[0].ID)).SetActive(false).ExecX(ctx)
			}},
			{name: "反转对冲", billID: history[1].ID, amount: 100, undo: func() {
				data.db.FinanceNettingAllocation.Update().Where(netalloc.BillIDEQ(history[1].ID)).SetActive(false).ExecX(ctx)
			}},
			{name: "混合结清中撤销核销", billID: history[2].ID, amount: 40, undo: func() {
				data.db.FinanceVerificationAllocation.Update().Where(verifyalloc.BillIDEQ(history[2].ID)).SetActive(false).ExecX(ctx)
			}},
			{name: "混合结清继续反转对冲", billID: history[2].ID, amount: 100, undo: func() {
				data.db.FinanceNettingAllocation.Update().Where(netalloc.BillIDEQ(history[2].ID)).SetActive(false).ExecX(ctx)
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				tc.undo()
				preview, err := f.newUsecase().Preview(ctx, f.organizationID, f.partnerID, "CNY")
				if err != nil {
					t.Fatal(err)
				}
				for _, bill := range preview.ReceivableBills {
					if bill.BillID == tc.billID {
						if !bill.AvailableAmount.Equal(decimal.NewFromInt(tc.amount)) {
							t.Fatalf("撤销后候选余额=%s，期望=%d", bill.AvailableAmount, tc.amount)
						}
						return
					}
				}
				t.Fatalf("撤销后账单未重新进入候选: %s", tc.billID)
			})
		}
	})

	t.Run("真正未清账单五百张成功五百零一张拒绝", func(t *testing.T) {
		f := newFinanceBillPostgresFixture(t, data)
		builders := make([]*ent.FinanceBillCreate, 0, biz.MaxFinanceNettingBills)
		for i := 0; i < biz.MaxFinanceNettingBills; i++ {
			builders = append(builders, nettingPreviewTestBill(f, fmt.Sprintf("open-%d", i), billent.DirectionRECEIVABLE, "CNY", f.partnerID))
		}
		data.db.FinanceBill.CreateBulk(builders...).ExecX(ctx)
		repo := NewFinanceNettingRepo(data)
		preview, err := repo.LoadPreview(ctx, f.organizationID, f.partnerID, "CNY")
		if err != nil || len(preview.ReceivableBills) != biz.MaxFinanceNettingBills {
			t.Fatalf("五百张真正候选应成功: preview=%v err=%v", preview, err)
		}
		nettingPreviewTestBill(f, "open-over-limit", billent.DirectionRECEIVABLE, "CNY", f.partnerID).ExecX(ctx)
		if _, err := repo.LoadPreview(ctx, f.organizationID, f.partnerID, "CNY"); !errors.Is(err, biz.ErrFinanceNettingTooManyBills) {
			t.Fatalf("五百零一张真正候选应超限: %v", err)
		}
	})

	t.Run("其他公司单位币种的未清账单不占当前候选上限", func(t *testing.T) {
		f := newNettingPostgresFixture(t, data)
		other := newFinanceBillPostgresFixture(t, data)
		party := data.db.Partner.Create().SetOrganizationID(f.organizationID).SetCode("OTHER-" + f.suffix).
			SetLegalName("其他结算单位").SetNormalizedName("其他结算单位").SaveX(ctx)
		builders := make([]*ent.FinanceBillCreate, 0, 3*(biz.MaxFinanceNettingBills+1))
		for i := 0; i <= biz.MaxFinanceNettingBills; i++ {
			label := fmt.Sprintf("scope-%d", i)
			builders = append(builders,
				nettingPreviewTestBill(other, label, billent.DirectionRECEIVABLE, "CNY", other.partnerID),
				nettingPreviewTestBill(f.financeBillPostgresFixture, label+"-party", billent.DirectionRECEIVABLE, "CNY", party.ID),
				nettingPreviewTestBill(f.financeBillPostgresFixture, label+"-usd", billent.DirectionRECEIVABLE, "USD", f.partnerID))
		}
		data.db.FinanceBill.CreateBulk(builders...).ExecX(ctx)
		preview, err := f.newUsecase().Preview(ctx, f.organizationID, f.partnerID, "CNY")
		if err != nil || len(preview.ReceivableBills) != 1 || len(preview.PayableBills) != 1 {
			t.Fatalf("其他范围不应进入当前候选: preview=%v err=%v", preview, err)
		}
	})
}
