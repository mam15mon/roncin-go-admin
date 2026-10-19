package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	reset "github.com/roncin/roncin-go-admin/server/internal/data/ent/passwordresetchallenge"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// 先由真实用户行锁阻塞请求，再让持久化时限跨过当前时刻，禁止锁前时间继续生效。
func TestPasswordResetRechecksTimeAfterUserLockWait(t *testing.T) {
	for _, scenario := range []string{"冷却结束", "激活过期", "消费过期"} {
		t.Run(scenario, func(t *testing.T) {
			f, r := setupPasswordReset(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			input, _ := reserveReset(t, r)
			if scenario == "激活过期" {
				f.data.db.PasswordResetChallenge.Update().SetStatus(reset.StatusPENDING).ExecX(ctx)
			}
			type result struct {
				accepted bool
				err      error
			}
			done := make(chan result, 1)
			var boundary time.Time
			err := f.data.WithTx(ctx, func(tx *ent.Tx) error {
				if _, err := tx.User.Query().Where(user.IDEQ(f.userID)).ForUpdate().Only(ctx); err != nil {
					return err
				}
				go func() {
					switch scenario {
					case "冷却结束":
						err := r.ReservePasswordReset(ctx, "13800138000", input)
						done <- result{err == nil, err}
					case "激活过期":
						ok, err := r.CompletePasswordResetSend(ctx, input, true)
						done <- result{ok, err}
					case "消费过期":
						ok, err := r.ConsumePasswordReset(ctx, input.TokenHash, input.CodeHash, func() (string, error) {
							return "", fmt.Errorf("锁等待期间过期的验证码不得计算密码哈希")
						})
						done <- result{ok, err}
					}
				}()
				// 直接观察当前隔离 Schema 的锁等待，避免 sleep 猜测请求是否真正阻塞。
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for {
					var waiting bool
					if err := f.data.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM pg_locks l JOIN pg_stat_activity a ON a.pid=l.pid
WHERE l.relation='users'::regclass AND a.wait_event_type='Lock'
AND a.pid<>pg_backend_pid() AND a.query LIKE '%"users"%')`).Scan(&waiting); err != nil {
						return err
					}
					if waiting {
						break
					}
					select {
					case early := <-done:
						return fmt.Errorf("用户锁持有时请求提前完成：%v", early.err)
					case <-ticker.C:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				boundary = time.Now().UTC()
				update := tx.PasswordResetChallenge.Update().Where(reset.TokenHashEQ(input.TokenHash))
				if scenario == "冷却结束" {
					return update.SetResendAvailableAt(boundary).Exec(ctx)
				}
				return update.SetExpiresAt(boundary).Exec(ctx)
			})
			if err != nil {
				t.Fatal(err)
			}
			var reply result
			select {
			case reply = <-done:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if reply.err != nil || reply.accepted != (scenario == "冷却结束") {
				t.Fatalf("锁后时效判断错误：accepted=%v err=%v", reply.accepted, reply.err)
			}
			current := f.data.db.PasswordResetChallenge.Query().OnlyX(ctx)
			switch scenario {
			case "冷却结束":
				if input.ResendAvailableAt.Before(boundary.Add(biz.PasswordResetResendInterval)) || input.ExpiresAt.Sub(input.ResendAvailableAt) != biz.PasswordResetTTL-biz.PasswordResetResendInterval {
					t.Fatal("新一代冷却或有效期仍从锁等待前起算")
				}
			case "激活过期":
				if current.Status != reset.StatusFAILED || current.CodeHash != "" {
					t.Fatal("过期发送未置为不可消费状态")
				}
			case "消费过期":
				if current.Status == reset.StatusCONSUMED {
					t.Fatal("过期验证码被消费")
				}
			}
		})
	}
}
