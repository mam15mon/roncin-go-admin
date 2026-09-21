package biz

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

func TestPartnerPermissionsStayInCurrentCompany(t *testing.T) {
	companyA, companyB := uuid.New(), uuid.New()
	permissions := []string{access.PartnerRead, access.PartnerExport, access.PartnerUpdate, access.PartnerAccountRead, access.PartnerContractRead, access.PartnerAttachmentRead, access.PartnerAuditRead, access.PartnerSettlementRuleRead, access.PartnerShippingPresetRead}
	for _, scope := range []DataScope{DataScopeAll, DataScopeOrganizationTree} {
		for _, admin := range []bool{false, true} {
			for _, permission := range permissions {
				t.Run(string(scope)+"/"+permission+map[bool]string{true: "/管理员", false: "/普通用户"}[admin], func(t *testing.T) {
					p := &Principal{Organization: Organization{ID: companyA, Kind: OrganizationKindCompany}, IsBootstrapAdmin: admin, OrganizationNodes: []OrganizationScopeNode{{ID: companyA, Kind: OrganizationKindCompany}, {ID: companyB, Kind: OrganizationKindCompany, ParentID: &companyA}}, RoleGrants: []RoleGrant{roleGrant("A角色", scope, []string{permission})}}
					result, err := p.ResolvePermissionOrganizationScope(permission)
					if err != nil || !slices.Equal(result.ReadableOrganizationIDs, []uuid.UUID{companyA}) || !slices.Equal(result.WritableOrganizationIDs, []uuid.UUID{companyA}) {
						t.Fatalf("伙伴权限须固定当前公司: %+v, %v", result, err)
					}
					p.Organization.ID = companyB
					p.RoleGrants = []RoleGrant{roleGrant("B只读角色", DataScopeAll, []string{access.PartnerRead})}
					result, err = p.ResolvePermissionOrganizationScope(access.PartnerRead)
					if err != nil || !slices.Equal(result.ReadableOrganizationIDs, []uuid.UUID{companyB}) {
						t.Fatalf("切换后的角色仅能读取公司 B: %+v, %v", result, err)
					}
					if _, err = p.ResolvePermissionOrganizationScope(access.PartnerUpdate); err != ErrPermissionDenied {
						t.Fatalf("不能继承公司 A 编辑权限: %v", err)
					}
				})
			}
		}
	}
}

func TestPartnerReadRejectsHeadquartersAndRelocatedPrincipal(t *testing.T) {
	companyA, companyB := uuid.New(), uuid.New()
	for _, test := range []struct {
		name   string
		kind   OrganizationKind
		anchor uuid.UUID
	}{
		{"总部", OrganizationKindHeadquarters, uuid.Nil},
		{"临时定位其他公司", OrganizationKindCompany, companyB},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &Principal{Organization: Organization{ID: companyA, Kind: test.kind}, WorkspaceOrganizationID: test.anchor, OrganizationNodes: []OrganizationScopeNode{{ID: companyA, Kind: test.kind}, {ID: companyB, Kind: OrganizationKindCompany}}, RoleGrants: []RoleGrant{roleGrant("查看", DataScopeAll, []string{access.PartnerRead})}}
			scope, err := p.ResolvePermissionOrganizationScope(access.PartnerRead)
			if err != nil || len(scope.ReadableOrganizationIDs) != 0 || len(scope.WritableOrganizationIDs) != 0 {
				t.Fatalf("不应获得伙伴范围: %+v %v", scope, err)
			}
		})
	}
}
