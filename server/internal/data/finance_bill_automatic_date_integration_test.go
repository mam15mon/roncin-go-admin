package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/shopspring/decimal"
)

func TestFinanceBillAutomaticDatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	today := biz.FinanceBusinessDate(time.Now())
	yesterday := biz.FinanceBusinessDate(time.Now().AddDate(0, 0, -1))

	t.Run("跨日增减保留创建日期账期汇率与首次创建时间", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		setting := f.createExchangeRateSetting("7.2")
		first := f.fee(billFeeSpec{key: "date-first", currency: "USD", total: "100", rate: "7.2"})
		tiny := f.fee(billFeeSpec{key: "date-tiny", currency: "USD", total: "1", rate: "7.2"})
		data.db.OrderFee.UpdateOneID(tiny).SetQuantity("0.0001").SetUnitPrice("0.0001").SetTotalAmount("0.00000001").SetNetAmount("0.00000001").SetBaseCurrencyAmount("0.00000007").ExecX(f.ctx)
		added := f.fee(billFeeSpec{key: "date-added", currency: "USD", total: "10", rate: "7.2"})
		bill := f.bill("date", f.usdAccountID, first, tiny)
		old := data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate("2026-11-18").SetPaymentTermsDays(30).SetDueDate("2026-12-18").SetExchangeRateDate("2026-11-18").SaveX(f.ctx)
		// 缺失任何可用汇率后仍可增、减和同时增减；业务日期不再参与维护。
		data.db.ExchangeRateSetting.UpdateOneID(setting).SetIsActive(false).ExecX(f.ctx)
		version := bill.Version
		for _, step := range []struct{ add, remove []uuid.UUID }{
			{[]uuid.UUID{added}, nil},
			{nil, []uuid.UUID{added}},
			{[]uuid.UUID{added}, []uuid.UUID{tiny}},
		} {
			updated, err := f.update(bill.ID, version, step.add, step.remove)
			if err != nil {
				t.Fatal(err)
			}
			if updated.BillDate != old.BillDate || updated.DueDate == nil || *updated.DueDate != "2026-12-18" || updated.PaymentTermsDays == nil || *updated.PaymentTermsDays != 30 || updated.ExchangeRateDate != old.ExchangeRateDate || updated.ExchangeRate.StringFixed(8) != old.ExchangeRate || updated.ExchangeRateSource != string(old.ExchangeRateSource) || updated.ExchangeRateSettingID == nil || *updated.ExchangeRateSettingID != setting || !updated.CreatedAt.Equal(old.CreatedAt) {
				t.Fatalf("增减不得改变日期、账期与汇率快照: %#v", updated)
			}
			sum := decimal.Zero
			for _, line := range updated.Lines {
				if line.Active {
					if !line.ExchangeRate.Equal(updated.ExchangeRate) {
						t.Fatalf("活动行必须采用原账单汇率: %#v", line)
					}
					sum = sum.Add(line.BaseCurrencyAmount)
				}
			}
			if !sum.Equal(updated.BaseCurrencyAmount) {
				t.Fatal("头行本币合计不一致")
			}
			version = updated.Version
		}
		f.requireBillConsistent(bill.ID, first, added)
		replayed, err := f.usecase.Create(f.ctx, f.organizationID, f.actorID, biz.CreateFinanceBillInput{FeeIDs: []uuid.UUID{first, tiny}, SettlementAccountID: f.usdAccountID, IdempotencyKey: "bill-fee-update-date-" + f.suffix})
		if err != nil || replayed.ID != bill.ID || replayed.Version != version {
			t.Fatalf("费用增减后原创建意图仍须回放同一账单: %#v %v", replayed, err)
		}
	})

	t.Run("已过到期日的账单增减费用不顺延", func(t *testing.T) {
		f := newBillFeeUpdateFixture(t, data)
		first := f.fee(billFeeSpec{key: "past-first", total: "100"})
		added := f.fee(billFeeSpec{key: "past-added", total: "10"})
		bill := f.bill("past", f.accountID, first)
		data.db.FinanceBill.UpdateOneID(bill.ID).SetBillDate(yesterday).SetDueDate(yesterday).ExecX(f.ctx)
		updated, err := f.update(bill.ID, bill.Version, []uuid.UUID{added}, nil)
		if err != nil || updated.BillDate != yesterday || updated.DueDate == nil || *updated.DueDate != yesterday {
			t.Fatalf("过期日不影响增减: %#v %v", updated, err)
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

func TestFinanceBillUpdateFixedExchangeSnapshotPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newBillFeeUpdateFixture(t, data)
	setting := f.createExchangeRateSetting("7.1235")
	first := f.fee(billFeeSpec{key: "edit-first", currency: "USD", total: "100", rate: "7.1235"})
	second := f.fee(billFeeSpec{key: "edit-second", currency: "USD", total: "20", rate: "7.1235"})
	third := f.fee(billFeeSpec{key: "edit-third", currency: "USD", total: "20", rate: "7.1235"})
	// 微额产生尾差，编辑必须保持每条行的本币快照及行身份。
	for _, id := range []uuid.UUID{second, third} {
		data.db.OrderFee.UpdateOneID(id).SetQuantity("0.0002").SetUnitPrice("0.0002").SetTotalAmount("0.00000004").SetNetAmount("0.00000004").SetBaseCurrencyAmount("0.00000028").ExecX(f.ctx)
	}
	original := f.bill("edit-fixed", f.usdAccountID, first, second, third)
	roundedSum := decimal.Zero
	for _, line := range original.Lines {
		roundedSum = roundedSum.Add(line.TotalAmount.Mul(original.ExchangeRate).RoundBank(8))
	}
	if !original.BaseCurrencyAmount.Sub(roundedSum).Equal(decimal.RequireFromString("0.00000001")) {
		t.Fatal("夹具必须包含非零尾差，才能验证编辑保留每行本币金额")
	}
	current := original
	due := time.Now().In(biz.ExchangeRateBusinessLocation()).AddDate(0, 0, 30).Format("2006-01-02")
	note := "汇率停用后只改备注"
	currency, invoiceRate := "CNY", decimal.RequireFromString("8")
	for _, action := range []string{"修改汇率后改到期日", "停用汇率后改备注", "指定到期日原样保存", "修改预计开票"} {
		t.Run(action, func(t *testing.T) {
			input := biz.UpdateFinanceBillInput{ID: original.ID, ExpectedVersion: current.Version, SettlementAccountID: f.usdAccountID, DueDate: &due}
			switch action {
			case "修改汇率后改到期日":
				data.db.ExchangeRateSetting.UpdateOneID(setting).SetRate("9").ExecX(f.ctx)
			case "停用汇率后改备注":
				data.db.ExchangeRateSetting.UpdateOneID(setting).SetIsActive(false).ExecX(f.ctx)
				input.Note = &note
			case "修改预计开票":
				input.EstimatedInvoiceCurrency, input.EstimatedInvoiceRate = &currency, &invoiceRate
			}
			updated, err := f.usecase.Update(f.ctx, f.scope, f.actorID, input)
			if err != nil {
				t.Fatal(err)
			}
			if updated.BillDate != original.BillDate || updated.PaymentTermsDays != nil || updated.ExchangeRateDate != original.ExchangeRateDate || updated.ExchangeRateSource != original.ExchangeRateSource || !updated.ExchangeRate.Equal(original.ExchangeRate) || updated.ExchangeRateSettingID == nil || *updated.ExchangeRateSettingID != setting || !updated.BaseCurrencyAmount.Equal(original.BaseCurrencyAmount) || updated.DueDate == nil || *updated.DueDate != due {
				t.Fatalf("编辑改变汇率/金额或丢失到期日: %#v", updated)
			}
			if len(updated.Lines) != len(original.Lines) {
				t.Fatal("编辑改变行数")
			}
			for i, line := range updated.Lines {
				if line.ID != original.Lines[i].ID || !line.BaseCurrencyAmount.Equal(original.Lines[i].BaseCurrencyAmount) || !line.ExchangeRate.Equal(original.Lines[i].ExchangeRate) {
					t.Fatalf("编辑改变行快照: %#v", line)
				}
			}
			if action == "修改预计开票" && (updated.EstimatedInvoiceAmount == nil || !updated.EstimatedInvoiceAmount.Equal(original.TotalAmount.Mul(invoiceRate).RoundBank(8))) {
				t.Fatal("预计开票金额没有更新")
			}
			current = updated
		})
	}
}
