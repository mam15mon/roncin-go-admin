package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	"github.com/shopspring/decimal"
)

// TestCommissionPaymentSourceEvidencePostgres 核验来源反转只追回应付的实际现金敞口。
func TestCommissionPaymentSourceEvidencePostgres(t *testing.T) {
	for _, scenario := range []string{"待执行增提取消且仅追回实际已付", "无实际付款快照拒绝反转"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCommissionPostgresFixture(t)
			ctx := context.Background()
			uc := f.newUsecase(NewCommissionRepo(f.data))
			created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("payment-evidence"))
			if err != nil {
				t.Fatal(err)
			}
			confirmed, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
			if err != nil {
				t.Fatal(err)
			}
			paid, err := markCommissionPaidForTest(ctx, uc, f.organizationID, f.actorID, confirmed.ID, confirmed.Version)
			if err != nil {
				t.Fatal(err)
			}
			reverse := func() error {
				_, err := NewVerificationRepo(f.data).Reverse(ctx, f.organizationID, f.verificationID, f.actorID, 1, "付款敞口核验", &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "finance.verification.reverse", Result: "success", ResourceType: "finance_verification", ResourceID: f.verificationID.String()})
				return err
			}
			if scenario == "无实际付款快照拒绝反转" {
				decrease, err := uc.CreateAdjustment(ctx, f.organizationID, f.actorID, biz.CreateCommissionAdjustmentInput{CommissionID: paid.ID, OrderID: f.orderID, Direction: biz.CommissionAdjustmentDecrease, Amount: decimal.NewFromInt(5), Reason: "历史付款扣回", IdempotencyKey: "history-decrease-" + f.suffix})
				if err != nil {
					t.Fatal(err)
				}
				decrease, err = uc.ConfirmAdjustment(ctx, f.organizationID, f.actorID, decrease.ID, decrease.Version)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.data.db.FinanceCommission.UpdateOneID(paid.ID).ClearPaidAmount().ClearPaymentSnapshot().Exec(ctx); err != nil {
					t.Fatal(err)
				}
				projection, err := uc.GetAdjustmentScoped(ctx, []uuid.UUID{f.organizationID}, decrease.ID)
				if err != nil || projection.CanExecute || projection.ExecutionBlockedReason == nil {
					t.Fatalf("缺付款证据应禁止执行投影: %+v err=%v", projection, err)
				}
				if _, err := uc.MarkAdjustmentPaid(ctx, f.organizationID, f.actorID, decrease.ID, decrease.Version); !errors.Is(err, biz.ErrCommissionPaymentEvidence) {
					t.Fatalf("缺付款证据应阻断独立扣回: %v", err)
				}
				if err := reverse(); !errors.Is(err, biz.ErrCommissionPaymentEvidence) {
					t.Fatalf("付款证据不足应拒绝: %v", err)
				}
				rows, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(paid.ID), adjustment.SourceTypeEQ(adjustment.SourceTypeVERIFICATION_REVERSAL)).Count(ctx)
				if err != nil || rows != 0 {
					t.Fatalf("拒绝不能生成追回: count=%d err=%v", rows, err)
				}
				return
			}
			increase, err := uc.CreateAdjustment(ctx, f.organizationID, f.actorID, biz.CreateCommissionAdjustmentInput{CommissionID: paid.ID, OrderID: f.orderID, Direction: biz.CommissionAdjustmentIncrease, Amount: decimal.NewFromInt(10), Reason: "后续增提", IdempotencyKey: "increase-" + f.suffix})
			if err != nil {
				t.Fatal(err)
			}
			increase, err = uc.ConfirmAdjustment(ctx, f.organizationID, f.actorID, increase.ID, increase.Version)
			if err != nil {
				t.Fatal(err)
			}
			if err := reverse(); err != nil {
				t.Fatalf("取消未执行调整并追回实际现金: %v", err)
			}
			cancelled, err := f.data.db.FinanceCommissionAdjustment.Get(ctx, increase.ID)
			if err != nil || cancelled.Status != adjustment.StatusCANCELLED {
				t.Fatalf("未执行人工增提应保留取消历史: %+v,%v", cancelled, err)
			}
			recovery, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(paid.ID), adjustment.SourceTypeEQ(adjustment.SourceTypeVERIFICATION_REVERSAL)).Only(ctx)
			if err != nil || recovery.Amount != "60.00000000" {
				t.Fatalf("仅追回原真实已付60: %+v,%v", recovery, err)
			}

			parent, err := f.data.db.FinanceCommission.Get(ctx, paid.ID)
			if err != nil || parent.Status != commission.StatusPAID || parent.PaidAmount == nil || *parent.PaidAmount != "60.00000000" {
				t.Fatalf("父单实付历史必须保留: %+v err=%v", parent, err)
			}
		})
	}
}
