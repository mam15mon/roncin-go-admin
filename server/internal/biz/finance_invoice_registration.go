package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const MaxInvoiceRegistrationRecords = 100
const MaxInvoiceRegistrationFees = 5000

var ErrFinanceInvoiceSettlementDependency = errors.Conflict("FINANCE_INVOICE_SETTLEMENT_DEPENDENCY", "关联账单已有有效核销或已确认对冲，请先核对影响并按权限完成必要反向处理")
var ErrFinanceInvoiceAllocationExceeded = errors.Conflict("FINANCE_INVOICE_ALLOCATION_EXCEEDED", "发票分配超过费用可登记余额，请刷新后重试")

type FinanceInvoiceFeeInput struct {
	OrderFeeID        uuid.UUID
	Amount, TaxAmount decimal.Decimal
}

type FinanceInvoiceAllocationInput struct {
	BillID          uuid.UUID
	ExpectedVersion uint64
	Amount          decimal.Decimal
	Fees            []FinanceInvoiceFeeInput
}

type FinanceInvoiceRegistrationInput struct {
	InvoiceProfileID          uuid.UUID
	InvoiceType               FinanceInvoiceType
	TaxInvoiceNo, InvoiceDate string
	TotalAmount               decimal.Decimal
	Note                      *string
	Allocations               []FinanceInvoiceAllocationInput
}

type RegisterFinanceInvoicesInput struct {
	IdempotencyKey string
	Records        []FinanceInvoiceRegistrationInput
}

type FinanceInvoiceBillVersion struct {
	BillID          uuid.UUID
	ExpectedVersion uint64
}

type FinanceInvoiceRevisionEntry struct {
	ID              uuid.UUID
	ExpectedVersion uint64
	Remove          bool
	Registration    *FinanceInvoiceRegistrationInput
}

type ReviseFinanceInvoiceRegistrationsInput struct {
	IdempotencyKey, Reason string
	BillVersions           []FinanceInvoiceBillVersion
	Entries                []FinanceInvoiceRevisionEntry
}

type FinanceInvoiceCommand struct {
	ID, OrganizationID                       uuid.UUID
	Kind, IdempotencyKey, RequestFingerprint string
	InvoiceIDs                               []uuid.UUID
	Completed                                bool
}

type FinanceInvoiceFeeAllocation struct {
	ID, InvoiceID, InvoiceBillID, BillID, BillLineID, OrderFeeID uuid.UUID
	FeeCode, FeeName, Currency                                   string
	SourceTaxRate                                                *decimal.Decimal
	Amount, NetAmount, TaxAmount                                 decimal.Decimal
}

type FinanceInvoiceRevision struct {
	ID, InvoiceID, ActorID uuid.UUID
	ActorName, Reason      string
	RevisionNumber         uint64
	Before, After          json.RawMessage
	CreatedAt              time.Time
}

type FinanceInvoiceSettlementDependency struct {
	ID, BillID                         uuid.UUID
	Kind, DocumentNo, BillNo, Currency string
	Amount                             decimal.Decimal
}

type FinanceInvoiceRevisionImpact struct {
	Invoice      *FinanceInvoice
	Bills        []*FinanceBill
	Dependencies []*FinanceInvoiceSettlementDependency
	CanRevise    bool
}

// RegisterScoped 仅在事务提交后读取完整响应；组内组合调用 RegisterWithinTransaction。
func (uc *FinanceInvoiceUsecase) RegisterScoped(ctx context.Context, scope []uuid.UUID, actor uuid.UUID, input RegisterFinanceInvoicesInput) ([]*FinanceInvoice, error) {
	ids, err := uc.RegisterWithinTransaction(ctx, scope, actor, input)
	if err != nil {
		return nil, err
	}
	return uc.readRegisteredInvoices(ctx, scope, ids)
}

// RegisterWithinTransaction 只返回登记身份，不在写事务中发起完整响应快照查询。
func (uc *FinanceInvoiceUsecase) RegisterWithinTransaction(ctx context.Context, scope []uuid.UUID, actor uuid.UUID, input RegisterFinanceInvoicesInput) ([]uuid.UUID, error) {
	if len(scope) != 1 || scope[0] == uuid.Nil || actor == uuid.Nil || uc.transactor == nil || uc.config == nil || uc.exchangeRate == nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	if err := normalizeInvoiceRegistrations(&input); err != nil {
		return nil, err
	}
	hash := invoiceCommandFingerprint(input)
	var ids []uuid.UUID
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		command, err := uc.repo.BeginRegistrationCommand(txCtx, scope[0], actor, "REGISTER", input.IdempotencyKey, hash)
		if err != nil {
			return err
		}
		if command.Completed {
			ids = command.InvoiceIDs
			return nil
		}
		versions, err := registrationBillVersions(input.Records)
		if err != nil {
			return err
		}
		bills, err := uc.repo.LockRegistrationBills(txCtx, scope, versions, nil, false)
		if err != nil {
			return err
		}
		invoices, err := uc.buildRegistrationInvoices(txCtx, scope[0], input.Records, bills, nil)
		if err != nil {
			return err
		}
		ids, err = uc.repo.SaveRegistrations(txCtx, scope[0], actor, invoices, nil, "", command.ID)
		if err != nil {
			return err
		}
		return uc.repo.CompleteRegistrationCommand(txCtx, command.ID, ids)
	})
	return ids, err
}

func (uc *FinanceInvoiceUsecase) ReviseRegistrations(ctx context.Context, scope []uuid.UUID, actor uuid.UUID, input ReviseFinanceInvoiceRegistrationsInput) ([]*FinanceInvoice, error) {
	if len(scope) != 1 || scope[0] == uuid.Nil || actor == uuid.Nil || uc.transactor == nil || uc.config == nil || uc.exchangeRate == nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	input.IdempotencyKey, input.Reason = strings.TrimSpace(input.IdempotencyKey), strings.TrimSpace(input.Reason)
	if input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || input.Reason == "" || utf8.RuneCountInString(input.Reason) > 500 || len(input.Entries) == 0 || len(input.Entries) > MaxInvoiceRegistrationRecords {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	records := make([]FinanceInvoiceRegistrationInput, 0, len(input.Entries))
	seen := map[uuid.UUID]bool{}
	for index := range input.Entries {
		entry := &input.Entries[index]
		if entry.ID == uuid.Nil {
			if entry.Remove || entry.ExpectedVersion != 0 {
				return nil, ErrFinanceInvoiceInvalidArgument
			}
		} else {
			if entry.ExpectedVersion == 0 || seen[entry.ID] {
				return nil, ErrFinanceInvoiceInvalidArgument
			}
			seen[entry.ID] = true
		}
		if entry.Remove {
			if entry.Registration != nil {
				return nil, ErrFinanceInvoiceInvalidArgument
			}
			continue
		}
		if entry.Registration == nil {
			return nil, ErrFinanceInvoiceInvalidArgument
		}
		records = append(records, *entry.Registration)
	}
	normalized := RegisterFinanceInvoicesInput{IdempotencyKey: input.IdempotencyKey, Records: records}
	if len(records) > 0 {
		if err := normalizeInvoiceRegistrations(&normalized); err != nil {
			return nil, err
		}
	}
	at := 0
	for index := range input.Entries {
		if !input.Entries[index].Remove {
			record := normalized.Records[at]
			input.Entries[index].Registration = &record
			at++
		}
	}
	if len(input.BillVersions) == 0 || len(input.BillVersions) > MaxInvoiceRegistrationFees {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	sort.Slice(input.BillVersions, func(i, j int) bool {
		return input.BillVersions[i].BillID.String() < input.BillVersions[j].BillID.String()
	})
	for i, version := range input.BillVersions {
		if version.BillID == uuid.Nil || version.ExpectedVersion == 0 || (i > 0 && input.BillVersions[i-1].BillID == version.BillID) {
			return nil, ErrFinanceInvoiceInvalidArgument
		}
	}
	var ids []uuid.UUID
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		command, err := uc.repo.BeginRegistrationCommand(txCtx, scope[0], actor, "REVISE", input.IdempotencyKey, invoiceCommandFingerprint(input))
		if err != nil {
			return err
		}
		if command.Completed {
			ids = command.InvoiceIDs
			return nil
		}
		old, err := uc.repo.LoadRevisionInvoices(txCtx, scope, input.Entries)
		if err != nil {
			return err
		}
		oldByID := map[uuid.UUID]*FinanceInvoice{}
		for _, invoice := range old {
			oldByID[invoice.ID] = invoice
		}
		for _, entry := range input.Entries {
			if entry.ID != uuid.Nil && !entry.Remove {
				current := oldByID[entry.ID]
				if current == nil || current.InvoiceProfileID == nil || *current.InvoiceProfileID != entry.Registration.InvoiceProfileID {
					return ErrFinanceInvoiceProfileRequired
				}
			}
		}

		excluded := make([]uuid.UUID, 0, len(old))
		for _, invoice := range old {
			excluded = append(excluded, invoice.ID)
		}
		bills, err := uc.repo.LockRegistrationBills(txCtx, scope, input.BillVersions, excluded, true)
		if err != nil {
			return err
		}
		previousRecords := make([]*FinanceInvoice, 0, len(normalized.Records))
		for _, entry := range input.Entries {
			if !entry.Remove {
				previousRecords = append(previousRecords, oldByID[entry.ID])
			}
		}
		invoices, err := uc.buildRegistrationInvoices(txCtx, scope[0], normalized.Records, bills, previousRecords)
		if err != nil {
			return err
		}
		at := 0
		for _, entry := range input.Entries {
			if entry.Remove {
				continue
			}
			if entry.ID != uuid.Nil {
				invoices[at].ID = entry.ID
				invoices[at].Version = entry.ExpectedVersion
			}
			at++
		}
		ids, err = uc.repo.SaveRegistrations(txCtx, scope[0], actor, invoices, old, input.Reason, command.ID)
		if err != nil {
			return err
		}
		return uc.repo.CompleteRegistrationCommand(txCtx, command.ID, ids)
	})
	if err != nil {
		return nil, err
	}
	return uc.readRegisteredInvoices(ctx, scope, ids)
}

func (uc *FinanceInvoiceUsecase) GetRevisionImpact(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (*FinanceInvoiceRevisionImpact, error) {
	if !validFinanceOrganizationIDs(scope) || id == uuid.Nil {
		return nil, ErrFinanceInvoiceInvalidArgument
	}
	return uc.repo.GetRevisionImpact(ctx, scope, id)
}

func (uc *FinanceInvoiceUsecase) readRegisteredInvoices(ctx context.Context, scope []uuid.UUID, ids []uuid.UUID) ([]*FinanceInvoice, error) {
	out := make([]*FinanceInvoice, 0, len(ids))
	for _, id := range ids {
		value, err := uc.repo.GetScoped(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func normalizeInvoiceRegistrations(input *RegisterFinanceInvoicesInput) error {
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || len(input.Records) == 0 || len(input.Records) > MaxInvoiceRegistrationRecords {
		return ErrFinanceInvoiceInvalidArgument
	}
	count := 0
	numbers := map[string]bool{}
	for i := range input.Records {
		r := &input.Records[i]
		r.TaxInvoiceNo = strings.TrimSpace(r.TaxInvoiceNo)
		r.InvoiceDate = strings.TrimSpace(r.InvoiceDate)
		r.Note = normalizedOptionalFinanceString(r.Note)
		if r.InvoiceProfileID == uuid.Nil || (r.InvoiceType != FinanceInvoiceNormal && r.InvoiceType != FinanceInvoiceSpecial) || r.TaxInvoiceNo == "" || utf8.RuneCountInString(r.TaxInvoiceNo) > 100 || numbers[r.TaxInvoiceNo] || !validFinanceDate(r.InvoiceDate) || !invoiceAmountValid(r.TotalAmount, true) || len(r.Allocations) == 0 || len(r.Allocations) > 500 || (r.Note != nil && utf8.RuneCountInString(*r.Note) > 500) {
			return invoiceRegistrationError(ErrFinanceInvoiceInvalidArgument, i)
		}
		numbers[r.TaxInvoiceNo] = true
		bills := map[uuid.UUID]bool{}
		total := decimal.Zero
		for ai := range r.Allocations {
			a := &r.Allocations[ai]
			if a.BillID == uuid.Nil || a.ExpectedVersion == 0 || bills[a.BillID] || !invoiceAmountValid(a.Amount, true) || len(a.Fees) == 0 {
				return invoiceRegistrationError(ErrFinanceInvoiceInvalidArgument, i)
			}
			bills[a.BillID] = true
			fees := map[uuid.UUID]bool{}
			sum := decimal.Zero
			for _, f := range a.Fees {
				count++
				if count > MaxInvoiceRegistrationFees || f.OrderFeeID == uuid.Nil || fees[f.OrderFeeID] || !invoiceAmountValid(f.Amount, true) || !invoiceAmountValid(f.TaxAmount, false) || f.TaxAmount.GreaterThan(f.Amount) {
					return invoiceRegistrationError(ErrFinanceInvoiceInvalidArgument, i)
				}
				fees[f.OrderFeeID] = true
				sum = sum.Add(f.Amount)
			}
			if !sum.Equal(a.Amount) {
				return invoiceRegistrationError(ErrFinanceInvoiceInvalidArgument, i)
			}
			total = total.Add(a.Amount)
			sort.Slice(a.Fees, func(i, j int) bool { return a.Fees[i].OrderFeeID.String() < a.Fees[j].OrderFeeID.String() })
		}
		if !total.Equal(r.TotalAmount) {
			return invoiceRegistrationError(ErrFinanceInvoiceInvalidArgument, i)
		}
		sort.Slice(r.Allocations, func(i, j int) bool { return r.Allocations[i].BillID.String() < r.Allocations[j].BillID.String() })
	}
	return nil
}

func invoiceAmountValid(value decimal.Decimal, positive bool) bool {
	// 先限制指数，避免比较或生成请求摘要时将极大指数展开为巨大十进制数。
	return value.Exponent() >= -8 && value.Exponent() <= 19 && (!positive || value.IsPositive()) && !value.IsNegative() && value.Abs().LessThan(decimal.New(1, 20))
}
func invoiceCommandFingerprint(value any) string {
	payload, _ := json.Marshal(value)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func registrationBillVersions(records []FinanceInvoiceRegistrationInput) ([]FinanceInvoiceBillVersion, error) {
	byID := map[uuid.UUID]uint64{}
	for _, r := range records {
		for _, a := range r.Allocations {
			if prior, exists := byID[a.BillID]; exists && prior != a.ExpectedVersion {
				return nil, ErrFinanceBillVersionConflict
			}
			byID[a.BillID] = a.ExpectedVersion
		}
	}
	out := make([]FinanceInvoiceBillVersion, 0, len(byID))
	for id, v := range byID {
		out = append(out, FinanceInvoiceBillVersion{BillID: id, ExpectedVersion: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BillID.String() < out[j].BillID.String() })
	return out, nil
}

func (uc *FinanceInvoiceUsecase) buildRegistrationInvoices(ctx context.Context, org uuid.UUID, records []FinanceInvoiceRegistrationInput, bills []*FinanceBill, previous []*FinanceInvoice) ([]*FinanceInvoice, error) {
	byID := map[uuid.UUID]*FinanceBill{}
	remaining := map[uuid.UUID]decimal.Decimal{}
	for _, b := range bills {
		byID[b.ID] = b
		for _, l := range b.Lines {
			if l.InvoiceableAmount != nil {
				remaining[l.ID] = *l.InvoiceableAmount
			} else {
				remaining[l.ID] = l.TotalAmount
			}
		}
	}
	out := make([]*FinanceInvoice, 0, len(records))
	for recordIndex, r := range records {
		first := byID[r.Allocations[0].BillID]
		if first == nil {
			return nil, invoiceRegistrationError(ErrFinanceInvoiceBillInvalid, recordIndex)
		}
		var profile *PartnerInvoiceProfile
		var prior *FinanceInvoice
		if recordIndex < len(previous) {
			prior = previous[recordIndex]
		}
		if prior != nil && prior.InvoiceProfileID != nil && *prior.InvoiceProfileID == r.InvoiceProfileID {
			profile = &PartnerInvoiceProfile{ID: *prior.InvoiceProfileID, InvoiceTitle: prior.InvoiceTitle, TaxpayerIdentificationNo: prior.TaxpayerIdentificationNo, RegisteredAddress: prior.RegisteredAddress, RegisteredPhone: prior.RegisteredPhone, BankName: prior.BankName, BankAccount: prior.BankAccount, Enabled: true}
		}
		var err error
		if profile == nil {
			profile, err = uc.repo.LoadInvoiceProfile(ctx, org, first.SettlementPartyID, r.InvoiceProfileID)
		}
		if err != nil {
			return nil, invoiceRegistrationError(ErrFinanceInvoiceProfileRequired, recordIndex)
		}
		if profile.InvoiceTitle == "" || profile.TaxpayerIdentificationNo == "" || !profile.Enabled || (r.InvoiceType == FinanceInvoiceSpecial && (profile.RegisteredAddress == "" || profile.RegisteredPhone == "" || profile.BankName == "" || profile.BankAccount == "")) {
			return nil, invoiceRegistrationError(ErrFinanceInvoiceProfileRequired, recordIndex)
		}
		id := uuid.Must(uuid.NewV7())
		inv := &FinanceInvoice{ID: id, OrganizationID: org, InvoiceProfileID: &r.InvoiceProfileID, Direction: first.Direction, Status: FinanceInvoiceIssued, InvoiceType: r.InvoiceType, SettlementPartyID: first.SettlementPartyID, SettlementPartyName: first.SettlementPartyName, Currency: first.Currency, BaseCurrency: first.BaseCurrency, InvoiceTitle: profile.InvoiceTitle, TaxpayerIdentificationNo: profile.TaxpayerIdentificationNo, RegisteredAddress: profile.RegisteredAddress, RegisteredPhone: profile.RegisteredPhone, BankName: profile.BankName, BankAccount: profile.BankAccount, TaxInvoiceNo: &r.TaxInvoiceNo, InvoiceDate: &r.InvoiceDate, Note: r.Note, Version: 1, TotalAmount: r.TotalAmount}
		for _, a := range r.Allocations {
			b := byID[a.BillID]
			if b == nil || b.OrganizationID != org || b.Status != FinanceBillConfirmed || b.Purpose != FinanceBillNormal {
				return nil, invoiceRegistrationError(ErrFinanceInvoiceBillInvalid, recordIndex)
			}
			if b.Direction != inv.Direction || b.SettlementPartyID != inv.SettlementPartyID || b.Currency != inv.Currency || b.BaseCurrency != inv.BaseCurrency {
				return nil, invoiceRegistrationError(ErrFinanceInvoiceBillMismatch, recordIndex)
			}
			link := &FinanceInvoiceBill{ID: uuid.Must(uuid.NewV7()), InvoiceID: id, BillID: b.ID, BillNo: b.BillNo, Amount: a.Amount, Active: true}
			lineMap := map[uuid.UUID]*FinanceBillLine{}
			for _, l := range b.Lines {
				lineMap[l.OrderFeeID] = l
			}
			for _, f := range a.Fees {
				l := lineMap[f.OrderFeeID]
				if l == nil || !l.Active {
					return nil, invoiceRegistrationError(ErrFinanceInvoiceBillInvalid, recordIndex)
				}
				if f.Amount.GreaterThan(remaining[l.ID]) {
					return nil, invoiceRegistrationError(ErrFinanceInvoiceAllocationExceeded, recordIndex)
				}
				remaining[l.ID] = remaining[l.ID].Sub(f.Amount)
				rate := l.TaxRate
				allocation := &FinanceInvoiceFeeAllocation{ID: uuid.Must(uuid.NewV7()), InvoiceID: id, InvoiceBillID: link.ID, BillID: b.ID, BillLineID: l.ID, OrderFeeID: l.OrderFeeID, FeeCode: l.FeeCode, FeeName: l.FeeName, Currency: l.Currency, SourceTaxRate: rate, Amount: f.Amount, TaxAmount: f.TaxAmount, NetAmount: f.Amount.Sub(f.TaxAmount)}
				link.SourceFeeIDs = append(link.SourceFeeIDs, f.OrderFeeID)
				link.Fees = append(link.Fees, allocation)
				link.TaxAmount = link.TaxAmount.Add(f.TaxAmount)
				inv.TaxAmount = inv.TaxAmount.Add(f.TaxAmount)
				inv.Lines = append(inv.Lines, &FinanceInvoiceLine{ID: uuid.Must(uuid.NewV7()), InvoiceID: id, LineNo: len(inv.Lines) + 1, SourceLineCount: 1, ItemCode: l.FeeCode, ItemName: l.FeeName, Currency: l.Currency, TaxRate: rate, NetAmount: allocation.NetAmount, TaxAmount: allocation.TaxAmount, TotalAmount: allocation.Amount})
			}
			inv.Links = append(inv.Links, link)
		}
		inv.BillCount = len(inv.Links)
		inv.NetAmount = inv.TotalAmount.Sub(inv.TaxAmount)
		if prior != nil {
			inv.RecordNo = prior.RecordNo
		} else {
			inv.RecordNo, err = uc.config.NextNumber(ctx, org, DocumentTypeInvoice)
			if err != nil {
				return nil, invoiceRegistrationError(err, recordIndex)
			}
		}
		resolved, err := uc.exchangeRate.ResolveRate(ctx, org, inv.Direction, inv.Currency, r.InvoiceDate)
		if err != nil {
			return nil, invoiceRegistrationError(err, recordIndex)
		}
		inv.ExchangeRate = &resolved.Rate
		inv.ExchangeRateSource = &resolved.Source
		inv.ExchangeRateDate = &resolved.Date
		inv.ExchangeRateSettingID = resolved.SettingID
		amount := inv.TotalAmount.Mul(resolved.Rate).RoundBank(8)
		inv.BaseCurrencyAmount = &amount
		out = append(out, inv)
	}
	return out, nil
}

// 附带项位置属于调用上下文，复制领域错误后附元数据，禁止修改共享错误常量。
func invoiceRegistrationError(err error, index int) error {
	source := errors.FromError(err)
	metadata := map[string]string{}
	for key, value := range source.Metadata {
		metadata[key] = value
	}
	metadata["invoice_index"] = strconv.Itoa(index)
	return errors.New(int(source.Code), source.Reason, source.Message).WithMetadata(metadata)
}
