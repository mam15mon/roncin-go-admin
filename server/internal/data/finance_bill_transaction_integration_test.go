package data

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	exchangeratesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
)

const financeBillIntegrationDate = "2026-08-30"

type financeBillPostgresFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	partnerID      uuid.UUID
	accountID      uuid.UUID
	usdAccountID   uuid.UUID
	orderID        uuid.UUID
	actorID        uuid.UUID
	suffix         string
}

type financeBillCreateResult struct {
	bill *biz.FinanceBill
	err  error
}

type invalidAuditResultFinanceBillRepo struct {
	biz.FinanceBillRepo
}

type pausingExchangeRateRepo struct {
	biz.ExchangeRateRepo
	resolved    chan struct{}
	release     chan struct{}
	resolvedOne sync.Once
	releaseOne  sync.Once
}

func (r *invalidAuditResultFinanceBillRepo) Create(ctx context.Context, bill *biz.FinanceBill, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	audit.Result = "invalid"
	return r.FinanceBillRepo.Create(ctx, bill, audit)
}

func (r *pausingExchangeRateRepo) ResolveRate(ctx context.Context, organizationID uuid.UUID, fromCurrency, toCurrency, pivotCurrency, rateDate string) (biz.ResolvedRate, error) {
	rate, err := r.ExchangeRateRepo.ResolveRate(ctx, organizationID, fromCurrency, toCurrency, pivotCurrency, rateDate)
	if err != nil {
		return biz.ResolvedRate{}, err
	}
	r.resolvedOne.Do(func() {
		close(r.resolved)
		<-r.release
	})
	return rate, nil
}

func (r *pausingExchangeRateRepo) continueResolve() {
	r.releaseOne.Do(func() { close(r.release) })
}

func TestFinanceBillCreateSharedTransactionPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	hasCNY, err := data.db.Currency.Query().Where(currencyent.CodeEQ("CNY"), currencyent.EnabledEQ(true)).Exist(context.Background())
	if err != nil {
		t.Fatalf("查询生产结算账户用例所需币种: %v", err)
	}
	if !hasCNY {
		if _, err := data.db.Currency.Create().SetCode("CNY").SetName("人民币").SetEnabled(true).Save(context.Background()); err != nil {
			t.Fatalf("创建生产结算账户用例所需币种: %v", err)
		}
	}

	t.Run("相同幂等键并发创建返回同一账单", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("same-key")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		input := biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-same-key-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		}

		results := createFinanceBillsConcurrently(usecase, fixture.organizationID, fixture.actorID, input, input)
		for index, result := range results {
			if result.err != nil {
				t.Fatalf("第 %d 个并发请求创建账单: %v", index+1, result.err)
			}
			if result.bill == nil {
				t.Fatalf("第 %d 个并发请求未返回账单", index+1)
			}
		}
		if results[0].bill.ID != results[1].bill.ID || results[0].bill.BillNo != results[1].bill.BillNo {
			t.Fatalf("相同幂等键返回了不同账单: first=%s/%s second=%s/%s", results[0].bill.ID, results[0].bill.BillNo, results[1].bill.ID, results[1].bill.BillNo)
		}
		fixture.requireCommittedState(feeID, 1)
	})

	t.Run("不同幂等键并发使用同一费用只有一个成功", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("different-key")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		first := biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-first-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		}
		second := first
		second.IdempotencyKey = "bill-second-" + fixture.suffix

		results := createFinanceBillsConcurrently(usecase, fixture.organizationID, fixture.actorID, first, second)
		successes := 0
		feeConflicts := 0
		for _, result := range results {
			switch {
			case result.err == nil && result.bill != nil:
				successes++
			case errors.Is(result.err, biz.ErrFinanceBillFeeInvalid):
				feeConflicts++
			default:
				t.Fatalf("并发创建返回非预期结果: bill=%#v error=%v", result.bill, result.err)
			}
		}
		if successes != 1 || feeConflicts != 1 {
			t.Fatalf("并发创建结果 success=%d feeConflict=%d，期望各 1", successes, feeConflicts)
		}
		fixture.requireCommittedState(feeID, 1)
	})

	t.Run("审计失败回滚账单费用状态和单号序列", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("rollback")
		repo := &invalidAuditResultFinanceBillRepo{FinanceBillRepo: NewFinanceBillRepo(data)}
		usecase := fixture.newUsecase(repo)

		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-rollback-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err == nil {
			t.Fatal("审计结果非法时创建账单未失败")
		}
		fixture.requireRolledBackState(feeID)
	})

	t.Run("并发修改汇率不改变事务内账单快照", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		settingID := fixture.createExchangeRateSetting("7.20000000")
		feeID := fixture.createConfirmedFeeWithCurrency("rate-snapshot", "USD", "7.20000000", "720.00000000", orderfeeent.ExchangeRateSourceSYSTEM)
		exchangeRepo := &pausingExchangeRateRepo{
			ExchangeRateRepo: NewExchangeRateRepo(data), resolved: make(chan struct{}), release: make(chan struct{}),
		}
		defer exchangeRepo.continueResolve()
		usecase := biz.NewFinanceBillUsecase(NewFinanceBillRepo(data), biz.NewExchangeRateUsecase(exchangeRepo), data)
		billResult := make(chan financeBillCreateResult, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			bill, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
				FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-rate-snapshot-" + fixture.suffix, SettlementAccountID: fixture.usdAccountID,
			})
			billResult <- financeBillCreateResult{bill: bill, err: err}
		}()
		select {
		case <-exchangeRepo.resolved:
		case <-time.After(5 * time.Second):
			t.Fatal("账单事务未完成汇率解析")
		}

		updateResult := make(chan error, 1)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := data.db.ExchangeRateSetting.UpdateOneID(settingID).SetRate("7.30000000").Save(ctx)
			updateResult <- err
		}()
		select {
		case err := <-updateResult:
			exchangeRepo.continueResolve()
			t.Fatalf("账单事务提交前汇率更新未等待共享锁: %v", err)
		case <-time.After(150 * time.Millisecond):
		}
		exchangeRepo.continueResolve()

		var created financeBillCreateResult
		select {
		case created = <-billResult:
		case <-time.After(10 * time.Second):
			t.Fatal("等待账单事务提交超时")
		}
		if created.err != nil || created.bill == nil {
			t.Fatalf("创建汇率快照账单: bill=%#v error=%v", created.bill, created.err)
		}
		if created.bill.ExchangeRate.StringFixed(8) != "7.20000000" || created.bill.BaseCurrencyAmount.StringFixed(8) != "720.00000000" {
			t.Fatalf("账单未保存事务内汇率快照: %#v", created.bill)
		}
		select {
		case err := <-updateResult:
			if err != nil {
				t.Fatalf("账单提交后更新汇率: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("等待汇率更新超时")
		}
		setting, err := data.db.ExchangeRateSetting.Get(context.Background(), settingID)
		if err != nil || setting.Rate != "7.30000000" {
			t.Fatalf("并发汇率最终值 = %#v，期望 7.30000000，error=%v", setting, err)
		}
		fixture.requireCommittedState(feeID, 1)
	})

	t.Run("候选后账户停用会在创建事务内被拒绝", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("account-disabled")
		candidates, err := data.db.PartnerAccount.Query().
			Where(partneraccountent.IDEQ(fixture.accountID), partneraccountent.EnabledEQ(true)).
			All(context.Background())
		if err != nil || len(candidates) != 1 {
			t.Fatalf("创建前账户候选不正确: accounts=%#v error=%v", candidates, err)
		}
		if _, err = data.db.PartnerAccount.UpdateOneID(fixture.accountID).SetIsDefaultReceivable(false).SetIsDefaultPayable(false).SetEnabled(false).Save(context.Background()); err != nil {
			t.Fatalf("停用候选账户失败: %v", err)
		}
		_, err = fixture.newUsecase(NewFinanceBillRepo(data)).Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-account-disabled-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if !errors.Is(err, biz.ErrFinanceBillSettlementAccountInvalid) {
			t.Fatalf("停用账户创建错误 = %v，期望 %v", err, biz.ErrFinanceBillSettlementAccountInvalid)
		}
		fixture.requireRolledBackState(feeID)
	})

	t.Run("不存在的账户在创建事务内返回领域错误", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("missing-account")
		_, err := fixture.newUsecase(NewFinanceBillRepo(data)).Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-missing-account-" + fixture.suffix, SettlementAccountID: uuid.New(),
		})
		if !errors.Is(err, biz.ErrFinanceBillSettlementAccountInvalid) {
			t.Fatalf("不存在账户创建错误 = %v，期望 %v", err, biz.ErrFinanceBillSettlementAccountInvalid)
		}
		fixture.requireRolledBackState(feeID)
	})

	t.Run("错误结算单位账户在创建事务内被拒绝", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("account-wrong-party")
		otherPartner, err := data.db.Partner.Create().
			SetOrganizationID(fixture.organizationID).
			SetCode("OTHER-" + fixture.suffix).
			SetLegalName("错误结算单位-" + fixture.suffix).
			SetNormalizedName("错误结算单位-" + fixture.suffix).
			Save(context.Background())
		if err != nil {
			t.Fatalf("创建错误结算单位: %v", err)
		}
		account, err := data.db.PartnerAccount.Create().
			SetPartnerID(otherPartner.ID).
			SetName("错误结算单位账户").
			SetAccountHolder("错误结算单位户名").
			SetBankName("集成测试银行").
			SetAccountNo("OTHER-" + fixture.suffix).
			SetCurrency("CNY").
			SetUsage(partneraccountent.UsageBOTH).
			SetEnabled(true).
			Save(context.Background())
		if err != nil {
			t.Fatalf("创建错误结算单位账户: %v", err)
		}
		_, err = fixture.newUsecase(NewFinanceBillRepo(data)).Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-account-wrong-party-" + fixture.suffix, SettlementAccountID: account.ID,
		})
		if !errors.Is(err, biz.ErrFinanceBillSettlementAccountInvalid) {
			t.Fatalf("错误结算单位账户创建错误 = %v，期望 %v", err, biz.ErrFinanceBillSettlementAccountInvalid)
		}
		fixture.requireRolledBackState(feeID)
		if err := data.db.PartnerAccount.DeleteOneID(account.ID).Exec(context.Background()); err != nil {
			t.Fatalf("清理错误结算单位账户: %v", err)
		}
		if err := data.db.Partner.DeleteOneID(otherPartner.ID).Exec(context.Background()); err != nil {
			t.Fatalf("清理错误结算单位: %v", err)
		}
	})

	t.Run("错误币种账户在创建事务内被拒绝", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("account-wrong-currency")
		_, err := fixture.newUsecase(NewFinanceBillRepo(data)).Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-account-wrong-currency-" + fixture.suffix, SettlementAccountID: fixture.usdAccountID,
		})
		if !errors.Is(err, biz.ErrFinanceBillSettlementAccountInvalid) {
			t.Fatalf("错误币种账户创建错误 = %v，期望 %v", err, biz.ErrFinanceBillSettlementAccountInvalid)
		}
		fixture.requireRolledBackState(feeID)
	})

	t.Run("错误用途账户在创建事务内被拒绝", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("account-wrong-usage")
		account, err := data.db.PartnerAccount.Create().
			SetPartnerID(fixture.partnerID).
			SetName("应付用途账户").
			SetAccountHolder("应付用途户名").
			SetBankName("集成测试银行").
			SetAccountNo("PAYABLE-" + fixture.suffix).
			SetCurrency("CNY").
			SetUsage(partneraccountent.UsagePAYABLE).
			SetEnabled(true).
			Save(context.Background())
		if err != nil {
			t.Fatalf("创建错误用途账户: %v", err)
		}
		_, err = fixture.newUsecase(NewFinanceBillRepo(data)).Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-account-wrong-usage-" + fixture.suffix, SettlementAccountID: account.ID,
		})
		if !errors.Is(err, biz.ErrFinanceBillSettlementAccountInvalid) {
			t.Fatalf("错误用途账户创建错误 = %v，期望 %v", err, biz.ErrFinanceBillSettlementAccountInvalid)
		}
		fixture.requireRolledBackState(feeID)
	})

	t.Run("草稿换账户由事务重读并生成新快照", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createConfirmedFee("account-update")
		usecase := fixture.newUsecase(NewFinanceBillRepo(data))
		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-account-update-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("创建草稿账单失败: %v", err)
		}
		replacement, err := data.db.PartnerAccount.Create().SetPartnerID(fixture.partnerID).SetName("更新后真实账户").SetAccountHolder("更新后真实户名").SetBankName("更新后真实银行").SetAccountNo("UPDATED-ACCOUNT").SetCurrency("CNY").SetUsage(partneraccountent.UsageRECEIVABLE).SetEnabled(true).Save(context.Background())
		if err != nil {
			t.Fatalf("创建替换账户失败: %v", err)
		}
		updated, err := usecase.Update(context.Background(), []uuid.UUID{fixture.organizationID}, fixture.actorID, biz.UpdateFinanceBillInput{
			ID: created.ID, BillDate: financeBillIntegrationDate, ExpectedVersion: created.Version, SettlementAccountID: replacement.ID,
		})
		if err != nil {
			t.Fatalf("草稿改选账户失败: %v", err)
		}
		if updated.SettlementAccountID != replacement.ID || updated.SettlementAccountName != replacement.Name || updated.SettlementAccountHolder != replacement.AccountHolder || updated.SettlementBankName != replacement.BankName || updated.SettlementBankAccount != replacement.AccountNo || updated.SettlementAccountCurrency != "CNY" {
			t.Fatalf("草稿账户快照未使用事务重读值: %#v", updated)
		}
	})

	t.Run("双默认并发最终各自唯一", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		if _, err := data.db.PartnerAccount.UpdateOneID(fixture.accountID).SetIsDefaultReceivable(false).SetIsDefaultPayable(false).Save(context.Background()); err != nil {
			t.Fatalf("清理初始默认账户失败: %v", err)
		}
		start := make(chan struct{})
		results := make(chan error, 2)
		for index := range 2 {
			index := index
			go func() {
				<-start
				_, err := data.db.PartnerAccount.Create().SetPartnerID(fixture.partnerID).SetName("并发默认账户").SetAccountHolder("测试户名").SetBankName("测试银行").SetAccountNo("DEFAULT-" + fixture.suffix + string(rune('A'+index))).SetCurrency("CNY").SetUsage(partneraccountent.UsageBOTH).SetEnabled(true).SetIsDefaultReceivable(true).SetIsDefaultPayable(true).Save(context.Background())
				results <- err
			}()
		}
		close(start)
		successes := 0
		for range 2 {
			if err := <-results; err == nil {
				successes++
			}
		}
		if successes != 1 {
			t.Fatalf("并发设置双默认成功数 = %d，期望 1", successes)
		}
		receivable, err := data.db.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(fixture.partnerID), partneraccountent.CurrencyEQ("CNY"), partneraccountent.IsDefaultReceivableEQ(true)).Count(context.Background())
		if err != nil || receivable != 1 {
			t.Fatalf("应收默认账户数 = %d，期望 1，error=%v", receivable, err)
		}
		payable, err := data.db.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(fixture.partnerID), partneraccountent.CurrencyEQ("CNY"), partneraccountent.IsDefaultPayableEQ(true)).Count(context.Background())
		if err != nil || payable != 1 {
			t.Fatalf("应付默认账户数 = %d，期望 1，error=%v", payable, err)
		}
	})

	t.Run("生产账户用例并发切换双默认无死锁且最终唯一", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		if _, err := data.db.PartnerAccount.UpdateOneID(fixture.accountID).SetIsDefaultReceivable(false).SetIsDefaultPayable(false).Save(context.Background()); err != nil {
			t.Fatalf("清理初始默认账户失败: %v", err)
		}
		accountIDs := make([]uuid.UUID, 0, 2)
		for index := range 2 {
			account, err := data.db.PartnerAccount.Create().
				SetPartnerID(fixture.partnerID).
				SetName("生产并发切换账户").
				SetAccountHolder("生产并发切换户名").
				SetBankName("集成测试银行").
				SetAccountNo("USECASE-" + fixture.suffix + string(rune('A'+index))).
				SetCurrency("CNY").
				SetUsage(partneraccountent.UsageBOTH).
				SetEnabled(true).
				Save(context.Background())
			if err != nil {
				t.Fatalf("创建并发切换账户: %v", err)
			}
			accountIDs = append(accountIDs, account.ID)
		}

		usecase := biz.NewPartnerAccountUsecase(NewPartnerAccountRepo(data))
		start := make(chan struct{})
		results := make(chan error, len(accountIDs))
		for index, accountID := range accountIDs {
			index, accountID := index, accountID
			go func() {
				<-start
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_, err := usecase.Update(ctx, fixture.organizationID, fixture.actorID, fixture.partnerID, accountID, &biz.PartnerAccount{
					Name:                "生产并发切换账户",
					AccountHolder:       "生产并发切换户名",
					BankName:            "集成测试银行",
					AccountNo:           "USECASE-" + fixture.suffix + string(rune('A'+index)),
					Currency:            "CNY",
					Usage:               biz.PartnerAccountUsageBoth,
					IsDefaultReceivable: true,
					IsDefaultPayable:    true,
					Enabled:             true,
				})
				results <- err
			}()
		}
		close(start)

		successes := 0
		for range accountIDs {
			err := <-results
			switch {
			case err == nil:
				successes++
			case errors.Is(err, biz.ErrPartnerAccountDefaultConflict):
				// 数据库唯一索引作为最终兜底时，此结果仍符合默认账户语义；
				// 其他错误（尤其死锁或超时）均应直接失败。
			default:
				t.Fatalf("生产账户用例并发切换返回非预期错误: %v", err)
			}
		}
		if successes == 0 {
			t.Fatal("生产账户用例并发切换没有任何成功结果")
		}
		receivable, err := data.db.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(fixture.partnerID), partneraccountent.CurrencyEQ("CNY"), partneraccountent.IsDefaultReceivableEQ(true)).Count(context.Background())
		if err != nil || receivable != 1 {
			t.Fatalf("生产账户用例并发切换后应收默认账户数 = %d，期望 1，error=%v", receivable, err)
		}
		payable, err := data.db.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(fixture.partnerID), partneraccountent.CurrencyEQ("CNY"), partneraccountent.IsDefaultPayableEQ(true)).Count(context.Background())
		if err != nil || payable != 1 {
			t.Fatalf("生产账户用例并发切换后应付默认账户数 = %d，期望 1，error=%v", payable, err)
		}
	})
}

func newFinanceBillPostgresFixture(t *testing.T, data *Data) *financeBillPostgresFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("BILL-TX-" + suffix).
		SetName("账单事务集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	fixture := &financeBillPostgresFixture{
		t: t, data: data, organizationID: organization.ID, actorID: uuid.New(), suffix: suffix,
	}
	t.Cleanup(fixture.cleanup)

	partner, err := data.db.Partner.Create().
		SetOrganizationID(organization.ID).
		SetCode("CUSTOMER-" + suffix).
		SetLegalName("账单事务测试客户-" + suffix).
		SetNormalizedName("账单事务测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}
	fixture.partnerID = partner.ID
	account, err := data.db.PartnerAccount.Create().
		SetPartnerID(partner.ID).
		SetName("账单事务测试结算账户-" + suffix).
		SetAccountHolder(partner.LegalName).
		SetBankName("集成测试银行").
		SetAccountNo("BILL-" + suffix).
		SetCurrency("CNY").
		SetUsage(partneraccountent.UsageBOTH).
		SetEnabled(true).
		SetIsDefaultReceivable(true).
		SetIsDefaultPayable(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试结算账户: %v", err)
	}
	fixture.accountID = account.ID
	usdAccount, err := data.db.PartnerAccount.Create().
		SetPartnerID(partner.ID).
		SetName("账单事务美元结算账户-" + suffix).
		SetAccountHolder(partner.LegalName).
		SetBankName("集成测试银行").
		SetAccountNo("USD-" + suffix).
		SetCurrency("USD").
		SetUsage(partneraccountent.UsageBOTH).
		SetEnabled(true).
		SetIsDefaultReceivable(true).
		SetIsDefaultPayable(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试美元结算账户: %v", err)
	}
	fixture.usdAccountID = usdAccount.ID

	order, err := data.db.Order.Create().
		SetOrganizationID(organization.ID).
		SetOrderNo("SE" + suffix).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单: %v", err)
	}
	fixture.orderID = order.ID

	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(organization.ID).
		SetDocumentType(numberruleent.DocumentTypeBill).
		SetPrefix("BILL-").
		SetDateFormat(numberruleent.DateFormatNone).
		SetSequenceLength(4).
		SetResetPolicy(numberruleent.ResetPolicyNever).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试账单编号规则: %v", err)
	}
	return fixture
}

func (f *financeBillPostgresFixture) createConfirmedFee(key string) uuid.UUID {
	return f.createConfirmedFeeWithCurrency(key, "CNY", "1.00000000", "100.00000000", orderfeeent.ExchangeRateSourceBASE_CURRENCY)
}

func (f *financeBillPostgresFixture) createConfirmedFeeWithCurrency(key, currency, rate, baseAmount string, source orderfeeent.ExchangeRateSource) uuid.UUID {
	f.t.Helper()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("fee-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusCONFIRMED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency(currency).
		SetExchangeRate(rate).
		SetExchangeRateSource(source).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(baseAmount).
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试费用: %v", err)
	}
	return fee.ID
}

func (f *financeBillPostgresFixture) createExchangeRateSetting(rate string) uuid.UUID {
	f.t.Helper()
	setting, err := f.data.db.ExchangeRateSetting.Create().
		SetOrganizationID(f.organizationID).
		SetFromCurrency("USD").
		SetToCurrency("CNY").
		SetEffectiveFrom(time.Date(2026, 8, 1, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).
		SetRate(rate).
		SetIsActive(true).
		Save(context.Background())
	if err != nil {
		f.t.Fatalf("创建测试汇率: %v", err)
	}
	return setting.ID
}

func (f *financeBillPostgresFixture) newUsecase(repo biz.FinanceBillRepo) *biz.FinanceBillUsecase {
	return biz.NewFinanceBillUsecase(repo, biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data)), f.data)
}

func createFinanceBillsConcurrently(usecase *biz.FinanceBillUsecase, organizationID, actorID uuid.UUID, inputs ...biz.CreateFinanceBillInput) []financeBillCreateResult {
	start := make(chan struct{})
	results := make(chan financeBillCreateResult, len(inputs))
	for _, input := range inputs {
		input := input
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			bill, err := usecase.Create(ctx, organizationID, actorID, input)
			results <- financeBillCreateResult{bill: bill, err: err}
		}()
	}
	close(start)
	collected := make([]financeBillCreateResult, 0, len(inputs))
	for range inputs {
		collected = append(collected, <-results)
	}
	return collected
}

func (f *financeBillPostgresFixture) requireCommittedState(feeID uuid.UUID, wantBills int) {
	f.t.Helper()
	ctx := context.Background()
	billCount, err := f.data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || billCount != wantBills {
		f.t.Fatalf("已提交账单数 = %d，期望 %d，error=%v", billCount, wantBills, err)
	}
	lineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lineCount != wantBills {
		f.t.Fatalf("已提交账单明细数 = %d，期望 %d，error=%v", lineCount, wantBills, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.bill.create")).Count(ctx)
	if err != nil || auditCount != wantBills {
		f.t.Fatalf("已提交账单审计数 = %d，期望 %d，error=%v", auditCount, wantBills, err)
	}
	sequences, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).All(ctx)
	if err != nil || len(sequences) != 1 || sequences[0].CurrentValue != int64(wantBills) {
		f.t.Fatalf("已提交账单序列 = %#v，期望当前值 %d，error=%v", sequences, wantBills, err)
	}
	fee, err := f.data.db.OrderFee.Get(ctx, feeID)
	if err != nil || fee.Status != orderfeeent.StatusBILLED || fee.Version != 2 {
		f.t.Fatalf("已提交费用状态 = %#v，期望 BILLED/version 2，error=%v", fee, err)
	}
}

func (f *financeBillPostgresFixture) requireRolledBackState(feeID uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	billCount, err := f.data.db.FinanceBill.Query().Where(financebillent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || billCount != 0 {
		f.t.Fatalf("回滚后账单数 = %d，期望 0，error=%v", billCount, err)
	}
	lineCount, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lineCount != 0 {
		f.t.Fatalf("回滚后账单明细数 = %d，期望 0，error=%v", lineCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后审计数 = %d，期望 0，error=%v", auditCount, err)
	}
	sequenceCount, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || sequenceCount != 0 {
		f.t.Fatalf("回滚后账单序列数 = %d，期望 0，error=%v", sequenceCount, err)
	}
	fee, err := f.data.db.OrderFee.Get(ctx, feeID)
	if err != nil || fee.Status != orderfeeent.StatusCONFIRMED || fee.Version != 1 {
		f.t.Fatalf("回滚后费用状态 = %#v，期望 CONFIRMED/version 1，error=%v", fee, err)
	}
}

func (f *financeBillPostgresFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "账单审计", run: func() error {
			_, err := f.data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "账单明细", run: func() error {
			_, err := f.data.db.FinanceBillLine.Delete().Where(financebilllineent.HasBillWith(financebillent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "账单", run: func() error {
			_, err := f.data.db.FinanceBill.Delete().Where(financebillent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "编号序列", run: func() error {
			_, err := f.data.db.NumberSequence.Delete().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "编号规则", run: func() error {
			_, err := f.data.db.NumberRule.Delete().Where(numberruleent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单费用", run: func() error {
			_, err := f.data.db.OrderFee.Delete().Where(orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单", run: func() error {
			_, err := f.data.db.Order.Delete().Where(orderent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "结算账户", run: func() error {
			_, err := f.data.db.PartnerAccount.Delete().Where(partneraccountent.PartnerID(f.partnerID)).Exec(ctx)
			return err
		}},
		{name: "往来单位", run: func() error {
			_, err := f.data.db.Partner.Delete().Where(partnerent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "汇率设置", run: func() error {
			_, err := f.data.db.ExchangeRateSetting.Delete().Where(exchangeratesettingent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "组织", run: func() error { return f.data.db.Organization.DeleteOneID(f.organizationID).Exec(ctx) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理测试%s: %v", step.name, err)
		}
	}
}

var _ biz.FinanceBillRepo = (*invalidAuditResultFinanceBillRepo)(nil)
var _ biz.ExchangeRateRepo = (*pausingExchangeRateRepo)(nil)
