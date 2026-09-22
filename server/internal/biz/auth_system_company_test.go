package biz

import (
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"testing"
)

func TestWorkspaceSeparatesSystemManagementAndCompanyBusiness(t *testing.T) {
	systemID, companyA, companyB := uuid.New(), uuid.New(), uuid.New()
	nodes := []OrganizationScopeNode{{ID: systemID, Kind: OrganizationKindSystem}, {ID: companyA, Kind: OrganizationKindCompany}, {ID: companyB, Kind: OrganizationKindCompany}}
	keys := []string{access.PartnerRead, access.FinanceBillRead, access.FinanceBillConfigure, access.OrderPermission(access.OrderBusinessSE, access.OrderRead), access.MasterDataPortUpdate, access.UserUpdate}
	for _, bootstrap := range []bool{false, true} {
		p := &Principal{Organization: Organization{ID: systemID, Kind: OrganizationKindSystem}, OrganizationNodes: nodes, IsBootstrapAdmin: bootstrap, RoleGrants: []RoleGrant{roleGrant("admin", DataScopeAll, keys)}}
		for _, key := range keys[:4] {
			if p.HasPermission(key) {
				t.Fatalf("system泄漏经营权限 %s", key)
			}
		}
		if !p.HasPermission(access.MasterDataPortUpdate) {
			t.Fatal("system应能维护公共港口")
		}
		p.Organization = Organization{ID: companyA, Kind: OrganizationKindCompany}
		if p.HasPermission(access.MasterDataPortUpdate) {
			t.Fatal("公司不得修改公共港口")
		}
		for _, key := range keys[:4] {
			scope, err := p.ResolvePermissionOrganizationScope(key)
			if err != nil || len(scope.ReadableOrganizationIDs) != 1 || scope.ReadableOrganizationIDs[0] != companyA || len(scope.WritableOrganizationIDs) != 1 || scope.WritableOrganizationIDs[0] != companyA {
				t.Fatalf("公司业务范围必须封闭: %s %#v %v", key, scope, err)
			}
		}
	}
}
