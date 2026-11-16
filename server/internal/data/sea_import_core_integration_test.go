package data

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	handlingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderimporthandling"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderimporthandlingevent"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func seaImportInput(f *orderPostgresFixture, direct bool) *biz.Order {
	input := f.validInput()
	input.BusinessType = biz.OrderBusinessSI
	input.TradeDirection = biz.OrderTradeImport
	if direct {
		structure := biz.SeaDocumentStructureDirect
		input.SeaDocumentInput = &biz.SeaOrderDocumentInput{DocumentStructure: &structure}
	}
	return input
}
func TestSeaImportCorePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := newOrderPostgresFixture(t, d)
	uc := f.newUsecase()
	var first *biz.Order
	for _, direct := range []bool{true, false} {
		input := seaImportInput(f, direct)
		input.SeaMasterBillInput.MasterNo = map[bool]string{true: "SIDIRECT", false: "SIHOUSE"}[direct]
		created, e := uc.Create(ctx, f.organizationID, f.actorID, input)
		if e != nil {
			t.Fatal(e)
		}
		if created.BusinessType != biz.OrderBusinessSI || created.TradeDirection != biz.OrderTradeImport || created.FlowStatus != biz.OrderFlowImportNotApplicable || created.SeaMasterBill == nil || created.SeaMasterBill.BusinessType != biz.OrderBusinessSI {
			t.Fatalf("进口真实模型不完整: %+v", created)
		}
		docs, e := NewSeaDocumentRepo(d).GetSeaOrderDocuments(ctx, f.organizationID, created.ID)
		if e != nil {
			t.Fatal(e)
		}
		if docs.MasterBill.BusinessType != biz.OrderBusinessSI || (docs.HouseBill == nil) != direct {
			t.Fatalf("直分单结构不正确: %+v", docs)
		}
		handling, e := uc.GetImportHandling(ctx, f.organizationID, created.ID)
		if e != nil || len(handling.Steps) != 5 || handling.OrderVersion != 1 {
			t.Fatalf("初始服务: %+v %v", handling, e)
		}
		update := seaImportInput(f, direct)
		update.SeaDocumentInput = nil
		update.SeaMasterBillInput.MasterNo = input.SeaMasterBillInput.MasterNo
		update.SeaMasterBillInput.ExpectedCandidateVersion = &docs.MasterBill.Version
		update.Notes = "进口普通保存"
		saved, e := uc.UpdateDraft(ctx, f.organizationID, f.actorID, created.ID, created.Version, update)
		if e != nil {
			t.Fatal(e)
		}
		if saved.Version != 2 || saved.FlowStatus != biz.OrderFlowImportNotApplicable || saved.ETA != input.ETA {
			t.Fatalf("进口普通保存未保留摘要和ETA: %+v", saved)
		}
		if _, e = uc.TransitionStatus(ctx, f.organizationID, f.actorID, saved.ID, saved.Version, biz.OrderFlowBooked, ""); e == nil {
			t.Fatalf("进口不可走出口进度: %v", e)
		}
		if direct {
			first = saved
		}
	}
	// 同公司同号的进出口主单各自独立，候选只查明确方向。
	export := f.validInput()
	export.SeaMasterBillInput.MasterNo = "SIDIRECT"
	se, e := uc.Create(ctx, f.organizationID, f.actorID, export)
	if e != nil {
		t.Fatal(e)
	}
	if se.SeaMasterBill.MasterBillID == first.SeaMasterBill.MasterBillID {
		t.Fatal("进出口主单被混合")
	}
	for _, kind := range []biz.OrderBusinessType{biz.OrderBusinessSI, biz.OrderBusinessSE} {
		match, e := uc.MatchSeaMasterBillCandidate(ctx, f.organizationID, f.shippingLineID, "SIDIRECT", nil, kind)
		if e != nil || !match.Matched || match.Candidate.BusinessType != kind {
			t.Fatalf("方向候选: %+v %v", match, e)
		}
	}
	// 直单主单不得加第二张操作票，失败不留下新票。
	docs, e := NewSeaDocumentRepo(d).GetSeaOrderDocuments(ctx, f.organizationID, first.ID)
	if e != nil {
		t.Fatal(e)
	}
	attach := seaImportInput(f, false)
	attach.SeaMasterBillInput = &biz.SeaMasterBillInput{MasterNo: "SIDIRECT", CandidateID: &docs.MasterBill.ID, ExpectedCandidateVersion: &docs.MasterBill.Version, CandidateTEID: &first.SeaMasterBill.TransportExecutionID, ExpectedCandidateTEVersion: &first.SeaMasterBill.TransportExecutionVersion}
	before := d.db.Order.Query().CountX(ctx)
	if _, e = uc.Create(ctx, f.organizationID, f.actorID, attach); !errors.Is(e, biz.ErrSeaMasterBillBatchDirectBlocked) {
		t.Fatalf("直单加票应拒绝: %v", e)
	}
	if d.db.Order.Query().CountX(ctx) != before {
		t.Fatal("拒绝加票留下订单")
	}
	// 直接ID传出口主单或另一公司目标均拒绝，服务也不能跨公司读写。
	attach.SeaMasterBillInput.CandidateID = &se.SeaMasterBill.MasterBillID
	if _, e = uc.Create(ctx, f.organizationID, f.actorID, attach); e == nil {
		t.Fatal("进口误挂出口主单")
	}
	foreignFixture := newOrderPostgresFixture(t, d)
	foreignInput := seaImportInput(foreignFixture, false)
	foreignOrder, err := foreignFixture.newUsecase().Create(ctx, foreignFixture.organizationID, foreignFixture.actorID, foreignInput)
	if err != nil {
		t.Fatal(err)
	}
	attach.ShippingLineID = &foreignFixture.shippingLineID
	attach.SeaMasterBillInput = &biz.SeaMasterBillInput{MasterNo: foreignOrder.SeaMasterBill.MasterNo, CandidateID: &foreignOrder.SeaMasterBill.MasterBillID, ExpectedCandidateVersion: &foreignOrder.SeaMasterBill.Version, CandidateTEID: &foreignOrder.SeaMasterBill.TransportExecutionID, ExpectedCandidateTEVersion: &foreignOrder.SeaMasterBill.TransportExecutionVersion}
	if _, err = uc.Create(ctx, f.organizationID, f.actorID, attach); err == nil {
		t.Fatal("进口不应关联另一公司真实主单")
	}
	if _, e = uc.GetImportHandling(ctx, uuid.New(), first.ID); e == nil {
		t.Fatal("跨公司读取服务")
	}
	if _, e = uc.GetImportHandling(ctx, f.organizationID, se.ID); !errors.Is(e, biz.ErrOrderBusinessUnsupported) {
		t.Fatalf("出口不应有进口服务: %v", e)
	}
	// 服务保存后普通整票编辑保持既有承接事实。
	handling, e := uc.GetImportHandling(ctx, f.organizationID, first.ID)
	if e != nil {
		t.Fatal(e)
	}
	handling.Steps[4].Undertaken = true
	handling, e = uc.SaveImportHandling(ctx, f.organizationID, f.actorID, first.ID, first.Version, handling, "")
	if e != nil {
		t.Fatal(e)
	}
	update := seaImportInput(f, true)
	update.SeaDocumentInput = nil
	update.SeaMasterBillInput.MasterNo = "SIDIRECT"
	update.SeaMasterBillInput.ExpectedCandidateVersion = &docs.MasterBill.Version
	saved, e := uc.UpdateDraft(ctx, f.organizationID, f.actorID, first.ID, handling.OrderVersion, update)
	if e != nil {
		t.Fatal(e)
	}
	kept, e := uc.GetImportHandling(ctx, f.organizationID, first.ID)
	if e != nil || !kept.Steps[4].Undertaken || saved.FlowStatus != biz.OrderFlowImportNotStarted {
		t.Fatalf("普通保存清空服务: %+v %v", kept, e)
	}
}

func cloneHandling(t *testing.T, input *biz.OrderImportHandling) *biz.OrderImportHandling {
	t.Helper()
	raw, e := json.Marshal(input)
	if e != nil {
		t.Fatal(e)
	}
	var out biz.OrderImportHandling
	if e = json.Unmarshal(raw, &out); e != nil {
		t.Fatal(e)
	}
	return &out
}
func TestSeaImportHandlingLifecyclePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	newCorrectionTestUsecase(t, f)
	o := f.createOrder("SI-HANDLING-" + uuid.NewString())
	uc := biz.NewOrderUsecase(NewOrderRepo(f.data), NewSeaMasterBillRepo(f.data), NewSeaDocumentRepo(f.data), nil, f.data)
	h, e := uc.GetImportHandling(f.ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	save := func(next *biz.OrderImportHandling, reason string) *biz.OrderImportHandling {
		t.Helper()
		out, e := uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, reason)
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	h.Steps[0].Undertaken = true
	h.Steps[2].Undertaken = true
	h.Steps[4].Undertaken = true
	h = save(h, "")
	if biz.ImportHandlingFlow(h) != biz.OrderFlowImportNotStarted {
		t.Fatal("未开始摘要错误")
	}
	// 并行先完成派送，无需依次完成到货/清关；完成需真实时间。
	now := time.Date(2026, 11, 16, 1, 2, 3, 0, time.UTC)
	h.Steps[4].OccurredAt = &now
	h.Steps[4].ResponsibleUserID = &f.approverID
	h.Steps[4].ReferenceNo = "DEL-001"
	h = save(h, "")
	if biz.ImportHandlingFlow(h) != biz.OrderFlowImportInProgress || h.Steps[4].ResponsibleUserName == "" {
		t.Fatalf("并行完成/人员投影: %+v", h)
	}
	old := h.OrderVersion
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, old-1, cloneHandling(t, h), ""); !errors.Is(e, biz.ErrOrderStatusConflict) {
		t.Fatalf("过期版本应409: %v", e)
	}
	if _, e = uc.TransitionClosure(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, biz.OrderClosureClosed, ""); !errors.Is(e, biz.ErrOrderClosureBlocked) {
		t.Fatalf("未完成承接应阻断完结: %v", e)
	}
	for _, kind := range []string{"撤回", "备注", "取消"} {
		next := cloneHandling(t, h)
		switch kind {
		case "撤回":
			next.Steps[4].OccurredAt = nil
		case "备注":
			next.Steps[4].Notes = "纠正"
		case "取消":
			next.Steps[4].Undertaken = false
			next.Steps[4].OccurredAt = nil
		}
		if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, ""); e == nil {
			t.Fatalf("%s应要求原因", kind)
		}
	}
	next := cloneHandling(t, h)
	next.Steps[4].Undertaken = false
	next.Steps[4].OccurredAt = nil
	h = save(next, "取消派送委托")
	h.Steps[0].OccurredAt = &now
	h.Steps[2].OccurredAt = &now
	h = save(h, "")
	closed, e := uc.TransitionClosure(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, biz.OrderClosureClosed, "")
	if e != nil || closed.FlowStatus != biz.OrderFlowImportCompleted {
		t.Fatalf("适用服务全部完成应可结案: %+v %v", closed, e)
	}
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, closed.Version, cloneHandling(t, h), ""); !errors.Is(e, biz.ErrOrderClosureBlocked) {
		t.Fatalf("结案服务修改必须显式重开: %v", e)
	}
	reopened, e := uc.TransitionClosure(f.ctx, f.organizationID, f.approverID, o.ID, closed.Version, biz.OrderClosureOpen, "")
	if e != nil {
		t.Fatal(e)
	}
	h.OrderVersion = reopened.Version
	next = cloneHandling(t, h)
	next.Steps[0].OccurredAt = nil
	h = save(next, "到货记录纠正")
	events, total, e := uc.ListImportHandlingEvents(f.ctx, f.organizationID, o.ID, 1, 200)
	if e != nil || total != 5 || len(events) != 5 || events[0].ActorName == "" {
		t.Fatalf("事件与真实操作者: %d %+v %v", total, events, e)
	}
	found := false
	for _, event := range events {
		if event.Reason == "取消派送委托" {
			found = true
			if event.Before.Steps[4].OccurredAt == nil || event.After.Steps[4].OccurredAt != nil || event.After.Steps[4].Undertaken {
				t.Fatal("取消必须保留旧完成事实")
			}
		}
	}
	if !found {
		t.Fatal("缺取消事件")
	}
	for _, size := range []int{0, 201} {
		if _, _, e = uc.ListImportHandlingEvents(f.ctx, f.organizationID, o.ID, 1, size); e == nil {
			t.Fatal("历史分页边界未拒绝")
		}
	}
	// 审核锁独立阻断服务写；解锁后终止恢复保持本票服务及原主单，不复活历史Link。
	lockRepo := NewOrderLockRepo(f.data, nil)
	review := lockTestReviewInput(t, f.data, f.ctx, o.ID)
	if _, e = lockRepo.LockOrder(f.ctx, f.approverPrincipal(), o.ID, h.OrderVersion, uuid.NewString(), review, nil); e != nil {
		t.Fatal(e)
	}
	locked := f.data.db.Order.GetX(f.ctx, o.ID)
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, locked.Version, cloneHandling(t, h), ""); e == nil {
		t.Fatal("审核锁未阻断服务")
	}
	record := f.data.db.OrderLockRecord.Query().Where().OnlyX(f.ctx)
	if record.MasterBillID == nil || record.TransportExecutionID == nil {
		t.Fatal("SI审核未固化真实海运快照")
	}
	if _, e = lockRepo.RequestOrderUnlock(f.ctx, f.approverPrincipal(), o.ID, locked.Version, uuid.NewString(), nil, nil); e != nil {
		t.Fatal(e)
	}
	current, e := uc.Get(f.ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	originalLink := f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(o.ID), linkent.StatusEQ(linkent.StatusACTIVE)).OnlyX(f.ctx)
	typ := biz.OrderTerminationOperationCancel
	current, e = uc.TransitionTermination(f.ctx, f.organizationID, f.approverID, o.ID, current.Version, biz.OrderTerminationTerminating, &typ, "测试终止")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, current.Version, cloneHandling(t, h), ""); e == nil {
		t.Fatal("终止中不允许服务写入")
	}
	current, e = uc.TransitionTermination(f.ctx, f.organizationID, f.approverID, o.ID, current.Version, biz.OrderTerminationTerminated, &typ, "完成终止")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = NewSeaDocumentRepo(f.data).GetSeaOrderDocuments(f.ctx, f.organizationID, o.ID); e != nil {
		t.Fatal(e)
	}
	current, e = uc.TransitionTermination(f.ctx, f.organizationID, f.approverID, o.ID, current.Version, biz.OrderTerminationActive, nil, "恢复原主单航程")
	if e != nil {
		t.Fatal(e)
	}
	restoredLink := f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(o.ID), linkent.StatusEQ(linkent.StatusACTIVE)).OnlyX(f.ctx)
	if restoredLink.ID == originalLink.ID || restoredLink.MasterBillID != originalLink.MasterBillID || restoredLink.TransportExecutionID != originalLink.TransportExecutionID {
		t.Fatal("恢复未创建原主单新Link")
	}
	preserved, e := uc.GetImportHandling(f.ctx, f.organizationID, o.ID)
	if e != nil || preserved.Steps[2].OccurredAt == nil || preserved.Steps[0].OccurredAt != nil {
		t.Fatalf("恢复清空服务: %+v %v", preserved, e)
	}
}

func TestSeaImportHandlingReferencesAndRollbackPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("SI-REF-" + uuid.NewString())
	uc := biz.NewOrderUsecase(NewOrderRepo(f.data), nil, nil, nil, f.data)
	h, e := uc.GetImportHandling(f.ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	supplier := f.data.db.Partner.Create().SetOrganizationID(f.organizationID).SetCode("EXCHANGE").SetLegalName("独立换单代理").SetNormalizedName("独立换单代理").SaveX(f.ctx)
	role := f.data.db.PartnerRole.Create().SetPartnerID(supplier.ID).SetRoleType(partnerroleent.RoleTypeSupplier).SetEnabled(true).SaveX(f.ctx)
	h.ExchangeAgentID = &supplier.ID
	h.Steps[1].Undertaken = true
	now := time.Now().UTC().Truncate(time.Second)
	h.Steps[1].OccurredAt = &now
	h, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, h, "")
	if e != nil || h.ExchangeAgentName != supplier.LegalName {
		t.Fatalf("换单代理: %+v %v", h, e)
	}
	next := cloneHandling(t, h)
	next.ExchangeAgentID = nil
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, ""); e == nil {
		t.Fatal("更正完成换单代理必须有原因")
	}
	before := f.data.db.Order.GetX(f.ctx, o.ID)
	count := f.data.db.OrderImportHandlingEvent.Query().CountX(f.ctx)
	assertUnchanged := func() {
		t.Helper()
		actual := f.data.db.Order.GetX(f.ctx, o.ID)
		if actual.Version != before.Version || f.data.db.OrderImportHandlingEvent.Query().CountX(f.ctx) != count {
			t.Fatal("失败服务写入未整体回滚")
		}
	}
	next = cloneHandling(t, h)
	next.Steps[3].Undertaken = true
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, uuid.New(), o.ID, h.OrderVersion, next, ""); e == nil {
		t.Fatal("不存在操作者应拒绝并回滚")
	}
	assertUnchanged()
	// 审计写入真实失败时，服务状态、派生进度、订单版本和业务事件均回滚。
	if _, e = f.data.sqlDB.ExecContext(f.ctx, `CREATE FUNCTION reject_import_service_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action = 'order.import_handling.save' THEN RAISE EXCEPTION '测试审计失败'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_import_service_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION reject_import_service_audit()`); e != nil {
		t.Fatal(e)
	}
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, cloneHandling(t, h), ""); e == nil {
		t.Fatal("审计失败必须原样返回")
	}
	assertUnchanged()
	if _, e = f.data.sqlDB.ExecContext(f.ctx, `DROP TRIGGER reject_import_service_audit ON audit_logs; DROP FUNCTION reject_import_service_audit()`); e != nil {
		t.Fatal(e)
	}
	next = cloneHandling(t, h)
	next.Steps[3].AttachmentIDs = []uuid.UUID{uuid.New()}
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, ""); e == nil {
		t.Fatal("不存在附件未拒绝")
	}
	assertUnchanged()
	next = cloneHandling(t, h)
	next.Steps[3].ResponsibleUserID = &supplier.ID
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, ""); e == nil {
		t.Fatal("非法责任人未拒绝")
	}
	assertUnchanged()
	f.data.db.PartnerRole.UpdateOne(role).SetBlacklisted(true).SaveX(f.ctx)
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, cloneHandling(t, h), ""); e == nil {
		t.Fatal("黑名单供应商未拒绝")
	}
	assertUnchanged()
	f.data.db.PartnerRole.UpdateOne(role).SetBlacklisted(false).SaveX(f.ctx)
	if _, e = uc.SaveImportHandling(f.ctx, uuid.New(), f.approverID, o.ID, h.OrderVersion, cloneHandling(t, h), ""); e == nil {
		t.Fatal("跨公司服务写未拒绝")
	}
	assertUnchanged()
	// 本票附件可关联，他票附件和跨公司资产引用均拒绝。
	otherOrder := f.createOrder("SI-ATTACH-" + uuid.NewString())
	asset := f.data.db.OrderAttachmentAsset.Create().SetOrganizationID(f.organizationID).SetObjectKey("test/" + uuid.NewString()).SetFileName("进口附件.pdf").SetMimeType("application/pdf").SetFileSize(16).SaveX(f.ctx)
	foreignAttachment := f.data.db.OrderAttachment.Create().SetOrderID(otherOrder.ID).SetAssetID(asset.ID).SetDocType("IMPORT").SetIdempotencyKey(uuid.NewString()).SaveX(f.ctx)
	next = cloneHandling(t, h)
	next.Steps[3].AttachmentIDs = []uuid.UUID{foreignAttachment.ID}
	if _, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, ""); e == nil {
		t.Fatal("外票附件未拒绝")
	}
	assertUnchanged()
	attachment := f.data.db.OrderAttachment.Create().SetOrderID(o.ID).SetAssetID(asset.ID).SetDocType("IMPORT").SetIdempotencyKey(uuid.NewString()).SaveX(f.ctx)
	next = cloneHandling(t, h)
	next.Steps[3].Undertaken = true
	next.Steps[3].AttachmentIDs = []uuid.UUID{attachment.ID}
	h, e = uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, "")
	if e != nil || len(h.Steps[3].AttachmentIDs) != 1 {
		t.Fatalf("合法本票附件: %+v %v", h, e)
	}
	// 两个相同基线的服务写入仅一个成功，另一个返回版本冲突。
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, i := range []int{0, 4} {
		next := cloneHandling(t, h)
		next.Steps[i].Undertaken = true
		go func() {
			<-start
			_, err := uc.SaveImportHandling(f.ctx, f.organizationID, f.approverID, o.ID, h.OrderVersion, next, "")
			results <- err
		}()
	}
	close(start)
	success, conflict := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, biz.ErrOrderStatusConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("并发服务保存应一成功一409: %d/%d", success, conflict)
	}
	if current := f.data.db.Order.GetX(f.ctx, o.ID); current.Version != h.OrderVersion+1 {
		t.Fatal("并发服务版本未精确推进一次")
	}
	// 独立事实并非共享主单上的其他票服务；不会由运输到达自动写入。
	other := f.createOrder("SI-OTHER-" + uuid.NewString())
	otherHandling, e := uc.GetImportHandling(f.ctx, f.organizationID, other.ID)
	if e != nil || biz.ImportHandlingFlow(otherHandling) != biz.OrderFlowImportNotApplicable {
		t.Fatal("另一票被服务修改污染")
	}
	if f.data.db.OrderImportHandlingEvent.Query().Where(eventent.OrderIDEQ(other.ID)).CountX(f.ctx) != 0 {
		t.Fatal("不应自动生成他票服务事件")
	}
	if _, _, e = uc.ListImportHandlingEvents(f.ctx, f.organizationID, o.ID, 0, 20); e == nil {
		t.Fatal("page=0未拒绝")
	}
	state := f.data.db.OrderImportHandling.Query().Where(handlingent.OrderIDEQ(o.ID)).OnlyX(f.ctx)
	var stateFields map[string]json.RawMessage
	if e = json.Unmarshal(state.State, &stateFields); e != nil {
		t.Fatal(e)
	}
	if _, ok := stateFields["orderVersion"]; ok {
		t.Fatal("当前服务不可另存订单镜像版本")
	}
	if !strings.Contains(string(state.State), "EXCHANGE") {
		t.Fatal("服务未持久保存")
	}
}
