package data

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	exchangeratesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	exchangeratetimestandardent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratetimestandard"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	membership "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

const financeCommissionIntegrationDate = "2026-08-30"

type commissionPostgresFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	customerID     uuid.UUID
	employeeID     uuid.UUID
	orderID        uuid.UUID
	billID         uuid.UUID
	cashflowID     uuid.UUID
	verificationID uuid.UUID
	ruleID         uuid.UUID
	actorID        uuid.UUID
	suffix         string
}

type invalidAuditResultCommissionRepo struct {
	biz.CommissionRepo
	data                 *Data
	createCalled         bool
	innerError           error
	evidenceError        error
	sawUncommittedMain   bool
	uncommittedLineCount int
}

func (r *invalidAuditResultCommissionRepo) Create(ctx context.Context, org uuid.UUID, c *biz.FinanceCommission, snapshot *biz.CommissionCNYSnapshot, audit *biz.AuditEvent) error {
	r.createCalled = true
	audit.Result = "invalid"
	r.innerError = r.CommissionRepo.Create(ctx, org, c, snapshot, audit)
	if r.innerError != nil {
		client, err := r.data.client(ctx)
		if err != nil {
			r.evidenceError = err
			return r.innerError
		}
		r.sawUncommittedMain, r.evidenceError = client.FinanceCommission.Query().Where(commission.IDEQ(c.ID), commission.OrganizationIDEQ(org)).Exist(ctx)
		if r.evidenceError == nil {
			r.uncommittedLineCount, r.evidenceError = client.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(c.ID), commissionline.OrganizationIDEQ(org)).Count(ctx)
		}
	}
	return r.innerError
}

type invalidSaveCommissionRepo struct {
	biz.CommissionRepo
	createCalled bool
	innerError   error
}

func (r *invalidSaveCommissionRepo) Create(ctx context.Context, org uuid.UUID, c *biz.FinanceCommission, snapshot *biz.CommissionCNYSnapshot, audit *biz.AuditEvent) error {
	r.createCalled = true
	snapshot.ExchangeRateSource = "INVALID_PERSISTENCE_SOURCE"
	r.innerError = r.CommissionRepo.Create(ctx, org, c, snapshot, audit)
	return r.innerError
}

type observingCommissionRepo struct {
	biz.CommissionRepo
	getCalls           int
	getUsedTransaction bool
}

func (r *observingCommissionRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCommission, error) {
	r.getCalls++
	_, r.getUsedTransaction = transactionFromContext(ctx)
	return r.CommissionRepo.Get(ctx, org, id)
}

func TestCommissionCreateSharedTransactionPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	t.Run("成功创建并提交后重读断言完整快照与审计", func(t *testing.T) {
		fixture := newCommissionPostgresFixture(t)
		repo := &observingCommissionRepo{CommissionRepo: NewCommissionRepo(fixture.data)}
		usecase := fixture.newUsecase(repo)

		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("success"))
		if err != nil {
			t.Fatalf("创建提成失败: %v", err)
		}
		if created == nil {
			t.Fatal("创建提成未返回实体")
		}
		if created.Status != biz.CommissionDraft || created.Version != 1 {
			t.Fatalf("返回提成状态或版本不符: status=%s version=%d", created.Status, created.Version)
		}
		if created.CommissionDate != financeCommissionIntegrationDate {
			t.Fatalf("返回提成归属日期不符: got=%s want=%s", created.CommissionDate, financeCommissionIntegrationDate)
		}
		if created.CNYExchangeRate.StringFixed(8) != "1.00000000" || created.CNYExchangeRateSource != biz.CommissionCNYRateSourceBaseCurrency || created.CNYExchangeRateDate != financeCommissionIntegrationDate {
			t.Fatalf("返回提成 CNY 汇率快照不符: rate=%s source=%s date=%s", created.CNYExchangeRate.StringFixed(8), created.CNYExchangeRateSource, created.CNYExchangeRateDate)
		}
		if created.CNYExchangeRateSettingID != nil || created.SourceFingerprint == "" || created.CustomerCount != 1 || created.OrderCount != 1 || created.FeeCount != 2 {
			t.Fatalf("返回提成快照统计不符: settingID=%v fingerprint=%q customers=%d orders=%d fees=%d", created.CNYExchangeRateSettingID, created.SourceFingerprint, created.CustomerCount, created.OrderCount, created.FeeCount)
		}
		if created.CommissionAmount.StringFixed(8) != "60.00000000" || created.CNYCommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("返回提成金额不符: amount=%s cnyAmount=%s", created.CommissionAmount.StringFixed(8), created.CNYCommissionAmount.StringFixed(8))
		}
		if len(created.Lines) != 1 {
			t.Fatalf("返回提成明细行数 = %d，期望 1", len(created.Lines))
		}
		line := created.Lines[0]
		if line.OrderID != fixture.orderID || line.CustomerID != fixture.customerID || line.EmployeeID != fixture.employeeID {
			t.Fatalf("返回提成明细关联实体不符: order=%s customer=%s employee=%s", line.OrderID, line.CustomerID, line.EmployeeID)
		}
		if line.RealizedRevenue.StringFixed(8) != "1000.00000000" || line.AllocatedCost.StringFixed(8) != "400.00000000" || line.RealizedProfit.StringFixed(8) != "600.00000000" || line.CommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("返回提成明细金额不符: rev=%s cost=%s profit=%s amount=%s", line.RealizedRevenue.StringFixed(8), line.AllocatedCost.StringFixed(8), line.RealizedProfit.StringFixed(8), line.CommissionAmount.StringFixed(8))
		}
		if repo.getCalls != 1 || repo.getUsedTransaction {
			t.Fatalf("创建成功后重读边界不符: getCalls=%d getUsedTransaction=%t", repo.getCalls, repo.getUsedTransaction)
		}
		fixture.requireCommittedState(created.ID)
	})

	t.Run("真实汇率解析失败时不创建提成主单明细与审计", func(t *testing.T) {
		fixture := newCommissionPostgresFixture(t)
		ctx := context.Background()
		// 删除核销汇率时间标准，触发真实 ExchangeRateUsecase.Resolve 日期缺失错误
		if _, err := fixture.data.db.ExchangeRateTimeStandard.Delete().Where(exchangeratetimestandardent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx); err != nil {
			t.Fatalf("删除汇率时间标准: %v", err)
		}
		usecase := fixture.newUsecase(NewCommissionRepo(fixture.data))

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("rate-fail"))
		if err == nil {
			t.Fatalf("汇率时间标准缺失时未返回错误: created=%#v", created)
		}
		if !errors.Is(err, biz.ErrExchangeRateDateMissing) {
			t.Fatalf("返回错误 = %v，期望 %v", err, biz.ErrExchangeRateDateMissing)
		}
		fixture.requireRolledBackState()
	})

	t.Run("委托真实仓储保存失败时回滚事务且无半成品", func(t *testing.T) {
		fixture := newCommissionPostgresFixture(t)
		repo := &invalidSaveCommissionRepo{CommissionRepo: NewCommissionRepo(fixture.data)}
		usecase := fixture.newUsecase(repo)

		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("save-fail"))
		if err == nil {
			t.Fatalf("持久化值非法时未返回错误: created=%#v", created)
		}
		if !repo.createCalled || repo.innerError == nil {
			t.Fatalf("保存失败装饰器未委托真实仓储: createCalled=%t innerError=%v", repo.createCalled, repo.innerError)
		}
		if !ent.IsValidationError(err) || !strings.Contains(err.Error(), "FinanceCommission.cny_exchange_rate_source") {
			t.Fatalf("保存失败未命中提成持久化字段校验: %T %v", err, err)
		}
		fixture.requireRolledBackState()
	})

	t.Run("主单与明细写入后审计失败回滚全部事务内产物", func(t *testing.T) {
		fixture := newCommissionPostgresFixture(t)
		repo := &invalidAuditResultCommissionRepo{CommissionRepo: NewCommissionRepo(fixture.data), data: fixture.data}
		usecase := fixture.newUsecase(repo)

		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("audit-fail"))
		if err == nil {
			t.Fatalf("审计结果非法时未返回错误: created=%#v", created)
		}
		if !repo.createCalled || repo.innerError == nil {
			t.Fatalf("审计失败装饰器未委托真实仓储: createCalled=%t innerError=%v", repo.createCalled, repo.innerError)
		}
		if !ent.IsValidationError(err) || !strings.Contains(err.Error(), "AuditLog.result") {
			t.Fatalf("审计失败未命中真实审计持久化字段校验: %T %v", err, err)
		}
		if repo.evidenceError != nil || !repo.sawUncommittedMain || repo.uncommittedLineCount != 1 {
			t.Fatalf("审计失败前未观察到同事务内主单与明细: main=%t lines=%d evidenceError=%v", repo.sawUncommittedMain, repo.uncommittedLineCount, repo.evidenceError)
		}
		fixture.requireRolledBackState()
	})
}

func newCommissionPostgresFixture(t *testing.T) *commissionPostgresFixture {
	t.Helper()
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	data, cleanup, err := newIntegrationData(source)
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	// 先注册关库，利用 Cleanup 的 LIFO 顺序保证夹具删除先于连接关闭。
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	actorID := uuid.New()

	org, err := data.db.Organization.Create().
		SetCode("COMM-TX-" + suffix).
		SetName("提成事务测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}

	fixture := &commissionPostgresFixture{
		t:              t,
		data:           data,
		organizationID: org.ID,
		actorID:        actorID,
		suffix:         suffix,
	}
	t.Cleanup(fixture.cleanup)

	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-" + suffix).
		SetLegalName("提成测试客户-" + suffix).
		SetNormalizedName("提成测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	fixture.customerID = customer.ID

	employee, err := data.db.User.Create().
		SetUsername("comm_" + suffix).
		SetDisplayName("提成业务员-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试员工: %v", err)
	}
	fixture.employeeID = employee.ID

	if _, err = data.db.Membership.Create().
		SetOrganizationID(org.ID).
		SetUserID(employee.ID).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试员工成员资格: %v", err)
	}

	order, err := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE" + suffix).
		SetCustomerID(customer.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetOrderDate(financeCommissionIntegrationDate).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单: %v", err)
	}
	fixture.orderID = order.ID

	if _, err = data.db.OrderCommissionAttribution.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetCustomerID(customer.ID).
		SetSourceAssignmentID(uuid.New()).
		SetEmployeeID(employee.ID).
		SetEmployeeName(employee.DisplayName).
		SetPersonnelRole(attribution.PersonnelRoleSALES).
		SetAttributedAt(time.Now()).
		Save(ctx); err != nil {
		t.Fatalf("创建测试提成归属: %v", err)
	}

	feeReceivable, err := data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fee-rec-" + suffix).
		SetDirection(fee.DirectionRECEIVABLE).
		SetStatus(fee.StatusCONFIRMED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(customer.ID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("1000.0000").
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(fee.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("1000.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试应收费用: %v", err)
	}

	if _, err = data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fee-pay-" + suffix).
		SetDirection(fee.DirectionPAYABLE).
		SetStatus(fee.StatusCONFIRMED).
		SetFeeCode("COST").
		SetFeeName("成本费").
		SetSettlementPartyID(customer.ID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("400.0000").
		SetTotalAmount("400.00000000").
		SetNetAmount("400.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(fee.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("400.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx); err != nil {
		t.Fatalf("创建测试应付费用: %v", err)
	}

	billItem, err := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("BILL-" + suffix).
		SetIdempotencyKey("bill-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetFeeCount(1).
		SetBillDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试账单: %v", err)
	}
	fixture.billID = billItem.ID

	if _, err = data.db.FinanceBillLine.Create().
		SetBillID(billItem.ID).
		SetOrderID(order.ID).
		SetOrderFeeID(feeReceivable.ID).
		SetOrderNo(order.OrderNo).
		SetFeeCode(feeReceivable.FeeCode).
		SetFeeName(feeReceivable.FeeName).
		SetQuantity("1.0000").
		SetUnitPrice("1000.0000").
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetBaseCurrency("CNY").
		SetActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试账单明细: %v", err)
	}

	cashflow, err := data.db.FinanceCashflow.Create().
		SetOrganizationID(org.ID).
		SetFlowNo("FLOW-" + suffix).
		SetIdempotencyKey("cashflow-" + suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetAmount("1000.00000000").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("1000.00000000").
		SetTransactionDate(financeCommissionIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试资金流水: %v", err)
	}
	fixture.cashflowID = cashflow.ID

	verificationItem, err := data.db.FinanceVerification.Create().
		SetOrganizationID(org.ID).
		SetVerificationNo("VR-" + suffix).
		SetIdempotencyKey("verification-" + suffix).
		SetDirection(verification.DirectionRECEIVABLE).
		SetStatus(verification.StatusACTIVE).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetAmount("1000.00000000").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(verification.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseAmount("1000.00000000").
		SetBillBaseAmount("1000.00000000").
		SetCashflowBaseAmount("1000.00000000").
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试核销单: %v", err)
	}
	fixture.verificationID = verificationItem.ID

	if _, err = data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationItem.ID).
		SetCashflowID(cashflow.ID).
		SetBillID(billItem.ID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo(billItem.BillNo).
		SetAmount("1000.00000000").
		SetBillBaseAmount("1000.00000000").
		SetCashflowBaseAmount("1000.00000000").
		SetWriteOffBaseAmount("1000.00000000").
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试核销分摊: %v", err)
	}

	ruleItem, err := data.db.FinanceCommissionRule.Create().
		SetOrganizationID(org.ID).
		SetName("销售提成规则-" + suffix).
		SetPersonnelRole(rule.PersonnelRoleSALES).
		SetCalculationBasis(rule.CalculationBasisREALIZED_PROFIT).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试提成规则: %v", err)
	}
	fixture.ruleID = ruleItem.ID

	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(org.ID).
		SetDocumentType(numberruleent.DocumentTypeCommission).
		SetPrefix("TC-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试提成编号规则: %v", err)
	}

	if _, err = data.db.ExchangeRateTimeStandard.Create().
		SetOrganizationID(org.ID).
		SetRateType(exchangeratetimestandardent.RateTypeWRITE_OFF).
		SetTimeStandard(exchangeratetimestandardent.TimeStandardWRITE_OFF_TIME).
		SetSortOrder(0).
		Save(ctx); err != nil {
		t.Fatalf("创建测试核销汇率时间标准: %v", err)
	}

	return fixture
}

func (f *commissionPostgresFixture) input(key string) biz.CreateCommissionInput {
	return biz.CreateCommissionInput{
		VerificationID: f.verificationID,
		EmployeeID:     f.employeeID,
		RuleID:         f.ruleID,
		IdempotencyKey: "commission-" + key + "-" + f.suffix,
	}
}

func (f *commissionPostgresFixture) newUsecase(repo biz.CommissionRepo) *biz.CommissionUsecase {
	return biz.NewCommissionUsecase(
		repo,
		biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)),
		biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data)),
		f.data,
	)
}

func (f *commissionPostgresFixture) requireCommittedState(commissionID uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	comm, err := f.data.db.FinanceCommission.Query().Where(commission.IDEQ(commissionID), commission.OrganizationIDEQ(f.organizationID)).Only(ctx)
	if err != nil || comm == nil {
		f.t.Fatalf("查询已提交提成主单失败: comm=%#v error=%v", comm, err)
	}
	if comm.Status != commission.StatusDRAFT || comm.Version != 1 {
		f.t.Fatalf("已提交提成状态或版本不符: status=%s version=%d", comm.Status, comm.Version)
	}
	if comm.CommissionDate != financeCommissionIntegrationDate {
		f.t.Fatalf("已提交提成归属日期不符: got=%s want=%s", comm.CommissionDate, financeCommissionIntegrationDate)
	}
	if comm.CnyExchangeRate != "1.00000000" || comm.CnyExchangeRateSource != commission.CnyExchangeRateSourceBASE_CURRENCY || comm.CnyExchangeRateDate != financeCommissionIntegrationDate || comm.CnyExchangeRateSettingID != nil || comm.CnyCommissionAmount != "60.00000000" {
		f.t.Fatalf("已提交提成 CNY 快照不符: rate=%s source=%s date=%s settingID=%v amount=%s", comm.CnyExchangeRate, comm.CnyExchangeRateSource, comm.CnyExchangeRateDate, comm.CnyExchangeRateSettingID, comm.CnyCommissionAmount)
	}
	if comm.VerificationID != f.verificationID || comm.EmployeeID != f.employeeID || comm.RuleID == nil || *comm.RuleID != f.ruleID || comm.SourceFingerprint == "" || comm.CustomerCount != 1 || comm.OrderCount != 1 || comm.FeeCount != 2 {
		f.t.Fatalf("已提交提成来源统计不符: verification=%s employee=%s rule=%v fingerprint=%q customers=%d orders=%d fees=%d", comm.VerificationID, comm.EmployeeID, comm.RuleID, comm.SourceFingerprint, comm.CustomerCount, comm.OrderCount, comm.FeeCount)
	}
	lines, err := f.data.db.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(commissionID), commissionline.OrganizationIDEQ(f.organizationID)).All(ctx)
	if err != nil || len(lines) != 1 {
		f.t.Fatalf("已提交提成明细数量不符: got=%d, error=%v", len(lines), err)
	}
	line := lines[0]
	if line.OrderID != f.orderID || line.EmployeeID != f.employeeID || line.RealizedRevenue != "1000.00000000" || line.AllocatedCost != "400.00000000" || line.RealizedProfit != "600.00000000" || line.CommissionAmount != "60.00000000" {
		f.t.Fatalf("已提交提成明细不符: order=%s employee=%s rev=%s cost=%s profit=%s amount=%s", line.OrderID, line.EmployeeID, line.RealizedRevenue, line.AllocatedCost, line.RealizedProfit, line.CommissionAmount)
	}
	audit, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.commission.create"), auditlogent.ResourceIDEQ(commissionID.String())).Only(ctx)
	if err != nil {
		f.t.Fatalf("查询已提交提成创建审计失败: %v", err)
	}
	if audit.UserID == nil || *audit.UserID != f.actorID || audit.ResourceType != "finance_commission" || audit.Result != auditlogent.ResultSuccess {
		f.t.Fatalf("已提交提成创建审计字段不符: user=%v resourceType=%s result=%s", audit.UserID, audit.ResourceType, audit.Result)
	}
	details := make(map[string]string)
	if err = json.Unmarshal(audit.Details, &details); err != nil {
		f.t.Fatalf("解析已提交提成创建审计详情失败: %v", err)
	}
	if details["commission_date"] != financeCommissionIntegrationDate || details["cny.exchange_rate"] != "1.00000000" || details["cny.rate_date"] != financeCommissionIntegrationDate || details["cny.source"] != biz.CommissionCNYRateSourceBaseCurrency {
		f.t.Fatalf("已提交提成创建审计详情不符: %#v", details)
	}
	sequences, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID), numberruleent.DocumentTypeEQ(numberruleent.DocumentTypeCommission))).All(ctx)
	if err != nil || len(sequences) != 1 || sequences[0].CurrentValue != 1 {
		f.t.Fatalf("已提交提成编号序列 = %#v，期望当前值 1，error=%v", sequences, err)
	}
}

func (f *commissionPostgresFixture) requireRolledBackState() {
	f.t.Helper()
	ctx := context.Background()
	commCount, err := f.data.db.FinanceCommission.Query().Where(commission.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || commCount != 0 {
		f.t.Fatalf("回滚后提成主单数 = %d，期望 0，error=%v", commCount, err)
	}
	lineCount, err := f.data.db.FinanceCommissionLine.Query().Where(commissionline.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || lineCount != 0 {
		f.t.Fatalf("回滚后提成明细数 = %d，期望 0，error=%v", lineCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.commission.create")).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后提成创建审计数 = %d，期望 0，error=%v", auditCount, err)
	}
	if _, err = f.data.db.FinanceVerification.Query().Where(verification.IDEQ(f.verificationID), verification.OrganizationIDEQ(f.organizationID)).Only(ctx); err != nil {
		f.t.Fatalf("回滚后核销来源不可读: %v", err)
	}
	allocationCount, err := f.data.db.FinanceVerificationAllocation.Query().Where(allocation.HasVerificationWith(verification.IDEQ(f.verificationID))).Count(ctx)
	if err != nil || allocationCount != 1 {
		f.t.Fatalf("回滚后核销分摊数 = %d，期望 1，error=%v", allocationCount, err)
	}
	if _, err = f.data.db.FinanceCashflow.Query().Where(financecashflowent.IDEQ(f.cashflowID), financecashflowent.OrganizationIDEQ(f.organizationID)).Only(ctx); err != nil {
		f.t.Fatalf("回滚后资金流水来源不可读: %v", err)
	}
	if _, err = f.data.db.FinanceBill.Query().Where(financebillent.IDEQ(f.billID), financebillent.OrganizationIDEQ(f.organizationID)).Only(ctx); err != nil {
		f.t.Fatalf("回滚后账单来源不可读: %v", err)
	}
	billLineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(f.billID), financebilllineent.OrderIDEQ(f.orderID), financebilllineent.ActiveEQ(true)).Count(ctx)
	if err != nil || billLineCount != 1 {
		f.t.Fatalf("回滚后活动账单明细数 = %d，期望 1，error=%v", billLineCount, err)
	}
	if _, err = f.data.db.Order.Query().Where(orderent.IDEQ(f.orderID), orderent.OrganizationIDEQ(f.organizationID)).Only(ctx); err != nil {
		f.t.Fatalf("回滚后订单来源不可读: %v", err)
	}
	feeCount, err := f.data.db.OrderFee.Query().Where(fee.OrderIDEQ(f.orderID), fee.StatusEQ(fee.StatusCONFIRMED)).Count(ctx)
	if err != nil || feeCount != 2 {
		f.t.Fatalf("回滚后已确认订单费用数 = %d，期望 2，error=%v", feeCount, err)
	}
}

func (f *commissionPostgresFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "审计日志", run: func() error {
			_, err := f.data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "提成明细", run: func() error {
			_, err := f.data.db.FinanceCommissionLine.Delete().Where(commissionline.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "提成调整", run: func() error {
			_, err := f.data.db.FinanceCommissionAdjustment.Delete().Where(adjustment.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "提成主单", run: func() error {
			_, err := f.data.db.FinanceCommission.Delete().Where(commission.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "提成规则", run: func() error {
			_, err := f.data.db.FinanceCommissionRule.Delete().Where(rule.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "提成归属", run: func() error {
			_, err := f.data.db.OrderCommissionAttribution.Delete().Where(attribution.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "核销分摊", run: func() error {
			_, err := f.data.db.FinanceVerificationAllocation.Delete().Where(allocation.HasVerificationWith(verification.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "核销单", run: func() error {
			_, err := f.data.db.FinanceVerification.Delete().Where(verification.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "资金流水", run: func() error {
			_, err := f.data.db.FinanceCashflow.Delete().Where(financecashflowent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
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
		{name: "订单费用", run: func() error {
			_, err := f.data.db.OrderFee.Delete().Where(fee.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单", run: func() error {
			_, err := f.data.db.Order.Delete().Where(orderent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "用户成员关系", run: func() error {
			_, err := f.data.db.Membership.Delete().Where(membership.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "测试用户", run: func() error {
			return f.data.db.User.DeleteOneID(f.employeeID).Exec(ctx)
		}},
		{name: "往来单位", run: func() error {
			_, err := f.data.db.Partner.Delete().Where(partnerent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
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
		{name: "汇率设置", run: func() error {
			_, err := f.data.db.ExchangeRateSetting.Delete().Where(exchangeratesettingent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "汇率时间标准", run: func() error {
			_, err := f.data.db.ExchangeRateTimeStandard.Delete().Where(exchangeratetimestandardent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "组织", run: func() error {
			return f.data.db.Organization.DeleteOneID(f.organizationID).Exec(ctx)
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理测试%s: %v", step.name, err)
		}
	}
}

func TestCommissionBillLockOrderConcurrentPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	data, cleanup, err := newIntegrationData(source)
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	// 先注册关库，利用 Cleanup 的 LIFO 顺序保证夹具删除先于连接关闭。
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	org, err := data.db.Organization.Create().
		SetCode("COMM-ORG-" + suffix).
		SetName("提成锁顺序测试组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if _, cleanupErr := data.db.FinanceBill.Delete().Where(financebillent.OrganizationIDEQ(org.ID)).Exec(cleanupCtx); cleanupErr != nil {
			t.Errorf("清理测试账单: %v", cleanupErr)
		}
		if _, cleanupErr := data.db.Partner.Delete().Where(partnerent.OrganizationIDEQ(org.ID)).Exec(cleanupCtx); cleanupErr != nil {
			t.Errorf("清理测试往来单位: %v", cleanupErr)
		}
		if cleanupErr := data.db.Organization.DeleteOneID(org.ID).Exec(cleanupCtx); cleanupErr != nil {
			t.Errorf("清理测试组织: %v", cleanupErr)
		}
	})

	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("PARTNER-" + suffix).
		SetLegalName("提成锁测试单位-" + suffix).
		SetNormalizedName("提成锁测试单位-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}

	billA, err := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("BILL-A-" + suffix).
		SetIdempotencyKey("bill-a-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(partner.ID).
		SetSettlementPartyName(partner.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate("2026-08-30").
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetFeeCount(1).
		SetBillDate("2026-08-30").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建账单 A 失败: %v", err)
	}

	billB, err := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("BILL-B-" + suffix).
		SetIdempotencyKey("bill-b-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(partner.ID).
		SetSettlementPartyName(partner.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate("2026-08-30").
		SetTotalAmount("2000.00000000").
		SetNetAmount("2000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("2000.00000000").
		SetFeeCount(1).
		SetBillDate("2026-08-30").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建账单 B 失败: %v", err)
	}

	var (
		wg               sync.WaitGroup
		transactionReady = make(chan struct{}, 2)
		startQueries     = make(chan struct{})
		err1, err2       error
		bills1, bills2   []*ent.FinanceBill
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		txCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err1 = data.WithTx(txCtx, func(tx *ent.Tx) error {
			transactionReady <- struct{}{}
			select {
			case <-startQueries:
			case <-txCtx.Done():
				return txCtx.Err()
			}
			var qErr error
			bills1, qErr = commissionCalculationBillsQuery(commissionStoreFromTx(tx), org.ID, []uuid.UUID{billA.ID, billB.ID}, true).All(txCtx)
			if qErr != nil {
				return qErr
			}
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}()

	go func() {
		defer wg.Done()
		txCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err2 = data.WithTx(txCtx, func(tx *ent.Tx) error {
			transactionReady <- struct{}{}
			select {
			case <-startQueries:
			case <-txCtx.Done():
				return txCtx.Err()
			}
			var qErr error
			bills2, qErr = commissionCalculationBillsQuery(commissionStoreFromTx(tx), org.ID, []uuid.UUID{billB.ID, billA.ID}, true).All(txCtx)
			if qErr != nil {
				return qErr
			}
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}()

	readyTimer := time.NewTimer(5 * time.Second)
	defer readyTimer.Stop()
	for readyCount := 0; readyCount < 2; readyCount++ {
		select {
		case <-transactionReady:
		case <-readyTimer.C:
			close(startQueries)
			wg.Wait()
			t.Fatal("等待两个 PostgreSQL 事务进入查询栅栏超时")
		}
	}
	close(startQueries)
	wg.Wait()

	if err1 != nil {
		t.Fatalf("事务 1 执行失败 (input: [A, B]): %v", err1)
	}
	if err2 != nil {
		t.Fatalf("事务 2 执行失败 (input: [B, A]): %v", err2)
	}
	if len(bills1) != 2 || len(bills2) != 2 {
		t.Fatalf("事务返回账单数量不符合预期: len(bills1)=%d, len(bills2)=%d", len(bills1), len(bills2))
	}
	if bills1[0].ID.String() >= bills1[1].ID.String() {
		t.Fatalf("事务 1 返回账单未按主键升序排序: %s >= %s", bills1[0].ID, bills1[1].ID)
	}
	if bills2[0].ID.String() >= bills2[1].ID.String() {
		t.Fatalf("事务 2 返回账单未按主键升序排序: %s >= %s", bills2[0].ID, bills2[1].ID)
	}
}

var _ biz.CommissionRepo = (*invalidAuditResultCommissionRepo)(nil)
var _ biz.CommissionRepo = (*invalidSaveCommissionRepo)(nil)
