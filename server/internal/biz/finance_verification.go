package biz

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log/slog"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrVerificationNotFound    = errors.NotFound("FINANCE_VERIFICATION_NOT_FOUND", "核销记录不存在")
	ErrVerificationInvalid     = errors.BadRequest("FINANCE_VERIFICATION_INVALID", "核销参数不合法")
	ErrVerificationBalance     = errors.Conflict("FINANCE_VERIFICATION_BALANCE", "核销金额超过资金或账单未核销余额")
	ErrVerificationMismatch    = errors.BadRequest("FINANCE_VERIFICATION_MISMATCH", "核销双方方向、结算单位或币种不一致")
	ErrVerificationTransition  = errors.Conflict("FINANCE_VERIFICATION_TRANSITION", "当前核销状态不允许该操作")
	ErrVerificationIdempotency = errors.Conflict("FINANCE_VERIFICATION_IDEMPOTENCY", "幂等键已用于不同的核销请求")
)

type VerificationStatus string

const (
	VerificationActive   VerificationStatus = "ACTIVE"
	VerificationReversed VerificationStatus = "REVERSED"
)

type VerificationAllocation struct {
	ID, VerificationID, CashflowID, BillID uuid.UUID
	CashflowNo, BillNo                     string
	Amount                                 decimal.Decimal
	BillBaseAmount, CashflowBaseAmount     decimal.Decimal
	ExchangeGainLoss                       decimal.Decimal
	Active                                 bool
}
type FinanceVerification struct {
	ID, OrganizationID, SettlementPartyID uuid.UUID
	VerificationNo, IdempotencyKey        string
	OrganizationName                      string
	Status                                VerificationStatus
	Direction                             OrderFeeDirection
	SettlementPartyName, Currency         string
	Amount                                decimal.Decimal
	BaseCurrency                          string
	BaseAmount, BillBaseAmount            decimal.Decimal
	CashflowBaseAmount, ExchangeGainLoss  decimal.Decimal
	VerificationDate                      string
	Note                                  *string
	Version                               uint64
	ReversedAt                            *time.Time
	ReversedBy                            *uuid.UUID
	ReversalReason                        *string
	Allocations                           []*VerificationAllocation
	CreatedAt, UpdatedAt                  time.Time
}
type CreateVerificationInput struct {
	Allocations      []*VerificationAllocation
	VerificationDate string
	Note             *string
	IdempotencyKey   string
}
type VerificationFilter struct {
	Page, PageSize int
	Keyword        string
	Status         VerificationStatus
	Direction      OrderFeeDirection
}
type VerificationListResult struct {
	Items   []*FinanceVerification
	Total   int64
	Summary VerificationSummary
}

// VerificationCreationCandidateFilter 是核销创建工作台的单组织候选约束。
// 最终创建仍在事务内重读并锁定资金流水与账单，不以候选结果作为写入依据。
type VerificationCreationCandidateFilter struct {
	Direction         OrderFeeDirection
	SettlementPartyID uuid.UUID
	Currency          string
}
type VerificationCreationCandidates struct {
	Cashflows []*FinanceCashflow
	Bills     []*FinanceBill
}
type VerificationSummary struct {
	AmountsByBaseCurrency []FinanceBaseCurrencyAmount
}
type VerificationRepo interface {
	List(context.Context, uuid.UUID, VerificationFilter) (*VerificationListResult, error)
	ListScoped(context.Context, []uuid.UUID, VerificationFilter) (*VerificationListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceVerification, error)
	GetScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceVerification, error)
	GetByKey(context.Context, uuid.UUID, string) (*FinanceVerification, error)
	LoadCashflowContext(context.Context, uuid.UUID, uuid.UUID) (*FinanceCashflow, error)
	LoadCashflowContextScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceCashflow, error)
	ListCreationCandidates(context.Context, uuid.UUID, VerificationCreationCandidateFilter) (*VerificationCreationCandidates, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *FinanceVerification, *AuditEvent) (*FinanceVerification, error)
	Reverse(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceVerification, error)
}
type VerificationUsecase struct {
	repo         VerificationRepo
	exchangeRate *ExchangeRateUsecase
	transactor   Transactor
	autoLock     *AutoOrderLockUsecase
	logger       *slog.Logger
}

func NewVerificationUsecase(r VerificationRepo, exchangeRate *ExchangeRateUsecase, transactor Transactor, autoLock *AutoOrderLockUsecase, logger *slog.Logger) *VerificationUsecase {
	return &VerificationUsecase{repo: r, exchangeRate: exchangeRate, transactor: transactor, autoLock: autoLock, logger: logger}
}
func (u *VerificationUsecase) List(ctx context.Context, org uuid.UUID, f VerificationFilter) (*VerificationListResult, error) {
	return u.ListScoped(ctx, []uuid.UUID{org}, f)
}
func (u *VerificationUsecase) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, f VerificationFilter) (*VerificationListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if !validFinanceVerificationOrganizationIDs(organizationIDs) || !ValidListPagination(f.Page, f.PageSize) || (f.Status != "" && f.Status != VerificationActive && f.Status != VerificationReversed) || (f.Direction != "" && f.Direction != OrderFeeReceivable && f.Direction != OrderFeePayable) {
		return nil, ErrVerificationInvalid
	}
	return u.repo.ListScoped(ctx, organizationIDs, f)
}

func (u *VerificationUsecase) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceVerification, error) {
	if !validFinanceVerificationOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrVerificationInvalid
	}
	return u.repo.GetScoped(ctx, organizationIDs, id)
}

func (u *VerificationUsecase) LoadCashflowContextScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCashflow, error) {
	if !validFinanceVerificationOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrVerificationInvalid
	}
	return u.repo.LoadCashflowContextScoped(ctx, organizationIDs, id)
}

func (u *VerificationUsecase) ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter VerificationCreationCandidateFilter) (*VerificationCreationCandidates, error) {
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if organizationID == uuid.Nil || filter.SettlementPartyID == uuid.Nil ||
		(filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable) ||
		!financeBillCurrencyPattern.MatchString(filter.Currency) {
		return nil, ErrVerificationInvalid
	}
	return u.repo.ListCreationCandidates(ctx, organizationID, filter)
}

func validFinanceVerificationOrganizationIDs(organizationIDs []uuid.UUID) bool {
	if len(organizationIDs) == 0 {
		return false
	}
	seen := make(map[uuid.UUID]struct{}, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if organizationID == uuid.Nil {
			return false
		}
		if _, exists := seen[organizationID]; exists {
			return false
		}
		seen[organizationID] = struct{}{}
	}
	return true
}
func (u *VerificationUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateVerificationInput) (*FinanceVerification, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || len(in.Allocations) == 0 || len(in.Allocations) > 500 || !validFinanceDate(in.VerificationDate) || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrVerificationInvalid
	}
	id := uuid.Must(uuid.NewV7())
	v := &FinanceVerification{ID: id, OrganizationID: org, IdempotencyKey: in.IdempotencyKey, Status: VerificationActive, VerificationDate: in.VerificationDate, Note: in.Note, Version: 1, Allocations: in.Allocations}
	seen := make(map[string]struct{}, len(v.Allocations))
	for _, a := range v.Allocations {
		if a == nil || a.CashflowID == uuid.Nil || a.BillID == uuid.Nil || !a.Amount.IsPositive() {
			return nil, ErrVerificationInvalid
		}
		pair := a.CashflowID.String() + ":" + a.BillID.String()
		if _, exists := seen[pair]; exists {
			return nil, ErrVerificationInvalid
		}
		seen[pair] = struct{}{}
		a.ID = uuid.Must(uuid.NewV7())
		a.VerificationID = id
		a.Active = true
		v.Amount = v.Amount.Add(a.Amount)
	}
	if u.exchangeRate == nil {
		return nil, ErrVerificationInvalid
	}
	var created *FinanceVerification
	isNew := false
	err := u.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		old, transactionErr := u.repo.GetByKey(txCtx, org, in.IdempotencyKey)
		if transactionErr != nil {
			return transactionErr
		}
		if old != nil {
			if !sameVerificationIntent(old, in) {
				return ErrVerificationIdempotency
			}
			created = old
			return nil
		}
		// 核销记录只能包含同一收付方向和币种；实际方向由仓储在锁内校验。
		// 先取首笔资金流水确定方向和币种，随后锁内再次核对全部分摊。
		firstCashflow, transactionErr := u.repo.LoadCashflowContext(txCtx, org, v.Allocations[0].CashflowID)
		if transactionErr != nil {
			return transactionErr
		}
		baseCurrency, transactionErr := u.exchangeRate.BaseCurrency(txCtx, org)
		if transactionErr != nil {
			return transactionErr
		}
		v.Direction = firstCashflow.Direction
		v.Currency = firstCashflow.Currency
		v.BaseCurrency = baseCurrency
		// 单头本位币金额严格等于行级流水本位币合计；核销不再解析汇率，由仓储在
		// 计算分摊时累加 cashflow_base_amount 回填。
		isNew = true
		created, transactionErr = u.repo.Create(txCtx, org, actor, v, verifyAudit(org, actor, id, "finance.verification.create"))
		return transactionErr
	})
	if err == nil {
		if isNew {
			// 应收核销创建生效后触发一次结清自动锁定检查：使用独立事务，
			// 锁定失败不回滚或改变已提交的核销事实，也不伪装成本接口失败。
			u.triggerAutoLock(ctx, org, actor, created.ID)
		}
		return u.repo.Get(ctx, org, created.ID)
	}
	old, lookupErr := u.repo.GetByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameVerificationIntent(old, in) {
		return old, nil
	}
	return nil, err
}

// CalculateVerificationAllocationAmounts 计算单笔分摊的账单侧与流水侧本位币金额及汇兑损益；
// 核销不再携带汇率，金额只由账单/流水各自已固化的本位币快照按分摊比例折算。
func CalculateVerificationAllocationAmounts(direction OrderFeeDirection, amount, billTotal, billBaseTotal, cashflowTotal, cashflowBaseTotal decimal.Decimal) (billBase, cashflowBase, gainLoss decimal.Decimal, err error) {
	if (direction != OrderFeeReceivable && direction != OrderFeePayable) || !amount.IsPositive() || !billTotal.IsPositive() || !billBaseTotal.IsPositive() || !cashflowTotal.IsPositive() || !cashflowBaseTotal.IsPositive() {
		return decimal.Zero, decimal.Zero, decimal.Zero, ErrVerificationInvalid
	}
	billBase = billBaseTotal.Mul(amount).Div(billTotal).RoundBank(8)
	cashflowBase = cashflowBaseTotal.Mul(amount).Div(cashflowTotal).RoundBank(8)
	if direction == OrderFeeReceivable {
		gainLoss = cashflowBase.Sub(billBase).RoundBank(8)
	} else {
		gainLoss = billBase.Sub(cashflowBase).RoundBank(8)
	}
	return billBase, cashflowBase, gainLoss, nil
}
func sameVerificationIntent(old *FinanceVerification, in CreateVerificationInput) bool {
	if old == nil || old.VerificationDate != in.VerificationDate || !stringPointersEqual(old.Note, in.Note) || len(old.Allocations) != len(in.Allocations) {
		return false
	}
	oldKeys := make([]string, 0, len(old.Allocations))
	newKeys := make([]string, 0, len(in.Allocations))
	for _, a := range old.Allocations {
		oldKeys = append(oldKeys, a.CashflowID.String()+":"+a.BillID.String()+":"+a.Amount.StringFixed(8))
	}
	for _, a := range in.Allocations {
		if a == nil {
			return false
		}
		newKeys = append(newKeys, a.CashflowID.String()+":"+a.BillID.String()+":"+a.Amount.StringFixed(8))
	}
	sort.Strings(oldKeys)
	sort.Strings(newKeys)
	for i := range oldKeys {
		if oldKeys[i] != newKeys[i] {
			return false
		}
	}
	return true
}
func (u *VerificationUsecase) Reverse(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceVerification, error) {
	reason = strings.TrimSpace(reason)
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrVerificationInvalid
	}
	return u.repo.Reverse(ctx, org, id, actor, version, reason, verifyAudit(org, actor, id, "finance.verification.reverse"))
}

// triggerAutoLock 在核销事务成功提交后触发结清自动锁定检查。
// 触发失败只记录警告日志：自动锁定失败不得回滚或改变已提交的核销事实，
// 也不得把锁定失败伪装成原业务失败；后续有效结清事件会重新检查。
func (u *VerificationUsecase) triggerAutoLock(ctx context.Context, org, actor, verificationID uuid.UUID) {
	if u.autoLock == nil {
		return
	}
	trigger := AutoOrderLockTrigger{
		Type:           AutoLockTriggerVerification,
		ResourceID:     verificationID,
		OrganizationID: org,
		TriggeredBy:    actor,
	}
	if err := u.autoLock.RunSettlementLockCheck(ctx, trigger); err != nil {
		if u.logger != nil {
			u.logger.WarnContext(ctx, "应收核销触发的自动锁定检查失败",
				slog.String("verification_id", verificationID.String()),
				slog.String("organization_id", org.String()),
				slog.String("error", err.Error()))
		}
	}
}
func verifyAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_verification", ResourceID: id.String()}
}
