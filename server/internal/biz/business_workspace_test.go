package biz

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

func TestBusinessWorkspacePermissionsAndSwitch(t *testing.T) {
	hq, company, other := uuid.New(), uuid.New(), uuid.New()
	grants := make(map[string]struct{})
	for _, permission := range access.Manifest() {
		grants[permission.Key] = struct{}{}
	}
	for _, admin := range []bool{false, true} {
		p := &Principal{UserID: uuid.New(), IsBootstrapAdmin: admin,
			Organization:      Organization{ID: hq, Kind: OrganizationKindSystem},
			OrganizationNodes: []OrganizationScopeNode{{ID: hq, Kind: OrganizationKindSystem}, {ID: company, Kind: OrganizationKindCompany}, {ID: other, Kind: OrganizationKindCompany}},
			RoleGrants:        []RoleGrant{{DataScope: DataScopeAll, Permissions: grants}},
		}
		for _, permission := range []string{access.PartnerRead, access.PartnerExport, access.FinanceCommissionExport, access.FinanceBillConfigure, access.FinanceCommissionConfigure, access.EnterpriseResourceCreate, access.PartnerUpdate, access.PartnerAttachmentRegister, access.FinanceBillConfirm, access.FinanceCommissionManage, access.OrderPermission(access.OrderBusinessSE, access.OrderSplit)} {
			if p.HasPermission(permission) || p.HasPermissionInScope(permission, DataScopeOrganization) || slices.Contains(p.PermissionKeys(), permission) {
				t.Fatalf("总部 admin=%v 暴露经营权限 %s", admin, permission)
			}
			if _, err := p.ResolvePermissionOrganizationScope(permission); err != ErrPermissionDenied {
				t.Fatalf("总部应拒绝经营范围: %v", err)
			}
		}
		for _, permission := range []string{access.MasterDataItemCreate, access.UserCreate} {
			if !p.HasPermission(permission) || !slices.Contains(p.PermissionKeys(), permission) {
				t.Fatalf("总部不得误禁治理或读取 %s", permission)
			}
		}
		p.Organization = Organization{ID: company, Kind: OrganizationKindCompany}
		if !p.HasPermission(access.FinanceBillConfirm) {
			t.Fatal("进入获授权公司后应恢复原角色经营权限")
		}
		scope, err := p.ResolvePermissionOrganizationScope(access.FinanceBillConfirm)
		if err != nil || !slices.Equal(scope.WritableOrganizationIDs, []uuid.UUID{company}) || !slices.Equal(scope.ReadableOrganizationIDs, []uuid.UUID{company}) {
			t.Fatalf("公司经营写范围必须限当前公司，读写均限当前公司: %#v %v", scope, err)
		}
		if p.CanAccessOrganizationForPermission(access.FinanceBillConfirm, other, true) {
			t.Fatal("全量授权不能跨工作台写")
		}
		p.Organization = Organization{ID: hq, Kind: OrganizationKindSystem}
		if p.HasPermission(access.FinanceBillConfirm) {
			t.Fatal("返回总部应再次禁写")
		}
	}
}

func TestBusinessWorkspaceFailsClosed(t *testing.T) {
	id := uuid.New()
	p := &Principal{Organization: Organization{ID: id, Kind: OrganizationKindCompany}}
	if p.CanOperateBusiness() {
		t.Fatal("缺组织投影不可办理")
	}
	p.OrganizationNodes = []OrganizationScopeNode{{ID: id, Kind: OrganizationKindCompany, Disabled: true}}
	if p.CanOperateBusiness() {
		t.Fatal("停用公司不可办理")
	}
	p.OrganizationNodes[0].Disabled = false
	for _, kind := range []OrganizationKind{OrganizationKindSystem, OrganizationKindDepartment, OrganizationKindTeam, ""} {
		p.Organization.Kind = kind
		if p.CanOperateBusiness() {
			t.Fatalf("非公司 %s 不可办理", kind)
		}
	}
	p.Organization.Kind = OrganizationKindCompany
	p.WorkspaceOrganizationID = uuid.New()
	if p.CanOperateBusiness() {
		t.Fatal("跨公司只读资源投影不得变成办理身份")
	}
}

func TestHeadquartersAuthenticatedBusinessActionsRejectBeforeRepository(t *testing.T) {
	p := headquartersPrincipal()
	p.UserID = uuid.New()
	p.IsBootstrapAdmin = true
	id := uuid.New()
	supplement := NewOrderFeeSupplementUsecase(nil, nil, nil)
	calls := []func() error{
		func() error {
			_, err := supplement.Create(context.Background(), p, p.Organization.ID, id, nil, false)
			return err
		},
		func() error {
			_, err := supplement.Approve(context.Background(), p, p.Organization.ID, id, id, 1)
			return err
		},
		func() error {
			_, err := supplement.Reject(context.Background(), p, p.Organization.ID, id, id, 1, nil)
			return err
		},
		func() error {
			_, err := supplement.Withdraw(context.Background(), p, p.Organization.ID, id, id, 1)
			return err
		},
		func() error {
			_, err := supplement.CancelApprovedFee(context.Background(), p, p.Organization.ID, id, id, 1, "撤销")
			return err
		},
		func() error {
			_, err := NewOrderLockUsecase(nil).LockOrder(context.Background(), p, id, 1, "lock", nil)
			return err
		},
		func() error {
			_, err := NewOrderLockUsecase(nil).RequestOrderUnlock(context.Background(), p, id, 1, "unlock", nil, nil)
			return err
		},
	}
	for index, call := range calls {
		if err := call(); err != ErrOperatingCompanyRequired {
			t.Fatalf("动作 %d 应在仓储副作用前拒绝: %v", index, err)
		}
	}
	scope := newApplicationTestScope()
	scope.CanOperateBusiness = false
	applications := NewFinanceCommissionApplicationUsecase(nil)
	if _, err := applications.Submit(context.Background(), scope); err != ErrOperatingCompanyRequired {
		t.Fatalf("总部提交申请: %v", err)
	}
	if _, err := applications.Resubmit(context.Background(), scope, id, 1); err != ErrOperatingCompanyRequired {
		t.Fatalf("总部重提申请: %v", err)
	}
}

func TestReadOnlyResourceDoesNotExposeLockOrOrderActions(t *testing.T) {
	p := companyPrincipal()
	p.UserID = uuid.New()
	current := p.Organization.ID
	permission := access.OrderPermission(access.OrderBusinessSE, access.OrderTransition)
	p.RoleGrants = []RoleGrant{{DataScope: DataScopeAll, Permissions: map[string]struct{}{permission: {}}}}
	order := &Order{OrganizationID: current, BusinessType: OrderBusinessSE, AllowedActions: []OrderAllowedAction{OrderActionTransitionFlow}}
	if len(p.OrderActions(order)) != 1 {
		t.Fatal("本公司授权流转应可见")
	}
	p.WorkspaceOrganizationID = uuid.New()
	if len(p.OrderActions(order)) != 0 {
		t.Fatal("外公司只读动作应为空")
	}
	state := &OrderLockState{CanLock: true, CanRoleDirectUnlock: true, CanAdminEmergencyUnlock: true, CanRequestUnlock: true}
	result, err := NewOrderLockUsecase(&orderLockRepoStub{state: state}).GetOrderLockState(context.Background(), current, uuid.New(), p)
	if err != nil || result.CanLock || result.CanRoleDirectUnlock || result.CanAdminEmergencyUnlock || result.CanRequestUnlock {
		t.Fatalf("跨组织只读不能暴露锁办理入口: %#v %v", result, err)
	}
}

func TestOrderActionsFiltersEditByUpdatePermissionAndWorkspace(t *testing.T) {
	updatePermission := access.OrderPermission(access.OrderBusinessSE, access.OrderUpdate)
	principal := companyPrincipal(updatePermission)
	order := &Order{
		OrganizationID: principal.Organization.ID,
		BusinessType:   OrderBusinessSE,
		AllowedActions: []OrderAllowedAction{OrderActionEdit},
	}
	if actions := principal.OrderActions(order); !slices.Equal(actions, []OrderAllowedAction{OrderActionEdit}) {
		t.Fatalf("本公司具备更新权限时应保留 EDIT，实际: %v", actions)
	}

	principal.RoleGrants = nil
	if actions := principal.OrderActions(order); len(actions) != 0 {
		t.Fatalf("缺少订单更新权限时不应暴露 EDIT，实际: %v", actions)
	}

	principal = companyPrincipal(updatePermission)
	order.OrganizationID = uuid.New()
	if actions := principal.OrderActions(order); len(actions) != 0 {
		t.Fatalf("非当前可写组织范围不应暴露 EDIT，实际: %v", actions)
	}
}

type workspaceTaskRepo struct {
	BackgroundTaskRepo
	kind   BackgroundTaskKind
	writes int
}

func (r *workspaceTaskRepo) Get(_ context.Context, org, id uuid.UUID) (*BackgroundTask, error) {
	return &BackgroundTask{ID: id, OrganizationID: org, Kind: r.kind}, nil
}
func (r *workspaceTaskRepo) Requeue(_ context.Context, org, id uuid.UUID, _ time.Time, _ *AuditEvent) (*BackgroundTask, error) {
	r.writes++
	return &BackgroundTask{ID: id, OrganizationID: org, Kind: r.kind}, nil
}

func TestHeadquartersTaskRequeuePreservesGovernanceOnly(t *testing.T) {
	principal := headquartersPrincipal()
	principal.UserID = uuid.New()
	ctx := WithPrincipal(context.Background(), principal)
	for _, kind := range []BackgroundTaskKind{BackgroundTaskKindDingTalkApproval, BackgroundTaskKindOrderReminder, BackgroundTaskKindIntegration, BackgroundTaskKindMasterDataImport, BackgroundTaskKindDingTalkNotice, BackgroundTaskKindObjectStorageDelete} {
		repo := &workspaceTaskRepo{kind: kind}
		_, err := NewBackgroundTaskUsecase(repo).Requeue(ctx, principal.Organization.ID, principal.UserID, uuid.New())
		business := kind == BackgroundTaskKindDingTalkApproval || kind == BackgroundTaskKindOrderReminder || kind == BackgroundTaskKindIntegration
		if business && (err != ErrOperatingCompanyRequired || repo.writes != 0) {
			t.Fatalf("总部不得重试业务任务 %s: %v", kind, err)
		}
		if !business && (err != nil || repo.writes != 1) {
			t.Fatalf("总部公共任务应保留 %s: %v", kind, err)
		}
	}
}
