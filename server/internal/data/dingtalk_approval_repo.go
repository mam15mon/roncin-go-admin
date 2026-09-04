package data

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	dingtalkapprovaldispatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovaldispatch"
	dingtalkapprovalinboxeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovalinboxevent"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
)

type dingTalkApprovalRepo struct {
	data *Data
}

func (r *dingTalkApprovalRepo) StoreCallback(ctx context.Context, event *biz.DingTalkApprovalCallbackEvent) error {
	if event == nil || strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.ProcessInstanceID) == "" {
		return fmt.Errorf("钉钉审批回调事件不完整")
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	_, err = client.DingTalkApprovalInboxEvent.Create().
		SetEventID(trimRunes(event.EventID, 128)).
		SetCorpID(trimRunes(event.CorpID, 64)).
		SetEventType(trimRunes(event.EventType, 64)).
		SetProcessInstanceID(trimRunes(event.ProcessInstanceID, 128)).
		SetEncryptedPayloadHash(event.EncryptedPayloadHash).
		SetParsedSummary("已验证审批事件；等待权威查询").
		Save(ctx)
	if err == nil {
		return nil
	}
	if !ent.IsConstraintError(err) {
		return err
	}
	existing, queryErr := client.DingTalkApprovalInboxEvent.Query().
		Where(dingtalkapprovalinboxeventent.EventIDEQ(trimRunes(event.EventID, 128))).
		Only(ctx)
	if queryErr != nil {
		return queryErr
	}
	if existing.EncryptedPayloadHash != event.EncryptedPayloadHash || existing.ProcessInstanceID != event.ProcessInstanceID {
		return fmt.Errorf("钉钉审批回调事件 ID 冲突")
	}
	return nil
}

func NewDingTalkApprovalRepo(data *Data) biz.DingTalkApprovalRepo {
	return &dingTalkApprovalRepo{data: data}
}

func (r *dingTalkApprovalRepo) PrepareDispatch(ctx context.Context, claimed *biz.BackgroundTask) (*biz.DingTalkApprovalDispatch, error) {
	if claimed == nil || claimed.ID == uuid.Nil || claimed.LeaseToken == nil {
		return nil, biz.ErrBackgroundTaskLeaseMismatch
	}
	var result *biz.DingTalkApprovalDispatch
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		task, err := tx.BackgroundTask.Query().
			Where(backgroundtaskent.IDEQ(claimed.ID), backgroundtaskent.OrganizationIDEQ(claimed.OrganizationID)).
			ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrBackgroundTaskNotFound, nil)
		}
		if err := ensureApprovalTaskLease(task, *claimed.LeaseToken, time.Now()); err != nil {
			return err
		}
		dispatch, err := tx.DingTalkApprovalDispatch.Query().
			Where(dingtalkapprovaldispatchent.BackgroundTaskIDEQ(task.ID)).
			WithUnlockRequest().
			ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		request := dispatch.Edges.UnlockRequest
		if request == nil {
			return fmt.Errorf("钉钉审批派发缺少解锁请求")
		}
		shouldSend := dispatch.DispatchStatus == dingtalkapprovaldispatchent.DispatchStatusPENDING
		if shouldSend {
			dispatch, err = tx.DingTalkApprovalDispatch.UpdateOne(dispatch).
				SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusSENDING).
				ClearErrorCategory().
				ClearResponseDigest().
				Save(ctx)
			if err != nil {
				return err
			}
		}
		processInstanceID := ""
		if dispatch.ProcessInstanceID != nil {
			processInstanceID = *dispatch.ProcessInstanceID
		}
		errorCategory := ""
		if dispatch.ErrorCategory != nil {
			errorCategory = *dispatch.ErrorCategory
		}
		result = &biz.DingTalkApprovalDispatch{
			TaskID:            task.ID,
			OrganizationID:    task.OrganizationID,
			UnlockRequestID:   request.ID,
			ProcessCode:       dispatch.ProcessCodeSnapshot,
			ApplicantUserID:   dispatch.ApplicantDingtalkUserid,
			ApproverUserIDs:   append([]string(nil), dispatch.CandidateDingtalkUserids...),
			OrderNo:           request.OrderNo,
			Reason:            request.Reason,
			DispatchStatus:    string(dispatch.DispatchStatus),
			ShouldSend:        shouldSend,
			ProcessInstanceID: processInstanceID,
			ErrorCategory:     errorCategory,
		}
		return nil
	})
	return result, err
}

func (r *dingTalkApprovalRepo) FinishDispatch(ctx context.Context, claimed *biz.BackgroundTask, outcome *biz.DingTalkApprovalDispatchOutcome, now time.Time) error {
	if claimed == nil || claimed.ID == uuid.Nil || claimed.LeaseToken == nil || outcome == nil {
		return biz.ErrBackgroundTaskInvalidArgument
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		task, err := tx.BackgroundTask.Query().
			Where(backgroundtaskent.IDEQ(claimed.ID), backgroundtaskent.OrganizationIDEQ(claimed.OrganizationID)).
			ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrBackgroundTaskNotFound, nil)
		}
		if err := ensureApprovalTaskLease(task, *claimed.LeaseToken, now); err != nil {
			return err
		}
		dispatch, err := tx.DingTalkApprovalDispatch.Query().
			Where(dingtalkapprovaldispatchent.BackgroundTaskIDEQ(task.ID)).
			WithUnlockRequest().
			ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		request := dispatch.Edges.UnlockRequest
		if request == nil {
			return fmt.Errorf("钉钉审批派发缺少解锁请求")
		}
		request, err = tx.OrderUnlockRequest.Query().Where(orderunlockrequestent.IDEQ(request.ID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}

		responseDigest := trimRunes(outcome.ResponseDigest, 500)
		errorCode := trimRunes(outcome.ErrorCode, 64)
		errorMessage := trimRunes(outcome.ErrorMessage, 500)
		dispatchUpdate := tx.DingTalkApprovalDispatch.UpdateOne(dispatch)
		requestUpdate := tx.OrderUnlockRequest.UpdateOne(request)

		switch outcome.FailureKind {
		case "":
			instanceID := strings.TrimSpace(outcome.ProcessInstanceID)
			if instanceID == "" {
				return fmt.Errorf("钉钉审批创建成功结果缺少实例 ID")
			}
			dispatchUpdate.SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusDISPATCHED).
				SetProcessInstanceID(instanceID).
				SetNillableResponseDigest(optionalApprovalString(responseDigest)).
				ClearErrorCategory()
			requestUpdate.SetDingtalkProcessInstanceID(instanceID).
				SetDingtalkProcessCode(dispatch.ProcessCodeSnapshot).
				ClearFailureCode().ClearFailureMessage()
			if request.Status == orderunlockrequestent.StatusPENDING_DISPATCH {
				requestUpdate.SetStatus(orderunlockrequestent.StatusPENDING_APPROVAL)
			}
			if _, err := requestUpdate.Save(ctx); err != nil {
				return err
			}
			if _, err := dispatchUpdate.Save(ctx); err != nil {
				return err
			}
			return completeApprovalTask(ctx, tx, task)

		case biz.DingTalkApprovalDispatchFailureRetryable:
			nextAttempts := task.Attempts + 1
			if nextAttempts >= task.MaxAttempts {
				dispatchUpdate.SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusFAILED)
				if request.Status == orderunlockrequestent.StatusPENDING_DISPATCH {
					requestUpdate.SetStatus(orderunlockrequestent.StatusDISPATCH_FAILED)
				}
			} else {
				dispatchUpdate.SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusPENDING)
			}
			dispatchUpdate.SetNillableResponseDigest(optionalApprovalString(responseDigest)).SetNillableErrorCategory(optionalApprovalString(errorCode))
			requestUpdate.SetNillableFailureCode(optionalApprovalString(errorCode)).SetNillableFailureMessage(optionalApprovalString(errorMessage))
			if _, err := requestUpdate.Save(ctx); err != nil {
				return err
			}
			if _, err := dispatchUpdate.Save(ctx); err != nil {
				return err
			}
			return failApprovalTask(ctx, tx, task, errorMessage, nextAttempts, now)

		case biz.DingTalkApprovalDispatchFailureRejected:
			dispatchUpdate.SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusFAILED).
				SetNillableResponseDigest(optionalApprovalString(responseDigest)).SetNillableErrorCategory(optionalApprovalString(errorCode))
			if request.Status == orderunlockrequestent.StatusPENDING_DISPATCH {
				requestUpdate.SetStatus(orderunlockrequestent.StatusDISPATCH_FAILED)
			}
			requestUpdate.SetNillableFailureCode(optionalApprovalString(errorCode)).SetNillableFailureMessage(optionalApprovalString(errorMessage))

		case biz.DingTalkApprovalDispatchFailureUnknown:
			dispatchUpdate.SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusUNKNOWN).
				SetNillableResponseDigest(optionalApprovalString(responseDigest)).SetNillableErrorCategory(optionalApprovalString(errorCode))
			if request.Status == orderunlockrequestent.StatusPENDING_DISPATCH {
				requestUpdate.SetStatus(orderunlockrequestent.StatusDISPATCH_UNKNOWN)
			}
			requestUpdate.SetNillableFailureCode(optionalApprovalString(errorCode)).SetNillableFailureMessage(optionalApprovalString(errorMessage))

		default:
			return fmt.Errorf("未知钉钉审批派发失败分类 %q", outcome.FailureKind)
		}
		if _, err := requestUpdate.Save(ctx); err != nil {
			return err
		}
		if _, err := dispatchUpdate.Save(ctx); err != nil {
			return err
		}
		return completeApprovalTask(ctx, tx, task)
	})
}

func ensureApprovalTaskLease(task *ent.BackgroundTask, leaseToken string, now time.Time) error {
	if task == nil || task.Status != backgroundtaskent.StatusRUNNING || task.LeaseToken == nil || *task.LeaseToken != leaseToken ||
		(task.LeaseExpiresAt != nil && task.LeaseExpiresAt.Before(now)) {
		return biz.ErrBackgroundTaskLeaseMismatch
	}
	return nil
}

func completeApprovalTask(ctx context.Context, tx *ent.Tx, task *ent.BackgroundTask) error {
	_, err := tx.BackgroundTask.UpdateOne(task).
		SetStatus(backgroundtaskent.StatusSUCCEEDED).
		ClearLeaseToken().ClearLeaseExpiresAt().ClearLastError().
		Save(ctx)
	return err
}

func failApprovalTask(ctx context.Context, tx *ent.Tx, task *ent.BackgroundTask, message string, attempts int, now time.Time) error {
	update := tx.BackgroundTask.UpdateOne(task).
		SetAttempts(attempts).
		SetLastError(trimRunes(message, 2000)).
		ClearLeaseToken().ClearLeaseExpiresAt()
	if attempts >= task.MaxAttempts {
		update.SetStatus(backgroundtaskent.StatusDEAD_LETTER)
	} else {
		backoff := 30 * time.Second * time.Duration(1<<min(attempts, 5))
		update.SetStatus(backgroundtaskent.StatusFAILED).SetNextRunAt(now.Add(backoff))
	}
	_, err := update.Save(ctx)
	return err
}

func trimRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	return string([]rune(value)[:max])
}

func optionalApprovalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

var _ biz.DingTalkApprovalRepo = (*dingTalkApprovalRepo)(nil)
