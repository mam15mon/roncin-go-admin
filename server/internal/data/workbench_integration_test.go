package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	assignment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/shopspring/decimal"
)

// 工作台集成夹具：单一 SE 订单（应收 1000 / 应付 400 已确认费用）+ 一张已确认
// 应收账单（1000）+ 一张部分核销（400）的 ACTIVE 核销单，供门禁、预计机会、
// 应收未结与财务摘要矩阵复用。
type workbenchIntegrationFixture struct {
	t                *testing.T
	data             *Data
	organizationID   uuid.UUID
	customerID       uuid.UUID
	actorID          uuid.UUID
	orderID          uuid.UUID
	orderNo          string
	receivableFeeID  uuid.UUID
	receivableBillID uuid.UUID
	verificationID   uuid.UUID
	suffix           string
}

const workbenchIntegrationDate = "2026-08-30"

func newWorkbenchIntegrationFixture(t *testing.T) *workbenchIntegrationFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	org, err := data.db.Organization.Create().
		SetCode("WB-" + suffix).
		SetName("工作台测试组织-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	actor, err := data.db.User.Create().
		SetUsername("wb_actor_" + suffix).
		SetDisplayName("工作台操作员-" + suffix).
		SetEnabled(true).
		SetIsBootstrapAdmin(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试操作员: %v", err)
	}
	if _, err = data.db.Membership.Create().SetOrganizationID(org.ID).SetUserID(actor.ID).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试操作员成员资格: %v", err)
	}
	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-WB-" + suffix).
		SetLegalName("工作台测试客户-" + suffix).
		SetNormalizedName("工作台测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	fixture := &workbenchIntegrationFixture{t: t, data: data, organizationID: org.ID, customerID: customer.ID, actorID: actor.ID, suffix: suffix}

	orderNo := "WB-SE" + suffix
	order, err := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(org.ID).
		SetOrderNo(orderNo).
		SetCustomerID(customer.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetOrderDate(workbenchIntegrationDate).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单: %v", err)
	}
	fixture.orderID = order.ID
	fixture.orderNo = orderNo

	receivableFee, err := data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("wb-fee-rec-" + suffix).
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
		SetExchangeRateDate(workbenchIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("1000.00000000").
		SetExpenseDate(workbenchIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试应收费用: %v", err)
	}
	fixture.receivableFeeID = receivableFee.ID
	if _, err = data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("wb-fee-pay-" + suffix).
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
		SetExchangeRateDate(workbenchIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("400.00000000").
		SetExpenseDate(workbenchIntegrationDate).
		SetVersion(1).
		Save(ctx); err != nil {
		t.Fatalf("创建测试应付费用: %v", err)
	}

	billCreate := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("WB-BILL-" + suffix).
		SetIdempotencyKey("wb-bill-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(workbenchIntegrationDate).
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetFeeCount(1).
		SetBillDate(workbenchIntegrationDate).
		SetConfirmedAt(time.Now().UTC()).
		SetConfirmedBy(actor.ID).
		SetVersion(1)
	bill, saveErr := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
	if saveErr != nil {
		t.Fatalf("创建测试应收账单: %v", saveErr)
	}
	fixture.receivableBillID = bill.ID
	if _, err = data.db.FinanceBillLine.Create().
		SetBillID(bill.ID).
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

	cashflow, err := data.db.FinanceCashflow.Create().
		SetOrganizationID(org.ID).
		SetFlowNo("WB-FLOW-" + suffix).
		SetIdempotencyKey("wb-cashflow-" + suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetAmount("400.00000000").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(workbenchIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("400.00000000").
		SetTransactionDate(workbenchIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试资金流水: %v", err)
	}
	verificationItem, err := data.db.FinanceVerification.Create().
		SetOrganizationID(org.ID).
		SetVerificationNo("WB-VR-" + suffix).
		SetIdempotencyKey("wb-verification-" + suffix).
		SetDirection(verification.DirectionRECEIVABLE).
		SetStatus(verification.StatusACTIVE).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetAmount("400.00000000").
		SetBaseCurrency("CNY").
		SetBaseAmount("400.00000000").
		SetBillBaseAmount("400.00000000").
		SetCashflowBaseAmount("400.00000000").
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(workbenchIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试核销单: %v", err)
	}
	fixture.verificationID = verificationItem.ID
	if _, err = data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationItem.ID).
		SetCashflowID(cashflow.ID).
		SetBillID(bill.ID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo(bill.BillNo).
		SetAmount("400.00000000").
		SetBillBaseAmount("400.00000000").
		SetCashflowBaseAmount("400.00000000").
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试核销分摊: %v", err)
	}
	return fixture
}

// newEmployee 创建启用员工与组织成员关系。
func (f *workbenchIntegrationFixture) newEmployee(label string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	employee, err := f.data.db.User.Create().
		SetUsername("wb_" + label + "_" + f.suffix).
		SetDisplayName("工作台员工-" + label + "-" + f.suffix).
		SetEnabled(true).
		SetIsBootstrapAdmin(false).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试员工 %s: %v", label, err)
	}
	if _, err = f.data.db.Membership.Create().SetOrganizationID(f.organizationID).SetUserID(employee.ID).SetEnabled(true).Save(ctx); err != nil {
		f.t.Fatalf("创建测试员工 %s 成员资格: %v", label, err)
	}
	return employee.ID
}

// addPersonnel 把员工挂到订单协作人员。
func (f *workbenchIntegrationFixture) addPersonnel(employeeID uuid.UUID, role orderpersonnelent.Role) {
	f.t.Helper()
	if _, err := f.data.db.OrderPersonnel.Create().
		SetOrderID(f.orderID).
		SetUserID(employeeID).
		SetOrganizationID(f.organizationID).
		SetRole(role).
		Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试订单协作人员: %v", err)
	}
}

// addAttribution 为员工追加订单提成归属（销售）。
func (f *workbenchIntegrationFixture) addAttribution(employeeID uuid.UUID, role biz.CommissionPersonnelRole) {
	f.t.Helper()
	if _, err := f.data.db.OrderCommissionAttribution.Create().
		SetOrganizationID(f.organizationID).
		SetOrderID(f.orderID).
		SetCustomerID(f.customerID).
		SetSourceAssignmentID(uuid.New()).
		SetEmployeeID(employeeID).
		SetEmployeeName("工作台员工-" + employeeID.String()[:8]).
		SetPersonnelRole(attribution.PersonnelRole(role)).
		SetAttributedAt(time.Now()).
		Save(context.Background()); err != nil {
		f.t.Fatalf("创建测试提成归属: %v", err)
	}
}

// workbenchRuleInfo 是测试方案的主键与全名。
type workbenchRuleInfo struct {
	ID   uuid.UUID
	Name string
}

// newRule 创建提成方案并返回主键与全名。
func (f *workbenchIntegrationFixture) newRule(name string, role biz.CommissionPersonnelRole, ratePercent, effectiveFrom string, effectiveTo *string) workbenchRuleInfo {
	f.t.Helper()
	fullName := name + "-" + f.suffix
	create := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName(fullName).
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
	return workbenchRuleInfo{ID: item.ID, Name: fullName}
}

// assign 为员工追加未取消分配段。
func (f *workbenchIntegrationFixture) assign(ruleID, employeeID uuid.UUID, effectiveFrom string, effectiveTo *string) {
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

// cancelAssignment 以撤销标记模拟「仅取消记录」。
func (f *workbenchIntegrationFixture) cancelAssignment(assignmentID uuid.UUID) {
	f.t.Helper()
	now := time.Now().UTC()
	if _, err := f.data.db.FinanceCommissionRuleAssignment.Update().
		Where(assignment.IDEQ(assignmentID)).
		SetCancelledAt(now).
		SetCancelledBy(f.actorID).
		Save(context.Background()); err != nil {
		f.t.Fatalf("撤销测试分配: %v", err)
	}
}

// createCommission 直接落库一条提成快照，用于历史事实与财务摘要矩阵。
func (f *workbenchIntegrationFixture) createCommission(employeeID uuid.UUID, ruleInfo workbenchRuleInfo, role biz.CommissionPersonnelRole, status commissionent.Status, amount string, paidAt *time.Time, withVerification bool) uuid.UUID {
	f.t.Helper()
	create := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo("WB-TC-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]).
		SetIdempotencyKey("wb-commission-" + uuid.NewString()).
		SetEmployeeID(employeeID).
		SetEmployeeName("工作台员工-" + employeeID.String()[:8]).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(ruleInfo.ID).
		SetRuleName(ruleInfo.Name).
		SetPersonnelRole(string(role)).
		SetCalculationBasis("REALIZED_PROFIT").
		SetBaseCurrency("CNY").
		SetRealizedRevenue("400.00000000").
		SetAllocatedCost("160.00000000").
		SetRealizedProfit("240.00000000").
		SetCommissionBaseAmount("240.00000000").
		SetRatePercent("10.0000").
		SetCommissionAmount(amount).
		SetCommissionDate(workbenchIntegrationDate).
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate(workbenchIntegrationDate).
		SetCnyCommissionAmount(amount).
		SetStatus(status).
		SetVersion(1)
	if withVerification {
		create = create.SetVerificationID(f.verificationID).SetVerificationNo("WB-VR-" + f.suffix)
	}
	if paidAt != nil {
		create = create.SetPaidAt(*paidAt)
	}
	item, err := create.Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试提成快照: %v", err)
	}
	return item.ID
}

// createAdjustment 直接落库一条调整单。
func (f *workbenchIntegrationFixture) createAdjustment(commissionID, employeeID uuid.UUID, direction string, status commissionadjustmentent.Status, amount string, sourceType commissionadjustmentent.SourceType, supplementRequestID *uuid.UUID) uuid.UUID {
	f.t.Helper()
	create := f.data.db.FinanceCommissionAdjustment.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(commissionID).
		SetOrderID(f.orderID).
		SetAdjustmentNo("WB-TA-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]).
		SetIdempotencyKey("wb-adjustment-" + uuid.NewString()).
		SetCommissionNo("WB-TC-PARENT").
		SetOrderNo(f.orderNo).
		SetEmployeeID(employeeID).
		SetEmployeeName("工作台员工-" + employeeID.String()[:8]).
		SetSourceType(sourceType).
		SetDirection(commissionadjustmentent.Direction(direction)).
		SetStatus(status).
		SetBaseCurrency("CNY").
		SetAmount(amount).
		SetReason("工作台测试调整").
		SetVersion(1)
	if supplementRequestID != nil {
		create = create.SetSourceFeeSupplementRequestID(*supplementRequestID)
	}
	item, err := create.Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试调整单: %v", err)
	}
	return item.ID
}

// createPendingSupplementRequest 落库一条 PENDING 锁后费用补录申请（业务锁依据）。
func (f *workbenchIntegrationFixture) createPendingSupplementRequest(requestedBy uuid.UUID) uuid.UUID {
	f.t.Helper()
	item, err := f.data.db.OrderFeeSupplementRequest.Create().
		SetOrganizationID(f.organizationID).
		SetOrderID(f.orderID).
		SetLockBasis(orderfeesupplementent.LockBasisBUSINESS).
		SetBusinessLockGeneration(1).
		SetIdempotencyKey("wb-supplement-" + uuid.NewString()).
		SetRequestFingerprint("wb-fp-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]).
		SetDirection(orderfeesupplementent.DirectionPAYABLE).
		SetFeeCode("DOC_FEE").
		SetFeeName("单证费").
		SetSettlementPartyID(f.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetTaxInclusive(true).
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeesupplementent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(workbenchIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("100.00000000").
		SetExpenseDate(workbenchIntegrationDate).
		SetReason("工作台测试补录").
		SetRequestedBy(requestedBy).
		SetRequestedAt(time.Now().UTC()).
		SetStatus(orderfeesupplementent.StatusPENDING).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试补录申请: %v", err)
	}
	return item.ID
}

// grantOrderLockPermission 授予员工当前组织 SE 订单锁定权限（实时审批资格）。
func (f *workbenchIntegrationFixture) grantOrderLockPermission(employeeID uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	membershipItem, err := f.data.db.Membership.Query().
		Where(membership.OrganizationIDEQ(f.organizationID), membership.UserIDEQ(employeeID)).
		Only(ctx)
	if err != nil {
		f.t.Fatalf("查询成员资格: %v", err)
	}
	permKey := "business.order.se.lock"
	permissionItem, err := f.data.db.Permission.Create().
		SetKey(permKey).
		SetName("海运出口订单锁定-工作台测试-" + f.suffix).
		SetDescription("工作台集成测试").
		SetGroup("order").
		Save(ctx)
	if err != nil {
		permissionItem, err = f.data.db.Permission.Query().Where(permissionent.KeyEQ(permKey)).First(ctx)
		if err != nil {
			f.t.Fatalf("准备锁权限: %v", err)
		}
	}
	roleItem, err := f.data.db.Role.Create().
		SetOrganizationID(f.organizationID).
		SetCode("wb_locker_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]).
		SetName("工作台锁资格-" + f.suffix).
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(permissionItem).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建锁资格角色: %v", err)
	}
	if _, err := f.data.db.RoleAssignment.Create().SetMembershipID(membershipItem.ID).SetRoleID(roleItem.ID).Save(ctx); err != nil {
		f.t.Fatalf("分配锁资格角色: %v", err)
	}
}

func (f *workbenchIntegrationFixture) repo() biz.WorkbenchRepo { return NewWorkbenchRepo(f.data) }

func (f *workbenchIntegrationFixture) scope(userID uuid.UUID, mutate func(*biz.WorkbenchScope)) biz.WorkbenchScope {
	now := time.Now()
	scope := biz.WorkbenchScope{
		OrganizationID: f.organizationID,
		UserID:         userID,
		Today:          biz.FinanceBusinessDate(now),
		Now:            now,
	}
	if mutate != nil {
		mutate(&scope)
	}
	return scope
}

// TestWorkbenchOverviewGateMatrixPostgres 覆盖门禁矩阵：无分配无历史、未来方案
// 分配、过期分配但有历史、仅取消记录、订单协作但固定薪。
func TestWorkbenchOverviewGateMatrixPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	t.Run("无方案分配无历史且订单协作但固定薪", func(t *testing.T) {
		fixture := newWorkbenchIntegrationFixture(t)
		employee := fixture.newEmployee("fixed")
		fixture.addPersonnel(employee, orderpersonnelent.RoleSALES)
		fixture.addAttribution(employee, biz.CommissionRoleSales)

		overview, err := fixture.repo().GetOverview(context.Background(), fixture.scope(employee, nil))
		if err != nil {
			t.Fatalf("读取工作台概览: %v", err)
		}
		if overview.Eligible {
			t.Fatalf("固定薪协作员工不应开启提成门禁")
		}
		if overview.CommissionSummary != nil {
			t.Fatalf("门禁为假时不应返回提成金额模块")
		}
		if overview.NextEffectiveDate != "" {
			t.Fatalf("无未来分配时不应返回生效日: %q", overview.NextEffectiveDate)
		}
		if len(overview.RecentOrders) != 1 || overview.RecentOrders[0].OrderID != fixture.orderID {
			t.Fatalf("固定薪员工仍应看到本人协作订单: %+v", overview.RecentOrders)
		}
		if overview.Finance == nil {
			t.Fatalf("财务摘要结构应始终返回（无权限字段为缺省）")
		}
		if overview.Finance.CanReadCommission || overview.Finance.CanManageCommission {
			t.Fatalf("未授权时财务权限标记应为假")
		}
		if overview.Finance.PendingSupplementApprovalCount != 0 {
			t.Fatalf("无实时资格时审批计数应为零")
		}
	})

	t.Run("未来方案分配开启门禁并返回最近生效日", func(t *testing.T) {
		fixture := newWorkbenchIntegrationFixture(t)
		employee := fixture.newEmployee("future")
		ruleInfo := fixture.newRule("未来销售方案", biz.CommissionRoleSales, "10.0000", "2099-01-01", nil)
		fixture.assign(ruleInfo.ID, employee, "2099-01-01", nil)

		overview, err := fixture.repo().GetOverview(context.Background(), fixture.scope(employee, nil))
		if err != nil {
			t.Fatalf("读取工作台概览: %v", err)
		}
		if !overview.Eligible {
			t.Fatalf("未来有效分配应开启门禁")
		}
		if overview.NextEffectiveDate != "2099-01-01" {
			t.Fatalf("next_effective_date = %q，期望 2099-01-01", overview.NextEffectiveDate)
		}
		if overview.CommissionSummary == nil {
			t.Fatalf("门禁为真时应返回提成汇总（业务空态）")
		}
		if !overview.CommissionSummary.DraftAmount.IsZero() {
			t.Fatalf("尚无提成数据时金额应为零")
		}
	})

	t.Run("过期方案分配但有已发历史仍可查看", func(t *testing.T) {
		fixture := newWorkbenchIntegrationFixture(t)
		employee := fixture.newEmployee("expired")
		expiredTo := "2026-01-31"
		ruleInfo := fixture.newRule("过期销售方案", biz.CommissionRoleSales, "10.0000", "2025-01-01", &expiredTo)
		fixture.assign(ruleInfo.ID, employee, "2025-01-01", &expiredTo)
		paidAt := time.Now().UTC().Add(-24 * time.Hour)
		fixture.createCommission(employee, ruleInfo, biz.CommissionRoleSales, commissionent.StatusPAID, "120.00000000", &paidAt, false)

		overview, err := fixture.repo().GetOverview(context.Background(), fixture.scope(employee, nil))
		if err != nil {
			t.Fatalf("读取工作台概览: %v", err)
		}
		if !overview.Eligible {
			t.Fatalf("历史提成单应开启门禁")
		}
		summary := overview.CommissionSummary
		if summary == nil {
			t.Fatalf("历史提成员工应获得提成汇总")
		}
		if summary.PaidCount != 1 || !summary.PaidAmount.Equal(decimal.RequireFromString("120")) {
			t.Fatalf("已发分桶不正确: %+v", summary)
		}
		if !summary.PaidThisMonth.Equal(summary.PaidAmount) || !summary.PaidThisYear.Equal(summary.PaidAmount) {
			t.Fatalf("本月/本年已发累计应包含昨日发放: %+v", summary)
		}
		if summary.BaseCurrency != "CNY" || overview.BaseCurrency != "CNY" {
			t.Fatalf("汇总本位币口径应来自当前组织")
		}
	})

	t.Run("仅取消记录不开启门禁", func(t *testing.T) {
		fixture := newWorkbenchIntegrationFixture(t)
		employee := fixture.newEmployee("cancelled")
		ruleInfo := fixture.newRule("取消销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
		assignmentCreate := f0AssignmentID(fixture, ruleInfo.ID, employee)
		fixture.cancelAssignment(assignmentCreate)
		commissionID := fixture.createCommission(employee, ruleInfo, biz.CommissionRoleSales, commissionent.StatusCANCELLED, "80.00000000", nil, false)
		fixture.createAdjustment(commissionID, employee, "DECREASE", commissionadjustmentent.StatusCANCELLED, "10.00000000", commissionadjustmentent.SourceTypeMANUAL, nil)

		overview, err := fixture.repo().GetOverview(context.Background(), fixture.scope(employee, nil))
		if err != nil {
			t.Fatalf("读取工作台概览: %v", err)
		}
		if overview.Eligible {
			t.Fatalf("仅取消记录不能开启门禁")
		}
		if overview.CommissionSummary != nil {
			t.Fatalf("仅取消记录时不应返回提成金额模块")
		}
	})
}

// f0AssignmentID 创建分配并返回分配 ID。
func f0AssignmentID(fixture *workbenchIntegrationFixture, ruleID, employeeID uuid.UUID) uuid.UUID {
	fixture.t.Helper()
	item, err := fixture.data.db.FinanceCommissionRuleAssignment.Create().
		SetOrganizationID(fixture.organizationID).
		SetRuleID(ruleID).
		SetEmployeeID(employeeID).
		SetEffectiveFrom("2026-01-01").
		SetCreatedBy(fixture.actorID).
		Save(context.Background())
	if err != nil {
		fixture.t.Fatalf("创建测试分配: %v", err)
	}
	return item.ID
}

// TestWorkbenchEstimatedOpportunityPostgres 覆盖预计可计提：唯一命中方案与分配
// 的来源形成机会；生成有效提成后不再是机会；固定薪员工无机会。
func TestWorkbenchEstimatedOpportunityPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newWorkbenchIntegrationFixture(t)
	ctx := context.Background()
	employee := fixture.newEmployee("sales")
	fixture.addPersonnel(employee, orderpersonnelent.RoleSALES)
	fixture.addAttribution(employee, biz.CommissionRoleSales)
	fixed := fixture.newEmployee("fixedpay")
	fixture.addPersonnel(fixed, orderpersonnelent.RoleOPERATOR)
	fixture.addAttribution(fixed, biz.CommissionRoleSales)
	ruleInfo := fixture.newRule("销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
	fixture.assign(ruleInfo.ID, employee, "2026-01-01", nil)

	repo := fixture.repo()
	overview, err := repo.GetOverview(ctx, fixture.scope(employee, nil))
	if err != nil {
		t.Fatalf("读取工作台概览: %v", err)
	}
	summary := overview.CommissionSummary
	if summary == nil || summary.Estimated == nil {
		t.Fatalf("唯一命中方案与分配的核销来源应形成预计机会: %+v", summary)
	}
	// 已实现收入 400（400/1000 × 1000 账单行），毛利 240，10% 提成 = 24。
	if summary.Estimated.OpportunityCount != 1 || !summary.Estimated.EstimatedAmount.Equal(decimal.RequireFromString("24")) {
		t.Fatalf("预计机会不正确: %+v", summary.Estimated)
	}
	if summary.Estimated.HasMore {
		t.Fatalf("单个来源不应触发截断标记")
	}

	fixedOverview, err := repo.GetOverview(ctx, fixture.scope(fixed, nil))
	if err != nil {
		t.Fatalf("读取固定薪员工概览: %v", err)
	}
	if fixedOverview.Eligible || fixedOverview.CommissionSummary != nil {
		t.Fatalf("固定薪员工不应有提成模块")
	}

	// 生成有效提成后该来源不再是机会。
	fixture.createCommission(employee, ruleInfo, biz.CommissionRoleSales, commissionent.StatusDRAFT, "24.00000000", nil, true)
	overviewAfter, err := repo.GetOverview(ctx, fixture.scope(employee, nil))
	if err != nil {
		t.Fatalf("读取生成后概览: %v", err)
	}
	if overviewAfter.CommissionSummary == nil || overviewAfter.CommissionSummary.Estimated != nil {
		t.Fatalf("已有有效基础提成的来源不应重复计入预计机会: %+v", overviewAfter.CommissionSummary)
	}
}

// TestWorkbenchReceivablesPostgres 覆盖本人应收未结项：部分核销账单按原币展示
// 未结余额与逾期口径，已结清来源不计入。
func TestWorkbenchReceivablesPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newWorkbenchIntegrationFixture(t)
	employee := fixture.newEmployee("sales")
	fixture.addAttribution(employee, biz.CommissionRoleSales)

	// 夹具账单 1000 已被核销 400，未结 600。
	result, err := fixture.repo().ListMyReceivables(context.Background(), fixture.scope(employee, nil), biz.WorkbenchReceivableFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("读取本人应收未结: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("应仅返回一张部分核销账单: total=%d", result.Total)
	}
	item := result.Items[0]
	if item.BillID != fixture.receivableBillID {
		t.Fatalf("返回账单不正确: %+v", item)
	}
	if !item.UnverifiedAmount.Equal(decimal.RequireFromString("600")) {
		t.Fatalf("未结余额 = %s，期望 600", item.UnverifiedAmount)
	}
	if item.Currency != "CNY" || item.OverdueDays < 0 {
		t.Fatalf("币种与逾期口径不正确: %+v", item)
	}

	// 无归属员工看不到该账单。
	other := fixture.newEmployee("other")
	emptyResult, err := fixture.repo().ListMyReceivables(context.Background(), fixture.scope(other, nil), biz.WorkbenchReceivableFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("读取他人应收未结: %v", err)
	}
	if emptyResult.Total != 0 {
		t.Fatalf("无提成归属员工不应看到他人账单: %d", emptyResult.Total)
	}
}

// TestWorkbenchFinanceSummaryPostgres 覆盖财务/审批摘要：权限标记由范围控制、
// 冲减建议计数、补录审批仅统计实时有资格者。
func TestWorkbenchFinanceSummaryPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newWorkbenchIntegrationFixture(t)
	ctx := context.Background()
	commissionEmployee := fixture.newEmployee("emp")
	summaryRule := fixture.newRule("摘要销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
	commissionID := fixture.createCommission(commissionEmployee, summaryRule, biz.CommissionRoleSales, commissionent.StatusCONFIRMED, "90.00000000", nil, false)
	requestID := fixture.createPendingSupplementRequest(commissionEmployee)
	fixture.createAdjustment(commissionID, commissionEmployee, "DECREASE", commissionadjustmentent.StatusDRAFT, "5.00000000", commissionadjustmentent.SourceTypeLOCKED_FEE_SUPPLEMENT, &requestID)

	t.Run("无财务权限时计数缺省", func(t *testing.T) {
		overview, err := fixture.repo().GetOverview(ctx, fixture.scope(commissionEmployee, nil))
		if err != nil {
			t.Fatalf("读取概览: %v", err)
		}
		if overview.Finance.CanReadCommission || overview.Finance.ConfirmedCommissionCount != 0 || overview.Finance.PendingDecreaseCount != 0 {
			t.Fatalf("无读取权限时提成计数不应返回: %+v", overview.Finance)
		}
		if overview.Finance.PendingSupplementApprovalCount != 0 {
			t.Fatalf("无实时资格时补录审批计数应为零")
		}
	})

	t.Run("财务读取权限与冲减建议计数", func(t *testing.T) {
		overview, err := fixture.repo().GetOverview(ctx, fixture.scope(commissionEmployee, func(s *biz.WorkbenchScope) {
			s.CommissionReadable = true
			s.CommissionManageable = true
		}))
		if err != nil {
			t.Fatalf("读取概览: %v", err)
		}
		if !overview.Finance.CanReadCommission || !overview.Finance.CanManageCommission {
			t.Fatalf("权限标记应随范围置真: %+v", overview.Finance)
		}
		if overview.Finance.ConfirmedCommissionCount != 1 {
			t.Fatalf("已确认待发提成计数 = %d，期望 1", overview.Finance.ConfirmedCommissionCount)
		}
		if overview.Finance.PendingDecreaseCount != 1 {
			t.Fatalf("冲减建议计数 = %d，期望 1", overview.Finance.PendingDecreaseCount)
		}
	})

	t.Run("补录审批只统计实时有资格者", func(t *testing.T) {
		approver := fixture.newEmployee("approver")
		fixture.grantOrderLockPermission(approver)
		overview, err := fixture.repo().GetOverview(ctx, fixture.scope(approver, nil))
		if err != nil {
			t.Fatalf("读取审批人概览: %v", err)
		}
		if overview.Finance.PendingSupplementApprovalCount != 1 || len(overview.Finance.PendingSupplementApprovals) != 1 {
			t.Fatalf("实时有资格审批人应看到 1 条待办: %+v", overview.Finance)
		}
		item := overview.Finance.PendingSupplementApprovals[0]
		if item.OrderID != fixture.orderID || item.RequestID != requestID || item.FeeCode != "DOC_FEE" {
			t.Fatalf("审批待办投影不正确: %+v", item)
		}
		if item.Amount.Sign() <= 0 {
			t.Fatalf("审批待办金额应大于零")
		}
	})

	t.Run("bootstrap admin 实时具备审批资格", func(t *testing.T) {
		overview, err := fixture.repo().GetOverview(ctx, fixture.scope(commissionEmployee, func(s *biz.WorkbenchScope) {
			s.IsBootstrapAdmin = true
		}))
		if err != nil {
			t.Fatalf("读取概览: %v", err)
		}
		if overview.Finance.PendingSupplementApprovalCount != 1 {
			t.Fatalf("bootstrap admin 应实时具备资格: %d", overview.Finance.PendingSupplementApprovalCount)
		}
	})
}

// TestWorkbenchListMyCommissionsPostgres 覆盖本人提成明细分页、状态与日期过滤
// 以及调整明细嵌套。
func TestWorkbenchListMyCommissionsPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newWorkbenchIntegrationFixture(t)
	ctx := context.Background()
	employee := fixture.newEmployee("emp")
	listRule := fixture.newRule("明细销售方案", biz.CommissionRoleSales, "10.0000", "2026-01-01", nil)
	draftID := fixture.createCommission(employee, listRule, biz.CommissionRoleSales, commissionent.StatusDRAFT, "30.00000000", nil, true)
	paidAt := time.Now().UTC().Add(-48 * time.Hour)
	paidID := fixture.createCommission(employee, listRule, biz.CommissionRoleSales, commissionent.StatusPAID, "50.00000000", &paidAt, false)
	fixture.createAdjustment(draftID, employee, "DECREASE", commissionadjustmentent.StatusDRAFT, "5.00000000", commissionadjustmentent.SourceTypeMANUAL, nil)
	other := fixture.newEmployee("other")
	fixture.createCommission(other, listRule, biz.CommissionRoleSales, commissionent.StatusDRAFT, "999.00000000", nil, false)

	repo := fixture.repo()
	scope := fixture.scope(employee, nil)
	all, err := repo.ListMyCommissions(ctx, scope, biz.WorkbenchCommissionFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("读取本人提成明细: %v", err)
	}
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("本人明细应只有 2 条（不包含他人）: %d", all.Total)
	}
	if all.Items[0].Status != biz.CommissionPaid && all.Items[0].Status != biz.CommissionDraft {
		t.Fatalf("状态投影不正确: %+v", all.Items[0])
	}

	draftOnly, err := repo.ListMyCommissions(ctx, scope, biz.WorkbenchCommissionFilter{Page: 1, PageSize: 20, Status: biz.CommissionDraft})
	if err != nil {
		t.Fatalf("状态过滤失败: %v", err)
	}
	if draftOnly.Total != 1 || draftOnly.Items[0].ID != draftID {
		t.Fatalf("DRAFT 过滤应返回本人草稿提成: %+v", draftOnly)
	}
	if len(draftOnly.Items[0].Adjustments) != 1 || draftOnly.Items[0].Adjustments[0].ID == uuid.Nil {
		t.Fatalf("草稿提成应嵌套本人调整明细: %+v", draftOnly.Items[0].Adjustments)
	}

	recentOnly, err := repo.ListMyCommissions(ctx, scope, biz.WorkbenchCommissionFilter{
		Page: 1, PageSize: 20, CommissionDateFrom: workbenchIntegrationDate, CommissionDateTo: workbenchIntegrationDate,
	})
	if err != nil {
		t.Fatalf("日期过滤失败: %v", err)
	}
	if recentOnly.Total != 2 {
		t.Fatalf("归属日期落在今天的提成应为 2 条: %d", recentOnly.Total)
	}
	_ = paidID
	// 分页上限校验位于用例层：pageSize=200 允许、201 拒绝。
	usecase := biz.NewWorkbenchUsecase(repo)
	if _, err := usecase.ListMyCommissions(ctx, scope, biz.WorkbenchCommissionFilter{Page: 1, PageSize: 200}); err != nil {
		t.Fatalf("pageSize=200 应允许: %v", err)
	}
	if _, err := usecase.ListMyCommissions(ctx, scope, biz.WorkbenchCommissionFilter{Page: 1, PageSize: 201}); !errors.Is(err, biz.ErrWorkbenchInvalid) {
		t.Fatalf("pageSize=201 应返回 ErrWorkbenchInvalid，实际 %v", err)
	}
}
