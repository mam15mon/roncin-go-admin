package data

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	invoiceheadent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	invoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correctionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
)

// FinancialFingerprint 锁定关联账单并绑定核销/对冲/票据事实。只向调用方返回哈希，
// 不将合并账单中其他订单的金额、名称和财务明细泄露给单票审批人。
func (r *orderFeeCorrectionRepo) FinancialFingerprint(ctx context.Context, request *biz.OrderFeeCorrectionRequest) (string, error) {
	if request.SourceFeeID == nil {
		return biz.FeeReviewHash([]byte("ADD")), nil
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return "", err
	}
	lines, err := client.FinanceBillLine.Query().Where(lineent.OrderFeeIDEQ(*request.SourceFeeID), lineent.ActiveEQ(true)).Order(lineent.ByID()).All(ctx)
	if err != nil {
		return "", err
	}
	ids := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		ids = append(ids, line.BillID)
	}
	bills, err := client.FinanceBill.Query().Where(billent.IDIn(ids...)).Order(billent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return "", err
	}
	verifications, err := client.FinanceVerificationAllocation.Query().Where(verificationent.BillIDIn(ids...)).WithVerification().Order(verificationent.ByID()).All(ctx)
	if err != nil {
		return "", err
	}
	nettings, err := client.FinanceNettingAllocation.Query().Where(nettingent.BillIDIn(ids...)).WithNetting().Order(nettingent.ByID()).All(ctx)
	if err != nil {
		return "", err
	}
	invoices, err := client.FinanceInvoiceBill.Query().Where(invoiceent.BillIDIn(ids...)).WithInvoice().Order(invoiceent.ByID()).All(ctx)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		Lines         []*ent.FinanceBillLine
		Bills         []*ent.FinanceBill
		Verifications []*ent.FinanceVerificationAllocation
		Nettings      []*ent.FinanceNettingAllocation
		Invoices      []*ent.FinanceInvoiceBill
	}{lines, bills, verifications, nettings, invoices})
	if err != nil {
		return "", err
	}
	return biz.FeeReviewHash(payload), nil
}

// decorateFinancialStatus 只评估本次更正链自己的账务与原票处理，不以整张合并账单余额推断目标费用。
func (r *orderFeeCorrectionRepo) decorateFinancialStatus(ctx context.Context, client *ent.Client, row *ent.OrderFeeCorrectionRequest, value *biz.OrderFeeCorrectionRequest) error {
	if row.Status != correctionent.StatusAPPROVED {
		return nil
	}
	value.FinancialStatus = "COMPLETED"
	feeIDs := []uuid.UUID{}
	billIDs := []uuid.UUID{}
	replacementBillIDs := []uuid.UUID{}
	unbilled := false
	for _, fee := range row.Edges.GeneratedFees {
		if fee.AccountingRequired {
			feeIDs = append(feeIDs, fee.ID)
			if fee.SupersededByRequestID == nil && len(fee.Edges.FinanceBillLines) == 0 {
				unbilled = true
			}
			for _, line := range fee.Edges.FinanceBillLines {
				billIDs = append(billIDs, line.BillID)
				if fee.EconomicEffect == feeent.EconomicEffectINCREASE && fee.SupersededByRequestID == nil {
					replacementBillIDs = append(replacementBillIDs, line.BillID)
				}
			}
		}
	}
	oldBillIDs := []uuid.UUID{}
	if row.SourceFeeID != nil {
		feeIDs = append(feeIDs, *row.SourceFeeID)
		oldLines, err := client.FinanceBillLine.Query().Where(lineent.OrderFeeIDEQ(*row.SourceFeeID), effectiveBillLinePredicate()).All(ctx)
		if err != nil {
			return err
		}
		for _, line := range oldLines {
			oldBillIDs = append(oldBillIDs, line.BillID)
			billIDs = append(billIDs, line.BillID)
		}
	}
	oldInvoice := false
	if row.DecidedAt != nil && len(oldBillIDs) > 0 {
		links, err := client.FinanceInvoiceBill.Query().Where(invoiceent.BillIDIn(oldBillIDs...), invoiceent.HasInvoiceWith(invoiceheadent.CreatedAtLTE(*row.DecidedAt))).WithInvoice().All(ctx)
		if err != nil {
			return err
		}
		for _, link := range links {
			covered := false
			for _, id := range link.SourceFeeIds {
				if row.SourceFeeID != nil && id == *row.SourceFeeID {
					covered = true
					break
				}
			}
			if !covered {
				continue
			}
			head := link.Edges.Invoice
			if head.Status == invoiceheadent.StatusCANCELLED {
				continue
			}
			oldInvoice = true
			if link.Active && head.Status != invoiceheadent.StatusRED_FLUSHED {
				value.FinancialStatus = "INVOICE_PENDING"
				return nil
			}
		}
	}
	if unbilled {
		value.FinancialStatus = "ADJUSTMENT_UNBILLED"
		return nil
	}
	projections, err := loadFinanceBillLineSettlements(ctx, client, uniqueSortedUUIDs(billIDs))
	if err != nil {
		return err
	}
	selected := map[uuid.UUID]bool{}
	for _, id := range feeIDs {
		selected[id] = true
	}
	for _, p := range projections {
		if selected[p.FeeID] && p.RemainingAmount.IsPositive() {
			value.FinancialStatus = "SETTLEMENT_PENDING"
			return nil
		}
	}
	if oldInvoice && len(replacementBillIDs) > 0 {
		exists, err := client.FinanceInvoiceBill.Query().Where(invoiceent.BillIDIn(replacementBillIDs...), invoiceent.ActiveEQ(true), invoiceent.HasInvoiceWith(invoiceheadent.StatusEQ(invoiceheadent.StatusISSUED))).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			value.FinancialStatus = "INVOICE_REISSUE_PENDING"
		}
	}
	return nil
}
