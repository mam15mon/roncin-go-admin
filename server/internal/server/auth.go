package server

import (
	"context"
	"fmt"
	nethttp "net/http"

	"github.com/google/uuid"

	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
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
			if rule.mode == accessModeOrderPermission {
				if order, directOrderRequest := requestOrder(ctx, request, orderUsecase); directOrderRequest {
					if order == nil || !principal.CanAccessOrderOrganization(order.OrganizationID, orderOperationWrites(rule.orderOperation)) {
						return nil, biz.ErrPermissionDenied
					}
					copy := *principal
					copy.Organization.ID = order.OrganizationID
					effectivePrincipal = &copy
				}
			}
			if (rule.mode == accessModePermission || rule.mode == accessModeOrderPermission) && !hasPermission(ctx, request, effectivePrincipal, rule, orderUsecase) {
				return nil, biz.ErrPermissionDenied
			}
			return handler(biz.WithPrincipal(ctx, effectivePrincipal), request)
		}
	}
}

func requestOrder(ctx context.Context, request any, orderUsecase *biz.OrderUsecase) (*biz.Order, bool) {
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
	order, err := orderUsecase.Find(ctx, id)
	if err != nil {
		return nil, true
	}
	return order, true
}

func orderOperationWrites(operation access.OrderOperation) bool {
	switch operation {
	case access.OrderRead, access.OrderMilestoneRead, access.OrderAttachmentRead, access.OrderPersonnelRead, access.OrderContainerRead, access.OrderCargoItemRead, access.OrderAbnormalCaseRead, access.OrderReleasePodRead:
		return false
	default:
		return true
	}
}

func hasPermission(ctx context.Context, request any, principal *biz.Principal, rule accessRule, orderUsecase *biz.OrderUsecase) bool {
	if rule.orderOperation == "" {
		return principal.HasPermissionInScope(rule.permission, rule.scope)
	}
	if _, ok := request.(*orderv1.CheckOrderReferenceRequest); ok {
		return hasAnyOrderPermission(principal, rule.orderOperation, rule.scope)
	}
	if list, ok := request.(*orderv1.ListOrdersRequest); ok && list.BusinessType == nil {
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
	case *orderv1.ListPersonnelOptionsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.ListOrderTagOptionsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.BatchAssignOrderTagsRequest:
		return orderBusinessTypeFromAPI(value.GetBusinessType())
	case *orderv1.BatchRemoveOrderTagsRequest:
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
