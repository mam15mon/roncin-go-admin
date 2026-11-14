package data

import (
	"context"
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type financeReportRepo struct{ data *Data }

func NewFinanceReportRepo(data *Data) biz.FinanceReportRepo { return &financeReportRepo{data: data} }

// 报表多层聚合使用参数化 SQL，统一经当前 Ent 客户端执行，自动加入只读一致快照。
// 不直连 sqlDB，避免分页行、汇总和图表跨连接取得不同财务事实。
func financeReportScan[T any](ctx context.Context, client *ent.Client, query string, args []any) ([]T, error) {
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]T, 0)
	if err := entsql.ScanSlice(rows, &items); err != nil {
		return nil, err
	}
	return items, nil
}

type reportSQL struct {
	args       []any
	conditions []string
}

func (q *reportSQL) arg(value any) string {
	q.args = append(q.args, value)
	return fmt.Sprintf("$%d", len(q.args))
}
func (q *reportSQL) where(expression string, value any) {
	q.conditions = append(q.conditions, expression+q.arg(value))
}
func (q *reportSQL) sql() string { return strings.Join(q.conditions, " AND ") }
func reportLimit(f biz.FinanceReportFilter, export bool) (int, int) {
	if export {
		return biz.FinanceReportExportLimit, 0
	}
	return f.PageSize, (f.Page - 1) * f.PageSize
}

// PostgreSQL round(numeric) 的中点远离零，领域 RoundBank 要求半偶；用整数商/余数保留一致精度。
// 入参仅由本层固定列名构造，筛选值一律占位参数，不能拼用户表达式。
func reportBankRound(expression string) string {
	scaled := "((" + expression + ") * 100000000)"
	return "(CASE WHEN ABS(" + scaled + " - TRUNC(" + scaled + ")) = 0.5 THEN (TRUNC(" + scaled + ") + CASE WHEN MOD(ABS(TRUNC(" + scaled + ")), 2) = 0 THEN 0 ELSE SIGN(" + scaled + ") END) / 100000000 ELSE ROUND((" + expression + "), 8) END)"
}

func (r *financeReportRepo) Options(ctx context.Context, org uuid.UUID, o biz.SelectorListOptions, personnel bool) (*biz.PagedList[*biz.FinanceReportOption], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if personnel {
		query, err := companyPersonnelQuery(ctx, client, org, o.Keyword)
		if err != nil {
			return nil, err
		}
		return paginate(ctx, func(c context.Context) (int, error) { return query.Clone().Count(c) }, func(c context.Context, offset, limit int) ([]*ent.User, error) {
			return query.Order(userent.ByDisplayName(), userent.ByID()).Offset(offset).Limit(limit).All(c)
		}, o.Page, o.PageSize, infalliblePageConverter(func(item *ent.User) *biz.FinanceReportOption {
			return &biz.FinanceReportOption{ID: item.ID, Name: item.DisplayName}
		}))
	}
	query := client.Partner.Query().Where(partnerent.OrganizationIDEQ(org))
	if o.Keyword != "" {
		query.Where(financePartyKeywordPredicate(o.Keyword))
	}
	return paginate(ctx, func(c context.Context) (int, error) { return query.Clone().Count(c) }, func(c context.Context, offset, limit int) ([]*ent.Partner, error) {
		return query.Order(partnerent.ByLegalName(), partnerent.ByID()).Offset(offset).Limit(limit).All(c)
	}, o.Page, o.PageSize, infalliblePageConverter(func(item *ent.Partner) *biz.FinanceReportOption {
		return &biz.FinanceReportOption{ID: item.ID, Name: item.LegalName}
	}))
}
