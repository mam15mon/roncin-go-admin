package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func seedLegacyImportOrder(t *testing.T, d *Data, ctx context.Context, org, customer uuid.UUID) *ent.Order {
	t.Helper()
	return d.db.Order.Create().SetOrganizationID(org).SetCustomerID(customer).SetOrderNo("SI-OLD-" + uuid.NewString()).SetIdempotencyKey(uuid.NewString()).SetBusinessType("SI").SetTradeDirection("import").SetPaymentTerm("PREPAID").SetNotes("原始进口备注完整保留").SaveX(ctx)
}
func TestSeaImportMigrationPreservesDataPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, "20261116160000_sea_import_handling.sql")
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := seaImportMigrationFixture(t, d)
	o := seedLegacyImportOrder(t, d, ctx, f.organizationID, f.partnerID)
	o = d.db.Order.GetX(ctx, o.ID)
	if e := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); e != nil {
		t.Fatal(e)
	}
	after := d.db.Order.GetX(ctx, o.ID)
	if after.Notes != o.Notes || after.Version != o.Version || !after.CreatedAt.Equal(o.CreatedAt) || string(after.FlowStatus) != "IMPORT_NOT_APPLICABLE" {
		t.Fatalf("迁移修改了非目标事实: %+v", after)
	}
	h, e := NewOrderRepo(d).GetImportHandling(ctx, f.organizationID, o.ID)
	if e != nil || len(h.Steps) != 5 {
		t.Fatalf("迁移未初始化合法无承接: %+v %v", h, e)
	}
	if e := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); e != nil {
		t.Fatal(e)
	}
	if d.db.OrderImportHandling.Query().CountX(ctx) != 1 || d.db.OrderImportHandlingEvent.Query().CountX(ctx) != 0 {
		t.Fatal("重复迁移不应重写服务或伪造事件")
	}
	if _, e = d.sqlDB.ExecContext(ctx, `UPDATE orders SET flow_status='BOOKED' WHERE id=$1`, o.ID); e == nil {
		t.Fatal("数据库未拒绝进口出口流程混用")
	}
	if _, e = d.sqlDB.ExecContext(ctx, `UPDATE orders SET business_type='SE' WHERE id=$1`, o.ID); e == nil {
		t.Fatal("数据库未拒绝出口进口摘要混用")
	}
}
func TestSeaImportMigrationRejectsUnverifiableSnapshotsPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, "20261116160000_sea_import_handling.sql")
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := seaImportMigrationFixture(t, d)
	o := seedLegacyImportOrder(t, d, ctx, f.organizationID, f.partnerID)
	d.db.OrderLockRecord.Create().SetOrganizationID(f.organizationID).SetOrderID(o.ID).SetOrderNo(o.OrderNo).SetBusinessType("SI").SetGeneration(1).SetLockedBy(f.actorID).SetLockedAt(time.Now()).SetOrderVersionAtLock(1).SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint("legacy").SaveX(ctx)
	if e := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); e == nil {
		t.Fatal("不可验证SI审核快照必须显式停止")
	}
	// 只检查隔离Schema，不能沿search_path命中开发库public中的同名正式表。
	var exists bool
	if e := d.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_tables WHERE schemaname = current_schema() AND tablename = 'order_import_handlings')`).Scan(&exists); e != nil {
		t.Fatal(e)
	}
	if exists {
		t.Fatal("迁移失败留下部分DDL")
	}
	if actual := d.db.Order.GetX(ctx, o.ID); actual.FlowStatus != o.FlowStatus {
		t.Fatal("迁移失败未回滚摘要")
	}
	var count int
	if e := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261116160000_sea_import_handling'`).Scan(&count); e != nil {
		t.Fatal(e)
	}
	if count != 0 {
		t.Fatal("失败迁移记录了revision")
	}
}
func TestSeaImportSchemaMetadata(t *testing.T) {
	checks := migrate.OrdersTable.Annotation.Checks
	if !strings.Contains(checks["orders_flow_status_check"], "IMPORT_COMPLETED") || !strings.Contains(checks["orders_import_flow_applicability_check"], "business_type IN ('SI', 'AI')") {
		t.Fatalf("Ent摘要约束不同源: %+v", checks)
	}
	if !strings.Contains(migrate.OrderLockRecordsTable.Annotation.Checks["order_lock_records_business_type_document_refs_check"], "business_type IN ('SE', 'SI')") {
		t.Fatal("SI审核快照CHECK未同步")
	}
	for _, table := range []string{migrate.OrderImportHandlingsTable.Name, migrate.OrderImportHandlingEventsTable.Name} {
		if table == "" {
			t.Fatal("缺少正式表元数据")
		}
	}
}

// 前态失败验证只依赖迁移前已有表；清理由隔离Schema统一负责。
func seaImportMigrationFixture(t *testing.T, d *Data) *orderPostgresFixture {
	t.Helper()
	ctx := context.Background()
	org := d.db.Organization.Create().SetCode(uuid.NewString()).SetName("进口迁移验证公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	user := d.db.User.Create().SetDisplayName("迁移操作者").SaveX(ctx)
	partner := d.db.Partner.Create().SetOrganizationID(org.ID).SetCode(uuid.NewString()).SetLegalName("迁移客户").SetNormalizedName("迁移客户").SaveX(ctx)
	return &orderPostgresFixture{t: t, data: d, organizationID: org.ID, partnerID: partner.ID, actorID: user.ID}
}
