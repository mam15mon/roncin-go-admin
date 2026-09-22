package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminOrganizationNotFound              = errors.NotFound("ADMIN_ORGANIZATION_NOT_FOUND", "组织不存在")
	ErrAdminOrganizationCodeExists            = errors.Conflict("ADMIN_ORGANIZATION_CODE_EXISTS", "组织编码已存在")
	ErrAdminOrganizationParentRequired        = errors.BadRequest("ADMIN_ORGANIZATION_PARENT_REQUIRED", "新建组织必须指定上级组织")
	ErrAdminOrganizationHierarchy             = errors.BadRequest("ADMIN_ORGANIZATION_HIERARCHY_INVALID", "组织层级不合法")
	ErrAdminOrganizationCurrency              = errors.BadRequest("ADMIN_ORGANIZATION_CURRENCY_INVALID", "组织本币必须是启用的 ISO 币种")
	ErrAdminOrganizationBaseCurrencyImmutable = errors.BadRequest("ADMIN_ORGANIZATION_BASE_CURRENCY_IMMUTABLE", "组织本币一旦设定不可变更")
)

type OrganizationKind string

const (
	OrganizationKindSystem     OrganizationKind = "system"
	OrganizationKindCompany    OrganizationKind = "company"
	OrganizationKindDepartment OrganizationKind = "department"
	OrganizationKindTeam       OrganizationKind = "team"
)

func (kind OrganizationKind) Valid() bool {
	return kind == OrganizationKindSystem || kind == OrganizationKindCompany || kind == OrganizationKindDepartment || kind == OrganizationKindTeam
}

type AdminOrganization struct {
	ID           uuid.UUID
	Code         string
	Name         string
	Kind         OrganizationKind
	ParentID     *uuid.UUID
	Enabled      bool
	BaseCurrency string
}

func (uc *AdminUsecase) ListOrganizations(ctx context.Context, organizationID uuid.UUID) ([]*AdminOrganization, error) {
	if organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListOrganizations(ctx)
}

func (uc *AdminUsecase) CreateOrganization(ctx context.Context, userID uuid.UUID, input *AdminOrganization) (*AdminOrganization, error) {
	normalized, err := normalizeOrganization(input)
	if err != nil {
		return nil, err
	}
	if normalized.Kind == OrganizationKindCompany {
		if normalized.ParentID != nil {
			return nil, ErrAdminOrganizationHierarchy
		}
		if principal, ok := PrincipalFromContext(ctx); ok && !principalIsSystemWorkspace(principal) {
			return nil, ErrPermissionDenied
		}
		if !validOrganizationCurrency(normalized.BaseCurrency) {
			return nil, ErrAdminOrganizationCurrency
		}
	} else {
		if normalized.ParentID == nil {
			return nil, ErrAdminOrganizationParentRequired
		}
		if *normalized.ParentID == uuid.Nil {
			return nil, ErrAdminInvalidArgument
		}
		if principal, ok := PrincipalFromContext(ctx); ok && !principalIsSystemWorkspace(principal) {
			if *normalized.ParentID != principal.Organization.ID && !containsOrganizationID(principal.baseOrganizationIDs(DataScopeOrganizationTree, principal.organizationScopeNodes()), *normalized.ParentID) {
				return nil, ErrPermissionDenied
			}
		}
		parent, err := uc.repo.GetOrganization(ctx, *normalized.ParentID)
		if err != nil {
			return nil, err
		}
		if !(parent.Kind == OrganizationKindCompany && normalized.Kind == OrganizationKindDepartment || parent.Kind == OrganizationKindDepartment && normalized.Kind == OrganizationKindTeam) {
			return nil, ErrAdminOrganizationHierarchy
		}
		if normalized.BaseCurrency != "" {
			return nil, ErrAdminOrganizationCurrency
		}
		normalized.BaseCurrency = parent.BaseCurrency
	}

	return uc.repo.CreateOrganization(ctx, normalized, adminAuditEvent(ctx, userID, nil, "admin.organization.create", ""))
}

func (uc *AdminUsecase) UpdateOrganization(ctx context.Context, userID, organizationID, id uuid.UUID, name string, enabled bool, baseCurrency string) (*AdminOrganization, error) {
	name = strings.TrimSpace(name)
	if organizationID == uuid.Nil || id == uuid.Nil || name == "" {
		return nil, ErrAdminInvalidArgument
	}
	current, err := uc.repo.GetOrganization(ctx, id)
	if err != nil {
		return nil, err
	}
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if current.Kind == OrganizationKindSystem || current.Kind == OrganizationKindCompany {
		if !validOrganizationCurrency(baseCurrency) {
			return nil, ErrAdminOrganizationCurrency
		}
		if current.BaseCurrency != "" && current.BaseCurrency != baseCurrency {
			return nil, ErrAdminOrganizationBaseCurrencyImmutable
		}
	} else if baseCurrency != "" {
		return nil, ErrAdminOrganizationCurrency
	}
	return uc.repo.UpdateOrganization(ctx, organizationID, &AdminOrganization{ID: id, Name: name, Enabled: enabled, Kind: current.Kind, BaseCurrency: baseCurrency}, adminAuditEvent(ctx, userID, &id, "admin.organization.update", current.Code))
}
func normalizeOrganization(input *AdminOrganization) (*AdminOrganization, error) {
	if input == nil {
		return nil, ErrAdminInvalidArgument
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	output.BaseCurrency = strings.ToUpper(strings.TrimSpace(output.BaseCurrency))
	if output.Code == "" || output.Name == "" || !output.Kind.Valid() || output.Kind == OrganizationKindSystem {
		return nil, ErrAdminInvalidArgument
	}
	return &output, nil
}
func validOrganizationCurrency(value string) bool {
	return currencyPattern.MatchString(value)
}
