package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderattachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seaordersplitresultent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitresult"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
)

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
	// 订单业务内容门禁：锁定/终止流程/终止/结案订单在预览阶段即写入 ORDER_GATE
	// 原因（与 Execute 的 409 门禁同源）；预览照常返回，不升级为错误。
	sourceOrder, gateQueryErr := client.Order.Query().
		Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if gateQueryErr != nil {
		return nil, mapEntError(gateQueryErr, biz.ErrOrderNotFound, nil)
	}
	if gateErr := ensureOrderBusinessContentEditable(ctx, client.User, sourceOrder); gateErr != nil {
		preview.IsValid = false
		preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
			Reason:  "ORDER_GATE",
			Message: "订单 " + sourceOrder.OrderNo + " " + orderBusinessEditBlockReason(gateErr),
		})
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

	// 3. 校验 HBL 分单号（与 ExecuteSplit 批次内一号一案同口径：请求内按目标
	// 批次去重；目标为已有主单时按（目标主单, 规范化分单号）查重，含作废行；
	// 新主单目标为全新空批次，无需查库。组织级排重已随全局唯一索引撤销移除，
	// 跨批次重号合法。）
	targetBatchKeys := make(map[string]string)
	targetBatchMbls := make(map[string]*uuid.UUID)
	for _, target := range input.Targets {
		switch target.TargetType {
		case biz.SplitTargetTypeCurrent:
			if splitCtx.CurrentMasterBill != nil {
				mblID := splitCtx.CurrentMasterBill.MasterBillID
				targetBatchKeys[target.ClientTargetKey] = "mbl:" + mblID.String()
				targetBatchMbls[target.ClientTargetKey] = &mblID
			} else {
				targetBatchKeys[target.ClientTargetKey] = "current:" + target.ClientTargetKey
			}
		case biz.SplitTargetTypeCandidate:
			if target.CandidateID != nil {
				mblID := *target.CandidateID
				targetBatchKeys[target.ClientTargetKey] = "mbl:" + mblID.String()
				targetBatchMbls[target.ClientTargetKey] = &mblID
			} else {
				targetBatchKeys[target.ClientTargetKey] = "candidate:" + target.ClientTargetKey
			}
		default: // NEW：全新批次
			targetBatchKeys[target.ClientTargetKey] = "new:" + target.ClientTargetKey
		}
	}
	seenHouseNos := make(map[string]string)
	if splitCtx.CurrentHouseBill != nil {
		normCurrent, err := biz.NormalizeSeaHouseNo(splitCtx.CurrentHouseBill.HouseNo)
		if err == nil {
			for _, res := range input.Results {
				if res.ResultRole == biz.ResultRoleOriginal {
					if batchKey, ok := targetBatchKeys[res.ClientTargetKey]; ok {
						seenHouseNos[batchKey+"|"+normCurrent] = "ORIGINAL"
					}
					break
				}
			}
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
						batchKey := targetBatchKeys[res.ClientTargetKey]
						if prevRes, seen := seenHouseNos[batchKey+"|"+normNo]; seen {
							preview.IsValid = false
							preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
								Reason:          "HOUSE_BILL_DUPLICATE",
								Message:         fmt.Sprintf("分单号 %s 在目标批次内重复 (与 %s 冲突)", res.HouseBill.HouseNo, prevRes),
								ClientResultKey: res.ClientResultKey,
							})
						} else {
							seenHouseNos[batchKey+"|"+normNo] = res.ClientResultKey
							// 目标为已有主单批次时与库中既有行（含作废）比对。
							if batchMblID := targetBatchMbls[res.ClientTargetKey]; batchMblID != nil {
								exists, qErr := client.SeaHouseBill.Query().Where(
									seahousebillent.OrganizationIDEQ(organizationID),
									seahousebillent.MasterBillIDEQ(*batchMblID),
									seahousebillent.NormalizedHouseNoEQ(normNo),
								).Exist(ctx)
								if qErr == nil && exists {
									preview.IsValid = false
									preview.ValidationErrors = append(preview.ValidationErrors, &biz.SeaOrderSplitValidationError{
										Reason:          "HOUSE_BILL_EXISTS",
										Message:         fmt.Sprintf("分单号 %s 在该主单批次内已存在（含作废）", res.HouseBill.HouseNo),
										ClientResultKey: res.ClientResultKey,
									})
								}
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
