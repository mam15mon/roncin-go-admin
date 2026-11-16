package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	personnel "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

func TestWorkbenchFourTypeIntersectionPostgres(t *testing.T) {
	f := newWorkbenchIntegrationFixture(t)
	ctx := context.Background()
	employee := f.newEmployee("four")
	original := f.orderID
	f.addPersonnel(employee, personnel.RoleOPERATOR)
	abnormalType := workbenchTodoType(t, f, "四类查验", true)
	f.data.db.OrderAbnormalCase.Create().SetOrderID(original).SetAbnormalCaseID(abnormalType.ID).SetMarkedBy(f.actorID).SaveX(ctx)
	for _, business := range []orderent.BusinessType{orderent.BusinessTypeSI, orderent.BusinessTypeAE, orderent.BusinessTypeAI} {
		q := f.data.db.Order.Create().SetOrganizationID(f.organizationID).SetCustomerID(f.customerID).SetOrderNo(string(business) + "-FOUR").SetIdempotencyKey(uuid.NewString()).SetBusinessType(business).SetPaymentTerm(orderent.PaymentTermPREPAID)
		if business == orderent.BusinessTypeSI || business == orderent.BusinessTypeAI {
			q.SetTradeDirection(orderent.TradeDirectionImport).SetFlowStatus(orderent.FlowStatusIMPORT_NOT_STARTED)
		} else {
			q.SetTradeDirection(orderent.TradeDirectionExport)
		}
		o := q.SaveX(ctx)
		f.orderID = o.ID
		f.addPersonnel(employee, personnel.RoleOPERATOR)
		workbenchTodoFee(t, f).SaveX(ctx)
		f.data.db.OrderAbnormalCase.Create().SetOrderID(o.ID).SetAbnormalCaseID(abnormalType.ID).SetMarkedBy(f.actorID).SaveX(ctx)
	}
	f.orderID = original
	all := []biz.OrderBusinessType{biz.OrderBusinessSE, biz.OrderBusinessSI, biz.OrderBusinessAE, biz.OrderBusinessAI}
	scope := f.scope(employee, nil)
	scope.ReadableOrderBusinessTypes = all
	scope.ReadableFeeBusinessTypes = all
	scope.ReadableAbnormalBusinessTypes = all
	assertWorkbenchTodoCounts(t, f, scope, 4, 4)
	for _, business := range all {
		s := scope
		s.ReadableOrderBusinessTypes = []biz.OrderBusinessType{business}
		s.ReadableFeeBusinessTypes = []biz.OrderBusinessType{business}
		s.ReadableAbnormalBusinessTypes = []biz.OrderBusinessType{business}
		assertWorkbenchTodoCounts(t, f, s, 1, 1)
		recent, err := f.repo().ListMyRecentOrders(ctx, s, biz.WorkbenchOrderFilter{Page: 1, PageSize: 20})
		if err != nil || recent.Total != 1 || len(recent.Items) != 1 || recent.Items[0].BusinessType != string(business) {
			t.Fatalf("四类近期分页读取范围错误: %+v %v", recent, err)
		}
		overview, err := f.repo().GetOverview(ctx, s)
		if err != nil {
			t.Fatal(err)
		}
		if len(overview.RecentOrders) != 1 || overview.RecentOrders[0].BusinessType != string(business) {
			t.Fatalf("四类近期摘要遗漏或借权: %+v", overview.RecentOrders)
		}
		// 只有其他类型的分类读取，不能与本类型普通read拼出不存在的权限。
		s.ReadableFeeBusinessTypes = []biz.OrderBusinessType{biz.OrderBusinessLand}
		s.ReadableAbnormalBusinessTypes = []biz.OrderBusinessType{biz.OrderBusinessLand}
		assertWorkbenchTodoCounts(t, f, s, 0, 0)
	}
	other := scope
	other.UserID = uuid.New()
	assertWorkbenchTodoCounts(t, f, other, 0, 0)
}
