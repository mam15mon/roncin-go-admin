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
	ErrOrderFeeNotFound        = errors.NotFound("ORDER_FEE_NOT_FOUND", "订单费用不存在")
	ErrOrderFeeInvalidArgument = errors.BadRequest("ORDER_FEE_INVALID_ARGUMENT", "订单费用字段不合法")
	ErrOrderFeePartyInvalid    = errors.BadRequest("ORDER_FEE_PARTY_INVALID", "结算单位必须是当前组织启用的往来单位")
	ErrOrderFeeCurrencyInvalid = errors.BadRequest("ORDER_FEE_CURRENCY_INVALID", "币种必须是启用的 ISO 币种")
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
	FeeCode               string
	FeeName               string
	SettlementPartyID     uuid.UUID
	SettlementPartyName   string
	BillingUnit           string
	Quantity              decimal.Decimal
	UnitPrice             decimal.Decimal
	TotalAmount           decimal.Decimal
	Currency              string
	ExchangeRate          decimal.Decimal
	ExchangeRateSource    string
	ExchangeRateDate      string
	ExchangeRateSettingID *uuid.UUID
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

type OrderFeeOptions struct {
	SettlementParties []OrderFeeSettlementPartyOption
	Currencies        []OrderFeeCurrencyOption
	BaseCurrency      string
}

type OrderFeeRepo interface {
	Options(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeOptions, error)
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

func (uc *OrderFeeUsecase) Add(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *OrderFee) (*OrderFee, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	normalized, err := normalizeOrderFee(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	if err := uc.resolveExchangeRate(ctx, organizationID, normalized); err != nil {
		return nil, err
	}
	return uc.repo.Add(ctx, organizationID, orderID, normalized, orderFeeAudit(organizationID, actorID, orderID, normalized.ID, "order.fee.add", normalized))
}

func (uc *OrderFeeUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderFee) (*OrderFee, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	normalized, err := normalizeOrderFee(input)
	if err != nil {
		return nil, err
	}
	if err := uc.resolveExchangeRate(ctx, organizationID, normalized); err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, orderID, id, normalized, orderFeeAudit(organizationID, actorID, orderID, id, "order.fee.update", normalized))
}

func (uc *OrderFeeUsecase) ResolveExchangeRate(ctx context.Context, organizationID, orderID uuid.UUID, direction OrderFeeDirection, currency, expenseDate string) (*ResolvedExchangeRate, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	return uc.exchangeRate.Resolve(ctx, organizationID, direction, currency, expenseDate)
}

func (uc *OrderFeeUsecase) resolveExchangeRate(ctx context.Context, organizationID uuid.UUID, fee *OrderFee) error {
	resolved, err := uc.exchangeRate.Resolve(ctx, organizationID, fee.Direction, fee.Currency, fee.ExpenseDate)
	if err != nil {
		return err
	}
	fee.ExchangeRate = resolved.Rate
	fee.ExchangeRateSource = resolved.Source
	fee.ExchangeRateDate = resolved.RateDate
	fee.ExchangeRateSettingID = resolved.SettingID
	return nil
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
			"fee.id":        feeID.String(),
			"order.id":      orderID.String(),
			"fee.code":      fee.FeeCode,
			"fee.direction": string(fee.Direction),
			"fee.amount":    fee.TotalAmount.StringFixed(8),
			"fee.currency":  fee.Currency,
		},
	}
}

func normalizeOrderFee(input *OrderFee) (*OrderFee, error) {
	if input == nil || input.SettlementPartyID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	if input.Direction != OrderFeeReceivable && input.Direction != OrderFeePayable {
		return nil, ErrOrderFeeInvalidArgument
	}
	feeCode := strings.ToUpper(strings.TrimSpace(input.FeeCode))
	feeName := strings.TrimSpace(input.FeeName)
	billingUnit := strings.TrimSpace(input.BillingUnit)
	if feeCode == "" || utf8.RuneCountInString(feeCode) > 30 || feeName == "" || utf8.RuneCountInString(feeName) > 80 || billingUnit == "" || utf8.RuneCountInString(billingUnit) > 32 {
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
	output.FeeCode = feeCode
	output.FeeName = feeName
	output.BillingUnit = billingUnit
	output.TotalAmount = totalAmount
	output.Currency = currency
	output.ExpenseDate = expenseDate
	output.Note = note
	return &output, nil
}
