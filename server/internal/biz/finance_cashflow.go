package biz

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrFinanceCashflowNotFound              = errors.NotFound("FINANCE_CASHFLOW_NOT_FOUND", "收付流水不存在")
	ErrFinanceCashflowInvalidArgument       = errors.BadRequest("FINANCE_CASHFLOW_INVALID_ARGUMENT", "收付流水字段不合法")
	ErrFinanceCashflowVersionConflict       = errors.Conflict("FINANCE_CASHFLOW_VERSION_CONFLICT", "收付流水已被其他操作人修改")
	ErrFinanceCashflowInvalidTransition     = errors.Conflict("FINANCE_CASHFLOW_INVALID_TRANSITION", "当前流水状态不允许执行该操作")
	ErrFinanceCashflowIdempotencyConflict   = errors.Conflict("FINANCE_CASHFLOW_IDEMPOTENCY_CONFLICT", "流水请求幂等键已被其他请求使用")
	ErrFinanceCashflowRateOverrideForbidden = errors.Forbidden("FINANCE_CASHFLOW_RATE_OVERRIDE_FORBIDDEN", "无权手工覆盖资金流水汇率")
)

type FinanceCashflowStatus string

const (
	FinanceCashflowDraft     FinanceCashflowStatus = "DRAFT"
	FinanceCashflowConfirmed FinanceCashflowStatus = "CONFIRMED"
	FinanceCashflowCancelled FinanceCashflowStatus = "CANCELLED"
)

type FinanceCashflow struct {
	ID, OrganizationID, SettlementPartyID                    uuid.UUID
	FlowNo, IdempotencyKey                                   string
	Direction                                                OrderFeeDirection
	Status                                                   FinanceCashflowStatus
	SettlementPartyName, Currency                            string
	Amount, ExchangeRate, BaseAmount                         decimal.Decimal
	ExchangeRateSource, ExchangeRateDate                     string
	ExchangeRateSettingID                                    *uuid.UUID
	VerifiedAmount, UnverifiedAmount                         decimal.Decimal
	BaseCurrency, TransactionDate, OurAccount, PaymentMethod string
	CounterpartyAccount, BankReferenceNo, Note               *string
	Version                                                  uint64
	ConfirmedAt, CancelledAt                                 *time.Time
	ConfirmedBy, CancelledBy                                 *uuid.UUID
	CancellationReason                                       *string
	CreatedAt, UpdatedAt                                     time.Time
}
type FinanceCashflowFilter struct {
	Page, PageSize    int
	Keyword           string
	Direction         OrderFeeDirection
	Status            FinanceCashflowStatus
	SettlementPartyID *uuid.UUID
	Currency          string
}
type FinanceCashflowListResult struct {
	Items   []*FinanceCashflow
	Total   int64
	Summary FinanceCashflowSummary
}
type FinanceCashflowSummary struct {
	ReceivableBaseAmount decimal.Decimal
	PayableBaseAmount    decimal.Decimal
	UnverifiedBaseAmount decimal.Decimal
	BaseCurrency         string
}
type CreateFinanceCashflowInput struct {
	Direction                                                OrderFeeDirection
	SettlementPartyID                                        uuid.UUID
	Currency                                                 string
	Amount                                                   decimal.Decimal
	ExchangeRateOverride                                     *decimal.Decimal
	BaseCurrency, TransactionDate, OurAccount, PaymentMethod string
	CounterpartyAccount, BankReferenceNo, Note               *string
	IdempotencyKey                                           string
}
type FinanceCashflowRepo interface {
	List(context.Context, uuid.UUID, FinanceCashflowFilter) (*FinanceCashflowListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceCashflow, error)
	GetByIdempotencyKey(context.Context, uuid.UUID, string) (*FinanceCashflow, error)
	ResolveParty(context.Context, uuid.UUID, uuid.UUID) (string, error)
	Create(context.Context, *FinanceCashflow, *AuditEvent) (*FinanceCashflow, error)
	Confirm(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, *AuditEvent) (*FinanceCashflow, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceCashflow, error)
}
type FinanceCashflowUsecase struct {
	repo         FinanceCashflowRepo
	exchangeRate *ExchangeRateUsecase
}

func NewFinanceCashflowUsecase(repo FinanceCashflowRepo, exchangeRate *ExchangeRateUsecase) *FinanceCashflowUsecase {
	return &FinanceCashflowUsecase{repo: repo, exchangeRate: exchangeRate}
}
func (uc *FinanceCashflowUsecase) List(ctx context.Context, org uuid.UUID, f FinanceCashflowFilter) (*FinanceCashflowListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	f.Currency = strings.ToUpper(strings.TrimSpace(f.Currency))
	if org == uuid.Nil || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 || (f.Direction != "" && f.Direction != OrderFeeReceivable && f.Direction != OrderFeePayable) || (f.Status != "" && f.Status != FinanceCashflowDraft && f.Status != FinanceCashflowConfirmed && f.Status != FinanceCashflowCancelled) || (f.SettlementPartyID != nil && *f.SettlementPartyID == uuid.Nil) || (f.Currency != "" && !financeBillCurrencyPattern.MatchString(f.Currency)) {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	return uc.repo.List(ctx, org, f)
}
func (uc *FinanceCashflowUsecase) Get(ctx context.Context, org, id uuid.UUID) (*FinanceCashflow, error) {
	if org == uuid.Nil || id == uuid.Nil {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	return uc.repo.Get(ctx, org, id)
}
func (uc *FinanceCashflowUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateFinanceCashflowInput, canOverrideExchangeRate bool) (*FinanceCashflow, error) {
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.BaseCurrency = strings.ToUpper(strings.TrimSpace(in.BaseCurrency))
	in.TransactionDate = strings.TrimSpace(in.TransactionDate)
	in.OurAccount = strings.TrimSpace(in.OurAccount)
	in.PaymentMethod = strings.TrimSpace(in.PaymentMethod)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.CounterpartyAccount = normalizedOptionalFinanceString(in.CounterpartyAccount)
	in.BankReferenceNo = normalizedOptionalFinanceString(in.BankReferenceNo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.SettlementPartyID == uuid.Nil || (in.Direction != OrderFeeReceivable && in.Direction != OrderFeePayable) || !financeBillCurrencyPattern.MatchString(in.Currency) || !in.Amount.IsPositive() || !validFinanceDate(in.TransactionDate) || in.OurAccount == "" || in.PaymentMethod == "" || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	if uc.exchangeRate == nil {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	if old, err := uc.repo.GetByIdempotencyKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		// 幂等重放必须优先于实时汇率解析。否则汇率配置在两次请求之间发生变化时，
		// 原请求可能被误判成手工覆盖汇率，甚至因旧汇率配置已停用而无法重放。
		in.BaseCurrency = old.BaseCurrency
		if sameFinanceCashflowIntent(old, in) {
			return old, nil
		}
		return nil, ErrFinanceCashflowIdempotencyConflict
	}
	systemRate, err := uc.exchangeRate.Resolve(ctx, org, SettlementRateType, in.Direction, in.Currency, map[string]string{TransactionDateStandard: in.TransactionDate})
	if err != nil {
		return nil, err
	}
	resolved := systemRate
	if in.ExchangeRateOverride != nil && !in.ExchangeRateOverride.Equal(systemRate.Rate) {
		if !canOverrideExchangeRate {
			return nil, ErrFinanceCashflowRateOverrideForbidden
		}
		if !validExchangeRate(*in.ExchangeRateOverride) {
			return nil, ErrFinanceCashflowInvalidArgument
		}
		resolved = &ResolvedExchangeRate{Rate: *in.ExchangeRateOverride, Source: "MANUAL", RateDate: in.TransactionDate}
	}
	baseCurrency, err := uc.exchangeRate.BaseCurrency(ctx, org)
	if err != nil {
		return nil, err
	}
	in.BaseCurrency = baseCurrency
	name, e := uc.repo.ResolveParty(ctx, org, in.SettlementPartyID)
	if e != nil {
		return nil, e
	}
	item := &FinanceCashflow{ID: uuid.Must(uuid.NewV7()), OrganizationID: org, IdempotencyKey: in.IdempotencyKey, Direction: in.Direction, Status: FinanceCashflowDraft, SettlementPartyID: in.SettlementPartyID, SettlementPartyName: name, Currency: in.Currency, Amount: in.Amount, ExchangeRate: resolved.Rate, ExchangeRateSource: resolved.Source, ExchangeRateDate: resolved.RateDate, ExchangeRateSettingID: resolved.SettingID, BaseCurrency: in.BaseCurrency, BaseAmount: in.Amount.Mul(resolved.Rate).RoundBank(8), TransactionDate: in.TransactionDate, OurAccount: in.OurAccount, PaymentMethod: in.PaymentMethod, CounterpartyAccount: in.CounterpartyAccount, BankReferenceNo: in.BankReferenceNo, Note: in.Note, Version: 1}
	created, e := uc.repo.Create(ctx, item, cashflowAudit(org, actor, item.ID, "finance.cashflow.create"))
	if e == nil {
		return created, nil
	}
	old, lookupErr := uc.repo.GetByIdempotencyKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameFinanceCashflowIntent(old, in) {
		return old, nil
	}
	return nil, e
}

func sameFinanceCashflowIntent(old *FinanceCashflow, in CreateFinanceCashflowInput) bool {
	return old != nil && old.Direction == in.Direction && old.SettlementPartyID == in.SettlementPartyID && old.Currency == in.Currency && old.Amount.Equal(in.Amount) && (in.ExchangeRateOverride == nil || old.ExchangeRate.Equal(*in.ExchangeRateOverride)) && old.BaseCurrency == in.BaseCurrency && old.TransactionDate == in.TransactionDate && old.OurAccount == in.OurAccount && old.PaymentMethod == in.PaymentMethod && stringPointersEqual(old.CounterpartyAccount, in.CounterpartyAccount) && stringPointersEqual(old.BankReferenceNo, in.BankReferenceNo) && stringPointersEqual(old.Note, in.Note)
}
func (uc *FinanceCashflowUsecase) Confirm(ctx context.Context, org, actor, id uuid.UUID, v uint64) (*FinanceCashflow, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || v == 0 {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	return uc.repo.Confirm(ctx, org, id, actor, v, cashflowAudit(org, actor, id, "finance.cashflow.confirm"))
}
func (uc *FinanceCashflowUsecase) Cancel(ctx context.Context, org, actor, id uuid.UUID, v uint64, reason string) (*FinanceCashflow, error) {
	reason = strings.TrimSpace(reason)
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || v == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	return uc.repo.Cancel(ctx, org, id, actor, v, reason, cashflowAudit(org, actor, id, "finance.cashflow.cancel"))
}
func cashflowAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_cashflow", ResourceID: id.String()}
}
