package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	exchangerateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/shopspring/decimal"
)

type exchangeRateRepo struct{ data *Data }

func NewExchangeRateRepo(data *Data) biz.ExchangeRateRepo { return &exchangeRateRepo{data: data} }

func (r *exchangeRateRepo) AccountingOrganization(ctx context.Context, organizationID uuid.UUID) (*biz.AccountingOrganization, error) {
	currentID := organizationID
	for {
		item, err := r.data.db.Organization.Query().Where(organizationent.IDEQ(currentID), organizationent.EnabledEQ(true)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return nil, err
		}
		if item.Kind == organizationent.KindHeadquarters || item.Kind == organizationent.KindCompany {
			if item.BaseCurrency == nil {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return &biz.AccountingOrganization{ID: item.ID, BaseCurrency: *item.BaseCurrency}, nil
		}
		if item.ParentID == nil {
			return nil, biz.ErrExchangeRateOrganizationInvalid
		}
		currentID = *item.ParentID
	}
}

func (r *exchangeRateRepo) List(ctx context.Context, organizationID uuid.UUID) ([]*biz.ExchangeRateSetting, error) {
	items, err := r.data.db.ExchangeRateSetting.Query().
		Where(exchangerateent.OrganizationIDEQ(organizationID)).
		Order(exchangerateent.ByFromCurrency(), exchangerateent.ByEffectiveFrom(), exchangerateent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.ExchangeRateSetting, 0, len(items))
	for _, item := range items {
		converted, err := exchangeRateToBiz(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *exchangeRateRepo) Create(ctx context.Context, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	return r.save(ctx, input, audit, false)
}

func (r *exchangeRateRepo) Update(ctx context.Context, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	return r.save(ctx, input, audit, true)
}

func (r *exchangeRateRepo) save(ctx context.Context, input *biz.ExchangeRateSetting, audit *biz.AuditEvent, updating bool) (*biz.ExchangeRateSetting, error) {
	if err := r.validateCurrencies(ctx, input.FromCurrency, input.ToCurrency); err != nil {
		return nil, err
	}
	lockKey := fmt.Sprintf("exchange-rate:%s:%s:%s:%s:%s", input.OrganizationID, input.RateType, input.FromCurrency, input.ToCurrency, input.TimeStandard)
	connection, err := r.data.sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext($1))", lockKey); err != nil {
		return nil, err
	}
	defer connection.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", lockKey)

	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if updating {
		current, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(input.ID), exchangerateent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			_ = tx.Rollback()
			if ent.IsNotFound(queryErr) {
				return nil, biz.ErrExchangeRateNotFound
			}
			return nil, queryErr
		}
		if !current.IsActive {
			_ = tx.Rollback()
			return nil, biz.ErrExchangeRateNotFound
		}
	}
	conflict := tx.ExchangeRateSetting.Query().Where(
		exchangerateent.OrganizationIDEQ(input.OrganizationID), exchangerateent.RateTypeEQ(exchangerateent.RateType(input.RateType)),
		exchangerateent.FromCurrencyEQ(input.FromCurrency), exchangerateent.ToCurrencyEQ(input.ToCurrency),
		exchangerateent.TimeStandardEQ(exchangerateent.TimeStandard(input.TimeStandard)), exchangerateent.IsActiveEQ(true),
		exchangerateent.IDNEQ(input.ID),
		exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(input.EffectiveFrom)),
	)
	if input.EffectiveTo != nil {
		conflict.Where(exchangerateent.EffectiveFromLT(*input.EffectiveTo))
	}
	hasConflict, err := conflict.Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if hasConflict {
		_ = tx.Rollback()
		return nil, biz.ErrExchangeRateOverlap
	}

	var saved *ent.ExchangeRateSetting
	if updating {
		builder := tx.ExchangeRateSetting.UpdateOneID(input.ID).
			SetRateType(exchangerateent.RateType(input.RateType)).SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
			SetTimeStandard(exchangerateent.TimeStandard(input.TimeStandard)).SetEffectiveFrom(input.EffectiveFrom).
			SetReceivableRate(input.ReceivableRate.StringFixed(8)).SetPayableRate(input.PayableRate.StringFixed(8))
		if input.EffectiveTo == nil {
			builder.ClearEffectiveTo()
		} else {
			builder.SetEffectiveTo(*input.EffectiveTo)
		}
		saved, err = builder.Save(ctx)
	} else {
		builder := tx.ExchangeRateSetting.Create().SetID(input.ID).SetOrganizationID(input.OrganizationID).
			SetRateType(exchangerateent.RateType(input.RateType)).SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
			SetTimeStandard(exchangerateent.TimeStandard(input.TimeStandard)).SetEffectiveFrom(input.EffectiveFrom).
			SetReceivableRate(input.ReceivableRate.StringFixed(8)).SetPayableRate(input.PayableRate.StringFixed(8)).SetIsActive(true)
		if input.EffectiveTo != nil {
			builder.SetEffectiveTo(*input.EffectiveTo)
		}
		saved, err = builder.Save(ctx)
	}
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
	return exchangeRateToBiz(saved)
}

func (r *exchangeRateRepo) Disable(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	item, err := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(id), exchangerateent.OrganizationIDEQ(organizationID), exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return biz.ErrExchangeRateNotFound
		}
		return err
	}
	if _, err = item.Update().SetIsActive(false).Save(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *exchangeRateRepo) Resolve(ctx context.Context, organizationID uuid.UUID, direction biz.OrderFeeDirection, fromCurrency, toCurrency, rateDate string) (*biz.ResolvedExchangeRate, error) {
	items, err := r.data.db.ExchangeRateSetting.Query().Where(
		exchangerateent.OrganizationIDEQ(organizationID), exchangerateent.RateTypeEQ(exchangerateent.RateTypeSETTLEMENT),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.TimeStandardEQ(exchangerateent.TimeStandardEXPENSE_DATE), exchangerateent.IsActiveEQ(true),
		exchangerateent.EffectiveFromLTE(rateDate), exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(rateDate)),
	).Limit(2).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, biz.ErrExchangeRateMissing
	}
	if len(items) > 1 {
		return nil, biz.ErrExchangeRateConflict
	}
	setting, err := exchangeRateToBiz(items[0])
	if err != nil {
		return nil, err
	}
	rate := setting.ReceivableRate
	if direction == biz.OrderFeePayable {
		rate = setting.PayableRate
	}
	id := setting.ID
	return &biz.ResolvedExchangeRate{Rate: rate, Source: "SYSTEM", RateDate: rateDate, SettingID: &id}, nil
}

func (r *exchangeRateRepo) validateCurrencies(ctx context.Context, codes ...string) error {
	count, err := r.data.db.Currency.Query().Where(currencyent.CodeIn(codes...), currencyent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(codes) {
		return biz.ErrExchangeRateCurrencyInvalid
	}
	return nil
}

func exchangeRateToBiz(item *ent.ExchangeRateSetting) (*biz.ExchangeRateSetting, error) {
	receivable, err := decimal.NewFromString(item.ReceivableRate)
	if err != nil {
		return nil, err
	}
	payable, err := decimal.NewFromString(item.PayableRate)
	if err != nil {
		return nil, err
	}
	return &biz.ExchangeRateSetting{ID: item.ID, OrganizationID: item.OrganizationID, RateType: string(item.RateType), FromCurrency: item.FromCurrency, ToCurrency: item.ToCurrency, TimeStandard: string(item.TimeStandard), EffectiveFrom: item.EffectiveFrom, EffectiveTo: item.EffectiveTo, ReceivableRate: receivable, PayableRate: payable, IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

var _ biz.ExchangeRateRepo = (*exchangeRateRepo)(nil)
