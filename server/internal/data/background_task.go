package data

import (
	"context"
	"strconv"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
)

type backgroundTaskRepo struct {
	data *Data
}

func NewBackgroundTaskRepo(data *Data) biz.BackgroundTaskRepo {
	return &backgroundTaskRepo{data: data}
}

func (r *backgroundTaskRepo) Enqueue(ctx context.Context, organizationID uuid.UUID, input *biz.BackgroundTask) (*biz.BackgroundTask, error) {
	create := r.data.db.BackgroundTask.Create().
		SetOrganizationID(organizationID).
		SetKind(backgroundtaskent.Kind(input.Kind)).
		SetIdempotencyKey(input.IdempotencyKey).
		SetStatus(backgroundtaskent.StatusPENDING).
		SetAttempts(0).
		SetMaxAttempts(input.MaxAttempts).
		SetNextRunAt(input.NextRunAt)

	created, err := create.Save(ctx)
	if err != nil {
		return nil, mapEntError(err, nil, biz.ErrBackgroundTaskExists)
	}
	return backgroundTaskToBiz(created), nil
}

func (r *backgroundTaskRepo) Claim(ctx context.Context, organizationID uuid.UUID, kinds []biz.BackgroundTaskKind, leaseDuration time.Duration, now time.Time) (*biz.BackgroundTask, error) {
	return r.claim(ctx, &organizationID, kinds, leaseDuration, now)
}

func (r *backgroundTaskRepo) ClaimAny(ctx context.Context, kinds []biz.BackgroundTaskKind, leaseDuration time.Duration, now time.Time) (*biz.BackgroundTask, error) {
	return r.claim(ctx, nil, kinds, leaseDuration, now)
}

func (r *backgroundTaskRepo) claim(ctx context.Context, organizationID *uuid.UUID, kinds []biz.BackgroundTaskKind, leaseDuration time.Duration, now time.Time) (*biz.BackgroundTask, error) {
	if now.IsZero() {
		now = time.Now()
	}
	runnablePred := backgroundtaskent.Or(
		backgroundtaskent.And(
			backgroundtaskent.StatusIn(backgroundtaskent.StatusPENDING, backgroundtaskent.StatusFAILED),
			backgroundtaskent.NextRunAtLTE(now),
		),
		backgroundtaskent.And(
			backgroundtaskent.StatusEQ(backgroundtaskent.StatusRUNNING),
			backgroundtaskent.LeaseExpiresAtLTE(now),
		),
	)

	var updated *ent.BackgroundTask
	committedNoTask := false
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		hasDeadLettered := false
		for {
			query := tx.BackgroundTask.Query().Where(runnablePred)
			if organizationID != nil {
				query = query.Where(backgroundtaskent.OrganizationIDEQ(*organizationID))
			}

			if len(kinds) > 0 {
				entKinds := make([]backgroundtaskent.Kind, len(kinds))
				for i, k := range kinds {
					entKinds[i] = backgroundtaskent.Kind(k)
				}
				query = query.Where(backgroundtaskent.KindIn(entKinds...))
			}

			task, queryErr := query.
				Order(backgroundtaskent.ByNextRunAt(), backgroundtaskent.ByCreatedAt()).
				ForUpdate().
				First(ctx)
			if queryErr != nil {
				if ent.IsNotFound(queryErr) {
					if hasDeadLettered {
						committedNoTask = true
						return nil
					}
					return biz.ErrBackgroundTaskNoTask
				}
				return queryErr
			}

			leaseToken := uuid.New().String()
			leaseExpiresAt := now.Add(leaseDuration)

			if task.Status == backgroundtaskent.StatusRUNNING {
				nextAttempts := task.Attempts + 1
				if nextAttempts >= task.MaxAttempts {
					_, updateErr := tx.BackgroundTask.UpdateOne(task).
						SetStatus(backgroundtaskent.StatusDEAD_LETTER).
						SetAttempts(nextAttempts).
						ClearLeaseToken().
						ClearLeaseExpiresAt().
						Save(ctx)
					if updateErr != nil {
						return updateErr
					}
					hasDeadLettered = true
					continue
				}

				var updateErr error
				updated, updateErr = tx.BackgroundTask.UpdateOne(task).
					SetStatus(backgroundtaskent.StatusRUNNING).
					SetAttempts(nextAttempts).
					SetLeaseToken(leaseToken).
					SetLeaseExpiresAt(leaseExpiresAt).
					Save(ctx)
				if updateErr != nil {
					return updateErr
				}
				return nil
			}

			var updateErr error
			updated, updateErr = tx.BackgroundTask.UpdateOne(task).
				SetStatus(backgroundtaskent.StatusRUNNING).
				SetLeaseToken(leaseToken).
				SetLeaseExpiresAt(leaseExpiresAt).
				Save(ctx)
			if updateErr != nil {
				return updateErr
			}
			return nil
		}
	})
	if err != nil {
		return nil, err
	}
	if committedNoTask {
		return nil, biz.ErrBackgroundTaskNoTask
	}
	return backgroundTaskToBiz(updated), nil
}

func (r *backgroundTaskRepo) Complete(ctx context.Context, organizationID, id uuid.UUID, leaseToken string) (*biz.BackgroundTask, error) {
	var updated *ent.BackgroundTask
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		task, queryErr := tx.BackgroundTask.Query().
			Where(
				backgroundtaskent.IDEQ(id),
				backgroundtaskent.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrBackgroundTaskNotFound, nil)
		}
		if task.Status != backgroundtaskent.StatusRUNNING {
			return biz.ErrBackgroundTaskInvalidStatus
		}
		if task.LeaseToken == nil || *task.LeaseToken != leaseToken {
			return biz.ErrBackgroundTaskLeaseMismatch
		}
		if task.LeaseExpiresAt != nil && task.LeaseExpiresAt.Before(time.Now()) {
			return biz.ErrBackgroundTaskLeaseMismatch
		}
		var updateErr error
		updated, updateErr = tx.BackgroundTask.UpdateOne(task).
			SetStatus(backgroundtaskent.StatusSUCCEEDED).
			ClearLeaseToken().
			ClearLeaseExpiresAt().
			Save(ctx)
		return updateErr
	})
	if err != nil {
		return nil, err
	}
	return backgroundTaskToBiz(updated), nil
}

func (r *backgroundTaskRepo) Fail(ctx context.Context, organizationID, id uuid.UUID, leaseToken, errorMessage string, nextRunAt time.Time) (*biz.BackgroundTask, error) {
	var updated *ent.BackgroundTask
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		task, queryErr := tx.BackgroundTask.Query().
			Where(
				backgroundtaskent.IDEQ(id),
				backgroundtaskent.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrBackgroundTaskNotFound, nil)
		}
		if task.Status != backgroundtaskent.StatusRUNNING {
			return biz.ErrBackgroundTaskInvalidStatus
		}
		if task.LeaseToken == nil || *task.LeaseToken != leaseToken {
			return biz.ErrBackgroundTaskLeaseMismatch
		}
		if task.LeaseExpiresAt != nil && task.LeaseExpiresAt.Before(time.Now()) {
			return biz.ErrBackgroundTaskLeaseMismatch
		}
		nextAttempts := task.Attempts + 1
		update := tx.BackgroundTask.UpdateOne(task).
			SetAttempts(nextAttempts).
			ClearLeaseToken().
			ClearLeaseExpiresAt()
		if errorMessage != "" {
			update.SetLastError(errorMessage)
		} else {
			update.ClearLastError()
		}
		if nextAttempts >= task.MaxAttempts {
			update.SetStatus(backgroundtaskent.StatusDEAD_LETTER)
		} else {
			update.SetStatus(backgroundtaskent.StatusFAILED).
				SetNextRunAt(nextRunAt)
		}
		var updateErr error
		updated, updateErr = update.Save(ctx)
		return updateErr
	})
	if err != nil {
		return nil, err
	}
	return backgroundTaskToBiz(updated), nil
}

func (r *backgroundTaskRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.BackgroundTask, error) {
	task, err := r.data.db.BackgroundTask.Query().
		Where(
			backgroundtaskent.IDEQ(id),
			backgroundtaskent.OrganizationIDEQ(organizationID),
		).
		WithNotificationDelivery(func(query *ent.NotificationDeliveryQuery) {
			query.WithRecipientUser()
		}).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrBackgroundTaskNotFound, nil)
	}
	return backgroundTaskToBiz(task), nil
}

func (r *backgroundTaskRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.BackgroundTaskListOptions) (*biz.BackgroundTaskList, error) {
	query := r.data.db.BackgroundTask.Query().Where(backgroundtaskent.OrganizationIDEQ(organizationID))
	if options.Status != nil {
		query.Where(backgroundtaskent.StatusEQ(backgroundtaskent.Status(*options.Status)))
	}
	if options.Phase != nil {
		statuses := options.Phase.Statuses()
		entStatuses := make([]backgroundtaskent.Status, len(statuses))
		for index, status := range statuses {
			entStatuses[index] = backgroundtaskent.Status(status)
		}
		query.Where(backgroundtaskent.StatusIn(entStatuses...))
	}
	if options.Kind != nil {
		query.Where(backgroundtaskent.KindEQ(backgroundtaskent.Kind(*options.Kind)))
	}
	if options.StartTime != nil {
		query.Where(backgroundtaskent.CreatedAtGTE(*options.StartTime))
	}
	if options.EndTime != nil {
		query.Where(backgroundtaskent.CreatedAtLTE(*options.EndTime))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.
		WithNotificationDelivery(func(query *ent.NotificationDeliveryQuery) {
			query.WithRecipientUser()
		}).
		Order(backgroundtaskent.ByCreatedAt(entsql.OrderDesc())).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	tasks := make([]*biz.BackgroundTask, 0, len(items))
	for _, item := range items {
		tasks = append(tasks, backgroundTaskToBiz(item))
	}
	return &biz.BackgroundTaskList{
		Items:    tasks,
		Total:    total,
		Page:     options.Page,
		PageSize: options.PageSize,
	}, nil
}

func (r *backgroundTaskRepo) Requeue(ctx context.Context, organizationID, id uuid.UUID, now time.Time, audit *biz.AuditEvent) (*biz.BackgroundTask, error) {
	var updated *ent.BackgroundTask
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		task, queryErr := tx.BackgroundTask.Query().
			Where(
				backgroundtaskent.IDEQ(id),
				backgroundtaskent.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrBackgroundTaskNotFound, nil)
		}
		if task.Status != backgroundtaskent.StatusFAILED && task.Status != backgroundtaskent.StatusDEAD_LETTER {
			return biz.ErrBackgroundTaskNotRequeueable
		}
		var updateErr error
		updated, updateErr = tx.BackgroundTask.UpdateOne(task).
			SetStatus(backgroundtaskent.StatusPENDING).
			SetAttempts(0).
			SetNextRunAt(now).
			ClearLeaseToken().
			ClearLeaseExpiresAt().
			ClearLastError().
			Save(ctx)
		if updateErr != nil {
			return updateErr
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["background_task.id"] = updated.ID.String()
		audit.Details["background_task.kind"] = string(updated.Kind)
		audit.Details["background_task.status"] = string(updated.Status)
		audit.Details["background_task.attempts"] = strconv.Itoa(updated.Attempts)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return backgroundTaskToBiz(updated), nil
}

func backgroundTaskToBiz(item *ent.BackgroundTask) *biz.BackgroundTask {
	if item == nil {
		return nil
	}
	var recipientDisplayName *string
	var recipientUserID *uuid.UUID
	if delivery := item.Edges.NotificationDelivery; delivery != nil && delivery.Edges.RecipientUser != nil {
		name := delivery.Edges.RecipientUser.DisplayName
		userID := delivery.Edges.RecipientUser.ID
		recipientDisplayName = &name
		recipientUserID = &userID
	}
	return &biz.BackgroundTask{
		ID:                   item.ID,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
		OrganizationID:       item.OrganizationID,
		Kind:                 biz.BackgroundTaskKind(item.Kind),
		IdempotencyKey:       item.IdempotencyKey,
		Status:               biz.BackgroundTaskStatus(item.Status),
		Attempts:             item.Attempts,
		MaxAttempts:          item.MaxAttempts,
		NextRunAt:            item.NextRunAt,
		LeaseToken:           item.LeaseToken,
		LeaseExpiresAt:       item.LeaseExpiresAt,
		LastError:            item.LastError,
		RecipientDisplayName: recipientDisplayName,
		RecipientUserID:      recipientUserID,
	}
}

var _ biz.BackgroundTaskRepo = (*backgroundTaskRepo)(nil)
