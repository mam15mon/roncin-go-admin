package biz

// MaxListPageSize 是所有常规后端列表接口统一允许的最大分页行数。
const MaxListPageSize = 200

// ValidListPagination 校验常规列表接口的分页参数。
func ValidListPagination(page, pageSize int) bool {
	return page >= 1 && pageSize >= 1 && pageSize <= MaxListPageSize
}
