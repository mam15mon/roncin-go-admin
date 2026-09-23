package data

import (
	"context"
	"fmt"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	entschema "github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"

	"github.com/google/uuid"
)

type masterDataRepo struct{ data *Data }

func NewMasterDataRepo(data *Data) biz.MasterDataRepo { return &masterDataRepo{data: data} }

// CreateDefaultOrderOptions 为空库写入订单默认选项种子。A 型主数据全局唯一，
// 不再挂载组织；迁移种子可能已落部分行，按 kind+code 幂等跳过已存在项。
func CreateDefaultOrderOptions(ctx context.Context, tx *ent.Tx) error {
	for _, item := range biz.DefaultOrderOptions() {
		exists, err := tx.MasterDataItem.Query().
			Where(masterdataent.KindEQ(masterdataent.Kind(item.Kind)), masterdataent.CodeEQ(item.Code)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("检查订单默认选项 %s/%s: %w", item.Kind, item.Code, err)
		}
		if exists {
			continue
		}
		if _, err := tx.MasterDataItem.Create().
			SetKind(masterdataent.Kind(item.Kind)).
			SetCode(item.Code).
			SetName(item.Name).
			SetNillableTeuFactor(item.TEUFactor).
			SetSource(item.Source).
			SetSortOrder(item.SortOrder).
			SetEnabled(true).
			Save(ctx); err != nil {
			return fmt.Errorf("创建订单默认选项 %s/%s: %w", item.Kind, item.Code, err)
		}
	}
	return nil
}

// CreateDefaultCountries 为空库写入默认国家种子。A 型主数据全局唯一，不再挂载组织；
// 迁移种子可能已落部分行，按 kind+code 幂等跳过已存在项。
func CreateDefaultCountries(ctx context.Context, tx *ent.Tx) error {
	for _, item := range biz.DefaultCountryOptions() {
		exists, err := tx.MasterDataItem.Query().
			Where(masterdataent.KindEQ(masterdataent.Kind(item.Kind)), masterdataent.CodeEQ(item.Code)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("检查默认国家 %s: %w", item.Code, err)
		}
		if exists {
			continue
		}
		if _, err := tx.MasterDataItem.Create().
			SetKind(masterdataent.Kind(item.Kind)).
			SetCode(item.Code).
			SetName(item.Name).
			SetNillableNameEn(item.NameEN).
			SetSource(item.Source).
			SetSortOrder(item.SortOrder).
			SetEnabled(true).
			SetAttributes(masterDataAttributesToEnt(item.Attributes)).
			Save(ctx); err != nil {
			return fmt.Errorf("创建默认国家 %s: %w", item.Code, err)
		}
	}
	return nil
}

func (r *masterDataRepo) List(ctx context.Context, _ uuid.UUID, options biz.MasterDataListOptions) (*biz.MasterDataList, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.MasterDataItem.Query()
	if options.Kind != "" {
		query.Where(masterdataent.KindEQ(masterdataent.Kind(options.Kind)))
	}
	if options.Keyword != "" {
		query.Where(masterdataent.Or(masterdataent.CodeContainsFold(options.Keyword), masterdataent.NameContainsFold(options.Keyword), masterdataent.NameEnContainsFold(options.Keyword), masterdataent.SearchKeywordsContainsFold(options.Keyword)))
	}
	if options.Enabled != nil {
		query.Where(masterdataent.EnabledEQ(*options.Enabled))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.MasterDataItem, error) {
		return query.Order(masterdataent.ByKind(), masterdataent.BySortOrder(), masterdataent.ByCode()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(masterDataItemToBiz))
}

func (r *masterDataRepo) ListEnabled(ctx context.Context, _ uuid.UUID) ([]*biz.MasterDataItem, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.MasterDataItem.Query().Where(masterdataent.EnabledEQ(true)).Order(masterdataent.ByKind(), masterdataent.BySortOrder(), masterdataent.ByCode()).All(ctx)
	if err != nil {
		return nil, err
	}
	return masterDataItemsToBiz(items), nil
}

func (r *masterDataRepo) Create(ctx context.Context, _ uuid.UUID, input *biz.MasterDataItem, audit *biz.AuditEvent) (*biz.MasterDataItem, error) {
	var created *ent.MasterDataItem
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		create := tx.MasterDataItem.Create().SetKind(masterdataent.Kind(input.Kind)).SetCode(input.Code).SetName(input.Name).SetNillableNameEn(input.NameEN).SetNillableParentCode(input.ParentCode).SetNillableTeuFactor(input.TEUFactor).SetSource(input.Source).SetSortOrder(input.SortOrder).SetEnabled(true).SetAttributes(masterDataAttributesToEnt(input.Attributes))
		var createErr error
		created, createErr = create.Save(ctx)
		if createErr != nil {
			return mapEntError(createErr, nil, biz.ErrMasterDataCodeExists)
		}
		audit.Details["master_data.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return masterDataItemToBiz(created), nil
}

func (r *masterDataRepo) Update(ctx context.Context, _ uuid.UUID, id uuid.UUID, input *biz.MasterDataItem, audit *biz.AuditEvent) (*biz.MasterDataItem, error) {
	var updated *ent.MasterDataItem
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.MasterDataItem.Query().Where(masterdataent.IDEQ(id), masterdataent.KindEQ(masterdataent.Kind(input.Kind))).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrMasterDataNotFound, nil)
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
		var updateErr error
		updated, updateErr = update.Save(ctx)
		if updateErr != nil {
			return updateErr
		}
		audit.Details["master_data.code"] = updated.Code
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return masterDataItemToBiz(updated), nil
}

func (r *masterDataRepo) Import(ctx context.Context, _ uuid.UUID, mode biz.MasterDataImportMode, inputs []*biz.MasterDataItem, audit *biz.AuditEvent) (*biz.MasterDataImportResult, error) {
	result := &biz.MasterDataImportResult{Items: make([]*biz.MasterDataItem, 0, len(inputs))}
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		for _, input := range inputs {
			existing, queryErr := tx.MasterDataItem.Query().Where(masterdataent.KindEQ(masterdataent.Kind(input.Kind)), masterdataent.CodeEQ(input.Code)).Only(ctx)
			if ent.IsNotFound(queryErr) {
				created, createErr := tx.MasterDataItem.Create().
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
					return mapEntError(createErr, nil, biz.ErrMasterDataCodeExists)
				}
				result.Items = append(result.Items, masterDataItemToBiz(created))
				result.Created++
				continue
			}
			if queryErr != nil {
				return queryErr
			}
			if mode == biz.MasterDataImportModeCreateOnly {
				return biz.ErrMasterDataCodeExists
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
				return updateErr
			}
			result.Items = append(result.Items, masterDataItemToBiz(updated))
			result.Updated++
		}
		audit.Details["created"] = fmt.Sprintf("%d", result.Created)
		audit.Details["updated"] = fmt.Sprintf("%d", result.Updated)
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
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
	return &biz.MasterDataItem{ID: item.ID, Kind: biz.MasterDataKind(item.Kind), Code: item.Code, Name: item.Name, NameEN: item.NameEn, ParentCode: item.ParentCode, TEUFactor: item.TeuFactor, Attributes: masterDataAttributesToBiz(item.Attributes), Source: item.Source, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
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
