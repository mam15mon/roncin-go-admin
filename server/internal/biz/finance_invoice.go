package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/shopspring/decimal"
)

var (
	ErrFinanceInvoiceNotFound            = errors.NotFound("FINANCE_INVOICE_NOT_FOUND", "开票记录不存在")
	ErrFinanceInvoiceInvalidArgument     = errors.BadRequest("FINANCE_INVOICE_INVALID_ARGUMENT", "开票记录字段不合法")
	ErrFinanceInvoiceBillInvalid         = errors.Conflict("FINANCE_INVOICE_BILL_INVALID", "所选账单必须已确认且存在可登记的有效费用余额")
	ErrFinanceInvoiceBillMismatch        = errors.BadRequest("FINANCE_INVOICE_BILL_MISMATCH", "同一开票记录的账单必须具有相同收付方向、结算单位和币种")
	ErrFinanceInvoiceVersionConflict     = errors.Conflict("FINANCE_INVOICE_VERSION_CONFLICT", "开票记录已被其他操作人修改，请刷新后重试")
	ErrFinanceInvoiceInvalidTransition   = errors.Conflict("FINANCE_INVOICE_INVALID_TRANSITION", "当前开票记录状态不允许执行该操作")
	ErrFinanceInvoiceIdempotencyConflict = errors.Conflict("FINANCE_INVOICE_IDEMPOTENCY_CONFLICT", "开票请求幂等键已被其他请求使用")
	ErrFinanceInvoiceProfileRequired     = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_INVOICE_PROFILE_REQUIRED), "请选择该结算单位下启用且完整的开票抬头")
	ErrFinanceInvoiceTaxNoExists         = errors.Conflict("FINANCE_INVOICE_TAX_NO_EXISTS", "税控发票号码已被其他开票记录使用")
	ErrFinanceInvoiceRedNoExists         = errors.Conflict("FINANCE_INVOICE_RED_NO_EXISTS", "红字发票号码已被其他开票记录使用")
)

type FinanceInvoiceStatus string
type FinanceInvoiceType string

const (
	FinanceInvoiceDraft      FinanceInvoiceStatus = "DRAFT"
	FinanceInvoiceIssued     FinanceInvoiceStatus = "ISSUED"
	FinanceInvoiceCancelled  FinanceInvoiceStatus = "CANCELLED"
	FinanceInvoiceRedFlushed FinanceInvoiceStatus = "RED_FLUSHED"
)

// 发票票种只用于登记记录与查询，不对接税务；专票类要求抬头开户资料齐全。
const (
	FinanceInvoiceDigitalNormal     FinanceInvoiceType = "DIGITAL_NORMAL"
	FinanceInvoiceDigitalSpecial    FinanceInvoiceType = "DIGITAL_SPECIAL"
	FinanceInvoicePaperNormal       FinanceInvoiceType = "PAPER_NORMAL"
	FinanceInvoicePaperSpecial      FinanceInvoiceType = "PAPER_SPECIAL"
	FinanceInvoiceElectronicNormal  FinanceInvoiceType = "ELECTRONIC_NORMAL"
	FinanceInvoiceElectronicSpecial FinanceInvoiceType = "ELECTRONIC_SPECIAL"
)

// Valid 判断是否为受支持票种。
func (t FinanceInvoiceType) Valid() bool {
	switch t {
	case FinanceInvoiceDigitalNormal, FinanceInvoiceDigitalSpecial, FinanceInvoicePaperNormal, FinanceInvoicePaperSpecial, FinanceInvoiceElectronicNormal, FinanceInvoiceElectronicSpecial:
		return true
	}
	return false
}

// Special 判断是否为专用发票类，需要抬头登记地址、电话、开户行与账号。
func (t FinanceInvoiceType) Special() bool {
	return t == FinanceInvoiceDigitalSpecial || t == FinanceInvoicePaperSpecial || t == FinanceInvoiceElectronicSpecial
}

type FinanceInvoice struct {
	ID, OrganizationID, SettlementPartyID  uuid.UUID
	InvoiceProfileID                       *uuid.UUID
	RecordNo, IdempotencyKey               string
	OrganizationName                       string
	Direction                              OrderFeeDirection
	Status                                 FinanceInvoiceStatus
	InvoiceType                            FinanceInvoiceType
	SettlementPartyName, Currency          string
	BaseCurrency                           string
	ExchangeRate                           *decimal.Decimal
	ExchangeRateSource, ExchangeRateDate   *string
	ExchangeRateSettingID                  *uuid.UUID
	BaseCurrencyAmount                     *decimal.Decimal
	InvoiceTitle, TaxpayerIdentificationNo string
	RegisteredAddress, RegisteredPhone     string
	BankName, BankAccount                  string
	TotalAmount, NetAmount, TaxAmount      decimal.Decimal
	BillCount                              int
	TaxInvoiceNo, InvoiceDate, Note        *string
	Version                                uint64
	RevisionNumber                         uint64
	IssuedAt, CancelledAt                  *time.Time
	IssuedBy, CancelledBy                  *uuid.UUID
	CancellationReason                     *string
	RedInvoiceNo, RedInvoiceDate           *string
	RedFlushedAt                           *time.Time
	RedFlushedBy                           *uuid.UUID
	RedFlushReason                         *string
	Links                                  []*FinanceInvoiceBill
	Lines                                  []*FinanceInvoiceLine
	Revisions                              []*FinanceInvoiceRevision
	CreatedAt, UpdatedAt                   time.Time
}

type FinanceInvoiceLine struct {
	ID, InvoiceID                     uuid.UUID
	LineNo, SourceLineCount           int
	ItemCode, ItemName, Currency      string
	TaxRate                           *decimal.Decimal
	NetAmount, TaxAmount, TotalAmount decimal.Decimal
}

type FinanceInvoiceBill struct {
	SourceFeeIDs          []uuid.UUID
	ID, InvoiceID, BillID uuid.UUID
	BillNo                string
	Amount, TaxAmount     decimal.Decimal
	Active                bool
	Fees                  []*FinanceInvoiceFeeAllocation
}

type FinanceInvoiceFilter struct {
	BillID         *uuid.UUID
	Page, PageSize int
	Keyword        string
	Direction      OrderFeeDirection
	Status         FinanceInvoiceStatus
}

type FinanceInvoiceListResult struct {
	MatchedSummary FinanceLedgerMatchedSummary
	Items          []*FinanceInvoice
	Total          int64
	Summary        FinanceInvoiceSummary
}

type FinanceInvoiceSummary struct {
	AmountsByBaseCurrency []FinanceBaseCurrencyAmount
	IssuedCount           int64
}

type FinanceInvoiceCreationBillFilter struct {
	BillID            *uuid.UUID
	Page, PageSize    int
	Keyword           string
	Direction         OrderFeeDirection
	SettlementPartyID *uuid.UUID
	Currency          string
}

type FinanceInvoiceCreationBillListResult struct {
	Items []*FinanceBill
	Total int64
}

type FinanceInvoiceProfilesForBill struct {
	OrganizationID, SettlementPartyID uuid.UUID
	Items                             []*PartnerInvoiceProfile
}

type FinanceInvoiceRepo interface {
	ListProfilesForParty(context.Context, []uuid.UUID, uuid.UUID) (*FinanceInvoiceProfilesForBill, error)
	BeginRegistrationCommand(context.Context, uuid.UUID, uuid.UUID, string, string, string) (*FinanceInvoiceCommand, error)
	CompleteRegistrationCommand(context.Context, uuid.UUID, []uuid.UUID) error
	LockRegistrationBills(context.Context, []uuid.UUID, []FinanceInvoiceBillVersion, []uuid.UUID, bool) ([]*FinanceBill, error)
	LoadRevisionInvoices(context.Context, []uuid.UUID, []FinanceInvoiceRevisionEntry) ([]*FinanceInvoice, error)
	SaveRegistrations(context.Context, uuid.UUID, uuid.UUID, []*FinanceInvoice, []*FinanceInvoice, string, uuid.UUID) ([]uuid.UUID, error)
	GetRevisionImpact(context.Context, []uuid.UUID, uuid.UUID) (*FinanceInvoiceRevisionImpact, error)

	SetBillInvoiceExemption(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, bool, *AuditEvent) error
	List(context.Context, uuid.UUID, FinanceInvoiceFilter) (*FinanceInvoiceListResult, error)
	ListScoped(context.Context, []uuid.UUID, FinanceInvoiceFilter) (*FinanceInvoiceListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceInvoice, error)
	GetScoped(context.Context, []uuid.UUID, uuid.UUID) (*FinanceInvoice, error)
	GetByIdempotencyKey(context.Context, uuid.UUID, string) (*FinanceInvoice, error)
	LoadBills(context.Context, uuid.UUID, []uuid.UUID) ([]*FinanceBill, error)
	LoadBillsScoped(context.Context, []uuid.UUID, []uuid.UUID) ([]*FinanceBill, error)
	ListCreationBills(context.Context, uuid.UUID, FinanceInvoiceCreationBillFilter) (*FinanceInvoiceCreationBillListResult, error)
	ListProfilesForBill(context.Context, []uuid.UUID, uuid.UUID) (*FinanceInvoiceProfilesForBill, error)
	LoadInvoiceProfile(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*PartnerInvoiceProfile, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceInvoice, error)
	RedFlush(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, string, string, *AuditEvent) (*FinanceInvoice, error)
}

type FinanceInvoiceUsecase struct {
	repo         FinanceInvoiceRepo
	config       *OrderConfigUsecase
	exchangeRate *ExchangeRateUsecase
	transactor   Transactor
}

func NewFinanceInvoiceUsecase(repo FinanceInvoiceRepo, config *OrderConfigUsecase, exchangeRate *ExchangeRateUsecase, transactor Transactor) *FinanceInvoiceUsecase {
	return &FinanceInvoiceUsecase{repo: repo, config: config, exchangeRate: exchangeRate, transactor: transactor}
}

func (uc *FinanceInvoiceUsecase) List(ctx context.Context, organizationID uuid.UUID, filter FinanceInvoiceFilter) (*FinanceInvoiceListResult, error) {
	return uc.ListScoped(ctx, []uuid.UUID{organizationID}, filter)
}

func (uc *FinanceInvoiceUsecase) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceInvoiceFilter) (*FinanceInvoiceListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if !validFinanceOrganizationIDs(organizationIDs) || !ValidListPagination(filter.Page, filter.PageSize) || (filter.BillID != nil && *filter.BillID == uuid.Nil) || utf8.RuneCountInString(filter.Keyword) > 100 || (filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable) || (filter.Status != "" && filter.Status != FinanceInvoiceDraft && filter.Status != FinanceInvoiceIssued && filter.Status != FinanceInvoiceCancelled && filter.Status != FinanceInvoiceRedFlushed) {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.ListScoped(ctx, organizationIDs, filter)
}

func (uc *FinanceInvoiceUsecase) ListCreationBills(ctx context.Context, organizationID uuid.UUID, filter FinanceInvoiceCreationBillFilter) (*FinanceInvoiceCreationBillListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if organizationID == uuid.Nil || !ValidListPagination(filter.Page, filter.PageSize) || (filter.BillID != nil && *filter.BillID == uuid.Nil) || utf8.RuneCountInString(filter.Keyword) > 100 ||
		(filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable) ||
		(filter.SettlementPartyID != nil && *filter.SettlementPartyID == uuid.Nil) ||
		(filter.Currency != "" && !financeBillCurrencyPattern.MatchString(filter.Currency)) {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.ListCreationBills(ctx, organizationID, filter)
}

func (uc *FinanceInvoiceUsecase) ListProfilesForBill(ctx context.Context, organizationIDs []uuid.UUID, billID uuid.UUID) (*FinanceInvoiceProfilesForBill, error) {
	if !validFinanceOrganizationIDs(organizationIDs) || billID == uuid.Nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.ListProfilesForBill(ctx, organizationIDs, billID)
}

func (uc *FinanceInvoiceUsecase) RedFlush(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, redInvoiceNo, redInvoiceDate, reason string) (*FinanceInvoice, error) {
	redInvoiceNo, redInvoiceDate, reason = strings.TrimSpace(redInvoiceNo), strings.TrimSpace(redInvoiceDate), strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || redInvoiceNo == "" || utf8.RuneCountInString(redInvoiceNo) > 100 || !validFinanceDate(redInvoiceDate) || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.RedFlush(ctx, organizationID, id, actorID, expectedVersion, redInvoiceNo, redInvoiceDate, reason, financeInvoiceAudit(organizationID, actorID, id, "finance.invoice.red_flush"))
}

func (uc *FinanceInvoiceUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*FinanceInvoice, error) {
	return uc.GetScoped(ctx, []uuid.UUID{organizationID}, id)
}

func (uc *FinanceInvoiceUsecase) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceInvoice, error) {
	if !validFinanceOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.GetScoped(ctx, organizationIDs, id)
}

func (uc *FinanceInvoiceUsecase) Cancel(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceInvoice, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.Cancel(ctx, organizationID, id, actorID, expectedVersion, reason, financeInvoiceAudit(organizationID, actorID, id, "finance.invoice.cancel"))
}

func financeInvoiceAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_invoice", ResourceID: id.String(), Details: map[string]string{"finance_invoice.id": id.String()}}
}

func (uc *FinanceInvoiceUsecase) ListProfilesForParty(ctx context.Context, scope []uuid.UUID, party uuid.UUID) (*FinanceInvoiceProfilesForBill, error) {
	if len(scope) != 1 || scope[0] == uuid.Nil || party == uuid.Nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.ListProfilesForParty(ctx, scope, party)
}
