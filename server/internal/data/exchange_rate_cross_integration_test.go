package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

func newCrossCalcOrgTree(t *testing.T, data *Data) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	for _, code := range []string{"USD", "CAD", "CNY"} {
		exists, err := data.db.Currency.Query().Where(currencyent.CodeEQ(code)).Exist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			if _, err := data.db.Currency.Create().SetCode(code).SetName(code).SetEnabled(true).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	suffix := uuid.NewString()[:8]
	system, err := data.db.Organization.Create().SetCode("XR-SYSTEM-" + suffix).SetName("汇率测试系统").SetKind(organizationent.KindSystem).SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	company, err := data.db.Organization.Create().SetCode("XR-COMPANY-" + suffix).SetName("汇率测试公司").SetKind(organizationent.KindCompany).SetBaseCurrency("CAD").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return system.ID, company.ID
}

func TestExchangeRateCompanyIsolationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	systemID, companyID := newCrossCalcOrgTree(t, data)
	week := time.Date(2026, 9, 7, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	other, err := data.db.Organization.Create().SetCode("XR-OTHER-" + uuid.NewString()[:8]).SetName("其他汇率测试公司").SetKind(organizationent.KindCompany).SetBaseCurrency("CAD").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := data.db.ExchangeRateSetting.Create().SetFromCurrency("USD").SetToCurrency("CAD").SetEffectiveFrom(week).SetRate("1.30000000").SetIsActive(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	repo := NewExchangeRateRepo(data)
	for _, id := range []uuid.UUID{companyID, other.ID} {
		if _, err := repo.ResolveRate(ctx, id, biz.OrderFeeReceivable, "USD", "CAD", "2026-09-12"); !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("只有公共行时公司 %s 必须报缺失，实际 %v", id, err)
		}
	}
	if _, err := repo.ResolveContext(ctx, systemID); !errors.Is(err, biz.ErrExchangeRateOrganizationInvalid) {
		t.Fatalf("系统工作台不能维护汇率，实际 %v", err)
	}
	for _, item := range []struct {
		id   uuid.UUID
		rate string
	}{{companyID, "1.5000"}, {other.ID, "1.6000"}} {
		if _, err := data.db.ExchangeRateSetting.Create().SetOrganizationID(item.id).SetFromCurrency("USD").SetToCurrency("CAD").SetEffectiveFrom(week).SetRate(item.rate).SetIsActive(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
		resolved, err := repo.ResolveRate(ctx, item.id, biz.OrderFeeReceivable, "USD", "CAD", "2026-09-12")
		if err != nil || resolved.Rate.StringFixed(4) != item.rate || resolved.Source != biz.ExchangeRateSourceWeekly {
			t.Fatalf("公司 %s 应取自身报价 %s，实际 %+v err=%v", item.id, item.rate, resolved, err)
		}
	}
	if _, err := repo.ResolveRate(ctx, companyID, biz.OrderFeeReceivable, "USD", "CAD", "2026-09-19"); !errors.Is(err, biz.ErrExchangeRateMissing) {
		t.Fatalf("不能继承上周公司行，实际 %v", err)
	}
}
