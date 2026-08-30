package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type FeeCatalogService struct {
	v1.UnimplementedFeeCatalogServiceServer
	usecase *biz.FeeCatalogUsecase
}

func NewFeeCatalogService(usecase *biz.FeeCatalogUsecase) *FeeCatalogService {
	return &FeeCatalogService{usecase: usecase}
}

func (s *FeeCatalogService) ListFeeSettings(ctx context.Context, _ *v1.ListFeeSettingsRequest) (*v1.ListFeeSettingsResponse, error) {
	data, total, page, pageSize, err := s.listFeeSettings(ctx, biz.FeeCatalogListOptions{Page: 1, PageSize: biz.MaxListPageSize})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListFeeSettingsResponse{Data: data, Total: total, Page: page, PageSize: pageSize}), nil
}

func (s *FeeCatalogService) SearchFeeSettings(ctx context.Context, request *v1.SearchFeeSettingsRequest) (*v1.SearchFeeSettingsResponse, error) {
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFeeCatalogInvalidArgument)
	if err != nil {
		return nil, err
	}
	data, total, resultPage, resultPageSize, err := s.listFeeSettings(ctx, biz.FeeCatalogListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.SearchFeeSettingsResponse{Data: data, Total: total, Page: resultPage, PageSize: resultPageSize}), nil
}

func (s *FeeCatalogService) listFeeSettings(ctx context.Context, options biz.FeeCatalogListOptions) ([]*v1.FeeSetting, int32, int32, int32, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, 0, 0, 0, principalErr
	}
	result, err := s.usecase.ListFeeSettings(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	data := make([]*v1.FeeSetting, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, feeSettingToAPI(item))
	}
	return data, int32(result.Total), int32(result.Page), int32(result.PageSize), nil
}

func (s *FeeCatalogService) CreateFeeSetting(ctx context.Context, request *v1.CreateFeeSettingRequest) (*v1.CreateFeeSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := feeSettingInputFromAPI(request.GetFeeCode(), request.GetNameZh(), request.NameEn, request.AliasName, request.ServiceTypeId, request.GetDefaultCurrency(), request.GetBillingUnitId(), request.AbnormalCaseId, request.GetTaxRate(), request.GetTaxableServiceId(), int(request.GetSortOrder()), true)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.CreateFeeSetting(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateFeeSettingResponse{Data: feeSettingToAPI(created)}), nil
}

func (s *FeeCatalogService) UpdateFeeSetting(ctx context.Context, request *v1.UpdateFeeSettingRequest) (*v1.UpdateFeeSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	return ok(ctx, &v1.UpdateFeeSettingResponse{Data: feeSettingToAPI(updated)}), nil
}

func (s *FeeCatalogService) ListBillingUnits(ctx context.Context, _ *v1.ListBillingUnitsRequest) (*v1.ListBillingUnitsResponse, error) {
	data, total, page, pageSize, err := s.listBillingUnits(ctx, biz.FeeCatalogListOptions{Page: 1, PageSize: biz.MaxListPageSize})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListBillingUnitsResponse{Data: data, Total: total, Page: page, PageSize: pageSize}), nil
}

func (s *FeeCatalogService) SearchBillingUnits(ctx context.Context, request *v1.SearchBillingUnitsRequest) (*v1.SearchBillingUnitsResponse, error) {
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFeeCatalogInvalidArgument)
	if err != nil {
		return nil, err
	}
	data, total, resultPage, resultPageSize, err := s.listBillingUnits(ctx, biz.FeeCatalogListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.SearchBillingUnitsResponse{Data: data, Total: total, Page: resultPage, PageSize: resultPageSize}), nil
}

func (s *FeeCatalogService) listBillingUnits(ctx context.Context, options biz.FeeCatalogListOptions) ([]*v1.BillingUnit, int32, int32, int32, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, 0, 0, 0, principalErr
	}
	result, err := s.usecase.ListBillingUnits(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	data := make([]*v1.BillingUnit, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, billingUnitToAPI(item))
	}
	return data, int32(result.Total), int32(result.Page), int32(result.PageSize), nil
}

func (s *FeeCatalogService) CreateBillingUnit(ctx context.Context, request *v1.CreateBillingUnitRequest) (*v1.CreateBillingUnitResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	created, err := s.usecase.CreateBillingUnit(ctx, principal.Organization.ID, principal.UserID, &biz.BillingUnit{Code: request.GetCode(), Name: request.GetName(), IsContainerUnit: request.GetIsContainerUnit(), SortOrder: int(request.GetSortOrder())})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateBillingUnitResponse{Data: billingUnitToAPI(created)}), nil
}

func (s *FeeCatalogService) UpdateBillingUnit(ctx context.Context, request *v1.UpdateBillingUnitRequest) (*v1.UpdateBillingUnitResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	updated, err := s.usecase.UpdateBillingUnit(ctx, principal.Organization.ID, principal.UserID, id, &biz.BillingUnit{Code: request.GetCode(), Name: request.GetName(), IsContainerUnit: request.GetIsContainerUnit(), SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateBillingUnitResponse{Data: billingUnitToAPI(updated)}), nil
}

func (s *FeeCatalogService) ListTaxableServices(ctx context.Context, _ *v1.ListTaxableServicesRequest) (*v1.ListTaxableServicesResponse, error) {
	data, total, page, pageSize, err := s.listTaxableServices(ctx, biz.FeeCatalogListOptions{Page: 1, PageSize: biz.MaxListPageSize})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListTaxableServicesResponse{Data: data, Total: total, Page: page, PageSize: pageSize}), nil
}

func (s *FeeCatalogService) SearchTaxableServices(ctx context.Context, request *v1.SearchTaxableServicesRequest) (*v1.SearchTaxableServicesResponse, error) {
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrFeeCatalogInvalidArgument)
	if err != nil {
		return nil, err
	}
	data, total, resultPage, resultPageSize, err := s.listTaxableServices(ctx, biz.FeeCatalogListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.SearchTaxableServicesResponse{Data: data, Total: total, Page: resultPage, PageSize: resultPageSize}), nil
}

func (s *FeeCatalogService) listTaxableServices(ctx context.Context, options biz.FeeCatalogListOptions) ([]*v1.TaxableService, int32, int32, int32, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, 0, 0, 0, principalErr
	}
	result, err := s.usecase.ListTaxableServices(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	data := make([]*v1.TaxableService, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, taxableServiceToAPI(item))
	}
	return data, int32(result.Total), int32(result.Page), int32(result.PageSize), nil
}

func (s *FeeCatalogService) CreateTaxableService(ctx context.Context, request *v1.CreateTaxableServiceRequest) (*v1.CreateTaxableServiceResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	taxRate, err := parsePlainDecimal(request.GetDefaultTaxRate())
	if err != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	created, err := s.usecase.CreateTaxableService(ctx, principal.Organization.ID, principal.UserID, &biz.TaxableService{Name: request.GetName(), ShortName: request.ShortName, GoodsCode: request.GoodsCode, DefaultTaxRate: taxRate})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateTaxableServiceResponse{Data: taxableServiceToAPI(created)}), nil
}

func (s *FeeCatalogService) UpdateTaxableService(ctx context.Context, request *v1.UpdateTaxableServiceRequest) (*v1.UpdateTaxableServiceResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	return ok(ctx, &v1.UpdateTaxableServiceResponse{Data: taxableServiceToAPI(updated)}), nil
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
	return &v1.BillingUnit{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Code: value.Code, Name: value.Name, IsContainerUnit: value.IsContainerUnit, SortOrder: int32(value.SortOrder), Enabled: value.Enabled, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

func taxableServiceToAPI(value *biz.TaxableService) *v1.TaxableService {
	return &v1.TaxableService{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), Name: value.Name, ShortName: value.ShortName, GoodsCode: value.GoodsCode, DefaultTaxRate: value.DefaultTaxRate.StringFixed(2), Enabled: value.Enabled, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

var _ v1.FeeCatalogServiceServer = (*FeeCatalogService)(nil)
