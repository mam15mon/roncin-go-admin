package biz

import (
	"context"
)

// RequireGlobalMasterDataWrite 拦截 A 型全局主数据的写路径：要求当前主体沿组织树
// 解析到根且根节点为总部（kind == headquarters），并持有对应全局治理权限码。
// 仅权限码不够——分支上下文的管理员不得写全局行。组织身份只在本写路径判定，
// 不进入任何读路径；判定基于 principal 缓存的组织树投影，不触发额外查询。
func RequireGlobalMasterDataWrite(ctx context.Context, permissionKey string) error {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	return requireHeadquartersPermission(principal, permissionKey)
}

// RequireBaselineWrite 拦截 B 型基线行（organization_id IS NULL）的写路径：要求同
// RequireGlobalMasterDataWrite 的总部身份与权限码。非总部主体的 B 型写入一律落
// 本组织行，不允许触碰基线行。
func RequireBaselineWrite(ctx context.Context, permissionKey string) error {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return err
	}
	return requireHeadquartersPermission(principal, permissionKey)
}

// IsHeadquartersOrganization 判定当前主体的工作台组织沿组织树向上解析到根后是否
// 为总部节点。写路径用于决定 B 型写入落基线行还是本组织行。
func IsHeadquartersOrganization(ctx context.Context) bool {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return false
	}
	return principalIsHeadquarters(principal)
}

// requireHeadquartersPermission 双重校验：总部组织身份 + 对应权限码。
func requireHeadquartersPermission(principal *Principal, permissionKey string) error {
	if !principalIsHeadquarters(principal) {
		return ErrMasterDataHeadquartersRequired
	}
	if !principal.HasPermission(permissionKey) {
		return ErrPermissionDenied
	}
	return nil
}

// principalIsHeadquarters 判定当前主体的工作台组织本身就是总部根节点：工作台组织
// 沿组织树向上无父节点且 kind == headquarters。公司/部门等分支工作台即使挂在总部
// 之下也判非总部（fail-closed）；节点表缺失当前组织时同样判非总部。
func principalIsHeadquarters(principal *Principal) bool {
	nodes := make(map[string]OrganizationScopeNode, len(principal.OrganizationNodes))
	for _, node := range principal.OrganizationNodes {
		nodes[node.ID.String()] = node
	}
	current, ok := nodes[principal.Organization.ID.String()]
	if !ok {
		return false
	}
	visited := make(map[string]struct{}, len(nodes))
	for {
		if current.ParentID == nil {
			// 到达根节点：仅当当前工作台本身就是该根节点且为总部时放行。
			return current.ID == principal.Organization.ID && current.Kind == OrganizationKindHeadquarters
		}
		key := current.ID.String()
		if _, seen := visited[key]; seen {
			// 环路防御：异常数据下直接判非总部，避免死循环。
			return false
		}
		visited[key] = struct{}{}
		parent, ok := nodes[current.ParentID.String()]
		if !ok {
			return false
		}
		current = parent
	}
}
