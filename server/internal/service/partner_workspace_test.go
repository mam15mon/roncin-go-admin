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
