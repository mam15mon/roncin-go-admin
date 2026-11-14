package biz

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// 组合载荷不接受尚未存在的账单ID/版本，也不接受另一个公司的结算归属。
type FinanceBillBatchInvoiceInput struct {
	InvoiceProfileID          uuid.UUID
	InvoiceType               FinanceInvoiceType
	TaxInvoiceNo, InvoiceDate string
	TotalAmount               decimal.Decimal
	Note                      *string
	Fees                      []FinanceInvoiceFeeInput
}

type FinanceBillBatchVerificationAllocation struct {
	CashflowID uuid.UUID
	Amount     decimal.Decimal
}

type FinanceBillBatchCashflowInput struct {
	Amount                                     decimal.Decimal
	ExchangeRateOverride                       *decimal.Decimal
	TransactionDate, OurAccount, PaymentMethod string
	CounterpartyAccount, BankReferenceNo, Note *string
}

type FinanceBillBatchVerificationInput struct {
	VerificationDate  string
	Note              *string
	Allocations       []FinanceBillBatchVerificationAllocation
	ShortfallCashflow *FinanceBillBatchCashflowInput
}

// FinanceBillBatchOperationInputError 保留边界解析的组内步骤定位，不携带Proto类型。
type FinanceBillBatchOperationInputError struct {
	Step      string
	ItemIndex int
	Err       error
}

func (e *FinanceBillBatchOperationInputError) Error() string { return e.Err.Error() }
func (e *FinanceBillBatchOperationInputError) Unwrap() error { return e.Err }

// 授权范围由传输层逐权限解析；业务层按组检查，不因缺附带动作权限退化成只建账。
type FinanceBillBatchOperationPermissions struct {
	RegisterInvoices, CreateVerifications, CreateCashflows, ConfirmCashflows, OverrideCashflowRate bool
}

type FinanceBillBatchOperations struct {
	Invoices      *FinanceInvoiceUsecase
	Verifications *VerificationUsecase
	Cashflows     *FinanceCashflowUsecase
	Permissions   FinanceBillBatchOperationPermissions
}

func normalizeFinanceBillBatchOperations(group *CreateFinanceBillBatchGroupInput) error {
	if len(group.Invoices) > 100 || len(group.Verifications) > 100 {
		return ErrFinanceBillInvalidArgument
	}
	group.Invoices = append([]FinanceBillBatchInvoiceInput(nil), group.Invoices...)
	for index := range group.Invoices {
		item := &group.Invoices[index]
		item.TaxInvoiceNo = strings.TrimSpace(item.TaxInvoiceNo)
		item.InvoiceDate = strings.TrimSpace(item.InvoiceDate)
		item.InvoiceType = FinanceInvoiceType(strings.ToUpper(strings.TrimSpace(string(item.InvoiceType))))
		item.Note = normalizedOptionalFinanceString(item.Note)
		item.Fees = append([]FinanceInvoiceFeeInput(nil), item.Fees...)
		sort.Slice(item.Fees, func(i, j int) bool { return item.Fees[i].OrderFeeID.String() < item.Fees[j].OrderFeeID.String() })
	}
	group.Verifications = append([]FinanceBillBatchVerificationInput(nil), group.Verifications...)
	for index := range group.Verifications {
		item := &group.Verifications[index]
		item.VerificationDate = strings.TrimSpace(item.VerificationDate)
		item.Note = normalizedOptionalFinanceString(item.Note)
		item.Allocations = append([]FinanceBillBatchVerificationAllocation(nil), item.Allocations...)
		sort.Slice(item.Allocations, func(i, j int) bool {
			return item.Allocations[i].CashflowID.String() < item.Allocations[j].CashflowID.String()
		})
		if item.ShortfallCashflow != nil {
			cash := *item.ShortfallCashflow
			cash.TransactionDate, cash.OurAccount, cash.PaymentMethod = strings.TrimSpace(cash.TransactionDate), strings.TrimSpace(cash.OurAccount), strings.TrimSpace(cash.PaymentMethod)
			cash.CounterpartyAccount, cash.BankReferenceNo, cash.Note = normalizedOptionalFinanceString(cash.CounterpartyAccount), normalizedOptionalFinanceString(cash.BankReferenceNo), normalizedOptionalFinanceString(cash.Note)
			item.ShortfallCashflow = &cash
		}
	}
	return nil
}

func financeBillBatchOperationsHash(group CreateFinanceBillBatchGroupInput) string {
	// decimal 的JSON编码使用规范数值文本，空可选文本已统一为nil。
	payload, _ := json.Marshal(struct {
		Invoices      []FinanceBillBatchInvoiceInput
		Verifications []FinanceBillBatchVerificationInput
	}{group.Invoices, group.Verifications})
	return financeSHA256(string(payload) + ":" + group.OperationInputFingerprint)
}

func financeBillBatchStepKey(batchKey, groupKey, step string, index int) string {
	var builder strings.Builder
	writeFinanceHashParts(&builder, batchKey, groupKey, step, strconv.Itoa(index))
	return "batch-step:" + financeSHA256(builder.String())
}

func financeBillBatchStepError(err error, groupKey, step string, index int) error {
	if err == nil {
		return nil
	}
	base := errors.FromError(err)
	metadata := make(map[string]string, len(base.Metadata)+3)
	for key, value := range base.Metadata {
		metadata[key] = value
	}
	metadata["step"], metadata["item_index"] = step, strconv.Itoa(index)
	if groupKey != "" {
		metadata["group_key"] = groupKey
	}
	message := base.Message
	if base.Reason == errors.UnknownReason || base.Code >= 500 {
		message = "组合操作失败，请稍后重试"
	}
	return errors.New(int(base.Code), base.Reason, message).WithMetadata(metadata).WithCause(err)
}

func validateFinanceBillBatchOperationPermissions(group CreateFinanceBillBatchGroupInput, operations FinanceBillBatchOperations) error {
	if group.OperationInputError != nil {
		return financeBillBatchStepError(group.OperationInputError.Err, group.GroupKey, group.OperationInputError.Step, group.OperationInputError.ItemIndex)
	}
	if len(group.Invoices) > 0 {
		if !operations.Permissions.RegisterInvoices {
			return financeBillBatchStepError(ErrPermissionDenied, group.GroupKey, "INVOICE", 1)
		}
		if operations.Invoices == nil {
			return financeBillBatchStepError(ErrFinanceBillInvalidArgument, group.GroupKey, "INVOICE", 0)
		}
	}
	for index, item := range group.Verifications {
		if !operations.Permissions.CreateVerifications {
			return financeBillBatchStepError(ErrPermissionDenied, group.GroupKey, "VERIFICATION", index+1)
		}
		if operations.Verifications == nil {
			return financeBillBatchStepError(ErrFinanceBillInvalidArgument, group.GroupKey, "VERIFICATION", index+1)
		}
		if item.ShortfallCashflow != nil {
			if !operations.Permissions.CreateCashflows || !operations.Permissions.ConfirmCashflows || (item.ShortfallCashflow.ExchangeRateOverride != nil && !operations.Permissions.OverrideCashflowRate) {
				return financeBillBatchStepError(ErrPermissionDenied, group.GroupKey, "VERIFICATION", index+1)
			}
			if operations.Cashflows == nil {
				return financeBillBatchStepError(ErrFinanceBillInvalidArgument, group.GroupKey, "VERIFICATION", index+1)
			}
		}
	}
	return nil
}

func lockFinanceBillBatchCashflows(ctx context.Context, org uuid.UUID, groups []CreateFinanceBillBatchGroupInput, operations FinanceBillBatchOperations) error {
	var ids []uuid.UUID
	for _, group := range groups {
		for _, item := range group.Verifications {
			for _, allocation := range item.Allocations {
				if allocation.CashflowID != uuid.Nil {
					ids = append(ids, allocation.CashflowID)
				}
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	err := operations.Verifications.LockCashflowsWithinTransaction(ctx, org, ids)
	if err == nil {
		return nil
	}
	missingID := errors.FromError(err).Metadata["cashflow_id"]
	for _, group := range groups {
		for index, item := range group.Verifications {
			for _, allocation := range item.Allocations {
				if allocation.CashflowID != uuid.Nil && (missingID == "" || allocation.CashflowID.String() == missingID) {
					return financeBillBatchStepError(err, group.GroupKey, "VERIFICATION", index+1)
				}
			}
		}
	}
	return err
}

func executeFinanceBillBatchOperations(ctx context.Context, org, actor uuid.UUID, batchKey string, group CreateFinanceBillBatchGroupInput, bill *FinanceBill, operations FinanceBillBatchOperations) error {
	if len(group.Invoices) > 0 {
		records := make([]FinanceInvoiceRegistrationInput, 0, len(group.Invoices))
		for _, item := range group.Invoices {
			records = append(records, FinanceInvoiceRegistrationInput{InvoiceProfileID: item.InvoiceProfileID, InvoiceType: item.InvoiceType, TaxInvoiceNo: item.TaxInvoiceNo, InvoiceDate: item.InvoiceDate, TotalAmount: item.TotalAmount, Note: item.Note,
				Allocations: []FinanceInvoiceAllocationInput{{BillID: bill.ID, ExpectedVersion: 1, Amount: item.TotalAmount, Fees: item.Fees}}})
		}
		_, err := operations.Invoices.RegisterWithinTransaction(ctx, []uuid.UUID{org}, actor, RegisterFinanceInvoicesInput{IdempotencyKey: financeBillBatchStepKey(batchKey, group.GroupKey, "INVOICE", 0), Records: records})
		if err != nil {
			index := 0
			if raw, ok := errors.FromError(err).Metadata["invoice_index"]; ok {
				if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed >= 0 && parsed < len(records) {
					index = parsed + 1
				}
			}
			return financeBillBatchStepError(err, group.GroupKey, "INVOICE", index)
		}
	}
	for index, item := range group.Verifications {
		input := CreateVerificationInput{VerificationDate: item.VerificationDate, Note: item.Note, IdempotencyKey: financeBillBatchStepKey(batchKey, group.GroupKey, "VERIFICATION", index)}
		for _, allocation := range item.Allocations {
			input.Allocations = append(input.Allocations, &VerificationAllocation{CashflowID: allocation.CashflowID, BillID: bill.ID, Amount: allocation.Amount})
		}
		var err error
		if item.ShortfallCashflow == nil {
			_, err = operations.Verifications.CreateWithinTransaction(ctx, org, actor, input)
		} else {
			cash := item.ShortfallCashflow
			cashInput := CreateFinanceCashflowInput{Direction: bill.Direction, SettlementPartyID: bill.SettlementPartyID, Currency: bill.Currency, Amount: cash.Amount, ExchangeRateOverride: cash.ExchangeRateOverride, TransactionDate: cash.TransactionDate, OurAccount: cash.OurAccount, PaymentMethod: cash.PaymentMethod, CounterpartyAccount: cash.CounterpartyAccount, BankReferenceNo: cash.BankReferenceNo, Note: cash.Note}
			_, err = operations.Verifications.CreateWithCashflowWithinTransaction(ctx, org, actor, input, cashInput, operations.Cashflows, operations.Permissions.OverrideCashflowRate)
		}
		if err != nil {
			return financeBillBatchStepError(err, group.GroupKey, "VERIFICATION", index+1)
		}
	}
	return nil
}

// 完整财务投影只在原子写入提交后读取；读取失败整请求返回未知结果，前端继续原键确认。
func (uc *FinanceBillUsecase) enrichFinanceBillBatchOperationResults(ctx context.Context, org uuid.UUID, input CreateFinanceBillBatchInput, result *FinanceBillBatchCreateResult) (*FinanceBillBatchCreateResult, error) {
	hasOperations := map[string]bool{}
	for _, group := range input.Groups {
		hasOperations[group.GroupKey] = len(group.Invoices) > 0 || len(group.Verifications) > 0
	}
	fullBills := map[uuid.UUID]*FinanceBill{}
	for _, group := range result.Groups {
		if group.Bill == nil || group.Err != nil || !hasOperations[group.GroupKey] {
			continue
		}
		bill, err := uc.repo.Get(ctx, []uuid.UUID{org}, group.Bill.ID)
		if err != nil {
			return nil, err
		}
		group.Bill = bill
		fullBills[bill.ID] = bill
	}
	for index, bill := range result.Batch.Bills {
		if full := fullBills[bill.ID]; full != nil {
			result.Batch.Bills[index] = full
		}
	}
	return result, nil
}
