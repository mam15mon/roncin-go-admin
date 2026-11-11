package data

import (
	"context"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
)

func TestCommissionAdjustmentCancelIncreaseBalancePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newAdjustmentTransitionUsecase(f)
	ctx := context.Background()
	orderA := f.createOrder("SE-CANCEL-BAL-A-" + f.suffix)
	orderB := f.createOrder("SE-CANCEL-BAL-B-" + f.suffix)

	for _, tc := range []struct {
		name, otherLineAmount, decrease string
		wantRejected                    bool
	}{
		{name: "100加100减150后取消增提不得使净额为负", otherLineAmount: "0", decrease: "150", wantRejected: true},
		{name: "其他订单正余额不能掩盖本行取消后负额", otherLineAmount: "500", decrease: "150", wantRejected: true},
		{name: "取消后保留正余额时成功", otherLineAmount: "0", decrease: "80"},
		{name: "取消后净额为零时成功", otherLineAmount: "0", decrease: "100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f.setT(t)
			parent := insertTwoLineCommission(f, orderA, orderB, "100", tc.otherLineAmount)
			increase := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionINCREASE, commissionadjustmentent.StatusDRAFT, "100", "cancel-inc-"+parent.ID.String(), "手工增提")
			confirmedIncrease, err := uc.ConfirmAdjustment(ctx, f.organizationID, f.approverID, increase.ID, increase.Version)
			if err != nil {
				t.Fatalf("确认增提: %v", err)
			}
			decrease := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionDECREASE, commissionadjustmentent.StatusDRAFT, tc.decrease, "cancel-dec-"+parent.ID.String(), "依赖增提的冲减")
			confirmedDecrease, err := uc.ConfirmAdjustment(ctx, f.organizationID, f.approverID, decrease.ID, decrease.Version)
			if err != nil {
				t.Fatalf("确认余额内冲减: %v", err)
			}

			// 草稿及已取消增提不能充当取消后的有效余额。
			insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionINCREASE, commissionadjustmentent.StatusDRAFT, "200", "cancel-unused-draft-"+parent.ID.String(), "未确认增提")
			insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionINCREASE, commissionadjustmentent.StatusCANCELLED, "200", "cancel-unused-cancelled-"+parent.ID.String(), "已取消增提")
			beforeAudit, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, err = uc.CancelAdjustment(ctx, f.organizationID, f.approverID, increase.ID, confirmedIncrease.Version, "撤销增提")
			if tc.wantRejected {
				if err == nil || kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentExceeds.Reason {
					t.Fatalf("取消应因余额不足被拒绝: %v", err)
				}
				requireAdjustmentState(f, increase.ID, "CONFIRMED", confirmedIncrease.Version, false)
				afterAudit, queryErr := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
				if queryErr != nil || afterAudit != beforeAudit {
					t.Fatalf("拒绝取消后产生了审计写入: before=%d after=%d err=%v", beforeAudit, afterAudit, queryErr)
				}
				// 先取消依赖冲减后，增提可正常取消，避免死锁业务流程。
				if _, err := uc.CancelAdjustment(ctx, f.organizationID, f.approverID, decrease.ID, confirmedDecrease.Version, "先撤销依赖冲减"); err != nil {
					t.Fatalf("取消冲减: %v", err)
				}
				if _, err := uc.CancelAdjustment(ctx, f.organizationID, f.approverID, increase.ID, confirmedIncrease.Version, "冲减已撤销"); err != nil {
					t.Fatalf("解除依赖后取消增提: %v", err)
				}
			} else if err != nil {
				t.Fatalf("安全取消增提被拒绝: %v", err)
			}
			requireAdjustmentState(f, increase.ID, "CANCELLED", confirmedIncrease.Version+1, true)
			storedParent, err := f.data.db.FinanceCommission.Get(ctx, parent.ID)
			if err != nil || storedParent.CommissionAmount != parent.CommissionAmount {
				t.Fatalf("调整取消改写了原始提成金额: parent=%v err=%v", storedParent, err)
			}
		})
	}

	t.Run("旧版本及空原因不能取消已确认增提", func(t *testing.T) {
		f.setT(t)
		parent := insertTwoLineCommission(f, orderA, orderB, "100", "0")
		increase := insertAdjustmentRow(f, orderA, parent, commissionadjustmentent.DirectionINCREASE, commissionadjustmentent.StatusDRAFT, "100", "cancel-invalid-"+parent.ID.String(), "手工增提")
		confirmed, err := uc.ConfirmAdjustment(ctx, f.organizationID, f.approverID, increase.ID, increase.Version)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := uc.CancelAdjustment(ctx, f.organizationID, f.approverID, increase.ID, increase.Version, "旧版本"); err == nil || kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentTransition.Reason {
			t.Fatalf("旧版本取消应拒绝: %v", err)
		}
		if _, err := uc.CancelAdjustment(ctx, f.organizationID, f.approverID, increase.ID, confirmed.Version, " "); err == nil || kratoserrors.FromError(err).Reason != biz.ErrCommissionAdjustmentInvalid.Reason {
			t.Fatalf("空原因取消应拒绝: %v", err)
		}
		requireAdjustmentState(f, increase.ID, "CONFIRMED", confirmed.Version, false)
	})
}
