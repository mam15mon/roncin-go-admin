package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFeeSupplementNotFound            = errors.NotFound("FEE_SUPPLEMENT_REQUEST_NOT_FOUND", "补录费用申请不存在")
	ErrFeeSupplementInvalidArgument     = errors.BadRequest("FEE_SUPPLEMENT_REQUEST_INVALID", "补录费用申请参数不合法")
	ErrFeeSupplementIdempotencyConflict = errors.Conflict("FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT", "同一幂等键的补录申请内容已变化，请刷新后重新发起")
	ErrFeeSupplementApproverUnavailable = errors.Conflict("FEE_SUPPLEMENT_APPROVER_UNAVAILABLE", "当前没有具备订单直接解锁资格的审批人，请先配置审批资格")
	ErrFeeSupplementLockBasisChanged    = errors.Conflict("LOCK_BASIS_CHANGED", "申请提交时的锁依据已全部失效，请根据当前锁状态重新发起补录或改走普通费用新增")
	ErrFeeSupplementTransition          = errors.Conflict("FEE_SUPPLEMENT_TRANSITION", "当前补录申请状态不允许该操作")
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

// CanTransitionTo 是补录申请状态机的唯一流转规则：PENDING 只能进入一个终态，
// 终态之间及向自身的流转一律拒绝；审批、驳回与撤回在申请行锁内竞争同一迁移，
// 只有先提交的一方成功。
func (s OrderFeeSupplementStatus) CanTransitionTo(target OrderFeeSupplementStatus) bool {
	if s != OrderFeeSupplementPending {
		return false
	}
	switch target {
	case OrderFeeSupplementApproved, OrderFeeSupplementRejected, OrderFeeSupplementWithdrawn:
		return true
	default:
		return false
	}
}

// OrderFeeSupplementLockBasis 申请提交时固化的锁依据类型。
type OrderFeeSupplementLockBasis string

const (
	// SupplementLockBasisBusiness 仅存在业务锁，依据为业务锁代次。
	SupplementLockBasisBusiness OrderFeeSupplementLockBasis = "BUSINESS"
	// SupplementLockBasisFinancial 仅存在财务锁，依据为版本化财务证据三元组。
	SupplementLockBasisFinancial OrderFeeSupplementLockBasis = "FINANCIAL"
	// SupplementLockBasisBoth 双锁并存，审批时任一依据匹配即可。
	SupplementLockBasisBoth OrderFeeSupplementLockBasis = "BOTH"
)

// Valid 判断锁依据取值是否已登记。
func (b OrderFeeSupplementLockBasis) Valid() bool {
	switch b {
	case SupplementLockBasisBusiness, SupplementLockBasisFinancial, SupplementLockBasisBoth:
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

// OrderFeeSupplementRequest 是锁后应付费用补录申请领域对象。除状态流转字段
// 外全部不可变；锁依据在提交事务的 Order 行锁内由服务端判定固化。
type OrderFeeSupplementRequest struct {
	ID, OrganizationID, OrderID  uuid.UUID
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
	RequestedAt                  time.Time
	Status                       OrderFeeSupplementStatus
	Version                      uint64
	DecidedBy                    *uuid.UUID
	DecidedAt                    *time.Time
	DecisionReason               *string
	CreatedAt, UpdatedAt         time.Time
}

// businessLockEvidenceComplete 报告业务锁依据是否为「大于零的锁代次」。
func (r *OrderFeeSupplementRequest) businessLockEvidenceComplete() bool {
	return r.BusinessLockGeneration != nil && *r.BusinessLockGeneration > 0
}

// businessLockEvidenceAbsent 报告业务锁依据是否完全未填写。
func (r *OrderFeeSupplementRequest) businessLockEvidenceAbsent() bool {
	return r.BusinessLockGeneration == nil
}

// financialLockEvidenceComplete 报告财务锁依据三元组是否完整且净额大于零。
func (r *OrderFeeSupplementRequest) financialLockEvidenceComplete() bool {
	return r.FinancialLockEvidenceVersion != nil && *r.FinancialLockEvidenceVersion != "" &&
		r.FinancialLockEvidenceHash != nil && *r.FinancialLockEvidenceHash != "" &&
		r.FinancialLockNetAmount != nil && r.FinancialLockNetAmount.Sign() > 0
}

// financialLockEvidenceAbsent 报告财务锁依据三元组是否完全未填写。
func (r *OrderFeeSupplementRequest) financialLockEvidenceAbsent() bool {
	return r.FinancialLockEvidenceVersion == nil &&
		r.FinancialLockEvidenceHash == nil &&
		r.FinancialLockNetAmount == nil
}

// ValidateLockBasisEvidence 校验锁依据字段组合与 lock_basis 一致，规则与数据库
// CHECK order_fee_supplement_requests_lock_basis_check 同源：BUSINESS 仅保存
// 大于零的业务锁代次；FINANCIAL 仅保存非空证据三元组且净额大于零；BOTH 同时
// 保存两组依据。
func (r *OrderFeeSupplementRequest) ValidateLockBasisEvidence() error {
	businessOK := r.businessLockEvidenceComplete()
	financialOK := r.financialLockEvidenceComplete()
	switch r.LockBasis {
	case SupplementLockBasisBusiness:
		if businessOK && r.financialLockEvidenceAbsent() {
			return nil
		}
	case SupplementLockBasisFinancial:
		if financialOK && r.businessLockEvidenceAbsent() {
			return nil
		}
	case SupplementLockBasisBoth:
		if businessOK && financialOK {
			return nil
		}
	default:
		return ErrFeeSupplementInvalidArgument
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
// 费用快照与锁依据字段创建后不可变；组织级幂等键唯一由数据库唯一索引兜底，
// 不得用先查后插替代。
type OrderFeeSupplementRequestRepo interface {
	// Create 幂等创建申请：同组织同幂等键同指纹返回既有申请，同键不同指纹
	// 返回 ErrFeeSupplementIdempotencyConflict。
	Create(ctx context.Context, request *OrderFeeSupplementRequest) (*OrderFeeSupplementRequest, error)
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
}
