package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type feeParentLockProbeKey struct{}

// observeFeeParentQuery 仅同步测试中的目标请求，实际 SQL、事务与锁仍由生产代码执行。
func observeFeeParentQuery(client *ent.Client, marker any) <-chan struct{} {
	queried := make(chan struct{})
	var once sync.Once
	client.User.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
			if ctx.Value(feeParentLockProbeKey{}) == marker {
				once.Do(func() { close(queried) })
			}
			return next.Query(ctx, query)
		})
	}))
	return queried
}

// 月申已经持有 User/Membership 时，审核必须先等待授权父锁，不能先占住 Order。
func TestOrderFeeReviewMonthlyParentLockOrderPostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	marker := new(int)
	querying := observeFeeParentQuery(f.data.db, marker)
	parentsLocked := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if err := lockApplicationMembership(ctx, tx.Client(), f.organizationID, p.UserID); err != nil {
				return err
			}
			close(parentsLocked)
			select {
			case <-querying:
			case <-ctx.Done():
				return ctx.Err()
			}
			return lockOrdersSortedForFinance(ctx, tx, []uuid.UUID{o.ID})
		})
	}()
	go func() {
		select {
		case <-parentsLocked:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
		_, err := u.Act(context.WithValue(ctx, feeParentLockProbeKey{}, marker), p, "SUBMIT", reviewAction(current, "monthly-parent-order"))
		done <- err
	}()
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("审核与月申应按父行先于订单完成，不能死锁: %v", err)
		}
	}
}

// 审核持有 User SHARE 后，月申不能先占 Membership 再等待 User UPDATE。
func TestOrderFeeReviewMonthlyMembershipLockOrderPostgres(t *testing.T) {
	f, _, o, _ := reviewFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	marker := new(int)
	querying := observeFeeParentQuery(f.data.db, marker)
	userLocked := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if _, err := tx.User.Query().Where(userent.IDEQ(f.approverID)).ForShare().Only(ctx); err != nil {
				return err
			}
			close(userLocked)
			select {
			case <-querying:
			case <-ctx.Done():
				return ctx.Err()
			}
			qualified, err := hasFeeReviewPermission(ctx, tx.Client(), f.organizationID, f.approverID, true)
			if err != nil {
				return err
			}
			if !qualified {
				return biz.ErrFeeReviewForbidden
			}
			return lockOrdersSortedForFinance(ctx, tx, []uuid.UUID{o.ID})
		})
	}()
	go func() {
		select {
		case <-userLocked:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
		done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			return lockApplicationMembership(context.WithValue(ctx, feeParentLockProbeKey{}, marker), tx.Client(), f.organizationID, f.approverID)
		})
	}()
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("User/Membership不能反序等待: %v", err)
		}
	}
}

// 账号停用先取得 User 锁时，审核等待提交后必须以实时资格拒绝，不沿用已读权限。
func TestOrderFeeReviewConcurrentUserDisablePostgres(t *testing.T) {
	f, u, o, _ := reviewFixture(t)
	p := f.approverPrincipal()
	current, err := u.Get(f.ctx, p, o.ID)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := u.Act(f.ctx, p, "SUBMIT", reviewAction(current, "disable-parent-submit"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	marker := new(int)
	querying := observeFeeParentQuery(f.data.db, marker)
	userLocked := make(chan struct{})
	disabled := make(chan error, 1)
	go func() {
		disabled <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if _, err := tx.User.Query().Where(userent.IDEQ(p.UserID)).ForUpdate().Only(ctx); err != nil {
				return err
			}
			if err := tx.User.UpdateOneID(p.UserID).SetEnabled(false).Exec(ctx); err != nil {
				return err
			}
			close(userLocked)
			select {
			case <-querying:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-userLocked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	_, err = u.Act(context.WithValue(ctx, feeParentLockProbeKey{}, marker), p, "APPROVE", reviewAction(pending, "disable-parent-approve"))
	if err != biz.ErrFeeReviewForbidden {
		t.Fatalf("并发停用账号后不能批准: %v", err)
	}
	if err := <-disabled; err != nil {
		t.Fatal(err)
	}
	latest, err := f.data.db.Order.Get(ctx, o.ID)
	if err != nil || string(latest.FeeReviewStatus) != "PENDING" {
		t.Fatalf("无效审核不能推进费用包: order=%v err=%v", latest, err)
	}
}

// runWhileMonthlyParentsLocked 将真实业务入口暂停在首次员工查询，让已持有
// User/Membership 的月申继续锁 Order；业务入口若先锁 Order 就会形成数据库死锁。
func runWhileMonthlyParentsLocked(t *testing.T, data *Data, organizationID, userID uuid.UUID, orderIDs []uuid.UUID, operation func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	marker := new(int)
	querying := observeFeeParentQuery(data.db, marker)
	parentsLocked := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- data.WithTx(ctx, func(tx *ent.Tx) error {
			if err := lockApplicationMembership(ctx, tx.Client(), organizationID, userID); err != nil {
				return err
			}
			close(parentsLocked)
			select {
			case <-querying:
			case <-ctx.Done():
				return ctx.Err()
			}
			return lockOrdersSortedForFinance(ctx, tx, orderIDs)
		})
	}()
	go func() {
		select {
		case <-parentsLocked:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
		done <- operation(context.WithValue(ctx, feeParentLockProbeKey{}, marker))
	}()
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("业务入口与月申授权父行/订单锁序应一致: %v", err)
		}
	}
}

func TestFeeSupplementMonthlyParentLockOrderPostgres(t *testing.T) {
	for _, action := range []string{"preview", "approve", "reject", "cancel"} {
		t.Run(action, func(t *testing.T) {
			f := newFeeSupplementFixture(t)
			o := f.createOrder("PARENT-" + f.suffix)
			f.approveInitialFees(o)
			r := f.createSupplement(o, "parent-"+action)
			var preview *biz.OrderFeeSupplementApprovalPreview
			var approved *biz.OrderFeeSupplementApproveResult
			var err error
			if action == "approve" {
				preview, err = f.usecase.PreviewApproval(f.ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
				if err != nil {
					t.Fatal(err)
				}
			}
			if action == "cancel" {
				approved = f.approveSupplement(o, r.ID, r.Version)
			}
			runWhileMonthlyParentsLocked(t, f.data, f.organizationID, f.approverID, []uuid.UUID{o.ID}, func(ctx context.Context) error {
				switch action {
				case "preview":
					_, err := f.usecase.PreviewApproval(ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version)
					return err
				case "approve":
					_, err := f.usecase.Approve(ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, preview.PreviewToken, "成本已核实")
					return err
				case "reject":
					reason := "费用需更正"
					_, err := f.usecase.Reject(ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, r.Version, &reason)
					return err
				default:
					_, err := f.usecase.CancelApprovedFee(ctx, f.approverPrincipal(), f.organizationID, o.ID, r.ID, approved.Fee.Version, "重复成本")
					return err
				}
			})
		})
	}
}

func TestCommissionMonthlyParentLockOrderPostgres(t *testing.T) {
	for _, action := range []string{"create", "confirm", "batch-confirm"} {
		t.Run(action, func(t *testing.T) {
			f := newCommissionNettingPostgresFixture(t)
			u := f.newUsecase()
			var created *biz.FinanceCommission
			var err error
			input := biz.CreateCommissionInput{NettingID: f.nettingID, EmployeeID: f.employeeID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: "parent-" + action}
			if action != "create" {
				created, err = u.Create(context.Background(), f.organizationID, f.actorID, input)
				if err != nil {
					t.Fatal(err)
				}
			}
			runWhileMonthlyParentsLocked(t, f.data, f.organizationID, f.employeeID, f.orderIDs, func(ctx context.Context) error {
				switch action {
				case "create":
					_, err := u.Create(ctx, f.organizationID, f.actorID, input)
					return err
				case "confirm":
					_, err := u.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
					return err
				default:
					return f.data.WithTx(ctx, func(tx *ent.Tx) error {
						return confirmCommissionsForApplicationApproval(ctx, tx, f.organizationID, f.actorID, []uuid.UUID{created.ID})
					})
				}
			})
		})
	}
}

// 财务批准与本人重提也须共用 User → 申请头顺序，避免头锁与员工锁互相等待。
func TestCommissionApplicationApproveParentLockOrderPostgres(t *testing.T) {
	f := newCommissionApplicationFixture(t)
	employeeID, application := f.submitSingleLineApplication(t, "parent-order")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	marker := new(int)
	querying := observeFeeParentQuery(f.data.db, marker)
	parentsLocked := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if err := lockApplicationMembership(ctx, tx.Client(), f.organizationID, employeeID); err != nil {
				return err
			}
			close(parentsLocked)
			select {
			case <-querying:
			case <-ctx.Done():
				return ctx.Err()
			}
			_, err := tx.FinanceCommissionApplication.Query().Where(applicationent.IDEQ(application.ID)).ForUpdate().Only(ctx)
			return err
		})
	}()
	go func() {
		select {
		case <-parentsLocked:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
		_, err := f.usecase().Approve(context.WithValue(ctx, feeParentLockProbeKey{}, marker), []uuid.UUID{f.organizationID}, f.actorID, application.ID, application.Version)
		done <- err
	}()
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("整单批准与本人重提父行/申请头锁序应一致: %v", err)
		}
	}
}
