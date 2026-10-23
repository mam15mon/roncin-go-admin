package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrFeeReviewInvalid           = errors.BadRequest("FEE_REVIEW_INVALID_ARGUMENT", "费用审核参数不合法")
	ErrFeeReviewConflict          = errors.Conflict("FEE_REVIEW_CONFLICT", "费用或审核已被更新，请刷新后重试")
	ErrFeeReviewForbidden         = errors.Forbidden("FEE_REVIEW_FORBIDDEN", "没有覆盖该订单组织的费用审核权限")
	ErrFeeReviewFrozen            = errors.Conflict("FEE_REVIEW_FROZEN", "费用审核中或已批准，不能直接修改；请撤回或按流程申请补录")
	ErrFeeReviewNotApproved       = errors.Conflict("FEE_REVIEW_NOT_APPROVED", "整票费用尚未有效批准，不能建账或处理提成")
	ErrFeeReviewPendingSupplement = errors.Conflict("FEE_REVIEW_PENDING_SUPPLEMENT", "存在待审核的费用补录，请先处理补录申请")
	ErrFeeReviewNoApprover        = errors.Conflict("FEE_REVIEW_NO_APPROVER", "没有具备该订单费用审核资格的成员，请先配置角色权限")
)

type FeeReviewSummary struct {
	BaseCurrency    string
	Receivable      decimal.Decimal
	Payable         decimal.Decimal
	Profit          decimal.Decimal
	ProfitRate      *decimal.Decimal
	ReceivableCount int
	PayableCount    int
	SubmittedByName string
	SubmittedAt     *time.Time
}

// FeeReviewSnapshot 排除建账关联和通用行版本，只固化审核时的完整费用内容。
type FeeReviewSnapshot struct {
	EncodingVersion string
	Fees            []*OrderFee
	Summary         FeeReviewSummary
}

func BuildFeeReviewSnapshot(fees []*OrderFee) (*FeeReviewSnapshot, string, error) {
	snapshot := &FeeReviewSnapshot{EncodingVersion: "ORDER_FEE_REVIEW_V1", Fees: make([]*OrderFee, 0, len(fees))}
	for _, fee := range fees {
		if fee != nil && !IsCurrentFee(fee) {
			continue
		}
		if fee == nil || fee.ID == uuid.Nil || fee.BaseCurrency == "" || !fee.ExchangeRate.IsPositive() || !fee.BaseCurrencyAmount.Equal(fee.TotalAmount.Mul(fee.ExchangeRate).RoundBank(8)) {
			return nil, "", ErrFeeReviewInvalid
		}
		if snapshot.Summary.BaseCurrency == "" {
			snapshot.Summary.BaseCurrency = fee.BaseCurrency
		}
		if snapshot.Summary.BaseCurrency != fee.BaseCurrency {
			return nil, "", ErrFeeReviewInvalid
		}
		copy := *fee
		copy.Version = 0
		copy.HasActiveBill = false
		copy.CanEdit = false
		copy.CanDelete = false
		copy.CanBill = false
		copy.CanCorrect = false
		copy.HasPendingChange = false
		copy.BlockedReasons = nil
		copy.CreatedAt = time.Time{}
		copy.UpdatedAt = time.Time{}
		copy.IdempotencyKey = ""
		copy.ExchangeRateOverride = nil
		copy.TaxRateOverride = nil
		copy.FeeNameOverride = nil
		copy.AccountingRequired = false
		snapshot.Fees = append(snapshot.Fees, &copy)
		switch fee.Direction {
		case OrderFeeReceivable:
			snapshot.Summary.Receivable = snapshot.Summary.Receivable.Add(fee.BaseCurrencyAmount)
			snapshot.Summary.ReceivableCount++
		case OrderFeePayable:
			snapshot.Summary.Payable = snapshot.Summary.Payable.Add(fee.BaseCurrencyAmount)
			snapshot.Summary.PayableCount++
		default:
			return nil, "", ErrFeeReviewInvalid
		}
	}
	sort.Slice(snapshot.Fees, func(i, j int) bool { return snapshot.Fees[i].ID.String() < snapshot.Fees[j].ID.String() })
	snapshot.Summary.Profit = snapshot.Summary.Receivable.Sub(snapshot.Summary.Payable)
	if !snapshot.Summary.Receivable.IsZero() {
		rate := snapshot.Summary.Profit.Div(snapshot.Summary.Receivable).Mul(decimal.NewFromInt(100)).Round(8)
		snapshot.Summary.ProfitRate = &rate
	}
	fingerprintSnapshot := *snapshot
	fingerprintSnapshot.Fees = make([]*OrderFee, 0, len(snapshot.Fees))
	for _, fee := range snapshot.Fees {
		economic := *fee
		economic.SettlementPartyName = ""
		fingerprintSnapshot.Fees = append(fingerprintSnapshot.Fees, &economic)
	}
	payload, err := json.Marshal(&fingerprintSnapshot)
	if err != nil {
		return nil, "", err
	}
	return snapshot, FeeReviewHash(payload), nil
}
func FeeReviewHash(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

type FeeReviewEvent struct {
	ID             uuid.UUID
	Action         string
	ActorID        uuid.UUID
	ActorName      string
	Comment        string
	CreatedAt      time.Time
	ContentVersion uint64
	SubmissionID   *uuid.UUID
	Summary        FeeReviewSummary
}
type OrderFeeReview struct {
	OrderID                  uuid.UUID
	OrderNo                  string
	Status                   string
	ContentVersion           uint64
	ReviewVersion            uint64
	SubmissionID             *uuid.UUID
	ApprovedEventID          *uuid.UUID
	PreviewToken             string
	Summary                  *FeeReviewSummary
	Fees                     []*OrderFee
	History                  []*FeeReviewEvent
	CanSubmit                bool
	CanWithdraw              bool
	CanApprove               bool
	CanRevoke                bool
	CanEdit                  bool
	CanBill                  bool
	HasPendingSupplement     bool
	HasPendingCorrection     bool
	HasPendingChange         bool
	BlockedReasons           []string
	NotificationFailedCount  int
	NotificationUnboundCount int
}
type OrderFeeReviewRepo interface {
	Get(context.Context, *Principal, uuid.UUID) (*OrderFeeReview, error)
}
type OrderFeeReviewUsecase struct{ repo OrderFeeReviewRepo }

func NewOrderFeeReviewUsecase(repo OrderFeeReviewRepo) *OrderFeeReviewUsecase {
	return &OrderFeeReviewUsecase{repo: repo}
}
func (u *OrderFeeReviewUsecase) Get(ctx context.Context, p *Principal, id uuid.UUID) (*OrderFeeReview, error) {
	return u.repo.Get(ctx, p, id)
}
