package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// correctedCommissionGateFixture 先回退成本账单，再真实批准 REPLACE/VOID；保留
// 不需建账的原费与冲回行，避免以手工拼接历史标记代替触发场景。
func correctedCommissionGateFixture(t *testing.T, operation string, billReplacement bool) (*commissionApplicationFixture, uuid.UUID, uuid.UUID) {
	t.Helper()
	f := newCommissionApplicationFixture(t)
	employee := f.newEmployee("correction-gate")
	f.addAttribution(employee, "SALES")
	f.addSalesScheme(employee)
	ctx := context.Background()
	cost, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(f.orderID), feeent.DirectionEQ(feeent.DirectionPAYABLE)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := f.data.db.FinanceBillLine.Query().Where(billlineent.OrderFeeIDEQ(cost.ID)).All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		bill, err := f.data.db.FinanceBill.Get(ctx, line.BillID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = NewFinanceBillRepo(f.data).Cancel(ctx, []uuid.UUID{f.organizationID}, bill.ID, f.actorID, bill.Version, "回退成本账单后更正", &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "finance.bill.cancel", Result: "success"}); err != nil {
			t.Fatal(err)
		}
	}
	// 此夹具不含海运单证，使用与既有更正提成夹具一致的 SI 品类满足正式锁快照约束。
	if _, err = f.data.db.Order.UpdateOneID(f.orderID).SetBusinessType(orderent.BusinessTypeSI).Save(ctx); err != nil {
		t.Fatal(err)
	}
	(&workbenchIntegrationFixture{t: t, data: f.data, organizationID: f.organizationID, suffix: f.suffix}).grantWorkbenchPermission(employee, "business.order.si.lock", "business.order.si.read", "business.order.si.update", "business.order.si.fee.read")
	seedManualOrderReviewForTest(t, f.data, f.orderID, f.actorID)
	current, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	var before *biz.OrderFee
	for _, item := range current {
		if item.ID == cost.ID {
			before = item
		}
	}
	if before == nil {
		t.Fatal("缺少原成本")
	}
	var after *biz.OrderFee
	if operation == "REPLACE" {
		copy := *before
		copy.ID = uuid.New()
		copy.Quantity = decimal.NewFromInt(1)
		copy.UnitPrice = decimal.NewFromInt(200)
		copy.TotalAmount = copy.UnitPrice
		copy.NetAmount = copy.UnitPrice
		copy.BaseCurrencyAmount = copy.UnitPrice
		after = &copy
	}
	adapter := &commissionPostgresFixture{t: t, data: f.data, organizationID: f.organizationID, customerID: f.customerID, employeeID: employee, orderID: f.orderID, actorID: f.actorID, suffix: f.suffix}
	requests := []uuid.UUID{}
	applyCommissionCorrectionTest(t, adapter, &requests, before, after)
	history, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(f.orderID), feeent.AccountingRequiredEQ(false)).All(ctx)
	if err != nil || len(history) != 2 {
		t.Fatalf("应保留原费和冲回两条不需建账历史: %d %v", len(history), err)
	}
	if after != nil && billReplacement {
		bill, err := withTestFinanceBillSettlementAccountSnapshot(f.data.db.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo("replacement-cost").SetIdempotencyKey("replacement-cost").SetDirection(billent.DirectionPAYABLE).SetStatus(billent.StatusCONFIRMED).SetSettlementPartyID(f.customerID).SetSettlementPartyName("客户").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(billent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(f.historicalDate).SetTotalAmount("200").SetNetAmount("200").SetTaxAmount("0").SetBaseCurrencyAmount("200").SetFeeCount(1).SetBillDate(f.historicalDate), uuid.New(), "CNY").Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.data.db.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderID(f.orderID).SetOrderFeeID(after.ID).SetOrderNo("replacement-cost").SetFeeCode("COST").SetFeeName("替代成本").SetQuantity("1").SetUnitPrice("200").SetTotalAmount("200").SetNetAmount("200").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrency("CNY").SetBaseCurrencyAmount("200").SetActive(true).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	verificationID := f.addVerification(f.historicalDate)
	return f, employee, verificationID
}

func TestCommissionCorrectionHistoryConfirmationPostgres(t *testing.T) {
	for _, operation := range []string{"REPLACE", "VOID"} {
		for _, monthly := range []bool{false, true} {
			name := operation + map[bool]string{false: "逐笔确认", true: "月申整批批准"}[monthly]
			t.Run(name, func(t *testing.T) {
				f, employee, verificationID := correctedCommissionGateFixture(t, operation, true)
				ctx := context.Background()
				expected := decimal.NewFromInt(80)
				if operation == "VOID" {
					expected = decimal.NewFromInt(100)
				}
				if monthly {
					application, err := f.usecase().Submit(ctx, f.scopeFor(employee))
					if err != nil {
						t.Fatal(err)
					}
					approved, err := f.usecase().Approve(ctx, []uuid.UUID{f.organizationID}, f.actorID, application.ID, application.Version)
					if err != nil {
						t.Fatalf("历史行不得阻断月申批准: %v", err)
					}
					if approved.Status != biz.CommissionApplicationApproved || !approved.TotalCommissionAmount.Equal(expected) {
						t.Fatalf("整批批准状态或金额错误: %+v", approved)
					}
				} else {
					uc := f.commissionUsecase()
					created, err := uc.Create(ctx, f.organizationID, f.actorID, biz.CreateCommissionInput{VerificationID: verificationID, EmployeeID: employee, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: uuid.NewString()})
					if err != nil {
						t.Fatal(err)
					}
					confirmed, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
					if err != nil {
						t.Fatalf("历史行不得阻断确认: %v", err)
					}
					if confirmed.Status != biz.CommissionConfirmed || !confirmed.CommissionAmount.Equal(expected) {
						t.Fatalf("确认状态或金额错误: %+v", confirmed)
					}
				}
				if count, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(f.orderID), feeent.AccountingRequiredEQ(false)).Count(ctx); err != nil || count != 2 {
					t.Fatalf("确认不得删除历史: %d %v", count, err)
				}
			})
		}
	}
}

func TestCommissionCurrentUnbilledCorrectionStillBlocksPostgres(t *testing.T) {
	for _, monthly := range []bool{false, true} {
		t.Run(map[bool]string{false: "逐笔确认", true: "月申批准"}[monthly], func(t *testing.T) {
			f, employee, verificationID := correctedCommissionGateFixture(t, "REPLACE", false)
			ctx := context.Background()
			if monthly {
				application, err := f.usecase().Submit(ctx, f.scopeFor(employee))
				if err != nil {
					t.Fatal(err)
				}
				if _, err = f.usecase().Approve(ctx, []uuid.UUID{f.organizationID}, f.actorID, application.ID, application.Version); !errors.Is(err, biz.ErrCommissionUnconfirmedFees) {
					t.Fatalf("当前替代费未建账仍须阻断: %v", err)
				}
				assertApplicationUntouched(t, f, application.ID, "当前替代费未建账")
			} else {
				uc := f.commissionUsecase()
				created, err := uc.Create(ctx, f.organizationID, f.actorID, biz.CreateCommissionInput{VerificationID: verificationID, EmployeeID: employee, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: uuid.NewString()})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version); !errors.Is(err, biz.ErrCommissionUnconfirmedFees) {
					t.Fatalf("当前替代费未建账仍须阻断: %v", err)
				}
				row, err := f.data.db.FinanceCommission.Get(ctx, created.ID)
				if err != nil || row.Version != created.Version || row.Status != commissionent.StatusDRAFT {
					t.Fatalf("拒绝确认不得部分写入: %+v %v", row, err)
				}
			}
		})
	}
}
