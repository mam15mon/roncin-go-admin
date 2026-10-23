package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

// OrderFeeEconomicEffect 保留收入/成本属性，只对经济金额取符号。
type OrderFeeEconomicEffect string

const (
	OrderFeeEconomicIncrease OrderFeeEconomicEffect = "INCREASE"
	OrderFeeEconomicDecrease OrderFeeEconomicEffect = "DECREASE"
)

func EconomicAmount(fee *OrderFee) decimal.Decimal {
	if fee.EconomicEffect == OrderFeeEconomicDecrease {
		return fee.BaseCurrencyAmount.Neg()
	}
	return fee.BaseCurrencyAmount
}
func SettlementDirection(fee *OrderFee) OrderFeeDirection {
	if fee.EconomicEffect != OrderFeeEconomicDecrease {
		return fee.Direction
	}
	if fee.Direction == OrderFeeReceivable {
		return OrderFeePayable
	}
	return OrderFeeReceivable
}
func IsCurrentFee(fee *OrderFee) bool {
	return fee != nil && fee.EconomicEffect != OrderFeeEconomicDecrease && fee.SupersededByRequestID == nil
}
func RequiresAccounting(fee *OrderFee) bool {
	return IsCurrentFee(fee) && fee.AccountingRequired
}

var (
	ErrFeeCorrectionInvalid   = errors.BadRequest("FEE_CORRECTION_INVALID", "费用更正参数不合法")
	ErrFeeCorrectionNotFound  = errors.NotFound("FEE_CORRECTION_NOT_FOUND", "费用更正申请不存在")
	ErrFeeCorrectionConflict  = errors.Conflict("FEE_CORRECTION_CONFLICT", "费用更正已被更新或预览已过期，请刷新后重试")
	ErrFeeCorrectionPending   = errors.Conflict("FEE_CORRECTION_PENDING", "存在待审核补录或费用更正，请先处理")
	ErrFeeCorrectionSource    = errors.Conflict("FEE_CORRECTION_SOURCE_INVALID", "原费用已被更正，请选择当前有效费用")
	ErrFeeCorrectionProtected = errors.Conflict("FEE_CORRECTION_PROTECTED", "更正费用及其来源必须保留，请通过费用更正处理")
)

type OrderFeeCorrectionRequest struct {
	ID                        uuid.UUID
	OrganizationID            uuid.UUID
	OrderID                   uuid.UUID
	Operation                 string
	SourceFeeID               *uuid.UUID
	Before                    *OrderFee
	After                     *OrderFee
	ApprovedFeeReviewEventID  uuid.UUID
	LockRecordID              uuid.UUID
	LockGeneration            uint64
	ApprovedFeeContentVersion uint64
	ApprovedFeeReviewVersion  uint64
	Reason                    string
	IdempotencyKey            string
	RequestFingerprint        string
	RequestedBy               uuid.UUID
	RequestedByName           string
	RequestedAt               time.Time
	Status                    string
	Version                   uint64
	DecidedBy                 *uuid.UUID
	DecidedByName             string
	DecidedAt                 *time.Time
	DecisionReason            *string
	ApprovalToken             *string
	CanApprove                bool
	CanWithdraw               bool
	ReversalFeeID             *uuid.UUID
	FinancialStatus           string
	ApprovalResult            *FeeCorrectionApprovalResult
}
type OrderFeeCorrectionInput struct {
	Operation          string
	SourceFeeID        *uuid.UUID
	ExpectedFeeVersion uint64
	After              *OrderFee
	Reason             string
	IdempotencyKey     string
}
type FeeCorrectionCommissionLineImpact struct {
	CommissionID   uuid.UUID
	CommissionNo   string
	OrderID        uuid.UUID
	EmployeeID     uuid.UUID
	EmployeeName   string
	BaseCurrency   string
	OriginalAmount decimal.Decimal
	CurrentAmount  decimal.Decimal
	TargetAmount   decimal.Decimal
	Difference     decimal.Decimal
}
type FeeCorrectionCommissionImpact struct {
	Fingerprint string
	Lines       []*FeeCorrectionCommissionLineImpact
}

// FeeCorrectionCommissionRepo 必须在共享事务内读取、锁定并复算；不得用空实现跳过提成。
type FeeCorrectionCommissionRepo interface {
	PreviewCorrectionImpact(context.Context, uuid.UUID, uuid.UUID, *OrderFeeCorrectionRequest, []*OrderFee, []*OrderFee) (*FeeCorrectionCommissionImpact, error)
	ReconcileCorrectionImpact(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *OrderFeeCorrectionRequest, *FeeCorrectionCommissionImpact) error
}
type OrderFeeCorrectionPreview struct {
	Request              *OrderFeeCorrectionRequest
	Before               *FeeReviewSnapshot
	After                *FeeReviewSnapshot
	CommissionImpact     *FeeCorrectionCommissionImpact
	PreviewToken         string
	FinancialFingerprint string
}
type OrderFeeCorrectionRepo interface {
	Authorize(context.Context, *Principal, uuid.UUID, uuid.UUID, string) error
	LockOrder(context.Context, uuid.UUID, uuid.UUID) (*OrderFeeSupplementLockEvidence, error)
	CurrentFees(context.Context, uuid.UUID) ([]*OrderFee, error)
	ValidateTarget(context.Context, uuid.UUID, uuid.UUID, *OrderFeeSupplementFeeSnapshot) error
	Get(context.Context, uuid.UUID, uuid.UUID) (*OrderFeeCorrectionRequest, error)
	GetByKey(context.Context, uuid.UUID, string) (*OrderFeeCorrectionRequest, error)
	List(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*OrderFeeCorrectionRequest, int, error)
	FinancialFingerprint(context.Context, *OrderFeeCorrectionRequest) (string, error)
	LockRequest(context.Context, uuid.UUID, uuid.UUID) (*OrderFeeCorrectionRequest, error)
	Create(context.Context, *OrderFeeCorrectionRequest, *AuditEvent) (*OrderFeeCorrectionRequest, error)
	Apply(context.Context, *OrderFeeCorrectionRequest, uuid.UUID, string) error
	SaveDecision(context.Context, *OrderFeeCorrectionRequest, uuid.UUID, string, string, string) error
}

func feeCorrectionHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

type FeeCorrectionApprovalResult struct {
	Current              FeeReviewSummary
	Projected            FeeReviewSummary
	CommissionImpact     *FeeCorrectionCommissionImpact
	FinancialFingerprint string
}
