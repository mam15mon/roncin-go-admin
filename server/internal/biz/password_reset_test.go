package biz

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type passwordResetRepoStub struct {
	record           *PasswordResetRecord
	eligible         bool
	completeAccepted bool
	invalid          bool
	consumeCalls     int
}

func (s *passwordResetRepoStub) ReservePasswordReset(_ context.Context, _ string, r *PasswordResetRecord, _ time.Time) error {
	s.record = r
	if s.eligible {
		id := uuid.New()
		r.UserID = &id
		r.DingTalkUserID = "bound-user"
	}
	return nil
}
func (s *passwordResetRepoStub) CompletePasswordResetSend(_ context.Context, _ *PasswordResetRecord, accepted bool, _ time.Time) (bool, error) {
	s.completeAccepted = accepted
	return accepted, nil
}
func (s *passwordResetRepoStub) ConsumePasswordReset(_ context.Context, tokenHash, codeHash string, hash func() (string, error), _ time.Time) (bool, error) {
	s.consumeCalls++
	if s.invalid {
		return false, nil
	}
	_, err := hash()
	return err == nil, err
}
func (s *passwordResetRepoStub) RecordPasswordResetSendIP(context.Context, string, time.Time) (bool, error) {
	return false, nil
}
func TestPasswordResetUsecaseBoundMessageAndRandomChallenge(t *testing.T) {
	repo := &passwordResetRepoStub{eligible: true}
	sender := &dingTalkNotificationSenderStub{enabled: true}
	uc := NewPasswordResetUsecase(repo, sender)
	result, err := uc.RequestCode(context.Background(), "+86 138-0013-8000", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ChallengeToken) != 64 || result.ChallengeToken == repo.record.TokenHash || sender.userID != "bound-user" {
		t.Fatal("令牌/绑定错误")
	}
	code := regexp.MustCompile(`验证码：(\d{6})`).FindStringSubmatch(sender.content)
	if len(code) != 2 || repo.record.CodeHash != PasswordResetCodeHash(result.ChallengeToken, code[1]) {
		t.Fatal("验证码格式或摘要错误")
	}
	if repo.record.ExpiresAt.Sub(repo.record.ResendAvailableAt) != 4*time.Minute || !repo.completeAccepted {
		t.Fatal("时限/发送结果错误")
	}
	if strings.Contains(repo.record.CodeHash, code[1]) {
		t.Fatal("持久化了验证码")
	}
}
func TestPasswordResetUsecaseUnknownAndFailedMessages(t *testing.T) {
	for _, scenario := range []string{"未知", "禁用消息", "发送失败"} {
		t.Run(scenario, func(t *testing.T) {
			repo := &passwordResetRepoStub{eligible: scenario != "未知"}
			sender := &dingTalkNotificationSenderStub{enabled: scenario != "禁用消息"}
			if scenario == "发送失败" {
				sender.err = errors.New("模拟超时")
			}
			uc := NewPasswordResetUsecase(repo, sender)
			result, err := uc.RequestCode(context.Background(), "13800138000", "127.0.0.1")
			switch scenario {
			case "未知":
				if err != nil || result == nil || sender.content != "" {
					t.Fatal("未知号码泄漏发送目标", err)
				}
			case "禁用消息":
				if !errors.Is(err, ErrDingTalkDisabled) {
					t.Fatal(err)
				}
			case "发送失败":
				if !errors.Is(err, ErrPasswordResetSend) || repo.completeAccepted {
					t.Fatal("失败消息被激活", err)
				}
			}
		})
	}
}
func TestPasswordResetUsecaseBoundary(t *testing.T) {
	repo := &passwordResetRepoStub{invalid: true}
	uc := NewPasswordResetUsecase(repo, &dingTalkNotificationSenderStub{enabled: true})
	if _, err := uc.RequestCode(context.Background(), "invalid", ""); !errors.Is(err, ErrPasswordResetMobile) {
		t.Fatal(err)
	}
	for _, item := range []struct{ token, code, password string }{{"bad", "012345", "long-enough-password"}, {strings.Repeat("a", 64), "abc123", "long-enough-password"}, {strings.Repeat("a", 64), "012345", "short"}} {
		if err := uc.Confirm(context.Background(), item.token, item.code, item.password); err == nil {
			t.Fatal("无效输入通过")
		}
	}
	if repo.consumeCalls != 0 {
		t.Fatal("边界错误仍进入消费")
	}
	if err := uc.Confirm(context.Background(), strings.Repeat("a", 64), "012345", "long-enough-password"); !errors.Is(err, ErrPasswordResetInvalid) {
		t.Fatal(err)
	}
}
