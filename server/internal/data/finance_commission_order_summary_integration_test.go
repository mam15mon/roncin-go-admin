package data

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	ruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	verificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// 订单列表提成摘要集成夹具：多组织、多订单、多员工与多身份的真实数据矩阵，
// 验证普通员工仅本人、组织级财务整票、同一主体跨组织权限差异、预计机会与
// 取消记录语义。
type orderSummaryFixture struct {
	t    *testing.T
	data *Data
}

type orderSummaryOrg struct {
	organizationID uuid.UUID
	customerID     uuid.UUID
}

func newOrderSummaryFixture(t *testing.T) *orderSummaryFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	return &orderSummaryFixture{t: t, data: data}
}

func (f *orderSummaryFixture) suffix() string {
	f.t.Helper()
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
}

// newOrg 创建测试组织、客户与提成编号规则。
func (f *orderSummaryFixture) newOrg(label, suffix string) *orderSummaryOrg {
	f.t.Helper()
	ctx := context.Background()
	org, err := f.data.db.Organization.Create().
		SetCode("OS-" + label + "-" + suffix).
		SetName("订单摘要测试组织-" + label + "-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试组织 %s: %v", label, err)
	}
	customer, err := f.data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("OS-CUST-" + label + "-" + suffix).
		SetLegalName("订单摘要客户-" + label + "-" + suffix).
		SetNormalizedName("订单摘要客户-" + label + "-" + suffix).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试客户 %s: %v", label, err)
	}
	if _, err = f.data.db.NumberRule.Create().
		SetOrganizationID(org.ID).
		SetDocumentType(numberrule.DocumentTypeCommission).
		SetPrefix("OS-TC-" + label + "-").
		SetDateFormat(numberrule.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberrule.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试提成编号规则 %s: %v", label, err)
	}
	return &orderSummaryOrg{organizationID: org.ID, customerID: customer.ID}
}

// newEmployee 创建启用员工与组织成员关系。
func (f *orderSummaryFixture) newEmployee(org *orderSummaryOrg, label, suffix string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	employee, err := f.data.db.User.Create().
		SetUsername("os_" + label + "_" + suffix).
		SetDisplayName("摘要员工-" + label + "-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试员工 %s: %v", label, err)
	}
	if _, err = f.data.db.Membership.Create().SetOrganizationID(org.organizationID).SetUserID(employee.ID).SetEnabled(true).Save(ctx); err != nil {
		f.t.Fatalf("创建测试员工 %s 成员资格: %v", label, err)
	}
	return employee.ID
}

// newOrderWithSource 创建海运出口订单：应收 1000 / 应付 400 已确认费用、
// CONFIRMED 应收账单（有效行 1000）与 ACTIVE 核销来源（归属日期 2026-08-30，
// 已实现分摊 1000），返回订单、账单与核销单标识。
func (f *orderSummaryFixture) newOrderWithSource(org *orderSummaryOrg, label, suffix string) (orderID, billID uuid.UUID, billNo string, verificationID uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	order, err := f.data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(org.organizationID).
		SetOrderNo("OS-" + strings.ToUpper(label) + suffix).
		SetCustomerID(org.customerID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetOrderDate(financeCommissionIntegrationDate).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试订单 %s: %v", label, err)
	}
	feeReceivable, err := f.data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("os-fee-rec-" + label + "-" + suffix).
		SetDirection(feeent.DirectionRECEIVABLE).
		SetStatus(feeent.StatusCONFIRMED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(org.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("1000.0000").
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("1000.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试应收费用 %s: %v", label, err)
	}
	if _, err = f.data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("os-fee-pay-" + label + "-" + suffix).
		SetDirection(feeent.DirectionPAYABLE).
		SetStatus(feeent.StatusCONFIRMED).
		SetFeeCode("COST").
		SetFeeName("成本费").
		SetSettlementPartyID(org.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("400.0000").
		SetTotalAmount("400.00000000").
		SetNetAmount("400.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("400.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试应付费用 %s: %v", label, err)
	}
	billCreate := f.data.db.FinanceBill.Create().
		SetOrganizationID(org.organizationID).
		SetBillNo("OS-BILL-" + label + "-" + suffix).
		SetIdempotencyKey("os-bill-" + label + "-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(org.customerID).
		SetSettlementPartyName("订单摘要客户-" + suffix).
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
		f.t.Fatalf("创建测试账单 %s: %v", label, err)
	}
	if _, err = f.data.db.FinanceBillLine.Create().
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
		f.t.Fatalf("创建测试账单明细 %s: %v", label, err)
	}
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(org.organizationID).
		SetFlowNo("OS-FLOW-" + label + "-v1-" + suffix).
		SetIdempotencyKey("os-cashflow-" + label + "-v1-" + suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(org.customerID).
		SetSettlementPartyName("订单摘要客户-" + suffix).
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
		f.t.Fatalf("创建测试资金流水 %s: %v", label, err)
	}
	verificationItem, err := f.data.db.FinanceVerification.Create().
		SetOrganizationID(org.organizationID).
		SetVerificationNo("OS-VR-" + label + "-v1-" + suffix).
		SetIdempotencyKey("os-verification-" + label + "-v1-" + suffix).
		SetDirection(verificationent.DirectionRECEIVABLE).
		SetStatus(verificationent.StatusACTIVE).
		SetSettlementPartyID(org.customerID).
		SetSettlementPartyName("订单摘要客户-" + suffix).
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
		f.t.Fatalf("创建测试核销单 %s: %v", label, err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().
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
		f.t.Fatalf("创建测试核销分摊 %s: %v", label, err)
	}
	return order.ID, billItem.ID, billItem.BillNo, verificationItem.ID
}

// addVerificationSource 为订单追加第二个完整核销来源：独立应收费用 600 +
// CONFIRMED 应收账单 + 资金流水 + ACTIVE 核销（已实现分摊 600），用于验证
// 同一订单多来源的预计机会计数。
func (f *orderSummaryFixture) addVerificationSource(org *orderSummaryOrg, orderID uuid.UUID, orderNo, label, suffix string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	feeReceivable, err := f.data.db.OrderFee.Create().
		SetOrderID(orderID).
		SetIdempotencyKey("os-fee2-" + label + "-" + suffix).
		SetDirection(feeent.DirectionRECEIVABLE).
		SetStatus(feeent.StatusCONFIRMED).
		SetFeeCode("DOCUMENTATION").
		SetFeeName("文件费").
		SetSettlementPartyID(org.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("600.0000").
		SetTotalAmount("600.00000000").
		SetNetAmount("600.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("600.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建第二笔应收费用 %s: %v", label, err)
	}
	billCreate := f.data.db.FinanceBill.Create().
		SetOrganizationID(org.organizationID).
		SetBillNo("OS-BILL-" + label + "-" + suffix).
		SetIdempotencyKey("os-bill-" + label + "-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(org.customerID).
		SetSettlementPartyName("订单摘要客户-" + suffix).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetTotalAmount("600.00000000").
		SetNetAmount("600.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("600.00000000").
		SetFeeCount(1).
		SetBillDate(financeCommissionIntegrationDate).
		SetVersion(1)
	billItem, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		f.t.Fatalf("创建第二张测试账单 %s: %v", label, err)
	}
	if _, err = f.data.db.FinanceBillLine.Create().
		SetBillID(billItem.ID).
		SetOrderID(orderID).
		SetOrderFeeID(feeReceivable.ID).
		SetOrderNo(orderNo).
		SetFeeCode(feeReceivable.FeeCode).
		SetFeeName(feeReceivable.FeeName).
		SetQuantity("1.0000").
		SetUnitPrice("600.0000").
		SetTotalAmount("600.00000000").
		SetNetAmount("600.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrencyAmount("600.00000000").
		SetBaseCurrency("CNY").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建第二张账单明细 %s: %v", label, err)
	}
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(org.organizationID).
		SetFlowNo("OS-FLOW-" + label + "-" + suffix).
		SetIdempotencyKey("os-cashflow-" + label + "-" + suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(org.customerID).
		SetSettlementPartyName("订单摘要客户-" + suffix).
		SetCurrency("CNY").
		SetAmount("600.00000000").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("600.00000000").
		SetTransactionDate(financeCommissionIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试资金流水 %s: %v", label, err)
	}
	verificationItem, err := f.data.db.FinanceVerification.Create().
		SetOrganizationID(org.organizationID).
		SetVerificationNo("OS-VR-" + label + "-" + suffix).
		SetIdempotencyKey("os-verification-" + label + "-" + suffix).
		SetDirection(verificationent.DirectionRECEIVABLE).
		SetStatus(verificationent.StatusACTIVE).
		SetSettlementPartyID(org.customerID).
		SetSettlementPartyName("订单摘要客户-" + suffix).
		SetCurrency("CNY").
		SetAmount("600.00000000").
		SetBaseCurrency("CNY").
		SetBaseAmount("600.00000000").
		SetBillBaseAmount("600.00000000").
		SetCashflowBaseAmount("600.00000000").
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试核销单 %s: %v", label, err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationItem.ID).
		SetCashflowID(cashflow.ID).
		SetBillID(billItem.ID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo(billItem.BillNo).
		SetAmount("600.00000000").
		SetBillBaseAmount("600.00000000").
		SetCashflowBaseAmount("600.00000000").
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试核销分摊 %s: %v", label, err)
	}
	return verificationItem.ID
}

func (f *orderSummaryFixture) addAttribution(org *orderSummaryOrg, orderID, employeeID uuid.UUID, role, employeeName string) {
	f.t.Helper()
	if _, err := f.data.db.OrderCommissionAttribution.Create().
		SetOrganizationID(org.organizationID).
		SetOrderID(orderID).
		SetCustomerID(org.customerID).
		SetSourceAssignmentID(uuid.New()).
		SetEmployeeID(employeeID).
		SetEmployeeName(employeeName).
		SetPersonnelRole(attributionent.PersonnelRole(role)).
		SetAttributedAt(time.Now()).
		Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试提成归属: %v", err)
	}
}

func (f *orderSummaryFixture) newRule(org *orderSummaryOrg, name string, role biz.CommissionPersonnelRole, ratePercent, effectiveFrom string, effectiveTo *string, suffix string) uuid.UUID {
	f.t.Helper()
	create := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(org.organizationID).
		SetName(name + "-" + suffix).
		SetPersonnelRole(ruleent.PersonnelRole(role)).
		SetCalculationBasis(ruleent.CalculationBasisREALIZED_PROFIT).
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

func (f *orderSummaryFixture) assign(org *orderSummaryOrg, ruleID, employeeID uuid.UUID, effectiveFrom string, effectiveTo *string, actorID uuid.UUID) {
	f.t.Helper()
	create := f.data.db.FinanceCommissionRuleAssignment.Create().
		SetOrganizationID(org.organizationID).
		SetRuleID(ruleID).
		SetEmployeeID(employeeID).
		SetEffectiveFrom(effectiveFrom).
		SetCreatedBy(actorID)
	if effectiveTo != nil {
		create = create.SetEffectiveTo(*effectiveTo)
	}
	if _, err := create.Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试方案员工分配: %v", err)
	}
}

func (f *orderSummaryFixture) usecase() *biz.CommissionUsecase {
	f.t.Helper()
	return biz.NewCommissionUsecase(
		NewCommissionRepo(f.data),
		biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)),
		f.data,
	)
}

func (f *orderSummaryFixture) summaries(t *testing.T, principal *biz.Principal, targets ...biz.OrderCommissionSummaryTarget) map[uuid.UUID]*biz.OrderCommissionSummary {
	t.Helper()
	result, err := f.usecase().BuildOrderListSummaries(context.Background(), principal, targets)
	if err != nil {
		t.Fatalf("构建订单提成摘要失败: %v", err)
	}
	return result
}

func assertSummaryFacts(t *testing.T, label string, summary *biz.OrderCommissionSummary, expectVisibility biz.OrderCommissionSummaryVisibility, expectOpportunityCount int, expectDraft, expectConfirmed, expectPaid, expectDecrease int, expectDraftAmount, expectConfirmedAmount, expectPaidAmount, expectDecreaseAmount string) {
	t.Helper()
	if summary == nil {
		t.Fatalf("%s: 摘要缺失", label)
	}
	if summary.Visibility != expectVisibility {
		t.Fatalf("%s: 可见模式 = %s，期望 %s", label, summary.Visibility, expectVisibility)
	}
	if summary.ExpectedOpportunityCount != expectOpportunityCount || summary.HasExpectedOpportunity != (expectOpportunityCount > 0) {
		t.Fatalf("%s: 预计机会 = %d/%t，期望 %d", label, summary.ExpectedOpportunityCount, summary.HasExpectedOpportunity, expectOpportunityCount)
	}
	for _, bucket := range []struct {
		name    string
		has     bool
		count   int
		amount  decimal.Decimal
		eCount  int
		eAmount string
	}{
		{"待确认草稿", summary.HasDraftCommission, summary.DraftCommissionCount, summary.DraftCommissionAmount, expectDraft, expectDraftAmount},
		{"已确认待发", summary.HasConfirmedCommission, summary.ConfirmedCommissionCount, summary.ConfirmedCommissionAmount, expectConfirmed, expectConfirmedAmount},
		{"已发放", summary.HasPaidCommission, summary.PaidCommissionCount, summary.PaidCommissionAmount, expectPaid, expectPaidAmount},
		{"待处理冲减", summary.HasPendingDecrease, summary.PendingDecreaseCount, summary.PendingDecreaseAmount, expectDecrease, expectDecreaseAmount},
	} {
		if bucket.count != bucket.eCount || bucket.has != (bucket.eCount > 0) {
			t.Fatalf("%s: %s 数量 = %d/%t，期望 %d", label, bucket.name, bucket.count, bucket.has, bucket.eCount)
		}
		if bucket.eAmount == "" {
			if !bucket.amount.IsZero() {
				t.Fatalf("%s: %s 金额应缺席: %s", label, bucket.name, bucket.amount)
			}
			continue
		}
		if !bucket.amount.Equal(decimal.RequireFromString(bucket.eAmount)) {
			t.Fatalf("%s: %s 金额 = %s，期望 %s", label, bucket.name, bucket.amount, bucket.eAmount)
		}
	}
}

func newOrderSummaryEmployeePrincipal(userID, currentOrganizationID uuid.UUID) *biz.Principal {
	return &biz.Principal{UserID: userID, Organization: biz.Organization{ID: currentOrganizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: currentOrganizationID, Kind: biz.OrganizationKindCompany}}}
}

func newOrderSummaryFinancePrincipal(userID, currentOrganizationID, readableOrganizationID uuid.UUID) *biz.Principal {
	return &biz.Principal{
		UserID:            userID,
		Organization:      biz.Organization{ID: currentOrganizationID, Kind: biz.OrganizationKindCompany},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: currentOrganizationID, Kind: biz.OrganizationKindCompany}, {ID: readableOrganizationID, Kind: biz.OrganizationKindCompany}},
		RoleGrants:        []biz.RoleGrant{{RoleCode: "commission-reader", DataScope: biz.DataScopeOrganizationTree, Permissions: map[string]struct{}{access.FinanceCommissionRead: {}}}},
	}
}

func TestOrderCommissionSummaryPrivacyPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	f := newOrderSummaryFixture(t)
	ctx := context.Background()
	usecase := f.usecase()
	suffix := f.suffix()

	// 组织 A 与方案目录：销售 10%、操作 5%。
	org := f.newOrg("a", suffix)
	actorID := f.newEmployee(org, "actor", suffix)
	salesRuleID := f.newRule(org, "销售摘要方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil, suffix)
	operatorRuleID := f.newRule(org, "操作摘要方案", biz.CommissionRoleOperator, "5.0000", "2026-01-01", nil, suffix)

	targetOf := func(orderID uuid.UUID) biz.OrderCommissionSummaryTarget {
		return biz.OrderCommissionSummaryTarget{OrderID: orderID, OrganizationID: org.organizationID}
	}

	// ── 准备订单 1「self」：调用者本人与同事各有销售归属，无方案分配。──
	selfUserID := f.newEmployee(org, "self", suffix)
	otherUserID := f.newEmployee(org, "other", suffix)
	orderSelfID, _, _, sourceSelfID := f.newOrderWithSource(org, "self", suffix)
	f.addAttribution(org, orderSelfID, selfUserID, "SALES", "摘要员工-self-"+suffix)
	f.addAttribution(org, orderSelfID, otherUserID, "SALES", "摘要员工-other-"+suffix)

	// ── 准备订单 2「mix」：三名员工、两种身份，验证多状态与冲减并存。──
	mixSalesID := f.newEmployee(org, "mixsales", suffix)
	mixOperatorID := f.newEmployee(org, "mixoperator", suffix)
	mixPaidID := f.newEmployee(org, "mixpaid", suffix)
	orderMixID, _, _, sourceMixID := f.newOrderWithSource(org, "mix", suffix)
	f.addAttribution(org, orderMixID, mixSalesID, "SALES", "摘要员工-mixsales-"+suffix)
	f.addAttribution(org, orderMixID, mixOperatorID, "OPERATOR", "摘要员工-mixoperator-"+suffix)
	f.addAttribution(org, orderMixID, mixPaidID, "SALES", "摘要员工-mixpaid-"+suffix)
	f.assign(org, salesRuleID, mixSalesID, "2026-01-01", nil, actorID)
	f.assign(org, operatorRuleID, mixOperatorID, "2026-01-01", nil, actorID)
	f.assign(org, salesRuleID, mixPaidID, "2026-01-01", nil, actorID)

	// ── 准备订单 3「opp」：唯一命中方案的员工 + 双核销来源，验证预计机会。──
	oppEmployeeID := f.newEmployee(org, "opp", suffix)
	orderOppID, _, _, sourceOppV1 := f.newOrderWithSource(org, "opp", suffix)
	f.addVerificationSource(org, orderOppID, "OS-OPP"+suffix, "opp-v2", suffix)
	f.addAttribution(org, orderOppID, oppEmployeeID, "SALES", "摘要员工-opp-"+suffix)
	f.assign(org, salesRuleID, oppEmployeeID, "2026-01-01", nil, actorID)

	// ── 准备组织 B：同一调用者成员身份 + 同事，验证跨组织权限差异。──
	orgB := f.newOrg("b", suffix)
	orderBID, _, _, sourceBID := f.newOrderWithSource(orgB, "cross", suffix)
	if _, err := f.data.db.Membership.Create().SetOrganizationID(orgB.organizationID).SetUserID(selfUserID).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建调用者组织 B 成员资格: %v", err)
	}
	colleagueBID := f.newEmployee(orgB, "colb", suffix)
	f.addAttribution(orgB, orderBID, selfUserID, "SALES", "摘要员工-self-"+suffix)
	f.addAttribution(orgB, orderBID, colleagueBID, "SALES", "摘要员工-colb-"+suffix)
	salesRuleBID := f.newRule(orgB, "跨组织销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil, suffix)
	f.assign(orgB, salesRuleBID, selfUserID, "2026-01-01", nil, actorID)
	f.assign(orgB, salesRuleBID, colleagueBID, "2026-01-01", nil, actorID)

	// 组织 A 调用者与同事也加入销售方案：本人无提成记录时仍应看到自己的预计
	// 机会，同时同事的草稿事实不可见。
	f.assign(org, salesRuleID, selfUserID, "2026-01-01", nil, actorID)
	f.assign(org, salesRuleID, otherUserID, "2026-01-01", nil, actorID)

	employeePrincipal := newOrderSummaryEmployeePrincipal(selfUserID, org.organizationID)
	// 财务主体与调用者是同一人（selfUserID）：在组织 A 持有提成读取权限得到
	// 组织级视图，在组织 B 退回仅本人视图。
	financePrincipal := newOrderSummaryFinancePrincipal(selfUserID, org.organizationID, org.organizationID)

	t.Run("普通员工仅本人且同事有记录时无泄露", func(t *testing.T) {
		// 同事先有一张 DRAFT 提成；本人视图必须完全不感知。
		if _, err := usecase.Create(ctx, org.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceSelfID, EmployeeID: otherUserID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-colleague-" + suffix,
		}); err != nil {
			t.Fatalf("创建同事提成草稿失败: %v", err)
		}
		// 本人尚无提成记录：只能看到本人的预计机会（方案已命中、来源尚未生成
		// 本人基础提成单），同事的草稿事实与金额不得出现。
		summary := f.summaries(t, employeePrincipal, targetOf(orderSelfID))[orderSelfID]
		assertSummaryFacts(t, "本人无记录", summary, biz.OrderCommissionVisibilityEmployee, 1, 0, 0, 0, 0, "", "", "", "")
		if summary.BaseCurrency != "" {
			t.Fatalf("本人无记录时不应携带金额币种: %s", summary.BaseCurrency)
		}

		// 本人随后生成自己的 DRAFT 提成：草稿事实出现，数量与金额仅含本人，
		// 该来源不再是本人的预计机会。
		if _, err := usecase.Create(ctx, org.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceSelfID, EmployeeID: selfUserID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-self-" + suffix,
		}); err != nil {
			t.Fatalf("创建本人提成草稿失败: %v", err)
		}
		assertSummaryFacts(t, "本人有草稿", f.summaries(t, employeePrincipal, targetOf(orderSelfID))[orderSelfID],
			biz.OrderCommissionVisibilityEmployee, 0, 1, 0, 0, 0, "60.00000000", "", "", "")

		// 同一张订单在组织级视图下可见两名员工的草稿（交叉验证数据真实存在）。
		assertSummaryFacts(t, "组织级对照", f.summaries(t, financePrincipal, targetOf(orderSelfID))[orderSelfID],
			biz.OrderCommissionVisibilityOrganization, 0, 2, 0, 0, 0, "120.00000000", "", "", "")
	})

	t.Run("组织级财务整票多员工多身份多状态并存", func(t *testing.T) {
		if _, err := usecase.Create(ctx, org.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceMixID, EmployeeID: mixSalesID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-mix-draft-" + suffix,
		}); err != nil {
			t.Fatalf("创建销售草稿失败: %v", err)
		}
		confirmedCommission, err := usecase.Create(ctx, org.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceMixID, EmployeeID: mixOperatorID, PersonnelRole: biz.CommissionRoleOperator, IdempotencyKey: "os-mix-conf-" + suffix,
		})
		if err != nil {
			t.Fatalf("创建操作提成失败: %v", err)
		}
		if _, err = usecase.Confirm(ctx, org.organizationID, actorID, confirmedCommission.ID, confirmedCommission.Version); err != nil {
			t.Fatalf("确认操作提成失败: %v", err)
		}
		paidCommission, err := usecase.Create(ctx, org.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceMixID, EmployeeID: mixPaidID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-mix-paid-" + suffix,
		})
		if err != nil {
			t.Fatalf("创建已发提成失败: %v", err)
		}
		confirmedPaid, err := usecase.Confirm(ctx, org.organizationID, actorID, paidCommission.ID, paidCommission.Version)
		if err != nil {
			t.Fatalf("确认已发提成失败: %v", err)
		}
		if _, err = usecase.MarkPaid(ctx, org.organizationID, actorID, paidCommission.ID, confirmedPaid.Version); err != nil {
			t.Fatalf("标记已发失败: %v", err)
		}
		// 操作提成的 DRAFT 冲减建议：待处理、尚未扣回。
		decrease, err := usecase.CreateAdjustment(ctx, org.organizationID, actorID, biz.CreateCommissionAdjustmentInput{
			CommissionID: confirmedCommission.ID, OrderID: orderMixID, Direction: biz.CommissionAdjustmentDecrease,
			Amount: decimal.RequireFromString("10.00000000"), Reason: "锁后补录冲减", IdempotencyKey: "os-mix-dec-" + suffix,
		})
		if err != nil {
			t.Fatalf("创建冲减建议失败: %v", err)
		}
		assertSummaryFacts(t, "组织级并存", f.summaries(t, financePrincipal, targetOf(orderMixID))[orderMixID],
			biz.OrderCommissionVisibilityOrganization, 0, 1, 1, 1, 1, "60.00000000", "30.00000000", "60.00000000", "10.00000000")
		if currency := f.summaries(t, financePrincipal, targetOf(orderMixID))[orderMixID].BaseCurrency; currency != "CNY" {
			t.Fatalf("组织级汇总币种应为组织本位币 CNY: %s", currency)
		}

		// 冲减确认后不再是「待处理」，但也不得宣称为已扣回（已发放事实不变）。
		confirmedDecrease, err := usecase.ConfirmAdjustment(ctx, org.organizationID, actorID, decrease.ID, decrease.Version)
		if err != nil {
			t.Fatalf("确认冲减失败: %v", err)
		}
		assertSummaryFacts(t, "冲减确认后", f.summaries(t, financePrincipal, targetOf(orderMixID))[orderMixID],
			biz.OrderCommissionVisibilityOrganization, 0, 1, 1, 1, 0, "60.00000000", "30.00000000", "60.00000000", "")
		if _, err = usecase.MarkAdjustmentPaid(ctx, org.organizationID, actorID, decrease.ID, confirmedDecrease.Version); err != nil {
			t.Fatalf("标记冲减已扣回失败: %v", err)
		}
		assertSummaryFacts(t, "冲减已扣回后", f.summaries(t, financePrincipal, targetOf(orderMixID))[orderMixID],
			biz.OrderCommissionVisibilityOrganization, 0, 1, 1, 1, 0, "60.00000000", "30.00000000", "60.00000000", "")
	})

	t.Run("预计机会随来源与基础提成单消长", func(t *testing.T) {
		// 双来源均唯一命中方案且无基础提成单：预计机会 = 2。
		assertSummaryFacts(t, "双来源机会", f.summaries(t, financePrincipal, targetOf(orderOppID))[orderOppID],
			biz.OrderCommissionVisibilityOrganization, 2, 0, 0, 0, 0, "", "", "", "")

		// 来源 1 生成 DRAFT 提成：该来源不再是机会，来源 2 仍在。第二笔应收费用
		// 参与订单行分母（应收 1600 / 应付 400），来源 1 已实现 1000 → 毛利 750
		// → 10% 计提 75。
		created, err := usecase.Create(ctx, org.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceOppV1, EmployeeID: oppEmployeeID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-opp-" + suffix,
		})
		if err != nil {
			t.Fatalf("创建预计来源提成失败: %v", err)
		}
		assertSummaryFacts(t, "生成后机会", f.summaries(t, financePrincipal, targetOf(orderOppID))[orderOppID],
			biz.OrderCommissionVisibilityOrganization, 1, 1, 0, 0, 0, "75.00000000", "", "", "")

		// 取消后：取消记录不参与有效汇总，来源 1 重新成为预计机会。
		if _, err = usecase.Cancel(ctx, org.organizationID, actorID, created.ID, created.Version, "测试取消"); err != nil {
			t.Fatalf("取消提成草稿失败: %v", err)
		}
		assertSummaryFacts(t, "取消后机会", f.summaries(t, financePrincipal, targetOf(orderOppID))[orderOppID],
			biz.OrderCommissionVisibilityOrganization, 2, 0, 0, 0, 0, "", "", "", "")
	})

	t.Run("同一主体在不同组织权限不同", func(t *testing.T) {
		// 同一调用者（selfUserID）在组织 B 生成自己的 DRAFT 提成；同事在组织 B
		// 也有 DRAFT 提成。
		if _, err := usecase.Create(ctx, orgB.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceBID, EmployeeID: selfUserID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-b-self-" + suffix,
		}); err != nil {
			t.Fatalf("创建组织 B 本人提成失败: %v", err)
		}
		if _, err := usecase.Create(ctx, orgB.organizationID, actorID, biz.CreateCommissionInput{
			VerificationID: sourceBID, EmployeeID: colleagueBID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "os-b-colleague-" + suffix,
		}); err != nil {
			t.Fatalf("创建组织 B 同事提成失败: %v", err)
		}

		// 同一页同时包含组织 A 与组织 B 的订单：组织 A 为组织级全员视图（草稿 2），
		// 组织 B 退回本人视图（只含本人 1 张草稿，同事记录不可见）。
		summaries := f.summaries(t, financePrincipal, targetOf(orderSelfID), biz.OrderCommissionSummaryTarget{OrderID: orderBID, OrganizationID: orgB.organizationID})
		assertSummaryFacts(t, "授权组织", summaries[orderSelfID],
			biz.OrderCommissionVisibilityOrganization, 0, 2, 0, 0, 0, "120.00000000", "", "", "")
		assertSummaryFacts(t, "未授权组织", summaries[orderBID],
			biz.OrderCommissionVisibilityEmployee, 0, 1, 0, 0, 0, "60.00000000", "", "", "")
	})
}
