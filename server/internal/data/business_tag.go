package data

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billtaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillenterprisetag"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	partnerlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourcepartner"
	feetaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordertaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
)

type businessTagRepo struct {
	data *Data
}

// NewBusinessTagRepo 返回订单、费用和账单共用的业务标签仓储。
func NewBusinessTagRepo(data *Data) biz.BusinessTagRepo {
	return &businessTagRepo{data: data}
}

func (r *businessTagRepo) ListTagOptions(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*biz.BusinessTagSummary, int64, error) {
	query := r.data.db.EnterpriseResource.Query().Where(resourceent.OrganizationIDEQ(organizationID), resourceent.ResourceTypeEQ(resourceent.ResourceTypeTAG), resourceent.EnabledEQ(true)).
		WithTag(func(q *ent.EnterpriseTagQuery) { q.WithGroup() })
	if keyword != "" {
		query.Where(resourceent.Or(resourceent.ShortNameContainsFold(keyword), resourceent.SearchKeywordsContainsFold(keyword)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.Order(resourceent.BySortOrder(), resourceent.ByCreatedAt(), resourceent.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return enterpriseTagResourcesToSummaries(items), int64(total), nil
}

func (r *businessTagRepo) LoadOrderTags(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]*biz.BusinessTagSummary, error) {
	links, err := r.data.db.OrderEnterpriseTag.Query().
		Where(ordertaglinkent.OrderIDIn(orderIDs...)).
		WithTagResource(func(q *ent.EnterpriseResourceQuery) { q.WithTag(func(tq *ent.EnterpriseTagQuery) { tq.WithGroup() }) }).
		Order(ordertaglinkent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]*biz.BusinessTagSummary, len(orderIDs))
	for _, link := range links {
		result[link.OrderID] = append(result[link.OrderID], enterpriseTagResourceToSummary(link.Edges.TagResource))
	}
	return result, nil
}

func (r *businessTagRepo) AssignOrderTags(ctx context.Context, organizationID uuid.UUID, businessType biz.OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return 0, err
	}
	if err := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, true); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := validateOrderTagTargets(ctx, tx, organizationID, businessType, orderIDs); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	existing, err := tx.OrderEnterpriseTag.Query().Where(ordertaglinkent.OrderIDIn(orderIDs...), ordertaglinkent.TagResourceIDIn(tagResourceIDs...)).All(ctx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	existingPairKeys := make(map[[2]uuid.UUID]struct{}, len(existing))
	for _, link := range existing {
		existingPairKeys[[2]uuid.UUID{link.OrderID, link.TagResourceID}] = struct{}{}
	}
	builders := make([]*ent.OrderEnterpriseTagCreate, 0, len(orderIDs)*len(tagResourceIDs))
	for _, orderID := range orderIDs {
		for _, tagResourceID := range tagResourceIDs {
			if _, exists := existingPairKeys[[2]uuid.UUID{orderID, tagResourceID}]; exists {
				continue
			}
			builders = append(builders, tx.OrderEnterpriseTag.Create().SetOrganizationID(organizationID).SetOrderID(orderID).SetTagResourceID(tagResourceID))
		}
	}
	affected := 0
	if len(builders) > 0 {
		created, err := tx.OrderEnterpriseTag.CreateBulk(builders...).Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		affected = len(created)
	}
	audit.Details["assigned_count"] = stringInt(affected)
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *businessTagRepo) RemoveOrderTags(ctx context.Context, organizationID uuid.UUID, businessType biz.OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return 0, err
	}
	if err := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, false); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := validateOrderTagTargets(ctx, tx, organizationID, businessType, orderIDs); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	affected, err := tx.OrderEnterpriseTag.Delete().Where(ordertaglinkent.OrganizationIDEQ(organizationID), ordertaglinkent.OrderIDIn(orderIDs...), ordertaglinkent.TagResourceIDIn(tagResourceIDs...)).Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	audit.Details["removed_count"] = stringInt(affected)
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *businessTagRepo) CountTagUsages(ctx context.Context, organizationID, tagResourceID uuid.UUID) (int, int, int, int, error) {
	partnerCount, err := r.data.db.EnterpriseResourcePartner.Query().Where(partnerlinkent.ResourceIDEQ(tagResourceID)).Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	orderCount, err := r.data.db.OrderEnterpriseTag.Query().Where(ordertaglinkent.OrganizationIDEQ(organizationID), ordertaglinkent.TagResourceIDEQ(tagResourceID)).Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	feeCount, err := r.data.db.OrderFeeEnterpriseTag.Query().Where(feetaglinkent.OrganizationIDEQ(organizationID), feetaglinkent.TagResourceIDEQ(tagResourceID)).Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	billCount, err := r.data.db.FinanceBillEnterpriseTag.Query().Where(billtaglinkent.OrganizationIDEQ(organizationID), billtaglinkent.TagResourceIDEQ(tagResourceID)).Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return partnerCount, orderCount, feeCount, billCount, nil
}

// lockAndValidateTagResources 按稳定顺序锁定标签资源并校验类型、组织与启用状态。
// requireEnabled 在添加场景要求启用；移除场景允许操作已停用标签。
func lockAndValidateTagResources(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, tagResourceIDs []uuid.UUID, requireEnabled bool) error {
	sorted := append([]uuid.UUID(nil), tagResourceIDs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].String() < sorted[j].String() })
	items, err := tx.EnterpriseResource.Query().Where(resourceent.IDIn(sorted...), resourceent.OrganizationIDEQ(organizationID)).Order(resourceent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(items) != len(sorted) {
		return biz.ErrBusinessTagInvalidArgument
	}
	for _, item := range items {
		if item.ResourceType != resourceent.ResourceType(biz.EnterpriseResourceTagType) || (requireEnabled && !item.Enabled) {
			return biz.ErrBusinessTagInvalidArgument
		}
	}
	return nil
}

func validateOrderTagTargets(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, businessType biz.OrderBusinessType, orderIDs []uuid.UUID) error {
	items, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(organizationID), orderent.IDIn(orderIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(items) != len(orderIDs) {
		return biz.ErrBusinessTagInvalidArgument
	}
	for _, item := range items {
		if biz.OrderBusinessType(item.BusinessType) != businessType {
			return biz.ErrBusinessTagInvalidArgument
		}
	}
	return nil
}

func enterpriseTagResourceToSummary(item *ent.EnterpriseResource) *biz.BusinessTagSummary {
	if item == nil {
		return nil
	}
	summary := &biz.BusinessTagSummary{ID: item.ID, Name: item.ShortName, Enabled: item.Enabled}
	if tag := item.Edges.Tag; tag != nil {
		summary.GroupID = tag.GroupID
		if group := tag.Edges.Group; group != nil {
			summary.GroupName = group.Name
			if group.Color != nil {
				summary.GroupColor = *group.Color
			}
		}
	}
	return summary
}

func enterpriseTagResourcesToSummaries(items []*ent.EnterpriseResource) []*biz.BusinessTagSummary {
	result := make([]*biz.BusinessTagSummary, 0, len(items))
	for _, item := range items {
		result = append(result, enterpriseTagResourceToSummary(item))
	}
	return result
}
