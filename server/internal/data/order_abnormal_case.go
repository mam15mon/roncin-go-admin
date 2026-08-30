package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
)

type orderAbnormalCaseRepo struct {
	data *Data
}

func NewOrderAbnormalCaseRepo(data *Data) biz.OrderAbnormalCaseRepo {
	return &orderAbnormalCaseRepo{data: data}
}

func (r *orderAbnormalCaseRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	if _, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx); err != nil {
		return mapEntError(err, biz.ErrOrderAbnormalCaseNotFound, nil)
	}
	return nil
}

func (r *orderAbnormalCaseRepo) validateAbnormalCaseKind(ctx context.Context, organizationID, abnormalCaseID uuid.UUID) error {
	headquartersID, err := resolveHeadquartersOrganizationID(ctx, r.data.db.Organization, organizationID)
	if err != nil {
		return err
	}
	count, err := r.data.db.MasterDataItem.Query().
		Where(
			masterdataitement.IDEQ(abnormalCaseID),
			masterdataitement.OrganizationIDEQ(headquartersID),
			masterdataitement.KindEQ(masterdataitement.KindAbnormalCase),
			masterdataitement.EnabledEQ(true),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return biz.ErrOrderAbnormalCaseKindInvalid
	}
	return nil
}

func (r *orderAbnormalCaseRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderAbnormalCase, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderAbnormalCase.Query().
		Where(orderabnormalcaseent.OrderIDEQ(orderID)).
		Order(orderabnormalcaseent.ByMarkedAt(), orderabnormalcaseent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderAbnormalCase, 0, len(items))
	for _, item := range items {
		result = append(result, orderAbnormalCaseToBiz(item))
	}
	return result, nil
}

func (r *orderAbnormalCaseRepo) Mark(ctx context.Context, organizationID, orderID, actorID, abnormalCaseID uuid.UUID, newID uuid.UUID, audit *biz.AuditEvent) (*biz.OrderAbnormalCase, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	if err := r.validateAbnormalCaseKind(ctx, organizationID, abnormalCaseID); err != nil {
		return nil, err
	}
	var saved *ent.OrderAbnormalCase
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderAbnormalCase.Query().
			Where(
				orderabnormalcaseent.OrderIDEQ(orderID),
				orderabnormalcaseent.AbnormalCaseIDEQ(abnormalCaseID),
			).
			ForUpdate().
			Only(ctx)
		if ent.IsNotFound(queryErr) {
			now := time.Now()
			var saveErr error
			saved, saveErr = tx.OrderAbnormalCase.Create().
				SetID(newID).
				SetOrderID(orderID).
				SetAbnormalCaseID(abnormalCaseID).
				SetStatus(orderabnormalcaseent.StatusACTIVE).
				SetMarkedAt(now).
				SetMarkedBy(actorID).
				Save(ctx)
			if saveErr != nil {
				return mapEntConstraint(saveErr, "orderabnormalcase_order_id_abnormal_case_id", biz.ErrOrderAbnormalCaseExists)
			}
		} else {
			if queryErr != nil {
				return queryErr
			}
			if item.Status == orderabnormalcaseent.StatusACTIVE {
				return biz.ErrOrderAbnormalCaseExists
			}
			now := time.Now()
			var saveErr error
			saved, saveErr = tx.OrderAbnormalCase.UpdateOne(item).
				SetStatus(orderabnormalcaseent.StatusACTIVE).
				SetMarkedAt(now).
				SetMarkedBy(actorID).
				ClearResolvedAt().
				ClearResolvedBy().
				Save(ctx)
			if saveErr != nil {
				return saveErr
			}
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["abnormal_case_record.id"] = saved.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderAbnormalCaseToBiz(saved), nil
}

func (r *orderAbnormalCaseRepo) Resolve(ctx context.Context, organizationID, orderID, actorID, id uuid.UUID, audit *biz.AuditEvent) (*biz.OrderAbnormalCase, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderAbnormalCase
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.OrderAbnormalCase.Query().
			Where(
				orderabnormalcaseent.IDEQ(id),
				orderabnormalcaseent.OrderIDEQ(orderID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderAbnormalCaseNotFound, nil)
		}
		if item.Status != orderabnormalcaseent.StatusACTIVE {
			return biz.ErrOrderAbnormalCaseStatusConflict
		}
		now := time.Now()
		var saveErr error
		updated, saveErr = tx.OrderAbnormalCase.UpdateOne(item).
			SetStatus(orderabnormalcaseent.StatusRESOLVED).
			SetResolvedAt(now).
			SetResolvedBy(actorID).
			Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return orderAbnormalCaseToBiz(updated), nil
}

func (r *orderAbnormalCaseRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		n, deleteErr := tx.OrderAbnormalCase.Delete().
			Where(
				orderabnormalcaseent.IDEQ(id),
				orderabnormalcaseent.OrderIDEQ(orderID),
			).
			Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		if n == 0 {
			return biz.ErrOrderAbnormalCaseNotFound
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func orderAbnormalCaseToBiz(item *ent.OrderAbnormalCase) *biz.OrderAbnormalCase {
	if item == nil {
		return nil
	}
	return &biz.OrderAbnormalCase{
		ID:             item.ID,
		OrderID:        item.OrderID,
		AbnormalCaseID: item.AbnormalCaseID,
		Status:         biz.OrderAbnormalCaseStatus(item.Status),
		MarkedAt:       item.MarkedAt,
		MarkedBy:       item.MarkedBy,
		ResolvedAt:     item.ResolvedAt,
		ResolvedBy:     item.ResolvedBy,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

var _ biz.OrderAbnormalCaseRepo = (*orderAbnormalCaseRepo)(nil)
