package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
)

type SettlementService struct {
	v1.UnimplementedSettlementServiceServer
	usecase *biz.SettlementUsecase
}

func NewSettlementService(usecase *biz.SettlementUsecase) *SettlementService {
	return &SettlementService{usecase: usecase}
}

func (s *SettlementService) ListFeeLedger(ctx context.Context, request *v1.ListFeeLedgerRequest) (*v1.ListFeeLedgerResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	filter := biz.FeeLedgerFilter{Page: int(request.GetPage()), PageSize: int(request.GetPageSize())}
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	filter.Keyword = financeOptionalString(request.Keyword)
	filter.BusinessType = financeOptionalString(request.BusinessType)
	filter.Direction = biz.OrderFeeDirection(strings.ToUpper(financeOptionalString(request.Direction)))
	filter.Status = biz.OrderFeeStatus(strings.ToUpper(financeOptionalString(request.Status)))
	filter.Currency = financeOptionalString(request.Currency)
	filter.ExpenseDateFrom = financeOptionalString(request.ExpenseDateFrom)
	filter.ExpenseDateTo = financeOptionalString(request.ExpenseDateTo)
	if request.SettlementPartyId != nil && strings.TrimSpace(*request.SettlementPartyId) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*request.SettlementPartyId))
		if err != nil {
			return nil, biz.ErrFinanceLedgerInvalidArgument
		}
		filter.SettlementPartyID = &id
	}
	result, err := s.usecase.ListFeeLedger(ctx, principal.Organization.ID, filter)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FeeLedgerItem, 0, len(result.Items))
	for _, item := range result.Items {
		fee := item.Fee
		data = append(data, &v1.FeeLedgerItem{
			Id: fee.ID.String(), OrderId: fee.OrderID.String(), OrderNo: item.OrderNo, BusinessType: item.Business,
			Direction: string(fee.Direction), Status: string(fee.Status), FeeCode: fee.FeeCode, FeeName: fee.FeeName,
			SettlementPartyId: fee.SettlementPartyID.String(), SettlementPartyName: fee.SettlementPartyName, BillingUnit: fee.BillingUnit,
			Quantity: fee.Quantity.StringFixed(4), UnitPrice: fee.UnitPrice.StringFixed(4), TotalAmount: fee.TotalAmount.StringFixed(8),
			NetAmount: fee.NetAmount.StringFixed(8), TaxAmount: fee.TaxAmount.StringFixed(8), Currency: fee.Currency,
			ExchangeRate: fee.ExchangeRate.StringFixed(8), BaseCurrency: fee.BaseCurrency, BaseCurrencyAmount: fee.BaseCurrencyAmount.StringFixed(8),
			ExpenseDate: fee.ExpenseDate, Note: fee.Note, Version: fee.Version,
			CreatedAt: fee.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: fee.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return &v1.ListFeeLedgerResponse{
		Success: true, Code: 0, Message: "OK", Data: data, Total: result.Total, TraceId: requestmeta.TraceID(ctx),
		Summary: &v1.FeeLedgerSummary{ActiveCount: result.Summary.ActiveCount, ReceivableBaseAmount: result.Summary.ReceivableBaseAmount.StringFixed(8), PayableBaseAmount: result.Summary.PayableBaseAmount.StringFixed(8), ProfitBaseAmount: result.Summary.ProfitBaseAmount.StringFixed(8), BaseCurrency: result.Summary.BaseCurrency},
	}, nil
}

func financeOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

var _ v1.SettlementServiceServer = (*SettlementService)(nil)
