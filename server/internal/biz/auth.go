package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials    = errors.Unauthorized("AUTH_INVALID_CREDENTIALS", "用户名或密码错误")
	ErrSessionRequired       = errors.Unauthorized("AUTH_SESSION_REQUIRED", "请先登录")
	ErrSessionExpired        = errors.Unauthorized("AUTH_SESSION_EXPIRED", "登录已过期")
	ErrPermissionDenied      = errors.Forbidden("AUTH_PERMISSION_DENIED", "无权执行此操作")
	ErrOrganizationForbidden = errors.Forbidden("AUTH_ORGANIZATION_FORBIDDEN", "无权访问该组织")
)

type DataScope string

const (
	DataScopeAll              DataScope = "all"
	DataScopeOrganization     DataScope = "organization"
	DataScopeOrganizationTree DataScope = "organization_tree"
	DataScopeSelf             DataScope = "self"
)

type Organization struct {
	ID   uuid.UUID
	Code string
	Name string
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
	PasswordHash          string
	PrimaryOrganizationID uuid.UUID
}

type Principal struct {
	SessionTokenHash string
	UserID           uuid.UUID
	Username         string
	DisplayName      string
	Email            *string
	Organization     Organization
	Organizations    []Organization
	Permissions      []string
	RoleScopes       []RoleScope
	RolePermissions  map[string]map[string]struct{}
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
	Result         string
	RequestID      string
	TraceID        string
	IPAddress      string
	Details        map[string]string
}

type AuditLog struct {
	ID             uuid.UUID
	OrganizationID *uuid.UUID
	UserID         *uuid.UUID
	Action         string
	ResourceType   *string
	ResourceID     *string
	Result         string
	RequestID      string
	TraceID        string
	IPAddress      string
	Details        map[string]string
	CreatedAt      time.Time
}

type AuthRepo interface {
	FindCredential(context.Context, string) (*Credential, error)
	ResolvePrincipal(context.Context, uuid.UUID, uuid.UUID) (*Principal, error)
	CreateSession(context.Context, *Session) error
	FindSession(context.Context, string, time.Time) (*Session, error)
	SwitchSessionOrganization(context.Context, string, uuid.UUID, uuid.UUID, time.Time) error
	RevokeSession(context.Context, string, time.Time) error
	WriteAudit(context.Context, *AuditEvent) error
}

type AuditRepo interface {
	WriteAudit(context.Context, *AuditEvent) error
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
	repo   AuthRepo
	policy *SessionPolicy
}

func NewAuthUsecase(repo AuthRepo, policy *SessionPolicy) *AuthUsecase {
	return &AuthUsecase{repo: repo, policy: policy}
}

func (uc *AuthUsecase) Login(ctx context.Context, username, plainPassword, userAgent string) (string, *Principal, time.Time, error) {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	credential, err := uc.repo.FindCredential(ctx, normalizedUsername)
	if err != nil {
		if stderrors.Is(err, ErrInvalidCredentials) {
			if auditErr := uc.repo.WriteAudit(ctx, &AuditEvent{Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}}); auditErr != nil {
				return "", nil, time.Time{}, auditErr
			}
		}
		return "", nil, time.Time{}, err
	}
	matched, err := password.Verify(plainPassword, credential.PasswordHash)
	if err != nil {
		return "", nil, time.Time{}, fmt.Errorf("verify password hash: %w", err)
	}
	if !matched {
		if err := uc.repo.WriteAudit(ctx, &AuditEvent{UserID: &credential.UserID, Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}}); err != nil {
			return "", nil, time.Time{}, err
		}
		return "", nil, time.Time{}, ErrInvalidCredentials
	}
	principal, err := uc.repo.ResolvePrincipal(ctx, credential.UserID, credential.PrimaryOrganizationID)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	rawToken, tokenHash, err := newSessionToken()
	if err != nil {
		return "", nil, time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(uc.policy.TTL)
	if err := uc.repo.CreateSession(ctx, &Session{TokenHash: tokenHash, UserID: credential.UserID, OrganizationID: credential.PrimaryOrganizationID, ExpiresAt: expiresAt, UserAgent: userAgent}); err != nil {
		return "", nil, time.Time{}, err
	}
	principal.SessionTokenHash = tokenHash
	if err := uc.repo.WriteAudit(ctx, &AuditEvent{OrganizationID: &credential.PrimaryOrganizationID, UserID: &credential.UserID, Action: "auth.login", Result: "success"}); err != nil {
		return "", nil, time.Time{}, err
	}
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
	if err := uc.repo.RevokeSession(ctx, principal.SessionTokenHash, now); err != nil {
		return err
	}
	return uc.repo.WriteAudit(ctx, &AuditEvent{OrganizationID: &principal.Organization.ID, UserID: &principal.UserID, Action: "auth.logout", Result: "success"})
}

func (uc *AuthUsecase) SwitchOrganization(ctx context.Context, principal *Principal, organizationID uuid.UUID) (*Principal, error) {
	now := time.Now().UTC()
	if err := uc.repo.SwitchSessionOrganization(ctx, principal.SessionTokenHash, principal.UserID, organizationID, now); err != nil {
		return nil, err
	}
	next, err := uc.repo.ResolvePrincipal(ctx, principal.UserID, organizationID)
	if err != nil {
		return nil, err
	}
	next.SessionTokenHash = principal.SessionTokenHash
	if err := uc.repo.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &principal.UserID, Action: "auth.organization.switch", Result: "success"}); err != nil {
		return nil, err
	}
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
