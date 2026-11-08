package data

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	lifecycle "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	houseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	masterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	transportent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

func seaRestoreConflict(message string) error {
	return errors.Conflict("SEA_DOCUMENT_STRUCTURE_CONFLICT", message)
}

// lockSeaOrderRestore 先排序锁定完整成员订单，再锁主单、关系、航程及分单；不在取得主单锁后补锁成员。
func lockSeaOrderRestore(ctx context.Context, tx *ent.Tx, org, orderID uuid.UUID, located *ent.SeaMasterBillOrderLink) (*ent.Order, *ent.SeaMasterBillOrderLink, error) {
	memberLinks, err := tx.SeaMasterBillOrderLink.Query().Where(linkent.OrganizationIDEQ(org), linkent.MasterBillIDEQ(located.MasterBillID), linkent.StatusEQ(linkent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	memberIDs := make([]uuid.UUID, 0, len(memberLinks)+1)
	memberIDs = append(memberIDs, orderID)
	for _, link := range memberLinks {
		memberIDs = append(memberIDs, link.OrderID)
	}
	memberIDs = sortAndDeduplicateUUIDs(memberIDs)
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(org), orderent.IDIn(memberIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(orders) != len(memberIDs) {
		return nil, nil, biz.ErrSeaDocumentStructureConflict
	}
	var target *ent.Order
	for _, order := range orders {
		if order.ID == orderID {
			target = order
		}
		if order.LockedAt != nil {
			return nil, nil, seaRestoreConflict("原主单关联订单已审核锁定，请先解锁后恢复")
		}
		if order.ID != orderID && order.TerminationStatus != orderent.TerminationStatusACTIVE {
			return nil, nil, seaRestoreConflict("原主单存在不可加入的成员状态，请先处理后恢复")
		}
	}
	if target == nil {
		return nil, nil, biz.ErrOrderNotFound
	}
	master, err := tx.SeaMasterBill.Query().Where(masterent.IDEQ(located.MasterBillID), masterent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	if master.Status == masterent.StatusVOIDED {
		return nil, nil, biz.ErrSeaDocumentVoided
	}
	links, err := tx.SeaMasterBillOrderLink.Query().Where(linkent.OrganizationIDEQ(org), linkent.Or(linkent.IDEQ(located.ID), linkent.And(linkent.MasterBillIDEQ(master.ID), linkent.StatusEQ(linkent.StatusACTIVE)))).Order(linkent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, nil, err
	}
	var source *ent.SeaMasterBillOrderLink
	actualMembers := []uuid.UUID{orderID}
	for _, link := range links {
		if link.ID == located.ID {
			source = link
		}
		if link.Status == linkent.StatusACTIVE {
			actualMembers = append(actualMembers, link.OrderID)
		}
	}
	if !equalUUIDSlices(sortAndDeduplicateUUIDs(actualMembers), memberIDs) {
		return nil, nil, biz.ErrSeaDocumentStructureConflict
	}
	if source == nil || source.OrderID != orderID || source.Status != linkent.StatusENDED || source.MasterBillID != located.MasterBillID || source.TransportExecutionID != located.TransportExecutionID || source.DocumentStructure != located.DocumentStructure || source.Version != located.Version || source.EndedAt == nil || located.EndedAt == nil || !source.EndedAt.Equal(*located.EndedAt) || source.EndedReason == nil || *source.EndedReason != orderTerminationLinkEndedReason {
		return nil, nil, biz.ErrSeaDocumentStructureConflict
	}
	active, err := tx.SeaMasterBillOrderLink.Query().Where(linkent.OrganizationIDEQ(org), linkent.OrderIDEQ(orderID), linkent.StatusEQ(linkent.StatusACTIVE)).Exist(ctx)
	if err != nil {
		return nil, nil, err
	}
	if active {
		return nil, nil, biz.ErrOrderStatusConflict
	}
	for _, link := range links {
		if link.Status != linkent.StatusACTIVE {
			continue
		}
		if source.DocumentStructure != linkent.DocumentStructureHOUSE || link.DocumentStructure != linkent.DocumentStructureHOUSE {
			return nil, nil, seaRestoreConflict("原主单当前不满足直单独占或全员分单约束，不能恢复")
		}
	}
	transport, err := tx.SeaTransportExecution.Query().Where(transportent.IDEQ(source.TransportExecutionID), transportent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	if transport.ShippingLineID != master.ShippingLineID || seaTransportExecutionDiffersFromOrder(transport, &biz.Order{ShippingLineID: target.ShippingLineID, OriginLocationID: target.OriginLocationID, DischargeLocationID: target.DischargeLocationID, TransitLocationID: target.TransitLocationID, VesselVoyage: target.VesselVoyage, ETD: target.Etd, ETA: target.Eta}) {
		return nil, nil, seaRestoreConflict("原主单或原航程资料与订单不一致，不能直接恢复，请核对运输安排")
	}
	houses, err := tx.SeaHouseBill.Query().Where(houseent.OrganizationIDEQ(org), houseent.OrderIDEQ(orderID), houseent.StatusNEQ(houseent.StatusVOIDED)).Order(houseent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, nil, err
	}
	switch source.DocumentStructure {
	case linkent.DocumentStructureDIRECT:
		if len(houses) != 0 {
			return nil, nil, biz.ErrSeaDocumentStructureConflict
		}
	case linkent.DocumentStructureHOUSE:
		if len(houses) != 1 || houses[0].MasterBillID != master.ID {
			return nil, nil, seaRestoreConflict("原分单已作废或结构已变化，不能恢复")
		}
	default:
		return nil, nil, biz.ErrSeaDocumentStructureInvalid
	}
	return target, source, nil
}

func createSeaOrderRestoreLink(ctx context.Context, tx *ent.Tx, source *ent.SeaMasterBillOrderLink, at time.Time) (*ent.SeaMasterBillOrderLink, error) {
	link, err := tx.SeaMasterBillOrderLink.Create().SetOrganizationID(source.OrganizationID).SetOrderID(source.OrderID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(source.DocumentStructure).SetStatus(linkent.StatusACTIVE).SetStartedAt(at).SetVersion(1).Save(ctx)
	return link, mapEntError(err, nil, biz.ErrSeaDocumentStructureConflict)
}

// SeaOrderRestoreRepairInput 是一次性显式补链命令的目标与证据，不用于普通状态转移。
type SeaOrderRestoreRepairInput struct {
	OrganizationID  uuid.UUID
	OrderID         uuid.UUID
	ExpectedVersion uint64
	EndedLinkID     uuid.UUID
	ActorID         uuid.UUID
	Reason          string
}

type SeaOrderRestoreRepairReport struct {
	OrderID              uuid.UUID  `json:"orderId"`
	OrderNo              string     `json:"orderNo"`
	Version              uint64     `json:"version"`
	EndedLinkID          uuid.UUID  `json:"endedLinkId"`
	MasterBillID         uuid.UUID  `json:"masterBillId"`
	TransportExecutionID uuid.UUID  `json:"transportExecutionId"`
	TerminationEventID   uuid.UUID  `json:"terminationEventId"`
	RestoreEventID       uuid.UUID  `json:"restoreEventId"`
	NewLinkID            *uuid.UUID `json:"newLinkId,omitempty"`
	Applied              bool       `json:"applied"`
}

// RepairSeaOrderRestoreLink 默认仅预检。Apply时锁内重查真实权限及证据，绝不伪造ACTIVE→ACTIVE事件。
func (d *Data) RepairSeaOrderRestoreLink(ctx context.Context, in SeaOrderRestoreRepairInput, apply bool) (*SeaOrderRestoreRepairReport, error) {
	if in.OrganizationID == uuid.Nil || in.OrderID == uuid.Nil || in.EndedLinkID == uuid.Nil || in.ActorID == uuid.Nil || in.ExpectedVersion == 0 || strings.TrimSpace(in.Reason) == "" || len([]rune(in.Reason)) > 500 {
		return nil, biz.ErrOrderTerminationInvalid
	}
	var report *SeaOrderRestoreRepairReport
	err := d.WithTx(ctx, func(tx *ent.Tx) error {
		// 沿用权限父行→订单的锁序，与角色/成员维护并发时锁内复核真实授权。
		for _, op := range []access.OrderOperation{access.OrderRead, access.OrderTransition} {
			ok, err := hasRealtimeFeePermission(ctx, tx.Client(), in.OrganizationID, in.ActorID, access.OrderPermission(access.OrderBusinessSE, op), true)
			if err != nil {
				return err
			}
			if !ok {
				return biz.ErrPermissionDenied
			}
		}
		located, err := tx.SeaMasterBillOrderLink.Query().Where(linkent.IDEQ(in.EndedLinkID), linkent.OrganizationIDEQ(in.OrganizationID), linkent.OrderIDEQ(in.OrderID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
		}
		order, source, err := lockSeaOrderRestore(ctx, tx, in.OrganizationID, in.OrderID, located)
		if err != nil {
			return err
		}
		if order.BusinessType != orderent.BusinessTypeSE || order.TerminationStatus != orderent.TerminationStatusACTIVE || order.ClosureStatus != orderent.ClosureStatusOPEN || order.Version != in.ExpectedVersion {
			return biz.ErrOrderStatusConflict
		}
		events, err := tx.OrderLifecycleEvent.Query().Where(lifecycle.OrderIDEQ(order.ID), lifecycle.DimensionEQ(lifecycle.DimensionTERMINATION)).Order(ent.Desc(lifecycle.FieldChangedAt), ent.Desc(lifecycle.FieldID)).Limit(3).All(ctx)
		if err != nil {
			return err
		}
		if len(events) < 2 || events[0].Action != "transition" || events[0].ReferenceID != nil || events[0].ReferenceType != nil || events[0].FromStatus == nil || *events[0].FromStatus != "TERMINATED" || events[0].ToStatus != "ACTIVE" || events[1].FromStatus == nil || *events[1].FromStatus != "TERMINATING" || events[1].ToStatus != "TERMINATED" || !events[0].ChangedAt.After(events[1].ChangedAt) || source.EndedAt == nil || !source.EndedAt.Equal(events[1].ChangedAt) {
			return seaRestoreConflict("恢复事件与指定退关关联不能唯一配对，拒绝补链")
		}
		if len(events) > 2 && !events[1].ChangedAt.After(events[2].ChangedAt) {
			return biz.ErrSeaDocumentStructureConflict
		}
		matched, err := tx.SeaMasterBillOrderLink.Query().Where(linkent.OrganizationIDEQ(in.OrganizationID), linkent.OrderIDEQ(order.ID), linkent.StatusEQ(linkent.StatusENDED), linkent.EndedReasonEQ(orderTerminationLinkEndedReason), linkent.EndedAtEQ(events[1].ChangedAt)).Count(ctx)
		if err != nil {
			return err
		}
		if matched != 1 {
			return biz.ErrSeaDocumentStructureConflict
		}
		changed, err := tx.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(order.ID), linkent.IDNEQ(source.ID), linkent.Or(linkent.CreatedAtGTE(events[0].ChangedAt), linkent.StartedAtGTE(events[0].ChangedAt), linkent.EndedAtGTE(events[0].ChangedAt))).Exist(ctx)
		if err != nil {
			return err
		}
		if changed {
			return seaRestoreConflict("恢复后已有其他运输关联，拒绝补链")
		}
		report = &SeaOrderRestoreRepairReport{OrderID: order.ID, OrderNo: order.OrderNo, Version: order.Version, EndedLinkID: source.ID, MasterBillID: source.MasterBillID, TransportExecutionID: source.TransportExecutionID, TerminationEventID: events[1].ID, RestoreEventID: events[0].ID}
		if !apply {
			return nil
		}
		link, err := createSeaOrderRestoreLink(ctx, tx, source, time.Now().UTC())
		if err != nil {
			return err
		}
		if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
			return err
		}
		if err = writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &in.OrganizationID, UserID: &in.ActorID, Action: "order.sea_restore_link.repair", ResourceID: order.ID.String(), Result: "success", Details: map[string]string{"reason": strings.TrimSpace(in.Reason), "ended_link_id": source.ID.String(), "new_link_id": link.ID.String(), "termination_event_id": events[1].ID.String(), "restore_event_id": events[0].ID.String()}}); err != nil {
			return err
		}
		report.Applied = true
		report.NewLinkID = &link.ID
		report.Version++
		return nil
	})
	return report, err
}
