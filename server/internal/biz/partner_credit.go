package biz

import (
	"context"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/shopspring/decimal"
)

// ErrPartnerCreditLimitExceeded 是直接干预模式下的刚性拦截错误：
// 仅在组织显式关闭「超额后允许选择」且目标客户确已超额时返回；
// 默认的仅提醒模式不产生本错误，建账入账环节也不使用本错误。
var ErrPartnerCreditLimitExceeded = errors.BadRequest(
	reasonFromProto(financev1.ErrorReason_ERROR_REASON_PARTNER_CREDIT_LIMIT_EXCEEDED),
	"该客户应收未核销总额已超出信用额度，系统已限制选择",
)

// PartnerCreditRepo 提供信用额度判定所需的聚合数据；
// 实现位于 internal/data/finance_bill.go，复用账单未核销折本币口径，不另写余额算法。
type PartnerCreditRepo interface {
	GetPartnerUnsettledReceivableBaseAmount(ctx context.Context, organizationID, partnerID uuid.UUID) (decimal.Decimal, error)
	GetPartnerCreditSummaries(ctx context.Context, organizationID uuid.UUID, partnerIDs []uuid.UUID) (map[uuid.UUID]*PartnerCreditSummary, error)
}

// PartnerCreditUsecase 组合管控策略与信用聚合，向订单、应收费用写入门禁暴露统一判定。
type PartnerCreditUsecase struct {
	repo          PartnerCreditRepo
	customSetting *FinanceCustomSettingUsecase
}

func NewPartnerCreditUsecase(repo PartnerCreditRepo, customSetting *FinanceCustomSettingUsecase) *PartnerCreditUsecase {
	return &PartnerCreditUsecase{repo: repo, customSetting: customSetting}
}

// EnsurePartnerSelectionAllowed 在直接干预模式下校验客户未超额；
// 仅提醒模式（默认）与未超额客户直接放行。校验为尽力而为：查询时点余额，
// 不做串行化加锁，并发窗口内的漏拦以超额预警与事后报表校正。
func (uc *PartnerCreditUsecase) EnsurePartnerSelectionAllowed(ctx context.Context, organizationID, partnerID uuid.UUID) error {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return ErrFinanceCustomSettingInvalidArgument
	}
	interventionActive, err := uc.customSetting.IsCreditLimitInterventionActive(ctx, organizationID)
	if err != nil {
		return err
	}
	if !interventionActive {
		return nil
	}
	summaries, err := uc.repo.GetPartnerCreditSummaries(ctx, organizationID, []uuid.UUID{partnerID})
	if err != nil {
		return err
	}
	if PartnerCreditExceeded(summaries[partnerID]) {
		return ErrPartnerCreditLimitExceeded
	}
	return nil
}
