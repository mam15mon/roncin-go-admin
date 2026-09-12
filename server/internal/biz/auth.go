package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"log/slog"
	"sort"
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
	ErrAuthOrganizationInvalid      = errors.BadRequest("AUTH_ORGANIZATION_INVALID", "所选组织无效或不可用")
	ErrAuthOrganizationForbidden    = errors.Forbidden("AUTH_ORGANIZATION_FORBIDDEN", "无该组织的成员资格或已停用")
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

// OrganizationChoice 是登录组织选择与应用内切换器共用的「本人启用中成员资格组织」候选视图。
// IsDefault 对应成员资格的 primary 标志。
type OrganizationChoice struct {
	OrganizationID   uuid.UUID
	OrganizationName string
	OrganizationCode string
	IsDefault        bool
}

type RoleScope struct {
	RoleCode  string
	DataScope DataScope
}

// RoleGrant 保留角色、权限、数据范围和追加组织访问项之间的来源关系。
// 组织范围必须按该角色授权的具体权限解析，不能先分别合并再交叉匹配。
type RoleGrant struct {
	RoleID               uuid.UUID
	RoleCode             string
	Permissions          map[string]struct{}
	DataScope            DataScope
	OrganizationAccesses []OrganizationAccess
}

// OrganizationScopeNode 是权限组织范围解析所需的组织树最小投影。
// 停用节点保留其父子关系，以便组织树范围仍能识别其下的启用后代；停用节点自身不会进入范围。
type OrganizationScopeNode struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
	Disabled bool
}

// PermissionOrganizationScope 是某一具体权限可读取和可写入的组织集合。
type PermissionOrganizationScope struct {
	ReadableOrganizationIDs []uuid.UUID
	WritableOrganizationIDs []uuid.UUID
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

// DingTalkDirectoryLookup 按企业通讯录解析成员手机号（匹配邀请的唯一依据）。
// 实现方接口不可用或无权限时返回错误；调用方必须降级到人工审批通道，
// 不得让钉钉通讯录故障阻断扫码登录。
type DingTalkDirectoryLookup interface {
	LookupMobileByUserID(ctx context.Context, dingTalkUserID string) (string, error)
}

// DingTalkLoginRegistrationRepo 是登录/注册流程所需的钉钉注册仓储子集：
// 通道 A 邀请匹配消费、注册目标组织解析与审批通知收件人路由。
type DingTalkLoginRegistrationRepo interface {
	// FindActiveInvitationByMobile 返回该手机号最早创建的未过期活跃定向邀请
	// （同一手机号可在多个组织各持一条活跃邀请，先建先得）。
	FindActiveInvitationByMobile(ctx context.Context, mobile string) (*DingTalkInvitation, error)
	// FindInvitationByToken 通过 128-bit Token 查找专属邀请（通用/定向）。
	FindInvitationByToken(ctx context.Context, token string) (*DingTalkInvitation, error)
	// ConsumeInvitationAndActivate 在单事务内消费邀请并完成激活原语：
	// 创建启用账号 + 目标组织成员资格 + 初始角色 + 通知本人与邀请人 + 审计。
	// 邀请已被并发消费或不再可用时返回业务错误，调用方降级通道 B。
	ConsumeInvitationAndActivate(ctx context.Context, identity *DingTalkIdentity, invitationID uuid.UUID) (*Credential, error)
	// ListRegistrationOrganizations 返回注册确认页可选的启用中公司组织。
	ListRegistrationOrganizations(ctx context.Context) ([]OrganizationChoice, error)
	// FindRegistrationOrganization 校验注册自选的目标组织（启用中的公司）。
	FindRegistrationOrganization(ctx context.Context, organizationID uuid.UUID) (*Organization, error)
	// FindHeadquartersOrganizationID 返回总部收口根组织（通道 B 未自选时的兜底路由）。
	FindHeadquartersOrganizationID(ctx context.Context) (uuid.UUID, error)
	// ListApproverRecipients 返回目标组织内持有钉钉邀请管理权限的启用中用户
	// （按最近活跃截断），作为注册审批通知的收件人。
	ListApproverRecipients(ctx context.Context, organizationID uuid.UUID) ([]*DingTalkApproverRecipient, error)
	// GetParentOrganizationID 查询组织的父级组织 ID。
	GetParentOrganizationID(ctx context.Context, orgID uuid.UUID) (*uuid.UUID, bool, error)
}

// ListApproverRecipientsWithEscalation 沿 parent_id 逐级向上追溯首个有候选审批人的
// 祖先节点（直至总部根节点），根链均无审批人时兜底总部；属于通知兜底而非权限变更。
// 追溯决策由 biz 组合仓储原语完成；一键转派事务内的 FOR SHARE 变体保留在 data 层。
func ListApproverRecipientsWithEscalation(ctx context.Context, repo DingTalkLoginRegistrationRepo, targetOrgID uuid.UUID) ([]*DingTalkApproverRecipient, uuid.UUID, bool, error) {
	if targetOrgID == uuid.Nil {
		return nil, uuid.Nil, false, ErrAdminInvalidArgument
	}
	currID := targetOrgID
	isEscalated := false
	visited := make(map[uuid.UUID]bool)
	for depth := 0; depth < 20; depth++ {
		if visited[currID] {
			break
		}
		visited[currID] = true

		recipients, err := repo.ListApproverRecipients(ctx, currID)
		if err != nil {
			return nil, uuid.Nil, false, err
		}
		if len(recipients) > 0 {
			return recipients, currID, isEscalated, nil
		}
		parentID, hasParent, err := repo.GetParentOrganizationID(ctx, currID)
		if err != nil {
			return nil, uuid.Nil, false, err
		}
		if !hasParent || parentID == nil {
			// 已达根组织，若当前不是总部且总部存在，尝试兜底总部
			hqID, hqErr := repo.FindHeadquartersOrganizationID(ctx)
			if hqErr == nil && hqID != currID {
				hqRecipients, hqQueryErr := repo.ListApproverRecipients(ctx, hqID)
				if hqQueryErr != nil {
					return nil, uuid.Nil, false, hqQueryErr
				}
				if len(hqRecipients) > 0 {
					return hqRecipients, hqID, true, nil
				}
			}
			return nil, currID, isEscalated, nil
		}
		currID = *parentID
		isEscalated = true
	}
	return nil, currID, isEscalated, nil
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
	// RegistrationOrganizations 在 REGISTRATION_REQUIRED 时携带可自选的目标公司列表。
	RegistrationOrganizations []OrganizationChoice
}

type Principal struct {
	SessionTokenHash  string
	UserID            uuid.UUID
	Username          string
	DisplayName       string
	Email             *string
	AvatarURL         *string
	IsBootstrapAdmin  bool
	Organization      Organization
	Organizations     []Organization
	RoleGrants        []RoleGrant
	OrganizationNodes []OrganizationScopeNode
}

func (p *Principal) HasPermission(key string) bool {
	for _, grant := range p.RoleGrants {
		if _, ok := grant.Permissions[key]; ok {
			return true
		}
	}
	return false
}

// PermissionKeys 从角色授权真相源投影登录响应所需的权限集合。
func (p *Principal) PermissionKeys() []string {
	permissions := make(map[string]struct{})
	for _, grant := range p.RoleGrants {
		for permission := range grant.Permissions {
			permissions[permission] = struct{}{}
		}
	}
	result := make([]string, 0, len(permissions))
	for permission := range permissions {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result
}

// HasPermissionInScope checks a permission together with the minimum data
// scope required by the operation. A role with a broader scope satisfies a
// narrower requirement, but a self-scoped role cannot manage organization
// resources.
func (p *Principal) HasPermissionInScope(key string, required DataScope) bool {
	for _, grant := range p.RoleGrants {
		if _, hasPermission := grant.Permissions[key]; !hasPermission {
			continue
		}
		if p.IsBootstrapAdmin || grant.DataScope.rank() >= required.rank() {
			return true
		}
	}
	return false
}

// RoleScopes 从角色授权真相源投影登录响应所需的角色范围，避免在 Principal 中保存第二份状态。
func (p *Principal) RoleScopes() []RoleScope {
	result := make([]RoleScope, 0, len(p.RoleGrants))
	for _, grant := range p.RoleGrants {
		result = append(result, RoleScope{RoleCode: grant.RoleCode, DataScope: grant.DataScope})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].RoleCode == result[j].RoleCode {
			return result[i].DataScope < result[j].DataScope
		}
		return result[i].RoleCode < result[j].RoleCode
	})
	return result
}

// ResolvePermissionOrganizationScope 只合并持有目标权限的角色范围。
// 没有匹配角色时返回统一权限错误，调用方不得回退为当前组织或其他角色的范围。
func (p *Principal) ResolvePermissionOrganizationScope(permission string) (PermissionOrganizationScope, error) {
	matchingGrants := make([]RoleGrant, 0, len(p.RoleGrants))
	for _, grant := range p.RoleGrants {
		if _, ok := grant.Permissions[permission]; ok {
			matchingGrants = append(matchingGrants, grant)
		}
	}
	if len(matchingGrants) == 0 {
		return PermissionOrganizationScope{}, ErrPermissionDenied
	}

	nodes := p.organizationScopeNodes()
	if p.IsBootstrapAdmin {
		enabledIDs := enabledOrganizationIDs(nodes)
		return permissionOrganizationScopeFromSets(enabledIDs, enabledIDs), nil
	}

	readable := make(map[uuid.UUID]struct{})
	writable := make(map[uuid.UUID]struct{})
	for _, grant := range matchingGrants {
		for _, organizationID := range p.baseOrganizationIDs(grant.DataScope, nodes) {
			readable[organizationID] = struct{}{}
			writable[organizationID] = struct{}{}
		}
		for _, access := range grant.OrganizationAccesses {
			node, exists := nodes[access.OrganizationID]
			if !exists || node.Disabled {
				continue
			}
			readable[access.OrganizationID] = struct{}{}
			if access.Writable {
				writable[access.OrganizationID] = struct{}{}
			}
		}
	}
	return permissionOrganizationScopeFromSets(readable, writable), nil
}

// CanAccessOrganizationForPermission 按单个权限的组织范围检查目标组织。
func (p *Principal) CanAccessOrganizationForPermission(permission string, organizationID uuid.UUID, writable bool) bool {
	scope, err := p.ResolvePermissionOrganizationScope(permission)
	if err != nil {
		return false
	}
	ids := scope.ReadableOrganizationIDs
	if writable {
		ids = scope.WritableOrganizationIDs
	}
	return containsOrganizationID(ids, organizationID)
}

func (p *Principal) organizationScopeNodes() map[uuid.UUID]OrganizationScopeNode {
	nodes := make(map[uuid.UUID]OrganizationScopeNode, len(p.OrganizationNodes))
	for _, node := range p.OrganizationNodes {
		nodes[node.ID] = node
	}
	if len(nodes) == 0 && p.Organization.ID != uuid.Nil {
		nodes[p.Organization.ID] = OrganizationScopeNode{ID: p.Organization.ID}
	}
	return nodes
}

func enabledOrganizationIDs(nodes map[uuid.UUID]OrganizationScopeNode) map[uuid.UUID]struct{} {
	ids := make(map[uuid.UUID]struct{}, len(nodes))
	for organizationID, node := range nodes {
		if !node.Disabled {
			ids[organizationID] = struct{}{}
		}
	}
	return ids
}

func (p *Principal) baseOrganizationIDs(scope DataScope, nodes map[uuid.UUID]OrganizationScopeNode) []uuid.UUID {
	current, currentExists := nodes[p.Organization.ID]
	if !currentExists || current.Disabled {
		return nil
	}
	switch scope {
	case DataScopeAll:
		return sortedOrganizationIDs(enabledOrganizationIDs(nodes))
	case DataScopeOrganization, DataScopeSelf:
		return []uuid.UUID{p.Organization.ID}
	case DataScopeOrganizationTree:
		result := make(map[uuid.UUID]struct{})
		for organizationID, node := range nodes {
			if !node.Disabled && (organizationID == p.Organization.ID || isOrganizationDescendant(organizationID, p.Organization.ID, nodes)) {
				result[organizationID] = struct{}{}
			}
		}
		return sortedOrganizationIDs(result)
	default:
		return nil
	}
}

func isOrganizationDescendant(organizationID, ancestorID uuid.UUID, nodes map[uuid.UUID]OrganizationScopeNode) bool {
	seen := make(map[uuid.UUID]struct{})
	for currentID := organizationID; currentID != uuid.Nil; {
		if currentID == ancestorID {
			return true
		}
		if _, visited := seen[currentID]; visited {
			return false
		}
		seen[currentID] = struct{}{}
		node, ok := nodes[currentID]
		if !ok || node.ParentID == nil {
			return false
		}
		currentID = *node.ParentID
	}
	return false
}

func permissionOrganizationScopeFromSets[T any](readable, writable map[uuid.UUID]T) PermissionOrganizationScope {
	return PermissionOrganizationScope{
		ReadableOrganizationIDs: sortedOrganizationIDs(readable),
		WritableOrganizationIDs: sortedOrganizationIDs(writable),
	}
}

func sortedOrganizationIDs[T any](ids map[uuid.UUID]T) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))
	for organizationID := range ids {
		result = append(result, organizationID)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func containsOrganizationID(ids []uuid.UUID, organizationID uuid.UUID) bool {
	for _, id := range ids {
		if id == organizationID {
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

// AuthSessionResult 是登录与切换组织同构的会话建立结果：新令牌、随组织重算的主体与候选组织列表。
type AuthSessionResult struct {
	Token               string
	Principal           *Principal
	OrganizationChoices []OrganizationChoice
	ExpiresAt           time.Time
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
	// RegisterDingTalkCredential 注册钉钉账号（PENDING 禁用，收口总部成员资格）；
	// requestedOrganizationID 为通道 B 自选目标组织（可空），notice 为注册审批
	// 通知的路由决策（收件人与展示组织名，含代管标注），仓储在注册同事务内
	// 入队（任务幂等键确定性去重）。
	RegisterDingTalkCredential(context.Context, *DingTalkIdentity, *uuid.UUID, *DingTalkApproverNotice, *AuditEvent) (*Credential, bool, error)
	ListEnabledMembershipOrganizations(context.Context, uuid.UUID) ([]OrganizationChoice, error)
	ResolvePrincipal(context.Context, uuid.UUID, uuid.UUID) (*Principal, error)
	CreateSession(context.Context, *Session, string, *AuditEvent) error
	FindSession(context.Context, string, time.Time) (*Session, error)
	RotateSession(context.Context, string, *Session, time.Time, *AuditEvent) error
	RevokeSession(context.Context, string, time.Time, *AuditEvent) error
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

// RequirePrincipal 返回当前登录主体；上下文未携带有效主体时返回统一会话错误。
func RequirePrincipal(ctx context.Context) (*Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return nil, ErrSessionRequired
	}
	return principal, nil
}

type AuthUsecase struct {
	repo                       AuthRepo
	policy                     *SessionPolicy
	wecom                      WeComIdentityProvider
	dingtalk                   DingTalkIdentityProvider
	dingTalkRegistrationTokens DingTalkRegistrationTokenCodec
	dingTalkRegistrations      DingTalkLoginRegistrationRepo
	dingTalkDirectory          DingTalkDirectoryLookup
	logger                     *slog.Logger
}

func NewAuthUsecase(repo AuthRepo, policy *SessionPolicy, wecom WeComIdentityProvider, dingtalk DingTalkIdentityProvider, dingTalkRegistrationTokens DingTalkRegistrationTokenCodec, dingTalkRegistrations DingTalkLoginRegistrationRepo, dingTalkDirectory DingTalkDirectoryLookup, logger *slog.Logger) *AuthUsecase {
	return &AuthUsecase{repo: repo, policy: policy, wecom: wecom, dingtalk: dingtalk, dingTalkRegistrationTokens: dingTalkRegistrationTokens, dingTalkRegistrations: dingTalkRegistrations, dingTalkDirectory: dingTalkDirectory, logger: logger}
}

const (
	loginRateLimitWindow              = time.Minute
	loginRateLimitMaxFailures         = 5
	dingTalkRegistrationTokenLifetime = 5 * time.Minute
)

// Login 校验账号口令后建立会话；requestedOrganizationID 非空时必须是本人启用中成员资格组织，
// 否则返回参数错误（不静默回退默认组织）。未指定时沿用默认（primary）组织。
func (uc *AuthUsecase) Login(ctx context.Context, username, plainPassword string, requestedOrganizationID uuid.UUID, userAgent, ipAddress string) (*AuthSessionResult, error) {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	now := time.Now().UTC()
	accountKeyHash, keyHashes := loginRateLimitKeys(normalizedUsername, ipAddress)
	exceeded, err := uc.repo.LoginRateLimitExceeded(ctx, keyHashes, now, loginRateLimitWindow, loginRateLimitMaxFailures)
	if err != nil {
		return nil, err
	}
	if exceeded {
		return nil, ErrLoginRateLimited
	}
	credential, err := uc.repo.FindCredential(ctx, normalizedUsername)
	if err != nil {
		if stderrors.Is(err, ErrInvalidCredentials) {
			return nil, uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}})
		}
		return nil, err
	}
	if credential.PasswordHash == nil {
		return nil, uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{UserID: &credential.UserID, Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}})
	}
	matched, err := password.Verify(plainPassword, *credential.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password hash: %w", err)
	}
	if !matched {
		return nil, uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{UserID: &credential.UserID, Action: "auth.login", Result: "failure", Details: map[string]string{"username": normalizedUsername}})
	}
	organizationID := credential.PrimaryOrganizationID
	choices, err := uc.listEnabledMembershipOrganizations(ctx, credential.UserID)
	if err != nil {
		return nil, err
	}
	if requestedOrganizationID != uuid.Nil {
		if !organizationChoicesContain(choices, requestedOrganizationID) {
			return nil, ErrAuthOrganizationInvalid
		}
		organizationID = requestedOrganizationID
	}
	return uc.createSession(ctx, credential, organizationID, choices, userAgent, "auth.login", accountKeyHash)
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
	result, err := uc.createSession(ctx, credential, credential.PrimaryOrganizationID, nil, userAgent, "auth.wecom.login", "")
	if err != nil {
		return "", nil, time.Time{}, err
	}
	return result.Token, result.Principal, result.ExpiresAt, nil
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
		// 通道 A：未注册时先按企业通讯录手机号匹配活跃邀请，命中即自动激活
		// 并直接建立会话；任何一步失败都降级通道 B，绝不阻断扫码登录。
		if activated := uc.activateDingTalkInvitation(ctx, identity); activated != nil {
			result, sessionErr := uc.createSession(ctx, activated, activated.PrimaryOrganizationID, nil, userAgent, "auth.dingtalk.login", "")
			if sessionErr != nil {
				return nil, sessionErr
			}
			return &DingTalkLoginResult{Status: DingTalkLoginStatusAuthenticated, Principal: result.Principal, SessionToken: result.Token, SessionExpiresAt: result.ExpiresAt}, nil
		}
		return uc.requireDingTalkRegistration(ctx, identity)
	}
	if !credential.Enabled {
		return nil, ErrDingTalkAuthorizationPending
	}
	result, err := uc.createSession(ctx, credential, credential.PrimaryOrganizationID, nil, userAgent, "auth.dingtalk.login", "")
	if err != nil {
		return nil, err
	}
	return &DingTalkLoginResult{Status: DingTalkLoginStatusAuthenticated, Principal: result.Principal, SessionToken: result.Token, SessionExpiresAt: result.ExpiresAt}, nil
}

// activateDingTalkInvitation 执行通道 A 匹配链：企业通讯录按 userId 取手机号 →
// 归一化 → 等值匹配活跃邀请 → 单事务消费并激活。匹配不到、钉钉接口失败或消费
// 冲突（含并发与邀请失效）一律返回 nil 降级通道 B；失败只记日志不外抛。
func (uc *AuthUsecase) activateDingTalkInvitation(ctx context.Context, identity *DingTalkIdentity) *Credential {
	if uc.dingTalkDirectory == nil || uc.dingTalkRegistrations == nil {
		return nil
	}
	mobile, err := uc.dingTalkDirectory.LookupMobileByUserID(ctx, identity.UserID)
	if err != nil {
		uc.logDingTalkInvitationDegraded(identity, "手机号查询失败", err)
		return nil
	}
	normalized := NormalizeDingTalkMobile(mobile)
	if !ValidDingTalkMobile(normalized) {
		uc.logDingTalkInvitationDegraded(identity, "手机号无法归一化", nil)
		return nil
	}
	invitation, err := uc.dingTalkRegistrations.FindActiveInvitationByMobile(ctx, normalized)
	if err != nil {
		uc.logDingTalkInvitationDegraded(identity, "无匹配邀请", nil)
		return nil
	}
	credential, err := uc.dingTalkRegistrations.ConsumeInvitationAndActivate(ctx, identity, invitation.ID)
	if err != nil {
		uc.logDingTalkInvitationDegraded(identity, "邀请消费失败", err)
		return nil
	}
	return credential
}

func (uc *AuthUsecase) logDingTalkInvitationDegraded(identity *DingTalkIdentity, reason string, cause error) {
	if uc.logger == nil {
		return
	}
	attributes := []any{slog.String("reason", reason), slog.String("dingtalk_unionid", identity.UnionID)}
	if cause != nil {
		// 只记录顶层错误消息，不透出可能携带手机号的错误细节。
		attributes = append(attributes, slog.String("error", cause.Error()))
	}
	uc.logger.Warn("钉钉邀请匹配降级为人工审批通道", attributes...)
}

// requireDingTalkRegistration 构造通道 B 的注册要求：签发注册令牌并携带可自选
// 的目标公司列表（启用中的公司组织，名称 + ID 稳定排序）。
func (uc *AuthUsecase) requireDingTalkRegistration(ctx context.Context, identity *DingTalkIdentity) (*DingTalkLoginResult, error) {
	var organizations []OrganizationChoice
	if uc.dingTalkRegistrations != nil {
		choices, err := uc.dingTalkRegistrations.ListRegistrationOrganizations(ctx)
		if err != nil {
			return nil, err
		}
		organizations = choices
	}
	expiresAt := time.Now().UTC().Add(dingTalkRegistrationTokenLifetime)
	registrationToken, sealErr := uc.dingTalkRegistrationTokens.Seal(identity, expiresAt)
	if sealErr != nil {
		return nil, sealErr
	}
	return &DingTalkLoginResult{
		Status:                    DingTalkLoginStatusRegistrationRequired,
		DisplayName:               identity.Name,
		RegistrationToken:         registrationToken,
		RegistrationExpiresAt:     expiresAt,
		RegistrationOrganizations: organizations,
	}, nil
}

// DingTalkRegistrationResult 是注册确认的对外结果：账号以 PENDING 状态落库，
// 等待目标组织管理员审批（或邀请自动激活）。
type DingTalkRegistrationResult struct {
	DisplayName string
	Status      string
}

// GetDingTalkInvitationInfo 扫码落地页查询专属邀请公开信息（未登录极简接口）：
// 1. IP 限流保护（复用登录限流口径）；
// 2. 仅返回组织名、邀请人与有效期，绝不透出 Token、预设角色或手机号；
// 3. 失效或不存在记录失败审计并返回对应错误。
func (uc *AuthUsecase) GetDingTalkInvitationInfo(ctx context.Context, token, ipAddress string) (*DingTalkInvitationPublicInfo, error) {
	now := time.Now().UTC()
	var keyHashes []string
	if normalizedIP := strings.TrimSpace(ipAddress); normalizedIP != "" {
		keyHashes = append(keyHashes, hashLoginRateLimitKey("ip", normalizedIP))
	}
	if len(keyHashes) > 0 {
		exceeded, err := uc.repo.LoginRateLimitExceeded(ctx, keyHashes, now, loginRateLimitWindow, loginRateLimitMaxFailures)
		if err != nil {
			return nil, err
		}
		if exceeded {
			return nil, ErrLoginRateLimited
		}
	}

	token = strings.TrimSpace(token)
	if token == "" || uc.dingTalkRegistrations == nil {
		if len(keyHashes) > 0 {
			_ = uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{Action: "auth.dingtalk.invitation.info", Result: "failure"})
		}
		return nil, ErrDingTalkInvitationNotFound
	}
	invitation, err := uc.dingTalkRegistrations.FindInvitationByToken(ctx, token)
	if err != nil {
		if len(keyHashes) > 0 {
			_ = uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{Action: "auth.dingtalk.invitation.info", Result: "failure"})
		}
		return nil, ErrDingTalkInvitationNotFound
	}
	if invitation.Status == DingTalkInvitationStatusConsumed {
		if len(keyHashes) > 0 {
			_ = uc.recordLoginFailure(ctx, keyHashes, now, &AuditEvent{Action: "auth.dingtalk.invitation.info", Result: "failure"})
		}
		return nil, ErrDingTalkInvitationNotFound
	}
	if invitation.Status == DingTalkInvitationStatusRevoked {
		return nil, ErrDingTalkInvitationRevoked
	}
	if invitation.EffectiveStatus(now) == DingTalkInvitationStatusExpired {
		return nil, ErrDingTalkInvitationExpired
	}
	return &DingTalkInvitationPublicInfo{
		OrganizationName:   invitation.OrganizationName,
		InviterDisplayName: invitation.InviterName,
		ExpiresAt:          invitation.ExpiresAt,
	}, nil
}

// ConfirmDingTalkRegistration 落库注册（PENDING 禁用 + 总部收口成员资格），
// 支持携带专属邀请 Token（服务端绑定组织，杜绝伪造）或自选目标公司；
// 注册同事务内向目标组织（空 = 总部收口）持邀请管理权限的启用中用户入队审批通知，
// 若目标组织无持权管理员则沿组织树逐级向上追溯兜底。
func (uc *AuthUsecase) ConfirmDingTalkRegistration(ctx context.Context, registrationToken, invitationToken string, requestedOrganizationID uuid.UUID) (*DingTalkRegistrationResult, error) {
	if !uc.dingtalk.Enabled() {
		return nil, ErrDingTalkDisabled
	}
	identity, err := uc.dingTalkRegistrationTokens.Open(strings.TrimSpace(registrationToken), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	var requestedOrganization *uuid.UUID
	var notice *DingTalkApproverNotice
	// notificationOrgName 是通知卡片展示的目标组织名：Token 路径取邀请锁定组织、
	// 自选路径取所选组织；未自选（总部收口）时留空，由仓储按总部名展示。
	notificationOrgName := ""

	invitationToken = strings.TrimSpace(invitationToken)
	if invitationToken != "" {
		if uc.dingTalkRegistrations == nil {
			return nil, ErrDingTalkRegistrationOrgInvalid
		}
		invitation, invErr := uc.dingTalkRegistrations.FindInvitationByToken(ctx, invitationToken)
		if invErr != nil {
			return nil, ErrDingTalkInvitationNotFound
		}
		if invitation.Status == DingTalkInvitationStatusConsumed {
			return nil, ErrDingTalkInvitationNotFound
		}
		if invitation.Status == DingTalkInvitationStatusRevoked {
			return nil, ErrDingTalkInvitationRevoked
		}
		if invitation.EffectiveStatus(time.Now().UTC()) == DingTalkInvitationStatusExpired {
			return nil, ErrDingTalkInvitationExpired
		}
		// 客户端零信任：专属码强绑定目标组织，忽略传入的 requestedOrganizationID，杜绝篡改伪造
		targetOrgID := invitation.OrganizationID
		requestedOrganization = &targetOrgID
		requestedOrganizationID = targetOrgID
		notificationOrgName = invitation.OrganizationName
	} else if requestedOrganizationID != uuid.Nil {
		if uc.dingTalkRegistrations == nil {
			return nil, ErrDingTalkRegistrationOrgInvalid
		}
		organization, orgErr := uc.dingTalkRegistrations.FindRegistrationOrganization(ctx, requestedOrganizationID)
		if orgErr != nil {
			return nil, ErrDingTalkRegistrationOrgInvalid
		}
		requestedOrganization = &organization.ID
		notificationOrgName = organization.Name
	}

	if uc.dingTalkRegistrations != nil {
		routingOrganizationID := requestedOrganizationID
		if routingOrganizationID == uuid.Nil {
			// 未自选目标组织：总部兜底，通知总部收口组织的管理员。
			var hqErr error
			routingOrganizationID, hqErr = uc.dingTalkRegistrations.FindHeadquartersOrganizationID(ctx)
			if hqErr != nil {
				return nil, hqErr
			}
		}
		recipients, _, isEscalated, recipientErr := ListApproverRecipientsWithEscalation(ctx, uc.dingTalkRegistrations, routingOrganizationID)
		if recipientErr != nil {
			return nil, recipientErr
		}
		approverUserIDs := make([]uuid.UUID, 0, len(recipients))
		for _, recipient := range recipients {
			approverUserIDs = append(approverUserIDs, recipient.UserID)
		}
		// 代管标注口径与一键转派一致：目标组织无管理员向上追溯时，
		// 通知卡片展示目标组织名并追加 DingTalkEscalatedOrgSuffix 后缀。
		if isEscalated && notificationOrgName != "" {
			notificationOrgName += DingTalkEscalatedOrgSuffix
		}
		notice = &DingTalkApproverNotice{ApproverUserIDs: approverUserIDs, OrganizationName: notificationOrgName}
	}
	credential, created, err := uc.repo.RegisterDingTalkCredential(ctx, identity, requestedOrganization, notice, &AuditEvent{Action: "auth.dingtalk.register", Result: "success"})
	if err != nil {
		return nil, err
	}
	if !created && credential.Enabled {
		return nil, ErrDingTalkAlreadyRegistered
	}
	return &DingTalkRegistrationResult{DisplayName: credential.DisplayName, Status: "PENDING"}, nil
}

func (uc *AuthUsecase) createSession(ctx context.Context, credential *Credential, organizationID uuid.UUID, choices []OrganizationChoice, userAgent, auditAction, clearLoginFailureKey string) (*AuthSessionResult, error) {
	principal, err := uc.repo.ResolvePrincipal(ctx, credential.UserID, organizationID)
	if err != nil {
		return nil, err
	}
	rawToken, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(uc.policy.TTL)
	if err := uc.repo.CreateSession(ctx, &Session{TokenHash: tokenHash, UserID: credential.UserID, OrganizationID: organizationID, ExpiresAt: expiresAt, UserAgent: userAgent}, clearLoginFailureKey, &AuditEvent{OrganizationID: &organizationID, UserID: &credential.UserID, Action: auditAction, Result: "success"}); err != nil {
		return nil, err
	}
	principal.SessionTokenHash = tokenHash
	return &AuthSessionResult{Token: rawToken, Principal: principal, OrganizationChoices: choices, ExpiresAt: expiresAt}, nil
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

// SwitchOrganization 把当前会话轮转为目标组织的新会话：校验目标在本人启用中成员资格候选集内，
// 权限主体随目标组织重算，事务内新建会话并仅失效当前令牌（同一用户其他设备的会话不受影响）。
func (uc *AuthUsecase) SwitchOrganization(ctx context.Context, principal *Principal, organizationID uuid.UUID) (*AuthSessionResult, error) {
	choices, err := uc.listEnabledMembershipOrganizations(ctx, principal.UserID)
	if err != nil {
		return nil, err
	}
	if !organizationChoicesContain(choices, organizationID) {
		return nil, ErrAuthOrganizationForbidden
	}
	next, err := uc.repo.ResolvePrincipal(ctx, principal.UserID, organizationID)
	if err != nil {
		return nil, err
	}
	rawToken, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(uc.policy.TTL)
	next.SessionTokenHash = tokenHash
	sourceOrganizationID := principal.Organization.ID
	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &principal.UserID,
		Action:         "auth.organization.switch",
		Result:         "success",
		Details:        map[string]string{"source_organization.id": sourceOrganizationID.String(), "target_organization.id": organizationID.String()},
	}
	if err := uc.repo.RotateSession(ctx, principal.SessionTokenHash, &Session{TokenHash: tokenHash, UserID: principal.UserID, OrganizationID: organizationID, ExpiresAt: expiresAt}, now, audit); err != nil {
		return nil, err
	}
	return &AuthSessionResult{Token: rawToken, Principal: next, OrganizationChoices: choices, ExpiresAt: expiresAt}, nil
}

// listEnabledMembershipOrganizations 返回登录组织选择与切换入口共用的候选组织列表：
// 本人启用中成员资格 × 组织启用中，默认组织置首并标记，其余按组织名称稳定排序。
func (uc *AuthUsecase) listEnabledMembershipOrganizations(ctx context.Context, userID uuid.UUID) ([]OrganizationChoice, error) {
	choices, err := uc.repo.ListEnabledMembershipOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}
	sortOrganizationChoices(choices)
	return choices, nil
}

func sortOrganizationChoices(choices []OrganizationChoice) {
	sort.SliceStable(choices, func(i, j int) bool {
		if choices[i].IsDefault != choices[j].IsDefault {
			return choices[i].IsDefault
		}
		if choices[i].OrganizationName != choices[j].OrganizationName {
			return choices[i].OrganizationName < choices[j].OrganizationName
		}
		return choices[i].OrganizationID.String() < choices[j].OrganizationID.String()
	})
}

func organizationChoicesContain(choices []OrganizationChoice, organizationID uuid.UUID) bool {
	for _, choice := range choices {
		if choice.OrganizationID == organizationID {
			return true
		}
	}
	return false
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
