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
	local := f.data.db.Port.Create().SetOrganizationID(f.orgID).SetUnLocode("CNNGB").SetNameEn("Ningbo").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SaveX(ctx)
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
			t.Fatalf("共享及本地港口名称应完整解析: %+v", summary)
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
	t.Run("共享与本地港口及重复中转ID", check)
	f.data.db.Port.Create().SetOrganizationID(f.orgID).SetUnLocode("CNSHA").SetNameEn("Local Shanghai").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SaveX(ctx)
	shared.Update().SetEnabled(false).SaveX(ctx)
	t.Run("已保存引用不受同码遮蔽及停用影响", check)
	foreignOrg := f.data.db.Organization.Create().SetCode("OTHER").SetName("其他公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	foreign := f.data.db.Port.Create().SetOrganizationID(foreignOrg.ID).SetUnLocode("CNTAO").SetNameEn("Qingdao").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SaveX(ctx)
	r := &seaMasterBillRepo{data: f.data}
	for _, id := range []uuid.UUID{foreign.ID, uuid.New()} {
		if err := r.populateLocationAndShippingLineNames(ctx, f.data.db, f.orgID, &biz.SeaTransportExecution{OriginLocationID: id}); err != biz.ErrSeaMasterBillNotFound {
			t.Fatalf("其他公司及缺失港口应拒绝: %v", err)
		}
	}
}

func TestSharedLocationOrderValidationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := t.Context()
	org := data.db.Organization.Create().SetCode("LOCAL").SetName("本公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	foreign := data.db.Organization.Create().SetCode("OTHER").SetName("其他公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	cases := []struct {
		name    string
		orgID   *uuid.UUID
		enabled bool
		valid   bool
	}{
		{"共享", nil, true, true}, {"本地", &org.ID, true, true}, {"其他公司", &foreign.ID, true, false}, {"停用共享", nil, false, false}, {"停用本地", &org.ID, false, false},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := string(rune('A' + i))
			port := data.db.Port.Create().SetNillableOrganizationID(c.orgID).SetUnLocode("CNAA" + code).SetNameEn(c.name).SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(c.enabled).SaveX(ctx)
			airport := data.db.Airport.Create().SetNillableOrganizationID(c.orgID).SetIataCode("AA" + code).SetNameEn(c.name).SetCountryCode("CN").SetEnabled(c.enabled).SaveX(ctx)
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
	t.Run("同码本地遮蔽只允许本地新写入", func(t *testing.T) {
		sharedPort := data.db.Port.Create().SetUnLocode("CNSHA").SetNameEn("Shared").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)
		localPort := data.db.Port.Create().SetOrganizationID(org.ID).SetUnLocode("CNSHA").SetNameEn("Local").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)
		sharedAirport := data.db.Airport.Create().SetIataCode("PVG").SetNameEn("Shared").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
		localAirport := data.db.Airport.Create().SetOrganizationID(org.ID).SetIataCode("PVG").SetNameEn("Local").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
		for _, c := range []struct {
			id    uuid.UUID
			port  bool
			valid bool
		}{{sharedPort.ID, true, false}, {localPort.ID, true, true}, {sharedAirport.ID, false, false}, {localAirport.ID, false, true}} {
			err := data.WithTx(ctx, func(tx *ent.Tx) error {
				if c.port {
					return validatePortIDs(ctx, tx, org.ID, []uuid.UUID{c.id})
				}
				return validateAirportIDs(ctx, tx, org.ID, []uuid.UUID{c.id})
			})
			if c.valid && err != nil || !c.valid && err != biz.ErrOrderInvalidArgument {
				t.Fatalf("写校验须与候选遮蔽一致: valid=%v err=%v", c.valid, err)
			}
		}
	})

	err := data.WithTx(ctx, func(tx *ent.Tx) error { return validatePortIDs(ctx, tx, org.ID, []uuid.UUID{uuid.New()}) })
	if err != biz.ErrOrderInvalidArgument {
		t.Fatalf("缺失港口应拒绝: %v", err)
	}
	err = data.WithTx(ctx, func(tx *ent.Tx) error { return validateAirportIDs(ctx, tx, org.ID, []uuid.UUID{uuid.New()}) })
	if err != biz.ErrOrderInvalidArgument {
		t.Fatalf("缺失机场应拒绝: %v", err)
	}
}
