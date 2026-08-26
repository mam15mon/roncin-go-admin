package data

import (
	"context"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	verificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type financeBillRepo struct{ data *Data }

func NewFinanceBillRepo(data *Data) biz.FinanceBillRepo { return &financeBillRepo{data: data} }

func (r *financeBillRepo) List(ctx context.Context, organizationID uuid.UUID, filter biz.FinanceBillFilter) (*biz.FinanceBillListResult, error) {
	predicates := []predicate.FinanceBill{financebillent.OrganizationIDEQ(organizationID)}
	if filter.Keyword != "" {
		predicates = append(predicates, financebillent.Or(
			financebillent.BillNoContainsFold(filter.Keyword),
			financebillent.SettlementPartyNameContainsFold(filter.Keyword),
			financebillent.HasLinesWith(financebilllineent.OrderNoContainsFold(filter.Keyword)),
		))
	}
	if filter.Direction != "" {
		predicates = append(predicates, financebillent.DirectionEQ(financebillent.Direction(filter.Direction)))
	}
	if filter.Status != "" {
		predicates = append(predicates, financebillent.StatusEQ(financebillent.Status(filter.Status)))
	}
	if filter.SettlementPartyID != nil {
		predicates = append(predicates, financebillent.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.Currency != "" {
		predicates = append(predicates, financebillent.CurrencyEQ(filter.Currency))
	}
	if filter.BillDateFrom != "" {
		predicates = append(predicates, financebillent.BillDateGTE(filter.BillDateFrom))
	}
	if filter.BillDateTo != "" {
		predicates = append(predicates, financebillent.BillDateLTE(filter.BillDateTo))
	}
	query := r.data.db.FinanceBill.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(financebillent.ByBillDate(entsql.OrderDesc()), financebillent.ByCreatedAt(entsql.OrderDesc()), financebillent.ByID(entsql.OrderDesc())).
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceBillListResult{Items: make([]*biz.FinanceBill, 0, len(items)), Total: int64(total)}
	for _, item := range items {
		converted, convertErr := financeBillToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	if err := r.enrichVerificationAmounts(ctx, result.Items); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *financeBillRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.FinanceBill, error) {
	item, err := r.financeBillQueryWithLines(r.data.db.FinanceBill.Query()).
		Where(financebillent.IDEQ(id), financebillent.OrganizationIDEQ(organizationID)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrFinanceBillNotFound
	}
	if err != nil {
		return nil, err
	}
	converted, err := financeBillToBiz(item)
	if err != nil {
		return nil, err
	}
	if err = r.enrichVerificationAmounts(ctx, []*biz.FinanceBill{converted}); err != nil {
		return nil, err
	}
	return converted, nil
}

func (r *financeBillRepo) enrichVerificationAmounts(ctx context.Context, bills []*biz.FinanceBill) error {
	if len(bills) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(bills))
	byID := make(map[uuid.UUID]*biz.FinanceBill, len(bills))
	for _, bill := range bills {
		ids = append(ids, bill.ID)
		byID[bill.ID] = bill
		bill.VerifiedAmount = decimal.Zero
		bill.UnverifiedAmount = bill.TotalAmount
	}
	allocations, err := r.data.db.FinanceVerificationAllocation.Query().Where(verificationallocationent.BillIDIn(ids...), verificationallocationent.ActiveEQ(true)).All(ctx)
	if err != nil {
		return err
	}
	for _, allocation := range allocations {
		amount, err := decimal.NewFromString(allocation.Amount)
		if err != nil {
			return err
		}
		bill := byID[allocation.BillID]
		bill.VerifiedAmount = bill.VerifiedAmount.Add(amount)
	}
	for _, bill := range bills {
		bill.VerifiedAmount = bill.VerifiedAmount.Round(8)
		bill.UnverifiedAmount = bill.TotalAmount.Sub(bill.VerifiedAmount).Round(8)
		if bill.UnverifiedAmount.IsNegative() {
			bill.UnverifiedAmount = decimal.Zero
		}
	}
	return nil
}

func (r *financeBillRepo) GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.FinanceBill, error) {
	item, err := r.financeBillQueryWithLines(r.data.db.FinanceBill.Query()).
		Where(financebillent.OrganizationIDEQ(organizationID), financebillent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return financeBillToBiz(item)
}

func (r *financeBillRepo) financeBillQueryWithLines(query *ent.FinanceBillQuery) *ent.FinanceBillQuery {
	return query.WithLines(func(lineQuery *ent.FinanceBillLineQuery) {
		lineQuery.WithOrder().Order(financebilllineent.ByCreatedAt(), financebilllineent.ByID())
	})
}

func (r *financeBillRepo) LoadBillableFees(ctx context.Context, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	items, err := r.data.db.OrderFee.Query().
		Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).
		WithSettlementParty().WithOrder().All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.FinanceBillableFee, 0, len(items))
	for _, item := range items {
		fee, convertErr := orderFeeToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		businessOrder, edgeErr := item.Edges.OrderOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		result = append(result, &biz.FinanceBillableFee{Fee: fee, OrderNo: businessOrder.OrderNo, BusinessType: string(businessOrder.BusinessType)})
	}
	return result, nil
}

func (r *financeBillRepo) Create(ctx context.Context, bill *biz.FinanceBill, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.FinanceBill, error) { _ = tx.Rollback(); return nil, value }
	feeIDs := make([]uuid.UUID, 0, len(bill.Lines))
	expected := make(map[uuid.UUID]*biz.FinanceBillLine, len(bill.Lines))
	for _, line := range bill.Lines {
		feeIDs = append(feeIDs, line.OrderFeeID)
		expected[line.OrderFeeID] = line
	}
	fees, err := tx.OrderFee.Query().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(bill.OrganizationID))).ForUpdate().All(ctx)
	if err != nil {
		return rollback(err)
	}
	if len(fees) != len(feeIDs) {
		return rollback(biz.ErrFinanceBillFeeInvalid)
	}
	for _, fee := range fees {
		line := expected[fee.ID]
		if line == nil || fee.Status != orderfeeent.StatusCONFIRMED || string(fee.Direction) != string(bill.Direction) || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != bill.Currency || fee.BaseCurrency != bill.BaseCurrency || fee.TotalAmount != line.TotalAmount.StringFixed(8) || fee.NetAmount != line.NetAmount.StringFixed(8) || fee.TaxAmount != line.TaxAmount.StringFixed(8) || fee.BaseCurrencyAmount != line.BaseCurrencyAmount.StringFixed(8) {
			return rollback(biz.ErrFinanceBillFeeInvalid)
		}
	}
	active, err := tx.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDIn(feeIDs...), financebilllineent.ActiveEQ(true)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if active {
		return rollback(biz.ErrFinanceBillFeeInvalid)
	}
	_, err = tx.FinanceBill.Create().
		SetID(bill.ID).SetOrganizationID(bill.OrganizationID).SetBillNo(bill.BillNo).SetIdempotencyKey(bill.IdempotencyKey).
		SetDirection(financebillent.Direction(bill.Direction)).SetStatus(financebillent.StatusDRAFT).
		SetSettlementPartyID(bill.SettlementPartyID).SetSettlementPartyName(bill.SettlementPartyName).
		SetCurrency(bill.Currency).SetBaseCurrency(bill.BaseCurrency).
		SetTotalAmount(bill.TotalAmount.StringFixed(8)).SetNetAmount(bill.NetAmount.StringFixed(8)).SetTaxAmount(bill.TaxAmount.StringFixed(8)).SetBaseCurrencyAmount(bill.BaseCurrencyAmount.StringFixed(8)).
		SetFeeCount(bill.FeeCount).SetBillDate(bill.BillDate).SetNillableDueDate(bill.DueDate).SetNillableNote(bill.Note).SetVersion(1).Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) && strings.Contains(err.Error(), "idempotency") {
			return rollback(biz.ErrFinanceBillIdempotencyConflict)
		}
		return rollback(err)
	}
	builders := make([]*ent.FinanceBillLineCreate, 0, len(bill.Lines))
	for _, line := range bill.Lines {
		builders = append(builders, tx.FinanceBillLine.Create().
			SetID(line.ID).SetBillID(bill.ID).SetOrderFeeID(line.OrderFeeID).SetOrderID(line.OrderID).
			SetOrderNo(line.OrderNo).SetFeeCode(line.FeeCode).SetFeeName(line.FeeName).
			SetTotalAmount(line.TotalAmount.StringFixed(8)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetCurrency(line.Currency).
			SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrency(line.BaseCurrency).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8)).SetActive(true))
	}
	if _, err = tx.FinanceBillLine.CreateBulk(builders...).Save(ctx); err != nil {
		if ent.IsConstraintError(err) {
			return rollback(biz.ErrFinanceBillFeeInvalid)
		}
		return rollback(err)
	}
	affected, err := tx.OrderFee.Update().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.StatusEQ(orderfeeent.StatusCONFIRMED)).SetStatus(orderfeeent.StatusBILLED).AddVersion(1).Save(ctx)
	if err != nil {
		return rollback(err)
	}
	if affected != len(feeIDs) {
		return rollback(biz.ErrFinanceBillFeeInvalid)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, bill.OrganizationID, bill.ID)
}

func (r *financeBillRepo) Update(ctx context.Context, organizationID uuid.UUID, input biz.UpdateFinanceBillInput, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.FinanceBill, error) { _ = tx.Rollback(); return nil, value }
	item, err := tx.FinanceBill.Query().Where(financebillent.IDEQ(input.ID), financebillent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrFinanceBillNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if item.Version != input.ExpectedVersion {
		return rollback(biz.ErrFinanceBillVersionConflict)
	}
	if item.Status != financebillent.StatusDRAFT {
		return rollback(biz.ErrFinanceBillInvalidTransition)
	}
	update := tx.FinanceBill.UpdateOneID(input.ID).SetBillDate(input.BillDate).SetVersion(item.Version + 1)
	if input.DueDate == nil {
		update.ClearDueDate()
	} else {
		update.SetDueDate(*input.DueDate)
	}
	if input.Note == nil {
		update.ClearNote()
	} else {
		update.SetNote(*input.Note)
	}
	if _, err = update.Save(ctx); err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, input.ID)
}

func (r *financeBillRepo) Confirm(ctx context.Context, organizationID, id, actorID uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.FinanceBill, error) { _ = tx.Rollback(); return nil, value }
	item, err := tx.FinanceBill.Query().Where(financebillent.IDEQ(id), financebillent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrFinanceBillNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if item.Version != expectedVersion {
		return rollback(biz.ErrFinanceBillVersionConflict)
	}
	if item.Status != financebillent.StatusDRAFT {
		return rollback(biz.ErrFinanceBillInvalidTransition)
	}
	now := time.Now()
	if _, err = tx.FinanceBill.UpdateOneID(id).SetStatus(financebillent.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actorID).SetVersion(item.Version + 1).Save(ctx); err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *financeBillRepo) Cancel(ctx context.Context, organizationID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.FinanceBill, error) { _ = tx.Rollback(); return nil, value }
	item, err := tx.FinanceBill.Query().Where(financebillent.IDEQ(id), financebillent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrFinanceBillNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if item.Version != expectedVersion {
		return rollback(biz.ErrFinanceBillVersionConflict)
	}
	if item.Status != financebillent.StatusDRAFT && item.Status != financebillent.StatusCONFIRMED {
		return rollback(biz.ErrFinanceBillInvalidTransition)
	}
	invoiced, err := tx.FinanceInvoiceBill.Query().Where(financeinvoicebillent.BillIDEQ(id), financeinvoicebillent.ActiveEQ(true)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if invoiced {
		return rollback(biz.ErrFinanceBillInvalidTransition)
	}
	verified, err := tx.FinanceVerificationAllocation.Query().Where(verificationallocationent.BillIDEQ(id), verificationallocationent.ActiveEQ(true)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if verified {
		return rollback(biz.ErrFinanceBillInvalidTransition)
	}
	lines, err := tx.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(id), financebilllineent.ActiveEQ(true)).ForUpdate().All(ctx)
	if err != nil {
		return rollback(err)
	}
	if len(lines) == 0 {
		return rollback(biz.ErrFinanceBillInvalidTransition)
	}
	feeIDs := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		feeIDs = append(feeIDs, line.OrderFeeID)
	}
	fees, err := tx.OrderFee.Query().Where(orderfeeent.IDIn(feeIDs...)).ForUpdate().All(ctx)
	if err != nil {
		return rollback(err)
	}
	if len(fees) != len(feeIDs) {
		return rollback(biz.ErrFinanceBillFeeInvalid)
	}
	for _, fee := range fees {
		if fee.Status != orderfeeent.StatusBILLED {
			return rollback(biz.ErrFinanceBillFeeInvalid)
		}
	}
	affected, err := tx.OrderFee.Update().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.StatusEQ(orderfeeent.StatusBILLED)).SetStatus(orderfeeent.StatusCONFIRMED).AddVersion(1).Save(ctx)
	if err != nil {
		return rollback(err)
	}
	if affected != len(feeIDs) {
		return rollback(biz.ErrFinanceBillFeeInvalid)
	}
	if _, err = tx.FinanceBillLine.Update().Where(financebilllineent.BillIDEQ(id), financebilllineent.ActiveEQ(true)).SetActive(false).Save(ctx); err != nil {
		return rollback(err)
	}
	now := time.Now()
	if _, err = tx.FinanceBill.UpdateOneID(id).SetStatus(financebillent.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actorID).SetCancellationReason(reason).SetVersion(item.Version + 1).Save(ctx); err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func financeBillToBiz(item *ent.FinanceBill) (*biz.FinanceBill, error) {
	totalAmount, err := decimal.NewFromString(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	netAmount, err := decimal.NewFromString(item.NetAmount)
	if err != nil {
		return nil, err
	}
	taxAmount, err := decimal.NewFromString(item.TaxAmount)
	if err != nil {
		return nil, err
	}
	baseAmount, err := decimal.NewFromString(item.BaseCurrencyAmount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceBill{
		ID: item.ID, OrganizationID: item.OrganizationID, BillNo: item.BillNo, IdempotencyKey: item.IdempotencyKey,
		Direction: biz.OrderFeeDirection(item.Direction), Status: biz.FinanceBillStatus(item.Status),
		SettlementPartyID: item.SettlementPartyID, SettlementPartyName: item.SettlementPartyName,
		Currency: item.Currency, BaseCurrency: item.BaseCurrency, TotalAmount: totalAmount, NetAmount: netAmount, TaxAmount: taxAmount,
		BaseCurrencyAmount: baseAmount, FeeCount: item.FeeCount, BillDate: item.BillDate, DueDate: item.DueDate, Note: item.Note,
		Version: item.Version, ConfirmedAt: item.ConfirmedAt, ConfirmedBy: item.ConfirmedBy, CancelledAt: item.CancelledAt,
		CancelledBy: item.CancelledBy, CancellationReason: item.CancellationReason, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		Lines: make([]*biz.FinanceBillLine, 0, len(item.Edges.Lines)),
	}
	for _, line := range item.Edges.Lines {
		converted, convertErr := financeBillLineToBiz(line)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Lines = append(result.Lines, converted)
	}
	return result, nil
}

func financeBillLineToBiz(item *ent.FinanceBillLine) (*biz.FinanceBillLine, error) {
	totalAmount, err := decimal.NewFromString(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	netAmount, err := decimal.NewFromString(item.NetAmount)
	if err != nil {
		return nil, err
	}
	taxAmount, err := decimal.NewFromString(item.TaxAmount)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := decimal.NewFromString(item.ExchangeRate)
	if err != nil {
		return nil, err
	}
	baseAmount, err := decimal.NewFromString(item.BaseCurrencyAmount)
	if err != nil {
		return nil, err
	}
	businessType := ""
	if businessOrder, edgeErr := item.Edges.OrderOrErr(); edgeErr == nil {
		businessType = string(businessOrder.BusinessType)
	}
	return &biz.FinanceBillLine{
		ID: item.ID, BillID: item.BillID, OrderFeeID: item.OrderFeeID, OrderID: item.OrderID, OrderNo: item.OrderNo, BusinessType: businessType,
		FeeCode: item.FeeCode, FeeName: item.FeeName, TotalAmount: totalAmount, NetAmount: netAmount, TaxAmount: taxAmount,
		Currency: item.Currency, ExchangeRate: exchangeRate, BaseCurrency: item.BaseCurrency, BaseCurrencyAmount: baseAmount,
		Active: item.Active, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}, nil
}

var _ biz.FinanceBillRepo = (*financeBillRepo)(nil)
