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
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

func setupTestOrderAbnormalCaseRepo(t *testing.T) (biz.OrderAbnormalCaseRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))
	repo := NewOrderAbnormalCaseRepo(&Data{db: client, sqlDB: db})
	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func TestOrderAbnormalCaseRepo_Mark_UniqueConstraintMapping(t *testing.T) {
	repo, mock, cleanup := setupTestOrderAbnormalCaseRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	actorID := uuid.New()
	caseID := uuid.New()
	newID := uuid.New()

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))
	mock.ExpectQuery(`SELECT "organizations"\."id"`).
		WithArgs(orgID).
		WillReturnRows(headquartersOrganizationRows(orgID))

	mock.ExpectQuery(`SELECT COUNT\("master_data_items"\."id"\) FROM "master_data_items"`).
		WithArgs(caseID, orgID, "abnormal_case").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	// Query existing case
	mock.ExpectQuery(`SELECT "order_abnormal_cases"\."id".*FOR UPDATE`).
		WithArgs(orderID, caseID).
		WillReturnRows(sqlmock.NewRows(orderabnormalcaseent.Columns))

	// Insert causes duplicate key
	mock.ExpectExec(`INSERT INTO "order_abnormal_cases"`).
		WillReturnError(errors.New(`pq: duplicate key value violates unique constraint "orderabnormalcase_order_id_abnormal_case_id"`))

	mock.ExpectRollback()

	_, err := repo.Mark(context.Background(), orgID, orderID, actorID, caseID, newID, nil)
	if !errors.Is(err, biz.ErrOrderAbnormalCaseExists) {
		t.Fatalf("expected ErrOrderAbnormalCaseExists, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOrderAbnormalCaseRepo_Mark_AuditErrorRollsBack(t *testing.T) {
	repo, mock, cleanup := setupTestOrderAbnormalCaseRepo(t)
	defer cleanup()

	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	caseID := uuid.New()
	newID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.abnormal_case.mark",
		Result:         "success",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).WithArgs(orderID, orgID).WillReturnRows(orderRows(orderID, orgID))
	mock.ExpectQuery(`SELECT "organizations"\."id"`).WithArgs(orgID).WillReturnRows(headquartersOrganizationRows(orgID))
	mock.ExpectQuery(`SELECT COUNT\("master_data_items"\."id"\) FROM "master_data_items"`).
		WithArgs(caseID, orgID, "abnormal_case").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id".*FOR UPDATE`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))
	mock.ExpectQuery(`SELECT "order_abnormal_cases"\."id".*FOR UPDATE`).
		WithArgs(orderID, caseID).
		WillReturnRows(sqlmock.NewRows(orderabnormalcaseent.Columns))
	mock.ExpectExec(`INSERT INTO "order_abnormal_cases"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	result, err := repo.Mark(context.Background(), orgID, orderID, actorID, caseID, newID, audit)
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if result != nil {
		t.Fatalf("审计写入失败时不应返回异常标记: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func headquartersOrganizationRows(id uuid.UUID) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows(organizationent.Columns).
		AddRow(id, now, now, "HQ", "总部", organizationent.KindHeadquarters, nil, true, "CNY", "ZONGBU ZB")
}
