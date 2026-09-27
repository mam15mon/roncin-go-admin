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
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
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

// replaySupplementChain 汇总迁移重放所需真实补录链路各实体的 ID。
type replaySupplementChain struct {
	requestID    uuid.UUID
	feeID        uuid.UUID
	adjustmentID uuid.UUID
	commissionID uuid.UUID
	ruleID       uuid.UUID
}

// cleanupReplaySupplementChain 按依赖顺序删除直构的补录链路行：申请与调整的
// NO ACTION 外键会阻断账单夹具自带的订单/组织清理（清理按 LIFO 先于夹具执行），
// 成功场景中费用已被迁移删除，删除空集不算失败。
func cleanupReplaySupplementChain(t *testing.T, data *Data, chain replaySupplementChain) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		steps := []struct {
			name string
			run  func() error
		}{
			{name: "冲减调整", run: func() error {
				_, err := data.db.FinanceCommissionAdjustment.Delete().
					Where(commissionadjustmentent.IDEQ(chain.adjustmentID)).Exec(cleanupCtx)
				return err
			}},
			{name: "补录费用", run: func() error {
				_, err := data.db.OrderFee.Delete().Where(orderfeeent.IDEQ(chain.feeID)).Exec(cleanupCtx)
				return err
			}},
			{name: "补录申请", run: func() error {
				_, err := data.db.OrderFeeSupplementRequest.Delete().
					Where(orderfeesupplementent.IDEQ(chain.requestID)).Exec(cleanupCtx)
				return err
			}},
			{name: "提成父单", run: func() error {
				_, err := data.db.FinanceCommission.Delete().
					Where(commissionent.IDEQ(chain.commissionID)).Exec(cleanupCtx)
				return err
			}},
			{name: "提成规则", run: func() error {
				_, err := data.db.FinanceCommissionRule.Delete().
					Where(commissionruleent.IDEQ(chain.ruleID)).Exec(cleanupCtx)
				return err
			}},
		}
		for _, step := range steps {
			if err := step.run(); err != nil {
				t.Errorf("清理迁移重放%s: %v", step.name, err)
			}
		}
	})
}

// createReplaySupplementChain 在迁移前状态上直构真实补录链路数据：APPROVED 补录
// 申请、其生成的应付费用（supplement_request_id 反向关联，幂等键沿用业务链路
// 派生规则）与 LOCKED_FEE_SUPPLEMENT 来源冲减调整（source_fee_supplement_request_id
// 关联申请）。调整的状态与确认/扣回/取消时间戳按场景参数写入，用于命中迁移守卫
// 的不同分支；调整所需的提成规则与父单按快照 CHECK 直构，不经过提成计算链路，
// 避免引入与迁移重放无关的依赖（沿用 createReplayBill 的直构原则）。
func createReplaySupplementChain(t *testing.T, data *Data, ctx context.Context, fixture *financeBillPostgresFixture,
	status commissionadjustmentent.Status, confirmedAt, paidAt, cancelledAt bool) replaySupplementChain {
	t.Helper()
	actorID, actorErr := ensureReplayCancelActor(t, data, ctx)
	if actorErr != nil {
		t.Fatalf("准备补录链路操作者: %v", actorErr)
	}
	decidedAt := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	request, requestErr := data.db.OrderFeeSupplementRequest.Create().
		SetOrganizationID(fixture.organizationID).
		SetOrderID(fixture.orderID).
		SetLockBasis(orderfeesupplementent.LockBasisBUSINESS).
		SetBusinessLockGeneration(1).
		SetIdempotencyKey("hd-supp-" + fixture.suffix).
		SetRequestFingerprint("hd-supp-fingerprint-" + fixture.suffix).
		SetDirection(orderfeesupplementent.DirectionPAYABLE).
		SetFeeCode("HD_SUPPLEMENT").
		SetFeeName("迁移重放补录费用").
		SetSettlementPartyID(fixture.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeesupplementent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("100.00000000").
		SetExpenseDate(financeBillIntegrationDate).
		SetReason("迁移重放测试补录").
		SetRequestedBy(actorID).
		SetRequestedAt(decidedAt).
		SetStatus(orderfeesupplementent.StatusAPPROVED).
		SetVersion(2).
		SetDecidedBy(actorID).
		SetDecidedAt(decidedAt).
		Save(ctx)
	if requestErr != nil {
		t.Fatalf("创建迁移重放补录申请: %v", requestErr)
	}

	fee, feeErr := data.db.OrderFee.Create().
		SetOrderID(fixture.orderID).
		SetSupplementRequestID(request.ID).
		SetIdempotencyKey("fee-supplement:" + request.ID.String()).
		SetDirection(orderfeeent.DirectionPAYABLE).
		SetFeeCode("HD_SUPPLEMENT").
		SetFeeName("迁移重放补录费用").
		SetSettlementPartyID(fixture.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("100.00000000").
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if feeErr != nil {
		t.Fatalf("创建迁移重放补录生成费用: %v", feeErr)
	}

	// 提成父单的规则快照 CHECK 要求 rule_id/rule_name/personnel_role/
	// calculation_basis 四元组完整，先直构规则行再挂快照。
	rule, ruleErr := data.db.FinanceCommissionRule.Create().
		SetOrganizationID(fixture.organizationID).
		SetName("迁移重放规则-" + fixture.suffix).
		SetPersonnelRole(commissionruleent.PersonnelRoleSALES).
		SetCalculationBasis(commissionruleent.CalculationBasisREALIZED_PROFIT).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(ctx)
	if ruleErr != nil {
		t.Fatalf("创建迁移重放提成规则: %v", ruleErr)
	}
	commission, commissionErr := data.db.FinanceCommission.Create().
		SetOrganizationID(fixture.organizationID).
		SetCommissionNo("FC-HD-" + fixture.suffix).
		SetIdempotencyKey("hd-commission-" + fixture.suffix).
		SetEmployeeID(actorID).
		SetEmployeeName("迁移重放提成员工").
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName("迁移重放规则-" + fixture.suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis("REALIZED_PROFIT").
		SetStatus(commissionent.StatusCONFIRMED).
		SetBaseCurrency("CNY").
		SetRealizedRevenue("1000.00000000").
		SetAllocatedCost("400.00000000").
		SetRealizedProfit("1000.00000000").
		SetCommissionBaseAmount("1000.00000000").
		SetRatePercent("10.0000").
		SetCommissionAmount("60.00000000").
		SetCommissionDate(financeBillIntegrationDate).
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource(commissionent.CnyExchangeRateSourceBASE_CURRENCY).
		SetCnyExchangeRateDate(financeBillIntegrationDate).
		SetCnyCommissionAmount("60.00000000").
		SetVersion(1).
		Save(ctx)
	if commissionErr != nil {
		t.Fatalf("创建迁移重放提成父单: %v", commissionErr)
	}

	// 状态与时间戳组合说明：DRAFT/CONFIRMED/PAID 命中「未取消」分支；CANCELLED
	// 且仅带 confirmed_at / paid_at 分别独立命中「曾确认」「曾扣回」分支（后者
	// 在通用取消门禁下当前不可经业务流转产生，但迁移守卫保护数据库层不变量，
	// 必须对任意存量状态失败关闭）。
	adjustmentCreate := data.db.FinanceCommissionAdjustment.Create().
		SetOrganizationID(fixture.organizationID).
		SetCommissionID(commission.ID).
		SetOrderID(fixture.orderID).
		SetAdjustmentNo(commission.CommissionNo + "-ADJ001").
		SetIdempotencyKey("hd-adj-" + fixture.suffix).
		SetCommissionNo(commission.CommissionNo).
		SetOrderNo("HD-SUPP-" + fixture.suffix).
		SetEmployeeID(actorID).
		SetEmployeeName("迁移重放提成员工").
		SetSourceType(commissionadjustmentent.SourceTypeLOCKED_FEE_SUPPLEMENT).
		SetSourceFeeSupplementRequestID(request.ID).
		SetDirection(commissionadjustmentent.DirectionDECREASE).
		SetStatus(status).
		SetBaseCurrency("CNY").
		SetAmount("10.00000000").
		SetReason("迁移重放测试冲减建议").
		SetVersion(1)
	stampAt := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	if confirmedAt {
		adjustmentCreate = adjustmentCreate.SetConfirmedAt(stampAt).SetConfirmedBy(actorID)
	}
	if paidAt {
		adjustmentCreate = adjustmentCreate.SetPaidAt(stampAt).SetPaidBy(actorID)
	}
	if cancelledAt {
		adjustmentCreate = adjustmentCreate.SetCancelledAt(stampAt).SetCancelledBy(actorID).
			SetCancellationReason("迁移重放测试取消建议")
	}
	adjustment, adjustmentErr := adjustmentCreate.Save(ctx)
	if adjustmentErr != nil {
		t.Fatalf("创建迁移重放关联冲减调整: %v", adjustmentErr)
	}
	chain := replaySupplementChain{
		requestID:    request.ID,
		feeID:        fee.ID,
		adjustmentID: adjustment.ID,
		commissionID: commission.ID,
		ruleID:       rule.ID,
	}
	cleanupReplaySupplementChain(t, data, chain)
	return chain
}

// TestOrderFeeStatusHardDeleteMigrationAdjustmentGuardMatrix 以真实补录链路夹具
// （APPROVED 申请 + 生成费用置 CANCELLED + LOCKED_FEE_SUPPLEMENT 关联冲减调整）
// 表驱动校验守卫拒绝矩阵：任何未取消调整（DRAFT/CONFIRMED/PAID）与已取消但保留
// 确认/扣回历史的调整都使迁移整体失败回滚，费用、调整、申请与列结构无部分变更。
// 每个场景在独立隔离 Schema 内执行；跳过不算通过。
func TestOrderFeeStatusHardDeleteMigrationAdjustmentGuardMatrix(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	cases := []struct {
		name        string
		status      commissionadjustmentent.Status
		confirmedAt bool
		paidAt      bool
		cancelledAt bool
	}{
		{name: "DRAFT 待处理调整拒绝迁移", status: commissionadjustmentent.StatusDRAFT},
		{name: "CONFIRMED 调整拒绝迁移", status: commissionadjustmentent.StatusCONFIRMED, confirmedAt: true},
		{name: "PAID 调整拒绝迁移", status: commissionadjustmentent.StatusPAID, confirmedAt: true, paidAt: true},
		{name: "已取消但曾确认的调整拒绝迁移", status: commissionadjustmentent.StatusCANCELLED, confirmedAt: true, cancelledAt: true},
		{name: "已取消但曾扣回的调整拒绝迁移", status: commissionadjustmentent.StatusCANCELLED, paidAt: true, cancelledAt: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			data, cleanupData := getIntegrationData(t)
			t.Cleanup(cleanupData)
			fixture := newFinanceBillPostgresFixture(t, data)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			chain := createReplaySupplementChain(t, data, ctx, fixture,
				testCase.status, testCase.confirmedAt, testCase.paidAt, testCase.cancelledAt)

			restoreOrderFeeStatusColumns(t, data, ctx, []uuid.UUID{chain.feeID})
			_, replayErr := data.sqlDB.ExecContext(ctx, readMigrationContent(t,
				"20260926100000_order_fee_status_hard_delete.sql"))
			if replayErr == nil || !strings.Contains(replayErr.Error(), "需先处置待处理调整") {
				t.Fatalf("未处置关联冲减调整必须使迁移失败，实际: %v", replayErr)
			}

			// 原子回滚断言：费用仍在且保留补录来源，调整与申请原样，无新增归档
			// 审计，旧状态列仍存在。
			fee, feeErr := data.db.OrderFee.Get(ctx, chain.feeID)
			if feeErr != nil {
				t.Fatalf("迁移失败必须整体回滚，已作废补录费用不得被删除: %v", feeErr)
			}
			if fee.SupplementRequestID == nil || *fee.SupplementRequestID != chain.requestID {
				t.Fatalf("回滚后费用补录来源关联不得变化: %v", fee.SupplementRequestID)
			}
			adjustment, adjustmentGetErr := data.db.FinanceCommissionAdjustment.Get(ctx, chain.adjustmentID)
			if adjustmentGetErr != nil {
				t.Fatalf("迁移失败不得改动关联调整: %v", adjustmentGetErr)
			}
			if adjustment.Status != testCase.status || adjustment.Version != 1 {
				t.Fatalf("回滚后调整状态与版本必须原样: %+v", adjustment)
			}
			if (adjustment.ConfirmedAt != nil) != testCase.confirmedAt || (adjustment.PaidAt != nil) != testCase.paidAt {
				t.Fatalf("回滚后调整确认/扣回时间戳必须原样: confirmed=%v paid=%v", adjustment.ConfirmedAt, adjustment.PaidAt)
			}
			request, requestErr := data.db.OrderFeeSupplementRequest.Get(ctx, chain.requestID)
			if requestErr != nil || request.Status != orderfeesupplementent.StatusAPPROVED {
				t.Fatalf("迁移失败不得改动补录申请: %v %+v", requestErr, request)
			}
			auditCount, auditErr := data.db.AuditLog.Query().
				Where(auditlogent.ActionEQ("order.fee.migration_hard_delete")).Count(ctx)
			if auditErr != nil || auditCount != 0 {
				t.Fatalf("迁移失败不得写入归档审计: count=%d err=%v", auditCount, auditErr)
			}
			var columnCount int
			if err := data.sqlDB.QueryRowContext(ctx,
				`SELECT count(*) FROM information_schema.columns
				 WHERE table_schema = current_schema() AND table_name = 'order_fees' AND column_name = 'status'`).Scan(&columnCount); err != nil {
				t.Fatalf("读取列元数据失败: %v", err)
			}
			if columnCount != 1 {
				t.Fatalf("迁移失败必须整体回滚，order_fees.status 应仍存在: %d", columnCount)
			}
		})
	}
}

// TestOrderFeeStatusHardDeleteMigrationAdjustmentResolvedSuccess 校验成功矩阵：
// 关联调整已全部取消且从未确认/扣回时迁移放行，申请与调整保留、费用删除并写入
// 迁移归档审计；无任何 CANCELLED 行时迁移正常完成且零审计写入（无关联调整的
// 普通 CANCELLED 费用路径由 TestOrderFeeStatusHardDeleteMigrationReplay 覆盖）。
// 两个场景分别在独立隔离 Schema 内执行；跳过不算通过。
func TestOrderFeeStatusHardDeleteMigrationAdjustmentResolvedSuccess(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}

	t.Run("仅关联已取消且无确认扣回历史的调整时迁移成功", func(t *testing.T) {
		data, cleanupData := getIntegrationData(t)
		t.Cleanup(cleanupData)
		fixture := newFinanceBillPostgresFixture(t, data)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		chain := createReplaySupplementChain(t, data, ctx, fixture,
			commissionadjustmentent.StatusCANCELLED, false, false, true)

		restoreOrderFeeStatusColumns(t, data, ctx, []uuid.UUID{chain.feeID})
		if _, err := data.sqlDB.ExecContext(ctx, readMigrationContent(t,
			"20260926100000_order_fee_status_hard_delete.sql")); err != nil {
			t.Fatalf("已处置关联调整时迁移应成功: %v", err)
		}

		// 申请与历史调整保留，费用被归档审计后物理删除。
		if _, err := data.db.OrderFeeSupplementRequest.Get(ctx, chain.requestID); err != nil {
			t.Fatalf("补录申请必须保留: %v", err)
		}
		adjustment, adjustmentErr := data.db.FinanceCommissionAdjustment.Get(ctx, chain.adjustmentID)
		if adjustmentErr != nil || adjustment.Status != commissionadjustmentent.StatusCANCELLED ||
			adjustment.ConfirmedAt != nil || adjustment.PaidAt != nil {
			t.Fatalf("已取消调整必须原样保留: %v %+v", adjustmentErr, adjustment)
		}
		if _, err := data.db.OrderFee.Get(ctx, chain.feeID); err == nil || !ent.IsNotFound(err) {
			t.Fatalf("已作废补录费用应被物理删除: %v", err)
		}
		archivedAudit, auditErr := data.db.AuditLog.Query().
			Where(auditlogent.ActionEQ("order.fee.migration_hard_delete"), auditlogent.ResourceIDEQ(chain.feeID.String())).
			Only(ctx)
		if auditErr != nil {
			t.Fatalf("迁移归档审计缺失: %v", auditErr)
		}
		if archivedAudit.OrganizationID == nil || *archivedAudit.OrganizationID != fixture.organizationID {
			t.Fatalf("归档审计组织归属不符: %+v", archivedAudit.OrganizationID)
		}
	})

	t.Run("无任何已作废费用时迁移正常完成", func(t *testing.T) {
		data, cleanupData := getIntegrationData(t)
		t.Cleanup(cleanupData)
		fixture := newFinanceBillPostgresFixture(t, data)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		keptFeeID := fixture.createUnbilledFee("hd-empty-kept")

		restoreOrderFeeStatusColumns(t, data, ctx, nil)
		if _, err := data.sqlDB.ExecContext(ctx, readMigrationContent(t,
			"20260926100000_order_fee_status_hard_delete.sql")); err != nil {
			t.Fatalf("无 CANCELLED 行时迁移应正常完成: %v", err)
		}

		if _, err := data.db.OrderFee.Get(ctx, keptFeeID); err != nil {
			t.Fatalf("正常费用不得被删除: %v", err)
		}
		auditCount, auditErr := data.db.AuditLog.Query().
			Where(auditlogent.ActionEQ("order.fee.migration_hard_delete")).Count(ctx)
		if auditErr != nil || auditCount != 0 {
			t.Fatalf("零删除时不得写入归档审计: count=%d err=%v", auditCount, auditErr)
		}
		var columnCount int
		if err := data.sqlDB.QueryRowContext(ctx,
			`SELECT count(*) FROM information_schema.columns
			 WHERE table_schema = current_schema() AND table_name = 'order_fees' AND column_name = 'status'`).Scan(&columnCount); err != nil {
			t.Fatalf("读取列元数据失败: %v", err)
		}
		if columnCount != 0 {
			t.Fatalf("无 CANCELLED 行时状态列仍应删除: %d", columnCount)
		}
	})
}
