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
	ErrCommissionRuleNotFound    = errors.NotFound("FINANCE_COMMISSION_RULE_NOT_FOUND", "提成规则不存在")
	ErrCommissionRuleInvalid     = errors.BadRequest("FINANCE_COMMISSION_RULE_INVALID", "提成规则字段不合法")
	ErrCommissionRuleConflict    = errors.Conflict("FINANCE_COMMISSION_RULE_CONFLICT", "提成规则名称已存在或版本已变化")
	ErrCommissionEmployeeRole    = errors.Conflict("FINANCE_COMMISSION_EMPLOYEE_ROLE", "所选员工未在核销涉及订单中担任规则指定角色")
)

type CommissionStatus string
type CommissionPersonnelRole string
type CommissionCalculationBasis string

const (
	CommissionDraft                CommissionStatus           = "DRAFT"
	CommissionConfirmed            CommissionStatus           = "CONFIRMED"
	CommissionPaid                 CommissionStatus           = "PAID"
	CommissionCancelled            CommissionStatus           = "CANCELLED"
	CommissionRoleSales            CommissionPersonnelRole    = "SALES"
	CommissionRoleOperator         CommissionPersonnelRole    = "OPERATOR"
	CommissionBasisRealizedProfit  CommissionCalculationBasis = "REALIZED_PROFIT"
	CommissionBasisRealizedRevenue CommissionCalculationBasis = "REALIZED_REVENUE"
)

type FinanceCommission struct {
	ID, OrganizationID, VerificationID, EmployeeID, RuleID     uuid.UUID
	CommissionNo, IdempotencyKey, VerificationNo, EmployeeName string
	RuleName                                                   string
	PersonnelRole                                              CommissionPersonnelRole
	CalculationBasis                                           CommissionCalculationBasis
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
type FinanceCommissionRule struct {
	ID, OrganizationID               uuid.UUID
	Name                             string
	PersonnelRole                    CommissionPersonnelRole
	CalculationBasis                 CommissionCalculationBasis
	RatePercent                      decimal.Decimal
	EffectiveFrom, EffectiveTo, Note *string
	Enabled                          bool
	Version                          uint64
	CreatedAt, UpdatedAt             time.Time
}
type CommissionRuleFilter struct {
	Page, PageSize int
	Keyword        string
	PersonnelRole  CommissionPersonnelRole
	Enabled        *bool
}
type CommissionRuleListResult struct {
	Items []*FinanceCommissionRule
	Total int64
}
type CreateCommissionRuleInput struct {
	Name                             string
	PersonnelRole                    CommissionPersonnelRole
	CalculationBasis                 CommissionCalculationBasis
	RatePercent                      decimal.Decimal
	EffectiveFrom, EffectiveTo, Note *string
	Enabled                          bool
}
type UpdateCommissionRuleInput struct {
	ID uuid.UUID
	CreateCommissionRuleInput
	ExpectedVersion uint64
}
type CreateCommissionInput struct {
	VerificationID, EmployeeID, RuleID uuid.UUID
	Note                               *string
	IdempotencyKey                     string
}

type CommissionRepo interface {
	List(context.Context, uuid.UUID, CommissionFilter) (*CommissionListResult, error)
	ListEmployees(context.Context, uuid.UUID) ([]*CommissionEmployeeOption, error)
	ListCandidates(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) ([]*CommissionEmployeeOption, error)
	ListRules(context.Context, uuid.UUID, CommissionRuleFilter) (*CommissionRuleListResult, error)
	CreateRule(context.Context, uuid.UUID, *FinanceCommissionRule, *AuditEvent) (*FinanceCommissionRule, error)
	UpdateRule(context.Context, uuid.UUID, UpdateCommissionRuleInput, *AuditEvent) (*FinanceCommissionRule, error)
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
func (u *CommissionUsecase) ListCandidates(ctx context.Context, org, verificationID, ruleID uuid.UUID) ([]*CommissionEmployeeOption, error) {
	if org == uuid.Nil || verificationID == uuid.Nil || ruleID == uuid.Nil {
		return nil, ErrCommissionInvalid
	}
	return u.repo.ListCandidates(ctx, org, verificationID, ruleID)
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

func (u *CommissionUsecase) ListRules(ctx context.Context, org uuid.UUID, f CommissionRuleFilter) (*CommissionRuleListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || f.Page < 1 || f.PageSize < 1 || f.PageSize > 200 || utf8.RuneCountInString(f.Keyword) > 100 || (f.PersonnelRole != "" && f.PersonnelRole != CommissionRoleSales && f.PersonnelRole != CommissionRoleOperator) {
		return nil, ErrCommissionRuleInvalid
	}
	return u.repo.ListRules(ctx, org, f)
}
func (u *CommissionUsecase) CreateRule(ctx context.Context, org, actor uuid.UUID, in CreateCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCommissionRuleInput(in)
	if err != nil || org == uuid.Nil || actor == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	rule := &FinanceCommissionRule{ID: uuid.Must(uuid.NewV7()), OrganizationID: org, Name: normalized.Name, PersonnelRole: normalized.PersonnelRole, CalculationBasis: normalized.CalculationBasis, RatePercent: normalized.RatePercent, EffectiveFrom: normalized.EffectiveFrom, EffectiveTo: normalized.EffectiveTo, Note: normalized.Note, Enabled: normalized.Enabled, Version: 1}
	return u.repo.CreateRule(ctx, org, rule, commissionRuleAudit(org, actor, rule.ID, "finance.commission_rule.create"))
}
func (u *CommissionUsecase) UpdateRule(ctx context.Context, org, actor uuid.UUID, in UpdateCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCommissionRuleInput(in.CreateCommissionRuleInput)
	if err != nil || org == uuid.Nil || actor == uuid.Nil || in.ID == uuid.Nil || in.ExpectedVersion == 0 {
		return nil, ErrCommissionRuleInvalid
	}
	in.CreateCommissionRuleInput = normalized
	return u.repo.UpdateRule(ctx, org, in, commissionRuleAudit(org, actor, in.ID, "finance.commission_rule.update"))
}
func normalizeCommissionRuleInput(in CreateCommissionRuleInput) (CreateCommissionRuleInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.EffectiveFrom = normalizedOptionalFinanceString(in.EffectiveFrom)
	in.EffectiveTo = normalizedOptionalFinanceString(in.EffectiveTo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 100 || (in.PersonnelRole != CommissionRoleSales && in.PersonnelRole != CommissionRoleOperator) || (in.CalculationBasis != CommissionBasisRealizedProfit && in.CalculationBasis != CommissionBasisRealizedRevenue) || !in.RatePercent.IsPositive() || in.RatePercent.GreaterThan(decimal.NewFromInt(100)) || (in.EffectiveFrom != nil && !validFinanceDate(*in.EffectiveFrom)) || (in.EffectiveTo != nil && !validFinanceDate(*in.EffectiveTo)) || (in.EffectiveFrom != nil && in.EffectiveTo != nil && *in.EffectiveFrom > *in.EffectiveTo) || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	return in, nil
}

// CalculateCommissionAmount 根据规则口径计算提成，亏损时按零基数计提。
func CalculateCommissionAmount(realizedRevenue, realizedProfit, ratePercent decimal.Decimal, basis CommissionCalculationBasis) (decimal.Decimal, error) {
	base := realizedProfit
	if basis == CommissionBasisRealizedRevenue {
		base = realizedRevenue
	} else if basis != CommissionBasisRealizedProfit {
		return decimal.Zero, ErrCommissionRuleInvalid
	}
	if base.IsNegative() {
		base = decimal.Zero
	}
	return base.Mul(ratePercent).Div(decimal.NewFromInt(100)).Round(8), nil
}

func (u *CommissionUsecase) Create(ctx context.Context, org, actor uuid.UUID, in CreateCommissionInput) (*FinanceCommission, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.VerificationID == uuid.Nil || in.EmployeeID == uuid.Nil || in.RuleID == uuid.Nil || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrCommissionInvalid
	}
	if old, err := u.repo.GetByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if old.VerificationID != in.VerificationID || old.EmployeeID != in.EmployeeID || old.RuleID != in.RuleID || !stringPointersEqual(old.Note, in.Note) {
			return nil, ErrCommissionDuplicate
		}
		return old, nil
	}
	id := uuid.Must(uuid.NewV7())
	commissionNo, err := u.config.NextNumber(ctx, org, DocumentTypeCommission)
	if err != nil {
		return nil, err
	}
	c := &FinanceCommission{ID: id, OrganizationID: org, CommissionNo: commissionNo, IdempotencyKey: in.IdempotencyKey, VerificationID: in.VerificationID, EmployeeID: in.EmployeeID, RuleID: in.RuleID, Status: CommissionDraft, Note: in.Note, Version: 1}
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
func commissionRuleAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission_rule", ResourceID: id.String()}
}
