package data

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	exchangeratesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	exchangeratetimestandardent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratetimestandard"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

const financeBillIntegrationDate = "2026-08-30"

type financeBillPostgresFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	partnerID      uuid.UUID
	orderID        uuid.UUID
	actorID        uuid.UUID
	suffix         string
}

type financeBillCreateResult struct {
	bill *biz.FinanceBill
	err  error
}

type invalidAuditResultFinanceBillRepo struct {
	biz.FinanceBillRepo
}

type pausingExchangeRateRepo struct {
	biz.ExchangeRateRepo
	resolved    chan struct{}
	release     chan struct{}
	resolvedOne sync.Once
	releaseOne  sync.Once
}

func (r *invalidAuditResultFinanceBillRepo) Create(ctx context.Context, bill *biz.FinanceBill, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	audit.Result = "invalid"
	return r.FinanceBillRepo.Create(ctx, bill, audit)
}

func (r *pausingExchangeRateRepo) Resolve(ctx context.Context, organizationID uuid.UUID, rateType string, direction biz.OrderFeeDirection, fromCurrency, toCurrency, rateDate string) (*biz.ResolvedExchangeRate, error) {
	resolved, err := r.ExchangeRateRepo.Resolve(ctx, organizationID, rateType, direction, fromCurrency, toCurrency, rateDate)
	if err != nil {
		return nil, err
	}
	r.resolvedOne.Do(func() {
		close(r.resolved)
		<-r.release
	})
	return resolved, nil
}

func (r *pausingExchangeRateRepo) continueResolve() {
	r.releaseOne.Do(func() { close(r.release) })
}

func TestFinanceBillCreateSharedTransactionPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             source,
		AutoMigrate:        true,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	defer cleanup()

	t.Run("相同幂等键并发创建返回同一账单", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("same-key")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-same-key-" + fixture.suffix,
		}

		results := createFinanceBillsConcurrently(usecase, fixture.organizationID, fixture.actorID, input, input)
		for index, result := range results {
			if result.err != nil {
				t.Fatalf("第 %d 个并发请求创建账单: %v", index+1, result.err)
			}
			if result.bill == nil {
				t.Fatalf("第 %d 个并发请求未返回账单", index+1)
			}
		}
		if results[0].bill.ID != results[1].bill.ID || results[0].bill.BillNo != results[1].bill.BillNo {
			t.Fatalf("相同幂等键返回了不同账单: first=%s/%s second=%s/%s", results[0].bill.ID, results[0].bill.BillNo, results[1].bill.ID, results[1].bill.BillNo)
		}
		fixture.requireCommittedState(feeID, 1)
	})

	t.Run("不同幂等键并发使用同一费用只有一个成功", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("different-key")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		first := biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-first-" + fixture.suffix,
		}
		second := first
		second.IdempotencyKey = "bill-second-" + fixture.suffix

		results := createFinanceBillsConcurrently(usecase, fixture.organizationID, fixture.actorID, first, second)
		successes := 0
		feeConflicts := 0
		for _, result := range results {
			switch {
			case result.err == nil && result.bill != nil:
				successes++
			case errors.Is(result.err, biz.ErrFinanceBillFeeInvalid):
				feeConflicts++
			default:
				t.Fatalf("并发创建返回非预期结果: bill=%#v error=%v", result.bill, result.err)
			}
		}
		if successes != 1 || feeConflicts != 1 {
			t.Fatalf("并发创建结果 success=%d feeConflict=%d，期望各 1", successes, feeConflicts)
		}
		fixture.requireCommittedState(feeID, 1)
	})

	t.Run("审计失败回滚账单费用状态和单号序列", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("rollback")
		repo := &invalidAuditResultFinanceBillRepo{FinanceBillRepo: NewFinanceBillRepo(data)}
		usecase := fixture.newUsecase(repo)

		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-rollback-" + fixture.suffix,
		})
		if err == nil {
			t.Fatal("审计结果非法时创建账单未失败")
		}
		fixture.requireRolledBackState(feeID)
	})

	t.Run("并发修改汇率不改变事务内账单快照", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		settingID := fixture.createExchangeRateSetting("7.20000000")
		feeID := fixture.createConfirmedFeeWithCurrency("rate-snapshot", "USD", "7.20000000", "720.00000000", orderfeeent.ExchangeRateSourceSYSTEM)
		exchangeRepo := &pausingExchangeRateRepo{
			ExchangeRateRepo: NewExchangeRateRepo(data), resolved: make(chan struct{}), release: make(chan struct{}),
		}
		defer exchangeRepo.continueResolve()
		usecase := biz.NewFinanceBillUsecase(NewFinanceBillRepo(data), biz.NewExchangeRateUsecase(exchangeRepo), data)
		billResult := make(chan financeBillCreateResult, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			bill, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
				FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-rate-snapshot-" + fixture.suffix,
			})
			billResult <- financeBillCreateResult{bill: bill, err: err}
		}()
		select {
		case <-exchangeRepo.resolved:
		case <-time.After(5 * time.Second):
			t.Fatal("账单事务未完成汇率解析")
		}

		updateResult := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := data.db.ExchangeRateSetting.UpdateOneID(settingID).SetReceivableRate("7.30000000").Save(ctx)
			updateResult <- err
		}()
		select {
		case err := <-updateResult:
			exchangeRepo.continueResolve()
			t.Fatalf("账单事务提交前汇率更新未等待共享锁: %v", err)
		case <-time.After(150 * time.Millisecond):
		}
		exchangeRepo.continueResolve()

		var created financeBillCreateResult
		select {
		case created = <-billResult:
		case <-time.After(10 * time.Second):
			t.Fatal("等待账单事务提交超时")
		}
		if created.err != nil || created.bill == nil {
			t.Fatalf("创建汇率快照账单: bill=%#v error=%v", created.bill, created.err)
		}
		if created.bill.ExchangeRate.StringFixed(8) != "7.20000000" || created.bill.BaseCurrencyAmount.StringFixed(8) != "720.00000000" || created.bill.ExchangeRateSettingID == nil || *created.bill.ExchangeRateSettingID != settingID {
			t.Fatalf("账单未保存事务内汇率快照: %#v", created.bill)
		}
		select {
		case err := <-updateResult:
			if err != nil {
				t.Fatalf("账单提交后更新汇率: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("等待汇率更新超时")
		}
		setting, err := data.db.ExchangeRateSetting.Get(context.Background(), settingID)
		if err != nil || setting.ReceivableRate != "7.30000000" {
			t.Fatalf("并发汇率最终值 = %#v，期望 7.30000000，error=%v", setting, err)
		}
		fixture.requireCommittedState(feeID, 1)
	})
}

func newFinanceBillPostgresFixture(t *testing.T, data *Data) *financeBillPostgresFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("BILL-TX-" + suffix).
		SetName("账单事务集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	fixture := &financeBillPostgresFixture{
		t: t, data: data, organizationID: organization.ID, actorID: uuid.New(), suffix: suffix,
	}
	t.Cleanup(fixture.cleanup)

	partner, err := data.db.Partner.Create().
		SetOrganizationID(organization.ID).
		SetCode("CUSTOMER-" + suffix).
		SetLegalName("账单事务测试客户-" + suffix).
		SetNormalizedName("账单事务测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}
	fixture.partnerID = partner.ID

	order, err := data.db.Order.Create().
		SetOrganizationID(organization.ID).
		SetOrderNo("SE" + suffix).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单: %v", err)
	}
	fixture.orderID = order.ID

	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(organization.ID).
		SetDocumentType(numberruleent.DocumentTypeBill).
		SetPrefix("BILL-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试账单编号规则: %v", err)
	}
	if _, err = data.db.ExchangeRateTimeStandard.Create().
		SetOrganizationID(organization.ID).
		SetRateType(exchangeratetimestandardent.RateTypeBILL).
		SetTimeStandard(exchangeratetimestandardent.TimeStandardBILL_DATE).
		SetSortOrder(0).
		Save(ctx); err != nil {
		t.Fatalf("创建测试账单汇率时间标准: %v", err)
	}
	return fixture
}

func (f *financeBillPostgresFixture) createConfirmedFee(key string) uuid.UUID {
	return f.createConfirmedFeeWithCurrency(key, "CNY", "1.00000000", "100.00000000", orderfeeent.ExchangeRateSourceBASE_CURRENCY)
}

func (f *financeBillPostgresFixture) createConfirmedFeeWithCurrency(key, currency, rate, baseAmount string, source orderfeeent.ExchangeRateSource) uuid.UUID {
	f.t.Helper()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("fee-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusCONFIRMED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency(currency).
		SetExchangeRate(rate).
		SetExchangeRateSource(source).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(baseAmount).
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试费用: %v", err)
	}
	return fee.ID
}

func (f *financeBillPostgresFixture) createExchangeRateSetting(receivableRate string) uuid.UUID {
	f.t.Helper()
	setting, err := f.data.db.ExchangeRateSetting.Create().
		SetOrganizationID(f.organizationID).
		SetRateType(exchangeratesettingent.RateTypeBILL).
		SetFromCurrency("USD").
		SetToCurrency("CNY").
		SetEffectiveFrom(time.Date(2026, 8, 1, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).
		SetReceivableRate(receivableRate).
		SetPayableRate(receivableRate).
		SetIsActive(true).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试汇率: %v", err)
	}
	return setting.ID
}

func (f *financeBillPostgresFixture) newUsecase(repo biz.FinanceBillRepo) *biz.FinanceBillUsecase {
	return biz.NewFinanceBillUsecase(repo, biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data)), f.data)
}

func createFinanceBillsConcurrently(usecase *biz.FinanceBillUsecase, organizationID, actorID uuid.UUID, inputs ...biz.CreateFinanceBillInput) []financeBillCreateResult {
	start := make(chan struct{})
	results := make(chan financeBillCreateResult, len(inputs))
	for _, input := range inputs {
		input := input
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			bill, err := usecase.Create(ctx, organizationID, actorID, input)
			results <- financeBillCreateResult{bill: bill, err: err}
		}()
	}
	close(start)
	collected := make([]financeBillCreateResult, 0, len(inputs))
	for range inputs {
		collected = append(collected, <-results)
	}
	return collected
}

func (f *financeBillPostgresFixture) requireCommittedState(feeID uuid.UUID, wantBills int) {
	f.t.Helper()
	ctx := context.Background()
	billCount, err := f.data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || billCount != wantBills {
		f.t.Fatalf("已提交账单数 = %d，期望 %d，error=%v", billCount, wantBills, err)
	}
	lineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lineCount != wantBills {
		f.t.Fatalf("已提交账单明细数 = %d，期望 %d，error=%v", lineCount, wantBills, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.bill.create")).Count(ctx)
	if err != nil || auditCount != wantBills {
		f.t.Fatalf("已提交账单审计数 = %d，期望 %d，error=%v", auditCount, wantBills, err)
	}
	sequences, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).All(ctx)
	if err != nil || len(sequences) != 1 || sequences[0].CurrentValue != int64(wantBills) {
		f.t.Fatalf("已提交账单序列 = %#v，期望当前值 %d，error=%v", sequences, wantBills, err)
	}
	fee, err := f.data.db.OrderFee.Get(ctx, feeID)
	if err != nil || fee.Status != orderfeeent.StatusBILLED || fee.Version != 2 {
		f.t.Fatalf("已提交费用状态 = %#v，期望 BILLED/version 2，error=%v", fee, err)
	}
}

func (f *financeBillPostgresFixture) requireRolledBackState(feeID uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	billCount, err := f.data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || billCount != 0 {
		f.t.Fatalf("回滚后账单数 = %d，期望 0，error=%v", billCount, err)
	}
	lineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lineCount != 0 {
		f.t.Fatalf("回滚后账单明细数 = %d，期望 0，error=%v", lineCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后审计数 = %d，期望 0，error=%v", auditCount, err)
	}
	sequenceCount, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || sequenceCount != 0 {
		f.t.Fatalf("回滚后账单序列数 = %d，期望 0，error=%v", sequenceCount, err)
	}
	fee, err := f.data.db.OrderFee.Get(ctx, feeID)
	if err != nil || fee.Status != orderfeeent.StatusCONFIRMED || fee.Version != 1 {
		f.t.Fatalf("回滚后费用状态 = %#v，期望 CONFIRMED/version 1，error=%v", fee, err)
	}
}

func (f *financeBillPostgresFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "账单审计", run: func() error {
			_, err := f.data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "账单明细", run: func() error {
			_, err := f.data.db.FinanceBillLine.Delete().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "账单", run: func() error {
			_, err := f.data.db.FinanceBill.Delete().Where(financebillent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "编号序列", run: func() error {
			_, err := f.data.db.NumberSequence.Delete().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "编号规则", run: func() error {
			_, err := f.data.db.NumberRule.Delete().Where(numberruleent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单费用", run: func() error {
			_, err := f.data.db.OrderFee.Delete().Where(orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单", run: func() error {
			_, err := f.data.db.Order.Delete().Where(orderent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "往来单位", run: func() error {
			_, err := f.data.db.Partner.Delete().Where(partnerent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "汇率设置", run: func() error {
			_, err := f.data.db.ExchangeRateSetting.Delete().Where(exchangeratesettingent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "汇率时间标准", run: func() error {
			_, err := f.data.db.ExchangeRateTimeStandard.Delete().Where(exchangeratetimestandardent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "组织", run: func() error { return f.data.db.Organization.DeleteOneID(f.organizationID).Exec(ctx) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理测试%s: %v", step.name, err)
		}
	}
}

var _ biz.FinanceBillRepo = (*invalidAuditResultFinanceBillRepo)(nil)
var _ biz.ExchangeRateRepo = (*pausingExchangeRateRepo)(nil)
