package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// 月度提成申请的财务侧接口：列表、详情下钻与整单批准/驳回。Service 只做 DTO
// 转换与主体/权限解析——组织范围一律由 system.finance.commission.read/manage
// 实时解析后显式传入领域层，跨组织查询必须显式带组织过滤；决策人取自当前会话，
// 不接受客户端改写员工、组织或金额。锁序、版本门禁与整批确认事务由领域层承担。

// ListCommissionApplications 财务申请批次列表：按员工/状态/提交月过滤，服务端分页。
func (s *SettlementService) ListCommissionApplications(ctx context.Context, r *v1.ListCommissionApplicationsRequest) (*v1.ListCommissionApplicationsResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrCommissionApplicationInvalid)
	if err != nil {
		return nil, err
	}
	f := biz.CommissionApplicationFinanceFilter{
		Page:             page,
		PageSize:         pageSize,
		Status:           commissionApplicationStatusFromAPI(r.Status),
		ApplicationMonth: financeOptionalString(r.ApplicationMonth),
	}
	if rawEmployeeID := strings.TrimSpace(r.GetEmployeeId()); rawEmployeeID != "" {
		employeeID, parseErr := uuid.Parse(rawEmployeeID)
		if parseErr != nil {
			return nil, biz.ErrCommissionApplicationInvalid
		}
		f.EmployeeID = employeeID
	}
	organizationIDs, scopeErr := organizationIDsForRequestedOrganization(p, access.FinanceCommissionRead, false, r.OrganizationId)
	if scopeErr != nil {
		return nil, scopeErr
	}
	result, err := s.commissionApplicationUsecase.ListForOrganization(ctx, organizationIDs, f)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceCommissionApplication, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, commissionApplicationToAPI(item))
	}
	return okList(ctx, &v1.ListCommissionApplicationsResponse{
		Data: data, Total: int64(result.Total),
		Page: int32(result.Page), PageSize: int32(result.PageSize),
	}), nil
}

// GetCommissionApplication 单张申请详情：申请头审计字段与明细快照下钻。
func (s *SettlementService) GetCommissionApplication(ctx context.Context, r *v1.GetCommissionApplicationRequest) (*v1.GetCommissionApplicationResponse, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, parseErr := uuid.Parse(strings.TrimSpace(r.GetId()))
	if parseErr != nil {
		return nil, biz.ErrCommissionApplicationInvalid
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionRead, false)
	if scopeErr != nil {
		return nil, scopeErr
	}
	detail, err := s.commissionApplicationUsecase.GetForOrganization(ctx, organizationIDs, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetCommissionApplicationResponse{Data: commissionApplicationDetailToAPI(detail)}), nil
}

// ApproveCommissionApplication 整单批准：expected_version 防并发审批；批准只
// 确认申请内提成计算结果，不代表付款；锁序与整批确认由领域层共享事务承担。
func (s *SettlementService) ApproveCommissionApplication(ctx context.Context, r *v1.ApproveCommissionApplicationRequest) (*v1.ApproveCommissionApplicationResponse, error) {
	p, id, err := commissionApplicationPrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.commissionApplicationUsecase.Approve(ctx, organizationIDs, p.UserID, id, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ApproveCommissionApplicationResponse{Data: commissionApplicationToAPI(item)}), nil
}

// RejectCommissionApplication 整单驳回：原因必填；驳回后员工只能在原申请上重提。
func (s *SettlementService) RejectCommissionApplication(ctx context.Context, r *v1.RejectCommissionApplicationRequest) (*v1.RejectCommissionApplicationResponse, error) {
	p, id, err := commissionApplicationPrincipalAndID(ctx, r.GetId())
	if err != nil {
		return nil, err
	}
	organizationIDs, scopeErr := organizationIDsForPermission(p, access.FinanceCommissionManage, true)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := s.commissionApplicationUsecase.Reject(ctx, organizationIDs, p.UserID, id, r.GetExpectedVersion(), r.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RejectCommissionApplicationResponse{Data: commissionApplicationToAPI(item)}), nil
}

// commissionApplicationPrincipalAndID 解析会话主体与申请 ID：ID 非法返回参数错误。
func commissionApplicationPrincipalAndID(ctx context.Context, rawID string) (*biz.Principal, uuid.UUID, error) {
	p, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, uuid.Nil, principalErr
	}
	id, parseErr := uuid.Parse(strings.TrimSpace(rawID))
	if parseErr != nil {
		return nil, uuid.Nil, biz.ErrCommissionApplicationInvalid
	}
	return p, id, nil
}

func commissionApplicationStatusFromAPI(value *v1.FinanceCommissionApplicationStatus) biz.CommissionApplicationStatus {
	if value == nil {
		return ""
	}
	return biz.CommissionApplicationStatus(strings.TrimPrefix(value.String(), "FINANCE_COMMISSION_APPLICATION_STATUS_"))
}

func commissionApplicationStatusToAPI(value biz.CommissionApplicationStatus) v1.FinanceCommissionApplicationStatus {
	switch value {
	case biz.CommissionApplicationPendingReview:
		return v1.FinanceCommissionApplicationStatus_FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW
	case biz.CommissionApplicationRejected:
		return v1.FinanceCommissionApplicationStatus_FINANCE_COMMISSION_APPLICATION_STATUS_REJECTED
	case biz.CommissionApplicationApproved:
		return v1.FinanceCommissionApplicationStatus_FINANCE_COMMISSION_APPLICATION_STATUS_APPROVED
	default:
		return v1.FinanceCommissionApplicationStatus_FINANCE_COMMISSION_APPLICATION_STATUS_UNSPECIFIED
	}
}

func commissionApplicationToAPI(x *biz.FinanceCommissionApplication) *v1.FinanceCommissionApplication {
	if x == nil {
		return nil
	}
	return &v1.FinanceCommissionApplication{
		Id:                       x.ID.String(),
		ApplicationMonth:         x.ApplicationMonth,
		CoverageTo:               x.CoverageTo,
		Status:                   commissionApplicationStatusToAPI(x.Status),
		Version:                  x.Version,
		CommissionCount:          int32(x.CommissionCount),
		BaseCurrency:             x.BaseCurrency,
		TotalCommissionAmount:    x.TotalCommissionAmount.StringFixed(8),
		TotalCnyCommissionAmount: x.TotalCNYCommissionAmount.StringFixed(8),
		EmployeeId:               x.EmployeeID.String(),
		EmployeeName:             x.EmployeeName,
		SubmittedAt:              x.SubmittedAt.UTC().Format(time.RFC3339),
		SubmittedBy:              x.SubmittedBy.String(),
		DecidedAt:                financeTime(x.DecidedAt),
		DecidedBy:                uuidStringPtr(x.DecidedBy),
		DecisionReason:           x.DecisionReason,
		OrganizationId:           x.OrganizationID.String(),
		OrganizationName:         x.OrganizationName,
		CreatedAt:                x.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:                x.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func commissionApplicationDetailToAPI(x *biz.FinanceCommissionApplicationDetail) *v1.FinanceCommissionApplicationDetail {
	if x == nil {
		return nil
	}
	lines := make([]*v1.FinanceCommissionApplicationLine, 0, len(x.Lines))
	for _, line := range x.Lines {
		lines = append(lines, commissionApplicationLineToAPI(line))
	}
	return &v1.FinanceCommissionApplicationDetail{Application: commissionApplicationToAPI(x.Application), Lines: lines}
}

func commissionApplicationLineToAPI(x *biz.FinanceCommissionApplicationLine) *v1.FinanceCommissionApplicationLine {
	if x == nil {
		return nil
	}
	// 来源二选一：核销或对冲快照，恰好一组非空。
	var verificationID, verificationNo, nettingID, nettingNo *string
	if x.VerificationID != nil {
		idValue := x.VerificationID.String()
		verificationID = &idValue
		verificationNo = x.VerificationNo
	}
	if x.NettingID != nil {
		idValue := x.NettingID.String()
		nettingID = &idValue
		nettingNo = x.NettingNo
	}
	return &v1.FinanceCommissionApplicationLine{
		Id:                  x.ID.String(),
		ApplicationId:       x.ApplicationID.String(),
		CommissionId:        x.CommissionID.String(),
		CommissionNo:        x.CommissionNo,
		CommissionDate:      x.CommissionDate,
		VerificationId:      verificationID,
		VerificationNo:      verificationNo,
		NettingId:           nettingID,
		NettingNo:           nettingNo,
		PersonnelRole:       string(x.PersonnelRole),
		RuleId:              uuidStringPtr(x.RuleID),
		RuleVersion:         x.RuleVersion,
		RuleName:            x.RuleName,
		CalculationBasis:    x.CalculationBasis,
		BaseCurrency:        x.BaseCurrency,
		CommissionAmount:    x.CommissionAmount.StringFixed(8),
		CnyCommissionAmount: x.CNYCommissionAmount.StringFixed(8),
		SourceFingerprint:   x.SourceFingerprint,
		CreatedAt:           x.CreatedAt.UTC().Format(time.RFC3339),
	}
}
