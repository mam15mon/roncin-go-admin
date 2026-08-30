package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
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
	return &v1.ListExchangeRateSettingsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx), BaseCurrency: baseCurrency}, nil
}

func (s *ExchangeRateService) CreateExchangeRateSetting(ctx context.Context, request *v1.CreateExchangeRateSettingRequest) (*v1.CreateExchangeRateSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	input, err := exchangeRateInputFromAPI(request.GetRateType(), request.GetFromCurrency(), request.GetToCurrency(), request.GetEffectiveFrom(), request.EffectiveTo, request.GetReceivableRate(), request.GetPayableRate())
	if err != nil {
		return nil, err
	}
	created, err := s.usecase.Create(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return &v1.CreateExchangeRateSettingResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
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
	input, err := exchangeRateInputFromAPI(request.GetRateType(), request.GetFromCurrency(), request.GetToCurrency(), request.GetEffectiveFrom(), request.EffectiveTo, request.GetReceivableRate(), request.GetPayableRate())
	if err != nil {
		return nil, err
	}
	updated, err := s.usecase.Update(ctx, principal.Organization.ID, principal.UserID, id, input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateExchangeRateSettingResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
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
	return &v1.DisableExchangeRateSettingResponse{Success: true, Code: 0, Message: "OK", TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *ExchangeRateService) ListExchangeRateTimeStandards(ctx context.Context, _ *v1.ListExchangeRateTimeStandardsRequest) (*v1.ListExchangeRateTimeStandardsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	settings, err := s.usecase.ListTimeStandards(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return &v1.ListExchangeRateTimeStandardsResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateTimeStandardsToAPI(settings), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *ExchangeRateService) UpdateExchangeRateTimeStandards(ctx context.Context, request *v1.UpdateExchangeRateTimeStandardsRequest) (*v1.UpdateExchangeRateTimeStandardsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	settings := make([]*biz.ExchangeRateTimeStandardSetting, 0, len(request.GetData()))
	for _, item := range request.GetData() {
		settings = append(settings, &biz.ExchangeRateTimeStandardSetting{RateType: item.GetRateType(), TimeStandards: append([]string(nil), item.GetTimeStandards()...)})
	}
	updated, err := s.usecase.UpdateTimeStandards(ctx, principal.Organization.ID, principal.UserID, settings)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateExchangeRateTimeStandardsResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateTimeStandardsToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *ExchangeRateService) GetExchangeRateCustomSetting(ctx context.Context, _ *v1.GetExchangeRateCustomSettingRequest) (*v1.GetExchangeRateCustomSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	setting, err := s.usecase.GetCustomSetting(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	return &v1.GetExchangeRateCustomSettingResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateCustomSettingToAPI(setting), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *ExchangeRateService) UpdateExchangeRateCustomSetting(ctx context.Context, request *v1.UpdateExchangeRateCustomSettingRequest) (*v1.UpdateExchangeRateCustomSettingResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	if request == nil || request.ExpectedVersion == nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	setting, err := s.usecase.UpdateCustomSetting(ctx, principal.Organization.ID, principal.UserID, request.GetInheritBaseCurrencyRate(), request.GetExpectedVersion().GetValue())
	if err != nil {
		return nil, err
	}
	return &v1.UpdateExchangeRateCustomSettingResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateCustomSettingToAPI(setting), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *ExchangeRateService) DownloadExchangeRateImportTemplate(ctx context.Context, _ *v1.DownloadExchangeRateImportTemplateRequest) (*v1.DownloadExchangeRateImportTemplateResponse, error) {
	if _, principalErr := biz.RequirePrincipal(ctx); principalErr != nil {
		return nil, principalErr
	}
	content, err := buildExchangeRateImportTemplate()
	if err != nil {
		return nil, err
	}
	return &v1.DownloadExchangeRateImportTemplateResponse{Success: true, Code: 0, Message: "OK", FileName: "汇率导入模板.xlsx", ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Content: content, TemplateVersion: biz.ExchangeRateImportTemplateVersion, TraceId: requestmeta.TraceID(ctx)}, nil
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
	return &v1.PreviewExchangeRateImportResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateImportBatchToAPI(batch), PreviewToken: token, TraceId: requestmeta.TraceID(ctx)}, nil
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
	return &v1.ConfirmExchangeRateImportResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateImportBatchToAPI(batch), TraceId: requestmeta.TraceID(ctx)}, nil
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
	return &v1.GetExchangeRateImportResponse{Success: true, Code: 0, Message: "OK", Data: exchangeRateImportBatchToAPI(batch), TraceId: requestmeta.TraceID(ctx)}, nil
}

func exchangeRateInputFromAPI(rateType, fromCurrency, toCurrency, effectiveFrom string, effectiveTo *string, receivableText, payableText string) (*biz.ExchangeRateSetting, error) {
	receivable, err := parsePlainDecimal(receivableText)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	payable, err := parsePlainDecimal(payableText)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return &biz.ExchangeRateSetting{RateType: rateType, FromCurrency: fromCurrency, ToCurrency: toCurrency, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, ReceivableRate: receivable, PayableRate: payable}, nil
}

func exchangeRateToAPI(value *biz.ExchangeRateSetting) *v1.ExchangeRateSetting {
	return &v1.ExchangeRateSetting{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), RateType: value.RateType, FromCurrency: value.FromCurrency, ToCurrency: value.ToCurrency, EffectiveFrom: value.EffectiveFrom, EffectiveTo: value.EffectiveTo, ReceivableRate: value.ReceivableRate.StringFixed(8), PayableRate: value.PayableRate.StringFixed(8), IsActive: value.IsActive, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

func exchangeRateTimeStandardsToAPI(settings []*biz.ExchangeRateTimeStandardSetting) []*v1.ExchangeRateTimeStandardSetting {
	result := make([]*v1.ExchangeRateTimeStandardSetting, 0, len(settings))
	for _, setting := range settings {
		result = append(result, &v1.ExchangeRateTimeStandardSetting{RateType: setting.RateType, TimeStandards: append([]string(nil), setting.TimeStandards...)})
	}
	return result
}

func exchangeRateCustomSettingToAPI(setting *biz.ExchangeRateCustomSetting) *v1.ExchangeRateCustomSetting {
	if setting == nil {
		return nil
	}
	result := &v1.ExchangeRateCustomSetting{
		OrganizationId:          setting.OrganizationID.String(),
		InheritBaseCurrencyRate: setting.InheritBaseCurrencyRate,
		Version:                 setting.Version,
	}
	if setting.UpdatedAt != nil {
		value := setting.UpdatedAt.UTC().Format(time.RFC3339)
		result.UpdatedAt = &value
	}
	if setting.UpdatedBy != nil {
		value := setting.UpdatedBy.String()
		result.UpdatedBy = &value
	}
	return result
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
		rows = append(rows, &v1.ExchangeRateImportRow{RowNumber: int32(row.RowNumber), RateType: row.RateType, FromCurrency: row.FromCurrency, ToCurrency: row.ToCurrency, ReceivableRate: row.ReceivableRate, PayableRate: row.PayableRate, EffectiveFrom: row.EffectiveFrom, EffectiveTo: row.EffectiveTo, Status: row.Status, Errors: append([]string(nil), row.Errors...)})
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
