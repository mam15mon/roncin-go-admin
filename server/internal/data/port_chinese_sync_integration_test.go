package data

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

// TestPortChineseSyncPostgres 使用正式迁移的隔离 Schema 验证同步写入、检索及整批回滚。
func TestPortChineseSyncPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	const source = "UNECE_UNLOCODE"
	old := client.Port.Create().SetUnLocode("CNSHG").SetNameEn("Shanghai").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetSource(source).SetSourceVersion("old").SetSourceHash("old").SaveX(ctx)
	if old.NameZh != nil || old.SearchKeywords == "" {
		t.Fatal("前态必须是中文为空、英文检索键非空的已有行")
	}
	organization := client.Organization.Create().SetCode("PORT-SYNC").SetName("港口同步验证公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	customer := client.Partner.Create().SetOrganizationID(organization.ID).SetCode("PORT-SYNC").SetLegalName("港口同步验证客户").SetNormalizedName("港口同步验证客户").SaveX(ctx)
	order := client.Order.Create().SetOrganizationID(organization.ID).SetCustomerID(customer.ID).SetOrderNo("PORT-SYNC").SetIdempotencyKey(uuid.NewString()).SetBusinessType("SE").SetTradeDirection("export").SetPaymentTerm("PREPAID").SetOriginLocationID(old.ID).SaveX(ctx)
	retired := client.Port.Create().SetUnLocode("CNOLD").SetNameEn("Old Port").SetCountryCode("CN").SetTransportModes([]string{"SEA"}).SetSource(source).SaveX(ctx)
	manual := client.Port.Create().SetUnLocode("USSEA").SetNameZh("西雅图").SetNameEn("Seattle").SetCountryCode("US").SetTransportModes([]string{"SEA"}).SetSource("manual").SaveX(ctx)
	rows := []PortSyncRecord{
		{UNLocode: "CNSHG", NameEN: "Shanghai", NameZH: "上海", CountryCode: "CN", TransportModes: []string{"SEA", "RAIL", "ROAD"}, Enabled: true},
		{UNLocode: "USLAX", NameEN: "Los Angeles", NameZH: "洛杉矶", CountryCode: "US", TransportModes: []string{"SEA"}, Enabled: true},
	}
	store := NewIndustryReferenceSyncStore(d)
	result, err := store.ApplyPorts(ctx, source, "2025-1", "combined-hash", rows)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created != 1 || result.Updated != 1 || result.Disabled != 1 {
		t.Fatalf("首次应新增 1、补齐 1、停用 1：%+v", result)
	}
	shanghai := client.Port.GetX(ctx, old.ID)
	losAngeles := client.Port.Query().Where(port.UnLocodeEQ("USLAX")).OnlyX(ctx)
	if shanghai.NameZh == nil || *shanghai.NameZh != "上海" || shanghai.SearchKeywords != searchtext.Build("上海", "Shanghai") || shanghai.SearchKeywords == old.SearchKeywords {
		t.Fatalf("已有行必须补中文并刷新非空检索键：%+v", shanghai)
	}
	if losAngeles.NameZh == nil || *losAngeles.NameZh != "洛杉矶" || losAngeles.SearchKeywords != searchtext.Build("洛杉矶", "Los Angeles") {
		t.Fatalf("新行必须写入中文及检索键：%+v", losAngeles)
	}
	if shanghai.SourceVersion == nil || *shanghai.SourceVersion != "2025-1" || shanghai.SourceHash == nil || *shanghai.SourceHash != "combined-hash" || client.Port.GetX(ctx, retired.ID).Enabled || !client.Port.GetX(ctx, manual.ID).Enabled {
		t.Fatal("版本、联合指纹或来源范围不正确")
	}

	t.Run("仓储支持中英文标准码和拼音", func(t *testing.T) {
		repo := NewIndustryReferenceRepo(d)
		for _, keyword := range []string{"CNSHG", "Shanghai", "上海", "shanghai"} {
			list, err := repo.ListPorts(ctx, uuid.Nil, biz.IndustryReferenceListOptions{Page: 1, PageSize: 20, Keyword: keyword})
			if err != nil || list.Total != 1 || list.Items[0].ID != old.ID {
				t.Fatalf("%q 必须命中补名后的同一个港口：%+v err=%v", keyword, list, err)
			}
		}
		// 用洛杉矶验证全拼和首字母，避免上海拼音与英文相同造成假阳性。
		for _, keyword := range []string{"luoshanji", "lsj"} {
			list, err := repo.ListPorts(ctx, uuid.Nil, biz.IndustryReferenceListOptions{Page: 1, PageSize: 20, Keyword: keyword})
			if err != nil || list.Total != 1 || list.Items[0].ID != losAngeles.ID {
				t.Fatalf("%q 必须通过中文检索键命中新港口：%+v err=%v", keyword, list, err)
			}
		}
	})

	t.Run("重复同步保持ID与目录规模", func(t *testing.T) {
		result, err := store.ApplyPorts(ctx, source, "2025-1", "combined-hash", rows)
		if err != nil || result.Created != 0 || result.Updated != 2 || result.Disabled != 1 {
			t.Fatalf("重复同步统计异常：%+v err=%v", result, err)
		}
		if client.Port.Query().CountX(ctx) != 4 || client.Port.Query().Where(port.UnLocodeEQ("CNSHG")).OnlyX(ctx).ID != old.ID || client.Port.Query().Where(port.UnLocodeEQ("USLAX")).OnlyX(ctx).ID != losAngeles.ID {
			t.Fatal("重复同步不得重建港口或追加重复项")
		}
		if saved := client.Order.GetX(ctx, order.ID); saved.OriginLocationID == nil || *saved.OriginLocationID != old.ID {
			t.Fatal("同步不得改写既有订单的港口引用")
		}
	})

	t.Run("其他来源占用仍显式拒绝", func(t *testing.T) {
		conflicting := append(append([]PortSyncRecord(nil), rows...), PortSyncRecord{UNLocode: "USSEA", NameEN: "Seattle", NameZH: "西雅图港", CountryCode: "US", TransportModes: []string{"SEA"}, Enabled: true})
		conflicts, err := store.CheckPorts(ctx, source, conflicting)
		if err != nil || len(conflicts) != 1 || conflicts[0].Code != "USSEA" || !strings.Contains(conflicts[0].Message, "manual") {
			t.Fatalf("必须显式报告其他来源冲突：%+v err=%v", conflicts, err)
		}
		if _, err := store.ApplyPorts(ctx, source, "bad", "bad", conflicting); err == nil {
			t.Fatal("其他来源占用必须阻止写入")
		}
		if item := client.Port.GetX(ctx, manual.ID); item.Source != "manual" || item.NameZh == nil || *item.NameZh != "西雅图" || !item.Enabled {
			t.Fatalf("来源冲突不得改变手工港口：%+v", item)
		}
		if item := client.Port.GetX(ctx, old.ID); item.SourceHash == nil || *item.SourceHash != "combined-hash" || !item.Enabled {
			t.Fatal("来源冲突不得改变已同步港口")
		}
	})

	t.Run("中途写入错误回滚整批更新新增和停用", func(t *testing.T) {
		broken := []PortSyncRecord{
			{UNLocode: "CNSHG", NameEN: "Changed", NameZH: "不应保留", CountryCode: "CN", TransportModes: []string{"SEA"}, Enabled: true},
			{UNLocode: "JPTYO", NameEN: "Tokyo", NameZH: "东京", CountryCode: "JP", TransportModes: []string{"SEA"}, Enabled: true},
			{UNLocode: "KRBAD", NameEN: strings.Repeat("x", 201), NameZH: "错误港口", CountryCode: "KR", TransportModes: []string{"SEA"}, Enabled: true},
		}
		if _, err := store.ApplyPorts(ctx, source, "bad", "bad", broken); err == nil {
			t.Fatal("中途违反字段长度约束必须失败")
		}
		if item := client.Port.GetX(ctx, old.ID); item.NameZh == nil || *item.NameZh != "上海" || item.NameEn != "Shanghai" || item.SourceHash == nil || *item.SourceHash != "combined-hash" || item.SearchKeywords != shanghai.SearchKeywords {
			t.Fatalf("错误必须回滚此前的字段与检索键修改：%+v", item)
		}
		if !client.Port.GetX(ctx, losAngeles.ID).Enabled || client.Port.Query().Where(port.UnLocodeIn("JPTYO", "KRBAD")).ExistX(ctx) || client.Port.Query().CountX(ctx) != 4 {
			t.Fatal("错误必须回滚此前新增及批量停用")
		}
	})
}
