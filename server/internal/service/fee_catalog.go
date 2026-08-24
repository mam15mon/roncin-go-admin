package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

type FeeCatalogService struct {
	v1.UnimplementedFeeCatalogServiceServer
	usecase *biz.FeeCatalogUsecase
}

func NewFeeCatalogService(usecase *biz.FeeCatalogUsecase) *FeeCatalogService {
	return &FeeCatalogService{usecase: usecase}
}

func (s *FeeCatalogService) ListFeeSettings(ctx context.Context, _ *v1.ListFeeSettingsRequest) (*v1.ListFeeSettingsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.usecase.ListFeeSettings(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FeeSetting, 0, len(items))
	for _, item := range items {
		data = append(data, feeSettingToAPI(item))
	}
	return &v1.ListFeeSettingsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) CreateFeeSetting(ctx context.Context, request *v1.CreateFeeSettingRequest) (*v1.CreateFeeSettingResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	input, err := feeSettingInputFromAPI(request.GetFeeCode(), request.GetNameZh(), request.NameEn, request.AliasName, request.ServiceTypeId, request.GetDefaultCurrency(), request.GetBillingUnitId(), request.AbnormalCaseId, request.GetTaxRate(), request.GetTaxableServiceId(), int(request.GetSortOrder()), true)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.CreateFeeSetting(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return &v1.CreateFeeSettingResponse{Success: true, Code: 0, Message: "OK", Data: feeSettingToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) UpdateFeeSetting(ctx context.Context, request *v1.UpdateFeeSettingRequest) (*v1.UpdateFeeSettingResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	input, err := feeSettingInputFromAPI(request.GetFeeCode(), request.GetNameZh(), request.NameEn, request.AliasName, request.ServiceTypeId, request.GetDefaultCurrency(), request.GetBillingUnitId(), request.AbnormalCaseId, request.GetTaxRate(), request.GetTaxableServiceId(), int(request.GetSortOrder()), request.GetEnabled())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.UpdateFeeSetting(ctx, principal.Organization.ID, principal.UserID, id, input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateFeeSettingResponse{Success: true, Code: 0, Message: "OK", Data: feeSettingToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) ListBillingUnits(ctx context.Context, _ *v1.ListBillingUnitsRequest) (*v1.ListBillingUnitsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.usecase.ListBillingUnits(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.BillingUnit, 0, len(items))
	for _, item := range items {
		data = append(data, billingUnitToAPI(item))
	}
	return &v1.ListBillingUnitsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) CreateBillingUnit(ctx context.Context, request *v1.CreateBillingUnitRequest) (*v1.CreateBillingUnitResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	created, err := s.usecase.CreateBillingUnit(ctx, principal.Organization.ID, principal.UserID, &biz.BillingUnit{Code: request.GetCode(), Name: request.GetName(), SortOrder: int(request.GetSortOrder())})
	if err != nil {
		return nil, err
	}
	return &v1.CreateBillingUnitResponse{Success: true, Code: 0, Message: "OK", Data: billingUnitToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) UpdateBillingUnit(ctx context.Context, request *v1.UpdateBillingUnitRequest) (*v1.UpdateBillingUnitResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	updated, err := s.usecase.UpdateBillingUnit(ctx, principal.Organization.ID, principal.UserID, id, &biz.BillingUnit{Code: request.GetCode(), Name: request.GetName(), SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateBillingUnitResponse{Success: true, Code: 0, Message: "OK", Data: billingUnitToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) ListTaxableServices(ctx context.Context, _ *v1.ListTaxableServicesRequest) (*v1.ListTaxableServicesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.usecase.ListTaxableServices(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.TaxableService, 0, len(items))
	for _, item := range items {
		data = append(data, taxableServiceToAPI(item))
	}
	return &v1.ListTaxableServicesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) CreateTaxableService(ctx context.Context, request *v1.CreateTaxableServiceRequest) (*v1.CreateTaxableServiceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	taxRate, err := parsePlainDecimal(request.GetDefaultTaxRate())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	created, err := s.usecase.CreateTaxableService(ctx, principal.Organization.ID, principal.UserID, &biz.TaxableService{Name: request.GetName(), ShortName: request.ShortName, GoodsCode: request.GoodsCode, DefaultTaxRate: taxRate})
	if err != nil {
		return nil, err
	}
	return &v1.CreateTaxableServiceResponse{Success: true, Code: 0, Message: "OK", Data: taxableServiceToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *FeeCatalogService) UpdateTaxableService(ctx context.Context, request *v1.UpdateTaxableServiceRequest) (*v1.UpdateTaxableServiceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	taxRate, err := parsePlainDecimal(request.GetDefaultTaxRate())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	updated, err := s.usecase.UpdateTaxableService(ctx, principal.Organization.ID, principal.UserID, id, &biz.TaxableService{Name: request.GetName(), ShortName: request.ShortName, GoodsCode: request.GoodsCode, DefaultTaxRate: taxRate, Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return &v1.UpdateTaxableServiceResponse{Success: true, Code: 0, Message: "OK", Data: taxableServiceToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func feeSettingInputFromAPI(feeCode, nameZH string, nameEN, aliasName, serviceTypeText *string, defaultCurrency, billingUnitText string, abnormalCaseText *string, taxRateText, taxableServiceText string, sortOrder int, enabled bool) (*biz.FeeSetting, error) {
	billingUnitID, err := uuid.Parse(billingUnitText)
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	taxableServiceID, err := uuid.Parse(taxableServiceText)
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	serviceTypeID, err := optionalCatalogUUID(serviceTypeText)
	if err != nil {
		return nil, err
	}
	abnormalCaseID, err := optionalCatalogUUID(abnormalCaseText)
	if err != nil {
		return nil, err
	}
	taxRate, err := parsePlainDecimal(taxRateText)
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	return &biz.FeeSetting{FeeCode: feeCode, NameZH: nameZH, NameEN: nameEN, AliasName: aliasName, ServiceTypeID: serviceTypeID, DefaultCurrency: defaultCurrency, BillingUnitID: billingUnitID, AbnormalCaseID: abnormalCaseID, TaxRate: taxRate, TaxableServiceID: taxableServiceID, SortOrder: sortOrder, Enabled: enabled}, nil
}

func optionalCatalogUUID(value *string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*value)
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	return &parsed, nil
}

func feeSettingToAPI(value *biz.FeeSetting) *v1.FeeSetting {
	result := &v1.FeeSetting{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), FeeCode: value.FeeCode, NameZh: value.NameZH, NameEn: value.NameEN, AliasName: value.AliasName, ServiceTypeName: value.ServiceTypeName, DefaultCurrency: value.DefaultCurrency, BillingUnitId: value.BillingUnitID.String(), BillingUnitName: value.BillingUnitName, AbnormalCaseName: value.AbnormalCaseName, TaxRate: value.TaxRate.StringFixed(2), TaxableServiceId: value.TaxableServiceID.String(), TaxableServiceName: value.TaxableServiceName, Enabled: value.Enabled, SortOrder: int32(value.SortOrder), CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
	if value.ServiceTypeID != nil {
		text := value.ServiceTypeID.String()
		result.ServiceTypeId = &text
	}
	if value.AbnormalCaseID != nil {
		text := value.AbnormalCaseID.String()
		result.AbnormalCaseId = &text
	}
	return result
}

func billingUnitToAPI(value *biz.BillingUnit) *v1.BillingUnit {
	return &v1.BillingUnit{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Code: value.Code, Name: value.Name, SortOrder: int32(value.SortOrder), Enabled: value.Enabled, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

func taxableServiceToAPI(value *biz.TaxableService) *v1.TaxableService {
	return &v1.TaxableService{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Name: value.Name, ShortName: value.ShortName, GoodsCode: value.GoodsCode, DefaultTaxRate: value.DefaultTaxRate.StringFixed(2), Enabled: value.Enabled, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

var _ v1.FeeCatalogServiceServer = (*FeeCatalogService)(nil)
