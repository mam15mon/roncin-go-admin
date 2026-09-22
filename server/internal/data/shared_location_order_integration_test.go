package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestSharedLocationOrderReadPostgres(t *testing.T) {
	f := newSeaDocumentChangeFixture(t)
	ctx := t.Context()
	shared := f.data.db.Port.Create().SetUnLocode("CNSHA").SetNameEn("Shanghai").SetNameZh("上海").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SaveX(ctx)
	local := f.data.db.Port.Create().SetUnLocode("CNNGB").SetNameEn("Ningbo").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SaveX(ctx)
	link := f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID)
	f.data.db.SeaTransportExecution.UpdateOneID(link.TransportExecutionID).SetOriginLocationID(shared.ID).SetDischargeLocationID(local.ID).SetTransitLocationID(shared.ID).SaveX(ctx)
	repo := NewSeaMasterBillRepo(f.data)
	uc := biz.NewOrderUsecase(NewOrderRepo(f.data), nil, repo, nil, nil, nil)
	check := func(t *testing.T) {
		t.Helper()
		summaries, err := repo.GetSummariesByOrderIDs(ctx, f.orgID, []uuid.UUID{f.orderID})
		if err != nil {
			t.Fatal(err)
		}
		summary := summaries[f.orderID]
		if summary == nil || summary.OriginLocationName != "上海 / Shanghai (CNSHA)" || summary.DischargeLocationName != "Ningbo (CNNGB)" || summary.TransitLocationName != summary.OriginLocationName {
			t.Fatalf("公共港口名称应完整解析: %+v", summary)
		}
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		candidate, err := repo.MatchCandidate(ctx, f.orgID, mbl.ShippingLineID, mbl.NormalizedMasterNo, nil)
		if err != nil || candidate == nil || !candidate.Matched {
			t.Fatalf("共享港口候选匹配失败: %+v %v", candidate, err)
		}
		result, err := uc.List(ctx, []biz.OrderOrganizationScope{{OrganizationIDs: []uuid.UUID{f.orgID}, BusinessType: biz.OrderBusinessSE}}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessSE})
		if err != nil || result == nil || len(result.Items) != 1 || result.Items[0].SeaMasterBill == nil {
			t.Fatalf("共享港口不得阻断订单列表: %+v %v", result, err)
		}
	}
	t.Run("公共港口及重复中转ID", check)
	t.Run("拆分改配校验公共港口", func(t *testing.T) {
		mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
		target := &biz.SeaOrderSplitTargetInput{ShippingLineID: &mbl.ShippingLineID, OriginLocationID: &shared.ID, DischargeLocationID: &local.ID}
		if err := validateTransportExecutionTargetInput(ctx, f.data.db, f.orgID, target, false); err != nil {
			t.Fatalf("拆分改配应允许公共港口: %v", err)
		}
	})
	shared.Update().SetEnabled(false).SaveX(ctx)
	t.Run("已保存引用不受停用影响", check)
	r := &seaMasterBillRepo{data: f.data}
	for _, id := range []uuid.UUID{uuid.New()} {
		if err := r.populateLocationAndShippingLineNames(ctx, f.data.db, f.orgID, &biz.SeaTransportExecution{OriginLocationID: id}); err != biz.ErrSeaMasterBillNotFound {
			t.Fatalf("缺失港口应拒绝: %v", err)
		}
	}
}

func TestSharedLocationOrderValidationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := t.Context()
	org := data.db.Organization.Create().SetCode("LOCAL").SetName("本公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	cases := []struct {
		name    string
		enabled bool
		valid   bool
	}{
		{"启用公共地点", true, true}, {"停用公共地点", false, false},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := string(rune('A' + i))
			port := data.db.Port.Create().SetUnLocode("CNAA" + code).SetNameEn(c.name).SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(c.enabled).SaveX(ctx)
			airport := data.db.Airport.Create().SetIataCode("AA" + code).SetNameEn(c.name).SetCountryCode("CN").SetEnabled(c.enabled).SaveX(ctx)
			for _, validate := range []func(*ent.Tx) error{
				func(tx *ent.Tx) error { return validatePortIDs(ctx, tx, org.ID, []uuid.UUID{port.ID}) },
				func(tx *ent.Tx) error { return validateAirportIDs(ctx, tx, org.ID, []uuid.UUID{airport.ID}) },
			} {
				err := data.WithTx(ctx, validate)
				if c.valid && err != nil || !c.valid && err != biz.ErrOrderInvalidArgument {
					t.Fatalf("港口机场写入范围错误: %v", err)
				}
			}
		})
	}
	err := data.WithTx(ctx, func(tx *ent.Tx) error { return validatePortIDs(ctx, tx, org.ID, []uuid.UUID{uuid.New()}) })
	if err != biz.ErrOrderInvalidArgument {
		t.Fatalf("缺失港口应拒绝: %v", err)
	}
	err = data.WithTx(ctx, func(tx *ent.Tx) error { return validateAirportIDs(ctx, tx, org.ID, []uuid.UUID{uuid.New()}) })
	if err != biz.ErrOrderInvalidArgument {
		t.Fatalf("缺失机场应拒绝: %v", err)
	}
}
