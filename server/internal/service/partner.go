package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

type PartnerService struct {
	v1.UnimplementedPartnerServiceServer
	usecase *biz.PartnerUsecase
}

func NewPartnerService(usecase *biz.PartnerUsecase) *PartnerService {
	return &PartnerService{usecase: usecase}
}

func (s *PartnerService) GetPartner(ctx context.Context, request *v1.GetPartnerRequest) (*v1.PartnerReply, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrPartnerNotFound
	}
	item, err := s.usecase.Get(ctx, principal.Organization.ID, partnerID)
	if err != nil {
		return nil, err
	}
	return partnerReply(ctx, item), nil
}

func (s *PartnerService) ListPartners(ctx context.Context, request *v1.ListPartnersRequest) (*v1.PartnerListReply, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	page, pageSize, err := pageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	options := biz.PartnerListOptions{
		Page:     page,
		PageSize: pageSize,
		Keyword:  request.GetKeyword(),
		Role:     partnerRoleTypeFromAPI(request.GetRole()),
	}
	if request.Enabled != nil {
		enabled := request.GetEnabled()
		options.Enabled = &enabled
	}
	result, err := s.usecase.List(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.Partner, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, partnerToAPI(item))
	}
	return &v1.PartnerListReply{
		Success: true, Code: 0, Message: "OK", Data: data,
		Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize), TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *PartnerService) CreatePartner(ctx context.Context, request *v1.CreatePartnerRequest) (*v1.PartnerReply, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, &biz.Partner{
		Code: request.GetCode(), LegalName: request.GetLegalName(),
		UnifiedSocialCreditCode: request.GetUnifiedSocialCreditCode(), RegisteredAddress: request.GetRegisteredAddress(),
		Roles: partnerRolesFromAPI(request.GetRoles()), Contacts: partnerContactsFromAPI(request.GetContacts()), Aliases: partnerAliasesFromAPI(request.GetAliases()),
	})
	if err != nil {
		return nil, err
	}
	return partnerReply(ctx, created), nil
}

func (s *PartnerService) UpdatePartner(ctx context.Context, request *v1.UpdatePartnerRequest) (*v1.PartnerReply, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrPartnerNotFound
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, &biz.Partner{
		LegalName: request.GetLegalName(), UnifiedSocialCreditCode: request.GetUnifiedSocialCreditCode(),
		RegisteredAddress: request.GetRegisteredAddress(), Enabled: request.GetEnabled(),
		Roles: partnerRolesFromAPI(request.GetRoles()), Contacts: partnerContactsFromAPI(request.GetContacts()), Aliases: partnerAliasesFromAPI(request.GetAliases()),
	})
	if err != nil {
		return nil, err
	}
	return partnerReply(ctx, updated), nil
}

func (s *PartnerService) SetSupplierBlacklist(ctx context.Context, request *v1.SetSupplierBlacklistRequest) (*v1.PartnerReply, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrPartnerNotFound
	}
	updated, err := s.usecase.SetSupplierBlacklist(ctx, principal.Organization.ID, principal.UserID, partnerID, request.GetBlacklisted(), request.GetReason())
	if err != nil {
		return nil, err
	}
	return partnerReply(ctx, updated), nil
}

func pageValues(page, pageSize int32) (int, int, error) {
	pageValue := int(page)
	if pageValue == 0 {
		pageValue = 1
	}
	pageSizeValue := int(pageSize)
	if pageSizeValue == 0 {
		pageSizeValue = 20
	}
	if pageValue < 1 || pageSizeValue < 1 || pageSizeValue > 100 {
		return 0, 0, biz.ErrPartnerInvalidArgument
	}
	return pageValue, pageSizeValue, nil
}

func partnerRoleTypeFromAPI(value v1.PartnerRoleType) biz.PartnerRoleType {
	switch value {
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER:
		return biz.PartnerRoleCustomer
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_SUPPLIER:
		return biz.PartnerRoleSupplier
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_AGENT:
		return biz.PartnerRoleAgent
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_CARRIER:
		return biz.PartnerRoleCarrier
	default:
		return ""
	}
}

func partnerRoleTypeToAPI(value biz.PartnerRoleType) v1.PartnerRoleType {
	switch value {
	case biz.PartnerRoleCustomer:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER
	case biz.PartnerRoleSupplier:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_SUPPLIER
	case biz.PartnerRoleAgent:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_AGENT
	case biz.PartnerRoleCarrier:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_CARRIER
	default:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_UNSPECIFIED
	}
}

func partnerRolesFromAPI(items []*v1.PartnerRoleInput) []*biz.PartnerRole {
	roles := make([]*biz.PartnerRole, 0, len(items))
	for _, item := range items {
		if item == nil {
			roles = append(roles, nil)
			continue
		}
		roles = append(roles, &biz.PartnerRole{Type: partnerRoleTypeFromAPI(item.GetType()), Enabled: item.GetEnabled()})
	}
	return roles
}

func partnerContactsFromAPI(items []*v1.PartnerContactInput) []*biz.PartnerContact {
	contacts := make([]*biz.PartnerContact, 0, len(items))
	for _, item := range items {
		if item == nil {
			contacts = append(contacts, nil)
			continue
		}
		contacts = append(contacts, &biz.PartnerContact{
			Name: item.GetName(), Phone: item.GetPhone(), Email: item.GetEmail(), Note: item.GetNote(), IsPrimary: item.GetIsPrimary(),
		})
	}
	return contacts
}

func partnerAliasesFromAPI(items []*v1.PartnerAliasInput) []*biz.PartnerAlias {
	aliases := make([]*biz.PartnerAlias, 0, len(items))
	for _, item := range items {
		if item == nil {
			aliases = append(aliases, nil)
			continue
		}
		aliases = append(aliases, &biz.PartnerAlias{AliasName: item.GetAliasName(), SortOrder: int(item.GetSortOrder())})
	}
	return aliases
}

func partnerReply(ctx context.Context, value *biz.Partner) *v1.PartnerReply {
	return &v1.PartnerReply{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(value), TraceId: requestmeta.TraceID(ctx)}
}

func partnerToAPI(value *biz.Partner) *v1.Partner {
	roles := make([]*v1.PartnerRole, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, &v1.PartnerRole{
			Type: partnerRoleTypeToAPI(role.Type), Enabled: role.Enabled, Blacklisted: role.Blacklisted,
			BlacklistReason: role.BlacklistReason, BlacklistedAt: formatOptionalTime(role.BlacklistedAt), BlacklistedBy: formatOptionalUUID(role.BlacklistedBy),
		})
	}
	contacts := make([]*v1.PartnerContact, 0, len(value.Contacts))
	for _, contact := range value.Contacts {
		contacts = append(contacts, &v1.PartnerContact{
			Id: contact.ID.String(), Name: contact.Name, Phone: contact.Phone, Email: contact.Email, Note: contact.Note,
			IsPrimary: contact.IsPrimary, CreatedAt: contact.CreatedAt.Format(time.RFC3339), UpdatedAt: contact.UpdatedAt.Format(time.RFC3339),
		})
	}
	aliases := make([]*v1.PartnerAlias, 0, len(value.Aliases))
	for _, alias := range value.Aliases {
		aliases = append(aliases, &v1.PartnerAlias{
			Id: alias.ID.String(), AliasName: alias.AliasName, SortOrder: int32(alias.SortOrder),
			CreatedAt: alias.CreatedAt.Format(time.RFC3339), UpdatedAt: alias.UpdatedAt.Format(time.RFC3339),
		})
	}
	return &v1.Partner{
		Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Code: value.Code, LegalName: value.LegalName,
		UnifiedSocialCreditCode: value.UnifiedSocialCreditCode, RegisteredAddress: value.RegisteredAddress, Enabled: value.Enabled,
		Roles: roles, Contacts: contacts, Aliases: aliases,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func formatOptionalUUID(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

var _ v1.PartnerServiceServer = (*PartnerService)(nil)
