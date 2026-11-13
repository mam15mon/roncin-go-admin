package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func TestFeeLedgerMasterNoPostgres(t *testing.T) {
	for _, business := range []order.BusinessType{order.BusinessTypeSE, order.BusinessTypeAE} {
		t.Run(string(business), func(t *testing.T) {
			d, cleanup := getIntegrationData(t)
			t.Cleanup(cleanup)
			f := newFeeLedgerPostgresFixture(t, d)
			ctx := context.Background()
			d.db.Order.UpdateOneID(f.orderID).SetBusinessType(business).SaveX(ctx)
			feeID := f.createLedgerFee("master-no", "300")
			at := time.Now().UTC().Truncate(time.Microsecond)
			var currentNo, oldNo, search string
			var endCurrent func()
			if business == order.BusinessTypeSE {
				currentNo, oldNo, search = "MBLNEW12345", "MBLOLD98765", "new123"
				carrier := d.db.ShippingLine.Create().SetScacCode(newTestSCAC()).SetNameZh("台账测试").SetNameEn("Ledger Test").SetCountryCode("CN").SaveX(ctx)
				execution := d.db.SeaTransportExecution.Create().SetOrganizationID(f.organizationID).SetShippingLineID(carrier.ID).SaveX(ctx)
				createMaster := func(no string) *ent.SeaMasterBill {
					return d.db.SeaMasterBill.Create().SetOrganizationID(f.organizationID).SetShippingLineID(carrier.ID).SetMasterNo(no).SetNormalizedMasterNo(no).SaveX(ctx)
				}
				old, current := createMaster(oldNo), createMaster(currentNo)
				d.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetMasterBillID(old.ID).SetTransportExecutionID(execution.ID).SetDocumentStructure("DIRECT").SetStatus("ENDED").SetEndedAt(at.Add(-time.Hour)).SetEndedReason(orderTerminationLinkEndedReason).SaveX(ctx)
				link := d.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetMasterBillID(current.ID).SetTransportExecutionID(execution.ID).SetDocumentStructure("DIRECT").SaveX(ctx)
				endCurrent = func() {
					d.db.SeaMasterBillOrderLink.UpdateOne(link).SetStatus("ENDED").SetEndedAt(at).SetEndedReason(orderTerminationLinkEndedReason).SaveX(ctx)
				}
				t.Cleanup(func() {
					d.db.SeaMasterBillOrderLink.Delete().Where(seamasterbillorderlink.OrderIDEQ(f.orderID)).ExecX(ctx)
					d.db.SeaMasterBill.DeleteOne(old).ExecX(ctx)
					d.db.SeaMasterBill.DeleteOne(current).ExecX(ctx)
					d.db.SeaTransportExecution.DeleteOne(execution).ExecX(ctx)
					d.db.ShippingLine.DeleteOne(carrier).ExecX(ctx)
				})
			} else {
				currentNo, oldNo, search = "78112345675", "78112345682", "781-12345675"
				airline := d.db.Airline.Create().SetIataCode("LT").SetNameEn("Ledger Test").SetCountryCode("CN").SaveX(ctx)
				airport := d.db.Airport.Create().SetIataCode("LGR").SetNameEn("Ledger Airport").SetCountryCode("CN").SaveX(ctx)
				createMaster := func(no string) *ent.AirMasterWaybill {
					return d.db.AirMasterWaybill.Create().SetOrganizationID(f.organizationID).SetAirlineID(airline.ID).SetOriginAirportID(airport.ID).SetDestinationAirportID(airport.ID).SetMasterNo(no).SaveX(ctx)
				}
				old, current := createMaster(oldNo), createMaster(currentNo)
				d.db.AirMasterWaybillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetMasterWaybillID(old.ID).SetStatus("ENDED").SetEndedAt(at.Add(-time.Hour)).SetEndedReason("订单终止").SaveX(ctx)
				link := d.db.AirMasterWaybillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderID).SetMasterWaybillID(current.ID).SaveX(ctx)
				endCurrent = func() {
					d.db.AirMasterWaybillOrderLink.UpdateOne(link).SetStatus("ENDED").SetEndedAt(at).SetEndedReason("订单终止").SaveX(ctx)
				}
				t.Cleanup(func() {
					d.db.AirMasterWaybillOrderLink.Delete().Where(airmasterwaybillorderlink.OrderIDEQ(f.orderID)).ExecX(ctx)
					d.db.AirMasterWaybill.DeleteOne(old).ExecX(ctx)
					d.db.AirMasterWaybill.DeleteOne(current).ExecX(ctx)
					d.db.Airport.DeleteOne(airport).ExecX(ctx)
					d.db.Airline.DeleteOne(airline).ExecX(ctx)
				})
			}
			repo := NewSettlementRepo(d)
			orgs := []uuid.UUID{f.organizationID}
			check := func() {
				t.Helper()
				for _, filter := range []biz.FeeLedgerFilter{{MasterNo: search}, {Keyword: search}} {
					filter.Page, filter.PageSize = 1, 1
					list, err := repo.ListFeeLedger(ctx, orgs, filter)
					if err != nil {
						t.Fatal(err)
					}
					if list.Total != 1 || len(list.Items) != 1 || list.Items[0].MasterNo != currentNo || list.MatchedSummary.Count != 1 {
						t.Fatalf("主单号筛选、分页前汇总及展示不一致: %+v", list)
					}
				}
				detail, err := repo.GetFeeLedgerOrderDetail(ctx, orgs, f.orderID)
				if err != nil {
					t.Fatal(err)
				}
				item := findFeeLedgerItem(detail.Items, feeID)
				if item == nil || item.MasterNo != currentNo {
					t.Fatalf("详情主单号错误: %+v %v", detail, err)
				}
				for _, filter := range []biz.FeeLedgerFilter{{MasterNo: oldNo}, {MasterNo: "-"}, {MasterNo: search, Currency: "USD"}} {
					filter.Page, filter.PageSize = 1, 20
					list, err := repo.ListFeeLedger(ctx, orgs, filter)
					if err != nil || list.Total != 0 {
						t.Fatalf("旧关联或不匹配的组合条件不应命中: %+v %v", list, err)
					}
				}
				list, err := repo.ListFeeLedger(ctx, []uuid.UUID{uuid.New()}, biz.FeeLedgerFilter{Page: 1, PageSize: 20, MasterNo: search})
				if err != nil || list.Total != 0 {
					t.Fatalf("主单号搜索不得跨公司: %+v %v", list, err)
				}
			}
			check()
			endCurrent()
			d.db.Order.UpdateOneID(f.orderID).SetTerminationStatus(order.TerminationStatusTERMINATED).SetTerminationType(order.TerminationTypeOTHER).SetTerminationReason("台账测试终止").SetTerminatedAt(at).SetTerminatedBy(f.actorID).SaveX(ctx)
			check()
			// 无本次终止保留关系时不能任取旧单号。
			d.db.Order.UpdateOneID(f.orderID).SetTerminatedAt(at.Add(time.Minute)).SaveX(ctx)
			list, err := repo.ListFeeLedger(ctx, orgs, biz.FeeLedgerFilter{Page: 1, PageSize: 20})
			if err != nil || len(list.Items) != 1 || list.Items[0].MasterNo != "" {
				t.Fatalf("缺少本次关联应保留费用并显示空主单号: %+v %v", list, err)
			}
		})
	}
}
