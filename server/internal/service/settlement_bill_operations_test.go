package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

type compositionBillServiceRepo struct {
	*billCreationCandidateServiceRepoStub
}

func (r compositionBillServiceRepo) LoadBillableFees(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	out := []*biz.FinanceBillableFee{}
	for _, fee := range r.fees {
		for _, id := range ids {
			if fee.Fee.ID == id {
				out = append(out, fee)
			}
		}
	}
	return out, nil
}

func TestBillBatchMalformedInvoiceKeepsOtherGroup(t *testing.T) {
	for _, malformedField := range []string{"amount", "tax", "precision", "exponent"} {
		t.Run(malformedField, func(t *testing.T) {
			org, party, order, account := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			fees := []*biz.FinanceBillableFee{}
			for _, direction := range []biz.OrderFeeDirection{biz.OrderFeeReceivable, biz.OrderFeePayable} {
				tax := decimal.Zero
				fees = append(fees, &biz.FinanceBillableFee{OrganizationID: org, OrderNo: "TEST-ORDER", BusinessType: "SE", Fee: &biz.OrderFee{ID: uuid.New(), OrderID: order, Direction: direction, SettlementPartyID: party, SettlementPartyName: "测试客商", FeeCode: "OCEAN", FeeName: "海运费", Currency: "CNY", BaseCurrency: "CNY", TaxRate: &tax, ExchangeRate: decimal.NewFromInt(1), Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100), NetAmount: decimal.NewFromInt(100), BaseCurrencyAmount: decimal.NewFromInt(100), Version: 1}})
			}
			principal := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: org, Kind: biz.OrganizationKindCompany}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: org, Kind: biz.OrganizationKindCompany}}, RoleGrants: []biz.RoleGrant{{RoleCode: "creator", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceBillCreate: {}, access.FinanceInvoiceCreate: {}}}}}
			repo := compositionBillServiceRepo{&billCreationCandidateServiceRepoStub{fees: fees}}
			rateRepo := &exchangeRateStub{context: &biz.ExchangeRateContext{OwnerOrganizationID: org, BaseCurrency: "CNY"}, rate: decimal.NewFromInt(1)}
			service := &SettlementService{billUsecase: biz.NewFinanceBillUsecase(repo, biz.NewExchangeRateUsecase(rateRepo, nil), settlementServiceTransactorStub{}, nil)}
			ctx := biz.WithPrincipal(context.Background(), principal)
			policy := &v1.BillGroupingPolicy{Mode: v1.BillGroupingMode_BILL_GROUPING_MODE_NORMAL}
			feeIDs := []string{fees[0].Fee.ID.String(), fees[1].Fee.ID.String()}
			initial, err := service.PreviewBillBatch(ctx, &v1.PreviewBillBatchRequest{OrganizationId: org.String(), GroupingPolicy: policy, FeeIds: feeIDs})
			if err != nil {
				t.Fatal(err)
			}
			configs := []*v1.BillBatchPreviewGroupConfigInput{}
			for _, group := range initial.Data {
				configs = append(configs, &v1.BillBatchPreviewGroupConfigInput{GroupKey: group.GroupKey, SettlementAccountId: account.String()})
			}
			preview, err := service.PreviewBillBatch(ctx, &v1.PreviewBillBatchRequest{OrganizationId: org.String(), GroupingPolicy: policy, FeeIds: feeIDs, GroupConfigs: configs})
			if err != nil {
				t.Fatal(err)
			}
			request := &v1.CreateBillBatchRequest{OrganizationId: org.String(), GroupingPolicy: policy, IdempotencyKey: "malformed-" + malformedField}
			for _, group := range preview.Data {
				input := &v1.CreateBillBatchGroupInput{GroupKey: group.GroupKey, GroupToken: group.GroupToken, StatementTitle: "测试客商", SettlementAccountId: account.String(), FeeIds: []string{group.Fees[0].Id}}
				if group.Direction == "RECEIVABLE" {
					fee := &v1.InvoiceRegistrationFeeInput{OrderFeeId: group.Fees[0].Id, Amount: "100", TaxAmount: "0"}
					switch malformedField {
					case "amount":
						fee.Amount = "invalid"
					case "tax":
						fee.TaxAmount = "invalid"
					case "precision":
						fee.Amount = "0.123456789"
					case "exponent":
						fee.Amount = "1e1000000000"
					}
					input.Invoices = []*v1.BillBatchInvoiceInput{{InvoiceProfileId: uuid.NewString(), InvoiceType: "DIGITAL_NORMAL", TaxInvoiceNo: "TEST-001", InvoiceDate: "2026-09-10", TotalAmount: "100", Fees: []*v1.InvoiceRegistrationFeeInput{fee}}}
				}
				request.Groups = append(request.Groups, input)
			}
			response, err := service.CreateBillBatch(ctx, request)
			if err != nil || response.SucceededCount != 1 || response.FailedCount != 1 {
				t.Fatalf("非法发票不应阻断其他组: %+v %v", response, err)
			}
			for _, result := range response.Results {
				if result.Success {
					if result.Bill.Direction != "PAYABLE" {
						t.Fatal("应付合法组未成功")
					}
					continue
				}
				if result.Error == nil || len(result.Error.Items) != 1 || result.Error.Items[0].Step != "INVOICE" || result.Error.Items[0].ItemIndex != 1 {
					t.Fatalf("缺少发票定位: %+v", result)
				}
				if strings.Contains(result.Error.Message, "invalid") {
					t.Fatal("内部未解析值不应透出")
				}
			}
		})
	}
}
