package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// partnerCreditRepoStub 是不关心信用拦截的测试默认仓储：无激活规则、零余额，
// 因此任何往来户都判定为不超额。
type partnerCreditRepoStub struct {
	summaries map[uuid.UUID]*PartnerCreditSummary
}

func (r *partnerCreditRepoStub) GetPartnerUnsettledReceivableBaseAmount(_ context.Context, _, partnerID uuid.UUID) (decimal.Decimal, error) {
	summary := r.summaries[partnerID]
	if summary == nil {
		return decimal.Zero, nil
	}
	return summary.UnsettledReceivableBase, nil
}

func (r *partnerCreditRepoStub) GetPartnerCreditSummaries(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]*PartnerCreditSummary, error) {
	return r.summaries, nil
}

// interventionModeSettingRepo 是可直接干预模式的自定义设置仓储存根。
type interventionModeSettingRepo struct {
	FinanceCustomSettingRepo
	allowSelection bool
}

func (r *interventionModeSettingRepo) GetCreditLimitControlPolicy(_ context.Context, organizationID uuid.UUID) (*CreditLimitControlPolicy, error) {
	return &CreditLimitControlPolicy{OrganizationID: organizationID, AllowSelectionWhenCreditExceeded: r.allowSelection}, nil
}

// newReminderModeCreditControl 返回默认（仅提醒）模式的信用管控用例，供不关心拦截的测试复用。
func newReminderModeCreditControl() *PartnerCreditUsecase {
	return NewPartnerCreditUsecase(&partnerCreditRepoStub{}, NewFinanceCustomSettingUsecase(&interventionModeSettingRepo{allowSelection: true}))
}

func TestPartnerCreditExceededOnlyWhenPositiveLimitAndGreaterBalance(t *testing.T) {
	limit := decimal.NewFromInt(1000)
	zeroLimit := decimal.Zero
	testCases := []struct {
		name     string
		summary  *PartnerCreditSummary
		exceeded bool
	}{
		{"未设置额度不判定", &PartnerCreditSummary{UnsettledReceivableBase: decimal.NewFromInt(9999)}, false},
		{"额度为零不判定", &PartnerCreditSummary{CreditLimitBase: &zeroLimit, UnsettledReceivableBase: decimal.NewFromInt(1)}, false},
		{"余额等于额度不算超额", &PartnerCreditSummary{CreditLimitBase: &limit, UnsettledReceivableBase: decimal.NewFromInt(1000)}, false},
		{"余额小于额度不算超额", &PartnerCreditSummary{CreditLimitBase: &limit, UnsettledReceivableBase: decimal.NewFromInt(999)}, false},
		{"余额大于额度判定超额", &PartnerCreditSummary{CreditLimitBase: &limit, UnsettledReceivableBase: decimal.NewFromInt(1001)}, true},
		{"空摘要不判定", nil, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := PartnerCreditExceeded(testCase.summary); got != testCase.exceeded {
				t.Fatalf("PartnerCreditExceeded = %v, 期望 %v", got, testCase.exceeded)
			}
		})
	}
}

func TestEnsurePartnerSelectionAllowedReminderModeNeverBlocks(t *testing.T) {
	ctx := context.Background()
	organizationID := uuid.Must(uuid.NewV7())
	partnerID := uuid.Must(uuid.NewV7())
	limit := decimal.NewFromInt(100)
	usecase := NewPartnerCreditUsecase(&partnerCreditRepoStub{summaries: map[uuid.UUID]*PartnerCreditSummary{
		partnerID: {PartnerID: partnerID, CreditLimitBase: &limit, UnsettledReceivableBase: decimal.NewFromInt(500)},
	}}, NewFinanceCustomSettingUsecase(&interventionModeSettingRepo{allowSelection: true}))

	if err := usecase.EnsurePartnerSelectionAllowed(ctx, organizationID, partnerID); err != nil {
		t.Fatalf("仅提醒模式下超额客户不应被拦截: %v", err)
	}
}

func TestEnsurePartnerSelectionAllowedInterventionModeBlocksExceededOnly(t *testing.T) {
	ctx := context.Background()
	organizationID := uuid.Must(uuid.NewV7())
	exceededID := uuid.Must(uuid.NewV7())
	normalID := uuid.Must(uuid.NewV7())
	limit := decimal.NewFromInt(100)
	usecase := NewPartnerCreditUsecase(&partnerCreditRepoStub{summaries: map[uuid.UUID]*PartnerCreditSummary{
		exceededID: {PartnerID: exceededID, CreditLimitBase: &limit, UnsettledReceivableBase: decimal.NewFromInt(150)},
		normalID:   {PartnerID: normalID, CreditLimitBase: &limit, UnsettledReceivableBase: decimal.NewFromInt(50)},
	}}, NewFinanceCustomSettingUsecase(&interventionModeSettingRepo{allowSelection: false}))

	if err := usecase.EnsurePartnerSelectionAllowed(ctx, organizationID, exceededID); err != ErrPartnerCreditLimitExceeded {
		t.Fatalf("直接干预模式下超额客户应返回 ErrPartnerCreditLimitExceeded, got %v", err)
	}
	if err := usecase.EnsurePartnerSelectionAllowed(ctx, organizationID, normalID); err != nil {
		t.Fatalf("直接干预模式下未超额客户不应被拦截: %v", err)
	}
	if err := usecase.EnsurePartnerSelectionAllowed(ctx, organizationID, uuid.Must(uuid.NewV7())); err != nil {
		t.Fatalf("直接干预模式下未配置额度的客户不应被拦截: %v", err)
	}
	if err := usecase.EnsurePartnerSelectionAllowed(ctx, uuid.Nil, exceededID); err != ErrFinanceCustomSettingInvalidArgument {
		t.Fatalf("组织 ID 为空应返回参数错误, got %v", err)
	}
}
