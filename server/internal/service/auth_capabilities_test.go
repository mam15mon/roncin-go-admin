package service

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestPrincipalToAPIProjectsEffectiveCapabilities(t *testing.T) {
	id := uuid.New()
	p := &biz.Principal{
		Organization:      biz.Organization{ID: id, Kind: biz.OrganizationKindHeadquarters},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: id, Kind: biz.OrganizationKindHeadquarters}},
		RoleGrants: []biz.RoleGrant{
			{RoleCode: "legacy", DataScope: biz.DataScopeSelf, Permissions: map[string]struct{}{access.UserCreate: {}}},
			{RoleCode: "reader", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.UserRead: {}}},
			{RoleCode: "operator", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{access.FinanceBillConfirm: {}}},
		},
	}
	got := principalToAPI(p)
	if len(got.PermissionCapabilities) != 1 || got.PermissionCapabilities[0].Key != access.UserRead || got.PermissionCapabilities[0].DataScope != "organization" {
		t.Fatalf("总部能力投影错误: %v", got.PermissionCapabilities)
	}
	if !slices.Equal(got.Permissions, []string{access.UserRead}) {
		t.Fatalf("权限列表必须与有效能力同源: %v", got.Permissions)
	}
	if len(got.RoleScopes) != 3 {
		t.Fatal("历史角色仍需保留展示")
	}
	p.Organization.Kind = biz.OrganizationKindCompany
	p.OrganizationNodes[0].Kind = biz.OrganizationKindCompany
	got = principalToAPI(p)
	if len(got.PermissionCapabilities) != 2 || !slices.Contains(got.Permissions, access.FinanceBillConfirm) {
		t.Fatalf("公司工作台应恢复合法经营能力: %v", got.PermissionCapabilities)
	}
}
