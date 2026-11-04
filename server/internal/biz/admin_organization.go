package biz

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrAdminBootstrapSystemRequired           = errors.Conflict("ADMIN_BOOTSTRAP_SYSTEM_REQUIRED", "初始管理员系统资格缺失，请先运行 bootstrap:admin 补齐后再创建公司")
	ErrAdminBootstrapIdentityInvalid          = errors.Conflict("ADMIN_BOOTSTRAP_IDENTITY_INVALID", "必须存在唯一初始管理员，请先核对管理员初始化状态")
	ErrAdminBootstrapInactive                 = errors.Conflict("ADMIN_BOOTSTRAP_INACTIVE", "初始管理员账号已停用或待分配，请人工恢复后重跑")
	ErrAdminBootstrapSystemInactive           = errors.Conflict("ADMIN_BOOTSTRAP_SYSTEM_INACTIVE", "系统工作台或其管理员资格不可用，请人工恢复或核对系统工作台配置")
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
	NameEn       string
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

func (uc *AdminUsecase) UpdateOrganization(ctx context.Context, userID, organizationID, id uuid.UUID, name string, enabled bool, baseCurrency, nameEn string) (*AdminOrganization, error) {
	name = strings.TrimSpace(name)
	if organizationID == uuid.Nil || id == uuid.Nil || name == "" {
		return nil, ErrAdminInvalidArgument
	}
	current, err := uc.repo.GetOrganization(ctx, id)
	if err != nil {
		return nil, err
	}
	nameEn, err = normalizeCompanyNameEn(current.Kind, nameEn)
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
	return uc.repo.UpdateOrganization(ctx, organizationID, &AdminOrganization{ID: id, Name: name, NameEn: nameEn, Enabled: enabled, Kind: current.Kind, BaseCurrency: baseCurrency}, adminAuditEvent(ctx, userID, &id, "admin.organization.update", current.Code))
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
	var err error
	output.NameEn, err = normalizeCompanyNameEn(output.Kind, output.NameEn)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

// normalizeCompanyNameEn 只规范首尾空白，不猜译或改写正式名称。
func normalizeCompanyNameEn(kind OrganizationKind, value string) (string, error) {
	value = strings.TrimSpace(value)
	if kind != OrganizationKindCompany && value != "" {
		return "", errors.BadRequest("ADMIN_ORGANIZATION_NAME_EN_INVALID", "仅公司可维护英文名称")
	}
	if utf8.RuneCountInString(value) > 200 || strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsControl(r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
	}) {
		return "", errors.BadRequest("ADMIN_ORGANIZATION_NAME_EN_INVALID", "公司英文名称不能超过200个字符，且不能包含换行或控制字符")
	}
	return value, nil
}
func validOrganizationCurrency(value string) bool {
	return currencyPattern.MatchString(value)
}
