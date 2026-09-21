package server

import (
	"context"
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/google/uuid"

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

func Authorization(usecase *biz.AuthUsecase, policy *biz.SessionPolicy, orderUsecase *biz.OrderUsecase, partnerUsecase *biz.PartnerUsecase) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, biz.ErrSessionRequired
			}
			operation := tr.Operation()
			rule, declared := operationAccessRules[operation]
			if !declared {
				return nil, biz.ErrPermissionDenied
			}
			if rule.mode == accessModePublic {
				return handler(ctx, request)
			}
			principal, err := usecase.AuthenticateSession(ctx, cookieValue(tr.RequestHeader().Get("Cookie"), policy.CookieName))
			if err != nil {
				return nil, err
			}
			effectivePrincipal := principal
			directOrderRequest := false
			directPartnerRequest := false
			if rule.mode == accessModeOrderPermission {
				var order *biz.Order
				order, directOrderRequest = requestOrder(ctx, request, orderUsecase, orderOrganizationScopes(principal, rule.orderOperation, orderOperationWrites(rule.orderOperation)))
				if directOrderRequest {
					if order == nil {
						return nil, biz.ErrPermissionDenied
					}
					copy := *principal
					copy.WorkspaceOrganizationID = principal.Organization.ID
					copy.Organization.ID = order.OrganizationID
					effectivePrincipal = &copy
				}
			}
			if rule.mode == accessModePermission && isPartnerPermission(rule.permission) {
				var partner *biz.Partner
				writable, _ := partnerPermissionWrites(rule.permission)
				partner, directPartnerRequest = requestPartner(ctx, request, partnerUsecase, partnerOrganizationIDs(principal, rule.permission, writable))
				if directPartnerRequest {
					if partner == nil {
						return nil, biz.ErrPermissionDenied
					}
				}
			}
			if (rule.mode == accessModePermission || rule.mode == accessModeOrderPermission) && !directOrderRequest && !directPartnerRequest && !hasPermission(request, effectivePrincipal, rule) {
				return nil, biz.ErrPermissionDenied
			}
			return handler(biz.WithPrincipal(ctx, effectivePrincipal), request)
		}
	}
}

// requestPartner 对携带往来单位主键的请求，以“ID + 当前接口具体权限的组织范围”
// 定位当前公司主档；账户、合同、附件和审计等子资源同样先校验所属公司，
// 不改变会话工作台，也不以跨公司主档反向切换下游 Service 的有效组织。
func requestPartner(ctx context.Context, request any, partnerUsecase *biz.PartnerUsecase, organizationIDs []uuid.UUID) (*biz.Partner, bool) {
	var partnerID string
	switch value := request.(type) {
	case *partnerv1.ListPartnersRequest,
		*partnerv1.CreatePartnerRequest,
		*partnerv1.ImportPartnersRequest,
		*partnerv1.ExportPartnersRequest,
		*partnerv1.ListPartnerAssignmentOptionsRequest,
		*partnerv1.SearchPartnerAssignmentOptionsRequest:
		return nil, false
	case interface{ GetPartnerId() string }:
		partnerID = value.GetPartnerId()
	case *partnerv1.GetPartnerRequest:
		partnerID = value.GetId()
	case *partnerv1.UpdatePartnerRequest:
		partnerID = value.GetId()
	case *partnerv1.SetPartnerRoleBlacklistRequest:
		partnerID = value.GetId()
	default:
		return nil, false
	}
	id, err := uuid.Parse(partnerID)
	if err != nil || partnerUsecase == nil || len(organizationIDs) == 0 {
		return nil, true
	}
	partner, err := partnerUsecase.FindAuthorized(ctx, id, organizationIDs)
	if err != nil {
		return nil, true
	}
	return partner, true
}

func isPartnerPermission(permission string) bool {
	_, known := partnerPermissionWrites(permission)
	return known
}

func partnerPermissionWrites(permission string) (bool, bool) {
	switch permission {
	case access.PartnerRead,
		access.PartnerExport,
		access.PartnerAccountRead,
		access.PartnerContractRead,
		access.PartnerSettlementRuleRead,
		access.PartnerAttachmentRead,
		access.PartnerShippingPresetRead,
		access.PartnerAuditRead,
		access.PartnerAssignmentOptionRead:
		return false, true
	case access.PartnerCreate,
		access.PartnerBlacklist,
		access.PartnerImport,
		access.PartnerAccountCreate,
		access.PartnerAccountUpdate,
		access.PartnerContractCreate,
		access.PartnerContractUpdate,
		access.PartnerSettlementRuleCreate,
		access.PartnerSettlementRuleUpdate,
		access.PartnerAttachmentRegister,
		access.PartnerShippingPresetCreate,
		access.PartnerShippingPresetUpdate,
		access.PartnerUpdate:
		return true, true
	default:
		return false, false
	}
}

func partnerOrganizationIDs(principal *biz.Principal, permission string, writable bool) []uuid.UUID {
	if principal == nil {
		return nil
	}
	scope, err := principal.ResolvePermissionOrganizationScope(permission)
	if err != nil {
		return nil
	}
	if writable {
		return scope.WritableOrganizationIDs
	}
	return scope.ReadableOrganizationIDs
}

func requestOrder(ctx context.Context, request any, orderUsecase *biz.OrderUsecase, scopes []biz.OrderOrganizationScope) (*biz.Order, bool) {
	var orderID string
	switch value := request.(type) {
	case *orderv1.ListOrdersRequest, *orderv1.CreateOrderRequest, *orderv1.CheckOrderReferenceRequest, *orderv1.ListPersonnelOptionsRequest:
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
	if orderUsecase == nil || len(scopes) == 0 {
		return nil, true
	}
	order, err := orderUsecase.FindAuthorized(ctx, id, scopes)
	if err != nil {
		return nil, true
	}
	return order, true
}

func orderOperationWrites(operation access.OrderOperation) bool {
	switch operation {
	case access.OrderRead,
		access.OrderMilestoneRead,
		access.OrderAttachmentRead,
		access.OrderPersonnelRead,
		access.OrderContainerRead,
		access.OrderCargoItemRead,
		access.OrderAbnormalCaseRead,
		access.OrderReleasePodRead,
		access.OrderFeeRead:
		return false
	default:
		return true
	}
}

func hasPermission(request any, principal *biz.Principal, rule accessRule) bool {
	if rule.orderOperation == "" {
		if isScopedFinancePermission(rule.permission) {
			writable, known := scopedFinancePermissionWrites(rule.permission)
			if !known {
				return false
			}
			organizationIDs := organizationIDsForPermission(principal, rule.permission, writable)
			// 财务资源的真实组织由 Service 按具体权限查询来源或目标对象；中间件
			// 读取保留跨组织授权，经营写范围已经由 biz 收敛到当前公司。
			return len(organizationIDs) > 0
		}
		if strings.HasPrefix(rule.permission, "business.partner.") {
			writable, known := partnerPermissionWrites(rule.permission)
			if !known {
				return false
			}
			organizationIDs := partnerOrganizationIDs(principal, rule.permission, writable)
			if isPartnerCollectionRequest(request) {
				return len(organizationIDs) > 0
			}
			return containsOrganizationID(organizationIDs, principal.Organization.ID)
		}
		return principal.HasPermissionInScope(rule.permission, rule.scope)
	}
	if _, ok := request.(*orderv1.CheckOrderReferenceRequest); ok {
		return hasAnyOrderPermissionInCurrentOrganization(principal, rule.orderOperation, orderOperationWrites(rule.orderOperation))
	}
	if list, ok := request.(*orderv1.ListOrdersRequest); ok && list.BusinessType == nil {
		return hasAnyOrderPermission(principal, rule.orderOperation, orderOperationWrites(rule.orderOperation))
	}
	businessType, ok := requestOrderBusinessType(request)
	if !ok {
		return false
	}
	return currentOrganizationInOrderScope(principal, businessType, rule.orderOperation, orderOperationWrites(rule.orderOperation))
}

// 财务资源的最终组织授权由 SettlementService 传入当前动作的显式 allowed IDs，
// 并由仓储执行 ID + OrganizationIDIn 查询。中间件仅验证该权限本身且存在对应范围。
func scopedFinancePermissionWrites(permission string) (bool, bool) {
	switch permission {
	case access.FinanceBillRead, access.FinanceInvoiceRead, access.FinanceCashflowRead,
		access.FinanceVerificationRead, access.FinanceNettingRead, access.FinanceCommissionRead,
		access.FinanceCommissionExport, access.FinanceFeeRead:
		return false, true
	case access.FinanceBillCreate, access.FinanceBillUpdate, access.FinanceBillConfirm,
		access.FinanceInvoiceCreate, access.FinanceInvoiceUpdate, access.FinanceCashflowCreate,
		access.FinanceCashflowUpdate, access.FinanceVerificationCreate, access.FinanceVerificationReverse,
		access.FinanceNettingCreate, access.FinanceNettingConfirm, access.FinanceNettingReverse,
		access.FinanceCommissionManage, access.FinanceFeeTag:
		return true, true
	default:
		return false, false
	}
}

func isScopedFinancePermission(permission string) bool {
	_, known := scopedFinancePermissionWrites(permission)
	return known
}

func organizationIDsForPermission(principal *biz.Principal, permission string, writable bool) []uuid.UUID {
	if principal == nil {
		return nil
	}
	scope, err := principal.ResolvePermissionOrganizationScope(permission)
	if err != nil {
		return nil
	}
	if writable {
		return scope.WritableOrganizationIDs
	}
	return scope.ReadableOrganizationIDs
}

func containsOrganizationID(organizationIDs []uuid.UUID, organizationID uuid.UUID) bool {
	for _, candidate := range organizationIDs {
		if candidate == organizationID {
			return true
		}
	}
	return false
}

func isPartnerCollectionRequest(request any) bool {
	switch request.(type) {
	case *partnerv1.ListPartnersRequest, *partnerv1.ExportPartnersRequest:
		return true
	default:
		return false
	}
}

func hasAnyOrderPermission(principal *biz.Principal, operation access.OrderOperation, writable bool) bool {
	return len(orderOrganizationScopes(principal, operation, writable)) > 0
}

func hasAnyOrderPermissionInCurrentOrganization(principal *biz.Principal, operation access.OrderOperation, writable bool) bool {
	if principal == nil {
		return false
	}
	for _, businessType := range access.OrderBusinessTypes() {
		if currentOrganizationInOrderScope(principal, businessType, operation, writable) {
			return true
		}
	}
	return false
}

func requestOrderBusinessType(request any) (access.OrderBusinessType, bool) {
	switch value := request.(type) {
	case *orderv1.ListOrdersRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.CreateOrderRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.ListPersonnelOptionsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.ListOrderTagOptionsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.BatchAssignOrderTagsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.BatchRemoveOrderTagsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.MatchSeaMasterBillCandidateRequest:
		// 共享主单候选匹配仅适用于海运出口订单，且服务实现只查询当前组织。
		return access.OrderBusinessSE, true
	}

	return "", false
}

func orderOrganizationScopes(principal *biz.Principal, operation access.OrderOperation, writable bool) []biz.OrderOrganizationScope {
	if principal == nil {
		return nil
	}
	scopes := make([]biz.OrderOrganizationScope, 0, len(access.OrderBusinessTypes()))
	for _, businessType := range access.OrderBusinessTypes() {
		permission := access.OrderPermission(businessType, operation)
		if permission == "" {
			continue
		}
		scope, err := principal.ResolvePermissionOrganizationScope(permission)
		if err != nil {
			continue
		}
		organizationIDs := scope.ReadableOrganizationIDs
		if writable {
			organizationIDs = scope.WritableOrganizationIDs
		}
		if len(organizationIDs) == 0 {
			continue
		}
		mapped, ok := orderBusinessTypeToBiz(businessType)
		if !ok {
			continue
		}
		scopes = append(scopes, biz.OrderOrganizationScope{BusinessType: mapped, OrganizationIDs: organizationIDs})
	}
	return scopes
}

func currentOrganizationInOrderScope(principal *biz.Principal, businessType access.OrderBusinessType, operation access.OrderOperation, writable bool) bool {
	permission := access.OrderPermission(businessType, operation)
	if permission == "" || principal == nil {
		return false
	}
	return principal.CanAccessOrganizationForPermission(permission, principal.Organization.ID, writable)
}

func orderBusinessTypeToBiz(value access.OrderBusinessType) (biz.OrderBusinessType, bool) {
	switch value {
	case access.OrderBusinessSE:
		return biz.OrderBusinessSE, true
	case access.OrderBusinessSI:
		return biz.OrderBusinessSI, true
	case access.OrderBusinessAE:
		return biz.OrderBusinessAE, true
	case access.OrderBusinessAI:
		return biz.OrderBusinessAI, true
	case access.OrderBusinessLand:
		return biz.OrderBusinessLand, true
	case access.OrderBusinessRail:
		return biz.OrderBusinessRail, true
	default:
		return "", false
	}
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
	case orderv1.BusinessType_BUSINESS_TYPE_LAND:
		return access.OrderBusinessLand, true
	case orderv1.BusinessType_BUSINESS_TYPE_RAIL:
		return access.OrderBusinessRail, true
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
	case biz.OrderBusinessLand:
		return access.OrderBusinessLand, true
	case biz.OrderBusinessRail:
		return access.OrderBusinessRail, true
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
