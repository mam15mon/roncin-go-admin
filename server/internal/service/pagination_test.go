package service

import (
	"errors"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestListPageValues(t *testing.T) {
	invalidErr := errors.New("分页参数不合法")
	tests := []struct {
		name         string
		page         int32
		pageSize     int32
		wantPage     int
		wantPageSize int
		wantErr      error
	}{
		{name: "零值使用默认分页", wantPage: 1, wantPageSize: defaultListPageSize},
		{name: "保留合法分页", page: 2, pageSize: 50, wantPage: 2, wantPageSize: 50},
		{name: "接受最大分页", page: 1, pageSize: biz.MaxListPageSize, wantPage: 1, wantPageSize: biz.MaxListPageSize},
		{name: "拒绝负数页码", page: -1, pageSize: 20, wantErr: invalidErr},
		{name: "拒绝负数行数", page: 1, pageSize: -1, wantErr: invalidErr},
		{name: "拒绝超大分页", page: 1, pageSize: biz.MaxListPageSize + 1, wantErr: invalidErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, pageSize, err := listPageValues(test.page, test.pageSize, invalidErr)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("分页错误 = %v，期望 %v", err, test.wantErr)
			}
			if page != test.wantPage || pageSize != test.wantPageSize {
				t.Fatalf("分页结果 = %d/%d，期望 %d/%d", page, pageSize, test.wantPage, test.wantPageSize)
			}
		})
	}
}
