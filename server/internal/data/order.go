package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// orderRepo 按职责拆分实现：查询（order_query.go）、写入事务
// （order_write.go）、关联数据同步与校验（order_sync.go）、ent↔biz 转换
// （order_convert.go）；本文件只保留仓储锚点。
type orderRepo struct{ data *Data }

func NewOrderRepo(data *Data) biz.OrderRepo { return &orderRepo{data: data} }

var _ biz.OrderRepo = (*orderRepo)(nil)
