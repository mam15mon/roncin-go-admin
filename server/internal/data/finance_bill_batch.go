package data

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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
			lineQuery.WithOrder().WithOrderFee().Order(financebilllineent.ByCreatedAt(), financebilllineent.ByID())
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

// CreateBatchHead 以独立短事务分配批次号并写入批次头；同组织同幂等键唯一冲突映射为批次冲突。
func (r *financeBillRepo) CreateBatchHead(ctx context.Context, batch *biz.FinanceBillBatch, audit *biz.AuditEvent) (*biz.FinanceBillBatch, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		return createFinanceBillBatchHead(ctx, tx, batch, time.Now().UTC(), audit)
	}); err != nil {
		return nil, err
	}
	return r.GetBatchByIdempotencyKey(ctx, batch.OrganizationID, batch.IdempotencyKey)
}

func createFinanceBillBatchHead(ctx context.Context, tx *ent.Tx, batch *biz.FinanceBillBatch, now time.Time, audit *biz.AuditEvent) error {
	batchRule, batchSequence, err := allocateNumberInTx(ctx, tx, batch.OrganizationID, biz.DocumentTypeBillBatch, now)
	if err != nil {
		return err
	}
	batch.BatchNo, err = biz.FormatAllocatedNumber(now, batchRule, batchSequence, "")
	if err != nil {
		return err
	}
	if _, err = tx.FinanceBillBatch.Create().SetID(batch.ID).SetOrganizationID(batch.OrganizationID).SetBatchNo(batch.BatchNo).SetIdempotencyKey(batch.IdempotencyKey).SetRequestHash(batch.RequestHash).SetSplitByOrder(batch.GroupingPolicy.SplitByOrder).SetSplitByTaxRate(batch.GroupingPolicy.SplitByTaxRate).SetGroupingMode(financebillbatchent.GroupingMode(batch.GroupingPolicy.Mode)).SetCreatedBy(batch.CreatedBy).Save(ctx); err != nil {
		return mapEntError(err, nil, biz.ErrFinanceBillBatchConflict)
	}
	return writeAudit(ctx, tx.AuditLog, audit)
}

// CreateBatchBills 在调用方事务内写入一组（普通模式）或整批（对冲模式）账单。
// 锁序与单张建账一致：订单闭包 → 费用 → 结算账户 → 编号；对冲模式批次头在账户锁之后、账单之前创建。
func (r *financeBillRepo) CreateBatchBills(ctx context.Context, write biz.FinanceBillBatchWrite) error {
	batch := write.Batch
	if batch == nil || write.ActorID == uuid.Nil || len(write.Bills) == 0 || len(write.BillAudits) != len(write.Bills) || (write.CreateHead && write.HeadAudit == nil) || (!write.CreateHead && batch.BatchNo == "") {
		return biz.ErrFinanceBillBatchMismatch
	}
	// 普通模式每个组事务只写一张账单，提成经济重算以该账单为触发标识，避免同批多组共用幂等键。
	if batch.GroupingPolicy.Mode == "NORMAL" && len(write.Bills) != 1 {
		return biz.ErrFinanceBillBatchMismatch
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		feeIDs := make([]uuid.UUID, 0)
		expectedLines := make(map[uuid.UUID]*biz.FinanceBillLine)
		for _, bill := range write.Bills {
			for _, line := range bill.Lines {
				feeIDs = append(feeIDs, line.OrderFeeID)
				expectedLines[line.OrderFeeID] = line
			}
		}
		if len(feeIDs) == 0 || len(expectedLines) != len(feeIDs) {
			return biz.ErrFinanceBillBatchMismatch
		}
		sort.Slice(feeIDs, func(i, j int) bool { return feeIDs[i].String() < feeIDs[j].String() })
		if err := lockBillFeeOrders(ctx, tx.Client(), []uuid.UUID{batch.OrganizationID}, feeIDs, true); err != nil {
			return err
		}
		fees, err := tx.OrderFee.Query().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(batch.OrganizationID))).WithSettlementParty().WithOrder().Order(orderfeeent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(fees) != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		for _, fee := range fees {
			line := expectedLines[fee.ID]
			if line == nil || !fee.AccountingRequired || fee.SupersededByRequestID != nil || fee.Currency != line.Currency || fee.BaseCurrency != line.BaseCurrency || fee.TotalAmount != line.TotalAmount.StringFixed(8) || fee.NetAmount != line.NetAmount.StringFixed(8) || fee.TaxAmount != line.TaxAmount.StringFixed(8) || !financeDecimalStringEqual(fee.TaxRate, line.TaxRate, 4) {
				return biz.ErrFinanceBillPreviewStale
			}
		}
		// 关联资格在插入账单行前锁内复核：任何目标费用已存在有效账单关联即拒绝，
		// 与预览期的占用口径一致。
		occupied, err := tx.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDIn(feeIDs...), effectiveBillLinePredicate()).Exist(ctx)
		if err != nil {
			return err
		}
		if occupied {
			return biz.ErrFinanceBillPreviewStale
		}
		// 与单张建账保持“费用 → 账户”的固定加锁顺序，避免批量与单张并发建账互相等待。
		if err := hydrateFinanceBillSettlementAccounts(ctx, tx, write.Bills); err != nil {
			return err
		}
		now := time.Now().UTC()
		if write.CreateHead {
			if err := createFinanceBillBatchHead(ctx, tx, batch, now, write.HeadAudit); err != nil {
				return err
			}
		}
		for index, bill := range write.Bills {
			billRule, billSequence, allocateErr := allocateNumberInTx(ctx, tx, batch.OrganizationID, biz.DocumentTypeBill, now)
			if allocateErr != nil {
				return allocateErr
			}
			bill.BillNo, allocateErr = biz.FormatAllocatedNumber(now, billRule, billSequence, "")
			if allocateErr != nil {
				return allocateErr
			}
			bill.BatchNo = batch.BatchNo
			_, saveErr := tx.FinanceBill.Create().SetID(bill.ID).SetOrganizationID(batch.OrganizationID).SetBatchID(batch.ID).SetBillNo(bill.BillNo).SetIdempotencyKey(bill.IdempotencyKey).SetPurpose(financebillent.Purpose(bill.Purpose)).SetDirection(financebillent.Direction(bill.Direction)).SetStatus(financebillent.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(write.ActorID).SetSettlementPartyID(bill.SettlementPartyID).SetSettlementPartyName(bill.SettlementPartyName).SetSettlementAccountID(bill.SettlementAccountID).SetSettlementAccountName(bill.SettlementAccountName).SetSettlementAccountHolder(bill.SettlementAccountHolder).SetSettlementBankName(bill.SettlementBankName).SetSettlementBankAccount(bill.SettlementBankAccount).SetSettlementAccountCurrency(bill.SettlementAccountCurrency).SetSettlementSwiftCode(bill.SettlementSwiftCode).SetNillableEstimatedInvoiceCurrency(bill.EstimatedInvoiceCurrency).SetNillableEstimatedInvoiceRate(financeDecimalString(bill.EstimatedInvoiceRate, 8)).SetNillableEstimatedInvoiceAmount(financeDecimalString(bill.EstimatedInvoiceAmount, 8)).SetCurrency(bill.Currency).SetBaseCurrency(bill.BaseCurrency).SetExchangeRate(bill.ExchangeRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(bill.ExchangeRateSource)).SetExchangeRateDate(bill.ExchangeRateDate).SetNillableExchangeRateSettingID(bill.ExchangeRateSettingID).SetTotalAmount(bill.TotalAmount.StringFixed(8)).SetNetAmount(bill.NetAmount.StringFixed(8)).SetTaxAmount(bill.TaxAmount.StringFixed(8)).SetBaseCurrencyAmount(bill.BaseCurrencyAmount.StringFixed(8)).SetFeeCount(bill.FeeCount).SetBillDate(bill.BillDate).SetNillableStatementTitle(bill.StatementTitle).SetNillablePaymentTermsDays(bill.PaymentTermsDays).SetNillableDueDate(bill.DueDate).SetNillableNote(bill.Note).SetVersion(1).Save(ctx)
			if saveErr != nil {
				return mapEntError(saveErr, nil, biz.ErrFinanceBillBatchConflict)
			}
			lineBuilders := make([]*ent.FinanceBillLineCreate, 0, len(bill.Lines))
			for _, line := range bill.Lines {
				lineBuilders = append(lineBuilders, tx.FinanceBillLine.Create().SetID(line.ID).SetBillID(bill.ID).SetOrderFeeID(line.OrderFeeID).SetOrderID(line.OrderID).SetOrderNo(line.OrderNo).SetFeeCode(line.FeeCode).SetFeeName(line.FeeName).SetQuantity(line.Quantity.StringFixed(4)).SetUnitPrice(line.UnitPrice.StringFixed(4)).SetTotalAmount(line.TotalAmount.StringFixed(8)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetNillableTaxRate(financeDecimalString(line.TaxRate, 4)).SetCurrency(line.Currency).SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrency(line.BaseCurrency).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8)).SetActive(true))
			}
			if _, saveErr = tx.FinanceBillLine.CreateBulk(lineBuilders...).Save(ctx); saveErr != nil {
				return mapEntError(saveErr, nil, biz.ErrFinanceBillFeeInvalid)
			}
			if auditErr := writeAudit(ctx, tx.AuditLog, write.BillAudits[index]); auditErr != nil {
				return auditErr
			}
		}
		// 费用行已在锁内通过关联资格复核，直接递增版本；受影响行数校验保留。
		affected, err := tx.OrderFee.Update().Where(orderfeeent.IDIn(feeIDs...)).AddVersion(1).Save(ctx)
		if err != nil {
			return err
		}
		if affected != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		billNos := make(map[uuid.UUID]string, len(write.Bills))
		for _, bill := range write.Bills {
			billNos[bill.ID] = bill.BillNo
		}
		for index, netting := range write.Nettings {
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
				SetNillableCorrectionRequestID(netting.CorrectionRequestID).SetBatchID(batch.ID).SetStatus(financenettingent.StatusDRAFT).
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
					SetNillableSourceFeeID(allocation.SourceFeeID).SetID(allocation.ID).SetNettingID(netting.ID).SetBillID(allocation.BillID).SetBillNo(allocation.BillNo).
					SetDirection(financenettingallocationent.Direction(allocation.Direction)).
					SetAmount(allocation.Amount.StringFixed(8)).SetBaseCurrencyAmount(allocation.BaseCurrencyAmount.StringFixed(8)).
					SetActive(false))
			}
			if _, nettingErr = tx.FinanceNettingAllocation.CreateBulk(allocationBuilders...).Save(ctx); nettingErr != nil {
				return mapEntConstraint(nettingErr, "netting_allocation_pair_unique", biz.ErrFinanceNettingInvalid)
			}
			if index < len(write.NettingAudits) {
				if auditErr := writeAudit(ctx, tx.AuditLog, write.NettingAudits[index]); auditErr != nil {
					return auditErr
				}
			}
		}
		var orderIDs []uuid.UUID
		for _, bill := range write.Bills {
			orderIDs = append(orderIDs, billEconomicOrderIDs(bill.Lines)...)
		}
		triggerID := batch.ID
		if batch.GroupingPolicy.Mode == "NORMAL" {
			triggerID = write.Bills[0].ID
		}
		return reconcileBillEconomics(ctx, tx.Client(), batch.OrganizationID, orderIDs, write.ActorID, biz.CommissionEconomicTrigger{Kind: "BILL_BATCH_CREATE", ID: triggerID, Version: 1})
	})
}

// financeBillBatchToBiz 转换批次头，并按本批次已生成账单聚合费用数、账单数与按本位币分组的合计。
func financeBillBatchToBiz(item *ent.FinanceBillBatch) (*biz.FinanceBillBatch, error) {
	result := &biz.FinanceBillBatch{ID: item.ID, OrganizationID: item.OrganizationID, CreatedBy: item.CreatedBy, BatchNo: item.BatchNo, IdempotencyKey: item.IdempotencyKey, RequestHash: item.RequestHash, GroupingPolicy: biz.FinanceBillGroupingPolicy{Mode: string(item.GroupingMode), SplitByOrder: item.SplitByOrder, SplitByTaxRate: item.SplitByTaxRate}, Bills: make([]*biz.FinanceBill, 0, len(item.Edges.Bills)), Nettings: make([]*biz.FinanceNetting, 0, len(item.Edges.Nettings)), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	baseTotals := make(map[string]decimal.Decimal)
	for _, billItem := range item.Edges.Bills {
		bill, convertErr := financeBillToBiz(billItem)
		if convertErr != nil {
			return nil, convertErr
		}
		bill.BatchNo = item.BatchNo
		result.Bills = append(result.Bills, bill)
		result.FeeCount += bill.FeeCount
		baseTotals[bill.BaseCurrency] = baseTotals[bill.BaseCurrency].Add(bill.BaseCurrencyAmount)
	}
	result.BillCount = len(result.Bills)
	currencies := make([]string, 0, len(baseTotals))
	for currency := range baseTotals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	for _, currency := range currencies {
		result.BaseTotals = append(result.BaseTotals, biz.FinanceBillBatchBaseTotal{BaseCurrency: currency, TotalBaseAmount: baseTotals[currency].RoundBank(8)})
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
