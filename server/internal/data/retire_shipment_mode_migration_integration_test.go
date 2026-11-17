package data

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const retireShipmentModeMigration = "20261117100000_retire_order_shipment_mode"

func TestRetireShipmentModeMigrationPreservesOrdersPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261117100000")
	t.Cleanup(cleanup)
	fixture := newOrderPostgresFixture(t, data)
	ctx := context.Background()
	usecase := fixture.newUsecase()
	modes := []*string{nil, directoryTestString("TRADITIONAL_FORWARDING"), directoryTestString("CROSS_BORDER")}
	for i, mode := range modes {
		input := fixture.validInput()
		input.SeaMasterBillInput.MasterNo = "RETIRE" + fixture.suffix + strconv.Itoa(i)
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
		if err != nil {
			t.Fatal(err)
		}
		// 新契约没有模式字段，仅在正式迁移前态用 SQL 写入待退役值。
		if _, err := data.sqlDB.ExecContext(ctx, `UPDATE orders SET shipment_mode=$1 WHERE id=$2`, mode, created.ID); err != nil {
			t.Fatal(err)
		}
	}
	queries := map[string]string{
		"订单":   `SELECT jsonb_agg(to_jsonb(t) - 'shipment_mode' ORDER BY id)::text FROM orders t`,
		"主单":   `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM sea_master_bills t`,
		"航程":   `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM sea_transport_executions t`,
		"活动关联": `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM sea_master_bill_order_links t`,
		"分单":   `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM sea_house_bills t`,
	}
	snapshots := make(map[string]string, len(queries))
	for name, query := range queries {
		var value string
		if err := data.sqlDB.QueryRowContext(ctx, query).Scan(&value); err != nil {
			t.Fatal(err)
		}
		snapshots[name] = value
	}
	dir := filepath.Join("..", "..", "migrations")
	if err := migration.Apply(ctx, data.sqlDB, dir); err != nil {
		t.Fatal(err)
	}
	assertShipmentModeRetired(t, data)
	for name, query := range queries {
		var value string
		if err := data.sqlDB.QueryRowContext(ctx, query).Scan(&value); err != nil {
			t.Fatal(err)
		}
		if value != snapshots[name] {
			t.Fatalf("退役迁移改写%s", name)
		}
	}
	if err := migration.Apply(ctx, data.sqlDB, dir); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version=$1`, retireShipmentModeMigration).Scan(&count); err != nil || count != 1 {
		t.Fatalf("重复迁移记录异常: count=%d err=%v", count, err)
	}
}

func TestRetireShipmentModeMigrationColdStartPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	assertShipmentModeRetired(t, data)
	fixture := newOrderPostgresFixture(t, data)
	created, err := fixture.newUsecase().Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.validInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.newUsecase().UpdateDraft(context.Background(), fixture.organizationID, fixture.actorID, created.ID, created.Version, fixture.validUpdateInput()); err != nil {
		t.Fatalf("退役后订单更新失败: %v", err)
	}
}

func assertShipmentModeRetired(t *testing.T, data *Data) {
	t.Helper()
	var count int
	err := data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='orders' AND column_name='shipment_mode'`).Scan(&count)
	if err != nil || count != 0 {
		t.Fatalf("模式列应已退役: count=%d err=%v", count, err)
	}
}
