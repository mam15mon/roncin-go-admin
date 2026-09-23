package data

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebillbatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillbatch"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

func (r *financeBillRepo) GetBatchByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.FinanceBillBatch, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := r.financeBillBatchQuery(client.FinanceBillBatch.Query()).Where(financebillbatchent.OrganizationIDEQ(organizationID), financebillbatchent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return financeBillBatchToBiz(item)
}

func (r *financeBillRepo) financeBillBatchQuery(query *ent.FinanceBillBatchQuery) *ent.FinanceBillBatchQuery {
	return query.WithBills(func(query *ent.FinanceBillQuery) {
		query.WithOrganization().WithLines(func(lineQuery *ent.FinanceBillLineQuery) {
			lineQuery.WithOrder().Order(financebilllineent.ByCreatedAt(), financebilllineent.ByID())
		}).Order(financebillent.ByCreatedAt(), financebillent.ByID())
	}).WithNettings(func(nettingQuery *ent.FinanceNettingQuery) {
		nettingQuery.WithAllocations(func(allocationQuery *ent.FinanceNettingAllocationQuery) {
			allocationQuery.Order(financenettingallocationent.ByCreatedAt(), financenettingallocationent.ByID())
		}).Order(financenettingent.ByCreatedAt(), financenettingent.ByID())
	})
}

func (r *financeBillRepo) GetBatch(ctx context.Context, organizationIDs []uuid.UUID, batchID uuid.UUID) (*biz.FinanceBillBatch, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := r.financeBillBatchQuery(client.FinanceBillBatch.Query()).Where(financebillbatchent.IDEQ(batchID), financebillbatchent.OrganizationIDIn(organizationIDs...)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceBillNotFound, nil)
	}
	return financeBillBatchToBiz(item)
}

func (r *financeBillRepo) ConfirmBatch(ctx context.Context, organizationIDs []uuid.UUID, batchID, actorID uuid.UUID, expectedVersions map[uuid.UUID]uint64, audit *biz.AuditEvent) (*biz.FinanceBillBatch, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		batch, err := tx.FinanceBillBatch.Query().Where(financebillbatchent.IDEQ(batchID), financebillbatchent.OrganizationIDIn(organizationIDs...)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		bills, err := tx.FinanceBill.Query().Where(financebillent.BatchIDEQ(batchID)).Order(financebillent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(bills) == 0 || len(bills) != len(expectedVersions) {
			return biz.ErrFinanceBillBatchMismatch
		}
		now := time.Now().UTC()
		for _, bill := range bills {
			if bill.OrganizationID != batch.OrganizationID {
				return biz.ErrFinanceBillBatchMismatch
			}
			expected, exists := expectedVersions[bill.ID]
			if !exists {
				return biz.ErrFinanceBillBatchMismatch
			}
			if bill.Version != expected {
				return biz.ErrFinanceBillVersionConflict
			}
			if bill.Status != financebillent.StatusDRAFT {
				return biz.ErrFinanceBillInvalidTransition
			}
			if _, err = tx.FinanceBill.UpdateOneID(bill.ID).SetStatus(financebillent.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actorID).SetVersion(bill.Version + 1).Save(ctx); err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetBatch(ctx, organizationIDs, batchID)
}

func (r *financeBillRepo) CreateBatch(ctx context.Context, batch *biz.FinanceBillBatch, _ string, audit *biz.AuditEvent, nettingAudits []*biz.AuditEvent) (*biz.FinanceBillBatch, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		feeIDs := make([]uuid.UUID, 0, batch.FeeCount)
		expectedLines := make(map[uuid.UUID]*biz.FinanceBillLine, batch.FeeCount)
		for _, bill := range batch.Bills {
			for _, line := range bill.Lines {
				feeIDs = append(feeIDs, line.OrderFeeID)
				expectedLines[line.OrderFeeID] = line
			}
		}
		if len(feeIDs) != batch.FeeCount || len(expectedLines) != batch.FeeCount || len(batch.Bills) != batch.BillCount {
			return biz.ErrFinanceBillBatchMismatch
		}
		sort.Slice(feeIDs, func(i, j int) bool { return feeIDs[i].String() < feeIDs[j].String() })
		fees, err := tx.OrderFee.Query().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(batch.OrganizationID))).WithSettlementParty().WithOrder().Order(orderfeeent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(fees) != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		for _, fee := range fees {
			line := expectedLines[fee.ID]
			if line == nil || fee.Status != orderfeeent.StatusUNBILLED || fee.Currency != line.Currency || fee.BaseCurrency != line.BaseCurrency || fee.TotalAmount != line.TotalAmount.StringFixed(8) || fee.NetAmount != line.NetAmount.StringFixed(8) || fee.TaxAmount != line.TaxAmount.StringFixed(8) || !financeDecimalStringEqual(fee.TaxRate, line.TaxRate, 4) {
				return biz.ErrFinanceBillPreviewStale
			}
		}
		active, err := tx.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDIn(feeIDs...), financebilllineent.ActiveEQ(true)).Exist(ctx)
		if err != nil {
			return err
		}
		if active {
			return biz.ErrFinanceBillFeeInvalid
		}
		// 与单张建账保持“费用 → 账户”的固定加锁顺序，避免批量与单张并发建账互相等待。
		if err := hydrateFinanceBillSettlementAccounts(ctx, tx, batch.Bills); err != nil {
			return err
		}
		now := time.Now().UTC()
		batchRule, batchSequence, err := allocateNumberInTx(ctx, tx, batch.OrganizationID, biz.DocumentTypeBillBatch, now)
		if err != nil {
			return err
		}
		batch.BatchNo, err = biz.FormatAllocatedNumber(now, batchRule, batchSequence, "")
		if err != nil {
			return err
		}
		_, err = tx.FinanceBillBatch.Create().SetID(batch.ID).SetOrganizationID(batch.OrganizationID).SetBatchNo(batch.BatchNo).SetIdempotencyKey(batch.IdempotencyKey).SetRequestHash(batch.RequestHash).SetSplitByOrder(batch.GroupingPolicy.SplitByOrder).SetSplitByTaxRate(batch.GroupingPolicy.SplitByTaxRate).SetGroupingMode(financebillbatchent.GroupingMode(batch.GroupingPolicy.Mode)).SetFeeCount(batch.FeeCount).SetBillCount(batch.BillCount).SetTotalBaseAmount(batch.TotalBaseAmount.StringFixed(8)).SetBaseCurrency(batch.BaseCurrency).SetCreatedBy(batch.CreatedBy).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrFinanceBillBatchConflict)
		}
		for _, bill := range batch.Bills {
			billRule, billSequence, allocateErr := allocateNumberInTx(ctx, tx, batch.OrganizationID, biz.DocumentTypeBill, now)
			if allocateErr != nil {
				return allocateErr
			}
			bill.BillNo, allocateErr = biz.FormatAllocatedNumber(now, billRule, billSequence, "")
			if allocateErr != nil {
				return allocateErr
			}
			bill.BatchNo = batch.BatchNo
			_, saveErr := tx.FinanceBill.Create().SetID(bill.ID).SetOrganizationID(batch.OrganizationID).SetBatchID(batch.ID).SetBillNo(bill.BillNo).SetIdempotencyKey(bill.IdempotencyKey).SetDirection(financebillent.Direction(bill.Direction)).SetStatus(financebillent.StatusDRAFT).SetSettlementPartyID(bill.SettlementPartyID).SetSettlementPartyName(bill.SettlementPartyName).SetSettlementAccountID(bill.SettlementAccountID).SetSettlementAccountName(bill.SettlementAccountName).SetSettlementAccountHolder(bill.SettlementAccountHolder).SetSettlementBankName(bill.SettlementBankName).SetSettlementBankAccount(bill.SettlementBankAccount).SetSettlementAccountCurrency(bill.SettlementAccountCurrency).SetSettlementSwiftCode(bill.SettlementSwiftCode).SetNillableEstimatedInvoiceCurrency(bill.EstimatedInvoiceCurrency).SetNillableEstimatedInvoiceRate(financeDecimalString(bill.EstimatedInvoiceRate, 8)).SetNillableEstimatedInvoiceAmount(financeDecimalString(bill.EstimatedInvoiceAmount, 8)).SetCurrency(bill.Currency).SetBaseCurrency(bill.BaseCurrency).SetExchangeRate(bill.ExchangeRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(bill.ExchangeRateSource)).SetExchangeRateDate(bill.ExchangeRateDate).SetNillableExchangeRateSettingID(bill.ExchangeRateSettingID).SetTotalAmount(bill.TotalAmount.StringFixed(8)).SetNetAmount(bill.NetAmount.StringFixed(8)).SetTaxAmount(bill.TaxAmount.StringFixed(8)).SetBaseCurrencyAmount(bill.BaseCurrencyAmount.StringFixed(8)).SetFeeCount(bill.FeeCount).SetBillDate(bill.BillDate).SetNillableStatementTitle(bill.StatementTitle).SetNillablePaymentTermsDays(bill.PaymentTermsDays).SetNillableDueDate(bill.DueDate).SetNillableNote(bill.Note).SetVersion(1).Save(ctx)
			if saveErr != nil {
				return saveErr
			}
			lineBuilders := make([]*ent.FinanceBillLineCreate, 0, len(bill.Lines))
			for _, line := range bill.Lines {
				lineBuilders = append(lineBuilders, tx.FinanceBillLine.Create().SetID(line.ID).SetBillID(bill.ID).SetOrderFeeID(line.OrderFeeID).SetOrderID(line.OrderID).SetOrderNo(line.OrderNo).SetFeeCode(line.FeeCode).SetFeeName(line.FeeName).SetQuantity(line.Quantity.StringFixed(4)).SetUnitPrice(line.UnitPrice.StringFixed(4)).SetTotalAmount(line.TotalAmount.StringFixed(8)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetNillableTaxRate(financeDecimalString(line.TaxRate, 4)).SetCurrency(line.Currency).SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrency(line.BaseCurrency).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8)).SetActive(true))
			}
			if _, saveErr = tx.FinanceBillLine.CreateBulk(lineBuilders...).Save(ctx); saveErr != nil {
				return mapEntError(saveErr, nil, biz.ErrFinanceBillFeeInvalid)
			}
		}
		affected, err := tx.OrderFee.Update().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.StatusEQ(orderfeeent.StatusUNBILLED)).SetStatus(orderfeeent.StatusBILLED).AddVersion(1).Save(ctx)
		if err != nil {
			return err
		}
		if affected != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		billNos := make(map[uuid.UUID]string, len(batch.Bills))
		for _, bill := range batch.Bills {
			billNos[bill.ID] = bill.BillNo
		}
		for index, netting := range batch.Nettings {
			nettingRule, nettingSequence, nettingErr := allocateNumberInTx(ctx, tx, batch.OrganizationID, biz.DocumentTypeNetting, now)
			if nettingErr != nil {
				return nettingErr
			}
			netting.NettingNo, nettingErr = biz.FormatAllocatedNumber(now, nettingRule, nettingSequence, "")
			if nettingErr != nil {
				return nettingErr
			}
			if _, nettingErr = tx.FinanceNetting.Create().
				SetID(netting.ID).SetOrganizationID(batch.OrganizationID).SetNettingNo(netting.NettingNo).
				SetIdempotencyKey(netting.IdempotencyKey).SetRequestHash(batch.RequestHash).
				SetBatchID(batch.ID).SetStatus(financenettingent.StatusDRAFT).
				SetSettlementPartyID(netting.SettlementPartyID).SetSettlementPartyName(netting.SettlementPartyName).
				SetCurrency(netting.Currency).SetAmount(netting.Amount.StringFixed(8)).
				SetBaseCurrency(netting.BaseCurrency).SetBaseCurrencyAmount(netting.BaseCurrencyAmount.StringFixed(8)).
				SetPayableBaseAmount(netting.PayableBaseAmount.StringFixed(8)).
				SetExchangeGainLoss(netting.ExchangeGainLoss.StringFixed(8)).
				SetNillableNote(netting.Note).SetVersion(1).Save(ctx); nettingErr != nil {
				return mapEntConstraint(nettingErr, "financenetting_organization_id_idempotency_key", biz.ErrFinanceNettingIdempotency)
			}
			allocationBuilders := make([]*ent.FinanceNettingAllocationCreate, 0, len(netting.Allocations))
			for _, allocation := range netting.Allocations {
				// 账单编号在仓储事务内分配；分摊快照在此固化同事务内的最终编号。
				allocation.BillNo = billNos[allocation.BillID]
				allocationBuilders = append(allocationBuilders, tx.FinanceNettingAllocation.Create().
					SetID(allocation.ID).SetNettingID(netting.ID).SetBillID(allocation.BillID).SetBillNo(allocation.BillNo).
					SetDirection(financenettingallocationent.Direction(allocation.Direction)).
					SetAmount(allocation.Amount.StringFixed(8)).SetBaseCurrencyAmount(allocation.BaseCurrencyAmount.StringFixed(8)).
					SetActive(false))
			}
			if _, nettingErr = tx.FinanceNettingAllocation.CreateBulk(allocationBuilders...).Save(ctx); nettingErr != nil {
				return mapEntConstraint(nettingErr, "netting_allocation_pair_unique", biz.ErrFinanceNettingInvalid)
			}
			if index < len(nettingAudits) {
				if auditErr := writeAudit(ctx, tx.AuditLog, nettingAudits[index]); auditErr != nil {
					return auditErr
				}
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.GetBatchByIdempotencyKey(ctx, batch.OrganizationID, batch.IdempotencyKey)
}

func financeBillBatchToBiz(item *ent.FinanceBillBatch) (*biz.FinanceBillBatch, error) {
	totalBaseAmount, err := decimalOf(item.TotalBaseAmount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceBillBatch{ID: item.ID, OrganizationID: item.OrganizationID, CreatedBy: item.CreatedBy, BatchNo: item.BatchNo, IdempotencyKey: item.IdempotencyKey, RequestHash: item.RequestHash, GroupingPolicy: biz.FinanceBillGroupingPolicy{Mode: string(item.GroupingMode), SplitByOrder: item.SplitByOrder, SplitByTaxRate: item.SplitByTaxRate}, FeeCount: item.FeeCount, BillCount: item.BillCount, TotalBaseAmount: totalBaseAmount, BaseCurrency: item.BaseCurrency, Bills: make([]*biz.FinanceBill, 0, len(item.Edges.Bills)), Nettings: make([]*biz.FinanceNetting, 0, len(item.Edges.Nettings)), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, billItem := range item.Edges.Bills {
		bill, convertErr := financeBillToBiz(billItem)
		if convertErr != nil {
			return nil, convertErr
		}
		bill.BatchNo = item.BatchNo
		result.Bills = append(result.Bills, bill)
	}
	for _, nettingItem := range item.Edges.Nettings {
		netting, convertErr := financeNettingToBiz(nettingItem)
		if convertErr != nil {
			return nil, convertErr
		}
		netting.BatchNo = item.BatchNo
		result.Nettings = append(result.Nettings, netting)
	}
	return result, nil
}
