package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *OrderService) GetImportHandling(ctx context.Context, request *v1.GetImportHandlingRequest) (*v1.GetImportHandlingResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrImportHandlingInvalid
	}
	value, err := s.usecase.GetImportHandling(ctx, principal.Organization.ID, orderID)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.GetImportHandlingResponse{Data: importHandlingToAPI(value)}), nil
}
func (s *OrderService) SaveImportHandling(ctx context.Context, request *v1.SaveImportHandlingRequest) (*v1.SaveImportHandlingResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrImportHandlingInvalid
	}
	agent, err := parseOptionalUUIDPointer(request.ExchangeAgentId)
	if err != nil {
		return nil, err
	}
	input := &biz.OrderImportHandling{OrderID: orderID, ExchangeAgentID: agent}
	for _, v := range request.GetSteps() {
		if v == nil {
			return nil, biz.ErrImportHandlingInvalid
		}
		kind := strings.TrimPrefix(v.GetKind().String(), "IMPORT_HANDLING_STEP_KIND_")
		step := biz.ImportHandlingStep{Kind: kind, Undertaken: v.GetUndertaken(), ReferenceNo: v.GetReferenceNo(), Notes: v.GetNotes()}
		if v.OccurredAt != nil {
			t, err := time.Parse(time.RFC3339, *v.OccurredAt)
			if err != nil {
				return nil, biz.ErrImportHandlingInvalid
			}
			t = t.UTC()
			step.OccurredAt = &t
		}
		step.ResponsibleUserID, err = parseOptionalUUIDPointer(v.ResponsibleUserId)
		if err != nil {
			return nil, err
		}
		step.AttachmentIDs, err = parseUUIDValues(v.AttachmentIds, biz.ErrImportHandlingInvalid)
		if err != nil {
			return nil, err
		}
		input.Steps = append(input.Steps, step)
	}
	value, err := s.usecase.SaveImportHandling(ctx, principal.Organization.ID, principal.UserID, orderID, request.GetExpectedVersion(), input, request.GetReason())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.SaveImportHandlingResponse{Data: importHandlingToAPI(value)}), nil
}
func (s *OrderService) ListImportHandlingEvents(ctx context.Context, request *v1.ListImportHandlingEventsRequest) (*v1.ListImportHandlingEventsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(request.GetOrderId())
	if err != nil {
		return nil, biz.ErrImportHandlingInvalid
	}
	values, total, err := s.usecase.ListImportHandlingEvents(ctx, principal.Organization.ID, orderID, int(request.GetPage()), int(request.GetPageSize()))
	if err != nil {
		return nil, err
	}
	out := &v1.ListImportHandlingEventsData{Total: int32(total), Items: []*v1.ImportHandlingEvent{}}
	for _, v := range values {
		out.Items = append(out.Items, &v1.ImportHandlingEvent{Id: v.ID.String(), OrderId: v.OrderID.String(), ActorId: v.ActorID.String(), ActorName: v.ActorName, OccurredAt: v.OccurredAt.UTC().Format(time.RFC3339), Reason: v.Reason, Before: importHandlingToAPI(v.Before), After: importHandlingToAPI(v.After)})
	}
	return ok(ctx, &v1.ListImportHandlingEventsResponse{Data: out}), nil
}
func importHandlingToAPI(v *biz.OrderImportHandling) *v1.OrderImportHandling {
	if v == nil {
		return nil
	}
	out := &v1.OrderImportHandling{OrderId: v.OrderID.String(), OrderVersion: v.OrderVersion, ExchangeAgentName: v.ExchangeAgentName, Steps: []*v1.ImportHandlingStep{}}
	if v.ExchangeAgentID != nil {
		a := v.ExchangeAgentID.String()
		out.ExchangeAgentId = &a
	}
	for _, step := range v.Steps {
		item := &v1.ImportHandlingStep{ResponsibleUserName: step.ResponsibleUserName, Kind: v1.ImportHandlingStepKind(v1.ImportHandlingStepKind_value["IMPORT_HANDLING_STEP_KIND_"+step.Kind]), Undertaken: step.Undertaken, ReferenceNo: step.ReferenceNo, Notes: step.Notes, AttachmentIds: []string{}}
		if step.OccurredAt != nil {
			t := step.OccurredAt.UTC().Format(time.RFC3339Nano)
			item.OccurredAt = &t
		}
		if step.ResponsibleUserID != nil {
			u := step.ResponsibleUserID.String()
			item.ResponsibleUserId = &u
		}
		for _, orderID := range step.AttachmentIDs {
			item.AttachmentIds = append(item.AttachmentIds, orderID.String())
		}
		out.Steps = append(out.Steps, item)
	}
	return out
}
