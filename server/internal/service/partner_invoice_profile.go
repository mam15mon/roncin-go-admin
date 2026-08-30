package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerInvoiceProfiles(ctx context.Context, request *v1.ListPartnerInvoiceProfilesRequest) (*v1.ListPartnerInvoiceProfilesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
