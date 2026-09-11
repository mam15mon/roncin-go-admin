package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
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
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	seadocumentmodechangeeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentmodechangeevent"
	seadocumentvoideventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentvoidevent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

type seaDocumentChangeRepo struct{ data *Data }

func NewSeaDocumentChangeRepo(data *Data) biz.SeaDocumentChangeRepo {
	return &seaDocumentChangeRepo{data: data}
}

func (r *seaDocumentChangeRepo) ListMasterBillVersions(ctx context.Context, orgID, orderID uuid.UUID, page, pageSize int) ([]*biz.SeaDocumentVersion, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(orgID),
		seamasterbillorderlinkent.OrderIDEQ(orderID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Only(ctx)
	if err != nil {
		return nil, 0, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	query := client.SeaMasterBillVersion.Query().Where(
		seamasterbillversionent.OrganizationIDEQ(orgID),
		seamasterbillversionent.MasterBillIDEQ(link.MasterBillID),
	).WithShippingLine()
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.Order(ent.Desc(seamasterbillversionent.FieldVersionNo), ent.Desc(seamasterbillversionent.FieldID)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.SeaDocumentVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, masterVersionToBiz(row, orderID))
	}
	return result, total, nil
}

func (r *seaDocumentChangeRepo) ListHouseBillVersions(ctx context.Context, orgID, orderID, houseBillID uuid.UUID, page, pageSize int) ([]*biz.SeaDocumentVersion, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	exists, err := client.SeaHouseBill.Query().Where(seahousebillent.IDEQ(houseBillID), seahousebillent.OrganizationIDEQ(orgID), seahousebillent.OrderIDEQ(orderID)).Exist(ctx)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, biz.ErrSeaHouseBillNotFound
	}
	query := client.SeaHouseBillVersion.Query().Where(
		seahousebillversionent.OrganizationIDEQ(orgID),
		seahousebillversionent.OrderIDEQ(orderID),
		seahousebillversionent.HouseBillIDEQ(houseBillID),
	)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.Order(ent.Desc(seahousebillversionent.FieldVersionNo), ent.Desc(seahousebillversionent.FieldID)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.SeaDocumentVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, houseVersionToBiz(row))
	}
	return result, total, nil
}

func (r *seaDocumentChangeRepo) GetDocumentVersion(ctx context.Context, orgID, orderID, versionID uuid.UUID, documentType biz.SeaDocumentType) (*biz.SeaDocumentVersion, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	switch documentType {
	case biz.SeaDocumentTypeMasterBill:
		row, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.IDEQ(versionID), seamasterbillversionent.OrganizationIDEQ(orgID)).WithShippingLine().Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaDocumentVersionNotFound, nil)
		}
		hasLink, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID), seamasterbillorderlinkent.MasterBillIDEQ(row.MasterBillID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !hasLink {
			return nil, biz.ErrSeaDocumentVersionNotFound
		}
		return masterVersionToBiz(row, orderID), nil
	case biz.SeaDocumentTypeHouseBill:
		row, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.IDEQ(versionID), seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.OrderIDEQ(orderID)).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaDocumentVersionNotFound, nil)
		}
		return houseVersionToBiz(row), nil
	default:
		return nil, biz.ErrSeaDocumentInvalidArgument
	}
}

func (r *seaDocumentChangeRepo) ListDocumentEvents(ctx context.Context, orgID, orderID uuid.UUID, page, pageSize int) ([]*biz.SeaDocumentEvent, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	links, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	mblIDs := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		mblIDs = append(mblIDs, link.MasterBillID)
	}
	events := make([]*biz.SeaDocumentEvent, 0)
	if len(mblIDs) > 0 {
		versions, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.OrganizationIDEQ(orgID), seamasterbillversionent.MasterBillIDIn(mblIDs...), seamasterbillversionent.SourceEQ(seamasterbillversionent.SourceAMENDMENT)).WithShippingLine().All(ctx)
		if err != nil {
			return nil, 0, err
		}
		for _, v := range versions {
			events = append(events, amendmentEventFromVersion(masterVersionToBiz(v, orderID)))
		}
	}
	hblVersions, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.OrderIDEQ(orderID), seahousebillversionent.SourceEQ(seahousebillversionent.SourceAMENDMENT)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, v := range hblVersions {
		events = append(events, amendmentEventFromVersion(houseVersionToBiz(v)))
	}
	voidPredicates := []predicate.SeaDocumentVoidEvent{
		seadocumentvoideventent.OrganizationIDEQ(orgID),
		seadocumentvoideventent.OrderIDEQ(orderID),
	}
	if len(mblIDs) > 0 {
		voidPredicates[1] = seadocumentvoideventent.Or(
			seadocumentvoideventent.OrderIDEQ(orderID),
			seadocumentvoideventent.MasterBillIDIn(mblIDs...),
		)
	}
	voids, err := client.SeaDocumentVoidEvent.Query().Where(voidPredicates...).WithMasterBillVersion().WithHouseBillVersion().All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, v := range voids {
		events = append(events, voidEventToBiz(v))
	}
	modeChanges, err := client.SeaDocumentModeChangeEvent.Query().Where(
		seadocumentmodechangeeventent.OrganizationIDEQ(orgID),
		seadocumentmodechangeeventent.OrderIDEQ(orderID),
	).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, event := range modeChanges {
		events = append(events, modeChangeEventToBiz(event))
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].ID.String() > events[j].ID.String()
		}
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
	total := len(events)
	start := (page - 1) * pageSize
	if start >= total {
		return []*biz.SeaDocumentEvent{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return events[start:end], total, nil
}

func (r *seaDocumentChangeRepo) PreviewAmendment(ctx context.Context, orgID uuid.UUID, input *biz.SeaDocumentAmendmentCommand) (*biz.SeaDocumentChangePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	base, diffs, orderIDs, err := loadAmendmentPreview(ctx, client, orgID, input)
	if err != nil {
		return nil, err
	}
	impacts, err := collectDocumentImpacts(ctx, client, orgID, orderIDs, amendmentImpactHouseBillID(input))
	if err != nil {
		return nil, err
	}
	if len(orderIDs) > 0 {
		orders, err := client.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(orderIDs...)).Order(orderent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			if impact := orderBusinessEditImpact(ctx, client.User, o); impact != nil {
				impacts = append(impacts, impact)
			}
		}
	}
	return &biz.SeaDocumentChangePreview{BaseVersion: base, Differences: diffs, Impacts: impacts, Executable: len(diffs) > 0 && !hasBlockingImpact(impacts)}, nil
}

func (r *seaDocumentChangeRepo) PreviewVoid(ctx context.Context, orgID uuid.UUID, input *biz.SeaDocumentVoidCommand) (*biz.SeaDocumentChangePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if input.DocumentType == biz.SeaDocumentTypeHouseBill {
		return nil, biz.ErrSeaDocumentStructureConflict
	}
	base, orderIDs, err := loadCurrentDocumentBase(ctx, client, orgID, input.OrderID, input.DocumentType, input.DocumentID, input.ExpectedOrderVersion, input.ExpectedDocumentVersion, input.ExpectedCurrentVersionID)
	if err != nil {
		return nil, err
	}
	diffs := []*biz.SeaDocumentFieldDifference{{Field: "status", Label: "状态", BeforeValue: base.Status, AfterValue: "VOIDED"}}
	impactHouseBillID := uuid.Nil
	if input.DocumentType == biz.SeaDocumentTypeHouseBill {
		impactHouseBillID = input.DocumentID
	}
	impacts, err := collectDocumentImpacts(ctx, client, orgID, orderIDs, impactHouseBillID)
	if err != nil {
		return nil, err
	}
	if len(orderIDs) > 0 {
		orders, err := client.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(orderIDs...)).Order(orderent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			if impact := orderBusinessEditImpact(ctx, client.User, o); impact != nil {
				impacts = append(impacts, impact)
			}
		}
	}
	return &biz.SeaDocumentChangePreview{BaseVersion: base, Differences: diffs, Impacts: impacts, Executable: !hasBlockingImpact(impacts)}, nil
}

func (r *seaDocumentChangeRepo) PreviewModeChange(ctx context.Context, orgID uuid.UUID, input *biz.SeaDocumentModeChangeCommand) (*biz.SeaDocumentChangePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(orgID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(orgID),
		seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	current := biz.SeaDocumentStructure(link.DocumentStructure)
	if current == input.TargetMode {
		return nil, biz.ErrSeaDocumentModeChangeConflict
	}
	differences := []*biz.SeaDocumentFieldDifference{{Field: "document_structure", Label: "单证模式", BeforeValue: string(current), AfterValue: string(input.TargetMode)}}
	var base *biz.SeaDocumentVersion
	// HOUSE 起点的模式切换会把当前唯一活动 HBL 置为 VOIDED：影响收集必须携带该
	// HBL 上下文，使其箱货分配进入阻断事实（与 ExecuteModeChange 同口径）。
	impactHouseBillID := uuid.Nil
	if current == biz.SeaDocumentStructureHouse {
		hbl, err := client.SeaHouseBill.Query().Where(
			seahousebillent.OrganizationIDEQ(orgID),
			seahousebillent.OrderIDEQ(input.OrderID),
			seahousebillent.MasterBillIDEQ(link.MasterBillID),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
		).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaDocumentStructureConflict, nil)
		}
		impactHouseBillID = hbl.ID
		if hbl.CurrentVersionID != nil {
			version, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.IDEQ(*hbl.CurrentVersionID)).Only(ctx)
			if err != nil {
				return nil, err
			}
			base = houseVersionToBiz(version)
		}
	} else {
		if input.NewHouseBill == nil {
			return nil, biz.ErrSeaDocumentInvalidArgument
		}
		if _, _, err := resolveHouseBillIssuerForDiff(ctx, client, orgID, input.OrderID, input.NewHouseBill); err != nil {
			return nil, err
		}
		differences = append(differences, &biz.SeaDocumentFieldDifference{Field: "house_no", Label: "HBL 号", BeforeValue: "", AfterValue: input.NewHouseBill.HouseNo})
	}
	impacts, err := collectDocumentImpacts(ctx, client, orgID, []uuid.UUID{order.ID}, impactHouseBillID)
	if err != nil {
		return nil, err
	}
	if impact := orderBusinessEditImpact(ctx, client.User, order); impact != nil {
		impacts = append(impacts, impact)
	}
	return &biz.SeaDocumentChangePreview{BaseVersion: base, Differences: differences, Impacts: impacts, Executable: !hasBlockingImpact(impacts)}, nil
}

// amendmentImpactHouseBillID 返回改单影响收集的 HBL 上下文：只有 HBL 改单需要
// 查询箱货分配，MBL 改单传入零值跳过。
func amendmentImpactHouseBillID(input *biz.SeaDocumentAmendmentCommand) uuid.UUID {
	if input.DocumentType == biz.SeaDocumentTypeHouseBill {
		return input.DocumentID
	}
	return uuid.Nil
}

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
	fees, err := client.OrderFee.Query().Where(orderfeeent.OrderIDIn(orderIDs...), orderfeeent.StatusNotIn(orderfeeent.StatusDRAFT, orderfeeent.StatusCANCELLED)).Order(orderfeeent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, fee := range fees {
		impacts = append(impacts, &biz.SeaDocumentDownstreamImpact{FactType: "ORDER_FEE", ReferenceID: fee.ID.String(), ReferenceNo: fee.FeeCode, Message: "费用 " + fee.FeeCode + " 已确认或进入结算，变更不会改写该事实", BlocksExecution: true})
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

func (r *seaDocumentChangeRepo) ExecuteAmendment(ctx context.Context, orgID, actorID uuid.UUID, input *biz.SeaDocumentAmendmentCommand, audit *biz.AuditEvent) (*biz.SeaDocumentVersion, error) {
	fingerprint := changeFingerprint(input)
	var resultID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if input.DocumentType == biz.SeaDocumentTypeMasterBill {
			existing, err := tx.SeaMasterBillVersion.Query().Where(seamasterbillversionent.OrganizationIDEQ(orgID), seamasterbillversionent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
			if err == nil {
				if existing.RequestFingerprint != nil && *existing.RequestFingerprint == fingerprint {
					resultID = existing.ID
					return nil
				}
				return biz.ErrSeaDocumentVersionConflict
			}
			if !ent.IsNotFound(err) {
				return err
			}
			return r.executeMasterAmendment(ctx, tx, orgID, actorID, input, fingerprint, audit, &resultID)
		}
		existing, err := tx.SeaHouseBillVersion.Query().Where(seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
		if err == nil {
			if existing.RequestFingerprint != nil && *existing.RequestFingerprint == fingerprint {
				resultID = existing.ID
				return nil
			}
			return biz.ErrSeaDocumentVersionConflict
		}
		if !ent.IsNotFound(err) {
			return err
		}
		return r.executeHouseAmendment(ctx, tx, orgID, actorID, input, fingerprint, audit, &resultID)
	})
	if err != nil {
		replayID, found, replayErr := r.findAmendmentReplay(ctx, orgID, input.DocumentType, input.IdempotencyKey, fingerprint)
		if replayErr != nil {
			return nil, replayErr
		}
		if !found {
			return nil, err
		}
		resultID = replayID
	}
	return r.GetDocumentVersion(ctx, orgID, input.OrderID, resultID, input.DocumentType)
}

func (r *seaDocumentChangeRepo) executeMasterAmendment(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentAmendmentCommand, fingerprint string, audit *biz.AuditEvent, resultID *uuid.UUID) error {
	memberIDs, activeLinkID, err := locateMasterMemberOrderIDs(ctx, tx.Client(), orgID, input.OrderID, input.DocumentID)
	if err != nil {
		return err
	}
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(memberIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	for _, order := range orders {
		if order.ID == input.OrderID && order.Version != input.ExpectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
	}
	mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(input.DocumentID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	activeLink, err := lockAndValidateMasterMemberLinks(ctx, tx, orgID, input.OrderID, mbl.ID, activeLinkID, memberIDs)
	if err != nil {
		return err
	}
	if mbl.Status == seamasterbillent.StatusVOIDED {
		return biz.ErrSeaDocumentVoided
	}
	if mbl.Version != input.ExpectedDocumentVersion || mbl.CurrentVersionID == nil || *mbl.CurrentVersionID != input.ExpectedCurrentVersionID {
		return biz.ErrSeaDocumentVersionConflict
	}
	if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, input.OrderID, input.Confirmation); err != nil {
		return err
	}
	exec, err := tx.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(activeLink.TransportExecutionID), seatransportexecutionent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	base, _, _, err := loadAmendmentPreview(ctx, tx.Client(), orgID, input)
	if err != nil {
		return err
	}
	impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, memberIDs, uuid.Nil)
	if err != nil {
		return err
	}
	if hasBlockingImpact(impacts) {
		return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
	}
	updatedBuilder := mbl.Update().SetVersion(mbl.Version + 1)
	setSeaMasterBillContent(updatedBuilder, input.Input.MasterBillContent)
	updated, err := updatedBuilder.Save(ctx)
	if err != nil {
		return err
	}
	version, err := createMasterVersion(ctx, tx, updated, exec, actorID, biz.VersionSourceAmendment, &input.Reason, &input.IdempotencyKey, &fingerprint, input.Confirmation)
	if err != nil {
		return err
	}
	if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
		return err
	}
	for _, order := range orders {
		if order.ID == input.OrderID {
			if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
				return err
			}
			break
		}
	}
	*resultID = version.ID
	audit.Action = "sea_master_bill.amend"
	audit.Details = map[string]string{"order.id": input.OrderID.String(), "master_bill.id": mbl.ID.String(), "previous_version.id": base.ID.String(), "result_version.id": version.ID.String(), "reason": input.Reason}
	return writeAudit(ctx, tx.AuditLog, audit)
}

func (r *seaDocumentChangeRepo) executeHouseAmendment(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentAmendmentCommand, fingerprint string, audit *biz.AuditEvent, resultID *uuid.UUID) error {
	order, link, mbl, hbl, err := lockHouseDocument(ctx, tx, orgID, input.OrderID, input.DocumentID, input.ExpectedOrderVersion, input.ExpectedDocumentVersion, input.ExpectedCurrentVersionID)
	if err != nil {
		return err
	}
	if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
		return err
	}
	if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, order.ID, input.Confirmation); err != nil {
		return err
	}
	base, _, _, err := loadAmendmentPreview(ctx, tx.Client(), orgID, input)
	if err != nil {
		return err
	}
	impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, []uuid.UUID{order.ID}, hbl.ID)
	if err != nil {
		return err
	}
	if hasBlockingImpact(impacts) {
		return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
	}
	issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), orgID, order.OrganizationID, order.CustomerID, input.Input.HouseBill)
	if err != nil {
		return err
	}
	normalized, _ := biz.NormalizeSeaHouseNo(input.Input.HouseBill.HouseNo)
	builder := hbl.Update().SetHouseNo(input.Input.HouseBill.HouseNo).SetNormalizedHouseNo(normalized).SetIssuerSource(seahousebillent.IssuerSource(input.Input.HouseBill.IssuerSource)).SetVersion(hbl.Version + 1)
	if issuerOrgID != nil {
		builder.SetIssuerOrganizationID(*issuerOrgID).ClearIssuerPartnerID()
	} else {
		builder.SetIssuerPartnerID(*issuerPartnerID).ClearIssuerOrganizationID()
	}
	if input.Input.HouseBill.Note != nil {
		builder.SetNote(*input.Input.HouseBill.Note)
	} else {
		builder.ClearNote()
	}
	setSeaHouseBillContentUpdate(builder, input.Input.HouseBill.Content)
	updated, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return biz.ErrSeaHouseBillExists
		}
		return err
	}
	version, err := createHouseVersion(ctx, tx, updated, actorID, biz.VersionSourceAmendment, &input.Reason, &input.IdempotencyKey, &fingerprint, input.Confirmation)
	if err != nil {
		return err
	}
	if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
		return err
	}
	if _, err = link.Update().SetVersion(link.Version + 1).Save(ctx); err != nil {
		return err
	}
	if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
		return err
	}
	*resultID = version.ID
	audit.Action = "sea_house_bill.amend"
	audit.Details = map[string]string{"order.id": order.ID.String(), "master_bill.id": mbl.ID.String(), "house_bill.id": hbl.ID.String(), "previous_version.id": base.ID.String(), "result_version.id": version.ID.String(), "reason": input.Reason}
	return writeAudit(ctx, tx.AuditLog, audit)
}

func (r *seaDocumentChangeRepo) ExecuteVoid(ctx context.Context, orgID, actorID uuid.UUID, input *biz.SeaDocumentVoidCommand, audit *biz.AuditEvent) (*biz.SeaDocumentEvent, error) {
	fingerprint := changeFingerprint(input)
	var eventID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.SeaDocumentVoidEvent.Query().Where(seadocumentvoideventent.OrganizationIDEQ(orgID), seadocumentvoideventent.IdempotencyKeyEQ(input.IdempotencyKey)).Only(ctx)
		if err == nil {
			if existing.RequestFingerprint == fingerprint {
				eventID = existing.ID
				return nil
			}
			return biz.ErrSeaDocumentVersionConflict
		}
		if !ent.IsNotFound(err) {
			return err
		}
		if input.DocumentType == biz.SeaDocumentTypeMasterBill {
			return r.executeMasterVoid(ctx, tx, orgID, actorID, input, fingerprint, audit, &eventID)
		}
		return r.executeHouseVoid(ctx, tx, orgID, actorID, input, fingerprint, audit, &eventID)
	})
	if err != nil {
		replayID, found, replayErr := r.findVoidReplay(ctx, orgID, input.IdempotencyKey, fingerprint)
		if replayErr != nil {
			return nil, replayErr
		}
		if !found {
			return nil, err
		}
		eventID = replayID
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	row, err := client.SeaDocumentVoidEvent.Query().Where(seadocumentvoideventent.IDEQ(eventID), seadocumentvoideventent.OrganizationIDEQ(orgID)).WithMasterBillVersion().WithHouseBillVersion().Only(ctx)
	if err != nil {
		return nil, err
	}
	return voidEventToBiz(row), nil
}

func (r *seaDocumentChangeRepo) executeMasterVoid(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentVoidCommand, fingerprint string, audit *biz.AuditEvent, eventID *uuid.UUID) error {
	memberIDs, activeLinkID, err := locateMasterMemberOrderIDs(ctx, tx.Client(), orgID, input.OrderID, input.DocumentID)
	if err != nil {
		return err
	}
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(memberIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	var requestOrder *ent.Order
	for _, order := range orders {
		if order.ID == input.OrderID {
			requestOrder = order
			if order.Version != input.ExpectedOrderVersion {
				return biz.ErrOrderStatusConflict
			}
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
	}
	mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(input.DocumentID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	activeLink, err := lockAndValidateMasterMemberLinks(ctx, tx, orgID, input.OrderID, mbl.ID, activeLinkID, memberIDs)
	if err != nil {
		return err
	}
	if mbl.Status == seamasterbillent.StatusVOIDED {
		return biz.ErrSeaDocumentVoided
	}
	if mbl.Version != input.ExpectedDocumentVersion || mbl.CurrentVersionID == nil || *mbl.CurrentVersionID != input.ExpectedCurrentVersionID {
		return biz.ErrSeaDocumentVersionConflict
	}
	if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, input.OrderID, input.Confirmation); err != nil {
		return err
	}
	exec, err := tx.SeaTransportExecution.Query().Where(seatransportexecutionent.IDEQ(activeLink.TransportExecutionID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, memberIDs, uuid.Nil)
	if err != nil {
		return err
	}
	if hasBlockingImpact(impacts) {
		return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
	}
	updated, err := mbl.Update().SetStatus(seamasterbillent.StatusVOIDED).SetVersion(mbl.Version + 1).Save(ctx)
	if err != nil {
		return err
	}
	version, err := createMasterVersion(ctx, tx, updated, exec, actorID, biz.VersionSourceVoid, &input.Reason, nil, nil, input.Confirmation)
	if err != nil {
		return err
	}
	if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
		return err
	}
	row, err := tx.SeaDocumentVoidEvent.Create().SetOrganizationID(orgID).SetOrderID(input.OrderID).SetDocumentType(seadocumentvoideventent.DocumentTypeMASTER).SetMasterBillID(mbl.ID).SetMasterBillVersionID(version.ID).SetPreviousMasterBillVersionID(input.ExpectedCurrentVersionID).SetPreviousStatus(string(mbl.Status)).SetVoidedStatus("VOIDED").SetReason(input.Reason).SetImpactSummary(impactSummary(impacts)).SetCreatedBy(actorID).SetIdempotencyKey(input.IdempotencyKey).SetRequestFingerprint(fingerprint).SetConfirmedByParty(input.Confirmation.ConfirmedByParty).SetConfirmedAt(input.Confirmation.ConfirmedAt).SetConfirmationNote(input.Confirmation.ConfirmationNote).SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).Save(ctx)
	if err != nil {
		return err
	}
	*eventID = row.ID
	if requestOrder != nil {
		if _, err = requestOrder.Update().SetVersion(requestOrder.Version + 1).Save(ctx); err != nil {
			return err
		}
	}
	audit.Action = "sea_master_bill.void"
	audit.Details = map[string]string{"order.id": input.OrderID.String(), "master_bill.id": mbl.ID.String(), "previous_version.id": input.ExpectedCurrentVersionID.String(), "result_version.id": version.ID.String(), "reason": input.Reason}
	return writeAudit(ctx, tx.AuditLog, audit)
}

func (r *seaDocumentChangeRepo) executeHouseVoid(ctx context.Context, tx *ent.Tx, orgID, actorID uuid.UUID, input *biz.SeaDocumentVoidCommand, fingerprint string, audit *biz.AuditEvent, eventID *uuid.UUID) error {
	return biz.ErrSeaDocumentStructureConflict
}

func (r *seaDocumentChangeRepo) ExecuteModeChange(ctx context.Context, orgID, actorID uuid.UUID, input *biz.SeaDocumentModeChangeCommand, audit *biz.AuditEvent) error {
	fingerprint := changeFingerprint(input)
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.SeaDocumentModeChangeEvent.Query().Where(
			seadocumentmodechangeeventent.OrganizationIDEQ(orgID),
			seadocumentmodechangeeventent.IdempotencyKeyEQ(input.IdempotencyKey),
		).Only(ctx)
		if err == nil {
			if existing.RequestFingerprint == fingerprint {
				return nil
			}
			return biz.ErrSeaDocumentModeChangeConflict
		}
		if !ent.IsNotFound(err) {
			return err
		}
		order, err := tx.Order.Query().Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrOrderNotFound, nil)
		}
		if order.Version != input.ExpectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}
		located, err := tx.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrganizationIDEQ(orgID),
			seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
		}
		mbl, err := tx.SeaMasterBill.Query().Where(seamasterbillent.IDEQ(located.MasterBillID), seamasterbillent.OrganizationIDEQ(orgID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
		}
		link, err := tx.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.IDEQ(located.ID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		if !seaDocumentLinkMatches(link, orgID, input.OrderID, mbl.ID) || link.Version != input.ExpectedLinkVersion {
			return biz.ErrSeaDocumentModeChangeConflict
		}
		previousMode := biz.SeaDocumentStructure(link.DocumentStructure)
		if previousMode == input.TargetMode {
			return biz.ErrSeaDocumentModeChangeConflict
		}
		if err := validateConfirmationAttachment(ctx, tx.Client(), orgID, order.ID, input.Confirmation); err != nil {
			return err
		}
		// HOUSE→DIRECT 会把当前唯一活动 HBL 置为 VOIDED：先定位该 HBL 并把其 ID
		// 传入影响收集，使箱货分配进入阻断事实；口径与 PreviewModeChange 一致，
		// 不允许 Preview 可执行而 Execute 被阻断（或反之）的漂移。
		impactHouseBillID := uuid.Nil
		if previousMode == biz.SeaDocumentStructureHouse {
			activeHBL, err := tx.SeaHouseBill.Query().Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.OrderIDEQ(order.ID),
				seahousebillent.MasterBillIDEQ(mbl.ID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
			).Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaDocumentStructureConflict, nil)
			}
			impactHouseBillID = activeHBL.ID
		}
		impacts, err := collectDocumentImpacts(ctx, tx.Client(), orgID, []uuid.UUID{order.ID}, impactHouseBillID)
		if err != nil {
			return err
		}
		if hasBlockingImpact(impacts) {
			return impactError(biz.ErrSeaDocumentChangeBlocked, impacts)
		}
		builder := tx.SeaDocumentModeChangeEvent.Create().
			SetOrganizationID(orgID).
			SetOrderID(order.ID).
			SetPreviousMode(seadocumentmodechangeeventent.PreviousMode(previousMode)).
			SetTargetMode(seadocumentmodechangeeventent.TargetMode(input.TargetMode)).
			SetReason(input.Reason).
			SetImpactSummary(impactSummary(impacts)).
			SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
			SetConfirmedAt(input.Confirmation.ConfirmedAt).
			SetConfirmationNote(input.Confirmation.ConfirmationNote).
			SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).
			SetCreatedBy(actorID).
			SetIdempotencyKey(input.IdempotencyKey).
			SetRequestFingerprint(fingerprint)

		switch {
		case previousMode == biz.SeaDocumentStructureHouse && input.TargetMode == biz.SeaDocumentStructureDirect:
			if input.ExpectedHouseBillVersion == nil || input.ExpectedCurrentVersionID == nil {
				return biz.ErrSeaDocumentInvalidArgument
			}
			hbl, err := tx.SeaHouseBill.Query().Where(
				seahousebillent.OrganizationIDEQ(orgID),
				seahousebillent.OrderIDEQ(order.ID),
				seahousebillent.MasterBillIDEQ(mbl.ID),
				seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
			).ForUpdate().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaDocumentStructureConflict, nil)
			}
			if hbl.Version != *input.ExpectedHouseBillVersion || hbl.CurrentVersionID == nil || *hbl.CurrentVersionID != *input.ExpectedCurrentVersionID {
				return biz.ErrSeaDocumentModeChangeConflict
			}
			updated, err := hbl.Update().SetStatus(seahousebillent.StatusVOIDED).SetVersion(hbl.Version + 1).Save(ctx)
			if err != nil {
				return err
			}
			version, err := createHouseVersion(ctx, tx, updated, actorID, biz.VersionSourceModeChange, &input.Reason, nil, nil, input.Confirmation)
			if err != nil {
				return err
			}
			if _, err = updated.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
				return err
			}
			builder.SetPreviousHouseBillID(hbl.ID).SetPreviousHouseBillVersionID(*input.ExpectedCurrentVersionID)
		case previousMode == biz.SeaDocumentStructureDirect && input.TargetMode == biz.SeaDocumentStructureHouse:
			if input.NewHouseBill == nil || input.ExpectedHouseBillVersion != nil || input.ExpectedCurrentVersionID != nil {
				return biz.ErrSeaDocumentInvalidArgument
			}
			activeCount, err := tx.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(order.ID), seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED)).Count(ctx)
			if err != nil {
				return err
			}
			if activeCount != 0 {
				return biz.ErrSeaDocumentStructureConflict
			}
			normalized, err := biz.NormalizeSeaHouseNo(input.NewHouseBill.HouseNo)
			if err != nil {
				return err
			}
			issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), orgID, order.OrganizationID, order.CustomerID, input.NewHouseBill)
			if err != nil {
				return err
			}
			hblBuilder := tx.SeaHouseBill.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(orgID).SetOrderID(order.ID).SetMasterBillID(mbl.ID).SetHouseNo(input.NewHouseBill.HouseNo).SetNormalizedHouseNo(normalized).SetIssuerSource(seahousebillent.IssuerSource(input.NewHouseBill.IssuerSource)).SetStatus(seahousebillent.StatusDRAFT).SetVersion(1)
			if issuerOrgID != nil {
				hblBuilder.SetIssuerOrganizationID(*issuerOrgID)
			}
			if issuerPartnerID != nil {
				hblBuilder.SetIssuerPartnerID(*issuerPartnerID)
			}
			if input.NewHouseBill.Note != nil {
				hblBuilder.SetNote(*input.NewHouseBill.Note)
			}
			setSeaHouseBillContentCreate(hblBuilder, input.NewHouseBill.Content)
			hbl, err := hblBuilder.Save(ctx)
			if err != nil {
				if ent.IsConstraintError(err) {
					return biz.ErrSeaHouseBillExists
				}
				return err
			}
			version, err := createHouseVersion(ctx, tx, hbl, actorID, biz.VersionSourceModeChange, &input.Reason, nil, nil, input.Confirmation)
			if err != nil {
				return err
			}
			if _, err = hbl.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
				return err
			}
			builder.SetTargetHouseBillID(hbl.ID).SetTargetHouseBillVersionID(version.ID)
		default:
			return biz.ErrSeaDocumentModeChangeConflict
		}
		if _, err = link.Update().SetDocumentStructure(seamasterbillorderlinkent.DocumentStructure(input.TargetMode)).SetVersion(link.Version + 1).Save(ctx); err != nil {
			return err
		}
		if _, err = order.Update().SetVersion(order.Version + 1).Save(ctx); err != nil {
			return err
		}
		if _, err = builder.Save(ctx); err != nil {
			return err
		}
		audit.Action = "sea_document.mode_change"
		audit.Details = map[string]string{"order.id": order.ID.String(), "previous_mode": string(previousMode), "target_mode": string(input.TargetMode), "reason": input.Reason}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *seaDocumentChangeRepo) findAmendmentReplay(ctx context.Context, orgID uuid.UUID, documentType biz.SeaDocumentType, idempotencyKey, fingerprint string) (uuid.UUID, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	if documentType == biz.SeaDocumentTypeMasterBill {
		row, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.OrganizationIDEQ(orgID), seamasterbillversionent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
		if ent.IsNotFound(err) {
			return uuid.Nil, false, nil
		}
		if err != nil {
			return uuid.Nil, false, err
		}
		if row.RequestFingerprint == nil || *row.RequestFingerprint != fingerprint {
			return uuid.Nil, false, biz.ErrSeaDocumentVersionConflict
		}
		return row.ID, true, nil
	}
	row, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	if row.RequestFingerprint == nil || *row.RequestFingerprint != fingerprint {
		return uuid.Nil, false, biz.ErrSeaDocumentVersionConflict
	}
	return row.ID, true, nil
}

func (r *seaDocumentChangeRepo) findVoidReplay(ctx context.Context, orgID uuid.UUID, idempotencyKey, fingerprint string) (uuid.UUID, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, false, err
	}
	row, err := client.SeaDocumentVoidEvent.Query().Where(seadocumentvoideventent.OrganizationIDEQ(orgID), seadocumentvoideventent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	if row.RequestFingerprint != fingerprint {
		return uuid.Nil, false, biz.ErrSeaDocumentVersionConflict
	}
	return row.ID, true, nil
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
		SetNillableClauses(mbl.Clauses)
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
	b := tx.SeaHouseBillVersion.Create().SetOrganizationID(hbl.OrganizationID).SetHouseBillID(hbl.ID).SetOrderID(hbl.OrderID).SetMasterBillID(hbl.MasterBillID).SetVersionNo(next).SetSourceEntityVersion(hbl.Version).SetHouseNo(hbl.HouseNo).SetNormalizedHouseNo(hbl.NormalizedHouseNo).SetIssuerSource(seahousebillversionent.IssuerSource(hbl.IssuerSource)).SetNillableIssuerOrganizationID(hbl.IssuerOrganizationID).SetNillableIssuerPartnerID(hbl.IssuerPartnerID).SetStatus(seahousebillversionent.Status(hbl.Status)).SetNillableNote(hbl.Note).SetContentHash(computeHBLContentHash(hbl)).SetSource(seahousebillversionent.Source(source)).SetNillableReason(reason).SetNillableCreatedBy(&actorID).SetNillableIdempotencyKey(idempotencyKey).SetNillableRequestFingerprint(fingerprint).SetNillableShipperText(hbl.ShipperText).SetNillableConsigneeText(hbl.ConsigneeText).SetNillableNotifyPartyText(hbl.NotifyPartyText).SetNillableSecondNotifyPartyText(hbl.SecondNotifyPartyText).SetNillableMarksText(hbl.MarksText).SetNillableGoodsDescriptionText(hbl.GoodsDescriptionText).SetNillablePackageCount(hbl.PackageCount).SetNillablePackageUnit(hbl.PackageUnit).SetNillableGrossWeightKg(hbl.GrossWeightKg).SetNillableVolumeCbm(hbl.VolumeCbm).SetNillableFreightTerms(hbl.FreightTerms).SetNillableTransportTerms(hbl.TransportTerms).SetNillableBillForm(hbl.BillForm).SetNillableReleaseType(hbl.ReleaseType).SetNillableClauses(hbl.Clauses)
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

func masterVersionToBiz(v *ent.SeaMasterBillVersion, orderID uuid.UUID) *biz.SeaDocumentVersion {
	if v == nil {
		return nil
	}
	result := &biz.SeaDocumentVersion{ID: v.ID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: v.MasterBillID, OrderID: orderID, MasterBillID: v.MasterBillID, VersionNo: v.VersionNo, SourceEntityVersion: v.SourceEntityVersion, DocumentNo: v.MasterNo, NormalizedDocumentNo: v.NormalizedMasterNo, Status: string(v.Status), Source: string(v.Source), Reason: v.Reason, ShippingLineID: &v.ShippingLineID, Content: versionContent(v.ShipperText, v.ConsigneeText, v.NotifyPartyText, v.SecondNotifyPartyText, v.MarksText, v.GoodsDescriptionText, v.PackageCount, v.PackageUnit, v.GrossWeightKg, v.VolumeCbm, v.FreightTerms, v.TransportTerms, v.BillForm, v.ReleaseType, v.Clauses), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, Confirmation: externalConfirmationFromVersion(v.ConfirmedByParty, v.ConfirmedAt, v.ConfirmationNote, v.ConfirmationAttachmentID)}
	if line := v.Edges.ShippingLine; line != nil {
		result.ShippingLineName = formatShippingLineName(line.NameZh, line.NameEn, line.ScacCode)
	}
	return result
}
func houseVersionToBiz(v *ent.SeaHouseBillVersion) *biz.SeaDocumentVersion {
	if v == nil {
		return nil
	}
	return &biz.SeaDocumentVersion{ID: v.ID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: v.HouseBillID, OrderID: v.OrderID, MasterBillID: v.MasterBillID, VersionNo: v.VersionNo, SourceEntityVersion: v.SourceEntityVersion, DocumentNo: v.HouseNo, NormalizedDocumentNo: v.NormalizedHouseNo, Status: string(v.Status), Source: string(v.Source), Reason: v.Reason, IssuerPartnerID: v.IssuerPartnerID, IssuerOrganizationID: v.IssuerOrganizationID, IssuerSource: biz.SeaHouseBillIssuerSource(v.IssuerSource), Note: v.Note, Content: versionContent(v.ShipperText, v.ConsigneeText, v.NotifyPartyText, v.SecondNotifyPartyText, v.MarksText, v.GoodsDescriptionText, v.PackageCount, v.PackageUnit, v.GrossWeightKg, v.VolumeCbm, v.FreightTerms, v.TransportTerms, v.BillForm, v.ReleaseType, v.Clauses), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, Confirmation: externalConfirmationFromVersion(v.ConfirmedByParty, v.ConfirmedAt, v.ConfirmationNote, v.ConfirmationAttachmentID)}
}

func externalConfirmationFromVersion(party *string, confirmedAt *time.Time, note *string, attachmentID *uuid.UUID) *biz.SeaExternalConfirmation {
	if party == nil || confirmedAt == nil || note == nil {
		return nil
	}
	return &biz.SeaExternalConfirmation{ConfirmedByParty: *party, ConfirmedAt: *confirmedAt, ConfirmationNote: *note, ConfirmationAttachmentID: attachmentID}
}
func versionContent(shipper, consignee, notify, secondNotify, marks, goods *string, packages *int, unit *string, weight, volume *float64, freight, transport, form, release, clauses *string) *biz.SeaBillContent {
	var count *int32
	if packages != nil {
		v := int32(*packages)
		count = &v
	}
	return &biz.SeaBillContent{ShipperText: shipper, ConsigneeText: consignee, NotifyPartyText: notify, SecondNotifyPartyText: secondNotify, MarksText: marks, GoodsDescriptionText: goods, PackageCount: count, PackageUnit: unit, GrossWeightKg: weight, VolumeCbm: volume, FreightTerms: freight, TransportTerms: transport, BillForm: form, ReleaseType: release, Clauses: clauses}
}
func documentStringPointer(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
func amendmentEventFromVersion(v *biz.SeaDocumentVersion) *biz.SeaDocumentEvent {
	id, no := v.DocumentID, v.DocumentNo
	return &biz.SeaDocumentEvent{ID: v.ID, EventType: biz.SeaDocumentEventTypeAmendment, DocumentType: v.DocumentType, DocumentID: &id, DocumentNo: &no, ResultVersionID: &v.ID, Reason: derefString(v.Reason), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt}
}
func voidEventToBiz(v *ent.SeaDocumentVoidEvent) *biz.SeaDocumentEvent {
	if v == nil {
		return nil
	}
	t := biz.SeaDocumentTypeHouseBill
	docID := v.HouseBillID
	prev := v.PreviousHouseBillVersionID
	result := v.HouseBillVersionID
	if v.DocumentType == seadocumentvoideventent.DocumentTypeMASTER {
		t = biz.SeaDocumentTypeMasterBill
		docID = v.MasterBillID
		prev = v.PreviousMasterBillVersionID
		result = v.MasterBillVersionID
	}
	var documentNo *string
	if v.Edges.MasterBillVersion != nil {
		documentNo = &v.Edges.MasterBillVersion.MasterNo
	}
	if v.Edges.HouseBillVersion != nil {
		documentNo = &v.Edges.HouseBillVersion.HouseNo
	}
	return &biz.SeaDocumentEvent{ID: v.ID, EventType: biz.SeaDocumentEventTypeVoid, DocumentType: t, DocumentID: docID, DocumentNo: documentNo, PreviousVersionID: prev, ResultVersionID: result, Reason: v.Reason, ImpactSummary: v.ImpactSummary, CreatedBy: &v.CreatedBy, CreatedAt: v.CreatedAt, Confirmation: externalConfirmationFromVoidEvent(v)}
}

func modeChangeEventToBiz(v *ent.SeaDocumentModeChangeEvent) *biz.SeaDocumentEvent {
	if v == nil {
		return nil
	}
	previous := biz.SeaDocumentStructure(v.PreviousMode)
	target := biz.SeaDocumentStructure(v.TargetMode)
	return &biz.SeaDocumentEvent{
		ID: v.ID, EventType: biz.SeaDocumentEventTypeModeChange, DocumentType: biz.SeaDocumentTypeHouseBill,
		Reason: v.Reason, ImpactSummary: v.ImpactSummary, PreviousMode: &previous, TargetMode: &target,
		CreatedBy: &v.CreatedBy, CreatedAt: v.CreatedAt,
		Confirmation: &biz.SeaExternalConfirmation{ConfirmedByParty: v.ConfirmedByParty, ConfirmedAt: v.ConfirmedAt, ConfirmationNote: v.ConfirmationNote, ConfirmationAttachmentID: v.ConfirmationAttachmentID},
	}
}

func externalConfirmationFromVoidEvent(v *ent.SeaDocumentVoidEvent) *biz.SeaExternalConfirmation {
	if v == nil {
		return nil
	}
	return &biz.SeaExternalConfirmation{ConfirmedByParty: v.ConfirmedByParty, ConfirmedAt: v.ConfirmedAt, ConfirmationNote: v.ConfirmationNote, ConfirmationAttachmentID: v.ConfirmationAttachmentID}
}

func resolveHouseBillIssuerForDiff(ctx context.Context, client *ent.Client, orgID, orderID uuid.UUID, input *biz.SeaHouseBillInput) (*uuid.UUID, *uuid.UUID, error) {
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(orgID)).Only(ctx)
	if err != nil {
		return nil, nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	return validateSeaHouseBillIssuer(ctx, client, orgID, order.OrganizationID, order.CustomerID, input)
}

func diffHouseVersionToInput(base *biz.SeaDocumentVersion, input *biz.SeaHouseBillInput, issuerOrgID, issuerPartnerID *uuid.UUID) []*biz.SeaDocumentFieldDifference {
	result := []*biz.SeaDocumentFieldDifference{}
	addDiff(&result, "house_no", "HBL 号", base.DocumentNo, input.HouseNo)
	addDiff(&result, "issuer_source", "签发主体来源", string(base.IssuerSource), string(input.IssuerSource))
	beforeIssuerID := uuidValue(base.IssuerPartnerID)
	if base.IssuerOrganizationID != nil {
		beforeIssuerID = base.IssuerOrganizationID.String()
	}
	afterIssuerID := uuidValue(issuerPartnerID)
	if issuerOrgID != nil {
		afterIssuerID = issuerOrgID.String()
	}
	addDiff(&result, "issuer_identity_id", "签发主体", beforeIssuerID, afterIssuerID)
	addDiff(&result, "note", "备注", documentStringValue(base.Note), documentStringValue(input.Note))
	return append(result, diffContent(base.Content, input.Content)...)
}
func diffContent(before, after *biz.SeaBillContent) []*biz.SeaDocumentFieldDifference {
	if before == nil {
		before = &biz.SeaBillContent{}
	}
	if after == nil {
		after = &biz.SeaBillContent{}
	}
	result := []*biz.SeaDocumentFieldDifference{}
	pairs := []struct{ key, label, before, after string }{{"shipper_text", "发货人", documentStringValue(before.ShipperText), documentStringValue(after.ShipperText)}, {"consignee_text", "收货人", documentStringValue(before.ConsigneeText), documentStringValue(after.ConsigneeText)}, {"notify_party_text", "通知人", documentStringValue(before.NotifyPartyText), documentStringValue(after.NotifyPartyText)}, {"second_notify_party_text", "第二通知人", documentStringValue(before.SecondNotifyPartyText), documentStringValue(after.SecondNotifyPartyText)}, {"marks_text", "唛头", documentStringValue(before.MarksText), documentStringValue(after.MarksText)}, {"goods_description_text", "货描", documentStringValue(before.GoodsDescriptionText), documentStringValue(after.GoodsDescriptionText)}, {"package_count", "件数", int32Value(before.PackageCount), int32Value(after.PackageCount)}, {"package_unit", "包装单位", documentStringValue(before.PackageUnit), documentStringValue(after.PackageUnit)}, {"gross_weight_kg", "毛重", floatValue(before.GrossWeightKg), floatValue(after.GrossWeightKg)}, {"volume_cbm", "体积", floatValue(before.VolumeCbm), floatValue(after.VolumeCbm)}, {"freight_terms", "运费条款", documentStringValue(before.FreightTerms), documentStringValue(after.FreightTerms)}, {"transport_terms", "运输条款", documentStringValue(before.TransportTerms), documentStringValue(after.TransportTerms)}, {"bill_form", "提单形式", documentStringValue(before.BillForm), documentStringValue(after.BillForm)}, {"release_type", "放单方式", documentStringValue(before.ReleaseType), documentStringValue(after.ReleaseType)}, {"clauses", "特别条款", documentStringValue(before.Clauses), documentStringValue(after.Clauses)}}
	for _, p := range pairs {
		addDiff(&result, p.key, p.label, p.before, p.after)
	}
	return result
}
func addDiff(result *[]*biz.SeaDocumentFieldDifference, key, label, before, after string) {
	if before != after {
		*result = append(*result, &biz.SeaDocumentFieldDifference{Field: key, Label: label, BeforeValue: before, AfterValue: after})
	}
}
func documentStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func uuidValue(v *uuid.UUID) string {
	if v == nil {
		return ""
	}
	return v.String()
}
func int32Value(v *int32) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(int64(*v), 10)
}
func floatValue(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}
func changeFingerprint(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func impactSummary(items []*biz.SeaDocumentDownstreamImpact) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.FactType+":"+item.ReferenceNo)
	}
	return strings.Join(parts, "；")
}
func impactError(base *kratoserrors.Error, items []*biz.SeaDocumentDownstreamImpact) error {
	metadata := map[string]string{"blocked_count": strconv.Itoa(len(items))}
	if len(items) > 0 {
		metadata["fact_type"] = items[0].FactType
		metadata["reference_id"] = items[0].ReferenceID
		metadata["reference_no"] = items[0].ReferenceNo
	}
	return biz.MetadataError(base, metadata)
}
