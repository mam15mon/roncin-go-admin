package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	t "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesettingtemplate"
	tax "github.com/roncin/roncin-go-admin/server/internal/data/ent/taxableservice"
)

func (r *feeCatalogRepo) ListFeeSettingTemplates(ctx context.Context, o biz.FeeCatalogListOptions) (*biz.PagedList[*biz.FeeSettingTemplate], error) {
	c, e := r.data.client(ctx)
	if e != nil {
		return nil, e
	}
	q := c.FeeSettingTemplate.Query()
	if o.Keyword != "" {
		q.Where(t.Or(t.FeeCodeContainsFold(o.Keyword), t.NameZhContainsFold(o.Keyword), t.NameEnContainsFold(o.Keyword), t.AliasNameContainsFold(o.Keyword), t.SearchKeywordsContainsFold(o.Keyword)))
	}
	return paginate(ctx, q.Count, func(ctx context.Context, offset, limit int) ([]*ent.FeeSettingTemplate, error) {
		return q.Order(t.BySortOrder(), t.ByFeeCode()).Offset(offset).Limit(limit).All(ctx)
	}, o.Page, o.PageSize, feeTemplateToBiz)
}
func (r *feeCatalogRepo) SaveFeeSettingTemplate(ctx context.Context, v *biz.FeeSettingTemplate, create bool, audit *biz.AuditEvent) (*biz.FeeSettingTemplate, error) {
	var saved *ent.FeeSettingTemplate
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var e error
		if err := validateFeePublicReferences(ctx, tx.Client(), &v.FeeSetting); err != nil {
			return err
		}
		if create {
			b := tx.FeeSettingTemplate.Create().SetID(v.ID).SetFeeCode(v.FeeCode).SetNameZh(v.NameZH).SetChargeCategoryID(v.ChargeCategoryID).SetDefaultCurrency(v.DefaultCurrency).SetBillingUnitID(v.BillingUnitID).SetTaxRate(v.TaxRate.StringFixed(2)).SetEnabled(v.Enabled).SetSortOrder(v.SortOrder).SetTaxableServiceName(v.TaxableService.Name).SetTaxableServiceDefaultTaxRate(v.TaxableService.DefaultTaxRate.StringFixed(2))
			b.SetNillableNameEn(v.NameEN)
			b.SetNillableAliasName(v.AliasName)
			b.SetNillableAbnormalCaseID(v.AbnormalCaseID)
			b.SetNillableTaxableServiceShortName(v.TaxableService.ShortName)
			b.SetNillableTaxableServiceGoodsCode(v.TaxableService.GoodsCode)
			saved, e = b.Save(ctx)
		} else {
			current, err := tx.FeeSettingTemplate.Query().Where(t.IDEQ(v.ID)).ForUpdate().Only(ctx)
			if err != nil {
				return mapEntError(err, biz.ErrFeeSettingNotFound, nil)
			}
			b := current.Update().SetFeeCode(v.FeeCode).SetNameZh(v.NameZH).SetChargeCategoryID(v.ChargeCategoryID).SetDefaultCurrency(v.DefaultCurrency).SetBillingUnitID(v.BillingUnitID).SetTaxRate(v.TaxRate.StringFixed(2)).SetEnabled(v.Enabled).SetSortOrder(v.SortOrder).SetTaxableServiceName(v.TaxableService.Name).SetTaxableServiceDefaultTaxRate(v.TaxableService.DefaultTaxRate.StringFixed(2))
			if v.NameEN == nil {
				b.ClearNameEn()
			} else {
				b.SetNameEn(*v.NameEN)
			}
			if v.AliasName == nil {
				b.ClearAliasName()
			} else {
				b.SetAliasName(*v.AliasName)
			}
			if v.AbnormalCaseID == nil {
				b.ClearAbnormalCaseID()
			} else {
				b.SetAbnormalCaseID(*v.AbnormalCaseID)
			}
			if v.TaxableService.ShortName == nil {
				b.ClearTaxableServiceShortName()
			} else {
				b.SetTaxableServiceShortName(*v.TaxableService.ShortName)
			}
			if v.TaxableService.GoodsCode == nil {
				b.ClearTaxableServiceGoodsCode()
			} else {
				b.SetTaxableServiceGoodsCode(*v.TaxableService.GoodsCode)
			}
			saved, e = b.Save(ctx)
		}
		if e != nil {
			return mapEntError(e, nil, biz.ErrFeeSettingCodeExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return feeTemplateToBiz(saved)
}
func feeTemplateToBiz(v *ent.FeeSettingTemplate) (*biz.FeeSettingTemplate, error) {
	rate, e := decimalOf(v.TaxRate)
	if e != nil {
		return nil, e
	}
	taxRate, e := decimalOf(v.TaxableServiceDefaultTaxRate)
	if e != nil {
		return nil, e
	}
	return &biz.FeeSettingTemplate{FeeSetting: biz.FeeSetting{ID: v.ID, FeeCode: v.FeeCode, NameZH: v.NameZh, NameEN: v.NameEn, AliasName: v.AliasName, ChargeCategoryID: v.ChargeCategoryID, DefaultCurrency: v.DefaultCurrency, BillingUnitID: v.BillingUnitID, AbnormalCaseID: v.AbnormalCaseID, TaxRate: rate, SortOrder: v.SortOrder, Enabled: v.Enabled, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, TaxableService: biz.TaxableService{Name: v.TaxableServiceName, ShortName: v.TaxableServiceShortName, GoodsCode: v.TaxableServiceGoodsCode, DefaultTaxRate: taxRate}}, nil
}

// initializeCompanyFeeSettings 仅在新建公司事务中复制一次，不与后续模板修改联动。
func initializeCompanyFeeSettings(ctx context.Context, c *ent.Client, organizationID uuid.UUID) error {
	templates, err := c.FeeSettingTemplate.Query().Where(t.EnabledEQ(true)).Order(t.ByID()).ForShare().All(ctx)
	if err != nil {
		return err
	}
	for _, v := range templates {
		normalized, err := feeTemplateToBiz(v)
		if err != nil {
			return err
		}
		if err := validateFeePublicReferences(ctx, c, &normalized.FeeSetting); err != nil {
			return err
		}
		taxable, err := c.TaxableService.Query().Where(tax.OrganizationIDEQ(organizationID), tax.NameEQ(v.TaxableServiceName)).Only(ctx)
		if ent.IsNotFound(err) {
			taxable, err = c.TaxableService.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(organizationID).SetName(v.TaxableServiceName).SetNillableShortName(v.TaxableServiceShortName).SetNillableGoodsCode(v.TaxableServiceGoodsCode).SetDefaultTaxRate(v.TaxableServiceDefaultTaxRate).SetEnabled(true).Save(ctx)
		}
		if err != nil {
			return err
		}
		if taxable.DefaultTaxRate != v.TaxableServiceDefaultTaxRate || !sameOptionalText(taxable.ShortName, v.TaxableServiceShortName) || !sameOptionalText(taxable.GoodsCode, v.TaxableServiceGoodsCode) {
			return biz.ErrFeeCatalogReferenceInvalid
		}
		_, err = c.FeeSetting.Create().SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(organizationID).SetFeeCode(v.FeeCode).SetNameZh(v.NameZh).SetNillableNameEn(v.NameEn).SetNillableAliasName(v.AliasName).SetChargeCategoryID(v.ChargeCategoryID).SetDefaultCurrency(v.DefaultCurrency).SetBillingUnitID(v.BillingUnitID).SetNillableAbnormalCaseID(v.AbnormalCaseID).SetTaxRate(v.TaxRate).SetTaxableServiceID(taxable.ID).SetEnabled(true).SetSortOrder(v.SortOrder).Save(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
func sameOptionalText(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
