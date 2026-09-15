package data

import (
	"context"
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

// resolveRoleOrganizationID 解析组织对应的角色所属组织：
// 总部与公司自身独立维护角色库；部门与团队公用其所属公司的角色与工作区，
// 沿组织树向上回溯到最近的总部/公司祖先节点。组织不存在、断链或成环时
// 显式返回组织不存在错误，不回退为传入的组织 ID。
func resolveRoleOrganizationID(ctx context.Context, client *ent.Client, organizationID uuid.UUID) (uuid.UUID, error) {
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return uuid.Nil, err
	}
	effectiveID := workspaceAncestorID(authOrganizationNodes(nodes), organizationID)
	if effectiveID == uuid.Nil {
		return uuid.Nil, biz.ErrAdminOrganizationNotFound
	}
	return effectiveID, nil
}

// resolveRoleAnchorOrganizationID 校验角色写入的锚定组织：角色库只归属工作台
// （总部/公司），部门/团队锚点一律拒绝，不做静默归一。
func resolveRoleAnchorOrganizationID(ctx context.Context, client *ent.Client, organizationID uuid.UUID) (uuid.UUID, error) {
	resolved, err := resolveRoleOrganizationID(ctx, client, organizationID)
	if err != nil {
		return uuid.Nil, err
	}
	if resolved != organizationID {
		return uuid.Nil, biz.ErrAdminRoleAnchorInvalid
	}
	return resolved, nil
}

func (r *adminRepo) ListRoles(ctx context.Context, organizationID uuid.UUID) ([]*biz.AdminRole, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	roleOrgID, err := resolveRoleOrganizationID(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	items, err := client.Role.Query().Where(role.OrganizationIDEQ(roleOrgID)).WithPermissions().All(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	counts, err := roleAssignmentCounts(ctx, client, roleOrgID)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AdminRole, 0, len(items))
	for _, item := range items {
		roleItem := roleToBiz(item)
		roleItem.AssignmentsCount = counts[item.ID]
		result = append(result, roleItem)
	}
	return result, nil
}

func (r *adminRepo) GetRole(ctx context.Context, organizationID, id uuid.UUID) (*biz.AdminRole, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	roleOrgID, err := resolveRoleOrganizationID(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	item, err := client.Role.Query().
		Where(role.IDEQ(id), role.OrganizationIDEQ(roleOrgID)).
		WithPermissions().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrAdminRoleNotFound, nil)
	}
	roleItem := roleToBiz(item)
	counts, err := roleAssignmentCounts(ctx, client, roleOrgID)
	if err != nil {
		return nil, err
	}
	roleItem.AssignmentsCount = counts[item.ID]
	return roleItem, nil
}

// roleAssignmentCounts 一次性统计组织内各角色被分配到的成员关系数量，
// 供列表展示与删除前置判断使用，避免逐角色 N+1 查询。
func roleAssignmentCounts(ctx context.Context, client *ent.Client, organizationID uuid.UUID) (map[uuid.UUID]int, error) {
	rows := make([]struct {
		RoleID uuid.UUID `json:"role_id"`
		Count  int       `json:"count"`
	}, 0)
	if err := client.RoleAssignment.Query().
		Where(roleassignment.HasRoleWith(role.OrganizationIDEQ(organizationID))).
		GroupBy(roleassignment.FieldRoleID).
		Aggregate(ent.As(ent.Count(), "count")).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		counts[row.RoleID] = row.Count
	}
	return counts, nil
}

// DeleteRole 在同一事务内删除角色：事务内先校验锚定组织（只允许工作台）→ 锁定角色行
// （ForUpdate，与并发分配的成员关系外键校验互斥）→ 复核成员关系分配数（防止业务校验
// 与删除之间新增分配）→ 清除角色权限关联 → 删除角色 → 写审计。仍被其他数据引用时
// 统一映射为业务错误。
func (r *adminRepo) DeleteRole(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		roleOrgID, resolveErr := resolveRoleAnchorOrganizationID(ctx, tx.Client(), organizationID)
		if resolveErr != nil {
			return resolveErr
		}
		current, queryErr := tx.Role.Query().
			Where(role.IDEQ(id), role.OrganizationIDEQ(roleOrgID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminRoleNotFound, nil)
		}
		assignments, countErr := tx.RoleAssignment.Query().Where(roleassignment.RoleIDEQ(current.ID)).Count(ctx)
		if countErr != nil {
			return countErr
		}
		if assignments > 0 {
			return biz.ErrAdminRoleAssigned
		}
		if _, clearErr := tx.Role.UpdateOneID(current.ID).ClearPermissions().Save(ctx); clearErr != nil {
			return clearErr
		}
		if _, deleteErr := tx.Role.Delete().Where(role.IDEQ(current.ID)).Exec(ctx); deleteErr != nil {
			return mapEntError(deleteErr, nil, biz.ErrAdminRoleInUse)
		}
		audit.Details["resource_id"] = current.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *adminRepo) GetActorRolesPrivilegeProfiles(ctx context.Context, organizationID, actorID uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return actorRolesPrivilegeProfiles(ctx, client, organizationID, actorID)
}

func (r *adminRepo) GetRolesPrivilegeProfiles(ctx context.Context, organizationID uuid.UUID, roleIDs []uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return rolesPrivilegeProfiles(ctx, client, organizationID, roleIDs)
}

// actorRolesPrivilegeProfiles 查询某组织或其所属工作台子树成员资格下启用角色的权限画像，
// 供用户管理与钉钉邀请/注册审批的提权校验共用同一口径。无法解析出工作台（组织不存在、
// 断链或成环）时显式返回组织不存在错误，不回退为原始组织。
func actorRolesPrivilegeProfiles(ctx context.Context, client *ent.Client, organizationID, actorID uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return nil, err
	}
	nodeMap := authOrganizationNodes(nodes)
	workspaceID := workspaceAncestorID(nodeMap, organizationID)
	if workspaceID == uuid.Nil {
		return nil, biz.ErrAdminOrganizationNotFound
	}
	scopeIDs := workspaceMembershipScopeIDs(nodeMap, workspaceID)
	if len(scopeIDs) == 0 {
		return nil, biz.ErrAdminOrganizationNotFound
	}
	actorMemberships, err := client.Membership.Query().
		Where(
			membership.UserIDEQ(actorID),
			membership.OrganizationIDIn(scopeIDs...),
			membership.EnabledEQ(true),
			membership.HasUserWith(userent.EnabledEQ(true)),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
		).
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) {
			query.WithRole(func(roleQuery *ent.RoleQuery) {
				roleQuery.Where(role.EnabledEQ(true)).WithPermissions()
			})
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(actorMemberships) == 0 {
		return nil, biz.ErrAdminPrivilegeEscalation
	}

	seenRoles := make(map[uuid.UUID]struct{})
	profiles := make([]*biz.AdminRoleProfile, 0)
	for _, m := range actorMemberships {
		for _, assignment := range m.Edges.RoleAssignments {
			assignedRole := assignment.Edges.Role
			if assignedRole == nil {
				continue
			}
			if _, ok := seenRoles[assignedRole.ID]; ok {
				continue
			}
			seenRoles[assignedRole.ID] = struct{}{}
			profiles = append(profiles, roleProfileToBiz(assignedRole))
		}
	}
	return profiles, nil
}

// rolesPrivilegeProfiles 查询目标组织（或所属公司）内指定角色的权限画像（角色必须全部命中且启用）。
func rolesPrivilegeProfiles(ctx context.Context, client *ent.Client, organizationID uuid.UUID, roleIDs []uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	roleOrgID, err := resolveRoleOrganizationID(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	items, err := client.Role.Query().
		Where(role.OrganizationIDEQ(roleOrgID), role.IDIn(roleIDs...), role.EnabledEQ(true)).
		WithPermissions().
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) != len(uniqueUUIDs(roleIDs)) {
		return nil, biz.ErrAdminRoleNotFound
	}
	profiles := make([]*biz.AdminRoleProfile, 0, len(items))
	for _, item := range items {
		profiles = append(profiles, roleProfileToBiz(item))
	}
	return profiles, nil
}

func (r *adminRepo) CreateRole(ctx context.Context, organizationID uuid.UUID, input *biz.AdminRole, permissionKeys []string, audit *biz.AuditEvent) (*biz.AdminRole, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	permissions, err := permissionsByKeys(ctx, client.Permission.Query(), permissionKeys)
	if err != nil {
		return nil, err
	}
	var created *ent.Role
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		roleOrgID, resolveErr := resolveRoleAnchorOrganizationID(ctx, tx.Client(), organizationID)
		if resolveErr != nil {
			return resolveErr
		}
		var saveErr error
		created, saveErr = tx.Role.Create().SetOrganizationID(roleOrgID).SetCode(input.Code).SetName(input.Name).SetDataScope(role.DataScope(input.DataScope)).SetEnabled(input.Enabled).AddPermissions(permissions...).Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, nil, biz.ErrAdminRoleCodeExists)
		}
		audit.Details["resource_id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	created, err = client.Role.Query().Where(role.IDEQ(created.ID)).WithPermissions().Only(ctx)
	if err != nil {
		return nil, err
	}
	return roleToBiz(created), nil
}

func (r *adminRepo) UpdateRole(ctx context.Context, organizationID, id uuid.UUID, input *biz.AdminRole, permissionKeys []string, audit *biz.AuditEvent) (*biz.AdminRole, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	permissions, err := permissionsByKeys(ctx, client.Permission.Query(), permissionKeys)
	if err != nil {
		return nil, err
	}
	var updated *ent.Role
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		roleOrgID, resolveErr := resolveRoleAnchorOrganizationID(ctx, tx.Client(), organizationID)
		if resolveErr != nil {
			return resolveErr
		}
		var saveErr error
		updated, saveErr = tx.Role.UpdateOneID(id).Where(role.OrganizationIDEQ(roleOrgID)).SetName(input.Name).SetDataScope(role.DataScope(input.DataScope)).SetEnabled(input.Enabled).ClearPermissions().AddPermissions(permissions...).Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, biz.ErrAdminRoleNotFound, nil)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	updated, err = client.Role.Query().Where(role.IDEQ(updated.ID)).WithPermissions().Only(ctx)
	if err != nil {
		return nil, err
	}
	return roleToBiz(updated), nil
}

func (r *adminRepo) ListPermissions(ctx context.Context) ([]*biz.AdminPermission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.Permission.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	manifestItems := access.Manifest()
	requiresByKey := make(map[string][]string, len(manifestItems))
	sequence := make(map[string]int, len(manifestItems))
	groupRank := make(map[string]int)
	for index, definition := range manifestItems {
		requiresByKey[definition.Key] = definition.Requires
		sequence[definition.Key] = index
		if _, ok := groupRank[definition.Group]; !ok {
			groupRank[definition.Group] = len(groupRank)
		}
	}
	// 分组与组内条目均按 Manifest 声明顺序展示，组内不再按权限码字母序打乱；
	// 清单外的历史权限统一排在末尾。
	sort.SliceStable(items, func(i, j int) bool {
		indexI, knownI := sequence[items[i].Key]
		indexJ, knownJ := sequence[items[j].Key]
		if knownI != knownJ {
			return knownJ
		}
		if !knownI {
			if items[i].Group != items[j].Group {
				return items[i].Group < items[j].Group
			}
			return items[i].Key < items[j].Key
		}
		if groupRank[manifestItems[indexI].Group] != groupRank[manifestItems[indexJ].Group] {
			return groupRank[manifestItems[indexI].Group] < groupRank[manifestItems[indexJ].Group]
		}
		return indexI < indexJ
	})
	result := make([]*biz.AdminPermission, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.AdminPermission{Key: item.Key, Name: item.Name, Group: item.Group, Description: item.Description, Requires: requiresByKey[item.Key]})
	}
	return result, nil
}
func rolesForOrganization(ctx context.Context, client *ent.Client, organizationID uuid.UUID, roleIDs []uuid.UUID) ([]*ent.Role, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	roleOrgID, err := resolveRoleOrganizationID(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	roles, err := client.Role.Query().Where(role.OrganizationIDEQ(roleOrgID), role.IDIn(roleIDs...), role.EnabledEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(roles) != len(uniqueUUIDs(roleIDs)) {
		return nil, biz.ErrAdminRoleNotFound
	}
	return roles, nil
}

func replaceRoleAssignments(ctx context.Context, tx *ent.Tx, membershipID uuid.UUID, roles []*ent.Role) error {
	if _, err := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDEQ(membershipID)).Exec(ctx); err != nil {
		return err
	}
	for _, assignedRole := range roles {
		if _, err := tx.RoleAssignment.Create().SetMembershipID(membershipID).SetRoleID(assignedRole.ID).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func permissionsByKeys(ctx context.Context, query *ent.PermissionQuery, keys []string) ([]*ent.Permission, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	items, err := query.Where(permission.KeyIn(keys...)).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) != len(keys) {
		return nil, biz.ErrAdminPermissionInvalid
	}
	return items, nil
}

func uniqueUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
func roleToBiz(item *ent.Role) *biz.AdminRole {
	result := &biz.AdminRole{ID: item.ID, OrganizationID: item.OrganizationID, Code: item.Code, Name: item.Name, DataScope: biz.DataScope(item.DataScope), Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, permissionItem := range item.Edges.Permissions {
		result.PermissionKeys = append(result.PermissionKeys, permissionItem.Key)
	}
	sort.Strings(result.PermissionKeys)
	return result
}

func roleProfileToBiz(item *ent.Role) *biz.AdminRoleProfile {
	result := &biz.AdminRoleProfile{ID: item.ID, Code: item.Code, DataScope: biz.DataScope(item.DataScope)}
	for _, permissionItem := range item.Edges.Permissions {
		result.PermissionKeys = append(result.PermissionKeys, permissionItem.Key)
	}
	return result
}
