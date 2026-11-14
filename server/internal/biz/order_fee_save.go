package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

// 整表保存（SaveOrderFees）：应收或应付单侧一次提交新增与修改，单事务、逐行校验后才写，
// 任一行失败整体回滚；不含删除，不连带另一侧。

const (
	// MaxOrderFeeSaveRows 单次整表保存的行数上限，防止单事务锁定与重算无界放大。
	MaxOrderFeeSaveRows = 200
	maxOrderFeeSaveKey  = 128
)

var ErrOrderFeeSavePartiallyApplied = errors.Conflict("ORDER_FEE_SAVE_PARTIALLY_APPLIED", "本次保存此前已部分生效，请刷新费用后核对再保存")

// OrderFeeSaveRowInput 是用例入口的一行：FeeID 为空表示新增，否则为修改并携带 ExpectedVersion。
type OrderFeeSaveRowInput struct {
	ClientRowID     string
	FeeID           *uuid.UUID
	ExpectedVersion uint64
	Fee             *OrderFee
}

// OrderFeeSavePermissions 是调用方已解析的细粒度授权；新增与修改分别授权。
type OrderFeeSavePermissions struct {
	CanCreate               bool
	CanUpdate               bool
	CanOverrideExchangeRate bool
	CanEditTaxRate          bool
}

// OrderFeeSaveRow 是已完成目录、税率、汇率与金额准备的一行，供仓储在事务内复核并写入。
type OrderFeeSaveRow struct {
	ClientRowID     string
	FeeID           *uuid.UUID // nil 表示新增
	ExpectedVersion uint64
	Fee             *OrderFee
	// ExchangeRateOverridden 表示该行使用手工汇率，币种集合条件由仓储在事务内按提交后整单集合判定。
	ExchangeRateOverridden bool
}

type OrderFeeSaveInput struct {
	Direction OrderFeeDirection
	Rows      []*OrderFeeSaveRow
}

type OrderFeeSaveResult struct {
	// Fees 为保存后本侧全部当前有效费用。
	Fees              []*OrderFee
	FeeContentVersion uint64
	// Replayed 表示请求此前已完整应用，本次仅返回当前状态。
	Replayed bool
}

// OrderFeeSaveRowFailure 是一行的失败事实；Err 保留原业务错误以便 errors.Is 判定。
type OrderFeeSaveRowFailure struct {
	ClientRowID string
	Field       string
	Err         *errors.Error
}

// NewOrderFeeSaveRowFailure 在 err 是业务错误时构造行级失败；其他错误（驱动、系统故障）返回 false，
// 调用方应原样外传，不得伪装成行级失败。
func NewOrderFeeSaveRowFailure(clientRowID string, err error) (OrderFeeSaveRowFailure, bool) {
	var kratosErr *errors.Error
	if !stderrors.As(err, &kratosErr) || kratosErr == nil {
		return OrderFeeSaveRowFailure{}, false
	}
	return OrderFeeSaveRowFailure{ClientRowID: clientRowID, Field: orderFeeSaveErrorField(err), Err: kratosErr}, true
}

// orderFeeSaveErrorField 把常见字段级业务错误映射到前端字段名（与请求 JSON 同名）；无法定位到字段时为空。
func orderFeeSaveErrorField(err error) string {
	switch {
	case errors.Is(err, ErrOrderFeePartyInvalid):
		return "settlementPartyId"
	case errors.Is(err, ErrOrderFeeCurrencyInvalid), errors.Is(err, ErrExchangeRateMissingForDate):
		return "currency"
	case errors.Is(err, ErrOrderFeeSettingInvalid):
		return "feeSettingId"
	case errors.Is(err, ErrOrderFeeBillingUnitInvalid):
		return "billingUnitId"
	case errors.Is(err, ErrOrderFeeQuantityMustBeInteger):
		return "quantity"
	case errors.Is(err, ErrOrderFeeTaxRateForbidden):
		return "taxRate"
	case errors.Is(err, ErrOrderFeeExchangeRateOverrideForbidden), errors.Is(err, ErrOrderFeeExchangeRateOverrideMismatch), errors.Is(err, ErrExchangeRateSameCurrencyInvalid):
		return "exchangeRateOverride"
	case errors.Is(err, ErrOrderFeeVersionConflict):
		return "version"
	}
	return ""
}

// NewOrderFeeSaveFailure 汇总行级失败为一个业务错误：HTTP 语义与 reason 取第一条，metadata 带
// client_row_id、field、reason；多行同时失败时 row_errors 以 JSON 数组列出全部行错误。
func NewOrderFeeSaveFailure(failures []OrderFeeSaveRowFailure) error {
	if len(failures) == 0 {
		return nil
	}
	first := failures[0]
	metadata := make(map[string]string, len(first.Err.Metadata)+4)
	for key, value := range first.Err.Metadata {
		metadata[key] = value
	}
	metadata["client_row_id"] = first.ClientRowID
	metadata["field"] = first.Field
	metadata["reason"] = first.Err.Reason
	if len(failures) > 1 {
		type rowError struct {
			ClientRowID string `json:"client_row_id"`
			Field       string `json:"field"`
			Reason      string `json:"reason"`
			Message     string `json:"message"`
		}
		rows := make([]rowError, 0, len(failures))
		for _, failure := range failures {
			rows = append(rows, rowError{ClientRowID: failure.ClientRowID, Field: failure.Field, Reason: failure.Err.Reason, Message: failure.Err.Message})
		}
		encoded, err := json.Marshal(rows)
		if err == nil {
			metadata["row_errors"] = string(encoded)
		}
	}
	return errors.New(int(first.Err.Code), first.Err.Reason, first.Err.Message).WithMetadata(metadata)
}

// OrderFeeSaveRowKey 派生新增行的幂等键：sha256(request_key:client_row_id) 的十六进制（64 字符，
// 满足费用幂等键 128 字符上限），同一次保存意图下重试得到相同键。
func OrderFeeSaveRowKey(requestKey, clientRowID string) string {
	sum := sha256.Sum256([]byte(requestKey + ":" + clientRowID))
	return hex.EncodeToString(sum[:])
}

// SameOrderFeeIntent 判断已存费用与本次准备好的费用是否表达同一业务意图（回放判定用）。
func SameOrderFeeIntent(existing, requested *OrderFee) bool {
	return sameOrderFeeCreateIntent(existing, requested)
}

// OrderFeeCurrencySetsMatchAfterSave 按「提交后的整单费用集合」判断应收币种集合与应付币种集合是否
// 一致且均非空：existing 为事务内重读的当前有效费用，submitted 中同 ID 的行取代既有行，新增行并入。
func OrderFeeCurrencySetsMatchAfterSave(existing, submitted []*OrderFee) bool {
	replaced := make(map[uuid.UUID]struct{}, len(submitted))
	for _, fee := range submitted {
		replaced[fee.ID] = struct{}{}
	}
	receivable := make(map[string]struct{})
	payable := make(map[string]struct{})
	add := func(fee *OrderFee) {
		if fee.Direction == OrderFeeReceivable {
			receivable[fee.Currency] = struct{}{}
		} else {
			payable[fee.Currency] = struct{}{}
		}
	}
	for _, fee := range existing {
		if _, skip := replaced[fee.ID]; !skip {
			add(fee)
		}
	}
	for _, fee := range submitted {
		add(fee)
	}
	if len(receivable) == 0 || len(payable) == 0 || len(receivable) != len(payable) {
		return false
	}
	for currency := range receivable {
		if _, ok := payable[currency]; !ok {
			return false
		}
	}
	return true
}

// OrderBusinessType 返回订单业务类型，供调用方按品类解析新增/修改的细粒度授权。
func (uc *OrderFeeUsecase) OrderBusinessType(ctx context.Context, organizationID, orderID uuid.UUID) (OrderBusinessType, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return "", ErrOrderFeeInvalidArgument
	}
	order, err := uc.repo.OrderRateDate(ctx, organizationID, orderID)
	if err != nil {
		return "", err
	}
	return order.BusinessType, nil
}

// Save 整表保存单侧费用。步骤：边界校验 → 各行独立准备（目录/税率/汇率/金额，收集全部行错误）→
// 仓储单事务锁定、完整复核后写入。只含新增与修改；调用方只应提交有改动的行。
func (uc *OrderFeeUsecase) Save(ctx context.Context, organizationID, actorID, orderID uuid.UUID, direction OrderFeeDirection, requestKey string, rows []*OrderFeeSaveRowInput, perms OrderFeeSavePermissions) (*OrderFeeSaveResult, error) {
	requestKey = strings.TrimSpace(requestKey)
	if organizationID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil ||
		(direction != OrderFeeReceivable && direction != OrderFeePayable) ||
		requestKey == "" || utf8.RuneCountInString(requestKey) > maxOrderFeeSaveKey ||
		len(rows) == 0 || len(rows) > MaxOrderFeeSaveRows {
		return nil, ErrOrderFeeInvalidArgument
	}
	seenRows := make(map[string]struct{}, len(rows))
	seenFees := make(map[uuid.UUID]struct{}, len(rows))
	for _, row := range rows {
		if row == nil || row.Fee == nil {
			return nil, ErrOrderFeeInvalidArgument
		}
		row.ClientRowID = strings.TrimSpace(row.ClientRowID)
		if row.ClientRowID == "" || utf8.RuneCountInString(row.ClientRowID) > maxOrderFeeSaveKey {
			return nil, ErrOrderFeeInvalidArgument
		}
		if _, dup := seenRows[row.ClientRowID]; dup {
			return nil, ErrOrderFeeInvalidArgument
		}
		seenRows[row.ClientRowID] = struct{}{}
		if row.FeeID != nil {
			if *row.FeeID == uuid.Nil || row.ExpectedVersion == 0 {
				return nil, ErrOrderFeeInvalidArgument
			}
			if _, dup := seenFees[*row.FeeID]; dup {
				return nil, ErrOrderFeeInvalidArgument
			}
			seenFees[*row.FeeID] = struct{}{}
		}
	}

	var failures []OrderFeeSaveRowFailure
	prepared := make([]*OrderFeeSaveRow, 0, len(rows))
	audits := make(map[string]*AuditEvent, len(rows))
	for _, row := range rows {
		item, audit, err := uc.prepareSaveRow(ctx, organizationID, actorID, orderID, direction, requestKey, row, perms)
		if err != nil {
			failure, isRowFailure := NewOrderFeeSaveRowFailure(row.ClientRowID, err)
			if !isRowFailure {
				return nil, err
			}
			failures = append(failures, failure)
			continue
		}
		prepared = append(prepared, item)
		audits[row.ClientRowID] = audit
	}
	if len(failures) > 0 {
		return nil, NewOrderFeeSaveFailure(failures)
	}
	return uc.repo.Save(ctx, organizationID, orderID, &OrderFeeSaveInput{Direction: direction, Rows: prepared}, audits)
}

// prepareSaveRow 复用 Add/Update 的目录、税率、汇率与金额口径；手工汇率的币种集合条件推迟到
// 仓储事务内按提交后的整单集合判定。
func (uc *OrderFeeUsecase) prepareSaveRow(ctx context.Context, organizationID, actorID, orderID uuid.UUID, direction OrderFeeDirection, requestKey string, row *OrderFeeSaveRowInput, perms OrderFeeSavePermissions) (*OrderFeeSaveRow, *AuditEvent, error) {
	input := row.Fee
	if input.Direction != direction {
		return nil, nil, ErrOrderFeeInvalidArgument
	}
	if row.FeeID == nil {
		return uc.prepareSaveAddRow(ctx, organizationID, actorID, orderID, requestKey, row, perms)
	}
	return uc.prepareSaveUpdateRow(ctx, organizationID, actorID, orderID, row, perms)
}

func (uc *OrderFeeUsecase) prepareSaveAddRow(ctx context.Context, organizationID, actorID, orderID uuid.UUID, requestKey string, row *OrderFeeSaveRowInput, perms OrderFeeSavePermissions) (*OrderFeeSaveRow, *AuditEvent, error) {
	if !perms.CanCreate {
		return nil, nil, ErrPermissionDenied
	}
	input := *row.Fee
	input.IdempotencyKey = OrderFeeSaveRowKey(requestKey, row.ClientRowID)
	normalized, err := normalizeOrderFee(&input)
	if err != nil {
		return nil, nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.Version = 1
	if err := uc.ensureReceivablePartySelectionAllowed(ctx, organizationID, normalized); err != nil {
		return nil, nil, err
	}
	if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized, true, false); err != nil {
		return nil, nil, err
	}
	taxRateChanged := normalized.TaxRateOverride != nil && !decimalPointersEqual(normalized.TaxRateOverride, normalized.TaxRate)
	if taxRateChanged {
		if !perms.CanEditTaxRate {
			return nil, nil, ErrOrderFeeTaxRateForbidden
		}
		normalized.TaxRate = normalized.TaxRateOverride
	}
	overridden := normalized.ExchangeRateOverride != nil
	if err := uc.resolveExchangeRateWithSets(ctx, organizationID, orderID, normalized, perms.CanOverrideExchangeRate, nil); err != nil {
		return nil, nil, err
	}
	if err := uc.calculateAmounts(ctx, organizationID, normalized); err != nil {
		return nil, nil, err
	}
	audit := orderFeeAudit(organizationID, actorID, orderID, normalized.ID, "order.fee.add", normalized)
	audit.Details["fee.save"] = "true"
	if taxRateChanged {
		audit.Details["fee.tax_rate_changed"] = "true"
	}
	return &OrderFeeSaveRow{ClientRowID: row.ClientRowID, Fee: normalized, ExchangeRateOverridden: overridden}, audit, nil
}

func (uc *OrderFeeUsecase) prepareSaveUpdateRow(ctx context.Context, organizationID, actorID, orderID uuid.UUID, row *OrderFeeSaveRowInput, perms OrderFeeSavePermissions) (*OrderFeeSaveRow, *AuditEvent, error) {
	if !perms.CanUpdate {
		return nil, nil, ErrPermissionDenied
	}
	id := *row.FeeID
	current, err := uc.repo.Get(ctx, organizationID, orderID, id)
	if err != nil {
		return nil, nil, err
	}
	// 已处于期望版本之后的目标内容属于响应丢失后的重试，交由仓储事务内的回放判定；这里只拦截
	// 明显不可保存的状态，版本差异统一由事务内复核。
	if current.Direction != row.Fee.Direction {
		return nil, nil, ErrOrderFeeInvalidArgument
	}
	if current.SupersededByRequestID != nil || current.EconomicEffect == OrderFeeEconomicDecrease {
		return nil, nil, ErrFeeCorrectionProtected
	}
	input := *row.Fee
	input.ID = id
	input.IdempotencyKey = ""
	normalized, err := normalizeOrderFee(&input)
	if err != nil {
		return nil, nil, err
	}
	if current.Version == row.ExpectedVersion+1 {
		// 疑似回放：以已存快照作为准备基线，目录与汇率口径保持与已落库内容可比较。
		current.Version = row.ExpectedVersion
	}
	if current.HasActiveBill {
		return nil, nil, ErrOrderFeeBillOccupied
	}
	if current.Version != row.ExpectedVersion {
		return nil, nil, ErrOrderFeeVersionConflict
	}
	if err := uc.ensureReceivablePartySelectionAllowed(ctx, organizationID, normalized); err != nil {
		return nil, nil, err
	}
	if err := uc.resolveCatalog(ctx, organizationID, orderID, normalized, false, uuidPointersEqual(current.BillingUnitID, normalized.BillingUnitID)); err != nil {
		return nil, nil, err
	}
	if normalized.TaxRateOverride != nil {
		normalized.TaxRate = normalized.TaxRateOverride
	} else if uuidPointersEqual(current.FeeSettingID, normalized.FeeSettingID) && current.TaxRate != nil {
		normalized.TaxRate = current.TaxRate
	}
	taxRateChanged := !decimalPointersEqual(current.TaxRate, normalized.TaxRate)
	if normalized.TaxRateOverride != nil && taxRateChanged && !perms.CanEditTaxRate {
		return nil, nil, ErrOrderFeeTaxRateForbidden
	}
	overridden := normalized.ExchangeRateOverride != nil
	if !overridden && normalized.Currency == current.Currency && normalized.Direction == current.Direction {
		normalized.ExchangeRate, normalized.ExchangeRateSource, normalized.ExchangeRateDate, normalized.ExchangeRateSettingID = current.ExchangeRate, current.ExchangeRateSource, current.ExchangeRateDate, current.ExchangeRateSettingID
	} else if err := uc.resolveExchangeRateWithSets(ctx, organizationID, orderID, normalized, perms.CanOverrideExchangeRate, nil); err != nil {
		return nil, nil, err
	}
	if err := uc.calculateAmounts(ctx, organizationID, normalized); err != nil {
		return nil, nil, err
	}
	normalized.Version = row.ExpectedVersion
	auditFee := *normalized
	auditFee.Version = row.ExpectedVersion + 1
	audit := orderFeeAudit(organizationID, actorID, orderID, id, "order.fee.update", &auditFee)
	audit.Details["fee.save"] = "true"
	if taxRateChanged {
		audit.Details["fee.tax_rate_before"] = decimalPointerAuditText(current.TaxRate)
		audit.Details["fee.tax_rate_after"] = decimalPointerAuditText(normalized.TaxRate)
	}
	return &OrderFeeSaveRow{ClientRowID: row.ClientRowID, FeeID: &id, ExpectedVersion: row.ExpectedVersion, Fee: normalized, ExchangeRateOverridden: overridden}, audit, nil
}
