package data

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
)

// TestSeaHouseBillPrintDetailsPostgres 验证正式迁移、真实写入门禁和不可变历史。
func TestSeaHouseBillPrintDetailsPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	fixture := newOrderPostgresFixture(t, data)
	var columns int
	if err := data.sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name IN ('sea_house_bills','sea_house_bill_versions') AND column_name='print_details' AND data_type='jsonb' AND is_nullable='YES'").Scan(&columns); err != nil || columns != 2 {
		t.Fatalf("正式迁移未形成两份可空JSONB字段: %d %v", columns, err)
	}
	count := int32(3)
	issueDate := "2026-11-04"
	contact := "业务联系窗口\nOPERATIONS"
	details := &biz.SeaHouseBillPrintDetails{OriginalCount: &count, IssueDate: &issueDate, HeaderContactText: &contact}
	input := fixture.validInput()
	input.SeaDocumentInput.HouseBill.PrintDetails = details
	uc := fixture.newUsecase()
	order, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewSeaDocumentRepo(data)
	read := func() *biz.SeaOrderDocuments {
		t.Helper()
		got, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	docs := read()
	t.Cleanup(func() {
		if _, err := data.db.SeaHouseBill.UpdateOneID(docs.HouseBill.ID).ClearCurrentVersionID().Save(ctx); err != nil {
			t.Error(err)
		}
		if _, err := data.db.SeaHouseBillVersion.Delete().Where(seahousebillversionent.HouseBillIDEQ(docs.HouseBill.ID)).Exec(ctx); err != nil {
			t.Error(err)
		}
	})
	if !reflect.DeepEqual(docs.HouseBill.PrintDetails, details) {
		t.Fatalf("创建后印字未往返: %#v", docs.HouseBill.PrintDetails)
	}
	update := fixture.validUpdateInput()
	update.IdempotencyKey = uuid.NewString()
	order, err = uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, update)
	if err != nil {
		t.Fatal(err)
	}
	docs = read()
	if !reflect.DeepEqual(docs.HouseBill.PrintDetails, details) {
		t.Fatal("普通订单保存清空印字")
	}
	makeInput := func() *biz.SeaHouseBillInput {
		return &biz.SeaHouseBillInput{HouseNo: docs.HouseBill.HouseNo, IssuerSource: docs.HouseBill.IssuerSource, Content: docs.HouseBill.Content, PrintDetails: details}
	}
	audit := func(actor uuid.UUID) *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &actor, Result: "success"}
	}
	initialVersion := docs.HouseBill.Version
	nextDate := "2026-11-05"
	nextDetails := *details
	nextDetails.IssueDate = &nextDate
	nextInput := makeInput()
	nextInput.PrintDetails = &nextDetails
	changed, err := repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, nextInput, audit(fixture.actorID))
	if err != nil || !reflect.DeepEqual(changed.PrintDetails, &nextDetails) {
		t.Fatalf("制单保存失败: %#v %v", changed, err)
	}
	if _, err = repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, initialVersion, docs.LinkVersion, makeInput(), audit(fixture.actorID)); err == nil {
		t.Fatal("旧版本覆盖未拒绝")
	}
	docs = read()
	if docs.HouseBill.Version != initialVersion+1 || !reflect.DeepEqual(docs.HouseBill.PrintDetails, &nextDetails) {
		t.Fatal("冲突发生部分写入")
	}
	failedAudit := audit(fixture.actorID)
	failedAudit.Result = "invalid-result"
	versionCount := data.db.SeaHouseBillVersion.Query().Where(seahousebillversionent.HouseBillIDEQ(docs.HouseBill.ID)).CountX(ctx)
	if _, err = repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, makeInput(), failedAudit); err == nil {
		t.Fatal("审计校验失败应回滚")
	}
	afterRollback := read()
	if afterRollback.HouseBill.Version != docs.HouseBill.Version || !reflect.DeepEqual(afterRollback.HouseBill.PrintDetails, &nextDetails) || !reflect.DeepEqual(afterRollback.HouseBill.CurrentVersionID, docs.HouseBill.CurrentVersionID) {
		t.Fatal("审计失败未回滚印字和当前快照指针")
	}
	if count := data.db.SeaHouseBillVersion.Query().Where(seahousebillversionent.HouseBillIDEQ(docs.HouseBill.ID)).CountX(ctx); count != versionCount {
		t.Fatal("审计失败留下了部分草稿快照")
	}
	var snapshotID uuid.UUID
	err = data.WithTx(ctx, func(tx *ent.Tx) error {
		hbl, err := tx.SeaHouseBill.Query().Where(seahousebillent.IDEQ(docs.HouseBill.ID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		snapshotID, err = ensureSeaHouseBillLockVersion(ctx, tx, fixture.organizationID, hbl, order.ID, docs.MasterBill.ID, &fixture.actorID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, makeInput(), audit(fixture.actorID)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := NewSeaDocumentChangeRepo(data).GetDocumentVersion(ctx, fixture.organizationID, order.ID, snapshotID, biz.SeaDocumentTypeHouseBill)
	if err != nil || !reflect.DeepEqual(snapshot.PrintDetails, &nextDetails) {
		t.Fatalf("历史快照被当前印字修改影响: %#v %v", snapshot, err)
	}
	docs = read()
	data.db.Order.UpdateOneID(order.ID).SetLockedAt(time.Now()).SetLockedBy(fixture.actorID).SetLockGeneration(1).SaveX(ctx)
	if _, err = repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, nextInput, audit(fixture.actorID)); err == nil {
		t.Fatal("锁单未阻止资料写入")
	}
	locked := read()
	if locked.HouseBill.PrintDetails == nil || *locked.HouseBill.PrintDetails.IssueDate != issueDate {
		t.Fatal("锁定后不能读取已保存印字")
	}
	if _, err = repo.GetSeaOrderDocuments(ctx, uuid.New(), order.ID); err == nil {
		t.Fatal("跨组织读取印字未拒绝")
	}
}

func TestSeaHouseBillPrintDetailsAmendmentAndModeChangePostgres(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	repo := NewSeaDocumentChangeRepo(f.data)
	uc := biz.NewSeaDocumentChangeUsecase(repo)
	house := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	issueDate := "2026-11-04"
	details := &biz.SeaHouseBillPrintDetails{IssueDate: &issueDate}
	command := &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: f.hblID,
		ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: house.Version, ExpectedCurrentVersionID: f.hblVerID,
		Reason: "补充本票签发日期", IdempotencyKey: uuid.NewString(), Confirmation: f.confirmation(),
		Input: &biz.SeaDocumentAmendmentInput{HouseBill: &biz.SeaHouseBillInput{HouseNo: house.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, Content: &biz.SeaBillContent{ShipperText: house.ShipperText}, PrintDetails: details}},
	}
	preview, err := uc.PreviewAmendment(ctx, f.orgID, command)
	if err != nil || !preview.Executable || len(preview.Differences) != 1 || preview.Differences[0].Field != "print_details.issue_date" {
		t.Fatalf("单独印字改单预览不正确: %#v %v", preview, err)
	}
	version, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, command, f.audit())
	if err != nil || !reflect.DeepEqual(version.PrintDetails, details) {
		t.Fatalf("改单未写入印字历史: %#v %v", version, err)
	}
	repeated, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, command, f.audit())
	if err != nil || repeated.ID != version.ID {
		t.Fatalf("印字改单幂等失败: %v", err)
	}
	old, err := repo.GetDocumentVersion(ctx, f.orgID, f.orderID, f.hblVerID, biz.SeaDocumentTypeHouseBill)
	if err != nil || old.PrintDetails != nil {
		t.Fatalf("旧版印字被回填: %#v %v", old, err)
	}
	house = f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	order = f.data.db.Order.GetX(ctx, f.orderID)
	link := f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID)
	commandMode := &biz.SeaDocumentModeChangeCommand{OrderID: f.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: link.Version, ExpectedHouseBillVersion: &house.Version, ExpectedCurrentVersionID: house.CurrentVersionID, TargetMode: biz.SeaDocumentStructureDirect, Reason: "客户确认改直单", IdempotencyKey: uuid.NewString(), Confirmation: f.confirmation()}
	if err = uc.ExecuteModeChange(ctx, f.orgID, f.actorID, commandMode, f.audit()); err != nil {
		t.Fatalf("HOUSE 转 DIRECT 失败: %v", err)
	}
	voidedHouse := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	if voidedHouse.Status != seahousebillent.StatusVOIDED || voidedHouse.CurrentVersionID == nil {
		t.Fatal("模式切换未生成作废版本")
	}
	voided, err := repo.GetDocumentVersion(ctx, f.orgID, f.orderID, *voidedHouse.CurrentVersionID, biz.SeaDocumentTypeHouseBill)
	if err != nil || !reflect.DeepEqual(voided.PrintDetails, details) {
		t.Fatalf("模式切换作废版本丢失印字: %#v %v", voided, err)
	}
	order = f.data.db.Order.GetX(ctx, f.orderID)
	link = f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID)
	nextDate := "2026-11-06"
	nextDetails := &biz.SeaHouseBillPrintDetails{IssueDate: &nextDate}
	newMode := &biz.SeaDocumentModeChangeCommand{OrderID: f.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: link.Version, TargetMode: biz.SeaDocumentStructureHouse, Reason: "客户确认改回分单", IdempotencyKey: uuid.NewString(), Confirmation: f.confirmation(), NewHouseBill: &biz.SeaHouseBillInput{HouseNo: house.HouseNo + "NEW", IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, PrintDetails: nextDetails}}
	if err = uc.ExecuteModeChange(ctx, f.orgID, f.actorID, newMode, f.audit()); err != nil {
		t.Fatalf("DIRECT 转 HOUSE 失败: %v", err)
	}
	newDocs, err := NewSeaDocumentRepo(f.data).GetSeaOrderDocuments(ctx, f.orgID, f.orderID)
	if err != nil || !reflect.DeepEqual(newDocs.HouseBill.PrintDetails, nextDetails) {
		t.Fatalf("模式切换新 HBL 未保存本次印字: %#v %v", newDocs, err)
	}
}
