package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type FinanceBillStatus string

const (
	FinanceBillDraft     FinanceBillStatus = "DRAFT"
	FinanceBillConfirmed FinanceBillStatus = "CONFIRMED"
	FinanceBillCancelled FinanceBillStatus = "CANCELLED"
)

type FinanceBillPurpose string

const (
	FinanceBillNormal             FinanceBillPurpose = "NORMAL"
	FinanceBillCorrectionReversal FinanceBillPurpose = "CORRECTION_REVERSAL"
)

func FinanceBillPurposeForFee(fee *OrderFee) FinanceBillPurpose {
	return FinanceBillNormal
}

type FinanceBill struct {
	CorrectionRequestID                                           *uuid.UUID
	InvoiceableAmount, InvoiceableNetAmount, InvoiceableTaxAmount *decimal.Decimal
	Purpose                                                       FinanceBillPurpose
	ID                                                            uuid.UUID
	OrganizationID                                                uuid.UUID
	OrganizationName                                              string
	BatchID                                                       *uuid.UUID
	BatchNo                                                       string
	BillNo                                                        string
	IdempotencyKey                                                string
	Direction                                                     OrderFeeDirection
	Status                                                        FinanceBillStatus
	SettlementPartyID                                             uuid.UUID
	SettlementPartyName                                           string
	SettlementAccountID                                           uuid.UUID
	SettlementAccountName                                         string
	SettlementAccountHolder                                       string
	SettlementBankName                                            string
	SettlementBankAccount                                         string
	SettlementAccountCurrency                                     string
	SettlementSwiftCode                                           string
	EstimatedInvoiceCurrency                                      *string
	EstimatedInvoiceRate                                          *decimal.Decimal
	EstimatedInvoiceAmount                                        *decimal.Decimal
	Currency                                                      string
	BaseCurrency                                                  string
	ExchangeRate                                                  decimal.Decimal
	ExchangeRateSource                                            string
	ExchangeRateDate                                              string
	ExchangeRateSettingID                                         *uuid.UUID
	TotalAmount                                                   decimal.Decimal
	NetAmount                                                     decimal.Decimal
	TaxAmount                                                     decimal.Decimal
	BaseCurrencyAmount                                            decimal.Decimal
	VerifiedAmount                                                decimal.Decimal
	NettedAmount                                                  decimal.Decimal
	UnverifiedAmount                                              decimal.Decimal
	OverdueDays                                                   int32
	FeeCount                                                      int
	BillDate                                                      string
	StatementTitle                                                *string
	PaymentTermsDays                                              *int
	DueDate                                                       *string
	Note                                                          *string
	Version                                                       uint64
	ConfirmedAt                                                   *time.Time
	ConfirmedBy                                                   *uuid.UUID
	CancelledAt                                                   *time.Time
	CancelledBy                                                   *uuid.UUID
	CancellationReason                                            *string
	Lines                                                         []*FinanceBillLine
	CreatedAt                                                     time.Time
	UpdatedAt                                                     time.Time
	// 以下为列表与详情读取时推导的投影（见 ResolveFinanceBillProgress），不落库；
	// 未经投影的账单（如建账响应）保持零值。
	SettlementProgress FinanceBillSettlementProgress
	InvoiceProgress    FinanceBillInvoiceProgress
	Completed          bool
	// FeeLockFacts 为阻止增删费用的有效财务事实类型，按固定顺序去重；非空即费用锁定。
	FeeLockFacts []FinanceBillFeeLockFact
	// InvoiceExemptions 仅详情提供免票决定历史，列表根据 FeeLockFacts 判断当前事实。
	InvoiceExemptions []*FinanceInvoiceExemption
}

// FeesLocked 表示账单存在阻止增删费用的有效财务事实。
func (b *FinanceBill) FeesLocked() bool { return len(b.FeeLockFacts) > 0 }

type FinanceBillLine struct {
	ID                                uuid.UUID
	BillID                            uuid.UUID
	OrderFeeID                        uuid.UUID
	OrderID                           uuid.UUID
	OrderNo                           string
	BusinessType                      string
	FeeCode                           string
	FeeName                           string
	Quantity                          decimal.Decimal
	UnitPrice                         decimal.Decimal
	TotalAmount                       decimal.Decimal
	NetAmount                         decimal.Decimal
	TaxAmount                         decimal.Decimal
	TaxRate                           *decimal.Decimal
	Currency                          string
	ExchangeRate                      decimal.Decimal
	BaseCurrency                      string
	BaseCurrencyAmount                decimal.Decimal
	Active                            bool
	CreatedAt                         time.Time
	UpdatedAt                         time.Time
	InvoicedAmount, InvoiceableAmount *decimal.Decimal
}

type FinanceBillableFee struct {
	OrderLocked             bool
	HasPendingFeeSupplement bool
	HasPendingFeeCorrection bool
	CanBill                 bool
	BlockedReasons          []string
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
	DueDateFrom       string
	DueDateTo         string
	OnlyUnsettled     bool
	OnlyOverdue       bool
	// 进度与完成筛选在 SQL 侧与 ResolveFinanceBillProgress 同源；空值/nil 表示不筛选。
	SettlementProgress FinanceBillSettlementProgress
	InvoiceProgress    FinanceBillInvoiceProgress
	Completed          *bool
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
	MatchedSummary FinanceLedgerMatchedSummary
	Items          []*FinanceBill
	Total          int64
	Summary        FinanceBillSummary
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
	// Protected 表示账单存在受保护财务事实，仓储只允许写回备注、对账标题与到期日。
	Protected bool
}

type FinanceBillGroupingPolicy struct {
	Mode           string
	SplitByOrder   bool
	SplitByTaxRate bool
}

type FinanceBillBatchPreviewGroup struct {
	CorrectionRequestID                                   *uuid.UUID
	Purpose                                               FinanceBillPurpose
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
	// CreditLimitAmount / CreditCurrency / CurrentUnsettledAmount 来自客户角色激活结算规则与
	// 已确认应收账单折本币未核销总额（本位币口径，仅提醒信息，不在建账入账环节拦截）。
	CreditLimitAmount      *decimal.Decimal
	CreditCurrency         *string
	CurrentUnsettledAmount *decimal.Decimal
	IsCreditExceeded       bool
	// GroupToken 是该组费用事实与叶子配置的摘要；仅配置完整时签发，创建时逐组比对。
	GroupToken   string
	preparedBill *FinanceBill
	config       FinanceBillBatchPreviewGroupConfig
}

type FinanceBillBatchPreview struct {
	Groups       []*FinanceBillBatchPreviewGroup
	NettingPairs []*FinanceBillBatchNettingPair
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
	// GroupToken 取自预览中该组的令牌；FeeIDs 为预览归入该组的全部费用。
	GroupToken                string
	FeeIDs                    []uuid.UUID
	Invoices                  []FinanceBillBatchInvoiceInput
	Verifications             []FinanceBillBatchVerificationInput
	OperationInputError       *FinanceBillBatchOperationInputError
	OperationInputFingerprint string
}

type CreateFinanceBillBatchInput struct {
	GroupingPolicy FinanceBillGroupingPolicy
	Groups         []CreateFinanceBillBatchGroupInput
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

// FinanceBillBatch 的批次头只保存身份；FeeCount、BillCount、BaseTotals 由已生成账单在读取时聚合。
type FinanceBillBatch struct {
	ID, OrganizationID, CreatedBy        uuid.UUID
	BatchNo, IdempotencyKey, RequestHash string
	GroupingPolicy                       FinanceBillGroupingPolicy
	FeeCount, BillCount                  int
	BaseTotals                           []FinanceBillBatchBaseTotal
	Bills                                []*FinanceBill
	Nettings                             []*FinanceNetting
	CreatedAt, UpdatedAt                 time.Time
}

// FinanceBillBatchBaseTotal 是批次内同一本位币账单的本位币合计，不跨本位币相加。
type FinanceBillBatchBaseTotal struct {
	BaseCurrency    string
	TotalBaseAmount decimal.Decimal
}

// FinanceBillBatchGroupResult 是批量建账单组的结果：成功时 Bill 非空，失败时 Err 非空。
type FinanceBillBatchGroupResult struct {
	GroupKey string
	Bill     *FinanceBill
	Err      error
}

// FinanceBillBatchCreateResult 汇总批次头（含已成功账单）与按 group_key 升序的逐组结果。
type FinanceBillBatchCreateResult struct {
	Batch  *FinanceBillBatch
	Groups []*FinanceBillBatchGroupResult
}

// FinanceBillBatchWrite 是一次组事务的写入内容。CreateHead 为真时（对冲整批原子模式）
// 批次头在同一事务内创建；普通模式批次头已由独立短事务写入。
type FinanceBillBatchWrite struct {
	Batch         *FinanceBillBatch
	ActorID       uuid.UUID
	CreateHead    bool
	HeadAudit     *AuditEvent
	Bills         []*FinanceBill
	BillAudits    []*AuditEvent
	Nettings      []*FinanceNetting
	NettingAudits []*AuditEvent
}

type FinanceBillRepo interface {
	List(ctx context.Context, organizationIDs []uuid.UUID, filter FinanceBillFilter) (*FinanceBillListResult, error)
	Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceBill, error)
	GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBill, error)
	GetBatchByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*FinanceBillBatch, error)
	GetBatch(ctx context.Context, organizationIDs []uuid.UUID, batchID uuid.UUID) (*FinanceBillBatch, error)
	LoadBillableFees(ctx context.Context, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error)
	LoadBillableFeesScoped(ctx context.Context, organizationIDs []uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error)
	ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter FinanceBillCreationCandidateFilter) (*FinanceBillCreationCandidateResult, error)
	Create(ctx context.Context, bill *FinanceBill, audit *AuditEvent) (*FinanceBill, error)
	// CreateBatchHead 以独立短事务写入批次头（分配批次号），同幂等键唯一冲突返回批次冲突。
	CreateBatchHead(ctx context.Context, batch *FinanceBillBatch, audit *AuditEvent) (*FinanceBillBatch, error)
	// CreateBatchBills 在调用方事务内锁定并复核费用后写入账单（及对冲单）。
	CreateBatchBills(ctx context.Context, write FinanceBillBatchWrite) error
	ValidateBillCurrencies(ctx context.Context, currencies []string) error
	HydrateBillSettlementAccounts(ctx context.Context, bills []*FinanceBill) error
	Update(ctx context.Context, organizationIDs []uuid.UUID, input UpdateFinanceBillInput, audit *AuditEvent) (*FinanceBill, error)
	// HasProtectedFinanceFact 在已锁定账单的事务内判断是否存在有效发票、核销或已确认对冲。
	HasProtectedFinanceFact(ctx context.Context, billID uuid.UUID) (bool, error)
	Cancel(ctx context.Context, organizationIDs []uuid.UUID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *AuditEvent) (*FinanceBill, error)
	// ListBillFeeCandidates 返回可加入账单的未建账费用（同公司、同方向、同结算单位、同币种与本币），服务端关键字过滤分页。
	ListBillFeeCandidates(ctx context.Context, bill *FinanceBill, filter FinanceBillFeeCandidateFilter) (*FinanceBillCreationCandidateResult, error)
	// UpdateFees 须在共享事务内调用：按「订单闭包升序 → 账单 → 费用升序 → 账单行」加锁，锁内复核费用锁定事实、
	// 逐项校验并按账单汇率重算金额，写入后执行提成经济重算与审计。
	UpdateFees(ctx context.Context, organizationIDs []uuid.UUID, input UpdateFinanceBillFeesInput, audit *AuditEvent) error
	// GetPartnerUnsettledReceivableBaseAmount 返回指定往来户在组织内已确认应收账单的折本币未核销总额
	// （总额 - 有效核销 - 有效对冲，负值钳零），口径与列表汇总的 unverified_base_amount 一致。
	GetPartnerUnsettledReceivableBaseAmount(ctx context.Context, organizationID, partnerID uuid.UUID) (decimal.Decimal, error)
	// GetPartnerCreditSummaries 批量返回往来户的默认账期、信用额度（客户角色激活规则）与折本币未核销应收总额。
	// 未配置客户角色激活规则的往来户同样返回余额，但额度为空，超额判定恒为否。
	GetPartnerCreditSummaries(ctx context.Context, organizationID uuid.UUID, partnerIDs []uuid.UUID) (map[uuid.UUID]*PartnerCreditSummary, error)
}

// PartnerCreditSummary 是信用额度判定的输入真相：额度来自客户角色激活结算规则，
// 未核销余额复用账单未核销折本币口径；多个激活规则取最大额度与最大账期（不收窄商业弹性）。
type PartnerCreditSummary struct {
	PartnerID               uuid.UUID
	DefaultPaymentTermsDays *int
	CreditLimitBase         *decimal.Decimal
	CreditCurrency          *string
	UnsettledReceivableBase decimal.Decimal
}

// PartnerCreditExceeded 判定往来户是否超额：仅当设置了大于 0 的信用额度且
// 未核销本币总额严格大于额度时成立；未设额度与散客（不配置额度）不触发。
func PartnerCreditExceeded(summary *PartnerCreditSummary) bool {
	if summary == nil || summary.CreditLimitBase == nil || !summary.CreditLimitBase.IsPositive() {
		return false
	}
	return summary.UnsettledReceivableBase.GreaterThan(*summary.CreditLimitBase)
}
