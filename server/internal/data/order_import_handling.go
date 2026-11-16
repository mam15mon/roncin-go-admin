package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	assetent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachmentasset"
	handlingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderimporthandling"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderimporthandlingevent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// 当前资料仅保存服务事实；订单版本和人员/代理名称始终从其所属实体投影。
func marshalImportHandlingState(value *biz.OrderImportHandling) (json.RawMessage, error) {
	return json.Marshal(struct {
		ExchangeAgentID *uuid.UUID               `json:"exchangeAgentId,omitempty"`
		Steps           []biz.ImportHandlingStep `json:"steps"`
	}{ExchangeAgentID: value.ExchangeAgentID, Steps: value.Steps})
}

func importHandlingOrder(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*ent.Order, error) {
	o, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if !biz.OrderBusinessType(o.BusinessType).IsImport() {
		return nil, biz.ErrOrderBusinessUnsupported
	}
	return o, nil
}
func readImportHandling(ctx context.Context, client *ent.Client, o *ent.Order) (*biz.OrderImportHandling, error) {
	row, err := client.OrderImportHandling.Query().Where(handlingent.OrderIDEQ(o.ID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	var v biz.OrderImportHandling
	if err = json.Unmarshal(row.State, &v); err != nil {
		return nil, err
	}
	if err := biz.ValidateImportHandling(&v, nil, ""); err != nil {
		return nil, err
	}
	v.OrderID = o.ID
	v.OrderVersion = o.Version
	if v.ExchangeAgentID != nil {
		agent, err := client.Partner.Query().Where(partnerent.IDEQ(*v.ExchangeAgentID), partnerent.OrganizationIDEQ(o.OrganizationID)).Only(ctx)
		if err != nil {
			return nil, err
		}
		v.ExchangeAgentName = agent.LegalName
	}
	if err := populateImportHandlingNames(ctx, client, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
func (r *orderRepo) GetImportHandling(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderImportHandling, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	o, err := importHandlingOrder(ctx, client, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	return readImportHandling(ctx, client, o)
}
func (r *orderRepo) SaveImportHandling(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedVersion uint64, input *biz.OrderImportHandling, reason string) (*biz.OrderImportHandling, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		o, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if !biz.OrderBusinessType(o.BusinessType).IsImport() {
			return biz.ErrOrderBusinessUnsupported
		}
		if o.Version != expectedVersion {
			return biz.ErrOrderStatusConflict
		}
		if o.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderClosureBlocked
		}
		if err = ensureOrderBusinessEditable(ctx, tx, o); err != nil {
			return err
		}
		before, err := readImportHandling(ctx, tx.Client(), o)
		if err != nil {
			return err
		}
		if err = biz.ValidateImportHandling(input, before, reason); err != nil {
			return err
		}
		input.OrderID = orderID
		input.OrderVersion = o.Version + 1
		input.ExchangeAgentName = ""
		if input.ExchangeAgentID != nil {
			agent, err := tx.Partner.Query().Where(partnerent.IDEQ(*input.ExchangeAgentID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).ForShare().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrImportHandlingInvalid, nil)
			}
			role, err := tx.PartnerRole.Query().Where(roleent.PartnerIDEQ(agent.ID), roleent.RoleTypeEQ(roleent.RoleTypeSupplier), roleent.EnabledEQ(true)).ForShare().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrImportHandlingInvalid, nil)
			}
			if role.Blacklisted {
				return biz.NewOrderPartnerRoleBlacklisted(biz.PartnerRoleSupplier)
			}
			input.ExchangeAgentName = agent.LegalName
		}
		for _, step := range input.Steps {
			if step.ResponsibleUserID != nil {
				q, err := companyPersonnelQuery(ctx, tx.Client(), organizationID, "")
				if err != nil {
					return err
				}
				ok, err := q.Where(userent.IDEQ(*step.ResponsibleUserID)).ForShare().Exist(ctx)
				if err != nil {
					return err
				}
				if !ok {
					return biz.ErrImportHandlingInvalid
				}
			}
			if len(step.AttachmentIDs) > 0 {
				rows, err := tx.OrderAttachment.Query().Where(attachmentent.IDIn(step.AttachmentIDs...), attachmentent.OrderIDEQ(orderID), attachmentent.HasAssetWith(assetent.OrganizationIDEQ(organizationID))).Select(attachmentent.FieldID).ForShare().All(ctx)
				if err != nil {
					return err
				}
				if len(rows) != len(step.AttachmentIDs) {
					return biz.ErrImportHandlingInvalid
				}
			}
		}
		previous, err := json.Marshal(before)
		if err != nil {
			return err
		}
		next, err := json.Marshal(input)
		if err != nil {
			return err
		}
		state, err := marshalImportHandlingState(input)
		if err != nil {
			return err
		}
		if _, err = tx.OrderImportHandling.Update().Where(handlingent.OrderIDEQ(orderID)).SetState(state).Save(ctx); err != nil {
			return err
		}
		if _, err = tx.Order.UpdateOne(o).SetVersion(o.Version + 1).SetFlowStatus(orderent.FlowStatus(biz.ImportHandlingFlow(input))).Save(ctx); err != nil {
			return err
		}
		if _, err = tx.OrderImportHandlingEvent.Create().SetOrderID(orderID).SetActorID(actorID).SetOccurredAt(time.Now().UTC()).SetReason(reason).SetBefore(previous).SetAfter(next).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "order.import_handling.save", Result: "success", Details: map[string]string{"order.id": orderID.String(), "reason": reason}})
	})
	if err != nil {
		return nil, err
	}
	return r.GetImportHandling(ctx, organizationID, orderID)
}
func (r *orderRepo) ListImportHandlingEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, size int) ([]*biz.ImportHandlingEvent, int, error) {
	if !biz.ValidListPagination(page, size) {
		return nil, 0, biz.ErrImportHandlingInvalid
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	if _, err = importHandlingOrder(ctx, client, organizationID, orderID); err != nil {
		return nil, 0, err
	}
	q := client.OrderImportHandlingEvent.Query().Where(eventent.OrderIDEQ(orderID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Order(ent.Desc(eventent.FieldOccurredAt), ent.Desc(eventent.FieldID)).Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.ImportHandlingEvent, 0, len(rows))
	for _, row := range rows {
		var before, after biz.OrderImportHandling
		if err = json.Unmarshal(row.Before, &before); err != nil {
			return nil, 0, err
		}
		if err = json.Unmarshal(row.After, &after); err != nil {
			return nil, 0, err
		}
		if err := populateImportHandlingNames(ctx, client, &before); err != nil {
			return nil, 0, err
		}
		if err := populateImportHandlingNames(ctx, client, &after); err != nil {
			return nil, 0, err
		}
		actorID, err := client.User.Get(ctx, row.ActorID)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, &biz.ImportHandlingEvent{ActorName: actorID.DisplayName, ID: row.ID, OrderID: orderID, ActorID: row.ActorID, OccurredAt: row.OccurredAt, Reason: row.Reason, Before: &before, After: &after})
	}
	return out, total, nil
}

func populateImportHandlingNames(ctx context.Context, client *ent.Client, value *biz.OrderImportHandling) error {
	for i := range value.Steps {
		step := &value.Steps[i]
		if step.ResponsibleUserID != nil {
			u, err := client.User.Get(ctx, *step.ResponsibleUserID)
			if err != nil {
				return err
			}
			step.ResponsibleUserName = u.DisplayName
		}
	}
	return nil
}
