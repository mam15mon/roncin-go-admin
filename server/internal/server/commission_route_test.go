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
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
)

type recordingSettlementServiceHTTPServer struct {
	financev1.UnimplementedSettlementServiceServer

	calledMethod     string
	getCommissionID  string
	exportCalled     bool
	employeesCalled  bool
	candidatesCalled bool
	getCalled        bool
}

func (s *recordingSettlementServiceHTTPServer) ExportCommissions(ctx context.Context, req *financev1.ExportCommissionsRequest) (*financev1.ExportCommissionsResponse, error) {
	s.calledMethod = "ExportCommissions"
	s.exportCalled = true
	return &financev1.ExportCommissionsResponse{Success: true}, nil
}

func (s *recordingSettlementServiceHTTPServer) ListCommissionEmployees(ctx context.Context, req *financev1.ListCommissionEmployeesRequest) (*financev1.ListCommissionEmployeesResponse, error) {
	s.calledMethod = "ListCommissionEmployees"
	s.employeesCalled = true
	return &financev1.ListCommissionEmployeesResponse{Success: true}, nil
}

func (s *recordingSettlementServiceHTTPServer) ListCommissionCandidates(ctx context.Context, req *financev1.ListCommissionCandidatesRequest) (*financev1.ListCommissionCandidatesResponse, error) {
	s.calledMethod = "ListCommissionCandidates"
	s.candidatesCalled = true
	return &financev1.ListCommissionCandidatesResponse{Success: true}, nil
}

func (s *recordingSettlementServiceHTTPServer) GetCommission(ctx context.Context, req *financev1.GetCommissionRequest) (*financev1.GetCommissionResponse, error) {
	s.calledMethod = "GetCommission"
	s.getCalled = true
	s.getCommissionID = req.GetId()
	return &financev1.GetCommissionResponse{Success: true, Data: &financev1.FinanceCommission{Id: req.GetId()}}, nil
}

func TestCommissionRouteDispatch(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		expectedMethod    string
		expectedOperation string
		validate          func(t *testing.T, s *recordingSettlementServiceHTTPServer)
	}{
		{
			name:              "静态路由 export 正确分发到 ExportCommissions",
			path:              "/api/v1/finance/commissions/export",
			expectedMethod:    "ExportCommissions",
			expectedOperation: financev1.OperationSettlementServiceExportCommissions,
			validate: func(t *testing.T, s *recordingSettlementServiceHTTPServer) {
				if !s.exportCalled {
					t.Fatal("未调用 ExportCommissions")
				}
				if s.getCalled {
					t.Fatalf("静态 export 路由被错误分发至 GetCommission, id=%q", s.getCommissionID)
				}
			},
		},
		{
			name:              "静态路由 employees 正确分发到 ListCommissionEmployees 而非 GetCommission",
			path:              "/api/v1/finance/commissions/employees",
			expectedMethod:    "ListCommissionEmployees",
			expectedOperation: financev1.OperationSettlementServiceListCommissionEmployees,
			validate: func(t *testing.T, s *recordingSettlementServiceHTTPServer) {
				if !s.employeesCalled {
					t.Fatal("未调用 ListCommissionEmployees")
				}
				if s.getCalled {
					t.Fatalf("静态 employees 路由被错误分发至 GetCommission, id=%q", s.getCommissionID)
				}
			},
		},
		{
			name:              "静态路由 candidates 正确分发到 ListCommissionCandidates 而非 GetCommission",
			path:              "/api/v1/finance/commissions/candidates?verification_id=" + uuid.New().String() + "&rule_id=" + uuid.New().String(),
			expectedMethod:    "ListCommissionCandidates",
			expectedOperation: financev1.OperationSettlementServiceListCommissionCandidates,
			validate: func(t *testing.T, s *recordingSettlementServiceHTTPServer) {
				if !s.candidatesCalled {
					t.Fatal("未调用 ListCommissionCandidates")
				}
				if s.getCalled {
					t.Fatalf("静态 candidates 路由被错误分发至 GetCommission, id=%q", s.getCommissionID)
				}
			},
		},
		{
			name: "正常 UUID 详情路由正确分发到 GetCommission",
			path: func() string {
				id := uuid.New().String()
				return "/api/v1/finance/commissions/" + id
			}(),
			expectedMethod:    "GetCommission",
			expectedOperation: financev1.OperationSettlementServiceGetCommission,
			validate: func(t *testing.T, s *recordingSettlementServiceHTTPServer) {
				if !s.getCalled {
					t.Fatal("未调用 GetCommission")
				}
				if _, err := uuid.Parse(s.getCommissionID); err != nil {
					t.Fatalf("GetCommission 接收到的 ID 不是有效 UUID: %q", s.getCommissionID)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &recordingSettlementServiceHTTPServer{}
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
			financev1.RegisterSettlementServiceHTTPServer(srv, stub)

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
			if tc.validate != nil {
				tc.validate(t, stub)
			}
		})
	}
}
