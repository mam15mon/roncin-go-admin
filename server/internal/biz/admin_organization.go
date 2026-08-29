package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminOrganizationNotFound       = errors.NotFound("ADMIN_ORGANIZATION_NOT_FOUND", "组织不存在")
	ErrAdminOrganizationCodeExists     = errors.Conflict("ADMIN_ORGANIZATION_CODE_EXISTS", "组织编码已存在")
	ErrAdminOrganizationParentRequired = errors.BadRequest("ADMIN_ORGANIZATION_PARENT_REQUIRED", "新建组织必须指定上级组织")
	ErrAdminOrganizationHierarchy      = errors.BadRequest("ADMIN_ORGANIZATION_HIERARCHY_INVALID", "组织层级不合法")
	ErrAdminOrganizationCurrency       = errors.BadRequest("ADMIN_ORGANIZATION_CURRENCY_INVALID", "组织本币必须是启用的 ISO 币种")
)

type OrganizationKind string

const (
	OrganizationKindHeadquarters OrganizationKind = "headquarters"
	OrganizationKindCompany      OrganizationKind = "company"
	OrganizationKindDepartment   OrganizationKind = "department"
	OrganizationKindTeam         OrganizationKind = "team"
)

func (kind OrganizationKind) Valid() bool {
	return kind == OrganizationKindHeadquarters || kind == OrganizationKindCompany || kind == OrganizationKindDepartment || kind == OrganizationKindTeam
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
	if normalized.ParentID == nil {
		return nil, ErrAdminOrganizationParentRequired
	}
	if *normalized.ParentID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	parent, err := uc.repo.GetOrganization(ctx, *normalized.ParentID)
	if err != nil {
		return nil, err
	}
	if parent.Kind == OrganizationKindHeadquarters && normalized.Kind != OrganizationKindCompany ||
		parent.Kind == OrganizationKindCompany && normalized.Kind != OrganizationKindDepartment ||
		parent.Kind == OrganizationKindDepartment && normalized.Kind != OrganizationKindTeam ||
		parent.Kind == OrganizationKindTeam {
		return nil, ErrAdminOrganizationHierarchy
	}
	if normalized.Kind == OrganizationKindCompany {
		if !validOrganizationCurrency(normalized.BaseCurrency) {
			return nil, ErrAdminOrganizationCurrency
		}
	} else {
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
	if current.Kind == OrganizationKindHeadquarters || current.Kind == OrganizationKindCompany {
		if !validOrganizationCurrency(baseCurrency) {
			return nil, ErrAdminOrganizationCurrency
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
	if output.Code == "" || output.Name == "" || !output.Kind.Valid() || output.Kind == OrganizationKindHeadquarters {
		return nil, ErrAdminInvalidArgument
	}
	return &output, nil
}
func validOrganizationCurrency(value string) bool {
	return currencyPattern.MatchString(value)
}
