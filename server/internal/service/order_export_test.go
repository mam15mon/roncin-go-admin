package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type orderExportServiceRepo struct {
	biz.OrderRepo
	called  bool
	options biz.OrderListOptions
	scopes  []biz.OrderOrganizationScope
	rows    []*biz.Order
}

func (r *orderExportServiceRepo) Export(_ context.Context, scopes []biz.OrderOrganizationScope, options biz.OrderListOptions) (*biz.OrderExport, error) {
	r.called, r.scopes, r.options = true, scopes, options
	return &biz.OrderExport{Items: r.rows, GeneratedAt: time.Date(2026, 11, 16, 1, 2, 3, 0, time.UTC)}, nil
}
func TestOrderExportServiceIdentityFiltersAndProjection(t *testing.T) {
	for _, kind := range []v1.BusinessType{v1.BusinessType_BUSINESS_TYPE_SE, v1.BusinessType_BUSINESS_TYPE_SI, v1.BusinessType_BUSINESS_TYPE_AE, v1.BusinessType_BUSINESS_TYPE_AI} {
		t.Run(kind.String(), func(t *testing.T) {
			org := uuid.New()
			business := orderBusinessTypeFromAPI(kind)
			permission := access.OrderPermission(access.OrderBusinessType(business), access.OrderRead)
			p := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{permission: {}}}}}
			count := 0
			repo := &orderExportServiceRepo{rows: []*biz.Order{{ID: uuid.New(), OrganizationID: org, BusinessType: business, OrderNo: "中文订单", ContainerNos: []string{"不能泄露箱号"}, AttachmentCount: &count}}}
			uc := biz.NewOrderUsecase(repo, nil, nil, nil, nil)
			svc := NewOrderService(uc)
			locked := false
			request := &v1.ExportOrdersRequest{BusinessType: kind, Keyword: "  中文  ", SortBy: "eta", SortOrder: "ascend", IsLocked: &locked, CreatedAtFrom: "2026-11-01", EtaTo: "2026-11-30"}
			response, err := svc.ExportOrders(biz.WithPrincipal(t.Context(), p), request)
			if err != nil || response.Total != 1 || response.GeneratedAt != "2026-11-16T01:02:03Z" || len(response.Data) != 1 {
				t.Fatalf("完整响应错误: %+v %v", response, err)
			}
			if repo.options.Page != 0 || repo.options.PageSize != 0 || repo.options.Keyword != "中文" || repo.options.IsLocked == nil || *repo.options.IsLocked || repo.options.CreatedAtRange.From == nil || repo.options.ETARange.ToExclusive == nil || repo.options.SortBy != "eta" || len(repo.scopes) != 1 || repo.scopes[0].BusinessType != business || len(repo.scopes[0].OrganizationIDs) != 1 || repo.scopes[0].OrganizationIDs[0] != org {
				t.Fatalf("筛选/排序/当前公司范围错误: %+v %+v", repo.options, repo.scopes)
			}
			if response.Data[0].AttachmentCount != nil || len(response.Data[0].ContainerNos) != 0 {
				t.Fatal("缺少子资源权限却泄露导出字段")
			}
			p.RoleGrants[0].Permissions = map[string]struct{}{access.OrderPermission(access.OrderBusinessSE, access.OrderRead): {}}
			if business != biz.OrderBusinessSE {
				repo.called = false
				if _, err = svc.ExportOrders(biz.WithPrincipal(t.Context(), p), request); err == nil || repo.called {
					t.Fatal("导出借用了另一类型的读取权限")
				}
			}
			p.Organization.Kind = biz.OrganizationKindSystem
			if _, err = svc.ExportOrders(biz.WithPrincipal(t.Context(), p), request); err == nil {
				t.Fatal("系统工作台不应导出经营数据")
			}
		})
	}
}
func TestOrderExportServiceRejectsInvalidInput(t *testing.T) {
	org := uuid.New()
	p := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.OrderPermission(access.OrderBusinessSE, access.OrderRead): {}}}}}
	repo := &orderExportServiceRepo{}
	svc := NewOrderService(biz.NewOrderUsecase(repo, nil, nil, nil, nil))
	for _, req := range []*v1.ExportOrdersRequest{{}, {BusinessType: v1.BusinessType_BUSINESS_TYPE_LAND}, {BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, SortBy: "amount"}, {BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, SortOrder: "wrong"}, {BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, CustomerId: "invalid"}, {BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, NumberKeyword: "without type"}, {BusinessType: v1.BusinessType_BUSINESS_TYPE_SE, EtdFrom: "not-date"}} {
		repo.called = false
		if _, err := svc.ExportOrders(biz.WithPrincipal(t.Context(), p), req); err == nil || repo.called {
			t.Fatalf("非法请求未在读取前拒绝: %+v", req)
		}
	}
	if _, err := svc.ExportOrders(t.Context(), &v1.ExportOrdersRequest{BusinessType: v1.BusinessType_BUSINESS_TYPE_SE}); err == nil {
		t.Fatal("缺少会话不应通过")
	}
}
