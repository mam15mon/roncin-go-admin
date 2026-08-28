package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
)

func setupTestBackgroundTaskRepo(t *testing.T) (biz.BackgroundTaskRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))
	repo := NewBackgroundTaskRepo(&Data{db: client, sqlDB: db})
	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func backgroundTaskRows(id, orgID uuid.UUID, kind, status string, attempts, maxAttempts int, nextRunAt time.Time, leaseExpiresAt *time.Time) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(backgroundtaskent.Columns)
	values := make([]driver.Value, len(backgroundtaskent.Columns))
	for i, col := range backgroundtaskent.Columns {
		switch col {
		case backgroundtaskent.FieldID:
			values[i] = id
		case backgroundtaskent.FieldOrganizationID:
			values[i] = orgID
		case backgroundtaskent.FieldKind:
			values[i] = kind
		case backgroundtaskent.FieldStatus:
			values[i] = status
		case backgroundtaskent.FieldAttempts:
			values[i] = attempts
		case backgroundtaskent.FieldMaxAttempts:
			values[i] = maxAttempts
		case backgroundtaskent.FieldNextRunAt:
			values[i] = nextRunAt
		case backgroundtaskent.FieldLeaseExpiresAt:
			values[i] = leaseExpiresAt
		case backgroundtaskent.FieldCreatedAt, backgroundtaskent.FieldUpdatedAt:
			values[i] = now
		default:
			values[i] = nil
		}
	}
	rows.AddRow(values...)
	return rows
}

func TestBackgroundTaskToBizIncludesNotificationRecipient(t *testing.T) {
	recipientDisplayName := "张冠楠"
	recipientUserID := uuid.New()
	value := backgroundTaskToBiz(&ent.BackgroundTask{
		Edges: ent.BackgroundTaskEdges{
			NotificationDelivery: &ent.NotificationDelivery{
				Edges: ent.NotificationDeliveryEdges{
					RecipientUser: &ent.User{ID: recipientUserID, DisplayName: recipientDisplayName},
				},
			},
		},
	})

	if value.RecipientDisplayName == nil || *value.RecipientDisplayName != recipientDisplayName {
		t.Fatalf("expected recipient display name %q, got %v", recipientDisplayName, value.RecipientDisplayName)
	}
	if value.RecipientUserID == nil || *value.RecipientUserID != recipientUserID {
		t.Fatalf("expected recipient user ID %q, got %v", recipientUserID, value.RecipientUserID)
	}
}

func TestBackgroundTaskRepo_Claim_ExpiredLeaseIncrementsAttempts(t *testing.T) {
	repo, mock, cleanup := setupTestBackgroundTaskRepo(t)
	defer cleanup()

	orgID := uuid.New()
	taskID := uuid.New()
	now := time.Now()
	expiredLease := now.Add(-5 * time.Minute)

	mock.ExpectBegin()

	// 1. Query runnable task
	mock.ExpectQuery(`SELECT "background_tasks"\."id".*FROM "background_tasks".*LIMIT 1 FOR UPDATE`).
		WillReturnRows(backgroundTaskRows(taskID, orgID, "order_export", "RUNNING", 1, 3, now.Add(-10*time.Minute), &expiredLease))

	// 2. Update task: status=RUNNING, attempts=2 (incremented), lease_expires_at, lease_token
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "background_tasks" SET`)).
		WithArgs(
			sqlmock.AnyArg(), // updated_at
			"RUNNING",        // status
			2,                // attempts incremented to 2
			sqlmock.AnyArg(), // lease_token
			sqlmock.AnyArg(), // lease_expires_at
			taskID,           // id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 3. Reload task
	reloadedLease := now.Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id", "created_at", "updated_at", "organization_id", "kind", "idempotency_key", "status", "attempts", "max_attempts", "next_run_at", "lease_token", "lease_expires_at", "last_error" FROM "background_tasks" WHERE "id" = $1`)).
		WithArgs(taskID).
		WillReturnRows(backgroundTaskRows(taskID, orgID, "order_export", "RUNNING", 2, 3, now.Add(-10*time.Minute), &reloadedLease))

	mock.ExpectCommit()

	claimed, err := repo.Claim(context.Background(), orgID, nil, time.Minute, now)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if claimed.Attempts != 2 {
		t.Fatalf("expected attempts 2, got %d", claimed.Attempts)
	}
	if claimed.Status != biz.BackgroundTaskStatusRunning {
		t.Fatalf("expected status RUNNING, got %v", claimed.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBackgroundTaskRepo_Claim_ExpiredLeaseDeadLetterMaxAttempts(t *testing.T) {
	repo, mock, cleanup := setupTestBackgroundTaskRepo(t)
	defer cleanup()

	orgID := uuid.New()
	task1ID := uuid.New()
	task2ID := uuid.New()
	now := time.Now()
	expiredLease := now.Add(-5 * time.Minute)

	mock.ExpectBegin()

	// 1. Query first runnable task (attempts=2, max=3 -> next is 3 >= 3 -> DEAD_LETTER)
	mock.ExpectQuery(`SELECT "background_tasks"\."id".*FROM "background_tasks".*LIMIT 1 FOR UPDATE`).
		WillReturnRows(backgroundTaskRows(task1ID, orgID, "order_export", "RUNNING", 2, 3, now.Add(-10*time.Minute), &expiredLease))

	// 2. Update task1 to DEAD_LETTER
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "background_tasks" SET`)).
		WithArgs(
			sqlmock.AnyArg(), // updated_at
			"DEAD_LETTER",    // status
			3,                // attempts incremented to 3
			task1ID,          // id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Ent reloads task1 after update
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id", "created_at", "updated_at", "organization_id", "kind", "idempotency_key", "status", "attempts", "max_attempts", "next_run_at", "lease_token", "lease_expires_at", "last_error" FROM "background_tasks" WHERE "id" = $1`)).
		WithArgs(task1ID).
		WillReturnRows(backgroundTaskRows(task1ID, orgID, "order_export", "DEAD_LETTER", 3, 3, now.Add(-10*time.Minute), nil))

	// 3. Query next runnable task (task2, PENDING)
	mock.ExpectQuery(`SELECT "background_tasks"\."id".*FROM "background_tasks".*LIMIT 1 FOR UPDATE`).
		WillReturnRows(backgroundTaskRows(task2ID, orgID, "order_export", "PENDING", 0, 3, now.Add(-time.Minute), nil))

	// 4. Update task2 to RUNNING
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "background_tasks" SET`)).
		WithArgs(
			sqlmock.AnyArg(), // updated_at
			"RUNNING",        // status
			sqlmock.AnyArg(), // lease_token
			sqlmock.AnyArg(), // lease_expires_at
			task2ID,          // id
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 5. Reload task2
	reloadedLease := now.Add(time.Minute)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT "id", "created_at", "updated_at", "organization_id", "kind", "idempotency_key", "status", "attempts", "max_attempts", "next_run_at", "lease_token", "lease_expires_at", "last_error" FROM "background_tasks" WHERE "id" = $1`)).
		WithArgs(task2ID).
		WillReturnRows(backgroundTaskRows(task2ID, orgID, "order_export", "RUNNING", 0, 3, now.Add(-time.Minute), &reloadedLease))

	mock.ExpectCommit()

	claimed, err := repo.Claim(context.Background(), orgID, nil, time.Minute, now)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if claimed.ID != task2ID {
		t.Fatalf("expected claimed task2, got %s", claimed.ID)
	}
	if claimed.Attempts != 0 {
		t.Fatalf("expected task2 attempts 0, got %d", claimed.Attempts)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBackgroundTaskRepo_Requeue_AuditErrorRollsBack(t *testing.T) {
	repo, mock, cleanup := setupTestBackgroundTaskRepo(t)
	defer cleanup()

	orgID := uuid.New()
	actorID := uuid.New()
	taskID := uuid.New()
	now := time.Now()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "background_task.requeue",
		Result:         "success",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "background_tasks"\."id".*FOR UPDATE`).
		WithArgs(taskID, orgID).
		WillReturnRows(backgroundTaskRows(taskID, orgID, "ORDER_REMINDER", "FAILED", 2, 3, now, nil))
	mock.ExpectExec(`UPDATE "background_tasks" SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT "id", "created_at".*FROM "background_tasks" WHERE "id" = \$1`).
		WithArgs(taskID).
		WillReturnRows(backgroundTaskRows(taskID, orgID, "ORDER_REMINDER", "PENDING", 0, 3, now, nil))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).
		WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	result, err := repo.Requeue(context.Background(), orgID, taskID, now, audit)
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if result != nil {
		t.Fatalf("审计写入失败时不应返回任务: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
