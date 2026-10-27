package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
)

// 完结必须允许录费后、审核锁单与建账前办理，且不能改变费用的账单关联。
func TestOrderClosureBeforeReviewAndBillingPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	newCorrectionTestUsecase(t, f)
	for _, businessType := range []orderent.BusinessType{orderent.BusinessTypeSE, orderent.BusinessTypeSI} {
		t.Run(string(businessType), func(t *testing.T) {
			f.setT(t)
			o := f.createOrder("CLOSE-BEFORE-BILL-" + uuid.NewString())
			fee := addOrdinaryTestFee(t, f, o, 100)
			o = f.data.db.Order.UpdateOneID(o.ID).SetBusinessType(businessType).SaveX(f.ctx)
			if o.LockedAt != nil || o.FlowStatus != orderent.FlowStatusDRAFT {
				t.Fatal("夹具应为未审核锁定的草稿订单")
			}
			repo := NewOrderRepo(f.data)
			uc := biz.NewOrderUsecase(repo, nil, nil, nil, nil, nil)
			closed, err := uc.TransitionClosure(f.ctx, f.organizationID, f.approverID, o.ID, o.Version, biz.OrderClosureClosed, "资料与费用录入完成")
			if err != nil || closed.ClosureStatus != biz.OrderClosureClosed || closed.FlowStatus != biz.OrderFlowDocumentReleased || closed.LockedAt != nil || closed.Version != o.Version+1 {
				t.Fatalf("未建账未审核订单应可完结并补齐主流程: order=%+v err=%v", closed, err)
			}
			storedFee := f.data.db.OrderFee.GetX(f.ctx, fee.ID)
			if storedFee.Version != fee.Version || f.data.db.OrderFee.Query().Where(feeent.IDEQ(fee.ID), feeent.HasFinanceBillLines()).ExistX(f.ctx) {
				t.Fatal("完结不得修改费用或自动建立账单")
			}
			if count := f.data.db.OrderLifecycleEvent.Query().Where(eventent.OrderIDEQ(o.ID), eventent.DimensionEQ(eventent.DimensionCLOSURE)).CountX(f.ctx); count != 1 {
				t.Fatalf("应写入一条完结事件: %d", count)
			}
			// 非海运共享单证品类复用同一结案流程，继续验证真实人工审核与建账准入。
			if businessType == orderent.BusinessTypeSI {
				lockRepo := NewOrderLockRepo(f.data, nil)
				if _, err := lockRepo.LockOrder(f.ctx, f.approverPrincipal(), o.ID, closed.Version, uuid.NewString(), lockTestReviewInput(t, f.data, f.ctx, o.ID), nil); err != nil {
					t.Fatalf("未建账订单完结后应可审核锁单: %v", err)
				}
				locked := f.data.db.Order.GetX(f.ctx, o.ID)
				if locked.ClosureStatus != orderent.ClosureStatusCLOSED || locked.LockedAt == nil {
					t.Fatal("审核应保留完结状态并锁单")
				}
				billable, err := NewFinanceBillRepo(f.data).LoadBillableFees(f.ctx, f.organizationID, []uuid.UUID{fee.ID})
				if err != nil || len(billable) != 1 || !billable[0].CanBill {
					t.Fatalf("完结并审核锁单后费用应可建账: fees=%+v err=%v", billable, err)
				}
			}
		})
	}
}
