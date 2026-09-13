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
		Organization:      Organization{ID: currentOrganizationID},
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
	principal := &Principal{RoleGrants: []RoleGrant{roleGrant("reader", DataScopeOrganization, []string{permission})}}

	if !principal.HasPermission(permission) {
		t.Fatal("RoleGrant 中的权限应通过授权检查")
	}
	if got := principal.PermissionKeys(); !slices.Equal(got, []string{permission}) {
		t.Fatalf("登录响应权限投影 = %#v，期望 %#v", got, []string{permission})
	}
	if got := principal.RoleScopes(); len(got) != 1 || got[0] != (RoleScope{RoleCode: "reader", DataScope: DataScopeOrganization}) {
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
		{ID: rootID},
		{ID: currentID, ParentID: &rootID},
		{ID: childID, ParentID: &currentID},
		{ID: grandchildID, ParentID: &childID},
		{ID: siblingID, ParentID: &rootID},
	}
	tests := []struct {
		name  string
		scope DataScope
		want  []uuid.UUID
	}{
		{name: "all", scope: DataScopeAll, want: sortedIDs(rootID, currentID, childID, grandchildID, siblingID)},
		{name: "organization_tree", scope: DataScopeOrganizationTree, want: sortedIDs(currentID, childID, grandchildID)},
		{name: "organization", scope: DataScopeOrganization, want: []uuid.UUID{currentID}},
		{name: "self", scope: DataScopeSelf, want: []uuid.UUID{currentID}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal := &Principal{
				Organization:      Organization{ID: currentID},
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
		Organization: Organization{ID: currentID},
		OrganizationNodes: []OrganizationScopeNode{
			{ID: currentID},
			{ID: disabledParentID, ParentID: &currentID, Disabled: true},
			{ID: enabledChildID, ParentID: &disabledParentID},
		},
		RoleGrants: []RoleGrant{roleGrant("tree-reader", DataScopeOrganizationTree, []string{"permission"})},
	}

	scope, err := principal.ResolvePermissionOrganizationScope("permission")
	if err != nil {
		t.Fatalf("解析组织树范围失败: %v", err)
	}
	if want := sortedIDs(currentID, enabledChildID); !slices.Equal(scope.ReadableOrganizationIDs, want) || !slices.Equal(scope.WritableOrganizationIDs, want) {
		t.Fatalf("停用父组织下的启用后代范围 = %#v，期望 %#v", scope, want)
	}
}

func TestResolvePermissionOrganizationScopeFiltersDisabledOrganizations(t *testing.T) {
	currentID := uuid.New()
	disabledID := uuid.New()
	principal := &Principal{
		Organization:      Organization{ID: currentID},
		OrganizationNodes: []OrganizationScopeNode{{ID: currentID}, {ID: disabledID, Disabled: true}},
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
		Organization: Organization{ID: currentID},
		OrganizationNodes: []OrganizationScopeNode{
			{ID: currentID},
			{ID: otherID, ParentID: &otherParentID},
		},
		RoleGrants: []RoleGrant{
			roleGrant("reader", DataScopeSelf, []string{readPermission}),
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
	if !slices.Equal(writeScope.ReadableOrganizationIDs, sortedIDs(currentID, otherID)) || !slices.Equal(writeScope.WritableOrganizationIDs, sortedIDs(currentID, otherID)) {
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
		Organization:      Organization{ID: currentID},
		OrganizationNodes: scopeNodes(currentID, otherID),
		RoleGrants:        []RoleGrant{roleGrant("administrator", DataScopeOrganization, []string{"system.user.manage"})},
	}
	actual, err := principal.ResolvePermissionOrganizationScope("system.user.manage")
	if err != nil {
		t.Fatalf("bootstrap 管理员解析范围失败: %v", err)
	}
	want := sortedIDs(currentID, otherID)
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
		result = append(result, OrganizationScopeNode{ID: id})
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
