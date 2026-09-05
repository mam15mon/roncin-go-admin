package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

type seaOrderLockHouseBillSnapshot struct {
	HouseBillID        uuid.UUID
	HouseBillVersionID uuid.UUID
	HouseNoSnapshot    string
}

type seaOrderLockSnapshot struct {
	MasterBillID        uuid.UUID
	MasterBillVersionID uuid.UUID
	HouseBills          []seaOrderLockHouseBillSnapshot
}

// createSeaOrderLockSnapshot 在已持有 Order 行锁的事务内创建或复用 SE 单证不可变版本。
// 固定锁序为 Order → MBL → Link → 按 UUID 排序的 HBL。
func createSeaOrderLockSnapshot(ctx context.Context, tx *ent.Tx, organizationID, orderID, callerID uuid.UUID) (*seaOrderLockSnapshot, error) {
	// 只读定位当前 Active Link，以便后续先锁 MBL 再锁 Link。
	activeLink, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.OrderIDEQ(orderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSeaDocumentNoActiveLink
		}
		return nil, err
	}

	// 锁序 2: MBL FOR UPDATE。
	mbl, err := tx.SeaMasterBill.Query().
		Where(seamasterbillent.IDEQ(activeLink.MasterBillID), seamasterbillent.OrganizationIDEQ(organizationID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 锁序 3: Active Link FOR UPDATE，并重验只读定位期间关系未发生变化。
	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(seamasterbillorderlinkent.IDEQ(activeLink.ID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if !seaDocumentLinkMatches(link, organizationID, orderID, mbl.ID) {
		return nil, biz.ErrSeaDocumentStructureConflict
	}

	execution, err := tx.SeaTransportExecution.Query().
		Where(seatransportexecutionent.IDEQ(mbl.TransportExecutionID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 锁序 4: HBL FOR UPDATE，查询显式按 UUID 升序。
	hbls, err := tx.SeaHouseBill.Query().
		Where(
			seahousebillent.OrganizationIDEQ(organizationID),
			seahousebillent.OrderIDEQ(orderID),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED, seahousebillent.StatusREPLACED),
		).
		Order(ent.Asc(seahousebillent.FieldID)).
		ForUpdate().
		All(ctx)
	if err != nil {
		return nil, err
	}
	if link.DocumentStructure == seamasterbillorderlinkent.DocumentStructureDIRECT && len(hbls) > 0 {
		return nil, biz.ErrSeaDocumentStructureConflict
	}
	if link.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE && len(hbls) == 0 {
		return nil, biz.ErrSeaDocumentStructureConflict
	}

	mblHash := computeMBLContentHash(mbl, execution)
	var mblVersionID uuid.UUID
	reusedMblVersion := false
	if mbl.CurrentVersionID != nil {
		current, currentErr := tx.SeaMasterBillVersion.Get(ctx, *mbl.CurrentVersionID)
		if currentErr == nil && current != nil && current.SourceEntityVersion == mbl.Version && current.ContentHash == mblHash {
			mblVersionID = current.ID
			reusedMblVersion = true
		}
	}
	if !reusedMblVersion {
		latest, _ := tx.SeaMasterBillVersion.Query().
			Where(seamasterbillversionent.MasterBillIDEQ(mbl.ID)).
			Order(ent.Desc(seamasterbillversionent.FieldVersionNo)).
			First(ctx)
		nextVersion := uint64(1)
		if latest != nil {
			nextVersion = latest.VersionNo + 1
		}

		var vesselVoyageSnapshot *string
		var etdSnapshot *string
		var etaSnapshot *string
		if execution != nil {
			vesselVoyage := strings.TrimSpace(execution.VesselName + " " + execution.VoyageNo)
			if vesselVoyage != "" {
				vesselVoyageSnapshot = &vesselVoyage
			}
			if execution.Etd != nil {
				value := execution.Etd.Format(time.RFC3339)
				etdSnapshot = &value
			}
			if execution.Eta != nil {
				value := execution.Eta.Format(time.RFC3339)
				etaSnapshot = &value
			}
		}

		create := tx.SeaMasterBillVersion.Create().
			SetOrganizationID(organizationID).
			SetMasterBillID(mbl.ID).
			SetVersionNo(nextVersion).
			SetSourceEntityVersion(mbl.Version).
			SetIssuerPartnerID(mbl.IssuerPartnerID).
			SetTransportExecutionID(mbl.TransportExecutionID).
			SetMasterNo(mbl.MasterNo).
			SetNormalizedMasterNo(mbl.NormalizedMasterNo).
			SetStatus(seamasterbillversionent.Status(mbl.Status)).
			SetNillableVesselVoyageSnapshot(vesselVoyageSnapshot).
			SetNillableEtdSnapshot(etdSnapshot).
			SetNillableEtaSnapshot(etaSnapshot).
			SetContentHash(mblHash).
			SetSource(seamasterbillversionent.SourceORDER_LOCK).
			SetNillableCreatedBy(&callerID).
			SetNillableShipperText(mbl.ShipperText).
			SetNillableConsigneeText(mbl.ConsigneeText).
			SetNillableNotifyPartyText(mbl.NotifyPartyText).
			SetNillableSecondNotifyPartyText(mbl.SecondNotifyPartyText).
			SetNillableMarksText(mbl.MarksText).
			SetNillableGoodsDescriptionText(mbl.GoodsDescriptionText).
			SetNillablePackageCount(mbl.PackageCount).
			SetNillablePackageUnit(mbl.PackageUnit).
			SetNillableGrossWeightKg(mbl.GrossWeightKg).
			SetNillableVolumeCbm(mbl.VolumeCbm).
			SetNillableFreightTerms(mbl.FreightTerms).
			SetNillableTransportTerms(mbl.TransportTerms).
			SetNillableBillForm(mbl.BillForm).
			SetNillableReleaseType(mbl.ReleaseType).
			SetNillableClauses(mbl.Clauses)
		if execution != nil {
			create.
				SetNillableCarrierID(execution.CarrierID).
				SetNillableOriginLocationID(execution.OriginLocationID).
				SetNillableDischargeLocationID(execution.DischargeLocationID).
				SetNillableTransitLocationID(execution.TransitLocationID).
				SetVesselName(execution.VesselName).
				SetVoyageNo(execution.VoyageNo).
				SetNillableEtd(execution.Etd).
				SetNillableEta(execution.Eta)
		}
		created, createErr := create.Save(ctx)
		if createErr != nil {
			return nil, createErr
		}
		if created.MasterBillID != mbl.ID {
			return nil, biz.ErrSeaDocumentStructureConflict
		}
		mblVersionID = created.ID
		if _, err := tx.SeaMasterBill.UpdateOneID(mbl.ID).SetCurrentVersionID(created.ID).Save(ctx); err != nil {
			return nil, err
		}
	}

	snapshot := &seaOrderLockSnapshot{
		MasterBillID:        mbl.ID,
		MasterBillVersionID: mblVersionID,
		HouseBills:          make([]seaOrderLockHouseBillSnapshot, 0, len(hbls)),
	}
	for _, hbl := range hbls {
		hash := computeHBLContentHash(hbl)
		var versionID uuid.UUID
		reused := false
		if hbl.CurrentVersionID != nil {
			current, currentErr := tx.SeaHouseBillVersion.Get(ctx, *hbl.CurrentVersionID)
			if currentErr == nil && current != nil && current.SourceEntityVersion == hbl.Version && current.ContentHash == hash {
				versionID = current.ID
				reused = true
			}
		}
		if !reused {
			latest, _ := tx.SeaHouseBillVersion.Query().
				Where(seahousebillversionent.HouseBillIDEQ(hbl.ID)).
				Order(ent.Desc(seahousebillversionent.FieldVersionNo)).
				First(ctx)
			nextVersion := uint64(1)
			if latest != nil {
				nextVersion = latest.VersionNo + 1
			}
			created, createErr := tx.SeaHouseBillVersion.Create().
				SetOrganizationID(organizationID).
				SetHouseBillID(hbl.ID).
				SetOrderID(orderID).
				SetMasterBillID(mbl.ID).
				SetVersionNo(nextVersion).
				SetSourceEntityVersion(hbl.Version).
				SetHouseNo(hbl.HouseNo).
				SetNormalizedHouseNo(hbl.NormalizedHouseNo).
				SetIssuerSource(seahousebillversionent.IssuerSource(hbl.IssuerSource)).
				SetNillableIssuerOrganizationID(hbl.IssuerOrganizationID).
				SetNillableIssuerPartnerID(hbl.IssuerPartnerID).
				SetStatus(seahousebillversionent.Status(hbl.Status)).
				SetNillableNote(hbl.Note).
				SetContentHash(hash).
				SetSource(seahousebillversionent.SourceORDER_LOCK).
				SetNillableCreatedBy(&callerID).
				SetNillableShipperText(hbl.ShipperText).
				SetNillableConsigneeText(hbl.ConsigneeText).
				SetNillableNotifyPartyText(hbl.NotifyPartyText).
				SetNillableSecondNotifyPartyText(hbl.SecondNotifyPartyText).
				SetNillableMarksText(hbl.MarksText).
				SetNillableGoodsDescriptionText(hbl.GoodsDescriptionText).
				SetNillablePackageCount(hbl.PackageCount).
				SetNillablePackageUnit(hbl.PackageUnit).
				SetNillableGrossWeightKg(hbl.GrossWeightKg).
				SetNillableVolumeCbm(hbl.VolumeCbm).
				SetNillableFreightTerms(hbl.FreightTerms).
				SetNillableTransportTerms(hbl.TransportTerms).
				SetNillableBillForm(hbl.BillForm).
				SetNillableReleaseType(hbl.ReleaseType).
				SetNillableClauses(hbl.Clauses).
				Save(ctx)
			if createErr != nil {
				return nil, createErr
			}
			if created.HouseBillID != hbl.ID {
				return nil, biz.ErrSeaDocumentStructureConflict
			}
			versionID = created.ID
			if _, err := tx.SeaHouseBill.UpdateOneID(hbl.ID).SetCurrentVersionID(created.ID).Save(ctx); err != nil {
				return nil, err
			}
		}
		snapshot.HouseBills = append(snapshot.HouseBills, seaOrderLockHouseBillSnapshot{
			HouseBillID:        hbl.ID,
			HouseBillVersionID: versionID,
			HouseNoSnapshot:    hbl.HouseNo,
		})
	}
	return snapshot, nil
}
