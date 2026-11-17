package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/shopspring/decimal"
)

func TestFinanceBillAutomaticDatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	today := biz.FinanceBusinessDate(time.Now())
	yesterday := biz.FinanceBusinessDate(time.Now().AddDate(0, 0, -1))

	t.Run("增减更新自动日期账期汇率且保留首次创建时间", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		f.createExchangeRateSetting("7.21")
		first := f.fee(billFeeSpec{key: "date-first", currency: "USD", total: "100", rate: "7.2"})
		tiny := f.fee(billFeeSpec{key: "date-tiny", currency: "USD", total: "1", rate: "7.2"})
		data.db.OrderFee.UpdateOneID(tiny).SetQuantity("0.0001").SetUnitPrice("0.0001").SetTotalAmount("0.00000001").SetNetAmount("0.00000001").SetBaseCurrencyAmount("0.00000007").ExecX(f.ctx)
		added := f.fee(billFeeSpec{key: "date-added", currency: "USD", total: "10", rate: "7.2"})
		bill := f.bill("date", f.usdAccountID, first, tiny)
		terms := 30
		oldDue := time.Now().In(biz.ExchangeRateBusinessLocation()).AddDate(0, 0, 29).Format("2006-01-02")
		old := data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate(yesterday).SetPaymentTermsDays(terms).SetDueDate(oldDue).SetExchangeRateDate(yesterday).SetExchangeRate("7.2").SetBaseCurrencyAmount("720.00000007").SaveX(f.ctx)
		data.db.FinanceBillLine.Update().Where(financebilllineent.BillIDEQ(bill.ID)).SetExchangeRate("7.2").ExecX(f.ctx)
		data.db.FinanceBillLine.Update().Where(financebilllineent.BillIDEQ(bill.ID), financebilllineent.OrderFeeIDEQ(first)).SetBaseCurrencyAmount("720.00000000").ExecX(f.ctx)
		updated, err := f.update(bill.ID, bill.Version, []uuid.UUID{added}, nil)
		if err != nil {
			t.Fatal(err)
		}
		wantDue := time.Now().In(biz.ExchangeRateBusinessLocation()).AddDate(0, 0, terms).Format("2006-01-02")
		if updated.BillDate != today || updated.ExchangeRateDate != today || updated.DueDate == nil || *updated.DueDate != wantDue || !updated.CreatedAt.Equal(old.CreatedAt) || !updated.ExchangeRate.Equal(decimal.RequireFromString("7.21")) {
			t.Fatalf("增减日期/账期/汇率/原创建时间错误: %#v", updated)
		}
		for _, line := range updated.Lines {
			if line.Active && !line.ExchangeRate.Equal(updated.ExchangeRate) {
				t.Fatalf("包括折本币金额不变的微额行在内，全部活动行必须使用新汇率: %#v", line)
			}
		}
		f.requireBillConsistent(bill.ID, first, tiny, added)
		replayed, replayErr := f.usecase.Create(f.ctx, f.organizationID, f.actorID, biz.CreateFinanceBillInput{FeeIDs: []uuid.UUID{first, tiny}, SettlementAccountID: f.usdAccountID, IdempotencyKey: "bill-fee-update-date-" + f.suffix})
		if replayErr != nil || replayed.ID != bill.ID || replayed.Version != updated.Version {
			t.Fatalf("费用已增减后原创建请求须通过不可变审计回放原单: %#v %v", replayed, replayErr)
		}
		// 移除同样推进旧业务日期，保留首次创建时刻。
		data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate(yesterday).ExecX(f.ctx)
		removed, err := f.update(bill.ID, updated.Version, nil, []uuid.UUID{added})
		if err != nil || removed.BillDate != today || !removed.CreatedAt.Equal(old.CreatedAt) {
			t.Fatalf("移除应刷新日期但不改首次创建时间: %#v %v", removed, err)
		}
	})

	t.Run("缺汇率或显式到期日早于新日期时整体回滚", func(t *testing.T) {
		for _, missingRate := range []bool{true, false} {
			f := newBillFeeUpdateFixture(t, data)
			setting := f.createExchangeRateSetting("7.2")
			first := f.fee(billFeeSpec{key: "rollback-first", currency: "USD", total: "100", rate: "7.2"})
			added := f.fee(billFeeSpec{key: "rollback-added", currency: "USD", total: "10", rate: "7.2"})
			bill := f.bill("date-rollback", f.usdAccountID, first)
			update := data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate(yesterday).SetExchangeRateDate(yesterday)
			if !missingRate {
				update.SetDueDate(yesterday)
			}
			before := update.SaveX(f.ctx)
			if missingRate {
				data.db.ExchangeRateSetting.UpdateOneID(setting).SetIsActive(false).ExecX(f.ctx)
			}
			auditCount := data.db.AuditLog.Query().Where(auditlogent.ResourceIDEQ(bill.ID.String()), auditlogent.ActionEQ("finance.bill.fees.update")).CountX(f.ctx)
			_, err := f.update(bill.ID, bill.Version, []uuid.UUID{added}, nil)
			if err == nil {
				t.Fatal("非法新日期/汇率必须拒绝")
			}
			after := f.requireBillConsistent(bill.ID, first)
			if after.Version != before.Version || after.BillDate != yesterday || after.ExchangeRateDate != yesterday || after.TotalAmount != before.TotalAmount || !after.CreatedAt.Equal(before.CreatedAt) {
				t.Fatalf("失败必须回滚日期、金额和版本: %#v", after)
			}
			if got := data.db.AuditLog.Query().Where(auditlogent.ResourceIDEQ(bill.ID.String()), auditlogent.ActionEQ("finance.bill.fees.update")).CountX(f.ctx); got != auditCount {
				t.Fatal("失败不得写成功审计")
			}
		}
	})

	t.Run("未开票普通编辑不推进创建日期", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		first := f.fee(billFeeSpec{key: "metadata-date-first", total: "100"})
		bill := f.bill("metadata-date", f.accountID, first)
		old := data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate(yesterday).SetExchangeRateDate(yesterday).SaveX(f.ctx)
		note := "未开票普通备注变更"
		metadata, err := f.usecase.Update(f.ctx, f.scope, f.actorID, biz.UpdateFinanceBillInput{ID: bill.ID, ExpectedVersion: bill.Version, SettlementAccountID: f.accountID, Note: &note})
		if err != nil || metadata.BillDate != yesterday || !metadata.CreatedAt.Equal(old.CreatedAt) {
			t.Fatalf("普通编辑不得推进创建日期: %#v %v", metadata, err)
		}
	})

	t.Run("开票锁定日期内容但保留非金额编辑", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		first := f.fee(billFeeSpec{key: "invoice-date-first", total: "100"})
		added := f.fee(billFeeSpec{key: "invoice-date-added", total: "10"})
		bill := f.bill("invoice-date", f.accountID, first)
		data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate(yesterday).SetExchangeRateDate(yesterday).ExecX(f.ctx)
		invoice := data.db.FinanceInvoice.Create().SetOrganizationID(f.organizationID).SetRecordNo("INV-DATE-" + f.suffix).SetIdempotencyKey("inv-date-" + f.suffix).
			SetDirection(financeinvoiceent.DirectionRECEIVABLE).SetInvoiceType(financeinvoiceent.InvoiceTypeDIGITAL_NORMAL).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("日期保护").
			SetCurrency("CNY").SetBaseCurrency("CNY").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBillCount(1).SetExchangeRate("1").SetExchangeRateSource(financeinvoiceent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(today).SetBaseCurrencyAmount("100").SetStatus(financeinvoiceent.StatusISSUED).SaveX(f.ctx)
		f.invoiceIDs = append(f.invoiceIDs, invoice.ID)
		data.db.FinanceInvoiceBill.Create().SetInvoiceID(invoice.ID).SetBillID(bill.ID).SetBillNo(bill.BillNo).SetAmount("100").SetTaxAmount("0").SetActive(true).SaveX(f.ctx)
		if _, err := f.update(bill.ID, bill.Version, []uuid.UUID{added}, nil); !errors.Is(err, biz.ErrFinanceBillFeesLocked) {
			t.Fatalf("开票后不得增减: %v", err)
		}
		note := "开票后补充备注"
		updated, err := f.usecase.Update(context.Background(), f.scope, f.actorID, biz.UpdateFinanceBillInput{ID: bill.ID, ExpectedVersion: bill.Version, SettlementAccountID: f.accountID, Note: &note})
		if err != nil || updated.BillDate != yesterday || updated.ExchangeRateDate != yesterday || updated.Note == nil || *updated.Note != note {
			t.Fatalf("非金额编辑不推进冻结日: %#v %v", updated, err)
		}
	})
}
