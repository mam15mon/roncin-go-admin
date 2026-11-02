package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type renewalAuthRepo struct {
	authRepoStub
	session    *Session
	findError  error
	renewError error
	renewed    bool
	renewedAt  time.Time
	expiresAt  time.Time
}

func (r *renewalAuthRepo) FindSession(context.Context, string, time.Time) (*Session, error) {
	return r.session, r.findError
}
func (r *renewalAuthRepo) RenewSession(_ context.Context, _ string, now, expires time.Time) error {
	r.renewed = true
	r.renewedAt, r.expiresAt = now, expires
	return r.renewError
}

func TestAuthenticateSessionSlidingExpiry(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		findError, renewError error
	}{
		{name: "有效会话续期"},
		{name: "过期会话拒绝", findError: ErrSessionExpired},
		{name: "校验后被撤销拒绝", renewError: ErrSessionExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &renewalAuthRepo{session: &Session{UserID: uuid.New(), OrganizationID: uuid.New()}, findError: tc.findError, renewError: tc.renewError}
			ttl := 30 * 24 * time.Hour
			uc := NewAuthUsecase(repo, &SessionPolicy{TTL: ttl}, nil, nil)
			principal, err := uc.AuthenticateSession(t.Context(), "session-token")
			wantErr := tc.findError
			if wantErr == nil {
				wantErr = tc.renewError
			}
			if err != wantErr {
				t.Fatalf("鉴权错误 = %v，期望 %v", err, wantErr)
			}
			if repo.renewed != (tc.findError == nil) {
				t.Fatal("失效会话进入续期")
			}
			if repo.renewed && repo.expiresAt.Sub(repo.renewedAt) != ttl {
				t.Fatal("续期窗口不是30天")
			}
			if err == nil && (principal.SessionTokenHash != hashSessionToken("session-token") || principal.UserID != repo.session.UserID) {
				t.Fatal("续期改变会话身份")
			}
		})
	}
}
