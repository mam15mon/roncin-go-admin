package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func nettingBillForTest(id uuid.UUID, direction OrderFeeDirection, total, verified, netted string, version uint64) *FinanceNettingBill {
	return &FinanceNettingBill{
		ID: id, BillNo: "B" + id.String()[:8], BillDate: "2026-09-10", Status: FinanceBillConfirmed,
		Direction: direction, SettlementPartyID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		SettlementPartyName: "验收同行", Currency: "CNY", BaseCurrency: "CNY",
		ExchangeRate: decimal.NewFromInt(1), TotalAmount: decimal.RequireFromString(total),
		VerifiedAmount: decimal.RequireFromString(verified), NettedAmount: decimal.RequireFromString(netted),
		Version: version,
	}
}

func nettingBillVersions(bills ...*FinanceNettingBill) []FinanceNettingBillVersion {
	result := make([]FinanceNettingBillVersion, 0, len(bills))
	for _, bill := range bills {
		result = append(result, FinanceNettingBillVersion{BillID: bill.ID, ExpectedVersion: bill.Version})
	}
	return result
}

func TestApplyFinanceNettingPreviewAmountsEqualOffset(t *testing.T) {
	preview := &FinanceNettingPreview{
		ReceivableBills: []*FinanceNettingBillBalance{
			{TotalAmount: decimal.RequireFromString("100"), NettedAmount: decimal.RequireFromString("20"), AvailableAmount: decimal.RequireFromString("80")},
			{TotalAmount: decimal.RequireFromString("60"), VerifiedAmount: decimal.RequireFromString("10"), AvailableAmount: decimal.RequireFromString("50")},
		},
		PayableBills: []*FinanceNettingBillBalance{
			{TotalAmount: decimal.RequireFromString("80"), AvailableAmount: decimal.RequireFromString("80")},
			{TotalAmount: decimal.RequireFromString("50"), NettedAmount: decimal.RequireFromString("10"), AvailableAmount: decimal.RequireFromString("40")},
		},
	}
	ApplyFinanceNettingPreviewAmounts(preview)
	if !preview.ReceivableAvailableAmount.Equal(decimal.RequireFromString("130")) || !preview.PayableAvailableAmount.Equal(decimal.RequireFromString("120")) {
		t.Fatalf("可用余额汇总错误: %#v", preview)
	}
	if !preview.OffsetAmount.Equal(decimal.RequireFromString("120")) {
		t.Fatalf("抵销额应为双方较小值: %#v", preview)
	}
	if !preview.NetReceivableAmount.Equal(decimal.RequireFromString("10")) || !preview.NetPayableAmount.IsZero() {
		t.Fatalf("净应收/净应付计算错误: %#v", preview)
	}
}

func TestApplyFinanceNettingPreviewAmountsNetPayableAndOneSideEmpty(t *testing.T) {
	preview := &FinanceNettingPreview{
		ReceivableBills: []*FinanceNettingBillBalance{{TotalAmount: decimal.RequireFromString("30"), AvailableAmount: decimal.RequireFromString("30")}},
		PayableBills:    []*FinanceNettingBillBalance{{TotalAmount: decimal.RequireFromString("90"), AvailableAmount: decimal.RequireFromString("90")}},
	}
	ApplyFinanceNettingPreviewAmounts(preview)
	if !preview.OffsetAmount.Equal(decimal.RequireFromString("30")) || !preview.NetPayableAmount.Equal(decimal.RequireFromString("60")) || !preview.NetReceivableAmount.IsZero() {
		t.Fatalf("净应付场景计算错误: %#v", preview)
	}
	empty := &FinanceNettingPreview{ReceivableBills: []*FinanceNettingBillBalance{{TotalAmount: decimal.RequireFromString("30"), AvailableAmount: decimal.RequireFromString("30")}}}
	ApplyFinanceNettingPreviewAmounts(empty)
	if !empty.OffsetAmount.IsZero() || !empty.NetReceivableAmount.Equal(decimal.RequireFromString("30")) {
		t.Fatalf("单边无可用余额时不得抵销: %#v", empty)
	}
}

func TestPlanFinanceNettingGreedyAllocationAndRemainingBalance(t *testing.T) {
	receivableFirst, receivableSecond := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	payable := uuid.Must(uuid.NewV7())
	bills := []*FinanceNettingBill{
		nettingBillForTest(receivableFirst, OrderFeeReceivable, "100", "10", "0", 1),
		nettingBillForTest(receivableSecond, OrderFeeReceivable, "50", "0", "20", 1),
		nettingBillForTest(payable, OrderFeePayable, "200", "0", "0", 3),
	}
	plan, err := PlanFinanceNetting(bills, nettingBillVersions(bills...))
	if err != nil {
		t.Fatalf("对冲计划失败: %v", err)
	}
	if !plan.Amount.Equal(decimal.RequireFromString("120")) {
		t.Fatalf("对冲金额应为双方可用余额较小值: %s", plan.Amount)
	}
	receivableSum, payableSum := decimal.Zero, decimal.Zero
	allocated := map[uuid.UUID]decimal.Decimal{}
	for _, allocation := range plan.Allocations {
		if allocation.Direction == OrderFeeReceivable {
			receivableSum = receivableSum.Add(allocation.Amount)
		} else {
			payableSum = payableSum.Add(allocation.Amount)
		}
		allocated[allocation.BillID] = allocated[allocation.BillID].Add(allocation.Amount)
	}
	if !receivableSum.Equal(plan.Amount) || !payableSum.Equal(plan.Amount) {
		t.Fatalf("双向分摊合计都必须等于对冲金额: %s %s %s", receivableSum, payableSum, plan.Amount)
	}
	if v, ok := allocated[receivableFirst]; !ok || !v.Equal(decimal.RequireFromString("90")) {
		t.Fatalf("应收账单应按可用余额 90 分摊: %#v", allocated)
	}
	if v, ok := allocated[receivableSecond]; !ok || !v.Equal(decimal.RequireFromString("30")) {
		t.Fatalf("应收第二张应仅分摊剩余 30: %#v", allocated)
	}
	if v, ok := allocated[payable]; !ok || !v.Equal(plan.Amount) {
		t.Fatalf("应付账单应完整分摊: %#v", allocated)
	}
}

func TestPlanFinanceNettingRejections(t *testing.T) {
	receivable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "100", "0", "0", 1)
	partyMismatch := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "100", "0", "0", 1)
	partyMismatch.SettlementPartyID = uuid.New()
	currencyMismatch := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "100", "0", "0", 1)
	currencyMismatch.Currency = "USD"
	baseMismatch := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "100", "0", "0", 1)
	baseMismatch.BaseCurrency = "USD"
	draftBill := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "100", "0", "0", 1)
	draftBill.Status = FinanceBillDraft
	fullyNetted := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "100", "0", "100", 1)
	anotherReceivable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "80", "0", "0", 1)
	cases := []struct {
		name        string
		bills       []*FinanceNettingBill
		expectedErr error
	}{
		{"跨结算单位拒绝", []*FinanceNettingBill{receivable, partyMismatch}, ErrFinanceNettingMismatch},
		{"跨账单币种拒绝", []*FinanceNettingBill{receivable, currencyMismatch}, ErrFinanceNettingMismatch},
		{"跨本位币拒绝", []*FinanceNettingBill{receivable, baseMismatch}, ErrFinanceNettingMismatch},
		{"未确认账单拒绝", []*FinanceNettingBill{receivable, draftBill}, ErrFinanceNettingTransition},
		{"单方向拒绝", []*FinanceNettingBill{receivable, anotherReceivable}, ErrFinanceNettingDirection},
		{"单边可用余额耗尽拒绝", []*FinanceNettingBill{receivable, fullyNetted}, ErrFinanceNettingBalance},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := PlanFinanceNetting(testCase.bills, nettingBillVersions(testCase.bills...)); err != testCase.expectedErr {
				t.Fatalf("错误=%v，期望=%v", err, testCase.expectedErr)
			}
		})
	}
}

func TestPlanFinanceNettingRejectsVersionMismatchAndDuplicates(t *testing.T) {
	receivable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "100", "0", "0", 2)
	payable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "100", "0", "0", 2)
	versions := []FinanceNettingBillVersion{
		{BillID: receivable.ID, ExpectedVersion: 1},
		{BillID: payable.ID, ExpectedVersion: 2},
	}
	if _, err := PlanFinanceNetting([]*FinanceNettingBill{receivable, payable}, versions); err != ErrFinanceNettingBillVersionConflict {
		t.Fatalf("版本不匹配错误=%v", err)
	}
	duplicated := append(nettingBillVersions(receivable, payable), FinanceNettingBillVersion{BillID: receivable.ID, ExpectedVersion: 2})
	if _, err := PlanFinanceNetting([]*FinanceNettingBill{receivable, payable}, duplicated); err != ErrFinanceNettingMismatch {
		t.Fatalf("重复账单错误=%v", err)
	}
}

func TestValidateFinanceNettingConfirmation(t *testing.T) {
	receivable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "100", "0", "0", 1)
	payable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "80", "0", "0", 1)
	bills := []*FinanceNettingBill{receivable, payable}
	netting := &FinanceNetting{
		SettlementPartyID: receivable.SettlementPartyID, Currency: "CNY",
		Amount: decimal.RequireFromString("80"),
		Allocations: []*FinanceNettingAllocation{
			{BillID: receivable.ID, Direction: OrderFeeReceivable, Amount: decimal.RequireFromString("80")},
			{BillID: payable.ID, Direction: OrderFeePayable, Amount: decimal.RequireFromString("80")},
		},
	}
	if err := ValidateFinanceNettingConfirmation(netting, bills); err != nil {
		t.Fatalf("合法确认应通过: %v", err)
	}
	insufficient := nettingBillForTest(payable.ID, OrderFeePayable, "80", "30", "0", 1)
	if err := ValidateFinanceNettingConfirmation(netting, []*FinanceNettingBill{receivable, insufficient}); err != ErrFinanceNettingBalance {
		t.Fatalf("余额不足错误=%v", err)
	}
	wrongDirection := &FinanceNetting{
		SettlementPartyID: receivable.SettlementPartyID, Currency: "CNY",
		Amount: decimal.RequireFromString("80"),
		Allocations: []*FinanceNettingAllocation{
			{BillID: receivable.ID, Direction: OrderFeeReceivable, Amount: decimal.RequireFromString("80")},
			{BillID: payable.ID, Direction: OrderFeeReceivable, Amount: decimal.RequireFromString("80")},
		},
	}
	if err := ValidateFinanceNettingConfirmation(wrongDirection, bills); err != ErrFinanceNettingInvalid {
		t.Fatalf("分摊方向与账单不一致错误=%v", err)
	}
	unbalanced := &FinanceNetting{
		SettlementPartyID: receivable.SettlementPartyID, Currency: "CNY",
		Amount: decimal.RequireFromString("80"),
		Allocations: []*FinanceNettingAllocation{
			{BillID: receivable.ID, Direction: OrderFeeReceivable, Amount: decimal.RequireFromString("60")},
			{BillID: payable.ID, Direction: OrderFeePayable, Amount: decimal.RequireFromString("80")},
		},
	}
	if err := ValidateFinanceNettingConfirmation(unbalanced, bills); err != ErrFinanceNettingInvalid {
		t.Fatalf("双向分摊合计不等错误=%v", err)
	}
}

func batchNettingBillForTest(id uuid.UUID, partyID uuid.UUID, direction OrderFeeDirection, total string) *FinanceBill {
	return &FinanceBill{
		ID: id, OrganizationID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		BillNo: "B" + id.String()[:8], Direction: direction, Status: FinanceBillDraft,
		SettlementPartyID: partyID, SettlementPartyName: "验收同行", Currency: "CNY", BaseCurrency: "CNY",
		ExchangeRate: decimal.NewFromInt(1), TotalAmount: decimal.RequireFromString(total), Version: 1,
	}
}

func TestPlanBatchFinanceNettingsGroupsByPartyAndCurrency(t *testing.T) {
	organizationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	batchID := uuid.New()
	partyA, partyB := uuid.New(), uuid.New()
	bills := []*FinanceBill{
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyA, OrderFeeReceivable, "100"),
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyA, OrderFeePayable, "40"),
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyA, OrderFeePayable, "30"),
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyB, OrderFeeReceivable, "10"),
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyB, OrderFeePayable, "90"),
	}
	nettings, err := planBatchFinanceNettings(organizationID, batchID, "batch-key", bills)
	if err != nil {
		t.Fatalf("批次对冲计划失败: %v", err)
	}
	if len(nettings) != 2 {
		t.Fatalf("应按结算单位与币种生成两张对冲单: %#v", nettings)
	}
	byParty := map[uuid.UUID]*FinanceNetting{}
	for _, netting := range nettings {
		byParty[netting.SettlementPartyID] = netting
	}
	if !byParty[partyA].Amount.Equal(decimal.RequireFromString("70")) || !byParty[partyB].Amount.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("对冲金额错误: A=%s B=%s", byParty[partyA].Amount, byParty[partyB].Amount)
	}
	for _, netting := range nettings {
		if netting.Status != FinanceNettingDraft || netting.BatchID == nil || *netting.BatchID != batchID {
			t.Fatalf("批次对冲单应为草稿并关联批次: %#v", netting)
		}
		receivableSum, payableSum := decimal.Zero, decimal.Zero
		activeSeen := false
		for _, allocation := range netting.Allocations {
			if allocation.Active {
				activeSeen = true
			}
			if allocation.Direction == OrderFeeReceivable {
				receivableSum = receivableSum.Add(allocation.Amount)
			} else {
				payableSum = payableSum.Add(allocation.Amount)
			}
		}
		if activeSeen {
			t.Fatalf("批次对冲分摊初始必须未生效: %#v", netting.Allocations)
		}
		if !receivableSum.Equal(netting.Amount) || !payableSum.Equal(netting.Amount) {
			t.Fatalf("双向分摊合计必须等于对冲金额: %#v", netting)
		}
		if netting.IdempotencyKey != financeBillBatchNettingKey("batch-key", netting.SettlementPartyID, netting.Currency) {
			t.Fatalf("幂等键不具确定性: %s", netting.IdempotencyKey)
		}
	}
}

func TestPlanBatchFinanceNettingsRejectsInvalidInputs(t *testing.T) {
	organizationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	partyID := uuid.New()
	singleDirection := []*FinanceBill{
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyID, OrderFeeReceivable, "100"),
	}
	if _, err := planBatchFinanceNettings(organizationID, uuid.New(), "batch-key", singleDirection); err != ErrFinanceNettingSingleDirection {
		t.Fatalf("单方向批次对冲错误=%v", err)
	}
	baseMismatch := batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyID, OrderFeePayable, "50")
	baseMismatch.BaseCurrency = "USD"
	if _, err := planBatchFinanceNettings(organizationID, uuid.New(), "batch-key", []*FinanceBill{
		batchNettingBillForTest(uuid.Must(uuid.NewV7()), partyID, OrderFeeReceivable, "100"), baseMismatch,
	}); err != ErrFinanceNettingMismatch {
		t.Fatalf("跨本位币批次对冲错误=%v", err)
	}
}

func TestBuildFinanceNettingPairsComputesGrossOffsetAndNet(t *testing.T) {
	partyID := uuid.New()
	groups := []*FinanceBillBatchPreviewGroup{
		{SettlementPartyID: partyID, SettlementPartyName: "验收同行", Currency: "CNY", Direction: OrderFeeReceivable, TotalAmount: decimal.RequireFromString("100")},
		{SettlementPartyID: partyID, SettlementPartyName: "验收同行", Currency: "CNY", Direction: OrderFeePayable, TotalAmount: decimal.RequireFromString("70")},
		{SettlementPartyID: partyID, SettlementPartyName: "验收同行", Currency: "USD", Direction: OrderFeeReceivable, TotalAmount: decimal.RequireFromString("10")},
		{SettlementPartyID: partyID, SettlementPartyName: "验收同行", Currency: "USD", Direction: OrderFeePayable, TotalAmount: decimal.RequireFromString("25")},
	}
	pairs := buildFinanceNettingPairs(groups)
	if len(pairs) != 2 {
		t.Fatalf("应按币种分别成对: %#v", pairs)
	}
	byCurrency := map[string]*FinanceBillBatchNettingPair{}
	for _, pair := range pairs {
		byCurrency[pair.Currency] = pair
	}
	cny, usd := byCurrency["CNY"], byCurrency["USD"]
	if !cny.OffsetAmount.Equal(decimal.RequireFromString("70")) || !cny.NetReceivableAmount.Equal(decimal.RequireFromString("30")) || !cny.NetPayableAmount.IsZero() {
		t.Fatalf("CNY 对汇总错误: %#v", cny)
	}
	if !usd.OffsetAmount.Equal(decimal.RequireFromString("10")) || !usd.NetPayableAmount.Equal(decimal.RequireFromString("15")) || !usd.NetReceivableAmount.IsZero() {
		t.Fatalf("USD 对汇总错误: %#v", usd)
	}
}

type financeNettingRepoStub struct {
	FinanceNettingRepo
	existing    *FinanceNetting
	created     *FinanceNetting
	lockedBills []*FinanceNettingBill
	confirmErr  error
}

func (s *financeNettingRepoStub) GetByKey(context.Context, uuid.UUID, string) (*FinanceNetting, error) {
	if s.existing != nil {
		return s.existing, nil
	}
	return s.created, nil
}

func (s *financeNettingRepoStub) Get(context.Context, []uuid.UUID, uuid.UUID) (*FinanceNetting, error) {
	if s.existing != nil {
		return s.existing, nil
	}
	return s.created, nil
}

func (s *financeNettingRepoStub) LockNettingBills(context.Context, uuid.UUID, []uuid.UUID) ([]*FinanceNettingBill, error) {
	return s.lockedBills, nil
}

func (s *financeNettingRepoStub) LockNetting(context.Context, []uuid.UUID, uuid.UUID) (*FinanceNetting, error) {
	return s.existing, nil
}

func (s *financeNettingRepoStub) Create(_ context.Context, _ uuid.UUID, _ uuid.UUID, netting *FinanceNetting, _ *AuditEvent) (*FinanceNetting, error) {
	s.created = netting
	return netting, nil
}

func (s *financeNettingRepoStub) Confirm(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uint64, *AuditEvent) (*FinanceNetting, error) {
	return nil, s.confirmErr
}

func TestFinanceNettingUsecaseCreateBuildsDraftNettingInTransaction(t *testing.T) {
	receivable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "100", "0", "0", 1)
	payable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "60", "0", "0", 1)
	repo := &financeNettingRepoStub{lockedBills: []*FinanceNettingBill{receivable, payable}}
	uc := NewFinanceNettingUsecase(repo, &financeBillTransactorStub{})
	input := CreateFinanceNettingInput{
		Bills:          nettingBillVersions(receivable, payable),
		IdempotencyKey: "netting-key",
	}
	created, err := uc.Create(t.Context(), receivable.SettlementPartyID, uuid.New(), input)
	if err != nil {
		t.Fatalf("创建对冲失败: %v", err)
	}
	if created.Status != FinanceNettingDraft || !created.Amount.Equal(decimal.RequireFromString("60")) || created.Version != 1 {
		t.Fatalf("对冲单字段错误: %#v", created)
	}
	if len(created.Allocations) != 2 || created.IdempotencyKey != "netting-key" || created.RequestHash == "" {
		t.Fatalf("对冲分摊或幂等信息错误: %#v", created)
	}
}

func TestFinanceNettingUsecaseCreateIdempotency(t *testing.T) {
	receivable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "100", "0", "0", 1)
	payable := nettingBillForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "60", "0", "0", 1)
	existing := &FinanceNetting{ID: uuid.New(), Status: FinanceNettingConfirmed}
	repo := &financeNettingRepoStub{existing: existing, lockedBills: []*FinanceNettingBill{receivable, payable}}
	uc := NewFinanceNettingUsecase(repo, &financeBillTransactorStub{})
	input := CreateFinanceNettingInput{Bills: nettingBillVersions(receivable, payable), IdempotencyKey: "netting-key"}
	existing.RequestHash = financeNettingRequestHash(input)
	if _, err := uc.Create(t.Context(), receivable.SettlementPartyID, uuid.New(), input); err != nil {
		t.Fatalf("幂等重放应返回原对冲单: %v", err)
	}
	if repo.created != nil {
		t.Fatalf("幂等重放不应再次创建对冲单")
	}
	other := input
	other.Bills = []FinanceNettingBillVersion{{BillID: receivable.ID, ExpectedVersion: 1}}
	existing.RequestHash = financeNettingRequestHash(other)
	if _, err := uc.Create(t.Context(), receivable.SettlementPartyID, uuid.New(), input); err != ErrFinanceNettingIdempotency {
		t.Fatalf("同键不同意图错误=%v", err)
	}
}

func TestFinanceNettingUsecaseCreateValidatesInput(t *testing.T) {
	uc := NewFinanceNettingUsecase(&financeNettingRepoStub{}, &financeBillTransactorStub{})
	if _, err := uc.Create(t.Context(), uuid.Nil, uuid.New(), CreateFinanceNettingInput{Bills: []FinanceNettingBillVersion{{BillID: uuid.New(), ExpectedVersion: 1}}, IdempotencyKey: "k"}); err != ErrFinanceNettingInvalid {
		t.Fatalf("组织缺失错误=%v", err)
	}
	if _, err := uc.Create(t.Context(), uuid.New(), uuid.New(), CreateFinanceNettingInput{Bills: []FinanceNettingBillVersion{{BillID: uuid.New(), ExpectedVersion: 1}}}); err != ErrFinanceNettingInvalid {
		t.Fatalf("幂等键缺失错误=%v", err)
	}
}

func TestFinanceNettingUsecaseConfirmRejectsVersionConflict(t *testing.T) {
	existing := &FinanceNetting{ID: uuid.New(), OrganizationID: uuid.New(), Status: FinanceNettingDraft, Version: 2}
	repo := &financeNettingRepoStub{existing: existing}
	uc := NewFinanceNettingUsecase(repo, &financeBillTransactorStub{})
	if _, err := uc.Confirm(t.Context(), []uuid.UUID{existing.OrganizationID}, uuid.New(), existing.ID, 1); err != ErrFinanceNettingVersionConflict {
		t.Fatalf("版本冲突错误=%v", err)
	}
}
