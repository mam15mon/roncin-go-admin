package biz

import (
	"context"
	"fmt"
	"net/mail"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerNotFound                = errors.NotFound("PARTNER_NOT_FOUND", "往来单位不存在")
	ErrPartnerCodeExists              = errors.Conflict("PARTNER_CODE_EXISTS", "往来单位编码已存在")
	ErrPartnerNameExists              = errors.Conflict("PARTNER_NAME_EXISTS", "往来单位名称已存在")
	ErrPartnerUSCCExists              = errors.Conflict("PARTNER_USCC_EXISTS", "统一社会信用代码已存在")
	ErrPartnerTaxIdentifierRequired   = errors.BadRequest("PARTNER_TAX_IDENTIFIER_REQUIRED", "客户或供应商必须填写纳税人识别号")
	ErrPartnerRoleRequired            = errors.BadRequest("PARTNER_ROLE_REQUIRED", "往来单位至少需要一个有效角色")
	ErrPartnerInvalidRole             = errors.BadRequest("PARTNER_INVALID_ROLE", "往来单位角色不合法")
	ErrPartnerInvalidArgument         = errors.BadRequest("PARTNER_INVALID_ARGUMENT", "往来单位字段不合法")
	ErrPartnerPrimaryContactConflict  = errors.BadRequest("PARTNER_PRIMARY_CONTACT_CONFLICT", "只能设置一个主联系人")
	ErrPartnerAliasExists             = errors.Conflict("PARTNER_ALIAS_EXISTS", "往来单位别名重复")
	ErrPartnerSupplierRoleRequired    = errors.BadRequest("PARTNER_SUPPLIER_ROLE_REQUIRED", "往来单位没有供应商角色")
	ErrPartnerBlacklistReasonRequired = errors.BadRequest("PARTNER_BLACKLIST_REASON_REQUIRED", "黑名单变更原因不能为空")
	ErrPartnerBlacklistedSupplierRole = errors.BadRequest("PARTNER_BLACKLISTED_SUPPLIER_ROLE", "清除黑名单前不能移除供应商角色")
	ErrPartnerImportInvalidArgument   = errors.BadRequest("PARTNER_INVALID_ARGUMENT", "往来单位导入参数不合法")
)

type PartnerRoleType string

const (
	PartnerRoleCustomer     PartnerRoleType = "customer"
	PartnerRoleSupplier     PartnerRoleType = "supplier"
	PartnerRoleForeignAgent PartnerRoleType = "foreign_agent"
	PartnerRoleCarrier      PartnerRoleType = "carrier"
)

func (t PartnerRoleType) Valid() bool {
	return t == PartnerRoleCustomer || t == PartnerRoleSupplier || t == PartnerRoleForeignAgent || t == PartnerRoleCarrier
}

type PartnerRole struct {
	Type            PartnerRoleType
	Enabled         bool
	Blacklisted     bool
	BlacklistReason string
	BlacklistedAt   *time.Time
	BlacklistedBy   *uuid.UUID
	SettlementRule  *PartnerSettlementRule
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

type PartnerCustomerType string

const (
	PartnerCustomerDirect PartnerCustomerType = "DIRECT"
	PartnerCustomerPeer   PartnerCustomerType = "PEER"
)

func (t PartnerCustomerType) Valid() bool {
	return t == PartnerCustomerDirect || t == PartnerCustomerPeer
}

type PartnerBusinessType string

const (
	PartnerBusinessSE   PartnerBusinessType = "SE"
	PartnerBusinessSI   PartnerBusinessType = "SI"
	PartnerBusinessAE   PartnerBusinessType = "AE"
	PartnerBusinessAI   PartnerBusinessType = "AI"
	PartnerBusinessLand PartnerBusinessType = "LAND"
	PartnerBusinessRail PartnerBusinessType = "RAIL"
)

func (t PartnerBusinessType) Valid() bool {
	switch t {
	case PartnerBusinessSE, PartnerBusinessSI, PartnerBusinessAE, PartnerBusinessAI, PartnerBusinessLand, PartnerBusinessRail:
		return true
	default:
		return false
	}
}

type PartnerProfile struct {
	NameEN            string
	AddressEN         string
	CountryCode       string
	ProvinceCode      string
	CityCode          string
	DistrictCode      string
	AddressDetail     string
	Nature            string
	DevelopmentMethod string
	CustomerTypes     []PartnerCustomerType
	BusinessTypes     []PartnerBusinessType
	Remark            string
}

type PartnerAssignmentRole string

const (
	PartnerAssignmentCreator         PartnerAssignmentRole = "CREATOR"
	PartnerAssignmentOperator        PartnerAssignmentRole = "OPERATOR"
	PartnerAssignmentSales           PartnerAssignmentRole = "SALES"
	PartnerAssignmentCustomerService PartnerAssignmentRole = "CUSTOMER_SERVICE"
	PartnerAssignmentFinance         PartnerAssignmentRole = "FINANCE"
	PartnerAssignmentCommercial      PartnerAssignmentRole = "COMMERCIAL"
	PartnerAssignmentInternalContact PartnerAssignmentRole = "INTERNAL_CONTACT"
	PartnerAssignmentDocument        PartnerAssignmentRole = "DOCUMENT"
)

func (r PartnerAssignmentRole) Valid() bool {
	switch r {
	case PartnerAssignmentCreator, PartnerAssignmentOperator, PartnerAssignmentSales, PartnerAssignmentCustomerService, PartnerAssignmentFinance, PartnerAssignmentCommercial, PartnerAssignmentInternalContact, PartnerAssignmentDocument:
		return true
	default:
		return false
	}
}

type PartnerAssignment struct {
	ID             uuid.UUID
	Role           PartnerAssignmentRole
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	SortOrder      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PartnerAssignmentOption struct {
	UserID            uuid.UUID
	DisplayName       string
	OrganizationID    uuid.UUID
	OrganizationName  string
	MembershipEnabled bool
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
	Profile                 *PartnerProfile
	Assignments             []*PartnerAssignment
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

type PartnerList = PagedList[*Partner]

type PartnerAuditLog struct {
	Log             *AuditLog
	UserDisplayName string
}

type PartnerAuditLogList = PagedList[*PartnerAuditLog]

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
	ListAssignmentOptions(context.Context, uuid.UUID, SelectorListOptions) (*PagedList[*PartnerAssignmentOption], error)
	ListAuditLogs(context.Context, uuid.UUID, uuid.UUID, int, int) (*PartnerAuditLogList, error)
	Create(context.Context, uuid.UUID, *Partner, *AuditEvent) (*Partner, error)
	Update(context.Context, uuid.UUID, uuid.UUID, *Partner, *AuditEvent) (*PartnerUpdateResult, error)
	SetSupplierBlacklist(context.Context, uuid.UUID, uuid.UUID, PartnerBlacklistUpdate, *AuditEvent) (*PartnerBlacklistResult, error)
	Import(context.Context, uuid.UUID, PartnerImportMode, []*Partner, *AuditEvent) (*PartnerImportResult, error)
}

type PartnerUsecase struct {
	repo PartnerRepo
	now  func() time.Time
}

func NewPartnerUsecase(repo PartnerRepo) *PartnerUsecase {
	return &PartnerUsecase{repo: repo, now: time.Now}
}

func (uc *PartnerUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*Partner, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrPartnerNotFound
	}
	return uc.repo.Get(ctx, organizationID, id)
}

func (uc *PartnerUsecase) List(ctx context.Context, organizationID uuid.UUID, options PartnerListOptions) (*PartnerList, error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrPartnerInvalidArgument
	}
	if options.Role != "" && !options.Role.Valid() {
		return nil, ErrPartnerInvalidRole
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *PartnerUsecase) ListAssignmentOptions(ctx context.Context, organizationID uuid.UUID, options SelectorListOptions) (*PagedList[*PartnerAssignmentOption], error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrPartnerInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.ListAssignmentOptions(ctx, organizationID, options)
}

func (uc *PartnerUsecase) ListAuditLogs(ctx context.Context, organizationID, partnerID uuid.UUID, page, pageSize int) (*PartnerAuditLogList, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil || !ValidListPagination(page, pageSize) {
		return nil, ErrPartnerInvalidArgument
	}
	return uc.repo.ListAuditLogs(ctx, organizationID, partnerID, page, pageSize)
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
		value.Assignments = append(value.Assignments, &PartnerAssignment{Role: PartnerAssignmentCreator, UserID: userID, OrganizationID: organizationID})
		normalized = append(normalized, value)
	}
	return uc.repo.Import(ctx, organizationID, input.Mode, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.import",
		Result:         "success",
		Details: map[string]string{
			"source": input.Source,
			"mode":   string(input.Mode),
		},
	})
}

func (uc *PartnerUsecase) Create(ctx context.Context, organizationID, userID uuid.UUID, input *Partner) (*Partner, error) {
	normalized, err := normalizePartner(input, true)
	if err != nil {
		return nil, err
	}
	normalized.Assignments = append(normalized.Assignments, &PartnerAssignment{Role: PartnerAssignmentCreator, UserID: userID, OrganizationID: organizationID})
	return uc.repo.Create(ctx, organizationID, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.create",
		ResourceType:   "partner",
		Result:         "success",
		Details: map[string]string{
			"partner.code": normalized.Code,
			"legal_name":   normalized.LegalName,
			"roles":        FormatPartnerRolesAuditValue(normalized.Roles),
		},
	})
}

func (uc *PartnerUsecase) Update(ctx context.Context, organizationID, userID, id uuid.UUID, input *Partner) (*Partner, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerNotFound
	}
	normalized, err := normalizePartner(input, false)
	if err != nil {
		return nil, err
	}
	result, err := uc.repo.Update(ctx, organizationID, id, normalized, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.update",
		ResourceType:   "partner",
		ResourceID:     id.String(),
		Result:         "success",
		Details: map[string]string{
			"partner.id": id.String(),
			"to_roles":   FormatPartnerRolesAuditValue(normalized.Roles),
		},
	})
	if err != nil {
		return nil, err
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
	}, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.supplier_blacklist.set",
		ResourceType:   "partner",
		ResourceID:     id.String(),
		Result:         "success",
		Details: map[string]string{
			"partner.id":  id.String(),
			"blacklisted": fmt.Sprintf("%t", blacklisted),
			"reason":      reason,
		},
	})
	if err != nil {
		return nil, err
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
	if requiresPartnerTaxIdentifier(roles) && output.UnifiedSocialCreditCode == "" {
		return nil, ErrPartnerTaxIdentifierRequired
	}
	output.Roles = roles
	output.Contacts = contacts
	output.Aliases = aliases
	profile, err := normalizePartnerProfile(output.Profile)
	if err != nil {
		return nil, err
	}
	assignments, err := normalizePartnerAssignments(output.Assignments)
	if err != nil {
		return nil, err
	}
	output.Profile = profile
	output.Assignments = assignments
	return &output, nil
}

func normalizePartnerProfile(input *PartnerProfile) (*PartnerProfile, error) {
	if input == nil {
		return nil, nil
	}
	output := *input
	output.NameEN = strings.TrimSpace(output.NameEN)
	output.AddressEN = strings.TrimSpace(output.AddressEN)
	output.CountryCode = strings.ToUpper(strings.TrimSpace(output.CountryCode))
	output.ProvinceCode = strings.TrimSpace(output.ProvinceCode)
	output.CityCode = strings.TrimSpace(output.CityCode)
	output.DistrictCode = strings.TrimSpace(output.DistrictCode)
	output.AddressDetail = strings.TrimSpace(output.AddressDetail)
	output.Nature = strings.TrimSpace(output.Nature)
	output.DevelopmentMethod = strings.TrimSpace(output.DevelopmentMethod)
	output.Remark = strings.TrimSpace(output.Remark)
	if output.CountryCode == "" {
		output.CountryCode = "CN"
	}
	if len(output.CountryCode) != 2 || utf8.RuneCountInString(output.NameEN) > 200 || utf8.RuneCountInString(output.AddressEN) > 500 || utf8.RuneCountInString(output.AddressDetail) > 500 || utf8.RuneCountInString(output.Nature) > 100 || utf8.RuneCountInString(output.DevelopmentMethod) > 100 || utf8.RuneCountInString(output.Remark) > 2000 {
		return nil, ErrPartnerInvalidArgument
	}
	locationCodes := []string{output.ProvinceCode, output.CityCode, output.DistrictCode}
	for _, code := range locationCodes {
		if code != "" && !administrativeRegionCodePattern.MatchString(code) {
			return nil, ErrPartnerInvalidArgument
		}
	}
	if output.CountryCode != "CN" && (output.ProvinceCode != "" || output.CityCode != "" || output.DistrictCode != "") {
		return nil, ErrPartnerInvalidArgument
	}
	if (output.CityCode != "" && output.ProvinceCode == "") || (output.DistrictCode != "" && output.CityCode == "") {
		return nil, ErrPartnerInvalidArgument
	}
	customerTypes, err := normalizeUniqueValues(output.CustomerTypes, func(value PartnerCustomerType) bool { return value.Valid() })
	if err != nil {
		return nil, err
	}
	businessTypes, err := normalizeUniqueValues(output.BusinessTypes, func(value PartnerBusinessType) bool { return value.Valid() })
	if err != nil {
		return nil, err
	}
	output.CustomerTypes = customerTypes
	output.BusinessTypes = businessTypes
	return &output, nil
}

func normalizeUniqueValues[T comparable](values []T, valid func(T) bool) ([]T, error) {
	seen := make(map[T]struct{}, len(values))
	result := make([]T, 0, len(values))
	for _, value := range values {
		if !valid(value) {
			return nil, ErrPartnerInvalidArgument
		}
		if _, exists := seen[value]; exists {
			return nil, ErrPartnerInvalidArgument
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func normalizePartnerAssignments(input []*PartnerAssignment) ([]*PartnerAssignment, error) {
	counts := make(map[PartnerAssignmentRole]int, len(input))
	seenMembers := make(map[string]struct{}, len(input))
	result := make([]*PartnerAssignment, 0, len(input))
	for _, item := range input {
		if item == nil || !item.Role.Valid() || item.Role == PartnerAssignmentCreator || item.UserID == uuid.Nil || item.OrganizationID == uuid.Nil {
			return nil, ErrPartnerInvalidArgument
		}
		counts[item.Role]++
		if item.Role == PartnerAssignmentInternalContact {
			if counts[item.Role] > 2 {
				return nil, ErrPartnerInvalidArgument
			}
		} else if counts[item.Role] > 1 {
			return nil, ErrPartnerInvalidArgument
		}
		memberKey := item.UserID.String() + ":" + item.OrganizationID.String()
		if _, exists := seenMembers[memberKey]; exists {
			return nil, ErrPartnerInvalidArgument
		}
		seenMembers[memberKey] = struct{}{}
		copy := *item
		if copy.Role == PartnerAssignmentInternalContact {
			copy.SortOrder = counts[item.Role]
		} else {
			copy.SortOrder = 0
		}
		result = append(result, &copy)
	}
	return result, nil
}

func requiresPartnerTaxIdentifier(roles []*PartnerRole) bool {
	for _, role := range roles {
		if role.Enabled && (role.Type == PartnerRoleCustomer || role.Type == PartnerRoleSupplier) {
			return true
		}
	}
	return false
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
		role := &PartnerRole{Type: item.Type, Enabled: item.Enabled}
		if item.SettlementRule != nil {
			settlementRule, err := normalizePartnerSettlementRule(item.SettlementRule)
			if err != nil {
				return nil, err
			}
			role.SettlementRule = settlementRule
		}
		roles = append(roles, role)
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

// FormatPartnerRolesAuditValue 生成稳定的往来单位角色审计值。
func FormatPartnerRolesAuditValue(roles []*PartnerRole) string {
	values := make([]string, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			values = append(values, fmt.Sprintf("%s:%t", role.Type, role.Enabled))
		}
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}
