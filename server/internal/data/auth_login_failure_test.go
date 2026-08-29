package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
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
