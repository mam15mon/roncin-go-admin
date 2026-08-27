package biz

import (
	"context"
	"fmt"
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
	ErrOrderFeeVersionConflict               = errors.Conflict("ORDER_FEE_VERSION_CONFLICT", "订单费用已被其他操作人修改，请刷新后重试")
	ErrOrderFeeInvalidTransition             = errors.Conflict("ORDER_FEE_INVALID_TRANSITION", "当前费用状态不允许执行该操作")
	ErrOrderFeeIdempotencyConflict           = errors.Conflict("ORDER_FEE_IDEMPOTENCY_CONFLICT", "费用请求幂等键已被使用")
	ErrOrderFeeFinanceLocked                 = errors.Conflict("ORDER_FEE_FINANCE_LOCKED", "订单已因确认或发放提成进入财务锁定，请通过提成调整记录处理后续差异")
)

var (
	quantityOrPricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$`)
	totalAmountPattern     = regexp.MustCompile(`^(0|[1-9][0-9]{0,19})(\.[0-9]{1,8})?$`)
	currencyPattern        = regexp.MustCompile(`^[A-Za-z]{3}$`)
	taxRatePattern         = regexp.MustCompile(`^(0|[1-9][0-9]{0,2})(\.[0-9]{1,2})?$`)
)

type OrderFeeDirection string
type OrderFeeStatus string

const (
	OrderFeeReceivable OrderFeeDirection = "RECEIVABLE"
	OrderFeePayable    OrderFeeDirection = "PAYABLE"
	OrderFeeDraft      OrderFeeStatus    = "DRAFT"
	OrderFeeConfirmed  OrderFeeStatus    = "CONFIRMED"
	OrderFeeBilled     OrderFeeStatus    = "BILLED"
	OrderFeeCancelled  OrderFeeStatus    = "CANCELLED"
)

type OrderFee struct {
	ID                    uuid.UUID
	OrderID               uuid.UUID
	IdempotencyKey        string
	Direction             OrderFeeDirection
	Status                OrderFeeStatus
	FeeSettingID          *uuid.UUID
	FeeCode               string
	FeeName               string
	FeeNameEN             *string
	SettlementPartyID     uuid.UUID
	SettlementPartyName   string
	BillingUnitID         *uuid.UUID
	BillingUnit           string
	TaxRate               *decimal.Decimal
	TaxRateOverride       *decimal.Decimal
	FeeNameOverride       *string
	TaxableServiceName    *string
	Quantity              decimal.Decimal
	UnitPrice             decimal.Decimal
	TotalAmount           decimal.Decimal
	TaxInclusive          bool
	NetAmount             decimal.Decimal
	TaxAmount             decimal.Decimal
	Currency              string
	ExchangeRate          decimal.Decimal
	ExchangeRateSource    string
	ExchangeRateDate      string
	ExchangeRateSettingID *uuid.UUID
	ExchangeRateOverride  *decimal.Decimal
	BaseCurrency          string
	BaseCurrencyAmount    decimal.Decimal
	ExpenseDate           string
	Note                  *string
	Version               uint64
	CancelledAt           *time.Time
	CancelledBy           *uuid.UUID
	CancellationReason    *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type BilledFeeBillContext struct {
	BillID   uuid.UUID
	Status   FinanceBillStatus
	BillDate string
	Currency string
	FeeCount int
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
	SettlementParties        []OrderFeeSettlementPartyOption
	Currencies               []OrderFeeCurrencyOption
	FeeSettings              []OrderFeeSettingOption
	BillingUnits             []OrderFeeBillingUnitOption
	BaseCurrency             string
	FinanceLocked            bool
	FinanceLockReason        string
	FinanceLockCommissionNos []string
	CustomerID               uuid.UUID
	CustomerName             string
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
	Get(ctx context.Context, organizationID, orderID, id uuid.UUID) (*OrderFee, error)
	BilledBillContext(ctx context.Context, organizationID, orderID, id uuid.UUID) (*BilledFeeBillContext, error)
	GetByIdempotencyKey(ctx context.Context, organizationID, orderID uuid.UUID, idempotencyKey string) (*OrderFee, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderFee, audit *AuditEvent) (*OrderFee, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderFee, billExchangeRate *ResolvedExchangeRate, audit *AuditEvent) (*OrderFee, error)
	Transition(ctx context.Context, organizationID, orderID, id, actorID uuid.UUID, expectedVersion uint64, from, to OrderFeeStatus, reason *string, audit *AuditEvent) (*OrderFee, error)
	Remove(ctx context.Context, organizationID, orderID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) error
}

type OrderFeeUsecase struct {
	repo          OrderFeeRepo
	exchangeRate  *ExchangeRateUsecase
	customSetting *FinanceCustomSettingUsecase
}

func NewOrderFeeUsecase(repo OrderFeeRepo, exchangeRate *ExchangeRateUsecase, customSetting *FinanceCustomSettingUsecase) *OrderFeeUsecase {
	return &OrderFeeUsecase{repo: repo, exchangeRate: exchangeRate, customSetting: customSetting}
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
	normalized.Status = OrderFeeDraft
	normalized.Version = 1
	if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized); err != nil {
		return nil, err
	}
	if err := uc.resolveExchangeRate(ctx, organizationID, orderID, normalized, canOverrideExchangeRate); err != nil {
		return nil, err
	}
	if err := uc.calculateAmounts(ctx, organizationID, normalized); err != nil {
		return nil, err
	}
	existing, err := uc.repo.GetByIdempotencyKey(ctx, organizationID, orderID, normalized.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if sameOrderFeeCreateIntent(existing, normalized) {
			return existing, nil
		}
		return nil, ErrOrderFeeIdempotencyConflict
	}
	created, err := uc.repo.Add(ctx, organizationID, orderID, normalized, orderFeeAudit(organizationID, actorID, orderID, normalized.ID, "order.fee.add", normalized))
	if err == nil {
		return created, nil
	}
	// 并发重试可能在预查后命中唯一索引；再次读取并仅在请求语义一致时复用结果。
	existing, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, orderID, normalized.IdempotencyKey)
	if lookupErr == nil && existing != nil && sameOrderFeeCreateIntent(existing, normalized) {
		return existing, nil
	}
	return nil, err
}

func sameOrderFeeCreateIntent(existing, requested *OrderFee) bool {
	if existing == nil || requested == nil {
		return false
	}
	return existing.Direction == requested.Direction &&
		uuidPointersEqual(existing.FeeSettingID, requested.FeeSettingID) &&
		existing.SettlementPartyID == requested.SettlementPartyID &&
		uuidPointersEqual(existing.BillingUnitID, requested.BillingUnitID) &&
		existing.Quantity.Equal(requested.Quantity) &&
		existing.UnitPrice.Equal(requested.UnitPrice) &&
		existing.Currency == requested.Currency &&
		existing.ExpenseDate == requested.ExpenseDate &&
		stringPointersEqual(existing.Note, requested.Note) &&
		existing.TaxInclusive == requested.TaxInclusive &&
		existing.TotalAmount.Equal(requested.TotalAmount) &&
		existing.ExchangeRate.Equal(requested.ExchangeRate)
}

func uuidPointersEqual(left, right *uuid.UUID) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func stringPointersEqual(left, right *string) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func (uc *OrderFeeUsecase) Update(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, input *OrderFee, canOverrideExchangeRate bool) (*OrderFee, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || input == nil || input.Version == 0 {
		return nil, ErrOrderFeeInvalidArgument
	}
	current, err := uc.repo.Get(ctx, organizationID, orderID, id)
	if err != nil {
		return nil, err
	}
	requestedTaxRate := input.TaxRateOverride
	input.ID = id
	normalized, err := normalizeOrderFee(input)
	if err != nil {
		return nil, err
	}
	switch current.Status {
	case OrderFeeDraft:
		if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized); err != nil {
			return nil, err
		}
	case OrderFeeBilled:
		if !uuidPointersEqual(current.FeeSettingID, normalized.FeeSettingID) || !uuidPointersEqual(current.BillingUnitID, normalized.BillingUnitID) {
			return nil, ErrBilledFeeFieldForbidden
		}
		normalized.FeeCode, normalized.FeeName, normalized.FeeNameEN = current.FeeCode, current.FeeName, current.FeeNameEN
		normalized.BillingUnit, normalized.TaxableServiceName = current.BillingUnit, current.TaxableServiceName
		normalized.TaxRate = current.TaxRate
		if input.FeeNameOverride != nil {
			normalized.FeeName = *input.FeeNameOverride
			normalized.FeeNameOverride = input.FeeNameOverride
		}
		if requestedTaxRate != nil {
			normalized.TaxRate = requestedTaxRate
			normalized.TaxRateOverride = requestedTaxRate
		}
	default:
		return nil, ErrOrderFeeInvalidTransition
	}
	if current.Status == OrderFeeBilled && normalized.ExchangeRateOverride == nil && normalized.Currency == current.Currency {
		normalized.ExchangeRate, normalized.ExchangeRateSource, normalized.ExchangeRateDate, normalized.ExchangeRateSettingID = current.ExchangeRate, current.ExchangeRateSource, current.ExchangeRateDate, current.ExchangeRateSettingID
	} else {
		if err := uc.resolveExchangeRate(ctx, organizationID, orderID, normalized, canOverrideExchangeRate); err != nil {
			return nil, err
		}
	}
	if err := uc.calculateAmounts(ctx, organizationID, normalized); err != nil {
		return nil, err
	}
	var billExchangeRate *ResolvedExchangeRate
	switch current.Status {
	case OrderFeeDraft:
		if requestedTaxRate != nil || input.FeeNameOverride != nil {
			return nil, ErrOrderFeeInvalidArgument
		}
	case OrderFeeBilled:
		if uc.customSetting == nil {
			return nil, ErrBilledFeeEditDisabled
		}
		policy, policyErr := uc.customSetting.GetBilledFeeEditPolicy(ctx, organizationID)
		if policyErr != nil {
			return nil, policyErr
		}
		if validateErr := ValidateBilledFeeUpdate(current, normalized, policy); validateErr != nil {
			return nil, validateErr
		}
		if current.Currency != normalized.Currency {
			billContext, contextErr := uc.repo.BilledBillContext(ctx, organizationID, orderID, id)
			if contextErr != nil {
				return nil, contextErr
			}
			if billContext.Status != FinanceBillDraft {
				return nil, ErrBilledFeeBillLocked
			}
			if billContext.FeeCount != 1 {
				return nil, ErrBilledFeeCurrencyConflict
			}
			billExchangeRate, contextErr = uc.exchangeRate.Resolve(ctx, organizationID, BillRateType, normalized.Direction, normalized.Currency, map[string]string{BillDateStandard: billContext.BillDate})
			if contextErr != nil {
				return nil, contextErr
			}
		}
	default:
		return nil, ErrOrderFeeInvalidTransition
	}
	normalized.Status = current.Status
	normalized.Version = input.Version
	auditFee := *normalized
	auditFee.Version = input.Version + 1
	return uc.repo.Update(ctx, organizationID, orderID, id, normalized, billExchangeRate, orderFeeAudit(organizationID, actorID, orderID, id, "order.fee.update", &auditFee))
}

// ValidateBilledFeeUpdate 校验已建账单费用的字段级修改范围；数据层会在事务锁内再次调用。
func ValidateBilledFeeUpdate(current, requested *OrderFee, policy *BilledFeeEditPolicy) error {
	if current == nil || requested == nil {
		return ErrOrderFeeInvalidArgument
	}
	if policy == nil || !policy.Enabled {
		return ErrBilledFeeEditDisabled
	}
	if current.Direction != requested.Direction || current.SettlementPartyID != requested.SettlementPartyID || current.ExpenseDate != requested.ExpenseDate || !stringPointersEqual(current.Note, requested.Note) || current.TaxInclusive != requested.TaxInclusive {
		return ErrBilledFeeFieldForbidden
	}
	if !uuidPointersEqual(current.FeeSettingID, requested.FeeSettingID) || current.FeeCode != requested.FeeCode || !stringPointersEqual(current.FeeNameEN, requested.FeeNameEN) || !uuidPointersEqual(current.BillingUnitID, requested.BillingUnitID) || current.BillingUnit != requested.BillingUnit || !stringPointersEqual(current.TaxableServiceName, requested.TaxableServiceName) {
		return ErrBilledFeeFieldForbidden
	}
	nameChanged := current.FeeName != requested.FeeName
	if nameChanged && !policy.Allows(BilledFeeFieldFeeName) {
		return ErrBilledFeeFieldForbidden
	}
	currencyChanged := current.Currency != requested.Currency
	if currencyChanged && !policy.Allows(BilledFeeFieldCurrency) {
		return ErrBilledFeeFieldForbidden
	}
	// 修改币种时系统必然会重新解析汇率；仅手工覆盖汇率时才额外要求“汇率”权限。
	if (!currencyChanged && !current.ExchangeRate.Equal(requested.ExchangeRate) || currencyChanged && requested.ExchangeRateOverride != nil) && !policy.Allows(BilledFeeFieldExchangeRate) {
		return ErrBilledFeeFieldForbidden
	}
	if !current.Quantity.Equal(requested.Quantity) && !policy.Allows(BilledFeeFieldQuantity) {
		return ErrBilledFeeFieldForbidden
	}
	if !current.UnitPrice.Equal(requested.UnitPrice) && !policy.Allows(BilledFeeFieldUnitPrice) {
		return ErrBilledFeeFieldForbidden
	}
	if !decimalPointersEqual(current.TaxRate, requested.TaxRate) && !policy.Allows(BilledFeeFieldTaxRate) {
		return ErrBilledFeeFieldForbidden
	}
	return nil
}

func decimalPointersEqual(left, right *decimal.Decimal) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && left.Equal(*right))
}

func (uc *OrderFeeUsecase) calculateAmounts(ctx context.Context, organizationID uuid.UUID, fee *OrderFee) error {
	if fee.TaxRate == nil {
		return ErrOrderFeeInvalidArgument
	}
	rateDivisor := decimal.NewFromInt(1).Add(fee.TaxRate.Div(decimal.NewFromInt(100)))
	if fee.TaxInclusive {
		fee.NetAmount = fee.TotalAmount.Div(rateDivisor).RoundBank(8)
		fee.TaxAmount = fee.TotalAmount.Sub(fee.NetAmount)
	} else {
		fee.NetAmount = fee.TotalAmount
		fee.TaxAmount = fee.NetAmount.Mul(*fee.TaxRate).Div(decimal.NewFromInt(100)).RoundBank(8)
		fee.TotalAmount = fee.NetAmount.Add(fee.TaxAmount)
	}
	baseCurrency, err := uc.exchangeRate.BaseCurrency(ctx, organizationID)
	if err != nil {
		return err
	}
	fee.BaseCurrency = baseCurrency
	fee.BaseCurrencyAmount = fee.TotalAmount.Mul(fee.ExchangeRate).RoundBank(8)
	if !totalAmountPattern.MatchString(fee.TotalAmount.String()) || !totalAmountPattern.MatchString(fee.NetAmount.String()) || !totalAmountPattern.MatchString(fee.TaxAmount.String()) || !totalAmountPattern.MatchString(fee.BaseCurrencyAmount.String()) {
		return ErrOrderFeeInvalidArgument
	}
	return nil
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

func (uc *OrderFeeUsecase) Confirm(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, expectedVersion uint64) (*OrderFee, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrOrderFeeInvalidArgument
	}
	return uc.repo.Transition(ctx, organizationID, orderID, id, actorID, expectedVersion, OrderFeeDraft, OrderFeeConfirmed, nil, &AuditEvent{
		OrganizationID: &organizationID, UserID: &actorID, Action: "order.fee.confirm", Result: "success",
		Details: map[string]string{"fee.id": id.String(), "order.id": orderID.String()},
	})
}

func (uc *OrderFeeUsecase) Reopen(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, expectedVersion uint64, reason string) (*OrderFee, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrOrderFeeInvalidArgument
	}
	return uc.repo.Transition(ctx, organizationID, orderID, id, actorID, expectedVersion, OrderFeeConfirmed, OrderFeeDraft, &reason, &AuditEvent{
		OrganizationID: &organizationID, UserID: &actorID, Action: "order.fee.reopen", Result: "success",
		Details: map[string]string{"fee.id": id.String(), "order.id": orderID.String(), "reason": reason},
	})
}

func (uc *OrderFeeUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, expectedVersion uint64, reason string) error {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return ErrOrderFeeInvalidArgument
	}
	return uc.repo.Remove(ctx, organizationID, orderID, id, actorID, expectedVersion, reason, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.fee.remove",
		Result:         "success",
		Details: map[string]string{
			"fee.id": id.String(), "order.id": orderID.String(), "reason": reason,
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
			"fee.status":               string(fee.Status),
			"fee.version":              fmt.Sprintf("%d", fee.Version),
			"fee.net_amount":           fee.NetAmount.StringFixed(8),
			"fee.tax_amount":           fee.TaxAmount.StringFixed(8),
			"fee.base_currency_amount": fee.BaseCurrencyAmount.StringFixed(8),
		},
	}
}

func normalizeOrderFee(input *OrderFee) (*OrderFee, error) {
	if input == nil || input.SettlementPartyID == uuid.Nil || input.FeeSettingID == nil || *input.FeeSettingID == uuid.Nil || input.BillingUnitID == nil || *input.BillingUnitID == uuid.Nil {
		return nil, ErrOrderFeeInvalidArgument
	}
	idempotencyKey := strings.TrimSpace(input.IdempotencyKey)
	if input.ID == uuid.Nil && (idempotencyKey == "" || utf8.RuneCountInString(idempotencyKey) > 128) {
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
	if input.TaxRateOverride != nil && (input.TaxRateOverride.IsNegative() || input.TaxRateOverride.GreaterThan(decimal.NewFromInt(100)) || !taxRatePattern.MatchString(input.TaxRateOverride.String())) {
		return nil, ErrOrderFeeInvalidArgument
	}
	if input.FeeNameOverride != nil {
		value := strings.TrimSpace(*input.FeeNameOverride)
		if value == "" || utf8.RuneCountInString(value) > 80 {
			return nil, ErrOrderFeeInvalidArgument
		}
		outputName := value
		input.FeeNameOverride = &outputName
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
	output.IdempotencyKey = idempotencyKey
	output.TotalAmount = totalAmount
	output.Currency = currency
	output.ExpenseDate = expenseDate
	output.Note = note
	return &output, nil
}
