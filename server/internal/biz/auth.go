package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

var (
	ErrInvalidCredentials           = errors.Unauthorized("AUTH_INVALID_CREDENTIALS", "用户名或密码错误")
	ErrLoginRateLimited             = errors.New(429, "AUTH_LOGIN_RATE_LIMITED", "登录尝试过于频繁，请稍后再试")
	ErrSessionRequired              = errors.Unauthorized("AUTH_SESSION_REQUIRED", "请先登录")
	ErrSessionExpired               = errors.Unauthorized("AUTH_SESSION_EXPIRED", "登录已过期")
	ErrPermissionDenied             = errors.Forbidden("AUTH_PERMISSION_DENIED", "无权执行此操作")
	ErrOrganizationForbidden        = errors.Forbidden("AUTH_ORGANIZATION_FORBIDDEN", "无权访问该组织")
	ErrWeComDisabled                = errors.ServiceUnavailable("AUTH_WECOM_DISABLED", "企业微信登录未启用")
	ErrWeComLoginFailed             = errors.Unauthorized("AUTH_WECOM_LOGIN_FAILED", "企业微信登录失败")
	ErrWeComCodeInvalid             = errors.Unauthorized("AUTH_WECOM_CODE_INVALID", "企业微信登录凭证已失效，请重新扫码")
	ErrWeComTrustedIPRequired       = errors.Unauthorized("AUTH_WECOM_TRUSTED_IP_REQUIRED", "企业微信拒绝当前服务器 IP，请在应用管理中配置企业可信 IP")
	ErrWeComPermissionDenied        = errors.Unauthorized("AUTH_WECOM_PERMISSION_DENIED", "企业微信应用无权读取成员信息，请检查应用可见范围和通讯录权限")
	ErrWeComStateInvalid            = errors.Unauthorized("AUTH_WECOM_STATE_INVALID", "企业微信登录状态已失效，请重新扫码")
	ErrWeComAuthorizationPending    = errors.Forbidden("AUTH_WECOM_AUTHORIZATION_PENDING", "账号已登记，请联系管理员分配角色并启用账号")
	ErrDingTalkDisabled             = errors.ServiceUnavailable("AUTH_DINGTALK_DISABLED", "钉钉认证未启用")
	ErrDingTalkLoginFailed          = errors.Unauthorized("AUTH_DINGTALK_LOGIN_FAILED", "钉钉登录失败")
	ErrDingTalkCodeInvalid          = errors.Unauthorized("AUTH_DINGTALK_CODE_INVALID", "钉钉登录凭证已失效，请重新扫码")
	ErrDingTalkPermissionDenied     = errors.Unauthorized("AUTH_DINGTALK_PERMISSION_DENIED", "钉钉应用无权读取成员信息，请检查应用权限")
	ErrDingTalkStateInvalid         = errors.Unauthorized("AUTH_DINGTALK_STATE_INVALID", "钉钉验证状态已失效，请重新扫码")
	ErrDingTalkAuthorizationPending = errors.Forbidden("AUTH_DINGTALK_AUTHORIZATION_PENDING", "账号已登记，请联系管理员分配角色并启用账号")
	ErrDingTalkOrganizationMismatch = errors.Forbidden("AUTH_DINGTALK_ORGANIZATION_MISMATCH", "当前钉钉账号不属于本企业，无法继续注册")
	ErrDingTalkNotRegistered        = errors.Unauthorized("AUTH_DINGTALK_NOT_REGISTERED", "当前人员尚未登记")
	ErrDingTalkRegistrationExpired  = errors.Unauthorized("AUTH_DINGTALK_REGISTRATION_EXPIRED", "钉钉身份确认已过期，请重新扫码")
	ErrDingTalkAlreadyRegistered    = errors.Conflict("AUTH_DINGTALK_ALREADY_REGISTERED", "该钉钉账号已完成注册，请直接登录")
)

type DataScope string

const (
	DataScopeAll              DataScope = "all"
	DataScopeOrganization     DataScope = "organization"
	DataScopeOrganizationTree DataScope = "organization_tree"
	DataScopeSelf             DataScope = "self"
)

type Organization struct {
	ID           uuid.UUID
	Code         string
	Name         string
	BaseCurrency string
}

type RoleScope struct {
	RoleCode  string
	DataScope DataScope
}

type Credential struct {
	UserID                uuid.UUID
	Username              string
	DisplayName           string
	Email                 *string
	PasswordHash          *string
	Enabled               bool
	PrimaryOrganizationID uuid.UUID
}

type WeComIdentity struct {
	UserID string
	Name   string
	Email  *string
}

type WeComIdentityProvider interface {
	Enabled() bool
	AuthorizeURL(string) (string, error)
	ResolveIdentity(context.Context, string) (*WeComIdentity, error)
}

type DingTalkIdentity struct {
	UnionID   string
	UserID    string
	CorpID    string
	Name      string
	Email     *string
	AvatarURL *string
}

type DingTalkIdentityProvider interface {
	Enabled() bool
	AuthorizeURL(string) (string, error)
	ResolveIdentity(context.Context, string) (*DingTalkIdentity, error)
}

type DingTalkRegistrationTokenCodec interface {
	Seal(*DingTalkIdentity, time.Time) (string, error)
	Open(string, time.Time) (*DingTalkIdentity, error)
}

type DingTalkLoginStatus string

const (
	DingTalkLoginStatusAuthenticated        DingTalkLoginStatus = "AUTHENTICATED"
	DingTalkLoginStatusRegistrationRequired DingTalkLoginStatus = "REGISTRATION_REQUIRED"
)

type DingTalkLoginResult struct {
	Status                DingTalkLoginStatus
	Principal             *Principal
	SessionToken          string
	SessionExpiresAt      time.Time
	DisplayName           string
	RegistrationToken     string
	RegistrationExpiresAt time.Time
}

type Principal struct {
	SessionTokenHash          string
	UserID                    uuid.UUID
	Username                  string
	DisplayName               string
	Email                     *string
	AvatarURL                 *string
	Organization              Organization
	Organizations             []Organization
	Permissions               []string
	RoleScopes                []RoleScope
	RolePermissions           map[string]map[string]struct{}
	OrderOrganizationAccesses []OrderOrganizationAccess
}

func (p *Principal) HasPermission(key string) bool {
	for _, permission := range p.Permissions {
		if permission == key {
			return true
		}
	}
	return false
}

// HasPermissionInScope checks a permission together with the minimum data
// scope required by the operation. A role with a broader scope satisfies a
// narrower requirement, but a self-scoped role cannot manage organization
// resources.
func (p *Principal) HasPermissionInScope(key string, required DataScope) bool {
	if !p.HasPermission(key) {
		return false
	}
	for _, roleScope := range p.RoleScopes {
		permissions, ok := p.RolePermissions[roleScope.RoleCode]
		if ok && roleScope.DataScope.rank() >= required.rank() {
			if _, hasPermission := permissions[key]; hasPermission {
				return true
			}
		}
	}
	return false
}

func (p *Principal) OrderOrganizationIDs() []uuid.UUID {
	ids := []uuid.UUID{p.Organization.ID}
	seen := map[uuid.UUID]struct{}{p.Organization.ID: {}}
	for _, access := range p.OrderOrganizationAccesses {
		if _, ok := seen[access.OrganizationID]; ok {
			continue
		}
		seen[access.OrganizationID] = struct{}{}
		ids = append(ids, access.OrganizationID)
	}
	return ids
}

func (p *Principal) CanAccessOrderOrganization(organizationID uuid.UUID, writable bool) bool {
	if organizationID == p.Organization.ID {
		return true
	}
	for _, access := range p.OrderOrganizationAccesses {
		if access.OrganizationID == organizationID && (!writable || access.Writable) {
			return true
		}
	}
	return false
}

func (s DataScope) rank() int {
	switch s {
	case DataScopeAll:
		return 4
	case DataScopeOrganizationTree:
		return 3
	case DataScopeOrganization:
		return 2
	case DataScopeSelf:
		return 1
	default:
		return 0
	}
}

type Session struct {
	TokenHash      string
	UserID         uuid.UUID
	OrganizationID uuid.UUID
	ExpiresAt      time.Time
	UserAgent      string
}

type AuditEvent struct {
	OrganizationID *uuid.UUID
	UserID         *uuid.UUID
	Action         string
	ResourceType   string
	ResourceID     string
	Result         string
	RequestID      string
	TraceID        string
	IPAddress      string
	Details        map[string]string
}

type AuditLog struct {
	ID                uuid.UUID
	OrganizationID    *uuid.UUID
	UserID            *uuid.UUID
	Action            string
	ResourceType      *string
	ResourceID        *string
	Result            string
	RequestID         string
	TraceID           string
	IPAddress         string
	Details           map[string]string
	CreatedAt         time.Time
	ActorDisplayName  *string
	TargetDisplayName *string
}

type AuthRepo interface {
	FindCredential(context.Context, string) (*Credential, error)
	LoginRateLimitExceeded(context.Context, []string, time.Time, time.Duration, int) (bool, error)
	RecordLoginFailure(context.Context, []string, time.Time, time.Duration, int, *AuditEvent) (bool, error)
	FindOrCreateWeComCredential(context.Context, *WeComIdentity, *AuditEvent) (*Credential, bool, error)
	FindDingTalkCredential(context.Context, *DingTalkIdentity) (*Credential, error)
	RegisterDingTalkCredential(context.Context, *DingTalkIdentity, *AuditEvent) (*Credential, bool, error)
	ResolvePrincipal(context.Context, uuid.UUID, uuid.UUID) (*Principal, error)
	CreateSession(context.Context, *Session, string, *AuditEvent) error
	FindSession(context.Context, string, time.Time) (*Session, error)
	SwitchSessionOrganization(context.Context, string, uuid.UUID, uuid.UUID, time.Time, *AuditEvent) error
	RevokeSession(context.Context, string, time.Time, *AuditEvent) error
}

type DingTalkRegistration struct {
	DisplayName string
	Status      string
}

type SessionPolicy struct {
	CookieName string
	TTL        time.Duration
	Secure     bool
	SameSite   string
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(*Principal)
	return principal, ok && principal != nil
}

type AuthUsecase struct {
	repo                       AuthRepo
	policy                     *SessionPolicy
	wecom                      WeComIdentityProvider
	dingtalk                   DingTalkIdentityProvider
	dingTalkRegistrationTokens DingTalkRegistrationTokenCodec
}

func NewAuthUsecase(repo AuthRepo, policy *SessionPolicy, wecom WeComIdentityProvider, dingtalk DingTalkIdentityProvider, dingTalkRegistrationTokens DingTalkRegistrationTokenCodec) *AuthUsecase {
	return &AuthUsecase{repo: repo, policy: policy, wecom: wecom, dingtalk: dingtalk, dingTalkRegistrationTokens: dingTalkRegistrationTokens}
}

const (
	loginRateLimitWindow              = time.Minute
	loginRateLimitMaxFailures         = 5
	dingTalkRegistrationTokenLifetime = 5 * time.Minute
)

func (uc *AuthUsecase) Login(ctx context.Context, username, plainPassword, userAgent, ipAddress string) (string, *Principal, time.Time, error) {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	now := time.Now().UTC()
	accountKeyHash, keyHashes := loginRateLimitKeys(normalizedUsername, ipAddress)
	exceeded, err := uc.repo.LoginRateLimitExceeded(ctx, keyHashes, now, loginRateLimitWindow, loginRateLimitMaxFailures)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	if exceeded {
		return "", nil, time.Time{}, ErrLoginRateLimited
	}
	credential, err := uc.repo.FindCredential(ctx, normalizedUsername)
	if err != nil {
		if stderrors.Is(err, ErrInvalidCredentials) {
			return "", nil, time.Time{}, uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}})
		}
		return "", nil, time.Time{}, err
	}
	if credential.PasswordHash == nil {
		return "", nil, time.Time{}, uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{UserID: &credential.UserID, Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}})
	}
	matched, err := password.Verify(plainPassword, *credential.PasswordHash)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("verify password hash: %w", err)
	}
	if !matched {
		return "", nil, time.Time{}, uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{UserID: &credential.UserID, Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}})
	}
	return uc.createSession(ctx, credential, userAgent, "auth.login", accountKeyHash)
}

func (uc *AuthUsecase) recordLoginFailure(ctx context.Context, keyHashes []string, now time.Time, event *AuditEvent) error {
	exceeded, err := uc.repo.RecordLoginFailure(ctx, keyHashes, now, loginRateLimitWindow, loginRateLimitMaxFailures, event)
	if err != nil {
		return err
	}
	if exceeded {
		return ErrLoginRateLimited
	}
	return ErrInvalidCredentials
}

func loginRateLimitKeys(username, ipAddress string) (string, []string) {
	accountKeyHash := hashLoginRateLimitKey("account", username)
	keyHashes := []string{accountKeyHash}
	if normalizedIP := strings.TrimSpace(ipAddress); normalizedIP != "" {
		keyHashes = append(keyHashes, hashLoginRateLimitKey("ip", normalizedIP))
	}
	return accountKeyHash, keyHashes
}

func hashLoginRateLimitKey(kind, value string) string {
	digest := sha256.Sum256([]byte(kind + ":" + value))
	return hex.EncodeToString(digest[:])
}

func (uc *AuthUsecase) StartWeComLogin() (bool, string, string, time.Time, error) {
	if !uc.wecom.Enabled() {
		return false, "", "", time.Time{}, nil
	}
	state, _, err := newSessionToken()
	if err != nil {
		return false, "", "", time.Time{}, err
	}
	authorizeURL, err := uc.wecom.AuthorizeURL(state)
	if err != nil {
		return false, "", "", time.Time{}, err
	}
	return true, authorizeURL, state, time.Now().UTC().Add(5 * time.Minute), nil
}

func (uc *AuthUsecase) LoginWeCom(ctx context.Context, code, state, expectedState, userAgent string) (string, *Principal, time.Time, error) {
	if !uc.wecom.Enabled() {
		return "", nil, time.Time{}, ErrWeComDisabled
	}
	code = strings.TrimSpace(code)
	state = strings.TrimSpace(state)
	expectedState = strings.TrimSpace(expectedState)
	if code == "" || state == "" || expectedState == "" || subtle.ConstantTimeCompare([]byte(state), []byte(expectedState)) != 1 {
		return "", nil, time.Time{}, ErrWeComStateInvalid
	}
	identity, err := uc.wecom.ResolveIdentity(ctx, code)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	credential, _, err := uc.repo.FindOrCreateWeComCredential(ctx, identity, &AuditEvent{Action: "auth.wecom.register", Result: "success"})
	if err != nil {
		return "", nil, time.Time{}, err
	}
	if !credential.Enabled {
		return "", nil, time.Time{}, ErrWeComAuthorizationPending
	}
	return uc.createSession(ctx, credential, userAgent, "auth.wecom.login", "")
}

func (uc *AuthUsecase) StartDingTalkLogin() (bool, string, string, time.Time, error) {
	if !uc.dingtalk.Enabled() {
		return false, "", "", time.Time{}, nil
	}
	state, _, err := newSessionToken()
	if err != nil {
		return false, "", "", time.Time{}, err
	}
	authorizeURL, err := uc.dingtalk.AuthorizeURL(state)
	if err != nil {
		return false, "", "", time.Time{}, err
	}
	return true, authorizeURL, state, time.Now().UTC().Add(5 * time.Minute), nil
}

func (uc *AuthUsecase) LoginDingTalk(ctx context.Context, authCode, state, expectedState, userAgent string) (*DingTalkLoginResult, error) {
	if !uc.dingtalk.Enabled() {
		return nil, ErrDingTalkDisabled
	}
	authCode = strings.TrimSpace(authCode)
	state = strings.TrimSpace(state)
	expectedState = strings.TrimSpace(expectedState)
	if authCode == "" || state == "" || expectedState == "" || subtle.ConstantTimeCompare([]byte(state), []byte(expectedState)) != 1 {
		return nil, ErrDingTalkStateInvalid
	}
	identity, err := uc.dingtalk.ResolveIdentity(ctx, authCode)
	if err != nil {
		return nil, err
	}
	credential, err := uc.repo.FindDingTalkCredential(ctx, identity)
	if err != nil {
		if !stderrors.Is(err, ErrDingTalkNotRegistered) {
			return nil, err
		}
		expiresAt := time.Now().UTC().Add(dingTalkRegistrationTokenLifetime)
		registrationToken, sealErr := uc.dingTalkRegistrationTokens.Seal(identity, expiresAt)
		if sealErr != nil {
			return nil, sealErr
		}
		return &DingTalkLoginResult{
			Status:                DingTalkLoginStatusRegistrationRequired,
			DisplayName:           identity.Name,
			RegistrationToken:     registrationToken,
			RegistrationExpiresAt: expiresAt,
		}, nil
	}
	if !credential.Enabled {
		return nil, ErrDingTalkAuthorizationPending
	}
	token, principal, expiresAt, err := uc.createSession(ctx, credential, userAgent, "auth.dingtalk.login", "")
	if err != nil {
		return nil, err
	}
	return &DingTalkLoginResult{Status: DingTalkLoginStatusAuthenticated, Principal: principal, SessionToken: token, SessionExpiresAt: expiresAt}, nil
}

func (uc *AuthUsecase) ConfirmDingTalkRegistration(ctx context.Context, registrationToken string) (*DingTalkRegistration, error) {
	if !uc.dingtalk.Enabled() {
		return nil, ErrDingTalkDisabled
	}
	identity, err := uc.dingTalkRegistrationTokens.Open(strings.TrimSpace(registrationToken), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	credential, created, err := uc.repo.RegisterDingTalkCredential(ctx, identity, &AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil {
		return nil, err
	}
	if !created && credential.Enabled {
		return nil, ErrDingTalkAlreadyRegistered
	}
	return &DingTalkRegistration{DisplayName: credential.DisplayName, Status: "PENDING"}, nil
}

func (uc *AuthUsecase) createSession(ctx context.Context, credential *Credential, userAgent, auditAction, clearLoginFailureKey string) (string, *Principal, time.Time, error) {
	principal, err := uc.repo.ResolvePrincipal(ctx, credential.UserID, credential.PrimaryOrganizationID)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	rawToken, tokenHash, err := newSessionToken()
	if err != nil {
		return "", nil, time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(uc.policy.TTL)
	if err := uc.repo.CreateSession(ctx, &Session{TokenHash: tokenHash, UserID: credential.UserID, OrganizationID: credential.PrimaryOrganizationID, ExpiresAt: expiresAt, UserAgent: userAgent}, clearLoginFailureKey, &AuditEvent{OrganizationID: &credential.PrimaryOrganizationID, UserID: &credential.UserID, Action: auditAction, Result: "success"}); err != nil {
		return "", nil, time.Time{}, err
	}
	principal.SessionTokenHash = tokenHash
	return rawToken, principal, expiresAt, nil
}

func (uc *AuthUsecase) AuthenticateSession(ctx context.Context, rawToken string) (*Principal, error) {
	if rawToken == "" {
		return nil, ErrSessionRequired
	}
	tokenHash := hashSessionToken(rawToken)
	storedSession, err := uc.repo.FindSession(ctx, tokenHash, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	principal, err := uc.repo.ResolvePrincipal(ctx, storedSession.UserID, storedSession.OrganizationID)
	if err != nil {
		return nil, err
	}
	principal.SessionTokenHash = tokenHash
	return principal, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, principal *Principal) error {
	now := time.Now().UTC()
	return uc.repo.RevokeSession(ctx, principal.SessionTokenHash, now, &AuditEvent{OrganizationID: &principal.Organization.ID, UserID: &principal.UserID, Action: "auth.logout", Result: "success"})
}

func (uc *AuthUsecase) SwitchOrganization(ctx context.Context, principal *Principal, organizationID uuid.UUID) (*Principal, error) {
	now := time.Now().UTC()
	if err := uc.repo.SwitchSessionOrganization(ctx, principal.SessionTokenHash, principal.UserID, organizationID, now, &AuditEvent{OrganizationID: &organizationID, UserID: &principal.UserID, Action: "auth.organization.switch", Result: "success"}); err != nil {
		return nil, err
	}
	next, err := uc.repo.ResolvePrincipal(ctx, principal.UserID, organizationID)
	if err != nil {
		return nil, err
	}
	next.SessionTokenHash = principal.SessionTokenHash
	return next, nil
}

func newSessionToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	rawToken := hex.EncodeToString(raw)
	return rawToken, hashSessionToken(rawToken), nil
}

func hashSessionToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}
