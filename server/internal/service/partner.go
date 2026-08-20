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
	usecase         *biz.PartnerUsecase
	accountUsecase  *biz.PartnerAccountUsecase
	contractUsecase *biz.PartnerContractUsecase
}

func NewPartnerService(usecase *biz.PartnerUsecase, accountUsecase *biz.PartnerAccountUsecase, contractUsecase *biz.PartnerContractUsecase) *PartnerService {
	return &PartnerService{usecase: usecase, accountUsecase: accountUsecase, contractUsecase: contractUsecase}
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
