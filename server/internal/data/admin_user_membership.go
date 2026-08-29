package data

import (
	"context"
	"sort"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	sessionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	"github.com/google/uuid"
)

func (r *adminRepo) ListUserMemberships(ctx context.Context, userID uuid.UUID) ([]*biz.AdminUserMembership, error) {
	exists, err := r.data.db.User.Query().Where(userent.IDEQ(userID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrAdminUserNotFound
	}
	items, err := r.data.db.Membership.Query().
		Where(membership.UserIDEQ(userID)).
		WithOrganization().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() }).
		Order(membership.ByOrganizationField(organization.FieldCode), membership.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AdminUserMembership, 0, len(items))
	for _, item := range items {
		result = append(result, membershipToBiz(item))
	}
	return result, nil
}

func (r *adminRepo) GetUserMembership(ctx context.Context, userID, membershipID uuid.UUID) (*biz.AdminUserMembership, error) {
	return r.findUserMembership(ctx, userID, membershipID)
}

func (r *adminRepo) CreateUserMembership(ctx context.Context, input *biz.AdminUserMembership, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUserMembership, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if _, queryErr := tx.User.Query().Where(userent.IDEQ(input.UserID)).ForUpdate().Only(ctx); queryErr != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(queryErr) {
			return nil, biz.ErrAdminUserNotFound
		}
		return nil, queryErr
	}
	if exists, queryErr := tx.Organization.Query().Where(organization.IDEQ(input.OrganizationID), organization.EnabledEQ(true)).Exist(ctx); queryErr != nil {
		_ = tx.Rollback()
		return nil, queryErr
	} else if !exists {
		_ = tx.Rollback()
		return nil, biz.ErrAdminOrganizationNotFound
	}
	roles, err := rolesForOrganization(ctx, tx.Role.Query(), input.OrganizationID, roleIDs)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	created, err := tx.Membership.Query().
		Where(membership.UserIDEQ(input.UserID), membership.OrganizationIDEQ(input.OrganizationID)).
		Only(ctx)
	if err == nil {
		if created.Enabled {
			_ = tx.Rollback()
			return nil, biz.ErrAdminUserMembershipExists
		}
		created, err = tx.Membership.UpdateOneID(created.ID).SetEnabled(true).SetPrimary(input.Primary).Save(ctx)
	} else if ent.IsNotFound(err) {
		created, err = tx.Membership.Create().
			SetUserID(input.UserID).
			SetOrganizationID(input.OrganizationID).
			SetEnabled(true).
			SetPrimary(input.Primary).
			Save(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replaceRoleAssignments(ctx, tx, created.ID, roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	var preferredID *uuid.UUID
	if input.Primary {
		preferredID = &created.ID
	}
	if err := normalizePrimaryMembership(ctx, tx, input.UserID, preferredID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findUserMembership(ctx, input.UserID, created.ID)
}

func (r *adminRepo) UpdateUserMembership(ctx context.Context, input *biz.AdminUserMembership, roleIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.AdminUserMembership, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	account, err := tx.User.Query().Where(userent.IDEQ(input.UserID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserNotFound
		}
		return nil, err
	}
	current, err := tx.Membership.Query().
		Where(membership.IDEQ(input.ID), membership.UserIDEQ(input.UserID)).
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserMembershipNotFound
		}
		return nil, err
	}
	if account.Enabled && current.Enabled && !input.Enabled {
		activeCount, countErr := tx.Membership.Query().Where(membership.UserIDEQ(input.UserID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Count(ctx)
		if countErr != nil {
			_ = tx.Rollback()
			return nil, countErr
		}
		if activeCount <= 1 {
			_ = tx.Rollback()
			return nil, biz.ErrAdminUserLastMembership
		}
	}
	roles, err := rolesForOrganization(ctx, tx.Role.Query(), current.OrganizationID, roleIDs)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Membership.UpdateOneID(current.ID).SetEnabled(input.Enabled).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replaceRoleAssignments(ctx, tx, current.ID, roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	var preferredID *uuid.UUID
	if input.Primary {
		preferredID = &current.ID
	}
	if err := normalizePrimaryMembership(ctx, tx, input.UserID, preferredID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if !input.Enabled {
		if _, err := tx.Session.Update().
			Where(sessionent.UserIDEQ(input.UserID), sessionent.OrganizationIDEQ(current.OrganizationID), sessionent.RevokedAtIsNil()).
			SetRevokedAt(time.Now().UTC()).
			Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.findUserMembership(ctx, input.UserID, current.ID)
}

func (r *adminRepo) DeleteUserMembership(ctx context.Context, userID, membershipID uuid.UUID, audit *biz.AuditEvent) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	account, err := tx.User.Query().Where(userent.IDEQ(userID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return biz.ErrAdminUserNotFound
		}
		return err
	}
	current, err := tx.Membership.Query().
		Where(membership.IDEQ(membershipID), membership.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return biz.ErrAdminUserMembershipNotFound
		}
		return err
	}
	if account.Enabled && current.Enabled {
		activeCount, countErr := tx.Membership.Query().Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Count(ctx)
		if countErr != nil {
			_ = tx.Rollback()
			return countErr
		}
		if activeCount <= 1 {
			_ = tx.Rollback()
			return biz.ErrAdminUserLastMembership
		}
	}
	if _, err := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDEQ(current.ID)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Membership.UpdateOneID(current.ID).SetEnabled(false).SetPrimary(false).Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := normalizePrimaryMembership(ctx, tx, userID, nil); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Session.Update().
		Where(sessionent.UserIDEQ(userID), sessionent.OrganizationIDEQ(current.OrganizationID), sessionent.RevokedAtIsNil()).
		SetRevokedAt(time.Now().UTC()).
		Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
func (r *adminRepo) findUserMembership(ctx context.Context, userID, membershipID uuid.UUID) (*biz.AdminUserMembership, error) {
	item, err := r.data.db.Membership.Query().
		Where(membership.IDEQ(membershipID), membership.UserIDEQ(userID)).
		WithOrganization().
		WithRoleAssignments(func(query *ent.RoleAssignmentQuery) { query.WithRole() }).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminUserMembershipNotFound
		}
		return nil, err
	}
	return membershipToBiz(item), nil
}

func normalizePrimaryMembership(ctx context.Context, tx *ent.Tx, userID uuid.UUID, preferredID *uuid.UUID) error {
	items, err := tx.Membership.Query().
		Where(membership.UserIDEQ(userID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).
		Order(membership.ByCreatedAt(), membership.ByID()).
		All(ctx)
	if err != nil {
		return err
	}
	var selectedID *uuid.UUID
	if preferredID != nil {
		for _, item := range items {
			if item.ID == *preferredID {
				id := item.ID
				selectedID = &id
				break
			}
		}
	}
	if selectedID == nil {
		for _, item := range items {
			if item.Primary {
				id := item.ID
				selectedID = &id
				break
			}
		}
	}
	if selectedID == nil && len(items) > 0 {
		id := items[0].ID
		selectedID = &id
	}
	if _, err := tx.Membership.Update().Where(membership.UserIDEQ(userID), membership.PrimaryEQ(true)).SetPrimary(false).Save(ctx); err != nil {
		return err
	}
	if selectedID == nil {
		return nil
	}
	_, err = tx.Membership.UpdateOneID(*selectedID).SetPrimary(true).Save(ctx)
	return err
}
func membershipToBiz(item *ent.Membership) *biz.AdminUserMembership {
	organizationRecord := item.Edges.Organization
	result := &biz.AdminUserMembership{
		ID:               item.ID,
		UserID:           item.UserID,
		OrganizationID:   item.OrganizationID,
		OrganizationCode: organizationRecord.Code,
		OrganizationName: organizationRecord.Name,
		OrganizationKind: biz.OrganizationKind(organizationRecord.Kind),
		Primary:          item.Primary,
		Enabled:          item.Enabled,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
	for _, assignment := range item.Edges.RoleAssignments {
		if assignedRole := assignment.Edges.Role; assignedRole != nil {
			result.RoleIDs = append(result.RoleIDs, assignedRole.ID)
			result.RoleCodes = append(result.RoleCodes, assignedRole.Code)
			result.RoleNames = append(result.RoleNames, assignedRole.Name)
		}
	}
	sort.Strings(result.RoleCodes)
	sort.Strings(result.RoleNames)
	return result
}
