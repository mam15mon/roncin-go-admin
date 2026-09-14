package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	"github.com/shopspring/decimal"
)

type verificationPostgresFixture struct {
	*financeBillPostgresFixture
	billID     uuid.UUID
	cashflowID uuid.UUID
}

type verificationCreateResult struct {
	verification *biz.FinanceVerification
	err          error
}

type invalidAuditResultVerificationRepo struct {
	biz.VerificationRepo
}

func (r *invalidAuditResultVerificationRepo) Create(ctx context.Context, organizationID, actorID uuid.UUID, verification *biz.FinanceVerification, audit *biz.AuditEvent) (*biz.FinanceVerification, error) {
	audit.Result = "invalid"
	return r.VerificationRepo.Create(ctx, organizationID, actorID, verification, audit)
}

func TestVerificationCreateSharedTransactionPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("相同幂等键并发创建返回同一核销", func(t *testing.T) {
		fixture := newVerificationPostgresFixture(t, data)
		usecase := fixture.newUsecase(NewVerificationRepo(fixture.data), NewExchangeRateRepo(fixture.data))
		inputs := []biz.CreateVerificationInput{fixture.input("same-key"), fixture.input("same-key")}

		results := createVerificationsConcurrently(usecase, fixture.organizationID, fixture.actorID, inputs...)
		for index, result := range results {
			if result.err != nil || result.verification == nil {
				t.Fatalf("第 %d 个并发请求创建核销: verification=%#v error=%v", index+1, result.verification, result.err)
			}
		}
		if results[0].verification.ID != results[1].verification.ID || results[0].verification.VerificationNo != results[1].verification.VerificationNo {
			t.Fatalf("相同幂等键返回了不同核销: first=%s/%s second=%s/%s", results[0].verification.ID, results[0].verification.VerificationNo, results[1].verification.ID, results[1].verification.VerificationNo)
		}
		fixture.requireCommittedState(1)
	})

	t.Run("审计失败回滚核销分摊和单号序列", func(t *testing.T) {
		fixture := newVerificationPostgresFixture(t, data)
		repo := &invalidAuditResultVerificationRepo{VerificationRepo: NewVerificationRepo(fixture.data)}
		usecase := fixture.newUsecase(repo, NewExchangeRateRepo(fixture.data))

		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("rollback"))
		if err == nil {
			t.Fatal("审计结果非法时创建核销未失败")
		}
		fixture.requireRolledBackState()
	})
}

func newVerificationPostgresFixture(t *testing.T, data *Data) *verificationPostgresFixture {
	t.Helper()
	base := newFinanceBillPostgresFixture(t, data)
	fixture := &verificationPostgresFixture{financeBillPostgresFixture: base}
	t.Cleanup(fixture.cleanupVerification)
	ctx := context.Background()

	if _, err := data.db.NumberRule.Create().SetOrganizationID(fixture.organizationID).SetDocumentType(numberruleent.DocumentTypeWriteOff).SetPrefix("WO-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试核销编号规则: %v", err)
	}
	billCreate := data.db.FinanceBill.Create().SetOrganizationID(fixture.organizationID).SetBillNo("BILL-V-" + fixture.suffix).SetIdempotencyKey("bill-verification-" + fixture.suffix).SetDirection(financebillent.DirectionRECEIVABLE).SetStatus(financebillent.StatusCONFIRMED).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).SetCurrency("USD").SetBaseCurrency("CNY").SetExchangeRate("7.20000000").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetTotalAmount("100.00000000").SetNetAmount("100.00000000").SetTaxAmount("0.00000000").SetBaseCurrencyAmount("720.00000000").SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1)
	bill, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, fixture.usdAccountID, "USD").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试已确认账单: %v", err)
	}
	fixture.billID = bill.ID
	cashflow, err := data.db.FinanceCashflow.Create().SetOrganizationID(fixture.organizationID).SetFlowNo("FLOW-" + fixture.suffix).SetIdempotencyKey("cashflow-verification-" + fixture.suffix).SetDirection(financecashflowent.DirectionRECEIVABLE).SetStatus(financecashflowent.StatusCONFIRMED).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).SetCurrency("USD").SetAmount("100.00000000").SetExchangeRate("7.25000000").SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetBaseCurrency("CNY").SetBaseAmount("725.00000000").SetTransactionDate(financeBillIntegrationDate).SetOurAccount("测试账户").SetPaymentMethod("BANK_TRANSFER").SetVersion(1).Save(ctx)
	if err != nil {
		t.Fatalf("创建测试已确认资金流水: %v", err)
	}
	fixture.cashflowID = cashflow.ID
	return fixture
}

func (f *verificationPostgresFixture) input(key string) biz.CreateVerificationInput {
	return biz.CreateVerificationInput{
		Allocations:      []*biz.VerificationAllocation{{CashflowID: f.cashflowID, BillID: f.billID, Amount: decimal.RequireFromString("40")}},
		VerificationDate: financeBillIntegrationDate,
		IdempotencyKey:   "verification-" + key + "-" + f.suffix,
	}
}

func (f *verificationPostgresFixture) newUsecase(repo biz.VerificationRepo, exchangeRepo biz.ExchangeRateRepo) *biz.VerificationUsecase {
	return biz.NewVerificationUsecase(repo, biz.NewExchangeRateUsecase(exchangeRepo, nil), f.data)
}

func createVerificationsConcurrently(usecase *biz.VerificationUsecase, organizationID, actorID uuid.UUID, inputs ...biz.CreateVerificationInput) []verificationCreateResult {
	start := make(chan struct{})
	results := make(chan verificationCreateResult, len(inputs))
	for _, input := range inputs {
		input := input
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			verification, err := usecase.Create(ctx, organizationID, actorID, input)
			results <- verificationCreateResult{verification: verification, err: err}
		}()
	}
	close(start)
	collected := make([]verificationCreateResult, 0, len(inputs))
	for range inputs {
		collected = append(collected, <-results)
	}
	return collected
}

func (f *verificationPostgresFixture) requireCommittedState(wantVerifications int) {
	f.t.Helper()
	ctx := context.Background()
	verifications, err := f.data.db.FinanceVerification.Query().Where(financeverificationent.OrganizationIDEQ(f.organizationID)).All(ctx)
	// B1：核销单头本位币金额严格等于行级流水本位币合计（725 × 40 / 100 = 290）。
	if err != nil || len(verifications) != wantVerifications || verifications[0].BaseAmount != "290.00000000" || verifications[0].CashflowBaseAmount != "290.00000000" {
		f.t.Fatalf("已提交核销 = %#v，期望数量 %d、单头本位币 290.00000000，error=%v", verifications, wantVerifications, err)
	}
	allocationCount, err := f.data.db.FinanceVerificationAllocation.Query().Where(financeverificationallocationent.HasVerificationWith(financeverificationent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || allocationCount != wantVerifications {
		f.t.Fatalf("已提交核销分摊数 = %d，期望 %d，error=%v", allocationCount, wantVerifications, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.verification.create")).Count(ctx)
	if err != nil || auditCount != wantVerifications {
		f.t.Fatalf("已提交核销审计数 = %d，期望 %d，error=%v", auditCount, wantVerifications, err)
	}
	sequences, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID), numberruleent.DocumentTypeEQ(numberruleent.DocumentTypeWriteOff))).All(ctx)
	if err != nil || len(sequences) != 1 || sequences[0].CurrentValue != int64(wantVerifications) {
		f.t.Fatalf("已提交核销序列 = %#v，期望当前值 %d，error=%v", sequences, wantVerifications, err)
	}
}

func (f *verificationPostgresFixture) requireRolledBackState() {
	f.t.Helper()
	ctx := context.Background()
	verificationCount, err := f.data.db.FinanceVerification.Query().Where(financeverificationent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || verificationCount != 0 {
		f.t.Fatalf("回滚后核销数 = %d，期望 0，error=%v", verificationCount, err)
	}
	allocationCount, err := f.data.db.FinanceVerificationAllocation.Query().Where(financeverificationallocationent.HasVerificationWith(financeverificationent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || allocationCount != 0 {
		f.t.Fatalf("回滚后核销分摊数 = %d，期望 0，error=%v", allocationCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.verification.create")).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后核销审计数 = %d，期望 0，error=%v", auditCount, err)
	}
	sequenceCount, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID), numberruleent.DocumentTypeEQ(numberruleent.DocumentTypeWriteOff))).Count(ctx)
	if err != nil || sequenceCount != 0 {
		f.t.Fatalf("回滚后核销序列数 = %d，期望 0，error=%v", sequenceCount, err)
	}
}

func (f *verificationPostgresFixture) cleanupVerification() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := f.data.db.FinanceVerificationAllocation.Delete().Where(financeverificationallocationent.HasVerificationWith(financeverificationent.OrganizationIDEQ(f.organizationID))).Exec(ctx); err != nil {
		f.t.Errorf("清理测试核销分摊: %v", err)
	}
	if _, err := f.data.db.FinanceVerification.Delete().Where(financeverificationent.OrganizationIDEQ(f.organizationID)).Exec(ctx); err != nil {
		f.t.Errorf("清理测试核销: %v", err)
	}
	if _, err := f.data.db.FinanceCashflow.Delete().Where(financecashflowent.OrganizationIDEQ(f.organizationID)).Exec(ctx); err != nil {
		f.t.Errorf("清理测试资金流水: %v", err)
	}
}

var _ biz.VerificationRepo = (*invalidAuditResultVerificationRepo)(nil)
