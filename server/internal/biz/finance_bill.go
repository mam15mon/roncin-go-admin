package biz

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
)

var (
	ErrFinanceBillNotFound                 = errors.NotFound("FINANCE_BILL_NOT_FOUND", "账单不存在")
	ErrFinanceBillInvalidArgument          = errors.BadRequest("FINANCE_BILL_INVALID_ARGUMENT", "账单字段不合法")
	ErrFinanceBillFeeInvalid               = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_FEE_INVALID), "所选费用必须为已确认状态且尚未进入其他账单")
	ErrFinanceBillFeeMismatch              = errors.BadRequest("FINANCE_BILL_FEE_MISMATCH", "同一账单的费用必须具有相同收付方向、结算单位、币种和本币")
	ErrFinanceBillVersionConflict          = errors.Conflict("FINANCE_BILL_VERSION_CONFLICT", "账单已被其他操作人修改，请刷新后重试")
	ErrFinanceBillInvalidTransition        = errors.Conflict("FINANCE_BILL_INVALID_TRANSITION", "当前账单状态不允许执行该操作")
	ErrFinanceBillIdempotencyConflict      = errors.Conflict("FINANCE_BILL_IDEMPOTENCY_CONFLICT", "账单请求幂等键已被其他请求使用")
	ErrFinanceBillPreviewStale             = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_PREVIEW_STALE), "费用或拆单结果已变化，请重新预览")
	ErrFinanceBillBatchMismatch            = errors.BadRequest("FINANCE_BILL_BATCH_MISMATCH", "批量账单分组资料与服务端预览不一致")
	ErrFinanceBillBatchConflict            = errors.Conflict("FINANCE_BILL_BATCH_CONFLICT", "批量建单幂等键已被其他请求使用")
	ErrFinanceBillSettlementAccountInvalid = errors.BadRequest(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_SETTLEMENT_ACCOUNT_INVALID), "结算账户与账单结算单位、方向、币种或启用状态不匹配")
	ErrFinanceBillMixedDirection           = errors.BadRequest("FINANCE_BILL_MIXED_DIRECTION", "普通账单需将应收、应付分别建账，请先完成一个方向，再创建另一个方向")
	ErrFinanceBillGroupingModeUnsupported  = errors.BadRequest("FINANCE_BILL_GROUPING_MODE_UNSUPPORTED", "当前阶段暂不支持对冲建账模式")
)

var financeBillCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type FinanceBillUsecase struct {
	repo         FinanceBillRepo
	exchangeRate *ExchangeRateUsecase
	transactor   Transactor
}

func NewFinanceBillUsecase(repo FinanceBillRepo, exchangeRate *ExchangeRateUsecase, transactor Transactor) *FinanceBillUsecase {
	return &FinanceBillUsecase{repo: repo, exchangeRate: exchangeRate, transactor: transactor}
}

func (uc *FinanceBillUsecase) List(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if !validFinanceBillOrganizationIDs(organizationIDs) || !ValidListPagination(filter.Page, filter.PageSize) || utf8.RuneCountInString(filter.Keyword) > 100 {
		return nil, ErrFinanceBillInvalidArgument
	}
	if filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable {
		return nil, ErrFinanceBillInvalidArgument
	}
	if filter.Status != "" && filter.Status != FinanceBillDraft && filter.Status != FinanceBillConfirmed && filter.Status != FinanceBillCancelled {
		return nil, ErrFinanceBillInvalidArgument
	}
	if filter.Currency != "" && !financeBillCurrencyPattern.MatchString(filter.Currency) {
		return nil, ErrFinanceBillInvalidArgument
	}
	if !validFinanceDateRange(filter.BillDateFrom, filter.BillDateTo) {
		return nil, ErrFinanceBillInvalidArgument
	}
	if !validFinanceDateRange(filter.DueDateFrom, filter.DueDateTo) {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.List(ctx, organizationIDs, filter)
}

func (uc *FinanceBillUsecase) Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceBill, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.Get(ctx, organizationIDs, id)
}

func (uc *FinanceBillUsecase) ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter FinanceBillCreationCandidateFilter) (*FinanceBillCreationCandidateResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if organizationID == uuid.Nil || !ValidListPagination(filter.Page, filter.PageSize) || utf8.RuneCountInString(filter.Keyword) > 100 || (filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable) {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.ListCreationCandidates(ctx, organizationID, filter)
}

func validFinanceBillOrganizationIDs(organizationIDs []uuid.UUID) bool {
	if len(organizationIDs) == 0 {
		return false
	}
	for _, organizationID := range organizationIDs {
		if organizationID == uuid.Nil {
			return false
		}
	}
	return true
}

func validateFinanceBillAmountInvariants(bill *FinanceBill) error {
	if bill == nil || len(bill.Lines) == 0 || !bill.ExchangeRate.IsPositive() || !bill.TotalAmount.Equal(bill.NetAmount.Add(bill.TaxAmount)) {
		return ErrFinanceBillBatchMismatch
	}
	total, net, tax, base := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	for _, line := range bill.Lines {
		if line == nil || !line.ExchangeRate.IsPositive() || !line.TotalAmount.Equal(line.NetAmount.Add(line.TaxAmount)) {
			return ErrFinanceBillBatchMismatch
		}
		total, net, tax, base = total.Add(line.TotalAmount), net.Add(line.NetAmount), tax.Add(line.TaxAmount), base.Add(line.BaseCurrencyAmount)
	}
	if !total.Equal(bill.TotalAmount) || !net.Equal(bill.NetAmount) || !tax.Equal(bill.TaxAmount) || !base.Equal(bill.BaseCurrencyAmount) {
		return ErrFinanceBillBatchMismatch
	}
	return nil
}

func (uc *FinanceBillUsecase) ResolveBillableFeeOrganization(ctx context.Context, organizationIDs, feeIDs []uuid.UUID) (uuid.UUID, error) {
	if len(organizationIDs) == 0 || len(feeIDs) == 0 {
		return uuid.Nil, ErrFinanceBillInvalidArgument
	}
	fees, err := uc.repo.LoadBillableFeesScoped(ctx, organizationIDs, feeIDs)
	if err != nil {
		return uuid.Nil, err
	}
	if len(fees) != len(feeIDs) {
		return uuid.Nil, ErrFinanceBillInvalidArgument
	}
	organizationID := fees[0].OrganizationID
	if organizationID == uuid.Nil {
		return uuid.Nil, ErrFinanceBillInvalidArgument
	}
	for _, fee := range fees[1:] {
		if fee.OrganizationID != organizationID {
			return uuid.Nil, ErrFinanceBillInvalidArgument
		}
	}
	return organizationID, nil
}

func (uc *FinanceBillUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillInput) (*FinanceBill, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	normalized, err := normalizeCreateFinanceBill(input)
	if err != nil {
		return nil, err
	}
	var created *FinanceBill
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		fees, transactionErr := uc.repo.LoadBillableFees(txCtx, organizationID, normalized.FeeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		// 服务端兜底注入（design 决策 A）：API 直录未显式提供账期且未提供到期日时，
		// 按预览同口径注入默认账期并联动到期日（散客恒 0；正式客户取激活规则默认账期）。
		// 注入在幂等意图比对之前，使重放请求按注入后的生效值与已存账单比对。
		if normalized.PaymentTermsDays == nil && normalized.DueDate == nil && len(fees) > 0 {
			injected, injectErr := uc.defaultFinanceBillPaymentTermsDays(txCtx, organizationID, fees[0])
			if injectErr != nil {
				return injectErr
			}
			if injected != nil {
				normalized.PaymentTermsDays = injected
				normalized.DueDate = normalizedFinanceBillDueDate(normalized.BillDate, nil, injected)
				if !validFinanceBillTerms(normalized.BillDate, normalized.DueDate, injected) {
					return ErrFinanceBillInvalidArgument
				}
			}
		}
		existing, transactionErr := uc.repo.GetByIdempotencyKey(txCtx, organizationID, normalized.IdempotencyKey)
		if transactionErr != nil {
			return transactionErr
		}
		if existing != nil {
			if !sameFinanceBillCreateIntent(existing, normalized) {
				return ErrFinanceBillIdempotencyConflict
			}
			created = existing
			return nil
		}
		bill, transactionErr := buildFinanceBill(organizationID, fees, normalized)
		if transactionErr != nil {
			return transactionErr
		}
		// 与批量建账保持相同锁序：费用 → 账户 → 汇率设置。
		if transactionErr = uc.repo.HydrateBillSettlementAccounts(txCtx, []*FinanceBill{bill}); transactionErr != nil {
			return transactionErr
		}
		if transactionErr = uc.repo.ValidateBillCurrencies(txCtx, []string{bill.Currency, bill.BaseCurrency}); transactionErr != nil {
			return transactionErr
		}
		if transactionErr = uc.applyBillExchangeRate(txCtx, organizationID, bill); transactionErr != nil {
			return transactionErr
		}
		created, transactionErr = uc.repo.Create(txCtx, bill, financeBillAudit(organizationID, actorID, bill.ID, "finance.bill.create"))
		return transactionErr
	})
	if err == nil {
		return uc.repo.Get(ctx, []uuid.UUID{organizationID}, created.ID)
	}
	existing, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey)
	if lookupErr == nil && existing != nil && sameFinanceBillCreateIntent(existing, normalized) {
		return existing, nil
	}
	return nil, err
}

func (uc *FinanceBillUsecase) applyBillExchangeRate(ctx context.Context, organizationID uuid.UUID, bill *FinanceBill) error {
	if uc.exchangeRate == nil || bill == nil {
		return ErrFinanceBillInvalidArgument
	}
	resolved, err := uc.exchangeRate.ResolveRate(ctx, organizationID, bill.Direction, bill.Currency, bill.BillDate)
	if err != nil {
		return err
	}
	baseCurrency, err := uc.exchangeRate.BaseCurrency(ctx, organizationID)
	if err != nil {
		return err
	}
	if baseCurrency != bill.BaseCurrency {
		return ErrFinanceBillFeeMismatch
	}
	bill.ExchangeRate = resolved.Rate.RoundBank(8)
	bill.ExchangeRateSource = resolved.Source
	bill.ExchangeRateDate = bill.BillDate
	bill.ExchangeRateSettingID = resolved.SettingID
	// 头本位币金额必须使用已固化（舍入到 8 位）的账单汇率，与批量内核口径一致。
	bill.BaseCurrencyAmount = bill.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
	allocated := decimal.Zero
	for index, line := range bill.Lines {
		line.ExchangeRate = bill.ExchangeRate
		lineBase := line.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
		if index == len(bill.Lines)-1 {
			lineBase = bill.BaseCurrencyAmount.Sub(allocated)
		}
		line.BaseCurrencyAmount = lineBase
		allocated = allocated.Add(lineBase)
	}
	return nil
}

func (uc *FinanceBillUsecase) Update(ctx context.Context, organizationIDs []uuid.UUID, actorID uuid.UUID, input UpdateFinanceBillInput) (*FinanceBill, error) {
	input.BillDate = strings.TrimSpace(input.BillDate)
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.DueDate = normalizedFinanceBillDueDate(input.BillDate, input.DueDate, input.PaymentTermsDays)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || input.ID == uuid.Nil || input.ExpectedVersion == 0 || !validFinanceDate(input.BillDate) || !validFinanceBillTerms(input.BillDate, input.DueDate, input.PaymentTermsDays) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, input.ID)
	if err != nil {
		return nil, err
	}
	if existing.Status != FinanceBillDraft || existing.Version != input.ExpectedVersion || uc.exchangeRate == nil || uc.transactor == nil {
		if existing.Status != FinanceBillDraft {
			return nil, ErrFinanceBillInvalidTransition
		}
		if existing.Version != input.ExpectedVersion {
			return nil, ErrFinanceBillVersionConflict
		}
		return nil, ErrFinanceBillInvalidArgument
	}
	feeIDs := make([]uuid.UUID, 0, len(existing.Lines))
	for _, line := range existing.Lines {
		feeIDs = append(feeIDs, line.OrderFeeID)
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 账单更新、取消和确认统一先锁账单；更新随后再按 ID 锁费用，避免与
		// “账单 → 费用”的状态操作形成反向等待。
		current, transactionErr := uc.repo.Get(txCtx, organizationIDs, input.ID)
		if transactionErr != nil {
			return transactionErr
		}
		if current.Status != FinanceBillDraft {
			return ErrFinanceBillInvalidTransition
		}
		if current.Version != input.ExpectedVersion {
			return ErrFinanceBillVersionConflict
		}
		lockedFees, transactionErr := uc.repo.LoadBillableFees(txCtx, current.OrganizationID, feeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		if transactionErr = validateLockedFinanceBillSourceFees(current, lockedFees); transactionErr != nil {
			return transactionErr
		}
		account := &FinanceBill{OrganizationID: current.OrganizationID, Direction: current.Direction, SettlementPartyID: current.SettlementPartyID, SettlementAccountID: input.SettlementAccountID, Currency: current.Currency}
		if transactionErr = uc.repo.HydrateBillSettlementAccounts(txCtx, []*FinanceBill{account}); transactionErr != nil {
			return transactionErr
		}
		currencies := []string{current.Currency, current.BaseCurrency}
		for _, fee := range lockedFees {
			currencies = append(currencies, fee.Fee.Currency)
		}
		if input.EstimatedInvoiceCurrency != nil {
			currencies = append(currencies, *input.EstimatedInvoiceCurrency)
		}
		if transactionErr = uc.repo.ValidateBillCurrencies(txCtx, currencies); transactionErr != nil {
			return transactionErr
		}
		group := &FinanceBillBatchPreviewGroup{Direction: current.Direction, SettlementPartyID: current.SettlementPartyID, SettlementPartyName: current.SettlementPartyName, Currency: current.Currency, BaseCurrency: current.BaseCurrency, BillDate: input.BillDate, SettlementAccountID: input.SettlementAccountID, Fees: lockedFees, config: FinanceBillBatchPreviewGroupConfig{BillDate: input.BillDate, SettlementAccountID: input.SettlementAccountID, EstimatedInvoiceCurrency: input.EstimatedInvoiceCurrency, EstimatedInvoiceRate: input.EstimatedInvoiceRate}}
		rebuilt, transactionErr := uc.buildFixedCurrencyFinanceBill(txCtx, current.OrganizationID, group, input.BillDate)
		if transactionErr != nil {
			return transactionErr
		}
		rebuilt.ID = current.ID
		rebuilt.SettlementAccountName, rebuilt.SettlementAccountHolder, rebuilt.SettlementBankName = account.SettlementAccountName, account.SettlementAccountHolder, account.SettlementBankName
		rebuilt.SettlementBankAccount, rebuilt.SettlementAccountCurrency, rebuilt.SettlementSwiftCode = account.SettlementBankAccount, account.SettlementAccountCurrency, account.SettlementSwiftCode
		lineIDs := make(map[uuid.UUID]uuid.UUID, len(current.Lines))
		for _, line := range current.Lines {
			lineIDs[line.OrderFeeID] = line.ID
		}
		for _, line := range rebuilt.Lines {
			line.ID = lineIDs[line.OrderFeeID]
			line.BillID = current.ID
		}
		input.ExchangeRate, input.ExchangeRateSource, input.ExchangeRateDate, input.ExchangeRateSettingID = rebuilt.ExchangeRate, rebuilt.ExchangeRateSource, rebuilt.ExchangeRateDate, rebuilt.ExchangeRateSettingID
		input.BaseCurrencyAmount, input.TotalAmount, input.NetAmount, input.TaxAmount, input.Lines = rebuilt.BaseCurrencyAmount, rebuilt.TotalAmount, rebuilt.NetAmount, rebuilt.TaxAmount, rebuilt.Lines
		input.EstimatedInvoiceCurrency, input.EstimatedInvoiceRate, input.EstimatedInvoiceAmount = rebuilt.EstimatedInvoiceCurrency, rebuilt.EstimatedInvoiceRate, rebuilt.EstimatedInvoiceAmount
		_, transactionErr = uc.repo.Update(txCtx, organizationIDs, input, financeBillAudit(current.OrganizationID, actorID, input.ID, "finance.bill.update"))
		return transactionErr
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, organizationIDs, input.ID)
}

func validateLockedFinanceBillSourceFees(bill *FinanceBill, fees []*FinanceBillableFee) error {
	if bill == nil || len(fees) != len(bill.Lines) {
		return ErrFinanceBillPreviewStale
	}
	lines := make(map[uuid.UUID]*FinanceBillLine, len(bill.Lines))
	for _, line := range bill.Lines {
		lines[line.OrderFeeID] = line
	}
	for _, item := range fees {
		fee, line := item.Fee, lines[item.Fee.ID]
		if line == nil || fee.Status != OrderFeeBilled || fee.Direction != bill.Direction || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != line.Currency || !fee.TotalAmount.Equal(line.TotalAmount) || !fee.NetAmount.Equal(line.NetAmount) || !fee.TaxAmount.Equal(line.TaxAmount) {
			return ErrFinanceBillPreviewStale
		}
	}
	return nil
}

func (uc *FinanceBillUsecase) Confirm(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64) (*FinanceBill, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.Confirm(ctx, organizationIDs, id, actorID, expectedVersion, financeBillAudit(existing.OrganizationID, actorID, id, "finance.bill.confirm"))
}

func (uc *FinanceBillUsecase) Cancel(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceBill, error) {
	reason = strings.TrimSpace(reason)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.Cancel(ctx, organizationIDs, id, actorID, expectedVersion, reason, financeBillAudit(existing.OrganizationID, actorID, id, "finance.bill.cancel"))
}

func normalizeCreateFinanceBill(input CreateFinanceBillInput) (CreateFinanceBillInput, error) {
	input.BillDate = strings.TrimSpace(input.BillDate)
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.DueDate = normalizedFinanceBillDueDate(input.BillDate, input.DueDate, input.PaymentTermsDays)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.FeeIDs) == 0 || len(input.FeeIDs) > 500 || input.SettlementAccountID == uuid.Nil || !validFinanceDate(input.BillDate) || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || !validFinanceBillTerms(input.BillDate, input.DueDate, input.PaymentTermsDays) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
		return CreateFinanceBillInput{}, ErrFinanceBillInvalidArgument
	}
	seen := make(map[uuid.UUID]struct{}, len(input.FeeIDs))
	for _, id := range input.FeeIDs {
		if id == uuid.Nil {
			return CreateFinanceBillInput{}, ErrFinanceBillInvalidArgument
		}
		if _, exists := seen[id]; exists {
			return CreateFinanceBillInput{}, ErrFinanceBillInvalidArgument
		}
		seen[id] = struct{}{}
	}
	sort.Slice(input.FeeIDs, func(i, j int) bool { return input.FeeIDs[i].String() < input.FeeIDs[j].String() })
	return input, nil
}

func buildFinanceBill(organizationID uuid.UUID, fees []*FinanceBillableFee, input CreateFinanceBillInput) (*FinanceBill, error) {
	if len(fees) != len(input.FeeIDs) || len(fees) == 0 {
		return nil, ErrFinanceBillFeeInvalid
	}
	first := fees[0]
	if first == nil || first.Fee == nil || first.Fee.Status != OrderFeeConfirmed {
		return nil, ErrFinanceBillFeeInvalid
	}
	billID := uuid.Must(uuid.NewV7())
	bill := &FinanceBill{
		ID: billID, OrganizationID: organizationID, IdempotencyKey: input.IdempotencyKey,
		Direction: first.Fee.Direction, Status: FinanceBillDraft, SettlementPartyID: first.Fee.SettlementPartyID,
		SettlementPartyName: first.Fee.SettlementPartyName, SettlementAccountID: input.SettlementAccountID, Currency: first.Fee.Currency, BaseCurrency: first.Fee.BaseCurrency,
		BillDate: input.BillDate, StatementTitle: input.StatementTitle, PaymentTermsDays: input.PaymentTermsDays, DueDate: input.DueDate, Note: input.Note, Version: 1,
		Lines: make([]*FinanceBillLine, 0, len(fees)),
	}
	if bill.StatementTitle == nil {
		bill.StatementTitle = &bill.SettlementPartyName
	}
	for _, item := range fees {
		if item == nil || item.Fee == nil || item.Fee.Status != OrderFeeConfirmed {
			return nil, ErrFinanceBillFeeInvalid
		}
		fee := item.Fee
		if fee.Direction != bill.Direction || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != bill.Currency || fee.BaseCurrency != bill.BaseCurrency {
			return nil, ErrFinanceBillFeeMismatch
		}
		bill.TotalAmount = bill.TotalAmount.Add(fee.TotalAmount)
		bill.NetAmount = bill.NetAmount.Add(fee.NetAmount)
		bill.TaxAmount = bill.TaxAmount.Add(fee.TaxAmount)
		bill.BaseCurrencyAmount = bill.BaseCurrencyAmount.Add(fee.BaseCurrencyAmount)
		bill.Lines = append(bill.Lines, &FinanceBillLine{
			ID: uuid.Must(uuid.NewV7()), BillID: billID, OrderFeeID: fee.ID, OrderID: fee.OrderID,
			OrderNo: item.OrderNo, BusinessType: item.BusinessType, FeeCode: fee.FeeCode, FeeName: fee.FeeName,
			Quantity: fee.Quantity, UnitPrice: fee.UnitPrice, TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, Currency: fee.Currency,
			TaxRate: fee.TaxRate, ExchangeRate: decimal.NewFromInt(1), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount, Active: true,
		})
	}
	bill.FeeCount = len(bill.Lines)
	return bill, nil
}

func sameFinanceBillCreateIntent(existing *FinanceBill, requested CreateFinanceBillInput) bool {
	if existing == nil {
		return false
	}
	requestedTitle := requested.StatementTitle
	if requestedTitle == nil {
		requestedTitle = &existing.SettlementPartyName
	}
	if existing.SettlementAccountID != requested.SettlementAccountID || existing.BillDate != requested.BillDate || !stringPointersEqual(existing.DueDate, requested.DueDate) || !stringPointersEqual(existing.Note, requested.Note) || !stringPointersEqual(existing.StatementTitle, requestedTitle) || !intPointersEqual(existing.PaymentTermsDays, requested.PaymentTermsDays) || len(existing.Lines) != len(requested.FeeIDs) {
		return false
	}
	ids := make([]string, 0, len(existing.Lines))
	for _, line := range existing.Lines {
		ids = append(ids, line.OrderFeeID.String())
	}
	sort.Strings(ids)
	for index, id := range requested.FeeIDs {
		if ids[index] != id.String() {
			return false
		}
	}
	return true
}

func normalizeFinanceBillFeeIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 || len(ids) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	result := append([]uuid.UUID(nil), ids...)
	seen := make(map[uuid.UUID]struct{}, len(result))
	for _, id := range result {
		if id == uuid.Nil {
			return nil, ErrFinanceBillInvalidArgument
		}
		if _, exists := seen[id]; exists {
			return nil, ErrFinanceBillInvalidArgument
		}
		seen[id] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result, nil
}

func validFinanceBillTerms(billDate string, dueDate *string, paymentTermsDays *int) bool {
	if paymentTermsDays != nil && (*paymentTermsDays < 0 || *paymentTermsDays > 3650) {
		return false
	}
	if dueDate != nil && (!validFinanceDate(*dueDate) || *dueDate < billDate) {
		return false
	}
	if paymentTermsDays == nil {
		return true
	}
	parsed, err := time.Parse("2006-01-02", billDate)
	if err != nil {
		return false
	}
	expected := parsed.AddDate(0, 0, *paymentTermsDays).Format("2006-01-02")
	return dueDate == nil || *dueDate == expected
}

func normalizedFinanceBillDueDate(billDate string, dueDate *string, paymentTermsDays *int) *string {
	if dueDate != nil || paymentTermsDays == nil {
		return dueDate
	}
	parsed, err := time.Parse("2006-01-02", billDate)
	if err != nil {
		return dueDate
	}
	value := parsed.AddDate(0, 0, *paymentTermsDays).Format("2006-01-02")
	return &value
}

func intPointersEqual(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func validFinanceDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func validFinanceDateRange(from, to string) bool {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	return (from == "" || validFinanceDate(from)) && (to == "" || validFinanceDate(to)) && (from == "" || to == "" || from <= to)
}

// CalculateOverdueDays 计算应收账单的逾期天数。只有已确认、存在未核销余额且到期日小于当前业务日期的应收账单才计算逾期天数。
func CalculateOverdueDays(direction OrderFeeDirection, status FinanceBillStatus, unverifiedAmount decimal.Decimal, dueDate *string, businessDate string) int32 {
	if direction != OrderFeeReceivable || status != FinanceBillConfirmed || !unverifiedAmount.IsPositive() || dueDate == nil || strings.TrimSpace(*dueDate) == "" {
		return 0
	}
	trimmedDueDate := strings.TrimSpace(*dueDate)
	if trimmedDueDate >= businessDate {
		return 0
	}
	dueT, err1 := time.Parse("2006-01-02", trimmedDueDate)
	bizT, err2 := time.Parse("2006-01-02", businessDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	days := int32(bizT.Sub(dueT).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func normalizedOptionalFinanceString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func financeBillAudit(organizationID, actorID, billID uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "finance_bill", ResourceID: billID.String(), Details: map[string]string{"finance_bill.id": billID.String()}}
}
