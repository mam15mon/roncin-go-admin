package data

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestFeeSupplementSchemaPostgres 在真实 PostgreSQL 上验证锁后费用补录与提成
// 复算快照 Schema 的约束真实生效：CHECK 拒绝非法组合、来源外键删除策略为
// NO ACTION、来源唯一索引与组织级幂等唯一索引存在。跳过不算通过，必须以
// RONCIN_INTEGRATION_DATABASE_SOURCE 注入专用库并确认 PASS。
func TestFeeSupplementSchemaPostgres(t *testing.T) {
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
				name:      "order_fee_supplement_requests_lock_basis_check",
				fragments: []string{"lock_basis", "business_lock_generation", "financial_lock_evidence_version", "financial_lock_evidence_hash", "financial_lock_net_amount_snapshot", "> 0"},
			},
			{
				name:      "order_fee_supplement_requests_status_check",
				fragments: []string{"PENDING", "APPROVED", "REJECTED", "WITHDRAWN"},
			},
			{
				name:      "commission_adjustment_source_supplement_check",
				fragments: []string{"LOCKED_FEE_SUPPLEMENT", "source_fee_supplement_request_id", "IS NOT NULL", "IS NULL"},
			},
			{
				name:      "commission_adjustment_source_type_check",
				fragments: []string{"MANUAL", "VERIFICATION_REVERSAL", "NETTING_REVERSAL", "LOCKED_FEE_SUPPLEMENT"},
			},
			{
				name:      "finance_commission_lines_snapshot_consistency_check",
				fragments: []string{"READY", "UNAVAILABLE", "NATIVE", "MIGRATED", "total_receivable_snapshot", "total_payable_snapshot", "snapshot_backfill_version", "snapshot_evidence_hash", "snapshot_unavailable_reason_code"},
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

	t.Run("来源外键删除策略为 NO ACTION", func(t *testing.T) {
		cases := []struct {
			table  string
			column string
		}{
			{table: "order_fees", column: "supplement_request_id"},
			{table: "finance_commission_adjustments", column: "source_fee_supplement_request_id"},
			{table: "order_fee_supplement_requests", column: "order_id"},
			{table: "order_fee_supplement_requests", column: "organization_id"},
			{table: "order_fee_supplement_requests", column: "requested_by"},
		}
		for _, tc := range cases {
			t.Run(tc.table+"_"+tc.column, func(t *testing.T) {
				var deleteRule string
				err := data.sqlDB.QueryRowContext(ctx,
					`SELECT rc.delete_rule
					   FROM information_schema.referential_constraints rc
					   JOIN information_schema.key_column_usage kcu
					     ON kcu.constraint_name = rc.constraint_name
					    AND kcu.constraint_schema = rc.constraint_schema
					  WHERE rc.constraint_schema = current_schema()
					    AND kcu.table_name = $1 AND kcu.column_name = $2`,
					tc.table, tc.column).Scan(&deleteRule)
				if err != nil {
					if err == sql.ErrNoRows {
						t.Fatalf("缺少外键 %s.%s", tc.table, tc.column)
					}
					t.Fatalf("查询外键 %s.%s 失败: %v", tc.table, tc.column, err)
				}
				if deleteRule != "NO ACTION" {
					t.Fatalf("外键 %s.%s delete_rule=%q，期望 NO ACTION", tc.table, tc.column, deleteRule)
				}
			})
		}
	})

	t.Run("唯一索引存在", func(t *testing.T) {
		cases := []struct {
			table     string
			fragments []string
		}{
			{
				table:     "order_fee_supplement_requests",
				fragments: []string{"organization_id", "idempotency_key"},
			},
			{
				table:     "order_fees",
				fragments: []string{"supplement_request_id"},
			},
			{
				table:     "finance_commission_adjustments",
				fragments: []string{"commission_id", "order_id", "source_type", "source_fee_supplement_request_id"},
			},
		}
		for _, tc := range cases {
			t.Run(tc.table, func(t *testing.T) {
				rows, err := data.sqlDB.QueryContext(ctx,
					`SELECT indexdef FROM pg_indexes
					  WHERE schemaname = current_schema() AND tablename = $1 AND indexdef LIKE 'CREATE UNIQUE INDEX%'`,
					tc.table)
				if err != nil {
					t.Fatalf("查询 %s 唯一索引失败: %v", tc.table, err)
				}
				defer rows.Close()
				matched := false
				for rows.Next() {
					var definition string
					if scanErr := rows.Scan(&definition); scanErr != nil {
						t.Fatalf("读取索引定义失败: %v", scanErr)
					}
					complete := true
					for _, fragment := range tc.fragments {
						if !strings.Contains(definition, fragment) {
							complete = false
							break
						}
					}
					if complete {
						matched = true
						break
					}
				}
				if err := rows.Err(); err != nil {
					t.Fatalf("遍历索引失败: %v", err)
				}
				if !matched {
					t.Fatalf("%s 缺少覆盖列 %v 的唯一索引", tc.table, tc.fragments)
				}
			})
		}
	})

	t.Run("CHECK 拒绝非法锁依据组合", func(t *testing.T) {
		stmt := `INSERT INTO order_fee_supplement_requests
			("id","created_at","updated_at","organization_id","order_id","lock_basis",
			 "idempotency_key","request_fingerprint","direction","fee_code","fee_name",
			 "settlement_party_id","billing_unit","quantity","unit_price","total_amount",
			 "net_amount","tax_amount","currency","exchange_rate","exchange_rate_source",
			 "exchange_rate_date","base_currency","base_currency_amount","expense_date",
			 "reason","requested_by","requested_at",
			 "business_lock_generation","financial_lock_evidence_version",
			 "financial_lock_evidence_hash","financial_lock_net_amount_snapshot")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,$4,
				$5,$6,'PAYABLE','TRUCK','集成测试拖车费',
				$7,'CTR',1,120,120,
				120,0,'CNY',1,'WEEKLY',
				'2026-09-18','CNY',120,'2026-09-18',
				'漏录拖车成本',$8,CURRENT_TIMESTAMP,
				$9,$10,$11,$12)`
		invalidHash := strPtr("a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8")
		zeroNet := strPtr("0")
		positiveNet := strPtr("120")
		cases := []struct {
			name       string
			lockBasis  string
			generation *uint64
			version    *string
			hash       *string
			net        *string
		}{
			{name: "BUSINESS 缺少业务锁代次", lockBasis: "BUSINESS"},
			{name: "BUSINESS 业务锁代次为零", lockBasis: "BUSINESS", generation: uint64Ptr(0)},
			{name: "BUSINESS 不得携带财务证据", lockBasis: "BUSINESS", generation: uint64Ptr(2),
				version: strPtr("FINANCIAL_LOCK_EVIDENCE_V1"), hash: invalidHash, net: positiveNet},
			{name: "FINANCIAL 缺少证据哈希", lockBasis: "FINANCIAL",
				version: strPtr("FINANCIAL_LOCK_EVIDENCE_V1"), net: positiveNet},
			{name: "FINANCIAL 净额为零", lockBasis: "FINANCIAL",
				version: strPtr("FINANCIAL_LOCK_EVIDENCE_V1"), hash: invalidHash, net: zeroNet},
			{name: "FINANCIAL 不得携带业务锁代次", lockBasis: "FINANCIAL", generation: uint64Ptr(1),
				version: strPtr("FINANCIAL_LOCK_EVIDENCE_V1"), hash: invalidHash, net: positiveNet},
			{name: "BOTH 缺少业务锁代次", lockBasis: "BOTH",
				version: strPtr("FINANCIAL_LOCK_EVIDENCE_V1"), hash: invalidHash, net: positiveNet},
			{name: "BOTH 缺少净额快照", lockBasis: "BOTH", generation: uint64Ptr(2),
				version: strPtr("FINANCIAL_LOCK_EVIDENCE_V1"), hash: invalidHash},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := data.sqlDB.ExecContext(ctx, stmt,
					uuid.New(), uuid.New(), uuid.New(), tc.lockBasis,
					"it-"+uuid.NewString(), "fp-v1:"+uuid.NewString(),
					uuid.New(), uuid.New(),
					tc.generation, tc.version, tc.hash, tc.net)
				assertCheckViolation(t, err, "order_fee_supplement_requests_lock_basis_check")
			})
		}

		t.Run("非法状态取值被拒绝", func(t *testing.T) {
			invalidStatus := `INSERT INTO order_fee_supplement_requests
				("id","created_at","updated_at","organization_id","order_id","lock_basis",
				 "business_lock_generation","idempotency_key","request_fingerprint","direction",
				 "fee_code","fee_name","settlement_party_id","billing_unit","quantity",
				 "unit_price","total_amount","net_amount","tax_amount","currency",
				 "exchange_rate","exchange_rate_source","exchange_rate_date","base_currency",
				 "base_currency_amount","expense_date","reason","requested_by","requested_at","status")
				VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,'BUSINESS',
					1,$4,$5,'PAYABLE',
					'TRUCK','集成测试拖车费',$6,'CTR',1,
					120,120,120,0,'CNY',
					1,'WEEKLY','2026-09-18','CNY',
					120,'2026-09-18','漏录拖车成本',$7,CURRENT_TIMESTAMP,'UNKNOWN')`
			_, err := data.sqlDB.ExecContext(ctx, invalidStatus,
				uuid.New(), uuid.New(), uuid.New(),
				"it-"+uuid.NewString(), "fp-v1:"+uuid.NewString(),
				uuid.New(), uuid.New())
			assertCheckViolation(t, err, "order_fee_supplement_requests_status_check")
		})
	})

	t.Run("CHECK 强制调整来源与补录申请双向对应", func(t *testing.T) {
		stmt := `INSERT INTO finance_commission_adjustments
			("id","created_at","updated_at","organization_id","commission_id","order_id",
			 "adjustment_no","idempotency_key","commission_no","order_no","employee_id",
			 "employee_name","source_type","direction","status","base_currency","amount",
			 "reason","version","source_fee_supplement_request_id")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,$4,
				$5,$6,'C-1','SO-1',$7,
				'集成测试',$8,'DECREASE','DRAFT','CNY',10,
				'锁后补录冲减',1,$9)`
		t.Run("LOCKED_FEE_SUPPLEMENT 缺少补录申请关联被拒绝", func(t *testing.T) {
			_, err := data.sqlDB.ExecContext(ctx, stmt,
				uuid.New(), uuid.New(), uuid.New(), uuid.New(),
				"ADJ-IT-"+uuid.NewString()[:8], "it-adj-"+uuid.NewString(),
				uuid.New(), "LOCKED_FEE_SUPPLEMENT", nil)
			assertCheckViolation(t, err, "commission_adjustment_source_supplement_check")
		})
		t.Run("人工来源携带补录申请关联被拒绝", func(t *testing.T) {
			_, err := data.sqlDB.ExecContext(ctx, stmt,
				uuid.New(), uuid.New(), uuid.New(), uuid.New(),
				"ADJ-IT-"+uuid.NewString()[:8], "it-adj-"+uuid.NewString(),
				uuid.New(), "MANUAL", uuid.New())
			assertCheckViolation(t, err, "commission_adjustment_source_supplement_check")
		})
	})

	t.Run("CHECK 保证提成复算快照一致性", func(t *testing.T) {
		stmt := `INSERT INTO finance_commission_lines
			("id","created_at","updated_at","organization_id","commission_id","order_id",
			 "order_no","order_date","customer_id","customer_code","customer_name",
			 "personnel_assignment_id","personnel_organization_id","personnel_assigned_at",
			 "fee_count","employee_id","employee_name","personnel_role","calculation_basis",
			 "base_currency","realized_revenue","allocated_cost","realized_profit",
			 "commission_base_amount","rate_percent","commission_amount",
			 "total_receivable_snapshot","total_payable_snapshot","snapshot_status",
			 "snapshot_source","snapshot_backfill_version","snapshot_evidence_hash",
			 "snapshot_unavailable_reason_code")
			VALUES ($1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,$2,$3,$4,
				$5,'2026-09-18',$6,'CUS-1','集成测试客户',
				$7,$8,CURRENT_TIMESTAMP,
				1,$9,'集成测试','SALES','REALIZED_PROFIT',
				'CNY',100,40,60,
				60,5,3,
				$10,$11,$12,$13,$14,$15,$16)`
		cases := []struct {
			name string
			args []any
		}{
			{
				name: "READY 缺少应付分母被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					strPtr("200"), nil, strPtr("READY"), strPtr("NATIVE"), nil, nil, nil},
			},
			{
				name: "READY 携带不可用原因码被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					strPtr("200"), strPtr("80"), strPtr("READY"), strPtr("NATIVE"), nil, nil, strPtr("SNAPSHOT_MISSING")},
			},
			{
				name: "READY+MIGRATED 缺少回填算法版本被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					strPtr("200"), strPtr("80"), strPtr("READY"), strPtr("MIGRATED"), nil, strPtr("hash"), nil},
			},
			{
				name: "READY+NATIVE 伪填证据哈希被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					strPtr("200"), strPtr("80"), strPtr("READY"), strPtr("NATIVE"), nil, strPtr("hash"), nil},
			},
			{
				name: "UNAVAILABLE 伪填估算分母被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					strPtr("200"), nil, strPtr("UNAVAILABLE"), nil, nil, nil, strPtr("SNAPSHOT_NOT_REPRODUCIBLE")},
			},
			{
				name: "UNAVAILABLE 缺少原因码被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					nil, nil, strPtr("UNAVAILABLE"), nil, nil, nil, nil},
			},
			{
				name: "READY 完全缺少分母被拒绝",
				args: []any{uuid.New(), uuid.New(), uuid.New(), uuid.New(), "SO-IT-" + uuid.NewString()[:8],
					uuid.New(), uuid.New(), uuid.New(), uuid.New(),
					nil, nil, strPtr("READY"), strPtr("NATIVE"), nil, nil, nil},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := data.sqlDB.ExecContext(ctx, stmt, tc.args...)
				assertCheckViolation(t, err, "finance_commission_lines_snapshot_consistency_check")
			})
		}
	})
}

func strPtr(s string) *string { return &s }

func uint64Ptr(v uint64) *uint64 { return &v }

// assertCheckViolation 断言写入因目标 CHECK 约束失败：CHECK 在外键触发器之前
// 求值，夹具无需父表数据即可验证约束真实生效。
func assertCheckViolation(t *testing.T, err error, constraint string) {
	t.Helper()
	if err == nil {
		t.Fatalf("预期触发 CHECK 约束 %s，但写入成功", constraint)
	}
	if !strings.Contains(err.Error(), constraint) {
		t.Fatalf("错误应包含约束名 %s，实际: %v", constraint, err)
	}
}
