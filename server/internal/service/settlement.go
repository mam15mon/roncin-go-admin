package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"github.com/shopspring/decimal"
)

type SettlementService struct {
	v1.UnimplementedSettlementServiceServer
	usecase             *biz.SettlementUsecase
	billUsecase         *biz.FinanceBillUsecase
	invoiceUsecase      *biz.FinanceInvoiceUsecase
	cashflowUsecase     *biz.FinanceCashflowUsecase
	verificationUsecase *biz.VerificationUsecase
	commissionUsecase   *biz.CommissionUsecase
}

func NewSettlementService(usecase *biz.SettlementUsecase, billUsecase *biz.FinanceBillUsecase, invoiceUsecase *biz.FinanceInvoiceUsecase, cashflowUsecase *biz.FinanceCashflowUsecase, verificationUsecase *biz.VerificationUsecase, commissionUsecase *biz.CommissionUsecase) *SettlementService {
	return &SettlementService{usecase: usecase, billUsecase: billUsecase, invoiceUsecase: invoiceUsecase, cashflowUsecase: cashflowUsecase, verificationUsecase: verificationUsecase, commissionUsecase: commissionUsecase}
}

func (s *SettlementService) ListFeeLedger(ctx context.Context, request *v1.ListFeeLedgerRequest) (*v1.ListFeeLedgerResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	filter := biz.FeeLedgerFilter{Page: int(request.GetPage()), PageSize: int(request.GetPageSize())}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	filter.Keyword = financeOptionalString(request.Keyword)
	filter.BusinessType = financeOptionalString(request.BusinessType)
	filter.Direction = biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction)))
	filter.Status = biz.OrderFeeStatus(strings.ToUpper(financeOptionalString(request.Status)))
	filter.Currency = financeOptionalString(request.Currency)
	filter.ExpenseDateFrom = financeOptionalString(request.ExpenseDateFrom)
	filter.ExpenseDateTo = financeOptionalString(request.ExpenseDateTo)
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceLedgerInvalidArgument
		}
		filter.SettlementPartyID = &id
	}
	result, err := s.usecase.ListFeeLedger(ctx, principal.Organization.ID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FeeLedgerItem, 0, len(result.Items))
	for _, item := range result.Items {
		fee := item.Fee
		data = append(data, &v1.FeeLedgerItem{
			Id: fee.ID.String(), OrderId: fee.OrderID.String(), OrderNo: item.OrderNo, BusinessType: item.Business,
			Direction: string(fee.Direction), Status: string(fee.Status), FeeCode: fee.FeeCode, FeeName: fee.FeeName,
			SettlementPartyId: fee.SettlementPartyID.String(), SettlementPartyName: fee.SettlementPartyName, BillingUnit: fee.BillingUnit,
			Quantity: fee.Quantity.StringFixed(4), UnitPrice: fee.UnitPrice.StringFixed(4), TotalAmount: fee.TotalAmount.StringFixed(8),
			NetAmount: fee.NetAmount.StringFixed(8), TaxAmount: fee.TaxAmount.StringFixed(8), Currency: fee.Currency,
			ExchangeRate: fee.ExchangeRate.StringFixed(8), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount.StringFixed(8),
			ExpenseDate: fee.ExpenseDate, Note: fee.Note, Version: fee.Version,
			CreatedAt: fee.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: fee.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			TaxRate: financeDecimalPointer(fee.TaxRate, 4),
		})
	}
	return &v1.ListFeeLedgerResponse{
		Success: true, Code: 0, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx),
		Summary: &v1.FeeLedgerSummary{ActiveCount: result.Summary.ActiveCount, ReceivableBaseAmount: result.Summary.ReceivableBaseAmount.StringFixed(8), PayableBaseAmount: result.Summary.PayableBaseAmount.StringFixed(8), ProfitBaseAmount: result.Summary.ProfitBaseAmount.StringFixed(8), BaseCurrency: result.Summary.BaseCurrency},
	}, nil
}

func (s *SettlementService) ListBills(ctx context.Context, request *v1.ListBillsRequest) (*v1.ListBillsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	filter := biz.FinanceBillFilter{
		Page: int(request.GetPage()), PageSize: int(request.GetPageSize()), Keyword: financeOptionalString(request.Keyword),
		Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction))),
		Status:    biz.FinanceBillStatus(strings.ToUpper(financeOptionalString(request.Status))),
		Currency:  strings.ToUpper(financeOptionalString(request.Currency)), BillDateFrom: financeOptionalString(request.BillDateFrom), BillDateTo: financeOptionalString(request.BillDateTo),
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		filter.SettlementPartyID = &id
	}
	result, err := s.billUsecase.List(ctx, principal.Organization.ID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceBill, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeBillToAPI(item))
	}
	return &v1.ListBillsResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) GetBill(ctx context.Context, request *v1.GetBillRequest) (*v1.GetBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetBillResponse{Success: true, Code: 0, Message: "OK", Data: financeBillToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) CreateBill(ctx context.Context, request *v1.CreateBillRequest) (*v1.CreateBillResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	feeIDs := make([]uuid.UUID, 0, len(request.GetFeeIds()))
	for _, rawID := range request.GetFeeIds() {
		id, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		feeIDs = append(feeIDs, id)
	}
	item, err := s.billUsecase.Create(ctx, principal.Organization.ID, principal.UserID, biz.CreateFinanceBillInput{
		FeeIDs: feeIDs, BillDate: request.GetBillDate(), DueDate: request.DueDate, Note: request.Note, StatementTitle: request.StatementTitle, PaymentTermsDays: financeInt32Pointer(request.PaymentTermsDays), IdempotencyKey: request.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateBillResponse{Success: true, Code: 0, Message: "OK", Data: financeBillToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) PreviewBillBatch(ctx context.Context, request *v1.PreviewBillBatchRequest) (*v1.PreviewBillBatchResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	feeIDs, err := financeUUIDs(request.GetFeeIds(), biz.ErrFinanceBillInvalidArgument)
	if err != nil || request.GetGroupingPolicy() == nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	preview, err := s.billUsecase.PreviewBatch(ctx, principal.Organization.ID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: financeBillGroupingPolicyFromAPI(request.GetGroupingPolicy())})
	if err != nil {
		return nil, err
	}
	groups := make([]*v1.BillBatchPreviewGroup, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		fees := make([]*v1.FeeLedgerItem, 0, len(group.Fees))
		for _, item := range group.Fees {
			fees = append(fees, financeBillableFeeToAPI(item))
		}
		groups = append(groups, &v1.BillBatchPreviewGroup{GroupKey: group.GroupKey, Direction: string(group.Direction), SettlementPartyId: group.SettlementPartyID.String(), SettlementPartyName: group.SettlementPartyName, Currency: group.Currency, BaseCurrency: group.BaseCurrency, OrderId: financeUUID(group.OrderID), OrderNo: group.OrderNo, TaxRate: financeDecimalPointer(group.TaxRate, 4), Fees: fees, TotalAmount: group.TotalAmount.StringFixed(8), NetAmount: group.NetAmount.StringFixed(8), TaxAmount: group.TaxAmount.StringFixed(8), BaseCurrencyAmount: group.BaseCurrencyAmount.StringFixed(8)})
	}
	return &v1.PreviewBillBatchResponse{Success: true, Message: "OK", Data: groups, PreviewToken: preview.PreviewToken, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) CreateBillBatch(ctx context.Context, request *v1.CreateBillBatchRequest) (*v1.CreateBillBatchResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	feeIDs, err := financeUUIDs(request.GetFeeIds(), biz.ErrFinanceBillInvalidArgument)
	if err != nil || request.GetGroupingPolicy() == nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	groups := make([]biz.CreateFinanceBillBatchGroupInput, 0, len(request.GetGroups()))
	for _, group := range request.GetGroups() {
		if group == nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		groups = append(groups, biz.CreateFinanceBillBatchGroupInput{GroupKey: group.GetGroupKey(), StatementTitle: group.GetStatementTitle(), BillDate: group.GetBillDate(), DueDate: group.DueDate, PaymentTermsDays: financeInt32Pointer(group.PaymentTermsDays), Note: group.Note})
	}
	batch, err := s.billUsecase.CreateBatch(ctx, principal.Organization.ID, principal.UserID, biz.CreateFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: financeBillGroupingPolicyFromAPI(request.GetGroupingPolicy()), Groups: groups, PreviewToken: request.GetPreviewToken(), IdempotencyKey: request.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return &v1.CreateBillBatchResponse{Success: true, Message: "OK", Data: financeBillBatchToAPI(batch), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) ConfirmBillBatch(ctx context.Context, request *v1.ConfirmBillBatchRequest) (*v1.ConfirmBillBatchResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	batchID, err := uuid.Parse(strings.TrimSpace(request.GetId()))
	if err != nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	expectedVersions := make(map[uuid.UUID]uint64, len(request.GetBills()))
	for _, item := range request.GetBills() {
		if item == nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		billID, parseErr := uuid.Parse(strings.TrimSpace(item.GetBillId()))
		if parseErr != nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		if _, exists := expectedVersions[billID]; exists {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		expectedVersions[billID] = item.GetExpectedVersion()
	}
	batch, err := s.billUsecase.ConfirmBatch(ctx, principal.Organization.ID, principal.UserID, batchID, expectedVersions)
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmBillBatchResponse{Success: true, Message: "OK", Data: financeBillBatchToAPI(batch), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) UpdateBill(ctx context.Context, request *v1.UpdateBillRequest) (*v1.UpdateBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Update(ctx, principal.Organization.ID, principal.UserID, biz.UpdateFinanceBillInput{
		ID: id, BillDate: request.GetBillDate(), DueDate: request.DueDate, Note: request.Note, StatementTitle: request.StatementTitle, PaymentTermsDays: financeInt32Pointer(request.PaymentTermsDays), ExpectedVersion: request.GetExpectedVersion(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateBillResponse{Success: true, Code: 0, Message: "OK", Data: financeBillToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) ConfirmBill(ctx context.Context, request *v1.ConfirmBillRequest) (*v1.ConfirmBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Confirm(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.ConfirmBillResponse{Success: true, Code: 0, Message: "OK", Data: financeBillToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) CancelBill(ctx context.Context, request *v1.CancelBillRequest) (*v1.CancelBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Cancel(ctx, principal.Organization.ID, principal.UserID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.CancelBillResponse{Success: true, Code: 0, Message: "OK", Data: financeBillToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func financePrincipalAndID(ctx context.Context, rawID string) (*biz.Principal, uuid.UUID, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, uuid.Nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return nil, uuid.Nil, biz.ErrFinanceBillInvalidArgument
	}
	return principal, id, nil
}

func financeBillToAPI(item *biz.FinanceBill) *v1.FinanceBill {
	if item == nil {
		return nil
	}
	lines := make([]*v1.FinanceBillLine, 0, len(item.Lines))
	for _, line := range item.Lines {
		lines = append(lines, &v1.FinanceBillLine{
			Id: line.ID.String(), OrderFeeId: line.OrderFeeID.String(), OrderId: line.OrderID.String(), OrderNo: line.OrderNo,
			BusinessType: line.BusinessType, FeeCode: line.FeeCode, FeeName: line.FeeName,
			TotalAmount: line.TotalAmount.StringFixed(8), NetAmount: line.NetAmount.StringFixed(8), TaxAmount: line.TaxAmount.StringFixed(8), Currency: line.Currency,
			ExchangeRate: line.ExchangeRate.StringFixed(8), BaseCurrency: line.BaseCurrency, BaseCurrencyAmount: line.BaseCurrencyAmount.StringFixed(8), Active: line.Active,
			TaxRate: financeDecimalPointer(line.TaxRate, 4),
		})
	}
	return &v1.FinanceBill{
		Id: item.ID.String(), BillNo: item.BillNo, Direction: string(item.Direction), Status: string(item.Status),
		SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName,
		Currency: item.Currency, BaseCurrency: item.BaseCurrency, TotalAmount: item.TotalAmount.StringFixed(8), NetAmount: item.NetAmount.StringFixed(8),
		TaxAmount: item.TaxAmount.StringFixed(8), BaseCurrencyAmount: item.BaseCurrencyAmount.StringFixed(8), FeeCount: int32(item.FeeCount),
		BillDate: item.BillDate, DueDate: item.DueDate, Note: item.Note, Version: item.Version,
		ConfirmedAt: financeTime(item.ConfirmedAt), ConfirmedBy: financeUUID(item.ConfirmedBy), CancelledAt: financeTime(item.CancelledAt),
		CancelledBy: financeUUID(item.CancelledBy), CancellationReason: item.CancellationReason, Lines: lines,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		VerifiedAmount: item.VerifiedAmount.StringFixed(8), UnverifiedAmount: item.UnverifiedAmount.StringFixed(8),
		BatchId: financeUUID(item.BatchID), BatchNo: financeOptionalValue(item.BatchNo), StatementTitle: item.StatementTitle, PaymentTermsDays: financeIntPointerToInt32(item.PaymentTermsDays),
	}
}

func financeBillBatchToAPI(item *biz.FinanceBillBatch) *v1.FinanceBillBatch {
	if item == nil {
		return nil
	}
	bills := make([]*v1.FinanceBill, 0, len(item.Bills))
	for _, bill := range item.Bills {
		bills = append(bills, financeBillToAPI(bill))
	}
	return &v1.FinanceBillBatch{Id: item.ID.String(), BatchNo: item.BatchNo, SplitByOrder: item.GroupingPolicy.SplitByOrder, SplitByTaxRate: item.GroupingPolicy.SplitByTaxRate, FeeCount: int32(item.FeeCount), BillCount: int32(item.BillCount), TotalBaseAmount: item.TotalBaseAmount.StringFixed(8), BaseCurrency: item.BaseCurrency, Bills: bills, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)}
}

func financeBillGroupingPolicyFromAPI(value *v1.BillGroupingPolicy) biz.FinanceBillGroupingPolicy {
	return biz.FinanceBillGroupingPolicy{SplitByOrder: value.GetSplitByOrder(), SplitByTaxRate: value.GetSplitByTaxRate()}
}

func financeBillableFeeToAPI(item *biz.FinanceBillableFee) *v1.FeeLedgerItem {
	fee := item.Fee
	return &v1.FeeLedgerItem{Id: fee.ID.String(), OrderId: fee.OrderID.String(), OrderNo: item.OrderNo, BusinessType: item.BusinessType, Direction: string(fee.Direction), Status: string(fee.Status), FeeCode: fee.FeeCode, FeeName: fee.FeeName, SettlementPartyId: fee.SettlementPartyID.String(), SettlementPartyName: fee.SettlementPartyName, BillingUnit: fee.BillingUnit, Quantity: fee.Quantity.StringFixed(4), UnitPrice: fee.UnitPrice.StringFixed(4), TotalAmount: fee.TotalAmount.StringFixed(8), NetAmount: fee.NetAmount.StringFixed(8), TaxAmount: fee.TaxAmount.StringFixed(8), TaxRate: financeDecimalPointer(fee.TaxRate, 4), Currency: fee.Currency, ExchangeRate: fee.ExchangeRate.StringFixed(8), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount.StringFixed(8), ExpenseDate: fee.ExpenseDate, Note: fee.Note, Version: fee.Version, CreatedAt: fee.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: fee.UpdatedAt.UTC().Format(time.RFC3339)}
}

func financeUUIDs(values []string, invalid error) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err != nil {
			return nil, invalid
		}
		result = append(result, id)
	}
	return result, nil
}

func financeInt32Pointer(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func financeIntPointerToInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func financeDecimalPointer(value *decimal.Decimal, scale int32) *string {
	if value == nil {
		return nil
	}
	formatted := value.StringFixed(scale)
	return &formatted
}

func financeOptionalValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func financeTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func financeUUID(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	formatted := value.String()
	return &formatted
}

func (s *SettlementService) ListInvoices(ctx context.Context, request *v1.ListInvoicesRequest) (*v1.ListInvoicesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	filter := biz.FinanceInvoiceFilter{Page: int(request.GetPage()), PageSize: int(request.GetPageSize()), Keyword: financeOptionalString(request.Keyword), Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction))), Status: biz.FinanceInvoiceStatus(strings.ToUpper(financeOptionalString(request.Status)))}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	result, err := s.invoiceUsecase.List(ctx, principal.Organization.ID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceInvoice, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeInvoiceToAPI(item))
	}
	return &v1.ListInvoicesResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
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
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
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
	return &v1.FinanceInvoice{Id: item.ID.String(), RecordNo: item.RecordNo, Direction: string(item.Direction), Status: string(item.Status), InvoiceType: string(item.InvoiceType), SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName, Currency: item.Currency, TotalAmount: item.TotalAmount.StringFixed(8), NetAmount: item.NetAmount.StringFixed(8), TaxAmount: item.TaxAmount.StringFixed(8), BillCount: int32(item.BillCount), TaxInvoiceNo: item.TaxInvoiceNo, InvoiceDate: item.InvoiceDate, Note: item.Note, Version: item.Version, IssuedAt: financeTime(item.IssuedAt), CancelledAt: financeTime(item.CancelledAt), CancellationReason: item.CancellationReason, RedInvoiceNo: item.RedInvoiceNo, RedInvoiceDate: item.RedInvoiceDate, RedFlushedAt: financeTime(item.RedFlushedAt), RedFlushReason: item.RedFlushReason, BillLinks: links, InvoiceProfileId: financeUUID(item.InvoiceProfileID), InvoiceTitle: financeOptionalValue(item.InvoiceTitle), TaxpayerIdentificationNo: financeOptionalValue(item.TaxpayerIdentificationNo), RegisteredAddress: financeOptionalValue(item.RegisteredAddress), RegisteredPhone: financeOptionalValue(item.RegisteredPhone), BankName: financeOptionalValue(item.BankName), BankAccount: financeOptionalValue(item.BankAccount), Lines: lines, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}

func (s *SettlementService) ListCashflows(ctx context.Context, r *v1.ListCashflowsRequest) (*v1.ListCashflowsResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	f := biz.FinanceCashflowFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(r.Direction))), Status: biz.FinanceCashflowStatus(strings.ToUpper(financeOptionalString(r.Status)))}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, e := s.cashflowUsecase.List(ctx, p.Organization.ID, f)
	if e != nil {
		return nil, e
	}
	data := make([]*v1.FinanceCashflow, 0, len(result.Items))
	for _, x := range result.Items {
		data = append(data, cashflowToAPI(x))
	}
	return &v1.ListCashflowsResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateCashflow(ctx context.Context, r *v1.CreateCashflowRequest) (*v1.CreateCashflowResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	party, e := uuid.Parse(strings.TrimSpace(r.GetSettlementPartyId()))
	if e != nil {
		return nil, biz.ErrFinanceCashflowInvalidArgument
	}
	amount, e := decimal.NewFromString(r.GetAmount())
	if e != nil {
		return nil, biz.ErrFinanceCashflowInvalidArgument
	}
	rate, e := decimal.NewFromString(r.GetExchangeRate())
	if e != nil {
		return nil, biz.ErrFinanceCashflowInvalidArgument
	}
	x, e := s.cashflowUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateFinanceCashflowInput{Direction: biz.OrderFeeDirection(strings.ToUpper(r.GetDirection())), SettlementPartyID: party, Currency: r.GetCurrency(), Amount: amount, ExchangeRate: rate, BaseCurrency: r.GetBaseCurrency(), TransactionDate: r.GetTransactionDate(), OurAccount: r.GetOurAccount(), CounterpartyAccount: r.CounterpartyAccount, PaymentMethod: r.GetPaymentMethod(), BankReferenceNo: r.BankReferenceNo, Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if e != nil {
		return nil, e
	}
	return &v1.CreateCashflowResponse{Success: true, Message: "OK", Data: cashflowToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ConfirmCashflow(ctx context.Context, r *v1.ConfirmCashflowRequest) (*v1.ConfirmCashflowResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	x, e := s.cashflowUsecase.Confirm(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if e != nil {
		return nil, e
	}
	return &v1.ConfirmCashflowResponse{Success: true, Message: "OK", Data: cashflowToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CancelCashflow(ctx context.Context, r *v1.CancelCashflowRequest) (*v1.CancelCashflowResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	x, e := s.cashflowUsecase.Cancel(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if e != nil {
		return nil, e
	}
	return &v1.CancelCashflowResponse{Success: true, Message: "OK", Data: cashflowToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func cashflowToAPI(x *biz.FinanceCashflow) *v1.FinanceCashflow {
	if x == nil {
		return nil
	}
	return &v1.FinanceCashflow{Id: x.ID.String(), FlowNo: x.FlowNo, Direction: string(x.Direction), Status: string(x.Status), SettlementPartyId: x.SettlementPartyID.String(), SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: x.Amount.StringFixed(8), ExchangeRate: x.ExchangeRate.StringFixed(8), BaseCurrency: x.BaseCurrency, BaseAmount: x.BaseAmount.StringFixed(8), TransactionDate: x.TransactionDate, OurAccount: x.OurAccount, CounterpartyAccount: x.CounterpartyAccount, PaymentMethod: x.PaymentMethod, BankReferenceNo: x.BankReferenceNo, Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), CancelledAt: financeTime(x.CancelledAt), CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339), VerifiedAmount: x.VerifiedAmount.StringFixed(8), UnverifiedAmount: x.UnverifiedAmount.StringFixed(8)}
}
func (s *SettlementService) ListVerifications(ctx context.Context, r *v1.ListVerificationsRequest) (*v1.ListVerificationsResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	f := biz.VerificationFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), Status: biz.VerificationStatus(strings.ToUpper(financeOptionalString(r.Status)))}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, e := s.verificationUsecase.List(ctx, p.Organization.ID, f)
	if e != nil {
		return nil, e
	}
	data := make([]*v1.FinanceVerification, 0, len(result.Items))
	for _, x := range result.Items {
		data = append(data, verificationToAPI(x))
	}
	return &v1.ListVerificationsResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateVerification(ctx context.Context, r *v1.CreateVerificationRequest) (*v1.CreateVerificationResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	as := make([]*biz.VerificationAllocation, 0, len(r.GetAllocations()))
	for _, x := range r.GetAllocations() {
		c, e := uuid.Parse(x.GetCashflowId())
		if e != nil {
			return nil, biz.ErrVerificationInvalid
		}
		b, e := uuid.Parse(x.GetBillId())
		if e != nil {
			return nil, biz.ErrVerificationInvalid
		}
		z, e := decimal.NewFromString(x.GetAmount())
		if e != nil {
			return nil, biz.ErrVerificationInvalid
		}
		as = append(as, &biz.VerificationAllocation{CashflowID: c, BillID: b, Amount: z})
	}
	x, e := s.verificationUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateVerificationInput{Allocations: as, VerificationDate: r.GetVerificationDate(), Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if e != nil {
		return nil, e
	}
	return &v1.CreateVerificationResponse{Success: true, Message: "OK", Data: verificationToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ReverseVerification(ctx context.Context, r *v1.ReverseVerificationRequest) (*v1.ReverseVerificationResponse, error) {
	p, id, e := financePrincipalAndID(ctx, r.GetId())
	if e != nil {
		return nil, e
	}
	x, e := s.verificationUsecase.Reverse(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if e != nil {
		return nil, e
	}
	return &v1.ReverseVerificationResponse{Success: true, Message: "OK", Data: verificationToAPI(x), TraceId: requestmeta.TraceID(ctx)}, nil
}
func verificationToAPI(x *biz.FinanceVerification) *v1.FinanceVerification {
	if x == nil {
		return nil
	}
	as := make([]*v1.FinanceVerificationAllocation, 0, len(x.Allocations))
	for _, a := range x.Allocations {
		as = append(as, &v1.FinanceVerificationAllocation{Id: a.ID.String(), CashflowId: a.CashflowID.String(), BillId: a.BillID.String(), CashflowNo: a.CashflowNo, BillNo: a.BillNo, Amount: a.Amount.StringFixed(8), Active: a.Active})
	}
	return &v1.FinanceVerification{Id: x.ID.String(), VerificationNo: x.VerificationNo, Status: string(x.Status), Direction: string(x.Direction), SettlementPartyId: x.SettlementPartyID.String(), SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: x.Amount.StringFixed(8), VerificationDate: x.VerificationDate, Note: x.Note, Version: x.Version, ReversedAt: financeTime(x.ReversedAt), ReversalReason: x.ReversalReason, Allocations: as, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339)}
}

func (s *SettlementService) ListCommissions(ctx context.Context, r *v1.ListCommissionsRequest) (*v1.ListCommissionsResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	f := biz.CommissionFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), Status: biz.CommissionStatus(strings.ToUpper(financeOptionalString(r.Status)))}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, err := s.commissionUsecase.List(ctx, p.Organization.ID, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommission, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionToAPI(item))
	}
	return &v1.ListCommissionsResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ListCommissionEmployees(ctx context.Context, _ *v1.ListCommissionEmployeesRequest) (*v1.ListCommissionEmployeesResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.commissionUsecase.ListEmployees(ctx, p.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionEmployeeOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.CommissionEmployeeOption{Id: item.ID.String(), DisplayName: item.DisplayName})
	}
	return &v1.ListCommissionEmployeesResponse{Success: true, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ListCommissionCandidates(ctx context.Context, r *v1.ListCommissionCandidatesRequest) (*v1.ListCommissionEmployeesResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	verificationID, err := uuid.Parse(strings.TrimSpace(r.GetVerificationId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	ruleID, err := uuid.Parse(strings.TrimSpace(r.GetRuleId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	items, err := s.commissionUsecase.ListCandidates(ctx, p.Organization.ID, verificationID, ruleID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionEmployeeOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.CommissionEmployeeOption{Id: item.ID.String(), DisplayName: item.DisplayName})
	}
	return &v1.ListCommissionEmployeesResponse{Success: true, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ListCommissionRules(ctx context.Context, r *v1.ListCommissionRulesRequest) (*v1.ListCommissionRulesResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	f := biz.CommissionRuleFilter{Page: int(r.GetPage()), PageSize: int(r.GetPageSize()), Keyword: financeOptionalString(r.Keyword), PersonnelRole: biz.CommissionPersonnelRole(strings.ToUpper(financeOptionalString(r.PersonnelRole))), Enabled: r.Enabled}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	result, err := s.commissionUsecase.ListRules(ctx, p.Organization.ID, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommissionRule, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionRuleToAPI(item))
	}
	return &v1.ListCommissionRulesResponse{Success: true, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CreateCommissionRule(ctx context.Context, r *v1.CreateCommissionRuleRequest) (*v1.CommissionRuleResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	in, err := commissionRuleInputFromAPI(r.GetRule())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.CreateRule(ctx, p.Organization.ID, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return &v1.CommissionRuleResponse{Success: true, Message: "OK", Data: commissionRuleToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) UpdateCommissionRule(ctx context.Context, r *v1.UpdateCommissionRuleRequest) (*v1.CommissionRuleResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	in, err := commissionRuleInputFromAPI(r.GetRule())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.UpdateRule(ctx, p.Organization.ID, p.UserID, biz.UpdateCommissionRuleInput{ID: id, CreateCommissionRuleInput: in, ExpectedVersion: r.GetExpectedVersion()})
	if err != nil {
		return nil, err
	}
	return &v1.CommissionRuleResponse{Success: true, Message: "OK", Data: commissionRuleToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func commissionRuleInputFromAPI(r *v1.CommissionRuleInput) (biz.CreateCommissionRuleInput, error) {
	if r == nil {
		return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
	}
	rate, err := decimal.NewFromString(r.GetRatePercent())
	if err != nil {
		return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
	}
	return biz.CreateCommissionRuleInput{Name: r.GetName(), PersonnelRole: biz.CommissionPersonnelRole(strings.ToUpper(r.GetPersonnelRole())), CalculationBasis: biz.CommissionCalculationBasis(strings.ToUpper(r.GetCalculationBasis())), RatePercent: rate, EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo, Enabled: r.GetEnabled(), Note: r.Note}, nil
}
func commissionRuleToAPI(x *biz.FinanceCommissionRule) *v1.FinanceCommissionRule {
	if x == nil {
		return nil
	}
	return &v1.FinanceCommissionRule{Id: x.ID.String(), Name: x.Name, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), RatePercent: x.RatePercent.StringFixed(4), EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339)}
}
func (s *SettlementService) CreateCommission(ctx context.Context, r *v1.CreateCommissionRequest) (*v1.CreateCommissionResponse, error) {
	p, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	verificationID, err := uuid.Parse(strings.TrimSpace(r.GetVerificationId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	employeeID, err := uuid.Parse(strings.TrimSpace(r.GetEmployeeId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	ruleID, err := uuid.Parse(strings.TrimSpace(r.GetRuleId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	item, err := s.commissionUsecase.Create(ctx, p.Organization.ID, p.UserID, biz.CreateCommissionInput{VerificationID: verificationID, EmployeeID: employeeID, RuleID: ruleID, Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return &v1.CreateCommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) ConfirmCommission(ctx context.Context, r *v1.CommissionTransitionRequest) (*v1.CommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Confirm(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.CommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) MarkCommissionPaid(ctx context.Context, r *v1.CommissionTransitionRequest) (*v1.CommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.MarkPaid(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return &v1.CommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *SettlementService) CancelCommission(ctx context.Context, r *v1.CancelCommissionRequest) (*v1.CommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Cancel(ctx, p.Organization.ID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if err != nil {
		return nil, err
	}
	return &v1.CommissionResponse{Success: true, Message: "OK", Data: commissionToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func commissionToAPI(x *biz.FinanceCommission) *v1.FinanceCommission {
	if x == nil {
		return nil
	}
	var ruleID, ruleName, personnelRole, calculationBasis *string
	if x.RuleID != uuid.Nil {
		value := x.RuleID.String()
		ruleID = &value
	}
	if x.RuleName != "" {
		value := x.RuleName
		ruleName = &value
	}
	if x.PersonnelRole != "" {
		value := string(x.PersonnelRole)
		personnelRole = &value
	}
	if x.CalculationBasis != "" {
		value := string(x.CalculationBasis)
		calculationBasis = &value
	}
	return &v1.FinanceCommission{Id: x.ID.String(), CommissionNo: x.CommissionNo, VerificationId: x.VerificationID.String(), VerificationNo: x.VerificationNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, Status: string(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), PaidAt: financeTime(x.PaidAt), CancelledAt: financeTime(x.CancelledAt), CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339), RuleId: ruleID, RuleName: ruleName, PersonnelRole: personnelRole, CalculationBasis: calculationBasis}
}

func financeOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ v1.SettlementServiceServer = (*SettlementService)(nil)
