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

type financePartyScopeRepo struct {
	biz.SettlementRepo
	organizationID uuid.UUID
	keyword        string
	calls          int
	err            error
}

func (r *financePartyScopeRepo) ListFinanceSettlementParties(_ context.Context, organizationID uuid.UUID, keyword string, _, _ int) ([]*biz.FinanceSettlementPartyOption, int64, error) {
	r.calls++
	r.organizationID, r.keyword = organizationID, keyword
	return []*biz.FinanceSettlementPartyOption{{ID: uuid.NewString(), Name: "当前公司结算单位", IsCasual: true}}, 1, r.err
}

// 删除公司候选不能改变保留的两种结算单位用途及独立动作权限。
func TestFinanceSettlementPartyOptionsKeepCurrentCompanyAndActionPermission(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		purpose    v1.FinanceOrganizationPurpose
		permission string
		other      v1.FinanceOrganizationPurpose
	}{
		{"收付创建", v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_CASHFLOW_CREATE, access.FinanceCashflowCreate, v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE},
		{"核销创建", v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE, access.FinanceVerificationCreate, v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_CASHFLOW_CREATE},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			current, foreign := uuid.New(), uuid.New()
			principal := &biz.Principal{Organization: biz.Organization{ID: current, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: current, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "creator", DataScope: biz.DataScopeAll, Permissions: map[string]struct{}{scenario.permission: {}}}}}
			ctx := biz.WithPrincipal(context.Background(), principal)
			repo := &financePartyScopeRepo{}
			svc := &SettlementService{usecase: biz.NewSettlementUsecase(repo)}
			keyword := " 结算单位 "
			request := &v1.ListFinanceSettlementPartyOptionsRequest{Purpose: scenario.purpose, OrganizationId: current.String(), Page: 1, PageSize: 20, Keyword: &keyword}
			response, err := svc.ListFinanceSettlementPartyOptions(ctx, request)
			if err != nil || !response.GetSuccess() || response.GetTotal() != 1 || len(response.GetData()) != 1 || repo.organizationID != current || repo.keyword != "结算单位" {
				t.Fatalf("当前公司候选失败: response=%v repo=%+v err=%v", response, repo, err)
			}
			request.Purpose = scenario.other
			if _, err := svc.ListFinanceSettlementPartyOptions(ctx, request); err != biz.ErrPermissionDenied {
				t.Fatalf("其他动作不能借用权限: %v", err)
			}
			request.Purpose = scenario.purpose
			request.OrganizationId = foreign.String()
			if _, err := svc.ListFinanceSettlementPartyOptions(ctx, request); err != biz.ErrPermissionDenied {
				t.Fatalf("其他公司不得使用All范围越界: %v", err)
			}
			request.OrganizationId = "not-a-uuid"
			if _, err := svc.ListFinanceSettlementPartyOptions(ctx, request); err != biz.ErrFinanceLedgerInvalidArgument {
				t.Fatalf("错误公司ID未拒绝: %v", err)
			}
			if repo.calls != 1 {
				t.Fatalf("拒绝的请求调用了仓储: %d", repo.calls)
			}
			request.OrganizationId = current.String()
			repo.err = errors.New("结算单位数据库错误")
			if _, err := svc.ListFinanceSettlementPartyOptions(ctx, request); !errors.Is(err, repo.err) {
				t.Fatalf("数据库错误未原样外传: %v", err)
			}
		})
	}
}

func TestFinanceSettlementPartyOptionsRejectRetiredPurposes(t *testing.T) {
	principal := &biz.Principal{Organization: biz.Organization{ID: uuid.New(), Kind: biz.OrganizationKindCompany}}
	ctx := biz.WithPrincipal(context.Background(), principal)
	repo := &financePartyScopeRepo{}
	svc := &SettlementService{usecase: biz.NewSettlementUsecase(repo)}
	for _, number := range []int32{0, 1, 2, 3, 5, 6, 7, 8, 10, 11, 12, 99} {
		_, err := svc.ListFinanceSettlementPartyOptions(ctx, &v1.ListFinanceSettlementPartyOptionsRequest{Purpose: v1.FinanceOrganizationPurpose(number), OrganizationId: principal.Organization.ID.String(), Page: 1, PageSize: 20})
		if err != biz.ErrFinanceLedgerInvalidArgument {
			t.Errorf("无效用途%d未拒绝: %v", number, err)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("无效用途调用了仓储: %d", repo.calls)
	}
}
