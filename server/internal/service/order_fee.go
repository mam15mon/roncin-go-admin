package service

import (
	"context"
	"regexp"
	"strings"
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
	usecase    *biz.OrderFeeUsecase
	tagUsecase *biz.BusinessTagUsecase
	supplement *biz.OrderFeeSupplementUsecase
}

func NewOrderFeeService(usecase *biz.OrderFeeUsecase, tagUsecase *biz.BusinessTagUsecase, supplement *biz.OrderFeeSupplementUsecase) *OrderFeeService {
	return &OrderFeeService{usecase: usecase, tagUsecase: tagUsecase, supplement: supplement}
}

func (s *OrderFeeService) ListFeeOptions(ctx context.Context, request *v1.ListFeeOptionsRequest) (*v1.ListFeeOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
		billingUnits = append(billingUnits, &v1.OrderFeeBillingUnitOption{Id: item.ID.String(), Code: item.Code, Name: item.Name, QuantityMustBeInteger: item.QuantityMustBeInteger})
	}
	response := okList(ctx, &v1.ListFeeOptionsResponse{SettlementParties: parties, Currencies: currencies, FeeSettings: feeSettings, BillingUnits: billingUnits, BaseCurrency: options.BaseCurrency, FinanceLocked: options.FinanceLocked, FinanceLockCommissionNos: options.FinanceLockCommissionNos, CustomerId: options.CustomerID.String(), CustomerName: options.CustomerName})
	if options.FinanceLockReason != "" {
		response.FinanceLockReason = &options.FinanceLockReason
	}
	return response, nil
}

func (s *OrderFeeService) ListFees(ctx context.Context, request *v1.ListFeesRequest) (*v1.ListFeesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	feeIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		feeIDs = append(feeIDs, item.ID)
	}
	feeTags, err := s.tagUsecase.LoadOrderFeeTags(ctx, feeIDs)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderFee, 0, len(items))
	for _, item := range items {
		converted := orderFeeToAPI(item)
		converted.Tags = businessTagSummariesToOrderAPI(feeTags[item.ID])
		data = append(data, converted)
	}
	return okList(ctx, &v1.ListFeesResponse{Data: data}), nil
}

func (s *OrderFeeService) ResolveFeeExchangeRate(ctx context.Context, request *v1.ResolveFeeExchangeRateRequest) (*v1.ResolveFeeExchangeRateResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	direction, valid := orderFeeDirectionFromAPI(request.GetDirection())
	if !valid {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	resolved, err := s.usecase.ResolveExchangeRate(ctx, principal.Organization.ID, orderID, direction, request.GetCurrency(), request.GetExpenseDate())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ResolveFeeExchangeRateResponse{ExchangeRate: resolved.Rate.StringFixed(8), ExchangeRateSource: resolved.Source, ExchangeRateDate: request.GetExpenseDate()}), nil
}

func (s *OrderFeeService) AddFee(ctx context.Context, request *v1.AddFeeRequest) (*v1.AddFeeResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, input, err := orderFeeInputFromAPI(request.GetOrderId(), request.GetDirection(), request.GetFeeSettingId(), request.GetSettlementPartyId(), request.GetBillingUnitId(), request.GetQuantity(), request.GetUnitPrice(), request.GetCurrency(), request.GetExpenseDate(), request.GetNote(), request.ExchangeRateOverride, request.GetIdempotencyKey(), request.TaxInclusive)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Add(ctx, principal.Organization.ID, principal.UserID, orderID, input, principal.HasPermissionInScope(access.FinanceExchangeRateUpdate, biz.DataScopeOrganization))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.AddFeeResponse{Data: orderFeeToAPI(created)}), nil
}

func (s *OrderFeeService) UpdateFee(ctx context.Context, request *v1.UpdateFeeRequest) (*v1.UpdateFeeResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	if request.TaxRate != nil {
		taxRate, parseErr := parsePlainDecimal(*request.TaxRate)
		if parseErr != nil {
			return nil, biz.ErrOrderFeeInvalidArgument
		}
		input.TaxRateOverride = &taxRate
	}
	if request.FeeName != nil {
		input.FeeNameOverride = request.FeeName
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, orderID, id, input, principal.HasPermissionInScope(access.FinanceExchangeRateUpdate, biz.DataScopeOrganization))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateFeeResponse{Data: orderFeeToAPI(updated)}), nil
}

func (s *OrderFeeService) RemoveFee(ctx context.Context, request *v1.RemoveFeeRequest) (*v1.RemoveFeeResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, id, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	if err := s.usecase.Remove(ctx, principal.Organization.ID, principal.UserID, orderID, id, request.GetExpectedVersion(), request.GetReason()); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RemoveFeeResponse{}), nil
}

func (s *OrderFeeService) BulkUpdateOrderFees(ctx context.Context, request *v1.BulkUpdateOrderFeesRequest) (*v1.BulkUpdateOrderFeesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, targets, err := parseOrderFeeBulkTargets(request.GetOrderId(), request.GetTargets())
	if err != nil {
		return nil, err
	}
	settlementPartyID, expenseDate, err := parseOrderFeeBulkUpdateValue(request.SettlementPartyId, request.ExpenseDate)
	if err != nil {
		return nil, err
	}
	if err := s.usecase.BulkUpdate(ctx, principal.Organization.ID, principal.UserID, orderID, targets, settlementPartyID, expenseDate); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.BulkUpdateOrderFeesResponse{UpdatedCount: int32(len(targets))}), nil
}

func (s *OrderFeeService) BulkRemoveOrderFees(ctx context.Context, request *v1.BulkRemoveOrderFeesRequest) (*v1.BulkRemoveOrderFeesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, targets, err := parseOrderFeeBulkTargets(request.GetOrderId(), request.GetTargets())
	if err != nil {
		return nil, err
	}
	if err := s.usecase.BulkRemove(ctx, principal.Organization.ID, principal.UserID, orderID, targets, request.GetReason()); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.BulkRemoveOrderFeesResponse{RemovedCount: int32(len(targets))}), nil
}

// parseOrderFeeBulkTargets 解析并结构校验批量目标集合：订单与费用 ID 必须是
// 合法 UUID、目标非空且费用 ID 不重复、乐观锁版本非零。
func parseOrderFeeBulkTargets(orderIDText string, requestTargets []*v1.BulkOrderFeeTarget) (uuid.UUID, []biz.OrderFeeBulkTarget, error) {
	orderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	if len(requestTargets) == 0 {
		return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	targets := make([]biz.OrderFeeBulkTarget, 0, len(requestTargets))
	seen := make(map[uuid.UUID]struct{}, len(requestTargets))
	for _, item := range requestTargets {
		feeID, parseErr := uuid.Parse(item.GetFeeId())
		if parseErr != nil || item.GetExpectedVersion() == 0 {
			return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
		}
		if _, exists := seen[feeID]; exists {
			return uuid.Nil, nil, biz.ErrOrderFeeInvalidArgument
		}
		seen[feeID] = struct{}{}
		targets = append(targets, biz.OrderFeeBulkTarget{FeeID: feeID, ExpectedVersion: item.GetExpectedVersion()})
	}
	return orderID, targets, nil
}

// parseOrderFeeBulkUpdateValue 解析批量修改的目标值：结算单位 ID 或费用时间
// 二选一，必须恰好提供一个且结算单位 ID 是合法 UUID；日期格式由领域层校验。
func parseOrderFeeBulkUpdateValue(settlementPartyIDText, expenseDateText *string) (*uuid.UUID, *string, error) {
	if (settlementPartyIDText == nil) == (expenseDateText == nil) {
		return nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	if settlementPartyIDText != nil {
		partyID, err := uuid.Parse(*settlementPartyIDText)
		if err != nil || partyID == uuid.Nil {
			return nil, nil, biz.ErrOrderFeeInvalidArgument
		}
		return &partyID, nil, nil
	}
	return nil, expenseDateText, nil
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
	case biz.OrderFeeUnbilled:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_UNBILLED
	case biz.OrderFeeBilled:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_BILLED
	case biz.OrderFeeCancelled:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_CANCELLED
	default:
		return v1.OrderFeeStatus_ORDER_FEE_STATUS_UNSPECIFIED
	}
}

func orderFeeStatusFromAPI(value *v1.OrderFeeStatus) biz.OrderFeeStatus {
	if value == nil {
		return ""
	}
	return biz.OrderFeeStatus(strings.TrimPrefix(value.String(), "ORDER_FEE_STATUS_"))
}

var _ v1.OrderFeeServiceServer = (*OrderFeeService)(nil)

func businessTagSummariesToOrderAPI(items []*biz.BusinessTagSummary) []*v1.BusinessTagSummary {
	if len(items) == 0 {
		return nil
	}
	result := make([]*v1.BusinessTagSummary, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.BusinessTagSummary{Id: item.ID.String(), Name: item.Name, GroupId: item.GroupID.String(), GroupName: item.GroupName, GroupColor: item.GroupColor, Enabled: item.Enabled})
	}
	return result
}

func (s *OrderFeeService) ListOrderFeeTagOptions(ctx context.Context, request *v1.ListOrderFeeTagOptionsRequest) (*v1.ListOrderFeeTagOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrBusinessTagInvalidArgument)
	if err != nil {
		return nil, err
	}
	items, total, err := s.tagUsecase.ListTagOptions(ctx, principal.Organization.ID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	return &v1.ListOrderFeeTagOptionsResponse{Tags: businessTagSummariesToOrderAPI(items), Total: total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) BatchAssignOrderFeeTags(ctx context.Context, request *v1.BatchAssignOrderFeeTagsRequest) (*v1.BatchAssignOrderFeeTagsResponse, error) {
	principal, orderID, feeIDs, tagIDs, err := orderFeeTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.AssignOrderFeesInOrder(ctx, principal.Organization.ID, principal.UserID, orderID, feeIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignOrderFeeTagsResponse{AssignedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderFeeService) BatchRemoveOrderFeeTags(ctx context.Context, request *v1.BatchRemoveOrderFeeTagsRequest) (*v1.BatchRemoveOrderFeeTagsResponse, error) {
	principal, orderID, feeIDs, tagIDs, err := orderFeeTagRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	affected, err := s.tagUsecase.RemoveOrderFeesInOrder(ctx, principal.Organization.ID, principal.UserID, orderID, feeIDs, tagIDs)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveOrderFeeTagsResponse{RemovedCount: int32(affected), TraceId: requestmeta.TraceID(ctx)}, nil
}

func orderFeeTagRequest[Req interface {
	GetOrderId() string
	GetFeeIds() []string
	GetTagIds() []string
}](ctx context.Context, request Req) (*biz.Principal, uuid.UUID, []uuid.UUID, []uuid.UUID, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, uuid.Nil, nil, nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, uuid.Nil, nil, nil, biz.ErrOrderFeeInvalidArgument
	}
	feeIDs, tagIDs, err := orderTagBatchIDs(request.GetFeeIds(), request.GetTagIds())
	if err != nil {
		return nil, uuid.Nil, nil, nil, err
	}
	return principal, orderID, feeIDs, tagIDs, nil
}
