package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	assignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
)

// seedCreatedOrderFeeReviews 只为本轮新建的演示订单构造明确的提交和批准事实。
// 重复同步不把存量订单补成已批准，也不覆盖人工审核或发送真实机器人消息。
func seedCreatedOrderFeeReviews(ctx context.Context, sc *seedContext) error {
	ids := make([]uuid.UUID, 0, len(sc.newOrderIDs))
	for id := range sc.newOrderIDs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	for _, id := range ids {
		o, err := sc.tx.Order.Get(ctx, id)
		if err != nil {
			return err
		}
		snapshot, fingerprint, err := data.LoadOrderFeeReviewSnapshot(ctx, sc.tx.Client(), id)
		if err != nil {
			return fmt.Errorf("构造订单 %s 费用快照: %w", o.OrderNo, err)
		}
		if len(snapshot.Fees) == 0 {
			continue
		}
		submission, err := seedFeeReviewEvent(ctx, sc, o, snapshot, fingerprint, eventent.ActionSUBMIT, nil, nil)
		if err != nil {
			return err
		}
		approval, err := seedFeeReviewEvent(ctx, sc, o, snapshot, fingerprint, eventent.ActionAPPROVE, &submission.ID, nil)
		if err != nil {
			return err
		}
		updated, err := sc.tx.Order.UpdateOne(o).
			SetFeeReviewStatus(orderent.FeeReviewStatusAPPROVED).
			AddFeeReviewVersion(2).
			SetCurrentFeeReviewEventID(approval.ID).
			SetApprovedFeeReviewEventID(approval.ID).
			Save(ctx)
		if err != nil {
			return err
		}
		sc.orders[o.OrderNo] = updated
	}
	return nil
}

func seedFeeReviewEvent(ctx context.Context, sc *seedContext, o *ent.Order, snapshot *biz.FeeReviewSnapshot, fingerprint string, action eventent.Action, submissionID, supplementID *uuid.UUID) (*ent.OrderFeeReviewEvent, error) {
	actor := sc.users["zhaofang"]
	if action == eventent.ActionSUBMIT {
		actor = sc.users["wangli"]
	}
	grant, err := sc.tx.RoleAssignment.Query().
		Where(assignmentent.HasMembershipWith(membershipent.UserIDEQ(actor.ID), membershipent.OrganizationIDEQ(sc.company.ID)), assignmentent.HasRoleWith(roleent.CodeEQ("administrator"))).
		WithRole().Order(assignmentent.ByID()).First(ctx)
	if err != nil {
		return nil, err
	}
	permission := access.FinanceFeeReview
	if action == eventent.ActionSUBMIT {
		permission = access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeCreate)
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	comment := "开发演示：整票费用已核对，按演示业务情形批准"
	if snapshot.Summary.Profit.IsNegative() {
		comment = "开发演示：该票因演示成本高于收入亏损，已核对完整费用，同意承担该项亏损"
	}
	key := "DEV-FEE-REVIEW-" + o.OrderNo + "-" + string(action)
	if supplementID != nil {
		key += "-" + supplementID.String()
	}
	return sc.tx.OrderFeeReviewEvent.Create().SetOrganizationID(o.OrganizationID).SetOrderID(o.ID).
		SetAction(action).SetActorID(actor.ID).SetActorName(actor.DisplayName).
		SetAuthorizationSnapshot(map[string]string{"permission": permission, "role_id": grant.RoleID.String(), "membership_id": grant.MembershipID.String(), "data_scope": string(grant.Edges.Role.DataScope), "source": "sync-dev-explicit-fixture"}).
		SetComment(comment).SetContentVersion(o.FeeContentVersion).SetContentFingerprint(fingerprint).
		SetSnapshot(payload).SetIdempotencyKey(key).SetRequestFingerprint(biz.FeeReviewHash([]byte(key + "|" + fingerprint))).
		SetNillableSubmissionID(submissionID).SetNillableSupplementRequestID(supplementID).Save(ctx)
}

func seedSupplementReviewSnapshot(ctx context.Context, sc *seedContext, orderID, supplementID uuid.UUID) (*ent.Order, error) {
	o, err := sc.tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).AddFeeReviewVersion(1).Save(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, fingerprint, err := data.LoadOrderFeeReviewSnapshot(ctx, sc.tx.Client(), orderID)
	if err != nil {
		return nil, err
	}
	event, err := seedFeeReviewEvent(ctx, sc, o, snapshot, fingerprint, eventent.ActionSUPPLEMENT_APPROVE, nil, &supplementID)
	if err != nil {
		return nil, err
	}
	return sc.tx.Order.UpdateOne(o).SetCurrentFeeReviewEventID(event.ID).SetApprovedFeeReviewEventID(event.ID).Save(ctx)
}
