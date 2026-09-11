package data

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financecommissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeeenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seaordersplitresultent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitresult"
	seasharedcontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
	shippinglineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/shippingline"
)

type seaOrderChangeRepo struct {
	data *Data
}

func NewSeaOrderChangeRepo(data *Data) biz.SeaOrderChangeRepo {
	return &seaOrderChangeRepo{data: data}
}

// ---------------------------------------------------------------------------
// 1. 动作摘要
// ---------------------------------------------------------------------------

func (r *seaOrderChangeRepo) GetChangeActions(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaOrderChangeActions, error) {
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

	actions := &biz.SeaOrderChangeActions{
		CanSplit:               true,
		CanReassign:            true,
		SplitBlockedReasons:    []string{},
		ReassignBlockedReasons: []string{},
	}

	// 1. 业务类型门禁：仅海运出口 (SE)
	if order.BusinessType != orderent.BusinessTypeSE {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "非海运出口订单不支持拆票与改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
		return actions, nil
	}

	// 2. 流程与生命周期门禁：仅未终止 (ACTIVE) 且未关单 (OPEN)
	if order.TerminationStatus != orderent.TerminationStatusACTIVE {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单已终止，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}
	if order.ClosureStatus != orderent.ClosureStatusOPEN {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单已关单，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 3. 唯一步调关系门禁
	activeLink, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrderIDEQ(orderID),
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		WithMasterBill().
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, err
		}
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单缺少活动的母单关联，无法拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
		return actions, nil
	}
	if activeLink.Edges.MasterBill == nil || activeLink.Edges.MasterBill.Status != seamasterbillent.StatusDRAFT {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "当前母单已确认或不可变，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 4. 下游财务与单证门禁检查
	// 仅检查唯一当前 HBL 是否仍为草稿；历史失效（VOIDED）HBL 不参与当前结构门禁
	nonDraftCurrentHblCount, err := client.SeaHouseBill.Query().
		Where(
			seahousebillent.OrderIDEQ(orderID),
			seahousebillent.StatusIn(seahousebillent.StatusCONFIRMED, seahousebillent.StatusRELEASED),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if nonDraftCurrentHblCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "当前分单(HBL)已确认或不可变，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 费用是否全为草稿或已作废
	nonDraftFeeCount, err := client.OrderFee.Query().
		Where(
			orderfeeent.OrderIDEQ(orderID),
			orderfeeent.StatusNEQ(orderfeeent.StatusDRAFT),
			orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if nonDraftFeeCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "存在已确认或已结算的费用，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 账单明细门禁
	activeBillLineCount, err := client.FinanceBillLine.Query().
		Where(financebilllineent.OrderIDEQ(orderID), financebilllineent.ActiveEQ(true)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if activeBillLineCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单费用已进入账单，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 提成事实门禁
	commissionCount, err := client.FinanceCommissionLine.Query().
		Where(financecommissionlineent.OrderIDEQ(orderID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if commissionCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "订单已产生提成计算事实，不允许拆票或改配"
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, msg)
		actions.ReassignBlockedReasons = append(actions.ReassignBlockedReasons, msg)
	}

	// 5. 拆票专属门禁：HOUSE 订单唯一当前 HBL 即可拆票，不要求多张 HBL
	if activeLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
		actions.CanSplit = false
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "DIRECT 当前没有 HBL 箱货分配，暂不支持部分拆票，可执行整票改配")
	} else {
		currentHblCount, err := client.SeaHouseBill.Query().
			Where(
				seahousebillent.OrderIDEQ(orderID),
				seahousebillent.StatusIn(seahousebillent.StatusDRAFT, seahousebillent.StatusCONFIRMED, seahousebillent.StatusRELEASED),
			).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if currentHblCount != 1 {
			actions.CanSplit = false
			actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "HOUSE 订单必须恰有一张当前分单(HBL)才能拆票")
		}
	}

	return actions, nil
}

// ---------------------------------------------------------------------------
// 2. 拆票上下文
// ---------------------------------------------------------------------------

func (r *seaOrderChangeRepo) GetSplitContext(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaOrderSplitContext, error) {
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

	activeLink, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrderIDEQ(orderID),
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		WithTransportExecution().
		WithMasterBill().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	mbl := activeLink.Edges.MasterBill
	if mbl == nil {
		return nil, biz.ErrSeaMasterBillNotFound
	}
	mblSummary, err := mblToSummary(ctx, client, organizationID, mbl, activeLink.Edges.TransportExecution)
	if err != nil {
		return nil, err
	}

	// 查询 HBL
	hbls, err := client.SeaHouseBill.Query().
		Where(seahousebillent.OrderIDEQ(orderID)).
		Order(seahousebillent.ByHouseNo()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	hblItems := make([]*biz.SeaOrderSplitHouseBillItem, 0, len(hbls))
	for _, h := range hbls {
		hblItems = append(hblItems, &biz.SeaOrderSplitHouseBillItem{
			ID:      h.ID,
			HouseNo: h.HouseNo,
			Status:  string(h.Status),
			Version: h.Version,
		})
	}

	// 查询 CargoItems
	cargoItems, err := client.OrderCargoItem.Query().
		Where(ordercargoitement.OrderIDEQ(orderID)).
		Order(ordercargoitement.ByCreatedAt(), ordercargoitement.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	cargoItemList := make([]*biz.SeaOrderSplitCargoItem, 0, len(cargoItems))
	for _, ci := range cargoItems {
		cargoItemList = append(cargoItemList, &biz.SeaOrderSplitCargoItem{
			ID:            ci.ID,
			CargoName:     ci.CargoName,
			PackageCount:  int32(ci.PackageCount),
			GrossWeightKg: decimal.NewFromFloat(ci.GrossWeightKg),
			VolumeCbm:     decimal.NewFromFloat(ci.VolumeCbm),
			Version:       ci.Version,
		})
	}

	// 查询 Containers
	containers, err := client.OrderContainer.Query().
		Where(ordercontainerent.OrderIDEQ(orderID)).
		Order(ordercontainerent.ByContainerNo()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	containerList := make([]*biz.SeaOrderSplitContainerItem, 0, len(containers))
	for _, c := range containers {
		cItem := &biz.SeaOrderSplitContainerItem{
			ID:              c.ID,
			ContainerNo:     c.ContainerNo,
			ContainerSpecID: c.ContainerSpecID,
			PackageCount:    int32(c.PackageCount),
			GrossWeightKg:   decimal.NewFromFloat(c.GrossWeightKg),
			VolumeCbm:       decimal.NewFromFloat(c.VolumeCbm),
			Version:         c.Version,
		}
		spec, err := client.MasterDataItem.Get(ctx, c.ContainerSpecID)
		if err != nil {
			return nil, err
		}
		cItem.ContainerSpecName = spec.Name
		containerList = append(containerList, cItem)
	}

	var currentHbl *biz.SeaOrderSplitHouseBillItem
	for _, h := range hbls {
		if h.Status != seahousebillent.StatusVOIDED {
			currentHbl = &biz.SeaOrderSplitHouseBillItem{
				ID:      h.ID,
				HouseNo: h.HouseNo,
				Status:  string(h.Status),
				Version: h.Version,
			}
			break
		}
	}

	// 查询共享箱分配
	sharedAllocRows, err := client.SeaSharedContainerAllocation.Query().
		Where(seasharedcontainerallocationent.OrderIDEQ(orderID)).
		WithSharedContainer().
		All(ctx)
	if err != nil {
		return nil, err
	}
	sharedAllocList := make([]*biz.SeaOrderSplitSharedContainerAllocationItem, 0, len(sharedAllocRows))
	for _, sca := range sharedAllocRows {
		gwDec, _ := decimal.NewFromString(sca.GrossWeightKg)
		volDec, _ := decimal.NewFromString(sca.VolumeCbm)
		item := &biz.SeaOrderSplitSharedContainerAllocationItem{
			AllocationID:  sca.ID,
			CargoItemID:   sca.CargoItemID,
			PackageCount:  int32(sca.PackageCount),
			GrossWeightKg: gwDec,
			VolumeCbm:     volDec,
		}
		if sc := sca.Edges.SharedContainer; sc != nil {
			item.SharedContainerID = sc.ID
			item.ContainerNo = sc.ContainerNo
			item.ContainerSpecID = sc.ContainerSpecID
			item.SharedContainerVersion = sc.Version
			spec, sErr := client.MasterDataItem.Get(ctx, sc.ContainerSpecID)
			if sErr == nil && spec != nil {
				item.ContainerSpecName = spec.Name
			}
		}
		sharedAllocList = append(sharedAllocList, item)
	}

	// 查询草稿费用 (排除已作废)
	fees, err := client.OrderFee.Query().
		Where(
			orderfeeent.OrderIDEQ(orderID),
			orderfeeent.StatusEQ(orderfeeent.StatusDRAFT),
		).
		Order(orderfeeent.ByFeeCode(), orderfeeent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	feeList := make([]*biz.SeaOrderSplitDraftFeeItem, 0, len(fees))
	for _, f := range fees {
		tot, err := decimal.NewFromString(f.TotalAmount)
		if err != nil {
			return nil, err
		}
		baseTot, err := decimal.NewFromString(f.BaseCurrencyAmount)
		if err != nil {
			return nil, err
		}
		item := &biz.SeaOrderSplitDraftFeeItem{
			ID:                 f.ID,
			FeeCode:            f.FeeCode,
			FeeName:            f.FeeName,
			Direction:          string(f.Direction),
			SettlementPartyID:  f.SettlementPartyID,
			Currency:           f.Currency,
			TotalAmount:        tot,
			BaseCurrency:       f.BaseCurrency,
			BaseCurrencyAmount: baseTot,
			Version:            f.Version,
		}
		sp, err := client.Partner.Query().Where(
			partnerent.IDEQ(f.SettlementPartyID),
			partnerent.OrganizationIDEQ(organizationID),
		).Only(ctx)
		if err != nil {
			return nil, err
		}
		item.SettlementPartyName = sp.LegalName
		feeList = append(feeList, item)
	}

	// 查询附件 (关联 Asset)
	attachments, err := client.OrderAttachment.Query().
		Where(orderattachmentent.OrderIDEQ(orderID)).
		WithAsset().
		Order(orderattachmentent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	attachList := make([]*biz.SeaOrderSplitAttachmentItem, 0, len(attachments))
	for _, att := range attachments {
		item := &biz.SeaOrderSplitAttachmentItem{
			ID:      att.ID,
			AssetID: att.AssetID,
			DocType: att.DocType,
		}
		if asset := att.Edges.Asset; asset != nil {
			item.FileName = asset.FileName
			item.MIMEType = asset.MimeType
			item.FileSize = asset.FileSize
		}
		attachList = append(attachList, item)
	}

	// 查询箱计划 requests
	plans, err := client.OrderContainerRequest.Query().
		Where(ordercontainerrequestent.OrderIDEQ(orderID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	planList := make([]*biz.SeaOrderSplitContainerPlanItem, 0, len(plans))
	for _, p := range plans {
		item := &biz.SeaOrderSplitContainerPlanItem{
			ContainerSpecID: p.ContainerSpecID,
			Quantity:        int32(p.Quantity),
		}
		spec, err := client.MasterDataItem.Get(ctx, p.ContainerSpecID)
		if err != nil {
			return nil, err
		}
		item.ContainerSpecName = spec.Name
		planList = append(planList, item)
	}

	shipmentType := ""
	if order.ShipmentType != nil {
		shipmentType = string(*order.ShipmentType)
	}

	splitCtx := &biz.SeaOrderSplitContext{
		OrderID:                        order.ID,
		OrderNo:                        order.OrderNo,
		BookingNo:                      order.BookingNo,
		BusinessType:                   string(order.BusinessType),
		ShipmentType:                   shipmentType,
		FlowStatus:                     string(order.FlowStatus),
		OrderVersion:                   order.Version,
		CustomerReferenceNo:            order.CustomerReferenceNo,
		InternalReferenceNo:            order.InternalReferenceNo,
		BookingNotes:                   order.BookingNotes,
		AllocationNotes:                order.AllocationNotes,
		OperationNotes:                 order.OperationNotes,
		CurrentMasterBill:              mblSummary,
		CurrentLinkID:                  activeLink.ID,
		CurrentLinkVersion:             activeLink.Version,
		DocumentStructure:              string(activeLink.DocumentStructure),
		HouseBills:                     hblItems,
		CurrentHouseBill:               currentHbl,
		CargoItems:                     cargoItemList,
		Containers:                     containerList,
		SharedContainerAllocations:     sharedAllocList,
		DraftFees:                      feeList,
		Attachments:                    attachList,
		ContainerPlans:                 planList,
		AttachmentReferenceFingerprint: biz.ComputeAttachmentFingerprint(attachList),
	}

	return splitCtx, nil
}

// ---------------------------------------------------------------------------
// 3. 拆票校验与守恒预览
// ---------------------------------------------------------------------------

func (r *seaOrderChangeRepo) PreviewSplit(ctx context.Context, organizationID uuid.UUID, input *biz.SeaOrderSplitInput) (*biz.SeaOrderSplitPreview, error) {
	splitCtx, err := r.GetSplitContext(ctx, organizationID, input.OrderID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	for _, target := range input.Targets {
		if target.TargetType == biz.SplitTargetTypeCandidate {
			candidate, queryErr := client.SeaMasterBill.Query().
				Where(
					seamasterbillent.IDEQ(*target.CandidateID),
					seamasterbillent.OrganizationIDEQ(organizationID),
				).
				WithOrderLinks(func(lq *ent.SeaMasterBillOrderLinkQuery) {
					lq.Where(seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).
						WithTransportExecution()
				}).
				Only(ctx)
			if queryErr != nil {
				if ent.IsNotFound(queryErr) {
					return nil, biz.ErrSeaOrderSplitVersionConflict
				}
				return nil, queryErr
			}
			var candidateTE *ent.SeaTransportExecution
			for _, l := range candidate.Edges.OrderLinks {
				if l.Edges.TransportExecution != nil {
					candidateTE = l.Edges.TransportExecution
					break
				}
			}
			if candidateTE == nil ||
				candidate.Status != seamasterbillent.StatusDRAFT ||
				candidate.Version != *target.CandidateVersion ||
				candidateTE.ID != *target.CandidateTEID ||
				candidateTE.Version != *target.CandidateTEVersion {
				return nil, biz.ErrSeaOrderSplitVersionConflict
			}
			if !seaMasterBillShippingLineConsistent(candidate, candidateTE) {
				return nil, biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
				})
			}
			if candidate.ShippingLineID != *target.ShippingLineID {
				return nil, biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			shippingLineEnabled, queryErr := enabledShippingLineExists(ctx, client, organizationID, candidate.ShippingLineID, false)
			if queryErr != nil {
				return nil, queryErr
			}
			if !shippingLineEnabled {
				return nil, biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
				})
			}
		} else if target.TargetType == biz.SplitTargetTypeNew {
			if _, err := validateNewMasterBillInput(ctx, client, organizationID, target, false); err != nil {
				return nil, err
			}
		}
	}

	preview := &biz.SeaOrderSplitPreview{
		IsValid:            true,
		ConservationPassed: true,
		ValidationErrors:   []*biz.SeaOrderSplitValidationError{},
	}
	actions, err := r.GetChangeActions(ctx, organizationID, input.OrderID)
	if err != nil {
		return nil, err
	}
	if !actions.CanSplit {
		preview.IsValid = false
		for _, reason := range actions.SplitBlockedReasons {
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:  "SPLIT_BLOCKED",
				Message: reason,
			})
		}
	}

	// 1. 基础校验：必须有且仅有一个 ORIGINAL，且至少一个 CREATED
	originalCount := 0
	createdCount := 0
	resultKeys := make(map[string]struct{})
	for _, res := range input.Results {
		if _, exists := resultKeys[res.ClientResultKey]; exists {
			preview.IsValid = false
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:          "DUPLICATE_RESULT_KEY",
				Message:         fmt.Sprintf("客户端结果键 %s 重复", res.ClientResultKey),
				ClientResultKey: res.ClientResultKey,
			})
		}
		resultKeys[res.ClientResultKey] = struct{}{}

		if res.ResultRole == biz.ResultRoleOriginal {
			originalCount++
		} else if res.ResultRole == biz.ResultRoleCreated {
			createdCount++
		} else {
			preview.IsValid = false
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:          "INVALID_RESULT_ROLE",
				Message:         fmt.Sprintf("未知的结果角色 %s", res.ResultRole),
				ClientResultKey: res.ClientResultKey,
			})
		}
	}

	if originalCount != 1 || createdCount < 1 {
		preview.IsValid = false
		preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
			Reason:  "INVALID_RESULT_COUNT",
			Message: "拆票结果必须包含且仅包含 1 个原票(ORIGINAL)和至少 1 个新票(CREATED)",
		})
	}

	// 2. 基线总量汇总
	baselinePkg := int32(0)
	baselineWeight := decimal.Zero
	baselineVol := decimal.Zero
	for _, ci := range splitCtx.CargoItems {
		baselinePkg += ci.PackageCount
		baselineWeight = baselineWeight.Add(ci.GrossWeightKg)
		baselineVol = baselineVol.Add(ci.VolumeCbm)
	}
	hbCount := int32(0)
	if splitCtx.CurrentHouseBill != nil {
		hbCount = 1
	}
	preview.Baseline = biz.SeaOrderSplitQuantitySummary{
		PackageCount:   baselinePkg,
		GrossWeightKg:  baselineWeight,
		VolumeCbm:      baselineVol,
		ContainerCount: int32(len(splitCtx.Containers)),
		HouseBillCount: hbCount,
		FeeCount:       int32(len(splitCtx.DraftFees)),
	}

	// 3. 校验 HBL 分单号
	seenHouseNos := make(map[string]string)
	if splitCtx.CurrentHouseBill != nil {
		normCurrent, err := biz.NormalizeSeaHouseNo(splitCtx.CurrentHouseBill.HouseNo)
		if err == nil {
			seenHouseNos[normCurrent] = "ORIGINAL"
		}
	}
	for _, res := range input.Results {
		if res.ResultRole == biz.ResultRoleCreated {
			if splitCtx.DocumentStructure == string(seamasterbillorderlinkent.DocumentStructureHOUSE) {
				if res.HouseBill == nil || strings.TrimSpace(res.HouseBill.HouseNo) == "" {
					preview.IsValid = false
					preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
						Reason:          "HOUSE_BILL_REQUIRED",
						Message:         fmt.Sprintf("新票 %s 必须提供分单号", res.ClientResultKey),
						ClientResultKey: res.ClientResultKey,
					})
				} else {
					normNo, err := biz.NormalizeSeaHouseNo(res.HouseBill.HouseNo)
					if err != nil {
						preview.IsValid = false
						preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
							Reason:          "HOUSE_BILL_INVALID",
							Message:         fmt.Sprintf("新票 %s 分单号不合法: %v", res.ClientResultKey, err),
							ClientResultKey: res.ClientResultKey,
						})
					} else {
						if prevRes, seen := seenHouseNos[normNo]; seen {
							preview.IsValid = false
							preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
								Reason:          "HOUSE_BILL_DUPLICATE",
								Message:         fmt.Sprintf("分单号 %s 重复 (与 %s 冲突)", res.HouseBill.HouseNo, prevRes),
								ClientResultKey: res.ClientResultKey,
							})
						} else {
							seenHouseNos[normNo] = res.ClientResultKey
							exists, qErr := client.SeaHouseBill.Query().Where(
								seahousebillent.OrganizationIDEQ(organizationID),
								seahousebillent.NormalizedHouseNoEQ(normNo),
								seahousebillent.StatusIn(seahousebillent.StatusDRAFT, seahousebillent.StatusCONFIRMED, seahousebillent.StatusRELEASED),
							).Exist(ctx)
							if qErr == nil && exists {
								preview.IsValid = false
								preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
									Reason:          "HOUSE_BILL_EXISTS",
									Message:         fmt.Sprintf("分单号 %s 在系统中已存在", res.HouseBill.HouseNo),
									ClientResultKey: res.ClientResultKey,
								})
							}
						}
					}
					if res.HouseBill.IssuerSource == "" {
						preview.IsValid = false
						preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
							Reason:          "HOUSE_BILL_ISSUER_REQUIRED",
							Message:         fmt.Sprintf("新票 %s 必须提供分单签发来源", res.ClientResultKey),
							ClientResultKey: res.ClientResultKey,
						})
					}
				}
			}
		}
	}

	// 4. 校验货物项分配与总量守恒
	cargoMap := make(map[uuid.UUID]*biz.SeaOrderSplitCargoItem, len(splitCtx.CargoItems))
	for _, ci := range splitCtx.CargoItems {
		cargoMap[ci.ID] = ci
	}
	allocatedPkgByItem := make(map[uuid.UUID]int32)
	allocatedWeightByItem := make(map[uuid.UUID]decimal.Decimal)
	allocatedVolByItem := make(map[uuid.UUID]decimal.Decimal)
	for ciID := range cargoMap {
		allocatedWeightByItem[ciID] = decimal.Zero
		allocatedVolByItem[ciID] = decimal.Zero
	}

	resultPkgMap := make(map[string]int32)
	resultWeightMap := make(map[string]decimal.Decimal)
	resultVolMap := make(map[string]decimal.Decimal)
	resCargoPkgByItem := make(map[string]map[uuid.UUID]int32)
	resCargoWtByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
	resCargoVolByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
	for _, res := range input.Results {
		resultWeightMap[res.ClientResultKey] = decimal.Zero
		resultVolMap[res.ClientResultKey] = decimal.Zero
		resCargoPkgByItem[res.ClientResultKey] = make(map[uuid.UUID]int32)
		resCargoWtByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
		resCargoVolByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
		for _, ca := range res.CargoAllocations {
			ci, ok := cargoMap[ca.CargoItemID]
			if !ok {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "CARGO_ITEM_NOT_FOUND",
					Message:         fmt.Sprintf("货物项 %s 不属于原订单", ca.CargoItemID),
					ClientResultKey: res.ClientResultKey,
					CargoItemID:     ca.CargoItemID.String(),
				})
				continue
			}
			if ca.PackageCount < 0 || ca.GrossWeightKg.IsNegative() || ca.VolumeCbm.IsNegative() {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "CARGO_ITEM_INVALID_QUANTITY",
					Message:         fmt.Sprintf("货物项 %s 分配数量不能为负数", ci.CargoName),
					ClientResultKey: res.ClientResultKey,
					CargoItemID:     ca.CargoItemID.String(),
				})
				continue
			}
			allocatedPkgByItem[ca.CargoItemID] += ca.PackageCount
			allocatedWeightByItem[ca.CargoItemID] = allocatedWeightByItem[ca.CargoItemID].Add(ca.GrossWeightKg)
			allocatedVolByItem[ca.CargoItemID] = allocatedVolByItem[ca.CargoItemID].Add(ca.VolumeCbm)

			resultPkgMap[res.ClientResultKey] += ca.PackageCount
			resultWeightMap[res.ClientResultKey] = resultWeightMap[res.ClientResultKey].Add(ca.GrossWeightKg)
			resultVolMap[res.ClientResultKey] = resultVolMap[res.ClientResultKey].Add(ca.VolumeCbm)

			resCargoPkgByItem[res.ClientResultKey][ca.CargoItemID] += ca.PackageCount
			resCargoWtByItem[res.ClientResultKey][ca.CargoItemID] = resCargoWtByItem[res.ClientResultKey][ca.CargoItemID].Add(ca.GrossWeightKg)
			resCargoVolByItem[res.ClientResultKey][ca.CargoItemID] = resCargoVolByItem[res.ClientResultKey][ca.CargoItemID].Add(ca.VolumeCbm)
		}
	}

	totalAllocatedPkg := int32(0)
	totalAllocatedWeight := decimal.Zero
	totalAllocatedVol := decimal.Zero

	for _, ci := range splitCtx.CargoItems {
		allocPkg := allocatedPkgByItem[ci.ID]
		allocWt := allocatedWeightByItem[ci.ID]
		allocVol := allocatedVolByItem[ci.ID]

		totalAllocatedPkg += allocPkg
		totalAllocatedWeight = totalAllocatedWeight.Add(allocWt)
		totalAllocatedVol = totalAllocatedVol.Add(allocVol)

		diffPkg := ci.PackageCount - allocPkg
		diffWt := ci.GrossWeightKg.Sub(allocWt)
		diffVol := ci.VolumeCbm.Sub(allocVol)

		if diffPkg != 0 || !diffWt.IsZero() || !diffVol.IsZero() {
			preview.ConservationPassed = false
			preview.IsValid = false
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:         "QUANTITY_CONSERVATION_FAILED",
				Message:        fmt.Sprintf("货物项 %s 拆票分配与原始总量不守恒", ci.CargoName),
				CargoItemID:    ci.ID.String(),
				BaselineValue:  fmt.Sprintf("pkg:%d, wt:%s, vol:%s", ci.PackageCount, ci.GrossWeightKg.String(), ci.VolumeCbm.String()),
				AllocatedValue: fmt.Sprintf("pkg:%d, wt:%s, vol:%s", allocPkg, allocWt.String(), allocVol.String()),
				DiffValue:      fmt.Sprintf("pkg:%d, wt:%s, vol:%s", diffPkg, diffWt.String(), diffVol.String()),
			})
		}
	}

	// 5. 校验各结果票非空
	for _, res := range input.Results {
		resPkg := resultPkgMap[res.ClientResultKey]
		resWt := resultWeightMap[res.ClientResultKey]
		resVol := resultVolMap[res.ClientResultKey]
		if resPkg <= 0 && resWt.IsZero() && resVol.IsZero() {
			preview.IsValid = false
			reason := "CREATED_ORDER_EMPTY"
			msg := "新票必须分配非零货物"
			if res.ResultRole == biz.ResultRoleOriginal {
				reason = "ORIGINAL_ORDER_EMPTY"
				msg = "原票必须保留非零货物，不可将全部货物拆出"
			}
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:          reason,
				Message:         msg,
				ClientResultKey: res.ClientResultKey,
			})
		}
	}

	// 6. 校验独占集装箱分配
	containerMap := make(map[uuid.UUID]*biz.SeaOrderSplitContainerItem, len(splitCtx.Containers))
	for _, c := range splitCtx.Containers {
		containerMap[c.ID] = c
	}
	containerToResultKey := make(map[uuid.UUID]string)
	for _, res := range input.Results {
		for _, cid := range res.ContainerIDs {
			c, ok := containerMap[cid]
			if !ok {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "CONTAINER_NOT_FOUND",
					Message:         fmt.Sprintf("集装箱 %s 不属于当前订单", cid),
					ClientResultKey: res.ClientResultKey,
					ContainerID:     cid.String(),
				})
				continue
			}
			if prevRes, seen := containerToResultKey[cid]; seen {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "CONTAINER_CROSSES_RESULTS",
					Message:         fmt.Sprintf("集装箱 %s(%s) 被同时分配到 %s 和 %s", c.ContainerNo, cid, prevRes, res.ClientResultKey),
					ClientResultKey: res.ClientResultKey,
					ContainerID:     cid.String(),
				})
			}
			containerToResultKey[cid] = res.ClientResultKey
		}
	}
	for cid, c := range containerMap {
		if _, seen := containerToResultKey[cid]; !seen {
			preview.IsValid = false
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:      "CONTAINER_UNASSIGNED",
				Message:     fmt.Sprintf("集装箱 %s(%s) 未被分配到任何结果票", c.ContainerNo, cid),
				ContainerID: cid.String(),
			})
		}
	}

	// 7. 校验共享箱分配守恒
	if len(splitCtx.SharedContainerAllocations) > 0 {
		sharedMap := make(map[uuid.UUID]*biz.SeaOrderSplitSharedContainerAllocationItem, len(splitCtx.SharedContainerAllocations))
		for _, sc := range splitCtx.SharedContainerAllocations {
			sharedMap[sc.AllocationID] = sc
		}
		allocPkgByShared := make(map[uuid.UUID]int32)
		allocWtByShared := make(map[uuid.UUID]decimal.Decimal)
		allocVolByShared := make(map[uuid.UUID]decimal.Decimal)
		for aid := range sharedMap {
			allocWtByShared[aid] = decimal.Zero
			allocVolByShared[aid] = decimal.Zero
		}
		for _, res := range input.Results {
			for _, sca := range res.SharedContainerAllocations {
				_, ok := sharedMap[sca.AllocationID]
				if !ok {
					preview.IsValid = false
					preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
						Reason:          "SHARED_ALLOCATION_NOT_FOUND",
						Message:         fmt.Sprintf("共享箱分配 %s 不属于当前订单", sca.AllocationID),
						ClientResultKey: res.ClientResultKey,
					})
					continue
				}
				if sca.PackageCount < 0 || sca.GrossWeightKg.IsNegative() || sca.VolumeCbm.IsNegative() {
					preview.IsValid = false
					preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
						Reason:          "SHARED_ALLOCATION_INVALID_QUANTITY",
						Message:         fmt.Sprintf("共享箱分配 %s 数量不能为负数", sca.AllocationID),
						ClientResultKey: res.ClientResultKey,
					})
					continue
				}
				allocPkgByShared[sca.AllocationID] += sca.PackageCount
				allocWtByShared[sca.AllocationID] = allocWtByShared[sca.AllocationID].Add(sca.GrossWeightKg)
				allocVolByShared[sca.AllocationID] = allocVolByShared[sca.AllocationID].Add(sca.VolumeCbm)
			}
		}
		for _, sc := range splitCtx.SharedContainerAllocations {
			aPkg := allocPkgByShared[sc.AllocationID]
			aWt := allocWtByShared[sc.AllocationID]
			aVol := allocVolByShared[sc.AllocationID]
			if aPkg != sc.PackageCount || !aWt.Equal(sc.GrossWeightKg) || !aVol.Equal(sc.VolumeCbm) {
				preview.ConservationPassed = false
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:         "SHARED_ALLOCATION_CONSERVATION_FAILED",
					Message:        fmt.Sprintf("共享箱分配 %s(%s) 拆票后与原始总量不守恒", sc.ContainerNo, sc.AllocationID),
					BaselineValue:  fmt.Sprintf("pkg:%d, wt:%s, vol:%s", sc.PackageCount, sc.GrossWeightKg.String(), sc.VolumeCbm.String()),
					AllocatedValue: fmt.Sprintf("pkg:%d, wt:%s, vol:%s", aPkg, aWt.String(), aVol.String()),
					DiffValue:      fmt.Sprintf("pkg:%d, wt:%s, vol:%s", sc.PackageCount-aPkg, sc.GrossWeightKg.Sub(aWt).String(), sc.VolumeCbm.Sub(aVol).String()),
				})
			}
		}

		// 7.1 交叉守恒：每个结果票内，共享箱分配合计不得超过该结果分到的同一来源货物件重尺，
		// 防止货物与共享箱两套全局守恒各自通过但结果票内部超分。
		resSharedPkgByItem := make(map[string]map[uuid.UUID]int32)
		resSharedWtByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
		resSharedVolByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
		for _, res := range input.Results {
			resSharedPkgByItem[res.ClientResultKey] = make(map[uuid.UUID]int32)
			resSharedWtByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
			resSharedVolByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
			for _, sca := range res.SharedContainerAllocations {
				orig, ok := sharedMap[sca.AllocationID]
				if !ok {
					continue
				}
				resSharedPkgByItem[res.ClientResultKey][orig.CargoItemID] += sca.PackageCount
				resSharedWtByItem[res.ClientResultKey][orig.CargoItemID] = resSharedWtByItem[res.ClientResultKey][orig.CargoItemID].Add(sca.GrossWeightKg)
				resSharedVolByItem[res.ClientResultKey][orig.CargoItemID] = resSharedVolByItem[res.ClientResultKey][orig.CargoItemID].Add(sca.VolumeCbm)
			}
		}
		for _, res := range input.Results {
			for ciID, sharedPkg := range resSharedPkgByItem[res.ClientResultKey] {
				sharedWt := resSharedWtByItem[res.ClientResultKey][ciID]
				sharedVol := resSharedVolByItem[res.ClientResultKey][ciID]
				cargoPkg := resCargoPkgByItem[res.ClientResultKey][ciID]
				cargoWt := resCargoWtByItem[res.ClientResultKey][ciID]
				cargoVol := resCargoVolByItem[res.ClientResultKey][ciID]
				if sharedPkg > cargoPkg || sharedWt.GreaterThan(cargoWt) || sharedVol.GreaterThan(cargoVol) {
					preview.ConservationPassed = false
					preview.IsValid = false
					ciName := ""
					if ci, ok := cargoMap[ciID]; ok {
						ciName = ci.CargoName
					}
					preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
						Reason:          "SHARED_ALLOCATION_EXCEEDS_RESULT_CARGO",
						Message:         fmt.Sprintf("结果票 %s 中货物项 %s 的共享箱分配(件:%d 重:%s 尺:%s)超过该票分到的货物(件:%d 重:%s 尺:%s)", res.ClientResultKey, ciName, sharedPkg, sharedWt.String(), sharedVol.String(), cargoPkg, cargoWt.String(), cargoVol.String()),
						ClientResultKey: res.ClientResultKey,
						CargoItemID:     ciID.String(),
						BaselineValue:   fmt.Sprintf("pkg:%d, wt:%s, vol:%s", cargoPkg, cargoWt.String(), cargoVol.String()),
						AllocatedValue:  fmt.Sprintf("pkg:%d, wt:%s, vol:%s", sharedPkg, sharedWt.String(), sharedVol.String()),
					})
				}
			}
		}
	}

	// 8. 校验草稿费用：每笔费用必须恰好分配到 1 个结果票
	feeMap := make(map[uuid.UUID]*biz.SeaOrderSplitDraftFeeItem)
	for _, f := range splitCtx.DraftFees {
		feeMap[f.ID] = f
	}
	feeToResultKey := make(map[uuid.UUID]string)
	for _, res := range input.Results {
		for _, feeID := range res.DraftFeeIDs {
			if _, exists := feeMap[feeID]; !exists {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "FEE_NOT_FOUND",
					Message:         fmt.Sprintf("费用 %s 不属于当前订单或非草稿状态", feeID),
					ClientResultKey: res.ClientResultKey,
					FeeID:           feeID.String(),
				})
				continue
			}
			if prevKey, exists := feeToResultKey[feeID]; exists {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "FEE_CROSSES_RESULTS",
					Message:         fmt.Sprintf("费用 %s 被同时分配到 %s 和 %s", feeID, prevKey, res.ClientResultKey),
					ClientResultKey: res.ClientResultKey,
					FeeID:           feeID.String(),
				})
			}
			feeToResultKey[feeID] = res.ClientResultKey
		}
	}
	for feeID := range feeMap {
		if _, exists := feeToResultKey[feeID]; !exists {
			preview.IsValid = false
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:  "FEE_UNASSIGNED",
				Message: fmt.Sprintf("草稿费用 %s 未被分配到任何结果票", feeID),
				FeeID:   feeID.String(),
			})
		}
	}

	// 9. 汇总每个结果票
	targetMap := make(map[string]*biz.SeaOrderSplitTargetInput, len(input.Targets))
	for _, t := range input.Targets {
		if t != nil {
			targetMap[t.ClientTargetKey] = t
		}
	}

	resultPreviews := make([]*biz.SeaOrderSplitPreviewResultItem, 0, len(input.Results))
	for _, res := range input.Results {
		target := targetMap[res.ClientTargetKey]
		if target == nil {
			return nil, biz.ErrSeaOrderSplitInvalidArgument
		}

		resPkg := resultPkgMap[res.ClientResultKey]
		resWeight := resultWeightMap[res.ClientResultKey]
		resVol := resultVolMap[res.ClientResultKey]

		resContainersCount := int32(0)
		resContainerSpecCounts := make(map[uuid.UUID]int32)
		for cID, rKey := range containerToResultKey {
			if rKey == res.ClientResultKey {
				resContainersCount++
				if c, ok := containerMap[cID]; ok {
					resContainerSpecCounts[c.ContainerSpecID]++
				}
			}
		}

		var containerPlans []*biz.SeaOrderSplitContainerPlanItem
		for _, plan := range splitCtx.ContainerPlans {
			actualInThisRes := resContainerSpecCounts[plan.ContainerSpecID]
			var calculatedQuantity int32
			if res.ResultRole == biz.ResultRoleCreated {
				calculatedQuantity = actualInThisRes
			} else {
				totalActualForSpec := int32(0)
				for _, c := range splitCtx.Containers {
					if c.ContainerSpecID == plan.ContainerSpecID {
						totalActualForSpec++
					}
				}
				unallocatedPlan := plan.Quantity - totalActualForSpec
				if unallocatedPlan < 0 {
					unallocatedPlan = 0
				}
				calculatedQuantity = actualInThisRes + unallocatedPlan
			}

			if calculatedQuantity > 0 {
				containerPlans = append(containerPlans, &biz.SeaOrderSplitContainerPlanItem{
					ContainerSpecID:   plan.ContainerSpecID,
					ContainerSpecName: plan.ContainerSpecName,
					Quantity:          calculatedQuantity,
				})
			}
		}

		bookingNotes := splitCtx.BookingNotes
		if res.BookingNotes != nil {
			bookingNotes = *res.BookingNotes
		}
		operationNotes := splitCtx.OperationNotes
		if res.OperationNotes != nil {
			operationNotes = *res.OperationNotes
		}
		allocationNotes := ""
		if res.AllocationNotes != nil {
			allocationNotes = *res.AllocationNotes
		} else if target.TargetType == biz.SplitTargetTypeCurrent {
			allocationNotes = splitCtx.AllocationNotes
		}

		var houseNo *string
		hbCount := int32(0)
		if splitCtx.DocumentStructure == string(seamasterbillorderlinkent.DocumentStructureHOUSE) {
			hbCount = 1
			if res.ResultRole == biz.ResultRoleOriginal {
				if splitCtx.CurrentHouseBill != nil {
					hNo := splitCtx.CurrentHouseBill.HouseNo
					houseNo = &hNo
				}
			} else if res.HouseBill != nil {
				hNo := res.HouseBill.HouseNo
				houseNo = &hNo
			}
		}

		resultPreviews = append(resultPreviews, &biz.SeaOrderSplitPreviewResultItem{
			ClientResultKey:     res.ClientResultKey,
			ResultRole:          res.ResultRole,
			ClientTargetKey:     res.ClientTargetKey,
			PackageCount:        resPkg,
			GrossWeightKg:       resWeight,
			VolumeCbm:           resVol,
			ContainerCount:      resContainersCount,
			HouseBillCount:      hbCount,
			FeeCount:            int32(len(res.DraftFeeIDs)),
			AttachmentCount:     int32(len(res.AttachmentReferenceIDs)),
			ContainerPlans:      containerPlans,
			InternalReferenceNo: res.InternalReferenceNo,
			BookingNotes:        bookingNotes,
			AllocationNotes:     allocationNotes,
			OperationNotes:      operationNotes,
			HouseNo:             houseNo,
		})
	}

	preview.Allocated = biz.SeaOrderSplitQuantitySummary{
		PackageCount:   totalAllocatedPkg,
		GrossWeightKg:  totalAllocatedWeight,
		VolumeCbm:      totalAllocatedVol,
		ContainerCount: preview.Baseline.ContainerCount,
		HouseBillCount: int32(len(input.Results)),
		FeeCount:       preview.Baseline.FeeCount,
	}

	remainingPkg := baselinePkg - totalAllocatedPkg
	remainingWeight := baselineWeight.Sub(totalAllocatedWeight)
	remainingVol := baselineVol.Sub(totalAllocatedVol)
	preview.Remaining = biz.SeaOrderSplitQuantitySummary{
		PackageCount:  remainingPkg,
		GrossWeightKg: remainingWeight,
		VolumeCbm:     remainingVol,
	}

	preview.Results = resultPreviews
	return preview, nil
}

// ---------------------------------------------------------------------------
// 4. 拆票原子写入与门禁
// ---------------------------------------------------------------------------

func (r *seaOrderChangeRepo) ExecuteSplit(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaOrderSplitInput, audit *biz.AuditEvent) (*biz.SeaOrderSplitEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	existingEvent, err := client.SeaOrderSplitEvent.Query().
		Where(
			seaorderspliteventent.OrganizationIDEQ(organizationID),
			seaorderspliteventent.IdempotencyKeyEQ(input.IdempotencyKey),
		).
		WithResults().
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if err == nil && existingEvent != nil {
		if existingEvent.RequestFingerprint != input.RequestFingerprint {
			return nil, biz.ErrSeaOrderSplitIdempotencyConflict
		}
		res := &biz.SeaOrderSplitEvent{
			ID:                 existingEvent.ID,
			CreatedAt:          existingEvent.CreatedAt,
			OrganizationID:     existingEvent.OrganizationID,
			SourceOrderID:      existingEvent.SourceOrderID,
			SourceOrderNo:      existingEvent.SourceOrderNo,
			IdempotencyKey:     existingEvent.IdempotencyKey,
			RequestFingerprint: existingEvent.RequestFingerprint,
			Note:               existingEvent.Note,
			BeforeSnapshot:     existingEvent.BeforeSnapshot,
			CreatedBy:          existingEvent.CreatedBy,
		}
		for _, r := range existingEvent.Edges.Results {
			res.Results = append(res.Results, &biz.SeaOrderSplitResult{
				ID:                  r.ID,
				CreatedAt:           r.CreatedAt,
				SplitEventID:        r.SplitEventID,
				OrganizationID:      r.OrganizationID,
				OrderID:             r.OrderID,
				OrderNo:             r.OrderNo,
				ResultRole:          string(r.ResultRole),
				Sequence:            r.Sequence,
				ClientResultKey:     r.ClientResultKey,
				InitialMasterBillID: r.InitialMasterBillID,
				FinalMasterBillID:   r.FinalMasterBillID,
				ResultSnapshot:      r.ResultSnapshot,
			})
		}
		return res, nil
	}

	var splitEventResult *biz.SeaOrderSplitEvent
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 锁序 1: 锁定源订单 Order
		sourceOrder, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if err := ensureOrderBusinessEditable(ctx, tx, sourceOrder); err != nil {
			return err
		}

		if input.ExpectedVersions == nil || input.ExpectedVersions.OrderVersion == 0 || sourceOrder.Version != input.ExpectedVersions.OrderVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}

		// 内嵌改配目标必须携带外部确认（领域层已校验，锁内兜底防 nil 解引用）
		hasNonCurrentTarget := false
		for _, t := range input.Targets {
			if t != nil && t.TargetType != biz.SplitTargetTypeCurrent {
				hasNonCurrentTarget = true
				break
			}
		}
		if hasNonCurrentTarget && input.Confirmation == nil {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason": "CONFIRMATION_REQUIRED",
			})
		}

		if sourceOrder.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrSeaOrderSplitBlocked
		}
		if sourceOrder.TerminationStatus != orderent.TerminationStatusACTIVE || sourceOrder.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrSeaOrderSplitBlocked
		}

		// 锁序 2: 组织订单号码序列：预先分配订单号
		allocatedAt := time.Now().UTC()
		createdOrdersCount := 0
		for _, r := range input.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrdersCount++
			}
		}

		createdOrderNumbers := make(map[string]string, createdOrdersCount)
		for _, r := range input.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				rule, sequence, allocErr := allocateNumberInTx(ctx, tx, organizationID, biz.DocumentTypeOrder, allocatedAt)
				if allocErr != nil {
					return allocErr
				}
				num, fmtErr := biz.FormatAllocatedNumber(allocatedAt, rule, sequence, string(biz.OrderBusinessSE))
				if fmtErr != nil {
					return fmtErr
				}
				createdOrderNumbers[r.ClientResultKey] = num
			}
		}

		// 锁序 3-5: 涉及的 MBL, Links, TEs (UUID 升序锁定)
		// 1) 先无锁定位当前 Link
		activeLink, linkErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(sourceOrder.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Select(seamasterbillorderlinkent.FieldID, seamasterbillorderlinkent.FieldMasterBillID, seamasterbillorderlinkent.FieldTransportExecutionID).
			Only(ctx)
		if linkErr != nil {
			if ent.IsNotFound(linkErr) {
				return biz.ErrSeaOrderSplitBlocked
			}
			return linkErr
		}

		mblIDsToLock := []uuid.UUID{activeLink.MasterBillID}
		for _, t := range input.Targets {
			if t.CandidateID != nil && *t.CandidateID != uuid.Nil {
				mblIDsToLock = append(mblIDsToLock, *t.CandidateID)
			}
		}
		mblIDsToLock = sortAndDeduplicateUUIDs(mblIDsToLock)

		lockedMBLs := make(map[uuid.UUID]*ent.SeaMasterBill, len(mblIDsToLock))
		teIDsToLock := make([]uuid.UUID, 0, len(mblIDsToLock)+len(input.Targets))
		teIDsToLock = append(teIDsToLock, activeLink.TransportExecutionID)
		for _, t := range input.Targets {
			if t.CandidateTEID != nil && *t.CandidateTEID != uuid.Nil {
				teIDsToLock = append(teIDsToLock, *t.CandidateTEID)
			}
		}
		for _, mblID := range mblIDsToLock {
			mbl, err := tx.SeaMasterBill.Query().
				Where(seamasterbillent.IDEQ(mblID), seamasterbillent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
			}
			lockedMBLs[mblID] = mbl
		}
		if lockedMBLs[activeLink.MasterBillID].Status != seamasterbillent.StatusDRAFT {
			return biz.ErrSeaOrderSplitBlocked
		}

		// 锁序 4: 锁定 Link 并重验它仍为该 Order 唯一 ACTIVE 且字段未变
		lockedActiveLink, err := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(activeLink.ID)).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return err
		}
		if lockedActiveLink.OrderID != sourceOrder.ID || lockedActiveLink.OrganizationID != organizationID ||
			lockedActiveLink.Status != seamasterbillorderlinkent.StatusACTIVE || lockedActiveLink.MasterBillID != activeLink.MasterBillID {
			return biz.ErrSeaOrderSplitBlocked
		}
		activeLinkCount, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(sourceOrder.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Count(ctx)
		if err != nil {
			return err
		}
		if activeLinkCount != 1 {
			return biz.ErrSeaOrderSplitBlocked
		}
		if input.ExpectedVersions.LinkVersion == 0 || lockedActiveLink.Version != input.ExpectedVersions.LinkVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		if lockedActiveLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaOrderSplitBlocked
		}

		// 锁序 5: 锁定 TE
		teIDsToLock = sortAndDeduplicateUUIDs(teIDsToLock)
		lockedTEs := make(map[uuid.UUID]*ent.SeaTransportExecution, len(teIDsToLock))
		for _, teID := range teIDsToLock {
			te, err := tx.SeaTransportExecution.Query().
				Where(seatransportexecutionent.IDEQ(teID), seatransportexecutionent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return err
			}
			lockedTEs[teID] = te
		}

		// 锁后精确核对所有候选 MBL/TE 版本与目标输入兼容性
		for _, t := range input.Targets {
			if t.CandidateID != nil && *t.CandidateID != uuid.Nil {
				candID := *t.CandidateID
				candMBL := lockedMBLs[candID]
				if candMBL == nil || t.CandidateVersion == nil || candMBL.Version != *t.CandidateVersion ||
					t.CandidateTEID == nil || *t.CandidateTEID == uuid.Nil {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				if candMBL.Status != seamasterbillent.StatusDRAFT {
					return biz.ErrSeaOrderSplitBlocked
				}
				if input.ExpectedVersions.CandidateMBLVersions == nil {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				expCandVer, hasMblVer := input.ExpectedVersions.CandidateMBLVersions[candID]
				if !hasMblVer || expCandVer == 0 || candMBL.Version != expCandVer {
					return biz.ErrSeaOrderSplitVersionConflict
				}

				candTE := lockedTEs[*t.CandidateTEID]
				if candTE == nil {
					return biz.ErrSeaTransportExecutionNotFound
				}
				if !seaMasterBillShippingLineConsistent(candMBL, candTE) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
					})
				}
				shippingLineEnabled, err := enabledShippingLineExists(ctx, tx.Client(), organizationID, candMBL.ShippingLineID, true)
				if err != nil {
					return err
				}
				if !shippingLineEnabled {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
					})
				}
				if t.CandidateTEVersion == nil || candTE.Version != *t.CandidateTEVersion {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				if input.ExpectedVersions.CandidateTEVersions == nil {
					return biz.ErrSeaOrderSplitVersionConflict
				}
				expTeVer, hasTeVer := input.ExpectedVersions.CandidateTEVersions[candTE.ID]
				if !hasTeVer || expTeVer == 0 || candTE.Version != expTeVer {
					return biz.ErrSeaOrderSplitVersionConflict
				}

				// 校验目标输入与候选权威 TE/MBL 是否一致 (允许不同于原票航程，但目标输入必须与候选一致)
				if t.MasterNo != "" {
					normInputMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(t.MasterNo)
					if err != nil {
						return err
					}
					if candMBL.NormalizedMasterNo != normInputMasterNo {
						return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
							"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
						})
					}
				}
				if t.ShippingLineID != nil && *t.ShippingLineID != uuid.Nil && candMBL.ShippingLineID != *t.ShippingLineID {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.ShippingLineID != nil && *t.ShippingLineID != uuid.Nil && candTE.ShippingLineID != *t.ShippingLineID {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.VesselName != "" && candTE.VesselName != t.VesselName {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.VoyageNo != "" && candTE.VoyageNo != t.VoyageNo {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.OriginLocationID != nil && *t.OriginLocationID != uuid.Nil && (candTE.OriginLocationID == nil || *candTE.OriginLocationID != *t.OriginLocationID) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.DischargeLocationID != nil && *t.DischargeLocationID != uuid.Nil && (candTE.DischargeLocationID == nil || *candTE.DischargeLocationID != *t.DischargeLocationID) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.TransitLocationID != nil && *t.TransitLocationID != uuid.Nil && (candTE.TransitLocationID == nil || *candTE.TransitLocationID != *t.TransitLocationID) {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.ETD != "" {
					etdTime := parseOptionalTime(t.ETD)
					if etdTime == nil || candTE.Etd == nil || !etdTime.Equal(*candTE.Etd) {
						return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
							"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
						})
					}
				}
				if t.ETA != "" {
					etaTime := parseOptionalTime(t.ETA)
					if etaTime == nil || candTE.Eta == nil || !etaTime.Equal(*candTE.Eta) {
						return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
							"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
						})
					}
				}
			}
		}

		// 锁序 6-10: 货物、HBL、集装箱、分配、费用 (UUID 升序锁定)
		if input.Confirmation != nil {
			if err := validateConfirmationAttachment(ctx, tx.Client(), organizationID, input.OrderID, input.Confirmation); err != nil {
				return err
			}
		}
		cargoItems, err := tx.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(sourceOrder.ID)).
			Order(ordercargoitement.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		if input.ExpectedVersions.CargoItemVersions == nil || len(input.ExpectedVersions.CargoItemVersions) != len(cargoItems) {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		lockedCargoItemMap := make(map[uuid.UUID]*ent.OrderCargoItem, len(cargoItems))
		for _, ci := range cargoItems {
			expV, ok := input.ExpectedVersions.CargoItemVersions[ci.ID]
			if !ok || expV == 0 || ci.Version != expV {
				return biz.ErrSeaOrderSplitVersionConflict
			}
			lockedCargoItemMap[ci.ID] = ci
		}

		hbls, err := tx.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(sourceOrder.ID)).
			Order(seahousebillent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		lockedHBLMap := make(map[uuid.UUID]*ent.SeaHouseBill, len(hbls))
		for _, h := range hbls {
			lockedHBLMap[h.ID] = h
		}

		var curHBL *ent.SeaHouseBill
		for _, h := range hbls {
			if h.Status != seahousebillent.StatusVOIDED {
				if h.Status != seahousebillent.StatusDRAFT {
					return biz.ErrSeaOrderSplitBlocked
				}
				curHBL = h
				break
			}
		}
		if lockedActiveLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE && curHBL == nil {
			return biz.ErrSeaHouseBillNotFound
		}
		// HOUSE 拆票必须携带唯一当前 HBL 的非零期望版本；缺失或为 0 属于参数错误
		if input.ExpectedVersions.CurrentHBLVersion == nil || *input.ExpectedVersions.CurrentHBLVersion == 0 {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason": "CURRENT_HBL_VERSION_REQUIRED",
			})
		}
		if curHBL == nil || curHBL.Version != *input.ExpectedVersions.CurrentHBLVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}

		containers, err := tx.OrderContainer.Query().
			Where(ordercontainerent.OrderIDEQ(sourceOrder.ID)).
			Order(ordercontainerent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		lockedContainerMap := make(map[uuid.UUID]*ent.OrderContainer, len(containers))
		for _, c := range containers {
			lockedContainerMap[c.ID] = c
		}
		// 独占箱将被移动并递增版本，期望版本 Map 必须完整覆盖且非零；
		// 缺 Map、缺 key 或版本为 0 属于参数错误，版本不一致返回 409。
		if len(containers) > 0 {
			if input.ExpectedVersions.ContainerVersions == nil {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason": "CONTAINER_VERSION_REQUIRED",
				})
			}
			if len(input.ExpectedVersions.ContainerVersions) != len(containers) {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason": "CONTAINER_VERSION_REQUIRED",
				})
			}
			for _, c := range containers {
				expV, ok := input.ExpectedVersions.ContainerVersions[c.ID]
				if !ok || expV == 0 {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":       "CONTAINER_VERSION_REQUIRED",
						"container_id": c.ID.String(),
					})
				}
				if c.Version != expV {
					return biz.ErrSeaOrderSplitVersionConflict
				}
			}
		}

		// 共享箱固定锁序：先无锁定位本订单分配所属共享箱 ID，按序锁定 SharedContainer，
		// 再锁定 Allocation，并在锁内重验分配集合未漂移；与共享箱工作台/删除路径的
		// SharedContainer → Allocation 顺序保持一致，避免跨操作死锁。
		preReadAllocIDs, err := tx.SeaSharedContainerAllocation.Query().
			Where(
				seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
				seasharedcontainerallocationent.OrderIDEQ(sourceOrder.ID),
			).
			Order(seasharedcontainerallocationent.ByID()).
			IDs(ctx)
		if err != nil {
			return err
		}
		preReadAllocIDSet := make(map[uuid.UUID]struct{}, len(preReadAllocIDs))
		sharedContainerIDs := make([]uuid.UUID, 0, len(preReadAllocIDs))
		for _, allocID := range preReadAllocIDs {
			preReadAllocIDSet[allocID] = struct{}{}
		}
		if len(preReadAllocIDs) > 0 {
			scIDRows, err := tx.SeaSharedContainerAllocation.Query().
				Where(
					seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
					seasharedcontainerallocationent.IDIn(preReadAllocIDs...),
				).
				Select(seasharedcontainerallocationent.FieldSharedContainerID).
				All(ctx)
			if err != nil {
				return err
			}
			for _, row := range scIDRows {
				sharedContainerIDs = append(sharedContainerIDs, row.SharedContainerID)
			}
		}
		sharedContainerIDs = sortAndDeduplicateUUIDs(sharedContainerIDs)
		lockedSharedContainers := make(map[uuid.UUID]*ent.SeaSharedContainer, len(sharedContainerIDs))
		for _, scID := range sharedContainerIDs {
			sc, err := tx.SeaSharedContainer.Query().
				Where(
					seasharedcontainerent.IDEQ(scID),
					seasharedcontainerent.OrganizationIDEQ(organizationID),
				).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return err
			}
			lockedSharedContainers[scID] = sc
		}
		sharedAllocs, err := tx.SeaSharedContainerAllocation.Query().
			Where(
				seasharedcontainerallocationent.OrganizationIDEQ(organizationID),
				seasharedcontainerallocationent.OrderIDEQ(sourceOrder.ID),
			).
			Order(seasharedcontainerallocationent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		if len(sharedAllocs) != len(preReadAllocIDs) {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		lockedSharedAllocMap := make(map[uuid.UUID]*ent.SeaSharedContainerAllocation, len(sharedAllocs))
		for _, sa := range sharedAllocs {
			if _, seen := preReadAllocIDSet[sa.ID]; !seen {
				return biz.ErrSeaOrderSplitVersionConflict
			}
			lockedSharedAllocMap[sa.ID] = sa
		}
		// 拆票修改共享箱 Allocation 前，所有受影响共享箱必须携带完整且非零的期望版本；
		// 缺 Map、缺 key 或版本为 0 属于参数错误，版本不一致返回 409。
		if len(sharedContainerIDs) > 0 {
			if input.ExpectedVersions.SharedContainerVersions == nil {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason": "SHARED_CONTAINER_VERSION_REQUIRED",
				})
			}
			for _, scID := range sharedContainerIDs {
				expV, ok := input.ExpectedVersions.SharedContainerVersions[scID]
				if !ok || expV == 0 {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":              "SHARED_CONTAINER_VERSION_REQUIRED",
						"shared_container_id": scID.String(),
					})
				}
				if sc := lockedSharedContainers[scID]; sc.Version != expV {
					return biz.ErrSeaOrderSplitVersionConflict
				}
			}
		}

		fees, err := tx.OrderFee.Query().
			Where(orderfeeent.OrderIDEQ(sourceOrder.ID)).
			Order(orderfeeent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		lockedFeesMap := make(map[uuid.UUID]*ent.OrderFee, len(fees))
		draftFeeCount := 0
		for _, f := range fees {
			if f.Status != orderfeeent.StatusDRAFT && f.Status != orderfeeent.StatusCANCELLED {
				return biz.ErrSeaOrderSplitBlocked
			}
			if f.Status == orderfeeent.StatusDRAFT {
				draftFeeCount++
				lockedFeesMap[f.ID] = f
			}
		}
		if input.ExpectedVersions.FeeVersions == nil || len(input.ExpectedVersions.FeeVersions) != draftFeeCount {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		for _, f := range fees {
			if f.Status == orderfeeent.StatusDRAFT {
				expV, ok := input.ExpectedVersions.FeeVersions[f.ID]
				if !ok || expV == 0 || f.Version != expV {
					return biz.ErrSeaOrderSplitVersionConflict
				}
			}
		}

		// 锁序 11: 附件资产与引用 (UUID 升序锁定) 并精确核对指纹
		orderAttachments, err := tx.OrderAttachment.Query().
			Where(orderattachmentent.OrderIDEQ(sourceOrder.ID)).
			Order(orderattachmentent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		attItems := make([]*biz.SeaOrderSplitAttachmentItem, 0, len(orderAttachments))
		for _, oa := range orderAttachments {
			attItems = append(attItems, &biz.SeaOrderSplitAttachmentItem{
				ID:      oa.ID,
				AssetID: oa.AssetID,
				DocType: oa.DocType,
			})
		}
		currentAttFp := biz.ComputeAttachmentFingerprint(attItems)
		if input.ExpectedVersions.AttachmentReferenceFingerprint == "" || input.ExpectedVersions.AttachmentReferenceFingerprint != currentAttFp {
			return biz.ErrSeaOrderSplitVersionConflict
		}

		// 锁序 12: 下游门禁重验 (账单、核销、提成)
		hasActiveBillLine, err := tx.FinanceBillLine.Query().
			Where(financebilllineent.OrderIDEQ(sourceOrder.ID), financebilllineent.ActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasActiveBillLine {
			return biz.ErrSeaOrderSplitBlocked
		}

		hasCommission, err := tx.FinanceCommissionLine.Query().
			Where(financecommissionlineent.OrderIDEQ(sourceOrder.ID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasCommission {
			return biz.ErrSeaOrderSplitBlocked
		}

		// -------------------------------------------------------------------
		// 锁后重算全部分配、守恒、结构、不跨箱与费用完整性
		// -------------------------------------------------------------------

		// 1. HBL 新票分单号唯一性重验
		seenHouseNos := make(map[string]string)
		if curHBL != nil {
			normCurrent, err := biz.NormalizeSeaHouseNo(curHBL.HouseNo)
			if err == nil {
				seenHouseNos[normCurrent] = "ORIGINAL"
			}
		}
		for _, res := range input.Results {
			if res.ResultRole == biz.ResultRoleCreated {
				if lockedActiveLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if res.HouseBill == nil || strings.TrimSpace(res.HouseBill.HouseNo) == "" {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":            "HOUSE_BILL_REQUIRED",
							"client_result_key": res.ClientResultKey,
						})
					}
					normNo, err := biz.NormalizeSeaHouseNo(res.HouseBill.HouseNo)
					if err != nil {
						return err
					}
					if prevRes, seen := seenHouseNos[normNo]; seen {
						return biz.MetadataError(biz.ErrSeaHouseBillExists, map[string]string{
							"reason":            "HOUSE_BILL_DUPLICATE",
							"house_no":          res.HouseBill.HouseNo,
							"conflict_result":   prevRes,
							"client_result_key": res.ClientResultKey,
						})
					}
					seenHouseNos[normNo] = res.ClientResultKey
					exists, qErr := tx.SeaHouseBill.Query().Where(
						seahousebillent.OrganizationIDEQ(organizationID),
						seahousebillent.NormalizedHouseNoEQ(normNo),
						seahousebillent.StatusIn(seahousebillent.StatusDRAFT, seahousebillent.StatusCONFIRMED, seahousebillent.StatusRELEASED),
					).Exist(ctx)
					if qErr != nil {
						return qErr
					}
					if exists {
						return biz.ErrSeaHouseBillExists
					}
					if res.HouseBill.IssuerSource == "" {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
				}
			}
		}

		// 2. 真实独占箱分配校验：每只箱恰好分配到 1 个结果
		containerToResultKey := make(map[uuid.UUID]string)
		for _, res := range input.Results {
			for _, cID := range res.ContainerIDs {
				if _, exists := lockedContainerMap[cID]; !exists {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":            "CONTAINER_NOT_FOUND",
						"container_id":      cID.String(),
						"client_result_key": res.ClientResultKey,
					})
				}
				if prevKey, seen := containerToResultKey[cID]; seen {
					return biz.MetadataError(biz.ErrSeaOrderSplitEntityCrossesResults, map[string]string{
						"reason":        "CONTAINER_CROSSES_RESULTS",
						"container_id":  cID.String(),
						"first_result":  prevKey,
						"second_result": res.ClientResultKey,
					})
				}
				containerToResultKey[cID] = res.ClientResultKey
			}
		}
		if len(containerToResultKey) != len(containers) {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason":  "CONTAINER_ALLOCATION_INCOMPLETE",
				"message": "所有集装箱必须分配且只能属于一个结果",
			})
		}

		// 3. 货物数量与重量体积严格守恒重算
		type resultCargoStat struct {
			pkg    int
			weight decimal.Decimal
			vol    decimal.Decimal
		}
		resCargoStats := make(map[string]*resultCargoStat)
		for _, res := range input.Results {
			resCargoStats[res.ClientResultKey] = &resultCargoStat{
				weight: decimal.Zero,
				vol:    decimal.Zero,
			}
		}

		allocPkgByItem := make(map[uuid.UUID]int)
		allocWtByItem := make(map[uuid.UUID]decimal.Decimal)
		allocVolByItem := make(map[uuid.UUID]decimal.Decimal)
		for _, ci := range cargoItems {
			allocWtByItem[ci.ID] = decimal.Zero
			allocVolByItem[ci.ID] = decimal.Zero
		}
		resCargoPkgByItem := make(map[string]map[uuid.UUID]int)
		resCargoWtByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
		resCargoVolByItem := make(map[string]map[uuid.UUID]decimal.Decimal)

		for _, res := range input.Results {
			rs := resCargoStats[res.ClientResultKey]
			resCargoPkgByItem[res.ClientResultKey] = make(map[uuid.UUID]int)
			resCargoWtByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
			resCargoVolByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
			for _, ca := range res.CargoAllocations {
				if _, ok := allocWtByItem[ca.CargoItemID]; !ok {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":        "CARGO_ITEM_NOT_FOUND",
						"cargo_item_id": ca.CargoItemID.String(),
					})
				}
				if ca.PackageCount < 0 || ca.GrossWeightKg.IsNegative() || ca.VolumeCbm.IsNegative() {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":        "CARGO_ITEM_INVALID_QUANTITY",
						"cargo_item_id": ca.CargoItemID.String(),
					})
				}
				allocPkgByItem[ca.CargoItemID] += int(ca.PackageCount)
				allocWtByItem[ca.CargoItemID] = allocWtByItem[ca.CargoItemID].Add(ca.GrossWeightKg)
				allocVolByItem[ca.CargoItemID] = allocVolByItem[ca.CargoItemID].Add(ca.VolumeCbm)

				rs.pkg += int(ca.PackageCount)
				rs.weight = rs.weight.Add(ca.GrossWeightKg)
				rs.vol = rs.vol.Add(ca.VolumeCbm)

				resCargoPkgByItem[res.ClientResultKey][ca.CargoItemID] += int(ca.PackageCount)
				resCargoWtByItem[res.ClientResultKey][ca.CargoItemID] = resCargoWtByItem[res.ClientResultKey][ca.CargoItemID].Add(ca.GrossWeightKg)
				resCargoVolByItem[res.ClientResultKey][ca.CargoItemID] = resCargoVolByItem[res.ClientResultKey][ca.CargoItemID].Add(ca.VolumeCbm)
			}
		}

		for _, ci := range cargoItems {
			ciAllocPkg := allocPkgByItem[ci.ID]
			ciAllocWeight := allocWtByItem[ci.ID]
			ciAllocVol := allocVolByItem[ci.ID]

			ciWeightDec := decimal.NewFromFloat(ci.GrossWeightKg)
			ciVolDec := decimal.NewFromFloat(ci.VolumeCbm)

			if ciAllocPkg != ci.PackageCount || !ciAllocWeight.Equal(ciWeightDec) || !ciAllocVol.Equal(ciVolDec) {
				return biz.MetadataError(biz.ErrSeaOrderSplitConservationFailed, map[string]string{
					"reason":        "QUANTITY_CONSERVATION_FAILED",
					"cargo_item_id": ci.ID.String(),
				})
			}
		}

		for resKey, rs := range resCargoStats {
			if rs.pkg <= 0 && rs.weight.IsZero() && rs.vol.IsZero() {
				return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
					"reason":            "EMPTY_RESULT_NOT_ALLOWED",
					"client_result_key": resKey,
				})
			}
		}

		// 3.1 共享箱分配守恒校验 (如果存在)
		if len(sharedAllocs) > 0 {
			allocPkgByShared := make(map[uuid.UUID]int)
			allocWtByShared := make(map[uuid.UUID]decimal.Decimal)
			allocVolByShared := make(map[uuid.UUID]decimal.Decimal)
			for _, sa := range sharedAllocs {
				allocWtByShared[sa.ID] = decimal.Zero
				allocVolByShared[sa.ID] = decimal.Zero
			}
			resSharedPkgByItem := make(map[string]map[uuid.UUID]int)
			resSharedWtByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
			resSharedVolByItem := make(map[string]map[uuid.UUID]decimal.Decimal)
			for _, res := range input.Results {
				resSharedPkgByItem[res.ClientResultKey] = make(map[uuid.UUID]int)
				resSharedWtByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
				resSharedVolByItem[res.ClientResultKey] = make(map[uuid.UUID]decimal.Decimal)
				for _, sca := range res.SharedContainerAllocations {
					origAlloc, exists := lockedSharedAllocMap[sca.AllocationID]
					if !exists {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":        "SHARED_ALLOCATION_NOT_FOUND",
							"allocation_id": sca.AllocationID.String(),
						})
					}
					if sca.PackageCount < 0 || sca.GrossWeightKg.IsNegative() || sca.VolumeCbm.IsNegative() {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":        "SHARED_ALLOCATION_INVALID_QUANTITY",
							"allocation_id": sca.AllocationID.String(),
						})
					}
					allocPkgByShared[sca.AllocationID] += int(sca.PackageCount)
					allocWtByShared[sca.AllocationID] = allocWtByShared[sca.AllocationID].Add(sca.GrossWeightKg)
					allocVolByShared[sca.AllocationID] = allocVolByShared[sca.AllocationID].Add(sca.VolumeCbm)

					resSharedPkgByItem[res.ClientResultKey][origAlloc.CargoItemID] += int(sca.PackageCount)
					resSharedWtByItem[res.ClientResultKey][origAlloc.CargoItemID] = resSharedWtByItem[res.ClientResultKey][origAlloc.CargoItemID].Add(sca.GrossWeightKg)
					resSharedVolByItem[res.ClientResultKey][origAlloc.CargoItemID] = resSharedVolByItem[res.ClientResultKey][origAlloc.CargoItemID].Add(sca.VolumeCbm)
				}
			}
			for _, sa := range sharedAllocs {
				sPkg := allocPkgByShared[sa.ID]
				sWt := allocWtByShared[sa.ID]
				sVol := allocVolByShared[sa.ID]
				saWt, _ := decimal.NewFromString(sa.GrossWeightKg)
				saVol, _ := decimal.NewFromString(sa.VolumeCbm)
				if sPkg != sa.PackageCount || !sWt.Equal(saWt) || !sVol.Equal(saVol) {
					return biz.MetadataError(biz.ErrSeaOrderSplitConservationFailed, map[string]string{
						"reason":        "SHARED_ALLOCATION_CONSERVATION_FAILED",
						"allocation_id": sa.ID.String(),
					})
				}
			}
			// 3.2 交叉守恒：每个结果票内，共享箱分配合计不得超过该结果分到的同一来源货物件重尺。
			for resKey, itemPkg := range resSharedPkgByItem {
				for ciID, sharedPkg := range itemPkg {
					sharedWt := resSharedWtByItem[resKey][ciID]
					sharedVol := resSharedVolByItem[resKey][ciID]
					cargoPkg := resCargoPkgByItem[resKey][ciID]
					cargoWt := resCargoWtByItem[resKey][ciID]
					cargoVol := resCargoVolByItem[resKey][ciID]
					if sharedPkg > cargoPkg || sharedWt.GreaterThan(cargoWt) || sharedVol.GreaterThan(cargoVol) {
						return biz.MetadataError(biz.ErrSeaOrderSplitConservationFailed, map[string]string{
							"reason":            "SHARED_ALLOCATION_EXCEEDS_RESULT_CARGO",
							"client_result_key": resKey,
							"cargo_item_id":     ciID.String(),
						})
					}
				}
			}
		}

		// 4. 草稿费用分配完整性与唯一性
		assignedFees := make(map[uuid.UUID]string)
		for _, res := range input.Results {
			for _, fID := range res.DraftFeeIDs {
				if _, exists := lockedFeesMap[fID]; !exists {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason": "DRAFT_FEE_NOT_FOUND",
						"fee_id": fID.String(),
					})
				}
				if prevKey, seen := assignedFees[fID]; seen {
					return biz.MetadataError(biz.ErrSeaOrderSplitEntityCrossesResults, map[string]string{
						"reason":        "FEE_CROSSES_RESULTS",
						"fee_id":        fID.String(),
						"first_result":  prevKey,
						"second_result": res.ClientResultKey,
					})
				}
				assignedFees[fID] = res.ClientResultKey
			}
		}
		if len(assignedFees) != draftFeeCount {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason":  "DRAFT_FEE_ALLOCATION_INCOMPLETE",
				"message": "所有未取消费用必须分配且只能属于一个结果",
			})
		}

		// -------------------------------------------------------------------
		// 锁序 13: 业务写入与快照
		// -------------------------------------------------------------------
		splitEventID := uuid.Must(uuid.NewV7())

		// 查询现有箱计划，用于完整 before_snapshot
		existingBeforePlans, err := tx.OrderContainerRequest.Query().
			Where(ordercontainerrequestent.OrderIDEQ(sourceOrder.ID)).
			All(ctx)
		if err != nil {
			return err
		}
		beforePlanData := make([]map[string]interface{}, 0, len(existingBeforePlans))
		for _, bp := range existingBeforePlans {
			beforePlanData = append(beforePlanData, map[string]interface{}{
				"container_spec_id": bp.ContainerSpecID,
				"quantity":          bp.Quantity,
			})
		}

		beforeHblData := make([]map[string]interface{}, 0, len(hbls))
		for _, h := range hbls {
			beforeHblData = append(beforeHblData, map[string]interface{}{
				"id":       h.ID,
				"house_no": h.HouseNo,
				"status":   h.Status,
				"version":  h.Version,
			})
		}

		beforeCargoData := make([]map[string]interface{}, 0, len(cargoItems))
		for _, ci := range cargoItems {
			beforeCargoData = append(beforeCargoData, map[string]interface{}{
				"id":              ci.ID,
				"cargo_name":      ci.CargoName,
				"package_count":   ci.PackageCount,
				"gross_weight_kg": ci.GrossWeightKg,
				"volume_cbm":      ci.VolumeCbm,
				"version":         ci.Version,
			})
		}

		beforeContainerData := make([]map[string]interface{}, 0, len(containers))
		for _, c := range containers {
			beforeContainerData = append(beforeContainerData, map[string]interface{}{
				"id":                c.ID,
				"container_no":      c.ContainerNo,
				"container_spec_id": c.ContainerSpecID,
				"version":           c.Version,
			})
		}

		beforeSharedAllocData := make([]map[string]interface{}, 0, len(sharedAllocs))
		for _, sa := range sharedAllocs {
			beforeSharedAllocData = append(beforeSharedAllocData, map[string]interface{}{
				"id":                  sa.ID,
				"shared_container_id": sa.SharedContainerID,
				"cargo_item_id":       sa.CargoItemID,
				"package_count":       sa.PackageCount,
				"gross_weight_kg":     sa.GrossWeightKg,
				"volume_cbm":          sa.VolumeCbm,
			})
		}

		beforeFeeData := make([]map[string]interface{}, 0, len(fees))
		for _, f := range fees {
			beforeFeeData = append(beforeFeeData, map[string]interface{}{
				"id":           f.ID,
				"fee_code":     f.FeeCode,
				"total_amount": f.TotalAmount,
				"currency":     f.Currency,
				"status":       f.Status,
				"version":      f.Version,
			})
		}

		beforeAttData := make([]map[string]interface{}, 0, len(orderAttachments))
		for _, oa := range orderAttachments {
			beforeAttData = append(beforeAttData, map[string]interface{}{
				"id":       oa.ID,
				"asset_id": oa.AssetID,
				"doc_type": oa.DocType,
			})
		}

		beforeSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"order": map[string]interface{}{
				"id":                    sourceOrder.ID,
				"order_no":              sourceOrder.OrderNo,
				"version":               sourceOrder.Version,
				"flow_status":           sourceOrder.FlowStatus,
				"customer_id":           sourceOrder.CustomerID,
				"customer_reference_no": sourceOrder.CustomerReferenceNo,
				"internal_reference_no": sourceOrder.InternalReferenceNo,
			},
			"active_link": map[string]interface{}{
				"id":                       lockedActiveLink.ID,
				"version":                  lockedActiveLink.Version,
				"cargo_allocation_version": 0,
				"status":                   lockedActiveLink.Status,
				"master_bill_id":           lockedActiveLink.MasterBillID,
			},
			"master_bill": map[string]interface{}{
				"id":        lockedMBLs[lockedActiveLink.MasterBillID].ID,
				"master_no": lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
				"version":   lockedMBLs[lockedActiveLink.MasterBillID].Version,
				"status":    lockedMBLs[lockedActiveLink.MasterBillID].Status,
			},
			"house_bills":                  beforeHblData,
			"cargo_items":                  beforeCargoData,
			"containers":                   beforeContainerData,
			"shared_container_allocations": beforeSharedAllocData,
			"draft_fees":                   beforeFeeData,
			"attachments":                  beforeAttData,
			"container_plans":              beforePlanData,
		}
		beforeSnapshotBytes, err := json.Marshal(beforeSnapshotMap)
		if err != nil {
			return err
		}

		conservationSnapshotMap := map[string]interface{}{
			"schema_version":                     1,
			"conservation_passed":                true,
			"cargo_items_count":                  len(cargoItems),
			"house_bills_count":                  len(hbls),
			"containers_count":                   len(containers),
			"shared_container_allocations_count": len(sharedAllocs),
			"draft_fees_count":                   draftFeeCount,
		}
		conservationSnapshotBytes, err := json.Marshal(conservationSnapshotMap)
		if err != nil {
			return err
		}

		splitEventBuilder := tx.SeaOrderSplitEvent.Create().
			SetID(splitEventID).
			SetOrganizationID(organizationID).
			SetSourceOrderID(sourceOrder.ID).
			SetSourceOrderNo(sourceOrder.OrderNo).
			SetIdempotencyKey(input.IdempotencyKey).
			SetRequestFingerprint(input.RequestFingerprint).
			SetSourceOrderVersion(sourceOrder.Version).
			SetSourceLinkID(lockedActiveLink.ID).
			SetSourceLinkVersion(lockedActiveLink.Version).
			SetSourceAllocationVersion(0).
			SetBeforeSnapshot(beforeSnapshotBytes).
			SetConservationSnapshot(conservationSnapshotBytes).
			SetCreatedBy(actorID)
		if input.Note != nil {
			splitEventBuilder.SetNote(*input.Note)
		}
		savedSplitEvent, err := splitEventBuilder.Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "sea_order_split_event_idempotency_key", biz.ErrSeaOrderSplitIdempotencyConflict)
		}

		targetInputMap := make(map[string]*biz.SeaOrderSplitTargetInput)
		for _, t := range input.Targets {
			targetInputMap[t.ClientTargetKey] = t
		}

		type targetEntityInfo struct {
			MBL *ent.SeaMasterBill
			TE  *ent.SeaTransportExecution
		}
		targetEntities := make(map[string]*targetEntityInfo)

		for _, target := range input.Targets {
			if target.TargetType == biz.SplitTargetTypeCandidate {
				if target.CandidateID != nil && *target.CandidateID != uuid.Nil {
					candMBL := lockedMBLs[*target.CandidateID]
					candTE := lockedTEs[*target.CandidateTEID]
					targetEntities[target.ClientTargetKey] = &targetEntityInfo{
						MBL: candMBL,
						TE:  candTE,
					}
				}
			} else if target.TargetType == biz.SplitTargetTypeNew {
				if _, exists := targetEntities[target.ClientTargetKey]; !exists {
					newMbl, newTE, createErr := createNewMasterBillInTx(ctx, tx, organizationID, target)
					if createErr != nil {
						return createErr
					}
					lockedMBLs[newMbl.ID] = newMbl
					lockedTEs[newTE.ID] = newTE
					targetEntities[target.ClientTargetKey] = &targetEntityInfo{
						MBL: newMbl,
						TE:  newTE,
					}
				}
			}
		}

		resultOrderMap := make(map[string]uuid.UUID)
		resultOrderNoMap := make(map[string]string)
		resultFinalMblMap := make(map[string]uuid.UUID)
		resultLinkMap := make(map[string]uuid.UUID)
		reassignmentEventIDs := make([]uuid.UUID, 0)
		createdOrdersMap := make(map[string]*ent.Order)
		originalResultKey := ""

		for _, res := range input.Results {
			target := targetInputMap[res.ClientTargetKey]
			if target == nil {
				return biz.ErrSeaOrderSplitInvalidArgument
			}
			now := time.Now().UTC()

			if res.ResultRole == biz.ResultRoleOriginal {
				originalResultKey = res.ClientResultKey
				resultOrderMap[res.ClientResultKey] = sourceOrder.ID
				resultOrderNoMap[res.ClientResultKey] = sourceOrder.OrderNo

				finalMblID := lockedActiveLink.MasterBillID
				if target.TargetType != biz.SplitTargetTypeCurrent {
					tEntity := targetEntities[target.ClientTargetKey]
					if tEntity == nil {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
					reassignMblID := tEntity.MBL.ID
					reassignTEID := tEntity.TE.ID

					// 结束原票当前活动 Link
					if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(lockedActiveLink.ID).
						SetStatus(seamasterbillorderlinkent.StatusENDED).
						SetEndedAt(now).
						SetVersion(lockedActiveLink.Version + 1).
						Save(ctx); err != nil {
						return err
					}

					// 为原票创建目标 MBL 的最终 Link
					finalLinkBuilder := tx.SeaMasterBillOrderLink.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetMasterBillID(reassignMblID).
						SetTransportExecutionID(reassignTEID).
						SetOrderID(sourceOrder.ID).
						SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
						SetStatus(seamasterbillorderlinkent.StatusACTIVE).
						SetStartedAt(now).
						SetVersion(1)
					finalLink, err := finalLinkBuilder.Save(ctx)
					if err != nil {
						return err
					}
					resultLinkMap[res.ClientResultKey] = finalLink.ID

					origBeforeReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        lockedActiveLink.ID,
						"link_version":   lockedActiveLink.Version,
						"master_bill_id": lockedActiveLink.MasterBillID,
						"master_no":      lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
					}
					origAfterReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        finalLink.ID,
						"link_version":   finalLink.Version,
						"master_bill_id": reassignMblID,
						"master_no":      lockedMBLs[reassignMblID].MasterNo,
					}
					origBeforeReassignBytes, err := json.Marshal(origBeforeReassignMap)
					if err != nil {
						return err
					}
					origAfterReassignBytes, err := json.Marshal(origAfterReassignMap)
					if err != nil {
						return err
					}

					reassignEvtID := uuid.Must(uuid.NewV7())
					reassignEvt, err := tx.SeaOrderReassignmentEvent.Create().
						SetID(reassignEvtID).
						SetOrganizationID(organizationID).
						SetOrderID(sourceOrder.ID).
						SetOrderNo(sourceOrder.OrderNo).
						SetSplitEventID(splitEventID).
						SetIdempotencyKey(input.IdempotencyKey + ":reassign:" + res.ClientResultKey).
						SetRequestFingerprint(input.RequestFingerprint).
						SetPreviousMasterBillID(lockedActiveLink.MasterBillID).
						SetTargetMasterBillID(reassignMblID).
						SetPreviousTransportExecutionID(lockedActiveLink.TransportExecutionID).
						SetTargetTransportExecutionID(reassignTEID).
						SetPreviousLinkID(lockedActiveLink.ID).
						SetTargetLinkID(finalLink.ID).
						SetPreviousLinkVersion(lockedActiveLink.Version).
						SetTargetLinkVersion(1).
						SetReason("部分拆票原票改配").
						SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityTypeOWN_COMPANY).
						SetBeforeSnapshot(origBeforeReassignBytes).
						SetAfterSnapshot(origAfterReassignBytes).
						SetCreatedBy(actorID).
						SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
						SetConfirmedAt(input.Confirmation.ConfirmedAt).
						SetConfirmationNote(input.Confirmation.ConfirmationNote).
						SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).
						Save(ctx)
					if err != nil {
						return err
					}
					reassignmentEventIDs = append(reassignmentEventIDs, reassignEvt.ID)
					finalMblID = reassignMblID

					// 更新原订单权威航程投影
					targetTE := lockedTEs[reassignTEID]
					orderUpdate := tx.Order.UpdateOneID(sourceOrder.ID).
						SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					orderUpdate.SetShippingLineID(targetTE.ShippingLineID)
					if targetTE.OriginLocationID != nil {
						orderUpdate.SetOriginLocationID(*targetTE.OriginLocationID)
					} else {
						orderUpdate.ClearOriginLocationID()
					}
					if targetTE.DischargeLocationID != nil {
						orderUpdate.SetDischargeLocationID(*targetTE.DischargeLocationID)
					} else {
						orderUpdate.ClearDischargeLocationID()
					}
					if targetTE.TransitLocationID != nil {
						orderUpdate.SetTransitLocationID(*targetTE.TransitLocationID)
					} else {
						orderUpdate.ClearTransitLocationID()
					}
					if targetTE.Etd != nil {
						orderUpdate.SetEtd(targetTE.Etd.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEtd()
					}
					if targetTE.Eta != nil {
						orderUpdate.SetEta(targetTE.Eta.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEta()
					}
					if _, err := orderUpdate.Save(ctx); err != nil {
						return err
					}
				} else {
					// 留在当前母单
					resultLinkMap[res.ClientResultKey] = lockedActiveLink.ID
				}
				resultFinalMblMap[res.ClientResultKey] = finalMblID

			} else {
				// 新建子操作票
				orderNo := createdOrderNumbers[res.ClientResultKey]
				createOrder := tx.Order.Create().
					SetOrganizationID(organizationID).
					SetOrderNo(orderNo).
					SetBusinessType(sourceOrder.BusinessType).
					SetTradeDirection(sourceOrder.TradeDirection).
					SetNillableTradeTerm(sourceOrder.TradeTerm).
					SetPaymentTerm(sourceOrder.PaymentTerm).
					SetNillableShipmentType(sourceOrder.ShipmentType).
					SetNillableContainerOwnership(sourceOrder.ContainerOwnership).
					SetNillableShipmentMode(sourceOrder.ShipmentMode).
					SetCustomerID(sourceOrder.CustomerID).
					SetCustomerReferenceNo(sourceOrder.CustomerReferenceNo).
					SetShipperShortName(sourceOrder.ShipperShortName).
					SetConsigneeShortName(sourceOrder.ConsigneeShortName).
					SetNillableBookingAgentID(sourceOrder.BookingAgentID).
					SetNillableForeignAgentID(sourceOrder.ForeignAgentID).
					SetNillableShippingAgentID(sourceOrder.ShippingAgentID).
					SetNillableDestinationLocationID(sourceOrder.DestinationLocationID).
					SetOrderDate(sourceOrder.OrderDate).
					SetNotes(sourceOrder.Notes).
					SetBookingNotes(sourceOrder.BookingNotes).
					SetOperationNotes(sourceOrder.OperationNotes).
					SetFlowStatus(sourceOrder.FlowStatus).
					SetTerminationStatus(orderent.TerminationStatusACTIVE).
					SetClosureStatus(orderent.ClosureStatusOPEN).
					SetVersion(1)

				if target.TargetType != biz.SplitTargetTypeCurrent {
					tEntity := targetEntities[target.ClientTargetKey]
					if tEntity == nil {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
					targetTE := tEntity.TE
					createOrder.SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					createOrder.SetShippingLineID(targetTE.ShippingLineID)
					createOrder.SetNillableOriginLocationID(targetTE.OriginLocationID)
					createOrder.SetNillableDischargeLocationID(targetTE.DischargeLocationID)
					createOrder.SetNillableTransitLocationID(targetTE.TransitLocationID)
					if targetTE.Etd != nil {
						createOrder.SetEtd(targetTE.Etd.Format(time.RFC3339))
					}
					if targetTE.Eta != nil {
						createOrder.SetEta(targetTE.Eta.Format(time.RFC3339))
					}
				} else {
					createOrder.SetVesselVoyage(sourceOrder.VesselVoyage)
					createOrder.SetNillableShippingLineID(sourceOrder.ShippingLineID)
					createOrder.SetNillableOriginLocationID(sourceOrder.OriginLocationID)
					createOrder.SetNillableDischargeLocationID(sourceOrder.DischargeLocationID)
					createOrder.SetNillableTransitLocationID(sourceOrder.TransitLocationID)
					createOrder.SetEtd(sourceOrder.Etd)
					createOrder.SetEta(sourceOrder.Eta)
				}

				if res.InternalReferenceNo != nil && strings.TrimSpace(*res.InternalReferenceNo) != "" {
					createOrder.SetInternalReferenceNo(strings.TrimSpace(*res.InternalReferenceNo))
				} else {
					createOrder.SetInternalReferenceNo("")
				}
				if res.BookingNotes != nil {
					createOrder.SetBookingNotes(*res.BookingNotes)
				}
				if res.AllocationNotes != nil {
					createOrder.SetAllocationNotes(*res.AllocationNotes)
				} else if target == nil || target.TargetType == biz.SplitTargetTypeCurrent {
					createOrder.SetAllocationNotes(sourceOrder.AllocationNotes)
				} else {
					createOrder.SetAllocationNotes("")
				}
				if res.OperationNotes != nil {
					createOrder.SetOperationNotes(*res.OperationNotes)
				}

				newOrder, err := createOrder.Save(ctx)
				if err != nil {
					return mapEntConstraint(err, "order_organization_id_order_no", biz.ErrOrderNumberExists)
				}

				createdOrdersMap[res.ClientResultKey] = newOrder
				resultOrderMap[res.ClientResultKey] = newOrder.ID
				resultOrderNoMap[res.ClientResultKey] = newOrder.OrderNo

				// 复制人员
				sourcePersonnel, err := tx.OrderPersonnel.Query().
					Where(orderpersonnelent.OrderIDEQ(sourceOrder.ID)).
					All(ctx)
				if err != nil {
					return err
				}
				pList := make([]*biz.OrderPersonnel, 0, len(sourcePersonnel))
				for _, sp := range sourcePersonnel {
					pList = append(pList, &biz.OrderPersonnel{
						UserID:         sp.UserID,
						OrganizationID: sp.OrganizationID,
						Role:           biz.OrderPersonnelRole(sp.Role),
					})
				}
				if err := createOrderPersonnel(ctx, tx, organizationID, newOrder.ID, newOrder.OrderNo, pList); err != nil {
					return err
				}

				// 复制提成归属快照
				sourceAttributions, err := tx.OrderCommissionAttribution.Query().
					Where(ordercommissionattributionent.OrderIDEQ(sourceOrder.ID)).
					All(ctx)
				if err != nil {
					return err
				}
				attrBuilders := make([]*ent.OrderCommissionAttributionCreate, 0, len(sourceAttributions))
				for _, sa := range sourceAttributions {
					attrBuilders = append(attrBuilders, tx.OrderCommissionAttribution.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(newOrder.ID).
						SetCustomerID(sa.CustomerID).
						SetSourceAssignmentID(sa.SourceAssignmentID).
						SetEmployeeID(sa.EmployeeID).
						SetEmployeeName(sa.EmployeeName).
						SetPersonnelRole(sa.PersonnelRole).
						SetAttributedAt(sa.AttributedAt),
					)
				}
				if len(attrBuilders) > 0 {
					if _, err := tx.OrderCommissionAttribution.CreateBulk(attrBuilders...).Save(ctx); err != nil {
						return err
					}
				}

				// 复制订单组织标签
				sourceTags, err := tx.OrderEnterpriseTag.Query().
					Where(orderenterprisetagent.OrderIDEQ(sourceOrder.ID)).
					All(ctx)
				if err != nil {
					return err
				}
				tagBuilders := make([]*ent.OrderEnterpriseTagCreate, 0, len(sourceTags))
				for _, st := range sourceTags {
					tagBuilders = append(tagBuilders, tx.OrderEnterpriseTag.Create().
						SetOrganizationID(organizationID).
						SetOrderID(newOrder.ID).
						SetTagResourceID(st.TagResourceID),
					)
				}
				if len(tagBuilders) > 0 {
					if _, err := tx.OrderEnterpriseTag.CreateBulk(tagBuilders...).Save(ctx); err != nil {
						return err
					}
				}

				// 复制服务类型与品类
				serviceTypes, err := sourceOrder.QueryServiceTypes().All(ctx)
				if err != nil {
					return err
				}
				cargoCategories, err := sourceOrder.QueryCargoCategories().All(ctx)
				if err != nil {
					return err
				}
				sIDs := make([]uuid.UUID, 0, len(serviceTypes))
				for _, s := range serviceTypes {
					sIDs = append(sIDs, s.MasterDataItemID)
				}
				cIDs := make([]uuid.UUID, 0, len(cargoCategories))
				for _, c := range cargoCategories {
					cIDs = append(cIDs, c.MasterDataItemID)
				}
				if err := replaceOrderSelections(ctx, tx, newOrder.ID, sIDs, cIDs); err != nil {
					return err
				}

				// 写入来源生命周期事件
				if _, err := tx.OrderLifecycleEvent.Create().
					SetOrderID(newOrder.ID).
					SetDimension(orderlifecycleeventent.DimensionORIGIN).
					SetToStatus(string(sourceOrder.FlowStatus)).
					SetAction("CREATED_BY_SPLIT").
					SetReferenceType("SEA_ORDER_SPLIT_EVENT").
					SetReferenceID(splitEventID).
					SetOperatorID(actorID).
					Save(ctx); err != nil {
					return err
				}

				// A.1: 新票必须先建立指向来源当前 MBL 的初始 ACTIVE Link!
				initialChildLinkBuilder := tx.SeaMasterBillOrderLink.Create().
					SetID(uuid.Must(uuid.NewV7())).
					SetOrganizationID(organizationID).
					SetMasterBillID(lockedActiveLink.MasterBillID).
					SetTransportExecutionID(lockedActiveLink.TransportExecutionID).
					SetOrderID(newOrder.ID).
					SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
					SetStatus(seamasterbillorderlinkent.StatusACTIVE).
					SetStartedAt(now).
					SetVersion(1)
				initialChildLink, err := initialChildLinkBuilder.Save(ctx)
				if err != nil {
					return err
				}

				var finalMblID uuid.UUID
				var finalTEID uuid.UUID

				if target.TargetType == biz.SplitTargetTypeCurrent {
					finalMblID = lockedActiveLink.MasterBillID
					resultLinkMap[res.ClientResultKey] = initialChildLink.ID
				} else {
					tEntity := targetEntities[target.ClientTargetKey]
					if tEntity == nil {
						return biz.ErrSeaOrderSplitInvalidArgument
					}
					finalMblID = tEntity.MBL.ID
					finalTEID = tEntity.TE.ID

					// 结束该新票自己的初始 Link
					if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(initialChildLink.ID).
						SetStatus(seamasterbillorderlinkent.StatusENDED).
						SetEndedAt(now).
						SetVersion(initialChildLink.Version + 1).
						Save(ctx); err != nil {
						return err
					}

					// 建立 final Link
					finalChildLinkBuilder := tx.SeaMasterBillOrderLink.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetMasterBillID(finalMblID).
						SetTransportExecutionID(finalTEID).
						SetOrderID(newOrder.ID).
						SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
						SetStatus(seamasterbillorderlinkent.StatusACTIVE).
						SetStartedAt(now).
						SetVersion(1)
					finalChildLink, err := finalChildLinkBuilder.Save(ctx)
					if err != nil {
						return err
					}
					resultLinkMap[res.ClientResultKey] = finalChildLink.ID

					childBeforeReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        initialChildLink.ID,
						"link_version":   initialChildLink.Version,
						"master_bill_id": lockedActiveLink.MasterBillID,
						"master_no":      lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
					}
					childAfterReassignMap := map[string]interface{}{
						"schema_version": 1,
						"link_id":        finalChildLink.ID,
						"link_version":   finalChildLink.Version,
						"master_bill_id": finalMblID,
						"master_no":      lockedMBLs[finalMblID].MasterNo,
					}
					childBeforeBytes, err := json.Marshal(childBeforeReassignMap)
					if err != nil {
						return err
					}
					childAfterBytes, err := json.Marshal(childAfterReassignMap)
					if err != nil {
						return err
					}

					reassignEvtID := uuid.Must(uuid.NewV7())
					reassignEvt, err := tx.SeaOrderReassignmentEvent.Create().
						SetID(reassignEvtID).
						SetOrganizationID(organizationID).
						SetOrderID(newOrder.ID).
						SetOrderNo(newOrder.OrderNo).
						SetSplitEventID(splitEventID).
						SetIdempotencyKey(input.IdempotencyKey + ":reassign:" + res.ClientResultKey).
						SetRequestFingerprint(input.RequestFingerprint).
						SetPreviousMasterBillID(lockedActiveLink.MasterBillID).
						SetTargetMasterBillID(finalMblID).
						SetPreviousTransportExecutionID(lockedActiveLink.TransportExecutionID).
						SetTargetTransportExecutionID(finalTEID).
						SetPreviousLinkID(initialChildLink.ID). // 必须属于该结果订单!
						SetTargetLinkID(finalChildLink.ID).
						SetPreviousLinkVersion(initialChildLink.Version).
						SetTargetLinkVersion(1).
						SetReason("部分拆票新票改配").
						SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityTypeOWN_COMPANY).
						SetBeforeSnapshot(childBeforeBytes).
						SetAfterSnapshot(childAfterBytes).
						SetCreatedBy(actorID).
						SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
						SetConfirmedAt(input.Confirmation.ConfirmedAt).
						SetConfirmationNote(input.Confirmation.ConfirmationNote).
						SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID).
						Save(ctx)
					if err != nil {
						return err
					}
					reassignmentEventIDs = append(reassignmentEventIDs, reassignEvt.ID)

					// 更新新订单权威航程投影
					targetTE := lockedTEs[finalTEID]
					orderUpdate := tx.Order.UpdateOneID(newOrder.ID).
						SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					orderUpdate.SetShippingLineID(targetTE.ShippingLineID)
					if targetTE.OriginLocationID != nil {
						orderUpdate.SetOriginLocationID(*targetTE.OriginLocationID)
					} else {
						orderUpdate.ClearOriginLocationID()
					}
					if targetTE.DischargeLocationID != nil {
						orderUpdate.SetDischargeLocationID(*targetTE.DischargeLocationID)
					} else {
						orderUpdate.ClearDischargeLocationID()
					}
					if targetTE.TransitLocationID != nil {
						orderUpdate.SetTransitLocationID(*targetTE.TransitLocationID)
					} else {
						orderUpdate.ClearTransitLocationID()
					}
					if targetTE.Etd != nil {
						orderUpdate.SetEtd(targetTE.Etd.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEtd()
					}
					if targetTE.Eta != nil {
						orderUpdate.SetEta(targetTE.Eta.Format(time.RFC3339))
					} else {
						orderUpdate.ClearEta()
					}
					if _, err := orderUpdate.Save(ctx); err != nil {
						return err
					}
				}
				resultFinalMblMap[res.ClientResultKey] = finalMblID
			}
		}

		// -------------------------------------------------------------------
		// 处理 HBL
		// -------------------------------------------------------------------
		resultHouseBillMap := make(map[string]*ent.SeaHouseBill)
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			finalMblID := resultFinalMblMap[res.ClientResultKey]

			if res.ResultRole == biz.ResultRoleOriginal {
				if curHBL != nil {
					if curHBL.MasterBillID != finalMblID {
						updatedHBL, err := tx.SeaHouseBill.UpdateOneID(curHBL.ID).
							SetMasterBillID(finalMblID).
							SetVersion(curHBL.Version + 1).
							Save(ctx)
						if err != nil {
							return err
						}
						resultHouseBillMap[res.ClientResultKey] = updatedHBL
					} else {
						resultHouseBillMap[res.ClientResultKey] = curHBL
					}
				}
			} else {
				// CREATED 新票
				if lockedActiveLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE {
					if res.HouseBill == nil {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":            "HOUSE_BILL_REQUIRED",
							"client_result_key": res.ClientResultKey,
						})
					}
					normNo, err := biz.NormalizeSeaHouseNo(res.HouseBill.HouseNo)
					if err != nil {
						return err
					}
					hbInput := &biz.SeaHouseBillInput{
						HouseNo:         res.HouseBill.HouseNo,
						IssuerSource:    biz.SeaHouseBillIssuerSource(res.HouseBill.IssuerSource),
						IssuerPartnerID: res.HouseBill.IssuerPartnerID,
						Note:            res.HouseBill.Note,
					}
					issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), organizationID, sourceOrder.OrganizationID, sourceOrder.CustomerID, hbInput)
					if err != nil {
						return err
					}
					b := tx.SeaHouseBill.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(targetOrderID).
						SetMasterBillID(finalMblID).
						SetHouseNo(res.HouseBill.HouseNo).
						SetNormalizedHouseNo(normNo).
						SetIssuerSource(seahousebillent.IssuerSource(res.HouseBill.IssuerSource)).
						SetStatus(seahousebillent.StatusDRAFT).
						SetVersion(1)
					if issuerOrgID != nil {
						b.SetIssuerOrganizationID(*issuerOrgID)
					}
					if issuerPartnerID != nil {
						b.SetIssuerPartnerID(*issuerPartnerID)
					}
					if res.HouseBill.Note != nil {
						b.SetNote(*res.HouseBill.Note)
					}
					createdHBL, err := b.Save(ctx)
					if err != nil {
						if ent.IsConstraintError(err) {
							return biz.ErrSeaHouseBillExists
						}
						return err
					}
					resultHouseBillMap[res.ClientResultKey] = createdHBL
				}
			}
		}

		// -------------------------------------------------------------------
		// 迁移集装箱 OrderContainer
		// -------------------------------------------------------------------
		for cID, rKey := range containerToResultKey {
			targetOrderID := resultOrderMap[rKey]
			lockedContainer := lockedContainerMap[cID]
			if lockedContainer != nil && lockedContainer.OrderID != targetOrderID {
				if _, err := tx.OrderContainer.UpdateOneID(cID).
					SetOrderID(targetOrderID).
					SetVersion(lockedContainer.Version + 1).
					Save(ctx); err != nil {
					return err
				}
			}
		}

		// -------------------------------------------------------------------
		// 货物重构与分配迁移
		// -------------------------------------------------------------------
		resultCargoOldNewMap := make(map[string]map[string]string)
		resultCargoRetainedIDs := make(map[string][]string)
		resultAllocOldNewMap := make(map[string]map[string]string)
		for _, res := range input.Results {
			resultCargoOldNewMap[res.ClientResultKey] = make(map[string]string)
			resultCargoRetainedIDs[res.ClientResultKey] = make([]string, 0)
			resultAllocOldNewMap[res.ClientResultKey] = make(map[string]string)
		}

		// 1. 处理货物 OrderCargoItem
		var zeroRemainingCargoIDs []uuid.UUID
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]

			if res.ResultRole == biz.ResultRoleCreated {
				for _, ca := range res.CargoAllocations {
					if ca.PackageCount <= 0 && ca.GrossWeightKg.IsZero() && ca.VolumeCbm.IsZero() {
						continue
					}
					origCargoItem := lockedCargoItemMap[ca.CargoItemID]
					if origCargoItem == nil {
						continue
					}
					fWeight, _ := ca.GrossWeightKg.Float64()
					fVol, _ := ca.VolumeCbm.Float64()

					createdItem, err := tx.OrderCargoItem.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(targetOrderID).
						SetCargoName(origCargoItem.CargoName).
						SetPackageCount(int(ca.PackageCount)).
						SetGrossWeightKg(fWeight).
						SetVolumeCbm(fVol).
						SetVersion(1).
						Save(ctx)
					if err != nil {
						return err
					}
					resultCargoOldNewMap[res.ClientResultKey][origCargoItem.ID.String()] = createdItem.ID.String()
				}
			} else {
				// 原票：逐个原始货物比对分配剩余
				caMap := make(map[uuid.UUID]*biz.SeaOrderSplitCargoAllocationInput)
				for _, ca := range res.CargoAllocations {
					caMap[ca.CargoItemID] = ca
				}
				for _, ci := range cargoItems {
					ca := caMap[ci.ID]
					if ca != nil && (ca.PackageCount > 0 || !ca.GrossWeightKg.IsZero() || !ca.VolumeCbm.IsZero()) {
						fWeight, _ := ca.GrossWeightKg.Float64()
						fVol, _ := ca.VolumeCbm.Float64()

						if _, err := tx.OrderCargoItem.UpdateOneID(ci.ID).
							SetPackageCount(int(ca.PackageCount)).
							SetGrossWeightKg(fWeight).
							SetVolumeCbm(fVol).
							SetVersion(ci.Version + 1).
							Save(ctx); err != nil {
							return err
						}
						resultCargoRetainedIDs[res.ClientResultKey] = append(resultCargoRetainedIDs[res.ClientResultKey], ci.ID.String())
						resultCargoOldNewMap[res.ClientResultKey][ci.ID.String()] = ci.ID.String()
					} else {
						zeroRemainingCargoIDs = append(zeroRemainingCargoIDs, ci.ID)
					}
				}
			}
		}

		// 2. 处理共享箱分配 SeaSharedContainerAllocation
		affectedSharedContainers := make(map[uuid.UUID]struct{})
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			targetHBL := resultHouseBillMap[res.ClientResultKey]
			var targetHBLID uuid.UUID
			if targetHBL != nil {
				targetHBLID = targetHBL.ID
			}

			if res.ResultRole == biz.ResultRoleCreated {
				for _, sca := range res.SharedContainerAllocations {
					if sca.PackageCount <= 0 && sca.GrossWeightKg.IsZero() && sca.VolumeCbm.IsZero() {
						continue
					}
					origAlloc := lockedSharedAllocMap[sca.AllocationID]
					if origAlloc == nil {
						continue
					}
					newCargoItemIDStr, ok := resultCargoOldNewMap[res.ClientResultKey][origAlloc.CargoItemID.String()]
					if !ok {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":        "CARGO_ITEM_MAPPING_NOT_FOUND",
							"cargo_item_id": origAlloc.CargoItemID.String(),
						})
					}
					newCargoItemID := uuid.MustParse(newCargoItemIDStr)

					createdAlloc, err := tx.SeaSharedContainerAllocation.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetSharedContainerID(origAlloc.SharedContainerID).
						SetOrderID(targetOrderID).
						SetHouseBillID(targetHBLID).
						SetCargoItemID(newCargoItemID).
						SetPackageCount(int(sca.PackageCount)).
						SetGrossWeightKg(sca.GrossWeightKg.StringFixed(3)).
						SetVolumeCbm(sca.VolumeCbm.StringFixed(6)).
						SetVersion(1).
						Save(ctx)
					if err != nil {
						return err
					}
					resultAllocOldNewMap[res.ClientResultKey][origAlloc.ID.String()] = createdAlloc.ID.String()
					affectedSharedContainers[origAlloc.SharedContainerID] = struct{}{}
				}
			} else {
				// 原票：更新剩余或删除零剩余分配；数值未变化时跳过写入避免虚假版本递增
				scaMap := make(map[uuid.UUID]*biz.SeaOrderSplitSharedContainerAllocationInput)
				for _, sca := range res.SharedContainerAllocations {
					scaMap[sca.AllocationID] = sca
				}
				for _, origAlloc := range sharedAllocs {
					sca := scaMap[origAlloc.ID]
					origWt, wtErr := decimal.NewFromString(origAlloc.GrossWeightKg)
					if wtErr != nil {
						return wtErr
					}
					origVol, volErr := decimal.NewFromString(origAlloc.VolumeCbm)
					if volErr != nil {
						return volErr
					}
					if sca != nil && (sca.PackageCount > 0 || !sca.GrossWeightKg.IsZero() || !sca.VolumeCbm.IsZero()) {
						if int(sca.PackageCount) != origAlloc.PackageCount || !sca.GrossWeightKg.Equal(origWt) || !sca.VolumeCbm.Equal(origVol) {
							if _, err := tx.SeaSharedContainerAllocation.UpdateOneID(origAlloc.ID).
								SetPackageCount(int(sca.PackageCount)).
								SetGrossWeightKg(sca.GrossWeightKg.StringFixed(3)).
								SetVolumeCbm(sca.VolumeCbm.StringFixed(6)).
								SetVersion(origAlloc.Version + 1).
								Save(ctx); err != nil {
								return err
							}
							affectedSharedContainers[origAlloc.SharedContainerID] = struct{}{}
						}
						resultAllocOldNewMap[res.ClientResultKey][origAlloc.ID.String()] = origAlloc.ID.String()
					} else {
						// 零剩余共享箱分配先删除
						if err := tx.SeaSharedContainerAllocation.DeleteOneID(origAlloc.ID).Exec(ctx); err != nil {
							return err
						}
						affectedSharedContainers[origAlloc.SharedContainerID] = struct{}{}
					}
				}
			}
		}

		// 2.1 共享箱聚合版本闭环：Allocation 发生创建、变更或删除后，
		// 每个受影响 SharedContainer 在同一事务内恰好递增一次版本，按已排序 ID 固定顺序写入。
		for _, scID := range sharedContainerIDs {
			if _, affected := affectedSharedContainers[scID]; !affected {
				continue
			}
			lockedSC := lockedSharedContainers[scID]
			if _, err := tx.SeaSharedContainer.UpdateOneID(scID).
				SetVersion(lockedSC.Version + 1).
				Save(ctx); err != nil {
				return err
			}
		}

		// 3. 彻底删除零剩余货物项 (此时其共享箱分配已删除，外键安全)
		for _, zID := range zeroRemainingCargoIDs {
			if err := tx.OrderCargoItem.DeleteOneID(zID).Exec(ctx); err != nil {
				return err
			}
		}

		// A.8: 迁移草稿费用：整行克隆到新订单，记录每个结果自己的 fee old->new ID 映射
		resultFeeOldNewMap := make(map[string]map[string]string)
		for _, res := range input.Results {
			resultFeeOldNewMap[res.ClientResultKey] = make(map[string]string)
			targetOrderID := resultOrderMap[res.ClientResultKey]
			if res.ResultRole == biz.ResultRoleCreated {
				for _, feeID := range res.DraftFeeIDs {
					oldFee := lockedFeesMap[feeID]
					newFeeID := uuid.Must(uuid.NewV7())
					newFeeIdempotencyKey := fmt.Sprintf("%s:split:%s", oldFee.IdempotencyKey, res.ClientResultKey)

					createBuilder := tx.OrderFee.Create().
						SetID(newFeeID).
						SetOrderID(targetOrderID).
						SetIdempotencyKey(newFeeIdempotencyKey).
						SetDirection(oldFee.Direction).
						SetStatus(oldFee.Status).
						SetNillableFeeSettingID(oldFee.FeeSettingID).
						SetFeeCode(oldFee.FeeCode).
						SetFeeName(oldFee.FeeName).
						SetNillableFeeNameEn(oldFee.FeeNameEn).
						SetSettlementPartyID(oldFee.SettlementPartyID).
						SetNillableBillingUnitID(oldFee.BillingUnitID).
						SetBillingUnit(oldFee.BillingUnit).
						SetNillableTaxRate(oldFee.TaxRate).
						SetNillableTaxableServiceName(oldFee.TaxableServiceName).
						SetQuantity(oldFee.Quantity).
						SetUnitPrice(oldFee.UnitPrice).
						SetTotalAmount(oldFee.TotalAmount).
						SetTaxInclusive(oldFee.TaxInclusive).
						SetNetAmount(oldFee.NetAmount).
						SetTaxAmount(oldFee.TaxAmount).
						SetCurrency(oldFee.Currency).
						SetExchangeRate(oldFee.ExchangeRate).
						SetExchangeRateSource(oldFee.ExchangeRateSource).
						SetExchangeRateDate(oldFee.ExchangeRateDate).
						SetNillableExchangeRateSettingID(oldFee.ExchangeRateSettingID).
						SetBaseCurrency(oldFee.BaseCurrency).
						SetBaseCurrencyAmount(oldFee.BaseCurrencyAmount).
						SetExpenseDate(oldFee.ExpenseDate).
						SetNote(oldFee.Note).
						SetVersion(1)

					if _, err := createBuilder.Save(ctx); err != nil {
						return err
					}

					// 复制企业标签关联
					existingTags, err := tx.OrderFeeEnterpriseTag.Query().
						Where(orderfeeenterprisetagent.OrderFeeIDEQ(oldFee.ID)).
						All(ctx)
					if err != nil {
						return err
					}
					for _, et := range existingTags {
						if _, err := tx.OrderFeeEnterpriseTag.Create().
							SetOrganizationID(et.OrganizationID).
							SetOrderFeeID(newFeeID).
							SetTagResourceID(et.TagResourceID).
							Save(ctx); err != nil {
							return err
						}
					}

					// 删除原费用的标签关联及原费用
					if _, err := tx.OrderFeeEnterpriseTag.Delete().
						Where(orderfeeenterprisetagent.OrderFeeIDEQ(oldFee.ID)).
						Exec(ctx); err != nil {
						return err
					}
					if err := tx.OrderFee.DeleteOneID(oldFee.ID).Exec(ctx); err != nil {
						return err
					}

					resultFeeOldNewMap[res.ClientResultKey][oldFee.ID.String()] = newFeeID.String()
				}
			}
		}

		// A.9: 处理附件引用 OrderAttachment (显式引用，不复制对象)
		for _, res := range input.Results {
			if res.ResultRole == biz.ResultRoleCreated {
				targetOrderID := resultOrderMap[res.ClientResultKey]
				for _, attID := range res.AttachmentReferenceIDs {
					var origAtt *ent.OrderAttachment
					for _, oa := range orderAttachments {
						if oa.ID == attID {
							origAtt = oa
							break
						}
					}
					if origAtt == nil {
						return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
							"reason":                  "ATTACHMENT_REFERENCE_NOT_FOUND",
							"attachment_reference_id": attID.String(),
						})
					}
					newKey := fmt.Sprintf("split:%s:%s", targetOrderID.String(), origAtt.AssetID.String())
					if _, err := tx.OrderAttachment.Create().
						SetOrderID(targetOrderID).
						SetAssetID(origAtt.AssetID).
						SetDocType(origAtt.DocType).
						SetIdempotencyKey(newKey).
						SetCreatedBy(actorID).
						Save(ctx); err != nil {
						return err
					}
				}
			}
		}

		// A.5: FCL 箱计划：新票=各规格实际箱数；原票=原计划减全部拆出实际箱后的未落实余量 + 原票实际箱；LCL/散杂不得生成计划
		resultContainerPlans := make(map[string][]map[string]interface{})
		for _, res := range input.Results {
			resultContainerPlans[res.ClientResultKey] = make([]map[string]interface{}, 0)
		}

		if sourceOrder.ShipmentType != nil && *sourceOrder.ShipmentType == orderent.ShipmentTypeFCL {
			origPlanBySpec := make(map[uuid.UUID]int)
			for _, bp := range existingBeforePlans {
				origPlanBySpec[bp.ContainerSpecID] = bp.Quantity
			}

			// 删除原票现有箱计划
			if _, err := tx.OrderContainerRequest.Delete().
				Where(ordercontainerrequestent.OrderIDEQ(sourceOrder.ID)).
				Exec(ctx); err != nil {
				return err
			}

			actualBySpecByResult := make(map[string]map[uuid.UUID]int)
			totalSplitOutBySpec := make(map[uuid.UUID]int)
			retainedActualBySpec := make(map[uuid.UUID]int)
			for _, res := range input.Results {
				actualBySpecByResult[res.ClientResultKey] = make(map[uuid.UUID]int)
			}
			for cID, rKey := range containerToResultKey {
				c := lockedContainerMap[cID]
				if c != nil && c.ContainerSpecID != uuid.Nil {
					actualBySpecByResult[rKey][c.ContainerSpecID]++
					if rKey != originalResultKey {
						totalSplitOutBySpec[c.ContainerSpecID]++
					} else {
						retainedActualBySpec[c.ContainerSpecID]++
					}
				}
			}

			allSpecs := make(map[uuid.UUID]bool)
			for specID := range origPlanBySpec {
				allSpecs[specID] = true
			}
			for _, specMap := range actualBySpecByResult {
				for specID := range specMap {
					allSpecs[specID] = true
				}
			}

			// 原票箱计划: 原计划减全部拆出实际箱后的未落实余量 + 原票实际箱
			for specID := range allSpecs {
				origPlan := origPlanBySpec[specID]
				splitOut := totalSplitOutBySpec[specID]
				retained := retainedActualBySpec[specID]
				planQty := (origPlan - splitOut)
				if planQty < retained {
					planQty = retained
				}
				if planQty > 0 {
					if _, err := tx.OrderContainerRequest.Create().
						SetOrderID(sourceOrder.ID).
						SetContainerSpecID(specID).
						SetQuantity(planQty).
						Save(ctx); err != nil {
						return err
					}
					resultContainerPlans[originalResultKey] = append(resultContainerPlans[originalResultKey], map[string]interface{}{
						"container_spec_id": specID,
						"quantity":          planQty,
					})
				}
			}

			// 新票箱计划: 各规格实际箱数
			for _, res := range input.Results {
				if res.ResultRole == biz.ResultRoleCreated {
					childOrderID := resultOrderMap[res.ClientResultKey]
					for specID, actualCount := range actualBySpecByResult[res.ClientResultKey] {
						if actualCount > 0 {
							if _, err := tx.OrderContainerRequest.Create().
								SetOrderID(childOrderID).
								SetContainerSpecID(specID).
								SetQuantity(actualCount).
								Save(ctx); err != nil {
								return err
							}
							resultContainerPlans[res.ClientResultKey] = append(resultContainerPlans[res.ClientResultKey], map[string]interface{}{
								"container_spec_id": specID,
								"quantity":          actualCount,
							})
						}
					}
				}
			}
		} else {
			// LCL / 散杂等不得生成计划，清理原票已存在的箱计划
			if _, err := tx.OrderContainerRequest.Delete().
				Where(ordercontainerrequestent.OrderIDEQ(sourceOrder.ID)).
				Exec(ctx); err != nil {
				return err
			}
		}

		// -------------------------------------------------------------------
		// 更新原订单版本与生命周期事件
		// -------------------------------------------------------------------
		if _, err := tx.Order.UpdateOneID(sourceOrder.ID).
			SetVersion(sourceOrder.Version + 1).
			Save(ctx); err != nil {
			return err
		}

		if _, err := tx.OrderLifecycleEvent.Create().
			SetOrderID(sourceOrder.ID).
			SetDimension(orderlifecycleeventent.DimensionFLOW).
			SetToStatus(string(sourceOrder.FlowStatus)).
			SetAction("SPLIT_SOURCE").
			SetReferenceType("SEA_ORDER_SPLIT_EVENT").
			SetReferenceID(splitEventID).
			SetOperatorID(actorID).
			Save(ctx); err != nil {
			return err
		}

		// -------------------------------------------------------------------
		// 写入 SeaOrderSplitResult 记录
		// -------------------------------------------------------------------
		createdResults := make([]*biz.SeaOrderSplitResult, 0, len(input.Results))
		for seqIdx, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			targetOrderNo := resultOrderNoMap[res.ClientResultKey]
			finalMblID := resultFinalMblMap[res.ClientResultKey]
			cStat := resCargoStats[res.ClientResultKey]

			var hbSnapshot map[string]interface{}
			if hb := resultHouseBillMap[res.ClientResultKey]; hb != nil {
				hbSnapshot = map[string]interface{}{
					"id":       hb.ID,
					"house_no": hb.HouseNo,
					"version":  hb.Version,
				}
			}

			resultSnapshotData := map[string]interface{}{
				"schema_version":            1,
				"order_id":                  targetOrderID,
				"order_no":                  targetOrderNo,
				"client_result_key":         res.ClientResultKey,
				"result_role":               res.ResultRole,
				"client_target_key":         res.ClientTargetKey,
				"package_count":             cStat.pkg,
				"gross_weight_kg":           biz.FormatDecimal3(cStat.weight),
				"volume_cbm":                biz.FormatDecimal6(cStat.vol),
				"house_bill":                hbSnapshot,
				"draft_fee_ids":             res.DraftFeeIDs,
				"attachment_reference_ids":  res.AttachmentReferenceIDs,
				"container_plans":           resultContainerPlans[res.ClientResultKey],
				"cargo_old_new_id_map":      resultCargoOldNewMap[res.ClientResultKey],
				"cargo_retained_ids":        resultCargoRetainedIDs[res.ClientResultKey],
				"allocation_old_new_id_map": resultAllocOldNewMap[res.ClientResultKey],
				"fee_old_new_id_map":        resultFeeOldNewMap[res.ClientResultKey], // 严格属于该结果
			}
			resultSnapshotBytes, err := json.Marshal(resultSnapshotData)
			if err != nil {
				return err
			}

			splitRes, err := tx.SeaOrderSplitResult.Create().
				SetID(uuid.Must(uuid.NewV7())).
				SetSplitEventID(splitEventID).
				SetOrganizationID(organizationID).
				SetOrderID(targetOrderID).
				SetOrderNo(targetOrderNo).
				SetResultRole(seaordersplitresultent.ResultRole(res.ResultRole)).
				SetSequence(seqIdx + 1).
				SetClientResultKey(res.ClientResultKey).
				SetInitialMasterBillID(lockedActiveLink.MasterBillID).
				SetFinalMasterBillID(finalMblID).
				SetResultSnapshot(resultSnapshotBytes).
				Save(ctx)
			if err != nil {
				return err
			}

			createdResults = append(createdResults, &biz.SeaOrderSplitResult{
				ID:                  splitRes.ID,
				CreatedAt:           splitRes.CreatedAt,
				SplitEventID:        splitRes.SplitEventID,
				OrganizationID:      splitRes.OrganizationID,
				OrderID:             splitRes.OrderID,
				OrderNo:             splitRes.OrderNo,
				ResultRole:          string(splitRes.ResultRole),
				Sequence:            splitRes.Sequence,
				ClientResultKey:     splitRes.ClientResultKey,
				InitialMasterBillID: splitRes.InitialMasterBillID,
				FinalMasterBillID:   splitRes.FinalMasterBillID,
				ResultSnapshot:      splitRes.ResultSnapshot,
			})
		}

		splitEventResult = &biz.SeaOrderSplitEvent{
			ID:                   savedSplitEvent.ID,
			CreatedAt:            savedSplitEvent.CreatedAt,
			OrganizationID:       savedSplitEvent.OrganizationID,
			SourceOrderID:        savedSplitEvent.SourceOrderID,
			SourceOrderNo:        savedSplitEvent.SourceOrderNo,
			IdempotencyKey:       savedSplitEvent.IdempotencyKey,
			RequestFingerprint:   savedSplitEvent.RequestFingerprint,
			Note:                 savedSplitEvent.Note,
			SourceOrderVersion:   savedSplitEvent.SourceOrderVersion,
			SourceLinkID:         savedSplitEvent.SourceLinkID,
			SourceLinkVersion:    savedSplitEvent.SourceLinkVersion,
			SourceAllocationVer:  savedSplitEvent.SourceAllocationVersion,
			BeforeSnapshot:       savedSplitEvent.BeforeSnapshot,
			ConservationSnapshot: savedSplitEvent.ConservationSnapshot,
			CreatedBy:            savedSplitEvent.CreatedBy,
			Results:              createdResults,
			ReassignmentEventIDs: reassignmentEventIDs,
		}

		audit.Details["split_event_id"] = splitEventID.String()
		audit.Details["source_order_no"] = sourceOrder.OrderNo
		audit.Details["result_count"] = fmt.Sprintf("%d", len(createdResults))
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}
	return splitEventResult, nil
}

// ---------------------------------------------------------------------------
// 5. 改配预览与原子执行
// ---------------------------------------------------------------------------

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
	differences := transportExecutionDifferences(execution, input.Input)
	executable := false
	for _, difference := range differences {
		if difference.IsDifferent {
			executable = true
			break
		}
	}
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

func (r *seaOrderChangeRepo) PreviewReassignment(ctx context.Context, organizationID uuid.UUID, input *biz.SeaOrderReassignmentInput) (*biz.SeaOrderReassignmentPreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}

	activeLink, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
			seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		WithMasterBill().
		WithTransportExecution().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	curMBL := activeLink.Edges.MasterBill
	if curMBL == nil {
		return nil, biz.ErrSeaMasterBillNotFound
	}
	curSummary, err := mblToSummary(ctx, client, organizationID, curMBL, activeLink.Edges.TransportExecution)
	if err != nil {
		return nil, err
	}

	preview := &biz.SeaOrderReassignmentPreview{
		IsValid:            true,
		Errors:             []string{},
		CurrentMasterBill:  curSummary,
		Differences:        []*biz.VoyageDifference{},
		OrderVersion:       order.Version,
		CurrentLinkVersion: activeLink.Version,
	}
	var targetSummary *biz.SeaMasterBillSummary
	targetMemberCount := int32(0)

	switch input.Target.TargetType {
	case biz.SplitTargetTypeCandidate:
		if input.Target.CandidateID == nil || *input.Target.CandidateID == uuid.Nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		candMBL, err := client.SeaMasterBill.Query().
			Where(
				seamasterbillent.IDEQ(*input.Target.CandidateID),
				seamasterbillent.OrganizationIDEQ(organizationID),
			).
			WithOrderLinks(func(lq *ent.SeaMasterBillOrderLinkQuery) {
				lq.Where(
					seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
					seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
				)
			}).
			Only(ctx)
		if err != nil {
			if !ent.IsNotFound(err) {
				return nil, err
			}
			preview.IsValid = false
			preview.Errors = append(preview.Errors, "目标母单不存在或不属于当前组织")
			return preview, nil
		}
		if input.Target.CandidateTEID == nil || input.Target.CandidateTEVersion == nil {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		candidateTE, err := client.SeaTransportExecution.Query().Where(
			seatransportexecutionent.IDEQ(*input.Target.CandidateTEID),
			seatransportexecutionent.OrganizationIDEQ(organizationID),
		).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaTransportExecutionNotFound, nil)
		}
		if candidateTE == nil ||
			candMBL.Status != seamasterbillent.StatusDRAFT ||
			candMBL.Version != *input.Target.CandidateVersion ||
			candidateTE.ID != *input.Target.CandidateTEID ||
			candidateTE.Version != *input.Target.CandidateTEVersion {
			return nil, biz.ErrSeaOrderReassignmentVersionConflict
		}
		if !seaMasterBillShippingLineConsistent(candMBL, candidateTE) {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
			})
		}
		if candMBL.ShippingLineID != *input.Target.ShippingLineID {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
			})
		}
		shippingLineEnabled, queryErr := enabledShippingLineExists(ctx, client, organizationID, candMBL.ShippingLineID, false)
		if queryErr != nil {
			return nil, queryErr
		}
		if !shippingLineEnabled {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
			})
		}
		if candidateTE.ID == activeLink.TransportExecutionID || (candMBL.ID != curMBL.ID && candMBL.ShippingLineID == curMBL.ShippingLineID) || (candMBL.ID == curMBL.ID && candidateTE.ShippingLineID != curMBL.ShippingLineID) {
			preview.IsValid = false
			preview.Errors = append(preview.Errors, "改配目标与当前 MBL/实际航次组合不符合船公司变更规则")
		}
		targetSummary, err = mblToSummary(ctx, client, organizationID, candMBL, candidateTE)
		if err != nil {
			return nil, err
		}
		targetMemberCount = int32(len(candMBL.Edges.OrderLinks))
	case biz.SplitTargetTypeNew:
		splitTarget := &biz.SeaOrderSplitTargetInput{
			MasterNo:            input.Target.MasterNo,
			ShippingLineID:      input.Target.ShippingLineID,
			VesselName:          input.Target.VesselName,
			VoyageNo:            input.Target.VoyageNo,
			ETD:                 input.Target.ETD,
			ETA:                 input.Target.ETA,
			OriginLocationID:    input.Target.OriginLocationID,
			DischargeLocationID: input.Target.DischargeLocationID,
			TransitLocationID:   input.Target.TransitLocationID,
		}
		normalizedMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(splitTarget.MasterNo)
		if err != nil {
			return nil, err
		}
		reuseCurrentMBL := input.Target.ShippingLineID != nil && *input.Target.ShippingLineID == curMBL.ShippingLineID && normalizedMasterNo == curMBL.NormalizedMasterNo
		if reuseCurrentMBL {
			if err := validateTransportExecutionTargetInput(ctx, client, organizationID, splitTarget, false); err != nil {
				return nil, err
			}
		} else {
			if input.Target.ShippingLineID != nil && *input.Target.ShippingLineID == curMBL.ShippingLineID {
				return nil, biz.ErrSeaOrderReassignmentTargetConflict
			}
			normalizedMasterNo, err = validateNewMasterBillInput(ctx, client, organizationID, splitTarget, false)
			if err != nil {
				return nil, err
			}
		}
		targetSummary = &biz.SeaMasterBillSummary{
			MasterNo:   normalizedMasterNo,
			VesselName: input.Target.VesselName,
			VoyageNo:   input.Target.VoyageNo,
			ETD:        input.Target.ETD,
			ETA:        input.Target.ETA,
		}
		if reuseCurrentMBL {
			targetSummary.MasterBillID = curMBL.ID
			targetSummary.Version = curMBL.Version
			targetSummary.Status = string(curMBL.Status)
		}
		if input.Target.ShippingLineID != nil {
			targetSummary.ShippingLineID = *input.Target.ShippingLineID
			line, err := client.ShippingLine.Query().Where(
				shippinglineent.IDEQ(*input.Target.ShippingLineID),
				shippinglineent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.ShippingLineName = formatShippingLineName(line.NameZh, line.NameEn, line.ScacCode)
		}
		if input.Target.OriginLocationID != nil {
			targetSummary.OriginLocationID = input.Target.OriginLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*input.Target.OriginLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.OriginLocationName = p.NameEn
		}
		if input.Target.DischargeLocationID != nil {
			targetSummary.DischargeLocationID = input.Target.DischargeLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*input.Target.DischargeLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.DischargeLocationName = p.NameEn
		}
		if input.Target.TransitLocationID != nil {
			targetSummary.TransitLocationID = input.Target.TransitLocationID
			p, err := client.Port.Query().Where(
				portent.IDEQ(*input.Target.TransitLocationID),
				portent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.TransitLocationName = p.NameEn
		}
	default:
		return nil, biz.ErrSeaOrderReassignmentInvalidArgument
	}

	preview.TargetMasterBill = targetSummary
	preview.TargetMemberCount = targetMemberCount

	preview.Differences = append(preview.Differences,
		makeDiff("master_no", "提单号(MBL)", curSummary.MasterNo, targetSummary.MasterNo),
		makeDiff("shipping_line_name", "承运人/船东", curSummary.ShippingLineName, targetSummary.ShippingLineName),
		makeDiff("vessel_name", "船名", curSummary.VesselName, targetSummary.VesselName),
		makeDiff("voyage_no", "航次", curSummary.VoyageNo, targetSummary.VoyageNo),
		makeDiff("origin_location_name", "起运港(POL)", curSummary.OriginLocationName, targetSummary.OriginLocationName),
		makeDiff("discharge_location_name", "卸货港(POD)", curSummary.DischargeLocationName, targetSummary.DischargeLocationName),
		makeDiff("transit_location_name", "中转港", curSummary.TransitLocationName, targetSummary.TransitLocationName),
		makeDiff("etd", "预计开航时间(ETD)", curSummary.ETD, targetSummary.ETD),
		makeDiff("eta", "预计到港时间(ETA)", curSummary.ETA, targetSummary.ETA),
	)

	_ = order
	return preview, nil
}

func (r *seaOrderChangeRepo) ExecuteReassignment(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.SeaOrderReassignmentInput, audit *biz.AuditEvent) (*biz.SeaOrderReassignmentEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	existingEvent, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.IdempotencyKeyEQ(input.IdempotencyKey),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if err == nil && existingEvent != nil {
		if existingEvent.RequestFingerprint != input.RequestFingerprint {
			return nil, biz.ErrSeaOrderReassignmentIdempotencyConflict
		}
		return &biz.SeaOrderReassignmentEvent{
			ID:                   existingEvent.ID,
			CreatedAt:            existingEvent.CreatedAt,
			OrganizationID:       existingEvent.OrganizationID,
			OrderID:              existingEvent.OrderID,
			OrderNo:              existingEvent.OrderNo,
			IdempotencyKey:       existingEvent.IdempotencyKey,
			RequestFingerprint:   existingEvent.RequestFingerprint,
			PreviousMasterBillID: existingEvent.PreviousMasterBillID,
			TargetMasterBillID:   existingEvent.TargetMasterBillID,
			TargetLinkID:         existingEvent.TargetLinkID,
			Reason:               existingEvent.Reason,
			ResponsibilityType:   string(existingEvent.ResponsibilityType),
			Confirmation:         externalConfirmationFromReassignment(existingEvent),
		}, nil
	}

	var reassignmentResult *biz.SeaOrderReassignmentEvent
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 锁序 1: Order ForUpdate
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if input.ExpectedOrderVersion == 0 || order.Version != input.ExpectedOrderVersion {
			return biz.ErrSeaOrderReassignmentVersionConflict
		}

		if order.BusinessType != orderent.BusinessTypeSE {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		if order.TerminationStatus != orderent.TerminationStatusACTIVE || order.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		// 改配属订单业务内容写入：在既有终止/结案检查之上执行统一内容门禁，
		// 业务锁定的订单不允许改配（ORDER_BUSINESS_LOCKED，409）。
		if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
			return err
		}

		// C.1: 锁序改造：先无锁定位 Link，仅作 ID 定位
		unlockedLink, linkErr := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(order.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Select(seamasterbillorderlinkent.FieldID, seamasterbillorderlinkent.FieldMasterBillID).
			Only(ctx)
		if linkErr != nil {
			if ent.IsNotFound(linkErr) {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			return linkErr
		}

		// 按 UUID 升序收集并锁定 MBL
		mblIDs := []uuid.UUID{unlockedLink.MasterBillID}
		if input.Target.TargetType == biz.SplitTargetTypeCandidate && input.Target.CandidateID != nil && *input.Target.CandidateID != uuid.Nil {
			mblIDs = append(mblIDs, *input.Target.CandidateID)
		}
		mblIDs = sortAndDeduplicateUUIDs(mblIDs)

		mbls := make(map[uuid.UUID]*ent.SeaMasterBill, len(mblIDs))
		for _, mID := range mblIDs {
			m, err := tx.SeaMasterBill.Query().
				Where(seamasterbillent.IDEQ(mID), seamasterbillent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
			}
			mbls[mID] = m
		}

		// 锁定 Link 并重验它仍为该 Order 唯一 ACTIVE 且 master_bill_id/status/org/version 未变
		oldLink, linkErr := tx.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.IDEQ(unlockedLink.ID)).
			ForUpdate().
			Only(ctx)
		if linkErr != nil {
			if ent.IsNotFound(linkErr) {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			return linkErr
		}
		if oldLink.OrderID != order.ID || oldLink.OrganizationID != organizationID ||
			oldLink.Status != seamasterbillorderlinkent.StatusACTIVE || oldLink.MasterBillID != unlockedLink.MasterBillID {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		activeLinkCount, err := tx.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(order.ID),
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Count(ctx)
		if err != nil {
			return err
		}
		if activeLinkCount != 1 {
			return biz.ErrSeaOrderReassignmentBlocked
		}
		if input.ExpectedLinkVersion == 0 || oldLink.Version != input.ExpectedLinkVersion {
			return biz.ErrSeaOrderReassignmentVersionConflict
		}

		oldMBL := mbls[oldLink.MasterBillID]
		if oldMBL == nil || oldMBL.Status != seamasterbillent.StatusDRAFT {
			return biz.ErrSeaOrderReassignmentBlocked
		}

		// 收集并锁定 TE
		teIDs := []uuid.UUID{oldLink.TransportExecutionID}
		if input.Target.TargetType == biz.SplitTargetTypeCandidate && input.Target.CandidateTEID != nil && *input.Target.CandidateTEID != uuid.Nil {
			teIDs = append(teIDs, *input.Target.CandidateTEID)
		}
		teIDs = sortAndDeduplicateUUIDs(teIDs)
		lockedTEs := make(map[uuid.UUID]*ent.SeaTransportExecution, len(teIDs))
		for _, tid := range teIDs {
			te, err := tx.SeaTransportExecution.Query().
				Where(seatransportexecutionent.IDEQ(tid), seatransportexecutionent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return err
			}
			lockedTEs[tid] = te
		}
		oldTE := lockedTEs[oldLink.TransportExecutionID]
		if err := validateConfirmationAttachment(ctx, tx.Client(), organizationID, order.ID, input.Confirmation); err != nil {
			return err
		}

		var targetMBLID uuid.UUID
		var targetTEID uuid.UUID
		var targetMBLNo string
		var targetTE *ent.SeaTransportExecution

		switch input.Target.TargetType {
		case biz.SplitTargetTypeCandidate:
			if input.Target.CandidateID == nil || *input.Target.CandidateID == uuid.Nil {
				return biz.ErrSeaOrderReassignmentInvalidArgument
			}
			targetMBLID = *input.Target.CandidateID
			targetMBL := mbls[targetMBLID]
			if targetMBL == nil || input.Target.CandidateVersion == nil || targetMBL.Version != *input.Target.CandidateVersion ||
				input.Target.CandidateTEID == nil || *input.Target.CandidateTEID == uuid.Nil {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}
			if targetMBL.Status != seamasterbillent.StatusDRAFT {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			if input.ExpectedCandidateMBLVersion == nil || *input.ExpectedCandidateMBLVersion == 0 || targetMBL.Version != *input.ExpectedCandidateMBLVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}
			targetMBLNo = targetMBL.MasterNo
			targetTEID = *input.Target.CandidateTEID
			targetTE = lockedTEs[targetTEID]
			if targetTE == nil {
				return biz.ErrSeaTransportExecutionNotFound
			}
			if !seaMasterBillShippingLineConsistent(targetMBL, targetTE) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_SHIPPING_LINE_INCONSISTENT",
				})
			}
			shippingLineEnabled, err := enabledShippingLineExists(ctx, tx.Client(), organizationID, targetMBL.ShippingLineID, true)
			if err != nil {
				return err
			}
			if !shippingLineEnabled {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_SHIPPING_LINE_UNAVAILABLE",
				})
			}
			if input.Target.CandidateTEVersion == nil || targetTE.Version != *input.Target.CandidateTEVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}

			if input.ExpectedCandidateTEVersion == nil || *input.ExpectedCandidateTEVersion == 0 || targetTE.Version != *input.ExpectedCandidateTEVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}

			// C.2: 校验用户目标输入与 candidate authoritative TE/MBL 是否一致 (禁止比对源订单 origin/discharge)
			if input.Target.MasterNo != "" {
				normInputMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(input.Target.MasterNo)
				if err != nil {
					return err
				}
				if targetMBL.NormalizedMasterNo != normInputMasterNo {
					return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
			}
			if input.Target.ShippingLineID != nil && *input.Target.ShippingLineID != uuid.Nil && targetMBL.ShippingLineID != *input.Target.ShippingLineID {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.ShippingLineID != nil && *input.Target.ShippingLineID != uuid.Nil && targetTE.ShippingLineID != *input.Target.ShippingLineID {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.VesselName != "" && targetTE.VesselName != input.Target.VesselName {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.VoyageNo != "" && targetTE.VoyageNo != input.Target.VoyageNo {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.OriginLocationID != nil && *input.Target.OriginLocationID != uuid.Nil && (targetTE.OriginLocationID == nil || *targetTE.OriginLocationID != *input.Target.OriginLocationID) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.DischargeLocationID != nil && *input.Target.DischargeLocationID != uuid.Nil && (targetTE.DischargeLocationID == nil || *targetTE.DischargeLocationID != *input.Target.DischargeLocationID) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.TransitLocationID != nil && *input.Target.TransitLocationID != uuid.Nil && (targetTE.TransitLocationID == nil || *targetTE.TransitLocationID != *input.Target.TransitLocationID) {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.ETD != "" {
				etdTime := parseOptionalTime(input.Target.ETD)
				if etdTime == nil || targetTE.Etd == nil || !etdTime.Equal(*targetTE.Etd) {
					return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
			}
			if input.Target.ETA != "" {
				etaTime := parseOptionalTime(input.Target.ETA)
				if etaTime == nil || targetTE.Eta == nil || !etaTime.Equal(*targetTE.Eta) {
					return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
			}
		case biz.SplitTargetTypeNew:
			splitTargetInput := &biz.SeaOrderSplitTargetInput{
				MasterNo:            input.Target.MasterNo,
				ShippingLineID:      input.Target.ShippingLineID,
				VesselName:          input.Target.VesselName,
				VoyageNo:            input.Target.VoyageNo,
				ETD:                 input.Target.ETD,
				ETA:                 input.Target.ETA,
				OriginLocationID:    input.Target.OriginLocationID,
				DischargeLocationID: input.Target.DischargeLocationID,
				TransitLocationID:   input.Target.TransitLocationID,
			}
			newMBL, newTE, err := createNewMasterBillInTx(ctx, tx, organizationID, splitTargetInput)
			if err != nil {
				return err
			}
			targetMBLID = newMBL.ID
			targetTEID = newTE.ID
			targetMBLNo = newMBL.MasterNo
			targetTE = newTE
		default:
			return biz.ErrSeaOrderReassignmentInvalidArgument
		}
		if targetTEID == oldTE.ID {
			return biz.ErrSeaOrderReassignmentTargetConflict
		}
		if targetMBLID == oldMBL.ID && targetTE.ShippingLineID != oldMBL.ShippingLineID {
			return biz.ErrSeaOrderReassignmentTargetConflict
		}
		if targetMBLID != oldMBL.ID && targetTE.ShippingLineID == oldMBL.ShippingLineID {
			return biz.ErrSeaOrderReassignmentTargetConflict
		}
		if _, err := ensureSeaTransportExecutionVersion(
			ctx, tx, organizationID, targetTE, &actorID,
			seatransportexecutionversionent.SourceREASSIGNMENT,
			&input.Reason, nil, nil, input.Confirmation,
		); err != nil {
			return err
		}

		now := time.Now().UTC()
		if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(oldLink.ID).
			SetStatus(seamasterbillorderlinkent.StatusENDED).
			SetEndedAt(now).
			SetVersion(oldLink.Version + 1).
			Save(ctx); err != nil {
			return err
		}

		newLinkBuilder := tx.SeaMasterBillOrderLink.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(organizationID).
			SetMasterBillID(targetMBLID).
			SetTransportExecutionID(targetTEID).
			SetOrderID(order.ID).
			SetDocumentStructure(oldLink.DocumentStructure).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetStartedAt(now).
			SetVersion(1)
		newLink, err := newLinkBuilder.Save(ctx)
		if err != nil {
			return err
		}

		// 锁定并迁移该订单现有的分单 HBL 至目标 MBL (UUID 升序锁定)
		hbls, err := tx.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(order.ID)).
			Order(seahousebillent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		for _, h := range hbls {
			if _, err := tx.SeaHouseBill.UpdateOneID(h.ID).
				SetMasterBillID(targetMBLID).
				SetVersion(h.Version + 1).
				Save(ctx); err != nil {
				return err
			}
		}

		beforeSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"link": map[string]interface{}{
				"id":                     oldLink.ID,
				"version":                oldLink.Version,
				"status":                 oldLink.Status,
				"transport_execution_id": oldLink.TransportExecutionID,
			},
			"master_bill": map[string]interface{}{
				"id":               oldMBL.ID,
				"master_no":        oldMBL.MasterNo,
				"shipping_line_id": oldMBL.ShippingLineID,
				"version":          oldMBL.Version,
				"status":           oldMBL.Status,
			},
			"transport_execution": map[string]interface{}{
				"id":                    oldTE.ID,
				"version":               oldTE.Version,
				"vessel_name":           oldTE.VesselName,
				"voyage_no":             oldTE.VoyageNo,
				"shipping_line_id":      oldTE.ShippingLineID,
				"origin_location_id":    oldTE.OriginLocationID,
				"discharge_location_id": oldTE.DischargeLocationID,
				"transit_location_id":   oldTE.TransitLocationID,
				"etd":                   oldTE.Etd,
				"eta":                   oldTE.Eta,
			},
		}
		afterMBLVersion := uint64(1)
		afterMBLIssuer := uuid.Nil
		afterMBLStatus := "DRAFT"
		if cand, ok := mbls[targetMBLID]; ok && cand != nil {
			afterMBLVersion = cand.Version
			afterMBLIssuer = cand.ShippingLineID
			afterMBLStatus = string(cand.Status)
		} else if input.Target.ShippingLineID != nil {
			afterMBLIssuer = *input.Target.ShippingLineID
		}

		afterSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"link": map[string]interface{}{
				"id":                     newLink.ID,
				"version":                newLink.Version,
				"status":                 newLink.Status,
				"transport_execution_id": newLink.TransportExecutionID,
			},
			"master_bill": map[string]interface{}{
				"id":               targetMBLID,
				"master_no":        targetMBLNo,
				"shipping_line_id": afterMBLIssuer,
				"version":          afterMBLVersion,
				"status":           afterMBLStatus,
			},
			"transport_execution": map[string]interface{}{
				"id":                    targetTE.ID,
				"version":               targetTE.Version,
				"vessel_name":           targetTE.VesselName,
				"voyage_no":             targetTE.VoyageNo,
				"shipping_line_id":      targetTE.ShippingLineID,
				"origin_location_id":    targetTE.OriginLocationID,
				"discharge_location_id": targetTE.DischargeLocationID,
				"transit_location_id":   targetTE.TransitLocationID,
				"etd":                   targetTE.Etd,
				"eta":                   targetTE.Eta,
			},
		}
		beforeSnapshotBytes, err := json.Marshal(beforeSnapshotMap)
		if err != nil {
			return err
		}
		afterSnapshotBytes, err := json.Marshal(afterSnapshotMap)
		if err != nil {
			return err
		}

		if _, err := tx.Order.UpdateOneID(order.ID).SetVersion(order.Version + 1).Save(ctx); err != nil {
			return err
		}

		reassignEvtID := uuid.Must(uuid.NewV7())
		reassignBuilder := tx.SeaOrderReassignmentEvent.Create().
			SetID(reassignEvtID).
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetOrderNo(order.OrderNo).
			SetIdempotencyKey(input.IdempotencyKey).
			SetRequestFingerprint(input.RequestFingerprint).
			SetPreviousMasterBillID(oldMBL.ID).
			SetTargetMasterBillID(targetMBLID).
			SetPreviousTransportExecutionID(oldTE.ID).
			SetTargetTransportExecutionID(targetTE.ID).
			SetPreviousLinkID(oldLink.ID).
			SetTargetLinkID(newLink.ID).
			SetPreviousLinkVersion(oldLink.Version).
			SetTargetLinkVersion(1).
			SetReason(input.Reason).
			SetResponsibilityType(seaorderreassignmenteventent.ResponsibilityType(input.ResponsibilityType)).
			SetBeforeSnapshot(beforeSnapshotBytes).
			SetAfterSnapshot(afterSnapshotBytes).
			SetCreatedBy(actorID).
			SetConfirmedByParty(input.Confirmation.ConfirmedByParty).
			SetConfirmedAt(input.Confirmation.ConfirmedAt).
			SetConfirmationNote(input.Confirmation.ConfirmationNote).
			SetNillableConfirmationAttachmentID(input.Confirmation.ConfirmationAttachmentID)

		if input.ResponsiblePartnerID != nil && *input.ResponsiblePartnerID != uuid.Nil {
			p, err := tx.Partner.Query().
				Where(partnerent.IDEQ(*input.ResponsiblePartnerID), partnerent.OrganizationIDEQ(organizationID)).
				Only(ctx)
			if err != nil {
				return err
			}
			reassignBuilder.SetResponsiblePartnerID(p.ID)
			reassignBuilder.SetResponsiblePartnerName(p.LegalName)
		}

		savedEvent, err := reassignBuilder.Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "sea_order_reassignment_event_idempotency_key", biz.ErrSeaOrderReassignmentIdempotencyConflict)
		}

		if _, err := tx.OrderLifecycleEvent.Create().
			SetOrderID(order.ID).
			SetDimension(orderlifecycleeventent.DimensionFLOW).
			SetToStatus(string(order.FlowStatus)).
			SetAction("REASSIGNED_MBL").
			SetReason(input.Reason).
			SetReferenceType("SEA_ORDER_REASSIGNMENT_EVENT").
			SetReferenceID(reassignEvtID).
			SetOperatorID(actorID).
			Save(ctx); err != nil {
			return err
		}

		reassignmentResult = &biz.SeaOrderReassignmentEvent{
			ID:                   savedEvent.ID,
			CreatedAt:            savedEvent.CreatedAt,
			OrganizationID:       savedEvent.OrganizationID,
			OrderID:              savedEvent.OrderID,
			OrderNo:              savedEvent.OrderNo,
			IdempotencyKey:       savedEvent.IdempotencyKey,
			RequestFingerprint:   savedEvent.RequestFingerprint,
			PreviousMasterBillID: savedEvent.PreviousMasterBillID,
			TargetMasterBillID:   savedEvent.TargetMasterBillID,
			TargetLinkID:         savedEvent.TargetLinkID,
			Reason:               savedEvent.Reason,
			ResponsibilityType:   string(savedEvent.ResponsibilityType),
			Confirmation:         externalConfirmationFromReassignment(savedEvent),
		}

		audit.Details["reassignment_event_id"] = reassignEvtID.String()
		audit.Details["target_master_no"] = targetMBLNo
		audit.Details["reason"] = input.Reason
		return writeAudit(ctx, tx.AuditLog, audit)
	})

	if err != nil {
		return nil, err
	}
	return reassignmentResult, nil
}

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

func decodeSplitResultSnapshotSummary(raw []byte) (int32, decimal.Decimal, decimal.Decimal, error) {
	var snapshot struct {
		SchemaVersion int     `json:"schema_version"`
		PackageCount  *int32  `json:"package_count"`
		GrossWeightKg *string `json:"gross_weight_kg"`
		VolumeCbm     *string `json:"volume_cbm"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return 0, decimal.Zero, decimal.Zero, err
	}
	if snapshot.SchemaVersion != 1 || snapshot.PackageCount == nil || snapshot.GrossWeightKg == nil || snapshot.VolumeCbm == nil {
		return 0, decimal.Zero, decimal.Zero, fmt.Errorf("拆票结果快照缺少必需字段或版本不受支持")
	}
	grossWeight, err := decimal.NewFromString(*snapshot.GrossWeightKg)
	if err != nil {
		return 0, decimal.Zero, decimal.Zero, err
	}
	volume, err := decimal.NewFromString(*snapshot.VolumeCbm)
	if err != nil {
		return 0, decimal.Zero, decimal.Zero, err
	}
	return *snapshot.PackageCount, grossWeight, volume, nil
}

func (r *seaOrderChangeRepo) ListChangeEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*biz.SeaOrderChangeEventSummary, int32, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}

	splitEvents, err := client.SeaOrderSplitEvent.Query().
		Where(
			seaorderspliteventent.OrganizationIDEQ(organizationID),
			seaorderspliteventent.Or(
				seaorderspliteventent.SourceOrderIDEQ(orderID),
				seaorderspliteventent.HasResultsWith(seaordersplitresultent.OrderIDEQ(orderID)),
			),
		).
		WithResults(func(query *ent.SeaOrderSplitResultQuery) {
			query.WithFinalMasterBill()
		}).
		WithCreator().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	reassignEvents, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.OrderIDEQ(orderID),
		).
		WithPreviousMasterBill().
		WithTargetMasterBill().
		WithCreator().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var allEvents []*biz.SeaOrderChangeEventSummary

	for _, se := range splitEvents {
		opName := ""
		if se.Edges.Creator != nil {
			opName = se.Edges.Creator.DisplayName
		}
		resItems := make([]*biz.SeaOrderSplitResultSummaryItem, 0, len(se.Edges.Results))
		for _, res := range se.Edges.Results {
			finalMasterBill, err := res.Edges.FinalMasterBillOrErr()
			if err != nil {
				return nil, 0, err
			}
			packageCount, grossWeight, volume, err := decodeSplitResultSnapshotSummary(res.ResultSnapshot)
			if err != nil {
				return nil, 0, err
			}
			resItems = append(resItems, &biz.SeaOrderSplitResultSummaryItem{
				ResultRole:    string(res.ResultRole),
				OrderID:       res.OrderID,
				OrderNo:       res.OrderNo,
				FinalMasterNo: finalMasterBill.MasterNo,
				PackageCount:  packageCount,
				GrossWeightKg: grossWeight,
				VolumeCbm:     volume,
			})
		}
		summary := &biz.SeaOrderSplitEventSummary{
			SourceOrderID: se.SourceOrderID,
			SourceOrderNo: se.SourceOrderNo,
			ResultCount:   int32(len(se.Edges.Results)),
			Results:       resItems,
		}
		note := ""
		if se.Note != nil {
			note = *se.Note
		}
		allEvents = append(allEvents, &biz.SeaOrderChangeEventSummary{
			ID:           se.ID,
			EventType:    biz.EventTypeSplit,
			CreatedAt:    se.CreatedAt,
			OperatorID:   se.CreatedBy,
			OperatorName: opName,
			NoteOrReason: note,
			SplitSummary: summary,
		})
	}

	for _, re := range reassignEvents {
		opName := ""
		if re.Edges.Creator != nil {
			opName = re.Edges.Creator.DisplayName
		}
		prevNo := ""
		if re.Edges.PreviousMasterBill != nil {
			prevNo = re.Edges.PreviousMasterBill.MasterNo
		}
		targetNo := ""
		if re.Edges.TargetMasterBill != nil {
			targetNo = re.Edges.TargetMasterBill.MasterNo
		}
		respName := ""
		if re.ResponsiblePartnerName != nil {
			respName = *re.ResponsiblePartnerName
		}
		summary := &biz.SeaOrderReassignmentEventSummary{
			OrderID:                re.OrderID,
			OrderNo:                re.OrderNo,
			PreviousMasterNo:       prevNo,
			TargetMasterNo:         targetNo,
			ResponsibilityType:     string(re.ResponsibilityType),
			ResponsiblePartnerName: respName,
			Reason:                 re.Reason,
			Confirmation:           externalConfirmationFromReassignment(re),
		}
		allEvents = append(allEvents, &biz.SeaOrderChangeEventSummary{
			ID:                  re.ID,
			EventType:           biz.EventTypeReassignment,
			CreatedAt:           re.CreatedAt,
			OperatorID:          re.CreatedBy,
			OperatorName:        opName,
			NoteOrReason:        re.Reason,
			ReassignmentSummary: summary,
		})
	}

	sort.Slice(allEvents, func(i, j int) bool {
		if allEvents[i].CreatedAt.Equal(allEvents[j].CreatedAt) {
			return allEvents[i].ID.String() > allEvents[j].ID.String()
		}
		return allEvents[i].CreatedAt.After(allEvents[j].CreatedAt)
	})

	total := int32(len(allEvents))
	start := int((page - 1) * pageSize)
	if start >= len(allEvents) {
		return []*biz.SeaOrderChangeEventSummary{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(allEvents) {
		end = len(allEvents)
	}

	return allEvents[start:end], total, nil
}

func (r *seaOrderChangeRepo) GetChangeEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*biz.SeaOrderChangeEventDetail, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	if eventType == biz.EventTypeSplit {
		se, err := client.SeaOrderSplitEvent.Query().
			Where(
				seaorderspliteventent.IDEQ(eventID),
				seaorderspliteventent.OrganizationIDEQ(organizationID),
				seaorderspliteventent.Or(
					seaorderspliteventent.SourceOrderIDEQ(orderID),
					seaorderspliteventent.HasResultsWith(seaordersplitresultent.OrderIDEQ(orderID)),
				),
			).
			WithResults(func(query *ent.SeaOrderSplitResultQuery) {
				query.WithFinalMasterBill()
			}).
			WithCreator().
			Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaOrderSplitInvalidArgument, nil)
		}
		opName := ""
		if se.Edges.Creator != nil {
			opName = se.Edges.Creator.DisplayName
		}
		resItems := make([]*biz.SeaOrderSplitResultSummaryItem, 0, len(se.Edges.Results))
		for _, res := range se.Edges.Results {
			finalMasterBill, err := res.Edges.FinalMasterBillOrErr()
			if err != nil {
				return nil, err
			}
			packageCount, grossWeight, volume, err := decodeSplitResultSnapshotSummary(res.ResultSnapshot)
			if err != nil {
				return nil, err
			}
			resItems = append(resItems, &biz.SeaOrderSplitResultSummaryItem{
				ResultRole:    string(res.ResultRole),
				OrderID:       res.OrderID,
				OrderNo:       res.OrderNo,
				FinalMasterNo: finalMasterBill.MasterNo,
				PackageCount:  packageCount,
				GrossWeightKg: grossWeight,
				VolumeCbm:     volume,
			})
		}
		note := ""
		if se.Note != nil {
			note = *se.Note
		}
		return &biz.SeaOrderChangeEventDetail{
			ID:                       se.ID,
			EventType:                biz.EventTypeSplit,
			CreatedAt:                se.CreatedAt,
			OperatorID:               se.CreatedBy,
			OperatorName:             opName,
			NoteOrReason:             note,
			BeforeSnapshotJSON:       string(se.BeforeSnapshot),
			ConservationSnapshotJSON: string(se.ConservationSnapshot),
			SplitSummary: &biz.SeaOrderSplitEventSummary{
				SourceOrderID: se.SourceOrderID,
				SourceOrderNo: se.SourceOrderNo,
				ResultCount:   int32(len(se.Edges.Results)),
				Results:       resItems,
			},
		}, nil
	}

	re, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.IDEQ(eventID),
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.OrderIDEQ(orderID),
		).
		WithPreviousMasterBill().
		WithTargetMasterBill().
		WithCreator().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaOrderReassignmentInvalidArgument, nil)
	}

	opName := ""
	if re.Edges.Creator != nil {
		opName = re.Edges.Creator.DisplayName
	}
	prevNo := ""
	if re.Edges.PreviousMasterBill != nil {
		prevNo = re.Edges.PreviousMasterBill.MasterNo
	}
	targetNo := ""
	if re.Edges.TargetMasterBill != nil {
		targetNo = re.Edges.TargetMasterBill.MasterNo
	}
	respName := ""
	if re.ResponsiblePartnerName != nil {
		respName = *re.ResponsiblePartnerName
	}

	return &biz.SeaOrderChangeEventDetail{
		ID:                 re.ID,
		EventType:          biz.EventTypeReassignment,
		CreatedAt:          re.CreatedAt,
		OperatorID:         re.CreatedBy,
		OperatorName:       opName,
		NoteOrReason:       re.Reason,
		BeforeSnapshotJSON: string(re.BeforeSnapshot),
		AfterSnapshotJSON:  string(re.AfterSnapshot),
		ReassignmentSummary: &biz.SeaOrderReassignmentEventSummary{
			OrderID:                re.OrderID,
			OrderNo:                re.OrderNo,
			PreviousMasterNo:       prevNo,
			TargetMasterNo:         targetNo,
			ResponsibilityType:     string(re.ResponsibilityType),
			ResponsiblePartnerName: respName,
			Reason:                 re.Reason,
			Confirmation:           externalConfirmationFromReassignment(re),
		},
	}, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
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
		shippinglineent.OrganizationIDEQ(organizationID),
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
		shippinglineent.OrganizationIDEQ(organizationID),
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

func (r *seaOrderChangeRepo) GetSplitEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.SeaOrderSplitEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	existingEvent, err := client.SeaOrderSplitEvent.Query().
		Where(
			seaorderspliteventent.OrganizationIDEQ(organizationID),
			seaorderspliteventent.IdempotencyKeyEQ(idempotencyKey),
		).
		WithResults().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	res := &biz.SeaOrderSplitEvent{
		ID:                  existingEvent.ID,
		CreatedAt:           existingEvent.CreatedAt,
		OrganizationID:      existingEvent.OrganizationID,
		SourceOrderID:       existingEvent.SourceOrderID,
		SourceOrderNo:       existingEvent.SourceOrderNo,
		IdempotencyKey:      existingEvent.IdempotencyKey,
		RequestFingerprint:  existingEvent.RequestFingerprint,
		Note:                existingEvent.Note,
		SourceOrderVersion:  existingEvent.SourceOrderVersion,
		SourceLinkID:        existingEvent.SourceLinkID,
		SourceLinkVersion:   existingEvent.SourceLinkVersion,
		SourceAllocationVer: existingEvent.SourceAllocationVersion,
		BeforeSnapshot:      existingEvent.BeforeSnapshot,
		CreatedBy:           existingEvent.CreatedBy,
	}
	for _, resItem := range existingEvent.Edges.Results {
		res.Results = append(res.Results, &biz.SeaOrderSplitResult{
			ID:                  resItem.ID,
			CreatedAt:           resItem.CreatedAt,
			SplitEventID:        resItem.SplitEventID,
			OrganizationID:      resItem.OrganizationID,
			OrderID:             resItem.OrderID,
			OrderNo:             resItem.OrderNo,
			ResultRole:          string(resItem.ResultRole),
			Sequence:            resItem.Sequence,
			ClientResultKey:     resItem.ClientResultKey,
			InitialMasterBillID: resItem.InitialMasterBillID,
			FinalMasterBillID:   resItem.FinalMasterBillID,
			ResultSnapshot:      resItem.ResultSnapshot,
		})
	}
	return res, nil
}

func (r *seaOrderChangeRepo) GetSplitEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*biz.SeaOrderSplitEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	ev, err := client.SeaOrderSplitEvent.Query().
		Where(
			seaorderspliteventent.IDEQ(eventID),
			seaorderspliteventent.OrganizationIDEQ(organizationID),
			seaorderspliteventent.Or(
				seaorderspliteventent.SourceOrderIDEQ(orderID),
				seaorderspliteventent.HasResultsWith(seaordersplitresultent.OrderIDEQ(orderID)),
			),
		).
		WithResults().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSeaOrderSplitInvalidArgument
		}
		return nil, err
	}
	bizResults := make([]*biz.SeaOrderSplitResult, 0, len(ev.Edges.Results))
	for _, res := range ev.Edges.Results {
		bizResults = append(bizResults, &biz.SeaOrderSplitResult{
			ID:                  res.ID,
			CreatedAt:           res.CreatedAt,
			SplitEventID:        res.SplitEventID,
			OrganizationID:      res.OrganizationID,
			OrderID:             res.OrderID,
			OrderNo:             res.OrderNo,
			ResultRole:          string(res.ResultRole),
			Sequence:            res.Sequence,
			ClientResultKey:     res.ClientResultKey,
			InitialMasterBillID: res.InitialMasterBillID,
			FinalMasterBillID:   res.FinalMasterBillID,
			ResultSnapshot:      res.ResultSnapshot,
		})
	}
	reasEvents, err := client.SeaOrderReassignmentEvent.Query().
		Where(seaorderreassignmenteventent.SplitEventIDEQ(ev.ID)).
		Select(seaorderreassignmenteventent.FieldID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	reasIDs := make([]uuid.UUID, 0, len(reasEvents))
	for _, re := range reasEvents {
		reasIDs = append(reasIDs, re.ID)
	}
	return &biz.SeaOrderSplitEvent{
		ID:                   ev.ID,
		CreatedAt:            ev.CreatedAt,
		OrganizationID:       ev.OrganizationID,
		SourceOrderID:        ev.SourceOrderID,
		SourceOrderNo:        ev.SourceOrderNo,
		IdempotencyKey:       ev.IdempotencyKey,
		RequestFingerprint:   ev.RequestFingerprint,
		Note:                 ev.Note,
		SourceOrderVersion:   ev.SourceOrderVersion,
		SourceLinkID:         ev.SourceLinkID,
		SourceLinkVersion:    ev.SourceLinkVersion,
		SourceAllocationVer:  ev.SourceAllocationVersion,
		BeforeSnapshot:       ev.BeforeSnapshot,
		ConservationSnapshot: ev.ConservationSnapshot,
		CreatedBy:            ev.CreatedBy,
		Results:              bizResults,
		ReassignmentEventIDs: reasIDs,
	}, nil
}

func (r *seaOrderChangeRepo) GetReassignmentEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.SeaOrderReassignmentEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	existingEvent, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.IdempotencyKeyEQ(idempotencyKey),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &biz.SeaOrderReassignmentEvent{
		ID:                           existingEvent.ID,
		CreatedAt:                    existingEvent.CreatedAt,
		OrganizationID:               existingEvent.OrganizationID,
		OrderID:                      existingEvent.OrderID,
		OrderNo:                      existingEvent.OrderNo,
		SplitEventID:                 existingEvent.SplitEventID,
		SplitResultID:                existingEvent.SplitResultID,
		IdempotencyKey:               existingEvent.IdempotencyKey,
		RequestFingerprint:           existingEvent.RequestFingerprint,
		PreviousMasterBillID:         existingEvent.PreviousMasterBillID,
		TargetMasterBillID:           existingEvent.TargetMasterBillID,
		PreviousTransportExecutionID: existingEvent.PreviousTransportExecutionID,
		TargetTransportExecutionID:   existingEvent.TargetTransportExecutionID,
		PreviousLinkID:               existingEvent.PreviousLinkID,
		TargetLinkID:                 existingEvent.TargetLinkID,
		PreviousLinkVersion:          existingEvent.PreviousLinkVersion,
		TargetLinkVersion:            existingEvent.TargetLinkVersion,
		Reason:                       existingEvent.Reason,
		ResponsibilityType:           string(existingEvent.ResponsibilityType),
		ResponsiblePartnerID:         existingEvent.ResponsiblePartnerID,
		BeforeSnapshot:               existingEvent.BeforeSnapshot,
		AfterSnapshot:                existingEvent.AfterSnapshot,
		CreatedBy:                    existingEvent.CreatedBy,
		Confirmation:                 externalConfirmationFromReassignment(existingEvent),
	}, nil
}

func (r *seaOrderChangeRepo) GetReassignmentEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*biz.SeaOrderReassignmentEvent, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	existingEvent, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.IDEQ(eventID),
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.OrderIDEQ(orderID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSeaOrderReassignmentInvalidArgument
		}
		return nil, err
	}
	return &biz.SeaOrderReassignmentEvent{
		ID:                           existingEvent.ID,
		CreatedAt:                    existingEvent.CreatedAt,
		OrganizationID:               existingEvent.OrganizationID,
		OrderID:                      existingEvent.OrderID,
		OrderNo:                      existingEvent.OrderNo,
		SplitEventID:                 existingEvent.SplitEventID,
		SplitResultID:                existingEvent.SplitResultID,
		IdempotencyKey:               existingEvent.IdempotencyKey,
		RequestFingerprint:           existingEvent.RequestFingerprint,
		PreviousMasterBillID:         existingEvent.PreviousMasterBillID,
		TargetMasterBillID:           existingEvent.TargetMasterBillID,
		PreviousTransportExecutionID: existingEvent.PreviousTransportExecutionID,
		TargetTransportExecutionID:   existingEvent.TargetTransportExecutionID,
		PreviousLinkID:               existingEvent.PreviousLinkID,
		TargetLinkID:                 existingEvent.TargetLinkID,
		PreviousLinkVersion:          existingEvent.PreviousLinkVersion,
		TargetLinkVersion:            existingEvent.TargetLinkVersion,
		Reason:                       existingEvent.Reason,
		ResponsibilityType:           string(existingEvent.ResponsibilityType),
		ResponsiblePartnerID:         existingEvent.ResponsiblePartnerID,
		ResponsiblePartnerName:       existingEvent.ResponsiblePartnerName,
		Confirmation:                 externalConfirmationFromReassignment(existingEvent),
		BeforeSnapshot:               existingEvent.BeforeSnapshot,
		AfterSnapshot:                existingEvent.AfterSnapshot,
		CreatedBy:                    existingEvent.CreatedBy,
	}, nil
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

var _ biz.SeaOrderChangeRepo = (*seaOrderChangeRepo)(nil)
