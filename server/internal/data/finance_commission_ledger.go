package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// commissionListPredicates 构造提成列表筛选谓词，列表与导出复用同一实现。
func commissionListPredicates(org uuid.UUID, f biz.CommissionFilter) []predicate.FinanceCommission {
	return commissionListPredicatesScoped([]uuid.UUID{org}, f)
}
func commissionListPredicatesScoped(organizationIDs []uuid.UUID, f biz.CommissionFilter) []predicate.FinanceCommission {
	p := []predicate.FinanceCommission{commission.OrganizationIDIn(organizationIDs...)}
	if f.Keyword != "" {
		p = append(p, commission.Or(commission.CommissionNoContainsFold(f.Keyword), commission.EmployeeNameContainsFold(f.Keyword), commission.RuleNameContainsFold(f.Keyword)))
	}
	if f.Status != "" {
		p = append(p, commission.StatusEQ(commission.Status(f.Status)))
	}
	if f.CommissionDateFrom != "" {
		p = append(p, commission.CommissionDateGTE(f.CommissionDateFrom))
	}
	if f.CommissionDateTo != "" {
		p = append(p, commission.CommissionDateLTE(f.CommissionDateTo))
	}
	return p
}

func (r *commissionRepo) List(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	return r.ListScoped(ctx, []uuid.UUID{org}, f)
}
func (r *commissionRepo) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionFilter) (*biz.CommissionListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	q := client.FinanceCommission.Query().Where(commissionListPredicatesScoped(organizationIDs, f)...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := q.WithOrganization().WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Order(commission.ByCommissionDate(entsql.OrderDesc()), commission.ByCreatedAt(entsql.OrderDesc()), commission.ByID(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.CommissionListResult{Items: make([]*biz.FinanceCommission, 0, len(xs)), Total: int64(total)}
	for _, x := range xs {
		item, err := commissionWithLinesToBiz(x)
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

// Count 按列表同一谓词统计提成总数，供导出上限门禁使用。
func (r *commissionRepo) Count(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) (int64, error) {
	return r.CountScoped(ctx, []uuid.UUID{org}, f)
}

func (r *commissionRepo) CountScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionFilter) (int64, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return 0, err
	}
	total, err := client.FinanceCommission.Query().Where(commissionListPredicatesScoped(organizationIDs, f)...).Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(total), nil
}

// ExportBatch 按列表同谓词与稳定排序（commission_date DESC, created_at DESC,
// id DESC）分页读取一批提成；调整单随行加载，保证 CNY 调整与有效金额的
// 动态口径与列表一致。
func (r *commissionRepo) ExportBatch(ctx context.Context, org uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
	return r.ExportBatchScoped(ctx, []uuid.UUID{org}, f)
}

func (r *commissionRepo) ExportBatchScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.CommissionFilter) ([]*biz.FinanceCommission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	xs, err := client.FinanceCommission.Query().Where(commissionListPredicatesScoped(organizationIDs, f)...).WithOrganization().WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Order(commission.ByCommissionDate(entsql.OrderDesc()), commission.ByCreatedAt(entsql.OrderDesc()), commission.ByID(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*biz.FinanceCommission, 0, len(xs))
	for _, x := range xs {
		item, convertErr := commissionWithLinesToBiz(x)
		if convertErr != nil {
			return nil, convertErr
		}
		items = append(items, item)
	}
	return items, nil
}

// SaveExportAudit 在导出成功返回前持久化业务审计；导出为只读链路，不参与
// 共享事务，审计写入失败时由用例整体失败。
func (r *commissionRepo) SaveExportAudit(ctx context.Context, event *biz.AuditEvent) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	return writeAudit(ctx, client.AuditLog, event)
}

func (r *commissionRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCommission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommission.Query().Where(commission.OrganizationIDEQ(org), commission.IdempotencyKeyEQ(key)).WithOrganization().WithLines(func(q *ent.FinanceCommissionLineQuery) {
		q.Order(commissionline.ByOrderNo(), commissionline.ByOrderID())
	}).WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return commissionWithLinesToBiz(x)
}

func (r *commissionRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCommission, error) {
	return r.GetScoped(ctx, []uuid.UUID{org}, id)
}
func (r *commissionRepo) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCommission, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	x, err := client.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDIn(organizationIDs...)).WithOrganization().WithLines(func(q *ent.FinanceCommissionLineQuery) {
		q.Order(commissionline.ByOrderNo(), commissionline.ByOrderID())
	}).WithAdjustments(func(q *ent.FinanceCommissionAdjustmentQuery) {
		q.Order(adjustment.ByCreatedAt())
	}).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionNotFound, nil)
	}
	return commissionWithLinesToBiz(x)
}
