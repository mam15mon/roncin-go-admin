package data

import (
	"context"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	partner "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type financeCashflowRepo struct{ data *Data }

type financeCashflowSummaryRow struct {
	Direction    string `json:"direction"`
	BaseCurrency string `json:"base_currency"`
	BaseAmount   string `json:"base_amount"`
}

type financeCashflowVerifiedSummaryRow struct {
	Active             bool   `json:"active"`
	VerifiedBaseAmount string `json:"verified_base_amount"`
}

func NewFinanceCashflowRepo(d *Data) biz.FinanceCashflowRepo { return &financeCashflowRepo{d} }
func (r *financeCashflowRepo) List(ctx context.Context, org uuid.UUID, f biz.FinanceCashflowFilter) (*biz.FinanceCashflowListResult, error) {
	p := []predicate.FinanceCashflow{cash.OrganizationIDEQ(org)}
	if f.Keyword != "" {
		p = append(p, cash.Or(cash.FlowNoContainsFold(f.Keyword), cash.SettlementPartyNameContainsFold(f.Keyword), cash.BankReferenceNoContainsFold(f.Keyword)))
	}
	if f.Direction != "" {
		p = append(p, cash.DirectionEQ(cash.Direction(f.Direction)))
	}
	if f.Status != "" {
		p = append(p, cash.StatusEQ(cash.Status(f.Status)))
	}
	if f.SettlementPartyID != nil {
		p = append(p, cash.SettlementPartyIDEQ(*f.SettlementPartyID))
	}
	if f.Currency != "" {
		p = append(p, cash.CurrencyEQ(f.Currency))
	}
	q := r.data.db.FinanceCashflow.Query().Where(p...)
	n, e := q.Clone().Count(ctx)
	if e != nil {
		return nil, e
	}
	summaryRows := make([]financeCashflowSummaryRow, 0)
	if e := q.Clone().
		GroupBy(cash.FieldDirection, cash.FieldBaseCurrency).
		Aggregate(ent.As(ent.Sum(cash.FieldBaseAmount), "base_amount")).
		Scan(ctx, &summaryRows); e != nil {
		return nil, e
	}
	summary := biz.FinanceCashflowSummary{
		ReceivableBaseAmount: decimal.Zero,
		PayableBaseAmount:    decimal.Zero,
		UnverifiedBaseAmount: decimal.Zero,
	}
	for _, row := range summaryRows {
		amount, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		summary.BaseCurrency = row.BaseCurrency
		if row.Direction == string(cash.DirectionRECEIVABLE) {
			summary.ReceivableBaseAmount = summary.ReceivableBaseAmount.Add(amount)
		} else {
			summary.PayableBaseAmount = summary.PayableBaseAmount.Add(amount)
		}
	}
	verifiedRows := make([]financeCashflowVerifiedSummaryRow, 0, 1)
	if e := r.data.db.FinanceVerificationAllocation.Query().
		Where(allocation.ActiveEQ(true), allocation.HasCashflowWith(p...)).
		GroupBy(allocation.FieldActive).
		Aggregate(ent.As(ent.Sum(allocation.FieldCashflowBaseAmount), "verified_base_amount")).
		Scan(ctx, &verifiedRows); e != nil {
		return nil, e
	}
	verifiedBaseAmount := decimal.Zero
	if len(verifiedRows) > 0 {
		verifiedBaseAmount, e = decimalOf(verifiedRows[0].VerifiedBaseAmount)
		if e != nil {
			return nil, e
		}
	}
	summary.UnverifiedBaseAmount = summary.ReceivableBaseAmount.Add(summary.PayableBaseAmount).Sub(verifiedBaseAmount)
	xs, e := q.Order(cash.ByTransactionDate(entsql.OrderDesc()), cash.ByCreatedAt(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if e != nil {
		return nil, e
	}
	out := &biz.FinanceCashflowListResult{Items: make([]*biz.FinanceCashflow, 0, len(xs)), Total: int64(n), Summary: summary}
	for _, x := range xs {
		v, e := cashflowToBiz(x)
		if e != nil {
			return nil, e
		}
		out.Items = append(out.Items, v)
	}
	if e = r.enrichVerificationAmounts(ctx, out.Items); e != nil {
		return nil, e
	}
	return out, nil
}
func (r *financeCashflowRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCashflow, error) {
	x, e := r.data.db.FinanceCashflow.Query().Where(cash.IDEQ(id), cash.OrganizationIDEQ(org)).Only(ctx)
	if e != nil {
		return nil, mapEntError(e, biz.ErrFinanceCashflowNotFound, nil)
	}
	v, e := cashflowToBiz(x)
	if e != nil {
		return nil, e
	}
	if e = r.enrichVerificationAmounts(ctx, []*biz.FinanceCashflow{v}); e != nil {
		return nil, e
	}
	return v, nil
}

func (r *financeCashflowRepo) enrichVerificationAmounts(ctx context.Context, cashflows []*biz.FinanceCashflow) error {
	if len(cashflows) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(cashflows))
	byID := make(map[uuid.UUID]*biz.FinanceCashflow, len(cashflows))
	for _, cashflow := range cashflows {
		ids = append(ids, cashflow.ID)
		byID[cashflow.ID] = cashflow
		cashflow.VerifiedAmount = decimal.Zero
		cashflow.UnverifiedAmount = cashflow.Amount
	}
	allocations, err := r.data.db.FinanceVerificationAllocation.Query().Where(allocation.CashflowIDIn(ids...), allocation.ActiveEQ(true)).All(ctx)
	if err != nil {
		return err
	}
	for _, item := range allocations {
		amount, err := decimalOf(item.Amount)
		if err != nil {
			return err
		}
		cashflow := byID[item.CashflowID]
		cashflow.VerifiedAmount = cashflow.VerifiedAmount.Add(amount)
	}
	for _, cashflow := range cashflows {
		cashflow.VerifiedAmount = cashflow.VerifiedAmount.Round(8)
		cashflow.UnverifiedAmount = cashflow.Amount.Sub(cashflow.VerifiedAmount).Round(8)
		if cashflow.UnverifiedAmount.IsNegative() {
			cashflow.UnverifiedAmount = decimal.Zero
		}
	}
	return nil
}
func (r *financeCashflowRepo) GetByIdempotencyKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceCashflow, error) {
	x, e := r.data.db.FinanceCashflow.Query().Where(cash.OrganizationIDEQ(org), cash.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return cashflowToBiz(x)
}
func (r *financeCashflowRepo) ResolveParty(ctx context.Context, org, id uuid.UUID) (string, error) {
	x, e := r.data.db.Partner.Query().Where(partner.IDEQ(id), partner.OrganizationIDEQ(org), partner.EnabledEQ(true)).Only(ctx)
	if e != nil {
		return "", mapEntError(e, biz.ErrOrderFeePartyInvalid, nil)
	}
	return x.LegalName, nil
}
func (r *financeCashflowRepo) Create(ctx context.Context, v *biz.FinanceCashflow, a *biz.AuditEvent) (*biz.FinanceCashflow, error) {
	e := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		now := time.Now().UTC()
		rule, sequence, allocateErr := allocateNumberInTx(ctx, tx, v.OrganizationID, biz.DocumentTypeReceiptPayment, now)
		if allocateErr != nil {
			return allocateErr
		}
		var formatErr error
		v.FlowNo, formatErr = biz.FormatAllocatedNumber(now, rule, sequence, "")
		if formatErr != nil {
			return formatErr
		}
		_, saveErr := tx.FinanceCashflow.Create().SetID(v.ID).SetOrganizationID(v.OrganizationID).SetFlowNo(v.FlowNo).SetIdempotencyKey(v.IdempotencyKey).SetDirection(cash.Direction(v.Direction)).SetStatus(cash.StatusDRAFT).SetSettlementPartyID(v.SettlementPartyID).SetSettlementPartyName(v.SettlementPartyName).SetCurrency(v.Currency).SetAmount(v.Amount.StringFixed(8)).SetExchangeRate(v.ExchangeRate.StringFixed(8)).SetExchangeRateSource(cash.ExchangeRateSource(v.ExchangeRateSource)).SetExchangeRateDate(v.ExchangeRateDate).SetNillableExchangeRateSettingID(v.ExchangeRateSettingID).SetBaseCurrency(v.BaseCurrency).SetBaseAmount(v.BaseAmount.StringFixed(8)).SetTransactionDate(v.TransactionDate).SetOurAccount(v.OurAccount).SetNillableCounterpartyAccount(v.CounterpartyAccount).SetPaymentMethod(v.PaymentMethod).SetNillableBankReferenceNo(v.BankReferenceNo).SetNillableNote(v.Note).SetVersion(1).Save(ctx)
		if saveErr != nil {
			return mapEntConstraint(saveErr, "financecashflow_organization_id_idempotency_key", biz.ErrFinanceCashflowIdempotencyConflict)
		}
		return writeAudit(ctx, tx.AuditLog, a)
	})
	if e != nil {
		return nil, e
	}
	return r.Get(ctx, v.OrganizationID, v.ID)
}

func (r *financeCashflowRepo) Confirm(ctx context.Context, org, id, actor uuid.UUID, v uint64, a *biz.AuditEvent) (*biz.FinanceCashflow, error) {
	return r.transition(ctx, org, id, actor, v, "", true, a)
}
func (r *financeCashflowRepo) Cancel(ctx context.Context, org, id, actor uuid.UUID, v uint64, reason string, a *biz.AuditEvent) (*biz.FinanceCashflow, error) {
	return r.transition(ctx, org, id, actor, v, reason, false, a)
}
func (r *financeCashflowRepo) transition(ctx context.Context, org, id, actor uuid.UUID, v uint64, reason string, confirm bool, a *biz.AuditEvent) (*biz.FinanceCashflow, error) {
	e := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		x, queryErr := tx.FinanceCashflow.Query().Where(cash.IDEQ(id), cash.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrFinanceCashflowNotFound, nil)
		}
		if x.Version != v {
			return biz.ErrFinanceCashflowVersionConflict
		}
		now := time.Now()
		u := tx.FinanceCashflow.UpdateOneID(id).SetVersion(v + 1)
		if confirm {
			if x.Status != cash.StatusDRAFT {
				return biz.ErrFinanceCashflowInvalidTransition
			}
			u.SetStatus(cash.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor)
		} else {
			if x.Status == cash.StatusCANCELLED {
				return biz.ErrFinanceCashflowInvalidTransition
			}
			used, checkErr := tx.FinanceVerificationAllocation.Query().Where(allocation.CashflowIDEQ(id), allocation.ActiveEQ(true)).Exist(ctx)
			if checkErr != nil {
				return checkErr
			}
			if used {
				return biz.ErrFinanceCashflowInvalidTransition
			}
			u.SetStatus(cash.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason)
		}
		if _, saveErr := u.Save(ctx); saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, a)
	})
	if e != nil {
		return nil, e
	}
	return r.Get(ctx, org, id)
}
func cashflowToBiz(x *ent.FinanceCashflow) (*biz.FinanceCashflow, error) {
	amount, e := decimalOf(x.Amount)
	if e != nil {
		return nil, e
	}
	rate, e := decimalOf(x.ExchangeRate)
	if e != nil {
		return nil, e
	}
	base, e := decimalOf(x.BaseAmount)
	if e != nil {
		return nil, e
	}
	return &biz.FinanceCashflow{ID: x.ID, OrganizationID: x.OrganizationID, FlowNo: x.FlowNo, IdempotencyKey: x.IdempotencyKey, Direction: biz.OrderFeeDirection(x.Direction), Status: biz.FinanceCashflowStatus(x.Status), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: amount, ExchangeRate: rate, ExchangeRateSource: string(x.ExchangeRateSource), ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingID: x.ExchangeRateSettingID, BaseCurrency: x.BaseCurrency, BaseAmount: base, TransactionDate: x.TransactionDate, OurAccount: x.OurAccount, CounterpartyAccount: x.CounterpartyAccount, PaymentMethod: x.PaymentMethod, BankReferenceNo: x.BankReferenceNo, Note: x.Note, Version: x.Version, ConfirmedAt: x.ConfirmedAt, ConfirmedBy: x.ConfirmedBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}, nil
}

var _ biz.FinanceCashflowRepo = (*financeCashflowRepo)(nil)
