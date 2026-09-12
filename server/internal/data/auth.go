package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/loginratelimitbucket"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

type authRepo struct{ data *Data }

func NewAuthRepo(data *Data) biz.AuthRepo { return &authRepo{data: data} }

func (r *authRepo) FindCredential(ctx context.Context, username string) (*biz.Credential, error) {
	account, err := r.data.db.User.Query().Where(user.UsernameEQ(username), user.EnabledEQ(true)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrInvalidCredentials, nil)
	}
	return r.credentialForAccount(ctx, account)
}

func (r *authRepo) LoginRateLimitExceeded(ctx context.Context, keyHashes []string, now time.Time, window time.Duration, maxAttempts int) (bool, error) {
	return r.data.db.LoginRateLimitBucket.Query().Where(
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
	account, err := r.data.db.User.Query().Where(user.WecomUseridEQ(wecomUserID)).Only(ctx)
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
	account, err := r.data.db.User.Query().Where(user.DingtalkUnionidEQ(strings.TrimSpace(identity.UnionID))).Only(ctx)
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
// requestedOrganizationID 为通道 B 自选目标组织（可空）；注册与「注册待审批」
// 通知入队同事务完成，路由组织 = 自选目标 ?? 总部收口组织。
func (r *authRepo) RegisterDingTalkCredential(ctx context.Context, identity *biz.DingTalkIdentity, requestedOrganizationID *uuid.UUID, approverUserIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.Credential, bool, error) {
	if identity == nil || strings.TrimSpace(identity.UnionID) == "" || strings.TrimSpace(identity.UserID) == "" || strings.TrimSpace(identity.Name) == "" {
		return nil, false, biz.ErrDingTalkLoginFailed
	}
	unionID := strings.TrimSpace(identity.UnionID)
	dingTalkUserID := strings.TrimSpace(identity.UserID)
	dingtalkName := strings.TrimSpace(identity.Name)
	account, err := r.data.db.User.Query().Where(user.DingtalkUnionidEQ(unionID)).Only(ctx)
	if err == nil {
		account, err = updateDingTalkProfile(ctx, account, identity)
		if err != nil {
			return nil, false, err
		}
		hasActiveMembership, queryErr := r.data.db.Membership.Query().Where(
			membership.UserIDEQ(account.ID),
			membership.EnabledEQ(true),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
		).Exist(ctx)
		if queryErr != nil {
			return nil, false, queryErr
		}
		if !hasActiveMembership {
			credential, prepareErr := r.prepareDingTalkRehire(ctx, account, requestedOrganizationID, approverUserIDs, audit)
			return credential, prepareErr == nil, prepareErr
		}
		if account.Enabled {
			// 已启用账号不做任何变更，由 biz 返回「已完成注册」。
			credential, credentialErr := r.credentialForAccount(ctx, account)
			return credential, false, credentialErr
		}
		// 仍处待授权状态：更新自选目标组织并（在目标变化时）重新提醒审批人。
		if err := r.updateDingTalkRequestedOrganization(ctx, account.ID, requestedOrganizationID, approverUserIDs, dingtalkName); err != nil {
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
		if err := enqueueDingTalkRegistrationPendingNotifications(ctx, tx, resolveRegistrationRoutingOrganization(requestedOrganizationID, headquarters.ID), account.ID, dingtalkName, headquarters.Name, approverUserIDs); err != nil {
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
func (r *authRepo) updateDingTalkRequestedOrganization(ctx context.Context, userID uuid.UUID, requestedOrganizationID *uuid.UUID, approverUserIDs []uuid.UUID, registrantName string) error {
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
		if !changed || len(approverUserIDs) == 0 {
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
		return enqueueDingTalkRegistrationPendingNotifications(ctx, tx, routingOrganization.ID, userID, registrantName, routingOrganization.Name, approverUserIDs)
	})
}

func (r *authRepo) prepareDingTalkRehire(ctx context.Context, account *ent.User, requestedOrganizationID *uuid.UUID, approverUserIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.Credential, error) {
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
		if err := enqueueDingTalkRegistrationPendingNotifications(ctx, tx, resolveRegistrationRoutingOrganization(requestedOrganizationID, headquarters.ID), userID, account.DisplayName, headquarters.Name, approverUserIDs); err != nil {
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

func credentialFromAccount(account *ent.User, organizationID uuid.UUID) *biz.Credential {
	return &biz.Credential{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, PasswordHash: account.PasswordHash, Enabled: account.Enabled, PrimaryOrganizationID: organizationID}
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

func (r *authRepo) credentialForAccount(ctx context.Context, account *ent.User) (*biz.Credential, error) {
	memberships, err := r.data.db.Membership.Query().Where(membership.UserIDEQ(account.ID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).WithOrganization().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, biz.ErrOrganizationForbidden
	}
	primaryID := memberships[0].OrganizationID
	for _, member := range memberships {
		if member.Primary {
			primaryID = member.OrganizationID
			break
		}
	}
	return &biz.Credential{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, PasswordHash: account.PasswordHash, Enabled: account.Enabled, PrimaryOrganizationID: primaryID}, nil
}

// ListEnabledMembershipOrganizations 返回用户「启用中成员资格 × 组织启用中」的候选组织快照，
// 供登录组织选择与应用内切换入口共用同一谓词；排序由 biz 统一处理。
func (r *authRepo) ListEnabledMembershipOrganizations(ctx context.Context, userID uuid.UUID) ([]biz.OrganizationChoice, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	memberships, err := client.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		WithOrganization().
		All(ctx)
	if err != nil {
		return nil, err
	}
	choices := make([]biz.OrganizationChoice, 0, len(memberships))
	for _, member := range memberships {
		org, edgeErr := member.Edges.OrganizationOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		choices = append(choices, biz.OrganizationChoice{OrganizationID: org.ID, OrganizationName: org.Name, OrganizationCode: org.Code, IsDefault: member.Primary})
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
	nodes, err := client.Organization.Query().
		Select(organization.FieldID, organization.FieldParentID, organization.FieldBaseCurrency, organization.FieldEnabled).
		All(ctx)
	if err != nil {
		return nil, err
	}
	memberships, err := client.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		WithOrganization().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) {
			query.WithRole(func(roleQuery *ent.RoleQuery) {
				roleQuery.Where(roleent.EnabledEQ(true)).WithPermissions().WithOrganizationAccesses()
			})
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	organizations := make([]biz.Organization, 0, len(memberships))
	roleGrants := make([]biz.RoleGrant, 0)
	organizationsByID := make(map[uuid.UUID]*ent.Organization, len(nodes))
	for _, node := range nodes {
		organizationsByID[node.ID] = node
	}
	var current *biz.Organization
	for _, member := range memberships {
		org, edgeErr := member.Edges.OrganizationOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		baseCurrency, currencyErr := resolvePrincipalOrganizationBaseCurrency(org, organizationsByID)
		if currencyErr != nil {
			return nil, currencyErr
		}
		organizationView := biz.Organization{ID: org.ID, Code: org.Code, Name: org.Name, BaseCurrency: baseCurrency}
		organizations = append(organizations, organizationView)
		if org.ID != organizationID {
			continue
		}
		current = &organizationView
		for _, assignment := range member.Edges.RoleAssignments {
			role, roleErr := assignment.Edges.RoleOrErr()
			if roleErr != nil {
				continue
			}
			grant, enabled := roleGrantFromEntRole(role)
			if !enabled {
				continue
			}
			roleGrants = append(roleGrants, grant)
		}
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
		organizationNodes = append(organizationNodes, biz.OrganizationScopeNode{ID: node.ID, ParentID: node.ParentID, Disabled: !node.Enabled})
	}
	sort.Slice(organizationNodes, func(i, j int) bool {
		return organizationNodes[i].ID.String() < organizationNodes[j].ID.String()
	})
	return &biz.Principal{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, AvatarURL: account.AvatarURL, IsBootstrapAdmin: account.IsBootstrapAdmin, Organization: *current, Organizations: organizations, RoleGrants: roleGrants, OrganizationNodes: organizationNodes}, nil
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
		RoleID:               role.ID,
		RoleCode:             role.Code,
		DataScope:            biz.DataScope(role.DataScope),
		Permissions:          make(map[string]struct{}, len(role.Edges.Permissions)),
		OrganizationAccesses: make([]biz.OrganizationAccess, 0, len(role.Edges.OrganizationAccesses)),
	}
	for _, permission := range role.Edges.Permissions {
		grant.Permissions[permission.Key] = struct{}{}
	}
	for _, access := range role.Edges.OrganizationAccesses {
		grant.OrganizationAccesses = append(grant.OrganizationAccesses, biz.OrganizationAccess{OrganizationID: access.OrganizationID, Writable: access.Writable})
	}
	sort.Slice(grant.OrganizationAccesses, func(i, j int) bool {
		return grant.OrganizationAccesses[i].OrganizationID.String() < grant.OrganizationAccesses[j].OrganizationID.String()
	})
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
	stored, err := r.data.db.Session.Query().Where(sessionent.TokenHashEQ(tokenHash), sessionent.RevokedAtIsNil(), sessionent.ExpiresAtGT(now), sessionent.HasUserWith(user.EnabledEQ(true)), sessionent.HasOrganizationWith(organization.EnabledEQ(true))).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSessionExpired, nil)
	}
	return &biz.Session{TokenHash: stored.TokenHash, UserID: stored.UserID, OrganizationID: stored.OrganizationID, ExpiresAt: stored.ExpiresAt, UserAgent: stored.UserAgent}, nil
}

// RotateSession 在同一事务内完成会话令牌轮转：校验目标成员资格（ForShare）→ 锁定当前
// 会话行（ForUpdate）→ 新建目标组织会话（沿用当前会话的 UA/IP）→ 失效旧令牌 → 写审计。
// 只轮转当前令牌，同一用户其他设备的会话不受影响。
func (r *authRepo) RotateSession(ctx context.Context, tokenHash string, next *biz.Session, now time.Time, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 事务内复核成员资格，防止校验与轮转之间成员资格被停用（TOCTOU）。
		exists, queryErr := tx.Membership.Query().Where(
			membership.UserIDEQ(next.UserID),
			membership.OrganizationIDEQ(next.OrganizationID),
			membership.EnabledEQ(true),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
		).ForShare().Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrAuthOrganizationForbidden
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
