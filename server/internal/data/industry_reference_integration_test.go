package data

// 集成测试覆盖行业主数据「总部共享 + 本组织扩展」：分公司查询可见总部行，
// 同一业务代码双行并存时本组织行优先（去重下推 SQL，分页计数一致），
// 总部自身查询不重复，is_active 过滤与 keyword 检索行为保持不变。

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestIndustryReferenceHeadquartersSharingPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]

	headquarters := data.db.Organization.Create().
		SetCode("HQ-IND-" + suffix).
		SetName("行业主数据测试总部").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		SaveX(ctx)
	branch := data.db.Organization.Create().
		SetCode("BR-IND-" + suffix).
		SetName("行业主数据测试分公司").
		SetKind("company").
		SetBaseCurrency("CNY").
		SetParentID(headquarters.ID).
		SaveX(ctx)

	// 总部港口：CNSHA 启用、CNTAO 停用；分公司同码 CNSHA 私有行 + 独有 CNNGB。
	data.db.Port.Create().SetOrganizationID(headquarters.ID).SetUnLocode("CNSHA").SetNameZh("上海港").SetNameEn("Shanghai").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)
	data.db.Port.Create().SetOrganizationID(headquarters.ID).SetUnLocode("CNTAO").SetNameZh("青岛港").SetNameEn("Qingdao").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(false).SaveX(ctx)
	data.db.Port.Create().SetOrganizationID(branch.ID).SetUnLocode("CNSHA").SetNameZh("上海港(分公司)").SetNameEn("Shanghai Branch").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)
	data.db.Port.Create().SetOrganizationID(branch.ID).SetUnLocode("CNNGB").SetNameZh("宁波港").SetNameEn("Ningbo").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)

	// 总部机场与航司；分公司同码航司私有行验证 IATA 去重。
	data.db.Airport.Create().SetOrganizationID(headquarters.ID).SetIataCode("PVG").SetNameZh("上海浦东").SetNameEn("Shanghai Pudong").SetCityNameZh("上海").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
	data.db.Airline.Create().SetOrganizationID(headquarters.ID).SetIataCode("MU").SetNameZh("中国东方航空").SetNameEn("China Eastern").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
	data.db.Airline.Create().SetOrganizationID(branch.ID).SetIataCode("MU").SetNameZh("东航(分公司)").SetNameEn("China Eastern Branch").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
	data.db.ShippingLine.Create().SetOrganizationID(headquarters.ID).SetScacCode("MSK").SetNameZh("马士基").SetNameEn("Maersk").SetCountryCode("DK").SetEnabled(true).SaveX(ctx)

	uc := biz.NewIndustryReferenceUsecase(NewIndustryReferenceRepo(data))
	enabled := true

	t.Run("分公司查询总部共享并同码去重本组织优先", func(t *testing.T) {
		result, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("分公司查询港口失败: %v", err)
		}
		if result.Total != 3 || len(result.Items) != 3 {
			t.Fatalf("分公司应可见 3 个港口（总部共享 + 私有，同码去重）: total=%d items=%d", result.Total, len(result.Items))
		}
		byCode := map[string]*biz.Port{}
		for _, item := range result.Items {
			byCode[item.UNLocode] = item
		}
		if item, ok := byCode["CNSHA"]; !ok || item.OrganizationID != branch.ID || item.NameZH != "上海港(分公司)" {
			t.Fatalf("同码 CNSHA 应保留本组织行: %+v", item)
		}
		if _, ok := byCode["CNTAO"]; !ok {
			t.Fatal("总部共享港口 CNTAO 应对分公司可见")
		}
		if _, ok := byCode["CNNGB"]; !ok {
			t.Fatal("分公司私有港口 CNNGB 应可见")
		}
	})

	t.Run("启用过滤与关键字检索在共享查询下保持行为", func(t *testing.T) {
		result, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Enabled: &enabled})
		if err != nil {
			t.Fatalf("分公司按启用过滤查询港口失败: %v", err)
		}
		if result.Total != 2 {
			t.Fatalf("启用过滤后应仅剩 2 个港口: total=%d", result.Total)
		}
		keyword := "宁波"
		keywordResult, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Keyword: keyword})
		if err != nil {
			t.Fatalf("分公司关键字查询港口失败: %v", err)
		}
		if keywordResult.Total != 1 || keywordResult.Items[0].UNLocode != "CNNGB" {
			t.Fatalf("关键字“宁波”应命中分公司私有港口: total=%d", keywordResult.Total)
		}
	})

	t.Run("总部自身查询不包含分公司私有行且不重复", func(t *testing.T) {
		result, err := uc.ListPorts(ctx, headquarters.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("总部查询港口失败: %v", err)
		}
		if result.Total != 2 {
			t.Fatalf("总部应仅可见自身 2 个港口: total=%d", result.Total)
		}
	})

	t.Run("机场航司船公司共享可见且航司同码本组织优先", func(t *testing.T) {
		airports, err := uc.ListAirports(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("分公司查询机场失败: %v", err)
		}
		if airports.Total != 1 || airports.Items[0].IATACode != "PVG" || airports.Items[0].OrganizationID != headquarters.ID {
			t.Fatalf("总部机场应对分公司可见: %+v", airports.Items)
		}
		airlines, err := uc.ListAirlines(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("分公司查询航司失败: %v", err)
		}
		if airlines.Total != 1 || airlines.Items[0].OrganizationID != branch.ID {
			t.Fatalf("同码航司应保留本组织行: %+v", airlines.Items)
		}
		shippingLines, err := uc.ListShippingLines(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("分公司查询船公司失败: %v", err)
		}
		if shippingLines.Total != 1 || shippingLines.Items[0].SCACCode != "MSK" || shippingLines.Items[0].OrganizationID != headquarters.ID {
			t.Fatalf("总部船公司应对分公司可见: %+v", shippingLines.Items)
		}
	})
}
