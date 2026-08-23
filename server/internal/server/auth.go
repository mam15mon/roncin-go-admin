package server

import (
	"context"
	"fmt"
	nethttp "net/http"

	"github.com/google/uuid"

	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	masterdatav1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	taskv1 "github.com/roncin/roncin-go-admin/server/api/task/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
)

func NewSessionPolicy(security *conf.Security) (*biz.SessionPolicy, error) {
	if security == nil || security.GetSession() == nil {
		return nil, fmt.Errorf("security session configuration is required")
	}
	session := security.GetSession()
	if session.GetCookieName() == "" || session.GetTtl() == nil || session.GetTtl().AsDuration() <= 0 {
		return nil, fmt.Errorf("session cookie name and positive ttl are required")
	}
	if session.GetSameSite() != "lax" && session.GetSameSite() != "strict" {
		return nil, fmt.Errorf("session same_site must be lax or strict")
	}
	return &biz.SessionPolicy{CookieName: session.GetCookieName(), TTL: session.GetTtl().AsDuration(), Secure: session.GetSecure(), SameSite: session.GetSameSite()}, nil
}

func Authorization(usecase *biz.AuthUsecase, policy *biz.SessionPolicy, orderUsecase *biz.OrderUsecase) middleware.Middleware {
	publicOperations := map[string]struct{}{
		authv1.OperationAuthServiceLogin:               {},
		authv1.OperationAuthServiceGetWeComLoginConfig: {},
		authv1.OperationAuthServiceWeComLogin:          {},
	}
	authenticatedOperations := map[string]struct{}{
		authv1.OperationAuthServiceLogout:             {},
		authv1.OperationAuthServiceMe:                 {},
		authv1.OperationAuthServiceSwitchOrganization: {},
	}
	permissionOperations := map[string]permissionRule{
		adminv1.OperationAdminServiceListOrganizations:                                {key: access.OrganizationRead, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceCreateOrganization:                               {key: access.OrganizationCreate, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceUpdateOrganization:                               {key: access.OrganizationUpdate, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceListUsers:                                        {key: access.UserRead, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceCreateUser:                                       {key: access.UserCreate, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceUpdateUser:                                       {key: access.UserUpdate, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceAuthorizeWeComUser:                               {key: access.UserAuthorizeWeCom, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceResetUserPassword:                                {key: access.UserResetPassword, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListRoles:                                        {key: access.RoleRead, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListOrganizationRoles:                            {key: access.RoleRead, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceCreateRole:                                       {key: access.RoleCreate, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceUpdateRole:                                       {key: access.RoleUpdate, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListPermissions:                                  {key: access.PermissionRead, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListAuditLogs:                                    {key: access.AuditRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceGetPartner:                                   {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartners:                                 {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerAssignmentOptions:                 {key: access.PartnerAssignmentOptionRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartner:                                {key: access.PartnerCreate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartner:                                {key: access.PartnerUpdate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceSetSupplierBlacklist:                         {key: access.PartnerBlacklist, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerAccounts:                          {key: access.PartnerAccountRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerAccount:                         {key: access.PartnerAccountCreate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerAccount:                         {key: access.PartnerAccountUpdate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerContracts:                         {key: access.PartnerContractRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerContract:                        {key: access.PartnerContractCreate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerContract:                        {key: access.PartnerContractUpdate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerSettlementRules:                   {key: access.PartnerSettlementRuleRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerSettlementRule:                  {key: access.PartnerSettlementRuleCreate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerSettlementRule:                  {key: access.PartnerSettlementRuleUpdate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerAttachments:                       {key: access.PartnerAttachmentRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceRegisterPartnerAttachment:                    {key: access.PartnerAttachmentRegister, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceImportPartners:                               {key: access.PartnerImport, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceExportPartners:                               {key: access.PartnerExport, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerShippingPresets:                   {key: access.PartnerShippingPresetRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerAuditLogs:                         {key: access.PartnerAuditRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerShippingPreset:                  {key: access.PartnerShippingPresetCreate, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerShippingPreset:                  {key: access.PartnerShippingPresetUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListCurrencies:                         {key: access.MasterDataCurrencyRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListAdministrativeRegions:              {key: access.MasterDataAdministrativeRegionRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListItems:                              {key: access.MasterDataItemRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListOptions:                            {key: access.MasterDataOptionRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateItem:                             {key: access.MasterDataItemCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateItem:                             {key: access.MasterDataItemUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceImportItems:                            {key: access.MasterDataItemImport, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListPorts:                              {key: access.MasterDataPortRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreatePort:                             {key: access.MasterDataPortCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdatePort:                             {key: access.MasterDataPortUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListAirports:                           {key: access.MasterDataAirportRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateAirport:                          {key: access.MasterDataAirportCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateAirport:                          {key: access.MasterDataAirportUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListAirlines:                           {key: access.MasterDataAirlineRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateAirline:                          {key: access.MasterDataAirlineCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateAirline:                          {key: access.MasterDataAirlineUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListShippingLines:                      {key: access.MasterDataShippingLineRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateShippingLine:                     {key: access.MasterDataShippingLineCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateShippingLine:                     {key: access.MasterDataShippingLineUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListNumberRules:                        {key: access.MasterDataNumberRuleRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateNumberRule:                       {key: access.MasterDataNumberRuleCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateNumberRule:                       {key: access.MasterDataNumberRuleUpdate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListStatusTemplates:                    {key: access.MasterDataStatusTemplateRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateStatusTemplate:                   {key: access.MasterDataStatusTemplateCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServicePublishStatusTemplate:                  {key: access.MasterDataStatusTemplatePublish, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceSetDefaultStatusTemplate:               {key: access.MasterDataStatusTemplateSetDefault, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListMilestoneTemplates:                 {key: access.MasterDataMilestoneTemplateRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateMilestoneTemplate:                {key: access.MasterDataMilestoneTemplateCreate, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServicePublishMilestoneTemplate:               {key: access.MasterDataMilestoneTemplatePublish, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceSetDefaultMilestoneTemplate:            {key: access.MasterDataMilestoneTemplateSetDefault, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceGetOrder:                                         {orderOperation: access.OrderRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceListOrders:                                       {orderOperation: access.OrderRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceCheckOrderReference:                              {orderOperation: access.OrderCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceCreateOrder:                                      {orderOperation: access.OrderCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceUpdateOrder:                                      {orderOperation: access.OrderUpdate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceTransitionOrderStatus:                            {orderOperation: access.OrderTransition, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderMilestoneServiceListMilestones:                          {orderOperation: access.OrderMilestoneRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderMilestoneServiceSetMilestone:                            {orderOperation: access.OrderMilestoneSet, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderAttachmentServiceListAttachments:                        {orderOperation: access.OrderAttachmentRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderAttachmentServiceRegisterAttachment:                     {orderOperation: access.OrderAttachmentRegister, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderPersonnelServiceListPersonnel:                           {orderOperation: access.OrderPersonnelRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderPersonnelServiceAssignPersonnel:                         {orderOperation: access.OrderPersonnelAssign, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderPersonnelServiceRemovePersonnel:                         {orderOperation: access.OrderPersonnelRemove, scope: biz.DataScopeOrganization},
		taskv1.OperationBackgroundTaskServiceListBackgroundTasks:                      {key: access.TaskRead, scope: biz.DataScopeOrganization},
		taskv1.OperationBackgroundTaskServiceGetBackgroundTask:                        {key: access.TaskRead, scope: biz.DataScopeOrganization},
		taskv1.OperationBackgroundTaskServiceRequeueBackgroundTask:                    {key: access.TaskRequeue, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderContainerServiceListContainers:                          {orderOperation: access.OrderContainerRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderContainerServiceAddContainer:                            {orderOperation: access.OrderContainerCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderContainerServiceUpdateContainer:                         {orderOperation: access.OrderContainerUpdate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderContainerServiceRemoveContainer:                         {orderOperation: access.OrderContainerDelete, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderCargoItemServiceListCargoItems:                          {orderOperation: access.OrderCargoItemRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderCargoItemServiceAddCargoItem:                            {orderOperation: access.OrderCargoItemCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderCargoItemServiceUpdateCargoItem:                         {orderOperation: access.OrderCargoItemUpdate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderCargoItemServiceRemoveCargoItem:                         {orderOperation: access.OrderCargoItemDelete, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderShippingDocumentServiceListShippingDocuments:            {orderOperation: access.OrderShippingDocumentRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderShippingDocumentServiceAddShippingDocument:              {orderOperation: access.OrderShippingDocumentCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderShippingDocumentServiceUpdateShippingDocument:           {orderOperation: access.OrderShippingDocumentUpdate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderShippingDocumentServiceTransitionShippingDocumentStatus: {orderOperation: access.OrderShippingDocumentTransition, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderShippingDocumentServiceRemoveShippingDocument:           {orderOperation: access.OrderShippingDocumentDelete, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderAbnormalCaseServiceListAbnormalCases:                    {orderOperation: access.OrderAbnormalCaseRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderAbnormalCaseServiceMarkAbnormalCase:                     {orderOperation: access.OrderAbnormalCaseCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderAbnormalCaseServiceResolveAbnormalCase:                  {orderOperation: access.OrderAbnormalCaseResolve, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderAbnormalCaseServiceRemoveAbnormalCase:                   {orderOperation: access.OrderAbnormalCaseDelete, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderReleasePodServiceListReleasePods:                        {orderOperation: access.OrderReleasePodRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderReleasePodServiceAddReleasePod:                          {orderOperation: access.OrderReleasePodCreate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderReleasePodServiceUpdateReleasePod:                       {orderOperation: access.OrderReleasePodUpdate, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderReleasePodServiceTransitionReleasePodStatus:             {orderOperation: access.OrderReleasePodTransition, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderReleasePodServiceRemoveReleasePod:                       {orderOperation: access.OrderReleasePodDelete, scope: biz.DataScopeOrganization},
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, biz.ErrSessionRequired
			}
			operation := tr.Operation()
			if _, isPublic := publicOperations[operation]; isPublic {
				return handler(ctx, request)
			}
			_, requiresSession := authenticatedOperations[operation]
			rule, requiresPermission := permissionOperations[operation]
			if !requiresSession && !requiresPermission {
				return nil, biz.ErrPermissionDenied
			}
			principal, err := usecase.AuthenticateSession(ctx, cookieValue(tr.RequestHeader().Get("Cookie"), policy.CookieName))
			if err != nil {
				return nil, err
			}
			effectivePrincipal := principal
			if requiresPermission && rule.orderOperation != "" {
				if order, directOrderRequest := requestOrder(ctx, request, orderUsecase); directOrderRequest {
					if order == nil || !principal.CanAccessOrderOrganization(order.OrganizationID, orderOperationWrites(rule.orderOperation)) {
						return nil, biz.ErrPermissionDenied
					}
					copy := *principal
					copy.Organization.ID = order.OrganizationID
					effectivePrincipal = &copy
				}
			}
			if requiresPermission && !hasPermission(ctx, request, effectivePrincipal, rule, orderUsecase) {
				return nil, biz.ErrPermissionDenied
			}
			return handler(biz.WithPrincipal(ctx, effectivePrincipal), request)
		}
	}
}

func requestOrder(ctx context.Context, request any, orderUsecase *biz.OrderUsecase) (*biz.Order, bool) {
	var orderID string
	switch value := request.(type) {
	case *orderv1.ListOrdersRequest, *orderv1.CreateOrderRequest, *orderv1.CheckOrderReferenceRequest:
		return nil, false
	case interface{ GetOrderId() string }:
		orderID = value.GetOrderId()
	case interface{ GetId() string }:
		orderID = value.GetId()
	default:
		return nil, false
	}
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, true
	}
	order, err := orderUsecase.Find(ctx, id)
	if err != nil {
		return nil, true
	}
	return order, true
}

func orderOperationWrites(operation access.OrderOperation) bool {
	switch operation {
	case access.OrderRead, access.OrderMilestoneRead, access.OrderAttachmentRead, access.OrderPersonnelRead, access.OrderContainerRead, access.OrderCargoItemRead, access.OrderShippingDocumentRead, access.OrderAbnormalCaseRead, access.OrderReleasePodRead:
		return false
	default:
		return true
	}
}

type permissionRule struct {
	key            string
	scope          biz.DataScope
	orderOperation access.OrderOperation
}

func hasPermission(ctx context.Context, request any, principal *biz.Principal, rule permissionRule, orderUsecase *biz.OrderUsecase) bool {
	if rule.orderOperation == "" {
		return principal.HasPermissionInScope(rule.key, rule.scope)
	}
	if _, ok := request.(*orderv1.CheckOrderReferenceRequest); ok {
		return hasAnyOrderPermission(principal, rule.orderOperation, rule.scope)
	}
	businessType, ok := requestOrderBusinessType(ctx, request, principal.Organization.ID, orderUsecase)
	if !ok {
		return false
	}
	return principal.HasPermissionInScope(access.OrderPermission(businessType, rule.orderOperation), rule.scope)
}

func hasAnyOrderPermission(principal *biz.Principal, operation access.OrderOperation, scope biz.DataScope) bool {
	for _, businessType := range []access.OrderBusinessType{access.OrderBusinessSE, access.OrderBusinessSI, access.OrderBusinessAE, access.OrderBusinessAI} {
		if principal.HasPermissionInScope(access.OrderPermission(businessType, operation), scope) {
			return true
		}
	}
	return false
}

func requestOrderBusinessType(ctx context.Context, request any, organizationID uuid.UUID, orderUsecase *biz.OrderUsecase) (access.OrderBusinessType, bool) {
	switch value := request.(type) {
	case *orderv1.ListOrdersRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.CreateOrderRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	}

	var orderID string
	switch value := request.(type) {
	case interface{ GetOrderId() string }:
		orderID = value.GetOrderId()
	case interface{ GetId() string }:
		orderID = value.GetId()
	default:
		return "", false
	}
	id, err := uuid.Parse(orderID)
	if err != nil {
		return "", false
	}
	order, err := orderUsecase.Get(ctx, organizationID, id)
	if err != nil {
		return "", false
	}
	return orderBusinessTypeFromBiz(order.BusinessType)
}

func orderBusinessTypeFromAPI(value orderv1.BusinessType) (access.OrderBusinessType, bool) {
	switch value {
	case orderv1.BusinessType_BUSINESS_TYPE_SE:
		return access.OrderBusinessSE, true
	case orderv1.BusinessType_BUSINESS_TYPE_SI:
		return access.OrderBusinessSI, true
	case orderv1.BusinessType_BUSINESS_TYPE_AE:
		return access.OrderBusinessAE, true
	case orderv1.BusinessType_BUSINESS_TYPE_AI:
		return access.OrderBusinessAI, true
	default:
		return "", false
	}
}

func orderBusinessTypeFromBiz(value biz.OrderBusinessType) (access.OrderBusinessType, bool) {
	switch value {
	case biz.OrderBusinessSE:
		return access.OrderBusinessSE, true
	case biz.OrderBusinessSI:
		return access.OrderBusinessSI, true
	case biz.OrderBusinessAE:
		return access.OrderBusinessAE, true
	case biz.OrderBusinessAI:
		return access.OrderBusinessAI, true
	default:
		return "", false
	}
}

func cookieValue(rawHeader, name string) string {
	request := &nethttp.Request{Header: nethttp.Header{"Cookie": []string{rawHeader}}}
	cookie, err := request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
