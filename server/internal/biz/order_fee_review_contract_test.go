package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFeeReviewSnapshotAcceptsRecordedBankRounding(t *testing.T) {
	fee := validOrderFeeForTest()
	fee.ID = uuid.New()
	fee.Quantity = decimal.RequireFromString("1.0001")
	fee.UnitPrice = decimal.RequireFromString("100.0003")
	fee.ExchangeRate = decimal.RequireFromString("7.5")
	zeroTax := decimal.Zero
	fee.TaxRate = &zeroTax
	normalized, err := normalizeOrderFee(fee)
	if err != nil {
		t.Fatalf("正常费用录入应接受数量和单价: %v", err)
	}
	u := NewOrderFeeUsecase(nil, NewExchangeRateUsecase(&orderFeeExchangeRateRepoStub{}, nil), nil, nil, nil)
	if err := u.calculateAmounts(context.Background(), uuid.New(), normalized); err != nil {
		t.Fatalf("正常费用计算失败: %v", err)
	}
	if normalized.TotalAmount.StringFixed(8) != "100.01030003" || normalized.BaseCurrencyAmount.StringFixed(8) != "750.07725022" {
		t.Fatalf("录入舍入口径错误: total=%s base=%s", normalized.TotalAmount, normalized.BaseCurrencyAmount)
	}
	snapshot, _, err := BuildFeeReviewSnapshot([]*OrderFee{normalized})
	if err != nil || snapshot == nil || !snapshot.Summary.Receivable.Equal(normalized.BaseCurrencyAmount) {
		t.Fatalf("审核必须接受正常录入保存的八位银行家舍入: %+v %v", snapshot, err)
	}
	normalized.BaseCurrencyAmount = decimal.RequireFromString("750.07725023")
	if _, _, err := BuildFeeReviewSnapshot([]*OrderFee{normalized}); !errors.Is(err, ErrFeeReviewInvalid) {
		t.Fatalf("金额不一致仍应拒绝: %v", err)
	}
}
