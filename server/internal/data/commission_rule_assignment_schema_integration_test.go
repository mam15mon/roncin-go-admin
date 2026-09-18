package data

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

// TestCommissionRuleAssignmentSchemaPostgres 在真实 PostgreSQL 上验证提成方案
// 员工分配 Schema 的约束真实生效：CHECK 拒绝起止倒置与不成对审计标记、方案/
// 组织/员工/创建人外键删除策略为 NO ACTION、唯一索引存在、旧角色规则停用迁移
// 真实改写存量行。跳过不算通过，必须以 RONCIN_INTEGRATION_DATABASE_SOURCE
// 注入专用库并确认 PASS。
func TestCommissionRuleAssignmentSchemaPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("CHECK 约束存在且包含关键条件", func(t *testing.T) {
		cases := []struct {
			name      string
			fragments []string
		}{
			{
				name:      "finance_commission_rule_assignments_effective_period_check",
				fragments: []string{"effective_to", "IS NULL", "effective_from", "<="},
			},
			{
				name:      "finance_commission_rule_assignments_cancel_audit_check",
				fragments: []string{"cancelled_at", "cancelled_by", "IS NOT NULL"},
			},
			{
				name:      "finance_commission_rule_assignments_terminate_audit_check",
				fragments: []string{"terminated_at", "terminated_by", "IS NOT NULL"},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var definition string
				err := data.sqlDB.QueryRowContext(ctx,
					`SELECT pg_get_constraintdef(c.oid)
					   FROM pg_constraint c
					  WHERE c.connamespace = current_schema()::regnamespace AND c.conname = $1`,
					tc.name).Scan(&definition)
				if err != nil {
					t.Fatalf("迁移后缺少 CHECK 约束 %s: %v", tc.name, err)
				}
				for _, fragment := range tc.fragments {
					if !strings.Contains(definition, fragment) {
						t.Errorf("CHECK 约束 %s 缺少 %q: %s", tc.name, fragment, definition)
					}
				}
			})
		}
	})

	t.Run("历史引用外键删除策略", func(t *testing.T) {
		cases := []struct {
			column     string
			deleteRule string
		}{
			{column: "rule_id", deleteRule: "NO ACTION"},
			{column: "organization_id", deleteRule: "NO ACTION"},
			{column: "employee_id", deleteRule: "NO ACTION"},
			{column: "created_by", deleteRule: "NO ACTION"},
			{column: "cancelled_by", deleteRule: "SET NULL"},
			{column: "terminated_by", deleteRule: "SET NULL"},
		}
		for _, tc := range cases {
			t.Run(tc.column, func(t *testing.T) {
				var deleteRule string
				err := data.sqlDB.QueryRowContext(ctx,
					`SELECT rc.delete_rule
					   FROM information_schema.referential_constraints rc
					   JOIN information_schema.key_column_usage kcu
					     ON kcu.constraint_name = rc.constraint_name
					    AND kcu.constraint_schema = rc.constraint_schema
					  WHERE rc.constraint_schema = current_schema()
					    AND kcu.table_name = 'finance_commission_rule_assignments'
					    AND kcu.column_name = $1`,
					tc.column).Scan(&deleteRule)
				if err != nil {
					if err == sql.ErrNoRows {
						t.Fatalf("缺少外键 finance_commission_rule_assignments.%s", tc.column)
					}
					t.Fatalf("查询外键 %s 失败: %v", tc.column, err)
				}
				if deleteRule != tc.deleteRule {
					t.Fatalf("外键 %s delete_rule=%q，期望 %s", tc.column, deleteRule, tc.deleteRule)
				}
			})
		}
	})

	t.Run("同方案员工起始日唯一索引存在", func(t *testing.T) {
		var definition string
		err := data.sqlDB.QueryRowContext(ctx,
			`SELECT indexdef FROM pg_indexes
			  WHERE schemaname = current_schema()
			    AND indexname = 'financecommissionruleassignment_organization_id_rule_id_employee_id_effective_from'`).
			Scan(&definition)
		if err != nil {
			t.Fatalf("缺少分配唯一索引: %v", err)
		}
		for _, fragment := range []string{"CREATE UNIQUE INDEX", "organization_id", "rule_id", "employee_id", "effective_from"} {
			if !strings.Contains(definition, fragment) {
				t.Fatalf("唯一索引定义缺少 %q: %s", fragment, definition)
			}
		}
	})

	t.Run("CHECK 拒绝起止倒置与不成对审计标记", func(t *testing.T) {
		base := `INSERT INTO finance_commission_rule_assignments
			("id","created_at","updated_at","organization_id","rule_id","employee_id",
			 "effective_from","effective_to","created_by","cancelled_at","cancelled_by",
			 "terminated_at","terminated_by")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
		args := func(effectiveTo, cancelledAt, cancelledBy, terminatedAt, terminatedBy any) []any {
			return []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(),
				"2026-10-01", effectiveTo, uuid.New(), cancelledAt, cancelledBy, terminatedAt, terminatedBy}
		}
		cases := []struct {
			name       string
			args       []any
			constraint string
		}{
			{name: "终止日早于起始日被拒绝", args: args(strPtr("2026-09-30"), nil, nil, nil, nil), constraint: "finance_commission_rule_assignments_effective_period_check"},
			{name: "撤销时间缺少操作者被拒绝", args: args(nil, time.Now(), nil, nil, nil), constraint: "finance_commission_rule_assignments_cancel_audit_check"},
			{name: "终止时间缺少操作者被拒绝", args: args(nil, nil, nil, time.Now(), nil), constraint: "finance_commission_rule_assignments_terminate_audit_check"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := data.sqlDB.ExecContext(ctx, base, tc.args...)
				assertCheckViolation(t, err, tc.constraint)
			})
		}
	})

	t.Run("删除被引用方案被 NO ACTION 阻止", func(t *testing.T) {
		suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		org, err := data.db.Organization.Create().
			SetCode("CRA-" + suffix).
			SetName("分配Schema测试组织-" + suffix).
			SetKind("headquarters").
			SetBaseCurrency("CNY").
			Save(ctx)
		if err != nil {
			t.Fatalf("创建测试组织: %v", err)
		}
		employee, err := data.db.User.Create().
			SetUsername("cra_" + suffix).
			SetDisplayName("分配Schema员工-" + suffix).
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建测试员工: %v", err)
		}
		rule, err := data.db.FinanceCommissionRule.Create().
			SetOrganizationID(org.ID).
			SetName("分配Schema方案-" + suffix).
			SetPersonnelRole("SALES").
			SetCalculationBasis("REALIZED_PROFIT").
			SetRatePercent("5").
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建测试方案: %v", err)
		}
		if _, err = data.db.FinanceCommissionRuleAssignment.Create().
			SetOrganizationID(org.ID).
			SetRuleID(rule.ID).
			SetEmployeeID(employee.ID).
			SetEffectiveFrom("2026-10-01").
			SetCreatedBy(employee.ID).
			Save(ctx); err != nil {
			t.Fatalf("创建测试分配: %v", err)
		}
		_, err = data.sqlDB.ExecContext(ctx, `DELETE FROM finance_commission_rules WHERE id = $1`, rule.ID)
		if err == nil {
			t.Fatalf("删除被分配引用的方案应被外键阻止")
		}
		// PostgreSQL 标识符上限 63 字节会截断超长约束名，断言可区分的稳定前缀。
		if !strings.Contains(err.Error(), "finance_commission_rule_assignments_finance_commission_rules") ||
			!strings.Contains(err.Error(), "23503") {
			t.Fatalf("错误应包含方案外键约束与违反码，实际: %v", err)
		}
	})

	// 旧角色规则停用迁移：恢复迁移前状态（无分配表、无标记列）后重放迁移文件，
	// 断言没有任何员工分配的启用规则被置为停用并标记历史只读。
	t.Run("停用迁移真实改写存量旧规则", func(t *testing.T) {
		suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		org, err := data.db.Organization.Create().
			SetCode("CRL-" + suffix).
			SetName("旧规则停用测试组织-" + suffix).
			SetKind("headquarters").
			SetBaseCurrency("CNY").
			Save(ctx)
		if err != nil {
			t.Fatalf("创建测试组织: %v", err)
		}
		// 恢复迁移前 Schema 状态并按完整迁移名删除记录，准备重放。
		if _, err = data.sqlDB.ExecContext(ctx,
			`DROP TABLE finance_commission_rule_assignments;
			     ALTER TABLE finance_commission_rules DROP COLUMN legacy_readonly;
			     DELETE FROM schema_migrations WHERE version = '20260918150000_commission_rule_assignments';`); err != nil {
			t.Fatalf("恢复迁移前状态失败: %v", err)
		}
		legacyRuleID := uuid.New()
		legacyWithDatesID := uuid.New()
		stmt := `INSERT INTO finance_commission_rules
			("id","created_at","updated_at","organization_id","name","personnel_role",
			 "calculation_basis","rate_percent","effective_from","effective_to","enabled","version")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,'SALES','REALIZED_PROFIT','5',$4,$5,true,1)`
		if _, err = data.sqlDB.ExecContext(ctx, stmt, legacyRuleID, org.ID, "旧规则-"+suffix, nil, nil); err != nil {
			t.Fatalf("插入无起始日旧规则失败: %v", err)
		}
		if _, err = data.sqlDB.ExecContext(ctx, stmt, legacyWithDatesID, org.ID, "旧规则带区间-"+suffix, "2026-01-01", nil); err != nil {
			t.Fatalf("插入带起始日旧规则失败: %v", err)
		}
		// 对真实迁移目录重放：其余文件已应用且校验和一致，仅补执行本迁移。
		if err = migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
			t.Fatalf("重放迁移失败: %v", err)
		}
		rows, err := data.sqlDB.QueryContext(ctx,
			`SELECT id, enabled, legacy_readonly FROM finance_commission_rules WHERE organization_id = $1`, org.ID)
		if err != nil {
			t.Fatalf("查询旧规则状态失败: %v", err)
		}
		defer rows.Close()
		seen := 0
		for rows.Next() {
			var id uuid.UUID
			var enabled, legacyReadonly bool
			if scanErr := rows.Scan(&id, &enabled, &legacyReadonly); scanErr != nil {
				t.Fatalf("读取旧规则状态失败: %v", scanErr)
			}
			seen++
			if enabled {
				t.Errorf("旧规则 %s 应被迁移停用", id)
			}
			if !legacyReadonly {
				t.Errorf("旧规则 %s 应被标记历史只读", id)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("遍历旧规则失败: %v", err)
		}
		if seen != 2 {
			t.Fatalf("应断言 2 条旧规则，实际 %d 条", seen)
		}
	})
}
