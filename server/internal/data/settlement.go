package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type settlementRepo struct {
	data *Data
}

func NewSettlementRepo(data *Data) biz.SettlementRepo {
	return &settlementRepo{data: data}
}

func (r *settlementRepo) ListFeeLedger(ctx context.Context, organizationID uuid.UUID, filter biz.FeeLedgerFilter) (*biz.FeeLedgerResult, error) {
	predicates := []predicate.OrderFee{orderfee.HasOrderWith(order.OrganizationIDEQ(organizationID))}
	if filter.Keyword != "" {
		predicates = append(predicates, orderfee.Or(
			orderfee.FeeCodeContainsFold(filter.Keyword),
			orderfee.FeeNameContainsFold(filter.Keyword),
			orderfee.HasOrderWith(order.OrderNoContainsFold(filter.Keyword)),
			orderfee.HasSettlementPartyWith(partner.LegalNameContainsFold(filter.Keyword)),
		))
	}
	if filter.BusinessType != "" {
		predicates = append(predicates, orderfee.HasOrderWith(order.BusinessTypeEQ(order.BusinessType(filter.BusinessType))))
	}
	if filter.Direction != "" {
		predicates = append(predicates, orderfee.DirectionEQ(orderfee.Direction(filter.Direction)))
	}
	if filter.Status != "" {
		predicates = append(predicates, orderfee.StatusEQ(orderfee.Status(filter.Status)))
	}
	if filter.SettlementPartyID != nil {
		predicates = append(predicates, orderfee.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.Currency != "" {
		predicates = append(predicates, orderfee.CurrencyEQ(filter.Currency))
	}
	if filter.ExpenseDateFrom != "" {
		predicates = append(predicates, orderfee.ExpenseDateGTE(filter.ExpenseDateFrom))
	}
	if filter.ExpenseDateTo != "" {
		predicates = append(predicates, orderfee.ExpenseDateLTE(filter.ExpenseDateTo))
	}
	baseQuery := r.data.db.OrderFee.Query().Where(predicates...)
	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	all, err := baseQuery.Clone().WithSettlementParty().WithOrder().All(ctx)
	if err != nil {
		return nil, err
	}
	summary := biz.FeeLedgerSummary{ReceivableBaseAmount: decimal.Zero, PayableBaseAmount: decimal.Zero}
	for _, item := range all {
		if item.Status == orderfee.StatusCANCELLED {
			continue
		}
		amount, parseErr := decimal.NewFromString(item.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		summary.ActiveCount++
		summary.BaseCurrency = item.BaseCurrency
		if item.Direction == orderfee.DirectionRECEIVABLE {
			summary.ReceivableBaseAmount = summary.ReceivableBaseAmount.Add(amount)
		} else {
			summary.PayableBaseAmount = summary.PayableBaseAmount.Add(amount)
		}
	}
	summary.ProfitBaseAmount = summary.ReceivableBaseAmount.Sub(summary.PayableBaseAmount)
	items, err := baseQuery.Clone().
		WithSettlementParty().WithOrder().
		Order(orderfee.ByExpenseDate(entsql.OrderDesc()), orderfee.ByCreatedAt(entsql.OrderDesc()), orderfee.ByID(entsql.OrderDesc())).
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FeeLedgerResult{Items: make([]*biz.FeeLedgerItem, 0, len(items)), Total: int64(total), Summary: summary}
	for _, item := range items {
		fee, convertErr := orderFeeToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		businessOrder, edgeErr := item.Edges.OrderOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		result.Items = append(result.Items, &biz.FeeLedgerItem{Fee: fee, OrderNo: businessOrder.OrderNo, Business: string(businessOrder.BusinessType)})
	}
	return result, nil
}

var _ biz.SettlementRepo = (*settlementRepo)(nil)
