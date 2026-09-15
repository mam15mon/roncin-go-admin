package data

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
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


func resolveHeadquartersOrganizationID(ctx context.Context, client *ent.OrganizationClient, organizationID uuid.UUID) (uuid.UUID, error) {
	currentID := organizationID
	for {
		item, err := client.Query().Where(organizationent.IDEQ(currentID), organizationent.EnabledEQ(true)).Only(ctx)
		if err != nil {
			return uuid.Nil, err
		}
		if item.ParentID == nil {
			if item.Kind != organizationent.KindHeadquarters {
				return uuid.Nil, fmt.Errorf("组织 %s 的根节点不是总部", organizationID)
			}
			return item.ID, nil
		}
		currentID = *item.ParentID
	}
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
