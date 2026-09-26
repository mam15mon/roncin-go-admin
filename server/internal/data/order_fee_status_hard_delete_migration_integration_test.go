package data

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// readMigrationContent 读取正式迁移文件内容，供在隔离库上定点重放。
func readMigrationContent(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
	if err != nil {
		t.Fatalf("读取迁移文件失败: %v", err)
	}
	return string(content)
}

// restoreOrderFeeStatusColumns 在费用状态已退役的当前 Schema 上恢复迁移前状态：
// 重加 status/cancelled_* 列、状态 CHECK、一致性 CHECK、状态索引与 cancelled_by
// 外键，并按旧行为把指定费用写成 CANCELLED 行，使重放真实迁移文件时覆盖其
// 完整 DDL/DML。语句不带参数走简单协议可多语句执行。
func restoreOrderFeeStatusColumns(t *testing.T, data *Data, ctx context.Context, cancelledIDs []uuid.UUID) {
	t.Helper()
	restore := `
		ALTER TABLE "order_fees" ADD COLUMN "status" character varying NOT NULL DEFAULT 'UNBILLED';
		ALTER TABLE "order_fees" ADD COLUMN "cancelled_at" timestamptz NULL;
		ALTER TABLE "order_fees" ADD COLUMN "cancelled_by" uuid NULL;
		ALTER TABLE "order_fees" ADD COLUMN "cancellation_reason" character varying NULL;
		ALTER TABLE "order_fees"
		  ADD CONSTRAINT "order_fees_status_check"
		  CHECK ("status" IN ('UNBILLED', 'BILLED', 'CANCELLED'));
		ALTER TABLE "order_fees"
		  ADD CONSTRAINT "order_fees_cancellation_consistency"
		  CHECK (
		    ("status" = 'CANCELLED' AND "cancelled_at" IS NOT NULL AND "cancelled_by" IS NOT NULL AND "cancellation_reason" IS NOT NULL)
		    OR
		    ("status" <> 'CANCELLED' AND "cancelled_at" IS NULL AND "cancelled_by" IS NULL AND "cancellation_reason" IS NULL)
		  );
		ALTER TABLE "order_fees" ALTER COLUMN "status" SET DEFAULT 'UNBILLED';
		CREATE INDEX "orderfee_order_id_status_created_at"
		  ON "order_fees" ("order_id", "status", "created_at");
		ALTER TABLE "order_fees"
		  ADD CONSTRAINT "order_fees_users_cancelled_by"
		  FOREIGN KEY ("cancelled_by") REFERENCES "users" ("id") ON DELETE NO ACTION;`
	if _, err := data.sqlDB.ExecContext(ctx, restore); err != nil {
		t.Fatalf("恢复迁移前列状态失败: %v", err)
	}
	// 取消事实需要操作者：无用户时创建迁移重放专用账号，不伪造既有身份。
	actorID, actorErr := ensureReplayCancelActor(t, data, ctx)
	if actorErr != nil {
		t.Fatalf("准备迁移重放操作者: %v", actorErr)
	}
	for _, id := range cancelledIDs {
		if _, err := data.sqlDB.ExecContext(ctx,
			`UPDATE "order_fees" SET "status" = 'CANCELLED', "cancelled_at" = '2026-09-25T00:00:00Z',
			     "cancelled_by" = $2, "cancellation_reason" = '迁移重放测试作废'
			 WHERE "id" = $1`, id, actorID); err != nil {
			t.Fatalf("写入已作废费用 %s 失败: %v", id, err)
		}
	}
}

// createReplayBill 直接以 ent 构造迁移前状态的账单，不经建账用例以避免汇率
// 解析等与迁移重放无关的依赖。
func createReplayBill(t *testing.T, data *Data, ctx context.Context, fixture *financeBillPostgresFixture, feeID uuid.UUID, billStatus financebillent.Status, lineActive bool) uuid.UUID {
	t.Helper()
	billID := uuid.Must(uuid.NewV7())
	billCreate := data.db.FinanceBill.Create().
		SetID(billID).
		SetOrganizationID(fixture.organizationID).
		SetBillNo("HD-BILL-" + billStatus.String() + "-" + fixture.suffix + "-" + billID.String()[:8]).
		SetIdempotencyKey("hd-bill-" + billStatus.String() + "-" + fixture.suffix + "-" + billID.String()[:8]).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(billStatus).
		SetSettlementPartyID(fixture.partnerID).
		SetSettlementPartyName("迁移重放测试客户-" + fixture.suffix).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("100.00000000").
		SetFeeCount(1).
		SetBillDate(financeBillIntegrationDate).
		SetVersion(1)
	if billStatus == financebillent.StatusCANCELLED {
		actorID, actorErr := ensureReplayCancelActor(t, data, ctx)
		if actorErr != nil {
			t.Fatalf("准备迁移重放操作者: %v", actorErr)
		}
		billCreate = billCreate.
			SetCancelledAt(time.Now().UTC()).
			SetCancelledBy(actorID).
			SetCancellationReason("迁移重放测试取消")
	}
	if _, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, fixture.accountID, "CNY").Save(ctx); err != nil {
		t.Fatalf("创建迁移重放账单: %v", err)
	}
	if _, err := data.db.FinanceBillLine.Create().
		SetBillID(billID).
		SetOrderFeeID(feeID).
		SetOrderID(fixture.orderID).
		SetOrderNo("HD-HISTORICAL").
		SetFeeCode("HD-FEE").
		SetFeeName("迁移重放历史费用").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("100.00000000").
		SetActive(lineActive).
		Save(ctx); err != nil {
		t.Fatalf("创建迁移重放账单行: %v", err)
	}
	return billID
}

// ensureReplayCancelActor 返回用于旧撤销元数据的操作者：优先复用既有用户，
// 隔离库无用户时创建迁移重放专用账号。
func ensureReplayCancelActor(t *testing.T, data *Data, ctx context.Context) (uuid.UUID, error) {
	existing, err := data.db.User.Query().Order(userent.ByID()).FirstID(ctx)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return uuid.Nil, err
	}
	created, err := data.db.User.Create().
		SetUsername("hd-replay-" + uuid.NewString()[:12]).
		SetDisplayName("迁移重放撤销人").
		SetPasswordHash("replay-not-a-login").
		Save(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

// TestOrderFeeStatusHardDeleteMigrationReplay 重放 20260926100000 迁移：正常费用与
// 历史账单快照保留、CANCELLED 费用被归档审计后物理删除、来源外键置空、列与约束
// 删除；跳过不算通过，必须以 RONCIN_INTEGRATION_DATABASE_SOURCE 提供真实 PostgreSQL。
func TestOrderFeeStatusHardDeleteMigrationReplay(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	t.Cleanup(cleanupData)
	fixture := newFinanceBillPostgresFixture(t, data)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	keptFeeID := fixture.createUnbilledFee("hd-kept")
	cancelledFeeID := fixture.createUnbilledFee("hd-cancelled")

	// 已取消账单 + 非活动历史行：费用删除后历史行保留快照且来源置空。
	historicalBillID := createReplayBill(t, data, ctx, fixture, cancelledFeeID, financebillent.StatusCANCELLED, false)

	restoreOrderFeeStatusColumns(t, data, ctx, []uuid.UUID{cancelledFeeID})
	if _, err := data.sqlDB.ExecContext(ctx, readMigrationContent(t,
		"20260926100000_order_fee_status_hard_delete.sql")); err != nil {
		t.Fatalf("重放迁移失败: %v", err)
	}

	// 正常费用保留、已作废费用被物理删除。
	if _, err := data.db.OrderFee.Get(ctx, keptFeeID); err != nil {
		t.Fatalf("正常费用不得被删除: %v", err)
	}
	if _, err := data.db.OrderFee.Get(ctx, cancelledFeeID); err == nil || !ent.IsNotFound(err) {
		t.Fatalf("已作废费用应被物理删除: %v", err)
	}

	// 历史账单行保留快照且来源外键置空。
	line, lineErr := data.db.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(historicalBillID)).Only(ctx)
	if lineErr != nil {
		t.Fatalf("读取历史账单行: %v", lineErr)
	}
	if line.Active || line.OrderFeeID != uuid.Nil || line.FeeCode != "HD-FEE" || line.TotalAmount != "100.00000000" {
		t.Fatalf("历史账单行应保留快照且来源置空: %+v", line)
	}

	// 迁移归档审计可追溯：组织来自 LEFT JOIN orders（order_fees 无 organization_id
	// 列）；order_fees.order_id 具有非空外键，孤儿费用在迁移链内不可能存在，
	// 因此审计行的 organization_id 必须命中夹具组织。
	archivedAudit, auditErr := data.db.AuditLog.Query().
		Where(auditlogent.ActionEQ("order.fee.migration_hard_delete"), auditlogent.ResourceIDEQ(cancelledFeeID.String())).
		Only(ctx)
	if auditErr != nil {
		t.Fatalf("迁移归档审计缺失: %v", auditErr)
	}
	if archivedAudit.OrganizationID == nil || *archivedAudit.OrganizationID != fixture.organizationID {
		t.Fatalf("归档审计组织归属不符: %+v", archivedAudit.OrganizationID)
	}

	// 状态列已删除。
	for _, column := range []string{"status", "cancelled_at", "cancelled_by", "cancellation_reason"} {
		var columnCount int
		if err := data.sqlDB.QueryRowContext(ctx,
			`SELECT count(*) FROM information_schema.columns
			 WHERE table_schema = current_schema() AND table_name = 'order_fees' AND column_name = $1`, column).Scan(&columnCount); err != nil {
			t.Fatalf("读取列元数据失败: %v", err)
		}
		if columnCount != 0 {
			t.Fatalf("order_fees.%s 应已删除", column)
		}
	}
}

// TestOrderFeeStatusHardDeleteMigrationRejectsOccupiedCancelFee 校验异常事实使迁移
// 整体失败：CANCELLED 费用仍被有效账单关联占用时重放必须报错，且不得部分删除
// （单条 ExecContext 的多语句隐式事务整体回滚）。
func TestOrderFeeStatusHardDeleteMigrationRejectsOccupiedCancelFee(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	t.Cleanup(cleanupData)
	fixture := newFinanceBillPostgresFixture(t, data)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	occupiedID := fixture.createUnbilledFee("hd-occupied")
	freeID := fixture.createUnbilledFee("hd-occupied-free")

	// 占用场景：有效账单关联（活动账单行且所属账单未取消）仍在，重放必须整体失败。
	createReplayBill(t, data, ctx, fixture, occupiedID, financebillent.StatusDRAFT, true)

	restoreOrderFeeStatusColumns(t, data, ctx, []uuid.UUID{occupiedID})
	replayErr := func() error {
		_, err := data.sqlDB.ExecContext(ctx, readMigrationContent(t,
			"20260926100000_order_fee_status_hard_delete.sql"))
		return err
	}()
	if replayErr == nil || !strings.Contains(replayErr.Error(), "有效账单关联") {
		t.Fatalf("占用中的已作废费用必须使迁移失败，实际: %v", replayErr)
	}

	// 整体回滚：未被删除的 CANCELLED 行与状态列仍在。
	var cancelledCount int
	if err := data.sqlDB.QueryRowContext(ctx,
		`SELECT count(*) FROM "order_fees" WHERE "status" = 'CANCELLED'`).Scan(&cancelledCount); err != nil {
		t.Fatalf("读取回滚后状态: %v", err)
	}
	if cancelledCount != 1 {
		t.Fatalf("迁移失败必须整体回滚：CANCELLED 行数 = %d，期望 1", cancelledCount)
	}
	if _, err := data.db.OrderFee.Get(ctx, freeID); err != nil {
		t.Fatalf("正常费用不得被部分删除: %v", err)
	}
}
