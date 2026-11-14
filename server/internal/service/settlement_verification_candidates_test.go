package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestVerificationBillCandidateScopePaginationAndKeyword(t *testing.T) {
	org, party := uuid.New(), uuid.New()
	principal := &biz.Principal{Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "creator", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceVerificationCreate: {}}}}}
	ctx := biz.WithPrincipal(context.Background(), principal)
	repo := &verificationCreationCandidateRepoStub{}
	service := &SettlementService{verificationUsecase: biz.NewVerificationUsecase(repo, nil, nil, nil)}
	keyword := " qdyh "
	request := &v1.ListVerificationBillCandidatesRequest{OrganizationId: org.String(), Direction: "receivable", SettlementPartyId: party.String(), Currency: " cny ", Page: 2, PageSize: 30, Keyword: &keyword}
	response, err := service.ListVerificationBillCandidates(ctx, request)
	if err != nil || response == nil {
		t.Fatalf("查询账单候选: %v", err)
	}
	if repo.organizationID != org || repo.filter.Page != 2 || repo.filter.PageSize != 30 || repo.filter.Keyword != "qdyh" || repo.filter.Currency != "CNY" || repo.filter.SettlementPartyID != party || repo.filter.Direction != biz.OrderFeeReceivable {
		t.Fatalf("候选范围未完整传递: %+v", repo.filter)
	}
	request.OrganizationId = uuid.NewString()
	if _, err := service.ListVerificationBillCandidates(ctx, request); err != biz.ErrPermissionDenied {
		t.Fatalf("跨公司错误: %v", err)
	}
	request.OrganizationId = org.String()
	request.PageSize = 201
	if _, err := service.ListVerificationBillCandidates(ctx, request); err != biz.ErrVerificationInvalid {
		t.Fatalf("超过200分页错误: %v", err)
	}
}
