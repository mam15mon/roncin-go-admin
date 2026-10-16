package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	noticeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"strings"
	"sync"
	"testing"
)

func reviewFixture(t *testing.T) (*feeSupplementPostgresFixture, *biz.OrderFeeReviewUsecase, *ent.Order, *ent.OrderFee) {
	f := newFeeSupplementFixture(t)
	permissions := []*ent.Permission{}
	for _, key := range []string{"system.finance.fee.review", "business.order.si.fee.read", "business.order.si.fee.create", "business.order.si.fee.update"} {
		p, err := f.data.db.Permission.Query().Where(permissionent.KeyEQ(key)).Only(f.ctx)
		if ent.IsNotFound(err) {
			p, err = f.data.db.Permission.Create().SetKey(key).SetName(key).SetGroup("测试").Save(f.ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		permissions = append(permissions, p)
	}
	if _, err := f.data.db.Role.UpdateOneID(f.lockRoleID).AddPermissions(permissions...).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	o := f.createOrder("REVIEW-" + f.suffix)
	fee, err := f.data.db.OrderFee.Create().SetOrderID(o.ID).SetIdempotencyKey("review-fee").SetDirection(feeent.DirectionRECEIVABLE).SetFeeCode("REVIEW").SetFeeName("审核测试收入").SetSettlementPartyID(f.customerID).SetBillingUnit("票").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-10-12").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetExpenseDate("2026-10-12").Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	return f, biz.NewOrderFeeReviewUsecase(NewOrderFeeReviewRepo(f.data)), o, fee
}
func reviewAction(data *biz.OrderFeeReview, key string) biz.FeeReviewAction {
	return biz.FeeReviewAction{OrderID: data.OrderID, ExpectedContentVersion: data.ContentVersion, ExpectedReviewVersion: data.ReviewVersion, SubmissionID: data.SubmissionID, PreviewToken: data.PreviewToken, IdempotencyKey: key, CompletenessConfirmed: true}
}
func TestOrderFeeReviewLifecyclePostgres(t *testing.T) {
	f, u, o, fee := reviewFixture(t)
	p := f.approverPrincipal()
	if err := validateApprovedOrderFees(f.ctx, f.data.db, o); err != biz.ErrFeeReviewNotApproved {
		t.Fatalf("未审建账应被拒绝: %v", err)
	}
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	submit := reviewAction(current, "submit")
	pending, err := u.Act(f.ctx, p, "SUBMIT", submit)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Status != "PENDING" || !pending.CanApprove || !pending.CanWithdraw {
		t.Fatalf("自审能力错误: %+v", pending)
	}
	if err := NewOrderFeeRepo(f.data).Remove(f.ctx, f.organizationID, o.ID, fee.ID, p.UserID, fee.Version, "", &biz.AuditEvent{Action: "test", Result: "success"}); err != biz.ErrFeeReviewFrozen {
		t.Fatalf("待审普通删除必须被阻止: %v", err)
	}
	approve := reviewAction(pending, "approve")
	approved, err := u.Act(f.ctx, p, "APPROVE", approve)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != "APPROVED" {
		t.Fatal("未批准")
	}
	if _, err := u.Act(f.ctx, p, "APPROVE", approve); err != nil {
		t.Fatal("同动作幂等重放应成功", err)
	}
	if count, err := f.data.db.OrderFeeReviewEvent.Query().Where(eventent.OrderIDEQ(o.ID), eventent.ActionEQ(eventent.ActionAPPROVE)).Count(f.ctx); err != nil || count != 1 {
		t.Fatal("重放重复批准", count, err)
	}
	if err := f.data.db.OrderFee.UpdateOne(fee).AddVersion(1).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	updated, err := f.data.db.Order.Get(f.ctx, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateApprovedOrderFees(f.ctx, f.data.db, updated); err != nil {
		t.Fatal("建账行版本不能使批准失效", err)
	}
	revoke := reviewAction(approved, "revoke")
	revoke.Comment = "更正费用"
	revoked, err := u.Act(f.ctx, p, "REVOKE", revoke)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.Status != "REVOKED" || revoked.ApprovedEventID != nil {
		t.Fatal("撤销必须清批准指针")
	}
}
func TestOrderFeeReviewConcurrentApprovePostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "submit-race"))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, key := range []string{"race1", "race2"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, err := u.Act(context.Background(), p, "APPROVE", reviewAction(pending, key))
			results <- err
		}(key)
	}
	wg.Wait()
	close(results)
	ok, conflict := 0, 0
	for err := range results {
		if err == nil {
			ok++
		} else if err == biz.ErrFeeReviewConflict {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("并发批准应只有一个成功: %d/%d", ok, conflict)
	}
}
func TestOrderFeeReviewBatchPermissionAndLossPostgres(t *testing.T) {
	f, u, o, fee := reviewFixture(t)
	p := f.approverPrincipal()
	if err := f.data.db.OrderFee.UpdateOne(fee).SetDirection(feeent.DirectionPAYABLE).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "loss-submit"))
	if err != nil {
		t.Fatal(err)
	}
	input := reviewAction(pending, "loss-approve")
	if _, err := u.Act(f.ctx, p, "APPROVE", input); err != biz.ErrFeeReviewInvalid {
		t.Fatalf("亏损空意见必须拒绝: %v", err)
	}
	input.Comment = "确认漏录收入已核查，承担本票亏损"
	unknown := input
	unknown.OrderID = uuid.New()
	unknown.IdempotencyKey = "unknown"
	results, err := u.ApproveBatch(f.ctx, p, []biz.FeeReviewAction{input, unknown})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Err != nil || results[1].Err == nil {
		t.Fatal("逐票事务应部分成功")
	}
	other := f.requesterPrincipal()
	if _, err := u.Get(f.ctx, other, o.ID); err == nil {
		t.Fatal("无费用权限不能取得利润")
	}
}

func TestOrderFeeReviewBillAdmissionAndRevokeRacePostgres(t *testing.T) {
	f, u, o, fee := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "billing-submit"))
	if err != nil {
		t.Fatal(err)
	}
	approved, err := u.Act(f.ctx, p, "APPROVE", reviewAction(pending, "billing-approve"))
	if err != nil {
		t.Fatal(err)
	}
	billRepo := NewFinanceBillRepo(f.data)
	if fees, err := billRepo.LoadBillableFees(f.ctx, f.organizationID, []uuid.UUID{fee.ID}); err != nil || len(fees) != 1 {
		t.Fatalf("已审可选费用: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		results <- f.data.WithinTransaction(context.Background(), func(ctx context.Context) error {
			_, err := billRepo.LoadBillableFees(ctx, f.organizationID, []uuid.UUID{fee.ID})
			if err != nil {
				return err
			}
			client, err := f.data.client(ctx)
			if err != nil {
				return err
			}
			bill, err := client.FinanceBill.Create().SetOrganizationID(f.organizationID).SetBillNo("RACE-" + f.suffix).SetIdempotencyKey("race-bill").SetDirection("RECEIVABLE").SetSettlementPartyID(f.customerID).SetSettlementPartyName("测试客户").SetSettlementAccountID(uuid.New()).SetSettlementAccountName("测试账户").SetSettlementAccountHolder("测试客户").SetSettlementBankName("测试银行").SetSettlementBankAccount("测试号码").SetSettlementAccountCurrency("CNY").SetCurrency("CNY").SetBaseCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource("SYSTEM").SetExchangeRateDate("2026-10-12").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").SetFeeCount(1).SetBillDate("2026-10-12").Save(ctx)
			if err != nil {
				return err
			}
			return client.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderFeeID(fee.ID).SetOrderID(o.ID).SetOrderNo(o.OrderNo).SetFeeCode(fee.FeeCode).SetFeeName(fee.FeeName).SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").Exec(ctx)
		})
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		input := reviewAction(approved, "revoke-bill-race")
		input.Comment = "纠错"
		_, err := u.Act(context.Background(), p, "REVOKE", input)
		results <- err
	}()
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil && err != biz.ErrFeeReviewNotApproved && err != biz.ErrOrderFeeBillOccupied {
			t.Fatal(err)
		}
	}
	latest, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Status == "REVOKED" {
		if _, err := billRepo.LoadBillableFees(f.ctx, f.organizationID, []uuid.UUID{fee.ID}); err != biz.ErrFeeReviewNotApproved {
			t.Fatal("撤销后不能建账", err)
		}
	} else if latest.Status == "APPROVED" {
		input := reviewAction(latest, "post-race-revoke")
		input.Comment = "纠错"
		if _, err := u.Act(f.ctx, p, "REVOKE", input); err != biz.ErrOrderFeeBillOccupied {
			t.Fatal("建账已成功不能再撤销批准", err)
		}
	} else {
		t.Fatal("并发结果非法", latest.Status)
	}
}
func TestOrderFeeReviewFiltersAndNotificationsPostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "notify-submit"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := NewOrderRepo(f.data).List(f.ctx, []biz.OrderOrganizationScope{{OrganizationIDs: []uuid.UUID{f.organizationID}, BusinessType: biz.OrderBusinessSI}}, biz.OrderListOptions{Page: 1, PageSize: 20, BusinessType: biz.OrderBusinessSI, FeeReviewStatus: "PENDING"})
	if err != nil {
		t.Fatal(err)
	}
	if items.Total != 1 || items.Items[0].ID != o.ID {
		t.Fatalf("审核过滤与总数不一致 total=%d items=%+v", items.Total, items.Items)
	}
	if _, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "notify-submit")); err != nil {
		t.Fatal(err)
	}
	count, err := f.data.db.NotificationDelivery.Query().Where(noticeent.ResourceIDEQ(o.ID), noticeent.TemplateEQ(noticeent.TemplateFEE_REVIEW_PENDING)).Count(f.ctx)
	if err != nil || count != 1 {
		t.Fatalf("重放通知应唯一: %d %v", count, err)
	}
	if _, err := u.Act(f.ctx, p, "APPROVE", reviewAction(pending, "notify-approve")); err != nil {
		t.Fatal(err)
	}
	count, err = f.data.db.NotificationDelivery.Query().Where(noticeent.ResourceIDEQ(o.ID), noticeent.TemplateEQ(noticeent.TemplateFEE_REVIEW_RESULT)).Count(f.ctx)
	if err != nil || count != 1 {
		t.Fatalf("审批结果通知应唯一: %d %v", count, err)
	}

	// 真实投递存储不为尚未注册的 SI 类型生成伪入口。
	notices, err := f.data.db.NotificationDelivery.Query().Where(noticeent.ResourceIDEQ(o.ID)).All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, notice := range notices {
		if strings.Contains(notice.Parameter, "/orders/") {
			t.Fatal("未注册SI类型不应生成链接", notice.Parameter)
		}
	}
}

func TestOrderFeeReviewGrantScopePostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "scope-submit"))
	if err != nil {
		t.Fatal(err)
	}
	unrelated, err := f.data.db.Organization.Create().SetCode("OTHER-" + f.suffix).SetName("无关组织").SetKind("company").SetBaseCurrency("CNY").Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	membership, err := f.data.db.Membership.Query().Where(membershipent.UserIDEQ(p.UserID)).Only(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.data.db.Membership.UpdateOne(membership).SetOrganizationID(unrelated.ID).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Act(f.ctx, p, "APPROVE", reviewAction(pending, "scope-wrong")); err != biz.ErrFeeReviewForbidden {
		t.Fatalf("同组织grant失效必须拒绝: %v", err)
	}
	if err := f.data.db.Membership.UpdateOne(membership).SetOrganizationID(f.organizationID).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Act(f.ctx, p, "APPROVE", reviewAction(pending, "scope-restored")); err != nil {
		t.Fatal("恢复本公司成员关系后应可审核", err)
	}
	if err := f.data.db.User.UpdateOneID(p.UserID).SetEnabled(false).Exec(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Get(f.ctx, p, o.ID); err != biz.ErrFeeReviewForbidden {
		t.Fatalf("停用用户不能读取: %v", err)
	}
}

func TestFeeReviewNotificationOrderEntry(t *testing.T) {
	id := uuid.New()
	expected := "\n订单入口：/orders/sea-export/" + id.String() + "/fees"
	if actual := feeReviewOrderEntry(&ent.Order{ID: id, BusinessType: "SE"}); actual != expected {
		t.Fatalf("海运出口入口错误: %q", actual)
	}
	for _, kind := range []string{"SI", "AE", "AI", "LAND", "RAIL"} {
		if actual := feeReviewOrderEntry(&ent.Order{ID: id, BusinessType: orderent.BusinessType(kind)}); actual != "" {
			t.Fatalf("未注册类型%s不应生成入口: %q", kind, actual)
		}
	}
}
