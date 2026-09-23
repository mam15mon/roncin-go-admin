package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// 计提自动解析集成夹具：单一订单（应收 1000 / 应付 400）+ 一张 ACTIVE 核销单
// （归属日期 financeCommissionIntegrationDate），通过「员工 × 身份」归属与
// 「方案 × 员工分配」组合验证候选发现与唯一方案解析。
type commissionResolutionFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	customerID     uuid.UUID
	actorID        uuid.UUID
	orderID        uuid.UUID
	verificationID uuid.UUID
	suffix         string
}

func newCommissionResolutionFixture(t *testing.T) *commissionResolutionFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	org, err := data.db.Organization.Create().
		SetCode("COMM-RS-" + suffix).
		SetName("计提解析测试组织-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	actor, err := data.db.User.Create().
		SetUsername("comm_rs_actor_" + suffix).
		SetDisplayName("计提解析操作员-" + suffix).
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
		SetCode("CUST-RS-" + suffix).
		SetLegalName("计提解析客户-" + suffix).
		SetNormalizedName("计提解析客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	fixture := &commissionResolutionFixture{t: t, data: data, organizationID: org.ID, customerID: customer.ID, actorID: actor.ID, suffix: suffix}

	order, err := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(org.ID).
		SetOrderNo("RS-SE" + suffix).
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

	feeReceivable, err := data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("rs-fee-rec-" + suffix).
		SetDirection(fee.DirectionRECEIVABLE).
		SetStatus(fee.StatusBILLED).
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
		SetIdempotencyKey("rs-fee-pay-" + suffix).
		SetDirection(fee.DirectionPAYABLE).
		SetStatus(fee.StatusBILLED).
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
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("400.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx); err != nil {
		t.Fatalf("创建测试应付费用: %v", err)
	}

	billCreate := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("RS-BILL-" + suffix).
		SetIdempotencyKey("rs-bill-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetFeeCount(1).
		SetBillDate(financeCommissionIntegrationDate).
		SetVersion(1)
	billItem, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试账单: %v", err)
	}
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
		SetFlowNo("RS-FLOW-" + suffix).
		SetIdempotencyKey("rs-cashflow-" + suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetAmount("1000.00000000").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
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
	verificationItem, err := data.db.FinanceVerification.Create().
		SetOrganizationID(org.ID).
		SetVerificationNo("RS-VR-" + suffix).
		SetIdempotencyKey("rs-verification-" + suffix).
		SetDirection(verification.DirectionRECEIVABLE).
		SetStatus(verification.StatusACTIVE).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetAmount("1000.00000000").
		SetBaseCurrency("CNY").
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
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试核销分摊: %v", err)
	}

	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(org.ID).
		SetDocumentType(numberruleent.DocumentTypeCommission).
		SetPrefix("RS-TC-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试提成编号规则: %v", err)
	}
	return fixture
}

// newEmployee 创建启用员工与组织成员关系。
func (f *commissionResolutionFixture) newEmployee(label string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	employee, err := f.data.db.User.Create().
		SetUsername("comm_rs_" + label + "_" + f.suffix).
		SetDisplayName("解析员工-" + label + "-" + f.suffix).
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

// addAttribution 为员工追加订单提成归属。
func (f *commissionResolutionFixture) addAttribution(employeeID uuid.UUID, role string, employeeName string) {
	f.t.Helper()
	if _, err := f.data.db.OrderCommissionAttribution.Create().
		SetOrganizationID(f.organizationID).
		SetOrderID(f.orderID).
		SetCustomerID(f.customerID).
		SetSourceAssignmentID(uuid.New()).
		SetEmployeeID(employeeID).
		SetEmployeeName(employeeName).
		SetPersonnelRole(attribution.PersonnelRole(role)).
		SetAttributedAt(time.Now()).
		Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试提成归属: %v", err)
	}
}

// newRule 创建已启用方案并返回 ID。
func (f *commissionResolutionFixture) newRule(name string, role biz.CommissionPersonnelRole, ratePercent, effectiveFrom string, effectiveTo *string) uuid.UUID {
	f.t.Helper()
	create := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName(name + "-" + f.suffix).
		SetPersonnelRole(rule.PersonnelRole(role)).
		SetCalculationBasis(rule.CalculationBasisREALIZED_PROFIT).
		SetRatePercent(ratePercent).
		SetEffectiveFrom(effectiveFrom).
		SetEnabled(true).
		SetVersion(1)
	if effectiveTo != nil {
		create = create.SetEffectiveTo(*effectiveTo)
	}
	item, err := create.Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试提成方案: %v", err)
	}
	return item.ID
}

// assign 为员工追加未取消分配段。
func (f *commissionResolutionFixture) assign(ruleID, employeeID uuid.UUID, effectiveFrom string, effectiveTo *string) {
	f.t.Helper()
	create := f.data.db.FinanceCommissionRuleAssignment.Create().
		SetOrganizationID(f.organizationID).
		SetRuleID(ruleID).
		SetEmployeeID(employeeID).
		SetEffectiveFrom(effectiveFrom).
		SetCreatedBy(f.actorID)
	if effectiveTo != nil {
		create = create.SetEffectiveTo(*effectiveTo)
	}
	if _, err := create.Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试方案员工分配: %v", err)
	}
}

func (f *commissionResolutionFixture) usecase() *biz.CommissionUsecase {
	f.t.Helper()
	return biz.NewCommissionUsecase(
		NewCommissionRepo(f.data),
		biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)),
		f.data,
	)
}

func (f *commissionResolutionFixture) input(employeeID uuid.UUID, role biz.CommissionPersonnelRole, key string) biz.CreateCommissionInput {
	return biz.CreateCommissionInput{
		VerificationID: f.verificationID,
		EmployeeID:     employeeID,
		PersonnelRole:  role,
		IdempotencyKey: "rs-commission-" + key + "-" + f.suffix,
	}
}

func TestCommissionResolutionPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	t.Run("多员工共享方案与员工差异方案分别解析且固定薪无候选", func(t *testing.T) {
		fixture := newCommissionResolutionFixture(t)
		ctx := context.Background()
		usecase := fixture.usecase()

		employeeA := fixture.newEmployee("a")
		employeeB := fixture.newEmployee("b")
		employeeC := fixture.newEmployee("c")
		display := map[uuid.UUID]string{}
		for id, label := range map[uuid.UUID]string{employeeA: "a", employeeB: "b", employeeC: "c"} {
			display[id] = "解析员工-" + label + "-" + fixture.suffix
		}
		for _, id := range []uuid.UUID{employeeA, employeeB, employeeC} {
			fixture.addAttribution(id, "SALES", display[id])
		}
		// A 共享 10% 方案，B 独享 20% 差异方案，C 固定薪不参加任何方案。
		sharedRule := fixture.newRule("共享销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		diffRule := fixture.newRule("差异销售方案", biz.CommissionRoleSales, "20.0000", "2026-01-01", nil)
		fixture.assign(sharedRule, employeeA, "2026-01-01", nil)
		fixture.assign(diffRule, employeeB, "2026-01-01", nil)

		candidates, err := usecase.ListCandidates(ctx, fixture.organizationID, biz.CommissionCandidateFilter{
			Page: 1, PageSize: 200, VerificationID: fixture.verificationID,
		})
		if err != nil {
			t.Fatalf("查询计提候选失败: %v", err)
		}
		if candidates.Total != 2 || len(candidates.Items) != 2 {
			t.Fatalf("候选数量 = %d/%d，期望仅两名方案内员工", candidates.Total, len(candidates.Items))
		}
		type candidateSummary struct {
			rate   string
			amount string
			ruleID uuid.UUID
		}
		amountByEmployee := map[uuid.UUID]candidateSummary{}
		for _, item := range candidates.Items {
			if item.RuleID != sharedRule && item.RuleID != diffRule {
				t.Fatalf("候选命中未知方案: %+v", item)
			}
			amountByEmployee[item.EmployeeID] = candidateSummary{rate: item.RatePercent.StringFixed(4), amount: item.CommissionAmount.StringFixed(8), ruleID: item.RuleID}
		}
		if got, ok := amountByEmployee[employeeA]; !ok || got.rate != "10.0000" || got.amount != "60.00000000" || got.ruleID != sharedRule {
			t.Fatalf("员工 A 应按共享方案 10%% 计提 60: %+v", got)
		}
		if got, ok := amountByEmployee[employeeB]; !ok || got.rate != "20.0000" || got.amount != "120.00000000" || got.ruleID != diffRule {
			t.Fatalf("员工 B 应按差异方案 20%% 计提 120: %+v", got)
		}
		if _, exists := amountByEmployee[employeeC]; exists {
			t.Fatal("固定薪员工不应产生候选")
		}

		previewA, err := usecase.Preview(ctx, fixture.organizationID, fixture.verificationID, uuid.Nil, employeeA, biz.CommissionRoleSales)
		if err != nil {
			t.Fatalf("员工 A 预览失败: %v", err)
		}
		if previewA.RuleID != sharedRule || previewA.CommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("员工 A 预览应命中共享方案: rule=%s amount=%s", previewA.RuleID, previewA.CommissionAmount)
		}
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input(employeeB, biz.CommissionRoleSales, "diff"))
		if err != nil {
			t.Fatalf("员工 B 创建提成失败: %v", err)
		}
		if created.RuleID != diffRule || created.CommissionAmount.StringFixed(8) != "120.00000000" {
			t.Fatalf("员工 B 创建应固化差异方案与金额: rule=%v amount=%s", created.RuleID, created.CommissionAmount)
		}
	})

	t.Run("同一员工多身份按各自方案形成独立候选与提成", func(t *testing.T) {
		fixture := newCommissionResolutionFixture(t)
		ctx := context.Background()
		usecase := fixture.usecase()

		employee := fixture.newEmployee("multi")
		display := "解析员工-multi-" + fixture.suffix
		fixture.addAttribution(employee, "SALES", display)
		fixture.addAttribution(employee, "OPERATOR", display)
		salesRule := fixture.newRule("销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		operatorRule := fixture.newRule("操作方案", biz.CommissionRoleOperator, "5.0000", "2026-01-01", nil)
		fixture.assign(salesRule, employee, "2026-01-01", nil)
		fixture.assign(operatorRule, employee, "2026-01-01", nil)

		candidates, err := usecase.ListCandidates(ctx, fixture.organizationID, biz.CommissionCandidateFilter{
			Page: 1, PageSize: 200, VerificationID: fixture.verificationID,
		})
		if err != nil {
			t.Fatalf("查询计提候选失败: %v", err)
		}
		if candidates.Total != 2 || len(candidates.Items) != 2 {
			t.Fatalf("同一员工多身份候选数 = %d/%d，期望 2", candidates.Total, len(candidates.Items))
		}
		amountByRole := map[biz.CommissionPersonnelRole]string{}
		for _, item := range candidates.Items {
			if item.EmployeeID != employee {
				t.Fatalf("候选员工不符: %+v", item)
			}
			amountByRole[item.PersonnelRole] = item.CommissionAmount.StringFixed(8)
		}
		if amountByRole[biz.CommissionRoleSales] != "60.00000000" || amountByRole[biz.CommissionRoleOperator] != "30.00000000" {
			t.Fatalf("多身份候选金额不符: %+v", amountByRole)
		}

		if _, err = usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input(employee, biz.CommissionRoleSales, "multi-sales")); err != nil {
			t.Fatalf("销售身份创建提成失败: %v", err)
		}
		operatorCreated, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input(employee, biz.CommissionRoleOperator, "multi-operator"))
		if err != nil {
			t.Fatalf("操作身份创建提成失败: %v", err)
		}
		if operatorCreated.RuleID != operatorRule || operatorCreated.CommissionAmount.StringFixed(8) != "30.00000000" {
			t.Fatalf("操作身份提成应命中操作方案: rule=%v amount=%s", operatorCreated.RuleID, operatorCreated.CommissionAmount)
		}
	})

	t.Run("名单加入退出日期边界决定候选资格", func(t *testing.T) {
		fixture := newCommissionResolutionFixture(t)
		ctx := context.Background()
		usecase := fixture.usecase()

		joinedSameDay := fixture.newEmployee("join")
		joinedLater := fixture.newEmployee("later")
		exitedBefore := fixture.newEmployee("exit")
		for id, label := range map[uuid.UUID]string{joinedSameDay: "join", joinedLater: "later", exitedBefore: "exit"} {
			fixture.addAttribution(id, "SALES", "解析员工-"+label+"-"+fixture.suffix)
		}
		ruleID := fixture.newRule("边界方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		fixture.assign(ruleID, joinedSameDay, financeCommissionIntegrationDate, nil)
		fixture.assign(ruleID, joinedLater, "2026-08-31", nil)
		fixture.assign(ruleID, exitedBefore, "2026-01-01", stringPtr("2026-08-29"))

		candidates, err := usecase.ListCandidates(ctx, fixture.organizationID, biz.CommissionCandidateFilter{
			Page: 1, PageSize: 200, VerificationID: fixture.verificationID,
		})
		if err != nil {
			t.Fatalf("查询计提候选失败: %v", err)
		}
		if candidates.Total != 1 || len(candidates.Items) != 1 {
			t.Fatalf("边界候选数 = %d/%d，期望仅来源日当天加入的员工", candidates.Total, len(candidates.Items))
		}
		if candidates.Items[0].EmployeeID != joinedSameDay {
			t.Fatalf("应只有来源日当天加入分配的员工成为候选: %+v", candidates.Items[0])
		}

		if _, err = usecase.Preview(ctx, fixture.organizationID, fixture.verificationID, uuid.Nil, joinedLater, biz.CommissionRoleSales); !errors.Is(err, biz.ErrCommissionRuleNotResolved) {
			t.Fatalf("生效日之后的分配不应解析出方案: %v", err)
		}
	})

	t.Run("离职员工来源日期落在历史分配期间仍可生成应得提成", func(t *testing.T) {
		fixture := newCommissionResolutionFixture(t)
		ctx := context.Background()
		usecase := fixture.usecase()

		departed := fixture.newEmployee("departed")
		fixture.addAttribution(departed, "SALES", "解析员工-departed-"+fixture.suffix)
		ruleID := fixture.newRule("离职衔接方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		fixture.assign(ruleID, departed, "2026-01-01", stringPtr("2026-09-01"))

		// 员工离职：停用成员关系与登录账号，不以当前状态抹除历史资格。
		if _, err := fixture.data.db.Membership.Delete().Where(membership.UserIDEQ(departed), membership.OrganizationIDEQ(fixture.organizationID)).Exec(ctx); err != nil {
			t.Fatalf("准备离职成员关系失败: %v", err)
		}
		if _, err := fixture.data.db.User.UpdateOneID(departed).SetEnabled(false).Save(ctx); err != nil {
			t.Fatalf("停用离职账号失败: %v", err)
		}

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input(departed, biz.CommissionRoleSales, "departed"))
		if err != nil {
			t.Fatalf("离职员工历史资格创建提成失败: %v", err)
		}
		if created.RuleID != ruleID || created.CommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("离职员工提成应命中历史方案: rule=%v amount=%s", created.RuleID, created.CommissionAmount)
		}
	})

	t.Run("已生效方案复制衔接后历史来源仍命中原方案", func(t *testing.T) {
		fixture := newCommissionResolutionFixture(t)
		ctx := context.Background()
		usecase := fixture.usecase()

		employee := fixture.newEmployee("copy")
		fixture.addAttribution(employee, "SALES", "解析员工-copy-"+fixture.suffix)
		oldRule := fixture.newRule("复制前方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		fixture.assign(oldRule, employee, "2026-01-01", nil)

		today := biz.FinanceBusinessDate(time.Now())
		newRuleItem, copyErr := usecase.CopyRule(ctx, fixture.organizationID, fixture.actorID, biz.CopyCommissionRuleInput{
			SourceRuleID: oldRule, Name: "复制后方案-" + fixture.suffix,
			PersonnelRole: biz.CommissionRoleSales, CalculationBasis: biz.CommissionBasisRealizedProfit,
			RatePercent: decimal.RequireFromString("15.0000"), EffectiveFrom: today, EmployeeIDs: []uuid.UUID{employee},
		})
		if copyErr != nil {
			t.Fatalf("复制为新方案失败: %v", copyErr)
		}
		if newRuleItem.ID == uuid.Nil || newRuleItem.EffectiveFrom == nil || *newRuleItem.EffectiveFrom != today {
			t.Fatalf("新方案应以当天日期生效: %+v", newRuleItem)
		}
		oldAfterCopy, err := usecase.GetRuleScoped(ctx, []uuid.UUID{fixture.organizationID}, oldRule)
		if err != nil {
			t.Fatalf("重读原方案失败: %v", err)
		}
		if oldAfterCopy.EffectiveTo == nil || *oldAfterCopy.EffectiveTo != biz.FinanceDateBefore(today) {
			t.Fatalf("原方案终止日应衔接为新方案生效日前一日: %+v", oldAfterCopy.EffectiveTo)
		}

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input(employee, biz.CommissionRoleSales, "copy"))
		if err != nil {
			t.Fatalf("复制衔接后创建提成失败: %v", err)
		}
		if created.RuleID != oldRule || created.CommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("历史来源应仍命中原方案并按原比例计提: rule=%v amount=%s", created.RuleID, created.CommissionAmount)
		}
	})

	t.Run("名单与方案后续变更不阻断历史快照确认", func(t *testing.T) {
		fixture := newCommissionResolutionFixture(t)
		ctx := context.Background()
		usecase := fixture.usecase()

		employee := fixture.newEmployee("snapshot")
		fixture.addAttribution(employee, "SALES", "解析员工-snapshot-"+fixture.suffix)
		ruleID := fixture.newRule("快照稳定方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		fixture.assign(ruleID, employee, "2026-01-01", nil)

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input(employee, biz.CommissionRoleSales, "snapshot"))
		if err != nil {
			t.Fatalf("创建历史提成草稿失败: %v", err)
		}

		// 以当天日期终止分配（生效于未来），来源日期仍在分配的实际有效期间内：
		// 历史快照确认不应被名单后续变更阻断。
		today := biz.FinanceBusinessDate(time.Now())
		if _, err = usecase.RemoveRuleEmployees(ctx, fixture.organizationID, fixture.actorID, biz.CommissionRuleEmployeeChange{
			RuleID: ruleID, EmployeeIDs: []uuid.UUID{employee}, ExpectedVersion: 1, ChangeEffectiveDate: today,
		}); err != nil {
			t.Fatalf("以当天日期移除名单失败: %v", err)
		}
		confirmed, err := usecase.Confirm(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version)
		if err != nil {
			t.Fatalf("名单变更后确认历史快照失败: %v", err)
		}
		if confirmed.Status != biz.CommissionConfirmed || confirmed.CommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("确认后快照不符: status=%s amount=%s", confirmed.Status, confirmed.CommissionAmount)
		}
	})
}
