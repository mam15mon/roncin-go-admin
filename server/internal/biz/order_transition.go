package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// OrderStatusChangedEvent 订单状态变更领域事件。
// P10 提醒/通知与后台任务入队等跨模块副作用统一通过
// handleOrderStatusChanged 挂载，禁止散落到 service 或 data 层。
type OrderStatusChangedEvent struct {
	OrganizationID uuid.UUID
	OrderID        uuid.UUID
	ActorID        uuid.UUID
	FromStatus     string
	ToStatus       string
	OccurredAt     time.Time
}

// TransitionStatus 校验并执行订单状态流转，成功后集中触发状态变更副作用。
func (uc *OrderUsecase) TransitionStatus(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedStatus, targetStatus, reason string) (*Order, error) {
	expectedStatus = strings.ToUpper(strings.TrimSpace(expectedStatus))
	targetStatus = strings.ToUpper(strings.TrimSpace(targetStatus))
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedStatus == "" || targetStatus == "" || targetStatus == "DRAFT" || expectedStatus == targetStatus || utf8.RuneCountInString(targetStatus) > 64 || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrOrderStatusInvalid
	}
	event := &OrderStatusChangedEvent{
		OrganizationID: organizationID,
		OrderID:        id,
		ActorID:        actorID,
		FromStatus:     expectedStatus,
		ToStatus:       targetStatus,
		OccurredAt:     time.Now().UTC(),
	}
	return uc.repo.TransitionStatus(ctx, organizationID, id, expectedStatus, targetStatus, strings.TrimSpace(reason), actorID, event)
}

// AuditEvent 返回与状态变更同事务持久化的审计事件。
func (event *OrderStatusChangedEvent) AuditEvent() *AuditEvent {
	return &AuditEvent{
		OrganizationID: &event.OrganizationID,
		UserID:         &event.ActorID,
		Action:         "order.status.transition",
		Result:         "success",
		Details: map[string]string{
			"order.id":    event.OrderID.String(),
			"from_status": event.FromStatus,
			"to_status":   event.ToStatus,
		},
	}
}
