package data

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	invoice "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	netting "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/shopspring/decimal"
)

// 使用正式迁移的隔离 Schema 验证真实 PostgreSQL 数值聚合，不以 sqlmock 代替分组精度。
func TestFinanceLedgerMatchedSummaryPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := newFinanceBillPostgresFixture(t, data)
	other := newFinanceBillPostgresFixture(t, data)
	// 新建的其他实体须在基础组织夹具清理前移除。
	t.Cleanup(func() {
		ids := []uuid.UUID{f.organizationID, other.organizationID}
		if _, err := data.db.FinanceInvoice.Delete().Where(invoice.OrganizationIDIn(ids...)).Exec(ctx); err != nil {
			t.Error(err)
		}
		if _, err := data.db.FinanceVerification.Delete().Where(verification.OrganizationIDIn(ids...)).Exec(ctx); err != nil {
			t.Error(err)
		}
		if _, err := data.db.FinanceNetting.Delete().Where(netting.OrganizationIDIn(ids...)).Exec(ctx); err != nil {
			t.Error(err)
		}
		if _, err := data.db.FinanceCashflow.Delete().Where(cash.OrganizationIDIn(ids...)).Exec(ctx); err != nil {
			t.Error(err)
		}
	})
	type sample struct {
		currency, base, amount, baseAmount string
		converted                          bool
	}
	samples := []sample{{"USD", "CNY", "0.10000001", "0.70000007", true}, {"USD", "CNY", "0.20000002", "1.40000014", true}, {"EUR", "USD", "3.00000003", "21.00000021", false}}
	for _, fixture := range []*financeBillPostgresFixture{f, other} {
		for i, row := range samples {
			key := fmt.Sprintf("MATCH-%s-%d", fixture.suffix, i)
			_, err := data.db.OrderFee.Create().SetOrderID(fixture.orderID).SetIdempotencyKey(key).SetDirection(fee.DirectionRECEIVABLE).SetFeeCode(key).SetFeeName("筛选汇总测试").SetSettlementPartyID(fixture.partnerID).SetBillingUnit("票").SetQuantity("1").SetUnitPrice(row.amount).SetTotalAmount(row.amount).SetNetAmount(row.amount).SetTaxAmount("0").SetCurrency(row.currency).SetExchangeRate("7").SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetBaseCurrency(row.base).SetBaseCurrencyAmount(row.baseAmount).Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			b := data.db.FinanceBill.Create().SetOrganizationID(fixture.organizationID).SetBillNo(key).SetIdempotencyKey(key).SetDirection(bill.DirectionRECEIVABLE).SetStatus(bill.StatusDRAFT).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("筛选汇总").SetCurrency(row.currency).SetBaseCurrency(row.base).SetExchangeRate("7").SetExchangeRateSource(bill.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetTotalAmount(row.amount).SetNetAmount(row.amount).SetTaxAmount("0").SetBaseCurrencyAmount(row.baseAmount).SetFeeCount(1).SetBillDate(financeBillIntegrationDate).SetVersion(1)
			if _, err := withTestFinanceBillSettlementAccountSnapshot(b, fixture.accountID, "CNY").Save(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := data.db.FinanceCashflow.Create().SetOrganizationID(fixture.organizationID).SetFlowNo(key).SetIdempotencyKey(key).SetDirection(cash.DirectionRECEIVABLE).SetStatus(cash.StatusDRAFT).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("筛选汇总").SetCurrency(row.currency).SetAmount(row.amount).SetExchangeRate("7").SetExchangeRateSource(cash.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).SetBaseCurrency(row.base).SetBaseAmount(row.baseAmount).SetTransactionDate(financeBillIntegrationDate).SetOurAccount("测试账户").SetPaymentMethod("BANK_TRANSFER").Save(ctx); err != nil {
				t.Fatal(err)
			}
			inv := data.db.FinanceInvoice.Create().SetOrganizationID(fixture.organizationID).SetRecordNo(key).SetIdempotencyKey(key).SetDirection(invoice.DirectionRECEIVABLE).SetInvoiceType(invoice.InvoiceTypeNORMAL).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("筛选汇总").SetCurrency(row.currency).SetBaseCurrency(row.base).SetTotalAmount(row.amount).SetNetAmount(row.amount).SetTaxAmount("0").SetBillCount(1)
			if row.converted {
				inv.SetStatus(invoice.StatusISSUED).SetBaseCurrencyAmount(row.baseAmount).SetExchangeRate("7").SetExchangeRateSource(invoice.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate)
			}
			if _, err := inv.Save(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := data.db.FinanceVerification.Create().SetOrganizationID(fixture.organizationID).SetVerificationNo(key).SetIdempotencyKey(key).SetStatus(verification.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(fixture.actorID).SetReversalReason("筛选汇总回归").SetDirection(verification.DirectionRECEIVABLE).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("筛选汇总").SetCurrency(row.currency).SetAmount(row.amount).SetBaseCurrency(row.base).SetBaseAmount(row.baseAmount).SetBillBaseAmount(row.baseAmount).SetCashflowBaseAmount(row.baseAmount).SetExchangeGainLoss("0").SetVerificationDate(financeBillIntegrationDate).Save(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := data.db.FinanceNetting.Create().SetOrganizationID(fixture.organizationID).SetNettingNo(key).SetIdempotencyKey(key).SetRequestHash(key).SetSettlementPartyID(fixture.partnerID).SetSettlementPartyName("筛选汇总").SetCurrency(row.currency).SetAmount(row.amount).SetBaseCurrency(row.base).SetBaseCurrencyAmount(row.baseAmount).SetPayableBaseAmount(row.baseAmount).SetExchangeGainLoss("0").Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	// 同组织另有已确认对冲：筛选草稿时顶部不得继续计入这笔已确认金额。
	if _, err := data.db.FinanceNetting.Create().SetOrganizationID(f.organizationID).SetNettingNo("CONFIRMED-" + f.suffix).SetIdempotencyKey("CONFIRMED-" + f.suffix).SetRequestHash("CONFIRMED-" + f.suffix).SetStatus(netting.StatusCONFIRMED).SetConfirmedAt(time.Now()).SetConfirmedBy(f.actorID).SetSettlementPartyID(f.partnerID).SetSettlementPartyName("筛选汇总").SetCurrency("USD").SetAmount("5").SetBaseCurrency("CNY").SetBaseCurrencyAmount("35").SetPayableBaseAmount("35").SetExchangeGainLoss("0").Save(ctx); err != nil {
		t.Fatal(err)
	}
	lists := []struct {
		name string
		load func(int) (biz.FinanceLedgerMatchedSummary, int64, error)
	}{
		{"费用", func(page int) (biz.FinanceLedgerMatchedSummary, int64, error) {
			r, e := (&settlementRepo{data: data}).ListFeeLedger(ctx, []uuid.UUID{f.organizationID}, biz.FeeLedgerFilter{Page: page, PageSize: 1})
			if e != nil {
				return biz.FinanceLedgerMatchedSummary{}, 0, e
			}
			return r.MatchedSummary, r.Total, nil
		}},
		{"账单", func(page int) (biz.FinanceLedgerMatchedSummary, int64, error) {
			r, e := NewFinanceBillRepo(data).List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceBillFilter{Page: page, PageSize: 1, Status: biz.FinanceBillDraft})
			if e != nil {
				return biz.FinanceLedgerMatchedSummary{}, 0, e
			}
			return r.MatchedSummary, r.Total, nil
		}},
		{"收付", func(page int) (biz.FinanceLedgerMatchedSummary, int64, error) {
			r, e := NewFinanceCashflowRepo(data).List(ctx, f.organizationID, biz.FinanceCashflowFilter{Page: page, PageSize: 1})
			if e != nil {
				return biz.FinanceLedgerMatchedSummary{}, 0, e
			}
			return r.MatchedSummary, r.Total, nil
		}},
		{"发票", func(page int) (biz.FinanceLedgerMatchedSummary, int64, error) {
			r, e := NewFinanceInvoiceRepo(data).List(ctx, f.organizationID, biz.FinanceInvoiceFilter{Page: page, PageSize: 1})
			if e != nil {
				return biz.FinanceLedgerMatchedSummary{}, 0, e
			}
			return r.MatchedSummary, r.Total, nil
		}},
		{"核销", func(page int) (biz.FinanceLedgerMatchedSummary, int64, error) {
			r, e := NewVerificationRepo(data).List(ctx, f.organizationID, biz.VerificationFilter{Page: page, PageSize: 1})
			if e != nil {
				return biz.FinanceLedgerMatchedSummary{}, 0, e
			}
			return r.MatchedSummary, r.Total, nil
		}},
		{"对冲", func(page int) (biz.FinanceLedgerMatchedSummary, int64, error) {
			r, e := NewFinanceNettingRepo(data).List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceNettingFilter{Page: page, PageSize: 1, Status: biz.FinanceNettingDraft})
			if e != nil {
				return biz.FinanceLedgerMatchedSummary{}, 0, e
			}
			if r.Summary.ConfirmedCount != 0 || len(r.Summary.AmountsByBaseCurrency) != 0 {
				t.Fatal("草稿筛选不应包含已确认顶部金额")
			}
			return r.MatchedSummary, r.Total, nil
		}},
	}
	for _, list := range lists {
		t.Run(list.name, func(t *testing.T) {
			for page := 1; page <= 2; page++ {
				summary, total, err := list.load(page)
				if err != nil {
					t.Fatal(err)
				}
				if summary.Count != 3 || total != 3 || len(summary.Amounts) != 2 {
					t.Fatalf("分页%d/组织汇总错误: %+v total=%d", page, summary, total)
				}
				for _, amount := range summary.Amounts {
					if amount.Currency == "USD" {
						if !amount.Amount.Equal(decimal.RequireFromString("0.30000003")) || amount.BaseAmount == nil || !amount.BaseAmount.Equal(decimal.RequireFromString("2.10000021")) || amount.BaseCurrency == nil || *amount.BaseCurrency != "CNY" {
							t.Fatalf("精确分桶错误: %+v", amount)
						}
					}
					if list.name == "发票" && amount.Currency == "EUR" && (amount.BaseAmount != nil || amount.UnconvertedCount != 1) {
						t.Fatalf("未折算发票误作零: %+v", amount)
					}
				}
			}
		})
	}
	// 同一字段的关键词/币种过滤直接作用于聚合查询，不只作用于当前页。
	result, err := NewFinanceNettingRepo(data).List(ctx, []uuid.UUID{f.organizationID}, biz.FinanceNettingFilter{Page: 1, PageSize: 1, Currency: "EUR", Keyword: "MATCH-" + f.suffix})
	if err != nil || result.Total != 1 || result.MatchedSummary.Count != 1 || len(result.MatchedSummary.Amounts) != 1 {
		t.Fatalf("完整筛选失败: %+v, %v", result, err)
	}
}
