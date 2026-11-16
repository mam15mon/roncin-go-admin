package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecution "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
)

type seaMasterBillRepo struct {
	data *Data
}

func NewSeaMasterBillRepo(data *Data) biz.SeaMasterBillRepo {
	return &seaMasterBillRepo{data: data}
}

// querySeaMasterBillActiveMemberLinks 查询共享主单批次的活动成员票。
// 口径与候选查询 member_count 同源：该 MBL 上 status = 'ACTIVE' 的 Link
// （订单无删除路径，外键必在）。lock 为 true 时按主键排序后加行锁，
// 固定加锁顺序防止死锁；调用方必须已按锁序（Order -> MBL -> Link）锁定 MBL。
func querySeaMasterBillActiveMemberLinks(ctx context.Context, client *ent.Client, organizationID, masterBillID uuid.UUID, lock bool) ([]*ent.SeaMasterBillOrderLink, error) {
	query := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.MasterBillIDEQ(masterBillID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		)
	if lock {
		query = query.Order(seamasterbillorderlink.ByID()).ForUpdate()
	}
	links, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateSeaMasterMemberIdentities(ctx, client, organizationID, masterBillID, links); err != nil {
		return nil, err
	}
	return links, nil
}

// validateSeaMasterMemberIdentities 复验完整关联集合，不以公司过滤隐藏异常成员。
// 写入口仍由调用方按既有 Order→MBL→Link 锁序取得锁，本函数不追加逆序锁。
func validateSeaMasterMemberIdentities(ctx context.Context, client *ent.Client, organizationID, masterBillID uuid.UUID, links []*ent.SeaMasterBillOrderLink) error {
	master, err := client.SeaMasterBill.Query().Where(seamasterbill.IDEQ(masterBillID), seamasterbill.OrganizationIDEQ(organizationID), seamasterbill.BusinessTypeIn("SE", "SI")).Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	ids := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		if link.OrganizationID != master.OrganizationID || link.MasterBillID != master.ID {
			return biz.ErrSeaDocumentStructureConflict
		}
		ids = append(ids, link.OrderID)
	}
	if len(ids) == 0 {
		return nil
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(sortAndDeduplicateUUIDs(ids)) {
		return biz.ErrSeaDocumentStructureConflict
	}
	for _, order := range orders {
		if order.OrganizationID != master.OrganizationID || string(order.BusinessType) != master.BusinessType || biz.OrderBusinessType(order.BusinessType).IsImport() != (order.TradeDirection == orderent.TradeDirectionImport) {
			return biz.ErrSeaDocumentStructureConflict
		}
	}
	return nil
}

func (r *seaMasterBillRepo) MatchCandidate(ctx context.Context, organizationID, shippingLineID uuid.UUID, normalizedMasterNo string, voyage *biz.SeaTransportExecution, businessType biz.OrderBusinessType) (*biz.SeaMasterBillMatchResult, error) {
	return r.matchCandidateInternal(ctx, organizationID, shippingLineID, normalizedMasterNo, voyage, businessType)
}

func (r *seaMasterBillRepo) matchCandidateInternal(ctx context.Context, organizationID, shippingLineID uuid.UUID, normalizedMasterNo string, voyage *biz.SeaTransportExecution, businessType biz.OrderBusinessType) (*biz.SeaMasterBillMatchResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	mbl, err := client.SeaMasterBill.Query().
		Where(
			seamasterbill.OrganizationIDEQ(organizationID), seamasterbill.BusinessTypeEQ(string(businessType)),
			seamasterbill.ShippingLineIDEQ(shippingLineID),
			seamasterbill.NormalizedMasterNoEQ(normalizedMasterNo),
		).
		WithOrderLinks(func(q *ent.SeaMasterBillOrderLinkQuery) {
			q.Where(
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
				WithTransportExecution(func(tq *ent.SeaTransportExecutionQuery) {
					tq.Where(seatransportexecution.OrganizationIDEQ(organizationID))
				}).
				WithOrder(func(oq *ent.OrderQuery) {
					oq.Where(orderent.OrganizationIDEQ(organizationID))
				})
		}).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return &biz.SeaMasterBillMatchResult{Matched: false}, nil
		}
		return nil, err
	}
	if err := validateSeaMasterMemberIdentities(ctx, client, organizationID, mbl.ID, mbl.Edges.OrderLinks); err != nil {
		return nil, err
	}

	transportExecutions := make([]*biz.SeaTransportExecution, 0, len(mbl.Edges.OrderLinks))
	seenExecutionIDs := make(map[uuid.UUID]struct{}, len(mbl.Edges.OrderLinks))
	for _, l := range mbl.Edges.OrderLinks {
		te := l.Edges.TransportExecution
		if te == nil {
			continue
		}
		if _, exists := seenExecutionIDs[te.ID]; exists {
			continue
		}
		seenExecutionIDs[te.ID] = struct{}{}
		candidateTE := seaTransportExecutionToBiz(te)
		if err := r.populateLocationAndShippingLineNames(ctx, client, organizationID, candidateTE); err != nil {
			return nil, err
		}
		transportExecutions = append(transportExecutions, candidateTE)
	}

	var members []*biz.SeaMasterBillMemberSummary
	for _, link := range mbl.Edges.OrderLinks {
		if link.Edges.Order != nil {
			members = append(members, &biz.SeaMasterBillMemberSummary{
				OrderID:             link.Edges.Order.ID,
				OrderNo:             link.Edges.Order.OrderNo,
				CustomerReferenceNo: link.Edges.Order.CustomerReferenceNo,
				DocumentStructure:   biz.SeaDocumentStructure(link.DocumentStructure),
			})
		}
	}

	// 批次内全部规范化分单号（含作废行，不过滤状态），供前端失焦即时排重提示。
	batchHouseNos, err := listSeaMasterBillBatchNormalizedHouseNos(ctx, client, organizationID, mbl.ID)
	if err != nil {
		return nil, err
	}

	shippingLineName, err := r.getShippingLineName(ctx, client, organizationID, mbl.ShippingLineID)
	if err != nil {
		return nil, err
	}

	candidate := &biz.SeaMasterBillCandidate{
		BusinessType:            biz.OrderBusinessType(mbl.BusinessType),
		ID:                      mbl.ID,
		Version:                 mbl.Version,
		MasterNo:                mbl.MasterNo,
		ShippingLineID:          mbl.ShippingLineID,
		ShippingLineName:        shippingLineName,
		TransportExecutions:     transportExecutions,
		MemberCount:             len(members),
		Members:                 members,
		BatchNormalizedHouseNos: batchHouseNos,
	}

	var conflicts []*biz.SeaVoyageConflict
	if voyage != nil && len(transportExecutions) == 1 {
		conflicts = biz.CheckSeaVoyageConflicts(transportExecutions[0], voyage)
	}

	return &biz.SeaMasterBillMatchResult{
		Matched:   true,
		Candidate: candidate,
		Conflicts: conflicts,
	}, nil
}

func (r *seaMasterBillRepo) GetSummariesByOrderIDs(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*biz.SeaMasterBillSummary, error) {
	result := make(map[uuid.UUID]*biz.SeaMasterBillSummary, len(orderIDs))
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
		WithOrder().
		WithTransportExecution().
		WithMasterBill(func(q *ent.SeaMasterBillQuery) {
			q.Where(seamasterbill.OrganizationIDEQ(organizationID), seamasterbill.BusinessTypeIn("SE", "SI"))
		}).
		All(ctx)

	if err != nil {
		return nil, err
	}

	for _, link := range links {
		if e := validateSeaLinkIdentity(link); e != nil {
			return nil, e
		}
	}
	if len(links) == 0 {
		return result, nil
	}

	// 收集所有 MBL ID 查询成员数
	mblIDs := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		if link.Edges.MasterBill != nil {
			mblIDs = append(mblIDs, link.MasterBillID)
		}
	}

	memberCountMap := make(map[uuid.UUID]int)
	if len(mblIDs) > 0 {
		allLinks, err := client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.MasterBillIDIn(mblIDs...),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, l := range allLinks {
			memberCountMap[l.MasterBillID]++
		}
	}

	for _, link := range links {
		mbl := link.Edges.MasterBill
		if mbl == nil {
			continue
		}
		te := link.Edges.TransportExecution
		shippingLineName, err := r.getShippingLineName(ctx, client, organizationID, mbl.ShippingLineID)
		if err != nil {
			return nil, err
		}
		summary := &biz.SeaMasterBillSummary{
			BusinessType:     biz.OrderBusinessType(mbl.BusinessType),
			MasterBillID:     mbl.ID,
			MasterNo:         mbl.MasterNo,
			ShippingLineID:   mbl.ShippingLineID,
			ShippingLineName: shippingLineName,
			Status:           string(mbl.Status),
			Version:          mbl.Version,
			MemberCount:      memberCountMap[mbl.ID],
		}
		if te != nil {
			transportExecution := &biz.SeaTransportExecution{
				ShippingLineID:      te.ShippingLineID,
				OriginLocationID:    uuid.Nil,
				DischargeLocationID: uuid.Nil,
				TransitLocationID:   te.TransitLocationID,
			}
			if te.OriginLocationID != nil {
				transportExecution.OriginLocationID = *te.OriginLocationID
			}
			if te.DischargeLocationID != nil {
				transportExecution.DischargeLocationID = *te.DischargeLocationID
			}
			if err := r.populateLocationAndShippingLineNames(ctx, client, organizationID, transportExecution); err != nil {
				return nil, err
			}
			summary.TransportExecutionID = te.ID
			summary.TransportExecutionVersion = te.Version
			summary.OriginLocationID = te.OriginLocationID
			summary.OriginLocationName = transportExecution.OriginLocationName
			summary.DischargeLocationID = te.DischargeLocationID
			summary.DischargeLocationName = transportExecution.DischargeLocationName
			summary.TransitLocationID = te.TransitLocationID
			summary.TransitLocationName = transportExecution.TransitLocationName
			summary.VesselName = te.VesselName
			summary.VoyageNo = te.VoyageNo
			if te.Etd != nil {
				summary.ETD = te.Etd.Format(time.RFC3339)
			}
			if te.Eta != nil {
				summary.ETA = te.Eta.Format(time.RFC3339)
			}
		}
		result[link.OrderID] = summary
	}

	return result, nil
}

// listSeaMasterBillBatchNormalizedHouseNos 聚合主单批次内全部规范化分单号。
// 含作废行（不过滤状态，一号一案），供候选查询与前端即时排重提示使用。
func listSeaMasterBillBatchNormalizedHouseNos(ctx context.Context, client *ent.Client, organizationID, masterBillID uuid.UUID) ([]string, error) {
	rows, err := client.SeaHouseBill.Query().
		Where(
			seahousebillent.OrganizationIDEQ(organizationID),
			seahousebillent.MasterBillIDEQ(masterBillID),
		).
		Order(seahousebillent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	houseNos := make([]string, 0, len(rows))
	for _, row := range rows {
		houseNos = append(houseNos, row.NormalizedHouseNo)
	}
	return houseNos, nil
}

func (r *seaMasterBillRepo) getShippingLineName(ctx context.Context, client *ent.Client, organizationID, shippingLineID uuid.UUID) (string, error) {
	if shippingLineID == uuid.Nil {
		return "", nil
	}
	line, err := client.ShippingLine.Query().Where(
		shippinglineent.IDEQ(shippingLineID),
	).Only(ctx)
	if err != nil {
		return "", mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	return formatShippingLineName(line.NameZh, line.NameEn, line.ScacCode), nil
}

func formatShippingLineName(nameZH, nameEN, scacCode string) string {
	return fmt.Sprintf("%s / %s (%s)", nameZH, nameEN, scacCode)
}

func (r *seaMasterBillRepo) populateLocationAndShippingLineNames(ctx context.Context, client *ent.Client, organizationID uuid.UUID, te *biz.SeaTransportExecution) error {
	if te == nil {
		return nil
	}
	if te.ShippingLineID != uuid.Nil {
		shippingLineName, err := r.getShippingLineName(ctx, client, organizationID, te.ShippingLineID)
		if err != nil {
			return err
		}
		te.ShippingLineName = shippingLineName
	}
	locIDs := make([]uuid.UUID, 0, 3)
	seenLocationIDs := make(map[uuid.UUID]struct{}, 3)
	appendLocationID := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, exists := seenLocationIDs[id]; exists {
			return
		}
		seenLocationIDs[id] = struct{}{}
		locIDs = append(locIDs, id)
	}
	appendLocationID(te.OriginLocationID)
	appendLocationID(te.DischargeLocationID)
	if te.TransitLocationID != nil {
		appendLocationID(*te.TransitLocationID)
	}
	if len(locIDs) == 0 {
		return nil
	}
	ports, err := client.Port.Query().Where(portent.IDIn(locIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(ports) != len(locIDs) {
		return biz.ErrSeaMasterBillNotFound
	}
	portMap := make(map[uuid.UUID]string)
	for _, p := range ports {
		name := p.NameEn
		if p.NameZh != nil && *p.NameZh != "" {
			name = *p.NameZh + " / " + p.NameEn
		}
		if p.UnLocode != "" {
			name = name + " (" + p.UnLocode + ")"
		}
		portMap[p.ID] = name
	}
	te.OriginLocationName = portMap[te.OriginLocationID]
	te.DischargeLocationName = portMap[te.DischargeLocationID]
	if te.TransitLocationID != nil {
		te.TransitLocationName = portMap[*te.TransitLocationID]
	}
	return nil
}

// validateSeaLinkIdentity 校验已批量加载的读取锚点，不把异常引用投影到其他方向。
func validateSeaLinkIdentity(link *ent.SeaMasterBillOrderLink) error {
	o, m := link.Edges.Order, link.Edges.MasterBill
	if o == nil || m == nil || !biz.OrderBusinessType(o.BusinessType).IsSea() || o.OrganizationID != link.OrganizationID || m.OrganizationID != link.OrganizationID || string(o.BusinessType) != m.BusinessType || biz.OrderBusinessType(o.BusinessType).IsImport() != (o.TradeDirection == orderent.TradeDirectionImport) {
		return biz.ErrSeaDocumentStructureConflict
	}
	if te := link.Edges.TransportExecution; te != nil && te.OrganizationID != link.OrganizationID {
		return biz.ErrSeaDocumentStructureConflict
	}
	return nil
}
