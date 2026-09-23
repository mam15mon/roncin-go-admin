package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

type ExchangeRateService struct {
	v1.UnimplementedExchangeRateServiceServer
	usecase *biz.ExchangeRateUsecase
}

func NewExchangeRateService(usecase *biz.ExchangeRateUsecase) *ExchangeRateService {
	return &ExchangeRateService{usecase: usecase}
}

func (s *ExchangeRateService) ListExchangeRateSettings(ctx context.Context, request *v1.ListExchangeRateSettingsRequest) (*v1.ListExchangeRateSettingsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	page, pageSize := biz.ListPagination(int(request.GetPage()), int(request.GetPageSize()), 20)
	list, baseCurrency, err := s.usecase.List(ctx, principal.Organization.ID, biz.ExchangeRateListOptions{
		Page:         page,
		PageSize:     pageSize,
		FromCurrency: request.GetFromCurrency(),
	})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.ExchangeRateSetting, 0, len(list.Items))
	for _, item := range list.Items {
		data = append(data, exchangeRateToAPI(item))
	}
	return okList(ctx, &v1.ListExchangeRateSettingsResponse{
		Data:         data,
		BaseCurrency: baseCurrency,
		Total:        int64(list.Total),
	}), nil
}

func (s *ExchangeRateService) CreateExchangeRateSetting(ctx context.Context, request *v1.CreateExchangeRateSettingRequest) (*v1.CreateExchangeRateSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := exchangeRateInputFromAPI(request.GetFromCurrency(), request.GetToCurrency(), request.GetEffectiveFrom(), request.GetArRate(), request.GetApRate(), request.GetRate())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateExchangeRateSettingResponse{Data: exchangeRateToAPI(created)}), nil
}

func (s *ExchangeRateService) UpdateExchangeRateSetting(ctx context.Context, request *v1.UpdateExchangeRateSettingRequest) (*v1.UpdateExchangeRateSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	input, err := exchangeRateInputFromAPI(request.GetFromCurrency(), request.GetToCurrency(), request.GetEffectiveFrom(), request.GetArRate(), request.GetApRate(), request.GetRate())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, id, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.UpdateExchangeRateSettingResponse{Data: exchangeRateToAPI(updated)}), nil
}

func (s *ExchangeRateService) DisableExchangeRateSetting(ctx context.Context, request *v1.DisableExchangeRateSettingRequest) (*v1.DisableExchangeRateSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	if err := s.usecase.Disable(ctx, principal.Organization.ID, principal.UserID, id); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.DisableExchangeRateSettingResponse{}), nil
}

func (s *ExchangeRateService) DownloadExchangeRateImportTemplate(ctx context.Context, _ *v1.DownloadExchangeRateImportTemplateRequest) (*v1.DownloadExchangeRateImportTemplateResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	if !principal.HasPermission(access.FinanceExchangeRateRead) {
		return nil, biz.ErrExchangeRatePermissionDenied
	}
	if _, err := s.usecase.BaseCurrency(ctx, principal.Organization.ID); err != nil {
		return nil, err
	}
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.DownloadExchangeRateImportTemplateResponse{FileName: "汇率导入模板.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: content, TemplateVersion: biz.ExchangeRateImportTemplateVersion}), nil
}

func (s *ExchangeRateService) PreviewExchangeRateImport(ctx context.Context, request *v1.PreviewExchangeRateImportRequest) (*v1.PreviewExchangeRateImportResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := parseExchangeRateImportWorkbook(request.GetFileName(), request.GetFileContent())
	if err != nil {
		return nil, err
	}
	batch, token, err := s.usecase.PreviewImport(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewExchangeRateImportResponse{Data: exchangeRateImportBatchToAPI(batch), PreviewToken: token}), nil
}

func (s *ExchangeRateService) ConfirmExchangeRateImport(ctx context.Context, request *v1.ConfirmExchangeRateImportRequest) (*v1.ConfirmExchangeRateImportResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	batch, err := s.usecase.ConfirmImport(ctx, principal.Organization.ID, principal.UserID, request.GetPreviewToken(), request.GetIdempotencyKey())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmExchangeRateImportResponse{Data: exchangeRateImportBatchToAPI(batch)}), nil
}

func (s *ExchangeRateService) GetExchangeRateImport(ctx context.Context, request *v1.GetExchangeRateImportRequest) (*v1.GetExchangeRateImportResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	batch, err := s.usecase.GetImport(ctx, principal.Organization.ID, id)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetExchangeRateImportResponse{Data: exchangeRateImportBatchToAPI(batch)}), nil
}

// FetchExchangeRates 按当前组织本币与目标周抓取牌价预览，不落库；
// 抓取失败返回业务错误，由前端显式引导手工录入。
func (s *ExchangeRateService) FetchExchangeRates(ctx context.Context, request *v1.FetchExchangeRatesRequest) (*v1.FetchExchangeRatesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	target, err := exchangeRateSyncTargetFromAPI(request.GetTarget())
	if err != nil {
		return nil, err
	}
	preview, err := s.usecase.FetchExchangeRates(ctx, principal.Organization.ID, target)
	if err != nil {
		return nil, err
	}
	rows := make([]*v1.ExchangeRateSyncPreviewRow, 0, len(preview.Rows))
	for _, row := range preview.Rows {
		rows = append(rows, &v1.ExchangeRateSyncPreviewRow{
			FromCurrency:   row.FromCurrency,
			ArRate:         row.ARRate.StringFixed(8),
			ApRate:         row.APRate.StringFixed(8),
			Rate:           row.Rate.StringFixed(8),
			ConversionPath: row.ConversionPath,
		})
	}
	return ok(ctx, &v1.FetchExchangeRatesResponse{Data: &v1.ExchangeRateSyncPreview{
		Target:        exchangeRateSyncTargetToAPI(preview.Target),
		BaseCurrency:  preview.BaseCurrency,
		EffectiveFrom: preview.EffectiveFrom,
		EffectiveTo:   preview.EffectiveTo,
		Source:        preview.Source,
		FallbackUsed:  preview.FallbackUsed,
		Rows:          rows,
	}}), nil
}

// SyncExchangeRates 将财务终审（含微调）后的牌价按自然周幂等入库生效。
func (s *ExchangeRateService) SyncExchangeRates(ctx context.Context, request *v1.SyncExchangeRatesRequest) (*v1.SyncExchangeRatesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	target, err := exchangeRateSyncTargetFromAPI(request.GetTarget())
	if err != nil {
		return nil, err
	}
	rows := make([]*biz.ExchangeRateSyncRow, 0, len(request.GetRows()))
	for _, row := range request.GetRows() {
		arRate, arErr := parsePlainDecimal(row.GetArRate())
		apRate, apErr := parsePlainDecimal(row.GetApRate())
		rate := decimal.Decimal{}
		if rateText := strings.TrimSpace(row.GetRate()); rateText != "" {
			rate, err = parsePlainDecimal(rateText)
			if err != nil {
				return nil, biz.ErrExchangeRateSyncRowsInvalid
			}
		}
		if arErr != nil || apErr != nil {
			return nil, biz.ErrExchangeRateSyncRowsInvalid
		}
		rows = append(rows, &biz.ExchangeRateSyncRow{FromCurrency: row.GetFromCurrency(), ARRate: arRate, APRate: apRate, Rate: rate})
	}
	syncedCount, effectiveFrom, effectiveTo, err := s.usecase.SyncExchangeRates(ctx, principal.Organization.ID, principal.UserID, target, rows)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.SyncExchangeRatesResponse{SyncedCount: int32(syncedCount), EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo}), nil
}

func exchangeRateSyncTargetFromAPI(target v1.ExchangeRateSyncTarget) (biz.ExchangeRateSyncTarget, error) {
	switch target {
	case v1.ExchangeRateSyncTarget_EXCHANGE_RATE_SYNC_TARGET_CURRENT_WEEK:
		return biz.ExchangeRateSyncTargetCurrentWeek, nil
	case v1.ExchangeRateSyncTarget_EXCHANGE_RATE_SYNC_TARGET_NEXT_WEEK:
		return biz.ExchangeRateSyncTargetNextWeek, nil
	default:
		return "", biz.ErrExchangeRateSyncTargetInvalid
	}
}

func exchangeRateSyncTargetToAPI(target biz.ExchangeRateSyncTarget) v1.ExchangeRateSyncTarget {
	switch target {
	case biz.ExchangeRateSyncTargetCurrentWeek:
		return v1.ExchangeRateSyncTarget_EXCHANGE_RATE_SYNC_TARGET_CURRENT_WEEK
	case biz.ExchangeRateSyncTargetNextWeek:
		return v1.ExchangeRateSyncTarget_EXCHANGE_RATE_SYNC_TARGET_NEXT_WEEK
	default:
		return v1.ExchangeRateSyncTarget_EXCHANGE_RATE_SYNC_TARGET_UNSPECIFIED
	}
}

// exchangeRateInputFromAPI 应收/应付双轨汇率必填；基准汇率可空（缺省按中间价记录），
// 生效时刻归一化为所在自然周窗口。
func exchangeRateInputFromAPI(fromCurrency, toCurrency, effectiveFrom string, arRateText, apRateText, rateText string) (*biz.ExchangeRateSetting, error) {
	arRate, arErr := parsePlainDecimal(arRateText)
	apRate, apErr := parsePlainDecimal(apRateText)
	if arErr != nil || apErr != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	input := &biz.ExchangeRateSetting{FromCurrency: fromCurrency, ToCurrency: toCurrency, EffectiveFrom: effectiveFrom, ARRate: &arRate, APRate: &apRate}
	if rateText = strings.TrimSpace(rateText); rateText != "" {
		rate, rateErr := parsePlainDecimal(rateText)
		if rateErr != nil {
			return nil, biz.ErrExchangeRateInvalidArgument
		}
		input.Rate = rate
	}
	return input, nil
}

func exchangeRateToAPI(value *biz.ExchangeRateSetting) *v1.ExchangeRateSetting {
	// 存量公共行可能为空归属；新配置恒返回所属公司 ID。
	organizationID := ""
	if value.OrganizationID != nil {
		organizationID = value.OrganizationID.String()
	}
	arRate, apRate := "", ""
	if value.ARRate != nil {
		arRate = value.ARRate.StringFixed(8)
	}
	if value.APRate != nil {
		apRate = value.APRate.StringFixed(8)
	}
	return &v1.ExchangeRateSetting{Id: value.ID.String(), OrganizationId: organizationID, FromCurrency: value.FromCurrency, ToCurrency: value.ToCurrency, EffectiveFrom: value.EffectiveFrom, EffectiveTo: value.EffectiveTo, ArRate: arRate, ApRate: apRate, Rate: value.Rate.StringFixed(8), IsActive: value.IsActive, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

func exchangeRateImportBatchToAPI(batch *biz.ExchangeRateImportBatch) *v1.ExchangeRateImportBatch {
	if batch == nil {
		return nil
	}
	rows := make([]*v1.ExchangeRateImportRow, 0, len(batch.Rows))
	for _, row := range batch.Rows {
		if row == nil {
			continue
		}
		rows = append(rows, &v1.ExchangeRateImportRow{RowNumber: int32(row.RowNumber), FromCurrency: row.FromCurrency, ToCurrency: row.ToCurrency, ArRate: row.ARRate, ApRate: row.APRate, Rate: row.Rate, EffectiveFrom: row.EffectiveFrom, EffectiveTo: row.EffectiveTo, Status: row.Status, Errors: append([]string(nil), row.Errors...)})
	}
	return &v1.ExchangeRateImportBatch{Id: batch.ID.String(), FileName: batch.FileName, FileChecksum: batch.FileChecksum, TemplateVersion: int32(batch.TemplateVersion), Status: batch.Status, TotalCount: int32(batch.TotalCount), ValidCount: int32(batch.ValidCount), InvalidCount: int32(batch.InvalidCount), ImportedCount: int32(batch.ImportedCount), CanConfirm: batch.Status == biz.ExchangeRateImportPreviewReady && batch.InvalidCount == 0 && batch.ExpiresAt.After(time.Now()), Rows: rows, ExpiresAt: batch.ExpiresAt.UTC().Format(time.RFC3339), ImportedAt: exchangeRateImportTime(batch.ImportedAt), CreatedAt: batch.CreatedAt.UTC().Format(time.RFC3339)}
}

func exchangeRateImportTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

var _ v1.ExchangeRateServiceServer = (*ExchangeRateService)(nil)
