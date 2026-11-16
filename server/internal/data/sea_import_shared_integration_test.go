package data

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	masterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

func TestSeaImportSharedDocumentsCargoAndArrivalPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := newOrderPostgresFixture(t, d)
	f.deferCleanupToSchema = true
	uc := f.newUsecase()
	input := seaImportInput(f, false)
	input.SeaMasterBillInput.MasterNo = "SISHARED"
	input.SeaDocumentInput.HouseBill.IssuerSource = biz.SeaHouseBillIssuerSourceOtherPartner
	input.SeaDocumentInput.HouseBill.IssuerPartnerID = &f.partnerID
	first, e := uc.Create(ctx, f.organizationID, f.actorID, input)
	if e != nil {
		t.Fatal(e)
	}
	docRepo := NewSeaDocumentRepo(d)
	docs, e := docRepo.GetSeaOrderDocuments(ctx, f.organizationID, first.ID)
	if e != nil {
		t.Fatal(e)
	}
	if docs.HouseBill.PrintTemplate != biz.SeaHouseBillPrintTemplateExternalArchive || docs.HouseBill.IssuerOrganizationID != nil || docs.HouseBill.IssuerPartnerID == nil || docs.HouseBill.IssuerPartnerName == "" {
		t.Fatalf("进口外部单证未保持真实主体: %+v", docs.HouseBill)
	}
	add := seaImportInput(f, false)
	add.SeaMasterBillInput = &biz.SeaMasterBillInput{MasterNo: "SISHARED", CandidateID: &docs.MasterBill.ID, ExpectedCandidateVersion: &docs.MasterBill.Version, CandidateTEID: &first.SeaMasterBill.TransportExecutionID, ExpectedCandidateTEVersion: &first.SeaMasterBill.TransportExecutionVersion}
	second, e := uc.Create(ctx, f.organizationID, f.actorID, add)
	if e != nil {
		t.Fatal(e)
	}
	if second.SeaMasterBill.MasterBillID != first.SeaMasterBill.MasterBillID {
		t.Fatal("同方向显式加票未共享MBL")
	}
	h, e := uc.GetImportHandling(ctx, f.organizationID, first.ID)
	if e != nil {
		t.Fatal(e)
	}
	h.Steps[0].Undertaken = true
	h.Steps[0].OccurredAt = pointerSeaImportTime(time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC))
	h, e = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, first.ID, h.OrderVersion, h, "")
	if e != nil {
		t.Fatal(e)
	}
	other, e := uc.GetImportHandling(ctx, f.organizationID, second.ID)
	if e != nil || biz.ImportHandlingFlow(other) != biz.OrderFlowImportNotApplicable {
		t.Fatal("共享主单其他票被服务变更污染")
	}
	// HBL独立维护真实外部印字及不可变历史，保存响应继续投影归档模板。
	shipper := "REAL EXTERNAL SHIPPER"
	issueDate := "2026-11-16"
	hbInput := &biz.SeaHouseBillInput{HouseNo: docs.HouseBill.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceOtherPartner, IssuerPartnerID: &f.partnerID, Content: &biz.SeaBillContent{ShipperText: &shipper}, PrintDetails: &biz.SeaHouseBillPrintDetails{IssueDate: &issueDate}}
	saved, e := biz.NewSeaDocumentUsecase(docRepo).UpdateSeaHouseBill(ctx, f.organizationID, f.actorID, first.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, hbInput, testGateAuditEvent(f, "sea_house_bill.update"))
	if e != nil {
		t.Fatal(e)
	}
	if saved.PrintTemplate != biz.SeaHouseBillPrintTemplateExternalArchive || saved.Version != docs.HouseBill.Version+1 {
		t.Fatalf("HBL保存响应: %+v", saved)
	}
	if _, e = docRepo.UpdateSeaHouseBill(ctx, f.organizationID, f.actorID, first.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, hbInput, testGateAuditEvent(f, "sea_house_bill.update")); e == nil {
		t.Fatalf("旧HBL/Link版本必须冲突: %v", e)
	}
	versions, total, e := NewSeaDocumentChangeRepo(d).ListHouseBillVersions(ctx, f.organizationID, first.ID, saved.ID, 1, 200)
	if e != nil || total < 2 || versions[0].PrintDetails == nil || versions[0].PrintDetails.IssueDate == nil || *versions[0].PrintDetails.IssueDate != issueDate {
		t.Fatalf("SI真实HBL历史: %+v %d %v", versions, total, e)
	}
	// 共享航程更改ETA只更新权威运输/经营日期，不完成其他票的私有到货服务。
	eta := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	etd, _ := time.Parse(time.RFC3339, input.ETD)
	change := &biz.SeaTransportExecutionUpdateCommand{OrderID: first.ID, ExpectedTransportExecutionVersion: first.SeaMasterBill.TransportExecutionVersion, Reason: "进口到港时间更新", IdempotencyKey: uuid.NewString(), Input: &biz.SeaTransportExecutionUpdateInput{ETD: &etd, ETA: &eta}}
	result, e := biz.NewSeaOrderChangeUsecase(NewSeaOrderChangeRepo(d), d).ExecuteTransportExecutionUpdate(ctx, f.organizationID, f.actorID, change)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []uuid.UUID{first.ID, second.ID} {
		o, e := uc.Get(ctx, f.organizationID, id)
		if e != nil || o.ETA != eta.Format(time.RFC3339) {
			t.Fatalf("共享ETA投影未更新: %+v %v", o, e)
		}
	}
	other, e = uc.GetImportHandling(ctx, f.organizationID, second.ID)
	if e != nil || biz.ImportHandlingFlow(other) != biz.OrderFlowImportNotApplicable {
		t.Fatal("共享运输自动完成了本票服务")
	}
	if result.TransportExecution.Version != 2 {
		t.Fatal("共享运输版本未推进")
	}
	// 实箱、货物与共享箱都绑定真实SI票；共享候选能返回两张SI HOUSE。
	spec := d.db.MasterDataItem.Create().SetKind(masterent.KindContainerSpec).SetCode(uuid.NewString()).SetName("进口20GP").SaveX(ctx)
	d.db.Order.UpdateOneID(first.ID).SetShipmentType(orderent.ShipmentTypeFCL).SaveX(ctx)
	actual, e := NewOrderContainerRepo(d).Add(ctx, f.organizationID, first.ID, &biz.OrderContainer{ID: uuid.New(), ContainerNo: "SI-ACTUAL", ContainerSpecID: spec.ID, PackageCount: 10, GrossWeightKg: 100, VolumeCbm: 1}, testGateAuditEvent(f, "order.container.add"))
	if e != nil || actual.ContainerNo != "SI-ACTUAL" {
		t.Fatal(e)
	}
	cargo, e := NewOrderCargoItemRepo(d).Add(ctx, f.organizationID, first.ID, &biz.OrderCargoItem{ID: uuid.New(), CargoName: "进口箱货", PackageCount: 10, GrossWeightKg: 100, VolumeCbm: 1}, testGateAuditEvent(f, "order.cargo.add"))
	if e != nil {
		t.Fatal(e)
	}
	sharedUC := biz.NewSeaSharedContainerUsecase(NewSeaSharedContainerRepo(d))
	shared, e := sharedUC.Create(ctx, f.organizationID, f.actorID, first.ID, &biz.SeaSharedContainer{ID: uuid.New(), TransportExecutionID: first.SeaMasterBill.TransportExecutionID, ContainerNo: "SISHARED-BOX", ContainerSpecID: spec.ID, PackageCount: 10, GrossWeightKg: decimal.NewFromInt(100), VolumeCbm: decimal.NewFromInt(1)})
	if e != nil {
		t.Fatal(e)
	}
	candidates, total, e := NewSeaSharedContainerRepo(d).ListCandidates(ctx, f.organizationID, first.ID, shared.TransportExecutionID, "", 1, 200)
	if e != nil || total != 2 {
		t.Fatalf("SI共享箱候选: %d %+v %v", total, candidates, e)
	}
	var allocation *biz.SeaSharedContainerAllocationInput
	for _, candidate := range candidates {
		if candidate.OrderID == first.ID {
			allocation = &biz.SeaSharedContainerAllocationInput{OrderID: first.ID, HouseBillID: candidate.HouseBillID, CargoItemID: cargo.ID, PackageCount: 10, GrossWeightKg: decimal.NewFromInt(100), VolumeCbm: decimal.NewFromInt(1), ExpectedOrderVersion: candidate.OrderVersion, ExpectedLinkVersion: candidate.LinkVersion, ExpectedHouseBillVersion: candidate.HouseBillVersion, ExpectedCargoItemVersion: cargo.Version}
		}
	}
	if allocation == nil {
		t.Fatal("缺SI分配候选")
	}
	if _, e = sharedUC.Confirm(ctx, f.organizationID, f.actorID, first.ID, shared.ID, shared.Version, []*biz.SeaSharedContainerAllocationInput{allocation}); e != nil {
		t.Fatal(e)
	}
	// 放货只登记所选真实本票HBL，不接受他票HBL或旧通用单证。
	pod, e := NewOrderReleasePodRepo(d).Add(ctx, f.organizationID, first.ID, &biz.OrderReleasePod{ID: uuid.New(), SeaDocumentType: biz.SeaDocumentTypeHouseBill, SeaDocumentID: &saved.ID}, testGateAuditEvent(f, "order.release_pod.add"))
	if e != nil || pod.SeaDocumentID == nil || *pod.SeaDocumentID != saved.ID {
		t.Fatalf("SI放货真实引用: %+v %v", pod, e)
	}
	secondDocs, e := docRepo.GetSeaOrderDocuments(ctx, f.organizationID, second.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = NewOrderReleasePodRepo(d).Add(ctx, f.organizationID, first.ID, &biz.OrderReleasePod{ID: uuid.New(), SeaDocumentType: biz.SeaDocumentTypeHouseBill, SeaDocumentID: &secondDocs.HouseBill.ID}, testGateAuditEvent(f, "order.release_pod.add")); !errors.Is(e, biz.ErrOrderReleasePodDocumentInvalid) {
		t.Fatalf("外票HBL放货未拒绝: %v", e)
	}
	// 锁内完整身份复验：伪造跨方向成员后共享写入必须拒绝。
	d.db.Order.UpdateOneID(second.ID).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetFlowStatus(orderent.FlowStatusDRAFT).SaveX(ctx)
	if _, e = docRepo.UpdateSeaMasterBillContent(ctx, f.organizationID, f.actorID, first.ID, docs.MasterBill.Version, &biz.SeaBillContent{ShipperText: &shipper}, testGateAuditEvent(f, "sea_master_bill.content.update")); e == nil {
		t.Fatal("跨方向异常成员未阻断共享正文")
	}
	if _, _, e = NewSeaSharedContainerRepo(d).List(ctx, f.organizationID, first.ID, shared.TransportExecutionID, "", 1, 20); e == nil {
		t.Fatal("跨方向共航程未阻断共享箱")
	}
	if _, e = uc.Get(ctx, f.organizationID, second.ID); e == nil {
		t.Fatal("订单摘要读取未拒绝跨方向异常引用")
	}
	// 只验证异常拒绝，不用修复分支掩盖业务写入。
}
func pointerSeaImportTime(v time.Time) *time.Time { return &v }
