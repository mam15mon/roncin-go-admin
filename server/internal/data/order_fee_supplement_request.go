package data

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

func (r *orderFeeSupplementRepo) ListByOrder(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderFeeSupplementRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.OrganizationIDEQ(organizationID), orderfeesupplementent.OrderIDEQ(orderID)).
		WithRequestedByUser().
		WithDecidedByUser().
		Order(orderfeesupplementent.ByRequestedAt(), orderfeesupplementent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	partyIDs := make([]uuid.UUID, 0, len(items))
	partySeen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if _, exists := partySeen[item.SettlementPartyID]; !exists {
			partySeen[item.SettlementPartyID] = struct{}{}
			partyIDs = append(partyIDs, item.SettlementPartyID)
		}
	}
	partyNames := make(map[uuid.UUID]string, len(partyIDs))
	if len(partyIDs) > 0 {
		parties, partyErr := client.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID), partnerent.IDIn(partyIDs...)).All(ctx)
		if partyErr != nil {
			return nil, partyErr
		}
		for _, party := range parties {
			partyNames[party.ID] = party.LegalName
		}
	}
	result := make([]*biz.OrderFeeSupplementRequest, 0, len(items))
	for _, item := range items {
		converted, convertErr := supplementRequestToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		converted.Fee.SettlementPartyName = partyNames[item.SettlementPartyID]
		result = append(result, converted)
	}
	// 列表按发起时间倒序展示（最新申请在前）。
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].RequestedAt.Equal(result[j].RequestedAt) {
			return result[i].RequestedAt.After(result[j].RequestedAt)
		}
		return result[i].ID.String() > result[j].ID.String()
	})
	return result, nil
}
