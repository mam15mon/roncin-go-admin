package data

import (
	"context"

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
		return nil, mapEntError(err, biz.ErrPartnerSettlementRuleInvalidArgument, nil)
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

func (r *partnerSettlementRuleRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, roleType biz.PartnerRoleType, input *biz.PartnerSettlementRule, audit *biz.AuditEvent) (*biz.PartnerSettlementRule, error) {
	role, err := r.role(ctx, organizationID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	if err := validateSettlementCurrencies(ctx, r.data.db.Currency.Query(), input); err != nil {
		return nil, err
	}
	var created *ent.PartnerSettlementRule
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var saveErr error
		created, saveErr = createPartnerSettlementRule(ctx, tx.PartnerSettlementRule.Create().SetPartnerRoleID(role.ID), input)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "partner_settlement_rule_key", biz.ErrPartnerSettlementRuleExists)
		}
		audit.Details["rule.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return partnerSettlementRuleToBiz(created), nil
}

func (r *partnerSettlementRuleRepo) Update(ctx context.Context, organizationID, partnerID uuid.UUID, roleType biz.PartnerRoleType, id uuid.UUID, input *biz.PartnerSettlementRule, audit *biz.AuditEvent) (*biz.PartnerSettlementRule, error) {
	role, err := r.role(ctx, organizationID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	if err := validateSettlementCurrencies(ctx, r.data.db.Currency.Query(), input); err != nil {
		return nil, err
	}
	var updated *ent.PartnerSettlementRule
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		update := tx.PartnerSettlementRule.UpdateOneID(id).Where(partnerfilterent.PartnerRoleIDEQ(role.ID))
		var saveErr error
		updated, saveErr = updatePartnerSettlementRule(ctx, update, input)
		if saveErr != nil {
			return mapEntConstraint(
				mapEntError(saveErr, biz.ErrPartnerSettlementRuleNotFound, nil),
				"partner_settlement_rule_key",
				biz.ErrPartnerSettlementRuleExists,
			)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
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
