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
	"github.com/shopspring/decimal"
)

type feeCatalogRepo struct{ data *Data }

func NewFeeCatalogRepo(data *Data) biz.FeeCatalogRepo { return &feeCatalogRepo{data: data} }

func (r *feeCatalogRepo) headquartersOrganizationID(ctx context.Context, organizationID uuid.UUID) (uuid.UUID, error) {
	return resolveHeadquartersOrganizationID(ctx, r.data.db.Organization, organizationID)
}

func (r *feeCatalogRepo) requireHeadquarters(ctx context.Context, organizationID uuid.UUID) error {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}
	if headquartersID != organizationID {
		return biz.ErrFeeCatalogHeadquartersRequired
	}
	return nil
}

func (r *feeCatalogRepo) ListFeeSettings(ctx context.Context, organizationID uuid.UUID) ([]*biz.FeeSetting, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	items, err := r.data.db.FeeSetting.Query().
		Where(feesettingent.OrganizationIDEQ(headquartersID)).
		WithServiceType().WithBillingUnit().WithAbnormalCase().WithTaxableService().
		Order(feesettingent.BySortOrder(), feesettingent.ByFeeCode(), feesettingent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.FeeSetting, 0, len(items))
	for _, item := range items {
		converted, err := feeSettingToBiz(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *feeCatalogRepo) CreateFeeSetting(ctx context.Context, input *biz.FeeSetting, audit *biz.AuditEvent) (*biz.FeeSetting, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	if err := r.validateFeeSettingReferences(ctx, input); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	builder := tx.FeeSetting.Create().
		SetID(input.ID).SetOrganizationID(input.OrganizationID).SetFeeCode(input.FeeCode).SetNameZh(input.NameZH).
		SetNillableNameEn(input.NameEN).SetNillableAliasName(input.AliasName).SetNillableServiceTypeID(input.ServiceTypeID).
		SetDefaultCurrency(input.DefaultCurrency).SetBillingUnitID(input.BillingUnitID).SetNillableAbnormalCaseID(input.AbnormalCaseID).
		SetTaxRate(input.TaxRate.StringFixed(2)).SetTaxableServiceID(input.TaxableServiceID).SetEnabled(true).SetSortOrder(input.SortOrder)
	if _, err = builder.Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrFeeSettingCodeExists
		}
		return nil, err
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	saved, err := tx.FeeSetting.Query().Where(feesettingent.IDEQ(input.ID)).
		WithServiceType().WithBillingUnit().WithAbnormalCase().WithTaxableService().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	converted, err := feeSettingToBiz(saved)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return converted, nil
}

func (r *feeCatalogRepo) UpdateFeeSetting(ctx context.Context, input *biz.FeeSetting, audit *biz.AuditEvent) (*biz.FeeSetting, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	if err := r.validateFeeSettingReferences(ctx, input); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	current, err := tx.FeeSetting.Query().Where(feesettingent.IDEQ(input.ID), feesettingent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrFeeSettingNotFound
		}
		return nil, err
	}
	builder := current.Update().
		SetFeeCode(input.FeeCode).SetNameZh(input.NameZH).SetDefaultCurrency(input.DefaultCurrency).
		SetBillingUnitID(input.BillingUnitID).SetTaxRate(input.TaxRate.StringFixed(2)).SetTaxableServiceID(input.TaxableServiceID).
		SetEnabled(input.Enabled).SetSortOrder(input.SortOrder)
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
	if input.ServiceTypeID == nil {
		builder.ClearServiceTypeID()
	} else {
		builder.SetServiceTypeID(*input.ServiceTypeID)
	}
	if input.AbnormalCaseID == nil {
		builder.ClearAbnormalCaseID()
	} else {
		builder.SetAbnormalCaseID(*input.AbnormalCaseID)
	}
	if _, err = builder.Save(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrFeeSettingCodeExists
		}
		return nil, err
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	saved, err := tx.FeeSetting.Query().Where(feesettingent.IDEQ(input.ID)).
		WithServiceType().WithBillingUnit().WithAbnormalCase().WithTaxableService().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	converted, err := feeSettingToBiz(saved)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return converted, nil
}

func (r *feeCatalogRepo) validateFeeSettingReferences(ctx context.Context, input *biz.FeeSetting) error {
	if input.ServiceTypeID != nil {
		exists, err := r.data.db.MasterDataItem.Query().Where(masterdataitement.IDEQ(*input.ServiceTypeID), masterdataitement.OrganizationIDEQ(input.OrganizationID), masterdataitement.KindEQ(masterdataitement.KindServiceType), masterdataitement.EnabledEQ(true)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrFeeCatalogReferenceInvalid
		}
	}
	if input.AbnormalCaseID != nil {
		exists, err := r.data.db.MasterDataItem.Query().Where(masterdataitement.IDEQ(*input.AbnormalCaseID), masterdataitement.OrganizationIDEQ(input.OrganizationID), masterdataitement.KindEQ(masterdataitement.KindAbnormalCase), masterdataitement.EnabledEQ(true)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrFeeCatalogReferenceInvalid
		}
	}
	billingExists, err := r.data.db.BillingUnit.Query().Where(billingunitent.IDEQ(input.BillingUnitID), billingunitent.OrganizationIDEQ(input.OrganizationID), billingunitent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	taxableExists, err := r.data.db.TaxableService.Query().Where(taxableserviceent.IDEQ(input.TaxableServiceID), taxableserviceent.OrganizationIDEQ(input.OrganizationID), taxableserviceent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	currencyExists, err := r.data.db.Currency.Query().Where(currencyent.CodeEQ(input.DefaultCurrency), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !billingExists || !taxableExists || !currencyExists {
		return biz.ErrFeeCatalogReferenceInvalid
	}
	return nil
}

func (r *feeCatalogRepo) ListBillingUnits(ctx context.Context, organizationID uuid.UUID) ([]*biz.BillingUnit, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	items, err := r.data.db.BillingUnit.Query().Where(billingunitent.OrganizationIDEQ(headquartersID)).Order(billingunitent.BySortOrder(), billingunitent.ByCode(), billingunitent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.BillingUnit, 0, len(items))
	for _, item := range items {
		result = append(result, billingUnitToBiz(item))
	}
	return result, nil
}

func (r *feeCatalogRepo) CreateBillingUnit(ctx context.Context, input *biz.BillingUnit, audit *biz.AuditEvent) (*biz.BillingUnit, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := tx.BillingUnit.Create().SetID(input.ID).SetOrganizationID(input.OrganizationID).SetCode(input.Code).SetName(input.Name).SetIsContainerUnit(input.IsContainerUnit).SetSortOrder(input.SortOrder).SetEnabled(true).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrBillingUnitCodeExists
		}
		return nil, err
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return billingUnitToBiz(saved), nil
}

func (r *feeCatalogRepo) UpdateBillingUnit(ctx context.Context, input *biz.BillingUnit, audit *biz.AuditEvent) (*biz.BillingUnit, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	current, err := tx.BillingUnit.Query().Where(billingunitent.IDEQ(input.ID), billingunitent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrBillingUnitNotFound
		}
		return nil, err
	}
	saved, err := current.Update().SetCode(input.Code).SetName(input.Name).SetIsContainerUnit(input.IsContainerUnit).SetSortOrder(input.SortOrder).SetEnabled(input.Enabled).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrBillingUnitCodeExists
		}
		return nil, err
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return billingUnitToBiz(saved), nil
}

func (r *feeCatalogRepo) ListTaxableServices(ctx context.Context, organizationID uuid.UUID) ([]*biz.TaxableService, error) {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	items, err := r.data.db.TaxableService.Query().Where(taxableserviceent.OrganizationIDEQ(headquartersID)).Order(taxableserviceent.ByName(), taxableserviceent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.TaxableService, 0, len(items))
	for _, item := range items {
		converted, err := taxableServiceToBiz(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *feeCatalogRepo) CreateTaxableService(ctx context.Context, input *biz.TaxableService, audit *biz.AuditEvent) (*biz.TaxableService, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	saved, err := tx.TaxableService.Create().SetID(input.ID).SetOrganizationID(input.OrganizationID).SetName(input.Name).SetNillableShortName(input.ShortName).SetNillableGoodsCode(input.GoodsCode).SetDefaultTaxRate(input.DefaultTaxRate.StringFixed(2)).SetEnabled(true).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrTaxableServiceNameExists
		}
		return nil, err
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return taxableServiceToBiz(saved)
}

func (r *feeCatalogRepo) UpdateTaxableService(ctx context.Context, input *biz.TaxableService, audit *biz.AuditEvent) (*biz.TaxableService, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	current, err := tx.TaxableService.Query().Where(taxableserviceent.IDEQ(input.ID), taxableserviceent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrTaxableServiceNotFound
		}
		return nil, err
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
	saved, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrTaxableServiceNameExists
		}
		return nil, err
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return taxableServiceToBiz(saved)
}

func feeSettingToBiz(item *ent.FeeSetting) (*biz.FeeSetting, error) {
	taxRate, err := decimal.NewFromString(item.TaxRate)
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
	result := &biz.FeeSetting{ID: item.ID, OrganizationID: item.OrganizationID, FeeCode: item.FeeCode, NameZH: item.NameZh, NameEN: item.NameEn, AliasName: item.AliasName, ServiceTypeID: item.ServiceTypeID, DefaultCurrency: item.DefaultCurrency, BillingUnitID: item.BillingUnitID, BillingUnitName: billingUnit.Name, AbnormalCaseID: item.AbnormalCaseID, TaxRate: taxRate, TaxableServiceID: item.TaxableServiceID, TaxableServiceName: taxableService.Name, Enabled: item.Enabled, SortOrder: item.SortOrder, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.Edges.ServiceType != nil {
		name := item.Edges.ServiceType.Name
		result.ServiceTypeName = &name
	}
	if item.Edges.AbnormalCase != nil {
		name := item.Edges.AbnormalCase.Name
		result.AbnormalCaseName = &name
	}
	return result, nil
}

func billingUnitToBiz(item *ent.BillingUnit) *biz.BillingUnit {
	return &biz.BillingUnit{ID: item.ID, OrganizationID: item.OrganizationID, Code: item.Code, Name: item.Name, IsContainerUnit: item.IsContainerUnit, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func taxableServiceToBiz(item *ent.TaxableService) (*biz.TaxableService, error) {
	taxRate, err := decimal.NewFromString(item.DefaultTaxRate)
	if err != nil {
		return nil, err
	}
	return &biz.TaxableService{ID: item.ID, OrganizationID: item.OrganizationID, Name: item.Name, ShortName: item.ShortName, GoodsCode: item.GoodsCode, DefaultTaxRate: taxRate, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

var _ biz.FeeCatalogRepo = (*feeCatalogRepo)(nil)
