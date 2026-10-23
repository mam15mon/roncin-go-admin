package data

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	taskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	adjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/shopspring/decimal"
)

// newCorrectionTestUsecase 使用真实授权与提成适配器，不以空实现跳过共享事务。
func newCorrectionTestUsecase(t *testing.T, f *feeSupplementPostgresFixture) *biz.OrderFeeCorrectionUsecase {
	t.Helper()
	for _, op := range []access.OrderOperation{access.OrderRead, access.OrderUpdate, access.OrderFeeCreate, access.OrderFeeUpdate, access.OrderFeeDelete, access.OrderFeeRead} {
		key := access.OrderPermission(access.OrderBusinessType("SI"), op)
		p, err := f.data.db.Permission.Query().Where(permissionent.KeyEQ(key)).First(f.ctx)
		if ent.IsNotFound(err) {
			p, err = f.data.db.Permission.Create().SetKey(key).SetName(key).SetDescription("费用更正测试").SetGroup("order").Save(f.ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.data.db.Role.UpdateOneID(f.lockRoleID).AddPermissions(p).Save(f.ctx); err != nil {
			t.Fatal(err)
		}
	}
	custom := biz.NewFinanceCustomSettingUsecase(NewFinanceCustomSettingRepo(f.data))
	credit := biz.NewPartnerCreditUsecase(NewFinanceBillRepo(f.data).(biz.PartnerCreditRepo), custom)
	fees := biz.NewOrderFeeUsecase(NewOrderFeeRepo(f.data), biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data), nil), custom, credit, nil)
	return biz.NewOrderFeeCorrectionUsecase(NewOrderFeeCorrectionRepo(f.data), fees, NewFeeCorrectionCommissionRepo(f.data), f.data)
}
func correctionTestInput(f *feeSupplementPostgresFixture, op, key string, source *biz.OrderFee, amount int64) *biz.OrderFeeCorrectionInput {
	in := &biz.OrderFeeCorrectionInput{Operation: op, Reason: "原始凭证核对后的费用更正", IdempotencyKey: key}
	if source != nil {
		in.SourceFeeID = &source.ID
		in.ExpectedFeeVersion = source.Version
	}
	if op != "VOID" {
		zero := decimal.Zero
		in.After = &biz.OrderFee{Direction: biz.OrderFeePayable, FeeSettingID: &f.feeSettingID, BillingUnitID: &f.billingUnitID, SettlementPartyID: f.customerID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(amount), TaxInclusive: true, TaxRateOverride: &zero, Currency: "CNY", ExpenseDate: "2026-09-01"}
	}
	return in
}
func correctionTestSource(t *testing.T, f *feeSupplementPostgresFixture, o *ent.Order, d biz.OrderFeeDirection) *biz.OrderFee {
	t.Helper()
	fees, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, fee := range fees {
		if fee.Direction == d {
			return fee
		}
	}
	t.Fatal("缺少原费用")
	return nil
}
func approveCorrectionForTest(t *testing.T, f *feeSupplementPostgresFixture, uc *biz.OrderFeeCorrectionUsecase, o *ent.Order, r *biz.OrderFeeCorrectionRequest) *biz.OrderFeeCorrectionRequest {
	t.Helper()
	p, err := uc.Preview(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	approved, err := uc.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, p.PreviewToken, "已核对更正及利润影响")
	if err != nil {
		t.Fatalf("批准失败: %v", err)
	}
	replay, err := uc.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, p.PreviewToken, "已核对更正及利润影响")
	if err != nil || replay.ID != approved.ID {
		t.Fatalf("审批重放失败: %v", err)
	}
	return approved
}
func TestFeeCorrectionEconomicOperationsPostgres(t *testing.T) {
	for _, op := range []string{"ADD", "REPLACE", "VOID"} {
		t.Run(op, func(t *testing.T) {
			f := newFeeSupplementFixture(t)
			uc := newCorrectionTestUsecase(t, f)
			o := f.createOrder("CORR-" + f.suffix)
			f.approveInitialFees(o)
			var before *biz.OrderFee
			if op != "ADD" {
				before = correctionTestSource(t, f, o, biz.OrderFeePayable)
			}
			in := correctionTestInput(f, op, "correction-"+op, before, 100)
			r, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, in, false)
			if err != nil {
				t.Fatal(err)
			}
			duplicate, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, in, false)
			if err != nil || duplicate.ID != r.ID {
				t.Fatalf("提交幂等失败: %v", err)
			}
			approved := approveCorrectionForTest(t, f, uc, o, r)
			if approved.Status != "APPROVED" || approved.ApprovalResult == nil {
				t.Fatalf("未保存批准结果: %+v", approved)
			}
			current, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(f.ctx, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, _, err := biz.BuildFeeReviewSnapshot(current)
			if err != nil {
				t.Fatal(err)
			}
			want := int64(500)
			count := 3
			if op == "REPLACE" {
				want = 100
				count = 2
			}
			if op == "VOID" {
				want = 0
				count = 1
			}
			if len(current) != count || !snapshot.Summary.Payable.Equal(decimal.NewFromInt(want)) {
				t.Fatalf("经济投影错误: count=%d summary=%+v", len(current), snapshot.Summary)
			}
			rows, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(o.ID)).All(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			var signed decimal.Decimal
			inverseCount := 0
			for _, row := range rows {
				if row.Direction != feeent.DirectionPAYABLE {
					continue
				}
				amount, _ := decimal.NewFromString(row.BaseCurrencyAmount)
				if row.EconomicEffect == feeent.EconomicEffectDECREASE {
					signed = signed.Sub(amount)
					inverseCount++
					if row.AccountingRequired {
						t.Fatal("原未建账时冲回不应建账")
					}
				} else {
					signed = signed.Add(amount)
				}
			}
			if !signed.Equal(decimal.NewFromInt(want)) {
				t.Fatalf("全量历史有符号金额%s应%d", signed, want)
			}
			if before != nil {
				old, err := f.data.db.OrderFee.Get(f.ctx, before.ID)
				if err != nil {
					t.Fatal(err)
				}
				if old.TotalAmount != before.TotalAmount.StringFixed(8) || old.SupersededByRequestID == nil || inverseCount != 1 {
					t.Fatalf("原费用必须不可变且冲回一次: %+v", old)
				}
				bad := correctionTestInput(f, "VOID", "old-again", before, 0)
				if _, err = uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, bad, false); err == nil {
					t.Fatal("原费用不能再次被更正")
				}
			}
			locked, err := f.data.db.Order.Get(f.ctx, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err = validateOrderApprovalSnapshot(f.ctx, f.data.db, locked); err != nil {
				t.Fatalf("新批准经济快照无效: %v", err)
			}
		})
	}
}

// 应收遗漏、客户优惠及录错收支/结算对象均走受控更正，而非只补成本。
func TestFeeCorrectionRevenueAndPartyPostgres(t *testing.T) {
	for _, scenario := range []string{"漏录应收", "应收优惠", "收支及结算对象录错"} {
		t.Run(scenario, func(t *testing.T) {
			f := newFeeSupplementFixture(t)
			uc := newCorrectionTestUsecase(t, f)
			o := f.createOrder("AR-CORR-" + f.suffix)
			f.approveInitialFees(o)
			op := "REPLACE"
			var before *biz.OrderFee
			if scenario == "漏录应收" {
				op = "ADD"
			} else if scenario == "应收优惠" {
				before = correctionTestSource(t, f, o, biz.OrderFeeReceivable)
			} else {
				before = correctionTestSource(t, f, o, biz.OrderFeePayable)
			}
			amount := int64(100)
			if scenario == "应收优惠" {
				amount = 800
			}
			input := correctionTestInput(f, op, "revenue-core", before, amount)
			input.After.Direction = biz.OrderFeeReceivable
			if scenario == "收支及结算对象录错" {
				party, err := f.data.db.Partner.Create().SetOrganizationID(f.organizationID).SetCode("CORR-NEW-" + f.suffix).SetLegalName("更正后的结算对象").SetNormalizedName("更正后的结算对象").Save(f.ctx)
				if err != nil {
					t.Fatal(err)
				}
				input.After.SettlementPartyID = party.ID
			}
			r, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, input, false)
			if err != nil {
				t.Fatal(err)
			}
			approved := approveCorrectionForTest(t, f, uc, o, r)
			summary := approved.ApprovalResult.Projected
			wantR, wantK := int64(1100), int64(400)
			if scenario == "应收优惠" {
				wantR = 800
			}
			if scenario == "收支及结算对象录错" {
				wantK = 0
			}
			if !summary.Receivable.Equal(decimal.NewFromInt(wantR)) || !summary.Payable.Equal(decimal.NewFromInt(wantK)) {
				t.Fatalf("应收更正经济口径错误: %+v", summary)
			}
			if approved.After == nil || approved.After.SettlementPartyID != input.After.SettlementPartyID || approved.After.Direction != biz.OrderFeeReceivable {
				t.Fatalf("新正式费用未沿用批准对象: %+v", approved.After)
			}
		})
	}
}
func TestFeeCorrectionPermissionsVersionsAndConcurrencyPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newCorrectionTestUsecase(t, f)
	o := f.createOrder("AUTH-" + f.suffix)
	f.approveInitialFees(o)
	before := correctionTestSource(t, f, o, biz.OrderFeePayable)
	in := correctionTestInput(f, "REPLACE", "auth-core", before, 100)
	if _, err := uc.Submit(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, in, false); err == nil {
		t.Fatal("无费用写权限不能更正")
	}
	wrong := *in
	wrong.ExpectedFeeVersion++
	if _, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, &wrong, false); err == nil {
		t.Fatal("源费用版本失效必须拒绝")
	}
	r, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, in, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = uc.Preview(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID, r.Version); err == nil {
		t.Fatal("无审核权限不能预览提成")
	}
	if _, err = uc.Get(f.ctx, f.requesterPrincipal(), f.organizationID, o.ID, r.ID); err == nil {
		t.Fatal("无权限不能读取别人的申请")
	}
	other := f.createOrder("OTHER-" + f.suffix)
	f.approveInitialFees(other)
	cross := correctionTestInput(f, "REPLACE", in.IdempotencyKey, correctionTestSource(t, f, other, biz.OrderFeePayable), 100)
	if _, err = uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, other.ID, cross, false); err == nil {
		t.Fatal("同幂等键不能返回其他订单的申请")
	}
	preview, err := uc.Preview(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.data.db.Order.UpdateOneID(o.ID).SetFeeReviewVersion(raw.FeeReviewVersion + 1).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = uc.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "版本失效"); err == nil {
		t.Fatal("失效预览不能批准")
	}
	count, err := f.data.db.OrderFee.Query().Where(feeent.CorrectionRequestIDEQ(r.ID)).Count(f.ctx)
	if err != nil || count != 0 {
		t.Fatalf("拒绝后不可残留费用: %d %v", count, err)
	}
	if _, err = uc.Decide(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, "REJECTED", "资料不足"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	out := make(chan error, 2)
	for _, key := range []string{"parallel-a", "parallel-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			<-start
			_, err := uc.Submit(context.Background(), f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "ADD", key, nil, 100), false)
			out <- err
		}(key)
	}
	close(start)
	wg.Wait()
	close(out)
	success := 0
	failure := 0
	for err := range out {
		if err == nil {
			success++
		} else {
			failure++
		}
	}
	if success != 1 || failure != 1 {
		t.Fatalf("同订单并发只准一个待审: %d/%d", success, failure)
	}
	if _, err = f.data.db.User.UpdateOneID(f.approverID).SetEnabled(false).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = uc.Get(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID); err == nil {
		t.Fatal("停用账号不能读取旧权限申请")
	}
}

// failingCorrectionCommissionRepo 保留真实预览，注入经济写入之后的提成失败，证明跨仓储原子性。
type failingCorrectionCommissionRepo struct {
	biz.FeeCorrectionCommissionRepo
}

func (r failingCorrectionCommissionRepo) ReconcileCorrectionImpact(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *biz.OrderFeeCorrectionRequest, *biz.FeeCorrectionCommissionImpact) error {
	return errors.New("提成更新注入失败")
}
func TestFeeCorrectionApprovalRollbackPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	normal := newCorrectionTestUsecase(t, f)
	o := f.createOrder("ROLLBACK-" + f.suffix)
	f.approveInitialFees(o)
	before := correctionTestSource(t, f, o, biz.OrderFeePayable)
	request, err := normal.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, correctionTestInput(f, "REPLACE", "rollback-core", before, 100), false)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := normal.Preview(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, request.ID, request.Version)
	if err != nil {
		t.Fatal(err)
	}
	orderBefore, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore, err := f.data.db.OrderFeeReviewEvent.Query().Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	tasksBefore, err := f.data.db.BackgroundTask.Query().Where(taskent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	adjustmentsBefore, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustmentent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	auditsBefore, err := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	failing := biz.NewOrderFeeCorrectionUsecase(NewOrderFeeCorrectionRepo(f.data), normalFeeUsecaseForRollback(f), failingCorrectionCommissionRepo{NewFeeCorrectionCommissionRepo(f.data)}, f.data)
	if _, err = failing.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, request.ID, request.Version, preview.PreviewToken, "回滚测试"); err == nil {
		t.Fatal("注入失败必须返回错误")
	}
	original, err := f.data.db.OrderFee.Get(f.ctx, before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if original.SupersededByRequestID != nil || original.Version != before.Version {
		t.Fatal("失败不得标记源费用被替代")
	}
	generated, err := f.data.db.OrderFee.Query().Where(feeent.CorrectionRequestIDEQ(request.ID)).Count(f.ctx)
	if err != nil || generated != 0 {
		t.Fatalf("失败残留冲回或替代:%d %v", generated, err)
	}
	orderAfter, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if orderAfter.FeeContentVersion != orderBefore.FeeContentVersion || orderAfter.FeeReviewVersion != orderBefore.FeeReviewVersion || orderAfter.LockGeneration != orderBefore.LockGeneration {
		t.Fatal("失败不得改经济/审核版本或批准事件")
	}
	pending, err := f.data.db.OrderFeeCorrectionRequest.Get(f.ctx, request.ID)
	if err != nil || pending.Status != "PENDING" || pending.Version != request.Version {
		t.Fatalf("失败请求必须保留待审:%+v %v", pending, err)
	}
	eventsAfter, _ := f.data.db.OrderFeeReviewEvent.Query().Count(f.ctx)
	tasksAfter, _ := f.data.db.BackgroundTask.Query().Where(taskent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	adjustmentsAfter, _ := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustmentent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	auditsAfter, _ := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).Count(f.ctx)
	if eventsAfter != eventsBefore || tasksAfter != tasksBefore || adjustmentsAfter != adjustmentsBefore || auditsAfter != auditsBefore {
		t.Fatalf("失败事务残留事件/消息/提成/审计: %d/%d %d/%d %d/%d %d/%d", eventsBefore, eventsAfter, tasksBefore, tasksAfter, adjustmentsBefore, adjustmentsAfter, auditsBefore, auditsAfter)
	}
	approveCorrectionForTest(t, f, normal, o, request)
}
func normalFeeUsecaseForRollback(f *feeSupplementPostgresFixture) *biz.OrderFeeUsecase {
	custom := biz.NewFinanceCustomSettingUsecase(NewFinanceCustomSettingRepo(f.data))
	credit := biz.NewPartnerCreditUsecase(NewFinanceBillRepo(f.data).(biz.PartnerCreditRepo), custom)
	return biz.NewOrderFeeUsecase(NewOrderFeeRepo(f.data), biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data), nil), custom, credit, nil)
}
