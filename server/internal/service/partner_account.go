package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerAccounts(ctx context.Context, request *v1.ListPartnerAccountsRequest) (*v1.ListPartnerAccountsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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

func partnerAccountToAPI(value *biz.PartnerAccount) *v1.PartnerAccount {
	return &v1.PartnerAccount{
		Id: value.ID.String(), PartnerRoleId: value.PartnerRoleID.String(), AccountType: value.AccountType,
		Currency: value.Currency, BankName: value.BankName, BankAccount: value.BankAccount,
		SwiftCode: value.SwiftCode, IsDefault: value.IsDefault, Status: partnerAccountStatusToAPI(value.Status), Remark: value.Remark,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
}
