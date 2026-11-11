package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type partnerRolesRepo struct {
	biz.PartnerRepo
	organizationID uuid.UUID
	partnerID      uuid.UUID
	roles          []*biz.PartnerRole
}

func (r *partnerRolesRepo) UpdateRoles(_ context.Context, organizationID, partnerID uuid.UUID, roles []*biz.PartnerRole, _ *biz.AuditEvent) (*biz.PartnerUpdateResult, error) {
	r.organizationID, r.partnerID, r.roles = organizationID, partnerID, roles
	return &biz.PartnerUpdateResult{Partner: &biz.Partner{ID: partnerID, OrganizationID: organizationID, Roles: roles, IsCasual: true}}, nil
}

func TestUpdatePartnerRolesUsesCurrentCompanyAndDedicatedInput(t *testing.T) {
	companyID, actorID, partnerID := uuid.New(), uuid.New(), uuid.New()
	repo := &partnerRolesRepo{}
	service := &PartnerService{usecase: biz.NewPartnerUsecase(repo)}
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: actorID, Organization: biz.Organization{ID: companyID, Kind: biz.OrganizationKindCompany}})
	response, err := service.UpdatePartnerRoles(ctx, &v1.UpdatePartnerRolesRequest{PartnerId: partnerID.String(), RoleTypes: []v1.PartnerRoleType{v1.PartnerRoleType_PARTNER_ROLE_TYPE_SUPPLIER}})
	if err != nil || response.Data.Id != partnerID.String() || !response.Data.IsCasual || response.Data.Enabled {
		t.Fatalf("身份更新响应错误: response=%+v err=%v", response, err)
	}
	if repo.organizationID != companyID || repo.partnerID != partnerID || len(repo.roles) != 1 || repo.roles[0].Type != biz.PartnerRoleSupplier || !repo.roles[0].Enabled || repo.roles[0].SettlementRule != nil {
		t.Fatalf("身份更新不得提交其他档案数据: %+v", repo)
	}
	if _, err := service.UpdatePartnerRoles(t.Context(), &v1.UpdatePartnerRolesRequest{PartnerId: partnerID.String()}); err == nil {
		t.Fatal("缺少身份必须拒绝")
	}
	if _, err := service.UpdatePartnerRoles(ctx, &v1.UpdatePartnerRolesRequest{PartnerId: "invalid"}); err != biz.ErrPartnerNotFound {
		t.Fatalf("非法目标 ID 应拒绝: %v", err)
	}
}
