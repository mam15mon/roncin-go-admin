package data

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
)

// 两个竞争请求都经过真实仓储与订单行锁，不能只用顺序调用模拟锁外旧判断。
func TestOrderContainerShipmentConcurrencyPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	for _, winner := range []string{"先新增实际箱", "先切换拼箱"} {
		t.Run(winner, func(t *testing.T) {
			fixture, created, container := newContainerShipmentFixture(t, data)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			input := fixture.validUpdateInput()
			lcl := biz.OrderShipmentLCL
			input.ShipmentType = &lcl
			beforeAudit := data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID)).CountX(ctx)
			add := func(operationCtx context.Context) error {
				_, err := NewOrderContainerRepo(data).Add(operationCtx, fixture.organizationID, created.ID, container,
					testGateAuditEvent(fixture, "order.container.add"))
				return err
			}
			update := func(operationCtx context.Context) error {
				_, err := fixture.newUsecase().UpdateDraft(operationCtx, fixture.organizationID, fixture.actorID, created.ID, created.Version, input)
				return err
			}
			winningOperation, competingOperation := add, update
			if winner == "先切换拼箱" {
				winningOperation, competingOperation = update, add
			}
			competingResult := make(chan error, 1)
			err := data.WithinTransaction(ctx, func(txCtx context.Context) error {
				client, err := data.client(txCtx)
				if err != nil {
					return err
				}
				if _, err = client.Order.Query().Where(orderent.IDEQ(created.ID)).ForUpdate().Only(txCtx); err != nil {
					return err
				}
				go func() { competingResult <- competingOperation(ctx) }()
				if err = waitContainerShipmentOrderLock(ctx, data, competingResult); err != nil {
					return err
				}
				// 嵌套 WithTx 复用已持有的订单锁；提交后才让竞争请求继续。
				return winningOperation(txCtx)
			})
			if err != nil {
				t.Fatalf("获胜操作失败: %v", err)
			}
			select {
			case err = <-competingResult:
			case <-ctx.Done():
				t.Fatal("竞争请求未在订单锁释放后结束", ctx.Err())
			}
			if !errors.Is(err, biz.ErrOrderContainerShipmentType) {
				t.Fatalf("竞争请求应因锁后类型/实箱复核拒绝，实际: %v", err)
			}
			actual := data.db.Order.GetX(ctx, created.ID)
			count := data.db.OrderContainer.Query().Where(ordercontainerent.OrderIDEQ(created.ID)).CountX(ctx)
			if winner == "先新增实际箱" {
				if actual.ShipmentType == nil || *actual.ShipmentType != orderent.ShipmentTypeFCL || count != 1 || actual.Version != created.Version {
					t.Fatalf("加箱获胜后必须保持 FCL、一箱及原父版本: order=%+v count=%d", actual, count)
				}
			} else if actual.ShipmentType == nil || *actual.ShipmentType != orderent.ShipmentTypeLCL || count != 0 || actual.Version != created.Version+1 {
				t.Fatalf("切拼箱获胜后必须是 LCL、无普通箱及递增父版本: order=%+v count=%d", actual, count)
			}
			afterAudit := data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID)).CountX(ctx)
			if afterAudit != beforeAudit+1 {
				t.Fatalf("失败方不能写成功审计: before=%d after=%d", beforeAudit, afterAudit)
			}
		})
	}
}

func TestOrderContainerShipmentMutationBoundariesPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	fixture, created, container := newContainerShipmentFixture(t, data)
	ctx := context.Background()
	repo := NewOrderContainerRepo(data)
	added, err := repo.Add(ctx, fixture.organizationID, created.ID, container, testGateAuditEvent(fixture, "order.container.add"))
	if err != nil {
		t.Fatalf("FCL 新增普通箱: %v", err)
	}
	container.Note = new("正常 FCL 编辑")
	updated, err := repo.Update(ctx, fixture.organizationID, created.ID, added.ID, added.Version, container, testGateAuditEvent(fixture, "order.container.update"))
	if err != nil || updated.Version != added.Version+1 {
		t.Fatalf("FCL 更新普通箱: result=%+v error=%v", updated, err)
	}
	for _, shipment := range []biz.OrderShipmentType{biz.OrderShipmentLCL, biz.OrderShipmentBreakBulk} {
		input := fixture.validUpdateInput()
		input.ShipmentType = &shipment
		if _, err = fixture.newUsecase().UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, input); !errors.Is(err, biz.ErrOrderContainerShipmentType) {
			t.Fatalf("已有普通箱时切换 %s 应拒绝: %v", shipment, err)
		}
	}
	// 构造既有异常形态，确认更新拒绝而删除仍可清理，避免修复门禁锁死资料。
	data.db.Order.UpdateOneID(created.ID).SetShipmentType(orderent.ShipmentTypeLCL).SaveX(ctx)
	if _, err = repo.Update(ctx, fixture.organizationID, created.ID, updated.ID, updated.Version, container, testGateAuditEvent(fixture, "order.container.update")); !errors.Is(err, biz.ErrOrderContainerShipmentType) {
		t.Fatalf("LCL 更新普通箱应拒绝: %v", err)
	}
	if err = repo.Remove(ctx, fixture.organizationID, created.ID, updated.ID, updated.Version, testGateAuditEvent(fixture, "order.container.remove")); err != nil {
		t.Fatalf("LCL 异常普通箱应允许删除: %v", err)
	}
	if _, err = repo.Add(ctx, fixture.organizationID, created.ID, container, testGateAuditEvent(fixture, "order.container.add")); !errors.Is(err, biz.ErrOrderContainerShipmentType) {
		t.Fatalf("LCL 新增普通箱应拒绝: %v", err)
	}
}

func newContainerShipmentFixture(t *testing.T, data *Data) (*orderPostgresFixture, *biz.Order, *biz.OrderContainer) {
	t.Helper()
	fixture := newOrderPostgresFixture(t, data)
	ctx := context.Background()
	input := fixture.validInput()
	fcl := biz.OrderShipmentFCL
	input.ShipmentType = &fcl
	created, err := fixture.newUsecase().Create(ctx, fixture.organizationID, fixture.actorID, input)
	if err != nil {
		t.Fatalf("创建 FCL 测试订单: %v", err)
	}
	spec := data.db.MasterDataItem.Create().SetKind(masterdataitement.KindContainerSpec).
		SetCode("SHIPMENT-" + fixture.suffix).SetName("运输类型并发测试箱型").SaveX(ctx)
	t.Cleanup(func() {
		if _, err := data.db.OrderContainer.Delete().Where(ordercontainerent.OrganizationIDEQ(fixture.organizationID)).Exec(context.Background()); err != nil {
			t.Errorf("清理并发测试实箱: %v", err)
		}
		if err := data.db.MasterDataItem.DeleteOneID(spec.ID).Exec(context.Background()); err != nil {
			t.Errorf("清理并发测试箱型: %v", err)
		}
	})
	return fixture, created, &biz.OrderContainer{
		ID: uuid.New(), ContainerNo: "SHIPMENT-" + fixture.suffix,
		ContainerSpecID: spec.ID, PackageCount: 10, GrossWeightKg: 100, VolumeCbm: 1,
	}
}

func waitContainerShipmentOrderLock(ctx context.Context, data *Data, result <-chan error) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		err := data.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM pg_locks l JOIN pg_stat_activity a ON a.pid = l.pid
WHERE l.relation = 'orders'::regclass AND a.wait_event_type = 'Lock'
AND a.pid <> pg_backend_pid() AND a.query LIKE '%"orders"%')`).Scan(&waiting)
		if err != nil {
			return err
		}
		if waiting {
			return nil
		}
		select {
		case err := <-result:
			return fmt.Errorf("持锁期间竞争操作提前结束: %v", err)
		case <-ticker.C:
		case <-ctx.Done():
			return fmt.Errorf("竞争请求未进入订单锁等待: %w", ctx.Err())
		}
	}
}
