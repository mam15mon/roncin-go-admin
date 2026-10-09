package data

import (
	"github.com/google/uuid"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestAdminUserStatus(t *testing.T) {
	dingTalkUnionID := "union-id"
	tests := []struct {
		name                string
		account             *ent.User
		currentMembership   *ent.Membership
		hasActiveMembership bool
		want                biz.AdminUserStatus
	}{
		{
			name:                "在职",
			account:             &ent.User{Enabled: true},
			currentMembership:   &ent.Membership{Enabled: true},
			hasActiveMembership: true,
			want:                biz.AdminUserStatusActive,
		},
		{
			name:                "钉钉待授权",
			account:             &ent.User{Enabled: false, DingtalkUnionid: &dingTalkUnionID},
			currentMembership:   &ent.Membership{Enabled: true},
			hasActiveMembership: true,
			want:                biz.AdminUserStatusPendingAuthorization,
		},
		{
			name:                "离职",
			account:             &ent.User{Enabled: false, DingtalkUnionid: &dingTalkUnionID},
			currentMembership:   &ent.Membership{Enabled: false},
			hasActiveMembership: false,
			want:                biz.AdminUserStatusTerminated,
		},
		{
			name:                "已移出当前组织",
			account:             &ent.User{Enabled: true},
			currentMembership:   &ent.Membership{Enabled: false},
			hasActiveMembership: true,
			want:                biz.AdminUserStatusRemovedFromOrganization,
		},
		{
			name:                "普通停用",
			account:             &ent.User{Enabled: false},
			currentMembership:   &ent.Membership{Enabled: true},
			hasActiveMembership: true,
			want:                biz.AdminUserStatusDisabled,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := adminUserStatus(test.account, test.currentMembership, test.hasActiveMembership); got != test.want {
				t.Fatalf("adminUserStatus() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPendingAssignmentHasNoOrganizationAnchorWithHistoricalMembership(t *testing.T) {
	account := &ent.User{PendingAssignment: true, Enabled: false, Edges: ent.UserEdges{Memberships: []*ent.Membership{{Enabled: false}}}}
	row, err := adminUserRow(account, nil)
	if err != nil || row.CurrentMembershipEnabled || len(row.RoleIDs) != 0 || len(row.Organizations) != 0 || row.CurrentOrganizationID != uuid.Nil || row.Status != biz.AdminUserStatusPendingAssignment {
		t.Fatalf("待分配投影带入历史组织: %+v %v", row, err)
	}
}
