package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// lockBillEconomicOrders 在首个订单锁前展开全部公司内的父提成订单闭包，
// 统一排序加锁并复核；不得持有部分订单锁后再反序补锁关联订单。
func lockBillEconomicOrders(ctx context.Context, client *ent.Client, organizationIDs, seedIDs []uuid.UUID) error {
	seeds, err := client.Order.Query().Where(orderent.IDIn(seedIDs...), orderent.OrganizationIDIn(organizationIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(seeds) != len(uniqueSortedUUIDs(seedIDs)) {
		return biz.ErrFinanceBillFeeInvalid
	}
	byOrg := make(map[uuid.UUID][]uuid.UUID)
	for _, seed := range seeds {
		byOrg[seed.OrganizationID] = append(byOrg[seed.OrganizationID], seed.ID)
	}
	expand := func() ([]uuid.UUID, error) {
		var ids []uuid.UUID
		for org, roots := range byOrg {
			group, err := commissionEconomicExpandOrderIDs(ctx, client, org, roots)
			if err != nil {
				return nil, err
			}
			ids = append(ids, group...)
		}
		return uniqueSortedUUIDs(ids), nil
	}
	ids, err := expand()
	if err != nil {
		return err
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDIn(organizationIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(ids) {
		return biz.ErrFinanceBillFeeInvalid
	}
	fresh, err := expand()
	if err != nil {
		return err
	}
	if len(ids) != len(fresh) {
		return biz.ErrOrderStatusConflict
	}
	for i, id := range ids {
		if fresh[i] != id {
			return biz.ErrOrderStatusConflict
		}
	}
	return nil
}

// reconcileBillEconomics 在账单头及全部行完成写入后重算受影响订单。
// 账单操作使用独立触发标识，复用经济重分配调整，保留原计提及实付事实。
func reconcileBillEconomics(ctx context.Context, client *ent.Client, org uuid.UUID, orderIDs []uuid.UUID, actor uuid.UUID, trigger biz.CommissionEconomicTrigger) error {
	for _, orderID := range uniqueSortedUUIDs(orderIDs) {
		exists, err := client.FinanceCommissionLine.Query().Where(commissionline.OrganizationIDEQ(org), commissionline.OrderIDEQ(orderID), commissionline.HasCommissionWith(commission.StatusIn(commission.StatusCONFIRMED, commission.StatusPAID))).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		projection, err := loadCommissionEconomicProjection(ctx, client, orderID, nil)
		if err != nil {
			return err
		}
		plans, err := planCommissionEconomicRows(ctx, client, org, orderID, projection, true)
		if err != nil {
			return err
		}
		for _, plan := range plans {
			if plan.Changed && actor == uuid.Nil {
				return biz.ErrFeeCorrectionInvalid
			}
		}
		if err := applyCommissionEconomicPlans(ctx, client, org, actor, trigger, plans); err != nil {
			return err
		}
	}
	return nil
}

func billEconomicActor(audit *biz.AuditEvent) uuid.UUID {
	if audit != nil && audit.UserID != nil {
		return *audit.UserID
	}
	return uuid.Nil
}

func billEconomicOrderIDs(lines []*biz.FinanceBillLine) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.OrderID)
	}
	return uniqueSortedUUIDs(ids)
}
