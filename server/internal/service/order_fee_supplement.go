package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// orderFeeSupplementToAPI 把补录申请领域对象转换为 API DTO；十进制字段与
// OrderFee 对齐使用普通十进制字符串。能力投影由调用方按列表用例结果填充。
func orderFeeSupplementToAPI(value *biz.OrderFeeSupplementRequest) *v1.OrderFeeSupplementRequestData {
	result := &v1.OrderFeeSupplementRequestData{
		Id:                 value.ID.String(),
		OrderId:            value.OrderID.String(),
		LockBasis:          string(value.LockBasis),
		Status:             string(value.Status),
		Version:            value.Version,
		Direction:          orderFeeDirectionToAPI(value.Fee.Direction),
		FeeCode:            value.Fee.FeeCode,
		FeeName:            value.Fee.FeeName,
		SettlementPartyId:  value.Fee.SettlementPartyID.String(),
		BillingUnit:        value.Fee.BillingUnit,
		Quantity:           value.Fee.Quantity.StringFixed(4),
		UnitPrice:          value.Fee.UnitPrice.StringFixed(4),
		TotalAmount:        value.Fee.TotalAmount.StringFixed(8),
		TaxInclusive:       value.Fee.TaxInclusive,
		NetAmount:          value.Fee.NetAmount.StringFixed(8),
		TaxAmount:          value.Fee.TaxAmount.StringFixed(8),
		Currency:           value.Fee.Currency,
		ExchangeRate:       value.Fee.ExchangeRate.StringFixed(8),
		ExchangeRateSource: value.Fee.ExchangeRateSource,
		ExchangeRateDate:   value.Fee.ExchangeRateDate,
		ExpenseDate:        value.Fee.ExpenseDate,
		BaseCurrency:       value.Fee.BaseCurrency,
		BaseCurrencyAmount: value.Fee.BaseCurrencyAmount.StringFixed(8),
		Reason:             value.Reason,
		RequestedBy:        value.RequestedBy.String(),
		RequestedByName:    value.RequestedByName,
		RequestedAt:        value.RequestedAt.UTC().Format(time.RFC3339),
	}
	if value.BusinessLockGeneration != nil {
		result.BusinessLockGeneration = value.BusinessLockGeneration
	}
	if value.FinancialLockEvidenceVersion != nil {
		version := *value.FinancialLockEvidenceVersion
		result.FinancialLockEvidenceVersion = &version
	}
	if value.FinancialLockEvidenceHash != nil {
		hash := *value.FinancialLockEvidenceHash
		result.FinancialLockEvidenceHash = &hash
	}
	if value.FinancialLockNetAmount != nil {
		net := value.FinancialLockNetAmount.StringFixed(8)
		result.FinancialLockNetAmount = &net
	}
	if value.Fee.FeeSettingID != nil {
		settingID := value.Fee.FeeSettingID.String()
		result.FeeSettingId = &settingID
	}
	result.FeeNameEn = value.Fee.FeeNameEN
	if value.Fee.BillingUnitID != nil {
		billingUnitID := value.Fee.BillingUnitID.String()
		result.BillingUnitId = &billingUnitID
	}
	if value.Fee.TaxRate != nil {
		taxRate := value.Fee.TaxRate.StringFixed(2)
		result.TaxRate = &taxRate
	}
	result.TaxableServiceName = value.Fee.TaxableServiceName
	if value.Fee.ExchangeRateSettingID != nil {
		settingID := value.Fee.ExchangeRateSettingID.String()
		result.ExchangeRateSettingId = &settingID
	}
	result.Note = value.Fee.Note
	if value.DecidedBy != nil {
		decidedBy := value.DecidedBy.String()
		result.DecidedBy = &decidedBy
	}
	result.DecidedByName = value.DecidedByName
	if value.DecidedAt != nil {
		decidedAt := value.DecidedAt.UTC().Format(time.RFC3339)
		result.DecidedAt = &decidedAt
	}
	result.DecisionReason = value.DecisionReason
	return result
}

// supplementViewCapabilities 把列表用例的能力投影填充到 DTO。
func supplementViewCapabilities(data *v1.OrderFeeSupplementRequestData, view *biz.OrderFeeSupplementRequestView) {
	data.CanApprove = view.CanApprove
	data.CanWithdraw = view.CanWithdraw
	data.CanCancel = view.CanCancel
	data.ApproverAvailable = view.ApproverAvailable
	if view.CancelBlockReason != "" {
		reason := view.CancelBlockReason
		data.CancelBlockedReason = &reason
	}
	if view.FeeID != nil {
		feeID := view.FeeID.String()
		data.FeeId = &feeID
	}
	if view.FeeStatus != "" {
		feeStatus := view.FeeStatus
		data.FeeStatus = &feeStatus
	}
}

// CreateOrderFeeSupplement 发起锁后应付费用补录申请；方向应收在服务边界与
// 领域边界双重拒绝，锁依据由服务端固化。
func (s *OrderFeeService) CreateOrderFeeSupplement(ctx context.Context, request *v1.CreateOrderFeeSupplementRequest) (*v1.CreateOrderFeeSupplementResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	feeSettingID, err := uuid.Parse(request.GetFeeSettingId())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	partyID, err := uuid.Parse(request.GetSettlementPartyId())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	billingUnitID, err := uuid.Parse(request.GetBillingUnitId())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	quantity, err := parsePlainDecimal(request.GetQuantity())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	unitPrice, err := parsePlainDecimal(request.GetUnitPrice())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	// 双重校验第一层：补录固定应付方向。
	direction, validDirection := orderFeeDirectionFromAPI(request.GetDirection())
	if !validDirection {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	input := &biz.OrderFeeSupplementCreateInput{
		Direction:         direction,
		FeeSettingID:      feeSettingID,
		SettlementPartyID: partyID,
		BillingUnitID:     billingUnitID,
		Quantity:          quantity,
		UnitPrice:         unitPrice,
		Currency:          request.GetCurrency(),
		ExpenseDate:       request.GetExpenseDate(),
		Note:              request.Note,
		TaxInclusive:      true,
		Reason:            request.GetReason(),
		IdempotencyKey:    request.GetIdempotencyKey(),
	}
	if request.TaxInclusive != nil {
		input.TaxInclusive = *request.TaxInclusive
	}
	if request.ExchangeRateOverride != nil {
		override, parseErr := parsePlainDecimal(*request.ExchangeRateOverride)
		if parseErr != nil {
			return nil, biz.ErrFeeSupplementInvalidArgument
		}
		input.ExchangeRateOverride = &override
	}
	if strings.TrimSpace(input.Reason) == "" || strings.TrimSpace(input.IdempotencyKey) == "" {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	created, err := s.supplement.Create(ctx, principal, principal.Organization.ID, orderID, input, principal.HasPermissionInScope(access.FinanceExchangeRateUpdate, biz.DataScopeOrganization))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateOrderFeeSupplementResponse{Data: orderFeeSupplementToAPI(created)}), nil
}

// ListOrderFeeSupplementRequests 订单维度分页读取补录申请；逐行授权与能力投影
// 由领域层完成。
func (s *OrderFeeService) ListOrderFeeSupplementRequests(ctx context.Context, request *v1.ListOrderFeeSupplementRequestsRequest) (*v1.ListOrderFeeSupplementRequestsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFeeSupplementInvalidArgument)
	if err != nil {
		return nil, err
	}
	list, err := s.supplement.List(ctx, principal, principal.Organization.ID, orderID, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.OrderFeeSupplementRequestData, 0, len(list.Items))
	for _, view := range list.Items {
		data := orderFeeSupplementToAPI(view.Request)
		supplementViewCapabilities(data, view)
		items = append(items, data)
	}
	return okList(ctx, &v1.ListOrderFeeSupplementRequestsResponse{
		Data: &v1.ListOrderFeeSupplementRequestsData{Items: items, Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.Size)},
	}), nil
}

// ApproveOrderFeeSupplement 审批通过补录申请；九步事务由领域层编排。
func (s *OrderFeeService) ApproveOrderFeeSupplement(ctx context.Context, request *v1.ApproveOrderFeeSupplementRequest) (*v1.ApproveOrderFeeSupplementResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, requestID, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil || request.GetExpectedVersion() == 0 {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	result, err := s.supplement.Approve(ctx, principal, principal.Organization.ID, orderID, requestID, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ApproveOrderFeeSupplementResponse{Data: orderFeeSupplementToAPI(result.Request), Fee: orderFeeToAPI(result.Fee)}), nil
}

// RejectOrderFeeSupplement 驳回补录申请，不产生费用或调整。
func (s *OrderFeeService) RejectOrderFeeSupplement(ctx context.Context, request *v1.RejectOrderFeeSupplementRequest) (*v1.RejectOrderFeeSupplementResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, requestID, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil || request.GetExpectedVersion() == 0 {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	updated, err := s.supplement.Reject(ctx, principal, principal.Organization.ID, orderID, requestID, request.GetExpectedVersion(), request.Reason)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RejectOrderFeeSupplementResponse{Data: orderFeeSupplementToAPI(updated)}), nil
}

// WithdrawOrderFeeSupplement 发起人撤回本人 PENDING 申请。
func (s *OrderFeeService) WithdrawOrderFeeSupplement(ctx context.Context, request *v1.WithdrawOrderFeeSupplementRequest) (*v1.WithdrawOrderFeeSupplementResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, requestID, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil || request.GetExpectedVersion() == 0 {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	updated, err := s.supplement.Withdraw(ctx, principal, principal.Organization.ID, orderID, requestID, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.WithdrawOrderFeeSupplementResponse{Data: orderFeeSupplementToAPI(updated)}), nil
}

// CancelApprovedOrderFeeSupplement 专用作废已批准补录生成的费用。
func (s *OrderFeeService) CancelApprovedOrderFeeSupplement(ctx context.Context, request *v1.CancelApprovedOrderFeeSupplementRequest) (*v1.CancelApprovedOrderFeeSupplementResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	orderID, requestID, err := parseOrderFeeIdentity(request.GetOrderId(), request.GetId())
	if err != nil || request.GetExpectedVersion() == 0 || strings.TrimSpace(request.GetReason()) == "" {
		return nil, biz.ErrFeeSupplementInvalidArgument
	}
	result, err := s.supplement.CancelApprovedFee(ctx, principal, principal.Organization.ID, orderID, requestID, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CancelApprovedOrderFeeSupplementResponse{Fee: orderFeeToAPI(result.Fee)}), nil
}
