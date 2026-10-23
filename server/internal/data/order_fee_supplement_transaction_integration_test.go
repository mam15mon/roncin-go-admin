package data

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"

	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	commissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"

	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"

	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
)

// feeSupplementPostgresFixture 锁后费用补录集成测试夹具：CNY 总部组织 + SI 订单
// + 费用设置引用 + 锁资格角色与审批人。测试直接调用 biz 用例与仓储，传输层
// 权限由 server 中间件测试另行覆盖。
type feeSupplementPostgresFixture struct {
	t              *testing.T
	data           *Data
	ctx            context.Context
	organizationID uuid.UUID
	customerID     uuid.UUID
	requesterID    uuid.UUID
	approverID     uuid.UUID
	employeeID     uuid.UUID
	feeSettingID   uuid.UUID
	billingUnitID  uuid.UUID
	chargeCategory uuid.UUID
	lockRoleID     uuid.UUID
	usecase        *biz.OrderFeeSupplementUsecase
	commissionRepo biz.CommissionRepo
	suffix         string
}

func newFeeSupplementFixture(t *testing.T) *feeSupplementPostgresFixture {
	t.Helper()
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:10]

	org, err := data.db.Organization.Create().
		SetCode("FSUP-" + suffix).
		SetName("费用补录测试组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-" + suffix).
		SetLegalName("费用补录测试客户-" + suffix).
		SetNormalizedName("费用补录测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	// CNY 属于全局基线数据（迁移种子已内置），存在即复用。
	if exists, currencyErr := data.db.Currency.Query().Where(currencyent.CodeEQ("CNY")).Exist(ctx); currencyErr != nil {
		t.Fatalf("查询币种: %v", currencyErr)
	} else if !exists {
		if _, createErr := data.db.Currency.Create().SetCode("CNY").SetName("人民币").Save(ctx); createErr != nil {
			t.Fatalf("创建币种: %v", createErr)
		}
	}
	billingUnit, billingErr := data.db.BillingUnit.Create().SetCode("FSUP-UNIT-" + suffix).SetName("补录测试票").SetEnabled(true).Save(ctx)
	if billingErr != nil {
		t.Fatalf("创建计费单位: %v", billingErr)
	}
	taxable, taxableErr := data.db.TaxableService.Create().SetOrganizationID(org.ID).SetName("补录测试应税服务-" + suffix).SetDefaultTaxRate("6").SetEnabled(true).Save(ctx)
	if taxableErr != nil {
		t.Fatalf("创建税务名称: %v", taxableErr)
	}
	chargeCategory, categoryErr := data.db.MasterDataItem.Create().SetKind("charge_category").SetCode("FSUP-CAT-" + suffix).SetName("补录测试大类").SetEnabled(true).Save(ctx)
	if categoryErr != nil {
		t.Fatalf("创建费用大类: %v", categoryErr)
	}
	feeSetting, settingErr := data.db.FeeSetting.Create().
		SetOrganizationID(org.ID).
		SetFeeCode("FSUP_TRUCK").
		SetNameZh("补录测试拖车费").
		SetChargeCategoryID(chargeCategory.ID).
		SetDefaultCurrency("CNY").
		SetBillingUnitID(billingUnit.ID).
		SetTaxRate("6.00").
		SetTaxableServiceID(taxable.ID).
		SetEnabled(true).
		Save(ctx)
	if settingErr != nil {
		t.Fatalf("创建费用设置: %v", settingErr)
	}

	newUser := func(label string, dingTalk bool) *ent.User {
		create := data.db.User.Create().SetDisplayName("补录" + label + "-" + suffix).SetEnabled(true).SetIsBootstrapAdmin(false)
		if dingTalk {
			create = create.SetDingtalkUserid("dt-fsup-" + label + "-" + suffix)
		}
		u, userErr := create.Save(ctx)
		if userErr != nil {
			t.Fatalf("创建用户 %s: %v", label, userErr)
		}
		return u
	}
	newMembership := func(u *ent.User) *ent.Membership {
		m, memberErr := data.db.Membership.Create().SetUserID(u.ID).SetOrganizationID(org.ID).SetEnabled(true).Save(ctx)
		if memberErr != nil {
			t.Fatalf("创建成员关系: %v", memberErr)
		}
		return m
	}
	requester := newUser("发起人", true)
	newMembership(requester)
	employee := newUser("员工", true)
	// 审批人：目标组织 ORG 范围角色，真实持有 business.order.si.lock。
	approver := newUser("审批人", true)
	approverMembership := newMembership(approver)
	lockPerm, permErr := data.db.Permission.Create().
		SetKey("business.order.si.lock").
		SetName("海运进口订单锁定-补录测试-" + suffix).
		SetDescription("费用补录集成测试").
		SetGroup("order").
		Save(ctx)
	if permErr != nil {
		lockPerm, permErr = data.db.Permission.Query().Where(permissionent.KeyEQ("business.order.si.lock")).First(ctx)
		if permErr != nil {
			t.Fatalf("准备锁权限: %v", permErr)
		}
	}
	role, roleErr := data.db.Role.Create().
		SetOrganizationID(org.ID).
		SetCode("fsup_locker_" + suffix).
		SetName("补录锁资格-" + suffix).
		AddPermissions(lockPerm).
		Save(ctx)
	if roleErr != nil {
		t.Fatalf("创建锁资格角色: %v", roleErr)
	}
	if _, assignErr := data.db.RoleAssignment.Create().SetMembershipID(approverMembership.ID).SetRoleID(role.ID).Save(ctx); assignErr != nil {
		t.Fatalf("分配锁资格角色: %v", assignErr)
	}

	return &feeSupplementPostgresFixture{
		t: t, data: data, ctx: ctx,
		organizationID: org.ID, customerID: customer.ID,
		requesterID: requester.ID, approverID: approver.ID, employeeID: employee.ID,
		feeSettingID: feeSetting.ID, billingUnitID: billingUnit.ID, chargeCategory: chargeCategory.ID,
		lockRoleID:     role.ID,
		usecase:        biz.NewOrderFeeSupplementUsecase(NewOrderFeeSupplementRepo(data)),
		commissionRepo: NewCommissionRepo(data),
		suffix:         suffix,
	}
}

// createOrder 创建 SI 测试订单并挂上费用大类服务类型。
func (f *feeSupplementPostgresFixture) createOrder(orderNo string) *ent.Order {
	f.t.Helper()
	order, err := f.data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(f.organizationID).
		SetOrderNo(orderNo).
		SetCustomerID(f.customerID).
		SetBusinessType(orderent.BusinessTypeSI).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建订单 %s: %v", orderNo, err)
	}
	if _, serviceErr := f.data.db.OrderServiceType.Create().SetOrderID(order.ID).SetMasterDataItemID(f.chargeCategory).Save(f.ctx); serviceErr != nil {
		f.t.Fatalf("创建订单服务类型: %v", serviceErr)
	}
	return order
}

// lockOrder 直接锁定订单行（业务锁代次 1），不重复验证锁定事务本身。
func (f *feeSupplementPostgresFixture) lockOrder(order *ent.Order) {
	f.t.Helper()
	if _, err := f.data.db.Order.UpdateOne(order).
		SetLockedAt(time.Now().UTC()).
		SetLockedBy(f.approverID).
		SetLockGeneration(1).
		SetLockSource(orderent.LockSourceMANUAL).
		SetVersion(2).
		Save(f.ctx); err != nil {
		f.t.Fatalf("锁定订单: %v", err)
	}
}

func (f *feeSupplementPostgresFixture) requesterPrincipal() *biz.Principal {
	return &biz.Principal{UserID: f.requesterID, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: f.organizationID, Kind: biz.OrganizationKindCompany}}}
}

// setT 把断言目标切换到当前子测试，避免子测试通过父测试 t 触发 FailNow。
func (f *feeSupplementPostgresFixture) setT(t *testing.T) {
	f.t = t
}

func (f *feeSupplementPostgresFixture) approverPrincipal() *biz.Principal {
	return &biz.Principal{UserID: f.approverID, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: f.organizationID, Kind: biz.OrganizationKindCompany}}}
}

func (f *feeSupplementPostgresFixture) createConfirmedCommission(order *ent.Order, commissionAmount, allocatedCost, realizedRevenue string) *ent.FinanceCommission {
	f.t.Helper()
	return f.insertCommission(order, commissionAmount, allocatedCost, realizedRevenue, nil, nil, biz.CommissionBasisRealizedProfit)
}

// createConfirmedCommissionWithSnapshot 插入带 READY + NATIVE 复算快照的
// CONFIRMED 提成行：历史总应收等于冻结已实现收入，总应付为指定快照。
func (f *feeSupplementPostgresFixture) createConfirmedCommissionWithSnapshot(order *ent.Order, commissionAmount, ratePercent, realizedRevenue, totalPayableSnapshot string) *ent.FinanceCommission {
	f.t.Helper()
	receivable := decimal.RequireFromString(realizedRevenue)
	payable := decimal.RequireFromString(totalPayableSnapshot)
	return f.insertCommission(order, commissionAmount, payable.StringFixed(8), realizedRevenue, &receivable, &payable, biz.CommissionBasisRealizedProfit)
}

func (f *feeSupplementPostgresFixture) insertCommission(order *ent.Order, commissionAmount, allocatedCost, realizedRevenue string, receivableSnapshot, payableSnapshot *decimal.Decimal, basis biz.CommissionCalculationBasis) *ent.FinanceCommission {
	f.t.Helper()
	sourceID, personnelRole := f.supplementCommissionSourceFixture(order)
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	commissionNo := "FC-FSUP-" + suffix
	// 数据库 CHECK 要求规则快照三元组完整：rule_id 与 rule_name 同时非空。
	rule, ruleErr := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName("补录测试规则-" + suffix).
		SetPersonnelRole(commissionruleent.PersonnelRole(personnelRole)).
		SetCalculationBasis(commissionruleent.CalculationBasis(basis)).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(f.ctx)
	if ruleErr != nil {
		f.t.Fatalf("插入提成规则: %v", ruleErr)
	}
	parent, err := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo(commissionNo).
		SetIdempotencyKey("fsup-commission-" + suffix).
		SetVerificationID(sourceID).
		SetVerificationNo("测试核销").
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName("补录测试规则-" + suffix).
		SetPersonnelRole(personnelRole).
		SetCalculationBasis(string(basis)).
		SetCalculationVersion(biz.CommissionCalculationVersion).
		SetSourceFingerprint(strings.Repeat("f", 64)).
		SetStatus(commissionent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetRealizedRevenue(realizedRevenue).
		SetAllocatedCost(allocatedCost).
		SetRealizedProfit(realizedRevenue).
		SetCommissionBaseAmount(realizedRevenue).
		SetRatePercent("10.0000").
		SetCommissionAmount(commissionAmount).
		SetCommissionDate("2026-09-01").
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate("2026-09-01").
		SetCnyCommissionAmount(commissionAmount).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入提成父单: %v", err)
	}
	savedFees, err := f.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(order.ID)).All(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	snapshot := make([]*biz.CommissionFeeDetail, 0, len(savedFees))
	for _, fee := range savedFees {
		snapshot = append(snapshot, &biz.CommissionFeeDetail{FeeID: fee.ID, Direction: string(fee.Direction), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: decimal.RequireFromString(fee.BaseCurrencyAmount)})
	}
	feeSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		f.t.Fatal(err)
	}
	lineCreate := f.data.db.FinanceCommissionLine.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(parent.ID).
		SetOrderID(order.ID).
		SetOrderNo(order.OrderNo).
		SetOrderDate("2026-08-01").
		SetCustomerID(f.customerID).
		SetCustomerCode("CUST").
		SetCustomerName("费用补录测试客户").
		SetPersonnelAssignmentID(uuid.New()).
		SetPersonnelOrganizationID(f.organizationID).
		SetPersonnelAssignedAt(time.Now().UTC()).
		SetFeeCount(1).
		SetFeeSnapshot(string(feeSnapshot)).
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetPersonnelRole(personnelRole).
		SetCalculationBasis(string(basis)).
		SetBaseCurrency("CNY").
		SetRealizedRevenue(realizedRevenue).
		SetAllocatedCost(allocatedCost).
		SetRealizedProfit(realizedRevenue).
		SetCommissionBaseAmount(realizedRevenue).
		SetRatePercent("10.0000").
		SetCommissionAmount(commissionAmount)
	if receivableSnapshot != nil && payableSnapshot != nil {
		lineCreate = lineCreate.
			SetTotalReceivableSnapshot(receivableSnapshot.StringFixed(8)).
			SetTotalPayableSnapshot(payableSnapshot.StringFixed(8)).
			SetSnapshotStatus(commissionlineent.SnapshotStatusREADY).
			SetSnapshotSource(commissionlineent.SnapshotSourceNATIVE)
	}
	if _, lineErr := lineCreate.Save(f.ctx); lineErr != nil {
		f.t.Fatalf("插入提成订单行: %v", lineErr)
	}
	return parent
}

func (f *feeSupplementPostgresFixture) commissionForOrder(orderID uuid.UUID) *ent.FinanceCommission {
	f.t.Helper()
	line, err := f.data.db.FinanceCommissionLine.Query().Where(commissionlineent.OrderIDEQ(orderID)).WithCommission().First(f.ctx)
	if err != nil {
		f.t.Fatalf("读取订单提成行: %v", err)
	}
	return line.Edges.Commission
}

// createConfirmedDecrease 插入一条 CONFIRMED 冲减调整（改变证据集合 / 释放净额）。
func (f *feeSupplementPostgresFixture) createConfirmedDecrease(order *ent.Order, parent *ent.FinanceCommission, amount, key string) {
	f.t.Helper()
	if _, err := f.data.db.FinanceCommissionAdjustment.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(parent.ID).
		SetOrderID(order.ID).
		SetAdjustmentNo(parent.CommissionNo + "-ADJ9" + key[len(key)-1:]).
		SetIdempotencyKey(key).
		SetCommissionNo(parent.CommissionNo).
		SetOrderNo(order.OrderNo).
		SetEmployeeID(parent.EmployeeID).
		SetEmployeeName(parent.EmployeeName).
		SetSourceType(commissionadjustmentent.SourceTypeMANUAL).
		SetDirection(commissionadjustmentent.DirectionDECREASE).
		SetStatus(commissionadjustmentent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetAmount(amount).
		SetReason("证据集合调整测试").
		SetVersion(1).
		Save(f.ctx); err != nil {
		f.t.Fatalf("插入确认冲减调整: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 场景 1：创建入口的锁判定、审批人解析与幂等
// ---------------------------------------------------------------------------

func (f *feeSupplementPostgresFixture) insertDraftDecrease(order *ent.Order, parent *ent.FinanceCommission, amount, key string) *ent.FinanceCommissionAdjustment {
	f.t.Helper()
	adjustment, err := f.data.db.FinanceCommissionAdjustment.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(parent.ID).
		SetOrderID(order.ID).
		SetAdjustmentNo(parent.CommissionNo + "-ADJ99" + key[len(key)-1:]).
		SetIdempotencyKey(key).
		SetCommissionNo(parent.CommissionNo).
		SetOrderNo(order.OrderNo).
		SetEmployeeID(parent.EmployeeID).
		SetEmployeeName(parent.EmployeeName).
		SetSourceType(commissionadjustmentent.SourceTypeMANUAL).
		SetDirection(commissionadjustmentent.DirectionDECREASE).
		SetStatus(commissionadjustmentent.StatusDRAFT).
		SetBaseCurrency("CNY").
		SetAmount(amount).
		SetReason("并发确认测试").
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入 DRAFT 冲减: %v", err)
	}
	return adjustment
}

// insertCommissionWithLineContext 插入指定计算版本/计提口径/行本位币与快照的
// CONFIRMED 提成，供成本敏感路由失败关闭测试使用；行上下文字段不可变，直接按
// 上下文插入。
func (f *feeSupplementPostgresFixture) insertCommissionWithLineContext(order *ent.Order, calculationVersion string, basis biz.CommissionCalculationBasis, lineBaseCurrency string, withSnapshot bool) *ent.FinanceCommission {
	f.t.Helper()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	amount := "60.00000000"
	if basis == biz.CommissionBasisRealizedRevenue {
		amount = "100.00000000"
	}
	rule, ruleErr := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName("补录路由规则-" + suffix).
		SetPersonnelRole(commissionruleent.PersonnelRoleSALES).
		SetCalculationBasis(commissionruleent.CalculationBasis(basis)).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(f.ctx)
	if ruleErr != nil {
		f.t.Fatalf("插入路由规则: %v", ruleErr)
	}
	sourceID, personnelRole := f.supplementCommissionSourceFixture(order)
	parent, err := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo("FC-FSUP-" + suffix).
		SetIdempotencyKey("fsup-route-" + suffix).
		SetVerificationID(sourceID).
		SetVerificationNo("测试核销").
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName("补录路由规则-" + suffix).
		SetPersonnelRole(personnelRole).
		SetCalculationBasis(string(basis)).
		SetCalculationVersion(calculationVersion).
		SetSourceFingerprint(strings.Repeat("f", 64)).
		SetStatus(commissionent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetRealizedRevenue("1000.00000000").
		SetAllocatedCost("400.00000000").
		SetRealizedProfit("1000.00000000").
		SetCommissionBaseAmount("1000.00000000").
		SetRatePercent("10.0000").
		SetCommissionAmount(amount).
		SetCommissionDate("2026-09-01").
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate("2026-09-01").
		SetCnyCommissionAmount(amount).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入路由提成父单: %v", err)
	}
	savedFees, err := f.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(order.ID)).All(f.ctx)
	if err != nil {
		f.t.Fatal(err)
	}
	snapshot := make([]*biz.CommissionFeeDetail, 0, len(savedFees))
	for _, fee := range savedFees {
		snapshot = append(snapshot, &biz.CommissionFeeDetail{FeeID: fee.ID, Direction: string(fee.Direction), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: decimal.RequireFromString(fee.BaseCurrencyAmount)})
	}
	feeSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		f.t.Fatal(err)
	}
	lineCreate := f.data.db.FinanceCommissionLine.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(parent.ID).
		SetOrderID(order.ID).
		SetOrderNo(order.OrderNo).
		SetOrderDate("2026-08-01").
		SetCustomerID(f.customerID).
		SetCustomerCode("CUST").
		SetCustomerName("费用补录测试客户").
		SetPersonnelAssignmentID(uuid.New()).
		SetPersonnelOrganizationID(f.organizationID).
		SetPersonnelAssignedAt(time.Now().UTC()).
		SetFeeCount(1).
		SetFeeSnapshot(string(feeSnapshot)).
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetPersonnelRole(personnelRole).
		SetCalculationBasis(string(basis)).
		SetBaseCurrency(lineBaseCurrency).
		SetRealizedRevenue("1000.00000000").
		SetAllocatedCost("400.00000000").
		SetRealizedProfit("1000.00000000").
		SetCommissionBaseAmount("1000.00000000").
		SetRatePercent("10.0000").
		SetCommissionAmount(amount)
	if withSnapshot {
		lineCreate = lineCreate.
			SetTotalReceivableSnapshot("1000.00000000").
			SetTotalPayableSnapshot("400.00000000").
			SetSnapshotStatus(commissionlineent.SnapshotStatusREADY).
			SetSnapshotSource(commissionlineent.SnapshotSourceNATIVE)
	}
	if _, lineErr := lineCreate.Save(f.ctx); lineErr != nil {
		f.t.Fatalf("插入路由提成行: %v", lineErr)
	}
	return parent
}

// approveInitialFees 构造基础费用与明确的人工订单审核事实。
func (f *feeSupplementPostgresFixture) approveInitialFees(order *ent.Order) {
	for _, row := range []struct {
		direction orderfeeent.Direction
		amount    string
	}{{orderfeeent.DirectionRECEIVABLE, "1000"}, {orderfeeent.DirectionPAYABLE, "400"}} {
		_, err := f.data.db.OrderFee.Create().SetOrderID(order.ID).SetIdempotencyKey(uuid.NewString()).SetDirection(row.direction).SetFeeCode("BASE").SetFeeName("原费用").SetSettlementPartyID(f.customerID).SetBillingUnit("票").SetQuantity("1").SetUnitPrice(row.amount).SetTotalAmount(row.amount).SetTaxInclusive(true).SetNetAmount(row.amount).SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-01").SetBaseCurrency("CNY").SetBaseCurrencyAmount(row.amount).SetExpenseDate("2026-09-01").Save(f.ctx)
		if err != nil {
			f.t.Fatal(err)
		}
	}
	seedManualOrderReviewForTest(f.t, f.data, order.ID, f.approverID)
}
