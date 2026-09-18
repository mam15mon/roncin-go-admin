package data

import (
	"testing"

	entsqlschema "entgo.io/ent/dialect/sql/schema"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

// TestCommissionRuleAssignmentSchemaFKMetadata 断言提成方案员工分配外键的删除
// 策略与生成元数据同源：方案、组织、员工与创建人是永久审计事实，必须 NO ACTION
// 保留历史引用，禁止删除造成历史断链；撤销/终止操作者是可逆审计引用保持 SET NULL。
func TestCommissionRuleAssignmentSchemaFKMetadata(t *testing.T) {
	cases := []struct {
		name     string
		fks      []*entsqlschema.ForeignKey
		symbol   string
		onDelete entsqlschema.ReferenceOption
	}{
		{
			name:     "分配方案外键必须 NO ACTION",
			fks:      migrate.FinanceCommissionRuleAssignmentsTable.ForeignKeys,
			symbol:   "finance_commission_rule_assignments_finance_commission_rules_assignments",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "分配组织外键必须 NO ACTION",
			fks:      migrate.FinanceCommissionRuleAssignmentsTable.ForeignKeys,
			symbol:   "finance_commission_rule_assignments_organizations_finance_commission_rule_assignments",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "分配员工外键必须 NO ACTION",
			fks:      migrate.FinanceCommissionRuleAssignmentsTable.ForeignKeys,
			symbol:   "finance_commission_rule_assignments_users_finance_commission_rule_assignments",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "分配创建人外键必须 NO ACTION",
			fks:      migrate.FinanceCommissionRuleAssignmentsTable.ForeignKeys,
			symbol:   "finance_commission_rule_assignments_users_created_finance_commission_rule_assignments",
			onDelete: entsqlschema.NoAction,
		},
		{
			name:     "分配撤销操作人外键保持 SET NULL",
			fks:      migrate.FinanceCommissionRuleAssignmentsTable.ForeignKeys,
			symbol:   "finance_commission_rule_assignments_users_cancelled_finance_commission_rule_assignments",
			onDelete: entsqlschema.SetNull,
		},
		{
			name:     "分配终止操作人外键保持 SET NULL",
			fks:      migrate.FinanceCommissionRuleAssignmentsTable.ForeignKeys,
			symbol:   "finance_commission_rule_assignments_users_terminated_finance_commission_rule_assignments",
			onDelete: entsqlschema.SetNull,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, fk := range tc.fks {
				if fk.Symbol != tc.symbol {
					continue
				}
				if fk.OnDelete != tc.onDelete {
					t.Fatalf("外键 %s 删除策略 = %v，期望 %v", tc.symbol, fk.OnDelete, tc.onDelete)
				}
				return
			}
			t.Fatalf("生成元数据缺少外键 %s", tc.symbol)
		})
	}
}

// TestCommissionRuleAssignmentSchemaIndexMetadata 断言分配唯一键与候选/门禁查询
// 索引存在于生成元数据：同一方案、员工和起始日唯一，允许退出后在不重叠区间
// 重新加入；组织+员工与方案+终止日索引支撑名单、停用检查与资格判定。
func TestCommissionRuleAssignmentSchemaIndexMetadata(t *testing.T) {
	cases := []struct {
		name    string
		unique  bool
		columns []string
	}{
		{
			name:    "financecommissionruleassignment_organization_id_rule_id_employee_id_effective_from",
			unique:  true,
			columns: []string{"organization_id", "rule_id", "employee_id", "effective_from"},
		},
		{
			name:    "financecommissionruleassignment_organization_id_employee_id_effective_to",
			unique:  false,
			columns: []string{"organization_id", "employee_id", "effective_to"},
		},
		{
			name:    "financecommissionruleassignment_rule_id_effective_to",
			unique:  false,
			columns: []string{"rule_id", "effective_to"},
		},
	}
	columns := migrate.FinanceCommissionRuleAssignmentsColumns
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, idx := range migrate.FinanceCommissionRuleAssignmentsTable.Indexes {
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

	// 分配段起始日必填不可变、终止日可空（列级事实，按列名定位）。
	assignmentColumn := func(name string) *entsqlschema.Column {
		t.Helper()
		for _, column := range columns {
			if column.Name == name {
				return column
			}
		}
		t.Fatalf("生成元数据缺少列 %s", name)
		return nil
	}
	if assignmentColumn("effective_from").Nullable {
		t.Fatalf("effective_from 必须必填")
	}
	if !assignmentColumn("effective_to").Nullable {
		t.Fatalf("effective_to 必须可空")
	}
	// 迁移停用的旧规则标记列：默认 false，新建方案不受影响。
	for _, column := range migrate.FinanceCommissionRulesColumns {
		if column.Name != "legacy_readonly" {
			continue
		}
		if column.Nullable || column.Default != false {
			t.Fatalf("legacy_readonly 列元数据异常: %#v", column)
		}
		return
	}
	t.Fatalf("finance_commission_rules 缺少 legacy_readonly 列")
}
