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
	token, principal, expiresAt, err := s.usecase.Login(ctx, request.GetUsername(), request.GetPassword(), userAgent, requestmeta.IPAddress(ctx))
	if err != nil {
		return nil, err
	}
	s.setCookie(ctx, token, expiresAt, 0)
	return ok(ctx, &v1.LoginResponse{Data: principalToAPI(principal)}), nil
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
	return ok(ctx, &v1.GetWeComLoginConfigResponse{Data: config}), nil
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
	return ok(ctx, &v1.WeComLoginResponse{Data: principalToAPI(principal)}), nil
}

func (s *AuthService) GetDingTalkLoginConfig(ctx context.Context, _ *v1.GetDingTalkLoginConfigRequest) (*v1.GetDingTalkLoginConfigResponse, error) {
	enabled, authorizeURL, state, expiresAt, err := s.usecase.StartDingTalkLogin()
	if err != nil {
		return nil, err
	}
	config := &v1.DingTalkLoginConfig{Enabled: enabled}
	if enabled {
		config.AuthorizeUrl = &authorizeURL
		s.setCookieNamed(ctx, s.dingTalkStateCookieName(), state, expiresAt, 300)
		s.setCookieNamed(ctx, s.dingTalkRegistrationTokenCookieName(), "", time.Unix(1, 0), -1)
	}
	return ok(ctx, &v1.GetDingTalkLoginConfigResponse{Data: config}), nil
}

func (s *AuthService) DingTalkLogin(ctx context.Context, request *v1.DingTalkLoginRequest) (*v1.DingTalkLoginResponse, error) {
	userAgent := ""
	expectedState := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		userAgent = tr.RequestHeader().Get("User-Agent")
		expectedState = cookieValue(tr.RequestHeader().Get("Cookie"), s.dingTalkStateCookieName())
	}
	s.setCookieNamed(ctx, s.dingTalkStateCookieName(), "", time.Unix(1, 0), -1)
	result, err := s.usecase.LoginDingTalk(ctx, request.GetAuthCode(), request.GetState(), expectedState, userAgent)
	if err != nil {
		return nil, err
	}
	data := &v1.DingTalkLoginResult{}
	switch result.Status {
	case biz.DingTalkLoginStatusAuthenticated:
		data.Status = v1.DingTalkLoginStatus_DING_TALK_LOGIN_STATUS_AUTHENTICATED
		data.CurrentUser = principalToAPI(result.Principal)
		s.setCookie(ctx, result.SessionToken, result.SessionExpiresAt, 0)
	case biz.DingTalkLoginStatusRegistrationRequired:
		data.Status = v1.DingTalkLoginStatus_DING_TALK_LOGIN_STATUS_REGISTRATION_REQUIRED
		data.DisplayName = &result.DisplayName
		s.setCookieNamed(ctx, s.dingTalkRegistrationTokenCookieName(), result.RegistrationToken, result.RegistrationExpiresAt, 300)
	default:
		return nil, biz.ErrDingTalkLoginFailed
	}
	return ok(ctx, &v1.DingTalkLoginResponse{Data: data}), nil
}

func (s *AuthService) RegisterDingTalkUser(ctx context.Context, _ *v1.RegisterDingTalkUserRequest) (*v1.RegisterDingTalkUserResponse, error) {
	registrationToken := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		registrationToken = cookieValue(tr.RequestHeader().Get("Cookie"), s.dingTalkRegistrationTokenCookieName())
	}
	registration, err := s.usecase.ConfirmDingTalkRegistration(ctx, registrationToken)
	if err != nil {
		return nil, err
	}
	s.setCookieNamed(ctx, s.dingTalkRegistrationTokenCookieName(), "", time.Unix(1, 0), -1)
	return ok(ctx, &v1.RegisterDingTalkUserResponse{
		Data: &v1.DingTalkRegistration{DisplayName: registration.DisplayName, Status: registration.Status},
	}), nil
}

func (s *AuthService) Logout(ctx context.Context, _ *v1.LogoutRequest) (*v1.LogoutResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	if err := s.usecase.Logout(ctx, principal); err != nil {
		return nil, err
	}
	s.setCookie(ctx, "", time.Unix(1, 0), -1)
	return ok(ctx, &v1.LogoutResponse{}), nil
}

func (s *AuthService) Me(ctx context.Context, _ *v1.MeRequest) (*v1.MeResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	return ok(ctx, &v1.MeResponse{Data: principalToAPI(principal)}), nil
}

func (s *AuthService) SwitchOrganization(ctx context.Context, request *v1.SwitchOrganizationRequest) (*v1.SwitchOrganizationResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	organizationID, err := uuid.Parse(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrOrganizationForbidden
	}
	next, err := s.usecase.SwitchOrganization(ctx, principal, organizationID)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.SwitchOrganizationResponse{Data: principalToAPI(next)}), nil
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

func (s *AuthService) dingTalkStateCookieName() string {
	return s.policy.CookieName + "_dingtalk_state"
}

func (s *AuthService) dingTalkRegistrationTokenCookieName() string {
	return s.policy.CookieName + "_dingtalk_registration"
}

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
		organizations = append(organizations, &v1.Organization{Id: organization.ID.String(), Code: organization.Code, Name: organization.Name, BaseCurrency: organization.BaseCurrency})
	}
	roleScopes := make([]*v1.RoleScope, 0, len(principal.RoleScopes))
	for _, roleScope := range principal.RoleScopes {
		roleScopes = append(roleScopes, &v1.RoleScope{RoleCode: roleScope.RoleCode, DataScope: string(roleScope.DataScope)})
	}
	return &v1.CurrentUser{Id: principal.UserID.String(), Username: principal.Username, DisplayName: principal.DisplayName, Email: principal.Email, AvatarUrl: principal.AvatarURL, CurrentOrganization: &v1.Organization{Id: principal.Organization.ID.String(), Code: principal.Organization.Code, Name: principal.Organization.Name, BaseCurrency: principal.Organization.BaseCurrency}, Organizations: organizations, Permissions: principal.Permissions, RoleScopes: roleScopes}
}
