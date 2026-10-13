package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

type blockingCorrectionCommissionRepo struct {
	biz.FeeCorrectionCommissionRepo
	entered chan struct{}
	release chan struct{}
}

func (r blockingCorrectionCommissionRepo) ReconcileCorrectionImpact(ctx context.Context, org, order, actor uuid.UUID, request *biz.OrderFeeCorrectionRequest, impact *biz.FeeCorrectionCommissionImpact) error {
	close(r.entered)
	select {
	case <-r.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	return r.FeeCorrectionCommissionRepo.ReconcileCorrectionImpact(ctx, org, order, actor, request, impact)
}

func TestFeeCorrectionApprovalSerializesCommissionPaymentPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	ctx, cancel := context.WithTimeout(f.ctx, 15*time.Second)
	defer cancel()
	normal := newCorrectionTestUsecase(t, f)
	o := f.createOrder("PAYMENT-CORR-" + f.suffix)
	f.approveInitialFees(o)
	parent := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	commissionRepo := NewCommissionRepo(f.data)
	oldPayment, err := commissionRepo.PreviewPayment(ctx, f.organizationID, parent.ID, parent.Version)
	if err != nil {
		t.Fatal(err)
	}
	before := correctionTestSource(t, f, o, biz.OrderFeePayable)
	request, err := normal.Submit(ctx, f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "REPLACE", "payment-correction", before, 900), false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := normal.Preview(ctx, f.approverPrincipal(), f.organizationID, o.ID, request.ID, request.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.CommissionImpact.Lines) != 1 || !preview.CommissionImpact.Lines[0].TargetAmount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("更正后目标应10：%+v", preview.CommissionImpact)
	}
	blocker := blockingCorrectionCommissionRepo{FeeCorrectionCommissionRepo: NewFeeCorrectionCommissionRepo(f.data), entered: make(chan struct{}), release: make(chan struct{})}
	defer func() {
		select {
		case <-blocker.release:
		default:
			close(blocker.release)
		}
	}()
	uc := biz.NewOrderFeeCorrectionUsecase(NewOrderFeeCorrectionRepo(f.data), normalFeeUsecaseForRollback(f), blocker, f.data)
	approvalResult := make(chan error, 1)
	go func() {
		_, err := uc.Approve(ctx, f.approverPrincipal(), f.organizationID, o.ID, request.ID, request.Version, preview.PreviewToken, "费用改为900")
		approvalResult <- err
	}()
	select {
	case <-blocker.entered:
	case err := <-approvalResult:
		t.Fatalf("批准未进入锁内重算：%v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	paymentResult := make(chan error, 1)
	go func() {
		_, err := commissionRepo.Pay(ctx, f.organizationID, parent.ID, f.approverID, parent.Version, biz.CommissionPaymentInput{PreviewToken: oldPayment.PreviewToken, ActualAmount: oldPayment.ActualAmount, IdempotencyKey: "old-payment"}, &biz.AuditEvent{Action: "test.payment", Result: "success"})
		paymentResult <- err
	}()
	// 从 PostgreSQL 的真实等待状态确认付款正在等同一Schema的Order锁。
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		err = f.data.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM pg_locks l JOIN pg_stat_activity a ON a.pid = l.pid
WHERE l.relation = 'orders'::regclass AND a.wait_event_type = 'Lock'
AND a.pid <> pg_backend_pid() AND a.query LIKE '%"orders"%')`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case err = <-paymentResult:
			t.Fatalf("批准持Order锁时付款提前返回：%v", err)
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("付款未进入预期Order锁等待", ctx.Err())
		}
	}
	close(blocker.release)
	if err = <-approvalResult; err != nil {
		t.Fatal(err)
	}
	if err = <-paymentResult; err == nil {
		t.Fatal("审批后不得按旧60预览付款")
	}
	current, err := commissionRepo.Get(ctx, f.organizationID, parent.ID)
	if err != nil || current.Status != biz.CommissionConfirmed || !current.EffectiveCommissionAmount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("旧付款失败后须保持待付净额10：%+v,%v", current, err)
	}
	fresh, err := commissionRepo.PreviewPayment(ctx, f.organizationID, parent.ID, current.Version)
	if err != nil || !fresh.ActualAmount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("刷新付款须是净额10：%+v,%v", fresh, err)
	}
	paid, err := commissionRepo.Pay(ctx, f.organizationID, parent.ID, f.approverID, current.Version, biz.CommissionPaymentInput{PreviewToken: fresh.PreviewToken, ActualAmount: fresh.ActualAmount, IdempotencyKey: "fresh-payment"}, &biz.AuditEvent{Action: "test.payment", Result: "success"})
	if err != nil || paid.PaidAmount == nil || !paid.PaidAmount.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("应实际净付10：%+v,%v", paid, err)
	}
}
