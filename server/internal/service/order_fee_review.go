package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
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

type feeReviewActionRequest interface {
	GetOrderId() string
	GetExpectedReviewVersion() uint64
	GetExpectedContentVersion() uint64
	GetSubmissionId() string
	GetPreviewToken() string
	GetComment() string
	GetIdempotencyKey() string
	GetCompletenessConfirmed() bool
}

func parseFeeReviewAction(r feeReviewActionRequest) (biz.FeeReviewAction, error) {
	id, err := uuid.Parse(r.GetOrderId())
	if err != nil {
		return biz.FeeReviewAction{}, biz.ErrFeeReviewInvalid
	}
	out := biz.FeeReviewAction{OrderID: id, ExpectedReviewVersion: r.GetExpectedReviewVersion(), ExpectedContentVersion: r.GetExpectedContentVersion(), PreviewToken: r.GetPreviewToken(), Comment: r.GetComment(), IdempotencyKey: r.GetIdempotencyKey(), CompletenessConfirmed: r.GetCompletenessConfirmed()}
	if r.GetSubmissionId() != "" {
		submission, err := uuid.Parse(r.GetSubmissionId())
		if err != nil {
			return out, biz.ErrFeeReviewInvalid
		}
		out.SubmissionID = &submission
	}
	return out, nil
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
func (s *OrderFeeService) feeReviewAct(ctx context.Context, r feeReviewActionRequest, action string) (*v1.OrderFeeReviewData, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	input, err := parseFeeReviewAction(r)
	if err != nil {
		return nil, err
	}
	data, err := s.review.Act(ctx, p, action, input)
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
func (s *OrderFeeService) PreviewOrderFeeReview(ctx context.Context, r *v1.PreviewOrderFeeReviewRequest) (*v1.PreviewOrderFeeReviewResponse, error) {
	data, err := s.getFeeReview(ctx, r.GetOrderId())
	if err != nil {
		return nil, err
	}
	return &v1.PreviewOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *OrderFeeService) SubmitOrderFeeReview(ctx context.Context, r *v1.SubmitOrderFeeReviewRequest) (*v1.SubmitOrderFeeReviewResponse, error) {
	data, err := s.feeReviewAct(ctx, r, "SUBMIT")
	if err != nil {
		return nil, err
	}
	return &v1.SubmitOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *OrderFeeService) WithdrawOrderFeeReview(ctx context.Context, r *v1.WithdrawOrderFeeReviewRequest) (*v1.WithdrawOrderFeeReviewResponse, error) {
	data, err := s.feeReviewAct(ctx, r, "WITHDRAW")
	if err != nil {
		return nil, err
	}
	return &v1.WithdrawOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *OrderFeeService) ApproveOrderFeeReview(ctx context.Context, r *v1.ApproveOrderFeeReviewRequest) (*v1.ApproveOrderFeeReviewResponse, error) {
	data, err := s.feeReviewAct(ctx, r, "APPROVE")
	if err != nil {
		return nil, err
	}
	return &v1.ApproveOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *OrderFeeService) RejectOrderFeeReview(ctx context.Context, r *v1.RejectOrderFeeReviewRequest) (*v1.RejectOrderFeeReviewResponse, error) {
	data, err := s.feeReviewAct(ctx, r, "REJECT")
	if err != nil {
		return nil, err
	}
	return &v1.RejectOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *OrderFeeService) RevokeOrderFeeReview(ctx context.Context, r *v1.RevokeOrderFeeReviewRequest) (*v1.RevokeOrderFeeReviewResponse, error) {
	data, err := s.feeReviewAct(ctx, r, "REVOKE")
	if err != nil {
		return nil, err
	}
	return &v1.RevokeOrderFeeReviewResponse{Success: true, Code: 200, Message: "成功", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}
func feeReviewResultsToAPI(results []biz.FeeReviewResult) []*v1.OrderFeeReviewResult {
	out := make([]*v1.OrderFeeReviewResult, 0, len(results))
	for _, item := range results {
		result := &v1.OrderFeeReviewResult{OrderId: item.OrderID.String(), Success: item.Err == nil, Data: feeReviewToAPI(item.Data)}
		if item.Err != nil {
			e := errors.FromError(item.Err)
			result.Reason = e.Reason
			result.Message = e.Message
		}
		out = append(out, result)
	}
	return out
}
func (s *OrderFeeService) PreviewOrderFeeReviews(ctx context.Context, r *v1.PreviewOrderFeeReviewsRequest) (*v1.PreviewOrderFeeReviewsResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(r.GetOrderIds()))
	for _, text := range r.GetOrderIds() {
		id, err := uuid.Parse(text)
		if err != nil {
			return nil, biz.ErrFeeReviewInvalid
		}
		ids = append(ids, id)
	}
	results, err := s.review.PreviewBatch(ctx, p, ids)
	if err != nil {
		return nil, err
	}
	return &v1.PreviewOrderFeeReviewsResponse{Data: feeReviewResultsToAPI(results), TraceId: requestmeta.TraceID(ctx)}, nil
}
func (s *OrderFeeService) ApproveOrderFeeReviews(ctx context.Context, r *v1.ApproveOrderFeeReviewsRequest) (*v1.ApproveOrderFeeReviewsResponse, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	inputs := make([]biz.FeeReviewAction, 0, len(r.GetItems()))
	for _, item := range r.GetItems() {
		input, err := parseFeeReviewAction(item)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	results, err := s.review.ApproveBatch(ctx, p, inputs)
	if err != nil {
		return nil, err
	}
	return &v1.ApproveOrderFeeReviewsResponse{Data: feeReviewResultsToAPI(results), TraceId: requestmeta.TraceID(ctx)}, nil
}
