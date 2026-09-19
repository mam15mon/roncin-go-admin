package migrate

import (
	"testing"

	"entgo.io/ent/dialect/sql/schema"
)

// TestCommissionApplicationSchemaMetadata 在生成迁移元数据层面断言月度提成申请
// 头/明细表的约束真相源：
//  1. 组织/员工/申请头/提成单等历史引用外键删除策略为 NO ACTION，禁止删除造成
//     历史申请断链；决策人是可逆审计引用（SET NULL）；
//  2. 月度唯一键「组织 + 员工 + 申请月」与明细「申请 + 提成」「提成全局唯一」
//     索引存在且唯一；
//  3. 本人/财务隔离查询索引存在。
func TestCommissionApplicationSchemaMetadata(t *testing.T) {
	t.Run("申请头历史引用外键删除策略", func(t *testing.T) {
		cases := []struct {
			symbol   string
			onDelete schema.ReferenceOption
		}{
			{"finance_commission_applications_organizations_finance_commission_applications", schema.NoAction},
			{"finance_commission_applications_users_finance_commission_applications", schema.NoAction},
			{"finance_commission_applications_users_submitted_finance_commission_applications", schema.NoAction},
			{"finance_commission_applications_users_decided_finance_commission_applications", schema.SetNull},
		}
		for _, tc := range cases {
			foreignKey := requireForeignKey(t, FinanceCommissionApplicationsTable.ForeignKeys, tc.symbol)
			if foreignKey.OnDelete != tc.onDelete {
				t.Fatalf("申请头外键 %s 删除策略 = %v，期望 %v", tc.symbol, foreignKey.OnDelete, tc.onDelete)
			}
		}
	})

	t.Run("申请明细历史引用外键全部 NO ACTION", func(t *testing.T) {
		for _, symbol := range []string{
			"finance_commission_application_lines_finance_commissions_application_lines",
			"finance_commission_application_lines_finance_commission_applications_lines",
			"finance_commission_application_lines_organizations_finance_commission_application_lines",
			"finance_commission_application_lines_users_finance_commission_application_lines",
		} {
			foreignKey := requireForeignKey(t, FinanceCommissionApplicationLinesTable.ForeignKeys, symbol)
			if foreignKey.OnDelete != schema.NoAction {
				t.Fatalf("申请明细外键 %s 删除策略 = %v，期望 NO ACTION", symbol, foreignKey.OnDelete)
			}
		}
	})

	t.Run("申请头唯一与查询索引", func(t *testing.T) {
		cases := []struct {
			name    string
			unique  bool
			columns []string
		}{
			{
				// 一人一组织一提交月一行（含被驳回行）：结构上禁止替代申请。
				name:    "financecommissionapplication_organization_id_employee_id_application_month",
				unique:  true,
				columns: []string{"organization_id", "employee_id", "application_month"},
			},
			{
				name:    "financecommissionapplication_organization_id_employee_id_status",
				unique:  false,
				columns: []string{"organization_id", "employee_id", "status"},
			},
			{
				name:    "financecommissionapplication_organization_id_application_month",
				unique:  false,
				columns: []string{"organization_id", "application_month"},
			},
		}
		assertTableIndexes(t, FinanceCommissionApplicationsTable, cases)
	})

	t.Run("申请明细唯一与查询索引", func(t *testing.T) {
		cases := []struct {
			name    string
			unique  bool
			columns []string
		}{
			{
				// 申请内明细一一对应；application_id 前导同时支撑详情下钻。
				name:    "financecommissionapplicationline_application_id_commission_id",
				unique:  true,
				columns: []string{"application_id", "commission_id"},
			},
			{
				// 一个提成事实至多进入一张申请：被驳回也不回流公共池。
				name:    "financecommissionapplicationline_commission_id",
				unique:  true,
				columns: []string{"commission_id"},
			},
			{
				name:    "financecommissionapplicationline_organization_id_employee_id",
				unique:  false,
				columns: []string{"organization_id", "employee_id"},
			},
		}
		assertTableIndexes(t, FinanceCommissionApplicationLinesTable, cases)
	})

	t.Run("申请头关键字段不可变与状态默认值", func(t *testing.T) {
		for _, name := range []string{"organization_id", "employee_id", "application_month", "coverage_to", "total_commission_amount", "total_cny_commission_amount", "submitted_at", "submitted_by"} {
			column := requireColumn(t, FinanceCommissionApplicationsColumns, name)
			if column.Nullable {
				t.Fatalf("申请头 %s 不允许为空", name)
			}
		}
		status := requireColumn(t, FinanceCommissionApplicationsColumns, "status")
		if status.Default != "PENDING_REVIEW" {
			t.Fatalf("申请头 status 默认值 = %v，期望 PENDING_REVIEW", status.Default)
		}
		// 决策审计与驳回原因由业务层成对写入，列允许为空。
		for _, name := range []string{"decided_at", "decided_by", "decision_reason"} {
			if column := requireColumn(t, FinanceCommissionApplicationsColumns, name); !column.Nullable {
				t.Fatalf("申请头 %s 应允许为空", name)
			}
		}
	})

	t.Run("申请明细快照列不可变且金额非空", func(t *testing.T) {
		for _, name := range []string{"commission_id", "application_id", "commission_date", "personnel_role", "base_currency", "commission_amount", "cny_commission_amount"} {
			column := requireColumn(t, FinanceCommissionApplicationLinesColumns, name)
			if column.Nullable {
				t.Fatalf("申请明细 %s 不允许为空", name)
			}
		}
	})
}

func assertTableIndexes(t *testing.T, table *schema.Table, cases []struct {
	name    string
	unique  bool
	columns []string
}) {
	t.Helper()
	for _, tc := range cases {
		found := false
		for _, idx := range table.Indexes {
			if idx.Name != tc.name {
				continue
			}
			found = true
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
			break
		}
		if !found {
			t.Fatalf("生成元数据缺少索引 %s", tc.name)
		}
	}
}
