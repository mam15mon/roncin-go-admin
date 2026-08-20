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
	usecase                *biz.MasterDataUsecase
	orderConfigUsecase     *biz.OrderConfigUsecase
	milestoneConfigUsecase *biz.MilestoneConfigUsecase
}

func NewMasterDataService(usecase *biz.MasterDataUsecase, orderConfigUsecase *biz.OrderConfigUsecase, milestoneConfigUsecase *biz.MilestoneConfigUsecase) *MasterDataService {
	return &MasterDataService{
		usecase:                usecase,
		orderConfigUsecase:     orderConfigUsecase,
		milestoneConfigUsecase: milestoneConfigUsecase,
	}
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

func (s *MasterDataService) ListNumberRules(ctx context.Context, _ *v1.ListNumberRulesRequest) (*v1.NumberRuleListReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.orderConfigUsecase.ListNumberRules(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return &v1.NumberRuleListReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    numberRulesToAPI(rules),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) CreateNumberRule(ctx context.Context, request *v1.CreateNumberRuleRequest) (*v1.NumberRuleReply, error) {
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
	return &v1.NumberRuleReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    numberRuleToAPI(created),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) UpdateNumberRule(ctx context.Context, request *v1.UpdateNumberRuleRequest) (*v1.NumberRuleReply, error) {
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
	return &v1.NumberRuleReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    numberRuleToAPI(updated),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) ListStatusTemplates(ctx context.Context, request *v1.ListStatusTemplatesRequest) (*v1.StatusTemplateListReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var published *bool
	if request.Published != nil {
		p := request.GetPublished()
		published = &p
	}
	templates, err := s.orderConfigUsecase.ListStatusTemplates(ctx, principal.Organization.ID, businessTypeFromAPI(request.GetBusinessType()), published)
	if err != nil {
		return nil, err
	}
	return &v1.StatusTemplateListReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    statusTemplatesToAPI(templates),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) CreateStatusTemplate(ctx context.Context, request *v1.CreateStatusTemplateRequest) (*v1.StatusTemplateReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.StatusTemplateItem, 0, len(request.GetItems()))
	for _, item := range request.GetItems() {
		if item == nil {
			return nil, biz.ErrStatusTemplateInvalid
		}
		items = append(items, &biz.StatusTemplateItem{
			Code:       item.GetCode(),
			Label:      item.GetLabel(),
			SortOrder:  int(item.GetSortOrder()),
			Enabled:    item.Enabled == nil || item.GetEnabled(),
			ColorToken: optionalString(item.GetColorToken(), item.ColorToken != nil),
			System:     item.GetSystem(),
		})
	}
	created, err := s.orderConfigUsecase.CreateStatusTemplate(ctx, principal.Organization.ID, principal.UserID, &biz.StatusTemplate{
		Code:         request.GetCode(),
		Name:         request.GetName(),
		BusinessType: businessTypeFromAPI(request.GetBusinessType()),
		Version:      int(request.GetVersion()),
		Items:        items,
	})
	if err != nil {
		return nil, err
	}
	return &v1.StatusTemplateReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    statusTemplateToAPI(created),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) PublishStatusTemplate(ctx context.Context, request *v1.PublishStatusTemplateRequest) (*v1.StatusTemplateReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrStatusTemplateInvalid
	}
	published, err := s.orderConfigUsecase.PublishStatusTemplate(ctx, principal.Organization.ID, principal.UserID, id, request.GetIsDefault())
	if err != nil {
		return nil, err
	}
	return &v1.StatusTemplateReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    statusTemplateToAPI(published),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) SetDefaultStatusTemplate(ctx context.Context, request *v1.SetDefaultStatusTemplateRequest) (*v1.StatusTemplateReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrStatusTemplateInvalid
	}
	updated, err := s.orderConfigUsecase.SetDefaultStatusTemplate(ctx, principal.Organization.ID, principal.UserID, id)
	if err != nil {
		return nil, err
	}
	return &v1.StatusTemplateReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    statusTemplateToAPI(updated),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) ListMilestoneTemplates(ctx context.Context, request *v1.ListMilestoneTemplatesRequest) (*v1.MilestoneTemplateListReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	var tradeTerm *string
	if request.TradeTerm != nil {
		t := request.GetTradeTerm()
		tradeTerm = &t
	}
	var published *bool
	if request.Published != nil {
		p := request.GetPublished()
		published = &p
	}
	templates, err := s.milestoneConfigUsecase.List(ctx, principal.Organization.ID, biz.MilestoneTemplateListOptions{
		BusinessType: businessTypeFromAPI(request.GetBusinessType()),
		TradeTerm:    tradeTerm,
		Published:    published,
	})
	if err != nil {
		return nil, err
	}
	return &v1.MilestoneTemplateListReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    milestoneTemplatesToAPI(templates),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) CreateMilestoneTemplate(ctx context.Context, request *v1.CreateMilestoneTemplateRequest) (*v1.MilestoneTemplateReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.MilestoneTemplateItem, 0, len(request.GetItems()))
	for _, item := range request.GetItems() {
		if item == nil {
			return nil, biz.ErrMilestoneTemplateInvalid
		}
		var dependsOn []string
		if item.DependsOn != nil {
			dependsOn = make([]string, len(item.DependsOn))
			copy(dependsOn, item.DependsOn)
		}
		items = append(items, &biz.MilestoneTemplateItem{
			Code:        item.GetCode(),
			Label:       item.GetLabel(),
			Description: optionalString(item.GetDescription(), item.Description != nil),
			Category:    optionalString(item.GetCategory(), item.Category != nil),
			SortOrder:   int(item.GetSortOrder()),
			Enabled:     item.Enabled == nil || item.GetEnabled(),
			DependsOn:   dependsOn,
		})
	}
	created, err := s.milestoneConfigUsecase.Create(ctx, principal.Organization.ID, principal.UserID, &biz.MilestoneTemplate{
		Code:         request.GetCode(),
		Name:         request.GetName(),
		BusinessType: businessTypeFromAPI(request.GetBusinessType()),
		TradeTerm:    request.GetTradeTerm(),
		Version:      int(request.GetVersion()),
		Items:        items,
	})
	if err != nil {
		return nil, err
	}
	return &v1.MilestoneTemplateReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    milestoneTemplateToAPI(created),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) PublishMilestoneTemplate(ctx context.Context, request *v1.PublishMilestoneTemplateRequest) (*v1.MilestoneTemplateReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrMilestoneTemplateInvalid
	}
	published, err := s.milestoneConfigUsecase.Publish(ctx, principal.Organization.ID, principal.UserID, id, request.GetIsDefault())
	if err != nil {
		return nil, err
	}
	return &v1.MilestoneTemplateReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    milestoneTemplateToAPI(published),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
}

func (s *MasterDataService) SetDefaultMilestoneTemplate(ctx context.Context, request *v1.SetDefaultMilestoneTemplateRequest) (*v1.MilestoneTemplateReply, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrMilestoneTemplateInvalid
	}
	updated, err := s.milestoneConfigUsecase.SetDefault(ctx, principal.Organization.ID, principal.UserID, id)
	if err != nil {
		return nil, err
	}
	return &v1.MilestoneTemplateReply{
		Success: true,
		Code:    0,
		Message: "OK",
		Data:    milestoneTemplateToAPI(updated),
		TraceId: requestmeta.TraceID(ctx),
	}, nil
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
	return &v1.MasterDataItem{
		Id:             item.ID.String(),
		OrganizationId: item.OrganizationID.String(),
		Kind:           masterDataKindToAPI(item.Kind),
		Code:           item.Code,
		Name:           item.Name,
		NameEn:         item.NameEN,
		ParentCode:     item.ParentCode,
		TransportMode:  item.TransportMode,
		TeuFactor:      item.TEUFactor,
		Source:         item.Source,
		SortOrder:      int32(item.SortOrder),
		Enabled:        item.Enabled,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

func documentTypeFromAPI(value v1.DocumentType) biz.DocumentType {
	switch value {
	case v1.DocumentType_DOCUMENT_TYPE_ORDER:
		return biz.DocumentTypeOrder
	case v1.DocumentType_DOCUMENT_TYPE_BOOKING:
		return biz.DocumentTypeBooking
	case v1.DocumentType_DOCUMENT_TYPE_HBL:
		return biz.DocumentTypeHBL
	case v1.DocumentType_DOCUMENT_TYPE_MBL:
		return biz.DocumentTypeMBL
	case v1.DocumentType_DOCUMENT_TYPE_BILL:
		return biz.DocumentTypeBill
	case v1.DocumentType_DOCUMENT_TYPE_STATEMENT:
		return biz.DocumentTypeStatement
	case v1.DocumentType_DOCUMENT_TYPE_PAYMENT:
		return biz.DocumentTypePayment
	case v1.DocumentType_DOCUMENT_TYPE_INVOICE:
		return biz.DocumentTypeInvoice
	default:
		return ""
	}
}

func documentTypeToAPI(value biz.DocumentType) v1.DocumentType {
	switch value {
	case biz.DocumentTypeOrder:
		return v1.DocumentType_DOCUMENT_TYPE_ORDER
	case biz.DocumentTypeBooking:
		return v1.DocumentType_DOCUMENT_TYPE_BOOKING
	case biz.DocumentTypeHBL:
		return v1.DocumentType_DOCUMENT_TYPE_HBL
	case biz.DocumentTypeMBL:
		return v1.DocumentType_DOCUMENT_TYPE_MBL
	case biz.DocumentTypeBill:
		return v1.DocumentType_DOCUMENT_TYPE_BILL
	case biz.DocumentTypeStatement:
		return v1.DocumentType_DOCUMENT_TYPE_STATEMENT
	case biz.DocumentTypePayment:
		return v1.DocumentType_DOCUMENT_TYPE_PAYMENT
	case biz.DocumentTypeInvoice:
		return v1.DocumentType_DOCUMENT_TYPE_INVOICE
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

func statusTemplatesToAPI(items []*biz.StatusTemplate) []*v1.StatusTemplate {
	result := make([]*v1.StatusTemplate, 0, len(items))
	for _, item := range items {
		result = append(result, statusTemplateToAPI(item))
	}
	return result
}

func statusTemplateToAPI(item *biz.StatusTemplate) *v1.StatusTemplate {
	var publishedAt *string
	if item.PublishedAt != nil {
		formatted := item.PublishedAt.Format(time.RFC3339)
		publishedAt = &formatted
	}
	return &v1.StatusTemplate{
		Id:             item.ID.String(),
		OrganizationId: item.OrganizationID.String(),
		Code:           item.Code,
		Name:           item.Name,
		BusinessType:   businessTypeToAPI(item.BusinessType),
		Version:        int32(item.Version),
		IsDefault:      item.IsDefault,
		PublishedAt:    publishedAt,
		Enabled:        item.Enabled,
		Items:          statusTemplateItemsToAPI(item.Items),
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

func statusTemplateItemsToAPI(items []*biz.StatusTemplateItem) []*v1.StatusTemplateItem {
	result := make([]*v1.StatusTemplateItem, 0, len(items))
	for _, item := range items {
		result = append(result, statusTemplateItemToAPI(item))
	}
	return result
}

func statusTemplateItemToAPI(item *biz.StatusTemplateItem) *v1.StatusTemplateItem {
	return &v1.StatusTemplateItem{
		Id:         item.ID.String(),
		Code:       item.Code,
		Label:      item.Label,
		SortOrder:  int32(item.SortOrder),
		Enabled:    item.Enabled,
		ColorToken: item.ColorToken,
		System:     item.System,
	}
}

func milestoneTemplatesToAPI(items []*biz.MilestoneTemplate) []*v1.MilestoneTemplate {
	result := make([]*v1.MilestoneTemplate, 0, len(items))
	for _, item := range items {
		result = append(result, milestoneTemplateToAPI(item))
	}
	return result
}

func milestoneTemplateToAPI(item *biz.MilestoneTemplate) *v1.MilestoneTemplate {
	var publishedAt *string
	if item.PublishedAt != nil {
		formatted := item.PublishedAt.Format(time.RFC3339)
		publishedAt = &formatted
	}
	return &v1.MilestoneTemplate{
		Id:             item.ID.String(),
		OrganizationId: item.OrganizationID.String(),
		Code:           item.Code,
		Name:           item.Name,
		BusinessType:   businessTypeToAPI(item.BusinessType),
		TradeTerm:      item.TradeTerm,
		Version:        int32(item.Version),
		IsDefault:      item.IsDefault,
		PublishedAt:    publishedAt,
		Enabled:        item.Enabled,
		Items:          milestoneTemplateItemsToAPI(item.Items),
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

func milestoneTemplateItemsToAPI(items []*biz.MilestoneTemplateItem) []*v1.MilestoneTemplateItem {
	result := make([]*v1.MilestoneTemplateItem, 0, len(items))
	for _, item := range items {
		result = append(result, milestoneTemplateItemToAPI(item))
	}
	return result
}

func milestoneTemplateItemToAPI(item *biz.MilestoneTemplateItem) *v1.MilestoneTemplateItem {
	var dependsOn []string
	if item.DependsOn != nil {
		dependsOn = make([]string, len(item.DependsOn))
		copy(dependsOn, item.DependsOn)
	}
	return &v1.MilestoneTemplateItem{
		Id:          item.ID.String(),
		Code:        item.Code,
		Label:       item.Label,
		Description: item.Description,
		Category:    item.Category,
		SortOrder:   int32(item.SortOrder),
		Enabled:     item.Enabled,
		DependsOn:   dependsOn,
	}
}

var _ v1.MasterDataServiceServer = (*MasterDataService)(nil)
