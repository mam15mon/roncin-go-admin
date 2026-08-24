package biz

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFeeCatalogInvalidArgument  = errors.BadRequest("FEE_CATALOG_INVALID_ARGUMENT", "费用设置字段不合法")
	ErrFeeSettingNotFound         = errors.NotFound("FEE_SETTING_NOT_FOUND", "费用设置不存在")
	ErrFeeSettingCodeExists       = errors.Conflict("FEE_SETTING_CODE_EXISTS", "费用代码已存在")
	ErrBillingUnitNotFound        = errors.NotFound("BILLING_UNIT_NOT_FOUND", "计费单位不存在")
	ErrBillingUnitCodeExists      = errors.Conflict("BILLING_UNIT_CODE_EXISTS", "计费单位代码已存在")
	ErrTaxableServiceNotFound     = errors.NotFound("TAXABLE_SERVICE_NOT_FOUND", "货物或应税劳务名称不存在")
	ErrTaxableServiceNameExists   = errors.Conflict("TAXABLE_SERVICE_NAME_EXISTS", "货物或应税劳务名称已存在")
	ErrFeeCatalogReferenceInvalid = errors.BadRequest("FEE_CATALOG_REFERENCE_INVALID", "费用设置引用的基础资料不存在、已停用或不属于当前组织")
)

var catalogCodePattern = regexp.MustCompile(`^[A-Z0-9_]{2,32}$`)

type BillingUnit struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	SortOrder      int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TaxableService struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	ShortName      *string
	GoodsCode      *string
	DefaultTaxRate decimal.Decimal
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type FeeSetting struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	FeeCode            string
	NameZH             string
	NameEN             *string
	AliasName          *string
	ServiceTypeID      *uuid.UUID
	ServiceTypeName    *string
	DefaultCurrency    string
	BillingUnitID      uuid.UUID
	BillingUnitName    string
	AbnormalCaseID     *uuid.UUID
	AbnormalCaseName   *string
	TaxRate            decimal.Decimal
	TaxableServiceID   uuid.UUID
	TaxableServiceName string
	Enabled            bool
	SortOrder          int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FeeCatalogRepo interface {
	ListFeeSettings(context.Context, uuid.UUID) ([]*FeeSetting, error)
	CreateFeeSetting(context.Context, *FeeSetting, *AuditEvent) (*FeeSetting, error)
	UpdateFeeSetting(context.Context, *FeeSetting, *AuditEvent) (*FeeSetting, error)
	ListBillingUnits(context.Context, uuid.UUID) ([]*BillingUnit, error)
	CreateBillingUnit(context.Context, *BillingUnit, *AuditEvent) (*BillingUnit, error)
	UpdateBillingUnit(context.Context, *BillingUnit, *AuditEvent) (*BillingUnit, error)
	ListTaxableServices(context.Context, uuid.UUID) ([]*TaxableService, error)
	CreateTaxableService(context.Context, *TaxableService, *AuditEvent) (*TaxableService, error)
	UpdateTaxableService(context.Context, *TaxableService, *AuditEvent) (*TaxableService, error)
}

type FeeCatalogUsecase struct{ repo FeeCatalogRepo }

func NewFeeCatalogUsecase(repo FeeCatalogRepo) *FeeCatalogUsecase {
	return &FeeCatalogUsecase{repo: repo}
}

func (uc *FeeCatalogUsecase) ListFeeSettings(ctx context.Context, organizationID uuid.UUID) ([]*FeeSetting, error) {
	if organizationID == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	return uc.repo.ListFeeSettings(ctx, organizationID)
}

func (uc *FeeCatalogUsecase) CreateFeeSetting(ctx context.Context, organizationID, actorID uuid.UUID, input *FeeSetting) (*FeeSetting, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	normalized, err := normalizeFeeSetting(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.OrganizationID = organizationID
	normalized.Enabled = true
	return uc.repo.CreateFeeSetting(ctx, normalized, feeCatalogAudit(organizationID, actorID, normalized.ID, "finance.fee_setting.create", "fee_setting"))
}

func (uc *FeeCatalogUsecase) UpdateFeeSetting(ctx context.Context, organizationID, actorID, id uuid.UUID, input *FeeSetting) (*FeeSetting, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	normalized, err := normalizeFeeSetting(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = id
	normalized.OrganizationID = organizationID
	return uc.repo.UpdateFeeSetting(ctx, normalized, feeCatalogAudit(organizationID, actorID, id, "finance.fee_setting.update", "fee_setting"))
}

func (uc *FeeCatalogUsecase) ListBillingUnits(ctx context.Context, organizationID uuid.UUID) ([]*BillingUnit, error) {
	if organizationID == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	return uc.repo.ListBillingUnits(ctx, organizationID)
}

func (uc *FeeCatalogUsecase) CreateBillingUnit(ctx context.Context, organizationID, actorID uuid.UUID, input *BillingUnit) (*BillingUnit, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	normalized, err := normalizeBillingUnit(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.OrganizationID = organizationID
	normalized.Enabled = true
	return uc.repo.CreateBillingUnit(ctx, normalized, feeCatalogAudit(organizationID, actorID, normalized.ID, "finance.billing_unit.create", "billing_unit"))
}

func (uc *FeeCatalogUsecase) UpdateBillingUnit(ctx context.Context, organizationID, actorID, id uuid.UUID, input *BillingUnit) (*BillingUnit, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	normalized, err := normalizeBillingUnit(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = id
	normalized.OrganizationID = organizationID
	return uc.repo.UpdateBillingUnit(ctx, normalized, feeCatalogAudit(organizationID, actorID, id, "finance.billing_unit.update", "billing_unit"))
}

func (uc *FeeCatalogUsecase) ListTaxableServices(ctx context.Context, organizationID uuid.UUID) ([]*TaxableService, error) {
	if organizationID == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	return uc.repo.ListTaxableServices(ctx, organizationID)
}

func (uc *FeeCatalogUsecase) CreateTaxableService(ctx context.Context, organizationID, actorID uuid.UUID, input *TaxableService) (*TaxableService, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	normalized, err := normalizeTaxableService(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.OrganizationID = organizationID
	normalized.Enabled = true
	return uc.repo.CreateTaxableService(ctx, normalized, feeCatalogAudit(organizationID, actorID, normalized.ID, "finance.taxable_service.create", "taxable_service"))
}

func (uc *FeeCatalogUsecase) UpdateTaxableService(ctx context.Context, organizationID, actorID, id uuid.UUID, input *TaxableService) (*TaxableService, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	normalized, err := normalizeTaxableService(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = id
	normalized.OrganizationID = organizationID
	return uc.repo.UpdateTaxableService(ctx, normalized, feeCatalogAudit(organizationID, actorID, id, "finance.taxable_service.update", "taxable_service"))
}

func normalizeFeeSetting(input *FeeSetting) (*FeeSetting, error) {
	if input == nil || input.BillingUnitID == uuid.Nil || input.TaxableServiceID == uuid.Nil || input.SortOrder < 0 {
		return nil, ErrFeeCatalogInvalidArgument
	}
	output := *input
	output.FeeCode = strings.TrimSpace(input.FeeCode)
	output.NameZH = strings.TrimSpace(input.NameZH)
	output.DefaultCurrency = strings.ToUpper(strings.TrimSpace(input.DefaultCurrency))
	output.NameEN = normalizeCatalogOptionalText(input.NameEN)
	output.AliasName = normalizeCatalogOptionalText(input.AliasName)
	if !catalogCodePattern.MatchString(output.FeeCode) || output.NameZH == "" || utf8.RuneCountInString(output.NameZH) > 64 || optionalTextTooLong(output.NameEN, 128) || optionalTextTooLong(output.AliasName, 64) || !currencyPattern.MatchString(output.DefaultCurrency) || !validTaxRate(output.TaxRate) {
		return nil, ErrFeeCatalogInvalidArgument
	}
	return &output, nil
}

func normalizeBillingUnit(input *BillingUnit) (*BillingUnit, error) {
	if input == nil || input.SortOrder < 0 {
		return nil, ErrFeeCatalogInvalidArgument
	}
	output := *input
	output.Code = strings.TrimSpace(input.Code)
	output.Name = strings.TrimSpace(input.Name)
	if !catalogCodePattern.MatchString(output.Code) || output.Name == "" || utf8.RuneCountInString(output.Name) > 64 {
		return nil, ErrFeeCatalogInvalidArgument
	}
	return &output, nil
}

func normalizeTaxableService(input *TaxableService) (*TaxableService, error) {
	if input == nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	output := *input
	output.Name = strings.TrimSpace(input.Name)
	output.ShortName = normalizeCatalogOptionalText(input.ShortName)
	output.GoodsCode = normalizeCatalogOptionalText(input.GoodsCode)
	if output.Name == "" || utf8.RuneCountInString(output.Name) > 128 || optionalTextTooLong(output.ShortName, 64) || optionalTextTooLong(output.GoodsCode, 64) || !validTaxRate(output.DefaultTaxRate) {
		return nil, ErrFeeCatalogInvalidArgument
	}
	return &output, nil
}

func normalizeCatalogOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalTextTooLong(value *string, limit int) bool {
	return value != nil && utf8.RuneCountInString(*value) > limit
}

func validTaxRate(value decimal.Decimal) bool {
	return !value.IsNegative() && value.LessThanOrEqual(decimal.NewFromInt(100)) && value.Exponent() >= -2
}

func feeCatalogAudit(organizationID, actorID, id uuid.UUID, action, resourceType string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: resourceType, ResourceID: id.String()}
}
