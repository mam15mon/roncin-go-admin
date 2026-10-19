package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	reset "github.com/roncin/roncin-go-admin/server/internal/data/ent/passwordresetchallenge"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type passwordResetRepo struct{ data *Data }

func NewPasswordResetRepo(data *Data) biz.PasswordResetRepo { return &passwordResetRepo{data: data} }

// resetEligible 锁定组织和成员快照；调用者必须先锁用户，之后才可锁挑战。
func resetEligible(ctx context.Context, tx *ent.Tx, account *ent.User) (bool, error) {
	if account == nil || !account.Enabled || account.PendingAssignment || account.DingtalkUserid == nil || *account.DingtalkUserid == "" || account.DingtalkUnionid == nil || *account.DingtalkUnionid == "" {
		return false, nil
	}
	nodes, err := tx.Organization.Query().Order(ent.Asc(organization.FieldID)).ForShare().All(ctx)
	if err != nil {
		return false, err
	}
	members, err := tx.Membership.Query().Where(membership.UserIDEQ(account.ID), membership.EnabledEQ(true), membership.HasOrganizationWith(organization.EnabledEQ(true))).Order(ent.Asc(membership.FieldID)).ForShare().All(ctx)
	if err != nil {
		return false, err
	}
	return len(resolveMembershipWorkspaces(membershipRefsFromEnt(members), authOrganizationNodes(nodes))) > 0, nil
}
func (r *passwordResetRepo) ReservePasswordReset(ctx context.Context, mobile string, input *biz.PasswordResetRecord, now time.Time) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 不按启停筛选，重复来源资料和本地账号冲突均不得猜测身份。
		accounts, err := loginAccountQuery(tx.Client(), mobile).Order(ent.Asc(user.FieldID)).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		var account *ent.User
		if len(accounts) == 1 && accounts[0].DingtalkMobileNormalized != nil && *accounts[0].DingtalkMobileNormalized == mobile {
			account = accounts[0]
		}
		eligible, err := resetEligible(ctx, tx, account)
		if err != nil {
			return err
		}
		if eligible {
			input.UserID = &account.ID
			input.DingTalkUserID = *account.DingtalkUserid
			input.DingTalkUnionID = *account.DingtalkUnionid
		}
		// 唯一键串行首次创建；冲突时原值更新并返回既有行，再锁当前行，不在约束错误后的已中止事务继续。
		err = tx.PasswordResetChallenge.Create().SetMobileKeyHash(input.MobileKeyHash).SetTokenHash(input.TokenHash).SetCodeHash("").SetStatus(reset.StatusFAILED).SetExpiresAt(now).SetResendAvailableAt(now).OnConflictColumns(reset.FieldMobileKeyHash).Ignore().Exec(ctx)
		if err != nil {
			return err
		}
		current, err := tx.PasswordResetChallenge.Query().Where(reset.MobileKeyHashEQ(input.MobileKeyHash)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		if current.ResendAvailableAt.After(now) {
			return biz.PasswordResetCooldown(now, current.ResendAvailableAt)
		}
		update := current.Update().SetTokenHash(input.TokenHash).SetCodeHash(input.CodeHash).SetStatus(reset.StatusPENDING).SetExpiresAt(input.ExpiresAt).SetResendAvailableAt(input.ResendAvailableAt).SetFailedAttempts(0).SetDingtalkUserid(input.DingTalkUserID).SetDingtalkUnionid(input.DingTalkUnionID)
		if input.UserID == nil {
			update.ClearUserID().SetStatus(reset.StatusFAILED).SetCodeHash("")
		} else {
			update.SetUserID(*input.UserID)
		}
		return update.Exec(ctx)
	})
}
func resetBindingMatches(account *ent.User, record *ent.PasswordResetChallenge) bool {
	return account != nil && record.UserID != nil && account.ID == *record.UserID && account.DingtalkMobileNormalized != nil && biz.PasswordResetMobileKey(*account.DingtalkMobileNormalized) == record.MobileKeyHash && account.DingtalkUserid != nil && *account.DingtalkUserid == record.DingtalkUserid && account.DingtalkUnionid != nil && *account.DingtalkUnionid == record.DingtalkUnionid
}

// lockResetTarget 先用户、组织成员，后挑战；token 快照在任何锁之外读，最终锁内重查。
func (r *passwordResetRepo) lockResetTarget(ctx context.Context, tx *ent.Tx, snapshot *ent.PasswordResetChallenge) (*ent.User, bool, error) {
	if snapshot.UserID == nil {
		return nil, false, nil
	}
	account, err := tx.User.Query().Where(user.IDEQ(*snapshot.UserID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !resetBindingMatches(account, snapshot) {
		return account, false, nil
	}
	// 再核对号码所有命中；不再锁其他用户，避免持有单用户锁后反序获取更小 ID。
	accounts, err := loginAccountQuery(tx.Client(), *account.DingtalkMobileNormalized).All(ctx)
	if err != nil {
		return nil, false, err
	}
	if len(accounts) != 1 || accounts[0].ID != account.ID {
		return account, false, nil
	}
	eligible, err := resetEligible(ctx, tx, account)
	return account, eligible, err
}
func (r *passwordResetRepo) CompletePasswordResetSend(ctx context.Context, input *biz.PasswordResetRecord, accepted bool, now time.Time) (bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	snapshot, err := client.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(input.TokenHash)).Only(ctx)
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	ready := false
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		_, eligible, err := r.lockResetTarget(ctx, tx, snapshot)
		if err != nil {
			return err
		}
		current, err := tx.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(input.TokenHash)).ForUpdate().Only(ctx)
		if ent.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if current.Status != reset.StatusPENDING {
			return nil
		}
		ready = accepted && eligible && current.ExpiresAt.After(now)
		update := current.Update().SetStatus(reset.StatusFAILED).SetCodeHash("")
		if ready {
			update.SetStatus(reset.StatusREADY).SetCodeHash(input.CodeHash)
		}
		return update.Exec(ctx)
	})
	return ready, err
}
func (r *passwordResetRepo) ConsumePasswordReset(ctx context.Context, tokenHash, codeHash string, hashPassword func() (string, error), now time.Time) (bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	snapshot, err := client.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(tokenHash)).Only(ctx)
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	consumed := false
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, eligible, err := r.lockResetTarget(ctx, tx, snapshot)
		if err != nil {
			return err
		}
		current, err := tx.PasswordResetChallenge.Query().Where(reset.TokenHashEQ(tokenHash)).ForUpdate().Only(ctx)
		if ent.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !eligible || !resetBindingMatches(account, current) || current.Status != reset.StatusREADY || !current.ExpiresAt.After(now) || current.FailedAttempts >= 5 {
			return nil
		}
		if !biz.PasswordResetCodeMatches(current.CodeHash, codeHash) {
			update := current.Update().AddFailedAttempts(1)
			if current.FailedAttempts >= 4 {
				update.SetStatus(reset.StatusFAILED).SetCodeHash("")
			}
			// 返回 nil 提交计数；业务错误在事务外返回，不能回滚失败次数。
			return update.Exec(ctx)
		}
		// 仅锁内正确的一次性验证码触发 Argon2，未知/错误码不付出密码哈希成本。
		passwordHash, err := hashPassword()
		if err != nil {
			return err
		}
		if err := tx.User.UpdateOneID(account.ID).SetPasswordHash(passwordHash).Exec(ctx); err != nil {
			return err
		}
		if err := current.Update().SetStatus(reset.StatusCONSUMED).SetCodeHash("").Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.Session.Update().Where(session.UserIDEQ(account.ID), session.RevokedAtIsNil()).SetRevokedAt(now).Save(ctx); err != nil {
			return err
		}
		if err := writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{UserID: &account.ID, Action: "auth.password.reset", Result: "success"}); err != nil {
			return err
		}
		consumed = true
		return nil
	})
	return consumed && err == nil, err
}
func (r *passwordResetRepo) RecordPasswordResetSendIP(ctx context.Context, keyHash string, now time.Time) (bool, error) {
	var attempts int
	err := r.data.withSQLTx(ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, `INSERT INTO login_rate_limit_buckets (id,created_at,updated_at,key_hash,window_started_at,attempts)
VALUES ($1,$2,$2,$3,$2,1) ON CONFLICT (key_hash) DO UPDATE SET updated_at=EXCLUDED.updated_at,
window_started_at=CASE WHEN login_rate_limit_buckets.window_started_at <= $4 THEN EXCLUDED.window_started_at ELSE login_rate_limit_buckets.window_started_at END,
attempts=CASE WHEN login_rate_limit_buckets.window_started_at <= $4 THEN 1 ELSE LEAST(login_rate_limit_buckets.attempts+1,21) END RETURNING attempts`, uuid.Must(uuid.NewV7()), now, keyHash, now.Add(-10*time.Minute)).Scan(&attempts)
	})
	return attempts > 20, err
}

// invalidatePasswordResetChallenges 与用户写入共用事务，按主键固定多行加锁顺序。
func invalidatePasswordResetChallenges(ctx context.Context, tx *ent.Tx, userID uuid.UUID) error {
	records, err := tx.PasswordResetChallenge.Query().Where(reset.UserIDEQ(userID), reset.StatusIn(reset.StatusPENDING, reset.StatusREADY)).Order(ent.Asc(reset.FieldID)).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := record.Update().SetStatus(reset.StatusFAILED).SetCodeHash("").Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
