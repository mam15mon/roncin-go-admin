package data

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/administrativeregion"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

type referenceDataRepo struct {
	data *Data
}

func NewReferenceDataRepo(data *Data) biz.ReferenceDataRepo {
	return &referenceDataRepo{data: data}
}

type orgCurrencyContext struct {
	orgID             uuid.UUID
	isHeadquarters    bool
	baseCurrency      string
	enabledCurrencies []string
}

func (r *referenceDataRepo) resolveOrgCurrencyContext(ctx context.Context, organizationID uuid.UUID) (*orgCurrencyContext, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if organizationID == uuid.Nil {
		hq, err := client.Organization.Query().Where(organizationent.KindEQ(organizationent.KindHeadquarters), organizationent.EnabledEQ(true)).First(ctx)
		if err != nil {
			return &orgCurrencyContext{
				orgID:          uuid.Nil,
				isHeadquarters: true,
				baseCurrency:   "CNY",
			}, nil
		}
		base := "CNY"
		if hq.BaseCurrency != nil && *hq.BaseCurrency != "" {
			base = *hq.BaseCurrency
		}
		return &orgCurrencyContext{
			orgID:          hq.ID,
			isHeadquarters: true,
			baseCurrency:   base,
		}, nil
	}

	currentID := organizationID
	baseCurrency := ""
	var targetOrg *ent.Organization
	for {
		item, err := client.Organization.Query().Where(organizationent.IDEQ(currentID), organizationent.EnabledEQ(true)).Only(ctx)
		if err != nil {
			return nil, err
		}
		if targetOrg == nil {
			targetOrg = item
		}
		if baseCurrency == "" && item.BaseCurrency != nil && *item.BaseCurrency != "" {
			baseCurrency = *item.BaseCurrency
		}
		if item.ParentID == nil {
			if baseCurrency == "" {
				baseCurrency = "CNY"
			}
			isHQ := (targetOrg.Kind == organizationent.KindHeadquarters)
			var enabled []string
			if !isHQ {
				if len(targetOrg.EnabledCurrencies) > 0 {
					enabled = targetOrg.EnabledCurrencies
				} else {
					enabled = defaultCompanyEnabledCurrencies(baseCurrency)
				}
			}
			return &orgCurrencyContext{
				orgID:             targetOrg.ID,
				isHeadquarters:    isHQ,
				baseCurrency:      baseCurrency,
				enabledCurrencies: enabled,
			}, nil
		}
		currentID = *item.ParentID
	}
}

func (r *referenceDataRepo) ListCurrencies(ctx context.Context, organizationID uuid.UUID, enabledOnly bool) ([]*biz.Currency, error) {
	orgCtx, err := r.resolveOrgCurrencyContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	// 全局所有启用的 ISO 4217 币种主库
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.Currency.Query().
		Where(currency.EnabledEQ(true)).
		Order(ent.Asc(currency.FieldCode)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	enabledMap := make(map[string]bool)
	if orgCtx.isHeadquarters {
		// 总部视角：全库所有有效币种均视为启用
		for _, item := range items {
			enabledMap[item.Code] = true
		}
	} else {
		// 分公司视角：读取组织启用的币种，本位币常开
		for _, code := range orgCtx.enabledCurrencies {
			enabledMap[strings.ToUpper(code)] = true
		}
		enabledMap[orgCtx.baseCurrency] = true
	}

	result := make([]*biz.Currency, 0, len(items))
	for _, item := range items {
		isBase := (item.Code == orgCtx.baseCurrency)
		isEnabled := isBase || enabledMap[item.Code]

		if enabledOnly && !isEnabled {
			continue
		}

		result = append(result, &biz.Currency{
			ID:             item.ID,
			Code:           item.Code,
			Name:           item.Name,
			Symbol:         item.Symbol,
			MinorUnit:      item.MinorUnit,
			Enabled:        isEnabled,
			IsBaseCurrency: isBase,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}

	// 本位币排第一位，其余按币种代码升序
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].IsBaseCurrency != result[j].IsBaseCurrency {
			return result[i].IsBaseCurrency
		}
		return result[i].Code < result[j].Code
	})

	return result, nil
}

func (r *referenceDataRepo) SetCurrencyEnabled(ctx context.Context, organizationID uuid.UUID, code string, enabled bool) (*biz.Currency, error) {
	var result *biz.Currency
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		curr, err := tx.Currency.Query().Where(currency.CodeEQ(code)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return biz.ErrCurrencyNotFound
			}
			return err
		}

		orgCtx, err := r.resolveOrgCurrencyContext(ctx, organizationID)
		if err != nil {
			return err
		}

		if orgCtx.isHeadquarters {
			// 总部操作：切换单行启用状态，UPDATE 本身原子生效，无需预锁
			updated, saveErr := tx.Currency.UpdateOneID(curr.ID).SetEnabled(enabled).Save(ctx)
			if saveErr != nil {
				return saveErr
			}
			result = &biz.Currency{
				ID:             updated.ID,
				Code:           updated.Code,
				Name:           updated.Name,
				Symbol:         updated.Symbol,
				MinorUnit:      updated.MinorUnit,
				Enabled:        updated.Enabled,
				IsBaseCurrency: (updated.Code == orgCtx.baseCurrency),
				CreatedAt:      updated.CreatedAt,
				UpdatedAt:      updated.UpdatedAt,
			}
			return nil
		}

		// 分公司操作：
		// 校验：本位币绝对不可禁用
		if !enabled && code == orgCtx.baseCurrency {
			return biz.ErrCurrencyBaseCannotBeDisabled
		}

		// 锁定分公司组织行，消除并发修改 enabled_currencies 时的读-改-写竞态
		lockedOrg, orgErr := tx.Organization.Query().Where(organizationent.IDEQ(orgCtx.orgID)).ForUpdate().Only(ctx)
		if orgErr != nil {
			return orgErr
		}

		currentCurrencies := lockedOrg.EnabledCurrencies
		if len(currentCurrencies) == 0 {
			currentCurrencies = defaultCompanyEnabledCurrencies(orgCtx.baseCurrency)
		}

		currentMap := make(map[string]bool)
		for _, c := range currentCurrencies {
			currentMap[strings.ToUpper(c)] = true
		}
		currentMap[orgCtx.baseCurrency] = true

		if enabled {
			currentMap[code] = true
		} else {
			delete(currentMap, code)
		}
		currentMap[orgCtx.baseCurrency] = true

		newCurrencies := make([]string, 0, len(currentMap))
		for k := range currentMap {
			newCurrencies = append(newCurrencies, k)
		}
		sort.Strings(newCurrencies)

		if err := tx.Organization.UpdateOneID(lockedOrg.ID).SetEnabledCurrencies(newCurrencies).Exec(ctx); err != nil {
			return err
		}

		result = &biz.Currency{
			ID:             curr.ID,
			Code:           curr.Code,
			Name:           curr.Name,
			Symbol:         curr.Symbol,
			MinorUnit:      curr.MinorUnit,
			Enabled:        enabled,
			IsBaseCurrency: (curr.Code == orgCtx.baseCurrency),
			CreatedAt:      curr.CreatedAt,
			UpdatedAt:      time.Now(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *referenceDataRepo) SearchCurrencies(ctx context.Context, options biz.SelectorListOptions) (*biz.PagedList[*biz.Currency], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.Currency.Query().Where(currency.EnabledEQ(true))
	if options.Keyword != "" {
		query.Where(currency.Or(currency.CodeContainsFold(options.Keyword), currency.NameContainsFold(options.Keyword), currency.SearchKeywordsContainsFold(options.Keyword)))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.Currency, error) {
		return query.Order(ent.Asc(currency.FieldCode)).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(func(item *ent.Currency) *biz.Currency {
		return &biz.Currency{ID: item.ID, Code: item.Code, Name: item.Name, Symbol: item.Symbol, MinorUnit: item.MinorUnit, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	}))
}

func (r *referenceDataRepo) ListAdministrativeRegions(ctx context.Context, query biz.AdministrativeRegionQuery) (*biz.PagedList[*biz.AdministrativeRegion], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	builder := client.AdministrativeRegion.Query().Where(administrativeregion.EnabledEQ(true))
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
			administrativeregion.SearchKeywordsContainsFold(query.Keyword),
		))
	}
	// Page/PageSize 同时为零表示维护页整表加载：跳过 COUNT 与深分页 OFFSET，一次按代码序返回全部。
	if query.Page == 0 && query.PageSize == 0 {
		entities, err := builder.Order(ent.Asc(administrativeregion.FieldCode)).All(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]*biz.AdministrativeRegion, 0, len(entities))
		for _, entity := range entities {
			items = append(items, convertAdministrativeRegion(entity))
		}
		return &biz.PagedList[*biz.AdministrativeRegion]{Items: items, Total: len(items), Page: 1, PageSize: len(items)}, nil
	}
	return paginate(ctx, builder.Count, func(ctx context.Context, offset, limit int) ([]*ent.AdministrativeRegion, error) {
		return builder.Order(ent.Asc(administrativeregion.FieldCode)).Offset(offset).Limit(limit).All(ctx)
	}, query.Page, query.PageSize, infalliblePageConverter(convertAdministrativeRegion))
}

func convertAdministrativeRegion(item *ent.AdministrativeRegion) *biz.AdministrativeRegion {
	return &biz.AdministrativeRegion{
		ID: item.ID, Code: item.Code, Name: item.Name, Level: item.Level,
		ParentCode: item.ParentCode, RegionType: item.RegionType, Source: item.Source,
		SourceVersion: item.SourceVersion, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

var _ biz.ReferenceDataRepo = (*referenceDataRepo)(nil)
