package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/enterprise_resource/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

// EnterpriseResourceService 负责企业资源请求与领域对象的转换。
type EnterpriseResourceService struct {
	v1.UnimplementedEnterpriseResourceServiceServer
	usecase *biz.EnterpriseResourceUsecase
}

func NewEnterpriseResourceService(usecase *biz.EnterpriseResourceUsecase) *EnterpriseResourceService {
	return &EnterpriseResourceService{usecase: usecase}
}

func (s *EnterpriseResourceService) SearchEnterpriseResourcePartnerOptions(ctx context.Context, request *v1.SearchEnterpriseResourcePartnerOptionsRequest) (*v1.SearchEnterpriseResourcePartnerOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrEnterpriseResourceInvalidArgument)
	if err != nil {
		return nil, err
	}
	items, total, err := s.usecase.SearchPartnerOptions(ctx, principal.Organization.ID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.EnterpriseResourcePartnerOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.EnterpriseResourcePartnerOption{Id: item.ID.String(), Code: item.Code, Name: item.Name})
	}
	return &v1.SearchEnterpriseResourcePartnerOptionsResponse{Success: true, Message: "OK", Data: data, Total: total, Page: int32(page), PageSize: int32(pageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) SearchEnterpriseResourceAssigneeOptions(ctx context.Context, request *v1.SearchEnterpriseResourceAssigneeOptionsRequest) (*v1.SearchEnterpriseResourceAssigneeOptionsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrEnterpriseResourceInvalidArgument)
	if err != nil {
		return nil, err
	}
	items, total, err := s.usecase.SearchAssigneeOptions(ctx, principal.Organization.ID, request.GetKeyword(), page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.EnterpriseResourceAssigneeOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.EnterpriseResourceAssigneeOption{Id: item.ID.String(), Username: item.Username, DisplayName: item.DisplayName})
	}
	return &v1.SearchEnterpriseResourceAssigneeOptionsResponse{Success: true, Message: "OK", Data: data, Total: total, Page: int32(page), PageSize: int32(pageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) ListEnterpriseResourceRegionOptions(ctx context.Context, request *v1.ListEnterpriseResourceRegionOptionsRequest) (*v1.ListEnterpriseResourceRegionOptionsResponse, error) {
	if _, principalErr := biz.RequirePrincipal(ctx); principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrEnterpriseResourceInvalidArgument)
	if err != nil {
		return nil, err
	}
	items, total, err := s.usecase.ListRegionOptions(ctx, int(request.GetLevel()), request.ParentCode, page, pageSize)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.EnterpriseResourceRegionOption, 0, len(items))
	for _, item := range items {
		data = append(data, &v1.EnterpriseResourceRegionOption{Code: item.Code, Name: item.Name, Level: int32(item.Level), ParentCode: item.ParentCode})
	}
	return &v1.ListEnterpriseResourceRegionOptionsResponse{Success: true, Message: "OK", Data: data, Total: total, Page: int32(page), PageSize: int32(pageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) GetEnterpriseResourceCapabilities(ctx context.Context, _ *v1.GetEnterpriseResourceCapabilitiesRequest) (*v1.GetEnterpriseResourceCapabilitiesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	usedStorage, err := s.usecase.ImageUsage(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return &v1.GetEnterpriseResourceCapabilitiesResponse{
		Success:                true,
		Message:                "OK",
		ImageEnabled:           s.usecase.ImageStorageEnabled(),
		ImageMaxFileSize:       biz.EnterpriseImageMaxFileSize,
		ImageUsedStorageBytes:  usedStorage,
		ImageStorageQuotaBytes: 0,
		TraceId:                requestmeta.TraceID(ctx),
	}, nil
}

func (s *EnterpriseResourceService) PrepareEnterpriseResourceImageUpload(ctx context.Context, request *v1.PrepareEnterpriseResourceImageUploadRequest) (*v1.PrepareEnterpriseResourceImageUploadResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	result, err := s.usecase.PrepareImageUpload(ctx, principal.Organization.ID, request.GetFileName(), request.GetMimeType(), request.GetFileSize(), request.GetChecksum())
	if err != nil {
		return nil, err
	}
	return &v1.PrepareEnterpriseResourceImageUploadResponse{Success: true, Message: "OK", UploadUrl: result.UploadURL, ObjectKey: result.ObjectKey, Headers: result.Headers, ExpiresAt: result.ExpiresAt.Format(time.RFC3339), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) GetEnterpriseResourceImageAccess(ctx context.Context, request *v1.GetEnterpriseResourceImageAccessRequest) (*v1.GetEnterpriseResourceImageAccessResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	url, expiresAt, err := s.usecase.GetImageAccess(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetEnterpriseResourceImageAccessResponse{Success: true, Message: "OK", Url: url, ExpiresAt: expiresAt.Format(time.RFC3339), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) ListEnterpriseResources(ctx context.Context, request *v1.ListEnterpriseResourcesRequest) (*v1.ListEnterpriseResourcesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrEnterpriseResourceInvalidArgument)
	if err != nil {
		return nil, err
	}
	resourceType := enterpriseResourceTypeFromAPI(request.GetResourceType())
	options := biz.EnterpriseResourceListOptions{ResourceType: resourceType, Keyword: request.GetKeyword(), SortBy: request.GetSortBy(), SortDesc: request.GetSortOrder() == "desc", Page: page, PageSize: pageSize, Linked: request.Linked, Enabled: request.Enabled}
	if request.GetSortOrder() != "" && request.GetSortOrder() != "asc" && request.GetSortOrder() != "desc" {
		return nil, biz.ErrEnterpriseResourceInvalidArgument
	}
	if request.PartnerId != nil {
		value, err := uuid.Parse(request.GetPartnerId())
		if err != nil {
			return nil, biz.ErrEnterpriseResourceInvalidArgument
		}
		options.PartnerID = &value
	}
	if request.AssigneeId != nil {
		value, err := uuid.Parse(request.GetAssigneeId())
		if err != nil {
			return nil, biz.ErrEnterpriseResourceInvalidArgument
		}
		options.AssigneeID = &value
	}
	if request.AddressType != nil {
		value := enterpriseAddressTypeFromAPI(request.GetAddressType())
		options.AddressType = &value
	}
	items, total, err := s.usecase.List(ctx, principal.Organization.ID, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.EnterpriseResource, len(items))
	for i, item := range items {
		data[i] = enterpriseResourceToAPI(item)
	}
	return &v1.ListEnterpriseResourcesResponse{Success: true, Message: "OK", Data: data, Total: total, Page: int32(page), PageSize: int32(pageSize), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) GetEnterpriseResource(ctx context.Context, request *v1.GetEnterpriseResourceRequest) (*v1.GetEnterpriseResourceResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetEnterpriseResourceResponse{Success: true, Message: "OK", Data: enterpriseResourceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) CreateEnterpriseResource(ctx context.Context, request *v1.CreateEnterpriseResourceRequest) (*v1.CreateEnterpriseResourceResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := enterpriseResourceFromAPI(request.GetResource())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return &v1.CreateEnterpriseResourceResponse{Success: true, Message: "OK", Data: enterpriseResourceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) UpdateEnterpriseResource(ctx context.Context, request *v1.UpdateEnterpriseResourceRequest) (*v1.UpdateEnterpriseResourceResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	input, err := enterpriseResourceFromAPI(request.GetResource())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, id, input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateEnterpriseResourceResponse{Success: true, Message: "OK", Data: enterpriseResourceToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) DeleteEnterpriseResource(ctx context.Context, request *v1.DeleteEnterpriseResourceRequest) (*v1.DeleteEnterpriseResourceResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.usecase.Delete(ctx, principal.Organization.ID, principal.UserID, id); err != nil {
		return nil, err
	}
	return &v1.DeleteEnterpriseResourceResponse{Success: true, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) BatchCreateAssociations(ctx context.Context, request *v1.BatchCreateAssociationsRequest) (*v1.BatchCreateAssociationsResponse, error) {
	count, err := s.batchAssociations(ctx, request.GetResourceIds(), request.GetPartnerIds(), true)
	if err != nil {
		return nil, err
	}
	return &v1.BatchCreateAssociationsResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) BatchDeleteAssociations(ctx context.Context, request *v1.BatchDeleteAssociationsRequest) (*v1.BatchDeleteAssociationsResponse, error) {
	count, err := s.batchAssociations(ctx, request.GetResourceIds(), request.GetPartnerIds(), false)
	if err != nil {
		return nil, err
	}
	return &v1.BatchDeleteAssociationsResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) batchAssociations(ctx context.Context, resourceIDValues, partnerIDValues []string, create bool) (int, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return 0, principalErr
	}
	resourceIDs, err := enterpriseParseUUIDs(resourceIDValues)
	if err != nil {
		return 0, err
	}
	partnerIDs, err := enterpriseParseUUIDs(partnerIDValues)
	if err != nil {
		return 0, err
	}
	count, err := s.usecase.BatchPartners(ctx, principal.Organization.ID, principal.UserID, resourceIDs, partnerIDs, create)
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (s *EnterpriseResourceService) BatchAssignAddressTypes(ctx context.Context, request *v1.BatchAssignAddressTypesRequest) (*v1.BatchAssignAddressTypesResponse, error) {
	count, err := s.batchAddressTypes(ctx, request.GetResourceIds(), request.GetAddressTypes(), true)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignAddressTypesResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) BatchRemoveAddressTypes(ctx context.Context, request *v1.BatchRemoveAddressTypesRequest) (*v1.BatchRemoveAddressTypesResponse, error) {
	count, err := s.batchAddressTypes(ctx, request.GetResourceIds(), request.GetAddressTypes(), false)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveAddressTypesResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) batchAddressTypes(ctx context.Context, resourceIDValues []string, addressTypeValues []v1.EnterpriseAddressType, assign bool) (int, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return 0, principalErr
	}
	resourceIDs, err := enterpriseParseUUIDs(resourceIDValues)
	if err != nil {
		return 0, err
	}
	values := make([]biz.EnterpriseAddressType, len(addressTypeValues))
	for i, value := range addressTypeValues {
		values[i] = enterpriseAddressTypeFromAPI(value)
	}
	count, err := s.usecase.BatchAddressTypes(ctx, principal.Organization.ID, principal.UserID, resourceIDs, values, assign)
	if err != nil {
		return 0, err
	}
	return count, nil
}
func (s *EnterpriseResourceService) BatchAssignAssignees(ctx context.Context, request *v1.BatchAssignAssigneesRequest) (*v1.BatchAssignAssigneesResponse, error) {
	count, err := s.batchAssignees(ctx, request.GetResourceIds(), request.GetAssigneeIds(), true)
	if err != nil {
		return nil, err
	}
	return &v1.BatchAssignAssigneesResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) BatchRemoveAssignees(ctx context.Context, request *v1.BatchRemoveAssigneesRequest) (*v1.BatchRemoveAssigneesResponse, error) {
	count, err := s.batchAssignees(ctx, request.GetResourceIds(), request.GetAssigneeIds(), false)
	if err != nil {
		return nil, err
	}
	return &v1.BatchRemoveAssigneesResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) batchAssignees(ctx context.Context, resourceIDValues, assigneeIDValues []string, assign bool) (int, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return 0, principalErr
	}
	resourceIDs, err := enterpriseParseUUIDs(resourceIDValues)
	if err != nil {
		return 0, err
	}
	userIDs, err := enterpriseParseUUIDs(assigneeIDValues)
	if err != nil {
		return 0, err
	}
	count, err := s.usecase.BatchAssignees(ctx, principal.Organization.ID, principal.UserID, resourceIDs, userIDs, assign)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *EnterpriseResourceService) ListEnterpriseTagGroups(ctx context.Context, _ *v1.ListEnterpriseTagGroupsRequest) (*v1.ListEnterpriseTagGroupsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	items, err := s.usecase.ListTagGroups(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.EnterpriseTagGroup, len(items))
	for i, item := range items {
		data[i] = enterpriseTagGroupToAPI(item)
	}
	return &v1.ListEnterpriseTagGroupsResponse{Success: true, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) CreateEnterpriseTagGroup(ctx context.Context, request *v1.CreateEnterpriseTagGroupRequest) (*v1.CreateEnterpriseTagGroupResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input := enterpriseTagGroupFromAPI(request.GetGroup())
	item, err := s.usecase.CreateTagGroup(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return &v1.CreateEnterpriseTagGroupResponse{Success: true, Message: "OK", Data: enterpriseTagGroupToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) UpdateEnterpriseTagGroup(ctx context.Context, request *v1.UpdateEnterpriseTagGroupRequest) (*v1.UpdateEnterpriseTagGroupResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.UpdateTagGroup(ctx, principal.Organization.ID, principal.UserID, id, enterpriseTagGroupFromAPI(request.GetGroup()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdateEnterpriseTagGroupResponse{Success: true, Message: "OK", Data: enterpriseTagGroupToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *EnterpriseResourceService) DeleteEnterpriseTagGroup(ctx context.Context, request *v1.DeleteEnterpriseTagGroupRequest) (*v1.DeleteEnterpriseTagGroupResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.usecase.DeleteTagGroup(ctx, principal.Organization.ID, principal.UserID, id); err != nil {
		return nil, err
	}
	return &v1.DeleteEnterpriseTagGroupResponse{Success: true, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) PreviewEnterpriseResourceImport(ctx context.Context, request *v1.PreviewEnterpriseResourceImportRequest) (*v1.PreviewEnterpriseResourceImportResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	inputs, err := enterpriseResourceInputsFromAPI(request.GetRows())
	if err != nil {
		return nil, err
	}
	outcome, err := s.usecase.PreviewImport(ctx, principal.Organization.ID, inputs, enterpriseResourceTypeFromAPI(request.GetResourceType()))
	if err != nil {
		return nil, err
	}
	result := enterpriseResourceImportResult(outcome)
	return &v1.PreviewEnterpriseResourceImportResponse{Success: result.invalidCount == 0, Message: "OK", Rows: result.rows, ValidCount: result.validCount, InvalidCount: result.invalidCount, CreatedCount: 0, TraceId: requestmeta.TraceID(ctx), ConflictCount: result.conflictCount, OverwriteAllowed: result.overwriteAllowed}, nil
}
func (s *EnterpriseResourceService) CommitEnterpriseResourceImport(ctx context.Context, request *v1.CommitEnterpriseResourceImportRequest) (*v1.CommitEnterpriseResourceImportResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	inputs, err := enterpriseResourceInputsFromAPI(request.GetRows())
	if err != nil {
		return nil, err
	}
	outcome, err := s.usecase.CommitImport(ctx, principal.Organization.ID, principal.UserID, inputs, enterpriseResourceTypeFromAPI(request.GetResourceType()), request.GetOverwriteConflicts())
	if err != nil {
		return nil, err
	}
	result := enterpriseResourceImportResult(outcome)
	success := result.invalidCount == 0 && int(result.validCount) == outcome.CreatedCount+outcome.UpdatedCount
	return &v1.CommitEnterpriseResourceImportResponse{Success: success, Message: "OK", Rows: result.rows, ValidCount: result.validCount, InvalidCount: result.invalidCount, CreatedCount: int32(outcome.CreatedCount), TraceId: requestmeta.TraceID(ctx), ConflictCount: result.conflictCount, UpdatedCount: int32(outcome.UpdatedCount)}, nil
}

func enterpriseResourceFromAPI(value *v1.EnterpriseResourceInput) (*biz.EnterpriseResource, error) {
	if value == nil {
		return nil, biz.ErrEnterpriseResourceInvalidArgument
	}
	result := &biz.EnterpriseResource{ResourceType: enterpriseResourceTypeFromAPI(value.GetResourceType()), ShortName: value.GetShortName(), Enabled: value.GetEnabled(), SortOrder: int(value.GetSortOrder())}
	if value.PartnerAssociations != nil {
		ids, err := enterpriseParseUUIDs(value.GetPartnerAssociations().GetPartnerIds())
		if err != nil {
			return nil, err
		}
		result.PartnerIDs = ids
	}
	if value.AssigneeIds != nil {
		ids, err := enterpriseParseUUIDs(value.GetAssigneeIds())
		if err != nil {
			return nil, err
		}
		result.AssigneeIDs = ids
	}
	if value.AddressTypes != nil {
		result.AddressTypes = make([]biz.EnterpriseAddressType, len(value.GetAddressTypes()))
		for i, item := range value.GetAddressTypes() {
			result.AddressTypes[i] = enterpriseAddressTypeFromAPI(item)
		}
	}
	if item := value.GetAddress(); item != nil {
		result.Address = &biz.EnterpriseResourceAddress{ContactName: item.GetContactName(), ContactPhone: item.GetContactPhone(), CountryCode: item.GetCountryCode(), ProvinceCode: item.GetProvinceCode(), CityCode: item.GetCityCode(), DistrictCode: item.GetDistrictCode(), AddressDetail: item.GetAddressDetail(), Remark: item.GetRemark()}
	}
	if item := value.GetRemark(); item != nil {
		result.Remark = &biz.EnterpriseResourceRemark{RemarkType: biz.EnterpriseRemarkType(strings.TrimPrefix(item.GetRemarkType().String(), "ENTERPRISE_REMARK_TYPE_")), Content: item.GetContent()}
	}
	if item := value.GetParty(); item != nil {
		result.Party = &biz.EnterpriseResourceParty{CompanyName: item.GetCompanyName(), BusinessCode: item.GetBusinessCode(), Address: item.GetAddress(), CountryCode: item.GetCountryCode(), ContactName: item.GetContactName(), ContactPhone: item.GetContactPhone(), Email: item.GetEmail(), TaxIdentifier: item.GetTaxIdentifier(), AEOCode: item.GetAeoCode(), CustomDisplay: item.GetCustomDisplay(), DisplayContent: item.GetDisplayContent(), Remark: item.GetRemark()}
	}
	if item := value.GetImage(); item != nil {
		result.Image = &biz.EnterpriseResourceImage{FileName: item.GetFileName(), MIMEType: item.GetMimeType(), FileSize: item.GetFileSize(), ObjectKey: item.GetObjectKey(), Checksum: item.GetChecksum()}
		if item.Width != nil {
			width := int(item.GetWidth())
			result.Image.Width = &width
		}
		if item.Height != nil {
			height := int(item.GetHeight())
			result.Image.Height = &height
		}
	}
	if item := value.GetTag(); item != nil {
		id, err := uuid.Parse(item.GetGroupId())
		if err != nil {
			return nil, biz.ErrEnterpriseResourceInvalidArgument
		}
		result.Tag = &biz.EnterpriseResourceTag{GroupID: id}
	}
	return result, nil
}

func enterpriseResourceToAPI(value *biz.EnterpriseResource) *v1.EnterpriseResource {
	result := &v1.EnterpriseResource{Id: value.ID.String(), ResourceType: enterpriseResourceTypeToAPI(value.ResourceType), ShortName: value.ShortName, Enabled: value.Enabled, SortOrder: int32(value.SortOrder), PartnerIds: enterpriseUUIDStrings(value.PartnerIDs), AssigneeIds: enterpriseUUIDStrings(value.AssigneeIDs), CreatedBy: formatOptionalUUID(value.CreatedBy), UpdatedBy: formatOptionalUUID(value.UpdatedBy), CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339)}
	for _, item := range value.AddressTypes {
		result.AddressTypes = append(result.AddressTypes, enterpriseAddressTypeToAPI(item))
	}
	if item := value.Address; item != nil {
		result.Detail = &v1.EnterpriseResource_Address{Address: &v1.EnterpriseResourceAddress{ContactName: item.ContactName, ContactPhone: item.ContactPhone, CountryCode: item.CountryCode, ProvinceCode: item.ProvinceCode, CityCode: item.CityCode, DistrictCode: item.DistrictCode, AddressDetail: item.AddressDetail, Remark: item.Remark}}
	}
	if item := value.Remark; item != nil {
		result.Detail = &v1.EnterpriseResource_Remark{Remark: &v1.EnterpriseResourceRemark{RemarkType: enterpriseRemarkTypeToAPI(item.RemarkType), Content: item.Content}}
	}
	if item := value.Party; item != nil {
		result.Detail = &v1.EnterpriseResource_Party{Party: &v1.EnterpriseResourceParty{CompanyName: item.CompanyName, BusinessCode: item.BusinessCode, Address: item.Address, CountryCode: item.CountryCode, ContactName: item.ContactName, ContactPhone: item.ContactPhone, Email: item.Email, TaxIdentifier: item.TaxIdentifier, AeoCode: item.AEOCode, CustomDisplay: item.CustomDisplay, DisplayContent: item.DisplayContent, Remark: item.Remark}}
	}
	if item := value.Image; item != nil {
		detail := &v1.EnterpriseResourceImage{FileName: item.FileName, MimeType: item.MIMEType, FileSize: item.FileSize, ObjectKey: item.ObjectKey, Checksum: item.Checksum}
		if item.Width != nil {
			width := int32(*item.Width)
			detail.Width = &width
		}
		if item.Height != nil {
			height := int32(*item.Height)
			detail.Height = &height
		}
		result.Detail = &v1.EnterpriseResource_Image{Image: detail}
	}
	if item := value.Tag; item != nil {
		result.Detail = &v1.EnterpriseResource_Tag{Tag: &v1.EnterpriseResourceTag{GroupId: item.GroupID.String()}}
	}
	return result
}

func enterpriseResourceTypeFromAPI(value v1.EnterpriseResourceType) biz.EnterpriseResourceType {
	return biz.EnterpriseResourceType(strings.TrimPrefix(value.String(), "ENTERPRISE_RESOURCE_TYPE_"))
}
func enterpriseResourceTypeToAPI(value biz.EnterpriseResourceType) v1.EnterpriseResourceType {
	return v1.EnterpriseResourceType(v1.EnterpriseResourceType_value["ENTERPRISE_RESOURCE_TYPE_"+string(value)])
}
func enterpriseAddressTypeFromAPI(value v1.EnterpriseAddressType) biz.EnterpriseAddressType {
	return biz.EnterpriseAddressType(strings.TrimPrefix(value.String(), "ENTERPRISE_ADDRESS_TYPE_"))
}
func enterpriseAddressTypeToAPI(value biz.EnterpriseAddressType) v1.EnterpriseAddressType {
	return v1.EnterpriseAddressType(v1.EnterpriseAddressType_value["ENTERPRISE_ADDRESS_TYPE_"+string(value)])
}
func enterpriseRemarkTypeToAPI(value biz.EnterpriseRemarkType) v1.EnterpriseRemarkType {
	return v1.EnterpriseRemarkType(v1.EnterpriseRemarkType_value["ENTERPRISE_REMARK_TYPE_"+string(value)])
}
func enterpriseResourcePrincipalAndID(ctx context.Context, idText string) (*biz.Principal, uuid.UUID, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, uuid.Nil, principalErr
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return nil, uuid.Nil, biz.ErrEnterpriseResourceInvalidArgument
	}
	return principal, id, nil
}
func enterpriseParseUUIDs(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, len(values))
	for i, value := range values {
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, biz.ErrEnterpriseResourceInvalidArgument
		}
		result[i] = id
	}
	return result, nil
}
func enterpriseUUIDStrings(values []uuid.UUID) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.String()
	}
	return result
}
func enterpriseTagGroupFromAPI(value *v1.EnterpriseTagGroupInput) *biz.EnterpriseTagGroup {
	if value == nil {
		return nil
	}
	return &biz.EnterpriseTagGroup{Name: value.GetName(), Color: value.GetColor(), SortOrder: int(value.GetSortOrder())}
}
func enterpriseTagGroupToAPI(value *biz.EnterpriseTagGroup) *v1.EnterpriseTagGroup {
	return &v1.EnterpriseTagGroup{Id: value.ID.String(), Name: value.Name, Color: value.Color, SortOrder: int32(value.SortOrder), CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339)}
}
func enterpriseResourceInputsFromAPI(values []*v1.EnterpriseResourceInput) ([]*biz.EnterpriseResource, error) {
	result := make([]*biz.EnterpriseResource, len(values))
	for i, value := range values {
		item, err := enterpriseResourceFromAPI(value)
		if err != nil {
			return nil, err
		}
		result[i] = item
	}
	return result, nil
}

type enterpriseResourceImportResultValue struct {
	rows                     []*v1.EnterpriseResourceImportRow
	validCount, invalidCount int32
	conflictCount            int32
	overwriteAllowed         bool
}

func enterpriseResourceImportResult(outcome *biz.EnterpriseResourceImportOutcome) enterpriseResourceImportResultValue {
	rows := make([]*v1.EnterpriseResourceImportRow, len(outcome.RowErrors))
	conflictsByRow := make(map[int][]*v1.EnterpriseResourceImportConflict)
	for _, conflict := range outcome.Conflicts {
		conflictsByRow[conflict.RowNumber] = append(conflictsByRow[conflict.RowNumber], &v1.EnterpriseResourceImportConflict{
			ExistingResourceId: conflict.ExistingResourceID.String(), ExistingShortName: conflict.ExistingShortName, MatchedFields: conflict.MatchedFields,
		})
	}
	valid := 0
	for i, err := range outcome.RowErrors {
		row := &v1.EnterpriseResourceImportRow{RowNumber: int32(i + 1), Conflicts: conflictsByRow[i+1]}
		if i < len(outcome.Rows) && outcome.Rows[i] != nil {
			row.Resource = enterpriseResourceToInputAPI(outcome.Rows[i])
		}
		if err != nil {
			row.Errors = []string{err.Error()}
		} else {
			valid++
		}
		rows[i] = row
	}
	overwriteAllowed := true
	for _, conflicts := range conflictsByRow {
		if len(conflicts) > 1 {
			overwriteAllowed = false
			break
		}
	}
	return enterpriseResourceImportResultValue{rows: rows, validCount: int32(valid), invalidCount: int32(len(outcome.RowErrors) - valid), conflictCount: int32(len(outcome.Conflicts)), overwriteAllowed: overwriteAllowed}
}
func enterpriseResourceToInputAPI(value *biz.EnterpriseResource) *v1.EnterpriseResourceInput {
	result := &v1.EnterpriseResourceInput{ResourceType: enterpriseResourceTypeToAPI(value.ResourceType), ShortName: value.ShortName, Enabled: value.Enabled, SortOrder: int32(value.SortOrder)}
	if value.Party != nil {
		result.Detail = &v1.EnterpriseResourceInput_Party{Party: &v1.EnterpriseResourceParty{CompanyName: value.Party.CompanyName, BusinessCode: value.Party.BusinessCode, Address: value.Party.Address, CountryCode: value.Party.CountryCode, ContactName: value.Party.ContactName, ContactPhone: value.Party.ContactPhone, Email: value.Party.Email, TaxIdentifier: value.Party.TaxIdentifier, AeoCode: value.Party.AEOCode, CustomDisplay: value.Party.CustomDisplay, DisplayContent: value.Party.DisplayContent, Remark: value.Party.Remark}}
	}
	return result
}

var _ v1.EnterpriseResourceServiceServer = (*EnterpriseResourceService)(nil)
