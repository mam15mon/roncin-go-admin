package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
)

func TestAirOrderLocationFiltersFollowSharedMasterPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	a := f.create(t, f.input("78112345675", "LOCATION-A"))
	input := f.input("78112345675", "LOCATION-B")
	master := a.AirExport.Master
	input.AirExportInput.MasterWaybillID = &master.ID
	input.AirExportInput.ExpectedMasterVersion = master.Version
	b := f.create(t, input)
	master = b.AirExport.Master
	newOrigin := d.db.Airport.Create().SetIataCode("N" + newTestSCAC()[:2]).SetNameEn("New Origin").SetCountryCode("CN").SaveX(ctx)
	newDestination := d.db.Airport.Create().SetIataCode("D" + newTestSCAC()[:2]).SetNameEn("New Destination").SetCountryCode("US").SaveX(ctx)
	next := master.Content
	next.OriginAirportID = newOrigin.ID
	next.DestinationAirportID = newDestination.ID
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	preview, err := air.Preview(ctx, f.organizationID, master.ID, master.Version, &next)
	if err != nil || !preview.CanExecute {
		t.Fatalf("机场变更预览失败: preview=%+v err=%v", preview, err)
	}
	if err := air.Update(ctx, f.organizationID, f.actorID, master.ID, master.Version, &next, preview.InputFingerprint, "机场调整"); err != nil {
		t.Fatal(err)
	}
	// 旧冗余地点仍在订单表，筛选必须与详情主单投影一致。
	stored := d.db.Order.GetX(ctx, a.ID)
	if stored.OriginLocationID == nil || *stored.OriginLocationID != f.originID {
		t.Fatal("测试未构造旧订单地点投影")
	}
	repo := NewOrderRepo(d)
	scopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessAE, OrganizationIDs: []uuid.UUID{f.organizationID}}}
	for _, tc := range []struct {
		name                string
		origin, destination *uuid.UUID
		want                int
	}{
		{"新起运机场", &newOrigin.ID, nil, 2},
		{"旧起运机场", &f.originID, nil, 0},
		{"新目的机场", nil, &newDestination.ID, 2},
		{"旧目的机场", nil, &f.destinationID, 0},
		{"新机场组合", &newOrigin.ID, &newDestination.ID, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list, err := repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessAE, OriginLocationID: tc.origin, DestinationLocationID: tc.destination, Page: 1, PageSize: 1})
			if err != nil || list.Total != tc.want {
				t.Fatalf("机场筛选错误: list=%+v err=%v", list, err)
			}
			if tc.want > 0 {
				if len(list.Items) != 1 || *list.Items[0].OriginLocationID != newOrigin.ID || *list.Items[0].DestinationLocationID != newDestination.ID {
					t.Fatalf("分页或返回投影错误: %+v", list)
				}
				second, err := repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessAE, OriginLocationID: tc.origin, DestinationLocationID: tc.destination, Page: 2, PageSize: 1})
				if err != nil || second.Total != 2 || len(second.Items) != 1 || second.Items[0].ID == list.Items[0].ID {
					t.Fatalf("翻页丢失同主单成员: second=%+v err=%v", second, err)
				}
			}
		})
	}
	// 终止后没有活动 Link，详情仍投影最后一次关联主单，地点筛选同样命中。
	uc := f.newUsecase()
	current, err := uc.Get(ctx, f.organizationID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	kind := biz.OrderTerminationOther
	current, err = uc.TransitionTermination(ctx, f.organizationID, f.actorID, b.ID, current.Version, biz.OrderTerminationTerminating, &kind, "客户取消")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.TransitionTermination(ctx, f.organizationID, f.actorID, b.ID, current.Version, biz.OrderTerminationTerminated, &kind, "终止完成"); err != nil {
		t.Fatal(err)
	}
	list, err := repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessAE, OriginLocationID: &newOrigin.ID, Page: 1, PageSize: 20})
	if err != nil || list.Total != 2 {
		t.Fatalf("终止订单的历史关联地点未命中: list=%+v err=%v", list, err)
	}
	// 没有关联的异常 AE 不能靠旧冗余字段在机场筛选下冒充有效主单。
	missing := f.create(t, f.input("78112345686", "LOCATION-MISSING"))
	d.db.AirMasterWaybillOrderLink.Delete().Where(airlink.OrderIDEQ(missing.ID)).ExecX(ctx)
	list, err = repo.List(ctx, scopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessAE, OriginLocationID: &f.originID, Page: 1, PageSize: 20})
	if err != nil || list.Total != 0 {
		t.Fatalf("缺关联的 AE 不应回退旧订单机场: list=%+v err=%v", list, err)
	}
	// 非空运订单继续使用订单地点字段。
	seaOrigin := f.createPort(ctx, "CN"+newTestSCAC()[:3], "海运起运港")
	seaDestination := f.createPort(ctx, "CN"+newTestSCAC()[:3], "海运目的港")
	seaInput := f.validInput()
	seaInput.OriginLocationID = &seaOrigin.ID
	seaInput.DestinationLocationID = &seaDestination.ID
	sea := f.create(t, seaInput)
	seaScopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{f.organizationID}}}
	list, err = repo.List(ctx, seaScopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, OriginLocationID: &seaOrigin.ID, DestinationLocationID: &seaDestination.ID, Page: 1, PageSize: 20})
	if err != nil || list.Total != 1 || list.Items[0].ID != sea.ID {
		t.Fatalf("海运地点筛选受到空运逻辑干扰: list=%+v err=%v", list, err)
	}
	otherScopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessAE, OrganizationIDs: []uuid.UUID{uuid.New()}}}
	list, err = repo.List(ctx, otherScopes, biz.OrderListOptions{BusinessType: biz.OrderBusinessAE, OriginLocationID: &newOrigin.ID, Page: 1, PageSize: 20})
	if err != nil || list.Total != 0 {
		t.Fatalf("地点筛选泄漏其他公司: list=%+v err=%v", list, err)
	}
}
