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

// 夹具只为结案验证建立账单关联；结案不以收付款完成为前提。
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
			SetTotalAmount(fee.TotalAmount).SetNetAmount(fee.NetAmount).SetTaxAmount(fee.TaxAmount).SetBaseCurrencyAmount(fee.BaseCurrencyAmount).SetFeeCount(1).SetBillDate(fee.ExchangeRateDate)
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

func submitClosureCorrection(t *testing.T, f *feeSupplementPostgresFixture, uc *biz.OrderFeeCorrectionUsecase, o *ent.Order, key string) *biz.OrderFeeCorrectionRequest {
	t.Helper()
	r, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "ADD", key, nil, 100), false)
	if err != nil {
		t.Fatalf("提交待补审费用: %v", err)
	}
	return r
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
	if pending && detail.Message != "存在待审费用变更，请先处理后再结案" {
		t.Fatalf("缺少待审具体原因: %v", err)
	}
}

func TestOrderClosurePendingCorrectionReadinessPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newCorrectionTestUsecase(t, f)
	o := readyClosureTestOrder(t, f)
	r := submitClosureCorrection(t, f, uc, o, "pending-close")
	repo := NewOrderRepo(f.data)
	readiness, err := repo.ClosureReadiness(f.ctx, f.organizationID, o.ID)
	if err != nil || !readiness.HasPendingFeeSupplement || readiness.HasUnbilledOrderFees {
		t.Fatalf("待补审应独立于正式费用: readiness=%+v err=%v", readiness, err)
	}
	projected, err := repo.Get(biz.WithPrincipal(f.ctx, f.requesterPrincipal()), f.organizationID, o.ID)
	if err != nil || projected.FeeReview == nil || !projected.FeeReview.HasPendingCorrection || projected.FeeReview.Summary != nil {
		t.Fatalf("无费用读取权限仍应看见待补审事实: order=%+v err=%v", projected, err)
	}
	auditBefore, err := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := uc.Decide(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, "WITHDRAWN", "撤回申请"); err != nil {
		t.Fatal(err)
	}
	r = submitClosureCorrection(t, f, uc, o, "reject-close")
	if _, err := uc.Decide(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, "REJECTED", "费用依据不足"); err != nil {
		t.Fatal(err)
	}
	readiness, err = repo.ClosureReadiness(f.ctx, f.organizationID, o.ID)
	if err != nil || readiness.HasPendingFeeSupplement {
		t.Fatalf("撤回与退回应清除待补审: %+v err=%v", readiness, err)
	}
	closed, err := transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed)
	if err != nil || closed.ClosureStatus != biz.OrderClosureClosed {
		t.Fatalf("待补审结束后应可结案: %v", err)
	}
}

func TestOrderClosurePendingCorrectionClosedWorkflowPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newCorrectionTestUsecase(t, f)
	o := readyClosureTestOrder(t, f)
	if _, err := transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed); err != nil {
		t.Fatal(err)
	}
	r := submitClosureCorrection(t, f, uc, o, "closed-approve")
	approved := approveCorrectionForTest(t, f, uc, o, r)
	if approved.ApprovalResult == nil {
		t.Fatalf("结案后批准应生成正式费用: %+v", approved)
	}
	generatedFee, err := f.data.db.OrderFee.Query().Where(feeent.CorrectionRequestIDEQ(r.ID)).Only(f.ctx)
	if err != nil {
		t.Fatalf("结案后批准应生成正式费用: %v", err)
	}
	after, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil || after.ClosureStatus != orderent.ClosureStatusCLOSED {
		t.Fatalf("补审不得自动反结案: %+v err=%v", after, err)
	}
	_, err = NewOrderFeeRepo(f.data).Update(f.ctx, f.organizationID, o.ID, generatedFee.ID, &biz.OrderFee{}, nil, &biz.AuditEvent{Action: "test.fee.update", Result: "success"})
	if err == nil {
		t.Fatal("结案后不得普通修改费用")
	}
	pending := submitClosureCorrection(t, f, uc, o, "closed-reopen")
	if _, err := transitionClosureForTest(f.ctx, f, after, biz.OrderClosureOpen); errors.FromError(err).Reason != "ORDER_BUSINESS_LOCKED" {
		t.Fatalf("有待补审且锁定时反结案仍须拒绝: %v", err)
	}
	if _, err := uc.Decide(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, pending.ID, pending.Version, "WITHDRAWN", "撤回申请"); err != nil {
		t.Fatal(err)
	}
	after, err = f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = NewOrderLockRepo(f.data, nil).RequestOrderUnlock(f.ctx, f.approverPrincipal(), o.ID, after.Version, uuid.NewString(), nil, nil); err != nil {
		t.Fatalf("待审撤回后应可解锁: %v", err)
	}
	after = f.data.db.Order.GetX(f.ctx, o.ID)
	if _, err = transitionClosureForTest(f.ctx, f, after, biz.OrderClosureOpen); err != nil {
		t.Fatalf("解锁后应可反结案: %v", err)
	}
	after = f.data.db.Order.GetX(f.ctx, o.ID)
	_, err = transitionClosureForTest(f.ctx, f, after, biz.OrderClosureClosed)
	requireClosureBlocked(t, err, false)
	billClosureTestFees(t, f, after)
	if _, err := transitionClosureForTest(f.ctx, f, after, biz.OrderClosureClosed); err != nil {
		t.Fatalf("补审费用建账后应可结案: %v", err)
	}
}

type closureOrderProbeKey struct{}

// 首操作持有真实订单行锁，次操作查询开始后才提交，验证结案与补审写入顺序。
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

func TestOrderClosurePendingCorrectionOrderedConcurrencyPostgres(t *testing.T) {
	for _, sequence := range []string{"补审申请先", "结案先", "补审批准先"} {
		t.Run(sequence, func(t *testing.T) {
			f := newFeeSupplementFixture(t)
			uc := newCorrectionTestUsecase(t, f)
			o := readyClosureTestOrder(t, f)
			closeOrder := func(ctx context.Context) error {
				_, err := transitionClosureForTest(ctx, f, o, biz.OrderClosureClosed)
				return err
			}
			create := func(ctx context.Context) error {
				_, err := uc.Submit(ctx, f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "ADD", uuid.NewString(), nil, 100), false)
				return err
			}
			first, second := create, closeOrder
			if sequence == "结案先" {
				first, second = closeOrder, create
			}
			if sequence == "补审批准先" {
				r := submitClosureCorrection(t, f, uc, o, "approve-first")
				preview, err := uc.Preview(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
				if err != nil {
					t.Fatal(err)
				}
				first = func(ctx context.Context) error {
					_, err := uc.Approve(ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "已核对费用")
					return err
				}
			}
			firstErr, secondErr := runClosureOperationsInOrder(t, f, first, second)
			if firstErr != nil {
				t.Fatalf("首操作应成功: %v", firstErr)
			}
			if sequence == "结案先" {
				if secondErr != nil {
					t.Fatalf("结案后仍允许补审申请: %v", secondErr)
				}
			} else {
				requireClosureBlocked(t, secondErr, sequence == "补审申请先")
			}
		})
	}
}
