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

func (s *SettlementService) ListInvoices(ctx context.Context, request *v1.ListInvoicesRequest) (*v1.ListInvoicesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceInvoiceInvalidArgument)
	if err != nil {
		return nil, err
	}
	filter := biz.FinanceInvoiceFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword), Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction))), Status: financeInvoiceStatusFromAPI(request.Status)}
	if request.BillId != nil {
		id, err := uuid.Parse(strings.TrimSpace(*request.BillId))
		if err != nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		filter.BillID = &id
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceInvoiceRead, false, request.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.invoiceUsecase.ListScoped(ctx, organizationIDs, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceInvoice, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeInvoiceToAPI(item))
	}
	return okList(ctx, &v1.ListInvoicesResponse{
		MatchedSummary: financeLedgerMatchedSummaryToAPI(result.MatchedSummary),
		Data:           data, Total: result.Total,
		Summary: &v1.FinanceInvoiceSummary{AmountsByBaseCurrency: financeBaseCurrencyAmountsToAPI(result.Summary.AmountsByBaseCurrency), IssuedCount: result.Summary.IssuedCount},
	}), nil
}

func (s *SettlementService) ListInvoiceCreationBills(ctx context.Context, request *v1.ListInvoiceCreationBillsRequest) (*v1.ListInvoiceCreationBillsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	rawOrganizationID := strings.TrimSpace(request.GetOrganizationId())
	if rawOrganizationID == "" {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceInvoiceCreate, true, &rawOrganizationID)
	if err != nil {
		return nil, err
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceInvoiceInvalidArgument)
	if err != nil {
		return nil, err
	}
	filter := biz.FinanceInvoiceCreationBillFilter{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword),
		Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction))),
		Currency:  strings.ToUpper(financeOptionalString(request.Currency)),
	}
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		settlementPartyID, parseErr := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if parseErr != nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		filter.SettlementPartyID = &settlementPartyID
	}
	if request.BillId != nil {
		id, err := uuid.Parse(strings.TrimSpace(*request.BillId))
		if err != nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		filter.BillID = &id
	}
	result, err := s.invoiceUsecase.ListCreationBills(ctx, organizationIDs[0], filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceBill, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeBillToAPI(item))
	}
	return okList(ctx, &v1.ListInvoiceCreationBillsResponse{Data: data, Total: result.Total}), nil
}

func (s *SettlementService) ListInvoiceProfilesForBill(ctx context.Context, request *v1.ListInvoiceProfilesForBillRequest) (*v1.ListInvoiceProfilesForBillResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	billID, err := uuid.Parse(strings.TrimSpace(request.GetBillId()))
	if err != nil {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceInvoiceCreate, true)
	if err != nil {
		return nil, err
	}
	result, err := s.invoiceUsecase.ListProfilesForBill(ctx, organizationIDs, billID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceInvoiceProfileOption, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.FinanceInvoiceProfileOption{
			Id:                       item.ID.String(),
			InvoiceTitle:             item.InvoiceTitle,
			TaxpayerIdentificationNo: item.TaxpayerIdentificationNo,
			DefaultInvoiceType:       string(item.DefaultInvoiceType),
			IsDefault:                item.IsDefault,
		})
	}
	return okList(ctx, &v1.ListInvoiceProfilesForBillResponse{Data: &v1.FinanceInvoiceProfilesForBill{
		OrganizationId: result.OrganizationID.String(), SettlementPartyId: result.SettlementPartyID.String(), Data: data,
	}}), nil
}

func (s *SettlementService) GetInvoice(ctx context.Context, request *v1.GetInvoiceRequest) (*v1.GetInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceInvoiceRead, false)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.invoiceUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetInvoiceResponse{Data: financeInvoiceToAPI(item)}), nil
}
func (s *SettlementService) CancelInvoice(ctx context.Context, request *v1.CancelInvoiceRequest) (*v1.CancelInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceInvoiceUpdate, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	existing, err := s.invoiceUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.invoiceUsecase.Cancel(ctx, existing.OrganizationID, p.UserID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CancelInvoiceResponse{Data: financeInvoiceToAPI(item)}), nil
}
func (s *SettlementService) RedFlushInvoice(ctx context.Context, request *v1.RedFlushInvoiceRequest) (*v1.RedFlushInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceInvoiceUpdate, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	existing, err := s.invoiceUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.invoiceUsecase.RedFlush(ctx, existing.OrganizationID, p.UserID, id, request.GetExpectedVersion(), request.GetRedInvoiceNo(), request.GetRedInvoiceDate(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RedFlushInvoiceResponse{Data: financeInvoiceToAPI(item)}), nil
}
func financeInvoiceToAPI(item *biz.FinanceInvoice) *v1.FinanceInvoice {
	if item == nil {
		return nil
	}
	links := make([]*v1.FinanceInvoiceBill, 0, len(item.Links))
	for _, l := range item.Links {
		sourceIDs := make([]string, 0, len(l.SourceFeeIDs))
		for _, id := range l.SourceFeeIDs {
			sourceIDs = append(sourceIDs, id.String())
		}
		fees := make([]*v1.FinanceInvoiceFeeAllocation, 0, len(l.Fees))
		for _, fee := range l.Fees {
			fees = append(fees, invoiceAllocationToAPI(fee))
		}
		links = append(links, &v1.FinanceInvoiceBill{SourceFeeIds: sourceIDs, Id: l.ID.String(), BillId: l.BillID.String(), BillNo: l.BillNo, Amount: l.Amount.StringFixed(8), TaxAmount: l.TaxAmount.StringFixed(8), Active: l.Active, Fees: fees})
	}
	lines := make([]*v1.FinanceInvoiceLine, 0, len(item.Lines))
	for _, line := range item.Lines {
		lines = append(lines, &v1.FinanceInvoiceLine{Id: line.ID.String(), LineNo: int32(line.LineNo), ItemCode: line.ItemCode, ItemName: line.ItemName, TaxRate: financeDecimalPointer(line.TaxRate, 4), NetAmount: line.NetAmount.StringFixed(8), TaxAmount: line.TaxAmount.StringFixed(8), TotalAmount: line.TotalAmount.StringFixed(8), Currency: line.Currency, SourceLineCount: int32(line.SourceLineCount)})
	}
	revisions := make([]*v1.FinanceInvoiceRevision, 0, len(item.Revisions))
	for _, revision := range item.Revisions {
		revisions = append(revisions, invoiceRevisionToAPI(revision))
	}
	return &v1.FinanceInvoice{Revisions: revisions, Id: item.ID.String(), RecordNo: item.RecordNo, Direction: string(item.Direction), Status: financeInvoiceStatusToAPI(item.Status), InvoiceType: string(item.InvoiceType), OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName, SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName, Currency: item.Currency, BaseCurrency: item.BaseCurrency, ExchangeRate: financeDecimalPointer(item.ExchangeRate, 8), ExchangeRateSource: item.ExchangeRateSource, ExchangeRateDate: item.ExchangeRateDate, ExchangeRateSettingId: uuidStringPtr(item.ExchangeRateSettingID), BaseCurrencyAmount: financeDecimalPointer(item.BaseCurrencyAmount, 8), TotalAmount: item.TotalAmount.StringFixed(8), NetAmount: item.NetAmount.StringFixed(8), TaxAmount: item.TaxAmount.StringFixed(8), BillCount: int32(item.BillCount), TaxInvoiceNo: item.TaxInvoiceNo, InvoiceDate: item.InvoiceDate, Note: item.Note, Version: item.Version, IssuedAt: financeTime(item.IssuedAt), CancelledAt: financeTime(item.CancelledAt), CancellationReason: item.CancellationReason, RedInvoiceNo: item.RedInvoiceNo, RedInvoiceDate: item.RedInvoiceDate, RedFlushedAt: financeTime(item.RedFlushedAt), RedFlushReason: item.RedFlushReason, BillLinks: links, InvoiceProfileId: uuidStringPtr(item.InvoiceProfileID), InvoiceTitle: financeOptionalValue(item.InvoiceTitle), TaxpayerIdentificationNo: financeOptionalValue(item.TaxpayerIdentificationNo), RegisteredAddress: financeOptionalValue(item.RegisteredAddress), RegisteredPhone: financeOptionalValue(item.RegisteredPhone), BankName: financeOptionalValue(item.BankName), BankAccount: financeOptionalValue(item.BankAccount), Lines: lines, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}
