package service

import (
	"context"
	nethttp "net/http"
	"strings"
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
	organizationID, err := parseOptionalOrganizationID(request.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Login(ctx, request.GetUsername(), request.GetPassword(), organizationID, userAgent, requestmeta.IPAddress(ctx))
	if err != nil {
		return nil, err
	}
	s.setCookie(ctx, result.Token, result.ExpiresAt, 0)
	return ok(ctx, &v1.LoginResponse{Data: principalToAPI(result.Principal), OrganizationChoices: organizationChoicesToAPI(result.OrganizationChoices)}), nil
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
		data.RegistrationOrganizations = organizationChoicesToAPI(result.RegistrationOrganizations)
		s.setCookieNamed(ctx, s.dingTalkRegistrationTokenCookieName(), result.RegistrationToken, result.RegistrationExpiresAt, 300)
	default:
		return nil, biz.ErrDingTalkLoginFailed
	}
	return ok(ctx, &v1.DingTalkLoginResponse{Data: data}), nil
}

func (s *AuthService) GetDingTalkInvitationInfo(ctx context.Context, request *v1.GetDingTalkInvitationInfoRequest) (*v1.GetDingTalkInvitationInfoResponse, error) {
	ipAddress := requestmeta.IPAddress(ctx)
	info, err := s.usecase.GetDingTalkInvitationInfo(ctx, request.GetToken(), ipAddress)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetDingTalkInvitationInfoResponse{
		Data: &v1.DingTalkInvitationPublicInfo{
			OrganizationName: info.OrganizationName,
			InviterName:      info.InviterDisplayName,
			ExpiresAt:        info.ExpiresAt.Format(time.RFC3339),
		},
	}), nil
}

func (s *AuthService) RegisterDingTalkUser(ctx context.Context, request *v1.RegisterDingTalkUserRequest) (*v1.RegisterDingTalkUserResponse, error) {
	registrationToken := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		registrationToken = cookieValue(tr.RequestHeader().Get("Cookie"), s.dingTalkRegistrationTokenCookieName())
	}
	requestedOrganizationID, err := parseOptionalOrganizationID(request.GetOrganizationId())
	if err != nil {
		return nil, biz.ErrDingTalkRegistrationOrgInvalid
	}
	registration, err := s.usecase.ConfirmDingTalkRegistration(ctx, registrationToken, request.GetInvitationToken(), requestedOrganizationID)
	if err != nil {
		return nil, err
	}
	s.setCookieNamed(ctx, s.dingTalkRegistrationTokenCookieName(), "", time.Unix(1, 0), -1)
	return ok(ctx, &v1.RegisterDingTalkUserResponse{
		Data: &v1.DingTalkRegistrationConfirmation{DisplayName: registration.DisplayName, Status: registration.Status},
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
		return nil, biz.ErrAuthOrganizationForbidden
	}
	result, err := s.usecase.SwitchOrganization(ctx, principal, organizationID)
	if err != nil {
		return nil, err
	}
	s.setCookie(ctx, result.Token, result.ExpiresAt, 0)
	return ok(ctx, &v1.SwitchOrganizationResponse{Data: principalToAPI(result.Principal), OrganizationChoices: organizationChoicesToAPI(result.OrganizationChoices)}), nil
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

func organizationChoicesToAPI(choices []biz.OrganizationChoice) []*v1.OrganizationChoice {
	result := make([]*v1.OrganizationChoice, 0, len(choices))
	for _, choice := range choices {
		result = append(result, &v1.OrganizationChoice{OrganizationId: choice.OrganizationID.String(), OrganizationName: choice.OrganizationName, OrganizationCode: choice.OrganizationCode, IsDefault: choice.IsDefault, Kind: authOrganizationKindToAPI(choice.Kind)})
	}
	return result
}

// parseOptionalOrganizationID 解析登录请求可选的所选组织；空串表示未选择（沿用默认组织）。
func parseOptionalOrganizationID(raw string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return uuid.Nil, nil
	}
	organizationID, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.Nil, biz.ErrAuthOrganizationInvalid
	}
	return organizationID, nil
}

func principalToAPI(principal *biz.Principal) *v1.CurrentUser {
	organizations := make([]*v1.Organization, 0, len(principal.Organizations))
	for _, organization := range principal.Organizations {
		organizations = append(organizations, authOrganizationToAPI(organization))
	}
	roleScopes := make([]*v1.RoleScope, 0, len(principal.RoleGrants))
	for _, roleScope := range principal.RoleScopes() {
		roleScopes = append(roleScopes, &v1.RoleScope{RoleCode: roleScope.RoleCode, DataScope: string(roleScope.DataScope), RoleName: roleScope.RoleName})
	}
	capabilities := make([]*v1.PermissionCapability, 0)
	for _, capability := range principal.PermissionCapabilities() {
		capabilities = append(capabilities, &v1.PermissionCapability{Key: capability.Key, DataScope: string(capability.DataScope)})
	}
	currentOrganization := principal.Organization
	return &v1.CurrentUser{Id: principal.UserID.String(), Username: principal.Username, DisplayName: principal.DisplayName, Email: principal.Email, AvatarUrl: principal.AvatarURL, CurrentOrganization: authOrganizationToAPI(currentOrganization), Organizations: organizations, Permissions: principal.PermissionKeys(), RoleScopes: roleScopes, PermissionCapabilities: capabilities}
}

// authOrganizationToAPI 投影登录视图组织：kind 供前端推导主数据维护视角（系统管理/公司）。
func authOrganizationToAPI(organization biz.Organization) *v1.Organization {
	return &v1.Organization{Id: organization.ID.String(), Code: organization.Code, Name: organization.Name, BaseCurrency: organization.BaseCurrency, Kind: authOrganizationKindToAPI(organization.Kind)}
}

func authOrganizationKindToAPI(kind biz.OrganizationKind) v1.OrganizationKind {
	switch kind {
	case biz.OrganizationKindSystem:
		return v1.OrganizationKind_ORGANIZATION_KIND_SYSTEM
	case biz.OrganizationKindCompany:
		return v1.OrganizationKind_ORGANIZATION_KIND_COMPANY
	case biz.OrganizationKindDepartment:
		return v1.OrganizationKind_ORGANIZATION_KIND_DEPARTMENT
	case biz.OrganizationKindTeam:
		return v1.OrganizationKind_ORGANIZATION_KIND_TEAM
	default:
		return v1.OrganizationKind_ORGANIZATION_KIND_UNSPECIFIED
	}
}
