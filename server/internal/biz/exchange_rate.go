package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrExchangeRateNotFound            = errors.NotFound("EXCHANGE_RATE_NOT_FOUND", "汇率设置不存在")
	ErrExchangeRateInvalidArgument     = errors.BadRequest("EXCHANGE_RATE_INVALID_ARGUMENT", "汇率设置字段不合法")
	ErrExchangeRateOverlap             = errors.Conflict("EXCHANGE_RATE_OVERLAP", "汇率生效区间与现有设置重叠")
	ErrExchangeRateMissing             = errors.BadRequest("FEE_EXCHANGE_RATE_MISSING", "汇率日期未命中生效汇率")
	ErrExchangeRateConflict            = errors.Conflict("FEE_EXCHANGE_RATE_CONFLICT", "汇率日期命中多条生效汇率")
	ErrExchangeRateCurrencyInvalid     = errors.BadRequest("EXCHANGE_RATE_CURRENCY_INVALID", "汇率币种必须是启用的 ISO 币种")
	ErrExchangeRateOrganizationInvalid = errors.BadRequest("EXCHANGE_RATE_ORGANIZATION_INVALID", "当前组织未配置有效本币")
	ErrExchangeRateDateMissing         = errors.BadRequest("EXCHANGE_RATE_DATE_MISSING", "未能按汇率类型时间标准确定汇率日期")
)

const (
	BaseCurrencyRateType      = "BASE_CURRENCY"
	InvoiceRateType           = "INVOICE"
	SettlementRateType        = "SETTLEMENT"
	WriteOffRateType          = "WRITE_OFF"
	BillRateType              = "BILL"
	ETDETAOrTrainDateStandard = "ETD_ETA_TRAIN_DATE"
	BusinessTimeStandard      = "BUSINESS_TIME"
	BargeETDStandard          = "BARGE_ETD"
	ExpenseTimeStandard       = "EXPENSE_TIME"
	OrderCreatedAtStandard    = "ORDER_CREATED_AT"
	BillDateStandard          = "BILL_DATE"
	InvoiceDateStandard       = "INVOICE_DATE"
	TransactionDateStandard   = "TRANSACTION_DATE"
	WriteOffTimeStandard      = "WRITE_OFF_TIME"
)

var exchangeRateValuePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$`)
var exchangeRateBusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type ExchangeRateSetting struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RateType       string
	FromCurrency   string
	ToCurrency     string
	EffectiveFrom  string
	EffectiveTo    *string
	ReceivableRate decimal.Decimal
	PayableRate    decimal.Decimal
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ExchangeRateTimeStandardSetting struct {
	RateType      string
	TimeStandards []string
}

type ResolvedExchangeRate struct {
	Rate      decimal.Decimal
	Source    string
	RateDate  string
	SettingID *uuid.UUID
}

type ExchangeRateContext struct {
	OwnerOrganizationID uuid.UUID
	BaseCurrency        string
}

type ExchangeRateRepo interface {
	ResolveContext(ctx context.Context, organizationID uuid.UUID) (*ExchangeRateContext, error)
	List(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateSetting, error)
	Create(ctx context.Context, input *ExchangeRateSetting, audit *AuditEvent) (*ExchangeRateSetting, error)
	Update(ctx context.Context, input *ExchangeRateSetting, audit *AuditEvent) (*ExchangeRateSetting, error)
	Disable(ctx context.Context, organizationID, id uuid.UUID, audit *AuditEvent) error
	ListTimeStandards(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error)
	ReplaceTimeStandards(ctx context.Context, organizationID uuid.UUID, settings []*ExchangeRateTimeStandardSetting, audit *AuditEvent) error
	Resolve(ctx context.Context, organizationID uuid.UUID, rateType string, direction OrderFeeDirection, fromCurrency, toCurrency, rateDate string) (*ResolvedExchangeRate, error)
}

type ExchangeRateUsecase struct{ repo ExchangeRateRepo }

func NewExchangeRateUsecase(repo ExchangeRateRepo) *ExchangeRateUsecase {
	return &ExchangeRateUsecase{repo: repo}
}

func (uc *ExchangeRateUsecase) List(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateSetting, string, error) {
	if organizationID == uuid.Nil {
		return nil, "", ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, "", err
	}
	items, err := uc.repo.List(ctx, rateContext.OwnerOrganizationID)
	return items, rateContext.BaseCurrency, err
}

func (uc *ExchangeRateUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if normalized.ToCurrency != rateContext.BaseCurrency {
		return nil, ErrExchangeRateCurrencyInvalid
	}
	normalized.OrganizationID = rateContext.OwnerOrganizationID
	normalized.IsActive = true
	return uc.repo.Create(ctx, normalized, exchangeRateAudit(organizationID, actorID, normalized.ID, "finance.exchange_rate.create"))
}

func (uc *ExchangeRateUsecase) Update(ctx context.Context, organizationID, actorID, id uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	normalized.ID = id
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if normalized.ToCurrency != rateContext.BaseCurrency {
		return nil, ErrExchangeRateCurrencyInvalid
	}
	normalized.OrganizationID = rateContext.OwnerOrganizationID
	return uc.repo.Update(ctx, normalized, exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.update"))
}

func (uc *ExchangeRateUsecase) Disable(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return err
	}
	return uc.repo.Disable(ctx, rateContext.OwnerOrganizationID, id, exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.disable"))
}

func (uc *ExchangeRateUsecase) ListTimeStandards(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateTimeStandardSetting, error) {
	if organizationID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListTimeStandards(ctx, rateContext.OwnerOrganizationID)
}

func (uc *ExchangeRateUsecase) UpdateTimeStandards(ctx context.Context, organizationID, actorID uuid.UUID, settings []*ExchangeRateTimeStandardSetting) ([]*ExchangeRateTimeStandardSetting, error) {
	normalized, err := normalizeExchangeRateTimeStandards(settings)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	audit := exchangeRateTimeStandardAudit(organizationID, actorID, rateContext.OwnerOrganizationID)
	if err := uc.repo.ReplaceTimeStandards(ctx, rateContext.OwnerOrganizationID, normalized, audit); err != nil {
		return nil, err
	}
	return uc.repo.ListTimeStandards(ctx, rateContext.OwnerOrganizationID)
}

func (uc *ExchangeRateUsecase) Resolve(ctx context.Context, organizationID uuid.UUID, rateType string, direction OrderFeeDirection, currency string, dateCandidates map[string]string) (*ResolvedExchangeRate, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !validExchangeRateType(rateType) || !currencyPattern.MatchString(currency) || (direction != OrderFeeReceivable && direction != OrderFeePayable) {
		return nil, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	settings, err := uc.repo.ListTimeStandards(ctx, rateContext.OwnerOrganizationID)
	if err != nil {
		return nil, err
	}
	rateDate := resolveExchangeRateDate(rateType, settings, dateCandidates)
	if rateDate == "" {
		return nil, ErrExchangeRateDateMissing
	}
	if currency == rateContext.BaseCurrency {
		return &ResolvedExchangeRate{Rate: decimal.NewFromInt(1), Source: "BASE_CURRENCY", RateDate: rateDate}, nil
	}
	return uc.repo.Resolve(ctx, rateContext.OwnerOrganizationID, rateType, direction, currency, rateContext.BaseCurrency, rateDate)
}

func (uc *ExchangeRateUsecase) BaseCurrency(ctx context.Context, organizationID uuid.UUID) (string, error) {
	if organizationID == uuid.Nil {
		return "", ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return "", err
	}
	return rateContext.BaseCurrency, nil
}

func normalizeExchangeRateSetting(input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	if input == nil || !validExchangeRateType(input.RateType) {
		return nil, ErrExchangeRateInvalidArgument
	}
	fromCurrency := strings.ToUpper(strings.TrimSpace(input.FromCurrency))
	toCurrency := strings.ToUpper(strings.TrimSpace(input.ToCurrency))
	if !currencyPattern.MatchString(fromCurrency) || !currencyPattern.MatchString(toCurrency) || fromCurrency == toCurrency {
		return nil, ErrExchangeRateInvalidArgument
	}
	effectiveFrom := strings.TrimSpace(input.EffectiveFrom)
	if !validISODate(effectiveFrom) {
		return nil, ErrExchangeRateInvalidArgument
	}
	var effectiveTo *string
	if input.EffectiveTo != nil {
		value := strings.TrimSpace(*input.EffectiveTo)
		if !validISODate(value) || value <= effectiveFrom {
			return nil, ErrExchangeRateInvalidArgument
		}
		effectiveTo = &value
	}
	if !validExchangeRate(input.ReceivableRate) || !validExchangeRate(input.PayableRate) {
		return nil, ErrExchangeRateInvalidArgument
	}
	output := *input
	output.FromCurrency = fromCurrency
	output.ToCurrency = toCurrency
	output.EffectiveFrom = effectiveFrom
	output.EffectiveTo = effectiveTo
	return &output, nil
}

func normalizeExchangeRateTimeStandards(settings []*ExchangeRateTimeStandardSetting) ([]*ExchangeRateTimeStandardSetting, error) {
	if len(settings) != 5 {
		return nil, ErrExchangeRateInvalidArgument
	}
	seenTypes := make(map[string]struct{}, len(settings))
	result := make([]*ExchangeRateTimeStandardSetting, 0, len(settings))
	for _, setting := range settings {
		if setting == nil || !validExchangeRateType(setting.RateType) || len(setting.TimeStandards) == 0 {
			return nil, ErrExchangeRateInvalidArgument
		}
		if _, exists := seenTypes[setting.RateType]; exists {
			return nil, ErrExchangeRateInvalidArgument
		}
		seenTypes[setting.RateType] = struct{}{}
		allowed := allowedTimeStandards(setting.RateType)
		seenStandards := make(map[string]struct{}, len(setting.TimeStandards))
		copySetting := &ExchangeRateTimeStandardSetting{RateType: setting.RateType, TimeStandards: make([]string, 0, len(setting.TimeStandards))}
		for _, standard := range setting.TimeStandards {
			if _, valid := allowed[standard]; !valid {
				return nil, ErrExchangeRateInvalidArgument
			}
			if _, duplicate := seenStandards[standard]; duplicate {
				return nil, ErrExchangeRateInvalidArgument
			}
			seenStandards[standard] = struct{}{}
			copySetting.TimeStandards = append(copySetting.TimeStandards, standard)
		}
		result = append(result, copySetting)
	}
	return result, nil
}

func allowedTimeStandards(rateType string) map[string]struct{} {
	values := []string{}
	switch rateType {
	case BaseCurrencyRateType:
		values = []string{ETDETAOrTrainDateStandard, BusinessTimeStandard, BargeETDStandard, OrderCreatedAtStandard}
	case InvoiceRateType:
		values = []string{InvoiceDateStandard}
	case BillRateType:
		values = []string{BillDateStandard}
	case SettlementRateType:
		values = []string{TransactionDateStandard}
	case WriteOffRateType:
		values = []string{WriteOffTimeStandard}
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func resolveExchangeRateDate(rateType string, settings []*ExchangeRateTimeStandardSetting, candidates map[string]string) string {
	for _, setting := range settings {
		if setting.RateType != rateType {
			continue
		}
		for _, standard := range setting.TimeStandards {
			if value := candidates[standard]; validISODate(value) {
				return value
			}
		}
		return ""
	}
	return ""
}

func validExchangeRateType(value string) bool {
	switch value {
	case BaseCurrencyRateType, InvoiceRateType, SettlementRateType, WriteOffRateType, BillRateType:
		return true
	default:
		return false
	}
}

func validExchangeRate(value decimal.Decimal) bool {
	return value.IsPositive() && exchangeRateValuePattern.MatchString(value.String())
}

func validISODate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func exchangeRateAudit(organizationID, actorID, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "exchange_rate_setting", ResourceID: id.String()}
}

func exchangeRateTimeStandardAudit(organizationID, actorID, ownerOrganizationID uuid.UUID) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "finance.exchange_rate.time_standard.update", Result: "success", ResourceType: "exchange_rate_time_standard", ResourceID: ownerOrganizationID.String()}
}
