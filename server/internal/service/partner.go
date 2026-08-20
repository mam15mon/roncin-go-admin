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
		Type:     partnerTypeFromAPI(request.GetType()),
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
		Success:  true,
		Code:     0,
		Message:  "OK",
		Data:     data,
		Total:    int32(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		TraceId:  requestmeta.TraceID(ctx),
	}, nil
}

func (s *PartnerService) CreatePartner(ctx context.Context, request *v1.CreatePartnerRequest) (*v1.PartnerReply, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, &biz.Partner{
		Code:        request.GetCode(),
		Name:        request.GetName(),
		Type:        partnerTypeFromAPI(request.GetType()),
		ContactName: request.GetContactName(),
		Phone:       request.GetPhone(),
		Email:       request.GetEmail(),
		Address:     request.GetAddress(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.PartnerReply{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
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
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, partnerID, &biz.Partner{
		Name:        request.GetName(),
		Type:        partnerTypeFromAPI(request.GetType()),
		ContactName: request.GetContactName(),
		Phone:       request.GetPhone(),
		Email:       request.GetEmail(),
		Address:     request.GetAddress(),
		Enabled:     request.GetEnabled(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.PartnerReply{Success: true, Code: 0, Message: "OK", Data: partnerToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
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

func partnerTypeFromAPI(value v1.PartnerType) biz.PartnerType {
	switch value {
	case v1.PartnerType_PARTNER_TYPE_CUSTOMER:
		return biz.PartnerTypeCustomer
	case v1.PartnerType_PARTNER_TYPE_SUPPLIER:
		return biz.PartnerTypeSupplier
	case v1.PartnerType_PARTNER_TYPE_BOTH:
		return biz.PartnerTypeBoth
	default:
		return ""
	}
}

func partnerTypeToAPI(value biz.PartnerType) v1.PartnerType {
	switch value {
	case biz.PartnerTypeCustomer:
		return v1.PartnerType_PARTNER_TYPE_CUSTOMER
	case biz.PartnerTypeSupplier:
		return v1.PartnerType_PARTNER_TYPE_SUPPLIER
	case biz.PartnerTypeBoth:
		return v1.PartnerType_PARTNER_TYPE_BOTH
	default:
		return v1.PartnerType_PARTNER_TYPE_UNSPECIFIED
	}
}

func partnerToAPI(value *biz.Partner) *v1.Partner {
	return &v1.Partner{
		Id:             value.ID.String(),
		OrganizationId: value.OrganizationID.String(),
		Code:           value.Code,
		Name:           value.Name,
		Type:           partnerTypeToAPI(value.Type),
		ContactName:    value.ContactName,
		Phone:          value.Phone,
		Email:          value.Email,
		Address:        value.Address,
		Enabled:        value.Enabled,
		CreatedAt:      value.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      value.UpdatedAt.Format(time.RFC3339),
	}
}

var _ v1.PartnerServiceServer = (*PartnerService)(nil)
