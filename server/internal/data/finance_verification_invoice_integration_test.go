package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financeinvoiceexemptionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
)

func removeTestBillInvoice(t *testing.T, data *Data, billID uuid.UUID) {
	t.Helper()
	if _, err := data.db.FinanceInvoiceBill.Update().Where(financeinvoicebillent.BillIDEQ(billID)).SetActive(false).Save(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestVerificationInvoicePrerequisitePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	for _, scenario := range []string{"完全无票拒绝", "部分开票可以核销超过已开票额", "票齐可核销", "免票可核销", "撤销发票后旧候选拒绝", "撤销免票后旧候选拒绝", "多账单混入无票整次拒绝", "成功后撤票同键仍回放"} {
		t.Run(scenario, func(t *testing.T) {
			f := newVerificationPostgresFixture(t, data)
			ctx := context.Background()
			uc := f.newUsecase(NewVerificationRepo(data), NewExchangeRateRepo(data))
			invoiceUC := biz.NewFinanceInvoiceUsecase(NewFinanceInvoiceRepo(data), nil, nil, data)
			bill := data.db.FinanceBill.GetX(ctx, f.billID)
			filter := biz.VerificationCreationCandidateFilter{Page: 1, PageSize: 200, Direction: biz.OrderFeeReceivable, SettlementPartyID: f.partnerID, Currency: "USD"}
			if scenario == "完全无票拒绝" || scenario == "部分开票可以核销超过已开票额" || scenario == "免票可核销" || scenario == "撤销免票后旧候选拒绝" {
				removeTestBillInvoice(t, data, bill.ID)
			}
			if scenario == "部分开票可以核销超过已开票额" {
				addTestBillInvoice(t, data, bill, "10", f.actorID)
			}
			if scenario == "免票可核销" || scenario == "撤销免票后旧候选拒绝" {
				if err := invoiceUC.SetBillInvoiceExemption(ctx, []uuid.UUID{f.organizationID}, f.actorID, bill.ID, 1, "明确免票前置", false); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if _, err := data.db.FinanceInvoiceExemption.Delete().Where(financeinvoiceexemptionent.BillIDEQ(bill.ID)).Exec(ctx); err != nil {
						t.Error(err)
					}
				})
			}
			before, err := uc.ListBillCandidates(ctx, f.organizationID, filter)
			if err != nil {
				t.Fatal(err)
			}
			wantCandidates := int64(1)
			if scenario == "完全无票拒绝" {
				wantCandidates = 0
			}
			if before.Total != wantCandidates {
				t.Fatalf("候选=%d，期望%d", before.Total, wantCandidates)
			}
			if scenario == "撤销发票后旧候选拒绝" {
				removeTestBillInvoice(t, data, bill.ID)
			}
			if scenario == "撤销免票后旧候选拒绝" {
				if err := invoiceUC.SetBillInvoiceExemption(ctx, []uuid.UUID{f.organizationID}, f.actorID, bill.ID, 2, "重新要求发票", true); err != nil {
					t.Fatal(err)
				}
			}
			input := f.input("invoice-prerequisite")
			if scenario == "多账单混入无票整次拒绝" {
				ledger := &feeLedgerPostgresFixture{financeBillPostgresFixture: f.financeBillPostgresFixture}
				other := ledger.createConfirmedBill("no-invoice", "100", financeBillIntegrationDate)
				if err := other.Update().SetCurrency("USD").SetExchangeRate("7.2").SetBaseCurrencyAmount("720").SetSettlementAccountID(f.usdAccountID).SetSettlementAccountCurrency("USD").Exec(ctx); err != nil {
					t.Fatal(err)
				}
				input.Allocations = append(input.Allocations, &biz.VerificationAllocation{CashflowID: f.cashflowID, BillID: other.ID, Amount: decimal.NewFromInt(10)})
			}
			created, err := uc.Create(ctx, f.organizationID, f.actorID, input)
			blocked := scenario == "完全无票拒绝" || scenario == "撤销发票后旧候选拒绝" || scenario == "撤销免票后旧候选拒绝" || scenario == "多账单混入无票整次拒绝"
			if blocked {
				if !errors.Is(err, biz.ErrVerificationInvoiceRequired) {
					t.Fatalf("无票错误=%v", err)
				}
				f.requireRolledBackState()
				if scenario != "多账单混入无票整次拒绝" {
					fresh, err := uc.ListBillCandidates(ctx, f.organizationID, filter)
					if err != nil || fresh.Total != 0 {
						t.Fatalf("失效候选仍可用: %+v %v", fresh, err)
					}
				}
				return
			}
			if err != nil || created == nil {
				t.Fatalf("合法核销失败:%v", err)
			}
			f.requireCommittedState(1)
			if scenario == "成功后撤票同键仍回放" {
				removeTestBillInvoice(t, data, bill.ID)
				replayed, err := uc.Create(ctx, f.organizationID, f.actorID, input)
				if err != nil || replayed.ID != created.ID {
					t.Fatalf("既有核销重试未回放:%v", err)
				}
			}
			// 撤销核销不受新的新增准入影响，保持反向业务可办理。
			if _, err := uc.Reverse(ctx, f.organizationID, f.actorID, created.ID, created.Version, "核对后反核销"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
