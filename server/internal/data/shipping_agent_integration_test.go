package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

func TestShippingAgentSyncAndSearchPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	var source struct {
		Data []ShippingAgentSyncRecord `json:"data"`
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "seeds", "shipping-agents.hgj.20261027.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	store := NewIndustryReferenceSyncStore(d)
	if n, err := store.ApplyShippingAgents(ctx, source.Data); err != nil || n != 29 {
		t.Fatalf("首次同步 %d %v", n, err)
	}
	items := d.db.MasterDataItem.Query().Where(masterdataitem.KindEQ(masterdataitem.KindShippingAgent)).AllX(ctx)
	if len(items) != 29 {
		t.Fatalf("初始船代数量 %d", len(items))
	}
	byCode := make(map[string]*ent.MasterDataItem)
	for _, item := range items {
		byCode[item.Code] = item
	}
	for _, row := range source.Data {
		item := byCode[row.Code()]
		if item == nil || item.Name != row.Name || item.NameEn == nil || *item.NameEn != row.NameEN || item.Source != ShippingAgentSnapshotSource || item.SortOrder != row.SortOrder || !item.Enabled || item.SearchKeywords != searchtext.Build(row.Name, row.NameEN) {
			t.Fatalf("来源字段或检索键不一致: row=%+v item=%+v", row, item)
		}
	}
	maintained := items[0].Update().SetName("管理员维护船代").SetNameEn("Administrator Agent").SetSource("manual").SetSortOrder(999).SetEnabled(false).SaveX(ctx)
	if n, err := store.ApplyShippingAgents(ctx, source.Data); err != nil || n != 0 {
		t.Fatalf("重复同步 %d %v", n, err)
	}
	after := d.db.MasterDataItem.GetX(ctx, maintained.ID)
	if after.Name != maintained.Name || after.NameEn == nil || *after.NameEn != *maintained.NameEn || after.Source != "manual" || after.SortOrder != 999 || after.Enabled || !after.UpdatedAt.Equal(maintained.UpdatedAt) || after.SearchKeywords != maintained.SearchKeywords {
		t.Fatalf("管理员修改被覆盖: %+v", after)
	}
	for _, row := range source.Data {
		if d.db.MasterDataItem.Query().Where(masterdataitem.CodeEQ(row.Code()), masterdataitem.KindEQ(masterdataitem.KindShippingAgent)).OnlyX(ctx).ID != byCode[row.Code()].ID {
			t.Fatal("重复同步改变 ID")
		}
	}

	target := d.db.MasterDataItem.Create().SetKind(masterdataitem.KindShippingAgent).SetCode("AGENT-SEARCH").SetName("星海船代").SetNameEn("Starlight Shipping Agency").SaveX(ctx)
	d.db.MasterDataItem.Create().SetKind(masterdataitem.KindShippingAgent).SetCode("DISABLED-SEARCH").SetName("星海船代").SetEnabled(false).SaveX(ctx)
	d.db.MasterDataItem.Create().SetKind(masterdataitem.KindCargoCategory).SetCode("CARGO-SEARCH").SetName("星海船代").SaveX(ctx)
	uc := biz.NewMasterDataUsecase(NewMasterDataRepo(d))
	enabled := true
	for _, keyword := range []string{"AGENT-SEARCH", "星海船代", "Starlight Shipping Agency", "xinghaichuandai", "XHCD"} {
		result, err := uc.List(ctx, uuid.New(), biz.MasterDataListOptions{Kind: biz.MasterDataKindShippingAgent, Keyword: keyword, Enabled: &enabled, Page: 1, PageSize: 20})
		if err != nil || result.Total != 1 || result.Items[0].ID != target.ID {
			t.Fatalf("搜索 %q: %+v %v", keyword, result, err)
		}
	}
	for i := 0; i < 205; i++ {
		d.db.MasterDataItem.Create().SetKind(masterdataitem.KindShippingAgent).SetCode(fmt.Sprintf("ALL-%03d", i)).SetName("全量船代").SetSortOrder(i).SaveX(ctx)
	}
	all, err := uc.ListAll(ctx, uuid.New(), biz.MasterDataKindShippingAgent, nil)
	if err != nil || len(all) != 236 {
		t.Fatalf("真正全量含停用项: %d %v", len(all), err)
	}
	onlyEnabled, err := uc.ListAll(ctx, uuid.New(), biz.MasterDataKindShippingAgent, &enabled)
	if err != nil || len(onlyEnabled) != 234 {
		t.Fatalf("启用筛选: %d %v", len(onlyEnabled), err)
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].SortOrder > all[i].SortOrder || (all[i-1].SortOrder == all[i].SortOrder && all[i-1].Code > all[i].Code) {
			t.Fatal("全量列表排序不稳定")
		}
	}
}

func TestShippingAgentOrderReferencesPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	f := newOrderPostgresFixture(t, d)
	agent := d.db.MasterDataItem.Create().SetKind(masterdataitem.KindShippingAgent).SetCode("ORDER-AGENT").SetName("订单公共船代").SaveX(ctx)
	otherKind := d.db.MasterDataItem.Create().SetKind(masterdataitem.KindCountry).SetCode("WRONG-KIND").SetName("其他字典").SaveX(ctx)
	input := f.validInput()
	input.ShippingAgentID = &agent.ID
	uc := f.newUsecase()
	created, err := uc.Create(ctx, f.organizationID, f.actorID, input)
	if err != nil || created.ShippingAgentName != agent.Name {
		t.Fatalf("创建公共船代订单: %+v %v", created, err)
	}
	agent.Update().SetEnabled(false).SaveX(ctx)
	for _, tc := range []struct {
		name     string
		id       *uuid.UUID
		existing *ent.Order
		valid    bool
	}{
		{"停用新建或普通复制", &agent.ID, nil, false},
		{"保留停用引用", &agent.ID, &ent.Order{ShippingAgentID: &agent.ID}, true},
		{"更换为停用船代", &agent.ID, &ent.Order{ShippingAgentID: &otherKind.ID}, false},
		{"清空", nil, &ent.Order{ShippingAgentID: &agent.ID}, true},
		{"其他种类", &otherKind.ID, nil, false},
		{"保留错误种类仍拒绝", &otherKind.ID, &ent.Order{ShippingAgentID: &otherKind.ID}, false},
		{"供应商ID", &f.partnerID, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := d.WithTx(ctx, func(tx *ent.Tx) error { return validateOrderShippingAgent(ctx, tx, tc.id, tc.existing) })
			if tc.valid && err != nil || !tc.valid && !errors.Is(err, biz.ErrOrderInvalidArgument) {
				t.Fatalf("校验 %v", err)
			}
		})
	}
	missing := uuid.New()
	if err := d.WithTx(ctx, func(tx *ent.Tx) error { return validateOrderShippingAgent(ctx, tx, &missing, nil) }); !errors.Is(err, biz.ErrOrderInvalidArgument) {
		t.Fatalf("缺失船代: %v", err)
	}
	update := f.validUpdateInput()
	update.ShippingAgentID = &agent.ID
	updated, err := uc.UpdateDraft(ctx, f.organizationID, f.actorID, created.ID, created.Version, update)
	if err != nil || updated.ShippingAgentID == nil || *updated.ShippingAgentID != agent.ID || updated.ShippingAgentName != agent.Name {
		t.Fatalf("普通编辑保留停用引用: %+v %v", updated, err)
	}
	got, err := NewOrderRepo(d).Get(ctx, f.organizationID, created.ID)
	if err != nil || got.ShippingAgentName != agent.Name {
		t.Fatalf("详情显示停用船代: %+v %v", got, err)
	}
	listed, err := NewOrderRepo(d).List(ctx, []biz.OrderOrganizationScope{{OrganizationIDs: []uuid.UUID{f.organizationID}, BusinessType: biz.OrderBusinessSE}}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessSE})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].ShippingAgentName != agent.Name {
		t.Fatalf("列表显示停用船代: %+v %v", listed, err)
	}
	copyInput := f.validInput()
	copyInput.ShippingAgentID = &agent.ID
	if _, err := uc.Create(ctx, f.organizationID, f.actorID, copyInput); !errors.Is(err, biz.ErrOrderInvalidArgument) {
		t.Fatalf("普通复制停用船代须重选: %v", err)
	}
}

func TestShippingAgentSplitPreservesDisabledReferencePostgres(t *testing.T) {
	env := newSplitTestEnv(t)
	ctx := t.Context()
	f := createTestSplitFixture(t, env, "AGENT", splitFixtureOptions{})
	agent := env.data.db.MasterDataItem.Create().SetKind(masterdataitem.KindShippingAgent).SetCode("SPLIT-AGENT").SetName("拆票停用船代").SetEnabled(false).SaveX(ctx)
	env.data.db.Order.UpdateOneID(f.order.ID).SetShippingAgentID(agent.ID).SaveX(ctx)
	input := f.standardSplitInput("split-agent-idempotency", "split-agent-fingerprint", splitFixtureOptions{})
	event, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range event.Results {
		order, err := NewOrderRepo(env.data).Get(ctx, env.orgID, result.OrderID)
		if err != nil || order.ShippingAgentID == nil || *order.ShippingAgentID != agent.ID || order.ShippingAgentName != agent.Name {
			t.Fatalf("拆票引用与名称丢失: %+v %v", order, err)
		}
	}
}

func TestShippingAgentMigrationPostgres(t *testing.T) {
	const version = "20261027100000"
	d, cleanup := getIntegrationDataBeforeMigration(t, version)
	t.Cleanup(cleanup)
	ctx := t.Context()
	f := newOrderPostgresFixture(t, d)
	input := f.validInput()
	created, err := f.newUsecase().Create(ctx, f.organizationID, f.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	// 历史合同允许供应商 UUID；仅构造这一已确认历史前态，迁移必须原子拒绝。
	d.db.Order.UpdateOneID(created.ID).SetShippingAgentID(f.partnerID).SaveX(ctx)
	if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err == nil || !strings.Contains(err.Error(), "未映射") {
		t.Fatalf("旧引用应显式拒绝: %v", err)
	}
	old := d.db.Order.GetX(ctx, created.ID)
	if old.ShippingAgentID == nil || *old.ShippingAgentID != f.partnerID {
		t.Fatal("失败迁移修改旧引用")
	}
	var revisions int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261027100000_shipping_agent_masterdata'`).Scan(&revisions); err != nil || revisions != 0 {
		t.Fatalf("失败迁移不应记录 revision %d %v", revisions, err)
	}
	d.db.Order.UpdateOneID(created.ID).ClearShippingAgentID().SaveX(ctx)
	// 已核实开发库缺少同名约束，验证该前态也能收敛。
	if _, err := d.sqlDB.ExecContext(ctx, `ALTER TABLE master_data_items DROP CONSTRAINT IF EXISTS master_data_items_kind_check`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := migration.Apply(ctx, d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatal(err)
		}
	}
	if n := d.db.MasterDataItem.Query().Where(masterdataitem.KindEQ(masterdataitem.KindShippingAgent)).CountX(ctx); n != 0 {
		t.Fatalf("SQL 迁移不应写入 seed: %d", n)
	}
	checks := migrate.MasterDataItemsTable.Annotation.Checks
	if !strings.Contains(checks["master_data_items_kind_check"], "'shipping_agent'") || strings.Contains(checks["master_data_items_kind_check"], "'charge_category'") {
		t.Fatalf("Ent CHECK 未同步: %+v", checks)
	}
	var count int
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid='master_data_items'::regclass AND conname='master_data_items_kind_check'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("数据库 CHECK %d %v", count, err)
	}
	if _, err := d.sqlDB.ExecContext(ctx, `INSERT INTO master_data_items(id,created_at,updated_at,kind,code,name) VALUES($1,now(),now(),'charge_category','RETIRED','退役种类')`, uuid.New()); err == nil {
		t.Fatal("物理约束未拒绝退役种类")
	}
	if err := d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='20261027100000_shipping_agent_masterdata'`).Scan(&revisions); err != nil || revisions != 1 {
		t.Fatalf("成功迁移 revision %d %v", revisions, err)
	}
}
