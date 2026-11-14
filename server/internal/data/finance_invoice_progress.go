package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// 余额过滤在 COUNT/LIMIT 之前执行；只计算当前有效且尚未被足额登记的来源行。
func financeInvoiceBillHasRemainingPredicate() predicate.FinanceBill {
	return func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("EXISTS (SELECT 1 FROM finance_bill_lines l JOIN order_fees f ON f.id=l.order_fee_id WHERE l.bill_id=").Ident(s.C("id")).WriteString(" AND l.active=TRUE AND f.economic_effect='INCREASE' AND f.superseded_by_request_id IS NULL AND l.total_amount > COALESCE((SELECT sum(a.amount) FROM finance_invoice_fee_allocations a JOIN finance_invoice_bills ib ON ib.id=a.invoice_bill_id JOIN finance_invoices i ON i.id=ib.invoice_id WHERE a.bill_line_id=l.id AND ib.active=TRUE AND i.status='ISSUED'),0))")
		}))
	}
}
func (r *financeInvoiceRepo) enrichInvoiceRevisionNames(ctx context.Context, client *ent.Client, item *biz.FinanceInvoice) error {
	ids := make([]uuid.UUID, 0, len(item.Revisions))
	for _, revision := range item.Revisions {
		ids = append(ids, revision.ActorID)
	}
	if len(ids) == 0 {
		return nil
	}
	people, err := client.User.Query().Where(user.IDIn(ids...)).Select(user.FieldID, user.FieldDisplayName).All(ctx)
	if err != nil {
		return err
	}
	names := map[uuid.UUID]string{}
	for _, person := range people {
		names[person.ID] = person.DisplayName
	}
	for _, revision := range item.Revisions {
		revision.ActorName = names[revision.ActorID]
	}
	return nil
}
