package data

// 集成测试覆盖全局公共目录、分页检索与业务码唯一性。

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestIndustryReferenceGlobalSharingPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]

	branch := data.db.Organization.Create().
		SetCode("BR-IND-" + suffix).
		SetName("行业主数据测试公司").
		SetKind("company").
		SetBaseCurrency("CNY").
		SaveX(ctx)

	// 公共港口：CNSHA、CNNGB 启用，CNTAO 停用。
	data.db.Port.Create().SetUnLocode("CNSHA").SetNameZh("上海港").SetNameEn("Shanghai").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)
	data.db.Port.Create().SetUnLocode("CNTAO").SetNameZh("青岛港").SetNameEn("Qingdao").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(false).SaveX(ctx)
	data.db.Port.Create().SetUnLocode("CNNGB").SetNameZh("宁波港").SetNameEn("Ningbo").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetEnabled(true).SaveX(ctx)

	// 公共机场与全局航司/船司。
	data.db.Airport.Create().SetIataCode("PVG").SetNameZh("上海浦东").SetNameEn("Shanghai Pudong").SetCityNameZh("上海").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
	data.db.Airline.Create().SetIataCode("MU").SetNameZh("中国东方航空").SetNameEn("China Eastern").SetCountryCode("CN").SetEnabled(true).SaveX(ctx)
	data.db.ShippingLine.Create().SetScacCode("MSK").SetNameZh("马士基").SetNameEn("Maersk").SetCountryCode("DK").SetEnabled(true).SaveX(ctx)

	uc := biz.NewIndustryReferenceUsecase(NewIndustryReferenceRepo(data))
	enabled := true

	t.Run("公司查询公共目录", func(t *testing.T) {
		result, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("公司查询港口失败: %v", err)
		}
		if result.Total != 3 || len(result.Items) != 3 {
			t.Fatalf("公司应可见 3 个港口（全局公共目录）: total=%d items=%d", result.Total, len(result.Items))
		}
		byCode := map[string]*biz.Port{}
		for _, item := range result.Items {
			byCode[item.UNLocode] = item
		}
		if item, ok := byCode["CNSHA"]; !ok || item.NameZH != "上海港" {
			t.Fatalf("CNSHA 应返回公共目录行: %+v", item)
		}
		if _, ok := byCode["CNTAO"]; !ok {
			t.Fatal("公共港口 CNTAO 应对公司可见")
		}
		if _, ok := byCode["CNNGB"]; !ok {
			t.Fatal("公共港口 CNNGB 应可见")
		}
	})

	t.Run("公共目录启用过滤与关键字检索", func(t *testing.T) {
		result, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Enabled: &enabled})
		if err != nil {
			t.Fatalf("公司按启用过滤查询港口失败: %v", err)
		}
		if result.Total != 2 {
			t.Fatalf("启用过滤后应仅剩 2 个港口: total=%d", result.Total)
		}
		keyword := "宁波"
		keywordResult, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Keyword: keyword})
		if err != nil {
			t.Fatalf("公司关键字查询港口失败: %v", err)
		}
		if keywordResult.Total != 1 || keywordResult.Items[0].UNLocode != "CNNGB" {
			t.Fatalf("关键字“宁波”应命中公共港口: total=%d", keywordResult.Total)
		}
	})

	t.Run("分页计数与公共目录一致", func(t *testing.T) {
		result, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 2})
		if err != nil {
			t.Fatalf("公司分页查询港口失败: %v", err)
		}
		if result.Total != 3 || len(result.Items) != 2 {
			t.Fatalf("分页尺寸 2 应返回 total=3 items=2: total=%d items=%d", result.Total, len(result.Items))
		}
	})

	t.Run("不同公司读取同一目录", func(t *testing.T) {
		observer := data.db.Organization.Create().
			SetCode("OB-IND-" + suffix).
			SetName("行业主数据观察组织").
			SetKind("company").
			SetBaseCurrency("CNY").
			SaveX(ctx)
		result, err := uc.ListPorts(ctx, observer.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("观察组织查询港口失败: %v", err)
		}
		if result.Total != 3 {
			t.Fatalf("各公司应见相同 3 个公共港口: total=%d", result.Total)
		}
		original, err := uc.ListPorts(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatal(err)
		}
		for i, item := range original.Items {
			if item.ID != result.Items[i].ID {
				t.Fatalf("各公司必须使用相同港口 ID: %s != %s", item.ID, result.Items[i].ID)
			}
		}
		airports, err := uc.ListAirports(ctx, observer.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		originalAirports, originalErr := uc.ListAirports(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil || originalErr != nil || airports.Total != 1 || originalAirports.Total != 1 || airports.Items[0].ID != originalAirports.Items[0].ID {
			t.Fatalf("各公司必须使用相同机场 ID: %v %v", err, originalErr)
		}
	})

	t.Run("机场航司船公司全员可见", func(t *testing.T) {
		airports, err := uc.ListAirports(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("公司查询机场失败: %v", err)
		}
		if airports.Total != 1 || airports.Items[0].IATACode != "PVG" {
			t.Fatalf("公共机场应对公司可见: %+v", airports.Items)
		}
		airlines, err := uc.ListAirlines(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("公司查询航司失败: %v", err)
		}
		if airlines.Total != 1 || airlines.Items[0].IATACode != "MU" {
			t.Fatalf("A 型航司应全局唯一可见: %+v", airlines.Items)
		}
		shippingLines, err := uc.ListShippingLines(ctx, branch.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("公司查询船公司失败: %v", err)
		}
		if shippingLines.Total != 1 || shippingLines.Items[0].SCACCode != "MSK" {
			t.Fatalf("A 型船公司应全局唯一可见: %+v", shippingLines.Items)
		}
	})

	t.Run("A 型全局唯一冲突映射业务错误", func(t *testing.T) {
		repo := NewIndustryReferenceRepo(data)
		_, err := repo.CreatePort(ctx, branch.ID, &biz.Port{UNLocode: "CNSHA", NameEN: "Duplicate", CountryCode: "CN", TransportModes: []string{"SEA"}, Source: "manual"}, &biz.AuditEvent{Details: map[string]string{}})
		if err != biz.ErrIndustryReferenceCodeExist {
			t.Fatalf("重复港口代码应拒绝: %v", err)
		}
		_, err = repo.CreateAirport(ctx, branch.ID, &biz.Airport{IATACode: "PVG", NameEN: "Duplicate", CountryCode: "CN", Source: "manual"}, &biz.AuditEvent{Details: map[string]string{}})
		if err != biz.ErrIndustryReferenceCodeExist {
			t.Fatalf("重复机场代码应拒绝: %v", err)
		}
		_, err = repo.CreateAirline(ctx, branch.ID, &biz.Airline{IATACode: "MU", NameEN: "Duplicate", CountryCode: "CN", Source: "manual"}, &biz.AuditEvent{Details: map[string]string{}})
		if err != biz.ErrIndustryReferenceCodeExist {
			t.Fatalf("重复 IATA 应映射 ErrIndustryReferenceCodeExist，实际 %v", err)
		}
		_, err = repo.CreateShippingLine(ctx, branch.ID, &biz.ShippingLine{SCACCode: "MSK", NameZH: "马士基重复", NameEN: "Duplicate", CountryCode: "DK", Source: "manual"}, &biz.AuditEvent{Details: map[string]string{}})
		if err != biz.ErrIndustryReferenceCodeExist {
			t.Fatalf("重复 SCAC 应映射 ErrIndustryReferenceCodeExist，实际 %v", err)
		}
	})
}
