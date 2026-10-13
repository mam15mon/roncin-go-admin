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
	"strings"
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

// validateInvoiceEconomicSnapshot 签发时重算原草稿快照，经济内容改变必须取消后重建。
func validateInvoiceEconomicSnapshot(bills []*ent.FinanceBill, invoice *biz.FinanceInvoice) error {
	if len(bills) != len(invoice.Links) {
		return biz.ErrFinanceInvoiceBillInvalid
	}
	links := make(map[uuid.UUID]*biz.FinanceInvoiceBill)
	for _, l := range invoice.Links {
		links[l.BillID] = l
	}
	type amount struct {
		total, net, tax decimal.Decimal
		count           int
	}
	expected := make(map[string]amount)
	total, net, tax := decimal.Zero, decimal.Zero, decimal.Zero
	for _, raw := range bills {
		if raw.Purpose != "NORMAL" || raw.Status != "CONFIRMED" {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		b, err := invoiceableFinanceBill(raw)
		if err != nil {
			return err
		}
		l := links[b.ID]
		if l == nil || !b.InvoiceableAmount.IsPositive() || !l.Amount.Equal(*b.InvoiceableAmount) || !l.TaxAmount.Equal(*b.InvoiceableTaxAmount) {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		expectedIDs := make(map[uuid.UUID]bool)
		for _, line := range b.Lines {
			expectedIDs[line.OrderFeeID] = true
		}
		if len(l.SourceFeeIDs) != len(expectedIDs) {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		for _, id := range l.SourceFeeIDs {
			if !expectedIDs[id] {
				return biz.ErrFinanceInvoiceBillInvalid
			}
			delete(expectedIDs, id)
		}
		total = total.Add(*b.InvoiceableAmount)
		net = net.Add(*b.InvoiceableNetAmount)
		tax = tax.Add(*b.InvoiceableTaxAmount)
		for _, line := range b.Lines {
			if line.TaxRate == nil {
				return biz.ErrFinanceInvoiceBillInvalid
			}
			key := strings.Join([]string{line.FeeCode, line.FeeName, line.TaxRate.StringFixed(4), line.Currency}, "\x00")
			a := expected[key]
			a.total = a.total.Add(line.TotalAmount)
			a.net = a.net.Add(line.NetAmount)
			a.tax = a.tax.Add(line.TaxAmount)
			a.count++
			expected[key] = a
		}
	}
	if !total.Equal(invoice.TotalAmount) || !net.Equal(invoice.NetAmount) || !tax.Equal(invoice.TaxAmount) || len(expected) != len(invoice.Lines) {
		return biz.ErrFinanceInvoiceBillInvalid
	}
	for _, l := range invoice.Lines {
		key := strings.Join([]string{l.ItemCode, l.ItemName, l.TaxRate.StringFixed(4), l.Currency}, "\x00")
		a, ok := expected[key]
		if !ok || !a.total.Equal(l.TotalAmount) || !a.net.Equal(l.NetAmount) || !a.tax.Equal(l.TaxAmount) || a.count != l.SourceLineCount {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		delete(expected, key)
	}
	return nil
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
