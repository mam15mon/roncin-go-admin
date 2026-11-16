package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// Export 在同一只读快照中取得完整结果、资源摘要和海空单证投影。
// 不经 paginate，也不使用超大 pageSize 规避常规列表上限。
func (r *orderRepo) Export(ctx context.Context, scopes []biz.OrderOrganizationScope, options biz.OrderListOptions) (*biz.OrderExport, error) {
	result := &biz.OrderExport{Items: []*biz.Order{}}
	err := r.data.withReadSnapshot(ctx, func(snapshotCtx context.Context) error {
		client, err := r.data.client(snapshotCtx)
		if err != nil {
			return err
		}
		rows, err := withOrderEdges(orderListQuery(client, scopes, options)).Order(orderListSort(options)...).All(snapshotCtx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			result.Items = append(result.Items, orderToBiz(row))
		}
		if err := attachOrderListResources(snapshotCtx, client, result.Items); err != nil {
			return err
		}
		if err := attachOrderDisplayNames(snapshotCtx, client, result.Items); err != nil {
			return err
		}
		if err := attachOrderFeeReviewProjections(snapshotCtx, client, result.Items); err != nil {
			return err
		}
		byOrganization := make(map[uuid.UUID][]*biz.Order)
		for _, item := range result.Items {
			if item.BusinessType == biz.OrderBusinessSE || item.BusinessType == biz.OrderBusinessSI {
				byOrganization[item.OrganizationID] = append(byOrganization[item.OrganizationID], item)
			}
		}
		for organizationID, items := range byOrganization {
			ids := make([]uuid.UUID, 0, len(items))
			for _, item := range items {
				ids = append(ids, item.ID)
			}
			masters, err := NewSeaMasterBillRepo(r.data).GetSummariesByOrderIDs(snapshotCtx, organizationID, ids)
			if err != nil {
				return err
			}
			documents, err := NewSeaDocumentRepo(r.data).GetSummariesByOrderIDs(snapshotCtx, organizationID, ids)
			if err != nil {
				return err
			}
			for _, item := range items {
				item.SeaMasterBill = masters[item.ID]
				if summary, ok := documents[item.ID]; ok {
					item.SeaDocumentSummary = summary
					item.SeaDocumentStructure = &summary.DocumentStructure
					item.SeaDocumentLinkVersion = &summary.LinkVersion
				}
			}
		}
		result.GeneratedAt = time.Now()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
