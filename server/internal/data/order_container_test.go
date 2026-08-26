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

func setupTestOrderContainerRepo(t *testing.T) (biz.OrderContainerRepo, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))
	repo := NewOrderContainerRepo(&Data{db: client, sqlDB: db})
	cleanup := func() {
		_ = client.Close()
		_ = db.Close()
	}
	return repo, mock, cleanup
}

func TestOrderContainerRepo_Add_UniqueConstraintMapping(t *testing.T) {
	repo, mock, cleanup := setupTestOrderContainerRepo(t)
	defer cleanup()

	orgID := uuid.New()
	orderID := uuid.New()
	seal := "SEAL001"
	input := &biz.OrderContainer{
		ID:            uuid.New(),
		ContainerNo:   "MSCU9999999",
		SealNo:        &seal,
		GrossWeightKg: 1000.0,
		VolumeCbm:     20.0,
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).
		WithArgs(orderID, orgID).
		WillReturnRows(orderRows(orderID, orgID))

	mock.ExpectQuery(`SELECT COUNT\("master_data_items"\."id"\) FROM "master_data_items"`).
		WithArgs(input.ContainerSpecID, orgID, "container_spec").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectBegin()

	// Unique constraint error with Postgres constraint name
	mock.ExpectExec(`INSERT INTO "order_containers"`).
		WillReturnError(errors.New(`pq: duplicate key value violates unique constraint "ordercontainer_order_id_container_no"`))

	mock.ExpectRollback()

	_, err := repo.Add(context.Background(), orgID, orderID, input, nil)
	if !errors.Is(err, biz.ErrOrderContainerExists) {
		t.Fatalf("expected ErrOrderContainerExists, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOrderContainerRepo_Add_RejectsNonFCLOrder(t *testing.T) {
	for _, shipmentType := range []string{"LCL", "BREAK_BULK"} {
		t.Run(shipmentType, func(t *testing.T) {
			repo, mock, cleanup := setupTestOrderContainerRepo(t)
			defer cleanup()

			orgID := uuid.New()
			orderID := uuid.New()
			input := &biz.OrderContainer{
				ID: uuid.New(), ContainerNo: "MSCU1234567", ContainerSpecID: uuid.New(),
				GrossWeightKg: 1000, VolumeCbm: 20,
			}
			mock.ExpectQuery(`SELECT "orders"\."id"`).
				WithArgs(orderID, orgID).
				WillReturnRows(orderRowsWithShipmentType(orderID, orgID, shipmentType))

			_, err := repo.Add(context.Background(), orgID, orderID, input, nil)
			if !errors.Is(err, biz.ErrOrderContainerShipmentType) {
				t.Fatalf("shipmentType=%s error = %v, want ErrOrderContainerShipmentType", shipmentType, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("未满足 sqlmock 期望: %v", err)
			}
		})
	}
}

func TestOrderContainerRepo_Add_AuditErrorRollsBack(t *testing.T) {
	repo, mock, cleanup := setupTestOrderContainerRepo(t)
	defer cleanup()

	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	input := &biz.OrderContainer{
		ID:              uuid.New(),
		ContainerNo:     "MSCU1234567",
		ContainerSpecID: uuid.New(),
		GrossWeightKg:   1000,
		VolumeCbm:       20,
	}
	audit := &biz.AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Action:         "order.container.add",
		Result:         "success",
	}

	mock.ExpectQuery(`SELECT "orders"\."id"`).WithArgs(orderID, orgID).WillReturnRows(orderRows(orderID, orgID))
	mock.ExpectQuery(`SELECT COUNT\("master_data_items"\."id"\) FROM "master_data_items"`).
		WithArgs(input.ContainerSpecID, orgID, "container_spec").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "order_containers"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	result, err := repo.Add(context.Background(), orgID, orderID, input, audit)
	if err == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if result != nil {
		t.Fatalf("审计写入失败时不应返回集装箱: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}
