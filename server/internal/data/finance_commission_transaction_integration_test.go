package data

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

func TestCommissionBillLockOrderConcurrentPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}

	data, cleanup, err := newIntegrationData(source)
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	// 先注册关库，利用 Cleanup 的 LIFO 顺序保证夹具删除先于连接关闭。
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	org, err := data.db.Organization.Create().
		SetCode("COMM-ORG-" + suffix).
		SetName("提成锁顺序测试组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if _, cleanupErr := data.db.FinanceBill.Delete().Where(financebillent.OrganizationIDEQ(org.ID)).Exec(cleanupCtx); cleanupErr != nil {
			t.Errorf("清理测试账单: %v", cleanupErr)
		}
		if _, cleanupErr := data.db.Partner.Delete().Where(partnerent.OrganizationIDEQ(org.ID)).Exec(cleanupCtx); cleanupErr != nil {
			t.Errorf("清理测试往来单位: %v", cleanupErr)
		}
		if cleanupErr := data.db.Organization.DeleteOneID(org.ID).Exec(cleanupCtx); cleanupErr != nil {
			t.Errorf("清理测试组织: %v", cleanupErr)
		}
	})

	partner, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("PARTNER-" + suffix).
		SetLegalName("提成锁测试单位-" + suffix).
		SetNormalizedName("提成锁测试单位-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}

	billA, err := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("BILL-A-" + suffix).
		SetIdempotencyKey("bill-a-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(partner.ID).
		SetSettlementPartyName(partner.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate("2026-08-30").
		SetTotalAmount("1000.00000000").
		SetNetAmount("1000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("1000.00000000").
		SetFeeCount(1).
		SetBillDate("2026-08-30").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建账单 A 失败: %v", err)
	}

	billB, err := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("BILL-B-" + suffix).
		SetIdempotencyKey("bill-b-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(partner.ID).
		SetSettlementPartyName(partner.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceBASE_CURRENCY).
		SetExchangeRateDate("2026-08-30").
		SetTotalAmount("2000.00000000").
		SetNetAmount("2000.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("2000.00000000").
		SetFeeCount(1).
		SetBillDate("2026-08-30").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建账单 B 失败: %v", err)
	}

	var (
		wg               sync.WaitGroup
		transactionReady = make(chan struct{}, 2)
		startQueries     = make(chan struct{})
		err1, err2       error
		bills1, bills2   []*ent.FinanceBill
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		txCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err1 = data.WithTx(txCtx, func(tx *ent.Tx) error {
			transactionReady <- struct{}{}
			select {
			case <-startQueries:
			case <-txCtx.Done():
				return txCtx.Err()
			}
			var qErr error
			bills1, qErr = commissionCalculationBillsQuery(commissionStoreFromTx(tx), org.ID, []uuid.UUID{billA.ID, billB.ID}, true).All(txCtx)
			if qErr != nil {
				return qErr
			}
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}()

	go func() {
		defer wg.Done()
		txCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err2 = data.WithTx(txCtx, func(tx *ent.Tx) error {
			transactionReady <- struct{}{}
			select {
			case <-startQueries:
			case <-txCtx.Done():
				return txCtx.Err()
			}
			var qErr error
			bills2, qErr = commissionCalculationBillsQuery(commissionStoreFromTx(tx), org.ID, []uuid.UUID{billB.ID, billA.ID}, true).All(txCtx)
			if qErr != nil {
				return qErr
			}
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}()

	readyTimer := time.NewTimer(5 * time.Second)
	defer readyTimer.Stop()
	for readyCount := 0; readyCount < 2; readyCount++ {
		select {
		case <-transactionReady:
		case <-readyTimer.C:
			close(startQueries)
			wg.Wait()
			t.Fatal("等待两个 PostgreSQL 事务进入查询栅栏超时")
		}
	}
	close(startQueries)
	wg.Wait()

	if err1 != nil {
		t.Fatalf("事务 1 执行失败 (input: [A, B]): %v", err1)
	}
	if err2 != nil {
		t.Fatalf("事务 2 执行失败 (input: [B, A]): %v", err2)
	}
	if len(bills1) != 2 || len(bills2) != 2 {
		t.Fatalf("事务返回账单数量不符合预期: len(bills1)=%d, len(bills2)=%d", len(bills1), len(bills2))
	}
	if bills1[0].ID.String() >= bills1[1].ID.String() {
		t.Fatalf("事务 1 返回账单未按主键升序排序: %s >= %s", bills1[0].ID, bills1[1].ID)
	}
	if bills2[0].ID.String() >= bills2[1].ID.String() {
		t.Fatalf("事务 2 返回账单未按主键升序排序: %s >= %s", bills2[0].ID, bills2[1].ID)
	}
}
