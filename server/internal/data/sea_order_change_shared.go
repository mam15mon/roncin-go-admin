package data

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
)

func externalConfirmationFromReassignment(event *ent.SeaOrderReassignmentEvent) *biz.SeaExternalConfirmation {
	if event == nil || strings.TrimSpace(event.ConfirmedByParty) == "" || event.ConfirmedAt.IsZero() || strings.TrimSpace(event.ConfirmationNote) == "" {
		return nil
	}
	return &biz.SeaExternalConfirmation{
		ConfirmedByParty:         event.ConfirmedByParty,
		ConfirmedAt:              event.ConfirmedAt,
		ConfirmationNote:         event.ConfirmationNote,
		ConfirmationAttachmentID: event.ConfirmationAttachmentID,
	}
}

// ---------------------------------------------------------------------------
// 6. 变更历史事件列表与详情
// ---------------------------------------------------------------------------

func validateNewMasterBillInput(ctx context.Context, client *ent.Client, organizationID uuid.UUID, target *biz.SeaOrderSplitTargetInput, forShare bool) (string, error) {
	if target == nil {
		return "", biz.ErrSeaMasterBillInvalidArgument
	}
	normalizedMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(target.MasterNo)
	if err != nil {
		return "", err
	}

	if err := validateTransportExecutionTargetInput(ctx, client, organizationID, target, forShare); err != nil {
		return "", err
	}
	shippingLineID := *target.ShippingLineID

	existingMBL, err := client.SeaMasterBill.Query().Where(
		seamasterbillent.OrganizationIDEQ(organizationID),
		seamasterbillent.ShippingLineIDEQ(shippingLineID),
		seamasterbillent.NormalizedMasterNoEQ(normalizedMasterNo),
	).Exist(ctx)
	if err != nil {
		return "", err
	}
	if existingMBL {
		return "", biz.ErrSeaMasterBillExists
	}

	return normalizedMasterNo, nil
}

func validateTransportExecutionTargetInput(ctx context.Context, client *ent.Client, organizationID uuid.UUID, target *biz.SeaOrderSplitTargetInput, forShare bool) error {
	if target == nil || target.ShippingLineID == nil || *target.ShippingLineID == uuid.Nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	lineExists, err := enabledShippingLineExists(ctx, client, organizationID, *target.ShippingLineID, forShare)
	if err != nil {
		return err
	}
	if !lineExists {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	portIDs := make([]uuid.UUID, 0, 3)
	if target.OriginLocationID != nil && *target.OriginLocationID != uuid.Nil {
		portIDs = append(portIDs, *target.OriginLocationID)
	}
	if target.DischargeLocationID != nil && *target.DischargeLocationID != uuid.Nil {
		portIDs = append(portIDs, *target.DischargeLocationID)
	}
	if target.TransitLocationID != nil && *target.TransitLocationID != uuid.Nil {
		portIDs = append(portIDs, *target.TransitLocationID)
	}
	if len(portIDs) > 0 {
		portCount, err := client.Port.Query().Where(
			portent.IDIn(portIDs...),
			portent.OrganizationIDEQ(organizationID),
			portent.EnabledEQ(true),
		).Count(ctx)
		if err != nil {
			return err
		}
		if portCount != len(portIDs) {
			return biz.ErrSeaMasterBillInvalidArgument
		}
	}

	if strings.TrimSpace(target.ETD) != "" && parseOptionalTime(target.ETD) == nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	if strings.TrimSpace(target.ETA) != "" && parseOptionalTime(target.ETA) == nil {
		return biz.ErrSeaMasterBillInvalidArgument
	}
	return nil
}

func createNewMasterBillInTx(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, target *biz.SeaOrderSplitTargetInput) (*ent.SeaMasterBill, *ent.SeaTransportExecution, error) {
	normalizedMasterNo, err := validateNewMasterBillInput(ctx, tx.Client(), organizationID, target, true)
	if err != nil {
		return nil, nil, err
	}

	te, err := createNewTransportExecutionInTx(ctx, tx, organizationID, target)
	if err != nil {
		return nil, nil, err
	}

	mbl, err := tx.SeaMasterBill.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetShippingLineID(*target.ShippingLineID).
		SetMasterNo(normalizedMasterNo).
		SetNormalizedMasterNo(normalizedMasterNo).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		return nil, nil, mapEntConstraint(err, "seamasterbill_organization_id_shipping_line_id_normalized_master_no", biz.ErrSeaMasterBillExists)
	}

	return mbl, te, nil
}

func createNewTransportExecutionInTx(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, target *biz.SeaOrderSplitTargetInput) (*ent.SeaTransportExecution, error) {
	teBuilder := tx.SeaTransportExecution.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetVesselName(target.VesselName).
		SetVoyageNo(target.VoyageNo).
		SetVersion(1)
	teBuilder.SetShippingLineID(*target.ShippingLineID)
	if target.OriginLocationID != nil && *target.OriginLocationID != uuid.Nil {
		teBuilder.SetOriginLocationID(*target.OriginLocationID)
	}
	if target.DischargeLocationID != nil && *target.DischargeLocationID != uuid.Nil {
		teBuilder.SetDischargeLocationID(*target.DischargeLocationID)
	}
	if target.TransitLocationID != nil && *target.TransitLocationID != uuid.Nil {
		teBuilder.SetTransitLocationID(*target.TransitLocationID)
	}
	if etd := parseOptionalTime(target.ETD); etd != nil {
		teBuilder.SetEtd(*etd)
	}
	if eta := parseOptionalTime(target.ETA); eta != nil {
		teBuilder.SetEta(*eta)
	}

	te, err := teBuilder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return te, nil
}

func seaMasterBillShippingLineConsistent(mbl *ent.SeaMasterBill, execution *ent.SeaTransportExecution) bool {
	return mbl != nil && execution != nil && mbl.ShippingLineID == execution.ShippingLineID
}

func enabledShippingLineExists(ctx context.Context, client *ent.Client, organizationID, shippingLineID uuid.UUID, forShare bool) (bool, error) {
	if client == nil || organizationID == uuid.Nil || shippingLineID == uuid.Nil {
		return false, nil
	}
	query := client.ShippingLine.Query().Where(
		shippinglineent.IDEQ(shippingLineID),
		shippinglineent.EnabledEQ(true),
	)
	if forShare {
		query.ForShare()
	}
	return query.Exist(ctx)
}

func mblToSummary(ctx context.Context, client *ent.Client, organizationID uuid.UUID, mbl *ent.SeaMasterBill, te *ent.SeaTransportExecution) (*biz.SeaMasterBillSummary, error) {
	s := &biz.SeaMasterBillSummary{
		MasterBillID:   mbl.ID,
		MasterNo:       mbl.MasterNo,
		ShippingLineID: mbl.ShippingLineID,
		Status:         string(mbl.Status),
		Version:        mbl.Version,
	}
	line, err := client.ShippingLine.Query().Where(
		shippinglineent.IDEQ(mbl.ShippingLineID),
	).Only(ctx)
	if err != nil {
		return nil, err
	}
	s.ShippingLineName = formatShippingLineName(line.NameZh, line.NameEn, line.ScacCode)
	if te != nil {
		s.TransportExecutionID = te.ID
		s.TransportExecutionVersion = te.Version
		s.VesselName = te.VesselName
		s.VoyageNo = te.VoyageNo
		if te.OriginLocationID != nil {
			s.OriginLocationID = te.OriginLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*te.OriginLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			s.OriginLocationName = p.NameEn
		}
		if te.DischargeLocationID != nil {
			s.DischargeLocationID = te.DischargeLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*te.DischargeLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			s.DischargeLocationName = p.NameEn
		}
		if te.TransitLocationID != nil {
			s.TransitLocationID = te.TransitLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*te.TransitLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			s.TransitLocationName = p.NameEn
		}
		if te.Etd != nil {
			s.ETD = te.Etd.Format(time.RFC3339)
		}
		if te.Eta != nil {
			s.ETA = te.Eta.Format(time.RFC3339)
		}
	}
	return s, nil
}

func makeDiff(field, label, cur, target string) *biz.VoyageDifference {
	return &biz.VoyageDifference{
		FieldName:    field,
		Label:        label,
		CurrentValue: cur,
		TargetValue:  target,
		IsDifferent:  cur != target,
	}
}

func sortAndDeduplicateUUIDs(ids []uuid.UUID) []uuid.UUID {
	set := make(map[uuid.UUID]struct{}, len(ids))
	unique := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, exists := set[id]; !exists {
			set[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	sort.Slice(unique, func(i, j int) bool {
		return strings.Compare(unique[i].String(), unique[j].String()) < 0
	})
	return unique
}
