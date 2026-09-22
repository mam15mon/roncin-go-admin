package data

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	exchangerateent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
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
	visited := map[uuid.UUID]struct{}{}
	for {
		if _, exists := visited[currentID]; exists {
			return nil, biz.ErrExchangeRateOrganizationInvalid
		}
		visited[currentID] = struct{}{}
		query := client.Organization.Query().Where(organizationent.IDEQ(currentID), organizationent.EnabledEQ(true))
		if _, transactional := transactionFromContext(ctx); transactional {
			query.ForShare()
		}
		item, err := query.Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrExchangeRateOrganizationInvalid, nil)
		}
		if item.Kind == organizationent.KindCompany || item.Kind == organizationent.KindSystem {
			if item.ParentID != nil || item.BaseCurrency == nil || strings.TrimSpace(*item.BaseCurrency) == "" {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			// 公司拥有业务汇率；公共参考汇率的交叉币种独立取系统配置，不依赖组织祖先。
			systemQuery := client.Organization.Query().Where(organizationent.KindEQ(organizationent.KindSystem), organizationent.EnabledEQ(true))
			if _, transactional := transactionFromContext(ctx); transactional {
				systemQuery.ForShare()
			}
			system, err := systemQuery.Only(ctx)
			if err != nil {
				return nil, mapEntError(err, biz.ErrExchangeRateOrganizationInvalid, nil)
			}
			if system.BaseCurrency == nil || strings.TrimSpace(*system.BaseCurrency) == "" {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return &biz.ExchangeRateContext{OwnerOrganizationID: item.ID, BaseCurrency: *item.BaseCurrency, PivotCurrency: *system.BaseCurrency}, nil
		}
		if item.ParentID == nil {
			return nil, biz.ErrExchangeRateOrganizationInvalid
		}
		currentID = *item.ParentID
	}
}

// List 返回调用方组织行 + 公共参考行；维护入口按行归属呈现（基线行对公司工作台只读）。
// 分页按最新生效周降序优先，便于财务首屏查看最新汇率。
func (r *exchangeRateRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.ExchangeRateListOptions) ([]*biz.ExchangeRateSetting, int64, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	predicates := []predicate.ExchangeRateSetting{
		exchangerateent.Or(
			exchangerateent.OrganizationIDEQ(organizationID),
			exchangerateent.OrganizationIDIsNil(),
		),
	}
	if options.FromCurrency != "" {
		predicates = append(predicates, exchangerateent.FromCurrencyEQ(options.FromCurrency))
	}
	query := client.ExchangeRateSetting.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.
		Order(
			exchangerateent.ByEffectiveFrom(entsql.OrderDesc()),
			exchangerateent.ByFromCurrency(),
			exchangerateent.ByID(),
		).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.ExchangeRateSetting, 0, len(items))
	for _, item := range items {
		converted, err := exchangeRateToBiz(item)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, converted)
	}
	return result, int64(total), nil
}

// UpsertWeeklyBatch 按自然周幂等写入汇率行：同作用域同货币对同周（effective_from
// 锚定周一）命中即覆盖更新，不抛唯一键冲突；行归属由 input.OrganizationID 决定
// （NULL=基线行，非空=组织行），重叠校验同域化（基线行只与基线行比对）。
func (r *exchangeRateRepo) UpsertWeeklyBatch(ctx context.Context, source string, inputs []*biz.ExchangeRateSetting, audit *biz.AuditEvent) ([]*biz.ExchangeRateSetting, error) {
	if len(inputs) == 0 {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	for _, input := range inputs {
		if err := r.validateCurrencies(ctx, input.FromCurrency, input.ToCurrency); err != nil {
			return nil, err
		}
	}
	// 作用域级 advisory 锁：同一作用域（基线/组织）的周写入串行化，避免并发
	// Upsert 与唯一索引冲突竞争。
	scope := "baseline"
	if inputs[0].OrganizationID != nil {
		scope = inputs[0].OrganizationID.String()
	}
	lockKey := fmt.Sprintf("exchange-rate-weekly:%s", scope)
	connection, err := r.data.sqlDB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext($1))", lockKey); err != nil {
		return nil, err
	}
	defer connection.ExecContext(context.Background(), "SELECT pg_advisory_unlock(hashtext($1))", lockKey)

	// 固定加锁顺序：按货币对排序后逐行 ForUpdate，防止死锁。
	ordered := append([]*biz.ExchangeRateSetting(nil), inputs...)
	sort.Slice(ordered, func(i, j int) bool {
		left, right := ordered[i], ordered[j]
		if left.FromCurrency != right.FromCurrency {
			return left.FromCurrency < right.FromCurrency
		}
		return left.ToCurrency < right.ToCurrency
	})
	saved := make([]*ent.ExchangeRateSetting, 0, len(ordered))
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		saved = saved[:0]
		for _, input := range ordered {
			item, err := upsertWeeklyExchangeRate(ctx, tx, input, source)
			if err != nil {
				return err
			}
			saved = append(saved, item)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	result := make([]*biz.ExchangeRateSetting, 0, len(saved))
	for _, item := range saved {
		converted, err := exchangeRateToBiz(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

// upsertWeeklyExchangeRate 在事务内按（作用域, 货币对, 当周周一）命中即覆盖更新，
// 未命中则插入新行；命中行（含已停用）恢复启用并刷新周窗口与三轨汇率。
func upsertWeeklyExchangeRate(ctx context.Context, tx *ent.Tx, input *biz.ExchangeRateSetting, source string) (*ent.ExchangeRateSetting, error) {
	effectiveFrom, err := parseExchangeRateStorageTime(input.EffectiveFrom)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	effectiveTo, err := parseExchangeRateStorageTime(*input.EffectiveTo)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	scopePredicate := scopePredicateFor(input.OrganizationID)
	existing, queryErr := tx.ExchangeRateSetting.Query().
		Where(
			scopePredicate,
			exchangerateent.FromCurrencyEQ(input.FromCurrency), exchangerateent.ToCurrencyEQ(input.ToCurrency),
			exchangerateent.EffectiveFromEQ(effectiveFrom),
		).
		ForUpdate().
		First(ctx)
	if queryErr != nil && !ent.IsNotFound(queryErr) {
		return nil, queryErr
	}
	if existing != nil {
		updated, updateErr := existing.Update().
			SetEffectiveTo(effectiveTo).
			SetArRate(input.ARRate.StringFixed(8)).
			SetApRate(input.APRate.StringFixed(8)).
			SetRate(input.Rate.StringFixed(8)).
			SetSource(exchangerateent.Source(source)).
			SetIsActive(true).
			Save(ctx)
		if updateErr != nil {
			return nil, updateErr
		}
		return updated, nil
	}
	created, createErr := tx.ExchangeRateSetting.Create().
		SetID(input.ID).
		SetNillableOrganizationID(input.OrganizationID).
		SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
		SetEffectiveFrom(effectiveFrom).SetEffectiveTo(effectiveTo).
		SetArRate(input.ARRate.StringFixed(8)).
		SetApRate(input.APRate.StringFixed(8)).
		SetRate(input.Rate.StringFixed(8)).
		SetSource(exchangerateent.Source(source)).
		SetIsActive(true).
		Save(ctx)
	if createErr != nil {
		return nil, mapEntError(createErr, nil, biz.ErrExchangeRateInvalidArgument)
	}
	return created, nil
}

// scopePredicateFor 行作用域谓词：NULL 输入限定基线行，非空限定对应组织行。
func scopePredicateFor(organizationID *uuid.UUID) predicate.ExchangeRateSetting {
	if organizationID == nil {
		return exchangerateent.OrganizationIDIsNil()
	}
	return exchangerateent.OrganizationIDEQ(*organizationID)
}

// UpdateScoped 按 ID 更新汇率行；allowBaseline 时系统工作台可命中 NULL 基线行，
// 否则仅限调用组织自己的组织行。周窗口与三轨汇率整体覆盖。
func (r *exchangeRateRepo) UpdateScoped(ctx context.Context, callerOrganizationID uuid.UUID, input *biz.ExchangeRateSetting, allowBaseline bool, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
	effectiveFrom, err := parseExchangeRateStorageTime(input.EffectiveFrom)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	effectiveTo, err := parseExchangeRateStorageTime(*input.EffectiveTo)
	if err != nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	var saved *ent.ExchangeRateSetting
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		scope := exchangerateent.Or(
			exchangerateent.OrganizationIDEQ(callerOrganizationID),
			exchangerateent.OrganizationIDIsNil(),
		)
		if !allowBaseline {
			scope = exchangerateent.OrganizationIDEQ(callerOrganizationID)
		}
		current, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(input.ID), scope, exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
		}
		updated, updateErr := current.Update().
			SetFromCurrency(input.FromCurrency).SetToCurrency(input.ToCurrency).
			SetEffectiveFrom(effectiveFrom).SetEffectiveTo(effectiveTo).
			SetArRate(input.ARRate.StringFixed(8)).
			SetApRate(input.APRate.StringFixed(8)).
			SetRate(input.Rate.StringFixed(8)).
			Save(ctx)
		if updateErr != nil {
			// 目标周已被同作用域同货币对的另一行占用（部分唯一索引冲突）映射业务错误。
			return mapEntError(updateErr, nil, biz.ErrExchangeRateOverlap)
		}
		saved = updated
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return exchangeRateToBiz(saved)
}

// DisableScoped 停用汇率行，作用域语义同 UpdateScoped。
func (r *exchangeRateRepo) DisableScoped(ctx context.Context, callerOrganizationID, id uuid.UUID, allowBaseline bool, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		scope := exchangerateent.Or(
			exchangerateent.OrganizationIDEQ(callerOrganizationID),
			exchangerateent.OrganizationIDIsNil(),
		)
		if !allowBaseline {
			scope = exchangerateent.OrganizationIDEQ(callerOrganizationID)
		}
		item, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(id), scope, exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
		}
		if _, updateErr := item.Update().SetIsActive(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ResolveRate 四级容灾解析（绝不阻断单据保存）：
//  1. 本组织当周行（自然周窗口覆盖目标日期）：快照源 WEEKLY / BOC_SYNC（按行 source）；
//  2. 回溯最近一个有效历史周的组织行：快照源 INHERITED_LAST_WEEK；
//  3. NULL 基线行直连（当周 → 回溯最近历史基线周，公共兜底同样不卡单）：快照源 SYSTEM；
//  4. NULL 基线行经基准币交叉套算（每腿按三级同款基线解析）：快照源 DERIVED。
//
// 汇率值按费用收支方向取列：RECEIVABLE→coalesce(ar_rate, rate)，PAYABLE→coalesce(ap_rate, rate)。
// 全链未命中返回 ErrExchangeRateMissing，由现场手工覆盖（MANUAL）兜底。
func (r *exchangeRateRepo) ResolveRate(ctx context.Context, organizationID uuid.UUID, direction biz.OrderFeeDirection, fromCurrency, toCurrency, pivotCurrency, rateDate string) (biz.ResolvedRate, error) {
	if fromCurrency == toCurrency {
		return biz.ResolvedRate{Rate: decimal.NewFromInt(1), Source: biz.ExchangeRateSourceSystem}, nil
	}
	lookup, err := parseExchangeRateStorageTime(rateDate)
	if err != nil {
		return biz.ResolvedRate{}, biz.ErrExchangeRateInvalidArgument
	}
	// 一级：本组织当周行。
	rate, settingID, rowSource, found, err := r.resolveWeeklyRow(ctx, &organizationID, direction, fromCurrency, toCurrency, lookup)
	if err != nil {
		return biz.ResolvedRate{}, err
	}
	if found {
		return biz.ResolvedRate{Rate: rate, Source: weeklySnapshotSource(rowSource), SettingID: settingID}, nil
	}
	// 二级：回溯最近一个有效历史周（继承上周，不阻断保存）。
	rate, settingID, _, found, err = r.resolveInheritedWeeklyRow(ctx, &organizationID, direction, fromCurrency, toCurrency, lookup)
	if err != nil {
		return biz.ResolvedRate{}, err
	}
	if found {
		return biz.ResolvedRate{Rate: rate, Source: biz.ExchangeRateSourceInheritedLastWeek, SettingID: settingID}, nil
	}
	// 三级：NULL 基线行直连（覆盖行缺失时同样回溯最近历史基线周），快照携带命中行。
	rate, settingID, found, err = r.resolveBaselineRowWithHistory(ctx, direction, fromCurrency, toCurrency, lookup)
	if err != nil {
		return biz.ResolvedRate{}, err
	}
	if found {
		return biz.ResolvedRate{Rate: rate, Source: biz.ExchangeRateSourceSystem, SettingID: settingID}, nil
	}
	// 四级：NULL 基线行经基准币交叉套算，每腿按三级同款基线解析（当周 → 回溯历史）。
	legFrom := decimal.NewFromInt(1)
	if fromCurrency != pivotCurrency {
		legFrom, err = r.resolveBaselineLeg(ctx, direction, fromCurrency, pivotCurrency, lookup)
		if err != nil {
			return biz.ResolvedRate{}, err
		}
	}
	legTo := decimal.NewFromInt(1)
	if toCurrency != pivotCurrency {
		legTo, err = r.resolveBaselineLeg(ctx, direction, toCurrency, pivotCurrency, lookup)
		if err != nil {
			return biz.ResolvedRate{}, err
		}
	}
	if !legTo.IsPositive() {
		return biz.ResolvedRate{}, biz.ErrExchangeRateMissing
	}
	return biz.ResolvedRate{Rate: legFrom.Div(legTo).RoundBank(8), Source: biz.ExchangeRateSourceDerived}, nil
}

// resolveBaselineLeg 解析套算单腿：NULL 基线覆盖行 → 回溯最近历史基线周；均缺失
// 返回 ErrExchangeRateMissing（fail-closed，不做静默 1）。
func (r *exchangeRateRepo) resolveBaselineLeg(ctx context.Context, direction biz.OrderFeeDirection, fromCurrency, toCurrency string, lookup time.Time) (decimal.Decimal, error) {
	rate, _, found, err := r.resolveBaselineRowWithHistory(ctx, direction, fromCurrency, toCurrency, lookup)
	if err != nil {
		return decimal.Decimal{}, err
	}
	if !found {
		return decimal.Decimal{}, biz.ErrExchangeRateMissing
	}
	return rate, nil
}

// weeklySnapshotSource 当周组织行的快照来源：牌价同步行标记 BOC_SYNC，其余标记 WEEKLY。
func weeklySnapshotSource(rowSource string) string {
	if rowSource == biz.ExchangeRateSettingSourceBOCSync {
		return biz.ExchangeRateSourceBOCSync
	}
	return biz.ExchangeRateSourceWeekly
}

// resolveWeeklyRow 解析单条覆盖目标时刻的汇率行（作用域限定：orgID 非空为组织行，
// nil 为基线行）。命中至多一行——部分唯一索引按周生效，同作用域同周多行视为
// 脏数据冲突（fail-closed）。事务内读取加 ForShare 共享锁，保证并发修改汇率
// 不影响同事务内已解析的账单快照。
func (r *exchangeRateRepo) resolveWeeklyRow(ctx context.Context, organizationID *uuid.UUID, direction biz.OrderFeeDirection, fromCurrency, toCurrency string, lookup time.Time) (decimal.Decimal, *uuid.UUID, string, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return decimal.Decimal{}, nil, "", false, err
	}
	query := client.ExchangeRateSetting.Query().Where(
		scopePredicateFor(organizationID),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.IsActiveEQ(true),
		// 周窗口为闭区间：周一 00:00:00 ≤ lookup ≤ 周日 23:59:59（兼容历史开口行）。
		exchangerateent.EffectiveFromLTE(lookup),
		exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToGTE(lookup)),
	).Limit(2)
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	items, err := query.All(ctx)
	if err != nil {
		return decimal.Decimal{}, nil, "", false, err
	}
	if len(items) > 1 {
		return decimal.Decimal{}, nil, "", false, biz.ErrExchangeRateConflict
	}
	if len(items) == 0 {
		return decimal.Decimal{}, nil, "", false, nil
	}
	rate, err := directionalRate(items[0], direction)
	if err != nil {
		return decimal.Decimal{}, nil, "", false, err
	}
	settingID := items[0].ID
	return rate, &settingID, string(items[0].Source), true, nil
}

// resolveInheritedWeeklyRow 回溯最近一个有效历史周的汇率行（整周已结束于目标时刻
// 之前，按 effective_from 倒序取最近一周）；作用域限定同 resolveWeeklyRow，
// 事务内读取加 ForShare。
func (r *exchangeRateRepo) resolveInheritedWeeklyRow(ctx context.Context, organizationID *uuid.UUID, direction biz.OrderFeeDirection, fromCurrency, toCurrency string, lookup time.Time) (decimal.Decimal, *uuid.UUID, string, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return decimal.Decimal{}, nil, "", false, err
	}
	query := client.ExchangeRateSetting.Query().Where(
		scopePredicateFor(organizationID),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.IsActiveEQ(true),
		exchangerateent.EffectiveFromLT(lookup),
		exchangerateent.Or(exchangerateent.EffectiveToIsNil(), exchangerateent.EffectiveToLT(lookup)),
	).Order(exchangerateent.ByEffectiveFrom(entsql.OrderDesc()))
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	item, err := query.First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return decimal.Decimal{}, nil, "", false, nil
		}
		return decimal.Decimal{}, nil, "", false, err
	}
	rate, err := directionalRate(item, direction)
	if err != nil {
		return decimal.Decimal{}, nil, "", false, err
	}
	settingID := item.ID
	return rate, &settingID, string(item.Source), true, nil
}

// resolveBaselineRowWithHistory NULL 基线行兜底解析：先取覆盖目标时刻的行，
// 缺失时回溯最近一个有效历史基线周（公共兜底与组织行同享容灾，不卡单据）。
func (r *exchangeRateRepo) resolveBaselineRowWithHistory(ctx context.Context, direction biz.OrderFeeDirection, fromCurrency, toCurrency string, lookup time.Time) (decimal.Decimal, *uuid.UUID, bool, error) {
	rate, settingID, _, found, err := r.resolveWeeklyRow(ctx, nil, direction, fromCurrency, toCurrency, lookup)
	if err != nil || found {
		return rate, settingID, found, err
	}
	rate, settingID, _, found, err = r.resolveInheritedWeeklyRow(ctx, nil, direction, fromCurrency, toCurrency, lookup)
	return rate, settingID, found, err
}

// directionalRate 按收支方向取汇率值：应收取 ar_rate（缺省回落 rate 基准价），
// 应付取 ap_rate（缺省回落 rate）。
func directionalRate(item *ent.ExchangeRateSetting, direction biz.OrderFeeDirection) (decimal.Decimal, error) {
	preferred := item.ArRate
	if direction == biz.OrderFeePayable {
		preferred = item.ApRate
	}
	if preferred != nil && strings.TrimSpace(*preferred) != "" {
		return decimalOf(*preferred)
	}
	return decimalOf(item.Rate)
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

// EnabledCurrencyCodes 返回全部启用币种代码（同步抓取与发布校验共用）。
func (r *exchangeRateRepo) EnabledCurrencyCodes(ctx context.Context) ([]string, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.Currency.Query().Where(currencyent.EnabledEQ(true)).Order(currencyent.ByCode()).All(ctx)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(items))
	for _, item := range items {
		codes = append(codes, item.Code)
	}
	return codes, nil
}

func exchangeRateToBiz(item *ent.ExchangeRateSetting) (*biz.ExchangeRateSetting, error) {
	rate, err := decimalOf(item.Rate)
	if err != nil {
		return nil, err
	}
	var arRate, apRate *decimal.Decimal
	if item.ArRate != nil {
		value, err := decimalOf(*item.ArRate)
		if err != nil {
			return nil, err
		}
		arRate = &value
	}
	if item.ApRate != nil {
		value, err := decimalOf(*item.ApRate)
		if err != nil {
			return nil, err
		}
		apRate = &value
	}
	effectiveFrom := item.EffectiveFrom.In(biz.ExchangeRateBusinessLocation()).Format(time.RFC3339)
	var effectiveTo *string
	if item.EffectiveTo != nil {
		value := item.EffectiveTo.In(biz.ExchangeRateBusinessLocation()).Format(time.RFC3339)
		effectiveTo = &value
	}
	return &biz.ExchangeRateSetting{ID: item.ID, OrganizationID: item.OrganizationID, FromCurrency: item.FromCurrency, ToCurrency: item.ToCurrency, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, ARRate: arRate, APRate: apRate, Rate: rate, Source: string(item.Source), IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}

func parseExchangeRateStorageTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil && parsed.Nanosecond() == 0 {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02", value, biz.ExchangeRateBusinessLocation())
}

var _ biz.ExchangeRateRepo = (*exchangeRateRepo)(nil)
