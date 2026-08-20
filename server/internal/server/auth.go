package server

import (
	"context"
	"fmt"
	nethttp "net/http"

	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	masterdatav1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
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

func Authorization(usecase *biz.AuthUsecase, policy *biz.SessionPolicy) middleware.Middleware {
	publicOperations := map[string]struct{}{authv1.OperationAuthServiceLogin: {}}
	authenticatedOperations := map[string]struct{}{
		authv1.OperationAuthServiceLogout:             {},
		authv1.OperationAuthServiceMe:                 {},
		authv1.OperationAuthServiceSwitchOrganization: {},
	}
	permissionOperations := map[string]permissionRule{
		adminv1.OperationAdminServiceListOrganizations:                     {key: access.OrganizationManage, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceCreateOrganization:                    {key: access.OrganizationManage, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceUpdateOrganization:                    {key: access.OrganizationManage, scope: biz.DataScopeAll},
		adminv1.OperationAdminServiceListUsers:                             {key: access.UserManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceCreateUser:                            {key: access.UserManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceUpdateUser:                            {key: access.UserManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceResetUserPassword:                     {key: access.UserManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListRoles:                             {key: access.RoleManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceCreateRole:                            {key: access.RoleManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceUpdateRole:                            {key: access.RoleManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListPermissions:                       {key: access.RoleManage, scope: biz.DataScopeOrganization},
		adminv1.OperationAdminServiceListAuditLogs:                         {key: access.AuditRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceGetPartner:                        {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartners:                      {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartner:                     {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartner:                     {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceSetSupplierBlacklist:              {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerAccounts:               {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerAccount:              {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerAccount:              {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerContracts:              {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerContract:             {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerContract:             {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerSettlementRules:        {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceCreatePartnerSettlementRule:       {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceUpdatePartnerSettlementRule:       {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceListPartnerAttachments:            {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceRegisterPartnerAttachment:         {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceImportPartners:                    {key: access.PartnerManage, scope: biz.DataScopeOrganization},
		partnerv1.OperationPartnerServiceExportPartners:                    {key: access.PartnerRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListItems:                   {key: access.MasterDataRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListOptions:                 {key: access.MasterDataRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateItem:                  {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateItem:                  {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceImportItems:                 {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListNumberRules:             {key: access.MasterDataRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateNumberRule:            {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceUpdateNumberRule:            {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListStatusTemplates:         {key: access.MasterDataRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateStatusTemplate:        {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServicePublishStatusTemplate:       {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceSetDefaultStatusTemplate:    {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceListMilestoneTemplates:      {key: access.MasterDataRead, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceCreateMilestoneTemplate:     {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServicePublishMilestoneTemplate:    {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		masterdatav1.OperationMasterDataServiceSetDefaultMilestoneTemplate: {key: access.MasterDataManage, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceGetOrder:                              {key: access.OrderRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceListOrders:                            {key: access.OrderRead, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceCreateOrder:                           {key: access.OrderManage, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceUpdateOrder:                           {key: access.OrderManage, scope: biz.DataScopeOrganization},
		orderv1.OperationOrderServiceTransitionOrderStatus:                 {key: access.OrderManage, scope: biz.DataScopeOrganization},
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
			if requiresPermission && !principal.HasPermissionInScope(rule.key, rule.scope) {
				return nil, biz.ErrPermissionDenied
			}
			return handler(biz.WithPrincipal(ctx, principal), request)
		}
	}
}

type permissionRule struct {
	key   string
	scope biz.DataScope
}

func cookieValue(rawHeader, name string) string {
	request := &nethttp.Request{Header: nethttp.Header{"Cookie": []string{rawHeader}}}
	cookie, err := request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}
