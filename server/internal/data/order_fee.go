package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/shopspring/decimal"
)

type orderFeeRepo struct {
	data *Data
}

func NewOrderFeeRepo(data *Data) biz.OrderFeeRepo {
	return &orderFeeRepo{data: data}
}

func (r *orderFeeRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	exists, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderFeeNotFound
	}
	return nil
}

func (r *orderFeeRepo) settlementParty(ctx context.Context, organizationID, partyID uuid.UUID) (*ent.Partner, error) {
	item, err := r.data.db.Partner.Query().Where(partnerent.IDEQ(partyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderFeePartyInvalid
		}
		return nil, err
	}
	return item, nil
}

func (r *orderFeeRepo) validateCurrency(ctx context.Context, code string) error {
	exists, err := r.data.db.Currency.Query().Where(currencyent.CodeEQ(code), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderFeeCurrencyInvalid
	}
	return nil
}

func (r *orderFeeRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderFee.Query().
		Where(orderfeeent.OrderIDEQ(orderID)).
		WithSettlementParty().
		Order(orderfeeent.ByDirection(), orderfeeent.ByCreatedAt(), orderfeeent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderFee, 0, len(items))
	for _, item := range items {
		converted, err := orderFeeToBiz(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *orderFeeRepo) Options(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeOptions, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	parties, err := r.data.db.Partner.Query().
		Where(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).
		Order(partnerent.ByLegalName(), partnerent.ByCode()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	currencies, err := r.data.db.Currency.Query().
		Where(currencyent.EnabledEQ(true)).
		Order(currencyent.ByCode()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFeeOptions{
		SettlementParties: make([]biz.OrderFeeSettlementPartyOption, 0, len(parties)),
		Currencies:        make([]biz.OrderFeeCurrencyOption, 0, len(currencies)),
	}
	for _, party := range parties {
		result.SettlementParties = append(result.SettlementParties, biz.OrderFeeSettlementPartyOption{ID: party.ID, Code: party.Code, Name: party.LegalName})
	}
	for _, currency := range currencies {
		result.Currencies = append(result.Currencies, biz.OrderFeeCurrencyOption{Code: currency.Code, Name: currency.Name, MinorUnit: currency.MinorUnit})
	}
	return result, nil
}

func (r *orderFeeRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	party, err := r.settlementParty(ctx, organizationID, input.SettlementPartyID)
	if err != nil {
		return nil, err
	}
	if err := r.validateCurrency(ctx, input.Currency); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := tx.OrderFee.Create().
		SetID(input.ID).
		SetOrderID(orderID).
		SetDirection(orderfeeent.Direction(input.Direction)).
		SetFeeCode(input.FeeCode).
		SetFeeName(input.FeeName).
		SetSettlementPartyID(input.SettlementPartyID).
		SetBillingUnit(input.BillingUnit).
		SetQuantity(input.Quantity.StringFixed(4)).
		SetUnitPrice(input.UnitPrice.StringFixed(4)).
		SetTotalAmount(input.TotalAmount.StringFixed(8)).
		SetCurrency(input.Currency).
		SetExchangeRate(input.ExchangeRate.StringFixed(6)).
		SetExpenseDate(input.ExpenseDate).
		SetNillableNote(input.Note).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	input.SettlementPartyName = party.LegalName
	input.OrderID = orderID
	input.CreatedAt = created.CreatedAt
	input.UpdatedAt = created.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	party, err := r.settlementParty(ctx, organizationID, input.SettlementPartyID)
	if err != nil {
		return nil, err
	}
	if err := r.validateCurrency(ctx, input.Currency); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderFeeNotFound
		}
		return nil, err
	}
	builder := tx.OrderFee.UpdateOne(item).
		SetDirection(orderfeeent.Direction(input.Direction)).
		SetFeeCode(input.FeeCode).
		SetFeeName(input.FeeName).
		SetSettlementPartyID(input.SettlementPartyID).
		SetBillingUnit(input.BillingUnit).
		SetQuantity(input.Quantity.StringFixed(4)).
		SetUnitPrice(input.UnitPrice.StringFixed(4)).
		SetTotalAmount(input.TotalAmount.StringFixed(8)).
		SetCurrency(input.Currency).
		SetExchangeRate(input.ExchangeRate.StringFixed(6)).
		SetExpenseDate(input.ExpenseDate)
	if input.Note != nil {
		builder.SetNote(*input.Note)
	} else {
		builder.ClearNote()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	input.ID = id
	input.OrderID = orderID
	input.SettlementPartyName = party.LegalName
	input.CreatedAt = updated.CreatedAt
	input.UpdatedAt = updated.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	count, err := tx.OrderFee.Delete().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if count == 0 {
		_ = tx.Rollback()
		return biz.ErrOrderFeeNotFound
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func orderFeeToBiz(item *ent.OrderFee) (*biz.OrderFee, error) {
	party, err := item.Edges.SettlementPartyOrErr()
	if err != nil {
		return nil, err
	}
	quantity, err := decimal.NewFromString(item.Quantity)
	if err != nil {
		return nil, err
	}
	unitPrice, err := decimal.NewFromString(item.UnitPrice)
	if err != nil {
		return nil, err
	}
	totalAmount, err := decimal.NewFromString(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := decimal.NewFromString(item.ExchangeRate)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFee{
		ID:                  item.ID,
		OrderID:             item.OrderID,
		Direction:           biz.OrderFeeDirection(item.Direction),
		FeeCode:             item.FeeCode,
		FeeName:             item.FeeName,
		SettlementPartyID:   item.SettlementPartyID,
		SettlementPartyName: party.LegalName,
		BillingUnit:         item.BillingUnit,
		Quantity:            quantity,
		UnitPrice:           unitPrice,
		TotalAmount:         totalAmount,
		Currency:            item.Currency,
		ExchangeRate:        exchangeRate,
		ExpenseDate:         item.ExpenseDate,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
	if item.Note != "" {
		note := item.Note
		result.Note = &note
	}
	return result, nil
}

var _ biz.OrderFeeRepo = (*orderFeeRepo)(nil)
