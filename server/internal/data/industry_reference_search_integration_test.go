package data

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestIndustryReferenceCodeRelevancePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	repo := NewIndustryReferenceRepo(d)
	organizationID := uuid.New()
	enabled, disabled := true, false

	// 精确代码项故意排到维护顺序末尾，模糊名称项排在最前，验证相关性在分页前生效。
	d.db.Airline.Create().SetIataCode("CA").SetIcaoCode("CCA").SetAwbPrefix("999").SetNameZh("上海航空").SetNameEn("Air China C_ C%").SetCountryCode("CN").SetSortOrder(999).SetEnabled(true).SaveX(ctx)
	d.db.Airline.Create().SetIataCode("CX").SetIcaoCode("CAX").SetAwbPrefix("998").SetNameZh("前缀航空").SetNameEn("Prefix").SetCountryCode("CN").SetSortOrder(1).SetEnabled(true).SaveX(ctx)
	d.db.Airline.Create().SetIataCode("AA").SetNameZh("模糊航空").SetNameEn("American Airlines CCA 999 C_ C%").SetCountryCode("US").SetSortOrder(0).SetEnabled(true).SaveX(ctx)
	d.db.Airline.Create().SetIataCode("AC").SetIcaoCode("ACA").SetNameZh("加拿大航空").SetNameEn("Air Canada").SetCountryCode("CA").SetSortOrder(0).SetEnabled(true).SaveX(ctx)
	d.db.Airline.Create().SetIataCode("CB").SetNameZh("停用航空").SetNameEn("CA disabled").SetCountryCode("CN").SetSortOrder(0).SetEnabled(false).SaveX(ctx)

	d.db.Airport.Create().SetIataCode("PVG").SetIcaoCode("ZSPD").SetNameZh("上海机场").SetNameEn("Shanghai Pudong").SetCountryCode("CN").SetSortOrder(999).SetEnabled(true).SaveX(ctx)
	d.db.Airport.Create().SetIataCode("PVA").SetIcaoCode("PVGA").SetNameZh("前缀机场").SetNameEn("Prefix").SetCountryCode("CN").SetSortOrder(1).SetEnabled(true).SaveX(ctx)
	d.db.Airport.Create().SetIataCode("AAA").SetNameZh("模糊机场").SetNameEn("PVG International ZSPD % _").SetCountryCode("CN").SetSortOrder(0).SetEnabled(true).SaveX(ctx)
	d.db.Airport.Create().SetIataCode("PVB").SetNameZh("停用机场").SetNameEn("PVG disabled").SetCountryCode("CN").SetSortOrder(0).SetEnabled(false).SaveX(ctx)

	d.db.Port.Create().SetUnLocode("CNSHA").SetNameZh("上海港").SetNameEn("Shanghai").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetSortOrder(999).SetEnabled(true).SaveX(ctx)
	d.db.Port.Create().SetUnLocode("CNSHB").SetNameZh("前缀港").SetNameEn("Prefix").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetSortOrder(1).SetEnabled(true).SaveX(ctx)
	d.db.Port.Create().SetUnLocode("CNAAB").SetNameZh("模糊港").SetNameEn("CNSHA International % _").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetSortOrder(0).SetEnabled(true).SaveX(ctx)
	d.db.Port.Create().SetUnLocode("CNSHC").SetNameZh("停用港").SetNameEn("CNSHA disabled").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetSortOrder(0).SetEnabled(false).SaveX(ctx)

	d.db.ShippingLine.Create().SetScacCode("MAE").SetNameZh("上海船司").SetNameEn("Shanghai").SetCountryCode("CN").SetSortOrder(999).SetEnabled(true).SaveX(ctx)
	d.db.ShippingLine.Create().SetScacCode("MAEU").SetNameZh("前缀船司").SetNameEn("Prefix").SetCountryCode("CN").SetSortOrder(1).SetEnabled(true).SaveX(ctx)
	d.db.ShippingLine.Create().SetScacCode("AAA").SetNameZh("模糊船司").SetNameEn("MAE Shipping % _").SetCountryCode("CN").SetSortOrder(0).SetEnabled(true).SaveX(ctx)
	d.db.ShippingLine.Create().SetScacCode("MAEB").SetNameZh("停用船司").SetNameEn("MAE disabled").SetCountryCode("CN").SetSortOrder(0).SetEnabled(false).SaveX(ctx)

	for _, directory := range []struct {
		name          string
		keyword       string
		prefix        string
		initials      string
		exactCode     string
		fuzzyCode     string
		disabledCode  string
		wantExact     []string
		wantPrefix    []string
		wantUnqueried []string
		alternate     []string
		literalCodes  bool
		list          func(biz.IndustryReferenceListOptions) ([]string, int, error)
	}{
		{
			name: "航空公司", keyword: "cA", prefix: "c", initials: "shhk", exactCode: "CA", fuzzyCode: "AA", disabledCode: "CB",
			wantExact: []string{"CA", "CX", "AA", "AC"}, wantPrefix: []string{"CX", "CA", "AA", "AC"}, wantUnqueried: []string{"AA", "AC", "CX", "CA"}, alternate: []string{"cca", "999"}, literalCodes: true,
			list: func(options biz.IndustryReferenceListOptions) ([]string, int, error) {
				result, err := repo.ListAirlines(ctx, organizationID, options)
				if err != nil {
					return nil, 0, err
				}
				codes := make([]string, 0, len(result.Items))
				for _, item := range result.Items {
					codes = append(codes, item.IATACode)
				}
				return codes, result.Total, nil
			},
		},
		{
			name: "机场", keyword: "pVg", prefix: "pv", initials: "shjc", exactCode: "PVG", fuzzyCode: "AAA", disabledCode: "PVB",
			wantExact: []string{"PVG", "PVA", "AAA"}, wantPrefix: []string{"PVA", "PVG", "AAA"}, wantUnqueried: []string{"AAA", "PVA", "PVG"}, alternate: []string{"zspd"},
			list: func(options biz.IndustryReferenceListOptions) ([]string, int, error) {
				result, err := repo.ListAirports(ctx, organizationID, options)
				if err != nil {
					return nil, 0, err
				}
				codes := make([]string, 0, len(result.Items))
				for _, item := range result.Items {
					codes = append(codes, item.IATACode)
				}
				return codes, result.Total, nil
			},
		},
		{
			name: "港口", keyword: "cNsHa", prefix: "cnsh", initials: "shg", exactCode: "CNSHA", fuzzyCode: "CNAAB", disabledCode: "CNSHC",
			wantExact: []string{"CNSHA", "CNAAB"}, wantPrefix: []string{"CNSHB", "CNSHA", "CNAAB"}, wantUnqueried: []string{"CNAAB", "CNSHB", "CNSHA"},
			list: func(options biz.IndustryReferenceListOptions) ([]string, int, error) {
				result, err := repo.ListPorts(ctx, organizationID, options)
				if err != nil {
					return nil, 0, err
				}
				codes := make([]string, 0, len(result.Items))
				for _, item := range result.Items {
					codes = append(codes, item.UNLocode)
				}
				return codes, result.Total, nil
			},
		},
		{
			name: "船公司", keyword: "mAe", prefix: "ma", initials: "shcs", exactCode: "MAE", fuzzyCode: "AAA", disabledCode: "MAEB",
			wantExact: []string{"MAE", "MAEU", "AAA"}, wantPrefix: []string{"MAEU", "MAE", "AAA"}, wantUnqueried: []string{"AAA", "MAEU", "MAE"},
			list: func(options biz.IndustryReferenceListOptions) ([]string, int, error) {
				result, err := repo.ListShippingLines(ctx, organizationID, options)
				if err != nil {
					return nil, 0, err
				}
				codes := make([]string, 0, len(result.Items))
				for _, item := range result.Items {
					codes = append(codes, item.SCACCode)
				}
				return codes, result.Total, nil
			},
		},
	} {
		t.Run(directory.name, func(t *testing.T) {
			assertList := func(options biz.IndustryReferenceListOptions, want []string, wantTotal int) {
				t.Helper()
				codes, total, err := directory.list(options)
				if err != nil {
					t.Fatalf("查询 %+v 失败: %v", options, err)
				}
				if !reflect.DeepEqual(codes, want) || total != wantTotal {
					t.Fatalf("查询 %+v: codes=%v total=%d，期望 codes=%v total=%d", options, codes, total, want, wantTotal)
				}
			}
			options := biz.IndustryReferenceListOptions{Page: 1, PageSize: 50, Enabled: &enabled, Keyword: directory.keyword}
			assertList(options, directory.wantExact, len(directory.wantExact))
			options.PageSize = 1
			assertList(options, directory.wantExact[:1], len(directory.wantExact))
			options.Page = 2
			assertList(options, directory.wantExact[1:2], len(directory.wantExact))
			options.Page, options.PageSize = 1, 50
			options.Keyword = directory.prefix
			assertList(options, directory.wantPrefix, len(directory.wantPrefix))
			options.Keyword = ""
			assertList(options, directory.wantUnqueried, len(directory.wantUnqueried))
			for _, keyword := range append([]string{"上海", "shanghai", directory.initials}, directory.alternate...) {
				options.Keyword = keyword
				want := []string{directory.exactCode}
				if keyword != "上海" && keyword != "shanghai" && keyword != directory.initials {
					want = append(want, directory.fuzzyCode)
				}
				assertList(options, want, len(want))
			}
			for _, keyword := range []string{"%", "_"} {
				options.Keyword = keyword
				want := []string{directory.fuzzyCode}
				if directory.literalCodes {
					want = append(want, directory.exactCode)
				}
				assertList(options, want, len(want))
			}
			if directory.literalCodes {
				// 两条名称都包含文字 C_ / C%；不能把它们当作代码 CA 的通配匹配提升。
				for _, keyword := range []string{"c_", "c%"} {
					options.Keyword = keyword
					assertList(options, []string{directory.fuzzyCode, directory.exactCode}, 2)
				}
			}
			options.Keyword = "' OR 1=1 --"
			assertList(options, []string{}, 0)
			options.Keyword, options.Enabled = directory.keyword, &disabled
			assertList(options, []string{directory.disabledCode}, 1)
		})
	}
}
