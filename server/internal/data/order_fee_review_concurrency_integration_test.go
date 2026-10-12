package data

import (
	"context"
	"sync"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
)

// 提交与普通编辑共用订单锁，不能得到待审核但费用已经被删的费用包。
func TestOrderFeeReviewSubmitRemoveRacePostgres(t *testing.T) {
	f, u, o, fee := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var submitErr, removeErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, submitErr = u.Act(context.Background(), p, "SUBMIT", reviewAction(current, "submit-edit-race"))
	}()
	go func() {
		defer wg.Done()
		removeErr = NewOrderFeeRepo(f.data).Remove(context.Background(), f.organizationID, o.ID, fee.ID, p.UserID, fee.Version, "", &biz.AuditEvent{Action: "test.fee_review.remove", Result: "success", Details: map[string]string{}})
	}()
	wg.Wait()
	if submitErr == nil {
		if removeErr != biz.ErrFeeReviewFrozen {
			t.Fatalf("提交成功后删除应冻结: %v", removeErr)
		}
	} else if removeErr != nil || submitErr != biz.ErrFeeReviewConflict {
		t.Fatalf("删除先完成应使提交预览失效: submit=%v remove=%v", submitErr, removeErr)
	}
	latest, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	count, err := f.data.db.OrderFee.Query().Where(feeent.OrderIDEQ(o.ID)).Count(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if submitErr == nil && (string(latest.FeeReviewStatus) != "PENDING" || count != 1) {
		t.Fatalf("提交成功应保留完整费用包: status=%s fees=%d", latest.FeeReviewStatus, count)
	}
	if removeErr == nil && (string(latest.FeeReviewStatus) != "UNSUBMITTED" || count != 0) {
		t.Fatalf("删除成功不能产生待审快照: status=%s fees=%d", latest.FeeReviewStatus, count)
	}
}

func TestOrderFeeReviewDisabledOrganizationPostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "disabled-org-submit"))
	if err != nil {
		t.Fatal(err)
	}
	// all 范围也不能绕过停用组织；候选和实际决策必须一致。
	if err := f.data.db.Role.UpdateOneID(f.lockRoleID).SetDataScope(roleent.DataScopeAll).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.data.db.Organization.UpdateOneID(f.organizationID).SetEnabled(false).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Act(f.ctx, p, "APPROVE", reviewAction(pending, "disabled-org-approve")); err != biz.ErrFeeReviewForbidden {
		t.Fatalf("停用组织应阻止审批: %v", err)
	}
	if _, err := u.Get(f.ctx, p, o.ID); err != biz.ErrFeeReviewForbidden {
		t.Fatalf("停用组织应阻止读取费用: %v", err)
	}
	grants, err := feeReviewGrants(f.ctx, f.data.db, f.organizationID)
	if err != nil || len(grants) != 0 {
		t.Fatalf("停用组织不能产生提醒候选: grants=%d err=%v", len(grants), err)
	}
	if err := f.data.db.Organization.UpdateOneID(f.organizationID).SetEnabled(true).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	other, err := f.data.db.Organization.Create().SetCode("DISABLED-" + f.suffix).SetName("停用授权组织").SetKind("company").SetBaseCurrency("CNY").SetEnabled(false).Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.data.db.Membership.Update().Where(membershipent.UserIDEQ(p.UserID)).SetOrganizationID(other.ID).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Act(f.ctx, p, "APPROVE", reviewAction(pending, "disabled-grant-org-approve")); err != biz.ErrFeeReviewForbidden {
		t.Fatalf("停用成员组织不能提供跨组织 all 授权: %v", err)
	}
	grants, err = feeReviewGrants(f.ctx, f.data.db, f.organizationID)
	if err != nil || len(grants) != 0 {
		t.Fatalf("停用成员组织不能成为提醒候选: grants=%d err=%v", len(grants), err)
	}
}

// 自审也必须与本人撤回竞争，两个显式动作只能有一个推进同一版本。
func TestOrderFeeReviewApproveWithdrawRacePostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "submit-withdraw-race"))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for _, action := range []string{"APPROVE", "WITHDRAW"} {
		wg.Add(1)
		go func(action string) {
			defer wg.Done()
			_, err := u.Act(context.Background(), p, action, reviewAction(pending, "decision-"+action))
			errors <- err
		}(action)
	}
	wg.Wait()
	close(errors)
	succeeded, conflicted := 0, 0
	for err := range errors {
		switch err {
		case nil:
			succeeded++
		case biz.ErrFeeReviewConflict:
			conflicted++
		default:
			t.Fatal(err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("批准/撤回应只有一方成功: success=%d conflict=%d", succeeded, conflicted)
	}
	latest, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Status != "APPROVED" && latest.Status != "UNSUBMITTED" {
		t.Fatalf("不合法的终态: %s", latest.Status)
	}
	if (latest.Status == "APPROVED") != (latest.ApprovedEventID != nil) {
		t.Fatalf("终态与批准指针不一致: %+v", latest)
	}
}
