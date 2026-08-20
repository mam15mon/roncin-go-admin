package data

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"

	"github.com/google/uuid"
)

type partnerRepo struct{ data *Data }

func NewPartnerRepo(data *Data) biz.PartnerRepo { return &partnerRepo{data: data} }

func (r *partnerRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.PartnerListOptions) (*biz.PartnerList, error) {
	query := r.data.db.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(partnerent.Or(partnerent.CodeContainsFold(options.Keyword), partnerent.NameContainsFold(options.Keyword), partnerent.ContactNameContainsFold(options.Keyword), partnerent.PhoneContainsFold(options.Keyword)))
	}
	if options.Type != "" {
		query.Where(partnerent.TypeEQ(partnerent.Type(options.Type)))
	}
	if options.Enabled != nil {
		query.Where(partnerent.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(partnerent.ByName()).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	partners := make([]*biz.Partner, 0, len(items))
	for _, item := range items {
		partners = append(partners, partnerToBiz(item))
	}
	return &biz.PartnerList{Items: partners, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *partnerRepo) Create(ctx context.Context, organizationID uuid.UUID, input *biz.Partner) (*biz.Partner, error) {
	exists, err := r.data.db.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID), partnerent.CodeEQ(input.Code)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, biz.ErrPartnerCodeExists
	}
	created, err := r.data.db.Partner.Create().
		SetOrganizationID(organizationID).
		SetCode(input.Code).
		SetName(input.Name).
		SetType(partnerent.Type(input.Type)).
		SetContactName(input.ContactName).
		SetPhone(input.Phone).
		SetEmail(input.Email).
		SetAddress(input.Address).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, biz.ErrPartnerCodeExists
		}
		return nil, err
	}
	return partnerToBiz(created), nil
}

func (r *partnerRepo) Update(ctx context.Context, organizationID, id uuid.UUID, input *biz.Partner) (*biz.Partner, error) {
	updated, err := r.data.db.Partner.UpdateOneID(id).
		Where(partnerent.OrganizationIDEQ(organizationID)).
		SetName(input.Name).
		SetType(partnerent.Type(input.Type)).
		SetContactName(input.ContactName).
		SetPhone(input.Phone).
		SetEmail(input.Email).
		SetAddress(input.Address).
		SetEnabled(input.Enabled).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerNotFound
		}
		return nil, err
	}
	return partnerToBiz(updated), nil
}

func partnerToBiz(item *ent.Partner) *biz.Partner {
	return &biz.Partner{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Code:           item.Code,
		Name:           item.Name,
		Type:           biz.PartnerType(item.Type),
		ContactName:    item.ContactName,
		Phone:          item.Phone,
		Email:          item.Email,
		Address:        item.Address,
		Enabled:        item.Enabled,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}
