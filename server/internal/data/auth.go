package data

import (
	"context"
	"sort"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

type authRepo struct{ data *Data }

func NewAuthRepo(data *Data) biz.AuthRepo { return &authRepo{data: data} }

func NewAuditRepo(data *Data) biz.AuditRepo { return &authRepo{data: data} }

func (r *authRepo) FindCredential(ctx context.Context, username string) (*biz.Credential, error) {
	account, err := r.data.db.User.Query().Where(user.UsernameEQ(username), user.EnabledEQ(true)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrInvalidCredentials
		}
		return nil, err
	}
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
	return &biz.Credential{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, PasswordHash: account.PasswordHash, PrimaryOrganizationID: primaryID}, nil
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
			query.WithRole(func(roleQuery *ent.RoleQuery) { roleQuery.WithPermissions() })
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}
	organizations := make([]biz.Organization, 0, len(memberships))
	permissionSet := make(map[string]struct{})
	rolePermissions := make(map[string]map[string]struct{})
	roleScopes := make([]biz.RoleScope, 0)
	var current *biz.Organization
	for _, member := range memberships {
		org, edgeErr := member.Edges.OrganizationOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		organizationView := biz.Organization{ID: org.ID, Code: org.Code, Name: org.Name}
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
	return &biz.Principal{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, Organization: *current, Organizations: organizations, Permissions: permissions, RoleScopes: roleScopes, RolePermissions: rolePermissions}, nil
}

func (r *authRepo) CreateSession(ctx context.Context, input *biz.Session) error {
	_, err := r.data.db.Session.Create().SetTokenHash(input.TokenHash).SetUserID(input.UserID).SetOrganizationID(input.OrganizationID).SetExpiresAt(input.ExpiresAt).SetUserAgent(input.UserAgent).Save(ctx)
	return err
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

func (r *authRepo) SwitchSessionOrganization(ctx context.Context, tokenHash string, userID, organizationID uuid.UUID, now time.Time) error {
	exists, err := r.data.db.Membership.Query().Where(membership.UserIDEQ(userID), membership.OrganizationIDEQ(organizationID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrganizationForbidden
	}
	updated, err := r.data.db.Session.Update().Where(sessionent.TokenHashEQ(tokenHash), sessionent.UserIDEQ(userID), sessionent.RevokedAtIsNil(), sessionent.ExpiresAtGT(now)).SetOrganizationID(organizationID).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return biz.ErrSessionExpired
	}
	return nil
}

func (r *authRepo) RevokeSession(ctx context.Context, tokenHash string, now time.Time) error {
	updated, err := r.data.db.Session.Update().Where(sessionent.TokenHashEQ(tokenHash), sessionent.RevokedAtIsNil()).SetRevokedAt(now).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return biz.ErrSessionExpired
	}
	return nil
}

func (r *authRepo) WriteAudit(ctx context.Context, event *biz.AuditEvent) error {
	return writeAudit(ctx, r.data.db.AuditLog, event)
}

var _ biz.AuthRepo = (*authRepo)(nil)
