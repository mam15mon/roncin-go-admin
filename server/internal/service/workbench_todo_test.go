package service

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/workbench/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

type workbenchTodoRepoStub struct {
	biz.WorkbenchRepo
	scope  biz.WorkbenchScope
	filter biz.WorkbenchTodoFilter
}

func (s *workbenchTodoRepoStub) ListMyUnbilledFees(_ context.Context, scope biz.WorkbenchScope, f biz.WorkbenchTodoFilter) (*biz.PagedList[*biz.WorkbenchUnbilledFee], error) {
	s.scope, s.filter = scope, f
	return &biz.PagedList[*biz.WorkbenchUnbilledFee]{Items: []*biz.WorkbenchUnbilledFee{{FeeID: uuid.New(), TotalAmount: decimal.RequireFromString("1200.50"), CreatedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), HasPendingChange: true, BlockedReasons: []string{"待补审"}}}, Total: 1, Page: f.Page, PageSize: f.PageSize}, nil
}
func (s *workbenchTodoRepoStub) ListMyOpenAbnormalCases(_ context.Context, scope biz.WorkbenchScope, f biz.WorkbenchTodoFilter) (*biz.PagedList[*biz.WorkbenchOpenAbnormalCase], error) {
	s.scope, s.filter = scope, f
	return &biz.PagedList[*biz.WorkbenchOpenAbnormalCase]{Items: []*biz.WorkbenchOpenAbnormalCase{{ID: uuid.New(), AbnormalCaseName: "查验", BusinessWriteBlockedReason: "订单已结案"}}, Total: 1, Page: f.Page, PageSize: f.PageSize}, nil
}

func TestWorkbenchTodoServiceScope(t *testing.T) {
	org, user := uuid.New(), uuid.New()
	readSE := access.OrderPermission(access.OrderBusinessSE, access.OrderRead)
	readAE := access.OrderPermission(access.OrderBusinessAE, access.OrderRead)
	feeSE := access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead)
	abnormalAE := access.OrderPermission(access.OrderBusinessAE, access.OrderAbnormalCaseRead)
	for _, scenario := range []struct {
		name                   string
		permissions            []string
		orders, fees, abnormal []biz.OrderBusinessType
	}{
		{"分品类读取", []string{readSE, readAE, feeSE, abnormalAE}, []biz.OrderBusinessType{biz.OrderBusinessSE, biz.OrderBusinessAE}, []biz.OrderBusinessType{biz.OrderBusinessSE}, []biz.OrderBusinessType{biz.OrderBusinessAE}},
		{"仅审核不能读取费用", []string{readSE, access.OrderPermission(access.OrderBusinessSE, access.OrderLock)}, []biz.OrderBusinessType{biz.OrderBusinessSE}, nil, nil},
		{"缺少订单读取", []string{feeSE, abnormalAE}, nil, nil, nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			permissions := map[string]struct{}{}
			for _, key := range scenario.permissions {
				permissions[key] = struct{}{}
			}
			p := &biz.Principal{UserID: user, Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: permissions}}}
			scope := workbenchScopeFromPrincipal(p)
			if scope.OrganizationID != org || scope.UserID != user || !reflect.DeepEqual(scope.ReadableOrderBusinessTypes, scenario.orders) || !reflect.DeepEqual(scope.ReadableFeeBusinessTypes, scenario.fees) || !reflect.DeepEqual(scope.ReadableAbnormalBusinessTypes, scenario.abnormal) {
				t.Fatalf("读取范围不符: %+v", scope)
			}
		})
	}
	stub := &workbenchTodoRepoStub{}
	service := NewWorkbenchService(biz.NewWorkbenchUsecase(stub), nil)
	ctx := commissionApplicationPrincipalContext(readSE, org)
	fees, err := service.ListMyUnbilledFees(ctx, &v1.ListMyUnbilledFeesRequest{})
	if err != nil || fees.Page != 1 || fees.PageSize != 20 || fees.Data[0].TotalAmount != "1200.50000000" || fees.Data[0].CreatedAt != "2026-10-01T00:00:00Z" || !fees.Data[0].HasPendingChange {
		t.Fatalf("费用默认分页/投影错误: %+v %v", fees, err)
	}
	abnormal, err := service.ListMyOpenAbnormalCases(ctx, &v1.ListMyOpenAbnormalCasesRequest{Page: 2, PageSize: 200})
	if err != nil || abnormal.PageSize != 200 || abnormal.Page != 2 || abnormal.Data[0].AbnormalCaseName != "查验" || abnormal.Data[0].BusinessWriteBlockedReason != "订单已结案" {
		t.Fatalf("异常投影错误: %+v %v", abnormal, err)
	}
	for _, input := range [][2]int32{{-1, 20}, {1, -1}, {1, 201}} {
		if _, err := service.ListMyUnbilledFees(ctx, &v1.ListMyUnbilledFeesRequest{Page: input[0], PageSize: input[1]}); err == nil {
			t.Fatalf("费用分页未拒绝 %v", input)
		}
		if _, err := service.ListMyOpenAbnormalCases(ctx, &v1.ListMyOpenAbnormalCasesRequest{Page: input[0], PageSize: input[1]}); err == nil {
			t.Fatalf("异常分页未拒绝 %v", input)
		}
	}
	if _, err := service.ListMyUnbilledFees(t.Context(), &v1.ListMyUnbilledFeesRequest{}); err == nil {
		t.Fatal("缺失会话未拒绝")
	}
	if _, err := service.ListMyOpenAbnormalCases(t.Context(), &v1.ListMyOpenAbnormalCasesRequest{}); err == nil {
		t.Fatal("缺失会话未拒绝")
	}
}
