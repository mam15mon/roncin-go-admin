package data

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const orderMilestoneRetirementVersion = "20261030140000_remove_order_milestones"

// 正式迁移必须移除退役表和授权，保留订单、其他子资源与既有业务审计。
func TestOrderMilestoneRetirementUpgradePostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261030140000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	client := data.db
	organization, err := client.Organization.Create().SetCode("MILESTONE-RETIRE").SetName("里程碑退役测试公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := client.User.Create().SetDisplayName("保留建单人员").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	partner, err := client.Partner.Create().SetOrganizationID(organization.ID).SetCode("KEEP-CUSTOMER").SetLegalName("保留客户").SetNormalizedName("保留客户").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	lockedAt := time.Date(2026, 10, 29, 3, 4, 5, 0, time.UTC)
	order, err := client.Order.Create().
		SetOrganizationID(organization.ID).SetCustomerID(partner.ID).
		SetOrderNo("SE-MILESTONE-RETIRE").SetIdempotencyKey(uuid.NewString()).
		SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).
		SetPaymentTerm(orderent.PaymentTermPREPAID).SetFlowStatus(orderent.FlowStatusBOOKED).
		SetClosureStatus(orderent.ClosureStatusCLOSED).SetVersion(7).
		SetClosureReason("保留完结状态").SetClosedAt(lockedAt).SetClosedBy(actor.ID).
		SetLockedAt(lockedAt).SetLockedBy(actor.ID).SetLockSource(orderent.LockSourceMANUAL).
		SetLockGeneration(2).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cargo, err := client.OrderCargoItem.Create().SetOrganizationID(organization.ID).SetOrderID(order.ID).
		SetCargoName("保留货物").SetPackageCount(3).SetGrossWeightKg(200).SetVolumeCbm(1.5).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	pod, err := client.OrderReleasePod.Create().SetOrderID(order.ID).SetReleaseNo("KEEP-POD").SetNote("保留放货记录").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := client.OrderAttachmentAsset.Create().SetOrganizationID(organization.ID).SetObjectKey("retirement/keep.pdf").
		SetFileName("保留附件.pdf").SetMimeType("application/pdf").SetFileSize(12).SetUploadedBy(actor.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	attachment, err := client.OrderAttachment.Create().SetOrderID(order.ID).SetAssetID(asset.ID).
		SetDocType("OTHER").SetIdempotencyKey(uuid.NewString()).SetCreatedBy(actor.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	milestoneID := uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO order_milestones(id,created_at,updated_at,order_id,type,occurred_at,note,updated_by)
		VALUES($1,now(),now(),$2,'BOOKING_CONFIRMED',$3,'待删除的独立里程碑',$4)`, milestoneID, order.ID, lockedAt, actor.ID); err != nil {
		t.Fatal(err)
	}
	audit, err := client.AuditLog.Create().SetOrganizationID(organization.ID).SetUserID(actor.ID).
		SetAction("order.milestone.set").SetResult(auditlog.ResultSuccess).
		SetResourceType("order").SetResourceID(order.ID.String()).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SyncPermissionManifest(ctx, data.sqlDB); err != nil {
		t.Fatal(err)
	}
	liveKeys := []string{
		access.OrderPermission(access.OrderBusinessSE, access.OrderRead),
		access.OrderPermission(access.OrderBusinessSE, access.OrderUpdate),
		access.OrderPermission(access.OrderBusinessSE, access.OrderAttachmentRead),
	}
	grants, err := client.Permission.Query().Where(permissionent.KeyIn(liveKeys...)).All(ctx)
	if err != nil || len(grants) != len(liveKeys) {
		t.Fatalf("准备保留权限失败: count=%d err=%v", len(grants), err)
	}
	var retiredIDs []uuid.UUID
	for _, kind := range []string{"se", "si", "ae", "ai", "land", "rail"} {
		for _, operation := range []string{"read", "set"} {
			retired, err := client.Permission.Create().SetKey("business.order." + kind + ".milestone." + operation).
				SetName("退役里程碑权限").SetGroup("测试").SetDescription("迁移前态授权").Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			retiredIDs = append(retiredIDs, retired.ID)
			grants = append(grants, retired)
		}
	}
	role, err := client.Role.Create().SetOrganizationID(organization.ID).SetCode("operator").SetName("保留操作角色").AddPermissions(grants...).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// 以整行快照核对，不能仅用数量不变掩盖状态、版本、锁或内容被改写。
	preserved := map[string]uuid.UUID{
		"orders": order.ID, "order_cargo_items": cargo.ID, "order_release_pods": pod.ID,
		"order_attachment_assets": asset.ID, "order_attachments": attachment.ID, "audit_logs": audit.ID,
	}
	snapshots := make(map[string]string, len(preserved))
	for table, id := range preserved {
		snapshots[table] = retirementRowSnapshot(t, data.sqlDB, table, id)
	}
	dir := filepath.Join("..", "..", "migrations")
	apply := func() {
		t.Helper()
		if err := migration.ApplyWithOptions(ctx, data.sqlDB, dir, migration.Options{
			PostStep: func(conn *sql.Conn) error {
				_, err := SyncPermissionManifest(ctx, conn)
				return err
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	assertMilestoneRetiredSchema(t, data)
	for table, id := range preserved {
		if got := retirementRowSnapshot(t, data.sqlDB, table, id); got != snapshots[table] {
			t.Fatalf("里程碑退役改写了 %s 的非目标数据", table)
		}
	}
	if count, err := client.Permission.Query().Where(permissionent.IDIn(retiredIDs...)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("退役权限未完全删除: count=%d err=%v", count, err)
	}
	if count, err := role.QueryPermissions().Count(ctx); err != nil || count != len(liveKeys) {
		t.Fatalf("目标外授权未保留或退役关联未清理: count=%d err=%v", count, err)
	}
	for _, key := range liveKeys {
		if exists, err := role.QueryPermissions().Where(permissionent.KeyEQ(key)).Exist(ctx); err != nil || !exists {
			t.Fatalf("相邻权限 %s 未保留: exists=%v err=%v", key, exists, err)
		}
	}
	// 再执行完整迁移与权限同步，不得重放删表或改写保留行。
	apply()
	for table, id := range preserved {
		if got := retirementRowSnapshot(t, data.sqlDB, table, id); got != snapshots[table] {
			t.Fatalf("重复迁移改写了 %s", table)
		}
	}
	var revisions int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version=$1 AND length(checksum)=64`, orderMilestoneRetirementVersion).Scan(&revisions); err != nil || revisions != 1 {
		t.Fatalf("迁移版本记录不正确: count=%d err=%v", revisions, err)
	}
}

func TestOrderMilestoneRetirementColdStartPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	assertMilestoneRetiredSchema(t, data)
}

func retirementRowSnapshot(t *testing.T, db *sql.DB, table string, id uuid.UUID) string {
	t.Helper()
	var snapshot string
	if err := db.QueryRowContext(context.Background(), `SELECT row_to_json(t)::text FROM "`+table+`" AS t WHERE id=$1`, id).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func assertMilestoneRetiredSchema(t *testing.T, data *Data) {
	t.Helper()
	var count int
	if err := data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM pg_class WHERE relnamespace=current_schema()::regnamespace AND relname IN ('order_milestones','ordermilestone_updated_at','ordermilestone_order_id_type','ordermilestone_order_id_occurred_at')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("退役表或索引仍存在: count=%d err=%v", count, err)
	}
}
