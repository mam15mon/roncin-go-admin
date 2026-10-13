package data

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"

	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	commissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	notificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"

	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"

	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
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
		SetKey("system.finance.fee.review").
		SetName("海运进口订单锁定-补录测试-" + suffix).
		SetDescription("费用补录集成测试").
		SetGroup("order").
		Save(ctx)
	if permErr != nil {
		lockPerm, permErr = data.db.Permission.Query().Where(permissionent.KeyEQ("system.finance.fee.review")).First(ctx)
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
	return &biz.Principal{UserID: f.requesterID, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: f.organizationID, Kind: biz.OrganizationKindCompany}}}
}

// setT 把断言目标切换到当前子测试，避免子测试通过父测试 t 触发 FailNow。
func (f *feeSupplementPostgresFixture) setT(t *testing.T) {
	f.t = t
}

func (f *feeSupplementPostgresFixture) approverPrincipal() *biz.Principal {
	return &biz.Principal{UserID: f.approverID, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: f.organizationID, Kind: biz.OrganizationKindCompany}}}
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
	preview, err := f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, requestID, version)
	if err != nil {
		f.t.Fatal(err)
	}
	result, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, requestID, version, preview.PreviewToken, "补录成本核实无误")
	if err != nil {
		f.t.Fatalf("审批补录申请: %v", err)
	}
	return result
}

func (f *feeSupplementPostgresFixture) approveExpectError(order *ent.Order, requestID uuid.UUID, version uint64) error {
	preview, err := f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, requestID, version)
	if err != nil {
		return err
	}
	_, err = f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, order.ID, requestID, version, preview.PreviewToken, "成本核实")
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

// approveInitialFees 构造明确的初始批准事实；补录场景不再借业务锁授权。
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
	seedApprovedFeeSnapshotForTest(f.t, f.data, order.ID, f.approverID)
}

func TestFeeSupplementApprovalAdmissionPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("NEW-" + f.suffix)
	if _, err := f.usecase.Create(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, f.supplementInput("no-review"), false); err == nil {
		t.Fatal("未批准不可补录")
	}
	f.approveInitialFees(o)
	r := f.createSupplement(o, "new-review")
	if r.LockBasis != biz.SupplementLockBasisApproval || r.ApprovedFeeReviewEventID == nil {
		t.Fatal("必须保存批准依据")
	}
	retry, err := f.usecase.Create(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, f.supplementInput("new-review"), false)
	if err != nil || retry.ID != r.ID {
		t.Fatalf("重放: %v", err)
	}
	if f.approvalTaskCount(r.ID) != 1 {
		t.Fatal("提醒只能一次")
	}
	if _, err := f.usecase.Create(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, f.supplementInput("second-pending"), false); err == nil {
		t.Fatal("同票第二待审必须拒绝")
	}
	if _, err := f.usecase.PreviewApproval(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID, r.Version); err != biz.ErrPermissionDenied {
		t.Fatalf("录入不是批准: %v", err)
	}
	otherOrg, err := f.data.db.Organization.Create().SetCode("OTHER-" + f.suffix).SetName("其他").SetKind("company").SetBaseCurrency("CNY").Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), otherOrg.ID, o.ID, r.ID, r.Version)
	if err == nil {
		t.Fatal("跨组织拒绝")
	}
	preview, err := f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, "stale-token", ""); err == nil {
		t.Fatal("旧预览拒绝")
	}
	result, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.feesForRequest(r.ID)) != 1 || result.Request.Status != biz.OrderFeeSupplementApproved {
		t.Fatal("批准应创建一次")
	}
	current, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.FeeReviewStatus != orderent.FeeReviewStatusAPPROVED || *current.ApprovedFeeReviewEventID == *r.ApprovedFeeReviewEventID {
		t.Fatal("应产生新整票批准")
	}
}

func TestFeeSupplementLossAndSelfApprovalPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("LOSS-" + f.suffix)
	f.approveInitialFees(o)
	input := f.supplementInput("self-loss")
	input.UnitPrice = decimal.NewFromInt(700)
	r, err := f.usecase.Create(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, input, false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.ProjectedProfit.IsNegative() {
		t.Fatal("亏损需明确")
	}
	if _, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "  "); err == nil {
		t.Fatal("亏损意见必填")
	}
	if _, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "本票客户赔付，批准承担"); err != nil {
		t.Fatal(err)
	}
}

func TestFeeSupplementConfirmedImpactAndPaymentPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("PAY-" + f.suffix)
	f.approveInitialFees(o)
	parent := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	r := f.createSupplement(o, "payment-cost")
	if _, err := f.commissionRepo.PreviewPayment(f.ctx, f.organizationID, parent.ID, parent.Version); err != biz.ErrCommissionPendingSupplement {
		t.Fatalf("待审阻断付款: %v", err)
	}
	result := f.approveSupplement(o, r.ID, r.Version)
	adjustments := f.adjustmentsForRequest(r.ID)
	if len(adjustments) != 1 || adjustments[0].Status != commissionadjustmentent.StatusCONFIRMED || !decimal.RequireFromString(adjustments[0].Amount).Equal(decimal.NewFromInt(10)) {
		t.Fatalf("批准同时确认冲减: %+v", adjustments)
	}
	a := adjustments[0]
	if _, err := f.commissionRepo.TransitionAdjustment(f.ctx, f.organizationID, a.ID, f.approverID, a.Version, biz.CommissionCancelled, "忽略", nil); err == nil {
		t.Fatal("系统确认冲减不可忽略")
	}
	if _, err := f.commissionRepo.TransitionAdjustment(f.ctx, f.organizationID, a.ID, f.approverID, a.Version, biz.CommissionPaid, "", nil); err == nil {
		t.Fatal("父未付不能独立执行")
	}
	if _, err := f.usecase.CancelApprovedFee(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, result.Fee.Version, "误录"); err == nil {
		t.Fatal("有确认冲减不可直接撤销")
	}
	parent, err := f.data.db.FinanceCommission.Get(f.ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := f.commissionRepo.PreviewPayment(f.ctx, f.organizationID, parent.ID, parent.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.ActualAmount.Equal(decimal.NewFromInt(50)) {
		t.Fatal(preview.ActualAmount)
	}
	in := biz.CommissionPaymentInput{PreviewToken: preview.PreviewToken, ActualAmount: preview.ActualAmount, IdempotencyKey: "pay-one"}
	paid, err := f.commissionRepo.Pay(f.ctx, f.organizationID, parent.ID, f.approverID, parent.Version, in, f.commissionAudit(parent.ID, "finance.commission.paid"))
	if err != nil {
		t.Fatal(err)
	}
	if paid.PaidAmount == nil || !paid.PaidAmount.Equal(decimal.NewFromInt(50)) {
		t.Fatal("实付快照")
	}
	if _, err := f.commissionRepo.Pay(f.ctx, f.organizationID, parent.ID, f.approverID, parent.Version, in, f.commissionAudit(parent.ID, "finance.commission.paid")); err != nil {
		t.Fatalf("付款重放: %v", err)
	}
	done, err := f.data.db.FinanceCommissionAdjustment.Get(f.ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.ExecutionMethod == nil || *done.ExecutionMethod != commissionadjustmentent.ExecutionMethodOFFSET_WITH_PARENT || done.ParentPaymentID == nil {
		t.Fatal("抵扣身份")
	}
	if _, err := f.commissionRepo.TransitionAdjustment(f.ctx, f.organizationID, a.ID, f.approverID, done.Version, biz.CommissionPaid, "", nil); err == nil {
		t.Fatal("不得重复扣回")
	}
	r2 := f.createSupplement(o, "after-payment")
	f.approveSupplement(o, r2.ID, r2.Version)
	a2 := f.adjustmentsForRequest(r2.ID)[0]
	if _, err := f.commissionRepo.TransitionAdjustment(f.ctx, f.organizationID, a2.ID, f.approverID, a2.Version, biz.CommissionPaid, "", f.commissionAudit(a2.ID, "finance.commission_adjustment.paid")); err != nil {
		t.Fatal(err)
	}
	frozen, err := f.data.db.FinanceCommission.Get(f.ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *frozen.PaidAmount != "50.00000000" {
		t.Fatal("历史实付不能变化")
	}
}

func TestFeeSupplementConsecutiveHistoricalSnapshotPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("COST-" + f.suffix)
	f.approveInitialFees(o)
	original := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	a := f.createSupplement(o, "cost-a")
	first := f.approveSupplement(o, a.ID, a.Version)
	later := f.createConfirmedCommissionWithSnapshot(o, "50", "10", "1000", "500")
	_ = first
	_ = later
	b := f.createSupplement(o, "cost-b")
	f.approveSupplement(o, b.ID, b.Version)
	rows := f.adjustmentsForRequest(b.ID)
	if len(rows) != 2 {
		t.Fatal("逐原单独立冲减")
	}
	for _, row := range rows {
		expected := decimal.NewFromInt(10)
		if row.CommissionID == original.ID {
			expected = decimal.NewFromInt(20)
		}
		if !decimal.RequireFromString(row.Amount).Equal(expected) {
			f.t.Fatalf("应按累计净额重建原单20、新单10: %s", row.Amount)
		}
	}
	old := f.adjustmentsForRequest(a.ID)
	if len(old) != 1 || old[0].Status != commissionadjustmentent.StatusCANCELLED {
		t.Fatalf("先前未执行系统冲减应留取消历史: %+v", old)
	}

}

func TestFeeSupplementWithdrawApproveRacePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("RACE-" + f.suffix)
	f.approveInitialFees(o)
	r := f.createSupplement(o, "race")
	preview, err := f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan error, 2)
	go func() {
		defer wg.Done()
		_, err := f.usecase.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "")
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := f.usecase.Withdraw(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
		results <- err
	}()
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("并发只能一方成功: %d", success)
	}
}

func TestFeeSupplementReadAndRollbackPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("ROLLBACK-" + f.suffix)
	f.approveInitialFees(o)
	r := f.createSupplement(o, "rollback")
	f.insertCommissionWithLineContext(o, "FUTURE_V9", biz.CommissionBasisRealizedProfit, "CNY", true)
	if err := f.approveExpectError(o, r.ID, r.Version); err == nil {
		t.Fatal("未知算法阻断")
	}
	if len(f.feesForRequest(r.ID)) != 0 || f.requestByID(r.ID).Status != orderfeesupplementent.StatusPENDING {
		t.Fatal("失败不得部分写入")
	}
	list, err := f.usecase.List(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, 1, 20)
	if err != nil || len(list.Items) != 1 || !list.Items[0].CanWithdraw || list.Items[0].CanApprove {
		t.Fatalf("本人列表能力: %v", err)
	}
	unrelated := *f.requesterPrincipal()
	unrelated.UserID = uuid.New()
	hidden, err := f.usecase.List(f.ctx, &unrelated, f.organizationID, o.ID, 1, 20)
	if err != nil || len(hidden.Items) != 0 {
		t.Fatalf("无关用户不可见: %v", err)
	}
}

func TestFeeSupplementRevenueAndCancellationPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("REVENUE-" + f.suffix)
	f.approveInitialFees(o)
	f.insertCommissionWithLineContext(o, biz.CommissionCalculationVersion, biz.CommissionBasisRealizedRevenue, "CNY", true)
	r := f.createSupplement(o, "revenue-cost")
	result := f.approveSupplement(o, r.ID, r.Version)
	if len(f.adjustmentsForRequest(r.ID)) != 0 {
		t.Fatal("收入计提不受成本影响")
	}
	pending := f.createSupplement(o, "cancel-pending")
	if _, err := f.usecase.CancelApprovedFee(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, result.Fee.Version, "待审并发撤销"); err == nil {
		t.Fatal("存在待审补录须阻断撤销批准")
	}
	if _, err := f.usecase.Withdraw(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, pending.ID, pending.Version); err != nil {
		t.Fatal(err)
	}
	_, err := f.usecase.CancelApprovedFee(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, result.Fee.Version, "重复成本")
	if err != nil {
		t.Fatal(err)
	}
	current, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.FeeReviewStatus != orderent.FeeReviewStatusREVOKED || current.ApprovedFeeReviewEventID != nil {
		t.Fatal("删除成本须整票重审")
	}
}

func (f *feeSupplementPostgresFixture) commissionAudit(id uuid.UUID, action string) *biz.AuditEvent {
	return &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.approverID, Action: action, Result: "success", ResourceType: "finance_commission", ResourceID: id.String()}
}
