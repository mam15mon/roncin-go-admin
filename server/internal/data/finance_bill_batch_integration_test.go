package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebillbatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillbatch"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// financeBillBatchPostgresFixture 在账单事务夹具之上补齐批量建账所需的操作用户、
// 批次编号规则和批次数据清理；基础组织、账户、订单与费用夹具直接复用。
type financeBillBatchPostgresFixture struct {
	*financeBillPostgresFixture
	actorUserID uuid.UUID
}

type financeBillBatchCreateResult struct {
	batch *biz.FinanceBillBatch
	err   error
}

func TestFinanceBillBatchCreatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	hasCNY, err := data.db.Currency.Query().Where(currencyent.CodeEQ("CNY"), currencyent.EnabledEQ(true)).Exist(context.Background())
	if err != nil {
		t.Fatalf("查询批量建账用例所需币种: %v", err)
	}
	if !hasCNY {
		if _, err := data.db.Currency.Create().SetCode("CNY").SetName("人民币").SetEnabled(true).Save(context.Background()); err != nil {
			t.Fatalf("创建批量建账用例所需币种: %v", err)
		}
	}

	t.Run("不同幂等键并发抢占同一组费用只有一个批次成功", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("race-first"), fixture.createBatchUnbilledFee("race-second")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		first := fixture.buildBatchInput(t, usecase, "batch-race-first-"+fixture.suffix, feeIDs)
		second := first
		second.IdempotencyKey = "batch-race-second-" + fixture.suffix

		results := createFinanceBillBatchesConcurrently(usecase, fixture.organizationID, fixture.actorUserID, first, second)
		successes, feeConflicts := 0, 0
		for _, result := range results {
			switch {
			case result.err == nil && result.batch != nil:
				successes++
			case errors.Is(result.err, biz.ErrFinanceBillFeeInvalid):
				feeConflicts++
			default:
				t.Fatalf("并发建批返回非预期结果: batch=%#v error=%v", result.batch, result.err)
			}
		}
		if successes != 1 || feeConflicts != 1 {
			t.Fatalf("并发建批结果 success=%d feeConflict=%d，期望各 1", successes, feeConflicts)
		}
		fixture.requireCommittedBatchState(feeIDs, 1, 1)
	})

	t.Run("陈旧预览令牌返回冲突且批次账单零写入", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("stale-token")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-stale-"+fixture.suffix, feeIDs)

		// 模拟预览签发令牌后费用事实被修改（版本与金额变化）。
		if _, err := data.db.OrderFee.UpdateOneID(feeIDs[0]).
			SetTotalAmount("200.00000000").
			SetNetAmount("200.00000000").
			SetBaseCurrencyAmount("200.00000000").
			AddVersion(1).
			Save(context.Background()); err != nil {
			t.Fatalf("修改费用事实失败: %v", err)
		}

		_, err = usecase.CreateBatch(context.Background(), fixture.organizationID, fixture.actorUserID, input)
		if !errors.Is(err, biz.ErrFinanceBillPreviewStale) {
			t.Fatalf("陈旧令牌建批错误 = %v，期望 %v", err, biz.ErrFinanceBillPreviewStale)
		}
		fixture.requireRolledBackBatchState(feeIDs[0])
	})

	t.Run("相同幂等键并发重放返回同一批次且无重复账单", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("replay-first"), fixture.createBatchUnbilledFee("replay-second")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-replay-"+fixture.suffix, feeIDs)

		results := createFinanceBillBatchesConcurrently(usecase, fixture.organizationID, fixture.actorUserID, input, input)
		for index, result := range results {
			if result.err != nil {
				t.Fatalf("第 %d 个并发重放请求失败: %v", index+1, result.err)
			}
			if result.batch == nil {
				t.Fatalf("第 %d 个并发重放请求未返回批次", index+1)
			}
		}
		if results[0].batch.ID != results[1].batch.ID || results[0].batch.BatchNo != results[1].batch.BatchNo {
			t.Fatalf("相同幂等键返回了不同批次: first=%s/%s second=%s/%s", results[0].batch.ID, results[0].batch.BatchNo, results[1].batch.ID, results[1].batch.BatchNo)
		}
		fixture.requireCommittedBatchState(feeIDs, 1, 1)
	})
}

func newFinanceBillBatchPostgresFixture(t *testing.T, data *Data) *financeBillBatchPostgresFixture {
	t.Helper()
	base := newFinanceBillPostgresFixture(t, data)
	fixture := &financeBillBatchPostgresFixture{financeBillPostgresFixture: base}
	ctx := context.Background()
	actor, err := data.db.User.Create().
		SetDisplayName("批量建账集成测试用户-" + base.suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建批量建账操作用户: %v", err)
	}
	fixture.actorUserID = actor.ID
	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(base.organizationID).
		SetDocumentType(numberruleent.DocumentTypeBillBatch).
		SetPrefix("BATCH-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建批量建账批次编号规则: %v", err)
	}
	// 该清理注册晚于基础夹具，LIFO 下先于基础清理执行：账单行、账单删除后才能删批次。
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		steps := []struct {
			name string
			run  func() error
		}{
			{name: "批量账单明细", run: func() error {
				_, err := data.db.FinanceBillLine.Delete().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(base.organizationID))).Exec(cleanupCtx)
				return err
			}},
			{name: "批量账单", run: func() error {
				_, err := data.db.FinanceBill.Delete().Where(financebillent.OrganizationIDEQ(base.organizationID)).Exec(cleanupCtx)
				return err
			}},
			{name: "账单批次", run: func() error {
				_, err := data.db.FinanceBillBatch.Delete().Where(financebillbatchent.OrganizationIDEQ(base.organizationID)).Exec(cleanupCtx)
				return err
			}},
			{name: "操作用户", run: func() error {
				return data.db.User.DeleteOneID(actor.ID).Exec(cleanupCtx)
			}},
		}
		for _, step := range steps {
			if err := step.run(); err != nil {
				t.Errorf("清理批量建账测试%s: %v", step.name, err)
			}
		}
	})
	return fixture
}

// createBatchUnbilledFee 创建带税率（分组必填）的未建账应收费用；批量分组路径要求税率非空。
func (f *financeBillBatchPostgresFixture) createBatchUnbilledFee(key string) uuid.UUID {
	f.t.Helper()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("batch-fee-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusUNBILLED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetTaxRate("0.00").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("100.00000000").
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建批量建账测试费用: %v", err)
	}
	return fee.ID
}

// buildBatchInput 复现前端两段式流程：先无配置预览拿叶子 groupKey，再带日期和账户
// 预览签发创建令牌，最后组装与令牌一致的批量创建输入。
func (f *financeBillBatchPostgresFixture) buildBatchInput(t *testing.T, usecase *biz.FinanceBillUsecase, idempotencyKey string, feeIDs []uuid.UUID) biz.CreateFinanceBillBatchInput {
	t.Helper()
	ctx := context.Background()
	policy := biz.FinanceBillGroupingPolicy{Mode: "NORMAL", SplitByOrder: true}
	initial, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy})
	if err != nil {
		t.Fatalf("无配置批量预览失败: %v", err)
	}
	if initial.PreviewToken != "" || len(initial.Groups) == 0 {
		t.Fatalf("无日期账户的预览不应签发令牌: token=%q groups=%d", initial.PreviewToken, len(initial.Groups))
	}
	configs := make([]biz.FinanceBillBatchPreviewGroupConfig, 0, len(initial.Groups))
	for _, group := range initial.Groups {
		configs = append(configs, biz.FinanceBillBatchPreviewGroupConfig{
			GroupKey: group.GroupKey, BillDate: financeBillIntegrationDate, SettlementAccountID: f.accountID,
		})
	}
	preview, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy, GroupConfigs: configs})
	if err != nil {
		t.Fatalf("完整配置批量预览失败: %v", err)
	}
	if preview.PreviewToken == "" {
		t.Fatal("完整配置预览未签发创建令牌")
	}
	groups := make([]biz.CreateFinanceBillBatchGroupInput, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		groups = append(groups, biz.CreateFinanceBillBatchGroupInput{
			GroupKey: group.GroupKey, StatementTitle: group.SettlementPartyName, BillDate: financeBillIntegrationDate, SettlementAccountID: f.accountID,
		})
	}
	return biz.CreateFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy, Groups: groups, PreviewToken: preview.PreviewToken, IdempotencyKey: idempotencyKey}
}

func createFinanceBillBatchesConcurrently(usecase *biz.FinanceBillUsecase, organizationID, actorID uuid.UUID, inputs ...biz.CreateFinanceBillBatchInput) []financeBillBatchCreateResult {
	start := make(chan struct{})
	results := make(chan financeBillBatchCreateResult, len(inputs))
	for _, input := range inputs {
		input := input
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			batch, err := usecase.CreateBatch(ctx, organizationID, actorID, input)
			results <- financeBillBatchCreateResult{batch: batch, err: err}
		}()
	}
	close(start)
	collected := make([]financeBillBatchCreateResult, 0, len(inputs))
	for range inputs {
		collected = append(collected, <-results)
	}
	return collected
}

func (f *financeBillBatchPostgresFixture) requireCommittedBatchState(feeIDs []uuid.UUID, wantBatches, wantBills int) {
	f.t.Helper()
	ctx := context.Background()
	batchCount, err := f.data.db.FinanceBillBatch.Query().Where(financebillbatchent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || batchCount != wantBatches {
		f.t.Fatalf("已提交账单批次数 = %d，期望 %d，error=%v", batchCount, wantBatches, err)
	}
	billCount, err := f.data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || billCount != wantBills {
		f.t.Fatalf("已提交批量账单数 = %d，期望 %d，error=%v", billCount, wantBills, err)
	}
	lineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID)), financebilllineent.ActiveEQ(true)).Count(ctx)
	if err != nil || lineCount != len(feeIDs) {
		f.t.Fatalf("已提交批量账单明细数 = %d，期望 %d，error=%v", lineCount, len(feeIDs), err)
	}
	for _, feeID := range feeIDs {
		feeLines, countErr := f.data.db.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(feeID), financebilllineent.ActiveEQ(true)).Count(ctx)
		if countErr != nil || feeLines != 1 {
			f.t.Fatalf("费用 %s 的有效账单行数 = %d，期望 1，error=%v", feeID, feeLines, countErr)
		}
		fee, getErr := f.data.db.OrderFee.Get(ctx, feeID)
		if getErr != nil || fee.Status != orderfeeent.StatusBILLED || fee.Version != 2 {
			f.t.Fatalf("批量建账后费用状态 = %#v，期望 BILLED/version 2，error=%v", fee, getErr)
		}
	}
}

func (f *financeBillBatchPostgresFixture) requireRolledBackBatchState(feeID uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	batchCount, err := f.data.db.FinanceBillBatch.Query().Where(financebillbatchent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || batchCount != 0 {
		f.t.Fatalf("零写入校验：账单批次数 = %d，期望 0，error=%v", batchCount, err)
	}
	billCount, err := f.data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || billCount != 0 {
		f.t.Fatalf("零写入校验：账单数 = %d，期望 0，error=%v", billCount, err)
	}
	lineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lineCount != 0 {
		f.t.Fatalf("零写入校验：账单明细数 = %d，期望 0，error=%v", lineCount, err)
	}
	fee, err := f.data.db.OrderFee.Get(ctx, feeID)
	if err != nil || fee.Status != orderfeeent.StatusUNBILLED {
		f.t.Fatalf("零写入校验：费用状态 = %#v，期望 UNBILLED（未被部分改成 BILLED），error=%v", fee, err)
	}
}

// createBatchUnbilledPayableFee 创建同结算单位、同币种、同税率的未建账应付费用，
// 与 createBatchUnbilledFee 组成对冲建账所需的双向费用事实。
func (f *financeBillBatchPostgresFixture) createBatchUnbilledPayableFee(key string) uuid.UUID {
	f.t.Helper()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("batch-payable-fee-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.DirectionPAYABLE).
		SetStatus(orderfeeent.StatusUNBILLED).
		SetFeeCode("AGENT_FREIGHT").
		SetFeeName("代理费").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("40.0000").
		SetTotalAmount("40.00000000").
		SetNetAmount("40.00000000").
		SetTaxAmount("0.00000000").
		SetTaxRate("0.00").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("40.00000000").
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建对冲建账测试应付费用: %v", err)
	}
	return fee.ID
}

// cleanupNettingRows 在批次夹具清理前删除对冲分摊与对冲单；对冲分摊引用账单（NO ACTION）。
func (f *financeBillBatchPostgresFixture) cleanupNettingRows() {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := f.data.db.FinanceNettingAllocation.Delete().Where(financenettingallocationent.HasNettingWith(financenettingent.OrganizationIDEQ(f.organizationID))).Exec(cleanupCtx); err != nil {
		f.t.Errorf("清理对冲建账测试对冲分摊: %v", err)
	}
	if _, err := f.data.db.FinanceNetting.Delete().Where(financenettingent.OrganizationIDEQ(f.organizationID)).Exec(cleanupCtx); err != nil {
		f.t.Errorf("清理对冲建账测试对冲单: %v", err)
	}
}

// buildNettingBatchInput 以对冲模式复现两段式预览并组装创建输入。
func (f *financeBillBatchPostgresFixture) buildNettingBatchInput(t *testing.T, usecase *biz.FinanceBillUsecase, idempotencyKey string, feeIDs []uuid.UUID) biz.CreateFinanceBillBatchInput {
	t.Helper()
	ctx := context.Background()
	policy := biz.FinanceBillGroupingPolicy{Mode: "NETTING", SplitByOrder: true}
	initial, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy})
	if err != nil {
		t.Fatalf("对冲模式无配置预览失败: %v", err)
	}
	if len(initial.NettingPairs) != 1 {
		t.Fatalf("对冲预览应返回 1 组抵销汇总: %#v", initial.NettingPairs)
	}
	configs := make([]biz.FinanceBillBatchPreviewGroupConfig, 0, len(initial.Groups))
	for _, group := range initial.Groups {
		configs = append(configs, biz.FinanceBillBatchPreviewGroupConfig{
			GroupKey: group.GroupKey, BillDate: financeBillIntegrationDate, SettlementAccountID: f.accountID,
		})
	}
	preview, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy, GroupConfigs: configs})
	if err != nil {
		t.Fatalf("对冲模式完整配置预览失败: %v", err)
	}
	if preview.PreviewToken == "" || len(preview.NettingPairs) != 1 || len(preview.Groups) != 2 {
		t.Fatalf("对冲预览结果不完整: token=%q groups=%d pairs=%d", preview.PreviewToken, len(preview.Groups), len(preview.NettingPairs))
	}
	if !preview.NettingPairs[0].OffsetAmount.Equal(decimal.RequireFromString("40")) {
		t.Fatalf("对冲抵销额应为 40: %#v", preview.NettingPairs[0])
	}
	groups := make([]biz.CreateFinanceBillBatchGroupInput, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		groups = append(groups, biz.CreateFinanceBillBatchGroupInput{
			GroupKey: group.GroupKey, StatementTitle: group.SettlementPartyName, BillDate: financeBillIntegrationDate, SettlementAccountID: f.accountID,
		})
	}
	return biz.CreateFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy, Groups: groups, PreviewToken: preview.PreviewToken, IdempotencyKey: idempotencyKey}
}

func TestFinanceBillBatchNettingCreatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	hasCNY, err := data.db.Currency.Query().Where(currencyent.CodeEQ("CNY"), currencyent.EnabledEQ(true)).Exist(context.Background())
	if err != nil {
		t.Fatalf("查询对冲建账用例所需币种: %v", err)
	}
	if !hasCNY {
		if _, err := data.db.Currency.Create().SetCode("CNY").SetName("人民币").SetEnabled(true).Save(context.Background()); err != nil {
			t.Fatalf("创建对冲建账用例所需币种: %v", err)
		}
	}

	t.Run("对冲批次原子生成双方账单与草稿对冲单并支持确认反转", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		if _, err := data.db.NumberRule.Create().SetOrganizationID(fixture.organizationID).SetDocumentType(numberruleent.DocumentTypeNetting).SetPrefix("NT-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(context.Background()); err != nil {
			t.Fatalf("创建测试对冲编号规则: %v", err)
		}
		t.Cleanup(fixture.cleanupNettingRows)
		receivableFeeID := fixture.createBatchUnbilledFee("netting-receivable")
		payableFeeID := fixture.createBatchUnbilledPayableFee("netting-payable")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildNettingBatchInput(t, usecase, "batch-netting-"+fixture.suffix, []uuid.UUID{receivableFeeID, payableFeeID})

		batch, err := usecase.CreateBatch(context.Background(), fixture.organizationID, fixture.actorUserID, input)
		if err != nil {
			t.Fatalf("对冲批次创建失败: %v", err)
		}
		if len(batch.Bills) != 2 || len(batch.Nettings) != 1 {
			t.Fatalf("对冲批次应生成两张账单与一张对冲单: bills=%d nettings=%d", len(batch.Bills), len(batch.Nettings))
		}
		netting := batch.Nettings[0]
		if netting.Status != biz.FinanceNettingDraft || !netting.Amount.Equal(decimal.RequireFromString("40")) || netting.BatchID == nil || *netting.BatchID != batch.ID {
			t.Fatalf("对冲单字段错误: %#v", netting)
		}
		if len(netting.Allocations) != 2 {
			t.Fatalf("对冲分摊应覆盖双方账单: %#v", netting.Allocations)
		}
		for _, allocation := range netting.Allocations {
			if allocation.Active {
				t.Fatalf("批次对冲分摊初始必须未生效: %#v", allocation)
			}
		}

		expectedVersions := make(map[uuid.UUID]uint64, len(batch.Bills))
		directions := map[biz.OrderFeeDirection]uuid.UUID{}
		for _, bill := range batch.Bills {
			expectedVersions[bill.ID] = bill.Version
			directions[bill.Direction] = bill.ID
		}
		confirmedBatch, err := usecase.ConfirmBatch(context.Background(), []uuid.UUID{fixture.organizationID}, fixture.actorUserID, batch.ID, expectedVersions)
		if err != nil {
			t.Fatalf("确认对冲批次账单失败: %v", err)
		}
		if len(confirmedBatch.Nettings) != 1 || confirmedBatch.Nettings[0].Status != biz.FinanceNettingDraft {
			t.Fatalf("批次确认不应改变对冲单状态: %#v", confirmedBatch.Nettings)
		}

		nettingUsecase := biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(data), data, nil, nil)
		confirmed, err := nettingUsecase.Confirm(context.Background(), []uuid.UUID{fixture.organizationID}, fixture.actorUserID, netting.ID, netting.Version)
		if err != nil {
			t.Fatalf("确认对冲单失败: %v", err)
		}
		if confirmed.Status != biz.FinanceNettingConfirmed || confirmed.Version != netting.Version+1 {
			t.Fatalf("对冲确认结果错误: %#v", confirmed)
		}
		billRepo := NewFinanceBillRepo(data)
		receivableBill, err := billRepo.Get(context.Background(), []uuid.UUID{fixture.organizationID}, directions[biz.OrderFeeReceivable])
		if err != nil {
			t.Fatalf("读取应收账单失败: %v", err)
		}
		payableBill, err := billRepo.Get(context.Background(), []uuid.UUID{fixture.organizationID}, directions[biz.OrderFeePayable])
		if err != nil {
			t.Fatalf("读取应付账单失败: %v", err)
		}
		if !receivableBill.NettedAmount.Equal(decimal.RequireFromString("40")) || !receivableBill.UnverifiedAmount.Equal(decimal.RequireFromString("60")) {
			t.Fatalf("应收账单对冲后余额错误: netted=%s unverified=%s", receivableBill.NettedAmount, receivableBill.UnverifiedAmount)
		}
		if !payableBill.NettedAmount.Equal(decimal.RequireFromString("40")) || !payableBill.UnverifiedAmount.IsZero() {
			t.Fatalf("应付账单对冲后余额错误: netted=%s unverified=%s", payableBill.NettedAmount, payableBill.UnverifiedAmount)
		}

		reversed, err := nettingUsecase.Reverse(context.Background(), []uuid.UUID{fixture.organizationID}, fixture.actorUserID, netting.ID, confirmed.Version, "冲销测试回退")
		if err != nil {
			t.Fatalf("反转对冲单失败: %v", err)
		}
		if reversed.Status != biz.FinanceNettingReversed {
			t.Fatalf("对冲反转结果错误: %#v", reversed)
		}
		restoredReceivable, err := billRepo.Get(context.Background(), []uuid.UUID{fixture.organizationID}, directions[biz.OrderFeeReceivable])
		if err != nil {
			t.Fatalf("重读应收账单失败: %v", err)
		}
		restoredPayable, err := billRepo.Get(context.Background(), []uuid.UUID{fixture.organizationID}, directions[biz.OrderFeePayable])
		if err != nil {
			t.Fatalf("重读应付账单失败: %v", err)
		}
		if !restoredReceivable.NettedAmount.IsZero() || !restoredReceivable.UnverifiedAmount.Equal(decimal.RequireFromString("100")) ||
			!restoredPayable.NettedAmount.IsZero() || !restoredPayable.UnverifiedAmount.Equal(decimal.RequireFromString("40")) {
			t.Fatalf("反转后余额未恢复: receivable=%#v payable=%#v", restoredReceivable, restoredPayable)
		}
	})

	t.Run("对冲批次拒绝单方向费用", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		t.Cleanup(fixture.cleanupNettingRows)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("netting-single")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		_, err := usecase.PreviewBatch(context.Background(), fixture.organizationID, biz.PreviewFinanceBillBatchInput{
			FeeIDs: feeIDs, GroupingPolicy: biz.FinanceBillGroupingPolicy{Mode: "NETTING", SplitByOrder: true},
		})
		if !errors.Is(err, biz.ErrFinanceNettingSingleDirection) {
			t.Fatalf("单方向费用走对冲模式错误 = %v，期望 %v", err, biz.ErrFinanceNettingSingleDirection)
		}
	})
}
