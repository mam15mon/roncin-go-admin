package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestDingTalkAssignmentItemsBoundary(t *testing.T) {
	org, role := uuid.NewString(), uuid.NewString()
	valid := &v1.UserMembershipCreation{OrganizationId: org, RoleIds: []string{role, role}}
	tests := []struct {
		name  string
		items []*v1.UserMembershipCreation
	}{
		{"空批次", nil}, {"空项", []*v1.UserMembershipCreation{nil}},
		{"非法组织", []*v1.UserMembershipCreation{{OrganizationId: "invalid"}}},
		{"零组织", []*v1.UserMembershipCreation{{OrganizationId: uuid.Nil.String()}}},
		{"重复组织", []*v1.UserMembershipCreation{valid, valid}},
		{"非法角色", []*v1.UserMembershipCreation{{OrganizationId: org, RoleIds: []string{"invalid"}}}},
		{"零角色", []*v1.UserMembershipCreation{{OrganizationId: org, RoleIds: []string{uuid.Nil.String()}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseUserMembershipCreations(tc.items); err != biz.ErrAdminInvalidArgument {
				t.Fatalf("边界校验=%v", err)
			}
		})
	}
	items := []*v1.UserMembershipCreation{valid, {OrganizationId: uuid.NewString()}}
	parsed, err := parseUserMembershipCreations(items)
	if err != nil || len(parsed) != 2 || len(parsed[0].RoleIDs) != 1 || len(parsed[1].RoleIDs) != 0 {
		t.Fatalf("独立组织与去重=%+v %v", parsed, err)
	}
	items = nil
	for i := 0; i < biz.MaxListPageSize; i++ {
		items = append(items, &v1.UserMembershipCreation{OrganizationId: uuid.NewString()})
	}
	if _, err := parseUserMembershipCreations(items); err != nil {
		t.Fatalf("200项被拒绝: %v", err)
	}
	items = append(items, &v1.UserMembershipCreation{OrganizationId: uuid.NewString()})
	if _, err := parseUserMembershipCreations(items); err != biz.ErrAdminInvalidArgument {
		t.Fatalf("201项未拒绝: %v", err)
	}
}

func TestDingTalkAssignmentRequestsRejectInvalidUsersBeforeUsecase(t *testing.T) {
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New(), Kind: biz.OrganizationKindSystem}})
	s := &AdminService{}
	valid := []*v1.UserMembershipCreation{{OrganizationId: uuid.NewString(), RoleIds: []string{uuid.NewString()}}}
	for _, id := range []string{"invalid", uuid.Nil.String()} {
		if _, err := s.AuthorizeDingTalkUser(ctx, &v1.AuthorizeDingTalkUserRequest{Id: id, Items: valid}); err != biz.ErrAdminInvalidArgument {
			t.Fatalf("单人非法ID: %v", err)
		}
		if _, err := s.BatchAuthorizeDingTalkUsers(ctx, &v1.BatchAuthorizeDingTalkUsersRequest{UserIds: []string{id}, Items: valid}); err != biz.ErrAdminInvalidArgument {
			t.Fatalf("批量非法ID: %v", err)
		}
	}
	for _, ids := range [][]string{nil, make([]string, biz.MaxListPageSize+1)} {
		if _, err := s.BatchAuthorizeDingTalkUsers(ctx, &v1.BatchAuthorizeDingTalkUsersRequest{UserIds: ids, Items: valid}); err != biz.ErrAdminInvalidArgument {
			t.Fatalf("批量人数边界: %v", err)
		}
	}
}

// 嵌入仓储接口，只实现本请求实际调用的方法，记录领域收到的完整组织矩阵。
type assignmentServiceRepo struct {
	biz.AdminRepo
	calls int
	items []biz.AdminUserMembershipCreation
}

func (r *assignmentServiceRepo) AuthorizeDingTalkUser(_ context.Context, _ uuid.UUID, user *biz.AdminUser, items []biz.AdminUserMembershipCreation, _ *biz.NotificationIntent, _ *biz.AuditEvent) (*biz.AdminUser, error) {
	r.calls++
	r.items = items
	return user, nil
}
func TestDingTalkAssignmentRequestsAcceptTwoOrganizationsAnd200Users(t *testing.T) {
	source := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: source, Kind: biz.OrganizationKindSystem}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: source, Kind: biz.OrganizationKindSystem}}})
	repo := &assignmentServiceRepo{}
	s := NewAdminService(biz.NewAdminUsecase(repo, nil), nil)
	first, second, role := uuid.New(), uuid.New(), uuid.New()
	items := []*v1.UserMembershipCreation{{OrganizationId: first.String(), RoleIds: []string{role.String(), role.String()}}, {OrganizationId: second.String()}}
	if response, err := s.AuthorizeDingTalkUser(ctx, &v1.AuthorizeDingTalkUserRequest{Id: uuid.NewString(), Items: items}); err != nil || !response.Success || len(repo.items) != 2 || len(repo.items[0].RoleIDs) != 1 || len(repo.items[1].RoleIDs) != 0 {
		t.Fatalf("单人items转换:%+v %v", repo.items, err)
	}
	ids := make([]string, biz.MaxListPageSize)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	repo.calls = 0
	response, err := s.BatchAuthorizeDingTalkUsers(ctx, &v1.BatchAuthorizeDingTalkUsersRequest{UserIds: ids, Items: items})
	if err != nil || !response.Success || len(response.Data) != biz.MaxListPageSize || repo.calls != biz.MaxListPageSize {
		t.Fatalf("200人边界:%+v %d %v", response, repo.calls, err)
	}
	ids = append(ids, ids[0])
	repo.calls = 0
	if _, err := s.BatchAuthorizeDingTalkUsers(ctx, &v1.BatchAuthorizeDingTalkUsersRequest{UserIds: ids, Items: items}); err != biz.ErrAdminInvalidArgument || repo.calls != 0 {
		t.Fatalf("201原始项被去重绕过:%d %v", repo.calls, err)
	}
}
