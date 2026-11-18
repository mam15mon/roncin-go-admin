package data

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	ruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const paymentTermsMigrationVersion = "20261118180000"

// 旧字段仅在正式迁移前态通过 SQL 构造，不改现行 Schema 或约束。
func insertLegacyPaymentTermsRule(t *testing.T, d *Data, roleID uuid.UUID, method string, terms *int, credit *int64, currency *string, active bool, updated time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := d.sqlDB.ExecContext(t.Context(), `INSERT INTO partner_settlement_rules
 (id,created_at,updated_at,partner_role_id,statement_mode,settlement_method,settlement_currency,is_active,payment_terms_days,credit_limit_minor,credit_currency)
 VALUES($1,$2,$3,$4,'single',$5,'CNY',$6,$7,$8,$9)`, id, updated.Add(-time.Hour), updated, roleID, method, active, terms, credit, currency)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func applyPaymentTermsMigration(t *testing.T, d *Data) {
	t.Helper()
	if err := migration.Apply(t.Context(), d.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
}

func paymentTermsRowsSnapshot(t *testing.T, d *Data) string {
	t.Helper()
	var snapshot string
	if err := d.sqlDB.QueryRowContext(t.Context(), `SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM partner_settlement_rules r`).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestPartnerPaymentTermsMigrationMergeAndDeletePostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, paymentTermsMigrationVersion)
	t.Cleanup(cleanup)
	f := newCreditPartnerFixture(t, d)
	stamp := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	currency := "CNY"
	_, duplicateRole := f.newCreditPartner("合并")
	largest := insertLegacyPaymentTermsRule(t, d, duplicateRole, "by_ticket", new(45), new(int64(30000000)), &currency, true, stamp)
	kept := insertLegacyPaymentTermsRule(t, d, duplicateRole, "monthly", new(15), new(int64(10000000)), &currency, true, stamp.Add(time.Minute))
	_, inactiveRole := f.newCreditPartner("停用")
	inactive := insertLegacyPaymentTermsRule(t, d, inactiveRole, "by_ticket", new(90), nil, nil, false, stamp)
	_, supplierRole := f.newCreditPartner("供应商")
	if err := d.db.PartnerRole.UpdateOneID(supplierRole).SetRoleType("supplier").Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	supplier := insertLegacyPaymentTermsRule(t, d, supplierRole, "by_ticket", new(30), nil, nil, true, stamp)
	_, singleRole := f.newCreditPartner("单条不变")
	single := insertLegacyPaymentTermsRule(t, d, singleRole, "by_ticket", nil, new(int64(0)), &currency, true, stamp)
	var beforeSingle string
	if err := d.sqlDB.QueryRowContext(t.Context(), `SELECT (to_jsonb(r)-'statement_mode'-'settlement_method'-'settlement_day'-'settlement_cycle_days'-'settlement_base'-'settlement_currency'-'is_active')::text FROM partner_settlement_rules r WHERE id=$1`, single).Scan(&beforeSingle); err != nil {
		t.Fatal(err)
	}
	applyPaymentTermsMigration(t, d)
	got, err := d.db.PartnerSettlementRule.Get(t.Context(), kept)
	if err != nil || got.PaymentTermsDays == nil || *got.PaymentTermsDays != 45 || got.CreditLimitMinor == nil || *got.CreditLimitMinor != 30000000 || got.CreditCurrency == nil || *got.CreditCurrency != "CNY" || !got.UpdatedAt.Equal(stamp.Add(time.Minute)) {
		t.Fatalf("保留最近行并合并业务值: %+v %v", got, err)
	}
	for _, id := range []uuid.UUID{largest, inactive, supplier} {
		if _, err := d.db.PartnerSettlementRule.Get(t.Context(), id); !ent.IsNotFound(err) {
			t.Fatalf("旧行应已删除 %s: %v", id, err)
		}
	}
	var afterSingle string
	if err := d.sqlDB.QueryRowContext(t.Context(), `SELECT to_jsonb(r)::text FROM partner_settlement_rules r WHERE id=$1`, single).Scan(&afterSingle); err != nil {
		t.Fatal(err)
	}
	if beforeSingle != afterSingle {
		t.Fatalf("单条规则非废弃字段应原样保留: before=%s after=%s", beforeSingle, afterSingle)
	}
	var removedColumns int
	if err := d.sqlDB.QueryRowContext(t.Context(), `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='partner_settlement_rules' AND column_name IN ('statement_mode','settlement_method','settlement_day','settlement_cycle_days','settlement_base','settlement_currency','is_active')`).Scan(&removedColumns); err != nil || removedColumns != 0 {
		t.Fatalf("废弃列应删除: count=%d %v", removedColumns, err)
	}
	var indexDef string
	if err := d.sqlDB.QueryRowContext(t.Context(), `SELECT indexdef FROM pg_indexes WHERE schemaname=current_schema() AND indexname='partner_settlement_rule_role_key'`).Scan(&indexDef); err != nil || !strings.Contains(indexDef, "UNIQUE") || !strings.Contains(indexDef, "(partner_role_id)") {
		t.Fatalf("角色唯一索引应存在: %s %v", indexDef, err)
	}
	if _, err := d.db.PartnerSettlementRule.Create().SetPartnerRoleID(duplicateRole).Save(t.Context()); !ent.IsConstraintError(err) {
		t.Fatalf("第二条约定应被数据库拒绝: %v", err)
	}
	snapshot := paymentTermsRowsSnapshot(t, d)
	applyPaymentTermsMigration(t, d)
	if paymentTermsRowsSnapshot(t, d) != snapshot {
		t.Fatal("重复应用正式迁移不得改数据")
	}
}

func TestPartnerPaymentTermsMigrationZeroCreditAndNullTermsPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, paymentTermsMigrationVersion)
	t.Cleanup(cleanup)
	f := newCreditPartnerFixture(t, d)
	stamp := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	currency := "CNY"
	type expected struct {
		id       uuid.UUID
		amount   *int64
		currency *string
	}
	var cases []expected
	for _, tc := range []struct {
		name                         string
		older, newer                 *int64
		olderCurrency, newerCurrency *string
		want                         *int64
		wantCurrency                 *string
	}{
		{"零与正额", new(int64(20000000)), new(int64(0)), &currency, &currency, new(int64(20000000)), &currency},
		{"只有零与空保留零", nil, new(int64(0)), nil, &currency, new(int64(0)), &currency},
		{"只有零与空保留空", new(int64(0)), nil, &currency, nil, nil, nil},
	} {
		_, role := f.newCreditPartner(tc.name)
		insertLegacyPaymentTermsRule(t, d, role, "by_ticket", nil, tc.older, tc.olderCurrency, true, stamp)
		id := insertLegacyPaymentTermsRule(t, d, role, "monthly", nil, tc.newer, tc.newerCurrency, true, stamp.Add(time.Minute))
		cases = append(cases, expected{id, tc.want, tc.wantCurrency})
	}
	// 更新时间相同的保留行由 ID DESC 唯一确定。
	_, tieRole := f.newCreditPartner("同时间按ID保留")
	tieA := insertLegacyPaymentTermsRule(t, d, tieRole, "by_ticket", nil, nil, nil, true, stamp)
	tieB := insertLegacyPaymentTermsRule(t, d, tieRole, "monthly", nil, nil, nil, true, stamp)
	tieKept := tieA
	if tieB.String() > tieA.String() {
		tieKept = tieB
	}
	applyPaymentTermsMigration(t, d)
	for _, tc := range cases {
		got, err := d.db.PartnerSettlementRule.Get(t.Context(), tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if got.PaymentTermsDays != nil {
			t.Fatalf("全组空账期不能变为现结: %+v", got)
		}
		if (got.CreditLimitMinor == nil) != (tc.amount == nil) || (tc.amount != nil && *got.CreditLimitMinor != *tc.amount) || (got.CreditCurrency == nil) != (tc.currency == nil) || (tc.currency != nil && *got.CreditCurrency != *tc.currency) {
			t.Fatalf("零额度不参与最大正额度合并，空值仍保留: %+v", got)
		}
	}
	if got, err := d.db.PartnerSettlementRule.Query().Where(ruleent.PartnerRoleIDEQ(tieRole)).Only(t.Context()); err != nil || got.ID != tieKept {
		t.Fatalf("同时间必须按ID确定保留行: %+v %v", got, err)
	}
}

func TestPartnerPaymentTermsMigrationCreditGuardRollbackPostgres(t *testing.T) {
	for _, tc := range []struct {
		name     string
		currency *string
	}{{"正额度缺币种", nil}, {"正额度异币", new("USD")}} {
		t.Run(tc.name, func(t *testing.T) {
			d, cleanup := getIntegrationDataBeforeMigration(t, paymentTermsMigrationVersion)
			t.Cleanup(cleanup)
			f := newCreditPartnerFixture(t, d)
			stamp := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
			_, role := f.newCreditPartner("异常客户")
			insertLegacyPaymentTermsRule(t, d, role, "by_ticket", new(30), new(int64(10000)), tc.currency, true, stamp)
			_, removedRole := f.newCreditPartner("失败时不能先删")
			insertLegacyPaymentTermsRule(t, d, removedRole, "by_ticket", nil, nil, nil, false, stamp)
			before := paymentTermsRowsSnapshot(t, d)
			err := migration.Apply(t.Context(), d.sqlDB, filepath.Join("..", "..", "migrations"))
			var pgErr *pgconn.PgError
			if err == nil || !errors.As(err, &pgErr) || pgErr.Code != "P0001" || !strings.Contains(pgErr.Message, "1 条启用的客户结算规则") {
				t.Fatalf("币种守卫应明确列出异常数: %v", err)
			}
			if after := paymentTermsRowsSnapshot(t, d); after != before {
				t.Fatalf("守卫失败必须完整回滚: before=%s after=%s", before, after)
			}
			var column string
			if err := d.sqlDB.QueryRowContext(t.Context(), `SELECT column_name FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='partner_settlement_rules' AND column_name='is_active'`).Scan(&column); err != nil {
				t.Fatalf("失败时不得删列: %v", err)
			}
			var revision string
			err = d.sqlDB.QueryRowContext(t.Context(), `SELECT version FROM schema_migrations WHERE version=$1`, paymentTermsMigrationVersion+"_partner_payment_terms_single_rule").Scan(&revision)
			if err != sql.ErrNoRows {
				t.Fatalf("失败迁移不得记录已应用: %s %v", revision, err)
			}
		})
	}
}

func TestPartnerPaymentTermsMigrationGuardIgnoresRemovedRulesPostgres(t *testing.T) {
	d, cleanup := getIntegrationDataBeforeMigration(t, paymentTermsMigrationVersion)
	t.Cleanup(cleanup)
	f := newCreditPartnerFixture(t, d)
	stamp := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	_, inactiveRole := f.newCreditPartner("停用异常")
	insertLegacyPaymentTermsRule(t, d, inactiveRole, "by_ticket", nil, new(int64(10000)), nil, false, stamp)
	_, supplierRole := f.newCreditPartner("供应商异常")
	if err := d.db.PartnerRole.UpdateOneID(supplierRole).SetRoleType("supplier").Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	insertLegacyPaymentTermsRule(t, d, supplierRole, "by_ticket", nil, new(int64(10000)), new("USD"), true, stamp)
	applyPaymentTermsMigration(t, d)
	if count, err := d.db.PartnerSettlementRule.Query().Count(t.Context()); err != nil || count != 0 {
		t.Fatalf("非目标异常不拦迁移且规则应删除: %d %v", count, err)
	}
}

func TestPartnerPaymentTermsMigrationEmptyChainPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	if count, err := d.db.PartnerSettlementRule.Query().Count(context.Background()); err != nil || count != 0 {
		t.Fatalf("空库完整迁移链应成功: %d %v", count, err)
	}
}
