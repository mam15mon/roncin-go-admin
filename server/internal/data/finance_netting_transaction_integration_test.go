package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/shopspring/decimal"
)

type nettingPostgresFixture struct {
	*financeBillPostgresFixture
	receivableBillID uuid.UUID
	payableBillID    uuid.UUID
	actorUserID      uuid.UUID
}

type nettingCreateResult struct {
	netting *biz.FinanceNetting
	err     error
}

func newNettingPostgresFixture(t *testing.T, data *Data) *nettingPostgresFixture {
	t.Helper()
	base := newFinanceBillPostgresFixture(t, data)
	fixture := &nettingPostgresFixture{financeBillPostgresFixture: base}
	t.Cleanup(fixture.cleanupNetting)
	ctx := context.Background()

	// 确认/反转写入 confirmed_by 等外键，操作者必须是真实用户。
	actor, err := data.db.User.Create().SetDisplayName("对冲集成测试用户-" + base.suffix).Save(ctx)
	if err != nil {
		t.Fatalf("创建对冲操作用户: %v", err)
	}
	fixture.actorUserID = actor.ID
	if _, err := data.db.NumberRule.Create().SetOrganizationID(fixture.organizationID).SetDocumentType(numberruleent.DocumentTypeNetting).SetPrefix("NT-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试对冲编号规则: %v", err)
	}

	receivableCreate := data.db.FinanceBill.Create().SetOrganizationID(fixture.organizationID).SetBillNo("BILL-NTR-" + fixture.suffix).SetIdempotencyKey("bill-netting-receivable-" + fixture.suffix).SetDirection(financebillent.DirectionRECEIVABLE).SetStatus(financebillent.StatusCONFIRMED).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1.00000000").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetTotalAmount("100.00000000").SetNetAmount("100.00000000").SetTaxAmount("0.00000000").SetBaseCurrencyAmount("100.00000000").SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1)
	receivable, err := withTestFinanceBillSettlementAccountSnapshot(receivableCreate, fixture.accountID, "CNY").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试应收账单: %v", err)
	}
	fixture.receivableBillID = receivable.ID

	payableCreate := data.db.FinanceBill.Create().SetOrganizationID(fixture.organizationID).SetBillNo("BILL-NTP-" + fixture.suffix).SetIdempotencyKey("bill-netting-payable-" + fixture.suffix).SetDirection(financebillent.DirectionPAYABLE).SetStatus(financebillent.StatusCONFIRMED).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1.00000000").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetTotalAmount("60.00000000").SetNetAmount("60.00000000").SetTaxAmount("0.00000000").SetBaseCurrencyAmount("60.00000000").SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1)
	payable, err := withTestFinanceBillSettlementAccountSnapshot(payableCreate, fixture.accountID, "CNY").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试应付账单: %v", err)
	}
	fixture.payableBillID = payable.ID
	return fixture
}

func (f *nettingPostgresFixture) newUsecase() *biz.FinanceNettingUsecase {
	return biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(f.data), f.data, nil, nil)
}

func (f *nettingPostgresFixture) input(key string) biz.CreateFinanceNettingInput {
	return biz.CreateFinanceNettingInput{
		Bills: []biz.FinanceNettingBillVersion{
			{BillID: f.receivableBillID, ExpectedVersion: 1},
			{BillID: f.payableBillID, ExpectedVersion: 1},
		},
		IdempotencyKey: "netting-" + key + "-" + f.suffix,
	}
}

func (f *nettingPostgresFixture) billBalance(t *testing.T, billID uuid.UUID) (netted, unverified decimal.Decimal) {
	t.Helper()
	bill, err := NewFinanceBillRepo(f.data).Get(context.Background(), []uuid.UUID{f.organizationID}, billID)
	if err != nil {
		t.Fatalf("读取账单余额: %v", err)
	}
	return bill.NettedAmount, bill.UnverifiedAmount
}

func (f *nettingPostgresFixture) requireNettingCount(t *testing.T, want int) {
	t.Helper()
	count, err := f.data.db.FinanceNetting.Query().Where(financenettingent.OrganizationIDEQ(f.organizationID)).Count(context.Background())
	if err != nil || count != want {
		t.Fatalf("对冲单数量 = %d，期望 %d，error=%v", count, want, err)
	}
}

func (f *nettingPostgresFixture) cleanupNetting() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := f.data.db.FinanceNettingAllocation.Delete().Where(financenettingallocationent.HasNettingWith(financenettingent.OrganizationIDEQ(f.organizationID))).Exec(ctx); err != nil {
		f.t.Errorf("清理测试对冲分摊: %v", err)
	}
	if _, err := f.data.db.FinanceNetting.Delete().Where(financenettingent.OrganizationIDEQ(f.organizationID)).Exec(ctx); err != nil {
		f.t.Errorf("清理测试对冲单: %v", err)
	}
	if f.actorUserID != uuid.Nil {
		if err := f.data.db.User.DeleteOneID(f.actorUserID).Exec(ctx); err != nil {
			f.t.Errorf("清理对冲操作用户: %v", err)
		}
	}
}

func TestFinanceNettingTransactionPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("相同幂等键并发创建返回同一对冲单", func(t *testing.T) {
		fixture := newNettingPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		results := createNettingsConcurrently(usecase, fixture.organizationID, fixture.actorID, fixture.input("same-key"), fixture.input("same-key"))
		for index, result := range results {
			if result.err != nil || result.netting == nil {
				t.Fatalf("第 %d 个并发请求创建对冲: %#v error=%v", index+1, result.netting, result.err)
			}
		}
		if results[0].netting.ID != results[1].netting.ID || results[0].netting.NettingNo != results[1].netting.NettingNo {
			t.Fatalf("相同幂等键返回了不同对冲单: %s/%s vs %s/%s", results[0].netting.ID, results[0].netting.NettingNo, results[1].netting.ID, results[1].netting.NettingNo)
		}
		if !results[0].netting.Amount.Equal(decimal.RequireFromString("60")) {
			t.Fatalf("对冲金额应为双方较小值 60: %s", results[0].netting.Amount)
		}
		fixture.requireNettingCount(t, 1)
		auditCount, err := data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID), auditlogent.ActionEQ("finance.netting.create")).Count(context.Background())
		if err != nil || auditCount != 1 {
			t.Fatalf("对冲创建审计数 = %d，期望 1，error=%v", auditCount, err)
		}
	})

	t.Run("并发确认两张草稿对冲恰好一个成功", func(t *testing.T) {
		fixture := newNettingPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		first, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("confirm-a"))
		if err != nil {
			t.Fatalf("创建第一张对冲: %v", err)
		}
		second, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.input("confirm-b"))
		if err != nil {
			t.Fatalf("创建第二张对冲: %v", err)
		}

		start := make(chan struct{})
		results := make(chan error, 2)
		for _, netting := range []*biz.FinanceNetting{first, second} {
			go func(target *biz.FinanceNetting) {
				<-start
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_, confirmErr := usecase.Confirm(ctx, []uuid.UUID{fixture.organizationID}, fixture.actorUserID, target.ID, target.Version)
				results <- confirmErr
			}(netting)
		}
		close(start)
		collected := []error{<-results, <-results}
		successes, balanceRejected := 0, 0
		for _, resultErr := range collected {
			switch {
			case resultErr == nil:
				successes++
			case errors.Is(resultErr, biz.ErrFinanceNettingBalance):
				balanceRejected++
			default:
				t.Fatalf("并发确认出现意外错误: %v", resultErr)
			}
		}
		if successes != 1 || balanceRejected != 1 {
			t.Fatalf("并发确认结果 = 成功 %d、余额拒绝 %d，期望各 1", successes, balanceRejected)
		}
		receivableNetted, receivableUnverified := fixture.billBalance(t, fixture.receivableBillID)
		payableNetted, _ := fixture.billBalance(t, fixture.payableBillID)
		if !receivableNetted.Equal(decimal.RequireFromString("60")) || !receivableUnverified.Equal(decimal.RequireFromString("40")) {
			t.Fatalf("应收账单对冲后余额错误: netted=%s unverified=%s", receivableNetted, receivableUnverified)
		}
		if !payableNetted.Equal(decimal.RequireFromString("60")) {
			t.Fatalf("应付账单对冲后余额错误: netted=%s", payableNetted)
		}
	})
}

func createNettingsConcurrently(usecase *biz.FinanceNettingUsecase, organizationID, actorID uuid.UUID, inputs ...biz.CreateFinanceNettingInput) []nettingCreateResult {
	start := make(chan struct{})
	results := make(chan nettingCreateResult, len(inputs))
	for _, input := range inputs {
		input := input
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			netting, err := usecase.Create(ctx, organizationID, actorID, input)
			results <- nettingCreateResult{netting: netting, err: err}
		}()
	}
	close(start)
	collected := make([]nettingCreateResult, 0, len(inputs))
	for range inputs {
		collected = append(collected, <-results)
	}
	return collected
}

// TestFinanceNettingPersistsBothBaseAmountsAndGainLoss 验证对冲创建路径端到端
// 落库并读回应收侧/应付侧本位币与对冲汇差（B2）。
func TestFinanceNettingPersistsBothBaseAmountsAndGainLoss(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	// 夹具在子测试内创建，保证其 t.Cleanup 在外层 defer 关闭数据库之前执行。
	t.Run("两端本位币与汇差落库并读回", func(t *testing.T) {
		fixture := newNettingPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		ctx := context.Background()

		// 双方同为 USD、不同汇率：应收 6.5、应付 7.0；对冲额为可用余额较小值 60。
		usdBill := func(key string, direction financebillent.Direction, total, rate string) uuid.UUID {
			create := data.db.FinanceBill.Create().
				SetOrganizationID(fixture.organizationID).
				SetBillNo("BILL-NTB-" + key + "-" + fixture.suffix).
				SetIdempotencyKey("bill-netting-base-" + key + "-" + fixture.suffix).
				SetDirection(direction).
				SetStatus(financebillent.StatusCONFIRMED).
				SetSettlementPartyID(fixture.partnerID).
				SetSettlementPartyName("账单事务测试客户-" + fixture.suffix).
				SetCurrency("USD").
				SetBaseCurrency("CNY").
				SetExchangeRate(rate).
				SetExchangeRateSource(financebillent.ExchangeRateSourceMANUAL).
				SetExchangeRateDate(financeBillIntegrationDate).
				SetTotalAmount(total).
				SetNetAmount(total).
				SetTaxAmount("0.00000000").
				SetBaseCurrencyAmount(decimal.RequireFromString(total).Mul(decimal.RequireFromString(rate)).StringFixed(8)).
				SetFeeCount(1).
				SetBillDate(financeBillIntegrationDate).
				SetVersion(1)
			bill, err := withTestFinanceBillSettlementAccountSnapshot(create, fixture.accountID, "USD").Save(ctx)
			if err != nil {
				t.Fatalf("创建 USD 测试账单 %s: %v", key, err)
			}
			return bill.ID
		}
		receivableID := usdBill("rec", financebillent.DirectionRECEIVABLE, "100", "6.50000000")
		payableID := usdBill("pay", financebillent.DirectionPAYABLE, "60", "7.00000000")

		netting, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateFinanceNettingInput{
			Bills: []biz.FinanceNettingBillVersion{
				{BillID: receivableID, ExpectedVersion: 1},
				{BillID: payableID, ExpectedVersion: 1},
			},
			IdempotencyKey: "netting-base-amounts-" + fixture.suffix,
		})
		if err != nil {
			t.Fatalf("创建两端本位币对冲单失败: %v", err)
		}
		if !netting.Amount.Equal(decimal.RequireFromString("60")) {
			t.Fatalf("对冲金额应为 60: %s", netting.Amount)
		}
		if !netting.BaseCurrencyAmount.Equal(decimal.RequireFromString("390")) {
			t.Fatalf("应收侧本位币应为 390: %s", netting.BaseCurrencyAmount)
		}
		if !netting.PayableBaseAmount.Equal(decimal.RequireFromString("420")) {
			t.Fatalf("应付侧本位币应为 420: %s", netting.PayableBaseAmount)
		}
		if !netting.ExchangeGainLoss.Equal(decimal.RequireFromString("30")) {
			t.Fatalf("对冲汇差应为 30: %s", netting.ExchangeGainLoss)
		}
		// Create 成功后经 GetByKey 普通上下文重读，两字段已通过持久化链路往返。
		persisted, err := NewFinanceNettingRepo(data).GetByKey(ctx, fixture.organizationID, "netting-base-amounts-"+fixture.suffix)
		if err != nil || persisted == nil {
			t.Fatalf("重读对冲单失败: %v", err)
		}
		if !persisted.PayableBaseAmount.Equal(netting.PayableBaseAmount) || !persisted.ExchangeGainLoss.Equal(netting.ExchangeGainLoss) {
			t.Fatalf("持久化往返后应付侧本位币/汇差不一致: payable=%s gainLoss=%s", persisted.PayableBaseAmount, persisted.ExchangeGainLoss)
		}
	})
}
