package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

type MasterDataService struct {
	v1.UnimplementedMasterDataServiceServer
	usecase *biz.MasterDataUsecase
}

func NewMasterDataService(usecase *biz.MasterDataUsecase) *MasterDataService {
	return &MasterDataService{usecase: usecase}
}

func (s *MasterDataService) ListItems(ctx context.Context, request *v1.ListMasterDataItemsRequest) (*v1.MasterDataItemListReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := adminPageValues(request.GetPage(), request.GetPageSize())
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
	return &v1.MasterDataItemListReply{Success: true, Code: 0, Message: "OK", Data: masterDataItemsToAPI(list.Items), Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *MasterDataService) CreateItem(ctx context.Context, request *v1.CreateMasterDataItemRequest) (*v1.MasterDataItemReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, &biz.MasterDataItem{Kind: masterDataKindFromAPI(request.GetKind()), Code: request.GetCode(), Name: request.GetName(), NameEN: optionalString(request.GetNameEn(), request.NameEn != nil), ParentCode: optionalString(request.GetParentCode(), request.ParentCode != nil), TransportMode: optionalString(request.GetTransportMode(), request.TransportMode != nil), TEUFactor: optionalString(request.GetTeuFactor(), request.TeuFactor != nil), Source: request.GetSource(), SortOrder: int(request.GetSortOrder()), Enabled: true})
	if err != nil {
		return nil, err
	}
	return &v1.MasterDataItemReply{Success: true, Code: 0, Message: "OK", Data: masterDataItemToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *MasterDataService) UpdateItem(ctx context.Context, request *v1.UpdateMasterDataItemRequest) (*v1.MasterDataItemReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrMasterDataInvalidArgument
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, id, &biz.MasterDataItem{Kind: masterDataKindFromAPI(request.GetKind()), Name: request.GetName(), NameEN: optionalString(request.GetNameEn(), request.NameEn != nil), ParentCode: optionalString(request.GetParentCode(), request.ParentCode != nil), TransportMode: optionalString(request.GetTransportMode(), request.TransportMode != nil), TEUFactor: optionalString(request.GetTeuFactor(), request.TeuFactor != nil), Source: request.GetSource(), SortOrder: int(request.GetSortOrder()), Enabled: request.GetEnabled()})
	if err != nil {
		return nil, err
	}
	return &v1.MasterDataItemReply{Success: true, Code: 0, Message: "OK", Data: masterDataItemToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *MasterDataService) ListOptions(ctx context.Context, _ *v1.ListMasterDataOptionsRequest) (*v1.MasterDataOptionsReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.usecase.ListOptions(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return &v1.MasterDataOptionsReply{Success: true, Code: 0, Message: "OK", Data: masterDataItemsToAPI(items), TraceId: requestmeta.TraceID(ctx)}, nil
}

func masterDataKindFromAPI(value v1.MasterDataKind) biz.MasterDataKind {
	switch value {
	case v1.MasterDataKind_MASTER_DATA_KIND_CURRENCY:
		return biz.MasterDataKindCurrency
	case v1.MasterDataKind_MASTER_DATA_KIND_COUNTRY:
		return biz.MasterDataKindCountry
	case v1.MasterDataKind_MASTER_DATA_KIND_REGION:
		return biz.MasterDataKindRegion
	case v1.MasterDataKind_MASTER_DATA_KIND_PORT:
		return biz.MasterDataKindPort
	case v1.MasterDataKind_MASTER_DATA_KIND_AIRPORT:
		return biz.MasterDataKindAirport
	case v1.MasterDataKind_MASTER_DATA_KIND_CARRIER:
		return biz.MasterDataKindCarrier
	case v1.MasterDataKind_MASTER_DATA_KIND_CONTAINER_SPEC:
		return biz.MasterDataKindContainerSpec
	case v1.MasterDataKind_MASTER_DATA_KIND_SERVICE_TYPE:
		return biz.MasterDataKindServiceType
	case v1.MasterDataKind_MASTER_DATA_KIND_CARGO_CATEGORY:
		return biz.MasterDataKindCargoCategory
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
	case biz.MasterDataKindPort:
		return v1.MasterDataKind_MASTER_DATA_KIND_PORT
	case biz.MasterDataKindAirport:
		return v1.MasterDataKind_MASTER_DATA_KIND_AIRPORT
	case biz.MasterDataKindCarrier:
		return v1.MasterDataKind_MASTER_DATA_KIND_CARRIER
	case biz.MasterDataKindContainerSpec:
		return v1.MasterDataKind_MASTER_DATA_KIND_CONTAINER_SPEC
	case biz.MasterDataKindServiceType:
		return v1.MasterDataKind_MASTER_DATA_KIND_SERVICE_TYPE
	case biz.MasterDataKindCargoCategory:
		return v1.MasterDataKind_MASTER_DATA_KIND_CARGO_CATEGORY
	default:
		return v1.MasterDataKind_MASTER_DATA_KIND_UNSPECIFIED
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
	return &v1.MasterDataItem{Id: item.ID.String(), OrganizationId: item.OrganizationID.String(), Kind: masterDataKindToAPI(item.Kind), Code: item.Code, Name: item.Name, NameEn: item.NameEN, ParentCode: item.ParentCode, TransportMode: item.TransportMode, TeuFactor: item.TEUFactor, Source: item.Source, SortOrder: int32(item.SortOrder), Enabled: item.Enabled, CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
}

var _ v1.MasterDataServiceServer = (*MasterDataService)(nil)
