package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/shopspring/decimal"
)

// 在真实数据库锁已持有时控制另一入口进入预锁阶段，避免靠定时等待碰运气制造竞争。
type compositionLockOrderVerificationRepo struct {
	biz.VerificationRepo
	beforeOrders    func()
	beforeCashflows func()
	afterCashflows  func(context.Context) error
}

func (r *compositionLockOrderVerificationRepo) LockBillOrders(ctx context.Context, org uuid.UUID, ids []uuid.UUID) error {
	if r.beforeOrders != nil {
		r.beforeOrders()
	}
	return r.VerificationRepo.LockBillOrders(ctx, org, ids)
}
func (r *compositionLockOrderVerificationRepo) LockCashflows(ctx context.Context, org uuid.UUID, ids []uuid.UUID) error {
	if r.beforeCashflows != nil {
		r.beforeCashflows()
	}
	if err := r.VerificationRepo.LockCashflows(ctx, org, ids); err != nil {
		return err
	}
	if r.afterCashflows != nil {
		return r.afterCashflows(ctx)
	}
	return nil
}

func TestFinanceBillCompositionAndShortfallLockOrderPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ensureFinanceBillBatchCurrencies(t, data, "CNY")
	for _, sameOrder := range []bool{true, false} {
		name := "不同订单共享资金"
		if sameOrder {
			name = "同订单不同账单"
		}
		t.Run(name, func(t *testing.T) {
			f, operations, profile := compositionFixture(t, data)
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			oldFee := f.createBatchUnbilledFee("old-bill")
			if !sameOrder {
				order, err := data.db.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.organizationID).SetOrderNo("OTHER-" + f.suffix).SetCustomerID(f.partnerID).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).Save(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if err := data.db.OrderFee.UpdateOneID(oldFee).SetOrderID(order.ID).Exec(ctx); err != nil {
					t.Fatal(err)
				}
			}
			billUC := f.newUsecase(NewFinanceBillRepo(data))
			oldBill, err := billUC.Create(ctx, f.organizationID, f.actorUserID, biz.CreateFinanceBillInput{FeeIDs: []uuid.UUID{oldFee}, BillDate: financeBillIntegrationDate, SettlementAccountID: f.accountID, IdempotencyKey: "old-bill-" + f.suffix})
			if err != nil {
				t.Fatal(err)
			}
			newFee := f.createBatchUnbilledFee("new-bill")
			addTestBillInvoice(t, data, data.db.FinanceBill.GetX(ctx, oldBill.ID), "100", f.actorUserID)
			input := f.buildBatchInput(t, billUC, "parallel-batch-"+f.suffix, []uuid.UUID{newFee}, biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}, nil)
			input.Groups[0].Invoices = []biz.FinanceBillBatchInvoiceInput{compositionInvoice(profile, newFee, "LOCK-ORDER-"+f.suffix, 100)}
			ledger := &feeLedgerPostgresFixture{financeBillPostgresFixture: f.financeBillPostgresFixture}
			cash := ledger.createConfirmedCashflow("shared", "100", financeBillIntegrationDate)
			input.Groups[0].Verifications = []biz.FinanceBillBatchVerificationInput{compositionShortfall(30)}
			input.Groups[0].Verifications[0].Allocations = append(input.Groups[0].Verifications[0].Allocations, biz.FinanceBillBatchVerificationAllocation{CashflowID: cash.ID, Amount: decimal.NewFromInt(20)})
			held, contender := make(chan struct{}), make(chan struct{})
			var heldOnce, contenderOnce sync.Once
			baseRepo := NewVerificationRepo(data)
			fRepo := &compositionLockOrderVerificationRepo{VerificationRepo: baseRepo, afterCashflows: func(ctx context.Context) error {
				var waitErr error
				heldOnce.Do(func() {
					close(held)
					select {
					case <-contender:
					case <-ctx.Done():
						waitErr = ctx.Err()
					}
				})
				return waitErr
			}}
			exchange := biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), nil)
			operations.Verifications = biz.NewVerificationUsecase(fRepo, exchange, data, nil)
			standaloneRepo := &compositionLockOrderVerificationRepo{VerificationRepo: baseRepo}
			signal := func() { contenderOnce.Do(func() { close(contender) }) }
			if sameOrder {
				standaloneRepo.beforeOrders = signal
			} else {
				standaloneRepo.beforeCashflows = signal
			}
			standalone := biz.NewVerificationUsecase(standaloneRepo, exchange, data, nil)
			batchDone := make(chan financeBillBatchCreateResult, 1)
			go func() {
				r, e := billUC.CreateBatchWithOperations(ctx, f.organizationID, f.actorUserID, input, operations)
				batchDone <- financeBillBatchCreateResult{result: r, err: e}
			}()
			select {
			case <-held:
			case <-ctx.Done():
				t.Fatal("F未取得既有资金锁")
			}
			standaloneDone := make(chan error, 1)
			go func() {
				_, e := standalone.CreateWithCashflow(ctx, f.organizationID, f.actorUserID, biz.CreateVerificationInput{IdempotencyKey: "parallel-shortfall-" + f.suffix, VerificationDate: financeBillIntegrationDate, Allocations: []*biz.VerificationAllocation{{BillID: oldBill.ID, CashflowID: cash.ID, Amount: decimal.NewFromInt(20)}, {BillID: oldBill.ID, Amount: decimal.NewFromInt(30)}}}, biz.CreateFinanceCashflowInput{Direction: biz.OrderFeeReceivable, SettlementPartyID: f.partnerID, Currency: "CNY", Amount: decimal.NewFromInt(30), TransactionDate: financeBillIntegrationDate, OurAccount: "独立差额账户", PaymentMethod: "BANK_TRANSFER"}, operations.Cashflows, false)
				standaloneDone <- e
			}()
			batch := <-batchDone
			if batch.err != nil || batch.result == nil || batch.result.Groups[0].Err != nil {
				t.Fatalf("F与差额核销发生锁序冲突: %+v", batch)
			}
			if err := <-standaloneDone; err != nil {
				t.Fatalf("独立差额核销发生锁序冲突: %v", err)
			}
			if !batch.result.Groups[0].Bill.VerifiedAmount.Equal(decimal.NewFromInt(50)) {
				t.Fatal("F核销金额错误")
			}
			fresh, err := billUC.Get(ctx, []uuid.UUID{f.organizationID}, oldBill.ID)
			if err != nil || !fresh.VerifiedAmount.Equal(decimal.NewFromInt(50)) {
				t.Fatalf("独立核销金额错误: %v", err)
			}
			if count, err := data.db.FinanceVerification.Query().Where(financeverification.OrganizationIDEQ(f.organizationID)).Count(ctx); err != nil || count != 2 {
				t.Fatalf("核销事实数量错误: %d %v", count, err)
			}
			if count, err := data.db.FinanceCashflow.Query().Where(financecashflow.OrganizationIDEQ(f.organizationID)).Count(ctx); err != nil || count != 3 {
				t.Fatalf("资金事实数量错误: %d %v", count, err)
			}
		})
	}
}
