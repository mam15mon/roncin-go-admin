package data

import (
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

// TestFinanceCommissionLineIndexMetadata 断言提成行明细查询索引存在于生成元数据：
// 「页面订单 + 员工」复合索引支撑订单列表提成摘要的 EMPLOYEE 视图，使其在统计
// 信息缺失时也保持 order_id 前导的精确扫描，不退化为组织全量行扫描。
func TestFinanceCommissionLineIndexMetadata(t *testing.T) {
	cases := []struct {
		name    string
		unique  bool
		columns []string
	}{
		{
			name:    "financecommissionline_commission_id_order_id",
			unique:  true,
			columns: []string{"commission_id", "order_id"},
		},
		{
			name:    "financecommissionline_organization_id_employee_id",
			unique:  false,
			columns: []string{"organization_id", "employee_id"},
		},
		{
			name:    "financecommissionline_order_id",
			unique:  false,
			columns: []string{"order_id"},
		},
		{
			name:    "financecommissionline_order_id_employee_id",
			unique:  false,
			columns: []string{"order_id", "employee_id"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, idx := range migrate.FinanceCommissionLinesTable.Indexes {
				if idx.Name != tc.name {
					continue
				}
				if idx.Unique != tc.unique {
					t.Fatalf("索引 %s 唯一性 = %v，期望 %v", tc.name, idx.Unique, tc.unique)
				}
				if len(idx.Columns) != len(tc.columns) {
					t.Fatalf("索引 %s 列数 = %d，期望 %d", tc.name, len(idx.Columns), len(tc.columns))
				}
				for i, column := range tc.columns {
					if idx.Columns[i].Name != column {
						t.Fatalf("索引 %s 第 %d 列 = %s，期望 %s", tc.name, i, idx.Columns[i].Name, column)
					}
				}
				return
			}
			t.Fatalf("生成元数据缺少索引 %s", tc.name)
		})
	}
}
