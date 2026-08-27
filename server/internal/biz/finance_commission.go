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
	ErrCommissionNotFound             = errors.NotFound("FINANCE_COMMISSION_NOT_FOUND", "提成记录不存在")
	ErrCommissionInvalid              = errors.BadRequest("FINANCE_COMMISSION_INVALID", "提成参数不合法")
	ErrCommissionSource               = errors.Conflict("FINANCE_COMMISSION_SOURCE", "仅有效应收核销可计提，且必须存在可计算的已实现收入")
	ErrCommissionDuplicate            = errors.Conflict("FINANCE_COMMISSION_DUPLICATE", "该核销与员工已存在提成记录")
	ErrCommissionTransition           = errors.Conflict("FINANCE_COMMISSION_TRANSITION", "当前提成状态不允许该操作")
	ErrVerificationHasCommission      = errors.Conflict("FINANCE_VERIFICATION_HAS_COMMISSION", "核销已生成未取消提成，请先取消提成")
	ErrCommissionRuleNotFound         = errors.NotFound("FINANCE_COMMISSION_RULE_NOT_FOUND", "提成规则不存在")
	ErrCommissionRuleInvalid          = errors.BadRequest("FINANCE_COMMISSION_RULE_INVALID", "提成规则字段不合法")
	ErrCommissionRuleConflict         = errors.Conflict("FINANCE_COMMISSION_RULE_CONFLICT", "提成规则名称已存在或版本已变化")
	ErrCommissionEmployeeRole         = errors.Conflict("FINANCE_COMMISSION_EMPLOYEE_ROLE", "所选员工未在核销涉及订单中担任规则指定角色")
	ErrCommissionSourceChanged        = errors.Conflict("FINANCE_COMMISSION_SOURCE_CHANGED", "提成来源数据已变化，请取消当前草稿并重新生成")
	ErrCommissionUnconfirmedFees      = errors.Conflict("FINANCE_COMMISSION_UNCONFIRMED_FEES", "关联订单仍有草稿费用，请先确认或作废后再确认提成")
	ErrCommissionAdjustmentNotFound   = errors.NotFound("FINANCE_COMMISSION_ADJUSTMENT_NOT_FOUND", "提成调整记录不存在")
	ErrCommissionAdjustmentInvalid    = errors.BadRequest("FINANCE_COMMISSION_ADJUSTMENT_INVALID", "提成调整参数不合法")
	ErrCommissionAdjustmentTransition = errors.Conflict("FINANCE_COMMISSION_ADJUSTMENT_TRANSITION", "当前提成调整状态不允许该操作")
	ErrCommissionAdjustmentExceeds    = errors.Conflict("FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS", "冲减后的有效提成金额不能小于零")
)

type CommissionStatus string
type CommissionPersonnelRole string
type CommissionCalculationBasis string
type CommissionAdjustmentDirection string

const (
	CommissionDraft                CommissionStatus              = "DRAFT"
	CommissionConfirmed            CommissionStatus              = "CONFIRMED"
	CommissionPaid                 CommissionStatus              = "PAID"
	CommissionCancelled            CommissionStatus              = "CANCELLED"
	CommissionRoleSales            CommissionPersonnelRole       = "SALES"
	CommissionRoleOperator         CommissionPersonnelRole       = "OPERATOR"
	CommissionBasisRealizedProfit  CommissionCalculationBasis    = "REALIZED_PROFIT"
	CommissionBasisRealizedRevenue CommissionCalculationBasis    = "REALIZED_REVENUE"
	CommissionCalculationVersion                                 = "ORDER_LINE_V1"
	CommissionAdjustmentIncrease   CommissionAdjustmentDirection = "INCREASE"
	CommissionAdjustmentDecrease   CommissionAdjustmentDirection = "DECREASE"
)

// FinanceCommissionAdjustment 以独立单据记录原始提成确认后的增提或冲减。
type FinanceCommissionAdjustment struct {
	ID, OrganizationID, CommissionID, OrderID, EmployeeID             uuid.UUID
	AdjustmentNo, IdempotencyKey, CommissionNo, OrderNo, EmployeeName string
	Direction                                                         CommissionAdjustmentDirection
	Status                                                            CommissionStatus
	BaseCurrency                                                      string
	Amount                                                            decimal.Decimal
	Reason                                                            string
	Note                                                              *string
	Version                                                           uint64
	ConfirmedAt, PaidAt, CancelledAt                                  *time.Time
	CancellationReason                                                *string
	CreatedAt, UpdatedAt                                              time.Time
}

// FinanceCommissionLine 保存逐订单提成计算快照，保证汇总金额可追溯。
type FinanceCommissionLine struct {
	ID, OrganizationID, CommissionID, OrderID, PersonnelAssignmentID, PersonnelOrganizationID, EmployeeID uuid.UUID
	OrderNo, EmployeeName, BaseCurrency                                                                   string
	PersonnelRole                                                                                         CommissionPersonnelRole
	CalculationBasis                                                                                      CommissionCalculationBasis
	RealizedRevenue, AllocatedCost, RealizedProfit                                                        decimal.Decimal
	RatePercent, CommissionAmount                                                                         decimal.Decimal
	PersonnelAssignedAt, CreatedAt, UpdatedAt                                                             time.Time
}

// CommissionCalculation 是预览和创建提成共用的计算结果，不包含持久化状态。
type CommissionCalculation struct {
	VerificationID, EmployeeID, RuleID                   uuid.UUID
	VerificationNo, EmployeeName, RuleName, BaseCurrency string
	PersonnelRole                                        CommissionPersonnelRole
	CalculationBasis                                     CommissionCalculationBasis
	RuleVersion                                          uint64
	CalculationVersion, SourceFingerprint                string
	RealizedRevenue, AllocatedCost, RealizedProfit       decimal.Decimal
	RatePercent, CommissionAmount                        decimal.Decimal
	Lines                                                []*FinanceCommissionLine
}

type FinanceCommission struct {
	ID, OrganizationID, VerificationID, EmployeeID, RuleID     uuid.UUID
	CommissionNo, IdempotencyKey, VerificationNo, EmployeeName string
	RuleName                                                   string
	PersonnelRole                                              CommissionPersonnelRole
	CalculationBasis                                           CommissionCalculationBasis
	RuleVersion                                                uint64
	CalculationVersion, SourceFingerprint                      string
	Status                                                     CommissionStatus
	BaseCurrency                                               string
	RealizedRevenue, AllocatedCost, RealizedProfit             decimal.Decimal
	RatePercent, CommissionAmount                              decimal.Decimal
	Note                                                       *string
	Version                                                    uint64
	ConfirmedAt, PaidAt, CancelledAt                           *time.Time
	CancellationReason                                         *string
	CreatedAt, UpdatedAt                                       time.Time
	Lines                                                      []*FinanceCommissionLine
	Adjustments                                                []*FinanceCommissionAdjustment
	AdjustmentAmount, EffectiveCommissionAmount                decimal.Decimal
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

type CreateCommissionAdjustmentInput struct {
	CommissionID, OrderID uuid.UUID
	Direction             CommissionAdjustmentDirection
	Amount                decimal.Decimal
	Reason                string
	Note                  *string
	IdempotencyKey        string
}

type CommissionRepo interface {
	List(context.Context, uuid.UUID, CommissionFilter) (*CommissionListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*FinanceCommission, error)
	ListEmployees(context.Context, uuid.UUID) ([]*CommissionEmployeeOption, error)
	ListCandidates(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) ([]*CommissionEmployeeOption, error)
	Preview(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) (*CommissionCalculation, error)
	ListRules(context.Context, uuid.UUID, CommissionRuleFilter) (*CommissionRuleListResult, error)
	CreateRule(context.Context, uuid.UUID, *FinanceCommissionRule, *AuditEvent) (*FinanceCommissionRule, error)
	UpdateRule(context.Context, uuid.UUID, UpdateCommissionRuleInput, *AuditEvent) (*FinanceCommissionRule, error)
	GetByKey(context.Context, uuid.UUID, string) (*FinanceCommission, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *FinanceCommission, *AuditEvent) (*FinanceCommission, error)
	Transition(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, CommissionStatus, string, *AuditEvent) (*FinanceCommission, error)
	GetAdjustmentByKey(context.Context, uuid.UUID, string) (*FinanceCommissionAdjustment, error)
	CreateAdjustment(context.Context, uuid.UUID, uuid.UUID, *FinanceCommissionAdjustment, *AuditEvent) (*FinanceCommissionAdjustment, error)
	TransitionAdjustment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, CommissionStatus, string, *AuditEvent) (*FinanceCommissionAdjustment, error)
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
	if org == uuid.Nil || !ValidListPagination(f.Page, f.PageSize) || (f.Status != "" && f.Status != CommissionDraft && f.Status != CommissionConfirmed && f.Status != CommissionPaid && f.Status != CommissionCancelled) {
		return nil, ErrCommissionInvalid
	}
	return u.repo.List(ctx, org, f)
}

func (u *CommissionUsecase) Get(ctx context.Context, org, id uuid.UUID) (*FinanceCommission, error) {
	if org == uuid.Nil || id == uuid.Nil {
		return nil, ErrCommissionInvalid
	}
	return u.repo.Get(ctx, org, id)
}

func (u *CommissionUsecase) Preview(ctx context.Context, org, verificationID, employeeID, ruleID uuid.UUID) (*CommissionCalculation, error) {
	if org == uuid.Nil || verificationID == uuid.Nil || employeeID == uuid.Nil || ruleID == uuid.Nil {
		return nil, ErrCommissionInvalid
	}
	return u.repo.Preview(ctx, org, verificationID, employeeID, ruleID)
}

func (u *CommissionUsecase) ListRules(ctx context.Context, org uuid.UUID, f CommissionRuleFilter) (*CommissionRuleListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if org == uuid.Nil || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 || (f.PersonnelRole != "" && f.PersonnelRole != CommissionRoleSales && f.PersonnelRole != CommissionRoleOperator) {
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

// CalculateCommissionLine 按单个订单计算本次回款对应的成本、毛利和提成。
// 提成必须逐订单归零后汇总，避免盈利订单与亏损订单先相互抵销而改变员工应得金额。
func CalculateCommissionLine(realizedRevenue, totalReceivable, totalPayable, ratePercent decimal.Decimal, basis CommissionCalculationBasis) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if !realizedRevenue.IsPositive() || !totalReceivable.IsPositive() || totalPayable.IsNegative() {
		return decimal.Zero, decimal.Zero, decimal.Zero, ErrCommissionSource
	}
	allocatedCost := realizedRevenue.Mul(totalPayable).Div(totalReceivable).Round(8)
	realizedProfit := realizedRevenue.Sub(allocatedCost).Round(8)
	commissionAmount, err := CalculateCommissionAmount(realizedRevenue, realizedProfit, ratePercent, basis)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, err
	}
	return allocatedCost, realizedProfit, commissionAmount, nil
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
	created, err := u.repo.Create(ctx, org, actor, c, commissionAudit(org, actor, id, "finance.commission.create"))
	if err == nil {
		return created, nil
	}
	// 并发重试可能在预查后命中幂等唯一索引；仅在请求语义一致时重放原结果。
	old, lookupErr := u.repo.GetByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && old.VerificationID == in.VerificationID && old.EmployeeID == in.EmployeeID && old.RuleID == in.RuleID && stringPointersEqual(old.Note, in.Note) {
		return old, nil
	}
	return nil, err
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

func (u *CommissionUsecase) CreateAdjustment(ctx context.Context, org, actor uuid.UUID, in CreateCommissionAdjustmentInput) (*FinanceCommissionAdjustment, error) {
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Note = normalizedOptionalFinanceString(in.Note)
	if org == uuid.Nil || actor == uuid.Nil || in.CommissionID == uuid.Nil || in.OrderID == uuid.Nil ||
		(in.Direction != CommissionAdjustmentIncrease && in.Direction != CommissionAdjustmentDecrease) ||
		!in.Amount.IsPositive() || !totalAmountPattern.MatchString(in.Amount.String()) || in.Reason == "" ||
		utf8.RuneCountInString(in.Reason) > 500 || in.IdempotencyKey == "" || utf8.RuneCountInString(in.IdempotencyKey) > 128 ||
		(in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	if old, err := u.repo.GetAdjustmentByKey(ctx, org, in.IdempotencyKey); err != nil {
		return nil, err
	} else if old != nil {
		if !sameCommissionAdjustmentIntent(old, in) {
			return nil, ErrCommissionAdjustmentInvalid
		}
		return old, nil
	}
	item := &FinanceCommissionAdjustment{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: org, CommissionID: in.CommissionID, OrderID: in.OrderID,
		IdempotencyKey: in.IdempotencyKey, Direction: in.Direction, Status: CommissionDraft,
		Amount: in.Amount.Round(8), Reason: in.Reason, Note: in.Note, Version: 1,
	}
	created, err := u.repo.CreateAdjustment(ctx, org, actor, item, commissionAdjustmentAudit(org, actor, item.ID, "finance.commission_adjustment.create"))
	if err == nil {
		return created, nil
	}
	old, lookupErr := u.repo.GetAdjustmentByKey(ctx, org, in.IdempotencyKey)
	if lookupErr == nil && old != nil && sameCommissionAdjustmentIntent(old, in) {
		return old, nil
	}
	return nil, err
}

func sameCommissionAdjustmentIntent(old *FinanceCommissionAdjustment, in CreateCommissionAdjustmentInput) bool {
	return old.CommissionID == in.CommissionID && old.OrderID == in.OrderID && old.Direction == in.Direction &&
		old.Amount.Equal(in.Amount.Round(8)) && old.Reason == in.Reason && stringPointersEqual(old.Note, in.Note)
}

func (u *CommissionUsecase) ConfirmAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionConfirmed, "")
}

func (u *CommissionUsecase) MarkAdjustmentPaid(ctx context.Context, org, actor, id uuid.UUID, version uint64) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionPaid, "")
}

func (u *CommissionUsecase) CancelAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64, reason string) (*FinanceCommissionAdjustment, error) {
	return u.transitionAdjustment(ctx, org, actor, id, version, CommissionCancelled, strings.TrimSpace(reason))
}

func (u *CommissionUsecase) transitionAdjustment(ctx context.Context, org, actor, id uuid.UUID, version uint64, target CommissionStatus, reason string) (*FinanceCommissionAdjustment, error) {
	if org == uuid.Nil || actor == uuid.Nil || id == uuid.Nil || version == 0 ||
		(target == CommissionCancelled && (reason == "" || utf8.RuneCountInString(reason) > 500)) {
		return nil, ErrCommissionAdjustmentInvalid
	}
	return u.repo.TransitionAdjustment(ctx, org, id, actor, version, target, reason, commissionAdjustmentAudit(org, actor, id, "finance.commission_adjustment."+strings.ToLower(string(target))))
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

func commissionAdjustmentAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_commission_adjustment", ResourceID: id.String()}
}
