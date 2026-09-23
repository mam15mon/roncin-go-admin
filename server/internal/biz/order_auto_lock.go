package biz

import (
	"context"
	"github.com/google/uuid"
)

// 自动业务锁定的稳定触发类型：只由有效应收结清事件、未建账费用终态事件
// 与费用进入账单事件驱动。
const (
	// AutoLockTriggerVerification 应收核销创建生效。
	AutoLockTriggerVerification AutoLockTriggerSource = "VERIFICATION"
	// AutoLockTriggerNetting 应收对冲确认生效。
	AutoLockTriggerNetting AutoLockTriggerSource = "NETTING"
	// AutoLockTriggerFeeCancel 未建账费用删除（阻断解除方向之一）。
	AutoLockTriggerFeeCancel AutoLockTriggerSource = "FEE_CANCEL"
	// AutoLockTriggerFeeBilled 费用进入账单（UNBILLED→BILLED，最后一笔未建账
	// 费用建账后订单可能转为可锁定）。
	AutoLockTriggerFeeBilled AutoLockTriggerSource = "FEE_BILLED"
)

// 自动锁定检查的结构化审计原因码。
const (
	AutoLockReasonLocked              = "LOCKED"
	AutoLockReasonAlreadyLocked       = "ALREADY_LOCKED"
	AutoLockReasonNoSettlement        = "NOT_ELIGIBLE_NO_SETTLEMENT"
	AutoLockReasonUnsettledReceivable = "NOT_ELIGIBLE_UNSETTLED_RECEIVABLE"
	AutoLockReasonUnbilledFee         = "NOT_ELIGIBLE_UNBILLED_FEE"
	AutoLockReasonLifecycle           = "NOT_ELIGIBLE_LIFECYCLE"
	AutoLockReasonNoActiveLink        = "NOT_ELIGIBLE_NO_ACTIVE_LINK"
	AutoLockReasonMemberSetChanged    = "NOT_ELIGIBLE_MEMBER_SET_CHANGED"
	AutoLockReasonMemberNotQualified  = "NOT_ELIGIBLE_MEMBER_NOT_QUALIFIED"
	AutoLockReasonDocumentStructure   = "EXECUTION_FAILED_DOCUMENT_STRUCTURE"
	AutoLockReasonExecutionFailed     = "EXECUTION_FAILED"
)

// AutoOrderLockTrigger 描述一次内部可信的自动锁定检查触发来源。
// 触发由服务端在原业务事务成功提交后发出，不携带任何调用人授权语义；
// TriggeredBy 只表示导致本次检查的业务操作人，不表示其执行或批准了锁单。
type AutoOrderLockTrigger struct {
	Type           AutoLockTriggerSource
	ResourceID     uuid.UUID
	OrganizationID uuid.UUID
	TriggeredBy    uuid.UUID
	// OrderID 供 FEE_CANCEL 触发直接给出目标订单；VERIFICATION/NETTING 触发由
	// 仓储按有效分摊解析全部受影响订单；FEE_BILLED 以 ResourceID 给出账单，
	// 由仓储按账单行解析全部受影响订单。
	OrderID uuid.UUID
}

// AutoLockTriggerSource 触发类型取值集合。
type AutoLockTriggerSource string

// Valid 校验触发类型是否为已登记的取值。
func (t AutoLockTriggerSource) Valid() bool {
	switch t {
	case AutoLockTriggerVerification, AutoLockTriggerNetting, AutoLockTriggerFeeCancel, AutoLockTriggerFeeBilled:
		return true
	default:
		return false
	}
}

// AutoOrderLockRepo 自动业务锁定检查的数据访问接口。
type AutoOrderLockRepo interface {
	// RunAutoSettlementLockCheck 执行一次独立事务的自动锁定检查。
	// 检查结果只写入审计，不向调用方回传业务错误以外的状态；
	// 任何失败都不得影响已经提交的原业务事实。
	RunAutoSettlementLockCheck(ctx context.Context, trigger AutoOrderLockTrigger) error
}

// AutoOrderLockUsecase 结清事件驱动的自动业务锁用例。
type AutoOrderLockUsecase struct {
	repo AutoOrderLockRepo
}

func NewAutoOrderLockUsecase(repo AutoOrderLockRepo) *AutoOrderLockUsecase {
	return &AutoOrderLockUsecase{repo: repo}
}

// RunSettlementLockCheck 在原业务事务成功提交后触发一次自动锁定检查。
// 触发方必须吞掉本方法的错误并记录日志：自动锁定失败不得回滚或改变
// 已提交的业务事实，也不得把锁定失败伪装成原业务失败。
func (uc *AutoOrderLockUsecase) RunSettlementLockCheck(ctx context.Context, trigger AutoOrderLockTrigger) error {
	if uc == nil || uc.repo == nil {
		return nil
	}
	if trigger.OrganizationID == uuid.Nil || trigger.ResourceID == uuid.Nil || trigger.TriggeredBy == uuid.Nil || !trigger.Type.Valid() {
		return ErrOrderInvalidArgument
	}
	switch trigger.Type {
	case AutoLockTriggerVerification, AutoLockTriggerNetting, AutoLockTriggerFeeBilled:
	case AutoLockTriggerFeeCancel:
		if trigger.OrderID == uuid.Nil {
			return ErrOrderInvalidArgument
		}
	}
	return uc.repo.RunAutoSettlementLockCheck(ctx, trigger)
}
