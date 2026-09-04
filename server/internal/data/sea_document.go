package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	seacargoallocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/seacargoallocation"
	seahousebill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type seaDocumentRepo struct {
	data *Data
}

func NewSeaDocumentRepo(data *Data) biz.SeaDocumentRepo {
	return &seaDocumentRepo{data: data}
}

func (r *seaDocumentRepo) GetSeaOrderDocuments(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaOrderDocuments, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if order.BusinessType != orderent.BusinessTypeSE {
		return nil, biz.ErrOrderBusinessUnsupported
	}

	link, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(orderID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		WithMasterBill().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSeaDocumentNoActiveLink
		}
		return nil, err
	}

	var mblDetail *biz.SeaMasterBillDetail
	if link.Edges.MasterBill != nil {
		mbl := link.Edges.MasterBill
		issuerName, err := r.getPartnerName(ctx, client, organizationID, mbl.IssuerPartnerID)
		if err != nil {
			return nil, err
		}
		activeMemberCount, err := client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.MasterBillIDEQ(mbl.ID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).Count(ctx)
		if err != nil {
			return nil, err
		}
		mblDetail = seaMasterBillToDetail(mbl, issuerName, activeMemberCount)
	}

	hbs, err := client.SeaHouseBill.Query().
		Where(
			seahousebill.OrganizationIDEQ(organizationID),
			seahousebill.OrderIDEQ(orderID),
			seahousebill.MasterBillIDEQ(link.MasterBillID),
		).
		Order(seahousebill.ByCreatedAt(), seahousebill.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	houseBills := make([]*biz.SeaHouseBill, 0, len(hbs))
	for _, hb := range hbs {
		var orgName, partnerName string
		if hb.IssuerOrganizationID != nil {
			orgName, err = r.getOrganizationName(ctx, client, *hb.IssuerOrganizationID)
			if err != nil {
				return nil, err
			}
		}
		if hb.IssuerPartnerID != nil {
			partnerName, err = r.getPartnerName(ctx, client, organizationID, *hb.IssuerPartnerID)
			if err != nil {
				return nil, err
			}
		}
		houseBills = append(houseBills, seaHouseBillToBiz(hb, orgName, partnerName))
	}

	structure := biz.SeaDocumentStructure(link.DocumentStructure)
	var allowedActions []biz.SeaDocumentAction
	switch structure {
	case biz.SeaDocumentStructureDirect:
		allowedActions = []biz.SeaDocumentAction{
			biz.SeaDocumentActionCancelDirect,
			biz.SeaDocumentActionUpdateMasterBillContent,
		}
	case biz.SeaDocumentStructureHouse:
		allowedActions = []biz.SeaDocumentAction{
			biz.SeaDocumentActionAddHouseBill,
			biz.SeaDocumentActionUpdateHouseBill,
			biz.SeaDocumentActionRemoveHouseBill,
			biz.SeaDocumentActionUpdateMasterBillContent,
		}
	default:
		allowedActions = []biz.SeaDocumentAction{
			biz.SeaDocumentActionMarkDirect,
			biz.SeaDocumentActionAddHouseBill,
			biz.SeaDocumentActionUpdateMasterBillContent,
		}
	}

	return &biz.SeaOrderDocuments{
		OrderID:           orderID,
		DocumentStructure: structure,
		LinkVersion:       link.Version,
		MasterBill:        mblDetail,
		HouseBills:        houseBills,
		AllowedActions:    allowedActions,
	}, nil
}

func (r *seaDocumentRepo) GetSummariesByOrderIDs(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*biz.SeaOrderDocumentSummary, error) {
	result := make(map[uuid.UUID]*biz.SeaOrderDocumentSummary, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	links, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDIn(orderIDs...),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	linkMap := make(map[uuid.UUID]*ent.SeaMasterBillOrderLink, len(links))
	var mblIDs []uuid.UUID
	for _, link := range links {
		linkMap[link.OrderID] = link
		mblIDs = append(mblIDs, link.MasterBillID)
	}

	if len(mblIDs) == 0 {
		return result, nil
	}

	hbs, err := client.SeaHouseBill.Query().
		Where(
			seahousebill.OrganizationIDEQ(organizationID),
			seahousebill.OrderIDIn(orderIDs...),
			seahousebill.MasterBillIDIn(mblIDs...),
		).
		Order(seahousebill.ByCreatedAt(), seahousebill.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	hbMap := make(map[uuid.UUID][]string)
	for _, hb := range hbs {
		if activeLink, ok := linkMap[hb.OrderID]; ok && activeLink.MasterBillID == hb.MasterBillID {
			hbMap[hb.OrderID] = append(hbMap[hb.OrderID], hb.HouseNo)
		}
	}

	for _, link := range links {
		houseNos := hbMap[link.OrderID]
		result[link.OrderID] = &biz.SeaOrderDocumentSummary{
			DocumentStructure: biz.SeaDocumentStructure(link.DocumentStructure),
			LinkVersion:       link.Version,
			HouseBillCount:    len(houseNos),
			HouseNos:          houseNos,
		}
	}

	return result, nil
}

func (r *seaDocumentRepo) MarkSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *biz.AuditEvent) (*biz.SeaOrderDocuments, error) {
	if actorID == uuid.Nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return nil, biz.ErrSeaDocumentInvalidArgument
	}

	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 固定锁顺序：Order
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		// 定位活动 link
		activeLinkQuery, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.OrderIDEQ(orderID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return queryErr
		}

		// 2. 固定锁顺序：MasterBill
		_, queryErr = tx.SeaMasterBill.Query().
			Where(
				seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}

		// 3. 固定锁顺序：Active Link
		link, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlink.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if !seaDocumentLinkMatches(link, organizationID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}

		if link.Version != expectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlink.DocumentStructureUNDETERMINED {
			return biz.ErrSeaDocumentStructureInvalid
		}

		hbCount, err := tx.SeaHouseBill.Query().
			Where(
				seahousebill.OrganizationIDEQ(organizationID),
				seahousebill.OrderIDEQ(orderID),
				seahousebill.MasterBillIDEQ(link.MasterBillID),
			).Count(ctx)
		if err != nil {
			return err
		}
		if hbCount > 0 {
			return biz.ErrSeaDocumentStructureInvalid
		}

		if link.CargoAllocationStatus == seamasterbillorderlink.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}
		hasAlloc, err := tx.SeaCargoAllocation.Query().
			Where(seacargoallocation.MasterBillOrderLinkIDEQ(link.ID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasAlloc {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		if _, err := link.Update().
			SetDocumentStructure(seamasterbillorderlink.DocumentStructureDIRECT).
			SetVersion(link.Version + 1).
			Save(ctx); err != nil {
			return err
		}

		audit.Action = "order.sea_document.mark_direct"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["document_structure.old"] = string(link.DocumentStructure)
		audit.Details["document_structure.new"] = string(biz.SeaDocumentStructureDirect)
		audit.Details["link.old_version"] = fmt.Sprintf("%d", link.Version)
		audit.Details["link.new_version"] = fmt.Sprintf("%d", link.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.GetSeaOrderDocuments(ctx, organizationID, orderID)
}

func (r *seaDocumentRepo) CancelSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *biz.AuditEvent) (*biz.SeaOrderDocuments, error) {
	if actorID == uuid.Nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return nil, biz.ErrSeaDocumentInvalidArgument
	}

	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 固定锁顺序：Order
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		activeLinkQuery, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.OrderIDEQ(orderID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return queryErr
		}

		// 2. 固定锁顺序：MasterBill
		_, queryErr = tx.SeaMasterBill.Query().
			Where(
				seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}

		// 3. 固定锁顺序：Active Link
		link, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlink.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if !seaDocumentLinkMatches(link, organizationID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}

		if link.Version != expectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.DocumentStructure != seamasterbillorderlink.DocumentStructureDIRECT {
			return biz.ErrSeaDocumentStructureInvalid
		}

		if _, err := link.Update().
			SetDocumentStructure(seamasterbillorderlink.DocumentStructureUNDETERMINED).
			SetVersion(link.Version + 1).
			SetCargoAllocationVersion(link.CargoAllocationVersion + 1).
			Save(ctx); err != nil {
			return err
		}

		audit.Action = "order.sea_document.cancel_direct"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["document_structure.old"] = string(link.DocumentStructure)
		audit.Details["document_structure.new"] = string(biz.SeaDocumentStructureUndetermined)
		audit.Details["link.old_version"] = fmt.Sprintf("%d", link.Version)
		audit.Details["link.new_version"] = fmt.Sprintf("%d", link.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.GetSeaOrderDocuments(ctx, organizationID, orderID)
}

func (r *seaDocumentRepo) AddSeaHouseBill(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, input *biz.SeaHouseBillInput, audit *biz.AuditEvent) (*biz.SeaHouseBill, error) {
	if actorID == uuid.Nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return nil, biz.ErrSeaDocumentInvalidArgument
	}

	normalized, err := biz.NormalizeSeaHouseNo(input.HouseNo)
	if err != nil {
		return nil, err
	}
	normalizedContent, err := biz.ValidateSeaBillContent(input.Content)
	if err != nil {
		return nil, err
	}

	var createdID uuid.UUID

	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 固定锁顺序：Order
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}

		activeLinkQuery, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.OrderIDEQ(orderID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return queryErr
		}

		// 2. 固定锁顺序：MasterBill
		mbl, queryErr := tx.SeaMasterBill.Query().
			Where(
				seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}

		// 3. 固定锁顺序：Active Link
		link, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlink.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if !seaDocumentLinkMatches(link, organizationID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}

		if link.Version != expectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}

		// DIRECT 状态下禁止直接添加 HBL
		if link.DocumentStructure == seamasterbillorderlink.DocumentStructureDIRECT {
			return biz.ErrSeaDocumentDirectAddHBLBlocked
		}
		if link.CargoAllocationStatus == seamasterbillorderlink.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		// 4. 校验签发主体
		issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), organizationID, order.OrganizationID, order.CustomerID, input)
		if err != nil {
			return err
		}

		createdID = uuid.Must(uuid.NewV7())
		builder := tx.SeaHouseBill.Create().
			SetID(createdID).
			SetOrganizationID(organizationID).
			SetOrderID(orderID).
			SetMasterBillID(mbl.ID).
			SetHouseNo(input.HouseNo).
			SetNormalizedHouseNo(normalized).
			SetIssuerSource(seahousebill.IssuerSource(input.IssuerSource)).
			SetStatus(seahousebill.StatusDRAFT).
			SetVersion(1)

		if issuerOrgID != nil {
			builder.SetIssuerOrganizationID(*issuerOrgID)
		}
		if issuerPartnerID != nil {
			builder.SetIssuerPartnerID(*issuerPartnerID)
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		}
		setSeaHouseBillContentCreate(builder, normalizedContent)

		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrSeaHouseBillExists
			}
			return err
		}

		// 结构转换：若当前为 UNDETERMINED，则转为 HOUSE
		linkUpdate := link.Update().SetVersion(link.Version + 1).SetCargoAllocationVersion(link.CargoAllocationVersion + 1)
		if link.DocumentStructure == seamasterbillorderlink.DocumentStructureUNDETERMINED {
			linkUpdate.SetDocumentStructure(seamasterbillorderlink.DocumentStructureHOUSE)
		}
		if _, err := linkUpdate.Save(ctx); err != nil {
			return err
		}

		audit.Action = "sea_house_bill.add"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["sea_house_bill.id"] = createdID.String()
		audit.Details["sea_house_bill.house_no"] = input.HouseNo
		audit.Details["sea_house_bill.issuer_source"] = string(input.IssuerSource)
		audit.Details["link.new_version"] = fmt.Sprintf("%d", link.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.getSeaHouseBillByID(ctx, organizationID, createdID)
}

func (r *seaDocumentRepo) UpdateSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, input *biz.SeaHouseBillInput, audit *biz.AuditEvent) (*biz.SeaHouseBill, error) {
	if actorID == uuid.Nil {
		return nil, biz.ErrSeaHouseBillInvalidArgument
	}
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return nil, biz.ErrSeaDocumentInvalidArgument
	}

	normalized, err := biz.NormalizeSeaHouseNo(input.HouseNo)
	if err != nil {
		return nil, err
	}
	normalizedContent, err := biz.ValidateSeaBillContent(input.Content)
	if err != nil {
		return nil, err
	}

	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 固定锁顺序：Order
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		activeLinkQuery, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.OrderIDEQ(orderID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return queryErr
		}

		// 2. 固定锁顺序：MasterBill
		mbl, queryErr := tx.SeaMasterBill.Query().
			Where(
				seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}

		// 3. 固定锁顺序：Active Link
		link, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlink.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if !seaDocumentLinkMatches(link, organizationID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.Version != expectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.CargoAllocationStatus == seamasterbillorderlink.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}

		// 4. 固定锁顺序：SeaHouseBill
		hb, queryErr := tx.SeaHouseBill.Query().
			Where(
				seahousebill.IDEQ(houseBillID),
				seahousebill.OrderIDEQ(orderID),
				seahousebill.MasterBillIDEQ(mbl.ID),
				seahousebill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaHouseBillNotFound, nil)
		}

		if hb.Version != expectedVersion {
			return biz.ErrSeaHouseBillConflict
		}
		if hb.Status == seahousebill.StatusRELEASED {
			return biz.ErrSeaHouseBillStatusConflict
		}

		// 5. 校验签发主体
		issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), organizationID, order.OrganizationID, order.CustomerID, input)
		if err != nil {
			return err
		}

		updater := hb.Update().
			SetHouseNo(input.HouseNo).
			SetNormalizedHouseNo(normalized).
			SetIssuerSource(seahousebill.IssuerSource(input.IssuerSource)).
			SetVersion(hb.Version + 1)

		if issuerOrgID != nil {
			updater.SetIssuerOrganizationID(*issuerOrgID).ClearIssuerPartnerID()
		} else if issuerPartnerID != nil {
			updater.SetIssuerPartnerID(*issuerPartnerID).ClearIssuerOrganizationID()
		}

		if input.Note != nil {
			updater.SetNote(*input.Note)
		} else {
			updater.ClearNote()
		}
		setSeaHouseBillContentUpdate(updater, normalizedContent)

		if _, err := updater.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrSeaHouseBillExists
			}
			return err
		}

		if _, err := link.Update().SetVersion(link.Version + 1).Save(ctx); err != nil {
			return err
		}

		audit.Action = "sea_house_bill.update"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["sea_house_bill.id"] = houseBillID.String()
		audit.Details["sea_house_bill.old_version"] = fmt.Sprintf("%d", hb.Version)
		audit.Details["sea_house_bill.new_version"] = fmt.Sprintf("%d", hb.Version+1)
		audit.Details["link.new_version"] = fmt.Sprintf("%d", link.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.getSeaHouseBillByID(ctx, organizationID, houseBillID)
}

func (r *seaDocumentRepo) RemoveSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, returnToUndetermined bool, audit *biz.AuditEvent) error {
	if actorID == uuid.Nil {
		return biz.ErrSeaHouseBillInvalidArgument
	}
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return biz.ErrSeaDocumentInvalidArgument
	}

	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 固定锁顺序：Order
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		activeLinkQuery, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.OrderIDEQ(orderID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return queryErr
		}

		// 2. 固定锁顺序：MasterBill
		mbl, queryErr := tx.SeaMasterBill.Query().
			Where(
				seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}

		// 3. 固定锁顺序：Active Link
		link, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlink.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if !seaDocumentLinkMatches(link, organizationID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}
		if link.Version != expectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}

		// 4. 固定锁顺序：SeaHouseBill
		hb, queryErr := tx.SeaHouseBill.Query().
			Where(
				seahousebill.IDEQ(houseBillID),
				seahousebill.OrderIDEQ(orderID),
				seahousebill.MasterBillIDEQ(mbl.ID),
				seahousebill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaHouseBillNotFound, nil)
		}
		if hb.Version != expectedVersion {
			return biz.ErrSeaHouseBillConflict
		}
		if hb.Status == seahousebill.StatusRELEASED {
			return biz.ErrSeaHouseBillStatusConflict
		}
		if link.CargoAllocationStatus == seamasterbillorderlink.CargoAllocationStatusCONFIRMED {
			return biz.ErrSeaCargoAllocationStatusConflict
		}
		hasAlloc, err := tx.SeaCargoAllocation.Query().
			Where(seacargoallocation.HouseBillIDEQ(houseBillID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasAlloc {
			return biz.ErrSeaCargoAllocationInvalidReference
		}

		// 5. 统计当前 MBL 下该订单剩余 HBL 数量
		count, err := tx.SeaHouseBill.Query().
			Where(
				seahousebill.OrganizationIDEQ(organizationID),
				seahousebill.OrderIDEQ(orderID),
				seahousebill.MasterBillIDEQ(mbl.ID),
			).Count(ctx)
		if err != nil {
			return err
		}

		linkUpdate := link.Update().SetVersion(link.Version + 1).SetCargoAllocationVersion(link.CargoAllocationVersion + 1)
		if count == 1 {
			// 最后一张 HBL 删除，必须显式确认回到未确定
			if !returnToUndetermined {
				return biz.ErrSeaDocumentDeleteLastHBLConfirmationRequired
			}
			linkUpdate.SetDocumentStructure(seamasterbillorderlink.DocumentStructureUNDETERMINED)
		}
		if _, err := linkUpdate.Save(ctx); err != nil {
			return err
		}

		if err := tx.SeaHouseBill.DeleteOneID(houseBillID).Exec(ctx); err != nil {
			return err
		}

		audit.Action = "sea_house_bill.remove"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["sea_house_bill.id"] = houseBillID.String()
		audit.Details["sea_house_bill.house_no"] = hb.HouseNo
		audit.Details["link.new_version"] = fmt.Sprintf("%d", link.Version+1)
		if count == 1 {
			audit.Details["document_structure.new"] = string(biz.SeaDocumentStructureUndetermined)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *seaDocumentRepo) UpdateSeaMasterBillContent(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedMblVersion uint64, content *biz.SeaBillContent, audit *biz.AuditEvent) (*biz.SeaMasterBillDetail, error) {
	if actorID == uuid.Nil {
		return nil, biz.ErrSeaMasterBillInvalidArgument
	}
	if audit == nil || audit.OrganizationID == nil || *audit.OrganizationID != organizationID || audit.UserID == nil || *audit.UserID != actorID {
		return nil, biz.ErrSeaDocumentInvalidArgument
	}

	normalizedContent, err := biz.ValidateSeaBillContent(content)
	if err != nil {
		return nil, err
	}

	var mblID uuid.UUID

	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		activeLinkQuery, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.OrderIDEQ(orderID),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrSeaDocumentNoActiveLink
			}
			return queryErr
		}

		if err := ensureSharedMBLNotLocked(ctx, tx, activeLinkQuery.MasterBillID); err != nil {
			return err
		}
		// ensureSharedMBLNotLocked 已按 UUID 顺序锁定全部活动成员 Order；此处重读并校验调用订单。
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrOrderBusinessUnsupported
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		// 2. 固定锁顺序：全部成员 Order → MasterBill
		mbl, queryErr := tx.SeaMasterBill.Query().
			Where(
				seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
				seamasterbill.OrganizationIDEQ(organizationID),
			).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Version != expectedMblVersion {
			return biz.ErrSeaMasterBillConflict
		}

		// 3. 固定锁顺序：Active Link
		link, queryErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlink.IDEQ(activeLinkQuery.ID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrSeaMasterBillNotFound, nil)
		}
		if !seaDocumentLinkMatches(link, organizationID, orderID, activeLinkQuery.MasterBillID) {
			return biz.ErrSeaDocumentStructureConflict
		}

		mblID = mbl.ID

		updater := mbl.Update().SetVersion(mbl.Version + 1)
		setSeaMasterBillContent(updater, normalizedContent)
		if _, err := updater.Save(ctx); err != nil {
			return err
		}

		audit.Action = "sea_master_bill.content.update"
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["order.id"] = orderID.String()
		audit.Details["sea_master_bill.id"] = mbl.ID.String()
		audit.Details["sea_master_bill.old_version"] = fmt.Sprintf("%d", mbl.Version)
		audit.Details["sea_master_bill.new_version"] = fmt.Sprintf("%d", mbl.Version+1)
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}

	return r.getSeaMasterBillDetailByID(ctx, organizationID, mblID)
}

func (r *seaDocumentRepo) getSeaMasterBillDetailByID(ctx context.Context, organizationID, mblID uuid.UUID) (*biz.SeaMasterBillDetail, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	mbl, err := client.SeaMasterBill.Query().
		Where(seamasterbill.IDEQ(mblID), seamasterbill.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	issuerName, err := r.getPartnerName(ctx, client, organizationID, mbl.IssuerPartnerID)
	if err != nil {
		return nil, err
	}
	memberCount, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.MasterBillIDEQ(mbl.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).Count(ctx)
	if err != nil {
		return nil, err
	}

	return seaMasterBillToDetail(mbl, issuerName, memberCount), nil
}

func seaDocumentLinkMatches(link *ent.SeaMasterBillOrderLink, organizationID, orderID, masterBillID uuid.UUID) bool {
	return link != nil &&
		link.OrganizationID == organizationID &&
		link.OrderID == orderID &&
		link.MasterBillID == masterBillID &&
		link.Status == seamasterbillorderlink.StatusACTIVE
}

func (r *seaDocumentRepo) getSeaHouseBillByID(ctx context.Context, organizationID, id uuid.UUID) (*biz.SeaHouseBill, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	hb, err := client.SeaHouseBill.Query().
		Where(seahousebill.IDEQ(id), seahousebill.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaHouseBillNotFound, nil)
	}

	var orgName, partnerName string
	if hb.IssuerOrganizationID != nil {
		orgName, err = r.getOrganizationName(ctx, client, *hb.IssuerOrganizationID)
		if err != nil {
			return nil, err
		}
	}
	if hb.IssuerPartnerID != nil {
		partnerName, err = r.getPartnerName(ctx, client, organizationID, *hb.IssuerPartnerID)
		if err != nil {
			return nil, err
		}
	}

	return seaHouseBillToBiz(hb, orgName, partnerName), nil
}

func (r *seaDocumentRepo) getPartnerName(ctx context.Context, client *ent.Client, organizationID, partnerID uuid.UUID) (string, error) {
	if partnerID == uuid.Nil {
		return "", nil
	}
	partner, err := client.Partner.Query().Where(
		partnerent.IDEQ(partnerID),
		partnerent.OrganizationIDEQ(organizationID),
	).Only(ctx)
	if err != nil {
		return "", mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	return partner.LegalName, nil
}

func (r *seaDocumentRepo) getOrganizationName(ctx context.Context, client *ent.Client, orgID uuid.UUID) (string, error) {
	if orgID == uuid.Nil {
		return "", nil
	}
	org, err := client.Organization.Query().Where(organizationent.IDEQ(orgID)).Only(ctx)
	if err != nil {
		return "", err
	}
	return org.Name, nil
}

func resolveIssuerOrganization(ctx context.Context, client *ent.Client, rootOrgID, orderOrgID uuid.UUID) (uuid.UUID, error) {
	currentID := orderOrgID
	visited := make(map[uuid.UUID]struct{})
	for currentID != uuid.Nil {
		if _, exists := visited[currentID]; exists {
			break
		}
		visited[currentID] = struct{}{}

		org, err := client.Organization.Query().
			Where(organizationent.IDEQ(currentID)).
			Only(ctx)
		if err != nil {
			return uuid.Nil, biz.ErrSeaDocumentIssuerOrgNotFound
		}
		if org.Kind == organizationent.KindCompany || org.Kind == organizationent.KindHeadquarters {
			return org.ID, nil
		}
		if org.ParentID == nil || *org.ParentID == uuid.Nil {
			break
		}
		currentID = *org.ParentID
	}
	return uuid.Nil, biz.ErrSeaDocumentIssuerOrgNotFound
}

func validateSeaHouseBillIssuer(ctx context.Context, client *ent.Client, organizationID, orderOrgID, customerID uuid.UUID, input *biz.SeaHouseBillInput) (*uuid.UUID, *uuid.UUID, error) {
	switch input.IssuerSource {
	case biz.SeaHouseBillIssuerSourceSelfOrganization:
		issuerOrgID, err := resolveIssuerOrganization(ctx, client, organizationID, orderOrgID)
		if err != nil {
			return nil, nil, err
		}
		return &issuerOrgID, nil, nil
	case biz.SeaHouseBillIssuerSourceCustomerPartner:
		if customerID == uuid.Nil {
			return nil, nil, biz.ErrSeaHouseBillInvalidArgument
		}
		exists, err := client.Partner.Query().Where(
			partnerent.IDEQ(customerID),
			partnerent.OrganizationIDEQ(organizationID),
			partnerent.EnabledEQ(true),
		).Exist(ctx)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			return nil, nil, biz.ErrSeaHouseBillInvalidArgument
		}
		return nil, &customerID, nil
	case biz.SeaHouseBillIssuerSourceOtherPartner:
		if input.IssuerPartnerID == nil || *input.IssuerPartnerID == uuid.Nil {
			return nil, nil, biz.ErrSeaHouseBillInvalidArgument
		}
		partnerID := *input.IssuerPartnerID
		exists, err := client.Partner.Query().Where(
			partnerent.IDEQ(partnerID),
			partnerent.OrganizationIDEQ(organizationID),
			partnerent.EnabledEQ(true),
		).Exist(ctx)
		if err != nil {
			return nil, nil, err
		}
		if !exists {
			return nil, nil, biz.ErrSeaHouseBillInvalidArgument
		}
		return nil, &partnerID, nil
	default:
		return nil, nil, biz.ErrSeaHouseBillInvalidArgument
	}
}

func setSeaMasterBillContent(builder *ent.SeaMasterBillUpdateOne, content *biz.SeaBillContent) {
	if content == nil {
		return
	}
	setOptionalText(builder.SetShipperText, builder.ClearShipperText, content.ShipperText)
	setOptionalText(builder.SetConsigneeText, builder.ClearConsigneeText, content.ConsigneeText)
	setOptionalText(builder.SetNotifyPartyText, builder.ClearNotifyPartyText, content.NotifyPartyText)
	setOptionalText(builder.SetSecondNotifyPartyText, builder.ClearSecondNotifyPartyText, content.SecondNotifyPartyText)
	setOptionalText(builder.SetMarksText, builder.ClearMarksText, content.MarksText)
	setOptionalText(builder.SetGoodsDescriptionText, builder.ClearGoodsDescriptionText, content.GoodsDescriptionText)
	setOptionalInt(builder.SetPackageCount, builder.ClearPackageCount, content.PackageCount)
	setOptionalString(builder.SetPackageUnit, builder.ClearPackageUnit, content.PackageUnit)
	setOptionalFloat(builder.SetGrossWeightKg, builder.ClearGrossWeightKg, content.GrossWeightKg)
	setOptionalFloat(builder.SetVolumeCbm, builder.ClearVolumeCbm, content.VolumeCbm)
	setOptionalString(builder.SetFreightTerms, builder.ClearFreightTerms, content.FreightTerms)
	setOptionalString(builder.SetTransportTerms, builder.ClearTransportTerms, content.TransportTerms)
	setOptionalString(builder.SetBillForm, builder.ClearBillForm, content.BillForm)
	setOptionalString(builder.SetReleaseType, builder.ClearReleaseType, content.ReleaseType)
	setOptionalText(builder.SetClauses, builder.ClearClauses, content.Clauses)
}

func setSeaHouseBillContentCreate(builder *ent.SeaHouseBillCreate, content *biz.SeaBillContent) {
	if content == nil {
		return
	}
	if content.ShipperText != nil {
		builder.SetShipperText(*content.ShipperText)
	}
	if content.ConsigneeText != nil {
		builder.SetConsigneeText(*content.ConsigneeText)
	}
	if content.NotifyPartyText != nil {
		builder.SetNotifyPartyText(*content.NotifyPartyText)
	}
	if content.SecondNotifyPartyText != nil {
		builder.SetSecondNotifyPartyText(*content.SecondNotifyPartyText)
	}
	if content.MarksText != nil {
		builder.SetMarksText(*content.MarksText)
	}
	if content.GoodsDescriptionText != nil {
		builder.SetGoodsDescriptionText(*content.GoodsDescriptionText)
	}
	if content.PackageCount != nil {
		builder.SetPackageCount(int(*content.PackageCount))
	}
	if content.PackageUnit != nil {
		builder.SetPackageUnit(*content.PackageUnit)
	}
	if content.GrossWeightKg != nil {
		builder.SetGrossWeightKg(*content.GrossWeightKg)
	}
	if content.VolumeCbm != nil {
		builder.SetVolumeCbm(*content.VolumeCbm)
	}
	if content.FreightTerms != nil {
		builder.SetFreightTerms(*content.FreightTerms)
	}
	if content.TransportTerms != nil {
		builder.SetTransportTerms(*content.TransportTerms)
	}
	if content.BillForm != nil {
		builder.SetBillForm(*content.BillForm)
	}
	if content.ReleaseType != nil {
		builder.SetReleaseType(*content.ReleaseType)
	}
	if content.Clauses != nil {
		builder.SetClauses(*content.Clauses)
	}
}

func setSeaHouseBillContentUpdate(builder *ent.SeaHouseBillUpdateOne, content *biz.SeaBillContent) {
	if content == nil {
		return
	}
	setOptionalText(builder.SetShipperText, builder.ClearShipperText, content.ShipperText)
	setOptionalText(builder.SetConsigneeText, builder.ClearConsigneeText, content.ConsigneeText)
	setOptionalText(builder.SetNotifyPartyText, builder.ClearNotifyPartyText, content.NotifyPartyText)
	setOptionalText(builder.SetSecondNotifyPartyText, builder.ClearSecondNotifyPartyText, content.SecondNotifyPartyText)
	setOptionalText(builder.SetMarksText, builder.ClearMarksText, content.MarksText)
	setOptionalText(builder.SetGoodsDescriptionText, builder.ClearGoodsDescriptionText, content.GoodsDescriptionText)
	setOptionalInt(builder.SetPackageCount, builder.ClearPackageCount, content.PackageCount)
	setOptionalString(builder.SetPackageUnit, builder.ClearPackageUnit, content.PackageUnit)
	setOptionalFloat(builder.SetGrossWeightKg, builder.ClearGrossWeightKg, content.GrossWeightKg)
	setOptionalFloat(builder.SetVolumeCbm, builder.ClearVolumeCbm, content.VolumeCbm)
	setOptionalString(builder.SetFreightTerms, builder.ClearFreightTerms, content.FreightTerms)
	setOptionalString(builder.SetTransportTerms, builder.ClearTransportTerms, content.TransportTerms)
	setOptionalString(builder.SetBillForm, builder.ClearBillForm, content.BillForm)
	setOptionalString(builder.SetReleaseType, builder.ClearReleaseType, content.ReleaseType)
	setOptionalText(builder.SetClauses, builder.ClearClauses, content.Clauses)
}

func setOptionalText[T any](set func(string) T, clear func() T, val *string) {
	if val == nil {
		clear()
	} else {
		set(*val)
	}
}

func setOptionalString[T any](set func(string) T, clear func() T, val *string) {
	if val == nil {
		clear()
	} else {
		set(*val)
	}
}

func setOptionalInt[T any](set func(int) T, clear func() T, val *int32) {
	if val == nil {
		clear()
	} else {
		set(int(*val))
	}
}

func setOptionalFloat[T any](set func(float64) T, clear func() T, val *float64) {
	if val == nil {
		clear()
	} else {
		set(*val)
	}
}

func seaHouseBillToBiz(item *ent.SeaHouseBill, orgName, partnerName string) *biz.SeaHouseBill {
	if item == nil {
		return nil
	}
	var count *int32
	if item.PackageCount != nil {
		c := int32(*item.PackageCount)
		count = &c
	}
	content := &biz.SeaBillContent{
		ShipperText:           item.ShipperText,
		ConsigneeText:         item.ConsigneeText,
		NotifyPartyText:       item.NotifyPartyText,
		SecondNotifyPartyText: item.SecondNotifyPartyText,
		MarksText:             item.MarksText,
		GoodsDescriptionText:  item.GoodsDescriptionText,
		PackageCount:          count,
		PackageUnit:           item.PackageUnit,
		GrossWeightKg:         item.GrossWeightKg,
		VolumeCbm:             item.VolumeCbm,
		FreightTerms:          item.FreightTerms,
		TransportTerms:        item.TransportTerms,
		BillForm:              item.BillForm,
		ReleaseType:           item.ReleaseType,
		Clauses:               item.Clauses,
	}
	return &biz.SeaHouseBill{
		ID:                     item.ID,
		OrganizationID:         item.OrganizationID,
		OrderID:                item.OrderID,
		MasterBillID:           item.MasterBillID,
		HouseNo:                item.HouseNo,
		NormalizedHouseNo:      item.NormalizedHouseNo,
		IssuerSource:           biz.SeaHouseBillIssuerSource(item.IssuerSource),
		IssuerOrganizationID:   item.IssuerOrganizationID,
		IssuerOrganizationName: orgName,
		IssuerPartnerID:        item.IssuerPartnerID,
		IssuerPartnerName:      partnerName,
		Status:                 biz.SeaHouseBillStatus(item.Status),
		Version:                item.Version,
		Note:                   item.Note,
		Content:                content,
		CreatedAt:              item.CreatedAt,
		UpdatedAt:              item.UpdatedAt,
	}
}

func seaMasterBillToDetail(item *ent.SeaMasterBill, issuerName string, memberCount int) *biz.SeaMasterBillDetail {
	if item == nil {
		return nil
	}
	var count *int32
	if item.PackageCount != nil {
		c := int32(*item.PackageCount)
		count = &c
	}
	content := &biz.SeaBillContent{
		ShipperText:           item.ShipperText,
		ConsigneeText:         item.ConsigneeText,
		NotifyPartyText:       item.NotifyPartyText,
		SecondNotifyPartyText: item.SecondNotifyPartyText,
		MarksText:             item.MarksText,
		GoodsDescriptionText:  item.GoodsDescriptionText,
		PackageCount:          count,
		PackageUnit:           item.PackageUnit,
		GrossWeightKg:         item.GrossWeightKg,
		VolumeCbm:             item.VolumeCbm,
		FreightTerms:          item.FreightTerms,
		TransportTerms:        item.TransportTerms,
		BillForm:              item.BillForm,
		ReleaseType:           item.ReleaseType,
		Clauses:               item.Clauses,
	}
	return &biz.SeaMasterBillDetail{
		ID:                item.ID,
		MasterNo:          item.MasterNo,
		IssuerPartnerID:   item.IssuerPartnerID,
		IssuerPartnerName: issuerName,
		Status:            string(item.Status),
		Version:           item.Version,
		Content:           content,
		MemberCount:       memberCount,
	}
}
