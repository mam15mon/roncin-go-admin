package data

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

func TestAirWeightConfirmationAndFeeSourcePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	principal := f.requesterPrincipal()
	principal.RoleGrants = []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.OrderPermission(access.OrderBusinessAE, access.OrderRead): {}, access.OrderPermission(access.OrderBusinessAE, access.OrderUpdate): {}}}}
	ctx := biz.WithPrincipal(context.Background(), principal)
	c := f.data.db
	a := f.createOrder("AE-WEIGHT-A-" + f.suffix)
	b := f.createOrder("AE-WEIGHT-B-" + f.suffix)
	c.Order.UpdateOne(a).SetBusinessType(orderent.BusinessTypeAE).SaveX(ctx)
	c.Order.UpdateOne(b).SetBusinessType(orderent.BusinessTypeAE).SaveX(ctx)
	ca := c.OrderCargoItem.Create().SetOrganizationID(f.organizationID).SetOrderID(a.ID).SetCargoName("本票实际货物").SetPackageCount(1).SetGrossWeightKg(100).SetVolumeCbm(1.2).SaveX(ctx)
	c.OrderCargoItem.Create().SetOrganizationID(f.organizationID).SetOrderID(b.ID).SetCargoName("其他成员实际货物").SetPackageCount(1).SetGrossWeightKg(300).SetVolumeCbm(0.6).SaveX(ctx)
	airline := c.Airline.Create().SetIataCode("ZT").SetNameEn("Weight Test Air").SetCountryCode("CN").SaveX(ctx)
	origin := c.Airport.Create().SetIataCode("ZTA").SetNameEn("Weight Origin").SetCountryCode("CN").SaveX(ctx)
	destination := c.Airport.Create().SetIataCode("ZTB").SetNameEn("Weight Destination").SetCountryCode("CN").SaveX(ctx)
	master := c.AirMasterWaybill.Create().SetOrganizationID(f.organizationID).SetMasterNo("99912345678").SetAirlineID(airline.ID).SetOriginAirportID(origin.ID).SetDestinationAirportID(destination.ID).SaveX(ctx)
	for _, o := range []*ent.Order{a, b} {
		c.AirMasterWaybillOrderLink.Create().SetOrganizationID(f.organizationID).SetMasterWaybillID(master.ID).SetOrderID(o.ID).SaveX(ctx)
	}
	uc := biz.NewAirWeightUsecase(NewAirWeightRepo(f.data))
	input := &biz.AirWeightInput{Scope: "ORDER", TargetID: a.ID}
	preview := func(in *biz.AirWeightInput) *biz.AirWeightPreview {
		t.Helper()
		p, e := uc.Preview(ctx, f.organizationID, in)
		if e != nil {
			t.Fatal(e)
		}
		return p
	}
	confirm := func(in *biz.AirWeightInput, weight, reason string) *biz.AirWeightConfirmation {
		t.Helper()
		p := preview(in)
		n := *in
		n.InputFingerprint = p.InputFingerprint
		n.ConfirmedWeightKG = weight
		n.Reason = reason
		if p.CurrentConfirmation != nil {
			n.ExpectedVersion = p.CurrentConfirmation.Version
		}
		got, e := uc.Confirm(ctx, f.organizationID, f.requesterID, &n)
		if e != nil {
			t.Fatal(e)
		}
		return got
	}
	if p := preview(input); !p.Available || p.SuggestedChargeableWeightKG != "200" {
		t.Fatalf("本票重量: %+v", p)
	}
	if p := preview(&biz.AirWeightInput{Scope: "MASTER", TargetID: master.ID}); !p.Available || p.SuggestedChargeableWeightKG != "400" {
		t.Fatalf("主单必须独立计算400而非分单合计500: %+v", p)
	}

	// 计划摘要与实际明细独立采用，不能以实际明细冒充计划值。
	c.Order.UpdateOneID(a.ID).SetTotalGrossWeightKg(90).SetTotalVolumeCbm(0.6).SaveX(ctx)
	c.Order.UpdateOneID(b.ID).SetTotalGrossWeightKg(250).SetTotalVolumeCbm(0.6).SaveX(ctx)
	plannedInput := &biz.AirWeightInput{Scope: "ORDER", TargetID: a.ID, Source: "PLANNED"}
	planned := preview(plannedInput)
	if !planned.Available || planned.SuggestedChargeableWeightKG != "100" {
		t.Fatalf("计划来源必须独立于实际200: %+v", planned)
	}
	if p := preview(&biz.AirWeightInput{Scope: "MASTER", TargetID: master.ID, Source: "PLANNED"}); p.SuggestedChargeableWeightKG != "340" {
		t.Fatalf("主单计划汇总独立计算: %+v", p)
	}
	planConfirmed := confirm(plannedInput, "105", "委托确认105")
	c.Order.UpdateOneID(a.ID).SetNotes("私有备注不改变计划重量").AddVersion(1).SaveX(ctx)
	if p := preview(plannedInput); p.ConfirmationStale {
		t.Fatal("私有备注变化不能使计划重量过期")
	}
	confirmed := confirm(input, "130", "客户确认计费重")
	unit, e := c.BillingUnit.Query().Where(billingunitent.CodeEQ("KG")).Only(ctx)
	if e != nil {
		t.Fatal(e)
	}
	repo := NewOrderFeeRepo(f.data).(*orderFeeRepo)
	feePreview, e := repo.ConfirmedQuantityPreview(ctx, f.organizationID, a.ID, unit.ID, "ACTUAL")
	if e != nil || !feePreview.Available || feePreview.Quantity != "130" {
		t.Fatalf("确认重量带入: %+v %v", feePreview, e)
	}
	planFeePreview, e := repo.ConfirmedQuantityPreview(ctx, f.organizationID, a.ID, unit.ID, "PLANNED")
	if e != nil || !planFeePreview.Available || planFeePreview.Quantity != "105" || planFeePreview.QuantitySource.ConfirmationID != planConfirmed.ID {
		t.Fatalf("计划确认带入不得串用实际130: %+v %v", planFeePreview, e)
	}
	gross, e := biz.NewOrderFeeUsecase(repo, nil, nil, nil, nil).PreviewFeeQuantity(ctx, f.organizationID, a.ID, unit.ID, "ACTUAL", "GROSS_WEIGHT")
	if e != nil || gross.Quantity != "100" {
		t.Fatalf("KG默认毛重不得改变: %+v %v", gross, e)
	}
	mkFee := func(quantity string, s *biz.FeeQuantitySelection) *biz.OrderFee {
		return &biz.OrderFee{ID: uuid.New(), OrderID: a.ID, IdempotencyKey: uuid.NewString(), Direction: biz.OrderFeeReceivable, FeeSettingID: &f.feeSettingID, FeeCode: "WEIGHT", FeeName: "空运费", SettlementPartyID: f.customerID, BillingUnitID: &unit.ID, BillingUnit: "公斤", Quantity: decimal.RequireFromString(quantity), QuantitySource: s, UnitPrice: decimal.NewFromInt(2), TotalAmount: decimal.RequireFromString(quantity).Mul(decimal.NewFromInt(2)), TaxInclusive: true, NetAmount: decimal.RequireFromString(quantity).Mul(decimal.NewFromInt(2)), TaxAmount: decimal.Zero, Currency: "CNY", ExchangeRate: decimal.NewFromInt(1), ExchangeRateSource: "SYSTEM", ExchangeRateDate: "2026-10-21", BaseCurrency: "CNY", BaseCurrencyAmount: decimal.RequireFromString(quantity).Mul(decimal.NewFromInt(2)), Version: 1}
	}
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.requesterID, Action: "air.weight.fee.test", Result: "success"}
	}
	direct := mkFee("130", feePreview.QuantitySource)
	saved, e := repo.Add(ctx, f.organizationID, a.ID, direct, audit())
	if e != nil {
		t.Fatal(e)
	}
	if saved.QuantitySource.Mode != "DIRECT" || saved.QuantitySource.ReferenceQuantity != "130" {
		t.Fatalf("未保存权威快照: %+v", saved.QuantitySource)
	}
	adjusted := mkFee("135", &biz.FeeQuantitySelection{Mode: "MANUAL_ADJUSTED", ConfirmationID: confirmed.ID, ConfirmationVersion: confirmed.Version, Reason: "客户另行确认135"})
	if _, e := repo.Add(ctx, f.organizationID, a.ID, adjusted, audit()); e != nil {
		t.Fatal(e)
	}
	spoof := mkFee("135", feePreview.QuantitySource)
	if _, e := repo.Add(ctx, f.organizationID, a.ID, spoof, audit()); !errors.Is(e, biz.ErrAirWeightConflict) {
		t.Fatalf("直接来源不得携带任意数值: %v", e)
	}
	foreign := mkFee("130", feePreview.QuantitySource)
	if _, e := repo.Add(ctx, f.organizationID, b.ID, foreign, audit()); !errors.Is(e, biz.ErrAirWeightNotFound) {
		t.Fatalf("不得引用其他票确认: %v", e)
	}
	oldPreview := preview(input)
	c.OrderCargoItem.UpdateOne(ca).SetVolumeCbm(1.8).SetVersion(ca.Version + 1).SaveX(ctx)
	if p := preview(plannedInput); p.ConfirmationStale {
		t.Fatal("实际明细变化不能误使计划确认过期")
	}
	stale := preview(input)
	if !stale.ConfirmationStale || stale.CurrentConfirmation.ConfirmedWeightKG != "130" {
		t.Fatalf("输入变化保留130并提示过期: %+v", stale)
	}
	staleInput := *input
	staleInput.InputFingerprint = oldPreview.InputFingerprint
	staleInput.ExpectedVersion = confirmed.Version
	staleInput.ConfirmedWeightKG = "130"
	staleInput.Reason = "确认旧依据"
	if _, e := uc.Confirm(ctx, f.organizationID, f.requesterID, &staleInput); !errors.Is(e, biz.ErrAirWeightConflict) {
		t.Fatalf("过期预览必须冲突: %v", e)
	}
	if p, e := repo.ConfirmedQuantityPreview(ctx, f.organizationID, a.ID, unit.ID, "ACTUAL"); e != nil || p.Available {
		t.Fatalf("过期确认禁止带入: %+v %v", p, e)
	}
	if _, e := repo.Add(ctx, f.organizationID, a.ID, mkFee("130", feePreview.QuantitySource), audit()); !errors.Is(e, biz.ErrAirWeightConflict) {
		t.Fatalf("保存必须锁内拒绝过期来源: %v", e)
	}
	// 审计失败与费用写入原子回滚，不留下半套来源快照。
	rollbackFee := mkFee("120", &biz.FeeQuantitySelection{Mode: "MANUAL"})
	badAudit := audit()
	badAudit.Action = ""
	if _, e := repo.Add(ctx, f.organizationID, a.ID, rollbackFee, badAudit); e == nil {
		t.Fatal("审计约束失败应回滚费用")
	}
	if exists, e := c.OrderFee.Query().Where(orderfeeent.IDEQ(rollbackFee.ID)).Exist(ctx); e != nil || exists {
		t.Fatalf("审计失败后不应留费用: %v %v", exists, e)
	}
	reloaded, e := repo.Get(ctx, f.organizationID, a.ID, saved.ID)
	if e != nil || !reloaded.Quantity.Equal(decimal.NewFromInt(130)) || reloaded.QuantitySource.ReferenceQuantity != "130" {
		t.Fatalf("货量变化不得改旧费用: %+v %v", reloaded, e)
	}

	// 不重新采用数量的备注编辑保留历史来源，输入变更不能迫使旧费用静默重取数量。
	note := "只修改费用备注"
	edit := *reloaded
	edit.Note = &note
	updated, e := repo.Update(ctx, f.organizationID, a.ID, saved.ID, &edit, nil, audit())
	if e != nil || !updated.Quantity.Equal(decimal.NewFromInt(130)) || updated.QuantitySource.ConfirmationID != confirmed.ID {
		t.Fatalf("私有编辑须保留历史数量快照: %+v %v", updated, e)
	}
	// 相同确认版本并发提交，仅一条成功，另一条冲突，不出现同版本双写。
	p := preview(input)
	concurrent := *input
	concurrent.InputFingerprint = p.InputFingerprint
	concurrent.ExpectedVersion = confirmed.Version
	concurrent.ConfirmedWeightKG = "300"
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := uc.Confirm(ctx, f.organizationID, f.requesterID, &concurrent)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, biz.ErrAirWeightConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("并发确认成功%d 冲突%d", success, conflicts)
	}

	// 确认重写与直接录费竞争同一订单锁：录费只能在旧确认仍当前时成功，否则冲突。
	latestPreview := preview(input)
	currentSelection, e := repo.ConfirmedQuantityPreview(ctx, f.organizationID, a.ID, unit.ID, "ACTUAL")
	if e != nil || !currentSelection.Available {
		t.Fatalf("当前来源: %+v %v", currentSelection, e)
	}
	racingFee := mkFee(currentSelection.Quantity, currentSelection.QuantitySource)
	reconfirm := *input
	reconfirm.InputFingerprint = latestPreview.InputFingerprint
	reconfirm.ExpectedVersion = latestPreview.CurrentConfirmation.Version
	reconfirm.ConfirmedWeightKG = latestPreview.CurrentConfirmation.ConfirmedWeightKG
	reconfirm.Reason = "并发重新确认依据"
	var sourceErr, confirmErr error
	wg.Add(2)
	go func() { defer wg.Done(); _, sourceErr = repo.Add(ctx, f.organizationID, a.ID, racingFee, audit()) }()
	go func() { defer wg.Done(); _, confirmErr = uc.Confirm(ctx, f.organizationID, f.requesterID, &reconfirm) }()
	wg.Wait()
	if confirmErr != nil {
		t.Fatal(confirmErr)
	}
	if sourceErr != nil && !errors.Is(sourceErr, biz.ErrAirWeightConflict) {
		t.Fatalf("来源竞争应成功或冲突: %v", sourceErr)
	}
	if sourceErr == nil {
		row, e := repo.Get(ctx, f.organizationID, a.ID, racingFee.ID)
		if e != nil || row.QuantitySource.ConfirmationID != currentSelection.QuantitySource.ConfirmationID || row.QuantitySource.ConfirmationVersion != currentSelection.QuantitySource.ConfirmationVersion {
			t.Fatalf("成功写入必须保留当时来源版本: %+v %v", row, e)
		}
	}
	history, total, e := uc.List(ctx, f.organizationID, "ORDER", a.ID, 1, 20)
	if e != nil || total != 4 || len(history) != 4 {
		t.Fatalf("确认历史: %d %v", total, e)
	}
	if _, e := uc.Preview(ctx, uuid.New(), input); !errors.Is(e, biz.ErrPermissionDenied) {
		t.Fatalf("跨公司拒绝: %v", e)
	}

	// 主单确认跟踪完整活动成员集合；加入空票使当前参考不可计算，旧确认仍保留。
	masterActual := &biz.AirWeightInput{Scope: "MASTER", TargetID: master.ID, Source: "ACTUAL"}
	masterPreview := preview(masterActual)
	masterConfirmed := confirm(masterActual, masterPreview.SuggestedChargeableWeightKG, "")
	extra := f.createOrder("AE-WEIGHT-C-" + f.suffix)
	c.Order.UpdateOneID(extra.ID).SetBusinessType(orderent.BusinessTypeAE).SaveX(ctx)
	extraLink := c.AirMasterWaybillOrderLink.Create().SetOrganizationID(f.organizationID).SetMasterWaybillID(master.ID).SetOrderID(extra.ID).SaveX(ctx)
	c.AirMasterWaybill.UpdateOne(master).AddVersion(1).SaveX(ctx)
	changed := preview(masterActual)
	if !changed.ConfirmationStale || changed.Available || changed.CurrentConfirmation.ID != masterConfirmed.ID {
		t.Fatalf("成员变化须使主单依据过期且空票不能伪造参考: %+v", changed)
	}
	if p := preview(plannedInput); p.ConfirmationStale {
		t.Fatal("主单成员变化不能误使未变分单依据过期")
	}
	c.AirMasterWaybillOrderLink.UpdateOne(extraLink).SetStatus("ENDED").AddVersion(1).SaveX(ctx)
	c.AirMasterWaybill.UpdateOneID(master.ID).AddVersion(1).SaveX(ctx)
	// 业务锁属于共享主单全部成员约束；不能从可写 A 绕过锁定的 B。
	f.lockOrder(b)
	masterInput := &biz.AirWeightInput{Scope: "MASTER", TargetID: master.ID}
	mp := preview(masterInput)
	masterInput.InputFingerprint = mp.InputFingerprint
	masterInput.ConfirmedWeightKG = mp.SuggestedChargeableWeightKG
	if _, e := uc.Confirm(ctx, f.organizationID, f.requesterID, masterInput); e == nil {
		t.Fatal("锁定成员必须阻止主单重量确认")
	}
	// 无业务权限的读取在资料查询前拒绝，不泄漏主单合计或确认历史。
	denied := biz.WithPrincipal(context.Background(), f.requesterPrincipal())
	if _, e := uc.Preview(denied, f.organizationID, input); !errors.Is(e, biz.ErrPermissionDenied) {
		t.Fatalf("无权读取: %v", e)
	}
}
