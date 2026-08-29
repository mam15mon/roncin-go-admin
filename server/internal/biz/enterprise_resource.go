package biz

import (
	"context"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrEnterpriseResourceNotFound        = errors.NotFound("ENTERPRISE_RESOURCE_NOT_FOUND", "企业资源不存在")
	ErrEnterpriseResourceInvalidArgument = errors.BadRequest("ENTERPRISE_RESOURCE_INVALID_ARGUMENT", "企业资源字段不合法")
	ErrEnterpriseTagGroupNotEmpty        = errors.Conflict("ENTERPRISE_TAG_GROUP_NOT_EMPTY", "标签组下仍有标签")
)

type EnterpriseResourceType string

const (
	EnterpriseResourceAddressType          EnterpriseResourceType = "ADDRESS"
	EnterpriseResourceRemarkType           EnterpriseResourceType = "REMARK"
	EnterpriseResourceImageType            EnterpriseResourceType = "IMAGE"
	EnterpriseResourceTagType              EnterpriseResourceType = "TAG"
	EnterpriseResourceShipperType          EnterpriseResourceType = "SHIPPER"
	EnterpriseResourceConsigneeType        EnterpriseResourceType = "CONSIGNEE"
	EnterpriseResourceNotifyPartyType      EnterpriseResourceType = "NOTIFY_PARTY"
	EnterpriseResourceEnglishCargoNameType EnterpriseResourceType = "ENGLISH_CARGO_NAME"
	EnterpriseResourceHSCodeType           EnterpriseResourceType = "HS_CODE"
	EnterpriseResourceMarksType            EnterpriseResourceType = "MARKS"
)

func (value EnterpriseResourceType) Public() bool {
	switch value {
	case EnterpriseResourceAddressType, EnterpriseResourceRemarkType, EnterpriseResourceImageType, EnterpriseResourceTagType, EnterpriseResourceShipperType, EnterpriseResourceConsigneeType, EnterpriseResourceNotifyPartyType:
		return true
	default:
		return false
	}
}

func (value EnterpriseResourceType) Party() bool {
	return value == EnterpriseResourceShipperType || value == EnterpriseResourceConsigneeType || value == EnterpriseResourceNotifyPartyType
}

func (value EnterpriseResourceType) BatchAssociable() bool {
	return value == EnterpriseResourceAddressType || value.Party()
}

type EnterpriseRemarkType string
type EnterpriseAddressType string

var enterpriseRemarkTypes = map[EnterpriseRemarkType]struct{}{
	"BOOKING": {}, "ALLOCATION": {}, "TRANSPORT": {}, "ORDER": {}, "BILL_OF_LADING": {}, "CUSTOMER": {}, "SUPPLIER": {}, "FOREIGN_AGENT": {}, "QUOTATION": {}, "MANIFEST": {}, "PACKING_LIST": {}, "OPERATION": {}, "COMMISSION": {}, "WAREHOUSE": {},
}
var enterpriseAddressTypes = map[EnterpriseAddressType]struct{}{"CONTAINER_OPERATION": {}, "PICKUP": {}, "DELIVERY": {}}
var tagColorPattern = regexp.MustCompile(`^#[0-9A-F]{6}$`)

type EnterpriseResourceAddress struct {
	ContactName, ContactPhone, CountryCode, ProvinceCode, CityCode, DistrictCode, AddressDetail, Remark string
}
type EnterpriseResourceRemark struct {
	RemarkType EnterpriseRemarkType
	Content    string
}
type EnterpriseResourceParty struct {
	CompanyName, BusinessCode, Address, CountryCode, ContactName, ContactPhone, Email, TaxIdentifier, AEOCode string
	CustomDisplay                                                                                             bool
	DisplayContent, Remark                                                                                    string
}
type EnterpriseResourceImage struct {
	FileName, MIMEType, ObjectKey, Checksum string
	FileSize                                int64
	Width, Height                           *int
}
type EnterpriseResourceTag struct{ GroupID uuid.UUID }

type EnterpriseResource struct {
	ID, OrganizationID      uuid.UUID
	ResourceType            EnterpriseResourceType
	ShortName               string
	Enabled                 bool
	SortOrder               int
	PartnerIDs, AssigneeIDs []uuid.UUID
	AddressTypes            []EnterpriseAddressType
	Address                 *EnterpriseResourceAddress
	Remark                  *EnterpriseResourceRemark
	Party                   *EnterpriseResourceParty
	Image                   *EnterpriseResourceImage
	Tag                     *EnterpriseResourceTag
	CreatedBy, UpdatedBy    *uuid.UUID
	CreatedAt, UpdatedAt    time.Time
}

type EnterpriseResourceListOptions struct {
	ResourceType          EnterpriseResourceType
	PartnerID, AssigneeID *uuid.UUID
	Linked, Enabled       *bool
	AddressType           *EnterpriseAddressType
	Keyword               string
	Page, PageSize        int
}

type EnterpriseTagGroup struct {
	ID, OrganizationID   uuid.UUID
	Name, Color          string
	SortOrder            int
	CreatedAt, UpdatedAt time.Time
}

type EnterpriseResourceRepo interface {
	List(context.Context, uuid.UUID, EnterpriseResourceListOptions) ([]*EnterpriseResource, int64, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (*EnterpriseResource, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *EnterpriseResource, *AuditEvent) (*EnterpriseResource, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *EnterpriseResource, *AuditEvent) (*EnterpriseResource, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, *AuditEvent) error
	BatchPartners(context.Context, uuid.UUID, []uuid.UUID, []uuid.UUID, bool, *AuditEvent) (int, error)
	BatchAddressTypes(context.Context, uuid.UUID, []uuid.UUID, []EnterpriseAddressType, bool, *AuditEvent) (int, error)
	BatchAssignees(context.Context, uuid.UUID, []uuid.UUID, []uuid.UUID, bool, *AuditEvent) (int, error)
	ListTagGroups(context.Context, uuid.UUID) ([]*EnterpriseTagGroup, error)
	CreateTagGroup(context.Context, uuid.UUID, *EnterpriseTagGroup, *AuditEvent) (*EnterpriseTagGroup, error)
	UpdateTagGroup(context.Context, uuid.UUID, uuid.UUID, *EnterpriseTagGroup, *AuditEvent) (*EnterpriseTagGroup, error)
	DeleteTagGroup(context.Context, uuid.UUID, uuid.UUID, *AuditEvent) error
	Import(context.Context, uuid.UUID, uuid.UUID, []*EnterpriseResource, *AuditEvent) ([]*EnterpriseResource, error)
}

type EnterpriseResourceUsecase struct{ repo EnterpriseResourceRepo }

func NewEnterpriseResourceUsecase(repo EnterpriseResourceRepo) *EnterpriseResourceUsecase {
	return &EnterpriseResourceUsecase{repo: repo}
}

func (uc *EnterpriseResourceUsecase) List(ctx context.Context, organizationID uuid.UUID, options EnterpriseResourceListOptions) ([]*EnterpriseResource, int64, error) {
	if organizationID == uuid.Nil || !options.ResourceType.Public() || options.Page < 1 || options.PageSize < 1 || options.PageSize > MaxListPageSize {
		return nil, 0, ErrEnterpriseResourceInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *EnterpriseResourceUsecase) Get(ctx context.Context, organizationID, id uuid.UUID) (*EnterpriseResource, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	return uc.repo.Get(ctx, organizationID, id)
}

func (uc *EnterpriseResourceUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *EnterpriseResource) (*EnterpriseResource, error) {
	value, err := normalizeEnterpriseResource(input)
	if err != nil {
		return nil, err
	}
	action := "enterprise_resource.create"
	if value.ResourceType == EnterpriseResourceTagType {
		action = "enterprise_tag.create"
	}
	return uc.repo.Create(ctx, organizationID, actorID, value, enterpriseResourceAudit(organizationID, actorID, action, uuid.Nil, value.ResourceType))
}

func (uc *EnterpriseResourceUsecase) Update(ctx context.Context, organizationID, actorID, id uuid.UUID, input *EnterpriseResource) (*EnterpriseResource, error) {
	if id == uuid.Nil {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	value, err := normalizeEnterpriseResource(input)
	if err != nil {
		return nil, err
	}
	action := "enterprise_resource.update"
	if value.ResourceType == EnterpriseResourceTagType {
		action = "enterprise_tag.update"
	}
	return uc.repo.Update(ctx, organizationID, actorID, id, value, enterpriseResourceAudit(organizationID, actorID, action, id, value.ResourceType))
}

func (uc *EnterpriseResourceUsecase) Delete(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrEnterpriseResourceInvalidArgument
	}
	return uc.repo.Delete(ctx, organizationID, id, enterpriseResourceAudit(organizationID, actorID, "enterprise_resource.delete", id, ""))
}

func (uc *EnterpriseResourceUsecase) BatchPartners(ctx context.Context, organizationID, actorID uuid.UUID, resourceIDs, partnerIDs []uuid.UUID, create bool) (int, error) {
	if len(resourceIDs) == 0 || len(partnerIDs) == 0 || containsNilUUID(resourceIDs) || containsNilUUID(partnerIDs) {
		return 0, ErrEnterpriseResourceInvalidArgument
	}
	action := "enterprise_resource.partner.batch_unlink"
	if create {
		action = "enterprise_resource.partner.batch_link"
	}
	audit := enterpriseResourceAudit(organizationID, actorID, action, uuid.Nil, "")
	audit.Details["resource.ids"] = joinUUIDs(resourceIDs)
	audit.Details["partner.ids"] = joinUUIDs(partnerIDs)
	return uc.repo.BatchPartners(ctx, organizationID, resourceIDs, partnerIDs, create, audit)
}

func (uc *EnterpriseResourceUsecase) BatchAddressTypes(ctx context.Context, organizationID, actorID uuid.UUID, resourceIDs []uuid.UUID, values []EnterpriseAddressType, assign bool) (int, error) {
	if len(resourceIDs) == 0 || len(values) == 0 || containsNilUUID(resourceIDs) {
		return 0, ErrEnterpriseResourceInvalidArgument
	}
	for _, value := range values {
		if _, ok := enterpriseAddressTypes[value]; !ok {
			return 0, ErrEnterpriseResourceInvalidArgument
		}
	}
	action := "enterprise_resource.address_type.batch_remove"
	if assign {
		action = "enterprise_resource.address_type.batch_assign"
	}
	return uc.repo.BatchAddressTypes(ctx, organizationID, resourceIDs, values, assign, enterpriseResourceAudit(organizationID, actorID, action, uuid.Nil, EnterpriseResourceAddressType))
}

func (uc *EnterpriseResourceUsecase) BatchAssignees(ctx context.Context, organizationID, actorID uuid.UUID, resourceIDs, userIDs []uuid.UUID, assign bool) (int, error) {
	if len(resourceIDs) == 0 || len(userIDs) == 0 || containsNilUUID(resourceIDs) || containsNilUUID(userIDs) {
		return 0, ErrEnterpriseResourceInvalidArgument
	}
	action := "enterprise_resource.assignee.batch_remove"
	if assign {
		action = "enterprise_resource.assignee.batch_assign"
	}
	return uc.repo.BatchAssignees(ctx, organizationID, resourceIDs, userIDs, assign, enterpriseResourceAudit(organizationID, actorID, action, uuid.Nil, EnterpriseResourceAddressType))
}

func (uc *EnterpriseResourceUsecase) ListTagGroups(ctx context.Context, organizationID uuid.UUID) ([]*EnterpriseTagGroup, error) {
	return uc.repo.ListTagGroups(ctx, organizationID)
}

func (uc *EnterpriseResourceUsecase) CreateTagGroup(ctx context.Context, organizationID, actorID uuid.UUID, input *EnterpriseTagGroup) (*EnterpriseTagGroup, error) {
	value, err := normalizeEnterpriseTagGroup(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.CreateTagGroup(ctx, organizationID, value, enterpriseResourceAudit(organizationID, actorID, "enterprise_tag_group.create", uuid.Nil, EnterpriseResourceTagType))
}

func (uc *EnterpriseResourceUsecase) UpdateTagGroup(ctx context.Context, organizationID, actorID, id uuid.UUID, input *EnterpriseTagGroup) (*EnterpriseTagGroup, error) {
	value, err := normalizeEnterpriseTagGroup(input)
	if err != nil || id == uuid.Nil {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	return uc.repo.UpdateTagGroup(ctx, organizationID, id, value, enterpriseResourceAudit(organizationID, actorID, "enterprise_tag_group.update", id, EnterpriseResourceTagType))
}

func (uc *EnterpriseResourceUsecase) DeleteTagGroup(ctx context.Context, organizationID, actorID, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrEnterpriseResourceInvalidArgument
	}
	return uc.repo.DeleteTagGroup(ctx, organizationID, id, enterpriseResourceAudit(organizationID, actorID, "enterprise_tag_group.delete", id, EnterpriseResourceTagType))
}

func (uc *EnterpriseResourceUsecase) PreviewImport(inputs []*EnterpriseResource, resourceType EnterpriseResourceType) ([]*EnterpriseResource, []error) {
	values := make([]*EnterpriseResource, len(inputs))
	errs := make([]error, len(inputs))
	if !resourceType.Party() {
		for i := range errs {
			errs[i] = ErrEnterpriseResourceInvalidArgument
		}
		return values, errs
	}
	seen := make(map[string]struct{})
	for i, input := range inputs {
		if input == nil || input.ResourceType != resourceType {
			errs[i] = ErrEnterpriseResourceInvalidArgument
			continue
		}
		values[i], errs[i] = normalizeEnterpriseResource(input)
		if errs[i] == nil && values[i].Party.BusinessCode != "" {
			key := strings.ToUpper(values[i].Party.BusinessCode)
			if _, ok := seen[key]; ok {
				errs[i] = ErrEnterpriseResourceInvalidArgument
			} else {
				seen[key] = struct{}{}
			}
		}
	}
	return values, errs
}

func (uc *EnterpriseResourceUsecase) CommitImport(ctx context.Context, organizationID, actorID uuid.UUID, inputs []*EnterpriseResource, resourceType EnterpriseResourceType) ([]*EnterpriseResource, []error, error) {
	values, rowErrors := uc.PreviewImport(inputs, resourceType)
	for _, err := range rowErrors {
		if err != nil {
			return nil, rowErrors, ErrEnterpriseResourceInvalidArgument
		}
	}
	created, err := uc.repo.Import(ctx, organizationID, actorID, values, enterpriseResourceAudit(organizationID, actorID, "enterprise_resource.import", uuid.Nil, resourceType))
	return created, rowErrors, err
}

func normalizeEnterpriseResource(input *EnterpriseResource) (*EnterpriseResource, error) {
	if input == nil || !input.ResourceType.Public() || input.SortOrder < 0 {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	value := *input
	value.ShortName = strings.TrimSpace(value.ShortName)
	if value.ShortName == "" || utf8.RuneCountInString(value.ShortName) > 200 {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	detailCount := 0
	for _, exists := range []bool{value.Address != nil, value.Remark != nil, value.Party != nil, value.Image != nil, value.Tag != nil} {
		if exists {
			detailCount++
		}
	}
	if detailCount != 1 {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	switch value.ResourceType {
	case EnterpriseResourceAddressType:
		if value.Address == nil {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		detail := *value.Address
		detail.CountryCode = strings.ToUpper(strings.TrimSpace(detail.CountryCode))
		detail.AddressDetail = strings.TrimSpace(detail.AddressDetail)
		if len(detail.CountryCode) != 2 || detail.AddressDetail == "" {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		value.Address = &detail
	case EnterpriseResourceRemarkType:
		if value.Remark == nil {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		detail := *value.Remark
		detail.Content = strings.TrimSpace(detail.Content)
		if _, ok := enterpriseRemarkTypes[detail.RemarkType]; !ok || detail.Content == "" || utf8.RuneCountInString(detail.Content) > 4000 {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		value.Remark = &detail
	case EnterpriseResourceImageType:
		if value.Image == nil || value.Image.FileSize <= 0 || value.Image.FileSize > 20*1024*1024 || strings.TrimSpace(value.Image.ObjectKey) == "" || strings.TrimSpace(value.Image.Checksum) == "" {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		switch value.Image.MIMEType {
		case "image/jpeg", "image/png", "image/bmp", "image/gif":
		default:
			return nil, ErrEnterpriseResourceInvalidArgument
		}
	case EnterpriseResourceTagType:
		if value.Tag == nil || value.Tag.GroupID == uuid.Nil {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
	default:
		if !value.ResourceType.Party() || value.Party == nil {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		detail := *value.Party
		detail.CompanyName = strings.TrimSpace(detail.CompanyName)
		detail.BusinessCode = strings.TrimSpace(detail.BusinessCode)
		detail.CountryCode = strings.ToUpper(strings.TrimSpace(detail.CountryCode))
		detail.Email = strings.TrimSpace(detail.Email)
		if detail.CompanyName == "" || len(detail.CountryCode) != 2 {
			return nil, ErrEnterpriseResourceInvalidArgument
		}
		if detail.Email != "" {
			parsed, err := mail.ParseAddress(detail.Email)
			if err != nil || parsed.Address != detail.Email {
				return nil, ErrEnterpriseResourceInvalidArgument
			}
		}
		if !detail.CustomDisplay {
			detail.DisplayContent = strings.TrimSpace(strings.Join([]string{detail.CompanyName, detail.Address, detail.ContactName, detail.ContactPhone}, "\n"))
		}
		value.Party = &detail
	}
	return &value, nil
}

func normalizeEnterpriseTagGroup(input *EnterpriseTagGroup) (*EnterpriseTagGroup, error) {
	if input == nil || input.SortOrder < 0 {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	value := *input
	value.Name = strings.TrimSpace(value.Name)
	value.Color = strings.ToUpper(strings.TrimSpace(value.Color))
	if value.Name == "" || utf8.RuneCountInString(value.Name) > 100 || (value.Color != "" && !tagColorPattern.MatchString(value.Color)) {
		return nil, ErrEnterpriseResourceInvalidArgument
	}
	return &value, nil
}

func enterpriseResourceAudit(organizationID, actorID uuid.UUID, action string, id uuid.UUID, resourceType EnterpriseResourceType) *AuditEvent {
	details := map[string]string{"resource.type": string(resourceType)}
	if id != uuid.Nil {
		details["resource.id"] = id.String()
	}
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, ResourceType: "enterprise_resource", ResourceID: id.String(), Result: "success", Details: details}
}

func containsNilUUID(values []uuid.UUID) bool {
	for _, value := range values {
		if value == uuid.Nil {
			return true
		}
	}
	return false
}

func joinUUIDs(values []uuid.UUID) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = value.String()
	}
	return strings.Join(parts, ",")
}
