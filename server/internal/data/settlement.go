package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
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
			orderfee.HasOrderWith(order.HasCustomerWith(partner.LegalNameContainsFold(filter.Keyword))),
			orderfee.HasFinanceBillLinesWith(
				financebillline.ActiveEQ(true),
				financebillline.HasBillWith(
					financebill.StatusNEQ(financebill.StatusCANCELLED),
					financebill.BillNoContainsFold(filter.Keyword),
				),
			),
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
	if filter.CustomerID != nil {
		predicates = append(predicates, orderfee.HasOrderWith(order.CustomerIDEQ(*filter.CustomerID)))
	}
	if filter.Currency != "" {
		predicates = append(predicates, orderfee.CurrencyEQ(filter.Currency))
	}
	if filter.BillNo != "" {
		predicates = append(predicates, orderfee.HasFinanceBillLinesWith(
			financebillline.ActiveEQ(true),
			financebillline.HasBillWith(
				financebill.StatusNEQ(financebill.StatusCANCELLED),
				financebill.BillNoContainsFold(filter.BillNo),
			),
		))
	}
	if filter.ExpenseDateFrom != "" {
		predicates = append(predicates, orderfee.ExpenseDateGTE(filter.ExpenseDateFrom))
	}
	if filter.ExpenseDateTo != "" {
		predicates = append(predicates, orderfee.ExpenseDateLTE(filter.ExpenseDateTo))
	}
	items, err := r.data.db.OrderFee.Query().Where(predicates...).
		WithSettlementParty().
		WithOrder(func(query *ent.OrderQuery) { query.WithCustomer() }).
		WithFinanceBillLines(func(lineQuery *ent.FinanceBillLineQuery) {
			lineQuery.
				Where(
					financebillline.ActiveEQ(true),
					financebillline.HasBillWith(financebill.StatusNEQ(financebill.StatusCANCELLED)),
				).
				WithBill(func(billQuery *ent.FinanceBillQuery) {
					billQuery.
						WithInvoiceLinks(func(linkQuery *ent.FinanceInvoiceBillQuery) {
							linkQuery.Where(
								financeinvoicebill.ActiveEQ(true),
								financeinvoicebill.HasInvoiceWith(financeinvoice.StatusEQ(financeinvoice.StatusISSUED)),
							)
						}).
						WithVerificationAllocations(func(allocationQuery *ent.FinanceVerificationAllocationQuery) {
							allocationQuery.Where(
								financeverificationallocation.ActiveEQ(true),
								financeverificationallocation.HasVerificationWith(financeverification.StatusEQ(financeverification.StatusACTIVE)),
							)
						})
				})
		}).
		Order(orderfee.ByExpenseDate(entsql.OrderDesc()), orderfee.ByCreatedAt(entsql.OrderDesc()), orderfee.ByID(entsql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]*biz.FeeLedgerItem, 0, len(items))
	summary := biz.FeeLedgerSummary{ReceivableBaseAmount: decimal.Zero, PayableBaseAmount: decimal.Zero}
	for _, item := range items {
		fee, convertErr := orderFeeToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		businessOrder, edgeErr := item.Edges.OrderOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		customer, edgeErr := businessOrder.Edges.CustomerOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		ledgerItem := &biz.FeeLedgerItem{
			Fee:               fee,
			OrderNo:           businessOrder.OrderNo,
			Business:          string(businessOrder.BusinessType),
			CustomerID:        customer.ID,
			CustomerName:      customer.LegalName,
			FinancialProgress: biz.FeeLedgerUnbilled,
		}
		if fee.Status == biz.OrderFeeCancelled {
			ledgerItem.FinancialProgress = ""
		}
		billLines, edgeErr := item.Edges.FinanceBillLinesOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if len(billLines) > 0 {
			bill, billErr := billLines[0].Edges.BillOrErr()
			if billErr != nil {
				return nil, billErr
			}
			billAmount, parseErr := decimal.NewFromString(bill.TotalAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			invoiceLinks, linkErr := bill.Edges.InvoiceLinksOrErr()
			if linkErr != nil {
				return nil, linkErr
			}
			allocations, allocationErr := bill.Edges.VerificationAllocationsOrErr()
			if allocationErr != nil {
				return nil, allocationErr
			}
			verifiedAmount := decimal.Zero
			for _, allocation := range allocations {
				amount, amountErr := decimal.NewFromString(allocation.Amount)
				if amountErr != nil {
					return nil, amountErr
				}
				verifiedAmount = verifiedAmount.Add(amount)
			}
			ledgerItem.BillNo = bill.BillNo
			ledgerItem.FinancialProgress = biz.ResolveFeeLedgerFinancialProgress(true, len(invoiceLinks) > 0, billAmount, verifiedAmount)
		}
		if filter.FinancialProgress != "" && ledgerItem.FinancialProgress != filter.FinancialProgress {
			continue
		}
		filtered = append(filtered, ledgerItem)
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
	start := (filter.Page - 1) * filter.PageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := min(start+filter.PageSize, len(filtered))
	result := &biz.FeeLedgerResult{
		Items:   filtered[start:end],
		Total:   int64(len(filtered)),
		Summary: summary,
	}
	return result, nil
}

var _ biz.SettlementRepo = (*settlementRepo)(nil)
