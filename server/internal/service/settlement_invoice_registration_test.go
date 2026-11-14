package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestInvoiceRevisionCannotAddWithUpdatePermissionAlone(t *testing.T) {
	org, actor := uuid.New(), uuid.New()
	principal := &biz.Principal{UserID: actor, Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "finance", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceInvoiceUpdate: {}}}}}
	s := &SettlementService{}
	_, err := s.ReviseInvoiceRegistrations(biz.WithPrincipal(context.Background(), principal), &v1.ReviseInvoiceRegistrationsRequest{IdempotencyKey: "new-with-update", Reason: "新增票据", Entries: []*v1.InvoiceRegistrationRevisionEntry{{Registration: &v1.InvoiceRegistrationInput{}}}})
	if !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("更新授权不能新增发票，必须在进入用例前拒绝: %v", err)
	}
}
func TestInvoiceAllocationWirePreservesUnknownAndZeroSourceRate(t *testing.T) {
	zero := decimal.Zero
	for _, tc := range []struct {
		name string
		rate *decimal.Decimal
		want bool
	}{{"未设置", nil, false}, {"显式零", &zero, true}} {
		t.Run(tc.name, func(t *testing.T) {
			value := invoiceAllocationToAPI(&biz.FinanceInvoiceFeeAllocation{SourceTaxRate: tc.rate, Amount: decimal.NewFromInt(10), TaxAmount: decimal.Zero, NetAmount: decimal.NewFromInt(10)})
			wire, err := protojson.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var decoded v1.FinanceInvoiceFeeAllocation
			if err := protojson.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			if (decoded.SourceTaxRate != nil) != tc.want {
				t.Fatalf("存在性丢失: %s", wire)
			}
			if tc.want && decoded.GetSourceTaxRate() != "0.0000" {
				t.Fatalf("真实零被丢失: %s", wire)
			}
		})
	}
}
