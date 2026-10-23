package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airhouse "github.com/roncin/roncin-go-admin/server/internal/data/ent/airhousewaybill"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

type airExportFixture struct {
	*orderPostgresFixture
	airlineID, originID, destinationID uuid.UUID
}

func newAirExportFixture(t *testing.T, d *Data) *airExportFixture {
	f := &airExportFixture{orderPostgresFixture: newOrderPostgresFixture(t, d)}
	ctx := context.Background()
	s := newTestSCAC()[:3]
	a := d.db.Airline.Create().SetIataCode(s[:2]).SetNameEn("AE Test " + s).SetCountryCode("CN").SaveX(ctx)
	o := d.db.Airport.Create().SetIataCode("A" + s[:2]).SetNameEn("AE Origin").SetCountryCode("CN").SaveX(ctx)
	dest := d.db.Airport.Create().SetIataCode("B" + s[:2]).SetNameEn("AE Destination").SetCountryCode("US").SaveX(ctx)
	t.Cleanup(func() {
		d.db.OrderLockRecord.Delete().ExecX(ctx)
		d.db.AirHouseWaybillVersion.Delete().ExecX(ctx)
		d.db.AirMasterWaybillHistory.Delete().ExecX(ctx)
		d.db.AirMasterWaybillOrderLink.Delete().ExecX(ctx)
		d.db.AirHouseWaybill.Delete().ExecX(ctx)
		d.db.AirMasterWaybill.Delete().ExecX(ctx)
	})
	f.airlineID = a.ID
	f.originID = o.ID
	f.destinationID = dest.ID
	return f
}
func (f *airExportFixture) input(no, house string) *biz.Order {
	in := f.validInput()
	in.BusinessType = biz.OrderBusinessAE
	in.ShippingLineID = nil
	in.SeaMasterBillInput = nil
	in.SeaDocumentInput = nil
	in.OriginLocationID = nil
	in.DestinationLocationID = nil
	g, v := 100.0, 1.2
	in.TotalGrossWeightKg = &g
	in.TotalVolumeCbm = &v
	in.TotalPackageUnit = "CTN"
	p := 1
	in.TotalPackages = &p
	in.GoodsDescription = "空运测试货物"
	in.OperationNotes = "报关资料已安排"
	in.AirExportInput = &biz.AirOrderInput{Master: &biz.AirMasterWaybillInput{MasterNo: no, AirlineID: f.airlineID, OriginAirportID: f.originID, DestinationAirportID: f.destinationID, FlightNo: "MU587", ETD: "2026-10-25T08:00:00Z", ETA: "2026-10-25T20:00:00Z"}, HouseNo: house}
	return in
}
func (f *airExportFixture) create(t *testing.T, in *biz.Order) *biz.Order {
	t.Helper()
	o, e := f.newUsecase().Create(context.Background(), f.organizationID, f.actorID, in)
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func TestAirExportSharedWaybillPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	a := f.create(t, f.input("781-12345675", "H-001"))
	if a.AirExport == nil || a.AirExport.HouseNo != "H-001" {
		t.Fatalf("建票读回缺航空关系: %#v", a)
	}
	input := f.input("78112345675", "H002")
	id := a.AirExport.Master.ID
	input.AirExportInput.MasterWaybillID = &id
	input.AirExportInput.ExpectedMasterVersion = a.AirExport.Master.Version
	b := f.create(t, input)
	detail, e := air.Get(ctx, f.organizationID, id, biz.SelectorListOptions{Page: 1, PageSize: 1})
	if e != nil {
		t.Fatal(e)
	}
	if detail.Total != 2 || len(detail.Members) != 1 || detail.Totals.GrossWeightKg != 200 || detail.Totals.VolumeCbm != 2.4 {
		t.Fatalf("服务端全量汇总错误: %#v", detail)
	}
	searched, e := uc.List(ctx, []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessAE, OrganizationIDs: []uuid.UUID{f.organizationID}}}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessAE, Keyword: "H-001"})
	if e != nil || searched.Total != 1 {
		t.Fatalf("合法连字符分单搜索失败 %#v %v", searched, e)
	}
	next := detail.Master.Content
	next.FlightNo = "MU588"
	preview, e := air.Preview(ctx, f.organizationID, id, detail.Master.Version, &next)
	if e != nil || !preview.CanExecute {
		t.Fatalf("共享预览: %#v %v", preview, e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, id, detail.Master.Version, &next, preview.InputFingerprint, "航班调整"); e != nil {
		t.Fatal(e)
	}
	for _, orderID := range []uuid.UUID{a.ID, b.ID} {
		o, e := uc.Get(ctx, f.organizationID, orderID)
		if e != nil || o.AirExport.Master.Content.FlightNo != "MU588" {
			t.Fatalf("两票读取权威航班: %v %v", o, e)
		}
	}
	if _, e = air.Get(ctx, uuid.New(), id, biz.SelectorListOptions{Page: 1, PageSize: 20}); !errors.Is(e, biz.ErrAirWaybillNotFound) {
		t.Fatalf("跨组织读取: %v", e)
	}
	now := time.Now()
	d.db.Order.UpdateOneID(b.ID).SetLockedAt(now).SetLockedBy(f.actorID).SetLockSource(orderent.LockSourceMANUAL).SetVersion(b.Version + 1).SaveX(ctx)
	detail, e = air.Get(ctx, f.organizationID, id, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil {
		t.Fatal(e)
	}
	next = detail.Master.Content
	next.FlightNo = "MU589"
	preview, e = air.Preview(ctx, f.organizationID, id, detail.Master.Version, &next)
	if e != nil || preview.CanExecute {
		t.Fatalf("锁成员应阻止共享: %#v %v", preview, e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, id, detail.Master.Version, &next, preview.InputFingerprint, "不能绕过锁"); !errors.Is(e, biz.ErrAirWaybillBlocked) {
		t.Fatalf("共享绕锁: %v", e)
	}
	current, e := uc.Get(ctx, f.organizationID, a.ID)
	if e != nil {
		t.Fatal(e)
	}
	private := *current
	private.Notes = "仅本票备注"
	private.AirExportInput = nil
	if _, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, a.ID, current.Version, &private); e != nil {
		t.Fatalf("B 锁定不得阻止 A 私有编辑: %v", e)
	}
}
func TestAirExportRequiredNumbersUniquenessAndRollbackPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	for _, kind := range []string{"master", "house", "both"} {
		in := f.input("78112345675", "H001")
		if kind != "house" {
			in.AirExportInput.Master.MasterNo = ""
		}
		if kind != "master" {
			in.AirExportInput.HouseNo = ""
		}
		if _, e := uc.Create(ctx, f.organizationID, f.actorID, in); e == nil {
			t.Fatalf("缺 %s 仍建票", kind)
		}
	}
	if n := d.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).CountX(ctx); n != 0 {
		t.Fatalf("缺号留部分订单 %d", n)
	}
	first := f.create(t, f.input("78112345675", "H001"))
	master := first.AirExport.Master
	in := f.input("78112345675", " h001 ")
	in.AirExportInput.MasterWaybillID = &master.ID
	in.AirExportInput.ExpectedMasterVersion = master.Version
	if _, e := uc.Create(ctx, f.organizationID, f.actorID, in); !errors.Is(e, biz.ErrAirWaybillExists) {
		t.Fatalf("同主单重复号: %v", e)
	}
	f.create(t, f.input("78112345686", "H001"))
	if n := d.db.AirHouseWaybill.Query().Where(airhouse.OrganizationIDEQ(f.organizationID)).CountX(ctx); n != 2 {
		t.Fatalf("跨主单重号应放行 %d", n)
	}
}
func TestAirExportTerminationRestoreAndConcurrentIdentityPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	a := f.create(t, f.input("78112345675", "H001"))
	kind := biz.OrderTerminationOther
	a, e := uc.TransitionTermination(ctx, f.organizationID, f.actorID, a.ID, a.Version, biz.OrderTerminationTerminating, &kind, "客户取消")
	if e != nil {
		t.Fatal(e)
	}
	a, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, a.ID, a.Version, biz.OrderTerminationTerminated, &kind, "终止完成")
	if e != nil {
		t.Fatal(e)
	}
	if n := d.db.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(a.ID), airlink.StatusEQ("ACTIVE")).CountX(ctx); n != 0 {
		t.Fatal("终止未结束关联")
	}
	if _, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, a.ID, a.Version, biz.OrderTerminationActive, nil, "恢复"); e == nil {
		t.Fatal("无关系恢复被放行")
	}
	m := a.AirExport.Master
	candidates, total, e := air.LinkCandidates(ctx, f.organizationID, m.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil || total != 1 || len(candidates) != 1 {
		t.Fatalf("恢复候选: %d %v", total, e)
	}
	if e = air.Link(ctx, f.organizationID, f.actorID, m.ID, a.ID, m.Version, a.Version, "H001", "恢复原主单"); e != nil {
		t.Fatal(e)
	}
	if n := d.db.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(a.ID)).CountX(ctx); n != 2 {
		t.Fatal("恢复应创建新关联而非复活旧记录")
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := uc.Create(ctx, f.organizationID, f.actorID, f.input("78112345686", fmt.Sprintf("RACE%d", i)))
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	success, conflict := 0, 0
	for e := range errs {
		if e == nil {
			success++
		} else if errors.Is(e, biz.ErrAirWaybillExists) {
			conflict++
		} else {
			t.Fatalf("并发未知错误: %v", e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("并发主单身份成功=%d 冲突=%d", success, conflict)
	}
}

func TestAirExportFlowVersionsAndSnapshotsPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	o := f.create(t, f.input("78112345675", "H001"))
	m := o.AirExport.Master
	// 合法主单号候选显式关联需命中版本，不允许仅号码拼单。
	bad := f.input("78112345675", "H002")
	bad.AirExportInput.MasterWaybillID = &m.ID
	bad.AirExportInput.ExpectedMasterVersion = m.Version + 1
	if _, e := uc.Create(ctx, f.organizationID, f.actorID, bad); !errors.Is(e, biz.ErrAirWaybillConflict) {
		t.Fatalf("过期主单版本未409: %v", e)
	}
	update := *o
	update.AirExportInput = &biz.AirOrderInput{HouseNo: o.AirExport.HouseNo, HouseStatus: o.AirExport.HouseStatus, ExpectedLinkVersion: o.AirExport.LinkVersion + 1}
	if _, e := uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &update); !errors.Is(e, biz.ErrAirWaybillConflict) {
		t.Fatalf("过期关联版本未409: %v", e)
	}
	if _, e := uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderFlowDocumentReleased, ""); e == nil {
		t.Fatal("非法跳转未阻断")
	}
	for _, target := range []biz.OrderFlowStatus{biz.OrderFlowBooked, biz.OrderFlowSpaceAllocated, biz.OrderFlowDocumentCutoff, biz.OrderFlowCustomsDeclarationArranged} {
		var e error
		o, e = uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, target, "资料已确认")
		if e != nil {
			t.Fatalf("推进 %s: %v", target, e)
		}
	}
	if _, e := uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderFlowDocumentReleased, ""); e == nil {
		t.Fatal("主分单未签发仍签单")
	}
	detail, e := air.Get(ctx, f.organizationID, m.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil {
		t.Fatal(e)
	}
	next := detail.Master.Content
	next.Status = "RELEASED"
	next.ActualDepartureAt = "2026-10-25T08:01:00Z"
	next.ActualArrivalAt = "2026-10-25T20:02:00Z"
	p, e := air.Preview(ctx, f.organizationID, m.ID, detail.Master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, m.ID, detail.Master.Version, &next, p.InputFingerprint, "签发主单并登记实际运输节点"); e != nil {
		t.Fatal(e)
	}
	o, e = uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	if o.FlowStatus != biz.OrderFlowCustomsDeclarationArranged || o.ClosureStatus != biz.OrderClosureOpen {
		t.Fatal("实际起降错误推进签单或结案")
	}
	private := *o
	private.AirExportInput = &biz.AirOrderInput{HouseNo: o.AirExport.HouseNo, HouseStatus: "RELEASED", ExpectedLinkVersion: o.AirExport.LinkVersion}
	o, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private)
	if e != nil {
		t.Fatal(e)
	}
	o, e = uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderFlowDocumentReleased, "签单完成")
	if e != nil {
		t.Fatal(e)
	}
	if o.ClosureStatus != biz.OrderClosureOpen {
		t.Fatal("签单不应财务结案")
	}
	// 本票签发后的改单必须留原因和不可变版本。
	private = *o
	private.AirExportInput = &biz.AirOrderInput{HouseNo: o.AirExport.HouseNo, HouseStatus: "RELEASED", HouseContent: "更正后货描", ExpectedLinkVersion: o.AirExport.LinkVersion}
	if _, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private); e == nil {
		t.Fatal("已签发分单无改单原因仍保存")
	}
	private.AirExportInput.ChangeReason = "客户更正货描"
	o, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private)
	if e != nil {
		t.Fatal(e)
	}
	versions, total, e := air.HouseVersions(ctx, f.organizationID, o.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil || total != 3 || versions[0].Reason != "客户更正货描" {
		t.Fatalf("不可变本票版本 %d %#v %v", total, versions, e)
	}
	// 人工审核锁使用航空独立快照，不污染 SE 外键。
	e = d.WithTx(ctx, func(tx *ent.Tx) error {
		locked, e := tx.Order.Query().Where(orderent.IDEQ(o.ID)).ForUpdate().Only(ctx)
		if e != nil {
			return e
		}
		spec, e := airTestReviewLockSpec(ctx, tx, locked, f.actorID, "air-lock")
		if e != nil {
			return e
		}
		rec, sea, e := applyOrderLockCore(ctx, tx, f.organizationID, locked, spec)
		if e != nil {
			return e
		}
		if sea != nil || rec.MasterBillID != nil || rec.AirDocumentSnapshot["master"] == "" || rec.AirDocumentSnapshot["house"] == "" {
			t.Fatalf("航空锁快照错误 %#v", rec)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}

func TestAirExportAuditRollbackAndOverduePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	input := f.input("78112345675", "H001")
	input.AirExportInput.Master.DocumentCutoffAt = "2020-01-01T00:00:00Z"
	o := f.create(t, input)
	for _, target := range []biz.OrderFlowStatus{biz.OrderFlowBooked, biz.OrderFlowSpaceAllocated} {
		var e error
		o, e = uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, target, "")
		if e != nil {
			t.Fatal(e)
		}
	}
	if _, e := uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderFlowDocumentCutoff, ""); e == nil {
		t.Fatal("逾期无处理原因未阻断")
	}
	var e error
	o, e = uc.TransitionStatus(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderFlowDocumentCutoff, "已与客户确认逾期处理")
	if e != nil {
		t.Fatalf("单纯逾期有原因不应阻断 %v", e)
	}
	master := o.AirExport.Master
	next := master.Content
	next.FlightNo = "MU999"
	p, e := air.Preview(ctx, f.organizationID, master.ID, master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	// 在隔离 Schema 内注入真实数据库审计失败，检验主单、历史与版本同事务。
	if _, e = d.sqlDB.ExecContext(ctx, `CREATE FUNCTION fail_air_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION '模拟审计写入失败'; END $$`); e != nil {
		t.Fatal(e)
	}
	if _, e = d.sqlDB.ExecContext(ctx, `CREATE TRIGGER fail_air_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION fail_air_audit()`); e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, master.ID, master.Version, &next, p.InputFingerprint, "模拟失败"); e == nil {
		t.Fatal("审计失败未回滚共享写")
	}
	if _, e = d.sqlDB.ExecContext(ctx, `DROP TRIGGER fail_air_audit ON audit_logs`); e != nil {
		t.Fatal(e)
	}
	after, e := air.Get(ctx, f.organizationID, master.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil {
		t.Fatal(e)
	}
	if after.Master.Version != master.Version || after.Master.Content.FlightNo != master.Content.FlightNo || after.HistoryTotal != 1 {
		t.Fatalf("共享写部分提交 %#v", after)
	}
	// 创建审计失败亦不得消耗编号、保留航空实体或半张订单。
	if _, e = d.sqlDB.ExecContext(ctx, `CREATE TRIGGER fail_air_audit BEFORE INSERT ON audit_logs FOR EACH ROW EXECUTE FUNCTION fail_air_audit()`); e != nil {
		t.Fatal(e)
	}
	if _, e = uc.Create(ctx, f.organizationID, f.actorID, f.input("78112345686", "H002")); e == nil {
		t.Fatal("审计失败未回滚建票")
	}
	if _, e = d.sqlDB.ExecContext(ctx, `DROP TRIGGER fail_air_audit ON audit_logs`); e != nil {
		t.Fatal(e)
	}
	if n := d.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).CountX(ctx); n != 1 {
		t.Fatalf("失败建票留订单 %d", n)
	}
	// 共享运输时刻的搜索读取权威主单，避免订单创建时副本导致漏单。
	next = after.Master.Content
	next.ETD = "2026-11-01T08:00:00Z"
	next.ETA = "2026-11-01T20:00:00Z"
	p, e = air.Preview(ctx, f.organizationID, master.ID, after.Master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	if e = air.Update(ctx, f.organizationID, f.actorID, master.ID, after.Master.Version, &next, p.InputFingerprint, "改期"); e != nil {
		t.Fatal(e)
	}
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	rows, e := uc.List(ctx, []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessAE, OrganizationIDs: []uuid.UUID{f.organizationID}}}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessAE, ETDRange: biz.OrderDateRange{From: &from, ToExclusive: &to}, NumberType: biz.OrderNumberFilterMaster, NumberKeyword: "781-12345675"})
	if e != nil || rows.Total != 1 {
		t.Fatalf("权威航班日期/主单搜索失败 %#v %v", rows, e)
	}
}

func TestAirExportMemberChangeAndLockRacePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	a := f.create(t, f.input("78112345675", "H001"))
	m := a.AirExport.Master
	next := m.Content
	next.FlightNo = "MU777"
	preview, e := air.Preview(ctx, f.organizationID, m.ID, m.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	in := f.input("78112345675", "H002")
	in.AirExportInput.MasterWaybillID = &m.ID
	in.AirExportInput.ExpectedMasterVersion = m.Version
	f.create(t, in)
	if e = air.Update(ctx, f.organizationID, f.actorID, m.ID, m.Version, &next, preview.InputFingerprint, "过期成员预览"); !errors.Is(e, biz.ErrAirWaybillConflict) {
		t.Fatalf("成员集合变化未拒绝旧预览 %v", e)
	}
	detail, e := air.Get(ctx, f.organizationID, m.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil {
		t.Fatal(e)
	}
	preview, e = air.Preview(ctx, f.organizationID, m.ID, detail.Master.Version, &next)
	if e != nil {
		t.Fatal(e)
	}
	start := make(chan struct{})
	out := make(chan error, 2)
	go func() {
		<-start
		out <- air.Update(ctx, f.organizationID, f.actorID, m.ID, detail.Master.Version, &next, preview.InputFingerprint, "与锁单并发")
	}()
	go func() {
		<-start
		out <- d.WithTx(ctx, func(tx *ent.Tx) error {
			o, e := tx.Order.Query().Where(orderent.IDEQ(a.ID)).ForUpdate().Only(ctx)
			if e != nil {
				return e
			}
			spec, e := airTestReviewLockSpec(ctx, tx, o, f.actorID, "air-race-lock")
			if e != nil {
				return e
			}
			_, _, e = applyOrderLockCore(ctx, tx, f.organizationID, o, spec)
			return e
		})
	}()
	close(start)
	for range 2 {
		e := <-out
		if e != nil && !errors.Is(e, biz.ErrAirWaybillBlocked) {
			t.Fatalf("并发共享/锁单未知失败 %v", e)
		}
	}
	// 任一执行顺序都必须在锁快照与最终主单中保持同一运输事实，不能锁后再改共享事实。
	row := d.db.Order.GetX(ctx, a.ID)
	if row.LockedAt == nil {
		t.Fatal("并发锁单未成功")
	}
	rec := d.db.OrderLockRecord.Query().OnlyX(ctx)
	master := d.db.AirMasterWaybill.GetX(ctx, m.ID)
	var snapshot biz.AirMasterWaybill
	if e = json.Unmarshal([]byte(rec.AirDocumentSnapshot["master"]), &snapshot); e != nil {
		t.Fatal(e)
	}
	if snapshot.Content.FlightNo != master.FlightNo {
		t.Fatalf("锁后共享事实被改写 snapshot=%s current=%s", snapshot.Content.FlightNo, master.FlightNo)
	}
}

// 航空单证并发测试直接验证锁核心，仍提供与人工审核入口相同的费用证据。
func airTestReviewLockSpec(ctx context.Context, tx *ent.Tx, o *ent.Order, reviewer uuid.UUID, key string) (orderLockCoreSpec, error) {
	snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, tx.Client(), o.ID)
	if err != nil {
		return orderLockCoreSpec{}, err
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return orderLockCoreSpec{}, err
	}
	return orderLockCoreSpec{reviewerID: reviewer, idempotencyKey: key, requestFingerprint: key, feeSnapshot: payload, feeContentVersion: o.FeeContentVersion, feeContentFingerprint: fingerprint, reviewComment: "航空单证集成验证：已核对订单与费用"}, nil
}
