package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	taskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	noticeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correctionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
	supplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	assignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type orderFeeReviewRepo struct{ data *Data }

func NewOrderFeeReviewRepo(data *Data) biz.OrderFeeReviewRepo { return &orderFeeReviewRepo{data: data} }

// feeReviewGrants 候选与动作共用同一 grant 谓词，权限不能与另一个角色的范围拼接。
func feeReviewGrants(ctx context.Context, client *ent.Client, organizationID uuid.UUID) ([]*ent.RoleAssignment, error) {
	enabled, err := client.Organization.Query().Where(organizationent.IDEQ(organizationID), organizationent.EnabledEQ(true)).Exist(ctx)
	if err != nil || !enabled {
		return nil, err
	}
	ancestors, err := organizationLockGrantAncestorIDs(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	return client.RoleAssignment.Query().Where(qualifiedBusinessLockGrantPredicate(access.FinanceFeeReview, organizationID, ancestors, membershipent.HasUserWith(userent.EnabledEQ(true)), membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)))).WithRole().WithMembership(func(q *ent.MembershipQuery) { q.WithUser() }).Order(assignmentent.ByID()).All(ctx)
}
func hasFeeReviewPermission(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, lock bool) (bool, error) {
	return hasRealtimeFeePermission(ctx, client, organizationID, userID, access.FinanceFeeReview, lock)
}
func hasRealtimeFeePermission(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, key string, lock bool) (bool, error) {
	enabled, err := client.Organization.Query().Where(organizationent.IDEQ(organizationID), organizationent.EnabledEQ(true)).Exist(ctx)
	if err != nil || !enabled {
		return false, err
	}
	if lock {
		if err := lockFeeAuthorizationParents(ctx, client, userID); err != nil {
			return false, err
		}
	}
	ancestors, err := organizationLockGrantAncestorIDs(ctx, client, organizationID)
	if err != nil {
		return false, err
	}
	return client.RoleAssignment.Query().Where(qualifiedBusinessLockGrantPredicate(key, organizationID, ancestors, membershipent.UserIDEQ(userID), membershipent.HasUserWith(userent.EnabledEQ(true)), membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)))).Exist(ctx)
}

// lockFeeAuthorizationParents 必须在订单行锁之前调用，与月度申请和账号维护
// 共用 User → Membership → Role → Order 锁序。持有父锁后重读同一授权关联，
// 防止成员、角色或权限在费用决策提交前失效。
func lockFeeAuthorizationParents(ctx context.Context, client *ent.Client, userID uuid.UUID) error {
	if _, err := client.User.Query().Where(userent.IDEQ(userID)).ForShare().Only(ctx); err != nil {
		return mapEntError(err, biz.ErrFeeReviewForbidden, nil)
	}
	if _, err := client.Membership.Query().Where(membershipent.UserIDEQ(userID)).Order(membershipent.ByID()).ForShare().All(ctx); err != nil {
		return err
	}
	_, err := client.Role.Query().Where(roleent.HasAssignmentsWith(assignmentent.HasMembershipWith(membershipent.UserIDEQ(userID)))).Order(roleent.ByID()).ForShare().All(ctx)
	return err
}

func feeReviewReadable(ctx context.Context, client *ent.Client, p *biz.Principal, o *ent.Order) (bool, error) {
	for _, key := range []string{access.FinanceFeeRead, access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeRead), access.FinanceFeeReview} {
		yes, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, key, false)
		if err != nil {
			return false, err
		}
		if yes {
			return true, nil
		}
	}
	return false, nil
}
func feeReviewSubmitter(ctx context.Context, client *ent.Client, p *biz.Principal, o *ent.Order, lock bool) (bool, error) {
	for _, operation := range []access.OrderOperation{access.OrderFeeCreate, access.OrderFeeUpdate} {
		key := access.OrderPermission(access.OrderBusinessType(o.BusinessType), operation)
		yes, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, key, lock)
		if err != nil {
			return false, err
		}
		if yes {
			return true, nil
		}
	}
	return false, nil
}
func loadFeeReviewSnapshot(ctx context.Context, client *ent.Client, orderID uuid.UUID) (*biz.FeeReviewSnapshot, string, error) {
	items, err := client.OrderFee.Query().Where(feeent.OrderIDEQ(orderID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if err != nil {
		return nil, "", err
	}
	fees := make([]*biz.OrderFee, 0, len(items))
	for _, item := range items {
		fee, err := orderFeeToBiz(item)
		if err != nil {
			return nil, "", err
		}
		fees = append(fees, fee)
	}
	return biz.BuildFeeReviewSnapshot(fees)
}
func hasPendingFeeSupplement(ctx context.Context, client *ent.Client, orderID uuid.UUID) (bool, error) {
	pending, err := client.OrderFeeSupplementRequest.Query().Where(supplementent.OrderIDEQ(orderID), supplementent.StatusEQ(supplementent.StatusPENDING)).Exist(ctx)
	if err != nil || pending {
		return pending, err
	}
	return client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDEQ(orderID), correctionent.StatusEQ(correctionent.StatusPENDING)).Exist(ctx)
}
func validateOrderApprovalSnapshot(ctx context.Context, client *ent.Client, o *ent.Order) error {
	if o.FeeReviewStatus != orderent.FeeReviewStatusAPPROVED || o.ApprovedFeeReviewEventID == nil {
		return biz.ErrFeeReviewNotApproved
	}
	event, err := client.OrderFeeReviewEvent.Get(ctx, *o.ApprovedFeeReviewEventID)
	if err != nil {
		return mapEntError(err, biz.ErrFeeReviewNotApproved, nil)
	}
	if event.OrderID != o.ID || event.OrganizationID != o.OrganizationID || event.ContentVersion != o.FeeContentVersion || (event.Action != eventent.ActionAPPROVE && event.Action != eventent.ActionSUPPLEMENT_APPROVE && event.Action != eventent.ActionCORRECTION_APPROVE) {
		return biz.ErrFeeReviewNotApproved
	}
	_, fingerprint, err := loadFeeReviewSnapshot(ctx, client, o.ID)
	if err != nil {
		return err
	}
	if fingerprint != event.ContentFingerprint {
		return biz.ErrFeeReviewNotApproved
	}
	return nil
}
func validateApprovedOrderFees(ctx context.Context, client *ent.Client, o *ent.Order) error {
	if err := validateOrderApprovalSnapshot(ctx, client, o); err != nil {
		return err
	}
	pending, err := hasPendingFeeSupplement(ctx, client, o.ID)
	if err != nil {
		return err
	}
	if pending {
		return biz.ErrFeeReviewPendingSupplement
	}
	return nil
}
func feeReviewPreviewToken(o *ent.Order, fingerprint string) string {
	return biz.FeeReviewHash([]byte(fmt.Sprintf("fee-review-v1|%s|%d|%d|%v|%s", o.ID, o.FeeContentVersion, o.FeeReviewVersion, o.CurrentFeeReviewEventID, fingerprint)))
}
func (r *orderFeeReviewRepo) Get(ctx context.Context, p *biz.Principal, id uuid.UUID) (*biz.OrderFeeReview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	o, err := client.Order.Get(ctx, id)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	readable, err := feeReviewReadable(ctx, client, p, o)
	if err != nil {
		return nil, err
	}
	if !readable {
		return nil, biz.ErrFeeReviewForbidden
	}
	return buildOrderFeeReview(ctx, client, p, o, true)
}
func buildOrderFeeReview(ctx context.Context, client *ent.Client, p *biz.Principal, o *ent.Order, history bool) (*biz.OrderFeeReview, error) {
	snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, client, o.ID)
	if err != nil {
		return nil, err
	}
	fullFees, loadErr := client.OrderFee.Query().Where(feeent.OrderIDEQ(o.ID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if loadErr != nil {
		return nil, loadErr
	}
	displayFees := make([]*biz.OrderFee, 0, len(fullFees))
	for _, f := range fullFees {
		fee, convertErr := orderFeeToBiz(f)
		if convertErr != nil {
			return nil, convertErr
		}
		if biz.IsCurrentFee(fee) {
			displayFees = append(displayFees, fee)
		}
	}
	result := &biz.OrderFeeReview{OrderID: o.ID, OrderNo: o.OrderNo, Status: string(o.FeeReviewStatus), ContentVersion: o.FeeContentVersion, ReviewVersion: o.FeeReviewVersion, ApprovedEventID: o.ApprovedFeeReviewEventID, Summary: &snapshot.Summary, Fees: displayFees, PreviewToken: feeReviewPreviewToken(o, fingerprint)}
	pending, err := hasPendingFeeSupplement(ctx, client, o.ID)
	if err != nil {
		return nil, err
	}
	result.HasPendingChange = pending
	result.HasPendingCorrection, err = client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDEQ(o.ID), correctionent.StatusEQ(correctionent.StatusPENDING)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	result.HasPendingSupplement = pending && !result.HasPendingCorrection
	qualified, err := hasFeeReviewPermission(ctx, client, o.OrganizationID, p.UserID, false)
	if err != nil {
		return nil, err
	}
	canSubmit, err := feeReviewSubmitter(ctx, client, p, o, false)
	if err != nil {
		return nil, err
	}
	editableStatus := o.FeeReviewStatus != orderent.FeeReviewStatusPENDING && o.FeeReviewStatus != orderent.FeeReviewStatusAPPROVED
	result.CanSubmit = canSubmit && editableStatus && !pending
	result.CanApprove = qualified && o.FeeReviewStatus == orderent.FeeReviewStatusPENDING
	if o.CurrentFeeReviewEventID != nil {
		current, err := client.OrderFeeReviewEvent.Get(ctx, *o.CurrentFeeReviewEventID)
		if err != nil {
			return nil, err
		}
		if current.Action == eventent.ActionSUBMIT {
			result.SubmissionID = &current.ID
			result.CanWithdraw = current.ActorID == p.UserID && o.FeeReviewStatus == orderent.FeeReviewStatusPENDING
			result.Summary.SubmittedAt = &current.CreatedAt
			result.Summary.SubmittedByName = current.ActorName
		}
	}
	occupied, err := client.FinanceBillLine.Query().Where(lineent.OrderIDEQ(o.ID), effectiveBillLinePredicate()).Exist(ctx)
	if err != nil {
		return nil, err
	}
	result.CanRevoke = qualified && o.FeeReviewStatus == orderent.FeeReviewStatusAPPROVED && !pending && !occupied
	locked, err := client.Order.Query().Where(orderent.IDEQ(o.ID), financeLockedOrderPredicate()).Exist(ctx)
	if err != nil {
		return nil, err
	}
	result.CanEdit = canSubmit && editableStatus && o.LockedAt == nil && !locked && o.ClosureStatus != orderent.ClosureStatusCLOSED && o.TerminationStatus == orderent.TerminationStatusACTIVE
	if !editableStatus {
		result.BlockedReasons = append(result.BlockedReasons, "费用审核中或已批准，不能直接修改")
	}
	if o.LockedAt != nil {
		result.BlockedReasons = append(result.BlockedReasons, "订单业务资料已锁定，原费用不能直接修改")
	}
	if locked {
		result.BlockedReasons = append(result.BlockedReasons, "存在有效提成，原费用不能直接修改")
	}
	if occupied {
		result.BlockedReasons = append(result.BlockedReasons, "存在有效账单，撤销批准前请先取消账单")
	}
	if pending {
		label := "费用补录"
		if result.HasPendingCorrection {
			label = "费用更正"
		}
		result.BlockedReasons = append(result.BlockedReasons, "存在待审核的"+label+"，暂停新建账及相关提成处理")
	}
	billPermission, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, access.FinanceBillCreate, false)
	if err != nil {
		return nil, err
	}
	result.CanBill = billPermission && validateApprovedOrderFees(ctx, client, o) == nil
	if o.FeeReviewStatus != orderent.FeeReviewStatusAPPROVED {
		result.BlockedReasons = append(result.BlockedReasons, "整票费用尚未批准，不能建账")
	}
	if history {
		notices, noticeErr := client.NotificationDelivery.Query().Where(noticeent.ResourceTypeEQ("ORDER"), noticeent.ResourceIDEQ(o.ID), noticeent.TemplateIn(noticeent.TemplateFEE_REVIEW_PENDING, noticeent.TemplateFEE_REVIEW_RESULT)).WithRecipientUser().WithBackgroundTask().All(ctx)
		if noticeErr != nil {
			return nil, noticeErr
		}
		for _, notice := range notices {
			recipient := notice.Edges.RecipientUser
			if recipient.DingtalkUserid == nil || strings.TrimSpace(*recipient.DingtalkUserid) == "" {
				result.NotificationUnboundCount++
			}
			task := notice.Edges.BackgroundTask
			if task.Status == taskent.StatusFAILED || task.Status == taskent.StatusDEAD_LETTER {
				result.NotificationFailedCount++
			}
		}
		events, err := client.OrderFeeReviewEvent.Query().Where(eventent.OrderIDEQ(o.ID)).Order(eventent.ByCreatedAt(), eventent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			var saved biz.FeeReviewSnapshot
			if err := json.Unmarshal(e.Snapshot, &saved); err != nil {
				return nil, err
			}
			result.History = append(result.History, &biz.FeeReviewEvent{ID: e.ID, Action: string(e.Action), ActorID: e.ActorID, ActorName: e.ActorName, Comment: e.Comment, CreatedAt: e.CreatedAt, ContentVersion: e.ContentVersion, SubmissionID: e.SubmissionID, Summary: saved.Summary})
		}
	}
	return result, nil
}
func (r *orderFeeReviewRepo) Act(ctx context.Context, p *biz.Principal, action string, input biz.FeeReviewAction) (*biz.OrderFeeReview, error) {
	payload, _ := json.Marshal(struct {
		Action  string
		Input   biz.FeeReviewAction
		ActorID uuid.UUID
	}{action, input, p.UserID})
	requestHash := biz.FeeReviewHash(payload)
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		client := tx.Client()
		if err := lockFeeAuthorizationParents(ctx, client, p.UserID); err != nil {
			return err
		}
		o, err := client.Order.Query().Where(orderent.IDEQ(input.OrderID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		authorized := false
		if action == "SUBMIT" || action == "WITHDRAW" {
			authorized, err = feeReviewSubmitter(ctx, client, p, o, false)
		} else {
			authorized, err = hasFeeReviewPermission(ctx, client, o.OrganizationID, p.UserID, false)
		}
		if err != nil {
			return err
		}
		if !authorized {
			return biz.ErrFeeReviewForbidden
		}
		previous, err := client.OrderFeeReviewEvent.Query().Where(eventent.OrderIDEQ(o.ID), eventent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
		if err == nil {
			if previous.RequestFingerprint != requestHash {
				return biz.ErrFeeReviewConflict
			}
			return nil
		}
		if !ent.IsNotFound(err) {
			return err
		}
		if o.FeeContentVersion != input.ExpectedContentVersion || o.FeeReviewVersion != input.ExpectedReviewVersion {
			return biz.ErrFeeReviewConflict
		}
		snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, client, o.ID)
		if err != nil {
			return err
		}
		status := o.FeeReviewStatus
		var submissionID *uuid.UUID
		if o.CurrentFeeReviewEventID != nil {
			e, err := client.OrderFeeReviewEvent.Get(ctx, *o.CurrentFeeReviewEventID)
			if err != nil {
				return err
			}
			if e.Action == eventent.ActionSUBMIT {
				submissionID = &e.ID
			}
		}
		pending, err := hasPendingFeeSupplement(ctx, client, o.ID)
		if err != nil {
			return err
		}
		if pending {
			return biz.ErrFeeReviewPendingSupplement
		}
		switch action {
		case "SUBMIT":
			if status == orderent.FeeReviewStatusPENDING || status == orderent.FeeReviewStatusAPPROVED {
				return biz.ErrFeeReviewConflict
			}
			if len(snapshot.Fees) == 0 {
				return biz.ErrFeeReviewInvalid
			}
			candidates, err := feeReviewGrants(ctx, client, o.OrganizationID)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				return biz.ErrFeeReviewNoApprover
			}
			status = orderent.FeeReviewStatusPENDING
		case "APPROVE", "REJECT", "WITHDRAW":
			if status != orderent.FeeReviewStatusPENDING || submissionID == nil || input.SubmissionID == nil || *input.SubmissionID != *submissionID {
				return biz.ErrFeeReviewConflict
			}
			submitted, err := client.OrderFeeReviewEvent.Get(ctx, *submissionID)
			if err != nil {
				return err
			}
			if submitted.ContentVersion != o.FeeContentVersion || submitted.ContentFingerprint != fingerprint {
				return biz.ErrFeeReviewConflict
			}
			if action == "WITHDRAW" {
				if submitted.ActorID != p.UserID {
					return biz.ErrFeeReviewForbidden
				}
				status = orderent.FeeReviewStatusUNSUBMITTED
			} else if action == "REJECT" {
				status = orderent.FeeReviewStatusREJECTED
			} else {
				if input.PreviewToken == "" || input.PreviewToken != feeReviewPreviewToken(o, fingerprint) {
					return biz.ErrFeeReviewConflict
				}
				if snapshot.Summary.Profit.IsNegative() && strings.TrimSpace(input.Comment) == "" {
					return biz.ErrFeeReviewInvalid
				}
				status = orderent.FeeReviewStatusAPPROVED
			}
		case "REVOKE":
			if status != orderent.FeeReviewStatusAPPROVED {
				return biz.ErrFeeReviewConflict
			}
			occupied, err := client.FinanceBillLine.Query().Where(lineent.OrderIDEQ(o.ID), effectiveBillLinePredicate()).Exist(ctx)
			if err != nil {
				return err
			}
			if occupied {
				return biz.ErrOrderFeeBillOccupied
			}
			status = orderent.FeeReviewStatusREVOKED
		default:
			return biz.ErrFeeReviewInvalid
		}
		event, err := appendFeeReviewEvent(ctx, client, o, p.UserID, action, input.Comment, input.IdempotencyKey, requestHash, submissionID, nil, snapshot, fingerprint)
		if err != nil {
			return err
		}
		update := client.Order.UpdateOne(o).SetFeeReviewStatus(status).SetFeeReviewVersion(o.FeeReviewVersion + 1).SetCurrentFeeReviewEventID(event.ID)
		if action == "APPROVE" {
			update.SetApprovedFeeReviewEventID(event.ID)
		} else if action == "REVOKE" {
			update.ClearApprovedFeeReviewEventID()
		}
		if _, err := update.Save(ctx); err != nil {
			return err
		}
		if err := enqueueFeeReviewNotifications(ctx, client, o, event); err != nil {
			return err
		}
		actorID := p.UserID
		orgID := o.OrganizationID
		return writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &orgID, UserID: &actorID, Action: "order.fee_review." + strings.ToLower(action), Result: "success", ResourceType: "order", ResourceID: o.ID.String(), Details: map[string]string{"event_id": event.ID.String(), "content_version": fmt.Sprint(o.FeeContentVersion), "content_fingerprint": fingerprint, "comment": input.Comment}})
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, p, input.OrderID)
}
func appendFeeReviewEvent(ctx context.Context, client *ent.Client, o *ent.Order, actorID uuid.UUID, action, comment, key, requestHash string, submissionID, supplementID *uuid.UUID, snapshot *biz.FeeReviewSnapshot, fingerprint string, correctionID ...*uuid.UUID) (*ent.OrderFeeReviewEvent, error) {
	actor, err := client.User.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	authorization := map[string]string{"permission": access.FinanceFeeReview, "organization_id": o.OrganizationID.String()}
	if action != "SUBMIT" && action != "WITHDRAW" {
		grants, err := feeReviewGrants(ctx, client, o.OrganizationID)
		if err != nil {
			return nil, err
		}
		for _, grant := range grants {
			if grant.Edges.Membership.UserID == actorID {
				authorization["role_id"] = grant.RoleID.String()
				authorization["membership_id"] = grant.MembershipID.String()
				authorization["data_scope"] = string(grant.Edges.Role.DataScope)
				break
			}
		}
	}
	builder := client.OrderFeeReviewEvent.Create().SetOrganizationID(o.OrganizationID).SetOrderID(o.ID).SetAction(eventent.Action(action)).SetActorID(actorID).SetActorName(actor.DisplayName).SetAuthorizationSnapshot(authorization).SetComment(comment).SetContentVersion(o.FeeContentVersion).SetContentFingerprint(fingerprint).SetSnapshot(payload).SetIdempotencyKey(key).SetRequestFingerprint(requestHash).SetNillableSubmissionID(submissionID).SetNillableSupplementRequestID(supplementID)
	if len(correctionID) > 0 {
		builder.SetNillableCorrectionRequestID(correctionID[0])
	}
	return builder.Save(ctx)
}
func bumpFeeReviewVersion(ctx context.Context, client *ent.Client, orderID uuid.UUID) error {
	return client.Order.UpdateOneID(orderID).AddFeeReviewVersion(1).Exec(ctx)
}
func refreshSupplementApproval(ctx context.Context, client *ent.Client, orderID, actorID, supplementID uuid.UUID, comment string) error {
	o, err := client.Order.UpdateOneID(orderID).AddFeeContentVersion(1).AddFeeReviewVersion(1).Save(ctx)
	if err != nil {
		return err
	}
	snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, client, orderID)
	if err != nil {
		return err
	}
	if snapshot.Summary.Profit.IsNegative() && strings.TrimSpace(comment) == "" {
		return biz.ErrFeeReviewInvalid
	}
	event, err := appendFeeReviewEvent(ctx, client, o, actorID, "SUPPLEMENT_APPROVE", comment, "supplement:"+supplementID.String(), fingerprint, nil, &supplementID, snapshot, fingerprint)
	if err != nil {
		return err
	}
	return client.Order.UpdateOneID(orderID).SetFeeReviewStatus(orderent.FeeReviewStatusAPPROVED).SetCurrentFeeReviewEventID(event.ID).SetApprovedFeeReviewEventID(event.ID).Exec(ctx)
}
func revokeSupplementApproval(ctx context.Context, client *ent.Client, orderID, actorID, supplementID uuid.UUID, reason string) error {
	o, err := client.Order.UpdateOneID(orderID).AddFeeContentVersion(1).AddFeeReviewVersion(1).Save(ctx)
	if err != nil {
		return err
	}
	snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, client, orderID)
	if err != nil {
		return err
	}
	e, err := appendFeeReviewEvent(ctx, client, o, actorID, "REVOKE", reason, "supplement-cancel:"+supplementID.String(), fingerprint, nil, &supplementID, snapshot, fingerprint)
	if err != nil {
		return err
	}
	return client.Order.UpdateOneID(orderID).SetFeeReviewStatus(orderent.FeeReviewStatusREVOKED).SetCurrentFeeReviewEventID(e.ID).ClearApprovedFeeReviewEventID().Exec(ctx)
}

// lockBillFeeOrders 在任何 Fee 行锁之前定位并锁完整订单集合；锁后重读归属。
func lockBillFeeOrders(ctx context.Context, client *ent.Client, organizationIDs, feeIDs []uuid.UUID, validate bool) error {
	fees, err := client.OrderFee.Query().Where(feeent.IDIn(feeIDs...), feeent.HasOrderWith(orderent.OrganizationIDIn(organizationIDs...))).All(ctx)
	if err != nil {
		return err
	}
	if len(fees) != len(feeIDs) {
		return biz.ErrFinanceBillFeeInvalid
	}
	seen := map[uuid.UUID]bool{}
	ids := []uuid.UUID{}
	for _, fee := range fees {
		if !seen[fee.OrderID] {
			seen[fee.OrderID] = true
			ids = append(ids, fee.OrderID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDIn(organizationIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(ids) {
		return biz.ErrFinanceBillFeeInvalid
	}
	latest, err := client.OrderFee.Query().Where(feeent.IDIn(feeIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(latest) != len(fees) {
		return biz.ErrFeeReviewConflict
	}
	old := map[uuid.UUID]uuid.UUID{}
	for _, fee := range fees {
		old[fee.ID] = fee.OrderID
	}
	for _, fee := range latest {
		if old[fee.ID] != fee.OrderID {
			return biz.ErrFeeReviewConflict
		}
	}
	if validate {
		for _, o := range orders {
			if err := validateApprovedOrderFees(ctx, client, o); err != nil {
				return err
			}
		}
	}
	return nil
}

// LoadOrderFeeReviewSnapshot 供正式种子在同一事务中固化明确的演示审核事实。
func LoadOrderFeeReviewSnapshot(ctx context.Context, client *ent.Client, orderID uuid.UUID) (*biz.FeeReviewSnapshot, string, error) {
	return loadFeeReviewSnapshot(ctx, client, orderID)
}
func qualifiedFeeReviewCandidates(ctx context.Context, client *ent.Client, organizationID uuid.UUID) ([]biz.OrderFeeSupplementApprover, error) {
	grants, err := feeReviewGrants(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	seen := map[uuid.UUID]bool{}
	out := []biz.OrderFeeSupplementApprover{}
	for _, grant := range grants {
		user := grant.Edges.Membership.Edges.User
		if seen[user.ID] {
			continue
		}
		seen[user.ID] = true
		dt := ""
		if user.DingtalkUserid != nil {
			dt = *user.DingtalkUserid
		}
		out = append(out, biz.OrderFeeSupplementApprover{UserID: user.ID, DisplayName: user.DisplayName, DingTalkUserID: dt})
	}
	return out, nil
}

func orderHasPendingSupplementPredicate() predicate.Order {
	return orderent.Or(orderent.HasFeeSupplementRequestsWith(supplementent.StatusEQ(supplementent.StatusPENDING)), orderent.HasFeeCorrectionRequestsWith(correctionent.StatusEQ(correctionent.StatusPENDING)))
}

// attachOrderFeeReviewProjections 仅批量读取当前页费用与事件；无费用读权限时不投影金额。
func attachOrderFeeReviewProjections(ctx context.Context, client *ent.Client, items []*biz.Order) error {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	orderMap := map[uuid.UUID]*ent.Order{}
	eventsIDs := []uuid.UUID{}
	readable := map[uuid.UUID]bool{}
	reviewer := map[uuid.UUID]bool{}
	permissionCache := map[string]bool{}
	for _, o := range orders {
		orderMap[o.ID] = o
		if o.CurrentFeeReviewEventID != nil {
			eventsIDs = append(eventsIDs, *o.CurrentFeeReviewEventID)
		}
		for _, key := range []string{access.FinanceFeeRead, access.FinanceFeeReview, access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeRead)} {
			cacheKey := o.OrganizationID.String() + ":" + key
			yes, cached := permissionCache[cacheKey]
			if !cached {
				yes, err = hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, key, false)
				if err != nil {
					return err
				}
				permissionCache[cacheKey] = yes
			}
			if yes {
				readable[o.ID] = true
				if key == access.FinanceFeeReview {
					reviewer[o.ID] = true
				}
			}
		}
	}
	permitted := []uuid.UUID{}
	for _, o := range orders {
		if readable[o.ID] {
			permitted = append(permitted, o.ID)
		}
	}
	fees, err := client.OrderFee.Query().Where(feeent.OrderIDIn(permitted...)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if err != nil {
		return err
	}
	grouped := map[uuid.UUID][]*biz.OrderFee{}
	for _, fee := range fees {
		converted, err := orderFeeToBiz(fee)
		if err != nil {
			return err
		}
		grouped[fee.OrderID] = append(grouped[fee.OrderID], converted)
	}
	pending, err := client.OrderFeeSupplementRequest.Query().Where(supplementent.OrderIDIn(ids...), supplementent.StatusEQ(supplementent.StatusPENDING)).All(ctx)
	if err != nil {
		return err
	}
	pendingSet := map[uuid.UUID]bool{}
	for _, request := range pending {
		pendingSet[request.OrderID] = true
	}
	corrections, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDIn(ids...), correctionent.StatusEQ(correctionent.StatusPENDING)).All(ctx)
	if err != nil {
		return err
	}
	for _, request := range corrections {
		pendingSet[request.OrderID] = true
	}
	events, err := client.OrderFeeReviewEvent.Query().Where(eventent.IDIn(eventsIDs...)).All(ctx)
	if err != nil {
		return err
	}
	eventMap := map[uuid.UUID]*ent.OrderFeeReviewEvent{}
	for _, e := range events {
		eventMap[e.ID] = e
	}
	for _, item := range items {
		o := orderMap[item.ID]
		if o == nil {
			continue
		}
		r := item.FeeReview
		if r == nil {
			continue
		}
		r.HasPendingChange = pendingSet[o.ID]
		for _, c := range corrections {
			if c.OrderID == o.ID {
				r.HasPendingCorrection = true
				break
			}
		}
		r.HasPendingSupplement = r.HasPendingChange && !r.HasPendingCorrection
		r.CanApprove = reviewer[o.ID] && o.FeeReviewStatus == orderent.FeeReviewStatusPENDING
		if readable[o.ID] {
			snapshot, fingerprint, err := biz.BuildFeeReviewSnapshot(grouped[o.ID])
			if err != nil {
				return err
			}
			r.Summary = &snapshot.Summary
			r.PreviewToken = feeReviewPreviewToken(o, fingerprint)
			if o.CurrentFeeReviewEventID != nil {
				e := eventMap[*o.CurrentFeeReviewEventID]
				if e != nil && e.Action == eventent.ActionSUBMIT {
					r.SubmissionID = &e.ID
					r.Summary.SubmittedByName = e.ActorName
					r.Summary.SubmittedAt = &e.CreatedAt
				}
			}
		}
	}
	return nil
}

func lockExistingBillOrders(ctx context.Context, client *ent.Client, organizationIDs, billIDs []uuid.UUID) error {
	lines, err := client.FinanceBillLine.Query().Where(lineent.BillIDIn(billIDs...), lineent.ActiveEQ(true)).All(ctx)
	if err != nil {
		return err
	}
	ids := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	for _, line := range lines {
		if !seen[line.OrderID] {
			ids = append(ids, line.OrderID)
			seen[line.OrderID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDIn(organizationIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(ids) {
		return biz.ErrFinanceBillNotFound
	}
	return nil
}

// batchFeeReviewApprovalStates 在台账/建账候选列表中按订单批量校验，避免每行重读整票费用。
func batchFeeReviewApprovalStates(ctx context.Context, client *ent.Client, orders []*ent.Order) (map[uuid.UUID]error, map[uuid.UUID]bool, error) {
	ids := []uuid.UUID{}
	eventIDs := []uuid.UUID{}
	unique := map[uuid.UUID]*ent.Order{}
	for _, o := range orders {
		if _, seen := unique[o.ID]; seen {
			continue
		}
		unique[o.ID] = o
		ids = append(ids, o.ID)
		if o.ApprovedFeeReviewEventID != nil {
			eventIDs = append(eventIDs, *o.ApprovedFeeReviewEventID)
		}
	}
	states := map[uuid.UUID]error{}
	pending := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return states, pending, nil
	}
	fees, err := client.OrderFee.Query().Where(feeent.OrderIDIn(ids...)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	grouped := map[uuid.UUID][]*biz.OrderFee{}
	for _, f := range fees {
		fee, err := orderFeeToBiz(f)
		if err != nil {
			return nil, nil, err
		}
		grouped[f.OrderID] = append(grouped[f.OrderID], fee)
	}
	events, err := client.OrderFeeReviewEvent.Query().Where(eventent.IDIn(eventIDs...)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	eventMap := map[uuid.UUID]*ent.OrderFeeReviewEvent{}
	for _, e := range events {
		eventMap[e.ID] = e
	}
	requests, err := client.OrderFeeSupplementRequest.Query().Where(supplementent.OrderIDIn(ids...), supplementent.StatusEQ(supplementent.StatusPENDING)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, request := range requests {
		pending[request.OrderID] = true
	}
	corrections, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDIn(ids...), correctionent.StatusEQ(correctionent.StatusPENDING)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, request := range corrections {
		pending[request.OrderID] = true
	}
	for id, o := range unique {
		states[id] = biz.ErrFeeReviewNotApproved
		if o.FeeReviewStatus != orderent.FeeReviewStatusAPPROVED || o.ApprovedFeeReviewEventID == nil {
			continue
		}
		e := eventMap[*o.ApprovedFeeReviewEventID]
		if e == nil || e.OrderID != id || e.OrganizationID != o.OrganizationID || e.ContentVersion != o.FeeContentVersion || (e.Action != eventent.ActionAPPROVE && e.Action != eventent.ActionSUPPLEMENT_APPROVE && e.Action != eventent.ActionCORRECTION_APPROVE) {
			continue
		}
		_, fingerprint, err := biz.BuildFeeReviewSnapshot(grouped[id])
		if err != nil {
			states[id] = err
			continue
		}
		if fingerprint != e.ContentFingerprint {
			continue
		}
		states[id] = nil
		if pending[id] {
			states[id] = biz.ErrFeeReviewPendingSupplement
		}
	}
	return states, pending, nil
}
