package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	personnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

// 这里只替换查询夹具的业务类型；空运建票关系与事务由空运仓储集成测试覆盖。
func TestAirWorkbenchRecentOrdersPostgres(t *testing.T) {
	f := newWorkbenchIntegrationFixture(t)
	ctx := t.Context()
	employee := f.newEmployee("air")
	f.addPersonnel(employee, personnelent.RoleOPERATOR)
	if err := f.data.db.Order.UpdateOneID(f.orderID).SetBusinessType(orderent.BusinessTypeAE).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	repo := f.repo()
	scope := f.scope(employee, nil)
	result, err := repo.ListMyRecentOrders(ctx, scope, biz.WorkbenchOrderFilter{Page: 1, PageSize: 20})
	if err != nil || result.Total != 1 || len(result.Items) != 1 || result.Items[0].BusinessType != "AE" {
		t.Fatalf("空运协作订单未进入工作台: %+v %v", result, err)
	}
	overview, err := repo.GetOverview(ctx, scope)
	if err != nil || len(overview.RecentOrders) != 1 || overview.RecentOrders[0].OrderID != f.orderID {
		t.Fatalf("工作台概览遗漏空运: %+v %v", overview, err)
	}
	for _, scenario := range []struct {
		name   string
		mutate func(*biz.WorkbenchScope)
	}{
		{"仅海运读取权限", func(s *biz.WorkbenchScope) {
			s.ReadableOrderBusinessTypes = []biz.OrderBusinessType{biz.OrderBusinessSE}
		}},
		{"无订单读取权限", func(s *biz.WorkbenchScope) { s.ReadableOrderBusinessTypes = nil }},
		{"跨公司", func(s *biz.WorkbenchScope) { s.OrganizationID = uuid.New() }},
		{"非协作人员", func(s *biz.WorkbenchScope) { s.UserID = uuid.New() }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s := scope
			scenario.mutate(&s)
			page, err := repo.ListMyRecentOrders(t.Context(), s, biz.WorkbenchOrderFilter{Page: 1, PageSize: 20})
			if err != nil || page.Total != 0 || len(page.Items) != 0 {
				t.Fatalf("超出范围的空运协作订单被返回: %+v %v", page, err)
			}
		})
	}
}
