package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// orderFeeSupplementToAPI 把补录申请领域对象转换为 API DTO；十进制字段与
// OrderFee 对齐使用普通十进制字符串。能力投影由调用方按列表用例结果填充。
func orderFeeSupplementToAPI(value *biz.OrderFeeSupplementRequest) *v1.OrderFeeSupplementRequestData {
	result := &v1.OrderFeeSupplementRequestData{
		Id:                  value.ID.String(),
		OrderId:             value.OrderID.String(),
		LockBasis:           string(value.LockBasis),
		Status:              string(value.Status),
		Version:             value.Version,
		Direction:           orderFeeDirectionToAPI(value.Fee.Direction),
		FeeCode:             value.Fee.FeeCode,
		FeeName:             value.Fee.FeeName,
		SettlementPartyId:   value.Fee.SettlementPartyID.String(),
		SettlementPartyName: value.Fee.SettlementPartyName,
		BillingUnit:         value.Fee.BillingUnit,
		Quantity:            value.Fee.Quantity.StringFixed(4),
		UnitPrice:           value.Fee.UnitPrice.StringFixed(4),
		TotalAmount:         value.Fee.TotalAmount.StringFixed(8),
		TaxInclusive:        value.Fee.TaxInclusive,
		NetAmount:           value.Fee.NetAmount.StringFixed(8),
		TaxAmount:           value.Fee.TaxAmount.StringFixed(8),
		Currency:            value.Fee.Currency,
		ExchangeRate:        value.Fee.ExchangeRate.StringFixed(8),
		ExchangeRateSource:  value.Fee.ExchangeRateSource,
		ExchangeRateDate:    value.Fee.ExchangeRateDate,
		BaseCurrency:        value.Fee.BaseCurrency,
		BaseCurrencyAmount:  value.Fee.BaseCurrencyAmount.StringFixed(8),
		Reason:              value.Reason,
		RequestedBy:         value.RequestedBy.String(),
		RequestedByName:     value.RequestedByName,
		RequestedAt:         value.RequestedAt.UTC().Format(time.RFC3339),
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
