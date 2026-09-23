package data

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// TestOrderFeeStatusUnbilledMigrationReplay 恢复迁移前状态（旧约束含 DRAFT/CONFIRMED、
// 旧默认值与旧行状态）后重放正式迁移文件，断言其真实改写效果：存量草稿与已确认
// 费用合并为 UNBILLED、默认值改为 UNBILLED、旧值写入被收紧后的约束拒绝。
// 跳过不算通过，必须以 RONCIN_INTEGRATION_DATABASE_SOURCE 提供真实 PostgreSQL。
func TestOrderFeeStatusUnbilledMigrationReplay(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	// 先注册数据清理（LIFO 后执行），保证夹具清理先于关闭数据库连接。
	t.Cleanup(cleanupData)
	fixture := newFinanceBillPostgresFixture(t, data)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	draftID := fixture.createUnbilledFee("mig-draft")
	confirmedID := fixture.createUnbilledFee("mig-confirmed")

	// 恢复迁移前状态：旧状态域（含 DRAFT/CONFIRMED）、旧默认值与旧存量行。
	// 夹具创建的费用默认 UNBILLED，必须先归位旧行状态再回加旧约束；
	// 语句不带参数走简单协议可多语句执行。
	restore := fmt.Sprintf(`
		ALTER TABLE "order_fees" DROP CONSTRAINT "order_fees_status_check";
		UPDATE "order_fees" SET "status" = 'DRAFT' WHERE "status" = 'UNBILLED';
		UPDATE "order_fees" SET "status" = 'CONFIRMED' WHERE "id" = '%s';
		ALTER TABLE "order_fees"
		  ADD CONSTRAINT "order_fees_status_check"
		  CHECK ("status" IN ('DRAFT', 'CONFIRMED', 'BILLED', 'CANCELLED'));
		ALTER TABLE "order_fees" ALTER COLUMN "status" SET DEFAULT 'DRAFT';`, confirmedID)
	if _, err := data.sqlDB.ExecContext(ctx, restore); err != nil {
		t.Fatalf("恢复迁移前状态失败: %v", err)
	}

	// 重放真实迁移文件：读取迁移内容在已恢复的迁移前状态上执行，断言其真实
	// 改写效果。不走 migration.Apply——隔离库已应用完整迁移链，框架的线性与
	// 完整性守卫会拦截定点补放；本用例验证的是迁移 DDL/DML 本身。
	content, err := os.ReadFile(filepath.Join("..", "..", "migrations",
		"20260923090000_order_fee_status_unbilled.sql"))
	if err != nil {
		t.Fatalf("读取迁移文件失败: %v", err)
	}
	if _, err := data.sqlDB.ExecContext(ctx, string(content)); err != nil {
		t.Fatalf("重放迁移失败: %v", err)
	}

	for _, id := range []uuid.UUID{draftID, confirmedID} {
		fee, feeErr := data.db.OrderFee.Get(ctx, id)
		if feeErr != nil || fee.Status != orderfeeent.StatusUNBILLED {
			t.Fatalf("迁移后存量费用 %s 状态 = %#v，期望 UNBILLED (err=%v)", id, fee, feeErr)
		}
	}

	var columnDefault string
	if err := data.sqlDB.QueryRowContext(ctx,
		`SELECT column_default FROM information_schema.columns
		 WHERE table_schema = current_schema() AND table_name = 'order_fees' AND column_name = 'status'`).Scan(&columnDefault); err != nil {
		t.Fatalf("读取状态列默认值失败: %v", err)
	}
	if !strings.Contains(columnDefault, "UNBILLED") {
		t.Fatalf("状态列默认值 = %q，期望 UNBILLED", columnDefault)
	}

	// 收紧后的约束必须拒绝退役状态写入。
	if _, err := data.sqlDB.ExecContext(ctx,
		`UPDATE "order_fees" SET "status" = 'DRAFT' WHERE "id" = $1`, draftID); err == nil || !strings.Contains(err.Error(), "order_fees_status_check") {
		t.Fatalf("迁移后写入退役状态 DRAFT 应被约束拒绝，实际: %v", err)
	}
	if _, err := data.sqlDB.ExecContext(ctx,
		`UPDATE "order_fees" SET "status" = 'CONFIRMED' WHERE "id" = $1`, confirmedID); err == nil || !strings.Contains(err.Error(), "order_fees_status_check") {
		t.Fatalf("迁移后写入退役状态 CONFIRMED 应被约束拒绝，实际: %v", err)
	}
}
