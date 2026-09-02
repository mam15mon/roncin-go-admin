package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	orderv1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
)

type recordingOrderServiceHTTPServer struct {
	orderv1.UnimplementedOrderServiceServer

	calledMethod string
	getOrderID   string
}

func (s *recordingOrderServiceHTTPServer) MatchSeaMasterBillCandidate(context.Context, *orderv1.MatchSeaMasterBillCandidateRequest) (*orderv1.MatchSeaMasterBillCandidateResponse, error) {
	s.calledMethod = "MatchSeaMasterBillCandidate"
	return &orderv1.MatchSeaMasterBillCandidateResponse{Success: true}, nil
}

func (s *recordingOrderServiceHTTPServer) GetOrder(_ context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	s.calledMethod = "GetOrder"
	s.getOrderID = req.GetId()
	return &orderv1.GetOrderResponse{Success: true}, nil
}

func TestSeaMasterBillCandidateRouteDispatch(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		expectedMethod    string
		expectedOperation string
	}{
		{
			name:              "静态候选路由不被订单详情路由吞掉",
			path:              "/api/v1/orders/sea-master-bill-candidate?issuer_partner_id=" + uuid.New().String() + "&master_no=COSCO123456",
			expectedMethod:    "MatchSeaMasterBillCandidate",
			expectedOperation: orderv1.OperationOrderServiceMatchSeaMasterBillCandidate,
		},
		{
			name:              "订单详情路由保持正常分发",
			path:              "/api/v1/orders/" + uuid.New().String(),
			expectedMethod:    "GetOrder",
			expectedOperation: orderv1.OperationOrderServiceGetOrder,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &recordingOrderServiceHTTPServer{}
			var capturedOperation string
			srv := khttp.NewServer(
				khttp.Middleware(func(handler middleware.Handler) middleware.Handler {
					return func(ctx context.Context, req any) (any, error) {
						if tr, ok := transport.FromServerContext(ctx); ok {
							capturedOperation = tr.Operation()
						}
						return handler(ctx, req)
					}
				}),
			)
			orderv1.RegisterOrderServiceHTTPServer(srv, stub)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("HTTP 状态码 = %d, 响应: %s", rec.Code, rec.Body.String())
			}
			if stub.calledMethod != tc.expectedMethod {
				t.Fatalf("命中的方法 = %q, 期望 = %q", stub.calledMethod, tc.expectedMethod)
			}
			if capturedOperation != tc.expectedOperation {
				t.Fatalf("命中的 Operation = %q, 期望 = %q", capturedOperation, tc.expectedOperation)
			}
			if tc.expectedMethod == "GetOrder" {
				if _, err := uuid.Parse(stub.getOrderID); err != nil {
					t.Fatalf("GetOrder 接收到的 ID 不是有效 UUID: %q", stub.getOrderID)
				}
			}
		})
	}
}
