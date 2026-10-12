package biz

import (
	"context"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

type feeReviewBatchCountingRepo struct{ reads, writes int }

func (r *feeReviewBatchCountingRepo) Get(_ context.Context, _ *Principal, id uuid.UUID) (*OrderFeeReview, error) {
	r.reads++
	return &OrderFeeReview{OrderID: id}, nil
}
func (r *feeReviewBatchCountingRepo) Act(_ context.Context, _ *Principal, _ string, input FeeReviewAction) (*OrderFeeReview, error) {
	r.writes++
	return &OrderFeeReview{OrderID: input.OrderID}, nil
}

func TestFeeReviewBatchUsecaseLimitsBeforeRepository(t *testing.T) {
	r := &feeReviewBatchCountingRepo{}
	u := NewOrderFeeReviewUsecase(r)
	ids := make([]uuid.UUID, MaxListPageSize)
	inputs := make([]FeeReviewAction, len(ids))
	for i := range ids {
		ids[i] = uuid.New()
		inputs[i] = FeeReviewAction{OrderID: ids[i], ExpectedReviewVersion: 1, ExpectedContentVersion: 1, IdempotencyKey: ids[i].String()}
	}
	if out, err := u.PreviewBatch(context.Background(), nil, ids); err != nil || len(out) != 200 || r.reads != 200 {
		t.Fatalf("200票预览应全部读取: %d %v", r.reads, err)
	}
	if out, err := u.ApproveBatch(context.Background(), nil, inputs); err != nil || len(out) != 200 || r.writes != 200 {
		t.Fatalf("200票批准应逐票调用: %d %v", r.writes, err)
	}
	if _, err := u.PreviewBatch(context.Background(), nil, append(ids, uuid.New())); err == nil || r.reads != 200 {
		t.Fatal("201票预览不得进入仓储")
	}
	if _, err := u.ApproveBatch(context.Background(), nil, append(inputs, FeeReviewAction{OrderID: uuid.New()})); err == nil || r.writes != 200 {
		t.Fatal("201票批准不得写入")
	}
	inputs[len(inputs)-1].IdempotencyKey = ""
	if _, err := u.ApproveBatch(context.Background(), nil, inputs); err == nil || r.writes != 200 {
		t.Fatal("末项参数错误也必须在首票写入前拒绝")
	}
}

func TestFeeReviewSnapshotEconomicIdentity(t *testing.T) {
	fee := &OrderFee{ID: uuid.New(), OrderID: uuid.New(), Direction: OrderFeeReceivable, SettlementPartyID: uuid.New(), SettlementPartyName: "原名", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100), Currency: "CNY", ExchangeRate: decimal.NewFromInt(1), BaseCurrency: "CNY", BaseCurrencyAmount: decimal.NewFromInt(100)}
	_, first, err := BuildFeeReviewSnapshot([]*OrderFee{fee})
	if err != nil {
		t.Fatal(err)
	}
	fee.Version = 5
	fee.HasActiveBill = true
	fee.UpdatedAt = time.Now()
	fee.SettlementPartyName = "主数据改名"
	snapshot, next, err := BuildFeeReviewSnapshot([]*OrderFee{fee})
	if err != nil {
		t.Fatal(err)
	}
	if first != next {
		t.Fatal("建账关联、行版本及结算单位展示名不应改变审核内容指纹")
	}
	if snapshot.Fees[0].SettlementPartyName != "主数据改名" {
		t.Fatal("展示名应留在历史快照")
	}
	cost := *fee
	cost.ID = uuid.New()
	cost.Direction = OrderFeePayable
	cost.TotalAmount = decimal.NewFromInt(150)
	cost.BaseCurrencyAmount = cost.TotalAmount
	loss, changed, err := BuildFeeReviewSnapshot([]*OrderFee{fee, &cost})
	if err != nil {
		t.Fatal(err)
	}
	if changed == first || !loss.Summary.Profit.Equal(decimal.NewFromInt(-50)) {
		t.Fatal("新增成本必须改变指纹并准确计算亏损")
	}
	_, reverse, err := BuildFeeReviewSnapshot([]*OrderFee{&cost, fee})
	if err != nil || reverse != changed {
		t.Fatal("同集合排序不应改变指纹")
	}
	fee.SupplementRequestID = &cost.ID
	_, source, err := BuildFeeReviewSnapshot([]*OrderFee{fee})
	if err != nil || source == first {
		t.Fatal("补录来源必须参与指纹")
	}
}
func TestFeeReviewSnapshotRejectsMissingRatesAndMixedCurrencies(t *testing.T) {
	fee := &OrderFee{ID: uuid.New(), Direction: OrderFeeReceivable, TotalAmount: decimal.NewFromInt(10), BaseCurrency: "CNY", BaseCurrencyAmount: decimal.NewFromInt(10)}
	if _, _, err := BuildFeeReviewSnapshot([]*OrderFee{fee}); err == nil {
		t.Fatal("缺汇率不能用1补齐")
	}
	fee.ExchangeRate = decimal.NewFromInt(1)
	other := *fee
	other.ID = uuid.New()
	other.BaseCurrency = "USD"
	if _, _, err := BuildFeeReviewSnapshot([]*OrderFee{fee, &other}); err == nil {
		t.Fatal("混合本位币不能相加")
	}
	fee.TotalAmount = decimal.Zero
	fee.BaseCurrencyAmount = decimal.Zero
	snapshot, _, err := BuildFeeReviewSnapshot([]*OrderFee{fee})
	if err != nil || snapshot.Summary.ProfitRate != nil {
		t.Fatal("零收入利润率必须不可计算")
	}
}

func TestFeeReviewBatchBoundaries(t *testing.T) {
	ids := make([]uuid.UUID, MaxListPageSize)
	for i := range ids {
		ids[i] = uuid.New()
	}
	if err := ValidateFeeReviewIDs(ids); err != nil {
		t.Fatal("200票应允许", err)
	}
	if err := ValidateFeeReviewIDs(append(ids, uuid.New())); err == nil {
		t.Fatal("201票应拒绝")
	}
	if err := ValidateFeeReviewIDs(nil); err == nil {
		t.Fatal("空批次应拒绝")
	}
	if err := ValidateFeeReviewIDs([]uuid.UUID{ids[0], ids[0]}); err == nil {
		t.Fatal("重复订单应拒绝")
	}
	if err := ValidateFeeReviewAction("APPROVE", FeeReviewAction{OrderID: uuid.New(), ExpectedContentVersion: 1, ExpectedReviewVersion: 1}); err == nil {
		t.Fatal("批次中格式不合法项应在任何写入前拒绝")
	}
}
