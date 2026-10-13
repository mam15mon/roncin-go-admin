package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	allocation "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	number "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// 退款先处理原收款中的一部分，再向客户收取替代账款也是合法财务路径。
// 此处刻意退款500超过优惠200，验证实际净收500低于应收800时重算为25。
func TestFeeCorrectionCommissionRefundAndSourceReversePostgres(t *testing.T) {
	for _, paid := range []bool{false, true} {
		name := "已确认未付"
		if paid {
			name = "已支付"
		}
		t.Run(name, func(t *testing.T) {
			f := newCommissionPostgresFixture(t)
			ctx := context.Background()
			requests := []uuid.UUID{}
			commissionCorrectionTestCleanup(t, f, &requests)
			// 更正夹具清理需先删除引用冲回账单的退款分摊。
			t.Cleanup(func() {
				_, _ = f.data.db.FinanceVerificationAllocation.Delete().Where(allocation.HasVerificationWith(verification.OrganizationIDEQ(f.organizationID)), allocation.VerificationIDNEQ(f.verificationID)).Exec(ctx)
				_, _ = f.data.db.FinanceVerification.Delete().Where(verification.OrganizationIDEQ(f.organizationID), verification.IDNEQ(f.verificationID)).Exec(ctx)
			})
			if _, err := f.data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(number.DocumentTypeWriteOff).SetPrefix("REF").SetDateFormat(number.DateFormatNone).SetSequenceLength(4).SetResetPolicy(number.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
				t.Fatal(err)
			}
			uc := f.newUsecase(NewCommissionRepo(f.data))
			created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("refund-parent"))
			if err != nil {
				t.Fatal(err)
			}
			parent, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
			if err != nil {
				t.Fatal(err)
			}
			if paid {
				parent, err = markCommissionPaidForTest(ctx, uc, f.organizationID, f.actorID, parent.ID, parent.Version)
				if err != nil {
					t.Fatal(err)
				}
			}
			assertEffective := func(amount int64) *biz.FinanceCommission {
				t.Helper()
				v, err := uc.Get(ctx, f.organizationID, parent.ID)
				if err != nil || !v.EffectiveCommissionAmount.Equal(decimal.NewFromInt(amount)) {
					t.Fatalf("有效提成应%d，得到%+v，错误%v", amount, v, err)
				}
				return v
			}
			assertEffective(60)
			fees, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(ctx, f.orderID)
			if err != nil {
				t.Fatal(err)
			}
			var before *biz.OrderFee
			for _, item := range fees {
				if item.Direction == biz.OrderFeeReceivable {
					before = item
				}
			}
			after := *before
			after.ID = uuid.New()
			after.Quantity = decimal.NewFromInt(1)
			after.UnitPrice = decimal.NewFromInt(800)
			after.TotalAmount, after.NetAmount, after.BaseCurrencyAmount = after.UnitPrice, after.UnitPrice, after.UnitPrice
			applyCommissionCorrectionTest(t, f, &requests, before, &after)
			assertEffective(40)
			rev, err := f.data.db.OrderFee.Query().Where(fee.CorrectionRequestIDEQ(requests[0]), fee.EconomicEffectEQ(fee.EconomicEffectDECREASE)).Only(ctx)
			if err != nil {
				t.Fatal(err)
			}
			ff := &financeInvoicePostgresFixture{data: f.data, organizationID: f.organizationID, orderID: f.orderID, partnerID: f.customerID, actorID: f.actorID, suffix: f.suffix}
			reverseBill := correctionFinanceBill(t, ff, "refund-reversal", bill.PurposeCORRECTION_REVERSAL, rev)
			refundCash := correctionFinanceCash(t, ff, "refund-500", "500", cash.DirectionPAYABLE)
			refund, err := correctionFinanceVerification(t, ff, refundCash, reverseBill, "500")
			if err != nil {
				t.Fatal(err)
			}
			assertEffective(25)
			current, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(parent.ID), adjustment.StatusEQ(adjustment.StatusCONFIRMED)).Only(ctx)
			if err != nil || current.SourceType != adjustment.SourceTypeECONOMIC_REALLOCATION || current.Amount != "35.00000000" || current.SourceVerificationID == nil || *current.SourceVerificationID != refund.ID {
				t.Fatalf("退款应净重建减35并关联退款核销，得到%+v，错误%v", current, err)
			}
			vr := NewVerificationRepo(f.data)
			if _, err = vr.Reverse(ctx, f.organizationID, f.verificationID, f.actorID, 1, "先撤原收款", &biz.AuditEvent{Action: "test.source.reverse", Result: "success"}); err == nil {
				t.Fatal("存在有效退款时必须阻止撤原收款")
			}
			source, err := f.data.db.FinanceVerification.Get(ctx, f.verificationID)
			if err != nil || source.Status != verification.StatusACTIVE || source.Version != 1 {
				t.Fatal("失败反转必须整体回滚", err)
			}
			assertEffective(25)
			if _, err = vr.Reverse(ctx, f.organizationID, refund.ID, f.actorID, refund.Version, "撤销退款", &biz.AuditEvent{Action: "test.refund.reverse", Result: "success"}); err != nil {
				t.Fatal(err)
			}
			assertEffective(40)
			if _, err = vr.Reverse(ctx, f.organizationID, f.verificationID, f.actorID, 1, "撤销原收款", &biz.AuditEvent{Action: "test.source.reverse", Result: "success"}); err != nil {
				t.Fatal(err)
			}
			result, err := uc.Get(ctx, f.organizationID, parent.ID)
			if err != nil {
				t.Fatal(err)
			}
			active := 0
			for _, row := range result.Adjustments {
				if string(row.Status) == string(adjustment.StatusCONFIRMED) {
					active++
					if !paid || row.SourceType != biz.CommissionAdjustmentSourceVerificationReversal || !row.Amount.Equal(decimal.NewFromInt(60)) {
						t.Fatalf("已支付只能有一笔按真实敞口60追回，得到%+v", row)
					}
				}
			}
			if paid {
				if result.Status != biz.CommissionPaid || active != 1 || result.PaidAmount == nil || !result.PaidAmount.Equal(decimal.NewFromInt(60)) {
					t.Fatalf("来源反转不得改写已支付60：%+v", result)
				}
			} else if string(result.Status) != string(commission.StatusCANCELLED) || active != 0 {
				t.Fatalf("未付父单及未执行调整须取消：%+v", result)
			}
		})
	}
}
