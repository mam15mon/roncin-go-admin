package data

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type financeNettingRepo struct{ data *Data }

func NewFinanceNettingRepo(data *Data) biz.FinanceNettingRepo { return &financeNettingRepo{data: data} }

type financeNettingSummaryRow struct {
	BaseCurrency string `json:"base_currency"`
	BaseAmount   string `json:"base_amount"`
}

func (r *financeNettingRepo) List(ctx context.Context, organizationIDs []uuid.UUID, filter biz.FinanceNettingFilter) (*biz.FinanceNettingListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceNetting{financenettingent.OrganizationIDIn(organizationIDs...)}
	if filter.Keyword != "" {
		predicates = append(predicates, financenettingent.Or(
			financenettingent.NettingNoContainsFold(filter.Keyword),
			financenettingent.SettlementPartyNameContainsFold(filter.Keyword),
		))
	}
	if filter.Status != "" {
		predicates = append(predicates, financenettingent.StatusEQ(financenettingent.Status(filter.Status)))
	}
	if filter.SettlementPartyID != nil {
		predicates = append(predicates, financenettingent.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.Currency != "" {
		predicates = append(predicates, financenettingent.CurrencyEQ(filter.Currency))
	}
	query := client.FinanceNetting.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	// 汇总口径只统计已确认对冲；草稿、已取消与已反转记录保留审计但不进入有效金额。
	summaryPredicates := []predicate.FinanceNetting{financenettingent.OrganizationIDIn(organizationIDs...), financenettingent.StatusEQ(financenettingent.StatusCONFIRMED)}
	if filter.Keyword != "" {
		summaryPredicates = append(summaryPredicates, financenettingent.Or(
			financenettingent.NettingNoContainsFold(filter.Keyword),
			financenettingent.SettlementPartyNameContainsFold(filter.Keyword),
		))
	}
	if filter.SettlementPartyID != nil {
		summaryPredicates = append(summaryPredicates, financenettingent.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.Currency != "" {
		summaryPredicates = append(summaryPredicates, financenettingent.CurrencyEQ(filter.Currency))
	}
	summaryQuery := client.FinanceNetting.Query().Where(summaryPredicates...)
	confirmedCount, err := summaryQuery.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	summaryRows := make([]financeNettingSummaryRow, 0)
	if err := summaryQuery.Clone().
		GroupBy(financenettingent.FieldBaseCurrency).
		Aggregate(ent.As(ent.Sum(financenettingent.FieldBaseCurrencyAmount), "base_amount")).
		Scan(ctx, &summaryRows); err != nil {
		return nil, err
	}
	amounts := make([]biz.FinanceNettingBaseCurrencyAmount, 0, len(summaryRows))
	for _, row := range summaryRows {
		value, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		amounts = append(amounts, biz.FinanceNettingBaseCurrencyAmount{BaseCurrency: row.BaseCurrency, NettingBaseAmount: value})
	}
	items, err := financeNettingQueryWithRelations(query).
		Order(financenettingent.ByCreatedAt(entsql.OrderDesc()), financenettingent.ByID(entsql.OrderDesc())).
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceNettingListResult{Items: make([]*biz.FinanceNetting, 0, len(items)), Total: int64(total), Summary: biz.FinanceNettingSummary{AmountsByBaseCurrency: amounts, ConfirmedCount: int64(confirmedCount)}}
	for _, item := range items {
		converted, convertErr := financeNettingToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

func financeNettingQueryWithRelations(query *ent.FinanceNettingQuery) *ent.FinanceNettingQuery {
	return query.WithOrganization().WithBatch().WithAllocations(func(allocationQuery *ent.FinanceNettingAllocationQuery) {
		allocationQuery.Order(financenettingallocationent.ByCreatedAt(), financenettingallocationent.ByID())
	})
}

func (r *financeNettingRepo) Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceNetting, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := financeNettingQueryWithRelations(client.FinanceNetting.Query()).
		Where(financenettingent.IDEQ(id), financenettingent.OrganizationIDIn(organizationIDs...)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceNettingNotFound, nil)
	}
	return financeNettingToBiz(item)
}

func (r *financeNettingRepo) GetByKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.FinanceNetting, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := financeNettingQueryWithRelations(client.FinanceNetting.Query()).
		Where(financenettingent.OrganizationIDEQ(organizationID), financenettingent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return financeNettingToBiz(item)
}

func (r *financeNettingRepo) LockNetting(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceNetting, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := financeNettingQueryWithRelations(client.FinanceNetting.Query()).
		Where(financenettingent.IDEQ(id), financenettingent.OrganizationIDIn(organizationIDs...))
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForUpdate()
	}
	item, err := query.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceNettingNotFound, nil)
	}
	return financeNettingToBiz(item)
}

// LoadPreview 按组织、结算单位和账单币种读取已确认账单的双向余额事实；预览只读，不加锁。
func (r *financeNettingRepo) LoadPreview(ctx context.Context, organizationID, settlementPartyID uuid.UUID, currency string) (*biz.FinanceNettingPreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	organization, err := client.Organization.Query().Where(organizationent.IDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceNettingNotFound, nil)
	}
	party, err := client.Partner.Query().Where(partnerent.IDEQ(settlementPartyID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceNettingMismatch, nil)
	}
	bills, err := client.FinanceBill.Query().
		Where(
			financebillent.OrganizationIDEQ(organizationID),
			financebillent.SettlementPartyIDEQ(settlementPartyID),
			financebillent.CurrencyEQ(currency),
			financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		).Order(financebillent.ByID()).
		Limit(biz.MaxFinanceNettingBills + 1).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(bills) > biz.MaxFinanceNettingBills {
		return nil, biz.ErrFinanceNettingTooManyBills
	}
	billIDs := make([]uuid.UUID, 0, len(bills))
	for _, bill := range bills {
		billIDs = append(billIDs, bill.ID)
	}
	verifiedSums, err := loadFinanceBillVerifiedAmounts(ctx, client, billIDs)
	if err != nil {
		return nil, err
	}
	nettedSums, err := loadFinanceBillNettedAmounts(ctx, client, billIDs)
	if err != nil {
		return nil, err
	}
	preview := &biz.FinanceNettingPreview{
		OrganizationID: organizationID, SettlementPartyID: settlementPartyID,
		OrganizationName: organization.Name, SettlementPartyName: party.LegalName, Currency: currency,
		ReceivableBills: make([]*biz.FinanceNettingBillBalance, 0), PayableBills: make([]*biz.FinanceNettingBillBalance, 0),
	}
	for _, bill := range bills {
		total, parseErr := decimalOf(bill.TotalAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		balance := &biz.FinanceNettingBillBalance{
			BillID: bill.ID, BillNo: bill.BillNo, BillDate: bill.BillDate,
			TotalAmount: total, VerifiedAmount: verifiedSums[bill.ID].Round(8), NettedAmount: nettedSums[bill.ID].Round(8),
			Version: bill.Version,
		}
		balance.AvailableAmount = total.Sub(balance.VerifiedAmount).Sub(balance.NettedAmount)
		if balance.AvailableAmount.IsNegative() {
			balance.AvailableAmount = decimal.Zero
		}
		if bill.Direction == financebillent.DirectionRECEIVABLE {
			preview.ReceivableBills = append(preview.ReceivableBills, balance)
		} else {
			preview.PayableBills = append(preview.PayableBills, balance)
		}
	}
	return preview, nil
}

func loadFinanceBillVerifiedAmounts(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	result := make(map[uuid.UUID]decimal.Decimal, len(billIDs))
	if len(billIDs) == 0 {
		return result, nil
	}
	allocations, err := client.FinanceVerificationAllocation.Query().
		Where(verificationallocationent.BillIDIn(billIDs...), verificationallocationent.ActiveEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range allocations {
		amount, parseErr := decimalOf(allocation.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		result[allocation.BillID] = result[allocation.BillID].Add(amount)
	}
	return result, nil
}

func loadFinanceBillNettedAmounts(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	result := make(map[uuid.UUID]decimal.Decimal, len(billIDs))
	if len(billIDs) == 0 {
		return result, nil
	}
	allocations, err := client.FinanceNettingAllocation.Query().
		Where(financenettingallocationent.BillIDIn(billIDs...), financenettingallocationent.ActiveEQ(true)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range allocations {
		amount, parseErr := decimalOf(allocation.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		result[allocation.BillID] = result[allocation.BillID].Add(amount)
	}
	return result, nil
}

// LockNettingBills 在共享事务内按账单主键稳定排序加锁，并汇总有效核销与对冲余额。
// 后续的对冲/核销写入都以账单行锁串行化，锁内读取的分摊余额因此不会丢失并发更新。
func (r *financeNettingRepo) LockNettingBills(ctx context.Context, organizationID uuid.UUID, billIDs []uuid.UUID) ([]*biz.FinanceNettingBill, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.FinanceBill.Query().
		Where(financebillent.OrganizationIDEQ(organizationID), financebillent.IDIn(billIDs...)).
		WithSettlementParty().Order(financebillent.ByID())
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForUpdate()
	}
	bills, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	lockedIDs := make([]uuid.UUID, 0, len(bills))
	for _, bill := range bills {
		lockedIDs = append(lockedIDs, bill.ID)
	}
	verifiedSums, err := loadFinanceBillVerifiedAmounts(ctx, client, lockedIDs)
	if err != nil {
		return nil, err
	}
	nettedSums, err := loadFinanceBillNettedAmounts(ctx, client, lockedIDs)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.FinanceNettingBill, 0, len(bills))
	for _, bill := range bills {
		total, parseErr := decimalOf(bill.TotalAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		exchangeRate, parseErr := decimalOf(bill.ExchangeRate)
		if parseErr != nil {
			return nil, parseErr
		}
		party, edgeErr := bill.Edges.SettlementPartyOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		result = append(result, &biz.FinanceNettingBill{
			ID: bill.ID, BillNo: bill.BillNo, BillDate: bill.BillDate,
			Status: biz.FinanceBillStatus(bill.Status), Direction: biz.OrderFeeDirection(bill.Direction),
			SettlementPartyID: bill.SettlementPartyID, SettlementPartyName: party.LegalName,
			Currency: bill.Currency, BaseCurrency: bill.BaseCurrency, ExchangeRate: exchangeRate,
			TotalAmount: total, VerifiedAmount: verifiedSums[bill.ID].Round(8), NettedAmount: nettedSums[bill.ID].Round(8),
			Version: bill.Version,
		})
	}
	return result, nil
}

func (r *financeNettingRepo) Create(ctx context.Context, organizationID, _ uuid.UUID, netting *biz.FinanceNetting, audit *biz.AuditEvent) (*biz.FinanceNetting, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		now := time.Now().UTC()
		rule, sequence, err := allocateNumberInTx(ctx, tx, organizationID, biz.DocumentTypeNetting, now)
		if err != nil {
			return err
		}
		netting.NettingNo, err = biz.FormatAllocatedNumber(now, rule, sequence, "")
		if err != nil {
			return err
		}
		_, err = tx.FinanceNetting.Create().
			SetID(netting.ID).SetOrganizationID(organizationID).SetNettingNo(netting.NettingNo).
			SetIdempotencyKey(netting.IdempotencyKey).SetRequestHash(netting.RequestHash).
			SetNillableBatchID(netting.BatchID).SetStatus(financenettingent.StatusDRAFT).
			SetSettlementPartyID(netting.SettlementPartyID).SetSettlementPartyName(netting.SettlementPartyName).
			SetCurrency(netting.Currency).SetAmount(netting.Amount.StringFixed(8)).
			SetBaseCurrency(netting.BaseCurrency).SetBaseCurrencyAmount(netting.BaseCurrencyAmount.StringFixed(8)).
			SetPayableBaseAmount(netting.PayableBaseAmount.StringFixed(8)).
			SetExchangeGainLoss(netting.ExchangeGainLoss.StringFixed(8)).
			SetNillableNote(netting.Note).SetVersion(1).Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "financenetting_organization_id_idempotency_key", biz.ErrFinanceNettingIdempotency)
		}
		builders := make([]*ent.FinanceNettingAllocationCreate, 0, len(netting.Allocations))
		for _, allocation := range netting.Allocations {
			builders = append(builders, tx.FinanceNettingAllocation.Create().
				SetID(allocation.ID).SetNettingID(netting.ID).SetBillID(allocation.BillID).SetBillNo(allocation.BillNo).
				SetDirection(financenettingallocationent.Direction(allocation.Direction)).
				SetAmount(allocation.Amount.StringFixed(8)).SetBaseCurrencyAmount(allocation.BaseCurrencyAmount.StringFixed(8)).
				SetActive(false))
		}
		if _, err = tx.FinanceNettingAllocation.CreateBulk(builders...).Save(ctx); err != nil {
			return mapEntConstraint(err, "netting_allocation_pair_unique", biz.ErrFinanceNettingInvalid)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	if _, transactional := transactionFromContext(ctx); transactional {
		return netting, nil
	}
	return r.Get(ctx, []uuid.UUID{organizationID}, netting.ID)
}

func (r *financeNettingRepo) Confirm(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) (*biz.FinanceNetting, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, err := tx.FinanceNetting.Query().Where(financenettingent.IDEQ(id), financenettingent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceNettingNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrFinanceNettingVersionConflict
		}
		if item.Status != financenettingent.StatusDRAFT {
			return biz.ErrFinanceNettingTransition
		}
		// 分摊在确认时才生效；生效后立即参与账单可用余额计算。
		if _, err = tx.FinanceNettingAllocation.Update().Where(financenettingallocationent.NettingIDEQ(id), financenettingallocationent.ActiveEQ(false)).SetActive(true).Save(ctx); err != nil {
			return err
		}
		now := time.Now().UTC()
		if _, err = tx.FinanceNetting.UpdateOneID(id).
			SetStatus(financenettingent.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actorID).
			SetVersion(item.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, []uuid.UUID{organizationID}, id)
}

func (r *financeNettingRepo) Cancel(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceNetting, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, err := tx.FinanceNetting.Query().Where(financenettingent.IDEQ(id), financenettingent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceNettingNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrFinanceNettingVersionConflict
		}
		// 只有未生效草稿可以取消；已确认对冲必须走反转恢复双方余额。
		if item.Status != financenettingent.StatusDRAFT {
			return biz.ErrFinanceNettingTransition
		}
		now := time.Now().UTC()
		if _, err = tx.FinanceNetting.UpdateOneID(id).
			SetStatus(financenettingent.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actorID).
			SetCancellationReason(reason).SetVersion(item.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, []uuid.UUID{organizationID}, id)
}

// Reverse 沿用创建/确认的锁顺序（先对冲单后分摊），反转后分摊失效、双方账单未结余额恢复。
// 账单余额由有效分摊派生，反转不需要改写账单行，也不触碰发票与账单折算快照。
func (r *financeNettingRepo) Reverse(ctx context.Context, organizationID, actorID, id uuid.UUID, expectedVersion uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceNetting, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, err := tx.FinanceNetting.Query().Where(financenettingent.IDEQ(id), financenettingent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceNettingNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrFinanceNettingVersionConflict
		}
		if item.Status != financenettingent.StatusCONFIRMED {
			return biz.ErrFinanceNettingTransition
		}
		if _, err = tx.FinanceNettingAllocation.Update().Where(financenettingallocationent.NettingIDEQ(id), financenettingallocationent.ActiveEQ(true)).SetActive(false).Save(ctx); err != nil {
			return err
		}
		now := time.Now().UTC()
		if _, err = tx.FinanceNetting.UpdateOneID(id).
			SetStatus(financenettingent.StatusREVERSED).SetReversedAt(now).SetReversedBy(actorID).
			SetReversalReason(reason).SetVersion(item.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, []uuid.UUID{organizationID}, id)
}

func financeNettingToBiz(item *ent.FinanceNetting) (*biz.FinanceNetting, error) {
	amount, err := decimalOf(item.Amount)
	if err != nil {
		return nil, err
	}
	baseAmount, err := decimalOf(item.BaseCurrencyAmount)
	if err != nil {
		return nil, err
	}
	payableBaseAmount, err := decimalOf(item.PayableBaseAmount)
	if err != nil {
		return nil, err
	}
	exchangeGainLoss, err := decimalOf(item.ExchangeGainLoss)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceNetting{
		ID: item.ID, OrganizationID: item.OrganizationID, BatchID: item.BatchID, NettingNo: item.NettingNo,
		IdempotencyKey: item.IdempotencyKey, RequestHash: item.RequestHash, Status: biz.FinanceNettingStatus(item.Status),
		SettlementPartyID: item.SettlementPartyID, SettlementPartyName: item.SettlementPartyName,
		Currency: item.Currency, Amount: amount, BaseCurrency: item.BaseCurrency, BaseCurrencyAmount: baseAmount,
		PayableBaseAmount: payableBaseAmount, ExchangeGainLoss: exchangeGainLoss,
		Note: item.Note, Version: item.Version,
		ConfirmedAt: item.ConfirmedAt, ConfirmedBy: item.ConfirmedBy,
		CancelledAt: item.CancelledAt, CancelledBy: item.CancelledBy, CancellationReason: item.CancellationReason,
		ReversedAt: item.ReversedAt, ReversedBy: item.ReversedBy, ReversalReason: item.ReversalReason,
		Allocations: make([]*biz.FinanceNettingAllocation, 0, len(item.Edges.Allocations)),
		CreatedAt:   item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if organization, edgeErr := item.Edges.OrganizationOrErr(); edgeErr == nil {
		result.OrganizationName = organization.Name
	}
	if batch, edgeErr := item.Edges.BatchOrErr(); edgeErr == nil {
		result.BatchNo = batch.BatchNo
	}
	for _, allocation := range item.Edges.Allocations {
		allocationAmount, parseErr := decimalOf(allocation.Amount)
		if parseErr != nil {
			return nil, parseErr
		}
		allocationBaseAmount, parseErr := decimalOf(allocation.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		result.Allocations = append(result.Allocations, &biz.FinanceNettingAllocation{
			ID: allocation.ID, NettingID: allocation.NettingID, BillID: allocation.BillID, BillNo: allocation.BillNo,
			Direction: biz.OrderFeeDirection(allocation.Direction), Amount: allocationAmount, BaseCurrencyAmount: allocationBaseAmount,
			Active: allocation.Active,
		})
	}
	return result, nil
}

var _ biz.FinanceNettingRepo = (*financeNettingRepo)(nil)
