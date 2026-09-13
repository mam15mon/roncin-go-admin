package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// noopPartnerCreditRepoStub 提供无激活规则、零余额的信用聚合存根：任何往来户都不超额。
type noopPartnerCreditRepoStub struct{}

func (noopPartnerCreditRepoStub) GetPartnerUnsettledReceivableBaseAmount(context.Context, uuid.UUID, uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (noopPartnerCreditRepoStub) GetPartnerCreditSummaries(context.Context, uuid.UUID, []uuid.UUID) (map[uuid.UUID]*biz.PartnerCreditSummary, error) {
	return map[uuid.UUID]*biz.PartnerCreditSummary{}, nil
}

// noopFinanceCustomSettingRepoStub 未保存过策略，读取时回落到默认仅提醒模式，不做拦截。
type noopFinanceCustomSettingRepoStub struct {
	biz.FinanceCustomSettingRepo
}

func (noopFinanceCustomSettingRepoStub) GetCreditLimitControlPolicy(context.Context, uuid.UUID) (*biz.CreditLimitControlPolicy, error) {
	return nil, nil
}

// newReminderModeCreditControl 供路由与鉴权测试构造默认（仅提醒）模式的信用管控用例。
func newReminderModeCreditControl() *biz.PartnerCreditUsecase {
	return biz.NewPartnerCreditUsecase(noopPartnerCreditRepoStub{}, biz.NewFinanceCustomSettingUsecase(noopFinanceCustomSettingRepoStub{}))
}
