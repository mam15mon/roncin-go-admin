package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
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
	result, err := s.invoiceUsecase.List(ctx, principal.Organization.ID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceInvoice, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeInvoiceToAPI(item))
	}
	return &v1.ListInvoicesResponse{
		Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx),
		Summary: &v1.FinanceInvoiceSummary{
			ReceivableBaseAmount: result.Summary.ReceivableBaseAmount.StringFixed(8),
			PayableBaseAmount:    result.Summary.PayableBaseAmount.StringFixed(8),
			IssuedCount:          result.Summary.IssuedCount,
			BaseCurrency:         result.Summary.BaseCurrency,
		},
	}, nil
}
func (s *SettlementService) GetInvoice(ctx context.Context, request *v1.GetInvoiceRequest) (*v1.GetInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.invoiceUsecase.Get(ctx, p.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetInvoiceResponse{Success: true, Message: "OK", Data: financeInvoiceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateInvoice(ctx context.Context, request *v1.CreateInvoiceRequest) (*v1.CreateInvoiceResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	ids := make([]uuid.UUID, 0, len(request.GetBillIds()))
	for _, raw := range request.GetBillIds() {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		ids = append(ids, id)
	}
	profileID, err := uuid.Parse(strings.TrimSpace(request.GetInvoiceProfileId()))
	if err != nil {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	item, err := s.invoiceUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateFinanceInvoiceInput{BillIDs: ids, InvoiceProfileID: profileID, InvoiceType: biz.FinanceInvoiceType(strings.ToUpper(request.GetInvoiceType())), Note: request.Note, IdempotencyKey: request.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return &v1.CreateInvoiceResponse{Success: true, Message: "OK", Data: financeInvoiceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) IssueInvoice(ctx context.Context, request *v1.IssueInvoiceRequest) (*v1.IssueInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.invoiceUsecase.Issue(ctx, p.Organization.ID, p.UserID, id, request.GetExpectedVersion(), request.GetTaxInvoiceNo(), request.GetInvoiceDate())
	if err != nil {
		return nil, err
	}
	return &v1.IssueInvoiceResponse{Success: true, Message: "OK", Data: financeInvoiceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CancelInvoice(ctx context.Context, request *v1.CancelInvoiceRequest) (*v1.CancelInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.invoiceUsecase.Cancel(ctx, p.Organization.ID, p.UserID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.CancelInvoiceResponse{Success: true, Message: "OK", Data: financeInvoiceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) RedFlushInvoice(ctx context.Context, request *v1.RedFlushInvoiceRequest) (*v1.RedFlushInvoiceResponse, error) {
	p, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.invoiceUsecase.RedFlush(ctx, p.Organization.ID, p.UserID, id, request.GetExpectedVersion(), request.GetRedInvoiceNo(), request.GetRedInvoiceDate(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.RedFlushInvoiceResponse{Success: true, Message: "OK", Data: financeInvoiceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func financeInvoiceToAPI(item *biz.FinanceInvoice) *v1.FinanceInvoice {
	if item == nil {
		return nil
	}
	links := make([]*v1.FinanceInvoiceBill, 0, len(item.Links))
	for _, l := range item.Links {
		links = append(links, &v1.FinanceInvoiceBill{Id: l.ID.String(), BillId: l.BillID.String(), BillNo: l.BillNo, Amount: l.Amount.StringFixed(8), TaxAmount: l.TaxAmount.StringFixed(8), Active: l.Active})
	}
	lines := make([]*v1.FinanceInvoiceLine, 0, len(item.Lines))
	for _, line := range item.Lines {
		lines = append(lines, &v1.FinanceInvoiceLine{Id: line.ID.String(), LineNo: int32(line.LineNo), ItemCode: line.ItemCode, ItemName: line.ItemName, TaxRate: line.TaxRate.StringFixed(4), NetAmount: line.NetAmount.StringFixed(8), TaxAmount: line.TaxAmount.StringFixed(8), TotalAmount: line.TotalAmount.StringFixed(8), Currency: line.Currency, SourceLineCount: int32(line.SourceLineCount)})
	}
	return &v1.FinanceInvoice{Id: item.ID.String(), RecordNo: item.RecordNo, Direction: string(item.Direction), Status: financeInvoiceStatusToAPI(item.Status), InvoiceType: string(item.InvoiceType), SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName, Currency: item.Currency, BaseCurrency: item.BaseCurrency, ExchangeRate: financeDecimalPointer(item.ExchangeRate, 8), ExchangeRateSource: item.ExchangeRateSource, ExchangeRateDate: item.ExchangeRateDate, ExchangeRateSettingId: uuidStringPtr(item.ExchangeRateSettingID), BaseCurrencyAmount: financeDecimalPointer(item.BaseCurrencyAmount, 8), TotalAmount: item.TotalAmount.StringFixed(8), NetAmount: item.NetAmount.StringFixed(8), TaxAmount: item.TaxAmount.StringFixed(8), BillCount: int32(item.BillCount), TaxInvoiceNo: item.TaxInvoiceNo, InvoiceDate: item.InvoiceDate, Note: item.Note, Version: item.Version, IssuedAt: financeTime(item.IssuedAt), CancelledAt: financeTime(item.CancelledAt), CancellationReason: item.CancellationReason, RedInvoiceNo: item.RedInvoiceNo, RedInvoiceDate: item.RedInvoiceDate, RedFlushedAt: financeTime(item.RedFlushedAt), RedFlushReason: item.RedFlushReason, BillLinks: links, InvoiceProfileId: uuidStringPtr(item.InvoiceProfileID), InvoiceTitle: financeOptionalValue(item.InvoiceTitle), TaxpayerIdentificationNo: financeOptionalValue(item.TaxpayerIdentificationNo), RegisteredAddress: financeOptionalValue(item.RegisteredAddress), RegisteredPhone: financeOptionalValue(item.RegisteredPhone), BankName: financeOptionalValue(item.BankName), BankAccount: financeOptionalValue(item.BankAccount), Lines: lines, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}
