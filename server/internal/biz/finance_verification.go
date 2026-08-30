package biz

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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
	WriteOffBaseAmount, ExchangeGainLoss   decimal.Decimal
	Active                                 bool
}
type FinanceVerification struct {
	ID, OrganizationID, SettlementPartyID uuid.UUID
	VerificationNo, IdempotencyKey        string
	Status                                VerificationStatus
	Direction                             OrderFeeDirection
	SettlementPartyName, Currency         string
	Amount                                decimal.Decimal
	BaseCurrency                          string
	ExchangeRate                          decimal.Decimal
	ExchangeRateSource, ExchangeRateDate  string
	ExchangeRateSettingID                 *uuid.UUID
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
}
type VerificationListResult struct {
	Items   []*FinanceVerification
	Total   int64
	Summary VerificationSummary
}
type VerificationSummary struct {
	ReceivableBaseAmount decimal.Decimal
	PayableBaseAmount    decimal.Decimal
	BaseCurrency         string
}
type VerificationRepo interface {
	List(context.Context, uuid.UUID, VerificationFilter) (*VerificationListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceVerification, error)
	GetByKey(context.Context, uuid.UUID, string) (*FinanceVerification, error)
	LoadCashflowContext(context.Context, uuid.UUID, uuid.UUID) (*FinanceCashflow, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *FinanceVerification, *AuditEvent) (*FinanceVerification, error)
	Reverse(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceVerification, error)
}
type VerificationUsecase struct {
	repo         VerificationRepo
	exchangeRate *ExchangeRateUsecase
}

func NewVerificationUsecase(r VerificationRepo, exchangeRate *ExchangeRateUsecase) *VerificationUsecase {
	return &VerificationUsecase{repo: r, exchangeRate: exchangeRate}
}
func (u *VerificationUsecase) List(ctx context.Context, org uuid.UUID, f VerificationFilter) (*VerificationListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || !ValidListPagination(f.Page, f.PageSize) || (f.Status != "" && f.Status != VerificationActive && f.Status != VerificationReversed) {
		return nil, ErrVerificationInvalid
	}
	return u.repo.List(ctx, org, f)
}
func (u *VerificationUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateVerificationInput) (*FinanceVerification, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || len(in.Allocations) == 0 || len(in.Allocations) > 500 || !validFinanceDate(in.VerificationDate) || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrVerificationInvalid
	}
	if old, e := u.repo.GetByKey(ctx, org, in.IdempotencyKey); e != nil {
		return nil, e
	} else if old != nil {
		if sameVerificationIntent(old, in) {
			return old, nil
		}
		return nil, ErrVerificationIdempotency
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
	// 核销记录只能包含同一收付方向和币种；实际方向由仓储在锁内校验。
	// 先取首笔资金流水确定方向和币种，随后锁内再次核对全部分摊。
	firstCashflow, err := u.repo.LoadCashflowContext(ctx, org, v.Allocations[0].CashflowID)
	if err != nil {
		return nil, err
	}
	resolved, err := u.exchangeRate.Resolve(ctx, org, WriteOffRateType, firstCashflow.Direction, firstCashflow.Currency, map[string]string{WriteOffTimeStandard: in.VerificationDate})
	if err != nil {
		return nil, err
	}
	baseCurrency, err := u.exchangeRate.BaseCurrency(ctx, org)
	if err != nil {
		return nil, err
	}
	v.Direction = firstCashflow.Direction
	v.Currency = firstCashflow.Currency
	v.BaseCurrency = baseCurrency
	v.ExchangeRate = resolved.Rate
	v.ExchangeRateSource = resolved.Source
	v.ExchangeRateDate = resolved.RateDate
	v.ExchangeRateSettingID = resolved.SettingID
	v.BaseAmount = v.Amount.Mul(resolved.Rate).RoundBank(8)
	created, e := u.repo.Create(ctx, org, actor, v, verifyAudit(org, actor, id, "finance.verification.create"))
	if e == nil {
		return created, nil
	}
	old, lookupErr := u.repo.GetByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameVerificationIntent(old, in) {
		return old, nil
	}
	return nil, e
}

func CalculateVerificationAllocationAmounts(direction OrderFeeDirection, amount, billTotal, billBaseTotal, cashflowTotal, cashflowBaseTotal, writeOffRate decimal.Decimal) (billBase, cashflowBase, writeOffBase, gainLoss decimal.Decimal, err error) {
	if (direction != OrderFeeReceivable && direction != OrderFeePayable) || !amount.IsPositive() || !billTotal.IsPositive() || !billBaseTotal.IsPositive() || !cashflowTotal.IsPositive() || !cashflowBaseTotal.IsPositive() || !writeOffRate.IsPositive() {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, ErrVerificationInvalid
	}
	billBase = billBaseTotal.Mul(amount).Div(billTotal).RoundBank(8)
	cashflowBase = cashflowBaseTotal.Mul(amount).Div(cashflowTotal).RoundBank(8)
	writeOffBase = amount.Mul(writeOffRate).RoundBank(8)
	if direction == OrderFeeReceivable {
		gainLoss = cashflowBase.Sub(billBase).RoundBank(8)
	} else {
		gainLoss = billBase.Sub(cashflowBase).RoundBank(8)
	}
	return billBase, cashflowBase, writeOffBase, gainLoss, nil
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
func verifyAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_verification", ResourceID: id.String()}
}
