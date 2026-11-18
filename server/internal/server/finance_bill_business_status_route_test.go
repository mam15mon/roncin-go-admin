package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	khttp "github.com/go-kratos/kratos/v3/transport/http"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
)

type billBusinessStatusRouteStub struct {
	financev1.UnimplementedSettlementServiceServer
	request *financev1.ListBillsRequest
}

func (s *billBusinessStatusRouteStub) ListBills(_ context.Context, request *financev1.ListBillsRequest) (*financev1.ListBillsResponse, error) {
	s.request = request
	return &financev1.ListBillsResponse{Success: true, Data: []*financev1.FinanceBill{{BusinessStatus: financev1.FinanceBillBusinessStatus_FINANCE_BILL_BUSINESS_STATUS_UNINVOICED}}}, nil
}

func TestBillBusinessStatusMultiQueryHTTPRoute(t *testing.T) {
	for _, query := range []string{
		"businessStatuses=1&businessStatuses=2",
		"business_statuses=1&business_statuses=2",
		"businessStatuses%5B%5D=1&businessStatuses%5B%5D=2",
		"businessStatuses=FINANCE_BILL_BUSINESS_STATUS_UNINVOICED&businessStatuses=FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED",
	} {
		t.Run(query, func(t *testing.T) {
			stub := &billBusinessStatusRouteStub{}
			srv := khttp.NewServer()
			financev1.RegisterSettlementServiceHTTPServer(srv, stub)
			response := httptest.NewRecorder()
			srv.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/finance/bills?page=1&pageSize=20&"+query, nil))
			want := []financev1.FinanceBillBusinessStatus{financev1.FinanceBillBusinessStatus_FINANCE_BILL_BUSINESS_STATUS_UNINVOICED, financev1.FinanceBillBusinessStatus_FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED}
			if response.Code != http.StatusOK || stub.request == nil || !reflect.DeepEqual(stub.request.BusinessStatuses, want) {
				t.Fatalf("多选查询未正确绑定: status=%d request=%v body=%s", response.Code, stub.request, response.Body.String())
			}
		})
	}
}
