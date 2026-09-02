package biz

import (
	"context"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderNotFound                  = errors.NotFound("ORDER_NOT_FOUND", "订单不存在")
	ErrOrderInvalidArgument           = errors.BadRequest("ORDER_INVALID_ARGUMENT", "订单字段不合法")
	ErrOrderNumberExists              = errors.Conflict("ORDER_NUMBER_EXISTS", "订单编号已存在")
	ErrOrderCustomerInvalid           = errors.BadRequest("ORDER_CUSTOMER_INVALID", "订单客户必须是启用的客户角色")
	ErrOrderStatusInvalid             = errors.BadRequest("ORDER_STATUS_INVALID", "订单状态不合法")
	ErrOrderStatusConflict            = errors.Conflict("ORDER_STATUS_CONFLICT", "订单状态已被其他操作修改")
	ErrOrderBusinessUnsupported       = errors.BadRequest("ORDER_BUSINESS_UNSUPPORTED", "当前仅支持海运出口订单")
	ErrOrderTerminationInvalid        = errors.BadRequest("ORDER_TERMINATION_INVALID", "订单终止状态流转不合法")
	ErrOrderClosureInvalid            = errors.BadRequest("ORDER_CLOSURE_INVALID", "订单结案状态流转不合法")
	ErrOrderClosureBlocked            = errors.Conflict("ORDER_CLOSURE_BLOCKED", "订单尚未满足结案条件")
	ErrOrderConsolidationShipmentType = errors.BadRequest("ORDER_CONSOLIDATION_SHIPMENT_TYPE_INVALID", "仅拼箱订单可查看自拼汇总")
)

// OrderUsecase 的领域错误与仓储接口集中在本文件；枚举与领域对象见
// order_types.go，查询、创建、草稿更新用例见 order_usecase.go，状态流转
// 用例见 order_transition.go。
type OrderRepo interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*Order, error)
	Find(context.Context, uuid.UUID) (*Order, error)
	List(context.Context, []uuid.UUID, OrderListOptions) (*OrderList, error)
	FindReferenceDuplicate(context.Context, uuid.UUID, OrderReferenceCheck) (*OrderReferenceMatch, error)
	ListPersonnelOptions(context.Context, uuid.UUID, SelectorListOptions) (*PagedList[*OrderPersonnelOption], error)
	HasContainers(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	ListConsolidationSummaries(context.Context, uuid.UUID, uuid.UUID) ([]*OrderConsolidationSummary, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *Order, *AuditEvent) (*Order, error)
	UpdateDraft(context.Context, uuid.UUID, uuid.UUID, uint64, *Order, *AuditEvent) (*Order, error)
	TransitionStatus(context.Context, uuid.UUID, uuid.UUID, uint64, OrderFlowStatus, string, uuid.UUID, *OrderStatusChangedEvent) (*Order, error)
	TransitionTermination(context.Context, uuid.UUID, uuid.UUID, uint64, OrderTerminationStatus, *OrderTerminationType, string, uuid.UUID, *OrderLifecycleChangedEvent) (*Order, error)
	ClosureReadiness(context.Context, uuid.UUID, uuid.UUID) (*OrderClosureReadiness, error)
	TransitionClosure(context.Context, uuid.UUID, uuid.UUID, uint64, OrderClosureStatus, string, uuid.UUID, *OrderLifecycleChangedEvent) (*Order, error)
}

type OrderUsecase struct {
	repo              OrderRepo
	tagRepo           BusinessTagRepo
	seaMasterBillRepo SeaMasterBillRepo
}

func NewOrderUsecase(repo OrderRepo, tagRepo BusinessTagRepo, seaMasterBillRepo SeaMasterBillRepo) *OrderUsecase {
	return &OrderUsecase{repo: repo, tagRepo: tagRepo, seaMasterBillRepo: seaMasterBillRepo}
}
