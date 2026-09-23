package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type billCreationCandidateRepoStub struct {
	FinanceBillRepo
	organizationID uuid.UUID
	filter         FinanceBillCreationCandidateFilter
	err            error
}

func (s *billCreationCandidateRepoStub) ListCreationCandidates(_ context.Context, organizationID uuid.UUID, filter FinanceBillCreationCandidateFilter) (*FinanceBillCreationCandidateResult, error) {
	s.organizationID = organizationID
	s.filter = filter
	return &FinanceBillCreationCandidateResult{}, s.err
}

func TestBuildFinanceBillAggregatesExactSnapshots(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(partyID, "100.00000000", "94.33962264", "5.66037736", "100.00000000"),
		financeBillableFeeForTest(partyID, "0.02000000", "0.01886792", "0.00113208", "0.02000000"),
	}
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID}, BillDate: "2026-08-26", IdempotencyKey: "bill-test"}

	bill, err := buildFinanceBill(organizationID, fees, input)
	if err != nil {
		t.Fatalf("构建账单失败: %v", err)
	}
	if bill.TotalAmount.StringFixed(8) != "100.02000000" || bill.NetAmount.StringFixed(8) != "94.35849056" || bill.TaxAmount.StringFixed(8) != "5.66150944" {
		t.Fatalf("账单精确汇总不正确: total=%s net=%s tax=%s", bill.TotalAmount, bill.NetAmount, bill.TaxAmount)
	}
	if bill.FeeCount != 2 || len(bill.Lines) != 2 {
		t.Fatalf("账单费用快照数量不正确: count=%d lines=%d", bill.FeeCount, len(bill.Lines))
	}
}

func TestBuildFinanceBillRejectsMixedSettlementScope(t *testing.T) {
	firstPartyID := uuid.Must(uuid.NewV7())
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(firstPartyID, "100", "100", "0", "100"),
		financeBillableFeeForTest(uuid.Must(uuid.NewV7()), "100", "100", "0", "100"),
	}
	input := CreateFinanceBillInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID}, BillDate: "2026-08-26", IdempotencyKey: "bill-test"}

	if _, err := buildFinanceBill(uuid.Must(uuid.NewV7()), fees, input); err != ErrFinanceBillFeeMismatch {
		t.Fatalf("混合结算单位应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeCreateFinanceBillRejectsDuplicateFeesAndInvalidDueDate(t *testing.T) {
	feeID := uuid.Must(uuid.NewV7())
	if _, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID, feeID}, BillDate: "2026-08-26", IdempotencyKey: "duplicate"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("重复费用应被拒绝，实际错误为 %v", err)
	}
	dueDate := "2026-08-25"
	if _, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", DueDate: &dueDate, IdempotencyKey: "due-date"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("早于账单日期的到期日应被拒绝，实际错误为 %v", err)
	}
}

func TestBuildConfiguredFinanceBillBatchPreviewUsesFixedAndOptionalDimensions(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	first := financeBillableFeeForTest(partyID, "100", "94.33962264", "5.66037736", "100")
	second := financeBillableFeeForTest(partyID, "200", "188.67924528", "11.32075472", "200")
	third := financeBillableFeeForTest(partyID, "300", "283.01886792", "16.98113208", "300")
	second.Fee.OrderID = first.Fee.OrderID
	second.OrderNo = first.OrderNo
	otherRate := decimal.RequireFromString("9")
	third.Fee.TaxRate = &otherRate
	fees := []*FinanceBillableFee{third, first, second}

	basePreview, err := BuildConfiguredFinanceBillBatchPreview(organizationID, fees, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil {
		t.Fatalf("按固定维度预览失败: %v", err)
	}
	if len(basePreview.Groups) != 1 {
		t.Fatalf("固定维度应合并为 1 组，实际为 %d", len(basePreview.Groups))
	}
	if basePreview.Groups[0].OrderID != nil || basePreview.Groups[0].TaxRate != nil {
		t.Fatalf("未启用可选策略时不应返回订单或税率分组维度")
	}

	orderPreview, err := BuildConfiguredFinanceBillBatchPreview(organizationID, fees, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL", SplitByOrder: true}})
	if err != nil {
		t.Fatalf("按订单拆分预览失败: %v", err)
	}
	if len(orderPreview.Groups) != 2 {
		t.Fatalf("按订单拆分应得到 2 组，实际为 %d", len(orderPreview.Groups))
	}

	fullPreview, err := BuildConfiguredFinanceBillBatchPreview(organizationID, fees, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL", SplitByOrder: true, SplitByTaxRate: true}})
	if err != nil {
		t.Fatalf("按订单和税率拆分预览失败: %v", err)
	}
	if len(fullPreview.Groups) != 2 {
		t.Fatalf("按订单和税率拆分应得到 2 组，实际为 %d", len(fullPreview.Groups))
	}
	for _, group := range fullPreview.Groups {
		if group.OrderID == nil || group.OrderNo == nil || group.TaxRate == nil {
			t.Fatalf("启用可选策略后分组必须返回完整维度: %#v", group)
		}
	}
}

func TestBuildConfiguredFinanceBillBatchPreviewIsDeterministic(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	partyID := uuid.Must(uuid.NewV7())
	first := financeBillableFeeForTest(partyID, "100", "94.33962264", "5.66037736", "100")
	second := financeBillableFeeForTest(partyID, "200", "188.67924528", "11.32075472", "200")
	fees := []*FinanceBillableFee{first, second}

	forward, err := BuildConfiguredFinanceBillBatchPreview(organizationID, fees, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil {
		t.Fatalf("首次预览失败: %v", err)
	}
	reverse, err := BuildConfiguredFinanceBillBatchPreview(organizationID, []*FinanceBillableFee{second, first}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil {
		t.Fatalf("倒序预览失败: %v", err)
	}
	if forward.Groups[0].GroupKey != reverse.Groups[0].GroupKey {
		t.Fatalf("相同费用集合不应受输入顺序影响")
	}
}

func TestNormalizeFinanceBillTermsDerivesDueDate(t *testing.T) {
	feeID := uuid.Must(uuid.NewV7())
	terms := 30
	normalized, err := normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", PaymentTermsDays: &terms, IdempotencyKey: "terms", SettlementAccountID: uuid.New()})
	if err != nil {
		t.Fatalf("账期归一化失败: %v", err)
	}
	if normalized.DueDate == nil || *normalized.DueDate != "2026-09-25" {
		t.Fatalf("30 天账期到期日错误: %v", normalized.DueDate)
	}

	inconsistent := "2026-09-24"
	if _, err = normalizeCreateFinanceBill(CreateFinanceBillInput{FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", DueDate: &inconsistent, PaymentTermsDays: &terms, IdempotencyKey: "bad-terms"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("账期与到期日不一致应被拒绝，实际错误为 %v", err)
	}
}

func TestSameFinanceBillCreateIntentRejectsChangedSettlementAccount(t *testing.T) {
	feeID := uuid.New()
	statementTitle := "测试结算单位"
	existing := &FinanceBill{
		SettlementPartyName: "测试结算单位", SettlementAccountID: uuid.New(), BillDate: "2026-08-26",
		StatementTitle: &statementTitle,
		Lines:          []*FinanceBillLine{{OrderFeeID: feeID}},
	}
	requested := CreateFinanceBillInput{
		FeeIDs: []uuid.UUID{feeID}, BillDate: "2026-08-26", SettlementAccountID: uuid.New(),
	}
	if sameFinanceBillCreateIntent(existing, requested) {
		t.Fatal("同一幂等键更换结算账户必须判定为不同创建意图")
	}
	requested.SettlementAccountID = existing.SettlementAccountID
	if !sameFinanceBillCreateIntent(existing, requested) {
		t.Fatal("相同结算账户和费用应保留幂等重试语义")
	}
}

func TestBillCreationCandidatesNormalizeAndRejectInvalid(t *testing.T) {
	repo := &billCreationCandidateRepoStub{}
	uc := NewFinanceBillUsecase(repo, nil, nil, nil, nil)
	org := uuid.New()
	if _, err := uc.ListCreationCandidates(context.Background(), org, FinanceBillCreationCandidateFilter{Page: 1, PageSize: 20, Keyword: "  海运  ", Direction: OrderFeeReceivable}); err != nil {
		t.Fatal(err)
	}
	if repo.organizationID != org || repo.filter.Keyword != "海运" {
		t.Fatalf("候选参数未规范化: %+v", repo.filter)
	}
	if _, err := uc.ListCreationCandidates(context.Background(), uuid.Nil, FinanceBillCreationCandidateFilter{Page: 1, PageSize: 20}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("非法组织错误=%v", err)
	}
	repo.err = errors.New("db")
	if _, err := uc.ListCreationCandidates(context.Background(), org, FinanceBillCreationCandidateFilter{Page: 1, PageSize: 20}); !errors.Is(err, repo.err) {
		t.Fatalf("错误未透传:%v", err)
	}
}

func financeBillableFeeForTest(partyID uuid.UUID, total, net, tax, base string) *FinanceBillableFee {
	feeID := uuid.Must(uuid.NewV7())
	taxRate := decimal.RequireFromString("6")
	return &FinanceBillableFee{
		OrderNo: "SE2026082600001", BusinessType: "SE",
		Fee: &OrderFee{
			ID: feeID, OrderID: uuid.Must(uuid.NewV7()), Direction: OrderFeeReceivable, Status: OrderFeeUnbilled,
			SettlementPartyID: partyID, SettlementPartyName: "验收客户", Currency: "CNY", BaseCurrency: "CNY",
			FeeCode: "OCEAN", FeeName: "海运费", TotalAmount: decimal.RequireFromString(total),
			NetAmount: decimal.RequireFromString(net), TaxAmount: decimal.RequireFromString(tax),
			TaxRate: &taxRate, ExchangeRate: decimal.NewFromInt(1), BaseCurrencyAmount: decimal.RequireFromString(base), Version: 1,
		},
	}
}

func TestCalculateOverdueDays(t *testing.T) {
	dueDate := "2026-09-01"
	futureDueDate := "2026-09-15"
	businessDate := "2026-09-10"

	// 1. 正常应收已确认且未结清、已逾期9天
	days := CalculateOverdueDays(OrderFeeReceivable, FinanceBillConfirmed, decimal.NewFromInt(100), &dueDate, businessDate)
	if days != 9 {
		t.Fatalf("预期逾期9天，实际=%d", days)
	}

	// 2. 应付账单不计算逾期天数
	if days := CalculateOverdueDays(OrderFeePayable, FinanceBillConfirmed, decimal.NewFromInt(100), &dueDate, businessDate); days != 0 {
		t.Fatalf("应付账单不应计算逾期天数，实际=%d", days)
	}

	// 3. 草稿账单不计算逾期天数
	if days := CalculateOverdueDays(OrderFeeReceivable, FinanceBillDraft, decimal.NewFromInt(100), &dueDate, businessDate); days != 0 {
		t.Fatalf("草稿账单不应计算逾期天数，实际=%d", days)
	}

	// 4. 已结清账单（未核销为0）不计算逾期天数
	if days := CalculateOverdueDays(OrderFeeReceivable, FinanceBillConfirmed, decimal.Zero, &dueDate, businessDate); days != 0 {
		t.Fatalf("已结清账单不应计算逾期天数，实际=%d", days)
	}

	// 5. 到期日为空不计算逾期天数
	if days := CalculateOverdueDays(OrderFeeReceivable, FinanceBillConfirmed, decimal.NewFromInt(100), nil, businessDate); days != 0 {
		t.Fatalf("空到期日不应计算逾期天数，实际=%d", days)
	}

	// 6. 到期日在业务日期当天或未来不计算逾期天数
	if days := CalculateOverdueDays(OrderFeeReceivable, FinanceBillConfirmed, decimal.NewFromInt(100), &futureDueDate, businessDate); days != 0 {
		t.Fatalf("未到期账单不应计算逾期天数，实际=%d", days)
	}
	sameDate := businessDate
	if days := CalculateOverdueDays(OrderFeeReceivable, FinanceBillConfirmed, decimal.NewFromInt(100), &sameDate, businessDate); days != 0 {
		t.Fatalf("当天到期账单不应计算逾期天数，实际=%d", days)
	}
}

func TestFinanceBillListDueDateValidation(t *testing.T) {
	uc := NewFinanceBillUsecase(nil, nil, nil, nil, nil)
	org := uuid.New()
	// 非法到期日格式
	if _, err := uc.List(context.Background(), []uuid.UUID{org}, FinanceBillFilter{Page: 1, PageSize: 20, DueDateFrom: "invalid"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("非法到期日应被拒绝，实际错误=%v", err)
	}
	// DueDateFrom > DueDateTo
	if _, err := uc.List(context.Background(), []uuid.UUID{org}, FinanceBillFilter{Page: 1, PageSize: 20, DueDateFrom: "2026-09-10", DueDateTo: "2026-09-01"}); err != ErrFinanceBillInvalidArgument {
		t.Fatalf("到期日范围倒置应被拒绝，实际错误=%v", err)
	}
}

func TestBuildConfiguredFinanceBillBatchPreviewCasualPartnerDefaultTerms(t *testing.T) {
	orgID := uuid.Must(uuid.NewV7())
	casualClientID := uuid.Must(uuid.NewV7())
	regularClientID := uuid.Must(uuid.NewV7())
	casualSupplierID := uuid.Must(uuid.NewV7())

	// 1. 散客应收费用：预览组应有 IsCasual = true 且 DefaultPaymentTermsDays = 0
	casualReceivable := financeBillableFeeForTest(casualClientID, "100", "94.33962264", "5.66037736", "100")
	casualReceivable.SettlementPartyIsCasual = true
	previewCasual, err := BuildConfiguredFinanceBillBatchPreview(orgID, []*FinanceBillableFee{casualReceivable}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil {
		t.Fatalf("散客预览失败: %v", err)
	}
	if len(previewCasual.Groups) != 1 {
		t.Fatalf("预期 1 组，实际=%d", len(previewCasual.Groups))
	}
	group := previewCasual.Groups[0]
	if !group.IsCasual {
		t.Fatalf("散客应收组 IsCasual 预期为 true，实际为 false")
	}
	if group.DefaultPaymentTermsDays == nil || *group.DefaultPaymentTermsDays != 0 {
		t.Fatalf("散客应收组 DefaultPaymentTermsDays 预期为 0，实际=%v", group.DefaultPaymentTermsDays)
	}

	// 2. 正式客户应收费用：预览组 IsCasual = false 且 DefaultPaymentTermsDays = nil
	regularReceivable := financeBillableFeeForTest(regularClientID, "100", "94.33962264", "5.66037736", "100")
	regularReceivable.SettlementPartyIsCasual = false
	previewRegular, err := BuildConfiguredFinanceBillBatchPreview(orgID, []*FinanceBillableFee{regularReceivable}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil {
		t.Fatalf("正式客户预览失败: %v", err)
	}
	if len(previewRegular.Groups) != 1 {
		t.Fatalf("预期 1 组，实际=%d", len(previewRegular.Groups))
	}
	regGroup := previewRegular.Groups[0]
	if regGroup.IsCasual {
		t.Fatalf("正式客户组 IsCasual 预期为 false，实际为 true")
	}
	if regGroup.DefaultPaymentTermsDays != nil {
		t.Fatalf("正式客户组 DefaultPaymentTermsDays 预期为 nil，实际=%v", regGroup.DefaultPaymentTermsDays)
	}

	// 3. 散客供应商应付费用：预览组 IsCasual = true 且 DefaultPaymentTermsDays = nil（应付不设置默认账期）
	casualPayable := financeBillableFeeForTest(casualSupplierID, "50", "47.16981132", "2.83018868", "50")
	casualPayable.Fee.Direction = OrderFeePayable
	casualPayable.SettlementPartyIsCasual = true
	previewPayable, err := BuildConfiguredFinanceBillBatchPreview(orgID, []*FinanceBillableFee{casualPayable}, PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
	if err != nil {
		t.Fatalf("散客应付预览失败: %v", err)
	}
	if len(previewPayable.Groups) != 1 {
		t.Fatalf("预期 1 组，实际=%d", len(previewPayable.Groups))
	}
	payGroup := previewPayable.Groups[0]
	if !payGroup.IsCasual {
		t.Fatalf("散客应付组 IsCasual 预期为 true，实际为 false")
	}
	if payGroup.DefaultPaymentTermsDays != nil {
		t.Fatalf("散客应付组 DefaultPaymentTermsDays 预期为 nil，实际=%v", payGroup.DefaultPaymentTermsDays)
	}
}

// creditPreviewRepoStub 只覆盖预览链路用到的仓储方法；其余方法不应被调用。
type creditPreviewRepoStub struct {
	FinanceBillRepo
	fees      []*FinanceBillableFee
	summaries map[uuid.UUID]*PartnerCreditSummary
}

func (s *creditPreviewRepoStub) LoadBillableFees(_ context.Context, _ uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error) {
	requested := make(map[uuid.UUID]struct{}, len(feeIDs))
	for _, id := range feeIDs {
		requested[id] = struct{}{}
	}
	fees := make([]*FinanceBillableFee, 0, len(feeIDs))
	for _, fee := range s.fees {
		if _, ok := requested[fee.Fee.ID]; ok {
			fees = append(fees, fee)
		}
	}
	return fees, nil
}

func (s *creditPreviewRepoStub) ValidateBillCurrencies(context.Context, []string) error { return nil }

func (s *creditPreviewRepoStub) GetPartnerCreditSummaries(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]*PartnerCreditSummary, error) {
	return s.summaries, nil
}

func TestPreviewBatchEnrichesFormalTermsAndCreditWarning(t *testing.T) {
	orgID := uuid.Must(uuid.NewV7())
	formalClientID := uuid.Must(uuid.NewV7())
	exceededClientID := uuid.Must(uuid.NewV7())
	casualClientID := uuid.Must(uuid.NewV7())
	supplierID := uuid.Must(uuid.NewV7())

	formalTerms := 30
	creditLimit := decimal.NewFromInt(1000)
	exceededLimit := decimal.NewFromInt(100)
	casualLimit := decimal.NewFromInt(500)
	fees := []*FinanceBillableFee{
		financeBillableFeeForTest(formalClientID, "100", "94.34", "5.66", "100"),
		financeBillableFeeForTest(exceededClientID, "100", "94.34", "5.66", "100"),
		func() *FinanceBillableFee {
			item := financeBillableFeeForTest(casualClientID, "100", "94.34", "5.66", "100")
			item.SettlementPartyIsCasual = true
			return item
		}(),
		func() *FinanceBillableFee {
			item := financeBillableFeeForTest(supplierID, "50", "47.17", "2.83", "50")
			item.Fee.Direction = OrderFeePayable
			item.SettlementPartyIsCasual = true
			return item
		}(),
	}
	repo := &creditPreviewRepoStub{fees: fees, summaries: map[uuid.UUID]*PartnerCreditSummary{
		formalClientID:   {PartnerID: formalClientID, DefaultPaymentTermsDays: &formalTerms, CreditLimitBase: &creditLimit, UnsettledReceivableBase: decimal.NewFromInt(900)},
		exceededClientID: {PartnerID: exceededClientID, CreditLimitBase: &exceededLimit, UnsettledReceivableBase: decimal.NewFromInt(150)},
		casualClientID:   {PartnerID: casualClientID, CreditLimitBase: &casualLimit, UnsettledReceivableBase: decimal.Zero},
	}}
	uc := NewFinanceBillUsecase(repo, NewExchangeRateUsecase(nil, nil), nil, nil, nil)

	input := PreviewFinanceBillBatchInput{GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}}
	// 普通模式一次建账只允许单一方向；应收与应付分别预览。
	receivablePreview, err := uc.PreviewBatch(context.Background(), orgID, PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{fees[0].Fee.ID, fees[1].Fee.ID, fees[2].Fee.ID}, GroupingPolicy: input.GroupingPolicy})
	if err != nil {
		t.Fatalf("应收预览失败: %v", err)
	}
	payablePreview, err := uc.PreviewBatch(context.Background(), orgID, PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{fees[3].Fee.ID}, GroupingPolicy: input.GroupingPolicy})
	if err != nil {
		t.Fatalf("应付预览失败: %v", err)
	}
	groupsByParty := make(map[uuid.UUID]*FinanceBillBatchPreviewGroup, len(receivablePreview.Groups)+len(payablePreview.Groups))
	for _, group := range append(append([]*FinanceBillBatchPreviewGroup{}, receivablePreview.Groups...), payablePreview.Groups...) {
		groupsByParty[group.SettlementPartyID] = group
	}

	// 1. 正式客户：按规则带出默认账期并注入额度比对信息；未超额。
	formalGroup := groupsByParty[formalClientID]
	if formalGroup.DefaultPaymentTermsDays == nil || *formalGroup.DefaultPaymentTermsDays != formalTerms {
		t.Fatalf("正式客户应按规则带出 30 天默认账期, got=%v", formalGroup.DefaultPaymentTermsDays)
	}
	if formalGroup.CreditLimitAmount == nil || !formalGroup.CreditLimitAmount.Equal(creditLimit) || formalGroup.CurrentUnsettledAmount == nil || formalGroup.IsCreditExceeded {
		t.Fatalf("正式客户信用信息注入不符: limit=%v unsettled=%v exceeded=%v", formalGroup.CreditLimitAmount, formalGroup.CurrentUnsettledAmount, formalGroup.IsCreditExceeded)
	}

	// 2. 超额正式客户：标记 IsCreditExceeded，仅供预警展示，不阻断预览。
	exceededGroup := groupsByParty[exceededClientID]
	if !exceededGroup.IsCreditExceeded {
		t.Fatalf("超额客户应标记 IsCreditExceeded")
	}
	if exceededGroup.DefaultPaymentTermsDays != nil {
		t.Fatalf("无默认账期规则的正式客户应保持 nil, got=%v", exceededGroup.DefaultPaymentTermsDays)
	}

	// 3. 散客应收：保持硬编码 0 天底线，不被规则覆盖；信用信息可注入但余额为零不超额。
	casualGroup := groupsByParty[casualClientID]
	if casualGroup.DefaultPaymentTermsDays == nil || *casualGroup.DefaultPaymentTermsDays != 0 {
		t.Fatalf("散客默认账期应保持 0, got=%v", casualGroup.DefaultPaymentTermsDays)
	}
	if casualGroup.IsCreditExceeded {
		t.Fatalf("零余额散客不应标记超额")
	}

	// 4. 应付组：不注入账期与信用信息。
	payableGroup := groupsByParty[supplierID]
	if payableGroup.DefaultPaymentTermsDays != nil || payableGroup.CreditLimitAmount != nil || payableGroup.CurrentUnsettledAmount != nil || payableGroup.IsCreditExceeded {
		t.Fatalf("应付组不应注入账期与信用信息: %+v", payableGroup)
	}
}

// defaultTermsWriteRepoStub 支撑 CreateBatch / 单笔 Create 写入路径的默认账期注入测试。
type defaultTermsWriteRepoStub struct {
	FinanceBillRepo
	fees           []*FinanceBillableFee
	summaries      map[uuid.UUID]*PartnerCreditSummary
	existingBill   *FinanceBill
	existingBatch  *FinanceBillBatch
	savedBatch     *FinanceBillBatch
	createdBill    *FinanceBill
	rateContext    *ExchangeRateContext
	rateByCurrency map[string]decimal.Decimal
}

func (s *defaultTermsWriteRepoStub) LoadBillableFees(_ context.Context, _ uuid.UUID, feeIDs []uuid.UUID) ([]*FinanceBillableFee, error) {
	requested := make(map[uuid.UUID]struct{}, len(feeIDs))
	for _, id := range feeIDs {
		requested[id] = struct{}{}
	}
	fees := make([]*FinanceBillableFee, 0, len(feeIDs))
	for _, fee := range s.fees {
		if _, ok := requested[fee.Fee.ID]; ok {
			fees = append(fees, fee)
		}
	}
	return fees, nil
}

func (s *defaultTermsWriteRepoStub) ValidateBillCurrencies(context.Context, []string) error {
	return nil
}

func (s *defaultTermsWriteRepoStub) HydrateBillSettlementAccounts(_ context.Context, bills []*FinanceBill) error {
	for _, bill := range bills {
		bill.SettlementAccountName = "结算账户"
	}
	return nil
}

func (s *defaultTermsWriteRepoStub) GetPartnerCreditSummaries(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]*PartnerCreditSummary, error) {
	return s.summaries, nil
}

func (s *defaultTermsWriteRepoStub) GetBatchByIdempotencyKey(_ context.Context, _ uuid.UUID, key string) (*FinanceBillBatch, error) {
	if s.savedBatch != nil && s.savedBatch.IdempotencyKey == key {
		return s.savedBatch, nil
	}
	if s.existingBatch != nil && s.existingBatch.IdempotencyKey == key {
		return s.existingBatch, nil
	}
	return nil, nil
}

func (s *defaultTermsWriteRepoStub) CreateBatch(_ context.Context, batch *FinanceBillBatch, _ string, _ *AuditEvent, _ []*AuditEvent) (*FinanceBillBatch, error) {
	s.savedBatch = batch
	return batch, nil
}

func (s *defaultTermsWriteRepoStub) GetByIdempotencyKey(context.Context, uuid.UUID, string) (*FinanceBill, error) {
	return s.existingBill, nil
}

func (s *defaultTermsWriteRepoStub) Create(_ context.Context, bill *FinanceBill, _ *AuditEvent) (*FinanceBill, error) {
	s.createdBill = bill
	return bill, nil
}

func (s *defaultTermsWriteRepoStub) Get(context.Context, []uuid.UUID, uuid.UUID) (*FinanceBill, error) {
	if s.createdBill != nil {
		return s.createdBill, nil
	}
	return s.existingBill, nil
}

func newDefaultTermsWriteUsecase(repo *defaultTermsWriteRepoStub) *FinanceBillUsecase {
	return NewFinanceBillUsecase(repo, NewExchangeRateUsecase(&exchangeRateRepoStub{
		rateContext:    repo.rateContext,
		rateByCurrency: repo.rateByCurrency,
	}, nil), &financeBillTransactorStub{}, nil, nil)
}

func defaultTermsWriteFees(partyID uuid.UUID, isCasual bool) []*FinanceBillableFee {
	fee := financeBillableFeeForTest(partyID, "100", "94.33962264", "5.66037736", "100")
	fee.SettlementPartyIsCasual = isCasual
	return []*FinanceBillableFee{fee}
}

func TestCreateBatchInjectsDefaultPaymentTermsForDirectAPICalls(t *testing.T) {
	const billDate = "2026-09-10"
	organizationID := uuid.Must(uuid.NewV7())
	accountID := uuid.Must(uuid.NewV7())
	// 两阶段预览：先取服务端生成的 group_key，再携带完整配置签发预览令牌。
	runCreateBatch := func(t *testing.T, repo *defaultTermsWriteRepoStub, feeIDs []uuid.UUID, group CreateFinanceBillBatchGroupInput) *FinanceBillBatch {
		t.Helper()
		uc := newDefaultTermsWriteUsecase(repo)
		initial, err := uc.PreviewBatch(context.Background(), organizationID, PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}})
		if err != nil || len(initial.Groups) != 1 {
			t.Fatalf("初次预览失败: err=%v groups=%d", err, len(initial.Groups))
		}
		group.GroupKey = initial.Groups[0].GroupKey
		configs := []FinanceBillBatchPreviewGroupConfig{{GroupKey: group.GroupKey, BillDate: billDate, SettlementAccountID: accountID}}
		preview, err := uc.PreviewBatch(context.Background(), organizationID, PreviewFinanceBillBatchInput{FeeIDs: feeIDs, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}, GroupConfigs: configs})
		if err != nil || preview.PreviewToken == "" {
			t.Fatalf("完整预览失败: token=%q err=%v", preview.PreviewToken, err)
		}
		created, err := uc.CreateBatch(context.Background(), organizationID, uuid.Must(uuid.NewV7()), CreateFinanceBillBatchInput{
			FeeIDs: feeIDs, GroupingPolicy: FinanceBillGroupingPolicy{Mode: "NORMAL"}, Groups: []CreateFinanceBillBatchGroupInput{group}, PreviewToken: preview.PreviewToken, IdempotencyKey: "batch-" + uuid.NewString(),
		})
		if err != nil {
			t.Fatalf("批量建账失败: %v", err)
		}
		return created
	}

	t.Run("API直录不带账期的正式客户按激活规则注入并联动到期日", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		terms := 30
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, false),
			summaries:      map[uuid.UUID]*PartnerCreditSummary{partyID: {PartnerID: partyID, DefaultPaymentTermsDays: &terms}},
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		created := runCreateBatch(t, repo, []uuid.UUID{repo.fees[0].Fee.ID}, CreateFinanceBillBatchGroupInput{StatementTitle: "测试结算单位", BillDate: billDate, SettlementAccountID: accountID})

		if len(created.Bills) != 1 {
			t.Fatalf("预期 1 张账单，实际 %d", len(created.Bills))
		}
		bill := created.Bills[0]
		if bill.PaymentTermsDays == nil || *bill.PaymentTermsDays != terms {
			t.Fatalf("正式客户应注入 30 天默认账期, got=%v", bill.PaymentTermsDays)
		}
		if bill.DueDate == nil || *bill.DueDate != "2026-10-10" {
			t.Fatalf("注入账期后到期日应联动为 2026-10-10, got=%v", bill.DueDate)
		}
	})

	t.Run("API直录不带账期的散客注入0天且到期日为账单日", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, true),
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		created := runCreateBatch(t, repo, []uuid.UUID{repo.fees[0].Fee.ID}, CreateFinanceBillBatchGroupInput{StatementTitle: "测试结算单位", BillDate: billDate, SettlementAccountID: accountID})

		bill := created.Bills[0]
		if bill.PaymentTermsDays == nil || *bill.PaymentTermsDays != 0 {
			t.Fatalf("散客应注入 0 天账期, got=%v", bill.PaymentTermsDays)
		}
		if bill.DueDate == nil || *bill.DueDate != billDate {
			t.Fatalf("散客到期日应等于账单日, got=%v", bill.DueDate)
		}
	})

	t.Run("显式账期与显式到期日不被注入覆盖", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		terms := 30
		explicitZero := 0
		explicitTerms := 15
		explicitDueDate := "2026-09-20"
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, false),
			summaries:      map[uuid.UUID]*PartnerCreditSummary{partyID: {PartnerID: partyID, DefaultPaymentTermsDays: &terms}},
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		feeIDs := []uuid.UUID{repo.fees[0].Fee.ID}

		// 显式 0 天：保持 0，到期日 = 账单日。
		created := runCreateBatch(t, repo, feeIDs, CreateFinanceBillBatchGroupInput{StatementTitle: "测试结算单位", BillDate: billDate, SettlementAccountID: accountID, PaymentTermsDays: &explicitZero})
		if bill := created.Bills[0]; bill.PaymentTermsDays == nil || *bill.PaymentTermsDays != 0 || bill.DueDate == nil || *bill.DueDate != billDate {
			t.Fatalf("显式 0 天不应被覆盖: terms=%v due=%v", bill.PaymentTermsDays, bill.DueDate)
		}

		// 显式 15 天：保持 15。
		created = runCreateBatch(t, repo, feeIDs, CreateFinanceBillBatchGroupInput{StatementTitle: "测试结算单位", BillDate: billDate, SettlementAccountID: accountID, PaymentTermsDays: &explicitTerms})
		if bill := created.Bills[0]; bill.PaymentTermsDays == nil || *bill.PaymentTermsDays != explicitTerms {
			t.Fatalf("显式 15 天不应被覆盖为 30: %v", bill.PaymentTermsDays)
		}

		// 显式到期日（无账期）：到期日是显式输入，注入会使两者冲突，维持现状不注入。
		created = runCreateBatch(t, repo, feeIDs, CreateFinanceBillBatchGroupInput{StatementTitle: "测试结算单位", BillDate: billDate, SettlementAccountID: accountID, DueDate: &explicitDueDate})
		if bill := created.Bills[0]; bill.PaymentTermsDays != nil || bill.DueDate == nil || *bill.DueDate != explicitDueDate {
			t.Fatalf("显式到期日不应注入账期: terms=%v due=%v", bill.PaymentTermsDays, bill.DueDate)
		}
	})
}

func TestCreateInjectsDefaultPaymentTermsForDirectAPICalls(t *testing.T) {
	const billDate = "2026-09-10"
	organizationID := uuid.Must(uuid.NewV7())
	accountID := uuid.Must(uuid.NewV7())

	runCreate := func(t *testing.T, repo *defaultTermsWriteRepoStub) *FinanceBill {
		t.Helper()
		uc := newDefaultTermsWriteUsecase(repo)
		created, err := uc.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{repo.fees[0].Fee.ID}, BillDate: billDate, SettlementAccountID: accountID, IdempotencyKey: "bill-" + uuid.NewString(),
		})
		if err != nil {
			t.Fatalf("单笔建账失败: %v", err)
		}
		return created
	}

	t.Run("API直录不带账期的正式客户按激活规则注入并联动到期日", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		terms := 30
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, false),
			summaries:      map[uuid.UUID]*PartnerCreditSummary{partyID: {PartnerID: partyID, DefaultPaymentTermsDays: &terms}},
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		created := runCreate(t, repo)
		if repo.createdBill.PaymentTermsDays == nil || *repo.createdBill.PaymentTermsDays != terms {
			t.Fatalf("正式客户应注入 30 天默认账期, got=%v", repo.createdBill.PaymentTermsDays)
		}
		if repo.createdBill.DueDate == nil || *repo.createdBill.DueDate != "2026-10-10" {
			t.Fatalf("注入账期后到期日应联动为 2026-10-10, got=%v", repo.createdBill.DueDate)
		}
		if created.ID != repo.createdBill.ID {
			t.Fatalf("返回账单应来自仓储回读")
		}
	})

	t.Run("API直录不带账期的散客注入0天且到期日为账单日", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, true),
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		runCreate(t, repo)
		if repo.createdBill.PaymentTermsDays == nil || *repo.createdBill.PaymentTermsDays != 0 {
			t.Fatalf("散客应注入 0 天账期, got=%v", repo.createdBill.PaymentTermsDays)
		}
		if repo.createdBill.DueDate == nil || *repo.createdBill.DueDate != billDate {
			t.Fatalf("散客到期日应等于账单日, got=%v", repo.createdBill.DueDate)
		}
	})

	t.Run("无规则正式客户与显式传值维持现状", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		terms := 30
		explicitZero := 0
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, false),
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		// 无规则正式客户：摘要无该往来户，注入 nil，账期与到期日保持空。
		runCreate(t, repo)
		if repo.createdBill.PaymentTermsDays != nil || repo.createdBill.DueDate != nil {
			t.Fatalf("无规则正式客户不应注入: terms=%v due=%v", repo.createdBill.PaymentTermsDays, repo.createdBill.DueDate)
		}

		// 显式 0 天：跳过注入，保持 0 与账单日到期。
		repo.summaries = map[uuid.UUID]*PartnerCreditSummary{partyID: {PartnerID: partyID, DefaultPaymentTermsDays: &terms}}
		uc := newDefaultTermsWriteUsecase(repo)
		if _, err := uc.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{repo.fees[0].Fee.ID}, BillDate: billDate, SettlementAccountID: accountID, IdempotencyKey: "bill-" + uuid.NewString(), PaymentTermsDays: &explicitZero,
		}); err != nil {
			t.Fatalf("显式 0 天建账失败: %v", err)
		}
		if repo.createdBill.PaymentTermsDays == nil || *repo.createdBill.PaymentTermsDays != 0 || repo.createdBill.DueDate == nil || *repo.createdBill.DueDate != billDate {
			t.Fatalf("显式 0 天不应被覆盖: terms=%v due=%v", repo.createdBill.PaymentTermsDays, repo.createdBill.DueDate)
		}
	})

	t.Run("注入后的同请求重放命中幂等返回原账单", func(t *testing.T) {
		partyID := uuid.Must(uuid.NewV7())
		terms := 30
		repo := &defaultTermsWriteRepoStub{
			fees:           defaultTermsWriteFees(partyID, false),
			summaries:      map[uuid.UUID]*PartnerCreditSummary{partyID: {PartnerID: partyID, DefaultPaymentTermsDays: &terms}},
			rateContext:    &ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
			rateByCurrency: map[string]decimal.Decimal{},
		}
		uc := newDefaultTermsWriteUsecase(repo)
		input := CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{repo.fees[0].Fee.ID}, BillDate: billDate, SettlementAccountID: accountID, IdempotencyKey: "bill-replay",
		}
		if _, err := uc.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), input); err != nil {
			t.Fatalf("首次建账失败: %v", err)
		}
		// 首次创建已注入 30 天并落库；重放同一原始请求应按注入后的生效值比对并返回原账单。
		repo.createdBill = nil
		repo.existingBill = &FinanceBill{
			SettlementPartyID: partyID, SettlementPartyName: "验收客户", SettlementAccountID: accountID, BillDate: billDate,
			PaymentTermsDays: &terms, DueDate: func() *string { value := "2026-10-10"; return &value }(),
			Lines: []*FinanceBillLine{{OrderFeeID: repo.fees[0].Fee.ID}},
		}
		repo.existingBill.StatementTitle = &repo.existingBill.SettlementPartyName
		existing, err := uc.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), input)
		if err != nil {
			t.Fatalf("重放同一请求应命中幂等，实际错误=%v", err)
		}
		if existing.ID != repo.existingBill.ID {
			t.Fatalf("重放应返回首次创建的账单")
		}
		if repo.createdBill != nil {
			t.Fatalf("重放不应再次落库")
		}
	})
}
