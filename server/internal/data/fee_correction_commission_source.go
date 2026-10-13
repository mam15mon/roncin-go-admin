package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	netalloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	veralloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/shopspring/decimal"
)

// commissionEconomicAffectedOrderIDs 在拿资金父锁前预读完整Order集合，含同提成父单其他行。
func commissionEconomicAffectedOrderIDs(ctx context.Context, client *ent.Client, org uuid.UUID, billIDs []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := client.FinanceBillLine.Query().Where(billline.BillIDIn(billIDs...), billline.ActiveEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, l := range rows {
		ids = append(ids, l.OrderID)
	}
	return commissionEconomicExpandOrderIDs(ctx, client, org, ids)
}
func commissionEconomicExpandOrderIDs(ctx context.Context, client *ent.Client, org uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	orders := make(map[uuid.UUID]bool)
	parents := make(map[uuid.UUID]bool)
	frontier := uniqueSortedUUIDs(ids)
	for _, id := range frontier {
		orders[id] = true
	}
	// 同一订单可属于多个来源父单；按批次展开传递闭包，避免AB连到BC时漏掉C。
	for len(frontier) > 0 {
		rows, err := client.FinanceCommissionLine.Query().Where(lineent.OrganizationIDEQ(org), lineent.OrderIDIn(frontier...), lineent.HasCommissionWith(commission.StatusIn(commission.StatusDRAFT, commission.StatusCONFIRMED, commission.StatusPAID))).All(ctx)
		if err != nil {
			return nil, err
		}
		parentIDs := make([]uuid.UUID, 0, len(rows))
		for _, l := range rows {
			if !parents[l.CommissionID] {
				parents[l.CommissionID] = true
				parentIDs = append(parentIDs, l.CommissionID)
			}
		}
		if len(parentIDs) == 0 {
			break
		}
		lines, err := client.FinanceCommissionLine.Query().Where(lineent.CommissionIDIn(parentIDs...)).All(ctx)
		if err != nil {
			return nil, err
		}
		frontier = nil
		for _, l := range lines {
			if !orders[l.OrderID] {
				orders[l.OrderID] = true
				frontier = append(frontier, l.OrderID)
			}
		}
	}
	result := make([]uuid.UUID, 0, len(orders))
	for id := range orders {
		result = append(result, id)
	}
	return uniqueSortedUUIDs(result), nil
}

// reconcileCommissionEconomicSource 必须由已预锁完整Order集合的资金事务调用。
// 先按资金事实重放全部来源，来源撤销只生成一次追回计划。
func reconcileCommissionEconomicSource(ctx context.Context, client *ent.Client, org, actor uuid.UUID, kind string, sourceID uuid.UUID, version uint64) error {
	billIDs := make([]uuid.UUID, 0)
	if kind == "VERIFICATION_CREATE" || kind == "VERIFICATION_REVERSE" {
		rows, err := client.FinanceVerificationAllocation.Query().Where(veralloc.VerificationIDEQ(sourceID)).All(ctx)
		if err != nil {
			return err
		}
		for _, a := range rows {
			billIDs = append(billIDs, a.BillID)
		}
	} else {
		rows, err := client.FinanceNettingAllocation.Query().Where(netalloc.NettingIDEQ(sourceID)).All(ctx)
		if err != nil {
			return err
		}
		for _, a := range rows {
			billIDs = append(billIDs, a.BillID)
		}
	}
	ids, err := commissionEconomicAffectedOrderIDs(ctx, client, org, billIDs)
	if err != nil {
		return err
	}
	// 全部受影响父单先排序一次加锁；后续反转自身父单再重算其他来源不追加反序父锁。
	parentIDs, err := client.FinanceCommissionLine.Query().Where(lineent.OrganizationIDEQ(org), lineent.OrderIDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	pids := make([]uuid.UUID, 0, len(parentIDs))
	for _, l := range parentIDs {
		pids = append(pids, l.CommissionID)
	}
	if _, err := client.FinanceCommission.Query().Where(commission.IDIn(pids...)).Order(commission.ByID()).ForUpdate().All(ctx); err != nil {
		return err
	}
	trigger := biz.CommissionEconomicTrigger{Kind: kind, ID: sourceID, Version: version}
	// 无已计提仍需重放退款依赖，保证反转不能制造负的真实贡献。
	projections := make(map[uuid.UUID]*biz.CommissionEconomicOrderProjection, len(ids))
	for _, orderID := range ids {
		p, err := loadCommissionEconomicProjection(ctx, client, orderID, nil)
		if err != nil {
			return err
		}
		projections[orderID] = p
	}
	if kind == "VERIFICATION_REVERSE" || kind == "NETTING_REVERSE" {
		if err := reverseCommissionEconomicParents(ctx, client, org, actor, trigger); err != nil {
			return err
		}
	}
	for _, orderID := range ids {
		plans, err := planCommissionEconomicRows(ctx, client, org, orderID, projections[orderID], true)
		if err != nil {
			return err
		}
		// 已反转源的PAID父单已由实际现金计划处理，不再按费用目标再次冲减。
		remaining := plans[:0]
		for _, p := range plans {
			same := kind == "VERIFICATION_REVERSE" && p.Parent.VerificationID != nil && *p.Parent.VerificationID == sourceID || kind == "NETTING_REVERSE" && p.Parent.NettingID != nil && *p.Parent.NettingID == sourceID
			if !same {
				remaining = append(remaining, p)
			}
		}
		if err := applyCommissionEconomicPlans(ctx, client, org, actor, trigger, remaining); err != nil {
			return err
		}
	}
	return nil
}
func reverseCommissionEconomicParents(ctx context.Context, client *ent.Client, org, actor uuid.UUID, trigger biz.CommissionEconomicTrigger) error {
	q := client.FinanceCommission.Query().Where(commission.OrganizationIDEQ(org), commission.StatusNEQ(commission.StatusCANCELLED))
	if trigger.Kind == "VERIFICATION_REVERSE" {
		q.Where(commission.VerificationIDEQ(trigger.ID))
	} else {
		q.Where(commission.NettingIDEQ(trigger.ID))
	}
	parents, err := q.Order(commission.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	cancellationReason := "对冲反转自动取消"
	sourceReason := ""
	if trigger.Kind == "VERIFICATION_REVERSE" {
		v, err := client.FinanceVerification.Get(ctx, trigger.ID)
		if err != nil {
			return err
		}
		cancellationReason = "核销撤销自动取消"
		if v.ReversalReason != nil {
			sourceReason = *v.ReversalReason
		}
	} else {
		n, err := client.FinanceNetting.Get(ctx, trigger.ID)
		if err != nil {
			return err
		}
		if n.ReversalReason != nil {
			sourceReason = *n.ReversalReason
		}
	}
	cancellationReason = limitedFinanceReason(cancellationReason + "：" + sourceReason)
	for _, parent := range parents {
		lines, err := client.FinanceCommissionLine.Query().Where(lineent.CommissionIDEQ(parent.ID)).Order(lineent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		rows, err := client.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(parent.ID)).Order(adjustment.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		cash := decimal.Zero
		if parent.Status == commission.StatusPAID {
			cash, err = commissionCashExposure(parent, rows)
			if err != nil {
				return err
			}
		} else {
			for _, a := range rows {
				if a.Status == adjustment.StatusPAID {
					return biz.ErrCommissionPaymentEvidence
				}
			}
		}
		exposure := make(map[uuid.UUID]decimal.Decimal)
		total := decimal.Zero
		for _, l := range lines {
			v, err := decimalOf(l.CommissionAmount)
			if err != nil {
				return err
			}
			exposure[l.OrderID] = v
		}
		cancel := make([]uuid.UUID, 0)
		for _, a := range rows {
			if a.Status == adjustment.StatusPAID {
				v, err := commissionSignedAdjustment(a)
				if err != nil {
					return err
				}
				exposure[a.OrderID] = exposure[a.OrderID].Add(v)
			} else if a.Status == adjustment.StatusDRAFT || a.Status == adjustment.StatusCONFIRMED {
				cancel = append(cancel, a.ID)
			}
		}
		for _, v := range exposure {
			if v.IsNegative() {
				return biz.ErrCommissionPaymentEvidence
			}
			total = total.Add(v)
		}
		if parent.Status == commission.StatusPAID && !total.Equal(cash) {
			return biz.ErrCommissionPaymentEvidence
		}
		if len(cancel) > 0 {
			if _, err := client.FinanceCommissionAdjustment.Update().Where(adjustment.IDIn(cancel...)).SetStatus(adjustment.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason("来源反转后按实际付款余额重建追回").AddVersion(1).Save(ctx); err != nil {
				return err
			}
		}
		if parent.Status != commission.StatusPAID {
			if err := client.FinanceCommission.UpdateOne(parent).SetStatus(commission.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(cancellationReason).AddVersion(1).Exec(ctx); err != nil {
				return err
			}
			continue
		}
		sequence := parent.AdjustmentSequence
		for _, l := range lines {
			amount := exposure[l.OrderID]
			if !amount.IsPositive() {
				continue
			}
			sequence++
			b := client.FinanceCommissionAdjustment.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(org).SetCommissionID(parent.ID).SetOrderID(l.OrderID).SetAdjustmentNo(fmt.Sprintf("%s-ADJ%03d", parent.CommissionNo, sequence)).SetIdempotencyKey(economicAdjustmentKey(trigger, parent.ID, l.OrderID)).SetCommissionNo(parent.CommissionNo).SetOrderNo(l.OrderNo).SetEmployeeID(parent.EmployeeID).SetEmployeeName(parent.EmployeeName).SetDirection(adjustment.DirectionDECREASE).SetStatus(adjustment.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor).SetBaseCurrency(parent.BaseCurrency).SetAmount(amount.StringFixed(8)).SetReason("真实结算来源反转，按实际已付余额追回").SetVersion(1)
			if trigger.Kind == "VERIFICATION_REVERSE" {
				b.SetSourceType(adjustment.SourceTypeVERIFICATION_REVERSAL).SetSourceVerificationID(trigger.ID)
			} else {
				b.SetSourceType(adjustment.SourceTypeNETTING_REVERSAL).SetSourceNettingID(trigger.ID)
			}
			if _, err := b.Save(ctx); err != nil {
				return err
			}
		}
		if sequence != parent.AdjustmentSequence {
			if err := client.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).AddVersion(1).Exec(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
