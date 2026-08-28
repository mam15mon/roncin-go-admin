package biz

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

type loginRateLimitRepoStub struct {
	wecomAuthRepoStub
	loginCredential *Credential
	credentialErr   error
	checkErr        error
	recordErr       error
	counts          map[string]int
	cleared         []string
}

func (s *loginRateLimitRepoStub) FindCredential(context.Context, string) (*Credential, error) {
	return s.loginCredential, s.credentialErr
}

func (s *loginRateLimitRepoStub) LoginRateLimitExceeded(_ context.Context, keyHashes []string, _ time.Time, _ time.Duration, maxAttempts int) (bool, error) {
	if s.checkErr != nil {
		return false, s.checkErr
	}
	for _, keyHash := range keyHashes {
		if s.counts[keyHash] >= maxAttempts {
			return true, nil
		}
	}
	return false, nil
}

func (s *loginRateLimitRepoStub) RecordLoginFailure(_ context.Context, keyHashes []string, _ time.Time, _ time.Duration, maxAttempts int) (bool, error) {
	if s.recordErr != nil {
		return false, s.recordErr
	}
	exceeded := false
	for _, keyHash := range keyHashes {
		s.counts[keyHash]++
		if s.counts[keyHash] > maxAttempts {
			exceeded = true
		}
	}
	return exceeded, nil
}

func (s *loginRateLimitRepoStub) ClearLoginFailures(_ context.Context, keyHash string) error {
	delete(s.counts, keyHash)
	s.cleared = append(s.cleared, keyHash)
	return nil
}

func newLoginRateLimitUsecase(repo AuthRepo) *AuthUsecase {
	return NewAuthUsecase(repo, &SessionPolicy{TTL: time.Hour}, &wecomProviderStub{}, &dingTalkProviderStub{}, &dingTalkRegistrationTokenCodecStub{})
}

func TestAuthUsecaseLoginRateLimitsSixthFailure(t *testing.T) {
	repo := &loginRateLimitRepoStub{
		loginCredential: &Credential{UserID: uuid.New()},
		counts:          make(map[string]int),
	}
	usecase := newLoginRateLimitUsecase(repo)

	for attempt := 1; attempt <= loginRateLimitMaxFailures; attempt++ {
		_, _, _, err := usecase.Login(context.Background(), " Admin ", "wrong-password", "test", "127.0.0.1")
		if err != ErrInvalidCredentials {
			t.Fatalf("第 %d 次失败返回 %v，期望 ErrInvalidCredentials", attempt, err)
		}
	}
	_, _, _, err := usecase.Login(context.Background(), "admin", "wrong-password", "test", "127.0.0.1")
	if err != ErrLoginRateLimited {
		t.Fatalf("第 6 次失败返回 %v，期望 ErrLoginRateLimited", err)
	}
}

func TestAuthUsecaseLoginRateLimitsByAccountOrIP(t *testing.T) {
	accountKey, keys := loginRateLimitKeys("admin", "127.0.0.1")
	ipKey := keys[1]
	tests := []struct {
		name   string
		counts map[string]int
	}{
		{name: "账号达到上限", counts: map[string]int{accountKey: loginRateLimitMaxFailures}},
		{name: "IP 达到上限", counts: map[string]int{ipKey: loginRateLimitMaxFailures}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &loginRateLimitRepoStub{counts: test.counts}
			_, _, _, err := newLoginRateLimitUsecase(repo).Login(context.Background(), "admin", "wrong-password", "test", "127.0.0.1")
			if err != ErrLoginRateLimited {
				t.Fatalf("Login() error = %v, want ErrLoginRateLimited", err)
			}
		})
	}
}

func TestAuthUsecaseSuccessfulLoginClearsOnlyAccountBucket(t *testing.T) {
	passwordHash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("password.Hash() error = %v", err)
	}
	organizationID := uuid.New()
	userID := uuid.New()
	accountKey, keys := loginRateLimitKeys("admin", "127.0.0.1")
	ipKey := keys[1]
	repo := &loginRateLimitRepoStub{
		loginCredential: &Credential{UserID: userID, PasswordHash: &passwordHash, Enabled: true, PrimaryOrganizationID: organizationID},
		counts:          map[string]int{accountKey: 2, ipKey: 2},
	}

	_, _, _, err = newLoginRateLimitUsecase(repo).Login(context.Background(), "admin", "correct-password", "test", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if len(repo.cleared) != 1 || repo.cleared[0] != accountKey {
		t.Fatalf("cleared = %v, want account bucket", repo.cleared)
	}
	if repo.counts[ipKey] != 2 {
		t.Fatalf("IP bucket attempts = %d, want 2", repo.counts[ipKey])
	}
}

func TestAuthUsecaseUnknownAccountRecordsFailure(t *testing.T) {
	_, keys := loginRateLimitKeys("missing", "127.0.0.1")
	repo := &loginRateLimitRepoStub{credentialErr: ErrInvalidCredentials, counts: make(map[string]int)}

	_, _, _, err := newLoginRateLimitUsecase(repo).Login(context.Background(), "missing", "wrong-password", "test", "127.0.0.1")
	if err != ErrInvalidCredentials {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	for _, key := range keys {
		if repo.counts[key] != 1 {
			t.Fatalf("bucket %s attempts = %d, want 1", key, repo.counts[key])
		}
	}
}

func TestAuthUsecaseLoginRateLimitRepoErrorIsReturned(t *testing.T) {
	wantErr := stderrors.New("rate limit storage failed")
	repo := &loginRateLimitRepoStub{checkErr: wantErr, counts: make(map[string]int)}

	_, _, _, err := newLoginRateLimitUsecase(repo).Login(context.Background(), "admin", "wrong-password", "test", "127.0.0.1")
	if !stderrors.Is(err, wantErr) {
		t.Fatalf("Login() error = %v, want %v", err, wantErr)
	}
}

var _ AuthRepo = (*loginRateLimitRepoStub)(nil)
