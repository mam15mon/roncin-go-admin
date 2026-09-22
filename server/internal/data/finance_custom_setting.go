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
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.FinanceCustomSetting.Query().Where(settingent.OrganizationIDEQ(organizationID)).WithUpdatedByUser().Only(ctx)
	if ent.IsNotFound(err) {
		return &biz.BilledFeeEditPolicy{OrganizationID: organizationID, EditableFields: []biz.BilledFeeEditableField{}}, nil
	}
	if err != nil {
		return nil, err
	}
	return financeCustomSettingToPolicy(item), nil
}

func (r *financeCustomSettingRepo) GetCreditLimitControlPolicy(ctx context.Context, organizationID uuid.UUID) (*biz.CreditLimitControlPolicy, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.FinanceCustomSetting.Query().Where(settingent.OrganizationIDEQ(organizationID)).WithUpdatedByUser().Only(ctx)
	if ent.IsNotFound(err) {
		return &biz.CreditLimitControlPolicy{OrganizationID: organizationID, AllowSelectionWhenCreditExceeded: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return creditLimitControlPolicyToBiz(item), nil
}

func (r *financeCustomSettingRepo) SaveCreditLimitControlPolicy(ctx context.Context, organizationID, actorID uuid.UUID, policy *biz.CreditLimitControlPolicy, expectedVersion uint64, audit *biz.AuditEvent) (*biz.CreditLimitControlPolicy, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.FinanceCustomSetting.Query().Where(settingent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		switch {
		case ent.IsNotFound(queryErr):
			if expectedVersion != 0 {
				return biz.ErrFinanceCustomSettingConflict
			}
			_, queryErr = tx.FinanceCustomSetting.Create().SetOrganizationID(organizationID).
				SetCreditLimitSelectionAllowed(policy.AllowSelectionWhenCreditExceeded).
				SetVersion(1).SetUpdatedBy(actorID).Save(ctx)
			if queryErr != nil {
				return mapEntError(queryErr, nil, biz.ErrFinanceCustomSettingConflict)
			}
		case queryErr != nil:
			return queryErr
		case current.Version != expectedVersion:
			return biz.ErrFinanceCustomSettingConflict
		default:
			_, queryErr = tx.FinanceCustomSetting.UpdateOneID(current.ID).
				SetCreditLimitSelectionAllowed(policy.AllowSelectionWhenCreditExceeded).
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
	// 完整业务响应在事务提交后用普通上下文重读，操作人姓名等关联数据取自已提交状态。
	return r.GetCreditLimitControlPolicy(ctx, organizationID)
}

func (r *financeCustomSettingRepo) SaveBilledFeeEditPolicy(ctx context.Context, organizationID, actorID uuid.UUID, policy *biz.BilledFeeEditPolicy, expectedVersion uint64, audit *biz.AuditEvent) (*biz.BilledFeeEditPolicy, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.FinanceCustomSetting.Query().Where(settingent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		flags := billedFeeFieldFlags(policy.EditableFields)
		switch {
		case ent.IsNotFound(queryErr):
			if expectedVersion != 0 {
				return biz.ErrFinanceCustomSettingConflict
			}
			_, queryErr = tx.FinanceCustomSetting.Create().SetOrganizationID(organizationID).SetBilledFeeEditEnabled(policy.Enabled).
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
			_, queryErr = tx.FinanceCustomSetting.UpdateOneID(current.ID).SetBilledFeeEditEnabled(policy.Enabled).
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
	// 完整业务响应在事务提交后用普通上下文重读，操作人姓名等关联数据取自已提交状态。
	return r.GetBilledFeeEditPolicy(ctx, organizationID)
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
	return &biz.BilledFeeEditPolicy{OrganizationID: item.OrganizationID, Enabled: item.BilledFeeEditEnabled, EditableFields: fields, Version: item.Version, UpdatedAt: &updatedAt, UpdatedBy: &updatedBy, UpdatedByName: updatedByUserName(item)}
}

// updatedByUserName 取策略操作人显示名；边未加载或用户不可考时返回空串，由传输层决定是否下发。
func updatedByUserName(item *ent.FinanceCustomSetting) string {
	if item.Edges.UpdatedByUser == nil {
		return ""
	}
	return item.Edges.UpdatedByUser.DisplayName
}

func creditLimitControlPolicyToBiz(item *ent.FinanceCustomSetting) *biz.CreditLimitControlPolicy {
	updatedAt, updatedBy := item.UpdatedAt, item.UpdatedBy
	return &biz.CreditLimitControlPolicy{OrganizationID: item.OrganizationID, AllowSelectionWhenCreditExceeded: item.CreditLimitSelectionAllowed, Version: item.Version, UpdatedAt: &updatedAt, UpdatedBy: &updatedBy, UpdatedByName: updatedByUserName(item)}
}

var _ biz.FinanceCustomSettingRepo = (*financeCustomSettingRepo)(nil)
