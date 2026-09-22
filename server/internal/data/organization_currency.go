package data

import (
	"context"
	"sort"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

var defaultMainstreamCurrencies = []string{"USD", "GBP", "EUR", "CHF", "AUD", "SGD", "HKD", "THB", "CNY"}

func defaultCompanyEnabledCurrencies(baseCurrency string) []string {
	m := make(map[string]struct{}, len(defaultMainstreamCurrencies)+1)
	for _, c := range defaultMainstreamCurrencies {
		m[c] = struct{}{}
	}
	if b := strings.ToUpper(strings.TrimSpace(baseCurrency)); b != "" {
		m[b] = struct{}{}
	}
	result := make([]string, 0, len(m))
	for c := range m {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

func resolveOrganizationBaseCurrency(ctx context.Context, client *ent.OrganizationClient, item *ent.Organization) (string, error) {
	current := item
	for {
		if current.BaseCurrency != nil {
			return *current.BaseCurrency, nil
		}
		if current.ParentID == nil {
			return "", biz.ErrAdminOrganizationCurrency
		}
		parent, err := client.Get(ctx, *current.ParentID)
		if err != nil {
			return "", err
		}
		current = parent
	}
}
