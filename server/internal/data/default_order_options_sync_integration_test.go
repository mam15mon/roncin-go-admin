package data

import (
	"context"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
)

// TestSyncDefaultOrderOptionsPostgres 在隔离 Schema 上验证默认订单主数据的幂等补齐：
// 迁移链本身会播种同批主数据，先清空主数据还原「种子缺失」前置，再断言同步补齐
// 全部默认项、不覆盖业务人员维护过的行、修复误删行且重复同步保持幂等。
func TestSyncDefaultOrderOptionsPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	client := data.db
	clearMasterDataItems(t, client)

	first, err := SyncDefaultOrderOptions(ctx, data.sqlDB)
	if err != nil {
		t.Fatalf("首次同步默认订单主数据: %v", err)
	}
	if first.Created < len(biz.DefaultOrderOptions()) {
		t.Fatalf("首次补齐数量 = %d, want >= %d", first.Created, len(biz.DefaultOrderOptions()))
	}
	total, err := client.MasterDataItem.Query().Count(ctx)
	if err != nil || total < len(biz.DefaultOrderOptions()) {
		t.Fatalf("全局默认订单主数据数 = %d, want >= %d, error = %v", total, len(biz.DefaultOrderOptions()), err)
	}

	booking, err := client.MasterDataItem.Query().Where(
		masterdataent.KindEQ(masterdataent.KindChargeCategory),
		masterdataent.CodeEQ("BOOKING"),
	).Only(ctx)
	if err != nil {
		t.Fatalf("查询订舱费用大类: %v", err)
	}
	if !strings.Contains(booking.SearchKeywords, "DINGCANG") {
		t.Fatalf("订舱费用大类缺少拼音检索键: %q", booking.SearchKeywords)
	}
	if _, err := booking.Update().SetName("自定义订舱").SetEnabled(false).Save(ctx); err != nil {
		t.Fatalf("修改订舱费用大类: %v", err)
	}
	if _, err := client.MasterDataItem.Delete().Where(
		masterdataent.KindEQ(masterdataent.KindChargeCategory),
		masterdataent.CodeEQ("TRUCKING"),
	).Exec(ctx); err != nil {
		t.Fatalf("删除拖车费用大类: %v", err)
	}

	second, err := SyncDefaultOrderOptions(ctx, data.sqlDB)
	if err != nil {
		t.Fatalf("补齐缺失订单主数据: %v", err)
	}
	if second.Created < 1 {
		t.Fatalf("补齐缺失订单主数据数量 = %d, want >= 1", second.Created)
	}
	booking, err = client.MasterDataItem.Query().Where(masterdataent.IDEQ(booking.ID)).Only(ctx)
	if err != nil || booking.Name != "自定义订舱" || booking.Enabled {
		t.Fatalf("同步覆盖了已有订舱主数据: booking=%#v error=%v", booking, err)
	}

	third, err := SyncDefaultOrderOptions(ctx, data.sqlDB)
	if err != nil {
		t.Fatalf("重复同步默认订单主数据: %v", err)
	}
	if third.Created != 0 {
		t.Fatalf("重复同步不是幂等操作: %+v", third)
	}
}

func TestCreateDefaultOrderOptionsPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	client := data.db
	// 迁移链已播种同批主数据；先清空还原「空库」前置，验证创建函数自身的写入行为。
	clearMasterDataItems(t, client)

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("开启测试事务: %v", err)
	}
	if err := CreateDefaultOrderOptions(ctx, tx); err != nil {
		_ = tx.Rollback()
		t.Fatalf("创建默认订单主数据: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("提交测试事务: %v", err)
	}

	containerSpec, err := client.MasterDataItem.Query().Where(
		masterdataent.KindEQ(masterdataent.KindContainerSpec),
		masterdataent.CodeEQ("20GP"),
	).Only(ctx)
	if err != nil {
		t.Fatalf("查询 20GP 箱型: %v", err)
	}
	if containerSpec.TeuFactor == nil || *containerSpec.TeuFactor != "1" {
		t.Fatalf("20GP TEU 系数 = %v, want 1", containerSpec.TeuFactor)
	}
}

// clearMasterDataItems 清空隔离 Schema 内的主数据行，还原默认订单选项种子缺失的前置。
func clearMasterDataItems(t *testing.T, client *ent.Client) {
	t.Helper()
	if _, err := client.MasterDataItem.Delete().Exec(context.Background()); err != nil {
		t.Fatalf("清空主数据失败: %v", err)
	}
}
