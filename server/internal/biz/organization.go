package biz

import (
	"context"
	"github.com/google/uuid"

	"github.com/go-kratos/kratos/v3/errors"
)

var ErrOperatingCompanyRequired = errors.Forbidden(
	"OPERATING_COMPANY_REQUIRED",
	"系统管理工作台不承载经营数据，请切换到具体公司后再维护经营数据",
)

// RequireOperatingCompany 要求当前工作台为启用公司；所有经营办理均适用。
func RequireOperatingCompany(ctx context.Context) error {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	if !principal.CanOperateBusiness() {
		return ErrOperatingCompanyRequired
	}
	return nil
}

// RequireGlobalMasterDataWrite 拦截 A 型全局主数据的写路径：要求当前主体沿组织树
// 解析到根且根节点为系统管理（kind == system），并持有对应全局治理权限码。
// 仅权限码不够——分支上下文的管理员不得写全局行。组织身份只在本写路径判定，
// 不进入任何读路径；判定基于 principal 缓存的组织树投影，不触发额外查询。
func RequireGlobalMasterDataWrite(ctx context.Context, permissionKey string) error {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	return requireSystemPermission(principal, permissionKey)
}

// RequireBaselineWrite 拦截 B 型基线行（organization_id IS NULL）的写路径：要求同
// RequireGlobalMasterDataWrite 的系统管理身份与权限码。非系统管理主体的 B 型写入一律落
// 本组织行，不允许触碰基线行。
func RequireBaselineWrite(ctx context.Context, permissionKey string) error {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	return requireSystemPermission(principal, permissionKey)
}

// IsSystemWorkspace 判定当前主体的工作台组织沿组织树向上解析到根后是否
// 为系统管理节点。写路径用于决定 B 型写入落基线行还是本组织行。
func IsSystemWorkspace(ctx context.Context) bool {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return false
	}
	return principalIsSystemWorkspace(principal)
}

// requireSystemPermission 双重校验：系统管理组织身份 + 对应权限码。
func requireSystemPermission(principal *Principal, permissionKey string) error {
	if !principalIsSystemWorkspace(principal) {
		return ErrMasterDataSystemRequired
	}
	if !principal.HasPermission(permissionKey) {
		return ErrPermissionDenied
	}
	return nil
}

// principalIsSystemWorkspace 只承认当前启用的系统根工作台，临时定位组织不能改变身份。
func principalIsSystemWorkspace(principal *Principal) bool {
	if principal == nil || principal.Organization.Kind != OrganizationKindSystem {
		return false
	}
	if principal.WorkspaceOrganizationID != uuid.Nil && principal.WorkspaceOrganizationID != principal.Organization.ID {
		return false
	}
	for _, node := range principal.OrganizationNodes {
		if node.ID == principal.Organization.ID {
			return node.Kind == OrganizationKindSystem && node.ParentID == nil && !node.Disabled
		}
	}
	return false
}
