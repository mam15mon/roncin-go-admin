package data

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	lockent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
)

func TestLossOrderReviewWithoutCommentPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	newCorrectionTestUsecase(t, f)
	o := f.createOrder("LOSS-REVIEW-" + f.suffix)
	fee := addOrdinaryTestFee(t, f, o, 100)
	uc := biz.NewOrderLockUsecase(NewOrderLockRepo(f.data, nil))
	state, err := uc.GetOrderLockState(f.ctx, f.organizationID, o.ID, f.approverPrincipal())
	if err != nil {
		t.Fatal(err)
	}
	if !state.CanLock || state.FeeSummary == nil || !state.FeeSummary.Profit.IsNegative() {
		t.Fatalf("亏损订单应可进入人工审核: %+v", state)
	}
	review := biz.OrderLockReviewInput{
		ExpectedFeeContentVersion: state.FeeContentVersion,
		FeeContentFingerprint:     state.FeeContentFingerprint,
	}
	for _, change := range []struct {
		name string
		edit func(*biz.OrderLockReviewInput)
	}{
		{"费用版本不匹配", func(input *biz.OrderLockReviewInput) { input.ExpectedFeeContentVersion++ }},
		{"费用指纹不匹配", func(input *biz.OrderLockReviewInput) { input.FeeContentFingerprint = strings.Repeat("0", 64) }},
	} {
		t.Run(change.name, func(t *testing.T) {
			stale := review
			change.edit(&stale)
			if _, err := uc.LockOrder(f.ctx, f.approverPrincipal(), o.ID, state.OrderVersion, uuid.NewString(), stale, nil); err != biz.ErrFeeReviewConflict {
				t.Fatalf("空意见审核仍必须校验费用快照: %v", err)
			}
			after := f.data.db.Order.GetX(f.ctx, o.ID)
			if after.LockedAt != nil || after.Version != state.OrderVersion || f.data.db.OrderLockRecord.Query().Where(lockent.OrderIDEQ(o.ID)).CountX(f.ctx) != 0 {
				t.Fatal("快照冲突不得产生锁状态或锁定事实")
			}
		})
	}
	key := uuid.NewString()
	result, err := uc.LockOrder(f.ctx, f.approverPrincipal(), o.ID, state.OrderVersion, key, review, nil)
	if err != nil {
		t.Fatalf("亏损订单应可不填写意见直接审核锁定: %v", err)
	}
	after := f.data.db.Order.GetX(f.ctx, o.ID)
	if after.LockedAt == nil || after.LockedBy == nil || *after.LockedBy != f.approverID || after.Version != state.OrderVersion+1 {
		t.Fatalf("审核成功应固化人工锁及订单版本: %+v", after)
	}
	record := f.data.db.OrderLockRecord.Query().Where(lockent.OrderIDEQ(o.ID)).OnlyX(f.ctx)
	var snapshot biz.FeeReviewSnapshot
	if err := json.Unmarshal(record.FeeSnapshot, &snapshot); err != nil || len(snapshot.Fees) != 1 || snapshot.Fees[0].ID != fee.ID || !snapshot.Summary.Profit.Equal(state.FeeSummary.Profit) {
		t.Fatalf("空意见审核仍应保存完整费用与亏损快照: %+v %v", snapshot, err)
	}
	if record.ReviewComment == nil || *record.ReviewComment != "" {
		t.Fatalf("不得为无意见审核伪造默认说明: %+v", record.ReviewComment)
	}
	replay, err := uc.LockOrder(f.ctx, f.approverPrincipal(), o.ID, state.OrderVersion, key, review, nil)
	if err != nil || replay.LockRecord.ID != result.LockRecord.ID || f.data.db.OrderLockRecord.Query().Where(lockent.OrderIDEQ(o.ID)).CountX(f.ctx) != 1 {
		t.Fatalf("空意见审核幂等重放不得重复锁定: %+v %v", replay, err)
	}
}
