package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
)

// TestMapEntConstraintPostgres 经正式迁移和随机 Schema 验证实际索引名及 pgx/Ent 错误链。
// 不能用逻辑 Schema 长名称或错误文本子串代替 PostgreSQL 的完整 ConstraintName。
func TestMapEntConstraintPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	suffix := uuid.NewString()[:8]
	org := client.Organization.Create().SetCode("CSTR-" + suffix).SetName("约束映射测试公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	actor := client.User.Create().SetDisplayName("约束映射测试员工").SaveX(ctx)
	party := client.Partner.Create().SetOrganizationID(org.ID).SetCode("CSTR-P-" + suffix).SetLegalName("约束映射测试客商").SetNormalizedName("约束映射测试客商").SaveX(ctx)
	shippingLine := client.ShippingLine.Create().SetScacCode("CSTR").SetNameZh("约束映射测试船公司").SetNameEn("Constraint Test Shipping Line").SetCountryCode("CN").SaveX(ctx)

	assertConstraint := func(t *testing.T, err error, name string, domainErr error) {
		t.Helper()
		var pgErr *pgconn.PgError
		if !ent.IsConstraintError(err) || !errors.As(err, &pgErr) {
			t.Fatalf("应有真实 Ent/pgx 约束错误链: %v", err)
		}
		if pgErr.Code != "23505" || pgErr.ConstraintName != name {
			t.Fatalf("数据库约束=%q SQLSTATE=%q，期望 %q / 23505", pgErr.ConstraintName, pgErr.Code, name)
		}
		var actualName string
		// 显式转 text，避免 PostgreSQL name 参数自身截断而让长名称误通过。
		if queryErr := data.sqlDB.QueryRowContext(ctx,
			`SELECT indexname::text FROM pg_indexes WHERE schemaname = current_schema() AND indexname::text = $1`, name).Scan(&actualName); queryErr != nil {
			t.Fatalf("真实 catalog 缺少完整索引名 %q: %v", name, queryErr)
		}
		if got := mapEntConstraint(err, actualName, domainErr); got != domainErr {
			t.Fatalf("完整数据库约束名未映射到领域错误: %v", got)
		}
		if got := mapEntConstraint(err, name[:len(name)-1], domainErr); got != err {
			t.Fatalf("约束名前缀不得映射: %v", got)
		}
	}

	t.Run("账单幂等使用完整名称", func(t *testing.T) {
		createBill := func(no string) *ent.FinanceBillCreate {
			return withTestFinanceBillSettlementAccountSnapshot(client.FinanceBill.Create().
				SetOrganizationID(org.ID).SetBillNo(no).SetIdempotencyKey("cstr-bill-"+suffix).
				SetDirection(financebillent.DirectionRECEIVABLE).SetSettlementPartyID(party.ID).SetSettlementPartyName(party.LegalName).
				SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
				SetExchangeRateDate("2026-11-10").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").
				SetFeeCount(1).SetBillDate("2026-11-10"), uuid.New(), "CNY")
		}
		if _, err := createBill("CSTR-BILL-A").Save(ctx); err != nil {
			t.Fatal(err)
		}
		_, err := createBill("CSTR-BILL-B").Save(ctx)
		assertConstraint(t, err, "financebill_organization_id_idempotency_key", biz.ErrFinanceBillIdempotencyConflict)
		if got := mapEntConstraint(err, "idempotency", biz.ErrFinanceBillIdempotencyConflict); got != err {
			t.Fatalf("旧子串不应映射: %v", got)
		}
	})

	t.Run("月度申请使用实际截断名", func(t *testing.T) {
		createApplication := func() *ent.FinanceCommissionApplicationCreate {
			return client.FinanceCommissionApplication.Create().SetOrganizationID(org.ID).SetEmployeeID(actor.ID).
				SetApplicationMonth("2026-11").SetCoverageTo("2026-10-31").SetCommissionCount(0).SetBaseCurrency("CNY").
				SetTotalCommissionAmount("0").SetTotalCnyCommissionAmount("0").SetSubmittedAt(time.Now()).SetSubmittedBy(actor.ID)
		}
		if _, err := createApplication().Save(ctx); err != nil {
			t.Fatal(err)
		}
		_, err := createApplication().Save(ctx)
		assertConstraint(t, err, "financecommissionapplication_organization_id_employee_id_applic", biz.ErrCommissionApplicationConflict)
		if got := mapEntConstraint(err, "financecommissionapplication_organization_id_employee_id_application_month", biz.ErrCommissionApplicationConflict); got != err {
			t.Fatalf("逻辑长名称不应匹配数据库实际名: %v", got)
		}
	})

	t.Run("主单身份使用实际截断名", func(t *testing.T) {
		createMaster := func() *ent.SeaMasterBillCreate {
			return client.SeaMasterBill.Create().SetOrganizationID(org.ID).SetShippingLineID(shippingLine.ID).
				SetMasterNo("CSTR-MBL-" + suffix).SetNormalizedMasterNo("CSTR-MBL-" + suffix)
		}
		if _, err := createMaster().Save(ctx); err != nil {
			t.Fatal(err)
		}
		_, err := createMaster().Save(ctx)
		assertConstraint(t, err, "seamasterbill_organization_id_shipping_line_id_normalized_maste", biz.ErrSeaMasterBillExists)
		if got := mapEntConstraint(err, "seamasterbill_organization_id_shipping_line_id_normalized_master_no", biz.ErrSeaMasterBillExists); got != err {
			t.Fatalf("逻辑长名称不应匹配数据库实际名: %v", got)
		}
	})
}
