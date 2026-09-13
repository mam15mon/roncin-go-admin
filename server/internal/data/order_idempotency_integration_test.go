package data

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// TestOrderIdempotencyPostgres 验证订单创建与草稿更新的幂等语义：
// 同键同意图重放返回既有资源、同键不同意图返回冲突、UpdateDraft 可变最新键
// 的重放返回当前草稿。依赖 RONCIN_INTEGRATION_DATABASE_SOURCE。
func TestOrderIdempotencyPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             source,
		AutoMigrate:        true,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	defer cleanup()

	t.Run("同键同意图创建重放返回同一订单", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.IdempotencyKey = "order-create-replay-" + fixture.suffix

		first, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, input)
		if err != nil {
			t.Fatalf("首次创建订单: %v", err)
		}
		replay := *input
		replay.SeaDocumentInput = input.SeaDocumentInput
		replay.SeaMasterBillInput = input.SeaMasterBillInput
		second, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, &replay)
		if err != nil {
			t.Fatalf("同键同意图重放: %v", err)
		}
		if second.ID != first.ID || second.OrderNo != first.OrderNo {
			t.Fatalf("重放返回不同订单: first=%s/%s second=%s/%s", first.ID, first.OrderNo, second.ID, second.OrderNo)
		}
		fixture.requireCommittedOrders(1, 1)
	})

	t.Run("同键不同意图创建返回幂等冲突", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.IdempotencyKey = "order-create-conflict-" + fixture.suffix
		if _, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, input); err != nil {
			t.Fatalf("首次创建订单: %v", err)
		}
		conflicting := fixture.validInput()
		conflicting.CustomerReferenceNo = "OTHER-INTENT-" + fixture.suffix
		conflicting.IdempotencyKey = input.IdempotencyKey
		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, conflicting)
		if err != biz.ErrOrderIdempotencyConflict {
			t.Fatalf("同键不同意图应返回幂等冲突, got %v", err)
		}
		fixture.requireCommittedOrders(1, 1)
	})

	t.Run("同键仅改请求内容返回幂等冲突", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.IdempotencyKey = "order-create-drift-" + fixture.suffix
		if _, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, input); err != nil {
			t.Fatalf("首次创建订单: %v", err)
		}
		// 全量请求哈希必须覆盖清单式比对遗漏的字段（如货物描述）。
		drifted := fixture.validInput()
		drifted.GoodsDescription = "内容漂移-" + fixture.suffix
		drifted.IdempotencyKey = input.IdempotencyKey
		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, drifted)
		if err != biz.ErrOrderIdempotencyConflict {
			t.Fatalf("同键仅改货物描述应返回幂等冲突, got %v", err)
		}
		fixture.requireCommittedOrders(1, 1)
	})

	t.Run("并发同键创建只落一单且双方返回同一订单", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.IdempotencyKey = "order-create-race-" + fixture.suffix
		operations := make([]func(context.Context) (*biz.Order, error), 0, 2)
		for range 2 {
			operations = append(operations, func(ctx context.Context) (*biz.Order, error) {
				return usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
			})
		}
		results := runOrderWritesConcurrently(operations...)
		for index, result := range results {
			if result.err != nil {
				t.Fatalf("第 %d 个并发同键请求失败: %v", index+1, result.err)
			}
			if result.order.ID != results[0].order.ID {
				t.Fatalf("并发同键创建返回了不同订单: %s vs %s", result.order.ID, results[0].order.ID)
			}
		}
		fixture.requireCommittedOrders(1, 1)
	})

	t.Run("草稿更新同键同版本重放返回当前草稿", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建订单: %v", err)
		}
		updateKey := "order-update-replay-" + fixture.suffix
		updateInput := fixture.validUpdateInput()
		updateInput.Notes = "第一轮草稿修改"
		updateInput.IdempotencyKey = updateKey
		updated, err := usecase.UpdateDraft(context.Background(), fixture.organizationID, fixture.actorID, created.ID, created.Version, updateInput)
		if err != nil {
			t.Fatalf("首次草稿更新: %v", err)
		}
		if updated.Version != created.Version+1 {
			t.Fatalf("首次更新后版本 = %d, want %d", updated.Version, created.Version+1)
		}

		// 模拟超时重试：同键 + 同 expectedVersion 的请求再次到达。
		replayInput := fixture.validUpdateInput()
		replayInput.Notes = updateInput.Notes
		replayInput.IdempotencyKey = updateKey
		replayed, err := usecase.UpdateDraft(context.Background(), fixture.organizationID, fixture.actorID, created.ID, created.Version, replayInput)
		if err != nil {
			t.Fatalf("同键同版本重放应返回当前草稿: %v", err)
		}
		if replayed.Version != created.Version+1 {
			t.Fatalf("重放不得再次递增版本: version = %d, want %d", replayed.Version, created.Version+1)
		}
		auditCount, err := data.db.AuditLog.Query().Where(
			auditlogent.OrganizationIDEQ(fixture.organizationID),
			auditlogent.ActionEQ("order.update"),
		).Count(context.Background())
		if err != nil {
			t.Fatalf("读取草稿更新审计: %v", err)
		}
		if auditCount != 1 {
			t.Fatalf("重放产生重复审计记录: %d 条, want 1 条", auditCount)
		}
	})

	t.Run("草稿更新键不同或版本不匹配走乐观锁冲突", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建订单: %v", err)
		}
		staleInput := fixture.validUpdateInput()
		staleInput.IdempotencyKey = "order-update-stale-" + fixture.suffix
		if _, err := usecase.UpdateDraft(context.Background(), fixture.organizationID, fixture.actorID, created.ID, created.Version+5, staleInput); err != biz.ErrOrderStatusConflict {
			t.Fatalf("版本不匹配应返回乐观锁冲突, got %v", err)
		}
	})

	t.Run("无键创建仍落非空幂等键且可正常更新", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("无键创建订单: %v", err)
		}
		stored, err := data.db.Order.Query().Where(orderent.IDEQ(created.ID)).Only(context.Background())
		if err != nil {
			t.Fatalf("读取订单: %v", err)
		}
		if stored.IdempotencyKey == "" {
			t.Fatal("无键创建也必须落非空幂等键")
		}
		updateInput := fixture.validUpdateInput()
		updateInput.IdempotencyKey = "order-update-after-naked-" + fixture.suffix
		updated, err := usecase.UpdateDraft(context.Background(), fixture.organizationID, fixture.actorID, created.ID, created.Version, updateInput)
		if err != nil {
			t.Fatalf("无键创建后带键更新: %v", err)
		}
		refreshed, err := data.db.Order.Query().Where(orderent.IDEQ(created.ID)).Only(context.Background())
		if err != nil {
			t.Fatalf("重读订单: %v", err)
		}
		if refreshed.IdempotencyKey != updateInput.IdempotencyKey {
			t.Fatalf("草稿更新应覆写幂等键 = %q, want %q", refreshed.IdempotencyKey, updateInput.IdempotencyKey)
		}
		if updated.Version != created.Version+1 {
			t.Fatalf("更新后版本 = %d, want %d", updated.Version, created.Version+1)
		}
	})
}
