package data

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	commissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	"github.com/shopspring/decimal"
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
		SetKind("headquarters").
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
		m, memberErr := data.db.Membership.Create().SetUserID(u.ID).SetOrganizationID(org.ID).SetPrimary(true).SetEnabled(true).Save(ctx)
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
		SetDataScope(roleent.DataScopeOrganization).
		AddPermissions(lockPerm).
		Save(ctx)
	if roleErr != nil {
		t.Fatalf("创建锁资格角色: %v", roleErr)
	}
	if _, assignErr := data.db.RoleAssignment.Create().SetMembershipID(approverMembership.ID).SetRoleID(role.ID).Save(ctx); assignErr != nil {
		t.Fatalf("分配锁资格角色: %v", assignErr)
	}

	exchangeRate := biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), NewExchangeRateQuoteProvider())
	feeUsecase := biz.NewOrderFeeUsecase(NewOrderFeeRepo(data), exchangeRate, nil, nil, nil, nil)
	return &feeSupplementPostgresFixture{
		t: t, data: data, ctx: ctx,
		organizationID: org.ID, customerID: customer.ID,
		requesterID: requester.ID, approverID: approver.ID, employeeID: employee.ID,
		feeSettingID: feeSetting.ID, billingUnitID: billingUnit.ID, chargeCategory: chargeCategory.ID,
		lockRoleID:     role.ID,
		usecase:        biz.NewOrderFeeSupplementUsecase(NewOrderFeeSupplementRepo(data), feeUsecase, data),
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

// supplementInput 构造一笔正数应付补录入参。
func (f *feeSupplementPostgresFixture) supplementInput(idempotencyKey string) *biz.OrderFeeSupplementCreateInput {
	return &biz.OrderFeeSupplementCreateInput{
		Direction:         biz.OrderFeePayable,
		FeeSettingID:      f.feeSettingID,
		SettlementPartyID: f.customerID,
		BillingUnitID:     f.billingUnitID,
		Quantity:          decimal.RequireFromString("1"),
		UnitPrice:         decimal.RequireFromString("100"),
		Currency:          "CNY",
		ExpenseDate:       "2026-09-01",
		TaxInclusive:      true,
		Reason:            "漏录的拖车成本",
		IdempotencyKey:    idempotencyKey,
	}
}

func (f *feeSupplementPostgresFixture) requesterPrincipal() *biz.Principal {
	return &biz.Principal{UserID: f.requesterID, Organization: biz.Organization{ID: f.organizationID}}
}

// setT 把断言目标切换到当前子测试，避免子测试通过父测试 t 触发 FailNow。
func (f *feeSupplementPostgresFixture) setT(t *testing.T) {
	f.t = t
}

func (f *feeSupplementPostgresFixture) approverPrincipal() *biz.Principal {
	return &biz.Principal{UserID: f.approverID, Organization: biz.Organization{ID: f.organizationID}}
}

// createSupplement 由发起人发起一笔补录申请并断言成功。
func (f *feeSupplementPostgresFixture) createSupplement(order *ent.Order, key string) *biz.OrderFeeSupplementRequest {
	f.t.Helper()
	created, err := f.usecase.Create(f.ctx, f.requesterPrincipal(), f.organizationID, order.ID, f.supplementInput(key), false)
	if err != nil {
		f.t.Fatalf("发起补录申请: %v", err)
	}
	return created
}

// approveSupplement 由审批人审批通过。
func (f *feeSupplementPostgresFixture) approveSupplement(order *ent.Order, requestID uuid.UUID, version uint64) *biz.OrderFeeSupplementApproveResult {
	f.t.Helper()
	result, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, requestID, version)
	if err != nil {
		f.t.Fatalf("审批补录申请: %v", err)
	}
	return result
}

func (f *feeSupplementPostgresFixture) approveExpectError(order *ent.Order, requestID uuid.UUID, version uint64) error {
	_, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, requestID, version)
	return err
}

func (f *feeSupplementPostgresFixture) requestByID(id uuid.UUID) *ent.OrderFeeSupplementRequest {
	f.t.Helper()
	item, err := f.data.db.OrderFeeSupplementRequest.Query().Where(orderfeesupplementent.IDEQ(id)).Only(f.ctx)
	if err != nil {
		f.t.Fatalf("读取补录申请: %v", err)
	}
	return item
}

func (f *feeSupplementPostgresFixture) feesForRequest(requestID uuid.UUID) []*ent.OrderFee {
	f.t.Helper()
	items, err := f.data.db.OrderFee.Query().Where(orderfeeent.SupplementRequestIDEQ(requestID)).All(f.ctx)
	if err != nil {
		f.t.Fatalf("读取补录生成费用: %v", err)
	}
	return items
}

func (f *feeSupplementPostgresFixture) adjustmentsForRequest(requestID uuid.UUID) []*ent.FinanceCommissionAdjustment {
	f.t.Helper()
	items, err := f.data.db.FinanceCommissionAdjustment.Query().Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(requestID)).Order(commissionadjustmentent.ByID()).All(f.ctx)
	if err != nil {
		f.t.Fatalf("读取补录关联调整: %v", err)
	}
	return items
}

func (f *feeSupplementPostgresFixture) taskTotal() int {
	f.t.Helper()
	count, err := f.data.db.BackgroundTask.Query().Count(f.ctx)
	if err != nil {
		f.t.Fatalf("统计后台任务: %v", err)
	}
	return count
}

// approvalTaskCount 统计目标申请的待审批通知任务数（幂等收敛断言）。
func (f *feeSupplementPostgresFixture) approvalTaskCount(requestID uuid.UUID) int {
	f.t.Helper()
	tasks, err := f.data.db.BackgroundTask.Query().
		Where(backgroundtaskent.IdempotencyKeyHasPrefix("dingtalk-notice:")).
		WithNotificationDelivery().
		All(f.ctx)
	if err != nil {
		f.t.Fatalf("统计待审批通知任务: %v", err)
	}
	total := 0
	for _, task := range tasks {
		delivery := task.Edges.NotificationDelivery
		if delivery != nil && delivery.Template == notificationent.TemplateFEE_SUPPLEMENT_APPROVAL_PENDING && delivery.ResourceID == requestID {
			total++
		}
	}
	return total
}

// decreaseTaskCount 统计目标申请生成建议的员工知情通知任务数。
func (f *feeSupplementPostgresFixture) decreaseTaskCount(requestID uuid.UUID) int {
	f.t.Helper()
	adjustmentIDs := make([]uuid.UUID, 0)
	for _, adjustment := range f.adjustmentsForRequest(requestID) {
		adjustmentIDs = append(adjustmentIDs, adjustment.ID)
	}
	if len(adjustmentIDs) == 0 {
		return 0
	}
	deliveries, err := f.data.db.NotificationDelivery.Query().
		Where(notificationent.TemplateEQ(notificationent.TemplateCOMMISSION_DECREASE_SUGGESTED)).
		All(f.ctx)
	if err != nil {
		f.t.Fatalf("统计知情通知明细: %v", err)
	}
	total := 0
	for _, delivery := range deliveries {
		for _, adjustmentID := range adjustmentIDs {
			if delivery.ResourceID == adjustmentID {
				total++
			}
		}
	}
	return total
}

// createConfirmedCommission 为订单插入一张 CONFIRMED 提成父单与订单行（不含
// 复算快照），返回父单。提成金额即有效净额（财务锁净额口径）。
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
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	commissionNo := "FC-FSUP-" + suffix
	// 数据库 CHECK 要求规则快照三元组完整：rule_id 与 rule_name 同时非空。
	rule, ruleErr := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName("补录测试规则-" + suffix).
		SetPersonnelRole("SALES").
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
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName("补录测试规则-" + suffix).
		SetPersonnelRole("SALES").
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
		SetFeeSnapshot("[]").
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetPersonnelRole("SALES").
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

func TestFeeSupplementCreateGatePostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)

	t.Run("业务锁与财务锁均不存在时拒绝并零写入", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-OPEN-" + fixture.suffix)
		beforeTasks := fixture.taskTotal()
		if _, err := fixture.usecase.Create(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, fixture.supplementInput("no-lock"), false); err != biz.ErrFeeSupplementNotApplicable {
			t.Fatalf("无锁订单应稳定拒绝并引导普通新增，实际 %v", err)
		}
		requests, requestErr := fixture.data.db.OrderFeeSupplementRequest.Query().Where(orderfeesupplementent.OrderIDEQ(order.ID)).Count(fixture.ctx)
		if requestErr != nil || requests != 0 {
			t.Fatalf("无锁拒绝必须零写入申请: %d %v", requests, requestErr)
		}
		if after := fixture.taskTotal(); after != beforeTasks {
			t.Fatalf("无锁拒绝不得入队通知: before=%d after=%d", beforeTasks, after)
		}
	})

	t.Run("仅业务锁订单固化为 BUSINESS 并逐审批人通知", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-BIZ-" + fixture.suffix)
		fixture.lockOrder(order)
		created := fixture.createSupplement(order, "biz-lock")
		if created.Status != biz.OrderFeeSupplementPending || created.LockBasis != biz.SupplementLockBasisBusiness {
			t.Fatalf("业务锁申请状态或依据不符: %s %s", created.Status, created.LockBasis)
		}
		if created.BusinessLockGeneration == nil || *created.BusinessLockGeneration != 1 {
			t.Fatalf("业务锁代次必须固化: %v", created.BusinessLockGeneration)
		}
		if created.FinancialLockEvidenceHash != nil || created.FinancialLockEvidenceVersion != nil {
			t.Fatalf("仅业务锁不得携带财务证据字段")
		}
		if count := fixture.approvalTaskCount(created.ID); count != 1 {
			t.Fatalf("审批人待审批通知应为 1 条，实际 %d", count)
		}
		// 同幂等键同指纹语义重放返回原申请，不重复入队通知。
		replayed, replayErr := fixture.usecase.Create(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, fixture.supplementInput("biz-lock"), false)
		if replayErr != nil || replayed.ID != created.ID {
			t.Fatalf("同键同指纹应返回原申请: %v", replayErr)
		}
		if count := fixture.approvalTaskCount(created.ID); count != 1 {
			t.Fatalf("语义重放不得重复入队通知，实际 %d", count)
		}
		// 同幂等键不同指纹返回稳定冲突。
		conflicting := fixture.supplementInput("biz-lock")
		conflicting.UnitPrice = decimal.RequireFromString("120")
		if _, err := fixture.usecase.Create(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, conflicting, false); err != biz.ErrFeeSupplementIdempotencyConflict {
			t.Fatalf("同键异指纹应返回幂等冲突，实际 %v", err)
		}
		// 应收方向双重拒绝。
		receivable := fixture.supplementInput("biz-receivable")
		receivable.Direction = biz.OrderFeeReceivable
		if _, err := fixture.usecase.Create(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, receivable, false); err != biz.ErrFeeSupplementInvalidArgument {
			t.Fatalf("应收补录必须被领域边界拒绝，实际 %v", err)
		}
		// 确认申请入库时固化指纹并带审计；审批人资格快照（含未绑定钉钉者）
		// 必须在创建审计中留下持久痕迹。
		stored := fixture.requestByID(created.ID)
		if stored.RequestFingerprint != created.RequestFingerprint || stored.RequestFingerprint == "" {
			t.Fatalf("申请指纹必须由服务端固化: %s", stored.RequestFingerprint)
		}
		audit, auditErr := fixture.data.db.AuditLog.Query().Where(auditent.ActionEQ("order.fee_supplement.create"), auditent.ResourceIDEQ(created.ID.String())).First(fixture.ctx)
		if auditErr != nil {
			t.Fatalf("读取创建审计: %v", auditErr)
		}
		auditDetails := string(audit.Details)
		if !strings.Contains(auditDetails, fixture.approverID.String()) || !strings.Contains(auditDetails, "approver_snapshot") {
			t.Fatalf("创建审计必须包含审批人资格快照: %s", auditDetails)
		}
	})

	t.Run("提交时没有任何合格审批人则稳定拒绝且零写入", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-NOAPPR-" + fixture.suffix)
		// 撤销夹具锁资格角色的全部分配，构造「当前无可用审批人」。
		if _, deleteErr := fixture.data.db.RoleAssignment.Delete().Where(roleassignmentent.RoleIDEQ(fixture.lockRoleID)).Exec(fixture.ctx); deleteErr != nil {
			t.Fatalf("移除角色分配: %v", deleteErr)
		}
		fixture.lockOrder(order)
		if _, createErr := fixture.usecase.Create(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, fixture.supplementInput("no-approver"), false); createErr != biz.ErrFeeSupplementApproverUnavailable {
			t.Fatalf("无审批人应返回 FEE_SUPPLEMENT_APPROVER_UNAVAILABLE，实际 %v", createErr)
		}
		requests, requestErr := fixture.data.db.OrderFeeSupplementRequest.Query().Where(orderfeesupplementent.OrderIDEQ(order.ID)).Count(fixture.ctx)
		if requestErr != nil || requests != 0 {
			t.Fatalf("无审批人拒绝必须零写入: %d %v", requests, requestErr)
		}
	})
}

// ---------------------------------------------------------------------------
// 场景 2：审批创建 CONFIRMED 费用、驳回与撤回/审批竞争
// ---------------------------------------------------------------------------

func TestFeeSupplementDecisionPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)

	t.Run("审批通过创建且只创建一条 CONFIRMED 费用且订单保持锁定", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-APPR-" + fixture.suffix)
		fixture.lockOrder(order)
		created := fixture.createSupplement(order, "approve-1")
		result := fixture.approveSupplement(order, created.ID, created.Version)
		if result.Request.Status != biz.OrderFeeSupplementApproved || result.Request.Version != 2 || result.Request.DecidedBy == nil || *result.Request.DecidedBy != fixture.approverID {
			t.Fatalf("审批终态投影不符: %+v", result.Request)
		}
		fees := fixture.feesForRequest(created.ID)
		if len(fees) != 1 {
			t.Fatalf("审批必须生成且只生成一条费用，实际 %d", len(fees))
		}
		fee := fees[0]
		if fee.Status != orderfeeent.StatusCONFIRMED || fee.Direction != orderfeeent.DirectionPAYABLE || fee.Version != 1 {
			t.Fatalf("补录费用必须为 CONFIRMED 应付: %+v", fee)
		}
		if fee.IdempotencyKey != "fee-supplement:"+created.ID.String() {
			t.Fatalf("补录费用幂等键必须由申请 ID 派生: %s", fee.IdempotencyKey)
		}
		if fee.ExpenseDate != "2026-09-01" || fee.ExchangeRate != "1.00000000" {
			t.Fatalf("补录费用必须保留发生日期与发生日汇率: %s %s", fee.ExpenseDate, fee.ExchangeRate)
		}
		// 重复审批稳定失败且不重复入账。
		if _, err := fixture.usecase.Approve(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, 1); err != biz.ErrFeeSupplementTransition {
			t.Fatalf("重复审批应返回状态冲突，实际 %v", err)
		}
		if _, err := fixture.usecase.Approve(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, result.Request.Version); err != biz.ErrFeeSupplementTransition {
			t.Fatalf("终态后审批应返回状态冲突，实际 %v", err)
		}
		if count := len(fixture.feesForRequest(created.ID)); count != 1 {
			t.Fatalf("重复审批不得重复入账: %d", count)
		}
		reloaded, reloadErr := fixture.data.db.Order.Query().Where(orderent.IDEQ(order.ID)).Only(fixture.ctx)
		if reloadErr != nil || reloaded.LockedAt == nil {
			t.Fatalf("审批不得改变订单锁定状态: %v", reloadErr)
		}
	})

	t.Run("驳回只写终态不产生费用", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-REJ-" + fixture.suffix)
		fixture.lockOrder(order)
		created := fixture.createSupplement(order, "reject-1")
		reason := "金额与对账单不符"
		rejected, err := fixture.usecase.Reject(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, created.Version, &reason)
		if err != nil || rejected.Status != biz.OrderFeeSupplementRejected {
			t.Fatalf("驳回失败: %v %+v", err, rejected)
		}
		if len(fixture.feesForRequest(created.ID)) != 0 || len(fixture.adjustmentsForRequest(created.ID)) != 0 {
			t.Fatalf("驳回不得产生费用或调整")
		}
	})

	t.Run("仅发起人可按版本撤回且与审批并发只有一个成功", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-WD-" + fixture.suffix)
		fixture.lockOrder(order)
		created := fixture.createSupplement(order, "withdraw-race")
		// 非发起人不可撤回且不可见。
		if _, err := fixture.usecase.Withdraw(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, created.Version); err != biz.ErrFeeSupplementNotFound {
			t.Fatalf("非发起人撤回应返回不存在，实际 %v", err)
		}
		// 版本不匹配稳定冲突。
		if _, err := fixture.usecase.Withdraw(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, created.ID, created.Version+5); err != biz.ErrFeeSupplementTransition {
			t.Fatalf("版本不匹配撤回应返回状态冲突，实际 %v", err)
		}
		// 并发撤回与审批：同一申请行锁内竞争，只有一个状态迁移成功。
		var wg sync.WaitGroup
		withdrawErr := make(chan error, 1)
		approveErr := make(chan error, 1)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := fixture.usecase.Withdraw(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, created.ID, created.Version)
			withdrawErr <- err
		}()
		go func() {
			defer wg.Done()
			_, err := fixture.usecase.Approve(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, created.Version)
			approveErr <- err
		}()
		wg.Wait()
		withdrawFailure := <-withdrawErr
		approveFailure := <-approveErr
		succeeded := 0
		if withdrawFailure == nil {
			succeeded++
		}
		if approveFailure == nil {
			succeeded++
		}
		if succeeded != 1 {
			t.Fatalf("撤回与审批并发必须只有一个成功: withdraw=%v approve=%v", withdrawFailure, approveFailure)
		}
		final := fixture.requestByID(created.ID)
		fees := fixture.feesForRequest(created.ID)
		adjustments := fixture.adjustmentsForRequest(created.ID)
		if final.Status == orderfeesupplementent.StatusWITHDRAWN {
			if withdrawFailure != nil || approveFailure == nil {
				t.Fatalf("撤回成功时审批必须以状态冲突失败: %v", approveFailure)
			}
			if len(fees) != 0 || len(adjustments) != 0 {
				t.Fatalf("撤回成功不得创建费用或调整: %d %d", len(fees), len(adjustments))
			}
		} else if final.Status == orderfeesupplementent.StatusAPPROVED {
			if approveFailure != nil || withdrawFailure == nil {
				t.Fatalf("审批成功时撤回必须以状态冲突失败: %v", withdrawFailure)
			}
			if len(fees) != 1 {
				t.Fatalf("审批成功必须恰好生成一条费用: %d", len(fees))
			}
		} else {
			t.Fatalf("并发竞争后申请终态不符: %s", final.Status)
		}
	})
}

// ---------------------------------------------------------------------------
// 场景 3：财务锁证据固化与审批复核（CONFIRMED↔PAID 不变证据、集合变化拒绝）
// ---------------------------------------------------------------------------

func TestFeeSupplementFinancialEvidencePostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)

	t.Run("CONFIRMED 转 PAID 不改变证据且审批仍可通过", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-FIN1-" + fixture.suffix)
		fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
		created := fixture.createSupplement(order, "fin-evidence-1")
		if created.LockBasis != biz.SupplementLockBasisFinancial {
			t.Fatalf("仅财务锁应固化为 FINANCIAL: %s", created.LockBasis)
		}
		if created.FinancialLockNetAmount == nil || created.FinancialLockNetAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("财务锁净额快照不符: %v", created.FinancialLockNetAmount)
		}
		// CONFIRMED → PAID 生命周期流转不改变证据，审批复核仍通过。
		commission := fixture.commissionForOrder(order.ID)
		if _, err := fixture.commissionRepo.Transition(fixture.ctx, fixture.organizationID, commission.ID, fixture.approverID, 1, biz.CommissionPaid, "", &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.approverID, Action: "finance.commission.paid", Result: "success", ResourceType: "finance_commission", ResourceID: commission.ID.String()}); err != nil {
			fixture.t.Fatalf("提成转 PAID 失败: %v", err)
		}
		fixture.approveSupplement(order, created.ID, created.Version)
	})

	t.Run("参与集合变化时 FINANCIAL 申请因证据变更拒绝且提示重新申请", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-FIN2-" + fixture.suffix)
		commission := fixture.createConfirmedCommission(order, "50.00000000", "300.00000000", "900.00000000")
		created := fixture.createSupplement(order, "fin-evidence-2")
		// 新增一条 CONFIRMED 冲减调整改变证据集合（净额仍大于零）。
		fixture.createConfirmedDecrease(order, commission, "10.00000000", "fin-evidence-2-adj")
		err := fixture.approveExpectError(order, created.ID, created.Version)
		if err == nil || !strings.Contains(err.Error(), "重新发起补录") {
			t.Fatalf("证据变更拒绝必须提示重新发起补录: %v", err)
		}
		if len(fixture.feesForRequest(created.ID)) != 0 {
			t.Fatalf("证据变更拒绝必须零写入费用")
		}
	})

	t.Run("财务锁释放后旧申请不得承接并提示普通新增", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-FIN3-" + fixture.suffix)
		commission := fixture.createConfirmedCommission(order, "30.00000000", "200.00000000", "800.00000000")
		created := fixture.createSupplement(order, "fin-evidence-3")
		// 全额冲减释放财务锁（净额归零）。
		fixture.createConfirmedDecrease(order, commission, "30.00000000", "fin-evidence-3-adj")
		err := fixture.approveExpectError(order, created.ID, created.Version)
		if err == nil || !strings.Contains(err.Error(), "普通费用新增") {
			t.Fatalf("财务锁释放后应提示改走普通新增: %v", err)
		}
	})

	t.Run("无实时 lock grant 不可审批", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-GRANT-" + fixture.suffix)
		fixture.lockOrder(order)
		created := fixture.createSupplement(order, "grant-check")
		if _, err := fixture.usecase.Approve(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, created.ID, created.Version); err != biz.ErrPermissionDenied {
			t.Fatalf("无 lock grant 用户审批应被拒绝，实际 %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// 场景 4：边际影响计算与专用作废
// ---------------------------------------------------------------------------

func TestFeeSupplementImpactAndCancelPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)

	t.Run("审批生成冲减建议且重复补录不重复计算历史成本", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-IMPACT-" + fixture.suffix)
		fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
		first := fixture.createSupplement(order, "impact-1")
		fixture.approveSupplement(order, first.ID, first.Version)
		// before: 应付 400 → 利润 600 → 提成 60；after 补录 100 → 应付 500 → 提成 50；差 10。
		adjustments := fixture.adjustmentsForRequest(first.ID)
		if len(adjustments) != 1 {
			t.Fatalf("应生成一条冲减建议，实际 %d", len(adjustments))
		}
		adjustment := adjustments[0]
		if adjustment.Direction != commissionadjustmentent.DirectionDECREASE || adjustment.Status != commissionadjustmentent.StatusDRAFT {
			t.Fatalf("冲减建议必须为 DECREASE + DRAFT: %+v", adjustment)
		}
		if adjustment.SourceType != commissionadjustmentent.SourceTypeLOCKED_FEE_SUPPLEMENT {
			t.Fatalf("来源必须为 LOCKED_FEE_SUPPLEMENT: %s", adjustment.SourceType)
		}
		if amount, amountErr := decimalOf(adjustment.Amount); amountErr != nil || amount.StringFixed(8) != "10.00000000" {
			t.Fatalf("边际差额应为 10，实际 %s (%v)", adjustment.Amount, amountErr)
		}
		if !strings.HasSuffix(adjustment.AdjustmentNo, "-ADJ001") {
			t.Fatalf("调整号必须挂在原提成单行内序号: %s", adjustment.AdjustmentNo)
		}
		if adjustment.EmployeeID != fixture.employeeID {
			t.Fatalf("建议必须挂回原提成员工: %s", adjustment.EmployeeID)
		}
		if count := fixture.decreaseTaskCount(first.ID); count != 1 {
			t.Fatalf("员工知情通知应为 1 条，实际 %d", count)
		}
		// 第二笔补录：此前未作废补录 100 进入 before 基线（应付 500、提成 50），
		// 计入本笔后应付 600、提成 40，本笔增量差额 10；两笔合计 20 等于真实总
		// 影响，前笔已建议的 10 不得重复计入。
		second := fixture.createSupplement(order, "impact-2")
		fixture.approveSupplement(order, second.ID, second.Version)
		secondAdjustments := fixture.adjustmentsForRequest(second.ID)
		if len(secondAdjustments) != 1 {
			t.Fatalf("第二笔补录应生成一条建议: %+v", secondAdjustments)
		}
		if amount, amountErr := decimalOf(secondAdjustments[0].Amount); amountErr != nil || amount.StringFixed(8) != "10.00000000" {
			t.Fatalf("第二笔补录边际差额应为 10（只计算本笔增量）: %+v (%v)", secondAdjustments, amountErr)
		}
	})

	t.Run("前笔建议被忽略后第二笔仍不得重复计算前笔成本", func(t *testing.T) {
		order := fixture.createOrder("FSUP-IGN-" + fixture.suffix)
		fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
		first := fixture.createSupplement(order, "ignore-1")
		fixture.approveSupplement(order, first.ID, first.Version)
		// 财务忽略 DRAFT 建议（复用通用取消，必填原因）；前笔费用仍是有效成本事实。
		adjustment := fixture.adjustmentsForRequest(first.ID)[0]
		if _, err := fixture.commissionRepo.TransitionAdjustment(fixture.ctx, fixture.organizationID, adjustment.ID, fixture.approverID, adjustment.Version, biz.CommissionCancelled, "财务忽略建议", &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.approverID, Action: "finance.commission_adjustment.cancelled", Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: adjustment.ID.String()}); err != nil {
			t.Fatalf("忽略建议: %v", err)
		}
		second := fixture.createSupplement(order, "ignore-2")
		fixture.approveSupplement(order, second.ID, second.Version)
		// before 基线含前笔补录应付（费用未作废），第二笔增量差额仍为 10。
		secondAdjustments := fixture.adjustmentsForRequest(second.ID)
		if len(secondAdjustments) != 1 {
			t.Fatalf("忽略前笔建议后第二笔应生成建议: %+v", secondAdjustments)
		}
		if amount, amountErr := decimalOf(secondAdjustments[0].Amount); amountErr != nil || amount.StringFixed(8) != "10.00000000" {
			t.Fatalf("建议被忽略不得导致重复计算前笔成本: %+v (%v)", secondAdjustments, amountErr)
		}
	})

	t.Run("前笔建议已确认后第二笔同样不重复计算", func(t *testing.T) {
		order := fixture.createOrder("FSUP-CFR-" + fixture.suffix)
		fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
		first := fixture.createSupplement(order, "confirm-1")
		fixture.approveSupplement(order, first.ID, first.Version)
		adjustment := fixture.adjustmentsForRequest(first.ID)[0]
		if _, err := fixture.commissionRepo.TransitionAdjustment(fixture.ctx, fixture.organizationID, adjustment.ID, fixture.approverID, adjustment.Version, biz.CommissionConfirmed, "", &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.approverID, Action: "finance.commission_adjustment.confirmed", Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: adjustment.ID.String()}); err != nil {
			t.Fatalf("确认建议: %v", err)
		}
		second := fixture.createSupplement(order, "confirm-2")
		fixture.approveSupplement(order, second.ID, second.Version)
		secondAdjustments := fixture.adjustmentsForRequest(second.ID)
		if len(secondAdjustments) != 1 {
			t.Fatalf("前笔已确认时第二笔应生成建议: %+v", secondAdjustments)
		}
		if amount, amountErr := decimalOf(secondAdjustments[0].Amount); amountErr != nil || amount.StringFixed(8) != "10.00000000" {
			t.Fatalf("前笔建议已确认后第二笔增量差额应为 10: %+v (%v)", secondAdjustments, amountErr)
		}
	})

	t.Run("专用作废原子取消费用与 DRAFT 建议且保持申请终态", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-CANCEL-" + fixture.suffix)
		fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
		created := fixture.createSupplement(order, "cancel-1")
		result := fixture.approveSupplement(order, created.ID, created.Version)
		feeVersion := result.Fee.Version
		cancelResult, err := fixture.usecase.CancelApprovedFee(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, feeVersion, "成本重复补录")
		if err != nil {
			fixture.t.Fatalf("专用作废失败: %v", err)
		}
		if cancelResult.Fee.Status != biz.OrderFeeCancelled || cancelResult.Fee.Version != feeVersion+1 || cancelResult.Fee.CancellationReason == nil || *cancelResult.Fee.CancellationReason != "成本重复补录" {
			t.Fatalf("费用作废投影不符: %+v", cancelResult.Fee)
		}
		if len(cancelResult.CancelledAdjustmentIDs) != 1 {
			t.Fatalf("DRAFT 建议应一并取消: %v", cancelResult.CancelledAdjustmentIDs)
		}
		final := fixture.requestByID(created.ID)
		if final.Status != orderfeesupplementent.StatusAPPROVED {
			t.Fatalf("APPROVED 申请必须保持终态: %s", final.Status)
		}
		adjustments := fixture.adjustmentsForRequest(created.ID)
		if adjustments[0].Status != commissionadjustmentent.StatusCANCELLED {
			t.Fatalf("关联 DRAFT 建议必须转为 CANCELLED: %s", adjustments[0].Status)
		}
		// 已作废补录不进后续成本基线：再次补录审批的边际差额回到 10。
		again := fixture.createSupplement(order, "cancel-again")
		fixture.approveSupplement(order, again.ID, again.Version)
		againAdjustments := fixture.adjustmentsForRequest(again.ID)
		if len(againAdjustments) != 1 {
			t.Fatalf("已作废补录后应重新生成建议: %+v", againAdjustments)
		}
		if amount, amountErr := decimalOf(againAdjustments[0].Amount); amountErr != nil || amount.StringFixed(8) != "10.00000000" {
			t.Fatalf("已作废补录不得进入成本基线: %+v (%v)", againAdjustments, amountErr)
		}
		// 再次专用作废稳定拒绝零写入。
		if _, cancelErr := fixture.usecase.CancelApprovedFee(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, created.ID, feeVersion+1, "重复作废"); cancelErr == nil {
			t.Fatalf("已作废费用必须拒绝再次专用作废")
		}
	})

	t.Run("存在更晚有效补录、曾确认冲减或版本竞争时零写入", func(t *testing.T) {
		fixture.setT(t)
		order := fixture.createOrder("FSUP-BLOCK-" + fixture.suffix)
		fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
		first := fixture.createSupplement(order, "block-1")
		fixture.approveSupplement(order, first.ID, first.Version)
		second := fixture.createSupplement(order, "block-2")
		secondResult := fixture.approveSupplement(order, second.ID, second.Version)
		// 更早一笔仍有效时作废必须按倒序拒绝。
		if _, err := fixture.usecase.CancelApprovedFee(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, first.ID, 1, "倒序校验"); err == nil || !strings.Contains(err.Error(), "更晚") {
			t.Fatalf("存在更晚有效补录必须拒绝作废早期补录: %v", err)
		}
		if len(fixture.feesForRequest(first.ID)) != 1 || fixture.feesForRequest(first.ID)[0].Status != orderfeeent.StatusCONFIRMED {
			t.Fatalf("被拒绝的作废必须零写入: %+v", fixture.feesForRequest(first.ID))
		}
		// 版本竞争稳定拒绝。
		if _, err := fixture.usecase.CancelApprovedFee(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, second.ID, secondResult.Fee.Version+3, "版本竞争"); err != biz.ErrOrderFeeVersionConflict {
			t.Fatalf("费用版本竞争必须返回版本冲突，实际 %v", err)
		}
		// 关联冲减曾进入 CONFIRMED 后禁止直接作废。
		adjustment := fixture.adjustmentsForRequest(second.ID)[0]
		if _, err := fixture.commissionRepo.TransitionAdjustment(fixture.ctx, fixture.organizationID, adjustment.ID, fixture.approverID, adjustment.Version, biz.CommissionConfirmed, "", &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.approverID, Action: "finance.commission_adjustment.confirmed", Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: adjustment.ID.String()}); err != nil {
			t.Fatalf("确认冲减建议: %v", err)
		}
		if _, cancelErr := fixture.usecase.CancelApprovedFee(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, second.ID, secondResult.Fee.Version, "已确认冲减"); cancelErr == nil || !strings.Contains(cancelErr.Error(), "已确认或已扣回") {
			t.Fatalf("曾确认冲减必须禁止直接作废: %v", cancelErr)
		}
		if reloaded := fixture.feesForRequest(second.ID); reloaded[0].Status != orderfeeent.StatusCONFIRMED {
			t.Fatalf("作废被拒绝后费用必须保持 CONFIRMED: %s", reloaded[0].Status)
		}
	})
}

// ---------------------------------------------------------------------------
// 场景 5：证据复核并发穿透、计算版本/币种失败关闭与收入口径不阻断
// ---------------------------------------------------------------------------

func TestFeeSupplementEvidenceConcurrencyPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)
	fixture.setT(t)
	order := fixture.createOrder("FSUP-RACE-" + fixture.suffix)
	commission := fixture.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.00", "1000.00000000", "400.00000000")
	created := fixture.createSupplement(order, "evidence-race")
	// 手工 DRAFT 调整并发确认与补录审批：确认会改变证据集合，二者以 Order 行锁
	// 线性化，只能一方先提交且结果可串行解释。
	draft := fixture.insertDraftDecrease(order, commission, "10.00000000", "evidence-race-draft")
	var wg sync.WaitGroup
	approveErr := make(chan error, 1)
	confirmErr := make(chan error, 1)
	wg.Add(2)
	go func() {
		defer wg.Done()
		approveErr <- fixture.approveExpectError(order, created.ID, created.Version)
	}()
	go func() {
		defer wg.Done()
		reloaded, reloadError := fixture.data.db.FinanceCommissionAdjustment.Query().Where(commissionadjustmentent.IDEQ(draft.ID)).Only(fixture.ctx)
		if reloadError != nil {
			confirmErr <- reloadError
			return
		}
		_, transitionError := fixture.commissionRepo.TransitionAdjustment(fixture.ctx, fixture.organizationID, reloaded.ID, fixture.approverID, reloaded.Version, biz.CommissionConfirmed, "", &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.approverID, Action: "finance.commission_adjustment.confirmed", Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: reloaded.ID.String()})
		confirmErr <- transitionError
	}()
	wg.Wait()
	approveFailure := <-approveErr
	confirmFailure := <-confirmErr
	if confirmFailure != nil {
		t.Fatalf("调整确认必须成功: %v", confirmFailure)
	}
	fees := fixture.feesForRequest(created.ID)
	if approveFailure == nil {
		// 审批先提交：确认后提交，费用已生成且审批通过。
		if len(fees) != 1 {
			t.Fatalf("审批成功必须恰好生成一条费用: %d", len(fees))
		}
	} else {
		// 确认先提交：证据集合变化必须令审批以 LOCK_BASIS_CHANGED 拒绝且零写入。
		if !strings.Contains(approveFailure.Error(), "LOCK_BASIS_CHANGED") && !strings.Contains(approveFailure.Error(), "重新发起补录") {
			t.Fatalf("证据被并发改写后审批必须以 LOCK_BASIS_CHANGED 拒绝: %v", approveFailure)
		}
		if len(fees) != 0 {
			t.Fatalf("证据变化拒绝必须零写入费用: %d", len(fees))
		}
		if len(fixture.adjustmentsForRequest(created.ID)) != 0 {
			t.Fatalf("证据变化拒绝不得生成建议")
		}
	}
	// 终态可串行解释：调整已确认，申请状态与费用事实一一对应。
	final := fixture.requestByID(created.ID)
	if approveFailure == nil && final.Status != orderfeesupplementent.StatusAPPROVED {
		t.Fatalf("审批成功后申请必须为 APPROVED: %s", final.Status)
	}
	if approveFailure != nil && final.Status != orderfeesupplementent.StatusPENDING {
		t.Fatalf("审批被拒后申请必须保持 PENDING: %s", final.Status)
	}
}

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
	parent, err := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo("FC-FSUP-" + suffix).
		SetIdempotencyKey("fsup-route-" + suffix).
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName("补录路由规则-" + suffix).
		SetPersonnelRole("SALES").
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
		SetCommissionAmount("60.00000000").
		SetCommissionDate("2026-09-01").
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate("2026-09-01").
		SetCnyCommissionAmount("60.00000000").
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入路由提成父单: %v", err)
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
		SetFeeSnapshot("[]").
		SetEmployeeID(f.employeeID).
		SetEmployeeName("补录员工-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis(string(basis)).
		SetBaseCurrency(lineBaseCurrency).
		SetRealizedRevenue("1000.00000000").
		SetAllocatedCost("400.00000000").
		SetRealizedProfit("1000.00000000").
		SetCommissionBaseAmount("1000.00000000").
		SetRatePercent("10.0000").
		SetCommissionAmount("60.00000000")
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

func TestFeeSupplementSensitivityRoutingPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)

	t.Run("未知计算版本失败关闭且零写入", func(t *testing.T) {
		order := fixture.createOrder("FSUP-VER-" + fixture.suffix)
		fixture.insertCommissionWithLineContext(order, "FUTURE_REALIZED_V9", biz.CommissionBasisRealizedProfit, "CNY", true)
		created := fixture.createSupplement(order, "version-unknown")
		err := fixture.approveExpectError(order, created.ID, created.Version)
		if err == nil || !strings.Contains(err.Error(), "COMMISSION_CALCULATION_VERSION_UNSUPPORTED") {
			t.Fatalf("未知计算版本必须失败关闭: %v", err)
		}
		if len(fixture.feesForRequest(created.ID)) != 0 {
			t.Fatalf("未知计算版本必须零写入费用")
		}
		final := fixture.requestByID(created.ID)
		if final.Status != orderfeesupplementent.StatusPENDING {
			t.Fatalf("失败关闭后申请必须保持 PENDING: %s", final.Status)
		}
	})

	t.Run("成本敏感行快照缺失阻断且币种不一致稳定冲突", func(t *testing.T) {
		order := fixture.createOrder("FSUP-SNAP-" + fixture.suffix)
		fixture.insertCommissionWithLineContext(order, biz.CommissionCalculationVersion, biz.CommissionBasisRealizedProfit, "CNY", false)
		created := fixture.createSupplement(order, "snapshot-missing")
		err := fixture.approveExpectError(order, created.ID, created.Version)
		if err == nil || !strings.Contains(err.Error(), "COMMISSION_SNAPSHOT_UNAVAILABLE") {
			t.Fatalf("快照缺失必须阻断审批: %v", err)
		}
		if len(fixture.feesForRequest(created.ID)) != 0 {
			t.Fatalf("快照缺失必须零写入费用")
		}

		order2 := fixture.createOrder("FSUP-CCY-" + fixture.suffix)
		fixture.insertCommissionWithLineContext(order2, biz.CommissionCalculationVersion, biz.CommissionBasisRealizedProfit, "USD", true)
		created2 := fixture.createSupplement(order2, "currency-mismatch")
		err2 := fixture.approveExpectError(order2, created2.ID, created2.Version)
		if err2 == nil || !strings.Contains(err2.Error(), "COMMISSION_BASE_CURRENCY_MISMATCH") {
			t.Fatalf("本位币不一致必须稳定冲突: %v", err2)
		}
		if len(fixture.feesForRequest(created2.ID)) != 0 {
			t.Fatalf("币种不一致必须零写入费用")
		}
	})

	t.Run("收入口径不因成本分母缺失阻断且不生成建议", func(t *testing.T) {
		order := fixture.createOrder("FSUP-REV-" + fixture.suffix)
		fixture.insertCommissionWithLineContext(order, biz.CommissionCalculationVersion, biz.CommissionBasisRealizedRevenue, "CNY", false)
		created := fixture.createSupplement(order, "revenue-basis")
		fixture.approveSupplement(order, created.ID, created.Version)
		if adjustments := fixture.adjustmentsForRequest(created.ID); len(adjustments) != 0 {
			t.Fatalf("收入口径不应生成冲减建议: %d", len(adjustments))
		}
		if count := fixture.decreaseTaskCount(created.ID); count != 0 {
			t.Fatalf("未产生建议的员工不得收到知情通知: %d", count)
		}
	})
}

// ---------------------------------------------------------------------------
// 场景 6：审批生成的 CONFIRMED 补录应付直接进入现有建账链路
// ---------------------------------------------------------------------------

func TestFeeSupplementBillChainPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)
	fixture.setT(t)
	order := fixture.createOrder("FSUP-BILL-" + fixture.suffix)
	fixture.lockOrder(order)
	created := fixture.createSupplement(order, "bill-chain")
	result := fixture.approveSupplement(order, created.ID, created.Version)
	feeID := result.Fee.ID

	// 建账候选识别补录来源：无需解锁订单即可把 CONFIRMED 补录应付建账。
	account, accountErr := fixture.data.db.PartnerAccount.Create().
		SetPartnerID(fixture.customerID).
		SetName("补录测试结算账户-" + fixture.suffix).
		SetAccountHolder("费用补录测试客户").
		SetBankName("集成测试银行").
		SetAccountNo("FSUP-BILL-" + fixture.suffix).
		SetCurrency("CNY").
		SetUsage(partneraccountent.UsageBOTH).
		SetEnabled(true).
		SetIsDefaultPayable(true).
		Save(fixture.ctx)
	if accountErr != nil {
		t.Fatalf("创建结算账户: %v", accountErr)
	}
	// 建账编号规则与结算账户：与现有账单集成测试同源夹具。
	if _, ruleErr := fixture.data.db.NumberRule.Create().
		SetOrganizationID(fixture.organizationID).
		SetDocumentType(numberruleent.DocumentTypeBill).
		SetPrefix("BILL-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(fixture.ctx); ruleErr != nil {
		t.Fatalf("创建账单编号规则: %v", ruleErr)
	}
	exchangeRate := biz.NewExchangeRateUsecase(NewExchangeRateRepo(fixture.data), NewExchangeRateQuoteProvider())
	billUsecase := biz.NewFinanceBillUsecase(NewFinanceBillRepo(fixture.data), exchangeRate, fixture.data)
	bill, createErr := billUsecase.Create(fixture.ctx, fixture.organizationID, fixture.approverID, biz.CreateFinanceBillInput{
		FeeIDs:              []uuid.UUID{feeID},
		BillDate:            "2026-09-18",
		IdempotencyKey:      "fsup-bill-" + created.ID.String(),
		SettlementAccountID: account.ID,
	})
	if createErr != nil {
		t.Fatalf("补录费用建账: %v", createErr)
	}
	billed, feeErr := fixture.data.db.OrderFee.Query().Where(orderfeeent.IDEQ(feeID)).Only(fixture.ctx)
	if feeErr != nil || billed.Status != orderfeeent.StatusBILLED {
		t.Fatalf("建账后补录费用必须转为 BILLED: %v %+v", feeErr, billed)
	}
	reloadedOrder, orderErr := fixture.data.db.Order.Query().Where(orderent.IDEQ(order.ID)).Only(fixture.ctx)
	if orderErr != nil || reloadedOrder.LockedAt == nil {
		t.Fatalf("建账不得解锁订单: %v", orderErr)
	}
	// 符合现有取消条件的草稿账单取消后，费用恢复 CONFIRMED。
	cancelled, cancelErr := billUsecase.Cancel(fixture.ctx, []uuid.UUID{fixture.organizationID}, fixture.approverID, bill.ID, bill.Version, "取消校验恢复链路")
	if cancelErr != nil {
		t.Fatalf("取消草稿账单: %v", cancelErr)
	}
	if cancelled.Status != biz.FinanceBillCancelled {
		t.Fatalf("账单取消状态不符: %s", cancelled.Status)
	}
	restored, feeErr := fixture.data.db.OrderFee.Query().Where(orderfeeent.IDEQ(feeID)).Only(fixture.ctx)
	if feeErr != nil || restored.Status != orderfeeent.StatusCONFIRMED {
		t.Fatalf("账单取消后补录费用必须恢复 CONFIRMED: %v %+v", feeErr, restored)
	}
	// 费用行保留补录来源关联。
	if restored.SupplementRequestID == nil || *restored.SupplementRequestID != created.ID {
		t.Fatalf("费用必须保留补录申请关联: %v", restored.SupplementRequestID)
	}
}

// ---------------------------------------------------------------------------
// 场景 7：列表逐行授权（fee.read / 发起人本人 / 实时 lock grant）
// ---------------------------------------------------------------------------

func TestFeeSupplementListAuthorizationPostgres(t *testing.T) {
	fixture := newFeeSupplementFixture(t)

	t.Run("无 fee.read 发起人只读本人申请且能力投影正确", func(t *testing.T) {
		order := fixture.createOrder("FSUP-LIST-" + fixture.suffix)
		fixture.lockOrder(order)
		pending := fixture.createSupplement(order, "list-pending")
		approved := fixture.createSupplement(order, "list-approved")
		fixture.approveSupplement(order, approved.ID, approved.Version)

		view, err := fixture.usecase.List(fixture.ctx, fixture.requesterPrincipal(), fixture.organizationID, order.ID, 1, 200)
		if err != nil {
			t.Fatalf("发起人读取列表: %v", err)
		}
		if view.Total != 2 || len(view.Items) != 2 {
			t.Fatalf("发起人应看到本人两条申请: total=%d", view.Total)
		}
		byID := make(map[uuid.UUID]*biz.OrderFeeSupplementRequestView, len(view.Items))
		for _, item := range view.Items {
			byID[item.Request.ID] = item
		}
		pendingView := byID[pending.ID]
		if pendingView == nil || !pendingView.CanWithdraw || pendingView.CanApprove || pendingView.CanCancel || pendingView.ApproverAvailable {
			t.Fatalf("本人 PENDING 投影不符: %+v", pendingView)
		}
		approvedView := byID[approved.ID]
		if approvedView == nil || approvedView.CanWithdraw || approvedView.CanApprove || approvedView.CanCancel {
			t.Fatalf("无 grant 发起人的 APPROVED 投影不符: %+v", approvedView)
		}
		if approvedView.FeeID == nil || approvedView.FeeStatus != "CONFIRMED" {
			t.Fatalf("APPROVED 行应投影生成费用状态: %+v", approvedView)
		}
	})

	t.Run("实时 grant 审批人无需 fee.read 可读订单申请并显示作废能力", func(t *testing.T) {
		order := fixture.createOrder("FSUP-LISTG-" + fixture.suffix)
		fixture.lockOrder(order)
		pending := fixture.createSupplement(order, "list-grant-pending")
		approved := fixture.createSupplement(order, "list-grant-approved")
		approvedResult := fixture.approveSupplement(order, approved.ID, approved.Version)

		view, err := fixture.usecase.List(fixture.ctx, fixture.approverPrincipal(), fixture.organizationID, order.ID, 1, 200)
		if err != nil {
			t.Fatalf("审批人读取列表: %v", err)
		}
		if view.Total != 2 {
			t.Fatalf("实时 grant 审批人应看到订单全部申请: total=%d", view.Total)
		}
		byID := make(map[uuid.UUID]*biz.OrderFeeSupplementRequestView, len(view.Items))
		for _, item := range view.Items {
			byID[item.Request.ID] = item
		}
		pendingView := byID[pending.ID]
		if pendingView == nil || !pendingView.CanApprove || pendingView.CanWithdraw || pendingView.ApproverAvailable != true {
			t.Fatalf("grant 审批人 PENDING 投影不符: %+v", pendingView)
		}
		approvedView := byID[approved.ID]
		if approvedView == nil || !approvedView.CanCancel || approvedView.CancelBlockReason != "" {
			t.Fatalf("最新有效 CONFIRMED 补录应可作废: %+v", approvedView)
		}
		if approvedView.FeeID == nil || *approvedView.FeeID != approvedResult.Fee.ID {
			t.Fatalf("APPROVED 行应投影生成费用 ID: %+v", approvedView)
		}
	})

	t.Run("无关用户列表为空且跨组织探测返回不存在", func(t *testing.T) {
		order := fixture.createOrder("FSUP-LISTN-" + fixture.suffix)
		fixture.lockOrder(order)
		fixture.createSupplement(order, "list-private")
		// 组织内无关用户：无 fee.read、无 grant、非发起人。
		outsider, outsiderErr := fixture.data.db.User.Create().SetDisplayName("补录无关用户-" + fixture.suffix).SetEnabled(true).SetIsBootstrapAdmin(false).Save(fixture.ctx)
		if outsiderErr != nil {
			t.Fatalf("创建无关用户: %v", outsiderErr)
		}
		if _, memberErr := fixture.data.db.Membership.Create().SetUserID(outsider.ID).SetOrganizationID(fixture.organizationID).SetPrimary(true).SetEnabled(true).Save(fixture.ctx); memberErr != nil {
			t.Fatalf("创建无关成员关系: %v", memberErr)
		}
		outsiderPrincipal := &biz.Principal{UserID: outsider.ID, Organization: biz.Organization{ID: fixture.organizationID}}
		view, err := fixture.usecase.List(fixture.ctx, outsiderPrincipal, fixture.organizationID, order.ID, 1, 200)
		if err != nil {
			t.Fatalf("无关用户列表不应报错: %v", err)
		}
		if view.Total != 0 || len(view.Items) != 0 {
			t.Fatalf("无关用户不得看到任何申请: total=%d", view.Total)
		}
		// 跨组织探测：稳定返回不存在，不泄露申请事实。
		otherOrg, orgErr := fixture.data.db.Organization.Create().SetCode("FSUP-OTHER-" + fixture.suffix).SetName("补录跨组织-" + fixture.suffix).SetKind("company").SetBaseCurrency("CNY").SetEnabled(true).Save(fixture.ctx)
		if orgErr != nil {
			t.Fatalf("创建跨组织: %v", orgErr)
		}
		otherPrincipal := &biz.Principal{UserID: outsider.ID, Organization: biz.Organization{ID: otherOrg.ID}}
		if _, listErr := fixture.usecase.List(fixture.ctx, otherPrincipal, otherOrg.ID, order.ID, 1, 200); listErr != biz.ErrFeeSupplementNotFound {
			t.Fatalf("跨组织探测应返回不存在，实际 %v", listErr)
		}
	})

	t.Run("持 fee.read 用户无需 grant 可读取全部申请", func(t *testing.T) {
		order := fixture.createOrder("FSUP-LISTF-" + fixture.suffix)
		fixture.lockOrder(order)
		fixture.createSupplement(order, "list-fee-read")
		feeReadUser, userErr := fixture.data.db.User.Create().SetDisplayName("补录费用查看-" + fixture.suffix).SetEnabled(true).SetIsBootstrapAdmin(false).Save(fixture.ctx)
		if userErr != nil {
			t.Fatalf("创建 fee.read 用户: %v", userErr)
		}
		feeReadPrincipal := &biz.Principal{
			UserID:       feeReadUser.ID,
			Organization: biz.Organization{ID: fixture.organizationID},
			RoleGrants: []biz.RoleGrant{{
				Permissions: map[string]struct{}{"business.order.si.fee.read": {}},
				DataScope:   biz.DataScopeOrganization,
			}},
		}
		view, err := fixture.usecase.List(fixture.ctx, feeReadPrincipal, fixture.organizationID, order.ID, 1, 200)
		if err != nil {
			t.Fatalf("fee.read 用户读取列表: %v", err)
		}
		if view.Total != 1 || len(view.Items) != 1 {
			t.Fatalf("fee.read 用户应读取订单全部申请: total=%d", view.Total)
		}
		// 仅 fee.read 不授予审批或撤回能力。
		if view.Items[0].CanApprove || view.Items[0].CanWithdraw || view.Items[0].CanCancel {
			t.Fatalf("仅 fee.read 不得获得操作能力: %+v", view.Items[0])
		}
	})
}
