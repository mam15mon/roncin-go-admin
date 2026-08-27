package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordermilestoneent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordermilestone"
)

type orderMilestoneRepo struct{ data *Data }

func NewOrderMilestoneRepo(data *Data) biz.OrderMilestoneRepo { return &orderMilestoneRepo{data: data} }

func (r *orderMilestoneRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderMilestone, error) {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := r.data.db.OrderMilestone.Query().Where(ordermilestoneent.OrderIDEQ(orderID)).Order(ordermilestoneent.ByOccurredAt(), ordermilestoneent.ByType()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderMilestone, 0, len(items))
	for _, item := range items {
		result = append(result, orderMilestoneToBiz(item))
	}
	return result, nil
}

func (r *orderMilestoneRepo) Set(ctx context.Context, organizationID, orderID uuid.UUID, milestoneType string, expectedVersion uint64, occurredAt *time.Time, note *string, clearOccurredAt bool, actorID uuid.UUID) (*biz.OrderMilestone, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	order, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	if order.Version != expectedVersion {
		_ = tx.Rollback()
		return nil, biz.ErrOrderStatusConflict
	}
	existing, err := tx.OrderMilestone.Query().Where(ordermilestoneent.OrderIDEQ(orderID), ordermilestoneent.TypeEQ(milestoneType)).Only(ctx)
	if ent.IsNotFound(err) {
		create := tx.OrderMilestone.Create().SetOrderID(orderID).SetType(milestoneType).SetNillableOccurredAt(occurredAt).SetNillableNote(note).SetUpdatedBy(actorID)
		created, saveErr := create.Save(ctx)
		if saveErr != nil {
			_ = tx.Rollback()
			return nil, saveErr
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return orderMilestoneToBiz(created), nil
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	update := existing.Update().SetNillableNote(note).SetUpdatedBy(actorID)
	if clearOccurredAt {
		update.ClearOccurredAt()
	} else if occurredAt != nil {
		update.SetOccurredAt(*occurredAt)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderMilestoneToBiz(updated), nil
}

func orderMilestoneToBiz(item *ent.OrderMilestone) *biz.OrderMilestone {
	result := &biz.OrderMilestone{ID: item.ID, OrderID: item.OrderID, Type: item.Type, OccurredAt: item.OccurredAt, Note: item.Note, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.UpdatedBy != nil {
		value := *item.UpdatedBy
		result.UpdatedBy = &value
	}
	if item.TemplateNodeCode != nil {
		value := *item.TemplateNodeCode
		result.TemplateNodeCode = &value
	}
	if item.TemplateNodeLabel != nil {
		value := *item.TemplateNodeLabel
		result.TemplateNodeLabel = &value
	}
	return result
}

var _ biz.OrderMilestoneRepo = (*orderMilestoneRepo)(nil)
