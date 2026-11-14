package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *SettlementService) SetBillInvoiceExemption(ctx context.Context, request *v1.SetBillInvoiceExemptionRequest) (*v1.SetBillInvoiceExemptionResponse, error) {
	item, err := s.changeBillInvoiceExemption(ctx, request.GetId(), request.GetExpectedVersion(), request.GetReason(), false)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.SetBillInvoiceExemptionResponse{Data: financeBillToAPI(item)}), nil
}

func (s *SettlementService) RevokeBillInvoiceExemption(ctx context.Context, request *v1.RevokeBillInvoiceExemptionRequest) (*v1.RevokeBillInvoiceExemptionResponse, error) {
	item, err := s.changeBillInvoiceExemption(ctx, request.GetId(), request.GetExpectedVersion(), request.GetReason(), true)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RevokeBillInvoiceExemptionResponse{Data: financeBillToAPI(item)}), nil
}

func (s *SettlementService) changeBillInvoiceExemption(ctx context.Context, rawID string, expectedVersion uint64, reason string, revoke bool) (*biz.FinanceBill, error) {
	principal, id, err := financePrincipalAndID(ctx, rawID)
	if err != nil {
		return nil, err
	}
	organizationIDs, err := organizationIDsForPermission(principal, access.FinanceInvoiceExempt, true)
	if err != nil {
		return nil, err
	}
	if err := s.invoiceUsecase.SetBillInvoiceExemption(ctx, organizationIDs, principal.UserID, id, expectedVersion, reason, revoke); err != nil {
		return nil, err
	}
	return s.billUsecase.Get(ctx, organizationIDs, id)
}

func financeInvoiceExemptionsToAPI(items []*biz.FinanceInvoiceExemption) []*v1.FinanceInvoiceExemption {
	result := make([]*v1.FinanceInvoiceExemption, 0, len(items))
	for _, item := range items {
		result = append(result, &v1.FinanceInvoiceExemption{Id: item.ID.String(), Reason: item.Reason, MarkedBy: item.MarkedBy.String(), MarkedByName: item.MarkedByName, RevokedByName: item.RevokedByName, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), Version: item.Version, RevokedAt: financeTime(item.RevokedAt), RevokedBy: uuidStringPtr(item.RevokedBy), RevocationReason: item.RevocationReason})
	}
	return result
}
