package data

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// billFeeUpdateFixture 在账单事务夹具之上提供账单费用增删所需的费用、账单与并发持锁工具。
type billFeeUpdateFixture struct {
	*feeLedgerPostgresFixture
	ctx        context.Context
	repo       biz.FinanceBillRepo
	usecase    *biz.FinanceBillUsecase
	scope      []uuid.UUID
	invoiceIDs []uuid.UUID
}

type billFeeSpec struct {
	key       string
	direction orderfeeent.Direction
	currency  string
	partyID   uuid.UUID
	orderID   uuid.UUID
	total     string
	rate      string
	taxRate   string
	noBilling bool
}

func newBillFeeUpdateFixture(t *testing.T, data *Data) *billFeeUpdateFixture {
	t.Helper()
	ledger := newFeeLedgerPostgresFixture(t, data)
	repo := NewFinanceBillRepo(data)
	f := &billFeeUpdateFixture{feeLedgerPostgresFixture: ledger, ctx: context.Background(), repo: repo, usecase: ledger.newUsecase(repo), scope: []uuid.UUID{ledger.organizationID}}
	// 后注册先执行：发票关联引用账单，需先于夹具删除账单。
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := data.db.FinanceInvoiceBill.Delete().Where(financeinvoicebillent.InvoiceIDIn(f.invoiceIDs...)).Exec(cctx); err != nil {
			t.Errorf("清理发票账单关联: %v", err)
		}
		if _, err := data.db.FinanceInvoice.Delete().Where(financeinvoiceent.IDIn(f.invoiceIDs...)).Exec(cctx); err != nil {
			t.Errorf("清理发票: %v", err)
		}
	})
	return f
}

func (f *billFeeUpdateFixture) fee(spec billFeeSpec) uuid.UUID {
	f.t.Helper()
	if spec.direction == "" {
		spec.direction = orderfeeent.DirectionRECEIVABLE
	}
	if spec.currency == "" {
		spec.currency = "CNY"
	}
	if spec.partyID == uuid.Nil {
		spec.partyID = f.partnerID
	}
	if spec.orderID == uuid.Nil {
		spec.orderID = f.orderID
	}
	if spec.rate == "" {
		spec.rate = "1"
	}
	total := decimal.RequireFromString(spec.total)
	rate := decimal.RequireFromString(spec.rate)
	create := f.data.db.OrderFee.Create().
		SetOrderID(spec.orderID).SetIdempotencyKey("fee-bill-update-" + spec.key + "-" + f.suffix).
		SetDirection(spec.direction).SetFeeCode("FEE_" + spec.key).SetFeeName("账单增删费用-" + spec.key).
		SetSettlementPartyID(spec.partyID).SetBillingUnit("票").SetQuantity("1.0000").SetUnitPrice(total.StringFixed(4)).
		SetTotalAmount(total.StringFixed(8)).SetNetAmount(total.StringFixed(8)).SetTaxAmount("0.00000000").
		SetCurrency(spec.currency).SetExchangeRate(rate.StringFixed(8)).SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).SetBaseCurrency("CNY").SetBaseCurrencyAmount(total.Mul(rate).RoundBank(8).StringFixed(8)).
		SetAccountingRequired(!spec.noBilling).SetVersion(1)
	if spec.taxRate != "" {
		create.SetTaxRate(spec.taxRate)
	}
	fee, err := create.Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建费用 %s: %v", spec.key, err)
	}
	return fee.ID
}

func (f *billFeeUpdateFixture) bill(key string, accountID uuid.UUID, feeIDs ...uuid.UUID) *biz.FinanceBill {
	f.t.Helper()
	bill, err := f.usecase.Create(f.ctx, f.organizationID, f.actorID, biz.CreateFinanceBillInput{FeeIDs: feeIDs, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-fee-update-" + key + "-" + f.suffix, SettlementAccountID: accountID})
	if err != nil {
		f.t.Fatalf("建账 %s: %v", key, err)
	}
	return bill
}

func (f *billFeeUpdateFixture) update(billID uuid.UUID, version uint64, add, remove []uuid.UUID) (*biz.FinanceBill, error) {
	return f.usecase.UpdateFees(f.ctx, f.scope, f.actorID, biz.UpdateFinanceBillFeesInput{ID: billID, ExpectedVersion: version, AddFeeIDs: add, RemoveFeeIDs: remove})
}

func (f *billFeeUpdateFixture) updateAsync(billID uuid.UUID, version uint64, add, remove []uuid.UUID) <-chan error {
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := f.usecase.UpdateFees(ctx, f.scope, f.actorID, biz.UpdateFinanceBillFeesInput{ID: billID, ExpectedVersion: version, AddFeeIDs: add, RemoveFeeIDs: remove})
		done <- err
	}()
	return done
}

func (f *billFeeUpdateFixture) entBill(id uuid.UUID) *ent.FinanceBill {
	f.t.Helper()
	bill, err := f.data.db.FinanceBill.Get(f.ctx, id)
	if err != nil {
		f.t.Fatalf("读取账单: %v", err)
	}
	return bill
}

// requireBillConsistent 断言账单头与有效行一致：fee_count = 有效行数，原币合计相等，
// 头本位币 = 总额 × 账单汇率（RoundBank(8)），行本位币合计等于头（末行吸收尾差），行汇率均为账单汇率。
func (f *billFeeUpdateFixture) requireBillConsistent(billID uuid.UUID, wantFees ...uuid.UUID) *ent.FinanceBill {
	f.t.Helper()
	bill := f.entBill(billID)
	lines, err := f.data.db.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(billID), financebilllineent.ActiveEQ(true)).All(f.ctx)
	if err != nil {
		f.t.Fatalf("读取账单行: %v", err)
	}
	if bill.FeeCount != len(lines) || len(lines) != len(wantFees) {
		f.t.Fatalf("fee_count=%d 有效行=%d 期望费用=%d", bill.FeeCount, len(lines), len(wantFees))
	}
	want := make(map[uuid.UUID]bool, len(wantFees))
	for _, id := range wantFees {
		want[id] = true
	}
	rate := decimal.RequireFromString(bill.ExchangeRate)
	total, base := decimal.Zero, decimal.Zero
	for _, line := range lines {
		if !want[line.OrderFeeID] {
			f.t.Fatalf("账单含非预期费用 %s", line.OrderFeeID)
		}
		if !decimal.RequireFromString(line.ExchangeRate).Equal(rate) {
			f.t.Fatalf("行汇率 %s 不等于账单汇率 %s", line.ExchangeRate, bill.ExchangeRate)
		}
		total = total.Add(decimal.RequireFromString(line.TotalAmount))
		base = base.Add(decimal.RequireFromString(line.BaseCurrencyAmount))
	}
	headTotal := decimal.RequireFromString(bill.TotalAmount)
	headBase := decimal.RequireFromString(bill.BaseCurrencyAmount)
	if !total.Equal(headTotal) || !base.Equal(headBase) || !headBase.Equal(headTotal.Mul(rate).RoundBank(8)) {
		f.t.Fatalf("金额不一致: 行合计 %s/%s 头 %s/%s 汇率 %s", total, base, headTotal, headBase, rate)
	}
	return bill
}

// heldTransaction 在事务内执行 hold 后保持锁直到 release，用于构造确定的并发先后顺序。
type heldTransaction struct {
	locked  chan error
	release chan struct{}
	done    chan error
}

func holdTransaction(data *Data, hold func(ctx context.Context, tx *ent.Tx) error) *heldTransaction {
	h := &heldTransaction{locked: make(chan error, 1), release: make(chan struct{}), done: make(chan error, 1)}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		h.done <- data.WithTx(ctx, func(tx *ent.Tx) error {
			err := hold(ctx, tx)
			h.locked <- err
			if err != nil {
				return err
			}
			<-h.release
			return nil
		})
	}()
	return h
}

// holdBillFeeUpdate 在共享事务内完成增删费用后保持全部锁，模拟「增删先获锁」。
func (f *billFeeUpdateFixture) holdBillFeeUpdate(billID uuid.UUID, version uint64, add, remove []uuid.UUID) *heldTransaction {
	h := &heldTransaction{locked: make(chan error, 1), release: make(chan struct{}), done: make(chan error, 1)}
	audit := &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.actorID, Action: "finance.bill.fees.update", Result: "success", ResourceType: "finance_bill", ResourceID: billID.String()}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		h.done <- f.data.WithinTransaction(ctx, func(txCtx context.Context) error {
			err := f.repo.UpdateFees(txCtx, f.scope, biz.UpdateFinanceBillFeesInput{ID: billID, ExpectedVersion: version, AddFeeIDs: add, RemoveFeeIDs: remove}, audit)
			h.locked <- err
			if err != nil {
				return err
			}
			<-h.release
			return nil
		})
	}()
	return h
}

func (h *heldTransaction) waitLocked(t *testing.T) {
	t.Helper()
	select {
	case err := <-h.locked:
		if err != nil {
			t.Fatalf("持锁事务失败: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("持锁事务未就绪")
	}
}

func (h *heldTransaction) commit(t *testing.T) {
	t.Helper()
	close(h.release)
	select {
	case err := <-h.done:
		if err != nil {
			t.Fatalf("持锁事务提交失败: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("持锁事务未提交")
	}
}

func requireBlocked(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("应等待对方锁，实际已返回: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
}

func awaitResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("等待并发结果超时")
		return nil
	}
}

func lockBillLikeFinanceWriter(ctx context.Context, tx *ent.Tx, organizationID, billID uuid.UUID) (*ent.FinanceBill, error) {
	if err := lockExistingBillOrders(ctx, tx.Client(), []uuid.UUID{organizationID}, []uuid.UUID{billID}); err != nil {
		return nil, err
	}
	return tx.FinanceBill.Get(ctx, billID)
}

// TestFinanceBillFeeUpdatePostgres 验证账单费用增删：候选口径、篡改请求拒绝、金额与尾差重算、
// 移除释放与重新建账、移空与版本冲突，以及与发票、核销、对冲、取消、汇率联动的并发复核。
func TestFinanceBillFeeUpdatePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	t.Cleanup(cleanupData)

	t.Run("候选只含同公司同方向同币种同结算对象的未建账费用", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		f.createExchangeRateSetting("7.20000000")
		otherParty, err := data.db.Partner.Create().SetOrganizationID(f.organizationID).SetCode("OTHER-" + f.suffix).SetLegalName("其他客户-" + f.suffix).SetNormalizedName("其他客户-" + f.suffix).Save(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		other := newBillFeeUpdateFixture(t, data)
		base := f.fee(billFeeSpec{key: "base", total: "100"})
		eligibleA := f.fee(billFeeSpec{key: "eligible-a", total: "10"})
		eligibleB := f.fee(billFeeSpec{key: "eligible-b", total: "20"})
		f.fee(billFeeSpec{key: "usd", currency: "USD", rate: "7.2", total: "10"})
		f.fee(billFeeSpec{key: "payable", direction: orderfeeent.DirectionPAYABLE, total: "10"})
		f.fee(billFeeSpec{key: "other-party", partyID: otherParty.ID, total: "10"})
		f.fee(billFeeSpec{key: "no-billing", total: "10", noBilling: true})
		occupied := f.fee(billFeeSpec{key: "occupied", total: "10"})
		f.bill("occupied", f.accountID, occupied)
		other.fee(billFeeSpec{key: "foreign", total: "10"})
		bill := f.bill("candidate", f.accountID, base)

		result, err := f.usecase.ListFeeCandidates(f.ctx, f.scope, bill.ID, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 200})
		if err != nil {
			t.Fatal(err)
		}
		got := make([]uuid.UUID, 0, len(result.Items))
		for _, item := range result.Items {
			if !item.CanBill {
				t.Fatalf("候选 %s 应可加入: %v", item.Fee.FeeCode, item.BlockedReasons)
			}
			got = append(got, item.Fee.ID)
		}
		if result.Total != 2 || !equalUUIDs(sortedUUIDs(got), sortedUUIDs([]uuid.UUID{eligibleA, eligibleB})) {
			t.Fatalf("候选=%v 总数=%d，期望仅 %s、%s", got, result.Total, eligibleA, eligibleB)
		}
		page, err := f.usecase.ListFeeCandidates(f.ctx, f.scope, bill.ID, biz.FinanceBillFeeCandidateFilter{Page: 2, PageSize: 1})
		if err != nil || page.Total != 2 || len(page.Items) != 1 {
			t.Fatalf("分页结果=%v（%v）", page, err)
		}
		keyword, err := f.usecase.ListFeeCandidates(f.ctx, f.scope, bill.ID, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 20, Keyword: "fee_eligible-b"})
		if err != nil || keyword.Total != 1 || keyword.Items[0].Fee.ID != eligibleB {
			t.Fatalf("关键字过滤结果=%v（%v）", keyword, err)
		}
		if _, err := f.usecase.ListFeeCandidates(f.ctx, f.scope, bill.ID, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 201}); !errors.Is(err, biz.ErrFinanceBillInvalidArgument) {
			t.Fatalf("页容量 201 应拒绝: %v", err)
		}
		if _, err := f.usecase.ListFeeCandidates(f.ctx, other.scope, bill.ID, biz.FinanceBillFeeCandidateFilter{Page: 1, PageSize: 20}); !errors.Is(err, biz.ErrFinanceBillNotFound) {
			t.Fatalf("其他公司不可见账单: %v", err)
		}
	})

	t.Run("篡改请求的跨币种跨对象跨公司已建账及非本账单费用均被拒", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		f.createExchangeRateSetting("7.20000000")
		otherParty, err := data.db.Partner.Create().SetOrganizationID(f.organizationID).SetCode("OTHER-" + f.suffix).SetLegalName("其他客户-" + f.suffix).SetNormalizedName("其他客户-" + f.suffix).Save(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		other := newBillFeeUpdateFixture(t, data)
		base := f.fee(billFeeSpec{key: "base", total: "100"})
		bill := f.bill("tamper", f.accountID, base)
		occupied := f.fee(billFeeSpec{key: "occupied", total: "10"})
		f.bill("occupied", f.accountID, occupied)
		unbilled := f.fee(billFeeSpec{key: "unbilled", total: "10"})
		cases := []struct {
			name   string
			add    []uuid.UUID
			remove []uuid.UUID
			want   error
		}{
			{"不同币种", []uuid.UUID{f.fee(billFeeSpec{key: "usd", currency: "USD", rate: "7.2", total: "10"})}, nil, biz.ErrFinanceBillFeeMismatch},
			{"不同方向", []uuid.UUID{f.fee(billFeeSpec{key: "payable", direction: orderfeeent.DirectionPAYABLE, total: "10"})}, nil, biz.ErrFinanceBillFeeMismatch},
			{"不同结算对象", []uuid.UUID{f.fee(billFeeSpec{key: "other-party", partyID: otherParty.ID, total: "10"})}, nil, biz.ErrFinanceBillFeeMismatch},
			{"无需建账", []uuid.UUID{f.fee(billFeeSpec{key: "no-billing", total: "10", noBilling: true})}, nil, biz.ErrFinanceBillFeeInvalid},
			{"跨公司", []uuid.UUID{other.fee(billFeeSpec{key: "foreign", total: "10"})}, nil, biz.ErrFinanceBillFeeInvalid},
			{"已被其他账单占用", []uuid.UUID{occupied}, nil, biz.ErrFinanceBillFeeInvalid},
			{"移除非本账单费用", nil, []uuid.UUID{unbilled}, biz.ErrFinanceBillFeeInvalid},
			{"合法与非法混合整体回滚", []uuid.UUID{unbilled, occupied}, nil, biz.ErrFinanceBillFeeInvalid},
		}
		for _, item := range cases {
			if _, err := f.update(bill.ID, 1, item.add, item.remove); !errors.Is(err, item.want) {
				t.Fatalf("%s: 结果=%v，期望 %v", item.name, err, item.want)
			}
		}
		current := f.requireBillConsistent(bill.ID, base)
		if current.Version != 1 {
			t.Fatalf("拒绝的请求不得推进版本: %d", current.Version)
		}
		fee, err := data.db.OrderFee.Get(f.ctx, unbilled)
		if err != nil || fee.Version != 1 {
			t.Fatalf("回滚后费用版本=%v（%v）", fee, err)
		}
	})

	t.Run("增删按账单汇率重算尾差且移除只释放关联可重新建账", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		f.createExchangeRateSetting("7.12350000")
		first := f.fee(billFeeSpec{key: "u1", currency: "USD", rate: "7", total: "100.01000001", taxRate: "0.0600"})
		second := f.fee(billFeeSpec{key: "u2", currency: "USD", rate: "7", total: "33.33333333"})
		third := f.fee(billFeeSpec{key: "u3", currency: "USD", rate: "6.5", total: "0.07000007"})
		bill := f.bill("rounding", f.usdAccountID, first, second)
		before, err := data.db.OrderFee.Get(f.ctx, first)
		if err != nil {
			t.Fatal(err)
		}

		updated, err := f.update(bill.ID, 1, []uuid.UUID{third}, []uuid.UUID{first})
		if err != nil {
			t.Fatalf("增删费用: %v", err)
		}
		activeLines := 0
		for _, line := range updated.Lines {
			if line.Active {
				activeLines++
			}
		}
		// 响应保留历史停用行（active=false）供追溯，fee_count 只计有效行。
		if updated.Version != 2 || updated.FeeCount != 2 || activeLines != 2 || len(updated.Lines) != 3 {
			t.Fatalf("响应版本/费用数/有效行/全部行=%d/%d/%d/%d", updated.Version, updated.FeeCount, activeLines, len(updated.Lines))
		}
		head := f.requireBillConsistent(bill.ID, second, third)
		if !decimal.RequireFromString(head.TotalAmount).Equal(decimal.RequireFromString("33.40333340")) || !decimal.RequireFromString(head.ExchangeRate).Equal(decimal.RequireFromString("7.1235")) {
			t.Fatalf("账单头总额=%s 汇率=%s", head.TotalAmount, head.ExchangeRate)
		}
		// 移除的费用只解除关联：金额、税率、结算对象、汇率保持原值，版本 +1。
		after, err := data.db.OrderFee.Get(f.ctx, first)
		if err != nil {
			t.Fatal(err)
		}
		if after.TotalAmount != before.TotalAmount || !sameOptionalDecimalString(after.TaxRate, before.TaxRate) || after.SettlementPartyID != before.SettlementPartyID || after.ExchangeRate != before.ExchangeRate || after.BaseCurrencyAmount != before.BaseCurrencyAmount || after.Version != before.Version+1 {
			t.Fatalf("移除后费用字段变化: before=%#v after=%#v", before, after)
		}
		historical, err := data.db.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(bill.ID), financebilllineent.OrderFeeIDEQ(first)).Only(f.ctx)
		if err != nil || historical.Active {
			t.Fatalf("移除行应保留为历史停用行: %#v（%v）", historical, err)
		}
		added, err := data.db.OrderFee.Get(f.ctx, third)
		if err != nil || added.Version != 2 {
			t.Fatalf("加入费用版本=%v（%v）", added, err)
		}
		audits, err := data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("finance.bill.fees.update")).Count(f.ctx)
		if err != nil || audits != 1 {
			t.Fatalf("增删审计数=%d（%v）", audits, err)
		}
		rebilled := f.bill("rebill", f.usdAccountID, first)
		f.requireBillConsistent(rebilled.ID, first)

		// 重新加入曾移除的费用会生成新行，历史行保持停用。
		rebill := f.fee(billFeeSpec{key: "u4", currency: "USD", rate: "7", total: "1"})
		if _, err := f.update(bill.ID, 2, []uuid.UUID{rebill}, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := f.update(bill.ID, 3, nil, []uuid.UUID{rebill}); err != nil {
			t.Fatal(err)
		}
		if _, err := f.update(bill.ID, 4, []uuid.UUID{rebill}, nil); err != nil {
			t.Fatalf("重新加入曾移除的费用: %v", err)
		}
		f.requireBillConsistent(bill.ID, second, third, rebill)
		count, err := data.db.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(bill.ID), financebilllineent.OrderFeeIDEQ(rebill)).Count(f.ctx)
		if err != nil || count != 2 {
			t.Fatalf("重新加入应保留历史行并新增一行: %d（%v）", count, err)
		}
	})

	t.Run("移除最后一条被拒且版本冲突返回409", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		only := f.fee(billFeeSpec{key: "only", total: "100"})
		replacement := f.fee(billFeeSpec{key: "replacement", total: "40"})
		bill := f.bill("last", f.accountID, only)
		if _, err := f.update(bill.ID, 1, nil, []uuid.UUID{only}); !errors.Is(err, biz.ErrFinanceBillLastFee) {
			t.Fatalf("移除最后一条应拒绝: %v", err)
		}
		_, err := f.update(bill.ID, 9, []uuid.UUID{replacement}, nil)
		if !errors.Is(err, biz.ErrFinanceBillVersionConflict) || biz.ErrFinanceBillVersionConflict.Code != 409 {
			t.Fatalf("版本冲突应返回 409: %v", err)
		}
		// 同时移除最后一条并加入另一条：账单仍保留一条有效费用，允许。
		if _, err := f.update(bill.ID, 1, []uuid.UUID{replacement}, []uuid.UUID{only}); err != nil {
			t.Fatalf("替换唯一费用: %v", err)
		}
		f.requireBillConsistent(bill.ID, replacement)
	})

	t.Run("两账单并发抢占同一费用只有一方成功", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		x := f.bill("x", f.accountID, f.fee(billFeeSpec{key: "x", total: "10"}))
		y := f.bill("y", f.accountID, f.fee(billFeeSpec{key: "y", total: "20"}))
		contested := f.fee(billFeeSpec{key: "contested", total: "5"})
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, bill := range []*biz.FinanceBill{x, y} {
			billID := bill.ID
			go func() {
				<-start
				_, err := f.update(billID, 1, []uuid.UUID{contested}, nil)
				results <- err
			}()
		}
		close(start)
		successes, conflicts := 0, 0
		for range 2 {
			err := awaitResult(t, results)
			switch {
			case err == nil:
				successes++
			case errors.Is(err, biz.ErrFinanceBillFeeInvalid):
				conflicts++
			default:
				t.Fatalf("非预期结果: %v", err)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("成功=%d 冲突=%d，期望各 1", successes, conflicts)
		}
		active, err := data.db.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(contested), financebilllineent.ActiveEQ(true)).Count(f.ctx)
		if err != nil || active != 1 {
			t.Fatalf("费用有效行数=%d（%v）", active, err)
		}
	})

	t.Run("增删与发票登记并发复核", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		bill := f.bill("invoice", f.accountID, f.fee(billFeeSpec{key: "base", total: "100"}))
		extra := f.fee(billFeeSpec{key: "extra", total: "10"})
		invoice, err := data.db.FinanceInvoice.Create().
			SetOrganizationID(f.organizationID).SetRecordNo("INV-FEE-" + f.suffix).SetIdempotencyKey("inv-fee-" + f.suffix).
			SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetInvoiceType(financeinvoiceent.InvoiceTypeDIGITAL_NORMAL).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("增删发票").SetCurrency("CNY").SetBaseCurrency("CNY").
			SetTotalAmount("100.00000000").SetNetAmount("100.00000000").SetTaxAmount("0").SetBillCount(1).SetStatus(financeinvoiceent.StatusDRAFT).Save(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		f.invoiceIDs = append(f.invoiceIDs, invoice.ID)

		// 增删先获锁：发票侧按订单锁等待，锁后重读账单行集合已变，以冲突失败而非在旧集合上登记。
		held := f.holdBillFeeUpdate(bill.ID, 1, []uuid.UUID{extra}, nil)
		held.waitLocked(t)
		invoiceDone := make(chan error, 1)
		go func() {
			invoiceDone <- data.WithTx(context.Background(), func(tx *ent.Tx) error {
				_, lockErr := lockBillLikeFinanceWriter(context.Background(), tx, f.organizationID, bill.ID)
				return lockErr
			})
		}()
		requireBlocked(t, invoiceDone)
		held.commit(t)
		if err := awaitResult(t, invoiceDone); !errors.Is(err, biz.ErrFinanceBillVersionConflict) {
			t.Fatalf("发票侧应看到费用集合变化: %v", err)
		}

		// 发票先获锁：增删等待后在锁内看到有效发票关联，拒绝。
		hold := holdTransaction(data, func(ctx context.Context, tx *ent.Tx) error {
			locked, lockErr := lockBillLikeFinanceWriter(ctx, tx, f.organizationID, bill.ID)
			if lockErr != nil {
				return lockErr
			}
			_, createErr := tx.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(locked.ID).SetBillNo(locked.BillNo).SetAmount("110.00000000").SetTaxAmount("0").SetActive(true).Save(ctx)
			return createErr
		})
		hold.waitLocked(t)
		done := f.updateAsync(bill.ID, 2, nil, []uuid.UUID{extra})
		requireBlocked(t, done)
		hold.commit(t)
		if err := awaitResult(t, done); !errors.Is(err, biz.ErrFinanceBillFeesLocked) {
			t.Fatalf("登记发票后增删应锁定: %v", err)
		}
		// 发票作废后恢复增删。
		invoice.Update().SetStatus(financeinvoiceent.StatusCANCELLED).SetCancelledAt(time.Now()).SetCancelledBy(f.actorID).SetCancellationReason("测试").ExecX(f.ctx)
		data.db.FinanceInvoiceBill.Update().Where(financeinvoicebillent.InvoiceIDEQ(invoice.ID)).SetActive(false).ExecX(f.ctx)
		if _, err := f.update(bill.ID, 2, nil, []uuid.UUID{extra}); err != nil {
			t.Fatalf("发票全部作废后应恢复增删: %v", err)
		}
	})

	t.Run("增删与核销并发复核", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		bill := f.bill("verification", f.accountID, f.fee(billFeeSpec{key: "base", total: "100"}))
		extra := f.fee(billFeeSpec{key: "extra", total: "10"})

		// 增删先获锁：核销侧按订单锁等待，锁后读到增删后的账单金额。
		held := f.holdBillFeeUpdate(bill.ID, 1, []uuid.UUID{extra}, nil)
		held.waitLocked(t)
		totals := make(chan string, 1)
		verifyDone := make(chan error, 1)
		go func() {
			verifyDone <- data.WithTx(context.Background(), func(tx *ent.Tx) error {
				if lockErr := lockOrdersForFinanceReversal(context.Background(), tx, []uuid.UUID{f.orderID}); lockErr != nil {
					return lockErr
				}
				locked, queryErr := tx.FinanceBill.Query().Where(financebillent.IDEQ(bill.ID)).ForUpdate().Only(context.Background())
				if queryErr != nil {
					return queryErr
				}
				totals <- locked.TotalAmount
				return nil
			})
		}()
		requireBlocked(t, verifyDone)
		held.commit(t)
		if err := awaitResult(t, verifyDone); err != nil {
			t.Fatal(err)
		}
		if total := <-totals; !decimal.RequireFromString(total).Equal(decimal.RequireFromString("110")) {
			t.Fatalf("核销侧读取金额=%s，期望增删后的 110", total)
		}

		// 核销先获锁：增删等待后看到有效核销，拒绝。
		cash := f.createConfirmedCashflow("verification", "50", financeBillIntegrationDate)
		verification := f.createActiveVerification("verification", "50", nil)
		hold := holdTransaction(data, func(ctx context.Context, tx *ent.Tx) error {
			locked, lockErr := lockBillLikeFinanceWriter(ctx, tx, f.organizationID, bill.ID)
			if lockErr != nil {
				return lockErr
			}
			_, createErr := tx.FinanceVerificationAllocation.Create().SetVerificationID(verification.ID).SetCashflowID(cash.ID).SetBillID(locked.ID).
				SetCashflowNo(cash.FlowNo).SetBillNo(locked.BillNo).SetAmount("50.00000000").SetBillBaseAmount("50.00000000").SetCashflowBaseAmount("50.00000000").
				SetExchangeGainLoss("0.00000000").SetActive(true).Save(ctx)
			return createErr
		})
		hold.waitLocked(t)
		done := f.updateAsync(bill.ID, 2, nil, []uuid.UUID{extra})
		requireBlocked(t, done)
		hold.commit(t)
		if err := awaitResult(t, done); !errors.Is(err, biz.ErrFinanceBillFeesLocked) {
			t.Fatalf("核销后增删应锁定: %v", err)
		}
	})

	t.Run("增删与对冲确认并发复核", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		bill := f.bill("netting", f.accountID, f.fee(billFeeSpec{key: "base", total: "100"}))
		extra := f.fee(billFeeSpec{key: "extra", total: "10"})

		// 增删先获锁：对冲侧（LockNettingBills）等待后读取增删后的账单金额与版本。
		held := f.holdBillFeeUpdate(bill.ID, 1, []uuid.UUID{extra}, nil)
		held.waitLocked(t)
		nettingRepo := NewFinanceNettingRepo(data)
		locked := make(chan []*biz.FinanceNettingBill, 1)
		nettingDone := make(chan error, 1)
		go func() {
			nettingDone <- data.WithinTransaction(context.Background(), func(txCtx context.Context) error {
				bills, lockErr := nettingRepo.LockNettingBills(txCtx, f.organizationID, []uuid.UUID{bill.ID})
				locked <- bills
				return lockErr
			})
		}()
		requireBlocked(t, nettingDone)
		held.commit(t)
		if err := awaitResult(t, nettingDone); err != nil {
			t.Fatal(err)
		}
		if bills := <-locked; len(bills) != 1 || !bills[0].TotalAmount.Equal(decimal.RequireFromString("110")) || bills[0].Version != 2 {
			t.Fatalf("对冲侧读取=%#v", bills)
		}

		// 草稿对冲同样阻止增删；对冲确认先获锁时增删等待后看到已确认对冲，拒绝。
		entBill := f.entBill(bill.ID)
		draft, _ := f.createNetting("draft", financenettingent.StatusDRAFT, entBill, "10", false)
		if _, err := f.update(bill.ID, 2, nil, []uuid.UUID{extra}); !errors.Is(err, biz.ErrFinanceBillFeesLocked) {
			t.Fatalf("草稿对冲应阻止增删: %v", err)
		}
		hold := holdTransaction(data, func(ctx context.Context, tx *ent.Tx) error {
			if _, lockErr := lockBillLikeFinanceWriter(ctx, tx, f.organizationID, bill.ID); lockErr != nil {
				return lockErr
			}
			if _, updateErr := tx.FinanceNetting.UpdateOneID(draft.ID).SetStatus(financenettingent.StatusCONFIRMED).SetConfirmedAt(time.Now()).Save(ctx); updateErr != nil {
				return updateErr
			}
			_, updateErr := tx.FinanceNettingAllocation.Update().Where(nettingallocent.NettingIDEQ(draft.ID)).SetActive(true).Save(ctx)
			return updateErr
		})
		hold.waitLocked(t)
		done := f.updateAsync(bill.ID, 2, nil, []uuid.UUID{extra})
		requireBlocked(t, done)
		hold.commit(t)
		err := awaitResult(t, done)
		if !errors.Is(err, biz.ErrFinanceBillFeesLocked) {
			t.Fatalf("对冲确认后增删应锁定: %v", err)
		}
		facts, factErr := billFeeLockFacts(f.ctx, data.db, bill.ID)
		if factErr != nil || !equalFeeLockFacts(facts, []biz.FinanceBillFeeLockFact{biz.FinanceBillFeeLockNetting}) {
			t.Fatalf("锁定事实=%v（%v）", facts, factErr)
		}
		// 撤销对冲（分摊失效、对冲反转）后恢复增删。
		data.db.FinanceNettingAllocation.Update().Where(nettingallocent.NettingIDEQ(draft.ID)).SetActive(false).ExecX(f.ctx)
		data.db.FinanceNetting.UpdateOneID(draft.ID).SetStatus(financenettingent.StatusREVERSED).SetReversedAt(time.Now()).ExecX(f.ctx)
		if _, err := f.update(bill.ID, 2, nil, []uuid.UUID{extra}); err != nil {
			t.Fatalf("对冲撤销后应恢复增删: %v", err)
		}
	})

	t.Run("增删与取消账单并发只有一方成功", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		base := f.fee(billFeeSpec{key: "base", total: "100"})
		extra := f.fee(billFeeSpec{key: "extra", total: "10"})
		bill := f.bill("cancel", f.accountID, base)
		start := make(chan struct{})
		updateDone := make(chan error, 1)
		cancelDone := make(chan error, 1)
		go func() {
			<-start
			_, err := f.update(bill.ID, 1, []uuid.UUID{extra}, nil)
			updateDone <- err
		}()
		go func() {
			<-start
			_, err := f.usecase.Cancel(f.ctx, f.scope, f.actorID, bill.ID, 1, "并发取消")
			cancelDone <- err
		}()
		close(start)
		updateErr, cancelErr := awaitResult(t, updateDone), awaitResult(t, cancelDone)
		if (updateErr == nil) == (cancelErr == nil) {
			t.Fatalf("应只有一方成功: update=%v cancel=%v", updateErr, cancelErr)
		}
		loser := updateErr
		if loser == nil {
			loser = cancelErr
		}
		if !errors.Is(loser, biz.ErrFinanceBillVersionConflict) {
			t.Fatalf("失败方应为版本冲突: %v", loser)
		}
		current := f.entBill(bill.ID)
		if updateErr == nil {
			f.requireBillConsistent(bill.ID, base, extra)
			if _, err := f.usecase.Cancel(f.ctx, f.scope, f.actorID, bill.ID, current.Version, "取消"); err != nil {
				t.Fatal(err)
			}
			current = f.entBill(bill.ID)
		}
		if _, err := f.update(bill.ID, current.Version, []uuid.UUID{extra}, nil); !errors.Is(err, biz.ErrFinanceBillInvalidTransition) {
			t.Fatalf("已取消账单增删应拒绝: %v", err)
		}
	})

	t.Run("增删与费用汇率联动并发以账单汇率重算", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		f.createExchangeRateSetting("7.12350000")
		if _, err := data.db.ExchangeRateSetting.Create().SetOrganizationID(f.organizationID).SetFromCurrency("USD").SetToCurrency("CNY").
			SetEffectiveFrom(time.Date(2026, 8, 31, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).SetRate("7.30000000").SetIsActive(true).Save(f.ctx); err != nil {
			t.Fatal(err)
		}
		data.db.Order.UpdateOneID(f.orderID).SetEtd("2026-08-26T10:00:00+08:00").ExecX(f.ctx)
		base := f.fee(billFeeSpec{key: "base", currency: "USD", rate: "7.1235", total: "100"})
		extra := f.fee(billFeeSpec{key: "extra", currency: "USD", rate: "7.1235", total: "10.01"})
		bill := f.bill("rate", f.usdAccountID, base)
		refresh := func(ctx context.Context, tx *ent.Tx, etd string) error {
			if _, err := tx.Order.Query().Where(orderent.IDEQ(f.orderID)).ForUpdate().Only(ctx); err != nil {
				return err
			}
			if err := tx.Order.UpdateOneID(f.orderID).SetEtd(etd).Exec(ctx); err != nil {
				return err
			}
			return refreshFeeRatesForOrders(ctx, tx, f.organizationID, f.actorID, []uuid.UUID{f.orderID})
		}

		// 汇率联动先获锁：增删等待后成功，新行采用账单汇率而非费用新汇率。
		hold := holdTransaction(data, func(ctx context.Context, tx *ent.Tx) error {
			return refresh(ctx, tx, "2026-09-02T10:00:00+08:00")
		})
		hold.waitLocked(t)
		done := f.updateAsync(bill.ID, 1, []uuid.UUID{extra}, nil)
		requireBlocked(t, done)
		hold.commit(t)
		if err := awaitResult(t, done); err != nil {
			t.Fatalf("汇率联动后增删: %v", err)
		}
		head := f.requireBillConsistent(bill.ID, base, extra)
		if !decimal.RequireFromString(head.ExchangeRate).Equal(decimal.RequireFromString("7.1235")) || !decimal.RequireFromString(head.BaseCurrencyAmount).Equal(decimal.RequireFromString("110.01").Mul(decimal.RequireFromString("7.1235")).RoundBank(8)) {
			t.Fatalf("账单汇率或本位币被费用汇率改写: %s/%s", head.ExchangeRate, head.BaseCurrencyAmount)
		}
		extraFee, err := data.db.OrderFee.Get(f.ctx, extra)
		if err != nil || !decimal.RequireFromString(extraFee.ExchangeRate).Equal(decimal.RequireFromString("7.3")) {
			t.Fatalf("费用自身汇率应已联动为 7.3: %v（%v）", extraFee, err)
		}

		// 增删先获锁：联动等待订单锁后执行，未受保护的已建账费用汇率照常联动，账单快照不变。
		third := f.fee(billFeeSpec{key: "third", currency: "USD", rate: "7.3", total: "1"})
		held := f.holdBillFeeUpdate(bill.ID, 2, []uuid.UUID{third}, nil)
		held.waitLocked(t)
		refreshDone := make(chan error, 1)
		go func() {
			refreshDone <- data.WithTx(context.Background(), func(tx *ent.Tx) error {
				return refresh(context.Background(), tx, "2026-08-26T10:00:00+08:00")
			})
		}()
		requireBlocked(t, refreshDone)
		held.commit(t)
		if err := awaitResult(t, refreshDone); err != nil {
			t.Fatalf("增删后汇率联动: %v", err)
		}
		head = f.requireBillConsistent(bill.ID, base, extra, third)
		if !decimal.RequireFromString(head.ExchangeRate).Equal(decimal.RequireFromString("7.1235")) {
			t.Fatalf("账单汇率不应随费用联动: %s", head.ExchangeRate)
		}
		thirdFee, err := data.db.OrderFee.Get(f.ctx, third)
		if err != nil || !decimal.RequireFromString(thirdFee.ExchangeRate).Equal(decimal.RequireFromString("7.1235")) {
			t.Fatalf("已建账未受保护费用应随日期联动: %v（%v）", thirdFee, err)
		}
	})
}

func sameOptionalDecimalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return decimal.RequireFromString(*left).Equal(decimal.RequireFromString(*right))
}
