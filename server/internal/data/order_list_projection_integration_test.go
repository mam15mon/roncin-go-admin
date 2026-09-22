package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// TestOrderListProjectionNamesPostgres 验证订单列表展示名称由服务端批量解析：
// 客户名与起运（港口）/目的（机场）地点名直接落到投影字段，前端不依赖本地候选
// 缓存补名，杜绝把原始 ID 当名称展示。跳过不算通过，须以
// RONCIN_INTEGRATION_DATABASE_SOURCE 注入专用库并确认 PASS。
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
	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("OLP-CUST-"+suffix).
		SetLegalName("列表投影客户-"+suffix).
		SetNormalizedName("列表投影客户-"+suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建委托单位: %v", err)
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
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订单: %v", err)
	}

	repo := NewOrderRepo(data)
	list, err := repo.List(ctx, []biz.OrderOrganizationScope{
		{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{org.ID}},
	}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessSE})
	if err != nil {
		t.Fatalf("查询订单列表: %v", err)
	}
	var projected *biz.Order
	for _, item := range list.Items {
		if item.ID == order.ID {
			projected = item
			break
		}
	}
	if projected == nil {
		t.Fatalf("列表未返回测试订单 %s", order.ID)
	}
	if projected.CustomerName != partner.LegalName {
		t.Fatalf("委托单位名称投影 = %q, 期望 %q", projected.CustomerName, partner.LegalName)
	}
	if projected.OriginLocationName != "上海港 (CNSHA)" {
		t.Fatalf("起运地点名称投影 = %q", projected.OriginLocationName)
	}
	if projected.DestinationLocationName != "洛杉矶机场 (LAX)" {
		t.Fatalf("目的地点名称投影 = %q", projected.DestinationLocationName)
	}
}
