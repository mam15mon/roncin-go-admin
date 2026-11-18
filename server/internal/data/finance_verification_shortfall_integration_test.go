package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	ver "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	"github.com/shopspring/decimal"
)

func TestVerificationShortfallAtomicPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	for _, scenario := range []string{"成功及重试", "核销失败回滚流水", "未开票核销回滚差额", "差额与分配不符", "相同键修改流水内容", "相同键修改分配", "并发余额不足不增大差额", "相同幂等键并发"} {
		t.Run(scenario, func(t *testing.T) {
			fixture := newVerificationPostgresFixture(t, data)
			settingID := fixture.createExchangeRateSetting("7.20000000")
			ctx := context.Background()
			_, err := data.db.NumberRule.Create().SetOrganizationID(fixture.organizationID).SetDocumentType(numberrule.DocumentTypeReceiptPayment).SetPrefix("CF-").SetDateFormat(numberrule.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberrule.ResetPolicyNever).SetEnabled(true).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			exchange := biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), nil)
			cashflowUsecase := biz.NewFinanceCashflowUsecase(NewFinanceCashflowRepo(data), exchange)
			usecase := fixture.newUsecase(NewVerificationRepo(data), NewExchangeRateRepo(data))
			in := fixture.input("shortfall")
			in.Allocations = []*biz.VerificationAllocation{
				{CashflowID: fixture.cashflowID, BillID: fixture.billID, Amount: decimal.NewFromInt(60)},
				{BillID: fixture.billID, Amount: decimal.NewFromInt(40)},
			}
			shortfall := biz.CreateFinanceCashflowInput{Direction: biz.OrderFeeReceivable, SettlementPartyID: fixture.partnerID, Currency: "USD", Amount: decimal.NewFromInt(40), TransactionDate: financeBillIntegrationDate, OurAccount: "差额账户", PaymentMethod: "BANK_TRANSFER"}
			expectFailure := false
			switch scenario {
			case "未开票核销回滚差额":
				removeTestBillInvoice(t, data, fixture.billID)
				expectFailure = true
			case "核销失败回滚流水":
				usecase = fixture.newUsecase(&invalidAuditResultVerificationRepo{VerificationRepo: NewVerificationRepo(data)}, NewExchangeRateRepo(data))
				expectFailure = true
			case "差额与分配不符":
				shortfall.Amount = decimal.NewFromInt(50)
				expectFailure = true

			}
			if scenario == "并发余额不足不增大差额" || scenario == "相同幂等键并发" {
				results := make(chan verificationCreateResult, 2)
				start := make(chan struct{})
				for _, suffix := range []string{"a", "b"} {
					request := in
					if scenario != "相同幂等键并发" {
						request.IdempotencyKey += suffix
					}
					go func() {
						<-start
						created, err := usecase.CreateWithCashflow(ctx, fixture.organizationID, fixture.actorID, request, shortfall, cashflowUsecase, false)
						results <- verificationCreateResult{verification: created, err: err}
					}()
				}
				close(start)
				successes := 0
				ids := map[uuid.UUID]bool{}
				for range 2 {
					result := <-results
					if result.err == nil {
						successes++
						ids[result.verification.ID] = true
					}
				}
				want := 1
				if scenario == "相同幂等键并发" {
					want = 2
				}
				if successes != want || len(ids) != 1 {
					t.Fatalf("并发组合成功数=%d，期望%d，核销数=%d", successes, want, len(ids))
				}
				cashCount, err := data.db.FinanceCashflow.Query().Where(cash.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
				if err != nil || cashCount != 2 {
					t.Fatalf("失败组合留下差额流水: count=%d err=%v", cashCount, err)
				}
				return
			}
			created, err := usecase.CreateWithCashflow(ctx, fixture.organizationID, fixture.actorID, in, shortfall, cashflowUsecase, false)
			if expectFailure {
				if err == nil {
					t.Fatal("非法组合意外成功")
				}
				if scenario == "未开票核销回滚差额" && !errors.Is(err, biz.ErrVerificationInvoiceRequired) {
					t.Fatalf("无票差额核销错误=%v", err)
				}
				cashCount, err := data.db.FinanceCashflow.Query().Where(cash.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
				if err != nil || cashCount != 1 {
					t.Fatalf("失败组合留下差额流水: count=%d err=%v", cashCount, err)
				}
				fixture.requireRolledBackState()
				sequences, err := data.db.NumberSequence.Query().Where(numbersequence.HasRuleWith(numberrule.OrganizationIDEQ(fixture.organizationID), numberrule.DocumentTypeEQ(numberrule.DocumentTypeReceiptPayment))).Count(ctx)
				if err != nil || sequences != 0 {
					t.Fatalf("失败组合留下流水编号序列: count=%d err=%v", sequences, err)
				}
				audits, err := data.db.AuditLog.Query().Where(auditlog.OrganizationIDEQ(fixture.organizationID), auditlog.ActionIn("finance.cashflow.create", "finance.cashflow.confirm")).Count(ctx)
				if err != nil || audits != 0 {
					t.Fatalf("失败组合留下流水审计: count=%d err=%v", audits, err)
				}
				return
			}
			if err != nil || created == nil {
				t.Fatalf("组合创建失败: %v", err)
			}
			cashflows, err := data.db.FinanceCashflow.Query().Where(cash.OrganizationIDEQ(fixture.organizationID)).All(ctx)
			if err != nil || len(cashflows) != 2 {
				t.Fatalf("差额流水数量不符: %d %v", len(cashflows), err)
			}
			for _, item := range cashflows {
				if item.ID != fixture.cashflowID && (item.Status != cash.StatusCONFIRMED || item.Amount != "40.00000000") {
					t.Fatalf("差额流水未确认或金额错误: %s %s", item.Status, item.Amount)
				}
			}
			if len(created.Allocations) != 2 || !created.Amount.Equal(decimal.NewFromInt(100)) {
				t.Fatal("核销分配不完整")
			}
			if in.Allocations[1].CashflowID != uuid.Nil {
				t.Fatal("组合编排修改了调用者原始输入")
			}
			if scenario == "成功及重试" {
				if _, err := data.db.ExchangeRateSetting.UpdateOneID(settingID).SetIsActive(false).Save(ctx); err != nil {
					t.Fatal(err)
				}
			}
			switch scenario {
			case "相同键修改流水内容":
				shortfall.OurAccount = "另一个账户"
			case "相同键修改分配":
				in.Allocations[0].Amount = decimal.NewFromInt(59)
			}
			replay, err := usecase.CreateWithCashflow(ctx, fixture.organizationID, fixture.actorID, in, shortfall, cashflowUsecase, false)
			if scenario == "成功及重试" {
				if err != nil || replay.ID != created.ID {
					t.Fatalf("重试未回放: %v", err)
				}
			} else if err == nil {
				t.Fatal("同键不同内容未冲突")
			}
			count, err := data.db.FinanceVerification.Query().Where(ver.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
			if err != nil || count != 1 {
				t.Fatalf("幂等核销数量=%d err=%v", count, err)
			}
		})
	}
}
