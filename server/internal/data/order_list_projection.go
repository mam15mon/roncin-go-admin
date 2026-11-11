package data

import (
	"context"
	"sort"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attachmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	containerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	sharedent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
	allocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
)

// orderListSort 只消费用例校验后的白名单；时间按真实时刻排序，缺失值始终置后。
func orderListSort(options biz.OrderListOptions) []orderent.OrderOption {
	field := orderent.FieldCreatedAt
	switch options.SortBy {
	case "orderNo":
		field = orderent.FieldOrderNo
	case "etd":
		field = orderent.FieldEtd
	case "eta":
		field = orderent.FieldEta
	}
	ascending := options.SortOrder == "ascend"
	return []orderent.OrderOption{func(s *entsql.Selector) {
		s.OrderExpr(entsql.ExprFunc(func(b *entsql.Builder) {
			if field == orderent.FieldEtd || field == orderent.FieldEta {
				// 海运共享航程为权威来源；无对应时刻时沿用订单响应的本票值。
				b.WriteString("COALESCE((SELECT ").Ident("sea_sort_transport").WriteByte('.').Ident(field).
					WriteString(" FROM sea_master_bill_order_links AS sea_sort_link JOIN sea_transport_executions AS sea_sort_transport ON sea_sort_transport.id = sea_sort_link.transport_execution_id WHERE sea_sort_link.order_id = ").Ident(s.C(orderent.FieldID)).
					WriteString(" AND sea_sort_link.organization_id = ").Ident(s.C(orderent.FieldOrganizationID)).
					WriteString(" AND sea_sort_link.status = 'ACTIVE' LIMIT 1), ")
				writeOrderTransportTimeExpression(b, s, field).WriteString(")")
			} else {
				b.Ident(s.C(field))
			}
			if ascending {
				b.WriteString(" ASC NULLS LAST")
			} else {
				b.WriteString(" DESC NULLS LAST")
			}
		}))
	}, orderent.ByID()}
}

// attachOrderListResources 批量加载本页订单的轻量摘要，不读取文件资产正文或其他票的箱货。
func attachOrderListResources(ctx context.Context, client *ent.Client, orders []*biz.Order) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(orders))
	organizations := make([]uuid.UUID, 0, len(orders))
	byID := make(map[uuid.UUID]*biz.Order, len(orders))
	numbers := make(map[uuid.UUID]map[string]struct{}, len(orders))
	for _, item := range orders {
		ids = append(ids, item.ID)
		organizations = append(organizations, item.OrganizationID)
		byID[item.ID] = item
		numbers[item.ID] = make(map[string]struct{})
		count := 0
		item.AttachmentCount = &count
	}
	containers, err := client.OrderContainer.Query().
		Where(containerent.OrderIDIn(ids...), containerent.OrganizationIDIn(organizations...)).
		Select(containerent.FieldID, containerent.FieldOrderID, containerent.FieldOrganizationID, containerent.FieldContainerNo).All(ctx)
	if err != nil {
		return err
	}
	for _, container := range containers {
		if item := byID[container.OrderID]; item != nil && item.OrganizationID == container.OrganizationID {
			numbers[item.ID][container.ContainerNo] = struct{}{}
		}
	}
	allocations, err := client.SeaSharedContainerAllocation.Query().
		Where(allocationent.OrderIDIn(ids...), allocationent.OrganizationIDIn(organizations...)).
		Select(allocationent.FieldID, allocationent.FieldOrderID, allocationent.FieldOrganizationID, allocationent.FieldSharedContainerID).
		WithSharedContainer(func(q *ent.SeaSharedContainerQuery) {
			q.Select(sharedent.FieldID, sharedent.FieldOrganizationID, sharedent.FieldContainerNo, sharedent.FieldTransportExecutionID)
		}).All(ctx)
	if err != nil {
		return err
	}
	for _, allocation := range allocations {
		item := byID[allocation.OrderID]
		container := allocation.Edges.SharedContainer
		if item == nil || container == nil || item.OrganizationID != allocation.OrganizationID || container.OrganizationID != item.OrganizationID {
			continue
		}
		// 只认本票分配，且箱必须属于本票当前活动航程，不能将同主单其他票的箱带入。
		if item.SeaMasterBill != nil && item.SeaMasterBill.TransportExecutionID == container.TransportExecutionID {
			numbers[item.ID][container.ContainerNo] = struct{}{}
		}
	}
	var counts []struct {
		OrderID uuid.UUID `json:"order_id"`
		Count   int       `json:"count"`
	}
	err = client.OrderAttachment.Query().Where(attachmentent.OrderIDIn(ids...),
		attachmentent.HasOrderWith(orderent.OrganizationIDIn(organizations...))).
		GroupBy(attachmentent.FieldOrderID).Aggregate(ent.Count()).Scan(ctx, &counts)
	if err != nil {
		return err
	}
	for _, count := range counts {
		if item := byID[count.OrderID]; item != nil {
			*item.AttachmentCount = count.Count
		}
	}
	for _, item := range orders {
		item.ContainerNos = make([]string, 0, len(numbers[item.ID]))
		for number := range numbers[item.ID] {
			item.ContainerNos = append(item.ContainerNos, number)
		}
		sort.Strings(item.ContainerNos)
	}
	return nil
}
