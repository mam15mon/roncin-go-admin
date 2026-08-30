package data

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

func (r *adminRepo) ListUsers(ctx context.Context, organizationID uuid.UUID, options biz.AdminUserListOptions) (*biz.AdminUserList, error) {
	predicates := []predicate.Membership{
		membership.OrganizationIDEQ(organizationID),
	}
	if options.Keyword != "" {
		predicates = append(predicates, membership.HasUserWith(userent.Or(
			userent.UsernameContainsFold(options.Keyword),
			userent.DisplayNameContainsFold(options.Keyword),
			userent.SearchKeywordsContainsFold(options.Keyword),
		)))
	}
	query := r.data.db.Membership.Query().
		Where(predicates...).
		WithUser(func(query *ent.UserQuery) {
			query.WithMemberships(func(query *ent.MembershipQuery) {
				query.Where(membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true)))
			})
		}).
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() })
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.Membership, error) {
		return query.Order(membership.ByUserField(userent.FieldUsername), membership.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(membershipToUser))
}

func (r *adminRepo) CreateUser(ctx context.Context, organizationID uuid.UUID, input *biz.AdminUser, passwordHash string, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	var account *ent.User
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		roles, queryErr := rolesForOrganization(ctx, tx.Role.Query(), organizationID, roleIDs)
		if queryErr != nil {
			return queryErr
		}
		create := tx.User.Create().SetUsername(input.Username).SetDisplayName(input.DisplayName).SetPasswordHash(passwordHash).SetEnabled(input.Enabled)
		if input.Email != nil {
			create.SetEmail(*input.Email)
		}
		var createErr error
		account, createErr = create.Save(ctx)
		if createErr != nil {
			return mapEntError(createErr, nil, biz.ErrAdminUsernameExists)
		}
		membershipRecord, createErr := tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(organizationID).SetPrimary(true).SetEnabled(true).Save(ctx)
		if createErr != nil {
			return createErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, membershipRecord.ID, roles); replaceErr != nil {
			return replaceErr
		}
		audit.Details["resource_id"] = account.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUser(ctx, organizationID, account.ID)
}

func (r *adminRepo) UpdateUser(ctx context.Context, organizationID, id uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := tx.User.Query().Where(userent.IDEQ(id)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		membershipRecord, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(id), membership.OrganizationIDEQ(organizationID), membership.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		if account.Enabled && !input.Enabled {
			return biz.ErrAdminUserTerminationRequired
		}
		if !account.Enabled && input.Enabled && (account.WecomUserid != nil || account.DingtalkUnionid != nil) {
			return biz.ErrAdminUserAuthorizationRequired
		}
		roles, queryErr := rolesForOrganization(ctx, tx.Role.Query(), organizationID, roleIDs)
		if queryErr != nil {
			return queryErr
		}
		update := tx.User.UpdateOneID(id).SetDisplayName(input.DisplayName).SetEnabled(input.Enabled)
		if input.Email == nil {
			update.ClearEmail()
		} else {
			update.SetEmail(*input.Email)
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return mapEntError(updateErr, biz.ErrAdminUserNotFound, nil)
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, membershipRecord.ID, roles); replaceErr != nil {
			return replaceErr
		}
		audit.Details["value"] = account.Username
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUser(ctx, organizationID, id)
}
func (r *adminRepo) TerminateUser(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, queryErr := tx.User.Query().Where(userent.IDEQ(id)).ForUpdate().Only(ctx); queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		if _, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(id), membership.OrganizationIDEQ(organizationID), membership.EnabledEQ(true)).Only(ctx); queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(id)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(id)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, updateErr := tx.User.UpdateOneID(id).SetEnabled(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(id), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *adminRepo) AuthorizeWeComUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	return r.authorizePendingUser(ctx, sourceOrganizationID, targetOrganizationID, input, roleIDs, nil, audit, func(account *ent.User) bool {
		return account.WecomUserid != nil
	})
}

func (r *adminRepo) AuthorizeDingTalkUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, notification *biz.NotificationIntent, audit *biz.AuditEvent) (*biz.AdminUser, error) {
	return r.authorizePendingUser(ctx, sourceOrganizationID, targetOrganizationID, input, roleIDs, notification, audit, func(account *ent.User) bool {
		return account.DingtalkUnionid != nil && account.DingtalkUserid != nil && strings.TrimSpace(*account.DingtalkUserid) != ""
	})
}

func (r *adminRepo) authorizePendingUser(ctx context.Context, sourceOrganizationID, targetOrganizationID uuid.UUID, input *biz.AdminUser, roleIDs []uuid.UUID, notification *biz.NotificationIntent, audit *biz.AuditEvent, hasExternalIdentity func(*ent.User) bool) (*biz.AdminUser, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := tx.User.Query().Where(userent.IDEQ(input.ID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		if _, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(input.ID), membership.OrganizationIDEQ(sourceOrganizationID), membership.EnabledEQ(true)).Only(ctx); queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		if !hasExternalIdentity(account) || account.Enabled {
			return biz.ErrAdminInvalidArgument
		}
		exists, queryErr := tx.Organization.Query().Where(organization.IDEQ(targetOrganizationID), organization.EnabledEQ(true)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrAdminOrganizationNotFound
		}
		roles, queryErr := rolesForOrganization(ctx, tx.Role.Query(), targetOrganizationID, roleIDs)
		if queryErr != nil {
			return queryErr
		}
		userUpdate := tx.User.UpdateOneID(input.ID).SetDisplayName(input.DisplayName).SetEnabled(true)
		if input.Email == nil {
			userUpdate.ClearEmail()
		} else {
			userUpdate.SetEmail(*input.Email)
		}
		if _, updateErr := userUpdate.Save(ctx); updateErr != nil {
			return updateErr
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(input.ID)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(input.ID)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		targetMembership, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(input.ID), membership.OrganizationIDEQ(targetOrganizationID)).Only(ctx)
		if ent.IsNotFound(queryErr) {
			targetMembership, queryErr = tx.Membership.Create().SetUserID(input.ID).SetOrganizationID(targetOrganizationID).SetEnabled(true).SetPrimary(true).Save(ctx)
		} else if queryErr == nil {
			targetMembership, queryErr = tx.Membership.UpdateOneID(targetMembership.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
		}
		if queryErr != nil {
			return queryErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, targetMembership.ID, roles); replaceErr != nil {
			return replaceErr
		}
		if notification != nil {
			if notificationErr := enqueueDingTalkUserAuthorizedNotification(ctx, tx, targetOrganizationID, account, notification); notificationErr != nil {
				return notificationErr
			}
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(input.ID), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		audit.Details["value"] = account.Username
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findUser(ctx, targetOrganizationID, input.ID)
}

func (r *adminRepo) ResetUserPassword(ctx context.Context, organizationID, id uuid.UUID, passwordHash string, username *string, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		targetUser, queryErr := tx.User.Query().
			Where(userent.IDEQ(id), userent.EnabledEQ(true)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminUserNotFound, nil)
		}
		exists, queryErr := tx.Membership.Query().Where(
			membership.UserIDEQ(id),
			membership.OrganizationIDEQ(organizationID),
			membership.EnabledEQ(true),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
		).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrAdminUserNotFound
		}
		userUpdate := tx.User.UpdateOneID(id).SetPasswordHash(passwordHash)
		if username != nil && strings.TrimSpace(*username) != "" {
			userUpdate.SetUsername(strings.TrimSpace(*username))
		} else if targetUser.Username == "" {
			return biz.ErrAdminInvalidArgument
		}
		if _, updateErr := userUpdate.Save(ctx); updateErr != nil {
			return mapEntError(updateErr, biz.ErrAdminUserNotFound, biz.ErrAdminUsernameExists)
		}
		if _, updateErr := tx.Session.Update().Where(sessionent.UserIDEQ(id), sessionent.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}
func (r *adminRepo) findUser(ctx context.Context, organizationID, userID uuid.UUID) (*biz.AdminUser, error) {
	item, err := r.data.db.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.OrganizationIDEQ(organizationID)).
		WithUser(func(query *ent.UserQuery) {
			query.WithMemberships(func(query *ent.MembershipQuery) {
				query.Where(membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true)))
			})
		}).
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() }).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrAdminUserNotFound, nil)
	}
	return membershipToUser(item), nil
}
func membershipToUser(item *ent.Membership) *biz.AdminUser {
	account := item.Edges.User
	result := &biz.AdminUser{ID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Email: account.Email, AvatarURL: account.AvatarURL, WeComUserID: account.WecomUserid, WeComName: account.WecomName, DingTalkUnionID: account.DingtalkUnionid, DingTalkUserID: account.DingtalkUserid, DingTalkName: account.DingtalkName, Enabled: account.Enabled, CurrentMembershipEnabled: item.Enabled, HasPassword: account.PasswordHash != nil, CreatedAt: account.CreatedAt, UpdatedAt: account.UpdatedAt}
	for _, assignment := range item.Edges.RoleAssignments {
		if assignedRole := assignment.Edges.Role; assignedRole != nil {
			result.RoleIDs = append(result.RoleIDs, assignedRole.ID)
			result.RoleCodes = append(result.RoleCodes, assignedRole.Code)
		}
	}
	sort.Strings(result.RoleCodes)
	result.Status = adminUserStatus(account, item, len(account.Edges.Memberships) > 0)
	return result
}

func adminUserStatus(account *ent.User, currentMembership *ent.Membership, hasActiveMembership bool) biz.AdminUserStatus {
	if !currentMembership.Enabled && hasActiveMembership {
		return biz.AdminUserStatusRemovedFromOrganization
	}
	if !account.Enabled && !hasActiveMembership {
		return biz.AdminUserStatusTerminated
	}
	if account.Enabled && currentMembership.Enabled {
		return biz.AdminUserStatusActive
	}
	hasExternalIdentity := account.WecomUserid != nil || account.DingtalkUnionid != nil
	if !account.Enabled && currentMembership.Enabled && hasExternalIdentity && len(currentMembership.Edges.RoleAssignments) == 0 {
		return biz.AdminUserStatusPendingAuthorization
	}
	return biz.AdminUserStatusDisabled
}
