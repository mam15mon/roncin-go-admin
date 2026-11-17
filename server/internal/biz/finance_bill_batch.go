package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (uc *FinanceBillUsecase) PreviewBatch(ctx context.Context, organizationID uuid.UUID, input PreviewFinanceBillBatchInput) (*FinanceBillBatchPreview, error) {
	feeIDs, err := normalizeFinanceBillFeeIDs(input.FeeIDs)
	if err != nil || organizationID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	fees, err := uc.repo.LoadBillableFees(ctx, organizationID, feeIDs)
	if err != nil {
		return nil, err
	}
	if len(fees) != len(feeIDs) {
		return nil, ErrFinanceBillFeeInvalid
	}
	return uc.buildConfiguredFinanceBillBatchPreview(ctx, organizationID, fees, input)
}

// BuildConfiguredFinanceBillBatchPreview 只构造确定性分组；完整预览由用例继续校验账户、币种并解析汇率。
// 该导出函数保留给纯分组测试使用，不能作为创建入口。
func BuildConfiguredFinanceBillBatchPreview(organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput) (*FinanceBillBatchPreview, error) {
	configs := make(map[string]FinanceBillBatchPreviewGroupConfig, len(input.GroupConfigs))
	for index := range input.GroupConfigs {
		config := input.GroupConfigs[index]
		config.GroupKey = strings.TrimSpace(config.GroupKey)
		config.BillDate = strings.TrimSpace(config.BillDate)
		if config.EstimatedInvoiceCurrency != nil {
			value := strings.ToUpper(strings.TrimSpace(*config.EstimatedInvoiceCurrency))
			config.EstimatedInvoiceCurrency = &value
		}
		if config.GroupKey == "" || (config.BillDate != "" && !validFinanceDate(config.BillDate)) || (config.EstimatedInvoiceCurrency != nil && !financeBillCurrencyPattern.MatchString(*config.EstimatedInvoiceCurrency)) || (config.EstimatedInvoiceRate != nil && (!config.EstimatedInvoiceRate.IsPositive() || config.EstimatedInvoiceRate.Exponent() < -8)) || (config.EstimatedInvoiceRate != nil && config.EstimatedInvoiceCurrency == nil) {
			return nil, ErrFinanceBillInvalidArgument
		}
		if _, exists := configs[config.GroupKey]; exists {
			return nil, ErrFinanceBillInvalidArgument
		}
		configs[config.GroupKey] = config
	}
	return buildConfiguredFinanceBillGroups(organizationID, fees, input, configs)
}

func buildConfiguredFinanceBillGroups(organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput, configs map[string]FinanceBillBatchPreviewGroupConfig) (*FinanceBillBatchPreview, error) {
	policy := input.GroupingPolicy
	if organizationID == uuid.Nil || len(fees) == 0 || len(fees) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	if policy.Mode != "NORMAL" && policy.Mode != "NETTING" {
		return nil, ErrFinanceBillGroupingModeUnsupported
	}
	ordered := append([]*FinanceBillableFee(nil), fees...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Fee.ID.String() < ordered[j].Fee.ID.String() })
	seen := make(map[uuid.UUID]struct{}, len(ordered))
	groupsByRawKey := make(map[string]*FinanceBillBatchPreviewGroup)
	rawKeys := make([]string, 0)
	for _, item := range ordered {
		if item == nil || item.Fee == nil || item.Fee.ID == uuid.Nil || item.Fee.OrderID == uuid.Nil || item.Fee.HasActiveBill || item.Fee.TaxRate == nil || !financeBillCurrencyPattern.MatchString(item.Fee.Currency) || !financeBillCurrencyPattern.MatchString(item.Fee.BaseCurrency) {
			return nil, ErrFinanceBillFeeInvalid
		}
		fee := item.Fee
		if _, duplicate := seen[fee.ID]; duplicate {
			return nil, ErrFinanceBillInvalidArgument
		}
		seen[fee.ID] = struct{}{}
		// 分组固定维度为「公司＋收付方向＋币种＋结算对象」（公司即请求公司）：同一请求可同时含应收、应付，
		// 方向不同必然落在不同组，每组只有单一方向；税率、订单是组内可选细分，默认关闭。
		// 普通账单始终以费用币种拆分；币种是叶子身份的一部分，不能由请求覆盖。
		parts := []string{string(SettlementDirection(fee)), string(FinanceBillPurposeForFee(fee)), fee.SettlementPartyID.String(), fee.Currency, fee.BaseCurrency}
		if fee.SourceFeeID != nil && fee.CorrectionRequestID != nil {
			parts = append(parts, fee.CorrectionRequestID.String())
		}
		if policy.SplitByTaxRate {
			parts = append(parts, fee.TaxRate.StringFixed(4))
		}
		if policy.SplitByOrder {
			parts = append(parts, fee.OrderID.String())
		}
		raw := strings.Join(parts, "\x00")
		group := groupsByRawKey[raw]
		if group == nil {
			group = &FinanceBillBatchPreviewGroup{GroupKey: financeSHA256(raw), Purpose: FinanceBillPurposeForFee(fee), Direction: SettlementDirection(fee), SettlementPartyID: fee.SettlementPartyID, SettlementPartyName: fee.SettlementPartyName, Currency: fee.Currency, BaseCurrency: fee.BaseCurrency, Fees: make([]*FinanceBillableFee, 0)}
			if fee.SourceFeeID != nil {
				group.CorrectionRequestID = fee.CorrectionRequestID
			}
			if policy.SplitByTaxRate {
				value := *fee.TaxRate
				group.TaxRate = &value
			}
			if policy.SplitByOrder {
				id, no := fee.OrderID, item.OrderNo
				group.OrderID, group.OrderNo = &id, &no
			}
			groupsByRawKey[raw] = group
			rawKeys = append(rawKeys, raw)
		}
		if item.SettlementPartyIsCasual && !group.IsCasual {
			group.IsCasual = true
			if fee.Direction == OrderFeeReceivable {
				zero := 0
				group.DefaultPaymentTermsDays = &zero
			}
		}
		group.Fees = append(group.Fees, item)
	}
	sort.Strings(rawKeys)
	result := &FinanceBillBatchPreview{Groups: make([]*FinanceBillBatchPreviewGroup, 0, len(rawKeys))}
	for _, raw := range rawKeys {
		group := groupsByRawKey[raw]
		if config, ok := configs[group.GroupKey]; ok {
			group.BillDate, group.SettlementAccountID, group.config = config.BillDate, config.SettlementAccountID, config
		}
		result.Groups = append(result.Groups, group)
	}
	if policy.Mode == "NETTING" {
		if err := validateFinanceNettingGroupPairs(result.Groups); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// validateFinanceNettingGroupPairs 要求对冲建账的每个“结算单位 + 账单币种”组合
// 至少各有一笔应收和应付费用；单方向费用应使用普通账单，不静默拆成两个普通批次。
func validateFinanceNettingGroupPairs(groups []*FinanceBillBatchPreviewGroup) error {
	type pairDirection struct{ receivable, payable bool }
	pairs := make(map[string]*pairDirection)
	for _, group := range groups {
		if group == nil {
			return ErrFinanceNettingInvalid
		}
		key := group.SettlementPartyID.String() + "|" + group.Currency
		pair := pairs[key]
		if pair == nil {
			pair = &pairDirection{}
			pairs[key] = pair
		}
		if group.Direction == OrderFeeReceivable {
			pair.receivable = true
		} else {
			pair.payable = true
		}
	}
	for _, pair := range pairs {
		if !pair.receivable || !pair.payable {
			return ErrFinanceNettingSingleDirection
		}
	}
	return nil
}

func (uc *FinanceBillUsecase) buildConfiguredFinanceBillBatchPreview(ctx context.Context, organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput) (*FinanceBillBatchPreview, error) {
	return uc.buildConfiguredFinanceBillBatchPreviewForDate(ctx, organizationID, fees, input, uc.businessDate())
}

func (uc *FinanceBillUsecase) buildConfiguredFinanceBillBatchPreviewForDate(ctx context.Context, organizationID uuid.UUID, fees []*FinanceBillableFee, input PreviewFinanceBillBatchInput, billDate string) (*FinanceBillBatchPreview, error) {
	preview, err := BuildConfiguredFinanceBillBatchPreview(organizationID, fees, input)
	if err != nil {
		return nil, err
	}
	// 应收叶子注入默认账期与信用额度比对信息：散客默认 0 天已由分组事实硬编码，
	// 正式客户取客户角色激活规则的默认账期；超额只做预警展示，不在预览或入账环节阻断。
	if enrichErr := uc.enrichReceivableGroupCreditInfo(ctx, organizationID, preview.Groups); enrichErr != nil {
		return nil, enrichErr
	}
	if uc.exchangeRate == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	accountSkeletons := make(map[string]*FinanceBill, len(preview.Groups))
	accounts := make([]*FinanceBill, 0, len(preview.Groups))
	for _, group := range preview.Groups {
		if group.Currency == "" || group.SettlementAccountID == uuid.Nil {
			continue
		}
		skeleton := &FinanceBill{OrganizationID: organizationID, Direction: group.Direction, SettlementPartyID: group.SettlementPartyID, SettlementAccountID: group.SettlementAccountID, Currency: group.Currency}
		accountSkeletons[group.GroupKey] = skeleton
		accounts = append(accounts, skeleton)
	}
	if len(accounts) > 0 {
		if err = uc.repo.HydrateBillSettlementAccounts(ctx, accounts); err != nil {
			return nil, err
		}
	}
	currencySet := make(map[string]struct{})
	for _, item := range fees {
		currencySet[item.Fee.Currency] = struct{}{}
		currencySet[item.Fee.BaseCurrency] = struct{}{}
	}
	for _, group := range preview.Groups {
		if group.Currency != "" {
			currencySet[group.Currency] = struct{}{}
		}
		if group.config.EstimatedInvoiceCurrency != nil {
			currencySet[*group.config.EstimatedInvoiceCurrency] = struct{}{}
		}
	}
	currencies := make([]string, 0, len(currencySet))
	for currency := range currencySet {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	if err = uc.repo.ValidateBillCurrencies(ctx, currencies); err != nil {
		return nil, err
	}

	for _, group := range preview.Groups {
		group.BillDate = billDate
		bill, buildErr := uc.buildFixedCurrencyFinanceBill(ctx, organizationID, group, group.BillDate)
		if buildErr != nil {
			return nil, buildErr
		}
		group.preparedBill = bill
		if account := accountSkeletons[group.GroupKey]; account != nil {
			bill.SettlementAccountName = account.SettlementAccountName
			bill.SettlementAccountHolder = account.SettlementAccountHolder
			bill.SettlementBankName = account.SettlementBankName
			bill.SettlementBankAccount = account.SettlementBankAccount
			bill.SettlementAccountCurrency = account.SettlementAccountCurrency
			bill.SettlementSwiftCode = account.SettlementSwiftCode
		}
		group.TotalAmount = bill.TotalAmount
		group.NetAmount = bill.NetAmount
		group.TaxAmount = bill.TaxAmount
		group.BaseCurrencyAmount = bill.BaseCurrencyAmount
		if group.BillDate == "" || group.SettlementAccountID == uuid.Nil {
			group.ConfigurationComplete = false
			continue
		}
		group.ConfigurationComplete = true
		// 令牌只摘要本组事实与配置，不含其他组：单组重算结果与整批预览一致。
		group.GroupToken = financeBillGroupToken(organizationID, input.GroupingPolicy, group)
	}
	if input.GroupingPolicy.Mode == "NETTING" {
		preview.NettingPairs = buildFinanceNettingPairs(preview.Groups)
	}
	return preview, nil
}

// enrichReceivableGroupCreditInfo 为应收叶子补齐默认账期与信用额度比对信息。
// 仅做信息注入：超额叶子标记 IsCreditExceeded 供前端黄色预警，不改变预览令牌与入账行为。
func (uc *FinanceBillUsecase) enrichReceivableGroupCreditInfo(ctx context.Context, organizationID uuid.UUID, groups []*FinanceBillBatchPreviewGroup) error {
	partyIDs := make([]uuid.UUID, 0, len(groups))
	seen := make(map[uuid.UUID]struct{}, len(groups))
	for _, group := range groups {
		if group.Direction != OrderFeeReceivable || group.SettlementPartyID == uuid.Nil {
			continue
		}
		if _, exists := seen[group.SettlementPartyID]; exists {
			continue
		}
		seen[group.SettlementPartyID] = struct{}{}
		partyIDs = append(partyIDs, group.SettlementPartyID)
	}
	if len(partyIDs) == 0 {
		return nil
	}
	summaries, err := uc.repo.GetPartnerCreditSummaries(ctx, organizationID, partyIDs)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.Direction != OrderFeeReceivable {
			continue
		}
		summary := summaries[group.SettlementPartyID]
		if summary == nil {
			continue
		}
		// 散客保持硬编码 0 天底线，不读规则；正式客户按主档规则带出，可调整。
		if !group.IsCasual {
			group.DefaultPaymentTermsDays = summary.DefaultPaymentTermsDays
		}
		unsettled := summary.UnsettledReceivableBase
		group.CurrentUnsettledAmount = &unsettled
		group.CreditLimitAmount = summary.CreditLimitBase
		group.CreditCurrency = summary.CreditCurrency
		group.IsCreditExceeded = PartnerCreditExceeded(summary)
	}
	return nil
}

// defaultFinanceBillPaymentTermsDays 返回单笔建账写入路径的默认账期注入值，
// 与预览 enrich（enrichReceivableGroupCreditInfo）完全同口径：散客应收恒 0，
// 正式客户取客户角色激活规则的默认账期（多条激活规则取最大值，由
// GetPartnerCreditSummaries 聚合层统一）；无规则、应付方向返回 nil 表示不注入。
func (uc *FinanceBillUsecase) defaultFinanceBillPaymentTermsDays(ctx context.Context, organizationID uuid.UUID, fee *FinanceBillableFee) (*int, error) {
	if fee == nil || fee.Fee == nil || fee.Fee.Direction != OrderFeeReceivable {
		return nil, nil
	}
	if fee.SettlementPartyIsCasual {
		zero := 0
		return &zero, nil
	}
	summaries, err := uc.repo.GetPartnerCreditSummaries(ctx, organizationID, []uuid.UUID{fee.Fee.SettlementPartyID})
	if err != nil {
		return nil, err
	}
	summary := summaries[fee.Fee.SettlementPartyID]
	if summary == nil {
		return nil, nil
	}
	return summary.DefaultPaymentTermsDays, nil
}

// buildFinanceNettingPairs 按结算单位与账单币种汇总叶子毛额，抵销额为双方较小值，
// 净应收/净应付为抵销后的剩余金额；金额只用共同账单币种。
func buildFinanceNettingPairs(groups []*FinanceBillBatchPreviewGroup) []*FinanceBillBatchNettingPair {
	type pairTotals struct {
		settlementPartyID   uuid.UUID
		settlementPartyName string
		currency            string
		receivable, payable decimal.Decimal
	}
	pairs := make(map[string]*pairTotals)
	keys := make([]string, 0)
	for _, group := range groups {
		key := group.SettlementPartyID.String() + "|" + group.Currency
		pair := pairs[key]
		if pair == nil {
			pair = &pairTotals{settlementPartyID: group.SettlementPartyID, settlementPartyName: group.SettlementPartyName, currency: group.Currency}
			pairs[key] = pair
			keys = append(keys, key)
		}
		if group.Direction == OrderFeeReceivable {
			pair.receivable = pair.receivable.Add(group.TotalAmount)
		} else {
			pair.payable = pair.payable.Add(group.TotalAmount)
		}
	}
	sort.Strings(keys)
	result := make([]*FinanceBillBatchNettingPair, 0, len(keys))
	for _, key := range keys {
		pair := pairs[key]
		item := &FinanceBillBatchNettingPair{
			SettlementPartyID: pair.settlementPartyID, SettlementPartyName: pair.settlementPartyName, Currency: pair.currency,
			ReceivableGrossAmount: pair.receivable.Round(8), PayableGrossAmount: pair.payable.Round(8),
		}
		if item.ReceivableGrossAmount.IsPositive() && item.PayableGrossAmount.IsPositive() {
			item.OffsetAmount = decimal.Min(item.ReceivableGrossAmount, item.PayableGrossAmount).Round(8)
		}
		item.NetReceivableAmount = item.ReceivableGrossAmount.Sub(item.OffsetAmount).Round(8)
		item.NetPayableAmount = item.PayableGrossAmount.Sub(item.OffsetAmount).Round(8)
		result = append(result, item)
	}
	return result
}

func (uc *FinanceBillUsecase) buildFixedCurrencyFinanceBill(ctx context.Context, organizationID uuid.UUID, group *FinanceBillBatchPreviewGroup, billDate string) (*FinanceBill, error) {
	if group == nil || len(group.Fees) == 0 || !validFinanceDate(billDate) || !financeBillCurrencyPattern.MatchString(group.Currency) {
		return nil, ErrFinanceBillInvalidArgument
	}
	if group.config.EstimatedInvoiceRate != nil && group.config.EstimatedInvoiceCurrency == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	for _, item := range group.Fees {
		if item == nil || item.Fee == nil || item.Fee.Currency != group.Currency || item.Fee.BaseCurrency != group.BaseCurrency {
			return nil, ErrFinanceBillFeeMismatch
		}
	}
	resolved, err := uc.exchangeRate.ResolveRate(ctx, organizationID, group.Direction, group.Currency, billDate)
	if err != nil {
		return nil, err
	}
	billID := uuid.Must(uuid.NewV7())
	bill := &FinanceBill{
		ID: billID, OrganizationID: organizationID, CorrectionRequestID: group.CorrectionRequestID, Purpose: group.Purpose, Direction: group.Direction, Status: FinanceBillConfirmed,
		SettlementPartyID: group.SettlementPartyID, SettlementPartyName: group.SettlementPartyName,
		SettlementAccountID: group.SettlementAccountID, Currency: group.Currency, BaseCurrency: group.BaseCurrency,
		ExchangeRate: resolved.Rate.RoundBank(4), ExchangeRateSource: resolved.Source, ExchangeRateDate: billDate,
		ExchangeRateSettingID: resolved.SettingID,
		BillDate:              billDate, Version: 1,
		Lines: make([]*FinanceBillLine, 0, len(group.Fees)),
	}
	for _, item := range group.Fees {
		fee := item.Fee
		bill.Lines = append(bill.Lines, &FinanceBillLine{
			ID: uuid.Must(uuid.NewV7()), BillID: billID, OrderFeeID: fee.ID, OrderID: fee.OrderID,
			OrderNo: item.OrderNo, BusinessType: item.BusinessType, FeeCode: fee.FeeCode, FeeName: fee.FeeName,
			Quantity: fee.Quantity, UnitPrice: fee.UnitPrice, TotalAmount: fee.TotalAmount, NetAmount: fee.NetAmount, TaxAmount: fee.TaxAmount, TaxRate: fee.TaxRate,
			Currency: group.Currency, ExchangeRate: bill.ExchangeRate, BaseCurrency: group.BaseCurrency, Active: true,
		})
	}
	sort.Slice(bill.Lines, func(i, j int) bool { return bill.Lines[i].OrderFeeID.String() < bill.Lines[j].OrderFeeID.String() })
	for _, line := range bill.Lines {
		bill.TotalAmount = bill.TotalAmount.Add(line.TotalAmount)
		bill.NetAmount = bill.NetAmount.Add(line.NetAmount)
		bill.TaxAmount = bill.TaxAmount.Add(line.TaxAmount)
	}
	bill.TotalAmount = bill.TotalAmount.RoundBank(8)
	bill.NetAmount = bill.NetAmount.RoundBank(8)
	bill.TaxAmount = bill.TotalAmount.Sub(bill.NetAmount)
	bill.BaseCurrencyAmount = bill.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
	allocatedBase := decimal.Zero
	for index, line := range bill.Lines {
		lineBase := line.TotalAmount.Mul(bill.ExchangeRate).RoundBank(8)
		if index == len(bill.Lines)-1 {
			lineBase = bill.BaseCurrencyAmount.Sub(allocatedBase)
		}
		line.BaseCurrencyAmount = lineBase
		allocatedBase = allocatedBase.Add(lineBase)
	}
	bill.FeeCount = len(bill.Lines)
	estimatedCurrency, estimatedRate, estimatedAmount, err := financeBillEstimatedInvoiceSnapshot(group.Currency, bill.TotalAmount, group.config)
	if err != nil {
		return nil, err
	}
	bill.EstimatedInvoiceCurrency = &estimatedCurrency
	bill.EstimatedInvoiceRate = &estimatedRate
	bill.EstimatedInvoiceAmount = &estimatedAmount
	group.EstimatedInvoiceCurrency = estimatedCurrency
	group.EstimatedInvoiceRate = estimatedRate
	group.EstimatedInvoiceAmount = estimatedAmount
	return bill, validateFinanceBillAmountInvariants(bill)
}

func financeBillEstimatedInvoiceSnapshot(billCurrency string, totalAmount decimal.Decimal, config FinanceBillBatchPreviewGroupConfig) (string, decimal.Decimal, decimal.Decimal, error) {
	estimatedCurrency := billCurrency
	estimatedRate := decimal.NewFromInt(1)
	if config.EstimatedInvoiceCurrency != nil {
		estimatedCurrency = strings.ToUpper(strings.TrimSpace(*config.EstimatedInvoiceCurrency))
		if config.EstimatedInvoiceRate != nil {
			estimatedRate = *config.EstimatedInvoiceRate
		} else if estimatedCurrency != billCurrency {
			return "", decimal.Zero, decimal.Zero, ErrFinanceBillInvalidArgument
		}
	}
	// 同币种预计开票不应凭人工汇率改变预计金额；跨币种才需要显式折算率。
	if !financeBillCurrencyPattern.MatchString(estimatedCurrency) || !estimatedRate.IsPositive() || estimatedRate.Exponent() < -8 || (estimatedCurrency == billCurrency && !estimatedRate.Equal(decimal.NewFromInt(1))) {
		return "", decimal.Zero, decimal.Zero, ErrFinanceBillInvalidArgument
	}
	return estimatedCurrency, estimatedRate, totalAmount.Mul(estimatedRate).RoundBank(8), nil
}

// financeBillGroupToken 摘要单组的费用 ID、版本、金额与叶子配置；组织与拆单策略参与摘要，
// 防止跨公司或换策略复用令牌。
func financeBillGroupToken(organizationID uuid.UUID, policy FinanceBillGroupingPolicy, group *FinanceBillBatchPreviewGroup) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, organizationID.String(), policy.Mode, strconv.FormatBool(policy.SplitByTaxRate), strconv.FormatBool(policy.SplitByOrder))
	bill := group.preparedBill
	orderID, orderNo, taxRate, headerSettingID := "", "", "", ""
	if group.OrderID != nil {
		orderID = group.OrderID.String()
	}
	if group.OrderNo != nil {
		orderNo = *group.OrderNo
	}
	if group.TaxRate != nil {
		taxRate = group.TaxRate.StringFixed(4)
	}
	if bill.ExchangeRateSettingID != nil {
		headerSettingID = bill.ExchangeRateSettingID.String()
	}
	writeFinanceHashParts(&builder, group.GroupKey, string(group.Direction), group.SettlementPartyID.String(), group.SettlementPartyName, group.Currency, group.BaseCurrency, orderID, orderNo, taxRate, group.BillDate, group.SettlementAccountID.String(), bill.SettlementAccountName, bill.SettlementAccountHolder, bill.SettlementBankName, bill.SettlementBankAccount, bill.SettlementAccountCurrency, bill.SettlementSwiftCode, bill.ExchangeRate.StringFixed(8), bill.ExchangeRateSource, bill.ExchangeRateDate, headerSettingID, group.EstimatedInvoiceCurrency, group.EstimatedInvoiceRate.StringFixed(8), group.EstimatedInvoiceAmount.StringFixed(8))
	for _, line := range bill.Lines {
		lineTaxRate := ""
		if line.TaxRate != nil {
			lineTaxRate = line.TaxRate.StringFixed(4)
		}
		writeFinanceHashParts(&builder, line.OrderFeeID.String(), line.OrderID.String(), line.OrderNo, line.BusinessType, line.FeeCode, line.FeeName, lineTaxRate, line.Currency, line.TotalAmount.StringFixed(8), line.NetAmount.StringFixed(8), line.TaxAmount.StringFixed(8), line.ExchangeRate.StringFixed(8), line.BaseCurrencyAmount.StringFixed(8), strconv.FormatUint(financeBillFeeVersion(group.Fees, line.OrderFeeID), 10))
	}
	// 令牌仍为不透明的64字符；日期前缀用于跨日先拒绝，后56位保留224位摘要。
	return strings.ReplaceAll(group.BillDate, "-", "") + financeSHA256(builder.String())[:56]
}

func financeBillFeeVersion(items []*FinanceBillableFee, id uuid.UUID) uint64 {
	if fee := financeBillFeeByID(items, id); fee != nil {
		return fee.Version
	}
	return 0
}

func financeBillFeeByID(items []*FinanceBillableFee, id uuid.UUID) *OrderFee {
	for _, item := range items {
		if item != nil && item.Fee != nil && item.Fee.ID == id {
			return item.Fee
		}
	}
	return nil
}

// CreateBatch 按建账模式提交批量建账：
//   - 普通模式（NORMAL）：批次头独立短事务写入，随后按 group_key 升序逐组独立 WithinTransaction；
//     某组失败不回滚其他组，逐组结果写入 Groups。组与组串行处理，不同时持有跨组锁。
//   - 对冲模式（NETTING）：双方账单与草稿对冲单依赖整批原子，批次头、全部账单和对冲单同一事务，
//     任一组失败整批返回错误。
//
// 同幂等键、同请求摘要重发：已成功的组按派生幂等键原样返回，失败的组重新尝试；
// 同幂等键、不同请求摘要返回批次冲突。
func (uc *FinanceBillUsecase) CreateBatch(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput) (*FinanceBillBatchCreateResult, error) {
	return uc.CreateBatchWithOperations(ctx, organizationID, actorID, input, FinanceBillBatchOperations{})
}

// CreateBatchWithOperations 在既有分组事务边界中附带发票登记和核销，依赖由调用方显式注入。
func (uc *FinanceBillUsecase) CreateBatchWithOperations(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput, operations FinanceBillBatchOperations) (*FinanceBillBatchCreateResult, error) {
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if organizationID == uuid.Nil || actorID == uuid.Nil || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || len(input.Groups) == 0 {
		return nil, ErrFinanceBillInvalidArgument
	}
	if input.GroupingPolicy.Mode != "NORMAL" && input.GroupingPolicy.Mode != "NETTING" {
		return nil, ErrFinanceBillGroupingModeUnsupported
	}
	normalizedGroups, err := normalizeFinanceBillBatchGroups(input.Groups)
	if err != nil {
		return nil, err
	}
	input.Groups = normalizedGroups
	if uc.transactor == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	requestHash := financeBillBatchRequestHash(input)
	var result *FinanceBillBatchCreateResult
	if input.GroupingPolicy.Mode == "NETTING" {
		result, err = uc.createNettingFinanceBillBatch(ctx, organizationID, actorID, input, requestHash, operations)
	} else {
		result, err = uc.createNormalFinanceBillBatch(ctx, organizationID, actorID, input, requestHash, operations)
	}
	if err != nil {
		return nil, err
	}
	return uc.enrichFinanceBillBatchOperationResults(ctx, organizationID, input, result)
}

func (uc *FinanceBillUsecase) createNettingFinanceBillBatch(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput, requestHash string, operations FinanceBillBatchOperations) (*FinanceBillBatchCreateResult, error) {
	for _, group := range input.Groups {
		if err := validateFinanceBillBatchOperationPermissions(group, operations); err != nil {
			return nil, err
		}
	}
	existing, err := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return financeBillBatchReplay(existing, input, requestHash)
	}
	feeIDs := financeBillBatchFeeIDs(input.Groups)
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		batchID := uuid.Must(uuid.NewV7())
		bills, prepareErr := uc.prepareFinanceBillBatchBills(txCtx, organizationID, actorID, batchID, input, input.Groups, feeIDs)
		if prepareErr != nil {
			return financeBillBatchStepError(prepareErr, "", "BILL", 0)
		}
		// 对冲批次在同一事务内原子生成双方原始账单与对冲结算单；对冲单初始为草稿，需单独确认生效。
		nettings, planErr := planBatchFinanceNettings(organizationID, batchID, input.IdempotencyKey, bills)
		if planErr != nil {
			return planErr
		}
		nettingAudits := make([]*AuditEvent, 0, len(nettings))
		for _, netting := range nettings {
			nettingAudits = append(nettingAudits, financeNettingAudit(organizationID, actorID, netting.ID, "finance.netting.create"))
		}
		batch := &FinanceBillBatch{ID: batchID, OrganizationID: organizationID, CreatedBy: actorID, IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash, GroupingPolicy: input.GroupingPolicy}
		if err := uc.repo.CreateBatchBills(txCtx, FinanceBillBatchWrite{
			Batch: batch, ActorID: actorID, CreateHead: true, HeadAudit: financeBillBatchAudit(organizationID, actorID, batchID, "finance.bill_batch.create"),
			Bills: bills, BillAudits: financeBillBatchBillAudits(organizationID, actorID, batchID, bills),
			Nettings: nettings, NettingAudits: nettingAudits,
		}); err != nil {
			return financeBillBatchStepError(err, "", "BILL", 0)
		}
		if err := lockFinanceBillBatchCashflows(txCtx, organizationID, input.Groups, operations); err != nil {
			return err
		}
		for index, group := range input.Groups {
			if err := executeFinanceBillBatchOperations(txCtx, organizationID, actorID, input.IdempotencyKey, group, bills[index], operations); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// 同键并发请求可能已由另一请求整批建成；摘要一致时回放，不一致时按批次冲突处理。
		if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr == nil && existing != nil {
			return financeBillBatchReplay(existing, input, requestHash)
		}
		return nil, err
	}
	created, err := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, ErrFinanceBillNotFound
	}
	return financeBillBatchReplay(created, input, requestHash)
}

// financeBillBatchReplay 以已提交批次回放对冲整批结果：每组账单按派生幂等键定位。
func financeBillBatchReplay(batch *FinanceBillBatch, input CreateFinanceBillBatchInput, requestHash string) (*FinanceBillBatchCreateResult, error) {
	if batch.RequestHash != requestHash {
		return nil, ErrFinanceBillBatchConflict
	}
	billsByKey := make(map[string]*FinanceBill, len(batch.Bills))
	for _, bill := range batch.Bills {
		billsByKey[bill.IdempotencyKey] = bill
	}
	result := &FinanceBillBatchCreateResult{Batch: batch, Groups: make([]*FinanceBillBatchGroupResult, 0, len(input.Groups))}
	for _, group := range input.Groups {
		bill := billsByKey[financeBillBatchBillKey(input.IdempotencyKey, group.GroupKey)]
		if bill == nil {
			return nil, ErrFinanceBillBatchMismatch
		}
		result.Groups = append(result.Groups, &FinanceBillBatchGroupResult{GroupKey: group.GroupKey, Bill: bill})
	}
	return result, nil
}

func (uc *FinanceBillUsecase) createNormalFinanceBillBatch(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput, requestHash string, operations FinanceBillBatchOperations) (*FinanceBillBatchCreateResult, error) {
	head, err := uc.ensureFinanceBillBatchHead(ctx, organizationID, actorID, input, requestHash)
	if err != nil {
		return nil, err
	}
	groups := make([]*FinanceBillBatchGroupResult, 0, len(input.Groups))
	for _, group := range input.Groups {
		result := uc.createFinanceBillBatchGroup(ctx, organizationID, actorID, head, input, group, operations)
		// 请求已取消或超时时不再继续后续组；客户端以同键重发即可续做。
		if result.Err != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		groups = append(groups, result)
	}
	batch, err := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if batch == nil {
		return nil, ErrFinanceBillNotFound
	}
	return &FinanceBillBatchCreateResult{Batch: batch, Groups: groups}, nil
}

// ensureFinanceBillBatchHead 返回同幂等键的已有批次头或以独立短事务新建；并发同键创建以唯一索引兜底。
func (uc *FinanceBillUsecase) ensureFinanceBillBatchHead(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput, requestHash string) (*FinanceBillBatch, error) {
	existing, err := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return nil, ErrFinanceBillBatchConflict
		}
		return existing, nil
	}
	batchID := uuid.Must(uuid.NewV7())
	batch := &FinanceBillBatch{ID: batchID, OrganizationID: organizationID, CreatedBy: actorID, IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash, GroupingPolicy: input.GroupingPolicy}
	created, err := uc.repo.CreateBatchHead(ctx, batch, financeBillBatchAudit(organizationID, actorID, batchID, "finance.bill_batch.create"))
	if err != nil {
		if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr == nil && existing != nil {
			if existing.RequestHash != requestHash {
				return nil, ErrFinanceBillBatchConflict
			}
			return existing, nil
		}
		return nil, err
	}
	return created, nil
}

// createFinanceBillBatchGroup 在独立事务内创建一组账单；已按派生幂等键建成的组原样返回。
func (uc *FinanceBillUsecase) createFinanceBillBatchGroup(ctx context.Context, organizationID, actorID uuid.UUID, head *FinanceBillBatch, input CreateFinanceBillBatchInput, group CreateFinanceBillBatchGroupInput, operations FinanceBillBatchOperations) *FinanceBillBatchGroupResult {
	result := &FinanceBillBatchGroupResult{GroupKey: group.GroupKey}
	if err := validateFinanceBillBatchOperationPermissions(group, operations); err != nil {
		result.Err = err
		return result
	}
	billKey := financeBillBatchBillKey(input.IdempotencyKey, group.GroupKey)
	existing, err := uc.repo.GetByIdempotencyKey(ctx, organizationID, billKey)
	if err != nil {
		result.Err = err
		return result
	}
	if existing != nil {
		return financeBillBatchGroupReplay(result, existing, head)
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		bills, prepareErr := uc.prepareFinanceBillBatchBills(txCtx, organizationID, actorID, head.ID, input, []CreateFinanceBillBatchGroupInput{group}, group.FeeIDs)
		if prepareErr != nil {
			return financeBillBatchStepError(prepareErr, group.GroupKey, "BILL", 0)
		}
		if err := uc.repo.CreateBatchBills(txCtx, FinanceBillBatchWrite{Batch: head, ActorID: actorID, Bills: bills, BillAudits: financeBillBatchBillAudits(organizationID, actorID, head.ID, bills)}); err != nil {
			return financeBillBatchStepError(err, group.GroupKey, "BILL", 0)
		}
		if err := lockFinanceBillBatchCashflows(txCtx, organizationID, []CreateFinanceBillBatchGroupInput{group}, operations); err != nil {
			return err
		}
		return executeFinanceBillBatchOperations(txCtx, organizationID, actorID, input.IdempotencyKey, group, bills[0], operations)
	})
	if err != nil {
		// 同键并发请求可能已由另一请求建成该组账单。
		if created, lookupErr := uc.repo.GetByIdempotencyKey(ctx, organizationID, billKey); lookupErr == nil && created != nil {
			return financeBillBatchGroupReplay(result, created, head)
		}
		if uc.logger != nil && errors.FromError(err).Reason == errors.UnknownReason {
			uc.logger.ErrorContext(ctx, "批量建账单组失败", "batch_id", head.ID.String(), "group_key", group.GroupKey, "error", err)
		}
		result.Err = err
		return result
	}
	created, err := uc.repo.GetByIdempotencyKey(ctx, organizationID, billKey)
	if err != nil {
		result.Err = err
		return result
	}
	if created == nil {
		result.Err = ErrFinanceBillNotFound
		return result
	}
	result.Bill = created
	return result
}

func financeBillBatchGroupReplay(result *FinanceBillBatchGroupResult, bill *FinanceBill, head *FinanceBillBatch) *FinanceBillBatchGroupResult {
	if bill.BatchID == nil || *bill.BatchID != head.ID {
		result.Err = ErrFinanceBillBatchConflict
		return result
	}
	result.Bill = bill
	return result
}

// prepareFinanceBillBatchBills 在事务内加锁读取费用、按同一纯分组内核重算预览，逐组比对令牌与费用集合，
// 再组装待写账单。普通模式每次只传一组；对冲模式传入整批全部组。
func (uc *FinanceBillUsecase) prepareFinanceBillBatchBills(txCtx context.Context, organizationID, actorID, batchID uuid.UUID, input CreateFinanceBillBatchInput, groups []CreateFinanceBillBatchGroupInput, feeIDs []uuid.UUID) ([]*FinanceBill, error) {
	fees, err := uc.repo.LoadBillableFees(txCtx, organizationID, feeIDs)
	if err != nil {
		return nil, err
	}
	if len(fees) != len(feeIDs) {
		return nil, ErrFinanceBillFeeInvalid
	}
	// 订单和费用锁取得之后固定一次业务日期，防止等待锁期间跨午夜。
	billDate := uc.businessDate()
	// 成功组在调用本方法前已按原键回放；剩余新建组的旧日令牌先拒绝，不能被今日缺汇率掩盖。
	datePrefix := strings.ReplaceAll(billDate, "-", "")
	for _, group := range groups {
		if len(group.GroupToken) != 64 || group.GroupToken[:8] != datePrefix {
			return nil, ErrFinanceBillPreviewStale
		}
	}
	configs := make([]FinanceBillBatchPreviewGroupConfig, 0, len(groups))
	for _, group := range groups {
		configs = append(configs, FinanceBillBatchPreviewGroupConfig{
			GroupKey: group.GroupKey, BillDate: group.BillDate, SettlementAccountID: group.SettlementAccountID,
			EstimatedInvoiceCurrency: group.EstimatedInvoiceCurrency, EstimatedInvoiceRate: group.EstimatedInvoiceRate,
		})
	}
	preview, err := uc.buildConfiguredFinanceBillBatchPreviewForDate(txCtx, organizationID, fees, PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: input.GroupingPolicy, GroupConfigs: configs}, billDate)
	if err != nil {
		return nil, err
	}
	// 费用事实变化使分组结果与请求不一致（组数、组身份或组内费用不同）时一律视为预览陈旧。
	if len(preview.Groups) != len(groups) {
		return nil, ErrFinanceBillPreviewStale
	}
	previewByKey := make(map[string]*FinanceBillBatchPreviewGroup, len(preview.Groups))
	for _, previewGroup := range preview.Groups {
		previewByKey[previewGroup.GroupKey] = previewGroup
	}
	bills := make([]*FinanceBill, 0, len(groups))
	for _, group := range groups {
		previewGroup := previewByKey[group.GroupKey]
		if previewGroup == nil || previewGroup.preparedBill == nil || previewGroup.GroupToken == "" || previewGroup.GroupToken != group.GroupToken || !financeBillGroupFeesMatch(previewGroup.Fees, group.FeeIDs) {
			return nil, ErrFinanceBillPreviewStale
		}
		bill := previewGroup.preparedBill
		bill.ID = uuid.Must(uuid.NewV7())
		bill.BatchID = &batchID
		bill.IdempotencyKey = financeBillBatchBillKey(input.IdempotencyKey, group.GroupKey)
		// 账单免人工确认：创建人即确认人。
		bill.Status, bill.ConfirmedBy = FinanceBillConfirmed, &actorID
		title := group.StatementTitle
		bill.StatementTitle = &title
		// 服务端兜底注入（design 决策 A）：请求未显式提供账期且未提供到期日时，
		// 按预览同口径注入默认账期并联动到期日；显式账期（含 0）与显式到期日均不覆盖。
		paymentTermsDays := group.PaymentTermsDays
		if paymentTermsDays == nil && group.DueDate == nil {
			paymentTermsDays = previewGroup.DefaultPaymentTermsDays
		}
		bill.PaymentTermsDays = paymentTermsDays
		bill.DueDate = normalizedFinanceBillDueDate(bill.BillDate, group.DueDate, paymentTermsDays)
		bill.Note = normalizedOptionalFinanceString(group.Note)
		for _, line := range bill.Lines {
			line.ID = uuid.Must(uuid.NewV7())
			line.BillID = bill.ID
		}
		if !validFinanceBillTerms(bill.BillDate, bill.DueDate, bill.PaymentTermsDays) {
			return nil, ErrFinanceBillInvalidArgument
		}
		bills = append(bills, bill)
	}
	return bills, nil
}

func financeBillGroupFeesMatch(fees []*FinanceBillableFee, feeIDs []uuid.UUID) bool {
	if len(fees) != len(feeIDs) {
		return false
	}
	expected := make(map[uuid.UUID]struct{}, len(feeIDs))
	for _, id := range feeIDs {
		expected[id] = struct{}{}
	}
	for _, item := range fees {
		if item == nil || item.Fee == nil {
			return false
		}
		if _, ok := expected[item.Fee.ID]; !ok {
			return false
		}
	}
	return true
}

func financeBillBatchFeeIDs(groups []CreateFinanceBillBatchGroupInput) []uuid.UUID {
	result := make([]uuid.UUID, 0)
	for _, group := range groups {
		result = append(result, group.FeeIDs...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func financeBillBatchBillAudits(organizationID, actorID, batchID uuid.UUID, bills []*FinanceBill) []*AuditEvent {
	audits := make([]*AuditEvent, 0, len(bills))
	for _, bill := range bills {
		audit := financeBillAudit(organizationID, actorID, bill.ID, "finance.bill.create")
		audit.Details["finance_bill_batch.id"] = batchID.String()
		audits = append(audits, audit)
	}
	return audits
}

func normalizeFinanceBillBatchGroups(groups []CreateFinanceBillBatchGroupInput) ([]CreateFinanceBillBatchGroupInput, error) {
	result := make([]CreateFinanceBillBatchGroupInput, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	seenFees := make(map[uuid.UUID]struct{})
	for _, item := range groups {
		item.GroupKey = strings.TrimSpace(item.GroupKey)
		item.GroupToken = strings.TrimSpace(item.GroupToken)
		item.StatementTitle = strings.TrimSpace(item.StatementTitle)
		item.BillDate = ""
		if item.EstimatedInvoiceCurrency != nil {
			value := strings.ToUpper(strings.TrimSpace(*item.EstimatedInvoiceCurrency))
			item.EstimatedInvoiceCurrency = &value
		}
		item.DueDate = normalizedOptionalFinanceString(item.DueDate)
		item.Note = normalizedOptionalFinanceString(item.Note)
		if item.GroupKey == "" || len(item.GroupKey) != 64 || len(item.GroupToken) != 64 || item.SettlementAccountID == uuid.Nil || item.StatementTitle == "" || utf8.RuneCountInString(item.StatementTitle) > 200 || (item.PaymentTermsDays != nil && (*item.PaymentTermsDays < 0 || *item.PaymentTermsDays > 3650)) || (item.DueDate != nil && !validFinanceDate(*item.DueDate)) || (item.Note != nil && utf8.RuneCountInString(*item.Note) > 500) || (item.EstimatedInvoiceCurrency != nil && !financeBillCurrencyPattern.MatchString(*item.EstimatedInvoiceCurrency)) || (item.EstimatedInvoiceRate != nil && (!item.EstimatedInvoiceRate.IsPositive() || item.EstimatedInvoiceRate.Exponent() < -8)) || (item.EstimatedInvoiceRate != nil && item.EstimatedInvoiceCurrency == nil) {
			return nil, ErrFinanceBillInvalidArgument
		}
		feeIDs, err := normalizeFinanceBillFeeIDs(item.FeeIDs)
		if err != nil {
			return nil, err
		}
		for _, id := range feeIDs {
			if _, exists := seenFees[id]; exists {
				return nil, ErrFinanceBillInvalidArgument
			}
			seenFees[id] = struct{}{}
		}
		item.FeeIDs = feeIDs
		if err := normalizeFinanceBillBatchOperations(&item); err != nil {
			return nil, err
		}
		if _, exists := seen[item.GroupKey]; exists {
			return nil, ErrFinanceBillBatchMismatch
		}
		seen[item.GroupKey] = struct{}{}
		result = append(result, item)
	}
	// 与单张建账同一费用上限：一次批量最多 500 笔费用。
	if len(seenFees) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GroupKey < result[j].GroupKey })
	return result, nil
}

func financeBillBatchRequestHash(input CreateFinanceBillBatchInput) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, input.GroupingPolicy.Mode, strconv.FormatBool(input.GroupingPolicy.SplitByOrder), strconv.FormatBool(input.GroupingPolicy.SplitByTaxRate))
	for _, group := range input.Groups {
		dueDate := ""
		if group.DueDate != nil {
			dueDate = *group.DueDate
		}
		paymentTermsDays := ""
		if group.PaymentTermsDays != nil {
			paymentTermsDays = strconv.Itoa(*group.PaymentTermsDays)
		}
		note := ""
		if group.Note != nil {
			note = *group.Note
		}
		estimatedCurrency, estimatedRate := "", ""
		if group.EstimatedInvoiceCurrency != nil {
			estimatedCurrency = *group.EstimatedInvoiceCurrency
		}
		if group.EstimatedInvoiceRate != nil {
			estimatedRate = group.EstimatedInvoiceRate.StringFixed(8)
		}
		writeFinanceHashParts(&builder, group.GroupKey, group.GroupToken, group.StatementTitle, dueDate, paymentTermsDays, note, group.SettlementAccountID.String(), estimatedCurrency, estimatedRate, strconv.Itoa(len(group.FeeIDs)))
		for _, id := range group.FeeIDs {
			writeFinanceHashParts(&builder, id.String())
		}
		writeFinanceHashParts(&builder, financeBillBatchOperationsHash(group))
	}
	return financeSHA256(builder.String())
}

func financeBillBatchBillKey(batchKey, groupKey string) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, batchKey, groupKey)
	return "batch-bill:" + financeSHA256(builder.String())
}

// financeBillBatchNettingKey 为对冲批次内每个“结算单位 + 账单币种”组合生成确定性幂等键。
func financeBillBatchNettingKey(batchKey string, settlementPartyID uuid.UUID, currency string) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, batchKey, settlementPartyID.String(), currency)
	return "batch-netting:" + financeSHA256(builder.String())
}

func financeSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func writeFinanceHashParts(builder *strings.Builder, values ...string) {
	for _, value := range values {
		builder.WriteString(strconv.Itoa(len(value)))
		builder.WriteByte(':')
		builder.WriteString(value)
	}
}

func financeBillBatchAudit(org, actor, id uuid.UUID, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_bill_batch", ResourceID: id.String(), Details: map[string]string{"finance_bill_batch.id": id.String()}}
}
