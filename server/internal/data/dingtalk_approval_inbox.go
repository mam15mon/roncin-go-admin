package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	dingtalkapprovalinboxeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovalinboxevent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockapprovercandidateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockapprovercandidate"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

func (r *dingTalkApprovalRepo) ClaimInbox(ctx context.Context, leaseDuration time.Duration, now time.Time) (*biz.DingTalkApprovalInboxJob, error) {
	var claimed *ent.DingTalkApprovalInboxEvent
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		event, err := tx.DingTalkApprovalInboxEvent.Query().
			Where(dingtalkapprovalinboxeventent.Or(
				dingtalkapprovalinboxeventent.And(
					dingtalkapprovalinboxeventent.StatusIn(dingtalkapprovalinboxeventent.StatusRECEIVED, dingtalkapprovalinboxeventent.StatusFAILED),
					dingtalkapprovalinboxeventent.NextRunAtLTE(now),
				),
				dingtalkapprovalinboxeventent.And(
					dingtalkapprovalinboxeventent.StatusEQ(dingtalkapprovalinboxeventent.StatusPROCESSING),
					dingtalkapprovalinboxeventent.ProcessingExpiresAtLTE(now),
				),
			)).
			Order(dingtalkapprovalinboxeventent.ByNextRunAt(), dingtalkapprovalinboxeventent.ByReceivedAt()).
			ForUpdate().
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrBackgroundTaskNoTask
			}
			return err
		}
		attempts := event.Attempts
		if event.Status == dingtalkapprovalinboxeventent.StatusPROCESSING {
			attempts++
		}
		claimed, err = tx.DingTalkApprovalInboxEvent.UpdateOne(event).
			SetStatus(dingtalkapprovalinboxeventent.StatusPROCESSING).
			SetAttempts(attempts).
			SetProcessingToken(uuid.NewString()).
			SetProcessingExpiresAt(now.Add(leaseDuration)).
			ClearErrorMessage().
			Save(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &biz.DingTalkApprovalInboxJob{ID: claimed.ID, LeaseToken: *claimed.ProcessingToken, ProcessInstanceID: claimed.ProcessInstanceID}, nil
}

func (r *dingTalkApprovalRepo) FailInbox(ctx context.Context, job *biz.DingTalkApprovalInboxJob, message string, nextRunAt time.Time) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		event, err := lockApprovalInboxEvent(ctx, tx, job)
		if err != nil {
			return err
		}
		_, err = tx.DingTalkApprovalInboxEvent.UpdateOne(event).
			SetStatus(dingtalkapprovalinboxeventent.StatusFAILED).
			SetAttempts(event.Attempts + 1).
			SetNextRunAt(nextRunAt).
			SetErrorMessage(trimRunes(message, 500)).
			ClearProcessingToken().ClearProcessingExpiresAt().
			Save(ctx)
		return err
	})
}

func (r *dingTalkApprovalRepo) IgnoreInbox(ctx context.Context, job *biz.DingTalkApprovalInboxJob, resultCode string) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		event, err := lockApprovalInboxEvent(ctx, tx, job)
		if err != nil {
			return err
		}
		return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusIGNORED, resultCode)
	})
}

func (r *dingTalkApprovalRepo) RecordRejected(ctx context.Context, job *biz.DingTalkApprovalInboxJob, result *biz.DingTalkApprovalQueryResult, now time.Time) error {
	request, err := r.findApprovalRequest(ctx, job.ProcessInstanceID)
	if err != nil {
		return err
	}
	if request == nil {
		return r.IgnoreInbox(ctx, job, "REQUEST_NOT_FOUND")
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		event, err := lockApprovalInboxEvent(ctx, tx, job)
		if err != nil {
			return err
		}
		order, lockRecord, request, err := lockApprovalOrderFacts(ctx, tx, request)
		if err != nil {
			return err
		}
		_ = lockRecord
		if request.Status == orderunlockrequestent.StatusPENDING_APPROVAL {
			decidedAt := now
			if result.DecidedAt != nil {
				decidedAt = result.DecidedAt.UTC()
			}
			update := tx.OrderUnlockRequest.UpdateOne(request).
				SetStatus(orderunlockrequestent.StatusREJECTED).
				SetDecidedAt(decidedAt).
				SetDecisionSource("DINGTALK_OA").
				ClearFailureCode().ClearFailureMessage()
			user, err := findDingTalkUser(ctx, tx.Client(), result.ApproverUserID)
			if err != nil {
				return err
			}
			if user != nil {
				update.SetDecidedBy(user.ID)
			}
			if _, err := update.Save(ctx); err != nil {
				return err
			}
			audit := &biz.AuditEvent{
				OrganizationID: &request.OrganizationID,
				Action:         "order.unlock.dingtalk_rejected",
				ResourceType:   "order",
				ResourceID:     order.ID.String(),
				Result:         "success",
				Details: map[string]string{
					"order_no":          order.OrderNo,
					"unlock_request_id": request.ID.String(),
					"lock_generation":   fmt.Sprintf("%d", request.LockGeneration),
				},
			}
			if user != nil {
				audit.UserID = &user.ID
			}
			if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
				return err
			}
		}
		if _, err := tx.DingTalkApprovalInboxEvent.UpdateOne(event).SetOrganizationID(request.OrganizationID).Save(ctx); err != nil {
			return err
		}
		return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusPROCESSED, "REJECTED")
	})
}

func (r *dingTalkApprovalRepo) PrepareApproved(ctx context.Context, job *biz.DingTalkApprovalInboxJob, result *biz.DingTalkApprovalQueryResult, now time.Time) (uuid.UUID, bool, error) {
	request, err := r.findApprovalRequest(ctx, job.ProcessInstanceID)
	if err != nil {
		return uuid.Nil, false, err
	}
	if request == nil {
		return uuid.Nil, false, r.IgnoreInbox(ctx, job, "REQUEST_NOT_FOUND")
	}
	shouldApply := false
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		event, err := lockApprovalInboxEvent(ctx, tx, job)
		if err != nil {
			return err
		}
		_, _, request, err = lockApprovalOrderFacts(ctx, tx, request)
		if err != nil {
			return err
		}
		switch request.Status {
		case orderunlockrequestent.StatusPENDING_APPROVAL:
			decidedAt := now
			if result.DecidedAt != nil {
				decidedAt = result.DecidedAt.UTC()
			}
			request, err = tx.OrderUnlockRequest.UpdateOne(request).
				SetStatus(orderunlockrequestent.StatusAPPROVED_PENDING_APPLY).
				SetDecidedAt(decidedAt).
				SetDecisionSource("DINGTALK_OA").
				ClearFailureCode().ClearFailureMessage().
				Save(ctx)
			if err != nil {
				return err
			}
			shouldApply = true
		case orderunlockrequestent.StatusAPPROVED_PENDING_APPLY:
			shouldApply = true
		case orderunlockrequestent.StatusAPPROVED:
			return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusPROCESSED, "ALREADY_APPLIED")
		default:
			return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusIGNORED, "REQUEST_SUPERSEDED")
		}
		_, err = tx.DingTalkApprovalInboxEvent.UpdateOne(event).SetOrganizationID(request.OrganizationID).Save(ctx)
		return err
	})
	return request.ID, shouldApply, err
}

func (r *dingTalkApprovalRepo) ApplyApproved(ctx context.Context, job *biz.DingTalkApprovalInboxJob, requestID uuid.UUID, result *biz.DingTalkApprovalQueryResult, now time.Time) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	request, err := client.OrderUnlockRequest.Get(ctx, requestID)
	if err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		event, err := lockApprovalInboxEvent(ctx, tx, job)
		if err != nil {
			return err
		}
		order, lockRecord, request, err := lockApprovalOrderFacts(ctx, tx, request)
		if err != nil {
			return err
		}
		if request.Status == orderunlockrequestent.StatusAPPROVED {
			return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusPROCESSED, "ALREADY_APPLIED")
		}
		if request.Status != orderunlockrequestent.StatusAPPROVED_PENDING_APPLY {
			return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusIGNORED, "REQUEST_SUPERSEDED")
		}
		if order.LockedAt == nil || order.LockGeneration != request.LockGeneration || order.Version != request.ExpectedOrderVersion || lockRecord.UnlockedAt != nil {
			return markApprovalStale(ctx, tx, event, request, "ORDER_LOCK_CHANGED", "审批对应的锁定代次或订单版本已变化")
		}

		candidates, err := tx.OrderUnlockApproverCandidate.Query().
			Where(orderunlockapprovercandidateent.RequestIDEQ(request.ID)).
			Order(orderunlockapprovercandidateent.ByID()).
			ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		approver, err := findDingTalkUser(ctx, tx.Client(), result.ApproverUserID)
		if err != nil {
			return err
		}
		candidateMatched := false
		if approver != nil {
			for _, candidate := range candidates {
				if candidate.UserID == approver.ID && candidate.DingtalkUseridSnapshot == result.ApproverUserID {
					candidateMatched = true
					break
				}
			}
		}
		if !candidateMatched || approver == nil {
			return markApprovalStale(ctx, tx, event, request, "APPROVER_NOT_IN_SNAPSHOT", "钉钉审批人不在申请时的候选快照中")
		}
		qualified, err := isUserQualifiedBusinessLockRole(ctx, tx.Client(), request.OrganizationID, approver.ID)
		if err != nil {
			return err
		}
		if !qualified {
			return markApprovalStale(ctx, tx, event, request, "APPROVER_NOT_QUALIFIED", "钉钉审批人当前已不具备解锁业务角色资格")
		}

		newVersion := order.Version + 1
		if _, err := tx.Order.UpdateOne(order).ClearLockedAt().ClearLockedBy().SetVersion(newVersion).Save(ctx); err != nil {
			return err
		}
		decidedAt := now
		if result.DecidedAt != nil {
			decidedAt = result.DecidedAt.UTC()
		}
		if _, err := tx.OrderUnlockRequest.UpdateOne(request).
			SetStatus(orderunlockrequestent.StatusAPPROVED).
			SetDecidedBy(approver.ID).
			SetDecidedAt(decidedAt).
			SetDecisionSource("DINGTALK_OA").
			SetUnlockedAt(now).
			SetResultOrderVersion(newVersion).
			ClearFailureCode().ClearFailureMessage().
			Save(ctx); err != nil {
			return err
		}
		if _, err := tx.OrderLockRecord.UpdateOne(lockRecord).
			SetUnlockedBy(approver.ID).
			SetUnlockedAt(now).
			SetOrderVersionAtUnlock(newVersion).
			SetUnlockRequestID(request.ID).
			SetUnlockMode(orderlockrecordent.UnlockModeDINGTALK_APPROVED).
			SetNillableUnlockReason(request.Reason).
			Save(ctx); err != nil {
			return err
		}
		audit := &biz.AuditEvent{
			OrganizationID: &request.OrganizationID,
			UserID:         &approver.ID,
			Action:         "order.unlock.dingtalk_approved",
			ResourceType:   "order",
			ResourceID:     order.ID.String(),
			Result:         "success",
			Details: map[string]string{
				"order_no":             order.OrderNo,
				"unlock_request_id":    request.ID.String(),
				"lock_generation":      fmt.Sprintf("%d", request.LockGeneration),
				"result_order_version": fmt.Sprintf("%d", newVersion),
			},
		}
		if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
			return err
		}
		return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusPROCESSED, "APPROVED_APPLIED")
	})
}

func (r *dingTalkApprovalRepo) findApprovalRequest(ctx context.Context, processInstanceID string) (*ent.OrderUnlockRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	request, err := client.OrderUnlockRequest.Query().
		Where(orderunlockrequestent.DingtalkProcessInstanceIDEQ(processInstanceID)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return request, err
}

func lockApprovalInboxEvent(ctx context.Context, tx *ent.Tx, job *biz.DingTalkApprovalInboxJob) (*ent.DingTalkApprovalInboxEvent, error) {
	if job == nil || job.ID == uuid.Nil || job.LeaseToken == "" {
		return nil, fmt.Errorf("Inbox 租约无效")
	}
	event, err := tx.DingTalkApprovalInboxEvent.Query().Where(dingtalkapprovalinboxeventent.IDEQ(job.ID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, err
	}
	if event.Status != dingtalkapprovalinboxeventent.StatusPROCESSING || event.ProcessingToken == nil || *event.ProcessingToken != job.LeaseToken ||
		(event.ProcessingExpiresAt != nil && event.ProcessingExpiresAt.Before(time.Now())) {
		return nil, biz.ErrBackgroundTaskLeaseMismatch
	}
	return event, nil
}

func lockApprovalOrderFacts(ctx context.Context, tx *ent.Tx, located *ent.OrderUnlockRequest) (*ent.Order, *ent.OrderLockRecord, *ent.OrderUnlockRequest, error) {
	// 与 RequestOrderUnlock 保持同一顺序：Order → LockRecord → UnlockRequest。
	order, err := tx.Order.Query().Where(orderent.IDEQ(located.OrderID), orderent.OrganizationIDEQ(located.OrganizationID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	lockRecord, err := tx.OrderLockRecord.Query().
		Where(orderlockrecordent.IDEQ(located.LockRecordID), orderlockrecordent.OrderIDEQ(order.ID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	request, err := tx.OrderUnlockRequest.Query().Where(orderunlockrequestent.IDEQ(located.ID)).ForUpdate().Only(ctx)
	return order, lockRecord, request, err
}

func finishApprovalInbox(ctx context.Context, tx *ent.Tx, event *ent.DingTalkApprovalInboxEvent, status dingtalkapprovalinboxeventent.Status, resultCode string) error {
	_, err := tx.DingTalkApprovalInboxEvent.UpdateOne(event).
		SetStatus(status).
		SetResultCode(trimRunes(resultCode, 64)).
		ClearErrorMessage().ClearProcessingToken().ClearProcessingExpiresAt().
		Save(ctx)
	return err
}

func markApprovalStale(ctx context.Context, tx *ent.Tx, event *ent.DingTalkApprovalInboxEvent, request *ent.OrderUnlockRequest, code, message string) error {
	if _, err := tx.OrderUnlockRequest.UpdateOne(request).
		SetStatus(orderunlockrequestent.StatusSTALE).
		SetFailureCode(code).
		SetFailureMessage(message).
		Save(ctx); err != nil {
		return err
	}
	return finishApprovalInbox(ctx, tx, event, dingtalkapprovalinboxeventent.StatusPROCESSED, code)
}

func findDingTalkUser(ctx context.Context, client *ent.Client, dingTalkUserID string) (*ent.User, error) {
	if dingTalkUserID == "" {
		return nil, nil
	}
	user, err := client.User.Query().Where(userent.DingtalkUseridEQ(dingTalkUserID)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}
