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
		if ent.IsNotFound(err) {
			return nil, biz.ErrInvalidCredentials
		}
		return nil, err
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

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, false, err
	}
	headquarters, err := tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	digest := sha256.Sum256([]byte(wecomUserID))
	username := "wecom_" + hex.EncodeToString(digest[:12])
	create := tx.User.Create().SetUsername(username).SetDisplayName(wecomName).SetWecomUserid(wecomUserID).SetWecomName(wecomName).SetEnabled(false)
	if identity.Email != nil && strings.TrimSpace(*identity.Email) != "" {
		create.SetEmail(strings.TrimSpace(*identity.Email))
	}
	account, err = create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	if _, err := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(headquarters.ID).SetPrimary(true).SetEnabled(true).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	audit.UserID = &account.ID
	audit.OrganizationID = &headquarters.ID
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
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
		if ent.IsNotFound(err) {
			return nil, biz.ErrDingTalkNotRegistered
		}
		return nil, err
	}
	account, err = updateDingTalkProfile(ctx, account, identity)
	if err != nil {
		return nil, err
	}
	return r.credentialForAccount(ctx, account)
}

func (r *authRepo) RegisterDingTalkCredential(ctx context.Context, identity *biz.DingTalkIdentity, audit *biz.AuditEvent) (*biz.Credential, bool, error) {
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
			credential, prepareErr := r.prepareDingTalkRehire(ctx, account, audit)
			return credential, prepareErr == nil, prepareErr
		}
		credential, credentialErr := r.credentialForAccount(ctx, account)
		return credential, false, credentialErr
	}
	if !ent.IsNotFound(err) {
		return nil, false, err
	}

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, false, err
	}
	headquarters, err := tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	create := tx.User.Create().SetDisplayName(dingtalkName).SetDingtalkUnionid(unionID).SetDingtalkUserid(dingTalkUserID).SetDingtalkName(dingtalkName).SetEnabled(false)
	if identity.Email != nil && strings.TrimSpace(*identity.Email) != "" {
		create.SetEmail(strings.TrimSpace(*identity.Email))
	}
	if identity.AvatarURL != nil && strings.TrimSpace(*identity.AvatarURL) != "" {
		create.SetAvatarURL(strings.TrimSpace(*identity.AvatarURL))
	}
	account, err = create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	if _, err := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(headquarters.ID).SetPrimary(true).SetEnabled(true).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	audit.UserID = &account.ID
	audit.OrganizationID = &headquarters.ID
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return credentialFromAccount(account, headquarters.ID), true, nil
}

func (r *authRepo) prepareDingTalkRehire(ctx context.Context, account *ent.User, audit *biz.AuditEvent) (*biz.Credential, error) {
	userID := account.ID
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	headquarters, err := tx.Organization.Query().Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	membershipIDs, err := tx.Membership.Query().Where(membership.UserIDEQ(userID)).IDs(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if len(membershipIDs) > 0 {
		if _, err := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if _, err := tx.Membership.Update().Where(membership.UserIDEQ(userID)).SetEnabled(false).SetPrimary(false).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	intake, err := tx.Membership.Query().Where(membership.UserIDEQ(userID), membership.OrganizationIDEQ(headquarters.ID)).Only(ctx)
	if ent.IsNotFound(err) {
		_, err = tx.Membership.Create().SetUserID(userID).SetOrganizationID(headquarters.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
	} else if err == nil {
		_, err = tx.Membership.UpdateOneID(intake.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.User.UpdateOneID(userID).SetEnabled(false).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Session.Update().Where(sessionent.UserIDEQ(userID), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.UserID = &userID
	audit.OrganizationID = &headquarters.ID
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
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

func (r *authRepo) ResolvePrincipal(ctx context.Context, userID, organizationID uuid.UUID) (*biz.Principal, error) {
	account, err := r.data.db.User.Query().Where(user.IDEQ(userID), user.EnabledEQ(true)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSessionExpired
		}
		return nil, err
	}
	memberships, err := r.data.db.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		WithOrganization().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) {
			query.WithRole(func(roleQuery *ent.RoleQuery) { roleQuery.WithPermissions().WithOrderOrganizationAccesses() })
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	organizations := make([]biz.Organization, 0, len(memberships))
	permissionSet := make(map[string]struct{})
	rolePermissions := make(map[string]map[string]struct{})
	roleScopes := make([]biz.RoleScope, 0)
	orderOrganizationAccesses := make(map[uuid.UUID]bool)
	var current *biz.Organization
	for _, member := range memberships {
		org, edgeErr := member.Edges.OrganizationOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		baseCurrency, currencyErr := resolveOrganizationBaseCurrency(ctx, r.data.db.Organization, org)
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
			if roleErr != nil || !role.Enabled {
				continue
			}
			roleScopes = append(roleScopes, biz.RoleScope{RoleCode: role.Code, DataScope: biz.DataScope(role.DataScope)})
			rolePermissionSet := make(map[string]struct{}, len(role.Edges.Permissions))
			for _, permission := range role.Edges.Permissions {
				permissionSet[permission.Key] = struct{}{}
				rolePermissionSet[permission.Key] = struct{}{}
			}
			rolePermissions[role.Code] = rolePermissionSet
			for _, access := range role.Edges.OrderOrganizationAccesses {
				orderOrganizationAccesses[access.OrganizationID] = orderOrganizationAccesses[access.OrganizationID] || access.Writable
			}
		}
	}
	if current == nil {
		return nil, biz.ErrOrganizationForbidden
	}
	sort.Slice(organizations, func(i, j int) bool { return organizations[i].Code < organizations[j].Code })
	permissions := make([]string, 0, len(permissionSet))
	for key := range permissionSet {
		permissions = append(permissions, key)
	}
	sort.Strings(permissions)
	sort.Slice(roleScopes, func(i, j int) bool { return roleScopes[i].RoleCode < roleScopes[j].RoleCode })
	accesses := make([]biz.OrderOrganizationAccess, 0, len(orderOrganizationAccesses))
	for organizationID, writable := range orderOrganizationAccesses {
		accesses = append(accesses, biz.OrderOrganizationAccess{OrganizationID: organizationID, Writable: writable})
	}
	sort.Slice(accesses, func(i, j int) bool { return accesses[i].OrganizationID.String() < accesses[j].OrganizationID.String() })
	return &biz.Principal{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, AvatarURL: account.AvatarURL, Organization: *current, Organizations: organizations, Permissions: permissions, RoleScopes: roleScopes, RolePermissions: rolePermissions, OrderOrganizationAccesses: accesses}, nil
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
		if ent.IsNotFound(err) {
			return nil, biz.ErrSessionExpired
		}
		return nil, err
	}
	return &biz.Session{TokenHash: stored.TokenHash, UserID: stored.UserID, OrganizationID: stored.OrganizationID, ExpiresAt: stored.ExpiresAt, UserAgent: stored.UserAgent}, nil
}

func (r *authRepo) SwitchSessionOrganization(ctx context.Context, tokenHash string, userID, organizationID uuid.UUID, now time.Time, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		exists, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(userID), membership.OrganizationIDEQ(organizationID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrOrganizationForbidden
		}
		updated, updateErr := tx.Session.Update().Where(sessionent.TokenHashEQ(tokenHash), sessionent.UserIDEQ(userID), sessionent.RevokedAtIsNil(), sessionent.ExpiresAtGT(now)).SetOrganizationID(organizationID).Save(ctx)
		if updateErr != nil {
			return updateErr
		}
		if updated != 1 {
			return biz.ErrSessionExpired
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
