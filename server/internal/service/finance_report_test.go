package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type reportServiceRepoStub struct {
	biz.FinanceReportRepo
	calls     int
	org       uuid.UUID
	f         biz.FinanceReportFilter
	export    bool
	personnel bool
}

func (r *reportServiceRepoStub) Profit(_ context.Context, org uuid.UUID, f biz.FinanceReportFilter, export bool) (*biz.ProfitReportResult, error) {
	r.calls++
	r.org, r.f, r.export = org, f, export
	return &biz.ProfitReportResult{GeneratedAt: time.Now()}, nil
}
func (r *reportServiceRepoStub) Cashflow(_ context.Context, org uuid.UUID, f biz.FinanceReportFilter, export bool) (*biz.CashflowReportResult, error) {
	r.calls++
	r.org, r.f, r.export = org, f, export
	return &biz.CashflowReportResult{GeneratedAt: time.Now()}, nil
}
func (r *reportServiceRepoStub) Aging(_ context.Context, org uuid.UUID, f biz.FinanceReportFilter, export bool, _ time.Time) (*biz.AgingReportResult, error) {
	r.calls++
	r.org, r.f, r.export = org, f, export
	return &biz.AgingReportResult{AsOfDate: "2026-11-14", GeneratedAt: time.Now()}, nil
}
func (r *reportServiceRepoStub) Options(_ context.Context, org uuid.UUID, _ biz.SelectorListOptions, personnel bool) (*biz.PagedList[*biz.FinanceReportOption], error) {
	r.calls++
	r.org, r.personnel = org, personnel
	return &biz.PagedList[*biz.FinanceReportOption]{}, nil
}
func reportTestPrincipal(org uuid.UUID, kind biz.OrganizationKind, permission string) *biz.Principal {
	return &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: kind}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: kind}}, RoleGrants: []biz.RoleGrant{{RoleCode: "report", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{permission: {}}}}}
}
func TestFinanceReportServiceCurrentCompanyAndDedicatedPermission(t *testing.T) {
	org := uuid.New()
	for _, test := range []struct {
		name       string
		kind       biz.OrganizationKind
		permission string
		target     uuid.UUID
		want       bool
	}{{"独立报表权", biz.OrganizationKindCompany, access.FinanceReportRead, org, true}, {"只有费用权", biz.OrganizationKindCompany, access.FinanceFeeRead, org, false}, {"越公司", biz.OrganizationKindCompany, access.FinanceReportRead, uuid.New(), false}, {"系统管理", biz.OrganizationKindSystem, access.FinanceReportRead, org, false}} {
		t.Run(test.name, func(t *testing.T) {
			repo := &reportServiceRepoStub{}
			s := NewFinanceReportService(biz.NewFinanceReportUsecase(repo))
			ctx := biz.WithPrincipal(context.Background(), reportTestPrincipal(org, test.kind, test.permission))
			response, err := s.ListProfitReport(ctx, &v1.ListProfitReportRequest{OrganizationId: test.target.String(), Page: 1, PageSize: 200})
			if test.want {
				if err != nil || !response.GetSuccess() || repo.org != org || repo.f.PageSize != 200 {
					t.Fatalf("独立报表权限应允许当前公司读: %#v %v", repo, err)
				}
			} else if !errors.Is(err, biz.ErrPermissionDenied) || repo.calls != 0 {
				t.Fatalf("越权请求不应进入仓储: %#v %v", repo, err)
			}
		})
	}
}
func TestFinanceReportServiceListsExportsAndCandidatesShareScope(t *testing.T) {
	org := uuid.New()
	repo := &reportServiceRepoStub{}
	s := NewFinanceReportService(biz.NewFinanceReportUsecase(repo))
	ctx := biz.WithPrincipal(context.Background(), reportTestPrincipal(org, biz.OrganizationKindCompany, access.FinanceReportRead))
	id := uuid.New().String()
	from, to := "2026-01-01", "2026-01-31"
	if _, err := s.ExportProfitReport(ctx, &v1.ExportProfitReportRequest{OrganizationId: org.String(), DateFrom: &from, DateTo: &to, CustomerId: &id}); err != nil || !repo.export || repo.f.DateFrom != from || repo.f.CustomerID == nil {
		t.Fatalf("导出应传同源筛选且无分页: %#v %v", repo, err)
	}
	if _, err := s.ListCashflowReport(ctx, &v1.ListCashflowReportRequest{OrganizationId: org.String()}); err != nil || repo.export || repo.f.Page != 1 {
		t.Fatalf("流水默认分页失败: %#v %v", repo, err)
	}
	if _, err := s.ExportCashflowReport(ctx, &v1.ExportCashflowReportRequest{OrganizationId: org.String()}); err != nil || !repo.export {
		t.Fatal(err)
	}
	if _, err := s.ListAgingReport(ctx, &v1.ListAgingReportRequest{OrganizationId: org.String()}); err != nil || repo.export {
		t.Fatal(err)
	}
	if _, err := s.ExportAgingReport(ctx, &v1.ExportAgingReportRequest{OrganizationId: org.String()}); err != nil || !repo.export {
		t.Fatal(err)
	}
	for _, personnel := range []bool{false, true} {
		var err error
		if personnel {
			_, err = s.ListFinanceReportPersonnelOptions(ctx, &v1.ListFinanceReportPersonnelOptionsRequest{OrganizationId: org.String()})
		} else {
			_, err = s.ListFinanceReportPartyOptions(ctx, &v1.ListFinanceReportPartyOptionsRequest{OrganizationId: org.String()})
		}
		if err != nil || repo.personnel != personnel || repo.org != org {
			t.Fatalf("候选必须只需报表权且同公司: %#v %v", repo, err)
		}
	}
	before := repo.calls
	if _, err := s.ListProfitReport(ctx, &v1.ListProfitReportRequest{OrganizationId: org.String(), Page: 1, PageSize: 201}); err == nil || repo.calls != before {
		t.Fatal("201分页应在仓储前拒绝")
	}
}
