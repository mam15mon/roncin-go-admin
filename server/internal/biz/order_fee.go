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
	ErrOrderFeeNotFound                      = errors.NotFound("ORDER_FEE_NOT_FOUND", "订单费用不存在")
	ErrOrderFeeInvalidArgument               = errors.BadRequest("ORDER_FEE_INVALID_ARGUMENT", "订单费用字段不合法")
	ErrOrderFeePartyInvalid                  = errors.BadRequest("ORDER_FEE_PARTY_INVALID", "结算单位必须是当前组织启用的往来单位")
	ErrOrderFeeCurrencyInvalid               = errors.BadRequest("ORDER_FEE_CURRENCY_INVALID", "币种必须是启用的 ISO 币种")
	ErrOrderFeeSettingInvalid                = errors.BadRequest("ORDER_FEE_SETTING_INVALID", "费用设置不存在、已停用或不适用于当前订单")
	ErrOrderFeeBillingUnitInvalid            = errors.BadRequest("ORDER_FEE_BILLING_UNIT_INVALID", "计费单位不存在、已停用或不属于当前组织")
	ErrOrderFeeExchangeRateOverrideForbidden = errors.Forbidden("ORDER_FEE_EXCHANGE_RATE_OVERRIDE_FORBIDDEN", "无权手工覆盖费用汇率")
)

var (
	quantityOrPricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$`)
	totalAmountPattern     = regexp.MustCompile(`^(0|[1-9][0-9]{0,19})(\.[0-9]{1,8})?$`)
	currencyPattern        = regexp.MustCompile(`^[A-Za-z]{3}$`)
)

type OrderFeeDirection string

const (
	OrderFeeReceivable OrderFeeDirection = "RECEIVABLE"
	OrderFeePayable    OrderFeeDirection = "PAYABLE"
)

type OrderFee struct {
	ID                    uuid.UUID
	OrderID               uuid.UUID
	Direction             OrderFeeDirection
	FeeSettingID          *uuid.UUID
	FeeCode               string
	FeeName               string
	FeeNameEN             *string
	SettlementPartyID     uuid.UUID
	SettlementPartyName   string
	BillingUnitID         *uuid.UUID
	BillingUnit           string
	TaxRate               *decimal.Decimal
	TaxableServiceName    *string
	Quantity              decimal.Decimal
	UnitPrice             decimal.Decimal
	TotalAmount           decimal.Decimal
	Currency              string
	ExchangeRate          decimal.Decimal
	ExchangeRateSource    string
	ExchangeRateDate      string
	ExchangeRateSettingID *uuid.UUID
	ExchangeRateOverride  *decimal.Decimal
	ExpenseDate           string
	Note                  *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type OrderFeeSettlementPartyOption struct {
	ID   uuid.UUID
	Code string
	Name string
}

type OrderFeeCurrencyOption struct {
	Code      string
	Name      string
	MinorUnit int
}

type OrderFeeSettingOption struct {
	ID                     uuid.UUID
	FeeCode                string
	NameZH                 string
	NameEN                 *string
	AliasName              *string
	DefaultCurrency        string
	DefaultBillingUnitID   uuid.UUID
	DefaultBillingUnitName string
	TaxRate                decimal.Decimal
	TaxableServiceName     string
}

type OrderFeeBillingUnitOption struct {
	ID   uuid.UUID
	Code string
	Name string
}

type OrderFeeCatalogSnapshot struct {
	FeeCode            string
	FeeName            string
	FeeNameEN          *string
	BillingUnit        string
	TaxRate            decimal.Decimal
	TaxableServiceName string
}

type OrderFeeOptions struct {
	SettlementParties []OrderFeeSettlementPartyOption
	Currencies        []OrderFeeCurrencyOption
	FeeSettings       []OrderFeeSettingOption
	BillingUnits      []OrderFeeBillingUnitOption
	BaseCurrency      string
}

type OrderFeeExchangeRateContext struct {
	TradeDirection OrderTradeDirection
	ETD            string
	ETA            string
	BusinessTime   string
	OrderCreatedAt time.Time
}

type OrderFeeRepo interface {
	Options(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeOptions, error)
	ExchangeRateContext(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeExchangeRateContext, error)
	ResolveCatalog(ctx context.Context, organizationID, orderID, feeSettingID, billingUnitID uuid.UUID) (*OrderFeeCatalogSnapshot, error)
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderFee, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderFee, audit *AuditEvent) (*OrderFee, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderFee, audit *AuditEvent) (*OrderFee, error)
	Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *AuditEvent) error
}

type OrderFeeUsecase struct {
	repo         OrderFeeRepo
	exchangeRate *ExchangeRateUsecase
}

func NewOrderFeeUsecase(repo OrderFeeRepo, exchangeRate *ExchangeRateUsecase) *OrderFeeUsecase {
	return &OrderFeeUsecase{repo: repo, exchangeRate: exchangeRate}
}

func (uc *OrderFeeUsecase) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderFee, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, orderID)
}

func (uc *OrderFeeUsecase) Options(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeOptions, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	options, err := uc.repo.Options(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	options.BaseCurrency, err = uc.exchangeRate.BaseCurrency(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return options, nil
}

func (uc *OrderFeeUsecase) Add(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderFee, canOverrideExchangeRate bool) (*OrderFee, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	normalized, err := normalizeOrderFee(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized); err != nil {
		return nil, err
	}
	if err := uc.resolveExchangeRate(ctx, organizationID, orderID, normalized, canOverrideExchangeRate); err != nil {
		return nil, err
	}
	return uc.repo.Add(ctx, organizationID, orderID, normalized, orderFeeAudit(organizationID, actorID, orderID, normalized.ID, "order.fee.add", normalized))
}

func (uc *OrderFeeUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderFee, canOverrideExchangeRate bool) (*OrderFee, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	normalized, err := normalizeOrderFee(input)
	if err != nil {
		return nil, err
	}
	if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized); err != nil {
		return nil, err
	}
	if err := uc.resolveExchangeRate(ctx, organizationID, orderID, normalized, canOverrideExchangeRate); err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, orderID, id, normalized, orderFeeAudit(organizationID, actorID, orderID, id, "order.fee.update", normalized))
}

func (uc *OrderFeeUsecase) resolveCatalog(ctx context.Context, organizationID, orderID uuid.UUID, fee *OrderFee) error {
	snapshot, err := uc.repo.ResolveCatalog(ctx, organizationID, orderID, *fee.FeeSettingID, *fee.BillingUnitID)
	if err != nil {
		return err
	}
	fee.FeeCode = snapshot.FeeCode
	fee.FeeName = snapshot.FeeName
	fee.FeeNameEN = snapshot.FeeNameEN
	fee.BillingUnit = snapshot.BillingUnit
	fee.TaxRate = &snapshot.TaxRate
	fee.TaxableServiceName = &snapshot.TaxableServiceName
	return nil
}

func (uc *OrderFeeUsecase) ResolveExchangeRate(ctx context.Context, organizationID, orderID uuid.UUID, direction OrderFeeDirection, currency, expenseDate string) (*ResolvedExchangeRate, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	candidates, err := uc.exchangeRateDateCandidates(ctx, organizationID, orderID, expenseDate)
	if err != nil {
		return nil, err
	}
	return uc.exchangeRate.Resolve(ctx, organizationID, BaseCurrencyRateType, direction, currency, candidates)
}

func (uc *OrderFeeUsecase) resolveExchangeRate(ctx context.Context, organizationID, orderID uuid.UUID, fee *OrderFee, canOverrideExchangeRate bool) error {
	if fee.ExchangeRateOverride != nil {
		if !canOverrideExchangeRate {
			return ErrOrderFeeExchangeRateOverrideForbidden
		}
		fee.ExchangeRate = *fee.ExchangeRateOverride
		fee.ExchangeRateSource = "MANUAL"
		fee.ExchangeRateDate = fee.ExpenseDate
		fee.ExchangeRateSettingID = nil
		return nil
	}
	candidates, err := uc.exchangeRateDateCandidates(ctx, organizationID, orderID, fee.ExpenseDate)
	if err != nil {
		return err
	}
	resolved, err := uc.exchangeRate.Resolve(ctx, organizationID, BaseCurrencyRateType, fee.Direction, fee.Currency, candidates)
	if err != nil {
		return err
	}
	fee.ExchangeRate = resolved.Rate
	fee.ExchangeRateSource = resolved.Source
	fee.ExchangeRateDate = resolved.RateDate
	fee.ExchangeRateSettingID = resolved.SettingID
	return nil
}

func (uc *OrderFeeUsecase) exchangeRateDateCandidates(ctx context.Context, organizationID, orderID uuid.UUID, expenseDate string) (map[string]string, error) {
	rateContext, err := uc.repo.ExchangeRateContext(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	candidates := map[string]string{ExpenseTimeStandard: expenseDate}
	scheduleTime := rateContext.ETD
	if rateContext.TradeDirection == OrderTradeImport {
		scheduleTime = rateContext.ETA
	}
	for standard, value := range map[string]string{
		ETDETAOrTrainDateStandard: scheduleTime,
		BusinessTimeStandard:      rateContext.BusinessTime,
	} {
		if value == "" {
			continue
		}
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			return nil, ErrOrderFeeInvalidArgument
		}
		candidates[standard] = parsed.In(exchangeRateBusinessLocation).Format("2006-01-02")
	}
	candidates[OrderCreatedAtStandard] = rateContext.OrderCreatedAt.In(exchangeRateBusinessLocation).Format("2006-01-02")
	return candidates, nil
}

func (uc *OrderFeeUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return ErrOrderFeeInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.fee.remove",
		Result:         "success",
		Details: map[string]string{
			"fee.id":   id.String(),
			"order.id": orderID.String(),
		},
	})
}

func orderFeeAudit(organizationID, actorID, orderID, feeID uuid.UUID, action string, fee *OrderFee) *AuditEvent {
	return &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         action,
		Result:         "success",
		Details: map[string]string{
			"fee.id":                   feeID.String(),
			"order.id":                 orderID.String(),
			"fee.code":                 fee.FeeCode,
			"fee.direction":            string(fee.Direction),
			"fee.amount":               fee.TotalAmount.StringFixed(8),
			"fee.currency":             fee.Currency,
			"fee.exchange_rate_source": fee.ExchangeRateSource,
		},
	}
}

func normalizeOrderFee(input *OrderFee) (*OrderFee, error) {
	if input == nil || input.SettlementPartyID == uuid.Nil || input.FeeSettingID == nil || *input.FeeSettingID == uuid.Nil || input.BillingUnitID == nil || *input.BillingUnitID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	if input.Direction != OrderFeeReceivable && input.Direction != OrderFeePayable {
		return nil, ErrOrderFeeInvalidArgument
	}
	if !quantityOrPricePattern.MatchString(input.Quantity.String()) || !input.Quantity.IsPositive() {
		return nil, ErrOrderFeeInvalidArgument
	}
	if !quantityOrPricePattern.MatchString(input.UnitPrice.String()) || !input.UnitPrice.IsPositive() {
		return nil, ErrOrderFeeInvalidArgument
	}
	totalAmount := input.Quantity.Mul(input.UnitPrice)
	if !totalAmountPattern.MatchString(totalAmount.String()) || !totalAmount.IsPositive() {
		return nil, ErrOrderFeeInvalidArgument
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if !currencyPattern.MatchString(currency) {
		return nil, ErrOrderFeeInvalidArgument
	}
	if input.ExchangeRateOverride != nil && !validExchangeRate(*input.ExchangeRateOverride) {
		return nil, ErrOrderFeeInvalidArgument
	}
	expenseDate := strings.TrimSpace(input.ExpenseDate)
	parsedDate, err := time.Parse("2006-01-02", expenseDate)
	if err != nil || parsedDate.Format("2006-01-02") != expenseDate {
		return nil, ErrOrderFeeInvalidArgument
	}
	var note *string
	if input.Note != nil {
		value := strings.TrimSpace(*input.Note)
		if value != "" {
			if utf8.RuneCountInString(value) > 500 {
				return nil, ErrOrderFeeInvalidArgument
			}
			note = &value
		}
	}
	output := *input
	output.TotalAmount = totalAmount
	output.Currency = currency
	output.ExpenseDate = expenseDate
	output.Note = note
	return &output, nil
}
