package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerSettlementRuleNotFound        = errors.NotFound("PARTNER_SETTLEMENT_RULE_NOT_FOUND", "收款约定不存在")
	ErrPartnerSettlementRuleExists          = errors.Conflict("PARTNER_SETTLEMENT_RULE_EXISTS", "收款约定已存在")
	ErrPartnerSettlementRuleInvalidArgument = errors.BadRequest("PARTNER_SETTLEMENT_RULE_INVALID_ARGUMENT", "收款约定字段不合法")
	ErrPartnerCreditCurrencyMismatch        = errors.BadRequest("PARTNER_CREDIT_CURRENCY_MISMATCH", "信用额度必须使用所属公司本币，请在客户收款约定中重新填写本币额度")
)

// PartnerSettlementRule 是客户角色唯一的收款约定。
type PartnerSettlementRule struct {
	ID               uuid.UUID
	PartnerRoleID    uuid.UUID
	CreditLimitMinor *int64
	CreditCurrency   *string
	PaymentTermsDays *int
}

type PartnerSettlementRuleRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType) ([]*PartnerSettlementRule, error)
	Create(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType, *PartnerSettlementRule, *AuditEvent) (*PartnerSettlementRule, error)
	Update(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType, uuid.UUID, *PartnerSettlementRule, *AuditEvent) (*PartnerSettlementRule, error)
}

type PartnerSettlementRuleUsecase struct {
	repo PartnerSettlementRuleRepo
}

func NewPartnerSettlementRuleUsecase(repo PartnerSettlementRuleRepo) *PartnerSettlementRuleUsecase {
	return &PartnerSettlementRuleUsecase{repo: repo}
}

func (uc *PartnerSettlementRuleUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID, roleType PartnerRoleType) ([]*PartnerSettlementRule, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil || !roleType.Valid() {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID, roleType)
}

func (uc *PartnerSettlementRuleUsecase) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, roleType PartnerRoleType, input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	if roleType != PartnerRoleCustomer {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	normalized, err := normalizePartnerSettlementRule(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Create(ctx, organizationID, partnerID, roleType, normalized, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.settlement_rule.create", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"partner.id": partnerID.String(), "role": string(roleType)}})
}

func (uc *PartnerSettlementRuleUsecase) Update(ctx context.Context, organizationID, actorID, partnerID, id uuid.UUID, roleType PartnerRoleType, input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	if roleType != PartnerRoleCustomer {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if id == uuid.Nil {
		return nil, ErrPartnerSettlementRuleNotFound
	}
	normalized, err := normalizePartnerSettlementRule(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, partnerID, roleType, id, normalized, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.settlement_rule.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"rule.id": id.String(), "partner.id": partnerID.String(), "role": string(roleType)}})
}

func normalizePartnerSettlementRule(input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	if input == nil {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	output := *input
	if output.CreditCurrency != nil {
		value := strings.ToUpper(strings.TrimSpace(*output.CreditCurrency))
		output.CreditCurrency = &value
	}
	if output.CreditLimitMinor != nil && (*output.CreditLimitMinor < 0 || output.CreditCurrency == nil) {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if output.CreditCurrency != nil && (len(*output.CreditCurrency) != 3 || output.CreditLimitMinor == nil) {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	// NULL 未约定、0 现结、正数为建账后 N 天；范围与 Ent Schema 一致。
	if output.PaymentTermsDays != nil && (*output.PaymentTermsDays < 0 || *output.PaymentTermsDays > 3650) {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	return &output, nil
}
