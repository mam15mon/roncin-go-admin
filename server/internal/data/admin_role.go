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
	roleorderorganizationaccess "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleorderorganizationaccess"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

func (r *adminRepo) ListRoles(ctx context.Context, organizationID uuid.UUID) ([]*biz.AdminRole, error) {
	items, err := r.data.db.Role.Query().Where(role.OrganizationIDEQ(organizationID)).WithPermissions().WithOrderOrganizationAccesses().All(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	result := make([]*biz.AdminRole, 0, len(items))
	for _, item := range items {
		result = append(result, roleToBiz(item))
	}
	return result, nil
}

func (r *adminRepo) GetRole(ctx context.Context, organizationID, id uuid.UUID) (*biz.AdminRole, error) {
	item, err := r.data.db.Role.Query().
		Where(role.IDEQ(id), role.OrganizationIDEQ(organizationID)).
		WithPermissions().
		WithOrderOrganizationAccesses().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminRoleNotFound
		}
		return nil, err
	}
	return roleToBiz(item), nil
}

func (r *adminRepo) GetActorRolesPrivilegeProfiles(ctx context.Context, organizationID, actorID uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	actorMembership, err := r.data.db.Membership.Query().
		Where(
			membership.UserIDEQ(actorID),
			membership.OrganizationIDEQ(organizationID),
			membership.EnabledEQ(true),
			membership.HasUserWith(userent.EnabledEQ(true)),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
		).
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) {
			query.WithRole(func(roleQuery *ent.RoleQuery) {
				roleQuery.Where(role.EnabledEQ(true)).WithPermissions().WithOrderOrganizationAccesses()
			})
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminPrivilegeEscalation
		}
		return nil, err
	}

	profiles := make([]*biz.AdminRoleProfile, 0, len(actorMembership.Edges.RoleAssignments))
	for _, assignment := range actorMembership.Edges.RoleAssignments {
		assignedRole := assignment.Edges.Role
		if assignedRole == nil {
			continue
		}
		profiles = append(profiles, roleProfileToBiz(assignedRole))
	}
	return profiles, nil
}

func (r *adminRepo) GetRolesPrivilegeProfiles(ctx context.Context, organizationID uuid.UUID, roleIDs []uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	items, err := r.data.db.Role.Query().
		Where(role.OrganizationIDEQ(organizationID), role.IDIn(roleIDs...), role.EnabledEQ(true)).
		WithPermissions().
		WithOrderOrganizationAccesses().
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
	permissions, err := permissionsByKeys(ctx, r.data.db.Permission.Query(), permissionKeys)
	if err != nil {
		return nil, err
	}
	var created *ent.Role
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var saveErr error
		created, saveErr = tx.Role.Create().SetOrganizationID(organizationID).SetCode(input.Code).SetName(input.Name).SetDataScope(role.DataScope(input.DataScope)).SetEnabled(input.Enabled).AddPermissions(permissions...).Save(ctx)
		if saveErr != nil {
			if ent.IsConstraintError(saveErr) {
				return biz.ErrAdminRoleCodeExists
			}
			return saveErr
		}
		if replaceErr := replaceRoleOrderOrganizationAccesses(ctx, tx, created.ID, input.OrderOrganizationAccesses); replaceErr != nil {
			return replaceErr
		}
		audit.Details["resource_id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	created, err = r.data.db.Role.Query().Where(role.IDEQ(created.ID)).WithPermissions().WithOrderOrganizationAccesses().Only(ctx)
	if err != nil {
		return nil, err
	}
	return roleToBiz(created), nil
}

func (r *adminRepo) UpdateRole(ctx context.Context, organizationID, id uuid.UUID, input *biz.AdminRole, permissionKeys []string, audit *biz.AuditEvent) (*biz.AdminRole, error) {
	permissions, err := permissionsByKeys(ctx, r.data.db.Permission.Query(), permissionKeys)
	if err != nil {
		return nil, err
	}
	var updated *ent.Role
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var saveErr error
		updated, saveErr = tx.Role.UpdateOneID(id).Where(role.OrganizationIDEQ(organizationID)).SetName(input.Name).SetDataScope(role.DataScope(input.DataScope)).SetEnabled(input.Enabled).ClearPermissions().AddPermissions(permissions...).Save(ctx)
		if saveErr != nil {
			if ent.IsNotFound(saveErr) {
				return biz.ErrAdminRoleNotFound
			}
			return saveErr
		}
		if replaceErr := replaceRoleOrderOrganizationAccesses(ctx, tx, updated.ID, input.OrderOrganizationAccesses); replaceErr != nil {
			return replaceErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	updated, err = r.data.db.Role.Query().Where(role.IDEQ(updated.ID)).WithPermissions().WithOrderOrganizationAccesses().Only(ctx)
	if err != nil {
		return nil, err
	}
	return roleToBiz(updated), nil
}

func (r *adminRepo) ListPermissions(ctx context.Context) ([]*biz.AdminPermission, error) {
	items, err := r.data.db.Permission.Query().All(ctx)
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
func rolesForOrganization(ctx context.Context, query *ent.RoleQuery, organizationID uuid.UUID, roleIDs []uuid.UUID) ([]*ent.Role, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	roles, err := query.Where(role.OrganizationIDEQ(organizationID), role.IDIn(roleIDs...), role.EnabledEQ(true)).All(ctx)
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
	for _, access := range item.Edges.OrderOrganizationAccesses {
		result.OrderOrganizationAccesses = append(result.OrderOrganizationAccesses, biz.OrderOrganizationAccess{OrganizationID: access.OrganizationID, Writable: access.Writable})
	}
	sort.Strings(result.PermissionKeys)
	sort.Slice(result.OrderOrganizationAccesses, func(i, j int) bool {
		return result.OrderOrganizationAccesses[i].OrganizationID.String() < result.OrderOrganizationAccesses[j].OrganizationID.String()
	})
	return result
}

func roleProfileToBiz(item *ent.Role) *biz.AdminRoleProfile {
	result := &biz.AdminRoleProfile{ID: item.ID, Code: item.Code, DataScope: biz.DataScope(item.DataScope)}
	for _, permissionItem := range item.Edges.Permissions {
		result.PermissionKeys = append(result.PermissionKeys, permissionItem.Key)
	}
	for _, access := range item.Edges.OrderOrganizationAccesses {
		result.OrderOrganizationAccesses = append(result.OrderOrganizationAccesses, biz.OrderOrganizationAccess{OrganizationID: access.OrganizationID, Writable: access.Writable})
	}
	return result
}

func replaceRoleOrderOrganizationAccesses(ctx context.Context, tx *ent.Tx, roleID uuid.UUID, accesses []biz.OrderOrganizationAccess) error {
	if _, err := tx.RoleOrderOrganizationAccess.Delete().Where(roleorderorganizationaccess.RoleIDEQ(roleID)).Exec(ctx); err != nil {
		return err
	}
	for _, access := range accesses {
		if _, err := tx.RoleOrderOrganizationAccess.Create().SetRoleID(roleID).SetOrganizationID(access.OrganizationID).SetWritable(access.Writable).Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
