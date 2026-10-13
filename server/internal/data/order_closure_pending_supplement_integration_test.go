package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
)

// 夹具固化有效账单关联，不添加收付款事实；结案只检查费用已建账。
func billClosureTestFees(t *testing.T, f *feeSupplementPostgresFixture, o *ent.Order) {
	t.Helper()
	fees, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(o.ID), feeent.Not(feeent.HasFinanceBillLinesWith(effectiveBillLinePredicate()))).All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, fee := range fees {
		key := uuid.NewString()
		create := f.data.db.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo("CLOSE-" + key).SetIdempotencyKey(key).
			SetDirection(billent.Direction(fee.Direction)).SetStatus(billent.StatusCONFIRMED).SetSettlementPartyID(f.customerID).SetSettlementPartyName("结案测试结算单位").
			SetCurrency(fee.Currency).SetBaseCurrency(fee.BaseCurrency).SetExchangeRate(fee.ExchangeRate).SetExchangeRateSource(billent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(fee.ExchangeRateDate).
			SetTotalAmount(fee.TotalAmount).SetNetAmount(fee.NetAmount).SetTaxAmount(fee.TaxAmount).SetBaseCurrencyAmount(fee.BaseCurrencyAmount).SetFeeCount(1).SetBillDate(fee.ExpenseDate)
		bill, err := withTestFinanceBillSettlementAccountSnapshot(create, uuid.New(), fee.Currency).Save(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.data.db.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderFeeID(fee.ID).SetOrderID(o.ID).SetOrderNo(o.OrderNo).
			SetFeeCode(fee.FeeCode).SetFeeName(fee.FeeName).SetQuantity(fee.Quantity).SetUnitPrice(fee.UnitPrice).SetTotalAmount(fee.TotalAmount).SetNetAmount(fee.NetAmount).SetTaxAmount(fee.TaxAmount).
			SetCurrency(fee.Currency).SetExchangeRate(fee.ExchangeRate).SetBaseCurrency(fee.BaseCurrency).SetBaseCurrencyAmount(fee.BaseCurrencyAmount).Save(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func readyClosureTestOrder(t *testing.T, f *feeSupplementPostgresFixture) *ent.Order {
	t.Helper()
	o := f.createOrder("CLOSE-" + uuid.NewString())
	f.approveInitialFees(o)
	billClosureTestFees(t, f, o)
	o, err := f.data.db.Order.UpdateOneID(o.ID).SetFlowStatus(orderent.FlowStatusDOCUMENT_RELEASED).Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func transitionClosureForTest(ctx context.Context, f *feeSupplementPostgresFixture, o *ent.Order, target biz.OrderClosureStatus) (*biz.Order, error) {
	from := biz.OrderClosureOpen
	if target == biz.OrderClosureOpen {
		from = biz.OrderClosureClosed
	}
	event := &biz.OrderLifecycleChangedEvent{OrganizationID: f.organizationID, OrderID: o.ID, ActorID: f.requesterID, Dimension: "CLOSURE", FromStatus: string(from), ToStatus: string(target), OccurredAt: time.Now().UTC()}
	return NewOrderRepo(f.data).TransitionClosure(ctx, f.organizationID, o.ID, o.Version, target, "核对业务完成", f.requesterID, event)
}

func requireClosureBlocked(t *testing.T, err error, pending bool) {
	t.Helper()
	if err == nil {
		t.Fatal("结案必须被阻断")
	}
	detail := errors.FromError(err)
	if detail.Code != 409 || detail.Reason != "ORDER_CLOSURE_BLOCKED" {
		t.Fatalf("结案错误不符: %v", err)
	}
	if pending && detail.Message != "存在待审核补录，请先处理后再结案" {
		t.Fatalf("缺少待审具体原因: %v", err)
	}
}

func TestOrderClosurePendingSupplementReadinessPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := readyClosureTestOrder(t, f)
	r := f.createSupplement(o, "pending-close")
	repo := NewOrderRepo(f.data)
	readiness, err := repo.ClosureReadiness(f.ctx, f.organizationID, o.ID)
	if err != nil || !readiness.HasPendingFeeSupplement || readiness.HasUnbilledOrderFees {
		t.Fatalf("待审应独立于正式费用: readiness=%+v err=%v", readiness, err)
	}
	// 即使没有费用金额读取权限，订单投影也应呈现待审事实。
	projected, err := repo.Get(biz.WithPrincipal(f.ctx, f.requesterPrincipal()), f.organizationID, o.ID)
	if err != nil || projected.FeeReview == nil || !projected.FeeReview.HasPendingSupplement || projected.FeeReview.Summary != nil {
		t.Fatalf("无费用读权限的待审投影: order=%+v err=%v", projected, err)
	}
	auditBefore, err := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 直接调用仓储，模拟领域预检后才提交的补录，证明锁内门禁有效。
	_, err = transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed)
	requireClosureBlocked(t, err, true)
	after, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil || after.Version != o.Version || after.ClosureStatus != orderent.ClosureStatusOPEN || after.ClosedAt != nil {
		t.Fatalf("拒绝应保留订单: %+v err=%v", after, err)
	}
	events, err := f.data.db.OrderLifecycleEvent.Query().Where(eventent.OrderIDEQ(o.ID)).Count(f.ctx)
	if err != nil || events != 0 {
		t.Fatalf("拒绝不得创建结案事件: %d err=%v", events, err)
	}
	auditAfter, err := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if err != nil || auditAfter != auditBefore {
		t.Fatalf("拒绝不得创建成功审计: before=%d after=%d err=%v", auditBefore, auditAfter, err)
	}
	if _, err := f.usecase.Withdraw(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID, r.Version); err != nil {
		t.Fatal(err)
	}
	r = f.createSupplement(o, "reject-close")
	reason := "费用依据不足"
	if _, err := f.usecase.Reject(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, &reason); err != nil {
		t.Fatal(err)
	}
	readiness, err = repo.ClosureReadiness(f.ctx, f.organizationID, o.ID)
	if err != nil || readiness.HasPendingFeeSupplement {
		t.Fatalf("撤回及退回应清除待审: %+v err=%v", readiness, err)
	}
	f.lockOrder(o)
	o, err = f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed)
	if err != nil || closed.ClosureStatus != biz.OrderClosureClosed {
		t.Fatalf("没有回款的业务锁订单应可结案: %v", err)
	}
}

func TestOrderClosurePendingSupplementClosedWorkflowPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := readyClosureTestOrder(t, f)
	parent := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	if _, err := transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed); err != nil {
		t.Fatal(err)
	}
	r := f.createSupplement(o, "closed-withdraw")
	if _, err := f.usecase.Withdraw(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID, r.Version); err != nil {
		t.Fatal(err)
	}
	r = f.createSupplement(o, "closed-reject")
	reason := "供应商重复提供费用"
	if _, err := f.usecase.Reject(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, &reason); err != nil {
		t.Fatal(err)
	}
	r = f.createSupplement(o, "closed-approve")
	if _, err := f.usecase.PreviewApproval(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID, r.Version); err != biz.ErrPermissionDenied {
		t.Fatalf("结案不能绕过审批权限: %v", err)
	}
	result := f.approveSupplement(o, r.ID, r.Version)
	adjustments := f.adjustmentsForRequest(r.ID)
	if result.Fee == nil || len(adjustments) != 1 || adjustments[0].CommissionID != parent.ID || string(adjustments[0].Status) != "CONFIRMED" {
		t.Fatalf("已结案补录须创建费用与确认冲减: %+v adjustments=%+v", result, adjustments)
	}
	after, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil || after.ClosureStatus != orderent.ClosureStatusCLOSED {
		t.Fatalf("补录不得自动反结案: %+v err=%v", after, err)
	}
	_, err = NewOrderFeeRepo(f.data).Update(f.ctx, f.organizationID, o.ID, result.Fee.ID, result.Fee, nil, &biz.AuditEvent{Action: "test.fee.update", Result: "success"})
	if err == nil {
		t.Fatal("结案后受控补录不授权普通费用修改")
	}
	// 有待审补录时仍允许反结案；随后结案必须阻断。
	pending := f.createSupplement(o, "closed-reopen")
	if _, err := transitionClosureForTest(f.ctx, f, after, biz.OrderClosureOpen); err != nil {
		t.Fatalf("有待审时反结案: %v", err)
	}
	if _, err := f.usecase.Withdraw(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, pending.ID, pending.Version); err != nil {
		t.Fatal(err)
	}
	after, err = f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = transitionClosureForTest(f.ctx, f, after, biz.OrderClosureClosed)
	requireClosureBlocked(t, err, false)
	billClosureTestFees(t, f, after)
	if _, err := transitionClosureForTest(f.ctx, f, after, biz.OrderClosureClosed); err != nil {
		t.Fatalf("补录费用建账后应可结案: %v", err)
	}
}

type closureOrderProbeKey struct{}

// 首操作写入并持有真实 Order 锁，等次操作开始查询后才提交；不靠 sleep 猜顺序。
func runClosureOperationsInOrder(t *testing.T, f *feeSupplementPostgresFixture, first, second func(context.Context) error) (error, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	marker := new(int)
	querying := make(chan struct{})
	var once sync.Once
	f.data.db.Order.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
			if ctx.Value(closureOrderProbeKey{}) == marker {
				once.Do(func() { close(querying) })
			}
			return next.Query(ctx, query)
		})
	}))
	applied := make(chan struct{})
	firstResult, secondResult := make(chan error, 1), make(chan error, 1)
	go func() {
		firstResult <- f.data.WithinTransaction(ctx, func(txCtx context.Context) error {
			if err := first(txCtx); err != nil {
				return err
			}
			close(applied)
			select {
			case <-querying:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	go func() {
		select {
		case <-applied:
			secondResult <- second(context.WithValue(ctx, closureOrderProbeKey{}, marker))
		case <-ctx.Done():
			secondResult <- ctx.Err()
		}
	}()
	return <-firstResult, <-secondResult
}

func TestOrderClosurePendingSupplementOrderedConcurrencyPostgres(t *testing.T) {
	for _, sequence := range []string{"补录先", "结案先", "批准先"} {
		t.Run(sequence, func(t *testing.T) {
			f := newFeeSupplementFixture(t)
			o := readyClosureTestOrder(t, f)
			closeOrder := func(ctx context.Context) error {
				_, err := transitionClosureForTest(ctx, f, o, biz.OrderClosureClosed)
				return err
			}
			create := func(ctx context.Context) error {
				_, err := f.usecase.Create(ctx, f.requesterPrincipal(), f.organizationID, o.ID, f.supplementInput("ordered"), false)
				return err
			}
			first, second := create, closeOrder
			if sequence == "结案先" {
				first, second = closeOrder, create
			}
			if sequence == "批准先" {
				r := f.createSupplement(o, "approve-first")
				preview, err := f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
				if err != nil {
					t.Fatal(err)
				}
				first = func(ctx context.Context) error {
					_, err := f.usecase.Approve(ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "成本核实无误")
					return err
				}
			}
			firstErr, secondErr := runClosureOperationsInOrder(t, f, first, second)
			if firstErr != nil {
				t.Fatalf("首操作应成功: %v", firstErr)
			}
			if sequence == "结案先" {
				if secondErr != nil {
					t.Fatalf("结案后补录应允许: %v", secondErr)
				}
			} else {
				requireClosureBlocked(t, secondErr, sequence == "补录先")
			}
			latest, err := f.data.db.Order.Get(f.ctx, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := orderent.ClosureStatusOPEN
			if sequence == "结案先" {
				want = orderent.ClosureStatusCLOSED
			}
			if latest.ClosureStatus != want {
				t.Fatalf("结案状态应按实际顺序: got=%s want=%s", latest.ClosureStatus, want)
			}
		})
	}
}
