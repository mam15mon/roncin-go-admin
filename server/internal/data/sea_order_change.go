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
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seacargoallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seacargoallocation"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seaordersplitresultent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitresult"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
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
	// HBL 是否全为草稿
	nonDraftHblCount, err := client.SeaHouseBill.Query().
		Where(seahousebillent.OrderIDEQ(orderID), seahousebillent.StatusNEQ(seahousebillent.StatusDRAFT)).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if nonDraftHblCount > 0 {
		actions.CanSplit = false
		actions.CanReassign = false
		msg := "存在已确认或不可变的分单(HBL)，不允许拆票或改配"
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

	// 5. 拆票专属门禁
	if activeLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
		actions.CanSplit = false
		actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "DIRECT 当前没有 HBL 箱货分配，暂不支持部分拆票，可执行整票改配")
	} else {
		if activeLink.CargoAllocationStatus != seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			actions.CanSplit = false
			actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "箱货分配未确认，请先完成箱货分配并确认后再拆票")
		}
		hblCount, err := client.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(orderID)).Count(ctx)
		if err != nil {
			return nil, err
		}
		if hblCount < 2 {
			actions.CanSplit = false
			actions.SplitBlockedReasons = append(actions.SplitBlockedReasons, "订单分单数量不足（至少需 2 个 HBL 才能拆出新票）")
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
		WithMasterBill(func(q *ent.SeaMasterBillQuery) {
			q.WithTransportExecution()
		}).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	mbl := activeLink.Edges.MasterBill
	if mbl == nil {
		return nil, biz.ErrSeaMasterBillNotFound
	}
	mblSummary, err := mblToSummary(ctx, client, organizationID, mbl)
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

	// 查询 Allocations
	allocs, err := client.SeaCargoAllocation.Query().
		Where(seacargoallocationent.OrderIDEQ(orderID)).
		Order(seacargoallocationent.ByCreatedAt(), seacargoallocationent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	allocList := make([]*biz.SeaOrderSplitAllocationItem, 0, len(allocs))
	for _, a := range allocs {
		gw, err := decimal.NewFromString(a.GrossWeightKg)
		if err != nil {
			return nil, err
		}
		vol, err := decimal.NewFromString(a.VolumeCbm)
		if err != nil {
			return nil, err
		}
		allocList = append(allocList, &biz.SeaOrderSplitAllocationItem{
			ID:            a.ID,
			CargoItemID:   a.CargoItemID,
			HouseBillID:   a.HouseBillID,
			ContainerID:   a.ContainerID,
			PackageCount:  int32(a.PackageCount),
			GrossWeightKg: gw,
			VolumeCbm:     vol,
		})
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
		CargoAllocationStatus:          string(activeLink.CargoAllocationStatus),
		CargoAllocationVersion:         activeLink.CargoAllocationVersion,
		HouseBills:                     hblItems,
		CargoItems:                     cargoItemList,
		Containers:                     containerList,
		Allocations:                    allocList,
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
				WithTransportExecution().
				Only(ctx)
			if queryErr != nil {
				if ent.IsNotFound(queryErr) {
					return nil, biz.ErrSeaOrderSplitVersionConflict
				}
				return nil, queryErr
			}
			candidateTE := candidate.Edges.TransportExecution
			if candidateTE == nil ||
				candidate.Status != seamasterbillent.StatusDRAFT ||
				candidate.Version != *target.CandidateVersion ||
				candidate.TransportExecutionID != *target.CandidateTEID ||
				candidateTE.Version != *target.CandidateTEVersion {
				return nil, biz.ErrSeaOrderSplitVersionConflict
			}
			if candidate.IssuerPartnerID != *target.IssuerPartnerID {
				return nil, biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
		} else if target.TargetType == biz.SplitTargetTypeNew {
			if _, err := validateNewMasterBillInput(ctx, client, organizationID, target); err != nil {
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
	preview.Baseline = biz.SeaOrderSplitQuantitySummary{
		PackageCount:   baselinePkg,
		GrossWeightKg:  baselineWeight,
		VolumeCbm:      baselineVol,
		ContainerCount: int32(len(splitCtx.Containers)),
		HouseBillCount: int32(len(splitCtx.HouseBills)),
		FeeCount:       int32(len(splitCtx.DraftFees)),
	}

	// 3. 校验 HBL 分配：每个 HBL 必须恰好分配到 1 个结果票
	hblMap := make(map[uuid.UUID]*biz.SeaOrderSplitHouseBillItem)
	for _, h := range splitCtx.HouseBills {
		hblMap[h.ID] = h
	}
	hblToResultKey := make(map[uuid.UUID]string)
	for _, res := range input.Results {
		for _, hblID := range res.HouseBillIDs {
			if _, exists := hblMap[hblID]; !exists {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "HOUSE_BILL_NOT_FOUND",
					Message:         fmt.Sprintf("分单 %s 不属于当前订单", hblID),
					ClientResultKey: res.ClientResultKey,
					HouseBillID:     hblID.String(),
				})
				continue
			}
			if prevKey, exists := hblToResultKey[hblID]; exists {
				houseNo := hblMap[hblID].HouseNo
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "HOUSE_BILL_CROSSES_RESULTS",
					Message:         fmt.Sprintf("分单 %s（%s）被同时分配到 %s 和 %s", houseNo, hblID, prevKey, res.ClientResultKey),
					ClientResultKey: res.ClientResultKey,
					HouseBillID:     hblID.String(),
				})
			}
			hblToResultKey[hblID] = res.ClientResultKey
		}
	}
	for hblID := range hblMap {
		if _, exists := hblToResultKey[hblID]; !exists {
			houseNo := hblMap[hblID].HouseNo
			preview.IsValid = false
			preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
				Reason:      "HOUSE_BILL_UNASSIGNED",
				Message:     fmt.Sprintf("分单 %s（%s）未被分配到任何结果票", houseNo, hblID),
				HouseBillID: hblID.String(),
			})
		}
	}

	// 4. 校验集装箱跨票规则：同一实际箱的所有货物分配必须全部归属于同一个结果票！
	containerToHbls := make(map[uuid.UUID]map[uuid.UUID]struct{})
	for _, a := range splitCtx.Allocations {
		if a.ContainerID != nil && *a.ContainerID != uuid.Nil {
			if _, ok := containerToHbls[*a.ContainerID]; !ok {
				containerToHbls[*a.ContainerID] = make(map[uuid.UUID]struct{})
			}
			containerToHbls[*a.ContainerID][a.HouseBillID] = struct{}{}
		}
	}

	containerToResultKey := make(map[uuid.UUID]string)
	containerMap := make(map[uuid.UUID]*biz.SeaOrderSplitContainerItem)
	for _, c := range splitCtx.Containers {
		containerMap[c.ID] = c
	}

	for cID, hbls := range containerToHbls {
		var targetResultKey string
		for hID := range hbls {
			rKey := hblToResultKey[hID]
			if targetResultKey == "" {
				targetResultKey = rKey
			} else if targetResultKey != rKey {
				preview.IsValid = false
				preview.ConservationPassed = false
				cNo := ""
				if c, ok := containerMap[cID]; ok {
					cNo = c.ContainerNo
				}
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:      "CONTAINER_CROSSES_RESULTS",
					Message:     fmt.Sprintf("集装箱 %s(%s) 内货物跨结果票分配（%s vs %s），违反集装箱不可跨票规则", cNo, cID, targetResultKey, rKey),
					ContainerID: cID.String(),
				})
			}
		}
		containerToResultKey[cID] = targetResultKey
	}

	// 5. 校验草稿费用：每笔费用必须恰好分配到 1 个结果票
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

	// 6. 汇总每个结果票的件重尺与箱货
	totalAllocatedPkg := int32(0)
	totalAllocatedWeight := decimal.Zero
	totalAllocatedVol := decimal.Zero

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

		hblSet := make(map[uuid.UUID]struct{}, len(res.HouseBillIDs))
		for _, hID := range res.HouseBillIDs {
			hblSet[hID] = struct{}{}
		}

		resPkg := int32(0)
		resWeight := decimal.Zero
		resVol := decimal.Zero

		for _, a := range splitCtx.Allocations {
			if _, ok := hblSet[a.HouseBillID]; ok {
				resPkg += a.PackageCount
				resWeight = resWeight.Add(a.GrossWeightKg)
				resVol = resVol.Add(a.VolumeCbm)
			}
		}

		totalAllocatedPkg += resPkg
		totalAllocatedWeight = totalAllocatedWeight.Add(resWeight)
		totalAllocatedVol = totalAllocatedVol.Add(resVol)

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

		resultPreviews = append(resultPreviews, &biz.SeaOrderSplitPreviewResultItem{
			ClientResultKey:     res.ClientResultKey,
			ResultRole:          res.ResultRole,
			ClientTargetKey:     res.ClientTargetKey,
			PackageCount:        resPkg,
			GrossWeightKg:       resWeight,
			VolumeCbm:           resVol,
			ContainerCount:      resContainersCount,
			HouseBillCount:      int32(len(res.HouseBillIDs)),
			FeeCount:            int32(len(res.DraftFeeIDs)),
			AttachmentCount:     int32(len(res.AttachmentReferenceIDs)),
			ContainerPlans:      containerPlans,
			InternalReferenceNo: res.InternalReferenceNo,
			BookingNotes:        bookingNotes,
			AllocationNotes:     allocationNotes,
			OperationNotes:      operationNotes,
		})
	}

	preview.Allocated = biz.SeaOrderSplitQuantitySummary{
		PackageCount:   totalAllocatedPkg,
		GrossWeightKg:  totalAllocatedWeight,
		VolumeCbm:      totalAllocatedVol,
		ContainerCount: preview.Baseline.ContainerCount,
		HouseBillCount: preview.Baseline.HouseBillCount,
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

	if remainingPkg != 0 || !remainingWeight.IsZero() || !remainingVol.IsZero() {
		preview.ConservationPassed = false
		preview.IsValid = false
		preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
			Reason:         "QUANTITY_CONSERVATION_FAILED",
			Message:        "拆票结果与原始订单总量不守恒",
			BaselineValue:  fmt.Sprintf("pkg:%d, wt:%s, vol:%s", baselinePkg, baselineWeight.String(), baselineVol.String()),
			AllocatedValue: fmt.Sprintf("pkg:%d, wt:%s, vol:%s", totalAllocatedPkg, totalAllocatedWeight.String(), totalAllocatedVol.String()),
			DiffValue:      fmt.Sprintf("pkg:%d, wt:%s, vol:%s", remainingPkg, remainingWeight.String(), remainingVol.String()),
		})
	}

	for _, rp := range resultPreviews {
		if rp.ResultRole == biz.ResultRoleOriginal {
			if rp.HouseBillCount == 0 || rp.PackageCount == 0 {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "ORIGINAL_ORDER_EMPTY",
					Message:         "原票必须保留至少 1 个分单及非零货物，不可将全部货物拆出",
					ClientResultKey: rp.ClientResultKey,
				})
			}
		} else {
			if rp.HouseBillCount == 0 || rp.PackageCount == 0 {
				preview.IsValid = false
				preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
					Reason:          "CREATED_ORDER_EMPTY",
					Message:         "新票必须分配至少 1 个分单及非零货物",
					ClientResultKey: rp.ClientResultKey,
				})
			}
		}
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

		if input.ExpectedVersions == nil || input.ExpectedVersions.OrderVersion == 0 || sourceOrder.Version != input.ExpectedVersions.OrderVersion {
			return biz.ErrSeaOrderSplitVersionConflict
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
			Select(seamasterbillorderlinkent.FieldID, seamasterbillorderlinkent.FieldMasterBillID).
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
		teIDsToLock := make([]uuid.UUID, 0, len(mblIDsToLock))
		for _, mblID := range mblIDsToLock {
			mbl, err := tx.SeaMasterBill.Query().
				Where(seamasterbillent.IDEQ(mblID), seamasterbillent.OrganizationIDEQ(organizationID)).
				ForUpdate().
				Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
			}
			lockedMBLs[mblID] = mbl
			teIDsToLock = append(teIDsToLock, mbl.TransportExecutionID)
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
		if input.ExpectedVersions.AllocationVersion == 0 || lockedActiveLink.CargoAllocationVersion != input.ExpectedVersions.AllocationVersion {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		if lockedActiveLink.DocumentStructure != seamasterbillorderlinkent.DocumentStructureHOUSE {
			return biz.ErrSeaOrderSplitBlocked
		}
		if lockedActiveLink.CargoAllocationStatus != seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
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
					t.CandidateTEID == nil || *t.CandidateTEID == uuid.Nil ||
					candMBL.TransportExecutionID != *t.CandidateTEID {
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

				candTE := lockedTEs[candMBL.TransportExecutionID]
				if candTE == nil {
					return biz.ErrSeaTransportExecutionNotFound
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
				if t.IssuerPartnerID != nil && *t.IssuerPartnerID != uuid.Nil && candMBL.IssuerPartnerID != *t.IssuerPartnerID {
					return biz.MetadataError(biz.ErrSeaOrderSplitBlocked, map[string]string{
						"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
					})
				}
				if t.CarrierID != nil && *t.CarrierID != uuid.Nil && (candTE.CarrierID == nil || *candTE.CarrierID != *t.CarrierID) {
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
		for _, ci := range cargoItems {
			expV, ok := input.ExpectedVersions.CargoItemVersions[ci.ID]
			if !ok || expV == 0 || ci.Version != expV {
				return biz.ErrSeaOrderSplitVersionConflict
			}
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
			if h.Status != seahousebillent.StatusDRAFT {
				return biz.ErrSeaOrderSplitBlocked
			}
			lockedHBLMap[h.ID] = h
		}
		if input.ExpectedVersions.HouseBillVersions == nil || len(input.ExpectedVersions.HouseBillVersions) != len(hbls) {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		for _, h := range hbls {
			expV, ok := input.ExpectedVersions.HouseBillVersions[h.ID]
			if !ok || expV == 0 || h.Version != expV {
				return biz.ErrSeaOrderSplitVersionConflict
			}
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
		if input.ExpectedVersions.ContainerVersions == nil || len(input.ExpectedVersions.ContainerVersions) != len(containers) {
			return biz.ErrSeaOrderSplitVersionConflict
		}
		for _, c := range containers {
			expV, ok := input.ExpectedVersions.ContainerVersions[c.ID]
			if !ok || expV == 0 || c.Version != expV {
				return biz.ErrSeaOrderSplitVersionConflict
			}
		}

		allocations, err := tx.SeaCargoAllocation.Query().
			Where(seacargoallocationent.OrderIDEQ(sourceOrder.ID)).
			Order(seacargoallocationent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
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

		// 1. HBL 分配完整性与唯一性
		assignedHBLs := make(map[uuid.UUID]string)
		for _, res := range input.Results {
			for _, hID := range res.HouseBillIDs {
				if _, exists := lockedHBLMap[hID]; !exists {
					return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
						"reason":        "HOUSE_BILL_NOT_FOUND",
						"house_bill_id": hID.String(),
					})
				}
				if prevKey, seen := assignedHBLs[hID]; seen {
					return biz.MetadataError(biz.ErrSeaOrderSplitEntityCrossesResults, map[string]string{
						"reason":        "HBL_CROSSES_RESULTS",
						"house_bill_id": hID.String(),
						"first_result":  prevKey,
						"second_result": res.ClientResultKey,
					})
				}
				assignedHBLs[hID] = res.ClientResultKey
			}
		}
		if len(assignedHBLs) != len(hbls) {
			return biz.MetadataError(biz.ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason":  "HOUSE_BILL_ALLOCATION_INCOMPLETE",
				"message": "所有分单必须分配且只能属于一个结果",
			})
		}

		// 2. 真实箱不可跨结果校验
		containerToResultKey := make(map[uuid.UUID]string)
		for _, a := range allocations {
			if a.ContainerID != nil && *a.ContainerID != uuid.Nil {
				cID := *a.ContainerID
				resKey := assignedHBLs[a.HouseBillID]
				if existingResKey, seen := containerToResultKey[cID]; seen && existingResKey != resKey {
					return biz.MetadataError(biz.ErrSeaOrderSplitEntityCrossesResults, map[string]string{
						"reason":        "CONTAINER_CROSSES_RESULTS",
						"container_id":  cID.String(),
						"first_result":  existingResKey,
						"second_result": resKey,
					})
				}
				containerToResultKey[cID] = resKey
			}
		}

		// 3. 数量守恒重算
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

		for _, ci := range cargoItems {
			ciAllocPkg := 0
			ciAllocWeight := decimal.Zero
			ciAllocVol := decimal.Zero

			for _, a := range allocations {
				if a.CargoItemID == ci.ID {
					gw, parseErr := decimal.NewFromString(a.GrossWeightKg)
					if parseErr != nil {
						return parseErr
					}
					v, parseErr := decimal.NewFromString(a.VolumeCbm)
					if parseErr != nil {
						return parseErr
					}

					resKey := assignedHBLs[a.HouseBillID]
					rs := resCargoStats[resKey]
					rs.pkg += a.PackageCount
					rs.weight = rs.weight.Add(gw)
					rs.vol = rs.vol.Add(v)

					ciAllocPkg += a.PackageCount
					ciAllocWeight = ciAllocWeight.Add(gw)
					ciAllocVol = ciAllocVol.Add(v)
				}
			}

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

		beforeAllocData := make([]map[string]interface{}, 0, len(allocations))
		for _, a := range allocations {
			beforeAllocData = append(beforeAllocData, map[string]interface{}{
				"id":            a.ID,
				"cargo_item_id": a.CargoItemID,
				"house_bill_id": a.HouseBillID,
				"container_id":  a.ContainerID,
				"package_count": a.PackageCount,
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
				"cargo_allocation_version": lockedActiveLink.CargoAllocationVersion,
				"status":                   lockedActiveLink.Status,
				"master_bill_id":           lockedActiveLink.MasterBillID,
			},
			"master_bill": map[string]interface{}{
				"id":        lockedMBLs[lockedActiveLink.MasterBillID].ID,
				"master_no": lockedMBLs[lockedActiveLink.MasterBillID].MasterNo,
				"version":   lockedMBLs[lockedActiveLink.MasterBillID].Version,
				"status":    lockedMBLs[lockedActiveLink.MasterBillID].Status,
			},
			"house_bills":     beforeHblData,
			"cargo_items":     beforeCargoData,
			"containers":      beforeContainerData,
			"allocations":     beforeAllocData,
			"draft_fees":      beforeFeeData,
			"attachments":     beforeAttData,
			"container_plans": beforePlanData,
		}
		beforeSnapshotBytes, err := json.Marshal(beforeSnapshotMap)
		if err != nil {
			return err
		}

		conservationSnapshotMap := map[string]interface{}{
			"schema_version":      1,
			"conservation_passed": true,
			"cargo_items_count":   len(cargoItems),
			"house_bills_count":   len(hbls),
			"containers_count":    len(containers),
			"draft_fees_count":    draftFeeCount,
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
			SetSourceAllocationVersion(lockedActiveLink.CargoAllocationVersion).
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
					candTE := lockedTEs[candMBL.TransportExecutionID]
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
						SetOrderID(sourceOrder.ID).
						SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
						SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
						SetCargoAllocationVersion(1).
						SetStatus(seamasterbillorderlinkent.StatusACTIVE).
						SetStartedAt(now).
						SetVersion(1)
					if lockedActiveLink.CargoAllocationConfirmedAt != nil {
						finalLinkBuilder.SetCargoAllocationConfirmedAt(*lockedActiveLink.CargoAllocationConfirmedAt)
					}
					if lockedActiveLink.CargoAllocationConfirmedBy != nil {
						finalLinkBuilder.SetCargoAllocationConfirmedBy(*lockedActiveLink.CargoAllocationConfirmedBy)
					}
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
						SetPreviousTransportExecutionID(lockedMBLs[lockedActiveLink.MasterBillID].TransportExecutionID).
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
					if targetTE.CarrierID != nil {
						orderUpdate.SetCarrierID(*targetTE.CarrierID)
					} else {
						orderUpdate.ClearCarrierID()
					}
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
					// 留在当前母单，分配版本递增
					if _, err := tx.SeaMasterBillOrderLink.UpdateOneID(lockedActiveLink.ID).
						SetCargoAllocationVersion(lockedActiveLink.CargoAllocationVersion + 1).
						SetCargoAllocationConfirmedAt(now).
						SetCargoAllocationConfirmedBy(actorID).
						Save(ctx); err != nil {
						return err
					}
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
					SetTradeTerm(sourceOrder.TradeTerm).
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
					createOrder.SetNillableCarrierID(targetTE.CarrierID)
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
					createOrder.SetNillableCarrierID(sourceOrder.CarrierID)
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
					SetOrderID(newOrder.ID).
					SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
					SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
					SetCargoAllocationVersion(1).
					SetStatus(seamasterbillorderlinkent.StatusACTIVE).
					SetStartedAt(now).
					SetVersion(1)
				if lockedActiveLink.CargoAllocationConfirmedAt != nil {
					initialChildLinkBuilder.SetCargoAllocationConfirmedAt(*lockedActiveLink.CargoAllocationConfirmedAt)
				}
				if lockedActiveLink.CargoAllocationConfirmedBy != nil {
					initialChildLinkBuilder.SetCargoAllocationConfirmedBy(*lockedActiveLink.CargoAllocationConfirmedBy)
				}
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
						SetOrderID(newOrder.ID).
						SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
						SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
						SetCargoAllocationVersion(1).
						SetStatus(seamasterbillorderlinkent.StatusACTIVE).
						SetStartedAt(now).
						SetVersion(1)
					if lockedActiveLink.CargoAllocationConfirmedAt != nil {
						finalChildLinkBuilder.SetCargoAllocationConfirmedAt(*lockedActiveLink.CargoAllocationConfirmedAt)
					}
					if lockedActiveLink.CargoAllocationConfirmedBy != nil {
						finalChildLinkBuilder.SetCargoAllocationConfirmedBy(*lockedActiveLink.CargoAllocationConfirmedBy)
					}
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
						SetPreviousTransportExecutionID(lockedMBLs[lockedActiveLink.MasterBillID].TransportExecutionID).
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
						Save(ctx)
					if err != nil {
						return err
					}
					reassignmentEventIDs = append(reassignmentEventIDs, reassignEvt.ID)

					// 更新新订单权威航程投影
					targetTE := lockedTEs[finalTEID]
					orderUpdate := tx.Order.UpdateOneID(newOrder.ID).
						SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo))
					if targetTE.CarrierID != nil {
						orderUpdate.SetCarrierID(*targetTE.CarrierID)
					} else {
						orderUpdate.ClearCarrierID()
					}
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
		// 迁移 HBL：真正迁移 master_bill_id 与 order_id
		// -------------------------------------------------------------------
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			finalMblID := resultFinalMblMap[res.ClientResultKey]
			for _, hblID := range res.HouseBillIDs {
				hbl := lockedHBLMap[hblID]
				if _, err := tx.SeaHouseBill.UpdateOneID(hblID).
					SetOrderID(targetOrderID).
					SetMasterBillID(finalMblID).
					SetVersion(hbl.Version + 1).
					Save(ctx); err != nil {
					return err
				}
			}
		}

		// -------------------------------------------------------------------
		// 迁移集装箱 OrderContainer
		// -------------------------------------------------------------------
		for cID, rKey := range containerToResultKey {
			targetOrderID := resultOrderMap[rKey]
			lockedContainer := lockedContainerMap[cID]
			curVer := uint64(1)
			if lockedContainer != nil {
				curVer = lockedContainer.Version + 1
			}
			if _, err := tx.OrderContainer.UpdateOneID(cID).
				SetOrderID(targetOrderID).
				SetVersion(curVer).
				Save(ctx); err != nil {
				return err
			}
		}

		// -------------------------------------------------------------------
		// 货物重构与分配 Link 迁移
		// -------------------------------------------------------------------
		resultCargoOldNewMap := make(map[string]map[string]string)
		resultCargoRetainedIDs := make(map[string][]string)
		resultAllocOldNewMap := make(map[string]map[string]string)
		for _, res := range input.Results {
			resultCargoOldNewMap[res.ClientResultKey] = make(map[string]string)
			resultCargoRetainedIDs[res.ClientResultKey] = make([]string, 0)
			resultAllocOldNewMap[res.ClientResultKey] = make(map[string]string)
		}

		var zeroRemainingCargoIDs []uuid.UUID

		// A.3: 处理货物 OrderCargoItem
		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]

			if res.ResultRole == biz.ResultRoleCreated {
				itemAllocMap := make(map[uuid.UUID][]*ent.SeaCargoAllocation)
				for _, hID := range res.HouseBillIDs {
					for _, a := range allocations {
						if a.HouseBillID == hID {
							itemAllocMap[a.CargoItemID] = append(itemAllocMap[a.CargoItemID], a)
						}
					}
				}

				for origCargoItemID, allocList := range itemAllocMap {
					var origCargoItem *ent.OrderCargoItem
					for _, ci := range cargoItems {
						if ci.ID == origCargoItemID {
							origCargoItem = ci
							break
						}
					}
					if origCargoItem == nil {
						continue
					}

					newPkg := 0
					newWeight := decimal.Zero
					newVol := decimal.Zero
					for _, a := range allocList {
						gw, parseErr := decimal.NewFromString(a.GrossWeightKg)
						if parseErr != nil {
							return parseErr
						}
						vol, parseErr := decimal.NewFromString(a.VolumeCbm)
						if parseErr != nil {
							return parseErr
						}
						newPkg += a.PackageCount
						newWeight = newWeight.Add(gw)
						newVol = newVol.Add(vol)
					}

					fWeight, _ := newWeight.Float64()
					fVol, _ := newVol.Float64()

					createdItem, err := tx.OrderCargoItem.Create().
						SetID(uuid.Must(uuid.NewV7())).
						SetOrganizationID(organizationID).
						SetOrderID(targetOrderID).
						SetCargoName(origCargoItem.CargoName).
						SetPackageCount(newPkg).
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
				// 原票：有剩余则更新，零剩余必须删除!
				for _, ci := range cargoItems {
					origAllocPkg := 0
					origAllocWeight := decimal.Zero
					origAllocVol := decimal.Zero

					for _, hID := range res.HouseBillIDs {
						for _, a := range allocations {
							if a.CargoItemID == ci.ID && a.HouseBillID == hID {
								gw, parseErr := decimal.NewFromString(a.GrossWeightKg)
								if parseErr != nil {
									return parseErr
								}
								vol, parseErr := decimal.NewFromString(a.VolumeCbm)
								if parseErr != nil {
									return parseErr
								}
								origAllocPkg += a.PackageCount
								origAllocWeight = origAllocWeight.Add(gw)
								origAllocVol = origAllocVol.Add(vol)
							}
						}
					}

					if origAllocPkg > 0 {
						fWeight, _ := origAllocWeight.Float64()
						fVol, _ := origAllocVol.Float64()

						if _, err := tx.OrderCargoItem.UpdateOneID(ci.ID).
							SetPackageCount(origAllocPkg).
							SetGrossWeightKg(fWeight).
							SetVolumeCbm(fVol).
							SetVersion(ci.Version + 1).
							Save(ctx); err != nil {
							return err
						}
						resultCargoRetainedIDs[res.ClientResultKey] = append(resultCargoRetainedIDs[res.ClientResultKey], ci.ID.String())
						resultCargoOldNewMap[res.ClientResultKey][ci.ID.String()] = ci.ID.String()
					} else {
						// 零剩余货物行先记录，待删除旧 allocation 后统一从数据库删除以避免外键约束冲突
						zeroRemainingCargoIDs = append(zeroRemainingCargoIDs, ci.ID)
					}
				}
			}
		}

		// A.2: 来源旧 SeaCargoAllocation 全部删除，并为每个结果全部创建新 UUID 的 allocation!
		if _, err := tx.SeaCargoAllocation.Delete().
			Where(seacargoallocationent.OrderIDEQ(sourceOrder.ID)).
			Exec(ctx); err != nil {
			return err
		}

		// 零剩余货物行在旧分配删除后彻底物理删除
		for _, zID := range zeroRemainingCargoIDs {
			if err := tx.OrderCargoItem.DeleteOneID(zID).Exec(ctx); err != nil {
				return err
			}
		}

		for _, res := range input.Results {
			targetOrderID := resultOrderMap[res.ClientResultKey]
			targetLinkID := resultLinkMap[res.ClientResultKey]

			for _, hID := range res.HouseBillIDs {
				for _, a := range allocations {
					if a.HouseBillID == hID {
						var targetCargoItemUUID uuid.UUID
						if res.ResultRole == biz.ResultRoleOriginal {
							targetCargoItemUUID = a.CargoItemID
						} else {
							newCargoIDStr, ok := resultCargoOldNewMap[res.ClientResultKey][a.CargoItemID.String()]
							if !ok {
								return biz.ErrSeaCargoAllocationNotFound
							}
							targetCargoItemUUID = uuid.Must(uuid.Parse(newCargoIDStr))
						}

						newAllocID := uuid.Must(uuid.NewV7())
						allocCreate := tx.SeaCargoAllocation.Create().
							SetID(newAllocID).
							SetOrganizationID(organizationID).
							SetOrderID(targetOrderID).
							SetMasterBillOrderLinkID(targetLinkID).
							SetCargoItemID(targetCargoItemUUID).
							SetHouseBillID(hID).
							SetPackageCount(a.PackageCount).
							SetGrossWeightKg(a.GrossWeightKg).
							SetVolumeCbm(a.VolumeCbm)
						if a.ContainerID != nil {
							allocCreate.SetContainerID(*a.ContainerID)
						}
						if _, err := allocCreate.Save(ctx); err != nil {
							return err
						}
						resultAllocOldNewMap[res.ClientResultKey][a.ID.String()] = newAllocID.String()
					}
				}
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
				"house_bill_ids":            res.HouseBillIDs,
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
		WithMasterBill(func(q *ent.SeaMasterBillQuery) {
			q.WithTransportExecution()
		}).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	curMBL := activeLink.Edges.MasterBill
	if curMBL == nil {
		return nil, biz.ErrSeaMasterBillNotFound
	}
	curSummary, err := mblToSummary(ctx, client, organizationID, curMBL)
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
	actions, err := r.GetChangeActions(ctx, organizationID, input.OrderID)
	if err != nil {
		return nil, err
	}
	if !actions.CanReassign {
		preview.IsValid = false
		preview.Errors = append(preview.Errors, actions.ReassignBlockedReasons...)
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
			WithTransportExecution().
			WithOrderLinks(func(lq *ent.SeaMasterBillOrderLinkQuery) {
				lq.Where(seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE))
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
		if candMBL.ID == curMBL.ID {
			preview.IsValid = false
			preview.Errors = append(preview.Errors, "目标母单与当前母单相同，无需改配")
			return preview, nil
		}
		candidateTE := candMBL.Edges.TransportExecution
		if candidateTE == nil ||
			candMBL.Status != seamasterbillent.StatusDRAFT ||
			candMBL.Version != *input.Target.CandidateVersion ||
			candMBL.TransportExecutionID != *input.Target.CandidateTEID ||
			candidateTE.Version != *input.Target.CandidateTEVersion {
			return nil, biz.ErrSeaOrderReassignmentVersionConflict
		}
		if candMBL.IssuerPartnerID != *input.Target.IssuerPartnerID {
			return nil, biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
				"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
			})
		}
		targetSummary, err = mblToSummary(ctx, client, organizationID, candMBL)
		if err != nil {
			return nil, err
		}
		targetMemberCount = int32(len(candMBL.Edges.OrderLinks))
	case biz.SplitTargetTypeNew:
		splitTarget := &biz.SeaOrderSplitTargetInput{
			MasterNo:            input.Target.MasterNo,
			IssuerPartnerID:     input.Target.IssuerPartnerID,
			CarrierID:           input.Target.CarrierID,
			VesselName:          input.Target.VesselName,
			VoyageNo:            input.Target.VoyageNo,
			ETD:                 input.Target.ETD,
			ETA:                 input.Target.ETA,
			OriginLocationID:    input.Target.OriginLocationID,
			DischargeLocationID: input.Target.DischargeLocationID,
			TransitLocationID:   input.Target.TransitLocationID,
		}
		normalizedMasterNo, err := validateNewMasterBillInput(ctx, client, organizationID, splitTarget)
		if err != nil {
			return nil, err
		}
		targetSummary = &biz.SeaMasterBillSummary{
			MasterNo:   normalizedMasterNo,
			VesselName: input.Target.VesselName,
			VoyageNo:   input.Target.VoyageNo,
			ETD:        input.Target.ETD,
			ETA:        input.Target.ETA,
		}
		if input.Target.IssuerPartnerID != nil {
			targetSummary.IssuerPartnerID = *input.Target.IssuerPartnerID
			p, err := client.Partner.Query().Where(
				partnerent.IDEQ(*input.Target.IssuerPartnerID),
				partnerent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.IssuerPartnerName = p.LegalName
		}
		if input.Target.CarrierID != nil {
			targetSummary.CarrierID = input.Target.CarrierID
			p, err := client.Partner.Query().Where(
				partnerent.IDEQ(*input.Target.CarrierID),
				partnerent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			targetSummary.CarrierName = p.LegalName
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
		makeDiff("carrier_name", "承运人/船东", curSummary.CarrierName, targetSummary.CarrierName),
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
		teIDs := []uuid.UUID{oldMBL.TransportExecutionID}
		if input.Target.TargetType == biz.SplitTargetTypeCandidate && input.Target.CandidateID != nil && *input.Target.CandidateID != uuid.Nil {
			candMBL := mbls[*input.Target.CandidateID]
			teIDs = append(teIDs, candMBL.TransportExecutionID)
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
		oldTE := lockedTEs[oldMBL.TransportExecutionID]

		// 门禁检查
		nonDraftHblCount, err := tx.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(order.ID), seahousebillent.StatusNEQ(seahousebillent.StatusDRAFT)).
			Count(ctx)
		if err != nil {
			return err
		}
		if nonDraftHblCount > 0 {
			return biz.ErrSeaOrderReassignmentBlocked
		}

		nonDraftFeeCount, err := tx.OrderFee.Query().
			Where(
				orderfeeent.OrderIDEQ(order.ID),
				orderfeeent.StatusNEQ(orderfeeent.StatusDRAFT),
				orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
			).
			Count(ctx)
		if err != nil {
			return err
		}
		if nonDraftFeeCount > 0 {
			return biz.ErrSeaOrderReassignmentBlocked
		}

		hasBillLine, err := tx.FinanceBillLine.Query().
			Where(financebilllineent.OrderIDEQ(order.ID), financebilllineent.ActiveEQ(true)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasBillLine {
			return biz.ErrSeaOrderReassignmentBlocked
		}

		hasCommission, err := tx.FinanceCommissionLine.Query().
			Where(financecommissionlineent.OrderIDEQ(order.ID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if hasCommission {
			return biz.ErrSeaOrderReassignmentBlocked
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
			if targetMBLID == oldLink.MasterBillID {
				return biz.ErrSeaOrderReassignmentTargetConflict
			}
			targetMBL := mbls[targetMBLID]
			if targetMBL == nil || input.Target.CandidateVersion == nil || targetMBL.Version != *input.Target.CandidateVersion ||
				input.Target.CandidateTEID == nil || *input.Target.CandidateTEID == uuid.Nil ||
				targetMBL.TransportExecutionID != *input.Target.CandidateTEID {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}
			if targetMBL.Status != seamasterbillent.StatusDRAFT {
				return biz.ErrSeaOrderReassignmentBlocked
			}
			if input.ExpectedCandidateMBLVersion == nil || *input.ExpectedCandidateMBLVersion == 0 || targetMBL.Version != *input.ExpectedCandidateMBLVersion {
				return biz.ErrSeaOrderReassignmentVersionConflict
			}
			targetMBLNo = targetMBL.MasterNo
			targetTEID = targetMBL.TransportExecutionID
			targetTE = lockedTEs[targetTEID]
			if targetTE == nil {
				return biz.ErrSeaTransportExecutionNotFound
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
			if input.Target.IssuerPartnerID != nil && *input.Target.IssuerPartnerID != uuid.Nil && targetMBL.IssuerPartnerID != *input.Target.IssuerPartnerID {
				return biz.MetadataError(biz.ErrSeaOrderReassignmentBlocked, map[string]string{
					"reason": "CANDIDATE_MBL_INPUT_MISMATCH",
				})
			}
			if input.Target.CarrierID != nil && *input.Target.CarrierID != uuid.Nil && (targetTE.CarrierID == nil || *targetTE.CarrierID != *input.Target.CarrierID) {
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
				IssuerPartnerID:     input.Target.IssuerPartnerID,
				CarrierID:           input.Target.CarrierID,
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
			SetOrderID(order.ID).
			SetDocumentStructure(oldLink.DocumentStructure).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetStartedAt(now).
			SetVersion(1)
		if oldLink.CargoAllocationStatus == seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			newLinkBuilder.SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED)
			newLinkBuilder.SetCargoAllocationVersion(oldLink.CargoAllocationVersion)
			if oldLink.CargoAllocationConfirmedAt != nil {
				newLinkBuilder.SetCargoAllocationConfirmedAt(*oldLink.CargoAllocationConfirmedAt)
			}
			if oldLink.CargoAllocationConfirmedBy != nil {
				newLinkBuilder.SetCargoAllocationConfirmedBy(*oldLink.CargoAllocationConfirmedBy)
			}
		} else {
			newLinkBuilder.SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusDRAFT)
			newLinkBuilder.SetCargoAllocationVersion(1)
		}
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

		// 锁定并迁移该订单现有的箱货分配至新活动 Link (UUID 升序锁定)
		allocs, err := tx.SeaCargoAllocation.Query().
			Where(seacargoallocationent.OrderIDEQ(order.ID)).
			Order(seacargoallocationent.ByID()).
			ForUpdate().
			All(ctx)
		if err != nil {
			return err
		}
		for _, a := range allocs {
			if _, err := tx.SeaCargoAllocation.UpdateOneID(a.ID).
				SetMasterBillOrderLinkID(newLink.ID).
				Save(ctx); err != nil {
				return err
			}
		}

		beforeSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"link": map[string]interface{}{
				"id":                            oldLink.ID,
				"version":                       oldLink.Version,
				"status":                        oldLink.Status,
				"cargo_allocation_status":       oldLink.CargoAllocationStatus,
				"cargo_allocation_version":      oldLink.CargoAllocationVersion,
				"cargo_allocation_confirmed_at": oldLink.CargoAllocationConfirmedAt,
				"cargo_allocation_confirmed_by": oldLink.CargoAllocationConfirmedBy,
			},
			"master_bill": map[string]interface{}{
				"id":                oldMBL.ID,
				"master_no":         oldMBL.MasterNo,
				"issuer_partner_id": oldMBL.IssuerPartnerID,
				"version":           oldMBL.Version,
				"status":            oldMBL.Status,
			},
			"transport_execution": map[string]interface{}{
				"id":                    oldTE.ID,
				"version":               oldTE.Version,
				"vessel_name":           oldTE.VesselName,
				"voyage_no":             oldTE.VoyageNo,
				"carrier_id":            oldTE.CarrierID,
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
			afterMBLIssuer = cand.IssuerPartnerID
			afterMBLStatus = string(cand.Status)
		} else if input.Target.IssuerPartnerID != nil {
			afterMBLIssuer = *input.Target.IssuerPartnerID
		}

		afterSnapshotMap := map[string]interface{}{
			"schema_version": 1,
			"link": map[string]interface{}{
				"id":                            newLink.ID,
				"version":                       newLink.Version,
				"status":                        newLink.Status,
				"cargo_allocation_status":       newLink.CargoAllocationStatus,
				"cargo_allocation_version":      newLink.CargoAllocationVersion,
				"cargo_allocation_confirmed_at": newLink.CargoAllocationConfirmedAt,
				"cargo_allocation_confirmed_by": newLink.CargoAllocationConfirmedBy,
			},
			"master_bill": map[string]interface{}{
				"id":                targetMBLID,
				"master_no":         targetMBLNo,
				"issuer_partner_id": afterMBLIssuer,
				"version":           afterMBLVersion,
				"status":            afterMBLStatus,
			},
			"transport_execution": map[string]interface{}{
				"id":                    targetTE.ID,
				"version":               targetTE.Version,
				"vessel_name":           targetTE.VesselName,
				"voyage_no":             targetTE.VoyageNo,
				"carrier_id":            targetTE.CarrierID,
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

		orderUpdate := tx.Order.UpdateOneID(order.ID).
			SetVesselVoyage(biz.CombineVesselVoyage(targetTE.VesselName, targetTE.VoyageNo)).
			SetVersion(order.Version + 1)
		if targetTE.CarrierID != nil {
			orderUpdate.SetCarrierID(*targetTE.CarrierID)
		} else {
			orderUpdate.ClearCarrierID()
		}
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
			SetCreatedBy(actorID)

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
		},
	}, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

func validateNewMasterBillInput(ctx context.Context, client *ent.Client, organizationID uuid.UUID, target *biz.SeaOrderSplitTargetInput) (string, error) {
	if target == nil {
		return "", biz.ErrSeaMasterBillInvalidArgument
	}
	normalizedMasterNo, err := biz.ValidateAndNormalizeSeaMasterNo(target.MasterNo)
	if err != nil {
		return "", err
	}

	if target.IssuerPartnerID == nil || *target.IssuerPartnerID == uuid.Nil {
		return "", biz.ErrSeaMasterBillInvalidArgument
	}
	issuerExists, err := client.PartnerRole.Query().Where(
		partnerroleent.PartnerIDEQ(*target.IssuerPartnerID),
		partnerroleent.RoleTypeIn(partnerroleent.RoleTypeSupplier, partnerroleent.RoleTypeCarrier),
		partnerroleent.EnabledEQ(true),
		partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)),
	).Exist(ctx)
	if err != nil {
		return "", err
	}
	if !issuerExists {
		return "", biz.ErrSeaMasterBillInvalidArgument
	}

	if target.CarrierID != nil && *target.CarrierID != uuid.Nil {
		carrierExists, err := client.PartnerRole.Query().Where(
			partnerroleent.PartnerIDEQ(*target.CarrierID),
			partnerroleent.RoleTypeEQ(partnerroleent.RoleTypeCarrier),
			partnerroleent.EnabledEQ(true),
			partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)),
		).Exist(ctx)
		if err != nil {
			return "", err
		}
		if !carrierExists {
			return "", biz.ErrSeaMasterBillInvalidArgument
		}
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
			return "", err
		}
		if portCount != len(portIDs) {
			return "", biz.ErrSeaMasterBillInvalidArgument
		}
	}

	if strings.TrimSpace(target.ETD) != "" && parseOptionalTime(target.ETD) == nil {
		return "", biz.ErrSeaMasterBillInvalidArgument
	}
	if strings.TrimSpace(target.ETA) != "" && parseOptionalTime(target.ETA) == nil {
		return "", biz.ErrSeaMasterBillInvalidArgument
	}

	existingMBL, err := client.SeaMasterBill.Query().Where(
		seamasterbillent.OrganizationIDEQ(organizationID),
		seamasterbillent.IssuerPartnerIDEQ(*target.IssuerPartnerID),
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

func createNewMasterBillInTx(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, target *biz.SeaOrderSplitTargetInput) (*ent.SeaMasterBill, *ent.SeaTransportExecution, error) {
	normalizedMasterNo, err := validateNewMasterBillInput(ctx, tx.Client(), organizationID, target)
	if err != nil {
		return nil, nil, err
	}

	teBuilder := tx.SeaTransportExecution.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetVesselName(target.VesselName).
		SetVoyageNo(target.VoyageNo).
		SetVersion(1)
	if target.CarrierID != nil && *target.CarrierID != uuid.Nil {
		teBuilder.SetCarrierID(*target.CarrierID)
	}
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
		return nil, nil, err
	}

	mbl, err := tx.SeaMasterBill.Create().
		SetID(uuid.Must(uuid.NewV7())).
		SetOrganizationID(organizationID).
		SetIssuerPartnerID(*target.IssuerPartnerID).
		SetTransportExecutionID(te.ID).
		SetMasterNo(normalizedMasterNo).
		SetNormalizedMasterNo(normalizedMasterNo).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		return nil, nil, mapEntConstraint(err, "seamasterbill_organization_id_issuer_partner_id_normalized_master_no", biz.ErrSeaMasterBillExists)
	}

	return mbl, te, nil
}

func mblToSummary(ctx context.Context, client *ent.Client, organizationID uuid.UUID, mbl *ent.SeaMasterBill) (*biz.SeaMasterBillSummary, error) {
	s := &biz.SeaMasterBillSummary{
		MasterBillID:    mbl.ID,
		MasterNo:        mbl.MasterNo,
		IssuerPartnerID: mbl.IssuerPartnerID,
		Status:          string(mbl.Status),
		Version:         mbl.Version,
	}
	issuer, err := client.Partner.Query().Where(
		partnerent.IDEQ(mbl.IssuerPartnerID),
		partnerent.OrganizationIDEQ(organizationID),
	).Only(ctx)
	if err != nil {
		return nil, err
	}
	s.IssuerPartnerName = issuer.LegalName
	if te := mbl.Edges.TransportExecution; te != nil {
		s.TransportExecutionID = te.ID
		s.TransportExecutionVersion = te.Version
		s.VesselName = te.VesselName
		s.VoyageNo = te.VoyageNo
		if te.CarrierID != nil {
			s.CarrierID = te.CarrierID
			carrier, err := client.Partner.Query().Where(
				partnerent.IDEQ(*te.CarrierID),
				partnerent.OrganizationIDEQ(organizationID),
			).Only(ctx)
			if err != nil {
				return nil, err
			}
			s.CarrierName = carrier.LegalName
		}
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
