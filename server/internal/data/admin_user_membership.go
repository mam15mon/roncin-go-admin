package data

import (
	"context"
	"sort"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
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

// CreateUserMemberships 在用例共享事务中锁定授权依据并执行本批写入，返回提交后重读所需 ID。
func (r *adminRepo) CreateUserMemberships(ctx context.Context, actorOrganizationID, actorID, userID uuid.UUID, items []biz.AdminUserMembershipCreation, audits []*biz.AuditEvent) ([]uuid.UUID, error) {
	var createdIDs []uuid.UUID
	err := r.data.WithinTransaction(ctx, func(ctx context.Context) error {
		return r.data.WithTx(ctx, func(tx *ent.Tx) error {
			client := tx.Client()
			// 两个用户均从入口按主键顺序独占锁定，避免操作者/目标用户互换时交叉等待。
			accounts, err := client.User.Query().Where(userent.IDIn(actorID, userID)).Order(userent.ByID()).ForUpdate().All(ctx)
			if err != nil {
				return err
			}
			var account *ent.User
			for _, candidate := range accounts {
				if candidate.ID == userID {
					account = candidate
				}
			}
			if account == nil {
				return biz.ErrAdminUserNotFound
			}
			if account.PendingAssignment {
				return biz.ErrAdminUserAuthorizationRequired
			}
			if !account.Enabled {
				return biz.ErrAdminUserInactive
			}
			// 组织树规模小；按固定顺序锁全树，父链与工作台范围在事务结束前保持不变。
			nodes, err := client.Organization.Query().Order(organization.ByID()).ForShare().All(ctx)
			if err != nil {
				return err
			}
			nodeMap := authOrganizationNodes(nodes)
			members, err := client.Membership.Query().Where(membership.UserIDIn(actorID, userID)).Order(membership.ByID()).ForUpdate().All(ctx)
			if err != nil {
				return err
			}
			roleIDs := make([]uuid.UUID, 0)
			for _, item := range items {
				roleIDs = append(roleIDs, item.RoleIDs...)
			}
			// 成员父行锁保护角色绑定的替换；角色共享锁保护角色状态、权限与数据范围。
			_, err = client.Role.Query().Where(roleent.Or(roleent.IDIn(roleIDs...), roleent.HasAssignmentsWith(roleassignment.HasMembershipWith(membership.UserIDEQ(actorID))))).Order(roleent.ByID()).ForShare().All(ctx)
			if err != nil {
				return err
			}
			actorPrincipal, err := (&authRepo{data: r.data}).ResolvePrincipal(ctx, actorID, actorOrganizationID)
			if err != nil {
				return err
			}
			if !actorPrincipal.HasPermission(access.UserUpdate) {
				return biz.ErrPermissionDenied
			}
			lockedCtx := biz.WithPrincipal(ctx, actorPrincipal)
			allowed, err := adminOrganizationManagementScope(lockedCtx, client, actorOrganizationID)
			if err != nil {
				return err
			}
			rolesByItem := make([][]*ent.Role, len(items))
			for i, item := range items {
				if !uuidInValues(allowed, item.OrganizationID) {
					return biz.ErrAdminPrivilegeEscalation
				}
				target := nodeMap[item.OrganizationID]
				itemError := func(err error) error { return biz.AdminUserMembershipItemError(target.Name, err) }
				anchor := workspaceAncestorID(nodeMap, item.OrganizationID)
				if anchor == uuid.Nil {
					return itemError(biz.ErrAdminOrganizationNotFound)
				}
				for current := item.OrganizationID; ; {
					node := nodeMap[current]
					if !node.Enabled {
						return itemError(biz.ErrAdminOrganizationNotFound)
					}
					if current == anchor {
						break
					}
					current = *node.ParentID
				}
				if err = biz.NewAdminUsecase(r, r.data).ValidateUserMembershipRolesPrivilege(lockedCtx, actorOrganizationID, actorID, item.OrganizationID, item.RoleIDs); err != nil {
					return itemError(err)
				}
				rolesByItem[i], err = rolesForOrganization(lockedCtx, client, item.OrganizationID, item.RoleIDs)
				if err != nil {
					return itemError(err)
				}
				for _, existing := range members {
					if existing.UserID == userID && existing.OrganizationID == item.OrganizationID {
						return itemError(biz.ErrAdminUserMembershipExists)
					}
				}
			}
			for i, item := range items {
				member, err := client.Membership.Create().SetUserID(userID).SetOrganizationID(item.OrganizationID).SetEnabled(true).Save(lockedCtx)
				if err != nil {
					return biz.AdminUserMembershipItemError(nodeMap[item.OrganizationID].Name, mapEntError(err, nil, biz.ErrAdminUserMembershipExists))
				}
				// 新成员在当前事务中尚不可见，无需在持有角色锁后再次升级已有成员锁。
				for _, assignedRole := range rolesByItem[i] {
					if _, err = client.RoleAssignment.Create().SetMembershipID(member.ID).SetRoleID(assignedRole.ID).Save(lockedCtx); err != nil {
						return mapEntError(err, nil, biz.ErrAdminUserMembershipExists)
					}
				}
				if err = writeAudit(lockedCtx, tx.AuditLog, audits[i]); err != nil {
					return err
				}
				createdIDs = append(createdIDs, member.ID)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return createdIDs, nil
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
		if account.PendingAssignment {
			return biz.ErrAdminUserAuthorizationRequired
		}
		current, queryErr = tx.Membership.Query().
			Where(membership.IDEQ(input.ID), membership.UserIDEQ(input.UserID), scope).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserMembershipNotFound, nil)
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
		if _, saveErr := tx.Membership.UpdateOneID(current.ID).SetEnabled(false).Save(ctx); saveErr != nil {
			return saveErr
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

func membershipToBiz(item *ent.Membership) *biz.AdminUserMembership {
	organizationRecord := item.Edges.Organization
	result := &biz.AdminUserMembership{
		ID:               item.ID,
		UserID:           item.UserID,
		OrganizationID:   item.OrganizationID,
		OrganizationCode: organizationRecord.Code,
		OrganizationName: organizationRecord.Name,
		OrganizationKind: biz.OrganizationKind(organizationRecord.Kind),
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
