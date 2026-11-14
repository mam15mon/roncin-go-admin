package data

import (
	"context"
	"encoding/json"
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
	result *biz.FinanceBillBatchCreateResult
	err    error
}

func ensureFinanceBillBatchCurrencies(t *testing.T, data *Data, codes ...string) {
	t.Helper()
	ctx := context.Background()
	for _, code := range codes {
		exists, err := data.db.Currency.Query().Where(currencyent.CodeEQ(code)).Exist(ctx)
		if err != nil {
			t.Fatalf("查询批量建账用例所需币种: %v", err)
		}
		if !exists {
			if _, err := data.db.Currency.Create().SetCode(code).SetName(code).SetEnabled(true).Save(ctx); err != nil {
				t.Fatalf("创建批量建账用例所需币种: %v", err)
			}
		}
	}
}

// groupResultsByKey 断言逐组结果覆盖请求全部组并按 group_key 索引。
func groupResultsByKey(t *testing.T, input biz.CreateFinanceBillBatchInput, result *biz.FinanceBillBatchCreateResult) map[string]*biz.FinanceBillBatchGroupResult {
	t.Helper()
	if result == nil || result.Batch == nil || len(result.Groups) != len(input.Groups) {
		t.Fatalf("逐组结果应覆盖请求内每一组: %#v", result)
	}
	byKey := make(map[string]*biz.FinanceBillBatchGroupResult, len(result.Groups))
	for _, group := range result.Groups {
		if (group.Bill == nil) == (group.Err == nil) {
			t.Fatalf("组结果必须恰有账单或错误之一: %#v", group)
		}
		byKey[group.GroupKey] = group
	}
	return byKey
}

func TestFinanceBillBatchCreatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ensureFinanceBillBatchCurrencies(t, data, "CNY", "USD")
	ctx := context.Background()

	t.Run("应收应付并存按方向分组且每组一张账单，批次按本位币聚合", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		receivable := fixture.createBatchUnbilledFee("mixed-ar")
		payable := fixture.createBatchUnbilledPayableFee("mixed-ap")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-mixed-"+fixture.suffix, []uuid.UUID{receivable, payable}, biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}, nil)
		if len(input.Groups) != 2 {
			t.Fatalf("应收、应付应各成一组: %#v", input.Groups)
		}
		result, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorUserID, input)
		if err != nil {
			t.Fatalf("混合方向批量建账失败: %v", err)
		}
		directions := map[biz.OrderFeeDirection]int{}
		for _, group := range groupResultsByKey(t, input, result) {
			if group.Err != nil {
				t.Fatalf("混合方向各组应成功: %v", group.Err)
			}
			if group.Bill.Status != biz.FinanceBillConfirmed || len(group.Bill.Lines) != 1 || group.Bill.BatchID == nil || *group.Bill.BatchID != result.Batch.ID {
				t.Fatalf("组账单字段错误: %#v", group.Bill)
			}
			directions[group.Bill.Direction]++
		}
		if directions[biz.OrderFeeReceivable] != 1 || directions[biz.OrderFeePayable] != 1 {
			t.Fatalf("应收、应付各生成一张账单: %#v", directions)
		}
		batch := result.Batch
		if batch.BillCount != 2 || batch.FeeCount != 2 || len(batch.BaseTotals) != 1 || batch.BaseTotals[0].BaseCurrency != "CNY" || !batch.BaseTotals[0].TotalBaseAmount.Equal(decimal.NewFromInt(140)) {
			t.Fatalf("批次读取时聚合错误: bills=%d fees=%d totals=%#v", batch.BillCount, batch.FeeCount, batch.BaseTotals)
		}
		fixture.requireCommittedBatchState([]uuid.UUID{receivable, payable}, 1, 2)
	})

	t.Run("某组失败其他组成功，同键重试成功组原样返回、失败组重试成功", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		cnyFee := fixture.createBatchUnbilledFee("retry-cny")
		usdFee := fixture.createBatchUnbilledFee("retry-usd")
		if _, err := data.db.OrderFee.UpdateOneID(usdFee).SetCurrency("USD").SetExchangeRate("7.00000000").SetBaseCurrencyAmount("700.00000000").Save(ctx); err != nil {
			t.Fatal(err)
		}
		settingID := fixture.createExchangeRateSetting("7")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-retry-"+fixture.suffix, []uuid.UUID{cnyFee, usdFee}, biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}, func(group *biz.FinanceBillBatchPreviewGroup) uuid.UUID {
			if group.Currency == "USD" {
				return fixture.usdAccountID
			}
			return fixture.accountID
		})
		usdGroupKey := ""
		for _, group := range input.Groups {
			if group.FeeIDs[0] == usdFee {
				usdGroupKey = group.GroupKey
			}
		}
		// 预览签发令牌后停用美元汇率：美元组因缺汇率失败，人民币组不受影响。
		if _, err := data.db.ExchangeRateSetting.UpdateOneID(settingID).SetIsActive(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		first, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorUserID, input)
		if err != nil {
			t.Fatalf("部分失败的批次请求本身不应失败: %v", err)
		}
		firstGroups := groupResultsByKey(t, input, first)
		var cnyBill *biz.FinanceBill
		for key, group := range firstGroups {
			if key == usdGroupKey {
				if !errors.Is(group.Err, biz.ErrExchangeRateMissing) {
					t.Fatalf("美元组应因缺汇率失败: %v", group.Err)
				}
				continue
			}
			if group.Err != nil {
				t.Fatalf("人民币组应成功: %v", group.Err)
			}
			cnyBill = group.Bill
		}
		if first.Batch.BillCount != 1 || first.Batch.FeeCount != 1 {
			t.Fatalf("部分成功批次只聚合成功账单: %#v", first.Batch)
		}
		if lines, countErr := data.db.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(usdFee)).Count(ctx); countErr != nil || lines != 0 {
			t.Fatalf("失败组不得留下账单行: lines=%d err=%v", lines, countErr)
		}
		if fee, getErr := data.db.OrderFee.Get(ctx, usdFee); getErr != nil || fee.Version != 1 {
			t.Fatalf("失败组费用版本不得变化: %#v %v", fee, getErr)
		}

		if _, err := data.db.ExchangeRateSetting.UpdateOneID(settingID).SetIsActive(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		// 另一位财务接手重试：批次创建者不变，新增成功组的确认人及经济重算操作者取本次实际人员。
		second, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorID, input)
		if err != nil {
			t.Fatalf("同键重试失败: %v", err)
		}
		if second.Batch.ID != first.Batch.ID || second.Batch.BatchNo != first.Batch.BatchNo {
			t.Fatalf("同键重试必须复用同一批次: first=%s second=%s", first.Batch.ID, second.Batch.ID)
		}
		for key, group := range groupResultsByKey(t, input, second) {
			if group.Err != nil {
				t.Fatalf("重试后各组应成功: %s %v", key, group.Err)
			}
			if key != usdGroupKey && (group.Bill.ID != cnyBill.ID || group.Bill.BillNo != cnyBill.BillNo || group.Bill.Version != cnyBill.Version) {
				t.Fatalf("已成功组应原样返回: before=%#v after=%#v", cnyBill, group.Bill)
			}
			if key == usdGroupKey && (group.Bill.ConfirmedBy == nil || *group.Bill.ConfirmedBy != fixture.actorID) {
				t.Fatalf("重试新建组应记录本次实际创建人: %#v", group.Bill.ConfirmedBy)
			}
			if key != usdGroupKey && (group.Bill.ConfirmedBy == nil || *group.Bill.ConfirmedBy != fixture.actorUserID) {
				t.Fatalf("已有成功组的确认人不随重试变化: %#v", group.Bill.ConfirmedBy)
			}
		}
		totals := second.Batch.BaseTotals
		if second.Batch.BillCount != 2 || second.Batch.FeeCount != 2 || len(totals) != 1 || totals[0].BaseCurrency != "CNY" || !totals[0].TotalBaseAmount.Equal(decimal.NewFromInt(800)) {
			t.Fatalf("批次本位币聚合错误: %#v", second.Batch)
		}
		fixture.requireCommittedBatchState([]uuid.UUID{cnyFee, usdFee}, 1, 2)
	})

	t.Run("组内陈旧令牌只影响该组", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		receivable := fixture.createBatchUnbilledFee("stale-ar")
		payable := fixture.createBatchUnbilledPayableFee("stale-ap")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-stale-"+fixture.suffix, []uuid.UUID{receivable, payable}, biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}, nil)
		// 模拟预览签发令牌后应付费用事实被修改（版本与金额变化）。
		if _, err := data.db.OrderFee.UpdateOneID(payable).SetTotalAmount("50.00000000").SetNetAmount("50.00000000").SetBaseCurrencyAmount("50.00000000").AddVersion(1).Save(ctx); err != nil {
			t.Fatalf("修改费用事实失败: %v", err)
		}
		result, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorUserID, input)
		if err != nil {
			t.Fatalf("组陈旧不应使整个请求失败: %v", err)
		}
		for _, group := range groupResultsByKey(t, input, result) {
			payableGroup := len(group.GroupKey) > 0 && containsFeeID(input, group.GroupKey, payable)
			if payableGroup && !errors.Is(group.Err, biz.ErrFinanceBillPreviewStale) {
				t.Fatalf("应付组应因令牌陈旧失败: %v", group.Err)
			}
			if !payableGroup && group.Err != nil {
				t.Fatalf("应收组不受影响: %v", group.Err)
			}
		}
		if fee, getErr := data.db.OrderFee.Get(ctx, payable); getErr != nil || fee.Version != 2 {
			t.Fatalf("陈旧组费用版本应保持修改后的 2: %#v %v", fee, getErr)
		}
		if lines, countErr := data.db.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(payable)).Count(ctx); countErr != nil || lines != 0 {
			t.Fatalf("陈旧组不得写入账单行: %d %v", lines, countErr)
		}
		if bills, countErr := data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(fixture.organizationID)).Count(ctx); countErr != nil || bills != 1 {
			t.Fatalf("应只生成应收组账单: %d %v", bills, countErr)
		}
	})

	t.Run("请求公司解析不因某组待审更正失败且实际建账仍拒绝该组", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		receivable := fixture.createBatchUnbilledFee("pending-ar")
		payable := fixture.createBatchUnbilledPayableFee("pending-ap")
		repo := NewFinanceBillRepo(data)
		usecase := fixture.newUsecase(repo)
		feeIDs := []uuid.UUID{receivable, payable}
		input := fixture.buildBatchInput(t, usecase, "batch-pending-"+fixture.suffix, feeIDs, biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}, nil)
		request, err := data.db.OrderFeeCorrectionRequest.Create().
			SetOrganizationID(fixture.organizationID).SetOrderID(fixture.orderID).
			SetOperation("VOID").SetSourceFeeID(payable).SetSourceFeeVersion(1).
			SetBeforeSnapshot(json.RawMessage(`{}`)).
			SetApprovedFeeContentVersion(1).SetApprovedFeeReviewVersion(1).
			SetReason("预览后提交费用作废补审").SetIdempotencyKey("pending-" + fixture.suffix).
			SetRequestFingerprint("0000000000000000000000000000000000000000000000000000000000000000").
			SetRequestedBy(fixture.actorUserID).SetRequestedAt(time.Now()).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := data.db.OrderFeeCorrectionRequest.DeleteOneID(request.ID).Exec(ctx); err != nil {
				t.Errorf("清理待审费用更正: %v", err)
			}
		})
		if organizationID, err := usecase.ResolveBillableFeeOrganization(ctx, []uuid.UUID{fixture.organizationID}, feeIDs); err != nil || organizationID != fixture.organizationID {
			t.Fatalf("请求级公司解析只检查归属，不因待审更正拒绝整批: org=%s err=%v", organizationID, err)
		}
		if _, err := repo.LoadBillableFees(ctx, fixture.organizationID, []uuid.UUID{payable}); !errors.Is(err, biz.ErrFeeCorrectionPending) {
			t.Fatalf("普通及实际组建账仍检查待审更正: %v", err)
		}
		result, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorUserID, input)
		if err != nil {
			t.Fatalf("待审费用更正不应拒绝整批: %v", err)
		}
		for _, group := range groupResultsByKey(t, input, result) {
			if containsFeeID(input, group.GroupKey, payable) {
				if !errors.Is(group.Err, biz.ErrFeeCorrectionPending) {
					t.Fatalf("应付组应因费用待审失败: %v", group.Err)
				}
			} else if group.Err != nil {
				t.Fatalf("应收组应成功: %v", group.Err)
			}
		}
		if result.Batch.BillCount != 1 {
			t.Fatalf("应只有无待审更正的应收组生成账单: %#v", result.Batch)
		}
	})

	t.Run("同幂等键不同请求返回批次冲突", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("conflict")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-conflict-"+fixture.suffix, feeIDs, biz.FinanceBillGroupingPolicy{Mode: "NORMAL", SplitByOrder: true}, nil)
		if _, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorUserID, input); err != nil {
			t.Fatal(err)
		}
		changed := input
		changed.Groups = append([]biz.CreateFinanceBillBatchGroupInput(nil), input.Groups...)
		changed.Groups[0].StatementTitle = "改过的对账抬头"
		if _, err := usecase.CreateBatch(ctx, fixture.organizationID, fixture.actorUserID, changed); !errors.Is(err, biz.ErrFinanceBillBatchConflict) {
			t.Fatalf("同键不同请求应返回批次冲突: %v", err)
		}
		fixture.requireCommittedBatchState(feeIDs, 1, 1)
	})

	t.Run("不同幂等键并发抢占同一组费用只有一张账单成功", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("race-first"), fixture.createBatchUnbilledFee("race-second")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		first := fixture.buildBatchInput(t, usecase, "batch-race-first-"+fixture.suffix, feeIDs, biz.FinanceBillGroupingPolicy{Mode: "NORMAL", SplitByOrder: true}, nil)
		second := first
		second.IdempotencyKey = "batch-race-second-" + fixture.suffix

		results := createFinanceBillBatchesConcurrently(usecase, fixture.organizationID, fixture.actorUserID, first, second)
		successes, conflicts := 0, 0
		for _, item := range results {
			if item.err != nil {
				t.Fatalf("并发建批请求本身不应失败: %v", item.err)
			}
			for _, group := range groupResultsByKey(t, first, item.result) {
				switch {
				case group.Err == nil:
					successes++
				case errors.Is(group.Err, biz.ErrFinanceBillFeeInvalid), errors.Is(group.Err, biz.ErrFinanceBillPreviewStale):
					conflicts++
				default:
					t.Fatalf("并发建批返回非预期组错误: %v", group.Err)
				}
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("并发建批结果 success=%d conflict=%d，期望各 1", successes, conflicts)
		}
		fixture.requireCommittedBatchState(feeIDs, 2, 1)
	})

	t.Run("相同幂等键并发重放返回同一批次且无重复账单", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		feeIDs := []uuid.UUID{fixture.createBatchUnbilledFee("replay-first"), fixture.createBatchUnbilledFee("replay-second")}
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildBatchInput(t, usecase, "batch-replay-"+fixture.suffix, feeIDs, biz.FinanceBillGroupingPolicy{Mode: "NORMAL", SplitByOrder: true}, nil)

		results := createFinanceBillBatchesConcurrently(usecase, fixture.organizationID, fixture.actorUserID, input, input)
		billIDs := map[uuid.UUID]struct{}{}
		for index, item := range results {
			if item.err != nil {
				t.Fatalf("第 %d 个并发重放请求失败: %v", index+1, item.err)
			}
			for _, group := range groupResultsByKey(t, input, item.result) {
				if group.Err != nil {
					t.Fatalf("第 %d 个并发重放请求组失败: %v", index+1, group.Err)
				}
				billIDs[group.Bill.ID] = struct{}{}
			}
		}
		if results[0].result.Batch.ID != results[1].result.Batch.ID || len(billIDs) != 1 {
			t.Fatalf("相同幂等键返回了不同批次或重复账单: %#v", billIDs)
		}
		fixture.requireCommittedBatchState(feeIDs, 1, 1)
	})
}

func containsFeeID(input biz.CreateFinanceBillBatchInput, groupKey string, feeID uuid.UUID) bool {
	for _, group := range input.Groups {
		if group.GroupKey != groupKey {
			continue
		}
		for _, id := range group.FeeIDs {
			if id == feeID {
				return true
			}
		}
	}
	return false
}

// previewGroupFeeIDs 取预览组内全部费用 ID，作为创建请求的组费用集合。
func previewGroupFeeIDs(group *biz.FinanceBillBatchPreviewGroup) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(group.Fees))
	for _, item := range group.Fees {
		ids = append(ids, item.Fee.ID)
	}
	return ids
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
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建批量建账测试费用: %v", err)
	}
	return fee.ID
}

// buildBatchInput 复现前端两段式流程：先无配置预览拿叶子 groupKey，再带日期和账户
// 预览签发各组令牌，最后组装与令牌一致的批量创建输入。account 为空时各组使用人民币账户。
func (f *financeBillBatchPostgresFixture) buildBatchInput(t *testing.T, usecase *biz.FinanceBillUsecase, idempotencyKey string, feeIDs []uuid.UUID, policy biz.FinanceBillGroupingPolicy, account func(*biz.FinanceBillBatchPreviewGroup) uuid.UUID) biz.CreateFinanceBillBatchInput {
	t.Helper()
	ctx := context.Background()
	if account == nil {
		account = func(*biz.FinanceBillBatchPreviewGroup) uuid.UUID { return f.accountID }
	}
	initial, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy})
	if err != nil {
		t.Fatalf("无配置批量预览失败: %v", err)
	}
	configs := make([]biz.FinanceBillBatchPreviewGroupConfig, 0, len(initial.Groups))
	for _, group := range initial.Groups {
		if group.GroupToken != "" {
			t.Fatalf("无日期账户的预览不应签发组令牌: %#v", group)
		}
		configs = append(configs, biz.FinanceBillBatchPreviewGroupConfig{
			GroupKey: group.GroupKey, BillDate: financeBillIntegrationDate, SettlementAccountID: account(group),
		})
	}
	preview, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy, GroupConfigs: configs})
	if err != nil {
		t.Fatalf("完整配置批量预览失败: %v", err)
	}
	groups := make([]biz.CreateFinanceBillBatchGroupInput, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		if group.GroupToken == "" {
			t.Fatalf("完整配置预览未签发组令牌: %#v", group)
		}
		groups = append(groups, biz.CreateFinanceBillBatchGroupInput{
			GroupKey: group.GroupKey, GroupToken: group.GroupToken, FeeIDs: previewGroupFeeIDs(group), StatementTitle: group.SettlementPartyName, BillDate: financeBillIntegrationDate, SettlementAccountID: group.SettlementAccountID,
		})
	}
	return biz.CreateFinanceBillBatchInput{GroupingPolicy: policy, Groups: groups, IdempotencyKey: idempotencyKey}
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
			result, err := usecase.CreateBatch(ctx, organizationID, actorID, input)
			results <- financeBillBatchCreateResult{result: result, err: err}
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
		if getErr != nil || fee.Version != 2 {
			f.t.Fatalf("批量建账后费用版本 = %#v，期望 2，error=%v", fee, getErr)
		}
	}
}

func (f *financeBillBatchPostgresFixture) requireRolledBackBatchState(feeID uuid.UUID, wantVersion uint64) {
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
	if err != nil || fee.Version != wantVersion {
		f.t.Fatalf("零写入校验：费用版本 = %#v，期望 %d（未发生部分写入），error=%v", fee, wantVersion, err)
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
	preview, err := usecase.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: policy})
	if err != nil || len(preview.NettingPairs) != 1 {
		t.Fatalf("对冲模式无配置预览失败: %#v %v", preview, err)
	}
	input := f.buildBatchInput(t, usecase, idempotencyKey, feeIDs, policy, nil)
	if len(input.Groups) != 2 {
		t.Fatalf("对冲预览应返回两组: %#v", input.Groups)
	}
	return input
}

func TestFinanceBillBatchNettingCreatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ensureFinanceBillBatchCurrencies(t, data, "CNY")

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

		created, err := usecase.CreateBatch(context.Background(), fixture.organizationID, fixture.actorUserID, input)
		if err != nil {
			t.Fatalf("对冲批次创建失败: %v", err)
		}
		for _, group := range groupResultsByKey(t, input, created) {
			if group.Err != nil {
				t.Fatalf("对冲批次各组应成功: %v", group.Err)
			}
		}
		batch := created.Batch
		if len(batch.Bills) != 2 || len(batch.Nettings) != 1 || batch.BillCount != 2 || batch.FeeCount != 2 {
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

		directions := map[biz.OrderFeeDirection]uuid.UUID{}
		for _, bill := range batch.Bills {
			// 账单免人工确认：批量创建即为已确认，确认人为批次创建人。
			if bill.Status != biz.FinanceBillConfirmed || bill.ConfirmedBy == nil || *bill.ConfirmedBy != fixture.actorUserID || bill.ConfirmedAt == nil {
				t.Fatalf("批量创建的账单应直接为已确认: %#v", bill)
			}
			directions[bill.Direction] = bill.ID
		}

		nettingUsecase := biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(data), data, nil)
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

	t.Run("对冲批次任一组陈旧整批失败且零写入", func(t *testing.T) {
		fixture := newFinanceBillBatchPostgresFixture(t, data)
		if _, err := data.db.NumberRule.Create().SetOrganizationID(fixture.organizationID).SetDocumentType(numberruleent.DocumentTypeNetting).SetPrefix("NT-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(context.Background()); err != nil {
			t.Fatalf("创建测试对冲编号规则: %v", err)
		}
		t.Cleanup(fixture.cleanupNettingRows)
		receivableFeeID := fixture.createBatchUnbilledFee("netting-atomic-receivable")
		payableFeeID := fixture.createBatchUnbilledPayableFee("netting-atomic-payable")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := fixture.buildNettingBatchInput(t, usecase, "batch-netting-atomic-"+fixture.suffix, []uuid.UUID{receivableFeeID, payableFeeID})
		if _, err := data.db.OrderFee.UpdateOneID(payableFeeID).SetTotalAmount("30.00000000").SetNetAmount("30.00000000").SetBaseCurrencyAmount("30.00000000").AddVersion(1).Save(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := usecase.CreateBatch(context.Background(), fixture.organizationID, fixture.actorUserID, input); !errors.Is(err, biz.ErrFinanceBillPreviewStale) {
			t.Fatalf("对冲批次一组陈旧应整批失败: %v", err)
		}
		fixture.requireRolledBackBatchState(receivableFeeID, 1)
		if count, err := data.db.FinanceNetting.Query().Where(financenettingent.OrganizationIDEQ(fixture.organizationID)).Count(context.Background()); err != nil || count != 0 {
			t.Fatalf("整批失败不得留下对冲单: %d %v", count, err)
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
