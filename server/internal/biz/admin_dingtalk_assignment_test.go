package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestAdminUsecaseAuthorizeDingTalkMultipleOrganizations(t *testing.T) {
	source, actor, user, first, second, role := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := WithPrincipal(context.Background(), &Principal{Organization: Organization{ID: source, Kind: OrganizationKindSystem}, OrganizationNodes: []OrganizationScopeNode{{ID: source, Kind: OrganizationKindSystem}}})
	repo := &adminRepoStub{}
	uc := NewAdminUsecase(repo, inlineTransactor{})
	items := []AdminUserMembershipCreation{{OrganizationID: first}, {OrganizationID: second, RoleIDs: []uuid.UUID{role}}}
	if _, err := uc.AuthorizeDingTalkUser(ctx, source, actor, &AdminUser{ID: user}, items); err != nil {
		t.Fatal(err)
	}
	if len(repo.authorizationItems) != 2 || repo.authorizationItems[0].OrganizationID != first || len(repo.authorizationItems[0].RoleIDs) != 0 || repo.authorizationItems[1].RoleIDs[0] != role {
		t.Fatalf("组织角色未独立透传:%+v", repo.authorizationItems)
	}
	invalid := [][]AdminUserMembershipCreation{nil, {{OrganizationID: first}}, {{OrganizationID: first, RoleIDs: []uuid.UUID{role}}, {OrganizationID: first}}, {{OrganizationID: first, RoleIDs: []uuid.UUID{uuid.Nil}}}}
	oversized := make([]AdminUserMembershipCreation, MaxListPageSize+1)
	for i := range oversized {
		oversized[i] = AdminUserMembershipCreation{OrganizationID: uuid.New(), RoleIDs: []uuid.UUID{role}}
	}
	invalid = append(invalid, oversized)
	for i, items := range invalid {
		if _, err := uc.AuthorizeDingTalkUser(ctx, source, actor, &AdminUser{ID: user}, items); err != ErrAdminInvalidArgument {
			t.Fatalf("无效矩阵%d:%v", i, err)
		}
	}
}

func TestAdminUsecaseBatchDingTalkMultipleOrganizationsPartialSuccess(t *testing.T) {
	source, actor, success, failure := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctx := WithPrincipal(context.Background(), &Principal{Organization: Organization{ID: source, Kind: OrganizationKindSystem}, OrganizationNodes: []OrganizationScopeNode{{ID: source, Kind: OrganizationKindSystem}}})
	repo := &adminRepoStub{authorizationErrors: map[uuid.UUID]error{failure: ErrDingTalkAssignmentConflict}}
	uc := NewAdminUsecase(repo, inlineTransactor{})
	items := []AdminUserMembershipCreation{{OrganizationID: uuid.New(), RoleIDs: []uuid.UUID{uuid.New()}}, {OrganizationID: uuid.New()}}
	results, err := uc.BatchAuthorizeDingTalkUsers(ctx, source, actor, []uuid.UUID{success, failure, success}, items)
	if err != nil || len(results) != 2 || !results[0].Success || results[1].Success || results[1].Reason != ErrDingTalkAssignmentConflict.Reason {
		t.Fatalf("人员独立结果:%+v %v", results, err)
	}
	if len(repo.authorizationUsers) != 2 || len(repo.authorizationItems) != 2 {
		t.Fatalf("重复人员或组织矩阵丢失:%+v", repo)
	}
	if _, err := uc.BatchAuthorizeDingTalkUsers(ctx, source, actor, []uuid.UUID{uuid.Nil}, items); err != ErrAdminInvalidArgument {
		t.Fatal(err)
	}
	if _, err := uc.BatchAuthorizeDingTalkUsers(ctx, source, actor, make([]uuid.UUID, MaxListPageSize+1), items); err != ErrAdminInvalidArgument {
		t.Fatal(err)
	}
	if _, err := uc.BatchAuthorizeDingTalkUsers(ctx, source, actor, []uuid.UUID{success}, []AdminUserMembershipCreation{{OrganizationID: uuid.New()}}); err != ErrAdminInvalidArgument {
		t.Fatal(err)
	}
}
