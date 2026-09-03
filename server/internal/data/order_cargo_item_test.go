package data

import (
	"context"
	"errors"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestOrderCargoItemRepo_Add_AuditErrorRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	defer client.Close()

	repo := NewOrderCargoItemRepo(&Data{db: client, sqlDB: db})
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	input := &biz.OrderCargoItem{
		ID:            uuid.New(),
		CargoName:     "测试货物",
		PackageCount:  10,
		GrossWeightKg: 1000,
		VolumeCbm:     20,
	}
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.cargo_item.add",
		Result:         "success",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "orders"\."id"`).WithArgs(orderID, orgID).WillReturnRows(orderRows(orderID, orgID))
	mock.ExpectQuery(`SELECT "sea_master_bill_order_links"\."id"`).WithArgs(orgID, orderID, "ACTIVE").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec(`INSERT INTO "order_cargo_items"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	result, err := repo.Add(context.Background(), orgID, orderID, input, audit)
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if result != nil {
		t.Fatalf("审计写入失败时不应返回货物明细: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
