package data

import (
	"context"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
)

func TestAuthRepoRenewSessionPostgres(t *testing.T) {
	f := newAuthOrganizationSwitchFixture(t)
	ctx := context.Background()
	hash := "renew-" + f.suffix
	f.createSession(ctx, hash, f.systemWorkspace)
	now := time.Now().UTC().Truncate(time.Microsecond)
	expiry := now.Add(30 * 24 * time.Hour)
	if err := f.repo.RenewSession(ctx, hash, now, expiry); err != nil {
		t.Fatal(err)
	}
	stored, err := f.repo.FindSession(ctx, hash, now)
	if err != nil || !stored.ExpiresAt.Equal(expiry) {
		t.Fatalf("续期未落库: %+v %v", stored, err)
	}
	// 迟到的较早请求不得缩短窗口。
	if err := f.repo.RenewSession(ctx, hash, now.Add(-time.Minute), expiry.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	stored, err = f.repo.FindSession(ctx, hash, now)
	if err != nil || !stored.ExpiresAt.Equal(expiry) {
		t.Fatalf("乱序请求缩短窗口: %+v %v", stored, err)
	}
	if err := f.repo.RevokeSession(ctx, hash, now, &biz.AuditEvent{OrganizationID: &f.systemWorkspace, UserID: &f.userID, Action: "auth.logout", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.RenewSession(ctx, hash, now, expiry.Add(time.Hour)); err != biz.ErrSessionExpired {
		t.Fatalf("已退出会话续活: %v", err)
	}
	for _, tc := range []struct {
		name   string
		offset time.Duration
	}{{"恰好过期", 0}, {"已过期", -time.Second}} {
		t.Run(tc.name, func(t *testing.T) {
			h := hash + tc.name
			f.createSession(ctx, h, f.systemWorkspace)
			if err := f.data.db.Session.Update().Where(session.TokenHashEQ(h)).SetExpiresAt(now.Add(tc.offset)).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			if err := f.repo.RenewSession(ctx, h, now, expiry); err != biz.ErrSessionExpired {
				t.Fatalf("过期会话续活: %v", err)
			}
		})
	}
	for _, state := range []string{"账号停用", "账号待分配", "组织停用"} {
		t.Run(state, func(t *testing.T) {
			h := hash + state
			f.createSession(ctx, h, f.systemWorkspace)
			switch state {
			case "账号停用":
				if err := f.data.db.User.UpdateOneID(f.userID).SetEnabled(false).Exec(ctx); err != nil {
					t.Fatal(err)
				}
			case "账号待分配":
				if err := f.data.db.User.UpdateOneID(f.userID).SetEnabled(false).SetPendingAssignment(true).Exec(ctx); err != nil {
					t.Fatal(err)
				}
			case "组织停用":
				if err := f.data.db.Organization.UpdateOneID(f.systemWorkspace).SetEnabled(false).Exec(ctx); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.repo.RenewSession(ctx, h, now, expiry); err != biz.ErrSessionExpired {
				t.Fatalf("不可用身份仍续期: %v", err)
			}
			if err := f.data.db.User.UpdateOneID(f.userID).SetEnabled(true).SetPendingAssignment(false).Exec(ctx); err != nil {
				t.Fatal(err)
			}
			if err := f.data.db.Organization.UpdateOneID(f.systemWorkspace).SetEnabled(true).Exec(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}

}

func TestAuthRepoRenewSessionAfterOrganizationSwitchPostgres(t *testing.T) {
	f := newAuthOrganizationSwitchFixture(t)
	ctx := context.Background()
	oldHash, nextHash := "renew-old-"+f.suffix, "renew-next-"+f.suffix
	f.createSession(ctx, oldHash, f.systemWorkspace)
	now := time.Now().UTC()
	next := &biz.Session{TokenHash: nextHash, UserID: f.userID, OrganizationID: f.branch, ExpiresAt: now.Add(30 * 24 * time.Hour)}
	if err := f.repo.RotateSession(ctx, oldHash, next, now, &biz.AuditEvent{OrganizationID: &f.branch, UserID: &f.userID, Action: "auth.switch_organization", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	if err := f.repo.RenewSession(ctx, oldHash, now, next.ExpiresAt); err != biz.ErrSessionExpired {
		t.Fatalf("切换后的旧会话续活: %v", err)
	}
	if err := f.repo.RenewSession(ctx, nextHash, now, next.ExpiresAt); err != nil {
		t.Fatal(err)
	}
}

func TestAuthRepoRenewSessionConcurrentRevokePostgres(t *testing.T) {
	f := newAuthOrganizationSwitchFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hash := "renew-concurrent-" + f.suffix
	f.createSession(ctx, hash, f.systemWorkspace)
	start := make(chan struct{})
	results := make(chan error, 16)
	for range 16 {
		go func() {
			<-start
			now := time.Now().UTC()
			results <- f.repo.RenewSession(ctx, hash, now, now.Add(30*24*time.Hour))
		}()
	}
	close(start)
	if err := f.repo.RevokeSession(ctx, hash, time.Now().UTC(), &biz.AuditEvent{OrganizationID: &f.systemWorkspace, UserID: &f.userID, Action: "auth.logout", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	for range 16 {
		if err := <-results; err != nil && err != biz.ErrSessionExpired {
			t.Fatal(err)
		}
	}
	if _, err := f.repo.FindSession(ctx, hash, time.Now().UTC()); err != biz.ErrSessionExpired {
		t.Fatalf("并发续期恢复已撤销会话: %v", err)
	}
}
