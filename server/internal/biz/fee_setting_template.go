package biz

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
)

// FeeSettingTemplate 保存公共费用默认值及税务文本，不能引用公司的应税劳务 ID。
type FeeSettingTemplate struct {
	FeeSetting
	TaxableService TaxableService
}

type FeeSettingTemplateRepo interface {
	ListFeeSettingTemplates(context.Context, FeeCatalogListOptions) (*PagedList[*FeeSettingTemplate], error)
	SaveFeeSettingTemplate(context.Context, *FeeSettingTemplate, bool, *AuditEvent) (*FeeSettingTemplate, error)
}

func (uc *FeeCatalogUsecase) ListFeeSettingTemplates(ctx context.Context, options FeeCatalogListOptions) (*PagedList[*FeeSettingTemplate], error) {
	if err := RequireGlobalMasterDataWrite(ctx, access.FinanceFeeSettingRead); err != nil {
		return nil, err
	}
	if !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrFeeCatalogInvalidArgument
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.ListFeeSettingTemplates(ctx, options)
}

func (uc *FeeCatalogUsecase) SaveFeeSettingTemplate(ctx context.Context, id uuid.UUID, input *FeeSettingTemplate) (*FeeSettingTemplate, error) {
	create := id == uuid.Nil
	permission := access.FinanceFeeSettingUpdate
	if create {
		permission = access.FinanceFeeSettingCreate
	}
	if err := RequireGlobalMasterDataWrite(ctx, permission); err != nil {
		return nil, err
	}
	if input == nil {
		return nil, ErrFeeCatalogInvalidArgument
	}
	// 复用字段规则，但模板不保留公司税务外键。
	fee := input.FeeSetting
	normalized, err := normalizeFeeSettingFields(&fee)
	if err != nil {
		return nil, err
	}
	normalized.TaxableServiceID = uuid.Nil
	tax, err := normalizeTaxableService(&input.TaxableService)
	if err != nil {
		return nil, err
	}
	if create {
		id = uuid.Must(uuid.NewV7())
	}
	normalized.ID = id
	p, _ := RequirePrincipal(ctx)
	return uc.repo.SaveFeeSettingTemplate(ctx, &FeeSettingTemplate{FeeSetting: *normalized, TaxableService: *tax}, create, feeCatalogAudit(p.Organization.ID, p.UserID, id, "finance.fee_setting_template.save", "fee_setting_template"))
}
