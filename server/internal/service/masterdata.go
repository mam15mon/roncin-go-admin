package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

type MasterDataService struct {
	v1.UnimplementedMasterDataServiceServer
	usecase              *biz.MasterDataUsecase
	industryUsecase      *biz.IndustryReferenceUsecase
	referenceDataUsecase *biz.ReferenceDataUsecase
	orderConfigUsecase   *biz.OrderConfigUsecase
}

func NewMasterDataService(usecase *biz.MasterDataUsecase, industryUsecase *biz.IndustryReferenceUsecase, referenceDataUsecase *biz.ReferenceDataUsecase, orderConfigUsecase *biz.OrderConfigUsecase) *MasterDataService {
	return &MasterDataService{
		usecase:              usecase,
		industryUsecase:      industryUsecase,
		referenceDataUsecase: referenceDataUsecase,
		orderConfigUsecase:   orderConfigUsecase,
	}
}

func (s *MasterDataService) ListCurrencies(ctx context.Context, _ *v1.ListCurrenciesRequest) (*v1.ListCurrenciesResponse, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	items, err := s.referenceDataUsecase.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.Currency, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.Currency{
			Id: item.ID.String(), Code: item.Code, Name: item.Name, Symbol: item.Symbol,
			MinorUnit: int32(item.MinorUnit), Enabled: item.Enabled,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return okList(ctx, &v1.ListCurrenciesResponse{Data: data, Total: int32(len(data)), Page: 1, PageSize: int32(biz.MaxListPageSize)}), nil
}

func (s *MasterDataService) SearchCurrencies(ctx context.Context, request *v1.SearchCurrenciesRequest) (*v1.SearchCurrenciesResponse, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrReferenceDataInvalidArgument)
	if err != nil {
		return nil, err
	}
	result, err := s.referenceDataUsecase.SearchCurrencies(ctx, biz.SelectorListOptions{Keyword: request.GetKeyword(), Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.Currency, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.Currency{Id: item.ID.String(), Code: item.Code, Name: item.Name, Symbol: item.Symbol, MinorUnit: int32(item.MinorUnit), Enabled: item.Enabled, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339Nano)})
	}
	return okList(ctx, &v1.SearchCurrenciesResponse{Data: data, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *MasterDataService) ListAdministrativeRegions(ctx context.Context, request *v1.ListAdministrativeRegionsRequest) (*v1.ListAdministrativeRegionsResponse, error) {
	if _, err := requirePrincipal(ctx); err != nil {
		return nil, err
	}
	page, pageSize := biz.ListPagination(int(request.GetPage()), int(request.GetPageSize()), biz.MaxListPageSize)
	result, err := s.referenceDataUsecase.ListAdministrativeRegions(ctx, biz.AdministrativeRegionQuery{
		Level: int(request.GetLevel()), ParentCode: optionalString(request.GetParentCode(), request.ParentCode != nil), Keyword: request.GetKeyword(), Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdministrativeRegion, 0, len(result.Items))
	for _, item := range result.Items {
		data = append(data, &v1.AdministrativeRegion{
			Id: item.ID.String(), Code: item.Code, Name: item.Name, Level: int32(item.Level),
			ParentCode: item.ParentCode, RegionType: item.RegionType, Source: item.Source,
			SourceVersion: item.SourceVersion, Enabled: item.Enabled,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return okList(ctx, &v1.ListAdministrativeRegionsResponse{Data: data, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *MasterDataService) ListItems(ctx context.Context, request *v1.ListItemsRequest) (*v1.ListItemsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, biz.ErrMasterDataInvalidArgument
	}
	options := biz.MasterDataListOptions{Page: page, PageSize: pageSize, Kind: masterDataKindFromAPI(request.GetKind()), Keyword: request.GetKeyword()}
	if request.Enabled != nil {
		enabled := request.GetEnabled()
		options.Enabled = &enabled
	}
	list, err := s.usecase.List(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListItemsResponse{Data: masterDataItemsToAPI(list.Items), Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize)}), nil
}

func (s *MasterDataService) CreateItem(ctx context.Context, request *v1.CreateItemRequest) (*v1.CreateItemResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, &biz.MasterDataItem{Kind: masterDataKindFromAPI(request.GetKind()), Code: request.GetCode(), Name: request.GetName(), NameEN: optionalString(request.GetNameEn(), request.NameEn != nil), ParentCode: optionalString(request.GetParentCode(), request.ParentCode != nil), TEUFactor: optionalString(request.GetTeuFactor(), request.TeuFactor != nil), Attributes: masterDataAttributesFromAPI(request.GetAttributes()), Source: request.GetSource(), SortOrder: int(request.GetSortOrder()), Enabled: true})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateItemResponse{Data: masterDataItemToAPI(created)}), nil
}

func (s *MasterDataService) UpdateItem(ctx context.Context, request *v1.UpdateItemRequest) (*v1.UpdateItemResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrMasterDataInvalidArgument
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, id, &biz.MasterDataItem{Kind: masterDataKindFromAPI(request.GetKind()), Name: request.GetName(), NameEN: optionalString(request.GetNameEn(), request.NameEn != nil), ParentCode: optionalString(request.GetParentCode(), request.ParentCode != nil), TEUFactor: optionalString(request.GetTeuFactor(), request.TeuFactor != nil), Attributes: masterDataAttributesFromAPI(request.GetAttributes()), Source: request.GetSource(), SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateItemResponse{Data: masterDataItemToAPI(updated)}), nil
}

func (s *MasterDataService) ListOptions(ctx context.Context, _ *v1.ListOptionsRequest) (*v1.ListOptionsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.usecase.ListOptions(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListOptionsResponse{Data: masterDataItemsToAPI(items)}), nil
}

func (s *MasterDataService) ListPorts(ctx context.Context, request *v1.ListPortsRequest) (*v1.ListPortsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	options := industryReferenceListOptions(request.GetPage(), request.GetPageSize(), request.GetKeyword(), request.Enabled)
	result, err := s.industryUsecase.ListPorts(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListPortsResponse{Data: portsToAPI(result.Items), Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *MasterDataService) CreatePort(ctx context.Context, request *v1.CreatePortRequest) (*v1.CreatePortResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.industryUsecase.CreatePort(ctx, principal.Organization.ID, principal.UserID, &biz.Port{UNLocode: request.GetUnLocode(), NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CountryCode: request.GetCountryCode(), TransportModes: request.GetTransportModes(), Source: "manual", SortOrder: int(request.GetSortOrder())})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreatePortResponse{Data: portToAPI(created)}), nil
}

func (s *MasterDataService) UpdatePort(ctx context.Context, request *v1.UpdatePortRequest) (*v1.UpdatePortResponse, error) {
	principal, id, err := principalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	updated, err := s.industryUsecase.UpdatePort(ctx, principal.Organization.ID, principal.UserID, id, &biz.Port{NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CountryCode: request.GetCountryCode(), TransportModes: request.GetTransportModes(), Source: "manual", SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdatePortResponse{Data: portToAPI(updated)}), nil
}

func (s *MasterDataService) ListAirports(ctx context.Context, request *v1.ListAirportsRequest) (*v1.ListAirportsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.industryUsecase.ListAirports(ctx, principal.Organization.ID, industryReferenceListOptions(request.GetPage(), request.GetPageSize(), request.GetKeyword(), request.Enabled))
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListAirportsResponse{Data: airportsToAPI(result.Items), Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *MasterDataService) CreateAirport(ctx context.Context, request *v1.CreateAirportRequest) (*v1.CreateAirportResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.industryUsecase.CreateAirport(ctx, principal.Organization.ID, principal.UserID, &biz.Airport{IATACode: request.GetIataCode(), ICAOCode: optionalString(request.GetIcaoCode(), request.IcaoCode != nil), NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CityNameZH: request.GetCityNameZh(), CityNameEN: optionalString(request.GetCityNameEn(), request.CityNameEn != nil), CountryCode: request.GetCountryCode(), Source: "manual", SortOrder: int(request.GetSortOrder())})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateAirportResponse{Data: airportToAPI(created)}), nil
}

func (s *MasterDataService) UpdateAirport(ctx context.Context, request *v1.UpdateAirportRequest) (*v1.UpdateAirportResponse, error) {
	principal, id, err := principalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	updated, err := s.industryUsecase.UpdateAirport(ctx, principal.Organization.ID, principal.UserID, id, &biz.Airport{ICAOCode: optionalString(request.GetIcaoCode(), request.IcaoCode != nil), NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CityNameZH: request.GetCityNameZh(), CityNameEN: optionalString(request.GetCityNameEn(), request.CityNameEn != nil), CountryCode: request.GetCountryCode(), Source: "manual", SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateAirportResponse{Data: airportToAPI(updated)}), nil
}

func (s *MasterDataService) ListAirlines(ctx context.Context, request *v1.ListAirlinesRequest) (*v1.ListAirlinesResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.industryUsecase.ListAirlines(ctx, principal.Organization.ID, industryReferenceListOptions(request.GetPage(), request.GetPageSize(), request.GetKeyword(), request.Enabled))
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListAirlinesResponse{Data: airlinesToAPI(result.Items), Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *MasterDataService) CreateAirline(ctx context.Context, request *v1.CreateAirlineRequest) (*v1.CreateAirlineResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.industryUsecase.CreateAirline(ctx, principal.Organization.ID, principal.UserID, &biz.Airline{IATACode: request.GetIataCode(), ICAOCode: optionalString(request.GetIcaoCode(), request.IcaoCode != nil), AWBPrefix: request.GetAwbPrefix(), NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CountryCode: request.GetCountryCode(), CargoOnly: request.GetCargoOnly(), Source: request.GetSource(), SortOrder: int(request.GetSortOrder())})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateAirlineResponse{Data: airlineToAPI(created)}), nil
}

func (s *MasterDataService) UpdateAirline(ctx context.Context, request *v1.UpdateAirlineRequest) (*v1.UpdateAirlineResponse, error) {
	principal, id, err := principalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	updated, err := s.industryUsecase.UpdateAirline(ctx, principal.Organization.ID, principal.UserID, id, &biz.Airline{ICAOCode: optionalString(request.GetIcaoCode(), request.IcaoCode != nil), AWBPrefix: request.GetAwbPrefix(), NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CountryCode: request.GetCountryCode(), CargoOnly: request.GetCargoOnly(), Source: request.GetSource(), SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateAirlineResponse{Data: airlineToAPI(updated)}), nil
}

func (s *MasterDataService) ListShippingLines(ctx context.Context, request *v1.ListShippingLinesRequest) (*v1.ListShippingLinesResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.industryUsecase.ListShippingLines(ctx, principal.Organization.ID, industryReferenceListOptions(request.GetPage(), request.GetPageSize(), request.GetKeyword(), request.Enabled))
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListShippingLinesResponse{Data: shippingLinesToAPI(result.Items), Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}

func (s *MasterDataService) CreateShippingLine(ctx context.Context, request *v1.CreateShippingLineRequest) (*v1.CreateShippingLineResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.industryUsecase.CreateShippingLine(ctx, principal.Organization.ID, principal.UserID, &biz.ShippingLine{SCACCode: request.GetScacCode(), NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CountryCode: request.GetCountryCode(), TrackingURL: optionalString(request.GetTrackingUrl(), request.TrackingUrl != nil), Alliance: optionalString(request.GetAlliance(), request.Alliance != nil), ContainerPrefixes: request.GetContainerPrefixes(), Source: request.GetSource(), SortOrder: int(request.GetSortOrder())})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateShippingLineResponse{Data: shippingLineToAPI(created)}), nil
}

func (s *MasterDataService) UpdateShippingLine(ctx context.Context, request *v1.UpdateShippingLineRequest) (*v1.UpdateShippingLineResponse, error) {
	principal, id, err := principalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	updated, err := s.industryUsecase.UpdateShippingLine(ctx, principal.Organization.ID, principal.UserID, id, &biz.ShippingLine{NameZH: request.GetNameZh(), NameEN: request.GetNameEn(), CountryCode: request.GetCountryCode(), TrackingURL: optionalString(request.GetTrackingUrl(), request.TrackingUrl != nil), Alliance: optionalString(request.GetAlliance(), request.Alliance != nil), ContainerPrefixes: request.GetContainerPrefixes(), Source: request.GetSource(), SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateShippingLineResponse{Data: shippingLineToAPI(updated)}), nil
}

func (s *MasterDataService) ImportItems(ctx context.Context, request *v1.ImportItemsRequest) (*v1.ImportItemsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.MasterDataItem, 0, len(request.GetItems()))
	for _, item := range request.GetItems() {
		if item == nil {
			return nil, biz.ErrMasterDataInvalidArgument
		}
		items = append(items, &biz.MasterDataItem{
			Code:       item.GetCode(),
			Name:       item.GetName(),
			NameEN:     optionalString(item.GetNameEn(), item.NameEn != nil),
			ParentCode: optionalString(item.GetParentCode(), item.ParentCode != nil),
			TEUFactor:  optionalString(item.GetTeuFactor(), item.TeuFactor != nil),
			Attributes: masterDataAttributesFromAPI(item.GetAttributes()),
			SortOrder:  int(item.GetSortOrder()),
			Enabled:    item.Enabled == nil || item.GetEnabled(),
		})
	}
	result, err := s.usecase.Import(ctx, principal.Organization.ID, principal.UserID, biz.MasterDataImportInput{
		Kind:   masterDataKindFromAPI(request.GetKind()),
		Source: request.GetSource(),
		Mode:   masterDataImportModeFromAPI(request.GetMode()),
		Items:  items,
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ImportItemsResponse{
		Data:         masterDataItemsToAPI(result.Items),
		CreatedCount: int32(result.Created),
		UpdatedCount: int32(result.Updated),
	}), nil
}

func (s *MasterDataService) ListNumberRules(ctx context.Context, _ *v1.ListNumberRulesRequest) (*v1.ListNumberRulesResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.orderConfigUsecase.ListNumberRules(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return okList(ctx, &v1.ListNumberRulesResponse{
		Data: numberRulesToAPI(rules),
	}), nil
}

func (s *MasterDataService) CreateNumberRule(ctx context.Context, request *v1.CreateNumberRuleRequest) (*v1.CreateNumberRuleResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.orderConfigUsecase.CreateNumberRule(ctx, principal.Organization.ID, principal.UserID, &biz.NumberRule{
		DocumentType:   documentTypeFromAPI(request.GetDocumentType()),
		Prefix:         request.GetPrefix(),
		DateFormat:     dateFormatFromAPI(request.GetDateFormat()),
		SequenceLength: int(request.GetSequenceLength()),
		ResetPolicy:    resetPolicyFromAPI(request.GetResetPolicy()),
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateNumberRuleResponse{
		Data: numberRuleToAPI(created),
	}), nil
}

func (s *MasterDataService) UpdateNumberRule(ctx context.Context, request *v1.UpdateNumberRuleRequest) (*v1.UpdateNumberRuleResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrMasterDataInvalidArgument
	}
	updated, err := s.orderConfigUsecase.UpdateNumberRule(ctx, principal.Organization.ID, principal.UserID, id, &biz.NumberRule{
		Prefix:         request.GetPrefix(),
		DateFormat:     dateFormatFromAPI(request.GetDateFormat()),
		SequenceLength: int(request.GetSequenceLength()),
		ResetPolicy:    resetPolicyFromAPI(request.GetResetPolicy()),
		Enabled:        request.GetEnabled(),
	})
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateNumberRuleResponse{
		Data: numberRuleToAPI(updated),
	}), nil
}

func masterDataKindFromAPI(value v1.MasterDataKind) biz.MasterDataKind {
	switch value {
	case v1.MasterDataKind_MASTER_DATA_KIND_CURRENCY:
		return biz.MasterDataKindCurrency
	case v1.MasterDataKind_MASTER_DATA_KIND_COUNTRY:
		return biz.MasterDataKindCountry
	case v1.MasterDataKind_MASTER_DATA_KIND_REGION:
		return biz.MasterDataKindRegion
	case v1.MasterDataKind_MASTER_DATA_KIND_CONTAINER_SPEC:
		return biz.MasterDataKindContainerSpec
	case v1.MasterDataKind_MASTER_DATA_KIND_SERVICE_TYPE:
		return biz.MasterDataKindServiceType
	case v1.MasterDataKind_MASTER_DATA_KIND_CARGO_CATEGORY:
		return biz.MasterDataKindCargoCategory
	case v1.MasterDataKind_MASTER_DATA_KIND_ABNORMAL_CASE:
		return biz.MasterDataKindAbnormalCase
	default:
		return ""
	}
}

func masterDataKindToAPI(value biz.MasterDataKind) v1.MasterDataKind {
	switch value {
	case biz.MasterDataKindCurrency:
		return v1.MasterDataKind_MASTER_DATA_KIND_CURRENCY
	case biz.MasterDataKindCountry:
		return v1.MasterDataKind_MASTER_DATA_KIND_COUNTRY
	case biz.MasterDataKindRegion:
		return v1.MasterDataKind_MASTER_DATA_KIND_REGION
	case biz.MasterDataKindContainerSpec:
		return v1.MasterDataKind_MASTER_DATA_KIND_CONTAINER_SPEC
	case biz.MasterDataKindServiceType:
		return v1.MasterDataKind_MASTER_DATA_KIND_SERVICE_TYPE
	case biz.MasterDataKindCargoCategory:
		return v1.MasterDataKind_MASTER_DATA_KIND_CARGO_CATEGORY
	case biz.MasterDataKindAbnormalCase:
		return v1.MasterDataKind_MASTER_DATA_KIND_ABNORMAL_CASE
	default:
		return v1.MasterDataKind_MASTER_DATA_KIND_UNSPECIFIED
	}
}

func masterDataImportModeFromAPI(value v1.MasterDataImportMode) biz.MasterDataImportMode {
	switch value {
	case v1.MasterDataImportMode_MASTER_DATA_IMPORT_MODE_CREATE_ONLY:
		return biz.MasterDataImportModeCreateOnly
	case v1.MasterDataImportMode_MASTER_DATA_IMPORT_MODE_UPSERT:
		return biz.MasterDataImportModeUpsert
	default:
		return ""
	}
}

func masterDataItemsToAPI(items []*biz.MasterDataItem) []*v1.MasterDataItem {
	result := make([]*v1.MasterDataItem, 0, len(items))
	for _, item := range items {
		result = append(result, masterDataItemToAPI(item))
	}
	return result
}

func masterDataItemToAPI(item *biz.MasterDataItem) *v1.MasterDataItem {
	return &v1.MasterDataItem{
		Id:             item.ID.String(),
		OrganizationId: item.OrganizationID.String(),
		Kind:           masterDataKindToAPI(item.Kind),
		Code:           item.Code,
		Name:           item.Name,
		NameEn:         item.NameEN,
		ParentCode:     item.ParentCode,
		TeuFactor:      item.TEUFactor,
		Source:         item.Source,
		SortOrder:      int32(item.SortOrder),
		Enabled:        item.Enabled,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
		Attributes:     masterDataAttributesToAPI(item.Attributes),
	}
}

func masterDataAttributesFromAPI(attributes *v1.MasterDataAttributes) biz.MasterDataAttributes {
	if attributes == nil {
		return biz.MasterDataAttributes{}
	}
	return biz.MasterDataAttributes{
		Continent:    optionalString(attributes.GetContinent(), attributes.Continent != nil),
		CurrencyCode: optionalString(attributes.GetCurrencyCode(), attributes.CurrencyCode != nil),
		RegionLevel:  optionalInt(attributes.GetRegionLevel(), attributes.RegionLevel != nil),
	}
}

func masterDataAttributesToAPI(attributes biz.MasterDataAttributes) *v1.MasterDataAttributes {
	return &v1.MasterDataAttributes{
		Continent: attributes.Continent, CurrencyCode: attributes.CurrencyCode,
		RegionLevel: intPointerToInt32(attributes.RegionLevel),
	}
}

func industryReferenceListOptions(rawPage, rawPageSize int32, keyword string, enabled *bool) biz.IndustryReferenceListOptions {
	page, pageSize := int(rawPage), int(rawPageSize)
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 100
	}
	enabledValue := false
	if enabled != nil {
		enabledValue = *enabled
	}
	return biz.IndustryReferenceListOptions{Page: page, PageSize: pageSize, Keyword: keyword, Enabled: optionalBool(enabledValue, enabled != nil)}
}

func principalAndID(ctx context.Context, rawID string) (*biz.Principal, uuid.UUID, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, uuid.Nil, err
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, uuid.Nil, biz.ErrMasterDataInvalidArgument
	}
	return principal, id, nil
}

func portsToAPI(items []*biz.Port) []*v1.Port {
	result := make([]*v1.Port, 0, len(items))
	for _, item := range items {
		result = append(result, portToAPI(item))
	}
	return result
}

func portToAPI(item *biz.Port) *v1.Port {
	return &v1.Port{Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), UnLocode: item.UNLocode, NameZh: optionalString(item.NameZH, item.NameZH != ""), NameEn: item.NameEN, CountryCode: item.CountryCode, TransportModes: append([]string(nil), item.TransportModes...), Source: item.Source, SortOrder: int32(item.SortOrder), Enabled: item.Enabled, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339), SourceVersion: item.SourceVersion, SourceHash: item.SourceHash}
}

func airportsToAPI(items []*biz.Airport) []*v1.Airport {
	result := make([]*v1.Airport, 0, len(items))
	for _, item := range items {
		result = append(result, airportToAPI(item))
	}
	return result
}

func airportToAPI(item *biz.Airport) *v1.Airport {
	return &v1.Airport{Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), IataCode: item.IATACode, IcaoCode: item.ICAOCode, NameZh: optionalString(item.NameZH, item.NameZH != ""), NameEn: item.NameEN, CityNameZh: optionalString(item.CityNameZH, item.CityNameZH != ""), CityNameEn: item.CityNameEN, CountryCode: item.CountryCode, Source: item.Source, SortOrder: int32(item.SortOrder), Enabled: item.Enabled, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339), SourceVersion: item.SourceVersion, SourceHash: item.SourceHash}
}

func airlinesToAPI(items []*biz.Airline) []*v1.Airline {
	result := make([]*v1.Airline, 0, len(items))
	for _, item := range items {
		result = append(result, airlineToAPI(item))
	}
	return result
}

func airlineToAPI(item *biz.Airline) *v1.Airline {
	return &v1.Airline{Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), IataCode: item.IATACode, IcaoCode: item.ICAOCode, AwbPrefix: item.AWBPrefix, NameZh: item.NameZH, NameEn: item.NameEN, CountryCode: item.CountryCode, CargoOnly: item.CargoOnly, Source: item.Source, SortOrder: int32(item.SortOrder), Enabled: item.Enabled, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}

func shippingLinesToAPI(items []*biz.ShippingLine) []*v1.ShippingLine {
	result := make([]*v1.ShippingLine, 0, len(items))
	for _, item := range items {
		result = append(result, shippingLineToAPI(item))
	}
	return result
}

func shippingLineToAPI(item *biz.ShippingLine) *v1.ShippingLine {
	return &v1.ShippingLine{Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), ScacCode: item.SCACCode, NameZh: item.NameZH, NameEn: item.NameEN, CountryCode: item.CountryCode, TrackingUrl: item.TrackingURL, Alliance: item.Alliance, ContainerPrefixes: append([]string(nil), item.ContainerPrefixes...), Source: item.Source, SortOrder: int32(item.SortOrder), Enabled: item.Enabled, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}

func optionalBool(value bool, present bool) *bool {
	if !present {
		return nil
	}
	return &value
}

func optionalInt(value int32, present bool) *int {
	if !present {
		return nil
	}
	converted := int(value)
	return &converted
}

func intPointerToInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func documentTypeFromAPI(value v1.DocumentType) biz.DocumentType {
	switch value {
	case v1.DocumentType_DOCUMENT_TYPE_ORDER:
		return biz.DocumentTypeOrder
	case v1.DocumentType_DOCUMENT_TYPE_BILL:
		return biz.DocumentTypeBill
	case v1.DocumentType_DOCUMENT_TYPE_BILL_BATCH:
		return biz.DocumentTypeBillBatch
	case v1.DocumentType_DOCUMENT_TYPE_QUOTATION:
		return biz.DocumentTypeQuotation
	case v1.DocumentType_DOCUMENT_TYPE_WRITE_OFF:
		return biz.DocumentTypeWriteOff
	case v1.DocumentType_DOCUMENT_TYPE_RECEIPT_PAYMENT:
		return biz.DocumentTypeReceiptPayment
	case v1.DocumentType_DOCUMENT_TYPE_CONTRACT:
		return biz.DocumentTypeContract
	case v1.DocumentType_DOCUMENT_TYPE_INTERNAL_REFERENCE:
		return biz.DocumentTypeInternalReference
	case v1.DocumentType_DOCUMENT_TYPE_CUSTOMER_REFERENCE:
		return biz.DocumentTypeCustomerReference
	case v1.DocumentType_DOCUMENT_TYPE_HOUSE_BILL:
		return biz.DocumentTypeHouseBill
	case v1.DocumentType_DOCUMENT_TYPE_INVOICE:
		return biz.DocumentTypeInvoice
	case v1.DocumentType_DOCUMENT_TYPE_FREIGHT_RATE:
		return biz.DocumentTypeFreightRate
	case v1.DocumentType_DOCUMENT_TYPE_COMMISSION:
		return biz.DocumentTypeCommission
	default:
		return ""
	}
}

func documentTypeToAPI(value biz.DocumentType) v1.DocumentType {
	switch value {
	case biz.DocumentTypeOrder:
		return v1.DocumentType_DOCUMENT_TYPE_ORDER
	case biz.DocumentTypeBill:
		return v1.DocumentType_DOCUMENT_TYPE_BILL
	case biz.DocumentTypeBillBatch:
		return v1.DocumentType_DOCUMENT_TYPE_BILL_BATCH
	case biz.DocumentTypeQuotation:
		return v1.DocumentType_DOCUMENT_TYPE_QUOTATION
	case biz.DocumentTypeWriteOff:
		return v1.DocumentType_DOCUMENT_TYPE_WRITE_OFF
	case biz.DocumentTypeReceiptPayment:
		return v1.DocumentType_DOCUMENT_TYPE_RECEIPT_PAYMENT
	case biz.DocumentTypeContract:
		return v1.DocumentType_DOCUMENT_TYPE_CONTRACT
	case biz.DocumentTypeInternalReference:
		return v1.DocumentType_DOCUMENT_TYPE_INTERNAL_REFERENCE
	case biz.DocumentTypeCustomerReference:
		return v1.DocumentType_DOCUMENT_TYPE_CUSTOMER_REFERENCE
	case biz.DocumentTypeHouseBill:
		return v1.DocumentType_DOCUMENT_TYPE_HOUSE_BILL
	case biz.DocumentTypeInvoice:
		return v1.DocumentType_DOCUMENT_TYPE_INVOICE
	case biz.DocumentTypeFreightRate:
		return v1.DocumentType_DOCUMENT_TYPE_FREIGHT_RATE
	case biz.DocumentTypeCommission:
		return v1.DocumentType_DOCUMENT_TYPE_COMMISSION
	default:
		return v1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED
	}
}

func dateFormatFromAPI(value v1.DateFormat) biz.DateFormat {
	switch value {
	case v1.DateFormat_DATE_FORMAT_YYYYMMDD:
		return biz.DateFormatYYYYMMDD
	case v1.DateFormat_DATE_FORMAT_YYYYMM:
		return biz.DateFormatYYYYMM
	case v1.DateFormat_DATE_FORMAT_YYYY:
		return biz.DateFormatYYYY
	case v1.DateFormat_DATE_FORMAT_NONE:
		return biz.DateFormatNone
	default:
		return ""
	}
}

func dateFormatToAPI(value biz.DateFormat) v1.DateFormat {
	switch value {
	case biz.DateFormatYYYYMMDD:
		return v1.DateFormat_DATE_FORMAT_YYYYMMDD
	case biz.DateFormatYYYYMM:
		return v1.DateFormat_DATE_FORMAT_YYYYMM
	case biz.DateFormatYYYY:
		return v1.DateFormat_DATE_FORMAT_YYYY
	case biz.DateFormatNone:
		return v1.DateFormat_DATE_FORMAT_NONE
	default:
		return v1.DateFormat_DATE_FORMAT_UNSPECIFIED
	}
}

func resetPolicyFromAPI(value v1.ResetPolicy) biz.ResetPolicy {
	switch value {
	case v1.ResetPolicy_RESET_POLICY_DAILY:
		return biz.ResetPolicyDaily
	case v1.ResetPolicy_RESET_POLICY_MONTHLY:
		return biz.ResetPolicyMonthly
	case v1.ResetPolicy_RESET_POLICY_YEARLY:
		return biz.ResetPolicyYearly
	case v1.ResetPolicy_RESET_POLICY_NEVER:
		return biz.ResetPolicyNever
	default:
		return ""
	}
}

func resetPolicyToAPI(value biz.ResetPolicy) v1.ResetPolicy {
	switch value {
	case biz.ResetPolicyDaily:
		return v1.ResetPolicy_RESET_POLICY_DAILY
	case biz.ResetPolicyMonthly:
		return v1.ResetPolicy_RESET_POLICY_MONTHLY
	case biz.ResetPolicyYearly:
		return v1.ResetPolicy_RESET_POLICY_YEARLY
	case biz.ResetPolicyNever:
		return v1.ResetPolicy_RESET_POLICY_NEVER
	default:
		return v1.ResetPolicy_RESET_POLICY_UNSPECIFIED
	}
}

func businessTypeFromAPI(value v1.BusinessType) biz.BusinessType {
	switch value {
	case v1.BusinessType_BUSINESS_TYPE_SE:
		return biz.BusinessTypeSE
	case v1.BusinessType_BUSINESS_TYPE_SI:
		return biz.BusinessTypeSI
	case v1.BusinessType_BUSINESS_TYPE_AE:
		return biz.BusinessTypeAE
	case v1.BusinessType_BUSINESS_TYPE_AI:
		return biz.BusinessTypeAI
	case v1.BusinessType_BUSINESS_TYPE_LAND:
		return biz.BusinessTypeLand
	case v1.BusinessType_BUSINESS_TYPE_RAIL:
		return biz.BusinessTypeRail
	default:
		return ""
	}
}

func businessTypeToAPI(value biz.BusinessType) v1.BusinessType {
	switch value {
	case biz.BusinessTypeSE:
		return v1.BusinessType_BUSINESS_TYPE_SE
	case biz.BusinessTypeSI:
		return v1.BusinessType_BUSINESS_TYPE_SI
	case biz.BusinessTypeAE:
		return v1.BusinessType_BUSINESS_TYPE_AE
	case biz.BusinessTypeAI:
		return v1.BusinessType_BUSINESS_TYPE_AI
	case biz.BusinessTypeLand:
		return v1.BusinessType_BUSINESS_TYPE_LAND
	case biz.BusinessTypeRail:
		return v1.BusinessType_BUSINESS_TYPE_RAIL
	default:
		return v1.BusinessType_BUSINESS_TYPE_UNSPECIFIED
	}
}

func numberRulesToAPI(items []*biz.NumberRule) []*v1.NumberRule {
	result := make([]*v1.NumberRule, 0, len(items))
	for _, item := range items {
		result = append(result, numberRuleToAPI(item))
	}
	return result
}

func numberRuleToAPI(item *biz.NumberRule) *v1.NumberRule {
	return &v1.NumberRule{
		Id:             item.ID.String(),
		OrganizationId: item.OrganizationID.String(),
		DocumentType:   documentTypeToAPI(item.DocumentType),
		Prefix:         item.Prefix,
		DateFormat:     dateFormatToAPI(item.DateFormat),
		SequenceLength: int32(item.SequenceLength),
		ResetPolicy:    resetPolicyToAPI(item.ResetPolicy),
		Enabled:        item.Enabled,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

var _ v1.MasterDataServiceServer = (*MasterDataService)(nil)
