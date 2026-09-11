package service

import (
	"strings"

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

// organizationIDsForRequestedOrganization 将前端可选的单一组织限定在当前动作的权限范围内。
// 未指定时保留该动作全部范围，任何越权 ID 一律失败，不回退到当前工作区。
func organizationIDsForRequestedOrganization(principal *biz.Principal, permission string, writable bool, rawID *string) ([]uuid.UUID, error) {
	organizationIDs, err := organizationIDsForPermission(principal, permission, writable)
	if err != nil || rawID == nil || strings.TrimSpace(*rawID) == "" {
		return organizationIDs, err
	}
	organizationID, parseErr := uuid.Parse(strings.TrimSpace(*rawID))
	if parseErr != nil {
		return nil, biz.ErrFinanceLedgerInvalidArgument
	}
	if !uuidIn(organizationID, organizationIDs) {
		return nil, biz.ErrPermissionDenied
	}
	return []uuid.UUID{organizationID}, nil
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

func uuidIn(target uuid.UUID, values []uuid.UUID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
