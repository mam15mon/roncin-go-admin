package data

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/administrativeregion"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
)

type referenceDataRepo struct {
	data *Data
}

func NewReferenceDataRepo(data *Data) biz.ReferenceDataRepo {
	return &referenceDataRepo{data: data}
}

func (r *referenceDataRepo) ListCurrencies(ctx context.Context) ([]*biz.Currency, error) {
	items, err := r.data.db.Currency.Query().
		Where(currency.EnabledEQ(true)).
		Order(ent.Asc(currency.FieldCode)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.Currency, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.Currency{
			ID: item.ID, Code: item.Code, Name: item.Name, Symbol: item.Symbol,
			MinorUnit: item.MinorUnit, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return result, nil
}

func (r *referenceDataRepo) ListAdministrativeRegions(ctx context.Context, query biz.AdministrativeRegionQuery) ([]*biz.AdministrativeRegion, error) {
	builder := r.data.db.AdministrativeRegion.Query().Where(administrativeregion.EnabledEQ(true))
	if query.Level != 0 {
		builder.Where(administrativeregion.LevelEQ(query.Level))
	}
	if query.ParentCode != nil {
		builder.Where(administrativeregion.ParentCodeEQ(*query.ParentCode))
	}
	if query.Keyword != "" {
		builder.Where(administrativeregion.Or(
			administrativeregion.NameContainsFold(query.Keyword),
			administrativeregion.CodeContains(query.Keyword),
		))
	}
	items, err := builder.Order(ent.Asc(administrativeregion.FieldCode)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AdministrativeRegion, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.AdministrativeRegion{
			ID: item.ID, Code: item.Code, Name: item.Name, Level: item.Level,
			ParentCode: item.ParentCode, RegionType: item.RegionType, Source: item.Source,
			SourceVersion: item.SourceVersion, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return result, nil
}

var _ biz.ReferenceDataRepo = (*referenceDataRepo)(nil)
