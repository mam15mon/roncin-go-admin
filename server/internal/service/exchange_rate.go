package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type ExchangeRateService struct {
	v1.UnimplementedExchangeRateServiceServer
	usecase *biz.ExchangeRateUsecase
}

func NewExchangeRateService(usecase *biz.ExchangeRateUsecase) *ExchangeRateService {
	return &ExchangeRateService{usecase: usecase}
}

func (s *ExchangeRateService) ListExchangeRateSettings(ctx context.Context, _ *v1.ListExchangeRateSettingsRequest) (*v1.ListExchangeRateSettingsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	items, baseCurrency, err := s.usecase.List(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.ExchangeRateSetting, 0, len(items))
	for _, item := range items {
		data = append(data, exchangeRateToAPI(item))
	}
	return okList(ctx, &v1.ListExchangeRateSettingsResponse{Data: data, BaseCurrency: baseCurrency}), nil
}

func (s *ExchangeRateService) CreateExchangeRateSetting(ctx context.Context, request *v1.CreateExchangeRateSettingRequest) (*v1.CreateExchangeRateSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := exchangeRateInputFromAPI(request.GetFromCurrency(), request.GetToCurrency(), request.GetEffectiveFrom(), request.EffectiveTo, request.GetRate())
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
	input, err := exchangeRateInputFromAPI(request.GetFromCurrency(), request.GetToCurrency(), request.GetEffectiveFrom(), request.EffectiveTo, request.GetRate())
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
	if _, principalErr := biz.RequirePrincipal(ctx); principalErr != nil {
		return nil, principalErr
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

func exchangeRateInputFromAPI(fromCurrency, toCurrency, effectiveFrom string, effectiveTo *string, rateText string) (*biz.ExchangeRateSetting, error) {
	rate, err := parsePlainDecimal(rateText)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return &biz.ExchangeRateSetting{FromCurrency: fromCurrency, ToCurrency: toCurrency, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, Rate: rate}, nil
}

func exchangeRateToAPI(value *biz.ExchangeRateSetting) *v1.ExchangeRateSetting {
	return &v1.ExchangeRateSetting{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), FromCurrency: value.FromCurrency, ToCurrency: value.ToCurrency, EffectiveFrom: value.EffectiveFrom, EffectiveTo: value.EffectiveTo, Rate: value.Rate.StringFixed(8), IsActive: value.IsActive, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
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
		rows = append(rows, &v1.ExchangeRateImportRow{RowNumber: int32(row.RowNumber), FromCurrency: row.FromCurrency, ToCurrency: row.ToCurrency, Rate: row.Rate, EffectiveFrom: row.EffectiveFrom, EffectiveTo: row.EffectiveTo, Status: row.Status, Errors: append([]string(nil), row.Errors...)})
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
