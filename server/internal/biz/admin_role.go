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
	ErrAdminRoleNotFound        = errors.NotFound("ADMIN_ROLE_NOT_FOUND", "角色不存在")
	ErrAdminRoleCodeExists      = errors.Conflict("ADMIN_ROLE_CODE_EXISTS", "角色编码已存在")
	ErrAdminPermissionInvalid   = errors.BadRequest("ADMIN_PERMISSION_INVALID", "权限不存在或不属于当前请求")
	ErrAdminPrivilegeEscalation = errors.Forbidden("ADMIN_PRIVILEGE_ESCALATION_DENIED", "不能分配超出自身权限范围的角色")
)

type AdminRole struct {
	ID                        uuid.UUID
	OrganizationID            uuid.UUID
	Code                      string
	Name                      string
	DataScope                 DataScope
	Enabled                   bool
	PermissionKeys            []string
	OrderOrganizationAccesses []OrderOrganizationAccess
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type OrderOrganizationAccess struct {
	OrganizationID uuid.UUID
	Writable       bool
}

type AdminPermission struct {
	Key         string
	Name        string
	Group       string
	Description string
	Requires    []string
}
type AdminPrivilegeProfile struct {
	IsSuperAdmin              bool
	Permissions               map[string]DataScope
	OrderOrganizationAccesses map[uuid.UUID]bool
}

type AdminRoleProfile struct {
	ID                        uuid.UUID
	Code                      string
	DataScope                 DataScope
	PermissionKeys            []string
	OrderOrganizationAccesses []OrderOrganizationAccess
}

func (uc *AdminUsecase) validateRolesPrivilege(ctx context.Context, actorOrganizationID, actorID, targetOrganizationID uuid.UUID, roleIDs []uuid.UUID) error {
	if actorOrganizationID == uuid.Nil || actorID == uuid.Nil || targetOrganizationID == uuid.Nil {
		return ErrAdminPrivilegeEscalation
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
	for _, roleProfile := range roleProfiles {
		if err := checkPrivilegeEscalation(profile, roleProfile.DataScope, roleProfile.PermissionKeys, roleProfile.OrderOrganizationAccesses, roleProfile.Code == "administrator"); err != nil {
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
		Permissions:               make(map[string]DataScope),
		OrderOrganizationAccesses: make(map[uuid.UUID]bool),
	}
	for _, role := range roles {
		if role == nil {
			continue
		}
		if role.Code == "administrator" {
			profile.IsSuperAdmin = true
		}
		for _, key := range role.PermissionKeys {
			currentScope, exists := profile.Permissions[key]
			if !exists || role.DataScope.rank() > currentScope.rank() {
				profile.Permissions[key] = role.DataScope
			}
		}
		for _, access := range role.OrderOrganizationAccesses {
			profile.OrderOrganizationAccesses[access.OrganizationID] = profile.OrderOrganizationAccesses[access.OrganizationID] || access.Writable
		}
	}
	return profile, nil
}

func checkPrivilegeEscalation(profile *AdminPrivilegeProfile, targetDataScope DataScope, permissionKeys []string, accesses []OrderOrganizationAccess, isAdministratorRole bool) error {
	if profile == nil {
		return ErrAdminPrivilegeEscalation
	}
	if profile.IsSuperAdmin {
		return nil
	}
	if isAdministratorRole {
		return ErrAdminPrivilegeEscalation
	}
	for _, key := range permissionKeys {
		scope, ok := profile.Permissions[key]
		if !ok {
			return ErrAdminPrivilegeEscalation
		}
		if targetDataScope.rank() > scope.rank() {
			return ErrAdminPrivilegeEscalation
		}
	}
	for _, access := range accesses {
		writable, ok := profile.OrderOrganizationAccesses[access.OrganizationID]
		if !ok {
			return ErrAdminPrivilegeEscalation
		}
		if access.Writable && !writable {
			return ErrAdminPrivilegeEscalation
		}
	}
	return nil
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
	if err := uc.validateOrderOrganizationAccesses(ctx, organizationID, normalized.OrderOrganizationAccesses); err != nil {
		return nil, err
	}
	profile, err := uc.getActorPrivilegeProfile(ctx, organizationID, actorID)
	if err != nil {
		return nil, err
	}
	granted := access.ResolveDependencies(normalizeKeys(permissionKeys))
	if err := checkPrivilegeEscalation(profile, normalized.DataScope, granted, normalized.OrderOrganizationAccesses, normalized.Code == "administrator"); err != nil {
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
	if err := uc.validateOrderOrganizationAccesses(ctx, organizationID, normalized.OrderOrganizationAccesses); err != nil {
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
	if err := checkPrivilegeEscalation(profile, normalized.DataScope, granted, normalized.OrderOrganizationAccesses, normalized.Code == "administrator"); err != nil {
		return nil, err
	}
	return uc.repo.UpdateRole(ctx, organizationID, id, normalized, granted, adminAuditEvent(ctx, actorID, &id, "admin.role.update", currentRole.Code))
}

func (uc *AdminUsecase) ListPermissions(ctx context.Context) ([]*AdminPermission, error) {
	return uc.repo.ListPermissions(ctx)
}
func normalizeRole(input *AdminRole) (*AdminRole, error) {
	if input == nil {
		return nil, ErrAdminInvalidArgument
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

func (uc *AdminUsecase) validateOrderOrganizationAccesses(ctx context.Context, sourceOrganizationID uuid.UUID, accesses []OrderOrganizationAccess) error {
	seen := make(map[uuid.UUID]struct{}, len(accesses))
	for _, access := range accesses {
		if access.OrganizationID == uuid.Nil || access.OrganizationID == sourceOrganizationID {
			return ErrAdminInvalidArgument
		}
		if _, ok := seen[access.OrganizationID]; ok {
			return ErrAdminInvalidArgument
		}
		seen[access.OrganizationID] = struct{}{}
		organization, err := uc.repo.GetOrganization(ctx, access.OrganizationID)
		if err != nil || !organization.Enabled || organization.Kind != OrganizationKindCompany {
			return ErrAdminInvalidArgument
		}
	}
	return nil
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
