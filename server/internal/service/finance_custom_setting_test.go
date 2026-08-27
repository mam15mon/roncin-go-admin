package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestCustomSettingUpdatesRejectMissingExpectedVersion(t *testing.T) {
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       uuid.New(),
		Organization: biz.Organization{ID: uuid.New()},
	})

	t.Run("汇率继承设置", func(t *testing.T) {
		service := &ExchangeRateService{}
		_, err := service.UpdateExchangeRateCustomSetting(ctx, &v1.UpdateExchangeRateCustomSettingRequest{})
		if err != biz.ErrExchangeRateInvalidArgument {
			t.Fatalf("未传 expected_version 时错误为 %v，期望 %v", err, biz.ErrExchangeRateInvalidArgument)
		}
	})

	t.Run("账单费用修改策略", func(t *testing.T) {
		service := &SettlementService{}
		_, err := service.UpdateBilledFeeEditPolicy(ctx, &v1.UpdateBilledFeeEditPolicyRequest{})
		if err != biz.ErrFinanceCustomSettingInvalidArgument {
			t.Fatalf("未传 expected_version 时错误为 %v，期望 %v", err, biz.ErrFinanceCustomSettingInvalidArgument)
		}
	})
}
