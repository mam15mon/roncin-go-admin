package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

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
	filter := biz.PartnerAccountFilter{}
	if request.Enabled != nil {
		value := request.GetEnabled()
		filter.Enabled = &value
	}
	if request.Usage != nil {
		filter.Usage = partnerAccountUsageFromAPI(request.GetUsage())
	}
	filter.Currency = request.GetCurrency()
	items, err := s.accountUsecase.List(ctx, principal.Organization.ID, partnerID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerAccount, 0, len(items))
	for _, item := range items {
		data = append(data, partnerAccountToAPI(item))
	}
	return okList(ctx, &v1.ListPartnerAccountsResponse{Data: data}), nil
}

func (s *PartnerService) CreatePartnerAccount(ctx context.Context, request *v1.CreatePartnerAccountRequest) (*v1.CreatePartnerAccountResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil || request.GetAccount() == nil || request.GetAccount().Enabled == nil {
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	created, err := s.accountUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, partnerAccountFromAPI(request.GetAccount()))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreatePartnerAccountResponse{Data: partnerAccountToAPI(created)}), nil
}

func (s *PartnerService) UpdatePartnerAccount(ctx context.Context, request *v1.UpdatePartnerAccountRequest) (*v1.UpdatePartnerAccountResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	if partnerErr != nil || idErr != nil || request.GetAccount() == nil || request.GetAccount().Enabled == nil {
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	updated, err := s.accountUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, partnerAccountFromAPI(request.GetAccount()))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdatePartnerAccountResponse{Data: partnerAccountToAPI(updated)}), nil
}

func partnerAccountFromAPI(value *v1.PartnerAccountInput) *biz.PartnerAccount {
	return &biz.PartnerAccount{
		Name: value.GetName(), AccountHolder: value.GetAccountHolder(), Currency: value.GetCurrency(), BankName: value.GetBankName(), AccountNo: value.GetAccountNo(),
		SwiftCode: value.GetSwiftCode(), Usage: partnerAccountUsageFromAPI(value.GetUsage()), IsDefaultReceivable: value.GetIsDefaultReceivable(), IsDefaultPayable: value.GetIsDefaultPayable(), Enabled: value.GetEnabled(), Remark: value.GetRemark(),
	}
}

func partnerAccountUsageFromAPI(value v1.PartnerAccountUsage) biz.PartnerAccountUsage {
	switch value {
	case v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_RECEIVABLE:
		return biz.PartnerAccountUsageReceivable
	case v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_PAYABLE:
		return biz.PartnerAccountUsagePayable
	case v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_BOTH:
		return biz.PartnerAccountUsageBoth
	default:
		return ""
	}
}

func partnerAccountUsageToAPI(value biz.PartnerAccountUsage) v1.PartnerAccountUsage {
	switch value {
	case biz.PartnerAccountUsageReceivable:
		return v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_RECEIVABLE
	case biz.PartnerAccountUsagePayable:
		return v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_PAYABLE
	case biz.PartnerAccountUsageBoth:
		return v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_BOTH
	default:
		return v1.PartnerAccountUsage_PARTNER_ACCOUNT_USAGE_UNSPECIFIED
	}
}

func partnerAccountToAPI(value *biz.PartnerAccount) *v1.PartnerAccount {
	return &v1.PartnerAccount{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), Name: value.Name, AccountHolder: value.AccountHolder,
		Currency: value.Currency, BankName: value.BankName, AccountNo: value.AccountNo,
		SwiftCode: value.SwiftCode, Usage: partnerAccountUsageToAPI(value.Usage), IsDefaultReceivable: value.IsDefaultReceivable, IsDefaultPayable: value.IsDefaultPayable, Enabled: value.Enabled, Remark: value.Remark,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
}
