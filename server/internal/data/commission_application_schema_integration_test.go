package data

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCommissionApplicationSchemaPostgres 在真实 PostgreSQL 上验证月度提成申请
// Schema：隔离空库完整迁移链重放成功；CHECK、外键删除策略与唯一索引真实生效；
// 迁移为纯增量 DDL，既有提成与方案数据不受影响、不产生虚假申请行。跳过不算
// 通过，必须以 RONCIN_INTEGRATION_DATABASE_SOURCE 注入专用库并确认 PASS。
// 涉及 Schema 初始化，禁止 t.Parallel()。
func TestCommissionApplicationSchemaPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	org, err := data.db.Organization.Create().
		SetCode("FCA-" + suffix).
		SetName("申请Schema测试组织-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	employee, err := data.db.User.Create().
		SetUsername("fca_" + suffix).
		SetDisplayName("申请Schema员工-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试员工: %v", err)
	}
	rule, err := data.db.FinanceCommissionRule.Create().
		SetOrganizationID(org.ID).
		SetName("申请Schema方案-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis("REALIZED_PROFIT").
		SetRatePercent("5").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试方案: %v", err)
	}
	commission, err := data.db.FinanceCommission.Create().
		SetOrganizationID(org.ID).
		SetCommissionNo("FC-FCA-" + suffix).
		SetIdempotencyKey("fca-commission-" + suffix).
		SetEmployeeID(employee.ID).
		SetEmployeeName("申请Schema员工-" + suffix).
		SetCustomerCount(1).
		SetOrderCount(1).
		SetFeeCount(1).
		SetRuleID(rule.ID).
		SetRuleName("申请Schema方案-" + suffix).
		SetPersonnelRole("SALES").
		SetCalculationBasis("REALIZED_PROFIT").
		SetSourceFingerprint(strings.Repeat("a", 64)).
		SetBaseCurrency("CNY").
		SetRealizedRevenue("2000.00000000").
		SetAllocatedCost("800.00000000").
		SetRealizedProfit("1200.00000000").
		SetCommissionBaseAmount("1200.00000000").
		SetRatePercent("5.0000").
		SetCommissionAmount("60.00000000").
		SetCommissionDate("2026-08-15").
		SetCnyExchangeRate("1.00000000").
		SetCnyExchangeRateSource("BASE_CURRENCY").
		SetCnyExchangeRateDate("2026-08-15").
		SetCnyCommissionAmount("60.00000000").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试提成单: %v", err)
	}

	// 合法申请头 INSERT 模板：CHECK 探测复用，约束在外键触发器之前求值。
	applicationInsert := `INSERT INTO finance_commission_applications
		("id","created_at","updated_at","organization_id","employee_id","application_month",
		 "coverage_to","commission_count","base_currency","total_commission_amount",
		 "total_cny_commission_amount","submitted_at","submitted_by")
		VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,$4,$5,$6,$7,$8,$9,CURRENT_TIMESTAMP,$3)`
	applicationArgs := func(month string, count int) []any {
		return []any{uuid.New(), org.ID, employee.ID, month, "2026-08-31", count, "CNY", "60.00000000", "60.00000000"}
	}

	t.Run("CHECK 约束存在且包含关键条件", func(t *testing.T) {
		cases := []struct {
			name      string
			fragments []string
		}{
			{
				name:      "finance_commission_applications_status_check",
				fragments: []string{"status", "PENDING_REVIEW", "REJECTED", "APPROVED"},
			},
			{
				name:      "finance_commission_applications_commission_count_non_negative",
				fragments: []string{"commission_count", ">="},
			},
			{
				name:      "finance_commission_application_lines_commission_amount_non_negative",
				fragments: []string{"commission_amount", ">="},
			},
			{
				name:      "finance_commission_application_lines_cny_commission_amount_non_negative",
				fragments: []string{"cny_commission_amount", ">="},
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
			table      string
			column     string
			deleteRule string
		}{
			{table: "finance_commission_applications", column: "organization_id", deleteRule: "NO ACTION"},
			{table: "finance_commission_applications", column: "employee_id", deleteRule: "NO ACTION"},
			{table: "finance_commission_applications", column: "submitted_by", deleteRule: "NO ACTION"},
			{table: "finance_commission_applications", column: "decided_by", deleteRule: "SET NULL"},
			{table: "finance_commission_application_lines", column: "organization_id", deleteRule: "NO ACTION"},
			{table: "finance_commission_application_lines", column: "employee_id", deleteRule: "NO ACTION"},
			{table: "finance_commission_application_lines", column: "application_id", deleteRule: "NO ACTION"},
			{table: "finance_commission_application_lines", column: "commission_id", deleteRule: "NO ACTION"},
		}
		for _, tc := range cases {
			t.Run(tc.table+"."+tc.column, func(t *testing.T) {
				var deleteRule string
				err := data.sqlDB.QueryRowContext(ctx,
					`SELECT rc.delete_rule
					   FROM information_schema.referential_constraints rc
					   JOIN information_schema.key_column_usage kcu
					     ON kcu.constraint_name = rc.constraint_name
					    AND kcu.constraint_schema = rc.constraint_schema
					  WHERE rc.constraint_schema = current_schema()
					    AND kcu.table_name = $1
					    AND kcu.column_name = $2`,
					tc.table, tc.column).Scan(&deleteRule)
				if err != nil {
					t.Fatalf("查询外键 %s.%s 失败: %v", tc.table, tc.column, err)
				}
				if deleteRule != tc.deleteRule {
					t.Fatalf("外键 %s.%s delete_rule=%q，期望 %s", tc.table, tc.column, deleteRule, tc.deleteRule)
				}
			})
		}
	})

	t.Run("月度唯一键与提成全局唯一索引存在", func(t *testing.T) {
		cases := []struct {
			name      string
			unique    bool
			fragments []string
		}{
			{
				name:      "financecommissionapplication_organization_id_employee_id_application_month",
				unique:    true,
				fragments: []string{"organization_id", "employee_id", "application_month"},
			},
			{
				name:      "financecommissionapplicationline_application_id_commission_id",
				unique:    true,
				fragments: []string{"application_id", "commission_id"},
			},
			{
				name:      "financecommissionapplicationline_commission_id",
				unique:    true,
				fragments: []string{"commission_id"},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var definition string
				err := data.sqlDB.QueryRowContext(ctx,
					`SELECT indexdef FROM pg_indexes
					  WHERE schemaname = current_schema() AND indexname = $1`, tc.name).Scan(&definition)
				if err != nil {
					t.Fatalf("缺少索引 %s: %v", tc.name, err)
				}
				wantPrefix := "CREATE INDEX"
				if tc.unique {
					wantPrefix = "CREATE UNIQUE INDEX"
				}
				if !strings.Contains(definition, wantPrefix) {
					t.Fatalf("索引定义缺少 %q: %s", wantPrefix, definition)
				}
				for _, fragment := range tc.fragments {
					if !strings.Contains(definition, fragment) {
						t.Fatalf("索引定义缺少 %q: %s", fragment, definition)
					}
				}
			})
		}
	})

	t.Run("CHECK 拒绝非法状态与负数汇总", func(t *testing.T) {
		statusInsert := `INSERT INTO finance_commission_applications
			("id","created_at","updated_at","organization_id","employee_id","application_month",
			 "coverage_to","status","commission_count","base_currency","total_commission_amount",
			 "total_cny_commission_amount","submitted_at","submitted_by")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,'2026-08','2026-08-31',$4,
			 1,'CNY','60.00000000','60.00000000',CURRENT_TIMESTAMP,$3)`
		cases := []struct {
			name       string
			query      string
			args       []any
			constraint string
		}{
			{
				name:       "非法状态被拒绝",
				query:      statusInsert,
				args:       []any{uuid.New(), org.ID, employee.ID, "WORMHOLE"},
				constraint: "finance_commission_applications_status_check",
			},
			{
				name:       "负数笔数被拒绝",
				query:      applicationInsert,
				args:       applicationArgs("2026-08", -1),
				constraint: "finance_commission_applications_commission_count_non_negative",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := data.sqlDB.ExecContext(ctx, tc.query, tc.args...)
				assertCheckViolation(t, err, tc.constraint)
			})
		}
	})

	t.Run("同一组织员工提交月禁止第二张申请", func(t *testing.T) {
		if _, err := data.sqlDB.ExecContext(ctx, applicationInsert, applicationArgs("2026-08", 1)...); err != nil {
			t.Fatalf("首张申请应写入成功: %v", err)
		}
		duplicate := applicationArgs("2026-08", 1)
		duplicate[0] = uuid.New()
		_, err := data.sqlDB.ExecContext(ctx, applicationInsert, duplicate...)
		if err == nil {
			t.Fatalf("同组织同员工同提交月的第二张申请应被唯一索引拒绝")
		}
		if !strings.Contains(err.Error(), "financecommissionapplication_organization_id_employee_id") ||
			!strings.Contains(err.Error(), "23505") {
			// PostgreSQL 标识符上限 63 字节会截断超长约束名，断言可区分的稳定前缀。
			t.Fatalf("错误应包含月度唯一索引与违反码，实际: %v", err)
		}
		// 不同提交月允许各自成单：跨月累计不冲突。
		next := applicationArgs("2026-07", 1)
		next[0] = uuid.New()
		if _, err = data.sqlDB.ExecContext(ctx, applicationInsert, next...); err != nil {
			t.Fatalf("不同提交月应允许独立申请: %v", err)
		}
	})

	t.Run("一个提成事实至多进入一张申请", func(t *testing.T) {
		var applicationID uuid.UUID
		if err := data.sqlDB.QueryRowContext(ctx,
			`SELECT id FROM finance_commission_applications
			  WHERE organization_id = $1 AND employee_id = $2 AND application_month = '2026-08'`,
			org.ID, employee.ID).Scan(&applicationID); err != nil {
			t.Fatalf("定位已提交申请失败: %v", err)
		}
		lineInsert := `INSERT INTO finance_commission_application_lines
			("id","created_at","updated_at","organization_id","employee_id","application_id",
			 "commission_id","commission_date","personnel_role","rule_id","rule_name",
			 "calculation_basis","base_currency","commission_amount","cny_commission_amount",
			 "source_fingerprint")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,$4,$5,$6,'SALES',$7,$8,
			 'REALIZED_PROFIT','CNY','60.00000000','60.00000000',$9)`
		lineArgs := func(application uuid.UUID) []any {
			return []any{uuid.New(), org.ID, employee.ID, application, commission.ID,
				"2026-08-15", rule.ID, "申请Schema方案-" + suffix, strings.Repeat("a", 64)}
		}
		if _, err := data.sqlDB.ExecContext(ctx, lineInsert, lineArgs(applicationID)...); err != nil {
			t.Fatalf("首条申请明细应写入成功: %v", err)
		}
		// 同一提成事实换一张申请再挂一次：commission_id 全局唯一索引拒绝，
		// 被驳回也不回流公共池的口径由该索引兜底。
		otherApplication := uuid.New()
		if _, err := data.sqlDB.ExecContext(ctx, applicationInsert, otherApplication,
			org.ID, employee.ID, "2026-06", "2026-08-31", 1, "CNY", "60.00000000", "60.00000000"); err != nil {
			t.Fatalf("创建对照申请失败: %v", err)
		}
		_, duplicateLineErr := data.sqlDB.ExecContext(ctx, lineInsert, lineArgs(otherApplication)...)
		if duplicateLineErr == nil {
			t.Fatalf("同一提成事实进入第二张申请应被全局唯一索引拒绝")
		}
		if !strings.Contains(duplicateLineErr.Error(), "financecommissionapplicationline_commission_id") ||
			!strings.Contains(duplicateLineErr.Error(), "23505") {
			t.Fatalf("错误应包含提成全局唯一索引与违反码，实际: %v", duplicateLineErr)
		}
	})

	t.Run("删除被引用提成单被 NO ACTION 阻止", func(t *testing.T) {
		_, err := data.sqlDB.ExecContext(ctx, `DELETE FROM finance_commissions WHERE id = $1`, commission.ID)
		if err == nil {
			t.Fatalf("删除被申请明细引用的提成单应被外键阻止")
		}
		if !strings.Contains(err.Error(), "finance_commission_application_lines_finance_commissions") ||
			!strings.Contains(err.Error(), "23503") {
			t.Fatalf("错误应包含提成外键约束与违反码，实际: %v", err)
		}
	})

	// 迁移重放：恢复迁移前状态（两张新表不存在）后重放真实迁移文件，断言
	// 既有提成/方案数据不受影响、不产生任何虚假申请行（无历史回填）。
	t.Run("迁移重放不改既有数据且无虚假申请行", func(t *testing.T) {
		if _, err = data.sqlDB.ExecContext(ctx,
			`DROP TABLE finance_commission_application_lines;
			     DROP TABLE finance_commission_applications;
			     DELETE FROM schema_migrations WHERE version = '20260920110000_commission_applications';`); err != nil {
			t.Fatalf("恢复迁移前状态失败: %v", err)
		}
		content, err := os.ReadFile(filepath.Join("..", "..", "migrations",
			"20260920110000_commission_applications.sql"))
		if err != nil {
			t.Fatalf("读取迁移文件失败: %v", err)
		}
		if _, err = data.sqlDB.ExecContext(ctx, string(content)); err != nil {
			t.Fatalf("重放迁移失败: %v", err)
		}
		var commissionNo, commissionAmount, commissionDate string
		if err = data.sqlDB.QueryRowContext(ctx,
			`SELECT commission_no, commission_amount, commission_date
			   FROM finance_commissions WHERE id = $1`, commission.ID).
			Scan(&commissionNo, &commissionAmount, &commissionDate); err != nil {
			t.Fatalf("重放后既有提成单应保持不变: %v", err)
		}
		if commissionNo != "FC-FCA-"+suffix || commissionAmount != "60.00000000" || commissionDate != "2026-08-15" {
			t.Fatalf("重放后提成单快照被改写: %s %s %s", commissionNo, commissionAmount, commissionDate)
		}
		var ruleName string
		if err = data.sqlDB.QueryRowContext(ctx,
			`SELECT name FROM finance_commission_rules WHERE id = $1`, rule.ID).Scan(&ruleName); err != nil {
			t.Fatalf("重放后既有方案应保持不变: %v", err)
		}
		if ruleName != "申请Schema方案-"+suffix {
			t.Fatalf("重放后方案名称被改写: %s", ruleName)
		}
		for _, table := range []string{"finance_commission_applications", "finance_commission_application_lines"} {
			var count int
			if err = data.sqlDB.QueryRowContext(ctx,
				`SELECT count(*) FROM `+table).Scan(&count); err != nil {
				t.Fatalf("统计 %s 失败: %v", table, err)
			}
			if count != 0 {
				t.Fatalf("迁移不得回填虚假历史数据：%s 存在 %d 行", table, count)
			}
		}
	})
}
