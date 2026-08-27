package biz

import "testing"

func TestValidListPaginationUsesUnifiedMaximum(t *testing.T) {
	for _, test := range []struct {
		name     string
		page     int
		pageSize int
		want     bool
	}{
		{name: "minimum", page: 1, pageSize: 1, want: true},
		{name: "maximum", page: 1, pageSize: MaxListPageSize, want: true},
		{name: "page zero", page: 0, pageSize: 20, want: false},
		{name: "size zero", page: 1, pageSize: 0, want: false},
		{name: "over maximum", page: 1, pageSize: MaxListPageSize + 1, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidListPagination(test.page, test.pageSize); got != test.want {
				t.Fatalf("ValidListPagination(%d, %d) = %v, want %v", test.page, test.pageSize, got, test.want)
			}
		})
	}
}
