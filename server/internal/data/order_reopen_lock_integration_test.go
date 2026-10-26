package data

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
)

func TestOrderReopenRequiresUnlockPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	newCorrectionTestUsecase(t, f)
	o := f.createOrder("REOPEN-LOCK-" + f.suffix)
	closed, err := transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed)
	if err != nil {
		t.Fatal(err)
	}
	lockRepo := NewOrderLockRepo(f.data, nil)
	review := lockTestReviewInput(t, f.data, f.ctx, o.ID)
	eventsBefore := f.data.db.OrderLifecycleEvent.Query().Where(eventent.OrderIDEQ(o.ID)).CountX(f.ctx)
	auditsBefore := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID), auditent.ActionEQ("order.closure.transition")).CountX(f.ctx)
	// 审核事务持有 Order 行锁；反结案查询开始后才提交审核，验证等待者读取新锁状态。
	lockErr, reopenErr := runClosureOperationsInOrder(t, f, func(ctx context.Context) error {
		_, err := lockRepo.LockOrder(ctx, f.approverPrincipal(), o.ID, closed.Version, uuid.NewString(), review, nil)
		return err
	}, func(ctx context.Context) error {
		_, err := NewOrderRepo(f.data).TransitionClosure(ctx, f.organizationID, o.ID, closed.Version+1, biz.OrderClosureOpen, "", f.approverID, &biz.OrderLifecycleChangedEvent{OrganizationID: f.organizationID, OrderID: o.ID, ActorID: f.approverID, Dimension: "CLOSURE", FromStatus: "CLOSED", ToStatus: "OPEN", OccurredAt: time.Now().UTC()})
		return err
	})
	if lockErr != nil || errors.FromError(reopenErr).Reason != "ORDER_BUSINESS_LOCKED" {
		t.Fatalf("审核先提交后反结案须读取锁并拒绝: lock=%v reopen=%v", lockErr, reopenErr)
	}
	stored := f.data.db.Order.GetX(f.ctx, o.ID)
	if stored.LockedAt == nil || stored.Version != closed.Version+1 || stored.ClosureStatus != orderent.ClosureStatusCLOSED || f.data.db.OrderLifecycleEvent.Query().Where(eventent.OrderIDEQ(o.ID)).CountX(f.ctx) != eventsBefore || f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID), auditent.ActionEQ("order.closure.transition")).CountX(f.ctx) != auditsBefore {
		t.Fatal("被拒反结案不得改变订单、版本、生命周期事件或审计")
	}
	if _, err = lockRepo.RequestOrderUnlock(f.ctx, f.approverPrincipal(), o.ID, stored.Version, uuid.NewString(), nil, nil); err != nil {
		t.Fatalf("审核人员直接解锁失败: %v", err)
	}
	stored = f.data.db.Order.GetX(f.ctx, o.ID)
	uc := biz.NewOrderUsecase(NewOrderRepo(f.data), nil, nil, nil, nil, nil)
	reopened, err := uc.TransitionClosure(f.ctx, f.organizationID, f.approverID, o.ID, stored.Version, biz.OrderClosureOpen, "  ")
	if err != nil || reopened.ClosureStatus != biz.OrderClosureOpen || reopened.Version != stored.Version+1 {
		t.Fatalf("解锁后应可空原因反结案: order=%+v err=%v", reopened, err)
	}
	event := f.data.db.OrderLifecycleEvent.Query().Where(eventent.OrderIDEQ(o.ID), eventent.ToStatusEQ("OPEN")).OnlyX(f.ctx)
	if event.Reason != nil && *event.Reason != "" {
		t.Fatalf("空原因不得伪造默认原因: %q", *event.Reason)
	}
}
