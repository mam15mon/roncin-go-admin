package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r *exchangeRateRepo) headquartersOrganizationID(ctx context.Context, organizationID uuid.UUID) (uuid.UUID, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return resolveHeadquartersOrganizationID(ctx, client.Organization, organizationID)
}

// requireHeadquarters 校验调用组织即总部；折本币基准汇率只允许总部写入，不提供重定向通道。
func (r *exchangeRateRepo) requireHeadquarters(ctx context.Context, organizationID uuid.UUID) error {
	headquartersID, err := r.headquartersOrganizationID(ctx, organizationID)
	if err != nil {
		return err
	}
	if headquartersID != organizationID {
		return biz.ErrExchangeRateHeadquartersRequired
	}
	return nil
}

func (r *exchangeRateRepo) ResolveContext(ctx context.Context, organizationID uuid.UUID) (*biz.ExchangeRateContext, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	currentID := organizationID
	baseCurrency := ""
	for {
		query := client.Organization.Query().Where(organizationent.IDEQ(currentID), organizationent.EnabledEQ(true))
		if _, transactional := transactionFromContext(ctx); transactional {
			query.ForShare()
		}
		item, err := query.Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrExchangeRateOrganizationInvalid, nil)
		}
		if baseCurrency == "" && item.BaseCurrency != nil {
			baseCurrency = *item.BaseCurrency
		}
		if item.ParentID == nil {
			if item.Kind != organizationent.KindHeadquarters || baseCurrency == "" {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			pivotCurrency := ""
			if item.BaseCurrency != nil {
				pivotCurrency = *item.BaseCurrency
			}
			if pivotCurrency == "" {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return &biz.ExchangeRateContext{OwnerOrganizationID: item.ID, BaseCurrency: baseCurrency, PivotCurrency: pivotCurrency}, nil
		}
		currentID = *item.ParentID
	}
}

func (r *exchangeRateRepo) List(ctx context.Context, organizationID uuid.UUID) ([]*biz.ExchangeRateSetting, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.ExchangeRateSetting.Query().
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

func (r *exchangeRateRepo) Create(ctx context.Context, organizationID uuid.UUID, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	if organizationID != input.OrganizationID {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return r.save(ctx, input, audit, false)
}

func (r *exchangeRateRepo) Update(ctx context.Context, organizationID uuid.UUID, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	if organizationID != input.OrganizationID {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return r.save(ctx, input, audit, true)
}

func (r *exchangeRateRepo) save(ctx context.Context, input *biz.ExchangeRateSetting, audit *biz.AuditEvent, updating bool) (*biz.ExchangeRateSetting, error) {
	if err := r.requireHeadquarters(ctx, input.OrganizationID); err != nil {
		return nil, err
	}
	if err := r.validateCurrencies(ctx, input.FromCurrency, input.ToCurrency); err != nil {
		return nil, err
	}
	lockKey := fmt.Sprintf("exchange-rate:%s:%s:%s", input.OrganizationID, input.FromCurrency, input.ToCurrency)
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
				return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
			}
			if !current.IsActive {
				return biz.ErrExchangeRateNotFound
			}
		}
		conflict := tx.ExchangeRateSetting.Query().Where(
			exchangerateent.OrganizationIDEQ(input.OrganizationID),
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
				SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
				SetEffectiveFrom(effectiveFrom).
				SetRate(input.Rate.StringFixed(8))
			if input.EffectiveTo == nil {
				builder.ClearEffectiveTo()
			} else {
				builder.SetEffectiveTo(*effectiveTo)
			}
			saved, saveErr = builder.Save(ctx)
		} else {
			builder := tx.ExchangeRateSetting.Create().SetID(input.ID).SetOrganizationID(input.OrganizationID).
				SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
				SetEffectiveFrom(effectiveFrom).
				SetRate(input.Rate.StringFixed(8)).SetIsActive(true)
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
	if err := r.requireHeadquarters(ctx, organizationID); err != nil {
		return err
	}
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(id), exchangerateent.OrganizationIDEQ(organizationID), exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
		}
		if _, updateErr := item.Update().SetIsActive(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ResolveRate 直连优先解析总部基准汇率：财务显式维护的直连行永远优先（来源 SYSTEM）；
// 直连缺失时经总部基准币单跳交叉套算 from→to = (from→pivot) ÷ (to→pivot)，来源 DERIVED。
// 任一腿缺失或 to 腿非正数均视为缺失（fail-closed，不做日期回退或静默 1）；
// 直连或任一腿命中多行沿用 ErrExchangeRateConflict；from == to 防御性恒为 1。
func (r *exchangeRateRepo) ResolveRate(ctx context.Context, ownerOrganizationID uuid.UUID, fromCurrency, toCurrency, pivotCurrency, rateDate string) (biz.ResolvedRate, error) {
	if fromCurrency == toCurrency {
		return biz.ResolvedRate{Rate: decimal.NewFromInt(1), Source: biz.ExchangeRateSourceSystem}, nil
	}
	direct, err := r.resolveDirectRate(ctx, ownerOrganizationID, fromCurrency, toCurrency, rateDate)
	if err == nil {
		return biz.ResolvedRate{Rate: direct, Source: biz.ExchangeRateSourceSystem}, nil
	}
	if !errors.Is(err, biz.ErrExchangeRateMissing) {
		return biz.ResolvedRate{}, err
	}
	legFrom := decimal.NewFromInt(1)
	if fromCurrency != pivotCurrency {
		legFrom, err = r.resolveDirectRate(ctx, ownerOrganizationID, fromCurrency, pivotCurrency, rateDate)
		if err != nil {
			return biz.ResolvedRate{}, err
		}
	}
	legTo := decimal.NewFromInt(1)
	if toCurrency != pivotCurrency {
		legTo, err = r.resolveDirectRate(ctx, ownerOrganizationID, toCurrency, pivotCurrency, rateDate)
		if err != nil {
			return biz.ResolvedRate{}, err
		}
	}
	if !legTo.IsPositive() {
		return biz.ResolvedRate{}, biz.ErrExchangeRateMissing
	}
	return biz.ResolvedRate{Rate: legFrom.Div(legTo).RoundBank(8), Source: biz.ExchangeRateSourceDerived}, nil
}

// resolveDirectRate 查询单条直连汇率行：有效区间覆盖 rateDate 的唯一启用行，
// 未命中返回 ErrExchangeRateMissing，多行命中返回 ErrExchangeRateConflict。
func (r *exchangeRateRepo) resolveDirectRate(ctx context.Context, ownerOrganizationID uuid.UUID, fromCurrency, toCurrency, rateDate string) (decimal.Decimal, error) {
	lookupTime, err := parseExchangeRateStorageTime(rateDate)
	if err != nil {
		return decimal.Decimal{}, biz.ErrExchangeRateInvalidArgument
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return decimal.Decimal{}, err
	}
	query := client.ExchangeRateSetting.Query().Where(
		exchangerateent.OrganizationIDEQ(ownerOrganizationID),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.IsActiveEQ(true),
		exchangerateent.EffectiveFromLTE(lookupTime), exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(lookupTime)),
	).Limit(2)
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	items, err := query.All(ctx)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if len(items) == 0 {
		return decimal.Decimal{}, biz.ErrExchangeRateMissing
	}
	if len(items) > 1 {
		return decimal.Decimal{}, biz.ErrExchangeRateConflict
	}
	return decimalOf(items[0].Rate)
}

func (r *exchangeRateRepo) validateCurrencies(ctx context.Context, codes ...string) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	count, err := client.Currency.Query().Where(currencyent.CodeIn(codes...), currencyent.EnabledEQ(true)).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(codes) {
		return biz.ErrExchangeRateCurrencyInvalid
	}
	return nil
}

func exchangeRateToBiz(item *ent.ExchangeRateSetting) (*biz.ExchangeRateSetting, error) {
	rate, err := decimalOf(item.Rate)
	if err != nil {
		return nil, err
	}
	effectiveFrom := item.EffectiveFrom.In(biz.ExchangeRateBusinessLocation()).Format(time.RFC3339)
	var effectiveTo *string
	if item.EffectiveTo != nil {
		value := item.EffectiveTo.In(biz.ExchangeRateBusinessLocation()).Format(time.RFC3339)
		effectiveTo = &value
	}
	return &biz.ExchangeRateSetting{ID: item.ID, OrganizationID: item.OrganizationID, FromCurrency: item.FromCurrency, ToCurrency: item.ToCurrency, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, Rate: rate, IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

func parseExchangeRateStorageTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil && parsed.Nanosecond() == 0 {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02", value, biz.ExchangeRateBusinessLocation())
}

var _ biz.ExchangeRateRepo = (*exchangeRateRepo)(nil)
