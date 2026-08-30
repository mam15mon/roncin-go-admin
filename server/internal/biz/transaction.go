package biz

import "context"

// Transactor 为需要协调多个仓储的用例提供数据库事务边界。
type Transactor interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
