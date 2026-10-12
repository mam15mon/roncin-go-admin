package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFeeSupplementNotFound            = errors.NotFound("FEE_SUPPLEMENT_REQUEST_NOT_FOUND", "补录费用申请不存在")
	ErrFeeSupplementInvalidArgument     = errors.BadRequest("FEE_SUPPLEMENT_REQUEST_INVALID", "补录费用申请参数不合法")
	ErrFeeSupplementIdempotencyConflict = errors.Conflict("FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT", "同一幂等键的补录申请内容已变化，请刷新后重新发起")
	ErrFeeSupplementApproverUnavailable = errors.Conflict("FEE_SUPPLEMENT_APPROVER_UNAVAILABLE", "当前没有具备费用审核权限且组织范围匹配的审批人，请先配置审批资格")
	ErrFeeSupplementTransition          = errors.Conflict("FEE_SUPPLEMENT_TRANSITION", "当前补录申请状态不允许该操作")
	// ErrFeeSupplementNotApplicable 当前整票费用无有效批准，不能提交受控成本补录。
	ErrFeeSupplementNotApplicable = errors.Conflict("FEE_SUPPLEMENT_NOT_APPLICABLE", "订单费用尚未有效批准，请先完成费用审核")
	// ErrFeeSupplementCancelBlocked 专用撤销被下游事实阻断；消息由仓储按具体
	// 阻断原因动态构造，稳定 reason 便于前端识别。
	ErrFeeSupplementCancelBlocked = errors.Conflict("FEE_SUPPLEMENT_CANCEL_BLOCKED", "当前补录费用不满足专用撤销条件")
)

// OrderFeeSupplementStatus 补录申请状态：PENDING 是唯一可流转状态；
// APPROVED、REJECTED、WITHDRAWN 均为终态。
type OrderFeeSupplementStatus string

const (
	OrderFeeSupplementPending   OrderFeeSupplementStatus = "PENDING"
	OrderFeeSupplementApproved  OrderFeeSupplementStatus = "APPROVED"
	OrderFeeSupplementRejected  OrderFeeSupplementStatus = "REJECTED"
	OrderFeeSupplementWithdrawn OrderFeeSupplementStatus = "WITHDRAWN"
)

// Valid 判断状态取值是否已登记。
func (s OrderFeeSupplementStatus) Valid() bool {
	switch s {
	case OrderFeeSupplementPending, OrderFeeSupplementApproved, OrderFeeSupplementRejected, OrderFeeSupplementWithdrawn:
		return true
	default:
		return false
	}
}

// Terminal 报告状态是否为终态。
func (s OrderFeeSupplementStatus) Terminal() bool {
	return s != OrderFeeSupplementPending
}

// OrderFeeSupplementLockBasis 申请提交时固化的锁依据类型。
type OrderFeeSupplementLockBasis string

const (
	// SupplementLockBasisBusiness 保留历史业务锁事实，不授权新动作。
	SupplementLockBasisBusiness OrderFeeSupplementLockBasis = "BUSINESS"
	// SupplementLockBasisFinancial 保留历史财务锁事实，不授权新动作。
	SupplementLockBasisFinancial OrderFeeSupplementLockBasis = "FINANCIAL"
	// SupplementLockBasisBoth 保留历史双锁事实；新动作不据此授权。
	SupplementLockBasisBoth     OrderFeeSupplementLockBasis = "BOTH"
	SupplementLockBasisApproval OrderFeeSupplementLockBasis = "APPROVAL"
)

// Valid 判断锁依据取值是否已登记。
func (b OrderFeeSupplementLockBasis) Valid() bool {
	switch b {
	case SupplementLockBasisBusiness, SupplementLockBasisFinancial, SupplementLockBasisBoth, SupplementLockBasisApproval:
		return true
	default:
		return false
	}
}

// OrderFeeSupplementFeeSnapshot 是申请固化的不可变应付费用快照，字段语义与
// OrderFee 领域对象一致；审批通过时按快照原样入账，不允许审批过程修改金额
// 或关键字段。
type OrderFeeSupplementFeeSnapshot struct {
	Direction             OrderFeeDirection
	FeeSettingID          *uuid.UUID
	FeeCode               string
	FeeName               string
	FeeNameEN             *string
	SettlementPartyID     uuid.UUID
	SettlementPartyName   string
	BillingUnitID         *uuid.UUID
	BillingUnit           string
	TaxRate               *decimal.Decimal
	TaxableServiceName    *string
	Quantity              decimal.Decimal
	UnitPrice             decimal.Decimal
	TotalAmount           decimal.Decimal
	TaxInclusive          bool
	NetAmount             decimal.Decimal
	TaxAmount             decimal.Decimal
	Currency              string
	ExchangeRate          decimal.Decimal
	ExchangeRateSource    string
	ExchangeRateDate      string
	ExchangeRateSettingID *uuid.UUID
	BaseCurrency          string
	BaseCurrencyAmount    decimal.Decimal
	ExpenseDate           string
	Note                  *string
}

// Validate 校验费用快照的领域规则：补录仅接受正数应付成本，应收方向在领域
// 边界拒绝（与 API 校验双重拒绝一致），金额正数由普通费用参数规则复核。
func (f OrderFeeSupplementFeeSnapshot) Validate() error {
	if f.Direction != OrderFeePayable {
		return ErrFeeSupplementInvalidArgument
	}
	if f.TotalAmount.Sign() <= 0 || f.NetAmount.Sign() <= 0 {
		return ErrFeeSupplementInvalidArgument
	}
	return nil
}

// OrderFeeSupplementRequest 是费用批准后的受控应付补录申请领域对象。除状态流转字段
// 外全部不可变；批准依据在提交事务的 Order 行锁内由服务端固化。
type OrderFeeSupplementRequest struct {
	ID, OrganizationID, OrderID  uuid.UUID
	ApprovedFeeReviewEventID     *uuid.UUID
	ApprovedFeeContentVersion    *uint64
	LockBasis                    OrderFeeSupplementLockBasis
	BusinessLockGeneration       *uint64
	FinancialLockEvidenceVersion *string
	FinancialLockEvidenceHash    *string
	FinancialLockNetAmount       *decimal.Decimal
	IdempotencyKey               string
	RequestFingerprint           string
	Fee                          OrderFeeSupplementFeeSnapshot
	Reason                       string
	RequestedBy                  uuid.UUID
	RequestedByName              string
	RequestedAt                  time.Time
	Status                       OrderFeeSupplementStatus
	Version                      uint64
	DecidedBy                    *uuid.UUID
	DecidedByName                *string
	DecidedAt                    *time.Time
	DecisionReason               *string
	CreatedAt, UpdatedAt         time.Time
}

// ValidateLockBasisEvidence 新申请仅接受完整的费用批准依据；旧枚举仅供历史投影。
func (r *OrderFeeSupplementRequest) ValidateLockBasisEvidence() error {
	if r.LockBasis == SupplementLockBasisApproval && r.ApprovedFeeReviewEventID != nil && *r.ApprovedFeeReviewEventID != uuid.Nil && r.ApprovedFeeContentVersion != nil && *r.ApprovedFeeContentVersion > 0 {
		return nil
	}
	return ErrFeeSupplementInvalidArgument
}

// Validate 汇总申请创建时的纯领域校验：只能以 PENDING 创建、状态与锁依据取值
// 合法、锁依据组合完整、费用快照为正数应付。
func (r *OrderFeeSupplementRequest) Validate() error {
	if r.Status != OrderFeeSupplementPending {
		return ErrFeeSupplementInvalidArgument
	}
	if !r.LockBasis.Valid() {
		return ErrFeeSupplementInvalidArgument
	}
	if r.OrganizationID == uuid.Nil || r.OrderID == uuid.Nil || r.RequestedBy == uuid.Nil ||
		r.IdempotencyKey == "" || r.RequestFingerprint == "" || r.Reason == "" {
		return ErrFeeSupplementInvalidArgument
	}
	if err := r.ValidateLockBasisEvidence(); err != nil {
		return err
	}
	return r.Fee.Validate()
}

// OrderFeeSupplementRequestRepo 补录申请持久化接口。
// 实现约束：审批、驳回与撤回必须在申请行 FOR UPDATE 锁内比对乐观锁版本；
// 费用快照与批准依据字段创建后不可变；组织级幂等键唯一由数据库唯一索引兜底，
// 不得用先查后插替代。
type OrderFeeSupplementRequestRepo interface {
	// Create 幂等创建申请：同组织同幂等键同指纹返回既有申请，同键不同指纹
	// 返回 ErrFeeSupplementIdempotencyConflict；审计与逐审批人通知由调用方
	// 在同一共享事务内随申请一并写入。
	Create(ctx context.Context, request *OrderFeeSupplementRequest, audit *AuditEvent) (*OrderFeeSupplementRequest, error)
	// Get 读取申请及其不可变费用快照。
	Get(ctx context.Context, organizationID, id uuid.UUID) (*OrderFeeSupplementRequest, error)
	// ListByOrder 列出目标订单的全部补录申请。
	ListByOrder(ctx context.Context, organizationID, orderID uuid.UUID) ([]*OrderFeeSupplementRequest, error)
	// LockForDecision 在事务内锁定申请行并重读最新状态，供审批、驳回与撤回
	// 在同一行锁内竞争，先提交的一方成功，失败方返回状态冲突。
	LockForDecision(ctx context.Context, organizationID, id uuid.UUID) (*OrderFeeSupplementRequest, error)
	// SaveDecision 在锁内写入终态与决策字段，版本号自增；只接受状态机的合法
	// 流转。
	SaveDecision(ctx context.Context, request *OrderFeeSupplementRequest) error

	// LockOrderForSupplement 按主键锁目标订单，复核当前整票有效批准；
	// 这是批准冻结下的受控成本补录入口，不能复用解锁授权。
	LockOrderForSupplement(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeSupplementLockEvidence, error)
	// ReadLockEvidence 读取当前有效批准版本，供审批预览复核。
	ReadLockEvidence(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeSupplementLockEvidence, error)
	// GetOrderRef 读取订单号与业务类型，供权限判定与通知摘要使用。
	GetOrderRef(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeSupplementOrderRef, error)
	// ListLockGrantApprovers 解析当前实时具备目标订单费用审核权限的审批候选人
	//（按 UserID 升序去重；bootstrap admin 不进入候选池）。同一目标订单与组织
	// 范围内的解析必须与单用户资格判断复用同一谓词。
	ListLockGrantApprovers(ctx context.Context, organizationID, orderID uuid.UUID) ([]OrderFeeSupplementApprover, error)
	// HasRealtimeLockGrant 实时判断费用审核资格及组织范围，不复用解锁授权或管理员旁路。
	HasRealtimeLockGrant(ctx context.Context, organizationID, orderID, userID uuid.UUID, isBootstrapAdmin bool) (bool, error)
	// CancelCapability 只读投影专用作废能力与稳定阻断原因，列表与专用作废
	// 命令复用同一校验口径。
	CancelCapability(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*OrderFeeSupplementCancelCapability, error)
	// ResolveFeeFactsForApproval 在审批事务的锁内重新解析结算对象、费用项、
	// 计费单位、币种与费用发生日汇率，形成待创建费用但不落库；任一引用失效
	// 时返回对应业务错误，整体回滚。
	ResolveFeeFactsForApproval(ctx context.Context, organizationID, orderID uuid.UUID, snapshot *OrderFeeSupplementFeeSnapshot) (*OrderFee, error)
	// LockCommissionImpactContext 按父单 UUID 升序 FOR UPDATE 锁定包含目标订单
	// 的 CONFIRMED/PAID 提成父单、目标订单行与会影响余额的调整，返回边际影响
	// 计算上下文（含此前未作废补录应付本位币合计）。
	LockCommissionImpactContext(ctx context.Context, organizationID, orderID uuid.UUID) (*OrderFeeSupplementImpactContext, error)
	// CreateApprovedFee 在审批事务内创建与申请一对一关联的 UNBILLED 补录费用；
	// 幂等键由申请 ID 派生，supplement_request_id 反向关联。
	CreateApprovedFee(ctx context.Context, organizationID, requestID uuid.UUID, fee *OrderFee, audit *AuditEvent) (*OrderFee, error)
	// CreateDecreaseSuggestion 在审批事务内创建 DECREASE+CONFIRMED+LOCKED_FEE_SUPPLEMENT
	// 调整：父单行内分配调整号并递增序号，来源关联必须指向补录申请。
	CreateDecreaseSuggestion(ctx context.Context, organizationID, requestID uuid.UUID, suggestion *OrderFeeSupplementDecreaseSuggestion, audit *AuditEvent) error
	// EnqueueApprovalPendingNotifications 在创建申请的同一事务内为审批人逐人
	// 入队 FEE_SUPPLEMENT_APPROVAL_PENDING 通知（确定性 ID + 幂等键）。
	EnqueueApprovalPendingNotifications(ctx context.Context, organizationID, requestID uuid.UUID, orderNo string, totalAmount decimal.Decimal, currency string, approvers []OrderFeeSupplementApprover) error
	// EnqueueDecreaseSuggestedNotifications 在审批事务内为实际生成已确认提成冲减的
	// 员工逐人入队 COMMISSION_DECREASE_SUGGESTED 通知（确定性 ID + 幂等键）。
	EnqueueDecreaseSuggestedNotifications(ctx context.Context, organizationID uuid.UUID, orderNo string, suggestions []OrderFeeSupplementDecreaseSuggestion) error
	// CancelApprovedFee 执行专用作废事务（design 5.1 六步，固定锁序 Order →
	// 申请/费用 → 提成父单 → 调整），返回作废后的费用与被取消的调整 ID。
	CancelApprovedFee(ctx context.Context, input *OrderFeeSupplementCancelInput) (*OrderFeeSupplementCancelResult, error)
	// SaveAudit 在共享事务内写入业务审计；驳回与撤回等轻量决策复用本方法，
	// 避免为单一审计写入扩展独立仓储命令。
	SaveAudit(ctx context.Context, event *AuditEvent) error
}

// OrderFeeSupplementOrderRef 是权限判定与通知摘要所需的订单最小引用。
type OrderFeeSupplementOrderRef struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	OrderNo        string
	BusinessType   string
}

// OrderFeeSupplementLockEvidence 是 Order 行锁内读取的当前锁事实：业务锁以
// locked_at 与 lock_generation 为准；财务锁复用有效提成净额口径（净额 > 0），
// 并携带版本化证据三元组（CONFIRMED/PAID 统一编码为 ACTIVE）。
type OrderFeeSupplementLockEvidence struct {
	ApprovedFeeReviewEventID *uuid.UUID
	FeeContentVersion        uint64
	FeeReviewVersion         uint64
	OrderNo                  string
	BusinessType             string
	BusinessLocked           bool
	BusinessLockGeneration   uint64
	FinancialLocked          bool
	FinancialEvidenceVersion string
	FinancialEvidenceHash    string
	FinancialNetAmount       decimal.Decimal
}

// OrderFeeSupplementApprover 是有效费用审核授权及组织范围匹配的审批人快照，仅用于通知
// 收件人投影；审批资格始终以接口调用时的实时校验为准。
type OrderFeeSupplementApprover struct {
	UserID         uuid.UUID
	DisplayName    string
	DingTalkUserID string
}

// OrderFeeSupplementCommissionLine 是锁定后的一条原提成订单行复算上下文：
// 行输入全部来自原提成行固化的快照字段，余额来自父单与订单行的有效/草稿调整。
type OrderFeeSupplementCommissionLine struct {
	CommissionID       uuid.UUID
	CommissionNo       string
	CommissionStatus   string
	CommissionAmount   decimal.Decimal
	AdjustmentSequence uint64
	EmployeeID         uuid.UUID
	EmployeeName       string
	ParentBaseCurrency string
	LineID             uuid.UUID
	OrderID            uuid.UUID
	OrderNo            string
	// 冻结的复算输入。
	CalculationVersion      string
	CalculationBasis        CommissionCalculationBasis
	CalculationRatePercent  decimal.Decimal
	RealizedRevenue         decimal.Decimal
	LineCommissionAmount    decimal.Decimal
	LineBaseCurrency        string
	SnapshotStatus          string
	TotalReceivableSnapshot *decimal.Decimal
	TotalPayableSnapshot    *decimal.Decimal
	// 双层余额：CONFIRMED/PAID 符号化合计与其他 DRAFT DECREASE 预留。
	PriorSupplementBaseAmount decimal.Decimal
	LineEffective             decimal.Decimal
	LineDraftDecrease         decimal.Decimal
	ParentEffective           decimal.Decimal
	ParentDraftDecrease       decimal.Decimal
}

// OrderFeeSupplementImpactContext 是一次补录审批的边际影响计算上下文。
type OrderFeeSupplementImpactContext struct {
	// Lines 按父单 UUID 升序排列，每张父单至多一行（commission_id + order_id 唯一）。
	Lines []*OrderFeeSupplementCommissionLine
}

// OrderFeeSupplementDecreaseSuggestion 是审批计算得出的单条确认冲减及预览影响。
type OrderFeeSupplementDecreaseSuggestion struct {
	AdjustmentID     uuid.UUID
	CommissionID     uuid.UUID
	CommissionNo     string
	CommissionStatus string
	OrderID          uuid.UUID
	OrderNo          string
	EmployeeID       uuid.UUID
	EmployeeName     string
	BaseCurrency     string
	Amount           decimal.Decimal
	Reason           string
	// 计算审计：before/after 提成金额与理论边际、超出封顶金额。
	AmountBefore  decimal.Decimal
	AmountAfter   decimal.Decimal
	MarginalDelta decimal.Decimal
	ExcessAmount  decimal.Decimal
}

// OrderFeeSupplementCancelCapability 是专用撤销的只读能力投影。
type OrderFeeSupplementCancelCapability struct {
	Cancellable     bool
	BlockReasonCode string
	BlockReason     string
	// FeeID 是 APPROVED 申请当前生成的费用 ID；生成费用已专用撤销删除时为空。
	FeeID *uuid.UUID
}

// OrderFeeSupplementCancelInput 是专用作废命令的事务入参。
type OrderFeeSupplementCancelInput struct {
	OrganizationID     uuid.UUID
	OrderID            uuid.UUID
	RequestID          uuid.UUID
	OperatorID         uuid.UUID
	IsBootstrapAdmin   bool
	FeeExpectedVersion uint64
	Reason             string
	Audit              *AuditEvent
}

// OrderFeeSupplementCancelResult 是专用撤销事务的结果投影：领域层保留已删除
// 费用 ID 与被取消调整 ID 供内部联动，不再携带已不存在的费用对象。Request 是
// 撤销后同一事务内重读的申请（保持 APPROVED），View 是刷新后的能力投影，
// 供响应向调用方展示「生成费用已删除」。
type OrderFeeSupplementCancelResult struct {
	Request                *OrderFeeSupplementRequest
	View                   *OrderFeeSupplementRequestView
	DeletedFeeID           uuid.UUID
	CancelledAdjustmentIDs []uuid.UUID
}

// feeSupplementFingerprintVersion 是 request_fingerprint 规范编码版本；编码
// 字段或语义变化时必须递增，避免旧指纹与新指纹混淆。
const feeSupplementFingerprintVersion = "v1"

// BuildOrderFeeSupplementFingerprint 由服务端对订单 ID、不可变费用快照和补录
// 原因计算版本化 request_fingerprint：同一幂等键同指纹的语义重放返回原申请，
// 同键不同指纹返回稳定幂等冲突。
func BuildOrderFeeSupplementFingerprint(orderID uuid.UUID, fee OrderFeeSupplementFeeSnapshot, reason string) string {
	parts := []string{
		"fee-supplement-fingerprint/" + feeSupplementFingerprintVersion,
		"order|" + orderID.String(),
		"direction|" + string(fee.Direction),
		"fee_setting|" + uuidOrDash(fee.FeeSettingID),
		"fee_code|" + fee.FeeCode,
		"fee_name|" + fee.FeeName,
		"fee_name_en|" + stringOrDash(fee.FeeNameEN),
		"party|" + fee.SettlementPartyID.String(),
		"billing_unit_id|" + uuidOrDash(fee.BillingUnitID),
		"billing_unit|" + fee.BillingUnit,
		"tax_rate|" + decimalOrDash(fee.TaxRate, 2),
		"taxable_service|" + stringOrDash(fee.TaxableServiceName),
		"quantity|" + fee.Quantity.StringFixed(4),
		"unit_price|" + fee.UnitPrice.StringFixed(4),
		"total|" + fee.TotalAmount.StringFixed(8),
		"tax_inclusive|" + boolToFlag(fee.TaxInclusive),
		"net|" + fee.NetAmount.StringFixed(8),
		"tax|" + fee.TaxAmount.StringFixed(8),
		"currency|" + fee.Currency,
		"expense_date|" + fee.ExpenseDate,
		"note|" + stringOrDash(fee.Note),
		"reason|" + reason,
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:])
}

func uuidOrDash(value *uuid.UUID) string {
	if value == nil {
		return "-"
	}
	return value.String()
}

func stringOrDash(value *string) string {
	if value == nil {
		return "-"
	}
	return *value
}

func decimalOrDash(value *decimal.Decimal, scale int32) string {
	if value == nil {
		return "-"
	}
	return value.StringFixed(scale)
}

func boolToFlag(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

// CommissionSupplementCostSensitivity 按 calculation_version 显式路由成本敏感性：
// 当前 CUSTOMER_REALIZED_PROFIT_V3 下 REALIZED_PROFIT 受应付成本影响、
// REALIZED_REVENUE 不受影响；未知版本或未知口径一律失败关闭，禁止猜测放行。
func CommissionSupplementCostSensitivity(calculationVersion string, basis CommissionCalculationBasis) (bool, error) {
	switch calculationVersion {
	case CommissionCalculationVersion:
		switch basis {
		case CommissionBasisRealizedProfit:
			return true, nil
		case CommissionBasisRealizedRevenue:
			return false, nil
		}
	}
	return false, ErrCommissionCalculationVersionUnsupported
}

// supplementCommissionAmountAtPayable 是纯计算器：按历史快照分母与给定的应付
// 分母计算该行的提成金额（与 CalculateCommissionLine 同口径，逐次 8 位舍入）。
func supplementCommissionAmountAtPayable(realizedRevenue, totalReceivable, totalPayable, ratePercent decimal.Decimal, basis CommissionCalculationBasis) decimal.Decimal {
	if basis == CommissionBasisRealizedRevenue {
		// 收入口径不随应付成本变化。
		return realizedRevenue.Mul(ratePercent).Div(decimal.NewFromInt(100)).Round(8)
	}
	allocatedCost := realizedRevenue.Mul(totalPayable).Div(totalReceivable).Round(8)
	realizedProfit := realizedRevenue.Sub(allocatedCost).Round(8)
	if realizedProfit.IsNegative() {
		realizedProfit = decimal.Zero
	}
	return realizedProfit.Mul(ratePercent).Div(decimal.NewFromInt(100)).Round(8)
}

// SupplementMarginalImpact 是单条原提成订单行的边际影响计算结果。
type SupplementMarginalImpact struct {
	AmountBefore decimal.Decimal
	AmountAfter  decimal.Decimal
	Delta        decimal.Decimal
}

// ComputeSupplementMarginalImpact 计算单条原提成订单行的边际冲减差额：
// before 基线 = 历史快照分母 + 此前已审批且尚未作废的补录应付（历史计算事实，
// 已作废补录不进基线），after 再计入本笔费用本位币金额；前后提成金额均按
// 8 位舍入后取非负差额，零差不生成建议。每笔补录只计算自身增量，此前补录
// 已建议的影响不得在后续补录中重复计入。
func ComputeSupplementMarginalImpact(realizedRevenue, totalReceivable, totalPayableSnapshot, priorSupplementBase, feeBaseAmount, ratePercent decimal.Decimal, basis CommissionCalculationBasis) SupplementMarginalImpact {
	baselinePayable := totalPayableSnapshot.Add(priorSupplementBase)
	before := supplementCommissionAmountAtPayable(realizedRevenue, totalReceivable, baselinePayable, ratePercent, basis)
	after := supplementCommissionAmountAtPayable(realizedRevenue, totalReceivable, baselinePayable.Add(feeBaseAmount), ratePercent, basis)
	delta := before.Sub(after)
	if delta.Sign() <= 0 {
		delta = decimal.Zero
	}
	return SupplementMarginalImpact{AmountBefore: before, AmountAfter: after, Delta: delta.Round(8)}
}

// CapSupplementSuggestion 对边际差额执行订单行与父单双层余额封顶：建议金额取
// 理论差额、订单行可用余额、父单可用余额三者最小值；超出部分返回给调用方进入
// 审计，不转扣其他订单行、原提成或员工。
func CapSupplementSuggestion(marginal, lineAvailable, parentAvailable decimal.Decimal) (suggested, excess decimal.Decimal) {
	suggested = marginal
	if lineAvailable.LessThan(suggested) {
		suggested = lineAvailable
	}
	if parentAvailable.LessThan(suggested) {
		suggested = parentAvailable
	}
	if suggested.Sign() < 0 {
		suggested = decimal.Zero
	}
	excess = marginal.Sub(suggested)
	return suggested, excess
}
