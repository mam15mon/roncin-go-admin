package data

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	attributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
)

// 同一个跨票来源的两个员工各只归属一票：首次锁必须覆盖来源全部订单，
// 不能先各锁自己的提成行子集，再同时追加另一票订单锁。
func TestCommissionConfirmationFullOrderSetPostgres(t *testing.T) {
	for _, action := range []string{"confirm", "batch-confirm"} {
		t.Run(action, func(t *testing.T) {
			f := newCommissionNettingPostgresFixture(t)
			ctx := context.Background()
			employee, err := f.data.db.User.Create().SetUsername("set-lock-" + f.suffix).SetDisplayName("第二归属员工").SetEnabled(true).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.data.db.Membership.Create().SetUserID(employee.ID).SetOrganizationID(f.organizationID).SetEnabled(true).Save(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := f.data.db.OrderCommissionAttribution.Delete().Where(attributionent.OrderIDEQ(f.orderIDs[1])).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := f.data.db.OrderCommissionAttribution.Create().SetOrganizationID(f.organizationID).SetOrderID(f.orderIDs[1]).SetCustomerID(f.customerID).SetSourceAssignmentID(uuid.New()).SetEmployeeID(employee.ID).SetEmployeeName(employee.DisplayName).SetPersonnelRole(attributionent.PersonnelRoleSALES).SetAttributedAt(time.Now()).Save(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := f.data.db.FinanceCommissionRuleAssignment.Create().SetOrganizationID(f.organizationID).SetRuleID(f.ruleID).SetEmployeeID(employee.ID).SetEffectiveFrom("2026-01-01").SetCreatedBy(f.actorID).Save(ctx); err != nil {
				t.Fatal(err)
			}
			u := f.newUsecase()
			parents := make([]*biz.FinanceCommission, 0, 2)
			for index, employeeID := range []uuid.UUID{f.employeeID, employee.ID} {
				created, err := u.Create(ctx, f.organizationID, f.actorID, biz.CreateCommissionInput{NettingID: f.nettingID, EmployeeID: employeeID, PersonnelRole: biz.CommissionRoleSales, IdempotencyKey: fmt.Sprintf("order-set-%d", index)})
				if err != nil {
					t.Fatal(err)
				}
				if len(created.Lines) != 1 {
					t.Fatalf("夹具每个员工应只命中一票: %d", len(created.Lines))
				}
				parents = append(parents, created)
			}
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			// 如入口曾返回单票订单锁，将两个真实事务同步到各持有子集的位置。
			// 完整集合锁返回两票，直接由 PostgreSQL 正常串行化，不设置人为等待。
			partialLocked := make(chan struct{})
			var mu sync.Mutex
			seen := make(map[any]bool)
			partialCount := 0
			f.data.db.Order.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
					result, err := next.Query(ctx, query)
					marker := ctx.Value(feeParentLockProbeKey{})
					orders, ok := result.([]*ent.Order)
					if err != nil || marker == nil || !ok {
						return result, err
					}
					mu.Lock()
					firstPartial := !seen[marker] && len(orders) == 1
					seen[marker] = true
					if firstPartial {
						partialCount++
						if partialCount == 2 {
							close(partialLocked)
						}
					}
					mu.Unlock()
					if firstPartial {
						select {
						case <-partialLocked:
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}
					return result, nil
				})
			}))
			done := make(chan error, 2)
			for _, parent := range parents {
				go func(parent *biz.FinanceCommission) {
					marked := context.WithValue(ctx, feeParentLockProbeKey{}, parent.ID)
					if action == "confirm" {
						_, err := u.Confirm(marked, f.organizationID, f.actorID, parent.ID, parent.Version)
						done <- err
						return
					}
					done <- f.data.WithTx(marked, func(tx *ent.Tx) error {
						return confirmCommissionsForApplicationApproval(marked, tx, f.organizationID, f.actorID, []uuid.UUID{parent.ID})
					})
				}(parent)
			}
			for range 2 {
				if err := <-done; err != nil {
					t.Fatalf("不同员工同来源确认须按完整集合串行成功: %v", err)
				}
			}
			// 成员关系保持原样；确认不应以变更员工授权规避并发条件。
			count, err := f.data.db.Membership.Query().Where(membershipent.UserIDIn(f.employeeID, employee.ID)).Count(ctx)
			if err != nil || count != 2 {
				t.Fatalf("员工成员关系异常: %d %v", count, err)
			}
		})
	}
}
