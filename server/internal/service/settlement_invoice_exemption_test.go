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

type exemptionServiceRepoStub struct {
	biz.FinanceInvoiceRepo
	scope       []uuid.UUID
	actor, bill uuid.UUID
	version     uint64
	reason      string
	revoke      bool
	err         error
}

func (r *exemptionServiceRepoStub) SetBillInvoiceExemption(_ context.Context, scope []uuid.UUID, actor, bill uuid.UUID, version uint64, reason string, revoke bool, _ *biz.AuditEvent) error {
	r.scope, r.actor, r.bill, r.version, r.reason, r.revoke = scope, actor, bill, version, reason, revoke
	return r.err
}

type exemptionBillRepoStub struct {
	biz.FinanceBillRepo
	item *biz.FinanceBill
}

func (r *exemptionBillRepoStub) Get(context.Context, []uuid.UUID, uuid.UUID) (*biz.FinanceBill, error) {
	return r.item, nil
}

func TestBillInvoiceExemptionUsesDedicatedPermissionAndWritableCompany(t *testing.T) {
	organizationID, actorID, billID := uuid.New(), uuid.New(), uuid.New()
	for _, permission := range []string{access.FinanceInvoiceExempt, access.FinanceInvoiceUpdate, access.FinanceBillUpdate, ""} {
		t.Run(permission, func(t *testing.T) {
			repo := &exemptionServiceRepoStub{}
			service := &SettlementService{invoiceUsecase: biz.NewFinanceInvoiceUsecase(repo, nil, nil), billUsecase: biz.NewFinanceBillUsecase(&exemptionBillRepoStub{item: &biz.FinanceBill{ID: billID, OrganizationID: organizationID, Version: 3}}, nil, nil, nil)}
			principal := &biz.Principal{UserID: actorID, Organization: biz.Organization{ID: organizationID, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{permission: {}}}}}
			ctx := biz.WithPrincipal(context.Background(), principal)
			request := &v1.SetBillInvoiceExemptionRequest{Id: billID.String(), ExpectedVersion: 2, Reason: "  境外服务  "}
			response, err := service.SetBillInvoiceExemption(ctx, request)
			if permission != access.FinanceInvoiceExempt {
				if !errors.Is(err, biz.ErrPermissionDenied) || repo.scope != nil {
					t.Fatalf("其他财务写权限不能代替免票权限: %v", err)
				}
				return
			}
			if err != nil || response.GetData().GetId() != billID.String() || len(repo.scope) != 1 || repo.scope[0] != organizationID || repo.actor != actorID || repo.bill != billID || repo.version != 2 || repo.reason != "境外服务" || repo.revoke {
				t.Fatalf("免票范围或载荷错误: %+v %v", repo, err)
			}
			revoked, err := service.RevokeBillInvoiceExemption(ctx, &v1.RevokeBillInvoiceExemptionRequest{Id: request.Id, ExpectedVersion: request.ExpectedVersion, Reason: request.Reason})
			if err != nil || !repo.revoke || !revoked.GetSuccess() || revoked.GetData().GetId() != billID.String() {
				t.Fatalf("撤销入口未明确传递撤销语义: %v", err)
			}
		})
	}
}

func TestBillInvoiceExemptionValidatesReasonAndVersionBeforeRepository(t *testing.T) {
	for _, input := range []struct {
		version uint64
		reason  string
	}{{0, "免票"}, {1, " \n "}, {1, string(make([]rune, 501))}} {
		repo := &exemptionServiceRepoStub{}
		uc := biz.NewFinanceInvoiceUsecase(repo, nil, nil)
		err := uc.SetBillInvoiceExemption(context.Background(), []uuid.UUID{uuid.New()}, uuid.New(), uuid.New(), input.version, input.reason, false)
		if !errors.Is(err, biz.ErrFinanceInvoiceInvalidArgument) || repo.scope != nil {
			t.Fatalf("不合法免票请求进入仓储: %v", err)
		}
	}
}
