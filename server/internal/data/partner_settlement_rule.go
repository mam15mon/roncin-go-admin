package data

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	partnerfilterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
)

type partnerSettlementRuleRepo struct{ data *Data }

func NewPartnerSettlementRuleRepo(data *Data) biz.PartnerSettlementRuleRepo {
	return &partnerSettlementRuleRepo{data: data}
}

func (r *partnerSettlementRuleRepo) role(ctx context.Context, organizationID, partnerID uuid.UUID, roleType biz.PartnerRoleType) (*ent.PartnerRole, error) {
	role, err := r.data.db.PartnerRole.Query().Where(
		partnerroleent.PartnerIDEQ(partnerID),
		partnerroleent.RoleTypeEQ(partnerroleent.RoleType(roleType)),
		partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerSettlementRuleInvalidArgument
		}
		return nil, err
	}
	return role, nil
}

func (r *partnerSettlementRuleRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, roleType biz.PartnerRoleType) ([]*biz.PartnerSettlementRule, error) {
	role, err := r.role(ctx, organizationID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	items, err := r.data.db.PartnerSettlementRule.Query().Where(
		partnerfilterent.PartnerRoleIDEQ(role.ID),
	).Order(partnerfilterent.ByCreatedAt()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerSettlementRule, 0, len(items))
	for _, item := range items {
		result = append(result, partnerSettlementRuleToBiz(item))
	}
	return result, nil
}

func (r *partnerSettlementRuleRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, roleType biz.PartnerRoleType, input *biz.PartnerSettlementRule) (*biz.PartnerSettlementRule, error) {
	role, err := r.role(ctx, organizationID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	if err := validateSettlementCurrencies(ctx, r.data.db.Currency.Query(), input); err != nil {
		return nil, err
	}
	created, err := createPartnerSettlementRule(ctx, r.data.db.PartnerSettlementRule.Create().SetPartnerRoleID(role.ID), input)
	if err != nil {
		return nil, mapPartnerSettlementRuleConstraint(err)
	}
	return partnerSettlementRuleToBiz(created), nil
}

func (r *partnerSettlementRuleRepo) Update(ctx context.Context, organizationID, partnerID uuid.UUID, roleType biz.PartnerRoleType, id uuid.UUID, input *biz.PartnerSettlementRule) (*biz.PartnerSettlementRule, error) {
	role, err := r.role(ctx, organizationID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	if err := validateSettlementCurrencies(ctx, r.data.db.Currency.Query(), input); err != nil {
		return nil, err
	}
	update := r.data.db.PartnerSettlementRule.UpdateOneID(id).Where(partnerfilterent.PartnerRoleIDEQ(role.ID))
	updated, err := updatePartnerSettlementRule(ctx, update, input)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerSettlementRuleNotFound
		}
		return nil, mapPartnerSettlementRuleConstraint(err)
	}
	return partnerSettlementRuleToBiz(updated), nil
}

func createPartnerSettlementRule(ctx context.Context, builder *ent.PartnerSettlementRuleCreate, input *biz.PartnerSettlementRule) (*ent.PartnerSettlementRule, error) {
	builder.SetStatementMode(partnerfilterent.StatementMode(input.StatementMode)).
		SetSettlementMethod(partnerfilterent.SettlementMethod(input.SettlementMethod)).
		SetSettlementCurrency(input.SettlementCurrency).
		SetIsActive(input.IsActive)
	if input.SettlementDay != nil {
		builder.SetSettlementDay(*input.SettlementDay)
	}
	if input.SettlementCycleDays != nil {
		builder.SetSettlementCycleDays(*input.SettlementCycleDays)
	}
	if input.SettlementBase != nil {
		builder.SetSettlementBase(partnerfilterent.SettlementBase(*input.SettlementBase))
	}
	if input.CreditLimitMinor != nil {
		builder.SetCreditLimitMinor(*input.CreditLimitMinor).SetCreditCurrency(*input.CreditCurrency)
	}
	return builder.Save(ctx)
}

func updatePartnerSettlementRule(ctx context.Context, update *ent.PartnerSettlementRuleUpdateOne, input *biz.PartnerSettlementRule) (*ent.PartnerSettlementRule, error) {
	update.SetStatementMode(partnerfilterent.StatementMode(input.StatementMode)).
		SetSettlementMethod(partnerfilterent.SettlementMethod(input.SettlementMethod)).
		SetSettlementCurrency(input.SettlementCurrency).
		SetIsActive(input.IsActive)
	if input.SettlementDay == nil {
		update.ClearSettlementDay()
	} else {
		update.SetSettlementDay(*input.SettlementDay)
	}
	if input.SettlementCycleDays == nil {
		update.ClearSettlementCycleDays()
	} else {
		update.SetSettlementCycleDays(*input.SettlementCycleDays)
	}
	if input.SettlementBase == nil {
		update.ClearSettlementBase()
	} else {
		update.SetSettlementBase(partnerfilterent.SettlementBase(*input.SettlementBase))
	}
	if input.CreditLimitMinor == nil {
		update.ClearCreditLimitMinor().ClearCreditCurrency()
	} else {
		update.SetCreditLimitMinor(*input.CreditLimitMinor).SetCreditCurrency(*input.CreditCurrency)
	}
	return update.Save(ctx)
}

func mapPartnerSettlementRuleConstraint(err error) error {
	if ent.IsConstraintError(err) && strings.Contains(err.Error(), "partner_settlement_rule_key") {
		return biz.ErrPartnerSettlementRuleExists
	}
	return err
}

func partnerSettlementRuleToBiz(item *ent.PartnerSettlementRule) *biz.PartnerSettlementRule {
	result := &biz.PartnerSettlementRule{
		ID: item.ID, PartnerRoleID: item.PartnerRoleID,
		StatementMode: biz.PartnerStatementMode(item.StatementMode), SettlementMethod: biz.PartnerSettlementMethod(item.SettlementMethod),
		SettlementCurrency: item.SettlementCurrency, IsActive: item.IsActive,
	}
	if item.SettlementDay != nil {
		value := *item.SettlementDay
		result.SettlementDay = &value
	}
	if item.SettlementCycleDays != nil {
		value := *item.SettlementCycleDays
		result.SettlementCycleDays = &value
	}
	if item.SettlementBase != nil {
		value := biz.PartnerSettlementBase(*item.SettlementBase)
		result.SettlementBase = &value
	}
	result.CreditLimitMinor = item.CreditLimitMinor
	result.CreditCurrency = item.CreditCurrency
	return result
}

func validateSettlementCurrencies(ctx context.Context, query *ent.CurrencyQuery, input *biz.PartnerSettlementRule) error {
	codes := []string{input.SettlementCurrency}
	if input.CreditCurrency != nil && *input.CreditCurrency != input.SettlementCurrency {
		codes = append(codes, *input.CreditCurrency)
	}
	count, err := query.Where(currencyent.CodeIn(codes...), currencyent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(codes) {
		return biz.ErrPartnerSettlementRuleInvalidArgument
	}
	return nil
}
