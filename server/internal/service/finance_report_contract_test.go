package service

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type reportContractRepo struct {
	biz.FinanceReportRepo
	profit   *biz.ProfitReportResult
	cashflow *biz.CashflowReportResult
	aging    *biz.AgingReportResult
}

func (r *reportContractRepo) Profit(context.Context, uuid.UUID, biz.FinanceReportFilter, bool) (*biz.ProfitReportResult, error) {
	return r.profit, nil
}
func (r *reportContractRepo) Cashflow(context.Context, uuid.UUID, biz.FinanceReportFilter, bool) (*biz.CashflowReportResult, error) {
	return r.cashflow, nil
}
func (r *reportContractRepo) Aging(context.Context, uuid.UUID, biz.FinanceReportFilter, bool, time.Time) (*biz.AgingReportResult, error) {
	return r.aging, nil
}

// Buf要求RPC消息独立命名；同源列表与导出的JSON字段及存在性仍必须一致。
func TestFinanceReportListAndExportWireContract(t *testing.T) {
	org, id := uuid.New(), uuid.New()
	day, reference, days := "2026-11-14", "参考号", 12
	now := time.Date(2026, 11, 14, 2, 3, 4, 0, time.UTC)
	amount := decimal.RequireFromString("123.45678901")
	profitAmount := biz.ProfitReportAmount{BaseCurrency: "CNY", ReceivableAmount: amount, PayableAmount: amount, ProfitAmount: amount, NetReceivableAmount: amount, NetPayableAmount: amount, NetProfitAmount: amount}
	cashAmount := biz.FinanceReportCashAmount{Currency: "CNY", ReceivableAmount: amount, PayableAmount: amount, NetAmount: amount}
	agingAmount := biz.AgingReportBucketAmount{Direction: "RECEIVABLE", Currency: "CNY", NotDue: amount, Days130: amount, Days3160: amount, Days6190: amount, Over90: amount, Undated: amount, Total: amount, BillCount: 7}
	repo := &reportContractRepo{
		profit:   &biz.ProfitReportResult{Items: []*biz.ProfitReportRow{{ID: id.String(), OrderID: id, OrderNo: "订单号", BusinessType: "SE", BusinessDate: &day, CustomerID: id, CustomerName: "客户", SalesName: "销售", OperatorName: "操作", CustomerServiceName: "客服", BaseCurrency: "CNY", ReceivableAmount: amount, PayableAmount: amount, ProfitAmount: amount, NetReceivableAmount: amount, NetPayableAmount: amount, NetProfitAmount: amount, FeeCount: 2}}, Total: 1, OrderCount: 1, FeeCount: 2, UndatedOrderCount: 3, Totals: []biz.ProfitReportAmount{profitAmount}, NativeTotals: []biz.FinanceReportNativeAmount{{Direction: "RECEIVABLE", Currency: "USD", Amount: amount, NetAmount: amount}}, Months: []biz.ProfitReportMonth{{Month: "2026-11", Amounts: []biz.ProfitReportAmount{profitAmount}}}, GeneratedAt: now},
		cashflow: &biz.CashflowReportResult{Items: []*biz.CashflowReportRow{{ID: id, FlowNo: "流水号", Direction: "PAYABLE", TransactionDate: day, SettlementPartyID: id, SettlementPartyName: "单位", Currency: "CNY", BaseCurrency: "CNY", OurAccount: "账户", PaymentMethod: "转账", Amount: amount, BaseAmount: amount, BankReferenceNo: &reference}}, Total: 1, NativeTotals: []biz.FinanceReportCashAmount{cashAmount}, BaseTotals: []biz.FinanceReportCashAmount{cashAmount}, GeneratedAt: now},
		aging:    &biz.AgingReportResult{Items: []*biz.AgingReportRow{{ID: id, BillNo: "账单号", Direction: "RECEIVABLE", SettlementPartyID: id, SettlementPartyName: "单位", BillDate: day, DueDate: &day, Currency: "CNY", BaseCurrency: "CNY", Bucket: "DAYS_1_30", TotalAmount: amount, SettledAmount: amount, Balance: amount, BaseBalance: amount, OverdueDays: &days}}, Total: 1, NativeTotals: []biz.AgingReportBucketAmount{agingAmount}, BaseTotals: []biz.AgingReportBucketAmount{agingAmount}, AsOfDate: day, GeneratedAt: now},
	}
	service := NewFinanceReportService(biz.NewFinanceReportUsecase(repo))
	ctx := biz.WithPrincipal(context.Background(), reportTestPrincipal(org, biz.OrganizationKindCompany, access.FinanceReportRead))
	tests := []struct {
		name         string
		list, export func() (proto.Message, error)
	}{
		{"利润", func() (proto.Message, error) {
			return service.ListProfitReport(ctx, &v1.ListProfitReportRequest{OrganizationId: org.String()})
		}, func() (proto.Message, error) {
			return service.ExportProfitReport(ctx, &v1.ExportProfitReportRequest{OrganizationId: org.String()})
		}},
		{"收付", func() (proto.Message, error) {
			return service.ListCashflowReport(ctx, &v1.ListCashflowReportRequest{OrganizationId: org.String()})
		}, func() (proto.Message, error) {
			return service.ExportCashflowReport(ctx, &v1.ExportCashflowReportRequest{OrganizationId: org.String()})
		}},
		{"账龄", func() (proto.Message, error) {
			return service.ListAgingReport(ctx, &v1.ListAgingReportRequest{OrganizationId: org.String()})
		}, func() (proto.Message, error) {
			return service.ExportAgingReport(ctx, &v1.ExportAgingReportRequest{OrganizationId: org.String()})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list, err := test.list()
			if err != nil {
				t.Fatal(err)
			}
			exported, err := test.export()
			if err != nil {
				t.Fatal(err)
			}
			left, err := protojson.Marshal(list)
			if err != nil {
				t.Fatal(err)
			}
			right, err := protojson.Marshal(exported)
			if err != nil {
				t.Fatal(err)
			}
			var leftValue, rightValue any
			if err := json.Unmarshal(left, &leftValue); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(right, &rightValue); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(leftValue, rightValue) {
				t.Fatalf("独立RPC消息不应改变同源JSON：列表=%s 导出=%s", left, right)
			}
		})
	}
}
