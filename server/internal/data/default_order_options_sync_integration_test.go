package data

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestSyncDefaultOrderOptionsPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开集成测试数据库: %v", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	// 关库注册为最早的 t.Cleanup（LIFO 中最后执行），保证数据清理先于连接关闭。
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("初始化集成测试 Schema: %v", err)
	}

	first, err := SyncDefaultOrderOptions(ctx, db)
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

	second, err := SyncDefaultOrderOptions(ctx, db)
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

	third, err := SyncDefaultOrderOptions(ctx, db)
	if err != nil {
		t.Fatalf("重复同步默认订单主数据: %v", err)
	}
	if third.Created != 0 {
		t.Fatalf("重复同步不是幂等操作: %+v", third)
	}
	// A 型主数据不再按组织重复落行，同步完成后清理全局种子，避免污染其他用例。
	t.Cleanup(func() { cleanupMasterDataItems(t, client) })
}

func TestCreateDefaultOrderOptionsPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开集成测试数据库: %v", err)
	}
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	// 关库注册为最早的 t.Cleanup（LIFO 中最后执行），保证数据清理先于连接关闭。
	t.Cleanup(func() { _ = client.Close() })
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("初始化集成测试 Schema: %v", err)
	}

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
	// A 型主数据全局唯一，测试完成后清理种子行。
	t.Cleanup(func() { cleanupMasterDataItems(t, client) })
}

// cleanupMasterDataItems 清理集成测试写入的全局主数据种子行。
func cleanupMasterDataItems(t *testing.T, client *ent.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.MasterDataItem.Delete().Exec(ctx); err != nil {
		t.Errorf("清理全局主数据失败: %v", err)
	}
}
