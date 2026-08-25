package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderconsolidationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderconsolidation"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
)

func setupTestOrderShippingDocumentRepo(t *testing.T) (biz.OrderShippingDocumentRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))
	repo := NewOrderShippingDocumentRepo(&Data{db: client, sqlDB: db})
	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func shippingDocRows(id, orderID, consolidationID uuid.UUID, houseNo, status string) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(ordershippingdocumentent.Columns)
	values := make([]driver.Value, len(ordershippingdocumentent.Columns))
	for i, col := range ordershippingdocumentent.Columns {
		switch col {
		case ordershippingdocumentent.FieldID:
			values[i] = id
		case ordershippingdocumentent.FieldOrderID:
			values[i] = orderID
		case ordershippingdocumentent.FieldConsolidationID:
			values[i] = consolidationID
		case ordershippingdocumentent.FieldHouseNo:
			values[i] = houseNo
		case ordershippingdocumentent.FieldStatus:
			values[i] = status
		case ordershippingdocumentent.FieldCreatedAt, ordershippingdocumentent.FieldUpdatedAt:
			values[i] = now
		default:
			values[i] = nil
		}
	}
	rows.AddRow(values...)
	return rows
}

func consolidationRows(id, organizationID uuid.UUID, masterNo string) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(orderconsolidationent.Columns)
	values := make([]driver.Value, len(orderconsolidationent.Columns))
	for i, col := range orderconsolidationent.Columns {
		switch col {
		case orderconsolidationent.FieldID:
			values[i] = id
		case orderconsolidationent.FieldOrganizationID:
			values[i] = organizationID
		case orderconsolidationent.FieldBusinessType:
			values[i] = "SE"
		case orderconsolidationent.FieldMasterNo:
			values[i] = masterNo
		case orderconsolidationent.FieldNormalizedMasterNo:
			values[i] = strings.ToLower(masterNo)
		case orderconsolidationent.FieldCreatedAt, orderconsolidationent.FieldUpdatedAt:
			values[i] = now
		}
	}
	rows.AddRow(values...)
	return rows
}

func TestOrderShippingDocumentRepo_Add_UniqueConstraintMapping(t *testing.T) {
	repo, mock, cleanup := setupTestOrderShippingDocumentRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	consolidationID := uuid.New()
	input := &biz.OrderShippingDocument{
		ID:       uuid.New(),
		MasterNo: "MBL999999",
		HouseNo:  "HBL999999",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "order_consolidations"\."id"`).
		WithArgs(orgID, "SE", "mbl999999").
		WillReturnRows(consolidationRows(consolidationID, orgID, "MBL999999"))

	// Unique constraint error with Postgres constraint name
	mock.ExpectExec(`INSERT INTO "order_shipping_documents"`).
		WillReturnError(errors.New(`pq: duplicate key value violates unique constraint "ordershippingdocument_order_id_house_no"`))

	mock.ExpectRollback()

	_, err := repo.Add(context.Background(), orgID, orderID, input, nil)
	if !errors.Is(err, biz.ErrOrderShippingDocumentExists) {
		t.Fatalf("expected ErrOrderShippingDocumentExists, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestResolveOrderConsolidationReusesSharedBatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	defer client.Close()

	organizationID := uuid.New()
	consolidationID := uuid.New()
	mock.ExpectBegin()
	tx, err := client.Tx(context.Background())
	if err != nil {
		t.Fatalf("开启事务失败: %v", err)
	}
	for range 2 {
		mock.ExpectQuery(`SELECT "order_consolidations"\."id"`).
			WithArgs(organizationID, "SE", "mbl-001").
			WillReturnRows(consolidationRows(consolidationID, organizationID, "MBL-001"))
		batch, resolveErr := resolveOrderConsolidation(context.Background(), tx, organizationID, biz.OrderBusinessSE, &biz.OrderShippingDocument{MasterNo: "MBL-001"})
		if resolveErr != nil {
			t.Fatalf("解析拼载批次失败: %v", resolveErr)
		}
		if batch.ID != consolidationID {
			t.Fatalf("拼载批次 ID = %s，期望 %s", batch.ID, consolidationID)
		}
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("回滚事务失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestOrderShippingDocumentRepo_Remove_ReleasedStatusPrevented(t *testing.T) {
	repo, mock, cleanup := setupTestOrderShippingDocumentRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	docID := uuid.New()
	consolidationID := uuid.New()

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	mock.ExpectBegin()

	// ForUpdate returns document with RELEASED status
	mock.ExpectQuery(`SELECT "order_shipping_documents"\."id".*FOR UPDATE`).
		WithArgs(docID, orderID).
		WillReturnRows(shippingDocRows(docID, orderID, consolidationID, "HBL999", "RELEASED"))

	mock.ExpectRollback()

	err := repo.Remove(context.Background(), orgID, orderID, docID, nil)
	if !errors.Is(err, biz.ErrOrderShippingDocumentInvalidStatus) {
		t.Fatalf("expected ErrOrderShippingDocumentInvalidStatus, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOrderShippingDocumentRepo_Remove_AuditErrorRollsBack(t *testing.T) {
	repo, mock, cleanup := setupTestOrderShippingDocumentRepo(t)
	defer cleanup()

	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	docID := uuid.New()
	consolidationID := uuid.New()
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.shipping_document.remove",
		Result:         "success",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).WithArgs(orderID, orgID).WillReturnRows(orderRows(orderID, orgID))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "order_shipping_documents"\."id".*FOR UPDATE`).
		WithArgs(docID, orderID).
		WillReturnRows(shippingDocRows(docID, orderID, consolidationID, "HBL001", "DRAFT"))
	mock.ExpectExec(`DELETE FROM "order_shipping_documents"`).
		WithArgs(docID, orderID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	err := repo.Remove(context.Background(), orgID, orderID, docID, audit)
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
