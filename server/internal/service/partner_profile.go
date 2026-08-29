package service

import (
	"context"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) GetPartner(ctx context.Context, request *v1.GetPartnerRequest) (*v1.GetPartnerResponse, error) {
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
	return &v1.GetPartnerResponse{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ListPartners(ctx context.Context, request *v1.ListPartnersRequest) (*v1.ListPartnersResponse, error) {
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
	return &v1.ListPartnersResponse{
		Success: true, Code: 0, Message: "OK", Data: data,
		Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize), TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *PartnerService) ListPartnerAssignmentOptions(ctx context.Context, _ *v1.ListPartnerAssignmentOptionsRequest) (*v1.ListPartnerAssignmentOptionsResponse, error) {
	data, total, page, pageSize, err := s.listPartnerAssignmentOptions(ctx, biz.SelectorListOptions{Page: 1, PageSize: biz.MaxListPageSize})
	if err != nil {
		return nil, err
	}
	return &v1.ListPartnerAssignmentOptionsResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: total, Page: page, PageSize: pageSize, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) SearchPartnerAssignmentOptions(ctx context.Context, request *v1.SearchPartnerAssignmentOptionsRequest) (*v1.SearchPartnerAssignmentOptionsResponse, error) {
	page, pageSize := biz.ListPagination(int(request.GetPage()), int(request.GetPageSize()), 20)
	data, total, resultPage, resultPageSize, err := s.listPartnerAssignmentOptions(ctx, biz.SelectorListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	return &v1.SearchPartnerAssignmentOptionsResponse{Success: true, Code: 0, Message: "OK", Data: data, Total: total, Page: resultPage, PageSize: resultPageSize, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) listPartnerAssignmentOptions(ctx context.Context, options biz.SelectorListOptions) ([]*v1.PartnerAssignmentOption, int32, int32, int32, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, 0, 0, 0, biz.ErrSessionRequired
	}
	result, err := s.usecase.ListAssignmentOptions(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	data := make([]*v1.PartnerAssignmentOption, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.PartnerAssignmentOption{
			UserId: item.UserID.String(), DisplayName: item.DisplayName,
			OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName,
			MembershipEnabled: item.MembershipEnabled,
		})
	}
	return data, int32(result.Total), int32(result.Page), int32(result.PageSize), nil
}

func (s *PartnerService) CreatePartner(ctx context.Context, request *v1.CreatePartnerRequest) (*v1.CreatePartnerResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, &biz.Partner{
		Code: request.GetCode(), LegalName: request.GetLegalName(),
		UnifiedSocialCreditCode: request.GetUnifiedSocialCreditCode(), RegisteredAddress: request.GetRegisteredAddress(),
		Roles: partnerRolesFromAPI(request.GetRoles()), Contacts: partnerContactsFromAPI(request.GetContacts()), Aliases: partnerAliasesFromAPI(request.GetAliases()),
		Profile: partnerProfileFromAPI(request.GetProfile()), Assignments: partnerAssignmentsFromAPI(request.GetAssignments()),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerResponse{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartner(ctx context.Context, request *v1.UpdatePartnerRequest) (*v1.UpdatePartnerResponse, error) {
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
		Profile: partnerProfileFromAPI(request.GetProfile()), Assignments: partnerAssignmentsFromAPI(request.GetAssignments()),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerResponse{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) SetSupplierBlacklist(ctx context.Context, request *v1.SetSupplierBlacklistRequest) (*v1.SetSupplierBlacklistResponse, error) {
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
	return &v1.SetSupplierBlacklistResponse{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ImportPartners(ctx context.Context, request *v1.ImportPartnersRequest) (*v1.ImportPartnersResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items := make([]*biz.Partner, 0, len(request.GetItems()))
	for _, item := range request.GetItems() {
		if item == nil {
			return nil, biz.ErrPartnerImportInvalidArgument
		}
		items = append(items, &biz.Partner{
			Code: item.GetCode(), LegalName: item.GetLegalName(),
			UnifiedSocialCreditCode: item.GetUnifiedSocialCreditCode(), RegisteredAddress: item.GetRegisteredAddress(),
			Roles: partnerRolesFromAPI(item.GetRoles()), Contacts: partnerContactsFromAPI(item.GetContacts()), Aliases: partnerAliasesFromAPI(item.GetAliases()),
			Profile: partnerProfileFromAPI(item.GetProfile()), Assignments: partnerAssignmentsFromAPI(item.GetAssignments()),
		})
	}
	result, err := s.usecase.Import(ctx, principal.Organization.ID, principal.UserID, biz.PartnerImportInput{
		Source: request.GetSource(), Mode: partnerImportModeFromAPI(request.GetMode()), Items: items,
	})
	if err != nil {
		return nil, err
	}
	return &v1.ImportPartnersResponse{Success: true, Code: 0, Message: "OK", CreatedCount: int32(result.CreatedCount), UpdatedCount: int32(result.UpdatedCount), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ExportPartners(ctx context.Context, request *v1.ExportPartnersRequest) (*v1.ExportPartnersResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	options := biz.PartnerListOptions{Page: 1, PageSize: biz.MaxListPageSize, Keyword: request.GetKeyword(), Role: partnerRoleTypeFromAPI(request.GetRole())}
	if request.Enabled != nil {
		enabled := request.GetEnabled()
		options.Enabled = &enabled
	}
	items := make([]*v1.PartnerExportItem, 0)
	for {
		result, err := s.usecase.List(ctx, principal.Organization.ID, options)
		if err != nil {
			return nil, err
		}
		for _, item := range result.Items {
			roles := make([]v1.PartnerRoleType, 0, len(item.Roles))
			for _, role := range item.Roles {
				if role.Enabled {
					roles = append(roles, partnerRoleTypeToAPI(role.Type))
				}
			}
			items = append(items, &v1.PartnerExportItem{Code: item.Code, LegalName: item.LegalName, UnifiedSocialCreditCode: item.UnifiedSocialCreditCode, RegisteredAddress: item.RegisteredAddress, Enabled: item.Enabled, Roles: roles})
		}
		if len(result.Items) == 0 || len(items) >= result.Total {
			break
		}
		options.Page++
	}
	return &v1.ExportPartnersResponse{Success: true, Code: 0, Message: "OK", Data: items, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ListPartnerAuditLogs(ctx context.Context, request *v1.ListPartnerAuditLogsRequest) (*v1.ListPartnerAuditLogsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerInvalidArgument
	}
	page, pageSize, err := pageValues(request.GetPage(), request.GetPageSize())
	if err != nil {
		return nil, err
	}
	list, err := s.usecase.ListAuditLogs(ctx, principal.Organization.ID, partnerID, page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerAuditLog, 0, len(list.Items))
	for _, item := range list.Items {
		data = append(data, partnerAuditLogToAPI(item))
	}
	return &v1.ListPartnerAuditLogsResponse{
		Success: true, Code: 0, Message: "OK", Data: data,
		Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize), TraceId: requestmeta.TraceID(ctx),
	}, nil
}
