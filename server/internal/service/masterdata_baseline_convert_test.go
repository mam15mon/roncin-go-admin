package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// TestBaselineConvertersTolerateNilOrganization 验证 B 型基线行（OrganizationID 为 nil）
// 经过 service DTO 转换不发生空指针 panic，且组织字段按契约落空（port/airport 为
// 可空指针字段，exchange_rate 为普通字符串字段落空串）。
func TestBaselineConvertersTolerateNilOrganization(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)

	port := portToAPI(&biz.Port{ID: uuid.Must(uuid.NewV7()), UNLocode: "CNSHA", NameEN: "Shanghai", CountryCode: "CN", TransportModes: []string{"SEA"}, Source: "manual", SortOrder: 10, Enabled: true, CreatedAt: now, UpdatedAt: now})
	if port.OrganizationId != nil {
		t.Fatalf("基线港口 OrganizationId 应为空指针，实际 %v", *port.OrganizationId)
	}
	if port.UnLocode != "CNSHA" || !port.Enabled {
		t.Fatalf("基线港口契约字段不符: %+v", port)
	}

	airport := airportToAPI(&biz.Airport{ID: uuid.Must(uuid.NewV7()), IATACode: "PVG", NameEN: "Shanghai Pudong", CountryCode: "CN", Source: "manual", SortOrder: 10, Enabled: true, CreatedAt: now, UpdatedAt: now})
	if airport.OrganizationId != nil {
		t.Fatalf("基线机场 OrganizationId 应为空指针，实际 %v", *airport.OrganizationId)
	}
	if airport.IataCode != "PVG" || !airport.Enabled {
		t.Fatalf("基线机场契约字段不符: %+v", airport)
	}

	rate := exchangeRateToAPI(&biz.ExchangeRateSetting{ID: uuid.Must(uuid.NewV7()), FromCurrency: "USD", ToCurrency: "CNY", EffectiveFrom: "2026-09-14T00:00:00+08:00", Rate: decimal.RequireFromString("7.2"), IsActive: true, CreatedAt: now, UpdatedAt: now})
	if rate.OrganizationId != "" {
		t.Fatalf("基线汇率 OrganizationId 应为空串，实际 %q", rate.OrganizationId)
	}
	if rate.Rate != "7.20000000" || !rate.IsActive {
		t.Fatalf("基线汇率契约字段不符: %+v", rate)
	}
}
