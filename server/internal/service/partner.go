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
}

func NewPartnerService(usecase *biz.PartnerUsecase, accountUsecase *biz.PartnerAccountUsecase, contractUsecase *biz.PartnerContractUsecase, settlementRuleUsecase *biz.PartnerSettlementRuleUsecase, attachmentUsecase *biz.PartnerAttachmentUsecase) *PartnerService {
	return &PartnerService{usecase: usecase, accountUsecase: accountUsecase, contractUsecase: contractUsecase, settlementRuleUsecase: settlementRuleUsecase, attachmentUsecase: attachmentUsecase}
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

func (s *PartnerService) ListPartnerAccounts(ctx context.Context, request *v1.ListPartnerAccountsRequest) (*v1.PartnerAccountListReply, error) {
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
	return &v1.PartnerAccountListReply{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerAccount(ctx context.Context, request *v1.CreatePartnerAccountRequest) (*v1.PartnerAccountReply, error) {
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
	return partnerAccountReply(ctx, created), nil
}

func (s *PartnerService) UpdatePartnerAccount(ctx context.Context, request *v1.UpdatePartnerAccountRequest) (*v1.PartnerAccountReply, error) {
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
	return partnerAccountReply(ctx, updated), nil
}

func (s *PartnerService) ListPartnerContracts(ctx context.Context, request *v1.ListPartnerContractsRequest) (*v1.PartnerContractListReply, error) {
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
	return &v1.PartnerContractListReply{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerContract(ctx context.Context, request *v1.CreatePartnerContractRequest) (*v1.PartnerContractReply, error) {
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
	return partnerContractReply(ctx, created), nil
}

func (s *PartnerService) UpdatePartnerContract(ctx context.Context, request *v1.UpdatePartnerContractRequest) (*v1.PartnerContractReply, error) {
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
	return partnerContractReply(ctx, updated), nil
}

func (s *PartnerService) ListPartnerSettlementRules(ctx context.Context, request *v1.ListPartnerSettlementRulesRequest) (*v1.PartnerSettlementRuleListReply, error) {
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
	return &v1.PartnerSettlementRuleListReply{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerSettlementRule(ctx context.Context, request *v1.CreatePartnerSettlementRuleRequest) (*v1.PartnerSettlementRuleReply, error) {
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
	return partnerSettlementRuleReply(ctx, created), nil
}

func (s *PartnerService) UpdatePartnerSettlementRule(ctx context.Context, request *v1.UpdatePartnerSettlementRuleRequest) (*v1.PartnerSettlementRuleReply, error) {
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
	return partnerSettlementRuleReply(ctx, updated), nil
}

func (s *PartnerService) ListPartnerAttachments(ctx context.Context, request *v1.ListPartnerAttachmentsRequest) (*v1.PartnerAttachmentListReply, error) {
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
	return &v1.PartnerAttachmentListReply{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) RegisterPartnerAttachment(ctx context.Context, request *v1.RegisterPartnerAttachmentRequest) (*v1.PartnerAttachmentReply, error) {
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
	return partnerAttachmentReply(ctx, created), nil
}

func (s *PartnerService) ImportPartners(ctx context.Context, request *v1.ImportPartnersRequest) (*v1.PartnerImportReply, error) {
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
		})
	}
	result, err := s.usecase.Import(ctx, principal.Organization.ID, principal.UserID, biz.PartnerImportInput{
		Source: request.GetSource(), Mode: partnerImportModeFromAPI(request.GetMode()), Items: items,
	})
	if err != nil {
		return nil, err
	}
	return &v1.PartnerImportReply{Success: true, Code: 0, Message: "OK", CreatedCount: int32(result.CreatedCount), UpdatedCount: int32(result.UpdatedCount), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) ExportPartners(ctx context.Context, request *v1.ExportPartnersRequest) (*v1.PartnerExportReply, error) {
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
	return &v1.PartnerExportReply{Success: true, Code: 0, Message: "OK", Data: items, TraceId: requestmeta.TraceID(ctx)}, nil
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
		Currency: value.GetCurrency(), InvoiceTitle: value.GetInvoiceTitle(), UnifiedSocialCreditCode: value.GetUnifiedSocialCreditCode(),
		BillingAddress: value.GetBillingAddress(), BillingPhone: value.GetBillingPhone(), BankName: value.GetBankName(), BankAccount: value.GetBankAccount(),
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

func partnerAccountReply(ctx context.Context, value *biz.PartnerAccount) *v1.PartnerAccountReply {
	return &v1.PartnerAccountReply{Success: true, Code: 0, Message: "OK", Data: partnerAccountToAPI(value), TraceId: requestmeta.TraceID(ctx)}
}

func partnerContractReply(ctx context.Context, value *biz.PartnerContract) *v1.PartnerContractReply {
	return &v1.PartnerContractReply{Success: true, Code: 0, Message: "OK", Data: partnerContractToAPI(value), TraceId: requestmeta.TraceID(ctx)}
}

func partnerAccountToAPI(value *biz.PartnerAccount) *v1.PartnerAccount {
	return &v1.PartnerAccount{
		Id: value.ID.String(), PartnerRoleId: value.PartnerRoleID.String(), AccountType: value.AccountType,
		Currency: value.Currency, InvoiceTitle: value.InvoiceTitle, UnifiedSocialCreditCode: value.UnifiedSocialCreditCode,
		BillingAddress: value.BillingAddress, BillingPhone: value.BillingPhone, BankName: value.BankName, BankAccount: value.BankAccount,
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
	return result
}

func partnerSettlementRuleToAPI(value *biz.PartnerSettlementRule) *v1.PartnerSettlementRule {
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
	return result
}

func partnerSettlementRuleReply(ctx context.Context, value *biz.PartnerSettlementRule) *v1.PartnerSettlementRuleReply {
	return &v1.PartnerSettlementRuleReply{Success: true, Code: 0, Message: "OK", Data: partnerSettlementRuleToAPI(value), TraceId: requestmeta.TraceID(ctx)}
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

func partnerAttachmentReply(ctx context.Context, value *biz.PartnerAttachment) *v1.PartnerAttachmentReply {
	return &v1.PartnerAttachmentReply{Success: true, Code: 0, Message: "OK", Data: partnerAttachmentToAPI(value), TraceId: requestmeta.TraceID(ctx)}
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
