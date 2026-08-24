package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

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
