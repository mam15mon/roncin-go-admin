package service

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"github.com/shopspring/decimal"
)

var plainDecimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// OrderFeeService 订单费用服务，只做 DTO 转换、边界校验和用例调用。
type OrderFeeService struct {
	v1.UnimplementedOrderFeeServiceServer
	usecase *biz.OrderFeeUsecase
}

func NewOrderFeeService(usecase *biz.OrderFeeUsecase) *OrderFeeService {
	return &OrderFeeService{usecase: usecase}
}

func (s *OrderFeeService) ListFeeOptions(ctx context.Context, request *v1.ListFeeOptionsRequest) (*v1.ListFeeOptionsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	options, err := s.usecase.Options(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	parties := make([]*v1.OrderFeeSettlementPartyOption, 0, len(options.SettlementParties))
	for _, item := range options.SettlementParties {
		parties = append(parties, &v1.OrderFeeSettlementPartyOption{Id: item.ID.String(), Code: item.Code, Name: item.Name})
	}
	currencies := make([]*v1.OrderFeeCurrencyOption, 0, len(options.Currencies))
	for _, item := range options.Currencies {
		currencies = append(currencies, &v1.OrderFeeCurrencyOption{Code: item.Code, Name: item.Name, MinorUnit: int32(item.MinorUnit)})
	}
	feeSettings := make([]*v1.OrderFeeSettingOption, 0, len(options.FeeSettings))
	for _, item := range options.FeeSettings {
		feeSettings = append(feeSettings, &v1.OrderFeeSettingOption{
			Id: item.ID.String(), FeeCode: item.FeeCode, NameZh: item.NameZH, NameEn: item.NameEN, AliasName: item.AliasName,
			DefaultCurrency: item.DefaultCurrency, DefaultBillingUnitId: item.DefaultBillingUnitID.String(), DefaultBillingUnitName: item.DefaultBillingUnitName,
			TaxRate: item.TaxRate.StringFixed(2), TaxableServiceName: item.TaxableServiceName,
		})
	}
	billingUnits := make([]*v1.OrderFeeBillingUnitOption, 0, len(options.BillingUnits))
	for _, item := range options.BillingUnits {
		billingUnits = append(billingUnits, &v1.OrderFeeBillingUnitOption{Id: item.ID.String(), Code: item.Code, Name: item.Name})
	}
	response := &v1.ListFeeOptionsResponse{Success: true, Code: 0, Message: "OK", SettlementParties: parties, Currencies: currencies, FeeSettings: feeSettings, BillingUnits: billingUnits, TraceId: requestmeta.TraceID(ctx), BaseCurrency: options.BaseCurrency, FinanceLocked: options.FinanceLocked, FinanceLockCommissionNos: options.FinanceLockCommissionNos}
	if options.FinanceLockReason != "" {
		response.FinanceLockReason = &options.FinanceLockReason
	}
	return response, nil
}

func (s *OrderFeeService) ListFees(ctx context.Context, request *v1.ListFeesRequest) (*v1.ListFeesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderFee, 0, len(items))
	for _, item := range items {
		data = append(data, orderFeeToAPI(item))
	}
	return &v1.ListFeesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) ResolveFeeExchangeRate(ctx context.Context, request *v1.ResolveFeeExchangeRateRequest) (*v1.ResolveFeeExchangeRateResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	direction, ok := orderFeeDirectionFromAPI(request.GetDirection())
	if !ok {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	resolved, err := s.usecase.ResolveExchangeRate(ctx, principal.Organization.ID, orderID, direction, request.GetCurrency(), request.GetExpenseDate())
	if err != nil {
		return nil, err
	}
	response := &v1.ResolveFeeExchangeRateResponse{Success: true, Code: 0, Message: "OK", ExchangeRate: resolved.Rate.StringFixed(8), ExchangeRateSource: resolved.Source, ExchangeRateDate: resolved.RateDate, TraceId: requestmeta.TraceID(ctx)}
	if resolved.SettingID != nil {
		value := resolved.SettingID.String()
		response.ExchangeRateSettingId = &value
	}
	return response, nil
}

func (s *OrderFeeService) AddFee(ctx context.Context, request *v1.AddFeeRequest) (*v1.AddFeeResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, input, err := orderFeeInputFromAPI(request.GetOrderId(), request.GetDirection(), request.GetFeeSettingId(), request.GetSettlementPartyId(), request.GetBillingUnitId(), request.GetQuantity(), request.GetUnitPrice(), request.GetCurrency(), request.GetExpenseDate(), request.GetNote(), request.ExchangeRateOverride, request.GetIdempotencyKey(), request.TaxInclusive)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Add(ctx, principal.Organization.ID, principal.UserID, orderID, input, principal.HasPermission(access.FinanceExchangeRateOverride))
	if err != nil {
		return nil, err
	}
	return &v1.AddFeeResponse{Success: true, Code: 0, Message: "OK", Data: orderFeeToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) UpdateFee(ctx context.Context, request *v1.UpdateFeeRequest) (*v1.UpdateFeeResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	orderID, input, err := orderFeeInputFromAPI(request.GetOrderId(), request.GetDirection(), request.GetFeeSettingId(), request.GetSettlementPartyId(), request.GetBillingUnitId(), request.GetQuantity(), request.GetUnitPrice(), request.GetCurrency(), request.GetExpenseDate(), request.GetNote(), request.ExchangeRateOverride, "", request.TaxInclusive)
	if err != nil {
		return nil, err
	}
	input.Version = request.GetExpectedVersion()
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, orderID, id, input, principal.HasPermission(access.FinanceExchangeRateOverride))
	if err != nil {
		return nil, err
	}
	return &v1.UpdateFeeResponse{Success: true, Code: 0, Message: "OK", Data: orderFeeToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) ConfirmFee(ctx context.Context, request *v1.ConfirmFeeRequest) (*v1.ConfirmFeeResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, id, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil || request.GetExpectedVersion() == 0 {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	updated, err := s.usecase.Confirm(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmFeeResponse{Success: true, Code: 0, Message: "OK", Data: orderFeeToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) ReopenFee(ctx context.Context, request *v1.ReopenFeeRequest) (*v1.ReopenFeeResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, id, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil || request.GetExpectedVersion() == 0 {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	updated, err := s.usecase.Reopen(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.ReopenFeeResponse{Success: true, Code: 0, Message: "OK", Data: orderFeeToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) RemoveFee(ctx context.Context, request *v1.RemoveFeeRequest) (*v1.RemoveFeeResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, id, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion(), request.GetReason()); err != nil {
		return nil, err
	}
	return &v1.RemoveFeeResponse{Success: true, Code: 0, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func orderFeeToAPI(value *biz.OrderFee) *v1.OrderFee {
	result := &v1.OrderFee{
		Id:                  value.ID.String(),
		OrderId:             value.OrderID.String(),
		Direction:           orderFeeDirectionToAPI(value.Direction),
		Status:              orderFeeStatusToAPI(value.Status),
		FeeCode:             value.FeeCode,
		FeeName:             value.FeeName,
		SettlementPartyId:   value.SettlementPartyID.String(),
		SettlementPartyName: value.SettlementPartyName,
		BillingUnit:         value.BillingUnit,
		Quantity:            value.Quantity.StringFixed(4),
		UnitPrice:           value.UnitPrice.StringFixed(4),
		TotalAmount:         value.TotalAmount.StringFixed(8),
		TaxInclusive:        value.TaxInclusive,
		NetAmount:           value.NetAmount.StringFixed(8),
		TaxAmount:           value.TaxAmount.StringFixed(8),
		Currency:            value.Currency,
		ExchangeRate:        value.ExchangeRate.StringFixed(8),
		ExchangeRateSource:  value.ExchangeRateSource,
		ExchangeRateDate:    value.ExchangeRateDate,
		ExpenseDate:         value.ExpenseDate,
		BaseCurrency:        value.BaseCurrency,
		BaseCurrencyAmount:  value.BaseCurrencyAmount.StringFixed(8),
		Version:             value.Version,
		Note:                value.Note,
		CreatedAt:           value.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           value.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if value.CancelledAt != nil {
		cancelledAt := value.CancelledAt.UTC().Format(time.RFC3339)
		result.CancelledAt = &cancelledAt
	}
	if value.CancelledBy != nil {
		cancelledBy := value.CancelledBy.String()
		result.CancelledBy = &cancelledBy
	}
	result.CancellationReason = value.CancellationReason
	if value.FeeSettingID != nil {
		settingID := value.FeeSettingID.String()
		result.FeeSettingId = &settingID
	}
	if value.BillingUnitID != nil {
		billingUnitID := value.BillingUnitID.String()
		result.BillingUnitId = &billingUnitID
	}
	result.FeeNameEn = value.FeeNameEN
	if value.TaxRate != nil {
		taxRate := value.TaxRate.StringFixed(2)
		result.TaxRate = &taxRate
	}
	result.TaxableServiceName = value.TaxableServiceName
	if value.ExchangeRateSettingID != nil {
		settingID := value.ExchangeRateSettingID.String()
		result.ExchangeRateSettingId = &settingID
	}
	return result
}

func orderFeeInputFromAPI(orderIDText string, direction v1.OrderFeeDirection, feeSettingIDText, partyIDText, billingUnitIDText, quantityText, unitPriceText, currency, expenseDate, note string, exchangeRateOverrideText *string, idempotencyKey string, taxInclusive *bool) (uuid.UUID, *biz.OrderFee, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	partyID, err := uuid.Parse(partyIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	feeSettingID, err := uuid.Parse(feeSettingIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	billingUnitID, err := uuid.Parse(billingUnitIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	quantity, err := parsePlainDecimal(quantityText)
	if err != nil {
		return uuid.Nil, nil, err
	}
	unitPrice, err := parsePlainDecimal(unitPriceText)
	if err != nil {
		return uuid.Nil, nil, err
	}
	feeDirection, ok := orderFeeDirectionFromAPI(direction)
	if !ok {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	input := &biz.OrderFee{
		IdempotencyKey:    idempotencyKey,
		Direction:         feeDirection,
		FeeSettingID:      &feeSettingID,
		SettlementPartyID: partyID,
		BillingUnitID:     &billingUnitID,
		Quantity:          quantity,
		UnitPrice:         unitPrice,
		Currency:          currency,
		ExpenseDate:       expenseDate,
		TaxInclusive:      true,
	}
	if taxInclusive != nil {
		input.TaxInclusive = *taxInclusive
	}
	if exchangeRateOverrideText != nil {
		exchangeRateOverride, parseErr := parsePlainDecimal(*exchangeRateOverrideText)
		if parseErr != nil {
			return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
		}
		input.ExchangeRateOverride = &exchangeRateOverride
	}
	if note != "" {
		input.Note = &note
	}
	return orderID, input, nil
}

func parseOrderFeeIdentity(orderIDText, idText string) (uuid.UUID, uuid.UUID, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return orderID, id, nil
}

func parsePlainDecimal(value string) (decimal.Decimal, error) {
	if !plainDecimalPattern.MatchString(value) {
		return decimal.Zero, biz.ErrOrderFeeInvalidArgument
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, biz.ErrOrderFeeInvalidArgument
	}
	return parsed, nil
}

func orderFeeDirectionFromAPI(value v1.OrderFeeDirection) (biz.OrderFeeDirection, bool) {
	switch value {
	case v1.OrderFeeDirection_ORDER_FEE_DIRECTION_RECEIVABLE:
		return biz.OrderFeeReceivable, true
	case v1.OrderFeeDirection_ORDER_FEE_DIRECTION_PAYABLE:
		return biz.OrderFeePayable, true
	default:
		return "", false
	}
}

func orderFeeDirectionToAPI(value biz.OrderFeeDirection) v1.OrderFeeDirection {
	switch value {
	case biz.OrderFeeReceivable:
		return v1.OrderFeeDirection_ORDER_FEE_DIRECTION_RECEIVABLE
	case biz.OrderFeePayable:
		return v1.OrderFeeDirection_ORDER_FEE_DIRECTION_PAYABLE
	default:
		return v1.OrderFeeDirection_ORDER_FEE_DIRECTION_UNSPECIFIED
	}
}

func orderFeeStatusToAPI(value biz.OrderFeeStatus) v1.OrderFeeStatus {
	switch value {
	case biz.OrderFeeDraft:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_DRAFT
	case biz.OrderFeeConfirmed:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_CONFIRMED
	case biz.OrderFeeBilled:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_BILLED
	case biz.OrderFeeCancelled:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_CANCELLED
	default:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_UNSPECIFIED
	}
}

var _ v1.OrderFeeServiceServer = (*OrderFeeService)(nil)
