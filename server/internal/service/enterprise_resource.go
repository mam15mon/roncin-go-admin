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

func (s *EnterpriseResourceService) GetEnterpriseResourceCapabilities(ctx context.Context, _ *v1.GetEnterpriseResourceCapabilitiesRequest) (*v1.GetEnterpriseResourceCapabilitiesResponse, error) {
	return &v1.GetEnterpriseResourceCapabilitiesResponse{
		Success:          true,
		Message:          "OK",
		ImageEnabled:     s.usecase.ImageStorageEnabled(),
		ImageMaxFileSize: biz.EnterpriseImageMaxFileSize,
		TraceId:          requestmeta.TraceID(ctx),
	}, nil
}

func (s *EnterpriseResourceService) PrepareEnterpriseResourceImageUpload(ctx context.Context, request *v1.PrepareEnterpriseResourceImageUploadRequest) (*v1.PrepareEnterpriseResourceImageUploadResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
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
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	page, pageSize, err := enterpriseResourcePageValues(request.GetPage(), request.GetPageSize())
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

func (s *EnterpriseResourceService) GetEnterpriseResource(ctx context.Context, request *v1.GetEnterpriseResourceRequest) (*v1.EnterpriseResourceResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Get(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return enterpriseResourceResponse(ctx, item), nil
}

func (s *EnterpriseResourceService) CreateEnterpriseResource(ctx context.Context, request *v1.CreateEnterpriseResourceRequest) (*v1.EnterpriseResourceResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	input, err := enterpriseResourceFromAPI(request.GetResource())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return enterpriseResourceResponse(ctx, item), nil
}

func (s *EnterpriseResourceService) UpdateEnterpriseResource(ctx context.Context, request *v1.UpdateEnterpriseResourceRequest) (*v1.EnterpriseResourceResponse, error) {
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
	return enterpriseResourceResponse(ctx, item), nil
}

func (s *EnterpriseResourceService) DeleteEnterpriseResource(ctx context.Context, request *v1.DeleteEnterpriseResourceRequest) (*v1.MutationResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.usecase.Delete(ctx, principal.Organization.ID, principal.UserID, id); err != nil {
		return nil, err
	}
	return &v1.MutationResponse{Success: true, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) BatchCreateAssociations(ctx context.Context, request *v1.BatchAssociationRequest) (*v1.BatchMutationResponse, error) {
	return s.batchAssociations(ctx, request, true)
}
func (s *EnterpriseResourceService) BatchDeleteAssociations(ctx context.Context, request *v1.BatchAssociationRequest) (*v1.BatchMutationResponse, error) {
	return s.batchAssociations(ctx, request, false)
}
func (s *EnterpriseResourceService) batchAssociations(ctx context.Context, request *v1.BatchAssociationRequest, create bool) (*v1.BatchMutationResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	resourceIDs, err := enterpriseParseUUIDs(request.GetResourceIds())
	if err != nil {
		return nil, err
	}
	partnerIDs, err := enterpriseParseUUIDs(request.GetPartnerIds())
	if err != nil {
		return nil, err
	}
	count, err := s.usecase.BatchPartners(ctx, principal.Organization.ID, principal.UserID, resourceIDs, partnerIDs, create)
	if err != nil {
		return nil, err
	}
	return batchMutationResponse(ctx, count), nil
}
func (s *EnterpriseResourceService) BatchAssignAddressTypes(ctx context.Context, request *v1.BatchAddressTypeRequest) (*v1.BatchMutationResponse, error) {
	return s.batchAddressTypes(ctx, request, true)
}
func (s *EnterpriseResourceService) BatchRemoveAddressTypes(ctx context.Context, request *v1.BatchAddressTypeRequest) (*v1.BatchMutationResponse, error) {
	return s.batchAddressTypes(ctx, request, false)
}
func (s *EnterpriseResourceService) batchAddressTypes(ctx context.Context, request *v1.BatchAddressTypeRequest, assign bool) (*v1.BatchMutationResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	resourceIDs, err := enterpriseParseUUIDs(request.GetResourceIds())
	if err != nil {
		return nil, err
	}
	values := make([]biz.EnterpriseAddressType, len(request.GetAddressTypes()))
	for i, value := range request.GetAddressTypes() {
		values[i] = enterpriseAddressTypeFromAPI(value)
	}
	count, err := s.usecase.BatchAddressTypes(ctx, principal.Organization.ID, principal.UserID, resourceIDs, values, assign)
	if err != nil {
		return nil, err
	}
	return batchMutationResponse(ctx, count), nil
}
func (s *EnterpriseResourceService) BatchAssignAssignees(ctx context.Context, request *v1.BatchAssigneeRequest) (*v1.BatchMutationResponse, error) {
	return s.batchAssignees(ctx, request, true)
}
func (s *EnterpriseResourceService) BatchRemoveAssignees(ctx context.Context, request *v1.BatchAssigneeRequest) (*v1.BatchMutationResponse, error) {
	return s.batchAssignees(ctx, request, false)
}
func (s *EnterpriseResourceService) batchAssignees(ctx context.Context, request *v1.BatchAssigneeRequest, assign bool) (*v1.BatchMutationResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	resourceIDs, err := enterpriseParseUUIDs(request.GetResourceIds())
	if err != nil {
		return nil, err
	}
	userIDs, err := enterpriseParseUUIDs(request.GetAssigneeIds())
	if err != nil {
		return nil, err
	}
	count, err := s.usecase.BatchAssignees(ctx, principal.Organization.ID, principal.UserID, resourceIDs, userIDs, assign)
	if err != nil {
		return nil, err
	}
	return batchMutationResponse(ctx, count), nil
}

func (s *EnterpriseResourceService) ListEnterpriseTagGroups(ctx context.Context, _ *v1.ListEnterpriseTagGroupsRequest) (*v1.ListEnterpriseTagGroupsResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
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
func (s *EnterpriseResourceService) CreateEnterpriseTagGroup(ctx context.Context, request *v1.CreateEnterpriseTagGroupRequest) (*v1.EnterpriseTagGroupResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	input := enterpriseTagGroupFromAPI(request.GetGroup())
	item, err := s.usecase.CreateTagGroup(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return enterpriseTagGroupResponse(ctx, item), nil
}
func (s *EnterpriseResourceService) UpdateEnterpriseTagGroup(ctx context.Context, request *v1.UpdateEnterpriseTagGroupRequest) (*v1.EnterpriseTagGroupResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	item, err := s.usecase.UpdateTagGroup(ctx, principal.Organization.ID, principal.UserID, id, enterpriseTagGroupFromAPI(request.GetGroup()))
	if err != nil {
		return nil, err
	}
	return enterpriseTagGroupResponse(ctx, item), nil
}
func (s *EnterpriseResourceService) DeleteEnterpriseTagGroup(ctx context.Context, request *v1.DeleteEnterpriseTagGroupRequest) (*v1.MutationResponse, error) {
	principal, id, err := enterpriseResourcePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.usecase.DeleteTagGroup(ctx, principal.Organization.ID, principal.UserID, id); err != nil {
		return nil, err
	}
	return &v1.MutationResponse{Success: true, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *EnterpriseResourceService) PreviewEnterpriseResourceImport(ctx context.Context, request *v1.PreviewEnterpriseResourceImportRequest) (*v1.EnterpriseResourceImportResponse, error) {
	if _, ok := biz.PrincipalFromContext(ctx); !ok {
		return nil, biz.ErrSessionRequired
	}
	inputs, err := enterpriseResourceInputsFromAPI(request.GetRows())
	if err != nil {
		return nil, err
	}
	values, errs := s.usecase.PreviewImport(inputs, enterpriseResourceTypeFromAPI(request.GetResourceType()))
	return enterpriseResourceImportResponse(ctx, values, errs, 0), nil
}
func (s *EnterpriseResourceService) CommitEnterpriseResourceImport(ctx context.Context, request *v1.CommitEnterpriseResourceImportRequest) (*v1.EnterpriseResourceImportResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	inputs, err := enterpriseResourceInputsFromAPI(request.GetRows())
	if err != nil {
		return nil, err
	}
	values, rowErrors, err := s.usecase.CommitImport(ctx, principal.Organization.ID, principal.UserID, inputs, enterpriseResourceTypeFromAPI(request.GetResourceType()))
	if err != nil && len(rowErrors) == 0 {
		return nil, err
	}
	return enterpriseResourceImportResponse(ctx, values, rowErrors, len(values)), nil
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
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, uuid.Nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		return nil, uuid.Nil, biz.ErrEnterpriseResourceInvalidArgument
	}
	return principal, id, nil
}
func enterpriseResourcePageValues(page, pageSize int32) (int, int, error) {
	p, ps := int(page), int(pageSize)
	if p == 0 {
		p = 1
	}
	if ps == 0 {
		ps = 20
	}
	if !biz.ValidListPagination(p, ps) {
		return 0, 0, biz.ErrEnterpriseResourceInvalidArgument
	}
	return p, ps, nil
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
func enterpriseResourceResponse(ctx context.Context, item *biz.EnterpriseResource) *v1.EnterpriseResourceResponse {
	return &v1.EnterpriseResourceResponse{Success: true, Message: "OK", Data: enterpriseResourceToAPI(item), TraceId: requestmeta.TraceID(ctx)}
}
func batchMutationResponse(ctx context.Context, count int) *v1.BatchMutationResponse {
	return &v1.BatchMutationResponse{Success: true, Message: "OK", AffectedCount: int32(count), TraceId: requestmeta.TraceID(ctx)}
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
func enterpriseTagGroupResponse(ctx context.Context, item *biz.EnterpriseTagGroup) *v1.EnterpriseTagGroupResponse {
	return &v1.EnterpriseTagGroupResponse{Success: true, Message: "OK", Data: enterpriseTagGroupToAPI(item), TraceId: requestmeta.TraceID(ctx)}
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
func enterpriseResourceImportResponse(ctx context.Context, values []*biz.EnterpriseResource, errs []error, created int) *v1.EnterpriseResourceImportResponse {
	rows := make([]*v1.EnterpriseResourceImportRow, len(errs))
	valid := 0
	for i, err := range errs {
		row := &v1.EnterpriseResourceImportRow{RowNumber: int32(i + 1)}
		if i < len(values) && values[i] != nil {
			row.Resource = enterpriseResourceToInputAPI(values[i])
		}
		if err != nil {
			row.Errors = []string{err.Error()}
		} else {
			valid++
		}
		rows[i] = row
	}
	return &v1.EnterpriseResourceImportResponse{Success: len(errs)-valid == 0, Message: "OK", Rows: rows, ValidCount: int32(valid), InvalidCount: int32(len(errs) - valid), CreatedCount: int32(created), TraceId: requestmeta.TraceID(ctx)}
}
func enterpriseResourceToInputAPI(value *biz.EnterpriseResource) *v1.EnterpriseResourceInput {
	result := &v1.EnterpriseResourceInput{ResourceType: enterpriseResourceTypeToAPI(value.ResourceType), ShortName: value.ShortName, Enabled: value.Enabled, SortOrder: int32(value.SortOrder)}
	if value.Party != nil {
		result.Detail = &v1.EnterpriseResourceInput_Party{Party: &v1.EnterpriseResourceParty{CompanyName: value.Party.CompanyName, BusinessCode: value.Party.BusinessCode, Address: value.Party.Address, CountryCode: value.Party.CountryCode, ContactName: value.Party.ContactName, ContactPhone: value.Party.ContactPhone, Email: value.Party.Email, TaxIdentifier: value.Party.TaxIdentifier, AeoCode: value.Party.AEOCode, CustomDisplay: value.Party.CustomDisplay, DisplayContent: value.Party.DisplayContent, Remark: value.Party.Remark}}
	}
	return result
}

var _ v1.EnterpriseResourceServiceServer = (*EnterpriseResourceService)(nil)
