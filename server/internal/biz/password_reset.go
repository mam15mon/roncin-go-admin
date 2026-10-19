package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

var (
	ErrAuthAccountConflict   = errors.Conflict("AUTH_ACCOUNT_CONFLICT", "账号信息存在冲突，请使用钉钉扫码或联系管理员")
	ErrPasswordResetMobile   = errors.BadRequest("AUTH_PASSWORD_RESET_MOBILE", "请输入有效的钉钉手机号")
	ErrPasswordResetInvalid  = errors.BadRequest("AUTH_PASSWORD_RESET_INVALID", "验证码无效或已过期，请重新获取")
	ErrPasswordResetSend     = errors.ServiceUnavailable("AUTH_PASSWORD_RESET_SEND_FAILED", "暂时无法发送验证码，请稍后重试或使用钉钉扫码")
	ErrPasswordResetPassword = errors.BadRequest("AUTH_PASSWORD_RESET_PASSWORD", "密码去除首尾空白后至少需要 12 位")
)

const (
	PasswordResetTTL            = 5 * time.Minute
	PasswordResetResendInterval = time.Minute
)

// PasswordResetRecord 是本次找回的最小状态，不包含原始手机号、验证码或令牌。
type PasswordResetRecord struct {
	MobileKeyHash, TokenHash, CodeHash string
	UserID                             *uuid.UUID
	DingTalkUserID, DingTalkUnionID    string
	ExpiresAt, ResendAvailableAt       time.Time
}
type PasswordResetCodeResult struct {
	ChallengeToken                           string
	ExpiresAt, ResendAvailableAt, ServerTime time.Time
}
type PasswordResetRepo interface {
	ReservePasswordReset(context.Context, string, *PasswordResetRecord) error
	CompletePasswordResetSend(context.Context, *PasswordResetRecord, bool) (bool, error)
	ConsumePasswordReset(context.Context, string, string, func() (string, error)) (bool, error)
	RecordPasswordResetSendIP(context.Context, string, time.Time) (bool, error)
}
type PasswordResetUsecase struct {
	repo   PasswordResetRepo
	sender DingTalkNotificationSender
}

func NewPasswordResetUsecase(repo PasswordResetRepo, sender DingTalkNotificationSender) *PasswordResetUsecase {
	return &PasswordResetUsecase{repo: repo, sender: sender}
}
func PasswordResetMobileKey(mobile string) string {
	return hashLoginRateLimitKey("password-reset-mobile", mobile)
}
func PasswordResetCodeHash(token, code string) string {
	sum := sha256.Sum256([]byte("password-reset-code:" + token + ":" + code))
	return hex.EncodeToString(sum[:])
}
func PasswordResetCodeMatches(expected, actual string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
func PasswordResetCooldown(now, available time.Time) error {
	return errors.New(429, "AUTH_PASSWORD_RESET_COOLDOWN", "验证码发送过于频繁，请等待倒计时结束").WithMetadata(map[string]string{
		"serverTime": now.Format(time.RFC3339Nano), "resendAvailableAt": available.Format(time.RFC3339Nano),
	})
}
func (uc *PasswordResetUsecase) RequestCode(ctx context.Context, rawMobile, ip string) (*PasswordResetCodeResult, error) {
	mobile := NormalizeDingTalkMobile(rawMobile)
	if !ValidDingTalkMobile(mobile) {
		return nil, ErrPasswordResetMobile
	}
	if !uc.sender.Enabled() {
		return nil, ErrDingTalkDisabled
	}
	now := time.Now().UTC()
	exceeded, err := uc.repo.RecordPasswordResetSendIP(ctx, hashLoginRateLimitKey("password-reset-send-ip", strings.TrimSpace(ip)), now)
	if err != nil {
		return nil, err
	}
	if exceeded {
		return nil, PasswordResetCooldown(now, now.Add(10*time.Minute))
	}
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	number, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%06d", number.Int64())
	record := &PasswordResetRecord{MobileKeyHash: PasswordResetMobileKey(mobile), TokenHash: tokenHash, CodeHash: PasswordResetCodeHash(token, code)}
	if err = uc.repo.ReservePasswordReset(ctx, mobile, record); err != nil {
		return nil, err
	}
	if record.UserID != nil {
		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		sendErr := uc.sender.SendMarkdown(sendCtx, record.DingTalkUserID, renderPasswordResetCode(code))
		cancel()
		// 即使浏览器断开，也在有界独立上下文关闭本代状态；失败保持冷却，禁止激活未知发送。
		finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		ready, finishErr := uc.repo.CompletePasswordResetSend(finishCtx, record, sendErr == nil)
		finishCancel()
		if finishErr != nil {
			return nil, ErrPasswordResetSend
		}
		if sendErr != nil || !ready {
			return nil, ErrPasswordResetSend
		}
	}
	return &PasswordResetCodeResult{ChallengeToken: token, ExpiresAt: record.ExpiresAt, ResendAvailableAt: record.ResendAvailableAt, ServerTime: time.Now().UTC()}, nil
}
func (uc *PasswordResetUsecase) Confirm(ctx context.Context, token, code, newPassword string) error {
	if len(strings.TrimSpace(newPassword)) < 12 {
		return ErrPasswordResetPassword
	}
	if len(token) != 64 || len(code) != 6 {
		return ErrPasswordResetInvalid
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return ErrPasswordResetInvalid
		}
	}
	matched, err := uc.repo.ConsumePasswordReset(ctx, hashSessionToken(token), PasswordResetCodeHash(token, code), func() (string, error) { return password.Hash(newPassword) })
	if err != nil {
		return err
	}
	if !matched {
		return ErrPasswordResetInvalid
	}
	return nil
}

// 验证码仅用于即时发送，不进入通用通知存储。
func renderPasswordResetCode(code string) DingTalkMarkdownMessage {
	return newDingTalkMarkdownMessage("Roncin 密码验证码", fmt.Sprintf("您的设置 / 重置密码验证码：\n\n**%s**\n\n有效期：**5 分钟**。\n\n请勿向他人透露。如非本人操作，请忽略本消息。", code))
}
