package service

import (
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// organizationIDsForPermission 只从持有目标权限的角色解析组织范围；
// 角色与范围关联的解析算法由 biz.Principal 统一维护，Service 不得自行拼接。
func organizationIDsForPermission(principal *biz.Principal, permission string, writable bool) ([]uuid.UUID, error) {
	if principal == nil {
		return nil, biz.ErrPermissionDenied
	}
	scope, err := principal.ResolvePermissionOrganizationScope(permission)
	if err != nil {
		return nil, err
	}
	organizationIDs := scope.ReadableOrganizationIDs
	if writable {
		organizationIDs = scope.WritableOrganizationIDs
	}
	if len(organizationIDs) == 0 {
		return nil, biz.ErrPermissionDenied
	}
	return organizationIDs, nil
}

func currentOrganizationAllowedForPermission(principal *biz.Principal, permission string, writable bool) error {
	organizationIDs, err := organizationIDsForPermission(principal, permission, writable)
	if err != nil {
		return err
	}
	for _, organizationID := range organizationIDs {
		if organizationID == principal.Organization.ID {
			return nil
		}
	}
	return biz.ErrPermissionDenied
}
