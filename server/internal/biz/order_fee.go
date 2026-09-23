package biz

import (
	"context"
	"fmt"
	"log/slog"
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
	ErrOrderFeeQuantityMustBeInteger         = errors.BadRequest("ORDER_FEE_QUANTITY_MUST_BE_INTEGER", "该计费单位的数量必须为正整数")
	ErrOrderFeeExchangeRateOverrideForbidden = errors.Forbidden("ORDER_FEE_EXCHANGE_RATE_OVERRIDE_FORBIDDEN", "无权手工覆盖费用汇率")
	ErrOrderFeeVersionConflict               = errors.Conflict("ORDER_FEE_VERSION_CONFLICT", "订单费用已被其他操作人修改，请刷新后重试")
	ErrOrderFeeInvalidTransition             = errors.Conflict("ORDER_FEE_INVALID_TRANSITION", "当前费用状态不允许执行该操作")
	ErrOrderFeeBillOccupied                  = errors.Conflict("ORDER_FEE_BILL_OCCUPIED", "费用已进入未取消的账单，请先取消对应账单后再删除")
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
	// OrderFeeUnbilled 未建账：费用保存后即进入该状态，可直接维护或建账；
	// 历史 DRAFT/CONFIRMED 区分已按用户决策合并，不设确认环节。
	OrderFeeUnbilled  OrderFeeStatus = "UNBILLED"
	OrderFeeBilled    OrderFeeStatus = "BILLED"
	OrderFeeCancelled OrderFeeStatus = "CANCELLED"
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
	ID                    uuid.UUID
	Code                  string
	Name                  string
	QuantityMustBeInteger bool
}

type OrderFeeCatalogSnapshot struct {
	FeeCode               string
	FeeName               string
	FeeNameEN             *string
	BillingUnit           string
	QuantityMustBeInteger bool
	TaxRate               decimal.Decimal
	TaxableServiceName    string
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

type OrderFeeRepo interface {
	Options(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeOptions, error)
	ResolveCatalog(ctx context.Context, organizationID, orderID, feeSettingID, billingUnitID uuid.UUID, allowDisabledUnit bool) (*OrderFeeCatalogSnapshot, error)
	List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderFee, error)
	Get(ctx context.Context, organizationID, orderID, id uuid.UUID) (*OrderFee, error)
	BilledBillContext(ctx context.Context, organizationID, orderID, id uuid.UUID) (*BilledFeeBillContext, error)
	GetByIdempotencyKey(ctx context.Context, organizationID, orderID uuid.UUID, idempotencyKey string) (*OrderFee, error)
	Add(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderFee, audit *AuditEvent) (*OrderFee, error)
	Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *OrderFee, billExchangeRate *ResolvedRate, audit *AuditEvent) (*OrderFee, error)
	Remove(ctx context.Context, organizationID, orderID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) error
	BulkUpdate(ctx context.Context, organizationID, orderID uuid.UUID, input *OrderFeeBulkUpdateInput, audits map[uuid.UUID]*AuditEvent) error
	BulkRemove(ctx context.Context, organizationID, orderID uuid.UUID, targets []OrderFeeBulkTarget, audits map[uuid.UUID]*AuditEvent) error
}

// OrderFeeBulkTarget 批量维护操作的单一费用目标：费用 ID 与乐观锁版本。
type OrderFeeBulkTarget struct {
	FeeID           uuid.UUID
	ExpectedVersion uint64
}

// OrderFeeBulkRatePlan 批量改费用时间时系统来源汇率的逐行重解析结果：
// 总额、数量与单价保持不变，折本币金额 = 总额 × 新汇率。
type OrderFeeBulkRatePlan struct {
	Rate       decimal.Decimal
	Source     string
	RateDate   string
	SettingID  *uuid.UUID
	BaseAmount decimal.Decimal
}

// OrderFeeBulkUpdateInput 批量定向修改计划：SettlementPartyID 与 ExpenseDate
// 二选一；RatePlans 仅包含系统来源汇率行，手工来源行保持原汇率、来源与折本币。
type OrderFeeBulkUpdateInput struct {
	Targets           []OrderFeeBulkTarget
	SettlementPartyID *uuid.UUID
	ExpenseDate       *string
	RatePlans         map[uuid.UUID]*OrderFeeBulkRatePlan
}

// BulkOrderFeeError 为批量维护构造携带具体费用定位的业务错误：保持与单条
// 操作相同的 reason 与 HTTP 语义（errors.Is 仍命中对应哨兵错误），message
// 额外指明费用 ID，便于界面定位失败行。
func BulkOrderFeeError(base *errors.Error, feeID uuid.UUID, message string) *errors.Error {
	return errors.Newf(int(base.Code), base.Reason, "费用 %s：%s", feeID, message)
}

type OrderFeeUsecase struct {
	repo          OrderFeeRepo
	exchangeRate  *ExchangeRateUsecase
	customSetting *FinanceCustomSettingUsecase
	creditControl *PartnerCreditUsecase
	autoLock      *AutoOrderLockUsecase
	logger        *slog.Logger
}

func NewOrderFeeUsecase(repo OrderFeeRepo, exchangeRate *ExchangeRateUsecase, customSetting *FinanceCustomSettingUsecase, creditControl *PartnerCreditUsecase, autoLock *AutoOrderLockUsecase, logger *slog.Logger) *OrderFeeUsecase {
	return &OrderFeeUsecase{repo: repo, exchangeRate: exchangeRate, customSetting: customSetting, creditControl: creditControl, autoLock: autoLock, logger: logger}
}

// ensureReceivablePartySelectionAllowed 在直接干预模式下校验应收费用结算单位未超额；
// 应付与成本方向不占用客户信用额度，不校验。校验为尽力而为的时点查询。
func (uc *OrderFeeUsecase) ensureReceivablePartySelectionAllowed(ctx context.Context, organizationID uuid.UUID, fee *OrderFee) error {
	if fee == nil || fee.Direction != OrderFeeReceivable || fee.SettlementPartyID == uuid.Nil {
		return nil
	}
	return uc.creditControl.EnsurePartnerSelectionAllowed(ctx, organizationID, fee.SettlementPartyID)
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
	normalized.Status = OrderFeeUnbilled
	normalized.Version = 1
	if err := uc.ensureReceivablePartySelectionAllowed(ctx, organizationID, normalized); err != nil {
		return nil, err
	}
	if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized, true, false); err != nil {
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
	// 仅未建账费用可更换结算单位；已建账费用的结算单位不可变，无需重复校验。
	if current.Status == OrderFeeUnbilled {
		if err := uc.ensureReceivablePartySelectionAllowed(ctx, organizationID, normalized); err != nil {
			return nil, err
		}
	}
	switch current.Status {
	case OrderFeeUnbilled:
		if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized, false, uuidPointersEqual(current.BillingUnitID, normalized.BillingUnitID)); err != nil {
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
	if normalized.ExchangeRateOverride == nil && normalized.Currency == current.Currency && normalized.Direction == current.Direction && normalized.ExpenseDate == current.ExpenseDate {
		normalized.ExchangeRate, normalized.ExchangeRateSource, normalized.ExchangeRateDate, normalized.ExchangeRateSettingID = current.ExchangeRate, current.ExchangeRateSource, current.ExchangeRateDate, current.ExchangeRateSettingID
	} else if current.Status == OrderFeeBilled && normalized.ExchangeRateOverride == nil && normalized.Currency == current.Currency {
		normalized.ExchangeRate, normalized.ExchangeRateSource, normalized.ExchangeRateDate, normalized.ExchangeRateSettingID = current.ExchangeRate, current.ExchangeRateSource, current.ExchangeRateDate, current.ExchangeRateSettingID
	} else {
		if err := uc.resolveExchangeRate(ctx, organizationID, orderID, normalized, canOverrideExchangeRate); err != nil {
			return nil, err
		}
	}
	if err := uc.calculateAmounts(ctx, organizationID, normalized); err != nil {
		return nil, err
	}
	var billExchangeRate *ResolvedRate
	switch current.Status {
	case OrderFeeUnbilled:
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
			billRate, contextErr := uc.exchangeRate.ResolveRate(ctx, organizationID, normalized.Direction, normalized.Currency, billContext.BillDate)
			if contextErr != nil {
				return nil, contextErr
			}
			billExchangeRate = &billRate
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

func (uc *OrderFeeUsecase) resolveCatalog(ctx context.Context, organizationID, orderID uuid.UUID, fee *OrderFee, validateQuantity, allowDisabledUnit bool) error {
	snapshot, err := uc.repo.ResolveCatalog(ctx, organizationID, orderID, *fee.FeeSettingID, *fee.BillingUnitID, allowDisabledUnit)
	if err != nil {
		return err
	}
	if validateQuantity {
		if err := ValidateFeeQuantityForUnit(fee.Quantity, snapshot.QuantityMustBeInteger); err != nil {
			return err
		}
	}
	fee.FeeCode = snapshot.FeeCode
	fee.FeeName = snapshot.FeeName
	fee.FeeNameEN = snapshot.FeeNameEN
	fee.BillingUnit = snapshot.BillingUnit
	fee.TaxRate = &snapshot.TaxRate
	fee.TaxableServiceName = &snapshot.TaxableServiceName
	return nil
}

func ValidateFeeQuantityForUnit(quantity decimal.Decimal, mustBeInteger bool) error {
	if mustBeInteger && (!quantity.IsPositive() || !quantity.Equal(quantity.Truncate(0))) {
		return ErrOrderFeeQuantityMustBeInteger
	}
	return nil
}

// ResolveExchangeRate 按费用发生日与收支方向解析折本币周汇率（携带来源与命中行），
// 供前端录入费用时预览；应收命中 ar_rate，应付命中 ap_rate。
func (uc *OrderFeeUsecase) ResolveExchangeRate(ctx context.Context, organizationID, orderID uuid.UUID, direction OrderFeeDirection, currency, expenseDate string) (ResolvedRate, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil || (direction != OrderFeeReceivable && direction != OrderFeePayable) {
		return ResolvedRate{}, ErrOrderFeeInvalidArgument
	}
	if _, err := uc.repo.Options(ctx, organizationID, orderID); err != nil {
		return ResolvedRate{}, err
	}
	return uc.exchangeRate.ResolveRate(ctx, organizationID, direction, currency, expenseDateDay(expenseDate))
}

// isValidExpenseDate 校验发生日期：纯日期（YYYY-MM-DD）或精确到分钟
// （YYYY-MM-DD HH:mm）。汇率按自然周解析，时刻不参与校验。
func isValidExpenseDate(value string) bool {
	if parsed, err := time.Parse("2006-01-02", value); err == nil && parsed.Format("2006-01-02") == value {
		return true
	}
	parsed, err := time.Parse("2006-01-02 15:04", value)
	return err == nil && parsed.Format("2006-01-02 15:04") == value
}

// expenseDateDay 截取发生日期的日期部分，供按周匹配的汇率解析使用。
func expenseDateDay(expenseDate string) string {
	if day, _, found := strings.Cut(expenseDate, " "); found {
		return day
	}
	return expenseDate
}

func (uc *OrderFeeUsecase) resolveExchangeRate(ctx context.Context, organizationID, orderID uuid.UUID, fee *OrderFee, canOverrideExchangeRate bool) error {
	if fee.ExchangeRateOverride != nil {
		if !canOverrideExchangeRate {
			return ErrOrderFeeExchangeRateOverrideForbidden
		}
		fee.ExchangeRate = *fee.ExchangeRateOverride
		fee.ExchangeRateSource = ExchangeRateSourceManual
		// 汇率日期列恒存日期部分（10 字符）：发生日期支持分钟精度后不能整串落库。
		fee.ExchangeRateDate = expenseDateDay(fee.ExpenseDate)
		fee.ExchangeRateSettingID = nil
		return nil
	}
	resolved, err := uc.exchangeRate.ResolveRate(ctx, organizationID, fee.Direction, fee.Currency, expenseDateDay(fee.ExpenseDate))
	if err != nil {
		return err
	}
	fee.ExchangeRate = resolved.Rate
	fee.ExchangeRateSource = resolved.Source
	fee.ExchangeRateDate = expenseDateDay(fee.ExpenseDate)
	fee.ExchangeRateSettingID = resolved.SettingID
	return nil
}

// Remove 按账单占用关系物理删除费用：费用未被未取消账单（含草稿账单）包含时
// 可删除；被占用时仓储在事务内以事实复核并返回 ErrOrderFeeBillOccupied。未建账
// 状态不阻止删除；历史账单与金额快照不受影响。reason 选填，仅写入审计明细。
func (uc *OrderFeeUsecase) Remove(ctx context.Context, organizationID, actorID, orderID, id uuid.UUID, expectedVersion uint64, reason string) error {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || utf8.RuneCountInString(reason) > 500 {
		return ErrOrderFeeInvalidArgument
	}
	// 删除未建账费用与作废同属自动锁定重试触发；先读取当前状态用于判断触发类型。
	current, err := uc.repo.Get(ctx, organizationID, orderID, id)
	if err != nil {
		return err
	}
	details := map[string]string{
		"fee.id": id.String(), "order.id": orderID.String(),
	}
	if reason != "" {
		details["reason"] = reason
	}
	if err := uc.repo.Remove(ctx, organizationID, orderID, id, actorID, expectedVersion, reason, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.fee.delete",
		Result:         "success",
		Details:        details,
	}); err != nil {
		return err
	}
	if current.Status == OrderFeeUnbilled {
		uc.triggerAutoLock(ctx, organizationID, actorID, id, orderID, AutoLockTriggerFeeCancel)
	}
	return nil
}

// validateOrderFeeBulkTargets 校验批量目标集合：非空、费用 ID 非空且不重复、
// 乐观锁版本非零。
func validateOrderFeeBulkTargets(targets []OrderFeeBulkTarget) error {
	if len(targets) == 0 {
		return ErrOrderFeeInvalidArgument
	}
	seen := make(map[uuid.UUID]struct{}, len(targets))
	for _, target := range targets {
		if target.FeeID == uuid.Nil || target.ExpectedVersion == 0 {
			return ErrOrderFeeInvalidArgument
		}
		if _, exists := seen[target.FeeID]; exists {
			return ErrOrderFeeInvalidArgument
		}
		seen[target.FeeID] = struct{}{}
	}
	return nil
}

// BulkUpdate 批量定向修改未建账费用：结算单位或费用时间二选一，整批单一事务。
// 用例层先读取目标费用并完成逐行汇率重解析（系统来源按新日期解析，任一缺失
// 整批拒绝）与应收信用校验，再交由仓储在订单锁与费用行锁内复核归属、版本与
// 状态后统一写入；任一行失败整批回滚，错误携带具体费用定位。
func (uc *OrderFeeUsecase) BulkUpdate(ctx context.Context, organizationID, actorID, orderID uuid.UUID, targets []OrderFeeBulkTarget, settlementPartyID *uuid.UUID, expenseDate *string) error {
	if err := validateOrderFeeBulkTargets(targets); err != nil {
		return err
	}
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return ErrOrderFeeInvalidArgument
	}
	// 目标值必须恰好提供一个。
	if (settlementPartyID == nil) == (expenseDate == nil) {
		return ErrOrderFeeInvalidArgument
	}
	if settlementPartyID != nil && *settlementPartyID == uuid.Nil {
		return ErrOrderFeeInvalidArgument
	}
	newExpenseDate := ""
	if expenseDate != nil {
		newExpenseDate = strings.TrimSpace(*expenseDate)
		if !isValidExpenseDate(newExpenseDate) {
			return ErrOrderFeeInvalidArgument
		}
	}
	fees, err := uc.repo.List(ctx, organizationID, orderID)
	if err != nil {
		return err
	}
	feeByID := make(map[uuid.UUID]*OrderFee, len(fees))
	for _, fee := range fees {
		feeByID[fee.ID] = fee
	}
	targetFees := make([]*OrderFee, 0, len(targets))
	for _, target := range targets {
		fee, ok := feeByID[target.FeeID]
		if !ok {
			return BulkOrderFeeError(ErrOrderFeeNotFound, target.FeeID, "不存在或不属于该订单")
		}
		targetFees = append(targetFees, fee)
	}
	input := &OrderFeeBulkUpdateInput{Targets: targets, SettlementPartyID: settlementPartyID}
	if expenseDate != nil {
		input.ExpenseDate = &newExpenseDate
	}
	if settlementPartyID != nil {
		// 批量目标共用同一结算单位：存在应收目标行时信用控制整批只需校验一次；
		// 全应付批次与单条修改同口径不占用客户信用，直接放行。
		anyReceivable := false
		for _, fee := range targetFees {
			if fee.Direction == OrderFeeReceivable {
				anyReceivable = true
				break
			}
		}
		if anyReceivable {
			probe := &OrderFee{Direction: OrderFeeReceivable, SettlementPartyID: *settlementPartyID}
			if err := uc.ensureReceivablePartySelectionAllowed(ctx, organizationID, probe); err != nil {
				return err
			}
		}
	} else {
		plans := make(map[uuid.UUID]*OrderFeeBulkRatePlan, len(targetFees))
		for _, fee := range targetFees {
			// 手工来源汇率保持原值、来源与折本币不动，不参与重解析。
			if fee.ExchangeRateSource == ExchangeRateSourceManual {
				continue
			}
			rateDate := expenseDateDay(newExpenseDate)
			resolved, resolveErr := uc.exchangeRate.ResolveRate(ctx, organizationID, fee.Direction, fee.Currency, rateDate)
			if resolveErr != nil {
				if errors.Is(resolveErr, ErrExchangeRateMissing) {
					return BulkOrderFeeError(ErrExchangeRateMissing, fee.ID, fmt.Sprintf("按新费用时间 %s 未命中 %s 折本币汇率，请先维护汇率", rateDate, fee.Currency))
				}
				return resolveErr
			}
			baseAmount := fee.TotalAmount.Mul(resolved.Rate).RoundBank(8)
			if !totalAmountPattern.MatchString(baseAmount.String()) {
				return BulkOrderFeeError(ErrOrderFeeInvalidArgument, fee.ID, "按新汇率重算折本币金额超出允许精度")
			}
			plans[fee.ID] = &OrderFeeBulkRatePlan{
				Rate: resolved.Rate, Source: resolved.Source, RateDate: rateDate,
				SettingID: resolved.SettingID, BaseAmount: baseAmount,
			}
		}
		input.RatePlans = plans
	}
	audits := make(map[uuid.UUID]*AuditEvent, len(targetFees))
	for _, fee := range targetFees {
		audits[fee.ID] = orderFeeBulkUpdateAudit(organizationID, actorID, orderID, fee, input)
	}
	return uc.repo.BulkUpdate(ctx, organizationID, orderID, input, audits)
}

// BulkRemove 批量删除未建账费用：整批单一事务，被未取消账单占用、版本冲突
// 或越订单任一不满足时整批回滚。删除成功后按单条删除同款语义对每个未建账
// 目标触发结清自动锁定重评（幂等）。
func (uc *OrderFeeUsecase) BulkRemove(ctx context.Context, organizationID, actorID, orderID uuid.UUID, targets []OrderFeeBulkTarget, reason string) error {
	if err := validateOrderFeeBulkTargets(targets); err != nil {
		return err
	}
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil {
		return ErrOrderFeeInvalidArgument
	}
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 500 {
		return ErrOrderFeeInvalidArgument
	}
	fees, err := uc.repo.List(ctx, organizationID, orderID)
	if err != nil {
		return err
	}
	feeByID := make(map[uuid.UUID]*OrderFee, len(fees))
	for _, fee := range fees {
		feeByID[fee.ID] = fee
	}
	targetFees := make([]*OrderFee, 0, len(targets))
	for _, target := range targets {
		fee, ok := feeByID[target.FeeID]
		if !ok {
			return BulkOrderFeeError(ErrOrderFeeNotFound, target.FeeID, "不存在或不属于该订单")
		}
		targetFees = append(targetFees, fee)
	}
	audits := make(map[uuid.UUID]*AuditEvent, len(targetFees))
	for _, fee := range targetFees {
		details := map[string]string{
			"fee.id":   fee.ID.String(),
			"order.id": orderID.String(),
			"fee.bulk": "true",
		}
		if reason != "" {
			details["reason"] = reason
		}
		audits[fee.ID] = &AuditEvent{
			OrganizationID: &organizationID,
			UserID:         &actorID,
			Action:         "order.fee.delete",
			Result:         "success",
			Details:        details,
		}
	}
	if err := uc.repo.BulkRemove(ctx, organizationID, orderID, targets, audits); err != nil {
		return err
	}
	for _, fee := range targetFees {
		if fee.Status == OrderFeeUnbilled {
			uc.triggerAutoLock(ctx, organizationID, actorID, fee.ID, orderID, AutoLockTriggerFeeCancel)
		}
	}
	return nil
}

// orderFeeBulkUpdateAudit 为批量定向修改构建逐行审计事件：记录修改维度的
// from→to、重解析汇率的 from→to 与批量来源标识；行内权威事实由仓储锁内补充。
func orderFeeBulkUpdateAudit(organizationID, actorID, orderID uuid.UUID, fee *OrderFee, input *OrderFeeBulkUpdateInput) *AuditEvent {
	details := map[string]string{
		"fee.id":        fee.ID.String(),
		"order.id":      orderID.String(),
		"fee.code":      fee.FeeCode,
		"fee.direction": string(fee.Direction),
		"fee.amount":    fee.TotalAmount.StringFixed(8),
		"fee.currency":  fee.Currency,
		"fee.bulk":      "true",
	}
	if input.SettlementPartyID != nil {
		details["fee.settlement_party.from"] = fee.SettlementPartyID.String()
		details["fee.settlement_party_name.from"] = fee.SettlementPartyName
		details["fee.settlement_party.to"] = input.SettlementPartyID.String()
	} else {
		details["fee.expense_date.from"] = fee.ExpenseDate
		details["fee.expense_date.to"] = *input.ExpenseDate
		if plan := input.RatePlans[fee.ID]; plan != nil {
			details["fee.exchange_rate.from"] = fee.ExchangeRate.StringFixed(8)
			details["fee.exchange_rate.to"] = plan.Rate.StringFixed(8)
			details["fee.base_currency_amount.from"] = fee.BaseCurrencyAmount.StringFixed(8)
			details["fee.base_currency_amount.to"] = plan.BaseAmount.StringFixed(8)
		}
	}
	return &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.fee.update",
		Result:         "success",
		Details:        details,
	}
}

// triggerAutoLock 在未建账费用作废成功提交后触发结清自动锁定检查。
// 触发失败只记录警告日志；纯成本等无有效结清事实的订单由仓储预检静默跳过。
func (uc *OrderFeeUsecase) triggerAutoLock(ctx context.Context, organizationID, actorID, feeID, orderID uuid.UUID, triggerType AutoLockTriggerSource) {
	if uc.autoLock == nil {
		return
	}
	trigger := AutoOrderLockTrigger{
		Type:           triggerType,
		ResourceID:     feeID,
		OrganizationID: organizationID,
		TriggeredBy:    actorID,
		OrderID:        orderID,
	}
	if err := uc.autoLock.RunSettlementLockCheck(ctx, trigger); err != nil {
		if uc.logger != nil {
			uc.logger.WarnContext(ctx, "费用状态流转触发的自动锁定检查失败",
				slog.String("fee_id", feeID.String()),
				slog.String("order_id", orderID.String()),
				slog.String("organization_id", organizationID.String()),
				slog.String("error", err.Error()))
		}
	}
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
	// 发生日期支持到分钟：接受纯日期或“日期 HH:mm”两种精确格式，秒级精度不收。
	expenseDate := strings.TrimSpace(input.ExpenseDate)
	if !isValidExpenseDate(expenseDate) {
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
