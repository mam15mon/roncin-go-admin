package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestVerificationShortfallRequiresEachPermission(t *testing.T) {
	for _, scenario := range []string{"缺流水创建", "缺流水确认", "缺汇率维护", "其他公司", "未带差额时不接受空流水ID"} {
		t.Run(scenario, func(t *testing.T) {
			org := uuid.New()
			permissions := map[string]struct{}{access.FinanceVerificationCreate: {}, access.FinanceCashflowCreate: {}, access.FinanceCashflowUpdate: {}}
			request := &v1.CreateVerificationRequest{Allocations: []*v1.VerificationAllocationInput{{BillId: uuid.NewString(), Amount: "40"}}, VerificationDate: "2026-11-14", IdempotencyKey: "permission-test", ShortfallCashflow: &v1.VerificationShortfallCashflow{OrganizationId: org.String(), Direction: "RECEIVABLE", SettlementPartyId: uuid.NewString(), Currency: "CNY", Amount: "40", TransactionDate: "2026-11-14", OurAccount: "收款账户", PaymentMethod: "BANK_TRANSFER"}}
			switch scenario {
			case "缺流水创建":
				delete(permissions, access.FinanceCashflowCreate)
			case "缺流水确认":
				delete(permissions, access.FinanceCashflowUpdate)
			case "缺汇率维护":
				rate := "1"
				request.ShortfallCashflow.ExchangeRate = &rate
			case "其他公司":
				request.ShortfallCashflow.OrganizationId = uuid.NewString()
			case "未带差额时不接受空流水ID":
				request.ShortfallCashflow = nil
			}
			principal := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization, Permissions: permissions}}}
			service := &SettlementService{}
			_, err := service.CreateVerification(biz.WithPrincipal(context.Background(), principal), request)
			if scenario == "未带差额时不接受空流水ID" {
				if err != biz.ErrVerificationInvalid {
					t.Fatalf("error=%v，期望非法参数", err)
				}
			} else if err != biz.ErrPermissionDenied {
				t.Fatalf("error=%v，期望权限拒绝", err)
			}
		})
	}
}
