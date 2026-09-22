package biz

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestRequirePrincipal(t *testing.T) {
	principal := &Principal{Username: "tester"}
	actual, err := RequirePrincipal(WithPrincipal(context.Background(), principal))
	if err != nil {
		t.Fatalf("读取登录主体失败: %v", err)
	}
	if actual != principal {
		t.Fatal("读取到的登录主体与上下文不一致")
	}
}

func TestRequirePrincipalRejectsMissingPrincipal(t *testing.T) {
	actual, err := RequirePrincipal(context.Background())
	if err != ErrSessionRequired {
		t.Fatalf("缺少登录主体时错误 = %v，期望 ErrSessionRequired", err)
	}
	if actual != nil {
		t.Fatal("缺少登录主体时不应返回主体")
	}
}

func TestResolvePermissionOrganizationScopeKeepsRolePermissionAndScopeTogether(t *testing.T) {
	currentOrganizationID := uuid.New()
	otherOrganizationID := uuid.New()
	orderRead := "business.order.se.read"
	financeRead := "finance.bill.read"
	principal := &Principal{
		Organization:      Organization{Kind: OrganizationKindCompany, ID: currentOrganizationID},
		OrganizationNodes: scopeNodes(currentOrganizationID, otherOrganizationID),
		RoleGrants: []RoleGrant{
			roleGrant("order", DataScopeOrganization, []string{orderRead}),
			roleGrant("finance", DataScopeAll, []string{financeRead}),
		},
	}

	scope, err := principal.ResolvePermissionOrganizationScope(orderRead)
	if err != nil {
		t.Fatalf("解析订单读取范围失败: %v", err)
	}
	if !slices.Equal(scope.ReadableOrganizationIDs, []uuid.UUID{currentOrganizationID}) || !slices.Equal(scope.WritableOrganizationIDs, []uuid.UUID{currentOrganizationID}) {
		t.Fatalf("订单权限不应借用财务角色范围，实际 %#v", scope)
	}
}

func TestPrincipalPermissionsAreDerivedOnlyFromRoleGrants(t *testing.T) {
	permission := "business.order.se.read"
	grant := roleGrant("reader", DataScopeOrganization, []string{permission})
	grant.RoleName = "只读角色"
	companyID := uuid.New()
	principal := &Principal{Organization: Organization{ID: companyID, Kind: OrganizationKindCompany}, OrganizationNodes: scopeNodes(companyID), RoleGrants: []RoleGrant{grant}}

	if !principal.HasPermission(permission) {
		t.Fatal("RoleGrant 中的权限应通过授权检查")
	}
	if got := principal.PermissionKeys(); !slices.Equal(got, []string{permission}) {
		t.Fatalf("登录响应权限投影 = %#v，期望 %#v", got, []string{permission})
	}
	if got := principal.RoleScopes(); len(got) != 1 || got[0] != (RoleScope{RoleCode: "reader", DataScope: DataScopeOrganization, RoleName: "只读角色"}) {
		t.Fatalf("登录响应角色范围投影 = %#v", got)
	}
	if principal.HasPermission("finance.bill.read") {
		t.Fatal("RoleGrant 未包含的权限不应放行")
	}
}

func TestResolvePermissionOrganizationScopeDataScopes(t *testing.T) {
	rootID := uuid.New()
	currentID := uuid.New()
	childID := uuid.New()
	grandchildID := uuid.New()
	siblingID := uuid.New()
	nodes := []OrganizationScopeNode{
		{Kind: OrganizationKindCompany, ID: rootID},
		{Kind: OrganizationKindCompany, ID: currentID, ParentID: &rootID},
		{Kind: OrganizationKindCompany, ID: childID, ParentID: &currentID},
		{Kind: OrganizationKindCompany, ID: grandchildID, ParentID: &childID},
		{Kind: OrganizationKindCompany, ID: siblingID, ParentID: &rootID},
	}
	tests := []struct {
		name  string
		scope DataScope
		want  []uuid.UUID
	}{
		{name: "all", scope: DataScopeAll, want: []uuid.UUID{currentID}},
		{name: "organization_tree", scope: DataScopeOrganizationTree, want: []uuid.UUID{currentID}},
		{name: "organization", scope: DataScopeOrganization, want: []uuid.UUID{currentID}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal := &Principal{
				Organization:      Organization{Kind: OrganizationKindCompany, ID: currentID},
				OrganizationNodes: nodes,
				RoleGrants:        []RoleGrant{roleGrant("role", test.scope, []string{"permission"})},
			}
			actual, err := principal.ResolvePermissionOrganizationScope("permission")
			if err != nil {
				t.Fatalf("解析范围失败: %v", err)
			}
			if !slices.Equal(actual.ReadableOrganizationIDs, test.want) || !slices.Equal(actual.WritableOrganizationIDs, test.want) {
				t.Fatalf("范围 = %#v，期望 %#v", actual, test.want)
			}
		})
	}
}

func TestResolvePermissionOrganizationScopeTreeIncludesEnabledDescendantBelowDisabledParent(t *testing.T) {
	currentID := uuid.New()
	disabledParentID := uuid.New()
	enabledChildID := uuid.New()
	principal := &Principal{
		Organization: Organization{Kind: OrganizationKindCompany, ID: currentID},
		OrganizationNodes: []OrganizationScopeNode{
			{Kind: OrganizationKindCompany, ID: currentID},
			{Kind: OrganizationKindCompany, ID: disabledParentID, ParentID: &currentID, Disabled: true},
			{Kind: OrganizationKindCompany, ID: enabledChildID, ParentID: &disabledParentID},
		},
		RoleGrants: []RoleGrant{roleGrant("tree-reader", DataScopeOrganizationTree, []string{"permission"})},
	}

	scope, err := principal.ResolvePermissionOrganizationScope("permission")
	if err != nil {
		t.Fatalf("解析组织树范围失败: %v", err)
	}
	if want := []uuid.UUID{currentID}; !slices.Equal(scope.ReadableOrganizationIDs, want) || !slices.Equal(scope.WritableOrganizationIDs, want) {
		t.Fatalf("停用父组织下的启用后代范围 = %#v，期望 %#v", scope, want)
	}
}

func TestResolvePermissionOrganizationScopeFiltersDisabledOrganizations(t *testing.T) {
	currentID := uuid.New()
	disabledID := uuid.New()
	principal := &Principal{
		Organization:      Organization{Kind: OrganizationKindCompany, ID: currentID},
		OrganizationNodes: []OrganizationScopeNode{{Kind: OrganizationKindCompany, ID: currentID}, {Kind: OrganizationKindCompany, ID: disabledID, Disabled: true}},
		RoleGrants:        []RoleGrant{roleGrant("operator", DataScopeAll, []string{"permission"})},
	}

	actual, err := principal.ResolvePermissionOrganizationScope("permission")
	if err != nil {
		t.Fatalf("解析范围失败: %v", err)
	}
	if want := []uuid.UUID{currentID}; !slices.Equal(actual.ReadableOrganizationIDs, want) || !slices.Equal(actual.WritableOrganizationIDs, want) {
		t.Fatalf("范围 = %#v，期望 %v", actual, want)
	}
	if principal.CanAccessOrganizationForPermission("permission", disabledID, false) {
		t.Fatal("停用组织不应进入任何范围")
	}
}

func TestResolvePermissionOrganizationScopeResolvesPermissionsIndependently(t *testing.T) {
	currentID := uuid.New()
	otherID := uuid.New()
	otherParentID := currentID
	readPermission := "business.order.se.read"
	writePermission := "business.order.se.update"
	principal := &Principal{
		Organization: Organization{Kind: OrganizationKindCompany, ID: currentID},
		OrganizationNodes: []OrganizationScopeNode{
			{Kind: OrganizationKindCompany, ID: currentID},
			{Kind: OrganizationKindCompany, ID: otherID, ParentID: &otherParentID},
		},
		RoleGrants: []RoleGrant{
			roleGrant("reader", DataScopeOrganization, []string{readPermission}),
			roleGrant("writer", DataScopeOrganizationTree, []string{writePermission}),
		},
	}

	readScope, err := principal.ResolvePermissionOrganizationScope(readPermission)
	if err != nil {
		t.Fatalf("解析读取范围失败: %v", err)
	}
	writeScope, err := principal.ResolvePermissionOrganizationScope(writePermission)
	if err != nil {
		t.Fatalf("解析写入范围失败: %v", err)
	}
	if !slices.Equal(readScope.ReadableOrganizationIDs, []uuid.UUID{currentID}) || !slices.Equal(readScope.WritableOrganizationIDs, []uuid.UUID{currentID}) {
		t.Fatalf("读取权限范围错误: %#v", readScope)
	}
	if !slices.Equal(writeScope.ReadableOrganizationIDs, []uuid.UUID{currentID}) || !slices.Equal(writeScope.WritableOrganizationIDs, []uuid.UUID{currentID}) {
		t.Fatalf("写入权限范围错误: %#v", writeScope)
	}
	if _, err := principal.ResolvePermissionOrganizationScope("missing"); err != ErrPermissionDenied {
		t.Fatalf("无匹配角色错误 = %v，期望 ErrPermissionDenied", err)
	}
}

func TestResolvePermissionOrganizationScopeBootstrapAdminUsesAllEnabledOrganizations(t *testing.T) {
	currentID := uuid.New()
	otherID := uuid.New()
	principal := &Principal{
		IsBootstrapAdmin:  true,
		Organization:      Organization{Kind: OrganizationKindCompany, ID: currentID},
		OrganizationNodes: scopeNodes(currentID, otherID),
		RoleGrants:        []RoleGrant{roleGrant("administrator", DataScopeOrganization, []string{"system.user.manage"})},
	}
	actual, err := principal.ResolvePermissionOrganizationScope("system.user.manage")
	if err != nil {
		t.Fatalf("bootstrap 管理员解析范围失败: %v", err)
	}
	want := []uuid.UUID{currentID}
	if !slices.Equal(actual.ReadableOrganizationIDs, want) || !slices.Equal(actual.WritableOrganizationIDs, want) {
		t.Fatalf("bootstrap 管理员应拥有全部启用组织范围，实际 %#v", actual)
	}
}

func roleGrant(code string, scope DataScope, permissions []string) RoleGrant {
	permissionSet := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		permissionSet[permission] = struct{}{}
	}
	return RoleGrant{RoleID: uuid.New(), RoleCode: code, DataScope: scope, Permissions: permissionSet}
}

func scopeNodes(ids ...uuid.UUID) []OrganizationScopeNode {
	result := make([]OrganizationScopeNode, 0, len(ids))
	for _, id := range ids {
		result = append(result, OrganizationScopeNode{Kind: OrganizationKindCompany, ID: id})
	}
	return result
}

func sortedIDs(ids ...uuid.UUID) []uuid.UUID {
	result := append([]uuid.UUID(nil), ids...)
	slices.SortFunc(result, func(left, right uuid.UUID) int {
		if left.String() < right.String() {
			return -1
		}
		if left.String() > right.String() {
			return 1
		}
		return 0
	})
	return result
}

func TestDisabledSelfCannotGrantPermissions(t *testing.T) {
	for _, bootstrap := range []bool{false, true} {
		p := &Principal{IsBootstrapAdmin: bootstrap, Organization: Organization{ID: uuid.New(), Kind: OrganizationKindCompany},
			RoleGrants: []RoleGrant{roleGrant("legacy", DataScopeSelf, []string{"system.user.create"}), roleGrant("unrelated", DataScopeAll, []string{"system.user.read"})}}
		if p.HasPermission("system.user.create") || p.HasPermissionInScope("system.user.create", DataScopeSelf) {
			t.Fatal("停用角色不得授予权限，包括初始化管理员")
		}
		if _, err := p.ResolvePermissionOrganizationScope("system.user.create"); err != ErrPermissionDenied {
			t.Fatalf("停用范围必须拒绝: %v", err)
		}
		if got := p.PermissionKeys(); !slices.Equal(got, []string{"system.user.read"}) {
			t.Fatalf("权限投影包含停用授权: %v", got)
		}
		if len(p.RoleScopes()) != 2 {
			t.Fatal("历史角色仍须展示")
		}
	}
}

func TestPermissionCapabilitiesKeepPermissionScopeSource(t *testing.T) {
	p := &Principal{Organization: Organization{ID: uuid.New(), Kind: OrganizationKindCompany}, RoleGrants: []RoleGrant{
		roleGrant("local", DataScopeOrganization, []string{"system.user.create"}),
		roleGrant("tree", DataScopeOrganizationTree, []string{"system.user.create"}),
		roleGrant("global", DataScopeAll, []string{"system.user.read"}),
		roleGrant("legacy", DataScopeSelf, []string{"system.role.create"}),
	}}
	want := []PermissionCapability{{Key: "system.user.create", DataScope: DataScopeOrganizationTree}, {Key: "system.user.read", DataScope: DataScopeAll}}
	if got := p.PermissionCapabilities(); !slices.Equal(got, want) {
		t.Fatalf("权限范围投影 = %v，期望 %v", got, want)
	}
	if p.HasPermissionInScope("system.user.create", DataScopeAll) {
		t.Fatal("不能借用其他权限的全局范围")
	}
	p.IsBootstrapAdmin = true
	if p.PermissionCapabilities()[0].DataScope != DataScopeOrganizationTree {
		t.Fatal("初始化管理员投影必须匹配接口范围判定")
	}
}
