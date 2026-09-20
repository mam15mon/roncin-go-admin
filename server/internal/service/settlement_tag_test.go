package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type financeTagOptionRepoStub struct {
	biz.BusinessTagRepo
	organizationID uuid.UUID
}

func (s *financeTagOptionRepoStub) ListTagOptions(_ context.Context, organizationID uuid.UUID, _ string, _, _ int) ([]*biz.BusinessTagSummary, int64, error) {
	s.organizationID = organizationID
	return []*biz.BusinessTagSummary{{ID: uuid.New(), Name: "重点客户", GroupID: uuid.New(), Enabled: true}}, 1, nil
}

func financeTagCandidatePrincipal(permission string, organizationID uuid.UUID) context.Context {
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{Kind: biz.OrganizationKindCompany, ID: organizationID}, OrganizationNodes: []biz.OrganizationScopeNode{{Kind: biz.OrganizationKindCompany, ID: organizationID}},
		RoleGrants: []biz.RoleGrant{{RoleCode: "tag-manager", DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{permission: {}}}},
	})
}

func TestFinanceTagAssignmentCandidatesUseActionWritableOrganization(t *testing.T) {
	allowed, denied := uuid.New(), uuid.New()
	cases := []struct {
		name       string
		permission string
		call       func(*SettlementService, context.Context, string) error
	}{
		{
			name:       "账单标签",
			permission: access.FinanceBillUpdate,
			call: func(service *SettlementService, ctx context.Context, organizationID string) error {
				_, err := service.ListFinanceBillTagAssignmentOptions(ctx, &v1.ListFinanceBillTagAssignmentOptionsRequest{OrganizationId: organizationID, Page: 1, PageSize: 20})
				return err
			},
		},
		{
			name:       "费用标签",
			permission: access.FinanceFeeTag,
			call: func(service *SettlementService, ctx context.Context, organizationID string) error {
				_, err := service.ListFinanceFeeTagAssignmentOptions(ctx, &v1.ListFinanceFeeTagAssignmentOptionsRequest{OrganizationId: organizationID, Page: 1, PageSize: 20})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &financeTagOptionRepoStub{}
			service := NewSettlementService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, biz.NewBusinessTagUsecase(repo), nil)
			ctx := financeTagCandidatePrincipal(tc.permission, allowed)

			if err := tc.call(service, ctx, allowed.String()); err != nil {
				t.Fatalf("仅动作权限的候选查询失败: %v", err)
			}
			if repo.organizationID != allowed {
				t.Fatalf("候选查询组织 = %s，期望 %s", repo.organizationID, allowed)
			}
			if err := tc.call(service, ctx, denied.String()); !errors.Is(err, biz.ErrPermissionDenied) {
				t.Fatalf("越权组织错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
			}
			if err := tc.call(service, ctx, ""); !errors.Is(err, biz.ErrBusinessTagInvalidArgument) {
				t.Fatalf("空组织错误 = %v，期望 %v", err, biz.ErrBusinessTagInvalidArgument)
			}

			readOnlyPermission := access.FinanceBillRead
			if tc.permission == access.FinanceFeeTag {
				readOnlyPermission = access.FinanceFeeRead
			}
			if err := tc.call(service, financeTagCandidatePrincipal(readOnlyPermission, allowed), allowed.String()); !errors.Is(err, biz.ErrPermissionDenied) {
				t.Fatalf("只读权限错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
			}
		})
	}
}
