package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	seacargoallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seacargoallocation"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type seaCargoAllocationRepo struct {
	data *Data
}

func NewSeaCargoAllocationRepo(data *Data) biz.SeaCargoAllocationRepo {
	return &seaCargoAllocationRepo{
		data: data,
	}
}

// lockActiveSeaCargoAllocationLink 按 MBL -> Link 固定顺序锁定当前活动关系；调用方须已锁定订单。
func lockActiveSeaCargoAllocationLink(ctx context.Context, tx *ent.Tx, orgID, orderID uuid.UUID) (*ent.SeaMasterBillOrderLink, error) {
	candidate, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrganizationIDEQ(orgID),
			seamasterbillorderlinkent.OrderIDEQ(orderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.SeaMasterBill.Query().
		Where(seamasterbillent.IDEQ(candidate.MasterBillID), seamasterbillent.OrganizationIDEQ(orgID)).
		ForUpdate().
		Only(ctx); err != nil {
		return nil, err
	}
	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(seamasterbillorderlinkent.IDEQ(candidate.ID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if !seaDocumentLinkMatches(link, orgID, orderID, candidate.MasterBillID) {
		return nil, biz.ErrSeaDocumentStructureConflict
	}
	return link, nil
}

func (r *seaCargoAllocationRepo) GetSeaCargoAllocation(ctx context.Context, orgID, orderID uuid.UUID) (*biz.SeaCargoAllocationAggregate, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}

	link, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrganizationIDEQ(orgID),
			seamasterbillorderlinkent.OrderIDEQ(orderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		WithCargoAllocationConfirmedByUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSeaDocumentNoActiveLink
		}
		return nil, err
	}

	cargoItemsEnt, err := client.OrderCargoItem.Query().
		Where(ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(orgID)).
		Order(ent.Asc(ordercargoitement.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	containersEnt, err := client.OrderContainer.Query().
		Where(ordercontainerent.OrderIDEQ(orderID), ordercontainerent.OrganizationIDEQ(orgID)).
		Order(ent.Asc(ordercontainerent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	hbsEnt, err := client.SeaHouseBill.Query().
		Where(
			seahousebillent.OrganizationIDEQ(orgID),
			seahousebillent.OrderIDEQ(orderID),
			seahousebillent.MasterBillIDEQ(link.MasterBillID),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED, seahousebillent.StatusREPLACED),
		).
		WithIssuerOrganization().
		WithIssuerPartner().
		Order(ent.Asc(seahousebillent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	allocsEnt, err := client.SeaCargoAllocation.Query().
		Where(
			seacargoallocationent.OrganizationIDEQ(orgID),
			seacargoallocationent.OrderIDEQ(orderID),
			seacargoallocationent.MasterBillOrderLinkIDEQ(link.ID),
		).
		Order(ent.Asc(seacargoallocationent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	cargoItems := make([]*biz.OrderCargoItem, 0, len(cargoItemsEnt))
	for _, ci := range cargoItemsEnt {
		cargoItems = append(cargoItems, orderCargoItemToBiz(ci))
	}

	containers := make([]*biz.OrderContainer, 0, len(containersEnt))
	for _, c := range containersEnt {
		containers = append(containers, orderContainerToBiz(c))
	}

	houseBills := make([]*biz.SeaHouseBill, 0, len(hbsEnt))
	for _, hb := range hbsEnt {
		var orgName, partnerName string
		if hb.Edges.IssuerOrganization != nil {
			orgName = hb.Edges.IssuerOrganization.Name
		}
		if hb.Edges.IssuerPartner != nil {
			partnerName = hb.Edges.IssuerPartner.LegalName
		}
		houseBills = append(houseBills, seaHouseBillToBiz(hb, orgName, partnerName))
	}

	allocations := make([]*biz.SeaCargoAllocation, 0, len(allocsEnt))
	for _, a := range allocsEnt {
		allocations = append(allocations, seaCargoAllocationToBiz(a))
	}

	shipmentType := ""
	if order.ShipmentType != nil {
		shipmentType = string(*order.ShipmentType)
	}

	progress := biz.CalculateAllocationProgress(cargoItems, containers, houseBills, allocations, shipmentType)

	var allowedActions []biz.SeaCargoAllocationAction
	docStruct := biz.SeaDocumentStructure(link.DocumentStructure)
	allocStatus := biz.SeaCargoAllocationStatus(link.CargoAllocationStatus)

	if docStruct == biz.SeaDocumentStructureHouse {
		if allocStatus == biz.SeaCargoAllocationStatusDraft {
			allowedActions = []biz.SeaCargoAllocationAction{
				biz.SeaCargoAllocationActionSaveDraft,
				biz.SeaCargoAllocationActionConfirm,
				biz.SeaCargoAllocationActionApplyHouseBillSummary,
			}
		} else if allocStatus == biz.SeaCargoAllocationStatusConfirmed {
			allowedActions = []biz.SeaCargoAllocationAction{
				biz.SeaCargoAllocationActionWithdraw,
				biz.SeaCargoAllocationActionApplyHouseBillSummary,
			}
		}
	} else if docStruct == biz.SeaDocumentStructureDirect {
		allowedActions = []biz.SeaCargoAllocationAction{
			biz.SeaCargoAllocationActionApplyMasterBillSummary,
		}
	}

	var confirmedByName string
	if link.Edges.CargoAllocationConfirmedByUser != nil {
		confirmedByName = link.Edges.CargoAllocationConfirmedByUser.DisplayName
	}

	return &biz.SeaCargoAllocationAggregate{
		OrderID:           orderID,
		DocumentStructure: docStruct,
		ShipmentType:      shipmentType,
		AllocationStatus:  allocStatus,
		AllocationVersion: link.CargoAllocationVersion,
		ConfirmedAt:       link.CargoAllocationConfirmedAt,
		ConfirmedBy:       link.CargoAllocationConfirmedBy,
		ConfirmedByName:   confirmedByName,
		CargoItems:        cargoItems,
		Containers:        containers,
		HouseBills:        houseBills,
		Allocations:       allocations,
		Progress:          progress,
		AllowedActions:    allowedActions,
	}, nil
}

func (r *seaCargoAllocationRepo) SaveDraft(
	ctx context.Context,
	orgID, actorID, orderID uuid.UUID,
	expectedAllocationVersion uint64,
	allocations []*biz.SeaCargoAllocationInput,
	audit *biz.AuditEvent,
) (*biz.SeaCargoAllocationAggregate, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, err := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}

		activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(orgID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return err
		}

		mbl, err := tx.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbillent.OrganizationIDEQ(orgID),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Status == seamasterbillent.StatusVOIDED {
			return biz.ErrSeaDocumentVoided
		}

		link, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}

		if !seaDocumentLinkMatches(link, orgID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaCargoAllocationStatusConflict
		}
		if link.CargoAllocationVersion != expectedAllocationVersion {
			return biz.ErrSeaCargoAllocationConflict
		}
		if link.CargoAllocationStatus == seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		cargoItemsEnt, err := tx.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(orgID)).
			Order(ent.Asc(ordercargoitement.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		hbsEnt, err := tx.SeaHouseBill.Query().
			Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.OrderIDEQ(orderID),
				seahousebillent.MasterBillIDEQ(link.MasterBillID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED, seahousebillent.StatusREPLACED),
			).
			Order(ent.Asc(seahousebillent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		containersEnt, err := tx.OrderContainer.Query().
			Where(ordercontainerent.OrderIDEQ(orderID), ordercontainerent.OrganizationIDEQ(orgID)).
			Order(ent.Asc(ordercontainerent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		existingAllocations, err := tx.SeaCargoAllocation.Query().
			Where(
				seacargoallocationent.OrganizationIDEQ(orgID),
				seacargoallocationent.OrderIDEQ(orderID),
				seacargoallocationent.MasterBillOrderLinkIDEQ(link.ID),
			).
			Order(ent.Asc(seacargoallocationent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		existingIDs := make(map[uuid.UUID]struct{}, len(existingAllocations))
		for _, allocation := range existingAllocations {
			existingIDs[allocation.ID] = struct{}{}
		}
		inputIDs := make(map[uuid.UUID]struct{}, len(allocations))
		for _, allocation := range allocations {
			if allocation.ID == nil {
				continue
			}
			if _, exists := existingIDs[*allocation.ID]; !exists {
				return biz.NewErrAllocationInvalidReference("allocation", *allocation.ID)
			}
			if _, duplicated := inputIDs[*allocation.ID]; duplicated {
				return biz.ErrSeaCargoAllocationInvalidArgument
			}
			inputIDs[*allocation.ID] = struct{}{}
		}

		cargoItems := make([]*biz.OrderCargoItem, 0, len(cargoItemsEnt))
		for _, ci := range cargoItemsEnt {
			cargoItems = append(cargoItems, orderCargoItemToBiz(ci))
		}
		containers := make([]*biz.OrderContainer, 0, len(containersEnt))
		for _, c := range containersEnt {
			containers = append(containers, orderContainerToBiz(c))
		}
		houseBills := make([]*biz.SeaHouseBill, 0, len(hbsEnt))
		for _, hb := range hbsEnt {
			houseBills = append(houseBills, seaHouseBillToBiz(hb, "", ""))
		}

		shipmentType := ""
		if order.ShipmentType != nil {
			shipmentType = string(*order.ShipmentType)
		}

		if err := biz.ValidateDraftAllocations(cargoItems, containers, houseBills, allocations, shipmentType); err != nil {
			return err
		}

		_, err = tx.SeaCargoAllocation.Delete().
			Where(
				seacargoallocationent.OrganizationIDEQ(orgID),
				seacargoallocationent.OrderIDEQ(orderID),
				seacargoallocationent.MasterBillOrderLinkIDEQ(link.ID),
			).
			Exec(ctx)
		if err != nil {
			return err
		}

		if len(allocations) > 0 {
			bulk := make([]*ent.SeaCargoAllocationCreate, 0, len(allocations))
			for _, a := range allocations {
				allocationID := uuid.Must(uuid.NewV7())
				if a.ID != nil {
					allocationID = *a.ID
				}
				builder := tx.SeaCargoAllocation.Create().
					SetID(allocationID).
					SetOrganizationID(orgID).
					SetOrderID(orderID).
					SetMasterBillOrderLinkID(link.ID).
					SetCargoItemID(a.CargoItemID).
					SetHouseBillID(a.HouseBillID).
					SetPackageCount(int(a.PackageCount)).
					SetGrossWeightKg(a.GrossWeightKg.StringFixed(3)).
					SetVolumeCbm(a.VolumeCbm.StringFixed(6))
				if a.ContainerID != nil {
					builder.SetContainerID(*a.ContainerID)
				}
				bulk = append(bulk, builder)
			}
			if _, err := tx.SeaCargoAllocation.CreateBulk(bulk...).Save(ctx); err != nil {
				return err
			}
		}

		if _, err := tx.SeaMasterBillOrderLink.UpdateOne(link).
			SetCargoAllocationVersion(expectedAllocationVersion + 1).
			Save(ctx); err != nil {
			return err
		}

		audit.Action = "order.sea_cargo_allocation.save_draft"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["link.id"] = link.ID.String()
		audit.Details["allocation.version.old"] = fmt.Sprintf("%d", expectedAllocationVersion)
		audit.Details["allocation.version.new"] = fmt.Sprintf("%d", expectedAllocationVersion+1)
		audit.Details["allocation.count"] = fmt.Sprintf("%d", len(allocations))
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.GetSeaCargoAllocation(ctx, orgID, orderID)
}

func (r *seaCargoAllocationRepo) Confirm(
	ctx context.Context,
	orgID, actorID, orderID uuid.UUID,
	expectedAllocationVersion uint64,
	audit *biz.AuditEvent,
) (*biz.SeaCargoAllocationAggregate, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, err := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(orgID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return err
		}

		mbl, err := tx.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbillent.OrganizationIDEQ(orgID),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Status == seamasterbillent.StatusVOIDED {
			return biz.ErrSeaDocumentVoided
		}

		link, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}

		if !seaDocumentLinkMatches(link, orgID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaCargoAllocationStatusConflict
		}
		if link.CargoAllocationVersion != expectedAllocationVersion {
			return biz.ErrSeaCargoAllocationConflict
		}
		if link.CargoAllocationStatus == seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		cargoItemsEnt, err := tx.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(orgID)).
			Order(ent.Asc(ordercargoitement.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		hbsEnt, err := tx.SeaHouseBill.Query().
			Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.OrderIDEQ(orderID),
				seahousebillent.MasterBillIDEQ(link.MasterBillID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED, seahousebillent.StatusREPLACED),
			).
			Order(ent.Asc(seahousebillent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		containersEnt, err := tx.OrderContainer.Query().
			Where(ordercontainerent.OrderIDEQ(orderID), ordercontainerent.OrganizationIDEQ(orgID)).
			Order(ent.Asc(ordercontainerent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		allocsEnt, err := tx.SeaCargoAllocation.Query().
			Where(
				seacargoallocationent.OrganizationIDEQ(orgID),
				seacargoallocationent.OrderIDEQ(orderID),
				seacargoallocationent.MasterBillOrderLinkIDEQ(link.ID),
			).
			Order(ent.Asc(seacargoallocationent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		cargoItems := make([]*biz.OrderCargoItem, 0, len(cargoItemsEnt))
		for _, ci := range cargoItemsEnt {
			cargoItems = append(cargoItems, orderCargoItemToBiz(ci))
		}
		containers := make([]*biz.OrderContainer, 0, len(containersEnt))
		for _, c := range containersEnt {
			containers = append(containers, orderContainerToBiz(c))
		}
		houseBills := make([]*biz.SeaHouseBill, 0, len(hbsEnt))
		for _, hb := range hbsEnt {
			houseBills = append(houseBills, seaHouseBillToBiz(hb, "", ""))
		}

		inputs := make([]*biz.SeaCargoAllocationInput, 0, len(allocsEnt))
		for _, a := range allocsEnt {
			w, _ := decimal.NewFromString(a.GrossWeightKg)
			v, _ := decimal.NewFromString(a.VolumeCbm)
			inputs = append(inputs, &biz.SeaCargoAllocationInput{
				ID:            &a.ID,
				CargoItemID:   a.CargoItemID,
				HouseBillID:   a.HouseBillID,
				ContainerID:   a.ContainerID,
				PackageCount:  int32(a.PackageCount),
				GrossWeightKg: w,
				VolumeCbm:     v,
			})
		}

		shipmentType := ""
		if order.ShipmentType != nil {
			shipmentType = string(*order.ShipmentType)
		}

		if err := biz.ValidateConfirmedAllocations(cargoItems, containers, houseBills, inputs, shipmentType); err != nil {
			return err
		}

		now := time.Now()
		if _, err := tx.SeaMasterBillOrderLink.UpdateOne(link).
			SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
			SetCargoAllocationVersion(expectedAllocationVersion + 1).
			SetCargoAllocationConfirmedAt(now).
			SetCargoAllocationConfirmedBy(actorID).
			Save(ctx); err != nil {
			return err
		}

		audit.Action = "order.sea_cargo_allocation.confirm"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["link.id"] = link.ID.String()
		audit.Details["allocation.version.old"] = fmt.Sprintf("%d", expectedAllocationVersion)
		audit.Details["allocation.version.new"] = fmt.Sprintf("%d", expectedAllocationVersion+1)
		audit.Details["allocation.confirmed_at"] = now.Format(time.RFC3339)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.GetSeaCargoAllocation(ctx, orgID, orderID)
}

func (r *seaCargoAllocationRepo) Withdraw(
	ctx context.Context,
	orgID, actorID, orderID uuid.UUID,
	expectedAllocationVersion uint64,
	audit *biz.AuditEvent,
) (*biz.SeaCargoAllocationAggregate, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, err := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(orgID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return err
		}

		mbl, err := tx.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbillent.OrganizationIDEQ(orgID),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Status == seamasterbillent.StatusVOIDED {
			return biz.ErrSeaDocumentVoided
		}

		link, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}

		if !seaDocumentLinkMatches(link, orgID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaCargoAllocationStatusConflict
		}
		if link.CargoAllocationVersion != expectedAllocationVersion {
			return biz.ErrSeaCargoAllocationConflict
		}
		if link.CargoAllocationStatus != seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		if _, err := tx.SeaMasterBillOrderLink.UpdateOne(link).
			SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusDRAFT).
			SetCargoAllocationVersion(expectedAllocationVersion + 1).
			ClearCargoAllocationConfirmedAt().
			ClearCargoAllocationConfirmedBy().
			Save(ctx); err != nil {
			return err
		}

		audit.Action = "order.sea_cargo_allocation.withdraw"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["link.id"] = link.ID.String()
		audit.Details["allocation.version.old"] = fmt.Sprintf("%d", expectedAllocationVersion)
		audit.Details["allocation.version.new"] = fmt.Sprintf("%d", expectedAllocationVersion+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.GetSeaCargoAllocation(ctx, orgID, orderID)
}

func (r *seaCargoAllocationRepo) ApplyHouseBillSummary(
	ctx context.Context,
	orgID, actorID, orderID, houseBillID uuid.UUID,
	expectedAllocationVersion, expectedHouseBillVersion uint64,
	audit *biz.AuditEvent,
) (*biz.SeaHouseBill, error) {
	var updatedHB *ent.SeaHouseBill

	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, err := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(orgID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return err
		}

		mbl, err := tx.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbillent.OrganizationIDEQ(orgID),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Status == seamasterbillent.StatusVOIDED {
			return biz.ErrSeaDocumentVoided
		}

		link, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}

		if !seaDocumentLinkMatches(link, orgID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaCargoAllocationStatusConflict
		}
		if link.CargoAllocationVersion != expectedAllocationVersion {
			return biz.ErrSeaCargoAllocationConflict
		}

		hb, err := tx.SeaHouseBill.Query().
			Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.IDEQ(houseBillID),
				seahousebillent.OrderIDEQ(orderID),
				seahousebillent.MasterBillIDEQ(link.MasterBillID),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaHouseBillNotFound, nil)
		}
		if hb.Version != expectedHouseBillVersion {
			return biz.ErrSeaHouseBillConflict
		}
		if hb.Status == seahousebillent.StatusVOIDED {
			return biz.ErrSeaDocumentVoided
		}
		if hb.Status == seahousebillent.StatusREPLACED {
			return biz.ErrSeaHouseBillSwitchConflict
		}

		allocs, err := tx.SeaCargoAllocation.Query().
			Where(
				seacargoallocationent.OrganizationIDEQ(orgID),
				seacargoallocationent.OrderIDEQ(orderID),
				seacargoallocationent.MasterBillOrderLinkIDEQ(link.ID),
				seacargoallocationent.HouseBillIDEQ(houseBillID),
			).
			Order(ent.Asc(seacargoallocationent.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		if len(allocs) == 0 {
			return biz.ErrSeaCargoAllocationIncomplete
		}

		var totalPkg int
		totalWeight := decimal.Zero
		totalVol := decimal.Zero
		for _, a := range allocs {
			totalPkg += a.PackageCount
			w, _ := decimal.NewFromString(a.GrossWeightKg)
			v, _ := decimal.NewFromString(a.VolumeCbm)
			totalWeight = totalWeight.Add(w)
			totalVol = totalVol.Add(v)
		}

		updatedHB, err = tx.SeaHouseBill.UpdateOne(hb).
			SetPackageCount(totalPkg).
			SetGrossWeightKg(totalWeight.InexactFloat64()).
			SetVolumeCbm(totalVol.InexactFloat64()).
			SetVersion(hb.Version + 1).
			Save(ctx)
		if err != nil {
			return err
		}

		audit.Action = "sea_house_bill.apply_allocation_summary"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["house_bill.id"] = houseBillID.String()
		audit.Details["house_bill.version.old"] = fmt.Sprintf("%d", expectedHouseBillVersion)
		audit.Details["house_bill.version.new"] = fmt.Sprintf("%d", hb.Version+1)
		audit.Details["applied.package_count"] = fmt.Sprintf("%d", totalPkg)
		audit.Details["applied.gross_weight_kg"] = totalWeight.StringFixed(3)
		audit.Details["applied.volume_cbm"] = totalVol.StringFixed(6)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return (&seaDocumentRepo{data: r.data}).getSeaHouseBillByID(ctx, orgID, updatedHB.ID)
}

func (r *seaCargoAllocationRepo) ApplyMasterBillSummary(
	ctx context.Context,
	orgID, actorID, orderID uuid.UUID,
	expectedMblVersion uint64,
	audit *biz.AuditEvent,
) (*biz.SeaMasterBillDetail, error) {
	var updatedMblID uuid.UUID

	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(orgID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return err
		}

		if err := ensureSharedMBLNotLocked(ctx, tx, activeLinkQuery.MasterBillID); err != nil {
			return err
		}
		// 已按 UUID 顺序锁定全部活动成员 Order；此处重读并校验调用订单。
		order, err := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		mbl, err := tx.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbillent.OrganizationIDEQ(orgID),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Status == seamasterbillent.StatusVOIDED {
			return biz.ErrSeaDocumentVoided
		}
		if mbl.Version != expectedMblVersion {
			return biz.ErrSeaMasterBillConflict
		}

		link, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}

		if !seaDocumentLinkMatches(link, orgID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlinkent.DocumentStructureDIRECT {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		cargoItems, err := tx.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(orderID), ordercargoitement.OrganizationIDEQ(orgID)).
			Order(ent.Asc(ordercargoitement.FieldID)).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}

		if len(cargoItems) == 0 {
			return biz.ErrSeaCargoAllocationIncomplete
		}

		var totalPkg int
		totalWeight := decimal.Zero
		totalVol := decimal.Zero
		for _, ci := range cargoItems {
			totalPkg += ci.PackageCount
			totalWeight = totalWeight.Add(decimal.NewFromFloat(ci.GrossWeightKg))
			totalVol = totalVol.Add(decimal.NewFromFloat(ci.VolumeCbm))
		}

		savedMbl, err := tx.SeaMasterBill.UpdateOne(mbl).
			SetPackageCount(totalPkg).
			SetGrossWeightKg(totalWeight.InexactFloat64()).
			SetVolumeCbm(totalVol.InexactFloat64()).
			SetVersion(mbl.Version + 1).
			Save(ctx)
		if err != nil {
			return err
		}
		updatedMblID = savedMbl.ID

		audit.Action = "sea_master_bill.apply_order_cargo_summary"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["master_bill.id"] = mbl.ID.String()
		audit.Details["master_bill.version.old"] = fmt.Sprintf("%d", expectedMblVersion)
		audit.Details["master_bill.version.new"] = fmt.Sprintf("%d", mbl.Version+1)
		audit.Details["applied.package_count"] = fmt.Sprintf("%d", totalPkg)
		audit.Details["applied.gross_weight_kg"] = totalWeight.StringFixed(3)
		audit.Details["applied.volume_cbm"] = totalVol.StringFixed(6)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return (&seaDocumentRepo{data: r.data}).getSeaMasterBillDetailByID(ctx, orgID, updatedMblID)
}

func seaCargoAllocationToBiz(item *ent.SeaCargoAllocation) *biz.SeaCargoAllocation {
	w, _ := decimal.NewFromString(item.GrossWeightKg)
	v, _ := decimal.NewFromString(item.VolumeCbm)
	return &biz.SeaCargoAllocation{
		ID:                    item.ID,
		OrganizationID:        item.OrganizationID,
		OrderID:               item.OrderID,
		MasterBillOrderLinkID: item.MasterBillOrderLinkID,
		CargoItemID:           item.CargoItemID,
		HouseBillID:           item.HouseBillID,
		ContainerID:           item.ContainerID,
		PackageCount:          int32(item.PackageCount),
		GrossWeightKg:         w,
		VolumeCbm:             v,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

var _ biz.SeaCargoAllocationRepo = (*seaCargoAllocationRepo)(nil)
