package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/shopspring/decimal"
)

var (
	ErrExchangeRateNotFound             = errors.NotFound("EXCHANGE_RATE_NOT_FOUND", "汇率设置不存在")
	ErrExchangeRateInvalidArgument      = errors.BadRequest("EXCHANGE_RATE_INVALID_ARGUMENT", "汇率设置字段不合法")
	ErrExchangeRateOverlap              = errors.Conflict("EXCHANGE_RATE_OVERLAP", "汇率生效区间与现有设置重叠")
	ErrExchangeRateMissing              = errors.BadRequest(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FEE_EXCHANGE_RATE_MISSING), "汇率日期未命中生效汇率")
	ErrExchangeRateConflict             = errors.Conflict("FEE_EXCHANGE_RATE_CONFLICT", "汇率日期命中多条生效汇率")
	ErrExchangeRateCurrencyInvalid      = errors.BadRequest("EXCHANGE_RATE_CURRENCY_INVALID", "汇率币种必须是启用的 ISO 币种")
	ErrExchangeRateOrganizationInvalid  = errors.BadRequest("EXCHANGE_RATE_ORGANIZATION_INVALID", "当前组织未配置有效本币")
	ErrExchangeRateHeadquartersRequired = errors.Forbidden("EXCHANGE_RATE_HEADQUARTERS_REQUIRED", "折本币基准汇率只能由总部维护")
)

var exchangeRateValuePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$`)
var exchangeRateBusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func ExchangeRateBusinessLocation() *time.Location { return exchangeRateBusinessLocation }

// ExchangeRateSetting 是总部维护的折本币基准汇率。
type ExchangeRateSetting struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FromCurrency   string
	ToCurrency     string
	EffectiveFrom  string
	EffectiveTo    *string
	Rate           decimal.Decimal
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExchangeRateContext 携带组织树根（总部）与基准币种；汇率统一归属总部维护。
type ExchangeRateContext struct {
	OwnerOrganizationID uuid.UUID
	BaseCurrency        string
}

type ExchangeRateRepo interface {
	ResolveContext(ctx context.Context, organizationID uuid.UUID) (*ExchangeRateContext, error)
	List(ctx context.Context, organizationID uuid.UUID) ([]*ExchangeRateSetting, error)
	Create(ctx context.Context, organizationID uuid.UUID, input *ExchangeRateSetting, audit *AuditEvent) (*ExchangeRateSetting, error)
	Update(ctx context.Context, organizationID uuid.UUID, input *ExchangeRateSetting, audit *AuditEvent) (*ExchangeRateSetting, error)
	Disable(ctx context.Context, organizationID, id uuid.UUID, audit *AuditEvent) error
	ResolveRate(ctx context.Context, ownerOrganizationID uuid.UUID, fromCurrency, toCurrency, rateDate string) (decimal.Decimal, error)
	InspectImport(ctx context.Context, ownerOrganizationID uuid.UUID, rows []*ExchangeRateImportRow) (map[int][]string, error)
	CreateImportPreview(ctx context.Context, batch *ExchangeRateImportBatch, audit *AuditEvent) (*ExchangeRateImportBatch, error)
	GetImport(ctx context.Context, organizationID, id uuid.UUID) (*ExchangeRateImportBatch, error)
	ConfirmImport(ctx context.Context, organizationID, ownerOrganizationID, actorID uuid.UUID, previewTokenHash, idempotencyKey string, now time.Time, audit *AuditEvent) (*ExchangeRateImportBatch, error)
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
	// 汇率只在总部落地；非总部调用由仓储层拒绝，不再重定向写入总部行。
	normalized.OrganizationID = organizationID
	normalized.IsActive = true
	return uc.repo.Create(ctx, organizationID, normalized, exchangeRateAudit(organizationID, actorID, normalized.ID, "finance.exchange_rate.create"))
}

func (uc *ExchangeRateUsecase) Update(ctx context.Context, organizationID, actorID uuid.UUID, id uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
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
	normalized.OrganizationID = organizationID
	return uc.repo.Update(ctx, organizationID, normalized, exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.update"))
}

func (uc *ExchangeRateUsecase) Disable(ctx context.Context, organizationID, actorID uuid.UUID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrExchangeRateInvalidArgument
	}
	return uc.repo.Disable(ctx, organizationID, id, exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.disable"))
}

// ResolveRate 按目标日期解析 currency 折组织基准币种的总部基准汇率；
// currency 即基准币种时恒为 1，未命中生效区间时返回 ErrExchangeRateMissing。
func (uc *ExchangeRateUsecase) ResolveRate(ctx context.Context, organizationID uuid.UUID, currency, targetDate string) (decimal.Decimal, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !currencyPattern.MatchString(currency) {
		return decimal.Decimal{}, ErrExchangeRateInvalidArgument
	}
	if _, valid := parseExchangeRateLookupTime(targetDate); !valid {
		return decimal.Decimal{}, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if currency == rateContext.BaseCurrency {
		return decimal.NewFromInt(1), nil
	}
	return uc.repo.ResolveRate(ctx, rateContext.OwnerOrganizationID, currency, rateContext.BaseCurrency, targetDate)
}

// ResolveBaseRate 按目标日期在总部基准汇率表解析 fromCurrency → toCurrency 的
// 正向汇率；两币相同恒为 1，不要求目标币种等于组织基准币种。提成 CNY 折算用它
// 解析「本位币 → CNY」，避免总部只维护 X→CNY 基准时反查 CNY→X 报缺失。
// 与 ResolveRate 一致，先在事务内读取组织汇率上下文再短路，保证同事务内的
// 组织上下文读取语义不变。
func (uc *ExchangeRateUsecase) ResolveBaseRate(ctx context.Context, organizationID uuid.UUID, fromCurrency, toCurrency, targetDate string) (decimal.Decimal, error) {
	fromCurrency = strings.ToUpper(strings.TrimSpace(fromCurrency))
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if organizationID == uuid.Nil || !currencyPattern.MatchString(fromCurrency) || !currencyPattern.MatchString(toCurrency) {
		return decimal.Decimal{}, ErrExchangeRateInvalidArgument
	}
	if _, valid := parseExchangeRateLookupTime(targetDate); !valid {
		return decimal.Decimal{}, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if fromCurrency == toCurrency {
		return decimal.NewFromInt(1), nil
	}
	return uc.repo.ResolveRate(ctx, rateContext.OwnerOrganizationID, fromCurrency, toCurrency, targetDate)
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
	if input == nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	fromCurrency := strings.ToUpper(strings.TrimSpace(input.FromCurrency))
	toCurrency := strings.ToUpper(strings.TrimSpace(input.ToCurrency))
	if !currencyPattern.MatchString(fromCurrency) || !currencyPattern.MatchString(toCurrency) || fromCurrency == toCurrency {
		return nil, ErrExchangeRateInvalidArgument
	}
	effectiveFrom := strings.TrimSpace(input.EffectiveFrom)
	normalizedFrom, fromTime, valid := normalizeExchangeRateTimestamp(effectiveFrom)
	if !valid {
		return nil, ErrExchangeRateInvalidArgument
	}
	var effectiveTo *string
	if input.EffectiveTo != nil {
		value := strings.TrimSpace(*input.EffectiveTo)
		normalizedTo, toTime, validTo := normalizeExchangeRateTimestamp(value)
		if !validTo || !toTime.After(fromTime) {
			return nil, ErrExchangeRateInvalidArgument
		}
		effectiveTo = &normalizedTo
	}
	if !validExchangeRate(input.Rate) {
		return nil, ErrExchangeRateInvalidArgument
	}
	output := *input
	output.FromCurrency = fromCurrency
	output.ToCurrency = toCurrency
	output.EffectiveFrom = normalizedFrom
	output.EffectiveTo = effectiveTo
	return &output, nil
}

func validExchangeRate(value decimal.Decimal) bool {
	return value.IsPositive() && exchangeRateValuePattern.MatchString(value.String())
}

// normalizeExchangeRateTimestamp 将外部时间统一为上海时区的秒级 RFC 3339。
// 汇率有效期不接受无时区时间或小数秒，避免不同客户端产生不一致的区间边界。
func normalizeExchangeRateTimestamp(value string) (string, time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil || parsed.Nanosecond() != 0 {
		return "", time.Time{}, false
	}
	parsed = parsed.In(exchangeRateBusinessLocation)
	return parsed.Format(time.RFC3339), parsed, true
}

// parseExchangeRateLookupTime 接受 YYYY-MM-DD（按业务时区解释）或带时区秒级 RFC 3339。
func parseExchangeRateLookupTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	lookup, err := time.ParseInLocation("2006-01-02", value, exchangeRateBusinessLocation)
	if err == nil && lookup.Format("2006-01-02") == value {
		return lookup, true
	}
	_, parsed, valid := normalizeExchangeRateTimestamp(value)
	return parsed, valid
}

func exchangeRateAudit(organizationID, actorID, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "exchange_rate_setting", ResourceID: id.String()}
}
