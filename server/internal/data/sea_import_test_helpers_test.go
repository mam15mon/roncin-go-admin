package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"testing"
)

// 费用相邻夹具也必须使用合法进口摘要及真实海运单证，不能拿SI冒充非海运。
func seedSeaImportTestResources(t *testing.T, d *Data, ctx context.Context, o *ent.Order) {
	t.Helper()
	if biz.OrderBusinessType(o.BusinessType).IsImport() {
		state, e := marshalImportHandlingState(biz.EmptyImportHandling(o.ID, o.Version))
		if e != nil {
			t.Fatal(e)
		}
		d.db.OrderImportHandling.Create().SetOrderID(o.ID).SetState(state).SaveX(ctx)
	}
	carrier := d.db.ShippingLine.Create().SetScacCode(newTestSCAC()).SetNameZh("进口测试船司").SetNameEn("Import carrier").SetCountryCode("CN").SaveX(ctx)
	master := d.db.SeaMasterBill.Create().SetBusinessType(string(o.BusinessType)).SetOrganizationID(o.OrganizationID).SetShippingLineID(carrier.ID).SetMasterNo(uuid.NewString()).SetNormalizedMasterNo(uuid.NewString()).SaveX(ctx)
	execution := d.db.SeaTransportExecution.Create().SetOrganizationID(o.OrganizationID).SetShippingLineID(carrier.ID).SetNillableEtd(parseOptionalTime(o.Etd)).SetNillableEta(parseOptionalTime(o.Eta)).SaveX(ctx)
	d.db.Order.UpdateOne(o).SetShippingLineID(carrier.ID).SaveX(ctx)
	d.db.SeaMasterBillOrderLink.Create().SetOrganizationID(o.OrganizationID).SetMasterBillID(master.ID).SetTransportExecutionID(execution.ID).SetOrderID(o.ID).SetDocumentStructure("DIRECT").SaveX(ctx)
}
