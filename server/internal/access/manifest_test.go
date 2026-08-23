package access

import (
	"strings"
	"testing"
)

func TestManifestPermissionKeysAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(manifest))
	for _, permission := range Manifest() {
		if permission.Key == "" || permission.Name == "" || permission.Group == "" || permission.Description == "" {
			t.Fatalf("权限定义字段不能为空: %+v", permission)
		}
		if _, exists := seen[permission.Key]; exists {
			t.Fatalf("权限码重复: %s", permission.Key)
		}
		seen[permission.Key] = struct{}{}
	}
}

func TestManifestDoesNotContainLegacyManagePermissions(t *testing.T) {
	legacyKeys := []string{
		"system.organization.manage",
		"system.user.manage",
		"system.role.manage",
		"business.partner.manage",
		"system.master_data.read",
		"system.master_data.manage",
		"business.order.manage",
		"system.task.manage",
	}

	keys := make(map[string]struct{}, len(manifest))
	for _, permission := range Manifest() {
		keys[permission.Key] = struct{}{}
	}
	for _, legacyKey := range legacyKeys {
		if _, exists := keys[legacyKey]; exists {
			t.Fatalf("权限清单仍包含旧权限码: %s", legacyKey)
		}
	}
}

func TestOrderPermissionsIncludeBusinessType(t *testing.T) {
	for _, permission := range Manifest() {
		if !strings.HasPrefix(permission.Key, "business.order.") {
			continue
		}
		if !strings.Contains(permission.Key, ".se.") && !strings.Contains(permission.Key, ".si.") && !strings.Contains(permission.Key, ".ae.") && !strings.Contains(permission.Key, ".ai.") {
			t.Fatalf("订单权限未包含业务类型: %s", permission.Key)
		}
	}
}
