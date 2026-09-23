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

// ResolveContext 沿组织父链解析调用方所属公司及本币：汇率完全下沉分公司后仅公司根
// 可持有业务汇率，部门共享所属公司配置；系统节点、禁用组织、无本币公司及异常链
// 一律显式失败（系统工作台不读取、不维护业务汇率）。
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
		if item.Kind == organizationent.KindSystem {
			return nil, biz.ErrExchangeRateOrganizationInvalid
		}
		if item.Kind == organizationent.KindCompany {
			if item.ParentID != nil || item.BaseCurrency == nil || strings.TrimSpace(*item.BaseCurrency) == "" {
				return nil, biz.ErrExchangeRateOrganizationInvalid
			}
			return &biz.ExchangeRateContext{OwnerOrganizationID: item.ID, BaseCurrency: *item.BaseCurrency}, nil
		}
		if item.ParentID == nil {
			return nil, biz.ErrExchangeRateOrganizationInvalid
		}
		currentID = *item.ParentID
	}
}

// List 返回调用公司自己的周汇率行；公共基线行已退役，不再返回也不再展示。
// 分页按最新生效周降序优先，便于财务首屏查看最新汇率。
func (r *exchangeRateRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.ExchangeRateListOptions) ([]*biz.ExchangeRateSetting, int64, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	predicates := []predicate.ExchangeRateSetting{
		exchangerateent.OrganizationIDEQ(organizationID),
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

// UpsertWeeklyBatch 按自然周幂等写入公司汇率行：同公司同货币对同周（effective_from
// 锚定周一）命中即覆盖更新，不抛唯一键冲突；全部行必须归属同一家公司，缺失或
// 跨公司混写一律拒绝。
func (r *exchangeRateRepo) UpsertWeeklyBatch(ctx context.Context, source string, inputs []*biz.ExchangeRateSetting, audit *biz.AuditEvent) ([]*biz.ExchangeRateSetting, error) {
	if len(inputs) == 0 || inputs[0] == nil || inputs[0].OrganizationID == nil || *inputs[0].OrganizationID == uuid.Nil {
		return nil, biz.ErrExchangeRateInvalidArgument
	}
	scopeID := *inputs[0].OrganizationID
	for _, input := range inputs {
		if input == nil || input.OrganizationID == nil || *input.OrganizationID != scopeID {
			return nil, biz.ErrExchangeRateInvalidArgument
		}
		if err := r.validateCurrencies(ctx, input.FromCurrency, input.ToCurrency); err != nil {
			return nil, err
		}
	}
	// 公司作用域级 advisory 锁：同一公司的周写入串行化，避免并发 Upsert 与唯一
	// 索引冲突竞争。
	lockKey := fmt.Sprintf("exchange-rate-weekly:%s", scopeID)
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

// upsertWeeklyExchangeRate 在事务内按（公司, 货币对, 当周周一）命中即覆盖更新，
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
	existing, queryErr := tx.ExchangeRateSetting.Query().
		Where(
			exchangerateent.OrganizationIDEQ(*input.OrganizationID),
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

// UpdateScoped 按 ID 更新本公司汇率行；仅命中调用公司自己的行，周窗口与三轨汇率
// 整体覆盖。
func (r *exchangeRateRepo) UpdateScoped(ctx context.Context, callerOrganizationID uuid.UUID, input *biz.ExchangeRateSetting, audit *biz.AuditEvent) (*biz.ExchangeRateSetting, error) {
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
		current, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(input.ID), exchangerateent.OrganizationIDEQ(callerOrganizationID), exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
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

// DisableScoped 停用本公司汇率行。
func (r *exchangeRateRepo) DisableScoped(ctx context.Context, callerOrganizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.ExchangeRateSetting.Query().Where(exchangerateent.IDEQ(id), exchangerateent.OrganizationIDEQ(callerOrganizationID), exchangerateent.IsActiveEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrExchangeRateNotFound, nil)
		}
		if _, updateErr := item.Update().SetIsActive(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ResolveRate 只匹配业务日期所在自然周的本公司启用行（汇率完全下沉分公司，
// 不再回溯历史周、不再兜底公共基线）：RECEIVABLE→coalesce(ar_rate, rate)，
// PAYABLE→coalesce(ap_rate, rate)。未命中（含上周、未来周、历史开口行与公共行）
// 返回 ErrExchangeRateMissing，由前端提示「请先维护汇率」或按权限手工覆盖（MANUAL）。
func (r *exchangeRateRepo) ResolveRate(ctx context.Context, organizationID uuid.UUID, direction biz.OrderFeeDirection, fromCurrency, toCurrency, rateDate string) (biz.ResolvedRate, error) {
	if fromCurrency == toCurrency {
		return biz.ResolvedRate{Rate: decimal.NewFromInt(1), Source: biz.ExchangeRateSourceSystem}, nil
	}
	lookup, err := parseExchangeRateStorageTime(rateDate)
	if err != nil {
		return biz.ResolvedRate{}, biz.ErrExchangeRateInvalidArgument
	}
	rate, settingID, rowSource, found, err := r.resolveWeeklyRow(ctx, organizationID, direction, fromCurrency, toCurrency, lookup)
	if err != nil {
		return biz.ResolvedRate{}, err
	}
	if !found {
		return biz.ResolvedRate{}, biz.ErrExchangeRateMissing
	}
	return biz.ResolvedRate{Rate: rate, Source: weeklySnapshotSource(rowSource), SettingID: settingID}, nil
}

// weeklySnapshotSource 当周公司行的快照来源：牌价同步行标记 BOC_SYNC，其余标记 WEEKLY。
func weeklySnapshotSource(rowSource string) string {
	if rowSource == biz.ExchangeRateSettingSourceBOCSync {
		return biz.ExchangeRateSourceBOCSync
	}
	return biz.ExchangeRateSourceWeekly
}

// resolveWeeklyRow 解析业务日期所在自然周的单条公司汇率行。行只在其自身自然周内
// 生效：写入路径恒把 effective_from 锚定到周一 00:00:00，历史开口行不得跨周命中，
// 因此按 effective_from == 业务日期所在周周一精确匹配，与周幂等写入键一致。
// 命中至多一行——部分唯一索引按周生效，同周多行视为脏数据冲突（fail-closed）。
// 事务内读取加 ForShare 共享锁，保证并发修改汇率不影响同事务内已解析的账单快照。
func (r *exchangeRateRepo) resolveWeeklyRow(ctx context.Context, organizationID uuid.UUID, direction biz.OrderFeeDirection, fromCurrency, toCurrency string, lookup time.Time) (decimal.Decimal, *uuid.UUID, string, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return decimal.Decimal{}, nil, "", false, err
	}
	weekFrom, _ := biz.ExchangeRateWeekWindow(lookup)
	query := client.ExchangeRateSetting.Query().Where(
		exchangerateent.OrganizationIDEQ(organizationID),
		exchangerateent.FromCurrencyEQ(fromCurrency), exchangerateent.ToCurrencyEQ(toCurrency),
		exchangerateent.IsActiveEQ(true),
		exchangerateent.EffectiveFromEQ(weekFrom),
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
