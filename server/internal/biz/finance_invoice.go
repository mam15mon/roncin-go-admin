package biz

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFinanceInvoiceNotFound            = errors.NotFound("FINANCE_INVOICE_NOT_FOUND", "开票记录不存在")
	ErrFinanceInvoiceInvalidArgument     = errors.BadRequest("FINANCE_INVOICE_INVALID_ARGUMENT", "开票记录字段不合法")
	ErrFinanceInvoiceBillInvalid         = errors.Conflict("FINANCE_INVOICE_BILL_INVALID", "所选账单必须已确认且未进入其他有效开票记录")
	ErrFinanceInvoiceBillMismatch        = errors.BadRequest("FINANCE_INVOICE_BILL_MISMATCH", "同一开票记录的账单必须具有相同收付方向、结算单位和币种")
	ErrFinanceInvoiceVersionConflict     = errors.Conflict("FINANCE_INVOICE_VERSION_CONFLICT", "开票记录已被其他操作人修改，请刷新后重试")
	ErrFinanceInvoiceInvalidTransition   = errors.Conflict("FINANCE_INVOICE_INVALID_TRANSITION", "当前开票记录状态不允许执行该操作")
	ErrFinanceInvoiceIdempotencyConflict = errors.Conflict("FINANCE_INVOICE_IDEMPOTENCY_CONFLICT", "开票请求幂等键已被其他请求使用")
)

type FinanceInvoiceStatus string
type FinanceInvoiceType string

const (
	FinanceInvoiceDraft     FinanceInvoiceStatus = "DRAFT"
	FinanceInvoiceIssued    FinanceInvoiceStatus = "ISSUED"
	FinanceInvoiceCancelled FinanceInvoiceStatus = "CANCELLED"
	FinanceInvoiceNormal    FinanceInvoiceType   = "NORMAL"
	FinanceInvoiceSpecial   FinanceInvoiceType   = "SPECIAL"
)

type FinanceInvoice struct {
	ID, OrganizationID, SettlementPartyID uuid.UUID
	RecordNo, IdempotencyKey              string
	Direction                             OrderFeeDirection
	Status                                FinanceInvoiceStatus
	InvoiceType                           FinanceInvoiceType
	SettlementPartyName, Currency         string
	TotalAmount, TaxAmount                decimal.Decimal
	BillCount                             int
	TaxInvoiceNo, InvoiceDate, Note       *string
	Version                               uint64
	IssuedAt, CancelledAt                 *time.Time
	IssuedBy, CancelledBy                 *uuid.UUID
	CancellationReason                    *string
	Links                                 []*FinanceInvoiceBill
	CreatedAt, UpdatedAt                  time.Time
}

type FinanceInvoiceBill struct {
	ID, InvoiceID, BillID uuid.UUID
	BillNo                string
	Amount, TaxAmount     decimal.Decimal
	Active                bool
}

type FinanceInvoiceFilter struct {
	Page, PageSize int
	Keyword        string
	Direction      OrderFeeDirection
	Status         FinanceInvoiceStatus
}

type FinanceInvoiceListResult struct {
	Items []*FinanceInvoice
	Total int64
}

type CreateFinanceInvoiceInput struct {
	BillIDs        []uuid.UUID
	InvoiceType    FinanceInvoiceType
	Note           *string
	IdempotencyKey string
}

type FinanceInvoiceRepo interface {
	List(context.Context, uuid.UUID, FinanceInvoiceFilter) (*FinanceInvoiceListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceInvoice, error)
	GetByIdempotencyKey(context.Context, uuid.UUID, string) (*FinanceInvoice, error)
	LoadBills(context.Context, uuid.UUID, []uuid.UUID) ([]*FinanceBill, error)
	Create(context.Context, *FinanceInvoice, *AuditEvent) (*FinanceInvoice, error)
	Issue(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, string, *AuditEvent) (*FinanceInvoice, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceInvoice, error)
}

type FinanceInvoiceUsecase struct {
	repo   FinanceInvoiceRepo
	config *OrderConfigUsecase
}

func NewFinanceInvoiceUsecase(repo FinanceInvoiceRepo, config *OrderConfigUsecase) *FinanceInvoiceUsecase {
	return &FinanceInvoiceUsecase{repo: repo, config: config}
}

func (uc *FinanceInvoiceUsecase) List(ctx context.Context, organizationID uuid.UUID, filter FinanceInvoiceFilter) (*FinanceInvoiceListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if organizationID == uuid.Nil || filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 200 || utf8.RuneCountInString(filter.Keyword) > 100 || (filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable) || (filter.Status != "" && filter.Status != FinanceInvoiceDraft && filter.Status != FinanceInvoiceIssued && filter.Status != FinanceInvoiceCancelled) {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, filter)
}

func (uc *FinanceInvoiceUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*FinanceInvoice, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.Get(ctx, organizationID, id)
}

func (uc *FinanceInvoiceUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceInvoiceInput) (*FinanceInvoice, error) {
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.Note = normalizedOptionalFinanceString(input.Note)
	if organizationID == uuid.Nil || actorID == uuid.Nil || len(input.BillIDs) == 0 || len(input.BillIDs) > 500 || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || (input.InvoiceType != FinanceInvoiceNormal && input.InvoiceType != FinanceInvoiceSpecial) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	seen := map[uuid.UUID]struct{}{}
	for _, id := range input.BillIDs {
		if id == uuid.Nil {
			return nil, ErrFinanceInvoiceInvalidArgument
		}
		if _, ok := seen[id]; ok {
			return nil, ErrFinanceInvoiceInvalidArgument
		}
		seen[id] = struct{}{}
	}
	sort.Slice(input.BillIDs, func(i, j int) bool { return input.BillIDs[i].String() < input.BillIDs[j].String() })
	if existing, err := uc.repo.GetByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		if existing.InvoiceType == input.InvoiceType && stringPointersEqual(existing.Note, input.Note) && len(existing.Links) == len(input.BillIDs) {
			ids := make([]string, 0, len(existing.Links))
			for _, l := range existing.Links {
				ids = append(ids, l.BillID.String())
			}
			sort.Strings(ids)
			same := true
			for i, id := range input.BillIDs {
				if ids[i] != id.String() {
					same = false
					break
				}
			}
			if same {
				return existing, nil
			}
		}
		return nil, ErrFinanceInvoiceIdempotencyConflict
	}
	bills, err := uc.repo.LoadBills(ctx, organizationID, input.BillIDs)
	if err != nil {
		return nil, err
	}
	invoice, err := buildFinanceInvoice(organizationID, bills, input)
	if err != nil {
		return nil, err
	}
	invoice.RecordNo, err = uc.config.NextNumber(ctx, organizationID, DocumentTypeInvoice)
	if err != nil {
		return nil, err
	}
	return uc.repo.Create(ctx, invoice, financeInvoiceAudit(organizationID, actorID, invoice.ID, "finance.invoice.create"))
}

func (uc *FinanceInvoiceUsecase) Issue(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, taxInvoiceNo, invoiceDate string) (*FinanceInvoice, error) {
	taxInvoiceNo, invoiceDate = strings.TrimSpace(taxInvoiceNo), strings.TrimSpace(invoiceDate)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || taxInvoiceNo == "" || utf8.RuneCountInString(taxInvoiceNo) > 100 || !validFinanceDate(invoiceDate) {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.Issue(ctx, organizationID, id, actorID, expectedVersion, taxInvoiceNo, invoiceDate, financeInvoiceAudit(organizationID, actorID, id, "finance.invoice.issue"))
}

func (uc *FinanceInvoiceUsecase) Cancel(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceInvoice, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.Cancel(ctx, organizationID, id, actorID, expectedVersion, reason, financeInvoiceAudit(organizationID, actorID, id, "finance.invoice.cancel"))
}

func buildFinanceInvoice(organizationID uuid.UUID, bills []*FinanceBill, input CreateFinanceInvoiceInput) (*FinanceInvoice, error) {
	if len(bills) != len(input.BillIDs) || len(bills) == 0 || bills[0].Status != FinanceBillConfirmed {
		return nil, ErrFinanceInvoiceBillInvalid
	}
	first := bills[0]
	id := uuid.Must(uuid.NewV7())
	invoice := &FinanceInvoice{ID: id, OrganizationID: organizationID, IdempotencyKey: input.IdempotencyKey, Direction: first.Direction, Status: FinanceInvoiceDraft, InvoiceType: input.InvoiceType, SettlementPartyID: first.SettlementPartyID, SettlementPartyName: first.SettlementPartyName, Currency: first.Currency, Note: input.Note, Version: 1, Links: make([]*FinanceInvoiceBill, 0, len(bills))}
	for _, bill := range bills {
		if bill.Status != FinanceBillConfirmed {
			return nil, ErrFinanceInvoiceBillInvalid
		}
		if bill.Direction != invoice.Direction || bill.SettlementPartyID != invoice.SettlementPartyID || bill.Currency != invoice.Currency {
			return nil, ErrFinanceInvoiceBillMismatch
		}
		invoice.TotalAmount = invoice.TotalAmount.Add(bill.TotalAmount)
		invoice.TaxAmount = invoice.TaxAmount.Add(bill.TaxAmount)
		invoice.Links = append(invoice.Links, &FinanceInvoiceBill{ID: uuid.Must(uuid.NewV7()), InvoiceID: id, BillID: bill.ID, BillNo: bill.BillNo, Amount: bill.TotalAmount, TaxAmount: bill.TaxAmount, Active: true})
	}
	invoice.BillCount = len(invoice.Links)
	return invoice, nil
}

func financeInvoiceAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_invoice", ResourceID: id.String(), Details: map[string]string{"finance_invoice.id": id.String()}}
}
