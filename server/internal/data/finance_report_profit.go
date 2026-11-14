package data

import (
	"context"
	"sort"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// 权威航期与订单列表共用表达式：AE 来自最新主单关联，其他品类用订单同步投影。
func reportBusinessDateSQL() string {
	selector := entsql.Dialect(dialect.Postgres).Select().From(entsql.Dialect(dialect.Postgres).Table("orders").As("o"))
	expression := func(field string) string {
		value, _ := entsql.P(func(b *entsql.Builder) {
			b.SetDialect(dialect.Postgres)
			writeOrderTransportTimeExpression(b, selector, field)
		}).Query()
		return value
	}
	zone := biz.ExchangeRateBusinessLocation().String()
	return "CASE WHEN o.trade_direction = 'export' THEN (" + expression("etd") + " AT TIME ZONE '" + zone + "')::date::text WHEN o.trade_direction = 'import' THEN (" + expression("eta") + " AT TIME ZONE '" + zone + "')::date::text END"
}

func profitReportSource(org uuid.UUID, f biz.FinanceReportFilter) (string, []any, string) {
	q := reportSQL{}
	q.where("o.organization_id = ", org)
	if f.BusinessType != "" {
		q.where("o.business_type = ", f.BusinessType)
	}
	if f.CustomerID != nil {
		q.where("o.customer_id = ", *f.CustomerID)
	}
	if f.Keyword != "" {
		p := q.arg("%" + strings.ToLower(f.Keyword) + "%")
		q.conditions = append(q.conditions, "(LOWER(o.order_no) LIKE "+p+" OR LOWER(c.legal_name) LIKE "+p+")")
	}
	for _, item := range []struct {
		role string
		id   *uuid.UUID
	}{{"SALES", f.SalesID}, {"OPERATOR", f.OperatorID}, {"CUSTOMER_SERVICE", f.CustomerServiceID}} {
		if item.id != nil {
			q.conditions = append(q.conditions, "EXISTS (SELECT 1 FROM order_personnels p WHERE p.order_id=o.id AND p.organization_id=o.organization_id AND p.role="+q.arg(item.role)+" AND p.user_id="+q.arg(*item.id)+")")
		}
	}
	dates := []string{"TRUE"}
	if f.DateFrom != "" {
		dates = append(dates, "business_date >= "+q.arg(f.DateFrom))
	}
	if f.DateTo != "" {
		dates = append(dates, "business_date <= "+q.arg(f.DateTo))
	}
	// 每费用先应用与 OrderFee.BaseCurrencyNetAmount 一致的半偶舍入，再做完整筛选聚合。
	// 领域先由 decimal.Div 按 DivisionPrecision 半离零，再 RoundBank(8)；
	// 省略中间舍入会令极接近八位中点的快照与费用/提成投影相差一个最小金额单位。
	net := reportBankRound("ROUND(f.base_currency_amount * f.net_amount / NULLIF(f.total_amount,0)," + stringInt(decimal.DivisionPrecision) + ")")
	source := `WITH report_orders AS (
 SELECT o.id AS order_id,o.order_no,o.business_type,o.customer_id,c.legal_name AS customer_name,` + reportBusinessDateSQL() + ` AS business_date
 FROM orders o JOIN partners c ON c.id=o.customer_id AND c.organization_id=o.organization_id WHERE ` + q.sql() + `
 ), report_fees AS (
 SELECT ro.*,f.id AS fee_id,f.direction,f.currency,f.base_currency,f.total_amount,f.net_amount,
 f.base_currency_amount AS gross_base,COALESCE(` + net + `,0) AS net_base
 FROM report_orders ro JOIN order_fees f ON f.order_id=ro.order_id
 WHERE f.economic_effect='INCREASE' AND f.superseded_by_request_id IS NULL
 ), matched AS (SELECT * FROM report_fees WHERE ` + strings.Join(dates, " AND ") + `)`
	return source, q.args, `SELECT count(DISTINCT order_id) AS total FROM report_fees WHERE business_date IS NULL`
}

const reportProfitSums = `COALESCE(SUM(CASE WHEN direction='RECEIVABLE' THEN gross_base ELSE 0 END),0) AS receivableamount,
 COALESCE(SUM(CASE WHEN direction='PAYABLE' THEN gross_base ELSE 0 END),0) AS payableamount,
 COALESCE(SUM(CASE WHEN direction='RECEIVABLE' THEN gross_base ELSE -gross_base END),0) AS profitamount,
 COALESCE(SUM(CASE WHEN direction='RECEIVABLE' THEN net_base ELSE 0 END),0) AS netreceivableamount,
 COALESCE(SUM(CASE WHEN direction='PAYABLE' THEN net_base ELSE 0 END),0) AS netpayableamount,
 COALESCE(SUM(CASE WHEN direction='RECEIVABLE' THEN net_base ELSE -net_base END),0) AS netprofitamount`

type reportCountRow struct{ Total int64 }
type profitCountRow struct{ Total, OrderCount, FeeCount int64 }
type profitMonthRow struct {
	biz.ProfitReportAmount
	Month string
}

func (r *financeReportRepo) Profit(ctx context.Context, org uuid.UUID, f biz.FinanceReportFilter, export bool) (*biz.ProfitReportResult, error) {
	out := &biz.ProfitReportResult{Items: []*biz.ProfitReportRow{}, GeneratedAt: time.Now().UTC()}
	err := r.data.withReadSnapshot(ctx, func(ctx context.Context) error {
		client, err := r.data.client(ctx)
		if err != nil {
			return err
		}
		source, args, undated := profitReportSource(org, f)
		grouped := `, grouped AS (SELECT order_id,order_no,business_type,business_date,customer_id,customer_name,base_currency,COUNT(*) AS feecount,` + reportProfitSums + ` FROM matched GROUP BY order_id,order_no,business_type,business_date,customer_id,customer_name,base_currency)`
		counts, err := financeReportScan[profitCountRow](ctx, client, source+grouped+` SELECT COUNT(*) AS total,COUNT(DISTINCT order_id) AS ordercount,COALESCE(SUM(feecount),0) AS feecount FROM grouped`, args)
		if err != nil {
			return err
		}
		out.Total, out.OrderCount, out.FeeCount = counts[0].Total, counts[0].OrderCount, counts[0].FeeCount
		if export && out.Total > biz.FinanceReportExportLimit {
			return biz.ErrFinanceReportExportLimit
		}
		missing, err := financeReportScan[reportCountRow](ctx, client, source+" "+undated, args)
		if err != nil {
			return err
		}
		out.UndatedOrderCount = missing[0].Total
		out.Totals, err = financeReportScan[biz.ProfitReportAmount](ctx, client, source+` SELECT base_currency AS basecurrency,`+reportProfitSums+` FROM matched GROUP BY base_currency ORDER BY base_currency`, args)
		if err != nil {
			return err
		}
		out.NativeTotals, err = financeReportScan[biz.FinanceReportNativeAmount](ctx, client, source+` SELECT direction,currency,SUM(total_amount) AS amount,SUM(net_amount) AS netamount FROM matched GROUP BY direction,currency ORDER BY direction,currency`, args)
		if err != nil {
			return err
		}
		months, err := financeReportScan[profitMonthRow](ctx, client, source+` SELECT COALESCE(SUBSTRING(business_date,1,7),'UNDATED') AS month,base_currency AS basecurrency,`+reportProfitSums+` FROM matched GROUP BY SUBSTRING(business_date,1,7),base_currency ORDER BY month,base_currency`, args)
		if err != nil {
			return err
		}
		out.Months = completeProfitReportMonths(months, out.Totals, f)
		limit, offset := reportLimit(f, export)
		rowArgs := append(append([]any{}, args...), limit, offset)
		// 岗位为单订单唯一角色子查询，不与费用直接多表 join，避免一单多人放大经营金额。
		personnel := func(role string) string {
			return `(SELECT u.display_name FROM order_personnels p JOIN users u ON u.id=p.user_id WHERE p.order_id=g.order_id AND p.role='` + role + `' LIMIT 1)`
		}
		out.Items, err = financeReportScan[*biz.ProfitReportRow](ctx, client, source+grouped+` SELECT g.order_id::text||':'||g.base_currency AS id,g.order_id AS orderid,g.order_no AS orderno,g.business_type AS businesstype,g.business_date AS businessdate,g.customer_id AS customerid,g.customer_name AS customername,
  COALESCE(`+personnel("SALES")+`,'') AS salesname,COALESCE(`+personnel("OPERATOR")+`,'') AS operatorname,COALESCE(`+personnel("CUSTOMER_SERVICE")+`,'') AS customerservicename,
  g.base_currency AS basecurrency,g.receivableamount,g.payableamount,g.profitamount,g.netreceivableamount,g.netpayableamount,g.netprofitamount,g.feecount
  FROM grouped g ORDER BY g.business_date DESC NULLS LAST,g.order_id,g.base_currency LIMIT $`+stringInt(len(args)+1)+` OFFSET $`+stringInt(len(args)+2), rowArgs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func completeProfitReportMonths(rows []profitMonthRow, totals []biz.ProfitReportAmount, f biz.FinanceReportFilter) []biz.ProfitReportMonth {
	byMonth := map[string]map[string]biz.ProfitReportAmount{}
	var first, last string
	for _, row := range rows {
		if byMonth[row.Month] == nil {
			byMonth[row.Month] = map[string]biz.ProfitReportAmount{}
		}
		byMonth[row.Month][row.BaseCurrency] = row.ProfitReportAmount
		if row.Month != "UNDATED" {
			if first == "" || row.Month < first {
				first = row.Month
			}
			if row.Month > last {
				last = row.Month
			}
		}
	}
	if f.DateFrom != "" {
		first = f.DateFrom[:7]
	}
	if f.DateTo != "" {
		last = f.DateTo[:7]
	}
	keys := []string{}
	if first != "" && last != "" {
		start, _ := time.Parse("2006-01", first)
		end, _ := time.Parse("2006-01", last)
		for t := start; !t.After(end); t = t.AddDate(0, 1, 0) {
			keys = append(keys, t.Format("2006-01"))
		}
	}
	if byMonth["UNDATED"] != nil {
		keys = append(keys, "UNDATED")
	}
	sort.Strings(keys)
	result := make([]biz.ProfitReportMonth, 0, len(keys))
	for _, month := range keys {
		item := biz.ProfitReportMonth{Month: month, Amounts: make([]biz.ProfitReportAmount, 0, len(totals))}
		for _, total := range totals {
			amount, ok := byMonth[month][total.BaseCurrency]
			if !ok {
				amount.BaseCurrency = total.BaseCurrency
			}
			item.Amounts = append(item.Amounts, amount)
		}
		result = append(result, item)
	}
	return result
}
