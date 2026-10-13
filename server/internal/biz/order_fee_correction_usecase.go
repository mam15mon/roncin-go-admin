package biz

import (
	"context"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"
)

// OrderFeeCorrectionUsecase 编排受控更正；费用、审批快照与提成影响同事务提交。
type OrderFeeCorrectionUsecase struct {
	repo       OrderFeeCorrectionRepo
	fee        *OrderFeeUsecase
	commission FeeCorrectionCommissionRepo
	transactor Transactor
}

func NewOrderFeeCorrectionUsecase(repo OrderFeeCorrectionRepo, fee *OrderFeeUsecase, commission FeeCorrectionCommissionRepo, tx Transactor) *OrderFeeCorrectionUsecase {
	return &OrderFeeCorrectionUsecase{repo: repo, fee: fee, commission: commission, transactor: tx}
}
func (uc *OrderFeeCorrectionUsecase) Submit(ctx context.Context, p *Principal, org, order uuid.UUID, in *OrderFeeCorrectionInput, override bool) (*OrderFeeCorrectionRequest, error) {
	if !validSupplementCaller(p, org, order) || in == nil {
		return nil, ErrFeeCorrectionInvalid
	}
	in.Reason = strings.TrimSpace(in.Reason)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.Reason == "" || utf8.RuneCountInString(in.Reason) > 1000 || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 {
		return nil, ErrFeeCorrectionInvalid
	}
	if (in.Operation == "ADD" && (in.SourceFeeID != nil || in.After == nil || in.ExpectedFeeVersion != 0)) || (in.Operation == "REPLACE" && (in.SourceFeeID == nil || in.After == nil)) || (in.Operation == "VOID" && (in.SourceFeeID == nil || in.After != nil)) || (in.Operation != "ADD" && in.Operation != "REPLACE" && in.Operation != "VOID") {
		return nil, ErrFeeCorrectionInvalid
	}
	fingerprint := feeCorrectionHash(struct {
		OrderID uuid.UUID
		Input   *OrderFeeCorrectionInput
	}{order, in})
	var result *OrderFeeCorrectionRequest
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Authorize(txCtx, p, org, order, "submit:"+in.Operation); err != nil {
			return err
		}
		evidence, err := uc.repo.LockOrder(txCtx, org, order)
		if err != nil {
			return err
		}
		existing, err := uc.repo.GetByKey(txCtx, org, in.IdempotencyKey)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.RequestFingerprint != fingerprint {
				return ErrOrderFeeIdempotencyConflict
			}
			result = existing
			return nil
		}
		current, err := uc.repo.CurrentFees(txCtx, order)
		if err != nil {
			return err
		}
		r := &OrderFeeCorrectionRequest{ID: uuid.Must(uuid.NewV7()), OrganizationID: org, OrderID: order, Operation: in.Operation, SourceFeeID: in.SourceFeeID, ApprovedFeeReviewEventID: *evidence.ApprovedFeeReviewEventID, ApprovedFeeContentVersion: evidence.FeeContentVersion, ApprovedFeeReviewVersion: evidence.FeeReviewVersion, Reason: in.Reason, IdempotencyKey: in.IdempotencyKey, RequestFingerprint: fingerprint, RequestedBy: p.UserID, RequestedAt: time.Now().UTC(), Status: "PENDING", Version: 1}
		if in.SourceFeeID != nil {
			for _, f := range current {
				if f.ID == *in.SourceFeeID {
					r.Before = f
					break
				}
			}
			if r.Before == nil {
				return ErrFeeCorrectionSource
			}
			if in.ExpectedFeeVersion == 0 || r.Before.Version != in.ExpectedFeeVersion {
				return ErrFeeCorrectionConflict
			}
		}
		if in.After != nil {
			shell := *in.After
			shell.ID = uuid.Nil
			shell.IdempotencyKey = in.IdempotencyKey
			shell.OrderID = order
			shell.EconomicEffect = OrderFeeEconomicIncrease
			shell.AccountingRequired = true
			shell.CorrectionRequestID = nil
			shell.SupersededByRequestID = nil
			shell.SourceFeeID = nil
			shell.SupplementRequestID = nil
			after, err := normalizeOrderFee(&shell)
			if err != nil {
				return err
			}
			if err = uc.fee.resolveCatalog(txCtx, org, order, after, true, false); err != nil {
				return err
			}
			if after.FeeNameOverride != nil {
				after.FeeName = *after.FeeNameOverride
			}
			if after.TaxRateOverride != nil {
				after.TaxRate = after.TaxRateOverride
			}
			if err = uc.fee.resolveExchangeRate(txCtx, org, order, after, override); err != nil {
				return err
			}
			if err = uc.fee.calculateAmounts(txCtx, org, after); err != nil {
				return err
			}
			if err = uc.fee.ensureReceivablePartySelectionAllowed(txCtx, org, after); err != nil {
				return err
			}

			after.ID = uuid.Must(uuid.NewV7())
			after.SourceFeeID = r.SourceFeeID
			after.CorrectionRequestID = &r.ID
			r.After = after
		}
		result, err = uc.repo.Create(txCtx, r, correctionAudit(r, p.UserID, "submit"))
		return err
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(WithPrincipal(ctx, p), org, result.ID)
}
func correctionAudit(r *OrderFeeCorrectionRequest, actor uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &r.OrganizationID, UserID: &actor, Action: "order.fee_correction." + action, Result: "success", ResourceType: "order_fee_correction_request", ResourceID: r.ID.String(), Details: map[string]string{"order.id": r.OrderID.String(), "request.id": r.ID.String(), "operation": r.Operation, "reason": r.Reason}}
}
func projectedCorrectionFees(current []*OrderFee, r *OrderFeeCorrectionRequest) []*OrderFee {
	projected := make([]*OrderFee, 0, len(current)+1)
	for _, f := range current {
		if r.SourceFeeID == nil || f.ID != *r.SourceFeeID {
			projected = append(projected, f)
		}
	}
	if r.After != nil {
		projected = append(projected, r.After)
	}
	return projected
}
func (uc *OrderFeeCorrectionUsecase) previewLocked(ctx context.Context, p *Principal, org, order, id uuid.UUID, version uint64) (*OrderFeeCorrectionPreview, error) {
	if err := uc.repo.Authorize(ctx, p, org, order, "approve"); err != nil {
		return nil, err
	}
	evidence, err := uc.repo.LockOrder(ctx, org, order)
	if err != nil {
		return nil, err
	}
	r, err := uc.repo.LockRequest(ctx, org, id)
	if err != nil {
		return nil, err
	}
	if r.OrderID != order {
		return nil, ErrFeeCorrectionNotFound
	}
	if r.Status != "PENDING" || r.Version != version {
		return nil, ErrFeeCorrectionConflict
	}
	if evidence.ApprovedFeeReviewEventID == nil || r.ApprovedFeeReviewEventID != *evidence.ApprovedFeeReviewEventID || r.ApprovedFeeContentVersion != evidence.FeeContentVersion {
		return nil, ErrFeeCorrectionConflict
	}
	current, err := uc.repo.CurrentFees(ctx, order)
	if err != nil {
		return nil, err
	}
	if r.Before != nil {
		found := false
		for _, f := range current {
			if f.ID == r.Before.ID {
				found = true
				if f.Version != r.Before.Version || feeCorrectionHash(feeSnapshotFromOrderFee(f)) != feeCorrectionHash(feeSnapshotFromOrderFee(r.Before)) {
					return nil, ErrFeeCorrectionConflict
				}
			}
		}
		if !found {
			return nil, ErrFeeCorrectionSource
		}
	}
	if r.After != nil {
		snapshot := feeSnapshotFromOrderFee(r.After)
		if err := uc.repo.ValidateTarget(ctx, org, order, &snapshot); err != nil {
			return nil, err
		}
		if err := uc.fee.ensureReceivablePartySelectionAllowed(ctx, org, r.After); err != nil {
			return nil, err
		}
	}
	projected := projectedCorrectionFees(current, r)
	before, _, err := BuildFeeReviewSnapshot(current)
	if err != nil {
		return nil, err
	}
	after, _, err := BuildFeeReviewSnapshot(projected)
	if err != nil {
		return nil, err
	}
	financialFingerprint, err := uc.repo.FinancialFingerprint(ctx, r)
	if err != nil {
		return nil, err
	}
	impact, err := uc.commission.PreviewCorrectionImpact(ctx, org, order, r, current, projected)
	if err != nil {
		return nil, err
	}
	preview := &OrderFeeCorrectionPreview{Request: r, Before: before, After: after, CommissionImpact: impact, FinancialFingerprint: financialFingerprint}
	preview.PreviewToken = feeCorrectionHash(struct {
		Request              *OrderFeeCorrectionRequest
		Evidence             *OrderFeeSupplementLockEvidence
		Before, After        *FeeReviewSnapshot
		Impact               *FeeCorrectionCommissionImpact
		FinancialFingerprint string
	}{r, evidence, before, after, impact, financialFingerprint})
	return preview, nil
}
func (uc *OrderFeeCorrectionUsecase) Preview(ctx context.Context, p *Principal, org, order, id uuid.UUID, version uint64) (*OrderFeeCorrectionPreview, error) {
	if !validSupplementCaller(p, org, order) || id == uuid.Nil || version == 0 {
		return nil, ErrFeeCorrectionInvalid
	}
	var result *OrderFeeCorrectionPreview
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		result, err = uc.previewLocked(txCtx, p, org, order, id, version)
		return err
	})
	return result, err
}
func (uc *OrderFeeCorrectionUsecase) Approve(ctx context.Context, p *Principal, org, order, id uuid.UUID, version uint64, token, comment string) (*OrderFeeCorrectionRequest, error) {
	if !validSupplementCaller(p, org, order) || id == uuid.Nil || version == 0 || strings.TrimSpace(token) == "" || utf8.RuneCountInString(comment) > 1000 {
		return nil, ErrFeeCorrectionInvalid
	}
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Authorize(txCtx, p, org, order, "approve"); err != nil {
			return err
		}
		// 成功批准同版本同令牌重放不重复写费用及调整。
		previous, err := uc.repo.Get(txCtx, org, id)
		if err != nil {
			return err
		}
		if previous.OrderID != order {
			return ErrFeeCorrectionNotFound
		}
		if previous.Status == "APPROVED" {
			if previous.Version == version+1 && previous.ApprovalToken != nil && *previous.ApprovalToken == token {
				return nil
			}
			return ErrFeeCorrectionConflict
		}
		preview, err := uc.previewLocked(txCtx, p, org, order, id, version)
		if err != nil {
			return err
		}
		if preview.PreviewToken != token {
			return ErrFeeCorrectionConflict
		}
		if preview.After.Summary.Profit.IsNegative() && strings.TrimSpace(comment) == "" {
			return ErrFeeCorrectionInvalid
		}
		preview.Request.ApprovalResult = &FeeCorrectionApprovalResult{Current: preview.Before.Summary, Projected: preview.After.Summary, CommissionImpact: preview.CommissionImpact, FinancialFingerprint: preview.FinancialFingerprint}
		if err = uc.repo.Apply(txCtx, preview.Request, p.UserID, comment); err != nil {
			return err
		}
		if err = uc.commission.ReconcileCorrectionImpact(txCtx, org, order, p.UserID, preview.Request, preview.CommissionImpact); err != nil {
			return err
		}
		return uc.repo.SaveDecision(txCtx, preview.Request, p.UserID, "APPROVED", comment, token)
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(WithPrincipal(ctx, p), org, id)
}
func (uc *OrderFeeCorrectionUsecase) Decide(ctx context.Context, p *Principal, org, order, id uuid.UUID, version uint64, status, reason string) (*OrderFeeCorrectionRequest, error) {
	reason = strings.TrimSpace(reason)
	if !validSupplementCaller(p, org, order) || id == uuid.Nil || version == 0 || (status != "WITHDRAWN" && status != "REJECTED") || utf8.RuneCountInString(reason) > 1000 || (status == "REJECTED" && reason == "") {
		return nil, ErrFeeCorrectionInvalid
	}
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		action := "withdraw"
		if status == "REJECTED" {
			action = "approve"
		}
		if err := uc.repo.Authorize(txCtx, p, org, order, action); err != nil {
			return err
		}
		// 撤回/退回只需锁订单，不要求批准快照仍有效。
		r, err := uc.repo.LockRequest(txCtx, org, id)
		if err != nil {
			return err
		}
		if r.OrderID != order {
			return ErrFeeCorrectionNotFound
		}
		if r.Version != version || r.Status != "PENDING" {
			return ErrFeeCorrectionConflict
		}
		if status == "WITHDRAWN" && r.RequestedBy != p.UserID {
			return ErrPermissionDenied
		}
		return uc.repo.SaveDecision(txCtx, r, p.UserID, status, reason, "")
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(WithPrincipal(ctx, p), org, id)
}
func (uc *OrderFeeCorrectionUsecase) List(ctx context.Context, p *Principal, org, order uuid.UUID, page, size int) ([]*OrderFeeCorrectionRequest, int, error) {
	if !validSupplementCaller(p, org, order) || !ValidListPagination(page, size) {
		return nil, 0, ErrFeeCorrectionInvalid
	}
	if err := uc.repo.Authorize(ctx, p, org, order, "read"); err != nil {
		return nil, 0, err
	}
	return uc.repo.List(WithPrincipal(ctx, p), org, order, page, size)
}

func (uc *OrderFeeCorrectionUsecase) Get(ctx context.Context, p *Principal, org, order, id uuid.UUID) (*OrderFeeCorrectionRequest, error) {
	if !validSupplementCaller(p, org, order) || id == uuid.Nil {
		return nil, ErrFeeCorrectionInvalid
	}
	if err := uc.repo.Authorize(ctx, p, org, order, "read:"+id.String()); err != nil {
		return nil, err
	}
	r, err := uc.repo.Get(WithPrincipal(ctx, p), org, id)
	if err != nil {
		return nil, err
	}
	if r.OrderID != order {
		return nil, ErrFeeCorrectionNotFound
	}
	return r, nil
}
