package data

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// companyPersonnelOrganizationIDs 仅纳入当前公司及其启用部门/团队，遇到另一公司停止向下遍历。
// 候选查询和业务岗位写入共同使用此边界，避免可选范围与可保存范围不一致。
func companyPersonnelOrganizationIDs(ctx context.Context, client *ent.Client, companyID uuid.UUID) ([]uuid.UUID, error) {
	organizations, err := client.Organization.Query().Select(organizationent.FieldID, organizationent.FieldParentID, organizationent.FieldKind, organizationent.FieldEnabled).All(ctx)
	if err != nil {
		return nil, err
	}
	return personnelOrganizationIDs(organizations, companyID), nil
}

func personnelOrganizationIDs(organizations []*ent.Organization, companyID uuid.UUID) []uuid.UUID {
	children := make(map[uuid.UUID][]*ent.Organization)
	var company *ent.Organization
	for _, org := range organizations {
		if org.ID == companyID {
			company = org
		}
		if org.ParentID != nil {
			children[*org.ParentID] = append(children[*org.ParentID], org)
		}
	}
	if company == nil || !company.Enabled || company.Kind != organizationent.KindCompany {
		return nil
	}
	result := []uuid.UUID{companyID}
	seen := map[uuid.UUID]bool{companyID: true}
	for i := 0; i < len(result); i++ {
		for _, org := range children[result[i]] {
			if seen[org.ID] || !org.Enabled || (org.Kind != organizationent.KindDepartment && org.Kind != organizationent.KindTeam) {
				continue
			}
			seen[org.ID] = true
			result = append(result, org.ID)
		}
	}
	return result
}

func companyPersonnelQuery(ctx context.Context, client *ent.Client, companyID uuid.UUID, keyword string) (*ent.UserQuery, error) {
	organizationIDs, err := companyPersonnelOrganizationIDs(ctx, client, companyID)
	if err != nil {
		return nil, err
	}
	scope := membershipent.And(membershipent.OrganizationIDIn(organizationIDs...), membershipent.EnabledEQ(true))
	query := client.User.Query().Where(userent.EnabledEQ(true), userent.HasMembershipsWith(scope)).
		WithMemberships(func(q *ent.MembershipQuery) { q.Where(scope).WithOrganization() })
	if keyword != "" {
		query.Where(userent.Or(
			userent.DisplayNameContainsFold(keyword), userent.SearchKeywordsContainsFold(keyword),
			userent.HasMembershipsWith(scope, membershipent.HasOrganizationWith(organizationent.Or(
				organizationent.CodeContainsFold(keyword), organizationent.NameContainsFold(keyword), organizationent.SearchKeywordsContainsFold(keyword),
			))),
		))
	}
	return query, nil
}

func personnelDepartmentNames(person *ent.User) []string {
	names := make(map[string]struct{})
	for _, membership := range person.Edges.Memberships {
		org := membership.Edges.Organization
		if org != nil && (org.Kind == organizationent.KindDepartment || org.Kind == organizationent.KindTeam) {
			names[org.Name] = struct{}{}
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
