package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	taxableserviceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/taxableservice"
)

type feeCatalogRepo struct{ data *Data }

func NewFeeCatalogRepo(data *Data) biz.FeeCatalogRepo { return &feeCatalogRepo{data: data} }

func (r *feeCatalogRepo) ListFeeSettings(ctx context.Context, organizationID uuid.UUID, options biz.FeeCatalogListOptions) (*biz.PagedList[*biz.FeeSetting], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	// 公司实际使用的科目不读取系统模板。
	query := client.FeeSetting.Query().Where(feesettingent.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(feesettingent.Or(feesettingent.FeeCodeContainsFold(options.Keyword), feesettingent.NameZhContainsFold(options.Keyword), feesettingent.NameEnContainsFold(options.Keyword), feesettingent.AliasNameContainsFold(options.Keyword), feesettingent.SearchKeywordsContainsFold(options.Keyword)))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.FeeSetting, error) {
		return query.WithChargeCategory().WithBillingUnit().WithAbnormalCase().WithTaxableService().
			Order(feesettingent.BySortOrder(), feesettingent.ByFeeCode(), feesettingent.ByID()).
			Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, feeSettingToBiz)
}

// CreateFeeSetting 创建本公司费用科目。
func (r *feeCatalogRepo) CreateFeeSetting(ctx context.Context, organizationID uuid.UUID, input *biz.FeeSetting, audit *biz.AuditEvent) (*biz.FeeSetting, error) {
	if err := r.validateFeeSettingReferences(ctx, organizationID, input); err != nil {
		return nil, err
	}
	var converted *biz.FeeSetting
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		builder := tx.FeeSetting.Create().
			SetID(input.ID).SetOrganizationID(organizationID).
			SetFeeCode(input.FeeCode).SetNameZh(input.NameZH).
			SetNillableNameEn(input.NameEN).SetNillableAliasName(input.AliasName).SetChargeCategoryID(input.ChargeCategoryID).
			SetDefaultCurrency(input.DefaultCurrency).SetBillingUnitID(input.BillingUnitID).SetNillableAbnormalCaseID(input.AbnormalCaseID).
			SetTaxRate(input.TaxRate.StringFixed(2)).SetTaxableServiceID(input.TaxableServiceID).SetEnabled(true).SetSortOrder(input.SortOrder)
		if _, createErr := builder.Save(ctx); createErr != nil {
			return mapEntError(createErr, nil, biz.ErrFeeSettingCodeExists)
		}
		if auditErr := writeAudit(ctx, tx.AuditLog, audit); auditErr != nil {
			return auditErr
		}
		saved, queryErr := tx.FeeSetting.Query().Where(feesettingent.IDEQ(input.ID)).
			WithChargeCategory().WithBillingUnit().WithAbnormalCase().WithTaxableService().Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		var convertErr error
		converted, convertErr = feeSettingToBiz(saved)
		return convertErr
	})
	if err != nil {
		return nil, err
	}
	return converted, nil
}

// UpdateFeeSetting 仅更新本公司科目，归属不可迁移。
func (r *feeCatalogRepo) UpdateFeeSetting(ctx context.Context, organizationID uuid.UUID, input *biz.FeeSetting, audit *biz.AuditEvent) (*biz.FeeSetting, error) {
	var converted *biz.FeeSetting
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		scope := feesettingent.OrganizationIDEQ(organizationID)
		current, queryErr := tx.FeeSetting.Query().Where(feesettingent.IDEQ(input.ID), scope).ForUpdate().Only(ctx)
		if queryErr != nil {
			// 归属校验先行：作用域外的行一律按不存在处理，不提前泄漏引用细节。
			return mapEntError(queryErr, biz.ErrFeeSettingNotFound, nil)
		}
		if err := r.validateFeeSettingReferences(ctx, organizationID, input); err != nil {
			return err
		}
		builder := current.Update().
			SetFeeCode(input.FeeCode).SetNameZh(input.NameZH).SetDefaultCurrency(input.DefaultCurrency).
			SetBillingUnitID(input.BillingUnitID).SetTaxRate(input.TaxRate.StringFixed(2)).SetTaxableServiceID(input.TaxableServiceID).
			SetEnabled(input.Enabled).SetSortOrder(input.SortOrder).SetChargeCategoryID(input.ChargeCategoryID)
		if input.NameEN == nil {
			builder.ClearNameEn()
		} else {
			builder.SetNameEn(*input.NameEN)
		}
		if input.AliasName == nil {
			builder.ClearAliasName()
		} else {
			builder.SetAliasName(*input.AliasName)
		}
		if input.AbnormalCaseID == nil {
			builder.ClearAbnormalCaseID()
		} else {
			builder.SetAbnormalCaseID(*input.AbnormalCaseID)
		}
		if _, updateErr := builder.Save(ctx); updateErr != nil {
			return mapEntError(updateErr, nil, biz.ErrFeeSettingCodeExists)
		}
		if auditErr := writeAudit(ctx, tx.AuditLog, audit); auditErr != nil {
			return auditErr
		}
		saved, queryErr := tx.FeeSetting.Query().Where(feesettingent.IDEQ(input.ID)).
			WithChargeCategory().WithBillingUnit().WithAbnormalCase().WithTaxableService().Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		var convertErr error
		converted, convertErr = feeSettingToBiz(saved)
		return convertErr
	})
	if err != nil {
		return nil, err
	}
	return converted, nil
}

// validateFeeSettingReferences 校验费用设置引用（design §5 放宽）：计费单位与
// 费用大类查 A 型全局表（无组织维度）；异常类型同属 A 型主数据；税务名称为
// C 型组织私有，查「本组织有效行」；币种查全局币种表；全部要求 enabled。
func (r *feeCatalogRepo) validateFeeSettingReferences(ctx context.Context, organizationID uuid.UUID, input *biz.FeeSetting) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	if err := validateFeePublicReferences(ctx, client, input); err != nil {
		return err
	}
	taxableExists, err := client.TaxableService.Query().Where(taxableserviceent.IDEQ(input.TaxableServiceID), taxableserviceent.OrganizationIDEQ(organizationID), taxableserviceent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !taxableExists {
		return biz.ErrFeeCatalogReferenceInvalid
	}
	return nil
}
func validateFeePublicReferences(ctx context.Context, client *ent.Client, input *biz.FeeSetting) error {
	chargeCategoryExists, err := client.MasterDataItem.Query().Where(masterdataitement.IDEQ(input.ChargeCategoryID), masterdataitement.KindEQ(masterdataitement.KindChargeCategory), masterdataitement.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !chargeCategoryExists {
		return biz.ErrFeeCatalogReferenceInvalid
	}
	if input.AbnormalCaseID != nil {
		exists, err := client.MasterDataItem.Query().Where(masterdataitement.IDEQ(*input.AbnormalCaseID), masterdataitement.KindEQ(masterdataitement.KindAbnormalCase), masterdataitement.EnabledEQ(true)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrFeeCatalogReferenceInvalid
		}
	}
	billingExists, err := client.BillingUnit.Query().Where(billingunitent.IDEQ(input.BillingUnitID), billingunitent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	currencyExists, err := client.Currency.Query().Where(currencyent.CodeEQ(input.DefaultCurrency), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !billingExists || !currencyExists {
		return biz.ErrFeeCatalogReferenceInvalid
	}
	return nil
}

func (r *feeCatalogRepo) ListBillingUnits(ctx context.Context, _ uuid.UUID, options biz.FeeCatalogListOptions) (*biz.PagedList[*biz.BillingUnit], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	// A 型全局主数据：全员同权可见，无组织过滤。
	query := client.BillingUnit.Query()
	if options.Keyword != "" {
		query.Where(billingunitent.Or(billingunitent.CodeContainsFold(options.Keyword), billingunitent.NameContainsFold(options.Keyword), billingunitent.SearchKeywordsContainsFold(options.Keyword)))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.BillingUnit, error) {
		return query.Order(billingunitent.BySortOrder(), billingunitent.ByCode(), billingunitent.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(billingUnitToBiz))
}

func (r *feeCatalogRepo) CreateBillingUnit(ctx context.Context, input *biz.BillingUnit, audit *biz.AuditEvent) (*biz.BillingUnit, error) {
	var saved *ent.BillingUnit
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var saveErr error
		saved, saveErr = tx.BillingUnit.Create().SetID(input.ID).SetCode(input.Code).SetName(input.Name).SetIsContainerUnit(input.IsContainerUnit).SetQuantityMustBeInteger(input.QuantityMustBeInteger).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, nil, biz.ErrBillingUnitCodeExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return billingUnitToBiz(saved), nil
}

func (r *feeCatalogRepo) UpdateBillingUnit(ctx context.Context, input *biz.BillingUnit, audit *biz.AuditEvent) (*biz.BillingUnit, error) {
	var saved *ent.BillingUnit
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.BillingUnit.Query().Where(billingunitent.IDEQ(input.ID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrBillingUnitNotFound, nil)
		}
		var saveErr error
		saved, saveErr = current.Update().SetCode(input.Code).SetName(input.Name).SetIsContainerUnit(input.IsContainerUnit).SetQuantityMustBeInteger(input.QuantityMustBeInteger).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled).Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, nil, biz.ErrBillingUnitCodeExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return billingUnitToBiz(saved), nil
}

func (r *feeCatalogRepo) ListTaxableServices(ctx context.Context, organizationID uuid.UUID, options biz.FeeCatalogListOptions) (*biz.PagedList[*biz.TaxableService], error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	// C 型组织私有：随组织税务主体自维护，仅读本组织行。
	query := client.TaxableService.Query().Where(taxableserviceent.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(taxableserviceent.Or(taxableserviceent.NameContainsFold(options.Keyword), taxableserviceent.ShortNameContainsFold(options.Keyword), taxableserviceent.GoodsCodeContainsFold(options.Keyword), taxableserviceent.SearchKeywordsContainsFold(options.Keyword)))
	}
	return paginate(ctx, query.Count, func(ctx context.Context, offset, limit int) ([]*ent.TaxableService, error) {
		return query.Order(taxableserviceent.ByName(), taxableserviceent.ByID()).Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, taxableServiceToBiz)
}

func (r *feeCatalogRepo) CreateTaxableService(ctx context.Context, input *biz.TaxableService, audit *biz.AuditEvent) (*biz.TaxableService, error) {
	var saved *ent.TaxableService
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var saveErr error
		saved, saveErr = tx.TaxableService.Create().SetID(input.ID).SetOrganizationID(input.OrganizationID).SetName(input.Name).SetNillableShortName(input.ShortName).SetNillableGoodsCode(input.GoodsCode).SetDefaultTaxRate(input.DefaultTaxRate.StringFixed(2)).SetEnabled(true).Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, nil, biz.ErrTaxableServiceNameExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return taxableServiceToBiz(saved)
}

func (r *feeCatalogRepo) UpdateTaxableService(ctx context.Context, input *biz.TaxableService, audit *biz.AuditEvent) (*biz.TaxableService, error) {
	var saved *ent.TaxableService
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.TaxableService.Query().Where(taxableserviceent.IDEQ(input.ID), taxableserviceent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrTaxableServiceNotFound, nil)
		}
		builder := current.Update().SetName(input.Name).SetDefaultTaxRate(input.DefaultTaxRate.StringFixed(2)).SetEnabled(input.Enabled)
		if input.ShortName == nil {
			builder.ClearShortName()
		} else {
			builder.SetShortName(*input.ShortName)
		}
		if input.GoodsCode == nil {
			builder.ClearGoodsCode()
		} else {
			builder.SetGoodsCode(*input.GoodsCode)
		}
		var saveErr error
		saved, saveErr = builder.Save(ctx)
		if saveErr != nil {
			return mapEntError(saveErr, nil, biz.ErrTaxableServiceNameExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return taxableServiceToBiz(saved)
}

func feeSettingToBiz(item *ent.FeeSetting) (*biz.FeeSetting, error) {
	taxRate, err := decimalOf(item.TaxRate)
	if err != nil {
		return nil, err
	}
	billingUnit, err := item.Edges.BillingUnitOrErr()
	if err != nil {
		return nil, err
	}
	taxableService, err := item.Edges.TaxableServiceOrErr()
	if err != nil {
		return nil, err
	}
	result := &biz.FeeSetting{ID: item.ID, OrganizationID: item.OrganizationID, FeeCode: item.FeeCode, NameZH: item.NameZh, NameEN: item.NameEn, AliasName: item.AliasName, ChargeCategoryID: item.ChargeCategoryID, DefaultCurrency: item.DefaultCurrency, BillingUnitID: item.BillingUnitID, BillingUnitName: billingUnit.Name, AbnormalCaseID: item.AbnormalCaseID, TaxRate: taxRate, TaxableServiceID: item.TaxableServiceID, TaxableServiceName: taxableService.Name, Enabled: item.Enabled, SortOrder: item.SortOrder, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.Edges.ChargeCategory != nil {
		result.ChargeCategoryName = item.Edges.ChargeCategory.Name
	}
	if item.Edges.AbnormalCase != nil {
		name := item.Edges.AbnormalCase.Name
		result.AbnormalCaseName = &name
	}
	return result, nil
}

func billingUnitToBiz(item *ent.BillingUnit) *biz.BillingUnit {
	return &biz.BillingUnit{ID: item.ID, Code: item.Code, Name: item.Name, IsContainerUnit: item.IsContainerUnit, QuantityMustBeInteger: item.QuantityMustBeInteger, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func taxableServiceToBiz(item *ent.TaxableService) (*biz.TaxableService, error) {
	taxRate, err := decimalOf(item.DefaultTaxRate)
	if err != nil {
		return nil, err
	}
	return &biz.TaxableService{ID: item.ID, OrganizationID: item.OrganizationID, Name: item.Name, ShortName: item.ShortName, GoodsCode: item.GoodsCode, DefaultTaxRate: taxRate, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

var _ biz.FeeCatalogRepo = (*feeCatalogRepo)(nil)
