package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// submitSingleLineApplication 为指定员工构造单来源（一笔 60 元销售提成）并提交
// 月度申请，返回员工 ID 与申请头。
func (f *commissionApplicationFixture) submitSingleLineApplication(t *testing.T, label string) (uuid.UUID, *biz.FinanceCommissionApplication) {
	t.Helper()
	employee := f.newEmployee(label)
	f.addAttribution(employee, "SALES")
	f.addSalesScheme(employee)
	f.addVerification(f.historicalDate)
	application, err := f.usecase().Submit(context.Background(), f.scopeFor(employee))
	if err != nil {
		t.Fatalf("员工 %s 提交月度申请失败: %v", label, err)
	}
	return employee, application
}

// TestCommissionApplicationFinanceReadPostgres 覆盖财务读取：列表过滤、服务端
// 分页、展示名解析、明细单号下钻与跨组织隔离。
func TestCommissionApplicationFinanceReadPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()

	// 先补齐两名员工与两张核销单再提交：两张核销都分摊同一账单、命中同一订单
	// 归属，因此每名员工的可申请候选都是 2 笔（每人每来源各 60 元）。
	employeeA := fixture.newEmployee("fina")
	fixture.addAttribution(employeeA, "SALES")
	fixture.addSalesScheme(employeeA)
	fixture.addVerification(fixture.historicalDate)
	employeeB := fixture.newEmployee("finb")
	fixture.addAttribution(employeeB, "SALES")
	fixture.addSalesScheme(employeeB)
	fixture.addVerification(fixture.historicalDate)
	applicationA, err := usecase.Submit(ctx, fixture.scopeFor(employeeA))
	if err != nil {
		t.Fatalf("员工 A 提交月度申请失败: %v", err)
	}
	applicationB, err := usecase.Submit(ctx, fixture.scopeFor(employeeB))
	if err != nil {
		t.Fatalf("员工 B 提交月度申请失败: %v", err)
	}
	orgScope := []uuid.UUID{fixture.organizationID}

	list, err := usecase.ListForOrganization(ctx, orgScope, biz.CommissionApplicationFinanceFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("财务申请列表失败: %v", err)
	}
	if list.Total != 2 || list.Page != 1 || list.PageSize != 20 {
		t.Fatalf("财务申请列表总数不符: %+v", list)
	}
	for _, item := range list.Items {
		if item.Status != biz.CommissionApplicationPendingReview {
			t.Fatalf("新提交申请应为待审: %+v", item)
		}
		if item.EmployeeName == "" || item.OrganizationName == "" {
			t.Fatalf("财务列表应解析员工与组织展示名: %+v", item)
		}
		if item.CommissionCount != 2 || item.TotalCommissionAmount.StringFixed(8) != "120.00000000" {
			t.Fatalf("申请头汇总快照不符: count=%d total=%s", item.CommissionCount, item.TotalCommissionAmount)
		}
	}

	// 按员工过滤只返回该员工的申请批次。
	byEmployee, err := usecase.ListForOrganization(ctx, orgScope, biz.CommissionApplicationFinanceFilter{Page: 1, PageSize: 20, EmployeeID: employeeA})
	if err != nil {
		t.Fatalf("按员工过滤失败: %v", err)
	}
	if byEmployee.Total != 1 || byEmployee.Items[0].ID != applicationA.ID {
		t.Fatalf("按员工过滤应只命中员工 A 的申请: %+v", byEmployee)
	}

	// 按状态与提交月过滤。
	byStatus, err := usecase.ListForOrganization(ctx, orgScope, biz.CommissionApplicationFinanceFilter{Page: 1, PageSize: 20, Status: biz.CommissionApplicationApproved})
	if err != nil {
		t.Fatalf("按状态过滤失败: %v", err)
	}
	if byStatus.Total != 0 {
		t.Fatalf("尚无批准申请: %d", byStatus.Total)
	}
	byMonth, err := usecase.ListForOrganization(ctx, orgScope, biz.CommissionApplicationFinanceFilter{Page: 1, PageSize: 20, ApplicationMonth: fixture.currentDate[:7]})
	if err != nil {
		t.Fatalf("按提交月过滤失败: %v", err)
	}
	if byMonth.Total != 2 {
		t.Fatalf("按提交月过滤应命中全部申请: %d", byMonth.Total)
	}
	if _, err := usecase.ListForOrganization(ctx, orgScope, biz.CommissionApplicationFinanceFilter{Page: 1, PageSize: 20, ApplicationMonth: "2000-01"}); err != nil {
		t.Fatalf("无命中提交月查询不应报错: %v", err)
	}

	// 跨组织列表隔离：其他组织范围看不到本组织申请。
	otherList, err := usecase.ListForOrganization(ctx, []uuid.UUID{uuid.Must(uuid.NewV7())}, biz.CommissionApplicationFinanceFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("跨组织列表查询失败: %v", err)
	}
	if otherList.Total != 0 {
		t.Fatalf("跨组织列表应隔离: %d", otherList.Total)
	}

	// 详情下钻：明细单号与展示名服务端解析。
	detail, err := usecase.GetForOrganization(ctx, orgScope, applicationA.ID)
	if err != nil {
		t.Fatalf("财务申请详情失败: %v", err)
	}
	if detail.Application.EmployeeName == "" || detail.Application.OrganizationName == "" {
		t.Fatalf("财务详情应解析展示名: %+v", detail.Application)
	}
	if len(detail.Lines) != 2 {
		t.Fatalf("申请明细应为 2 条: %d", len(detail.Lines))
	}
	for _, line := range detail.Lines {
		if !strings.HasPrefix(line.CommissionNo, "FCA-TC-") {
			t.Fatalf("明细应携带提成单号: %+v", line)
		}
		if line.CommissionDate != fixture.historicalDate || line.SourceFingerprint == "" {
			t.Fatalf("明细快照不符: %+v", line)
		}
		if line.CommissionAmount.StringFixed(8) != "60.00000000" {
			t.Fatalf("明细金额不符: %s", line.CommissionAmount)
		}
	}

	// 跨组织详情按不存在处理，不泄露记录事实。
	if _, err := usecase.GetForOrganization(ctx, []uuid.UUID{uuid.Must(uuid.NewV7())}, applicationA.ID); !errors.Is(err, biz.ErrCommissionApplicationNotFound) {
		t.Fatalf("跨组织详情应按不存在处理: %v", err)
	}
	if _, err := usecase.GetForOrganization(ctx, orgScope, applicationB.ID); err != nil {
		t.Fatalf("第二张申请详情失败: %v", err)
	}
}

// TestCommissionApplicationApprovePostgres 覆盖整单批准主链路：版本门禁、明细
// 提成整批转 CONFIRMED、财务锁净额联动、决策审计与批准后重复提交拒绝。
func TestCommissionApplicationApprovePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()
	orgScope := []uuid.UUID{fixture.organizationID}

	employee, application := fixture.submitSingleLineApplication(t, "approve")

	// 过期版本稳定拒绝，不产生任何状态变化。
	if _, err := usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 99); !errors.Is(err, biz.ErrCommissionApplicationStatusConflict) {
		t.Fatalf("过期版本的批准应稳定拒绝: %v", err)
	}
	header, err := fixture.data.db.FinanceCommissionApplication.Query().Where(applicationent.IDEQ(application.ID)).Only(ctx)
	if err != nil {
		t.Fatalf("查询申请头失败: %v", err)
	}
	if header.Status != applicationent.StatusPENDING_REVIEW || header.Version != 1 || header.DecidedAt != nil {
		t.Fatalf("被拒绝的批准不得改变申请头: %+v", header)
	}

	approved, err := usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 1)
	if err != nil {
		t.Fatalf("整单批准失败: %v", err)
	}
	if approved.Status != biz.CommissionApplicationApproved || approved.Version != 2 {
		t.Fatalf("批准后申请头状态/版本不符: %+v", approved)
	}
	if approved.DecidedAt == nil || approved.DecidedBy == nil || *approved.DecidedBy != fixture.actorID {
		t.Fatalf("批准应记录决策人与时间: %+v", approved)
	}

	// 明细提成整批转 CONFIRMED：决策人是财务操作者，版本递增。
	commissionRow, err := fixture.data.db.FinanceCommission.Query().Where(commission.EmployeeIDEQ(employee)).Only(ctx)
	if err != nil {
		t.Fatalf("查询明细提成失败: %v", err)
	}
	if commissionRow.Status != commission.StatusCONFIRMED || commissionRow.Version != 2 {
		t.Fatalf("批准后提成应整批确认: %+v", commissionRow)
	}
	if commissionRow.ConfirmedBy == nil || *commissionRow.ConfirmedBy != fixture.actorID || commissionRow.ConfirmedAt == nil {
		t.Fatalf("提成确认审计应指向财务操作者: %+v", commissionRow)
	}

	// 财务锁净额口径：确认后净额 60 > 0，订单费用写入被财务锁谓词拦截。
	lockLines, err := fixture.data.db.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(commissionRow.ID)).All(ctx)
	if err != nil {
		t.Fatalf("查询提成行失败: %v", err)
	}
	netAmount, err := financeCommissionLockNetAmount(lockLines, nil)
	if err != nil {
		t.Fatalf("计算财务锁净额失败: %v", err)
	}
	if netAmount.StringFixed(8) != "60.00000000" {
		t.Fatalf("批准后订单财务锁净额应为 60: %s", netAmount)
	}
	lockedCount, err := fixture.data.db.Order.Query().Where(orderent.IDEQ(fixture.orderID), financeLockedOrderPredicate()).Count(ctx)
	if err != nil {
		t.Fatalf("查询财务锁谓词失败: %v", err)
	}
	if lockedCount != 1 {
		t.Fatalf("批准后订单应被财务锁谓词命中: %d", lockedCount)
	}

	// 决策审计与逐笔确认审计各恰好一条。
	approveAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ResourceIDEQ(application.ID.String()), auditlog.ActionEQ("finance.commission_application.approve")).Count(ctx)
	if err != nil {
		t.Fatalf("查询批准审计失败: %v", err)
	}
	if approveAudits != 1 {
		t.Fatalf("批准审计应恰好一条: %d", approveAudits)
	}
	confirmAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ResourceIDEQ(commissionRow.ID.String()), auditlog.ActionEQ("finance.commission.confirm")).Count(ctx)
	if err != nil {
		t.Fatalf("查询确认审计失败: %v", err)
	}
	if confirmAudits != 1 {
		t.Fatalf("整批确认应写逐笔确认审计: %d", confirmAudits)
	}

	// 重复批准与批准后的员工重复提交均稳定拒绝。
	if _, err := usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 2); !errors.Is(err, biz.ErrCommissionApplicationStatusConflict) {
		t.Fatalf("重复批准应稳定拒绝: %v", err)
	}
	if _, err := usecase.Submit(ctx, fixture.scopeFor(employee)); !errors.Is(err, biz.ErrCommissionApplicationConflict) {
		t.Fatalf("批准后的同月重复提交应稳定拒绝: %v", err)
	}
}

// TestCommissionApplicationRejectResubmitApprovePostgres 验证整单驳回与员工原单
// 重提的串联：驳回只改申请头并保存原因，提成保持 DRAFT；重提后可再次审批。
func TestCommissionApplicationRejectResubmitApprovePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()
	orgScope := []uuid.UUID{fixture.organizationID}

	employee, application := fixture.submitSingleLineApplication(t, "reject")

	// 空原因在领域入口稳定拒绝。
	if _, err := usecase.Reject(ctx, orgScope, fixture.actorID, application.ID, 1, "   "); !errors.Is(err, biz.ErrCommissionApplicationInvalid) {
		t.Fatalf("空原因的驳回应稳定拒绝: %v", err)
	}

	rejected, err := usecase.Reject(ctx, orgScope, fixture.actorID, application.ID, 1, "明细存疑，整单驳回")
	if err != nil {
		t.Fatalf("整单驳回失败: %v", err)
	}
	if rejected.Status != biz.CommissionApplicationRejected || rejected.Version != 2 {
		t.Fatalf("驳回后申请头状态/版本不符: %+v", rejected)
	}
	if rejected.DecisionReason == nil || *rejected.DecisionReason != "明细存疑，整单驳回" {
		t.Fatalf("驳回应保存原因: %+v", rejected)
	}
	rejectAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ResourceIDEQ(application.ID.String()), auditlog.ActionEQ("finance.commission_application.reject")).Count(ctx)
	if err != nil {
		t.Fatalf("查询驳回审计失败: %v", err)
	}
	if rejectAudits != 1 {
		t.Fatalf("驳回审计应恰好一条: %d", rejectAudits)
	}

	// 明细提成保持 DRAFT，员工可在原申请上重提（与阶段 B 的重提路径串联）。
	commissionRow, err := fixture.data.db.FinanceCommission.Query().Where(commission.EmployeeIDEQ(employee)).Only(ctx)
	if err != nil {
		t.Fatalf("查询明细提成失败: %v", err)
	}
	if commissionRow.Status != commission.StatusDRAFT {
		t.Fatalf("驳回不得改变提成状态: %+v", commissionRow)
	}
	lines, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(application.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("统计申请明细失败: %v", err)
	}
	if lines != 1 {
		t.Fatalf("驳回应保留申请明细: %d", lines)
	}
	resubmitted, err := usecase.Resubmit(ctx, fixture.scopeFor(employee), application.ID, 2)
	if err != nil {
		t.Fatalf("驳回后原单重提失败: %v", err)
	}
	if resubmitted.ID != application.ID || resubmitted.Status != biz.CommissionApplicationPendingReview || resubmitted.Version != 3 {
		t.Fatalf("重提应沿用原申请并回到待审: %+v", resubmitted)
	}

	// 重提后再次整单批准。
	approved, err := usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 3)
	if err != nil {
		t.Fatalf("重提后整单批准失败: %v", err)
	}
	if approved.Status != biz.CommissionApplicationApproved || approved.Version != 4 {
		t.Fatalf("重提后批准状态/版本不符: %+v", approved)
	}
	confirmedCommission, err := fixture.data.db.FinanceCommission.Query().Where(commission.EmployeeIDEQ(employee)).Only(ctx)
	if err != nil {
		t.Fatalf("重查明细提成失败: %v", err)
	}
	if confirmedCommission.Status != commission.StatusCONFIRMED {
		t.Fatalf("重提后批准应确认提成: %+v", confirmedCommission)
	}
}

// TestCommissionApplicationApproveRollbackPostgres 验证批准阻断事实整体回滚：
// 来源指纹变化与草稿费用任一命中即整单失败，无任何提成被确认、申请头保持待审。
func TestCommissionApplicationApproveRollbackPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	ctx := context.Background()

	// 场景一：批准时来源指纹已变化（确认费用金额事后被修改）→ 整单失败。
	driftFixture := newCommissionApplicationFixture(t)
	driftUsecase := driftFixture.usecase()
	driftScope := []uuid.UUID{driftFixture.organizationID}
	_, driftApplication := driftFixture.submitSingleLineApplication(t, "drift")
	if _, err := driftFixture.data.db.OrderFee.UpdateOneID(driftFixture.receivableFeeID).
		SetBaseCurrencyAmount("1100.00000000").
		SetTotalAmount("1100.00000000").
		SetNetAmount("1100.00000000").
		SetUnitPrice("1100.0000").
		SetVersion(2).
		Save(ctx); err != nil {
		t.Fatalf("修改来源费用制造指纹漂移失败: %v", err)
	}
	if _, err := driftUsecase.Approve(ctx, driftScope, driftFixture.actorID, driftApplication.ID, 1); !errors.Is(err, biz.ErrCommissionSourceChanged) {
		t.Fatalf("指纹变化的批准应返回来源已变化: %v", err)
	}
	assertApplicationUntouched(t, driftFixture, driftApplication.ID, "指纹漂移")

	// 场景二：批准时来源订单仍有草稿费用 → 整单失败（指纹未变化）。
	draftFixture := newCommissionApplicationFixture(t)
	draftUsecase := draftFixture.usecase()
	draftScope := []uuid.UUID{draftFixture.organizationID}
	_, draftApplication := draftFixture.submitSingleLineApplication(t, "draftfee")
	if _, err := draftFixture.data.db.OrderFee.Create().
		SetOrderID(draftFixture.orderID).
		SetIdempotencyKey("fca-draft-block-" + draftFixture.suffix).
		SetDirection(fee.DirectionRECEIVABLE).
		SetStatus(fee.StatusDRAFT).
		SetFeeCode("DOCUMENT").
		SetFeeName("未确认杂费").
		SetSettlementPartyID(draftFixture.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("50.0000").
		SetTotalAmount("50.00000000").
		SetNetAmount("50.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(draftFixture.historicalDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("50.00000000").
		SetExpenseDate(draftFixture.historicalDate).
		SetVersion(1).
		Save(ctx); err != nil {
		t.Fatalf("创建草稿阻断费用失败: %v", err)
	}
	if _, err := draftUsecase.Approve(ctx, draftScope, draftFixture.actorID, draftApplication.ID, 1); !errors.Is(err, biz.ErrCommissionUnconfirmedFees) {
		t.Fatalf("草稿费用阻断的批准应返回未确认费用: %v", err)
	}
	assertApplicationUntouched(t, draftFixture, draftApplication.ID, "草稿费用")
}

// assertApplicationUntouched 断言批准失败后整体回滚：提成仍为 DRAFT、申请头
// 保持待审版本 1、未写任何决策或确认审计。
func assertApplicationUntouched(t *testing.T, fixture *commissionApplicationFixture, applicationID uuid.UUID, label string) {
	t.Helper()
	ctx := context.Background()
	header, err := fixture.data.db.FinanceCommissionApplication.Query().Where(applicationent.IDEQ(applicationID)).Only(ctx)
	if err != nil {
		t.Fatalf("查询申请头失败: %v", err)
	}
	if header.Status != applicationent.StatusPENDING_REVIEW || header.Version != 1 || header.DecidedAt != nil {
		t.Fatalf("%s 场景申请头应保持待审: %+v", label, header)
	}
	lines, err := fixture.data.db.FinanceCommissionApplicationLine.Query().
		Where(applicationline.ApplicationIDEQ(applicationID)).All(ctx)
	if err != nil {
		t.Fatalf("查询申请明细失败: %v", err)
	}
	for _, line := range lines {
		commissionRow, err := fixture.data.db.FinanceCommission.Query().Where(commission.IDEQ(line.CommissionID)).Only(ctx)
		if err != nil {
			t.Fatalf("查询明细提成失败: %v", err)
		}
		if commissionRow.Status != commission.StatusDRAFT || commissionRow.Version != 1 {
			t.Fatalf("%s 场景提成应保持 DRAFT: %+v", label, commissionRow)
		}
	}
	decisionAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ResourceIDEQ(applicationID.String()), auditlog.ActionIn("finance.commission_application.approve", "finance.commission_application.reject")).Count(ctx)
	if err != nil {
		t.Fatalf("查询决策审计失败: %v", err)
	}
	if decisionAudits != 0 {
		t.Fatalf("%s 场景不得写决策审计: %d", label, decisionAudits)
	}
	confirmAudits, err := fixture.data.db.AuditLog.Query().
		Where(auditlog.ActionEQ("finance.commission.confirm"), auditlog.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
	if err != nil {
		t.Fatalf("查询确认审计失败: %v", err)
	}
	if confirmAudits != 0 {
		t.Fatalf("%s 场景不得写确认审计: %d", label, confirmAudits)
	}
}

// TestCommissionApplicationConcurrentApproveRejectPostgres 验证并发批准/驳回：
// 申请头 FOR UPDATE + expected_version 下恰好一个事务成功，其余稳定冲突，且
// 终态与唯一成功方一致（批准则提成全部确认，驳回则提成保持 DRAFT）。
func TestCommissionApplicationConcurrentApproveRejectPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	fixture := newCommissionApplicationFixture(t)
	ctx := context.Background()
	usecase := fixture.usecase()
	orgScope := []uuid.UUID{fixture.organizationID}

	employee, application := fixture.submitSingleLineApplication(t, "race")

	type decisionResult struct {
		action string
		err    error
	}
	const approverCount = 3
	const rejecterCount = 3
	results := make([]decisionResult, approverCount+rejecterCount)
	var start sync.WaitGroup
	var done sync.WaitGroup
	start.Add(1)
	for i := 0; i < approverCount+rejecterCount; i++ {
		done.Add(1)
		go func(index int) {
			defer done.Done()
			start.Wait()
			if index < approverCount {
				_, err := usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 1)
				results[index] = decisionResult{action: "approve", err: err}
				return
			}
			_, err := usecase.Reject(ctx, orgScope, fixture.actorID, application.ID, 1, "并发驳回")
			results[index] = decisionResult{action: "reject", err: err}
		}(i)
	}
	start.Add(-1)
	done.Wait()

	successCount := 0
	winnerAction := ""
	for _, result := range results {
		switch {
		case result.err == nil:
			successCount++
			winnerAction = result.action
		case errors.Is(result.err, biz.ErrCommissionApplicationStatusConflict):
		default:
			t.Fatalf("并发决策出现意外错误: %v", result.err)
		}
	}
	if successCount != 1 {
		t.Fatalf("并发批准/驳回应恰好一成功: success=%d", successCount)
	}

	header, err := fixture.data.db.FinanceCommissionApplication.Query().Where(applicationent.IDEQ(application.ID)).Only(ctx)
	if err != nil {
		t.Fatalf("查询申请头失败: %v", err)
	}
	commissionRow, err := fixture.data.db.FinanceCommission.Query().Where(commission.EmployeeIDEQ(employee)).Only(ctx)
	if err != nil {
		t.Fatalf("查询明细提成失败: %v", err)
	}
	if winnerAction == "approve" {
		if header.Status != applicationent.StatusAPPROVED || header.Version != 2 {
			t.Fatalf("批准胜出后申请头应为 APPROVED v2: %+v", header)
		}
		if commissionRow.Status != commission.StatusCONFIRMED {
			t.Fatalf("批准胜出后提成应确认: %+v", commissionRow)
		}
	} else {
		if header.Status != applicationent.StatusREJECTED || header.Version != 2 {
			t.Fatalf("驳回胜出后申请头应为 REJECTED v2: %+v", header)
		}
		if commissionRow.Status != commission.StatusDRAFT {
			t.Fatalf("驳回胜出后提成应保持 DRAFT: %+v", commissionRow)
		}
	}

	// 唯一成功方之外的状态：败者的重复动作基于过期版本再次稳定拒绝。
	var staleErr error
	if winnerAction == "approve" {
		_, staleErr = usecase.Reject(ctx, orgScope, fixture.actorID, application.ID, 1, "过期版本")
	} else {
		_, staleErr = usecase.Approve(ctx, orgScope, fixture.actorID, application.ID, 1)
	}
	if !errors.Is(staleErr, biz.ErrCommissionApplicationStatusConflict) {
		t.Fatalf("过期版本的后续决策应稳定拒绝: %v", staleErr)
	}
}
