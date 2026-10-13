package data

import (
	"context"
	"encoding/json"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	noticeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correctionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"strings"
	"time"
)

type orderFeeCorrectionRepo struct{ data *Data }

func NewOrderFeeCorrectionRepo(data *Data) biz.OrderFeeCorrectionRepo {
	return &orderFeeCorrectionRepo{data: data}
}

// Authorize 在Order锁之前稳定授权父行，再重读同一grant中的资格与组织范围。
func (r *orderFeeCorrectionRepo) Authorize(ctx context.Context, p *biz.Principal, org, order uuid.UUID, action string) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	o, err := client.Order.Query().Where(orderent.IDEQ(order), orderent.OrganizationIDEQ(org)).Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	_, transactional := transactionFromContext(ctx)
	if transactional {
		if err = lockFeeAuthorizationParents(ctx, client, p.UserID); err != nil {
			return err
		}
	}
	enabled, err := client.Membership.Query().Where(membershipent.UserIDEQ(p.UserID), membershipent.EnabledEQ(true), membershipent.HasUserWith(userent.EnabledEQ(true)), membershipent.HasOrganizationWith(organizationent.EnabledEQ(true))).Exist(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return biz.ErrFeeReviewForbidden
	}
	var allowed bool
	switch action {
	case "approve":
		allowed, err = hasFeeReviewPermission(ctx, client, org, p.UserID, false)
	case "submit:ADD", "submit:REPLACE", "submit:VOID":
		operation := access.OrderFeeUpdate
		if action == "submit:ADD" {
			operation = access.OrderFeeCreate
		}
		allowed, err = hasRealtimeFeePermission(ctx, client, org, p.UserID, access.OrderPermission(access.OrderBusinessType(o.BusinessType), operation), false)
	case "read":
		allowed, err = feeReviewReadable(ctx, client, p, o)
		if err == nil && !allowed {
			allowed, err = client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDEQ(order), correctionent.RequestedByEQ(p.UserID)).Exist(ctx)
		}
	case "withdraw":
		// 本人身份在锁定目标申请后校验，无须费用审核权限。
		allowed = true
	}
	if strings.HasPrefix(action, "read:") {
		allowed, err = feeReviewReadable(ctx, client, p, o)
		if err == nil && !allowed {
			id, parseErr := uuid.Parse(strings.TrimPrefix(action, "read:"))
			if parseErr != nil {
				return biz.ErrFeeCorrectionInvalid
			}
			allowed, err = client.OrderFeeCorrectionRequest.Query().Where(correctionent.IDEQ(id), correctionent.OrderIDEQ(order), correctionent.RequestedByEQ(p.UserID)).Exist(ctx)
		}
	}
	if err != nil {
		return err
	}
	if !allowed {
		return biz.ErrFeeReviewForbidden
	}
	return nil
}
func (r *orderFeeCorrectionRepo) LockOrder(ctx context.Context, org, order uuid.UUID) (*biz.OrderFeeSupplementLockEvidence, error) {
	return (&orderFeeSupplementRepo{data: r.data}).LockOrderForSupplement(ctx, org, order)
}
func (r *orderFeeCorrectionRepo) CurrentFees(ctx context.Context, order uuid.UUID) ([]*biz.OrderFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := client.OrderFee.Query().Where(feeent.OrderIDEQ(order), currentOrderFeePredicate()).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderFee, 0, len(rows))
	for _, row := range rows {
		f, err := orderFeeToBiz(row)
		if err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, nil
}
func correctionRequestToBiz(row *ent.OrderFeeCorrectionRequest) (*biz.OrderFeeCorrectionRequest, error) {
	r := &biz.OrderFeeCorrectionRequest{ID: row.ID, OrganizationID: row.OrganizationID, OrderID: row.OrderID, Operation: string(row.Operation), SourceFeeID: row.SourceFeeID, ApprovedFeeReviewEventID: row.ApprovedFeeReviewEventID, ApprovedFeeContentVersion: row.ApprovedFeeContentVersion, ApprovedFeeReviewVersion: row.ApprovedFeeReviewVersion, Reason: row.Reason, IdempotencyKey: row.IdempotencyKey, RequestFingerprint: row.RequestFingerprint, RequestedBy: row.RequestedBy, RequestedAt: row.RequestedAt, Status: string(row.Status), Version: row.Version, DecidedBy: row.DecidedBy, DecidedAt: row.DecidedAt, DecisionReason: row.DecisionReason, ApprovalToken: row.ApprovalToken}
	if len(row.ApprovalResultSnapshot) > 0 {
		if err := json.Unmarshal(row.ApprovalResultSnapshot, &r.ApprovalResult); err != nil {
			return nil, err
		}
	}
	if len(row.BeforeSnapshot) > 0 {
		if err := json.Unmarshal(row.BeforeSnapshot, &r.Before); err != nil {
			return nil, err
		}
	}
	if len(row.AfterSnapshot) > 0 {
		if err := json.Unmarshal(row.AfterSnapshot, &r.After); err != nil {
			return nil, err
		}
	}
	if row.Edges.RequestedByUser != nil {
		r.RequestedByName = row.Edges.RequestedByUser.DisplayName
	}
	if row.Edges.DecidedByUser != nil {
		r.DecidedByName = row.Edges.DecidedByUser.DisplayName
	}
	if row.Status == correctionent.StatusAPPROVED {
		r.FinancialStatus = "FINANCE_FOLLOW_UP"
		for _, fee := range row.Edges.GeneratedFees {
			if fee.EconomicEffect == feeent.EconomicEffectDECREASE {
				r.ReversalFeeID = &fee.ID
			}
			if fee.AccountingRequired && fee.SupersededByRequestID == nil && len(fee.Edges.FinanceBillLines) == 0 {
				r.FinancialStatus = "ADJUSTMENT_UNBILLED"
			}
		}
	}
	return r, nil
}
func (r *orderFeeCorrectionRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.OrderFeeCorrectionRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	row, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.IDEQ(id), correctionent.OrganizationIDEQ(org)).WithRequestedByUser().WithDecidedByUser().WithGeneratedFees(func(q *ent.OrderFeeQuery) { q.WithFinanceBillLines(effectiveBillLineFilter) }).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeCorrectionNotFound, nil)
	}
	value, err := correctionRequestToBiz(row)
	if err != nil {
		return nil, err
	}
	if err = r.decorateFinancialStatus(ctx, client, row, value); err != nil {
		return nil, err
	}
	if p, principalErr := biz.RequirePrincipal(ctx); principalErr == nil {
		grant, err := hasFeeReviewPermission(ctx, client, org, p.UserID, false)
		if err != nil {
			return nil, err
		}
		value.CanApprove = grant && value.Status == "PENDING"
		value.CanWithdraw = value.RequestedBy == p.UserID && value.Status == "PENDING"
	}
	return value, nil
}
func (r *orderFeeCorrectionRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.OrderFeeCorrectionRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	row, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrganizationIDEQ(org), correctionent.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return correctionRequestToBiz(row)
}
func (r *orderFeeCorrectionRepo) List(ctx context.Context, org, order uuid.UUID, page, size int) ([]*biz.OrderFeeCorrectionRequest, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, 0, err
	}
	o, err := client.Order.Get(ctx, order)
	if err != nil {
		return nil, 0, err
	}
	readable, err := feeReviewReadable(ctx, client, p, o)
	if err != nil {
		return nil, 0, err
	}
	grant, err := hasFeeReviewPermission(ctx, client, org, p.UserID, false)
	if err != nil {
		return nil, 0, err
	}
	q := client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrganizationIDEQ(org), correctionent.OrderIDEQ(order))
	if !readable {
		q.Where(correctionent.RequestedByEQ(p.UserID))
	}
	count, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.WithRequestedByUser().WithDecidedByUser().WithGeneratedFees(func(q *ent.OrderFeeQuery) { q.WithFinanceBillLines(effectiveBillLineFilter) }).Order(correctionent.ByRequestedAt(sql.OrderDesc()), correctionent.ByID(sql.OrderDesc())).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.OrderFeeCorrectionRequest, 0, len(rows))
	for _, row := range rows {
		value, err := correctionRequestToBiz(row)
		if err != nil {
			return nil, 0, err
		}
		if err = r.decorateFinancialStatus(ctx, client, row, value); err != nil {
			return nil, 0, err
		}
		value.CanApprove = grant && value.Status == "PENDING"
		value.CanWithdraw = value.RequestedBy == p.UserID && value.Status == "PENDING"
		result = append(result, value)
	}
	return result, count, nil
}
func (r *orderFeeCorrectionRepo) LockRequest(ctx context.Context, org, id uuid.UUID) (*biz.OrderFeeCorrectionRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	ref, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.IDEQ(id), correctionent.OrganizationIDEQ(org)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeCorrectionNotFound, nil)
	}
	if _, err = client.Order.Query().Where(orderent.IDEQ(ref.OrderID), orderent.OrganizationIDEQ(org)).ForUpdate().Only(ctx); err != nil {
		return nil, err
	}
	row, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.IDEQ(id), correctionent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeCorrectionNotFound, nil)
	}
	return correctionRequestToBiz(row)
}
func (r *orderFeeCorrectionRepo) Create(ctx context.Context, value *biz.OrderFeeCorrectionRequest, audit *biz.AuditEvent) (*biz.OrderFeeCorrectionRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	pending, err := hasPendingFeeSupplement(ctx, client, value.OrderID)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, biz.ErrFeeCorrectionPending
	}
	candidates, err := qualifiedFeeReviewCandidates(ctx, client, value.OrganizationID)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, biz.ErrFeeReviewNoApprover
	}
	builder := client.OrderFeeCorrectionRequest.Create().SetID(value.ID).SetOrganizationID(value.OrganizationID).SetOrderID(value.OrderID).SetOperation(correctionent.Operation(value.Operation)).SetNillableSourceFeeID(value.SourceFeeID).SetApprovedFeeReviewEventID(value.ApprovedFeeReviewEventID).SetApprovedFeeContentVersion(value.ApprovedFeeContentVersion).SetApprovedFeeReviewVersion(value.ApprovedFeeReviewVersion).SetReason(value.Reason).SetIdempotencyKey(value.IdempotencyKey).SetRequestFingerprint(value.RequestFingerprint).SetRequestedBy(value.RequestedBy).SetRequestedAt(value.RequestedAt)
	if value.Before != nil {
		encoded, err := json.Marshal(value.Before)
		if err != nil {
			return nil, err
		}
		builder.SetBeforeSnapshot(encoded).SetSourceFeeVersion(value.Before.Version)
	}
	if value.After != nil {
		encoded, err := json.Marshal(value.After)
		if err != nil {
			return nil, err
		}
		builder.SetAfterSnapshot(encoded)
	}
	row, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, biz.ErrFeeCorrectionConflict
		}
		return nil, err
	}
	if err = bumpFeeReviewVersion(ctx, client, value.OrderID); err != nil {
		return nil, err
	}
	if err = writeAudit(ctx, client.AuditLog, audit); err != nil {
		return nil, err
	}
	o, err := client.Order.Get(ctx, value.OrderID)
	if err != nil {
		return nil, err
	}
	// 复用整票审核提醒渠道，不新增通知来源类型或第二套发消息实现。
	for _, candidate := range candidates {
		if err = enqueueFeeReviewNoticeForUser(ctx, client, o, row.ID, candidate.UserID, value.RequestedByName, "费用更正待审核", noticeent.TemplateFEE_REVIEW_PENDING, "fee-correction-pending:"); err != nil {
			return nil, err
		}
	}
	return correctionRequestToBiz(row)
}
func (r *orderFeeCorrectionRepo) SaveDecision(ctx context.Context, value *biz.OrderFeeCorrectionRequest, actor uuid.UUID, status, reason, token string) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	updated, err := client.OrderFeeCorrectionRequest.Update().Where(correctionent.IDEQ(value.ID), correctionent.StatusEQ(correctionent.StatusPENDING), correctionent.VersionEQ(value.Version)).SetStatus(correctionent.Status(status)).SetVersion(value.Version + 1).SetDecidedBy(actor).SetDecidedAt(time.Now().UTC()).SetDecisionReason(reason).SetApprovalToken(token).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return biz.ErrFeeCorrectionConflict
	}
	if status != "APPROVED" {
		if err = bumpFeeReviewVersion(ctx, client, value.OrderID); err != nil {
			return err
		}
	}
	o, err := client.Order.Get(ctx, value.OrderID)
	if err != nil {
		return err
	}
	user, err := client.User.Get(ctx, actor)
	if err != nil {
		return err
	}
	if err = enqueueFeeReviewResultForUser(ctx, client, o, value.ID, value.RequestedBy, user.DisplayName, "费用更正已处理"); err != nil {
		return err
	}
	return writeAudit(ctx, client.AuditLog, &biz.AuditEvent{OrganizationID: &value.OrganizationID, UserID: &actor, Action: "order.fee_correction." + status, Result: "success", ResourceType: "order_fee_correction_request", ResourceID: value.ID.String(), Details: map[string]string{"order.id": value.OrderID.String(), "reason": reason}})
}

var _ biz.OrderFeeCorrectionRepo = (*orderFeeCorrectionRepo)(nil)

// ValidateTarget 在预览与审批事务中复核快照引用，停用的费项/单位/币种/结算对象不能进入新正式费用。
func (r *orderFeeCorrectionRepo) ValidateTarget(ctx context.Context, org, orderID uuid.UUID, snapshot *biz.OrderFeeSupplementFeeSnapshot) error {
	_, err := (&orderFeeSupplementRepo{data: r.data}).ResolveFeeFactsForApproval(ctx, org, orderID, snapshot)
	return err
}
