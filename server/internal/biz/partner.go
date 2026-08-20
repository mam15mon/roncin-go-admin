package biz

import (
	"context"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerNotFound                = errors.NotFound(v1.ErrorReason_PARTNER_NOT_FOUND.String(), "往来单位不存在")
	ErrPartnerCodeExists              = errors.Conflict(v1.ErrorReason_PARTNER_CODE_EXISTS.String(), "往来单位编码已存在")
	ErrPartnerNameExists              = errors.Conflict(v1.ErrorReason_PARTNER_NAME_EXISTS.String(), "往来单位名称已存在")
	ErrPartnerUSCCExists              = errors.Conflict(v1.ErrorReason_PARTNER_USCC_EXISTS.String(), "统一社会信用代码已存在")
	ErrPartnerRoleRequired            = errors.BadRequest(v1.ErrorReason_PARTNER_ROLE_REQUIRED.String(), "往来单位至少需要一个有效角色")
	ErrPartnerInvalidRole             = errors.BadRequest(v1.ErrorReason_PARTNER_INVALID_ROLE.String(), "往来单位角色不合法")
	ErrPartnerInvalidArgument         = errors.BadRequest(v1.ErrorReason_PARTNER_INVALID_ARGUMENT.String(), "往来单位字段不合法")
	ErrPartnerPrimaryContactConflict  = errors.BadRequest(v1.ErrorReason_PARTNER_PRIMARY_CONTACT_CONFLICT.String(), "只能设置一个主联系人")
	ErrPartnerAliasExists             = errors.Conflict(v1.ErrorReason_PARTNER_ALIAS_EXISTS.String(), "往来单位别名重复")
	ErrPartnerSupplierRoleRequired    = errors.BadRequest(v1.ErrorReason_PARTNER_SUPPLIER_ROLE_REQUIRED.String(), "往来单位没有供应商角色")
	ErrPartnerBlacklistReasonRequired = errors.BadRequest(v1.ErrorReason_PARTNER_BLACKLIST_REASON_REQUIRED.String(), "黑名单变更原因不能为空")
	ErrPartnerBlacklistedSupplierRole = errors.BadRequest(v1.ErrorReason_PARTNER_BLACKLISTED_SUPPLIER_ROLE.String(), "清除黑名单前不能移除供应商角色")
	ErrPartnerImportInvalidArgument   = errors.BadRequest(v1.ErrorReason_PARTNER_INVALID_ARGUMENT.String(), "往来单位导入参数不合法")
)

type PartnerRoleType string

const (
	PartnerRoleCustomer PartnerRoleType = "customer"
	PartnerRoleSupplier PartnerRoleType = "supplier"
	PartnerRoleAgent    PartnerRoleType = "agent"
	PartnerRoleCarrier  PartnerRoleType = "carrier"
)

func (t PartnerRoleType) Valid() bool {
	return t == PartnerRoleCustomer || t == PartnerRoleSupplier || t == PartnerRoleAgent || t == PartnerRoleCarrier
}

type PartnerRole struct {
	Type            PartnerRoleType
	Enabled         bool
	Blacklisted     bool
	BlacklistReason string
	BlacklistedAt   *time.Time
	BlacklistedBy   *uuid.UUID
}

type PartnerContact struct {
	ID        uuid.UUID
	Name      string
	Phone     string
	Email     string
	Note      string
	IsPrimary bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PartnerAlias struct {
	ID                  uuid.UUID
	AliasName           string
	NormalizedAliasName string
	SortOrder           int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Partner struct {
	ID                      uuid.UUID
	OrganizationID          uuid.UUID
	Code                    string
	LegalName               string
	NormalizedName          string
	UnifiedSocialCreditCode string
	RegisteredAddress       string
	Enabled                 bool
	Roles                   []*PartnerRole
	Contacts                []*PartnerContact
	Aliases                 []*PartnerAlias
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type PartnerListOptions struct {
	Page     int
	PageSize int
	Keyword  string
	Role     PartnerRoleType
	Enabled  *bool
}

type PartnerList struct {
	Items    []*Partner
	Total    int
	Page     int
	PageSize int
}

type PartnerImportMode string

const (
	PartnerImportCreateOnly PartnerImportMode = "create_only"
	PartnerImportUpsert     PartnerImportMode = "upsert"
)

func (mode PartnerImportMode) Valid() bool {
	return mode == PartnerImportCreateOnly || mode == PartnerImportUpsert
}

type PartnerImportInput struct {
	Source string
	Mode   PartnerImportMode
	Items  []*Partner
}

type PartnerImportResult struct {
	CreatedCount int
	UpdatedCount int
}

type PartnerUpdateResult struct {
	Partner       *Partner
	PreviousRoles []*PartnerRole
}

type PartnerBlacklistUpdate struct {
	Blacklisted bool
	Reason      string
	ChangedAt   time.Time
	ChangedBy   uuid.UUID
}

type PartnerBlacklistResult struct {
	Partner               *Partner
	PreviouslyBlacklisted bool
}

type PartnerRepo interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*Partner, error)
	List(context.Context, uuid.UUID, PartnerListOptions) (*PartnerList, error)
	Create(context.Context, uuid.UUID, *Partner) (*Partner, error)
	Update(context.Context, uuid.UUID, uuid.UUID, *Partner) (*PartnerUpdateResult, error)
	SetSupplierBlacklist(context.Context, uuid.UUID, uuid.UUID, PartnerBlacklistUpdate) (*PartnerBlacklistResult, error)
	Import(context.Context, uuid.UUID, PartnerImportMode, []*Partner) (*PartnerImportResult, error)
}

type PartnerUsecase struct {
	repo  PartnerRepo
	audit AuditRepo
	now   func() time.Time
}

func NewPartnerUsecase(repo PartnerRepo, audit AuditRepo) *PartnerUsecase {
	return &PartnerUsecase{repo: repo, audit: audit, now: time.Now}
}

func (uc *PartnerUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*Partner, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrPartnerNotFound
	}
	return uc.repo.Get(ctx, organizationID, id)
}

func (uc *PartnerUsecase) List(ctx context.Context, organizationID uuid.UUID, options PartnerListOptions) (*PartnerList, error) {
	if organizationID == uuid.Nil || options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 {
		return nil, ErrPartnerInvalidArgument
	}
	if options.Role != "" && !options.Role.Valid() {
		return nil, ErrPartnerInvalidRole
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *PartnerUsecase) Import(ctx context.Context, organizationID, userID uuid.UUID, input PartnerImportInput) (*PartnerImportResult, error) {
	if organizationID == uuid.Nil || userID == uuid.Nil || strings.TrimSpace(input.Source) == "" || !input.Mode.Valid() || len(input.Items) == 0 || len(input.Items) > 500 {
		return nil, ErrPartnerImportInvalidArgument
	}
	normalized := make([]*Partner, 0, len(input.Items))
	seenCodes := make(map[string]struct{}, len(input.Items))
	for _, item := range input.Items {
		value, err := normalizePartner(item, true)
		if err != nil {
			return nil, err
		}
		if _, exists := seenCodes[value.Code]; exists {
			return nil, ErrPartnerImportInvalidArgument
		}
		seenCodes[value.Code] = struct{}{}
		normalized = append(normalized, value)
	}
	result, err := uc.repo.Import(ctx, organizationID, input.Mode, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.import",
		Result:         "success",
		Details: map[string]string{
			"source":        input.Source,
			"mode":          string(input.Mode),
			"created_count": fmt.Sprintf("%d", result.CreatedCount),
			"updated_count": fmt.Sprintf("%d", result.UpdatedCount),
		},
	}); err != nil {
		return nil, fmt.Errorf("write partner import audit: %w", err)
	}
	return result, nil
}

func (uc *PartnerUsecase) Create(ctx context.Context, organizationID, userID uuid.UUID, input *Partner) (*Partner, error) {
	normalized, err := normalizePartner(input, true)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.create",
		Result:         "success",
		Details: map[string]string{
			"partner.id":   created.ID.String(),
			"partner.code": created.Code,
			"legal_name":   created.LegalName,
			"roles":        partnerRolesAuditValue(created.Roles),
		},
	}); err != nil {
		return nil, fmt.Errorf("write partner create audit: %w", err)
	}
	return created, nil
}

func (uc *PartnerUsecase) Update(ctx context.Context, organizationID, userID, id uuid.UUID, input *Partner) (*Partner, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerNotFound
	}
	normalized, err := normalizePartner(input, false)
	if err != nil {
		return nil, err
	}
	result, err := uc.repo.Update(ctx, organizationID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.update",
		Result:         "success",
		Details: map[string]string{
			"partner.id": result.Partner.ID.String(),
			"from_roles": partnerRolesAuditValue(result.PreviousRoles),
			"to_roles":   partnerRolesAuditValue(result.Partner.Roles),
		},
	}); err != nil {
		return nil, fmt.Errorf("write partner update audit: %w", err)
	}
	return result.Partner, nil
}

func (uc *PartnerUsecase) SetSupplierBlacklist(ctx context.Context, organizationID, userID, id uuid.UUID, blacklisted bool, reason string) (*Partner, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || userID == uuid.Nil || id == uuid.Nil {
		return nil, ErrPartnerNotFound
	}
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrPartnerBlacklistReasonRequired
	}
	result, err := uc.repo.SetSupplierBlacklist(ctx, organizationID, id, PartnerBlacklistUpdate{
		Blacklisted: blacklisted,
		Reason:      reason,
		ChangedAt:   uc.now().UTC(),
		ChangedBy:   userID,
	})
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.supplier_blacklist.set",
		Result:         "success",
		Details: map[string]string{
			"partner.id":             result.Partner.ID.String(),
			"previously_blacklisted": fmt.Sprintf("%t", result.PreviouslyBlacklisted),
			"blacklisted":            fmt.Sprintf("%t", blacklisted),
			"reason":                 reason,
		},
	}); err != nil {
		return nil, fmt.Errorf("write partner blacklist audit: %w", err)
	}
	return result.Partner, nil
}

func normalizePartner(input *Partner, creating bool) (*Partner, error) {
	if input == nil {
		return nil, ErrPartnerInvalidArgument
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.LegalName = strings.TrimSpace(output.LegalName)
	output.NormalizedName = normalizedPartnerName(output.LegalName)
	output.UnifiedSocialCreditCode = strings.ToUpper(strings.TrimSpace(output.UnifiedSocialCreditCode))
	output.RegisteredAddress = strings.TrimSpace(output.RegisteredAddress)
	if creating {
		output.Enabled = true
	}
	if (creating && output.Code == "") || utf8.RuneCountInString(output.Code) > 64 || output.LegalName == "" || utf8.RuneCountInString(output.LegalName) > 200 || utf8.RuneCountInString(output.RegisteredAddress) > 500 {
		return nil, ErrPartnerInvalidArgument
	}
	if output.UnifiedSocialCreditCode != "" && !validUnifiedSocialCreditCode(output.UnifiedSocialCreditCode) {
		return nil, ErrPartnerInvalidArgument
	}

	roles, err := normalizePartnerRoles(output.Roles, output.Enabled)
	if err != nil {
		return nil, err
	}
	contacts, err := normalizePartnerContacts(output.Contacts)
	if err != nil {
		return nil, err
	}
	aliases, err := normalizePartnerAliases(output.Aliases)
	if err != nil {
		return nil, err
	}
	output.Roles = roles
	output.Contacts = contacts
	output.Aliases = aliases
	return &output, nil
}

func normalizePartnerRoles(input []*PartnerRole, partnerEnabled bool) ([]*PartnerRole, error) {
	if len(input) == 0 {
		return nil, ErrPartnerRoleRequired
	}
	seen := make(map[PartnerRoleType]struct{}, len(input))
	roles := make([]*PartnerRole, 0, len(input))
	hasEnabled := false
	for _, item := range input {
		if item == nil || !item.Type.Valid() {
			return nil, ErrPartnerInvalidRole
		}
		if _, exists := seen[item.Type]; exists {
			return nil, ErrPartnerInvalidRole
		}
		seen[item.Type] = struct{}{}
		hasEnabled = hasEnabled || item.Enabled
		roles = append(roles, &PartnerRole{Type: item.Type, Enabled: item.Enabled})
	}
	if partnerEnabled && !hasEnabled {
		return nil, ErrPartnerRoleRequired
	}
	return roles, nil
}

func normalizePartnerContacts(input []*PartnerContact) ([]*PartnerContact, error) {
	contacts := make([]*PartnerContact, 0, len(input))
	primaryCount := 0
	for _, item := range input {
		if item == nil {
			return nil, ErrPartnerInvalidArgument
		}
		contact := *item
		contact.Name = strings.TrimSpace(contact.Name)
		contact.Phone = strings.TrimSpace(contact.Phone)
		contact.Email = strings.TrimSpace(contact.Email)
		contact.Note = strings.TrimSpace(contact.Note)
		if contact.Name == "" || utf8.RuneCountInString(contact.Name) > 100 || utf8.RuneCountInString(contact.Phone) > 64 || utf8.RuneCountInString(contact.Email) > 254 || utf8.RuneCountInString(contact.Note) > 500 {
			return nil, ErrPartnerInvalidArgument
		}
		if contact.Email != "" {
			parsed, err := mail.ParseAddress(contact.Email)
			if err != nil || parsed.Address != contact.Email {
				return nil, ErrPartnerInvalidArgument
			}
		}
		if contact.IsPrimary {
			primaryCount++
		}
		contacts = append(contacts, &contact)
	}
	if primaryCount > 1 {
		return nil, ErrPartnerPrimaryContactConflict
	}
	return contacts, nil
}

func normalizePartnerAliases(input []*PartnerAlias) ([]*PartnerAlias, error) {
	aliases := make([]*PartnerAlias, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		if item == nil {
			return nil, ErrPartnerInvalidArgument
		}
		alias := *item
		alias.AliasName = strings.TrimSpace(alias.AliasName)
		normalized := normalizedPartnerName(alias.AliasName)
		alias.NormalizedAliasName = normalized
		if alias.AliasName == "" || utf8.RuneCountInString(alias.AliasName) > 200 || alias.SortOrder < 0 {
			return nil, ErrPartnerInvalidArgument
		}
		if _, exists := seen[normalized]; exists {
			return nil, ErrPartnerAliasExists
		}
		seen[normalized] = struct{}{}
		aliases = append(aliases, &alias)
	}
	return aliases, nil
}

func normalizedPartnerName(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}

func validUnifiedSocialCreditCode(value string) bool {
	if len(value) != 18 {
		return false
	}
	const allowed = "0123456789ABCDEFGHJKLMNPQRTUWXY"
	for _, character := range value {
		if !strings.ContainsRune(allowed, character) {
			return false
		}
	}
	return true
}

func partnerRolesAuditValue(roles []*PartnerRole) string {
	values := make([]string, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			values = append(values, fmt.Sprintf("%s:%t", role.Type, role.Enabled))
		}
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}
