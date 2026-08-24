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
	ErrExchangeRateNotFound        = errors.NotFound("EXCHANGE_RATE_NOT_FOUND", "汇率设置不存在")
	ErrExchangeRateInvalidArgument = errors.BadRequest("EXCHANGE_RATE_INVALID_ARGUMENT", "汇率设置字段不合法")
	ErrExchangeRateOverlap         = errors.Conflict("EXCHANGE_RATE_OVERLAP", "汇率生效区间与现有设置重叠")
	ErrExchangeRateMissing         = errors.BadRequest("FEE_EXCHANGE_RATE_MISSING", "费用日期未命中生效汇率")
	ErrExchangeRateConflict        = errors.Conflict("FEE_EXCHANGE_RATE_CONFLICT", "费用日期命中多条生效汇率")
	ErrExchangeRateCurrencyInvalid = errors.BadRequest("EXCHANGE_RATE_CURRENCY_INVALID", "汇率币种必须是启用的 ISO 币种")
)

const (
	SettlementRateType  = "SETTLEMENT"
	ExpenseDateStandard = "EXPENSE_DATE"
	BaseCurrencyCode    = "CNY"
)

var exchangeRateValuePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$`)

type ExchangeRateSetting struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	RateType       string
	FromCurrency   string
	ToCurrency     string
	TimeStandard   string
	EffectiveFrom  string
	EffectiveTo    *string
	ReceivableRate decimal.Decimal
	PayableRate    decimal.Decimal
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ResolvedExchangeRate struct {
	Rate      decimal.Decimal
	Source    string
	RateDate  string
	SettingID *uuid.UUID
}

type ExchangeRateRepo interface {
	List(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateSetting, error)
	Create(ctx context.Context, input *ExchangeRateSetting, audit *AuditEvent) (*ExchangeRateSetting, error)
	Update(ctx context.Context, input *ExchangeRateSetting, audit *AuditEvent) (*ExchangeRateSetting, error)
	Disable(ctx context.Context, organizationID, id uuid.UUID, audit *AuditEvent) error
	Resolve(ctx context.Context, organizationID uuid.UUID, direction OrderFeeDirection, fromCurrency, toCurrency, rateDate string) (*ResolvedExchangeRate, error)
}

type ExchangeRateUsecase struct{ repo ExchangeRateRepo }

func NewExchangeRateUsecase(repo ExchangeRateRepo) *ExchangeRateUsecase {
	return &ExchangeRateUsecase{repo: repo}
}

func (uc *ExchangeRateUsecase) List(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateSetting, error) {
	if organizationID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	return uc.repo.List(ctx, organizationID)
}

func (uc *ExchangeRateUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.OrganizationID = organizationID
	normalized.IsActive = true
	return uc.repo.Create(ctx, normalized, exchangeRateAudit(organizationID, actorID, normalized.ID, "finance.exchange_rate.create"))
}

func (uc *ExchangeRateUsecase) Update(ctx context.Context, organizationID, actorID, id uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	normalized.ID = id
	normalized.OrganizationID = organizationID
	return uc.repo.Update(ctx, normalized, exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.update"))
}

func (uc *ExchangeRateUsecase) Disable(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrExchangeRateInvalidArgument
	}
	return uc.repo.Disable(ctx, organizationID, id, exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.disable"))
}

func (uc *ExchangeRateUsecase) Resolve(ctx context.Context, organizationID uuid.UUID, direction OrderFeeDirection, currency, rateDate string) (*ResolvedExchangeRate, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !currencyPattern.MatchString(currency) || (direction != OrderFeeReceivable && direction != OrderFeePayable) || !validISODate(rateDate) {
		return nil, ErrExchangeRateInvalidArgument
	}
	if currency == BaseCurrencyCode {
		return &ResolvedExchangeRate{Rate: decimal.NewFromInt(1), Source: "BASE_CURRENCY", RateDate: rateDate}, nil
	}
	return uc.repo.Resolve(ctx, organizationID, direction, currency, BaseCurrencyCode, rateDate)
}

func normalizeExchangeRateSetting(input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	if input == nil || input.RateType != SettlementRateType || input.TimeStandard != ExpenseDateStandard {
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
