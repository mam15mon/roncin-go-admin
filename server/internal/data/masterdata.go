package data

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	entschema "github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"

	"github.com/google/uuid"
)

type masterDataRepo struct{ data *Data }

func NewMasterDataRepo(data *Data) biz.MasterDataRepo { return &masterDataRepo{data: data} }

func (r *masterDataRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.MasterDataListOptions) (*biz.MasterDataList, error) {
	query := r.data.db.MasterDataItem.Query().Where(masterdataent.OrganizationIDEQ(organizationID))
	if options.Kind != "" {
		query.Where(masterdataent.KindEQ(masterdataent.Kind(options.Kind)))
	}
	if options.Keyword != "" {
		query.Where(masterdataent.Or(masterdataent.CodeContainsFold(options.Keyword), masterdataent.NameContainsFold(options.Keyword), masterdataent.NameEnContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(masterdataent.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(masterdataent.ByKind(), masterdataent.BySortOrder(), masterdataent.ByCode()).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.MasterDataList{Items: masterDataItemsToBiz(items), Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *masterDataRepo) ListEnabled(ctx context.Context, organizationID uuid.UUID) ([]*biz.MasterDataItem, error) {
	items, err := r.data.db.MasterDataItem.Query().Where(masterdataent.OrganizationIDEQ(organizationID), masterdataent.EnabledEQ(true)).Order(masterdataent.ByKind(), masterdataent.BySortOrder(), masterdataent.ByCode()).All(ctx)
	if err != nil {
		return nil, err
	}
	return masterDataItemsToBiz(items), nil
}

func (r *masterDataRepo) Create(ctx context.Context, organizationID uuid.UUID, input *biz.MasterDataItem) (*biz.MasterDataItem, error) {
	create := r.data.db.MasterDataItem.Create().SetOrganizationID(organizationID).SetKind(masterdataent.Kind(input.Kind)).SetCode(input.Code).SetName(input.Name).SetNillableNameEn(input.NameEN).SetNillableParentCode(input.ParentCode).SetNillableTeuFactor(input.TEUFactor).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).SetAttributes(masterDataAttributesToEnt(input.Attributes))
	created, err := create.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, biz.ErrMasterDataCodeExists
		}
		return nil, err
	}
	return masterDataItemToBiz(created), nil
}

func (r *masterDataRepo) Update(ctx context.Context, organizationID, id uuid.UUID, input *biz.MasterDataItem) (*biz.MasterDataItem, error) {
	existing, err := r.data.db.MasterDataItem.Query().Where(masterdataent.IDEQ(id), masterdataent.OrganizationIDEQ(organizationID), masterdataent.KindEQ(masterdataent.Kind(input.Kind))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrMasterDataNotFound
		}
		return nil, err
	}
	update := existing.Update().SetName(input.Name).SetNillableNameEn(input.NameEN).SetNillableParentCode(input.ParentCode).SetNillableTeuFactor(input.TEUFactor).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled).SetAttributes(masterDataAttributesToEnt(input.Attributes))
	if input.NameEN == nil {
		update.ClearNameEn()
	}
	if input.ParentCode == nil {
		update.ClearParentCode()
	}
	if input.TEUFactor == nil {
		update.ClearTeuFactor()
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}
	return masterDataItemToBiz(updated), nil
}

func (r *masterDataRepo) Import(ctx context.Context, organizationID uuid.UUID, mode biz.MasterDataImportMode, inputs []*biz.MasterDataItem) (*biz.MasterDataImportResult, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.MasterDataImportResult{Items: make([]*biz.MasterDataItem, 0, len(inputs))}
	for _, input := range inputs {
		existing, err := tx.MasterDataItem.Query().Where(masterdataent.OrganizationIDEQ(organizationID), masterdataent.KindEQ(masterdataent.Kind(input.Kind)), masterdataent.CodeEQ(input.Code)).Only(ctx)
		if ent.IsNotFound(err) {
			created, createErr := tx.MasterDataItem.Create().
				SetOrganizationID(organizationID).
				SetKind(masterdataent.Kind(input.Kind)).
				SetCode(input.Code).
				SetName(input.Name).
				SetNillableNameEn(input.NameEN).
				SetNillableParentCode(input.ParentCode).
				SetNillableTeuFactor(input.TEUFactor).
				SetSource(input.Source).
				SetSortOrder(input.SortOrder).
				SetEnabled(input.Enabled).
				SetAttributes(masterDataAttributesToEnt(input.Attributes)).
				Save(ctx)
			if createErr != nil {
				_ = tx.Rollback()
				if ent.IsConstraintError(createErr) {
					return nil, biz.ErrMasterDataCodeExists
				}
				return nil, createErr
			}
			result.Items = append(result.Items, masterDataItemToBiz(created))
			result.Created++
			continue
		}
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if mode == biz.MasterDataImportModeCreateOnly {
			_ = tx.Rollback()
			return nil, biz.ErrMasterDataCodeExists
		}
		update := existing.Update().
			SetName(input.Name).
			SetNillableNameEn(input.NameEN).
			SetNillableParentCode(input.ParentCode).
			SetNillableTeuFactor(input.TEUFactor).
			SetSource(input.Source).
			SetSortOrder(input.SortOrder).
			SetEnabled(input.Enabled)
		update.SetAttributes(masterDataAttributesToEnt(input.Attributes))
		if input.NameEN == nil {
			update.ClearNameEn()
		}
		if input.ParentCode == nil {
			update.ClearParentCode()
		}
		if input.TEUFactor == nil {
			update.ClearTeuFactor()
		}
		updated, updateErr := update.Save(ctx)
		if updateErr != nil {
			_ = tx.Rollback()
			return nil, updateErr
		}
		result.Items = append(result.Items, masterDataItemToBiz(updated))
		result.Updated++
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func masterDataItemsToBiz(items []*ent.MasterDataItem) []*biz.MasterDataItem {
	result := make([]*biz.MasterDataItem, 0, len(items))
	for _, item := range items {
		result = append(result, masterDataItemToBiz(item))
	}
	return result
}

func masterDataItemToBiz(item *ent.MasterDataItem) *biz.MasterDataItem {
	return &biz.MasterDataItem{ID: item.ID, OrganizationID: item.OrganizationID, Kind: biz.MasterDataKind(item.Kind), Code: item.Code, Name: item.Name, NameEN: item.NameEn, ParentCode: item.ParentCode, TEUFactor: item.TeuFactor, Attributes: masterDataAttributesToBiz(item.Attributes), Source: item.Source, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func masterDataAttributesToEnt(attributes biz.MasterDataAttributes) *entschema.MasterDataAttributes {
	return &entschema.MasterDataAttributes{
		Continent: attributes.Continent, CurrencyCode: attributes.CurrencyCode,
		RegionLevel: attributes.RegionLevel,
	}
}

func masterDataAttributesToBiz(attributes *entschema.MasterDataAttributes) biz.MasterDataAttributes {
	if attributes == nil {
		return biz.MasterDataAttributes{}
	}
	return biz.MasterDataAttributes{
		Continent: attributes.Continent, CurrencyCode: attributes.CurrencyCode,
		RegionLevel: attributes.RegionLevel,
	}
}

var _ biz.MasterDataRepo = (*masterDataRepo)(nil)
