package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type FinanceReportService struct {
	v1.UnimplementedFinanceReportServiceServer
	usecase *biz.FinanceReportUsecase
}

func NewFinanceReportService(usecase *biz.FinanceReportUsecase) *FinanceReportService {
	return &FinanceReportService{usecase: usecase}
}

func financeReportOrganization(ctx context.Context, raw string) (uuid.UUID, error) {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	org, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, biz.ErrFinanceReportInvalid
	}
	if p.Organization.Kind != biz.OrganizationKindCompany || org != p.Organization.ID {
		return uuid.Nil, biz.ErrPermissionDenied
	}
	if err := currentOrganizationAllowedForPermission(p, access.FinanceReportRead, false); err != nil {
		return uuid.Nil, err
	}
	return org, nil
}
func reportOptionalID(raw string) (*uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil || id == uuid.Nil {
		return nil, biz.ErrFinanceReportInvalid
	}
	return &id, nil
}
func reportIDs(values ...string) ([]*uuid.UUID, error) {
	ids := make([]*uuid.UUID, len(values))
	for i, value := range values {
		id, err := reportOptionalID(value)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}
	return ids, nil
}

type profitReportRequest interface {
	GetDateFrom() string
	GetDateTo() string
	GetKeyword() string
	GetBusinessType() string
	GetCustomerId() string
	GetSalesId() string
	GetOperatorId() string
	GetCustomerServiceId() string
}

func profitReportFilter(req profitReportRequest) (biz.FinanceReportFilter, error) {
	ids, err := reportIDs(req.GetCustomerId(), req.GetSalesId(), req.GetOperatorId(), req.GetCustomerServiceId())
	if err != nil {
		return biz.FinanceReportFilter{}, err
	}
	return biz.FinanceReportFilter{DateFrom: req.GetDateFrom(), DateTo: req.GetDateTo(), Keyword: req.GetKeyword(), BusinessType: req.GetBusinessType(), CustomerID: ids[0], SalesID: ids[1], OperatorID: ids[2], CustomerServiceID: ids[3]}, nil
}

type cashflowReportRequest interface {
	GetDateFrom() string
	GetDateTo() string
	GetKeyword() string
	GetDirection() string
	GetSettlementPartyId() string
	GetCurrency() string
	GetPaymentMethod() string
	GetOurAccount() string
}

func cashflowReportFilter(req cashflowReportRequest) (biz.FinanceReportFilter, error) {
	id, err := reportOptionalID(req.GetSettlementPartyId())
	if err != nil {
		return biz.FinanceReportFilter{}, err
	}
	return biz.FinanceReportFilter{DateFrom: req.GetDateFrom(), DateTo: req.GetDateTo(), Keyword: req.GetKeyword(), Direction: biz.OrderFeeDirection(req.GetDirection()), SettlementPartyID: id, Currency: req.GetCurrency(), PaymentMethod: req.GetPaymentMethod(), OurAccount: req.GetOurAccount()}, nil
}

type agingReportRequest interface {
	GetKeyword() string
	GetDirection() string
	GetSettlementPartyId() string
	GetCurrency() string
	GetBucket() string
}

func agingReportFilter(req agingReportRequest) (biz.FinanceReportFilter, error) {
	id, err := reportOptionalID(req.GetSettlementPartyId())
	if err != nil {
		return biz.FinanceReportFilter{}, err
	}
	return biz.FinanceReportFilter{Keyword: req.GetKeyword(), Direction: biz.OrderFeeDirection(req.GetDirection()), SettlementPartyID: id, Currency: req.GetCurrency(), Bucket: req.GetBucket()}, nil
}

func (s *FinanceReportService) ListProfitReport(ctx context.Context, req *v1.ListProfitReportRequest) (*v1.ListProfitReportResponse, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	f, err := profitReportFilter(req)
	if err != nil {
		return nil, err
	}
	f.Page, f.PageSize, err = listPageValues(req.GetPage(), req.GetPageSize(), biz.ErrFinanceReportInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Profit(ctx, org, f, false)
	if err != nil {
		return nil, err
	}
	return ok(ctx, profitReportToAPI(result)), nil
}
func (s *FinanceReportService) ExportProfitReport(ctx context.Context, req *v1.ExportProfitReportRequest) (*v1.ExportProfitReportResponse, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	f, err := profitReportFilter(req)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Profit(ctx, org, f, true)
	if err != nil {
		return nil, err
	}
	mapped := profitReportToAPI(result)
	return ok(ctx, &v1.ExportProfitReportResponse{Success: mapped.Success, Code: mapped.Code, Message: mapped.Message, Data: mapped.Data, Total: mapped.Total, TraceId: mapped.TraceId, Totals: mapped.Totals, NativeTotals: mapped.NativeTotals, Months: mapped.Months, OrderCount: mapped.OrderCount, FeeCount: mapped.FeeCount, UndatedOrderCount: mapped.UndatedOrderCount, GeneratedAt: mapped.GeneratedAt}), nil
}
func (s *FinanceReportService) ListCashflowReport(ctx context.Context, req *v1.ListCashflowReportRequest) (*v1.ListCashflowReportResponse, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	f, err := cashflowReportFilter(req)
	if err != nil {
		return nil, err
	}
	f.Page, f.PageSize, err = listPageValues(req.GetPage(), req.GetPageSize(), biz.ErrFinanceReportInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Cashflow(ctx, org, f, false)
	if err != nil {
		return nil, err
	}
	return ok(ctx, cashflowReportToAPI(result)), nil
}
func (s *FinanceReportService) ExportCashflowReport(ctx context.Context, req *v1.ExportCashflowReportRequest) (*v1.ExportCashflowReportResponse, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	f, err := cashflowReportFilter(req)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Cashflow(ctx, org, f, true)
	if err != nil {
		return nil, err
	}
	mapped := cashflowReportToAPI(result)
	return ok(ctx, &v1.ExportCashflowReportResponse{Success: mapped.Success, Code: mapped.Code, Message: mapped.Message, Data: mapped.Data, Total: mapped.Total, TraceId: mapped.TraceId, NativeTotals: mapped.NativeTotals, BaseTotals: mapped.BaseTotals, GeneratedAt: mapped.GeneratedAt}), nil
}
func (s *FinanceReportService) ListAgingReport(ctx context.Context, req *v1.ListAgingReportRequest) (*v1.ListAgingReportResponse, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	f, err := agingReportFilter(req)
	if err != nil {
		return nil, err
	}
	f.Page, f.PageSize, err = listPageValues(req.GetPage(), req.GetPageSize(), biz.ErrFinanceReportInvalid)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Aging(ctx, org, f, false)
	if err != nil {
		return nil, err
	}
	return ok(ctx, agingReportToAPI(result)), nil
}
func (s *FinanceReportService) ExportAgingReport(ctx context.Context, req *v1.ExportAgingReportRequest) (*v1.ExportAgingReportResponse, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, err
	}
	f, err := agingReportFilter(req)
	if err != nil {
		return nil, err
	}
	result, err := s.usecase.Aging(ctx, org, f, true)
	if err != nil {
		return nil, err
	}
	mapped := agingReportToAPI(result)
	return ok(ctx, &v1.ExportAgingReportResponse{Success: mapped.Success, Code: mapped.Code, Message: mapped.Message, Data: mapped.Data, Total: mapped.Total, TraceId: mapped.TraceId, NativeTotals: mapped.NativeTotals, BaseTotals: mapped.BaseTotals, AsOfDate: mapped.AsOfDate, GeneratedAt: mapped.GeneratedAt}), nil
}

type reportOptionsRequest interface {
	GetOrganizationId() string
	GetPage() int32
	GetPageSize() int32
	GetKeyword() string
}

func (s *FinanceReportService) ListFinanceReportPartyOptions(ctx context.Context, req *v1.ListFinanceReportPartyOptionsRequest) (*v1.ListFinanceReportPartyOptionsResponse, error) {
	items, total, err := s.reportOptions(ctx, req, false)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ListFinanceReportPartyOptionsResponse{Data: items, Total: total}), nil
}
func (s *FinanceReportService) ListFinanceReportPersonnelOptions(ctx context.Context, req *v1.ListFinanceReportPersonnelOptionsRequest) (*v1.ListFinanceReportPersonnelOptionsResponse, error) {
	items, total, err := s.reportOptions(ctx, req, true)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ListFinanceReportPersonnelOptionsResponse{Data: items, Total: total}), nil
}
func (s *FinanceReportService) reportOptions(ctx context.Context, req reportOptionsRequest, personnel bool) ([]*v1.FinanceReportOption, int64, error) {
	org, err := financeReportOrganization(ctx, req.GetOrganizationId())
	if err != nil {
		return nil, 0, err
	}
	page, size, err := listPageValues(req.GetPage(), req.GetPageSize(), biz.ErrFinanceReportInvalid)
	if err != nil {
		return nil, 0, err
	}
	result, err := s.usecase.Options(ctx, org, biz.SelectorListOptions{Page: page, PageSize: size, Keyword: req.GetKeyword()}, personnel)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*v1.FinanceReportOption, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &v1.FinanceReportOption{Id: item.ID.String(), Name: item.Name})
	}
	return items, int64(result.Total), nil
}

func profitReportAmountToAPI(x biz.ProfitReportAmount) *v1.ProfitReportAmount {
	return &v1.ProfitReportAmount{BaseCurrency: x.BaseCurrency, ReceivableAmount: x.ReceivableAmount.StringFixed(8), PayableAmount: x.PayableAmount.StringFixed(8), ProfitAmount: x.ProfitAmount.StringFixed(8), NetReceivableAmount: x.NetReceivableAmount.StringFixed(8), NetPayableAmount: x.NetPayableAmount.StringFixed(8), NetProfitAmount: x.NetProfitAmount.StringFixed(8)}
}
func profitReportToAPI(x *biz.ProfitReportResult) *v1.ListProfitReportResponse {
	out := &v1.ListProfitReportResponse{Data: make([]*v1.ProfitReportRow, 0, len(x.Items)), Total: x.Total, OrderCount: x.OrderCount, FeeCount: x.FeeCount, UndatedOrderCount: x.UndatedOrderCount, GeneratedAt: x.GeneratedAt.Format(time.RFC3339)}
	for _, r := range x.Items {
		out.Data = append(out.Data, &v1.ProfitReportRow{Id: r.ID, OrderId: r.OrderID.String(), OrderNo: r.OrderNo, BusinessType: r.BusinessType, BusinessDate: r.BusinessDate, CustomerId: r.CustomerID.String(), CustomerName: r.CustomerName, SalesName: r.SalesName, OperatorName: r.OperatorName, CustomerServiceName: r.CustomerServiceName, BaseCurrency: r.BaseCurrency, ReceivableAmount: r.ReceivableAmount.StringFixed(8), PayableAmount: r.PayableAmount.StringFixed(8), ProfitAmount: r.ProfitAmount.StringFixed(8), NetReceivableAmount: r.NetReceivableAmount.StringFixed(8), NetPayableAmount: r.NetPayableAmount.StringFixed(8), NetProfitAmount: r.NetProfitAmount.StringFixed(8), FeeCount: r.FeeCount})
	}
	for _, r := range x.Totals {
		out.Totals = append(out.Totals, profitReportAmountToAPI(r))
	}
	for _, r := range x.NativeTotals {
		out.NativeTotals = append(out.NativeTotals, &v1.FinanceReportNativeAmount{Direction: r.Direction, Currency: r.Currency, Amount: r.Amount.StringFixed(8), NetAmount: r.NetAmount.StringFixed(8)})
	}
	for _, r := range x.Months {
		month := &v1.ProfitReportMonth{Month: r.Month}
		for _, a := range r.Amounts {
			month.Amounts = append(month.Amounts, profitReportAmountToAPI(a))
		}
		out.Months = append(out.Months, month)
	}
	return out
}
func cashReportAmountsToAPI(items []biz.FinanceReportCashAmount) []*v1.FinanceReportCashAmount {
	out := make([]*v1.FinanceReportCashAmount, 0, len(items))
	for _, r := range items {
		out = append(out, &v1.FinanceReportCashAmount{Currency: r.Currency, ReceivableAmount: r.ReceivableAmount.StringFixed(8), PayableAmount: r.PayableAmount.StringFixed(8), NetAmount: r.NetAmount.StringFixed(8)})
	}
	return out
}
func cashflowReportToAPI(x *biz.CashflowReportResult) *v1.ListCashflowReportResponse {
	out := &v1.ListCashflowReportResponse{Data: make([]*v1.CashflowReportRow, 0, len(x.Items)), Total: x.Total, NativeTotals: cashReportAmountsToAPI(x.NativeTotals), BaseTotals: cashReportAmountsToAPI(x.BaseTotals), GeneratedAt: x.GeneratedAt.Format(time.RFC3339)}
	for _, r := range x.Items {
		out.Data = append(out.Data, &v1.CashflowReportRow{Id: r.ID.String(), FlowNo: r.FlowNo, Direction: r.Direction, TransactionDate: r.TransactionDate, SettlementPartyId: r.SettlementPartyID.String(), SettlementPartyName: r.SettlementPartyName, Currency: r.Currency, Amount: r.Amount.StringFixed(8), BaseCurrency: r.BaseCurrency, BaseAmount: r.BaseAmount.StringFixed(8), OurAccount: r.OurAccount, PaymentMethod: r.PaymentMethod, BankReferenceNo: r.BankReferenceNo})
	}
	return out
}
func agingReportToAPI(x *biz.AgingReportResult) *v1.ListAgingReportResponse {
	out := &v1.ListAgingReportResponse{Data: make([]*v1.AgingReportRow, 0, len(x.Items)), Total: x.Total, AsOfDate: x.AsOfDate, GeneratedAt: x.GeneratedAt.Format(time.RFC3339), NativeTotals: agingReportAmountsToAPI(x.NativeTotals), BaseTotals: agingReportAmountsToAPI(x.BaseTotals)}
	for _, r := range x.Items {
		var days *int32
		if r.OverdueDays != nil {
			v := int32(*r.OverdueDays)
			days = &v
		}
		out.Data = append(out.Data, &v1.AgingReportRow{Id: r.ID.String(), BillNo: r.BillNo, Direction: r.Direction, SettlementPartyId: r.SettlementPartyID.String(), SettlementPartyName: r.SettlementPartyName, BillDate: r.BillDate, DueDate: r.DueDate, Currency: r.Currency, TotalAmount: r.TotalAmount.StringFixed(8), SettledAmount: r.SettledAmount.StringFixed(8), Balance: r.Balance.StringFixed(8), BaseCurrency: r.BaseCurrency, BaseBalance: r.BaseBalance.StringFixed(8), Bucket: r.Bucket, OverdueDays: days})
	}
	return out
}
func agingReportAmountsToAPI(items []biz.AgingReportBucketAmount) []*v1.AgingReportBucketAmount {
	out := make([]*v1.AgingReportBucketAmount, 0, len(items))
	for _, r := range items {
		out = append(out, &v1.AgingReportBucketAmount{Direction: r.Direction, Currency: r.Currency, NotDue: r.NotDue.StringFixed(8), Days_1_30: r.Days130.StringFixed(8), Days_31_60: r.Days3160.StringFixed(8), Days_61_90: r.Days6190.StringFixed(8), Over_90: r.Over90.StringFixed(8), Undated: r.Undated.StringFixed(8), Total: r.Total.StringFixed(8), BillCount: r.BillCount})
	}
	return out
}
