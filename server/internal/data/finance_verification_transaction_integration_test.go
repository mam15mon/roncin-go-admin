package data

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	exchangeratesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	exchangeratetimestandardent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratetimestandard"
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
	settingID  uuid.UUID
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
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	t.Run("相同幂等键并发创建返回同一核销", func(t *testing.T) {
		fixture := newVerificationPostgresFixture(t)
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
		fixture.requireCommittedState(1, "7.30000000")
	})

	t.Run("审计失败回滚核销分摊和单号序列", func(t *testing.T) {
		fixture := newVerificationPostgresFixture(t)
		repo := &invalidAuditResultVerificationRepo{VerificationRepo: NewVerificationRepo(fixture.data)}
		usecase := fixture.newUsecase(repo, NewExchangeRateRepo(fixture.data))

		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("rollback"))
		if err == nil {
			t.Fatal("审计结果非法时创建核销未失败")
		}
		fixture.requireRolledBackState()
	})

	t.Run("并发修改汇率不改变事务内核销快照", func(t *testing.T) {
		fixture := newVerificationPostgresFixture(t)
		exchangeRepo := &pausingExchangeRateRepo{
			ExchangeRateRepo: NewExchangeRateRepo(fixture.data), resolved: make(chan struct{}), release: make(chan struct{}),
		}
		defer exchangeRepo.continueResolve()
		usecase := fixture.newUsecase(NewVerificationRepo(fixture.data), exchangeRepo)
		verificationResult := make(chan verificationCreateResult, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			verification, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("rate-snapshot"))
			verificationResult <- verificationCreateResult{verification: verification, err: err}
		}()
		select {
		case <-exchangeRepo.resolved:
		case <-time.After(5 * time.Second):
			t.Fatal("核销事务未完成汇率解析")
		}

		updateResult := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := fixture.data.db.ExchangeRateSetting.UpdateOneID(fixture.settingID).SetReceivableRate("7.40000000").Save(ctx)
			updateResult <- err
		}()
		select {
		case err := <-updateResult:
			exchangeRepo.continueResolve()
			t.Fatalf("核销事务提交前汇率更新未等待共享锁: %v", err)
		case <-time.After(150 * time.Millisecond):
		}
		exchangeRepo.continueResolve()

		var created verificationCreateResult
		select {
		case created = <-verificationResult:
		case <-time.After(10 * time.Second):
			t.Fatal("等待核销事务提交超时")
		}
		if created.err != nil || created.verification == nil {
			t.Fatalf("创建汇率快照核销: verification=%#v error=%v", created.verification, created.err)
		}
		if created.verification.ExchangeRate.StringFixed(8) != "7.30000000" || created.verification.BaseAmount.StringFixed(8) != "292.00000000" || created.verification.ExchangeRateSettingID == nil || *created.verification.ExchangeRateSettingID != fixture.settingID {
			t.Fatalf("核销未保存事务内汇率快照: %#v", created.verification)
		}
		select {
		case err := <-updateResult:
			if err != nil {
				t.Fatalf("核销提交后更新汇率: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("等待汇率更新超时")
		}
		setting, err := fixture.data.db.ExchangeRateSetting.Get(context.Background(), fixture.settingID)
		if err != nil || setting.ReceivableRate != "7.40000000" {
			t.Fatalf("并发汇率最终值 = %#v，期望 7.40000000，error=%v", setting, err)
		}
		fixture.requireCommittedState(1, "7.30000000")
	})
}

func newVerificationPostgresFixture(t *testing.T) *verificationPostgresFixture {
	t.Helper()
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	data, cleanup, err := newIntegrationData(source)
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	t.Cleanup(cleanup)
	base := newFinanceBillPostgresFixture(t, data)
	fixture := &verificationPostgresFixture{financeBillPostgresFixture: base}
	t.Cleanup(fixture.cleanupVerification)
	ctx := context.Background()

	if _, err = data.db.NumberRule.Create().SetOrganizationID(fixture.organizationID).SetDocumentType(numberruleent.DocumentTypeWriteOff).SetPrefix("WO-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试核销编号规则: %v", err)
	}
	if _, err = data.db.ExchangeRateTimeStandard.Create().SetOrganizationID(fixture.organizationID).SetRateType(exchangeratetimestandardent.RateTypeWRITE_OFF).SetTimeStandard(exchangeratetimestandardent.TimeStandardWRITE_OFF_TIME).SetSortOrder(0).Save(ctx); err != nil {
		t.Fatalf("创建测试核销汇率时间标准: %v", err)
	}
	setting, err := data.db.ExchangeRateSetting.Create().SetOrganizationID(fixture.organizationID).SetRateType(exchangeratesettingent.RateTypeWRITE_OFF).SetFromCurrency("USD").SetToCurrency("CNY").SetEffectiveFrom(time.Date(2026, 8, 1, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).SetReceivableRate("7.30000000").SetPayableRate("7.30000000").SetIsActive(true).Save(ctx)
	if err != nil {
		t.Fatalf("创建测试核销汇率: %v", err)
	}
	fixture.settingID = setting.ID

	bill, err := data.db.FinanceBill.Create().SetOrganizationID(fixture.organizationID).SetBillNo("BILL-V-" + fixture.suffix).SetIdempotencyKey("bill-verification-" + fixture.suffix).SetDirection(financebillent.DirectionRECEIVABLE).SetStatus(financebillent.StatusCONFIRMED).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).SetCurrency("USD").SetBaseCurrency("CNY").SetExchangeRate("7.20000000").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetTotalAmount("100.00000000").SetNetAmount("100.00000000").SetTaxAmount("0.00000000").SetBaseCurrencyAmount("720.00000000").SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1).Save(ctx)
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

func newIntegrationData(source string) (*Data, func(), error) {
	return NewData(&conf.Data{Database: &conf.Data_Database{Driver: "postgres", Source: source, AutoMigrate: true, MaxOpenConnections: 8, MaxIdleConnections: 8}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func (f *verificationPostgresFixture) input(key string) biz.CreateVerificationInput {
	return biz.CreateVerificationInput{
		Allocations:      []*biz.VerificationAllocation{{CashflowID: f.cashflowID, BillID: f.billID, Amount: decimal.RequireFromString("40")}},
		VerificationDate: financeBillIntegrationDate,
		IdempotencyKey:   "verification-" + key + "-" + f.suffix,
	}
}

func (f *verificationPostgresFixture) newUsecase(repo biz.VerificationRepo, exchangeRepo biz.ExchangeRateRepo) *biz.VerificationUsecase {
	return biz.NewVerificationUsecase(repo, biz.NewExchangeRateUsecase(exchangeRepo), f.data)
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

func (f *verificationPostgresFixture) requireCommittedState(wantVerifications int, wantRate string) {
	f.t.Helper()
	ctx := context.Background()
	verifications, err := f.data.db.FinanceVerification.Query().Where(financeverificationent.OrganizationIDEQ(f.organizationID)).All(ctx)
	if err != nil || len(verifications) != wantVerifications || verifications[0].ExchangeRate != wantRate {
		f.t.Fatalf("已提交核销 = %#v，期望数量 %d、汇率 %s，error=%v", verifications, wantVerifications, wantRate, err)
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
