package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// attachOrderStatusRecords 只在订单范围校验成功后读取该订单事件，不扩大列表响应。
// 同一维度、同一目标状态只保留最近一次记录；停用员工仍须显示真实姓名。
func attachOrderStatusRecords(ctx context.Context, client *ent.Client, order *biz.Order) error {
	events, err := client.OrderLifecycleEvent.Query().
		Where(eventent.OrderIDEQ(order.ID), eventent.DimensionIn(eventent.DimensionFLOW, eventent.DimensionTERMINATION, eventent.DimensionCLOSURE)).
		Order(eventent.ByChangedAt(entsql.OrderDesc()), eventent.ByID(entsql.OrderDesc())).All(ctx)
	if err != nil {
		return err
	}
	latest := make([]*ent.OrderLifecycleEvent, 0)
	seen := make(map[string]bool)
	operatorIDs := make([]uuid.UUID, 0)
	for _, event := range events {
		key := string(event.Dimension) + ":" + event.ToStatus
		if seen[key] {
			continue
		}
		seen[key] = true
		latest = append(latest, event)
		if event.OperatorID != nil {
			operatorIDs = append(operatorIDs, *event.OperatorID)
		}
	}
	names := make(map[uuid.UUID]string)
	if len(operatorIDs) > 0 {
		users, queryErr := client.User.Query().Where(userent.IDIn(operatorIDs...)).All(ctx)
		if queryErr != nil {
			return queryErr
		}
		for _, user := range users {
			names[user.ID] = user.DisplayName
		}
	}
	for _, event := range latest {
		record := &biz.OrderStatusRecord{Dimension: string(event.Dimension), Status: event.ToStatus, ChangedAt: event.ChangedAt}
		if event.OperatorID != nil {
			record.OperatorName = names[*event.OperatorID]
		}
		if event.Reason != nil {
			record.Reason = *event.Reason
		}
		order.StatusRecords = append(order.StatusRecords, record)
	}
	return nil
}
