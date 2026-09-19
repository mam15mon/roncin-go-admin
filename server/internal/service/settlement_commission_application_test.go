package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// commissionApplicationRepoStub 实现 biz.FinanceCommissionApplicationRepo：
// 只捕获财务侧入参并返回固定申请，不触发数据库。
type commissionApplicationRepoStub struct {
	biz.FinanceCommissionApplicationRepo
	listScope  []uuid.UUID
	listFilter biz.CommissionApplicationFinanceFilter
	approve    struct {
		organizationIDs []uuid.UUID
		decisionMaker   uuid.UUID
		id              uuid.UUID
		expectedVersion uint64
	}
	reject struct {
		organizationIDs []uuid.UUID
		decisionMaker   uuid.UUID
		id              uuid.UUID
		expectedVersion uint64
		reason          string
	}
	result *biz.FinanceCommissionApplication
}

func (s *commissionApplicationRepoStub) ApplicationSummary(_ context.Context, _ biz.WorkbenchScope, _ string) (*biz.WorkbenchApplicationSummary, error) {
	return &biz.WorkbenchApplicationSummary{}, nil
}

func (s *commissionApplicationRepoStub) ListMyCandidates(_ context.Context, _ biz.WorkbenchScope, _ biz.WorkbenchApplicationCandidateFilter) (*biz.PagedList[*biz.WorkbenchApplicationCandidate], error) {
	return &biz.PagedList[*biz.WorkbenchApplicationCandidate]{}, nil
}

func (s *commissionApplicationRepoStub) Submit(_ context.Context, _ biz.WorkbenchScope) (*biz.FinanceCommissionApplication, error) {
	return &biz.FinanceCommissionApplication{}, nil
}

func (s *commissionApplicationRepoStub) ListMyApplications(_ context.Context, _ biz.WorkbenchScope, _ biz.WorkbenchApplicationFilter) (*biz.PagedList[*biz.FinanceCommissionApplication], error) {
	return &biz.PagedList[*biz.FinanceCommissionApplication]{}, nil
}

func (s *commissionApplicationRepoStub) GetMyApplication(_ context.Context, _ biz.WorkbenchScope, _ uuid.UUID) (*biz.FinanceCommissionApplicationDetail, error) {
	return &biz.FinanceCommissionApplicationDetail{}, nil
}

func (s *commissionApplicationRepoStub) ListForOrganization(_ context.Context, organizationIDs []uuid.UUID, filter biz.CommissionApplicationFinanceFilter) (*biz.PagedList[*biz.FinanceCommissionApplication], error) {
	s.listScope = organizationIDs
	s.listFilter = filter
	if s.result == nil {
		return &biz.PagedList[*biz.FinanceCommissionApplication]{}, nil
	}
	return &biz.PagedList[*biz.FinanceCommissionApplication]{Items: []*biz.FinanceCommissionApplication{s.result}, Total: 1, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (s *commissionApplicationRepoStub) GetForOrganization(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (*biz.FinanceCommissionApplicationDetail, error) {
	return &biz.FinanceCommissionApplicationDetail{Application: s.result}, nil
}

func (s *commissionApplicationRepoStub) Approve(_ context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64) (*biz.FinanceCommissionApplication, error) {
	s.approve.organizationIDs = organizationIDs
	s.approve.decisionMaker = decisionMaker
	s.approve.id = id
	s.approve.expectedVersion = expectedVersion
	return s.result, nil
}

func (s *commissionApplicationRepoStub) Reject(_ context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64, reason string) (*biz.FinanceCommissionApplication, error) {
	s.reject.organizationIDs = organizationIDs
	s.reject.decisionMaker = decisionMaker
	s.reject.id = id
	s.reject.expectedVersion = expectedVersion
	s.reject.reason = reason
	return s.result, nil
}

// commissionApplicationPrincipalContext 构造只持有指定权限的组织级主体会话。
func commissionApplicationPrincipalContext(permission string, organizationID uuid.UUID) context.Context {
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:            uuid.New(),
		Organization:      biz.Organization{ID: organizationID},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{permission: {}}}},
	})
}

func newCommissionApplicationServiceForTest(repo *commissionApplicationRepoStub) *SettlementService {
	return NewSettlementService(nil, nil, nil, nil, nil, nil, nil,
		biz.NewFinanceCommissionApplicationUsecase(repo), nil, nil, nil, nil)
}

// TestCommissionApplicationServicePermissionGate 验证服务层权限解析：只持有
// commission.read 的主体不能执行批准/驳回，也不能查看越权组织的申请列表；
// 持有 commission.manage 的主体通过后，决策人与组织范围原样下发领域层。
func TestCommissionApplicationServicePermissionGate(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	foreignOrg := uuid.Must(uuid.NewV7()).String()
	applicationID := uuid.Must(uuid.NewV7())
	stub := &commissionApplicationRepoStub{result: &biz.FinanceCommissionApplication{
		ID: applicationID, Status: biz.CommissionApplicationApproved, Version: 2,
	}}
	service := newCommissionApplicationServiceForTest(stub)

	readCtx := commissionApplicationPrincipalContext(access.FinanceCommissionRead, organizationID)
	if _, err := service.ApproveCommissionApplication(readCtx, &v1.ApproveCommissionApplicationRequest{Id: applicationID.String(), ExpectedVersion: 1}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("read 权限执行批准应被拒绝: %v", err)
	}
	if _, err := service.RejectCommissionApplication(readCtx, &v1.RejectCommissionApplicationRequest{Id: applicationID.String(), ExpectedVersion: 1, Reason: "明细存疑"}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("read 权限执行驳回应被拒绝: %v", err)
	}
	if _, err := service.ListCommissionApplications(readCtx, &v1.ListCommissionApplicationsRequest{
		Page: 1, PageSize: 20, OrganizationId: &foreignOrg,
	}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("read 权限查看越权组织列表应被拒绝: %v", err)
	}

	manageCtx := commissionApplicationPrincipalContext(access.FinanceCommissionManage, organizationID)
	// 列表 RPC 契约要求 commission.read；财务角色同时持有 read + manage。
	listCtx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:            uuid.New(),
		Organization:      biz.Organization{ID: organizationID},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{access.FinanceCommissionRead: {}, access.FinanceCommissionManage: {}}}},
	})
	if _, err := service.ListCommissionApplications(listCtx, &v1.ListCommissionApplicationsRequest{
		Page: 1, PageSize: 20, Status: v1.FinanceCommissionApplicationStatus_FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW.Enum(),
	}); err != nil {
		t.Fatalf("manage 权限列表失败: %v", err)
	}
	if len(stub.listScope) != 1 || stub.listScope[0] != organizationID {
		t.Fatalf("列表组织范围应来自 manage 权限解析: %v", stub.listScope)
	}
	if stub.listFilter.Status != biz.CommissionApplicationPendingReview {
		t.Fatalf("状态过滤应原样下发: %+v", stub.listFilter)
	}

	principal, _ := biz.RequirePrincipal(manageCtx)
	if _, err := service.ApproveCommissionApplication(manageCtx, &v1.ApproveCommissionApplicationRequest{Id: applicationID.String(), ExpectedVersion: 1}); err != nil {
		t.Fatalf("manage 权限批准失败: %v", err)
	}
	if stub.approve.decisionMaker != principal.UserID || stub.approve.expectedVersion != 1 || stub.approve.id != applicationID {
		t.Fatalf("批准决策人应取自会话: %+v", stub.approve)
	}
	if _, err := service.RejectCommissionApplication(manageCtx, &v1.RejectCommissionApplicationRequest{Id: applicationID.String(), ExpectedVersion: 1, Reason: " 明细存疑 "}); err != nil {
		t.Fatalf("manage 权限驳回失败: %v", err)
	}
	if stub.reject.reason != "明细存疑" || stub.reject.decisionMaker != principal.UserID {
		t.Fatalf("驳回原因/决策人应归一化下发: %+v", stub.reject)
	}

	// 无任何财务权限的员工执行读取同样拒绝。
	if _, err := service.ListCommissionApplications(biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: organizationID},
	}), &v1.ListCommissionApplicationsRequest{Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("无权限列表应被拒绝: %v", err)
	}
}
