package data

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	roleorderorganizationaccess "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleorderorganizationaccess"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

type adminRepo struct{ data *Data }

func NewAdminRepo(data *Data) biz.AdminRepo { return &adminRepo{data: data} }

func (r *adminRepo) ListOrganizations(ctx context.Context) ([]*biz.AdminOrganization, error) {
	items, err := r.data.db.Organization.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	result := make([]*biz.AdminOrganization, 0, len(items))
	for _, item := range items {
		converted, convertErr := organizationToBizWithCurrency(item, items)
		if convertErr != nil {
			return nil, convertErr
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *adminRepo) GetOrganization(ctx context.Context, id uuid.UUID) (*biz.AdminOrganization, error) {
	item, err := r.data.db.Organization.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminOrganizationNotFound
		}
		return nil, err
	}
	return r.organizationToBiz(ctx, item)
}

func (r *adminRepo) CreateOrganization(ctx context.Context, input *biz.AdminOrganization) (*biz.AdminOrganization, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	create := tx.Organization.Create().SetCode(input.Code).SetName(input.Name).SetKind(organization.Kind(input.Kind))
	if input.Kind == biz.OrganizationKindCompany || input.Kind == biz.OrganizationKindHeadquarters {
		if err := validateOrganizationCurrency(ctx, tx.Currency, input.BaseCurrency); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		create.SetBaseCurrency(input.BaseCurrency)
	}
	if input.ParentID != nil {
		create.SetParentID(*input.ParentID)
	}
	created, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrAdminOrganizationCodeExists
		}
		return nil, err
	}
	if err := CreateDefaultNumberRules(ctx, tx, created.ID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := CreateDefaultStatusTemplates(ctx, tx, created.ID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := CreateDefaultOrderOptions(ctx, tx, created.ID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result := organizationToBiz(created)
	if result.BaseCurrency == "" {
		result.BaseCurrency = input.BaseCurrency
	}
	return result, nil
}

func (r *adminRepo) UpdateOrganization(ctx context.Context, organizationID uuid.UUID, input *biz.AdminOrganization) (*biz.AdminOrganization, error) {
	update := r.data.db.Organization.UpdateOneID(input.ID).Where(organization.Or(organization.IDEQ(organizationID), organization.ParentIDEQ(organizationID))).SetName(input.Name).SetEnabled(input.Enabled)
	if input.Kind == biz.OrganizationKindHeadquarters || input.Kind == biz.OrganizationKindCompany {
		if err := validateOrganizationCurrency(ctx, r.data.db.Currency, input.BaseCurrency); err != nil {
			return nil, err
		}
		update.SetBaseCurrency(input.BaseCurrency)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminOrganizationNotFound
		}
		return nil, err
	}
	return r.organizationToBiz(ctx, updated)
}

func (r *adminRepo) ListUsers(ctx context.Context, organizationID uuid.UUID, options biz.AdminUserListOptions) (*biz.AdminUserList, error) {
	query := r.data.db.Membership.Query().
		Where(membership.OrganizationIDEQ(organizationID), membership.EnabledEQ(true)).
		WithUser().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() })
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Edges.User.Username < items[j].Edges.User.Username })
	keyword := strings.ToLower(options.Keyword)
	filtered := make([]*ent.Membership, 0, len(items))
	for _, item := range items {
		account, edgeErr := item.Edges.UserOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if keyword != "" && !strings.Contains(strings.ToLower(account.Username), keyword) && !strings.Contains(strings.ToLower(account.DisplayName), keyword) {
			continue
		}
		filtered = append(filtered, item)
	}
	total := len(filtered)
	start := (options.Page - 1) * options.PageSize
	if start > total {
		start = total
	}
	end := start + options.PageSize
	if end > total {
		end = total
	}
	result := make([]*biz.AdminUser, 0, end-start)
	for _, item := range filtered[start:end] {
		result = append(result, membershipToUser(item))
	}
	return &biz.AdminUserList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *adminRepo) CreateUser(ctx context.Context, organizationID uuid.UUID, input *biz.AdminUser, passwordHash string, roleIDs []uuid.UUID) (*biz.AdminUser, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := rolesForOrganization(ctx, tx.Role.Query(), organizationID, roleIDs)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	create := tx.User.Create().SetUsername(input.Username).SetDisplayName(input.DisplayName).SetPasswordHash(passwordHash).SetEnabled(input.Enabled)
	if input.Email != nil {
		create.SetEmail(*input.Email)
	}
	account, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrAdminUsernameExists
		}
		return nil, err
	}
	membershipRecord, err := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(organizationID).SetPrimary(false).SetEnabled(true).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replaceRoleAssignments(ctx, tx, membershipRecord.ID, roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findUser(ctx, organizationID, account.ID)
}

func (r *adminRepo) UpdateUser(ctx context.Context, organizationID, id uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID) (*biz.AdminUser, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	membershipRecord, err := tx.Membership.Query().Where(membership.UserIDEQ(id), membership.OrganizationIDEQ(organizationID), membership.EnabledEQ(true)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserNotFound
		}
		return nil, err
	}
	roles, err := rolesForOrganization(ctx, tx.Role.Query(), organizationID, roleIDs)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := tx.User.UpdateOneID(id).SetDisplayName(input.DisplayName).SetEnabled(input.Enabled)
	if input.Email == nil {
		update.ClearEmail()
	} else {
		update.SetEmail(*input.Email)
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserNotFound
		}
		return nil, err
	}
	if err := replaceRoleAssignments(ctx, tx, membershipRecord.ID, roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findUser(ctx, organizationID, id)
}

func (r *adminRepo) AuthorizeWeComUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID) (*biz.AdminUser, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	membershipRecord, err := tx.Membership.Query().Where(membership.UserIDEQ(input.ID), membership.OrganizationIDEQ(sourceOrganizationID), membership.EnabledEQ(true)).WithUser().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserNotFound
		}
		return nil, err
	}
	account, err := membershipRecord.Edges.UserOrErr()
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if account.WecomUserid == nil || account.Enabled {
		_ = tx.Rollback()
		return nil, biz.ErrAdminInvalidArgument
	}
	if exists, queryErr := tx.Organization.Query().Where(organization.IDEQ(targetOrganizationID), organization.EnabledEQ(true)).Exist(ctx); queryErr != nil {
		_ = tx.Rollback()
		return nil, queryErr
	} else if !exists {
		_ = tx.Rollback()
		return nil, biz.ErrAdminOrganizationNotFound
	}
	roles, err := rolesForOrganization(ctx, tx.Role.Query(), targetOrganizationID, roleIDs)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	userUpdate := tx.User.UpdateOneID(input.ID).SetDisplayName(input.DisplayName).SetEnabled(true)
	if input.Email == nil {
		userUpdate.ClearEmail()
	} else {
		userUpdate.SetEmail(*input.Email)
	}
	if _, err := userUpdate.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Membership.UpdateOneID(membershipRecord.ID).SetOrganizationID(targetOrganizationID).SetPrimary(true).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replaceRoleAssignments(ctx, tx, membershipRecord.ID, roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findUser(ctx, targetOrganizationID, input.ID)
}

func (r *adminRepo) ResetUserPassword(ctx context.Context, organizationID, id uuid.UUID, passwordHash string) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if exists, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(id), membership.OrganizationIDEQ(organizationID)).Exist(ctx); queryErr != nil {
		_ = tx.Rollback()
		return queryErr
	} else if !exists {
		_ = tx.Rollback()
		return biz.ErrAdminUserNotFound
	}
	if _, err := tx.User.UpdateOneID(id).SetPasswordHash(passwordHash).Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return biz.ErrAdminUserNotFound
		}
		return err
	}
	if _, err := tx.Session.Update().Where(sessionent.UserIDEQ(id), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

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

func (r *adminRepo) CreateRole(ctx context.Context, organizationID uuid.UUID, input *biz.AdminRole, permissionKeys []string) (*biz.AdminRole, error) {
	permissions, err := permissionsByKeys(ctx, r.data.db.Permission.Query(), permissionKeys)
	if err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := tx.Role.Create().SetOrganizationID(organizationID).SetCode(input.Code).SetName(input.Name).SetDataScope(role.DataScope(input.DataScope)).SetEnabled(input.Enabled).AddPermissions(permissions...).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrAdminRoleCodeExists
		}
		return nil, err
	}
	if err := replaceRoleOrderOrganizationAccesses(ctx, tx, created.ID, input.OrderOrganizationAccesses); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	created, err = r.data.db.Role.Query().Where(role.IDEQ(created.ID)).WithPermissions().WithOrderOrganizationAccesses().Only(ctx)
	if err != nil {
		return nil, err
	}
	return roleToBiz(created), nil
}

func (r *adminRepo) UpdateRole(ctx context.Context, organizationID, id uuid.UUID, input *biz.AdminRole, permissionKeys []string) (*biz.AdminRole, error) {
	permissions, err := permissionsByKeys(ctx, r.data.db.Permission.Query(), permissionKeys)
	if err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := tx.Role.UpdateOneID(id).Where(role.OrganizationIDEQ(organizationID)).SetName(input.Name).SetDataScope(role.DataScope(input.DataScope)).SetEnabled(input.Enabled).ClearPermissions().AddPermissions(permissions...).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminRoleNotFound
		}
		return nil, err
	}
	if err := replaceRoleOrderOrganizationAccesses(ctx, tx, updated.ID, input.OrderOrganizationAccesses); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
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
	sort.Slice(items, func(i, j int) bool {
		if items[i].Group == items[j].Group {
			return items[i].Key < items[j].Key
		}
		return items[i].Group < items[j].Group
	})
	result := make([]*biz.AdminPermission, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.AdminPermission{Key: item.Key, Name: item.Name, Group: item.Group, Description: item.Description})
	}
	return result, nil
}

func (r *adminRepo) ListAuditLogs(ctx context.Context, organizationID uuid.UUID, options biz.AdminAuditLogListOptions) (*biz.AdminAuditLogList, error) {
	query := r.data.db.AuditLog.Query().Where(auditlog.OrganizationIDEQ(organizationID))
	if options.Action != "" {
		query.Where(auditlog.ActionContains(options.Action))
	}
	if options.UserID != nil {
		query.Where(auditlog.UserIDEQ(*options.UserID))
	}
	if options.ResourceType != "" {
		query.Where(auditlog.ResourceTypeEQ(options.ResourceType))
	}
	if options.ResourceID != "" {
		query.Where(auditlog.ResourceIDEQ(options.ResourceID))
	}
	if options.StartTime != nil {
		query.Where(auditlog.CreatedAtGTE(*options.StartTime))
	}
	if options.EndTime != nil {
		query.Where(auditlog.CreatedAtLT(*options.EndTime))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(auditlog.ByCreatedAt(entsql.OrderDesc())).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AuditLog, 0, len(items))
	for _, item := range items {
		mapped, mapErr := auditLogToBiz(item)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, mapped)
	}
	return &biz.AdminAuditLogList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func auditLogToBiz(item *ent.AuditLog) (*biz.AuditLog, error) {
	details := make(map[string]string)
	if len(item.Details) > 0 {
		if err := json.Unmarshal(item.Details, &details); err != nil {
			return nil, err
		}
	}
	return &biz.AuditLog{
		ID: item.ID, OrganizationID: item.OrganizationID, UserID: item.UserID,
		Action: item.Action, ResourceType: optionalAuditString(item.ResourceType), ResourceID: optionalAuditString(item.ResourceID),
		Result: item.Result.String(), RequestID: item.RequestID, TraceID: item.TraceID, IPAddress: item.IPAddress,
		Details: details, CreatedAt: item.CreatedAt,
	}, nil
}

func optionalAuditString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (r *adminRepo) findUser(ctx context.Context, organizationID, userID uuid.UUID) (*biz.AdminUser, error) {
	item, err := r.data.db.Membership.Query().Where(membership.UserIDEQ(userID), membership.OrganizationIDEQ(organizationID)).WithUser().WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() }).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserNotFound
		}
		return nil, err
	}
	return membershipToUser(item), nil
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

func organizationToBiz(item *ent.Organization) *biz.AdminOrganization {
	result := &biz.AdminOrganization{ID: item.ID, Code: item.Code, Name: item.Name, Kind: biz.OrganizationKind(item.Kind), ParentID: item.ParentID, Enabled: item.Enabled}
	if item.BaseCurrency != nil {
		result.BaseCurrency = *item.BaseCurrency
	}
	return result
}

func organizationToBizWithCurrency(item *ent.Organization, items []*ent.Organization) (*biz.AdminOrganization, error) {
	result := organizationToBiz(item)
	if result.BaseCurrency != "" {
		return result, nil
	}
	byID := make(map[uuid.UUID]*ent.Organization, len(items))
	for _, candidate := range items {
		byID[candidate.ID] = candidate
	}
	current := item
	for current.ParentID != nil {
		parent, ok := byID[*current.ParentID]
		if !ok {
			return nil, biz.ErrAdminOrganizationCurrency
		}
		if parent.BaseCurrency != nil {
			result.BaseCurrency = *parent.BaseCurrency
			return result, nil
		}
		current = parent
	}
	return nil, biz.ErrAdminOrganizationCurrency
}

func (r *adminRepo) organizationToBiz(ctx context.Context, item *ent.Organization) (*biz.AdminOrganization, error) {
	result := organizationToBiz(item)
	current := item
	for result.BaseCurrency == "" && current.ParentID != nil {
		parent, err := r.data.db.Organization.Get(ctx, *current.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.BaseCurrency != nil {
			result.BaseCurrency = *parent.BaseCurrency
		}
		current = parent
	}
	if result.BaseCurrency == "" {
		return nil, biz.ErrAdminOrganizationCurrency
	}
	return result, nil
}

type currencyQuery interface {
	Query() *ent.CurrencyQuery
}

func validateOrganizationCurrency(ctx context.Context, client currencyQuery, code string) error {
	exists, err := client.Query().Where(currencyent.CodeEQ(code), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrAdminOrganizationCurrency
	}
	return nil
}

func membershipToUser(item *ent.Membership) *biz.AdminUser {
	account := item.Edges.User
	result := &biz.AdminUser{ID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, WeComUserID: account.WecomUserid, WeComName: account.WecomName, Enabled: account.Enabled, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt}
	for _, assignment := range item.Edges.RoleAssignments {
		if assignedRole := assignment.Edges.Role; assignedRole != nil {
			result.RoleIDs = append(result.RoleIDs, assignedRole.ID)
			result.RoleCodes = append(result.RoleCodes, assignedRole.Code)
		}
	}
	sort.Strings(result.RoleCodes)
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

var _ biz.AdminRepo = (*adminRepo)(nil)
