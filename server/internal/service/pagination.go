package service

import "github.com/roncin/roncin-go-admin/server/internal/biz"

const defaultListPageSize = 20

// listPageValues 解析传输层分页零值，并使用调用方所属领域错误拒绝非法范围。
func listPageValues(page, pageSize int32, invalidErr error) (int, int, error) {
	pageValue, pageSizeValue := biz.ListPagination(int(page), int(pageSize), defaultListPageSize)
	if !biz.ValidListPagination(pageValue, pageSizeValue) {
		return 0, 0, invalidErr
	}
	return pageValue, pageSizeValue, nil
}
