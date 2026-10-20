package service

import (
	"context"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *OrderFeeService) PreviewFeeQuantity(ctx context.Context, request *v1.PreviewFeeQuantityRequest) (*v1.PreviewFeeQuantityResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	unitID, err := uuid.Parse(request.GetBillingUnitId())
	if err != nil {
		return nil, biz.ErrOrderFeeInvalidArgument
	}
	preview, err := s.usecase.PreviewFeeQuantity(ctx, principal.Organization.ID, orderID, unitID, request.GetSource())
	if err != nil {
		return nil, err
	}
	inputs := make([]*v1.FeeQuantityInput, 0, len(preview.Inputs))
	for _, input := range preview.Inputs {
		inputs = append(inputs, &v1.FeeQuantityInput{Label: input.Label, Value: input.Value, Unit: input.Unit})
	}
	return ok(ctx, &v1.PreviewFeeQuantityResponse{Data: &v1.FeeQuantityPreview{Available: preview.Available, Quantity: preview.Quantity, SourceLabel: preview.SourceLabel, Formula: preview.Formula, Inputs: inputs, Reason: preview.Reason}}), nil
}
