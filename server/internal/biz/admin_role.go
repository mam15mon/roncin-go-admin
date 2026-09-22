package biz

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/access"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminRoleScopeDisabled   = errors.BadRequest("ADMIN_ROLE_SCOPE_DISABLED", "仅本人数据范围已停用，请明确选择其他数据范围")
	ErrAdminRoleNotFound        = errors.NotFound("ADMIN_ROLE_NOT_FOUND", "角色不存在")
	ErrAdminRoleCodeExists      = errors.Conflict("ADMIN_ROLE_CODE_EXISTS", "角色编码已存在")
	ErrAdminRoleAnchorInvalid   = errors.BadRequest("ADMIN_ROLE_ANCHOR_INVALID", "角色只能在公司/系统管理维护")
	ErrAdminPermissionInvalid   = errors.BadRequest("ADMIN_PERMISSION_INVALID", "权限不存在或不属于当前请求")
	ErrAdminPrivilegeEscalation = errors.Forbidden("ADMIN_PRIVILEGE_ESCALATION_DENIED", "不能分配超出自身权限范围的角色")
	ErrAdminRoleProtected       = errors.Forbidden("ADMIN_ROLE_PROTECTED", "系统管理员角色不允许删除")
	ErrAdminRoleAssigned        = errors.Conflict("ADMIN_ROLE_ASSIGNED", "该角色已分配成员，请先移除后重试")
	ErrAdminRoleInUse           = errors.Conflict("ADMIN_ROLE_IN_USE", "该角色仍被其他数据引用，无法删除")
)

type AdminRole struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	DataScope      DataScope
	Enabled        bool
	PermissionKeys []string
	// AssignmentsCount 是角色当前被分配到的成员关系数量，仅作展示与删除前置判断；
	// 删除时仓储在事务内重新复核，防止校验与删除之间新增分配。
	AssignmentsCount int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AdminPermission struct {
	Key         string
	Name        string
	Group       string
	Description string
	Requires    []string
}
type AdminPrivilegeProfile struct {
	IsSuperAdmin bool
	// RoleProfiles 保留权限与数据范围的角色来源关系。角色管理中的提权校验
	// 必须与运行时组织范围解析一致，不能把一个角色的权限和另一个角色的
	// 数据范围拼成新的授权能力。
	RoleProfiles []AdminRoleProfile
}

type AdminRoleProfile struct {
	ID             uuid.UUID
	Code           string
	DataScope      DataScope
	PermissionKeys []string
}

func (uc *AdminUsecase) validateRolesPrivilege(ctx context.Context, actorOrganizationID, actorID, targetOrganizationID uuid.UUID, roleIDs []uuid.UUID) error {
	if actorOrganizationID == uuid.Nil || actorID == uuid.Nil || targetOrganizationID == uuid.Nil {
		return ErrAdminPrivilegeEscalation
	}
	if principal, ok := PrincipalFromContext(ctx); ok && !principalIsSystemWorkspace(principal) {
		allowed := principal.baseOrganizationIDs(DataScopeOrganizationTree, principal.organizationScopeNodes())
		if !containsOrganizationID(allowed, targetOrganizationID) {
			return ErrAdminPrivilegeEscalation
		}
	}
	if len(roleIDs) == 0 {
		return nil
	}
	profile, err := uc.getActorPrivilegeProfile(ctx, actorOrganizationID, actorID)
	if err != nil {
		return err
	}
	roleProfiles, err := uc.repo.GetRolesPrivilegeProfiles(ctx, targetOrganizationID, roleIDs)
	if err != nil {
		return err
	}
	if principal, ok := PrincipalFromContext(ctx); ok && principalIsSystemWorkspace(principal) {
		for _, node := range principal.OrganizationNodes {
			if node.ID == targetOrganizationID && node.Kind != OrganizationKindSystem && (principal.HasPermission(access.UserUpdate) || principal.HasPermission(access.UserCreate)) {
				profile.IsSuperAdmin = true
			}
		}
	}
	for _, roleProfile := range roleProfiles {
		if err := checkPrivilegeEscalation(profile, roleProfile.DataScope, roleProfile.PermissionKeys, roleProfile.Code == "administrator"); err != nil {
			return err
		}
	}
	return nil
}

func (uc *AdminUsecase) getActorPrivilegeProfile(ctx context.Context, organizationID, actorID uuid.UUID) (*AdminPrivilegeProfile, error) {
	roles, err := uc.repo.GetActorRolesPrivilegeProfiles(ctx, organizationID, actorID)
	if err != nil {
		return nil, err
	}
	profile := &AdminPrivilegeProfile{
		RoleProfiles: make([]AdminRoleProfile, 0, len(roles)),
	}
	for _, role := range roles {
		if role == nil {
			continue
		}
		if role.Code == "administrator" && role.DataScope.active() {
			profile.IsSuperAdmin = true
		}
		profile.RoleProfiles = append(profile.RoleProfiles, *role)
	}
	return profile, nil
}

func checkPrivilegeEscalation(profile *AdminPrivilegeProfile, targetDataScope DataScope, permissionKeys []string, isAdministratorRole bool) error {
	if profile == nil {
		return ErrAdminPrivilegeEscalation
	}
	if profile.IsSuperAdmin {
		return nil
	}
	if isAdministratorRole {
		return ErrAdminPrivilegeEscalation
	}
	// 空权限角色保留为无效配置，后续新增权限时会在本函数按该权限的来源角色
	// 重新校验范围。
	for _, key := range permissionKeys {
		matchingRoles := actorRolesWithPermission(profile.RoleProfiles, key)
		if len(matchingRoles) == 0 || !anyRoleHasDataScope(matchingRoles, targetDataScope) {
			return ErrAdminPrivilegeEscalation
		}
	}
	return nil
}

func actorRolesWithPermission(roles []AdminRoleProfile, permission string) []AdminRoleProfile {
	matching := make([]AdminRoleProfile, 0, len(roles))
	for _, role := range roles {
		if !role.DataScope.active() {
			continue
		}
		for _, key := range role.PermissionKeys {
			if key == permission {
				matching = append(matching, role)
				break
			}
		}
	}
	return matching
}

func anyRoleHasDataScope(roles []AdminRoleProfile, target DataScope) bool {
	for _, role := range roles {
		if role.DataScope.rank() >= target.rank() {
			return true
		}
	}
	return false
}
func (uc *AdminUsecase) ListRoles(ctx context.Context, organizationID uuid.UUID) ([]*AdminRole, error) {
	if organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListRoles(ctx, organizationID)
}

func (uc *AdminUsecase) CreateRole(ctx context.Context, organizationID, actorID uuid.UUID, input *AdminRole, permissionKeys []string) (*AdminRole, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	normalized, err := normalizeRole(input)
	if err != nil {
		return nil, err
	}
	if normalized.Code == "" {
		generated, generateErr := generateRoleCode()
		if generateErr != nil {
			return nil, generateErr
		}
		normalized.Code = generated
	}
	profile, err := uc.getActorPrivilegeProfile(ctx, organizationID, actorID)
	if err != nil {
		return nil, err
	}
	granted := access.ResolveDependencies(normalizeKeys(permissionKeys))
	if err := checkPrivilegeEscalation(profile, normalized.DataScope, granted, normalized.Code == "administrator"); err != nil {
		return nil, err
	}
	return uc.repo.CreateRole(ctx, organizationID, normalized, granted, adminAuditEvent(ctx, actorID, nil, "admin.role.create", normalized.Code))
}

func (uc *AdminUsecase) UpdateRole(ctx context.Context, organizationID, actorID, id uuid.UUID, input *AdminRole, permissionKeys []string) (*AdminRole, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	normalized, err := normalizeRole(input)
	if err != nil {
		return nil, err
	}
	currentRole, err := uc.repo.GetRole(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	profile, err := uc.getActorPrivilegeProfile(ctx, organizationID, actorID)
	if err != nil {
		return nil, err
	}
	if currentRole.Code == "administrator" && !profile.IsSuperAdmin {
		return nil, ErrAdminPrivilegeEscalation
	}
	granted := access.ResolveDependencies(normalizeKeys(permissionKeys))
	if err := checkPrivilegeEscalation(profile, normalized.DataScope, granted, normalized.Code == "administrator"); err != nil {
		return nil, err
	}
	return uc.repo.UpdateRole(ctx, organizationID, id, normalized, granted, adminAuditEvent(ctx, actorID, &id, "admin.role.update", currentRole.Code))
}

// DeleteRole 删除当前组织内未被任何成员关系引用的角色：系统内置 administrator 角色
// 拒绝删除；已分配成员的角色必须先移除分配。成员关系复核在仓储事务内重新执行，
// 防止业务校验与删除之间出现并发分配。
func (uc *AdminUsecase) DeleteRole(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	currentRole, err := uc.repo.GetRole(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if currentRole.Code == "administrator" {
		return ErrAdminRoleProtected
	}
	if currentRole.AssignmentsCount > 0 {
		return ErrAdminRoleAssigned
	}
	return uc.repo.DeleteRole(ctx, organizationID, id, adminAuditEvent(ctx, actorID, &id, "admin.role.delete", currentRole.Code))
}

func (uc *AdminUsecase) ListPermissions(ctx context.Context) ([]*AdminPermission, error) {
	return uc.repo.ListPermissions(ctx)
}
func normalizeRole(input *AdminRole) (*AdminRole, error) {
	if input == nil {
		return nil, ErrAdminInvalidArgument
	}
	if input.DataScope == DataScopeSelf {
		return nil, ErrAdminRoleScopeDisabled
	}
	output := *input
	output.Code = strings.ToLower(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	if output.Name == "" || !output.DataScope.Valid() {
		return nil, ErrAdminInvalidArgument
	}
	return &output, nil
}

// generateRoleCode 为未指定编码的角色生成机器标识，仅用于审计与调试对照；
// 组织内唯一性由 (organization_id, code) 唯一索引兜底。
func generateRoleCode() (string, error) {
	const alphabet = "23456789abcdefghjkmnpqrstuvwxyz"
	raw := make([]byte, 10)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate role code: %w", err)
	}
	out := make([]byte, len(raw))
	for index, value := range raw {
		out[index] = alphabet[int(value)%len(alphabet)]
	}
	return "role_" + string(out), nil
}

func (scope DataScope) Valid() bool {
	return scope == DataScopeAll || scope == DataScopeOrganization || scope == DataScopeOrganizationTree || scope == DataScopeSelf
}

func normalizeKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}
