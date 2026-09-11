package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *SettlementService) ListNettings(ctx context.Context, request *v1.ListNettingsRequest) (*v1.ListNettingsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceNettingInvalid)
	if err != nil {
		return nil, err
	}
	filter := biz.FinanceNettingFilter{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword),
		Status: financeNettingStatusFromAPI(request.Status), Currency: strings.ToUpper(financeOptionalString(request.Currency)),
	}
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		partyID, parseErr := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if parseErr != nil {
			return nil, biz.ErrFinanceNettingInvalid
		}
		filter.SettlementPartyID = &partyID
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceNettingRead, false, request.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.nettingUsecase.List(ctx, organizationIDs, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceNetting, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeNettingToAPI(item))
	}
	amounts := make([]*v1.FinanceNettingBaseCurrencyAmount, 0, len(result.Summary.AmountsByBaseCurrency))
	for _, item := range result.Summary.AmountsByBaseCurrency {
		amounts = append(amounts, &v1.FinanceNettingBaseCurrencyAmount{BaseCurrency: item.BaseCurrency, NettingBaseAmount: item.NettingBaseAmount.StringFixed(8)})
	}
	return okList(ctx, &v1.ListNettingsResponse{
		Data: data, Total: result.Total,
		Summary: &v1.FinanceNettingSummary{AmountsByBaseCurrency: amounts, ConfirmedCount: result.Summary.ConfirmedCount},
	}), nil
}

func (s *SettlementService) PreviewNetting(ctx context.Context, request *v1.PreviewNettingRequest) (*v1.PreviewNettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	rawOrganizationID := strings.TrimSpace(request.GetOrganizationId())
	settlementPartyID, partyErr := uuid.Parse(strings.TrimSpace(request.GetSettlementPartyId()))
	if rawOrganizationID == "" || partyErr != nil {
		return nil, biz.ErrFinanceNettingInvalid
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceNettingCreate, true, &rawOrganizationID)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrFinanceNettingInvalid
	}
	preview, err := s.nettingUsecase.Preview(ctx, organizationIDs[0], settlementPartyID, request.GetCurrency())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewNettingResponse{Data: financeNettingPreviewToAPI(preview)}), nil
}

func (s *SettlementService) CreateNetting(ctx context.Context, request *v1.CreateNettingRequest) (*v1.CreateNettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	rawOrganizationID := strings.TrimSpace(request.GetOrganizationId())
	if rawOrganizationID == "" {
		return nil, biz.ErrFinanceNettingInvalid
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceNettingCreate, true, &rawOrganizationID)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrFinanceNettingInvalid
	}
	bills := make([]biz.FinanceNettingBillVersion, 0, len(request.GetBills()))
	for _, item := range request.GetBills() {
		if item == nil {
			return nil, biz.ErrFinanceNettingInvalid
		}
		billID, parseErr := uuid.Parse(strings.TrimSpace(item.GetBillId()))
		if parseErr != nil {
			return nil, biz.ErrFinanceNettingInvalid
		}
		bills = append(bills, biz.FinanceNettingBillVersion{BillID: billID, ExpectedVersion: item.GetExpectedVersion()})
	}
	item, err := s.nettingUsecase.Create(ctx, organizationIDs[0], principal.UserID, biz.CreateFinanceNettingInput{Bills: bills, Note: request.Note, IdempotencyKey: request.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateNettingResponse{Data: financeNettingToAPI(item)}), nil
}

func (s *SettlementService) GetNetting(ctx context.Context, request *v1.GetNettingRequest) (*v1.GetNettingResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(principal, access.FinanceNettingRead, false)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.nettingUsecase.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetNettingResponse{Data: financeNettingToAPI(item)}), nil
}

func (s *SettlementService) ConfirmNetting(ctx context.Context, request *v1.ConfirmNettingRequest) (*v1.ConfirmNettingResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(principal, access.FinanceNettingConfirm, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.nettingUsecase.Confirm(ctx, organizationIDs, principal.UserID, id, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmNettingResponse{Data: financeNettingToAPI(item)}), nil
}

func (s *SettlementService) CancelNetting(ctx context.Context, request *v1.CancelNettingRequest) (*v1.CancelNettingResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(principal, access.FinanceNettingReverse, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.nettingUsecase.Cancel(ctx, organizationIDs, principal.UserID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CancelNettingResponse{Data: financeNettingToAPI(item)}), nil
}

func (s *SettlementService) ReverseNetting(ctx context.Context, request *v1.ReverseNettingRequest) (*v1.ReverseNettingResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(principal, access.FinanceNettingReverse, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.nettingUsecase.Reverse(ctx, organizationIDs, principal.UserID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ReverseNettingResponse{Data: financeNettingToAPI(item)}), nil
}

func financeNettingStatusFromAPI(value *v1.FinanceNettingStatus) biz.FinanceNettingStatus {
	if value == nil {
		return ""
	}
	return biz.FinanceNettingStatus(strings.TrimPrefix(value.String(), "FINANCE_NETTING_STATUS_"))
}

func financeNettingStatusToAPI(value biz.FinanceNettingStatus) v1.FinanceNettingStatus {
	switch value {
	case biz.FinanceNettingDraft:
		return v1.FinanceNettingStatus_FINANCE_NETTING_STATUS_DRAFT
	case biz.FinanceNettingConfirmed:
		return v1.FinanceNettingStatus_FINANCE_NETTING_STATUS_CONFIRMED
	case biz.FinanceNettingCancelled:
		return v1.FinanceNettingStatus_FINANCE_NETTING_STATUS_CANCELLED
	case biz.FinanceNettingReversed:
		return v1.FinanceNettingStatus_FINANCE_NETTING_STATUS_REVERSED
	default:
		return v1.FinanceNettingStatus_FINANCE_NETTING_STATUS_UNSPECIFIED
	}
}

func financeNettingToAPI(item *biz.FinanceNetting) *v1.FinanceNetting {
	if item == nil {
		return nil
	}
	allocations := make([]*v1.FinanceNettingAllocation, 0, len(item.Allocations))
	for _, allocation := range item.Allocations {
		allocations = append(allocations, &v1.FinanceNettingAllocation{
			Id: allocation.ID.String(), BillId: allocation.BillID.String(), BillNo: allocation.BillNo,
			Direction: string(allocation.Direction), Amount: allocation.Amount.StringFixed(8),
			BaseCurrencyAmount: allocation.BaseCurrencyAmount.StringFixed(8), Active: allocation.Active,
		})
	}
	return &v1.FinanceNetting{
		Id: item.ID.String(), NettingNo: item.NettingNo, Status: financeNettingStatusToAPI(item.Status),
		OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName,
		SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName,
		Currency: item.Currency, Amount: item.Amount.StringFixed(8), BaseCurrency: item.BaseCurrency,
		BaseCurrencyAmount: item.BaseCurrencyAmount.StringFixed(8), Note: item.Note, Version: item.Version,
		Allocations: allocations,
		CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		ConfirmedAt: financeTime(item.ConfirmedAt), CancelledAt: financeTime(item.CancelledAt), CancellationReason: item.CancellationReason,
		ReversedAt: financeTime(item.ReversedAt), ReversalReason: item.ReversalReason,
		BatchId: uuidStringPtr(item.BatchID), BatchNo: financeOptionalValue(item.BatchNo),
	}
}

func financeNettingPreviewToAPI(preview *biz.FinanceNettingPreview) *v1.FinanceNettingPreview {
	if preview == nil {
		return nil
	}
	toBalances := func(items []*biz.FinanceNettingBillBalance) []*v1.FinanceNettingBillBalance {
		result := make([]*v1.FinanceNettingBillBalance, 0, len(items))
		for _, item := range items {
			result = append(result, &v1.FinanceNettingBillBalance{
				BillId: item.BillID.String(), BillNo: item.BillNo, BillDate: item.BillDate,
				TotalAmount: item.TotalAmount.StringFixed(8), VerifiedAmount: item.VerifiedAmount.StringFixed(8),
				NettedAmount: item.NettedAmount.StringFixed(8), AvailableAmount: item.AvailableAmount.StringFixed(8),
				Version: item.Version,
			})
		}
		return result
	}
	return &v1.FinanceNettingPreview{
		OrganizationId: preview.OrganizationID.String(), OrganizationName: preview.OrganizationName,
		SettlementPartyId: preview.SettlementPartyID.String(), SettlementPartyName: preview.SettlementPartyName,
		Currency: preview.Currency, ReceivableBills: toBalances(preview.ReceivableBills), PayableBills: toBalances(preview.PayableBills),
		ReceivableAvailableAmount: preview.ReceivableAvailableAmount.StringFixed(8), PayableAvailableAmount: preview.PayableAvailableAmount.StringFixed(8),
		OffsetAmount: preview.OffsetAmount.StringFixed(8), NetReceivableAmount: preview.NetReceivableAmount.StringFixed(8), NetPayableAmount: preview.NetPayableAmount.StringFixed(8),
	}
}

func billBatchNettingPairsToAPI(pairs []*biz.FinanceBillBatchNettingPair) []*v1.BillBatchNettingPair {
	result := make([]*v1.BillBatchNettingPair, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, &v1.BillBatchNettingPair{
			SettlementPartyId: pair.SettlementPartyID.String(), SettlementPartyName: pair.SettlementPartyName, Currency: pair.Currency,
			ReceivableGrossAmount: pair.ReceivableGrossAmount.StringFixed(8), PayableGrossAmount: pair.PayableGrossAmount.StringFixed(8),
			OffsetAmount: pair.OffsetAmount.StringFixed(8), NetReceivableAmount: pair.NetReceivableAmount.StringFixed(8), NetPayableAmount: pair.NetPayableAmount.StringFixed(8),
		})
	}
	return result
}
