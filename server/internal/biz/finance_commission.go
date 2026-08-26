package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrCommissionNotFound        = errors.NotFound("FINANCE_COMMISSION_NOT_FOUND", "提成记录不存在")
	ErrCommissionInvalid         = errors.BadRequest("FINANCE_COMMISSION_INVALID", "提成参数不合法")
	ErrCommissionSource          = errors.Conflict("FINANCE_COMMISSION_SOURCE", "仅有效应收核销可计提，且必须存在可计算的已实现收入")
	ErrCommissionDuplicate       = errors.Conflict("FINANCE_COMMISSION_DUPLICATE", "该核销与员工已存在提成记录")
	ErrCommissionTransition      = errors.Conflict("FINANCE_COMMISSION_TRANSITION", "当前提成状态不允许该操作")
	ErrVerificationHasCommission = errors.Conflict("FINANCE_VERIFICATION_HAS_COMMISSION", "核销已生成未取消提成，请先取消提成")
)

type CommissionStatus string

const (
	CommissionDraft     CommissionStatus = "DRAFT"
	CommissionConfirmed CommissionStatus = "CONFIRMED"
	CommissionPaid      CommissionStatus = "PAID"
	CommissionCancelled CommissionStatus = "CANCELLED"
)

type FinanceCommission struct {
	ID, OrganizationID, VerificationID, EmployeeID             uuid.UUID
	CommissionNo, IdempotencyKey, VerificationNo, EmployeeName string
	Status                                                     CommissionStatus
	BaseCurrency                                               string
	RealizedRevenue, AllocatedCost, RealizedProfit             decimal.Decimal
	RatePercent, CommissionAmount                              decimal.Decimal
	Note                                                       *string
	Version                                                    uint64
	ConfirmedAt, PaidAt, CancelledAt                           *time.Time
	CancellationReason                                         *string
	CreatedAt, UpdatedAt                                       time.Time
}

type CommissionFilter struct {
	Page, PageSize int
	Keyword        string
	Status         CommissionStatus
}
type CommissionListResult struct {
	Items []*FinanceCommission
	Total int64
}
type CommissionEmployeeOption struct {
	ID          uuid.UUID
	DisplayName string
}
type CreateCommissionInput struct {
	VerificationID, EmployeeID uuid.UUID
	RatePercent                decimal.Decimal
	Note                       *string
	IdempotencyKey             string
}

type CommissionRepo interface {
	List(context.Context, uuid.UUID, CommissionFilter) (*CommissionListResult, error)
	ListEmployees(context.Context, uuid.UUID) ([]*CommissionEmployeeOption, error)
	GetByKey(context.Context, uuid.UUID, string) (*FinanceCommission, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *FinanceCommission, *AuditEvent) (*FinanceCommission, error)
	Transition(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, CommissionStatus, string, *AuditEvent) (*FinanceCommission, error)
}

func (u *CommissionUsecase) ListEmployees(ctx context.Context, org uuid.UUID) ([]*CommissionEmployeeOption, error) {
	if org == uuid.Nil {
		return nil, ErrCommissionInvalid
	}
	return u.repo.ListEmployees(ctx, org)
}

type CommissionUsecase struct {
	repo   CommissionRepo
	config *OrderConfigUsecase
}

func NewCommissionUsecase(repo CommissionRepo, config *OrderConfigUsecase) *CommissionUsecase {
	return &CommissionUsecase{repo: repo, config: config}
}

func (u *CommissionUsecase) List(ctx context.Context, org uuid.UUID, f CommissionFilter) (*CommissionListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || f.Page < 1 || f.PageSize < 1 || f.PageSize > 200 || (f.Status != "" && f.Status != CommissionDraft && f.Status != CommissionConfirmed && f.Status != CommissionPaid && f.Status != CommissionCancelled) {
		return nil, ErrCommissionInvalid
	}
	return u.repo.List(ctx, org, f)
}

func (u *CommissionUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateCommissionInput) (*FinanceCommission, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.VerificationID == uuid.Nil || in.EmployeeID == uuid.Nil || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) || !in.RatePercent.IsPositive() || in.RatePercent.GreaterThan(decimal.NewFromInt(100)) {
		return nil, ErrCommissionInvalid
	}
	if old, err := u.repo.GetByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if old.VerificationID != in.VerificationID || old.EmployeeID != in.EmployeeID || !old.RatePercent.Equal(in.RatePercent) || !stringPointersEqual(old.Note, in.Note) {
			return nil, ErrCommissionDuplicate
		}
		return old, nil
	}
	id := uuid.Must(uuid.NewV7())
	commissionNo, err := u.config.NextNumber(ctx, org, DocumentTypeCommission)
	if err != nil {
		return nil, err
	}
	c := &FinanceCommission{ID: id, OrganizationID: org, CommissionNo: commissionNo, IdempotencyKey: in.IdempotencyKey, VerificationID: in.VerificationID, EmployeeID: in.EmployeeID, Status: CommissionDraft, RatePercent: in.RatePercent, Note: in.Note, Version: 1}
	return u.repo.Create(ctx, org, actor, c, commissionAudit(org, actor, id, "finance.commission.create"))
}

func (u *CommissionUsecase) Confirm(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionConfirmed, "")
}
func (u *CommissionUsecase) MarkPaid(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionPaid, "")
}
func (u *CommissionUsecase) Cancel(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceCommission, error) {
	return u.transition(ctx, org, actor, id, version, CommissionCancelled, strings.TrimSpace(reason))
}
func (u *CommissionUsecase) transition(ctx context.Context, org, actor, id uuid.UUID, version uint64, target CommissionStatus, reason string) (*FinanceCommission, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 || (target == CommissionCancelled && (reason == "" || utf8.RuneCountInString(reason) > 500)) {
		return nil, ErrCommissionInvalid
	}
	return u.repo.Transition(ctx, org, id, actor, version, target, reason, commissionAudit(org, actor, id, "finance.commission."+strings.ToLower(string(target))))
}
func commissionAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission", ResourceID: id.String()}
}
