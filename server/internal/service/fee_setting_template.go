package service

import (
	"context"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"time"
)

func (s *FeeCatalogService) ListFeeSettingTemplates(ctx context.Context, r *v1.ListFeeSettingTemplatesRequest) (*v1.ListFeeSettingTemplatesResponse, error) {
	page, size, err := listPageValues(r.GetPage(), r.GetPageSize(), biz.ErrFeeCatalogInvalidArgument)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.ListFeeSettingTemplates(ctx, biz.FeeCatalogListOptions{Page: page, PageSize: size, Keyword: r.GetKeyword()})
	if err != nil {
		return nil, err
	}
	items := make([]*v1.FeeSettingTemplate, 0, len(result.Items))
	for _, v := range result.Items {
		items = append(items, feeTemplateToAPI(v))
	}
	return okList(ctx, &v1.ListFeeSettingTemplatesResponse{Data: items, Total: int32(result.Total), Page: int32(result.Page), PageSize: int32(result.PageSize)}), nil
}
func (s *FeeCatalogService) CreateFeeSettingTemplate(ctx context.Context, r *v1.CreateFeeSettingTemplateRequest) (*v1.CreateFeeSettingTemplateResponse, error) {
	input, err := feeTemplateFromAPI(r.GetInput())
	if err != nil {
		return nil, err
	}
	v, err := s.usecase.SaveFeeSettingTemplate(ctx, uuid.Nil, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateFeeSettingTemplateResponse{Data: feeTemplateToAPI(v)}), nil
}
func (s *FeeCatalogService) UpdateFeeSettingTemplate(ctx context.Context, r *v1.UpdateFeeSettingTemplateRequest) (*v1.UpdateFeeSettingTemplateResponse, error) {
	id, err := uuid.Parse(r.GetId())
	if err != nil || id == uuid.Nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	input, err := feeTemplateFromAPI(r.GetInput())
	if err != nil {
		return nil, err
	}
	v, err := s.usecase.SaveFeeSettingTemplate(ctx, id, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateFeeSettingTemplateResponse{Data: feeTemplateToAPI(v)}), nil
}
func feeTemplateFromAPI(v *v1.FeeSettingTemplateInput) (*biz.FeeSettingTemplate, error) {
	if v == nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	unit, e := uuid.Parse(v.BillingUnitId)
	if e != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	category, e := uuid.Parse(v.ChargeCategoryId)
	if e != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	abnormal, e := optionalCatalogUUID(v.AbnormalCaseId)
	if e != nil {
		return nil, e
	}
	rate, e := parsePlainDecimal(v.TaxRate)
	if e != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	taxRate, e := parsePlainDecimal(v.TaxableServiceDefaultTaxRate)
	if e != nil {
		return nil, biz.ErrFeeCatalogInvalidArgument
	}
	return &biz.FeeSettingTemplate{FeeSetting: biz.FeeSetting{FeeCode: v.FeeCode, NameZH: v.NameZh, NameEN: v.NameEn, AliasName: v.AliasName, ChargeCategoryID: category, DefaultCurrency: v.DefaultCurrency, BillingUnitID: unit, AbnormalCaseID: abnormal, TaxRate: rate, SortOrder: int(v.SortOrder), Enabled: v.Enabled}, TaxableService: biz.TaxableService{Name: v.TaxableServiceName, ShortName: v.TaxableServiceShortName, GoodsCode: v.TaxableServiceGoodsCode, DefaultTaxRate: taxRate}}, nil
}
func feeTemplateToAPI(v *biz.FeeSettingTemplate) *v1.FeeSettingTemplate {
	input := &v1.FeeSettingTemplateInput{FeeCode: v.FeeCode, NameZh: v.NameZH, NameEn: v.NameEN, AliasName: v.AliasName, ChargeCategoryId: v.ChargeCategoryID.String(), DefaultCurrency: v.DefaultCurrency, BillingUnitId: v.BillingUnitID.String(), TaxRate: v.TaxRate.StringFixed(2), SortOrder: int32(v.SortOrder), Enabled: v.Enabled, TaxableServiceName: v.TaxableService.Name, TaxableServiceShortName: v.TaxableService.ShortName, TaxableServiceGoodsCode: v.TaxableService.GoodsCode, TaxableServiceDefaultTaxRate: v.TaxableService.DefaultTaxRate.StringFixed(2)}
	if v.AbnormalCaseID != nil {
		s := v.AbnormalCaseID.String()
		input.AbnormalCaseId = &s
	}
	return &v1.FeeSettingTemplate{Id: v.ID.String(), Input: input, CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339)}
}
