package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FinanceReportFilter 仅持有调用方已提交筛选，组织边界由独立参数给出。
type FinanceReportFilter struct {
	Page, PageSize                                     int
	DateFrom, DateTo, Keyword, BusinessType            string
	CustomerID, SalesID, OperatorID, CustomerServiceID *uuid.UUID
	Direction                                          OrderFeeDirection
	SettlementPartyID                                  *uuid.UUID
	Currency, PaymentMethod, OurAccount, Bucket        string
}

const FinanceReportExportLimit = 10000

type ProfitReportRow struct {
	ID, OrderNo, BusinessType, CustomerName, SalesName, OperatorName, CustomerServiceName, BaseCurrency string
	OrderID, CustomerID                                                                                 uuid.UUID
	BusinessDate                                                                                        *string
	ReceivableAmount, PayableAmount, ProfitAmount                                                       decimal.Decimal
	NetReceivableAmount, NetPayableAmount, NetProfitAmount                                              decimal.Decimal
	FeeCount                                                                                            int64
}
type ProfitReportAmount struct {
	BaseCurrency                                           string
	ReceivableAmount, PayableAmount, ProfitAmount          decimal.Decimal
	NetReceivableAmount, NetPayableAmount, NetProfitAmount decimal.Decimal
}
type FinanceReportNativeAmount struct {
	Direction, Currency string
	Amount, NetAmount   decimal.Decimal
}
type ProfitReportMonth struct {
	Month   string
	Amounts []ProfitReportAmount
}
type ProfitReportResult struct {
	Items                                          []*ProfitReportRow
	Total, OrderCount, FeeCount, UndatedOrderCount int64
	Totals                                         []ProfitReportAmount
	NativeTotals                                   []FinanceReportNativeAmount
	Months                                         []ProfitReportMonth
	GeneratedAt                                    time.Time
}
type CashflowReportRow struct {
	ID                                                                     uuid.UUID
	FlowNo, Direction, TransactionDate                                     string
	SettlementPartyID                                                      uuid.UUID
	SettlementPartyName, Currency, BaseCurrency, OurAccount, PaymentMethod string
	Amount, BaseAmount                                                     decimal.Decimal
	BankReferenceNo                                                        *string
}
type FinanceReportCashAmount struct {
	Currency                                   string
	ReceivableAmount, PayableAmount, NetAmount decimal.Decimal
}
type CashflowReportResult struct {
	Items                    []*CashflowReportRow
	Total                    int64
	NativeTotals, BaseTotals []FinanceReportCashAmount
	GeneratedAt              time.Time
}
type AgingReportRow struct {
	ID                                               uuid.UUID
	BillNo, Direction, SettlementPartyName, BillDate string
	SettlementPartyID                                uuid.UUID
	DueDate                                          *string
	Currency, BaseCurrency, Bucket                   string
	TotalAmount, SettledAmount, Balance, BaseBalance decimal.Decimal
	OverdueDays                                      *int
}
type AgingReportBucketAmount struct {
	Direction, Currency                                         string
	NotDue, Days130, Days3160, Days6190, Over90, Undated, Total decimal.Decimal
	BillCount                                                   int64
}
type AgingReportResult struct {
	Items                    []*AgingReportRow
	Total                    int64
	NativeTotals, BaseTotals []AgingReportBucketAmount
	AsOfDate                 string
	GeneratedAt              time.Time
}
type FinanceReportOption struct {
	ID   uuid.UUID
	Name string
}
type FinanceReportRepo interface {
	Profit(context.Context, uuid.UUID, FinanceReportFilter, bool) (*ProfitReportResult, error)
	Cashflow(context.Context, uuid.UUID, FinanceReportFilter, bool) (*CashflowReportResult, error)
	Aging(context.Context, uuid.UUID, FinanceReportFilter, bool, time.Time) (*AgingReportResult, error)
	Options(context.Context, uuid.UUID, SelectorListOptions, bool) (*PagedList[*FinanceReportOption], error)
}
