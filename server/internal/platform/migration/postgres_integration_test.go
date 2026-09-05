package migration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	entmigrate "github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresColdStartMigration(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		t.Fatal("DATABASE_SOURCE 不能为空")
	}

	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("连接 PostgreSQL 失败: %v", err)
	}

	schemaName := "roncin_migration_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("创建临时 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("删除临时 Schema 失败: %v", err)
		}
	})
	// pg_trgm 扩展安装在 public，隔离业务表的同时保留扩展运算符类可见性。
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
		t.Fatalf("切换临时 Schema 失败: %v", err)
	}

	dir := filepath.Join("..", "..", "..", "migrations")
	if err := Apply(ctx, db, dir); err != nil {
		t.Fatalf("冷启动迁移失败: %v", err)
	}
	if err := Apply(ctx, db, dir); err != nil {
		t.Fatalf("重复执行迁移失败: %v", err)
	}

	for _, table := range entmigrate.Tables {
		var exists bool
		query := fmt.Sprintf(`SELECT to_regclass('%s.%s') IS NOT NULL`, schemaName, table.Name)
		if err := db.QueryRowContext(ctx, query).Scan(&exists); err != nil {
			t.Fatalf("检查表 %s 失败: %v", table.Name, err)
		}
		if !exists {
			t.Errorf("迁移后缺少表 %s", table.Name)
		}
	}

	expectedIndexes := map[string][]string{
		"financeinvoice_org_tax_invoice_no":    {"UNIQUE", "organization_id", "tax_invoice_no", "tax_invoice_no IS NOT NULL"},
		"financeinvoice_org_red_invoice_no":    {"UNIQUE", "organization_id", "red_invoice_no", "red_invoice_no IS NOT NULL"},
		"finance_bill_lines_active_fee_unique": {"UNIQUE", "order_fee_id", "active = true"},
		"verification_allocation_pair_unique":  {"UNIQUE", "verification_id", "cashflow_id", "bill_id"},
	}
	for name, fragments := range expectedIndexes {
		var definition string
		if err := db.QueryRowContext(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname = $1 AND indexname = $2`, schemaName, name).Scan(&definition); err != nil {
			t.Errorf("迁移后缺少索引 %s: %v", name, err)
			continue
		}
		for _, fragment := range fragments {
			if !strings.Contains(definition, fragment) {
				t.Errorf("索引 %s 定义缺少 %q: %s", name, fragment, definition)
			}
		}
	}
}

func TestPostgresSeaDocumentStage2Migration(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		t.Fatal("DATABASE_SOURCE 不能为空")
	}

	const stage1Migration = "20260902120000_sea_export_mbl_foundation.sql"
	const stage2Revision = "20260902140000_sea_export_document_content"
	fullDir := filepath.Join("..", "..", "..", "migrations")
	stage1Dir := t.TempDir()
	entries, err := os.ReadDir(fullDir)
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() > stage1Migration {
			continue
		}
		content, err := os.ReadFile(filepath.Join(fullDir, entry.Name()))
		if err != nil {
			t.Fatalf("读取阶段 1 迁移 %s 失败: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(stage1Dir, entry.Name()), content, 0o600); err != nil {
			t.Fatalf("复制阶段 1 迁移 %s 失败: %v", entry.Name(), err)
		}
	}

	newIsolatedDB := func(t *testing.T) (*sql.DB, string) {
		t.Helper()
		db, err := sql.Open("pgx", source)
		if err != nil {
			t.Fatalf("打开 PostgreSQL 失败: %v", err)
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			t.Fatalf("连接 PostgreSQL 失败: %v", err)
		}
		schemaName := "roncin_sea_document_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		quotedSchema := `"` + schemaName + `"`
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
			_ = db.Close()
			t.Fatalf("创建临时 Schema 失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
			_, _ = db.ExecContext(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
			_ = db.Close()
			t.Fatalf("切换临时 Schema 失败: %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cleanupCancel()
			if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
				t.Errorf("删除临时 Schema 失败: %v", err)
			}
			_ = db.Close()
		})
		return db, schemaName
	}

	t.Run("从阶段1真实迁移到阶段2并核对结构", func(t *testing.T) {
		db, schemaName := newIsolatedDB(t)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := Apply(ctx, db, stage1Dir); err != nil {
			t.Fatalf("建立阶段 1 数据库状态失败: %v", err)
		}

		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = 'sea_master_bill_order_links'
			  AND column_name = 'document_structure'
		)`, schemaName).Scan(&exists); err != nil {
			t.Fatalf("检查阶段 1 结构失败: %v", err)
		}
		if exists {
			t.Fatal("阶段 1 状态不应提前包含 document_structure")
		}

		if err := Apply(ctx, db, fullDir); err != nil {
			t.Fatalf("阶段 2 真实迁移失败: %v", err)
		}
		if err := Apply(ctx, db, fullDir); err != nil {
			t.Fatalf("阶段 2 重复执行失败: %v", err)
		}

		for _, column := range []struct {
			table string
			name  string
		}{
			{table: "sea_master_bill_order_links", name: "document_structure"},
			{table: "sea_master_bills", name: "shipper_text"},
			{table: "sea_master_bills", name: "clauses"},
		} {
			if err := db.QueryRowContext(ctx, `SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
			)`, schemaName, column.table, column.name).Scan(&exists); err != nil || !exists {
				t.Errorf("阶段 2 缺少列 %s.%s: exists=%v err=%v", column.table, column.name, exists, err)
			}
		}
		if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, schemaName+".sea_house_bills").Scan(&exists); err != nil || !exists {
			t.Errorf("阶段 2 缺少 sea_house_bills: exists=%v err=%v", exists, err)
		}

		var constraintCount int
		if err := db.QueryRowContext(ctx, `SELECT count(*)
			FROM pg_constraint c
			JOIN pg_namespace n ON n.oid = c.connamespace
			WHERE n.nspname = $1 AND c.conname IN (
			  'sea_master_bill_order_links_document_structure_check',
			  'sea_house_bills_issuer_check',
			  'sea_house_bills_package_count_check',
			  'sea_house_bills_gross_weight_kg_check',
			  'sea_house_bills_volume_cbm_check'
			)`, schemaName).Scan(&constraintCount); err != nil || constraintCount != 5 {
			t.Errorf("阶段 2 CHECK 约束数量应为 5，实际 %d，err=%v", constraintCount, err)
		}

		var indexCount int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_indexes
			WHERE schemaname = $1 AND indexname IN (
			  'idx_sea_house_bills_self_org_unique',
			  'idx_sea_house_bills_partner_unique'
			)`, schemaName).Scan(&indexCount); err != nil || indexCount != 2 {
			t.Errorf("阶段 2 条件唯一索引数量应为 2，实际 %d，err=%v", indexCount, err)
		}

		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)`, stage2Revision).Scan(&exists); err != nil || !exists {
			t.Errorf("阶段 2 迁移记录不存在: exists=%v err=%v", exists, err)
		}
	})

	t.Run("存在SE数据时真实迁移原子拒绝", func(t *testing.T) {
		db, schemaName := newIsolatedDB(t)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := Apply(ctx, db, stage1Dir); err != nil {
			t.Fatalf("建立阶段 1 数据库状态失败: %v", err)
		}

		orgID, partnerID, orderID := uuid.New(), uuid.New(), uuid.New()
		if _, err := db.ExecContext(ctx, `INSERT INTO organizations
			(id, created_at, updated_at, code, name, kind, base_currency)
			VALUES ($1, now(), now(), $2, '迁移测试组织', 'company', 'CNY')`,
			orgID, "MIG-ORG-"+orgID.String()[:8]); err != nil {
			t.Fatalf("插入迁移测试组织失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO partners
			(id, created_at, updated_at, organization_id, code, legal_name, normalized_name)
			VALUES ($1, now(), now(), $2, $3, '迁移测试客户', '迁移测试客户')`,
			partnerID, orgID, "MIG-PARTNER-"+partnerID.String()[:8]); err != nil {
			t.Fatalf("插入迁移测试客户失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO orders
			(id, created_at, updated_at, organization_id, order_no, customer_id,
			 business_type, trade_direction, trade_term, payment_term,
			 flow_status, termination_status, closure_status, version)
			VALUES ($1, now(), now(), $2, $3, $4,
			 'SE', 'export', 'FOB', 'PREPAID', 'DRAFT', 'ACTIVE', 'OPEN', 1)`,
			orderID, orgID, "MIG-ORDER-"+orderID.String()[:8], partnerID); err != nil {
			t.Fatalf("插入迁移测试 SE 订单失败: %v", err)
		}

		err := Apply(ctx, db, fullDir)
		if err == nil || !strings.Contains(err.Error(), "orders 存在 SE 业务数据") {
			t.Fatalf("存在 SE 数据时应拒绝阶段 2 迁移，实际错误: %v", err)
		}

		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = 'sea_master_bill_order_links'
			  AND column_name = 'document_structure'
		)`, schemaName).Scan(&exists); err != nil {
			t.Fatalf("检查失败迁移后的列失败: %v", err)
		}
		if exists {
			t.Error("失败迁移不得留下 document_structure 列")
		}
		if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, schemaName+".sea_house_bills").Scan(&exists); err != nil {
			t.Fatalf("检查失败迁移后的表失败: %v", err)
		}
		if exists {
			t.Error("失败迁移不得留下 sea_house_bills 表")
		}
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)`, stage2Revision).Scan(&exists); err != nil {
			t.Fatalf("检查失败迁移记录失败: %v", err)
		}
		if exists {
			t.Error("失败迁移不得写入阶段 2 revision")
		}
	})
}

func TestPostgresSeaExportCargoAllocationStage3Migration(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		t.Fatal("DATABASE_SOURCE 不能为空")
	}

	const stage2Migration = "20260902140000_sea_export_document_content.sql"
	const stage3Revision = "20260903100000_sea_export_cargo_allocation"
	fullDir := filepath.Join("..", "..", "..", "migrations")
	stage2Dir := t.TempDir()
	entries, err := os.ReadDir(fullDir)
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() > stage2Migration {
			continue
		}
		content, err := os.ReadFile(filepath.Join(fullDir, entry.Name()))
		if err != nil {
			t.Fatalf("读取阶段 2 迁移 %s 失败: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(stage2Dir, entry.Name()), content, 0o600); err != nil {
			t.Fatalf("复制阶段 2 迁移 %s 失败: %v", entry.Name(), err)
		}
	}

	newIsolatedDB := func(t *testing.T) (*sql.DB, string) {
		t.Helper()
		db, err := sql.Open("pgx", source)
		if err != nil {
			t.Fatalf("打开 PostgreSQL 失败: %v", err)
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			t.Fatalf("连接 PostgreSQL 失败: %v", err)
		}
		schemaName := "roncin_sea_cargo_alloc_mig_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		quotedSchema := `"` + schemaName + `"`
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
			_ = db.Close()
			t.Fatalf("创建临时 Schema 失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
			_, _ = db.ExecContext(ctx, "DROP SCHEMA "+quotedSchema+" CASCADE")
			_ = db.Close()
			t.Fatalf("切换临时 Schema 失败: %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cleanupCancel()
			if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
				t.Errorf("删除临时 Schema 失败: %v", err)
			}
			_ = db.Close()
		})
		return db, schemaName
	}

	t.Run("从阶段2真实迁移到阶段3并核对结构与索引", func(t *testing.T) {
		db, schemaName := newIsolatedDB(t)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := Apply(ctx, db, stage2Dir); err != nil {
			t.Fatalf("建立阶段 2 数据库状态失败: %v", err)
		}

		if err := Apply(ctx, db, fullDir); err != nil {
			t.Fatalf("应用阶段 3 迁移失败: %v", err)
		}

		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)`, stage3Revision).Scan(&exists); err != nil || !exists {
			t.Fatalf("未找到阶段 3 迁移记录: %v", err)
		}

		// 检查 sea_cargo_allocations 表
		if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, schemaName+".sea_cargo_allocations").Scan(&exists); err != nil || !exists {
			t.Fatalf("缺少 sea_cargo_allocations 表: %v", err)
		}

		// 检查 numeric 精度
		var weightUdt, volUdt string
		var weightPrecision, weightScale, volPrecision, volScale int
		if err := db.QueryRowContext(ctx, `SELECT udt_name, numeric_precision, numeric_scale FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = 'sea_cargo_allocations' AND column_name = 'gross_weight_kg'`, schemaName).Scan(&weightUdt, &weightPrecision, &weightScale); err != nil {
			t.Fatalf("查询 gross_weight_kg 列类型失败: %v", err)
		}
		if weightUdt != "numeric" || weightPrecision != 18 || weightScale != 3 {
			t.Errorf("gross_weight_kg 精度期望 numeric(18,3), 实际 %s(%d,%d)", weightUdt, weightPrecision, weightScale)
		}

		if err := db.QueryRowContext(ctx, `SELECT udt_name, numeric_precision, numeric_scale FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = 'sea_cargo_allocations' AND column_name = 'volume_cbm'`, schemaName).Scan(&volUdt, &volPrecision, &volScale); err != nil {
			t.Fatalf("查询 volume_cbm 列类型失败: %v", err)
		}
		if volUdt != "numeric" || volPrecision != 18 || volScale != 6 {
			t.Errorf("volume_cbm 精度期望 numeric(18,6), 实际 %s(%d,%d)", volUdt, volPrecision, volScale)
		}

		for _, check := range []struct {
			table, column    string
			precision, scale int
		}{
			{"order_cargo_items", "gross_weight_kg", 18, 3},
			{"order_cargo_items", "net_weight_kg", 18, 3},
			{"order_cargo_items", "volume_cbm", 18, 6},
			{"order_containers", "gross_weight_kg", 18, 3},
			{"order_containers", "volume_cbm", 18, 6},
		} {
			var dataType string
			var precision, scale int
			if err := db.QueryRowContext(ctx, `SELECT udt_name, numeric_precision, numeric_scale FROM information_schema.columns
				WHERE table_schema = $1 AND table_name = $2 AND column_name = $3`, schemaName, check.table, check.column).Scan(&dataType, &precision, &scale); err != nil {
				t.Fatalf("查询 %s.%s 类型失败: %v", check.table, check.column, err)
			}
			if dataType != "numeric" || precision != check.precision || scale != check.scale {
				t.Errorf("%s.%s 期望 numeric(%d,%d)，实际 %s(%d,%d)", check.table, check.column, check.precision, check.scale, dataType, precision, scale)
			}
		}

		var linkColumns int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema = $1
			AND table_name = 'sea_master_bill_order_links' AND column_name IN
			('cargo_allocation_status','cargo_allocation_version','cargo_allocation_confirmed_at','cargo_allocation_confirmed_by')`, schemaName).Scan(&linkColumns); err != nil || linkColumns != 4 {
			t.Errorf("Link 箱货分配状态列不完整，count=%d err=%v", linkColumns, err)
		}

		var constraintCount int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.table_constraints WHERE table_schema = $1 AND constraint_name IN (
			'sea_master_bill_order_links_cargo_allocation_status_check',
			'sea_master_bill_order_links_users_cargo_allocation_confirmed_by',
			'order_cargo_items_organizations_order_cargo_items',
			'order_containers_organizations_order_containers',
			'sea_cargo_allocations_package_count_check',
			'sea_cargo_allocations_gross_weight_kg_check',
			'sea_cargo_allocations_volume_cbm_check',
			'sea_cargo_allocations_organizations_sea_cargo_allocations',
			'sea_cargo_allocations_orders_sea_cargo_allocations',
			'sea_cargo_allocations_links_cargo_allocations',
			'sea_cargo_allocations_cargo_items_cargo_allocations',
			'sea_cargo_allocations_house_bills_cargo_allocations',
			'sea_cargo_allocations_containers_cargo_allocations'
		)`, schemaName).Scan(&constraintCount); err != nil || constraintCount != 13 {
			t.Errorf("阶段 3 关键 CHECK/FK 不完整，count=%d err=%v", constraintCount, err)
		}

		// 检查 order_containers 移除 shipping_document_id 并增加了 package_count
		var hasDocID bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = 'order_containers' AND column_name = 'shipping_document_id'
		)`, schemaName).Scan(&hasDocID); err != nil {
			t.Fatalf("检查 order_containers.shipping_document_id 失败: %v", err)
		}
		if hasDocID {
			t.Error("order_containers 不应再包含 shipping_document_id 列")
		}

		var hasPkgCount bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = 'order_containers' AND column_name = 'package_count'
		)`, schemaName).Scan(&hasPkgCount); err != nil {
			t.Fatalf("检查 order_containers.package_count 失败: %v", err)
		}
		if !hasPkgCount {
			t.Error("order_containers 应包含 package_count 列")
		}

		// 检查局部唯一索引
		var hasNoCntrIdx, hasCntrIdx bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_indexes WHERE schemaname = $1 AND tablename = 'sea_cargo_allocations' AND indexname = 'idx_sea_cargo_allocations_no_cntr_unique'
		)`, schemaName).Scan(&hasNoCntrIdx); err != nil || !hasNoCntrIdx {
			t.Errorf("缺少 idx_sea_cargo_allocations_no_cntr_unique 索引: %v", err)
		}
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_indexes WHERE schemaname = $1 AND tablename = 'sea_cargo_allocations' AND indexname = 'idx_sea_cargo_allocations_cntr_unique'
		)`, schemaName).Scan(&hasCntrIdx); err != nil || !hasCntrIdx {
			t.Errorf("缺少 idx_sea_cargo_allocations_cntr_unique 索引: %v", err)
		}
	})

	t.Run("存在SE货物明细数据时真实迁移原子拒绝", func(t *testing.T) {
		db, schemaName := newIsolatedDB(t)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := Apply(ctx, db, stage2Dir); err != nil {
			t.Fatalf("建立阶段 2 数据库状态失败: %v", err)
		}

		orgID := uuid.New()
		partnerID := uuid.New()
		orderID := uuid.New()
		cargoItemID := uuid.New()

		if _, err := db.ExecContext(ctx, `INSERT INTO organizations
			(id, created_at, updated_at, code, name, kind, base_currency)
			VALUES ($1, now(), now(), $2, '迁移测试组织', 'company', 'CNY')`,
			orgID, "MIG-ORG-"+orgID.String()[:8]); err != nil {
			t.Fatalf("插入迁移测试组织失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO partners
			(id, created_at, updated_at, organization_id, code, legal_name, normalized_name)
			VALUES ($1, now(), now(), $2, $3, '迁移测试客户', '迁移测试客户')`,
			partnerID, orgID, "MIG-PARTNER-"+partnerID.String()[:8]); err != nil {
			t.Fatalf("插入迁移测试客户失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO orders
			(id, created_at, updated_at, organization_id, order_no, customer_id,
			 business_type, trade_direction, trade_term, payment_term,
			 flow_status, termination_status, closure_status, version)
			VALUES ($1, now(), now(), $2, $3, $4,
			 'SE', 'export', 'FOB', 'PREPAID', 'DRAFT', 'ACTIVE', 'OPEN', 1)`,
			orderID, orgID, "MIG-ORDER-"+orderID.String()[:8], partnerID); err != nil {
			t.Fatalf("插入迁移测试 SE 订单失败: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO order_cargo_items
			(id, created_at, updated_at, order_id, cargo_name, package_count, gross_weight_kg, volume_cbm)
			VALUES ($1, now(), now(), $2, '测试货物', 10, 500, 2.5)`,
			cargoItemID, orderID); err != nil {
			t.Fatalf("插入迁移测试 cargo item 失败: %v", err)
		}

		err := Apply(ctx, db, fullDir)
		if err == nil || !strings.Contains(err.Error(), "海运箱货分配迁移已停止：") {
			t.Fatalf("存在 SE 历史数据时应拒绝阶段 3 迁移，实际错误: %v", err)
		}

		var exists bool
		if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, schemaName+".sea_cargo_allocations").Scan(&exists); err != nil {
			t.Fatalf("检查失败迁移后的表失败: %v", err)
		}
		if exists {
			t.Error("失败迁移不得留下 sea_cargo_allocations 表")
		}
		if err := db.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM schema_migrations WHERE version = $1
		)`, stage3Revision).Scan(&exists); err != nil {
			t.Fatalf("检查失败迁移记录失败: %v", err)
		}
		if exists {
			t.Error("失败迁移不得写入阶段 3 revision")
		}
	})
}

func TestPostgresSeaDocumentChangeMigrationFromVersioningBaseline(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		t.Fatal("DATABASE_SOURCE 不能为空")
	}
	const baselineMigration = "20260904120000_sea_export_document_versioning.sql"
	const changeRevision = "20260904160000_sea_document_change_idempotency"
	fullDir := filepath.Join("..", "..", "..", "migrations")
	baselineDir := t.TempDir()
	entries, err := os.ReadDir(fullDir)
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() > baselineMigration {
			continue
		}
		content, err := os.ReadFile(filepath.Join(fullDir, entry.Name()))
		if err != nil {
			t.Fatalf("读取版本化基线迁移 %s 失败: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(baselineDir, entry.Name()), content, 0o600); err != nil {
			t.Fatalf("复制版本化基线迁移 %s 失败: %v", entry.Name(), err)
		}
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	schemaName := "roncin_document_change_mig_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("创建临时 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, _ = db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE")
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
		t.Fatalf("切换临时 Schema 失败: %v", err)
	}
	if err := Apply(ctx, db, baselineDir); err != nil {
		t.Fatalf("建立版本化基线失败: %v", err)
	}
	if err := Apply(ctx, db, fullDir); err != nil {
		t.Fatalf("从版本化基线升级单证变更迁移失败: %v", err)
	}
	if err := Apply(ctx, db, fullDir); err != nil {
		t.Fatalf("单证变更迁移重复执行失败: %v", err)
	}
	var revisionExists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, changeRevision).Scan(&revisionExists); err != nil || !revisionExists {
		t.Fatalf("单证变更迁移记录不存在: exists=%v err=%v", revisionExists, err)
	}
	var notNullable bool
	if err := db.QueryRowContext(ctx, `SELECT is_nullable = 'NO' FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'sea_document_void_events' AND column_name = 'order_id'`, schemaName).Scan(&notNullable); err != nil || !notNullable {
		t.Fatalf("作废事件 order_id 未收紧为 NOT NULL: value=%v err=%v", notNullable, err)
	}
	var constraintCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace WHERE n.nspname = $1 AND c.conname IN ('sea_document_void_events_document_type_check','sea_document_void_events_status_check','sea_document_void_events_sea_master_bill_versions_previous_void_events','sea_document_void_events_sea_house_bill_versions_previous_void_events')`, schemaName).Scan(&constraintCount); err != nil || constraintCount != 4 {
		t.Fatalf("单证变更迁移 CHECK/FK 不完整: count=%d err=%v", constraintCount, err)
	}
	var switchIndexDefinition string
	if err := db.QueryRowContext(ctx, `SELECT indexdef FROM pg_indexes WHERE schemaname = $1 AND indexname = 'seahousebillswitchevent_chain_id_sequence'`, schemaName).Scan(&switchIndexDefinition); err != nil || !strings.Contains(switchIndexDefinition, "UNIQUE") {
		t.Fatalf("Switch 链序号唯一索引缺失: definition=%q err=%v", switchIndexDefinition, err)
	}
}

func TestPostgresUniversalOrderLockMigrationFromSEBaseline(t *testing.T) {
	if os.Getenv("RONCIN_POSTGRES_MIGRATION_TEST") != "1" {
		t.Skip("设置 RONCIN_POSTGRES_MIGRATION_TEST=1 后运行真实 PostgreSQL 迁移测试")
	}
	source := os.Getenv("DATABASE_SOURCE")
	if source == "" {
		t.Fatal("DATABASE_SOURCE 不能为空")
	}

	const baselineMigration = "20260904160000_sea_document_change_idempotency.sql"
	const universalLockRevision = "20260905120000_universal_order_lock"
	fullDir := filepath.Join("..", "..", "..", "migrations")
	baselineDir := t.TempDir()
	entries, err := os.ReadDir(fullDir)
	if err != nil {
		t.Fatalf("读取迁移目录失败: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() > baselineMigration {
			continue
		}
		content, err := os.ReadFile(filepath.Join(fullDir, entry.Name()))
		if err != nil {
			t.Fatalf("读取全业务订单锁基线迁移 %s 失败: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(baselineDir, entry.Name()), content, 0o600); err != nil {
			t.Fatalf("复制全业务订单锁基线迁移 %s 失败: %v", entry.Name(), err)
		}
	}

	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatalf("打开 PostgreSQL 失败: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("连接 PostgreSQL 失败: %v", err)
	}

	schemaName := "roncin_universal_lock_mig_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := `"` + schemaName + `"`
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("创建临时 Schema 失败: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if _, err := db.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("删除临时 Schema 失败: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+quotedSchema+", public"); err != nil {
		t.Fatalf("切换临时 Schema 失败: %v", err)
	}
	if err := Apply(ctx, db, baselineDir); err != nil {
		t.Fatalf("建立全业务订单锁升级基线失败: %v", err)
	}

	orgID, userID, partnerID := uuid.New(), uuid.New(), uuid.New()
	orderID, executionID, masterBillID := uuid.New(), uuid.New(), uuid.New()
	masterVersionID, lockRecordID, unlockRequestID := uuid.New(), uuid.New(), uuid.New()
	orderNo := "MIG-SE-" + orderID.String()[:8]
	if _, err := db.ExecContext(ctx, `INSERT INTO organizations
		(id, created_at, updated_at, code, name, kind, base_currency)
		VALUES ($1, now(), now(), $2, '全业务订单锁迁移组织', 'company', 'CNY')`,
		orgID, "MIG-ORG-"+orgID.String()[:8]); err != nil {
		t.Fatalf("插入迁移测试组织失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO users
		(id, created_at, updated_at, username, display_name, password_hash)
		VALUES ($1, now(), now(), $2, '迁移锁定人', 'migration-test')`,
		userID, "mig-user-"+userID.String()[:8]); err != nil {
		t.Fatalf("插入迁移测试用户失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO partners
		(id, created_at, updated_at, organization_id, code, legal_name, normalized_name)
		VALUES ($1, now(), now(), $2, $3, '迁移测试客户', '迁移测试客户')`,
		partnerID, orgID, "MIG-PARTNER-"+partnerID.String()[:8]); err != nil {
		t.Fatalf("插入迁移测试客户失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO orders
		(id, created_at, updated_at, organization_id, order_no, customer_id,
		 business_type, trade_direction, trade_term, payment_term,
		 flow_status, termination_status, closure_status, version)
		VALUES ($1, now(), now(), $2, $3, $4,
		 'SE', 'export', 'FOB', 'PREPAID', 'DRAFT', 'ACTIVE', 'OPEN', 2)`,
		orderID, orgID, orderNo, partnerID); err != nil {
		t.Fatalf("插入迁移测试 SE 订单失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sea_transport_executions
		(id, created_at, updated_at, organization_id)
		VALUES ($1, now(), now(), $2)`, executionID, orgID); err != nil {
		t.Fatalf("插入迁移测试运输执行失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sea_master_bills
		(id, created_at, updated_at, organization_id, issuer_partner_id,
		 transport_execution_id, master_no, normalized_master_no)
		VALUES ($1, now(), now(), $2, $3, $4, 'MIG-MBL-001', 'MIG-MBL-001')`,
		masterBillID, orgID, partnerID, executionID); err != nil {
		t.Fatalf("插入迁移测试 MBL 失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sea_master_bill_versions
		(id, created_at, version_no, source_entity_version, issuer_partner_id,
		 transport_execution_id, master_no, normalized_master_no, status,
		 content_hash, source, master_bill_id, organization_id)
		VALUES ($1, now(), 1, 1, $2, $3, 'MIG-MBL-001', 'MIG-MBL-001', 'CONFIRMED',
		 $4, 'ORDER_LOCK', $5, $6)`,
		masterVersionID, partnerID, executionID, strings.Repeat("a", 64), masterBillID, orgID); err != nil {
		t.Fatalf("插入迁移测试 MBL 版本失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO order_lock_records
		(id, created_at, order_no, generation, locked_at, order_version_at_lock,
		 idempotency_key, request_fingerprint, order_id, organization_id, locked_by,
		 master_bill_id, master_bill_version_id)
		VALUES ($1, now(), $2, 1, now(), 1, $3, $4, $5, $6, $7, $8, $9)`,
		lockRecordID, orderNo, "lock-"+lockRecordID.String(), "lock-fingerprint",
		orderID, orgID, userID, masterBillID, masterVersionID); err != nil {
		t.Fatalf("插入升级前 SE 锁定周期失败: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO order_unlock_requests
		(id, created_at, order_no, lock_generation, requested_at, reason,
		 expected_order_version, idempotency_key, request_fingerprint, route, status,
		 dingtalk_process_instance_id, dingtalk_process_code,
		 order_id, lock_record_id, organization_id, requested_by)
		VALUES ($1, now(), $2, 1, now(), '升级前解锁申请', 2, $3, $4,
		 'DINGTALK_APPROVAL', 'PENDING_APPROVAL', 'migration-process-instance',
		 'migration-process-code', $5, $6, $7, $8)`,
		unlockRequestID, orderNo, "unlock-"+unlockRequestID.String(), "unlock-fingerprint",
		orderID, lockRecordID, orgID, userID); err != nil {
		t.Fatalf("插入升级前 SE 解锁请求失败: %v", err)
	}

	if err := Apply(ctx, db, fullDir); err != nil {
		t.Fatalf("升级全业务订单锁迁移失败: %v", err)
	}
	if err := Apply(ctx, db, fullDir); err != nil {
		t.Fatalf("全业务订单锁迁移重复执行失败: %v", err)
	}

	var lockBusinessType, requestBusinessType, processInstanceID, processCode string
	if err := db.QueryRowContext(ctx, `SELECT business_type FROM order_lock_records WHERE id = $1`, lockRecordID).Scan(&lockBusinessType); err != nil {
		t.Fatalf("读取迁移后锁定周期失败: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT business_type, dingtalk_process_instance_id, dingtalk_process_code
		FROM order_unlock_requests WHERE id = $1`, unlockRequestID).
		Scan(&requestBusinessType, &processInstanceID, &processCode); err != nil {
		t.Fatalf("读取迁移后解锁请求失败: %v", err)
	}
	if lockBusinessType != "SE" || requestBusinessType != "SE" {
		t.Fatalf("升级前 SE 事实回填异常: lock=%q request=%q", lockBusinessType, requestBusinessType)
	}
	if processInstanceID != "migration-process-instance" || processCode != "migration-process-code" {
		t.Fatalf("升级不得改写既有 OA 关联: instance=%q code=%q", processInstanceID, processCode)
	}

	for _, column := range []struct {
		table, name string
		nullable    bool
	}{
		{table: "order_lock_records", name: "business_type", nullable: false},
		{table: "order_unlock_requests", name: "business_type", nullable: false},
		{table: "order_lock_records", name: "master_bill_id", nullable: true},
		{table: "order_lock_records", name: "master_bill_version_id", nullable: true},
	} {
		var nullable string
		if err := db.QueryRowContext(ctx, `SELECT is_nullable FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 AND column_name = $3`,
			schemaName, column.table, column.name).Scan(&nullable); err != nil {
			t.Fatalf("读取列元数据 %s.%s 失败: %v", column.table, column.name, err)
		}
		if got := nullable == "YES"; got != column.nullable {
			t.Errorf("列 %s.%s nullable=%v，期望 %v", column.table, column.name, got, column.nullable)
		}
	}

	var checkValidated bool
	var checkDefinition string
	if err := db.QueryRowContext(ctx, `SELECT c.convalidated, pg_get_constraintdef(c.oid)
		FROM pg_constraint c
		JOIN pg_namespace n ON n.oid = c.connamespace
		WHERE n.nspname = $1 AND c.conname = 'order_lock_records_business_type_document_refs_check'`,
		schemaName).Scan(&checkValidated, &checkDefinition); err != nil {
		t.Fatalf("读取全业务锁文档引用 CHECK 失败: %v", err)
	}
	for _, fragment := range []string{"business_type", "master_bill_id", "master_bill_version_id", "LAND", "RAIL"} {
		if !strings.Contains(checkDefinition, fragment) {
			t.Errorf("全业务锁文档引用 CHECK 缺少 %q: %s", fragment, checkDefinition)
		}
	}
	if !checkValidated {
		t.Error("全业务锁文档引用 CHECK 必须立即验证")
	}

	for _, constraintName := range []string{
		"order_lock_records_sea_master_bills_lock_records",
		"order_lock_records_sea_master_bill_versions_lock_records",
	} {
		var deleteRule string
		if err := db.QueryRowContext(ctx, `SELECT delete_rule FROM information_schema.referential_constraints
			WHERE constraint_schema = $1 AND constraint_name = $2`, schemaName, constraintName).Scan(&deleteRule); err != nil {
			t.Fatalf("读取历史快照外键 %s 失败: %v", constraintName, err)
		}
		if deleteRule != "NO ACTION" {
			t.Errorf("历史快照外键 %s delete_rule=%q，期望 NO ACTION", constraintName, deleteRule)
		}
	}

	var revisionChecksum string
	if err := db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, universalLockRevision).Scan(&revisionChecksum); err != nil {
		t.Fatalf("全业务订单锁迁移记录不存在: %v", err)
	}
	if len(revisionChecksum) != 64 {
		t.Errorf("全业务订单锁迁移 checksum 长度=%d，期望 64", len(revisionChecksum))
	}
}
