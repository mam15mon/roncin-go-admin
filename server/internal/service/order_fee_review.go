package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

func feeReviewSummaryToAPI(s *biz.FeeReviewSummary) *v1.OrderFeeReviewSummary {
	if s == nil {
		return nil
	}
	out := &v1.OrderFeeReviewSummary{BaseCurrency: s.BaseCurrency, Receivable: s.Receivable.StringFixed(8), Payable: s.Payable.StringFixed(8), Profit: s.Profit.StringFixed(8), ReceivableCount: int32(s.ReceivableCount), PayableCount: int32(s.PayableCount), SubmittedByName: s.SubmittedByName}
	if s.ProfitRate != nil {
		text := s.ProfitRate.String()
		out.ProfitRate = &text
	}
	if s.SubmittedAt != nil {
		out.SubmittedAt = s.SubmittedAt.Format(time.RFC3339)
	}
	return out
}
func feeReviewToAPI(r *biz.OrderFeeReview) *v1.OrderFeeReviewData {
	if r == nil {
		return nil
	}
	out := &v1.OrderFeeReviewData{OrderId: r.OrderID.String(), OrderNo: r.OrderNo, Status: r.Status, ContentVersion: r.ContentVersion, ReviewVersion: r.ReviewVersion, PreviewToken: r.PreviewToken, Summary: feeReviewSummaryToAPI(r.Summary), CanSubmit: r.CanSubmit, CanWithdraw: r.CanWithdraw, CanApprove: r.CanApprove, CanRevoke: r.CanRevoke, CanEdit: r.CanEdit, CanBill: r.CanBill, HasPendingSupplement: r.HasPendingSupplement, HasPendingCorrection: r.HasPendingCorrection, HasPendingChange: r.HasPendingChange, BlockedReasons: r.BlockedReasons, NotificationFailedCount: int32(r.NotificationFailedCount), NotificationUnboundCount: int32(r.NotificationUnboundCount)}
	if r.SubmissionID != nil {
		out.SubmissionId = r.SubmissionID.String()
	}
	if r.ApprovedEventID != nil {
		out.ApprovedEventId = r.ApprovedEventID.String()
	}
	for _, fee := range r.Fees {
		out.Fees = append(out.Fees, orderFeeToAPI(fee))
	}
	for _, e := range r.History {
		event := &v1.OrderFeeReviewEventData{Id: e.ID.String(), Action: e.Action, ActorId: e.ActorID.String(), ActorName: e.ActorName, Comment: e.Comment, CreatedAt: e.CreatedAt.Format(time.RFC3339), ContentVersion: e.ContentVersion, Summary: feeReviewSummaryToAPI(&e.Summary)}
		if e.SubmissionID != nil {
			event.SubmissionId = e.SubmissionID.String()
		}
		out.History = append(out.History, event)
	}
	return out
}

func (s *OrderFeeService) getFeeReview(ctx context.Context, orderID string) (*v1.OrderFeeReviewData, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, biz.ErrFeeReviewInvalid
	}
	data, err := s.review.Get(ctx, p, id)
	if err != nil {
		return nil, err
	}
	return feeReviewToAPI(data), nil
}
func (s *OrderFeeService) GetOrderFeeReview(ctx context.Context, r *v1.GetOrderFeeReviewRequest) (*v1.GetOrderFeeReviewResponse, error) {
	data, err := s.getFeeReview(ctx, r.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &v1.GetOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
