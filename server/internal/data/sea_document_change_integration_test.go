package data

import (
	"context"
	"sync"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type seaDocumentChangeFixture struct {
	data           *Data
	orgID          uuid.UUID
	actorID        uuid.UUID
	partnerID      uuid.UUID
	orderID        uuid.UUID
	mblID          uuid.UUID
	mblVerID       uuid.UUID
	hblID          uuid.UUID
	hblVerID       uuid.UUID
	hblNo          string
	linkID         uuid.UUID
	currentHBLLock *uint64
}

func stringPtr(s string) *string { return &s }

func newSeaDocumentChangeFixture(t *testing.T) *seaDocumentChangeFixture {
	t.Helper()
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	suffix := uuid.NewString()[:8]

	org := data.db.Organization.Create().
		SetCode("DOC-" + suffix).
		SetName("单证变更测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		SaveX(ctx)
	actor := data.db.User.Create().SetDisplayName("单证变更测试用户").SetEnabled(true).SaveX(ctx)
	partner := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("DOC-P-" + suffix).
		SetLegalName("单证变更测试合作伙伴").
		SetNormalizedName("单证变更测试合作伙伴").
		SaveX(ctx)
	shippingLine := data.db.ShippingLine.Create().
		SetScacCode("DCTL").
		SetNameZh("单证变更测试船公司").
		SetNameEn("Document Change Test Shipping Line").
		SetCountryCode("CN").
		SetEnabled(true).
		SaveX(ctx)
	order := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(org.ID).
		SetOrderNo("SE-DOC-" + suffix).
		SetCustomerID(partner.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		SetFlowStatus("DRAFT").
		SetVersion(1).
		SaveX(ctx)
	exec := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(shippingLine.ID).
		SetVesselName("EVER TEST").
		SetVoyageNo("V001").
		SetVersion(1).
		SaveX(ctx)
	mbl := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(shippingLine.ID).
		SetMasterNo("MBLDOC" + suffix).
		SetNormalizedMasterNo("MBLDOC" + suffix).
		SetShipperText("旧主单发货人").
		SetVersion(1).
		SaveX(ctx)
	link := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetTransportExecutionID(exec.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetVersion(1).
		SaveX(ctx)
	hblNo := "HBLDOC" + suffix
	hbl := data.db.SeaHouseBill.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetHouseNo(hblNo).
		SetNormalizedHouseNo(hblNo).
		SetIssuerSource(seahousebillent.IssuerSourceCUSTOMER_PARTNER).
		SetIssuerPartnerID(partner.ID).
		SetShipperText("旧分单发货人").
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		SaveX(ctx)

	f := &seaDocumentChangeFixture{data: data, orgID: org.ID, actorID: actor.ID, partnerID: partner.ID, orderID: order.ID, mblID: mbl.ID, hblID: hbl.ID, hblNo: hblNo, linkID: link.ID}
	err := data.WithTx(ctx, func(tx *ent.Tx) error {
		txMBL, err := tx.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			return err
		}
		mblVersion, err := createMasterVersion(ctx, tx, txMBL, exec, actor.ID, biz.VersionSourceOrderLock, nil, nil, nil, nil)
		if err != nil {
			return err
		}
		if _, err = txMBL.Update().SetCurrentVersionID(mblVersion.ID).Save(ctx); err != nil {
			return err
		}
		f.mblVerID = mblVersion.ID
		txHBL, err := tx.SeaHouseBill.Get(ctx, hbl.ID)
		if err != nil {
			return err
		}
		hblVersion, err := createHouseVersion(ctx, tx, txHBL, actor.ID, biz.VersionSourceOrderLock, nil, nil, nil, nil)
		if err != nil {
			return err
		}
		if _, err = txHBL.Update().SetCurrentVersionID(hblVersion.ID).Save(ctx); err != nil {
			return err
		}
		f.hblVerID = hblVersion.ID
		return nil
	})
	if err != nil {
		t.Fatalf("创建初始不可变版本失败: %v", err)
	}
	return f
}

func (f *seaDocumentChangeFixture) audit() *biz.AuditEvent {
	return &biz.AuditEvent{OrganizationID: &f.orgID, UserID: &f.actorID, Result: "success"}
}

func (f *seaDocumentChangeFixture) confirmation() *biz.SeaExternalConfirmation {
	return &biz.SeaExternalConfirmation{
		ConfirmedByParty: "船代确认窗口",
		ConfirmedAt:      time.Now().UTC().Truncate(time.Second),
		ConfirmationNote: "船代邮件确认可以变更",
	}
}

func TestSeaDocumentChangePostgresFlows(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))

	t.Run("MBL 单改追加不可变版本且幂等", func(t *testing.T) {
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		cmd := &biz.SeaDocumentAmendmentCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
			Reason: "客户更正发货人", IdempotencyKey: "mbl-amend-" + uuid.NewString(),
			Input:        &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPtr("新主单发货人")}},
			Confirmation: f.confirmation(),
		}
		preview, err := uc.PreviewAmendment(ctx, f.orgID, cmd)
		if err != nil || !preview.Executable || preview.BaseVersion.ID != f.mblVerID {
			t.Fatalf("MBL改单预览不符合预期: preview=%+v err=%v", preview, err)
		}
		result, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil {
			t.Fatalf("执行 MBL 单改失败: %v", err)
		}
		if result.VersionNo != 2 || result.Source != biz.VersionSourceAmendment || result.ID == f.mblVerID {
			t.Fatalf("MBL 新版本不符合预期: %+v", result)
		}
		oldVersion := f.data.db.SeaMasterBillVersion.GetX(ctx, f.mblVerID)
		if oldVersion.ShipperText == nil || *oldVersion.ShipperText != "旧主单发货人" {
			t.Fatalf("旧 MBL 版本被修改: %+v", oldVersion.ShipperText)
		}
		current := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		if current.CurrentVersionID == nil || *current.CurrentVersionID != result.ID {
			t.Fatalf("MBL current_version_id 未切换到新版本")
		}
		idempotent, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil || idempotent.ID != result.ID {
			t.Fatalf("MBL 单改幂等重试失败: result=%+v err=%v", idempotent, err)
		}
	})

	t.Run("MBL 作废携带外部确认且幂等", func(t *testing.T) {
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		cmd := &biz.SeaDocumentVoidCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
			Reason: "船公司确认主单作废", IdempotencyKey: "mbl-void-" + uuid.NewString(),
			Confirmation: f.confirmation(),
		}
		preview, err := uc.PreviewVoid(ctx, f.orgID, cmd)
		if err != nil || !preview.Executable {
			t.Fatalf("MBL作废预览不符合预期: preview=%+v err=%v", preview, err)
		}
		event, err := uc.ExecuteVoid(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil {
			t.Fatalf("执行 MBL 作废失败: %v", err)
		}
		mbl = f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		if mbl.Status != seamasterbillent.StatusVOIDED || event.ResultVersionID == nil || *mbl.CurrentVersionID != *event.ResultVersionID {
			t.Fatalf("MBL作废身份或版本指针错误: mbl=%+v event=%+v", mbl, event)
		}
		old := f.data.db.SeaMasterBillVersion.GetX(ctx, f.mblVerID)
		if old.Status == "VOIDED" {
			t.Fatal("旧 MBL 不可变版本被覆盖为 VOIDED")
		}
		idempotent, err := uc.ExecuteVoid(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil || idempotent.ID != event.ID {
			t.Fatalf("MBL 作废幂等重试失败: event=%+v err=%v", idempotent, err)
		}
	})

	t.Run("HBL 单改保留完整历史且独立作废被拒绝", func(t *testing.T) {
		vef := newSeaDocumentChangeFixture(t)
		vefUC := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(vef.data))
		hbl := vef.data.db.SeaHouseBill.GetX(ctx, vef.hblID)
		order := vef.data.db.Order.GetX(ctx, vef.orderID)
		amendCmd := &biz.SeaDocumentAmendmentCommand{
			OrderID: vef.orderID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: vef.hblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: hbl.Version, ExpectedCurrentVersionID: *hbl.CurrentVersionID,
			Reason: "客户更正分单发货人", IdempotencyKey: "hbl-amend-" + uuid.NewString(),
			Input: &biz.SeaDocumentAmendmentInput{HouseBill: &biz.SeaHouseBillInput{
				HouseNo: hbl.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner,
				Content: &biz.SeaBillContent{ShipperText: stringPtr("新分单发货人")},
			}},
			Confirmation: vef.confirmation(),
		}
		amended, err := vefUC.ExecuteAmendment(ctx, vef.orgID, vef.actorID, amendCmd, vef.audit())
		if err != nil {
			t.Fatalf("执行 HBL 单改失败: %v", err)
		}
		old := vef.data.db.SeaHouseBillVersion.GetX(ctx, vef.hblVerID)
		if old.ShipperText == nil || *old.ShipperText != "旧分单发货人" {
			t.Fatal("旧 HBL 版本被修改")
		}
		hbl = vef.data.db.SeaHouseBill.GetX(ctx, vef.hblID)
		order = vef.data.db.Order.GetX(ctx, vef.orderID)
		voidCmd := &biz.SeaDocumentVoidCommand{
			OrderID: vef.orderID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: vef.hblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: hbl.Version, ExpectedCurrentVersionID: amended.ID,
			Reason: "客户确认作废", IdempotencyKey: "hbl-void-" + uuid.NewString(),
			Confirmation: vef.confirmation(),
		}
		if _, err = vefUC.ExecuteVoid(ctx, vef.orgID, vef.actorID, voidCmd, vef.audit()); kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentStructureConflict.Reason {
			t.Fatalf("HBL 独立作废必须走模式切换，错误=%v", err)
		}
	})

	t.Run("HOUSE 切换 DIRECT 作废当前 HBL 且保留模式事件", func(t *testing.T) {
		vef := newSeaDocumentChangeFixture(t)
		vefUC := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(vef.data))
		hbl := vef.data.db.SeaHouseBill.GetX(ctx, vef.hblID)
		order := vef.data.db.Order.GetX(ctx, vef.orderID)
		hblVersion := hbl.Version
		cmd := &biz.SeaDocumentModeChangeCommand{
			OrderID: vef.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 1,
			ExpectedHouseBillVersion: &hblVersion, ExpectedCurrentVersionID: hbl.CurrentVersionID,
			TargetMode: biz.SeaDocumentStructureDirect,
			Reason:     "客户改直单", IdempotencyKey: "mode-h2d-" + uuid.NewString(),
			Confirmation: vef.confirmation(),
		}
		if err := vefUC.ExecuteModeChange(ctx, vef.orgID, vef.actorID, cmd, vef.audit()); err != nil {
			t.Fatalf("HOUSE 切换 DIRECT 失败: %v", err)
		}
		hbl = vef.data.db.SeaHouseBill.GetX(ctx, vef.hblID)
		if hbl.Status != seahousebillent.StatusVOIDED {
			t.Fatalf("原当前 HBL 应为 VOIDED, 实际: %s", hbl.Status)
		}
		link := vef.data.db.SeaMasterBillOrderLink.GetX(ctx, vef.linkID)
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureDIRECT {
			t.Fatalf("Link 模式应为 DIRECT, 实际: %s", link.DocumentStructure)
		}
		events, total, err := vefUC.ListDocumentEvents(ctx, vef.orgID, vef.orderID, 1, 50)
		if err != nil || total == 0 {
			t.Fatalf("模式切换事件未记录: total=%d err=%v", total, err)
		}
		foundModeChange := false
		for _, event := range events {
			if event.EventType == biz.SeaDocumentEventTypeModeChange {
				foundModeChange = true
				if event.Confirmation == nil || event.Confirmation.ConfirmedByParty == "" {
					t.Fatalf("模式事件缺少外部确认: %+v", event)
				}
			}
		}
		if !foundModeChange {
			t.Fatal("事件历史中缺少模式切换事件")
		}
		// 幂等重试
		if err := vefUC.ExecuteModeChange(ctx, vef.orgID, vef.actorID, cmd, vef.audit()); err != nil {
			t.Fatalf("模式切换幂等重试失败: %v", err)
		}
		// VOIDED HBL 不可再走普通编辑
		link = vef.data.db.SeaMasterBillOrderLink.GetX(ctx, vef.linkID)
		_, err = NewSeaDocumentRepo(vef.data).UpdateSeaHouseBill(ctx, vef.orgID, vef.actorID, vef.orderID, hbl.ID, hbl.Version, link.Version, &biz.SeaHouseBillInput{HouseNo: hbl.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, Content: &biz.SeaBillContent{}}, vef.audit())
		if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentVoided.Reason {
			t.Fatalf("VOIDED HBL 仍可走普通编辑，错误=%v", err)
		}
	})

	t.Run("DIRECT 切换 HOUSE 建立新当前 HBL", func(t *testing.T) {
		vef := newSeaDocumentChangeFixture(t)
		vefUC := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(vef.data))
		hbl := vef.data.db.SeaHouseBill.GetX(ctx, vef.hblID)
		order := vef.data.db.Order.GetX(ctx, vef.orderID)
		hblVersion := hbl.Version
		toDirect := &biz.SeaDocumentModeChangeCommand{
			OrderID: vef.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 1,
			ExpectedHouseBillVersion: &hblVersion, ExpectedCurrentVersionID: hbl.CurrentVersionID,
			TargetMode: biz.SeaDocumentStructureDirect,
			Reason:     "客户改直单", IdempotencyKey: "mode-d1-" + uuid.NewString(),
			Confirmation: vef.confirmation(),
		}
		if err := vefUC.ExecuteModeChange(ctx, vef.orgID, vef.actorID, toDirect, vef.audit()); err != nil {
			t.Fatalf("切换 DIRECT 失败: %v", err)
		}
		order = vef.data.db.Order.GetX(ctx, vef.orderID)
		newHouseNo := "HBLNEW" + uuid.NewString()[:8]
		toHouse := &biz.SeaDocumentModeChangeCommand{
			OrderID: vef.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 2,
			TargetMode: biz.SeaDocumentStructureHouse,
			NewHouseBill: &biz.SeaHouseBillInput{
				HouseNo: newHouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization,
			},
			Reason: "客户恢复 HOUSE", IdempotencyKey: "mode-d2h-" + uuid.NewString(),
			Confirmation: vef.confirmation(),
		}
		if err := vefUC.ExecuteModeChange(ctx, vef.orgID, vef.actorID, toHouse, vef.audit()); err != nil {
			t.Fatalf("DIRECT 切换 HOUSE 失败: %v", err)
		}
		newHBL, err := vef.data.db.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(vef.orderID), seahousebillent.StatusNEQ(seahousebillent.StatusVOIDED)).
			Only(ctx)
		if err != nil || newHBL.HouseNo != newHouseNo || newHBL.Status != seahousebillent.StatusDRAFT || newHBL.Version != 1 {
			t.Fatalf("新当前 HBL 不符合预期: hbl=%+v err=%v", newHBL, err)
		}
		link := vef.data.db.SeaMasterBillOrderLink.GetX(ctx, vef.linkID)
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			t.Fatalf("Link 模式应回到 HOUSE, 实际: %s", link.DocumentStructure)
		}
	})

	t.Run("模式切换缺少外部确认返回参数错误", func(t *testing.T) {
		vef := newSeaDocumentChangeFixture(t)
		vefUC := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(vef.data))
		hbl := vef.data.db.SeaHouseBill.GetX(ctx, vef.hblID)
		order := vef.data.db.Order.GetX(ctx, vef.orderID)
		hblVersion := hbl.Version
		cmd := &biz.SeaDocumentModeChangeCommand{
			OrderID: vef.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 1,
			ExpectedHouseBillVersion: &hblVersion, ExpectedCurrentVersionID: hbl.CurrentVersionID,
			TargetMode:     biz.SeaDocumentStructureDirect,
			Reason:         "缺少确认",
			IdempotencyKey: "mode-noconfirm-" + uuid.NewString(),
		}
		if err := vefUC.ExecuteModeChange(ctx, vef.orgID, vef.actorID, cmd, vef.audit()); kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentInvalidArgument.Reason {
			t.Fatalf("缺少外部确认应返回参数错误，实际: %v", err)
		}
	})
}

func TestSeaDocumentAmendmentAuditRollback(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	failedAudit := f.audit()
	failedAudit.Result = "invalid-result"
	_, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
		ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
		Reason: "审计失败回滚验证", IdempotencyKey: "audit-rollback-" + uuid.NewString(),
		Input:        &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPtr("不应落库")}},
		Confirmation: f.confirmation(),
	}, failedAudit)
	if err == nil {
		t.Fatal("审计写入失败时改单事务应回滚")
	}
	mbl = f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	if mbl.Version != 1 || mbl.CurrentVersionID == nil || *mbl.CurrentVersionID != f.mblVerID || f.data.db.SeaMasterBillVersion.Query().CountX(ctx) != 1 {
		t.Fatalf("审计失败后工作实体或不可变版本未回滚: mbl=%+v", mbl)
	}
}

func TestSeaDocumentAmendmentConcurrentPublish(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	makeCommand := func(value string) *biz.SeaDocumentAmendmentCommand {
		return &biz.SeaDocumentAmendmentCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: 1, ExpectedDocumentVersion: 1, ExpectedCurrentVersionID: f.mblVerID,
			Reason: "并发改单", IdempotencyKey: "concurrent-" + uuid.NewString(),
			Input:        &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPtr(value)}},
			Confirmation: f.confirmation(),
		}
	}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, value := range []string{"并发版本甲", "并发版本乙"} {
		value := value
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, makeCommand(value), f.audit())
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			successes++
		} else if kratoserrors.IsConflict(err) {
			conflicts++
		} else {
			t.Fatalf("并发改单返回非预期错误: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("并发改单应恰有一个成功：success=%d conflict=%d", successes, conflicts)
	}
	if count := f.data.db.SeaMasterBillVersion.Query().CountX(ctx); count != 2 {
		t.Fatalf("并发改单产生了重复或缺失版本：count=%d", count)
	}
}

func TestSeaDocumentAmendmentConcurrentIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	cmd := &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
		ExpectedOrderVersion: 1, ExpectedDocumentVersion: 1, ExpectedCurrentVersionID: f.mblVerID,
		Reason: "并发幂等改单", IdempotencyKey: "concurrent-idempotent-" + uuid.NewString(),
		Input:        &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPtr("并发幂等版本")}},
		Confirmation: f.confirmation(),
	}
	type result struct {
		versionID uuid.UUID
		err       error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			version, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
			var id uuid.UUID
			if version != nil {
				id = version.ID
			}
			results <- result{versionID: id, err: err}
		}()
	}
	wg.Wait()
	close(results)
	var publishedID uuid.UUID
	for item := range results {
		if item.err != nil {
			t.Fatalf("相同幂等请求并发重放失败: %v", item.err)
		}
		if publishedID == uuid.Nil {
			publishedID = item.versionID
		} else if item.versionID != publishedID {
			t.Fatalf("相同幂等请求返回不同版本: first=%s current=%s", publishedID, item.versionID)
		}
	}
	if count := f.data.db.SeaMasterBillVersion.Query().CountX(ctx); count != 2 {
		t.Fatalf("相同幂等请求并发产生重复版本: count=%d", count)
	}
}

func TestSeaMasterBillMemberSetRevalidatedAfterLock(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	expectedIDs, activeLinkID, err := locateMasterMemberOrderIDs(ctx, f.data.db, f.orgID, f.orderID, f.mblID)
	if err != nil {
		t.Fatalf("定位初始 MBL 成员失败: %v", err)
	}
	member := f.data.db.Order.Create().SetOrganizationID(f.orgID).SetOrderNo("SE-MEMBER-" + uuid.NewString()[:8]).SetCustomerID(f.partnerID).SetBusinessType("SE").SetTradeDirection("export").SetTradeTerm("FOB").SetPaymentTerm("PREPAID").SetIdempotencyKey(uuid.NewString()).SaveX(ctx)
	extraLink := f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.orgID).SetOrderID(member.ID).SetMasterBillID(f.mblID).SetTransportExecutionID(f.data.db.SeaMasterBillOrderLink.GetX(ctx, activeLinkID).TransportExecutionID).SetStatus(seamasterbillorderlinkent.StatusACTIVE).SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).SaveX(ctx)
	t.Cleanup(func() { _ = f.data.db.SeaMasterBillOrderLink.DeleteOneID(extraLink.ID).Exec(ctx) })
	err = f.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(f.mblID)).ForUpdate().Only(ctx); err != nil {
			return err
		}
		_, err := lockAndValidateMasterMemberLinks(ctx, tx, f.orgID, f.orderID, f.mblID, activeLinkID, expectedIDs)
		return err
	})
	if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentStructureConflict.Reason {
		t.Fatalf("MBL 锁后成员集合变化未返回结构冲突: %v", err)
	}
}

func TestSeaDocumentDownstreamFactsBlockExecution(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	suffix := uuid.NewString()[:8]
	fee := f.data.db.OrderFee.Create().SetOrderID(f.orderID).SetIdempotencyKey("historical-fee-" + suffix).SetDirection(orderfeeent.DirectionRECEIVABLE).SetStatus(orderfeeent.StatusBILLED).SetFeeCode("HIS-FEE-" + suffix).SetFeeName("历史财务事实费用").SetSettlementPartyID(f.partnerID).SetBillingUnit("BILL").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-04").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetExpenseDate("2026-09-04").SaveX(ctx)
	billCreate := f.data.db.FinanceBill.Create().SetOrganizationID(f.orgID).SetBillNo("HIS-BILL-" + suffix).SetIdempotencyKey("historical-bill-" + suffix).SetDirection(financebillent.DirectionRECEIVABLE).SetStatus(financebillent.StatusDRAFT).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("单证变更测试合作伙伴").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-04").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").SetFeeCount(1).SetBillDate("2026-09-04")
	bill := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").SaveX(ctx)
	f.data.db.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderID(f.orderID).SetOrderFeeID(fee.ID).SetOrderNo("HISTORICAL").SetFeeCode(fee.FeeCode).SetFeeName(fee.FeeName).SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrencyAmount("100").SetBaseCurrency("CNY").SetActive(true).SaveX(ctx)
	invoice := f.data.db.FinanceInvoice.Create().SetOrganizationID(f.orgID).SetRecordNo("HIS-INV-" + suffix).SetIdempotencyKey("historical-invoice-" + suffix).SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetStatus(financeinvoiceent.StatusDRAFT).SetInvoiceType(financeinvoiceent.InvoiceTypeNORMAL).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("单证变更测试合作伙伴").SetCurrency("CNY").SetBaseCurrency("CNY").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBillCount(1).SaveX(ctx)
	f.data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).SetAmount("100").SetTaxAmount("0").SetActive(true).SaveX(ctx)

	mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	voidCmd := &biz.SeaDocumentVoidCommand{OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID, ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID, Reason: "下游事实阻断", IdempotencyKey: "historical-impact-" + suffix, Confirmation: f.confirmation()}
	preview, err := uc.PreviewVoid(ctx, f.orgID, voidCmd)
	if err != nil {
		t.Fatalf("存在下游财务事实时预览失败: %v", err)
	}
	facts := map[string]string{}
	for _, impact := range preview.Impacts {
		facts[impact.FactType] = impact.ReferenceNo
		if !impact.BlocksExecution {
			t.Fatalf("下游事实 %s 必须标记阻断执行: %+v", impact.FactType, impact)
		}
	}
	if facts["ORDER_FEE"] != fee.FeeCode || facts["FINANCE_BILL"] != bill.BillNo || facts["FINANCE_INVOICE"] != invoice.RecordNo {
		t.Fatalf("历史费用/账单/发票未进入影响摘要: %+v", facts)
	}
	if preview.Executable {
		t.Fatalf("存在下游财务事实时 Preview 不可执行: %+v", preview.Impacts)
	}
	_, err = uc.ExecuteVoid(ctx, f.orgID, f.actorID, voidCmd, f.audit())
	if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentChangeBlocked.Reason || !kratoserrors.IsConflict(err) {
		t.Fatalf("存在下游财务事实时作废必须被 409 阻断: %v", err)
	}
	blocked := kratoserrors.FromError(err)
	if blocked.Metadata == nil || blocked.Metadata["blocked_count"] == "" || blocked.Metadata["fact_type"] != "ORDER_FEE" || blocked.Metadata["reference_id"] != fee.ID.String() {
		t.Fatalf("阻断错误缺少 blocked_count/fact_type/reference_id 元数据: %+v", blocked.Metadata)
	}
	// 单证与财务事实保持原归属不被改写
	mblAfter := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	if mblAfter.Status == seamasterbillent.StatusVOIDED || mblAfter.Version != mbl.Version {
		t.Fatalf("被阻断的作废改写了工作实体: %+v", mblAfter)
	}
	feeAfter := f.data.db.OrderFee.GetX(ctx, fee.ID)
	if feeAfter.OrderID != f.orderID || feeAfter.Status != orderfeeent.StatusBILLED {
		t.Fatalf("阻断路径改写了财务事实: %+v", feeAfter)
	}
}

func TestSeaDocumentModeChangeBlockedByDownstreamFacts(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	suffix := uuid.NewString()[:8]
	f.data.db.OrderFee.Create().SetOrderID(f.orderID).SetIdempotencyKey("mode-block-fee-" + suffix).SetDirection(orderfeeent.DirectionRECEIVABLE).SetStatus(orderfeeent.StatusBILLED).SetFeeCode("MODE-FEE-" + suffix).SetFeeName("模式切换阻断费用").SetSettlementPartyID(f.partnerID).SetBillingUnit("BILL").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-09-04").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetExpenseDate("2026-09-04").SaveX(ctx)

	hbl := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	hblVersion := hbl.Version
	cmd := &biz.SeaDocumentModeChangeCommand{
		OrderID: f.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 1,
		ExpectedHouseBillVersion: &hblVersion, ExpectedCurrentVersionID: hbl.CurrentVersionID,
		TargetMode:     biz.SeaDocumentStructureDirect,
		Reason:         "存在下游事实时模式切换应被阻断",
		IdempotencyKey: "mode-blocked-" + suffix,
		Confirmation:   f.confirmation(),
	}
	preview, err := uc.PreviewModeChange(ctx, f.orgID, cmd)
	if err != nil {
		t.Fatalf("模式切换预览失败: %v", err)
	}
	if preview.Executable {
		t.Fatalf("存在下游财务事实时模式切换预览不可执行: %+v", preview.Impacts)
	}
	if err = uc.ExecuteModeChange(ctx, f.orgID, f.actorID, cmd, f.audit()); kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentChangeBlocked.Reason {
		t.Fatalf("存在下游财务事实时模式切换必须被阻断，实际: %v", err)
	}
	hblAfter := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	if hblAfter.Status == seahousebillent.StatusVOIDED {
		t.Fatal("被阻断的模式切换把 HBL 置为 VOIDED")
	}
	linkAfter := f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID)
	if linkAfter.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
		t.Fatalf("被阻断的模式切换改写了单证模式: %s", linkAfter.DocumentStructure)
	}
}

// TestSeaDocumentModeChangeBlockedByHouseBillAllocation 验证 HOUSE→DIRECT 模式切换
// 与 HBL 改单同口径：当前活动 HBL 存在箱货分配时，Preview 不可执行、Execute 409
// 阻断（metadata 指向分配事实），HBL 不被作废。
func TestSeaDocumentModeChangeBlockedByHouseBillAllocation(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	suffix := uuid.NewString()[:8]
	container := createHouseBillSharedAllocation(t, f, "MODE-"+suffix)

	hbl := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	hblVersion := hbl.Version
	cmd := &biz.SeaDocumentModeChangeCommand{
		OrderID: f.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 1,
		ExpectedHouseBillVersion: &hblVersion, ExpectedCurrentVersionID: hbl.CurrentVersionID,
		TargetMode:     biz.SeaDocumentStructureDirect,
		Reason:         "箱货分配阻断模式切换",
		IdempotencyKey: "mode-alloc-block-" + suffix,
		Confirmation:   f.confirmation(),
	}
	preview, err := uc.PreviewModeChange(ctx, f.orgID, cmd)
	if err != nil {
		t.Fatalf("模式切换预览失败: %v", err)
	}
	foundAllocation := false
	for _, impact := range preview.Impacts {
		if impact.FactType == "SEA_SHARED_CONTAINER_ALLOCATION" && impact.ReferenceNo == container.ContainerNo && impact.BlocksExecution {
			foundAllocation = true
		}
	}
	if !foundAllocation {
		t.Fatalf("箱货分配未进入模式切换下游影响: %+v", preview.Impacts)
	}
	if preview.Executable {
		t.Fatalf("存在箱货分配时模式切换预览不可执行: %+v", preview.Impacts)
	}
	err = uc.ExecuteModeChange(ctx, f.orgID, f.actorID, cmd, f.audit())
	if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentChangeBlocked.Reason || !kratoserrors.IsConflict(err) {
		t.Fatalf("存在箱货分配时模式切换必须被 409 阻断，实际: %v", err)
	}
	blocked := kratoserrors.FromError(err)
	if blocked.Metadata == nil || blocked.Metadata["fact_type"] != "SEA_SHARED_CONTAINER_ALLOCATION" || blocked.Metadata["reference_no"] != container.ContainerNo {
		t.Fatalf("阻断错误未指向箱货分配事实: %+v", blocked.Metadata)
	}
	hblAfter := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	if hblAfter.Status == seahousebillent.StatusVOIDED {
		t.Fatal("被阻断的模式切换把 HBL 置为 VOIDED，分配将悬挂指向已作废 HBL")
	}
	linkAfter := f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID)
	if linkAfter.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
		t.Fatalf("被阻断的模式切换改写了单证模式: %s", linkAfter.DocumentStructure)
	}
}

// createHouseBillSharedAllocation 为夹具的当前 HBL 落一条共享箱货分配，返回共享箱。
func createHouseBillSharedAllocation(t *testing.T, f *seaDocumentChangeFixture, suffix string) *ent.SeaSharedContainer {
	t.Helper()
	ctx := context.Background()
	spec := f.data.db.MasterDataItem.Create().
		SetKind(masterdataitement.KindContainerSpec).
		SetCode("40HQ-" + suffix).
		SetName("40HQ超高箱").
		SetSortOrder(1).
		SetEnabled(true).
		SaveX(ctx)
	cargoItem := f.data.db.OrderCargoItem.Create().
		SetOrganizationID(f.orgID).
		SetOrderID(f.orderID).
		SetCargoName("共享箱分配阻断货物").
		SetPackageCount(10).
		SetGrossWeightKg(100.0).
		SetVolumeCbm(1.0).
		SetVersion(1).
		SaveX(ctx)
	container := f.data.db.SeaSharedContainer.Create().
		SetOrganizationID(f.orgID).
		SetTransportExecutionID(f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID).TransportExecutionID).
		SetContainerNo("SHCU-BLK-" + suffix).
		SetContainerSpecID(spec.ID).
		SetPackageCount(10).
		SetGrossWeightKg("100.000").
		SetVolumeCbm("1.000000").
		SetVersion(1).
		SaveX(ctx)
	f.data.db.SeaSharedContainerAllocation.Create().
		SetOrganizationID(f.orgID).
		SetSharedContainerID(container.ID).
		SetOrderID(f.orderID).
		SetHouseBillID(f.hblID).
		SetCargoItemID(cargoItem.ID).
		SetPackageCount(10).
		SetGrossWeightKg("100.000").
		SetVolumeCbm("1.000000").
		SetVersion(1).
		SaveX(ctx)
	return container
}

func TestSeaDocumentHouseBillAllocationBlocksAmendment(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	suffix := uuid.NewString()[:8]
	container := createHouseBillSharedAllocation(t, f, suffix)

	hbl := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	cmd := &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: f.hblID,
		ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: hbl.Version, ExpectedCurrentVersionID: *hbl.CurrentVersionID,
		Reason: "箱货分配阻断改单", IdempotencyKey: "hbl-alloc-block-" + suffix,
		Input: &biz.SeaDocumentAmendmentInput{HouseBill: &biz.SeaHouseBillInput{
			HouseNo: hbl.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner,
			Content: &biz.SeaBillContent{ShipperText: stringPtr("共享箱分配下的新发货人")},
		}},
		Confirmation: f.confirmation(),
	}
	preview, err := uc.PreviewAmendment(ctx, f.orgID, cmd)
	if err != nil {
		t.Fatalf("HBL 改单预览失败: %v", err)
	}
	foundAllocation := false
	for _, impact := range preview.Impacts {
		if impact.FactType == "SEA_SHARED_CONTAINER_ALLOCATION" && impact.ReferenceNo == container.ContainerNo && impact.BlocksExecution {
			foundAllocation = true
		}
	}
	if !foundAllocation {
		t.Fatalf("箱货分配未进入 HBL 下游影响: %+v", preview.Impacts)
	}
	if preview.Executable {
		t.Fatalf("存在箱货分配时 HBL 改单预览不可执行: %+v", preview.Impacts)
	}
	if _, err = uc.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit()); kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentChangeBlocked.Reason {
		t.Fatalf("存在箱货分配时 HBL 改单必须被阻断，实际: %v", err)
	}
	hblAfter := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
	if hblAfter.Version != hbl.Version {
		t.Fatalf("被阻断的 HBL 改单改写了版本: %d", hblAfter.Version)
	}
}

func TestSeaDocumentChangeBusinessLockAndLifecycleGate(t *testing.T) {
	ctx := context.Background()
	masterAmendCmd := func(f *seaDocumentChangeFixture) *biz.SeaDocumentAmendmentCommand {
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		return &biz.SeaDocumentAmendmentCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
			Reason: "门禁阻断验证", IdempotencyKey: "gate-" + uuid.NewString(),
			Input:        &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPtr("门禁下不应落库")}},
			Confirmation: f.confirmation(),
		}
	}

	t.Run("业务锁定订单拒绝改单", func(t *testing.T) {
		f := newSeaDocumentChangeFixture(t)
		uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
		f.data.db.Order.UpdateOneID(f.orderID).SetLockedAt(time.Now().UTC()).SetLockedBy(f.actorID).SetLockGeneration(1).SaveX(ctx)
		cmd := masterAmendCmd(f)
		preview, err := uc.PreviewAmendment(ctx, f.orgID, cmd)
		if err != nil {
			t.Fatalf("改单预览不应报错，实际: %v", err)
		}
		if preview.Executable {
			t.Fatalf("业务锁定订单改单预览 Executable 应为 false")
		}
		foundLockImpact := false
		for _, imp := range preview.Impacts {
			if imp.FactType == "ORDER_BUSINESS_LOCK" && imp.BlocksExecution {
				foundLockImpact = true
				break
			}
		}
		if !foundLockImpact {
			t.Fatalf("业务锁定订单改单预览未包含 ORDER_BUSINESS_LOCK 阻断事实: %+v", preview.Impacts)
		}
		if _, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit()); kratoserrors.FromError(err).Reason != "ORDER_BUSINESS_LOCKED" {
			t.Fatalf("业务锁定订单改单应返回 ORDER_BUSINESS_LOCKED，实际: %v", err)
		}
	})

	t.Run("业务锁定订单拒绝模式切换", func(t *testing.T) {
		f := newSeaDocumentChangeFixture(t)
		uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
		f.data.db.Order.UpdateOneID(f.orderID).SetLockedAt(time.Now().UTC()).SetLockedBy(f.actorID).SetLockGeneration(1).SaveX(ctx)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		hbl := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
		hblVersion := hbl.Version
		cmd := &biz.SeaDocumentModeChangeCommand{
			OrderID: f.orderID, ExpectedOrderVersion: order.Version, ExpectedLinkVersion: 1,
			ExpectedHouseBillVersion: &hblVersion, ExpectedCurrentVersionID: hbl.CurrentVersionID,
			TargetMode: biz.SeaDocumentStructureDirect, Reason: "锁定下模式切换",
			IdempotencyKey: "mode-locked-" + uuid.NewString(), Confirmation: f.confirmation(),
		}
		preview, err := uc.PreviewModeChange(ctx, f.orgID, cmd)
		if err != nil {
			t.Fatalf("模式切换预览不应报错，实际: %v", err)
		}
		if preview.Executable {
			t.Fatalf("业务锁定订单模式切换预览 Executable 应为 false")
		}
		foundLockImpact := false
		for _, imp := range preview.Impacts {
			if imp.FactType == "ORDER_BUSINESS_LOCK" && imp.BlocksExecution {
				foundLockImpact = true
				break
			}
		}
		if !foundLockImpact {
			t.Fatalf("业务锁定订单模式切换预览未包含 ORDER_BUSINESS_LOCK 阻断事实: %+v", preview.Impacts)
		}
		if err := uc.ExecuteModeChange(ctx, f.orgID, f.actorID, cmd, f.audit()); kratoserrors.FromError(err).Reason != "ORDER_BUSINESS_LOCKED" {
			t.Fatalf("业务锁定订单模式切换应返回 ORDER_BUSINESS_LOCKED，实际: %v", err)
		}
	})

	t.Run("终止流程订单拒绝改单", func(t *testing.T) {
		f := newSeaDocumentChangeFixture(t)
		uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
		f.data.db.Order.UpdateOneID(f.orderID).
			SetTerminationStatus(orderent.TerminationStatusTERMINATING).
			SetTerminationType(orderent.TerminationTypeCARRIER_CANCEL).
			SetTerminationReason("门禁测试进入终止流程").
			SaveX(ctx)
		if _, err := uc.ExecuteAmendment(ctx, f.orgID, f.actorID, masterAmendCmd(f), f.audit()); kratoserrors.FromError(err).Reason != biz.ErrOrderTerminationInProgress.Reason {
			t.Fatalf("终止流程订单改单应返回 ORDER_TERMINATION_IN_PROGRESS，实际: %v", err)
		}
	})

	t.Run("已结案订单拒绝作废", func(t *testing.T) {
		f := newSeaDocumentChangeFixture(t)
		uc := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
		f.data.db.Order.UpdateOneID(f.orderID).
			SetClosureStatus(orderent.ClosureStatusCLOSED).
			SetClosureReason("门禁测试结案").
			SetClosedAt(time.Now().UTC()).
			SetClosedBy(f.actorID).
			SaveX(ctx)
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		cmd := &biz.SeaDocumentVoidCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
			Reason: "结案订单作废", IdempotencyKey: "void-closed-" + uuid.NewString(), Confirmation: f.confirmation(),
		}
		preview, err := uc.PreviewVoid(ctx, f.orgID, cmd)
		if err != nil {
			t.Fatalf("作废预览不应报错，实际: %v", err)
		}
		if preview.Executable {
			t.Fatalf("已结案订单作废预览 Executable 应为 false")
		}
		foundLockImpact := false
		for _, imp := range preview.Impacts {
			if imp.FactType == "ORDER_BUSINESS_LOCK" && imp.BlocksExecution {
				foundLockImpact = true
				break
			}
		}
		if !foundLockImpact {
			t.Fatalf("已结案订单作废预览未包含 ORDER_BUSINESS_LOCK 阻断事实: %+v", preview.Impacts)
		}
		if _, err := uc.ExecuteVoid(ctx, f.orgID, f.actorID, cmd, f.audit()); kratoserrors.FromError(err).Reason != biz.ErrOrderClosed.Reason {
			t.Fatalf("已结案订单作废应返回 ORDER_CLOSED，实际: %v", err)
		}
	})
}
