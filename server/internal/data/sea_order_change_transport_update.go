package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
)

func transportExecutionDifferences(current *ent.SeaTransportExecution, target *biz.SeaTransportExecutionUpdateInput) []*biz.VoyageDifference {
	formatID := func(value *uuid.UUID) string {
		if value == nil {
			return ""
		}
		return value.String()
	}
	formatTime := func(value *time.Time) string {
		if value == nil {
			return ""
		}
		return value.UTC().Format(time.RFC3339)
	}
	return []*biz.VoyageDifference{
		makeDiff("origin_location_id", "起运港(POL)", formatID(current.OriginLocationID), formatID(target.OriginLocationID)),
		makeDiff("discharge_location_id", "卸货港(POD)", formatID(current.DischargeLocationID), formatID(target.DischargeLocationID)),
		makeDiff("transit_location_id", "中转港", formatID(current.TransitLocationID), formatID(target.TransitLocationID)),
		makeDiff("vessel_name", "船名", current.VesselName, target.VesselName),
		makeDiff("voyage_no", "航次", current.VoyageNo, target.VoyageNo),
		makeDiff("etd", "预计开航时间(ETD)", formatTime(current.Etd), formatTime(target.ETD)),
		makeDiff("eta", "预计到港时间(ETA)", formatTime(current.Eta), formatTime(target.ETA)),
	}
}

func (r *seaOrderChangeRepo) PreviewTransportExecutionUpdate(ctx context.Context, organizationID uuid.UUID, input *biz.SeaTransportExecutionUpdateCommand) (*biz.SeaTransportExecutionUpdatePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
		seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	execution, err := client.SeaTransportExecution.Query().Where(
		seatransportexecutionent.IDEQ(link.TransportExecutionID),
		seatransportexecutionent.OrganizationIDEQ(organizationID),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaTransportExecutionNotFound, nil)
	}
	if execution.Version != input.ExpectedTransportExecutionVersion {
		return nil, biz.ErrSeaOrderReassignmentVersionConflict
	}
	links, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
		seamasterbillorderlinkent.TransportExecutionIDEQ(execution.ID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Order(seamasterbillorderlinkent.ByOrderID()).All(ctx)
	if err != nil {
		return nil, err
	}
	memberIDs := make([]uuid.UUID, 0, len(links))
	for _, member := range links {
		memberIDs = append(memberIDs, member.OrderID)
	}
	impacts, err := collectDocumentImpacts(ctx, client, organizationID, memberIDs, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if len(memberIDs) > 0 {
		orders, err := client.Order.Query().Where(
			orderent.OrganizationIDEQ(organizationID),
			orderent.IDIn(memberIDs...),
		).Order(orderent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			if impact := orderBusinessEditImpact(ctx, client.User, o); impact != nil {
				impacts = append(impacts, impact)
			}
		}
	}
	differences := transportExecutionDifferences(execution, input.Input)
	hasDiff := false
	for _, difference := range differences {
		if difference.IsDifferent {
			hasDiff = true
			break
		}
	}
	executable := hasDiff && !hasBlockingImpact(impacts)
	return &biz.SeaTransportExecutionUpdatePreview{TransportExecutionID: execution.ID, TransportExecutionVersion: execution.Version, MemberOrderIDs: memberIDs, Differences: differences, Impacts: impacts, Executable: executable}, nil
}

func (r *seaOrderChangeRepo) ExecuteTransportExecutionUpdate(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaTransportExecutionUpdateCommand, audit *biz.AuditEvent) (*biz.SeaTransportExecutionUpdateResult, error) {
	fingerprint := changeFingerprint(input)
	var executionID, versionID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		replay, err := tx.SeaTransportExecutionVersion.Query().Where(seatransportexecutionversionent.OrganizationIDEQ(organizationID), seatransportexecutionversionent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
		if err == nil {
			if replay.RequestFingerprint == nil || *replay.RequestFingerprint != fingerprint {
				return biz.ErrSeaOrderReassignmentIdempotencyConflict
			}
			executionID, versionID = replay.TransportExecutionID, replay.ID
			return nil
		}
		if !ent.IsNotFound(err) {
			return err
		}
		requestLink, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(organizationID), seamasterbillorderlinkent.OrderIDEQ(input.OrderID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
		}
		locatedLinks, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(organizationID), seamasterbillorderlinkent.TransportExecutionIDEQ(requestLink.TransportExecutionID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).All(ctx)
		if err != nil {
			return err
		}
		memberOrderIDs, masterBillIDs, linkIDs := make([]uuid.UUID, 0, len(locatedLinks)), make([]uuid.UUID, 0, len(locatedLinks)), make([]uuid.UUID, 0, len(locatedLinks))
		for _, link := range locatedLinks {
			memberOrderIDs = append(memberOrderIDs, link.OrderID)
			masterBillIDs = append(masterBillIDs, link.MasterBillID)
			linkIDs = append(linkIDs, link.ID)
		}
		memberOrderIDs, masterBillIDs, linkIDs = sortAndDeduplicateUUIDs(memberOrderIDs), sortAndDeduplicateUUIDs(masterBillIDs), sortAndDeduplicateUUIDs(linkIDs)
		orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(organizationID), orderent.IDIn(memberOrderIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
		if err != nil || len(orders) != len(memberOrderIDs) {
			return biz.ErrSeaOrderReassignmentVersionConflict
		}
		// 共享航程的船期更新属全部成员订单的业务内容写入：逐单执行统一内容门禁，
		// 任一成员被业务锁定或处于终止/结案状态即整体回滚。
		for _, memberOrder := range orders {
			if err := ensureOrderBusinessEditable(ctx, tx, memberOrder); err != nil {
				return err
			}
		}
		if err := validateConfirmationAttachment(ctx, tx.Client(), organizationID, input.OrderID, input.Confirmation); err != nil {
			return err
		}
		if _, err = tx.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(organizationID), seamasterbillent.IDIn(masterBillIDs...)).Order(seamasterbillent.ByID()).ForUpdate().All(ctx); err != nil {
			return err
		}
		lockedLinks, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(organizationID), seamasterbillorderlinkent.IDIn(linkIDs...), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Order(seamasterbillorderlinkent.ByID()).ForUpdate().All(ctx)
		if err != nil || len(lockedLinks) != len(linkIDs) {
			return biz.ErrSeaDocumentStructureConflict
		}
		for _, lockedLink := range lockedLinks {
			if lockedLink.TransportExecutionID != requestLink.TransportExecutionID {
				return biz.ErrSeaDocumentStructureConflict
			}
		}
		execution, err := tx.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(requestLink.TransportExecutionID), seatransportexecutionent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaTransportExecutionNotFound, nil)
		}
		if execution.Version != input.ExpectedTransportExecutionVersion {
			return biz.ErrSeaOrderReassignmentVersionConflict
		}
		currentLinks, err := tx.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.TransportExecutionIDEQ(execution.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Order(seamasterbillorderlinkent.ByID()).All(ctx)
		if err != nil {
			return err
		}
		currentLinkIDs := make([]uuid.UUID, 0, len(currentLinks))
		for _, currentLink := range currentLinks {
			currentLinkIDs = append(currentLinkIDs, currentLink.ID)
		}
		if !equalUUIDSlices(linkIDs, currentLinkIDs) {
			return biz.ErrSeaDocumentStructureConflict
		}
		differences := transportExecutionDifferences(execution, input.Input)
		changed := false
		for _, difference := range differences {
			changed = changed || difference.IsDifferent
		}
		if !changed {
			return biz.ErrSeaDocumentAmendmentEmpty
		}
		updater := execution.Update().SetVersion(execution.Version + 1).SetVesselName(input.Input.VesselName).SetVoyageNo(input.Input.VoyageNo)
		if input.Input.OriginLocationID != nil {
			updater.SetOriginLocationID(*input.Input.OriginLocationID)
		} else {
			updater.ClearOriginLocationID()
		}
		if input.Input.DischargeLocationID != nil {
			updater.SetDischargeLocationID(*input.Input.DischargeLocationID)
		} else {
			updater.ClearDischargeLocationID()
		}
		if input.Input.TransitLocationID != nil {
			updater.SetTransitLocationID(*input.Input.TransitLocationID)
		} else {
			updater.ClearTransitLocationID()
		}
		if input.Input.ETD != nil {
			updater.SetEtd(*input.Input.ETD)
		} else {
			updater.ClearEtd()
		}
		if input.Input.ETA != nil {
			updater.SetEta(*input.Input.ETA)
		} else {
			updater.ClearEta()
		}
		updated, err := updater.Save(ctx)
		if err != nil {
			return err
		}
		version, err := ensureSeaTransportExecutionVersion(
			ctx, tx, organizationID, updated, &actorID,
			seatransportexecutionversionent.SourceSHARED_UPDATE,
			&input.Reason, &input.IdempotencyKey, &fingerprint, input.Confirmation,
		)
		if err != nil {
			return err
		}
		executionID, versionID = updated.ID, version.ID
		audit.Details["transport_execution.id"] = updated.ID.String()
		audit.Details["transport_execution.version_id"] = version.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	execution, err := client.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(executionID), seatransportexecutionent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.SeaTransportExecutionUpdateResult{TransportExecution: seaTransportExecutionToBiz(execution), VersionID: versionID}, nil
}
