package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// 月申持有员工快照锁而方案持有 Membership 时，分配插入的 User 外键
// KEY SHARE 必须可完成；月申重入员工锁也不能升级到 FOR UPDATE 导致隐式死锁。
func TestCommissionEmployeeSnapshotForeignKeyLockPostgres(t *testing.T) {
	f := newCommissionNettingPostgresFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userReady := make(chan struct{})
	membersReady := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if _, err := commissionEmployeeSnapshotLock(tx.User.Query().Where(userent.IDEQ(f.actorID))).Only(ctx); err != nil {
				return err
			}
			close(userReady)
			select {
			case <-membersReady:
			case <-ctx.Done():
				return ctx.Err()
			}
			return lockApplicationMembership(ctx, tx.Client(), f.organizationID, f.actorID)
		})
	}()
	go func() {
		select {
		case <-userReady:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
		done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if _, err := lockCommissionRuleMemberships(ctx, tx, f.organizationID, []uuid.UUID{f.actorID}); err != nil {
				return err
			}
			close(membersReady)
			_, err := tx.FinanceCommissionRuleAssignment.Create().SetOrganizationID(f.organizationID).SetRuleID(f.ruleID).SetEmployeeID(f.actorID).SetEffectiveFrom("2027-01-01").SetCreatedBy(f.employeeID).Save(ctx)
			return err
		})
	}()
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("员工快照锁应允许方案分配外键完成: %v", err)
		}
	}
}
