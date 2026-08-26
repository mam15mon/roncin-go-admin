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
	usecase               *biz.PartnerUsecase
	accountUsecase        *biz.PartnerAccountUsecase
	contractUsecase       *biz.PartnerContractUsecase
	settlementRuleUsecase *biz.PartnerSettlementRuleUsecase
	attachmentUsecase     *biz.PartnerAttachmentUsecase
	shippingPresetUsecase *biz.PartnerShippingPresetUsecase
	invoiceProfileUsecase *biz.PartnerInvoiceProfileUsecase
}

func NewPartnerService(usecase *biz.PartnerUsecase, accountUsecase *biz.PartnerAccountUsecase, contractUsecase *biz.PartnerContractUsecase, settlementRuleUsecase *biz.PartnerSettlementRuleUsecase, attachmentUsecase *biz.PartnerAttachmentUsecase, shippingPresetUsecase *biz.PartnerShippingPresetUsecase, invoiceProfileUsecase *biz.PartnerInvoiceProfileUsecase) *PartnerService {
	return &PartnerService{usecase: usecase, accountUsecase: accountUsecase, contractUsecase: contractUsecase, settlementRuleUsecase: settlementRuleUsecase, attachmentUsecase: attachmentUsecase, shippingPresetUsecase: shippingPresetUsecase, invoiceProfileUsecase: invoiceProfileUsecase}
}

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

func (s *PartnerService) ListPartnerInvoiceProfiles(ctx context.Context, request *v1.ListPartnerInvoiceProfilesRequest) (*v1.ListPartnerInvoiceProfilesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerInvoiceProfileInvalidArgument
	}
	items, err := s.invoiceProfileUsecase.List(ctx, principal.Organization.ID, partnerID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerInvoiceProfile, 0, len(items))
	for _, item := range items {
		data = append(data, partnerInvoiceProfileToAPI(item))
	}
	return &v1.ListPartnerInvoiceProfilesResponse{Success: true, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerInvoiceProfile(ctx context.Context, request *v1.CreatePartnerInvoiceProfileRequest) (*v1.CreatePartnerInvoiceProfileResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerInvoiceProfileInvalidArgument
	}
	item, err := s.invoiceProfileUsecase.Create(ctx, principal.Organization.ID, principal.UserID, biz.CreatePartnerInvoiceProfileInput{PartnerID: partnerID, InvoiceTitle: request.GetInvoiceTitle(), TaxpayerIdentificationNo: request.GetTaxpayerIdentificationNo(), RegisteredAddress: request.GetRegisteredAddress(), RegisteredPhone: request.GetRegisteredPhone(), BankName: request.GetBankName(), BankAccount: request.GetBankAccount(), DefaultInvoiceType: biz.FinanceInvoiceType(request.GetDefaultInvoiceType()), IsDefault: request.GetIsDefault()})
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerInvoiceProfileResponse{Success: true, Message: "OK", Data: partnerInvoiceProfileToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerInvoiceProfile(ctx context.Context, request *v1.UpdatePartnerInvoiceProfileRequest) (*v1.UpdatePartnerInvoiceProfileResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	profileID, profileErr := uuid.Parse(request.GetId())
	if partnerErr != nil || profileErr != nil {
		return nil, biz.ErrPartnerInvoiceProfileInvalidArgument
	}
	item, err := s.invoiceProfileUsecase.Update(ctx, principal.Organization.ID, principal.UserID, biz.UpdatePartnerInvoiceProfileInput{PartnerID: partnerID, ID: profileID, InvoiceTitle: request.GetInvoiceTitle(), TaxpayerIdentificationNo: request.GetTaxpayerIdentificationNo(), RegisteredAddress: request.GetRegisteredAddress(), RegisteredPhone: request.GetRegisteredPhone(), BankName: request.GetBankName(), BankAccount: request.GetBankAccount(), DefaultInvoiceType: biz.FinanceInvoiceType(request.GetDefaultInvoiceType()), IsDefault: request.GetIsDefault(), Enabled: request.GetEnabled(), ExpectedVersion: request.GetExpectedVersion()})
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerInvoiceProfileResponse{Success: true, Message: "OK", Data: partnerInvoiceProfileToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func partnerInvoiceProfileToAPI(item *biz.PartnerInvoiceProfile) *v1.PartnerInvoiceProfile {
	return &v1.PartnerInvoiceProfile{Id: item.ID.String(), PartnerId: item.PartnerID.String(), InvoiceTitle: item.InvoiceTitle, TaxpayerIdentificationNo: item.TaxpayerIdentificationNo, RegisteredAddress: item.RegisteredAddress, RegisteredPhone: item.RegisteredPhone, BankName: item.BankName, BankAccount: item.BankAccount, DefaultInvoiceType: string(item.DefaultInvoiceType), Version: item.Version, IsDefault: item.IsDefault, Enabled: item.Enabled, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
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
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.usecase.ListAssignmentOptions(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerAssignmentOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.PartnerAssignmentOption{
			UserId: item.UserID.String(), DisplayName: item.DisplayName,
			OrganizationId: item.OrganizationID.String(), OrganizationName: item.OrganizationName,
			MembershipEnabled: item.MembershipEnabled,
		})
	}
	return &v1.ListPartnerAssignmentOptionsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
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

func (s *PartnerService) ListPartnerAccounts(ctx context.Context, request *v1.ListPartnerAccountsRequest) (*v1.ListPartnerAccountsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	var enabled *bool
	if request.Enabled != nil {
		value := request.GetEnabled()
		enabled = &value
	}
	items, err := s.accountUsecase.List(ctx, principal.Organization.ID, partnerID, enabled)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerAccount, 0, len(items))
	for _, item := range items {
		data = append(data, partnerAccountToAPI(item))
	}
	return &v1.ListPartnerAccountsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerAccount(ctx context.Context, request *v1.CreatePartnerAccountRequest) (*v1.CreatePartnerAccountResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil || request.GetAccount() == nil {
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	created, err := s.accountUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, partnerAccountFromAPI(request.GetAccount()))
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerAccountResponse{Success: true, Code: 0, Message: "OK", Data: partnerAccountToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerAccount(ctx context.Context, request *v1.UpdatePartnerAccountRequest) (*v1.UpdatePartnerAccountResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	if partnerErr != nil || idErr != nil || request.GetAccount() == nil {
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	updated, err := s.accountUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, partnerAccountFromAPI(request.GetAccount()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerAccountResponse{Success: true, Code: 0, Message: "OK", Data: partnerAccountToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ListPartnerContracts(ctx context.Context, request *v1.ListPartnerContractsRequest) (*v1.ListPartnerContractsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerContractInvalidArgument
	}
	var status *biz.PartnerContractStatus
	if request.Status != nil {
		value := partnerContractStatusFromAPI(request.GetStatus())
		status = &value
	}
	items, err := s.contractUsecase.List(ctx, principal.Organization.ID, partnerID, status)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerContract, 0, len(items))
	for _, item := range items {
		data = append(data, partnerContractToAPI(item))
	}
	return &v1.ListPartnerContractsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerContract(ctx context.Context, request *v1.CreatePartnerContractRequest) (*v1.CreatePartnerContractResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil || request.GetContract() == nil {
		return nil, biz.ErrPartnerContractInvalidArgument
	}
	input := request.GetContract()
	startDate, endDate, err := parseContractDates(input.GetStartDate(), input.GetEndDate())
	if err != nil {
		return nil, err
	}
	created, err := s.contractUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, &biz.PartnerContract{
		ContractNo: input.GetContractNo(), Name: input.GetName(), Status: partnerContractStatusFromAPI(input.GetStatus()),
		StartDate: startDate, EndDate: endDate, PaymentTerms: input.GetPaymentTerms(), DisputeResolution: input.GetDisputeResolution(), OtherNotes: input.GetOtherNotes(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerContractResponse{Success: true, Code: 0, Message: "OK", Data: partnerContractToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerContract(ctx context.Context, request *v1.UpdatePartnerContractRequest) (*v1.UpdatePartnerContractResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	if partnerErr != nil || idErr != nil || request.GetContract() == nil {
		return nil, biz.ErrPartnerContractInvalidArgument
	}
	input := request.GetContract()
	startDate, endDate, err := parseContractDates(input.GetStartDate(), input.GetEndDate())
	if err != nil {
		return nil, err
	}
	updated, err := s.contractUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, &biz.PartnerContract{
		Name: input.GetName(), Status: partnerContractStatusFromAPI(input.GetStatus()),
		StartDate: startDate, EndDate: endDate, PaymentTerms: input.GetPaymentTerms(), DisputeResolution: input.GetDisputeResolution(), OtherNotes: input.GetOtherNotes(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerContractResponse{Success: true, Code: 0, Message: "OK", Data: partnerContractToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ListPartnerSettlementRules(ctx context.Context, request *v1.ListPartnerSettlementRulesRequest) (*v1.ListPartnerSettlementRulesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	roleType := partnerRoleTypeFromAPI(request.GetRoleType())
	if err != nil || !roleType.Valid() {
		return nil, biz.ErrPartnerSettlementRuleInvalidArgument
	}
	items, err := s.settlementRuleUsecase.List(ctx, principal.Organization.ID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerSettlementRule, 0, len(items))
	for _, item := range items {
		data = append(data, partnerSettlementRuleToAPI(item))
	}
	return &v1.ListPartnerSettlementRulesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerSettlementRule(ctx context.Context, request *v1.CreatePartnerSettlementRuleRequest) (*v1.CreatePartnerSettlementRuleResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	roleType := partnerRoleTypeFromAPI(request.GetRoleType())
	if err != nil || !roleType.Valid() || request.GetRule() == nil {
		return nil, biz.ErrPartnerSettlementRuleInvalidArgument
	}
	created, err := s.settlementRuleUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, roleType, partnerSettlementRuleFromAPI(request.GetRule()))
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerSettlementRuleResponse{Success: true, Code: 0, Message: "OK", Data: partnerSettlementRuleToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerSettlementRule(ctx context.Context, request *v1.UpdatePartnerSettlementRuleRequest) (*v1.UpdatePartnerSettlementRuleResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	roleType := partnerRoleTypeFromAPI(request.GetRoleType())
	if partnerErr != nil || idErr != nil || !roleType.Valid() || request.GetRule() == nil {
		return nil, biz.ErrPartnerSettlementRuleInvalidArgument
	}
	updated, err := s.settlementRuleUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, roleType, partnerSettlementRuleFromAPI(request.GetRule()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerSettlementRuleResponse{Success: true, Code: 0, Message: "OK", Data: partnerSettlementRuleToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ListPartnerAttachments(ctx context.Context, request *v1.ListPartnerAttachmentsRequest) (*v1.ListPartnerAttachmentsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerAttachmentInvalidArgument
	}
	items, err := s.attachmentUsecase.List(ctx, principal.Organization.ID, partnerID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerAttachment, 0, len(items))
	for _, item := range items {
		data = append(data, partnerAttachmentToAPI(item))
	}
	return &v1.ListPartnerAttachmentsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) RegisterPartnerAttachment(ctx context.Context, request *v1.RegisterPartnerAttachmentRequest) (*v1.RegisterPartnerAttachmentResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerAttachmentInvalidArgument
	}
	created, err := s.attachmentUsecase.Register(ctx, principal.Organization.ID, principal.UserID, partnerID, &biz.PartnerAttachment{
		IdempotencyKey: request.GetIdempotencyKey(), FileName: request.GetFileName(), MIMEType: request.GetMimeType(), FileSize: request.GetFileSize(), ObjectKey: request.GetObjectKey(), Checksum: request.GetChecksum(),
	})
	if err != nil {
		return nil, err
	}
	return partnerAttachmentResponse(ctx, created), nil
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
	options := biz.PartnerListOptions{Page: 1, PageSize: 100, Keyword: request.GetKeyword(), Role: partnerRoleTypeFromAPI(request.GetRole())}
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

func (s *PartnerService) ListPartnerShippingPresets(ctx context.Context, request *v1.ListPartnerShippingPresetsRequest) (*v1.ListPartnerShippingPresetsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	options := biz.PartnerShippingPresetListOptions{}
	if request.PresetType != nil {
		options.PresetType = partnerShippingPresetTypeFromAPI(request.GetPresetType())
	}
	if request.Enabled != nil {
		enabled := request.GetEnabled()
		options.Enabled = &enabled
	}
	items, err := s.shippingPresetUsecase.List(ctx, principal.Organization.ID, partnerID, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerShippingPreset, 0, len(items))
	for _, item := range items {
		data = append(data, partnerShippingPresetToAPI(item))
	}
	return &v1.ListPartnerShippingPresetsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
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

func (s *PartnerService) CreatePartnerShippingPreset(ctx context.Context, request *v1.CreatePartnerShippingPresetRequest) (*v1.CreatePartnerShippingPresetResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	created, err := s.shippingPresetUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, partnerShippingPresetFromAPI(request.GetPreset()))
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerShippingPresetResponse{Success: true, Code: 0, Message: "OK", Data: partnerShippingPresetToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerShippingPreset(ctx context.Context, request *v1.UpdatePartnerShippingPresetRequest) (*v1.UpdatePartnerShippingPresetResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	if partnerErr != nil || idErr != nil {
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	updated, err := s.shippingPresetUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, partnerShippingPresetFromAPI(request.GetPreset()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerShippingPresetResponse{Success: true, Code: 0, Message: "OK", Data: partnerShippingPresetToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func parseContractDates(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Time{}, time.Time{}, biz.ErrPartnerContractInvalidArgument
	}
	endDate, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return time.Time{}, time.Time{}, biz.ErrPartnerContractInvalidArgument
	}
	return startDate, endDate, nil
}

func partnerImportModeFromAPI(value v1.PartnerImportMode) biz.PartnerImportMode {
	switch value {
	case v1.PartnerImportMode_PARTNER_IMPORT_MODE_CREATE_ONLY:
		return biz.PartnerImportCreateOnly
	case v1.PartnerImportMode_PARTNER_IMPORT_MODE_UPSERT:
		return biz.PartnerImportUpsert
	default:
		return ""
	}
}

func partnerAccountFromAPI(value *v1.PartnerAccountInput) *biz.PartnerAccount {
	return &biz.PartnerAccount{
		Currency: value.GetCurrency(), BankName: value.GetBankName(), BankAccount: value.GetBankAccount(),
		SwiftCode: value.GetSwiftCode(), IsDefault: value.GetIsDefault(), Status: partnerAccountStatusFromAPI(value.GetStatus()), Remark: value.GetRemark(),
	}
}

func partnerAccountStatusFromAPI(value v1.PartnerAccountStatus) biz.PartnerAccountStatus {
	if value == v1.PartnerAccountStatus_PARTNER_ACCOUNT_STATUS_ACTIVE {
		return biz.PartnerAccountActive
	}
	if value == v1.PartnerAccountStatus_PARTNER_ACCOUNT_STATUS_INACTIVE {
		return biz.PartnerAccountInactive
	}
	return ""
}

func partnerAccountStatusToAPI(value biz.PartnerAccountStatus) v1.PartnerAccountStatus {
	if value == biz.PartnerAccountActive {
		return v1.PartnerAccountStatus_PARTNER_ACCOUNT_STATUS_ACTIVE
	}
	if value == biz.PartnerAccountInactive {
		return v1.PartnerAccountStatus_PARTNER_ACCOUNT_STATUS_INACTIVE
	}
	return v1.PartnerAccountStatus_PARTNER_ACCOUNT_STATUS_UNSPECIFIED
}

func partnerContractStatusFromAPI(value v1.PartnerContractStatus) biz.PartnerContractStatus {
	switch value {
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_PENDING:
		return biz.PartnerContractPending
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_ACTIVE:
		return biz.PartnerContractActive
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_EXPIRED:
		return biz.PartnerContractExpired
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_TERMINATED:
		return biz.PartnerContractTerminated
	default:
		return ""
	}
}

func partnerContractStatusToAPI(value biz.PartnerContractStatus) v1.PartnerContractStatus {
	switch value {
	case biz.PartnerContractPending:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_PENDING
	case biz.PartnerContractActive:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_ACTIVE
	case biz.PartnerContractExpired:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_EXPIRED
	case biz.PartnerContractTerminated:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_TERMINATED
	default:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_UNSPECIFIED
	}
}

func partnerAccountToAPI(value *biz.PartnerAccount) *v1.PartnerAccount {
	return &v1.PartnerAccount{
		Id: value.ID.String(), PartnerRoleId: value.PartnerRoleID.String(), AccountType: value.AccountType,
		Currency: value.Currency, BankName: value.BankName, BankAccount: value.BankAccount,
		SwiftCode: value.SwiftCode, IsDefault: value.IsDefault, Status: partnerAccountStatusToAPI(value.Status), Remark: value.Remark,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
}

func partnerContractToAPI(value *biz.PartnerContract) *v1.PartnerContract {
	return &v1.PartnerContract{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), ContractNo: value.ContractNo, Name: value.Name,
		Status: partnerContractStatusToAPI(value.Status), StartDate: value.StartDate.Format(time.RFC3339), EndDate: value.EndDate.Format(time.RFC3339),
		PaymentTerms: value.PaymentTerms, DisputeResolution: value.DisputeResolution, OtherNotes: value.OtherNotes,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
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

func partnerShippingPresetTypeFromAPI(value v1.PartnerShippingPresetType) biz.PartnerShippingPresetType {
	switch value {
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_SHIPPER:
		return biz.PartnerShippingPresetShipper
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_CONSIGNEE:
		return biz.PartnerShippingPresetConsignee
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_NOTIFY_PARTY:
		return biz.PartnerShippingPresetNotifyParty
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_ENGLISH_CARGO_NAME:
		return biz.PartnerShippingPresetEnglishCargoName
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_HS_CODE:
		return biz.PartnerShippingPresetHSCode
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_MARKS:
		return biz.PartnerShippingPresetMarks
	default:
		return ""
	}
}

func partnerShippingPresetTypeToAPI(value biz.PartnerShippingPresetType) v1.PartnerShippingPresetType {
	switch value {
	case biz.PartnerShippingPresetShipper:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_SHIPPER
	case biz.PartnerShippingPresetConsignee:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_CONSIGNEE
	case biz.PartnerShippingPresetNotifyParty:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_NOTIFY_PARTY
	case biz.PartnerShippingPresetEnglishCargoName:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_ENGLISH_CARGO_NAME
	case biz.PartnerShippingPresetHSCode:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_HS_CODE
	case biz.PartnerShippingPresetMarks:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_MARKS
	default:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_UNSPECIFIED
	}
}

func partnerShippingPresetFromAPI(value *v1.PartnerShippingPresetInput) *biz.PartnerShippingPreset {
	if value == nil {
		return nil
	}
	result := &biz.PartnerShippingPreset{
		PresetType: partnerShippingPresetTypeFromAPI(value.GetPresetType()), Title: value.GetTitle(),
		IsDefault: value.GetIsDefault(), SortOrder: int(value.GetSortOrder()), Remark: value.GetRemark(), Enabled: value.GetEnabled(),
	}
	if party := value.GetParty(); party != nil {
		result.Party = &biz.PartnerShippingPartyPayload{
			CompanyName: party.GetCompanyName(), Address: party.GetAddress(), ContactName: party.GetContactName(),
			Phone: party.GetPhone(), Email: party.GetEmail(), CountryCode: party.GetCountryCode(), TaxIdentifier: party.GetTaxIdentifier(),
		}
	}
	if textPayload := value.GetText(); textPayload != nil {
		result.Text = &biz.PartnerShippingTextPayload{Content: textPayload.GetContent(), Code: textPayload.GetCode()}
	}
	return result
}

func partnerShippingPresetToAPI(value *biz.PartnerShippingPreset) *v1.PartnerShippingPreset {
	result := &v1.PartnerShippingPreset{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), PresetType: partnerShippingPresetTypeToAPI(value.PresetType),
		Title: value.Title, IsDefault: value.IsDefault, SortOrder: int32(value.SortOrder), Remark: value.Remark, Enabled: value.Enabled,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
	if value.Party != nil {
		result.Payload = &v1.PartnerShippingPreset_Party{Party: &v1.PartnerShippingPartyPayload{
			CompanyName: value.Party.CompanyName, Address: value.Party.Address, ContactName: value.Party.ContactName,
			Phone: value.Party.Phone, Email: value.Party.Email, CountryCode: value.Party.CountryCode, TaxIdentifier: value.Party.TaxIdentifier,
		}}
	}
	if value.Text != nil {
		result.Payload = &v1.PartnerShippingPreset_Text{Text: &v1.PartnerShippingTextPayload{Content: value.Text.Content, Code: value.Text.Code}}
	}
	return result
}

func partnerAuditLogToAPI(value *biz.PartnerAuditLog) *v1.PartnerAuditLog {
	result := &v1.PartnerAuditLog{
		Id: value.Log.ID.String(), UserDisplayName: value.UserDisplayName, Action: value.Log.Action,
		Result: value.Log.Result, TraceId: value.Log.TraceID, Details: value.Log.Details, CreatedAt: value.Log.CreatedAt.Format(time.RFC3339),
	}
	if value.Log.UserID != nil {
		userID := value.Log.UserID.String()
		result.UserId = &userID
	}
	return result
}

func partnerRoleTypeFromAPI(value v1.PartnerRoleType) biz.PartnerRoleType {
	switch value {
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER:
		return biz.PartnerRoleCustomer
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_SUPPLIER:
		return biz.PartnerRoleSupplier
	case v1.PartnerRoleType_PARTNER_ROLE_TYPE_FOREIGN_AGENT:
		return biz.PartnerRoleForeignAgent
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
	case biz.PartnerRoleForeignAgent:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_FOREIGN_AGENT
	case biz.PartnerRoleCarrier:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_CARRIER
	default:
		return v1.PartnerRoleType_PARTNER_ROLE_TYPE_UNSPECIFIED
	}
}

func partnerCustomerTypeFromAPI(value v1.PartnerCustomerType) biz.PartnerCustomerType {
	switch value {
	case v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_DIRECT:
		return biz.PartnerCustomerDirect
	case v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_PEER:
		return biz.PartnerCustomerPeer
	default:
		return ""
	}
}

func partnerCustomerTypeToAPI(value biz.PartnerCustomerType) v1.PartnerCustomerType {
	switch value {
	case biz.PartnerCustomerDirect:
		return v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_DIRECT
	case biz.PartnerCustomerPeer:
		return v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_PEER
	default:
		return v1.PartnerCustomerType_PARTNER_CUSTOMER_TYPE_UNSPECIFIED
	}
}

func partnerBusinessTypeFromAPI(value v1.PartnerBusinessType) biz.PartnerBusinessType {
	switch value {
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SE:
		return biz.PartnerBusinessSE
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SI:
		return biz.PartnerBusinessSI
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AE:
		return biz.PartnerBusinessAE
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AI:
		return biz.PartnerBusinessAI
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_LAND:
		return biz.PartnerBusinessLand
	case v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_RAIL:
		return biz.PartnerBusinessRail
	default:
		return ""
	}
}

func partnerBusinessTypeToAPI(value biz.PartnerBusinessType) v1.PartnerBusinessType {
	switch value {
	case biz.PartnerBusinessSE:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SE
	case biz.PartnerBusinessSI:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_SI
	case biz.PartnerBusinessAE:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AE
	case biz.PartnerBusinessAI:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_AI
	case biz.PartnerBusinessLand:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_LAND
	case biz.PartnerBusinessRail:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_RAIL
	default:
		return v1.PartnerBusinessType_PARTNER_BUSINESS_TYPE_UNSPECIFIED
	}
}

func partnerAssignmentRoleFromAPI(value v1.PartnerAssignmentRole) biz.PartnerAssignmentRole {
	switch value {
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CREATOR:
		return biz.PartnerAssignmentCreator
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_OPERATOR:
		return biz.PartnerAssignmentOperator
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_SALES:
		return biz.PartnerAssignmentSales
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE:
		return biz.PartnerAssignmentCustomerService
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_DOCUMENT:
		return biz.PartnerAssignmentDocument
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_COMMERCIAL:
		return biz.PartnerAssignmentCommercial
	case v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT:
		return biz.PartnerAssignmentInternalContact
	default:
		return ""
	}
}

func partnerAssignmentRoleToAPI(value biz.PartnerAssignmentRole) v1.PartnerAssignmentRole {
	switch value {
	case biz.PartnerAssignmentCreator:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CREATOR
	case biz.PartnerAssignmentOperator:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_OPERATOR
	case biz.PartnerAssignmentSales:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_SALES
	case biz.PartnerAssignmentCustomerService:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE
	case biz.PartnerAssignmentDocument:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_DOCUMENT
	case biz.PartnerAssignmentCommercial:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_COMMERCIAL
	case biz.PartnerAssignmentInternalContact:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT
	default:
		return v1.PartnerAssignmentRole_PARTNER_ASSIGNMENT_ROLE_UNSPECIFIED
	}
}

func partnerRolesFromAPI(items []*v1.PartnerRoleInput) []*biz.PartnerRole {
	roles := make([]*biz.PartnerRole, 0, len(items))
	for _, item := range items {
		if item == nil {
			roles = append(roles, nil)
			continue
		}
		roles = append(roles, &biz.PartnerRole{
			Type: partnerRoleTypeFromAPI(item.GetType()), Enabled: item.GetEnabled(),
			SettlementRule: partnerSettlementRuleFromAPI(item.GetSettlementRule()),
		})
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

func partnerProfileFromAPI(value *v1.PartnerProfile) *biz.PartnerProfile {
	if value == nil {
		return nil
	}
	customerTypes := make([]biz.PartnerCustomerType, 0, len(value.GetCustomerTypes()))
	for _, item := range value.GetCustomerTypes() {
		customerTypes = append(customerTypes, partnerCustomerTypeFromAPI(item))
	}
	businessTypes := make([]biz.PartnerBusinessType, 0, len(value.GetBusinessTypes()))
	for _, item := range value.GetBusinessTypes() {
		businessTypes = append(businessTypes, partnerBusinessTypeFromAPI(item))
	}
	return &biz.PartnerProfile{
		NameEN: value.GetNameEn(), AddressEN: value.GetAddressEn(), CountryCode: value.GetCountryCode(),
		ProvinceCode: value.GetProvinceCode(), CityCode: value.GetCityCode(), DistrictCode: value.GetDistrictCode(),
		AddressDetail: value.GetAddressDetail(), Nature: value.GetNature(), DevelopmentMethod: value.GetDevelopmentMethod(),
		CustomerTypes: customerTypes, BusinessTypes: businessTypes, Remark: value.GetRemark(),
	}
}

func partnerAssignmentsFromAPI(items []*v1.PartnerAssignmentInput) []*biz.PartnerAssignment {
	result := make([]*biz.PartnerAssignment, 0, len(items))
	for _, item := range items {
		if item == nil {
			result = append(result, nil)
			continue
		}
		userID, _ := uuid.Parse(item.GetUserId())
		organizationID, _ := uuid.Parse(item.GetOrganizationId())
		result = append(result, &biz.PartnerAssignment{
			Role: partnerAssignmentRoleFromAPI(item.GetRole()), UserID: userID, OrganizationID: organizationID,
		})
	}
	return result
}

func partnerToAPI(value *biz.Partner) *v1.Partner {
	roles := make([]*v1.PartnerRole, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, &v1.PartnerRole{
			Type: partnerRoleTypeToAPI(role.Type), Enabled: role.Enabled, Blacklisted: role.Blacklisted,
			BlacklistReason: role.BlacklistReason, BlacklistedAt: formatOptionalTime(role.BlacklistedAt), BlacklistedBy: formatOptionalUUID(role.BlacklistedBy),
			SettlementRule: partnerSettlementRuleToAPI(role.SettlementRule),
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
		Profile: partnerProfileToAPI(value.Profile), Assignments: partnerAssignmentsToAPI(value.Assignments),
	}
}

func partnerProfileToAPI(value *biz.PartnerProfile) *v1.PartnerProfile {
	if value == nil {
		return nil
	}
	customerTypes := make([]v1.PartnerCustomerType, 0, len(value.CustomerTypes))
	for _, item := range value.CustomerTypes {
		customerTypes = append(customerTypes, partnerCustomerTypeToAPI(item))
	}
	businessTypes := make([]v1.PartnerBusinessType, 0, len(value.BusinessTypes))
	for _, item := range value.BusinessTypes {
		businessTypes = append(businessTypes, partnerBusinessTypeToAPI(item))
	}
	return &v1.PartnerProfile{
		NameEn: value.NameEN, AddressEn: value.AddressEN, CountryCode: value.CountryCode,
		ProvinceCode: value.ProvinceCode, CityCode: value.CityCode, DistrictCode: value.DistrictCode,
		AddressDetail: value.AddressDetail, Nature: value.Nature, DevelopmentMethod: value.DevelopmentMethod,
		CustomerTypes: customerTypes, BusinessTypes: businessTypes, Remark: value.Remark,
	}
}

func partnerAssignmentsToAPI(items []*biz.PartnerAssignment) []*v1.PartnerAssignment {
	result := make([]*v1.PartnerAssignment, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.PartnerAssignment{
			Id: item.ID.String(), Role: partnerAssignmentRoleToAPI(item.Role), UserId: item.UserID.String(),
			OrganizationId: item.OrganizationID.String(), CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
			SortOrder: int32(item.SortOrder),
		})
	}
	return result
}

func partnerStatementModeFromAPI(value v1.PartnerStatementMode) biz.PartnerStatementMode {
	if value == v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_SINGLE {
		return biz.PartnerStatementSingle
	}
	if value == v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_MULTI {
		return biz.PartnerStatementMulti
	}
	return ""
}

func partnerStatementModeToAPI(value biz.PartnerStatementMode) v1.PartnerStatementMode {
	if value == biz.PartnerStatementSingle {
		return v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_SINGLE
	}
	if value == biz.PartnerStatementMulti {
		return v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_MULTI
	}
	return v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_UNSPECIFIED
}

func partnerSettlementMethodFromAPI(value v1.PartnerSettlementMethod) biz.PartnerSettlementMethod {
	switch value {
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BY_TICKET:
		return biz.PartnerSettlementByTicket
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_MONTHLY:
		return biz.PartnerSettlementMonthly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_WEEKLY:
		return biz.PartnerSettlementWeekly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY:
		return biz.PartnerSettlementSemiMonthly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BI_MONTHLY:
		return biz.PartnerSettlementBiMonthly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_QUARTERLY:
		return biz.PartnerSettlementQuarterly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_DAYS_45:
		return biz.PartnerSettlementDays45
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_PREPAID:
		return biz.PartnerSettlementPrepaid
	default:
		return ""
	}
}

func partnerSettlementMethodToAPI(value biz.PartnerSettlementMethod) v1.PartnerSettlementMethod {
	switch value {
	case biz.PartnerSettlementByTicket:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BY_TICKET
	case biz.PartnerSettlementMonthly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_MONTHLY
	case biz.PartnerSettlementWeekly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_WEEKLY
	case biz.PartnerSettlementSemiMonthly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY
	case biz.PartnerSettlementBiMonthly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BI_MONTHLY
	case biz.PartnerSettlementQuarterly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_QUARTERLY
	case biz.PartnerSettlementDays45:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_DAYS_45
	case biz.PartnerSettlementPrepaid:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_PREPAID
	default:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_UNSPECIFIED
	}
}

func partnerSettlementBaseFromAPI(value v1.PartnerSettlementBase) *biz.PartnerSettlementBase {
	var result biz.PartnerSettlementBase
	switch value {
	case v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_BILL_DATE:
		result = biz.PartnerSettlementBillDate
	case v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_SAILING_DATE:
		result = biz.PartnerSettlementSailingDate
	case v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE:
		result = biz.PartnerSettlementArrivalDate
	default:
		return nil
	}
	return &result
}

func partnerSettlementBaseToAPI(value *biz.PartnerSettlementBase) *v1.PartnerSettlementBase {
	if value == nil {
		return nil
	}
	result := v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_UNSPECIFIED
	switch *value {
	case biz.PartnerSettlementBillDate:
		result = v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_BILL_DATE
	case biz.PartnerSettlementSailingDate:
		result = v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_SAILING_DATE
	case biz.PartnerSettlementArrivalDate:
		result = v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE
	}
	return &result
}

func partnerSettlementRuleFromAPI(value *v1.PartnerSettlementRuleInput) *biz.PartnerSettlementRule {
	if value == nil {
		return nil
	}
	result := &biz.PartnerSettlementRule{
		StatementMode: partnerStatementModeFromAPI(value.GetStatementMode()), SettlementMethod: partnerSettlementMethodFromAPI(value.GetSettlementMethod()),
		SettlementCurrency: value.GetSettlementCurrency(), IsActive: value.GetIsActive(),
	}
	if value.SettlementDay != nil {
		item := int(value.GetSettlementDay())
		result.SettlementDay = &item
	}
	if value.SettlementCycleDays != nil {
		item := int(value.GetSettlementCycleDays())
		result.SettlementCycleDays = &item
	}
	if value.SettlementBase != nil {
		result.SettlementBase = partnerSettlementBaseFromAPI(value.GetSettlementBase())
	}
	if value.CreditLimitMinor != nil {
		item := value.GetCreditLimitMinor()
		result.CreditLimitMinor = &item
	}
	if value.CreditCurrency != nil {
		item := value.GetCreditCurrency()
		result.CreditCurrency = &item
	}
	return result
}

func partnerSettlementRuleToAPI(value *biz.PartnerSettlementRule) *v1.PartnerSettlementRule {
	if value == nil {
		return nil
	}
	result := &v1.PartnerSettlementRule{
		Id: value.ID.String(), PartnerRoleId: value.PartnerRoleID.String(), StatementMode: partnerStatementModeToAPI(value.StatementMode),
		SettlementMethod: partnerSettlementMethodToAPI(value.SettlementMethod), SettlementCurrency: value.SettlementCurrency, IsActive: value.IsActive,
	}
	if value.SettlementDay != nil {
		item := int32(*value.SettlementDay)
		result.SettlementDay = &item
	}
	if value.SettlementCycleDays != nil {
		item := int32(*value.SettlementCycleDays)
		result.SettlementCycleDays = &item
	}
	result.SettlementBase = partnerSettlementBaseToAPI(value.SettlementBase)
	result.CreditLimitMinor = value.CreditLimitMinor
	result.CreditCurrency = value.CreditCurrency
	return result
}

func partnerAttachmentToAPI(value *biz.PartnerAttachment) *v1.PartnerAttachment {
	result := &v1.PartnerAttachment{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), IdempotencyKey: value.IdempotencyKey, FileName: value.FileName,
		MimeType: value.MIMEType, FileSize: value.FileSize, ObjectKey: value.ObjectKey, Checksum: value.Checksum,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
	if value.UploadedBy != nil {
		result.UploadedBy = value.UploadedBy.String()
	}
	return result
}

func partnerAttachmentResponse(ctx context.Context, value *biz.PartnerAttachment) *v1.RegisterPartnerAttachmentResponse {
	return &v1.RegisterPartnerAttachmentResponse{Success: true, Code: 0, Message: "OK", Data: partnerAttachmentToAPI(value), TraceId: requestmeta.TraceID(ctx)}
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
