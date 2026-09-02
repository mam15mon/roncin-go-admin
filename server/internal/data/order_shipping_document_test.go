package data

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
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

func airOrderRows(id, orgID uuid.UUID) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(orderent.Columns)
	values := make([]driver.Value, len(orderent.Columns))
	for i, col := range orderent.Columns {
		switch col {
		case orderent.FieldID:
			values[i] = id
		case orderent.FieldOrganizationID:
			values[i] = orgID
		case orderent.FieldOrderNo:
			values[i] = "ORD-20260821-0001"
		case orderent.FieldBusinessType:
			values[i] = "AE"
		case orderent.FieldTradeDirection:
			values[i] = "export"
		case orderent.FieldTradeTerm:
			values[i] = "FOB"
		case orderent.FieldPaymentTerm:
			values[i] = "PREPAID"
		case orderent.FieldShipmentType:
			values[i] = "FCL"
		case orderent.FieldFlowStatus:
			values[i] = "DRAFT"
		case orderent.FieldTerminationStatus:
			values[i] = "ACTIVE"
		case orderent.FieldClosureStatus:
			values[i] = "OPEN"
		case orderent.FieldCreatedAt, orderent.FieldUpdatedAt:
			values[i] = now
		default:
			values[i] = nil
		}
	}
	rows.AddRow(values...)
	return rows
}

func shippingDocRows(id, orderID uuid.UUID, houseNo, status string) *sqlmock.Rows {
	now := time.Now()
	rows := sqlmock.NewRows(ordershippingdocumentent.Columns)
	values := make([]driver.Value, len(ordershippingdocumentent.Columns))
	for i, col := range ordershippingdocumentent.Columns {
		switch col {
		case ordershippingdocumentent.FieldID:
			values[i] = id
		case ordershippingdocumentent.FieldOrderID:
			values[i] = orderID
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

func TestOrderShippingDocumentRepo_RejectsSEOrder(t *testing.T) {
	repo, mock, cleanup := setupTestOrderShippingDocumentRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	input := &biz.OrderShippingDocument{
		ID:      uuid.New(),
		HouseNo: "HBL999999",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	_, err := repo.Add(context.Background(), orgID, orderID, input, nil)
	if !errors.Is(err, biz.ErrSeaShippingDocumentsDeprecated) {
		t.Fatalf("expected ErrSeaShippingDocumentsDeprecated for SE order, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOrderShippingDocumentRepo_Add_UniqueConstraintMapping(t *testing.T) {
	repo, mock, cleanup := setupTestOrderShippingDocumentRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	input := &biz.OrderShippingDocument{
		ID:      uuid.New(),
		HouseNo: "HBL999999",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(airOrderRows(orderID, orgID))

	mock.ExpectBegin()

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

func TestOrderShippingDocumentRepo_Remove_ReleasedStatusPrevented(t *testing.T) {
	repo, mock, cleanup := setupTestOrderShippingDocumentRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(airOrderRows(orderID, orgID))

	mock.ExpectBegin()

	// ForUpdate returns document with RELEASED status
	mock.ExpectQuery(`SELECT "order_shipping_documents"\."id".*FOR UPDATE`).
		WithArgs(docID, orderID).
		WillReturnRows(shippingDocRows(docID, orderID, "HBL999", "RELEASED"))

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
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.shipping_document.remove",
		Result:         "success",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).WithArgs(orderID, orgID).WillReturnRows(airOrderRows(orderID, orgID))
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "order_shipping_documents"\."id".*FOR UPDATE`).
		WithArgs(docID, orderID).
		WillReturnRows(shippingDocRows(docID, orderID, "HBL001", "DRAFT"))
	mock.ExpectExec(`DELETE FROM "order_shipping_documents"`).
		WithArgs(docID).
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
