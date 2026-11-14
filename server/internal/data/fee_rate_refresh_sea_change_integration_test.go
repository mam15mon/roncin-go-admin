package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

const (
	rateRefreshW2ETD = "2026-09-15T00:00:00Z"
	rateRefreshW3ETD = "2026-09-22T00:00:00Z"
)

func newSeaChangeRateEnv(t *testing.T) (*splitTestEnv, *rateRefreshEnv) {
	t.Helper()
	env := newSplitTestEnv(t)
	cleanupRateRefresh(t, env.data, env.orgID)
	e := &rateRefreshEnv{t: t, data: env.data, orgID: env.orgID, partnerID: env.customerID, actorID: env.userID, suffix: uuid.NewString()[:8]}
	e.seedUSDRates()
	return env, e
}

// prepareSeaChangeOrder 把夹具订单设为出口、ETD 落在 W1，并把夹具费用改成系统周汇率（W1 汇率 6.8）或保留手工汇率。
func prepareSeaChangeOrder(t *testing.T, env *splitTestEnv, f *splitTestFixture, manual bool) {
	t.Helper()
	ctx := context.Background()
	env.data.db.Order.UpdateOneID(f.order.ID).SetEtd(rateRefreshW1).SaveX(ctx)
	if manual {
		return
	}
	env.data.db.OrderFee.UpdateOneID(f.fee1.ID).
		SetExchangeRate("6.80000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceWEEKLY).
		SetExchangeRateDate("2026-09-10").
		SetBaseCurrencyAmount("10200.00000000").
		SaveX(ctx)
}

func seaChangeReassignInput(env *splitTestEnv, f *splitTestFixture, key, etd string) *biz.SeaOrderReassignmentInput {
	return &biz.SeaOrderReassignmentInput{
		OrderID:            f.order.ID,
		IdempotencyKey:     key,
		RequestFingerprint: "fp-" + key,
		Reason:             "联动汇率改配验证",
		ResponsibilityType: biz.ResponsibilityTypeCarrier,
		Confirmation:       &biz.SeaExternalConfirmation{ConfirmedByParty: "船代窗口", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "船代确认"},
		Target: &biz.SeaOrderReassignmentTargetInput{
			TargetType: biz.SplitTargetTypeNew, MasterNo: "RRMBL" + uuid.NewString()[:6],
			ShippingLineID: &env.carrier2ID, VesselName: "RATE SHIP", VoyageNo: "001E", ETD: etd,
		},
		ExpectedOrderVersion: f.order.Version,
		ExpectedLinkVersion:  f.link.Version,
	}
}

// TestFeeRateRefreshReassignmentPostgres 验证改配使订单取汇率日变化时联动重取费用汇率，缺汇率整体回滚。
func TestFeeRateRefreshReassignmentPostgres(t *testing.T) {
	env, e := newSeaChangeRateEnv(t)
	ctx := context.Background()

	t.Run("缺目标周汇率整个改配回滚", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rr1", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		mblBefore := env.data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(env.orgID)).CountX(ctx)
		_, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, seaChangeReassignInput(env, f, "rr-reas-fail", rateRefreshW3ETD))
		if !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("应返回缺汇率错误，实际 %v", err)
		}
		if got := e.order(f.order.ID); got.Etd != rateRefreshW1 || got.Version != f.order.Version {
			t.Fatalf("订单应回滚: etd=%s version=%d", got.Etd, got.Version)
		}
		e.expectFee("费用不变", f.fee1.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "10200", 1)
		if link := env.data.db.SeaMasterBillOrderLink.GetX(ctx, f.link.ID); link.Status != seamasterbillorderlinkent.StatusACTIVE {
			t.Fatalf("原关联应保持 ACTIVE: %s", link.Status)
		}
		if after := env.data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(env.orgID)).CountX(ctx); after != mblBefore {
			t.Fatalf("新母单应回滚: %d→%d", mblBefore, after)
		}
	})

	t.Run("日期变化联动更新系统汇率", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rr2", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		if _, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, seaChangeReassignInput(env, f, "rr-reas-ok", rateRefreshW2ETD)); err != nil {
			t.Fatalf("改配失败: %v", err)
		}
		if got := e.order(f.order.ID); got.Etd != rateRefreshW2ETD {
			t.Fatalf("订单 ETD 应同步目标航程: %s", got.Etd)
		}
		e.expectFee("联动", f.fee1.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "10500", 2)
		if e.refreshAuditCount(f.order.ID) != 1 {
			t.Fatal("应写入一条联动审计")
		}
	})

	t.Run("手工汇率保留", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rr3", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, true)
		if _, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, seaChangeReassignInput(env, f, "rr-reas-manual", rateRefreshW2ETD)); err != nil {
			t.Fatalf("改配失败: %v", err)
		}
		e.expectFee("手工保留", f.fee1.ID, "7.1", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-07", "10650", 1)
	})

	t.Run("无可联动费用照常成功", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rr4", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		env.data.db.OrderFee.DeleteOneID(f.fee1.ID).ExecX(ctx)
		// 目标周无汇率也不影响：没有费用需要重取。
		if _, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, seaChangeReassignInput(env, f, "rr-reas-nofee", rateRefreshW3ETD)); err != nil {
			t.Fatalf("无费用改配应成功: %v", err)
		}
		if got := e.order(f.order.ID); got.Etd != rateRefreshW3ETD {
			t.Fatalf("订单 ETD 应同步: %s", got.Etd)
		}
	})
}

// seaChangeSplitInput 构造 60/40 拆票：feeToCreated 为 true 时费用迁往新票，否则留在原票；
// originalTarget/createdTarget 分别指定两票的目标（"current" 或带日期的新母单）。
func seaChangeSplitInput(env *splitTestEnv, f *splitTestFixture, key, etd string, originalToNew, createdToNew, feeToCreated, hasFee bool) *biz.SeaOrderSplitInput {
	in := f.standardSplitInput(key, "fp-"+key, splitFixtureOptions{})
	in.Targets = []*biz.SeaOrderSplitTargetInput{
		{ClientTargetKey: "target-current", TargetType: biz.SplitTargetTypeCurrent},
		{
			ClientTargetKey: "target-new", TargetType: biz.SplitTargetTypeNew,
			MasterNo: "RRSPLIT" + uuid.NewString()[:6], ShippingLineID: &env.carrierID,
			VesselName: "RATE SPLIT", VoyageNo: "002W", ETD: etd,
		},
	}
	if originalToNew {
		in.Results[0].ClientTargetKey = "target-new"
	}
	if createdToNew {
		in.Results[1].ClientTargetKey = "target-new"
	}
	in.Results[0].DraftFeeIDs, in.Results[1].DraftFeeIDs = nil, nil
	if hasFee {
		if feeToCreated {
			in.Results[1].DraftFeeIDs = []uuid.UUID{f.fee1.ID}
		} else {
			in.Results[0].DraftFeeIDs = []uuid.UUID{f.fee1.ID}
		}
	} else {
		in.ExpectedVersions.FeeVersions = map[uuid.UUID]uint64{}
	}
	in.Confirmation = &biz.SeaExternalConfirmation{ConfirmedByParty: "船代窗口", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "船代确认拆票改配"}
	return in
}

func seaChangeCreatedOrderID(t *testing.T, evt *biz.SeaOrderSplitEvent) uuid.UUID {
	t.Helper()
	for _, r := range evt.Results {
		if r.ResultRole == biz.ResultRoleCreated {
			return r.OrderID
		}
	}
	t.Fatal("拆票结果缺少新票")
	return uuid.Nil
}

// TestFeeRateRefreshSplitPostgres 验证拆票：新票改配（费用整行克隆后联动）、原票改配（留在原票的费用联动），
// 缺汇率整体回滚，手工汇率保留，无费用照常成功。
func TestFeeRateRefreshSplitPostgres(t *testing.T) {
	env, e := newSeaChangeRateEnv(t)
	ctx := context.Background()
	feeOf := func(orderID uuid.UUID) *ent.OrderFee {
		fees := env.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(orderID)).AllX(ctx)
		if len(fees) != 1 {
			t.Fatalf("订单 %s 应恰有一条费用，实际 %d", orderID, len(fees))
		}
		return fees[0]
	}

	t.Run("缺目标周汇率整个拆票回滚", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rs1", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		orders := env.data.db.Order.Query().Where(orderent.OrganizationIDEQ(env.orgID)).CountX(ctx)
		mbls := env.data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(env.orgID)).CountX(ctx)
		_, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, seaChangeSplitInput(env, f, "rr-split-fail", rateRefreshW3ETD, false, true, true, true))
		if !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("应返回缺汇率错误，实际 %v", err)
		}
		if after := env.data.db.Order.Query().Where(orderent.OrganizationIDEQ(env.orgID)).CountX(ctx); after != orders {
			t.Fatalf("新订单应回滚: %d→%d", orders, after)
		}
		if after := env.data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(env.orgID)).CountX(ctx); after != mbls {
			t.Fatalf("新母单应回滚: %d→%d", mbls, after)
		}
		if got := e.order(f.order.ID); got.Etd != rateRefreshW1 || got.Version != f.order.Version {
			t.Fatalf("原订单应回滚: etd=%s version=%d", got.Etd, got.Version)
		}
		e.expectFee("费用仍在原票且不变", f.fee1.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "10200", 1)
	})

	t.Run("新票改配联动克隆费用", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rs2", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		evt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, seaChangeSplitInput(env, f, "rr-split-created", rateRefreshW2ETD, false, true, true, true))
		if err != nil {
			t.Fatalf("拆票失败: %v", err)
		}
		createdID := seaChangeCreatedOrderID(t, evt)
		if got := e.order(createdID); got.Etd != rateRefreshW2ETD {
			t.Fatalf("新票 ETD 应为目标航程: %s", got.Etd)
		}
		cloned := feeOf(createdID)
		e.expectFee("克隆费用联动", cloned.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "10500", 2)
		if got := e.order(f.order.ID); got.Etd != rateRefreshW1 {
			t.Fatalf("原票仍在当前航程，ETD 不变: %s", got.Etd)
		}
	})

	t.Run("新票改配手工汇率保留", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rs3", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, true)
		evt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, seaChangeSplitInput(env, f, "rr-split-manual", rateRefreshW2ETD, false, true, true, true))
		if err != nil {
			t.Fatalf("拆票失败: %v", err)
		}
		cloned := feeOf(seaChangeCreatedOrderID(t, evt))
		e.expectFee("手工保留", cloned.ID, "7.1", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-07", "10650", 1)
	})

	t.Run("原票改配联动留存费用", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rs4", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, seaChangeSplitInput(env, f, "rr-split-original", rateRefreshW2ETD, true, false, false, true)); err != nil {
			t.Fatalf("拆票失败: %v", err)
		}
		if got := e.order(f.order.ID); got.Etd != rateRefreshW2ETD {
			t.Fatalf("原票 ETD 应为目标航程: %s", got.Etd)
		}
		e.expectFee("原票费用联动", f.fee1.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "10500", 2)
		if e.refreshAuditCount(f.order.ID) != 1 {
			t.Fatal("原票应写入一条联动审计")
		}
	})

	t.Run("无可联动费用照常成功", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "rs5", splitFixtureOptions{})
		prepareSeaChangeOrder(t, env, f, false)
		env.data.db.OrderFee.DeleteOneID(f.fee1.ID).ExecX(ctx)
		evt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, seaChangeSplitInput(env, f, "rr-split-nofee", rateRefreshW3ETD, true, true, false, false))
		if err != nil {
			t.Fatalf("无费用拆票应成功: %v", err)
		}
		if got := e.order(seaChangeCreatedOrderID(t, evt)); got.Etd != rateRefreshW3ETD {
			t.Fatalf("新票 ETD 应为目标航程: %s", got.Etd)
		}
	})
}
