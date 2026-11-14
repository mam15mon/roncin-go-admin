package data

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	adjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// 整表保存（SaveOrderFees）的真实 PostgreSQL 验收，覆盖单侧隔离、整体回滚、幂等回放、
// 半应用冲突、手工汇率集合、提成重算与并发版本冲突。

var allSavePerms = biz.OrderFeeSavePermissions{CanCreate: true, CanUpdate: true, CanOverrideExchangeRate: true, CanEditTaxRate: true}

// saveRow 构造一行：existing 非空表示修改（带其当前版本），否则新增。税率显式 0，避免依赖目录税率。
func saveRow(f *feeSupplementPostgresFixture, direction biz.OrderFeeDirection, clientRowID string, existing *biz.OrderFee, amount int64) *biz.OrderFeeSaveRowInput {
	zero := decimal.Zero
	row := &biz.OrderFeeSaveRowInput{ClientRowID: clientRowID, Fee: &biz.OrderFee{
		Direction: direction, FeeSettingID: &f.feeSettingID, BillingUnitID: &f.billingUnitID, SettlementPartyID: f.customerID,
		Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(amount), TaxInclusive: true, TaxRateOverride: &zero, Currency: "CNY",
	}}
	if existing != nil {
		row.FeeID = &existing.ID
		row.ExpectedVersion = existing.Version
	}
	return row
}

func addSideFee(t *testing.T, f *feeSupplementPostgresFixture, o *ent.Order, direction biz.OrderFeeDirection, amount int64) *biz.OrderFee {
	t.Helper()
	row := saveRow(f, direction, "seed", nil, amount)
	row.Fee.IdempotencyKey = uuid.NewString()
	fee, err := normalFeeUsecaseForRollback(f).Add(f.ctx, f.organizationID, f.approverID, o.ID, row.Fee, false, true)
	if err != nil {
		t.Fatal(err)
	}
	return fee
}

// saveSnapshot 汇总订单费用与订单费用内容版本、费用审计条数，用于断言「零部分写入」。
func saveSnapshot(t *testing.T, f *feeSupplementPostgresFixture, orderID uuid.UUID) string {
	t.Helper()
	fees, err := f.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(orderID)).All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	parts := make([]string, 0, len(fees))
	for _, fee := range fees {
		parts = append(parts, fmt.Sprintf("%s|%d|%s|%s|%s", fee.ID, fee.Version, fee.TotalAmount, fee.Currency, fee.ExchangeRateSource))
	}
	sort.Strings(parts)
	order, err := f.data.db.Order.Get(f.ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	audits, err := f.data.db.AuditLog.Query().All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	feeAudits := 0
	for _, a := range audits {
		if strings.HasPrefix(a.Action, "order.fee.") {
			feeAudits++
		}
	}
	return fmt.Sprintf("%s#cv=%d#ov=%d#audits=%d", strings.Join(parts, ";"), order.FeeContentVersion, order.Version, feeAudits)
}

func feeContentVersion(t *testing.T, f *feeSupplementPostgresFixture, orderID uuid.UUID) uint64 {
	t.Helper()
	order, err := f.data.db.Order.Get(f.ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	return order.FeeContentVersion
}

func requireSaveRowError(t *testing.T, err error, want *kratoserrors.Error, clientRowID, field string) {
	t.Helper()
	if err == nil || !kratoserrors.Is(err, want) {
		t.Fatalf("期望错误 %v，实际 %v", want, err)
	}
	var ke *kratoserrors.Error
	if !errors.As(err, &ke) {
		t.Fatalf("不是业务错误: %v", err)
	}
	if ke.Metadata["client_row_id"] != clientRowID || ke.Metadata["field"] != field || ke.Metadata["reason"] != want.Reason {
		t.Fatalf("行级 metadata 不符: %+v，期望 client_row_id=%s field=%s", ke.Metadata, clientRowID, field)
	}
}

func TestOrderFeeSavePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := normalFeeUsecaseForRollback(f)
	save := func(o *ent.Order, direction biz.OrderFeeDirection, key string, perms biz.OrderFeeSavePermissions, rows ...*biz.OrderFeeSaveRowInput) (*biz.OrderFeeSaveResult, error) {
		return uc.Save(f.ctx, f.organizationID, f.approverID, o.ID, direction, key, rows, perms)
	}

	t.Run("单侧提交只影响本侧且订单费用版本只递增一次", func(t *testing.T) {
		o := f.createOrder("SAVE-SIDE-" + f.suffix)
		r1 := addSideFee(t, f, o, biz.OrderFeeReceivable, 500)
		p1 := addSideFee(t, f, o, biz.OrderFeePayable, 100)
		before := feeContentVersion(t, f, o.ID)
		result, err := save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms,
			saveRow(f, biz.OrderFeePayable, "row-p1", p1, 150), saveRow(f, biz.OrderFeePayable, "row-p2", nil, 70))
		if err != nil {
			t.Fatal(err)
		}
		if result.Replayed || len(result.Fees) != 2 {
			t.Fatalf("应返回本侧 2 条费用: %+v", result)
		}
		for _, fee := range result.Fees {
			if fee.Direction != biz.OrderFeePayable {
				t.Fatalf("响应混入另一侧费用: %+v", fee)
			}
		}
		if after := feeContentVersion(t, f, o.ID); after != before+1 || result.FeeContentVersion != after {
			t.Fatalf("订单费用内容版本应只 +1: before=%d after=%d resp=%d", before, after, result.FeeContentVersion)
		}
		gotR1, err := NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, r1.ID)
		if err != nil || gotR1.Version != r1.Version || !gotR1.TotalAmount.Equal(r1.TotalAmount) {
			t.Fatalf("另一侧应收费用不得被影响: %+v %v", gotR1, err)
		}
		gotP1, err := NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, p1.ID)
		if err != nil || gotP1.Version != p1.Version+1 || !gotP1.TotalAmount.Equal(decimal.NewFromInt(150)) {
			t.Fatalf("修改行应 version+1 且金额更新: %+v %v", gotP1, err)
		}
		audits, _ := f.data.db.AuditLog.Query().All(f.ctx)
		saved := 0
		for _, a := range audits {
			if strings.HasPrefix(a.Action, "order.fee.") && strings.Contains(string(a.Details), `"fee.save"`) {
				saved++
			}
		}
		if saved != 2 {
			t.Fatalf("每个变更行应各写一条审计，实际 %d", saved)
		}
	})

	t.Run("任一行失败整体零落库并带行级定位", func(t *testing.T) {
		o := f.createOrder("SAVE-FAIL-" + f.suffix)
		addSideFee(t, f, o, biz.OrderFeeReceivable, 500)
		p1 := addSideFee(t, f, o, biz.OrderFeePayable, 100)
		p2 := addSideFee(t, f, o, biz.OrderFeePayable, 200)
		base := saveSnapshot(t, f, o.ID)
		good := func() *biz.OrderFeeSaveRowInput { return saveRow(f, biz.OrderFeePayable, "good", p1, 111) }

		// 版本冲突：定位到失败行。
		stale := *p2
		stale.Version = p2.Version + 5
		_, err := save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, good(), saveRow(f, biz.OrderFeePayable, "stale", &stale, 222), saveRow(f, biz.OrderFeePayable, "new", nil, 5))
		requireSaveRowError(t, err, biz.ErrOrderFeeVersionConflict, "stale", "version")
		// 越权税率：无税率权限且税率与目录不同。
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), biz.OrderFeeSavePermissions{CanCreate: true, CanUpdate: true}, good(), saveRow(f, biz.OrderFeePayable, "tax", nil, 5))
		requireSaveRowError(t, err, biz.ErrOrderFeeTaxRateForbidden, "tax", "taxRate")
		// 方向与请求方向不符。
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, good(), saveRow(f, biz.OrderFeeReceivable, "dir", nil, 5))
		requireSaveRowError(t, err, biz.ErrOrderFeeInvalidArgument, "dir", "")
		// 新增行缺 fee.create：修改行仍可，但整体拒绝。
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), biz.OrderFeeSavePermissions{CanUpdate: true, CanEditTaxRate: true}, good(), saveRow(f, biz.OrderFeePayable, "noc", nil, 5))
		requireSaveRowError(t, err, biz.ErrPermissionDenied, "noc", "")
		// 手工汇率：无手工汇率权限。
		override := decimal.NewFromInt(1)
		withOverride := saveRow(f, biz.OrderFeePayable, "ovr", nil, 5)
		withOverride.Fee.ExchangeRateOverride = &override
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), biz.OrderFeeSavePermissions{CanCreate: true, CanUpdate: true, CanEditTaxRate: true}, good(), withOverride)
		requireSaveRowError(t, err, biz.ErrOrderFeeExchangeRateOverrideForbidden, "ovr", "exchangeRateOverride")
		// 已建账：先对 p2 建账后再保存。
		createOccupiedTestFeeBill(t, f, o, p2)
		if base2 := saveSnapshot(t, f, o.ID); base2 == base {
			// 建账不改费用行，只断言快照在后续失败后保持不变。
			base = base2
		}
		billedP2, _ := NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, p2.ID)
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, good(), saveRow(f, biz.OrderFeePayable, "billed", billedP2, 333))
		requireSaveRowError(t, err, biz.ErrOrderFeeBillOccupied, "billed", "")
		// 多行同时出错：返回第一条并在 row_errors 列出全部。
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), biz.OrderFeeSavePermissions{CanCreate: true, CanUpdate: true}, saveRow(f, biz.OrderFeePayable, "t1", nil, 5), saveRow(f, biz.OrderFeePayable, "t2", nil, 6))
		var ke *kratoserrors.Error
		if !errors.As(err, &ke) || !strings.Contains(ke.Metadata["row_errors"], `"t1"`) || !strings.Contains(ke.Metadata["row_errors"], `"t2"`) {
			t.Fatalf("应在 row_errors 列出全部行错误: %v", err)
		}
		if got := saveSnapshot(t, f, o.ID); got != base {
			t.Fatalf("失败后必须零部分写入:\n前=%s\n后=%s", base, got)
		}
	})

	t.Run("手工汇率币种集合按提交后整单集合判定", func(t *testing.T) {
		o := f.createOrder("SAVE-RATE-" + f.suffix)
		override := decimal.NewFromInt(1)
		mk := func(id string) *biz.OrderFeeSaveRowInput {
			row := saveRow(f, biz.OrderFeePayable, id, nil, 50)
			row.Fee.ExchangeRateOverride = &override
			return row
		}
		// 订单尚无应收：提交后应付 {CNY} 对应收 {} 不一致。
		base := saveSnapshot(t, f, o.ID)
		_, err := save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, mk("a"))
		requireSaveRowError(t, err, biz.ErrOrderFeeExchangeRateOverrideMismatch, "a", "exchangeRateOverride")
		if got := saveSnapshot(t, f, o.ID); got != base {
			t.Fatal("集合不满足时必须零写入")
		}
		// 事务外已落库的应付为空，但应收 {CNY}；提交后应付 {CNY} 与之一致，应通过且来源为手工。
		addSideFee(t, f, o, biz.OrderFeeReceivable, 500)
		result, err := save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, mk("b"))
		if err != nil || len(result.Fees) != 1 || result.Fees[0].ExchangeRateSource != biz.ExchangeRateSourceManual {
			t.Fatalf("提交后集合一致应通过并记手工来源: %+v %v", result, err)
		}
		// 修改既有应付币种使提交后集合失配：新增 USD 后改为 USD 覆盖汇率 7，应收仍只有 CNY。
		if _, createErr := f.data.db.Currency.Create().SetCode("USD").SetName("美元").Save(f.ctx); createErr != nil && !ent.IsConstraintError(createErr) {
			t.Fatal(createErr)
		}
		seven := decimal.NewFromInt(7)
		toUSD := saveRow(f, biz.OrderFeePayable, "c", result.Fees[0], 50)
		toUSD.Fee.Currency = "USD"
		toUSD.Fee.ExchangeRateOverride = &seven
		before := saveSnapshot(t, f, o.ID)
		_, err = save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, toUSD)
		requireSaveRowError(t, err, biz.ErrOrderFeeExchangeRateOverrideMismatch, "c", "exchangeRateOverride")
		if got := saveSnapshot(t, f, o.ID); got != before {
			t.Fatal("币种集合失配必须零写入")
		}
	})

	t.Run("响应丢失后沿用 request_key 重试不重复新增", func(t *testing.T) {
		o := f.createOrder("SAVE-REPLAY-" + f.suffix)
		p1 := addSideFee(t, f, o, biz.OrderFeePayable, 100)
		key := uuid.NewString()
		rows := func() []*biz.OrderFeeSaveRowInput {
			return []*biz.OrderFeeSaveRowInput{saveRow(f, biz.OrderFeePayable, "p1", p1, 120), saveRow(f, biz.OrderFeePayable, "n1", nil, 30), saveRow(f, biz.OrderFeePayable, "n2", nil, 40)}
		}
		first, err := save(o, biz.OrderFeePayable, key, allSavePerms, rows()...)
		if err != nil || first.Replayed || len(first.Fees) != 3 {
			t.Fatalf("首次保存: %+v %v", first, err)
		}
		afterFirst := saveSnapshot(t, f, o.ID)
		again, err := save(o, biz.OrderFeePayable, key, allSavePerms, rows()...)
		if err != nil || !again.Replayed || len(again.Fees) != 3 || again.FeeContentVersion != first.FeeContentVersion {
			t.Fatalf("重试应视为成功回放: %+v %v", again, err)
		}
		if got := saveSnapshot(t, f, o.ID); got != afterFirst {
			t.Fatalf("回放不得产生任何新写入:\n%s\n%s", afterFirst, got)
		}
		// 同 request_key、同行标识但内容不同：意图冲突。
		changed := rows()
		changed[1].Fee.UnitPrice = decimal.NewFromInt(31)
		_, err = save(o, biz.OrderFeePayable, key, allSavePerms, changed...)
		requireSaveRowError(t, err, biz.ErrOrderFeeIdempotencyConflict, "n1", "")
		// 半应用：同 request_key 多出一条未落库的新增行，不重复新增也不猜测。
		_, err = save(o, biz.OrderFeePayable, key, allSavePerms, append(rows(), saveRow(f, biz.OrderFeePayable, "n3", nil, 9))...)
		if !kratoserrors.Is(err, biz.ErrOrderFeeSavePartiallyApplied) {
			t.Fatalf("半应用应返回 409: %v", err)
		}
		if got := saveSnapshot(t, f, o.ID); got != afterFirst {
			t.Fatal("半应用冲突不得写入")
		}
	})

	t.Run("并发修改同一行只有一次成功", func(t *testing.T) {
		o := f.createOrder("SAVE-RACE-" + f.suffix)
		addSideFee(t, f, o, biz.OrderFeeReceivable, 500)
		p1 := addSideFee(t, f, o, biz.OrderFeePayable, 100)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range errs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				_, errs[i] = save(o, biz.OrderFeePayable, uuid.NewString(), allSavePerms, saveRow(f, biz.OrderFeePayable, "p1", p1, int64(200+i)))
			}(i)
		}
		close(start)
		wg.Wait()
		success, conflict := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				success++
			case kratoserrors.Is(err, biz.ErrOrderFeeVersionConflict):
				conflict++
			default:
				t.Fatalf("意外错误: %v", err)
			}
		}
		if success != 1 || conflict != 1 {
			t.Fatalf("应一成功一版本冲突: %v", errs)
		}
		got, err := NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, p1.ID)
		if err != nil || got.Version != p1.Version+1 {
			t.Fatalf("版本只应递增一次: %+v %v", got, err)
		}
	})
}

// 已确认并已付的提成：整表保存的一次提交只做一次提成经济重算，已付事实保持，系统差额按合并后的成本给出。
func TestOrderFeeSaveCommissionReconcileOncePostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("SAVECOMM-" + f.suffix)
	f.approveInitialFees(o)
	parent := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	cuc := biz.NewCommissionUsecase(NewCommissionRepo(f.data), nil, f.data)
	if _, err := markCommissionPaidForTest(f.ctx, cuc, f.organizationID, f.approverID, parent.ID, parent.Version); err != nil {
		t.Fatal(err)
	}
	unlockFeeTestOrder(t, f, o)
	uc := normalFeeUsecaseForRollback(f)
	result, err := uc.Save(f.ctx, f.organizationID, f.approverID, o.ID, biz.OrderFeePayable, uuid.NewString(),
		[]*biz.OrderFeeSaveRowInput{saveRow(f, biz.OrderFeePayable, "a", nil, 100), saveRow(f, biz.OrderFeePayable, "b", nil, 50)}, allSavePerms)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fees) != 3 {
		t.Fatalf("应有基础应付与两条新增: %d", len(result.Fees))
	}
	rows, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustmentent.CommissionIDEQ(parent.ID)).All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	net := decimal.RequireFromString("60")
	for _, row := range rows {
		amount := decimal.RequireFromString(row.Amount)
		if row.Direction == adjustmentent.DirectionDECREASE {
			net = net.Sub(amount)
		} else {
			net = net.Add(amount)
		}
	}
	// 新增成本 150、比例 10% → 追回 15，净额 45；一次重算只生成一条系统差额。
	if !net.Equal(decimal.RequireFromString("45")) || len(rows) != 1 {
		t.Fatalf("提成净额=%s 调整条数=%d，预期 45 / 1", net, len(rows))
	}
	stored, err := f.data.db.FinanceCommission.Get(f.ctx, parent.ID)
	if err != nil || stored.Status != "PAID" || !decimal.RequireFromString(stored.CommissionAmount).Equal(decimal.RequireFromString(parent.CommissionAmount)) {
		t.Fatalf("已付提成事实不得改变: %+v %v", stored, err)
	}
}
