package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
)

// orderRepo 按职责拆分实现：查询（order_query.go）、写入事务
// （order_write.go）、关联数据同步与校验（order_sync.go）、ent↔biz 转换
// （order_convert.go）；本文件只保留仓储锚点。
type orderRepo struct{ data *Data }

func NewOrderRepo(data *Data) biz.OrderRepo { return &orderRepo{data: data} }

var _ biz.OrderRepo = (*orderRepo)(nil)

// GetByIdempotencyKey 按组织与幂等键查既有订单；未命中返回 (nil, nil)，
// 与建账幂等查询口径一致。加载业务选择集合与岗位人员（创建人由哈希侧过滤），
// 供创建意图哈希比对使用。
func (r *orderRepo) GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.Order, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := withOrderEdges(client.Order.Query().Where(
		orderent.OrganizationIDEQ(organizationID),
		orderent.IdempotencyKeyEQ(idempotencyKey),
	)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	order := orderToBiz(item)
	personnelRows, err := client.OrderPersonnel.Query().
		Where(orderpersonnelent.OrderIDEQ(item.ID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	order.PersonnelAssignments = make([]*biz.OrderPersonnel, 0, len(personnelRows))
	for _, row := range personnelRows {
		order.PersonnelAssignments = append(order.PersonnelAssignments, &biz.OrderPersonnel{
			UserID:         row.UserID,
			OrganizationID: row.OrganizationID,
			Role:           biz.OrderPersonnelRole(row.Role),
		})
	}
	return order, nil
}
