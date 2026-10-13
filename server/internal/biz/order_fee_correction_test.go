package biz

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"testing"
)

// 反向经济行仍是正数原始金额，资金方向与业务收支分别计算。
func TestFeeCorrectionEconomicAndAccountingViews(t *testing.T) {
	source := uuid.New()
	request := uuid.New()
	original := &OrderFee{ID: source, Direction: OrderFeeReceivable, EconomicEffect: OrderFeeEconomicIncrease, AccountingRequired: true, BaseCurrencyAmount: decimal.NewFromInt(100)}
	inverse := *original
	inverse.ID = uuid.New()
	inverse.SourceFeeID = &source
	inverse.CorrectionRequestID = &request
	inverse.EconomicEffect = OrderFeeEconomicDecrease
	replacement := *original
	replacement.ID = uuid.New()
	replacement.SourceFeeID = &source
	replacement.CorrectionRequestID = &request
	replacement.BaseCurrencyAmount = decimal.NewFromInt(80)
	original.SupersededByRequestID = &request
	if IsCurrentFee(original) || IsCurrentFee(&inverse) || !IsCurrentFee(&replacement) {
		t.Fatal("经济视图只能含当前正向叶子")
	}
	if RequiresAccounting(original) || !RequiresAccounting(&inverse) || !RequiresAccounting(&replacement) {
		t.Fatal("账务候选须保留应建冲回行")
	}
	if SettlementDirection(&inverse) != OrderFeePayable || inverse.Direction != OrderFeeReceivable || !EconomicAmount(&inverse).Equal(decimal.NewFromInt(-100)) {
		t.Fatal("冲回经济符号或资金方向错误")
	}
	if !EconomicAmount(original).Add(EconomicAmount(&inverse)).Add(EconomicAmount(&replacement)).Equal(decimal.NewFromInt(80)) {
		t.Fatal("完整更正历史不能重复计算利润")
	}
	inverse.AccountingRequired = false
	if RequiresAccounting(&inverse) {
		t.Fatal("原未建账时虚拟冲回不能进入财务候选")
	}
}
