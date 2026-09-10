package biz

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFinanceNettingNotFound            = errors.NotFound("FINANCE_NETTING_NOT_FOUND", "对冲单不存在")
	ErrFinanceNettingInvalid             = errors.BadRequest("FINANCE_NETTING_INVALID", "对冲参数不合法")
	ErrFinanceNettingMismatch            = errors.BadRequest("FINANCE_NETTING_MISMATCH", "对冲账单必须属于同一组织、同一结算单位并具有相同账单币种，或不在当前权限范围内")
	ErrFinanceNettingDirection           = errors.BadRequest("FINANCE_NETTING_DIRECTION", "对冲需至少包含一张应收账单和一张应付账单")
	ErrFinanceNettingBalance             = errors.Conflict("FINANCE_NETTING_BALANCE", "账单可用余额不足以完成本次对冲")
	ErrFinanceNettingTransition          = errors.Conflict("FINANCE_NETTING_TRANSITION", "当前对冲状态不允许执行该操作")
	ErrFinanceNettingBillNotConfirmed    = errors.Conflict("FINANCE_NETTING_BILL_NOT_CONFIRMED", "来源账单必须均为已确认状态方可确认对冲单")
	ErrFinanceNettingVersionConflict     = errors.Conflict("FINANCE_NETTING_VERSION_CONFLICT", "对冲单已被其他操作人修改，请刷新后重试")
	ErrFinanceNettingBillVersionConflict = errors.Conflict("FINANCE_NETTING_BILL_VERSION_CONFLICT", "来源账单已被其他操作人修改，请刷新后重试")
	ErrFinanceNettingIdempotency         = errors.Conflict("FINANCE_NETTING_IDEMPOTENCY", "对冲请求幂等键已被其他请求使用")
	ErrFinanceNettingSingleDirection     = errors.BadRequest("FINANCE_NETTING_SINGLE_DIRECTION", "对冲建账需在同一结算单位、同一账单币种下同时包含应收和应付费用，单方向费用请使用普通账单")
	ErrFinanceNettingTooManyBills        = errors.BadRequest("FINANCE_NETTING_TOO_MANY_BILLS", "同一结算单位与账单币种下的候选账单数量超过上限，请缩小范围后重试")
)

// MaxFinanceNettingBills 是单次对冲预览与计划允许参与的最大账单数量，
// 预览加载与计划校验共用同一上限，避免预览静默截断。
const MaxFinanceNettingBills = 500

type FinanceNettingStatus string

const (
	FinanceNettingDraft     FinanceNettingStatus = "DRAFT"
	FinanceNettingConfirmed FinanceNettingStatus = "CONFIRMED"
	FinanceNettingCancelled FinanceNettingStatus = "CANCELLED"
	FinanceNettingReversed  FinanceNettingStatus = "REVERSED"
)

// FinanceNettingAllocation 是对冲确认后在原始账单上形成的抵销分摊；只有 active 分摊参与可用余额。
type FinanceNettingAllocation struct {
	ID                 uuid.UUID
	NettingID          uuid.UUID
	BillID             uuid.UUID
	BillNo             string
	Direction          OrderFeeDirection
	Amount             decimal.Decimal
	BaseCurrencyAmount decimal.Decimal
	Active             bool
}

// FinanceNetting 只保存双方共同账单币种的抵销金额，不保存对冲汇率，也不产生混合币种总额。
type FinanceNetting struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	OrganizationName     string
	BatchID              *uuid.UUID
	BatchNo              string
	NettingNo            string
	IdempotencyKey       string
	RequestHash          string
	Status               FinanceNettingStatus
	SettlementPartyID    uuid.UUID
	SettlementPartyName  string
	Currency             string
	Amount               decimal.Decimal
	BaseCurrency         string
	BaseCurrencyAmount   decimal.Decimal
	Note                 *string
	Version              uint64
	ConfirmedAt          *time.Time
	ConfirmedBy          *uuid.UUID
	CancelledAt          *time.Time
	CancelledBy          *uuid.UUID
	CancellationReason   *string
	ReversedAt           *time.Time
	ReversedBy           *uuid.UUID
	ReversalReason       *string
	Allocations          []*FinanceNettingAllocation
	CreatedAt, UpdatedAt time.Time
}

// FinanceNettingBill 是锁内读取的账单余额事实；可用余额 = 总额 - 有效核销 - 有效对冲。
type FinanceNettingBill struct {
	ID                  uuid.UUID
	BillNo              string
	BillDate            string
	Status              FinanceBillStatus
	Direction           OrderFeeDirection
	SettlementPartyID   uuid.UUID
	SettlementPartyName string
	Currency            string
	BaseCurrency        string
	ExchangeRate        decimal.Decimal
	TotalAmount         decimal.Decimal
	VerifiedAmount      decimal.Decimal
	NettedAmount        decimal.Decimal
	Version             uint64
}

func (b *FinanceNettingBill) AvailableAmount() decimal.Decimal {
	available := b.TotalAmount.Sub(b.VerifiedAmount).Sub(b.NettedAmount)
	if available.IsNegative() {
		return decimal.Zero
	}
	return available.Round(8)
}

type FinanceNettingBillBalance struct {
	BillID                       uuid.UUID
	BillNo, BillDate             string
	TotalAmount                  decimal.Decimal
	VerifiedAmount, NettedAmount decimal.Decimal
	AvailableAmount              decimal.Decimal
	Version                      uint64
}

type FinanceNettingPreview struct {
	OrganizationID, SettlementPartyID     uuid.UUID
	OrganizationName, SettlementPartyName string
	Currency                              string
	ReceivableBills, PayableBills         []*FinanceNettingBillBalance
	ReceivableAvailableAmount             decimal.Decimal
	PayableAvailableAmount                decimal.Decimal
	OffsetAmount                          decimal.Decimal
	NetReceivableAmount                   decimal.Decimal
	NetPayableAmount                      decimal.Decimal
}

type FinanceNettingBillVersion struct {
	BillID          uuid.UUID
	ExpectedVersion uint64
}

type CreateFinanceNettingInput struct {
	Bills          []FinanceNettingBillVersion
	Note           *string
	IdempotencyKey string
}

type FinanceNettingFilter struct {
	Page, PageSize    int
	Keyword           string
	Status            FinanceNettingStatus
	SettlementPartyID *uuid.UUID
	Currency          string
}

type FinanceNettingListResult struct {
	Items   []*FinanceNetting
	Total   int64
	Summary FinanceNettingSummary
}

type FinanceNettingSummary struct {
	AmountsByBaseCurrency []FinanceNettingBaseCurrencyAmount
	ConfirmedCount        int64
}

type FinanceNettingBaseCurrencyAmount struct {
	BaseCurrency      string
	NettingBaseAmount decimal.Decimal
}

type FinanceNettingRepo interface {
	List(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceNettingFilter) (*FinanceNettingListResult, error)
	Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceNetting, error)
	GetByKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceNetting, error)
	LoadPreview(ctx context.Context, organizationID, settlementPartyID uuid.UUID, currency string) (*FinanceNettingPreview, error)
	// LockNettingBills 在共享事务内按账单主键稳定排序加锁并计算核销与对冲余额。
	LockNettingBills(ctx context.Context, organizationID uuid.UUID, billIDs []uuid.UUID) ([]*FinanceNettingBill, error)
	// LockNetting 在共享事务内锁定对冲单本体（含分摊）。
	LockNetting(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceNetting, error)
	Create(ctx context.Context, organizationID, actorID uuid.UUID, netting *FinanceNetting, audit *AuditEvent) (*FinanceNetting, error)
	Confirm(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, audit *AuditEvent) (*FinanceNetting, error)
	Cancel(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) (*FinanceNetting, error)
	Reverse(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) (*FinanceNetting, error)
}

type FinanceNettingUsecase struct {
	repo       FinanceNettingRepo
	transactor Transactor
}

func NewFinanceNettingUsecase(repo FinanceNettingRepo, transactor Transactor) *FinanceNettingUsecase {
	return &FinanceNettingUsecase{repo: repo, transactor: transactor}
}

func (uc *FinanceNettingUsecase) List(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceNettingFilter) (*FinanceNettingListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if !validFinanceBillOrganizationIDs(organizationIDs) || !ValidListPagination(filter.Page, filter.PageSize) || utf8.RuneCountInString(filter.Keyword) > 100 {
		return nil, ErrFinanceNettingInvalid
	}
	if filter.Status != "" && filter.Status != FinanceNettingDraft && filter.Status != FinanceNettingConfirmed && filter.Status != FinanceNettingCancelled && filter.Status != FinanceNettingReversed {
		return nil, ErrFinanceNettingInvalid
	}
	if filter.Currency != "" && !financeBillCurrencyPattern.MatchString(filter.Currency) {
		return nil, ErrFinanceNettingInvalid
	}
	return uc.repo.List(ctx, organizationIDs, filter)
}

func (uc *FinanceNettingUsecase) Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceNetting, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrFinanceNettingInvalid
	}
	return uc.repo.Get(ctx, organizationIDs, id)
}

// Preview 按组织、结算单位和账单币种分支计算双方未结余额与较小值；金额只用共同账单币种。
func (uc *FinanceNettingUsecase) Preview(ctx context.Context, organizationID, settlementPartyID uuid.UUID, currency string) (*FinanceNettingPreview, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || settlementPartyID == uuid.Nil || !financeBillCurrencyPattern.MatchString(currency) {
		return nil, ErrFinanceNettingInvalid
	}
	preview, err := uc.repo.LoadPreview(ctx, organizationID, settlementPartyID, currency)
	if err != nil {
		return nil, err
	}
	ApplyFinanceNettingPreviewAmounts(preview)
	return preview, nil
}

// ApplyFinanceNettingPreviewAmounts 按双方可用余额计算抵销额与净应收/净应付。
func ApplyFinanceNettingPreviewAmounts(preview *FinanceNettingPreview) {
	if preview == nil {
		return
	}
	preview.ReceivableAvailableAmount = decimal.Zero
	preview.PayableAvailableAmount = decimal.Zero
	for _, bill := range preview.ReceivableBills {
		preview.ReceivableAvailableAmount = preview.ReceivableAvailableAmount.Add(bill.AvailableAmount)
	}
	for _, bill := range preview.PayableBills {
		preview.PayableAvailableAmount = preview.PayableAvailableAmount.Add(bill.AvailableAmount)
	}
	preview.OffsetAmount = decimal.Zero
	// 单边无可用余额时不发生抵销，净额保持该边可用余额全额。
	preview.NetReceivableAmount = preview.ReceivableAvailableAmount
	preview.NetPayableAmount = preview.PayableAvailableAmount
	if preview.ReceivableAvailableAmount.IsPositive() && preview.PayableAvailableAmount.IsPositive() {
		preview.OffsetAmount = decimal.Min(preview.ReceivableAvailableAmount, preview.PayableAvailableAmount).Round(8)
		preview.NetReceivableAmount = preview.ReceivableAvailableAmount.Sub(preview.OffsetAmount).Round(8)
		preview.NetPayableAmount = preview.PayableAvailableAmount.Sub(preview.OffsetAmount).Round(8)
	}
}

type financeNettingPlan struct {
	SettlementPartyID   uuid.UUID
	SettlementPartyName string
	Currency            string
	BaseCurrency        string
	Amount              decimal.Decimal
	BaseCurrencyAmount  decimal.Decimal
	Allocations         []*FinanceNettingAllocation
}

// PlanFinanceNetting 校验锁内账单事实并形成对冲计划：对冲金额为双方可用余额较小值，
// 分摊按账单主键稳定顺序贪心填充，双方各自恰好分配到对冲金额。
// 校验同组织（由锁内查询范围保证数量一致）、同结算单位、相反方向、相同最终账单币种、
// 状态有效、版本匹配与可用余额充足。
func PlanFinanceNetting(bills []*FinanceNettingBill, expectedVersions []FinanceNettingBillVersion) (*financeNettingPlan, error) {
	if len(bills) == 0 || len(bills) != len(expectedVersions) || len(bills) > MaxFinanceNettingBills {
		return nil, ErrFinanceNettingMismatch
	}
	expected := make(map[uuid.UUID]uint64, len(expectedVersions))
	for _, item := range expectedVersions {
		if item.BillID == uuid.Nil || item.ExpectedVersion == 0 {
			return nil, ErrFinanceNettingInvalid
		}
		if _, duplicate := expected[item.BillID]; duplicate {
			return nil, ErrFinanceNettingInvalid
		}
		expected[item.BillID] = item.ExpectedVersion
	}
	seen := make(map[uuid.UUID]struct{}, len(bills))
	receivableTotal, payableTotal := decimal.Zero, decimal.Zero
	hasReceivable, hasPayable := false, false
	plan := &financeNettingPlan{}
	for _, bill := range bills {
		if bill == nil || bill.ID == uuid.Nil {
			return nil, ErrFinanceNettingMismatch
		}
		if _, duplicate := seen[bill.ID]; duplicate {
			return nil, ErrFinanceNettingMismatch
		}
		seen[bill.ID] = struct{}{}
		if bill.Status != FinanceBillConfirmed {
			return nil, ErrFinanceNettingTransition
		}
		if bill.Version != expected[bill.ID] {
			return nil, ErrFinanceNettingBillVersionConflict
		}
		if (bill.Direction != OrderFeeReceivable && bill.Direction != OrderFeePayable) || !financeBillCurrencyPattern.MatchString(bill.Currency) || !bill.ExchangeRate.IsPositive() || !bill.TotalAmount.IsPositive() {
			return nil, ErrFinanceNettingInvalid
		}
		if plan.SettlementPartyID == uuid.Nil {
			plan.SettlementPartyID = bill.SettlementPartyID
			plan.SettlementPartyName = bill.SettlementPartyName
			plan.Currency = bill.Currency
			plan.BaseCurrency = bill.BaseCurrency
		} else if bill.SettlementPartyID != plan.SettlementPartyID || bill.Currency != plan.Currency || bill.BaseCurrency != plan.BaseCurrency {
			return nil, ErrFinanceNettingMismatch
		}
		if bill.Direction == OrderFeeReceivable {
			receivableTotal = receivableTotal.Add(bill.AvailableAmount())
			hasReceivable = true
		} else {
			payableTotal = payableTotal.Add(bill.AvailableAmount())
			hasPayable = true
		}
	}
	if !hasReceivable || !hasPayable {
		return nil, ErrFinanceNettingDirection
	}
	if receivableTotal.IsZero() || payableTotal.IsZero() {
		return nil, ErrFinanceNettingBalance
	}
	plan.Amount = decimal.Min(receivableTotal, payableTotal).Round(8)
	if !plan.Amount.IsPositive() {
		return nil, ErrFinanceNettingBalance
	}
	sorted := append([]*FinanceNettingBill(nil), bills...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID.String() < sorted[j].ID.String() })
	for _, direction := range []OrderFeeDirection{OrderFeeReceivable, OrderFeePayable} {
		remaining := plan.Amount
		for _, bill := range sorted {
			if bill.Direction != direction || !remaining.IsPositive() {
				continue
			}
			allocated := decimal.Min(remaining, bill.AvailableAmount())
			if !allocated.IsPositive() {
				continue
			}
			plan.Allocations = append(plan.Allocations, &FinanceNettingAllocation{
				BillID: bill.ID, BillNo: bill.BillNo, Direction: direction,
				Amount: allocated, BaseCurrencyAmount: allocated.Mul(bill.ExchangeRate).RoundBank(8),
			})
			remaining = remaining.Sub(allocated)
		}
		if remaining.IsPositive() {
			return nil, ErrFinanceNettingBalance
		}
	}
	for _, allocation := range plan.Allocations {
		if allocation.Direction == OrderFeeReceivable {
			plan.BaseCurrencyAmount = plan.BaseCurrencyAmount.Add(allocation.BaseCurrencyAmount)
		}
	}
	plan.BaseCurrencyAmount = plan.BaseCurrencyAmount.RoundBank(8)
	return plan, nil
}

// ValidateFinanceNettingConfirmation 在确认事务内复核每条分摊：账单有效、同单位同币种且可用余额充足。
func ValidateFinanceNettingConfirmation(netting *FinanceNetting, bills []*FinanceNettingBill) error {
	if netting == nil || len(netting.Allocations) == 0 || len(bills) == 0 {
		return ErrFinanceNettingInvalid
	}
	billByID := make(map[uuid.UUID]*FinanceNettingBill, len(bills))
	for _, bill := range bills {
		if bill == nil {
			return ErrFinanceNettingInvalid
		}
		billByID[bill.ID] = bill
	}
	receivableSum, payableSum := decimal.Zero, decimal.Zero
	for _, allocation := range netting.Allocations {
		bill := billByID[allocation.BillID]
		if bill == nil || bill.SettlementPartyID != netting.SettlementPartyID || bill.Currency != netting.Currency {
			return ErrFinanceNettingTransition
		}
		if bill.Status != FinanceBillConfirmed {
			return ErrFinanceNettingBillNotConfirmed
		}
		if allocation.Direction != bill.Direction {
			return ErrFinanceNettingInvalid
		}
		if bill.AvailableAmount().LessThan(allocation.Amount) {
			return ErrFinanceNettingBalance
		}
		if allocation.Direction == OrderFeeReceivable {
			receivableSum = receivableSum.Add(allocation.Amount)
		} else {
			payableSum = payableSum.Add(allocation.Amount)
		}
	}
	if !receivableSum.Equal(netting.Amount) || !payableSum.Equal(netting.Amount) {
		return ErrFinanceNettingInvalid
	}
	return nil
}

func (uc *FinanceNettingUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceNettingInput) (*FinanceNetting, error) {
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.Note = normalizedOptionalFinanceString(input.Note)
	if organizationID == uuid.Nil || actorID == uuid.Nil || len(input.Bills) < 2 || len(input.Bills) > 500 || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) {
		return nil, ErrFinanceNettingInvalid
	}
	sort.Slice(input.Bills, func(i, j int) bool { return input.Bills[i].BillID.String() < input.Bills[j].BillID.String() })
	billIDs := make([]uuid.UUID, 0, len(input.Bills))
	for _, item := range input.Bills {
		if item.BillID == uuid.Nil {
			return nil, ErrFinanceNettingInvalid
		}
		billIDs = append(billIDs, item.BillID)
	}
	requestHash := financeNettingRequestHash(input)
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, transactionErr := uc.repo.GetByKey(txCtx, organizationID, input.IdempotencyKey)
		if transactionErr != nil {
			return transactionErr
		}
		if existing != nil {
			if existing.RequestHash != requestHash {
				return ErrFinanceNettingIdempotency
			}
			return nil
		}
		bills, transactionErr := uc.repo.LockNettingBills(txCtx, organizationID, billIDs)
		if transactionErr != nil {
			return transactionErr
		}
		plan, planErr := PlanFinanceNetting(bills, input.Bills)
		if planErr != nil {
			return planErr
		}
		id := uuid.Must(uuid.NewV7())
		netting := &FinanceNetting{
			ID: id, OrganizationID: organizationID, IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash,
			Status: FinanceNettingDraft, SettlementPartyID: plan.SettlementPartyID, SettlementPartyName: plan.SettlementPartyName,
			Currency: plan.Currency, Amount: plan.Amount, BaseCurrency: plan.BaseCurrency, BaseCurrencyAmount: plan.BaseCurrencyAmount,
			Note: input.Note, Version: 1,
			Allocations: make([]*FinanceNettingAllocation, 0, len(plan.Allocations)),
		}
		for _, allocation := range plan.Allocations {
			allocation.ID = uuid.Must(uuid.NewV7())
			allocation.NettingID = id
			netting.Allocations = append(netting.Allocations, allocation)
		}
		_, transactionErr = uc.repo.Create(txCtx, organizationID, actorID, netting, financeNettingAudit(organizationID, actorID, id, "finance.netting.create"))
		return transactionErr
	})
	if err == nil {
		return uc.repo.GetByKey(ctx, organizationID, input.IdempotencyKey)
	}
	if existing, lookupErr := uc.repo.GetByKey(ctx, organizationID, input.IdempotencyKey); lookupErr == nil && existing != nil && existing.RequestHash == requestHash {
		return existing, nil
	}
	return nil, err
}

func (uc *FinanceNettingUsecase) Confirm(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64) (*FinanceNetting, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFinanceNettingInvalid
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		netting, transactionErr := uc.repo.LockNetting(txCtx, organizationIDs, id)
		if transactionErr != nil {
			return transactionErr
		}
		if netting.Version != expectedVersion {
			return ErrFinanceNettingVersionConflict
		}
		if netting.Status != FinanceNettingDraft {
			return ErrFinanceNettingTransition
		}
		billIDs := make([]uuid.UUID, 0, len(netting.Allocations))
		for _, allocation := range netting.Allocations {
			billIDs = append(billIDs, allocation.BillID)
		}
		bills, transactionErr := uc.repo.LockNettingBills(txCtx, existing.OrganizationID, billIDs)
		if transactionErr != nil {
			return transactionErr
		}
		if transactionErr = ValidateFinanceNettingConfirmation(netting, bills); transactionErr != nil {
			return transactionErr
		}
		_, transactionErr = uc.repo.Confirm(txCtx, existing.OrganizationID, actorID, id, expectedVersion, financeNettingAudit(existing.OrganizationID, actorID, id, "finance.netting.confirm"))
		return transactionErr
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, organizationIDs, id)
}

func (uc *FinanceNettingUsecase) Cancel(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceNetting, error) {
	reason = strings.TrimSpace(reason)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceNettingInvalid
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.Cancel(ctx, existing.OrganizationID, actorID, id, expectedVersion, reason, financeNettingAudit(existing.OrganizationID, actorID, id, "finance.netting.cancel"))
}

func (uc *FinanceNettingUsecase) Reverse(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceNetting, error) {
	reason = strings.TrimSpace(reason)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceNettingInvalid
	}
	existing, err := uc.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.Reverse(ctx, existing.OrganizationID, actorID, id, expectedVersion, reason, financeNettingAudit(existing.OrganizationID, actorID, id, "finance.netting.reverse"))
}

func financeNettingRequestHash(input CreateFinanceNettingInput) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, input.IdempotencyKey)
	note := ""
	if input.Note != nil {
		note = *input.Note
	}
	writeFinanceHashParts(&builder, note)
	for _, item := range input.Bills {
		writeFinanceHashParts(&builder, item.BillID.String(), strconv.FormatUint(item.ExpectedVersion, 10))
	}
	return financeSHA256(builder.String())
}

// planBatchFinanceNettings 为对冲建账批次按“结算单位 + 账单币种”生成草稿对冲结算单。
// 输入是同事务内新建的原始账单（未核销、未对冲），可用余额即账单总额；
// 抵销额为双方总额较小值，分摊按账单主键稳定顺序贪心填充，与独立创建口径一致。
func planBatchFinanceNettings(organizationID, batchID uuid.UUID, batchKey string, bills []*FinanceBill) ([]*FinanceNetting, error) {
	if organizationID == uuid.Nil || batchID == uuid.Nil || batchKey == "" || len(bills) == 0 {
		return nil, ErrFinanceNettingInvalid
	}
	type pairBills struct {
		settlementPartyID   uuid.UUID
		settlementPartyName string
		currency, base      string
		receivable, payable []*FinanceBill
	}
	pairs := make(map[string]*pairBills)
	keys := make([]string, 0)
	for _, bill := range bills {
		if bill == nil || bill.OrganizationID != organizationID || (bill.Direction != OrderFeeReceivable && bill.Direction != OrderFeePayable) ||
			!financeBillCurrencyPattern.MatchString(bill.Currency) || !bill.TotalAmount.IsPositive() || !bill.ExchangeRate.IsPositive() {
			return nil, ErrFinanceNettingInvalid
		}
		key := bill.SettlementPartyID.String() + "|" + bill.Currency
		pair := pairs[key]
		if pair == nil {
			pair = &pairBills{settlementPartyID: bill.SettlementPartyID, settlementPartyName: bill.SettlementPartyName, currency: bill.Currency, base: bill.BaseCurrency}
			pairs[key] = pair
			keys = append(keys, key)
		}
		if bill.BaseCurrency != pair.base {
			return nil, ErrFinanceNettingMismatch
		}
		if bill.Direction == OrderFeeReceivable {
			pair.receivable = append(pair.receivable, bill)
		} else {
			pair.payable = append(pair.payable, bill)
		}
	}
	sort.Strings(keys)
	result := make([]*FinanceNetting, 0, len(keys))
	for _, key := range keys {
		pair := pairs[key]
		if len(pair.receivable) == 0 || len(pair.payable) == 0 {
			return nil, ErrFinanceNettingSingleDirection
		}
		netting := &FinanceNetting{
			ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, BatchID: &batchID,
			IdempotencyKey: financeBillBatchNettingKey(batchKey, pair.settlementPartyID, pair.currency),
			Status:         FinanceNettingDraft, SettlementPartyID: pair.settlementPartyID, SettlementPartyName: pair.settlementPartyName,
			Currency: pair.currency, BaseCurrency: pair.base, Version: 1,
			Allocations: make([]*FinanceNettingAllocation, 0, len(pair.receivable)+len(pair.payable)),
		}
		receivableTotal, payableTotal := decimal.Zero, decimal.Zero
		for _, bill := range pair.receivable {
			receivableTotal = receivableTotal.Add(bill.TotalAmount)
		}
		for _, bill := range pair.payable {
			payableTotal = payableTotal.Add(bill.TotalAmount)
		}
		netting.Amount = decimal.Min(receivableTotal, payableTotal).Round(8)
		if !netting.Amount.IsPositive() {
			return nil, ErrFinanceNettingBalance
		}
		for _, direction := range []OrderFeeDirection{OrderFeeReceivable, OrderFeePayable} {
			var directionBills []*FinanceBill
			if direction == OrderFeeReceivable {
				directionBills = pair.receivable
			} else {
				directionBills = pair.payable
			}
			sorted := append([]*FinanceBill(nil), directionBills...)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID.String() < sorted[j].ID.String() })
			remaining := netting.Amount
			for _, bill := range sorted {
				if !remaining.IsPositive() {
					break
				}
				allocated := decimal.Min(remaining, bill.TotalAmount)
				if !allocated.IsPositive() {
					continue
				}
				netting.Allocations = append(netting.Allocations, &FinanceNettingAllocation{
					ID: uuid.Must(uuid.NewV7()), NettingID: netting.ID, BillID: bill.ID, BillNo: bill.BillNo,
					Direction: direction, Amount: allocated, BaseCurrencyAmount: allocated.Mul(bill.ExchangeRate).RoundBank(8),
				})
				if direction == OrderFeeReceivable {
					netting.BaseCurrencyAmount = netting.BaseCurrencyAmount.Add(allocated.Mul(bill.ExchangeRate).RoundBank(8))
				}
				remaining = remaining.Sub(allocated)
			}
			if remaining.IsPositive() {
				return nil, ErrFinanceNettingBalance
			}
		}
		netting.BaseCurrencyAmount = netting.BaseCurrencyAmount.RoundBank(8)
		result = append(result, netting)
	}
	return result, nil
}

func financeNettingAudit(organizationID, actorID, nettingID uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "finance_netting", ResourceID: nettingID.String(), Details: map[string]string{"finance_netting.id": nettingID.String()}}
}
