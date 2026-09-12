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
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/shopspring/decimal"
)

var (
	ErrFinanceBillNotFound                 = errors.NotFound("FINANCE_BILL_NOT_FOUND", "账单不存在")
	ErrFinanceBillInvalidArgument          = errors.BadRequest("FINANCE_BILL_INVALID_ARGUMENT", "账单字段不合法")
	ErrFinanceBillFeeInvalid               = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_FEE_INVALID), "所选费用必须为已确认状态且尚未进入其他账单")
	ErrFinanceBillFeeMismatch              = errors.BadRequest("FINANCE_BILL_FEE_MISMATCH", "同一账单的费用必须具有相同收付方向、结算单位、币种和本币")
	ErrFinanceBillVersionConflict          = errors.Conflict("FINANCE_BILL_VERSION_CONFLICT", "账单已被其他操作人修改，请刷新后重试")
	ErrFinanceBillInvalidTransition        = errors.Conflict("FINANCE_BILL_INVALID_TRANSITION", "当前账单状态不允许执行该操作")
	ErrFinanceBillIdempotencyConflict      = errors.Conflict("FINANCE_BILL_IDEMPOTENCY_CONFLICT", "账单请求幂等键已被其他请求使用")
	ErrFinanceBillPreviewStale             = errors.Conflict(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_PREVIEW_STALE), "费用或拆单结果已变化，请重新预览")
	ErrFinanceBillBatchMismatch            = errors.BadRequest("FINANCE_BILL_BATCH_MISMATCH", "批量账单分组资料与服务端预览不一致")
	ErrFinanceBillBatchConflict            = errors.Conflict("FINANCE_BILL_BATCH_CONFLICT", "批量建单幂等键已被其他请求使用")
	ErrFinanceBillSettlementAccountInvalid = errors.BadRequest(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FINANCE_BILL_SETTLEMENT_ACCOUNT_INVALID), "结算账户与账单结算单位、方向、币种或启用状态不匹配")
	ErrFinanceBillMixedDirection           = errors.BadRequest("FINANCE_BILL_MIXED_DIRECTION", "普通账单需将应收、应付分别建账，请先完成一个方向，再创建另一个方向")
	ErrFinanceBillGroupingModeUnsupported  = errors.BadRequest("FINANCE_BILL_GROUPING_MODE_UNSUPPORTED", "当前阶段暂不支持对冲建账模式")
)

var financeBillCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type FinanceBillStatus string

const (
	FinanceBillDraft     FinanceBillStatus = "DRAFT"
	FinanceBillConfirmed FinanceBillStatus = "CONFIRMED"
	FinanceBillCancelled FinanceBillStatus = "CANCELLED"
)

type FinanceBill struct {
	ID                        uuid.UUID
	OrganizationID            uuid.UUID
	OrganizationName          string
	BatchID                   *uuid.UUID
	BatchNo                   string
	BillNo                    string
	IdempotencyKey            string
	Direction                 OrderFeeDirection
	Status                    FinanceBillStatus
	SettlementPartyID         uuid.UUID
	SettlementPartyName       string
	SettlementAccountID       uuid.UUID
	SettlementAccountName     string
	SettlementAccountHolder   string
	SettlementBankName        string
	SettlementBankAccount     string
	SettlementAccountCurrency string
	SettlementSwiftCode       string
	EstimatedInvoiceCurrency  *string
	EstimatedInvoiceRate      *decimal.Decimal
	EstimatedInvoiceAmount    *decimal.Decimal
	Currency                  string
	BaseCurrency              string
	ExchangeRate              decimal.Decimal
	ExchangeRateSource        string
	ExchangeRateDate          string
	ExchangeRateSettingID     *uuid.UUID
	TotalAmount               decimal.Decimal
	NetAmount                 decimal.Decimal
	TaxAmount                 decimal.Decimal
	BaseCurrencyAmount        decimal.Decimal
	VerifiedAmount            decimal.Decimal
	NettedAmount              decimal.Decimal
	UnverifiedAmount          decimal.Decimal
	OverdueDays               int32
	FeeCount                  int
	BillDate                  string
	StatementTitle            *string
	PaymentTermsDays          *int
	DueDate                   *string
	Note                      *string
	Version                   uint64
	ConfirmedAt               *time.Time
	ConfirmedBy               *uuid.UUID
	CancelledAt               *time.Time
	CancelledBy               *uuid.UUID
	CancellationReason        *string
	Lines                     []*FinanceBillLine
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
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
	Quantity           decimal.Decimal
	UnitPrice          decimal.Decimal
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
	Fee                     *OrderFee
	OrganizationID          uuid.UUID
	OrderNo                 string
	BusinessType            string
	SettlementPartyIsCasual bool
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
	TagIDs            []uuid.UUID
	DueDateFrom       string
	DueDateTo         string
	OnlyUnsettled     bool
	OnlyOverdue       bool
}

type FinanceBillCreationCandidateFilter struct {
	Page, PageSize int
	Keyword        string
	Direction      OrderFeeDirection
}
type FinanceBillCreationCandidateResult struct {
	Items []*FinanceBillableFee
	Total int64
}

type FinanceBillListResult struct {
	Items   []*FinanceBill
	Total   int64
	Summary FinanceBillSummary
}

type FinanceBillSummary struct {
	AmountsByBaseCurrency []FinanceBaseCurrencyAmount
}

// FinanceBaseCurrencyAmount 保持本位币边界；跨组织聚合不得直接相加不同币种金额。
type FinanceBaseCurrencyAmount struct {
	BaseCurrency                string
	ReceivableBaseAmount        decimal.Decimal
	PayableBaseAmount           decimal.Decimal
	UnverifiedBaseAmount        decimal.Decimal
	OverdueReceivableBaseAmount decimal.Decimal
}

type CreateFinanceBillInput struct {
	FeeIDs              []uuid.UUID
	BillDate            string
	DueDate             *string
	Note                *string
	StatementTitle      *string
	PaymentTermsDays    *int
	IdempotencyKey      string
	SettlementAccountID uuid.UUID
}

type UpdateFinanceBillInput struct {
	ID                       uuid.UUID
	BillDate                 string
	DueDate                  *string
	Note                     *string
	StatementTitle           *string
	PaymentTermsDays         *int
	ExpectedVersion          uint64
	ExchangeRate             decimal.Decimal
	ExchangeRateSource       string
	ExchangeRateDate         string
	ExchangeRateSettingID    *uuid.UUID
	BaseCurrencyAmount       decimal.Decimal
	SettlementAccountID      uuid.UUID
	EstimatedInvoiceCurrency *string
	EstimatedInvoiceRate     *decimal.Decimal
	EstimatedInvoiceAmount   *decimal.Decimal
	TotalAmount              decimal.Decimal
	NetAmount                decimal.Decimal
	TaxAmount                decimal.Decimal
	Lines                    []*FinanceBillLine
}

type FinanceBillGroupingPolicy struct {
	Mode           string
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
	ConfigurationComplete                                 bool
	TemporaryBillDate                                     bool
	BillDate                                              string
	SettlementAccountID                                   uuid.UUID
	EstimatedInvoiceCurrency                              string
	EstimatedInvoiceRate                                  decimal.Decimal
	EstimatedInvoiceAmount                                decimal.Decimal
	IsCasual                                              bool
	DefaultPaymentTermsDays                               *int
	preparedBill                                          *FinanceBill
	config                                                FinanceBillBatchPreviewGroupConfig
}

type FinanceBillBatchPreview struct {
	Groups       []*FinanceBillBatchPreviewGroup
	NettingPairs []*FinanceBillBatchNettingPair
	PreviewToken string
}

// FinanceBillBatchNettingPair 是对冲建账模式下同一结算单位、同一账单币种的抵销汇总；
// 金额只用双方共同账单币种计算，不产生对冲汇率或混合币种总额。
type FinanceBillBatchNettingPair struct {
	SettlementPartyID     uuid.UUID
	SettlementPartyName   string
	Currency              string
	ReceivableGrossAmount decimal.Decimal
	PayableGrossAmount    decimal.Decimal
	OffsetAmount          decimal.Decimal
	NetReceivableAmount   decimal.Decimal
	NetPayableAmount      decimal.Decimal
}

type CreateFinanceBillBatchGroupInput struct {
	GroupKey                 string
	StatementTitle           string
	BillDate                 string
	DueDate                  *string
	PaymentTermsDays         *int
	Note                     *string
	SettlementAccountID      uuid.UUID
	EstimatedInvoiceCurrency *string
	EstimatedInvoiceRate     *decimal.Decimal
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
	GroupConfigs   []FinanceBillBatchPreviewGroupConfig
}

// FinanceBillBatchPreviewGroupConfig 是一次预览的叶子配置真相；group_key 只由服务端分组事实生成。
type FinanceBillBatchPreviewGroupConfig struct {
	GroupKey                 string
	BillDate                 string
	SettlementAccountID      uuid.UUID
	EstimatedInvoiceCurrency *string
	EstimatedInvoiceRate     *decimal.Decimal
}

type FinanceBillBatch struct {
	ID, OrganizationID, CreatedBy        uuid.UUID
	BatchNo, IdempotencyKey, RequestHash string
	GroupingPolicy                       FinanceBillGroupingPolicy
	FeeCount, BillCount                  int
	TotalBaseAmount                      decimal.Decimal
	BaseCurrency                         string
	Bills                                []*FinanceBill
	Nettings                             []*FinanceNetting
	CreatedAt, UpdatedAt                 time.Time
}

type FinanceBillRepo interface {
	List(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error)
	Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceBill, error)
	GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBill, error)
	GetBatchByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBillBatch, error)
	GetBatch(ctx context.Context, organizationIDs []uuid.UUID, batchID uuid.UUID) (*FinanceBillBatch, error)
	ConfirmBatch(ctx context.Context, organizationIDs []uuid.UUID, batchID, actorID uuid.UUID, expectedVersions map[uuid.UUID]uint64, audit *AuditEvent) (*FinanceBillBatch, error)
	LoadBillableFees(ctx context.Context, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error)
	LoadBillableFeesScoped(ctx context.Context, organizationIDs []uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error)
	ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter FinanceBillCreationCandidateFilter) (*FinanceBillCreationCandidateResult, error)
	Create(ctx context.Context, bill *FinanceBill, audit *AuditEvent) (*FinanceBill, error)
	CreateBatch(ctx context.Context, batch *FinanceBillBatch, previewToken string, audit *AuditEvent, nettingAudits []*AuditEvent) (*FinanceBillBatch, error)
	ValidateBillCurrencies(ctx context.Context, currencies []string) error
	HydrateBillSettlementAccounts(ctx context.Context, bills []*FinanceBill) error
	Update(ctx context.Context, organizationIDs []uuid.UUID, input UpdateFinanceBillInput, audit *AuditEvent) (*FinanceBill, error)
	Confirm(ctx context.Context, organizationIDs []uuid.UUID, id, actorID uuid.UUID, expectedVersion uint64, audit *AuditEvent) (*FinanceBill, error)
	Cancel(ctx context.Context, organizationIDs []uuid.UUID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) (*FinanceBill, error)
}

type FinanceBillUsecase struct {
	repo         FinanceBillRepo
	exchangeRate *ExchangeRateUsecase
	transactor   Transactor
}

func NewFinanceBillUsecase(repo FinanceBillRepo, exchangeRate *ExchangeRateUsecase, transactor Transactor) *FinanceBillUsecase {
	return &FinanceBillUsecase{repo: repo, exchangeRate: exchangeRate, transactor: transactor}
}

func (uc *FinanceBillUsecase) List(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if !validFinanceBillOrganizationIDs(organizationIDs) || !ValidListPagination(filter.Page, filter.PageSize) || utf8.RuneCountInString(filter.Keyword) > 100 {
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
	if !validFinanceDateRange(filter.DueDateFrom, filter.DueDateTo) {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.List(ctx, organizationIDs, filter)
}

func (uc *FinanceBillUsecase) Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceBill, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.Get(ctx, organizationIDs, id)
}

func (uc *FinanceBillUsecase) ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter FinanceBillCreationCandidateFilter) (*FinanceBillCreationCandidateResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if organizationID == uuid.Nil || !ValidListPagination(filter.Page, filter.PageSize) || utf8.RuneCountInString(filter.Keyword) > 100 || (filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable) {
		return nil, ErrFinanceBillInvalidArgument
	}
	return uc.repo.ListCreationCandidates(ctx, organizationID, filter)
}

func validFinanceBillOrganizationIDs(organizationIDs []uuid.UUID) bool {
	if len(organizationIDs) == 0 {
		return false
	}
	for _, organizationID := range organizationIDs {
		if organizationID == uuid.Nil {
			return false
		}
	}
	return true
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
	return uc.buildConfiguredFinanceBillBatchPreview(ctx, organizationID, fees, input)
}

// BuildConfiguredFinanceBillBatchPreview 只构造确定性分组；完整预览由用例继续校验账户、币种并解析汇率。
// 该导出函数保留给纯分组测试使用，不能作为创建入口。
func BuildConfiguredFinanceBillBatchPreview(organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput) (*FinanceBillBatchPreview, error) {
	configs := make(map[string]FinanceBillBatchPreviewGroupConfig, len(input.GroupConfigs))
	for index := range input.GroupConfigs {
		config := input.GroupConfigs[index]
		config.GroupKey = strings.TrimSpace(config.GroupKey)
		config.BillDate = strings.TrimSpace(config.BillDate)
		if config.EstimatedInvoiceCurrency != nil {
			value := strings.ToUpper(strings.TrimSpace(*config.EstimatedInvoiceCurrency))
			config.EstimatedInvoiceCurrency = &value
		}
		if config.GroupKey == "" || (config.BillDate != "" && !validFinanceDate(config.BillDate)) || (config.EstimatedInvoiceCurrency != nil && !financeBillCurrencyPattern.MatchString(*config.EstimatedInvoiceCurrency)) || (config.EstimatedInvoiceRate != nil && (!config.EstimatedInvoiceRate.IsPositive() || config.EstimatedInvoiceRate.Exponent() < -8)) || (config.EstimatedInvoiceRate != nil && config.EstimatedInvoiceCurrency == nil) {
			return nil, ErrFinanceBillInvalidArgument
		}
		if _, exists := configs[config.GroupKey]; exists {
			return nil, ErrFinanceBillInvalidArgument
		}
		configs[config.GroupKey] = config
	}
	return buildConfiguredFinanceBillGroups(organizationID, fees, input, configs)
}

func buildConfiguredFinanceBillGroups(organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput, configs map[string]FinanceBillBatchPreviewGroupConfig) (*FinanceBillBatchPreview, error) {
	policy := input.GroupingPolicy
	if organizationID == uuid.Nil || len(fees) == 0 || len(fees) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	if policy.Mode != "NORMAL" && policy.Mode != "NETTING" {
		return nil, ErrFinanceBillGroupingModeUnsupported
	}
	ordered := append([]*FinanceBillableFee(nil), fees...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Fee.ID.String() < ordered[j].Fee.ID.String() })
	seen := make(map[uuid.UUID]struct{}, len(ordered))
	var direction OrderFeeDirection
	groupsByRawKey := make(map[string]*FinanceBillBatchPreviewGroup)
	rawKeys := make([]string, 0)
	for _, item := range ordered {
		if item == nil || item.Fee == nil || item.Fee.ID == uuid.Nil || item.Fee.OrderID == uuid.Nil || item.Fee.Status != OrderFeeConfirmed || item.Fee.TaxRate == nil || !financeBillCurrencyPattern.MatchString(item.Fee.Currency) || !financeBillCurrencyPattern.MatchString(item.Fee.BaseCurrency) {
			return nil, ErrFinanceBillFeeInvalid
		}
		fee := item.Fee
		if _, duplicate := seen[fee.ID]; duplicate {
			return nil, ErrFinanceBillInvalidArgument
		}
		seen[fee.ID] = struct{}{}
		// 普通模式一次建账只允许单一收付方向；对冲模式允许混合方向，但方向仍参与叶子身份，
		// 应收、应付费用分别形成各自的原始账单叶子。
		if policy.Mode == "NORMAL" {
			if direction == "" {
				direction = fee.Direction
			} else if direction != fee.Direction {
				return nil, ErrFinanceBillMixedDirection
			}
		}
		// 普通账单始终以费用币种拆分；币种是叶子身份的一部分，不能由请求覆盖。
		parts := []string{string(fee.Direction), fee.SettlementPartyID.String(), fee.Currency, fee.BaseCurrency}
		if policy.SplitByTaxRate {
			parts = append(parts, fee.TaxRate.StringFixed(4))
		}
		if policy.SplitByOrder {
			parts = append(parts, fee.OrderID.String())
		}
		raw := strings.Join(parts, "\x00")
		group := groupsByRawKey[raw]
		if group == nil {
			group = &FinanceBillBatchPreviewGroup{GroupKey: financeSHA256(raw), Direction: fee.Direction, SettlementPartyID: fee.SettlementPartyID, SettlementPartyName: fee.SettlementPartyName, Currency: fee.Currency, BaseCurrency: fee.BaseCurrency, Fees: make([]*FinanceBillableFee, 0)}
			if policy.SplitByTaxRate {
				value := *fee.TaxRate
				group.TaxRate = &value
			}
			if policy.SplitByOrder {
				id, no := fee.OrderID, item.OrderNo
				group.OrderID, group.OrderNo = &id, &no
			}
			groupsByRawKey[raw] = group
			rawKeys = append(rawKeys, raw)
		}
		if item.SettlementPartyIsCasual && !group.IsCasual {
			group.IsCasual = true
			if fee.Direction == OrderFeeReceivable {
				zero := 0
				group.DefaultPaymentTermsDays = &zero
			}
		}
		group.Fees = append(group.Fees, item)
	}
	sort.Strings(rawKeys)
	result := &FinanceBillBatchPreview{Groups: make([]*FinanceBillBatchPreviewGroup, 0, len(rawKeys))}
	for _, raw := range rawKeys {
		group := groupsByRawKey[raw]
		if config, ok := configs[group.GroupKey]; ok {
			group.BillDate, group.SettlementAccountID, group.config = config.BillDate, config.SettlementAccountID, config
		}
		result.Groups = append(result.Groups, group)
	}
	if policy.Mode == "NETTING" {
		if err := validateFinanceNettingGroupPairs(result.Groups); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// validateFinanceNettingGroupPairs 要求对冲建账的每个“结算单位 + 账单币种”组合
// 至少各有一笔应收和应付费用；单方向费用应使用普通账单，不静默拆成两个普通批次。
func validateFinanceNettingGroupPairs(groups []*FinanceBillBatchPreviewGroup) error {
	type pairDirection struct{ receivable, payable bool }
	pairs := make(map[string]*pairDirection)
	for _, group := range groups {
		if group == nil {
			return ErrFinanceNettingInvalid
		}
		key := group.SettlementPartyID.String() + "|" + group.Currency
		pair := pairs[key]
		if pair == nil {
			pair = &pairDirection{}
			pairs[key] = pair
		}
		if group.Direction == OrderFeeReceivable {
			pair.receivable = true
		} else {
			pair.payable = true
		}
	}
	for _, pair := range pairs {
		if !pair.receivable || !pair.payable {
			return ErrFinanceNettingSingleDirection
		}
	}
	return nil
}

func (uc *FinanceBillUsecase) buildConfiguredFinanceBillBatchPreview(ctx context.Context, organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput) (*FinanceBillBatchPreview, error) {
	preview, err := BuildConfiguredFinanceBillBatchPreview(organizationID, fees, input)
	if err != nil {
		return nil, err
	}
	if uc.exchangeRate == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	accountSkeletons := make(map[string]*FinanceBill, len(preview.Groups))
	accounts := make([]*FinanceBill, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		if group.Currency == "" || group.SettlementAccountID == uuid.Nil {
			continue
		}
		skeleton := &FinanceBill{OrganizationID: organizationID, Direction: group.Direction, SettlementPartyID: group.SettlementPartyID, SettlementAccountID: group.SettlementAccountID, Currency: group.Currency}
		accountSkeletons[group.GroupKey] = skeleton
		accounts = append(accounts, skeleton)
	}
	if len(accounts) > 0 {
		if err = uc.repo.HydrateBillSettlementAccounts(ctx, accounts); err != nil {
			return nil, err
		}
	}
	currencySet := make(map[string]struct{})
	for _, item := range fees {
		currencySet[item.Fee.Currency] = struct{}{}
		currencySet[item.Fee.BaseCurrency] = struct{}{}
	}
	for _, group := range preview.Groups {
		if group.Currency != "" {
			currencySet[group.Currency] = struct{}{}
		}
		if group.config.EstimatedInvoiceCurrency != nil {
			currencySet[*group.config.EstimatedInvoiceCurrency] = struct{}{}
		}
	}
	currencies := make([]string, 0, len(currencySet))
	for currency := range currencySet {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	if err = uc.repo.ValidateBillCurrencies(ctx, currencies); err != nil {
		return nil, err
	}

	complete := true
	for _, group := range preview.Groups {
		if strings.TrimSpace(group.BillDate) == "" {
			group.TemporaryBillDate = true
			for _, item := range group.Fees {
				group.TotalAmount = group.TotalAmount.Add(item.Fee.TotalAmount)
				group.NetAmount = group.NetAmount.Add(item.Fee.NetAmount)
			}
			group.TotalAmount = group.TotalAmount.RoundBank(8)
			group.NetAmount = group.NetAmount.RoundBank(8)
			group.TaxAmount = group.TotalAmount.Sub(group.NetAmount)
			estimatedCurrency, estimatedRate, estimatedAmount, estimatedErr := financeBillEstimatedInvoiceSnapshot(group.Currency, group.TotalAmount, group.config)
			if estimatedErr != nil {
				return nil, estimatedErr
			}
			group.EstimatedInvoiceCurrency = estimatedCurrency
			group.EstimatedInvoiceRate = estimatedRate
			group.EstimatedInvoiceAmount = estimatedAmount
			group.ConfigurationComplete = false
			complete = false
			continue
		}
		bill, buildErr := uc.buildFixedCurrencyFinanceBill(ctx, organizationID, group, group.BillDate)
		if buildErr != nil {
			return nil, buildErr
		}
		group.preparedBill = bill
		if account := accountSkeletons[group.GroupKey]; account != nil {
			bill.SettlementAccountName = account.SettlementAccountName
			bill.SettlementAccountHolder = account.SettlementAccountHolder
			bill.SettlementBankName = account.SettlementBankName
			bill.SettlementBankAccount = account.SettlementBankAccount
			bill.SettlementAccountCurrency = account.SettlementAccountCurrency
			bill.SettlementSwiftCode = account.SettlementSwiftCode
		}
		group.TotalAmount = bill.TotalAmount
		group.NetAmount = bill.NetAmount
		group.TaxAmount = bill.TaxAmount
		group.BaseCurrencyAmount = bill.BaseCurrencyAmount
		if group.BillDate == "" || group.SettlementAccountID == uuid.Nil {
			group.ConfigurationComplete = false
			complete = false
			continue
		}
		group.ConfigurationComplete = true
	}
	if input.GroupingPolicy.Mode == "NETTING" {
		preview.NettingPairs = buildFinanceNettingPairs(preview.Groups)
	}
	if !complete {
		preview.PreviewToken = ""
		return preview, nil
	}
	preview.PreviewToken = financeBillConfiguredPreviewToken(organizationID, input.GroupingPolicy, preview.Groups)
	return preview, nil
}

// buildFinanceNettingPairs 按结算单位与账单币种汇总叶子毛额，抵销额为双方较小值，
// 净应收/净应付为抵销后的剩余金额；金额只用共同账单币种。
func buildFinanceNettingPairs(groups []*FinanceBillBatchPreviewGroup) []*FinanceBillBatchNettingPair {
	type pairTotals struct {
		settlementPartyID   uuid.UUID
		settlementPartyName string
		currency            string
		receivable, payable decimal.Decimal
	}
	pairs := make(map[string]*pairTotals)
	keys := make([]string, 0)
	for _, group := range groups {
		key := group.SettlementPartyID.String() + "|" + group.Currency
		pair := pairs[key]
		if pair == nil {
			pair = &pairTotals{settlementPartyID: group.SettlementPartyID, settlementPartyName: group.SettlementPartyName, currency: group.Currency}
			pairs[key] = pair
			keys = append(keys, key)
		}
		if group.Direction == OrderFeeReceivable {
			pair.receivable = pair.receivable.Add(group.TotalAmount)
		} else {
			pair.payable = pair.payable.Add(group.TotalAmount)
		}
	}
	sort.Strings(keys)
	result := make([]*FinanceBillBatchNettingPair, 0, len(keys))
	for _, key := range keys {
		pair := pairs[key]
		item := &FinanceBillBatchNettingPair{
			SettlementPartyID: pair.settlementPartyID, SettlementPartyName: pair.settlementPartyName, Currency: pair.currency,
			ReceivableGrossAmount: pair.receivable.Round(8), PayableGrossAmount: pair.payable.Round(8),
		}
		if item.ReceivableGrossAmount.IsPositive() && item.PayableGrossAmount.IsPositive() {
			item.OffsetAmount = decimal.Min(item.ReceivableGrossAmount, item.PayableGrossAmount).Round(8)
		}
		item.NetReceivableAmount = item.ReceivableGrossAmount.Sub(item.OffsetAmount).Round(8)
		item.NetPayableAmount = item.PayableGrossAmount.Sub(item.OffsetAmount).Round(8)
		result = append(result, item)
	}
	return result
}

func (uc *FinanceBillUsecase) buildFixedCurrencyFinanceBill(ctx context.Context, organizationID uuid.UUID, group *FinanceBillBatchPreviewGroup, billDate string) (*FinanceBill, error) {
	if group == nil || len(group.Fees) == 0 || !validFinanceDate(billDate) || !financeBillCurrencyPattern.MatchString(group.Currency) {
		return nil, ErrFinanceBillInvalidArgument
	}
	if group.config.EstimatedInvoiceRate != nil && group.config.EstimatedInvoiceCurrency == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	for _, item := range group.Fees {
		if item == nil || item.Fee == nil || item.Fee.Currency != group.Currency || item.Fee.BaseCurrency != group.BaseCurrency {
			return nil, ErrFinanceBillFeeMismatch
		}
	}
	resolved, err := uc.exchangeRate.ResolveRate(ctx, organizationID, group.Currency, billDate)
	if err != nil {
		return nil, err
	}
	billID := uuid.Must(uuid.NewV7())
	bill := &FinanceBill{
		ID: billID, OrganizationID: organizationID, Direction: group.Direction, Status: FinanceBillDraft,
		SettlementPartyID: group.SettlementPartyID, SettlementPartyName: group.SettlementPartyName,
		SettlementAccountID: group.SettlementAccountID, Currency: group.Currency, BaseCurrency: group.BaseCurrency,
		ExchangeRate: resolved.Rate.RoundBank(8), ExchangeRateSource: resolved.Source, ExchangeRateDate: billDate,
		BillDate: billDate, Version: 1,
		Lines: make([]*FinanceBillLine, 0, len(group.Fees)),
	}
	for _, item := range group.Fees {
		fee := item.Fee
		bill.Lines = append(bill.Lines, &FinanceBillLine{
			ID: uuid.Must(uuid.NewV7()), BillID: billID, OrderFeeID: fee.ID, OrderID: fee.OrderID,
			OrderNo: item.OrderNo, BusinessType: item.BusinessType, FeeCode: fee.FeeCode, FeeName: fee.FeeName,
			Quantity: fee.Quantity, UnitPrice: fee.UnitPrice, TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, TaxRate: fee.TaxRate,
			Currency: group.Currency, ExchangeRate: bill.ExchangeRate, BaseCurrency: group.BaseCurrency, Active: true,
		})
	}
	sort.Slice(bill.Lines, func(i, j int) bool { return bill.Lines[i].OrderFeeID.String() < bill.Lines[j].OrderFeeID.String() })
	for _, line := range bill.Lines {
		bill.TotalAmount = bill.TotalAmount.Add(line.TotalAmount)
		bill.NetAmount = bill.NetAmount.Add(line.NetAmount)
		bill.TaxAmount = bill.TaxAmount.Add(line.TaxAmount)
	}
	bill.TotalAmount = bill.TotalAmount.RoundBank(8)
	bill.NetAmount = bill.NetAmount.RoundBank(8)
	bill.TaxAmount = bill.TotalAmount.Sub(bill.NetAmount)
	bill.BaseCurrencyAmount = bill.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
	allocatedBase := decimal.Zero
	for index, line := range bill.Lines {
		lineBase := line.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
		if index == len(bill.Lines)-1 {
			lineBase = bill.BaseCurrencyAmount.Sub(allocatedBase)
		}
		line.BaseCurrencyAmount = lineBase
		allocatedBase = allocatedBase.Add(lineBase)
	}
	bill.FeeCount = len(bill.Lines)
	estimatedCurrency, estimatedRate, estimatedAmount, err := financeBillEstimatedInvoiceSnapshot(group.Currency, bill.TotalAmount, group.config)
	if err != nil {
		return nil, err
	}
	bill.EstimatedInvoiceCurrency = &estimatedCurrency
	bill.EstimatedInvoiceRate = &estimatedRate
	bill.EstimatedInvoiceAmount = &estimatedAmount
	group.EstimatedInvoiceCurrency = estimatedCurrency
	group.EstimatedInvoiceRate = estimatedRate
	group.EstimatedInvoiceAmount = estimatedAmount
	return bill, validateFinanceBillAmountInvariants(bill)
}

func financeBillEstimatedInvoiceSnapshot(billCurrency string, totalAmount decimal.Decimal, config FinanceBillBatchPreviewGroupConfig) (string, decimal.Decimal, decimal.Decimal, error) {
	estimatedCurrency := billCurrency
	estimatedRate := decimal.NewFromInt(1)
	if config.EstimatedInvoiceCurrency != nil {
		estimatedCurrency = strings.ToUpper(strings.TrimSpace(*config.EstimatedInvoiceCurrency))
		if config.EstimatedInvoiceRate != nil {
			estimatedRate = *config.EstimatedInvoiceRate
		} else if estimatedCurrency != billCurrency {
			return "", decimal.Zero, decimal.Zero, ErrFinanceBillInvalidArgument
		}
	}
	// 同币种预计开票不应凭人工汇率改变预计金额；跨币种才需要显式折算率。
	if !financeBillCurrencyPattern.MatchString(estimatedCurrency) || !estimatedRate.IsPositive() || estimatedRate.Exponent() < -8 || (estimatedCurrency == billCurrency && !estimatedRate.Equal(decimal.NewFromInt(1))) {
		return "", decimal.Zero, decimal.Zero, ErrFinanceBillInvalidArgument
	}
	return estimatedCurrency, estimatedRate, totalAmount.Mul(estimatedRate).RoundBank(8), nil
}

func validateFinanceBillAmountInvariants(bill *FinanceBill) error {
	if bill == nil || len(bill.Lines) == 0 || !bill.ExchangeRate.IsPositive() || !bill.TotalAmount.Equal(bill.NetAmount.Add(bill.TaxAmount)) {
		return ErrFinanceBillBatchMismatch
	}
	total, net, tax, base := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	for _, line := range bill.Lines {
		if line == nil || !line.ExchangeRate.IsPositive() || !line.TotalAmount.Equal(line.NetAmount.Add(line.TaxAmount)) {
			return ErrFinanceBillBatchMismatch
		}
		total, net, tax, base = total.Add(line.TotalAmount), net.Add(line.NetAmount), tax.Add(line.TaxAmount), base.Add(line.BaseCurrencyAmount)
	}
	if !total.Equal(bill.TotalAmount) || !net.Equal(bill.NetAmount) || !tax.Equal(bill.TaxAmount) || !base.Equal(bill.BaseCurrencyAmount) {
		return ErrFinanceBillBatchMismatch
	}
	return nil
}

func financeBillConfiguredPreviewToken(organizationID uuid.UUID, policy FinanceBillGroupingPolicy, groups []*FinanceBillBatchPreviewGroup) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, organizationID.String(), policy.Mode, strconv.FormatBool(policy.SplitByTaxRate), strconv.FormatBool(policy.SplitByOrder))
	for _, group := range groups {
		bill := group.preparedBill
		orderID, orderNo, taxRate, headerSettingID := "", "", "", ""
		if group.OrderID != nil {
			orderID = group.OrderID.String()
		}
		if group.OrderNo != nil {
			orderNo = *group.OrderNo
		}
		if group.TaxRate != nil {
			taxRate = group.TaxRate.StringFixed(4)
		}
		if bill.ExchangeRateSettingID != nil {
			headerSettingID = bill.ExchangeRateSettingID.String()
		}
		writeFinanceHashParts(&builder, group.GroupKey, string(group.Direction), group.SettlementPartyID.String(), group.SettlementPartyName, group.Currency, group.BaseCurrency, orderID, orderNo, taxRate, group.BillDate, group.SettlementAccountID.String(), bill.SettlementAccountName, bill.SettlementAccountHolder, bill.SettlementBankName, bill.SettlementBankAccount, bill.SettlementAccountCurrency, bill.SettlementSwiftCode, bill.ExchangeRate.StringFixed(8), bill.ExchangeRateSource, bill.ExchangeRateDate, headerSettingID, group.EstimatedInvoiceCurrency, group.EstimatedInvoiceRate.StringFixed(8), group.EstimatedInvoiceAmount.StringFixed(8))
		for _, line := range bill.Lines {
			taxRate := ""
			if line.TaxRate != nil {
				taxRate = line.TaxRate.StringFixed(4)
			}
			fee := financeBillFeeByID(group.Fees, line.OrderFeeID)
			feeStatus := ""
			if fee != nil {
				feeStatus = string(fee.Status)
			}
			writeFinanceHashParts(&builder, line.OrderFeeID.String(), line.OrderID.String(), line.OrderNo, line.BusinessType, line.FeeCode, line.FeeName, taxRate, line.Currency, line.TotalAmount.StringFixed(8), line.NetAmount.StringFixed(8), line.TaxAmount.StringFixed(8), line.ExchangeRate.StringFixed(8), line.BaseCurrencyAmount.StringFixed(8), strconv.FormatUint(financeBillFeeVersion(group.Fees, line.OrderFeeID), 10), feeStatus)
		}
	}
	return financeSHA256(builder.String())
}

func financeBillFeeVersion(items []*FinanceBillableFee, id uuid.UUID) uint64 {
	if fee := financeBillFeeByID(items, id); fee != nil {
		return fee.Version
	}
	return 0
}

func financeBillFeeByID(items []*FinanceBillableFee, id uuid.UUID) *OrderFee {
	for _, item := range items {
		if item != nil && item.Fee != nil && item.Fee.ID == id {
			return item.Fee
		}
	}
	return nil
}

func (uc *FinanceBillUsecase) ResolveBillableFeeOrganization(ctx context.Context, organizationIDs, feeIDs []uuid.UUID) (uuid.UUID, error) {
	if len(organizationIDs) == 0 || len(feeIDs) == 0 {
		return uuid.Nil, ErrFinanceBillInvalidArgument
	}
	fees, err := uc.repo.LoadBillableFeesScoped(ctx, organizationIDs, feeIDs)
	if err != nil {
		return uuid.Nil, err
	}
	if len(fees) != len(feeIDs) {
		return uuid.Nil, ErrFinanceBillInvalidArgument
	}
	organizationID := fees[0].OrganizationID
	if organizationID == uuid.Nil {
		return uuid.Nil, ErrFinanceBillInvalidArgument
	}
	for _, fee := range fees[1:] {
		if fee.OrganizationID != organizationID {
			return uuid.Nil, ErrFinanceBillInvalidArgument
		}
	}
	return organizationID, nil
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
	if uc.transactor == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		fees, transactionErr := uc.repo.LoadBillableFees(txCtx, organizationID, input.FeeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		if len(fees) != len(input.FeeIDs) {
			return ErrFinanceBillFeeInvalid
		}
		groupConfigs := make([]FinanceBillBatchPreviewGroupConfig, 0, len(input.Groups))
		for _, group := range input.Groups {
			groupConfigs = append(groupConfigs, FinanceBillBatchPreviewGroupConfig{
				GroupKey: group.GroupKey, BillDate: group.BillDate,
				SettlementAccountID:      group.SettlementAccountID,
				EstimatedInvoiceCurrency: group.EstimatedInvoiceCurrency, EstimatedInvoiceRate: group.EstimatedInvoiceRate,
			})
		}
		previewInput := PreviewFinanceBillBatchInput{FeeIDs: input.FeeIDs, GroupingPolicy: input.GroupingPolicy, GroupConfigs: groupConfigs}
		preview, transactionErr := uc.buildConfiguredFinanceBillBatchPreview(txCtx, organizationID, fees, previewInput)
		if transactionErr != nil {
			return transactionErr
		}
		if preview.PreviewToken == "" || preview.PreviewToken != input.PreviewToken {
			return ErrFinanceBillPreviewStale
		}
		groupInputs := make(map[string]CreateFinanceBillBatchGroupInput, len(input.Groups))
		for _, group := range input.Groups {
			groupInputs[group.GroupKey] = group
		}
		batchID := uuid.Must(uuid.NewV7())
		batch := &FinanceBillBatch{ID: batchID, OrganizationID: organizationID, CreatedBy: actorID, IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash, GroupingPolicy: input.GroupingPolicy, FeeCount: len(input.FeeIDs), BillCount: len(preview.Groups), Bills: make([]*FinanceBill, 0, len(preview.Groups))}
		for _, previewGroup := range preview.Groups {
			groupInput, ok := groupInputs[previewGroup.GroupKey]
			if !ok || previewGroup.preparedBill == nil {
				return ErrFinanceBillBatchMismatch
			}
			delete(groupInputs, previewGroup.GroupKey)
			bill := previewGroup.preparedBill
			bill.ID = uuid.Must(uuid.NewV7())
			bill.BatchID = &batchID
			bill.IdempotencyKey = financeBillBatchBillKey(input.IdempotencyKey, previewGroup.GroupKey)
			title := groupInput.StatementTitle
			bill.StatementTitle = &title
			bill.PaymentTermsDays = groupInput.PaymentTermsDays
			bill.DueDate = normalizedFinanceBillDueDate(groupInput.BillDate, groupInput.DueDate, groupInput.PaymentTermsDays)
			bill.Note = normalizedOptionalFinanceString(groupInput.Note)
			for _, line := range bill.Lines {
				line.ID = uuid.Must(uuid.NewV7())
				line.BillID = bill.ID
			}
			if !validFinanceBillTerms(bill.BillDate, bill.DueDate, bill.PaymentTermsDays) {
				return ErrFinanceBillInvalidArgument
			}
			batch.Bills = append(batch.Bills, bill)
			batch.TotalBaseAmount = batch.TotalBaseAmount.Add(bill.BaseCurrencyAmount)
			if batch.BaseCurrency == "" {
				batch.BaseCurrency = bill.BaseCurrency
			} else if batch.BaseCurrency != bill.BaseCurrency {
				return ErrFinanceBillBatchMismatch
			}
		}
		if len(groupInputs) != 0 {
			return ErrFinanceBillBatchMismatch
		}
		batch.TotalBaseAmount = batch.TotalBaseAmount.RoundBank(8)
		nettingAudits := make([]*AuditEvent, 0)
		if input.GroupingPolicy.Mode == "NETTING" {
			// 对冲批次在同一事务内原子生成双方原始账单与对冲结算单；对冲单初始为草稿，
			// 待账单确认后再单独确认生效。
			nettings, planErr := planBatchFinanceNettings(organizationID, batchID, input.IdempotencyKey, batch.Bills)
			if planErr != nil {
				return planErr
			}
			batch.Nettings = nettings
			for _, netting := range nettings {
				nettingAudits = append(nettingAudits, financeNettingAudit(organizationID, actorID, netting.ID, "finance.netting.create"))
			}
		}
		_, transactionErr = uc.repo.CreateBatch(txCtx, batch, input.PreviewToken, financeBillBatchAudit(organizationID, actorID, batchID, "finance.bill_batch.create"), nettingAudits)
		return transactionErr
	})
	if err == nil {
		return uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	}
	if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr == nil && existing != nil && existing.RequestHash == requestHash {
		return existing, nil
	}
	return nil, err
}

func (uc *FinanceBillUsecase) ConfirmBatch(ctx context.Context, organizationIDs []uuid.UUID, actorID, batchID uuid.UUID, expectedVersions map[uuid.UUID]uint64) (*FinanceBillBatch, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || batchID == uuid.Nil || len(expectedVersions) == 0 || len(expectedVersions) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	for billID, version := range expectedVersions {
		if billID == uuid.Nil || version == 0 {
			return nil, ErrFinanceBillInvalidArgument
		}
	}
	batch, err := uc.repo.GetBatch(ctx, organizationIDs, batchID)
	if err != nil {
		return nil, err
	}
	return uc.repo.ConfirmBatch(ctx, organizationIDs, batchID, actorID, expectedVersions, financeBillBatchAudit(batch.OrganizationID, actorID, batchID, "finance.bill_batch.confirm"))
}

func (uc *FinanceBillUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillInput) (*FinanceBill, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	normalized, err := normalizeCreateFinanceBill(input)
	if err != nil {
		return nil, err
	}
	var created *FinanceBill
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, transactionErr := uc.repo.GetByIdempotencyKey(txCtx, organizationID, normalized.IdempotencyKey)
		if transactionErr != nil {
			return transactionErr
		}
		if existing != nil {
			if !sameFinanceBillCreateIntent(existing, normalized) {
				return ErrFinanceBillIdempotencyConflict
			}
			created = existing
			return nil
		}
		fees, transactionErr := uc.repo.LoadBillableFees(txCtx, organizationID, normalized.FeeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		bill, transactionErr := buildFinanceBill(organizationID, fees, normalized)
		if transactionErr != nil {
			return transactionErr
		}
		// 与批量建账保持相同锁序：费用 → 账户 → 汇率设置。
		if transactionErr = uc.repo.HydrateBillSettlementAccounts(txCtx, []*FinanceBill{bill}); transactionErr != nil {
			return transactionErr
		}
		if transactionErr = uc.repo.ValidateBillCurrencies(txCtx, []string{bill.Currency, bill.BaseCurrency}); transactionErr != nil {
			return transactionErr
		}
		if transactionErr = uc.applyBillExchangeRate(txCtx, organizationID, bill); transactionErr != nil {
			return transactionErr
		}
		created, transactionErr = uc.repo.Create(txCtx, bill, financeBillAudit(organizationID, actorID, bill.ID, "finance.bill.create"))
		return transactionErr
	})
	if err == nil {
		return uc.repo.Get(ctx, []uuid.UUID{organizationID}, created.ID)
	}
	existing, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, normalized.IdempotencyKey)
	if lookupErr == nil && existing != nil && sameFinanceBillCreateIntent(existing, normalized) {
		return existing, nil
	}
	return nil, err
}

func (uc *FinanceBillUsecase) applyBillExchangeRate(ctx context.Context, organizationID uuid.UUID, bill *FinanceBill) error {
	if uc.exchangeRate == nil || bill == nil {
		return ErrFinanceBillInvalidArgument
	}
	resolved, err := uc.exchangeRate.ResolveRate(ctx, organizationID, bill.Currency, bill.BillDate)
	if err != nil {
		return err
	}
	baseCurrency, err := uc.exchangeRate.BaseCurrency(ctx, organizationID)
	if err != nil {
		return err
	}
	if baseCurrency != bill.BaseCurrency {
		return ErrFinanceBillFeeMismatch
	}
	bill.ExchangeRate = resolved.Rate.RoundBank(8)
	bill.ExchangeRateSource = resolved.Source
	bill.ExchangeRateDate = bill.BillDate
	bill.ExchangeRateSettingID = nil
	// 头本位币金额必须使用已固化（舍入到 8 位）的账单汇率，与批量内核口径一致。
	bill.BaseCurrencyAmount = bill.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
	allocated := decimal.Zero
	for index, line := range bill.Lines {
		line.ExchangeRate = bill.ExchangeRate
		lineBase := line.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
		if index == len(bill.Lines)-1 {
			lineBase = bill.BaseCurrencyAmount.Sub(allocated)
		}
		line.BaseCurrencyAmount = lineBase
		allocated = allocated.Add(lineBase)
	}
	return nil
}

func (uc *FinanceBillUsecase) Update(ctx context.Context, organizationIDs []uuid.UUID, actorID uuid.UUID, input UpdateFinanceBillInput) (*FinanceBill, error) {
	input.BillDate = strings.TrimSpace(input.BillDate)
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.DueDate = normalizedFinanceBillDueDate(input.BillDate, input.DueDate, input.PaymentTermsDays)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || input.ID == uuid.Nil || input.ExpectedVersion == 0 || !validFinanceDate(input.BillDate) || !validFinanceBillTerms(input.BillDate, input.DueDate, input.PaymentTermsDays) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, input.ID)
	if err != nil {
		return nil, err
	}
	if existing.Status != FinanceBillDraft || existing.Version != input.ExpectedVersion || uc.exchangeRate == nil || uc.transactor == nil {
		if existing.Status != FinanceBillDraft {
			return nil, ErrFinanceBillInvalidTransition
		}
		if existing.Version != input.ExpectedVersion {
			return nil, ErrFinanceBillVersionConflict
		}
		return nil, ErrFinanceBillInvalidArgument
	}
	feeIDs := make([]uuid.UUID, 0, len(existing.Lines))
	for _, line := range existing.Lines {
		feeIDs = append(feeIDs, line.OrderFeeID)
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 账单更新、取消和确认统一先锁账单；更新随后再按 ID 锁费用，避免与
		// “账单 → 费用”的状态操作形成反向等待。
		current, transactionErr := uc.repo.Get(txCtx, organizationIDs, input.ID)
		if transactionErr != nil {
			return transactionErr
		}
		if current.Status != FinanceBillDraft {
			return ErrFinanceBillInvalidTransition
		}
		if current.Version != input.ExpectedVersion {
			return ErrFinanceBillVersionConflict
		}
		lockedFees, transactionErr := uc.repo.LoadBillableFees(txCtx, current.OrganizationID, feeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		if transactionErr = validateLockedFinanceBillSourceFees(current, lockedFees); transactionErr != nil {
			return transactionErr
		}
		account := &FinanceBill{OrganizationID: current.OrganizationID, Direction: current.Direction, SettlementPartyID: current.SettlementPartyID, SettlementAccountID: input.SettlementAccountID, Currency: current.Currency}
		if transactionErr = uc.repo.HydrateBillSettlementAccounts(txCtx, []*FinanceBill{account}); transactionErr != nil {
			return transactionErr
		}
		currencies := []string{current.Currency, current.BaseCurrency}
		for _, fee := range lockedFees {
			currencies = append(currencies, fee.Fee.Currency)
		}
		if input.EstimatedInvoiceCurrency != nil {
			currencies = append(currencies, *input.EstimatedInvoiceCurrency)
		}
		if transactionErr = uc.repo.ValidateBillCurrencies(txCtx, currencies); transactionErr != nil {
			return transactionErr
		}
		group := &FinanceBillBatchPreviewGroup{Direction: current.Direction, SettlementPartyID: current.SettlementPartyID, SettlementPartyName: current.SettlementPartyName, Currency: current.Currency, BaseCurrency: current.BaseCurrency, BillDate: input.BillDate, SettlementAccountID: input.SettlementAccountID, Fees: lockedFees, config: FinanceBillBatchPreviewGroupConfig{BillDate: input.BillDate, SettlementAccountID: input.SettlementAccountID, EstimatedInvoiceCurrency: input.EstimatedInvoiceCurrency, EstimatedInvoiceRate: input.EstimatedInvoiceRate}}
		rebuilt, transactionErr := uc.buildFixedCurrencyFinanceBill(txCtx, current.OrganizationID, group, input.BillDate)
		if transactionErr != nil {
			return transactionErr
		}
		rebuilt.ID = current.ID
		rebuilt.SettlementAccountName, rebuilt.SettlementAccountHolder, rebuilt.SettlementBankName = account.SettlementAccountName, account.SettlementAccountHolder, account.SettlementBankName
		rebuilt.SettlementBankAccount, rebuilt.SettlementAccountCurrency, rebuilt.SettlementSwiftCode = account.SettlementBankAccount, account.SettlementAccountCurrency, account.SettlementSwiftCode
		lineIDs := make(map[uuid.UUID]uuid.UUID, len(current.Lines))
		for _, line := range current.Lines {
			lineIDs[line.OrderFeeID] = line.ID
		}
		for _, line := range rebuilt.Lines {
			line.ID = lineIDs[line.OrderFeeID]
			line.BillID = current.ID
		}
		input.ExchangeRate, input.ExchangeRateSource, input.ExchangeRateDate, input.ExchangeRateSettingID = rebuilt.ExchangeRate, rebuilt.ExchangeRateSource, rebuilt.ExchangeRateDate, rebuilt.ExchangeRateSettingID
		input.BaseCurrencyAmount, input.TotalAmount, input.NetAmount, input.TaxAmount, input.Lines = rebuilt.BaseCurrencyAmount, rebuilt.TotalAmount, rebuilt.NetAmount, rebuilt.TaxAmount, rebuilt.Lines
		input.EstimatedInvoiceCurrency, input.EstimatedInvoiceRate, input.EstimatedInvoiceAmount = rebuilt.EstimatedInvoiceCurrency, rebuilt.EstimatedInvoiceRate, rebuilt.EstimatedInvoiceAmount
		_, transactionErr = uc.repo.Update(txCtx, organizationIDs, input, financeBillAudit(current.OrganizationID, actorID, input.ID, "finance.bill.update"))
		return transactionErr
	})
	if err != nil {
		return nil, err
	}
	return uc.repo.Get(ctx, organizationIDs, input.ID)
}

func validateLockedFinanceBillSourceFees(bill *FinanceBill, fees []*FinanceBillableFee) error {
	if bill == nil || len(fees) != len(bill.Lines) {
		return ErrFinanceBillPreviewStale
	}
	lines := make(map[uuid.UUID]*FinanceBillLine, len(bill.Lines))
	for _, line := range bill.Lines {
		lines[line.OrderFeeID] = line
	}
	for _, item := range fees {
		fee, line := item.Fee, lines[item.Fee.ID]
		if line == nil || fee.Status != OrderFeeBilled || fee.Direction != bill.Direction || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != line.Currency || !fee.TotalAmount.Equal(line.TotalAmount) || !fee.NetAmount.Equal(line.NetAmount) || !fee.TaxAmount.Equal(line.TaxAmount) {
			return ErrFinanceBillPreviewStale
		}
	}
	return nil
}

func (uc *FinanceBillUsecase) Confirm(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64) (*FinanceBill, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.Confirm(ctx, organizationIDs, id, actorID, expectedVersion, financeBillAudit(existing.OrganizationID, actorID, id, "finance.bill.confirm"))
}

func (uc *FinanceBillUsecase) Cancel(ctx context.Context, organizationIDs []uuid.UUID, actorID, id uuid.UUID, expectedVersion uint64, reason string) (*FinanceBill, error) {
	reason = strings.TrimSpace(reason)
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	existing, err := uc.repo.Get(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return uc.repo.Cancel(ctx, organizationIDs, id, actorID, expectedVersion, reason, financeBillAudit(existing.OrganizationID, actorID, id, "finance.bill.cancel"))
}

func normalizeCreateFinanceBill(input CreateFinanceBillInput) (CreateFinanceBillInput, error) {
	input.BillDate = strings.TrimSpace(input.BillDate)
	input.DueDate = normalizedOptionalFinanceString(input.DueDate)
	input.Note = normalizedOptionalFinanceString(input.Note)
	input.StatementTitle = normalizedOptionalFinanceString(input.StatementTitle)
	input.DueDate = normalizedFinanceBillDueDate(input.BillDate, input.DueDate, input.PaymentTermsDays)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if len(input.FeeIDs) == 0 || len(input.FeeIDs) > 500 || input.SettlementAccountID == uuid.Nil || !validFinanceDate(input.BillDate) || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || !validFinanceBillTerms(input.BillDate, input.DueDate, input.PaymentTermsDays) || (input.Note != nil && utf8.RuneCountInString(*input.Note) > 500) || (input.StatementTitle != nil && utf8.RuneCountInString(*input.StatementTitle) > 200) {
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
		SettlementPartyName: first.Fee.SettlementPartyName, SettlementAccountID: input.SettlementAccountID, Currency: first.Fee.Currency, BaseCurrency: first.Fee.BaseCurrency,
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
			Quantity: fee.Quantity, UnitPrice: fee.UnitPrice, TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, Currency: fee.Currency,
			TaxRate: fee.TaxRate, ExchangeRate: decimal.NewFromInt(1), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount, Active: true,
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
	if existing.SettlementAccountID != requested.SettlementAccountID || existing.BillDate != requested.BillDate || !stringPointersEqual(existing.DueDate, requested.DueDate) || !stringPointersEqual(existing.Note, requested.Note) || !stringPointersEqual(existing.StatementTitle, requestedTitle) || !intPointersEqual(existing.PaymentTermsDays, requested.PaymentTermsDays) || len(existing.Lines) != len(requested.FeeIDs) {
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
		if item.EstimatedInvoiceCurrency != nil {
			value := strings.ToUpper(strings.TrimSpace(*item.EstimatedInvoiceCurrency))
			item.EstimatedInvoiceCurrency = &value
		}
		item.DueDate = normalizedOptionalFinanceString(item.DueDate)
		item.Note = normalizedOptionalFinanceString(item.Note)
		item.DueDate = normalizedFinanceBillDueDate(item.BillDate, item.DueDate, item.PaymentTermsDays)
		if item.GroupKey == "" || len(item.GroupKey) != 64 || item.SettlementAccountID == uuid.Nil || item.StatementTitle == "" || utf8.RuneCountInString(item.StatementTitle) > 200 || !validFinanceDate(item.BillDate) || !validFinanceBillTerms(item.BillDate, item.DueDate, item.PaymentTermsDays) || (item.Note != nil && utf8.RuneCountInString(*item.Note) > 500) || (item.EstimatedInvoiceCurrency != nil && !financeBillCurrencyPattern.MatchString(*item.EstimatedInvoiceCurrency)) || (item.EstimatedInvoiceRate != nil && (!item.EstimatedInvoiceRate.IsPositive() || item.EstimatedInvoiceRate.Exponent() < -8)) || (item.EstimatedInvoiceRate != nil && item.EstimatedInvoiceCurrency == nil) {
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

func financeBillBatchRequestHash(input CreateFinanceBillBatchInput) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, input.PreviewToken, input.GroupingPolicy.Mode, strconv.FormatBool(input.GroupingPolicy.SplitByOrder), strconv.FormatBool(input.GroupingPolicy.SplitByTaxRate))
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
		estimatedCurrency, estimatedRate := "", ""
		if group.EstimatedInvoiceCurrency != nil {
			estimatedCurrency = *group.EstimatedInvoiceCurrency
		}
		if group.EstimatedInvoiceRate != nil {
			estimatedRate = group.EstimatedInvoiceRate.StringFixed(8)
		}
		writeFinanceHashParts(&builder, group.GroupKey, group.StatementTitle, group.BillDate, dueDate, paymentTermsDays, note, group.SettlementAccountID.String(), estimatedCurrency, estimatedRate)
	}
	return financeSHA256(builder.String())
}

func financeBillBatchBillKey(batchKey, groupKey string) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, batchKey, groupKey)
	return "batch-bill:" + financeSHA256(builder.String())
}

// financeBillBatchNettingKey 为对冲批次内每个“结算单位 + 账单币种”组合生成确定性幂等键。
func financeBillBatchNettingKey(batchKey string, settlementPartyID uuid.UUID, currency string) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, batchKey, settlementPartyID.String(), currency)
	return "batch-netting:" + financeSHA256(builder.String())
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

// CalculateOverdueDays 计算应收账单的逾期天数。只有已确认、存在未核销余额且到期日小于当前业务日期的应收账单才计算逾期天数。
func CalculateOverdueDays(direction OrderFeeDirection, status FinanceBillStatus, unverifiedAmount decimal.Decimal, dueDate *string, businessDate string) int32 {
	if direction != OrderFeeReceivable || status != FinanceBillConfirmed || !unverifiedAmount.IsPositive() || dueDate == nil || strings.TrimSpace(*dueDate) == "" {
		return 0
	}
	trimmedDueDate := strings.TrimSpace(*dueDate)
	if trimmedDueDate >= businessDate {
		return 0
	}
	dueT, err1 := time.Parse("2006-01-02", trimmedDueDate)
	bizT, err2 := time.Parse("2006-01-02", businessDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	days := int32(bizT.Sub(dueT).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
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
