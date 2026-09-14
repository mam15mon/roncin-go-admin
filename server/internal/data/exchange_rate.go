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

// List 返回调用方组织行 + 集团基线行；维护入口按行归属呈现（阶段二开放组织行维护）。
func (r *exchangeRateRepo) List(ctx context.Context, organizationID uuid.UUID) ([]*biz.ExchangeRateSetting, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.ExchangeRateSetting.Query().
		Where(exchangerateent.Or(
			exchangerateent.OrganizationIDEQ(organizationID),
			exchangerateent.OrganizationIDIsNil(),
		)).
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

func (r *exchangeRateRepo) Create(ctx context.Context, _ uuid.UUID, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	if input.OrganizationID != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return r.save(ctx, input, audit, false)
}

func (r *exchangeRateRepo) Update(ctx context.Context, _ uuid.UUID, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	if input.OrganizationID != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	return r.save(ctx, input, audit, true)
}

// save 写入基线行（organization_id IS NULL）。阶段一仅总部可写（biz 层拦截器已校验）；
// 组织行落位与重叠校验同域化随阶段二开放。
func (r *exchangeRateRepo) save(ctx context.Context, input *biz.ExchangeRateSetting, audit *biz.AuditEvent, updating bool) (*biz.ExchangeRateSetting, error) {
	if err := r.validateCurrencies(ctx, input.FromCurrency, input.ToCurrency); err != nil {
		return nil, err
	}
	lockKey := fmt.Sprintf("exchange-rate:%s:%s:%s", "baseline", input.FromCurrency, input.ToCurrency)
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
			current, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(input.ID), exchangerateent.OrganizationIDIsNil()).ForUpdate().Only(ctx)
			if queryErr != nil {
				return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
			}
			if !current.IsActive {
				return biz.ErrExchangeRateNotFound
			}
		}
		// 重叠校验同域：基线行只与基线行比对。
		conflict := tx.ExchangeRateSetting.Query().Where(
			exchangerateent.OrganizationIDIsNil(),
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
			builder := tx.ExchangeRateSetting.Create().SetID(input.ID).
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

func (r *exchangeRateRepo) Disable(ctx context.Context, _ uuid.UUID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(id), exchangerateent.OrganizationIDIsNil(), exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
		}
		if _, updateErr := item.Update().SetIsActive(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ResolveRate 直连优先解析：本组织行直连（SYSTEM）优先于基线行直连（SYSTEM），
// 由 BaselineOrLocal 点查形态 + 本组织行优先排序表达；直连缺失时经基线基准币单跳
// 交叉套算 from→to = (from→pivot) ÷ (to→pivot)，来源 DERIVED。orgID 为 uuid.Nil
// 时仅解析基线行（跨组织资金流结算域，跳过一切组织行）。
// 任一腿缺失或 to 腿非正数均视为缺失（fail-closed，不做日期回退或静默 1）；
// from == to 防御性恒为 1。
func (r *exchangeRateRepo) ResolveRate(ctx context.Context, organizationID uuid.UUID, fromCurrency, toCurrency, pivotCurrency, rateDate string) (biz.ResolvedRate, error) {
	if fromCurrency == toCurrency {
		return biz.ResolvedRate{Rate: decimal.NewFromInt(1), Source: biz.ExchangeRateSourceSystem}, nil
	}
	direct, err := r.resolveDirectRate(ctx, organizationID, fromCurrency, toCurrency, rateDate)
	if err == nil {
		return biz.ResolvedRate{Rate: direct, Source: biz.ExchangeRateSourceSystem}, nil
	}
	if !errors.Is(err, biz.ErrExchangeRateMissing) {
		return biz.ResolvedRate{}, err
	}
	legFrom := decimal.NewFromInt(1)
	if fromCurrency != pivotCurrency {
		legFrom, err = r.resolveDirectRate(ctx, organizationID, fromCurrency, pivotCurrency, rateDate)
		if err != nil {
			return biz.ResolvedRate{}, err
		}
	}
	legTo := decimal.NewFromInt(1)
	if toCurrency != pivotCurrency {
		legTo, err = r.resolveDirectRate(ctx, organizationID, toCurrency, pivotCurrency, rateDate)
		if err != nil {
			return biz.ResolvedRate{}, err
		}
	}
	if !legTo.IsPositive() {
		return biz.ResolvedRate{}, biz.ErrExchangeRateMissing
	}
	return biz.ResolvedRate{Rate: legFrom.Div(legTo).RoundBank(8), Source: biz.ExchangeRateSourceDerived}, nil
}

// resolveDirectRate 查询单条直连汇率行：本组织行优先于基线行（同码覆盖），
// 有效区间覆盖 rateDate 的启用行，未命中返回 ErrExchangeRateMissing。
// 部分唯一索引按 (organization_id, from_currency, to_currency, effective_from)
// 与基线/组织两个作用域分别约束 effective_from 唯一，但同一作用域内历史任意区间行
// 仍可能同时覆盖同一日期：同作用域命中多行视为脏数据冲突（fail-closed）；
// 本组织行与基线行并存属预期遮蔽（Shadowing），本组织行优先，不算冲突。
func (r *exchangeRateRepo) resolveDirectRate(ctx context.Context, organizationID uuid.UUID, fromCurrency, toCurrency, rateDate string) (decimal.Decimal, error) {
	lookupTime, err := parseExchangeRateStorageTime(rateDate)
	if err != nil {
		return decimal.Decimal{}, biz.ErrExchangeRateInvalidArgument
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return decimal.Decimal{}, err
	}
	// 本组织行 + 基线行各至多取 1 行，第 3 行仅用于同作用域冲突探测。
	query := client.ExchangeRateSetting.Query().Where(
		exchangeRateBaselineScope(organizationID),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.IsActiveEQ(true),
		exchangerateent.EffectiveFromLTE(lookupTime), exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGT(lookupTime)),
	)
	if organizationID != uuid.Nil {
		// 本组织行优先；uuid.Nil 表示仅解析基线行，无需排序。
		query.Order(exchangeRateLocalFirstOrder(organizationID))
	}
	query.Limit(3)
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	items, err := query.All(ctx)
	if err != nil {
		return decimal.Decimal{}, err
	}
	var localRow, baselineRow *ent.ExchangeRateSetting
	localCount, baselineCount := 0, 0
	for _, item := range items {
		if item.OrganizationID != nil && organizationID != uuid.Nil && *item.OrganizationID == organizationID {
			localCount++
			localRow = item
			continue
		}
		baselineCount++
		baselineRow = item
	}
	switch {
	case localCount > 1, baselineCount > 1:
		return decimal.Decimal{}, biz.ErrExchangeRateConflict
	case localRow != nil:
		return decimalOf(localRow.Rate)
	case baselineRow != nil:
		return decimalOf(baselineRow.Rate)
	default:
		return decimal.Decimal{}, biz.ErrExchangeRateMissing
	}
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
