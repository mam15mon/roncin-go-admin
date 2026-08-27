package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrFinanceLedgerInvalidArgument = errors.BadRequest("FINANCE_LEDGER_INVALID_ARGUMENT", "费用台账查询条件不合法")

var ledgerCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

type FeeLedgerFilter struct {
	Page              int
	PageSize          int
	Keyword           string
	BusinessType      string
	Direction         OrderFeeDirection
	Status            OrderFeeStatus
	SettlementPartyID *uuid.UUID
	CustomerID        *uuid.UUID
	Currency          string
	ExpenseDateFrom   string
	ExpenseDateTo     string
}

type FeeLedgerItem struct {
	Fee          *OrderFee
	OrderNo      string
	Business     string
	CustomerID   uuid.UUID
	CustomerName string
}

type FeeLedgerSummary struct {
	ActiveCount          int64
	ReceivableBaseAmount decimal.Decimal
	PayableBaseAmount    decimal.Decimal
	ProfitBaseAmount     decimal.Decimal
	BaseCurrency         string
}

type FeeLedgerResult struct {
	Items   []*FeeLedgerItem
	Total   int64
	Summary FeeLedgerSummary
}

type SettlementRepo interface {
	ListFeeLedger(ctx context.Context, organizationID uuid.UUID, filter FeeLedgerFilter) (*FeeLedgerResult, error)
}

type SettlementUsecase struct {
	repo SettlementRepo
}

func NewSettlementUsecase(repo SettlementRepo) *SettlementUsecase {
	return &SettlementUsecase{repo: repo}
}

func (uc *SettlementUsecase) ListFeeLedger(ctx context.Context, organizationID uuid.UUID, filter FeeLedgerFilter) (*FeeLedgerResult, error) {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.BusinessType = strings.ToUpper(strings.TrimSpace(filter.BusinessType))
	filter.Currency = strings.ToUpper(strings.TrimSpace(filter.Currency))
	if organizationID == uuid.Nil || filter.Page < 1 || filter.PageSize < 1 || filter.PageSize > 200 || len([]rune(filter.Keyword)) > 100 {
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
	if filter.Status != "" && filter.Status != OrderFeeDraft && filter.Status != OrderFeeConfirmed && filter.Status != OrderFeeBilled && filter.Status != OrderFeeCancelled {
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
	return uc.repo.ListFeeLedger(ctx, organizationID, filter)
}
