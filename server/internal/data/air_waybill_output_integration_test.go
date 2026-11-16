package data

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// 正式迁移后的真实公司投影；公司抬头变化不应修改运单正文或版本。
func TestAirWaybillOutputCompanyProjectionPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	for _, businessType := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		t.Run(string(businessType), func(t *testing.T) {
			f := newAirExportFixture(t, d)
			d.db.Organization.UpdateOneID(f.organizationID).SetNameEn("ACTUAL OPERATING COMPANY").SaveX(ctx)
			d.db.Organization.UpdateOneID(f.headquartersID).SetNameEn("SYSTEM ORGANIZATION CANNOT ISSUE").SaveX(ctx)
			other := d.db.Organization.Create().SetCode(uuid.NewString()).SetName("另一经营公司").SetKind("company").SetNameEn("OTHER COMPANY CANNOT ISSUE").SetBaseCurrency("CNY").SaveX(ctx)
			in := airInputForBusiness(f, businessType, "78112345675", "H-OUTPUT")
			in.AirExportInput.Master.WaybillContent = &biz.AirWaybillContent{IssuedBy: "SAVED EXTERNAL AIRLINE"}
			in.AirExportInput.House.Content = &biz.AirWaybillContent{IssuedBy: "SAVED HOUSE ISSUER"}
			o := f.create(t, in)
			if o.AirExport.Master.OrganizationNameEn != "ACTUAL OPERATING COMPANY" {
				t.Fatal("订单详情未投影真实经营公司英文名")
			}
			master := o.AirExport.Master
			d.db.Organization.UpdateOneID(f.organizationID).SetNameEn("CHANGED REAL COMPANY").SaveX(ctx)
			repo := NewAirExportRepo(d)
			if _, err := repo.Get(ctx, other.ID, master.ID, biz.SelectorListOptions{Page: 1, PageSize: 1}); !errors.Is(err, biz.ErrAirWaybillNotFound) {
				t.Fatalf("跨公司主单未拒绝：%v", err)
			}
			detail, err := repo.Get(ctx, f.organizationID, master.ID, biz.SelectorListOptions{Page: 1, PageSize: 1})
			if err != nil {
				t.Fatal(err)
			}
			if detail.Master.OrganizationNameEn != "CHANGED REAL COMPANY" || detail.Master.Version != master.Version || detail.Master.ContentHash != master.ContentHash || detail.Master.Content.WaybillContent.IssuedBy != "SAVED EXTERNAL AIRLINE" {
				t.Fatal("公司投影未更新、借错主体或改写已存正文/版本")
			}
			// 公司英文名可以未设置；资料照常读取，输出阻止由制作边界明确解释。
			d.db.Organization.UpdateOneID(f.organizationID).ClearNameEn().SaveX(ctx)
			current, err := f.newUsecase().Get(ctx, f.organizationID, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			if current.AirExport.Master.OrganizationNameEn != "" || current.AirExport.House.Content.IssuedBy != "SAVED HOUSE ISSUER" {
				t.Fatal("缺英文名不应伪造抬头或修改分单印发文字")
			}
		})
	}
}
