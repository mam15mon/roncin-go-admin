package biz

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerSettlementRuleNotFound        = errors.NotFound("PARTNER_SETTLEMENT_RULE_NOT_FOUND", "结算规则不存在")
	ErrPartnerSettlementRuleExists          = errors.Conflict("PARTNER_SETTLEMENT_RULE_EXISTS", "结算规则已存在")
	ErrPartnerSettlementRuleInvalidArgument = errors.BadRequest("PARTNER_SETTLEMENT_RULE_INVALID_ARGUMENT", "结算规则字段不合法")
)

type PartnerStatementMode string

const (
	PartnerStatementSingle PartnerStatementMode = "single"
	PartnerStatementMulti  PartnerStatementMode = "multi"
)

func (m PartnerStatementMode) Valid() bool {
	return m == PartnerStatementSingle || m == PartnerStatementMulti
}

type PartnerSettlementMethod string

const (
	PartnerSettlementByTicket    PartnerSettlementMethod = "by_ticket"
	PartnerSettlementMonthly     PartnerSettlementMethod = "monthly"
	PartnerSettlementWeekly      PartnerSettlementMethod = "weekly"
	PartnerSettlementSemiMonthly PartnerSettlementMethod = "semi_monthly"
	PartnerSettlementBiMonthly   PartnerSettlementMethod = "bi_monthly"
	PartnerSettlementQuarterly   PartnerSettlementMethod = "quarterly"
	PartnerSettlementDays45      PartnerSettlementMethod = "days_45"
	PartnerSettlementPrepaid     PartnerSettlementMethod = "prepaid"
)

func (m PartnerSettlementMethod) Valid() bool {
	switch m {
	case PartnerSettlementByTicket, PartnerSettlementMonthly, PartnerSettlementWeekly, PartnerSettlementSemiMonthly, PartnerSettlementBiMonthly, PartnerSettlementQuarterly, PartnerSettlementDays45, PartnerSettlementPrepaid:
		return true
	default:
		return false
	}
}

type PartnerSettlementBase string

const (
	PartnerSettlementBillDate    PartnerSettlementBase = "bill_date"
	PartnerSettlementSailingDate PartnerSettlementBase = "sailing_date"
	PartnerSettlementArrivalDate PartnerSettlementBase = "arrival_date"
)

func (b PartnerSettlementBase) Valid() bool {
	return b == PartnerSettlementBillDate || b == PartnerSettlementSailingDate || b == PartnerSettlementArrivalDate
}

type PartnerSettlementRule struct {
	ID                  uuid.UUID
	PartnerRoleID       uuid.UUID
	StatementMode       PartnerStatementMode
	SettlementMethod    PartnerSettlementMethod
	SettlementDay       *int
	SettlementCycleDays *int
	SettlementBase      *PartnerSettlementBase
	SettlementCurrency  string
	CreditLimitMinor    *int64
	CreditCurrency      *string
	IsActive            bool
}

type PartnerSettlementRuleRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType) ([]*PartnerSettlementRule, error)
	Create(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType, *PartnerSettlementRule) (*PartnerSettlementRule, error)
	Update(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType, uuid.UUID, *PartnerSettlementRule) (*PartnerSettlementRule, error)
}

type PartnerSettlementRuleUsecase struct {
	repo  PartnerSettlementRuleRepo
	audit AuditRepo
}

func NewPartnerSettlementRuleUsecase(repo PartnerSettlementRuleRepo, audit AuditRepo) *PartnerSettlementRuleUsecase {
	return &PartnerSettlementRuleUsecase{repo: repo, audit: audit}
}

func (uc *PartnerSettlementRuleUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID, roleType PartnerRoleType) ([]*PartnerSettlementRule, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil || !roleType.Valid() {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID, roleType)
}

func (uc *PartnerSettlementRuleUsecase) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, roleType PartnerRoleType, input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	normalized, err := normalizePartnerSettlementRule(input)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, partnerID, roleType, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.settlement_rule.create", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"rule.id": created.ID.String(), "partner.id": partnerID.String(), "role": string(roleType)}}); err != nil {
		return nil, fmt.Errorf("write partner settlement rule create audit: %w", err)
	}
	return created, nil
}

func (uc *PartnerSettlementRuleUsecase) Update(ctx context.Context, organizationID, actorID, partnerID, id uuid.UUID, roleType PartnerRoleType, input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerSettlementRuleNotFound
	}
	normalized, err := normalizePartnerSettlementRule(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, partnerID, roleType, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.settlement_rule.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"rule.id": updated.ID.String(), "partner.id": partnerID.String(), "role": string(roleType)}}); err != nil {
		return nil, fmt.Errorf("write partner settlement rule update audit: %w", err)
	}
	return updated, nil
}

func normalizePartnerSettlementRule(input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	if input == nil {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	output := *input
	output.SettlementCurrency = strings.ToUpper(strings.TrimSpace(output.SettlementCurrency))
	if output.CreditCurrency != nil {
		value := strings.ToUpper(strings.TrimSpace(*output.CreditCurrency))
		output.CreditCurrency = &value
	}
	if !output.StatementMode.Valid() || !output.SettlementMethod.Valid() || len(output.SettlementCurrency) != 3 {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if output.SettlementDay != nil {
		max := 31
		if output.SettlementMethod == PartnerSettlementWeekly {
			max = 7
		}
		if *output.SettlementDay < 1 || *output.SettlementDay > max {
			return nil, ErrPartnerSettlementRuleInvalidArgument
		}
	}
	if output.SettlementCycleDays != nil && (*output.SettlementCycleDays < 1 || *output.SettlementCycleDays > 365) {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if output.SettlementBase != nil && !output.SettlementBase.Valid() {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if output.SettlementMethod == PartnerSettlementByTicket || output.SettlementMethod == PartnerSettlementPrepaid {
		if output.SettlementDay != nil || output.SettlementCycleDays != nil || output.SettlementBase != nil {
			return nil, ErrPartnerSettlementRuleInvalidArgument
		}
	} else if output.SettlementBase == nil {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if output.CreditLimitMinor != nil && (*output.CreditLimitMinor < 0 || output.CreditCurrency == nil) {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	if output.CreditCurrency != nil && (len(*output.CreditCurrency) != 3 || output.CreditLimitMinor == nil) {
		return nil, ErrPartnerSettlementRuleInvalidArgument
	}
	return &output, nil
}
