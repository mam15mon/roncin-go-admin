package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	background "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	delivery "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	event "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
	lockent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	role "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	assignment "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	"github.com/shopspring/decimal"
)

func commissionCorrectionTestAudit(f *commissionPostgresFixture, id uuid.UUID) *biz.AuditEvent {
	return &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "order.fee_correction.approve", Result: "success", ResourceType: "order_fee_correction_request", ResourceID: id.String()}
}
func commissionCorrectionTestCleanup(t *testing.T, f *commissionPostgresFixture, requests *[]uuid.UUID) {
	t.Helper()
	if _, err := f.data.db.Order.UpdateOneID(f.orderID).SetBusinessType(orderent.BusinessTypeSI).Save(context.Background()); err != nil {
		t.Fatal(err)
	}
	(&workbenchIntegrationFixture{t: t, data: f.data, organizationID: f.organizationID, suffix: f.suffix}).grantWorkbenchPermission(f.employeeID, "business.order.si.lock", "business.order.si.read", "business.order.si.update", "business.order.si.fee.read")
	seedManualOrderReviewForTest(t, f.data, f.orderID, f.actorID)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = f.data.db.NotificationDelivery.Delete().Where(delivery.HasBackgroundTaskWith(background.OrganizationIDEQ(f.organizationID))).Exec(ctx)
		_, _ = f.data.db.BackgroundTask.Delete().Where(background.OrganizationIDEQ(f.organizationID)).Exec(ctx)
		_, _ = f.data.db.RoleAssignment.Delete().Where(assignment.HasRoleWith(role.OrganizationIDEQ(f.organizationID))).Exec(ctx)
		_, _ = f.data.db.Role.Delete().Where(role.OrganizationIDEQ(f.organizationID)).Exec(ctx)
		_, _ = f.data.db.FinanceCommissionAdjustment.Delete().Where(adjustment.OrganizationIDEQ(f.organizationID)).Exec(ctx)
		_, _ = f.data.db.OrderFee.Update().Where(fee.OrderIDEQ(f.orderID)).ClearSupersededByRequestID().Save(ctx)
		_, _ = f.data.db.Order.UpdateOneID(f.orderID).ClearCurrentFeeReviewEventID().ClearApprovedFeeReviewEventID().Save(ctx)
		_, _ = f.data.db.OrderFeeReviewEvent.Delete().Where(event.OrderIDEQ(f.orderID)).Exec(ctx)
		for i := len(*requests) - 1; i >= 0; i-- {
			id := (*requests)[i]
			lines, err := f.data.db.FinanceBillLine.Query().Where(billline.HasOrderFeeWith(fee.CorrectionRequestIDEQ(id))).All(ctx)
			if err != nil {
				t.Error(err)
			}
			for _, l := range lines {
				if err = f.data.db.FinanceBillLine.DeleteOneID(l.ID).Exec(ctx); err != nil {
					t.Error(err)
				}
				if err = f.data.db.FinanceBill.DeleteOneID(l.BillID).Exec(ctx); err != nil {
					t.Error(err)
				}
			}
			if _, err := f.data.db.OrderFee.Delete().Where(fee.CorrectionRequestIDEQ(id)).Exec(ctx); err != nil {
				t.Errorf("清理更正生成行: %v", err)
			}
			if err := f.data.db.OrderFeeCorrectionRequest.DeleteOneID(id).Exec(ctx); err != nil && !ent.IsNotFound(err) {
				t.Errorf("清理更正申请: %v", err)
			}
		}
		_, _ = f.data.db.OrderLockRecord.Delete().Where(lockent.OrderIDEQ(f.orderID)).Exec(ctx)
	})
}

// applyCommissionCorrectionTest 通过正式更正仓储在共享事务内执行费用及提成计划。
func applyCommissionCorrectionTest(t *testing.T, f *commissionPostgresFixture, requests *[]uuid.UUID, before, after *biz.OrderFee) *biz.FeeCorrectionCommissionImpact {
	t.Helper()
	ctx := context.Background()
	repo := NewOrderFeeCorrectionRepo(f.data)
	commissionRepo := NewFeeCorrectionCommissionRepo(f.data)
	var impact *biz.FeeCorrectionCommissionImpact
	id := uuid.New()
	*requests = append(*requests, id)
	err := f.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		evidence, err := repo.LockOrder(txCtx, f.organizationID, f.orderID)
		if err != nil {
			return err
		}
		o, err := f.data.client(txCtx)
		if err != nil {
			return err
		}
		rawOrder, err := o.Order.Get(txCtx, f.orderID)
		if err != nil {
			return err
		}
		req := &biz.OrderFeeCorrectionRequest{ID: id, OrganizationID: f.organizationID, OrderID: f.orderID, Operation: "ADD", After: after, Before: before, LockRecordID: *evidence.LockRecordID, LockGeneration: evidence.BusinessLockGeneration, ApprovedFeeContentVersion: rawOrder.FeeContentVersion, ApprovedFeeReviewVersion: rawOrder.FeeReviewVersion, Reason: "更正回归", IdempotencyKey: id.String(), RequestFingerprint: "test-" + id.String(), RequestedBy: f.actorID, RequestedAt: time.Now().UTC(), Status: "PENDING", Version: 1}
		_ = evidence
		if before != nil {
			req.Operation = "REPLACE"
			req.SourceFeeID = &before.ID
		}
		if after == nil {
			req.Operation = "VOID"
		}
		if _, err = repo.Create(txCtx, req, commissionCorrectionTestAudit(f, id)); err != nil {
			return err
		}
		current, err := repo.CurrentFees(txCtx, f.orderID)
		if err != nil {
			return err
		}
		projected := make([]*biz.OrderFee, 0, len(current)+1)
		for _, fee := range current {
			if before == nil || fee.ID != before.ID {
				projected = append(projected, fee)
			}
		}
		if after != nil {
			projected = append(projected, after)
		}
		impact, err = commissionRepo.PreviewCorrectionImpact(txCtx, f.organizationID, f.orderID, req, current, projected)
		if err != nil {
			return err
		}
		if err = repo.Apply(txCtx, req, f.actorID, "批准回归"); err != nil {
			return err
		}
		if err = commissionRepo.ReconcileCorrectionImpact(txCtx, f.organizationID, f.orderID, f.actorID, req, impact); err != nil {
			return err
		}
		return repo.SaveDecision(txCtx, req, f.actorID, "APPROVED", "批准回归", "test-token")
	})
	if err != nil {
		t.Fatal(err)
	}
	return impact
}
func TestFeeCorrectionCommissionPartialRevenuePostgres(t *testing.T) {
	f := newCommissionPostgresFixture(t)
	ctx := context.Background()
	requests := []uuid.UUID{}
	commissionCorrectionTestCleanup(t, f, &requests)
	v, err := f.data.db.FinanceVerification.Get(ctx, f.verificationID)
	if err != nil {
		t.Fatal(err)
	}
	a, err := f.data.db.FinanceVerificationAllocation.Query().Where(allocation.VerificationIDEQ(f.verificationID)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.data.db.FinanceVerificationAllocation.DeleteOneID(a.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err = f.data.db.FinanceVerification.DeleteOneID(v.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.FinanceVerification.Create().SetID(v.ID).SetOrganizationID(v.OrganizationID).SetVerificationNo(v.VerificationNo).SetIdempotencyKey(v.IdempotencyKey).SetDirection(v.Direction).SetStatus(v.Status).SetSettlementPartyID(v.SettlementPartyID).SetSettlementPartyName(v.SettlementPartyName).SetCurrency(v.Currency).SetAmount("500.00000000").SetBaseCurrency(v.BaseCurrency).SetBaseAmount("500.00000000").SetBillBaseAmount("500.00000000").SetCashflowBaseAmount("500.00000000").SetExchangeGainLoss("0.00000000").SetVerificationDate(v.VerificationDate).SetVersion(1).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().SetID(a.ID).SetVerificationID(v.ID).SetCashflowID(a.CashflowID).SetBillID(a.BillID).SetCashflowNo(a.CashflowNo).SetBillNo(a.BillNo).SetAmount("500.00000000").SetBillBaseAmount("500.00000000").SetCashflowBaseAmount("500.00000000").SetExchangeGainLoss("0.00000000").SetActive(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	uc := f.newUsecase(NewCommissionRepo(f.data))
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("partial-correction"))
	if err != nil {
		t.Fatal(err)
	}
	parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !parent.CommissionAmount.Equal(decimal.NewFromInt(30)) {
		t.Fatalf("原提成应30,得到%s", parent.CommissionAmount)
	}
	fees, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	var before *biz.OrderFee
	for _, fee := range fees {
		if fee.Direction == biz.OrderFeePayable {
			before = fee
		}
	}
	after := *before
	after.ID = uuid.New()
	after.Quantity = decimal.NewFromInt(1)
	after.UnitPrice = decimal.NewFromInt(200)
	after.TotalAmount = after.UnitPrice
	after.NetAmount = after.UnitPrice
	after.BaseCurrencyAmount = after.UnitPrice
	impact := applyCommissionCorrectionTest(t, f, &requests, nil, &after)
	if len(impact.Lines) != 1 || !impact.Lines[0].TargetAmount.Equal(decimal.NewFromInt(20)) || !impact.Lines[0].Difference.Equal(decimal.NewFromInt(-10)) {
		t.Fatalf("部分回款增加成本后应20且调减10: %+v", impact)
	}
	result, err := uc.Get(ctx, f.organizationID, parent.ID)
	if err != nil || !result.EffectiveCommissionAmount.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("有效权益应20: %+v,%v", result, err)
	}
}
func TestFeeCorrectionCommissionPaidNetRebuildPostgres(t *testing.T) {
	for _, state := range []string{"父未付冲减未执行", "父已付冲减未执行", "父已付且独立追回", "父净额零已付", "人工减提并存"} {
		t.Run(state, func(t *testing.T) {
			f := newCommissionPostgresFixture(t)
			ctx := context.Background()
			requests := []uuid.UUID{}
			commissionCorrectionTestCleanup(t, f, &requests)
			// 夹具建好后删去测试原成本，让原始提成100；只操作本隔离Schema。
			costs, err := f.data.db.OrderFee.Query().Where(fee.OrderIDEQ(f.orderID), fee.DirectionEQ(fee.DirectionPAYABLE)).All(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, cost := range costs {
				lines, err := f.data.db.FinanceBillLine.Query().Where(billline.OrderFeeIDEQ(cost.ID)).All(ctx)
				if err != nil {
					t.Fatal(err)
				}
				for _, l := range lines {
					if err = f.data.db.FinanceBillLine.DeleteOneID(l.ID).Exec(ctx); err != nil {
						t.Fatal(err)
					}
					if err = f.data.db.FinanceBill.DeleteOneID(l.BillID).Exec(ctx); err != nil {
						t.Fatal(err)
					}
				}
				if err = f.data.db.OrderFee.DeleteOneID(cost.ID).Exec(ctx); err != nil {
					t.Fatal(err)
				}
			}
			seedManualOrderReviewForTest(t, f.data, f.orderID, f.actorID)
			uc := f.newUsecase(NewCommissionRepo(f.data))
			created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("net-rebuild"))
			if err != nil {
				t.Fatal(err)
			}
			parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
			if err != nil {
				t.Fatal(err)
			}
			if state == "人工减提并存" {
				a, err := uc.CreateAdjustment(ctx, f.organizationID, f.actorID, biz.CreateCommissionAdjustmentInput{CommissionID: parent.ID, OrderID: f.orderID, Direction: biz.CommissionAdjustmentDecrease, Amount: decimal.NewFromInt(80), Reason: "人工处罚保留", IdempotencyKey: "manual-80"})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = uc.ConfirmAdjustment(ctx, f.organizationID, f.actorID, a.ID, a.Version); err != nil {
					t.Fatal(err)
				}
			}
			if state == "父已付冲减未执行" || state == "父已付且独立追回" {
				parent, err = markCommissionPaidForTest(ctx, uc, f.organizationID, f.actorID, parent.ID, parent.Version)
				if err != nil {
					t.Fatal(err)
				}
			}
			// 首次批准误录成本1000，经济目标0。
			current, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
			if err != nil {
				t.Fatal(err)
			}
			cost := *current[0]
			cost.ID = uuid.New()
			cost.Direction = biz.OrderFeePayable
			cost.Quantity = decimal.NewFromInt(1)
			cost.UnitPrice = decimal.NewFromInt(1000)
			cost.TotalAmount = cost.UnitPrice
			cost.NetAmount = cost.UnitPrice
			cost.BaseCurrencyAmount = cost.UnitPrice
			applyCommissionCorrectionTest(t, f, &requests, nil, &cost)
			result, err := uc.Get(ctx, f.organizationID, parent.ID)
			if err != nil {
				t.Fatal(err)
			}
			var originalAdjustment *biz.FinanceCommissionAdjustment
			for _, a := range result.Adjustments {
				if a.SourceType == biz.CommissionAdjustmentSourceFeeCorrection {
					originalAdjustment = a
				}
			}
			if originalAdjustment == nil {
				t.Fatal("缺系统调整")
			}
			if state == "人工减提并存" && (!originalAdjustment.Amount.Equal(decimal.NewFromInt(20))) {
				t.Fatalf("人工80后系统最多减20: %+v", originalAdjustment)
			}
			if state == "父已付且独立追回" {
				if _, err = uc.MarkAdjustmentPaid(ctx, f.organizationID, f.actorID, originalAdjustment.ID, originalAdjustment.Version); err != nil {
					t.Fatal(err)
				}
			}
			if state == "父净额零已付" {
				parent, err = markCommissionPaidForTest(ctx, uc, f.organizationID, f.actorID, result.ID, result.Version)
				if err != nil {
					t.Fatal(err)
				}
				if parent.PaidAmount == nil || !parent.PaidAmount.IsZero() {
					t.Fatalf("应按零净额支付: %+v", parent)
				}
			}
			current, err = NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
			if err != nil {
				t.Fatal(err)
			}
			var before *biz.OrderFee
			for _, fee := range current {
				if fee.ID == cost.ID {
					before = fee
				}
			}
			after := *before
			after.ID = uuid.New()
			after.UnitPrice = decimal.NewFromInt(100)
			after.TotalAmount = after.UnitPrice
			after.NetAmount = after.UnitPrice
			after.BaseCurrencyAmount = after.UnitPrice
			impact := applyCommissionCorrectionTest(t, f, &requests, before, &after)
			expected := decimal.NewFromInt(90)
			if state == "人工减提并存" {
				expected = decimal.NewFromInt(10)
			}
			if len(impact.Lines) != 1 || !impact.Lines[0].TargetAmount.Equal(expected) {
				t.Fatalf("1000→100目标90: %+v", impact)
			}
			result, err = uc.Get(ctx, f.organizationID, parent.ID)
			if err != nil || !result.EffectiveCommissionAmount.Equal(expected) {
				t.Fatalf("最终有效权益应90: %+v,%v", result, err)
			}
			old, err := f.data.db.FinanceCommissionAdjustment.Get(ctx, originalAdjustment.ID)
			if err != nil {
				t.Fatal(err)
			}
			newest, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(parent.ID), adjustment.SourceFeeCorrectionRequestIDEQ(requests[1])).Only(ctx)
			if err != nil {
				t.Fatal(err)
			}
			executed := state == "父已付且独立追回" || state == "父净额零已付"
			if executed {
				if old.Status != adjustment.StatusPAID || newest.Direction != adjustment.DirectionINCREASE || newest.Amount != "90.00000000" {
					t.Fatalf("已执行保持并待补90: old=%+v,new=%+v", old, newest)
				}
			} else {
				if old.Status != adjustment.StatusCANCELLED || old.SupersededByFeeCorrectionRequestID == nil || newest.Direction != adjustment.DirectionDECREASE || newest.Amount != "10.00000000" {
					t.Fatalf("未执行净重建减10: old=%+v,new=%+v", old, newest)
				}
			}
		})
	}
}

func TestFeeCorrectionCommissionFirstGenerationPostgres(t *testing.T) {
	f := newCommissionPostgresFixture(t)
	ctx := context.Background()
	requests := []uuid.UUID{}
	commissionCorrectionTestCleanup(t, f, &requests)
	fees, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	cost := *fees[0]
	cost.ID = uuid.New()
	cost.Direction = biz.OrderFeePayable
	cost.UnitPrice = decimal.NewFromInt(100)
	cost.TotalAmount = cost.UnitPrice
	cost.NetAmount = cost.UnitPrice
	cost.BaseCurrencyAmount = cost.UnitPrice
	impact := applyCommissionCorrectionTest(t, f, &requests, nil, &cost)
	if len(impact.Lines) != 0 {
		t.Fatal("首次计提前不生成空调整")
	}
	b, err := withTestFinanceBillSettlementAccountSnapshot(f.data.db.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo("new-cost").SetIdempotencyKey("new-cost").SetDirection(bill.DirectionPAYABLE).SetStatus(bill.StatusCONFIRMED).SetSettlementPartyID(f.customerID).SetSettlementPartyName("客户").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(bill.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeCommissionIntegrationDate).SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").SetFeeCount(1).SetBillDate(financeCommissionIntegrationDate), uuid.New(), "CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.FinanceBillLine.Create().SetBillID(b.ID).SetOrderID(f.orderID).SetOrderFeeID(cost.ID).SetOrderNo("new-cost").SetFeeCode("COST").SetFeeName("新增成本").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrencyAmount("100").SetBaseCurrency("CNY").SetActive(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	uc := f.newUsecase(NewCommissionRepo(f.data))
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("after-correction-generation"))
	if err != nil {
		t.Fatal(err)
	}
	if !created.CommissionAmount.Equal(decimal.NewFromInt(50)) || len(created.Lines) != 1 || created.Lines[0].RealizationSnapshot == nil || created.Lines[0].FeeContentVersionSnapshot < 2 {
		t.Fatalf("首次应完整含新成本500与来源证据，目标50: %+v", created)
	}
	parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	second := cost
	second.ID = uuid.New()
	applyCommissionCorrectionTest(t, f, &requests, nil, &second)
	result, err := uc.Get(ctx, f.organizationID, parent.ID)
	if err != nil || !result.EffectiveCommissionAmount.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("首次已含历史成本，后来只多减10: %+v,%v", result, err)
	}
	for _, a := range result.Adjustments {
		if _, err = uc.CancelAdjustment(ctx, f.organizationID, f.actorID, a.ID, a.Version, "绕过更正"); err == nil {
			t.Fatal("费用更正系统调整不可人工取消")
		}
		if _, err = uc.ConfirmAdjustment(ctx, f.organizationID, f.actorID, a.ID, a.Version); err == nil {
			t.Fatal("费用更正系统调整不可人工确认")
		}
	}
}
