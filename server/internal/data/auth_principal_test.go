package data

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

func TestResolvePrincipalBuildsRoleGrantOnlyForEnabledRoles(t *testing.T) {
	roleID := uuid.New()
	readOnlyOrganizationID := uuid.New()
	writableOrganizationID := uuid.New()
	role := &ent.Role{
		ID:        roleID,
		Code:      "operator",
		DataScope: roleent.DataScopeOrganizationTree,
		Enabled:   true,
		Edges: ent.RoleEdges{
			Permissions: []*ent.Permission{{Key: "business.order.se.read"}},
			OrganizationAccesses: []*ent.RoleOrganizationAccess{
				{OrganizationID: writableOrganizationID, Writable: true},
				{OrganizationID: readOnlyOrganizationID},
			},
		},
	}

	grant, enabled := roleGrantFromEntRole(role)
	if !enabled {
		t.Fatal("启用角色应进入 ResolvePrincipal 的 RoleGrants")
	}
	if grant.RoleID != roleID || grant.RoleCode != "operator" || grant.DataScope != biz.DataScopeOrganizationTree {
		t.Fatalf("角色来源信息 = %#v", grant)
	}
	if _, ok := grant.Permissions["business.order.se.read"]; !ok {
		t.Fatalf("角色权限未保留: %#v", grant.Permissions)
	}
	wantAccesses := []biz.OrganizationAccess{
		{OrganizationID: readOnlyOrganizationID},
		{OrganizationID: writableOrganizationID, Writable: true},
	}
	slices.SortFunc(wantAccesses, func(left, right biz.OrganizationAccess) int {
		if left.OrganizationID.String() < right.OrganizationID.String() {
			return -1
		}
		if left.OrganizationID.String() > right.OrganizationID.String() {
			return 1
		}
		return 0
	})
	if !slices.EqualFunc(grant.OrganizationAccesses, wantAccesses, func(left, right biz.OrganizationAccess) bool {
		return left.OrganizationID == right.OrganizationID && left.Writable == right.Writable
	}) {
		t.Fatalf("角色组织访问未保留或未排序: %#v", grant.OrganizationAccesses)
	}

	role.Enabled = false
	if _, enabled := roleGrantFromEntRole(role); enabled {
		t.Fatal("停用角色不应进入 ResolvePrincipal 的 RoleGrants")
	}
}

func TestResolvePrincipalOrganizationBaseCurrencyRetainsDisabledAncestorLookup(t *testing.T) {
	parentID := uuid.New()
	childID := uuid.New()
	currency := "CNY"
	parent := &ent.Organization{ID: parentID, Enabled: false, BaseCurrency: &currency}
	child := &ent.Organization{ID: childID, Enabled: true, ParentID: &parentID}

	actual, err := resolvePrincipalOrganizationBaseCurrency(child, map[uuid.UUID]*ent.Organization{
		parentID: parent,
		childID:  child,
	})
	if err != nil {
		t.Fatalf("停用父组织的本币继承不应改变: %v", err)
	}
	if actual != currency {
		t.Fatalf("本币 = %q，期望 %q", actual, currency)
	}
}
