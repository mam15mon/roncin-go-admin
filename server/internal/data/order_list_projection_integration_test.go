package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

// TestOrderListProjectionNamesPostgres 验证订单用户可见展示名全部由服务端解析：
// 客户、代理（订舱/船代）、船公司、起运（港口）/目的（机场）地点、操作/业务/创建人，
// 列表与详情两条路径一致；前端不得依赖本地候选缓存补名，杜绝把原始 ID 当名称展示。
// 跳过不算通过，须以 RONCIN_INTEGRATION_DATABASE_SOURCE 注入专用库并确认 PASS。
func TestOrderListProjectionNamesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	suffix := uuid.NewString()[:8]
	org, err := data.db.Organization.Create().
		SetCode("OLP-"+suffix).
		SetName("列表投影测试组织-"+suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组织: %v", err)
	}
	newUser := func(name string) (id uuid.UUID) {
		t.Helper()
		account, userErr := data.db.User.Create().
			SetDisplayName(name).
			SetDingtalkUnionid(uuid.NewString()).
			SetDingtalkUserid(uuid.NewString()).
			SetDingtalkName(name).
			SetEnabled(true).
			Save(ctx)
		if userErr != nil {
			t.Fatalf("创建员工 %s: %v", name, userErr)
		}
		return account.ID
	}
	operatorID := newUser("投影操作员-" + suffix)
	salesID := newUser("投影业务员-" + suffix)
	creatorID := newUser("投影创建人-" + suffix)
	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("OLP-CUST-"+suffix).
		SetLegalName("列表投影客户-"+suffix).
		SetNormalizedName("列表投影客户-"+suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建委托单位: %v", err)
	}
	bookingAgent, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("OLP-AGENT-"+suffix).
		SetLegalName("投影订舱代理-"+suffix).
		SetNormalizedName("投影订舱代理-"+suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订舱代理: %v", err)
	}
	originPort := data.db.Port.Create().
		SetUnLocode("CNSHA").
		SetNameZh("上海港").
		SetNameEn("Shanghai").
		SetCountryCode("CN").
		SetTransportModes([]string{"SEA"}).
		SetEnabled(true).
		SaveX(ctx)
	destAirport := data.db.Airport.Create().
		SetIataCode("LAX").
		SetNameZh("洛杉矶机场").
		SetNameEn("Los Angeles").
		SetCityNameZh("洛杉矶").
		SetCountryCode("US").
		SetEnabled(true).
		SaveX(ctx)
	shippingLine := data.db.ShippingLine.Create().
		SetScacCode("OLP").
		SetNameZh("投影船公司").
		SetNameEn("Projection Line").
		SetCountryCode("CN").
		SetEnabled(true).
		SaveX(ctx)
	order, err := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(org.ID).
		SetOrderNo("OLP"+suffix).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetOriginLocationID(originPort.ID).
		SetDestinationLocationID(destAirport.ID).
		SetBookingAgentID(bookingAgent.ID).
		SetShippingLineID(shippingLine.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订单: %v", err)
	}
	for _, assignment := range []struct {
		userID uuid.UUID
		role   orderpersonnelent.Role
	}{{operatorID, orderpersonnelent.RoleOPERATOR}, {salesID, orderpersonnelent.RoleSALES}, {creatorID, orderpersonnelent.RoleCREATOR}} {
		if _, err := data.db.OrderPersonnel.Create().
			SetOrderID(order.ID).
			SetUserID(assignment.userID).
			SetOrganizationID(org.ID).
			SetRole(assignment.role).
			Save(ctx); err != nil {
			t.Fatalf("创建订单人员分配 %s: %v", assignment.role, err)
		}
	}

	assertProjection := func(label string, projected *biz.Order) {
		t.Helper()
		if projected.CustomerName != partner.LegalName {
			t.Fatalf("%s 委托单位名称投影 = %q, 期望 %q", label, projected.CustomerName, partner.LegalName)
		}
		if projected.OriginLocationName != "上海港 (CNSHA)" {
			t.Fatalf("%s 起运地点名称投影 = %q", label, projected.OriginLocationName)
		}
		if projected.DestinationLocationName != "洛杉矶机场 (LAX)" {
			t.Fatalf("%s 目的地点名称投影 = %q", label, projected.DestinationLocationName)
		}
		if projected.BookingAgentName != bookingAgent.LegalName {
			t.Fatalf("%s 订舱代理名称投影 = %q", label, projected.BookingAgentName)
		}
		if projected.ShippingLineName != "投影船公司" {
			t.Fatalf("%s 船公司名称投影 = %q", label, projected.ShippingLineName)
		}
		if projected.OperatorName != "投影操作员-"+suffix || projected.OperatorBranch != org.Name {
			t.Fatalf("%s 操作员投影 = %q/%q", label, projected.OperatorName, projected.OperatorBranch)
		}
		if projected.SalesName != "投影业务员-"+suffix || projected.SalesBranch != org.Name {
			t.Fatalf("%s 业务员投影 = %q/%q", label, projected.SalesName, projected.SalesBranch)
		}
		if projected.CreatorName != "投影创建人-"+suffix {
			t.Fatalf("%s 创建人投影 = %q", label, projected.CreatorName)
		}
	}

	repo := NewOrderRepo(data)
	list, err := repo.List(ctx, []biz.OrderOrganizationScope{
		{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{org.ID}},
	}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessSE})
	if err != nil {
		t.Fatalf("查询订单列表: %v", err)
	}
	var listed *biz.Order
	for _, item := range list.Items {
		if item.ID == order.ID {
			listed = item
			break
		}
	}
	if listed == nil {
		t.Fatalf("列表未返回测试订单 %s", order.ID)
	}
	assertProjection("列表", listed)

	detail, err := repo.Get(ctx, org.ID, order.ID)
	if err != nil {
		t.Fatalf("查询订单详情: %v", err)
	}
	assertProjection("详情", detail)
}
