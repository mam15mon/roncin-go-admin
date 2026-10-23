package biz

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

// OrderFeeSupplementUsecase 只读查询退役补录流程的历史申请。
type OrderFeeSupplementUsecase struct{ repo OrderFeeSupplementRequestRepo }

func NewOrderFeeSupplementUsecase(repo OrderFeeSupplementRequestRepo) *OrderFeeSupplementUsecase {
	return &OrderFeeSupplementUsecase{repo: repo}
}
func validSupplementCaller(caller *Principal, organizationID, orderID uuid.UUID) bool {
	return caller != nil && caller.UserID != uuid.Nil && organizationID != uuid.Nil && orderID != uuid.Nil
}

type OrderFeeSupplementListView struct {
	Items []*OrderFeeSupplementRequestView
	Total int
	Page  int
	Size  int
}

// OrderFeeSupplementRequestView 在申请领域对象之上追加当前调用人的能力投影。
type OrderFeeSupplementRequestView struct {
	Request           *OrderFeeSupplementRequest
	CanApprove        bool
	CanWithdraw       bool
	CanCancel         bool
	CancelBlockReason string
	CancelBlockCode   string
	FeeID             *uuid.UUID
	ApproverAvailable bool
}

func feeSnapshotFromOrderFee(fee *OrderFee) OrderFeeSupplementFeeSnapshot {
	return OrderFeeSupplementFeeSnapshot{
		Direction:             fee.Direction,
		FeeSettingID:          fee.FeeSettingID,
		FeeCode:               fee.FeeCode,
		FeeName:               fee.FeeName,
		FeeNameEN:             fee.FeeNameEN,
		SettlementPartyID:     fee.SettlementPartyID,
		BillingUnitID:         fee.BillingUnitID,
		BillingUnit:           fee.BillingUnit,
		TaxRate:               fee.TaxRate,
		TaxableServiceName:    fee.TaxableServiceName,
		Quantity:              fee.Quantity,
		UnitPrice:             fee.UnitPrice,
		TotalAmount:           fee.TotalAmount,
		TaxInclusive:          fee.TaxInclusive,
		NetAmount:             fee.NetAmount,
		TaxAmount:             fee.TaxAmount,
		Currency:              fee.Currency,
		ExchangeRate:          fee.ExchangeRate,
		ExchangeRateSource:    fee.ExchangeRateSource,
		ExchangeRateDate:      fee.ExchangeRateDate,
		ExchangeRateSettingID: fee.ExchangeRateSettingID,
		BaseCurrency:          fee.BaseCurrency,
		BaseCurrencyAmount:    fee.BaseCurrencyAmount,
		ExpenseDate:           fee.ExpenseDate,
		Note:                  fee.Note,
	}
}

func (uc *OrderFeeSupplementUsecase) List(ctx context.Context, caller *Principal, organizationID, orderID uuid.UUID, page, pageSize int) (*OrderFeeSupplementListView, error) {
	if !validSupplementCaller(caller, organizationID, orderID) || !ValidListPagination(page, pageSize) {
		return nil, ErrFeeSupplementInvalidArgument
	}
	orderRef, err := uc.repo.GetOrderRef(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	// fee.read 逐行授权：按订单业务类型解析调用人的组织范围；不持有该权限时
	// 仅本人申请或实时具备费用审核权限 的行可见。
	hasFeeRead := false
	if businessType := access.OrderBusinessType(orderRef.BusinessType); businessType.Valid() {
		permission := access.OrderPermission(businessType, access.OrderFeeRead)
		hasFeeRead = caller.CanAccessOrganizationForPermission(permission, organizationID, false)
	}
	grant, err := uc.repo.HasRealtimeLockGrant(ctx, organizationID, orderID, caller.UserID, caller.IsBootstrapAdmin)
	if err != nil {
		return nil, err
	}
	rows, err := uc.repo.ListByOrder(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	visible := make([]*OrderFeeSupplementRequest, 0, len(rows))
	for _, row := range rows {
		if hasFeeRead || grant || row.RequestedBy == caller.UserID {
			visible = append(visible, row)
		}
	}
	result := &OrderFeeSupplementListView{Items: make([]*OrderFeeSupplementRequestView, 0, len(visible)), Total: len(visible), Page: page, Size: pageSize}
	start := (page - 1) * pageSize
	if start >= len(visible) {
		return result, nil
	}
	end := start + pageSize
	if end > len(visible) {
		end = len(visible)
	}
	for _, row := range visible[start:end] {
		view := &OrderFeeSupplementRequestView{
			Request:           row,
			CanApprove:        false,
			CanWithdraw:       false,
			ApproverAvailable: grant,
		}

		result.Items = append(result.Items, view)
	}
	return result, nil
}
