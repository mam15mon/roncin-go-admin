package data

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	numberent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	accountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	"github.com/shopspring/decimal"
)

func prepareBillEconomicCost(t *testing.T, f *commissionPostgresFixture) (*ent.OrderFee, *ent.FinanceBill, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	cost, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(f.orderID), feeent.DirectionEQ(feeent.DirectionPAYABLE)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	payable, err := f.data.db.FinanceBill.Query().Where(billent.OrganizationIDEQ(f.organizationID), billent.DirectionEQ(billent.DirectionPAYABLE)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 费用 ETD 周快照为6，已有账单日快照为7，100USD成本分别对应600/700本币。
	cost, err = f.data.db.OrderFee.UpdateOneID(cost.ID).SetCurrency("USD").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetExchangeRate("6").SetBaseCurrencyAmount("600").SetTaxRate("0").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.data.db.FinanceBill.UpdateOneID(payable.ID).SetCurrency("USD").SetSettlementAccountCurrency("USD").SetTotalAmount("100").SetNetAmount("100").SetExchangeRate("7").SetBaseCurrencyAmount("700").Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.FinanceBillLine.Update().Where(lineent.BillIDEQ(payable.ID)).SetCurrency("USD").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetExchangeRate("7").SetBaseCurrencyAmount("700").Save(ctx); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"CNY", "USD"} {
		exists, err := f.data.db.Currency.Query().Where(currencyent.CodeEQ(code)).Exist(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			if _, err = f.data.db.Currency.Create().SetCode(code).SetName(code).SetEnabled(true).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	account, err := f.data.db.PartnerAccount.Create().SetPartnerID(f.customerID).SetName("USD成本账户").SetAccountHolder("测试").SetBankName("测试银行").SetAccountNo("USD-" + f.suffix).SetCurrency("USD").SetUsage(accountent.UsageBOTH).SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 主夹具未创建账户；在其清理往来单位之前删除新增账户。
	t.Cleanup(func() {
		if err := f.data.db.PartnerAccount.DeleteOneID(account.ID).Exec(context.Background()); err != nil {
			t.Error(err)
		}
	})
	for _, kind := range []numberent.DocumentType{numberent.DocumentTypeBill, numberent.DocumentTypeBillBatch} {
		if _, err = f.data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(kind).SetPrefix(string(kind) + "-").SetDateFormat(numberent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	for _, rate := range []struct {
		day   int
		value string
	}{{24, "7"}, {31, "8"}} {
		if _, err = f.data.db.ExchangeRateSetting.Create().SetOrganizationID(f.organizationID).SetFromCurrency("USD").SetToCurrency("CNY").SetEffectiveFrom(time.Date(2026, 8, rate.day, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).SetRate(rate.value).SetIsActive(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	return cost, payable, account.ID
}

func requireBillEconomicNet(t *testing.T, f *commissionPostgresFixture, parentID uuid.UUID, want string) *ent.FinanceCommission {
	t.Helper()
	parent, err := f.data.db.FinanceCommission.Get(context.Background(), parentID)
	if err != nil {
		t.Fatal(err)
	}
	net := decimal.RequireFromString(parent.CommissionAmount)
	rows, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustmentent.CommissionIDEQ(parentID), adjustmentent.StatusIn(adjustmentent.StatusCONFIRMED, adjustmentent.StatusPAID)).All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		v := decimal.RequireFromString(row.Amount)
		if row.Direction == adjustmentent.DirectionDECREASE {
			v = v.Neg()
		}
		net = net.Add(v)
	}
	if !net.Equal(decimal.RequireFromString(want)) {
		t.Fatalf("经济权益净额=%s，期望%s；调整=%v", net, want, rows)
	}
	return parent
}

func TestFinanceBillEconomicReconcilePostgres(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, paid := range []bool{false, true} {
			name := "单张"
			if batch {
				name = "批量"
			}
			if paid {
				name += "已付"
			} else {
				name += "已确认"
			}
			t.Run(name, func(t *testing.T) {
				f := newCommissionPostgresFixture(t)
				ctx := context.Background()
				cost, payable, accountID := prepareBillEconomicCost(t, f)
				uc := f.newUsecase(NewCommissionRepo(f.data))
				created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("bill-economics"))
				if err != nil {
					t.Fatal(err)
				}
				parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
				if err != nil {
					t.Fatal(err)
				}
				if !parent.CommissionAmount.Equal(decimal.NewFromInt(30)) {
					t.Fatalf("初始提成=%s", parent.CommissionAmount)
				}
				if paid {
					preview, err := uc.PreviewPayment(ctx, f.organizationID, parent.ID, parent.Version)
					if err != nil {
						t.Fatal(err)
					}
					parent, err = uc.MarkPaid(ctx, f.organizationID, f.actorID, parent.ID, parent.Version, biz.CommissionPaymentInput{PreviewToken: preview.PreviewToken, IdempotencyKey: "pay-" + f.suffix, ActualAmount: preview.ActualAmount})
					if err != nil {
						t.Fatal(err)
					}
				}
				manual, err := uc.CreateAdjustment(ctx, f.organizationID, f.actorID, biz.CreateCommissionAdjustmentInput{CommissionID: parent.ID, OrderID: f.orderID, Direction: biz.CommissionAdjustmentIncrease, Amount: decimal.NewFromInt(5), Reason: "手工奖励", IdempotencyKey: "manual-" + f.suffix})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = uc.ConfirmAdjustment(ctx, f.organizationID, f.actorID, manual.ID, manual.Version); err != nil {
					t.Fatal(err)
				}
				repo := NewFinanceBillRepo(f.data)
				audit := &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "finance.bill.cancel", Result: "invalid", ResourceType: "finance_bill", ResourceID: payable.ID.String()}
				// 让最后审计失败，验证已经计算出的差额也随账单一起回滚。
				if _, err = repo.Cancel(ctx, []uuid.UUID{f.organizationID}, payable.ID, f.actorID, payable.Version, "失败回滚", audit); err == nil {
					t.Fatal("非法审计必须失败")
				}
				requireBillEconomicNet(t, f, parent.ID, "35")
				unchanged, err := f.data.db.FinanceBill.Get(ctx, payable.ID)
				if err != nil || unchanged.Status != billent.StatusCONFIRMED || unchanged.Version != payable.Version {
					t.Fatalf("账单未完整回滚:%v %v", unchanged, err)
				}
				audit.Result = "success"
				if _, err = repo.Cancel(ctx, []uuid.UUID{f.organizationID}, payable.ID, f.actorID, payable.Version, "成本账单回退", audit); err != nil {
					t.Fatal(err)
				}
				current := requireBillEconomicNet(t, f, parent.ID, "45")
				if !paid {
					preview, err := uc.PreviewPayment(ctx, f.organizationID, parent.ID, current.Version)
					if err != nil || !preview.ActualAmount.Equal(decimal.NewFromInt(45)) {
						t.Fatalf("回退后发放预览=%v,%v", preview, err)
					}
				}
				billUC := biz.NewFinanceBillUsecase(repo, biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data), nil), f.data, nil)
				var rebuilt *biz.FinanceBill
				if batch {
					policy := biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}
					pre, err := billUC.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{cost.ID}, GroupingPolicy: policy})
					if err != nil {
						t.Fatal(err)
					}
					config := biz.FinanceBillBatchPreviewGroupConfig{GroupKey: pre.Groups[0].GroupKey, BillDate: financeCommissionIntegrationDate, SettlementAccountID: accountID}
					pre, err = billUC.PreviewBatch(ctx, f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{cost.ID}, GroupingPolicy: policy, GroupConfigs: []biz.FinanceBillBatchPreviewGroupConfig{config}})
					if err != nil {
						t.Fatal(err)
					}
					input := biz.CreateFinanceBillBatchInput{FeeIDs: []uuid.UUID{cost.ID}, GroupingPolicy: policy, PreviewToken: pre.PreviewToken, IdempotencyKey: "rebuild-batch-" + f.suffix, Groups: []biz.CreateFinanceBillBatchGroupInput{{GroupKey: config.GroupKey, StatementTitle: "测试账单", BillDate: config.BillDate, SettlementAccountID: config.SettlementAccountID}}}
					result, err := billUC.CreateBatch(ctx, f.organizationID, f.actorID, input)
					if err != nil {
						t.Fatal(err)
					}
					rebuilt = result.Bills[0]
					// 幂等回放不得再次生成差额。
					if _, err = billUC.CreateBatch(ctx, f.organizationID, f.actorID, input); err != nil {
						t.Fatal(err)
					}
				} else {
					input := biz.CreateFinanceBillInput{FeeIDs: []uuid.UUID{cost.ID}, BillDate: financeCommissionIntegrationDate, SettlementAccountID: accountID, IdempotencyKey: "rebuild-" + f.suffix}
					rebuilt, err = billUC.Create(ctx, f.organizationID, f.actorID, input)
					if err != nil {
						t.Fatal(err)
					}
					if _, err = billUC.Create(ctx, f.organizationID, f.actorID, input); err != nil {
						t.Fatal(err)
					}
				}
				requireBillEconomicNet(t, f, parent.ID, "35")
				if !rebuilt.BaseCurrencyAmount.Equal(decimal.NewFromInt(700)) || !rebuilt.TotalAmount.Equal(decimal.NewFromInt(100)) || len(rebuilt.Lines) != 1 || !rebuilt.Lines[0].BaseCurrencyAmount.Equal(decimal.NewFromInt(700)) || !rebuilt.Lines[0].ExchangeRate.Equal(decimal.NewFromInt(7)) {
					t.Fatalf("建账未按账单日派生头行金额:%v", rebuilt)
				}
				updated, err := billUC.Update(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.UpdateFinanceBillInput{ID: rebuilt.ID, ExpectedVersion: rebuilt.Version, BillDate: "2026-09-01", SettlementAccountID: accountID})
				if err != nil {
					t.Fatal(err)
				}
				current = requireBillEconomicNet(t, f, parent.ID, "25")
				if !updated.BaseCurrencyAmount.Equal(decimal.NewFromInt(800)) {
					t.Fatalf("改日期后账单本币=%s", updated.BaseCurrencyAmount)
				}
				if _, err = billUC.Update(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.UpdateFinanceBillInput{ID: updated.ID, ExpectedVersion: rebuilt.Version, BillDate: "2026-09-01", SettlementAccountID: accountID}); !errors.Is(err, biz.ErrFinanceBillVersionConflict) {
					t.Fatalf("旧版本更新错误=%v", err)
				}
				if _, err = repo.Cancel(ctx, []uuid.UUID{f.organizationID}, updated.ID, f.actorID, updated.Version, "二次回退", audit); err != nil {
					t.Fatal(err)
				}
				current = requireBillEconomicNet(t, f, parent.ID, "45")
				if current.CommissionAmount != "30.00000000" || (paid && (current.Status != commissionent.StatusPAID || current.PaidAmount == nil || *current.PaidAmount != "30.00000000")) {
					t.Fatalf("原计提/实付事实被改写:%v", current)
				}
				reward, err := f.data.db.FinanceCommissionAdjustment.Get(ctx, manual.ID)
				if err != nil || reward.Status != adjustmentent.StatusCONFIRMED || reward.Amount != "5.00000000" {
					t.Fatalf("手工调整被吞并:%v,%v", reward, err)
				}
				// 新增的批次记录先删除，随后由公共夹具清理它的账单及其余依赖。
				if batch {
					t.Cleanup(func() {
						if _, err := f.data.db.FinanceBillLine.Delete().Where(lineent.BillIDEQ(rebuilt.ID)).Exec(ctx); err != nil {
							t.Error(err)
						}
						if err := f.data.db.FinanceBill.DeleteOneID(rebuilt.ID).Exec(ctx); err != nil {
							t.Error(err)
						}
						if err := f.data.db.FinanceBillBatch.DeleteOneID(*rebuilt.BatchID).Exec(ctx); err != nil {
							t.Error(err)
						}
					})
				}
			})
		}
	}
}

func TestFinanceBillCancelSerializesCommissionPaymentPostgres(t *testing.T) {
	f := newCommissionPostgresFixture(t)
	ctx := context.Background()
	_, payable, _ := prepareBillEconomicCost(t, f)
	uc := f.newUsecase(NewCommissionRepo(f.data))
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("bill-payment-race"))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := uc.PreviewPayment(ctx, f.organizationID, parent.ID, parent.Version)
	if err != nil {
		t.Fatal(err)
	}
	reached, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.data.db.FinanceBill.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			mutation := m.(*ent.FinanceBillMutation)
			if state, ok := mutation.Status(); ok && state == billent.StatusCANCELLED {
				once.Do(func() { close(reached); <-release })
			}
			return next.Mutate(ctx, m)
		})
	})
	doneCancel := make(chan error, 1)
	go func() {
		_, err := NewFinanceBillRepo(f.data).Cancel(ctx, []uuid.UUID{f.organizationID}, payable.ID, f.actorID, payable.Version, "取消成本账单与提成支付竞争", &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "finance.bill.cancel", Result: "success"})
		doneCancel <- err
	}()
	select {
	case <-reached:
	case <-time.After(10 * time.Second):
		t.Fatal("取消未进入持锁写入阶段")
	}
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	donePay := make(chan error, 1)
	go func() {
		_, err := uc.MarkPaid(ctx, f.organizationID, f.actorID, parent.ID, parent.Version, biz.CommissionPaymentInput{PreviewToken: preview.PreviewToken, IdempotencyKey: "race-pay-" + f.suffix, ActualAmount: preview.ActualAmount})
		donePay <- err
	}()
	select {
	case err := <-donePay:
		t.Fatalf("取消持有订单锁时支付不应越过:%v", err)
	case <-time.After(100 * time.Millisecond):
	}
	unblock()
	select {
	case err := <-doneCancel:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("取消与支付发生锁序死锁")
	}
	select {
	case err := <-donePay:
		if !errors.Is(err, biz.ErrCommissionTransition) {
			t.Fatalf("旧支付预览应版本冲突:%v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("取消后支付未结束")
	}
	current := requireBillEconomicNet(t, f, parent.ID, "40")
	if current.Status != commissionent.StatusCONFIRMED || current.PaidAmount != nil {
		t.Fatalf("旧预览错误发放:%v", current)
	}
}
