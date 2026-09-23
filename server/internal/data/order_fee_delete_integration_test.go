package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	enterpriseresourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeeenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
)

// TestOrderFeeDeleteByBillOccupancyPostgres 验证费用按账单占用关系物理删除：
// 未被未取消账单（含草稿账单）包含即可删；占用时事务内以事实复核拒绝；取消
// 账单后恢复可删且历史行快照保留、来源引用置空；与建账并发在费用行锁上互斥。
// 隔离 Schema 执行全量迁移链冷启动，同时验证外键置空/级联契约。
func TestOrderFeeDeleteByBillOccupancyPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	// 先注册数据清理（LIFO 后执行），保证夹具清理先于关闭数据库连接。
	t.Cleanup(cleanupData)
	fixture := newFinanceBillPostgresFixture(t, data)

	orderFeeRepo := NewOrderFeeRepo(data)
	billUsecase := fixture.newUsecase(NewFinanceBillRepo(data))
	// 账单取消会写入 cancelled_by 用户外键，需要真实用户行。
	actorUser, userErr := data.db.User.Create().
		SetDisplayName("费用删除集成测试用户-" + fixture.suffix).
		Save(context.Background())
	if userErr != nil {
		t.Fatalf("创建测试操作用户: %v", userErr)
	}
	actor := actorUser.ID
	orgIDs := []uuid.UUID{fixture.organizationID}

	deleteFee := func(t *testing.T, feeID uuid.UUID, expectedVersion uint64) error {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return orderFeeRepo.Remove(ctx, fixture.organizationID, fixture.orderID, feeID, actor, expectedVersion, "", &biz.AuditEvent{
			OrganizationID: &fixture.organizationID,
			UserID:         &actor,
			Action:         "order.fee.delete",
			Result:         "success",
			Details:        map[string]string{"fee.id": feeID.String()},
		})
	}

	loadFeeVersion := func(t *testing.T, feeID uuid.UUID) uint64 {
		t.Helper()
		fee, err := data.db.OrderFee.Get(context.Background(), feeID)
		if err != nil {
			t.Fatalf("读取费用: %v", err)
		}
		return fee.Version
	}

	t.Run("未建账费用可直接删除且记录消失", func(t *testing.T) {
		firstID := fixture.createUnbilledFee("del-unbilled-1")
		secondID := fixture.createUnbilledFee("del-unbilled-2")

		if err := deleteFee(t, firstID, 1); err != nil {
			t.Fatalf("删除未建账费用被拒绝: %v", err)
		}
		if err := deleteFee(t, secondID, 1); err != nil {
			t.Fatalf("删除未建账费用被拒绝: %v", err)
		}
		count, err := data.db.OrderFee.Query().Where(orderfeeent.IDIn(firstID, secondID)).Count(context.Background())
		if err != nil || count != 0 {
			t.Fatalf("删除后费用记录仍存在: count=%d err=%v", count, err)
		}
		audits, auditErr := data.db.AuditLog.Query().
			Where(auditlogent.ActionEQ("order.fee.delete"), auditlogent.OrganizationIDEQ(fixture.organizationID)).
			All(context.Background())
		if auditErr != nil {
			t.Fatalf("读取删除审计: %v", auditErr)
		}
		matched := false
		for _, event := range audits {
			if strings.Contains(string(event.Details), firstID.String()) {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("删除审计事件缺失费用快照明细: %d 条", len(audits))
		}
	})

	t.Run("草稿账单占用即拒绝，其他费用建账不影响本行", func(t *testing.T) {
		occupiedID := fixture.createUnbilledFee("del-occupied")
		freeID := fixture.createUnbilledFee("del-free")
		bill, err := billUsecase.Create(context.Background(), fixture.organizationID, actor, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{occupiedID}, BillDate: financeBillIntegrationDate,
			IdempotencyKey: "bill-occupy-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("建立草稿账单失败: %v", err)
		}

		if err := deleteFee(t, occupiedID, loadFeeVersion(t, occupiedID)); !errors.Is(err, biz.ErrOrderFeeBillOccupied) {
			t.Fatalf("占用费用删除错误 = %v，期望 ErrOrderFeeBillOccupied", err)
		}
		// 同订单其他费用未建账，不受影响仍可删除。
		if err := deleteFee(t, freeID, 1); err != nil {
			t.Fatalf("未占用费用删除被同订单账单阻断: %v", err)
		}
		if _, err := data.db.OrderFee.Get(context.Background(), occupiedID); err != nil {
			t.Fatalf("被占用费用不应被删除: %v", err)
		}
		if bill == nil || bill.BillNo == "" {
			t.Fatal("账单号缺失")
		}
	})

	t.Run("取消账单后可删除且历史行快照保留、来源置空", func(t *testing.T) {
		feeID := fixture.createUnbilledFee("del-after-cancel")
		bill, err := billUsecase.Create(context.Background(), fixture.organizationID, actor, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate,
			IdempotencyKey: "bill-cancel-del-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("建立账单失败: %v", err)
		}
		if _, err := billUsecase.Cancel(context.Background(), orgIDs, actor, bill.ID, bill.Version, "删除前取消"); err != nil {
			t.Fatalf("取消账单失败: %v", err)
		}

		if err := deleteFee(t, feeID, loadFeeVersion(t, feeID)); err != nil {
			t.Fatalf("取消账单后删除费用被拒绝: %v", err)
		}
		line, err := data.db.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(bill.ID)).Only(context.Background())
		if err != nil {
			t.Fatalf("历史账单行应保留: %v", err)
		}
		if line.OrderFeeID != uuid.Nil {
			t.Fatalf("删除后历史行来源应置空: order_fee_id=%s", line.OrderFeeID)
		}
		if line.FeeCode != "OCEAN_FREIGHT" || line.TotalAmount != "100.00000000" || line.Currency != "CNY" {
			t.Fatalf("历史行快照缺失: %#v", line)
		}
	})

	t.Run("取消后重新建账则再次拒绝", func(t *testing.T) {
		feeID := fixture.createUnbilledFee("del-rebill")
		first, err := billUsecase.Create(context.Background(), fixture.organizationID, actor, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate,
			IdempotencyKey: "bill-rebill-1-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("首次建账失败: %v", err)
		}
		if _, err := billUsecase.Cancel(context.Background(), orgIDs, actor, first.ID, first.Version, "重建前取消"); err != nil {
			t.Fatalf("取消首次账单失败: %v", err)
		}
		second, err := billUsecase.Create(context.Background(), fixture.organizationID, actor, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate,
			IdempotencyKey: "bill-rebill-2-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("重新建账失败: %v", err)
		}
		if err := deleteFee(t, feeID, loadFeeVersion(t, feeID)); !errors.Is(err, biz.ErrOrderFeeBillOccupied) {
			t.Fatalf("重新建账后删除错误 = %v，期望 ErrOrderFeeBillOccupied", err)
		}
		if second == nil {
			t.Fatal("新账单缺失")
		}
	})

	t.Run("费用标签关联随删除级联清理", func(t *testing.T) {
		feeID := fixture.createUnbilledFee("del-tags")
		resource, err := data.db.EnterpriseResource.Create().
			SetOrganizationID(fixture.organizationID).
			SetResourceType(enterpriseresourceent.ResourceTypeTAG).
			SetShortName("删除级联标签-" + fixture.suffix).
			Save(context.Background())
		if err != nil {
			t.Fatalf("创建标签资源: %v", err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = data.db.EnterpriseResource.DeleteOneID(resource.ID).Exec(ctx)
		})
		if _, err := data.db.OrderFeeEnterpriseTag.Create().
			SetOrganizationID(fixture.organizationID).
			SetOrderFeeID(feeID).
			SetTagResourceID(resource.ID).
			Save(context.Background()); err != nil {
			t.Fatalf("创建费用标签关联: %v", err)
		}
		if err := deleteFee(t, feeID, 1); err != nil {
			t.Fatalf("删除带标签费用失败: %v", err)
		}
		links, err := data.db.OrderFeeEnterpriseTag.Query().Where(orderfeeenterprisetagent.OrderFeeIDEQ(feeID)).Count(context.Background())
		if err != nil || links != 0 {
			t.Fatalf("费用标签关联应级联清理: count=%d err=%v", links, err)
		}
	})

	t.Run("版本冲突拒绝且不删除", func(t *testing.T) {
		feeID := fixture.createUnbilledFee("del-version")
		if err := deleteFee(t, feeID, 99); !errors.Is(err, biz.ErrOrderFeeVersionConflict) {
			t.Fatalf("版本冲突删除错误 = %v，期望 ErrOrderFeeVersionConflict", err)
		}
		exists, err := data.db.OrderFee.Query().Where(orderfeeent.IDEQ(feeID)).Exist(context.Background())
		if err != nil || !exists {
			t.Fatalf("版本冲突不应删除费用: exists=%t err=%v", exists, err)
		}
	})

	t.Run("删除与建账并发互斥且不产生悬挂引用", func(t *testing.T) {
		feeID := fixture.createUnbilledFee("del-race")
		start := make(chan struct{})
		var wg sync.WaitGroup
		deleteErr := make(chan error, 1)
		billErr := make(chan error, 1)
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			deleteErr <- deleteFee(t, feeID, 1)
		}()
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := billUsecase.Create(ctx, fixture.organizationID, actor, biz.CreateFinanceBillInput{
				FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate,
				IdempotencyKey: "bill-race-" + fixture.suffix, SettlementAccountID: fixture.accountID,
			})
			billErr <- err
		}()
		close(start)
		wg.Wait()
		delResult := <-deleteErr
		billResult := <-billErr

		feeExists, err := data.db.OrderFee.Query().Where(orderfeeent.IDEQ(feeID)).Exist(context.Background())
		if err != nil {
			t.Fatalf("核对费用存在性: %v", err)
		}
		dangling, err := data.db.FinanceBillLine.Query().
			Where(financebilllineent.OrderFeeIDEQ(feeID), financebilllineent.ActiveEQ(true)).
			Count(context.Background())
		if err != nil {
			t.Fatalf("核对活动账单行: %v", err)
		}
		switch {
		case delResult == nil:
			// 删除胜出：费用必须已消失，建账必须失败，不得留下活动账单行。
			if feeExists {
				t.Fatal("删除成功后费用仍存在")
			}
			if billResult == nil {
				t.Fatal("费用已删除仍建账成功")
			}
			if dangling != 0 {
				t.Fatalf("删除胜出后出现悬挂活动账单行: %d", dangling)
			}
		case errors.Is(delResult, biz.ErrOrderFeeBillOccupied),
			errors.Is(delResult, biz.ErrOrderFeeVersionConflict):
			// 建账胜出：费用保留——或被活动账单占用，或因建账推进版本而以版本
			// 冲突拒绝，两种失败都以事务内事实保证了互斥。
			if !feeExists {
				t.Fatal("占用/版本拒绝后费用不应被删除")
			}
			if billResult != nil {
				t.Fatalf("占用或版本冲突成立的先决是建账成功: billErr=%v", billResult)
			}
			if dangling != 1 {
				t.Fatalf("建账胜出后活动账单行数 = %d，期望 1", dangling)
			}
		default:
			t.Fatalf("并发删除返回非预期错误: %v", delResult)
		}
	})
}
