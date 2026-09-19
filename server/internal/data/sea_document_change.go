package data

import (
	"context"
	"sort"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	seadocumentmodechangeeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentmodechangeevent"
	seadocumentvoideventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentvoidevent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
)

type seaDocumentChangeRepo struct{ data *Data }

func NewSeaDocumentChangeRepo(data *Data) biz.SeaDocumentChangeRepo {
	return &seaDocumentChangeRepo{data: data}
}

func (r *seaDocumentChangeRepo) ListMasterBillVersions(ctx context.Context, orgID, orderID uuid.UUID, page, pageSize int) ([]*biz.SeaDocumentVersion, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(orgID),
		seamasterbillorderlinkent.OrderIDEQ(orderID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Only(ctx)
	if err != nil {
		return nil, 0, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	query := client.SeaMasterBillVersion.Query().Where(
		seamasterbillversionent.OrganizationIDEQ(orgID),
		seamasterbillversionent.MasterBillIDEQ(link.MasterBillID),
	).WithShippingLine()
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.Order(ent.Desc(seamasterbillversionent.FieldVersionNo), ent.Desc(seamasterbillversionent.FieldID)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.SeaDocumentVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, masterVersionToBiz(row, orderID))
	}
	return result, total, nil
}

func (r *seaDocumentChangeRepo) ListHouseBillVersions(ctx context.Context, orgID, orderID, houseBillID uuid.UUID, page, pageSize int) ([]*biz.SeaDocumentVersion, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	exists, err := client.SeaHouseBill.Query().Where(seahousebillent.IDEQ(houseBillID), seahousebillent.OrganizationIDEQ(orgID), seahousebillent.OrderIDEQ(orderID)).Exist(ctx)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, biz.ErrSeaHouseBillNotFound
	}
	query := client.SeaHouseBillVersion.Query().Where(
		seahousebillversionent.OrganizationIDEQ(orgID),
		seahousebillversionent.OrderIDEQ(orderID),
		seahousebillversionent.HouseBillIDEQ(houseBillID),
	)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.Order(ent.Desc(seahousebillversionent.FieldVersionNo), ent.Desc(seahousebillversionent.FieldID)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.SeaDocumentVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, houseVersionToBiz(row))
	}
	return result, total, nil
}

func (r *seaDocumentChangeRepo) GetDocumentVersion(ctx context.Context, orgID, orderID, versionID uuid.UUID, documentType biz.SeaDocumentType) (*biz.SeaDocumentVersion, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	switch documentType {
	case biz.SeaDocumentTypeMasterBill:
		row, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.IDEQ(versionID), seamasterbillversionent.OrganizationIDEQ(orgID)).WithShippingLine().Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaDocumentVersionNotFound, nil)
		}
		hasLink, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID), seamasterbillorderlinkent.MasterBillIDEQ(row.MasterBillID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !hasLink {
			return nil, biz.ErrSeaDocumentVersionNotFound
		}
		return masterVersionToBiz(row, orderID), nil
	case biz.SeaDocumentTypeHouseBill:
		row, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.IDEQ(versionID), seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.OrderIDEQ(orderID)).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaDocumentVersionNotFound, nil)
		}
		return houseVersionToBiz(row), nil
	default:
		return nil, biz.ErrSeaDocumentInvalidArgument
	}
}

func (r *seaDocumentChangeRepo) ListDocumentEvents(ctx context.Context, orgID, orderID uuid.UUID, page, pageSize int) ([]*biz.SeaDocumentEvent, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	links, err := client.SeaMasterBillOrderLink.Query().Where(seamasterbillorderlinkent.OrganizationIDEQ(orgID), seamasterbillorderlinkent.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	mblIDs := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		mblIDs = append(mblIDs, link.MasterBillID)
	}
	events := make([]*biz.SeaDocumentEvent, 0)
	if len(mblIDs) > 0 {
		versions, err := client.SeaMasterBillVersion.Query().Where(seamasterbillversionent.OrganizationIDEQ(orgID), seamasterbillversionent.MasterBillIDIn(mblIDs...), seamasterbillversionent.SourceEQ(seamasterbillversionent.SourceAMENDMENT)).WithShippingLine().All(ctx)
		if err != nil {
			return nil, 0, err
		}
		for _, v := range versions {
			events = append(events, amendmentEventFromVersion(masterVersionToBiz(v, orderID)))
		}
	}
	hblVersions, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.OrganizationIDEQ(orgID), seahousebillversionent.OrderIDEQ(orderID), seahousebillversionent.SourceEQ(seahousebillversionent.SourceAMENDMENT)).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, v := range hblVersions {
		events = append(events, amendmentEventFromVersion(houseVersionToBiz(v)))
	}
	voidPredicates := []predicate.SeaDocumentVoidEvent{
		seadocumentvoideventent.OrganizationIDEQ(orgID),
		seadocumentvoideventent.OrderIDEQ(orderID),
	}
	if len(mblIDs) > 0 {
		voidPredicates[1] = seadocumentvoideventent.Or(
			seadocumentvoideventent.OrderIDEQ(orderID),
			seadocumentvoideventent.MasterBillIDIn(mblIDs...),
		)
	}
	voids, err := client.SeaDocumentVoidEvent.Query().Where(voidPredicates...).WithMasterBillVersion().WithHouseBillVersion().All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, v := range voids {
		events = append(events, voidEventToBiz(v))
	}
	modeChanges, err := client.SeaDocumentModeChangeEvent.Query().Where(
		seadocumentmodechangeeventent.OrganizationIDEQ(orgID),
		seadocumentmodechangeeventent.OrderIDEQ(orderID),
	).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, event := range modeChanges {
		events = append(events, modeChangeEventToBiz(event))
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].CreatedAt.Equal(events[j].CreatedAt) {
			return events[i].ID.String() > events[j].ID.String()
		}
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
	total := len(events)
	start := (page - 1) * pageSize
	if start >= total {
		return []*biz.SeaDocumentEvent{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return events[start:end], total, nil
}
