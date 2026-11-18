package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

type businessStatusListBillRepoStub struct {
	biz.FinanceBillRepo
	filter biz.FinanceBillFilter
	calls  int
}

func (s *businessStatusListBillRepoStub) List(_ context.Context, _ []uuid.UUID, filter biz.FinanceBillFilter) (*biz.FinanceBillListResult, error) {
	s.filter = filter
	s.calls++
	return &biz.FinanceBillListResult{}, nil
}

func TestListBillsBusinessStatusesValidation(t *testing.T) {
	org := uuid.New()
	principal := &biz.Principal{Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "bill-reader", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceBillRead: {}}}}}
	repo := &businessStatusListBillRepoStub{}
	svc := &SettlementService{billUsecase: biz.NewFinanceBillUsecase(repo, nil, nil, nil)}
	ctx := biz.WithPrincipal(context.Background(), principal)
	request := &v1.ListBillsRequest{Page: 1, PageSize: 20, BusinessStatuses: []v1.FinanceBillBusinessStatus{v1.FinanceBillBusinessStatus_FINANCE_BILL_BUSINESS_STATUS_UNINVOICED, v1.FinanceBillBusinessStatus_FINANCE_BILL_BUSINESS_STATUS_PARTIALLY_INVOICED}}
	if _, err := svc.ListBills(ctx, request); err != nil {
		t.Fatal(err)
	}
	if len(repo.filter.BusinessStatuses) != 2 || repo.filter.BusinessStatuses[0] != biz.FinanceBillBusinessUninvoiced || repo.filter.BusinessStatuses[1] != biz.FinanceBillBusinessPartiallyInvoiced {
		t.Fatalf("多选条件转换不完整:%+v", repo.filter)
	}
	for _, invalid := range []v1.FinanceBillBusinessStatus{0, 999} {
		request.BusinessStatuses = []v1.FinanceBillBusinessStatus{invalid}
		if _, err := svc.ListBills(ctx, request); err != biz.ErrFinanceBillInvalidArgument {
			t.Fatalf("非法状态%d没有被拒绝:%v", invalid, err)
		}
	}
	if repo.calls != 1 {
		t.Fatalf("非法状态进入了仓储:%d", repo.calls)
	}
}

func TestBillBusinessStatusProjectionWire(t *testing.T) {
	for _, tc := range []struct {
		status biz.FinanceBillBusinessStatus
		want   float64
	}{
		{biz.FinanceBillBusinessUninvoiced, 1},
		{biz.FinanceBillBusinessPartiallyInvoiced, 2},
		{biz.FinanceBillBusinessUnverified, 3},
		{biz.FinanceBillBusinessPartiallyVerified, 4},
		{biz.FinanceBillBusinessCompleted, 5},
		{"", 0},
	} {
		wire, err := (protojson.MarshalOptions{UseEnumNumbers: true}).Marshal(financeBillToAPI(&biz.FinanceBill{ID: uuid.New(), BusinessStatus: tc.status}))
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(wire, &body); err != nil {
			t.Fatal(err)
		}
		if tc.want == 0 {
			if _, exists := body["businessStatus"]; exists {
				t.Fatalf("未投影状态不应伪造默认值:%s", wire)
			}
		} else if body["businessStatus"] != tc.want {
			t.Fatalf("业务状态wire=%v，期望%v", body["businessStatus"], tc.want)
		}
	}
}
