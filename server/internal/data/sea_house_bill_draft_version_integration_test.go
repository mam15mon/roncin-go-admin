package data

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	seadocumentmodechangeeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentmodechangeevent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const seaHouseBillDraftVersionMigration = "20261106090000_sea_house_bill_draft_versions.sql"

func TestSeaHouseBillDraftVersionLifecyclePostgres(t *testing.T) {
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	fixture := newOrderPostgresFixture(t, data)
	t.Cleanup(func() {
		if _, err := data.db.SeaDocumentModeChangeEvent.Delete().Where(seadocumentmodechangeeventent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx); err != nil {
			t.Error(err)
		}
	})
	orders := fixture.newUsecase()
	input := fixture.validInput()
	initialShipper := "首次创建的真实分单发货人"
	input.SeaDocumentInput.HouseBill.Content = &biz.SeaBillContent{ShipperText: &initialShipper}
	order, err := orders.Create(ctx, fixture.organizationID, fixture.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	documents := NewSeaDocumentRepo(data)
	read := func() *biz.SeaOrderDocuments {
		t.Helper()
		docs, err := documents.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
		if err != nil {
			t.Fatal(err)
		}
		return docs
	}
	docs := read()
	initial := requireHouseBillDraftVersion(t, data, docs.HouseBill)
	if initial.VersionNo != 1 || initial.SourceEntityVersion != 1 || initial.CreatedBy == nil || *initial.CreatedBy != fixture.actorID || initial.ShipperText == nil || *initial.ShipperText != initialShipper {
		t.Fatalf("普通创建没有归档当前真实分单: %#v", initial)
	}
	if locked := data.db.Order.GetX(ctx, order.ID).LockedAt; locked != nil {
		t.Fatal("保存分单快照不能自动审核订单")
	}

	changes := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(data))
	preview, err := changes.PreviewModeChange(ctx, fixture.organizationID, &biz.SeaDocumentModeChangeCommand{OrderID: order.ID, TargetMode: biz.SeaDocumentStructureDirect, Reason: "客户确认改直单"})
	if err != nil || !preview.Executable || preview.BaseVersion == nil || preview.BaseVersion.ID != initial.ID {
		t.Fatalf("新建未审核订单应直接取得可执行预览及当前快照: %#v %v", preview, err)
	}

	updatedShipper := "整单保存后的分单发货人"
	update := fixture.validUpdateInput()
	update.SeaDocumentInput = &biz.SeaOrderDocumentInput{
		ExpectedLinkVersion: &docs.LinkVersion,
		HouseBill: &biz.SeaHouseBillInput{
			ID: &docs.HouseBill.ID, ExpectedVersion: &docs.HouseBill.Version,
			HouseNo: docs.HouseBill.HouseNo, IssuerSource: docs.HouseBill.IssuerSource,
			Content: &biz.SeaBillContent{ShipperText: &updatedShipper},
		},
	}
	order, err = orders.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, update)
	if err != nil {
		t.Fatal(err)
	}
	docs = read()
	saved := requireHouseBillDraftVersion(t, data, docs.HouseBill)
	if saved.ID == initial.ID || saved.VersionNo != 2 || saved.ShipperText == nil || *saved.ShipperText != updatedShipper {
		t.Fatalf("整单保存后当前快照未更新: %#v", saved)
	}

	standaloneShipper := "独立分单保存后的发货人"
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.actorID, Result: "success"}
	}
	if _, err := documents.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, &biz.SeaHouseBillInput{
		HouseNo: docs.HouseBill.HouseNo, IssuerSource: docs.HouseBill.IssuerSource,
		Content: &biz.SeaBillContent{ShipperText: &standaloneShipper},
	}, audit()); err != nil {
		t.Fatal(err)
	}
	docs = read()
	current := requireHouseBillDraftVersion(t, data, docs.HouseBill)
	if current.VersionNo != 3 || current.ShipperText == nil || *current.ShipperText != standaloneShipper {
		t.Fatalf("独立保存未同步当前快照: %#v", current)
	}
	if old := data.db.SeaHouseBillVersion.GetX(ctx, initial.ID); old.ShipperText == nil || *old.ShipperText != initialShipper {
		t.Fatal("普通保存改写了首次不可变快照")
	}

	command := &biz.SeaDocumentModeChangeCommand{
		OrderID: order.ID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: docs.LinkVersion,
		ExpectedHouseBillVersion: &docs.HouseBill.Version, ExpectedCurrentVersionID: &initial.ID,
		TargetMode: biz.SeaDocumentStructureDirect, Reason: "客户确认改直单",
		IdempotencyKey: uuid.NewString(), Confirmation: seaBatchTestConfirmation(),
	}
	if err := changes.ExecuteModeChange(ctx, fixture.organizationID, fixture.actorID, command, audit()); !errors.Is(err, biz.ErrSeaDocumentModeChangeConflict) {
		t.Fatalf("旧快照基线必须冲突，不能仅凭最新实体版本执行: %v", err)
	}
	command.ExpectedCurrentVersionID = &current.ID
	if err := changes.ExecuteModeChange(ctx, fixture.organizationID, fixture.actorID, command, audit()); err != nil {
		t.Fatal(err)
	}
	if err := changes.ExecuteModeChange(ctx, fixture.organizationID, fixture.actorID, command, audit()); err != nil {
		t.Fatalf("模式切换重放失败: %v", err)
	}
	voided := data.db.SeaHouseBill.GetX(ctx, docs.HouseBill.ID)
	if voided.Status != seahousebillent.StatusVOIDED || voided.Version != docs.HouseBill.Version+1 || voided.CurrentVersionID == nil {
		t.Fatalf("模式切换没有原子作废分单: %#v", voided)
	}
	event := data.db.SeaDocumentModeChangeEvent.Query().Where(seadocumentmodechangeeventent.OrderIDEQ(order.ID)).OnlyX(ctx)
	if event.PreviousHouseBillVersionID == nil || *event.PreviousHouseBillVersionID != current.ID {
		t.Fatal("模式切换事件没有指向最后一次保存的真实分单快照")
	}
	voidVersion := data.db.SeaHouseBillVersion.GetX(ctx, *voided.CurrentVersionID)
	if voidVersion.Source != seahousebillversionent.SourceMODE_CHANGE || voidVersion.Status != seahousebillversionent.StatusVOIDED || voidVersion.ShipperText == nil || *voidVersion.ShipperText != standaloneShipper {
		t.Fatalf("作废历史丢失最后保存正文: %#v", voidVersion)
	}

	docs = read()
	newOrder := data.db.Order.GetX(ctx, order.ID)
	reverse := &biz.SeaDocumentModeChangeCommand{
		OrderID: order.ID, ExpectedOrderVersion: newOrder.Version, ExpectedLinkVersion: docs.LinkVersion,
		TargetMode: biz.SeaDocumentStructureHouse, Reason: "客户确认改回分单", IdempotencyKey: uuid.NewString(),
		Confirmation: seaBatchTestConfirmation(), NewHouseBill: &biz.SeaHouseBillInput{
			HouseNo: input.SeaDocumentInput.HouseBill.HouseNo + "NEW", IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
		},
	}
	if preview, err := changes.PreviewModeChange(ctx, fixture.organizationID, reverse); err != nil || !preview.Executable {
		t.Fatalf("反向切换预览失败: %#v %v", preview, err)
	}
	if err := changes.ExecuteModeChange(ctx, fixture.organizationID, fixture.actorID, reverse, audit()); err != nil {
		t.Fatal(err)
	}
	docs = read()
	if docs.HouseBill == nil || docs.HouseBill.ID == voided.ID || docs.HouseBill.CurrentVersionID == nil {
		t.Fatalf("反向切换应创建独立且有当前快照的新分单: %#v", docs.HouseBill)
	}
}

func requireHouseBillDraftVersion(t *testing.T, data *Data, house *biz.SeaHouseBill) *ent.SeaHouseBillVersion {
	t.Helper()
	if house == nil || house.CurrentVersionID == nil {
		t.Fatal("正式写入入口未建立分单当前快照")
	}
	version := data.db.SeaHouseBillVersion.GetX(context.Background(), *house.CurrentVersionID)
	if version.HouseBillID != house.ID || version.OrderID != house.OrderID || version.MasterBillID != house.MasterBillID || version.SourceEntityVersion != house.Version || version.Source != seahousebillversionent.SourceDRAFT_SAVE {
		t.Fatalf("当前快照与工作分单版本或来源不一致: %#v", version)
	}
	return version
}

func TestSeaHouseBillModePreviewRejectsInvalidBaselinePostgres(t *testing.T) {
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	for _, kind := range []string{"缺当前快照", "当前快照落后于工作版本"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newOrderPostgresFixture(t, data)
			order, err := fixture.newUsecase().Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
			if err != nil {
				t.Fatal(err)
			}
			house := data.db.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(order.ID)).OnlyX(ctx)
			if kind == "缺当前快照" {
				data.db.SeaHouseBill.UpdateOne(house).ClearCurrentVersionID().ExecX(ctx)
			} else {
				data.db.SeaHouseBill.UpdateOne(house).SetVersion(house.Version + 1).ExecX(ctx)
			}
			preview, err := NewSeaDocumentChangeRepo(data).PreviewModeChange(ctx, fixture.organizationID, &biz.SeaDocumentModeChangeCommand{OrderID: order.ID, TargetMode: biz.SeaDocumentStructureDirect, Reason: "验证无效版本基线"})
			if !errors.Is(err, biz.ErrSeaDocumentVersionConflict) || preview != nil {
				t.Fatalf("不完整版本基线不得伪装成可执行预览: %#v %v", preview, err)
			}
		})
	}
}

func TestSeaHouseBillDraftVersionSourceMetadata(t *testing.T) {
	checks := migrate.SeaHouseBillVersionsTable.Annotation.Checks
	if checks["sea_house_bill_versions_source_check"] != "source IN ('ORDER_LOCK', 'AMENDMENT', 'VOID', 'MODE_CHANGE', 'DRAFT_SAVE')" {
		t.Fatalf("Ent 生成元数据与正式版本来源约束不一致: %#v", checks)
	}
}

func TestSeaHouseBillDraftVersionMigrationUpgradePostgres(t *testing.T) {
	ctx := context.Background()
	data, cleanup := getIntegrationDataBeforeMigration(t, seaHouseBillDraftVersionMigration)
	t.Cleanup(cleanup)
	constraint := func() string {
		t.Helper()
		var value string
		if err := data.sqlDB.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='sea_house_bill_versions'::regclass AND conname='sea_house_bill_versions_source_check'`).Scan(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	if strings.Contains(constraint(), "DRAFT_SAVE") {
		t.Fatal("升级前基线不应接受新增来源")
	}
	for range 2 {
		if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(constraint(), "DRAFT_SAVE") {
		t.Fatal("正式升级未扩展版本来源 CHECK")
	}
	var count int
	if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version=$1`, strings.TrimSuffix(seaHouseBillDraftVersionMigration, ".sql")).Scan(&count); err != nil || count != 1 {
		t.Fatalf("正式迁移重复执行应只有一条 revision: %d %v", count, err)
	}
	fixture := newOrderPostgresFixture(t, data)
	if _, err := fixture.newUsecase().Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput()); err != nil {
		t.Fatalf("升级后正常创建路径不能写入草稿来源: %v", err)
	}
}
