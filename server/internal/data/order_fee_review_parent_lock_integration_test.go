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
