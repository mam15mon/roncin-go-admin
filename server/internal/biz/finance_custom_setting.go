package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrFinanceCustomSettingInvalidArgument = errors.BadRequest("FINANCE_CUSTOM_SETTING_INVALID_ARGUMENT", "财务自定义设置字段不合法")
	ErrFinanceCustomSettingConflict        = errors.Conflict("FINANCE_CUSTOM_SETTING_CONFLICT", "财务自定义设置已被更新，请刷新后重试")
	ErrBilledFeeEditDisabled               = errors.Conflict("BILLED_FEE_EDIT_DISABLED", "账单创建后不允许修改费用")
	ErrBilledFeeFieldForbidden             = errors.Forbidden("BILLED_FEE_FIELD_FORBIDDEN", "当前字段不允许在账单创建后修改")
	ErrBilledFeeBillLocked                 = errors.Conflict("BILLED_FEE_BILL_LOCKED", "仅草稿账单中的费用允许修改")
	ErrBilledFeeCurrencyConflict           = errors.Conflict("BILLED_FEE_CURRENCY_CONFLICT", "多费用账单不允许单独修改其中一条费用的币种")
)

type BilledFeeEditableField string

const (
	BilledFeeFieldFeeName      BilledFeeEditableField = "FEE_NAME"
	BilledFeeFieldCurrency     BilledFeeEditableField = "CURRENCY"
	BilledFeeFieldExchangeRate BilledFeeEditableField = "EXCHANGE_RATE"
	BilledFeeFieldQuantity     BilledFeeEditableField = "QUANTITY"
	BilledFeeFieldUnitPrice    BilledFeeEditableField = "UNIT_PRICE"
	BilledFeeFieldTaxRate      BilledFeeEditableField = "TAX_RATE"
)

type BilledFeeEditPolicy struct {
	OrganizationID uuid.UUID
	Enabled        bool
	EditableFields []BilledFeeEditableField
	Version        uint64
	UpdatedAt      *time.Time
	UpdatedBy      *uuid.UUID
}

func (p *BilledFeeEditPolicy) Allows(field BilledFeeEditableField) bool {
	if p == nil || !p.Enabled {
		return false
	}
	for _, candidate := range p.EditableFields {
		if candidate == field {
			return true
		}
	}
	return false
}

type FinanceCustomSettingRepo interface {
	GetBilledFeeEditPolicy(context.Context, uuid.UUID) (*BilledFeeEditPolicy, error)
	SaveBilledFeeEditPolicy(context.Context, uuid.UUID, uuid.UUID, *BilledFeeEditPolicy, uint64, *AuditEvent) (*BilledFeeEditPolicy, error)
}

type FinanceCustomSettingUsecase struct{ repo FinanceCustomSettingRepo }

func NewFinanceCustomSettingUsecase(repo FinanceCustomSettingRepo) *FinanceCustomSettingUsecase {
	return &FinanceCustomSettingUsecase{repo: repo}
}

func (uc *FinanceCustomSettingUsecase) GetBilledFeeEditPolicy(ctx context.Context, organizationID uuid.UUID) (*BilledFeeEditPolicy, error) {
	if uc == nil || uc.repo == nil || organizationID == uuid.Nil {
		return nil, ErrFinanceCustomSettingInvalidArgument
	}
	policy, err := uc.repo.GetBilledFeeEditPolicy(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return &BilledFeeEditPolicy{OrganizationID: organizationID, EditableFields: []BilledFeeEditableField{}}, nil
	}
	return policy, nil
}

func (uc *FinanceCustomSettingUsecase) UpdateBilledFeeEditPolicy(ctx context.Context, organizationID, actorID uuid.UUID, input *BilledFeeEditPolicy, expectedVersion uint64) (*BilledFeeEditPolicy, error) {
	if uc == nil || uc.repo == nil || organizationID == uuid.Nil || actorID == uuid.Nil || input == nil {
		return nil, ErrFinanceCustomSettingInvalidArgument
	}
	fields, err := normalizeBilledFeeEditableFields(input.EditableFields)
	if err != nil {
		return nil, err
	}
	policy := &BilledFeeEditPolicy{Enabled: input.Enabled, EditableFields: fields}
	audit := &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "finance.custom_setting.billed_fee_edit.update", Result: "success", ResourceType: "finance_custom_setting", ResourceID: organizationID.String()}
	return uc.repo.SaveBilledFeeEditPolicy(ctx, organizationID, actorID, policy, expectedVersion, audit)
}

func normalizeBilledFeeEditableFields(fields []BilledFeeEditableField) ([]BilledFeeEditableField, error) {
	seen := make(map[BilledFeeEditableField]struct{}, len(fields))
	result := make([]BilledFeeEditableField, 0, len(fields))
	for _, field := range fields {
		switch field {
		case BilledFeeFieldFeeName, BilledFeeFieldCurrency, BilledFeeFieldExchangeRate, BilledFeeFieldQuantity, BilledFeeFieldUnitPrice, BilledFeeFieldTaxRate:
		default:
			return nil, ErrFinanceCustomSettingInvalidArgument
		}
		if _, duplicate := seen[field]; duplicate {
			return nil, ErrFinanceCustomSettingInvalidArgument
		}
		seen[field] = struct{}{}
		result = append(result, field)
	}
	return result, nil
}
