package service

import (
	"context"

	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func (s *AdminService) ListNotificationTestTypes(ctx context.Context, _ *v1.ListNotificationTestTypesRequest) (*v1.ListNotificationTestTypesResponse, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	catalog, err := s.notificationTest.Catalog(ctx, p)
	if err != nil {
		return nil, err
	}
	types := make([]*v1.NotificationTestType, 0, len(catalog.Types))
	for _, item := range catalog.Types {
		types = append(types, &v1.NotificationTestType{NotificationType: item.Type, Label: item.Label, SupportsOrder: item.SupportsOrder})
	}
	return ok(ctx, &v1.ListNotificationTestTypesResponse{Data: &v1.NotificationTestCatalog{Types: types, RecipientName: catalog.RecipientName, RecipientBound: catalog.RecipientBound, IntegrationEnabled: catalog.IntegrationEnabled, CanSend: catalog.CanSend, SendUnavailableReason: catalog.SendUnavailableReason, CanSelectOrder: catalog.CanSelectOrder, OrderSelectionUnavailableReason: catalog.OrderSelectionUnavailableReason}}), nil
}
func notificationTestOrderID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	return &id, nil
}
func (s *AdminService) PreviewNotificationTest(ctx context.Context, req *v1.PreviewNotificationTestRequest) (*v1.PreviewNotificationTestResponse, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := notificationTestOrderID(req.GetOrderId())
	if err != nil {
		return nil, err
	}
	preview, err := s.notificationTest.Preview(ctx, p, req.GetNotificationType(), orderID)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.PreviewNotificationTestResponse{Data: &v1.NotificationTestPreview{NotificationType: preview.Type, Title: preview.Title, Markdown: preview.Markdown, OrderUrl: preview.OrderURL}}), nil
}
func (s *AdminService) SendNotificationTest(ctx context.Context, req *v1.SendNotificationTestRequest) (*v1.SendNotificationTestResponse, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := notificationTestOrderID(req.GetOrderId())
	if err != nil {
		return nil, err
	}
	if err := s.notificationTest.Send(ctx, p, req.GetNotificationType(), orderID); err != nil {
		return nil, err
	}
	response := ok(ctx, &v1.SendNotificationTestResponse{Data: &v1.NotificationTestAcceptance{Accepted: true}})
	response.Message = "钉钉已受理，请在客户端确认"
	return response, nil
}
