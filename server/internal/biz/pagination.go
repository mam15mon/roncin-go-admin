package biz

// MaxListPageSize 是所有常规后端列表接口统一允许的最大分页行数。
const MaxListPageSize = 200

// ValidListPagination 校验常规列表接口的分页参数。
func ValidListPagination(page, pageSize int) bool {
	return page >= 1 && pageSize >= 1 && pageSize <= MaxListPageSize
}

// ListPagination 将传输层的零值分页参数转换为统一默认值。
func ListPagination(page, pageSize, defaultPageSize int) (int, int) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	return page, pageSize
}

// PagedList 是领域层常规列表的统一分页结果。
type PagedList[T any] struct {
	Items    []T
	Total    int
	Page     int
	PageSize int
}

// SelectorListOptions 是下拉框和联想输入候选接口的统一查询参数。
type SelectorListOptions struct {
	Keyword  string
	Page     int
	PageSize int
}
