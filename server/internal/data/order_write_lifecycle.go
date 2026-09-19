package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func (r *orderRepo) TransitionStatus(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, targetStatus biz.OrderFlowStatus, reason string, actorID uuid.UUID, event *biz.OrderStatusChangedEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if existing.Version != expectedVersion || biz.OrderFlowStatus(existing.FlowStatus) != event.FromStatus {
			return biz.ErrOrderStatusConflict
		}
		// 主流程推进属于业务写入：生命周期专属校验要求终止维度 ACTIVE、结案维度
		// OPEN；业务锁规则保持现状，锁定后不得推进主流程。
		if existing.TerminationStatus != orderent.TerminationStatusACTIVE || existing.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderStatusConflict
		}
		if existing.LockedAt != nil {
			return ensureOrderNotBusinessLocked(ctx, tx.User, existing)
		}
		if _, updateErr := existing.Update().SetFlowStatus(orderent.FlowStatus(targetStatus)).SetVersion(existing.Version + 1).Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, eventErr := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionFLOW).SetFromStatus(string(event.FromStatus)).SetToStatus(string(targetStatus)).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); eventErr != nil {
			return eventErr
		}
		return writeAudit(ctx, tx.AuditLog, event.AuditEvent())
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *orderRepo) TransitionTermination(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderTerminationStatus, terminationType *biz.OrderTerminationType, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		if existing.Version != expectedVersion || string(existing.TerminationStatus) != event.FromStatus {
			return biz.ErrOrderStatusConflict
		}
		// 终止维度是生命周期命令：不复用内容写门禁。专属校验只要求结案维度
		// 保持 OPEN；完成、取消与恢复不得因业务锁或历史终止状态被误封。
		if existing.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderTerminationInvalid
		}
		// ACTIVE → TERMINATING 属于发起业务变更，保持“锁定资料不可发起退关”
		// 的现行约束；其余流转路径不因业务锁被阻断。
		if existing.TerminationStatus == orderent.TerminationStatusACTIVE &&
			target == biz.OrderTerminationTerminating && existing.LockedAt != nil {
			return ensureOrderNotBusinessLocked(ctx, tx.User, existing)
		}
		update := existing.Update().SetTerminationStatus(orderent.TerminationStatus(target)).SetVersion(existing.Version + 1)
		if target == biz.OrderTerminationActive {
			update.ClearTerminationType().ClearTerminationReason().ClearTerminatedAt().ClearTerminatedBy()
		} else {
			update.SetTerminationType(orderent.TerminationType(*terminationType)).SetTerminationReason(reason)
			if target == biz.OrderTerminationTerminated {
				update.SetTerminatedAt(event.OccurredAt).SetTerminatedBy(actorID)
			} else {
				update.ClearTerminatedAt().ClearTerminatedBy()
			}
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return updateErr
		}
		// 最终进入 TERMINATED 时在同一事务结束该订单的活动 SE Link；任何一步
		// 失败（订单更新、Link 结束、生命周期事件、审计）整体回滚。
		if target == biz.OrderTerminationTerminated {
			if linkErr := endActiveSeaMasterBillLinksOnTermination(ctx, tx, existing, event.OccurredAt); linkErr != nil {
				return linkErr
			}
		}
		if _, eventErr := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionTERMINATION).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); eventErr != nil {
			return eventErr
		}
		return writeAudit(ctx, tx.AuditLog, event.AuditEvent())
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

// orderTerminationLinkEndedReason 是退关结束活动 Link 的固定中文原因。
const orderTerminationLinkEndedReason = "订单退关"

// endActiveSeaMasterBillLinksOnTermination 在订单最终流转到 TERMINATED 的同一
// 事务中结束其活动 SE Link。调用前必须已锁定订单行；活动 Link 按 ID 排序后
// FOR UPDATE，数量超过一条时返回结构冲突（fail-closed，不静默修复）。结束时
// 写入 UTC ended_at、固定中文 ended_reason 并递增 Link 版本；不修改 MBL 与
// TransportExecution 的内容版本。非 SE 订单没有 Link 操作。
func endActiveSeaMasterBillLinksOnTermination(ctx context.Context, tx *ent.Tx, order *ent.Order, occurredAt time.Time) error {
	if order.BusinessType != orderent.BusinessTypeSE {
		return nil
	}
	activeLinks, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(order.OrganizationID),
			seamasterbillorderlink.OrderIDEQ(order.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Order(seamasterbillorderlink.ByID()).
		ForUpdate().
		All(ctx)
	if err != nil {
		return err
	}
	if len(activeLinks) == 0 {
		return nil
	}
	if len(activeLinks) > 1 {
		return biz.ErrSeaDocumentStructureConflict
	}
	link := activeLinks[0]
	_, err = link.Update().
		SetStatus(seamasterbillorderlink.StatusENDED).
		SetEndedAt(occurredAt.UTC()).
		SetEndedReason(orderTerminationLinkEndedReason).
		SetVersion(link.Version + 1).
		Save(ctx)
	return err
}

func (r *orderRepo) ClosureReadiness(ctx context.Context, organizationID, id uuid.UUID) (*biz.OrderClosureReadiness, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	hasActiveException, err := client.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	hasUnbilledFees, err := client.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.OrderClosureReadiness{FlowStatus: biz.OrderFlowStatus(item.FlowStatus), TerminationStatus: biz.OrderTerminationStatus(item.TerminationStatus), ClosureStatus: biz.OrderClosureStatus(item.ClosureStatus), HasActiveException: hasActiveException, HasUnbilledOrderFees: hasUnbilledFees}, nil
}

func (r *orderRepo) TransitionClosure(ctx context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target biz.OrderClosureStatus, reason string, actorID uuid.UUID, event *biz.OrderLifecycleChangedEvent) (*biz.Order, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		// 结案与反结案是生命周期命令：不复用内容写门禁，也不校验业务锁，
		// 允许已业务锁定的 DOCUMENT_RELEASED 订单结案；版本与来源状态仍在
		// 事务内权威校验，结案 readiness 在下方重验。
		if existing.Version != expectedVersion || string(existing.ClosureStatus) != event.FromStatus {
			return biz.ErrOrderStatusConflict
		}
		if target == biz.OrderClosureClosed {
			flowFinished := biz.OrderFlowStatus(existing.FlowStatus) == biz.OrderFlowDocumentReleased
			terminated := biz.OrderTerminationStatus(existing.TerminationStatus) == biz.OrderTerminationTerminated
			if !flowFinished && !terminated {
				return biz.ErrOrderClosureBlocked
			}
			hasActiveException, readinessErr := tx.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(id), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).Exist(ctx)
			if readinessErr != nil {
				return readinessErr
			}
			hasUnbilledFees, readinessErr := tx.OrderFee.Query().Where(orderfeeent.OrderIDEQ(id), orderfeeent.StatusNotIn(orderfeeent.StatusBILLED, orderfeeent.StatusCANCELLED)).Exist(ctx)
			if readinessErr != nil {
				return readinessErr
			}
			if hasActiveException || hasUnbilledFees {
				return biz.ErrOrderClosureBlocked
			}
		}
		update := existing.Update().SetClosureStatus(orderent.ClosureStatus(target)).SetVersion(existing.Version + 1)
		if target == biz.OrderClosureClosed {
			update.SetClosureReason(reason).SetClosedAt(event.OccurredAt).SetClosedBy(actorID)
		} else {
			update.ClearClosureReason().ClearClosedAt().ClearClosedBy()
		}
		if _, updateErr := update.Save(ctx); updateErr != nil {
			return updateErr
		}
		if _, eventErr := tx.OrderLifecycleEvent.Create().SetOrderID(id).SetDimension(orderlifecycleeventent.DimensionCLOSURE).SetFromStatus(event.FromStatus).SetToStatus(event.ToStatus).SetAction("transition").SetReason(reason).SetOperatorID(actorID).SetChangedAt(event.OccurredAt).Save(ctx); eventErr != nil {
			return eventErr
		}
		return writeAudit(ctx, tx.AuditLog, event.AuditEvent())
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func parseOptionalTime(s string) *time.Time {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	return &t
}
