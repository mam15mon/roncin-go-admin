package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/loginratelimitbucket"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
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
	account, err := client.User.Query().Where(user.UsernameEQ(username), user.EnabledEQ(true)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrInvalidCredentials, nil)
	}
	return r.credentialForAccount(ctx, account)
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

func (r *authRepo) FindOrCreateWeComCredential(ctx context.Context, identity *biz.WeComIdentity, audit *biz.AuditEvent) (*biz.Credential, bool, error) {
	if identity == nil || strings.TrimSpace(identity.UserID) == "" || strings.TrimSpace(identity.Name) == "" {
		return nil, false, biz.ErrWeComLoginFailed
	}
	wecomUserID := strings.TrimSpace(identity.UserID)
	wecomName := strings.TrimSpace(identity.Name)
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, false, err
	}
	account, err := client.User.Query().Where(user.WecomUseridEQ(wecomUserID)).Only(ctx)
	if err == nil {
		if account.WecomName == nil || *account.WecomName != wecomName {
			account, err = account.Update().SetWecomName(wecomName).Save(ctx)
			if err != nil {
				return nil, false, err
			}
		}
		credential, credentialErr := r.credentialForAccount(ctx, account)
		return credential, false, credentialErr
	}
	if !ent.IsNotFound(err) {
		return nil, false, err
	}

	digest := sha256.Sum256([]byte(wecomUserID))
	username := "wecom_" + hex.EncodeToString(digest[:12])
	var headquarters *ent.Organization
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var queryErr error
		headquarters, queryErr = tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		create := tx.User.Create().SetUsername(username).SetDisplayName(wecomName).SetWecomUserid(wecomUserID).SetWecomName(wecomName).SetEnabled(false)
		if identity.Email != nil && strings.TrimSpace(*identity.Email) != "" {
			create.SetEmail(strings.TrimSpace(*identity.Email))
		}
		var createErr error
		account, createErr = create.Save(ctx)
		if createErr != nil {
			return createErr
		}
		if _, createErr := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(headquarters.ID).SetPrimary(true).SetEnabled(true).Save(ctx); createErr != nil {
			return createErr
		}
		audit.UserID = &account.ID
		audit.OrganizationID = &headquarters.ID
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, false, err
	}
	return credentialFromAccount(account, headquarters.ID), true, nil
}

func (r *authRepo) FindDingTalkCredential(ctx context.Context, identity *biz.DingTalkIdentity) (*biz.Credential, error) {
	if identity == nil || strings.TrimSpace(identity.UnionID) == "" || strings.TrimSpace(identity.UserID) == "" {
		return nil, biz.ErrDingTalkLoginFailed
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	account, err := client.User.Query().Where(user.DingtalkUnionidEQ(strings.TrimSpace(identity.UnionID))).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkNotRegistered, nil)
	}
	account, err = updateDingTalkProfile(ctx, account, identity)
	if err != nil {
		return nil, err
	}
	return r.credentialForAccount(ctx, account)
}

// RegisterDingTalkCredential 注册钉钉账号（PENDING 禁用 + 总部收口成员资格）。
// requestedOrganizationID 为通道 B 自选目标组织（可空）；notice 为 biz 决策好的
// 审批通知路由（追溯收件人 + 展示组织名，含代管标注）。注册与「注册待审批」
// 通知入队同事务完成，路由组织 = 自选目标 ?? 总部收口组织。
func (r *authRepo) RegisterDingTalkCredential(ctx context.Context, identity *biz.DingTalkIdentity, requestedOrganizationID *uuid.UUID, notice *biz.DingTalkApproverNotice, audit *biz.AuditEvent) (*biz.Credential, bool, error) {
	if identity == nil || strings.TrimSpace(identity.UnionID) == "" || strings.TrimSpace(identity.UserID) == "" || strings.TrimSpace(identity.Name) == "" {
		return nil, false, biz.ErrDingTalkLoginFailed
	}
	unionID := strings.TrimSpace(identity.UnionID)
	dingTalkUserID := strings.TrimSpace(identity.UserID)
	dingtalkName := strings.TrimSpace(identity.Name)
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, false, err
	}
	account, err := client.User.Query().Where(user.DingtalkUnionidEQ(unionID)).Only(ctx)
	if err == nil {
		account, err = updateDingTalkProfile(ctx, account, identity)
		if err != nil {
			return nil, false, err
		}
		hasActiveMembership, queryErr := client.Membership.Query().Where(
			membership.UserIDEQ(account.ID),
			membership.EnabledEQ(true),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
		).Exist(ctx)
		if queryErr != nil {
			return nil, false, queryErr
		}
		if !hasActiveMembership {
			credential, prepareErr := r.prepareDingTalkRehire(ctx, account, requestedOrganizationID, notice, audit)
			return credential, prepareErr == nil, prepareErr
		}
		if account.Enabled {
			// 已启用账号不做任何变更，由 biz 返回「已完成注册」。
			credential, credentialErr := r.credentialForAccount(ctx, account)
			return credential, false, credentialErr
		}
		// 仍处待授权状态：更新自选目标组织并（在目标变化时）重新提醒审批人。
		if err := r.updateDingTalkRequestedOrganization(ctx, account.ID, requestedOrganizationID, notice, dingtalkName); err != nil {
			return nil, false, err
		}
		credential, credentialErr := r.credentialForAccount(ctx, account)
		return credential, false, credentialErr
	}
	if !ent.IsNotFound(err) {
		return nil, false, err
	}

	var headquarters *ent.Organization
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var queryErr error
		headquarters, queryErr = tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		create := tx.User.Create().SetDisplayName(dingtalkName).SetDingtalkUnionid(unionID).SetDingtalkUserid(dingTalkUserID).SetDingtalkName(dingtalkName).SetEnabled(false)
		if requestedOrganizationID != nil {
			create.SetDingtalkRequestedOrganizationID(*requestedOrganizationID)
		}
		if identity.Email != nil && strings.TrimSpace(*identity.Email) != "" {
			create.SetEmail(strings.TrimSpace(*identity.Email))
		}
		if identity.AvatarURL != nil && strings.TrimSpace(*identity.AvatarURL) != "" {
			create.SetAvatarURL(strings.TrimSpace(*identity.AvatarURL))
		}
		var createErr error
		account, createErr = create.Save(ctx)
		if createErr != nil {
			return createErr
		}
		if _, createErr := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(headquarters.ID).SetPrimary(true).SetEnabled(true).Save(ctx); createErr != nil {
			return createErr
		}
		if err := enqueueDingTalkRegistrationPendingNotifications(ctx, tx, resolveRegistrationRoutingOrganization(requestedOrganizationID, headquarters.ID), account.ID, dingtalkName, approverNoticeOrganizationName(notice, headquarters.Name), approverNoticeUserIDs(notice)); err != nil {
			return err
		}
		audit.UserID = &account.ID
		audit.OrganizationID = &headquarters.ID
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, false, err
	}
	return credentialFromAccount(account, headquarters.ID), true, nil
}

// approverNoticeOrganizationName 返回审批通知卡片展示的组织名：biz 已按代管
// 口径组合好展示名；空值表示未自选目标组织（总部收口），按路由组织自身名展示。
func approverNoticeOrganizationName(notice *biz.DingTalkApproverNotice, fallback string) string {
	if notice != nil && strings.TrimSpace(notice.OrganizationName) != "" {
		return notice.OrganizationName
	}
	return fallback
}

// approverNoticeUserIDs 返回审批通知收件人（biz 向上追溯后的结果）。
func approverNoticeUserIDs(notice *biz.DingTalkApproverNotice) []uuid.UUID {
	if notice == nil {
		return nil
	}
	return notice.ApproverUserIDs
}

// resolveRegistrationRoutingOrganization 返回注册审批通知的路由组织：
// 自选了目标组织用自选值，否则用总部收口组织（总部兜底）。
func resolveRegistrationRoutingOrganization(requestedOrganizationID *uuid.UUID, headquartersID uuid.UUID) uuid.UUID {
	if requestedOrganizationID != nil {
		return *requestedOrganizationID
	}
	return headquartersID
}

// updateDingTalkRequestedOrganization 更新待授权注册的自选目标组织；目标组织
// 变化时向新路由组织重新入队审批通知（确定性幂等键自动去重同一组织的重复提醒）。
func (r *authRepo) updateDingTalkRequestedOrganization(ctx context.Context, userID uuid.UUID, requestedOrganizationID *uuid.UUID, notice *biz.DingTalkApproverNotice, registrantName string) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := tx.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		update := tx.User.UpdateOneID(userID)
		changed := false
		if requestedOrganizationID == nil {
			if account.DingtalkRequestedOrganizationID != nil {
				update.ClearDingtalkRequestedOrganizationID()
				changed = true
			}
		} else if account.DingtalkRequestedOrganizationID == nil || *account.DingtalkRequestedOrganizationID != *requestedOrganizationID {
			update.SetDingtalkRequestedOrganizationID(*requestedOrganizationID)
			changed = true
		}
		if changed {
			if _, updateErr := update.Save(ctx); updateErr != nil {
				return updateErr
			}
		}
		if !changed || len(approverNoticeUserIDs(notice)) == 0 {
			return nil
		}
		// 自选目标变化（或清空回总部兜底）时向新路由组织重新提醒审批人。
		var routingOrganization *ent.Organization
		var routingErr error
		if requestedOrganizationID != nil {
			routingOrganization, routingErr = tx.Organization.Query().Where(organization.IDEQ(*requestedOrganizationID)).Only(ctx)
		} else {
			routingOrganization, routingErr = tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
		}
		if routingErr != nil {
			return routingErr
		}
		return enqueueDingTalkRegistrationPendingNotifications(ctx, tx, routingOrganization.ID, userID, registrantName, approverNoticeOrganizationName(notice, routingOrganization.Name), approverNoticeUserIDs(notice))
	})
}

func (r *authRepo) prepareDingTalkRehire(ctx context.Context, account *ent.User, requestedOrganizationID *uuid.UUID, notice *biz.DingTalkApproverNotice, audit *biz.AuditEvent) (*biz.Credential, error) {
	userID := account.ID
	var headquarters *ent.Organization
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, queryErr := tx.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx); queryErr != nil {
			return queryErr
		}
		var queryErr error
		headquarters, queryErr = tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(userID)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(userID)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		intake, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(userID), membership.OrganizationIDEQ(headquarters.ID)).Only(ctx)
		if ent.IsNotFound(queryErr) {
			_, queryErr = tx.Membership.Create().SetUserID(userID).SetOrganizationID(headquarters.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
		} else if queryErr == nil {
			_, queryErr = tx.Membership.UpdateOneID(intake.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
		}
		if queryErr != nil {
			return queryErr
		}
		userUpdate := tx.User.UpdateOneID(userID).SetEnabled(false)
		if requestedOrganizationID == nil {
			userUpdate.ClearDingtalkRequestedOrganizationID()
		} else {
			userUpdate.SetDingtalkRequestedOrganizationID(*requestedOrganizationID)
		}
		if _, updateErr := userUpdate.Save(ctx); updateErr != nil {
			return updateErr
		}
		if err := enqueueDingTalkRegistrationPendingNotifications(ctx, tx, resolveRegistrationRoutingOrganization(requestedOrganizationID, headquarters.ID), userID, account.DisplayName, approverNoticeOrganizationName(notice, headquarters.Name), approverNoticeUserIDs(notice)); err != nil {
			return err
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(userID), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		audit.UserID = &userID
		audit.OrganizationID = &headquarters.ID
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	account.Enabled = false
	return credentialFromAccount(account, headquarters.ID), nil
}

// authOrgNode 是工作台判定的组织最小投影（kind 用字符串承载，纯函数不依赖 Ent 类型）。
type authOrgNode struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
	Kind     string
	Enabled  bool
}

// authOrganizationNodes 把组织实体列表投影为按 ID 索引的工作台判定节点表。
func authOrganizationNodes(nodes []*ent.Organization) map[uuid.UUID]authOrgNode {
	result := make(map[uuid.UUID]authOrgNode, len(nodes))
	for _, node := range nodes {
		result[node.ID] = authOrgNode{ID: node.ID, ParentID: node.ParentID, Kind: string(node.Kind), Enabled: node.Enabled}
	}
	return result
}

// isWorkspaceKind 判定组织 kind 是否为工作台：工作台 = 总部 + 公司节点；
// 部门/团队任何情况下不是登录/切换的工作台。
func isWorkspaceKind(kind string) bool {
	return kind == string(organization.KindHeadquarters) || kind == string(organization.KindCompany)
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
// 总部工作台只含总部节点本身（不跨公司聚合，保持公司间隔离），
// 公司工作台含公司节点及其整个子树（部门角色在公司工作区生效）。
func workspaceMembershipScopeIDs(nodes map[uuid.UUID]authOrgNode, workspaceID uuid.UUID) []uuid.UUID {
	node, ok := nodes[workspaceID]
	if !ok || !isWorkspaceKind(node.Kind) {
		return nil
	}
	if node.Kind == string(organization.KindHeadquarters) {
		return []uuid.UUID{workspaceID}
	}
	return organizationSubtreeIDs(nodes, workspaceID)
}

// membershipRef 是工作台映射所需的成员关系最小投影。
type membershipRef struct {
	OrganizationID uuid.UUID
	Primary        bool
}

func membershipRefsFromEnt(memberships []*ent.Membership) []membershipRef {
	refs := make([]membershipRef, 0, len(memberships))
	for _, member := range memberships {
		refs = append(refs, membershipRef{OrganizationID: member.OrganizationID, Primary: member.Primary})
	}
	return refs
}

// resolveMembershipWorkspaces 把启用成员关系向上取整到所属工作台：成员关系节点沿父链
// 归入最近的工作台 kind 祖先（部门/小组成员关系归属所属公司，总部成员关系归属总部
// 本身）；最近工作台祖先停用时不向上越级回退，直接丢弃。返回去重后的启用工作台集合
// 与 primary 工作台（primary 成员关系映射出的工作台，primary 缺失或不可用时回退
// 第一个可用工作台）。
func resolveMembershipWorkspaces(memberships []membershipRef, nodes map[uuid.UUID]authOrgNode) (workspaceIDs []uuid.UUID, primaryWorkspaceID uuid.UUID) {
	seen := make(map[uuid.UUID]struct{}, len(memberships))
	fallback := uuid.Nil
	for _, member := range memberships {
		workspaceID := workspaceAncestorID(nodes, member.OrganizationID)
		if workspaceID == uuid.Nil {
			continue
		}
		if workspace := nodes[workspaceID]; !workspace.Enabled {
			continue
		}
		if _, duplicated := seen[workspaceID]; !duplicated {
			seen[workspaceID] = struct{}{}
			workspaceIDs = append(workspaceIDs, workspaceID)
		}
		if fallback == uuid.Nil {
			fallback = workspaceID
		}
		if primaryWorkspaceID == uuid.Nil && member.Primary {
			primaryWorkspaceID = workspaceID
		}
	}
	if primaryWorkspaceID == uuid.Nil {
		primaryWorkspaceID = fallback
	}
	return workspaceIDs, primaryWorkspaceID
}

// collectWorkspaceRoleGrants 收集工作台范围内启用成员关系上的启用角色并集（按角色
// 去重）：范围由 workspaceMembershipScopeIDs 决定——总部工作台只取总部节点本身的
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
			grant, enabled := roleGrantFromEntRole(role)
			if !enabled {
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

func credentialFromAccount(account *ent.User, organizationID uuid.UUID) *biz.Credential {
	return &biz.Credential{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, PasswordHash: account.PasswordHash, Enabled: account.Enabled, IsBootstrapAdmin: account.IsBootstrapAdmin, PrimaryOrganizationID: organizationID}
}

func updateDingTalkProfile(ctx context.Context, account *ent.User, identity *biz.DingTalkIdentity) (*ent.User, error) {
	update := account.Update()
	changed := false
	dingtalkName := strings.TrimSpace(identity.Name)
	dingTalkUserID := strings.TrimSpace(identity.UserID)
	if dingTalkUserID != "" && (account.DingtalkUserid == nil || *account.DingtalkUserid != dingTalkUserID) {
		update.SetDingtalkUserid(dingTalkUserID)
		changed = true
	}
	if dingtalkName != "" && (account.DingtalkName == nil || *account.DingtalkName != dingtalkName) {
		update.SetDingtalkName(dingtalkName)
		changed = true
	}
	if identity.AvatarURL != nil {
		avatarURL := strings.TrimSpace(*identity.AvatarURL)
		if avatarURL != "" && (account.AvatarURL == nil || *account.AvatarURL != avatarURL) {
			update.SetAvatarURL(avatarURL)
			changed = true
		}
	}
	if !changed {
		return account, nil
	}
	return update.Save(ctx)
}

// credentialForAccount 计算登录默认工作台：primary 成员关系所在组织向上取整到所属
// 工作台节点（部门/小组 → 所属公司，总部 → 总部）；primary 缺失或其工作台不可用时
// 回退第一个可用工作台。primary 标记语义不变，只影响默认落点。
func (r *authRepo) credentialForAccount(ctx context.Context, account *ent.User) (*biz.Credential, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	refs, err := r.loadEnabledMembershipRefs(ctx, client, account.ID)
	if err != nil {
		return nil, err
	}
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return nil, err
	}
	_, primaryWorkspaceID := resolveMembershipWorkspaces(refs, authOrganizationNodes(nodes))
	if primaryWorkspaceID == uuid.Nil {
		return nil, biz.ErrOrganizationForbidden
	}
	return &biz.Credential{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, PasswordHash: account.PasswordHash, Enabled: account.Enabled, IsBootstrapAdmin: account.IsBootstrapAdmin, PrimaryOrganizationID: primaryWorkspaceID}, nil
}

// ListEnabledMembershipOrganizations 返回普通用户的工作台候选快照，供登录组织选择与
// 应用内切换入口共用同一谓词：总部节点要求本人在总部节点本身有启用成员关系；公司节点
// 要求本人在该公司节点或其子树内任一启用节点有启用成员关系（部门/小组成员关系让所属
// 公司成为候选）。候选均为启用中工作台节点，部门/团队任何情况下不进入候选。
// IsDefault 按 primary 成员关系映射出的工作台标记，排序由 biz 统一处理。
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
	workspaceIDs, primaryWorkspaceID := resolveMembershipWorkspaces(refs, authOrganizationNodes(nodes))
	choices := make([]biz.OrganizationChoice, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		org, ok := organizationsByID[workspaceID]
		if !ok {
			return nil, biz.ErrOrganizationForbidden
		}
		choices = append(choices, biz.OrganizationChoice{OrganizationID: org.ID, OrganizationName: org.Name, OrganizationCode: org.Code, IsDefault: workspaceID == primaryWorkspaceID, Kind: biz.OrganizationKind(org.Kind)})
	}
	return choices, nil
}

// ListEnabledOrganizations 返回全部启用中工作台节点（总部+公司）的候选快照，仅供
// bootstrap 管理员的工作台穿透候选使用；部门/团队不是工作台，任何情况下不进入候选；
// 普通用户候选仍由 ListEnabledMembershipOrganizations 的子树成员资格谓词提供。
// IsDefault 按用户 primary 启用成员资格映射出的工作台标记，排序由 biz 统一处理。
func (r *authRepo) ListEnabledOrganizations(ctx context.Context, userID uuid.UUID) ([]biz.OrganizationChoice, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := loadOrganizationTree(ctx, client)
	if err != nil {
		return nil, err
	}
	refs, err := r.loadEnabledMembershipRefs(ctx, client, userID)
	if err != nil {
		return nil, err
	}
	_, primaryWorkspaceID := resolveMembershipWorkspaces(refs, authOrganizationNodes(nodes))
	choices := make([]biz.OrganizationChoice, 0, len(nodes))
	for _, org := range nodes {
		if !org.Enabled || !isWorkspaceKind(string(org.Kind)) {
			continue
		}
		choices = append(choices, biz.OrganizationChoice{OrganizationID: org.ID, OrganizationName: org.Name, OrganizationCode: org.Code, IsDefault: org.ID == primaryWorkspaceID, Kind: biz.OrganizationKind(org.Kind)})
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
	if account.IsBootstrapAdmin {
		// bootstrap 管理员按已批准的全组织穿透方案解析任意启用中工作台：候选组织为
		// 全部启用中总部+公司节点；无成员关系时授权合成为等价 administrator 的全量授权投影。
		organizations, current, roleGrants, err = resolveBootstrapAdminPrincipalView(ctx, client, nodes, organizationsByID, organizationID)
		if err != nil {
			return nil, err
		}
	} else {
		// 普通用户按工作台口径解析：候选 = 成员关系向上取整出的启用工作台集合；
		// 会话工作区必须落在候选内，角色按工作台范围收集（总部只取总部节点本身，
		// 公司聚合公司子树），公司之间互不泄漏。
		nodeMap := authOrganizationNodes(nodes)
		workspaceIDs, _ := resolveMembershipWorkspaces(membershipRefsFromEnt(memberships), nodeMap)
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
	sort.Slice(organizations, func(i, j int) bool { return organizations[i].Code < organizations[j].Code })
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

// resolveBootstrapAdminPrincipalView 构建 bootstrap 管理员的主体投影：候选组织为全部
// 启用中工作台节点（总部+公司）；目标组织仅要求是启用中的工作台节点（不要求成员关系，
// 部门/团队不是工作台）。授权合成为等价 administrator 的全量授权，只存在于本次内存
// 投影，不写库、不改角色与权限数据。
func resolveBootstrapAdminPrincipalView(ctx context.Context, client *ent.Client, nodes []*ent.Organization, organizationsByID map[uuid.UUID]*ent.Organization, organizationID uuid.UUID) ([]biz.Organization, *biz.Organization, []biz.RoleGrant, error) {
	target, ok := organizationsByID[organizationID]
	if !ok || !target.Enabled || !isWorkspaceKind(string(target.Kind)) {
		return nil, nil, nil, biz.ErrOrganizationForbidden
	}
	organizations := make([]biz.Organization, 0, len(nodes))
	for _, node := range nodes {
		if !node.Enabled || !isWorkspaceKind(string(node.Kind)) {
			continue
		}
		baseCurrency, currencyErr := resolvePrincipalOrganizationBaseCurrency(node, organizationsByID)
		if currencyErr != nil {
			return nil, nil, nil, currencyErr
		}
		organizations = append(organizations, biz.Organization{ID: node.ID, Code: node.Code, Name: node.Name, BaseCurrency: baseCurrency, Kind: biz.OrganizationKind(node.Kind)})
	}
	baseCurrency, currencyErr := resolvePrincipalOrganizationBaseCurrency(target, organizationsByID)
	if currencyErr != nil {
		return nil, nil, nil, currencyErr
	}
	current := &biz.Organization{ID: target.ID, Code: target.Code, Name: target.Name, BaseCurrency: baseCurrency, Kind: biz.OrganizationKind(target.Kind)}
	roleGrants, err := bootstrapAdminRoleGrants(ctx, client)
	if err != nil {
		return nil, nil, nil, err
	}
	return organizations, current, roleGrants, nil
}

// bootstrapAdminRoleGrants 为 bootstrap 管理员合成等价 administrator 的授权投影：
// 最宽数据范围 + permissions 表全量权限键。仅在内存 Principal 中生效，不落库。
func bootstrapAdminRoleGrants(ctx context.Context, client *ent.Client) ([]biz.RoleGrant, error) {
	keys, err := client.Permission.Query().Select(permission.FieldKey).All(ctx)
	if err != nil {
		return nil, err
	}
	permissions := make(map[string]struct{}, len(keys))
	for _, item := range keys {
		permissions[item.Key] = struct{}{}
	}
	return []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: permissions}}, nil
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

func roleGrantFromEntRole(role *ent.Role) (biz.RoleGrant, bool) {
	if role == nil || !role.Enabled {
		return biz.RoleGrant{}, false
	}
	grant := biz.RoleGrant{
		RoleID:      role.ID,
		RoleCode:    role.Code,
		RoleName:    role.Name,
		DataScope:   biz.DataScope(role.DataScope),
		Permissions: make(map[string]struct{}, len(role.Edges.Permissions)),
	}
	for _, permission := range role.Edges.Permissions {
		grant.Permissions[permission.Key] = struct{}{}
	}
	return grant, true
}

func (r *authRepo) CreateSession(ctx context.Context, input *biz.Session, clearLoginFailureKey string, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
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

// RotateSession 在同一事务内完成会话令牌轮转：校验目标准入（ForShare）→ 锁定当前
// 会话行（ForUpdate）→ 新建目标组织会话（沿用当前会话的 UA/IP）→ 失效旧令牌 → 写审计。
// 普通用户准入 = 目标为启用中工作台节点，且总部要求本人总部启用成员关系、公司要求本人
// 于公司子树内任一启用节点有启用成员关系（子树口径）；bootstrap 管理员按工作台穿透方案
// 只复核目标为启用中总部/公司节点。只轮转当前令牌，同一用户其他设备的会话不受影响。
func (r *authRepo) RotateSession(ctx context.Context, tokenHash string, next *biz.Session, now time.Time, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 事务内复核准入，防止校验与轮转之间资格被停用（TOCTOU）。
		isBootstrap, queryErr := tx.User.Query().
			Where(user.IDEQ(next.UserID), user.EnabledEQ(true), user.IsBootstrapAdminEQ(true)).
			Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if isBootstrap {
			enabled, orgErr := tx.Organization.Query().
				Where(organization.IDEQ(next.OrganizationID), organization.EnabledEQ(true), organization.KindIn(organization.KindHeadquarters, organization.KindCompany)).
				ForShare().Exist(ctx)
			if orgErr != nil {
				return orgErr
			}
			if !enabled {
				return biz.ErrAuthOrganizationForbidden
			}
		} else {
			// 普通用户复核目标工作台：部门/团队不是工作台；总部与公司按各自口径复核。
			target, orgErr := tx.Organization.Query().
				Where(organization.IDEQ(next.OrganizationID), organization.EnabledEQ(true), organization.KindIn(organization.KindHeadquarters, organization.KindCompany)).
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
