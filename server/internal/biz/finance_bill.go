package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
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
	ErrFinanceBillNotFound            = errors.NotFound("FINANCE_BILL_NOT_FOUND", "账单不存在")
	ErrFinanceBillInvalidArgument     = errors.BadRequest("FINANCE_BILL_INVALID_ARGUMENT", "账单字段不合法")
	ErrFinanceBillFeeInvalid          = errors.Conflict("FINANCE_BILL_FEE_INVALID", "所选费用必须为已确认状态且尚未进入其他账单")
	ErrFinanceBillFeeMismatch         = errors.BadRequest("FINANCE_BILL_FEE_MISMATCH", "同一账单的费用必须具有相同收付方向、结算单位、币种和本币")
	ErrFinanceBillVersionConflict     = errors.Conflict("FINANCE_BILL_VERSION_CONFLICT", "账单已被其他操作人修改，请刷新后重试")
	ErrFinanceBillInvalidTransition   = errors.Conflict("FINANCE_BILL_INVALID_TRANSITION", "当前账单状态不允许执行该操作")
	ErrFinanceBillIdempotencyConflict = errors.Conflict("FINANCE_BILL_IDEMPOTENCY_CONFLICT", "账单请求幂等键已被其他请求使用")
	ErrFinanceBillPreviewStale        = errors.Conflict("FINANCE_BILL_PREVIEW_STALE", "费用或拆单结果已变化，请重新预览")
	ErrFinanceBillBatchMismatch       = errors.BadRequest("FINANCE_BILL_BATCH_MISMATCH", "批量账单分组资料与服务端预览不一致")
	ErrFinanceBillBatchConflict       = errors.Conflict("FINANCE_BILL_BATCH_CONFLICT", "批量建单幂等键已被其他请求使用")
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
	BatchID             *uuid.UUID
	BatchNo             string
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
	StatementTitle      *string
	PaymentTermsDays    *int
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
	TaxRate            *decimal.Decimal
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
	FeeIDs           []uuid.UUID
	BillDate         string
	DueDate          *string
	Note             *string
	StatementTitle   *string
	PaymentTermsDays *int
	IdempotencyKey   string
}

type UpdateFinanceBillInput struct {
	ID               uuid.UUID
	BillDate         string
	DueDate          *string
	Note             *string
	StatementTitle   *string
	PaymentTermsDays *int
	ExpectedVersion  uint64
}

type FinanceBillGroupingPolicy struct {
	SplitByOrder   bool
	SplitByTaxRate bool
}

type FinanceBillBatchPreviewGroup struct {
	GroupKey, SettlementPartyName, Currency, BaseCurrency string
	Direction                                             OrderFeeDirection
	SettlementPartyID                                     uuid.UUID
	OrderID                                               *uuid.UUID
	OrderNo                                               *string
	TaxRate                                               *decimal.Decimal
	Fees                                                  []*FinanceBillableFee
	TotalAmount, NetAmount, TaxAmount, BaseCurrencyAmount decimal.Decimal
}

type FinanceBillBatchPreview struct {
	Groups       []*FinanceBillBatchPreviewGroup
	PreviewToken string
}

type CreateFinanceBillBatchGroupInput struct {
	GroupKey         string
	StatementTitle   string
	BillDate         string
	DueDate          *string
	PaymentTermsDays *int
	Note             *string
}

type CreateFinanceBillBatchInput struct {
	FeeIDs         []uuid.UUID
	GroupingPolicy FinanceBillGroupingPolicy
	Groups         []CreateFinanceBillBatchGroupInput
	PreviewToken   string
	IdempotencyKey string
}

type PreviewFinanceBillBatchInput struct {
	FeeIDs         []uuid.UUID
	GroupingPolicy FinanceBillGroupingPolicy
}

type FinanceBillBatch struct {
	ID, OrganizationID, CreatedBy        uuid.UUID
	BatchNo, IdempotencyKey, RequestHash string
	GroupingPolicy                       FinanceBillGroupingPolicy
	FeeCount, BillCount                  int
	TotalBaseAmount                      decimal.Decimal
	BaseCurrency                         string
	Bills                                []*FinanceBill
	CreatedAt, UpdatedAt                 time.Time
}

type FinanceBillRepo interface {
	List(ctx context.Context, organizationID uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error)
	Get(ctx context.Context, organizationID, id uuid.UUID) (*FinanceBill, error)
	GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBill, error)
	GetBatchByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBillBatch, error)
	LoadBillableFees(ctx context.Context, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error)
	Create(ctx context.Context, bill *FinanceBill, audit *AuditEvent) (*FinanceBill, error)
	CreateBatch(ctx context.Context, batch *FinanceBillBatch, previewToken string, audit *AuditEvent) (*FinanceBillBatch, error)
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

func (uc *FinanceBillUsecase) PreviewBatch(ctx context.Context, organizationID uuid.UUID, input PreviewFinanceBillBatchInput) (*FinanceBillBatchPreview, error) {
	feeIDs, err := normalizeFinanceBillFeeIDs(input.FeeIDs)
	if err != nil || organizationID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	fees, err := uc.repo.LoadBillableFees(ctx, organizationID, feeIDs)
	if err != nil {
		return nil, err
	}
	if len(fees) != len(feeIDs) {
		return nil, ErrFinanceBillFeeInvalid
	}
	return BuildFinanceBillBatchPreview(organizationID, fees, input.GroupingPolicy)
}

func (uc *FinanceBillUsecase) CreateBatch(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput) (*FinanceBillBatch, error) {
	feeIDs, err := normalizeFinanceBillFeeIDs(input.FeeIDs)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	input.FeeIDs = feeIDs
	input.PreviewToken = strings.TrimSpace(input.PreviewToken)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.PreviewToken == "" || len(input.PreviewToken) != 64 || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || len(input.Groups) == 0 || len(input.Groups) > len(input.FeeIDs) {
		return nil, ErrFinanceBillInvalidArgument
	}
	normalizedGroups, err := normalizeFinanceBillBatchGroups(input.Groups)
	if err != nil {
		return nil, err
	}
	input.Groups = normalizedGroups
	requestHash := financeBillBatchRequestHash(input)
	if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr != nil {
		return nil, lookupErr
	} else if existing != nil {
		if existing.RequestHash == requestHash {
			return existing, nil
		}
		return nil, ErrFinanceBillBatchConflict
	}
	fees, err := uc.repo.LoadBillableFees(ctx, organizationID, input.FeeIDs)
	if err != nil {
		return nil, err
	}
	if len(fees) != len(input.FeeIDs) {
		return nil, ErrFinanceBillFeeInvalid
	}
	preview, err := BuildFinanceBillBatchPreview(organizationID, fees, input.GroupingPolicy)
	if err != nil {
		return nil, err
	}
	if preview.PreviewToken != input.PreviewToken {
		return nil, ErrFinanceBillPreviewStale
	}
	groupInputs := make(map[string]CreateFinanceBillBatchGroupInput, len(input.Groups))
	for _, group := range input.Groups {
		groupInputs[group.GroupKey] = group
	}
	batchID := uuid.Must(uuid.NewV7())
	batch := &FinanceBillBatch{ID: batchID, OrganizationID: organizationID, CreatedBy: actorID, IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash, GroupingPolicy: input.GroupingPolicy, FeeCount: len(input.FeeIDs), BillCount: len(preview.Groups), Bills: make([]*FinanceBill, 0, len(preview.Groups))}
	for _, previewGroup := range preview.Groups {
		groupInput, ok := groupInputs[previewGroup.GroupKey]
		if !ok {
			return nil, ErrFinanceBillBatchMismatch
		}
		delete(groupInputs, previewGroup.GroupKey)
		billInput := CreateFinanceBillInput{FeeIDs: financeBillableFeeIDs(previewGroup.Fees), BillDate: groupInput.BillDate, DueDate: groupInput.DueDate, Note: groupInput.Note, StatementTitle: &groupInput.StatementTitle, PaymentTermsDays: groupInput.PaymentTermsDays, IdempotencyKey: financeBillBatchBillKey(input.IdempotencyKey, previewGroup.GroupKey)}
		billInput, err = normalizeCreateFinanceBill(billInput)
		if err != nil {
			return nil, err
		}
		bill, buildErr := buildFinanceBill(organizationID, previewGroup.Fees, billInput)
		if buildErr != nil {
			return nil, buildErr
		}
		bill.BatchID = &batchID
		batch.Bills = append(batch.Bills, bill)
		batch.TotalBaseAmount = batch.TotalBaseAmount.Add(bill.BaseCurrencyAmount)
		if batch.BaseCurrency == "" {
			batch.BaseCurrency = bill.BaseCurrency
		} else if batch.BaseCurrency != bill.BaseCurrency {
			return nil, ErrFinanceBillBatchMismatch
		}
	}
	if len(groupInputs) != 0 {
		return nil, ErrFinanceBillBatchMismatch
	}
	batch.TotalBaseAmount = batch.TotalBaseAmount.Round(8)
	created, err := uc.repo.CreateBatch(ctx, batch, preview.PreviewToken, financeBillBatchAudit(organizationID, actorID, batchID, "finance.bill_batch.create"))
	if err == nil {
		return created, nil
	}
	if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr == nil && existing != nil && existing.RequestHash == requestHash {
		return existing, nil
	}
	return nil, err
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
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.DueDate = normalizedFinanceBillDueDate(input.BillDate, input.DueDate, input.PaymentTermsDays)
	if organizationID == uuid.Nil || actorID == uuid.Nil || input.ID == uuid.Nil || input.ExpectedVersion == 0 || !validFinanceDate(input.BillDate) || !validFinanceBillTerms(input.BillDate, input.DueDate, input.PaymentTermsDays) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
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
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.DueDate = normalizedFinanceBillDueDate(input.BillDate, input.DueDate, input.PaymentTermsDays)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.FeeIDs) == 0 || len(input.FeeIDs) > 500 || !validFinanceDate(input.BillDate) || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || !validFinanceBillTerms(input.BillDate, input.DueDate, input.PaymentTermsDays) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
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
			TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, Currency: fee.Currency,
			TaxRate:      fee.TaxRate,
			ExchangeRate: fee.ExchangeRate, BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount, Active: true,
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
	if existing.BillDate != requested.BillDate || !stringPointersEqual(existing.DueDate, requested.DueDate) || !stringPointersEqual(existing.Note, requested.Note) || !stringPointersEqual(existing.StatementTitle, requestedTitle) || !intPointersEqual(existing.PaymentTermsDays, requested.PaymentTermsDays) || len(existing.Lines) != len(requested.FeeIDs) {
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

func normalizeFinanceBillBatchGroups(groups []CreateFinanceBillBatchGroupInput) ([]CreateFinanceBillBatchGroupInput, error) {
	result := make([]CreateFinanceBillBatchGroupInput, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, item := range groups {
		item.GroupKey = strings.TrimSpace(item.GroupKey)
		item.StatementTitle = strings.TrimSpace(item.StatementTitle)
		item.BillDate = strings.TrimSpace(item.BillDate)
		item.DueDate = normalizedOptionalFinanceString(item.DueDate)
		item.Note = normalizedOptionalFinanceString(item.Note)
		item.DueDate = normalizedFinanceBillDueDate(item.BillDate, item.DueDate, item.PaymentTermsDays)
		if item.GroupKey == "" || len(item.GroupKey) != 64 || item.StatementTitle == "" || utf8.RuneCountInString(item.StatementTitle) > 200 || !validFinanceDate(item.BillDate) || !validFinanceBillTerms(item.BillDate, item.DueDate, item.PaymentTermsDays) || (item.Note != nil && utf8.RuneCountInString(*item.Note) > 500) {
			return nil, ErrFinanceBillInvalidArgument
		}
		if _, exists := seen[item.GroupKey]; exists {
			return nil, ErrFinanceBillBatchMismatch
		}
		seen[item.GroupKey] = struct{}{}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GroupKey < result[j].GroupKey })
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

// BuildFinanceBillBatchPreview 按固定结算维度和可选策略生成确定性的服务端拆单预览。
func BuildFinanceBillBatchPreview(organizationID uuid.UUID, fees []*FinanceBillableFee, policy FinanceBillGroupingPolicy) (*FinanceBillBatchPreview, error) {
	if organizationID == uuid.Nil || len(fees) == 0 || len(fees) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	ordered := append([]*FinanceBillableFee(nil), fees...)
	for _, item := range ordered {
		if item == nil || item.Fee == nil || item.Fee.ID == uuid.Nil || item.Fee.OrderID == uuid.Nil || item.Fee.Status != OrderFeeConfirmed || item.Fee.TaxRate == nil || !financeBillCurrencyPattern.MatchString(item.Fee.Currency) || !financeBillCurrencyPattern.MatchString(item.Fee.BaseCurrency) {
			return nil, ErrFinanceBillFeeInvalid
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Fee.ID.String() < ordered[j].Fee.ID.String()
	})
	groupsByRawKey := make(map[string]*FinanceBillBatchPreviewGroup)
	rawKeys := make([]string, 0)
	tokenSource := strings.Builder{}
	writeFinanceHashParts(&tokenSource, organizationID.String(), strconv.FormatBool(policy.SplitByOrder), strconv.FormatBool(policy.SplitByTaxRate))
	seen := make(map[uuid.UUID]struct{}, len(ordered))
	for _, item := range ordered {
		fee := item.Fee
		if _, exists := seen[fee.ID]; exists {
			return nil, ErrFinanceBillInvalidArgument
		}
		seen[fee.ID] = struct{}{}
		taxRate := fee.TaxRate.StringFixed(4)
		parts := []string{string(fee.Direction), fee.SettlementPartyID.String(), fee.Currency, fee.BaseCurrency}
		if policy.SplitByOrder {
			parts = append(parts, fee.OrderID.String())
		}
		if policy.SplitByTaxRate {
			parts = append(parts, taxRate)
		}
		rawKey := strings.Join(parts, "\x00")
		group := groupsByRawKey[rawKey]
		if group == nil {
			group = &FinanceBillBatchPreviewGroup{GroupKey: financeSHA256(rawKey), Direction: fee.Direction, SettlementPartyID: fee.SettlementPartyID, SettlementPartyName: fee.SettlementPartyName, Currency: fee.Currency, BaseCurrency: fee.BaseCurrency, Fees: make([]*FinanceBillableFee, 0)}
			if policy.SplitByOrder {
				orderID, orderNo := fee.OrderID, item.OrderNo
				group.OrderID, group.OrderNo = &orderID, &orderNo
			}
			if policy.SplitByTaxRate {
				rate := *fee.TaxRate
				group.TaxRate = &rate
			}
			groupsByRawKey[rawKey] = group
			rawKeys = append(rawKeys, rawKey)
		}
		group.Fees = append(group.Fees, item)
		group.TotalAmount = group.TotalAmount.Add(fee.TotalAmount)
		group.NetAmount = group.NetAmount.Add(fee.NetAmount)
		group.TaxAmount = group.TaxAmount.Add(fee.TaxAmount)
		group.BaseCurrencyAmount = group.BaseCurrencyAmount.Add(fee.BaseCurrencyAmount)
		writeFinanceHashParts(
			&tokenSource,
			fee.ID.String(), fee.OrderID.String(), item.OrderNo, item.BusinessType,
			strconv.FormatUint(fee.Version, 10), string(fee.Status), string(fee.Direction),
			fee.SettlementPartyID.String(), fee.SettlementPartyName, fee.FeeCode, fee.FeeName,
			fee.Currency, fee.BaseCurrency, fee.TotalAmount.StringFixed(8), fee.NetAmount.StringFixed(8),
			fee.TaxAmount.StringFixed(8), taxRate, fee.ExchangeRate.StringFixed(8), fee.BaseCurrencyAmount.StringFixed(8),
		)
	}
	sort.Strings(rawKeys)
	result := &FinanceBillBatchPreview{Groups: make([]*FinanceBillBatchPreviewGroup, 0, len(rawKeys)), PreviewToken: financeSHA256(tokenSource.String())}
	for _, rawKey := range rawKeys {
		group := groupsByRawKey[rawKey]
		group.TotalAmount = group.TotalAmount.Round(8)
		group.NetAmount = group.NetAmount.Round(8)
		group.TaxAmount = group.TaxAmount.Round(8)
		group.BaseCurrencyAmount = group.BaseCurrencyAmount.Round(8)
		result.Groups = append(result.Groups, group)
	}
	return result, nil
}

func financeBillBatchRequestHash(input CreateFinanceBillBatchInput) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, input.PreviewToken, strconv.FormatBool(input.GroupingPolicy.SplitByOrder), strconv.FormatBool(input.GroupingPolicy.SplitByTaxRate))
	for _, id := range input.FeeIDs {
		writeFinanceHashParts(&builder, id.String())
	}
	for _, group := range input.Groups {
		dueDate := ""
		if group.DueDate != nil {
			dueDate = *group.DueDate
		}
		paymentTermsDays := ""
		if group.PaymentTermsDays != nil {
			paymentTermsDays = strconv.Itoa(*group.PaymentTermsDays)
		}
		note := ""
		if group.Note != nil {
			note = *group.Note
		}
		writeFinanceHashParts(&builder, group.GroupKey, group.StatementTitle, group.BillDate, dueDate, paymentTermsDays, note)
	}
	return financeSHA256(builder.String())
}

func financeBillBatchBillKey(batchKey, groupKey string) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, batchKey, groupKey)
	return "batch-bill:" + financeSHA256(builder.String())
}

func financeSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func writeFinanceHashParts(builder *strings.Builder, values ...string) {
	for _, value := range values {
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
	}
}

func financeBillableFeeIDs(fees []*FinanceBillableFee) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(fees))
	for _, item := range fees {
		result = append(result, item.Fee.ID)
	}
	return result
}

func financeBillBatchAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_bill_batch", ResourceID: id.String(), Details: map[string]string{"finance_bill_batch.id": id.String()}}
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
