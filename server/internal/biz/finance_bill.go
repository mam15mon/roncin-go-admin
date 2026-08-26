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
)

var (
	ErrFinanceBillNotFound            = errors.NotFound("FINANCE_BILL_NOT_FOUND", "账单不存在")
	ErrFinanceBillInvalidArgument     = errors.BadRequest("FINANCE_BILL_INVALID_ARGUMENT", "账单字段不合法")
	ErrFinanceBillFeeInvalid          = errors.Conflict("FINANCE_BILL_FEE_INVALID", "所选费用必须为已确认状态且尚未进入其他账单")
	ErrFinanceBillFeeMismatch         = errors.BadRequest("FINANCE_BILL_FEE_MISMATCH", "同一账单的费用必须具有相同收付方向、结算单位、币种和本币")
	ErrFinanceBillVersionConflict     = errors.Conflict("FINANCE_BILL_VERSION_CONFLICT", "账单已被其他操作人修改，请刷新后重试")
	ErrFinanceBillInvalidTransition   = errors.Conflict("FINANCE_BILL_INVALID_TRANSITION", "当前账单状态不允许执行该操作")
	ErrFinanceBillIdempotencyConflict = errors.Conflict("FINANCE_BILL_IDEMPOTENCY_CONFLICT", "账单请求幂等键已被其他请求使用")
)

var financeBillCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type FinanceBillStatus string

const (
	FinanceBillDraft     FinanceBillStatus = "DRAFT"
	FinanceBillConfirmed FinanceBillStatus = "CONFIRMED"
	FinanceBillCancelled FinanceBillStatus = "CANCELLED"
)

type FinanceBill struct {
	ID                  uuid.UUID
	OrganizationID      uuid.UUID
	BillNo              string
	IdempotencyKey      string
	Direction           OrderFeeDirection
	Status              FinanceBillStatus
	SettlementPartyID   uuid.UUID
	SettlementPartyName string
	Currency            string
	BaseCurrency        string
	TotalAmount         decimal.Decimal
	NetAmount           decimal.Decimal
	TaxAmount           decimal.Decimal
	BaseCurrencyAmount  decimal.Decimal
	VerifiedAmount      decimal.Decimal
	UnverifiedAmount    decimal.Decimal
	FeeCount            int
	BillDate            string
	DueDate             *string
	Note                *string
	Version             uint64
	ConfirmedAt         *time.Time
	ConfirmedBy         *uuid.UUID
	CancelledAt         *time.Time
	CancelledBy         *uuid.UUID
	CancellationReason  *string
	Lines               []*FinanceBillLine
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type FinanceBillLine struct {
	ID                 uuid.UUID
	BillID             uuid.UUID
	OrderFeeID         uuid.UUID
	OrderID            uuid.UUID
	OrderNo            string
	BusinessType       string
	FeeCode            string
	FeeName            string
	TotalAmount        decimal.Decimal
	NetAmount          decimal.Decimal
	TaxAmount          decimal.Decimal
	Currency           string
	ExchangeRate       decimal.Decimal
	BaseCurrency       string
	BaseCurrencyAmount decimal.Decimal
	Active             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FinanceBillableFee struct {
	Fee          *OrderFee
	OrderNo      string
	BusinessType string
}

type FinanceBillFilter struct {
	Page              int
	PageSize          int
	Keyword           string
	Direction         OrderFeeDirection
	Status            FinanceBillStatus
	SettlementPartyID *uuid.UUID
	Currency          string
	BillDateFrom      string
	BillDateTo        string
}

type FinanceBillListResult struct {
	Items []*FinanceBill
	Total int64
}

type CreateFinanceBillInput struct {
	FeeIDs         []uuid.UUID
	BillDate       string
	DueDate        *string
	Note           *string
	IdempotencyKey string
}

type UpdateFinanceBillInput struct {
	ID              uuid.UUID
	BillDate        string
	DueDate         *string
	Note            *string
	ExpectedVersion uint64
}

type FinanceBillRepo interface {
	List(ctx context.Context, organizationID uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error)
	Get(ctx context.Context, organizationID, id uuid.UUID) (*FinanceBill, error)
	GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBill, error)
	LoadBillableFees(ctx context.Context, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error)
	Create(ctx context.Context, bill *FinanceBill, audit *AuditEvent) (*FinanceBill, error)
	Update(ctx context.Context, organizationID uuid.UUID, input UpdateFinanceBillInput, audit *AuditEvent) (*FinanceBill, error)
	Confirm(ctx context.Context, organizationID, id, actorID uuid.UUID, expectedVersion uint64, audit *AuditEvent) (*FinanceBill, error)
	Cancel(ctx context.Context, organizationID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) (*FinanceBill, error)
}

type FinanceBillUsecase struct {
	repo   FinanceBillRepo
	config *OrderConfigUsecase
}

func NewFinanceBillUsecase(repo FinanceBillRepo, config *OrderConfigUsecase) *FinanceBillUsecase {
	return &FinanceBillUsecase{repo: repo, config: config}
}

func (uc *FinanceBillUsecase) List(ctx context.Context, organizationID uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if organizationID == uuid.Nil || filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 200 || utf8.RuneCountInString(filter.Keyword) > 100 {
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
	return uc.repo.List(ctx, organizationID, filter)
}

func (uc *FinanceBillUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*FinanceBill, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.Get(ctx, organizationID, id)
}

func (uc *FinanceBillUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillInput) (*FinanceBill, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	normalized, err := normalizeCreateFinanceBill(input)
	if err != nil {
		return nil, err
	}
	existing, err := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if sameFinanceBillCreateIntent(existing, normalized) {
			return existing, nil
		}
		return nil, ErrFinanceBillIdempotencyConflict
	}
	fees, err := uc.repo.LoadBillableFees(ctx, organizationID, normalized.FeeIDs)
	if err != nil {
		return nil, err
	}
	bill, err := buildFinanceBill(organizationID, fees, normalized)
	if err != nil {
		return nil, err
	}
	bill.BillNo, err = uc.config.NextNumber(ctx, organizationID, DocumentTypeBill)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, bill, financeBillAudit(organizationID, actorID, bill.ID, "finance.bill.create"))
	if err == nil {
		return created, nil
	}
	existing, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey)
	if lookupErr == nil && existing != nil && sameFinanceBillCreateIntent(existing, normalized) {
		return existing, nil
	}
	return nil, err
}

func (uc *FinanceBillUsecase) Update(ctx context.Context, organizationID, actorID uuid.UUID, input UpdateFinanceBillInput) (*FinanceBill, error) {
	input.BillDate = strings.TrimSpace(input.BillDate)
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	if organizationID == uuid.Nil || actorID == uuid.Nil || input.ID == uuid.Nil || input.ExpectedVersion == 0 || !validFinanceDate(input.BillDate) || (input.DueDate != nil && (!validFinanceDate(*input.DueDate) || *input.DueDate < input.BillDate)) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.Update(ctx, organizationID, input, financeBillAudit(organizationID, actorID, input.ID, "finance.bill.update"))
}

func (uc *FinanceBillUsecase) Confirm(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64) (*FinanceBill, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.Confirm(ctx, organizationID, id, actorID, expectedVersion, financeBillAudit(organizationID, actorID, id, "finance.bill.confirm"))
}

func (uc *FinanceBillUsecase) Cancel(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceBill, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.Cancel(ctx, organizationID, id, actorID, expectedVersion, reason, financeBillAudit(organizationID, actorID, id, "finance.bill.cancel"))
}

func normalizeCreateFinanceBill(input CreateFinanceBillInput) (CreateFinanceBillInput, error) {
	input.BillDate = strings.TrimSpace(input.BillDate)
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.FeeIDs) == 0 || len(input.FeeIDs) > 500 || !validFinanceDate(input.BillDate) || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || (input.DueDate != nil && (!validFinanceDate(*input.DueDate) || *input.DueDate < input.BillDate)) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) {
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
		SettlementPartyName: first.Fee.SettlementPartyName, Currency: first.Fee.Currency, BaseCurrency: first.Fee.BaseCurrency,
		BillDate: input.BillDate, DueDate: input.DueDate, Note: input.Note, Version: 1,
		Lines: make([]*FinanceBillLine, 0, len(fees)),
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
			TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, Currency: fee.Currency,
			ExchangeRate: fee.ExchangeRate, BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount, Active: true,
		})
	}
	bill.FeeCount = len(bill.Lines)
	return bill, nil
}

func sameFinanceBillCreateIntent(existing *FinanceBill, requested CreateFinanceBillInput) bool {
	if existing == nil || existing.BillDate != requested.BillDate || !stringPointersEqual(existing.DueDate, requested.DueDate) || !stringPointersEqual(existing.Note, requested.Note) || len(existing.Lines) != len(requested.FeeIDs) {
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

func validFinanceDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func validFinanceDateRange(from, to string) bool {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	return (from == "" || validFinanceDate(from)) && (to == "" || validFinanceDate(to)) && (from == "" || to == "" || from <= to)
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
