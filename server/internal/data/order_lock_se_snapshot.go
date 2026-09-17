package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
)

type seaOrderLockHouseBillSnapshot struct {
	HouseBillID        uuid.UUID
	HouseBillVersionID uuid.UUID
	HouseNoSnapshot    string
}

type seaOrderLockSnapshot struct {
	MasterBillID                uuid.UUID
	MasterBillVersionID         uuid.UUID
	TransportExecutionID        uuid.UUID
	TransportExecutionVersionID uuid.UUID
	HouseBills                  []seaOrderLockHouseBillSnapshot
}

// createSeaOrderLockSnapshot 在已持有 Order 行锁的事务内创建或复用 SE 单证不可变版本。
// 固定锁序为 Order → MBL → Link → 按 UUID 排序的 HBL。
// actorID 是写入新建版本 created_by 的人工锁定人；系统自动锁定传 nil，
// 新建版本的 created_by 保持为空，复用既有版本时不改写原 created_by。
func createSeaOrderLockSnapshot(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID, actorID *uuid.UUID) (*seaOrderLockSnapshot, error) {
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
		Where(
			seatransportexecutionent.IDEQ(link.TransportExecutionID),
			seatransportexecutionent.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 锁序 4: HBL FOR UPDATE，查询显式按 UUID 升序。
	hbls, err := tx.SeaHouseBill.Query().
		Where(
			seahousebillent.OrganizationIDEQ(organizationID),
			seahousebillent.OrderIDEQ(orderID),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
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

	// 运输执行快照（复用或新建 SeaTransportExecutionVersion）。
	teVersion, err := ensureSeaTransportExecutionVersion(
		ctx, tx, organizationID, execution, actorID,
		seatransportexecutionversionent.SourceORDER_LOCK, nil, nil, nil, nil,
	)
	if err != nil {
		return nil, err
	}
	teVersionID := teVersion.ID

	// MBL 快照（复用或新建 SeaMasterBillVersion）。
	mblVersionID, err := ensureSeaMasterBillLockVersion(ctx, tx, organizationID, mbl, actorID)
	if err != nil {
		return nil, err
	}

	snapshot := &seaOrderLockSnapshot{
		MasterBillID:                mbl.ID,
		MasterBillVersionID:         mblVersionID,
		TransportExecutionID:        execution.ID,
		TransportExecutionVersionID: teVersionID,
		HouseBills:                  make([]seaOrderLockHouseBillSnapshot, 0, len(hbls)),
	}
	for _, hbl := range hbls {
		versionID, versionErr := ensureSeaHouseBillLockVersion(ctx, tx, organizationID, hbl, orderID, mbl.ID, actorID)
		if versionErr != nil {
			return nil, versionErr
		}
		snapshot.HouseBills = append(snapshot.HouseBills, seaOrderLockHouseBillSnapshot{
			HouseBillID:        hbl.ID,
			HouseBillVersionID: versionID,
			HouseNoSnapshot:    hbl.HouseNo,
		})
	}
	return snapshot, nil
}

// ensureSeaMasterBillLockVersion 复用或新建与 MBL 当前实体版本和内容哈希一致的
// 不可变版本，并把 current_version_id 推进到胜出版本。actorID 仅在人工锁定时
// 作为新建版本 created_by；系统自动锁定传 nil，复用既有版本时不改写原 created_by。
func ensureSeaMasterBillLockVersion(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, mbl *ent.SeaMasterBill, actorID *uuid.UUID) (uuid.UUID, error) {
	mblHash := computeMBLContentHash(mbl)
	if mbl.CurrentVersionID != nil {
		current, currentErr := tx.SeaMasterBillVersion.Get(ctx, *mbl.CurrentVersionID)
		if currentErr == nil && current != nil && current.SourceEntityVersion == mbl.Version && current.ContentHash == mblHash {
			return current.ID, nil
		}
	}
	latest, _ := tx.SeaMasterBillVersion.Query().
		Where(seamasterbillversionent.MasterBillIDEQ(mbl.ID)).
		Order(ent.Desc(seamasterbillversionent.FieldVersionNo)).
		First(ctx)
	nextVersion := uint64(1)
	if latest != nil {
		nextVersion = latest.VersionNo + 1
	}

	created, createErr := tx.SeaMasterBillVersion.Create().
		SetOrganizationID(organizationID).
		SetMasterBillID(mbl.ID).
		SetVersionNo(nextVersion).
		SetSourceEntityVersion(mbl.Version).
		SetShippingLineID(mbl.ShippingLineID).
		SetMasterNo(mbl.MasterNo).
		SetNormalizedMasterNo(mbl.NormalizedMasterNo).
		SetStatus(seamasterbillversionent.Status(mbl.Status)).
		SetContentHash(mblHash).
		SetSource(seamasterbillversionent.SourceORDER_LOCK).
		SetNillableCreatedBy(actorID).
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
		Save(ctx)
	if createErr != nil {
		return uuid.Nil, createErr
	}
	if created.MasterBillID != mbl.ID {
		return uuid.Nil, biz.ErrSeaDocumentStructureConflict
	}
	if _, err := tx.SeaMasterBill.UpdateOneID(mbl.ID).SetCurrentVersionID(created.ID).Save(ctx); err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

// ensureSeaHouseBillLockVersion 复用或新建与 HBL 当前实体版本和内容哈希一致的
// 不可变版本，并把 current_version_id 推进到胜出版本。actorID 语义与 MBL 版本一致。
func ensureSeaHouseBillLockVersion(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, hbl *ent.SeaHouseBill, orderID, masterBillID uuid.UUID, actorID *uuid.UUID) (uuid.UUID, error) {
	hash := computeHBLContentHash(hbl)
	if hbl.CurrentVersionID != nil {
		current, currentErr := tx.SeaHouseBillVersion.Get(ctx, *hbl.CurrentVersionID)
		if currentErr == nil && current != nil && current.SourceEntityVersion == hbl.Version && current.ContentHash == hash {
			return current.ID, nil
		}
	}
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
		SetMasterBillID(masterBillID).
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
		SetNillableCreatedBy(actorID).
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
		return uuid.Nil, createErr
	}
	if created.HouseBillID != hbl.ID {
		return uuid.Nil, biz.ErrSeaDocumentStructureConflict
	}
	if _, err := tx.SeaHouseBill.UpdateOneID(hbl.ID).SetCurrentVersionID(created.ID).Save(ctx); err != nil {
		return uuid.Nil, err
	}
	return created.ID, nil
}

// seaGroupLockContext 共享 MBL 组级自动锁定的单证上下文。
// 全部行锁（MBL、Link、运输执行、HBL）在组级事务内取得后填充。
type seaGroupLockContext struct {
	MasterBill         *ent.SeaMasterBill
	TransportExecution *ent.SeaTransportExecution
	Links              []*ent.SeaMasterBillOrderLink
	// MemberOrderIDs 重验后的活动成员订单 ID（按 UUID 升序）。
	MemberOrderIDs    []uuid.UUID
	HouseBillsByOrder map[uuid.UUID][]*ent.SeaHouseBill
}

// lockSeaGroupDocumentGraph 在全部成员 Order 行锁已按 UUID 升序取得的前提下，
// 锁定共享 MBL、按 ID 升序的全部活动 Link、共享运输执行与各成员 HBL（按 ID
// 升序），随后重验活动成员集合。集合与期望不一致时返回结构冲突，整组零写入。
func lockSeaGroupDocumentGraph(ctx context.Context, tx *ent.Tx, organizationID, masterBillID uuid.UUID, expectedMemberIDs []uuid.UUID) (*seaGroupLockContext, error) {
	mbl, err := tx.SeaMasterBill.Query().
		Where(seamasterbillent.IDEQ(masterBillID), seamasterbillent.OrganizationIDEQ(organizationID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 锁定按 ID 升序的全部活动 Link，随后重验活动成员集合。
	links, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.MasterBillIDEQ(masterBillID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		Order(ent.Asc(seamasterbillorderlinkent.FieldID)).
		ForUpdate().
		All(ctx)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[uuid.UUID]struct{}, len(expectedMemberIDs))
	for _, id := range expectedMemberIDs {
		memberSet[id] = struct{}{}
	}
	verifiedMemberIDs := make([]uuid.UUID, 0, len(links))
	transportExecutionID := uuid.Nil
	for _, link := range links {
		if link.OrderID != uuid.Nil {
			if _, ok := memberSet[link.OrderID]; !ok {
				return nil, biz.ErrSeaDocumentStructureConflict
			}
			verifiedMemberIDs = append(verifiedMemberIDs, link.OrderID)
		}
		if transportExecutionID == uuid.Nil {
			transportExecutionID = link.TransportExecutionID
		}
		if link.TransportExecutionID != transportExecutionID {
			return nil, biz.ErrSeaDocumentStructureConflict
		}
	}
	if len(verifiedMemberIDs) != len(expectedMemberIDs) {
		return nil, biz.ErrSeaDocumentStructureConflict
	}

	execution, err := tx.SeaTransportExecution.Query().
		Where(
			seatransportexecutionent.IDEQ(transportExecutionID),
			seatransportexecutionent.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 锁定全部成员 HBL（按 ID 升序，不含作废行）。
	hbls, err := tx.SeaHouseBill.Query().
		Where(
			seahousebillent.OrganizationIDEQ(organizationID),
			seahousebillent.OrderIDIn(expectedMemberIDs...),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
		).
		Order(ent.Asc(seahousebillent.FieldID)).
		ForUpdate().
		All(ctx)
	if err != nil {
		return nil, err
	}

	graph := &seaGroupLockContext{
		MasterBill:         mbl,
		TransportExecution: execution,
		Links:              links,
		MemberOrderIDs:     verifiedMemberIDs,
		HouseBillsByOrder:  make(map[uuid.UUID][]*ent.SeaHouseBill),
	}
	for _, hbl := range hbls {
		graph.HouseBillsByOrder[hbl.OrderID] = append(graph.HouseBillsByOrder[hbl.OrderID], hbl)
	}
	return graph, nil
}

// ensureSeaGroupSharedVersions 为整组复用或新建同一份 MBL/运输执行不可变版本；
// 整组锁定共享同一版本引用，createdBy 语义与单票快照一致。
func ensureSeaGroupSharedVersions(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, graph *seaGroupLockContext, createdBy *uuid.UUID) (masterBillVersionID, transportExecutionVersionID uuid.UUID, err error) {
	teVersion, err := ensureSeaTransportExecutionVersion(
		ctx, tx, organizationID, graph.TransportExecution, createdBy,
		seatransportexecutionversionent.SourceORDER_LOCK, nil, nil, nil, nil,
	)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	mblVersionID, err := ensureSeaMasterBillLockVersion(ctx, tx, organizationID, graph.MasterBill, createdBy)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return mblVersionID, teVersion.ID, nil
}
