package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func billBatchOperationsFromAPI(group *v1.CreateBillBatchGroupInput) (_ []biz.FinanceBillBatchInvoiceInput, _ []biz.FinanceBillBatchVerificationInput, returnErr error) {
	step, itemIndex := "INVOICE", 0
	defer func() {
		if returnErr != nil {
			returnErr = &biz.FinanceBillBatchOperationInputError{Step: step, ItemIndex: itemIndex, Err: returnErr}
		}
	}()
	invoices := make([]biz.FinanceBillBatchInvoiceInput, 0, len(group.GetInvoices()))
	for index, raw := range group.GetInvoices() {
		step, itemIndex = "INVOICE", index+1
		if raw == nil {
			return nil, nil, biz.ErrFinanceBillInvalidArgument
		}
		profile, err := uuid.Parse(strings.TrimSpace(raw.GetInvoiceProfileId()))
		if err != nil {
			return nil, nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		total, err := billBatchOperationDecimal(raw.GetTotalAmount())
		if err != nil {
			return nil, nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		item := biz.FinanceBillBatchInvoiceInput{InvoiceProfileID: profile, InvoiceType: biz.FinanceInvoiceType(raw.GetInvoiceType()), TaxInvoiceNo: raw.GetTaxInvoiceNo(), InvoiceDate: raw.GetInvoiceDate(), TotalAmount: total, Note: raw.Note}
		for _, rawFee := range raw.GetFees() {
			if rawFee == nil {
				return nil, nil, biz.ErrFinanceInvoiceInvalidArgument
			}
			id, err := uuid.Parse(strings.TrimSpace(rawFee.GetOrderFeeId()))
			if err != nil {
				return nil, nil, biz.ErrFinanceInvoiceInvalidArgument
			}
			amount, err := billBatchOperationDecimal(rawFee.GetAmount())
			if err != nil {
				return nil, nil, biz.ErrFinanceInvoiceInvalidArgument
			}
			tax, err := billBatchOperationDecimal(rawFee.GetTaxAmount())
			if err != nil {
				return nil, nil, biz.ErrFinanceInvoiceInvalidArgument
			}
			item.Fees = append(item.Fees, biz.FinanceInvoiceFeeInput{OrderFeeID: id, Amount: amount, TaxAmount: tax})
		}
		invoices = append(invoices, item)
	}
	verifications := make([]biz.FinanceBillBatchVerificationInput, 0, len(group.GetVerifications()))
	for index, raw := range group.GetVerifications() {
		step, itemIndex = "VERIFICATION", index+1
		if raw == nil {
			return nil, nil, biz.ErrVerificationInvalid
		}
		item := biz.FinanceBillBatchVerificationInput{VerificationDate: raw.GetVerificationDate(), Note: raw.Note}
		for _, rawAllocation := range raw.GetAllocations() {
			if rawAllocation == nil {
				return nil, nil, biz.ErrVerificationInvalid
			}
			id := uuid.Nil
			if value := strings.TrimSpace(rawAllocation.GetCashflowId()); value != "" {
				var err error
				id, err = uuid.Parse(value)
				if err != nil {
					return nil, nil, biz.ErrVerificationInvalid
				}
			}
			amount, err := billBatchOperationDecimal(rawAllocation.GetAmount())
			if err != nil {
				return nil, nil, biz.ErrVerificationInvalid
			}
			item.Allocations = append(item.Allocations, biz.FinanceBillBatchVerificationAllocation{CashflowID: id, Amount: amount})
		}
		if rawCash := raw.GetShortfallCashflow(); rawCash != nil {
			amount, err := billBatchOperationDecimal(rawCash.GetAmount())
			if err != nil {
				return nil, nil, biz.ErrFinanceCashflowInvalidArgument
			}
			cash := &biz.FinanceBillBatchCashflowInput{Amount: amount, TransactionDate: rawCash.GetTransactionDate(), OurAccount: rawCash.GetOurAccount(), PaymentMethod: rawCash.GetPaymentMethod(), CounterpartyAccount: rawCash.CounterpartyAccount, BankReferenceNo: rawCash.BankReferenceNo, Note: rawCash.Note}
			if rawCash.ExchangeRate != nil {
				rate, err := billBatchOperationDecimal(*rawCash.ExchangeRate)
				if err != nil {
					return nil, nil, biz.ErrFinanceCashflowInvalidArgument
				}
				cash.ExchangeRateOverride = &rate
			}
			item.ShortfallCashflow = cash
		}
		verifications = append(verifications, item)
	}
	return invoices, verifications, nil
}

func billBatchOperationPermissions(principal *biz.Principal, org uuid.UUID) biz.FinanceBillBatchOperationPermissions {
	allowed := func(permission string) bool {
		ids, err := organizationIDsForPermission(principal, permission, true)
		return err == nil && uuidIn(org, ids)
	}
	return biz.FinanceBillBatchOperationPermissions{
		RegisterInvoices: allowed(access.FinanceInvoiceCreate), CreateVerifications: allowed(access.FinanceVerificationCreate),
		CreateCashflows: allowed(access.FinanceCashflowCreate), ConfirmCashflows: allowed(access.FinanceCashflowUpdate), OverrideCashflowRate: allowed(access.FinanceExchangeRateUpdate),
	}
}

// boundaryOperationFingerprint 仅保存边界未解析操作的摘要，原始账户/票号不写日志或数据库。
func boundaryOperationFingerprint(group *v1.CreateBillBatchGroupInput) string {
	raw, _ := json.Marshal(struct {
		Invoices      []*v1.BillBatchInvoiceInput
		Verifications []*v1.BillBatchVerificationInput
	}{group.GetInvoices(), group.GetVerifications()})
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// 先限制指数再比较或序列化，避免非法指数金额在组摘要阶段展开为巨型文本。
func billBatchOperationDecimal(raw string) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || value.Exponent() < -8 || value.Exponent() > 20 || !value.Abs().LessThan(decimal.New(1, 20)) {
		return decimal.Zero, biz.ErrFinanceBillInvalidArgument
	}
	return value, nil
}
