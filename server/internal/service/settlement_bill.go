package service

import (
	"context"
	"strings"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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
		Direction:     biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction))),
		Status:        financeBillStatusFromAPI(request.Status),
		Currency:      strings.ToUpper(financeOptionalString(request.Currency)),
		BillDateFrom:  financeOptionalString(request.BillDateFrom),
		BillDateTo:    financeOptionalString(request.BillDateTo),
		DueDateFrom:   financeOptionalString(request.DueDateFrom),
		DueDateTo:     financeOptionalString(request.DueDateTo),
		OnlyUnsettled: request.GetOnlyUnsettled(),
		OnlyOverdue:   request.GetOnlyOverdue(),
		Completed:     request.Completed,
	}
	if request.SettlementProgress != nil {
		filter.SettlementProgress = biz.FinanceBillSettlementProgress(strings.TrimPrefix(request.GetSettlementProgress().String(), "FINANCE_BILL_SETTLEMENT_PROGRESS_"))
	}
	if request.InvoiceProgress != nil {
		filter.InvoiceProgress = biz.FinanceBillInvoiceProgress(strings.TrimPrefix(request.GetInvoiceProgress().String(), "FINANCE_BILL_INVOICE_PROGRESS_"))
	}
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		filter.SettlementPartyID = &id
	}
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceBillRead, false, request.OrganizationId)
	if err != nil {
		return nil, err
	}
	result, err := s.billUsecase.List(ctx, organizationIDs, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceBill, 0, len(result.Items))
	for _, item := range result.Items {
		converted := financeBillToAPI(item)
		data = append(data, converted)
	}
	return okList(ctx, &v1.ListBillsResponse{
		MatchedSummary: financeLedgerMatchedSummaryToAPI(result.MatchedSummary),
		Data:           data, Total: result.Total,
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

func (s *SettlementService) ListBillSettlementAccountCandidates(ctx context.Context, request *v1.ListBillSettlementAccountCandidatesRequest) (*v1.ListBillSettlementAccountCandidatesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	organizationID, partyID, direction, currency, err := billSettlementAccountCandidateInput(request.GetOrganizationId(), request.GetSettlementPartyId(), request.GetDirection(), request.GetCurrency())
	if err != nil {
		return nil, err
	}
	rawOrganizationID := organizationID.String()
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceBillCreate, true, &rawOrganizationID)
	if err != nil || len(organizationIDs) != 1 {
		if err != nil {
			return nil, err
		}
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	items, err := s.accountUsecase.List(ctx, organizationID, partyID, biz.PartnerAccountFilter{Enabled: boolPointer(true), Currency: currency})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListBillSettlementAccountCandidatesResponse{Data: financeSettlementAccountOptions(items, direction)}), nil
}

func (s *SettlementService) ListBillSettlementAccountUpdateCandidates(ctx context.Context, request *v1.ListBillSettlementAccountUpdateCandidatesRequest) (*v1.ListBillSettlementAccountUpdateCandidatesResponse, error) {
	principal, billID, err := financePrincipalAndID(ctx, request.GetBillId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true)
	if err != nil {
		return nil, err
	}
	bill, err := s.billUsecase.Get(ctx, organizationIDs, billID)
	if err != nil {
		return nil, err
	}
	if bill.Status != biz.FinanceBillConfirmed {
		return nil, biz.ErrFinanceBillInvalidTransition
	}
	// 普通账单的费用币种是不可变边界；改账户只能查询账单自身币种的候选。
	items, err := s.accountUsecase.List(ctx, bill.OrganizationID, bill.SettlementPartyID, biz.PartnerAccountFilter{Enabled: boolPointer(true), Currency: bill.Currency})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListBillSettlementAccountUpdateCandidatesResponse{Data: financeSettlementAccountOptions(items, bill.Direction)}), nil
}

func billSettlementAccountCandidateInput(rawOrganizationID, rawPartyID, rawDirection, rawCurrency string) (uuid.UUID, uuid.UUID, biz.OrderFeeDirection, string, error) {
	organizationID, orgErr := uuid.Parse(strings.TrimSpace(rawOrganizationID))
	partyID, partyErr := uuid.Parse(strings.TrimSpace(rawPartyID))
	direction := biz.OrderFeeDirection(strings.ToUpper(strings.TrimSpace(rawDirection)))
	currency := strings.ToUpper(strings.TrimSpace(rawCurrency))
	if orgErr != nil || partyErr != nil || (direction != biz.OrderFeeReceivable && direction != biz.OrderFeePayable) || len(currency) != 3 {
		return uuid.Nil, uuid.Nil, "", "", biz.ErrFinanceBillInvalidArgument
	}
	return organizationID, partyID, direction, currency, nil
}

func financeSettlementAccountOptions(items []*biz.PartnerAccount, direction biz.OrderFeeDirection) []*v1.FinanceSettlementAccountOption {
	result := make([]*v1.FinanceSettlementAccountOption, 0, len(items))
	for _, item := range items {
		if item.Usage.Supports(direction) {
			result = append(result, &v1.FinanceSettlementAccountOption{Id: item.ID.String(), Name: item.Name, AccountHolder: item.AccountHolder, BankName: item.BankName, AccountNo: item.AccountNo, Currency: item.Currency, SwiftCode: item.SwiftCode, IsDefault: (direction == biz.OrderFeeReceivable && item.IsDefaultReceivable) || (direction == biz.OrderFeePayable && item.IsDefaultPayable)})
		}
	}
	return result
}

func boolPointer(value bool) *bool { return &value }

func financeBillSummaryToAPI(summary biz.FinanceBillSummary) *v1.FinanceBillSummary {
	return &v1.FinanceBillSummary{AmountsByBaseCurrency: financeBaseCurrencyAmountsToAPI(summary.AmountsByBaseCurrency)}
}

func financeBaseCurrencyAmountsToAPI(items []biz.FinanceBaseCurrencyAmount) []*v1.FinanceBaseCurrencyAmount {
	data := make([]*v1.FinanceBaseCurrencyAmount, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.FinanceBaseCurrencyAmount{
			BaseCurrency:                item.BaseCurrency,
			ReceivableBaseAmount:        item.ReceivableBaseAmount.StringFixed(8),
			PayableBaseAmount:           item.PayableBaseAmount.StringFixed(8),
			UnverifiedBaseAmount:        item.UnverifiedBaseAmount.StringFixed(8),
			OverdueReceivableBaseAmount: item.OverdueReceivableBaseAmount.StringFixed(8),
		})
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
	converted := financeBillToAPI(item)
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
	accountID, err := uuid.Parse(strings.TrimSpace(request.GetSettlementAccountId()))
	if err != nil {
		return nil, biz.ErrFinanceBillSettlementAccountInvalid
	}
	input := biz.CreateFinanceBillInput{FeeIDs: feeIDs, BillDate: request.GetBillDate(), DueDate: request.DueDate, Note: request.Note, StatementTitle: request.StatementTitle, PaymentTermsDays: financeInt32Pointer(request.PaymentTermsDays), IdempotencyKey: request.GetIdempotencyKey(), SettlementAccountID: accountID}
	item, err := s.billUsecase.Create(ctx, organizationID, principal.UserID, input)
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
	previewConfigs, configErr := previewBillBatchConfigsFromAPI(request)
	if configErr != nil {
		return nil, configErr
	}
	preview, err := s.billUsecase.PreviewBatch(ctx, organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: financeBillGroupingPolicyFromAPI(request.GetGroupingPolicy()), GroupConfigs: previewConfigs.groups})
	if err != nil {
		return nil, err
	}
	groups := make([]*v1.BillBatchPreviewGroup, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		fees := make([]*v1.FeeLedgerItem, 0, len(group.Fees))
		for _, item := range group.Fees {
			fees = append(fees, financeBillableFeeToAPI(item))
		}
		var defaultPaymentTermsDays *int32
		if group.DefaultPaymentTermsDays != nil {
			v := int32(*group.DefaultPaymentTermsDays)
			defaultPaymentTermsDays = &v
		}
		groups = append(groups, &v1.BillBatchPreviewGroup{GroupKey: group.GroupKey, Direction: string(group.Direction), SettlementPartyId: group.SettlementPartyID.String(), SettlementPartyName: group.SettlementPartyName, Currency: group.Currency, BaseCurrency: group.BaseCurrency, OrderId: uuidStringPtr(group.OrderID), OrderNo: group.OrderNo, TaxRate: financeDecimalPointer(group.TaxRate, 4), Fees: fees, TotalAmount: group.TotalAmount.StringFixed(8), NetAmount: group.NetAmount.StringFixed(8), TaxAmount: group.TaxAmount.StringFixed(8), BaseCurrencyAmount: group.BaseCurrencyAmount.StringFixed(8), IsTemporaryBillDate: group.TemporaryBillDate, ConfigurationComplete: group.ConfigurationComplete, EstimatedInvoiceCurrency: group.EstimatedInvoiceCurrency, EstimatedInvoiceRate: group.EstimatedInvoiceRate.StringFixed(8), EstimatedInvoiceAmount: group.EstimatedInvoiceAmount.StringFixed(8), IsCasual: group.IsCasual, DefaultPaymentTermsDays: defaultPaymentTermsDays, CreditLimitAmount: financeDecimalPointer(group.CreditLimitAmount, 2), CreditCurrency: group.CreditCurrency, CurrentUnsettledAmount: financeDecimalPointer(group.CurrentUnsettledAmount, 2), IsCreditExceeded: group.IsCreditExceeded, GroupToken: group.GroupToken})
	}
	return ok(ctx, &v1.PreviewBillBatchResponse{Data: groups, NettingPairs: billBatchNettingPairsToAPI(preview.NettingPairs)}), nil
}

func (s *SettlementService) CreateBillBatch(ctx context.Context, request *v1.CreateBillBatchRequest) (*v1.CreateBillBatchResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	requestedOrganizationID, organizationErr := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
	if organizationErr != nil || request.GetGroupingPolicy() == nil || len(request.GetGroups()) == 0 {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	groups := make([]biz.CreateFinanceBillBatchGroupInput, 0, len(request.GetGroups()))
	feeIDs := make([]uuid.UUID, 0)
	for _, group := range request.GetGroups() {
		if group == nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		groupFeeIDs, feeErr := parseTrimmedUUIDValues(group.GetFeeIds(), biz.ErrFinanceBillInvalidArgument)
		if feeErr != nil || len(groupFeeIDs) == 0 {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		feeIDs = append(feeIDs, groupFeeIDs...)
		accountID, accountErr := uuid.Parse(strings.TrimSpace(group.GetSettlementAccountId()))
		if accountErr != nil {
			return nil, biz.ErrFinanceBillSettlementAccountInvalid
		}
		var estimatedCurrency *string
		if value := strings.ToUpper(strings.TrimSpace(group.GetEstimatedInvoiceCurrency())); value != "" {
			estimatedCurrency = &value
		}
		var estimatedRate *decimal.Decimal
		if value := strings.TrimSpace(group.GetEstimatedInvoiceRate()); value != "" {
			parsed, parseErr := decimal.NewFromString(value)
			if parseErr != nil || !parsed.IsPositive() {
				return nil, biz.ErrFinanceBillInvalidArgument
			}
			estimatedRate = &parsed
		}
		if estimatedRate != nil && estimatedCurrency == nil {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		groups = append(groups, biz.CreateFinanceBillBatchGroupInput{GroupKey: group.GetGroupKey(), StatementTitle: group.GetStatementTitle(), BillDate: group.GetBillDate(), DueDate: group.DueDate, PaymentTermsDays: financeInt32Pointer(group.PaymentTermsDays), Note: group.Note, SettlementAccountID: accountID, EstimatedInvoiceCurrency: estimatedCurrency, EstimatedInvoiceRate: estimatedRate, GroupToken: group.GetGroupToken(), FeeIDs: groupFeeIDs})
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
	created, err := s.billUsecase.CreateBatch(ctx, organizationID, principal.UserID, biz.CreateFinanceBillBatchInput{GroupingPolicy: financeBillGroupingPolicyFromAPI(request.GetGroupingPolicy()), Groups: groups, IdempotencyKey: request.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	// HTTP 200 只表示请求已逐组处理完毕；各组成败由 results 表达，失败组不影响已成功组。
	response := &v1.CreateBillBatchResponse{Data: financeBillBatchToAPI(created.Batch), Results: make([]*v1.BillBatchGroupResult, 0, len(created.Groups))}
	for _, group := range created.Groups {
		result := &v1.BillBatchGroupResult{GroupKey: group.GroupKey}
		if group.Err == nil && group.Bill != nil {
			result.Success = true
			result.Bill = financeBillToAPI(group.Bill)
			response.SucceededCount++
		} else {
			result.Error = billBatchGroupErrorToAPI(group.Err)
			response.FailedCount++
		}
		response.Results = append(response.Results, result)
	}
	return ok(ctx, response), nil
}

// billBatchGroupPublicMetadataKeys 与 HTTP 错误响应的公开键口径一致，仅放行缺汇率提示。
var billBatchGroupPublicMetadataKeys = []string{"currency", "week_start", "direction"}

// billBatchGroupErrorToAPI 把单组失败转换为响应体错误；未知内部错误不透出原始文本。
func billBatchGroupErrorToAPI(err error) *v1.BillBatchGroupError {
	if err == nil {
		err = biz.ErrFinanceBillNotFound
	}
	serviceError := kratoserrors.FromError(err)
	if serviceError.Reason == kratoserrors.UnknownReason || serviceError.Code >= 500 {
		return &v1.BillBatchGroupError{Reason: "FINANCE_BILL_BATCH_GROUP_FAILED", Message: "该组账单生成失败，请稍后重试"}
	}
	result := &v1.BillBatchGroupError{Reason: serviceError.Reason, Message: serviceError.Message}
	for _, key := range billBatchGroupPublicMetadataKeys {
		if value, exists := serviceError.Metadata[key]; exists {
			if result.Metadata == nil {
				result.Metadata = make(map[string]string, len(billBatchGroupPublicMetadataKeys))
			}
			result.Metadata[key] = value
		}
	}
	return result
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
	accountID, err := uuid.Parse(strings.TrimSpace(request.GetSettlementAccountId()))
	if err != nil {
		return nil, biz.ErrFinanceBillSettlementAccountInvalid
	}
	var estimatedCurrency *string
	if value := strings.ToUpper(strings.TrimSpace(request.GetEstimatedInvoiceCurrency())); value != "" {
		estimatedCurrency = &value
	}
	var estimatedRate *decimal.Decimal
	if value := strings.TrimSpace(request.GetEstimatedInvoiceRate()); value != "" {
		parsed, parseErr := decimal.NewFromString(value)
		if parseErr != nil || !parsed.IsPositive() {
			return nil, biz.ErrFinanceBillInvalidArgument
		}
		estimatedRate = &parsed
	}
	if estimatedRate != nil && estimatedCurrency == nil {
		return nil, biz.ErrFinanceBillInvalidArgument
	}
	input := biz.UpdateFinanceBillInput{ID: id, BillDate: request.GetBillDate(), DueDate: request.DueDate, Note: request.Note, StatementTitle: request.StatementTitle, PaymentTermsDays: financeInt32Pointer(request.PaymentTermsDays), ExpectedVersion: request.GetExpectedVersion(), SettlementAccountID: accountID, EstimatedInvoiceCurrency: estimatedCurrency, EstimatedInvoiceRate: estimatedRate}
	item, err := s.billUsecase.Update(ctx, organizationIDs, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateBillResponse{Data: financeBillToAPI(item)}), nil
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

// ListBillFeeCandidates 返回可加入账单的费用候选；账单范围按 bill.update 写权限解析。
func (s *SettlementService) ListBillFeeCandidates(ctx context.Context, request *v1.ListBillFeeCandidatesRequest) (*v1.ListBillFeeCandidatesResponse, error) {
	principal, billID, err := financePrincipalAndID(ctx, request.GetBillId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceBillInvalidArgument)
	if err != nil {
		return nil, err
	}
	result, err := s.billUsecase.ListFeeCandidates(ctx, organizationIDs, billID, biz.FinanceBillFeeCandidateFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword)})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FeeLedgerItem, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeBillableFeeToAPI(item))
	}
	return okList(ctx, &v1.ListBillFeeCandidatesResponse{Data: data, Total: result.Total}), nil
}

// UpdateBillFees 为账单增加与移除费用，复用账单编辑权限 bill.update。
func (s *SettlementService) UpdateBillFees(ctx context.Context, request *v1.UpdateBillFeesRequest) (*v1.UpdateBillFeesResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceBillUpdate, true)
	if err != nil {
		return nil, err
	}
	addFeeIDs, err := parseTrimmedUUIDValues(request.GetAddFeeIds(), biz.ErrFinanceBillInvalidArgument)
	if err != nil {
		return nil, err
	}
	removeFeeIDs, err := parseTrimmedUUIDValues(request.GetRemoveFeeIds(), biz.ErrFinanceBillInvalidArgument)
	if err != nil {
		return nil, err
	}
	item, err := s.billUsecase.UpdateFees(ctx, organizationIDs, principal.UserID, biz.UpdateFinanceBillFeesInput{ID: id, ExpectedVersion: request.GetExpectedVersion(), AddFeeIDs: addFeeIDs, RemoveFeeIDs: removeFeeIDs})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateBillFeesResponse{Data: financeBillToAPI(item)}), nil
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
		SettlementAccountId: item.SettlementAccountID.String(), SettlementAccountName: item.SettlementAccountName, SettlementAccountHolder: item.SettlementAccountHolder, SettlementBankName: item.SettlementBankName, SettlementBankAccount: item.SettlementBankAccount, SettlementAccountCurrency: item.SettlementAccountCurrency, SettlementSwiftCode: item.SettlementSwiftCode,
		Currency: item.Currency, BaseCurrency: item.BaseCurrency, TotalAmount: item.TotalAmount.StringFixed(8), NetAmount: item.NetAmount.StringFixed(8),
		TaxAmount: item.TaxAmount.StringFixed(8), BaseCurrencyAmount: item.BaseCurrencyAmount.StringFixed(8), FeeCount: int32(item.FeeCount),
		BillDate: item.BillDate, DueDate: item.DueDate, Note: item.Note, Version: item.Version,
		ConfirmedAt: financeTime(item.ConfirmedAt), ConfirmedBy: uuidStringPtr(item.ConfirmedBy), CancelledAt: financeTime(item.CancelledAt),
		CancelledBy: uuidStringPtr(item.CancelledBy), CancellationReason: item.CancellationReason, Lines: lines,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		VerifiedAmount: item.VerifiedAmount.StringFixed(8), UnverifiedAmount: item.UnverifiedAmount.StringFixed(8), NettedAmount: item.NettedAmount.StringFixed(8),
		BatchId: uuidStringPtr(item.BatchID), BatchNo: financeOptionalValue(item.BatchNo), StatementTitle: item.StatementTitle, PaymentTermsDays: financeIntPointerToInt32(item.PaymentTermsDays),
		ExchangeRate: item.ExchangeRate.StringFixed(8), ExchangeRateSource: item.ExchangeRateSource, ExchangeRateDate: item.ExchangeRateDate, ExchangeRateSettingId: uuidStringPtr(item.ExchangeRateSettingID),
		EstimatedInvoiceCurrency: item.EstimatedInvoiceCurrency, EstimatedInvoiceRate: financeDecimalPointer(item.EstimatedInvoiceRate, 8), EstimatedInvoiceAmount: financeDecimalPointer(item.EstimatedInvoiceAmount, 8),
		Purpose: string(item.Purpose), InvoiceableAmount: financeDecimalPointer(item.InvoiceableAmount, 8), InvoiceableNetAmount: financeDecimalPointer(item.InvoiceableNetAmount, 8), InvoiceableTaxAmount: financeDecimalPointer(item.InvoiceableTaxAmount, 8), OverdueDays: item.OverdueDays,
		SettlementProgress: financeBillSettlementProgressToAPI(item.SettlementProgress), InvoiceProgress: financeBillInvoiceProgressToAPI(item.InvoiceProgress),
		Completed: item.Completed, FeesLocked: item.FeesLocked(), FeeLockFacts: financeBillFeeLockFactsToAPI(item.FeeLockFacts),
		InvoiceExemptions: financeInvoiceExemptionsToAPI(item.InvoiceExemptions),
	}
}

// financeBillSettlementProgressToAPI 未经投影的账单（如建账响应）返回 UNSPECIFIED。
func financeBillSettlementProgressToAPI(value biz.FinanceBillSettlementProgress) v1.FinanceBillSettlementProgress {
	switch value {
	case biz.FinanceBillSettlementUnsettled:
		return v1.FinanceBillSettlementProgress_FINANCE_BILL_SETTLEMENT_PROGRESS_UNSETTLED
	case biz.FinanceBillSettlementPartial:
		return v1.FinanceBillSettlementProgress_FINANCE_BILL_SETTLEMENT_PROGRESS_PARTIAL
	case biz.FinanceBillSettlementSettled:
		return v1.FinanceBillSettlementProgress_FINANCE_BILL_SETTLEMENT_PROGRESS_SETTLED
	default:
		return v1.FinanceBillSettlementProgress_FINANCE_BILL_SETTLEMENT_PROGRESS_UNSPECIFIED
	}
}

func financeBillInvoiceProgressToAPI(value biz.FinanceBillInvoiceProgress) v1.FinanceBillInvoiceProgress {
	switch value {
	case biz.FinanceBillInvoiceNotRequired:
		return v1.FinanceBillInvoiceProgress_FINANCE_BILL_INVOICE_PROGRESS_NOT_REQUIRED
	case biz.FinanceBillInvoiceNone:
		return v1.FinanceBillInvoiceProgress_FINANCE_BILL_INVOICE_PROGRESS_NONE
	case biz.FinanceBillInvoicePartial:
		return v1.FinanceBillInvoiceProgress_FINANCE_BILL_INVOICE_PROGRESS_PARTIAL
	case biz.FinanceBillInvoiceFull:
		return v1.FinanceBillInvoiceProgress_FINANCE_BILL_INVOICE_PROGRESS_FULL
	default:
		return v1.FinanceBillInvoiceProgress_FINANCE_BILL_INVOICE_PROGRESS_UNSPECIFIED
	}
}

func financeBillFeeLockFactsToAPI(values []biz.FinanceBillFeeLockFact) []v1.FinanceBillFeeLockFact {
	result := make([]v1.FinanceBillFeeLockFact, 0, len(values))
	for _, value := range values {
		if converted, exists := v1.FinanceBillFeeLockFact_value["FINANCE_BILL_FEE_LOCK_FACT_"+string(value)]; exists {
			result = append(result, v1.FinanceBillFeeLockFact(converted))
		}
	}
	return result
}

func financeBillBatchToAPI(item *biz.FinanceBillBatch) *v1.FinanceBillBatch {
	if item == nil {
		return nil
	}
	bills := make([]*v1.FinanceBill, 0, len(item.Bills))
	for _, bill := range item.Bills {
		bills = append(bills, financeBillToAPI(bill))
	}
	nettings := make([]*v1.FinanceNetting, 0, len(item.Nettings))
	for _, netting := range item.Nettings {
		nettings = append(nettings, financeNettingToAPI(netting))
	}
	baseTotals := make([]*v1.FinanceBillBatchBaseTotal, 0, len(item.BaseTotals))
	for _, total := range item.BaseTotals {
		baseTotals = append(baseTotals, &v1.FinanceBillBatchBaseTotal{BaseCurrency: total.BaseCurrency, TotalBaseAmount: total.TotalBaseAmount.StringFixed(8)})
	}
	return &v1.FinanceBillBatch{Id: item.ID.String(), BatchNo: item.BatchNo, SplitByOrder: item.GroupingPolicy.SplitByOrder, SplitByTaxRate: item.GroupingPolicy.SplitByTaxRate, Mode: financeBillGroupingModeToAPI(item.GroupingPolicy.Mode), FeeCount: int32(item.FeeCount), BillCount: int32(item.BillCount), BaseTotals: baseTotals, Bills: bills, Nettings: nettings, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)}
}

func financeBillGroupingModeToAPI(value string) v1.BillGroupingMode {
	if value == "NORMAL" {
		return v1.BillGroupingMode_BILL_GROUPING_MODE_NORMAL
	}
	if value == "NETTING" {
		return v1.BillGroupingMode_BILL_GROUPING_MODE_NETTING
	}
	return v1.BillGroupingMode_BILL_GROUPING_MODE_UNSPECIFIED
}

func financeBillGroupingPolicyFromAPI(value *v1.BillGroupingPolicy) biz.FinanceBillGroupingPolicy {
	mode := ""
	if value.GetMode() == v1.BillGroupingMode_BILL_GROUPING_MODE_NORMAL {
		mode = "NORMAL"
	}
	if value.GetMode() == v1.BillGroupingMode_BILL_GROUPING_MODE_NETTING {
		mode = "NETTING"
	}
	return biz.FinanceBillGroupingPolicy{Mode: mode, SplitByOrder: value.GetSplitByOrder(), SplitByTaxRate: value.GetSplitByTaxRate()}
}

type previewBillBatchConfigs struct {
	groups []biz.FinanceBillBatchPreviewGroupConfig
}

func previewBillBatchConfigsFromAPI(request *v1.PreviewBillBatchRequest) (previewBillBatchConfigs, error) {
	result := previewBillBatchConfigs{}
	for _, item := range request.GetGroupConfigs() {
		if item == nil {
			return previewBillBatchConfigs{}, biz.ErrFinanceBillInvalidArgument
		}
		config := biz.FinanceBillBatchPreviewGroupConfig{GroupKey: strings.TrimSpace(item.GetGroupKey()), BillDate: strings.TrimSpace(item.GetBillDate())}
		if value := strings.TrimSpace(item.GetSettlementAccountId()); value != "" {
			id, err := uuid.Parse(value)
			if err != nil {
				return previewBillBatchConfigs{}, biz.ErrFinanceBillInvalidArgument
			}
			config.SettlementAccountID = id
		}
		if value := strings.ToUpper(strings.TrimSpace(item.GetEstimatedInvoiceCurrency())); value != "" {
			config.EstimatedInvoiceCurrency = &value
		}
		if value := strings.TrimSpace(item.GetEstimatedInvoiceRate()); value != "" {
			rate, err := decimal.NewFromString(value)
			if err != nil || !rate.IsPositive() {
				return previewBillBatchConfigs{}, biz.ErrFinanceBillInvalidArgument
			}
			config.EstimatedInvoiceRate = &rate
		}
		if config.EstimatedInvoiceRate != nil && config.EstimatedInvoiceCurrency == nil {
			return previewBillBatchConfigs{}, biz.ErrFinanceBillInvalidArgument
		}
		result.groups = append(result.groups, config)
	}
	return result, nil
}

func financeBillableFeeToAPI(item *biz.FinanceBillableFee) *v1.FeeLedgerItem {
	fee := item.Fee
	return &v1.FeeLedgerItem{EconomicEffect: string(fee.EconomicEffect), CorrectionRequestId: uuidStringPtr(fee.CorrectionRequestID), HasPendingFeeCorrection: item.HasPendingFeeCorrection, OrderLocked: item.OrderLocked, HasPendingFeeSupplement: item.HasPendingFeeSupplement, CanBill: item.CanBill, BlockedReasons: item.BlockedReasons, Id: fee.ID.String(), OrderId: fee.OrderID.String(), OrderNo: item.OrderNo, BusinessType: item.BusinessType, OrganizationId: item.OrganizationID.String(), Direction: string(biz.SettlementDirection(fee)), FeeCode: fee.FeeCode, FeeName: fee.FeeName, SettlementPartyId: fee.SettlementPartyID.String(), SettlementPartyName: fee.SettlementPartyName, BillingUnit: fee.BillingUnit, Quantity: fee.Quantity.StringFixed(4), UnitPrice: fee.UnitPrice.StringFixed(4), TotalAmount: fee.TotalAmount.StringFixed(8), NetAmount: fee.NetAmount.StringFixed(8), TaxAmount: fee.TaxAmount.StringFixed(8), TaxRate: financeDecimalPointer(fee.TaxRate, 4), Currency: fee.Currency, ExchangeRate: fee.ExchangeRate.StringFixed(8), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount.StringFixed(8), BaseCurrencyNetAmount: fee.BaseCurrencyNetAmount().StringFixed(8), Note: fee.Note, Version: fee.Version, CreatedAt: fee.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: fee.UpdatedAt.UTC().Format(time.RFC3339)}
}
