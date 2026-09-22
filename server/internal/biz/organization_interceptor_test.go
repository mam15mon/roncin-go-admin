package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestRequireOperatingCompany(t *testing.T) {
	t.Run("公司工作台允许创建经营根数据", func(t *testing.T) {
		if err := RequireOperatingCompany(principalContext(companyPrincipal())); err != nil {
			t.Fatalf("公司工作台不应被拦截: %v", err)
		}
	})

	t.Run("总部工作台禁止持有经营根数据", func(t *testing.T) {
		err := RequireOperatingCompany(principalContext(headquartersPrincipal()))
		if !errors.Is(err, ErrOperatingCompanyRequired) {
			t.Fatalf("总部工作台应返回 ErrOperatingCompanyRequired，实际: %v", err)
		}
	})

	t.Run("缺少主体时保持鉴权错误", func(t *testing.T) {
		if err := RequireOperatingCompany(context.Background()); err == nil || errors.Is(err, ErrOperatingCompanyRequired) {
			t.Fatalf("缺少主体时应返回原始鉴权错误，实际: %v", err)
		}
	})
}

// headquartersPrincipal 构造总部根节点身份且持有指定权限码集合的测试主体。
func headquartersPrincipal(permissions ...string) *Principal {
	orgID := uuid.Must(uuid.NewV7())
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return &Principal{
		Organization:      Organization{ID: orgID, Kind: OrganizationKindSystem},
		OrganizationNodes: []OrganizationScopeNode{{ID: orgID, Kind: OrganizationKindSystem}},
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
			{ID: headquartersID, Kind: OrganizationKindSystem},
			{ID: companyID, ParentID: &headquartersID, Kind: OrganizationKindCompany},
		},
		RoleGrants: []RoleGrant{{RoleCode: "administrator", DataScope: DataScopeAll, Permissions: keys}},
	}
}

func principalContext(principal *Principal) context.Context {
	return WithPrincipal(context.Background(), principal)
}
