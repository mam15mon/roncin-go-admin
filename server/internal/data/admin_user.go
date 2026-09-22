package data

import (
	"context"
	"sort"
	"strings"
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

// adminWorkspaceScopeIDs 返回用户管理工作台的组织管理范围：系统管理工作台 = 全树全部
// 启用组织；公司工作台 = 公司子树（含公司节点本身）中的启用组织；非启用工作台节点
// 返回空。注意与 auth.workspaceMembershipScopeIDs 的权限解析口径区分：后者是角色
// 生效口径（系统管理工作台只聚合系统管理节点本身成员关系上的角色），本函数是用户管理的
// 可见/可管理范围口径（系统管理可管理全树任意组织的成员，公司只管理本公司子树成员）。
func adminWorkspaceScopeIDs(nodes map[uuid.UUID]authOrgNode, workspaceID uuid.UUID) []uuid.UUID {
	node, ok := nodes[workspaceID]
	if !ok || !node.Enabled || !isWorkspaceKind(node.Kind) {
		return nil
	}
	var candidateIDs []uuid.UUID
	if node.Kind == string(organization.KindSystem) {
		candidateIDs = make([]uuid.UUID, 0, len(nodes))
		for _, item := range nodes {
			candidateIDs = append(candidateIDs, item.ID)
		}
	} else {
		candidateIDs = organizationSubtreeIDs(nodes, workspaceID)
	}
	scope := make([]uuid.UUID, 0, len(candidateIDs))
	for _, candidateID := range candidateIDs {
		if nodes[candidateID].Enabled {
			scope = append(scope, candidateID)
		}
	}
	return scope
}

// adminWorkspaceScope 一次加载组织树并计算工作台管理范围，避免逐层/N+1 查询。
func adminWorkspaceScope(ctx context.Context, client *ent.Client, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return nil, err
	}
	return adminWorkspaceScopeIDs(authOrganizationNodes(nodes), workspaceID), nil
}

// preferAdminAnchorMembership 按锚定规则从工作台范围内的候选成员关系中选取锚定
// 成员关系（决定行的角色展示与 CurrentMembershipEnabled）；候选为空返回 nil。
func preferAdminAnchorMembership(candidates []*ent.Membership) *ent.Membership {
	var anchor *ent.Membership
	for _, candidate := range candidates {
		if anchor == nil || betterAdminAnchorMembership(candidate, anchor) {
			anchor = candidate
		}
	}
	return anchor
}

// betterAdminAnchorMembership 报告 candidate 是否比 current 更适合作为锚定成员
// 关系：primary 优先，其次启用关系，再则创建时间最早；ID 升序做稳定兜底。
func betterAdminAnchorMembership(candidate, current *ent.Membership) bool {
	if candidate.Primary != current.Primary {
		return candidate.Primary
	}
	if candidate.Enabled != current.Enabled {
		return candidate.Enabled
	}
	if !candidate.CreatedAt.Equal(current.CreatedAt) {
		return candidate.CreatedAt.Before(current.CreatedAt)
	}
	return candidate.ID.String() < current.ID.String()
}

// adminUserRow 把用户实体转换为工作台视角的用户行：在管理范围内选取锚定成员关系，
// 组织摘要仍展示全部启用成员关系（与旧口径一致）。范围内无任何成员关系时视为
// 用户不在当前工作台范围。
func adminUserRow(account *ent.User, scope map[uuid.UUID]struct{}) (*biz.AdminUser, error) {
	candidates := make([]*ent.Membership, 0, len(account.Edges.Memberships))
	active := make([]*ent.Membership, 0, len(account.Edges.Memberships))
	for _, member := range account.Edges.Memberships {
		if _, inScope := scope[member.OrganizationID]; inScope {
			candidates = append(candidates, member)
		}
		if _, inScope := scope[member.OrganizationID]; inScope && member.Enabled && member.Edges.Organization != nil && member.Edges.Organization.Enabled {
			active = append(active, member)
		}
	}
	anchor := preferAdminAnchorMembership(candidates)
	if anchor == nil {
		return nil, biz.ErrAdminUserNotFound
	}
	anchor.Edges.User = account
	account.Edges.Memberships = active
	return membershipToUser(anchor), nil
}

// adminUserScopeSet 把范围切片转为成员关系归属判定用的集合。
func adminUserScopeSet(scope []uuid.UUID) map[uuid.UUID]struct{} {
	scopeSet := make(map[uuid.UUID]struct{}, len(scope))
	for _, scopeID := range scope {
		scopeSet[scopeID] = struct{}{}
	}
	return scopeSet
}

// listAdminUsersQuery 装载用户及其全部成员关系（含停用，保留查看被移除记录的能力）
// 与成员关系角色，锚定与摘要转换在 adminUserRow 中纯内存完成。
func listAdminUsersQuery(client *ent.Client, predicates []predicate.User) *ent.UserQuery {
	return client.User.Query().Where(predicates...).
		WithMemberships(func(query *ent.MembershipQuery) {
			query.WithOrganization().WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() })
		})
}

func (r *adminRepo) ListUsers(ctx context.Context, organizationID uuid.UUID, options biz.AdminUserListOptions) (*biz.AdminUserList, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := adminWorkspaceScope(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	if len(scope) == 0 {
		return &biz.AdminUserList{Items: []*biz.AdminUser{}, Page: options.Page, PageSize: options.PageSize}, nil
	}
	predicates := []predicate.User{
		userent.HasMembershipsWith(membership.OrganizationIDIn(scope...)),
	}
	if options.Keyword != "" {
		predicates = append(predicates, userent.Or(
			userent.UsernameContainsFold(options.Keyword),
			userent.DisplayNameContainsFold(options.Keyword),
			userent.SearchKeywordsContainsFold(options.Keyword),
		))
	}
	if options.Enabled != nil {
		predicates = append(predicates, userent.EnabledEQ(*options.Enabled))
	}
	scopeSet := adminUserScopeSet(scope)
	query := listAdminUsersQuery(client, predicates)
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.User, error) {
		return query.Order(userent.ByUsername(), userent.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, func(item *ent.User) (*biz.AdminUser, error) {
		return adminUserRow(item, scopeSet)
	})
}

func (r *adminRepo) CreateUser(ctx context.Context, organizationID uuid.UUID, input *biz.AdminUser, passwordHash string, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	var account *ent.User
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		roles, queryErr := rolesForOrganization(ctx, tx.Client(), organizationID, roleIDs)
		if queryErr != nil {
			return queryErr
		}
		create := tx.User.Create().SetUsername(input.Username).SetDisplayName(input.DisplayName).SetPasswordHash(passwordHash).SetEnabled(input.Enabled)
		if input.Email != nil {
			create.SetEmail(*input.Email)
		}
		var createErr error
		account, createErr = create.Save(ctx)
		if createErr != nil {
			return mapEntError(createErr, nil, biz.ErrAdminUsernameExists)
		}
		membershipRecord, createErr := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(organizationID).SetPrimary(true).SetEnabled(true).Save(ctx)
		if createErr != nil {
			return createErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, membershipRecord.ID, roles); replaceErr != nil {
			return replaceErr
		}
		audit.Details["resource_id"] = account.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUser(ctx, organizationID, account.ID)
}

func (r *adminRepo) UpdateUser(ctx context.Context, organizationID, id uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	if _, ok := biz.PrincipalFromContext(ctx); ok && !biz.IsSystemWorkspace(ctx) {
		return nil, biz.ErrPermissionDenied
	}
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := tx.User.Query().Where(userent.IDEQ(id)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		// 锚定成员关系在工作台管理范围内选取，不再要求当前工作台组织上的精确启用
		// 成员关系（范围口径见 adminWorkspaceScopeIDs）。成员关系写入路径都先锁用户
		// 行，这里复用用户行锁即可，无需再单独锁定成员关系行。
		scope, scopeErr := adminWorkspaceScope(ctx, tx.Client(), organizationID)
		if scopeErr != nil {
			return scopeErr
		}
		if len(scope) == 0 {
			return biz.ErrAdminUserNotFound
		}
		candidates, queryErr := tx.Membership.Query().
			Where(membership.UserIDEQ(id), membership.OrganizationIDIn(scope...)).
			All(ctx)
		if queryErr != nil {
			return queryErr
		}
		anchor := preferAdminAnchorMembership(candidates)
		if anchor == nil {
			return biz.ErrAdminUserNotFound
		}
		if account.Enabled && !input.Enabled {
			return biz.ErrAdminUserTerminationRequired
		}
		if !account.Enabled && input.Enabled && (account.WecomUserid != nil || account.DingtalkUnionid != nil) {
			return biz.ErrAdminUserAuthorizationRequired
		}
		// 角色必须与锚定成员关系同组织（既有规则不变）：按锚定组织校验后写入该成员关系。
		roles, queryErr := rolesForOrganization(ctx, tx.Client(), anchor.OrganizationID, roleIDs)
		if queryErr != nil {
			return queryErr
		}
		update := tx.User.UpdateOneID(id).SetDisplayName(input.DisplayName).SetEnabled(input.Enabled)
		if input.Email == nil {
			update.ClearEmail()
		} else {
			update.SetEmail(*input.Email)
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return mapEntError(updateErr, biz.ErrAdminUserNotFound, nil)
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, anchor.ID, roles); replaceErr != nil {
			return replaceErr
		}
		audit.Details["value"] = account.Username
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUser(ctx, organizationID, id)
}
func (r *adminRepo) TerminateUser(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	if _, ok := biz.PrincipalFromContext(ctx); ok && !biz.IsSystemWorkspace(ctx) {
		return biz.ErrPermissionDenied
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, queryErr := tx.User.Query().Where(userent.IDEQ(id)).ForUpdate().Only(ctx); queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		// 工作台范围内存在启用成员关系即可办理离职（不再要求当前工作台组织上的精确关系）。
		scope, scopeErr := adminWorkspaceScope(ctx, tx.Client(), organizationID)
		if scopeErr != nil {
			return scopeErr
		}
		if len(scope) == 0 {
			return biz.ErrAdminUserNotFound
		}
		inScope, queryErr := tx.Membership.Query().
			Where(membership.UserIDEQ(id), membership.OrganizationIDIn(scope...), membership.EnabledEQ(true)).
			Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !inScope {
			return biz.ErrAdminUserNotFound
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(id)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(id)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, updateErr := tx.User.UpdateOneID(id).SetEnabled(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(id), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *adminRepo) AuthorizeWeComUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	return r.authorizePendingUser(ctx, sourceOrganizationID, targetOrganizationID, input, roleIDs, nil, audit, func(account *ent.User) bool {
		return account.WecomUserid != nil
	})
}

func (r *adminRepo) AuthorizeDingTalkUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, notification *biz.NotificationIntent, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	return r.authorizePendingUser(ctx, sourceOrganizationID, targetOrganizationID, input, roleIDs, notification, audit, func(account *ent.User) bool {
		return account.DingtalkUnionid != nil && account.DingtalkUserid != nil && strings.TrimSpace(*account.DingtalkUserid) != ""
	})
}

func (r *adminRepo) authorizePendingUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, notification *biz.NotificationIntent, audit *biz.AuditEvent, hasExternalIdentity func(*ent.User) bool) (*biz.AdminUser, error) {
	if _, ok := biz.PrincipalFromContext(ctx); ok && !biz.IsSystemWorkspace(ctx) {
		return nil, biz.ErrPermissionDenied
	}
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := tx.User.Query().Where(userent.IDEQ(input.ID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		// 待授权用户在来源工作台管理范围内存在启用成员关系即可发起授权（范围口径
		// 见 adminWorkspaceScopeIDs）；目标组织逻辑保持不变。
		sourceScope, scopeErr := adminWorkspaceScope(ctx, tx.Client(), sourceOrganizationID)
		if scopeErr != nil {
			return scopeErr
		}
		if len(sourceScope) == 0 {
			return biz.ErrAdminUserNotFound
		}
		sourceInScope, queryErr := tx.Membership.Query().
			Where(membership.UserIDEQ(input.ID), membership.OrganizationIDIn(sourceScope...), membership.EnabledEQ(true)).
			Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !sourceInScope {
			return biz.ErrAdminUserNotFound
		}
		if !hasExternalIdentity(account) || account.Enabled {
			return biz.ErrAdminInvalidArgument
		}
		exists, queryErr := tx.Organization.Query().Where(organization.IDEQ(targetOrganizationID), organization.EnabledEQ(true), organization.KindEQ(organization.KindCompany)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrAdminOrganizationNotFound
		}
		roles, queryErr := rolesForOrganization(ctx, tx.Client(), targetOrganizationID, roleIDs)
		if queryErr != nil {
			return queryErr
		}
		userUpdate := tx.User.UpdateOneID(input.ID).SetDisplayName(input.DisplayName).SetEnabled(true)
		if input.Email == nil {
			userUpdate.ClearEmail()
		} else {
			userUpdate.SetEmail(*input.Email)
		}
		if _, updateErr := userUpdate.Save(ctx); updateErr != nil {
			return updateErr
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(input.ID)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(input.ID)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		targetMembership, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(input.ID), membership.OrganizationIDEQ(targetOrganizationID)).Only(ctx)
		if ent.IsNotFound(queryErr) {
			targetMembership, queryErr = tx.Membership.Create().SetUserID(input.ID).SetOrganizationID(targetOrganizationID).SetEnabled(true).SetPrimary(true).Save(ctx)
		} else if queryErr == nil {
			targetMembership, queryErr = tx.Membership.UpdateOneID(targetMembership.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
		}
		if queryErr != nil {
			return queryErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, targetMembership.ID, roles); replaceErr != nil {
			return replaceErr
		}
		if notification != nil {
			if notificationErr := enqueueDingTalkUserAuthorizedNotification(ctx, tx, targetOrganizationID, account, notification); notificationErr != nil {
				return notificationErr
			}
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(input.ID), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		audit.Details["value"] = account.Username
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUser(ctx, targetOrganizationID, input.ID)
}

func (r *adminRepo) ResetUserPassword(ctx context.Context, organizationID, id uuid.UUID, passwordHash string, username *string, audit *biz.AuditEvent) error {
	if _, ok := biz.PrincipalFromContext(ctx); ok && !biz.IsSystemWorkspace(ctx) {
		return biz.ErrPermissionDenied
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		targetUser, queryErr := tx.User.Query().
			Where(userent.IDEQ(id), userent.EnabledEQ(true)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		// 工作台管理范围内存在启用成员关系即可重置密码（不再要求当前工作台组织上的
		// 精确关系；范围口径见 adminWorkspaceScopeIDs）。
		scope, scopeErr := adminWorkspaceScope(ctx, tx.Client(), organizationID)
		if scopeErr != nil {
			return scopeErr
		}
		if len(scope) == 0 {
			return biz.ErrAdminUserNotFound
		}
		exists, queryErr := tx.Membership.Query().Where(
			membership.UserIDEQ(id),
			membership.OrganizationIDIn(scope...),
			membership.EnabledEQ(true),
		).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrAdminUserNotFound
		}
		userUpdate := tx.User.UpdateOneID(id).SetPasswordHash(passwordHash)
		if username != nil && strings.TrimSpace(*username) != "" {
			userUpdate.SetUsername(strings.TrimSpace(*username))
		} else if targetUser.Username == "" {
			return biz.ErrAdminInvalidArgument
		}
		if _, updateErr := userUpdate.Save(ctx); updateErr != nil {
			return mapEntError(updateErr, biz.ErrAdminUserNotFound, biz.ErrAdminUsernameExists)
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(id), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// GetUser 返回用户在工作台管理范围下的锚定视图，供用例层在编辑前取得锚定组织：
// 角色必须与锚定成员关系同组织，提权校验与写入都按锚定组织执行。
func (r *adminRepo) GetUser(ctx context.Context, organizationID, userID uuid.UUID) (*biz.AdminUser, error) {
	return r.findUser(ctx, organizationID, userID)
}

// findUser 在工作台管理范围内按锚定规则选取成员关系并组装用户视图；范围内无任何
// 成员关系时报「用户不存在或不在当前工作台范围」。
func (r *adminRepo) findUser(ctx context.Context, organizationID, userID uuid.UUID) (*biz.AdminUser, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := adminWorkspaceScope(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	if len(scope) == 0 {
		return nil, biz.ErrAdminUserNotFound
	}
	account, err := listAdminUsersQuery(client, []predicate.User{userent.IDEQ(userID)}).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrAdminUserNotFound, nil)
	}
	return adminUserRow(account, adminUserScopeSet(scope))
}
func membershipToUser(item *ent.Membership) *biz.AdminUser {
	account := item.Edges.User
	result := &biz.AdminUser{ID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, AvatarURL: account.AvatarURL, WeComUserID: account.WecomUserid, WeComName: account.WecomName, DingTalkUnionID: account.DingtalkUnionid, DingTalkUserID: account.DingtalkUserid, DingTalkName: account.DingtalkName, Enabled: account.Enabled, CurrentMembershipEnabled: item.Enabled, CurrentOrganizationID: item.OrganizationID, HasPassword: account.PasswordHash != nil, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt}
	for _, assignment := range item.Edges.RoleAssignments {
		if assignedRole := assignment.Edges.Role; assignedRole != nil {
			result.RoleIDs = append(result.RoleIDs, assignedRole.ID)
			result.RoleCodes = append(result.RoleCodes, assignedRole.Code)
			result.RoleNames = append(result.RoleNames, assignedRole.Name)
		}
	}
	sortRoleCodeNames(result.RoleCodes, result.RoleNames)
	for _, membershipRecord := range account.Edges.Memberships {
		organizationRecord := membershipRecord.Edges.Organization
		if organizationRecord == nil {
			continue
		}
		result.Organizations = append(result.Organizations, &biz.AdminUserOrganizationSummary{
			OrganizationID: organizationRecord.ID,
			Name:           organizationRecord.Name,
			Primary:        membershipRecord.Primary,
		})
	}
	sort.Slice(result.Organizations, func(i, j int) bool {
		if result.Organizations[i].Primary != result.Organizations[j].Primary {
			return result.Organizations[i].Primary
		}
		return result.Organizations[i].Name < result.Organizations[j].Name
	})
	result.Status = adminUserStatus(account, item, len(account.Edges.Memberships) > 0)
	return result
}

func adminUserStatus(account *ent.User, currentMembership *ent.Membership, hasActiveMembership bool) biz.AdminUserStatus {
	if !currentMembership.Enabled && hasActiveMembership {
		return biz.AdminUserStatusRemovedFromOrganization
	}
	if !account.Enabled && !hasActiveMembership {
		return biz.AdminUserStatusTerminated
	}
	if account.Enabled && currentMembership.Enabled {
		return biz.AdminUserStatusActive
	}
	hasExternalIdentity := account.WecomUserid != nil || account.DingtalkUnionid != nil
	if !account.Enabled && currentMembership.Enabled && hasExternalIdentity && len(currentMembership.Edges.RoleAssignments) == 0 {
		return biz.AdminUserStatusPendingAuthorization
	}
	return biz.AdminUserStatusDisabled
}
