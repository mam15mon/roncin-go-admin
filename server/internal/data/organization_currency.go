package data

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

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
