package data

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestOrderRepoFindReferenceDuplicateUsesCustomerScopeAndExclusion(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	defer func() {
		_ = client.Close()
		_ = db.Close()
	}()

	organizationID := uuid.New()
	customerID := uuid.New()
	excludeOrderID := uuid.New()
	duplicateOrderID := uuid.New()
	repo := NewOrderRepo(&Data{db: client, sqlDB: db})

	mock.ExpectQuery(`SELECT .* FROM "orders" WHERE .*`).
		WithArgs(organizationID, customerID, "customer-001", excludeOrderID).
		WillReturnRows(orderReferenceRows(duplicateOrderID, organizationID, customerID, "SE0001"))

	match, err := repo.FindReferenceDuplicate(context.Background(), organizationID, biz.OrderReferenceCheck{
		ReferenceType:  biz.OrderReferenceCustomer,
		ReferenceNo:    "Customer-001",
		CustomerID:     &customerID,
		ExcludeOrderID: &excludeOrderID,
	})
	if err != nil {
		t.Fatalf("FindReferenceDuplicate() error = %v", err)
	}
	if match == nil || match.OrderID != duplicateOrderID || match.OrderNo != "SE0001" {
		t.Fatalf("reference match = %#v", match)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func orderReferenceRows(id, organizationID, customerID uuid.UUID, orderNo string) *sqlmock.Rows {
	now := time.Now()
	values := make([]driver.Value, len(orderent.Columns))
	for index, column := range orderent.Columns {
		switch column {
		case orderent.FieldID:
			values[index] = id
		case orderent.FieldCreatedAt, orderent.FieldUpdatedAt:
			values[index] = now
		case orderent.FieldOrganizationID:
			values[index] = organizationID
		case orderent.FieldOrderNo:
			values[index] = orderNo
		case orderent.FieldCustomerID:
			values[index] = customerID
		case orderent.FieldBusinessType:
			values[index] = "SE"
		case orderent.FieldTradeDirection:
			values[index] = "export"
		case orderent.FieldTradeTerm:
			values[index] = "FOB"
		case orderent.FieldPaymentTerm:
			values[index] = "PREPAID"
		case orderent.FieldStatus:
			values[index] = "DRAFT"
		case orderent.FieldStatusTemplateID:
			values[index] = uuid.New()
		default:
			values[index] = nil
		}
	}
	return sqlmock.NewRows(orderent.Columns).AddRow(values...)
}
