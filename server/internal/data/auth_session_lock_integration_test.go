package data

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type authSessionLockProbeKey struct{}

// 真实轮转必须先等待授权写入的用户锁，不能先占住组织或成员再等待会话外键。
// 查询探针只协调交错时序，不替换实际 SQL、事务或 PostgreSQL 行锁。
func TestAuthRepoRotateSessionAuthorizationLockOrderPostgres(t *testing.T) {
	for _, target := range []string{"组织", "成员"} {
		t.Run(target, func(t *testing.T) {
			f := newAuthOrganizationSwitchFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			currentHash, nextHash := "lock-current-"+f.suffix, "lock-next-"+f.suffix
			f.createSession(ctx, currentHash, f.systemWorkspace)
			querying := make(chan struct{})
			var once sync.Once
			marker := new(int)
			notify := func(ctx context.Context) {
				if ctx.Value(authSessionLockProbeKey{}) == marker {
					once.Do(func() { close(querying) })
				}
			}
			f.data.db.User.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
					notify(ctx)
					return next.Query(ctx, query)
				})
			}))
			childProbe := ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
					value, err := next.Query(ctx, query)
					if err == nil {
						notify(ctx)
					}
					return value, err
				})
			})
			if target == "组织" {
				f.data.db.Organization.Intercept(childProbe)
			} else {
				f.data.db.Membership.Intercept(childProbe)
			}
			userLocked := make(chan struct{})
			done := make(chan error, 2)
			go func() {
				done <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
					if _, err := tx.User.Query().Where(user.IDEQ(f.userID)).ForUpdate().Only(ctx); err != nil {
						return err
					}
					close(userLocked)
					select {
					case <-querying:
					case <-ctx.Done():
						return ctx.Err()
					}
					if target == "组织" {
						_, err := tx.Organization.Query().Where(organization.IDEQ(f.branch)).ForUpdate().Only(ctx)
						return err
					}
					_, err := tx.Membership.Query().Where(membership.UserIDEQ(f.userID), membership.OrganizationIDEQ(f.branch)).ForUpdate().Only(ctx)
					return err
				})
			}()
			go func() {
				select {
				case <-userLocked:
				case <-ctx.Done():
					done <- ctx.Err()
					return
				}
				done <- f.repo.RotateSession(context.WithValue(ctx, authSessionLockProbeKey{}, marker), currentHash,
					&biz.Session{TokenHash: nextHash, UserID: f.userID, OrganizationID: f.branch, ExpiresAt: f.sessionExpiry},
					time.Now().UTC(), &biz.AuditEvent{OrganizationID: &f.branch, UserID: &f.userID, Action: "auth.organization.switch", Result: "success"})
			}()
			for range 2 {
				if err := <-done; err != nil {
					t.Errorf("授权写入与切换必须均完成，不能反向等待: %v", err)
				}
			}
			if _, err := f.repo.FindSession(ctx, nextHash, time.Now().UTC()); err != nil {
				t.Fatalf("切换后的会话应可用: %v", err)
			}
		})
	}
}

func TestAuthRepoRotateSessionRejectsInactiveAccountPostgres(t *testing.T) {
	for _, status := range []string{"停用", "待分配"} {
		t.Run(status, func(t *testing.T) {
			f := newAuthOrganizationSwitchFixture(t)
			ctx := context.Background()
			currentHash, nextHash := "inactive-current-"+f.suffix, "inactive-next-"+f.suffix
			f.createSession(ctx, currentHash, f.systemWorkspace)
			update := f.data.db.User.UpdateOneID(f.userID)
			if status == "停用" {
				update.SetEnabled(false)
			} else {
				update.SetEnabled(false).SetPendingAssignment(true)
			}
			if _, err := update.Save(ctx); err != nil {
				t.Fatal(err)
			}
			err := f.repo.RotateSession(ctx, currentHash,
				&biz.Session{TokenHash: nextHash, UserID: f.userID, OrganizationID: f.branch, ExpiresAt: f.sessionExpiry},
				time.Now().UTC(), &biz.AuditEvent{OrganizationID: &f.branch, UserID: &f.userID, Action: "auth.organization.switch", Result: "success"})
			if err != biz.ErrSessionExpired {
				t.Fatalf("无效账号应拒绝轮转，得到 %v", err)
			}
			if exists, err := f.data.db.Session.Query().Count(ctx); err != nil || exists != 1 {
				t.Fatalf("拒绝后不应新建会话: count=%d error=%v", exists, err)
			}
			current, err := f.data.db.Session.Query().Where(session.TokenHashEQ(currentHash)).Only(ctx)
			if err != nil || current.RevokedAt != nil {
				t.Fatalf("拒绝后不应撤销原会话: session=%+v error=%v", current, err)
			}
		})
	}
}
