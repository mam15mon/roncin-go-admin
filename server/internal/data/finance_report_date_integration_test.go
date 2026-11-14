package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

func TestFinanceReportAuthoritativeAirAndImportDatePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newReportFixture(t, d)
	d.db.Order.UpdateOneID(f.order.ID).SetBusinessType(orderent.BusinessTypeAE).SetEtd("2026-01-01T00:00:00Z").SaveX(ctx)
	airline := d.db.Airline.Create().SetIataCode("RZ").SetNameZh("报表航空").SetNameEn("Report Airline").SetCountryCode("CN").SaveX(ctx)
	airport := d.db.Airport.Create().SetIataCode("RZZ").SetIcaoCode("RZZZ").SetNameEn("Report Airport").SetCountryCode("CN").SaveX(ctx)
	master := d.db.AirMasterWaybill.Create().SetOrganizationID(f.org).SetMasterNo("99912345678").SetAirlineID(airline.ID).SetOriginAirportID(airport.ID).SetDestinationAirportID(airport.ID).SetEtd("2026-04-30T17:00:00Z").SaveX(ctx)
	d.db.AirMasterWaybillOrderLink.Create().SetOrganizationID(f.org).SetOrderID(f.order.ID).SetMasterWaybillID(master.ID).SaveX(ctx)
	f.fee(f.order.ID, feeent.DirectionRECEIVABLE, "100", "100")
	imported := d.db.Order.Create().SetOrganizationID(f.org).SetIdempotencyKey(uuid.NewString()).SetOrderNo("IMPORT").SetCustomerID(f.party).SetBusinessType(orderent.BusinessTypeSI).SetTradeDirection(orderent.TradeDirectionImport).SetPaymentTerm(orderent.PaymentTermPREPAID).SetEtd("2026-01-01T00:00:00Z").SetEta("2026-05-01T03:00:00+08:00").SaveX(ctx)
	f.fee(imported.ID, feeent.DirectionPAYABLE, "40", "40")
	report, err := NewFinanceReportRepo(d).Profit(ctx, f.org, biz.FinanceReportFilter{Page: 1, PageSize: 20, DateFrom: "2026-05-01", DateTo: "2026-05-01"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || report.Totals[0].ProfitAmount.String() != "60" {
		t.Fatalf("AE应取权威主单ETD，进口应取ETA并按中国业务日筛选: %#v", report)
	}
	for _, row := range report.Items {
		if row.BusinessDate == nil || *row.BusinessDate != "2026-05-01" {
			t.Fatalf("业务日期来源错误: %#v", row)
		}
	}
}
