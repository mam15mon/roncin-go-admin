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

type invoiceCreationRouteStub struct {
	financev1.UnimplementedSettlementServiceServer
	method, partyID, invoiceID string
	calls                      int
}

func (s *invoiceCreationRouteStub) ListInvoiceProfilesForParty(_ context.Context, request *financev1.ListInvoiceProfilesForPartyRequest) (*financev1.ListInvoiceProfilesForPartyResponse, error) {
	s.method, s.partyID = "ListInvoiceProfilesForParty", request.GetSettlementPartyId()
	s.calls++
	return &financev1.ListInvoiceProfilesForPartyResponse{Success: true, Data: &financev1.FinanceInvoiceProfilesForBill{SettlementPartyId: s.partyID}}, nil
}

func (s *invoiceCreationRouteStub) GetInvoice(_ context.Context, request *financev1.GetInvoiceRequest) (*financev1.GetInvoiceResponse, error) {
	s.method, s.invoiceID = "GetInvoice", request.GetId()
	s.calls++
	return &financev1.GetInvoiceResponse{Success: true, Data: &financev1.FinanceInvoice{Id: s.invoiceID}}, nil
}

// 注册完整生成的HTTP服务并实际发送请求，防止创建候选被先注册的invoices/{id}吞掉。
func TestInvoiceCreationPartyProfilesHTTPRoute(t *testing.T) {
	partyID, invoiceID := uuid.NewString(), uuid.NewString()
	for _, tc := range []struct {
		name, path, method, operation, party, invoice string
	}{
		{"候选camelCase绑定", "/api/v1/finance/invoice-creation/party-profiles?settlementPartyId=" + partyID, "ListInvoiceProfilesForParty", financev1.OperationSettlementServiceListInvoiceProfilesForParty, partyID, ""},
		{"候选snake_case绑定", "/api/v1/finance/invoice-creation/party-profiles?settlement_party_id=" + partyID, "ListInvoiceProfilesForParty", financev1.OperationSettlementServiceListInvoiceProfilesForParty, partyID, ""},
		{"正常详情仍正确分发", "/api/v1/finance/invoices/" + invoiceID, "GetInvoice", financev1.OperationSettlementServiceGetInvoice, "", invoiceID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &invoiceCreationRouteStub{}
			var operation string
			srv := khttp.NewServer(khttp.Middleware(func(next middleware.Handler) middleware.Handler {
				return func(ctx context.Context, request any) (any, error) {
					if tr, ok := transport.FromServerContext(ctx); ok {
						operation = tr.Operation()
					}
					return next(ctx, request)
				}
			}, RequiredFieldsValidator()))
			financev1.RegisterSettlementServiceHTTPServer(srv, stub)
			candidateRegistered := false
			if err := srv.WalkRoute(func(route khttp.RouteInfo) error {
				if route.Method == http.MethodGet && route.Path == "/api/v1/finance/invoice-creation/party-profiles" {
					candidateRegistered = true
				}
				if route.Path == "/api/v1/finance/invoices/party-profiles" {
					t.Error("不得保留被动态详情遮蔽的旧候选路径")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if !candidateRegistered {
				t.Fatal("生成服务必须注册独立创建候选路径")
			}
			response := httptest.NewRecorder()
			srv.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != http.StatusOK || stub.calls != 1 || stub.method != tc.method || operation != tc.operation || stub.partyID != tc.party || stub.invoiceID != tc.invoice {
				t.Fatalf("HTTP候选/详情分发或参数绑定错误：status=%d method=%s operation=%s calls=%d party=%s invoice=%s response=%s", response.Code, stub.method, operation, stub.calls, stub.partyID, stub.invoiceID, response.Body.String())
			}
		})
	}
}
