package data

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// loadCommissionEconomicProjection 的账务归属只来自统一的行清偿重放。
// override 是审批预览的有效叶子，历史链始终从完整费用事实读取。
func loadCommissionEconomicProjection(ctx context.Context, client *ent.Client, orderID uuid.UUID, override []*biz.OrderFee) (*biz.CommissionEconomicOrderProjection, error) {
	order, err := client.Order.Get(ctx, orderID)
	if err != nil {
		return nil, err
	}
	raw, err := client.OrderFee.Query().Where(fee.OrderIDEQ(orderID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(fee.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	all := make([]*biz.OrderFee, 0, len(raw))
	current := make([]*biz.OrderFee, 0, len(raw))
	for _, item := range raw {
		f, err := orderFeeToBiz(item)
		if err != nil {
			return nil, err
		}
		all = append(all, f)
		if biz.IsCurrentFee(f) {
			current = append(current, f)
		}
	}
	if override != nil {
		current = append([]*biz.OrderFee(nil), override...)
	}
	sort.Slice(current, func(i, j int) bool { return current[i].ID.String() < current[j].ID.String() })
	bills, err := client.FinanceBill.Query().Where(bill.HasLinesWith(billline.OrderIDEQ(orderID), billline.ActiveEQ(true)), bill.StatusIn(bill.StatusDRAFT, bill.StatusCONFIRMED)).WithLines().Order(bill.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(bills))
	bases := make(map[uuid.UUID]decimal.Decimal)
	for _, b := range bills {
		ids = append(ids, b.ID)
		if b.Purpose != bill.PurposeNORMAL {
			continue
		}
		for _, l := range b.Edges.Lines {
			if !l.Active || l.OrderID != orderID {
				continue
			}
			amount, err := decimalOf(l.BaseCurrencyAmount)
			if err != nil {
				return nil, err
			}
			bases[l.OrderFeeID] = bases[l.OrderFeeID].Add(amount)
		}
	}
	settlements, err := loadFinanceBillLineSettlements(ctx, client, ids)
	if err != nil {
		return nil, err
	}
	return biz.ProjectCommissionEconomics(orderID, order.FeeContentVersion, all, current, bases, settlements)
}
