package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// OrderStatusChangedEvent 订单主流程状态变更领域事件。
type OrderStatusChangedEvent struct {
	OrganizationID uuid.UUID
	OrderID        uuid.UUID
	ActorID        uuid.UUID
	FromStatus     OrderFlowStatus
	ToStatus       OrderFlowStatus
	OccurredAt     time.Time
}

// OrderLifecycleChangedEvent 订单终止或结案维度变更领域事件。
type OrderLifecycleChangedEvent struct {
	OrganizationID uuid.UUID
	OrderID        uuid.UUID
	ActorID        uuid.UUID
	Dimension      string
	FromStatus     string
	ToStatus       string
	OccurredAt     time.Time
}

// OrderClosureReadiness 是结案前从持久化层读取的业务事实。
type OrderClosureReadiness struct {
	FlowStatus           OrderFlowStatus
	TerminationStatus    OrderTerminationStatus
	ClosureStatus        OrderClosureStatus
	HasActiveException   bool
	HasUnbilledOrderFees bool
}

var allowedSEFlowTransitions = map[OrderFlowStatus][]OrderFlowStatus{
	OrderFlowDraft:                      {OrderFlowBooked},
	OrderFlowBooked:                     {OrderFlowSpaceAllocated},
	OrderFlowSpaceAllocated:             {OrderFlowTruckingArranged, OrderFlowDocumentCutoff},
	OrderFlowTruckingArranged:           {OrderFlowDocumentCutoff},
	OrderFlowDocumentCutoff:             {OrderFlowCustomsDeclarationArranged},
	OrderFlowCustomsDeclarationArranged: {OrderFlowDocumentReleased},
}

func validSEFlowTransition(from, to OrderFlowStatus) bool {
	for _, target := range allowedSEFlowTransitions[from] {
		if target == to {
			return true
		}
	}
	return false
}

// AllowedTargetFlowStatuses 返回当前订单可执行的主流程目标状态。
func (order *Order) AllowedTargetFlowStatuses() []OrderFlowStatus {
	if order.BusinessType != OrderBusinessSE || order.TerminationStatus != OrderTerminationActive || order.ClosureStatus != OrderClosureOpen {
		return nil
	}
	return append([]OrderFlowStatus(nil), allowedSEFlowTransitions[order.FlowStatus]...)
}

func validTerminationTransition(from, to OrderTerminationStatus) bool {
	switch from {
	case OrderTerminationActive:
		return to == OrderTerminationTerminating
	case OrderTerminationTerminating:
		return to == OrderTerminationActive || to == OrderTerminationTerminated
	case OrderTerminationTerminated:
		return to == OrderTerminationActive
	default:
		return false
	}
}

// TransitionStatus 校验并执行 SE 订单主流程流转。
//
// 主流程推进属于业务写入：仅允许终止维度 ACTIVE、结案维度 OPEN 且未业务锁定
// 的订单流转；该约束由生命周期专属校验表达，仓储层在 Order FOR UPDATE 后独立
// 复验，不复用内容写门禁。
func (uc *OrderUsecase) TransitionStatus(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, targetStatus OrderFlowStatus, reason string) (*Order, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || !targetStatus.Valid() || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrOrderStatusInvalid
	}
	current, err := uc.repo.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if current.Version != expectedVersion {
		return nil, ErrOrderStatusConflict
	}
	if current.BusinessType != OrderBusinessSE || current.TerminationStatus != OrderTerminationActive || current.ClosureStatus != OrderClosureOpen || !validSEFlowTransition(current.FlowStatus, targetStatus) {
		return nil, ErrOrderStatusInvalid
	}
	event := &OrderStatusChangedEvent{OrganizationID: organizationID, OrderID: id, ActorID: actorID, FromStatus: current.FlowStatus, ToStatus: targetStatus, OccurredAt: time.Now().UTC()}
	return uc.repo.TransitionStatus(ctx, organizationID, id, expectedVersion, targetStatus, reason, actorID, event)
}

// TransitionTermination 校验并执行订单终止维度流转。
//
// 终止维度是生命周期命令，不经过内容写门禁：完成（TERMINATING → TERMINATED）、
// 取消（TERMINATING → ACTIVE）与恢复（TERMINATED → ACTIVE）不因业务锁或历史
// 终止状态被阻断；仅 ACTIVE → TERMINATING 发起时仍要求订单未业务锁定。
// 最终进入 TERMINATED 时，仓储层在同一事务结束该订单的活动 SE Link，且恢复
// 订单不自动复活历史 Link。
func (uc *OrderUsecase) TransitionTermination(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, target OrderTerminationStatus, terminationType *OrderTerminationType, reason string) (*Order, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || !target.Valid() || reason == "" || utf8.RuneCountInString(reason) > 500 || target != OrderTerminationActive && (terminationType == nil || !terminationType.Valid()) {
		return nil, ErrOrderTerminationInvalid
	}
	current, err := uc.repo.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if current.Version != expectedVersion {
		return nil, ErrOrderStatusConflict
	}
	if current.ClosureStatus != OrderClosureOpen || !validTerminationTransition(current.TerminationStatus, target) {
		return nil, ErrOrderTerminationInvalid
	}
	event := &OrderLifecycleChangedEvent{OrganizationID: organizationID, OrderID: id, ActorID: actorID, Dimension: "TERMINATION", FromStatus: string(current.TerminationStatus), ToStatus: string(target), OccurredAt: time.Now().UTC()}
	return uc.repo.TransitionTermination(ctx, organizationID, id, expectedVersion, target, terminationType, reason, actorID, event)
}

// TransitionClosure 校验并执行订单结案或反结案。
//
// 结案与反结案是生命周期命令，不经过内容写门禁也不校验业务锁：已业务锁定的
// DOCUMENT_RELEASED 订单允许结案；结案 readiness（主流程完成或终止完成、无
// 活动异常、无未开账费用）由仓储层在事务内重验。
func (uc *OrderUsecase) TransitionClosure(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, target OrderClosureStatus, reason string) (*Order, error) {
	reason = strings.TrimSpace(reason)
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 || !target.Valid() || reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrOrderClosureInvalid
	}
	readiness, err := uc.repo.ClosureReadiness(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	if target == readiness.ClosureStatus {
		return nil, ErrOrderClosureInvalid
	}
	if target == OrderClosureClosed {
		flowFinished := readiness.FlowStatus == OrderFlowDocumentReleased
		terminated := readiness.TerminationStatus == OrderTerminationTerminated
		if (!flowFinished && !terminated) || readiness.HasActiveException || readiness.HasUnbilledOrderFees {
			return nil, ErrOrderClosureBlocked
		}
	}
	event := &OrderLifecycleChangedEvent{OrganizationID: organizationID, OrderID: id, ActorID: actorID, Dimension: "CLOSURE", FromStatus: string(readiness.ClosureStatus), ToStatus: string(target), OccurredAt: time.Now().UTC()}
	return uc.repo.TransitionClosure(ctx, organizationID, id, expectedVersion, target, reason, actorID, event)
}

func (event *OrderStatusChangedEvent) AuditEvent() *AuditEvent {
	return lifecycleAuditEvent(event.OrganizationID, event.OrderID, event.ActorID, "order.flow.transition", string(event.FromStatus), string(event.ToStatus))
}

func (event *OrderLifecycleChangedEvent) AuditEvent() *AuditEvent {
	return lifecycleAuditEvent(event.OrganizationID, event.OrderID, event.ActorID, "order."+strings.ToLower(event.Dimension)+".transition", event.FromStatus, event.ToStatus)
}

func lifecycleAuditEvent(organizationID, orderID, actorID uuid.UUID, action, from, to string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", Details: map[string]string{"order.id": orderID.String(), "from_status": from, "to_status": to}}
}
