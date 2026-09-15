package data

import (
	"reflect"
	"testing"
)

func TestSortRoleCodeNamesKeepsPairsAligned(t *testing.T) {
	codes := []string{"role_c", "role_a", "role_b"}
	names := []string{"丙角色", "甲角色", "乙角色"}

	sortRoleCodeNames(codes, names)

	if !reflect.DeepEqual(codes, []string{"role_a", "role_b", "role_c"}) {
		t.Fatalf("角色码排序 = %v，期望升序", codes)
	}
	if !reflect.DeepEqual(names, []string{"甲角色", "乙角色", "丙角色"}) {
		t.Fatalf("角色名排序 = %v，期望与角色码保持下标对应", names)
	}
}
