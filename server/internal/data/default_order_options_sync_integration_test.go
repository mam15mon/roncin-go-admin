package data

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
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
	defer client.Close()
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("初始化集成测试 Schema: %v", err)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	organization, err := client.Organization.Create().
		SetCode("ORDER-SEED-" + suffix).
		SetName("订单种子测试组织").
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	defer client.Organization.DeleteOneID(organization.ID).Exec(ctx)

	first, err := SyncDefaultOrderOptions(ctx, db)
	if err != nil {
		t.Fatalf("首次同步默认订单主数据: %v", err)
	}
	if first.Created < len(biz.DefaultOrderOptions()) {
		t.Fatalf("首次补齐数量 = %d, want >= %d", first.Created, len(biz.DefaultOrderOptions()))
	}
	count, err := client.MasterDataItem.Query().Where(masterdataent.OrganizationIDEQ(organization.ID)).Count(ctx)
	if err != nil || count != len(biz.DefaultOrderOptions()) {
		t.Fatalf("组织默认订单主数据数 = %d, want %d, error = %v", count, len(biz.DefaultOrderOptions()), err)
	}

	booking, err := client.MasterDataItem.Query().Where(
		masterdataent.OrganizationIDEQ(organization.ID),
		masterdataent.KindEQ(masterdataent.KindServiceType),
		masterdataent.CodeEQ("BOOKING"),
	).Only(ctx)
	if err != nil {
		t.Fatalf("查询订舱服务类型: %v", err)
	}
	if _, err := booking.Update().SetName("自定义订舱").SetEnabled(false).Save(ctx); err != nil {
		t.Fatalf("修改订舱服务类型: %v", err)
	}
	if _, err := client.MasterDataItem.Delete().Where(
		masterdataent.OrganizationIDEQ(organization.ID),
		masterdataent.KindEQ(masterdataent.KindServiceType),
		masterdataent.CodeEQ("TRUCKING"),
	).Exec(ctx); err != nil {
		t.Fatalf("删除拖车服务类型: %v", err)
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
}
