package biz

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrVerificationNotFound   = errors.NotFound("FINANCE_VERIFICATION_NOT_FOUND", "核销记录不存在")
	ErrVerificationInvalid    = errors.BadRequest("FINANCE_VERIFICATION_INVALID", "核销参数不合法")
	ErrVerificationBalance    = errors.Conflict("FINANCE_VERIFICATION_BALANCE", "核销金额超过资金或账单未核销余额")
	ErrVerificationMismatch   = errors.BadRequest("FINANCE_VERIFICATION_MISMATCH", "核销双方方向、结算单位或币种不一致")
	ErrVerificationTransition = errors.Conflict("FINANCE_VERIFICATION_TRANSITION", "当前核销状态不允许该操作")
)

type VerificationStatus string

const (
	VerificationActive   VerificationStatus = "ACTIVE"
	VerificationReversed VerificationStatus = "REVERSED"
)

type VerificationAllocation struct {
	ID, VerificationID, CashflowID, BillID uuid.UUID
	CashflowNo, BillNo                     string
	Amount                                 decimal.Decimal
	Active                                 bool
}
type FinanceVerification struct {
	ID, OrganizationID, SettlementPartyID uuid.UUID
	VerificationNo, IdempotencyKey        string
	Status                                VerificationStatus
	Direction                             OrderFeeDirection
	SettlementPartyName, Currency         string
	Amount                                decimal.Decimal
	VerificationDate                      string
	Note                                  *string
	Version                               uint64
	ReversedAt                            *time.Time
	ReversedBy                            *uuid.UUID
	ReversalReason                        *string
	Allocations                           []*VerificationAllocation
	CreatedAt, UpdatedAt                  time.Time
}
type CreateVerificationInput struct {
	Allocations      []*VerificationAllocation
	VerificationDate string
	Note             *string
	IdempotencyKey   string
}
type VerificationFilter struct {
	Page, PageSize int
	Keyword        string
	Status         VerificationStatus
}
type VerificationListResult struct {
	Items []*FinanceVerification
	Total int64
}
type VerificationRepo interface {
	List(context.Context, uuid.UUID, VerificationFilter) (*VerificationListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceVerification, error)
	GetByKey(context.Context, uuid.UUID, string) (*FinanceVerification, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *FinanceVerification, *AuditEvent) (*FinanceVerification, error)
	Reverse(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, string, *AuditEvent) (*FinanceVerification, error)
}
type VerificationUsecase struct {
	repo   VerificationRepo
	config *OrderConfigUsecase
}

func NewVerificationUsecase(r VerificationRepo, c *OrderConfigUsecase) *VerificationUsecase {
	return &VerificationUsecase{r, c}
}
func (u *VerificationUsecase) List(ctx context.Context, org uuid.UUID, f VerificationFilter) (*VerificationListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || f.Page < 1 || f.PageSize < 1 || f.PageSize > 200 || (f.Status != "" && f.Status != VerificationActive && f.Status != VerificationReversed) {
		return nil, ErrVerificationInvalid
	}
	return u.repo.List(ctx, org, f)
}
func (u *VerificationUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateVerificationInput) (*FinanceVerification, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || len(in.Allocations) == 0 || len(in.Allocations) > 500 || !validFinanceDate(in.VerificationDate) || in.IdempotencyKey == "" {
		return nil, ErrVerificationInvalid
	}
	if old, e := u.repo.GetByKey(ctx, org, in.IdempotencyKey); e != nil {
		return nil, e
	} else if old != nil {
		return old, nil
	}
	id := uuid.Must(uuid.NewV7())
	v := &FinanceVerification{ID: id, OrganizationID: org, IdempotencyKey: in.IdempotencyKey, Status: VerificationActive, VerificationDate: in.VerificationDate, Note: in.Note, Version: 1, Allocations: in.Allocations}
	for _, a := range v.Allocations {
		if a == nil || a.CashflowID == uuid.Nil || a.BillID == uuid.Nil || !a.Amount.IsPositive() {
			return nil, ErrVerificationInvalid
		}
		a.ID = uuid.Must(uuid.NewV7())
		a.VerificationID = id
		a.Active = true
		v.Amount = v.Amount.Add(a.Amount)
	}
	no, e := u.config.NextNumber(ctx, org, DocumentTypeWriteOff)
	if e != nil {
		return nil, e
	}
	v.VerificationNo = no
	return u.repo.Create(ctx, org, actor, v, verifyAudit(org, actor, id, "finance.verification.create"))
}
func (u *VerificationUsecase) Reverse(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceVerification, error) {
	reason = strings.TrimSpace(reason)
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrVerificationInvalid
	}
	return u.repo.Reverse(ctx, org, id, actor, version, reason, verifyAudit(org, actor, id, "finance.verification.reverse"))
}
func verifyAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_verification", ResourceID: id.String()}
}
