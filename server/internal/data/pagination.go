package data

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// paginate 统一执行计数、分页查询、转换与结果组装。
func paginate[E, T any](
	ctx context.Context,
	countFn func(context.Context) (int, error),
	itemsFn func(context.Context, int, int) ([]E, error),
	page, pageSize int,
	convert func(E) (T, error),
) (*biz.PagedList[T], error) {
	total, err := countFn(ctx)
	if err != nil {
		return nil, err
	}
	entities, err := itemsFn(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]T, 0, len(entities))
	for _, entity := range entities {
		item, convertErr := convert(entity)
		if convertErr != nil {
			return nil, convertErr
		}
		items = append(items, item)
	}
	return &biz.PagedList[T]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func infalliblePageConverter[E, T any](convert func(E) T) func(E) (T, error) {
	return func(entity E) (T, error) {
		return convert(entity), nil
	}
}
