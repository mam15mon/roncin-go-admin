package biz

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (u *CommissionUsecase) ListRules(ctx context.Context, org uuid.UUID, f CommissionRuleFilter) (*CommissionRuleListResult, error) {
	return u.ListRulesScoped(ctx, []uuid.UUID{org}, f)
}
func (u *CommissionUsecase) ListRulesScoped(ctx context.Context, organizationIDs []uuid.UUID, f CommissionRuleFilter) (*CommissionRuleListResult, error) {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if !validCommissionOrganizationIDs(organizationIDs) || !ValidListPagination(f.Page, f.PageSize) || utf8.RuneCountInString(f.Keyword) > 100 || (f.PersonnelRole != "" && !validCommissionPersonnelRole(f.PersonnelRole)) {
		return nil, ErrCommissionRuleInvalid
	}
	return u.repo.ListRulesScoped(ctx, organizationIDs, f)
}

func (u *CommissionUsecase) GetRuleScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*FinanceCommissionRule, error) {
	if !validCommissionOrganizationIDs(organizationIDs) || id == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	return u.repo.GetRuleScoped(ctx, organizationIDs, id)
}

// normalizeCommissionRuleEmployeeIDs 去重并按 UUID 升序整理员工集合；批量
// 多选上限与列表分页上限一致（200），空集合合法（草稿可无员工）。
func normalizeCommissionRuleEmployeeIDs(ids []uuid.UUID) ([]uuid.UUID, bool) {
	if len(ids) > MaxListPageSize {
		return nil, false
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result, true
}

func (u *CommissionUsecase) CreateRule(ctx context.Context, org, actor uuid.UUID, in CreateCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCommissionRuleInput(in)
	if err != nil || org == uuid.Nil || actor == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	if normalized.Enabled && len(normalized.EmployeeIDs) == 0 {
		return nil, ErrCommissionRuleEnableNoEmployee
	}
	rule := &FinanceCommissionRule{ID: uuid.Must(uuid.NewV7()), OrganizationID: org, Name: normalized.Name, PersonnelRole: normalized.PersonnelRole, CalculationBasis: normalized.CalculationBasis, RatePercent: normalized.RatePercent, EffectiveFrom: normalized.EffectiveFrom, EffectiveTo: normalized.EffectiveTo, Note: normalized.Note, Enabled: normalized.Enabled, LegacyReadOnly: false, Version: 1}
	if len(normalized.EmployeeIDs) > 0 {
		// 初始分配：起始日等于方案起始日，终止日跟随方案终止日。
		if rule.EffectiveFrom == nil {
			return nil, ErrCommissionRuleInvalid
		}
		rule.Assignments = make([]*FinanceCommissionRuleAssignment, 0, len(normalized.EmployeeIDs))
		for _, employeeID := range normalized.EmployeeIDs {
			rule.Assignments = append(rule.Assignments, &FinanceCommissionRuleAssignment{
				ID: uuid.Must(uuid.NewV7()), OrganizationID: org, RuleID: rule.ID,
				EmployeeID: employeeID, EffectiveFrom: *rule.EffectiveFrom, EffectiveTo: rule.EffectiveTo, CreatedBy: actor,
			})
		}
	}
	return u.repo.CreateRule(ctx, org, actor, rule, commissionRuleAudit(org, actor, rule.ID, "finance.commission_rule.create"))
}

func (u *CommissionUsecase) UpdateRule(ctx context.Context, org, actor uuid.UUID, in UpdateCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCommissionRuleInput(in.CreateCommissionRuleInput)
	if err != nil || org == uuid.Nil || actor == uuid.Nil || in.ID == uuid.Nil || in.ExpectedVersion == 0 {
		return nil, ErrCommissionRuleInvalid
	}
	in.CreateCommissionRuleInput = normalized
	// 名单变更只经 Assign/RemoveRuleEmployees 专属路径，更新入口忽略员工集合。
	in.CreateCommissionRuleInput.EmployeeIDs = nil
	return u.repo.UpdateRule(ctx, org, in, commissionRuleAudit(org, actor, in.ID, "finance.commission_rule.update"))
}

func (u *CommissionUsecase) AssignRuleEmployees(ctx context.Context, org, actor uuid.UUID, in CommissionRuleEmployeeChange) (*FinanceCommissionRule, error) {
	change, err := validCommissionRuleEmployeeChange(in)
	if err != nil {
		return nil, err
	}
	audit := commissionRuleEmployeeChangeAudit(org, actor, change, "finance.commission_rule.assign_employees")
	return u.repo.AssignRuleEmployees(ctx, org, actor, change, audit)
}

func (u *CommissionUsecase) RemoveRuleEmployees(ctx context.Context, org, actor uuid.UUID, in CommissionRuleEmployeeChange) (*FinanceCommissionRule, error) {
	change, err := validCommissionRuleEmployeeChange(in)
	if err != nil {
		return nil, err
	}
	audit := commissionRuleEmployeeChangeAudit(org, actor, change, "finance.commission_rule.remove_employees")
	return u.repo.RemoveRuleEmployees(ctx, org, actor, change, audit)
}

func (u *CommissionUsecase) CopyRule(ctx context.Context, org, actor uuid.UUID, in CopyCommissionRuleInput) (*FinanceCommissionRule, error) {
	normalized, err := normalizeCopyCommissionRuleInput(in)
	if err != nil || org == uuid.Nil || actor == uuid.Nil {
		return nil, ErrCommissionRuleInvalid
	}
	plan := &FinanceCommissionRule{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: org, Name: normalized.Name,
		PersonnelRole: normalized.PersonnelRole, CalculationBasis: normalized.CalculationBasis,
		RatePercent: normalized.RatePercent, EffectiveFrom: &normalized.EffectiveFrom,
		EffectiveTo: normalized.EffectiveTo, Note: normalized.Note, Enabled: true, LegacyReadOnly: false, Version: 1,
	}
	plan.Assignments = make([]*FinanceCommissionRuleAssignment, 0, len(normalized.EmployeeIDs))
	for _, employeeID := range normalized.EmployeeIDs {
		plan.Assignments = append(plan.Assignments, &FinanceCommissionRuleAssignment{
			ID: uuid.Must(uuid.NewV7()), OrganizationID: org, RuleID: plan.ID,
			EmployeeID: employeeID, EffectiveFrom: normalized.EffectiveFrom, EffectiveTo: normalized.EffectiveTo, CreatedBy: actor,
		})
	}
	sourceAdjustAudit := commissionRuleAudit(org, actor, normalized.SourceRuleID, "finance.commission_rule.copy_terminate_source")
	sourceAdjustAudit.Details = map[string]string{
		"copy.effective_from": normalized.EffectiveFrom,
		"copy.rule_id":        plan.ID.String(),
	}
	createAudit := commissionRuleAudit(org, actor, plan.ID, "finance.commission_rule.copy")
	createAudit.Details = map[string]string{
		"copy.source_rule_id": normalized.SourceRuleID.String(),
		"employee_ids":        commissionEmployeeIDList(normalized.EmployeeIDs),
	}
	return u.repo.CopyRule(ctx, org, actor, normalized, plan, sourceAdjustAudit, createAudit)
}

// validCommissionRuleEmployeeChange 归一化名单增删输入：员工集合非空且不超过
// 分页上限，变更生效日期必须是合法财务日期。
func validCommissionRuleEmployeeChange(in CommissionRuleEmployeeChange) (CommissionRuleEmployeeChange, error) {
	in.Today = strings.TrimSpace(in.Today)
	if in.Today == "" {
		in.Today = FinanceBusinessDate(time.Now())
	}
	in.ChangeEffectiveDate = strings.TrimSpace(in.ChangeEffectiveDate)
	employees, ok := normalizeCommissionRuleEmployeeIDs(in.EmployeeIDs)
	if !ok {
		return CommissionRuleEmployeeChange{}, ErrCommissionRuleAssignmentInvalid
	}
	in.EmployeeIDs = employees
	if in.RuleID == uuid.Nil || in.ExpectedVersion == 0 || len(in.EmployeeIDs) == 0 {
		return CommissionRuleEmployeeChange{}, ErrCommissionRuleAssignmentInvalid
	}
	if in.ChangeEffectiveDate != "" && !validFinanceDate(in.ChangeEffectiveDate) {
		return CommissionRuleEmployeeChange{}, ErrCommissionRuleAssignmentInvalid
	}
	return in, nil
}

// normalizeCopyCommissionRuleInput 归一化【复制为新方案】输入：新方案必须以
// 当天或未来日期生效、至少一名真实员工，计算参数全部显式提供。
func normalizeCopyCommissionRuleInput(in CopyCommissionRuleInput) (CopyCommissionRuleInput, error) {
	in.Today = strings.TrimSpace(in.Today)
	if in.Today == "" {
		in.Today = FinanceBusinessDate(time.Now())
	}
	in.Name = strings.TrimSpace(in.Name)
	in.EffectiveFrom = strings.TrimSpace(in.EffectiveFrom)
	in.EffectiveTo = normalizedOptionalFinanceString(in.EffectiveTo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	employees, ok := normalizeCommissionRuleEmployeeIDs(in.EmployeeIDs)
	if !ok || len(employees) == 0 {
		return CopyCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	in.EmployeeIDs = employees
	if in.SourceRuleID == uuid.Nil || in.Name == "" || utf8.RuneCountInString(in.Name) > 100 ||
		!validCommissionPersonnelRole(in.PersonnelRole) ||
		(in.CalculationBasis != CommissionBasisRealizedProfit && in.CalculationBasis != CommissionBasisRealizedRevenue) ||
		!in.RatePercent.IsPositive() || in.RatePercent.GreaterThan(decimal.NewFromInt(100)) ||
		!validFinanceDate(in.EffectiveFrom) || in.EffectiveFrom < in.Today ||
		(in.EffectiveTo != nil && (!validFinanceDate(*in.EffectiveTo) || in.EffectiveFrom > *in.EffectiveTo)) ||
		(in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return CopyCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	return in, nil
}

// NewCommissionRuleMemberAssignmentBlocked 构造成员停用阻断错误：列出需先以
// 当天或未来日期终止分配的方案名，管理员先处理方案名单再停用成员。
func NewCommissionRuleMemberAssignmentBlocked(ruleNames []string) error {
	if len(ruleNames) == 0 {
		return ErrCommissionRuleMemberAssignmentBlocked
	}
	return errors.Conflict("FINANCE_COMMISSION_RULE_MEMBER_ASSIGNMENT", fmt.Sprintf("员工仍有当前或未来的提成方案分配（%s），请先以当天或未来的日期终止分配后再停用成员", strings.Join(ruleNames, "、")))
}

func commissionEmployeeIDList(ids []uuid.UUID) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, id.String())
	}
	return strings.Join(parts, ",")
}

func commissionRuleEmployeeChangeAudit(org, actor uuid.UUID, change CommissionRuleEmployeeChange, action string) *AuditEvent {
	event := commissionRuleAudit(org, actor, change.RuleID, action)
	details := map[string]string{
		"employee_ids":     commissionEmployeeIDList(change.EmployeeIDs),
		"expected_version": strconv.FormatUint(change.ExpectedVersion, 10),
		"today":            change.Today,
	}
	if change.ChangeEffectiveDate != "" {
		details["change_effective_date"] = change.ChangeEffectiveDate
	}
	event.Details = details
	return event
}

func normalizeCommissionRuleInput(in CreateCommissionRuleInput) (CreateCommissionRuleInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.EffectiveFrom = normalizedOptionalFinanceString(in.EffectiveFrom)
	in.EffectiveTo = normalizedOptionalFinanceString(in.EffectiveTo)
	in.Note = normalizedOptionalFinanceString(in.Note)
	in.Today = strings.TrimSpace(in.Today)
	if in.Today == "" {
		in.Today = FinanceBusinessDate(time.Now())
	}
	employees, ok := normalizeCommissionRuleEmployeeIDs(in.EmployeeIDs)
	if !ok {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	in.EmployeeIDs = employees
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 100 || !validCommissionPersonnelRole(in.PersonnelRole) || (in.CalculationBasis != CommissionBasisRealizedProfit && in.CalculationBasis != CommissionBasisRealizedRevenue) || !in.RatePercent.IsPositive() || in.RatePercent.GreaterThan(decimal.NewFromInt(100)) || (in.EffectiveFrom != nil && !validFinanceDate(*in.EffectiveFrom)) || (in.EffectiveTo != nil && !validFinanceDate(*in.EffectiveTo)) || (in.EffectiveFrom != nil && in.EffectiveTo != nil && *in.EffectiveFrom > *in.EffectiveTo) || (in.Note != nil && utf8.RuneCountInString(*in.Note) > 500) {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	// 新启用方案必须有当天或未来的生效起始日；未启用草稿可暂不设起始日。
	if in.Enabled && (in.EffectiveFrom == nil || *in.EffectiveFrom < in.Today) {
		return CreateCommissionRuleInput{}, ErrCommissionRuleInvalid
	}
	return in, nil
}

func validCommissionPersonnelRole(role CommissionPersonnelRole) bool {
	return role == CommissionRoleSales || role == CommissionRoleOperator || role == CommissionRoleCustomerService
}
