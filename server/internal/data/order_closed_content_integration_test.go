package data

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/shopspring/decimal"
)

// 完结只标识业务进度，人工审核锁与有效账单占用分别决定普通费用写入资格。
func TestClosedOrderFeeContentEditableUntilReviewPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	_ = newCorrectionTestUsecase(t, f)
	o := f.createOrder("CLOSED-EDIT-" + f.suffix)
	closed, err := NewOrderRepo(f.data).TransitionClosure(f.ctx, f.organizationID, o.ID, o.Version, biz.OrderClosureClosed, "业务完成待核对", f.approverID, &biz.OrderLifecycleChangedEvent{OrganizationID: f.organizationID, OrderID: o.ID, ActorID: f.approverID, Dimension: "CLOSURE", FromStatus: "OPEN", ToStatus: "CLOSED", OccurredAt: time.Now().UTC()})
	if err != nil || closed.ClosureStatus != biz.OrderClosureClosed {
		t.Fatalf("完结订单失败: order=%+v err=%v", closed, err)
	}
	fee := addOrdinaryTestFee(t, f, o, 100)
	fee.UnitPrice = decimal.NewFromInt(120)
	fee, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, fee, false)
	if err != nil || !fee.UnitPrice.Equal(decimal.NewFromInt(120)) {
		t.Fatalf("完结未锁未建账费用应可修改: fee=%+v err=%v", fee, err)
	}
	ctx := biz.WithPrincipal(f.ctx, f.approverPrincipal())
	assertCapabilities := func(wantEdit, wantCorrect bool) {
		t.Helper()
		projected, readErr := NewOrderFeeRepo(f.data).Get(ctx, f.organizationID, o.ID, fee.ID)
		if readErr != nil || projected.CanEdit != wantEdit || projected.CanDelete != wantEdit || projected.CanCorrect != wantCorrect {
			t.Fatalf("费用资格应与执行一致: fee=%+v err=%v", projected, readErr)
		}
	}
	assertCapabilities(true, false)
	bill := createOccupiedTestFeeBill(t, f, o, fee)
	assertCapabilities(false, false)
	if _, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, fee, false); err != biz.ErrOrderFeeBillOccupied {
		t.Fatalf("已建草稿账单仍须阻止直接改费: %v", err)
	}
	if _, err = NewFinanceBillRepo(f.data).Cancel(f.ctx, []uuid.UUID{f.organizationID}, bill.ID, f.approverID, bill.Version, "按权限回退至未建账", &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.approverID, Action: "finance.bill.cancel", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	assertCapabilities(true, false)
	fee, err = NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, fee.ID)
	if err != nil {
		t.Fatal(err)
	}
	fee.UnitPrice = decimal.NewFromInt(130)
	fee, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, fee, false)
	if err != nil {
		t.Fatalf("回退至未建账后应可修改: %v", err)
	}
	deletable := addOrdinaryTestFee(t, f, o, 10)
	if err = normalFeeUsecaseForRollback(f).Remove(f.ctx, f.organizationID, f.approverID, o.ID, deletable.ID, deletable.Version, "核对后删除"); err != nil {
		t.Fatalf("完结未锁未建账费用应可删除: %v", err)
	}
	seedManualOrderReviewForTest(t, f.data, o.ID, f.approverID)
	assertCapabilities(false, true)
	if _, err = normalFeeUsecaseForRollback(f).Update(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, fee, false); errors.FromError(err).Reason != "ORDER_ALREADY_LOCKED" {
		t.Fatalf("审核锁定后普通改费必须拒绝: %v", err)
	}
	stored, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil || stored.ClosureStatus != orderent.ClosureStatusCLOSED || stored.LockedAt == nil {
		t.Fatalf("最终状态应保持完结并审核锁定: order=%+v err=%v", stored, err)
	}
}
