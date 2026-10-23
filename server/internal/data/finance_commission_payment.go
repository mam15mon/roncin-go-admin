package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	correction "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	supplement "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	"github.com/shopspring/decimal"
)

// commissionCashExposure 只以持久化付款快照及实际独立执行的调整计算现金余额。
// 确认但尚未执行的增提不能扩张可扣回金额。
func commissionCashExposure(parent *ent.FinanceCommission, rows []*ent.FinanceCommissionAdjustment) (decimal.Decimal, error) {
	if parent.Status != commission.StatusPAID || parent.PaidAmount == nil || parent.PaymentSnapshot == nil {
		return decimal.Zero, biz.ErrCommissionPaymentEvidence
	}
	cash, err := decimalOf(*parent.PaidAmount)
	if err != nil || cash.IsNegative() {
		return decimal.Zero, biz.ErrCommissionPaymentEvidence
	}
	var snapshot biz.CommissionPaymentPreview
	if json.Unmarshal([]byte(*parent.PaymentSnapshot), &snapshot) != nil || snapshot.CommissionID != parent.ID || snapshot.BaseCurrency != parent.BaseCurrency || !snapshot.ActualAmount.Equal(cash) {
		return decimal.Zero, biz.ErrCommissionPaymentEvidence
	}
	for _, row := range rows {
		if row.Status != adjustment.StatusPAID {
			continue
		}
		if row.ExecutionMethod == nil {
			return decimal.Zero, biz.ErrCommissionPaymentEvidence
		}
		if *row.ExecutionMethod == adjustment.ExecutionMethodOFFSET_WITH_PARENT {
			if row.ParentPaymentID == nil || *row.ParentPaymentID != parent.ID {
				return decimal.Zero, biz.ErrCommissionPaymentEvidence
			}
			continue
		}
		if *row.ExecutionMethod != adjustment.ExecutionMethodSEPARATE || row.ParentPaymentID != nil || row.BaseCurrency != parent.BaseCurrency {
			return decimal.Zero, biz.ErrCommissionPaymentEvidence
		}
		amount, err := decimalOf(row.Amount)
		if err != nil || !amount.IsPositive() {
			return decimal.Zero, biz.ErrCommissionPaymentEvidence
		}
		if row.Direction == adjustment.DirectionDECREASE {
			cash = cash.Sub(amount)
		} else {
			cash = cash.Add(amount)
		}
	}
	if cash.IsNegative() {
		return decimal.Zero, biz.ErrCommissionPaymentEvidence
	}
	return cash, nil
}

func populateAdjustmentExecutionCapability(result *biz.FinanceCommissionAdjustment, row *ent.FinanceCommissionAdjustment, parent *ent.FinanceCommission, rows []*ent.FinanceCommissionAdjustment) {
	if row.Status != adjustment.StatusCONFIRMED || row.ExecutionMethod != nil {
		return
	}
	blocked := func(reason string) { result.CanExecute = false; result.ExecutionBlockedReason = &reason }
	if parent == nil || parent.Status != commission.StatusPAID {
		blocked("原提成尚未发放，调整将随原提成按净额一并结清")
		return
	}
	cash, err := commissionCashExposure(parent, rows)
	if err != nil {
		blocked("原提成缺少有效实际付款证据，不能执行现金调整")
		return
	}
	if row.Direction == adjustment.DirectionDECREASE && cash.LessThan(result.Amount) {
		blocked("实际已支付余额不足，请先完成相关增提发放")
		return
	}
	for _, l := range parent.Edges.Lines {
		if l.Edges.Order != nil && (len(l.Edges.Order.Edges.FeeSupplementRequests) > 0 || len(l.Edges.Order.Edges.FeeCorrectionRequests) > 0) {
			blocked("关联订单存在待审核补录或费用更正，请处理后再执行调整")
			return
		}
	}
	result.CanExecute = true
	result.ExecutionBlockedReason = nil
}

// ensureCommissionOrdersReady 调用方先锁定完整订单集合，待审补录与确认/发放在相同行串行。
func ensureCommissionOrdersReady(ctx context.Context, client *ent.Client, orderIDs []uuid.UUID) error {
	pending, err := client.OrderFeeSupplementRequest.Query().Where(supplement.OrderIDIn(orderIDs...), supplement.StatusEQ(supplement.StatusPENDING)).Exist(ctx)
	if err != nil {
		return err
	}
	if pending {
		return biz.ErrCommissionPendingSupplement
	}
	pending, err = client.OrderFeeCorrectionRequest.Query().Where(correction.OrderIDIn(orderIDs...), correction.StatusEQ(correction.StatusPENDING)).Exist(ctx)
	if err != nil {
		return err
	}
	if pending {
		return biz.ErrFeeCorrectionPending
	}

	return nil
}

func lockCommissionPayment(ctx context.Context, tx *ent.Tx, org, id uuid.UUID) (*ent.FinanceCommission, []*ent.FinanceCommissionAdjustment, error) {
	lines, err := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(id), commissionline.OrganizationIDEQ(org)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	ids := orderUUIDsFromLines(lines)
	if err := lockOrdersSortedForFinance(ctx, tx, ids); err != nil {
		return nil, nil, err
	}
	if err := ensureCommissionOrdersReady(ctx, tx.Client(), ids); err != nil {
		return nil, nil, err
	}
	parent, err := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrCommissionNotFound, nil)
	}
	rows, err := tx.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(id)).Order(adjustment.ByID()).ForUpdate().All(ctx)
	return parent, rows, err
}

func paymentPreviewFromRows(parent *ent.FinanceCommission, rows []*ent.FinanceCommissionAdjustment) (*biz.CommissionPaymentPreview, error) {
	p, err := commissionToBiz(parent)
	if err != nil {
		return nil, err
	}
	as := make([]*biz.FinanceCommissionAdjustment, 0, len(rows))
	for _, r := range rows {
		a, err := commissionAdjustmentToBiz(r)
		if err != nil {
			return nil, err
		}
		as = append(as, a)
	}
	return biz.CalculateCommissionPayment(p, as)
}

func (r *commissionRepo) PreviewPayment(ctx context.Context, org, id uuid.UUID, version uint64) (*biz.CommissionPaymentPreview, error) {
	var result *biz.CommissionPaymentPreview
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		parent, rows, err := lockCommissionPayment(ctx, tx, org, id)
		if err != nil {
			return err
		}
		if parent.Version != version {
			return biz.ErrCommissionTransition
		}
		result, err = paymentPreviewFromRows(parent, rows)
		return err
	})
	return result, err
}

func (r *commissionRepo) Pay(ctx context.Context, org, id, actor uuid.UUID, version uint64, in biz.CommissionPaymentInput, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		parent, rows, err := lockCommissionPayment(ctx, tx, org, id)
		if err != nil {
			return err
		}
		if parent.Status == commission.StatusPAID && parent.PaymentKey != nil && *parent.PaymentKey == in.IdempotencyKey {
			if parent.PaymentFingerprint == nil || *parent.PaymentFingerprint != in.PreviewToken || parent.PaidAmount == nil {
				return biz.ErrCommissionTransition
			}
			amount, err := decimalOf(*parent.PaidAmount)
			if err != nil {
				return err
			}
			if !amount.Equal(in.ActualAmount) {
				return biz.ErrCommissionTransition
			}
			return nil
		}
		if parent.Version != version {
			return biz.ErrCommissionTransition
		}
		preview, err := paymentPreviewFromRows(parent, rows)
		if err != nil {
			return err
		}
		if preview.PreviewToken != in.PreviewToken || !preview.ActualAmount.Equal(in.ActualAmount) {
			return biz.ErrCommissionTransition
		}
		encoded, err := json.Marshal(preview)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, a := range rows {
			if a.Status != adjustment.StatusCONFIRMED {
				continue
			}
			if _, err := tx.FinanceCommissionAdjustment.UpdateOne(a).SetStatus(adjustment.StatusPAID).SetPaidAt(now).SetPaidBy(actor).SetExecutionMethod(adjustment.ExecutionMethodOFFSET_WITH_PARENT).SetParentPaymentID(id).SetVersion(a.Version + 1).Save(ctx); err != nil {
				return err
			}
		}
		if _, err := tx.FinanceCommission.UpdateOne(parent).SetStatus(commission.StatusPAID).SetPaidAt(now).SetPaidBy(actor).SetPaidAmount(preview.ActualAmount.StringFixed(8)).SetPaymentKey(in.IdempotencyKey).SetPaymentFingerprint(preview.PreviewToken).SetPaymentSnapshot(string(encoded)).SetVersion(parent.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}
