package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

type OrderMilestoneService struct {
	v1.UnimplementedOrderMilestoneServiceServer
	usecase *biz.OrderMilestoneUsecase
}

func NewOrderMilestoneService(usecase *biz.OrderMilestoneUsecase) *OrderMilestoneService {
	return &OrderMilestoneService{usecase: usecase}
}

func (s *OrderMilestoneService) ListMilestones(ctx context.Context, request *v1.ListMilestonesRequest) (*v1.ListMilestonesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	items, err := s.usecase.List(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.OrderMilestone, 0, len(items))
	for _, item := range items {
		data = append(data, orderMilestoneToAPI(item))
	}
	return &v1.ListMilestonesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *OrderMilestoneService) SetMilestone(ctx context.Context, request *v1.SetMilestoneRequest) (*v1.SetMilestoneResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderNotFound
	}
	expected := request.GetExpectedOrderVersion()
	occurredAt, err := parseOptionalRFC3339(request.GetOccurredAt())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	var note *string
	if request.Note != nil {
		value := request.GetNote()
		note = &value
	}
	item, err := s.usecase.Set(ctx, principal.Organization.ID, principal.UserID, orderID, request.GetType(), expected, occurredAt, note, request.GetClearOccurredAt())
	if err != nil {
		return nil, err
	}
	return &v1.SetMilestoneResponse{Success: true, Code: 0, Message: "OK", Data: orderMilestoneToAPI(item), TraceId: requestmeta.TraceID(ctx)}, nil
}

func parseOptionalRFC3339(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func orderMilestoneToAPI(item *biz.OrderMilestone) *v1.OrderMilestone {
	result := &v1.OrderMilestone{Id: item.ID.String(), OrderId: item.OrderID.String(), Type: item.Type, TemplateNodeCode: item.TemplateNodeCode, TemplateNodeLabel: item.TemplateNodeLabel, OccurredAt: timeStringPtr(item.OccurredAt), Note: item.Note, UpdatedBy: uuidStringPtr(item.UpdatedBy), CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
	return result
}

func timeStringPtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	result := value.UTC().Format(time.RFC3339)
	return &result
}

var _ v1.OrderMilestoneServiceServer = (*OrderMilestoneService)(nil)
