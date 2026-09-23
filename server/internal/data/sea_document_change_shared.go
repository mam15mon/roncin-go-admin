package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecommissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	financecommissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
)

func loadAmendmentPreview(ctx context.Context, client *ent.Client, orgID uuid.UUID, input *biz.SeaDocumentAmendmentCommand) (*biz.SeaDocumentVersion, []*biz.SeaDocumentFieldDifference, []uuid.UUID, error) {
	base, orderIDs, err := loadCurrentDocumentBase(ctx, client, orgID, input.OrderID, input.DocumentType, input.DocumentID, input.ExpectedOrderVersion, input.ExpectedDocumentVersion, input.ExpectedCurrentVersionID)
	if err != nil {
		return nil, nil, nil, err
	}
	var diffs []*biz.SeaDocumentFieldDifference
	if input.DocumentType == biz.SeaDocumentTypeMasterBill {
		diffs = diffContent(base.Content, input.Input.MasterBillContent)
	} else {
		issuerOrgID, issuerPartnerID, err := resolveHouseBillIssuerForDiff(ctx, client, orgID, input.OrderID, input.Input.HouseBill)
		if err != nil {
			return nil, nil, nil, err
		}
		diffs = diffHouseVersionToInput(base, input.Input.HouseBill, issuerOrgID, issuerPartnerID)
	}
	if len(diffs) == 0 {
		return nil, nil, nil, biz.ErrSeaDocumentAmendmentEmpty
	}
	return base, diffs, orderIDs, nil
}

func loadCurrentDocumentBase(ctx context.Context, client *ent.Client, orgID, orderID uuid.UUID, documentType biz.SeaDocumentType, documentID uuid.UUID, expectedOrderVersion, expectedDocumentVersion uint64, expectedCurrentVersionID uuid.UUID) (*biz.SeaDocumentVersion, []uuid.UUID, error) {
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if order.Version != expectedOrderVersion {
		return nil, nil, biz.ErrOrderStatusConflict
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	switch documentType {
	case biz.SeaDocumentTypeMasterBill:
		if link.MasterBillID != documentID {
			return nil, nil, biz.ErrSeaDocumentVersionConflict
		}
		mbl, err := client.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(documentID), seamasterbillent.OrganizationIDEQ(orgID)).Only(ctx)
		if err != nil {
			return nil, nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		if mbl.Status == seamasterbillent.StatusVOIDED {
			return nil, nil, biz.ErrSeaDocumentVoided
		}
		if mbl.Version != expectedDocumentVersion || mbl.CurrentVersionID == nil || *mbl.CurrentVersionID != expectedCurrentVersionID {
			return nil, nil, biz.ErrSeaDocumentVersionConflict
		}
		version, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.IDEQ(expectedCurrentVersionID), seamasterbillversionent.MasterBillIDEQ(mbl.ID), seamasterbillversionent.OrganizationIDEQ(orgID)).WithShippingLine().Only(ctx)
		if err != nil {
			return nil, nil, mapEntError(err, biz.ErrSeaDocumentVersionNotFound, nil)
		}
		members, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.MasterBillIDEQ(mbl.ID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).All(ctx)
		if err != nil {
			return nil, nil, err
		}
		ids := make([]uuid.UUID, 0, len(members))
		for _, member := range members {
			ids = append(ids, member.OrderID)
		}
		ids = sortAndDeduplicateUUIDs(ids)
		return masterVersionToBiz(version, orderID), ids, nil
	case biz.SeaDocumentTypeHouseBill:
		hbl, err := client.SeaHouseBill.Query().Where(seahousebillent.IDEQ(documentID), seahousebillent.OrganizationIDEQ(orgID), seahousebillent.OrderIDEQ(orderID), seahousebillent.MasterBillIDEQ(link.MasterBillID)).Only(ctx)
		if err != nil {
			return nil, nil, mapEntError(err, biz.ErrSeaHouseBillNotFound, nil)
		}
		if hbl.Status == seahousebillent.StatusVOIDED {
			return nil, nil, biz.ErrSeaDocumentVoided
		}
		if hbl.Version != expectedDocumentVersion || hbl.CurrentVersionID == nil || *hbl.CurrentVersionID != expectedCurrentVersionID {
			return nil, nil, biz.ErrSeaDocumentVersionConflict
		}
		version, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.IDEQ(expectedCurrentVersionID), seahousebillversionent.HouseBillIDEQ(hbl.ID), seahousebillversionent.OrderIDEQ(orderID), seahousebillversionent.OrganizationIDEQ(orgID)).Only(ctx)
		if err != nil {
			return nil, nil, mapEntError(err, biz.ErrSeaDocumentVersionNotFound, nil)
		}
		return houseVersionToBiz(version), []uuid.UUID{orderID}, nil
	default:
		return nil, nil, biz.ErrSeaDocumentInvalidArgument
	}
}

// collectDocumentImpacts 收集目标订单集合的六类下游财务事实；houseBillID 非空时
// 表示 HBL 变更上下文，额外查询该 HBL 的共享箱货分配。所有事实均为不可自动调整
// 的阻断事实（BlocksExecution=true），由 hasBlockingImpact 统一判定。
func collectDocumentImpacts(ctx context.Context, client *ent.Client, orgID uuid.UUID, orderIDs []uuid.UUID, houseBillID uuid.UUID) ([]*biz.SeaDocumentDownstreamImpact, error) {
	if len(orderIDs) == 0 {
		return nil, nil
	}
	impacts := make([]*biz.SeaDocumentDownstreamImpact, 0)
	fees, err := client.OrderFee.Query().Where(orderfeeent.OrderIDIn(orderIDs...), orderfeeent.StatusNotIn(orderfeeent.StatusUNBILLED, orderfeeent.StatusCANCELLED)).Order(orderfeeent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, fee := range fees {
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "ORDER_FEE", ReferenceID: fee.ID.String(), ReferenceNo: fee.FeeCode, Message: "费用 " + fee.FeeCode + " 已建账，变更不会改写该事实", BlocksExecution: true})
	}
	lines, err := client.FinanceBillLine.Query().Where(financebilllineent.OrderIDIn(orderIDs...), financebilllineent.ActiveEQ(true)).WithBill().Order(financebilllineent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		no := line.ID.String()
		if line.Edges.Bill != nil {
			no = line.Edges.Bill.BillNo
		}
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "FINANCE_BILL", ReferenceID: line.BillID.String(), ReferenceNo: no, Message: "账单 " + no + " 已引用订单费用，变更不会改写该事实", BlocksExecution: true})
	}
	invoices, err := client.FinanceInvoice.Query().Where(
		financeinvoiceent.OrganizationIDEQ(orgID),
		financeinvoiceent.HasBillLinksWith(financeinvoicebillent.HasBillWith(financebillent.HasLinesWith(financebilllineent.OrderIDIn(orderIDs...)))),
	).Order(financeinvoiceent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, invoice := range invoices {
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "FINANCE_INVOICE", ReferenceID: invoice.ID.String(), ReferenceNo: invoice.RecordNo, Message: "发票 " + invoice.RecordNo + " 已形成开票事实，变更不会改写该事实", BlocksExecution: true})
	}
	verifications, err := client.FinanceVerification.Query().Where(
		financeverificationent.OrganizationIDEQ(orgID),
		financeverificationent.HasAllocationsWith(financeverificationallocationent.HasBillWith(financebillent.HasLinesWith(financebilllineent.OrderIDIn(orderIDs...)))),
	).Order(financeverificationent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, verification := range verifications {
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "FINANCE_VERIFICATION", ReferenceID: verification.ID.String(), ReferenceNo: verification.VerificationNo, Message: "核销单 " + verification.VerificationNo + " 已形成核销事实，变更不会改写该事实", BlocksExecution: true})
	}
	commissions, err := client.FinanceCommissionLine.Query().Where(financecommissionlineent.OrganizationIDEQ(orgID), financecommissionlineent.OrderIDIn(orderIDs...)).WithCommission().Order(financecommissionlineent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, line := range commissions {
		no := line.CommissionID.String()
		if line.Edges.Commission != nil {
			no = line.Edges.Commission.CommissionNo
		}
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "FINANCE_COMMISSION", ReferenceID: line.CommissionID.String(), ReferenceNo: no, Message: "提成单 " + no + " 已形成计算事实，变更不会改写该事实", BlocksExecution: true})
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().Where(financecommissionadjustmentent.OrganizationIDEQ(orgID), financecommissionadjustmentent.OrderIDIn(orderIDs...)).Order(financecommissionadjustmentent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, adjustment := range adjustments {
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "FINANCE_COMMISSION_ADJUSTMENT", ReferenceID: adjustment.ID.String(), ReferenceNo: adjustment.AdjustmentNo, Message: "提成调整单 " + adjustment.AdjustmentNo + " 已形成调整事实，变更不会改写该事实", BlocksExecution: true})
	}
	if houseBillID != uuid.Nil {
		allocations, err := client.SeaSharedContainerAllocation.Query().Where(
			seasharedcontainerallocationent.OrganizationIDEQ(orgID),
			seasharedcontainerallocationent.HouseBillIDEQ(houseBillID),
		).WithSharedContainer().Order(seasharedcontainerallocationent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, allocation := range allocations {
			no := allocation.SharedContainerID.String()
			if allocation.Edges.SharedContainer != nil {
				no = allocation.Edges.SharedContainer.ContainerNo
			}
			impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "SEA_SHARED_CONTAINER_ALLOCATION", ReferenceID: allocation.ID.String(), ReferenceNo: no, Message: "共享箱 " + no + " 已存在箱货分配，变更不会改写该事实", BlocksExecution: true})
		}
	}
	return impacts, nil
}

func hasBlockingImpact(items []*biz.SeaDocumentDownstreamImpact) bool {
	for _, item := range items {
		if item.BlocksExecution {
			return true
		}
	}
	return false
}

func validateConfirmationAttachment(ctx context.Context, client *ent.Client, orgID, orderID uuid.UUID, confirmation *biz.SeaExternalConfirmation) error {
	if confirmation == nil || confirmation.ConfirmationAttachmentID == nil {
		return nil
	}
	exists, err := client.OrderAttachment.Query().Where(
		orderattachmentent.IDEQ(*confirmation.ConfirmationAttachmentID),
		orderattachmentent.OrderIDEQ(orderID),
		orderattachmentent.HasOrderWith(orderent.OrganizationIDEQ(orgID)),
	).ForShare().Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrSeaDocumentInvalidArgument
	}
	return nil
}

func locateMasterMemberOrderIDs(ctx context.Context, client *ent.Client, orgID, orderID, mblID uuid.UUID) ([]uuid.UUID, uuid.UUID, error) {
	active, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID), seamasterbillorderlinkent.MasterBillIDEQ(mblID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
	if err != nil {
		return nil, uuid.Nil, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	members, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.MasterBillIDEQ(mblID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, uuid.Nil, err
	}
	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.OrderID)
	}
	return sortAndDeduplicateUUIDs(ids), active.ID, nil
}

// lockAndValidateMasterMemberLinks 在 MBL 锁之后锁定并重查全部 ACTIVE Link。
// 若成员集合在首次定位后发生变化，则让调用方刷新预览，避免用旧成员集合执行财务门禁。
func lockAndValidateMasterMemberLinks(ctx context.Context, tx *ent.Tx, orgID, orderID, mblID, expectedActiveLinkID uuid.UUID, expectedMemberIDs []uuid.UUID) (*ent.SeaMasterBillOrderLink, error) {
	links, err := tx.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(orgID),
		seamasterbillorderlinkent.MasterBillIDEQ(mblID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Order(seamasterbillorderlinkent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	actualMemberIDs := make([]uuid.UUID, 0, len(links))
	var activeLink *ent.SeaMasterBillOrderLink
	for _, link := range links {
		actualMemberIDs = append(actualMemberIDs, link.OrderID)
		if link.ID == expectedActiveLinkID && seaDocumentLinkMatches(link, orgID, orderID, mblID) {
			activeLink = link
		}
	}
	actualMemberIDs = sortAndDeduplicateUUIDs(actualMemberIDs)
	if activeLink == nil || !equalUUIDSlices(actualMemberIDs, expectedMemberIDs) {
		return nil, biz.ErrSeaDocumentStructureConflict
	}
	return activeLink, nil
}

func equalUUIDSlices(left, right []uuid.UUID) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func lockHouseDocument(ctx context.Context, tx *ent.Tx, orgID, orderID, hblID uuid.UUID, expectedOrderVersion, expectedHBLVersion uint64, expectedCurrentVersionID uuid.UUID) (*ent.Order, *ent.SeaMasterBillOrderLink, *ent.SeaMasterBill, *ent.SeaHouseBill, error) {
	order, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if order.Version != expectedOrderVersion {
		return nil, nil, nil, nil, biz.ErrOrderStatusConflict
	}
	located, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(located.MasterBillID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if mbl.Status == seamasterbillent.StatusVOIDED {
		return nil, nil, nil, nil, biz.ErrSeaDocumentVoided
	}
	link, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.IDEQ(located.ID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if !seaDocumentLinkMatches(link, orgID, orderID, mbl.ID) {
		return nil, nil, nil, nil, biz.ErrSeaDocumentVersionConflict
	}
	hbl, err := tx.SeaHouseBill.Query().Where(seahousebillent.IDEQ(hblID), seahousebillent.OrganizationIDEQ(orgID), seahousebillent.OrderIDEQ(orderID), seahousebillent.MasterBillIDEQ(mbl.ID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, nil, nil, nil, mapEntError(err, biz.ErrSeaHouseBillNotFound, nil)
	}
	if hbl.Status == seahousebillent.StatusVOIDED {
		return nil, nil, nil, nil, biz.ErrSeaDocumentVoided
	}
	if hbl.Version != expectedHBLVersion || hbl.CurrentVersionID == nil || *hbl.CurrentVersionID != expectedCurrentVersionID {
		return nil, nil, nil, nil, biz.ErrSeaDocumentVersionConflict
	}
	return order, link, mbl, hbl, nil
}

func createMasterVersion(ctx context.Context, tx *ent.Tx, mbl *ent.SeaMasterBill, exec *ent.SeaTransportExecution, actorID uuid.UUID, source string, reason, idempotencyKey, fingerprint *string, confirmation *biz.SeaExternalConfirmation) (*ent.SeaMasterBillVersion, error) {
	_ = exec
	latest, err := tx.SeaMasterBillVersion.Query().Where(seamasterbillversionent.MasterBillIDEQ(mbl.ID)).Order(ent.Desc(seamasterbillversionent.FieldVersionNo)).First(ctx)
	next := uint64(1)
	if err == nil {
		next = latest.VersionNo + 1
	} else if !ent.IsNotFound(err) {
		return nil, err
	}
	b := tx.SeaMasterBillVersion.Create().
		SetOrganizationID(mbl.OrganizationID).
		SetMasterBillID(mbl.ID).
		SetVersionNo(next).
		SetSourceEntityVersion(mbl.Version).
		SetShippingLineID(mbl.ShippingLineID).
		SetMasterNo(mbl.MasterNo).
		SetNormalizedMasterNo(mbl.NormalizedMasterNo).
		SetStatus(seamasterbillversionent.Status(mbl.Status)).
		SetContentHash(computeMBLContentHash(mbl)).
		SetSource(seamasterbillversionent.Source(source)).
		SetNillableReason(reason).
		SetNillableCreatedBy(&actorID).
		SetNillableIdempotencyKey(idempotencyKey).
		SetNillableRequestFingerprint(fingerprint).
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
		SetNillableClauses(mbl.Clauses).
		SetNillableForeignAgentText(mbl.ForeignAgentText)
	setMasterVersionConfirmation(b, confirmation)
	return b.Save(ctx)
}

func createHouseVersion(ctx context.Context, tx *ent.Tx, hbl *ent.SeaHouseBill, actorID uuid.UUID, source string, reason, idempotencyKey, fingerprint *string, confirmation *biz.SeaExternalConfirmation) (*ent.SeaHouseBillVersion, error) {
	latest, err := tx.SeaHouseBillVersion.Query().Where(seahousebillversionent.HouseBillIDEQ(hbl.ID)).Order(ent.Desc(seahousebillversionent.FieldVersionNo)).First(ctx)
	next := uint64(1)
	if err == nil {
		next = latest.VersionNo + 1
	} else if !ent.IsNotFound(err) {
		return nil, err
	}
	b := tx.SeaHouseBillVersion.Create().SetOrganizationID(hbl.OrganizationID).SetHouseBillID(hbl.ID).SetOrderID(hbl.OrderID).SetMasterBillID(hbl.MasterBillID).SetVersionNo(next).SetSourceEntityVersion(hbl.Version).SetHouseNo(hbl.HouseNo).SetNormalizedHouseNo(hbl.NormalizedHouseNo).SetIssuerSource(seahousebillversionent.IssuerSource(hbl.IssuerSource)).SetNillableIssuerOrganizationID(hbl.IssuerOrganizationID).SetNillableIssuerPartnerID(hbl.IssuerPartnerID).SetStatus(seahousebillversionent.Status(hbl.Status)).SetNillableNote(hbl.Note).SetContentHash(computeHBLContentHash(hbl)).SetSource(seahousebillversionent.Source(source)).SetNillableReason(reason).SetNillableCreatedBy(&actorID).SetNillableIdempotencyKey(idempotencyKey).SetNillableRequestFingerprint(fingerprint).SetNillableShipperText(hbl.ShipperText).SetNillableConsigneeText(hbl.ConsigneeText).SetNillableNotifyPartyText(hbl.NotifyPartyText).SetNillableSecondNotifyPartyText(hbl.SecondNotifyPartyText).SetNillableMarksText(hbl.MarksText).SetNillableGoodsDescriptionText(hbl.GoodsDescriptionText).SetNillablePackageCount(hbl.PackageCount).SetNillablePackageUnit(hbl.PackageUnit).SetNillableGrossWeightKg(hbl.GrossWeightKg).SetNillableVolumeCbm(hbl.VolumeCbm).SetNillableFreightTerms(hbl.FreightTerms).SetNillableTransportTerms(hbl.TransportTerms).SetNillableBillForm(hbl.BillForm).SetNillableReleaseType(hbl.ReleaseType).SetNillableClauses(hbl.Clauses).SetNillableForeignAgentText(hbl.ForeignAgentText)
	setHouseVersionConfirmation(b, confirmation)
	return b.Save(ctx)
}

func setMasterVersionConfirmation(builder *ent.SeaMasterBillVersionCreate, confirmation *biz.SeaExternalConfirmation) {
	if confirmation == nil {
		return
	}
	builder.SetConfirmedByParty(confirmation.ConfirmedByParty).
		SetConfirmedAt(confirmation.ConfirmedAt).
		SetConfirmationNote(confirmation.ConfirmationNote).
		SetNillableConfirmationAttachmentID(confirmation.ConfirmationAttachmentID)
}

func setHouseVersionConfirmation(builder *ent.SeaHouseBillVersionCreate, confirmation *biz.SeaExternalConfirmation) {
	if confirmation == nil {
		return
	}
	builder.SetConfirmedByParty(confirmation.ConfirmedByParty).
		SetConfirmedAt(confirmation.ConfirmedAt).
		SetConfirmationNote(confirmation.ConfirmationNote).
		SetNillableConfirmationAttachmentID(confirmation.ConfirmationAttachmentID)
}

func resolveHouseBillIssuerForDiff(ctx context.Context, client *ent.Client, orgID, orderID uuid.UUID, input *biz.SeaHouseBillInput) (*uuid.UUID, *uuid.UUID, error) {
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	return validateSeaHouseBillIssuer(ctx, client, orgID, order.OrganizationID, order.CustomerID, input)
}
