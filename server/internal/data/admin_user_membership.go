package data

import (
	"context"
	"sort"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

func (r *adminRepo) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*biz.AdminUserMembership, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	exists, err := client.User.Query().Where(userent.IDEQ(userID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrAdminUserNotFound
	}
	scope, scopeErr := adminMembershipPredicate(ctx, client)
	if scopeErr != nil {
		return nil, scopeErr
	}
	items, err := client.Membership.Query().
		Where(membership.UserIDEQ(userID), scope).
		WithOrganization().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() }).
		Order(membership.ByOrganizationField(organization.FieldCode), membership.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AdminUserMembership, 0, len(items))
	for _, item := range items {
		result = append(result, membershipToBiz(item))
	}
	return result, nil
}

func (r *adminRepo) GetUserMembership(ctx context.Context, userID, membershipID uuid.UUID) (*biz.AdminUserMembership, error) {
	return r.findUserMembership(ctx, userID, membershipID)
}

func (r *adminRepo) CreateUserMembership(ctx context.Context, input *biz.AdminUserMembership, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUserMembership, error) {
	var created *ent.Membership
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if principal, ok := biz.PrincipalFromContext(ctx); ok {
			scope, err := adminWorkspaceScope(ctx, tx.Client(), principal.Organization.ID)
			if err != nil {
				return err
			}
			if principal.Organization.Kind != biz.OrganizationKindSystem && input.Primary {
				return biz.ErrPermissionDenied
			}
			if !uuidInValues(scope, input.OrganizationID) {
				return biz.ErrAdminOrganizationNotFound
			}
		}

		if _, queryErr := tx.User.Query().Where(userent.IDEQ(input.UserID)).ForUpdate().Only(ctx); queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		if exists, queryErr := tx.Organization.Query().Where(organization.IDEQ(input.OrganizationID), organization.EnabledEQ(true)).Exist(ctx); queryErr != nil {
			return queryErr
		} else if !exists {
			return biz.ErrAdminOrganizationNotFound
		}
		roles, rolesErr := rolesForOrganization(ctx, tx.Client(), input.OrganizationID, roleIDs)
		if rolesErr != nil {
			return rolesErr
		}
		var saveErr error
		created, saveErr = tx.Membership.Query().
			Where(membership.UserIDEQ(input.UserID), membership.OrganizationIDEQ(input.OrganizationID)).
			Only(ctx)
		if saveErr == nil {
			if created.Enabled {
				return biz.ErrAdminUserMembershipExists
			}
			created, saveErr = tx.Membership.UpdateOneID(created.ID).SetEnabled(true).SetPrimary(input.Primary).Save(ctx)
		} else if ent.IsNotFound(saveErr) {
			created, saveErr = tx.Membership.Create().
				SetUserID(input.UserID).
				SetOrganizationID(input.OrganizationID).
				SetEnabled(true).
				SetPrimary(input.Primary).
				Save(ctx)
		}
		if saveErr != nil {
			return saveErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, created.ID, roles); replaceErr != nil {
			return replaceErr
		}
		var preferredID *uuid.UUID
		if input.Primary {
			preferredID = &created.ID
		}
		if normalizeErr := normalizePrimaryMembership(ctx, tx, input.UserID, preferredID); normalizeErr != nil {
			return normalizeErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUserMembership(ctx, input.UserID, created.ID)
}

func (r *adminRepo) UpdateUserMembership(ctx context.Context, input *biz.AdminUserMembership, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUserMembership, error) {
	var current *ent.Membership
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		scope, scopeErr := adminMembershipPredicate(ctx, tx.Client())
		if scopeErr != nil {
			return scopeErr
		}
		account, queryErr := tx.User.Query().Where(userent.IDEQ(input.UserID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		current, queryErr = tx.Membership.Query().
			Where(membership.IDEQ(input.ID), membership.UserIDEQ(input.UserID), scope).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserMembershipNotFound, nil)
		}
		if principal, ok := biz.PrincipalFromContext(ctx); ok && principal.Organization.Kind != biz.OrganizationKindSystem && input.Primary != current.Primary {
			return biz.ErrPermissionDenied
		}
		if account.Enabled && current.Enabled && !input.Enabled {
			activeCount, countErr := tx.Membership.Query().Where(membership.UserIDEQ(input.UserID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Count(ctx)
			if countErr != nil {
				return countErr
			}
			if activeCount <= 1 {
				return biz.ErrAdminUserLastMembership
			}
			// 停用成员前检查当前/未来提成方案分配：存在未结束分配时拒绝停用，
			// 管理员须先以当天或未来的离开日期终止分配；成员行锁与方案写入
			// 共用串行化点，禁止静默保留无限期资格或直接删除历史分配。
			names, namesErr := activeCommissionAssignmentRuleNames(ctx, tx, current.OrganizationID, input.UserID, biz.FinanceBusinessDate(time.Now()))
			if namesErr != nil {
				return namesErr
			}
			if len(names) > 0 {
				return biz.NewCommissionRuleMemberAssignmentBlocked(names)
			}
		}
		roles, rolesErr := rolesForOrganization(ctx, tx.Client(), current.OrganizationID, roleIDs)
		if rolesErr != nil {
			return rolesErr
		}
		if _, saveErr := tx.Membership.UpdateOneID(current.ID).SetEnabled(input.Enabled).Save(ctx); saveErr != nil {
			return saveErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, current.ID, roles); replaceErr != nil {
			return replaceErr
		}
		var preferredID *uuid.UUID
		if input.Primary {
			preferredID = &current.ID
		}
		if normalizeErr := normalizePrimaryMembership(ctx, tx, input.UserID, preferredID); normalizeErr != nil {
			return normalizeErr
		}
		if !input.Enabled {
			if _, sessionErr := tx.Session.Update().
				Where(sessionent.UserIDEQ(input.UserID), sessionent.OrganizationIDEQ(current.OrganizationID), sessionent.RevokedAtIsNil()).
				SetRevokedAt(time.Now().UTC()).
				Save(ctx); sessionErr != nil {
				return sessionErr
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUserMembership(ctx, input.UserID, current.ID)
}

func (r *adminRepo) DeleteUserMembership(ctx context.Context, userID, membershipID uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		scope, scopeErr := adminMembershipPredicate(ctx, tx.Client())
		if scopeErr != nil {
			return scopeErr
		}
		account, queryErr := tx.User.Query().Where(userent.IDEQ(userID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		current, queryErr := tx.Membership.Query().
			Where(membership.IDEQ(membershipID), membership.UserIDEQ(userID), scope).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserMembershipNotFound, nil)
		}
		if account.Enabled && current.Enabled {
			activeCount, countErr := tx.Membership.Query().Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Count(ctx)
			if countErr != nil {
				return countErr
			}
			if activeCount <= 1 {
				return biz.ErrAdminUserLastMembership
			}
			// 删除成员同样走停用语义：仍有当前/未来方案分配时拒绝并返回需先
			// 处理的方案清单，不删除或截断历史分配。
			names, namesErr := activeCommissionAssignmentRuleNames(ctx, tx, current.OrganizationID, userID, biz.FinanceBusinessDate(time.Now()))
			if namesErr != nil {
				return namesErr
			}
			if len(names) > 0 {
				return biz.NewCommissionRuleMemberAssignmentBlocked(names)
			}
		}
		if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDEQ(current.ID)).Exec(ctx); deleteErr != nil {
			return deleteErr
		}
		if _, saveErr := tx.Membership.UpdateOneID(current.ID).SetEnabled(false).SetPrimary(false).Save(ctx); saveErr != nil {
			return saveErr
		}
		if normalizeErr := normalizePrimaryMembership(ctx, tx, userID, nil); normalizeErr != nil {
			return normalizeErr
		}
		if _, sessionErr := tx.Session.Update().
			Where(sessionent.UserIDEQ(userID), sessionent.OrganizationIDEQ(current.OrganizationID), sessionent.RevokedAtIsNil()).
			SetRevokedAt(time.Now().UTC()).
			Save(ctx); sessionErr != nil {
			return sessionErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}
func (r *adminRepo) findUserMembership(ctx context.Context, userID, membershipID uuid.UUID) (*biz.AdminUserMembership, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	scope, scopeErr := adminMembershipPredicate(ctx, client)
	if scopeErr != nil {
		return nil, scopeErr
	}
	item, err := client.Membership.Query().
		Where(membership.IDEQ(membershipID), membership.UserIDEQ(userID), scope).
		WithOrganization().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() }).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrAdminUserMembershipNotFound, nil)
	}
	return membershipToBiz(item), nil
}

func normalizePrimaryMembership(ctx context.Context, tx *ent.Tx, userID uuid.UUID, preferredID *uuid.UUID) error {
	items, err := tx.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		Order(membership.ByCreatedAt(), membership.ByID()).
		All(ctx)
	if err != nil {
		return err
	}
	var selectedID *uuid.UUID
	if preferredID != nil {
		for _, item := range items {
			if item.ID == *preferredID {
				id := item.ID
				selectedID = &id
				break
			}
		}
	}
	if selectedID == nil {
		for _, item := range items {
			if item.Primary {
				id := item.ID
				selectedID = &id
				break
			}
		}
	}
	if selectedID == nil && len(items) > 0 {
		id := items[0].ID
		selectedID = &id
	}
	if _, err := tx.Membership.Update().Where(membership.UserIDEQ(userID), membership.PrimaryEQ(true)).SetPrimary(false).Save(ctx); err != nil {
		return err
	}
	if selectedID == nil {
		return nil
	}
	_, err = tx.Membership.UpdateOneID(*selectedID).SetPrimary(true).Save(ctx)
	return err
}
func membershipToBiz(item *ent.Membership) *biz.AdminUserMembership {
	organizationRecord := item.Edges.Organization
	result := &biz.AdminUserMembership{
		ID:               item.ID,
		UserID:           item.UserID,
		OrganizationID:   item.OrganizationID,
		OrganizationCode: organizationRecord.Code,
		OrganizationName: organizationRecord.Name,
		OrganizationKind: biz.OrganizationKind(organizationRecord.Kind),
		Primary:          item.Primary,
		Enabled:          item.Enabled,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
	for _, assignment := range item.Edges.RoleAssignments {
		if assignedRole := assignment.Edges.Role; assignedRole != nil {
			result.RoleIDs = append(result.RoleIDs, assignedRole.ID)
			result.RoleCodes = append(result.RoleCodes, assignedRole.Code)
			result.RoleNames = append(result.RoleNames, assignedRole.Name)
		}
	}
	sortRoleCodeNames(result.RoleCodes, result.RoleNames)
	return result
}

// sortRoleCodeNames 按角色码升序排序并保持 code/name 下标一一对应，
// 两个切片必须等长（由同一循环成对追加保证）。
func sortRoleCodeNames(codes, names []string) {
	type codeNamePair struct{ code, name string }
	pairs := make([]codeNamePair, len(codes))
	for i := range codes {
		pairs[i] = codeNamePair{code: codes[i], name: names[i]}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].code < pairs[j].code })
	for i := range pairs {
		codes[i], names[i] = pairs[i].code, pairs[i].name
	}
}

// adminMembershipPredicate 将公司成员管理限制在本公司子树，系统工作台可管理各公司。
func adminMembershipPredicate(ctx context.Context, client *ent.Client) (predicate.Membership, error) {
	if principal, ok := biz.PrincipalFromContext(ctx); ok {
		ids, err := adminOrganizationManagementScope(ctx, client, principal.Organization.ID)
		if err != nil {
			return nil, err
		}
		return membership.OrganizationIDIn(ids...), nil
	}
	// 内部种子与隔离测试不经过登录请求；HTTP 层始终要求 Principal。
	return membership.IDNEQ(uuid.Nil), nil
}
