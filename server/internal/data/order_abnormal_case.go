package data

import (
	"context"
	"strings"
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
		if ent.IsNotFound(err) {
			return biz.ErrOrderAbnormalCaseNotFound
		}
		return err
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
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderAbnormalCase.Query().
		Where(
			orderabnormalcaseent.OrderIDEQ(orderID),
			orderabnormalcaseent.AbnormalCaseIDEQ(abnormalCaseID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			_ = tx.Rollback()
			return nil, err
		}
		now := time.Now()
		created, err := tx.OrderAbnormalCase.Create().
			SetID(newID).
			SetOrderID(orderID).
			SetAbnormalCaseID(abnormalCaseID).
			SetStatus(orderabnormalcaseent.StatusACTIVE).
			SetMarkedAt(now).
			SetMarkedBy(actorID).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			if ent.IsConstraintError(err) && strings.Contains(err.Error(), "orderabnormalcase_order_id_abnormal_case_id") {
				return nil, biz.ErrOrderAbnormalCaseExists
			}
			return nil, err
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["abnormal_case_record.id"] = created.ID.String()
		if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return orderAbnormalCaseToBiz(created), nil
	}

	if item.Status == orderabnormalcaseent.StatusACTIVE {
		_ = tx.Rollback()
		return nil, biz.ErrOrderAbnormalCaseExists
	}

	now := time.Now()
	updated, err := tx.OrderAbnormalCase.UpdateOne(item).
		SetStatus(orderabnormalcaseent.StatusACTIVE).
		SetMarkedAt(now).
		SetMarkedBy(actorID).
		ClearResolvedAt().
		ClearResolvedBy().
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if audit.Details == nil {
		audit.Details = make(map[string]string)
	}
	audit.Details["abnormal_case_record.id"] = updated.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return orderAbnormalCaseToBiz(updated), nil
}

func (r *orderAbnormalCaseRepo) Resolve(ctx context.Context, organizationID, orderID, actorID, id uuid.UUID, audit *biz.AuditEvent) (*biz.OrderAbnormalCase, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderAbnormalCase.Query().
		Where(
			orderabnormalcaseent.IDEQ(id),
			orderabnormalcaseent.OrderIDEQ(orderID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderAbnormalCaseNotFound
		}
		return nil, err
	}
	if item.Status != orderabnormalcaseent.StatusACTIVE {
		_ = tx.Rollback()
		return nil, biz.ErrOrderAbnormalCaseStatusConflict
	}
	now := time.Now()
	updated, err := tx.OrderAbnormalCase.UpdateOne(item).
		SetStatus(orderabnormalcaseent.StatusRESOLVED).
		SetResolvedAt(now).
		SetResolvedBy(actorID).
		Save(ctx)
	if err != nil {
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
	return orderAbnormalCaseToBiz(updated), nil
}

func (r *orderAbnormalCaseRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	n, err := tx.OrderAbnormalCase.Delete().
		Where(
			orderabnormalcaseent.IDEQ(id),
			orderabnormalcaseent.OrderIDEQ(orderID),
		).
		Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if n == 0 {
		_ = tx.Rollback()
		return biz.ErrOrderAbnormalCaseNotFound
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
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
