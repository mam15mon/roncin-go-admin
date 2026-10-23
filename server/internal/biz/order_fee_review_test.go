package biz

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

func TestFeeReviewSnapshotEconomicIdentity(t *testing.T) {
	fee := &OrderFee{ID: uuid.New(), OrderID: uuid.New(), Direction: OrderFeeReceivable, SettlementPartyID: uuid.New(), SettlementPartyName: "原名", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100), Currency: "CNY", ExchangeRate: decimal.NewFromInt(1), BaseCurrency: "CNY", BaseCurrencyAmount: decimal.NewFromInt(100)}
	_, first, err := BuildFeeReviewSnapshot([]*OrderFee{fee})
	if err != nil {
		t.Fatal(err)
	}
	fee.Version = 5
	fee.HasActiveBill = true
	fee.CanEdit = true
	fee.CanDelete = true
	fee.CanBill = true
	fee.CanCorrect = true
	fee.HasPendingChange = true
	fee.BlockedReasons = []string{"展示原因"}
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
