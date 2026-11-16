package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	airhouse "github.com/roncin/roncin-go-admin/server/internal/data/ent/airhousewaybill"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	unitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/shopspring/decimal"
)

func airTestPrincipal(f *airExportFixture) *biz.Principal {
	return &biz.Principal{UserID: f.actorID, IsBootstrapAdmin: true, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: f.organizationID, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.OrderPermission(access.OrderBusinessAE, access.OrderRead): {}, access.OrderPermission(access.OrderBusinessAE, access.OrderUpdate): {}}}}}
}
func directAirInput(f *airExportFixture, no string) *biz.Order {
	in := f.input(no, "")
	in.AirExportInput.DocumentMode = "DIRECT"
	in.AirExportInput.House = nil
	return in
}

func TestAirDirectCreateEditRestoreWeightFeesAndReviewPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	p := airTestPrincipal(f)
	ctx := biz.WithPrincipal(context.Background(), p)
	uc := f.newUsecase()
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	in := directAirInput(f, "78112345675")
	in.IdempotencyKey = uuid.NewString()
	in.AirExportInput.Master.WaybillContent = &biz.AirWaybillContent{ShipperText: "直单发货人\n完整地址", Remarks: "主单正文"}
	o := f.create(t, in)
	if o.AirExport.DocumentMode != "DIRECT" || o.AirExport.House != nil || d.db.AirHouseWaybill.Query().Where(airhouse.OrderIDEQ(o.ID)).CountX(ctx) != 0 {
		t.Fatal("直单不应存在HAWB")
	}
	// 本组不开放AI：同公司同号进口主单也不能借AE重量权限读取。
	foreign := d.db.AirMasterWaybill.Create().SetBusinessType("AI").SetOrganizationID(f.organizationID).SetMasterNo("78112345675").SetAirlineID(f.airlineID).SetOriginAirportID(f.originID).SetDestinationAirportID(f.destinationID).SaveX(ctx)
	if _, e := biz.NewAirWeightUsecase(NewAirWeightRepo(d)).Preview(ctx, f.organizationID, &biz.AirWeightInput{Scope: "MASTER", TargetID: foreign.ID, Source: "PLANNED"}); !errors.Is(e, biz.ErrAirWeightNotFound) {
		t.Fatalf("AE重量入口越权读AI主单：%v", e)
	}
	replay, e := uc.Create(ctx, f.organizationID, f.actorID, in)
	if e != nil || replay.ID != o.ID {
		t.Fatalf("航空幂等创建失败：%v", e)
	}
	for _, mode := range []string{"DIRECT", "HOUSE"} {
		next := f.input("78112345675", "H-ILLEGAL")
		next.AirExportInput.DocumentMode = mode
		if mode == "DIRECT" {
			next.AirExportInput.House = nil
		}
		next.AirExportInput.MasterWaybillID = &o.AirExport.Master.ID
		next.AirExportInput.ExpectedMasterVersion = o.AirExport.Master.Version
		next.AirExportInput.Master = &o.AirExport.Master.Content
		if _, e = uc.Create(ctx, f.organizationID, f.actorID, next); e == nil {
			t.Fatalf("直单第二票或混挂%s被放行", mode)
		}
	}
	if d.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != 1 {
		t.Fatal("拒绝新挂票遗留半票")
	}
	candidates, _, e := air.List(ctx, f.organizationID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil || len(candidates) != 1 || candidates[0].CanAttach || candidates[0].DocumentMode != "DIRECT" || candidates[0].AttachBlockedReason == "" {
		t.Fatalf("候选未表达实际模式和阻断原因：%+v %v", candidates, e)
	}
	private := *o
	private.Notes = "修改本票经营备注"
	o, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private)
	if e != nil {
		t.Fatal(e)
	}
	if o.AirExport.House != nil || o.AirExport.Master.Content.WaybillContent.Remarks != "主单正文" {
		t.Fatal("普通保存改变单证存在性或清空正文")
	}
	forbidden := *o
	forbidden.AirExportInput = &biz.AirOrderInput{DocumentMode: "HOUSE", ExpectedLinkVersion: o.AirExport.LinkVersion, House: &biz.AirHouseWaybillInput{HouseNo: "H-MODE"}}
	if _, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &forbidden); e == nil {
		t.Fatal("普通保存改变已保存模式")
	}
	// 终止保留最近主单与DIRECT身份，恢复原票不需要虚构HAWB。
	kind := biz.OrderTerminationOther
	o, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderTerminationTerminating, &kind, "取消测试")
	if e != nil {
		t.Fatal(e)
	}
	o, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderTerminationTerminated, &kind, "终止完成")
	if e != nil {
		t.Fatal(e)
	}
	oldLink := d.db.AirMasterWaybillOrderLink.GetX(ctx, o.AirExport.LinkID)
	restored, _, e := air.LinkCandidates(ctx, f.organizationID, o.AirExport.Master.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil || len(restored) != 1 || restored[0].HouseNo != "" {
		t.Fatalf("DIRECT恢复候选缺失：%+v %v", restored, e)
	}
	if e = air.Link(ctx, f.organizationID, f.actorID, o.AirExport.Master.ID, o.ID, o.AirExport.Master.Version, o.Version, "", "恢复原直单"); e != nil {
		t.Fatal(e)
	}
	o, e = uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	historical := d.db.AirMasterWaybillOrderLink.GetX(ctx, oldLink.ID)
	if o.AirExport.LinkID == oldLink.ID || historical.Status != "ENDED" || historical.Version != oldLink.Version || o.AirExport.House != nil {
		t.Fatal("恢复改写历史或创建虚假HAWB")
	}
	// 即使一票直单，MASTER与ORDER也必须各自显式确认。
	weight := biz.NewAirWeightUsecase(NewAirWeightRepo(d))
	mi := &biz.AirWeightInput{Scope: "MASTER", TargetID: o.AirExport.Master.ID, Source: "PLANNED"}
	mp, e := weight.Preview(ctx, f.organizationID, mi)
	if e != nil {
		t.Fatal(e)
	}
	mi.InputFingerprint = mp.InputFingerprint
	mi.ConfirmedWeightKG = mp.SuggestedChargeableWeightKG
	if _, e = weight.Confirm(ctx, f.organizationID, f.actorID, mi); e != nil {
		t.Fatal(e)
	}
	oi := &biz.AirWeightInput{Scope: "ORDER", TargetID: o.ID, Source: "PLANNED"}
	op, e := weight.Preview(ctx, f.organizationID, oi)
	if e != nil || op.CurrentConfirmation != nil {
		t.Fatalf("MASTER确认被当作ORDER：%+v %v", op, e)
	}
	oi.InputFingerprint = op.InputFingerprint
	oi.ConfirmedWeightKG = op.SuggestedChargeableWeightKG
	confirmed, e := weight.Confirm(ctx, f.organizationID, f.actorID, oi)
	if e != nil {
		t.Fatal(e)
	}
	unit := d.db.BillingUnit.Query().Where(unitent.CodeEQ("KG")).OnlyX(ctx)
	fees := NewOrderFeeRepo(d).(*orderFeeRepo)
	source, e := fees.ConfirmedQuantityPreview(ctx, f.organizationID, o.ID, unit.ID, "PLANNED")
	if e != nil || !source.Available || source.QuantitySource.ConfirmationID != confirmed.ID {
		t.Fatalf("直单ORDER费用来源错误：%+v %v", source, e)
	}
	tax := d.db.TaxableService.Create().SetOrganizationID(f.organizationID).SetName("直单测试应税服务").SetDefaultTaxRate("0").SaveX(ctx)
	setting := d.db.FeeSetting.Create().SetOrganizationID(f.organizationID).SetFeeCode("DIRECT_AIR").SetNameZh("直单空运费").SetDefaultCurrency("CNY").SetBillingUnitID(unit.ID).SetTaxRate("0").SetTaxableServiceID(tax.ID).SaveX(ctx)
	t.Cleanup(func() {
		d.db.OrderFee.Delete().ExecX(context.Background())
		d.db.FeeSetting.DeleteOne(setting).ExecX(context.Background())
		d.db.TaxableService.DeleteOne(tax).ExecX(context.Background())
	})
	q := decimal.RequireFromString(source.Quantity)
	day, e := orderRateDayOf(d.db.Order.GetX(ctx, o.ID))
	if e != nil {
		t.Fatal(e)
	}
	fee := &biz.OrderFee{ID: uuid.New(), OrderID: o.ID, IdempotencyKey: uuid.NewString(), Direction: biz.OrderFeeReceivable, FeeSettingID: &setting.ID, FeeCode: "DIRECT_AIR", FeeName: "直单空运费", SettlementPartyID: f.partnerID, BillingUnitID: &unit.ID, BillingUnit: unit.Name, Quantity: q, QuantitySource: source.QuantitySource, UnitPrice: decimal.NewFromInt(2), TotalAmount: q.Mul(decimal.NewFromInt(2)), NetAmount: q.Mul(decimal.NewFromInt(2)), TaxAmount: decimal.Zero, TaxInclusive: true, Currency: "CNY", ExchangeRate: decimal.NewFromInt(1), ExchangeRateSource: "SYSTEM", ExchangeRateDate: day, BaseCurrency: "CNY", BaseCurrencyAmount: q.Mul(decimal.NewFromInt(2)), Version: 1}
	if _, e = fees.Add(ctx, f.organizationID, o.ID, fee, &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "air.direct.fee.test", Result: "success"}); e != nil {
		t.Fatal(e)
	}
	lockRepo := NewOrderLockRepo(d, nil)
	state, e := lockRepo.GetOrderLockState(ctx, f.organizationID, o.ID, p)
	if e != nil {
		t.Fatal(e)
	}
	review := biz.OrderLockReviewInput{ExpectedFeeContentVersion: state.FeeContentVersion, FeeContentFingerprint: state.FeeContentFingerprint, AirDocumentFingerprint: state.AirDocumentFingerprint, Comment: "已核对直单与费用"}
	result, e := lockRepo.LockOrder(ctx, p, o.ID, state.OrderVersion, uuid.NewString(), review, nil)
	if e != nil {
		t.Fatal(e)
	}
	record := d.db.OrderLockRecord.GetX(ctx, result.LockRecord.ID)
	if record.AirDocumentSnapshot["document_mode"] != "DIRECT" || record.AirDocumentSnapshot["house"] != "null" || record.AirDocumentSnapshot["link_version"] == "" {
		t.Fatalf("DIRECT审核快照错误：%+v", record.AirDocumentSnapshot)
	}
	o, e = uc.Get(ctx, f.organizationID, o.ID)
	if e != nil {
		t.Fatal(e)
	}
	private = *o
	private.Notes = "锁后拒绝"
	if _, e = uc.UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &private); e == nil {
		t.Fatal("锁后仍可改直单")
	}
	versions, total, e := air.HouseVersions(ctx, f.organizationID, o.ID, biz.SelectorListOptions{Page: 1, PageSize: 20})
	if e != nil || total != 0 || len(versions) != 0 {
		t.Fatalf("DIRECT存在虚假分单历史：%v", e)
	}
}

func TestAirDirectConcurrentAttachAndHistoricalModePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	first := f.create(t, directAirInput(f, "78112345675"))
	kind := biz.OrderTerminationOther
	first, e := uc.TransitionTermination(ctx, f.organizationID, f.actorID, first.ID, first.Version, biz.OrderTerminationTerminating, &kind, "取消")
	if e != nil {
		t.Fatal(e)
	}
	first, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, first.ID, first.Version, biz.OrderTerminationTerminated, &kind, "结束")
	if e != nil {
		t.Fatal(e)
	}
	m := first.AirExport.Master
	operations := make([]func(context.Context) (*biz.Order, error), 2)
	for i := range operations {
		operations[i] = func(ctx context.Context) (*biz.Order, error) {
			in := directAirInput(f, "78112345675")
			in.AirExportInput.MasterWaybillID = &m.ID
			in.AirExportInput.ExpectedMasterVersion = m.Version
			in.AirExportInput.Master = &m.Content
			return uc.Create(ctx, f.organizationID, f.actorID, in)
		}
	}
	results := runOrderWritesConcurrently(operations...)
	success := 0
	for _, result := range results {
		if result.err == nil {
			success++
		} else if !errors.Is(result.err, biz.ErrAirWaybillConflict) {
			t.Fatalf("并发直单未知错误：%v", result.err)
		}
	}
	if success != 1 || d.db.AirMasterWaybillOrderLink.Query().Where(airlink.MasterWaybillIDEQ(m.ID), airlink.StatusEQ("ACTIVE")).CountX(ctx) != 1 {
		t.Fatal("并发直单产生多个当前成员")
	}
	if d.db.AirHouseWaybill.Query().CountX(ctx) != 0 {
		t.Fatal("并发直单产生分单")
	}
	// 同一主单历史DIRECT身份不能在零成员时被HOUSE复用。
	currentRow := d.db.Order.Query().Where(orderent.TerminationStatusEQ(orderent.TerminationStatusACTIVE)).OnlyX(ctx)
	current, e := uc.TransitionTermination(ctx, f.organizationID, f.actorID, currentRow.ID, currentRow.Version, biz.OrderTerminationTerminating, &kind, "取消新票")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = uc.TransitionTermination(ctx, f.organizationID, f.actorID, current.ID, current.Version, biz.OrderTerminationTerminated, &kind, "终止新票"); e != nil {
		t.Fatal(e)
	}
	m.Version = d.db.AirMasterWaybill.GetX(ctx, m.ID).Version
	mixed := f.input(m.Content.MasterNo, "H-HISTORY")
	mixed.AirExportInput.MasterWaybillID = &m.ID
	mixed.AirExportInput.ExpectedMasterVersion = m.Version
	mixed.AirExportInput.Master = &m.Content
	if _, e = uc.Create(ctx, f.organizationID, f.actorID, mixed); e == nil {
		t.Fatal("HOUSE混入历史DIRECT主单")
	}
}
