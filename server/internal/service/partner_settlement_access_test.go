package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type partnerSettlementAccessRepo struct {
	biz.PartnerRepo
	item       *biz.Partner
	ruleAccess biz.PartnerSettlementRuleWriteAccess
}

func (r *partnerSettlementAccessRepo) Get(context.Context, uuid.UUID, uuid.UUID) (*biz.Partner, error) {
	return r.item, nil
}
func (r *partnerSettlementAccessRepo) List(_ context.Context, _ []uuid.UUID, options biz.PartnerListOptions) (*biz.PartnerList, error) {
	return &biz.PartnerList{Items: []*biz.Partner{r.item}, Total: 1, Page: options.Page, PageSize: options.PageSize}, nil
}
func (r *partnerSettlementAccessRepo) Create(_ context.Context, _ uuid.UUID, _ *biz.Partner, ruleAccess biz.PartnerSettlementRuleWriteAccess, _ *biz.AuditEvent) (*biz.Partner, error) {
	r.ruleAccess = ruleAccess
	return r.item, nil
}
func (r *partnerSettlementAccessRepo) Update(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *biz.Partner, ruleAccess biz.PartnerSettlementRuleWriteAccess, _ *biz.AuditEvent) (*biz.PartnerUpdateResult, error) {
	r.ruleAccess = ruleAccess
	return &biz.PartnerUpdateResult{Partner: r.item}, nil
}
func (r *partnerSettlementAccessRepo) UpdateRoles(context.Context, uuid.UUID, uuid.UUID, []*biz.PartnerRole, *biz.AuditEvent) (*biz.PartnerUpdateResult, error) {
	return &biz.PartnerUpdateResult{Partner: r.item}, nil
}
func (r *partnerSettlementAccessRepo) SetPartnerRoleBlacklist(context.Context, uuid.UUID, uuid.UUID, biz.PartnerBlacklistUpdate, *biz.AuditEvent) (*biz.PartnerBlacklistResult, error) {
	return &biz.PartnerBlacklistResult{Partner: r.item}, nil
}
func (r *partnerSettlementAccessRepo) Import(_ context.Context, _ uuid.UUID, _ biz.PartnerImportMode, _ []*biz.Partner, ruleAccess biz.PartnerSettlementRuleWriteAccess, _ *biz.AuditEvent) (*biz.PartnerImportResult, error) {
	r.ruleAccess = ruleAccess
	return &biz.PartnerImportResult{CreatedCount: 1}, nil
}

func partnerSettlementAccessPrincipal(companyID uuid.UUID, permissions ...string) *biz.Principal {
	grants := map[string]struct{}{}
	for _, permission := range permissions {
		grants[permission] = struct{}{}
	}
	return &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: companyID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: companyID, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{DataScope: biz.DataScopeOrganization, Permissions: grants}}}
}

func TestPartnerAggregateSettlementReadAccess(t *testing.T) {
	companyID := uuid.New()
	roles := []*v1.PartnerRoleInput{{Type: v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER, Enabled: true}}
	for _, readable := range []bool{false, true} {
		for _, sameCompany := range []bool{false, true} {
			permissions := []string{access.PartnerRead, access.PartnerExport}
			if readable {
				permissions = append(permissions, access.PartnerSettlementRuleRead)
			}
			principal := partnerSettlementAccessPrincipal(companyID, permissions...)
			resultCompanyID := companyID
			if !sameCompany {
				resultCompanyID = uuid.New()
			}
			item := &biz.Partner{ID: uuid.New(), OrganizationID: resultCompanyID, LegalName: "测试客户", Roles: []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true, SettlementRule: &biz.PartnerSettlementRule{ID: uuid.New(), PaymentTermsDays: new(30)}}}}
			repo := &partnerSettlementAccessRepo{item: item}
			service := &PartnerService{usecase: biz.NewPartnerUsecase(repo)}
			ctx := biz.WithPrincipal(t.Context(), principal)
			calls := map[string]func() ([]*v1.PartnerRole, error){
				"Get": func() ([]*v1.PartnerRole, error) {
					r, e := service.GetPartner(ctx, &v1.GetPartnerRequest{Id: item.ID.String()})
					if e != nil {
						return nil, e
					}
					return r.Data.Roles, nil
				},
				"List": func() ([]*v1.PartnerRole, error) {
					r, e := service.ListPartners(ctx, &v1.ListPartnersRequest{})
					if e != nil {
						return nil, e
					}
					return r.Data[0].Roles, nil
				},
				"Create": func() ([]*v1.PartnerRole, error) {
					r, e := service.CreatePartner(ctx, &v1.CreatePartnerRequest{LegalName: "测试客户", Roles: roles})
					if e != nil {
						return nil, e
					}
					return r.Data.Roles, nil
				},
				"Update": func() ([]*v1.PartnerRole, error) {
					r, e := service.UpdatePartner(ctx, &v1.UpdatePartnerRequest{Id: item.ID.String(), LegalName: "测试客户", Enabled: true, Roles: roles})
					if e != nil {
						return nil, e
					}
					return r.Data.Roles, nil
				},
				"UpdateRoles": func() ([]*v1.PartnerRole, error) {
					r, e := service.UpdatePartnerRoles(ctx, &v1.UpdatePartnerRolesRequest{PartnerId: item.ID.String(), RoleTypes: []v1.PartnerRoleType{v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER}})
					if e != nil {
						return nil, e
					}
					return r.Data.Roles, nil
				},
				"Blacklist": func() ([]*v1.PartnerRole, error) {
					r, e := service.SetPartnerRoleBlacklist(ctx, &v1.SetPartnerRoleBlacklistRequest{Id: item.ID.String(), RoleType: v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER})
					if e != nil {
						return nil, e
					}
					return r.Data.Roles, nil
				},
				"Export": func() ([]*v1.PartnerRole, error) {
					r, e := service.ExportPartners(ctx, &v1.ExportPartnersRequest{})
					if e != nil {
						return nil, e
					}
					return r.Data[0].Roles, nil
				},
			}
			for name, call := range calls {
				t.Run(name+"/"+resultCompanyID.String()+"/read="+map[bool]string{true: "true", false: "false"}[readable], func(t *testing.T) {
					got, err := call()
					if err != nil {
						t.Fatal(err)
					}
					if len(got) != 1 || (got[0].SettlementRule != nil) != (readable && sameCompany) {
						t.Fatalf("必须按结果公司和独立规则读权限裁剪: %+v", got)
					}
					if item.Roles[0].SettlementRule == nil {
						t.Fatal("输出裁剪不得修改领域对象")
					}
				})
			}
		}
	}
}

func TestPartnerAggregateSettlementWriteAccessPassed(t *testing.T) {
	companyID := uuid.New()
	for _, permission := range []string{"", access.PartnerSettlementRuleCreate, access.PartnerSettlementRuleUpdate} {
		principal := partnerSettlementAccessPrincipal(companyID, permission)
		item := &biz.Partner{ID: uuid.New(), OrganizationID: companyID}
		repo := &partnerSettlementAccessRepo{item: item}
		service := &PartnerService{usecase: biz.NewPartnerUsecase(repo)}
		ctx := biz.WithPrincipal(t.Context(), principal)
		roles := []*v1.PartnerRoleInput{{Type: v1.PartnerRoleType_PARTNER_ROLE_TYPE_CUSTOMER, Enabled: true}}
		calls := map[string]func() error{
			"Create": func() error {
				_, e := service.CreatePartner(ctx, &v1.CreatePartnerRequest{LegalName: "测试", Roles: roles})
				return e
			},
			"Update": func() error {
				_, e := service.UpdatePartner(ctx, &v1.UpdatePartnerRequest{Id: item.ID.String(), LegalName: "测试", Enabled: true, Roles: roles})
				return e
			},
			"Import": func() error {
				_, e := service.ImportPartners(ctx, &v1.ImportPartnersRequest{Source: "权限验证", Mode: v1.PartnerImportMode_PARTNER_IMPORT_MODE_CREATE_ONLY, Items: []*v1.PartnerImportItemInput{{LegalName: "测试", Roles: roles}}})
				return e
			},
		}
		for name, call := range calls {
			t.Run(name+"/"+permission, func(t *testing.T) {
				if err := call(); err != nil {
					t.Fatal(err)
				}
				want := biz.PartnerSettlementRuleWriteAccess{Create: permission == access.PartnerSettlementRuleCreate, Update: permission == access.PartnerSettlementRuleUpdate}
				if repo.ruleAccess != want {
					t.Fatalf("独立写能力传递错误: got=%+v want=%+v", repo.ruleAccess, want)
				}
			})
		}
	}
}
