package data

import (
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// 使用真实成员和角色授权，证明待审不入账、普通审核人批准一次生效及初始化管理员不可代审。
func TestFeeCorrectionApprovalAcceptancePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := newCorrectionTestUsecase(t, f)
	o := f.createOrder("APPROVAL-" + f.suffix)
	f.approveInitialFees(o)
	before := f.data.db.Order.GetX(f.ctx, o.ID)
	input := correctionTestInput(f, "ADD", "approval-acceptance", nil, 100)
	input.After.Direction = biz.OrderFeeReceivable
	request, err := uc.Submit(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, input, false)
	if err != nil {
		t.Fatal(err)
	}
	assertRevenue := func(want int64, count int) {
		t.Helper()
		fees, err := NewOrderFeeCorrectionRepo(f.data).CurrentFees(f.ctx, o.ID)
		if err != nil {
			t.Fatal(err)
		}
		snapshot, _, err := biz.BuildFeeReviewSnapshot(fees)
		if err != nil || !snapshot.Summary.Receivable.Equal(decimal.NewFromInt(want)) || len(fees) != count {
			t.Fatalf("正式费用不符: want=%d count=%d snapshot=%+v err=%v", want, len(fees), snapshot, err)
		}
	}
	assertRevenue(1000, 2)
	if request.Status != "PENDING" {
		t.Fatal("申请应待审")
	}
	for _, p := range []*biz.Principal{f.requesterPrincipal()} {
		if _, err := uc.Preview(f.ctx, p, f.organizationID, o.ID, request.ID, request.Version); err == nil {
			t.Fatal("无授权用户不可审核")
		}
	}
	bootstrap := f.data.db.User.Create().SetDisplayName("隔离验收初始化管理员").SetEnabled(true).SetIsBootstrapAdmin(true).SaveX(f.ctx)
	membership := f.data.db.Membership.Create().SetUserID(bootstrap.ID).SetOrganizationID(f.organizationID).SetEnabled(true).SaveX(f.ctx)
	f.data.db.RoleAssignment.Create().SetMembershipID(membership.ID).SetRoleID(f.lockRoleID).SaveX(f.ctx)
	principal := *f.approverPrincipal()
	principal.UserID = bootstrap.ID
	if _, err := uc.Preview(f.ctx, &principal, f.organizationID, o.ID, request.ID, request.Version); err == nil {
		t.Fatal("初始化管理员即使有相同角色也不可代替普通审核人")
	}
	ctx := biz.WithPrincipal(f.ctx, f.approverPrincipal())
	view, err := NewOrderFeeCorrectionRepo(f.data).Get(ctx, f.organizationID, request.ID)
	if err != nil || !view.CanApprove {
		t.Fatalf("真实合格审核人应有审核入口: %+v %v", view, err)
	}
	preview, err := uc.Preview(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, request.ID, request.Version)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Approve(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, request.ID, request.Version+1, preview.PreviewToken, "失效版本"); err == nil {
		t.Fatal("失效版本应拒绝")
	}
	assertRevenue(1000, 2)
	approved := approveCorrectionForTest(t, f, uc, o, request)
	if approved.Status != "APPROVED" {
		t.Fatal("批准应成功")
	}
	assertRevenue(1100, 3)
	after := f.data.db.Order.GetX(f.ctx, o.ID)
	if after.LockedAt == nil || after.LockGeneration != before.LockGeneration || after.FeeContentVersion != before.FeeContentVersion+1 {
		t.Fatal("批准应保持锁及代次，仅推进费用内容版本一次")
	}
	count, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(o.ID)).Count(f.ctx)
	if err != nil || count != 3 {
		t.Fatalf("批准重放不得重复创建费用: %d %v", count, err)
	}
}
