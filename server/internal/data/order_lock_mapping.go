package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
)

func (r *orderLockRepo) mapLockRecordByID(ctx context.Context, recordID uuid.UUID) (*biz.OrderLockRecord, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := client.OrderLockRecord.Query().
		Where(orderlockrecordent.IDEQ(recordID)).
		WithHouseBillSnapshots().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapLockRecord(ctx, client, rec), nil
}

func (r *orderLockRepo) mapLockRecord(ctx context.Context, client *ent.Client, rec *ent.OrderLockRecord) *biz.OrderLockRecord {
	var lockedByName string
	if rec.LockedBy != nil {
		if u, err := client.User.Get(ctx, *rec.LockedBy); err == nil && u != nil {
			lockedByName = u.DisplayName
		}
	}
	var unlockedByName *string
	if rec.UnlockedBy != nil {
		if u, err := client.User.Get(ctx, *rec.UnlockedBy); err == nil && u != nil {
			unlockedByName = &u.DisplayName
		}
	}
	var unlockMode *string
	if rec.UnlockMode != nil {
		modeStr := string(*rec.UnlockMode)
		unlockMode = &modeStr
	}
	// 自动锁定没有实际锁定人；触发操作人只作为审计信息随记录返回。
	var triggerType *string
	if rec.TriggerType != nil {
		value := string(*rec.TriggerType)
		triggerType = &value
	}
	var triggeredByName *string
	if rec.TriggeredBy != nil {
		if u, err := client.User.Get(ctx, *rec.TriggeredBy); err == nil && u != nil {
			triggeredByName = &u.DisplayName
		}
	}

	res := &biz.OrderLockRecord{
		ID:                   rec.ID,
		OrganizationID:       rec.OrganizationID,
		OrderID:              rec.OrderID,
		OrderNo:              rec.OrderNo,
		BusinessType:         biz.OrderBusinessType(rec.BusinessType),
		Generation:           rec.Generation,
		LockSource:           string(rec.LockSource),
		LockedBy:             rec.LockedBy,
		LockedByName:         lockedByName,
		LockedAt:             rec.LockedAt,
		OrderVersionAtLock:   rec.OrderVersionAtLock,
		TriggerType:          triggerType,
		TriggerResourceID:    rec.TriggerResourceID,
		TriggeredBy:          rec.TriggeredBy,
		TriggeredByName:      triggeredByName,
		MasterBillID:         rec.MasterBillID,
		MasterBillVersionID:  rec.MasterBillVersionID,
		UnlockedBy:           rec.UnlockedBy,
		UnlockedByName:       unlockedByName,
		UnlockedAt:           rec.UnlockedAt,
		OrderVersionAtUnlock: rec.OrderVersionAtUnlock,
		UnlockRequestID:      rec.UnlockRequestID,
		UnlockReason:         rec.UnlockReason,
		UnlockMode:           unlockMode,
	}

	for _, s := range rec.Edges.HouseBillSnapshots {
		res.HouseBillSnapshots = append(res.HouseBillSnapshots, &biz.OrderLockHouseBillSnapshot{
			ID:                 s.ID,
			OrganizationID:     s.OrganizationID,
			LockRecordID:       s.LockRecordID,
			HouseBillID:        s.HouseBillID,
			HouseBillVersionID: s.HouseBillVersionID,
			HouseNoSnapshot:    s.HouseNoSnapshot,
			CreatedAt:          s.CreatedAt,
		})
	}
	return res
}

func (r *orderLockRepo) mapUnlockRequestByID(ctx context.Context, requestID uuid.UUID) (*biz.OrderUnlockRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	req, err := client.OrderUnlockRequest.Query().
		Where(orderunlockrequestent.IDEQ(requestID)).
		WithApproverCandidates().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapUnlockRequest(ctx, client, req), nil
}

func (r *orderLockRepo) mapUnlockRequest(ctx context.Context, client *ent.Client, req *ent.OrderUnlockRequest) *biz.OrderUnlockRequest {
	var requestedByName string
	if u, err := client.User.Get(ctx, req.RequestedBy); err == nil && u != nil {
		requestedByName = u.DisplayName
	}
	var decidedByName *string
	if req.DecidedBy != nil {
		if u, err := client.User.Get(ctx, *req.DecidedBy); err == nil && u != nil {
			decidedByName = &u.DisplayName
		}
	}

	res := &biz.OrderUnlockRequest{
		ID:                        req.ID,
		OrganizationID:            req.OrganizationID,
		OrderID:                   req.OrderID,
		OrderNo:                   req.OrderNo,
		BusinessType:              biz.OrderBusinessType(req.BusinessType),
		LockRecordID:              req.LockRecordID,
		LockGeneration:            req.LockGeneration,
		RequestedBy:               req.RequestedBy,
		RequestedByName:           requestedByName,
		RequestedAt:               req.RequestedAt,
		Reason:                    req.Reason,
		ExpectedOrderVersion:      req.ExpectedOrderVersion,
		IdempotencyKey:            req.IdempotencyKey,
		RequestFingerprint:        req.RequestFingerprint,
		Route:                     string(req.Route),
		Status:                    string(req.Status),
		DingTalkProcessInstanceID: req.DingtalkProcessInstanceID,
		DingTalkProcessCode:       req.DingtalkProcessCode,
		DecidedBy:                 req.DecidedBy,
		DecidedByName:             decidedByName,
		DecidedAt:                 req.DecidedAt,
		DecisionSource:            req.DecisionSource,
		FailureCode:               req.FailureCode,
		FailureMessage:            req.FailureMessage,
		SupersededByRequestID:     req.SupersededByRequestID,
		UnlockedAt:                req.UnlockedAt,
		ResultOrderVersion:        req.ResultOrderVersion,
	}

	for _, c := range req.Edges.ApproverCandidates {
		res.ApproverCandidates = append(res.ApproverCandidates, &biz.OrderUnlockApproverCandidate{
			ID:                     c.ID,
			RequestID:              c.RequestID,
			UserID:                 c.UserID,
			MembershipID:           c.MembershipID,
			RoleID:                 c.RoleID,
			DisplayNameSnapshot:    c.DisplayNameSnapshot,
			DingTalkUserIDSnapshot: c.DingtalkUseridSnapshot,
			CreatedAt:              c.CreatedAt,
		})
	}
	return res
}
