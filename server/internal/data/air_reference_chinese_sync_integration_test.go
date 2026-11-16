package data

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

// TestAirReferenceChineseSyncPostgres 验证已有英文目录补中文、检索、重复同步与原子回滚。
func TestAirReferenceChineseSyncPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const airportSource, airlineSource = "OURAIRPORTS", "OPENFLIGHTS"
	oldAirport := client.Airport.Create().SetIataCode("PVG").SetIcaoCode("ZSPD").SetNameEn("Shanghai Pudong International Airport").SetCityNameZh("上海市").SetCityNameEn("Shanghai").SetCountryCode("CN").SetSource(airportSource).SaveX(ctx)
	oldAirline := client.Airline.Create().SetIataCode("CA").SetIcaoCode("CCA").SetAwbPrefix("999").SetNameEn("Air China").SetCountryCode("CN").SetSource(airlineSource).SaveX(ctx)
	retiredAirport := client.Airport.Create().SetIataCode("QQQ").SetNameEn("Retired Airport").SetCountryCode("CN").SetSource(airportSource).SaveX(ctx)
	retiredAirline := client.Airline.Create().SetIataCode("QQ").SetNameEn("Retired Airline").SetCountryCode("CN").SetSource(airlineSource).SaveX(ctx)
	manualAirport := client.Airport.Create().SetIataCode("MNL").SetNameZh("手工机场").SetNameEn("Manual Airport").SetCountryCode("PH").SetSource("manual").SaveX(ctx)
	manualAirline := client.Airline.Create().SetIataCode("ZZ").SetNameZh("手工航司").SetNameEn("Manual Airline").SetCountryCode("CN").SetSource("manual").SaveX(ctx)
	organization := client.Organization.Create().SetCode("AIR-ZH").SetName("航空中文同步验证公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	customer := client.Partner.Create().SetOrganizationID(organization.ID).SetCode("AIR-ZH").SetLegalName("航空同步验证客户").SetNormalizedName("航空同步验证客户").SaveX(ctx)
	order := client.Order.Create().SetOrganizationID(organization.ID).SetCustomerID(customer.ID).SetOrderNo("AIR-ZH").SetIdempotencyKey(uuid.NewString()).SetBusinessType("AE").SetTradeDirection("export").SetPaymentTerm("PREPAID").SetOriginLocationID(oldAirport.ID).SaveX(ctx)
	airports := []AirportSyncRecord{
		{IATACode: "PVG", ICAOCode: optionalString("ZSPD"), NameZH: "上海浦东国际机场", NameEN: "Shanghai Pudong International Airport", CityNameEN: optionalString("Shanghai"), CountryCode: "CN", Enabled: true},
		{IATACode: "LAX", ICAOCode: optionalString("KLAX"), NameZH: "洛杉矶国际机场", NameEN: "Los Angeles International Airport", CountryCode: "US", Enabled: true},
	}
	airlines := []AirlineSyncRecord{
		{IATACode: "CA", ICAOCode: optionalString("CCA"), AWBPrefix: optionalString("999"), NameZH: optionalString("中国国际航空"), NameEN: "Air China", CountryCode: "CN", Enabled: true},
		{IATACode: "AA", ICAOCode: optionalString("AAL"), AWBPrefix: optionalString("001"), NameZH: optionalString("美国航空"), NameEN: "American Airlines", CountryCode: "US", Enabled: true},
	}
	store := NewIndustryReferenceSyncStore(d)
	for iteration := 0; iteration < 2; iteration++ {
		airportResult, err := store.ApplyAirports(ctx, airportSource, "zh-test", "airport-combined-hash", airports)
		if err != nil || airportResult.Created != 1-iteration || airportResult.Updated != 1+iteration || airportResult.Disabled != 1 {
			t.Fatalf("第 %d 次机场同步统计异常：%+v err=%v", iteration+1, airportResult, err)
		}
		airlineResult, err := store.ApplyAirlines(ctx, airlineSource, "zh-test", "airline-combined-hash", airlines)
		if err != nil || airlineResult.Created != 1-iteration || airlineResult.Updated != 1+iteration || airlineResult.Disabled != 1 {
			t.Fatalf("第 %d 次航司同步统计异常：%+v err=%v", iteration+1, airlineResult, err)
		}
		pvg := client.Airport.GetX(ctx, oldAirport.ID)
		ca := client.Airline.GetX(ctx, oldAirline.ID)
		if pvg.NameZh == nil || *pvg.NameZh != airports[0].NameZH || pvg.CityNameZh == nil || *pvg.CityNameZh != "上海市" || pvg.SearchKeywords != searchtext.Build(airports[0].NameZH, airports[0].NameEN, "上海市", "Shanghai") || pvg.SearchKeywords == oldAirport.SearchKeywords {
			t.Fatalf("已有机场必须补中文并刷新非空检索键：%+v", pvg)
		}
		if ca.NameZh == nil || *ca.NameZh != *airlines[0].NameZH || ca.SearchKeywords != searchtext.Build(*airlines[0].NameZH, airlines[0].NameEN) || ca.SearchKeywords == oldAirline.SearchKeywords {
			t.Fatalf("已有航司必须补中文并刷新非空检索键：%+v", ca)
		}
		if pvg.SourceHash == nil || *pvg.SourceHash != "airport-combined-hash" || ca.SourceHash == nil || *ca.SourceHash != "airline-combined-hash" {
			t.Fatal("补名必须记录各自的联合源文件摘要")
		}
		if client.Airport.Query().CountX(ctx) != 4 || client.Airline.Query().CountX(ctx) != 4 || client.Airport.Query().Where(airport.IataCodeEQ("PVG")).OnlyX(ctx).ID != oldAirport.ID || client.Airline.Query().Where(airline.IataCodeEQ("CA")).OnlyX(ctx).ID != oldAirline.ID {
			t.Fatal("重复同步必须保持目录规模和原有 ID")
		}
		if saved := client.Order.GetX(ctx, order.ID); saved.OriginLocationID == nil || *saved.OriginLocationID != oldAirport.ID {
			t.Fatal("补名不得改变已有订单的机场引用")
		}
		if client.Airport.GetX(ctx, retiredAirport.ID).Enabled || client.Airline.GetX(ctx, retiredAirline.ID).Enabled || !client.Airport.GetX(ctx, manualAirport.ID).Enabled || !client.Airline.GetX(ctx, manualAirline.ID).Enabled {
			t.Fatal("停用范围必须限于当前来源，不得影响手工目录")
		}
	}

	t.Run("代码中英文全拼和首字母均可检索", func(t *testing.T) {
		repo := NewIndustryReferenceRepo(d)
		for _, keyword := range []string{"PVG", "ZSPD", "Shanghai Pudong", "浦东", "shanghaipudongguojijichang", "shpdgjjc"} {
			result, err := repo.ListAirports(ctx, organization.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Keyword: keyword})
			if err != nil || result.Total != 1 || result.Items[0].ID != oldAirport.ID {
				t.Fatalf("机场关键字 %q 未命中补名后的同一条记录：%+v err=%v", keyword, result, err)
			}
		}
		for _, keyword := range []string{"CA", "CCA", "999", "Air China", "中国国际", "zhongguoguojihangkong", "zggjhk"} {
			result, err := repo.ListAirlines(ctx, organization.ID, biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Keyword: keyword})
			if err != nil || len(result.Items) == 0 || result.Items[0].ID != oldAirline.ID {
				t.Fatalf("航司关键字 %q 未优先命中补名后的国航：%+v err=%v", keyword, result, err)
			}
		}
		lax := client.Airport.Query().Where(airport.IataCodeEQ("LAX")).OnlyX(ctx)
		aa := client.Airline.Query().Where(airline.IataCodeEQ("AA")).OnlyX(ctx)
		if lax.NameZh == nil || *lax.NameZh != "洛杉矶国际机场" || aa.NameZh == nil || *aa.NameZh != "美国航空" {
			t.Fatal("新增机场与航司也必须写入中文")
		}
	})

	t.Run("其他来源冲突保持原有内容", func(t *testing.T) {
		conflictingAirports := append(append([]AirportSyncRecord(nil), airports...), AirportSyncRecord{IATACode: "MNL", NameZH: "不应覆盖", NameEN: "Manual Airport", CountryCode: "PH", Enabled: true})
		conflictingAirlines := append(append([]AirlineSyncRecord(nil), airlines...), AirlineSyncRecord{IATACode: "ZZ", NameZH: optionalString("不应覆盖"), NameEN: "Manual Airline", CountryCode: "CN", Enabled: true})
		if _, err := store.ApplyAirports(ctx, airportSource, "bad", "bad", conflictingAirports); err == nil {
			t.Fatal("机场其他来源同码必须拒绝")
		}
		if _, err := store.ApplyAirlines(ctx, airlineSource, "bad", "bad", conflictingAirlines); err == nil {
			t.Fatal("航司其他来源同码必须拒绝")
		}
		if pvg := client.Airport.GetX(ctx, oldAirport.ID); pvg.SourceHash == nil || *pvg.SourceHash != "airport-combined-hash" || !pvg.Enabled {
			t.Fatal("来源冲突不得影响已同步机场")
		}
		if ca := client.Airline.GetX(ctx, oldAirline.ID); ca.SourceHash == nil || *ca.SourceHash != "airline-combined-hash" || !ca.Enabled {
			t.Fatal("来源冲突不得影响已同步航司")
		}
	})

	t.Run("中途失败回滚中文和停用状态", func(t *testing.T) {
		brokenAirports := append([]AirportSyncRecord(nil), airports...)
		brokenAirports[0].NameZH = "不应保留"
		brokenAirports = append(brokenAirports, AirportSyncRecord{IATACode: "BAD", NameZH: "错误机场", NameEN: strings.Repeat("x", 201), CountryCode: "CN", Enabled: true})
		brokenAirlines := append([]AirlineSyncRecord(nil), airlines...)
		brokenAirlines[0].NameZH = optionalString("不应保留")
		brokenAirlines = append(brokenAirlines, AirlineSyncRecord{IATACode: "XX", NameZH: optionalString("错误航司"), NameEN: strings.Repeat("x", 201), CountryCode: "CN", Enabled: true})
		if _, err := store.ApplyAirports(ctx, airportSource, "bad", "bad", brokenAirports); err == nil {
			t.Fatal("机场字段长度错误必须整批失败")
		}
		if _, err := store.ApplyAirlines(ctx, airlineSource, "bad", "bad", brokenAirlines); err == nil {
			t.Fatal("航司字段长度错误必须整批失败")
		}
		if pvg := client.Airport.GetX(ctx, oldAirport.ID); pvg.NameZh == nil || *pvg.NameZh != airports[0].NameZH || pvg.SourceHash == nil || *pvg.SourceHash != "airport-combined-hash" || !pvg.Enabled {
			t.Fatal("机场错误必须回滚此前的中文更新与停用")
		}
		if ca := client.Airline.GetX(ctx, oldAirline.ID); ca.NameZh == nil || *ca.NameZh != *airlines[0].NameZH || ca.SourceHash == nil || *ca.SourceHash != "airline-combined-hash" || !ca.Enabled {
			t.Fatal("航司错误必须回滚此前的中文更新与停用")
		}
		if client.Airport.Query().CountX(ctx) != 4 || client.Airline.Query().CountX(ctx) != 4 {
			t.Fatal("失败同步不得保留部分新增目录")
		}
	})
}
