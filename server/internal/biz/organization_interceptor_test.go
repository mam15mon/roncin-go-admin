package biz

import (
	"context"

	"github.com/google/uuid"
)

// headquartersPrincipal 构造总部根节点身份且持有指定权限码集合的测试主体。
func headquartersPrincipal(permissions ...string) *Principal {
	orgID := uuid.Must(uuid.NewV7())
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return &Principal{
		Organization:      Organization{ID: orgID, Kind: OrganizationKindHeadquarters},
		OrganizationNodes: []OrganizationScopeNode{{ID: orgID, Kind: OrganizationKindHeadquarters}},
		RoleGrants:        []RoleGrant{{RoleCode: "administrator", DataScope: DataScopeAll, Permissions: keys}},
	}
}

// companyPrincipal 构造公司工作台身份（挂在总部之下）且持有指定权限码集合的测试
// 主体：持有权限码但组织身份非总部，写 A 型/基线行必须被拦截。
func companyPrincipal(permissions ...string) *Principal {
	headquartersID := uuid.Must(uuid.NewV7())
	companyID := uuid.Must(uuid.NewV7())
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return &Principal{
		Organization: Organization{ID: companyID, Kind: OrganizationKindCompany},
		OrganizationNodes: []OrganizationScopeNode{
			{ID: headquartersID, Kind: OrganizationKindHeadquarters},
			{ID: companyID, ParentID: &headquartersID, Kind: OrganizationKindCompany},
		},
		RoleGrants: []RoleGrant{{RoleCode: "administrator", DataScope: DataScopeAll, Permissions: keys}},
	}
}

func principalContext(principal *Principal) context.Context {
	return WithPrincipal(context.Background(), principal)
}
