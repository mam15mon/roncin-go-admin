package service

import (
	"context"
	"encoding/json"
	nethttp "net/http"
	"strings"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
)

type AuthService struct {
	v1.UnimplementedAuthServiceServer
	usecase       *biz.AuthUsecase
	policy        *biz.SessionPolicy
	passwordReset *biz.PasswordResetUsecase
}

func NewAuthService(usecase *biz.AuthUsecase, policy *biz.SessionPolicy, passwordReset *biz.PasswordResetUsecase) *AuthService {
	return &AuthService{usecase: usecase, policy: policy, passwordReset: passwordReset}
}

func (s *AuthService) Login(ctx context.Context, request *v1.LoginRequest) (*v1.LoginResponse, error) {
	authNoStore(ctx)
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

func (s *AuthService) GetDingTalkLoginConfig(ctx context.Context, _ *v1.GetDingTalkLoginConfigRequest) (*v1.GetDingTalkLoginConfigResponse, error) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		tr.ReplyHeader().Set("Cache-Control", "no-store")
		tr.ReplyHeader().Set("Pragma", "no-cache")
	}
	response, cookie, err := s.prepareDingTalkLoginConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cookie != nil {
		if tr, ok := transport.FromServerContext(ctx); ok {
			tr.ReplyHeader().Add("Set-Cookie", cookie.String())
		}
	}
	return response, nil
}

// DingTalkLoginBootstrap 复用登录配置 DTO，为首屏 HTML 提供独立校验信息及 Cookie。
func (s *AuthService) DingTalkLoginBootstrap(ctx context.Context) (json.RawMessage, *nethttp.Cookie, error) {
	response, cookie, err := s.prepareDingTalkLoginConfig(ctx)
	if err != nil {
		// 显式错误信封保留管理员登录入口，不把内部错误或敏感配置带入 HTML。
		response = &v1.GetDingTalkLoginConfigResponse{Code: 500, Message: "钉钉登录配置加载失败，请稍后重试"}
		payload, marshalErr := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(response)
		return payload, nil, marshalErr
	}
	payload, err := protojson.Marshal(response)
	return payload, cookie, err
}

func (s *AuthService) prepareDingTalkLoginConfig(ctx context.Context) (*v1.GetDingTalkLoginConfigResponse, *nethttp.Cookie, error) {
	enabled, authorizeURL, state, expiresAt, err := s.usecase.StartDingTalkLogin()
	if err != nil {
		return nil, nil, err
	}
	config := &v1.DingTalkLoginConfig{Enabled: enabled}
	var cookie *nethttp.Cookie
	if enabled {
		config.AuthorizeUrl = &authorizeURL
		expires := expiresAt.Format(time.RFC3339)
		config.ExpiresAt = &expires
		cookie = s.newCookie(s.dingTalkStateCookieName(), state, expiresAt, 300)
	}
	return ok(ctx, &v1.GetDingTalkLoginConfigResponse{Data: config}), cookie, nil
}

func (s *AuthService) GetDingTalkEmbeddedConfig(ctx context.Context, _ *v1.GetDingTalkEmbeddedConfigRequest) (*v1.GetDingTalkEmbeddedConfigResponse, error) {
	authNoStore(ctx)
	enabled, corpID, state, expiresAt, err := s.usecase.StartDingTalkEmbeddedLogin()
	if err != nil {
		return nil, err
	}
	config := &v1.DingTalkEmbeddedConfig{Enabled: enabled}
	if enabled {
		config.CorpId = &corpID
		config.State = &state
		expires := expiresAt.Format(time.RFC3339)
		config.ExpiresAt = &expires
		s.setCookieNamed(ctx, s.dingTalkEmbeddedStateCookieName(), state, expiresAt, 300)
	}
	return ok(ctx, &v1.GetDingTalkEmbeddedConfigResponse{Data: config}), nil
}

func (s *AuthService) DingTalkEmbeddedLogin(ctx context.Context, request *v1.DingTalkEmbeddedLoginRequest) (*v1.DingTalkEmbeddedLoginResponse, error) {
	authNoStore(ctx)
	userAgent, expectedState := "", ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		userAgent = tr.RequestHeader().Get("User-Agent")
		expectedState = cookieValue(tr.RequestHeader().Get("Cookie"), s.dingTalkEmbeddedStateCookieName())
	}
	s.setCookieNamed(ctx, s.dingTalkEmbeddedStateCookieName(), "", time.Unix(1, 0), -1)
	result, err := s.usecase.LoginDingTalkEmbedded(ctx, request.GetAuthCode(), request.GetState(), expectedState, userAgent)
	if err != nil {
		return nil, err
	}
	data, err := s.dingTalkLoginData(ctx, result)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.DingTalkEmbeddedLoginResponse{Data: data}), nil
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
	data, err := s.dingTalkLoginData(ctx, result)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.DingTalkLoginResponse{Data: data}), nil
}

func (s *AuthService) dingTalkLoginData(ctx context.Context, result *biz.DingTalkLoginResult) (*v1.DingTalkLoginResult, error) {
	data := &v1.DingTalkLoginResult{Message: result.Message, DisplayName: optionalString(result.DisplayName, result.DisplayName != "")}
	switch result.Status {
	case biz.DingTalkLoginStatusAuthenticated:
		data.Status = v1.DingTalkLoginStatus_DING_TALK_LOGIN_STATUS_AUTHENTICATED
		data.CurrentUser = principalToAPI(result.Principal)
		s.setCookie(ctx, result.SessionToken, result.SessionExpiresAt, 0)
	case biz.DingTalkLoginStatusNotSynced:
		data.Status = v1.DingTalkLoginStatus_DING_TALK_LOGIN_STATUS_NOT_SYNCED
	case biz.DingTalkLoginStatusPendingAssignment:
		data.Status = v1.DingTalkLoginStatus_DING_TALK_LOGIN_STATUS_PENDING_ASSIGNMENT
	case biz.DingTalkLoginStatusDisabled:
		data.Status = v1.DingTalkLoginStatus_DING_TALK_LOGIN_STATUS_DISABLED
	default:
		return nil, biz.ErrDingTalkLoginFailed
	}
	return data, nil
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
		cookie := s.newCookie(name, value, expires, maxAge)
		tr.ReplyHeader().Add("Set-Cookie", cookie.String())
	}
}

func (s *AuthService) newCookie(name, value string, expires time.Time, maxAge int) *nethttp.Cookie {
	return &nethttp.Cookie{Name: name, Value: value, Path: "/", Expires: expires, MaxAge: maxAge, HttpOnly: true, Secure: s.policy.Secure, SameSite: parseSameSite(s.policy.SameSite)}
}

func (s *AuthService) dingTalkEmbeddedStateCookieName() string {
	return s.policy.CookieName + "_dingtalk_embedded_state"
}

func (s *AuthService) dingTalkStateCookieName() string {
	return s.policy.CookieName + "_dingtalk_state"
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
		result = append(result, &v1.OrganizationChoice{OrganizationId: choice.OrganizationID.String(), OrganizationName: choice.OrganizationName, OrganizationCode: choice.OrganizationCode, Kind: authOrganizationKindToAPI(choice.Kind)})
	}
	return result
}

// parseOptionalOrganizationID 解析登录请求可选的所选组织；空串表示未选择，由排序后的有效候选首项建立初始会话。
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
		roleScopes = append(roleScopes, &v1.RoleScope{RoleCode: roleScope.RoleCode, RoleName: roleScope.RoleName})
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

func authNoStore(ctx context.Context) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		tr.ReplyHeader().Set("Cache-Control", "no-store")
		tr.ReplyHeader().Set("Pragma", "no-cache")
	}
}
func (s *AuthService) RequestPasswordResetCode(ctx context.Context, request *v1.RequestPasswordResetCodeRequest) (*v1.RequestPasswordResetCodeResponse, error) {
	authNoStore(ctx)
	result, err := s.passwordReset.RequestCode(ctx, request.GetMobile(), requestmeta.IPAddress(ctx))
	if err != nil {
		return nil, err
	}
	response := ok(ctx, &v1.RequestPasswordResetCodeResponse{Data: &v1.PasswordResetChallenge{ChallengeToken: result.ChallengeToken, ExpiresAt: result.ExpiresAt.Format(time.RFC3339Nano), ResendAvailableAt: result.ResendAvailableAt.Format(time.RFC3339Nano), ServerTime: result.ServerTime.Format(time.RFC3339Nano)}})
	response.Message = "若该手机号对应已开通的钉钉账号，验证码将发送到本人钉钉。"
	return response, nil
}
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, request *v1.ConfirmPasswordResetRequest) (*v1.ConfirmPasswordResetResponse, error) {
	authNoStore(ctx)
	if err := s.passwordReset.Confirm(ctx, request.GetChallengeToken(), request.GetCode(), request.GetNewPassword()); err != nil {
		return nil, err
	}
	s.setCookie(ctx, "", time.Unix(1, 0), -1)
	return ok(ctx, &v1.ConfirmPasswordResetResponse{}), nil
}
