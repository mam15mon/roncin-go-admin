package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	invoicelink "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// invoiceableFinanceBill 保留账单历史总额，仅投影仍有效的普通经济行及可开票金额。
func invoiceableFinanceBill(item *ent.FinanceBill) (*biz.FinanceBill, error) {
	converted, err := financeBillToBiz(item)
	if err != nil {
		return nil, err
	}
	converted.Lines = nil
	total, net, tax := decimal.Zero, decimal.Zero, decimal.Zero
	for _, line := range item.Edges.Lines {
		if !line.Active {
			continue
		}
		original, err := line.Edges.OrderFeeOrErr()
		if err != nil {
			return nil, err
		}
		if !line.Active || original.EconomicEffect == fee.EconomicEffectDECREASE || original.SupersededByRequestID != nil {
			continue
		}
		l, err := financeBillLineToBiz(line)
		if err != nil {
			return nil, err
		}
		converted.Lines = append(converted.Lines, l)
		total = total.Add(l.TotalAmount)
		net = net.Add(l.NetAmount)
		tax = tax.Add(l.TaxAmount)
	}
	converted.InvoiceableAmount, converted.InvoiceableNetAmount, converted.InvoiceableTaxAmount = &total, &net, &tax
	return converted, nil
}

// checkCorrectionInvoiceAncestors 禁止原票未取消/红冲时直接对替代内容重复开票。
func checkCorrectionInvoiceAncestors(ctx context.Context, client *ent.Client, bills []*ent.FinanceBill) error {
	seen := make(map[uuid.UUID]bool)
	ids := make([]uuid.UUID, 0)
	for _, b := range bills {
		for _, line := range b.Edges.Lines {
			f, err := line.Edges.OrderFeeOrErr()
			if err != nil {
				return err
			}
			for f.SourceFeeID != nil {
				id := *f.SourceFeeID
				if seen[id] {
					break
				}
				seen[id] = true
				ids = append(ids, id)
				f, err = client.OrderFee.Get(ctx, id)
				if err != nil {
					return err
				}
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}
	links, err := client.FinanceInvoiceBill.Query().Where(invoicelink.ActiveEQ(true), invoicelink.HasBillWith(bill.HasLinesWith(billline.OrderFeeIDIn(ids...)))).All(ctx)
	if err != nil {
		return err
	}
	for _, link := range links {
		for _, id := range link.SourceFeeIds {
			if seen[id] {
				return biz.ErrFinanceInvoiceBillInvalid
			}
		}
	}
	return nil
}
