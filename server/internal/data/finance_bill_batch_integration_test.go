package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	financebillbatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillbatch"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
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
		feeIDs := []uuid.UUID{fixture.createBatchConfirmedFee("race-first"), fixture.createBatchConfirmedFee("race-second")}
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
		feeIDs := []uuid.UUID{fixture.createBatchConfirmedFee("stale-token")}
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
		feeIDs := []uuid.UUID{fixture.createBatchConfirmedFee("replay-first"), fixture.createBatchConfirmedFee("replay-second")}
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

// createBatchConfirmedFee 创建带税率（分组必填）的已确认应收费用；批量分组路径要求税率非空。
func (f *financeBillBatchPostgresFixture) createBatchConfirmedFee(key string) uuid.UUID {
	f.t.Helper()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("batch-fee-"+key+"-"+f.suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusCONFIRMED).
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
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceBASE_CURRENCY).
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
	if err != nil || fee.Status != orderfeeent.StatusCONFIRMED {
		f.t.Fatalf("零写入校验：费用状态 = %#v，期望 CONFIRMED（未被部分改成 BILLED），error=%v", fee, err)
	}
}
