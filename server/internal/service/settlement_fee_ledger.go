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
	feeStatus, financialProgress, err := feeLedgerStatusFilters(
		financeOptionalString(request.Status),
		financeOptionalString(request.FinancialProgress),
	)
	if err != nil {
		return nil, err
	}
	filter.Status = feeStatus
	filter.FinancialProgress = financialProgress
	filter.Currency = financeOptionalString(request.Currency)
	filter.BillNo = financeOptionalString(request.BillNo)
	filter.ExpenseDateFrom = financeOptionalString(request.ExpenseDateFrom)
	filter.ExpenseDateTo = financeOptionalString(request.ExpenseDateTo)
	filter.FinanceLocked = request.FinanceLocked
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceLedgerInvalidArgument
		}
		filter.SettlementPartyID = &id
	}
	if request.CustomerId != nil && strings.TrimSpace(*request.CustomerId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.CustomerId))
		if err != nil {
			return nil, biz.ErrFinanceLedgerInvalidArgument
		}
		filter.CustomerID = &id
	}
	result, err := s.usecase.ListFeeLedger(ctx, principal.Organization.ID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FeeLedgerItem, 0, len(result.Items))
	for _, item := range result.Items {
		fee := item.Fee
		data = append(data, &v1.FeeLedgerItem{
			Id: fee.ID.String(), OrderId: fee.OrderID.String(), OrderNo: item.OrderNo, BusinessType: item.Business, CustomerId: item.CustomerID.String(), CustomerName: item.CustomerName,
			Direction: string(fee.Direction), Status: string(fee.Status), FeeCode: fee.FeeCode, FeeName: fee.FeeName,
			SettlementPartyId: fee.SettlementPartyID.String(), SettlementPartyName: fee.SettlementPartyName, BillingUnit: fee.BillingUnit,
			Quantity: fee.Quantity.StringFixed(4), UnitPrice: fee.UnitPrice.StringFixed(4), TotalAmount: fee.TotalAmount.StringFixed(8),
			NetAmount: fee.NetAmount.StringFixed(8), TaxAmount: fee.TaxAmount.StringFixed(8), Currency: fee.Currency,
			ExchangeRate: fee.ExchangeRate.StringFixed(8), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount.StringFixed(8),
			ExpenseDate: fee.ExpenseDate, Note: fee.Note, Version: fee.Version,
			CreatedAt: fee.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: fee.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			TaxRate:           financeDecimalPointer(fee.TaxRate, 4),
			FinancialProgress: string(item.FinancialProgress),
			FinanceLocked:     item.FinanceLocked,
		})
		if item.BillNo != "" {
			data[len(data)-1].BillNo = &item.BillNo
		}
	}
	return &v1.ListFeeLedgerResponse{
		Success: true, Code: 0, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx),
		Summary: &v1.FeeLedgerSummary{ActiveCount: result.Summary.ActiveCount, ReceivableBaseAmount: result.Summary.ReceivableBaseAmount.StringFixed(8), PayableBaseAmount: result.Summary.PayableBaseAmount.StringFixed(8), ProfitBaseAmount: result.Summary.ProfitBaseAmount.StringFixed(8), BaseCurrency: result.Summary.BaseCurrency},
	}, nil
}

func feeLedgerStatusFilters(statusValue, progressValue string) (biz.OrderFeeStatus, biz.FeeLedgerFinancialProgress, error) {
	statusValue = strings.ToUpper(strings.TrimSpace(statusValue))
	progress := biz.FeeLedgerFinancialProgress(strings.ToUpper(strings.TrimSpace(progressValue)))
	// 兼容现有费用明细页把七种财务进度放入 status 的请求；正式客户端应使用 financial_progress。
	if legacyProgress := biz.FeeLedgerFinancialProgress(statusValue); biz.IsFeeLedgerFinancialProgress(legacyProgress) {
		if progress != "" && progress != legacyProgress {
			return "", "", biz.ErrFinanceLedgerInvalidArgument
		}
		return "", legacyProgress, nil
	}
	return biz.OrderFeeStatus(statusValue), progress, nil
}

func (s *SettlementService) GetFeeLedgerPreference(ctx context.Context, _ *v1.GetFeeLedgerPreferenceRequest) (*v1.GetFeeLedgerPreferenceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	preference, err := s.preferenceUsecase.Get(ctx, principal.Organization.ID, principal.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.GetFeeLedgerPreferenceResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    feeLedgerPreferenceToAPI(preference),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *SettlementService) UpdateFeeLedgerPreference(ctx context.Context, request *v1.UpdateFeeLedgerPreferenceRequest) (*v1.UpdateFeeLedgerPreferenceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	columns := make([]biz.FeeLedgerColumnPreference, 0, len(request.GetColumns()))
	for _, column := range request.GetColumns() {
		if column == nil {
			return nil, biz.ErrFeeLedgerPreferenceInvalidArgument
		}
		columns = append(columns, biz.FeeLedgerColumnPreference{FieldKey: column.GetFieldKey(), Visible: column.GetVisible()})
	}
	colors := request.GetRowColors()
	if colors == nil {
		return nil, biz.ErrFeeLedgerPreferenceInvalidArgument
	}
	preference, err := s.preferenceUsecase.Save(ctx, principal.Organization.ID, principal.UserID, &biz.FeeLedgerPreference{
		Columns:       columns,
		PageSize:      int(request.GetPageSize()),
		SortField:     financeOptionalString(request.SortField),
		SortDirection: financeOptionalString(request.SortDirection),
		RowColors: biz.FeeLedgerRowColors{
			Unbilled:                    colors.GetUnbilled(),
			UnverifiedUninvoiced:        colors.GetUnverifiedUninvoiced(),
			InvoicedUnverified:          colors.GetInvoicedUnverified(),
			VerifiedUninvoiced:          colors.GetVerifiedUninvoiced(),
			InvoicedPartiallyVerified:   colors.GetInvoicedPartiallyVerified(),
			PartiallyVerifiedUninvoiced: colors.GetPartiallyVerifiedUninvoiced(),
			Completed:                   colors.GetCompleted(),
		},
		Version: request.GetVersion(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateFeeLedgerPreferenceResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    feeLedgerPreferenceToAPI(preference),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *SettlementService) ResetFeeLedgerPreference(ctx context.Context, request *v1.ResetFeeLedgerPreferenceRequest) (*v1.ResetFeeLedgerPreferenceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	preference, err := s.preferenceUsecase.Reset(ctx, principal.Organization.ID, principal.UserID, request.GetVersion())
	if err != nil {
		return nil, err
	}
	return &v1.ResetFeeLedgerPreferenceResponse{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    feeLedgerPreferenceToAPI(preference),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *SettlementService) GetBilledFeeEditPolicy(ctx context.Context, _ *v1.GetBilledFeeEditPolicyRequest) (*v1.GetBilledFeeEditPolicyResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	policy, err := s.customSettingUsecase.GetBilledFeeEditPolicy(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return &v1.GetBilledFeeEditPolicyResponse{Success: true, Code: 0, Message: "OK", Data: billedFeeEditPolicyToAPI(policy), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *SettlementService) UpdateBilledFeeEditPolicy(ctx context.Context, request *v1.UpdateBilledFeeEditPolicyRequest) (*v1.UpdateBilledFeeEditPolicyResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	if request == nil || request.ExpectedVersion == nil {
		return nil, biz.ErrFinanceCustomSettingInvalidArgument
	}
	fields := make([]biz.BilledFeeEditableField, 0, len(request.GetEditableFields()))
	for _, field := range request.GetEditableFields() {
		converted, valid := billedFeeEditableFieldFromAPI(field)
		if !valid {
			return nil, biz.ErrFinanceCustomSettingInvalidArgument
		}
		fields = append(fields, converted)
	}
	policy, err := s.customSettingUsecase.UpdateBilledFeeEditPolicy(ctx, principal.Organization.ID, principal.UserID, &biz.BilledFeeEditPolicy{Enabled: request.GetEnabled(), EditableFields: fields}, request.GetExpectedVersion().GetValue())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateBilledFeeEditPolicyResponse{Success: true, Code: 0, Message: "OK", Data: billedFeeEditPolicyToAPI(policy), TraceId: requestmeta.TraceID(ctx)}, nil
}

func billedFeeEditableFieldFromAPI(field v1.BilledFeeEditableField) (biz.BilledFeeEditableField, bool) {
	switch field {
	case v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_FEE_NAME:
		return biz.BilledFeeFieldFeeName, true
	case v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_CURRENCY:
		return biz.BilledFeeFieldCurrency, true
	case v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_EXCHANGE_RATE:
		return biz.BilledFeeFieldExchangeRate, true
	case v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_QUANTITY:
		return biz.BilledFeeFieldQuantity, true
	case v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_UNIT_PRICE:
		return biz.BilledFeeFieldUnitPrice, true
	case v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_TAX_RATE:
		return biz.BilledFeeFieldTaxRate, true
	default:
		return "", false
	}
}

func billedFeeEditableFieldToAPI(field biz.BilledFeeEditableField) v1.BilledFeeEditableField {
	switch field {
	case biz.BilledFeeFieldFeeName:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_FEE_NAME
	case biz.BilledFeeFieldCurrency:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_CURRENCY
	case biz.BilledFeeFieldExchangeRate:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_EXCHANGE_RATE
	case biz.BilledFeeFieldQuantity:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_QUANTITY
	case biz.BilledFeeFieldUnitPrice:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_UNIT_PRICE
	case biz.BilledFeeFieldTaxRate:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_TAX_RATE
	default:
		return v1.BilledFeeEditableField_BILLED_FEE_EDITABLE_FIELD_UNSPECIFIED
	}
}

func billedFeeEditPolicyToAPI(policy *biz.BilledFeeEditPolicy) *v1.BilledFeeEditPolicy {
	if policy == nil {
		return nil
	}
	fields := make([]v1.BilledFeeEditableField, 0, len(policy.EditableFields))
	for _, field := range policy.EditableFields {
		fields = append(fields, billedFeeEditableFieldToAPI(field))
	}
	result := &v1.BilledFeeEditPolicy{OrganizationId: policy.OrganizationID.String(), Enabled: policy.Enabled, EditableFields: fields, Version: policy.Version}
	if policy.UpdatedAt != nil {
		value := policy.UpdatedAt.UTC().Format(time.RFC3339)
		result.UpdatedAt = &value
	}
	if policy.UpdatedBy != nil {
		value := policy.UpdatedBy.String()
		result.UpdatedBy = &value
	}
	return result
}

func feeLedgerPreferenceToAPI(preference *biz.FeeLedgerPreference) *v1.FeeLedgerPreference {
	columns := make([]*v1.FeeLedgerColumnPreference, 0, len(preference.Columns))
	for _, column := range preference.Columns {
		columns = append(columns, &v1.FeeLedgerColumnPreference{FieldKey: column.FieldKey, Visible: column.Visible})
	}
	result := &v1.FeeLedgerPreference{
		Columns:    columns,
		PageSize:   int32(preference.PageSize),
		RowColors:  &v1.FeeLedgerRowColors{Unbilled: preference.RowColors.Unbilled, UnverifiedUninvoiced: preference.RowColors.UnverifiedUninvoiced, InvoicedUnverified: preference.RowColors.InvoicedUnverified, VerifiedUninvoiced: preference.RowColors.VerifiedUninvoiced, InvoicedPartiallyVerified: preference.RowColors.InvoicedPartiallyVerified, PartiallyVerifiedUninvoiced: preference.RowColors.PartiallyVerifiedUninvoiced, Completed: preference.RowColors.Completed},
		Version:    preference.Version,
		Customized: preference.Customized,
	}
	if preference.SortField != "" {
		result.SortField = &preference.SortField
		result.SortDirection = &preference.SortDirection
	}
	if !preference.UpdatedAt.IsZero() {
		updatedAt := preference.UpdatedAt.UTC().Format(time.RFC3339)
		result.UpdatedAt = &updatedAt
	}
	return result
}
