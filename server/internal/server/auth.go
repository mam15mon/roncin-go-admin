package server

import (
	"context"
	"fmt"
	nethttp "net/http"

	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
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
	permissionOperations := map[string]string{}
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
			requiredPermission, requiresPermission := permissionOperations[operation]
			if !requiresSession && !requiresPermission {
				return nil, biz.ErrPermissionDenied
			}
			principal, err := usecase.AuthenticateSession(ctx, cookieValue(tr.RequestHeader().Get("Cookie"), policy.CookieName))
			if err != nil {
				return nil, err
			}
			if requiresPermission && !principal.HasPermission(requiredPermission) {
				return nil, biz.ErrPermissionDenied
			}
			return handler(biz.WithPrincipal(ctx, principal), request)
		}
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
