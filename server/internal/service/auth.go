package service

import (
	"context"
	nethttp "net/http"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"
)

type AuthService struct {
	v1.UnimplementedAuthServiceServer
	usecase *biz.AuthUsecase
	policy  *biz.SessionPolicy
}

func NewAuthService(usecase *biz.AuthUsecase, policy *biz.SessionPolicy) *AuthService {
	return &AuthService{usecase: usecase, policy: policy}
}

func (s *AuthService) Login(ctx context.Context, request *v1.LoginRequest) (*v1.LoginResponse, error) {
	userAgent := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		userAgent = tr.RequestHeader().Get("User-Agent")
	}
	token, principal, expiresAt, err := s.usecase.Login(ctx, request.GetUsername(), request.GetPassword(), userAgent)
	if err != nil {
		return nil, err
	}
	s.setCookie(ctx, token, expiresAt, 0)
	return &v1.LoginResponse{Success: true, Code: 0, Message: "OK", Data: principalToAPI(principal), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AuthService) GetWeComLoginConfig(ctx context.Context, _ *v1.GetWeComLoginConfigRequest) (*v1.GetWeComLoginConfigResponse, error) {
	enabled, authorizeURL, state, expiresAt, err := s.usecase.StartWeComLogin()
	if err != nil {
		return nil, err
	}
	config := &v1.WeComLoginConfig{Enabled: enabled}
	if enabled {
		config.AuthorizeUrl = &authorizeURL
		s.setCookieNamed(ctx, s.wecomStateCookieName(), state, expiresAt, 300)
	}
	return &v1.GetWeComLoginConfigResponse{Success: true, Code: 0, Message: "OK", Data: config, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AuthService) WeComLogin(ctx context.Context, request *v1.WeComLoginRequest) (*v1.WeComLoginResponse, error) {
	userAgent := ""
	expectedState := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		userAgent = tr.RequestHeader().Get("User-Agent")
		expectedState = cookieValue(tr.RequestHeader().Get("Cookie"), s.wecomStateCookieName())
	}
	s.setCookieNamed(ctx, s.wecomStateCookieName(), "", time.Unix(1, 0), -1)
	token, principal, expiresAt, err := s.usecase.LoginWeCom(ctx, request.GetCode(), request.GetState(), expectedState, userAgent)
	if err != nil {
		return nil, err
	}
	s.setCookie(ctx, token, expiresAt, 0)
	return &v1.WeComLoginResponse{Success: true, Code: 0, Message: "OK", Data: principalToAPI(principal), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AuthService) Logout(ctx context.Context, _ *v1.LogoutRequest) (*v1.LogoutResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	if err := s.usecase.Logout(ctx, principal); err != nil {
		return nil, err
	}
	s.setCookie(ctx, "", time.Unix(1, 0), -1)
	return &v1.LogoutResponse{Success: true, Code: 0, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AuthService) Me(ctx context.Context, _ *v1.MeRequest) (*v1.MeResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	return &v1.MeResponse{Success: true, Code: 0, Message: "OK", Data: principalToAPI(principal), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AuthService) SwitchOrganization(ctx context.Context, request *v1.SwitchOrganizationRequest) (*v1.SwitchOrganizationResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	organizationID, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrOrganizationForbidden
	}
	next, err := s.usecase.SwitchOrganization(ctx, principal, organizationID)
	if err != nil {
		return nil, err
	}
	return &v1.SwitchOrganizationResponse{Success: true, Code: 0, Message: "OK", Data: principalToAPI(next), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *AuthService) setCookie(ctx context.Context, value string, expires time.Time, maxAge int) {
	s.setCookieNamed(ctx, s.policy.CookieName, value, expires, maxAge)
}

func (s *AuthService) setCookieNamed(ctx context.Context, name, value string, expires time.Time, maxAge int) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		cookie := &nethttp.Cookie{Name: name, Value: value, Path: "/", Expires: expires, MaxAge: maxAge, HttpOnly: true, Secure: s.policy.Secure, SameSite: parseSameSite(s.policy.SameSite)}
		tr.ReplyHeader().Add("Set-Cookie", cookie.String())
	}
}

func (s *AuthService) wecomStateCookieName() string { return s.policy.CookieName + "_wecom_state" }

func cookieValue(rawHeader, name string) string {
	request := &nethttp.Request{Header: nethttp.Header{"Cookie": []string{rawHeader}}}
	cookie, err := request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func parseSameSite(value string) nethttp.SameSite {
	if value == "strict" {
		return nethttp.SameSiteStrictMode
	}
	return nethttp.SameSiteLaxMode
}

func principalToAPI(principal *biz.Principal) *v1.CurrentUser {
	organizations := make([]*v1.Organization, 0, len(principal.Organizations))
	for _, organization := range principal.Organizations {
		organizations = append(organizations, &v1.Organization{Id: organization.ID.String(), Code: organization.Code, Name: organization.Name})
	}
	roleScopes := make([]*v1.RoleScope, 0, len(principal.RoleScopes))
	for _, roleScope := range principal.RoleScopes {
		roleScopes = append(roleScopes, &v1.RoleScope{RoleCode: roleScope.RoleCode, DataScope: string(roleScope.DataScope)})
	}
	return &v1.CurrentUser{Id: principal.UserID.String(), Username: principal.Username, DisplayName: principal.DisplayName, Email: principal.Email, CurrentOrganization: &v1.Organization{Id: principal.Organization.ID.String(), Code: principal.Organization.Code, Name: principal.Organization.Name}, Organizations: organizations, Permissions: principal.Permissions, RoleScopes: roleScopes}
}
