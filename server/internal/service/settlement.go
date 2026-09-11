package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// SettlementService 按子域拆分实现：费账台账（settlement_fee_ledger.go）、
// 账单（settlement_bill.go）、发票（settlement_invoice.go）、资金流水
// （settlement_cashflow.go）、核销（settlement_verification.go）、对冲
// （settlement_netting.go）、佣金（settlement_commission.go）；本文件保留
// 服务锚点与跨子域共享的转换辅助。
type SettlementService struct {
	v1.UnimplementedSettlementServiceServer
	usecase              *biz.SettlementUsecase
	billUsecase          *biz.FinanceBillUsecase
	invoiceUsecase       *biz.FinanceInvoiceUsecase
	cashflowUsecase      *biz.FinanceCashflowUsecase
	verificationUsecase  *biz.VerificationUsecase
	nettingUsecase       *biz.FinanceNettingUsecase
	commissionUsecase    *biz.CommissionUsecase
	preferenceUsecase    *biz.FeeLedgerPreferenceUsecase
	customSettingUsecase *biz.FinanceCustomSettingUsecase
	tagUsecase           *biz.BusinessTagUsecase
	accountUsecase       *biz.PartnerAccountUsecase
}

func (s *SettlementService) ListFinanceOrganizationOptions(ctx context.Context, request *v1.ListFinanceOrganizationOptionsRequest) (*v1.ListFinanceOrganizationOptionsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	permission, writable, ok := financeOrganizationPurposePermission(request.GetPurpose())
	if !ok {
		return nil, biz.ErrFinanceLedgerInvalidArgument
	}
	organizationIDs, err := organizationIDsForPermission(principal, permission, writable)
	if err != nil {
		return nil, err
	}
	items, err := s.usecase.ListFinanceOrganizations(ctx, organizationIDs, financeOptionalString(request.Keyword))
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceOrganizationOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.FinanceOrganizationOption{Id: item.ID.String(), Code: item.Code, Name: item.Name, BaseCurrency: item.BaseCurrency})
	}
	return okList(ctx, &v1.ListFinanceOrganizationOptionsResponse{Data: data}), nil
}

func (s *SettlementService) ListFinanceSettlementPartyOptions(ctx context.Context, request *v1.ListFinanceSettlementPartyOptionsRequest) (*v1.ListFinanceSettlementPartyOptionsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	permission, writable, ok := financeOrganizationPurposePermission(request.GetPurpose())
	if !ok || !writable || (request.GetPurpose() != v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_CASHFLOW_CREATE &&
		request.GetPurpose() != v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE) {
		return nil, biz.ErrFinanceLedgerInvalidArgument
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFinanceLedgerInvalidArgument)
	if err != nil {
		return nil, err
	}
	rawOrganizationID := request.GetOrganizationId()
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, permission, writable, &rawOrganizationID)
	if err != nil {
		return nil, err
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrFinanceLedgerInvalidArgument
	}
	items, total, err := s.usecase.ListFinanceSettlementParties(ctx, organizationIDs[0], financeOptionalString(request.Keyword), page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceSettlementPartyOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.FinanceSettlementPartyOption{Id: item.ID, Code: item.Code, Name: item.Name})
	}
	return okList(ctx, &v1.ListFinanceSettlementPartyOptionsResponse{Data: data, Total: total}), nil
}

func financeOrganizationPurposePermission(purpose v1.FinanceOrganizationPurpose) (string, bool, bool) {
	switch purpose {
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_BILL_READ:
		return access.FinanceBillRead, false, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_BILL_CREATE:
		return access.FinanceBillCreate, true, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_INVOICE_READ:
		return access.FinanceInvoiceRead, false, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_INVOICE_CREATE:
		return access.FinanceInvoiceCreate, true, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_CASHFLOW_READ:
		return access.FinanceCashflowRead, false, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_CASHFLOW_CREATE:
		return access.FinanceCashflowCreate, true, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_READ:
		return access.FinanceVerificationRead, false, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE:
		return access.FinanceVerificationCreate, true, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_COMMISSION_READ:
		return access.FinanceCommissionRead, false, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_COMMISSION_MANAGE:
		return access.FinanceCommissionManage, true, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_FEE_READ:
		return access.FinanceFeeRead, false, true
	case v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_NETTING_READ:
		return access.FinanceNettingRead, false, true
	default:
		return "", false, false
	}
}

func NewSettlementService(usecase *biz.SettlementUsecase, billUsecase *biz.FinanceBillUsecase, invoiceUsecase *biz.FinanceInvoiceUsecase, cashflowUsecase *biz.FinanceCashflowUsecase, verificationUsecase *biz.VerificationUsecase, nettingUsecase *biz.FinanceNettingUsecase, commissionUsecase *biz.CommissionUsecase, preferenceUsecase *biz.FeeLedgerPreferenceUsecase, customSettingUsecase *biz.FinanceCustomSettingUsecase, tagUsecase *biz.BusinessTagUsecase, accountUsecase *biz.PartnerAccountUsecase) *SettlementService {
	return &SettlementService{usecase: usecase, billUsecase: billUsecase, invoiceUsecase: invoiceUsecase, cashflowUsecase: cashflowUsecase, verificationUsecase: verificationUsecase, nettingUsecase: nettingUsecase, commissionUsecase: commissionUsecase, preferenceUsecase: preferenceUsecase, customSettingUsecase: customSettingUsecase, tagUsecase: tagUsecase, accountUsecase: accountUsecase}
}

func financePrincipalAndID(ctx context.Context, rawID string) (*biz.Principal, uuid.UUID, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, uuid.Nil, principalErr
	}
	id, err := uuid.Parse(strings.TrimSpace(rawID))
	if err != nil {
		return nil, uuid.Nil, biz.ErrFinanceBillInvalidArgument
	}
	return principal, id, nil
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

func financeOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func financeBillStatusFromAPI(value *v1.FinanceBillStatus) biz.FinanceBillStatus {
	if value == nil {
		return ""
	}
	return biz.FinanceBillStatus(strings.TrimPrefix(value.String(), "FINANCE_BILL_STATUS_"))
}

func financeBillStatusToAPI(value biz.FinanceBillStatus) v1.FinanceBillStatus {
	switch value {
	case biz.FinanceBillDraft:
		return v1.FinanceBillStatus_FINANCE_BILL_STATUS_DRAFT
	case biz.FinanceBillConfirmed:
		return v1.FinanceBillStatus_FINANCE_BILL_STATUS_CONFIRMED
	case biz.FinanceBillCancelled:
		return v1.FinanceBillStatus_FINANCE_BILL_STATUS_CANCELLED
	default:
		return v1.FinanceBillStatus_FINANCE_BILL_STATUS_UNSPECIFIED
	}
}

func financeInvoiceStatusFromAPI(value *v1.FinanceInvoiceStatus) biz.FinanceInvoiceStatus {
	if value == nil {
		return ""
	}
	return biz.FinanceInvoiceStatus(strings.TrimPrefix(value.String(), "FINANCE_INVOICE_STATUS_"))
}

func financeInvoiceStatusToAPI(value biz.FinanceInvoiceStatus) v1.FinanceInvoiceStatus {
	switch value {
	case biz.FinanceInvoiceDraft:
		return v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_DRAFT
	case biz.FinanceInvoiceIssued:
		return v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_ISSUED
	case biz.FinanceInvoiceCancelled:
		return v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_CANCELLED
	case biz.FinanceInvoiceRedFlushed:
		return v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_RED_FLUSHED
	default:
		return v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_UNSPECIFIED
	}
}

func financeCashflowStatusFromAPI(value *v1.FinanceCashflowStatus) biz.FinanceCashflowStatus {
	if value == nil {
		return ""
	}
	return biz.FinanceCashflowStatus(strings.TrimPrefix(value.String(), "FINANCE_CASHFLOW_STATUS_"))
}

func financeCashflowStatusToAPI(value biz.FinanceCashflowStatus) v1.FinanceCashflowStatus {
	switch value {
	case biz.FinanceCashflowDraft:
		return v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_DRAFT
	case biz.FinanceCashflowConfirmed:
		return v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_CONFIRMED
	case biz.FinanceCashflowCancelled:
		return v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_CANCELLED
	default:
		return v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_UNSPECIFIED
	}
}

func financeVerificationStatusFromAPI(value *v1.FinanceVerificationStatus) biz.VerificationStatus {
	if value == nil {
		return ""
	}
	return biz.VerificationStatus(strings.TrimPrefix(value.String(), "FINANCE_VERIFICATION_STATUS_"))
}

func financeVerificationStatusToAPI(value biz.VerificationStatus) v1.FinanceVerificationStatus {
	switch value {
	case biz.VerificationActive:
		return v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_ACTIVE
	case biz.VerificationReversed:
		return v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_REVERSED
	default:
		return v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_UNSPECIFIED
	}
}

func financeCommissionStatusFromAPI(value *v1.FinanceCommissionStatus) biz.CommissionStatus {
	if value == nil {
		return ""
	}
	return biz.CommissionStatus(strings.TrimPrefix(value.String(), "FINANCE_COMMISSION_STATUS_"))
}

func financeCommissionStatusToAPI(value biz.CommissionStatus) v1.FinanceCommissionStatus {
	switch value {
	case biz.CommissionDraft:
		return v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_DRAFT
	case biz.CommissionConfirmed:
		return v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_CONFIRMED
	case biz.CommissionPaid:
		return v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_PAID
	case biz.CommissionCancelled:
		return v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_CANCELLED
	default:
		return v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_UNSPECIFIED
	}
}

func feeLedgerFinancialProgressFromAPI(value *v1.FeeLedgerFinancialProgress) biz.FeeLedgerFinancialProgress {
	if value == nil {
		return ""
	}
	return biz.FeeLedgerFinancialProgress(strings.TrimPrefix(value.String(), "FEE_LEDGER_FINANCIAL_PROGRESS_"))
}

func feeLedgerFinancialProgressToAPI(value biz.FeeLedgerFinancialProgress) v1.FeeLedgerFinancialProgress {
	switch value {
	case biz.FeeLedgerUnbilled:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_UNBILLED
	case biz.FeeLedgerUnverifiedUninvoiced:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_UNVERIFIED_UNINVOICED
	case biz.FeeLedgerInvoicedUnverified:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_UNVERIFIED
	case biz.FeeLedgerVerifiedUninvoiced:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_VERIFIED_UNINVOICED
	case biz.FeeLedgerInvoicedPartiallyVerified:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED
	case biz.FeeLedgerPartiallyVerifiedUninvoiced:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_PARTIALLY_VERIFIED_UNINVOICED
	case biz.FeeLedgerCompleted:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED
	default:
		return v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_UNSPECIFIED
	}
}

var _ v1.SettlementServiceServer = (*SettlementService)(nil)
