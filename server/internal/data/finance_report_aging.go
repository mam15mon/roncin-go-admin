package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func agingReportSource(org uuid.UUID, f biz.FinanceReportFilter, today string) (string, []any) {
	q := reportSQL{}
	q.where("b.organization_id = ", org)
	q.conditions = append(q.conditions, "b.status='CONFIRMED'")
	if f.Direction != "" {
		q.where("b.direction = ", string(f.Direction))
	}
	if f.Currency != "" {
		q.where("b.currency = ", f.Currency)
	}
	if f.SettlementPartyID != nil {
		q.where("b.settlement_party_id = ", *f.SettlementPartyID)
	}
	if f.Keyword != "" {
		p := q.arg("%" + strings.ToLower(f.Keyword) + "%")
		q.conditions = append(q.conditions, "(LOWER(b.bill_no) LIKE "+p+" OR LOWER(b.settlement_party_name) LIKE "+p+")")
	}
	date := q.arg(today)
	bucketFilter := "TRUE"
	if f.Bucket != "" {
		bucketFilter = "bucket=" + q.arg(f.Bucket)
	}
	// 两种清偿各自按账单聚合，避免核销×对冲的直接联接把金额相乘。
	source := `WITH bills AS (SELECT b.* FROM finance_bills b WHERE ` + q.sql() + `),
 verified AS (SELECT a.bill_id,SUM(a.amount) AS amount FROM finance_verification_allocations a JOIN finance_verifications v ON v.id=a.verification_id JOIN bills b ON b.id=a.bill_id WHERE a.active=TRUE AND v.status='ACTIVE' AND v.organization_id=b.organization_id GROUP BY a.bill_id),
 netted AS (SELECT a.bill_id,SUM(a.amount) AS amount FROM finance_netting_allocations a JOIN finance_nettings n ON n.id=a.netting_id JOIN bills b ON b.id=a.bill_id WHERE a.active=TRUE AND n.status='CONFIRMED' AND n.organization_id=b.organization_id GROUP BY a.bill_id),
 balances AS (SELECT b.*,COALESCE(v.amount,0)+COALESCE(n.amount,0) AS settled_amount,b.total_amount-COALESCE(v.amount,0)-COALESCE(n.amount,0) AS balance,
 NULLIF(b.due_date,'')::date AS due_day FROM bills b LEFT JOIN verified v ON v.bill_id=b.id LEFT JOIN netted n ON n.bill_id=b.id),
 aged AS (SELECT *,` + reportBankRound("balance*exchange_rate") + ` AS base_balance,
 CASE WHEN due_day IS NULL THEN NULL ELSE GREATEST(` + date + `::date-due_day,0) END AS overdue_days,
 CASE WHEN due_day IS NULL THEN 'UNDATED' WHEN due_day>=` + date + `::date THEN 'NOT_DUE' WHEN ` + date + `::date-due_day<=30 THEN 'DAYS_1_30' WHEN ` + date + `::date-due_day<=60 THEN 'DAYS_31_60' WHEN ` + date + `::date-due_day<=90 THEN 'DAYS_61_90' ELSE 'OVER_90' END AS bucket FROM balances WHERE balance<>0),
 matched AS (SELECT * FROM aged WHERE ` + bucketFilter + `)`
	return source, q.args
}
func reportAgingSums(amount string) string {
	sums := []string{}
	for _, pair := range []struct{ bucket, alias string }{{"NOT_DUE", "notdue"}, {"DAYS_1_30", "days130"}, {"DAYS_31_60", "days3160"}, {"DAYS_61_90", "days6190"}, {"OVER_90", "over90"}, {"UNDATED", "undated"}} {
		sums = append(sums, `COALESCE(SUM(CASE WHEN bucket='`+pair.bucket+`' THEN `+amount+` ELSE 0 END),0) AS `+pair.alias)
	}
	return strings.Join(sums, ",") + `,SUM(` + amount + `) AS total,COUNT(*) AS billcount`
}
func (r *financeReportRepo) Aging(ctx context.Context, org uuid.UUID, f biz.FinanceReportFilter, export bool, now time.Time) (*biz.AgingReportResult, error) {
	out := &biz.AgingReportResult{Items: []*biz.AgingReportRow{}, AsOfDate: now.In(biz.ExchangeRateBusinessLocation()).Format("2006-01-02"), GeneratedAt: now.UTC()}
	err := r.data.withReadSnapshot(ctx, func(ctx context.Context) error {
		client, err := r.data.client(ctx)
		if err != nil {
			return err
		}
		source, args := agingReportSource(org, f, out.AsOfDate)
		count, err := financeReportScan[reportCountRow](ctx, client, source+` SELECT COUNT(*) AS total FROM matched`, args)
		if err != nil {
			return err
		}
		out.Total = count[0].Total
		if export && out.Total > biz.FinanceReportExportLimit {
			return biz.ErrFinanceReportExportLimit
		}
		out.NativeTotals, err = financeReportScan[biz.AgingReportBucketAmount](ctx, client, source+` SELECT direction,currency,`+reportAgingSums("balance")+` FROM matched GROUP BY direction,currency ORDER BY direction,currency`, args)
		if err != nil {
			return err
		}
		out.BaseTotals, err = financeReportScan[biz.AgingReportBucketAmount](ctx, client, source+` SELECT direction,base_currency AS currency,`+reportAgingSums("base_balance")+` FROM matched GROUP BY direction,base_currency ORDER BY direction,base_currency`, args)
		if err != nil {
			return err
		}
		limit, offset := reportLimit(f, export)
		rowArgs := append(append([]any{}, args...), limit, offset)
		out.Items, err = financeReportScan[*biz.AgingReportRow](ctx, client, source+` SELECT id,bill_no AS billno,direction,settlement_party_id AS settlementpartyid,settlement_party_name AS settlementpartyname,bill_date AS billdate,due_date AS duedate,currency,total_amount AS totalamount,settled_amount AS settledamount,balance,base_currency AS basecurrency,base_balance AS basebalance,bucket,overdue_days AS overduedays FROM matched ORDER BY due_day ASC NULLS LAST,id LIMIT $`+stringInt(len(args)+1)+` OFFSET $`+stringInt(len(args)+2), rowArgs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
