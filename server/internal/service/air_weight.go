package service

import (
	"context"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"time"
)

type AirWeightService struct {
	v1.UnimplementedAirWeightServiceServer
	usecase *biz.AirWeightUsecase
}

func NewAirWeightService(usecase *biz.AirWeightUsecase) *AirWeightService {
	return &AirWeightService{usecase: usecase}
}
func airWeightInput(scope, target, source, divisor string, n, d int32) (*biz.AirWeightInput, error) {
	id, err := uuid.Parse(target)
	if err != nil {
		return nil, biz.ErrAirWeightInvalid
	}
	return &biz.AirWeightInput{Scope: scope, TargetID: id, Source: source, VolumetricDivisor: divisor, BubbleShareNumerator: n, BubbleShareDenominator: d}, nil
}
func airWeightConfirmationToAPI(c *biz.AirWeightConfirmation) *v1.AirWeightConfirmation {
	if c == nil {
		return nil
	}
	return &v1.AirWeightConfirmation{Id: c.ID.String(), Scope: c.Scope, TargetId: c.TargetID.String(), Source: c.Source, Version: c.Version, InputFingerprint: c.InputFingerprint, ConfirmedWeightKg: c.ConfirmedWeightKG, Reason: c.Reason, ConfirmedBy: c.ConfirmedBy.String(), ConfirmedAt: c.ConfirmedAt.Format(time.RFC3339), Snapshot: c.Snapshot}
}
func (s *AirWeightService) PreviewAirWeight(ctx context.Context, req *v1.PreviewAirWeightRequest) (*v1.PreviewAirWeightResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := airWeightInput(req.GetScope(), req.GetTargetId(), req.GetSource(), req.GetVolumetricDivisor(), req.GetBubbleShareNumerator(), req.GetBubbleShareDenominator())
	if err != nil {
		return nil, err
	}
	p, err := s.usecase.Preview(ctx, principal.Organization.ID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewAirWeightResponse{Data: &v1.AirWeightPreview{Scope: p.Scope, TargetId: p.TargetID.String(), Source: p.Source, InputFingerprint: p.InputFingerprint, Available: p.Available, Reason: p.Reason, GrossWeightKg: p.GrossWeightKG, VolumeCbm: p.VolumeCBM, VolumetricWeightKg: p.VolumetricWeightKG, BubbleDifferenceKg: p.BubbleDifferenceKG, SharedBubbleWeightKg: p.SharedBubbleWeightKG, SuggestedChargeableWeightKg: p.SuggestedChargeableWeightKG, VolumetricDivisor: p.VolumetricDivisor, BubbleShareNumerator: p.BubbleShareNumerator, BubbleShareDenominator: p.BubbleShareDenominator, CurrentConfirmation: airWeightConfirmationToAPI(p.CurrentConfirmation), ConfirmationStale: p.ConfirmationStale}}), nil
}
func (s *AirWeightService) ConfirmAirWeight(ctx context.Context, req *v1.ConfirmAirWeightRequest) (*v1.ConfirmAirWeightResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := airWeightInput(req.GetScope(), req.GetTargetId(), req.GetSource(), req.GetVolumetricDivisor(), req.GetBubbleShareNumerator(), req.GetBubbleShareDenominator())
	if err != nil {
		return nil, err
	}
	input.InputFingerprint = req.GetInputFingerprint()
	input.ExpectedVersion = req.GetExpectedVersion()
	input.ConfirmedWeightKG = req.GetConfirmedWeightKg()
	input.Reason = req.GetReason()
	c, err := s.usecase.Confirm(ctx, principal.Organization.ID, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ConfirmAirWeightResponse{Data: airWeightConfirmationToAPI(c)}), nil
}
func (s *AirWeightService) ListAirWeightConfirmations(ctx context.Context, req *v1.ListAirWeightConfirmationsRequest) (*v1.ListAirWeightConfirmationsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetTargetId())
	if err != nil {
		return nil, biz.ErrAirWeightInvalid
	}
	page, size := int(req.GetPage()), int(req.GetPageSize())
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	rows, total, err := s.usecase.List(ctx, principal.Organization.ID, req.GetScope(), id, page, size)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AirWeightConfirmation, 0, len(rows))
	for _, row := range rows {
		data = append(data, airWeightConfirmationToAPI(row))
	}
	return okList(ctx, &v1.ListAirWeightConfirmationsResponse{Data: data, Total: int32(total), Page: int32(page), PageSize: int32(size)}), nil
}
