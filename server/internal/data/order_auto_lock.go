package data

import (
	"context"
	"fmt"
	"sort"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// orderAutoLockRepo 实现结清事件驱动的自动业务锁定检查。
// 检查只在原业务事务成功提交后由触发方调用，使用独立事务；任何失败都只写入
// 结构化审计并反馈给调用方记录日志，绝不回滚或改变已提交的业务事实。
type orderAutoLockRepo struct {
	data *Data
}

func NewAutoOrderLockRepo(d *Data) biz.AutoOrderLockRepo { return &orderAutoLockRepo{d} }

// autoLockCheckOutcome 记录一次自动锁定检查的结构化结果。
type autoLockCheckOutcome struct {
	targetOrderID  uuid.UUID
	targetOrderNo  string
	affectedOrders []string
	lockedOrders   []string
	reasonCode     string
	detail         string
}

// autoLockOrderScope 单订单组织范围谓词。
type autoLockOrderScope struct {
	organizationID uuid.UUID
	orderID        uuid.UUID
}

// RunAutoSettlementLockCheck 解析触发来源影响的订单并对每个订单（或共用 MBL 的
// 成员组）执行一次独立事务的自动锁定检查。资格不满足或执行失败只写审计；
// 本方法返回错误仅表示检查或审计写入本身失败，由触发方记录日志。
func (r *orderAutoLockRepo) RunAutoSettlementLockCheck(ctx context.Context, trigger biz.AutoOrderLockTrigger) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}

	orderIDs, err := r.resolveAffectedOrderIDs(ctx, client, trigger)
	if err != nil {
		// 解析阶段失败：以触发单据为目标写执行失败审计后返回。
		return r.writeCheckAudit(ctx, trigger, &autoLockCheckOutcome{
			targetOrderID: trigger.OrderID,
			reasonCode:    biz.AutoLockReasonExecutionFailed,
			detail:        "解析受影响订单失败: " + err.Error(),
		})
	}
	if len(orderIDs) == 0 {
		// 应付方向核销或无有效应收分摊：不构成应收结清事件，静默返回。
		return nil
	}

	// 未建账费用删除/费用进入账单触发：只对已存在有效应收结清事实的订单重试；
	// 纯成本、从未产生应收或仅有作废应收的订单不因余额为零进入检查。
	if trigger.Type == biz.AutoLockTriggerFeeCancel || trigger.Type == biz.AutoLockTriggerFeeBilled {
		qualified := make([]uuid.UUID, 0, len(orderIDs))
		for _, orderID := range orderIDs {
			hasSettlement, err := orderHasValidReceivableSettlement(ctx, client, autoLockOrderScope{
				organizationID: trigger.OrganizationID,
				orderID:        orderID,
			})
			if err != nil {
				return r.writeCheckAudit(ctx, trigger, &autoLockCheckOutcome{
					targetOrderID: orderID,
					reasonCode:    biz.AutoLockReasonExecutionFailed,
					detail:        "预检有效结清事实失败: " + err.Error(),
				})
			}
			if hasSettlement {
				qualified = append(qualified, orderID)
			}
		}
		orderIDs = qualified
		if len(orderIDs) == 0 {
			return nil
		}
	}

	handledGroups := make(map[uuid.UUID]struct{})
	for _, orderID := range orderIDs {
		plan, planErr := r.planCheck(ctx, client, trigger.OrganizationID, orderID)
		if planErr != nil {
			if auditErr := r.writeCheckAudit(ctx, trigger, &autoLockCheckOutcome{
				targetOrderID: orderID,
				reasonCode:    biz.AutoLockReasonExecutionFailed,
				detail:        "定位订单共享形态失败: " + planErr.Error(),
			}); auditErr != nil {
				return auditErr
			}
			continue
		}
		if plan.skip {
			continue
		}
		if plan.group {
			if _, handled := handledGroups[plan.mblID]; handled {
				continue
			}
		}
		var outcome *autoLockCheckOutcome
		if plan.group {
			outcome = r.runGroupCheck(ctx, trigger, plan)
			handledGroups[plan.mblID] = struct{}{}
		} else {
			outcome = r.runSingleCheck(ctx, trigger, orderID)
		}
		if auditErr := r.writeCheckAudit(ctx, trigger, outcome); auditErr != nil {
			return auditErr
		}
	}
	return nil
}

// autoLockCheckPlan 只读定位订单的自动锁定执行形态：共用 MBL 的多成员 SE 订单
// 提升为组级检查，其余按单票检查。
type autoLockCheckPlan struct {
	skip      bool
	group     bool
	mblID     uuid.UUID
	memberIDs []uuid.UUID
}

func (r *orderAutoLockRepo) planCheck(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*autoLockCheckPlan, error) {
	order, err := client.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	businessType, parseErr := orderAccessBusinessType(order.BusinessType)
	if parseErr != nil {
		return nil, parseErr
	}
	if businessType == access.OrderBusinessSE {
		link, linkErr := client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if linkErr == nil {
			memberIDs, memberErr := seaGroupMemberOrderIDs(ctx, client, link.MasterBillID)
			if memberErr != nil {
				return nil, memberErr
			}
			if len(memberIDs) > 1 {
				return &autoLockCheckPlan{group: true, mblID: link.MasterBillID, memberIDs: memberIDs}, nil
			}
		} else if !ent.IsNotFound(linkErr) {
			return nil, linkErr
		}
	}
	return &autoLockCheckPlan{}, nil
}

// runSingleCheck 单票自动锁定检查：Order 行锁内重验生命周期、有效结清事实、
// 订单级未结清应收、未建账费用与现有锁定条件后写入锁定事实。
func (r *orderAutoLockRepo) runSingleCheck(ctx context.Context, trigger biz.AutoOrderLockTrigger, orderID uuid.UUID) *autoLockCheckOutcome {
	outcome := &autoLockCheckOutcome{
		targetOrderID:  orderID,
		reasonCode:     biz.AutoLockReasonLocked,
		affectedOrders: []string{},
		lockedOrders:   []string{},
	}
	organizationID := trigger.OrganizationID
	checkErr := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		outcome.targetOrderNo = order.OrderNo
		outcome.affectedOrders = []string{order.OrderNo}
		// 已锁定时幂等结束：人工锁定与自动锁定并发时最多一个锁代次。
		if order.LockedAt != nil {
			outcome.reasonCode = biz.AutoLockReasonAlreadyLocked
			return nil
		}
		if reason := autoLockLifecycleBlockReason(order); reason != "" {
			outcome.reasonCode = reason
			return nil
		}
		eligible, reason, evalErr := evaluateAutoLockEligibility(ctx, tx, orderID)
		if evalErr != nil {
			return evalErr
		}
		if !eligible {
			outcome.reasonCode = reason
			return nil
		}
		if _, _, coreErr := applyOrderLockCore(ctx, tx, organizationID, order, autoLockCoreSpec(trigger, orderID)); coreErr != nil {
			return coreErr
		}
		outcome.lockedOrders = []string{order.OrderNo}
		return nil
	})
	if checkErr != nil {
		outcome.reasonCode, outcome.detail = autoLockFailureOutcome(checkErr)
	}
	return outcome
}

// runGroupCheck 共用 MBL 的成员组级自动锁定检查：先按 UUID 升序锁定全部成员
// Order，再锁 MBL、按 ID 升序的全部活动 Link、共享运输执行与各成员 HBL，
// 锁内重验活动成员集合；全部尚未锁定成员各自合格才在同一事务整组锁定，
// 任一不合格、集合变化或快照失败整组零写入。
func (r *orderAutoLockRepo) runGroupCheck(ctx context.Context, trigger biz.AutoOrderLockTrigger, plan *autoLockCheckPlan) *autoLockCheckOutcome {
	outcome := &autoLockCheckOutcome{
		targetOrderID:  plan.memberIDs[0],
		reasonCode:     biz.AutoLockReasonLocked,
		affectedOrders: []string{},
		lockedOrders:   []string{},
	}
	organizationID := trigger.OrganizationID
	checkErr := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 固定锁序：按 Order UUID 升序一次性锁定全部成员。
		members, queryErr := tx.Order.Query().
			Where(orderent.IDIn(plan.memberIDs...), orderent.OrganizationIDEQ(organizationID)).
			Order(ent.Asc(orderent.FieldID)).
			ForUpdate().
			All(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(members) != len(plan.memberIDs) {
			return biz.ErrSeaDocumentStructureConflict
		}
		var unlockedMembers []*ent.Order
		for _, member := range members {
			outcome.affectedOrders = append(outcome.affectedOrders, member.OrderNo)
			if outcome.targetOrderNo == "" {
				outcome.targetOrderNo = member.OrderNo
			}
			if member.LockedAt == nil {
				unlockedMembers = append(unlockedMembers, member)
			}
		}
		// 全部活动成员均已锁定：幂等结束，不重复执行组级单证加锁，
		// 也不创建没有对应锁定记录的 MBL/运输执行版本。
		if len(unlockedMembers) == 0 {
			outcome.reasonCode = biz.AutoLockReasonAlreadyLocked
			return nil
		}

		// 2. 锁 MBL → 按 ID 升序的全部活动 Link → 共享运输执行 → 各成员 HBL，
		//    随后重验活动成员集合；已锁成员必须属于同一集合，否则整组零写入。
		graph, graphErr := lockSeaGroupDocumentGraph(ctx, tx, organizationID, plan.mblID, plan.memberIDs)
		if graphErr != nil {
			return graphErr
		}

		// 3. 全部尚未锁定成员各自合格才整组锁定；任一不合格只记录原因。
		for _, member := range unlockedMembers {
			if reason := autoLockLifecycleBlockReason(member); reason != "" {
				outcome.reasonCode = biz.AutoLockReasonMemberNotQualified
				outcome.detail = member.OrderNo + ":" + reason
				return nil
			}
			eligible, reason, evalErr := evaluateAutoLockEligibility(ctx, tx, member.ID)
			if evalErr != nil {
				return evalErr
			}
			if !eligible {
				outcome.reasonCode = biz.AutoLockReasonMemberNotQualified
				outcome.detail = member.OrderNo + ":" + reason
				return nil
			}
		}

		// 4. 整组复用同一份 MBL/运输执行不可变版本；自动锁定新建版本 created_by 为空。
		masterBillVersionID, transportExecutionVersionID, versionErr := ensureSeaGroupSharedVersions(ctx, tx, organizationID, graph, nil)
		if versionErr != nil {
			return versionErr
		}

		// 5. 逐成员生成 HBL 快照、锁定记录与订单锁状态。
		now := time.Now().UTC()
		for _, member := range unlockedMembers {
			if err := lockAutoGroupMember(ctx, tx, organizationID, member, trigger, graph, masterBillVersionID, transportExecutionVersionID, now); err != nil {
				return err
			}
			outcome.lockedOrders = append(outcome.lockedOrders, member.OrderNo)
		}
		return nil
	})
	if checkErr != nil {
		outcome.reasonCode, outcome.detail = autoLockFailureOutcome(checkErr)
	}
	return outcome
}

// lockAutoGroupMember 为组内单个成员写入锁状态、共享版本引用的锁定记录与
// 自身 HBL 快照。调用方必须已持有该成员 Order 行锁与整组单证行锁。
func lockAutoGroupMember(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	member *ent.Order,
	trigger biz.AutoOrderLockTrigger,
	graph *seaGroupLockContext,
	masterBillVersionID, transportExecutionVersionID uuid.UUID,
	now time.Time,
) error {
	var memberLink *ent.SeaMasterBillOrderLink
	for _, link := range graph.Links {
		if link.OrderID == member.ID {
			memberLink = link
			break
		}
	}
	if memberLink == nil {
		return biz.ErrSeaDocumentStructureConflict
	}
	hbls := graph.HouseBillsByOrder[member.ID]
	// 结构校验与单票快照一致：DIRECT/UNDETERMINED 无 HBL，HOUSE 至少一张。
	if memberLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureDIRECT && len(hbls) > 0 {
		return biz.ErrSeaDocumentStructureConflict
	}
	if memberLink.DocumentStructure == seamasterbillorderlinkent.DocumentStructureHOUSE && len(hbls) == 0 {
		return biz.ErrSeaDocumentStructureConflict
	}
	snaps := make([]seaOrderLockHouseBillSnapshot, 0, len(hbls))
	for _, hbl := range hbls {
		versionID, err := ensureSeaHouseBillLockVersion(ctx, tx, organizationID, hbl, member.ID, graph.MasterBill.ID, nil)
		if err != nil {
			return err
		}
		snaps = append(snaps, seaOrderLockHouseBillSnapshot{
			HouseBillID:        hbl.ID,
			HouseBillVersionID: versionID,
			HouseNoSnapshot:    hbl.HouseNo,
		})
	}

	triggerType := orderlockrecordent.TriggerType(trigger.Type)
	newLockGen := member.LockGeneration + 1
	orderVersionAtLock := member.Version + 1
	if _, err := tx.Order.UpdateOne(member).
		SetLockedAt(now).
		SetLockGeneration(newLockGen).
		SetLockSource(orderent.LockSourceAUTO_SETTLEMENT).
		SetAutoLockTriggerType(orderent.AutoLockTriggerType(triggerType)).
		SetAutoLockTriggerResourceID(trigger.ResourceID).
		SetAutoLockTriggeredBy(trigger.TriggeredBy).
		SetVersion(orderVersionAtLock).
		Save(ctx); err != nil {
		return err
	}
	rec, err := tx.OrderLockRecord.Create().
		SetOrganizationID(organizationID).
		SetOrderID(member.ID).
		SetOrderNo(member.OrderNo).
		SetBusinessType(orderlockrecordent.BusinessType(member.BusinessType)).
		SetGeneration(newLockGen).
		SetLockSource(orderlockrecordent.LockSourceAUTO_SETTLEMENT).
		SetLockedAt(now).
		SetOrderVersionAtLock(orderVersionAtLock).
		SetIdempotencyKey(autoLockIdempotencyKey(trigger, member.ID)).
		SetRequestFingerprint(computeAutoLockFingerprint(organizationID, member.ID, trigger.ResourceID, string(triggerType))).
		SetTriggerType(triggerType).
		SetTriggerResourceID(trigger.ResourceID).
		SetTriggeredBy(trigger.TriggeredBy).
		SetMasterBillID(graph.MasterBill.ID).
		SetMasterBillVersionID(masterBillVersionID).
		SetTransportExecutionID(graph.TransportExecution.ID).
		SetTransportExecutionVersionID(transportExecutionVersionID).
		Save(ctx)
	if err != nil {
		return err
	}
	for _, snap := range snaps {
		if _, err := tx.OrderLockHouseBillSnapshot.Create().
			SetOrganizationID(organizationID).
			SetLockRecordID(rec.ID).
			SetHouseBillID(snap.HouseBillID).
			SetHouseBillVersionID(snap.HouseBillVersionID).
			SetHouseNoSnapshot(snap.HouseNoSnapshot).
			Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

// writeCheckAudit 在检查事务之外写入结构化审计，包含稳定原因码、触发类型、
// 触发资源与目标订单。审计写入失败原样返回，由触发方记录日志。
func (r *orderAutoLockRepo) writeCheckAudit(ctx context.Context, trigger biz.AutoOrderLockTrigger, outcome *autoLockCheckOutcome) error {
	details := map[string]string{
		"reason_code":         outcome.reasonCode,
		"trigger_type":        string(trigger.Type),
		"trigger_resource_id": trigger.ResourceID.String(),
		"triggered_by":        trigger.TriggeredBy.String(),
		"affected_order_nos":  joinStrings(outcome.affectedOrders),
		"locked_order_nos":    joinStrings(outcome.lockedOrders),
	}
	if outcome.detail != "" {
		details["detail"] = outcome.detail
	}
	targetID := outcome.targetOrderID
	if targetID == uuid.Nil {
		targetID = trigger.ResourceID
	}
	event := &biz.AuditEvent{
		OrganizationID: &trigger.OrganizationID,
		UserID:         &trigger.TriggeredBy,
		Action:         "order.auto_lock.check",
		Result:         autoLockAuditResult(outcome.reasonCode),
		ResourceType:   "order",
		ResourceID:     targetID.String(),
		Details:        details,
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		return writeAudit(ctx, tx.AuditLog, event)
	})
}

// resolveAffectedOrderIDs 解析触发来源实际影响的订单：跨订单分摊收集全部
// 受影响订单并去重排序，一张跨订单账单不得只检查某个锚点订单。
func (r *orderAutoLockRepo) resolveAffectedOrderIDs(ctx context.Context, client *ent.Client, trigger biz.AutoOrderLockTrigger) ([]uuid.UUID, error) {
	switch trigger.Type {
	case biz.AutoLockTriggerVerification:
		verification, err := client.FinanceVerification.Query().
			Where(
				financeverificationent.IDEQ(trigger.ResourceID),
				financeverificationent.OrganizationIDEQ(trigger.OrganizationID),
			).
			Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrVerificationNotFound, nil)
		}
		// 只有应收核销构成应收结清事件；应付核销不触发自动锁定检查。
		if verification.Direction != financeverificationent.DirectionRECEIVABLE {
			return nil, nil
		}
		allocations, err := client.FinanceVerificationAllocation.Query().
			Where(financeverificationallocationent.VerificationIDEQ(trigger.ResourceID)).
			All(ctx)
		if err != nil {
			return nil, err
		}
		billIDs := make([]uuid.UUID, 0, len(allocations))
		for _, allocation := range allocations {
			billIDs = append(billIDs, allocation.BillID)
		}
		return orderIDsForBills(ctx, client, billIDs)
	case biz.AutoLockTriggerNetting:
		allocations, err := client.FinanceNettingAllocation.Query().
			Where(
				financenettingallocationent.NettingIDEQ(trigger.ResourceID),
				financenettingallocationent.ActiveEQ(true),
				financenettingallocationent.HasBillWith(financebillent.DirectionEQ(financebillent.DirectionRECEIVABLE)),
			).
			All(ctx)
		if err != nil {
			return nil, err
		}
		billIDs := make([]uuid.UUID, 0, len(allocations))
		for _, allocation := range allocations {
			billIDs = append(billIDs, allocation.BillID)
		}
		return orderIDsForBills(ctx, client, billIDs)
	case biz.AutoLockTriggerFeeCancel:
		return []uuid.UUID{trigger.OrderID}, nil
	case biz.AutoLockTriggerFeeBilled:
		// ResourceID 为新账单 ID：按账单行解析全部受影响订单（跨订单账单）。
		return orderIDsForBills(ctx, client, []uuid.UUID{trigger.ResourceID})
	default:
		return nil, nil
	}
}

// orderIDsForBills 解析账单集合关联的全部订单并去重排序。
func orderIDsForBills(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) ([]uuid.UUID, error) {
	if len(billIDs) == 0 {
		return nil, nil
	}
	lines, err := client.FinanceBillLine.Query().
		Where(financebilllineent.BillIDIn(billIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{}, len(lines))
	orderIDs := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		if _, ok := seen[line.OrderID]; ok {
			continue
		}
		seen[line.OrderID] = struct{}{}
		orderIDs = append(orderIDs, line.OrderID)
	}
	sortOrderUUIDs(orderIDs)
	return orderIDs, nil
}

// seaGroupMemberOrderIDs 返回共享 MBL 当前全部活动成员订单 ID（升序）。
func seaGroupMemberOrderIDs(ctx context.Context, client *ent.Client, masterBillID uuid.UUID) ([]uuid.UUID, error) {
	links, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.MasterBillIDEQ(masterBillID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]struct{}, len(links))
	memberIDs := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		if _, ok := seen[link.OrderID]; ok {
			continue
		}
		seen[link.OrderID] = struct{}{}
		memberIDs = append(memberIDs, link.OrderID)
	}
	sortOrderUUIDs(memberIDs)
	return memberIDs, nil
}

// autoLockLifecycleBlockReason 校验订单生命周期：终止/结案订单不参与自动锁定。
func autoLockLifecycleBlockReason(order *ent.Order) string {
	switch order.TerminationStatus {
	case orderent.TerminationStatusACTIVE:
	case orderent.TerminationStatusTERMINATING, orderent.TerminationStatusTERMINATED:
		return biz.AutoLockReasonLifecycle
	default:
		return biz.AutoLockReasonLifecycle
	}
	if order.ClosureStatus != orderent.ClosureStatusOPEN {
		return biz.AutoLockReasonLifecycle
	}
	return ""
}

// evaluateAutoLockEligibility 评估订单的自动锁定财务资格：
//   - 任意方向存在未建账费用即阻止（未建账应收同时构成未结清应收，
//     未建账应付不再因原已确认身份绕过阻断）；
//   - 至少一笔有效应收结清事实（有效核销或已确认对冲分摊命中本订单应收账单）；
//   - 订单级未结清已确认应收为零（应收账单行合计 − 有效核销分摊 − 已确认对冲
//     分摊推导；已反转/已取消/失效分摊不计入）。
func evaluateAutoLockEligibility(ctx context.Context, tx *ent.Tx, orderID uuid.UUID) (eligible bool, reasonCode string, err error) {
	// 1) 任意方向的未建账费用都阻止自动锁定。
	unbilledFeeExists, err := tx.OrderFee.Query().
		Where(orderfeeent.OrderIDEQ(orderID), orderfeeent.StatusEQ(orderfeeent.StatusUNBILLED)).
		Exist(ctx)
	if err != nil {
		return false, "", err
	}
	if unbilledFeeExists {
		return false, biz.AutoLockReasonUnbilledFee, nil
	}

	// 2) 订单关联的应收账单行（含未确认账单的行，仍是未结清应收事实）。
	lines, err := tx.FinanceBillLine.Query().
		Where(
			financebilllineent.OrderIDEQ(orderID),
			financebilllineent.ActiveEQ(true),
			financebilllineent.HasBillWith(
				financebillent.DirectionEQ(financebillent.DirectionRECEIVABLE),
				financebillent.StatusIn(financebillent.StatusDRAFT, financebillent.StatusCONFIRMED),
			),
		).
		All(ctx)
	if err != nil {
		return false, "", err
	}
	if len(lines) == 0 {
		return false, biz.AutoLockReasonNoSettlement, nil
	}
	billIDs := make([]uuid.UUID, 0, len(lines))
	lineSum := decimal.Zero
	for _, line := range lines {
		billIDs = append(billIDs, line.BillID)
		amount, parseErr := decimalOf(line.TotalAmount)
		if parseErr != nil {
			return false, "", parseErr
		}
		lineSum = lineSum.Add(amount)
	}

	// 3) 有效核销分摊：分摊有效且核销单仍然 ACTIVE。
	verifications, err := tx.FinanceVerificationAllocation.Query().
		Where(
			financeverificationallocationent.BillIDIn(billIDs...),
			financeverificationallocationent.ActiveEQ(true),
			financeverificationallocationent.HasVerificationWith(financeverificationent.StatusEQ(financeverificationent.StatusACTIVE)),
		).
		All(ctx)
	if err != nil {
		return false, "", err
	}
	// 4) 已确认对冲分摊：分摊有效且对冲单处于 CONFIRMED。
	nettings, err := tx.FinanceNettingAllocation.Query().
		Where(
			financenettingallocationent.BillIDIn(billIDs...),
			financenettingallocationent.ActiveEQ(true),
			financenettingallocationent.HasNettingWith(financenettingent.StatusEQ(financenettingent.StatusCONFIRMED)),
		).
		All(ctx)
	if err != nil {
		return false, "", err
	}
	settledSum := decimal.Zero
	for _, allocation := range verifications {
		amount, parseErr := decimalOf(allocation.Amount)
		if parseErr != nil {
			return false, "", parseErr
		}
		settledSum = settledSum.Add(amount)
	}
	for _, allocation := range nettings {
		amount, parseErr := decimalOf(allocation.Amount)
		if parseErr != nil {
			return false, "", parseErr
		}
		settledSum = settledSum.Add(amount)
	}

	// 从未发生有效应收结清事件的订单绝不因余额为零进入检查。
	if len(verifications) == 0 && len(nettings) == 0 {
		return false, biz.AutoLockReasonNoSettlement, nil
	}
	// 跨订单账单的分摊是账单级事实，只能按订单行推导；无论剩余为正还是分摊
	// 超过该订单行合计（负值），都无法据此刻定本订单已全额结清，一律 fail-closed
	// 放弃自动锁定（漏锁由人工兜底），不得把负 unsettled 误判为已结清。
	unsettled := lineSum.Sub(settledSum)
	if !unsettled.IsZero() {
		return false, biz.AutoLockReasonUnsettledReceivable, nil
	}
	return true, "", nil
}

// orderHasValidReceivableSettlement 判断订单是否已存在有效应收结清事实，
// 作为未建账费用删除/建账触发的重试预检。
func orderHasValidReceivableSettlement(ctx context.Context, client *ent.Client, scope autoLockOrderScope) (bool, error) {
	billPredicates := []predicate.FinanceBill{
		financebillent.DirectionEQ(financebillent.DirectionRECEIVABLE),
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		financebillent.HasLinesWith(
			financebilllineent.OrderIDEQ(scope.orderID),
			financebilllineent.ActiveEQ(true),
		),
	}
	verExists, err := client.FinanceVerificationAllocation.Query().
		Where(
			financeverificationallocationent.ActiveEQ(true),
			financeverificationallocationent.HasVerificationWith(financeverificationent.StatusEQ(financeverificationent.StatusACTIVE)),
			financeverificationallocationent.HasBillWith(billPredicates...),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	if verExists {
		return true, nil
	}
	netExists, err := client.FinanceNettingAllocation.Query().
		Where(
			financenettingallocationent.ActiveEQ(true),
			financenettingallocationent.HasNettingWith(financenettingent.StatusEQ(financenettingent.StatusCONFIRMED)),
			financenettingallocationent.HasBillWith(billPredicates...),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return netExists, nil
}

// lockOrdersForFinanceReversal 按订单 UUID 升序对财务反转涉及的订单取得
// FOR UPDATE 行锁。该锁只用于与自动锁定按 Order 优先固定锁序线性化，
// 不施加业务内容门禁：订单已锁定不得阻止反核销或反对冲。
func lockOrdersForFinanceReversal(ctx context.Context, tx *ent.Tx, orderIDs []uuid.UUID) error {
	if len(orderIDs) == 0 {
		return nil
	}
	_, err := tx.Order.Query().
		Where(orderent.IDIn(orderIDs...)).
		Order(ent.Asc(orderent.FieldID)).
		ForUpdate().
		All(ctx)
	return err
}

// autoLockCoreSpec 构造自动锁定的核心参数：locked_by 为空，触发审计完整；
// SE 快照新建版本的 created_by 保持为空。
func autoLockCoreSpec(trigger biz.AutoOrderLockTrigger, orderID uuid.UUID) orderLockCoreSpec {
	triggerType := orderlockrecordent.TriggerType(trigger.Type)
	return orderLockCoreSpec{
		lockSource:         biz.LockSourceAutoSettlement,
		idempotencyKey:     autoLockIdempotencyKey(trigger, orderID),
		requestFingerprint: computeAutoLockFingerprint(trigger.OrganizationID, orderID, trigger.ResourceID, string(triggerType)),
		triggerType:        triggerType,
		triggerResourceID:  &trigger.ResourceID,
		triggeredBy:        &trigger.TriggeredBy,
	}
}

// autoLockIdempotencyKey 为自动锁定记录生成确定性幂等键：与人工幂等键空间隔离，
// 同一触发单据对同一订单的重复检查不会追加锁定事实。
func autoLockIdempotencyKey(trigger biz.AutoOrderLockTrigger, orderID uuid.UUID) string {
	return fmt.Sprintf("auto-lock:%s:%s:%s", trigger.Type, trigger.ResourceID, orderID)
}

// autoLockFailureOutcome 把检查事务错误映射为稳定原因码与细节描述。
func autoLockFailureOutcome(err error) (reasonCode string, detail string) {
	detail = err.Error()
	if ke := kratoserrors.FromError(err); ke != nil && ke.Reason != "" {
		switch ke.Reason {
		case "SEA_DOCUMENT_NO_ACTIVE_LINK":
			return biz.AutoLockReasonNoActiveLink, detail
		case "SEA_DOCUMENT_STRUCTURE_CONFLICT":
			return biz.AutoLockReasonDocumentStructure, detail
		case "ORDER_STATUS_CONFLICT", "ORDER_ALREADY_LOCKED":
			return biz.AutoLockReasonAlreadyLocked, detail
		}
	}
	return biz.AutoLockReasonExecutionFailed, detail
}

func autoLockAuditResult(reasonCode string) string {
	// 审计结果只有 success/failure：执行失败类（结构冲突、执行异常）写 failure；
	// 成功锁定、幂等结束与资格不满足的检查本身执行成功，写 success，
	// 检查结论由 reason_code 字段区分。
	switch reasonCode {
	case biz.AutoLockReasonDocumentStructure, biz.AutoLockReasonExecutionFailed:
		return "failure"
	default:
		return "success"
	}
}

func sortOrderUUIDs(values []uuid.UUID) {
	sort.Slice(values, func(i, j int) bool { return values[i].String() < values[j].String() })
}

func joinStrings(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	default:
		out := values[0]
		for _, v := range values[1:] {
			out += "," + v
		}
		return out
	}
}

var _ biz.AutoOrderLockRepo = (*orderAutoLockRepo)(nil)
