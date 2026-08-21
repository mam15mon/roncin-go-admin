package biz

import (
	"context"
	"fmt"
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
	updated, err := uc.repo.TransitionStatus(ctx, organizationID, id, expectedStatus, targetStatus, strings.TrimSpace(reason), actorID)
	if err != nil {
		return nil, err
	}
	event := &OrderStatusChangedEvent{
		OrganizationID: organizationID,
		OrderID:        updated.ID,
		ActorID:        actorID,
		FromStatus:     expectedStatus,
		ToStatus:       targetStatus,
		OccurredAt:     time.Now().UTC(),
	}
	if err := uc.handleOrderStatusChanged(ctx, event); err != nil {
		return nil, err
	}
	return updated, nil
}

// handleOrderStatusChanged 是状态变更提交后副作用的唯一挂载点。
// 当前写入业务审计；P10 提醒任务入队、通知分发等后续副作用在此追加。
func (uc *OrderUsecase) handleOrderStatusChanged(ctx context.Context, event *OrderStatusChangedEvent) error {
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &event.OrganizationID,
		UserID:         &event.ActorID,
		Action:         "order.status.transition",
		Result:         "success",
		Details: map[string]string{
			"order.id":    event.OrderID.String(),
			"from_status": event.FromStatus,
			"to_status":   event.ToStatus,
		},
	}); err != nil {
		return fmt.Errorf("write order status audit: %w", err)
	}
	return nil
}
