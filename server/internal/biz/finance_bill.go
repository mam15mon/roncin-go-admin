package biz

import (
	"context"
	"encoding/json"
	"log/slog"
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
	ErrFinanceBillFeeInvalid               = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_FEE_INVALID), "所选费用必须为未建账状态且尚未进入其他账单")
	ErrFinanceBillFeeMismatch              = errors.BadRequest("FINANCE_BILL_FEE_MISMATCH", "同一账单的费用必须具有相同收付方向、结算单位、币种和本币")
	ErrFinanceBillVersionConflict          = errors.Conflict("FINANCE_BILL_VERSION_CONFLICT", "账单已被其他操作人修改，请刷新后重试")
	ErrFinanceBillInvalidTransition        = errors.Conflict("FINANCE_BILL_INVALID_TRANSITION", "当前账单状态不允许执行该操作")
	ErrFinanceBillIdempotencyConflict      = errors.Conflict("FINANCE_BILL_IDEMPOTENCY_CONFLICT", "账单请求幂等键已被其他请求使用")
	ErrFinanceBillPreviewStale             = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_PREVIEW_STALE), "费用或拆单结果已变化，请重新预览")
	ErrFinanceBillBatchMismatch            = errors.BadRequest("FINANCE_BILL_BATCH_MISMATCH", "批量账单分组资料与服务端预览不一致")
	ErrFinanceBillBatchConflict            = errors.Conflict("FINANCE_BILL_BATCH_CONFLICT", "批量建单幂等键已被其他请求使用")
	ErrFinanceBillSettlementAccountInvalid = errors.BadRequest(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_SETTLEMENT_ACCOUNT_INVALID), "结算账户与账单结算单位、方向、币种或启用状态不匹配")
	ErrFinanceBillGroupingModeUnsupported  = errors.BadRequest("FINANCE_BILL_GROUPING_MODE_UNSUPPORTED", "当前阶段暂不支持对冲建账模式")
	// ErrFinanceBillProtectedFields 表示账单已有有效发票、核销或已确认对冲，只能修改不影响金额与结算的字段。
	ErrFinanceBillProtectedFields = errors.Conflict("FINANCE_BILL_PROTECTED_FIELDS", "账单已登记发票、核销或对冲，只能修改备注、对账标题和到期日；如需调整账户、账期或预计开票，请先撤销相关财务记录")
)

var financeBillCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type FinanceBillUsecase struct {
	repo         FinanceBillRepo
	exchangeRate *ExchangeRateUsecase
	transactor   Transactor
	logger       *slog.Logger
	now          func() time.Time
}

func NewFinanceBillUsecase(repo FinanceBillRepo, exchangeRate *ExchangeRateUsecase, transactor Transactor, logger *slog.Logger) *FinanceBillUsecase {
	return &FinanceBillUsecase{repo: repo, exchangeRate: exchangeRate, transactor: transactor, logger: logger, now: time.Now}
}

// businessDate 以服务端北京时间确定业务创建日；不读取客户端日期或主机本地时区。
func (uc *FinanceBillUsecase) businessDate() string {
	now := time.Now
	if uc.now != nil {
		now = uc.now
	}
	return FinanceBusinessDate(now())
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
	if filter.SettlementProgress != "" && !ValidFinanceBillSettlementProgress(filter.SettlementProgress) {
		return nil, ErrFinanceBillInvalidArgument
	}
	if filter.InvoiceProgress != "" && !ValidFinanceBillInvoiceProgress(filter.InvoiceProgress) {
		return nil, ErrFinanceBillInvalidArgument
	}
	for _, status := range filter.BusinessStatuses {
		if !ValidFinanceBillBusinessStatus(status) {
			return nil, ErrFinanceBillInvalidArgument
		}
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
	if organizationID == uuid.Nil || actorID == uuid.Nil || uc.transactor == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	// 幂等摘要只含客户明确提交的意图；服务器日期、默认账期与自动到期日不参与。
	normalized, err := normalizeFinanceBillCreateIntent(input)
	if err != nil {
		return nil, err
	}
	requestHash := financeBillCreateRequestHash(normalized)
	existing, err := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.CreateRequestHash != requestHash {
			return nil, ErrFinanceBillIdempotencyConflict
		}
		return uc.repo.Get(ctx, []uuid.UUID{organizationID}, existing.ID)
	}
	var created *FinanceBill
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 同键并发在进入新建准入前先复核；重放不依赖当前费用资格和今日汇率。
		existing, transactionErr := uc.repo.GetByIdempotencyKey(txCtx, organizationID, normalized.IdempotencyKey)
		if transactionErr != nil {
			return transactionErr
		}
		if existing != nil {
			if existing.CreateRequestHash != requestHash {
				return ErrFinanceBillIdempotencyConflict
			}
			created = existing
			return nil
		}
		fees, transactionErr := uc.repo.LoadBillableFees(txCtx, organizationID, normalized.FeeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		normalized.BillDate = uc.businessDate()
		normalized, transactionErr = normalizeCreateFinanceBill(normalized)
		if transactionErr != nil {
			return transactionErr
		}
		if normalized.PaymentTermsDays == nil && normalized.DueDate == nil && len(fees) > 0 {
			injected, injectErr := uc.defaultFinanceBillPaymentTermsDays(txCtx, organizationID, fees[0])
			if injectErr != nil {
				return injectErr
			}
			if injected != nil {
				normalized.PaymentTermsDays = injected
				normalized.DueDate = normalizedFinanceBillDueDate(normalized.BillDate, nil, injected)
			}
		}
		bill, transactionErr := buildFinanceBill(organizationID, fees, normalized)
		if transactionErr != nil {
			return transactionErr
		}
		bill.ConfirmedBy, bill.CreateRequestHash = &actorID, requestHash
		if transactionErr = uc.repo.HydrateBillSettlementAccounts(txCtx, []*FinanceBill{bill}); transactionErr != nil {
			return transactionErr
		}
		if transactionErr = uc.repo.ValidateBillCurrencies(txCtx, []string{bill.Currency, bill.BaseCurrency}); transactionErr != nil {
			return transactionErr
		}
		if transactionErr = uc.applyBillExchangeRate(txCtx, organizationID, bill); transactionErr != nil {
			return transactionErr
		}
		audit := financeBillAudit(organizationID, actorID, bill.ID, "finance.bill.create")
		audit.Details["create_request_hash"] = requestHash
		created, transactionErr = uc.repo.Create(txCtx, bill, audit)
		return transactionErr
	})
	if err == nil {
		return uc.repo.Get(ctx, []uuid.UUID{organizationID}, created.ID)
	}
	existing, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey)
	if lookupErr == nil && existing != nil && existing.CreateRequestHash == requestHash {
		return uc.repo.Get(ctx, []uuid.UUID{organizationID}, existing.ID)
	}
	return nil, err
}

// 原始客户意图在创建审计中固化，后续增减费用和修改备注不会改变该摘要。
func financeBillCreateRequestHash(input CreateFinanceBillInput) string {
	payload, _ := json.Marshal(struct {
		FeeIDs                        []uuid.UUID
		DueDate, Note, StatementTitle *string
		PaymentTermsDays              *int
		SettlementAccountID           uuid.UUID
	}{input.FeeIDs, input.DueDate, input.Note, input.StatementTitle, input.PaymentTermsDays, input.SettlementAccountID})
	return financeSHA256(string(payload))
}

func normalizeFinanceBillCreateIntent(input CreateFinanceBillInput) (CreateFinanceBillInput, error) {
	input.BillDate = ""
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || input.SettlementAccountID == uuid.Nil || (input.DueDate != nil && !validFinanceDate(*input.DueDate)) || (input.PaymentTermsDays != nil && (*input.PaymentTermsDays < 0 || *input.PaymentTermsDays > 3650)) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
		return CreateFinanceBillInput{}, ErrFinanceBillInvalidArgument
	}
	feeIDs, err := normalizeFinanceBillFeeIDs(input.FeeIDs)
	if err != nil {
		return CreateFinanceBillInput{}, err
	}
	input.FeeIDs = feeIDs
	return input, nil
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
	bill.ExchangeRate = resolved.Rate.RoundBank(4)
	bill.ExchangeRateSource = resolved.Source
	bill.ExchangeRateDate = bill.BillDate
	bill.ExchangeRateSettingID = resolved.SettingID
	// 头本位币金额必须使用已固化（舍入到 4 位）的账单汇率，与批量内核口径一致。
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
	input.BillDate = ""
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || input.ID == uuid.Nil || input.ExpectedVersion == 0 || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, input.ID)
	if err != nil {
		return nil, err
	}
	if existing.Status != FinanceBillConfirmed || existing.Version != input.ExpectedVersion || uc.transactor == nil {
		if existing.Status != FinanceBillConfirmed {
			return nil, ErrFinanceBillInvalidTransition
		}
		if existing.Version != input.ExpectedVersion {
			return nil, ErrFinanceBillVersionConflict
		}
		return nil, ErrFinanceBillInvalidArgument
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 账单更新与取消统一先锁订单与账单；更新随后再按 ID 锁费用，避免与
		// “账单 → 费用”的状态操作形成反向等待。
		current, transactionErr := uc.repo.Get(txCtx, organizationIDs, input.ID)
		if transactionErr != nil {
			return transactionErr
		}
		if current.Status != FinanceBillConfirmed {
			return ErrFinanceBillInvalidTransition
		}
		if current.Version != input.ExpectedVersion {
			return ErrFinanceBillVersionConflict
		}
		input.BillDate = current.BillDate
		input.DueDate = normalizedFinanceBillDueDate(current.BillDate, input.DueDate, input.PaymentTermsDays)
		if !validFinanceBillTerms(current.BillDate, input.DueDate, input.PaymentTermsDays) {
			return ErrFinanceBillInvalidArgument
		}
		// 受保护判定在订单与账单行锁内读取；发票、核销、对冲写路径同样先锁订单再锁账单。
		protected, transactionErr := uc.repo.HasProtectedFinanceFact(txCtx, current.ID)
		if transactionErr != nil {
			return transactionErr
		}
		if protected {
			if !financeBillProtectedFieldsUnchanged(current, input) {
				return ErrFinanceBillProtectedFields
			}
			// 受保护账单不重取汇率、不重算金额，只写回备注、对账标题与到期日。
			input.Protected = true
			input.ExchangeRate, input.ExchangeRateSource, input.ExchangeRateDate, input.ExchangeRateSettingID = current.ExchangeRate, current.ExchangeRateSource, current.ExchangeRateDate, current.ExchangeRateSettingID
			input.BaseCurrencyAmount, input.TotalAmount, input.NetAmount, input.TaxAmount, input.Lines = current.BaseCurrencyAmount, current.TotalAmount, current.NetAmount, current.TaxAmount, current.Lines
			input.EstimatedInvoiceCurrency, input.EstimatedInvoiceRate, input.EstimatedInvoiceAmount = current.EstimatedInvoiceCurrency, current.EstimatedInvoiceRate, current.EstimatedInvoiceAmount
			_, transactionErr = uc.repo.Update(txCtx, organizationIDs, input, financeBillAudit(current.OrganizationID, actorID, input.ID, "finance.bill.update"))
			return transactionErr
		}
		// 详情保留已移除行；账单编辑只重算当前活动行，历史行及已释放/删除费用不得再次参与。
		feeIDs := make([]uuid.UUID, 0, len(current.Lines))
		for _, line := range current.Lines {
			if line.Active {
				feeIDs = append(feeIDs, line.OrderFeeID)
			}
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
		// 编辑仅沿用锁内快照，活动行保持读取顺序，避免改变末行尾差归属。
		rebuilt := *current
		activeLines := make([]*FinanceBillLine, 0, len(current.Lines))
		for _, line := range current.Lines {
			if line.Active {
				copied := *line
				activeLines = append(activeLines, &copied)
			}
		}
		if input.EstimatedInvoiceRate != nil && input.EstimatedInvoiceCurrency == nil {
			return ErrFinanceBillInvalidArgument
		}
		currency, rate, _, transactionErr := financeBillEstimatedInvoiceSnapshot(current.Currency, current.TotalAmount, FinanceBillBatchPreviewGroupConfig{EstimatedInvoiceCurrency: input.EstimatedInvoiceCurrency, EstimatedInvoiceRate: input.EstimatedInvoiceRate})
		if transactionErr != nil {
			return transactionErr
		}
		rebuilt.EstimatedInvoiceCurrency, rebuilt.EstimatedInvoiceRate = &currency, &rate
		if transactionErr = RecomputeFinanceBillAmounts(&rebuilt, activeLines); transactionErr != nil {
			return transactionErr
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
	if bill == nil {
		return ErrFinanceBillPreviewStale
	}
	lines := make(map[uuid.UUID]*FinanceBillLine, len(bill.Lines))
	for _, line := range bill.Lines {
		if line.Active {
			lines[line.OrderFeeID] = line
		}
	}
	if len(fees) != len(lines) {
		return ErrFinanceBillPreviewStale
	}
	for _, item := range fees {
		fee, line := item.Fee, lines[item.Fee.ID]
		if line == nil || !fee.HasActiveBill || SettlementDirection(fee) != bill.Direction || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != line.Currency || !fee.TotalAmount.Equal(line.TotalAmount) || !fee.NetAmount.Equal(line.NetAmount) || !fee.TaxAmount.Equal(line.TaxAmount) {
			return ErrFinanceBillPreviewStale
		}
	}
	return nil
}

// financeBillProtectedFieldsUnchanged 判断受保护账单的请求是否只改动备注、对账标题与到期日。
// 账单日期决定账单汇率；账户、账期与预计开票属于结算口径，均须撤销财务记录后才能调整。
// 预计开票按与建账相同的默认规则归一后再与已存快照比对。
func financeBillProtectedFieldsUnchanged(current *FinanceBill, input UpdateFinanceBillInput) bool {
	if current.SettlementAccountID != input.SettlementAccountID || !intPointersEqual(current.PaymentTermsDays, input.PaymentTermsDays) {
		return false
	}
	if current.EstimatedInvoiceCurrency == nil || current.EstimatedInvoiceRate == nil {
		return input.EstimatedInvoiceCurrency == nil && input.EstimatedInvoiceRate == nil
	}
	currency, rate, _, err := financeBillEstimatedInvoiceSnapshot(current.Currency, current.TotalAmount, FinanceBillBatchPreviewGroupConfig{EstimatedInvoiceCurrency: input.EstimatedInvoiceCurrency, EstimatedInvoiceRate: input.EstimatedInvoiceRate})
	if err != nil {
		return false
	}
	return currency == *current.EstimatedInvoiceCurrency && rate.Equal(*current.EstimatedInvoiceRate)
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
	if first == nil || first.Fee == nil || first.Fee.HasActiveBill {
		return nil, ErrFinanceBillFeeInvalid
	}
	var correctionID *uuid.UUID
	if first.Fee.SourceFeeID != nil {
		correctionID = first.Fee.CorrectionRequestID
	}
	billID := uuid.Must(uuid.NewV7())
	bill := &FinanceBill{
		ID: billID, OrganizationID: organizationID, IdempotencyKey: input.IdempotencyKey,
		CorrectionRequestID: correctionID, Purpose: FinanceBillPurposeForFee(first.Fee), Direction: SettlementDirection(first.Fee), Status: FinanceBillConfirmed, SettlementPartyID: first.Fee.SettlementPartyID,
		SettlementPartyName: first.Fee.SettlementPartyName, SettlementAccountID: input.SettlementAccountID, Currency: first.Fee.Currency, BaseCurrency: first.Fee.BaseCurrency,
		BillDate: input.BillDate, StatementTitle: input.StatementTitle, PaymentTermsDays: input.PaymentTermsDays, DueDate: input.DueDate, Note: input.Note, Version: 1,
		Lines: make([]*FinanceBillLine, 0, len(fees)),
	}
	if bill.StatementTitle == nil {
		bill.StatementTitle = &bill.SettlementPartyName
	}
	for _, item := range fees {
		if err := validateBillFeeScope(bill, item); err != nil {
			return nil, err
		}
		fee := item.Fee
		bill.TotalAmount = bill.TotalAmount.Add(fee.TotalAmount)
		bill.NetAmount = bill.NetAmount.Add(fee.NetAmount)
		bill.TaxAmount = bill.TaxAmount.Add(fee.TaxAmount)
		bill.BaseCurrencyAmount = bill.BaseCurrencyAmount.Add(fee.BaseCurrencyAmount)
		bill.Lines = append(bill.Lines, NewFinanceBillLineFromFee(billID, item))
	}
	bill.FeeCount = len(bill.Lines)
	return bill, nil
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

// CalculateOverdueDays 计算应收账单的逾期天数。只有已确认、存在正数未核销余额且到期日小于当前业务日期的应收账单才计算逾期天数。
// unverifiedAmount 为带符号余额（总额 − 有效核销 − 有效对冲，不钳零）：零额账单、负数更正账单及
// 余额为零或负的账单一律返回 0；SQL 侧与之同源的是 billOverduePredicate（正数余额谓词）。
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
