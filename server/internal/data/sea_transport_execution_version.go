package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
)

// ensureSeaTransportExecutionVersion 为已加写锁的运输执行创建或复用不可变快照。
// 同一实体版本与内容散列只保留一份快照；调用方仍需把本次外部确认固化在对应业务事件中。
func ensureSeaTransportExecutionVersion(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	execution *ent.SeaTransportExecution,
	actorID *uuid.UUID,
	source seatransportexecutionversionent.Source,
	reason, idempotencyKey, fingerprint *string,
	confirmation *biz.SeaExternalConfirmation,
) (*ent.SeaTransportExecutionVersion, error) {
	hash := computeTransportExecutionContentHash(execution)
	existing, err := tx.SeaTransportExecutionVersion.Query().Where(
		seatransportexecutionversionent.OrganizationIDEQ(organizationID),
		seatransportexecutionversionent.TransportExecutionIDEQ(execution.ID),
		seatransportexecutionversionent.SourceEntityVersionEQ(execution.Version),
		seatransportexecutionversionent.ContentHashEQ(hash),
	).Only(ctx)
	if err == nil {
		if execution.CurrentVersionID == nil || *execution.CurrentVersionID != existing.ID {
			if _, err := execution.Update().SetCurrentVersionID(existing.ID).Save(ctx); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}

	latest, err := tx.SeaTransportExecutionVersion.Query().Where(
		seatransportexecutionversionent.OrganizationIDEQ(organizationID),
		seatransportexecutionversionent.TransportExecutionIDEQ(execution.ID),
	).Order(ent.Desc(seatransportexecutionversionent.FieldVersionNo)).First(ctx)
	nextVersion := uint64(1)
	if err == nil {
		nextVersion = latest.VersionNo + 1
	} else if !ent.IsNotFound(err) {
		return nil, err
	}

	builder := tx.SeaTransportExecutionVersion.Create().
		SetOrganizationID(organizationID).
		SetTransportExecutionID(execution.ID).
		SetVersionNo(nextVersion).
		SetSourceEntityVersion(execution.Version).
		SetShippingLineID(execution.ShippingLineID).
		SetNillableOriginLocationID(execution.OriginLocationID).
		SetNillableDischargeLocationID(execution.DischargeLocationID).
		SetNillableTransitLocationID(execution.TransitLocationID).
		SetVesselName(execution.VesselName).
		SetVoyageNo(execution.VoyageNo).
		SetNillableEtd(execution.Etd).
		SetNillableEta(execution.Eta).
		SetContentHash(hash).
		SetSource(source).
		SetNillableReason(reason).
		SetNillableCreatedBy(actorID).
		SetNillableIdempotencyKey(idempotencyKey).
		SetNillableRequestFingerprint(fingerprint)
	if confirmation != nil {
		builder.SetConfirmedByParty(confirmation.ConfirmedByParty).
			SetConfirmedAt(confirmation.ConfirmedAt).
			SetConfirmationNote(confirmation.ConfirmationNote).
			SetNillableConfirmationAttachmentID(confirmation.ConfirmationAttachmentID)
	}
	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := execution.Update().SetCurrentVersionID(created.ID).Save(ctx); err != nil {
		return nil, err
	}
	return created, nil
}
