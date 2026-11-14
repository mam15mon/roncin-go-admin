package biz

import (
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
)

func TestOrderRateDay(t *testing.T) {
	cases := []struct {
		name    string
		order   *Order
		want    string
		wantErr error
	}{
		{"出口取 ETD 忽略 ETA", &Order{TradeDirection: OrderTradeExport, ETD: "2026-09-18T00:00:00+08:00", ETA: "2026-10-30T00:00:00+08:00"}, "2026-09-18", nil},
		{"进口取 ETA 忽略 ETD", &Order{TradeDirection: OrderTradeImport, ETD: "2026-09-18T00:00:00+08:00", ETA: "2026-10-30T00:00:00+08:00"}, "2026-10-30", nil},
		{"出口缺 ETD 不回退 ETA", &Order{TradeDirection: OrderTradeExport, ETA: "2026-10-30T00:00:00+08:00"}, "", ErrOrderFeeEtdMissing},
		{"进口缺 ETA 不回退 ETD", &Order{TradeDirection: OrderTradeImport, ETD: "2026-09-18T00:00:00+08:00"}, "", ErrOrderFeeEtaMissing},
		{"出口 ETD 格式非法", &Order{TradeDirection: OrderTradeExport, ETD: "2026-09-18"}, "", ErrOrderFeeEtdMissing},
		{"业务时区跨日：UTC 周日 17:00 为上海周一 01:00", &Order{TradeDirection: OrderTradeExport, ETD: "2026-09-20T17:00:00Z"}, "2026-09-21", nil},
		{"业务时区跨日：UTC 周一 15:59 仍为上海周一", &Order{TradeDirection: OrderTradeImport, ETA: "2026-09-21T15:59:00Z"}, "2026-09-21", nil},
		{"贸易方向缺失显式失败", &Order{ETD: "2026-09-18T00:00:00+08:00"}, "", ErrOrderFeeInvalidArgument},
		{"订单为空", nil, "", ErrOrderFeeInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := OrderRateDay(tc.order)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("期望错误 %v，实际 %v", tc.wantErr, err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("期望 %s，实际 %s，err=%v", tc.want, got, err)
			}
		})
	}
}

func TestNewExchangeRateMissingForDateMetadata(t *testing.T) {
	// 2026-09-23 为周三，目标周周一为 2026-09-21。
	err := newExchangeRateMissingForDate("usd", "2026-09-23", OrderFeePayable)
	if !errors.Is(err, ErrExchangeRateMissing) || !errors.Is(err, ErrExchangeRateMissingForDate) {
		t.Fatalf("应与缺汇率错误互通: %v", err)
	}
	meta := errors.FromError(err).Metadata
	if meta["currency"] != "USD" || meta["week_start"] != "2026-09-21" || meta["direction"] != string(OrderFeePayable) {
		t.Fatalf("元数据不符: %v", meta)
	}
	other := errors.BadRequest("OTHER", "x")
	if wrapExchangeRateMissingForDate(other, "USD", "2026-09-23", OrderFeePayable) != other {
		t.Fatal("非缺汇率错误必须原样返回")
	}
}
