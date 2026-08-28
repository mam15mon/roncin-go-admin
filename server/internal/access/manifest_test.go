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

func TestManifestRequiresReferenceExistingKeys(t *testing.T) {
	keys := make(map[string]struct{}, len(manifest))
	for _, permission := range Manifest() {
		keys[permission.Key] = struct{}{}
	}
	for _, permission := range Manifest() {
		for _, required := range permission.Requires {
			if _, ok := keys[required]; !ok {
				t.Fatalf("权限 %s 依赖了不存在的权限码: %s", permission.Key, required)
			}
		}
	}
}

func TestManifestRequiresHaveNoCycle(t *testing.T) {
	requiresOf := make(map[string][]string, len(manifest))
	for _, permission := range Manifest() {
		requiresOf[permission.Key] = permission.Requires
	}
	var visit func(key string, path []string)
	visit = func(key string, path []string) {
		for _, ancestor := range path {
			if ancestor == key {
				t.Fatalf("权限依赖出现环: %s -> %s", strings.Join(path, " -> "), key)
			}
		}
		next := append(path[:len(path):len(path)], key)
		for _, required := range requiresOf[key] {
			visit(required, next)
		}
	}
	for _, permission := range Manifest() {
		visit(permission.Key, nil)
	}
}

func TestManifestNonReadPermissionsRequireRead(t *testing.T) {
	for _, permission := range Manifest() {
		if strings.HasSuffix(permission.Key, ".read") || permission.Key == PlatformAccess {
			continue
		}
		if len(permission.Requires) == 0 {
			t.Fatalf("非读权限 %s 未声明任何依赖", permission.Key)
		}
	}
}

func TestOrderPermissionRequires(t *testing.T) {
	cases := []struct {
		operation OrderOperation
		required  []string
	}{
		{operation: OrderRead, required: nil},
		{operation: OrderCreate, required: []string{OrderPermission(OrderBusinessSE, OrderRead)}},
		{operation: OrderTransition, required: []string{OrderPermission(OrderBusinessSE, OrderRead)}},
		{operation: OrderMilestoneRead, required: []string{OrderPermission(OrderBusinessSE, OrderRead)}},
		{operation: OrderMilestoneSet, required: []string{OrderPermission(OrderBusinessSE, OrderMilestoneRead), OrderPermission(OrderBusinessSE, OrderRead)}},
		{operation: OrderCargoItemCreate, required: []string{OrderPermission(OrderBusinessSE, OrderCargoItemRead), OrderPermission(OrderBusinessSE, OrderRead)}},
		{operation: OrderFeeDelete, required: []string{OrderPermission(OrderBusinessSE, OrderFeeRead), OrderPermission(OrderBusinessSE, OrderRead)}},
	}
	for _, item := range cases {
		if got := orderPermissionRequires(OrderBusinessSE, item.operation); !slicesEqual(got, item.required) {
			t.Fatalf("orderPermissionRequires(SE, %s) = %v, want %v", item.operation, got, item.required)
		}
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func TestResolveDependenciesExpandsTransitiveRequires(t *testing.T) {
	granted := ResolveDependencies([]string{OrderPermission(OrderBusinessSE, OrderCargoItemCreate), PartnerUpdate, RoleUpdate})
	expected := []string{
		OrderPermission(OrderBusinessSE, OrderCargoItemCreate),
		OrderPermission(OrderBusinessSE, OrderCargoItemRead),
		OrderPermission(OrderBusinessSE, OrderRead),
		PartnerUpdate,
		PartnerRead,
		RoleUpdate,
		RoleRead,
		PermissionRead,
		OrganizationRead,
	}
	if !slicesEqual(granted, expected) {
		t.Fatalf("ResolveDependencies() = %v, want %v", granted, expected)
	}
}

func TestResolveDependenciesKeepsUnknownKeysAndIsIdempotent(t *testing.T) {
	source := []string{PartnerCreate, "custom.unknown", PartnerRead, PartnerCreate}
	resolved := ResolveDependencies(source)
	if !slicesEqual(resolved, []string{PartnerCreate, PartnerRead, "custom.unknown"}) {
		t.Fatalf("ResolveDependencies() = %v", resolved)
	}
	if again := ResolveDependencies(resolved); !slicesEqual(again, resolved) {
		t.Fatalf("ResolveDependencies 不幂等: %v", again)
	}
}
