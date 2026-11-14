package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func cashflowReportSource(org uuid.UUID, f biz.FinanceReportFilter) (string, []any) {
	q := reportSQL{}
	q.where("organization_id = ", org)
	q.conditions = append(q.conditions, "status='CONFIRMED'")
	if f.DateFrom != "" {
		q.where("transaction_date >= ", f.DateFrom)
	}
	if f.DateTo != "" {
		q.where("transaction_date <= ", f.DateTo)
	}
	if f.Direction != "" {
		q.where("direction = ", string(f.Direction))
	}
	if f.Currency != "" {
		q.where("currency = ", f.Currency)
	}
	if f.SettlementPartyID != nil {
		q.where("settlement_party_id = ", *f.SettlementPartyID)
	}
	if f.PaymentMethod != "" {
		q.where("payment_method = ", f.PaymentMethod)
	}
	if f.OurAccount != "" {
		q.where("our_account = ", f.OurAccount)
	}
	if f.Keyword != "" {
		p := q.arg("%" + strings.ToLower(f.Keyword) + "%")
		q.conditions = append(q.conditions, "(LOWER(flow_no) LIKE "+p+" OR LOWER(settlement_party_name) LIKE "+p+" OR LOWER(COALESCE(bank_reference_no,'')) LIKE "+p+")")
	}
	return `WITH matched AS (SELECT * FROM finance_cashflows WHERE ` + q.sql() + `)`, q.args
}
func reportCashSums(amount string) string {
	return `COALESCE(SUM(CASE WHEN direction='RECEIVABLE' THEN ` + amount + ` ELSE 0 END),0) AS receivableamount,COALESCE(SUM(CASE WHEN direction='PAYABLE' THEN ` + amount + ` ELSE 0 END),0) AS payableamount,COALESCE(SUM(CASE WHEN direction='RECEIVABLE' THEN ` + amount + ` ELSE -` + amount + ` END),0) AS netamount`
}

func (r *financeReportRepo) Cashflow(ctx context.Context, org uuid.UUID, f biz.FinanceReportFilter, export bool) (*biz.CashflowReportResult, error) {
	out := &biz.CashflowReportResult{Items: []*biz.CashflowReportRow{}, GeneratedAt: time.Now().UTC()}
	err := r.data.withReadSnapshot(ctx, func(ctx context.Context) error {
		client, err := r.data.client(ctx)
		if err != nil {
			return err
		}
		source, args := cashflowReportSource(org, f)
		count, err := financeReportScan[reportCountRow](ctx, client, source+` SELECT COUNT(*) AS total FROM matched`, args)
		if err != nil {
			return err
		}
		out.Total = count[0].Total
		if export && out.Total > biz.FinanceReportExportLimit {
			return biz.ErrFinanceReportExportLimit
		}
		out.NativeTotals, err = financeReportScan[biz.FinanceReportCashAmount](ctx, client, source+` SELECT currency,`+reportCashSums("amount")+` FROM matched GROUP BY currency ORDER BY currency`, args)
		if err != nil {
			return err
		}
		out.BaseTotals, err = financeReportScan[biz.FinanceReportCashAmount](ctx, client, source+` SELECT base_currency AS currency,`+reportCashSums("base_amount")+` FROM matched GROUP BY base_currency ORDER BY base_currency`, args)
		if err != nil {
			return err
		}
		limit, offset := reportLimit(f, export)
		rowArgs := append(append([]any{}, args...), limit, offset)
		out.Items, err = financeReportScan[*biz.CashflowReportRow](ctx, client, source+` SELECT id,flow_no AS flowno,direction,transaction_date AS transactiondate,settlement_party_id AS settlementpartyid,settlement_party_name AS settlementpartyname,currency,amount,base_currency AS basecurrency,base_amount AS baseamount,our_account AS ouraccount,payment_method AS paymentmethod,bank_reference_no AS bankreferenceno FROM matched ORDER BY transaction_date DESC,id LIMIT $`+stringInt(len(args)+1)+` OFFSET $`+stringInt(len(args)+2), rowArgs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
