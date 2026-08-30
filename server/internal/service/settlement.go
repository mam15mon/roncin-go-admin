package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// SettlementService 按子域拆分实现：费账台账（settlement_fee_ledger.go）、
// 账单（settlement_bill.go）、发票（settlement_invoice.go）、资金流水
// （settlement_cashflow.go）、核销（settlement_verification.go）、佣金
// （settlement_commission.go）；本文件保留服务锚点与跨子域共享的转换辅助。
type SettlementService struct {
	v1.UnimplementedSettlementServiceServer
	usecase              *biz.SettlementUsecase
	billUsecase          *biz.FinanceBillUsecase
	invoiceUsecase       *biz.FinanceInvoiceUsecase
	cashflowUsecase      *biz.FinanceCashflowUsecase
	verificationUsecase  *biz.VerificationUsecase
	commissionUsecase    *biz.CommissionUsecase
	preferenceUsecase    *biz.FeeLedgerPreferenceUsecase
	customSettingUsecase *biz.FinanceCustomSettingUsecase
	tagUsecase           *biz.BusinessTagUsecase
}

func NewSettlementService(usecase *biz.SettlementUsecase, billUsecase *biz.FinanceBillUsecase, invoiceUsecase *biz.FinanceInvoiceUsecase, cashflowUsecase *biz.FinanceCashflowUsecase, verificationUsecase *biz.VerificationUsecase, commissionUsecase *biz.CommissionUsecase, preferenceUsecase *biz.FeeLedgerPreferenceUsecase, customSettingUsecase *biz.FinanceCustomSettingUsecase, tagUsecase *biz.BusinessTagUsecase) *SettlementService {
	return &SettlementService{usecase: usecase, billUsecase: billUsecase, invoiceUsecase: invoiceUsecase, cashflowUsecase: cashflowUsecase, verificationUsecase: verificationUsecase, commissionUsecase: commissionUsecase, preferenceUsecase: preferenceUsecase, customSettingUsecase: customSettingUsecase, tagUsecase: tagUsecase}
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

func financeOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ v1.SettlementServiceServer = (*SettlementService)(nil)
