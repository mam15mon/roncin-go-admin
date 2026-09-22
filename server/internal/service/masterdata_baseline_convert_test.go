package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// 验证公共地点及共享参考汇率经过 DTO 转换保留业务字段。
func TestBaselineConvertersTolerateNilOrganization(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)

	port := portToAPI(&biz.Port{ID: uuid.Must(uuid.NewV7()), UNLocode: "CNSHA", NameEN: "Shanghai", CountryCode: "CN", TransportModes: []string{"SEA"}, Source: "manual", SortOrder: 10, Enabled: true, CreatedAt: now, UpdatedAt: now})
	if port.UnLocode != "CNSHA" || !port.Enabled {
		t.Fatalf("基线港口契约字段不符: %+v", port)
	}

	airport := airportToAPI(&biz.Airport{ID: uuid.Must(uuid.NewV7()), IATACode: "PVG", NameEN: "Shanghai Pudong", CountryCode: "CN", Source: "manual", SortOrder: 10, Enabled: true, CreatedAt: now, UpdatedAt: now})
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
