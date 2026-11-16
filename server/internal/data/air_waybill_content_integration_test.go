package data

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	airhouse "github.com/roncin/roncin-go-admin/server/internal/data/ent/airhousewaybill"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestAirWaybillOrdinarySaveChecksExplicitMasterVersionPostgres(t *testing.T) {
	for _, businessType := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		t.Run(string(businessType), func(t *testing.T) { verifyAirWaybillOrdinarySaveChecksExplicitMasterVersionPostgres(t, businessType) })
	}
}

func verifyAirWaybillOrdinarySaveChecksExplicitMasterVersionPostgres(t *testing.T, businessType biz.OrderBusinessType) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	in := airInputForBusiness(f, businessType, "78112345675", "H-MASTER-BASELINE")
	in.AirExportInput.House.Content = &biz.AirWaybillContent{Remarks: "原始分单"}
	o := f.create(t, in)
	master := o.AirExport.Master
	next := master.Content
	next.WaybillContent = &biz.AirWaybillContent{Remarks: "其他操作员的新主单正文"}
	preview, err := air.Preview(ctx, f.organizationID, master.ID, master.Version, &next)
	if err != nil {
		t.Fatal(err)
	}
	if err = air.Update(ctx, f.organizationID, f.actorID, master.ID, master.Version, &next, preview.InputFingerprint, "更新主单印字"); err != nil {
		t.Fatal(err)
	}
	// 主单正文不推进订单版本；旧主单基线必须在主单锁内单独复验。
	stale := *o
	stale.Notes = "不能提交的本票修改"
	stale.AirExportInput = &biz.AirOrderInput{
		DocumentMode: "HOUSE", ExpectedMasterVersion: master.Version, ExpectedLinkVersion: o.AirExport.LinkVersion,
		House: &biz.AirHouseWaybillInput{HouseNo: o.AirExport.House.HouseNo, Status: "DRAFT", ExpectedVersion: o.AirExport.House.Version, Content: &biz.AirWaybillContent{Remarks: "不能提交的分单修改"}},
	}
	if _, err = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &stale); !errors.Is(err, biz.ErrAirWaybillConflict) {
		t.Fatalf("旧主单版本普通保存未冲突：%v", err)
	}
	stored, err := uc.Get(ctx, f.organizationID, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version != o.Version || stored.Notes != o.Notes || stored.AirExport.House.Version != o.AirExport.House.Version || stored.AirExport.House.ContentHash != o.AirExport.House.ContentHash || stored.AirExport.LinkVersion != o.AirExport.LinkVersion || d.db.AirHouseWaybillVersion.Query().CountX(ctx) != 1 {
		t.Fatal("旧主单版本冲突留下本票、分单或历史部分写入")
	}
	if stored.AirExport.Master.Version != master.Version+1 || stored.AirExport.Master.ContentHash != biz.AirWaybillContentHash(next.WaybillContent) || d.db.AirMasterWaybillHistory.Query().CountX(ctx) != 2 {
		t.Fatal("失败保存改写主单或主单历史")
	}
	// 省略主单预期版本的纯私有更新不写主单，仍允许独立保存。
	private := *o
	private.Notes = "只保存本票资料"
	private.AirExportInput = &biz.AirOrderInput{DocumentMode: "HOUSE", ExpectedLinkVersion: o.AirExport.LinkVersion}
	saved, err := uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != o.Version+1 || saved.Notes != private.Notes || saved.AirExport.Master.ContentHash != stored.AirExport.Master.ContentHash || saved.AirExport.House.ContentHash != o.AirExport.House.ContentHash || saved.AirExport.House.Version != o.AirExport.House.Version || saved.AirExport.LinkVersion != o.AirExport.LinkVersion {
		t.Fatal("私有保存覆盖主分正文或推进子实体版本")
	}
	private = *saved
	private.AirExportInput = &biz.AirOrderInput{DocumentMode: "HOUSE", ExpectedMasterVersion: saved.AirExport.Master.Version, ExpectedLinkVersion: saved.AirExport.LinkVersion}
	if _, err = uc.UpdateDraft(ctx, f.organizationID, f.actorID, saved.ID, saved.Version, &private); err != nil {
		t.Fatalf("当前主单版本被误拒：%v", err)
	}
}

func TestAirWaybillIndependentContentVersionsAndReviewPostgres(t *testing.T) {
	for _, businessType := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		t.Run(string(businessType), func(t *testing.T) { verifyAirWaybillIndependentContentVersionsAndReviewPostgres(t, businessType) })
	}
}

func verifyAirWaybillIndependentContentVersionsAndReviewPostgres(t *testing.T, businessType biz.OrderBusinessType) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	p := airPrincipalForBusiness(f, businessType)
	ctx := biz.WithPrincipal(context.Background(), p)
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	in := airInputForBusiness(f, businessType, "78112345675", "H-CONTENT")
	in.AirExportInput.Master.WaybillContent = &biz.AirWaybillContent{ShipperText: "主单发货人", Remarks: "主单独立正文"}
	in.AirExportInput.House.Content = &biz.AirWaybillContent{ShipperText: strings.Repeat("中", 5000), Marks: strings.Repeat("唛", 10000), Remarks: "  原始分单\n换行  ", IssueDate: "2026-11-16", CustomsValue: &biz.AirDeclaredValue{Kind: "AMOUNT", Amount: "0"}}
	o := f.create(t, in)
	master := o.AirExport.Master
	originalHouse := o.AirExport.House
	if originalHouse.Content.Remarks != in.AirExportInput.House.Content.Remarks || master.Content.WaybillContent.Remarks != "主单独立正文" || originalHouse.ContentHash == master.ContentHash {
		t.Fatal("独立正文、换行或哈希错误")
	}
	// 按ID选择已有主单，日期与机场只取锁内权威事实；幂等重放不依赖副本。
	existingInput := airInputForBusiness(f, businessType, "", "H-ID-ONLY")
	existingInput.ETD = ""
	existingInput.ETA = ""
	existingInput.DestinationLocationID = nil
	existingInput.IdempotencyKey = uuid.NewString()
	existingInput.AirExportInput.MasterWaybillID = &master.ID
	existingInput.AirExportInput.ExpectedMasterVersion = master.Version
	existingInput.AirExportInput.Master = nil
	joined, e := uc.Create(ctx, f.organizationID, f.actorID, existingInput)
	if e != nil {
		t.Fatal(e)
	}
	if joined.AirExport.Master.ID != master.ID || joined.ETD != master.Content.ETD || joined.DestinationLocationID == nil || *joined.DestinationLocationID != master.Content.DestinationAirportID {
		t.Fatal("ID-only建票未读取权威运输")
	}
	replay, e := uc.Create(ctx, f.organizationID, f.actorID, existingInput)
	if e != nil || replay.ID != joined.ID {
		t.Fatalf("ID-only幂等重放错误：%v", e)
	}
	o, e = uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	master = o.AirExport.Master
	// 公共订单保存不带单证输入时保持正文与子实体版本。
	private := *o
	private.Notes = "普通经营字段"
	o, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private)
	if e != nil {
		t.Fatal(e)
	}
	if o.AirExport.House.Version != originalHouse.Version || o.AirExport.LinkVersion != 1 || o.AirExport.House.ContentHash != originalHouse.ContentHash {
		t.Fatal("普通保存覆盖正文或推进子版本")
	}
	// 主单重量只绑定成员身份及货量，改单Link版本不应制造重量过期。
	weights := biz.NewAirWeightUsecase(NewAirWeightRepo(d))
	wi := &biz.AirWeightInput{Scope: "MASTER", TargetID: master.ID, Source: "PLANNED"}
	wp, e := weights.Preview(ctx, f.organizationID, wi)
	if e != nil {
		t.Fatal(e)
	}
	wi.InputFingerprint = wp.InputFingerprint
	wi.ConfirmedWeightKG = wp.SuggestedChargeableWeightKG
	if _, e = weights.Confirm(ctx, f.organizationID, f.actorID, wi); e != nil {
		t.Fatal(e)
	}
	// 主单正文在自己的共享预览入口保存，分单与Order版本保持独立。
	lockRepo := NewOrderLockRepo(d, nil)
	state, e := lockRepo.GetOrderLockState(ctx, f.organizationID, o.ID, p)
	if e != nil {
		t.Fatal(e)
	}
	staleReview := biz.OrderLockReviewInput{ExpectedFeeContentVersion: state.FeeContentVersion, FeeContentFingerprint: state.FeeContentFingerprint, AirDocumentFingerprint: state.AirDocumentFingerprint}
	next := master.Content
	next.WaybillContent = &biz.AirWaybillContent{ShipperText: "新主单发货人", Remarks: "主单更新", Currency: "USD", WeightPayment: "PPD", RateLines: []*biz.AirRateLine{{Amount: "0.1"}, {Amount: "0.2"}}}
	preview, e := air.Preview(ctx, f.organizationID, master.ID, master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, master.ID, master.Version, &next, preview.InputFingerprint, "修改主单印字"); e != nil {
		t.Fatal(e)
	}
	o, e = uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if o.Version != state.OrderVersion || o.AirExport.Master.Version != master.Version+1 || o.AirExport.House.ContentHash != originalHouse.ContentHash || o.AirExport.Master.Content.WaybillContent.Prepaid.WeightCharge != "0.3" {
		t.Fatal("主单正文影响分单/经营版本或合计错误")
	}
	if _, e = lockRepo.LockOrder(ctx, p, o.ID, o.Version, uuid.NewString(), staleReview, nil); !errors.Is(e, biz.ErrAirWaybillConflict) {
		t.Fatalf("旧航空审核指纹未冲突：%v", e)
	}
	if d.db.Order.GetX(ctx, o.ID).LockedAt != nil || d.db.Order.GetX(ctx, o.ID).Version != o.Version || d.db.OrderLockRecord.Query().CountX(ctx) != 0 {
		t.Fatal("过期审核部分写入")
	}
	detail, e := air.Get(ctx, f.organizationID, master.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil {
		t.Fatal(e)
	}
	if len(detail.History) != 3 || detail.History[0].ContentHash != o.AirExport.Master.ContentHash || detail.History[2].Content.WaybillContent.Remarks != "主单独立正文" {
		t.Fatalf("主单不可变历史错误：%+v", detail.History)
	}
	// house.content={} 是清空，不是省略。目标House和Link都推进版本。
	before := o
	private = *o
	private.AirExportInput = &biz.AirOrderInput{DocumentMode: "HOUSE", ExpectedLinkVersion: o.AirExport.LinkVersion, House: &biz.AirHouseWaybillInput{HouseNo: o.AirExport.House.HouseNo, Status: o.AirExport.House.Status, ExpectedVersion: o.AirExport.House.Version, Content: &biz.AirWaybillContent{}}}
	o, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private)
	if e != nil {
		t.Fatal(e)
	}
	if o.AirExport.House.Version != before.AirExport.House.Version+1 || o.AirExport.LinkVersion != before.AirExport.LinkVersion+1 || o.AirExport.House.Content.Remarks != "" || o.AirExport.House.ContentHash != biz.AirWaybillContentHash(nil) {
		t.Fatal("分单清空或版本推进错误")
	}
	history, total, e := air.HouseVersions(ctx, f.organizationID, o.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil || total != 2 || history[1].Content.Remarks != originalHouse.Content.Remarks || history[1].Content.CustomsValue.Amount != "0" || history[1].ContentHash != originalHouse.ContentHash {
		t.Fatalf("分单历史丢失旧正文：%+v %v", history, e)
	}
	unchangedWeight, e := weights.Preview(ctx, f.organizationID, &biz.AirWeightInput{Scope: "MASTER", TargetID: master.ID, Source: "PLANNED"})
	if e != nil || unchangedWeight.ConfirmationStale {
		t.Fatalf("印字更新误使MASTER重量过期：%+v %v", unchangedWeight, e)
	}
	// House与Link的预期版本均独立复验；普通保存不能切模式、改号或换主单。
	for _, scenario := range []string{"house版本", "link版本", "号码", "主单"} {
		invalid := *o
		h := &biz.AirHouseWaybillInput{HouseNo: o.AirExport.House.HouseNo, Status: o.AirExport.House.Status, ExpectedVersion: o.AirExport.House.Version, Content: &biz.AirWaybillContent{Remarks: "不应提交"}}
		ai := &biz.AirOrderInput{DocumentMode: "HOUSE", ExpectedLinkVersion: o.AirExport.LinkVersion, House: h}
		invalid.AirExportInput = ai
		switch scenario {
		case "house版本":
			h.ExpectedVersion--
		case "link版本":
			ai.ExpectedLinkVersion--
		case "号码":
			h.HouseNo = "H-NEW"
		case "主单":
			id := uuid.New()
			ai.MasterWaybillID = &id
		}
		if _, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &invalid); e == nil {
			t.Fatalf("非法%s被放行", scenario)
		}
	}
	if stored := d.db.AirHouseWaybill.Query().Where(airhouse.OrderIDEQ(o.ID)).OnlyX(ctx); stored.Version != o.AirExport.House.Version || stored.ContentHash != o.AirExport.House.ContentHash {
		t.Fatal("失败更新部分改写分单")
	}
	// 共同运输保存省略正文保持，明确{}清空只影响Master自己的历史。
	next = o.AirExport.Master.Content
	next.WaybillContent = nil
	next.Notes = "运输备注与印字分开"
	preview, e = air.Preview(ctx, f.organizationID, master.ID, o.AirExport.Master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, master.ID, o.AirExport.Master.Version, &next, preview.InputFingerprint, "只修改运输备注"); e != nil {
		t.Fatal(e)
	}
	o, e = uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if o.AirExport.Master.Content.WaybillContent.Remarks != "主单更新" {
		t.Fatal("共同资料省略正文却被清空")
	}
	next = o.AirExport.Master.Content
	next.WaybillContent = &biz.AirWaybillContent{}
	preview, e = air.Preview(ctx, f.organizationID, master.ID, o.AirExport.Master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, master.ID, o.AirExport.Master.Version, &next, preview.InputFingerprint, "显式清空主单正文"); e != nil {
		t.Fatal(e)
	}
	after := d.db.AirMasterWaybill.GetX(ctx, master.ID)
	if after.ContentHash != biz.AirWaybillContentHash(nil) || after.Notes != "运输备注与印字分开" {
		t.Fatal("清空主单正文串改运输备注")
	}
}

func TestAirWaybillBodyAuditFailureAndPermanentNumberPostgres(t *testing.T) {
	for _, businessType := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		t.Run(string(businessType), func(t *testing.T) { verifyAirWaybillBodyAuditFailureAndPermanentNumberPostgres(t, businessType) })
	}
}

func verifyAirWaybillBodyAuditFailureAndPermanentNumberPostgres(t *testing.T, businessType biz.OrderBusinessType) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	in := airInputForBusiness(f, businessType, "78112345675", "H-PERMANENT")
	in.AirExportInput.House.Content = &biz.AirWaybillContent{Remarks: "旧分单正文"}
	o := f.create(t, in)
	if _, e := d.sqlDB.ExecContext(ctx, `CREATE FUNCTION fail_air_content_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION '正文审计失败测试'; END $$`); e != nil {
		t.Fatal(e)
	}
	if _, e := d.sqlDB.ExecContext(ctx, `CREATE TRIGGER fail_air_content_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION fail_air_content_audit()`); e != nil {
		t.Fatal(e)
	}
	private := *o
	private.AirExportInput = &biz.AirOrderInput{DocumentMode: "HOUSE", ExpectedLinkVersion: o.AirExport.LinkVersion, House: &biz.AirHouseWaybillInput{HouseNo: o.AirExport.House.HouseNo, Status: "RELEASED", ExpectedVersion: o.AirExport.House.Version, Content: &biz.AirWaybillContent{Remarks: "不应保存"}}}
	if _, e := uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private); e == nil {
		t.Fatal("审计失败仍提交分单")
	}
	next := o.AirExport.Master.Content
	next.WaybillContent = &biz.AirWaybillContent{Remarks: "不应保存主单"}
	preview, e := air.Preview(ctx, f.organizationID, o.AirExport.Master.ID, o.AirExport.Master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, o.AirExport.Master.ID, o.AirExport.Master.Version, &next, preview.InputFingerprint, "验证回滚"); e == nil {
		t.Fatal("审计失败仍提交主单")
	}
	if _, e = d.sqlDB.ExecContext(ctx, `DROP TRIGGER fail_air_content_audit ON audit_logs`); e != nil {
		t.Fatal(e)
	}
	after, e := uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if after.Version != o.Version || after.AirExport.House.Version != o.AirExport.House.Version || after.AirExport.LinkVersion != o.AirExport.LinkVersion || after.AirExport.Master.Version != o.AirExport.Master.Version || after.AirExport.House.Content.Remarks != "旧分单正文" || d.db.AirHouseWaybillVersion.Query().CountX(ctx) != 1 || d.db.AirMasterWaybillHistory.Query().CountX(ctx) != 1 {
		t.Fatal("审计失败留下正文、版本或历史部分写入")
	}
	kind := biz.OrderTerminationOther
	o, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderTerminationTerminating, &kind, "取消")
	if e != nil {
		t.Fatal(e)
	}
	o, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderTerminationTerminated, &kind, "完成")
	if e != nil {
		t.Fatal(e)
	}
	duplicate := airInputForBusiness(f, businessType, "78112345675", "H-PERMANENT")
	duplicate.AirExportInput.Master = &o.AirExport.Master.Content
	duplicate.AirExportInput.MasterWaybillID = &o.AirExport.Master.ID
	duplicate.AirExportInput.ExpectedMasterVersion = o.AirExport.Master.Version
	if _, e = uc.Create(ctx, f.organizationID, f.actorID, duplicate); !errors.Is(e, biz.ErrAirWaybillExists) {
		t.Fatalf("历史分单号未永久占用：%v", e)
	}
	if d.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != 1 {
		t.Fatal("永久号冲突遗留半票")
	}
	f.create(t, airInputForBusiness(f, businessType, "78112345686", "H-PERMANENT"))
}
