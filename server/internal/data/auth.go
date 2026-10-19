package data

import (
	"context"
	"database/sql"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/loginratelimitbucket"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

type authRepo struct{ data *Data }

func NewAuthRepo(data *Data) biz.AuthRepo { return &authRepo{data: data} }

func (r *authRepo) FindCredential(ctx context.Context, username string) (*biz.Credential, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := loginAccountQuery(client, username).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(accounts) > 1 {
		return nil, biz.ErrAuthAccountConflict
	}
	if len(accounts) == 0 || !loginAccountMatches(accounts[0], username) {
		return nil, biz.ErrInvalidCredentials
	}
	return credentialFromAccount(accounts[0]), nil
}

// loginAccountQuery 在密码校验前联合解析所有身份，启停状态不参与消歧。
func loginAccountQuery(client *ent.Client, account string) *ent.UserQuery {
	normalized := biz.NormalizeDingTalkMobile(account)
	if biz.ValidDingTalkMobile(normalized) {
		return client.User.Query().Where(user.Or(user.UsernameEQ(account), user.UsernameEQ(normalized), user.DingtalkMobileNormalizedEQ(normalized)))
	}
	return client.User.Query().Where(user.UsernameEQ(account))
}

// 规范手机号形式仅作为冲突候选，本地用户名本身仍要求精确匹配。
func loginAccountMatches(account *ent.User, input string) bool {
	if account.Username == input {
		return true
	}
	mobile := biz.NormalizeDingTalkMobile(input)
	return biz.ValidDingTalkMobile(mobile) && account.DingtalkMobileNormalized != nil && *account.DingtalkMobileNormalized == mobile
}

func (r *authRepo) LoginRateLimitExceeded(ctx context.Context, keyHashes []string, now time.Time, window time.Duration, maxAttempts int) (bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	return client.LoginRateLimitBucket.Query().Where(
		loginratelimitbucket.KeyHashIn(keyHashes...),
		loginratelimitbucket.WindowStartedAtGT(now.Add(-window)),
		loginratelimitbucket.AttemptsGTE(maxAttempts),
	).Exist(ctx)
}

func (r *authRepo) RecordLoginFailure(ctx context.Context, keyHashes []string, now time.Time, window time.Duration, maxAttempts int, audit *biz.AuditEvent) (bool, error) {
	exceeded := false
	err := r.data.withSQLTx(ctx, func(tx *sql.Tx) error {
		for _, keyHash := range keyHashes {
			var attempts int
			// PostgreSQL upsert 保证同一限流桶的并发失败计数不会丢失。
			if err := tx.QueryRowContext(ctx, `
INSERT INTO "login_rate_limit_buckets" ("id", "created_at", "updated_at", "key_hash", "window_started_at", "attempts")
VALUES ($1, $2, $2, $3, $2, 1)
ON CONFLICT ("key_hash") DO UPDATE SET
  "updated_at" = EXCLUDED."updated_at",
  "window_started_at" = CASE
    WHEN "login_rate_limit_buckets"."window_started_at" <= $4 THEN EXCLUDED."window_started_at"
    ELSE "login_rate_limit_buckets"."window_started_at"
  END,
  "attempts" = CASE
    WHEN "login_rate_limit_buckets"."window_started_at" <= $4 THEN 1
    ELSE "login_rate_limit_buckets"."attempts" + 1
  END
RETURNING "attempts"`, uuid.Must(uuid.NewV7()), now, keyHash, now.Add(-window)).Scan(&attempts); err != nil {
				return err
			}
			if attempts > maxAttempts {
				exceeded = true
			}
		}
		requestID, traceID, ipAddress, details, err := resolveAuditValues(ctx, audit)
		if err != nil {
			return err
		}
		var resourceType, resourceID, auditDetails any
		if audit.ResourceType != "" {
			resourceType = audit.ResourceType
		}
		if audit.ResourceID != "" {
			resourceID = audit.ResourceID
		}
		if len(details) > 0 {
			auditDetails = string(details)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO "audit_logs" ("id", "created_at", "updated_at", "organization_id", "user_id", "action", "resource_type", "resource_id", "result", "request_id", "trace_id", "ip_address", "details")
VALUES ($1, $2, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, uuid.Must(uuid.NewV7()), now, audit.OrganizationID, audit.UserID, audit.Action, resourceType, resourceID, audit.Result, requestID, traceID, ipAddress, auditDetails)
		return err
	})
	if err != nil {
		return false, err
	}
	return exceeded, nil
}

func (r *authRepo) FindDingTalkCredential(ctx context.Context, identity *biz.DingTalkIdentity) (*biz.Credential, error) {
	if identity == nil || strings.TrimSpace(identity.UnionID) == "" || strings.TrimSpace(identity.UserID) == "" {
		return nil, biz.ErrDingTalkLoginFailed
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	account, err := client.User.Query().Where(user.DingtalkUnionidEQ(identity.UnionID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkNotRegistered, nil)
	}
	if account.DingtalkUserid == nil || *account.DingtalkUserid != identity.UserID {
		return nil, biz.ErrDingTalkIdentityConflict
	}
	// 账号状态由用例先判定，通过后再查询工作台候选。
	return credentialFromAccount(account), nil
}

func authOrganizationNodes(nodes []*ent.Organization) map[uuid.UUID]authOrgNode {
	result := make(map[uuid.UUID]authOrgNode, len(nodes))
	for _, node := range nodes {
		result[node.ID] = authOrgNode{ID: node.ID, ParentID: node.ParentID, Kind: string(node.Kind), Name: node.Name, Enabled: node.Enabled}
	}
	return result
}

// isWorkspaceKind 判定组织 kind 是否为工作台：工作台 = 系统管理 + 公司节点；
// 部门/团队任何情况下不是登录/切换的工作台。
func isWorkspaceKind(kind string) bool {
	return kind == string(organization.KindSystem) || kind == string(organization.KindCompany)
}

// workspaceAncestorID 返回组织节点自身或其最近的工作台 kind 祖先（不关注启用态，
// 由调用方过滤）；组织不在树内或无工作台祖先时返回 uuid.Nil。循环与断链安全。
func workspaceAncestorID(nodes map[uuid.UUID]authOrgNode, organizationID uuid.UUID) uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	for currentID := organizationID; currentID != uuid.Nil; {
		if _, visited := seen[currentID]; visited {
			return uuid.Nil
		}
		seen[currentID] = struct{}{}
		node, ok := nodes[currentID]
		if !ok {
			return uuid.Nil
		}
		if isWorkspaceKind(node.Kind) {
			return currentID
		}
		if node.ParentID == nil {
			return uuid.Nil
		}
		currentID = *node.ParentID
	}
	return uuid.Nil
}

// organizationSubtreeIDs 返回组织节点及其全部后代（含停用后代；启用过滤由查询谓词
// 或调用方负责）。组织总量小，一次全量构建子邻接表后 BFS，避免逐层查询。
func organizationSubtreeIDs(nodes map[uuid.UUID]authOrgNode, rootID uuid.UUID) []uuid.UUID {
	children := make(map[uuid.UUID][]uuid.UUID, len(nodes))
	for _, node := range nodes {
		if node.ParentID == nil {
			continue
		}
		children[*node.ParentID] = append(children[*node.ParentID], node.ID)
	}
	result := []uuid.UUID{rootID}
	visited := map[uuid.UUID]struct{}{rootID: {}}
	for index := 0; index < len(result); index++ {
		for _, childID := range children[result[index]] {
			if _, ok := visited[childID]; ok {
				continue
			}
			visited[childID] = struct{}{}
			result = append(result, childID)
		}
	}
	return result
}

// workspaceMembershipScopeIDs 返回工作台的角色收集与成员资格复核节点范围：
// 系统管理工作台只含系统管理节点本身（不跨公司聚合，保持公司间隔离），
// 公司工作台含公司及其部门/团队，遇到其他工作台截止；启用态由成员查询过滤。
func workspaceMembershipScopeIDs(nodes map[uuid.UUID]authOrgNode, workspaceID uuid.UUID) []uuid.UUID {
	node, ok := nodes[workspaceID]
	if !ok || !isWorkspaceKind(node.Kind) {
		return nil
	}
	if node.Kind == string(organization.KindSystem) {
		return []uuid.UUID{workspaceID}
	}
	result := []uuid.UUID{workspaceID}
	for _, id := range organizationSubtreeIDs(nodes, workspaceID) {
		child := nodes[id]
		if (child.Kind == string(organization.KindDepartment) || child.Kind == string(organization.KindTeam)) && workspaceAncestorID(nodes, id) == workspaceID {
			result = append(result, id)
		}
	}
	return result
}

// membershipRef 是工作台映射所需的成员关系最小投影。
type membershipRef struct {
	OrganizationID uuid.UUID
}

func membershipRefsFromEnt(memberships []*ent.Membership) []membershipRef {
	refs := make([]membershipRef, 0, len(memberships))
	for _, member := range memberships {
		refs = append(refs, membershipRef{OrganizationID: member.OrganizationID})
	}
	return refs
}

// resolveMembershipWorkspaces 将有效成员关系映射为去重的启用工作台，按名称和 ID 排序。
// 部门和组归入所属公司，最近工作台停用时不越级进入其他工作台。
func resolveMembershipWorkspaces(memberships []membershipRef, nodes map[uuid.UUID]authOrgNode) (workspaceIDs []uuid.UUID) {
	seen := make(map[uuid.UUID]struct{}, len(memberships))
	for _, member := range memberships {
		workspaceID := workspaceAncestorID(nodes, member.OrganizationID)
		if workspaceID == uuid.Nil || !nodes[workspaceID].Enabled {
			continue
		}
		if _, exists := seen[workspaceID]; exists {
			continue
		}
		seen[workspaceID] = struct{}{}
		workspaceIDs = append(workspaceIDs, workspaceID)
	}
	sort.Slice(workspaceIDs, func(i, j int) bool {
		if nodes[workspaceIDs[i]].Name != nodes[workspaceIDs[j]].Name {
			return nodes[workspaceIDs[i]].Name < nodes[workspaceIDs[j]].Name
		}
		return workspaceIDs[i].String() < workspaceIDs[j].String()
	})
	return workspaceIDs
}

// collectWorkspaceRoleGrants 收集工作台范围内启用成员关系上的启用角色并集（按角色
// 去重）：范围由 workspaceMembershipScopeIDs 决定——系统管理工作台只取系统管理节点本身的
// 成员关系角色，公司工作台聚合公司子树内全部成员关系角色。
func collectWorkspaceRoleGrants(memberships []*ent.Membership, nodes map[uuid.UUID]authOrgNode, workspaceID uuid.UUID) []biz.RoleGrant {
	scope := make(map[uuid.UUID]struct{})
	for _, scopeID := range workspaceMembershipScopeIDs(nodes, workspaceID) {
		scope[scopeID] = struct{}{}
	}
	roleGrants := make([]biz.RoleGrant, 0)
	seenRoles := make(map[uuid.UUID]struct{})
	for _, member := range memberships {
		if _, inScope := scope[member.OrganizationID]; !inScope {
			continue
		}
		for _, assignment := range member.Edges.RoleAssignments {
			role, roleErr := assignment.Edges.RoleOrErr()
			if roleErr != nil {
				continue
			}
			grant, enabled := roleGrantFromEntRole(role, workspaceDataScope(nodes, workspaceID))
			if !enabled || role.OrganizationID != workspaceID {
				continue
			}
			if _, duplicated := seenRoles[grant.RoleID]; duplicated {
				continue
			}
			seenRoles[grant.RoleID] = struct{}{}
			roleGrants = append(roleGrants, grant)
		}
	}
	return roleGrants
}

// loadEnabledMembershipRefs 查询用户「启用成员关系 × 启用组织」的最小投影。
func (r *authRepo) loadEnabledMembershipRefs(ctx context.Context, client *ent.Client, userID uuid.UUID) ([]membershipRef, error) {
	memberships, err := client.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return membershipRefsFromEnt(memberships), nil
}

// loadOrganizationTree 一次取全量组织（组织总量小）并返回实体列表，供工作台判定
// 构建父子映射，避免逐层/N+1 查询。auth 与 admin 用户管理共用同一口径。
func loadOrganizationTree(ctx context.Context, client *ent.Client) ([]*ent.Organization, error) {
	return client.Organization.Query().
		Select(organization.FieldID, organization.FieldParentID, organization.FieldKind, organization.FieldCode, organization.FieldName, organization.FieldBaseCurrency, organization.FieldEnabled).
		All(ctx)
}

func credentialFromAccount(account *ent.User) *biz.Credential {
	return &biz.Credential{PendingAssignment: account.PendingAssignment, UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, PasswordHash: account.PasswordHash, Enabled: account.Enabled, IsBootstrapAdmin: account.IsBootstrapAdmin}
}

// ListEnabledMembershipOrganizations 返回普通用户的工作台候选快照，供登录组织选择与
// 应用内切换入口共用同一谓词：系统管理节点要求本人在系统管理节点本身有启用成员关系；公司节点
// 要求本人在该公司节点或其子树内任一启用节点有启用成员关系（部门/小组成员关系让所属
// 公司成为候选）。候选均为启用中工作台节点，部门/团队任何情况下不进入候选。
func (r *authRepo) ListEnabledMembershipOrganizations(ctx context.Context, userID uuid.UUID) ([]biz.OrganizationChoice, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	refs, err := r.loadEnabledMembershipRefs(ctx, client, userID)
	if err != nil {
		return nil, err
	}
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return nil, err
	}
	organizationsByID := make(map[uuid.UUID]*ent.Organization, len(nodes))
	for _, node := range nodes {
		organizationsByID[node.ID] = node
	}
	workspaceIDs := resolveMembershipWorkspaces(refs, authOrganizationNodes(nodes))
	choices := make([]biz.OrganizationChoice, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		org, ok := organizationsByID[workspaceID]
		if !ok {
			return nil, biz.ErrOrganizationForbidden
		}
		choices = append(choices, biz.OrganizationChoice{OrganizationID: org.ID, OrganizationName: org.Name, OrganizationCode: org.Code, Kind: biz.OrganizationKind(org.Kind)})
	}
	return choices, nil
}

func (r *authRepo) ResolvePrincipal(ctx context.Context, userID, organizationID uuid.UUID) (*biz.Principal, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	account, err := client.User.Query().Where(user.IDEQ(userID), user.EnabledEQ(true)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSessionExpired, nil)
	}
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return nil, err
	}
	organizationsByID := make(map[uuid.UUID]*ent.Organization, len(nodes))
	for _, node := range nodes {
		organizationsByID[node.ID] = node
	}
	memberships, err := client.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) {
			query.WithRole(func(roleQuery *ent.RoleQuery) {
				roleQuery.Where(roleent.EnabledEQ(true)).WithPermissions()
			})
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var organizations []biz.Organization
	roleGrants := make([]biz.RoleGrant, 0)
	var current *biz.Organization
	{
		// 普通用户按工作台口径解析：候选 = 成员关系向上取整出的启用工作台集合；
		// 会话工作区必须落在候选内，角色按工作台范围收集（系统管理只取系统管理节点本身，
		// 公司聚合公司子树），公司之间互不泄漏。
		nodeMap := authOrganizationNodes(nodes)
		workspaceIDs := resolveMembershipWorkspaces(membershipRefsFromEnt(memberships), nodeMap)
		organizations = make([]biz.Organization, 0, len(workspaceIDs))
		for _, workspaceID := range workspaceIDs {
			org, ok := organizationsByID[workspaceID]
			if !ok {
				return nil, biz.ErrOrganizationForbidden
			}
			baseCurrency, currencyErr := resolvePrincipalOrganizationBaseCurrency(org, organizationsByID)
			if currencyErr != nil {
				return nil, currencyErr
			}
			organizations = append(organizations, biz.Organization{ID: org.ID, Code: org.Code, Name: org.Name, BaseCurrency: baseCurrency, Kind: biz.OrganizationKind(org.Kind)})
		}
		if !slices.Contains(workspaceIDs, organizationID) {
			return nil, biz.ErrOrganizationForbidden
		}
		target := organizationsByID[organizationID]
		baseCurrency, currencyErr := resolvePrincipalOrganizationBaseCurrency(target, organizationsByID)
		if currencyErr != nil {
			return nil, currencyErr
		}
		current = &biz.Organization{ID: target.ID, Code: target.Code, Name: target.Name, BaseCurrency: baseCurrency, Kind: biz.OrganizationKind(target.Kind)}
		roleGrants = collectWorkspaceRoleGrants(memberships, nodeMap, organizationID)
	}
	if current == nil {
		return nil, biz.ErrOrganizationForbidden
	}
	sort.Slice(organizations, func(i, j int) bool {
		if organizations[i].Name != organizations[j].Name {
			return organizations[i].Name < organizations[j].Name
		}
		return organizations[i].ID.String() < organizations[j].ID.String()
	})
	sort.Slice(roleGrants, func(i, j int) bool {
		if roleGrants[i].RoleID == roleGrants[j].RoleID {
			return roleGrants[i].RoleCode < roleGrants[j].RoleCode
		}
		return roleGrants[i].RoleID.String() < roleGrants[j].RoleID.String()
	})
	organizationNodes := make([]biz.OrganizationScopeNode, 0, len(nodes))
	for _, node := range nodes {
		organizationNodes = append(organizationNodes, biz.OrganizationScopeNode{ID: node.ID, ParentID: node.ParentID, Disabled: !node.Enabled, Kind: biz.OrganizationKind(node.Kind)})
	}
	sort.Slice(organizationNodes, func(i, j int) bool {
		return organizationNodes[i].ID.String() < organizationNodes[j].ID.String()
	})
	return &biz.Principal{WorkspaceOrganizationID: organizationID, UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, AvatarURL: account.AvatarURL, IsBootstrapAdmin: account.IsBootstrapAdmin, Organization: *current, Organizations: organizations, RoleGrants: roleGrants, OrganizationNodes: organizationNodes}, nil
}

func resolvePrincipalOrganizationBaseCurrency(item *ent.Organization, organizationsByID map[uuid.UUID]*ent.Organization) (string, error) {
	seen := make(map[uuid.UUID]struct{})
	for current := item; current != nil; {
		if _, visited := seen[current.ID]; visited {
			return "", biz.ErrAdminOrganizationCurrency
		}
		seen[current.ID] = struct{}{}
		if current.BaseCurrency != nil {
			return *current.BaseCurrency, nil
		}
		if current.ParentID == nil {
			return "", biz.ErrAdminOrganizationCurrency
		}
		current = organizationsByID[*current.ParentID]
	}
	return "", biz.ErrAdminOrganizationCurrency
}

func roleGrantFromEntRole(role *ent.Role, scope biz.DataScope) (biz.RoleGrant, bool) {
	if role == nil || !role.Enabled {
		return biz.RoleGrant{}, false
	}
	grant := biz.RoleGrant{
		RoleID:      role.ID,
		RoleCode:    role.Code,
		RoleName:    role.Name,
		DataScope:   scope,
		Permissions: make(map[string]struct{}, len(role.Edges.Permissions)),
	}
	for _, permission := range role.Edges.Permissions {
		grant.Permissions[permission.Key] = struct{}{}
	}
	return grant, true
}

func (r *authRepo) CreateSession(ctx context.Context, input *biz.Session, clearLoginFailureKey string, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, err := tx.User.Query().Where(user.IDEQ(input.UserID)).ForShare().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrInvalidCredentials, nil)
		}
		if !account.Enabled || account.PendingAssignment {
			return biz.ErrInvalidCredentials
		}
		if input.VerifiedPasswordHash != nil {
			if account.PasswordHash == nil || *account.PasswordHash != *input.VerifiedPasswordHash {
				return biz.ErrInvalidCredentials
			}
			accounts, err := loginAccountQuery(tx.Client(), input.LoginAccount).All(ctx)
			if err != nil {
				return err
			}
			if len(accounts) > 1 {
				return biz.ErrAuthAccountConflict
			}
			if len(accounts) != 1 || accounts[0].ID != input.UserID || !loginAccountMatches(accounts[0], input.LoginAccount) {
				return biz.ErrInvalidCredentials
			}
		}
		if err := validateSessionWorkspace(ctx, tx, input.UserID, input.OrganizationID); err != nil {
			return err
		}
		if clearLoginFailureKey != "" {
			if _, deleteErr := tx.LoginRateLimitBucket.Delete().Where(loginratelimitbucket.KeyHashEQ(clearLoginFailureKey)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, createErr := tx.Session.Create().SetTokenHash(input.TokenHash).SetUserID(input.UserID).SetOrganizationID(input.OrganizationID).SetExpiresAt(input.ExpiresAt).SetUserAgent(input.UserAgent).Save(ctx); createErr != nil {
			return createErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *authRepo) FindSession(ctx context.Context, tokenHash string, now time.Time) (*biz.Session, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	stored, err := client.Session.Query().Where(sessionent.TokenHashEQ(tokenHash), sessionent.RevokedAtIsNil(), sessionent.ExpiresAtGT(now), sessionent.HasUserWith(user.EnabledEQ(true)), sessionent.HasOrganizationWith(organization.EnabledEQ(true))).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSessionExpired, nil)
	}
	return &biz.Session{TokenHash: stored.TokenHash, UserID: stored.UserID, OrganizationID: stored.OrganizationID, ExpiresAt: stored.ExpiresAt, UserAgent: stored.UserAgent}, nil
}

// RotateSession 在同一事务内完成会话令牌轮转：锁定账号（ForShare）→ 校验目标准入（ForShare）→ 锁定当前
// 会话行（ForUpdate）→ 新建目标组织会话（沿用当前会话的 UA/IP）→ 失效旧令牌 → 写审计。
// 普通用户准入 = 目标为启用中工作台节点，且系统管理要求本人系统管理启用成员关系、公司要求本人
// 于公司子树内任一启用节点有启用成员关系（子树口径）；初始化管理员同样适用。只轮转当前令牌，同一用户其他设备的会话不受影响。
func (r *authRepo) RotateSession(ctx context.Context, tokenHash string, next *biz.Session, now time.Time, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 与成员授权、公司管理员补齐及账号停用统一按用户→组织→成员→会话加锁。
		// 若先锁组织/成员，插入会话取得用户外键 KEY SHARE 时会与授权写入反向等待。
		account, accountErr := tx.User.Query().Where(user.IDEQ(next.UserID)).ForShare().Only(ctx)
		if accountErr != nil {
			return mapEntError(accountErr, biz.ErrSessionExpired, nil)
		}
		if !account.Enabled || account.PendingAssignment {
			return biz.ErrSessionExpired
		}
		// 事务内复核准入，防止校验与轮转之间资格被停用（TOCTOU）。
		{
			// 普通用户复核目标工作台：部门/团队不是工作台；系统管理与公司按各自口径复核。
			target, orgErr := tx.Organization.Query().
				Where(organization.IDEQ(next.OrganizationID), organization.EnabledEQ(true), organization.KindIn(organization.KindSystem, organization.KindCompany)).
				ForShare().Only(ctx)
			if ent.IsNotFound(orgErr) {
				return biz.ErrAuthOrganizationForbidden
			}
			if orgErr != nil {
				return orgErr
			}
			treeNodes, treeErr := tx.Organization.Query().
				Select(organization.FieldID, organization.FieldParentID, organization.FieldKind).
				All(ctx)
			if treeErr != nil {
				return treeErr
			}
			scopeIDs := workspaceMembershipScopeIDs(authOrganizationNodes(treeNodes), target.ID)
			if len(scopeIDs) == 0 {
				return biz.ErrAuthOrganizationForbidden
			}
			exists, memberErr := tx.Membership.Query().Where(
				membership.UserIDEQ(next.UserID),
				membership.EnabledEQ(true),
				membership.OrganizationIDIn(scopeIDs...),
				membership.HasOrganizationWith(organization.EnabledEQ(true)),
			).ForShare().Exist(ctx)
			if memberErr != nil {
				return memberErr
			}
			if !exists {
				return biz.ErrAuthOrganizationForbidden
			}
		}
		current, queryErr := tx.Session.Query().Where(
			sessionent.TokenHashEQ(tokenHash),
			sessionent.RevokedAtIsNil(),
			sessionent.ExpiresAtGT(now),
		).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSessionExpired, nil)
		}
		if current.UserID != next.UserID {
			return biz.ErrSessionExpired
		}
		if _, createErr := tx.Session.Create().
			SetTokenHash(next.TokenHash).
			SetUserID(next.UserID).
			SetOrganizationID(next.OrganizationID).
			SetExpiresAt(next.ExpiresAt).
			SetNillableUserAgent(&current.UserAgent).
			SetNillableIPAddress(&current.IPAddress).
			Save(ctx); createErr != nil {
			return createErr
		}
		if _, updateErr := current.Update().SetRevokedAt(now).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *authRepo) RevokeSession(ctx context.Context, tokenHash string, now time.Time, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		updated, updateErr := tx.Session.Update().Where(sessionent.TokenHashEQ(tokenHash), sessionent.RevokedAtIsNil()).SetRevokedAt(now).Save(ctx)
		if updateErr != nil {
			return updateErr
		}
		if updated != 1 {
			return biz.ErrSessionExpired
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

var _ biz.AuthRepo = (*authRepo)(nil)

// authOrgNode 是工作台判定的最小组织投影。
type authOrgNode struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
	Kind     string
	Name     string
	Enabled  bool
}

func uuidInValues(values []uuid.UUID, target uuid.UUID) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

// workspaceDataScope 只从工作台类型推导内部授权边界，未知节点不给予有效授权。
func workspaceDataScope(nodes map[uuid.UUID]authOrgNode, workspaceID uuid.UUID) biz.DataScope {
	node, ok := nodes[workspaceID]
	if !ok {
		return ""
	}
	switch node.Kind {
	case "system":
		return biz.DataScopeAll
	case "company":
		return biz.DataScopeOrganization
	default:
		return ""
	}
}

// validateSessionWorkspace 在用户锁后复核工作台和成员资格，供建立新会话使用。
func validateSessionWorkspace(ctx context.Context, tx *ent.Tx, userID, organizationID uuid.UUID) error {
	nodes, err := tx.Organization.Query().Order(ent.Asc(organization.FieldID)).ForShare().All(ctx)
	if err != nil {
		return err
	}
	members, err := tx.Membership.Query().Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Order(ent.Asc(membership.FieldID)).ForShare().All(ctx)
	if err != nil {
		return err
	}
	for _, id := range resolveMembershipWorkspaces(membershipRefsFromEnt(members), authOrganizationNodes(nodes)) {
		if id == organizationID {
			return nil
		}
	}
	return biz.ErrAuthOrganizationForbidden
}
