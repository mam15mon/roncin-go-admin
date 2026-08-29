package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestAuthRepoRecordLoginFailureAuditErrorRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	defer db.Close()

	repo := NewAuthRepo(&Data{sqlDB: db})
	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "login_rate_limit_buckets"`).
		WillReturnRows(sqlmock.NewRows([]string{"attempts"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).
		WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	exceeded, err := repo.RecordLoginFailure(context.Background(), []string{"account-key"}, now, time.Minute, 5, &biz.AuditEvent{
		Action: "auth.login",
		Result: "failure",
	})
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if exceeded {
		t.Fatal("事务失败时不应返回已超过限流")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestAuthRepoCreateSessionAuditErrorRollsBackFailureClear(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})

	repo := NewAuthRepo(&Data{db: client, sqlDB: db})
	userID, organizationID := uuid.New(), uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "login_rate_limit_buckets"`).WithArgs("account-key").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "sessions"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	repoErr := repo.CreateSession(context.Background(), &biz.Session{
		TokenHash:      "session-token-hash",
		UserID:         userID,
		OrganizationID: organizationID,
		ExpiresAt:      time.Now().Add(time.Hour),
		UserAgent:      "test",
	}, "account-key", &biz.AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "auth.login",
		Result:         "success",
	})
	if repoErr == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v；仓储错误: %v", err, repoErr)
	}
}
