package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

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
	var direction OrderFeeDirection
	groupsByRawKey := make(map[string]*FinanceBillBatchPreviewGroup)
	rawKeys := make([]string, 0)
	for _, item := range ordered {
		if item == nil || item.Fee == nil || item.Fee.ID == uuid.Nil || item.Fee.OrderID == uuid.Nil || item.Fee.Status != OrderFeeConfirmed || item.Fee.TaxRate == nil || !financeBillCurrencyPattern.MatchString(item.Fee.Currency) || !financeBillCurrencyPattern.MatchString(item.Fee.BaseCurrency) {
			return nil, ErrFinanceBillFeeInvalid
		}
		fee := item.Fee
		if _, duplicate := seen[fee.ID]; duplicate {
			return nil, ErrFinanceBillInvalidArgument
		}
		seen[fee.ID] = struct{}{}
		// 普通模式一次建账只允许单一收付方向；对冲模式允许混合方向，但方向仍参与叶子身份，
		// 应收、应付费用分别形成各自的原始账单叶子。
		if policy.Mode == "NORMAL" {
			if direction == "" {
				direction = fee.Direction
			} else if direction != fee.Direction {
				return nil, ErrFinanceBillMixedDirection
			}
		}
		// 普通账单始终以费用币种拆分；币种是叶子身份的一部分，不能由请求覆盖。
		parts := []string{string(fee.Direction), fee.SettlementPartyID.String(), fee.Currency, fee.BaseCurrency}
		if policy.SplitByTaxRate {
			parts = append(parts, fee.TaxRate.StringFixed(4))
		}
		if policy.SplitByOrder {
			parts = append(parts, fee.OrderID.String())
		}
		raw := strings.Join(parts, "\x00")
		group := groupsByRawKey[raw]
		if group == nil {
			group = &FinanceBillBatchPreviewGroup{GroupKey: financeSHA256(raw), Direction: fee.Direction, SettlementPartyID: fee.SettlementPartyID, SettlementPartyName: fee.SettlementPartyName, Currency: fee.Currency, BaseCurrency: fee.BaseCurrency, Fees: make([]*FinanceBillableFee, 0)}
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

	complete := true
	for _, group := range preview.Groups {
		if strings.TrimSpace(group.BillDate) == "" {
			group.TemporaryBillDate = true
			for _, item := range group.Fees {
				group.TotalAmount = group.TotalAmount.Add(item.Fee.TotalAmount)
				group.NetAmount = group.NetAmount.Add(item.Fee.NetAmount)
			}
			group.TotalAmount = group.TotalAmount.RoundBank(8)
			group.NetAmount = group.NetAmount.RoundBank(8)
			group.TaxAmount = group.TotalAmount.Sub(group.NetAmount)
			estimatedCurrency, estimatedRate, estimatedAmount, estimatedErr := financeBillEstimatedInvoiceSnapshot(group.Currency, group.TotalAmount, group.config)
			if estimatedErr != nil {
				return nil, estimatedErr
			}
			group.EstimatedInvoiceCurrency = estimatedCurrency
			group.EstimatedInvoiceRate = estimatedRate
			group.EstimatedInvoiceAmount = estimatedAmount
			group.ConfigurationComplete = false
			complete = false
			continue
		}
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
			complete = false
			continue
		}
		group.ConfigurationComplete = true
	}
	if input.GroupingPolicy.Mode == "NETTING" {
		preview.NettingPairs = buildFinanceNettingPairs(preview.Groups)
	}
	if !complete {
		preview.PreviewToken = ""
		return preview, nil
	}
	preview.PreviewToken = financeBillConfiguredPreviewToken(organizationID, input.GroupingPolicy, preview.Groups)
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
		ID: billID, OrganizationID: organizationID, Direction: group.Direction, Status: FinanceBillDraft,
		SettlementPartyID: group.SettlementPartyID, SettlementPartyName: group.SettlementPartyName,
		SettlementAccountID: group.SettlementAccountID, Currency: group.Currency, BaseCurrency: group.BaseCurrency,
		ExchangeRate: resolved.Rate.RoundBank(8), ExchangeRateSource: resolved.Source, ExchangeRateDate: billDate,
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

func financeBillConfiguredPreviewToken(organizationID uuid.UUID, policy FinanceBillGroupingPolicy, groups []*FinanceBillBatchPreviewGroup) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, organizationID.String(), policy.Mode, strconv.FormatBool(policy.SplitByTaxRate), strconv.FormatBool(policy.SplitByOrder))
	for _, group := range groups {
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
			taxRate := ""
			if line.TaxRate != nil {
				taxRate = line.TaxRate.StringFixed(4)
			}
			fee := financeBillFeeByID(group.Fees, line.OrderFeeID)
			feeStatus := ""
			if fee != nil {
				feeStatus = string(fee.Status)
			}
			writeFinanceHashParts(&builder, line.OrderFeeID.String(), line.OrderID.String(), line.OrderNo, line.BusinessType, line.FeeCode, line.FeeName, taxRate, line.Currency, line.TotalAmount.StringFixed(8), line.NetAmount.StringFixed(8), line.TaxAmount.StringFixed(8), line.ExchangeRate.StringFixed(8), line.BaseCurrencyAmount.StringFixed(8), strconv.FormatUint(financeBillFeeVersion(group.Fees, line.OrderFeeID), 10), feeStatus)
		}
	}
	return financeSHA256(builder.String())
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

func (uc *FinanceBillUsecase) CreateBatch(ctx context.Context, organizationID, actorID uuid.UUID, input CreateFinanceBillBatchInput) (*FinanceBillBatch, error) {
	feeIDs, err := normalizeFinanceBillFeeIDs(input.FeeIDs)
	if err != nil || organizationID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	input.FeeIDs = feeIDs
	input.PreviewToken = strings.TrimSpace(input.PreviewToken)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.PreviewToken == "" || len(input.PreviewToken) != 64 || input.IdempotencyKey == "" || utf8.RuneCountInString(input.IdempotencyKey) > 128 || len(input.Groups) == 0 || len(input.Groups) > len(input.FeeIDs) {
		return nil, ErrFinanceBillInvalidArgument
	}
	normalizedGroups, err := normalizeFinanceBillBatchGroups(input.Groups)
	if err != nil {
		return nil, err
	}
	input.Groups = normalizedGroups
	requestHash := financeBillBatchRequestHash(input)
	if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr != nil {
		return nil, lookupErr
	} else if existing != nil {
		if existing.RequestHash == requestHash {
			return existing, nil
		}
		return nil, ErrFinanceBillBatchConflict
	}
	if uc.transactor == nil {
		return nil, ErrFinanceBillInvalidArgument
	}
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		fees, transactionErr := uc.repo.LoadBillableFees(txCtx, organizationID, input.FeeIDs)
		if transactionErr != nil {
			return transactionErr
		}
		if len(fees) != len(input.FeeIDs) {
			return ErrFinanceBillFeeInvalid
		}
		groupConfigs := make([]FinanceBillBatchPreviewGroupConfig, 0, len(input.Groups))
		for _, group := range input.Groups {
			groupConfigs = append(groupConfigs, FinanceBillBatchPreviewGroupConfig{
				GroupKey: group.GroupKey, BillDate: group.BillDate,
				SettlementAccountID:      group.SettlementAccountID,
				EstimatedInvoiceCurrency: group.EstimatedInvoiceCurrency, EstimatedInvoiceRate: group.EstimatedInvoiceRate,
			})
		}
		previewInput := PreviewFinanceBillBatchInput{FeeIDs: input.FeeIDs, GroupingPolicy: input.GroupingPolicy, GroupConfigs: groupConfigs}
		preview, transactionErr := uc.buildConfiguredFinanceBillBatchPreview(txCtx, organizationID, fees, previewInput)
		if transactionErr != nil {
			return transactionErr
		}
		if preview.PreviewToken == "" || preview.PreviewToken != input.PreviewToken {
			return ErrFinanceBillPreviewStale
		}
		groupInputs := make(map[string]CreateFinanceBillBatchGroupInput, len(input.Groups))
		for _, group := range input.Groups {
			groupInputs[group.GroupKey] = group
		}
		batchID := uuid.Must(uuid.NewV7())
		batch := &FinanceBillBatch{ID: batchID, OrganizationID: organizationID, CreatedBy: actorID, IdempotencyKey: input.IdempotencyKey, RequestHash: requestHash, GroupingPolicy: input.GroupingPolicy, FeeCount: len(input.FeeIDs), BillCount: len(preview.Groups), Bills: make([]*FinanceBill, 0, len(preview.Groups))}
		for _, previewGroup := range preview.Groups {
			groupInput, ok := groupInputs[previewGroup.GroupKey]
			if !ok || previewGroup.preparedBill == nil {
				return ErrFinanceBillBatchMismatch
			}
			delete(groupInputs, previewGroup.GroupKey)
			bill := previewGroup.preparedBill
			bill.ID = uuid.Must(uuid.NewV7())
			bill.BatchID = &batchID
			bill.IdempotencyKey = financeBillBatchBillKey(input.IdempotencyKey, previewGroup.GroupKey)
			title := groupInput.StatementTitle
			bill.StatementTitle = &title
			// 服务端兜底注入（design 决策 A）：请求未显式提供账期且未提供到期日时，
			// 按预览同口径注入默认账期并联动到期日；显式账期（含 0）与显式到期日均不覆盖。
			paymentTermsDays := groupInput.PaymentTermsDays
			if paymentTermsDays == nil && groupInput.DueDate == nil {
				paymentTermsDays = previewGroup.DefaultPaymentTermsDays
			}
			bill.PaymentTermsDays = paymentTermsDays
			bill.DueDate = normalizedFinanceBillDueDate(groupInput.BillDate, groupInput.DueDate, paymentTermsDays)
			bill.Note = normalizedOptionalFinanceString(groupInput.Note)
			for _, line := range bill.Lines {
				line.ID = uuid.Must(uuid.NewV7())
				line.BillID = bill.ID
			}
			if !validFinanceBillTerms(bill.BillDate, bill.DueDate, bill.PaymentTermsDays) {
				return ErrFinanceBillInvalidArgument
			}
			batch.Bills = append(batch.Bills, bill)
			batch.TotalBaseAmount = batch.TotalBaseAmount.Add(bill.BaseCurrencyAmount)
			if batch.BaseCurrency == "" {
				batch.BaseCurrency = bill.BaseCurrency
			} else if batch.BaseCurrency != bill.BaseCurrency {
				return ErrFinanceBillBatchMismatch
			}
		}
		if len(groupInputs) != 0 {
			return ErrFinanceBillBatchMismatch
		}
		batch.TotalBaseAmount = batch.TotalBaseAmount.RoundBank(8)
		nettingAudits := make([]*AuditEvent, 0)
		if input.GroupingPolicy.Mode == "NETTING" {
			// 对冲批次在同一事务内原子生成双方原始账单与对冲结算单；对冲单初始为草稿，
			// 待账单确认后再单独确认生效。
			nettings, planErr := planBatchFinanceNettings(organizationID, batchID, input.IdempotencyKey, batch.Bills)
			if planErr != nil {
				return planErr
			}
			batch.Nettings = nettings
			for _, netting := range nettings {
				nettingAudits = append(nettingAudits, financeNettingAudit(organizationID, actorID, netting.ID, "finance.netting.create"))
			}
		}
		_, transactionErr = uc.repo.CreateBatch(txCtx, batch, input.PreviewToken, financeBillBatchAudit(organizationID, actorID, batchID, "finance.bill_batch.create"), nettingAudits)
		return transactionErr
	})
	if err == nil {
		return uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	}
	if existing, lookupErr := uc.repo.GetBatchByIdempotencyKey(ctx, organizationID, input.IdempotencyKey); lookupErr == nil && existing != nil && existing.RequestHash == requestHash {
		return existing, nil
	}
	return nil, err
}

func (uc *FinanceBillUsecase) ConfirmBatch(ctx context.Context, organizationIDs []uuid.UUID, actorID, batchID uuid.UUID, expectedVersions map[uuid.UUID]uint64) (*FinanceBillBatch, error) {
	if !validFinanceBillOrganizationIDs(organizationIDs) || actorID == uuid.Nil || batchID == uuid.Nil || len(expectedVersions) == 0 || len(expectedVersions) > 500 {
		return nil, ErrFinanceBillInvalidArgument
	}
	for billID, version := range expectedVersions {
		if billID == uuid.Nil || version == 0 {
			return nil, ErrFinanceBillInvalidArgument
		}
	}
	batch, err := uc.repo.GetBatch(ctx, organizationIDs, batchID)
	if err != nil {
		return nil, err
	}
	return uc.repo.ConfirmBatch(ctx, organizationIDs, batchID, actorID, expectedVersions, financeBillBatchAudit(batch.OrganizationID, actorID, batchID, "finance.bill_batch.confirm"))
}

func normalizeFinanceBillBatchGroups(groups []CreateFinanceBillBatchGroupInput) ([]CreateFinanceBillBatchGroupInput, error) {
	result := make([]CreateFinanceBillBatchGroupInput, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, item := range groups {
		item.GroupKey = strings.TrimSpace(item.GroupKey)
		item.StatementTitle = strings.TrimSpace(item.StatementTitle)
		item.BillDate = strings.TrimSpace(item.BillDate)
		if item.EstimatedInvoiceCurrency != nil {
			value := strings.ToUpper(strings.TrimSpace(*item.EstimatedInvoiceCurrency))
			item.EstimatedInvoiceCurrency = &value
		}
		item.DueDate = normalizedOptionalFinanceString(item.DueDate)
		item.Note = normalizedOptionalFinanceString(item.Note)
		item.DueDate = normalizedFinanceBillDueDate(item.BillDate, item.DueDate, item.PaymentTermsDays)
		if item.GroupKey == "" || len(item.GroupKey) != 64 || item.SettlementAccountID == uuid.Nil || item.StatementTitle == "" || utf8.RuneCountInString(item.StatementTitle) > 200 || !validFinanceDate(item.BillDate) || !validFinanceBillTerms(item.BillDate, item.DueDate, item.PaymentTermsDays) || (item.Note != nil && utf8.RuneCountInString(*item.Note) > 500) || (item.EstimatedInvoiceCurrency != nil && !financeBillCurrencyPattern.MatchString(*item.EstimatedInvoiceCurrency)) || (item.EstimatedInvoiceRate != nil && (!item.EstimatedInvoiceRate.IsPositive() || item.EstimatedInvoiceRate.Exponent() < -8)) || (item.EstimatedInvoiceRate != nil && item.EstimatedInvoiceCurrency == nil) {
			return nil, ErrFinanceBillInvalidArgument
		}
		if _, exists := seen[item.GroupKey]; exists {
			return nil, ErrFinanceBillBatchMismatch
		}
		seen[item.GroupKey] = struct{}{}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].GroupKey < result[j].GroupKey })
	return result, nil
}

func financeBillBatchRequestHash(input CreateFinanceBillBatchInput) string {
	builder := strings.Builder{}
	writeFinanceHashParts(&builder, input.PreviewToken, input.GroupingPolicy.Mode, strconv.FormatBool(input.GroupingPolicy.SplitByOrder), strconv.FormatBool(input.GroupingPolicy.SplitByTaxRate))
	for _, id := range input.FeeIDs {
		writeFinanceHashParts(&builder, id.String())
	}
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
		writeFinanceHashParts(&builder, group.GroupKey, group.StatementTitle, group.BillDate, dueDate, paymentTermsDays, note, group.SettlementAccountID.String(), estimatedCurrency, estimatedRate)
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
