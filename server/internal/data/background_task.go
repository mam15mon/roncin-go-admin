package data

import (
	"context"
	"time"

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
		if ent.IsConstraintError(err) {
			return nil, biz.ErrBackgroundTaskExists
		}
		return nil, err
	}
	return backgroundTaskToBiz(created), nil
}

func (r *backgroundTaskRepo) Claim(ctx context.Context, organizationID uuid.UUID, kinds []biz.BackgroundTaskKind, leaseDuration time.Duration, now time.Time) (*biz.BackgroundTask, error) {
	if now.IsZero() {
		now = time.Now()
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
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

	query := tx.BackgroundTask.Query().
		Where(
			backgroundtaskent.OrganizationIDEQ(organizationID),
			runnablePred,
		)

	if len(kinds) > 0 {
		entKinds := make([]backgroundtaskent.Kind, len(kinds))
		for i, k := range kinds {
			entKinds[i] = backgroundtaskent.Kind(k)
		}
		query = query.Where(backgroundtaskent.KindIn(entKinds...))
	}

	task, err := query.
		Order(backgroundtaskent.ByNextRunAt(), backgroundtaskent.ByCreatedAt()).
		ForUpdate().
		First(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrBackgroundTaskNoTask
		}
		return nil, err
	}

	leaseToken := uuid.New().String()
	leaseExpiresAt := now.Add(leaseDuration)

	updated, err := tx.BackgroundTask.UpdateOne(task).
		SetStatus(backgroundtaskent.StatusRUNNING).
		SetLeaseToken(leaseToken).
		SetLeaseExpiresAt(leaseExpiresAt).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return backgroundTaskToBiz(updated), nil
}

func (r *backgroundTaskRepo) Complete(ctx context.Context, organizationID, id uuid.UUID, leaseToken string) (*biz.BackgroundTask, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	task, err := tx.BackgroundTask.Query().
		Where(
			backgroundtaskent.IDEQ(id),
			backgroundtaskent.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrBackgroundTaskNotFound
		}
		return nil, err
	}

	if task.Status != backgroundtaskent.StatusRUNNING {
		_ = tx.Rollback()
		return nil, biz.ErrBackgroundTaskInvalidStatus
	}
	if task.LeaseToken == nil || *task.LeaseToken != leaseToken {
		_ = tx.Rollback()
		return nil, biz.ErrBackgroundTaskLeaseMismatch
	}
	if task.LeaseExpiresAt != nil && task.LeaseExpiresAt.Before(time.Now()) {
		_ = tx.Rollback()
		return nil, biz.ErrBackgroundTaskLeaseMismatch
	}

	updated, err := tx.BackgroundTask.UpdateOne(task).
		SetStatus(backgroundtaskent.StatusSUCCEEDED).
		ClearLeaseToken().
		ClearLeaseExpiresAt().
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return backgroundTaskToBiz(updated), nil
}

func (r *backgroundTaskRepo) Fail(ctx context.Context, organizationID, id uuid.UUID, leaseToken, errorMessage string, nextRunAt time.Time) (*biz.BackgroundTask, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}

	task, err := tx.BackgroundTask.Query().
		Where(
			backgroundtaskent.IDEQ(id),
			backgroundtaskent.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrBackgroundTaskNotFound
		}
		return nil, err
	}

	if task.Status != backgroundtaskent.StatusRUNNING {
		_ = tx.Rollback()
		return nil, biz.ErrBackgroundTaskInvalidStatus
	}
	if task.LeaseToken == nil || *task.LeaseToken != leaseToken {
		_ = tx.Rollback()
		return nil, biz.ErrBackgroundTaskLeaseMismatch
	}
	if task.LeaseExpiresAt != nil && task.LeaseExpiresAt.Before(time.Now()) {
		_ = tx.Rollback()
		return nil, biz.ErrBackgroundTaskLeaseMismatch
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

	updated, err := update.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrBackgroundTaskNotFound
		}
		return nil, err
	}
	return backgroundTaskToBiz(task), nil
}

func backgroundTaskToBiz(item *ent.BackgroundTask) *biz.BackgroundTask {
	if item == nil {
		return nil
	}
	return &biz.BackgroundTask{
		ID:             item.ID,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		OrganizationID: item.OrganizationID,
		Kind:           biz.BackgroundTaskKind(item.Kind),
		IdempotencyKey: item.IdempotencyKey,
		Status:         biz.BackgroundTaskStatus(item.Status),
		Attempts:       item.Attempts,
		MaxAttempts:    item.MaxAttempts,
		NextRunAt:      item.NextRunAt,
		LeaseToken:     item.LeaseToken,
		LeaseExpiresAt: item.LeaseExpiresAt,
		LastError:      item.LastError,
	}
}

var _ biz.BackgroundTaskRepo = (*backgroundTaskRepo)(nil)
