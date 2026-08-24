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
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.ExchangeRateSetting, 0, len(items))
	for _, item := range items {
		data = append(data, exchangeRateToAPI(item))
	}
	return &v1.ListExchangeRateSettingsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *ExchangeRateService) CreateExchangeRateSetting(ctx context.Context, request *v1.CreateExchangeRateSettingRequest) (*v1.CreateExchangeRateSettingResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	input, err := exchangeRateInputFromAPI(request.GetRateType(), request.GetFromCurrency(), request.GetToCurrency(), request.GetTimeStandard(), request.GetEffectiveFrom(), request.EffectiveTo, request.GetReceivableRate(), request.GetPayableRate())
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
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	input, err := exchangeRateInputFromAPI(request.GetRateType(), request.GetFromCurrency(), request.GetToCurrency(), request.GetTimeStandard(), request.GetEffectiveFrom(), request.EffectiveTo, request.GetReceivableRate(), request.GetPayableRate())
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
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
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

func exchangeRateInputFromAPI(rateType, fromCurrency, toCurrency, timeStandard, effectiveFrom string, effectiveTo *string, receivableText, payableText string) (*biz.ExchangeRateSetting, error) {
	receivable, err := parsePlainDecimal(receivableText)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	payable, err := parsePlainDecimal(payableText)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return &biz.ExchangeRateSetting{RateType: rateType, FromCurrency: fromCurrency, ToCurrency: toCurrency, TimeStandard: timeStandard, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, ReceivableRate: receivable, PayableRate: payable}, nil
}

func exchangeRateToAPI(value *biz.ExchangeRateSetting) *v1.ExchangeRateSetting {
	return &v1.ExchangeRateSetting{Id: value.ID.String(), OrganizationId: value.OrganizationID.String(), RateType: value.RateType, FromCurrency: value.FromCurrency, ToCurrency: value.ToCurrency, TimeStandard: value.TimeStandard, EffectiveFrom: value.EffectiveFrom, EffectiveTo: value.EffectiveTo, ReceivableRate: value.ReceivableRate.StringFixed(8), PayableRate: value.PayableRate.StringFixed(8), IsActive: value.IsActive, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339)}
}

var _ v1.ExchangeRateServiceServer = (*ExchangeRateService)(nil)
