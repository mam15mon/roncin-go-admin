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

func (s *SettlementService) ListBills(ctx context.Context, request *v1.ListBillsRequest) (*v1.ListBillsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceBillInvalidArgument)
	if err != nil {
		return nil, err
	}
	filter := biz.FinanceBillFilter{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword),
		Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction))),
		Status:    financeBillStatusFromAPI(request.Status),
		Currency:  strings.ToUpper(financeOptionalString(request.Currency)), BillDateFrom: financeOptionalString(request.BillDateFrom), BillDateTo: financeOptionalString(request.BillDateTo),
	}
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		filter.SettlementPartyID = &id
	}
	tagIDs, err := parseUUIDValues(request.GetTagIds(), biz.ErrFinanceBillInvalidArgument)
	if err != nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	filter.TagIDs = tagIDs
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceBillRead, false, request.OrganizationId)
	if err != nil {
		return nil, err
	}
	result, err := s.billUsecase.List(ctx, organizationIDs, filter)
	if err != nil {
		return nil, err
	}
	billIDs := make([]uuid.UUID, 0, len(result.Items))
	for _, item := range result.Items {
		billIDs = append(billIDs, item.ID)
	}
	billTags, err := s.tagUsecase.LoadFinanceBillTags(ctx, billIDs)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceBill, 0, len(result.Items))
	for _, item := range result.Items {
		converted := financeBillToAPI(item)
		converted.Tags = businessTagSummariesToFinanceAPI(billTags[item.ID])
		data = append(data, converted)
	}
	return okList(ctx, &v1.ListBillsResponse{
		Data: data, Total: result.Total,
		Summary: financeBillSummaryToAPI(result.Summary),
	}), nil
}

func (s *SettlementService) ListBillCreationCandidates(ctx context.Context, request *v1.ListBillCreationCandidatesRequest) (*v1.ListBillCreationCandidatesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	rawOrganizationID := strings.TrimSpace(request.GetOrganizationId())
	if rawOrganizationID == "" {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceBillCreate, true, &rawOrganizationID)
	if err != nil {
		return nil, err
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceBillInvalidArgument)
	if err != nil {
		return nil, err
	}
	result, err := s.billUsecase.ListCreationCandidates(ctx, organizationIDs[0], biz.FinanceBillCreationCandidateFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword), Direction: biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction)))})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FeeLedgerItem, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeBillableFeeToAPI(item))
	}
	return okList(ctx, &v1.ListBillCreationCandidatesResponse{Data: data, Total: result.Total}), nil
}

func financeBillSummaryToAPI(summary biz.FinanceBillSummary) *v1.FinanceBillSummary {
	return &v1.FinanceBillSummary{AmountsByBaseCurrency: financeBaseCurrencyAmountsToAPI(summary.AmountsByBaseCurrency)}
}

func financeBaseCurrencyAmountsToAPI(items []biz.FinanceBaseCurrencyAmount) []*v1.FinanceBaseCurrencyAmount {
	data := make([]*v1.FinanceBaseCurrencyAmount, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.FinanceBaseCurrencyAmount{BaseCurrency: item.BaseCurrency, ReceivableBaseAmount: item.ReceivableBaseAmount.StringFixed(8), PayableBaseAmount: item.PayableBaseAmount.StringFixed(8), UnverifiedBaseAmount: item.UnverifiedBaseAmount.StringFixed(8)})
	}
	return data
}

func (s *SettlementService) GetBill(ctx context.Context, request *v1.GetBillRequest) (*v1.GetBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillRead, false)
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	billTags, err := s.tagUsecase.LoadFinanceBillTags(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	converted := financeBillToAPI(item)
	converted.Tags = businessTagSummariesToFinanceAPI(billTags[id])
	return ok(ctx, &v1.GetBillResponse{Data: converted}), nil
}

func (s *SettlementService) CreateBill(ctx context.Context, request *v1.CreateBillRequest) (*v1.CreateBillResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	feeIDs := make([]uuid.UUID, 0, len(request.GetFeeIds()))
	for _, rawID := range request.GetFeeIds() {
		id, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		feeIDs = append(feeIDs, id)
	}
	organizationIDs, scopeErr := organizationIDsForPermission(principal, access.FinanceBillCreate, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	organizationID, scopeErr := s.billUsecase.ResolveBillableFeeOrganization(ctx, organizationIDs, feeIDs)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.billUsecase.Create(ctx, organizationID, principal.UserID, biz.CreateFinanceBillInput{
		FeeIDs: feeIDs, BillDate: request.GetBillDate(), DueDate: request.DueDate, Note: request.Note, StatementTitle: request.StatementTitle, PaymentTermsDays: financeInt32Pointer(request.PaymentTermsDays), IdempotencyKey: request.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateBillResponse{Data: financeBillToAPI(item)}), nil
}

func (s *SettlementService) PreviewBillBatch(ctx context.Context, request *v1.PreviewBillBatchRequest) (*v1.PreviewBillBatchResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	feeIDs, err := parseTrimmedUUIDValues(request.GetFeeIds(), biz.ErrFinanceBillInvalidArgument)
	requestedOrganizationID, organizationErr := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
	if err != nil || organizationErr != nil || request.GetGroupingPolicy() == nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	requestedOrganization := requestedOrganizationID.String()
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceBillCreate, true, &requestedOrganization)
	if scopeErr != nil {
		return nil, scopeErr
	}
	organizationID, scopeErr := s.billUsecase.ResolveBillableFeeOrganization(ctx, organizationIDs, feeIDs)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if organizationID != requestedOrganizationID {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	preview, err := s.billUsecase.PreviewBatch(ctx, organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: financeBillGroupingPolicyFromAPI(request.GetGroupingPolicy())})
	if err != nil {
		return nil, err
	}
	groups := make([]*v1.BillBatchPreviewGroup, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		fees := make([]*v1.FeeLedgerItem, 0, len(group.Fees))
		for _, item := range group.Fees {
			fees = append(fees, financeBillableFeeToAPI(item))
		}
		groups = append(groups, &v1.BillBatchPreviewGroup{GroupKey: group.GroupKey, Direction: string(group.Direction), SettlementPartyId: group.SettlementPartyID.String(), SettlementPartyName: group.SettlementPartyName, Currency: group.Currency, BaseCurrency: group.BaseCurrency, OrderId: uuidStringPtr(group.OrderID), OrderNo: group.OrderNo, TaxRate: financeDecimalPointer(group.TaxRate, 4), Fees: fees, TotalAmount: group.TotalAmount.StringFixed(8), NetAmount: group.NetAmount.StringFixed(8), TaxAmount: group.TaxAmount.StringFixed(8), BaseCurrencyAmount: group.BaseCurrencyAmount.StringFixed(8)})
	}
	return ok(ctx, &v1.PreviewBillBatchResponse{Data: groups, PreviewToken: preview.PreviewToken}), nil
}

func (s *SettlementService) CreateBillBatch(ctx context.Context, request *v1.CreateBillBatchRequest) (*v1.CreateBillBatchResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	feeIDs, err := parseTrimmedUUIDValues(request.GetFeeIds(), biz.ErrFinanceBillInvalidArgument)
	requestedOrganizationID, organizationErr := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
	if err != nil || organizationErr != nil || request.GetGroupingPolicy() == nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	groups := make([]biz.CreateFinanceBillBatchGroupInput, 0, len(request.GetGroups()))
	for _, group := range request.GetGroups() {
		if group == nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		groups = append(groups, biz.CreateFinanceBillBatchGroupInput{GroupKey: group.GetGroupKey(), StatementTitle: group.GetStatementTitle(), BillDate: group.GetBillDate(), DueDate: group.DueDate, PaymentTermsDays: financeInt32Pointer(group.PaymentTermsDays), Note: group.Note})
	}
	requestedOrganization := requestedOrganizationID.String()
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(principal, access.FinanceBillCreate, true, &requestedOrganization)
	if scopeErr != nil {
		return nil, scopeErr
	}
	organizationID, scopeErr := s.billUsecase.ResolveBillableFeeOrganization(ctx, organizationIDs, feeIDs)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if organizationID != requestedOrganizationID {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	batch, err := s.billUsecase.CreateBatch(ctx, organizationID, principal.UserID, biz.CreateFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: financeBillGroupingPolicyFromAPI(request.GetGroupingPolicy()), Groups: groups, PreviewToken: request.GetPreviewToken(), IdempotencyKey: request.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateBillBatchResponse{Data: financeBillBatchToAPI(batch)}), nil
}

func (s *SettlementService) ConfirmBillBatch(ctx context.Context, request *v1.ConfirmBillBatchRequest) (*v1.ConfirmBillBatchResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillConfirm, true)
	if err != nil {
		return nil, err
	}
	batch, err := s.billUsecase.ConfirmBatch(ctx, organizationIDs, principal.UserID, batchID, expectedVersions)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmBillBatchResponse{Data: financeBillBatchToAPI(batch)}), nil
}

func (s *SettlementService) UpdateBill(ctx context.Context, request *v1.UpdateBillRequest) (*v1.UpdateBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true)
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Update(ctx, organizationIDs, principal.UserID, biz.UpdateFinanceBillInput{
		ID: id, BillDate: request.GetBillDate(), DueDate: request.DueDate, Note: request.Note, StatementTitle: request.StatementTitle, PaymentTermsDays: financeInt32Pointer(request.PaymentTermsDays), ExpectedVersion: request.GetExpectedVersion(),
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateBillResponse{Data: financeBillToAPI(item)}), nil
}

func (s *SettlementService) ConfirmBill(ctx context.Context, request *v1.ConfirmBillRequest) (*v1.ConfirmBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillConfirm, true)
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Confirm(ctx, organizationIDs, principal.UserID, id, request.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmBillResponse{Data: financeBillToAPI(item)}), nil
}

func (s *SettlementService) CancelBill(ctx context.Context, request *v1.CancelBillRequest) (*v1.CancelBillResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true)
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.Cancel(ctx, organizationIDs, principal.UserID, id, request.GetExpectedVersion(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CancelBillResponse{Data: financeBillToAPI(item)}), nil
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
			Quantity: line.Quantity.StringFixed(4), UnitPrice: line.UnitPrice.StringFixed(4),
			TotalAmount: line.TotalAmount.StringFixed(8), NetAmount: line.NetAmount.StringFixed(8), TaxAmount: line.TaxAmount.StringFixed(8), Currency: line.Currency,
			ExchangeRate: line.ExchangeRate.StringFixed(8), BaseCurrency: line.BaseCurrency, BaseCurrencyAmount: line.BaseCurrencyAmount.StringFixed(8), Active: line.Active,
			TaxRate: financeDecimalPointer(line.TaxRate, 4),
		})
	}
	return &v1.FinanceBill{
		Id: item.ID.String(), BillNo: item.BillNo, Direction: string(item.Direction), Status: financeBillStatusToAPI(item.Status),
		OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName,
		SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName,
		Currency: item.Currency, BaseCurrency: item.BaseCurrency, TotalAmount: item.TotalAmount.StringFixed(8), NetAmount: item.NetAmount.StringFixed(8),
		TaxAmount: item.TaxAmount.StringFixed(8), BaseCurrencyAmount: item.BaseCurrencyAmount.StringFixed(8), FeeCount: int32(item.FeeCount),
		BillDate: item.BillDate, DueDate: item.DueDate, Note: item.Note, Version: item.Version,
		ConfirmedAt: financeTime(item.ConfirmedAt), ConfirmedBy: uuidStringPtr(item.ConfirmedBy), CancelledAt: financeTime(item.CancelledAt),
		CancelledBy: uuidStringPtr(item.CancelledBy), CancellationReason: item.CancellationReason, Lines: lines,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		VerifiedAmount: item.VerifiedAmount.StringFixed(8), UnverifiedAmount: item.UnverifiedAmount.StringFixed(8),
		BatchId: uuidStringPtr(item.BatchID), BatchNo: financeOptionalValue(item.BatchNo), StatementTitle: item.StatementTitle, PaymentTermsDays: financeIntPointerToInt32(item.PaymentTermsDays),
		ExchangeRate: item.ExchangeRate.StringFixed(8), ExchangeRateSource: item.ExchangeRateSource, ExchangeRateDate: item.ExchangeRateDate, ExchangeRateSettingId: uuidStringPtr(item.ExchangeRateSettingID),
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
	return &v1.FeeLedgerItem{Id: fee.ID.String(), OrderId: fee.OrderID.String(), OrderNo: item.OrderNo, BusinessType: item.BusinessType, OrganizationId: item.OrganizationID.String(), Direction: string(fee.Direction), Status: orderFeeStatusToAPI(fee.Status), FeeCode: fee.FeeCode, FeeName: fee.FeeName, SettlementPartyId: fee.SettlementPartyID.String(), SettlementPartyName: fee.SettlementPartyName, BillingUnit: fee.BillingUnit, Quantity: fee.Quantity.StringFixed(4), UnitPrice: fee.UnitPrice.StringFixed(4), TotalAmount: fee.TotalAmount.StringFixed(8), NetAmount: fee.NetAmount.StringFixed(8), TaxAmount: fee.TaxAmount.StringFixed(8), TaxRate: financeDecimalPointer(fee.TaxRate, 4), Currency: fee.Currency, ExchangeRate: fee.ExchangeRate.StringFixed(8), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount.StringFixed(8), ExpenseDate: fee.ExpenseDate, Note: fee.Note, Version: fee.Version, CreatedAt: fee.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: fee.UpdatedAt.UTC().Format(time.RFC3339)}
}
