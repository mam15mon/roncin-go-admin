package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

func TestShippingLineChineseSyncPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const source = "SMDG_LCL"
	old := client.ShippingLine.Create().SetScacCode("MSK").SetNameZh("Maersk").SetNameEn("Maersk").SetCountryCode("DK").SetSource(source).SaveX(ctx)
	organization := client.Organization.Create().SetCode("LINE-SYNC").SetName("船公司同步验证公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	customer := client.Partner.Create().SetOrganizationID(organization.ID).SetCode("LINE-SYNC").SetLegalName("同步验证客户").SetNormalizedName("同步验证客户").SaveX(ctx)
	order := client.Order.Create().SetOrganizationID(organization.ID).SetCustomerID(customer.ID).SetOrderNo("LINE-SYNC").SetIdempotencyKey(uuid.NewString()).SetBusinessType("SE").SetTradeDirection("export").SetPaymentTerm("PREPAID").SetShippingLineID(old.ID).SaveX(ctx)
	manual := client.ShippingLine.Create().SetScacCode("MAN").SetNameZh("手工船司").SetNameEn("Manual").SetCountryCode("CN").SetSource("manual").SaveX(ctx)
	rows := []ShippingLineSyncRecord{
		{SCACCode: "MSK", NameEN: "Maersk", NameZH: "马士基航运", CountryCode: "DK", ContainerPrefixes: []string{"MSKU"}, Enabled: true},
		{SCACCode: "DJS", NameEN: "Dongjin Shipping", NameZH: "东进商船", CountryCode: "KR", Enabled: true},
		{SCACCode: "G2O", NameEN: "G2 Ocean AS", NameZH: "古德航运", CountryCode: "NO", Enabled: true},
		{SCACCode: "TI2", NameEN: "Ti2 Container lines pte ltd", NameZH: "TI2集装箱航运", CountryCode: "SG", Enabled: true},
	}
	store := NewIndustryReferenceSyncStore(d)
	for iteration := 0; iteration < 2; iteration++ {
		result, err := store.ApplyShippingLines(ctx, source, rows)
		if err != nil || result.Created != 3*(1-iteration) || result.Updated != 1+3*iteration || result.Disabled != 0 {
			t.Fatalf("第 %d 次同步统计异常：%+v err=%v", iteration+1, result, err)
		}
		updated := client.ShippingLine.GetX(ctx, old.ID)
		if updated.NameZh != "马士基航运" || updated.SearchKeywords != searchtext.Build("马士基航运", "Maersk") || updated.SearchKeywords == old.SearchKeywords {
			t.Fatalf("旧英文占位必须更新中文及非空检索键：%+v", updated)
		}
		if saved := client.Order.GetX(ctx, order.ID); client.ShippingLine.Query().CountX(ctx) != 5 || saved.ShippingLineID == nil || *saved.ShippingLineID != old.ID {
			t.Fatal("重复同步必须保持船公司 ID、订单引用和目录规模")
		}
		if prefixes := updated.QueryContainerPrefixes().AllX(ctx); len(prefixes) != 1 || prefixes[0].Prefix != "MSKU" {
			t.Fatal("重复同步必须保持箱号前缀")
		}
		if item := client.ShippingLine.GetX(ctx, manual.ID); item.NameZh != "手工船司" || !item.Enabled {
			t.Fatal("同来源同步不得修改手工船司")
		}
	}
	repo := NewIndustryReferenceRepo(d)
	for _, keyword := range []string{"MSK", "Maersk", "马士基", "mashiji", "msjhy"} {
		list, err := repo.ListShippingLines(ctx, uuid.Nil, biz.IndustryReferenceListOptions{Page: 1, PageSize: 20, Keyword: keyword})
		if err != nil || list.Total != 1 || list.Items[0].ID != old.ID {
			t.Fatalf("%q 必须命中补名后的马士基：%+v err=%v", keyword, list, err)
		}
	}
	if row := client.ShippingLine.Query().Where(shippingline.ScacCodeEQ("DJS")).OnlyX(ctx); row.NameZh != "东进商船" || row.SearchKeywords != searchtext.Build("东进商船", "Dongjin Shipping") {
		t.Fatal("新增船司必须同时具备中文及拼音检索键")
	}
	conflicting := append(append([]ShippingLineSyncRecord(nil), rows...), ShippingLineSyncRecord{SCACCode: "MAN", NameEN: "Manual", NameZH: "不应覆盖", CountryCode: "CN", Enabled: true})
	if _, err := store.ApplyShippingLines(ctx, source, conflicting); err == nil {
		t.Fatal("其他来源占用同码必须拒绝")
	}
	for _, code := range []string{"G@O", "A2", "TI22"} {
		_, err := client.ShippingLine.Create().SetScacCode(code).SetNameZh("非法代码测试").SetNameEn("Invalid").SetCountryCode("CN").Save(ctx)
		if !ent.IsConstraintError(err) {
			t.Fatalf("数据库必须拒绝非法船司代码 %s：%v", code, err)
		}
	}
}

func TestShippingLineCodeSchemaMetadata(t *testing.T) {
	const want = "scac_code ~ '^([A-Z]{2,4}|[A-Z0-9]{3})$'"
	if got := migrate.ShippingLinesTable.Annotation.Checks["shipping_lines_scac_code_check"]; got != want {
		t.Fatalf("Ent 生成的船司代码约束必须与正式迁移一致：%s", got)
	}
}
