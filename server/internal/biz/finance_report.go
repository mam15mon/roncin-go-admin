package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrFinanceReportInvalid     = errors.BadRequest("FINANCE_REPORT_INVALID_ARGUMENT", "报表筛选参数不合法")
	ErrFinanceReportExportLimit = errors.BadRequest("FINANCE_REPORT_EXPORT_LIMIT", "报表导出超过10000行，请缩小筛选范围")
)

type FinanceReportUsecase struct{ repo FinanceReportRepo }

func NewFinanceReportUsecase(repo FinanceReportRepo) *FinanceReportUsecase {
	return &FinanceReportUsecase{repo: repo}
}

func normalizeFinanceReportFilter(org uuid.UUID, f FinanceReportFilter, export bool) (FinanceReportFilter, error) {
	f.Keyword, f.BusinessType = strings.TrimSpace(f.Keyword), strings.ToUpper(strings.TrimSpace(f.BusinessType))
	f.DateFrom, f.DateTo = strings.TrimSpace(f.DateFrom), strings.TrimSpace(f.DateTo)
	f.Currency, f.Direction = strings.ToUpper(strings.TrimSpace(f.Currency)), OrderFeeDirection(strings.ToUpper(strings.TrimSpace(string(f.Direction))))
	f.PaymentMethod, f.OurAccount, f.Bucket = strings.TrimSpace(f.PaymentMethod), strings.TrimSpace(f.OurAccount), strings.ToUpper(strings.TrimSpace(f.Bucket))
	if org == uuid.Nil || (!export && !ValidListPagination(f.Page, f.PageSize)) || !validFinanceDateRange(f.DateFrom, f.DateTo) || utf8.RuneCountInString(f.Keyword) > 100 || utf8.RuneCountInString(f.PaymentMethod) > 50 || utf8.RuneCountInString(f.OurAccount) > 200 {
		return f, ErrFinanceReportInvalid
	}
	for _, id := range []*uuid.UUID{f.CustomerID, f.SalesID, f.OperatorID, f.CustomerServiceID, f.SettlementPartyID} {
		if id != nil && *id == uuid.Nil {
			return f, ErrFinanceReportInvalid
		}
	}
	if f.BusinessType != "" && f.BusinessType != "SE" && f.BusinessType != "SI" && f.BusinessType != "AE" && f.BusinessType != "AI" && f.BusinessType != "LAND" && f.BusinessType != "RAIL" {
		return f, ErrFinanceReportInvalid
	}
	if f.Direction != "" && f.Direction != OrderFeeReceivable && f.Direction != OrderFeePayable {
		return f, ErrFinanceReportInvalid
	}
	if f.Currency != "" && !financeBillCurrencyPattern.MatchString(f.Currency) {
		return f, ErrFinanceReportInvalid
	}
	if f.Bucket != "" && f.Bucket != "NOT_DUE" && f.Bucket != "DAYS_1_30" && f.Bucket != "DAYS_31_60" && f.Bucket != "DAYS_61_90" && f.Bucket != "OVER_90" && f.Bucket != "UNDATED" {
		return f, ErrFinanceReportInvalid
	}
	return f, nil
}
func (u *FinanceReportUsecase) Profit(ctx context.Context, org uuid.UUID, f FinanceReportFilter, export bool) (*ProfitReportResult, error) {
	f, err := normalizeFinanceReportFilter(org, f, export)
	if err != nil {
		return nil, err
	}
	return u.repo.Profit(ctx, org, f, export)
}
func (u *FinanceReportUsecase) Cashflow(ctx context.Context, org uuid.UUID, f FinanceReportFilter, export bool) (*CashflowReportResult, error) {
	f, err := normalizeFinanceReportFilter(org, f, export)
	if err != nil {
		return nil, err
	}
	return u.repo.Cashflow(ctx, org, f, export)
}
func (u *FinanceReportUsecase) Aging(ctx context.Context, org uuid.UUID, f FinanceReportFilter, export bool) (*AgingReportResult, error) {
	f, err := normalizeFinanceReportFilter(org, f, export)
	if err != nil {
		return nil, err
	}
	return u.repo.Aging(ctx, org, f, export, time.Now().In(ExchangeRateBusinessLocation()))
}
func (u *FinanceReportUsecase) Options(ctx context.Context, org uuid.UUID, options SelectorListOptions, personnel bool) (*PagedList[*FinanceReportOption], error) {
	options.Keyword = strings.TrimSpace(options.Keyword)
	if org == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) || utf8.RuneCountInString(options.Keyword) > 100 {
		return nil, ErrFinanceReportInvalid
	}
	return u.repo.Options(ctx, org, options, personnel)
}

// FinanceReportAgingBucket 按当前业务自然日分桶，空到期日独立展示，禁止回退到其他日期。
func FinanceReportAgingBucket(due *string, today time.Time) (string, *int, error) {
	if due == nil || strings.TrimSpace(*due) == "" {
		return "UNDATED", nil, nil
	}
	day, err := time.ParseInLocation("2006-01-02", *due, ExchangeRateBusinessLocation())
	if err != nil {
		return "", nil, ErrFinanceReportInvalid
	}
	current, err := time.ParseInLocation("2006-01-02", today.In(ExchangeRateBusinessLocation()).Format("2006-01-02"), ExchangeRateBusinessLocation())
	if err != nil {
		return "", nil, ErrFinanceReportInvalid
	}
	days := int(current.Sub(day).Hours() / 24)
	bucket := "NOT_DUE"
	switch {
	case days > 90:
		bucket = "OVER_90"
	case days > 60:
		bucket = "DAYS_61_90"
	case days > 30:
		bucket = "DAYS_31_60"
	case days > 0:
		bucket = "DAYS_1_30"
	}
	if days < 0 {
		days = 0
	}
	return bucket, &days, nil
}
