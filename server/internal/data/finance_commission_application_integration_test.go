package data

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebillline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// 月度申请集成夹具：单一订单（应收 1000 / 应付 400）+ 一张账单；每张核销单
// 按指定归属日期对整张账单分摊，通过「员工 × 身份归属」与「方案 × 员工分配」
// 组合验证候选解析、提交事务、并发唯一与驳回重提。金额口径：利润 600 × 10% = 60。
type commissionApplicationFixture struct {
	t               *testing.T
	data            *Data
	organizationID  uuid.UUID
	customerID      uuid.UUID
	actorID         uuid.UUID
	orderID         uuid.UUID
	receivableFeeID uuid.UUID
	billID          uuid.UUID
	suffix          string
	sourceSeq       int
	schemeSeq       int
	historicalDate  string
	currentDate     string
}

func newCommissionApplicationFixture(t *testing.T) *commissionApplicationFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	today := biz.FinanceBusinessDate(time.Now())
	_, coverageTo, ok := biz.CommissionApplicationPeriod(today)
	if !ok {
		t.Fatalf("推导申请期间失败: %s", today)
	}
	coverageDay, err := time.ParseInLocation("2006-01-02", coverageTo, time.UTC)
	if err != nil {
		t.Fatalf("解析覆盖截止日失败: %v", err)
	}
	historicalDate := coverageDay.AddDate(0, 0, -5).Format("2006-01-02")

	org, err := data.db.Organization.Create().
		SetCode("FCA-" + suffix).
		SetName("月度申请测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	actor, err := data.db.User.Create().
		SetUsername("fca_actor_" + suffix).
		SetDisplayName("月度申请操作员-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试操作员: %v", err)
	}
	if _, err = data.db.Membership.Create().SetOrganizationID(org.ID).SetUserID(actor.ID).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试操作员成员资格: %v", err)
	}
	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-FCA-" + suffix).
		SetLegalName("月度申请客户-" + suffix).
		SetNormalizedName("月度申请客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	order, err := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(org.ID).
		SetOrderNo("FCA-SE" + suffix).
		SetCustomerID(customer.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetOrderDate(historicalDate).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单: %v", err)
	}
	receivableFee, err := data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fca-fee-rec-" + suffix).
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
		SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(historicalDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("1000.00000000").
		SetExpenseDate(historicalDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试应收费用: %v", err)
	}
	if _, err = data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fca-fee-pay-" + suffix).
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
		SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(historicalDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("400.00000000").
		SetExpenseDate(historicalDate).
		SetVersion(1).
		Save(ctx); err != nil {
		t.Fatalf("创建测试应付费用: %v", err)
	}
	billCreate := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("FCA-BILL-" + suffix).
		SetIdempotencyKey("fca-bill-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(historicalDate).
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetFeeCount(1).
		SetBillDate(historicalDate).
		SetVersion(1)
	billItem, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试账单: %v", err)
	}
	if _, err = data.db.FinanceBillLine.Create().
		SetBillID(billItem.ID).
		SetOrderID(order.ID).
		SetOrderFeeID(receivableFee.ID).
		SetOrderNo(order.OrderNo).
		SetFeeCode(receivableFee.FeeCode).
		SetFeeName(receivableFee.FeeName).
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
	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(org.ID).
		SetDocumentType(numberruleent.DocumentTypeCommission).
		SetPrefix("FCA-TC-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试提成编号规则: %v", err)
	}
	fixture := &commissionApplicationFixture{
		t: t, data: data, organizationID: org.ID, customerID: customer.ID, actorID: actor.ID,
		orderID: order.ID, receivableFeeID: receivableFee.ID, billID: billItem.ID,
		suffix: suffix, historicalDate: historicalDate, currentDate: today,
	}
	return fixture
}

func (f *commissionApplicationFixture) newEmployee(label string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	employee, err := f.data.db.User.Create().
		SetUsername("fca_emp_" + label + "_" + f.suffix).
		SetDisplayName("申请员工-" + label + "-" + f.suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试员工 %s: %v", label, err)
	}
	if _, err = f.data.db.Membership.Create().SetOrganizationID(f.organizationID).SetUserID(employee.ID).SetEnabled(true).Save(ctx); err != nil {
		f.t.Fatalf("创建测试员工 %s 成员资格: %v", label, err)
	}
	return employee.ID
}

func (f *commissionApplicationFixture) addAttribution(employeeID uuid.UUID, role string) {
	f.t.Helper()
	if _, err := f.data.db.OrderCommissionAttribution.Create().
		SetOrganizationID(f.organizationID).
		SetOrderID(f.orderID).
		SetCustomerID(f.customerID).
		SetSourceAssignmentID(uuid.New()).
		SetEmployeeID(employeeID).
		SetEmployeeName("申请员工-" + f.suffix).
		SetPersonnelRole(attribution.PersonnelRole(role)).
		SetAttributedAt(time.Now()).
		Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试提成归属: %v", err)
	}
}

// addSalesScheme 为员工配置 10% 已实现毛利方案与起始分配段；方案名按调用序
// 去重，满足组织内方案名唯一约束。
func (f *commissionApplicationFixture) addSalesScheme(employeeID uuid.UUID) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	f.schemeSeq++
	ruleItem, err := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName(fmt.Sprintf("申请销售方案-%s-%02d", f.suffix, f.schemeSeq)).
		SetPersonnelRole(rule.PersonnelRoleSALES).
		SetCalculationBasis(rule.CalculationBasisREALIZED_PROFIT).
		SetRatePercent("10.0000").
		SetEffectiveFrom("2026-01-01").
		SetEnabled(true).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试提成方案: %v", err)
	}
	if _, err = f.data.db.FinanceCommissionRuleAssignment.Create().
		SetOrganizationID(f.organizationID).
		SetRuleID(ruleItem.ID).
		SetEmployeeID(employeeID).
		SetEffectiveFrom("2026-01-01").
		SetCreatedBy(f.actorID).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试方案员工分配: %v", err)
	}
	return ruleItem.ID
}

// addVerification 追加一张对整张账单全额分摊的 ACTIVE 应收核销单，归属日期
// 由调用方指定（历史月或当前月）。
func (f *commissionApplicationFixture) addVerification(verificationDate string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	f.sourceSeq++
	seqLabel := fmt.Sprintf("%02d", f.sourceSeq)
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(f.organizationID).
		SetFlowNo("FCA-FLOW-" + f.suffix + "-" + seqLabel).
		SetIdempotencyKey("fca-cashflow-" + f.suffix + "-" + seqLabel).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(f.customerID).
		SetSettlementPartyName("月度申请客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount("1000.00000000").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(verificationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("1000.00000000").
		SetTransactionDate(verificationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试资金流水: %v", err)
	}
	verificationItem, err := f.data.db.FinanceVerification.Create().
		SetOrganizationID(f.organizationID).
		SetVerificationNo("FCA-VR-" + f.suffix + "-" + seqLabel).
		SetIdempotencyKey("fca-verification-" + f.suffix + "-" + seqLabel).
		SetDirection(verification.DirectionRECEIVABLE).
		SetStatus(verification.StatusACTIVE).
		SetSettlementPartyID(f.customerID).
		SetSettlementPartyName("月度申请客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount("1000.00000000").
		SetBaseCurrency("CNY").
		SetBaseAmount("1000.00000000").
		SetBillBaseAmount("1000.00000000").
		SetCashflowBaseAmount("1000.00000000").
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(verificationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试核销单: %v", err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationItem.ID).
		SetCashflowID(cashflow.ID).
		SetBillID(f.billID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo("FCA-BILL-" + f.suffix).
		SetAmount("1000.00000000").
		SetBillBaseAmount("1000.00000000").
		SetCashflowBaseAmount("1000.00000000").
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试核销分摊: %v", err)
	}
	return verificationItem.ID
}

func (f *commissionApplicationFixture) usecase() *biz.FinanceCommissionApplicationUsecase {
	f.t.Helper()
	return biz.NewFinanceCommissionApplicationUsecase(NewFinanceCommissionApplicationRepo(f.data))
}

func (f *commissionApplicationFixture) commissionUsecase() *biz.CommissionUsecase {
	f.t.Helper()
	return biz.NewCommissionUsecase(
		NewCommissionRepo(f.data),
		biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)),
		f.data,
	)
}

func (f *commissionApplicationFixture) scopeFor(employeeID uuid.UUID) biz.WorkbenchScope {
	now := time.Now()
	return biz.WorkbenchScope{CanOperateBusiness: true,
		OrganizationID: f.organizationID,
		UserID:         employeeID,
		Today:          f.currentDate,
		Now:            now,
	}
}

func (f *commissionApplicationFixture) rejectApplication(t *testing.T, applicationID uuid.UUID, reason string) {
	t.Helper()
	ctx := context.Background()
	header, err := f.data.db.FinanceCommissionApplication.Query().
		Where(applicationent.IDEQ(applicationID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("定位待驳回申请失败: %v", err)
	}
	if _, err = f.data.db.FinanceCommissionApplication.UpdateOneID(header.ID).
		SetStatus(applicationent.StatusREJECTED).
		SetVersion(header.Version + 1).
		SetDecidedAt(time.Now()).
		SetDecidedBy(f.actorID).
		SetDecisionReason(reason).
		Save(ctx); err != nil {
		t.Fatalf("模拟财务驳回失败: %v", err)
	}
}

// TestCommissionApplicationSubmitPostgres 覆盖提交主链路：受控创建 DRAFT 提成、
// 明细快照、月度唯一冲突、迟到来源顺延与申请摘要。
func TestCommissionApplicationSubmitPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()

	employee := fixture.newEmployee("main")
	fixture.addAttribution(employee, "SALES")
	fixture.addSalesScheme(employee)
	verificationID := fixture.addVerification(fixture.historicalDate)
	scope := fixture.scopeFor(employee)

	application, err := usecase.Submit(ctx, scope)
	if err != nil {
		t.Fatalf("提交月度申请失败: %v", err)
	}
	if application.Status != biz.CommissionApplicationPendingReview || application.Version != 1 {
		t.Fatalf("申请头状态/版本不符: %+v", application)
	}
	if application.ApplicationMonth != fixture.currentDate[:7] {
		t.Fatalf("提交月应为当前自然月: %s", application.ApplicationMonth)
	}
	_, wantCoverageTo, _ := biz.CommissionApplicationPeriod(fixture.currentDate)
	if application.CoverageTo != wantCoverageTo {
		t.Fatalf("覆盖截止日应为上一自然月末: %s", application.CoverageTo)
	}
	if application.CommissionCount != 1 || application.TotalCommissionAmount.StringFixed(8) != "60.00000000" {
		t.Fatalf("申请汇总应为 1 笔 60: count=%d total=%s", application.CommissionCount, application.TotalCommissionAmount)
	}

	detail, err := usecase.GetMyApplication(ctx, scope, application.ID)
	if err != nil {
		t.Fatalf("读取本人申请详情失败: %v", err)
	}
	if len(detail.Lines) != 1 {
		t.Fatalf("申请明细应为 1 条: %d", len(detail.Lines))
	}
	line := detail.Lines[0]
	if line.CommissionDate != fixture.historicalDate || string(line.PersonnelRole) != "SALES" {
		t.Fatalf("明细归属日期/身份不符: %+v", line)
	}
	if line.CommissionAmount.StringFixed(8) != "60.00000000" || line.SourceFingerprint == "" {
		t.Fatalf("明细金额/指纹不符: %+v", line)
	}
	// 受控创建路径应生成 DRAFT 提成事实，来源指向核销单。
	commissionRow, err := fixture.data.db.FinanceCommission.Query().
		Where(commission.EmployeeIDEQ(employee)).
		Only(ctx)
	if err != nil {
		t.Fatalf("查询受控创建的提成事实失败: %v", err)
	}
	if commissionRow.Status != commission.StatusDRAFT || commissionRow.VerificationID == nil || *commissionRow.VerificationID != verificationID {
		t.Fatalf("受控创建提成应为来源核销的 DRAFT: %+v", commissionRow)
	}
	if commissionRow.CommissionAmount != "60.00000000" {
		t.Fatalf("受控创建提成金额不符: %s", commissionRow.CommissionAmount)
	}

	// 重复提交：PENDING_REVIEW 月度唯一键下稳定冲突。
	if _, dupErr := usecase.Submit(ctx, scope); !errors.Is(dupErr, biz.ErrCommissionApplicationConflict) {
		t.Fatalf("重复提交应稳定拒绝: %v", dupErr)
	}

	// 提交后可申请候选清空；迟到历史来源进入下一次候选，不回填已提交申请。
	candidates, err := usecase.ListMyCandidates(ctx, scope, biz.WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询可申请候选失败: %v", err)
	}
	if candidates.Total != 0 {
		t.Fatalf("提交后可申请候选应为空: %d", candidates.Total)
	}
	lateVerification := fixture.addVerification(fixture.historicalDate)
	candidates, err = usecase.ListMyCandidates(ctx, scope, biz.WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询迟到候选失败: %v", err)
	}
	if candidates.Total != 1 || len(candidates.Items) != 1 {
		t.Fatalf("迟到历史来源应成为下一次候选: %d", candidates.Total)
	}
	if candidates.Items[0].CommissionAmount.StringFixed(8) != "60.00000000" {
		t.Fatalf("迟到候选金额不符: %s", candidates.Items[0].CommissionAmount)
	}
	detail, err = usecase.GetMyApplication(ctx, scope, application.ID)
	if err != nil {
		t.Fatalf("重读申请详情失败: %v", err)
	}
	if len(detail.Lines) != 1 {
		t.Fatalf("迟到来源不得回填已提交申请: %d", len(detail.Lines))
	}
	_ = lateVerification

	// 摘要：审批中计数 1、最近申请指向本单；迟到候选按归属月分组展示。
	summary, err := usecase.ApplicationSummary(ctx, scope, "CNY")
	if err != nil {
		t.Fatalf("读取申请摘要失败: %v", err)
	}
	if summary.PendingReviewCount != 1 || summary.ApprovedCount != 0 {
		t.Fatalf("申请计数不符: pending=%d approved=%d", summary.PendingReviewCount, summary.ApprovedCount)
	}
	if summary.LatestApplication == nil || summary.LatestApplication.ApplicationID != application.ID {
		t.Fatalf("最近申请概要应指向本单: %+v", summary.LatestApplication)
	}
	if len(summary.ApplyGroups) != 1 || summary.ApplyGroups[0].CommissionCount != 1 ||
		summary.ApplyGroups[0].CommissionAmount.StringFixed(8) != "60.00000000" {
		t.Fatalf("可申请分组应只含迟到候选: %+v", summary.ApplyGroups)
	}

	// 申请历史列表可见。
	list, err := usecase.ListMyApplications(ctx, scope, biz.WorkbenchApplicationFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询本人申请历史失败: %v", err)
	}
	if list.Total != 1 || list.Items[0].ID != application.ID {
		t.Fatalf("本人申请历史不符: %+v", list)
	}

	// 提交审计已落库。
	audits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ResourceIDEQ(application.ID.String()), auditlog.ActionEQ("finance.commission_application.submit")).
		Count(ctx)
	if err != nil {
		t.Fatalf("查询申请审计失败: %v", err)
	}
	if audits != 1 {
		t.Fatalf("提交审计应恰好一条: %d", audits)
	}
}

// TestCommissionApplicationConcurrentSubmitPostgres 验证重复点击与并发提交：
// 成员行锁 + 月度唯一键下恰好一个事务成功，另一侧稳定冲突。
func TestCommissionApplicationConcurrentSubmitPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()

	employee := fixture.newEmployee("race")
	fixture.addAttribution(employee, "SALES")
	fixture.addSalesScheme(employee)
	fixture.addVerification(fixture.historicalDate)
	scope := fixture.scopeFor(employee)

	const submitterCount = 4
	results := make([]error, submitterCount)
	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	for i := 0; i < submitterCount; i++ {
		done.Add(1)
		go func(index int) {
			defer done.Done()
			start.Wait()
			_, err := usecase.Submit(ctx, scope)
			results[index] = err
		}(i)
	}
	start.Add(-1)
	done.Wait()

	successCount := 0
	conflictCount := 0
	for _, err := range results {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, biz.ErrCommissionApplicationConflict):
			conflictCount++
		default:
			t.Fatalf("并发提交出现意外错误: %v", err)
		}
	}
	if successCount != 1 || conflictCount != submitterCount-1 {
		t.Fatalf("并发提交应恰好一成功其余冲突: success=%d conflict=%d", successCount, conflictCount)
	}
	headers, err := fixture.data.db.FinanceCommissionApplication.Query().
		Where(applicationent.OrganizationIDEQ(fixture.organizationID), applicationent.EmployeeIDEQ(employee)).
		All(ctx)
	if err != nil {
		t.Fatalf("查询申请头失败: %v", err)
	}
	if len(headers) != 1 || headers[0].CommissionCount != 1 {
		t.Fatalf("月度唯一键下应只有一张申请且一笔明细: %+v", headers)
	}
	lines, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.EmployeeIDEQ(employee)).
		All(ctx)
	if err != nil {
		t.Fatalf("查询申请明细失败: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("并发提交不得重复纳入提成事实: %d", len(lines))
	}
}

// TestCommissionApplicationGatesPostgres 覆盖当前月拒绝、空候选拒绝与跨组织、
// 跨员工隔离。
func TestCommissionApplicationGatesPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()

	// 仅当前自然月来源：只能累计展示，提交因无截止日前候选被拒绝，不创建空申请。
	currentOnly := fixture.newEmployee("current")
	fixture.addAttribution(currentOnly, "SALES")
	fixture.addSalesScheme(currentOnly)
	fixture.addVerification(fixture.currentDate)
	currentScope := fixture.scopeFor(currentOnly)
	if _, err := usecase.Submit(ctx, currentScope); !errors.Is(err, biz.ErrCommissionApplicationEmpty) {
		t.Fatalf("仅当前月来源的提交应被拒绝: %v", err)
	}
	headers, err := fixture.data.db.FinanceCommissionApplication.Query().
		Where(applicationent.EmployeeIDEQ(currentOnly)).Count(ctx)
	if err != nil {
		t.Fatalf("统计申请头失败: %v", err)
	}
	if headers != 0 {
		t.Fatalf("拒绝提交不得创建空申请: %d", headers)
	}
	summary, err := usecase.ApplicationSummary(ctx, currentScope, "CNY")
	if err != nil {
		t.Fatalf("读取累计摘要失败: %v", err)
	}
	if summary.AccumulatingCount != 1 || len(summary.ApplyGroups) != 0 {
		t.Fatalf("当前月来源应只进入累计中: %+v", summary)
	}

	// 无任何来源的员工同样拒绝提交。
	empty := fixture.newEmployee("empty")
	fixture.addAttribution(empty, "SALES")
	fixture.addSalesScheme(empty)
	if _, err := usecase.Submit(ctx, fixture.scopeFor(empty)); !errors.Is(err, biz.ErrCommissionApplicationEmpty) {
		t.Fatalf("空候选提交应被拒绝: %v", err)
	}

	// 跨组织：员工在另一组织无成员关系，提交稳定拒绝。
	otherScope := fixture.scopeFor(currentOnly)
	otherScope.OrganizationID = uuid.Must(uuid.NewV7())
	if _, err := usecase.Submit(ctx, otherScope); !errors.Is(err, biz.ErrCommissionApplicationEmployeeInvalid) {
		t.Fatalf("跨组织提交应被拒绝: %v", err)
	}

	// 跨员工：本人详情接口查询他人申请按不存在处理；候选只含本人归属。
	main := fixture.newEmployee("main")
	fixture.addAttribution(main, "SALES")
	fixture.addSalesScheme(main)
	fixture.addVerification(fixture.historicalDate)
	mainScope := fixture.scopeFor(main)
	application, err := usecase.Submit(ctx, mainScope)
	if err != nil {
		t.Fatalf("本人提交失败: %v", err)
	}
	otherEmployee := fixture.newEmployee("other")
	if _, err := usecase.GetMyApplication(ctx, fixture.scopeFor(otherEmployee), application.ID); !errors.Is(err, biz.ErrCommissionApplicationNotFound) {
		t.Fatalf("查询他人申请应按不存在处理: %v", err)
	}
	otherScope = fixture.scopeFor(otherEmployee)
	otherCandidates, err := usecase.ListMyCandidates(ctx, otherScope, biz.WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询他人候选失败: %v", err)
	}
	if otherCandidates.Total != 0 {
		t.Fatalf("未分配员工不得看到他人候选: %d", otherCandidates.Total)
	}
	if _, err := usecase.Submit(ctx, otherScope); !errors.Is(err, biz.ErrCommissionApplicationEmpty) {
		t.Fatalf("无资格员工提交应被拒绝: %v", err)
	}
}

// TestCommissionApplicationResubmitPostgres 验证驳回后显式原单重提：按申请 ID
// 与 expected_version 定位原申请，沿用原 ID 与申请月、保留最近一次决策审计、
// 版本递增；提成被其他路径处理后重提整体拒绝，不产生替代申请。
func TestCommissionApplicationResubmitPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()

	employee := fixture.newEmployee("resubmit")
	fixture.addAttribution(employee, "SALES")
	fixture.addSalesScheme(employee)
	fixture.addVerification(fixture.historicalDate)
	scope := fixture.scopeFor(employee)

	application, err := usecase.Submit(ctx, scope)
	if err != nil {
		t.Fatalf("首次提交失败: %v", err)
	}
	fixture.rejectApplication(t, application.ID, "明细存疑，整单驳回")

	// 空提交不再隐式重提：REJECTED 申请占用月度唯一键，必须显式重提。
	if _, err := usecase.Submit(ctx, scope); !errors.Is(err, biz.ErrCommissionApplicationConflict) {
		t.Fatalf("驳回月空提交应稳定冲突并引导显式重提: %v", err)
	}

	// 期望版本不匹配稳定拒绝，不改变申请头。
	if _, err := usecase.Resubmit(ctx, scope, application.ID, 99); !errors.Is(err, biz.ErrCommissionApplicationStatusConflict) {
		t.Fatalf("过期版本的重提应稳定拒绝: %v", err)
	}

	resubmitted, err := usecase.Resubmit(ctx, scope, application.ID, 2)
	if err != nil {
		t.Fatalf("显式原单重提失败: %v", err)
	}
	if resubmitted.ID != application.ID {
		t.Fatalf("重提必须沿用原申请 ID: %s vs %s", resubmitted.ID, application.ID)
	}
	if resubmitted.Status != biz.CommissionApplicationPendingReview || resubmitted.Version != 3 {
		t.Fatalf("重提后应回到待审且版本递增: %+v", resubmitted)
	}
	if resubmitted.DecisionReason == nil || *resubmitted.DecisionReason != "明细存疑，整单驳回" {
		t.Fatalf("重提应保留最近一次驳回原因供员工查看: %+v", resubmitted)
	}
	headers, err := fixture.data.db.FinanceCommissionApplication.Query().
		Where(applicationent.OrganizationIDEQ(fixture.organizationID), applicationent.EmployeeIDEQ(employee)).
		Count(ctx)
	if err != nil {
		t.Fatalf("统计申请头失败: %v", err)
	}
	if headers != 1 {
		t.Fatalf("重提不得新建替代申请: %d", headers)
	}
	resubmitAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ResourceIDEQ(application.ID.String()), auditlog.ActionEQ("finance.commission_application.resubmit")).
		Count(ctx)
	if err != nil {
		t.Fatalf("查询重提审计失败: %v", err)
	}
	if resubmitAudits != 1 {
		t.Fatalf("重提审计应恰好一条: %d", resubmitAudits)
	}

	// 再提交：重提后回到 PENDING_REVIEW，重复提交仍稳定冲突。
	if _, err := usecase.Submit(ctx, scope); !errors.Is(err, biz.ErrCommissionApplicationConflict) {
		t.Fatalf("重提后重复提交应稳定拒绝: %v", err)
	}

	// 明细提成被财务单独确认后重提：唯一明细被剔除后申请内已无有效明细，
	// 稳定拒绝且整体回滚（申请保持 REJECTED、明细保留）。
	line, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(application.ID)).Only(ctx)
	if err != nil {
		t.Fatalf("定位申请明细失败: %v", err)
	}
	if _, err := fixture.data.db.FinanceCommission.UpdateOneID(line.CommissionID).
		SetStatus(commission.StatusCONFIRMED).
		SetConfirmedAt(time.Now()).
		SetConfirmedBy(fixture.actorID).
		Save(ctx); err != nil {
		t.Fatalf("模拟财务确认提成失败: %v", err)
	}
	fixture.rejectApplication(t, application.ID, "再次驳回")
	if _, err := usecase.Resubmit(ctx, scope, application.ID, 4); !errors.Is(err, biz.ErrCommissionApplicationNoValidLines) {
		t.Fatalf("全部明细失效时重提应返回无有效明细: %v", err)
	}
	header, err := fixture.data.db.FinanceCommissionApplication.Query().
		Where(applicationent.IDEQ(application.ID)).Only(ctx)
	if err != nil {
		t.Fatalf("查询申请头失败: %v", err)
	}
	if header.Status != applicationent.StatusREJECTED || header.Version != 4 {
		t.Fatalf("失败重提不得改变申请头: %+v", header)
	}
	lineCount, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(application.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("统计申请明细失败: %v", err)
	}
	if lineCount != 1 {
		t.Fatalf("失败重提不得删除原明细: %d", lineCount)
	}
}

// TestCommissionApplicationSourceConflictPostgres 验证 design §3.3 的提交冲突：
// 指纹失效与已被占用的 DRAFT 提成均整体拒绝，不静默复制或改变历史金额。
func TestCommissionApplicationSourceConflictPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()

	// 场景一：已有 DRAFT 指纹失效（来源费用随后变化）→ 提交返回明确冲突。
	staleEmployee := fixture.newEmployee("stale")
	fixture.addAttribution(staleEmployee, "SALES")
	fixture.addSalesScheme(staleEmployee)
	staleVerification := fixture.addVerification(fixture.historicalDate)
	if _, err := fixture.commissionUsecase().Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateCommissionInput{
		VerificationID: staleVerification,
		EmployeeID:     staleEmployee,
		PersonnelRole:  biz.CommissionRoleSales,
		IdempotencyKey: "fca-stale-" + fixture.suffix,
	}); err != nil {
		t.Fatalf("预置 DRAFT 提成失败: %v", err)
	}
	if _, err := fixture.data.db.OrderFee.UpdateOneID(fixture.receivableFeeID).
		SetBaseCurrencyAmount("1100.00000000").
		SetTotalAmount("1100.00000000").
		SetNetAmount("1100.00000000").
		SetUnitPrice("1100.0000").
		SetVersion(2).
		Save(ctx); err != nil {
		t.Fatalf("修改来源费用制造指纹漂移失败: %v", err)
	}
	staleScope := fixture.scopeFor(staleEmployee)
	if _, err := usecase.Submit(ctx, staleScope); !errors.Is(err, biz.ErrCommissionApplicationSourceConflict) {
		t.Fatalf("指纹失效 DRAFT 应返回来源冲突: %v", err)
	}
	// 读取路径静默排除失效候选：候选为空而非冲突。
	candidates, err := usecase.ListMyCandidates(ctx, staleScope, biz.WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询失效候选失败: %v", err)
	}
	if candidates.Total != 0 {
		t.Fatalf("失效 DRAFT 不得出现在可申请候选: %d", candidates.Total)
	}

	// 场景二：DRAFT 已被其他申请占用 → 提交返回明确冲突。
	occupiedEmployee := fixture.newEmployee("occupied")
	fixture.addAttribution(occupiedEmployee, "SALES")
	fixture.addSalesScheme(occupiedEmployee)
	occupiedVerification := fixture.addVerification(fixture.historicalDate)
	created, err := fixture.commissionUsecase().Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateCommissionInput{
		VerificationID: occupiedVerification,
		EmployeeID:     occupiedEmployee,
		PersonnelRole:  biz.CommissionRoleSales,
		IdempotencyKey: "fca-occupied-" + fixture.suffix,
	})
	if err != nil {
		t.Fatalf("预置占用 DRAFT 失败: %v", err)
	}
	// 直接写入一张更早月份的占位申请与明细，模拟该提成已被占用。
	fakeApplicationID := uuid.New()
	if _, err := fixture.data.db.FinanceCommissionApplication.Create().
		SetID(fakeApplicationID).
		SetOrganizationID(fixture.organizationID).
		SetEmployeeID(occupiedEmployee).
		SetApplicationMonth("2000-01").
		SetCoverageTo("1999-12-31").
		SetStatus(applicationent.StatusREJECTED).
		SetVersion(1).
		SetCommissionCount(1).
		SetBaseCurrency("CNY").
		SetTotalCommissionAmount("60.00000000").
		SetTotalCnyCommissionAmount("60.00000000").
		SetSubmittedAt(time.Now()).
		SetSubmittedBy(occupiedEmployee).
		Save(ctx); err != nil {
		t.Fatalf("写入占位申请失败: %v", err)
	}
	if _, err := fixture.data.db.FinanceCommissionApplicationLine.Create().
		SetOrganizationID(fixture.organizationID).
		SetEmployeeID(occupiedEmployee).
		SetApplicationID(fakeApplicationID).
		SetCommissionID(created.ID).
		SetCommissionDate(fixture.historicalDate).
		SetPersonnelRole("SALES").
		SetRuleVersion(1).
		SetBaseCurrency("CNY").
		SetCommissionAmount("60.00000000").
		SetCnyCommissionAmount("60.00000000").
		SetSourceFingerprint(strings.Repeat("b", 64)).
		Save(ctx); err != nil {
		t.Fatalf("写入占用明细失败: %v", err)
	}
	if _, err := usecase.Submit(ctx, fixture.scopeFor(occupiedEmployee)); !errors.Is(err, biz.ErrCommissionApplicationSourceConflict) {
		t.Fatalf("已占用 DRAFT 应返回来源冲突: %v", err)
	}

	// 场景三：仅有已取消提成记录的来源不得通过申请重新生成提成。
	// 使用独立夹具，保证被取消来源是该员工唯一可见来源。
	cancelledFixture := newCommissionApplicationFixture(t)
	cancelledEmployee := cancelledFixture.newEmployee("cancelled")
	cancelledFixture.addAttribution(cancelledEmployee, "SALES")
	cancelledFixture.addSalesScheme(cancelledEmployee)
	cancelledVerification := cancelledFixture.addVerification(cancelledFixture.historicalDate)
	createdCommission, err := cancelledFixture.commissionUsecase().Create(ctx, cancelledFixture.organizationID, cancelledFixture.actorID, biz.CreateCommissionInput{
		VerificationID: cancelledVerification,
		EmployeeID:     cancelledEmployee,
		PersonnelRole:  biz.CommissionRoleSales,
		IdempotencyKey: "fca-cancelled-" + cancelledFixture.suffix,
	})
	if err != nil {
		t.Fatalf("预置待取消 DRAFT 失败: %v", err)
	}
	if _, err := cancelledFixture.data.db.FinanceCommission.UpdateOneID(createdCommission.ID).
		SetStatus(commission.StatusCANCELLED).
		SetCancelledAt(time.Now()).
		SetCancelledBy(cancelledFixture.actorID).
		SetCancellationReason("测试取消").
		Save(ctx); err != nil {
		t.Fatalf("模拟财务取消提成失败: %v", err)
	}
	cancelledUsecase := cancelledFixture.usecase()
	cancelledScope := cancelledFixture.scopeFor(cancelledEmployee)
	cancelledCandidates, err := cancelledUsecase.ListMyCandidates(ctx, cancelledScope, biz.WorkbenchApplicationCandidateFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("查询已取消来源候选失败: %v", err)
	}
	if cancelledCandidates.Total != 0 {
		t.Fatalf("已取消来源不得重新成为候选: %d", cancelledCandidates.Total)
	}
	if _, err := cancelledUsecase.Submit(ctx, cancelledScope); !errors.Is(err, biz.ErrCommissionApplicationEmpty) {
		t.Fatalf("已取消来源不得通过申请重新生成提成: %v", err)
	}
}

// scopeForToday 构造指定财务业务日期的本人范围：跨月重提用例以显式日期模拟
// 「4 月提交、5 月重提」，不依赖真实时钟。
func (f *commissionApplicationFixture) scopeForToday(employeeID uuid.UUID, today string) biz.WorkbenchScope {
	return biz.WorkbenchScope{CanOperateBusiness: true,
		OrganizationID: f.organizationID,
		UserID:         employeeID,
		Today:          today,
		Now:            time.Now(),
	}
}

// TestCommissionApplicationResubmitRefreshPostgres 是显式重提刷新语义的跨月
// 回归（P0）：4 月提交 → 财务驳回 → 修正上游来源费用（指纹与金额变化）→
// 5 月以显式申请 ID 重提 → 断言明细快照与父单金额已按上游当前事实刷新、
// 明细集合不变、版本递增回待审、财务可再次批准；另覆盖全部明细失效与
// 他人/跨组织申请 ID 的稳定拒绝。
func TestCommissionApplicationResubmitRefreshPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()
	orgScope := []uuid.UUID{fixture.organizationID}

	// 员工 A：3 月来源、4 月提交、驳回后修正费用、5 月显式重提。
	employeeA := fixture.newEmployee("refresh")
	fixture.addAttribution(employeeA, "SALES")
	fixture.addSalesScheme(employeeA)
	fixture.addVerification("2026-03-20")
	scopeApril := fixture.scopeForToday(employeeA, "2026-04-10")
	application, err := usecase.Submit(ctx, scopeApril)
	if err != nil {
		t.Fatalf("4 月提交失败: %v", err)
	}
	if application.ApplicationMonth != "2026-04" || application.CoverageTo != "2026-03-31" {
		t.Fatalf("申请期间应为 2026-04 / 2026-03-31: %+v", application)
	}
	if application.CommissionCount != 1 || application.TotalCommissionAmount.StringFixed(8) != "60.00000000" {
		t.Fatalf("4 月申请应为 1 笔 60: %+v", application)
	}
	rejected, err := usecase.Reject(ctx, orgScope, fixture.actorID, application.ID, 1, "金额与来源不一致，整单驳回")
	if err != nil {
		t.Fatalf("财务整单驳回失败: %v", err)
	}
	if rejected.Status != biz.CommissionApplicationRejected || rejected.Version != 2 {
		t.Fatalf("驳回后应为 REJECTED v2: %+v", rejected)
	}

	// 修正上游来源事实：账单行本位币金额 1000 → 1100（已实现收入分子随账单行
	// 快照变化，指纹与提成金额同时漂移），提成应按 (1100-400)×10% = 70 刷新。
	billLine, err := fixture.data.db.FinanceBillLine.Query().
		Where(financebillline.BillIDEQ(fixture.billID)).Only(ctx)
	if err != nil {
		t.Fatalf("定位来源账单行失败: %v", err)
	}
	originalLine, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(application.ID)).Only(ctx)
	if err != nil {
		t.Fatalf("定位重提前明细失败: %v", err)
	}
	originalCommission, err := fixture.data.db.FinanceCommission.Query().
		Where(commission.IDEQ(originalLine.CommissionID)).Only(ctx)
	if err != nil {
		t.Fatalf("定位重提前提成事实失败: %v", err)
	}
	if _, err := fixture.data.db.FinanceBillLine.UpdateOneID(billLine.ID).
		SetBaseCurrencyAmount("1100.00000000").
		SetTotalAmount("1100.00000000").
		SetNetAmount("1100.00000000").
		Save(ctx); err != nil {
		t.Fatalf("修正上游账单行失败: %v", err)
	}

	// 5 月以显式申请 ID 重提：金额/指纹刷新、明细集合不变、版本递增回待审。
	scopeMay := fixture.scopeForToday(employeeA, "2026-05-10")
	resubmitted, err := usecase.Resubmit(ctx, scopeMay, application.ID, 2)
	if err != nil {
		t.Fatalf("5 月显式重提失败: %v", err)
	}
	if resubmitted.ID != application.ID || resubmitted.Status != biz.CommissionApplicationPendingReview || resubmitted.Version != 3 {
		t.Fatalf("重提应沿用原申请并回到待审 v3: %+v", resubmitted)
	}
	if resubmitted.ApplicationMonth != "2026-04" || resubmitted.CoverageTo != "2026-03-31" {
		t.Fatalf("重提不得改变申请期间: %+v", resubmitted)
	}
	if resubmitted.CommissionCount != 1 || resubmitted.TotalCommissionAmount.StringFixed(8) != "70.00000000" {
		t.Fatalf("重提后汇总应刷新为 1 笔 70: count=%d total=%s", resubmitted.CommissionCount, resubmitted.TotalCommissionAmount)
	}
	if resubmitted.DecisionReason == nil || *resubmitted.DecisionReason != "金额与来源不一致，整单驳回" {
		t.Fatalf("重提后员工仍应能看到最近一次驳回原因: %+v", resubmitted)
	}

	detail, err := usecase.GetMyApplication(ctx, scopeMay, application.ID)
	if err != nil {
		t.Fatalf("重读申请详情失败: %v", err)
	}
	if len(detail.Lines) != 1 {
		t.Fatalf("重提不得改变明细集合: %d", len(detail.Lines))
	}
	refreshedLine := detail.Lines[0]
	if refreshedLine.ID != originalLine.ID || refreshedLine.CommissionID != originalLine.CommissionID {
		t.Fatalf("重提应保持同一明细绑定: %+v vs %+v", refreshedLine, originalLine)
	}
	if refreshedLine.CommissionAmount.StringFixed(8) != "70.00000000" || refreshedLine.SourceFingerprint == originalLine.SourceFingerprint {
		t.Fatalf("明细快照金额/指纹应按上游当前事实刷新: %+v", refreshedLine)
	}
	refreshedCommission, err := fixture.data.db.FinanceCommission.Query().
		Where(commission.IDEQ(originalLine.CommissionID)).Only(ctx)
	if err != nil {
		t.Fatalf("重查提成事实失败: %v", err)
	}
	if refreshedCommission.Status != commission.StatusDRAFT || refreshedCommission.CommissionAmount != "70.00000000" {
		t.Fatalf("提成父单金额应刷新且保持 DRAFT: %+v", refreshedCommission)
	}
	if refreshedCommission.SourceFingerprint != refreshedLine.SourceFingerprint || refreshedCommission.Version != originalCommission.Version+1 {
		t.Fatalf("提成父单指纹/版本应随刷新递增: %+v", refreshedCommission)
	}
	commissionLines, err := fixture.data.db.FinanceCommissionLine.Query().
		Where(commissionline.CommissionIDEQ(originalLine.CommissionID)).All(ctx)
	if err != nil {
		t.Fatalf("查询提成行失败: %v", err)
	}
	if len(commissionLines) != 1 || commissionLines[0].CommissionAmount != "70.00000000" {
		t.Fatalf("提成行应随重提刷新为当前计算事实: %+v", commissionLines)
	}
	refreshAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ActionEQ("finance.commission.resubmit_refresh"), auditlog.ResourceIDEQ(originalLine.CommissionID.String())).
		Count(ctx)
	if err != nil {
		t.Fatalf("查询提成刷新审计失败: %v", err)
	}
	if refreshAudits != 1 {
		t.Fatalf("刷新提成应恰好一条刷新审计: %d", refreshAudits)
	}

	// 财务对刷新后的申请再次整单批准：指纹复核应通过。
	approved, err := usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 3)
	if err != nil {
		t.Fatalf("重提后财务再次批准失败: %v", err)
	}
	if approved.Status != biz.CommissionApplicationApproved || approved.Version != 4 {
		t.Fatalf("再次批准后应为 APPROVED v4: %+v", approved)
	}
	confirmedCommission, err := fixture.data.db.FinanceCommission.Query().
		Where(commission.IDEQ(originalLine.CommissionID)).Only(ctx)
	if err != nil {
		t.Fatalf("重查提成事实失败: %v", err)
	}
	if confirmedCommission.Status != commission.StatusCONFIRMED {
		t.Fatalf("再次批准应确认刷新后的提成: %+v", confirmedCommission)
	}

	// 员工 B：修正后的费用下提交 → 驳回 → 取消其全部提成 → 重提应因全部明细
	// 失效拒绝（B 与 A 共享订单归属，两张核销都是 B 的候选，明细为 2 笔）。
	employeeB := fixture.newEmployee("nolines")
	fixture.addAttribution(employeeB, "SALES")
	fixture.addSalesScheme(employeeB)
	fixture.addVerification("2026-03-25")
	scopeB := fixture.scopeForToday(employeeB, "2026-04-10")
	applicationB, err := usecase.Submit(ctx, scopeB)
	if err != nil {
		t.Fatalf("员工 B 提交失败: %v", err)
	}
	if _, err := usecase.Reject(ctx, orgScope, fixture.actorID, applicationB.ID, 1, "整单驳回"); err != nil {
		t.Fatalf("员工 B 申请驳回失败: %v", err)
	}
	bLines, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(applicationB.ID)).All(ctx)
	if err != nil {
		t.Fatalf("定位员工 B 明细失败: %v", err)
	}
	if len(bLines) == 0 {
		t.Fatalf("员工 B 申请应至少有一笔明细")
	}
	for _, item := range bLines {
		if _, err := fixture.data.db.FinanceCommission.UpdateOneID(item.CommissionID).
			SetStatus(commission.StatusCANCELLED).
			SetCancelledAt(time.Now()).
			SetCancelledBy(fixture.actorID).
			SetCancellationReason("测试取消").
			Save(ctx); err != nil {
			t.Fatalf("模拟取消提成失败: %v", err)
		}
	}
	if _, err := usecase.Resubmit(ctx, scopeB, applicationB.ID, 2); !errors.Is(err, biz.ErrCommissionApplicationNoValidLines) {
		t.Fatalf("全部明细失效时重提应稳定拒绝: %v", err)
	}
	headerB, err := fixture.data.db.FinanceCommissionApplication.Query().
		Where(applicationent.IDEQ(applicationB.ID)).Only(ctx)
	if err != nil {
		t.Fatalf("查询员工 B 申请头失败: %v", err)
	}
	if headerB.Status != applicationent.StatusREJECTED || headerB.Version != 2 {
		t.Fatalf("失败重提应整体回滚且申请保持 REJECTED: %+v", headerB)
	}

	// 他人与跨组织申请 ID 一律按不存在处理，不泄露记录事实。
	otherEmployee := fixture.newEmployee("outsider")
	if _, err := usecase.Resubmit(ctx, fixture.scopeForToday(otherEmployee, "2026-05-10"), applicationB.ID, 2); !errors.Is(err, biz.ErrCommissionApplicationNotFound) {
		t.Fatalf("重提他人申请应按不存在处理: %v", err)
	}
	if _, err := usecase.Resubmit(ctx, fixture.scopeForToday(employeeB, "2026-05-10"), applicationB.ID, 99); !errors.Is(err, biz.ErrCommissionApplicationStatusConflict) {
		t.Fatalf("过期版本重提应稳定拒绝: %v", err)
	}
	// 跨组织重提：员工在另一组织有合法成员关系（与服务端会话语义一致），
	// 目标组织内查无此申请，按不存在处理，不泄露记录事实。
	foreignOrg, err := fixture.data.db.Organization.Create().
		SetCode("FCA-FR-" + fixture.suffix).
		SetName("月度申请外部组织-" + fixture.suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建外部组织失败: %v", err)
	}
	if _, err := fixture.data.db.Membership.Create().
		SetOrganizationID(foreignOrg.ID).
		SetUserID(employeeB).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建外部组织成员资格失败: %v", err)
	}
	foreignScope := fixture.scopeForToday(employeeB, "2026-05-10")
	foreignScope.OrganizationID = foreignOrg.ID
	if _, err := usecase.Resubmit(ctx, foreignScope, applicationB.ID, 2); !errors.Is(err, biz.ErrCommissionApplicationNotFound) {
		t.Fatalf("跨组织重提应按不存在处理: %v", err)
	}
}
