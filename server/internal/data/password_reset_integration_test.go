package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	reset "github.com/roncin/roncin-go-admin/server/internal/data/ent/passwordresetchallenge"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/roncin/roncin-go-admin/server/internal/security/password"
)

func resetTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func setupPasswordReset(t *testing.T) (*authOrganizationSwitchFixture, *passwordResetRepo) {
	t.Helper()
	f := newAuthOrganizationSwitchFixture(t)
	if err := f.data.db.User.UpdateOneID(f.userID).SetDingtalkMobile("+86 138-0013-8000").SetDingtalkMobileNormalized("13800138000").SetDingtalkUserid("ding-reset").SetDingtalkUnionid("union-reset").Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	return f, NewPasswordResetRepo(f.data).(*passwordResetRepo)
}
func reserveReset(t *testing.T, r *passwordResetRepo) (*biz.PasswordResetRecord, string) {
	t.Helper()
	token := strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "")
	input := &biz.PasswordResetRecord{MobileKeyHash: biz.PasswordResetMobileKey("13800138000"), TokenHash: resetTokenHash(token), CodeHash: biz.PasswordResetCodeHash(token, "012345")}
	if err := r.ReservePasswordReset(context.Background(), "13800138000", input); err != nil {
		t.Fatal(err)
	}
	if input.UserID != nil {
		if ready, err := r.CompletePasswordResetSend(context.Background(), input, true); err != nil || !ready {
			t.Fatalf("激活失败 %v %v", ready, err)
		}
	}
	return input, token
}
func TestPasswordResetCooldownConcurrencyAndRestart(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	var success atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			token := uuid.NewString()
			err := r.ReservePasswordReset(ctx, "13800138000", &biz.PasswordResetRecord{MobileKeyHash: biz.PasswordResetMobileKey("13800138000"), TokenHash: resetTokenHash(token), CodeHash: "hash"})
			if err == nil {
				success.Add(1)
			} else if !strings.Contains(err.Error(), "AUTH_PASSWORD_RESET_COOLDOWN") {
				t.Errorf("并发申请 %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("成功次数 %d", success.Load())
	}
	// 新建仓储模拟进程重建，冷却仍由数据库维持。
	r = NewPasswordResetRepo(f.data).(*passwordResetRepo)
	current := f.data.db.PasswordResetChallenge.Query().OnlyX(ctx)
	if current.ExpiresAt.Sub(current.ResendAvailableAt) != biz.PasswordResetTTL-biz.PasswordResetResendInterval || time.Until(current.ResendAvailableAt) <= 0 || time.Until(current.ResendAvailableAt) > biz.PasswordResetResendInterval {
		t.Fatal("锁内未建立完整冷却与有效期")
	}
	input := &biz.PasswordResetRecord{MobileKeyHash: biz.PasswordResetMobileKey("13800138000"), TokenHash: resetTokenHash("new")}
	if err := r.ReservePasswordReset(ctx, "13800138000", input); err == nil {
		t.Fatal("60 秒前允许重发")
	}
	// 将持久化冷却置于当前边界，无需等待一分钟或向仓储注入假时钟。
	f.data.db.PasswordResetChallenge.UpdateOneID(current.ID).SetResendAvailableAt(time.Now().UTC()).ExecX(ctx)
	if err := r.ReservePasswordReset(ctx, "13800138000", input); err != nil {
		t.Fatal(err)
	}
}
func TestPasswordResetWrongCodePersistsAndFifthInvalidates(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	input, _ := reserveReset(t, r)
	for i := 1; i <= 5; i++ {
		ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, "wrong", func() (string, error) { t.Error("错误码不应计算密码哈希"); return "new-hash", nil })
		if err != nil || ok {
			t.Fatalf("错误码结果 %v %v", ok, err)
		}
		stored := f.data.db.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(input.TokenHash)).OnlyX(ctx)
		if stored.FailedAttempts != i {
			t.Fatalf("次数 %d", stored.FailedAttempts)
		}
	}
	if ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { return "new-hash", nil }); err != nil || ok {
		t.Fatal("第五次后正确码仍可用")
	}
	current := f.data.db.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(input.TokenHash)).OnlyX(ctx)
	if current.Status != reset.StatusFAILED || current.CodeHash != "" {
		t.Fatal("未作废")
	}
}
func TestPasswordResetConsumeOnceRevokesAllAndRejectsOldCredential(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.data.db.User.UpdateOneID(f.userID).SetPasswordHash("old-hash").ExecX(ctx)
	f.createSession(ctx, "old-device-one", f.systemWorkspace)
	f.createSession(ctx, "old-device-two", f.branch)
	input, _ := reserveReset(t, r)
	var wg sync.WaitGroup
	var successes atomic.Int32
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { return "new-hash", nil })
			if err != nil {
				t.Errorf("消费 %v", err)
			}
			if ok {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("成功次数 %d", successes.Load())
	}
	if count := f.data.db.Session.Query().Where(session.UserIDEQ(f.userID), session.RevokedAtIsNil()).CountX(ctx); count != 0 {
		t.Fatalf("仍有效会话 %d", count)
	}
	if _, err := f.repo.FindSession(ctx, "old-device-one", now); err == nil {
		t.Fatal("旧会话仍可验证")
	}
	oldHash := "old-hash"
	err := f.repo.CreateSession(ctx, &biz.Session{TokenHash: "racing-old-login", UserID: f.userID, OrganizationID: f.branch, ExpiresAt: now.Add(time.Hour), LoginAccount: "13800138000", VerifiedPasswordHash: &oldHash}, "", &biz.AuditEvent{UserID: &f.userID, Action: "auth.login", Result: "success"})
	if !errors.Is(err, biz.ErrInvalidCredentials) {
		t.Fatalf("旧凭据建会话 %v", err)
	}
	if f.data.db.AuditLog.Query().Where(auditlog.ActionEQ("auth.password.reset")).CountX(ctx) != 1 {
		t.Fatal("成功审计次数错误")
	}
}
func TestPasswordResetExpiryResendAndBindingChanges(t *testing.T) {
	for _, scenario := range []string{"过期", "重发", "手机变更", "钉钉变更", "停用", "待分配", "重复手机号", "用户名冲突", "无成员"} {
		t.Run(scenario, func(t *testing.T) {
			f, r := setupPasswordReset(t)
			ctx := context.Background()
			input, _ := reserveReset(t, r)
			switch scenario {
			case "过期":
				f.data.db.PasswordResetChallenge.Update().SetExpiresAt(time.Now().UTC()).ExecX(ctx)
			case "重发":
				f.data.db.PasswordResetChallenge.Update().SetResendAvailableAt(time.Now().UTC()).ExecX(ctx)
				reserveReset(t, r)
			case "手机变更":
				f.data.db.User.UpdateOneID(f.userID).SetDingtalkMobileNormalized("13900139000").ExecX(ctx)
			case "钉钉变更":
				f.data.db.User.UpdateOneID(f.userID).SetDingtalkUserid("changed").ExecX(ctx)
			case "停用":
				f.data.db.User.UpdateOneID(f.userID).SetEnabled(false).ExecX(ctx)
			case "待分配":
				f.data.db.User.UpdateOneID(f.userID).SetEnabled(false).SetPendingAssignment(true).ExecX(ctx)
			case "重复手机号":
				f.data.db.User.Create().SetDisplayName("重复号").SetDingtalkMobileNormalized("13800138000").SaveX(ctx)
			case "用户名冲突":
				f.data.db.User.Create().SetDisplayName("同名账号").SetUsername("13800138000").SaveX(ctx)
			case "无成员":
				f.data.db.Membership.Update().SetEnabled(false).ExecX(ctx)
			}
			if ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { return "new-hash", nil }); err != nil || ok {
				t.Fatalf("不应消费 %v %v", ok, err)
			}
		})
	}
}
func TestPasswordResetAuditFailureRollsBackAll(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	f.data.db.User.UpdateOneID(f.userID).SetPasswordHash("old-hash").ExecX(ctx)
	f.createSession(ctx, "old-device", f.branch)
	input, _ := reserveReset(t, r)
	f.data.db.AuditLog.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			return nil, fmt.Errorf("模拟审计失败")
		})
	})
	if ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { return "new-hash", nil }); err == nil || ok {
		t.Fatal("审计失败应回滚")
	}
	if hash := f.data.db.User.GetX(ctx, f.userID).PasswordHash; hash == nil || *hash != "old-hash" {
		t.Fatal("密码未回滚")
	}
	if current := f.data.db.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(input.TokenHash)).OnlyX(ctx); current.Status != reset.StatusREADY {
		t.Fatal("挑战未回滚")
	}
	if f.data.db.Session.Query().Where(session.UserIDEQ(f.userID), session.RevokedAtIsNil()).CountX(ctx) != 1 {
		t.Fatal("会话未回滚")
	}
}
func TestPasswordResetUnknownAndUnqualifiedTargetsUseUnusableChallenge(t *testing.T) {
	for _, scenario := range []string{"未知", "重复", "停用", "无手机号", "无钉钉", "待分配", "无成员"} {
		t.Run(scenario, func(t *testing.T) {
			f, r := setupPasswordReset(t)
			ctx := context.Background()
			switch scenario {
			case "未知", "无手机号":
				f.data.db.User.UpdateOneID(f.userID).ClearDingtalkMobileNormalized().ExecX(ctx)
			case "重复":
				f.data.db.User.Create().SetDisplayName("重复").SetDingtalkMobileNormalized("13800138000").SaveX(ctx)
			case "停用":
				f.data.db.User.UpdateOneID(f.userID).SetEnabled(false).ExecX(ctx)
			case "无钉钉":
				f.data.db.User.UpdateOneID(f.userID).ClearDingtalkUserid().ExecX(ctx)
			case "待分配":
				f.data.db.User.UpdateOneID(f.userID).SetEnabled(false).SetPendingAssignment(true).ExecX(ctx)
			case "无成员":
				f.data.db.Membership.Update().SetEnabled(false).ExecX(ctx)
			}
			input, _ := reserveReset(t, r)
			if input.UserID != nil {
				t.Fatal("不合格目标获得发送资格")
			}
			if ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { return "new-hash", nil }); err != nil || ok {
				t.Fatal("不合格挑战可消费")
			}
		})
	}
}
func TestPasswordResetFailedPendingAndAdminResetInvalidation(t *testing.T) {
	for _, scenario := range []string{"失败", "未激活", "管理员重置"} {
		t.Run(scenario, func(t *testing.T) {
			f, r := setupPasswordReset(t)
			ctx := context.Background()
			input, _ := reserveReset(t, r)
			switch scenario {
			case "失败":
				f.data.db.PasswordResetChallenge.Update().SetStatus(reset.StatusPENDING).ExecX(ctx)
				if ready, err := r.CompletePasswordResetSend(ctx, input, false); err != nil || ready {
					t.Fatal("失败发送被激活")
				}
			case "未激活":
				f.data.db.PasswordResetChallenge.Update().SetStatus(reset.StatusPENDING).ExecX(ctx)
			case "管理员重置":
				if err := NewAdminRepo(f.data).ResetUserPassword(ctx, f.systemWorkspace, f.userID, "admin-new", nil, &biz.AuditEvent{UserID: &f.userID, Action: "admin.password.reset", Result: "success"}); err != nil {
					t.Fatal(err)
				}
			}
			if ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { return "new-hash", nil }); err != nil || ok {
				t.Fatal("失效码可消费")
			}
		})
	}
}
func TestPasswordResetIPBucketIndependentAndBounded(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	for i := 1; i <= 21; i++ {
		exceeded, err := r.RecordPasswordResetSendIP(ctx, "reset-ip", now)
		if err != nil || exceeded != (i > 20) {
			t.Fatalf("第 %d 次 %v %v", i, exceeded, err)
		}
	}
	if exceeded, err := NewAuthRepo(f.data).LoginRateLimitExceeded(ctx, []string{"account-different"}, now, time.Minute, 5); err != nil || exceeded {
		t.Fatal("污染登录限流")
	}
	if exceeded, err := r.RecordPasswordResetSendIP(ctx, "reset-ip", now.Add(10*time.Minute)); err != nil || exceeded {
		t.Fatal("窗口后未恢复")
	}
}
func TestMobileLoginIdentityResolutionAndNoUsername(t *testing.T) {
	f, _ := setupPasswordReset(t)
	ctx := context.Background()
	hash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	f.data.db.User.UpdateOneID(f.userID).ClearUsername().SetPasswordHash(hash).ExecX(ctx)
	uc := biz.NewAuthUsecase(f.repo, &biz.SessionPolicy{TTL: time.Hour}, nil, nil)
	for _, input := range []string{"13800138000", "+86 138-0013-8000", "0086 13800138000"} {
		result, err := uc.Login(ctx, input, "correct-password", uuid.Nil, "test", "")
		if err != nil || result.Principal.UserID != f.userID {
			t.Fatalf("手机号登录 %v", err)
		}
	}
	// 同用户双命中可登录，其他用户重复（即便停用）必须拒绝。
	f.data.db.User.UpdateOneID(f.userID).SetUsername("13800138000").ExecX(ctx)
	if _, err := f.repo.FindCredential(ctx, "13800138000"); err != nil {
		t.Fatal(err)
	}
	other := f.data.db.User.Create().SetDisplayName("另一个用户").SetDingtalkMobileNormalized("13800138000").SetEnabled(false).SaveX(ctx)
	if _, err := f.repo.FindCredential(ctx, "13800138000"); !errors.Is(err, biz.ErrAuthAccountConflict) {
		t.Fatalf("重复手机号 %v", err)
	}
	f.data.db.User.UpdateOneID(other.ID).ClearDingtalkMobileNormalized().SetUsername("13800138000-other").ExecX(ctx)
	f.data.db.User.UpdateOneID(f.userID).ClearUsername().ExecX(ctx)
	f.data.db.User.UpdateOneID(other.ID).SetUsername("13800138000").ExecX(ctx)
	if _, err := f.repo.FindCredential(ctx, "13800138000"); !errors.Is(err, biz.ErrAuthAccountConflict) {
		t.Fatalf("用户名冲突 %v", err)
	}
	f.data.db.User.UpdateOneID(f.userID).ClearDingtalkMobileNormalized().ExecX(ctx)
	if _, err := f.repo.FindCredential(ctx, "+86 13800138000"); !errors.Is(err, biz.ErrInvalidCredentials) {
		t.Fatal("未绑定本地数字用户名不接受手机号前缀别名", err)
	}
	if _, err := f.repo.FindCredential(ctx, "13800138000"); err != nil {
		t.Fatal("本地数字用户名仍可精确登录", err)
	}
	if _, err := f.repo.FindCredential(ctx, "missing-user"); !errors.Is(err, biz.ErrInvalidCredentials) {
		t.Fatal(err)
	}
}

// 验证真实发送编排在事务提交后执行，8 个客户端只会向绑定收件人发送一次。
type resetIntegrationSender struct {
	data    *Data
	calls   atomic.Int32
	content string
	mu      sync.Mutex
}

func (s *resetIntegrationSender) Enabled() bool { return true }
func (s *resetIntegrationSender) SendMarkdown(ctx context.Context, userID string, message biz.DingTalkMarkdownMessage) error {
	s.calls.Add(1)
	if userID != "ding-reset" {
		return fmt.Errorf("收件人不是绑定用户")
	}
	current, err := s.data.db.PasswordResetChallenge.Query().Only(ctx)
	if err != nil {
		return err
	}
	if current.Status != reset.StatusPENDING {
		return fmt.Errorf("外部调用前状态未提交")
	}
	s.mu.Lock()
	s.content = message.Text
	s.mu.Unlock()
	return nil
}
func TestPasswordResetSendOnceOutsideTransactionAndFirstPassword(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	f.data.db.User.UpdateOneID(f.userID).ClearUsername().ClearPasswordHash().ExecX(ctx)
	sender := &resetIntegrationSender{data: f.data}
	uc := biz.NewPasswordResetUsecase(r, sender)
	var wg sync.WaitGroup
	var success atomic.Int32
	var result *biz.PasswordResetCodeResult
	var mu sync.Mutex
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reply, err := uc.RequestCode(ctx, "13800138000", "127.0.0.1")
			if err == nil {
				success.Add(1)
				mu.Lock()
				result = reply
				mu.Unlock()
			} else if !strings.Contains(err.Error(), "AUTH_PASSWORD_RESET_COOLDOWN") {
				t.Errorf("申请 %v", err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || sender.calls.Load() != 1 {
		t.Fatalf("申请成功 %d，实际发送 %d", success.Load(), sender.calls.Load())
	}
	code := regexp.MustCompile(`\*\*(\d{6})\*\*`).FindStringSubmatch(sender.content)
	if len(code) != 2 {
		t.Fatal("验证码格式错误")
	}
	if err := uc.Confirm(ctx, result.ChallengeToken, code[1], "first-password-123"); err != nil {
		t.Fatal(err)
	}
	auth := biz.NewAuthUsecase(f.repo, &biz.SessionPolicy{TTL: time.Hour}, nil, nil)
	if reply, err := auth.Login(ctx, "13800138000", "first-password-123", uuid.Nil, "test", ""); err != nil || reply.Principal.UserID != f.userID {
		t.Fatalf("首次密码登录 %v", err)
	}
}
func TestPasswordResetInvalidChallengeNeverHashes(t *testing.T) {
	f, r := setupPasswordReset(t)
	ctx := context.Background()
	if ok, err := r.ConsumePasswordReset(ctx, resetTokenHash("unknown"), "wrong", func() (string, error) { t.Error("未知码不得哈希"); return "", nil }); err != nil || ok {
		t.Fatalf("未知令牌 %v %v", ok, err)
	}
	input, _ := reserveReset(t, r)
	f.data.db.PasswordResetChallenge.Update().SetExpiresAt(time.Now().UTC()).ExecX(ctx)
	if ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) { t.Error("过期码不得哈希"); return "", nil }); err != nil || ok {
		t.Fatalf("过期令牌 %v %v", ok, err)
	}
}

func TestMobileLoginWaitsForPasswordResetBeforeCreatingSession(t *testing.T) {
	f, _ := setupPasswordReset(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f.data.db.User.UpdateOneID(f.userID).SetPasswordHash("old-hash").ExecX(ctx)
	queried := make(chan struct{})
	marker := new(int)
	var once sync.Once
	f.data.db.User.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			if ctx.Value(authSessionLockProbeKey{}) == marker {
				once.Do(func() { close(queried) })
			}
			return next.Query(ctx, q)
		})
	}))
	locked := make(chan struct{})
	resetDone := make(chan error, 1)
	loginDone := make(chan error, 1)
	go func() {
		resetDone <- f.data.WithTx(ctx, func(tx *ent.Tx) error {
			if _, err := tx.User.Query().Where(user.IDEQ(f.userID)).ForUpdate().Only(ctx); err != nil {
				return err
			}
			close(locked)
			select {
			case <-queried:
			case <-ctx.Done():
				return ctx.Err()
			}
			return tx.User.UpdateOneID(f.userID).SetPasswordHash("new-hash").Exec(ctx)
		})
	}()
	go func() {
		select {
		case <-locked:
		case <-ctx.Done():
			loginDone <- ctx.Err()
			return
		}
		oldHash := "old-hash"
		loginDone <- f.repo.CreateSession(context.WithValue(ctx, authSessionLockProbeKey{}, marker), &biz.Session{UserID: f.userID, OrganizationID: f.branch, TokenHash: "old-password-race", ExpiresAt: time.Now().Add(time.Hour), LoginAccount: "13800138000", VerifiedPasswordHash: &oldHash}, "", &biz.AuditEvent{UserID: &f.userID, Action: "auth.login", Result: "success"})
	}()
	if err := <-resetDone; err != nil {
		t.Fatal(err)
	}
	if err := <-loginDone; !errors.Is(err, biz.ErrInvalidCredentials) {
		t.Fatalf("旧密码在重置后建立会话 %v", err)
	}
	if f.data.db.Session.Query().CountX(ctx) != 0 {
		t.Fatal("竞争后产生会话")
	}
}
