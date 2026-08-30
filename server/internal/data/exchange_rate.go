package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	exchangeratecustomsettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratecustomsetting"
	exchangerateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	exchangeratetimestandardent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratetimestandard"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/shopspring/decimal"
)

type exchangeRateRepo struct{ data *Data }

func NewExchangeRateRepo(data *Data) biz.ExchangeRateRepo { return &exchangeRateRepo{data: data} }

func (r *exchangeRateRepo) ResolveContext(ctx context.Context, organizationID uuid.UUID) (*biz.ExchangeRateContext, error) {
	currentID := organizationID
	baseCurrency := ""
	for {
		item, err := r.data.db.Organization.Query().Where(organizationent.IDEQ(currentID), organizationent.EnabledEQ(true)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return nil, err
		}
		if baseCurrency == "" && item.BaseCurrency != nil {
			baseCurrency = *item.BaseCurrency
		}
		if item.ParentID == nil {
			if item.Kind != organizationent.KindHeadquarters || baseCurrency == "" {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return &biz.ExchangeRateContext{OwnerOrganizationID: item.ID, BaseCurrency: baseCurrency}, nil
		}
		currentID = *item.ParentID
	}
}

func (r *exchangeRateRepo) List(ctx context.Context, organizationID uuid.UUID) ([]*biz.ExchangeRateSetting, error) {
	items, err := r.data.db.ExchangeRateSetting.Query().
		Where(exchangerateent.OrganizationIDEQ(organizationID)).
		Order(exchangerateent.ByRateType(), exchangerateent.ByFromCurrency(), exchangerateent.ByEffectiveFrom(), exchangerateent.ByID()).All(ctx)
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
	lockKey := fmt.Sprintf("exchange-rate:%s:%s:%s:%s", input.OrganizationID, input.RateType, input.FromCurrency, input.ToCurrency)
	connection, err := r.data.sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext($1))", lockKey); err != nil {
		return nil, err
	}
	defer connection.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", lockKey)
	effectiveFrom, err := parseExchangeRateStorageTime(input.EffectiveFrom)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	var effectiveTo *time.Time
	if input.EffectiveTo != nil {
		parsed, parseErr := parseExchangeRateStorageTime(*input.EffectiveTo)
		if parseErr != nil {
			return nil, biz.ErrExchangeRateInvalidArgument
		}
		effectiveTo = &parsed
	}

	var saved *ent.ExchangeRateSetting
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if updating {
			current, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(input.ID), exchangerateent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
			if queryErr != nil {
				if ent.IsNotFound(queryErr) {
					return biz.ErrExchangeRateNotFound
				}
				return queryErr
			}
			if !current.IsActive {
				return biz.ErrExchangeRateNotFound
			}
		}
		conflict := tx.ExchangeRateSetting.Query().Where(
			exchangerateent.OrganizationIDEQ(input.OrganizationID), exchangerateent.RateTypeEQ(exchangerateent.RateType(input.RateType)),
			exchangerateent.FromCurrencyEQ(input.FromCurrency), exchangerateent.ToCurrencyEQ(input.ToCurrency),
			exchangerateent.IsActiveEQ(true),
			exchangerateent.IDNEQ(input.ID),
			exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(effectiveFrom)),
		)
		if effectiveTo != nil {
			conflict.Where(exchangerateent.EffectiveFromLT(*effectiveTo))
		}
		hasConflict, queryErr := conflict.Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if hasConflict {
			return biz.ErrExchangeRateOverlap
		}
		var saveErr error
		if updating {
			builder := tx.ExchangeRateSetting.UpdateOneID(input.ID).
				SetRateType(exchangerateent.RateType(input.RateType)).SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
				SetEffectiveFrom(effectiveFrom).
				SetReceivableRate(input.ReceivableRate.StringFixed(8)).SetPayableRate(input.PayableRate.StringFixed(8))
			if input.EffectiveTo == nil {
				builder.ClearEffectiveTo()
			} else {
				builder.SetEffectiveTo(*effectiveTo)
			}
			saved, saveErr = builder.Save(ctx)
		} else {
			builder := tx.ExchangeRateSetting.Create().SetID(input.ID).SetOrganizationID(input.OrganizationID).
				SetRateType(exchangerateent.RateType(input.RateType)).SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
				SetEffectiveFrom(effectiveFrom).
				SetReceivableRate(input.ReceivableRate.StringFixed(8)).SetPayableRate(input.PayableRate.StringFixed(8)).SetIsActive(true)
			if effectiveTo != nil {
				builder.SetEffectiveTo(*effectiveTo)
			}
			saved, saveErr = builder.Save(ctx)
		}
		if saveErr != nil {
			return saveErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return exchangeRateToBiz(saved)
}

func (r *exchangeRateRepo) Disable(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(id), exchangerateent.OrganizationIDEQ(organizationID), exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			if ent.IsNotFound(queryErr) {
				return biz.ErrExchangeRateNotFound
			}
			return queryErr
		}
		if _, updateErr := item.Update().SetIsActive(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *exchangeRateRepo) ListTimeStandards(ctx context.Context, organizationID uuid.UUID) ([]*biz.ExchangeRateTimeStandardSetting, error) {
	items, err := r.data.db.ExchangeRateTimeStandard.Query().
		Where(exchangeratetimestandardent.OrganizationIDEQ(organizationID)).
		Order(exchangeratetimestandardent.ByRateType(), exchangeratetimestandardent.BySortOrder()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.ExchangeRateTimeStandardSetting, 0, 5)
	byType := make(map[string]*biz.ExchangeRateTimeStandardSetting, 5)
	for _, item := range items {
		rateType := string(item.RateType)
		setting := byType[rateType]
		if setting == nil {
			setting = &biz.ExchangeRateTimeStandardSetting{RateType: rateType, TimeStandards: []string{}}
			byType[rateType] = setting
			result = append(result, setting)
		}
		setting.TimeStandards = append(setting.TimeStandards, string(item.TimeStandard))
	}
	return result, nil
}

func (r *exchangeRateRepo) ReplaceTimeStandards(ctx context.Context, organizationID uuid.UUID, settings []*biz.ExchangeRateTimeStandardSetting, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, deleteErr := tx.ExchangeRateTimeStandard.Delete().Where(exchangeratetimestandardent.OrganizationIDEQ(organizationID)).Exec(ctx); deleteErr != nil {
			return deleteErr
		}
		for _, setting := range settings {
			for index, standard := range setting.TimeStandards {
				if _, createErr := tx.ExchangeRateTimeStandard.Create().
					SetOrganizationID(organizationID).
					SetRateType(exchangeratetimestandardent.RateType(setting.RateType)).
					SetTimeStandard(exchangeratetimestandardent.TimeStandard(standard)).
					SetSortOrder(index).
					Save(ctx); createErr != nil {
					return createErr
				}
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *exchangeRateRepo) GetCustomSetting(ctx context.Context, organizationID uuid.UUID) (*biz.ExchangeRateCustomSetting, error) {
	item, err := r.data.db.ExchangeRateCustomSetting.Query().
		Where(exchangeratecustomsettingent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return exchangeRateCustomSettingToBiz(item), nil
}

func (r *exchangeRateRepo) SaveCustomSetting(ctx context.Context, setting *biz.ExchangeRateCustomSetting, expectedVersion uint64, audit *biz.AuditEvent) (*biz.ExchangeRateCustomSetting, error) {
	if setting == nil || setting.OrganizationID == uuid.Nil || setting.UpdatedBy == nil || *setting.UpdatedBy == uuid.Nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	var saved *ent.ExchangeRateCustomSetting
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		current, queryErr := tx.ExchangeRateCustomSetting.Query().
			Where(exchangeratecustomsettingent.OrganizationIDEQ(setting.OrganizationID)).
			ForUpdate().
			Only(ctx)
		switch {
		case ent.IsNotFound(queryErr):
			if expectedVersion != 0 {
				return biz.ErrExchangeRateCustomSettingConflict
			}
			var createErr error
			saved, createErr = tx.ExchangeRateCustomSetting.Create().
				SetOrganizationID(setting.OrganizationID).
				SetInheritBaseCurrencyRate(setting.InheritBaseCurrencyRate).
				SetVersion(1).
				SetUpdatedBy(*setting.UpdatedBy).
				Save(ctx)
			if ent.IsConstraintError(createErr) {
				return biz.ErrExchangeRateCustomSettingConflict
			}
			if createErr != nil {
				return createErr
			}
		case queryErr != nil:
			return queryErr
		case current.Version != expectedVersion:
			return biz.ErrExchangeRateCustomSettingConflict
		default:
			var updateErr error
			saved, updateErr = tx.ExchangeRateCustomSetting.UpdateOneID(current.ID).
				SetInheritBaseCurrencyRate(setting.InheritBaseCurrencyRate).
				SetVersion(current.Version + 1).
				SetUpdatedBy(*setting.UpdatedBy).
				Save(ctx)
			if updateErr != nil {
				return updateErr
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return exchangeRateCustomSettingToBiz(saved), nil
}

func (r *exchangeRateRepo) Resolve(ctx context.Context, organizationID uuid.UUID, rateType string, direction biz.OrderFeeDirection, fromCurrency, toCurrency, rateDate string) (*biz.ResolvedExchangeRate, error) {
	lookupTime, err := parseExchangeRateStorageTime(rateDate)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	items, err := r.data.db.ExchangeRateSetting.Query().Where(
		exchangerateent.OrganizationIDEQ(organizationID), exchangerateent.RateTypeEQ(exchangerateent.RateType(rateType)),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.IsActiveEQ(true),
		exchangerateent.EffectiveFromLTE(lookupTime), exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(lookupTime)),
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
	effectiveFrom := item.EffectiveFrom.In(biz.ExchangeRateBusinessLocation()).Format(time.RFC3339)
	var effectiveTo *string
	if item.EffectiveTo != nil {
		value := item.EffectiveTo.In(biz.ExchangeRateBusinessLocation()).Format(time.RFC3339)
		effectiveTo = &value
	}
	return &biz.ExchangeRateSetting{ID: item.ID, OrganizationID: item.OrganizationID, RateType: string(item.RateType), FromCurrency: item.FromCurrency, ToCurrency: item.ToCurrency, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, ReceivableRate: receivable, PayableRate: payable, IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

func exchangeRateCustomSettingToBiz(item *ent.ExchangeRateCustomSetting) *biz.ExchangeRateCustomSetting {
	updatedAt := item.UpdatedAt
	updatedBy := item.UpdatedBy
	return &biz.ExchangeRateCustomSetting{
		OrganizationID:          item.OrganizationID,
		InheritBaseCurrencyRate: item.InheritBaseCurrencyRate,
		Version:                 item.Version,
		UpdatedAt:               &updatedAt,
		UpdatedBy:               &updatedBy,
	}
}

func parseExchangeRateStorageTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil && parsed.Nanosecond() == 0 {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02", value, biz.ExchangeRateBusinessLocation())
}

var _ biz.ExchangeRateRepo = (*exchangeRateRepo)(nil)
