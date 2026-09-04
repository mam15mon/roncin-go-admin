package data

import (
	"context"
	"sync"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type seaDocumentChangeFixture struct {
	data      *Data
	orgID     uuid.UUID
	actorID   uuid.UUID
	partnerID uuid.UUID
	orderID   uuid.UUID
	mblID     uuid.UUID
	mblVerID  uuid.UUID
	hblID     uuid.UUID
	hblVerID  uuid.UUID
	switchID  uuid.UUID
	switchVer uuid.UUID
}

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
	order := data.db.Order.Create().
		SetOrganizationID(org.ID).
		SetOrderNo("SE-DOC-" + suffix).
		SetCustomerID(partner.ID).
		SetBusinessType("SE").
		SetTradeDirection("export").
		SetTradeTerm("FOB").
		SetPaymentTerm("PREPAID").
		SaveX(ctx)
	exec := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetVesselName("EVER TEST").
		SetVoyageNo("V001").
		SaveX(ctx)
	mbl := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetIssuerPartnerID(partner.ID).
		SetTransportExecutionID(exec.ID).
		SetMasterNo("MBL-" + suffix).
		SetNormalizedMasterNo("MBL-" + suffix).
		SetShipperText("旧主单发货人").
		SaveX(ctx)
	data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(org.ID).
		SetOrderID(order.ID).
		SetMasterBillID(mbl.ID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SaveX(ctx)
	newHouseBill := func(no string) *ent.SeaHouseBill {
		return data.db.SeaHouseBill.Create().
			SetOrganizationID(org.ID).
			SetOrderID(order.ID).
			SetMasterBillID(mbl.ID).
			SetHouseNo(no).
			SetNormalizedHouseNo(no).
			SetIssuerSource(seahousebillent.IssuerSourceCUSTOMER_PARTNER).
			SetIssuerPartnerID(partner.ID).
			SetShipperText("旧分单发货人").
			SaveX(ctx)
	}
	hbl := newHouseBill("HBL-A-" + suffix)
	switchHBL := newHouseBill("HBL-S-" + suffix)

	f := &seaDocumentChangeFixture{data: data, orgID: org.ID, actorID: actor.ID, partnerID: partner.ID, orderID: order.ID, mblID: mbl.ID, hblID: hbl.ID, switchID: switchHBL.ID}
	err := data.WithTx(ctx, func(tx *ent.Tx) error {
		txMBL, err := tx.SeaMasterBill.Get(ctx, mbl.ID)
		if err != nil {
			return err
		}
		txExec, err := tx.SeaTransportExecution.Get(ctx, exec.ID)
		if err != nil {
			return err
		}
		mblVersion, err := createMasterVersion(ctx, tx, txMBL, txExec, actor.ID, biz.VersionSourceOrderLock, nil, nil, nil)
		if err != nil {
			return err
		}
		if _, err = txMBL.Update().SetCurrentVersionID(mblVersion.ID).Save(ctx); err != nil {
			return err
		}
		f.mblVerID = mblVersion.ID
		for _, item := range []struct {
			id     uuid.UUID
			result *uuid.UUID
		}{{hbl.ID, &f.hblVerID}, {switchHBL.ID, &f.switchVer}} {
			txHBL, err := tx.SeaHouseBill.Get(ctx, item.id)
			if err != nil {
				return err
			}
			version, err := createHouseVersion(ctx, tx, txHBL, actor.ID, biz.VersionSourceOrderLock, nil, nil, nil)
			if err != nil {
				return err
			}
			if _, err = txHBL.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
				return err
			}
			*item.result = version.ID
		}
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

func TestSeaDocumentChangePostgresFlows(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	repo := NewSeaDocumentChangeRepo(f.data)

	t.Run("MBL 单改追加不可变版本且幂等", func(t *testing.T) {
		cmd := &biz.SeaDocumentAmendmentCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: 1, ExpectedDocumentVersion: 1, ExpectedCurrentVersionID: f.mblVerID,
			Reason: "客户更正发货人", IdempotencyKey: "mbl-amend-" + uuid.NewString(),
			Input: &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPointer("新主单发货人")}},
		}
		preview, err := repo.PreviewAmendment(ctx, f.orgID, cmd)
		if err != nil || !preview.Executable || len(preview.Differences) != 1 || preview.BaseVersion.ID != f.mblVerID {
			t.Fatalf("MBL改单预览不符合预期: preview=%+v err=%v", preview, err)
		}
		result, err := repo.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
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
		idempotent, err := repo.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil || idempotent.ID != result.ID {
			t.Fatalf("MBL 单改幂等重试失败: result=%+v err=%v", idempotent, err)
		}
		f.mblVerID = result.ID
	})

	t.Run("HBL 单改与作废保留完整历史", func(t *testing.T) {
		hbl := f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		cmd := &biz.SeaDocumentAmendmentCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: f.hblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: hbl.Version, ExpectedCurrentVersionID: *hbl.CurrentVersionID,
			Reason: "客户更正分单发货人", IdempotencyKey: "hbl-amend-" + uuid.NewString(),
			Input: &biz.SeaDocumentAmendmentInput{HouseBill: &biz.SeaHouseBillInput{
				HouseNo: hbl.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner,
				Content: &biz.SeaBillContent{ShipperText: stringPointer("新分单发货人")},
			}},
		}
		preview, err := repo.PreviewAmendment(ctx, f.orgID, cmd)
		if err != nil || !preview.Executable || len(preview.Differences) != 1 {
			t.Fatalf("HBL改单预览不符合预期: preview=%+v err=%v", preview, err)
		}
		amended, err := repo.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil {
			t.Fatalf("执行 HBL 单改失败: %v", err)
		}
		old := f.data.db.SeaHouseBillVersion.GetX(ctx, f.hblVerID)
		if old.ShipperText == nil || *old.ShipperText != "旧分单发货人" {
			t.Fatal("旧 HBL 版本被修改")
		}

		hbl = f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
		order = f.data.db.Order.GetX(ctx, f.orderID)
		voidCmd := &biz.SeaDocumentVoidCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: f.hblID,
			ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: hbl.Version, ExpectedCurrentVersionID: amended.ID,
			Reason: "客户确认作废", IdempotencyKey: "hbl-void-" + uuid.NewString(),
		}
		voidPreview, err := repo.PreviewVoid(ctx, f.orgID, voidCmd)
		if err != nil || !voidPreview.Executable || len(voidPreview.Differences) != 1 {
			t.Fatalf("HBL作废预览不符合预期: preview=%+v err=%v", voidPreview, err)
		}
		event, err := repo.ExecuteVoid(ctx, f.orgID, f.actorID, voidCmd, f.audit())
		if err != nil {
			t.Fatalf("执行 HBL 作废失败: %v", err)
		}
		hbl = f.data.db.SeaHouseBill.GetX(ctx, f.hblID)
		if hbl.Status != seahousebillent.StatusVOIDED || hbl.CurrentVersionID == nil || event.ResultVersionID == nil || *hbl.CurrentVersionID != *event.ResultVersionID {
			t.Fatalf("HBL作废状态或当前版本错误: hbl=%+v event=%+v", hbl, event)
		}
		if count := f.data.db.SeaHouseBillVersion.Query().Where().CountX(ctx); count < 4 {
			t.Fatalf("作废未追加不可变版本，当前版本总数=%d", count)
		}
		link := f.data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(f.orderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).OnlyX(ctx)
		documentRepo := NewSeaDocumentRepo(f.data)
		_, err = documentRepo.UpdateSeaHouseBill(ctx, f.orgID, f.actorID, f.orderID, hbl.ID, hbl.Version, link.Version, &biz.SeaHouseBillInput{HouseNo: hbl.HouseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, Content: &biz.SeaBillContent{}}, f.audit())
		if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentVoided.Reason {
			t.Fatalf("VOIDED HBL 仍可走普通编辑，错误=%v", err)
		}
		err = documentRepo.RemoveSeaHouseBill(ctx, f.orgID, f.actorID, f.orderID, hbl.ID, hbl.Version, link.Version, true, f.audit())
		if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentVoided.Reason {
			t.Fatalf("VOIDED HBL 仍可走普通删除，错误=%v", err)
		}
		_, err = NewSeaCargoAllocationRepo(f.data).ApplyHouseBillSummary(ctx, f.orgID, f.actorID, f.orderID, hbl.ID, link.CargoAllocationVersion, hbl.Version, f.audit())
		if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentVoided.Reason {
			t.Fatalf("VOIDED HBL 仍可通过箱货汇总改写，错误=%v", err)
		}
	})

	t.Run("Switch 创建真实 HBL 并形成唯一替代链", func(t *testing.T) {
		old := f.data.db.SeaHouseBill.GetX(ctx, f.switchID)
		originalOldNo := old.HouseNo
		order := f.data.db.Order.GetX(ctx, f.orderID)
		cmd := &biz.SeaHouseBillSwitchCommand{
			OrderID: f.orderID, OldHouseBillID: old.ID, ExpectedOrderVersion: order.Version,
			ExpectedHouseBillVersion: old.Version, ExpectedCurrentVersionID: *old.CurrentVersionID,
			Reason: "船公司要求换单", IdempotencyKey: "hbl-switch-" + uuid.NewString(),
			NewHouseBill: &biz.SeaHouseBillInput{HouseNo: " hbl-new-001 ", IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, Content: &biz.SeaBillContent{ShipperText: stringPointer("新换单发货人")}},
		}
		preview, err := repo.PreviewSwitch(ctx, f.orgID, cmd)
		if err != nil || !preview.Executable || len(preview.Differences) == 0 {
			t.Fatalf("Switch预览不符合预期: preview=%+v err=%v", preview, err)
		}
		result, err := repo.ExecuteSwitch(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil {
			t.Fatalf("执行 Switch 失败: %v", err)
		}
		old = f.data.db.SeaHouseBill.GetX(ctx, f.switchID)
		if old.Status != seahousebillent.StatusREPLACED || result.NewHouseBill == nil || result.NewHouseBill.ID == old.ID || result.NewHouseBill.NormalizedHouseNo != "HBL-NEW-001" {
			t.Fatalf("Switch 新旧 HBL 状态错误: old=%+v result=%+v", old, result)
		}
		firstNewNo := result.NewHouseBill.HouseNo
		idempotent, err := repo.ExecuteSwitch(ctx, f.orgID, f.actorID, cmd, f.audit())
		if err != nil || idempotent.Event.ID != result.Event.ID || idempotent.NewHouseBill.ID != result.NewHouseBill.ID {
			t.Fatalf("Switch 幂等重试失败: result=%+v err=%v", idempotent, err)
		}
		current := f.data.db.SeaHouseBill.GetX(ctx, result.NewHouseBill.ID)
		order = f.data.db.Order.GetX(ctx, f.orderID)
		secondCmd := &biz.SeaHouseBillSwitchCommand{
			OrderID: f.orderID, OldHouseBillID: current.ID, ExpectedOrderVersion: order.Version,
			ExpectedHouseBillVersion: current.Version, ExpectedCurrentVersionID: *current.CurrentVersionID,
			Reason: "再次换单", IdempotencyKey: "hbl-switch-second-" + uuid.NewString(),
			NewHouseBill: &biz.SeaHouseBillInput{HouseNo: "HBL-NEW-002", IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, Content: &biz.SeaBillContent{ShipperText: stringPointer("二次换单发货人")}},
		}
		second, err := repo.ExecuteSwitch(ctx, f.orgID, f.actorID, secondCmd, f.audit())
		if err != nil {
			t.Fatalf("执行二次 Switch 失败: %v", err)
		}
		if second.Event.ChainID == nil || result.Event.ChainID == nil || *second.Event.ChainID != *result.Event.ChainID || second.Event.Sequence == nil || *second.Event.Sequence != 2 {
			t.Fatalf("二次 Switch 未延续唯一替代链: first=%+v second=%+v", result.Event, second.Event)
		}
		current = f.data.db.SeaHouseBill.GetX(ctx, current.ID)
		if current.Status != seahousebillent.StatusREPLACED {
			t.Fatalf("替代链中间 HBL 未置为 REPLACED: %s", current.Status)
		}
		// 即使工作身份字段被底层维护改动，历史事件仍必须只读取事件绑定的不可变版本。
		f.data.db.SeaHouseBill.UpdateOneID(old.ID).SetHouseNo("MUTATED-OLD").SetNormalizedHouseNo("MUTATED-OLD").ExecX(ctx)
		f.data.db.SeaHouseBill.UpdateOneID(current.ID).SetHouseNo("MUTATED-MIDDLE").SetNormalizedHouseNo("MUTATED-MIDDLE").ExecX(ctx)
		events, total, err := repo.ListDocumentEvents(ctx, f.orgID, f.orderID, 1, 200)
		if err != nil || total < 5 || len(events) != total {
			t.Fatalf("事件历史查询失败: total=%d len=%d err=%v", total, len(events), err)
		}
		for _, event := range events {
			if event.EventType != biz.SeaDocumentEventTypeSwitch || event.Sequence == nil {
				continue
			}
			if *event.Sequence == 1 && (event.OldHouseNo == nil || *event.OldHouseNo != originalOldNo || event.NewHouseNo == nil || *event.NewHouseNo != firstNewNo) {
				t.Fatalf("Switch 事件未读取首轮不可变版本号: %+v", event)
			}
			if *event.Sequence == 2 && (event.OldHouseNo == nil || *event.OldHouseNo != firstNewNo) {
				t.Fatalf("二次 Switch 事件未读取链中不可变版本号: %+v", event)
			}
		}
	})

	t.Run("财务事实明确阻断且返回类型和编号", func(t *testing.T) {
		f.data.db.OrderFee.Create().
			SetOrderID(f.orderID).
			SetIdempotencyKey("confirmed-fee").
			SetDirection(orderfeeent.DirectionRECEIVABLE).
			SetStatus(orderfeeent.StatusCONFIRMED).
			SetFeeCode("DOC-FEE-001").
			SetFeeName("单证测试费用").
			SetSettlementPartyID(f.partnerID).
			SetBillingUnit("BILL").
			SetQuantity("1").
			SetUnitPrice("100").
			SetTotalAmount("100").
			SetNetAmount("100").
			SetTaxAmount("0").
			SetCurrency("CNY").
			SetExchangeRate("1").
			SetExchangeRateSource(orderfeeent.ExchangeRateSourceBASE_CURRENCY).
			SetExchangeRateDate("2026-09-04").
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount("100").
			SetExpenseDate("2026-09-04").
			SaveX(ctx)
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		order := f.data.db.Order.GetX(ctx, f.orderID)
		cmd := &biz.SeaDocumentVoidCommand{OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID, ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID, Reason: "尝试作废主单", IdempotencyKey: "blocked-void-" + uuid.NewString()}
		preview, err := repo.PreviewVoid(ctx, f.orgID, cmd)
		if err != nil || preview.Executable || len(preview.Impacts) == 0 || preview.Impacts[0].FactType != "ORDER_FEE" || preview.Impacts[0].ReferenceNo != "DOC-FEE-001" {
			t.Fatalf("财务阻断预览不符合预期: preview=%+v err=%v", preview, err)
		}
		_, err = repo.ExecuteVoid(ctx, f.orgID, f.actorID, cmd, f.audit())
		if !kratoserrors.IsConflict(err) {
			t.Fatalf("财务阻断执行应返回冲突: %v", err)
		}
		metadata := kratoserrors.FromError(err).Metadata
		if metadata["fact_type"] != "ORDER_FEE" || metadata["reference_no"] != "DOC-FEE-001" {
			t.Fatalf("财务阻断错误缺少事实类型或编号: %+v", metadata)
		}
	})
}

func TestSeaMasterBillVoidPostgresFlow(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	repo := NewSeaDocumentChangeRepo(f.data)
	mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	failedAudit := f.audit()
	failedAudit.Result = "invalid-result"
	_, err := repo.ExecuteAmendment(ctx, f.orgID, f.actorID, &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
		ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
		Reason: "审计失败回滚验证", IdempotencyKey: "audit-rollback-" + uuid.NewString(),
		Input: &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPointer("不应落库")}},
	}, failedAudit)
	if err == nil {
		t.Fatal("审计写入失败时改单事务应回滚")
	}
	mbl = f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	if mbl.Version != 1 || mbl.CurrentVersionID == nil || *mbl.CurrentVersionID != f.mblVerID || f.data.db.SeaMasterBillVersion.Query().CountX(ctx) != 1 {
		t.Fatalf("审计失败后工作实体或不可变版本未回滚: mbl=%+v", mbl)
	}
	cmd := &biz.SeaDocumentVoidCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
		ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
		Reason: "船公司确认主单作废", IdempotencyKey: "mbl-void-" + uuid.NewString(),
	}
	preview, err := repo.PreviewVoid(ctx, f.orgID, cmd)
	if err != nil || !preview.Executable || len(preview.Differences) != 1 || preview.BaseVersion.ID != f.mblVerID {
		t.Fatalf("MBL作废预览不符合预期: preview=%+v err=%v", preview, err)
	}
	event, err := repo.ExecuteVoid(ctx, f.orgID, f.actorID, cmd, f.audit())
	if err != nil {
		t.Fatalf("执行 MBL 作废失败: %v", err)
	}
	mbl = f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	if string(mbl.Status) != "VOIDED" || mbl.CurrentVersionID == nil || event.ResultVersionID == nil || *mbl.CurrentVersionID != *event.ResultVersionID || event.PreviousVersionID == nil || *event.PreviousVersionID != f.mblVerID {
		t.Fatalf("MBL作废身份、版本指针或事件错误: mbl=%+v event=%+v", mbl, event)
	}
	old := f.data.db.SeaMasterBillVersion.GetX(ctx, f.mblVerID)
	if string(old.Status) == "VOIDED" {
		t.Fatal("旧 MBL 不可变版本被覆盖为 VOIDED")
	}
	_, err = NewSeaCargoAllocationRepo(f.data).ApplyMasterBillSummary(ctx, f.orgID, f.actorID, f.orderID, mbl.Version, f.audit())
	if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentVoided.Reason {
		t.Fatalf("VOIDED MBL 仍可通过箱货汇总改写，错误=%v", err)
	}
	idempotent, err := repo.ExecuteVoid(ctx, f.orgID, f.actorID, cmd, f.audit())
	if err != nil || idempotent.ID != event.ID {
		t.Fatalf("MBL 作废幂等重试失败: event=%+v err=%v", idempotent, err)
	}
}

func TestSeaDocumentAmendmentConcurrentPublish(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	repo := NewSeaDocumentChangeRepo(f.data)
	makeCommand := func(value string) *biz.SeaDocumentAmendmentCommand {
		return &biz.SeaDocumentAmendmentCommand{
			OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
			ExpectedOrderVersion: 1, ExpectedDocumentVersion: 1, ExpectedCurrentVersionID: f.mblVerID,
			Reason: "并发改单", IdempotencyKey: "concurrent-" + uuid.NewString(),
			Input: &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPointer(value)}},
		}
	}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, value := range []string{"并发版本甲", "并发版本乙"} {
		value := value
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.ExecuteAmendment(ctx, f.orgID, f.actorID, makeCommand(value), f.audit())
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
	repo := NewSeaDocumentChangeRepo(f.data)
	cmd := &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
		ExpectedOrderVersion: 1, ExpectedDocumentVersion: 1, ExpectedCurrentVersionID: f.mblVerID,
		Reason: "并发幂等改单", IdempotencyKey: "concurrent-idempotent-" + uuid.NewString(),
		Input: &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPointer("并发幂等版本")}},
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
			version, err := repo.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
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

func TestSeaHouseBillSwitchConcurrentSingleSuccess(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	repo := NewSeaDocumentChangeRepo(f.data)
	old := f.data.db.SeaHouseBill.GetX(ctx, f.switchID)
	makeCommand := func(houseNo string) *biz.SeaHouseBillSwitchCommand {
		return &biz.SeaHouseBillSwitchCommand{
			OrderID: f.orderID, OldHouseBillID: old.ID, ExpectedOrderVersion: 1,
			ExpectedHouseBillVersion: old.Version, ExpectedCurrentVersionID: *old.CurrentVersionID,
			Reason: "并发换单", IdempotencyKey: "concurrent-switch-" + uuid.NewString(),
			NewHouseBill: &biz.SeaHouseBillInput{HouseNo: houseNo, IssuerSource: biz.SeaHouseBillIssuerSourceCustomerPartner, Content: &biz.SeaBillContent{}},
		}
	}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, no := range []string{"CONCURRENT-SWITCH-A", "CONCURRENT-SWITCH-B"} {
		wg.Add(1)
		go func(houseNo string) {
			defer wg.Done()
			_, err := repo.ExecuteSwitch(ctx, f.orgID, f.actorID, makeCommand(houseNo), f.audit())
			errs <- err
		}(no)
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
			t.Fatalf("并发 Switch 返回非预期错误: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("并发 Switch 应只有一个成功: success=%d conflict=%d", successes, conflicts)
	}
	if count := f.data.db.SeaHouseBillSwitchEvent.Query().CountX(ctx); count != 1 {
		t.Fatalf("并发 Switch 产生重复事件: count=%d", count)
	}
}

func TestSeaMasterBillMemberSetRevalidatedAfterLock(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	expectedIDs, activeLinkID, err := locateMasterMemberOrderIDs(ctx, f.data.db, f.orgID, f.orderID, f.mblID)
	if err != nil {
		t.Fatalf("定位初始 MBL 成员失败: %v", err)
	}
	member := f.data.db.Order.Create().SetOrganizationID(f.orgID).SetOrderNo("SE-MEMBER-" + uuid.NewString()[:8]).SetCustomerID(f.partnerID).SetBusinessType("SE").SetTradeDirection("export").SetTradeTerm("FOB").SetPaymentTerm("PREPAID").SaveX(ctx)
	f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.orgID).SetOrderID(member.ID).SetMasterBillID(f.mblID).SetStatus(seamasterbillorderlinkent.StatusACTIVE).SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).SaveX(ctx)
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

func TestSeaDocumentHistoricalInvoiceAndVerificationBlockChange(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	repo := NewSeaDocumentChangeRepo(f.data)
	suffix := uuid.NewString()[:8]
	fee := f.data.db.OrderFee.Create().SetOrderID(f.orderID).SetIdempotencyKey("historical-fee-" + suffix).SetDirection(orderfeeent.DirectionRECEIVABLE).SetStatus(orderfeeent.StatusDRAFT).SetFeeCode("HIS-FEE-" + suffix).SetFeeName("历史财务事实费用").SetSettlementPartyID(f.partnerID).SetBillingUnit("BILL").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(orderfeeent.ExchangeRateSourceBASE_CURRENCY).SetExchangeRateDate("2026-09-04").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetExpenseDate("2026-09-04").SaveX(ctx)
	bill := f.data.db.FinanceBill.Create().SetOrganizationID(f.orgID).SetBillNo("HIS-BILL-" + suffix).SetIdempotencyKey("historical-bill-" + suffix).SetDirection(financebillent.DirectionRECEIVABLE).SetStatus(financebillent.StatusDRAFT).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("单证变更测试合作伙伴").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).SetExchangeRateDate("2026-09-04").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").SetFeeCount(1).SetBillDate("2026-09-04").SaveX(ctx)
	f.data.db.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderID(f.orderID).SetOrderFeeID(fee.ID).SetOrderNo("HISTORICAL").SetFeeCode(fee.FeeCode).SetFeeName(fee.FeeName).SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrencyAmount("100").SetBaseCurrency("CNY").SetActive(false).SaveX(ctx)
	invoice := f.data.db.FinanceInvoice.Create().SetOrganizationID(f.orgID).SetRecordNo("HIS-INV-" + suffix).SetIdempotencyKey("historical-invoice-" + suffix).SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetStatus(financeinvoiceent.StatusDRAFT).SetInvoiceType(financeinvoiceent.InvoiceTypeNORMAL).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("单证变更测试合作伙伴").SetCurrency("CNY").SetBaseCurrency("CNY").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBillCount(1).SaveX(ctx)
	f.data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).SetAmount("100").SetTaxAmount("0").SetActive(false).SaveX(ctx)
	cashflow := f.data.db.FinanceCashflow.Create().SetOrganizationID(f.orgID).SetFlowNo("HIS-FLOW-" + suffix).SetIdempotencyKey("historical-flow-" + suffix).SetDirection(financecashflowent.DirectionRECEIVABLE).SetStatus(financecashflowent.StatusDRAFT).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("单证变更测试合作伙伴").SetCurrency("CNY").SetAmount("100").SetExchangeRate("1").SetExchangeRateSource(financecashflowent.ExchangeRateSourceBASE_CURRENCY).SetExchangeRateDate("2026-09-04").SetBaseCurrency("CNY").SetBaseAmount("100").SetTransactionDate("2026-09-04").SetOurAccount("历史账户").SetPaymentMethod("BANK_TRANSFER").SaveX(ctx)
	verification := f.data.db.FinanceVerification.Create().SetOrganizationID(f.orgID).SetVerificationNo("HIS-VER-" + suffix).SetIdempotencyKey("historical-verification-" + suffix).SetStatus(financeverificationent.StatusACTIVE).SetDirection(financeverificationent.DirectionRECEIVABLE).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("单证变更测试合作伙伴").SetCurrency("CNY").SetAmount("100").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(financeverificationent.ExchangeRateSourceBASE_CURRENCY).SetExchangeRateDate("2026-09-04").SetBaseAmount("100").SetBillBaseAmount("100").SetCashflowBaseAmount("100").SetExchangeGainLoss("0").SetVerificationDate("2026-09-04").SaveX(ctx)
	f.data.db.FinanceVerificationAllocation.Create().SetVerificationID(verification.ID).SetCashflowID(cashflow.ID).SetBillID(bill.ID).SetCashflowNo(cashflow.FlowNo).SetBillNo(bill.BillNo).SetAmount("100").SetBillBaseAmount("100").SetCashflowBaseAmount("100").SetWriteOffBaseAmount("100").SetExchangeGainLoss("0").SetActive(false).SaveX(ctx)

	cmd := &biz.SeaDocumentVoidCommand{OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID, ExpectedOrderVersion: 1, ExpectedDocumentVersion: 1, ExpectedCurrentVersionID: f.mblVerID, Reason: "历史事实门禁", IdempotencyKey: "historical-block-" + suffix}
	preview, err := repo.PreviewVoid(ctx, f.orgID, cmd)
	if err != nil {
		t.Fatalf("历史财务事实预览失败: %v", err)
	}
	facts := map[string]string{}
	for _, impact := range preview.Impacts {
		facts[impact.FactType] = impact.ReferenceNo
	}
	if preview.Executable || facts["FINANCE_INVOICE"] != invoice.RecordNo || facts["FINANCE_VERIFICATION"] != verification.VerificationNo {
		t.Fatalf("历史发票/核销未被直接门禁: executable=%v facts=%+v", preview.Executable, facts)
	}
}
