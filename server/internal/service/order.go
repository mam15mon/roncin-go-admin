package service

import (
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// OrderService 按职责拆分实现：查询（order_query.go）、写入与状态流转
// （order_write.go）、对象转换（order_convert.go）、枚举映射（order_enum.go）；
// 本文件保留服务锚点与列表查询所用时区。
type OrderService struct {
	v1.UnimplementedOrderServiceServer
	usecase *biz.OrderUsecase
}

var orderListDateLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func NewOrderService(usecase *biz.OrderUsecase) *OrderService { return &OrderService{usecase: usecase} }

var _ v1.OrderServiceServer = (*OrderService)(nil)
