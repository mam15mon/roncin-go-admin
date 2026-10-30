package data

import (
	"errors"
	"reflect"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	ordercargoent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargocategory"
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
	t.Cleanup(cleanup)
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
				func(tx *ent.Tx) error {
					return validatePortIDs(ctx, tx, org.ID, []uuid.UUID{port.ID, port.ID, port.ID, port.ID})
				},
				func(tx *ent.Tx) error {
					return validateAirportIDs(ctx, tx, org.ID, []uuid.UUID{airport.ID, airport.ID, airport.ID, airport.ID})
				},
			} {
				err := data.WithTx(ctx, validate)
				if c.valid && err != nil || !c.valid && !errors.Is(err, biz.ErrOrderInvalidArgument) {
					t.Fatalf("港口机场写入范围错误: %v", err)
				}
			}
		})
	}
	err := data.WithTx(ctx, func(tx *ent.Tx) error { return validatePortIDs(ctx, tx, org.ID, []uuid.UUID{uuid.New()}) })
	if !errors.Is(err, biz.ErrOrderPortInvalid) || kratoserrors.FromError(err).Message != biz.ErrOrderPortInvalid.Message {
		t.Fatalf("缺失港口应拒绝: %v", err)
	}
	err = data.WithTx(ctx, func(tx *ent.Tx) error { return validateAirportIDs(ctx, tx, org.ID, []uuid.UUID{uuid.New()}) })
	if !errors.Is(err, biz.ErrOrderAirportInvalid) || kratoserrors.FromError(err).Message != biz.ErrOrderAirportInvalid.Message {
		t.Fatalf("缺失机场应拒绝: %v", err)
	}
}

func TestSharedLocationOrderUnchangedUpdatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	for _, business := range []biz.OrderBusinessType{biz.OrderBusinessSE, biz.OrderBusinessAE} {
		t.Run(string(business), func(t *testing.T) {
			for combinationIndex, combination := range []string{"起运目的相同", "四字段相同", "不同有效地点"} {
				name := combination
				if business == biz.OrderBusinessAE {
					name = []string{"起运卸货相同", "四字段复用两个机场", "不同有效机场"}[combinationIndex]
				}
				t.Run(name, func(t *testing.T) {
					var fixture *orderPostgresFixture
					var input *biz.Order
					var firstID, secondID, disabledID uuid.UUID
					var invalidErr error
					if business == biz.OrderBusinessSE {
						fixture = newOrderPostgresFixture(t, data)
						input = fixture.validInput()
						for i := range 3 {
							code := string([]byte{'C', 'N', 'A' + byte(combinationIndex), 'A', 'A' + byte(i)})
							port := data.db.Port.Create().SetUnLocode(code).SetNameEn("Location validation port").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(i < 2).SaveX(ctx)
							switch i {
							case 0:
								firstID = port.ID
							case 1:
								secondID = port.ID
							case 2:
								disabledID = port.ID
							}
						}
						invalidErr = biz.ErrOrderPortInvalid
					} else {
						air := newAirExportFixture(t, data)
						fixture = air.orderPostgresFixture
						input = air.input("78112345675", "LOCATION-HOUSE")
						firstID, secondID = air.originID, air.destinationID
						disabledID = data.db.Airport.Create().SetIataCode(string([]byte{'Z', 'A' + byte(combinationIndex), 'Z'})).SetNameEn("Disabled location validation airport").SetCountryCode("CN").SetEnabled(false).SaveX(ctx).ID
						invalidErr = biz.ErrOrderAirportInvalid
					}
					input.OriginLocationID, input.DestinationLocationID = &firstID, &firstID
					switch combination {
					case "四字段相同":
						input.DischargeLocationID, input.TransitLocationID = &firstID, &firstID
					case "不同有效地点":
						input.DestinationLocationID, input.TransitLocationID = &secondID, &secondID
					}
					if business == biz.OrderBusinessAE {
						// 空运共享主单仍遵循既有起降机场不同规则，复用的是卸货/中转引用。
						input.DestinationLocationID = &secondID
						if combination != "不同有效地点" {
							input.DischargeLocationID = &firstID
						}
						invalidMaster := *input.AirExportInput.Master
						invalidMaster.DestinationAirportID = invalidMaster.OriginAirportID
						if _, err := biz.NormalizeAirMasterInput(&invalidMaster); !errors.Is(err, biz.ErrAirWaybillInvalid) {
							t.Fatalf("机场引用去重不得放宽起降机场不同规则: %v", err)
						}
					}
					uc := fixture.newUsecase()
					created, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, input)
					if err != nil {
						t.Fatalf("重复有效地点应允许建票: %v", err)
					}
					refreshed, err := uc.Get(ctx, fixture.organizationID, created.ID)
					if err != nil {
						t.Fatal(err)
					}
					update := *refreshed
					update.IdempotencyKey = uuid.NewString()
					if business == biz.OrderBusinessSE {
						version := refreshed.SeaMasterBill.Version
						update.SeaMasterBillInput = &biz.SeaMasterBillInput{MasterNo: refreshed.SeaMasterBill.MasterNo, ExpectedCandidateVersion: &version}
					}
					wantLocations := [4]*uuid.UUID{refreshed.OriginLocationID, refreshed.DestinationLocationID, refreshed.DischargeLocationID, refreshed.TransitLocationID}
					updated, err := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, refreshed.Version, &update)
					if err != nil {
						t.Fatalf("刷新后未修改直接保存应通过: %v", err)
					}
					gotLocations := [4]*uuid.UUID{updated.OriginLocationID, updated.DestinationLocationID, updated.DischargeLocationID, updated.TransitLocationID}
					if !reflect.DeepEqual(wantLocations, gotLocations) || updated.Version != refreshed.Version+1 {
						t.Fatalf("无修改保存不得改动地点字段: want=%v got=%v version=%d", wantLocations, gotLocations, updated.Version)
					}
					before := data.db.Order.GetX(ctx, created.ID)
					auditCount := data.db.AuditLog.Query().CountX(ctx)
					for _, invalidID := range []uuid.UUID{uuid.New(), disabledID} {
						bad := update
						bad.IdempotencyKey = uuid.NewString()
						bad.OriginLocationID, bad.DestinationLocationID, bad.TransitLocationID = &firstID, &firstID, &invalidID
						bad.Notes = "地点无效时不得部分保存"
						_, err := uc.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, before.Version, &bad)
						if !errors.Is(err, invalidErr) || kratoserrors.FromError(err).Message != kratoserrors.FromError(invalidErr).Message {
							t.Fatalf("无效地点应拒绝并显示具体原因: %v", err)
						}
						after := data.db.Order.GetX(ctx, created.ID)
						if after.Version != before.Version || after.Notes != before.Notes || !reflect.DeepEqual(after.TransitLocationID, before.TransitLocationID) || data.db.AuditLog.Query().CountX(ctx) != auditCount {
							t.Fatal("无效地点更新必须原子拒绝，不改订单或审计")
						}
					}
					category := data.db.MasterDataItem.Create().SetKind(masterdataent.KindCargoCategory).SetCode("DUP-" + uuid.NewString()).SetName("禁止重复的货物分类").SaveX(ctx)
					err = data.WithTx(ctx, func(tx *ent.Tx) error {
						return validateMasterDataIDs(ctx, tx, fixture.organizationID, []uuid.UUID{category.ID, category.ID}, masterdataent.KindCargoCategory)
					})
					if !errors.Is(err, biz.ErrOrderInvalidArgument) || data.db.OrderCargoCategory.Query().Where(ordercargoent.OrderIDEQ(created.ID)).CountX(ctx) != 0 {
						t.Fatal("地点去重不得放宽货物分类重复约束")
					}
				})
			}
		})
	}
}
