package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *SettlementService) ListCommissions(ctx context.Context, r *v1.ListCommissionsRequest) (*v1.ListCommissionsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrCommissionInvalid)
	if err != nil {
		return nil, err
	}
	f := biz.CommissionFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword), Status: financeCommissionStatusFromAPI(r.Status), CommissionDateFrom: financeOptionalString(r.CommissionDateFrom), CommissionDateTo: financeOptionalString(r.CommissionDateTo)}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionRead, false, r.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.commissionUsecase.ListScoped(ctx, organizationIDs, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommission, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionToAPI(item))
	}
	return okList(ctx, &v1.ListCommissionsResponse{Data: data, Total: result.Total}), nil
}

// ExportCommissions 同步导出提成：service 只转换筛选与扁平 DTO，不生成 CSV；
// 上限门禁、分批读取与成功审计由用例编排。
func (s *SettlementService) ExportCommissions(ctx context.Context, r *v1.ExportCommissionsRequest) (*v1.ExportCommissionsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	f := biz.CommissionFilter{Keyword: financeOptionalString(r.Keyword), Status: financeCommissionStatusFromAPI(r.Status), CommissionDateFrom: financeOptionalString(r.CommissionDateFrom), CommissionDateTo: financeOptionalString(r.CommissionDateTo)}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionExport, false, r.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	items, err := s.commissionUsecase.ExportScoped(ctx, organizationIDs, p.UserID, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionExportItem, 0, len(items))
	for _, item := range items {
		data = append(data, commissionExportItemToAPI(item))
	}
	return ok(ctx, &v1.ExportCommissionsResponse{Data: data}), nil
}
func (s *SettlementService) GetCommission(ctx context.Context, r *v1.GetCommissionRequest) (*v1.GetCommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionRead, false)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.commissionUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetCommissionResponse{Data: commissionToAPI(item)}), nil
}
func (s *SettlementService) ListCommissionEmployees(ctx context.Context, request *v1.ListCommissionEmployeesRequest) (*v1.ListCommissionEmployeesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrCommissionInvalid)
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionRead, false, request.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.commissionUsecase.ListEmployeesScoped(ctx, organizationIDs, biz.SelectorListOptions{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(request.Keyword),
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionEmployeeOption, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.CommissionEmployeeOption{Id: item.ID.String(), DisplayName: item.DisplayName})
	}
	return okList(ctx, &v1.ListCommissionEmployeesResponse{
		Data:  data,
		Total: int64(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize),
	}), nil
}

// ListCommissionCandidates 按来源单返回「员工 + 人员身份 + 已解析方案」候选：
// 来源二选一，不再接受客户端指定规则；组织按 commission.manage 可写范围校验，
// 来源单与组织归属由领域校验兜底。
func (s *SettlementService) ListCommissionCandidates(ctx context.Context, r *v1.ListCommissionCandidatesRequest) (*v1.ListCommissionCandidatesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	verificationID, nettingID, err := commissionSourceFromAPI(r.GetVerificationId(), r.GetNettingId())
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrCommissionInvalid)
	if err != nil {
		return nil, err
	}
	organizationID, parseErr := uuid.Parse(strings.TrimSpace(r.GetOrganizationId()))
	if parseErr != nil {
		return nil, biz.ErrCommissionInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if !uuidIn(organizationID, organizationIDs) {
		return nil, biz.ErrPermissionDenied
	}
	if sourceErr := s.ensureCommissionSourceOrganization(ctx, organizationID, verificationID, nettingID); sourceErr != nil {
		return nil, sourceErr
	}
	result, err := s.commissionUsecase.ListCandidates(ctx, organizationID, biz.CommissionCandidateFilter{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword), VerificationID: verificationID, NettingID: nettingID,
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CommissionCandidateSummary, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionCandidateSummaryToAPI(item))
	}
	return okList(ctx, &v1.ListCommissionCandidatesResponse{
		Data: data, Total: result.Total,
		Page: int32(result.Page), PageSize: int32(result.PageSize),
	}), nil
}
func (s *SettlementService) ListCommissionRules(ctx context.Context, r *v1.ListCommissionRulesRequest) (*v1.ListCommissionRulesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrCommissionInvalid)
	if err != nil {
		return nil, err
	}
	f := biz.CommissionRuleFilter{Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword), PersonnelRole: biz.CommissionPersonnelRole(strings.ToUpper(financeOptionalString(r.PersonnelRole))), Enabled: r.Enabled}
	if rawEmployeeID := strings.TrimSpace(r.GetEmployeeId()); rawEmployeeID != "" {
		employeeID, parseErr := uuid.Parse(rawEmployeeID)
		if parseErr != nil {
			return nil, biz.ErrCommissionRuleInvalid
		}
		f.EmployeeID = employeeID
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionRead, false, r.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.commissionUsecase.ListRulesScoped(ctx, organizationIDs, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommissionRule, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionRuleToAPI(item))
	}
	return okList(ctx, &v1.ListCommissionRulesResponse{Data: data, Total: result.Total}), nil
}

func (s *SettlementService) CreateCommissionRule(ctx context.Context, r *v1.CreateCommissionRuleRequest) (*v1.CreateCommissionRuleResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	in, err := commissionRuleInputFromAPI(r.GetRule())
	if err != nil {
		return nil, err
	}
	organizationID, parseErr := uuid.Parse(strings.TrimSpace(r.GetOrganizationId()))
	if parseErr != nil {
		return nil, biz.ErrCommissionRuleInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if !uuidIn(organizationID, organizationIDs) {
		return nil, biz.ErrPermissionDenied
	}
	item, err := s.commissionUsecase.CreateRule(ctx, organizationID, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateCommissionRuleResponse{Data: commissionRuleToAPI(item)}), nil
}
func (s *SettlementService) UpdateCommissionRule(ctx context.Context, r *v1.UpdateCommissionRuleRequest) (*v1.UpdateCommissionRuleResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	in, err := commissionRuleInputFromAPI(r.GetRule())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	rule, err := s.commissionUsecase.GetRuleScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.UpdateRule(ctx, rule.OrganizationID, p.UserID, biz.UpdateCommissionRuleInput{ID: id, CreateCommissionRuleInput: in, ExpectedVersion: r.GetExpectedVersion()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateCommissionRuleResponse{Data: commissionRuleToAPI(item)}), nil
}

// AssignCommissionRuleEmployees / RemoveCommissionRuleEmployees 为方案名单的独立
// 增删入口：service 只解析 ID、员工集合与变更生效日，锁序、生效日约束与区间
// 唯一校验由领域层执行。
func (s *SettlementService) AssignCommissionRuleEmployees(ctx context.Context, r *v1.AssignCommissionRuleEmployeesRequest) (*v1.AssignCommissionRuleEmployeesResponse, error) {
	p, organizationID, change, err := s.commissionRuleEmployeeChangeFromAPI(ctx, r.GetId(), r.GetChange())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.AssignRuleEmployees(ctx, organizationID, p.UserID, change)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.AssignCommissionRuleEmployeesResponse{Data: commissionRuleToAPI(item)}), nil
}

func (s *SettlementService) RemoveCommissionRuleEmployees(ctx context.Context, r *v1.RemoveCommissionRuleEmployeesRequest) (*v1.RemoveCommissionRuleEmployeesResponse, error) {
	p, organizationID, change, err := s.commissionRuleEmployeeChangeFromAPI(ctx, r.GetId(), r.GetChange())
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.RemoveRuleEmployees(ctx, organizationID, p.UserID, change)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RemoveCommissionRuleEmployeesResponse{Data: commissionRuleToAPI(item)}), nil
}

// commissionRuleEmployeeChangeFromAPI 校验名单变更请求：先解析主体与方案归属组织
// （按 commission.manage 可写范围），再转换员工集合与变更生效日。
func (s *SettlementService) commissionRuleEmployeeChangeFromAPI(ctx context.Context, rawRuleID string, in *v1.CommissionRuleEmployeeChangeInput) (*biz.Principal, uuid.UUID, biz.CommissionRuleEmployeeChange, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, uuid.Nil, biz.CommissionRuleEmployeeChange{}, principalErr
	}
	ruleID, parseErr := uuid.Parse(strings.TrimSpace(rawRuleID))
	if parseErr != nil {
		return nil, uuid.Nil, biz.CommissionRuleEmployeeChange{}, biz.ErrCommissionRuleAssignmentInvalid
	}
	if in == nil {
		return nil, uuid.Nil, biz.CommissionRuleEmployeeChange{}, biz.ErrCommissionRuleAssignmentInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, uuid.Nil, biz.CommissionRuleEmployeeChange{}, scopeErr
	}
	rule, err := s.commissionUsecase.GetRuleScoped(ctx, organizationIDs, ruleID)
	if err != nil {
		return nil, uuid.Nil, biz.CommissionRuleEmployeeChange{}, err
	}
	change := biz.CommissionRuleEmployeeChange{
		RuleID:              ruleID,
		ExpectedVersion:     in.GetExpectedVersion(),
		ChangeEffectiveDate: financeOptionalString(in.ChangeEffectiveDate),
	}
	for _, rawEmployeeID := range in.GetEmployeeIds() {
		employeeID, employeeErr := uuid.Parse(strings.TrimSpace(rawEmployeeID))
		if employeeErr != nil {
			return nil, uuid.Nil, biz.CommissionRuleEmployeeChange{}, biz.ErrCommissionRuleAssignmentInvalid
		}
		change.EmployeeIDs = append(change.EmployeeIDs, employeeID)
	}
	return p, rule.OrganizationID, change, nil
}

// CopyCommissionRule 实现【复制为新方案】：service 只转换输入，源方案归属组织按
// commission.manage 可写范围解析后由领域层锁内衔接与校验。
func (s *SettlementService) CopyCommissionRule(ctx context.Context, r *v1.CopyCommissionRuleRequest) (*v1.CopyCommissionRuleResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	sourceRuleID, parseErr := uuid.Parse(strings.TrimSpace(r.GetId()))
	if parseErr != nil {
		return nil, biz.ErrCommissionRuleInvalid
	}
	rate, rateErr := decimal.NewFromString(strings.TrimSpace(r.GetRatePercent()))
	if rateErr != nil {
		return nil, biz.ErrCommissionRuleInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	source, err := s.commissionUsecase.GetRuleScoped(ctx, organizationIDs, sourceRuleID)
	if err != nil {
		return nil, err
	}
	in := biz.CopyCommissionRuleInput{
		SourceRuleID:     sourceRuleID,
		Name:             r.GetName(),
		PersonnelRole:    commissionPersonnelRoleFromAPI(r.GetPersonnelRole()),
		CalculationBasis: biz.CommissionCalculationBasis(strings.ToUpper(strings.TrimSpace(r.GetCalculationBasis()))),
		RatePercent:      rate,
		EffectiveFrom:    r.GetEffectiveFrom(),
		EffectiveTo:      r.EffectiveTo,
		Note:             r.Note,
	}
	for _, rawEmployeeID := range r.GetEmployeeIds() {
		employeeID, employeeErr := uuid.Parse(strings.TrimSpace(rawEmployeeID))
		if employeeErr != nil {
			return nil, biz.ErrCommissionRuleInvalid
		}
		in.EmployeeIDs = append(in.EmployeeIDs, employeeID)
	}
	item, err := s.commissionUsecase.CopyRule(ctx, source.OrganizationID, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CopyCommissionRuleResponse{Data: commissionRuleToAPI(item)}), nil
}
func commissionRuleInputFromAPI(r *v1.CommissionRuleInput) (biz.CreateCommissionRuleInput, error) {
	if r == nil {
		return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
	}
	rate, err := decimal.NewFromString(r.GetRatePercent())
	if err != nil {
		return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
	}
	in := biz.CreateCommissionRuleInput{Name: r.GetName(), PersonnelRole: commissionPersonnelRoleFromAPI(r.GetPersonnelRole()), CalculationBasis: biz.CommissionCalculationBasis(strings.ToUpper(r.GetCalculationBasis())), RatePercent: rate, EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo, Enabled: r.GetEnabled(), Note: r.Note}
	for _, rawEmployeeID := range r.GetEmployeeIds() {
		employeeID, employeeErr := uuid.Parse(strings.TrimSpace(rawEmployeeID))
		if employeeErr != nil {
			return biz.CreateCommissionRuleInput{}, biz.ErrCommissionRuleInvalid
		}
		in.EmployeeIDs = append(in.EmployeeIDs, employeeID)
	}
	return in, nil
}
func commissionRuleToAPI(x *biz.FinanceCommissionRule) *v1.FinanceCommissionRule {
	if x == nil {
		return nil
	}
	// 名单投影：只输出抽屉展示与资格追溯所需字段；适用人数按未取消分配段去重。
	assignments := make([]*v1.CommissionRuleAssignmentProjection, 0, len(x.Assignments))
	employees := make(map[uuid.UUID]struct{}, len(x.Assignments))
	for _, item := range x.Assignments {
		employees[item.EmployeeID] = struct{}{}
		assignments = append(assignments, &v1.CommissionRuleAssignmentProjection{
			Id: item.ID.String(), EmployeeId: item.EmployeeID.String(), EmployeeName: item.EmployeeName,
			EffectiveFrom: item.EffectiveFrom, EffectiveTo: item.EffectiveTo,
		})
	}
	return &v1.FinanceCommissionRule{Id: x.ID.String(), Name: x.Name, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), RatePercent: x.RatePercent.StringFixed(4), EffectiveFrom: x.EffectiveFrom, EffectiveTo: x.EffectiveTo, Enabled: x.Enabled, Note: x.Note, Version: x.Version, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339), OrganizationId: x.OrganizationID.String(), OrganizationName: x.OrganizationName, LegacyReadonly: x.LegacyReadOnly, ActiveEmployeeCount: int32(len(employees)), Assignments: assignments}
}

// commissionSourceFromAPI 解析提成来源二选一：核销与对冲恰好提供一个，
// 同时缺失或同时提供返回参数错误。
func commissionSourceFromAPI(rawVerificationID, rawNettingID string) (verificationID, nettingID uuid.UUID, err error) {
	rawVerificationID, rawNettingID = strings.TrimSpace(rawVerificationID), strings.TrimSpace(rawNettingID)
	if rawVerificationID != "" {
		if rawNettingID != "" {
			return uuid.Nil, uuid.Nil, biz.ErrCommissionInvalid
		}
		parsed, parseErr := uuid.Parse(rawVerificationID)
		if parseErr != nil {
			return uuid.Nil, uuid.Nil, biz.ErrCommissionInvalid
		}
		return parsed, uuid.Nil, nil
	}
	parsed, parseErr := uuid.Parse(rawNettingID)
	if parseErr != nil {
		return uuid.Nil, uuid.Nil, biz.ErrCommissionInvalid
	}
	return uuid.Nil, parsed, nil
}

// ensureCommissionSourceOrganization 校验来源单与规则属于同一组织，防止跨组织计提。
func (s *SettlementService) ensureCommissionSourceOrganization(ctx context.Context, organizationID, verificationID, nettingID uuid.UUID) error {
	if verificationID != uuid.Nil {
		verification, err := s.verificationUsecase.GetScoped(ctx, []uuid.UUID{organizationID}, verificationID)
		if err != nil {
			return err
		}
		if verification.OrganizationID != organizationID {
			return biz.ErrPermissionDenied
		}
		return nil
	}
	netting, err := s.nettingUsecase.Get(ctx, []uuid.UUID{organizationID}, nettingID)
	if err != nil {
		return err
	}
	if netting.OrganizationID != organizationID {
		return biz.ErrPermissionDenied
	}
	return nil
}

// ListCommissionNettingCandidates 为生成提成提供已确认对冲单候选（存在有效应收
// 分摊），与核销候选接口分离，保持各自响应形态。
func (s *SettlementService) ListCommissionNettingCandidates(ctx context.Context, r *v1.ListCommissionNettingCandidatesRequest) (*v1.ListCommissionNettingCandidatesResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	raw := strings.TrimSpace(r.GetOrganizationId())
	if raw == "" {
		return nil, biz.ErrCommissionInvalid
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionManage, true, &raw)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if len(organizationIDs) != 1 {
		return nil, biz.ErrCommissionInvalid
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrCommissionInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.commissionUsecase.ListNettingCandidates(ctx, organizationIDs[0], biz.CommissionNettingCandidateFilter{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword),
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceNetting, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, financeNettingToAPI(item))
	}
	return okList(ctx, &v1.ListCommissionNettingCandidatesResponse{Data: data, Total: result.Total}), nil
}

// commissionPersonnelRoleFromAPI 解析并归一化客户端提交的人员身份。
func commissionPersonnelRoleFromAPI(raw string) biz.CommissionPersonnelRole {
	return biz.CommissionPersonnelRole(strings.ToUpper(strings.TrimSpace(raw)))
}

func (s *SettlementService) PreviewCommission(ctx context.Context, r *v1.PreviewCommissionRequest) (*v1.PreviewCommissionResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	verificationID, nettingID, err := commissionSourceFromAPI(r.GetVerificationId(), r.GetNettingId())
	if err != nil {
		return nil, err
	}
	employeeID, err := uuid.Parse(strings.TrimSpace(r.GetEmployeeId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	organizationID, orgErr := uuid.Parse(strings.TrimSpace(r.GetOrganizationId()))
	if orgErr != nil {
		return nil, biz.ErrCommissionInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if !uuidIn(organizationID, organizationIDs) {
		return nil, biz.ErrPermissionDenied
	}
	if sourceErr := s.ensureCommissionSourceOrganization(ctx, organizationID, verificationID, nettingID); sourceErr != nil {
		return nil, sourceErr
	}
	item, err := s.commissionUsecase.Preview(ctx, organizationID, verificationID, nettingID, employeeID, commissionPersonnelRoleFromAPI(r.GetPersonnelRole()))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewCommissionResponse{Data: commissionCalculationToAPI(item)}), nil
}
func (s *SettlementService) CreateCommission(ctx context.Context, r *v1.CreateCommissionRequest) (*v1.CreateCommissionResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	verificationID, nettingID, err := commissionSourceFromAPI(r.GetVerificationId(), r.GetNettingId())
	if err != nil {
		return nil, err
	}
	employeeID, err := uuid.Parse(strings.TrimSpace(r.GetEmployeeId()))
	if err != nil {
		return nil, biz.ErrCommissionInvalid
	}
	organizationID, orgErr := uuid.Parse(strings.TrimSpace(r.GetOrganizationId()))
	if orgErr != nil {
		return nil, biz.ErrCommissionInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	if !uuidIn(organizationID, organizationIDs) {
		return nil, biz.ErrPermissionDenied
	}
	if sourceErr := s.ensureCommissionSourceOrganization(ctx, organizationID, verificationID, nettingID); sourceErr != nil {
		return nil, sourceErr
	}
	item, err := s.commissionUsecase.Create(ctx, organizationID, p.UserID, biz.CreateCommissionInput{VerificationID: verificationID, NettingID: nettingID, EmployeeID: employeeID, PersonnelRole: commissionPersonnelRoleFromAPI(r.GetPersonnelRole()), Note: r.Note, IdempotencyKey: r.GetIdempotencyKey()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateCommissionResponse{Data: commissionToAPI(item)}), nil
}
func (s *SettlementService) ConfirmCommission(ctx context.Context, r *v1.ConfirmCommissionRequest) (*v1.ConfirmCommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	commission, err := s.commissionUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Confirm(ctx, commission.OrganizationID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmCommissionResponse{Data: commissionToAPI(item)}), nil
}
func (s *SettlementService) MarkCommissionPaid(ctx context.Context, r *v1.MarkCommissionPaidRequest) (*v1.MarkCommissionPaidResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	commission, err := s.commissionUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.MarkPaid(ctx, commission.OrganizationID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.MarkCommissionPaidResponse{Data: commissionToAPI(item)}), nil
}
func (s *SettlementService) CancelCommission(ctx context.Context, r *v1.CancelCommissionRequest) (*v1.CancelCommissionResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	commission, err := s.commissionUsecase.GetScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.Cancel(ctx, commission.OrganizationID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CancelCommissionResponse{Data: commissionToAPI(item)}), nil
}
func (s *SettlementService) CreateCommissionAdjustment(ctx context.Context, r *v1.CreateCommissionAdjustmentRequest) (*v1.CreateCommissionAdjustmentResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	commissionID, err := uuid.Parse(strings.TrimSpace(r.GetCommissionId()))
	if err != nil {
		return nil, biz.ErrCommissionAdjustmentInvalid
	}
	orderID, err := uuid.Parse(strings.TrimSpace(r.GetOrderId()))
	if err != nil {
		return nil, biz.ErrCommissionAdjustmentInvalid
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(r.GetAmount()))
	if err != nil {
		return nil, biz.ErrCommissionAdjustmentInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	commission, err := s.commissionUsecase.GetScoped(ctx, organizationIDs, commissionID)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.CreateAdjustment(ctx, commission.OrganizationID, p.UserID, biz.CreateCommissionAdjustmentInput{
		CommissionID: commissionID, OrderID: orderID, Direction: biz.CommissionAdjustmentDirection(strings.ToUpper(r.GetDirection())),
		Amount: amount, Reason: r.GetReason(), Note: r.Note, IdempotencyKey: r.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateCommissionAdjustmentResponse{Data: commissionAdjustmentToAPI(item)}), nil
}

func (s *SettlementService) ConfirmCommissionAdjustment(ctx context.Context, r *v1.ConfirmCommissionAdjustmentRequest) (*v1.ConfirmCommissionAdjustmentResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	adjustment, err := s.commissionUsecase.GetAdjustmentScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.ConfirmAdjustment(ctx, adjustment.OrganizationID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmCommissionAdjustmentResponse{Data: commissionAdjustmentToAPI(item)}), nil
}

func (s *SettlementService) MarkCommissionAdjustmentPaid(ctx context.Context, r *v1.MarkCommissionAdjustmentPaidRequest) (*v1.MarkCommissionAdjustmentPaidResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	adjustment, err := s.commissionUsecase.GetAdjustmentScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.MarkAdjustmentPaid(ctx, adjustment.OrganizationID, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.MarkCommissionAdjustmentPaidResponse{Data: commissionAdjustmentToAPI(item)}), nil
}

func (s *SettlementService) CancelCommissionAdjustment(ctx context.Context, r *v1.CancelCommissionAdjustmentRequest) (*v1.CancelCommissionAdjustmentResponse, error) {
	p, id, err := financePrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	adjustment, err := s.commissionUsecase.GetAdjustmentScoped(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	item, err := s.commissionUsecase.CancelAdjustment(ctx, adjustment.OrganizationID, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CancelCommissionAdjustmentResponse{Data: commissionAdjustmentToAPI(item)}), nil
}

// ListCommissionAdjustments 财务调整列表：服务端分页，按 commission.read 解析
// 组织范围；状态、来源、员工与关键字过滤由领域层校验后下推数据库。
func (s *SettlementService) ListCommissionAdjustments(ctx context.Context, r *v1.ListCommissionAdjustmentsRequest) (*v1.ListCommissionAdjustmentsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrCommissionAdjustmentInvalid)
	if err != nil {
		return nil, err
	}
	f := biz.CommissionAdjustmentFilter{
		Page: page, PageSize: pageSize, Keyword: financeOptionalString(r.Keyword),
		Status:     financeCommissionStatusFromAPI(r.Status),
		SourceType: biz.CommissionAdjustmentSourceType(strings.ToUpper(financeOptionalString(r.SourceType))),
	}
	if rawEmployeeID := strings.TrimSpace(r.GetEmployeeId()); rawEmployeeID != "" {
		employeeID, parseErr := uuid.Parse(rawEmployeeID)
		if parseErr != nil {
			return nil, biz.ErrCommissionAdjustmentInvalid
		}
		f.EmployeeID = employeeID
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionRead, false, r.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.commissionUsecase.ListAdjustmentsScoped(ctx, organizationIDs, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommissionAdjustment, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionAdjustmentToAPI(item))
	}
	return okList(ctx, &v1.ListCommissionAdjustmentsResponse{
		Data: data, Total: result.Total,
		Page: int32(result.Page), PageSize: int32(result.PageSize),
	}), nil
}

// GetMyFeeSupplementAdjustmentSource 员工本人专属冲减来源最小详情：只要求登录，
// 授权（employee_id = 当前用户 + 组织成员关系）在领域与仓储层执行；不要求组织级
// commission.read，他人调整稳定返回不存在。
func (s *SettlementService) GetMyFeeSupplementAdjustmentSource(ctx context.Context, r *v1.GetMyFeeSupplementAdjustmentSourceRequest) (*v1.GetMyFeeSupplementAdjustmentSourceResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, parseErr := uuid.Parse(strings.TrimSpace(r.GetId()))
	if parseErr != nil {
		// 参数非法与他人数据一样按不存在处理，不泄露记录事实。
		return nil, biz.ErrCommissionAdjustmentNotFound
	}
	source, err := s.commissionUsecase.GetMyFeeSupplementAdjustmentSource(ctx, p, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetMyFeeSupplementAdjustmentSourceResponse{Data: &v1.MyFeeSupplementAdjustmentSource{
		AdjustmentId: source.AdjustmentID.String(), AdjustmentNo: source.AdjustmentNo,
		OrderNo: source.OrderNo, CommissionNo: source.CommissionNo,
		Status: financeCommissionStatusToAPI(source.Status), SuggestedAmount: source.SuggestedAmount.StringFixed(8),
		BaseCurrency: source.BaseCurrency, CreatedAt: source.CreatedAt.UTC().Format(time.RFC3339),
		FeeCode: source.FeeCode, FeeName: source.FeeName, FeeCurrency: source.FeeCurrency,
		FeeTotalAmount: source.FeeTotalAmount.StringFixed(8), FeeBaseCurrency: source.FeeBaseCurrency,
		FeeBaseCurrencyAmount: source.FeeBaseCurrencyAmount.StringFixed(8), FeeExpenseDate: source.FeeExpenseDate,
		SupplementReason: source.SupplementReason,
	}}), nil
}

func commissionToAPI(x *biz.FinanceCommission) *v1.FinanceCommission {
	if x == nil {
		return nil
	}
	var ruleID, ruleName, personnelRole, calculationBasis *string
	if x.RuleID != uuid.Nil {
		value := x.RuleID.String()
		ruleID = &value
	}
	if x.RuleName != "" {
		value := x.RuleName
		ruleName = &value
	}
	if x.PersonnelRole != "" {
		value := string(x.PersonnelRole)
		personnelRole = &value
	}
	if x.CalculationBasis != "" {
		value := string(x.CalculationBasis)
		calculationBasis = &value
	}
	// 来源二选一：核销或对冲快照，恰好一组非空。
	var verificationID, verificationNo, nettingID, nettingNo *string
	if x.VerificationID != uuid.Nil {
		idValue, noValue := x.VerificationID.String(), x.VerificationNo
		verificationID, verificationNo = &idValue, &noValue
	}
	if x.NettingID != uuid.Nil {
		idValue, noValue := x.NettingID.String(), x.NettingNo
		nettingID, nettingNo = &idValue, &noValue
	}
	lines := make([]*v1.FinanceCommissionLine, 0, len(x.Lines))
	for _, line := range x.Lines {
		lines = append(lines, commissionLineToAPI(line))
	}
	adjustments := make([]*v1.FinanceCommissionAdjustment, 0, len(x.Adjustments))
	for _, item := range x.Adjustments {
		adjustments = append(adjustments, commissionAdjustmentToAPI(item))
	}
	return &v1.FinanceCommission{Id: x.ID.String(), CommissionNo: x.CommissionNo, VerificationId: verificationID, VerificationNo: verificationNo, NettingId: nettingID, NettingNo: nettingNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, Status: financeCommissionStatusToAPI(x.Status), OrganizationId: x.OrganizationID.String(), OrganizationName: x.OrganizationName, BaseCurrency: x.BaseCurrency, CustomerCount: int32(x.CustomerCount), OrderCount: int32(x.OrderCount), FeeCount: int32(x.FeeCount), RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), ConfirmedBy: uuidStringPtr(x.ConfirmedBy), PaidAt: financeTime(x.PaidAt), PaidBy: uuidStringPtr(x.PaidBy), CancelledAt: financeTime(x.CancelledAt), CancelledBy: uuidStringPtr(x.CancelledBy), CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339), RuleId: ruleID, RuleName: ruleName, PersonnelRole: personnelRole, CalculationBasis: calculationBasis, RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, Lines: lines, Adjustments: adjustments, AdjustmentAmount: x.AdjustmentAmount.StringFixed(8), EffectiveCommissionAmount: x.EffectiveCommissionAmount.StringFixed(8), CommissionDate: x.CommissionDate, CnyExchangeRate: x.CNYExchangeRate.StringFixed(8), CnyExchangeRateSource: x.CNYExchangeRateSource, CnyExchangeRateDate: x.CNYExchangeRateDate, CnyExchangeRateSettingId: uuidStringPtr(x.CNYExchangeRateSettingID), CnyCommissionAmount: x.CNYCommissionAmount.StringFixed(8), CnyAdjustmentAmount: x.CNYAdjustmentAmount.StringFixed(8), CnyEffectiveCommissionAmount: x.CNYEffectiveCommissionAmount.StringFixed(8)}
}

// commissionExportItemToAPI 输出导出扁平 DTO：金额字段与本位币、CNY 双口径及
// 列表动态汇总一致，不包含明细行与调整单。
func commissionExportItemToAPI(x *biz.FinanceCommission) *v1.CommissionExportItem {
	if x == nil {
		return nil
	}
	var verificationNo, nettingNo *string
	if x.VerificationNo != "" {
		verificationNo = &x.VerificationNo
	}
	if x.NettingNo != "" {
		nettingNo = &x.NettingNo
	}
	return &v1.CommissionExportItem{CommissionNo: x.CommissionNo, Status: financeCommissionStatusToAPI(x.Status), VerificationNo: verificationNo, CommissionDate: x.CommissionDate, EmployeeName: x.EmployeeName, PersonnelRole: string(x.PersonnelRole), RuleName: x.RuleName, CalculationBasis: string(x.CalculationBasis), RatePercent: x.RatePercent.StringFixed(4), BaseCurrency: x.BaseCurrency, OrganizationId: x.OrganizationID.String(), OrganizationName: x.OrganizationName, CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), CommissionAmount: x.CommissionAmount.StringFixed(8), CnyCommissionAmount: x.CNYCommissionAmount.StringFixed(8), AdjustmentAmount: x.AdjustmentAmount.StringFixed(8), CnyAdjustmentAmount: x.CNYAdjustmentAmount.StringFixed(8), EffectiveCommissionAmount: x.EffectiveCommissionAmount.StringFixed(8), CnyEffectiveCommissionAmount: x.CNYEffectiveCommissionAmount.StringFixed(8), NettingNo: nettingNo}
}

func commissionAdjustmentToAPI(x *biz.FinanceCommissionAdjustment) *v1.FinanceCommissionAdjustment {
	if x == nil {
		return nil
	}
	return &v1.FinanceCommissionAdjustment{
		Id: x.ID.String(), AdjustmentNo: x.AdjustmentNo, CommissionId: x.CommissionID.String(), CommissionNo: x.CommissionNo,
		OrganizationId: x.OrganizationID.String(), OrganizationName: x.OrganizationName,
		OrderId: x.OrderID.String(), OrderNo: x.OrderNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName,
		Direction: string(x.Direction), Status: financeCommissionStatusToAPI(x.Status), BaseCurrency: x.BaseCurrency, Amount: x.Amount.StringFixed(8),
		Reason: x.Reason, Note: x.Note, Version: x.Version, ConfirmedAt: financeTime(x.ConfirmedAt), ConfirmedBy: uuidStringPtr(x.ConfirmedBy),
		PaidAt: financeTime(x.PaidAt), PaidBy: uuidStringPtr(x.PaidBy),
		CancelledAt: financeTime(x.CancelledAt), CancelledBy: uuidStringPtr(x.CancelledBy), CancellationReason: x.CancellationReason,
		SourceType: string(x.SourceType), SourceVerificationId: uuidStringPtr(x.SourceVerificationID),
		CreatedAt: x.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: x.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func commissionCalculationToAPI(x *biz.CommissionCalculation) *v1.CommissionCalculation {
	if x == nil {
		return nil
	}
	// 来源二选一：核销或对冲快照，恰好一组非空。
	var verificationID, verificationNo, nettingID, nettingNo *string
	if x.VerificationID != uuid.Nil {
		idValue, noValue := x.VerificationID.String(), x.VerificationNo
		verificationID, verificationNo = &idValue, &noValue
	}
	if x.NettingID != uuid.Nil {
		idValue, noValue := x.NettingID.String(), x.NettingNo
		nettingID, nettingNo = &idValue, &noValue
	}
	lines := make([]*v1.FinanceCommissionLine, 0, len(x.Lines))
	for _, line := range x.Lines {
		lines = append(lines, commissionLineToAPI(line))
	}
	result := &v1.CommissionCalculation{VerificationId: verificationID, VerificationNo: verificationNo, NettingId: nettingID, NettingNo: nettingNo, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, RuleId: x.RuleID.String(), RuleName: x.RuleName, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, BaseCurrency: x.BaseCurrency, CustomerCount: int32(x.CustomerCount), OrderCount: int32(x.OrderCount), FeeCount: int32(x.FeeCount), RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), Lines: lines}
	if x.CNY != nil {
		result.CnyExchangeRate = x.CNY.ExchangeRate.StringFixed(8)
		result.CnyExchangeRateSource = x.CNY.ExchangeRateSource
		result.CnyExchangeRateDate = x.CNY.ExchangeRateDate
		result.CnyExchangeRateSettingId = uuidStringPtr(x.CNY.ExchangeRateSettingID)
		result.CnyCommissionAmount = x.CNY.CommissionAmount.StringFixed(8)
	}
	return result
}

func commissionLineToAPI(x *biz.FinanceCommissionLine) *v1.FinanceCommissionLine {
	if x == nil {
		return nil
	}
	fees := make([]*v1.CommissionFeeDetail, 0, len(x.Fees))
	for _, item := range x.Fees {
		fees = append(fees, &v1.CommissionFeeDetail{FeeId: item.FeeID.String(), Direction: item.Direction, FeeCode: item.FeeCode, FeeName: item.FeeName, SettlementPartyId: item.SettlementPartyID.String(), SettlementPartyName: item.SettlementPartyName, Currency: item.Currency, TotalAmount: item.TotalAmount.StringFixed(8), ExchangeRate: item.ExchangeRate.StringFixed(8), BaseCurrency: item.BaseCurrency, BaseCurrencyAmount: item.BaseCurrencyAmount.StringFixed(8), ExpenseDate: item.ExpenseDate, Status: orderFeeStatusToAPI(biz.OrderFeeStatus(item.Status))})
	}
	return &v1.FinanceCommissionLine{Id: x.ID.String(), OrderId: x.OrderID.String(), OrderNo: x.OrderNo, OrderDate: x.OrderDate, CustomerId: x.CustomerID.String(), CustomerCode: x.CustomerCode, CustomerName: x.CustomerName, EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, PersonnelRole: string(x.PersonnelRole), CalculationBasis: string(x.CalculationBasis), BaseCurrency: x.BaseCurrency, RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), CustomerAssignmentId: x.CustomerAssignmentID.String(), CustomerAssignmentOrganizationId: x.CustomerAssignmentOrganizationID.String(), CustomerAssignedAt: x.CustomerAssignedAt.UTC().Format(time.RFC3339), FeeCount: int32(x.FeeCount), Fees: fees, PersonnelOrganizationId: x.CustomerAssignmentOrganizationID.String(), PersonnelAssignedAt: x.CustomerAssignedAt.UTC().Format(time.RFC3339)}
}

func commissionCandidateSummaryToAPI(x *biz.CommissionCalculation) *v1.CommissionCandidateSummary {
	return &v1.CommissionCandidateSummary{EmployeeId: x.EmployeeID.String(), EmployeeName: x.EmployeeName, PersonnelRole: string(x.PersonnelRole), CustomerCount: int32(x.CustomerCount), OrderCount: int32(x.OrderCount), FeeCount: int32(x.FeeCount), BaseCurrency: x.BaseCurrency, RealizedRevenue: x.RealizedRevenue.StringFixed(8), AllocatedCost: x.AllocatedCost.StringFixed(8), RealizedProfit: x.RealizedProfit.StringFixed(8), CommissionBaseAmount: x.CommissionBaseAmount.StringFixed(8), RatePercent: x.RatePercent.StringFixed(4), CommissionAmount: x.CommissionAmount.StringFixed(8), RuleId: x.RuleID.String(), RuleName: x.RuleName, CalculationBasis: string(x.CalculationBasis), RuleVersion: x.RuleVersion}
}
