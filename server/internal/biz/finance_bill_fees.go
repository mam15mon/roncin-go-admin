package biz

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
)

var (
	// ErrFinanceBillFeesLocked 表示账单存在有效发票、核销、对冲（含草稿对冲）等财务事实，禁止增删费用。
	ErrFinanceBillFeesLocked = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_FEES_LOCKED), "账单已有发票、核销或对冲记录，不能增删费用；请先撤销相关财务记录")
	// ErrFinanceBillLastFee 表示增删后账单不再有有效费用；释放全部费用须取消账单。
	ErrFinanceBillLastFee = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_LAST_FEE), "账单至少保留一条费用；如需释放全部费用，请取消账单")
)

var financeBillFeeLockFactLabels = map[FinanceBillFeeLockFact]string{
	FinanceBillFeeLockInvoice:          "发票",
	FinanceBillFeeLockVerification:     "核销",
	FinanceBillFeeLockNetting:          "已确认对冲",
	FinanceBillFeeLockNettingDraft:     "草稿对冲",
	FinanceBillFeeLockInvoiceExemption: "无需发票标记",
}

// FinanceBillFeesLockedError 返回带事实类型的费用锁定错误；提示文案列出全部事实，元数据 facts 供审计与排查。
func FinanceBillFeesLockedError(facts []FinanceBillFeeLockFact) error {
	codes := make([]string, 0, len(facts))
	labels := make([]string, 0, len(facts))
	for _, fact := range facts {
		codes = append(codes, string(fact))
		if label, exists := financeBillFeeLockFactLabels[fact]; exists {
			labels = append(labels, label)
		}
	}
	message := "账单已有" + strings.Join(labels, "、") + "，不能增删费用；请先撤销相关财务记录"
	return errors.New(int(ErrFinanceBillFeesLocked.Code), ErrFinanceBillFeesLocked.Reason, message).WithMetadata(map[string]string{"facts": strings.Join(codes, ",")})
}

// UpdateFinanceBillFeesInput 是账单费用增删请求；增加与移除可同时出现。
type UpdateFinanceBillFeesInput struct {
	ID              uuid.UUID
	ExpectedVersion uint64
	AddFeeIDs       []uuid.UUID
	RemoveFeeIDs    []uuid.UUID
}

// FinanceBillFeeCandidateFilter 是账单可加入费用候选的分页与关键字过滤条件。
type FinanceBillFeeCandidateFilter struct {
	Page, PageSize int
	Keyword        string
}

// normalizeUpdateFinanceBillFees 校验增删列表：至少一项非空、各不超过分页上限、无空 ID、无重复且增删不交叉；
// 结果按 ID 排序，便于仓储固定加锁顺序。
func normalizeUpdateFinanceBillFees(input UpdateFinanceBillFeesInput) (UpdateFinanceBillFeesInput, error) {
	if input.ID == uuid.Nil || input.ExpectedVersion == 0 || (len(input.AddFeeIDs) == 0 && len(input.RemoveFeeIDs) == 0) || len(input.AddFeeIDs) > MaxListPageSize || len(input.RemoveFeeIDs) > MaxListPageSize {
		return UpdateFinanceBillFeesInput{}, ErrFinanceBillInvalidArgument
	}
	seen := make(map[uuid.UUID]struct{}, len(input.AddFeeIDs)+len(input.RemoveFeeIDs))
	for _, ids := range [][]uuid.UUID{input.AddFeeIDs, input.RemoveFeeIDs} {
		for _, id := range ids {
			if id == uuid.Nil {
				return UpdateFinanceBillFeesInput{}, ErrFinanceBillInvalidArgument
			}
			if _, exists := seen[id]; exists {
				return UpdateFinanceBillFeesInput{}, ErrFinanceBillInvalidArgument
			}
			seen[id] = struct{}{}
		}
	}
	input.AddFeeIDs = sortedFinanceBillFeeIDs(input.AddFeeIDs)
	input.RemoveFeeIDs = sortedFinanceBillFeeIDs(input.RemoveFeeIDs)
	return input, nil
}

func sortedFinanceBillFeeIDs(ids []uuid.UUID) []uuid.UUID {
	result := append([]uuid.UUID{}, ids...)
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

// ValidateBillFeeForAttach 是费用加入已有账单的逐项校验，账单费用候选与增删提交共用：
// 费用须当前有效且需要建账、未被有效账单占用、属于账单公司，并满足 validateBillFeeScope 的账单口径。
// 不检查待审更正：仓储在费用锁内单独复核。
func ValidateBillFeeForAttach(bill *FinanceBill, item *FinanceBillableFee) error {
	if bill == nil || item == nil || item.Fee == nil || !RequiresAccounting(item.Fee) || item.OrganizationID != bill.OrganizationID {
		return ErrFinanceBillFeeInvalid
	}
	return validateBillFeeScope(bill, item)
}

// validateBillFeeScope 是建账与增删费用共用的账单口径校验：费用未被有效账单占用，且收付方向、结算单位、
// 币种、本币、账单用途与更正来源均与账单一致。建账的「需要建账、当前有效、同公司」由仓储锁内复核。
func validateBillFeeScope(bill *FinanceBill, item *FinanceBillableFee) error {
	if bill == nil || item == nil || item.Fee == nil || item.Fee.HasActiveBill {
		return ErrFinanceBillFeeInvalid
	}
	fee := item.Fee
	if fee.SourceFeeID != nil && (bill.CorrectionRequestID == nil || fee.CorrectionRequestID == nil || *bill.CorrectionRequestID != *fee.CorrectionRequestID) {
		return ErrFinanceBillFeeMismatch
	}
	if fee.SourceFeeID == nil && bill.CorrectionRequestID != nil {
		return ErrFinanceBillFeeMismatch
	}
	if FinanceBillPurposeForFee(fee) != bill.Purpose || SettlementDirection(fee) != bill.Direction || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != bill.Currency || fee.BaseCurrency != bill.BaseCurrency {
		return ErrFinanceBillFeeMismatch
	}
	return nil
}

// NewFinanceBillLineFromFee 以费用当前金额快照构造有效账单行；汇率与本位币金额由账单重算统一填写。
func NewFinanceBillLineFromFee(billID uuid.UUID, item *FinanceBillableFee) *FinanceBillLine {
	fee := item.Fee
	return &FinanceBillLine{
		ID: uuid.Must(uuid.NewV7()), BillID: billID, OrderFeeID: fee.ID, OrderID: fee.OrderID,
		OrderNo: item.OrderNo, BusinessType: item.BusinessType, FeeCode: fee.FeeCode, FeeName: fee.FeeName,
		Quantity: fee.Quantity, UnitPrice: fee.UnitPrice, TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, Currency: fee.Currency,
		TaxRate: fee.TaxRate, ExchangeRate: decimal.NewFromInt(1), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount, Active: true,
	}
}

// RecomputeFinanceBillAmounts 按账单自身汇率快照重算账单头与全部有效行：
// 总额、不含税额、税额为有效行之和；本位币金额 = 总额 × 账单汇率（RoundBank(8)），
// 行本位币逐行乘账单汇率、最后一行吸收尾差；fee_count 为有效行数；预计开票金额按已存预计币种与汇率重算。
// 行汇率统一为账单汇率，不读取费用自身汇率。lines 的顺序决定尾差归属，调用方须传入稳定顺序。
func RecomputeFinanceBillAmounts(bill *FinanceBill, lines []*FinanceBillLine) error {
	if bill == nil || !bill.ExchangeRate.IsPositive() {
		return ErrFinanceBillInvalidArgument
	}
	if len(lines) == 0 {
		return ErrFinanceBillLastFee
	}
	total, net, tax := decimal.Zero, decimal.Zero, decimal.Zero
	for _, line := range lines {
		if line == nil || line.Currency != bill.Currency || line.BaseCurrency != bill.BaseCurrency {
			return ErrFinanceBillFeeMismatch
		}
		total, net, tax = total.Add(line.TotalAmount), net.Add(line.NetAmount), tax.Add(line.TaxAmount)
	}
	bill.TotalAmount, bill.NetAmount, bill.TaxAmount = total, net, tax
	bill.BaseCurrencyAmount = total.Mul(bill.ExchangeRate).RoundBank(8)
	allocated := decimal.Zero
	for index, line := range lines {
		line.ExchangeRate = bill.ExchangeRate
		lineBase := line.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
		if index == len(lines)-1 {
			lineBase = bill.BaseCurrencyAmount.Sub(allocated)
		}
		line.BaseCurrencyAmount = lineBase
		allocated = allocated.Add(lineBase)
	}
	bill.Lines = lines
	bill.FeeCount = len(lines)
	if bill.EstimatedInvoiceCurrency != nil && bill.EstimatedInvoiceRate != nil {
		currency, rate, amount, err := financeBillEstimatedInvoiceSnapshot(bill.Currency, total, FinanceBillBatchPreviewGroupConfig{EstimatedInvoiceCurrency: bill.EstimatedInvoiceCurrency, EstimatedInvoiceRate: bill.EstimatedInvoiceRate})
		if err != nil {
			return err
		}
		bill.EstimatedInvoiceCurrency, bill.EstimatedInvoiceRate, bill.EstimatedInvoiceAmount = &currency, &rate, &amount
	}
	return validateFinanceBillAmountInvariants(bill)
}

// ListFeeCandidates 返回可加入账单的费用候选；服务端筛选后再以 ValidateBillFeeForAttach 复核，
// 不满足的候选标记为不可选并给出原因，保证候选与提交同一口径。
func (uc *FinanceBillUsecase) ListFeeCandidates(ctx context.Context, organizationIDs []uuid.UUID, billID uuid.UUID, filter FinanceBillFeeCandidateFilter) (*FinanceBillCreationCandidateResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if !validFinanceBillOrganizationIDs(organizationIDs) || billID == uuid.Nil || !ValidListPagination(filter.Page, filter.PageSize) || utf8.RuneCountInString(filter.Keyword) > 100 {
		return nil, ErrFinanceBillInvalidArgument
	}
	bill, err := uc.repo.Get(ctx, organizationIDs, billID)
	if err != nil {
		return nil, err
	}
	if bill.Status != FinanceBillConfirmed {
		return nil, ErrFinanceBillInvalidTransition
	}
	result, err := uc.repo.ListBillFeeCandidates(ctx, bill, filter)
	if err != nil {
		return nil, err
	}
	for _, item := range result.Items {
		if validationErr := ValidateBillFeeForAttach(bill, item); validationErr != nil {
			item.CanBill = false
			item.BlockedReasons = append(item.BlockedReasons, errors.FromError(validationErr).Message)
		}
	}
	return result, nil
}

// UpdateFees 在同一事务中为账单增加与移除费用。锁序、锁定事实复核与金额重算由仓储在事务内完成，
// 完整响应在事务提交后用普通上下文重读。
func (uc *FinanceBillUsecase) UpdateFees(ctx context.Context, organizationIDs []uuid.UUID, actorID uuid.UUID, input UpdateFinanceBillFeesInput) (*FinanceBill, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || uc.transactor == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	normalized, err := normalizeUpdateFinanceBillFees(input)
	if err != nil {
		return nil, err
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, normalized.ID)
	if err != nil {
		return nil, err
	}
	audit := financeBillAudit(existing.OrganizationID, actorID, normalized.ID, "finance.bill.fees.update")
	audit.Details["added_fee_ids"] = joinFinanceBillFeeIDs(normalized.AddFeeIDs)
	audit.Details["removed_fee_ids"] = joinFinanceBillFeeIDs(normalized.RemoveFeeIDs)
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		return uc.repo.UpdateFees(txCtx, organizationIDs, normalized, func(lockedCtx context.Context, bill *FinanceBill) error {
			bill.BillDate = uc.businessDate()
			if bill.PaymentTermsDays != nil {
				bill.DueDate = normalizedFinanceBillDueDate(bill.BillDate, nil, bill.PaymentTermsDays)
			}
			if !validFinanceBillTerms(bill.BillDate, bill.DueDate, bill.PaymentTermsDays) {
				return errors.BadRequest("FINANCE_BILL_DUE_DATE_BEFORE_CREATION", "到期日早于本次费用变动日期，请先调整到期日后再增减费用")
			}
			return uc.applyBillExchangeRate(lockedCtx, bill.OrganizationID, bill)
		}, audit)
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, organizationIDs, normalized.ID)
}

func joinFinanceBillFeeIDs(ids []uuid.UUID) string {
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	return strings.Join(values, ",")
}
