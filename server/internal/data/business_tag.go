package data

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	partnerlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourcepartner"
	tagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterprisetag"
	taggroupent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterprisetaggroup"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billtaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillenterprisetag"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordertaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	feetaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
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
		query.Where(resourceent.Or(
			resourceent.ShortNameContainsFold(keyword),
			resourceent.SearchKeywordsContainsFold(keyword),
			resourceent.HasTagWith(tagent.HasGroupWith(taggroupent.Or(
				taggroupent.NameContainsFold(keyword),
				taggroupent.SearchKeywordsContainsFold(keyword),
			))),
		))
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
	affected := 0
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if validateErr := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, true); validateErr != nil {
			return validateErr
		}
		if validateErr := validateOrderTagTargets(ctx, tx, organizationID, businessType, orderIDs); validateErr != nil {
			return validateErr
		}
		existing, queryErr := tx.OrderEnterpriseTag.Query().Where(ordertaglinkent.OrderIDIn(orderIDs...), ordertaglinkent.TagResourceIDIn(tagResourceIDs...)).All(ctx)
		if queryErr != nil {
			return queryErr
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
		if len(builders) > 0 {
			created, createErr := tx.OrderEnterpriseTag.CreateBulk(builders...).Save(ctx)
			if createErr != nil {
				return createErr
			}
			affected = len(created)
		}
		audit.Details["assigned_count"] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *businessTagRepo) RemoveOrderTags(ctx context.Context, organizationID uuid.UUID, businessType biz.OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	affected := 0
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if validateErr := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, false); validateErr != nil {
			return validateErr
		}
		if validateErr := validateOrderTagTargets(ctx, tx, organizationID, businessType, orderIDs); validateErr != nil {
			return validateErr
		}
		var deleteErr error
		affected, deleteErr = tx.OrderEnterpriseTag.Delete().Where(ordertaglinkent.OrganizationIDEQ(organizationID), ordertaglinkent.OrderIDIn(orderIDs...), ordertaglinkent.TagResourceIDIn(tagResourceIDs...)).Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		audit.Details["removed_count"] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
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

func (r *businessTagRepo) LoadOrderFeeTags(ctx context.Context, feeIDs []uuid.UUID) (map[uuid.UUID][]*biz.BusinessTagSummary, error) {
	links, err := r.data.db.OrderFeeEnterpriseTag.Query().
		Where(feetaglinkent.OrderFeeIDIn(feeIDs...)).
		WithTagResource(func(q *ent.EnterpriseResourceQuery) { q.WithTag(func(tq *ent.EnterpriseTagQuery) { tq.WithGroup() }) }).
		Order(feetaglinkent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]*biz.BusinessTagSummary, len(feeIDs))
	for _, link := range links {
		result[link.OrderFeeID] = append(result[link.OrderFeeID], enterpriseTagResourceToSummary(link.Edges.TagResource))
	}
	return result, nil
}

func (r *businessTagRepo) AssignOrderFeeTags(ctx context.Context, organizationID uuid.UUID, orderID *uuid.UUID, feeIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	affected := 0
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if validateErr := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, true); validateErr != nil {
			return validateErr
		}
		if validateErr := validateOrderFeeTagTargets(ctx, tx, organizationID, orderID, feeIDs); validateErr != nil {
			return validateErr
		}
		var upsertErr error
		affected, upsertErr = upsertOrderFeeTagLinks(ctx, tx, organizationID, feeIDs, tagResourceIDs, true)
		if upsertErr != nil {
			return upsertErr
		}
		audit.Details["assigned_count"] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *businessTagRepo) RemoveOrderFeeTags(ctx context.Context, organizationID uuid.UUID, orderID *uuid.UUID, feeIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	affected := 0
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if validateErr := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, false); validateErr != nil {
			return validateErr
		}
		if validateErr := validateOrderFeeTagTargets(ctx, tx, organizationID, orderID, feeIDs); validateErr != nil {
			return validateErr
		}
		var upsertErr error
		affected, upsertErr = upsertOrderFeeTagLinks(ctx, tx, organizationID, feeIDs, tagResourceIDs, false)
		if upsertErr != nil {
			return upsertErr
		}
		audit.Details["removed_count"] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}

// validateOrderFeeTagTargets 校验费用存在且通过所属订单归属当前组织；
// orderID 非空时（订单费用页入口）还要求全部费用属于该订单。
func validateOrderFeeTagTargets(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, orderID *uuid.UUID, feeIDs []uuid.UUID) error {
	fees, err := tx.OrderFee.Query().Where(feeent.IDIn(feeIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(fees) != len(feeIDs) {
		return biz.ErrBusinessTagInvalidArgument
	}
	orderIDSet := make(map[uuid.UUID]struct{}, len(fees))
	for _, fee := range fees {
		if orderID != nil && fee.OrderID != *orderID {
			return biz.ErrBusinessTagInvalidArgument
		}
		orderIDSet[fee.OrderID] = struct{}{}
	}
	orderIDs := make([]uuid.UUID, 0, len(orderIDSet))
	for id := range orderIDSet {
		orderIDs = append(orderIDs, id)
	}
	count, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(organizationID), orderent.IDIn(orderIDs...)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(orderIDs) {
		return biz.ErrBusinessTagInvalidArgument
	}
	return nil
}

func upsertOrderFeeTagLinks(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, feeIDs, tagResourceIDs []uuid.UUID, assign bool) (int, error) {
	if assign {
		existing, err := tx.OrderFeeEnterpriseTag.Query().Where(feetaglinkent.OrderFeeIDIn(feeIDs...), feetaglinkent.TagResourceIDIn(tagResourceIDs...)).All(ctx)
		if err != nil {
			return 0, err
		}
		existingKeys := make(map[[2]uuid.UUID]struct{}, len(existing))
		for _, link := range existing {
			existingKeys[[2]uuid.UUID{link.OrderFeeID, link.TagResourceID}] = struct{}{}
		}
		builders := make([]*ent.OrderFeeEnterpriseTagCreate, 0, len(feeIDs)*len(tagResourceIDs))
		for _, feeID := range feeIDs {
			for _, tagResourceID := range tagResourceIDs {
				if _, exists := existingKeys[[2]uuid.UUID{feeID, tagResourceID}]; exists {
					continue
				}
				builders = append(builders, tx.OrderFeeEnterpriseTag.Create().SetOrganizationID(organizationID).SetOrderFeeID(feeID).SetTagResourceID(tagResourceID))
			}
		}
		if len(builders) == 0 {
			return 0, nil
		}
		created, err := tx.OrderFeeEnterpriseTag.CreateBulk(builders...).Save(ctx)
		if err != nil {
			return 0, err
		}
		return len(created), nil
	}
	affected, err := tx.OrderFeeEnterpriseTag.Delete().Where(feetaglinkent.OrganizationIDEQ(organizationID), feetaglinkent.OrderFeeIDIn(feeIDs...), feetaglinkent.TagResourceIDIn(tagResourceIDs...)).Exec(ctx)
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *businessTagRepo) LoadFinanceBillTags(ctx context.Context, billIDs []uuid.UUID) (map[uuid.UUID][]*biz.BusinessTagSummary, error) {
	links, err := r.data.db.FinanceBillEnterpriseTag.Query().
		Where(billtaglinkent.FinanceBillIDIn(billIDs...)).
		WithTagResource(func(q *ent.EnterpriseResourceQuery) { q.WithTag(func(tq *ent.EnterpriseTagQuery) { tq.WithGroup() }) }).
		Order(billtaglinkent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]*biz.BusinessTagSummary, len(billIDs))
	for _, link := range links {
		result[link.FinanceBillID] = append(result[link.FinanceBillID], enterpriseTagResourceToSummary(link.Edges.TagResource))
	}
	return result, nil
}

func (r *businessTagRepo) AssignFinanceBillTags(ctx context.Context, organizationID uuid.UUID, billIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	affected := 0
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if validateErr := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, true); validateErr != nil {
			return validateErr
		}
		count, queryErr := tx.FinanceBill.Query().Where(billent.OrganizationIDEQ(organizationID), billent.IDIn(billIDs...)).Count(ctx)
		if queryErr != nil {
			return queryErr
		}
		if count != len(billIDs) {
			return biz.ErrBusinessTagInvalidArgument
		}
		existing, queryErr := tx.FinanceBillEnterpriseTag.Query().Where(billtaglinkent.FinanceBillIDIn(billIDs...), billtaglinkent.TagResourceIDIn(tagResourceIDs...)).All(ctx)
		if queryErr != nil {
			return queryErr
		}
		existingKeys := make(map[[2]uuid.UUID]struct{}, len(existing))
		for _, link := range existing {
			existingKeys[[2]uuid.UUID{link.FinanceBillID, link.TagResourceID}] = struct{}{}
		}
		builders := make([]*ent.FinanceBillEnterpriseTagCreate, 0, len(billIDs)*len(tagResourceIDs))
		for _, billID := range billIDs {
			for _, tagResourceID := range tagResourceIDs {
				if _, exists := existingKeys[[2]uuid.UUID{billID, tagResourceID}]; exists {
					continue
				}
				builders = append(builders, tx.FinanceBillEnterpriseTag.Create().SetOrganizationID(organizationID).SetFinanceBillID(billID).SetTagResourceID(tagResourceID))
			}
		}
		if len(builders) > 0 {
			created, createErr := tx.FinanceBillEnterpriseTag.CreateBulk(builders...).Save(ctx)
			if createErr != nil {
				return createErr
			}
			affected = len(created)
		}
		audit.Details["assigned_count"] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *businessTagRepo) RemoveFinanceBillTags(ctx context.Context, organizationID uuid.UUID, billIDs, tagResourceIDs []uuid.UUID, audit *biz.AuditEvent) (int, error) {
	affected := 0
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if validateErr := lockAndValidateTagResources(ctx, tx, organizationID, tagResourceIDs, false); validateErr != nil {
			return validateErr
		}
		count, queryErr := tx.FinanceBill.Query().Where(billent.OrganizationIDEQ(organizationID), billent.IDIn(billIDs...)).Count(ctx)
		if queryErr != nil {
			return queryErr
		}
		if count != len(billIDs) {
			return biz.ErrBusinessTagInvalidArgument
		}
		var deleteErr error
		affected, deleteErr = tx.FinanceBillEnterpriseTag.Delete().Where(billtaglinkent.OrganizationIDEQ(organizationID), billtaglinkent.FinanceBillIDIn(billIDs...), billtaglinkent.TagResourceIDIn(tagResourceIDs...)).Exec(ctx)
		if deleteErr != nil {
			return deleteErr
		}
		audit.Details["removed_count"] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return 0, err
	}
	return affected, nil
}
