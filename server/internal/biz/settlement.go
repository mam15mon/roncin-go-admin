package biz

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrFinanceLedgerInvalidArgument = errors.BadRequest("FINANCE_LEDGER_INVALID_ARGUMENT", "费用台账查询条件不合法")

var ledgerCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type FeeLedgerFinancialProgress string

const (
	FeeLedgerUnbilled                    FeeLedgerFinancialProgress = "UNBILLED"
	FeeLedgerUnverifiedUninvoiced        FeeLedgerFinancialProgress = "UNVERIFIED_UNINVOICED"
	FeeLedgerInvoicedUnverified          FeeLedgerFinancialProgress = "INVOICED_UNVERIFIED"
	FeeLedgerVerifiedUninvoiced          FeeLedgerFinancialProgress = "VERIFIED_UNINVOICED"
	FeeLedgerInvoicedPartiallyVerified   FeeLedgerFinancialProgress = "INVOICED_PARTIALLY_VERIFIED"
	FeeLedgerPartiallyVerifiedUninvoiced FeeLedgerFinancialProgress = "PARTIALLY_VERIFIED_UNINVOICED"
	FeeLedgerCompleted                   FeeLedgerFinancialProgress = "COMPLETED"
)

type FeeLedgerFilter struct {
	Page              int
	PageSize          int
	Keyword           string
	BusinessType      string
	Direction         OrderFeeDirection
	Status            OrderFeeStatus
	FinancialProgress FeeLedgerFinancialProgress
	SettlementPartyID *uuid.UUID
	CustomerID        *uuid.UUID
	Currency          string
	BillNo            string
	ExpenseDateFrom   string
	ExpenseDateTo     string
	FinanceLocked     *bool
	TagIDs            []uuid.UUID
}

type FeeLedgerItem struct {
	Fee               *OrderFee
	OrganizationID    uuid.UUID
	OrganizationName  string
	OrderNo           string
	Business          string
	CustomerID        uuid.UUID
	CustomerName      string
	FinancialProgress FeeLedgerFinancialProgress
	BillNo            string
	FinanceLocked     bool
}

// ResolveFeeLedgerFinancialProgress 依据账单事实解析费用台账的财务进度。
// settledAmount 是「有效结清金额」：有效核销分摊（allocation active 且核销单 ACTIVE）
// 加有效对冲分摊（allocation active 且对冲单 CONFIRMED）；对外状态枚举保持既有契约不变。
func ResolveFeeLedgerFinancialProgress(hasBill, invoiced bool, billAmount, settledAmount decimal.Decimal) FeeLedgerFinancialProgress {
	if !hasBill {
		return FeeLedgerUnbilled
	}
	if settledAmount.LessThanOrEqual(decimal.Zero) {
		if invoiced {
			return FeeLedgerInvoicedUnverified
		}
		return FeeLedgerUnverifiedUninvoiced
	}
	if settledAmount.LessThan(billAmount) {
		if invoiced {
			return FeeLedgerInvoicedPartiallyVerified
		}
		return FeeLedgerPartiallyVerifiedUninvoiced
	}
	if invoiced {
		return FeeLedgerCompleted
	}
	return FeeLedgerVerifiedUninvoiced
}

type FeeLedgerSummary struct {
	ActiveCount           int64
	AmountsByBaseCurrency []FeeLedgerBaseCurrencyAmount
}

type FeeLedgerBaseCurrencyAmount struct {
	BaseCurrency                                              string
	ReceivableBaseAmount, PayableBaseAmount, ProfitBaseAmount decimal.Decimal
}

type FeeLedgerResult struct {
	Items   []*FeeLedgerItem
	Total   int64
	Summary FeeLedgerSummary
}

type FeeLedgerOrderDetail struct {
	OrderID, OrderNo, Business, CustomerName string
	OrganizationID                           uuid.UUID
	OrganizationName                         string
	Items                                    []*FeeLedgerItem
	AmountsByBaseCurrency                    []FeeLedgerBaseCurrencyAmount
}

type SettlementRepo interface {
	ListFeeLedger(ctx context.Context, organizationIDs []uuid.UUID, filter FeeLedgerFilter) (*FeeLedgerResult, error)
	GetFeeLedgerOrderDetail(context.Context, []uuid.UUID, uuid.UUID) (*FeeLedgerOrderDetail, error)
	ResolveFeeLedgerOrganization(ctx context.Context, organizationIDs, feeIDs []uuid.UUID) (uuid.UUID, error)
	ListFinanceOrganizations(context.Context, []uuid.UUID, string) ([]*FinanceOrganizationOption, error)
	ListFinanceSettlementParties(context.Context, uuid.UUID, string, int, int) ([]*FinanceSettlementPartyOption, int64, error)
}

func (uc *SettlementUsecase) ResolveFeeLedgerOrganization(ctx context.Context, organizationIDs, feeIDs []uuid.UUID) (uuid.UUID, error) {
	if !validFinanceOrganizationIDs(organizationIDs) || len(feeIDs) == 0 {
		return uuid.Nil, ErrFinanceLedgerInvalidArgument
	}
	return uc.repo.ResolveFeeLedgerOrganization(ctx, organizationIDs, feeIDs)
}

func (uc *SettlementUsecase) GetFeeLedgerOrderDetail(ctx context.Context, organizationIDs []uuid.UUID, orderID uuid.UUID) (*FeeLedgerOrderDetail, error) {
	if !validFinanceOrganizationIDs(organizationIDs) || orderID == uuid.Nil {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	return uc.repo.GetFeeLedgerOrderDetail(ctx, organizationIDs, orderID)
}

type FinanceOrganizationOption struct {
	ID                       uuid.UUID
	Code, Name, BaseCurrency string
}

type FinanceSettlementPartyOption struct {
	ID, Code, Name string
	IsCasual       bool
	// CreditExceeded 表示该往来户已配置信用额度且折本币未核销应收总额超出额度；
	// 前端仅在直接干预模式下据此置灰禁用，标签文本不得拼进 label。
	CreditExceeded bool
}

type SettlementUsecase struct {
	repo SettlementRepo
}

func (uc *SettlementUsecase) ListFinanceOrganizations(ctx context.Context, organizationIDs []uuid.UUID, keyword string) ([]*FinanceOrganizationOption, error) {
	if len(organizationIDs) == 0 || utf8.RuneCountInString(strings.TrimSpace(keyword)) > 100 {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	seen := make(map[uuid.UUID]struct{}, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if organizationID == uuid.Nil {
			return nil, ErrFinanceLedgerInvalidArgument
		}
		if _, exists := seen[organizationID]; exists {
			return nil, ErrFinanceLedgerInvalidArgument
		}
		seen[organizationID] = struct{}{}
	}
	return uc.repo.ListFinanceOrganizations(ctx, organizationIDs, strings.TrimSpace(keyword))
}

func (uc *SettlementUsecase) ListFinanceSettlementParties(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*FinanceSettlementPartyOption, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if organizationID == uuid.Nil || !ValidListPagination(page, pageSize) || utf8.RuneCountInString(keyword) > 100 {
		return nil, 0, ErrFinanceLedgerInvalidArgument
	}
	return uc.repo.ListFinanceSettlementParties(ctx, organizationID, keyword, page, pageSize)
}

func NewSettlementUsecase(repo SettlementRepo) *SettlementUsecase {
	return &SettlementUsecase{repo: repo}
}

func (uc *SettlementUsecase) ListFeeLedger(ctx context.Context, organizationIDs []uuid.UUID, filter FeeLedgerFilter) (*FeeLedgerResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.BusinessType = strings.ToUpper(strings.TrimSpace(filter.BusinessType))
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	filter.BillNo = strings.TrimSpace(filter.BillNo)
	filter.FinancialProgress = FeeLedgerFinancialProgress(strings.ToUpper(strings.TrimSpace(string(filter.FinancialProgress))))
	if !validFinanceOrganizationIDs(organizationIDs) || !ValidListPagination(filter.Page, filter.PageSize) || len([]rune(filter.Keyword)) > 100 || len([]rune(filter.BillNo)) > 64 {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	if filter.CustomerID != nil && *filter.CustomerID == uuid.Nil {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	if filter.BusinessType != "" && filter.BusinessType != "SE" && filter.BusinessType != "SI" && filter.BusinessType != "AE" && filter.BusinessType != "AI" && filter.BusinessType != "LAND" && filter.BusinessType != "RAIL" {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	if filter.Direction != "" && filter.Direction != OrderFeeReceivable && filter.Direction != OrderFeePayable {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	if filter.Status != "" && filter.Status != OrderFeeUnbilled && filter.Status != OrderFeeBilled && filter.Status != OrderFeeCancelled {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	if filter.FinancialProgress != "" && !IsFeeLedgerFinancialProgress(filter.FinancialProgress) {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	if filter.Currency != "" && !ledgerCurrencyPattern.MatchString(filter.Currency) {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	for _, date := range []string{filter.ExpenseDateFrom, filter.ExpenseDateTo} {
		if date == "" {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", date); err != nil || parsed.Format("2006-01-02") != date {
			return nil, ErrFinanceLedgerInvalidArgument
		}
	}
	if filter.ExpenseDateFrom != "" && filter.ExpenseDateTo != "" && filter.ExpenseDateFrom > filter.ExpenseDateTo {
		return nil, ErrFinanceLedgerInvalidArgument
	}
	return uc.repo.ListFeeLedger(ctx, organizationIDs, filter)
}

// IsFeeLedgerFinancialProgress 判断费用台账的账单、开票与核销综合进度是否合法。
func IsFeeLedgerFinancialProgress(value FeeLedgerFinancialProgress) bool {
	switch value {
	case FeeLedgerUnbilled,
		FeeLedgerUnverifiedUninvoiced,
		FeeLedgerInvoicedUnverified,
		FeeLedgerVerifiedUninvoiced,
		FeeLedgerInvoicedPartiallyVerified,
		FeeLedgerPartiallyVerifiedUninvoiced,
		FeeLedgerCompleted:
		return true
	default:
		return false
	}
}
