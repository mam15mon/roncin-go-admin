package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type orderPersonnelRepo struct {
	data *Data
}

func NewOrderPersonnelRepo(data *Data) biz.OrderPersonnelRepo {
	return &orderPersonnelRepo{data: data}
}

func (r *orderPersonnelRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	if _, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		return mapEntError(err, biz.ErrOrderPersonnelNotFound, nil)
	}
	return nil
}

func (r *orderPersonnelRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderPersonnel, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderPersonnel.Query().
		Where(orderpersonnelent.OrderIDEQ(orderID)).
		Order(orderpersonnelent.ByAssignedAt(), orderpersonnelent.ByRole()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderPersonnel, 0, len(items))
	for _, item := range items {
		result = append(result, orderPersonnelToBiz(item))
	}
	return result, nil
}

func (r *orderPersonnelRepo) Assign(ctx context.Context, organizationID, orderID, userID uuid.UUID, role biz.OrderPersonnelRole, notification *biz.NotificationIntent, audit *biz.AuditEvent) (*biz.OrderPersonnel, error) {
	var created *ent.OrderPersonnel
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		orderRecord, queryErr := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderPersonnelNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, orderRecord); err != nil {
			return err
		}
		organizations, queryErr := tx.Organization.Query().Select(organizationent.FieldID, organizationent.FieldParentID, organizationent.FieldEnabled).All(ctx)
		if queryErr != nil {
			return queryErr
		}
		parentByID := make(map[uuid.UUID]*uuid.UUID, len(organizations))
		for _, organization := range organizations {
			parentByID[organization.ID] = organization.ParentID
		}
		subtreeOrganizationIDs := make([]uuid.UUID, 0, len(organizations))
		for _, organization := range organizations {
			if organization.Enabled && organizationWithinRoot(parentByID, organizationID, organization.ID) {
				subtreeOrganizationIDs = append(subtreeOrganizationIDs, organization.ID)
			}
		}
		user, queryErr := tx.User.Query().
			Where(
				userent.IDEQ(userID),
				userent.EnabledEQ(true),
				userent.HasMembershipsWith(
					membershipent.OrganizationIDIn(subtreeOrganizationIDs...),
					membershipent.EnabledEQ(true),
				),
			).
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderPersonnelUserInvalid, nil)
		}
		var saveErr error
		created, saveErr = tx.OrderPersonnel.Create().
			SetOrderID(orderID).
			SetUserID(userID).
			SetOrganizationID(organizationID).
			SetRole(orderpersonnelent.Role(role)).
			Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "order_personnel_order_id_role", biz.ErrOrderPersonnelExists)
		}
		if notificationErr := enqueueOrderPersonnelNotification(ctx, tx, organizationID, orderID, orderRecord.OrderNo, role, user, notification); notificationErr != nil {
			return notificationErr
		}
		audit.Details["personnel.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderPersonnelToBiz(created), nil
}

func (r *orderPersonnelRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		orderRecord, queryErr := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderPersonnelNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, orderRecord); err != nil {
			return err
		}
		n, deleteErr := tx.OrderPersonnel.Delete().
			Where(
				orderpersonnelent.IDEQ(id),
				orderpersonnelent.OrderIDEQ(orderID),
			).
			Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if n == 0 {
			return biz.ErrOrderPersonnelNotFound
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderPersonnelToBiz(item *ent.OrderPersonnel) *biz.OrderPersonnel {
	return &biz.OrderPersonnel{
		ID:             item.ID,
		OrderID:        item.OrderID,
		UserID:         item.UserID,
		OrganizationID: item.OrganizationID,
		Role:           biz.OrderPersonnelRole(item.Role),
		AssignedAt:     item.AssignedAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

var _ biz.OrderPersonnelRepo = (*orderPersonnelRepo)(nil)
