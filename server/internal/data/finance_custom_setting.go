package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	settingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecustomsetting"
)

type financeCustomSettingRepo struct{ data *Data }

func NewFinanceCustomSettingRepo(data *Data) biz.FinanceCustomSettingRepo {
	return &financeCustomSettingRepo{data: data}
}

func (r *financeCustomSettingRepo) GetBilledFeeEditPolicy(ctx context.Context, organizationID uuid.UUID) (*biz.BilledFeeEditPolicy, error) {
	ownerID, err := resolveHeadquartersOrganizationID(ctx, r.data.db.Organization, organizationID)
	if err != nil {
		return nil, err
	}
	item, err := r.data.db.FinanceCustomSetting.Query().Where(settingent.OrganizationIDEQ(ownerID)).Only(ctx)
	if ent.IsNotFound(err) {
		return &biz.BilledFeeEditPolicy{OrganizationID: ownerID, EditableFields: []biz.BilledFeeEditableField{}}, nil
	}
	if err != nil {
		return nil, err
	}
	return financeCustomSettingToPolicy(item), nil
}

func (r *financeCustomSettingRepo) SaveBilledFeeEditPolicy(ctx context.Context, organizationID, actorID uuid.UUID, policy *biz.BilledFeeEditPolicy, expectedVersion uint64, audit *biz.AuditEvent) (*biz.BilledFeeEditPolicy, error) {
	ownerID, err := resolveHeadquartersOrganizationID(ctx, r.data.db.Organization, organizationID)
	if err != nil {
		return nil, err
	}
	var saved *ent.FinanceCustomSetting
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.FinanceCustomSetting.Query().Where(settingent.OrganizationIDEQ(ownerID)).ForUpdate().Only(ctx)
		flags := billedFeeFieldFlags(policy.EditableFields)
		switch {
		case ent.IsNotFound(queryErr):
			if expectedVersion != 0 {
				return biz.ErrFinanceCustomSettingConflict
			}
			saved, queryErr = tx.FinanceCustomSetting.Create().SetOrganizationID(ownerID).SetBilledFeeEditEnabled(policy.Enabled).
				SetBilledFeeNameEditable(flags[biz.BilledFeeFieldFeeName]).SetBilledFeeCurrencyEditable(flags[biz.BilledFeeFieldCurrency]).
				SetBilledFeeExchangeRateEditable(flags[biz.BilledFeeFieldExchangeRate]).SetBilledFeeQuantityEditable(flags[biz.BilledFeeFieldQuantity]).
				SetBilledFeeUnitPriceEditable(flags[biz.BilledFeeFieldUnitPrice]).SetBilledFeeTaxRateEditable(flags[biz.BilledFeeFieldTaxRate]).
				SetVersion(1).SetUpdatedBy(actorID).Save(ctx)
			if queryErr != nil {
				return mapEntError(queryErr, nil, biz.ErrFinanceCustomSettingConflict)
			}
		case queryErr != nil:
			return queryErr
		case current.Version != expectedVersion:
			return biz.ErrFinanceCustomSettingConflict
		default:
			saved, queryErr = tx.FinanceCustomSetting.UpdateOneID(current.ID).SetBilledFeeEditEnabled(policy.Enabled).
				SetBilledFeeNameEditable(flags[biz.BilledFeeFieldFeeName]).SetBilledFeeCurrencyEditable(flags[biz.BilledFeeFieldCurrency]).
				SetBilledFeeExchangeRateEditable(flags[biz.BilledFeeFieldExchangeRate]).SetBilledFeeQuantityEditable(flags[biz.BilledFeeFieldQuantity]).
				SetBilledFeeUnitPriceEditable(flags[biz.BilledFeeFieldUnitPrice]).SetBilledFeeTaxRateEditable(flags[biz.BilledFeeFieldTaxRate]).
				SetVersion(current.Version + 1).SetUpdatedBy(actorID).Save(ctx)
		}
		if queryErr != nil {
			return queryErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return financeCustomSettingToPolicy(saved), nil
}

func billedFeeFieldFlags(fields []biz.BilledFeeEditableField) map[biz.BilledFeeEditableField]bool {
	result := make(map[biz.BilledFeeEditableField]bool, len(fields))
	for _, field := range fields {
		result[field] = true
	}
	return result
}

func financeCustomSettingToPolicy(item *ent.FinanceCustomSetting) *biz.BilledFeeEditPolicy {
	fields := make([]biz.BilledFeeEditableField, 0, 6)
	for _, candidate := range []struct {
		field   biz.BilledFeeEditableField
		enabled bool
	}{
		{biz.BilledFeeFieldFeeName, item.BilledFeeNameEditable}, {biz.BilledFeeFieldCurrency, item.BilledFeeCurrencyEditable},
		{biz.BilledFeeFieldExchangeRate, item.BilledFeeExchangeRateEditable}, {biz.BilledFeeFieldQuantity, item.BilledFeeQuantityEditable},
		{biz.BilledFeeFieldUnitPrice, item.BilledFeeUnitPriceEditable}, {biz.BilledFeeFieldTaxRate, item.BilledFeeTaxRateEditable},
	} {
		if candidate.enabled {
			fields = append(fields, candidate.field)
		}
	}
	updatedAt, updatedBy := item.UpdatedAt, item.UpdatedBy
	return &biz.BilledFeeEditPolicy{OrganizationID: item.OrganizationID, Enabled: item.BilledFeeEditEnabled, EditableFields: fields, Version: item.Version, UpdatedAt: &updatedAt, UpdatedBy: &updatedBy}
}

var _ biz.FinanceCustomSettingRepo = (*financeCustomSettingRepo)(nil)
