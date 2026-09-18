package data

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	"github.com/shopspring/decimal"
)

// newAdjustmentTransitionUsecase 构造真实提成用例与仓储，供调整状态迁移与
// 本人来源详情测试直接调用通用入口。
func newAdjustmentTransitionUsecase(f *feeSupplementPostgresFixture) *biz.CommissionUsecase {
	return biz.NewCommissionUsecase(
		NewCommissionRepo(f.data),
		biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)),
		f.data,
	)
}

// insertAdjustmentRow 按指定方向/状态插入 MANUAL 来源调整行（LFS 来源的调整
// 一律通过真实补录审批链路生成），金额为 8 位字符串。
func insertAdjustmentRow(f *feeSupplementPostgresFixture, order *ent.Order, parent *ent.FinanceCommission, direction commissionadjustmentent.Direction, status commissionadjustmentent.Status, amount, key, reason string) *ent.FinanceCommissionAdjustment {
	f.t.Helper()
	adjustment, err := f.data.db.FinanceCommissionAdjustment.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionID(parent.ID).
		SetOrderID(order.ID).
		SetAdjustmentNo(parent.CommissionNo + "-ADJ7" + strings.ReplaceAll(uuid.NewString(), "-", "")[:6]).
		SetIdempotencyKey(key).
		SetCommissionNo(parent.CommissionNo).
		SetOrderNo(order.OrderNo).
		SetEmployeeID(parent.EmployeeID).
		SetEmployeeName(parent.EmployeeName).
		SetSourceType(commissionadjustmentent.SourceTypeMANUAL).
		SetDirection(direction).
		SetStatus(status).
		SetBaseCurrency("CNY").
		SetAmount(amount).
		SetReason(reason).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入调整行 %s: %v", key, err)
	}
	return adjustment
}

// insertTwoLineCommission 插入一张 CONFIRMED 提成父单与两条订单行（A/B 各一），
// 供订单行/父单双层余额校验测试构造「其他订单行正余额不兜底」场景。
func insertTwoLineCommission(f *feeSupplementPostgresFixture, orderA, orderB *ent.Order, amountA, amountB string) *ent.FinanceCommission {
	f.t.Helper()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	total := decimal.RequireFromString(amountA).Add(decimal.RequireFromString(amountB)).StringFixed(8)
	rule, ruleErr := f.data.db.FinanceCommissionRule.Create().
		SetOrganizationID(f.organizationID).
		SetName("余额测试规则-" + suffix).
		SetPersonnelRole(commissionruleent.PersonnelRoleSALES).
		SetCalculationBasis(commissionruleent.CalculationBasisREALIZED_PROFIT).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(f.ctx)
	if ruleErr != nil {
		f.t.Fatalf("插入余额测试规则: %v", ruleErr)
	}
	parent, err := f.data.db.FinanceCommission.Create().
		SetOrganizationID(f.organizationID).
		SetCommissionNo("FC-BAL-" + suffix).
		SetIdempotencyKey("bal-commission-" + suffix).
		SetEmployeeID(f.employeeID).
		SetEmployeeName("余额测试员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(2).
		SetFeeCount(2).
		SetRuleID(rule.ID).
		SetRuleName("余额测试规则-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis("REALIZED_PROFIT").
		SetCalculationVersion(biz.CommissionCalculationVersion).
		SetSourceFingerprint(strings.Repeat("b", 64)).
		SetStatus(commissionent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetRealizedRevenue("2000.00000000").
		SetAllocatedCost("800.00000000").
		SetRealizedProfit("1200.00000000").
		SetCommissionBaseAmount("1200.00000000").
		SetRatePercent("10.0000").
		SetCommissionAmount(total).
		SetCommissionDate("2026-09-01").
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate("2026-09-01").
		SetCnyCommissionAmount(total).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("插入双行提成父单: %v", err)
	}
	for _, item := range []struct {
		order  *ent.Order
		amount string
	}{{orderA, amountA}, {orderB, amountB}} {
		if _, lineErr := f.data.db.FinanceCommissionLine.Create().
			SetOrganizationID(f.organizationID).
			SetCommissionID(parent.ID).
			SetOrderID(item.order.ID).
			SetOrderNo(item.order.OrderNo).
			SetOrderDate("2026-08-01").
			SetCustomerID(f.customerID).
			SetCustomerCode("CUST").
			SetCustomerName("余额测试客户").
			SetPersonnelAssignmentID(uuid.New()).
			SetPersonnelOrganizationID(f.organizationID).
			SetPersonnelAssignedAt(time.Now().UTC()).
			SetFeeCount(1).
			SetFeeSnapshot("[]").
			SetEmployeeID(f.employeeID).
			SetEmployeeName("余额测试员工-" + suffix).
			SetPersonnelRole("SALES").
			SetCalculationBasis("REALIZED_PROFIT").
			SetBaseCurrency("CNY").
			SetRealizedRevenue("1000.00000000").
			SetAllocatedCost("400.00000000").
			SetRealizedProfit("600.00000000").
			SetCommissionBaseAmount("600.00000000").
			SetRatePercent("10.0000").
			SetCommissionAmount(item.amount).
			Save(f.ctx); lineErr != nil {
			f.t.Fatalf("插入双行提成订单行: %v", lineErr)
		}
	}
	return parent
}

// requireAdjustmentState 读取调整行并断言状态、版本与取消字段符合预期。
func requireAdjustmentState(f *feeSupplementPostgresFixture, id uuid.UUID, wantStatus string, wantVersion uint64, wantCancelled bool) {
	f.t.Helper()
	item, err := f.data.db.FinanceCommissionAdjustment.Query().Where(commissionadjustmentent.IDEQ(id)).Only(f.ctx)
	if err != nil {
		f.t.Fatalf("读取调整行: %v", err)
	}
	if string(item.Status) != wantStatus || item.Version != wantVersion {
		f.t.Fatalf("调整状态或版本不符: status=%s version=%d want=%s/%d", item.Status, item.Version, wantStatus, wantVersion)
	}
	if cancelled := item.CancelledAt != nil; cancelled != wantCancelled {
		f.t.Fatalf("调整取消时间不符: cancelledAt=%v wantCancelled=%t", item.CancelledAt, wantCancelled)
	}
}

// TestCommissionAdjustmentCancelGatePostgres 验证 LOCKED_FEE_SUPPLEMENT 来源的
// 通用取消门禁（上一轮检查 MEDIUM-3 强制验收）：CONFIRMED/PAID 即使绕过页面
// 直接调用通用取消接口也稳定拒绝且零写入；DRAFT 可忽略；MANUAL 来源行为不变。
func TestCommissionAdjustmentCancelGatePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	f := newFeeSupplementFixture(t)
	uc := newAdjustmentTransitionUsecase(f)
	actor := f.approverID

	// 通过真实补录审批链路生成 LFS 调整，保证来源关联与业务事实完整。
	newLFSAdjustment := func(order *ent.Order, key string) *ent.FinanceCommissionAdjustment {
		f.t.Helper()
		// READY + NATIVE 快照父单：历史分母 1000/400，审批后建议金额为正。
		f.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.0000", "1000.00000000", "400.00000000")
		request := f.createSupplement(order, key)
		f.approveSupplement(order, request.ID, 1)
		adjustments := f.adjustmentsForRequest(request.ID)
		if len(adjustments) != 1 {
			f.t.Fatalf("补录审批应生成一条 LFS 建议: got=%d", len(adjustments))
		}
		return adjustments[0]
	}

	t.Run("LFS_DRAFT 可经通用取消接口忽略", func(t *testing.T) {
		f.setT(t)
		order := f.createOrder("SE-CG-DRAFT-" + f.suffix)
		f.lockOrder(order)
		adjustment := newLFSAdjustment(order, "cancel-gate-draft-"+f.suffix)
		cancelled, err := uc.CancelAdjustment(context.Background(), f.organizationID, actor, adjustment.ID, adjustment.Version, "财务忽略建议")
		if err != nil {
			f.t.Fatalf("LFS DRAFT 忽略被拒绝: %v", err)
		}
		if cancelled.Status != biz.CommissionCancelled {
			f.t.Fatalf("忽略后状态不符: %s", cancelled.Status)
		}
		requireAdjustmentState(f, adjustment.ID, "CANCELLED", adjustment.Version+1, true)
	})

	t.Run("LFS_CONFIRMED 直接调用通用取消接口稳定拒绝且零写入", func(t *testing.T) {
		f.setT(t)
		order := f.createOrder("SE-CG-CONF-" + f.suffix)
		f.lockOrder(order)
		adjustment := newLFSAdjustment(order, "cancel-gate-conf-"+f.suffix)
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, adjustment.ID, adjustment.Version); err != nil {
			f.t.Fatalf("确认 LFS 建议失败: %v", err)
		}
		_, err := uc.CancelAdjustment(context.Background(), f.organizationID, actor, adjustment.ID, adjustment.Version+1, "尝试状态洗白")
		if err == nil {
			f.t.Fatal("LFS CONFIRMED 经通用取消接口未拒绝")
		}
		if kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentCancelNotAllowed.Reason {
			f.t.Fatalf("稳定错误码不符: got=%s want=%s", kratoserrors.FromError(err).Reason, biz.ErrCommissionAdjustmentCancelNotAllowed.Reason)
		}
		requireAdjustmentState(f, adjustment.ID, "CONFIRMED", adjustment.Version+1, false)
	})

	t.Run("LFS_PAID 直接调用通用取消接口稳定拒绝且零写入", func(t *testing.T) {
		f.setT(t)
		order := f.createOrder("SE-CG-PAID-" + f.suffix)
		f.lockOrder(order)
		adjustment := newLFSAdjustment(order, "cancel-gate-paid-"+f.suffix)
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, adjustment.ID, adjustment.Version); err != nil {
			f.t.Fatalf("确认 LFS 建议失败: %v", err)
		}
		if _, err := uc.MarkAdjustmentPaid(context.Background(), f.organizationID, actor, adjustment.ID, adjustment.Version+1); err != nil {
			f.t.Fatalf("标记已扣回失败: %v", err)
		}
		_, err := uc.CancelAdjustment(context.Background(), f.organizationID, actor, adjustment.ID, adjustment.Version+2, "尝试状态洗白")
		if err == nil {
			f.t.Fatal("LFS PAID 经通用取消接口未拒绝")
		}
		if kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentCancelNotAllowed.Reason {
			f.t.Fatalf("稳定错误码不符: got=%s want=%s", kratoserrors.FromError(err).Reason, biz.ErrCommissionAdjustmentCancelNotAllowed.Reason)
		}
		requireAdjustmentState(f, adjustment.ID, "PAID", adjustment.Version+2, false)
	})

	t.Run("MANUAL 来源取消行为保持不变", func(t *testing.T) {
		f.setT(t)
		order := f.createOrder("SE-CG-MAN-" + f.suffix)
		parent := f.insertCommission(order, "60.00000000", "40.00000000", "1000.00000000", nil, nil, biz.CommissionBasisRealizedProfit)
		confirmed := insertAdjustmentRow(f, order, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusCONFIRMED, "5.00000000", "cancel-gate-man-conf"+f.suffix, "人工已确认")
		if _, err := uc.CancelAdjustment(context.Background(), f.organizationID, actor, confirmed.ID, 1, "人工调整取消"); err != nil {
			f.t.Fatalf("MANUAL CONFIRMED 取消被拒绝: %v", err)
		}
		requireAdjustmentState(f, confirmed.ID, "CANCELLED", 2, true)
		draft := insertAdjustmentRow(f, order, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "3.00000000", "cancel-gate-man-draft"+f.suffix, "人工草稿")
		if _, err := uc.CancelAdjustment(context.Background(), f.organizationID, actor, draft.ID, 1, "人工草稿取消"); err != nil {
			f.t.Fatalf("MANUAL DRAFT 取消被拒绝: %v", err)
		}
		requireAdjustmentState(f, draft.ID, "CANCELLED", 2, true)
	})
}

// TestCommissionAdjustmentTwoLayerBalancePostgres 验证确认（DRAFT→CONFIRMED）
// 在 Order → 父单 → 调整固定锁序内的双层有效余额校验：不含其他 DRAFT，订单行
// 与父单任一层确认后小于零返回 COMMISSION_ADJUSTMENT_EXCEEDS，同父单其他订单
// 行的正余额不替当前行兜底。
func TestCommissionAdjustmentTwoLayerBalancePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	f := newFeeSupplementFixture(t)
	uc := newAdjustmentTransitionUsecase(f)
	actor := f.approverID
	orderA := f.createOrder("SE-BAL-A-" + f.suffix)
	orderB := f.createOrder("SE-BAL-B-" + f.suffix)

	t.Run("订单行余额不足而父单充足时拒绝确认", func(t *testing.T) {
		f.setT(t)
		// 行 A 60，父单 100（B 行 40 为正余额）：冲减 70 只在订单行层越界；
		// 若缺少订单行校验，父单层校验会放行。
		parent := insertTwoLineCommission(f, orderA, orderB, "60.00000000", "40.00000000")
		draft := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "70.00000000", "bal-line-70"+f.suffix, "订单行超额")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft.ID, draft.Version); err == nil {
			f.t.Fatal("订单行余额不足时未拒绝确认")
		} else if kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentExceeds.Reason {
			f.t.Fatalf("稳定错误码不符: %s", kratoserrors.FromError(err).Reason)
		}
		requireAdjustmentState(f, draft.ID, "DRAFT", draft.Version, false)
	})

	t.Run("确认前重算不含其他 DRAFT 且计入 CONFIRMED 与 PAID", func(t *testing.T) {
		f.setT(t)
		parent := insertTwoLineCommission(f, orderA, orderB, "60.00000000", "40.00000000")
		// 行 A 60 − CONFIRMED 20 = 40：DRAFT 45 越界拒绝，DRAFT 35 放行，
		// 证明其他 DRAFT 不计入、CONFIRMED 计入。
		insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusCONFIRMED, "20.00000000", "bal-recalc-conf"+f.suffix, "已确认冲减")
		draft45 := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "45.00000000", "bal-recalc-45"+f.suffix, "不含草稿重算")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft45.ID, draft45.Version); err == nil {
			f.t.Fatal("确认未排除其他 DRAFT 并按有效余额拒绝")
		} else if kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentExceeds.Reason {
			f.t.Fatalf("稳定错误码不符: %s", kratoserrors.FromError(err).Reason)
		}
		draft35 := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "35.00000000", "bal-recalc-35"+f.suffix, "有效余额内")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft35.ID, draft35.Version); err != nil {
			f.t.Fatalf("有效余额内确认被拒绝: %v", err)
		}
		// PAID 计入有效余额：行 A 剩余 5，PAID 2 后再确认 4 若不计 PAID 可放行
		//（5−4≥0），计入 PAID 后 5−2−4<0 必须拒绝。
		insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusPAID, "2.00000000", "bal-recalc-paid"+f.suffix, "已扣回")
		draft4 := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "4.00000000", "bal-recalc-4"+f.suffix, "PAID 计入")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft4.ID, draft4.Version); err == nil {
			f.t.Fatal("确认未把 PAID 调整计入订单行有效余额")
		} else if kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentExceeds.Reason {
			f.t.Fatalf("稳定错误码不符: %s", kratoserrors.FromError(err).Reason)
		}
		requireAdjustmentState(f, draft4.ID, "DRAFT", draft4.Version, false)
	})

	t.Run("DRAFT_INCREASE 不提前扩张订单行余额", func(t *testing.T) {
		f.setT(t)
		parent := insertTwoLineCommission(f, orderA, orderB, "60.00000000", "40.00000000")
		insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionINCREASE, commissionadjustmentent.StatusDRAFT, "100.00000000", "bal-inc-draft"+f.suffix, "草稿增提")
		insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusCONFIRMED, "50.00000000", "bal-inc-conf"+f.suffix, "已确认冲减")
		draft := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "15.00000000", "bal-inc-15"+f.suffix, "增提不扩张")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft.ID, draft.Version); err == nil {
			f.t.Fatal("确认把 DRAFT INCREASE 计入订单行有效余额")
		} else if kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentExceeds.Reason {
			f.t.Fatalf("稳定错误码不符: %s", kratoserrors.FromError(err).Reason)
		}
		requireAdjustmentState(f, draft.ID, "DRAFT", draft.Version, false)
	})

	t.Run("订单行与父单余额均充足时确认成功", func(t *testing.T) {
		f.setT(t)
		parent := insertTwoLineCommission(f, orderA, orderB, "60.00000000", "40.00000000")
		draft := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, "4.00000000", "bal-ok-4"+f.suffix, "余额内确认")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft.ID, draft.Version); err != nil {
			f.t.Fatalf("余额内确认被拒绝: %v", err)
		}
		requireAdjustmentState(f, draft.ID, "CONFIRMED", draft.Version+1, false)
	})

	t.Run("增提确认不执行余额校验", func(t *testing.T) {
		f.setT(t)
		parent := insertTwoLineCommission(f, orderA, orderB, "60.00000000", "40.00000000")
		draft := insertAdjustmentRow(f, orderB, parent, commissionadjustmentent.DirectionINCREASE, commissionadjustmentent.StatusDRAFT, "5.00000000", "bal-inc-ok"+f.suffix, "增提确认")
		if _, err := uc.ConfirmAdjustment(context.Background(), f.organizationID, actor, draft.ID, draft.Version); err != nil {
			f.t.Fatalf("增提确认被拒绝: %v", err)
		}
		requireAdjustmentState(f, draft.ID, "CONFIRMED", draft.Version+1, false)
	})
}

// TestCommissionAdjustmentMySourcePostgres 验证员工本人专属冲减来源最小详情：
// 授权同时限定 employee_id 与启用组织成员关系，本人可读取最小字段集，他人或
// 缺成员关系时稳定返回不存在，不泄露记录事实。
func TestCommissionAdjustmentMySourcePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	f := newFeeSupplementFixture(t)
	uc := newAdjustmentTransitionUsecase(f)

	order := f.createOrder("SE-SRC-" + f.suffix)
	f.lockOrder(order)
	f.createConfirmedCommissionWithSnapshot(order, "60.00000000", "10.0000", "1000.00000000", "400.00000000")
	request := f.createSupplement(order, "my-source-"+f.suffix)
	f.approveSupplement(order, request.ID, 1)
	adjustments := f.adjustmentsForRequest(request.ID)
	if len(adjustments) != 1 {
		f.t.Fatalf("补录审批应生成一条 LFS 建议: got=%d", len(adjustments))
	}
	adjustment := adjustments[0]
	principal := func(userID uuid.UUID) *biz.Principal {
		return &biz.Principal{UserID: userID, Organization: biz.Organization{ID: f.organizationID}}
	}

	t.Run("员工本人缺少组织成员关系时按不存在处理", func(t *testing.T) {
		f.setT(t)
		if _, err := uc.GetMyFeeSupplementAdjustmentSource(f.ctx, principal(f.employeeID), adjustment.ID); err == nil ||
			kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentNotFound.Reason {
			f.t.Fatalf("缺成员关系查询未稳定返回不存在: %v", err)
		}
	})

	// 员工本人组织成员关系：归属判定需要启用成员关系。
	if _, err := f.data.db.Membership.Create().
		SetUserID(f.employeeID).
		SetOrganizationID(f.organizationID).
		SetEnabled(true).
		Save(f.ctx); err != nil {
		f.t.Fatalf("创建员工成员关系: %v", err)
	}

	t.Run("本人可读取最小来源详情", func(t *testing.T) {
		f.setT(t)
		source, err := uc.GetMyFeeSupplementAdjustmentSource(f.ctx, principal(f.employeeID), adjustment.ID)
		if err != nil {
			f.t.Fatalf("本人读取来源详情被拒绝: %v", err)
		}
		if source.AdjustmentID != adjustment.ID || source.OrderNo != order.OrderNo || source.CommissionNo != adjustment.CommissionNo {
			f.t.Fatalf("来源详情标识不符: %+v", source)
		}
		if source.Status != biz.CommissionDraft || source.SuggestedAmount.Sign() <= 0 {
			f.t.Fatalf("来源详情状态或建议金额不符: status=%s amount=%s", source.Status, source.SuggestedAmount)
		}
		if source.FeeExpenseDate != "2026-09-01" || source.FeeCurrency != "CNY" || source.FeeTotalAmount.Sign() <= 0 || source.SupplementReason == "" {
			f.t.Fatalf("补录费用摘要不符: %+v", source)
		}
	})

	t.Run("其他用户查询按不存在处理", func(t *testing.T) {
		f.setT(t)
		other := f.data.db.User.Create().
			SetDisplayName("无关用户-" + f.suffix).
			SetEnabled(true).
			SetIsBootstrapAdmin(false).
			SaveX(f.ctx)
		if _, err := uc.GetMyFeeSupplementAdjustmentSource(f.ctx, principal(other.ID), adjustment.ID); err == nil ||
			kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentNotFound.Reason {
			f.t.Fatalf("他人查询未稳定返回不存在: %v", err)
		}
	})
}
