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
	ErrFinanceCashflowNotFound            = errors.NotFound("FINANCE_CASHFLOW_NOT_FOUND", "收付流水不存在")
	ErrFinanceCashflowInvalidArgument     = errors.BadRequest("FINANCE_CASHFLOW_INVALID_ARGUMENT", "收付流水字段不合法")
	ErrFinanceCashflowVersionConflict     = errors.Conflict("FINANCE_CASHFLOW_VERSION_CONFLICT", "收付流水已被其他操作人修改")
	ErrFinanceCashflowInvalidTransition   = errors.Conflict("FINANCE_CASHFLOW_INVALID_TRANSITION", "当前流水状态不允许执行该操作")
	ErrFinanceCashflowIdempotencyConflict = errors.Conflict("FINANCE_CASHFLOW_IDEMPOTENCY_CONFLICT", "流水请求幂等键已被其他请求使用")
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
	Page, PageSize int
	Keyword        string
	Direction      OrderFeeDirection
	Status         FinanceCashflowStatus
}
type FinanceCashflowListResult struct {
	Items []*FinanceCashflow
	Total int64
}
type CreateFinanceCashflowInput struct {
	Direction                                                OrderFeeDirection
	SettlementPartyID                                        uuid.UUID
	Currency                                                 string
	Amount, ExchangeRate                                     decimal.Decimal
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
	repo   FinanceCashflowRepo
	config *OrderConfigUsecase
}

func NewFinanceCashflowUsecase(repo FinanceCashflowRepo, config *OrderConfigUsecase) *FinanceCashflowUsecase {
	return &FinanceCashflowUsecase{repo: repo, config: config}
}
func (uc *FinanceCashflowUsecase) List(ctx context.Context, org uuid.UUID, f FinanceCashflowFilter) (*FinanceCashflowListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || f.Page < 1 || f.PageSize < 1 || f.PageSize > 200 || utf8.RuneCountInString(f.Keyword) > 100 || (f.Direction != "" && f.Direction != OrderFeeReceivable && f.Direction != OrderFeePayable) || (f.Status != "" && f.Status != FinanceCashflowDraft && f.Status != FinanceCashflowConfirmed && f.Status != FinanceCashflowCancelled) {
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
func (uc *FinanceCashflowUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateFinanceCashflowInput) (*FinanceCashflow, error) {
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.BaseCurrency = strings.ToUpper(strings.TrimSpace(in.BaseCurrency))
	in.TransactionDate = strings.TrimSpace(in.TransactionDate)
	in.OurAccount = strings.TrimSpace(in.OurAccount)
	in.PaymentMethod = strings.TrimSpace(in.PaymentMethod)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.CounterpartyAccount = normalizedOptionalFinanceString(in.CounterpartyAccount)
	in.BankReferenceNo = normalizedOptionalFinanceString(in.BankReferenceNo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.SettlementPartyID == uuid.Nil || (in.Direction != OrderFeeReceivable && in.Direction != OrderFeePayable) || !financeBillCurrencyPattern.MatchString(in.Currency) || !financeBillCurrencyPattern.MatchString(in.BaseCurrency) || !in.Amount.IsPositive() || !in.ExchangeRate.IsPositive() || !validFinanceDate(in.TransactionDate) || in.OurAccount == "" || in.PaymentMethod == "" || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 {
		return nil, ErrFinanceCashflowInvalidArgument
	}
	if old, e := uc.repo.GetByIdempotencyKey(ctx, org, in.IdempotencyKey); e != nil {
		return nil, e
	} else if old != nil {
		if old.Direction == in.Direction && old.SettlementPartyID == in.SettlementPartyID && old.Currency == in.Currency && old.Amount.Equal(in.Amount) && old.ExchangeRate.Equal(in.ExchangeRate) && old.BaseCurrency == in.BaseCurrency && old.TransactionDate == in.TransactionDate && old.OurAccount == in.OurAccount && old.PaymentMethod == in.PaymentMethod && stringPointersEqual(old.CounterpartyAccount, in.CounterpartyAccount) && stringPointersEqual(old.BankReferenceNo, in.BankReferenceNo) && stringPointersEqual(old.Note, in.Note) {
			return old, nil
		}
		return nil, ErrFinanceCashflowIdempotencyConflict
	}
	name, e := uc.repo.ResolveParty(ctx, org, in.SettlementPartyID)
	if e != nil {
		return nil, e
	}
	item := &FinanceCashflow{ID: uuid.Must(uuid.NewV7()), OrganizationID: org, IdempotencyKey: in.IdempotencyKey, Direction: in.Direction, Status: FinanceCashflowDraft, SettlementPartyID: in.SettlementPartyID, SettlementPartyName: name, Currency: in.Currency, Amount: in.Amount, ExchangeRate: in.ExchangeRate, BaseCurrency: in.BaseCurrency, BaseAmount: in.Amount.Mul(in.ExchangeRate).RoundBank(8), TransactionDate: in.TransactionDate, OurAccount: in.OurAccount, PaymentMethod: in.PaymentMethod, CounterpartyAccount: in.CounterpartyAccount, BankReferenceNo: in.BankReferenceNo, Note: in.Note, Version: 1}
	item.FlowNo, e = uc.config.NextNumber(ctx, org, DocumentTypeReceiptPayment)
	if e != nil {
		return nil, e
	}
	return uc.repo.Create(ctx, item, cashflowAudit(org, actor, item.ID, "finance.cashflow.create"))
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
