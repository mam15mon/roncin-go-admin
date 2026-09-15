package biz

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/shopspring/decimal"
)

var (
	ErrExchangeRateNotFound            = errors.NotFound("EXCHANGE_RATE_NOT_FOUND", "汇率设置不存在")
	ErrExchangeRateInvalidArgument     = errors.BadRequest("EXCHANGE_RATE_INVALID_ARGUMENT", "汇率设置字段不合法")
	ErrExchangeRateMissing             = errors.BadRequest(reasonFromProto(financev1.ErrorReason_ERROR_REASON_FEE_EXCHANGE_RATE_MISSING), "汇率日期未命中生效汇率")
	ErrExchangeRateOverlap             = errors.Conflict("EXCHANGE_RATE_OVERLAP", "汇率生效周与现有设置冲突，请刷新后重试")
	ErrExchangeRateConflict            = errors.Conflict("FEE_EXCHANGE_RATE_CONFLICT", "汇率日期命中多条生效汇率")
	ErrExchangeRateCurrencyInvalid     = errors.BadRequest("EXCHANGE_RATE_CURRENCY_INVALID", "汇率币种必须是启用的 ISO 币种")
	ErrExchangeRateOrganizationInvalid = errors.BadRequest("EXCHANGE_RATE_ORGANIZATION_INVALID", "当前组织未配置有效本币")
	ErrExchangeRatePermissionDenied    = errors.Forbidden("EXCHANGE_RATE_PERMISSION_DENIED", "无权维护当前组织汇率")
	ErrExchangeRateQuoteUnavailable    = errors.BadRequest("EXCHANGE_RATE_QUOTE_UNAVAILABLE", "外汇牌价抓取失败，请稍后重试或手工录入汇率")
	ErrExchangeRateSyncTargetInvalid   = errors.BadRequest("EXCHANGE_RATE_SYNC_TARGET_INVALID", "汇率同步目标周不合法")
	ErrExchangeRateSyncRowsInvalid     = errors.BadRequest("EXCHANGE_RATE_SYNC_ROWS_INVALID", "汇率同步数据不合法")
	ErrExchangeRateSyncRowsEmpty       = errors.BadRequest("EXCHANGE_RATE_SYNC_ROWS_EMPTY", "汇率同步数据为空")
)

var exchangeRateValuePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,9})(\.[0-9]{1,8})?$`)
var exchangeRateBusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func ExchangeRateBusinessLocation() *time.Location { return exchangeRateBusinessLocation }

// 汇率行写入来源（exchange_rate_settings.source）：费用快照来源据此区分
// WEEKLY 与 BOC_SYNC。
const (
	ExchangeRateSettingSourceManual  = "MANUAL"   // 页面手工维护
	ExchangeRateSettingSourceImport  = "IMPORT"   // Excel 导入
	ExchangeRateSettingSourceBOCSync = "BOC_SYNC" // 牌价一键同步
)

// 汇率快照来源；费用/账单/流水/发票的 exchange_rate_source 如实记录。
// 解析链：本组织当周行（WEEKLY/BOC_SYNC）→ 回溯最近历史周（INHERITED_LAST_WEEK）
// → NULL 基线行直连（SYSTEM）→ 基线交叉套算（DERIVED）；现场手工覆盖为 MANUAL。
const (
	ExchangeRateSourceSystem            = "SYSTEM"
	ExchangeRateSourceDerived           = "DERIVED"
	ExchangeRateSourceManual            = "MANUAL"
	ExchangeRateSourceWeekly            = "WEEKLY"
	ExchangeRateSourceInheritedLastWeek = "INHERITED_LAST_WEEK"
	ExchangeRateSourceBOCSync           = "BOC_SYNC"
)

// ExchangeRateSyncTarget 汇率同步目标自然周（本周/预设下周）。
type ExchangeRateSyncTarget string

const (
	ExchangeRateSyncTargetCurrentWeek = "CURRENT_WEEK"
	ExchangeRateSyncTargetNextWeek    = "NEXT_WEEK"
)

// ExchangeRateSyncWeekWindow 返回目标周的自然周窗口：周一 00:00:00 至周日
// 23:59:59（Asia/Shanghai）。本周锚定 now 所在周，预设下周顺延一个自然周。
func ExchangeRateSyncWeekWindow(target ExchangeRateSyncTarget, now time.Time) (time.Time, time.Time, bool) {
	from, to := ExchangeRateWeekWindow(now)
	switch target {
	case ExchangeRateSyncTargetCurrentWeek:
		return from, to, true
	case ExchangeRateSyncTargetNextWeek:
		return from.AddDate(0, 0, 7), to.AddDate(0, 0, 7), true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// ExchangeRateWeekWindow 返回包含 t 的自然周窗口：周一 00:00:00 至周日
// 23:59:59（Asia/Shanghai），两端均为闭区间。
func ExchangeRateWeekWindow(t time.Time) (time.Time, time.Time) {
	local := t.In(exchangeRateBusinessLocation)
	// time.Periodic 周一为 weekday=1；归一到本周周一 00:00:00。
	offset := (int(local.Weekday()) + 6) % 7
	monday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, exchangeRateBusinessLocation).AddDate(0, 0, -offset)
	sunday := monday.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return monday, sunday
}

// ExchangeRateSetting 是折本币周汇率：NULL 组织为集团基线行（总部维护的公共兜底），
// 非空为本组织行（各核算组织面向自身本币自治维护）。区间恒为单自然周。
type ExchangeRateSetting struct {
	ID uuid.UUID
	// OrganizationID 为空表示集团基线行（NULL），非空表示本组织行。
	OrganizationID *uuid.UUID
	FromCurrency   string
	ToCurrency     string
	EffectiveFrom  string
	EffectiveTo    *string
	// ARRate 为应收汇率（现汇卖出价），APRate 为应付汇率（现汇买入价）；
	// Rate 为基准汇率（中行折算价口径，内部审计与报表基准）。
	ARRate    *decimal.Decimal
	APRate    *decimal.Decimal
	Rate      decimal.Decimal
	Source    string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ExchangeRateContext 携带调用组织基准币种与总部本位币；BaseCurrency 是当前核算
// 组织的本币（汇率行 to_currency 必须等于它），PivotCurrency 是 NULL 基线行
// 交叉套算的基准币（总部本币）。
type ExchangeRateContext struct {
	OwnerOrganizationID uuid.UUID
	BaseCurrency        string
	PivotCurrency       string
}

// ResolvedRate 是汇率解析结果：值携带来源与命中行，供消费方快照区分
// 周行命中、跨周继承、基线兜底与手工覆盖。
type ResolvedRate struct {
	Rate      decimal.Decimal
	Source    string
	SettingID *uuid.UUID
}

type ExchangeRateListOptions struct {
	Page         int
	PageSize     int
	FromCurrency string
}

type ExchangeRateRepo interface {
	ResolveContext(ctx context.Context, organizationID uuid.UUID) (*ExchangeRateContext, error)
	List(ctx context.Context, organizationID uuid.UUID, options ExchangeRateListOptions) ([]*ExchangeRateSetting, int64, error)
	// UpsertWeeklyBatch 按自然周幂等写入汇率行（同作用域同货币对同周命中即覆盖更新，
	// 不报唯一键冲突），source 标记行写入来源。
	UpsertWeeklyBatch(ctx context.Context, source string, inputs []*ExchangeRateSetting, audit *AuditEvent) ([]*ExchangeRateSetting, error)
	// UpdateScoped 按 ID 更新汇率行；allowBaseline 时可命中 NULL 基线行（总部），
	// 否则仅限调用组织自己的组织行。
	UpdateScoped(ctx context.Context, callerOrganizationID uuid.UUID, input *ExchangeRateSetting, allowBaseline bool, audit *AuditEvent) (*ExchangeRateSetting, error)
	// DisableScoped 停用汇率行，作用域语义同 UpdateScoped。
	DisableScoped(ctx context.Context, callerOrganizationID, id uuid.UUID, allowBaseline bool, audit *AuditEvent) error
	// ResolveRate 四级容灾解析：本组织当周行（WEEKLY/BOC_SYNC）→ 回溯最近历史周
	// （INHERITED_LAST_WEEK）→ NULL 基线直连（SYSTEM）→ 基线交叉套算（DERIVED）；
	// 任一级都绝不阻断单据保存，全部未命中返回 ErrExchangeRateMissing。
	ResolveRate(ctx context.Context, organizationID uuid.UUID, direction OrderFeeDirection, fromCurrency, toCurrency, pivotCurrency, rateDate string) (ResolvedRate, error)
	EnabledCurrencyCodes(ctx context.Context) ([]string, error)
	InspectImport(ctx context.Context, ownerOrganizationID uuid.UUID, rows []*ExchangeRateImportRow) (map[int][]string, error)
	CreateImportPreview(ctx context.Context, batch *ExchangeRateImportBatch, audit *AuditEvent) (*ExchangeRateImportBatch, error)
	GetImport(ctx context.Context, organizationID, id uuid.UUID) (*ExchangeRateImportBatch, error)
	ConfirmImport(ctx context.Context, organizationID, ownerOrganizationID, actorID uuid.UUID, previewTokenHash, idempotencyKey string, now time.Time, audit *AuditEvent) (*ExchangeRateImportBatch, error)
}

// ExchangeRateQuote 是单币种抓取建议值（1 外币折本币），ar/ap 分别来自现汇
// 卖出价（Ask）与现汇买入价（Bid），rate 为折算价（中行折算价或中间价口径）。
type ExchangeRateQuote struct {
	ARRate decimal.Decimal
	APRate decimal.Decimal
	Rate   decimal.Decimal
	// Detail 说明来源与换算路径（如「中行交叉盘 USD→CNY ÷ HKD→CNY」）。
	Detail string
}

// ExchangeRateQuoteSet 是一次抓取的整组结果，来源在集合级明示；
// FallbackUsed=true 表示主源失败后启用备选源（非静默兜底）。
type ExchangeRateQuoteSet struct {
	Source       string
	FallbackUsed bool
	Quotes       map[string]ExchangeRateQuote
}

// ExchangeRateQuoteProvider 由 data 层实现外部牌价抓取（HTTP），biz 层按组织本币
// 编排主备源路由。
type ExchangeRateQuoteProvider interface {
	// FetchCNYBankQuotes 抓取外币→CNY 的中国银行牌价（新浪历史中行专线优先 +
	// 新浪实时专线主源 + 中行官方牌价页兜底），报价已归一化为 1 外币折 CNY。
	// 可选 targetDate 指定周一发盘日期（如 "2026-09-14"）。
	FetchCNYBankQuotes(ctx context.Context, currencies []string, targetDate ...string) (*ExchangeRateQuoteSet, error)
	// FetchDirectQuotes 抓取 currencies→baseCurrency 的国际直盘（Ask→ar、Bid→ap）。
	FetchDirectQuotes(ctx context.Context, baseCurrency string, currencies []string) (*ExchangeRateQuoteSet, error)
}

type ExchangeRateUsecase struct {
	repo          ExchangeRateRepo
	quoteProvider ExchangeRateQuoteProvider
	now           func() time.Time
}

func NewExchangeRateUsecase(repo ExchangeRateRepo, quoteProvider ExchangeRateQuoteProvider) *ExchangeRateUsecase {
	return &ExchangeRateUsecase{repo: repo, quoteProvider: quoteProvider, now: time.Now}
}

func (uc *ExchangeRateUsecase) List(ctx context.Context, organizationID uuid.UUID, options ExchangeRateListOptions) (*PagedList[*ExchangeRateSetting], string, error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, "", ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, "", err
	}
	options.FromCurrency = strings.ToUpper(strings.TrimSpace(options.FromCurrency))
	items, total, err := uc.repo.List(ctx, organizationID, options)
	if err != nil {
		return nil, "", err
	}
	return &PagedList[*ExchangeRateSetting]{
		Items:    items,
		Total:    int(total),
		Page:     options.Page,
		PageSize: options.PageSize,
	}, rateContext.BaseCurrency, nil
}

func (uc *ExchangeRateUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	if err := uc.ensureOrganizationScope(ctx, organizationID, normalized, access.FinanceExchangeRateCreate); err != nil {
		return nil, err
	}
	normalized.IsActive = true
	normalized.Source = ExchangeRateSettingSourceManual
	saved, err := uc.repo.UpsertWeeklyBatch(ctx, normalized.Source, []*ExchangeRateSetting{normalized}, exchangeRateAudit(organizationID, actorID, normalized.ID, "finance.exchange_rate.create"))
	if err != nil {
		return nil, err
	}
	return saved[0], nil
}

func (uc *ExchangeRateUsecase) Update(ctx context.Context, organizationID, actorID uuid.UUID, id uuid.UUID, input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	normalized, err := normalizeExchangeRateSetting(input)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	normalized.ID = id
	if err := uc.ensureOrganizationScope(ctx, organizationID, normalized, access.FinanceExchangeRateUpdate); err != nil {
		return nil, err
	}
	return uc.repo.UpdateScoped(ctx, organizationID, normalized, IsHeadquartersOrganization(ctx), exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.update"))
}

func (uc *ExchangeRateUsecase) Disable(ctx context.Context, organizationID, actorID uuid.UUID, id uuid.UUID) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || id == uuid.Nil {
		return ErrExchangeRateInvalidArgument
	}
	var permissionKey = access.FinanceExchangeRateDisable
	if IsHeadquartersOrganization(ctx) {
		// 总部可停用 NULL 基线行（权限码 + 总部身份双重校验）。
		if err := RequireBaselineWrite(ctx, permissionKey); err != nil {
			return err
		}
	} else if !requireExchangeRatePermission(ctx, permissionKey) {
		return ErrExchangeRatePermissionDenied
	}
	return uc.repo.DisableScoped(ctx, organizationID, id, IsHeadquartersOrganization(ctx), exchangeRateAudit(organizationID, actorID, id, "finance.exchange_rate.disable"))
}

// ensureOrganizationScope 校验 to_currency 必须等于组织本币，并按组织身份决定行
// 归属：总部写 NULL 基线行（RequireBaselineWrite 双重校验），分公司写本组织行。
func (uc *ExchangeRateUsecase) ensureOrganizationScope(ctx context.Context, organizationID uuid.UUID, normalized *ExchangeRateSetting, permissionKey string) error {
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return err
	}
	if normalized.ToCurrency != rateContext.BaseCurrency {
		return ErrExchangeRateCurrencyInvalid
	}
	if IsHeadquartersOrganization(ctx) {
		if err := RequireBaselineWrite(ctx, permissionKey); err != nil {
			return err
		}
		normalized.OrganizationID = nil
		return nil
	}
	if !requireExchangeRatePermission(ctx, permissionKey) {
		return ErrExchangeRatePermissionDenied
	}
	normalized.OrganizationID = &organizationID
	return nil
}

func requireExchangeRatePermission(ctx context.Context, permissionKey string) bool {
	principal, err := RequirePrincipal(ctx)
	if err != nil {
		return false
	}
	return principal.HasPermission(permissionKey)
}

// ResolveRate 按目标日期与费用收支方向（RECEIVABLE→ar_rate，PAYABLE→ap_rate）
// 解析 currency 折组织本币的周汇率；currency 即本币时恒为 1（来源 SYSTEM）。
// 解析链四级容灾，绝不阻断单据保存；全链未命中返回 ErrExchangeRateMissing，
// 由前端引导现场手工覆盖（MANUAL）。
func (uc *ExchangeRateUsecase) ResolveRate(ctx context.Context, organizationID uuid.UUID, direction OrderFeeDirection, currency, targetDate string) (ResolvedRate, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !currencyPattern.MatchString(currency) || !validExchangeRateDirection(direction) {
		return ResolvedRate{}, ErrExchangeRateInvalidArgument
	}
	if _, valid := parseExchangeRateLookupTime(targetDate); !valid {
		return ResolvedRate{}, ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return ResolvedRate{}, err
	}
	if currency == rateContext.BaseCurrency {
		return ResolvedRate{Rate: decimal.NewFromInt(1), Source: ExchangeRateSourceSystem}, nil
	}
	return uc.repo.ResolveRate(ctx, organizationID, direction, currency, rateContext.BaseCurrency, rateContext.PivotCurrency, targetDate)
}

func validExchangeRateDirection(direction OrderFeeDirection) bool {
	return direction == OrderFeeReceivable || direction == OrderFeePayable
}

func (uc *ExchangeRateUsecase) BaseCurrency(ctx context.Context, organizationID uuid.UUID) (string, error) {
	if organizationID == uuid.Nil {
		return "", ErrExchangeRateInvalidArgument
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return "", err
	}
	return rateContext.BaseCurrency, nil
}

// ExchangeRateSyncPreview 是一键同步的抓取预览：来源与换算路径明示，财务微调并
// 终审「确认发布」后才批量入库。
type ExchangeRateSyncPreview struct {
	Target        ExchangeRateSyncTarget
	BaseCurrency  string
	EffectiveFrom string
	EffectiveTo   string
	Source        string
	FallbackUsed  bool
	Rows          []*ExchangeRateSyncPreviewRow
}

type ExchangeRateSyncPreviewRow struct {
	FromCurrency   string
	ARRate         decimal.Decimal
	APRate         decimal.Decimal
	Rate           decimal.Decimal
	ConversionPath string
}

// ExchangeRateSyncRow 是财务终审后的单行入库数据。
type ExchangeRateSyncRow struct {
	FromCurrency string
	ARRate       decimal.Decimal
	APRate       decimal.Decimal
	Rate         decimal.Decimal
}

// FetchExchangeRates 按当前组织本币路由数据源抓取目标周牌价，返回结构化预览，
// 不落库。CNY 本币走中行牌价（新浪专线主源 + 官方页兜底）；非 CNY 本币走国际
// 直盘首选 + 中行交叉盘备选（备选在预览中明示，绝不静默兜底）；全部失败返回
// 业务错误，由前端显式引导手工录入。
func (uc *ExchangeRateUsecase) FetchExchangeRates(ctx context.Context, organizationID uuid.UUID, target ExchangeRateSyncTarget) (*ExchangeRateSyncPreview, error) {
	if uc == nil || uc.repo == nil || uc.quoteProvider == nil || organizationID == uuid.Nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	from, to, valid := ExchangeRateSyncWeekWindow(target, uc.now())
	if !valid {
		return nil, ErrExchangeRateSyncTargetInvalid
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	currencies, err := uc.repo.EnabledCurrencyCodes(ctx)
	if err != nil {
		return nil, err
	}
	targets := make([]string, 0, len(currencies))
	for _, code := range currencies {
		if code != rateContext.BaseCurrency {
			targets = append(targets, code)
		}
	}
	if len(targets) == 0 {
		return nil, ErrExchangeRateQuoteUnavailable
	}
	var targetDate string
	mondayDate := from.Format("2006-01-02")
	if !from.After(uc.now()) {
		targetDate = mondayDate
	}
	var quotes *ExchangeRateQuoteSet
	if rateContext.BaseCurrency == cnyCurrency {
		quotes, err = uc.quoteProvider.FetchCNYBankQuotes(ctx, targets, targetDate)
	} else {
		quotes, err = uc.fetchNonCNYQuotes(ctx, rateContext.BaseCurrency, targets, targetDate)
	}
	if err != nil {
		return nil, err
	}
	rows := make([]*ExchangeRateSyncPreviewRow, 0, len(quotes.Quotes))
	for _, code := range targets {
		quote, ok := quotes.Quotes[code]
		if !ok {
			continue
		}
		rows = append(rows, &ExchangeRateSyncPreviewRow{FromCurrency: code, ARRate: quote.ARRate, APRate: quote.APRate, Rate: quote.Rate, ConversionPath: quote.Detail})
	}
	if len(rows) == 0 {
		return nil, ErrExchangeRateQuoteUnavailable
	}
	return &ExchangeRateSyncPreview{
		Target: target, BaseCurrency: rateContext.BaseCurrency,
		EffectiveFrom: from.Format(time.RFC3339), EffectiveTo: to.Format(time.RFC3339),
		Source: quotes.Source, FallbackUsed: quotes.FallbackUsed, Rows: rows,
	}, nil
}

// fetchNonCNYQuotes 非本币组织的两级抓取：国际直盘首选；失败时基于中行牌价
// 交叉换算备选（备用来源与换算路径在预览中明示）。
func (uc *ExchangeRateUsecase) fetchNonCNYQuotes(ctx context.Context, baseCurrency string, currencies []string, targetDate ...string) (*ExchangeRateQuoteSet, error) {
	direct, directErr := uc.quoteProvider.FetchDirectQuotes(ctx, baseCurrency, currencies)
	if directErr == nil {
		return direct, nil
	}
	cnySet, cnyErr := uc.quoteProvider.FetchCNYBankQuotes(ctx, append(append([]string(nil), currencies...), baseCurrency), targetDate...)
	if cnyErr != nil {
		return nil, ErrExchangeRateQuoteUnavailable
	}
	cross := DeriveCrossQuotes(baseCurrency, currencies, cnySet.Quotes)
	if len(cross) == 0 {
		return nil, ErrExchangeRateQuoteUnavailable
	}
	return &ExchangeRateQuoteSet{Source: "中国银行交叉盘换算", FallbackUsed: true, Quotes: cross}, nil
}

// DeriveCrossQuotes 按银行交叉盘算法换算 外币→本币 牌价（bid/ask 交叉口径）：
// ar_rate = Rate(外币→CNY 现汇卖出价) ÷ Rate(本币→CNY 现汇买入价)，
// ap_rate = Rate(外币→CNY 现汇买入价) ÷ Rate(本币→CNY 现汇卖出价)；
// 交叉商法保证 ar > ap（买卖点差方向不丢失），rate 基准价按折算价腿直除。
// 本币或外币缺少 CNY 腿时跳过该币种。
func DeriveCrossQuotes(baseCurrency string, currencies []string, cnyQuotes map[string]ExchangeRateQuote) map[string]ExchangeRateQuote {
	baseLeg, ok := cnyQuotes[baseCurrency]
	if !ok || !baseLeg.APRate.IsPositive() || !baseLeg.ARRate.IsPositive() {
		return nil
	}
	result := make(map[string]ExchangeRateQuote, len(currencies))
	for _, code := range currencies {
		if code == baseCurrency || code == cnyCurrency {
			continue
		}
		leg, ok := cnyQuotes[code]
		if !ok || !leg.APRate.IsPositive() || !leg.ARRate.IsPositive() {
			continue
		}
		result[code] = ExchangeRateQuote{
			ARRate: leg.ARRate.Div(baseLeg.APRate).RoundBank(8),
			APRate: leg.APRate.Div(baseLeg.ARRate).RoundBank(8),
			Rate:   leg.Rate.Div(baseLeg.Rate).RoundBank(8),
			Detail: fmt.Sprintf("中行交叉盘 %s→CNY ÷ %s→CNY", code, baseCurrency),
		}
	}
	return result
}

// SyncExchangeRates 将财务终审（含微调）后的牌价按目标自然周幂等入库：
// 同作用域同货币对同周命中即覆盖更新，绝不报唯一键冲突；历史费用快照不受影响。
func (uc *ExchangeRateUsecase) SyncExchangeRates(ctx context.Context, organizationID, actorID uuid.UUID, target ExchangeRateSyncTarget, rows []*ExchangeRateSyncRow) (int, string, string, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil {
		return 0, "", "", ErrExchangeRateInvalidArgument
	}
	from, to, valid := ExchangeRateSyncWeekWindow(target, uc.now())
	if !valid {
		return 0, "", "", ErrExchangeRateSyncTargetInvalid
	}
	if len(rows) == 0 {
		return 0, "", "", ErrExchangeRateSyncRowsEmpty
	}
	rateContext, err := uc.repo.ResolveContext(ctx, organizationID)
	if err != nil {
		return 0, "", "", err
	}
	// 行归属与组织身份一致（与页面维护同款判定）：总部写 NULL 基线行（权限码 +
	// 总部身份双重校验），分公司写本组织 org 行——分公司一键同步严禁覆盖全网基线。
	var organizationScope *uuid.UUID
	if IsHeadquartersOrganization(ctx) {
		if err := RequireBaselineWrite(ctx, access.FinanceExchangeRateCreate); err != nil {
			return 0, "", "", err
		}
	} else if !requireExchangeRatePermission(ctx, access.FinanceExchangeRateCreate) {
		return 0, "", "", ErrExchangeRatePermissionDenied
	} else {
		organizationScope = &organizationID
	}
	enabled, err := uc.repo.EnabledCurrencyCodes(ctx)
	if err != nil {
		return 0, "", "", err
	}
	enabledSet := make(map[string]struct{}, len(enabled))
	for _, code := range enabled {
		enabledSet[code] = struct{}{}
	}
	inputs := make([]*ExchangeRateSetting, 0, len(rows))
	for _, row := range rows {
		if row == nil || !currencyPattern.MatchString(strings.ToUpper(strings.TrimSpace(row.FromCurrency))) {
			return 0, "", "", ErrExchangeRateSyncRowsInvalid
		}
		code := strings.ToUpper(strings.TrimSpace(row.FromCurrency))
		if code == rateContext.BaseCurrency {
			return 0, "", "", ErrExchangeRateCurrencyInvalid
		}
		if _, enabledOK := enabledSet[code]; !enabledOK {
			return 0, "", "", ErrExchangeRateCurrencyInvalid
		}
		if !validExchangeRate(row.ARRate) || !validExchangeRate(row.APRate) {
			return 0, "", "", ErrExchangeRateSyncRowsInvalid
		}
		// 基准价缺省按应收/应付中间价记录（与手工维护口径一致）。
		rate := row.Rate
		if !validExchangeRate(rate) {
			rate = row.ARRate.Add(row.APRate).Div(decimal.NewFromInt(2)).RoundBank(8)
		}
		inputs = append(inputs, &ExchangeRateSetting{
			ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationScope,
			FromCurrency: code, ToCurrency: rateContext.BaseCurrency,
			EffectiveFrom: from.Format(time.RFC3339), EffectiveTo: strPtr(to.Format(time.RFC3339)),
			ARRate: decimalPtr(row.ARRate), APRate: decimalPtr(row.APRate), Rate: rate,
		})
	}
	saved, err := uc.repo.UpsertWeeklyBatch(ctx, ExchangeRateSettingSourceBOCSync, inputs, exchangeRateSyncAudit(organizationID, actorID, target, from, to, len(inputs)))
	if err != nil {
		return 0, "", "", err
	}
	return len(saved), from.Format(time.RFC3339), to.Format(time.RFC3339), nil
}

func normalizeExchangeRateSetting(input *ExchangeRateSetting) (*ExchangeRateSetting, error) {
	if input == nil {
		return nil, ErrExchangeRateInvalidArgument
	}
	fromCurrency := strings.ToUpper(strings.TrimSpace(input.FromCurrency))
	toCurrency := strings.ToUpper(strings.TrimSpace(input.ToCurrency))
	if !currencyPattern.MatchString(fromCurrency) || !currencyPattern.MatchString(toCurrency) || fromCurrency == toCurrency {
		return nil, ErrExchangeRateInvalidArgument
	}
	// 周汇率管理：生效时刻归一化为所在自然周窗口（周一 00:00:00 至周日 23:59:59）。
	_, _, fromTime, valid := parseExchangeRateWeekInput(input.EffectiveFrom)
	if !valid {
		return nil, ErrExchangeRateInvalidArgument
	}
	weekFrom, weekTo := ExchangeRateWeekWindow(fromTime)
	if input.ARRate == nil || input.APRate == nil {
		// 同步接入后 ar/ap 收紧为必填（双轨买卖点差）。
		return nil, ErrExchangeRateInvalidArgument
	}
	if !validExchangeRate(*input.ARRate) || !validExchangeRate(*input.APRate) {
		return nil, ErrExchangeRateInvalidArgument
	}
	rate := input.Rate
	// 基准价缺省按应收/应付中间价记录（中行折算价口径的工程近似，用于审计基线）。
	if !validExchangeRate(rate) {
		rate = input.ARRate.Add(*input.APRate).Div(decimal.NewFromInt(2)).RoundBank(8)
	}
	output := *input
	output.FromCurrency = fromCurrency
	output.ToCurrency = toCurrency
	output.EffectiveFrom = weekFrom.Format(time.RFC3339)
	output.EffectiveTo = strPtr(weekTo.Format(time.RFC3339))
	output.Rate = rate
	return &output, nil
}

// parseExchangeRateWeekInput 接受 YYYY-MM-DD 或带时区秒级 RFC 3339 的周内时刻。
func parseExchangeRateWeekInput(value string) (string, time.Time, time.Time, bool) {
	parsed, valid := parseExchangeRateLookupTime(value)
	if !valid {
		return "", time.Time{}, time.Time{}, false
	}
	from, to := ExchangeRateWeekWindow(parsed)
	return from.Format(time.RFC3339), from, to, true
}

func validExchangeRate(value decimal.Decimal) bool {
	return value.IsPositive() && exchangeRateValuePattern.MatchString(value.String())
}

// normalizeExchangeRateTimestamp 将外部时间统一为上海时区的秒级 RFC 3339。
// 汇率有效期不接受无时区时间或小数秒，避免不同客户端产生不一致的区间边界。
func normalizeExchangeRateTimestamp(value string) (string, time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil || parsed.Nanosecond() != 0 {
		return "", time.Time{}, false
	}
	parsed = parsed.In(exchangeRateBusinessLocation)
	return parsed.Format(time.RFC3339), parsed, true
}

// parseExchangeRateLookupTime 接受 YYYY-MM-DD（按业务时区解释）或带时区秒级 RFC 3339。
func parseExchangeRateLookupTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	lookup, err := time.ParseInLocation("2006-01-02", value, exchangeRateBusinessLocation)
	if err == nil && lookup.Format("2006-01-02") == value {
		return lookup, true
	}
	_, parsed, valid := normalizeExchangeRateTimestamp(value)
	return parsed, valid
}

func exchangeRateAudit(organizationID, actorID, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "exchange_rate_setting", ResourceID: id.String()}
}

func exchangeRateSyncAudit(organizationID, actorID uuid.UUID, target ExchangeRateSyncTarget, from, to time.Time, count int) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "finance.exchange_rate.sync", Result: "success", ResourceType: "exchange_rate_setting", Details: map[string]string{
		"sync.target":         string(target),
		"sync.effective_from": from.Format(time.RFC3339),
		"sync.effective_to":   to.Format(time.RFC3339),
		"sync.row_count":      fmt.Sprintf("%d", count),
	}}
}

func decimalPtr(value decimal.Decimal) *decimal.Decimal { return &value }

func strPtr(value string) *string { return &value }
