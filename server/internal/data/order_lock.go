package data

import (
	"context"

	"github.com/google/uuid"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type orderLockRepo struct {
	data     *Data
	security *conf.Security
}

// NewOrderLockRepo 创建订单锁数据仓储实现。
func NewOrderLockRepo(data *Data, security *conf.Security) biz.OrderLockRepo {
	return &orderLockRepo{data: data, security: security}
}

// orderAccessBusinessType 把订单持久化类型收敛为权限目录类型，拒绝任何非法值。
func orderAccessBusinessType(businessType orderent.BusinessType) (access.OrderBusinessType, error) {
	parsed := access.OrderBusinessType(businessType)
	if !parsed.Valid() {
		return "", biz.ErrOrderBusinessUnsupported
	}
	return parsed, nil
}

// ensureOrderBusinessContentEditable 是全部订单业务内容写入口的统一门禁：
// 订单必须处于 termination_status=ACTIVE、closure_status=OPEN 且未业务锁定。
// 校验顺序为先生命周期后业务锁，使不可逆程度更高的生命周期原因优先呈现，
// 避免订单已终止但残留历史锁信息时误提示“先解锁即可编辑”。
// 本门禁只用于业务字段及子资源写入；终止、恢复、结案、反结案等生命周期命令
// 在锁定订单行后执行专属状态校验，不得复用本门禁。
// users 仅在订单已锁定时用于补充锁定人姓名，可为 nil。
func ensureOrderBusinessContentEditable(ctx context.Context, users *ent.UserClient, existing *ent.Order) error {
	if existing == nil {
		return nil
	}
	if _, err := orderAccessBusinessType(existing.BusinessType); err != nil {
		return err
	}
	switch existing.TerminationStatus {
	case orderent.TerminationStatusACTIVE:
	case orderent.TerminationStatusTERMINATING:
		return biz.ErrOrderTerminationInProgress
	case orderent.TerminationStatusTERMINATED:
		return biz.ErrOrderTerminated
	default:
		return biz.ErrOrderStatusConflict
	}
	if existing.ClosureStatus != orderent.ClosureStatusOPEN {
		return biz.ErrOrderClosed
	}
	if existing.LockedAt == nil {
		return nil
	}
	return ensureOrderNotBusinessLocked(ctx, users, existing)
}

// ensureOrderNotBusinessLocked 校验订单未被业务锁定；已锁定时返回携带订单、
// 代次、时间和锁定人元数据的结构化 ORDER_BUSINESS_LOCKED 错误。
func ensureOrderNotBusinessLocked(ctx context.Context, users *ent.UserClient, existing *ent.Order) error {
	var lockedByName string
	if existing.LockedBy != nil && users != nil {
		u, err := users.Get(ctx, *existing.LockedBy)
		if err == nil && u != nil {
			lockedByName = u.DisplayName
		}
	}
	return biz.NewErrOrderBusinessLocked(existing.ID, existing.OrderNo, existing.LockGeneration, *existing.LockedAt, lockedByName)
}

// ensureOrderBusinessEditable 是统一内容门禁的事务上下文适配，供在 *ent.Tx
// 中完成订单加锁的既有写入口复用；规则真相源在 ensureOrderBusinessContentEditable。
func ensureOrderBusinessEditable(ctx context.Context, tx *ent.Tx, existing *ent.Order) error {
	return ensureOrderBusinessContentEditable(ctx, tx.User, existing)
}

// orderBusinessEditBlockReason 返回订单业务内容门禁未通过时的中文原因；可编辑时返回空串。
func orderBusinessEditBlockReason(err error) string {
	if err == nil {
		return ""
	}
	if ke := kratoserrors.FromError(err); ke != nil && ke.Message != "" {
		return ke.Message
	}
	if err.Error() != "" {
		return err.Error()
	}
	return "不可编辑"
}

// orderBusinessEditImpact 为预览流程提供单订单业务内容门禁事实：当订单处于终止、结案或业务
// 锁定状态时返回 BlocksExecution=true 的下游影响对象，便于调用方统一阻断 Executable 并展示原因。
func orderBusinessEditImpact(ctx context.Context, users *ent.UserClient, o *ent.Order) *biz.SeaDocumentDownstreamImpact {
	if o == nil {
		return nil
	}
	if err := ensureOrderBusinessContentEditable(ctx, users, o); err != nil {
		return &biz.SeaDocumentDownstreamImpact{
			FactType:        "ORDER_BUSINESS_LOCK",
			ReferenceID:     o.ID.String(),
			ReferenceNo:     o.OrderNo,
			Message:         "订单 " + o.OrderNo + " " + orderBusinessEditBlockReason(err),
			BlocksExecution: true,
		}
	}
	return nil
}

func (r *orderLockRepo) GetOrderLockState(ctx context.Context, organizationID, orderID uuid.UUID, caller *biz.Principal) (*biz.OrderLockState, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	businessType, err := orderAccessBusinessType(order.BusinessType)
	if err != nil {
		return nil, err
	}
	updatePermission := access.OrderPermission(businessType, access.OrderUpdate)

	isLocked := order.LockedAt != nil
	var lockedByName *string
	if order.LockedBy != nil {
		u, err := client.User.Get(ctx, *order.LockedBy)
		if err == nil && u != nil {
			lockedByName = &u.DisplayName
		}
	}
	// 锁定来源由服务端统一投影：MANUAL 归属实际锁定人；AUTO_SETTLEMENT 由前端
	// 固定展示【系统自动锁定】，不根据 locked_by 为空自行猜测。
	var lockSource *string
	if order.LockSource != nil {
		value := string(*order.LockSource)
		lockSource = &value
	}

	state := &biz.OrderLockState{
		OrderID:        order.ID,
		OrderNo:        order.OrderNo,
		BusinessType:   biz.OrderBusinessType(businessType),
		IsLocked:       isLocked,
		LockGeneration: order.LockGeneration,
		LockedAt:       order.LockedAt,
		LockedBy:       order.LockedBy,
		LockedByName:   lockedByName,
		LockSource:     lockSource,
		OrderVersion:   order.Version,
	}

	// 读取当前锁定记录（若存在）
	if order.LockGeneration > 0 {
		rec, err := client.OrderLockRecord.Query().
			Where(
				orderlockrecordent.OrderIDEQ(order.ID),
				orderlockrecordent.GenerationEQ(order.LockGeneration),
			).
			WithHouseBillSnapshots().
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return nil, err
		}
		if rec != nil {
			if access.OrderBusinessType(rec.BusinessType) != businessType {
				return nil, biz.ErrOrderStatusConflict
			}
			state.CurrentLockRecord = r.mapLockRecord(ctx, client, rec)
		}
	}

	// 读取当前活动解锁请求
	activeReq, err := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.OrderIDEQ(order.ID),
			orderunlockrequestent.LockGenerationEQ(order.LockGeneration),
			orderunlockrequestent.StatusIn(
				biz.UnlockStatusPendingDispatch,
				biz.UnlockStatusPendingApproval,
				biz.UnlockStatusApprovedPendingApply,
				biz.UnlockStatusDispatchUnknown,
			),
		).
		WithApproverCandidates().
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if activeReq != nil {
		if access.OrderBusinessType(activeReq.BusinessType) != businessType {
			return nil, biz.ErrOrderStatusConflict
		}
		state.ActiveUnlockRequest = r.mapUnlockRequest(ctx, client, activeReq)
	}

	// 判断调用人角色资格：bootstrap admin 显式具备锁单资格（应急解锁入口见下方
	// 锁定分支）；普通用户按统一 lock grant 口径判定。
	isQualifiedBusinessRole := false
	if caller != nil {
		if caller.IsBootstrapAdmin {
			isQualifiedBusinessRole = true
		} else {
			isQualifiedBusinessRole, err = isUserQualifiedBusinessLockRole(ctx, client, organizationID, caller.UserID, businessType)
			if err != nil {
				return nil, err
			}
		}
	}

	// 只有 SE 锁定会形成海运单证版本，因此只对 SE 要求活动提单关系。
	hasActiveLink := true
	if businessType == access.OrderBusinessSE {
		hasActiveLink, err = client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
	}

	// 计算可执行动作与阻断原因
	if !isLocked {
		if order.TerminationStatus != orderent.TerminationStatusACTIVE {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "订单已进入终止流程")
		}
		if order.ClosureStatus != orderent.ClosureStatusOPEN {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "订单已结案")
		}
		if businessType == access.OrderBusinessSE && !hasActiveLink {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "订单未关联有效海运提单")
		}
		if !isQualifiedBusinessRole {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "当前用户未分配对应业务类型的订单锁定角色")
		}
		state.CanLock = len(state.LockBlockedReasons) == 0
	} else {
		state.LockBlockedReasons = append(state.LockBlockedReasons, "订单已被锁定")

		if caller != nil {
			if caller.IsBootstrapAdmin {
				state.CanAdminEmergencyUnlock = true
			} else if isQualifiedBusinessRole {
				state.CanRoleDirectUnlock = true
			} else if caller.HasPermissionInScope(updatePermission, biz.DataScopeOrganization) && caller.CanAccessOrganizationForPermission(updatePermission, order.OrganizationID, true) {
				if state.ActiveUnlockRequest == nil {
					state.CanRequestUnlock = true
				} else {
					state.UnlockBlockedReasons = append(state.UnlockBlockedReasons, "当前已有生效中或审批中的解锁请求")
				}
			} else {
				state.UnlockBlockedReasons = append(state.UnlockBlockedReasons, "无订单编辑权限")
			}
		}
	}

	return state, nil
}
