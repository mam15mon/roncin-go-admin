package data

import (
	"context"
	"errors"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	exchangeratesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// 周汇率测试日期：W1=2026-09-07 周、W2=2026-09-14 周、W3=2026-09-21 周（始终不维护汇率）。
const (
	rateRefreshW1 = "2026-09-10T00:00:00Z"
	rateRefreshW2 = "2026-09-15T00:00:00Z"
	rateRefreshW3 = "2026-09-22T00:00:00Z"
)

type rateRefreshEnv struct {
	t         *testing.T
	data      *Data
	orgID     uuid.UUID
	partnerID uuid.UUID
	actorID   uuid.UUID
	suffix    string
}

func (e *rateRefreshEnv) seedUSDRates() {
	e.t.Helper()
	w1 := time.Date(2026, 9, 7, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	w2 := time.Date(2026, 9, 14, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	createWeeklyRate(e.t, e.data, &e.orgID, biz.ExchangeRateSettingSourceManual, "USD", w1, "6.8000", "6.7000")
	createWeeklyRate(e.t, e.data, &e.orgID, biz.ExchangeRateSettingSourceManual, "USD", w2, "7.0000", "6.9000")
}

func (e *rateRefreshEnv) newOrder(key string, direction orderent.TradeDirection, etd, eta string) uuid.UUID {
	e.t.Helper()
	order, err := e.data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(e.orgID).
		SetOrderNo("RR-" + key + "-" + e.suffix).
		SetCustomerID(e.partnerID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(direction).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetEtd(etd).SetEta(eta).
		Save(context.Background())
	if err != nil {
		e.t.Fatalf("创建订单 %s: %v", key, err)
	}
	return order.ID
}

func (e *rateRefreshEnv) addFee(orderID uuid.UUID, key string, direction orderfeeent.Direction, currency, total, rate string, source orderfeeent.ExchangeRateSource, rateDate string) *ent.OrderFee {
	e.t.Helper()
	totalDec := decimal.RequireFromString(total)
	rateDec := decimal.RequireFromString(rate)
	item, err := e.data.db.OrderFee.Create().
		SetOrderID(orderID).
		SetIdempotencyKey("rr-" + key + "-" + e.suffix).
		SetDirection(direction).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(e.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice(totalDec.StringFixed(4)).
		SetTotalAmount(totalDec.StringFixed(8)).
		SetNetAmount(totalDec.StringFixed(8)).
		SetTaxAmount("0.00000000").
		SetCurrency(currency).
		SetExchangeRate(rateDec.StringFixed(8)).
		SetExchangeRateSource(source).
		SetExchangeRateDate(rateDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(totalDec.Mul(rateDec).StringFixed(8)).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		e.t.Fatalf("创建费用 %s: %v", key, err)
	}
	return item
}

func (e *rateRefreshEnv) fee(id uuid.UUID) *ent.OrderFee {
	e.t.Helper()
	item, err := e.data.db.OrderFee.Get(context.Background(), id)
	if err != nil {
		e.t.Fatalf("读取费用: %v", err)
	}
	return item
}

func (e *rateRefreshEnv) order(id uuid.UUID) *ent.Order {
	e.t.Helper()
	item, err := e.data.db.Order.Get(context.Background(), id)
	if err != nil {
		e.t.Fatalf("读取订单: %v", err)
	}
	return item
}

// changeDates 模拟调用方：锁订单、写日期列、取汇率日变化时联动；整体在同一事务内。
func (e *rateRefreshEnv) changeDates(orderID uuid.UUID, etd, eta string) error {
	ctx := context.Background()
	return e.data.WithTx(ctx, func(tx *ent.Tx) error {
		before, err := tx.Order.Query().Where(orderent.IDEQ(orderID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		after, err := tx.Order.UpdateOne(before).SetEtd(etd).SetEta(eta).Save(ctx)
		if err != nil {
			return err
		}
		if orderRateDayChanged(before, after) {
			return refreshFeeRatesForOrders(ctx, tx, e.orgID, e.actorID, []uuid.UUID{orderID})
		}
		return nil
	})
}

func (e *rateRefreshEnv) expectFee(name string, id uuid.UUID, rate string, source orderfeeent.ExchangeRateSource, day, base string, version uint64) {
	e.t.Helper()
	got := e.fee(id)
	if !decimal.RequireFromString(got.ExchangeRate).Equal(decimal.RequireFromString(rate)) || got.ExchangeRateSource != source ||
		got.ExchangeRateDate != day || !decimal.RequireFromString(got.BaseCurrencyAmount).Equal(decimal.RequireFromString(base)) || got.Version != version {
		e.t.Fatalf("%s: 汇率=%s 来源=%s 日期=%s 本币=%s 版本=%d，期望 %s/%s/%s/%s/%d",
			name, got.ExchangeRate, got.ExchangeRateSource, got.ExchangeRateDate, got.BaseCurrencyAmount, got.Version, rate, source, day, base, version)
	}
}

func (e *rateRefreshEnv) refreshAuditCount(orderID uuid.UUID) int {
	e.t.Helper()
	count, err := e.data.db.AuditLog.Query().Where(auditlogent.ActionEQ("order.fee.exchange_rate_refresh"), auditlogent.ResourceIDEQ(orderID.String())).Count(context.Background())
	if err != nil {
		e.t.Fatal(err)
	}
	return count
}

// cleanupRateRefresh 清理订单夹具不负责的费用与汇率行（LIFO，须在夹具创建后注册）。
func cleanupRateRefresh(t *testing.T, data *Data, orgID uuid.UUID) {
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := data.db.OrderFee.Delete().Where(orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(orgID))).Exec(ctx); err != nil {
			t.Errorf("清理联动测试费用: %v", err)
		}
		if _, err := data.db.ExchangeRateSetting.Delete().Where(exchangeratesettingent.OrganizationIDEQ(orgID)).Exec(ctx); err != nil {
			t.Errorf("清理联动测试汇率: %v", err)
		}
	})
}

// TestFeeRateRefreshMatrixPostgres 覆盖联动的可更新判定、整体失败与触发条件。
func TestFeeRateRefreshMatrixPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFeeLedgerPostgresFixture(t, data)
	e := &rateRefreshEnv{t: t, data: data, orgID: f.organizationID, partnerID: f.partnerID, actorID: f.actorID, suffix: f.suffix}
	e.seedUSDRates()
	ctx := context.Background()
	var invoiceIDs []uuid.UUID
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = data.db.FinanceInvoiceBill.Delete().Where(financeinvoicebillent.InvoiceIDIn(invoiceIDs...)).Exec(cctx)
		_, _ = data.db.FinanceInvoice.Delete().Where(financeinvoiceent.IDIn(invoiceIDs...)).Exec(cctx)
	})
	billFee := func(orderID uuid.UUID, key string, fee *ent.OrderFee) *ent.FinanceBill {
		t.Helper()
		bill := f.createConfirmedBill(key, "100", financeBillIntegrationDate)
		if _, err := data.db.FinanceBillLine.Create().
			SetBillID(bill.ID).SetOrderFeeID(fee.ID).SetOrderID(orderID).SetOrderNo("RR-" + key).
			SetFeeCode("OCEAN_FREIGHT").SetFeeName("海运费").SetQuantity("1.0000").SetUnitPrice("100.0000").
			SetTotalAmount("100.00000000").SetNetAmount("100.00000000").SetTaxAmount("0.00000000").
			SetCurrency("CNY").SetExchangeRate("1.00000000").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100.00000000").
			SetActive(true).Save(ctx); err != nil {
			t.Fatalf("创建账单行: %v", err)
		}
		return bill
	}
	expectLine := func(feeID uuid.UUID) {
		t.Helper()
		line, err := data.db.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(feeID)).Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if line.ExchangeRate != "1.00000000" || line.BaseCurrencyAmount != "100.00000000" || line.TotalAmount != "100.00000000" {
			t.Fatalf("账单行快照被改动: %+v", line)
		}
	}

	t.Run("未建账系统汇率更新且手工与非系统外费用保留", func(t *testing.T) {
		orderID := e.newOrder("sys", orderent.TradeDirectionExport, rateRefreshW1, "")
		receivable := e.addFee(orderID, "sys-r", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		payable := e.addFee(orderID, "sys-p", orderfeeent.DirectionPAYABLE, "USD", "50", "6.0", orderfeeent.ExchangeRateSourceINHERITED_LAST_WEEK, "2026-09-10")
		manual := e.addFee(orderID, "sys-m", orderfeeent.DirectionRECEIVABLE, "USD", "10", "7.5", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10")
		cny := e.addFee(orderID, "sys-c", orderfeeent.DirectionRECEIVABLE, "CNY", "20", "1", orderfeeent.ExchangeRateSourceSYSTEM, "2026-09-10")
		before := e.order(orderID)
		if err := e.changeDates(orderID, rateRefreshW2, ""); err != nil {
			t.Fatalf("联动失败: %v", err)
		}
		e.expectFee("应收", receivable.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "700", 2)
		e.expectFee("应付", payable.ID, "6.9", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "345", 2)
		e.expectFee("手工汇率保留", manual.ID, "7.5", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10", "75", 1)
		e.expectFee("本币费用日期随取值日", cny.ID, "1", orderfeeent.ExchangeRateSourceSYSTEM, "2026-09-15", "20", 2)
		if got := e.fee(receivable.ID); got.TotalAmount != "100.00000000" || got.Currency != "USD" {
			t.Fatalf("原币金额不得变化: %+v", got)
		}
		after := e.order(orderID)
		if after.FeeContentVersion != before.FeeContentVersion+1 {
			t.Fatalf("订单费用内容版本应+1: %d→%d", before.FeeContentVersion, after.FeeContentVersion)
		}
		if e.refreshAuditCount(orderID) != 1 {
			t.Fatal("应写一条联动审计")
		}
	})

	t.Run("已建账但无财务事实更新费用而账单行不动", func(t *testing.T) {
		orderID := e.newOrder("billed", orderent.TradeDirectionExport, rateRefreshW1, "")
		fee := e.addFee(orderID, "billed", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		billFee(orderID, "billed", fee)
		if err := e.changeDates(orderID, rateRefreshW2, ""); err != nil {
			t.Fatal(err)
		}
		e.expectFee("已建账无事实", fee.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "700", 2)
		expectLine(fee.ID)
	})

	t.Run("发票核销对冲保护的费用保留", func(t *testing.T) {
		orderID := e.newOrder("protected", orderent.TradeDirectionExport, rateRefreshW1, "")
		invoiced := e.addFee(orderID, "p-inv", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		verified := e.addFee(orderID, "p-ver", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		netted := e.addFee(orderID, "p-net", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		free := e.addFee(orderID, "p-free", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")

		invBill := billFee(orderID, "p-inv", invoiced)
		invoice, err := data.db.FinanceInvoice.Create().
			SetOrganizationID(f.organizationID).SetRecordNo("INV-RR-" + f.suffix).SetIdempotencyKey("inv-rr-" + f.suffix).
			SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetInvoiceType(financeinvoiceent.InvoiceTypeNORMAL).
			SetSettlementPartyID(f.partnerID).SetSettlementPartyName("联动").SetCurrency("CNY").SetBaseCurrency("CNY").
			SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBillCount(1).SetStatus(financeinvoiceent.StatusDRAFT).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		invoiceIDs = append(invoiceIDs, invoice.ID)
		if _, err := data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(invBill.ID).SetBillNo(invBill.BillNo).SetAmount("100").SetTaxAmount("0").SetActive(true).Save(ctx); err != nil {
			t.Fatal(err)
		}

		verBill := billFee(orderID, "p-ver", verified)
		cash := f.createConfirmedCashflow("rr-ver", "100", financeBillIntegrationDate)
		f.createActiveVerification("rr-ver", "100", func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, verBill, "100")}
		})

		netBill := billFee(orderID, "p-net", netted)
		f.createNetting("rr-net", financenettingent.StatusCONFIRMED, netBill, "100", true)

		billFee(orderID, "p-free", free)

		if err := e.changeDates(orderID, rateRefreshW2, ""); err != nil {
			t.Fatal(err)
		}
		for name, fee := range map[string]*ent.OrderFee{"发票": invoiced, "核销": verified, "对冲": netted} {
			e.expectFee(name+"保护", fee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)
			expectLine(fee.ID)
		}
		e.expectFee("无事实同单更新", free.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "700", 2)
	})

	t.Run("缺目标周汇率整体失败且日期与费用都不变", func(t *testing.T) {
		orderID := e.newOrder("missing", orderent.TradeDirectionExport, rateRefreshW1, "")
		okFee := e.addFee(orderID, "miss-ok", orderfeeent.DirectionRECEIVABLE, "CNY", "10", "1", orderfeeent.ExchangeRateSourceSYSTEM, "2026-09-10")
		usd := e.addFee(orderID, "miss-usd", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		before := e.order(orderID)
		err := e.changeDates(orderID, rateRefreshW3, "")
		if !errors.Is(err, biz.ErrExchangeRateMissing) {
			t.Fatalf("应返回缺汇率错误，实际 %v", err)
		}
		if meta := kratoserrors.FromError(err).Metadata; meta["currency"] != "USD" || meta["week_start"] != "2026-09-21" || meta["direction"] != "RECEIVABLE" {
			t.Fatalf("缺汇率错误应带币种、目标周、方向元数据: %v", meta)
		}
		after := e.order(orderID)
		if after.Etd != before.Etd || after.FeeContentVersion != before.FeeContentVersion {
			t.Fatalf("日期与订单费用版本应回滚: %+v", after)
		}
		e.expectFee("同单本币费用未部分写入", okFee.ID, "1", orderfeeent.ExchangeRateSourceSYSTEM, "2026-09-10", "10", 1)
		e.expectFee("美元费用不变", usd.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)
	})

	t.Run("无可更新费用时不要求汇率", func(t *testing.T) {
		orderID := e.newOrder("noupd", orderent.TradeDirectionExport, rateRefreshW1, "")
		manual := e.addFee(orderID, "noupd-m", orderfeeent.DirectionRECEIVABLE, "USD", "10", "7.5", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10")
		fee := e.addFee(orderID, "noupd-b", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		bill := billFee(orderID, "noupd-b", fee)
		cash := f.createConfirmedCashflow("rr-noupd", "100", financeBillIntegrationDate)
		f.createActiveVerification("rr-noupd", "100", func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
			return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, bill, "100")}
		})
		if err := e.changeDates(orderID, rateRefreshW3, ""); err != nil {
			t.Fatalf("无可更新费用应照常保存: %v", err)
		}
		if got := e.order(orderID); got.Etd != rateRefreshW3 {
			t.Fatalf("日期应保存: %s", got.Etd)
		}
		e.expectFee("手工", manual.ID, "7.5", orderfeeent.ExchangeRateSourceMANUAL, "2026-09-10", "75", 1)
		e.expectFee("受保护", fee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)

		emptyID := e.newOrder("nofee", orderent.TradeDirectionExport, rateRefreshW1, "")
		if err := e.changeDates(emptyID, rateRefreshW3, ""); err != nil {
			t.Fatalf("无费用订单应照常保存: %v", err)
		}
	})

	t.Run("非基准日期不触发且进口取ETA", func(t *testing.T) {
		exportID := e.newOrder("exp-eta", orderent.TradeDirectionExport, rateRefreshW1, rateRefreshW1)
		exportFee := e.addFee(exportID, "exp-eta", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		if err := e.changeDates(exportID, rateRefreshW1, rateRefreshW3); err != nil {
			t.Fatalf("出口改 ETA 不应触发联动: %v", err)
		}
		e.expectFee("出口改ETA", exportFee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)
		if err := e.changeDates(exportID, "2026-09-10T12:00:00Z", rateRefreshW3); err != nil {
			t.Fatalf("同一业务日内改时分不应触发: %v", err)
		}
		e.expectFee("同日改时分", exportFee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)

		importID := e.newOrder("imp", orderent.TradeDirectionImport, rateRefreshW3, rateRefreshW1)
		importFee := e.addFee(importID, "imp", orderfeeent.DirectionRECEIVABLE, "USD", "100", "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10")
		if err := e.changeDates(importID, rateRefreshW1, rateRefreshW1); err != nil {
			t.Fatalf("进口改 ETD 不应触发联动: %v", err)
		}
		e.expectFee("进口改ETD", importFee.ID, "6.8", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-10", "680", 1)
		if err := e.changeDates(importID, rateRefreshW1, rateRefreshW2); err != nil {
			t.Fatal(err)
		}
		e.expectFee("进口改ETA", importFee.ID, "7.0", orderfeeent.ExchangeRateSourceWEEKLY, "2026-09-15", "700", 2)
	})
}
