package service

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type partnerWorkspaceRepo struct {
	biz.PartnerRepo
	ids     []uuid.UUID
	options biz.PartnerListOptions
}

func (r *partnerWorkspaceRepo) List(_ context.Context, ids []uuid.UUID, options biz.PartnerListOptions) (*biz.PartnerList, error) {
	r.ids = ids
	r.options = options
	return &biz.PartnerList{Page: options.Page, PageSize: options.PageSize}, nil
}

func TestPartnerListAndExportUseCurrentCompany(t *testing.T) {
	companyA, companyB := uuid.New(), uuid.New()
	for _, scope := range []biz.DataScope{biz.DataScopeAll, biz.DataScopeOrganizationTree} {
		t.Run(string(scope), func(t *testing.T) {
			p := &biz.Principal{Organization: biz.Organization{ID: companyA, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: companyA, Kind: biz.OrganizationKindCompany}, {ID: companyB, Kind: biz.OrganizationKindCompany, ParentID: &companyA}}, RoleGrants: []biz.RoleGrant{{DataScope: scope, Permissions: map[string]struct{}{access.PartnerRead: {}, access.PartnerExport: {}}}}}
			repo := &partnerWorkspaceRepo{}
			service := &PartnerService{usecase: biz.NewPartnerUsecase(repo)}
			ctx := biz.WithPrincipal(t.Context(), p)
			if _, err := service.ListPartners(ctx, &v1.ListPartnersRequest{Keyword: "测试"}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(repo.ids, []uuid.UUID{companyA}) || repo.options.Keyword != "测试" {
				t.Fatalf("列表及订单选择器必须限定当前公司: %v %+v", repo.ids, repo.options)
			}
			if _, err := service.ExportPartners(ctx, &v1.ExportPartnersRequest{}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(repo.ids, []uuid.UUID{companyA}) {
				t.Fatalf("导出必须限定当前公司: %v", repo.ids)
			}
			p.RoleGrants[0].Permissions = map[string]struct{}{access.PartnerRead: {}}
			if _, err := service.ExportPartners(ctx, &v1.ExportPartnersRequest{}); err != biz.ErrPermissionDenied {
				t.Fatalf("有读取权限也不能借用导出权限: %v", err)
			}
		})
	}
}

func TestPartnerExportPreservesCasualFilter(t *testing.T) {
	companyID := uuid.New()
	p := &biz.Principal{Organization: biz.Organization{ID: companyID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: companyID, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.PartnerExport: {}}}}}
	ctx := biz.WithPrincipal(t.Context(), p)
	for _, isCasual := range []*bool{nil, new(true), new(false)} {
		repo := &partnerWorkspaceRepo{}
		service := &PartnerService{usecase: biz.NewPartnerUsecase(repo)}
		if _, err := service.ExportPartners(ctx, &v1.ExportPartnersRequest{Keyword: "测试", Role: v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER, IsCasual: isCasual, Enabled: new(true), Blacklisted: new(true)}); err != nil {
			t.Fatal(err)
		}
		if (repo.options.IsCasual == nil) != (isCasual == nil) || isCasual != nil && *repo.options.IsCasual != *isCasual {
			t.Fatalf("导出应保留未筛选、散客与正式客户三种语义: %+v", repo.options)
		}
		if !slices.Equal(repo.ids, []uuid.UUID{companyID}) || repo.options.Keyword != "测试" || repo.options.Role != biz.PartnerRoleCustomer || repo.options.Enabled == nil || !*repo.options.Enabled || repo.options.Blacklisted == nil || !*repo.options.Blacklisted || repo.options.PageSize != biz.MaxListPageSize {
			t.Fatalf("散客筛选不能改变公司范围与其他筛选: %+v", repo.options)
		}
	}
}
