package data

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/shopspring/decimal"
)

func TestFinanceInvoiceBillListAllocationProjectionPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newRegistrationFixture(t, d)
	firstBill, secondBill := f.createConfirmedBill("projection-first"), f.createConfirmedBill("projection-second")
	across := f.record(firstBill, "PROJECTION-ACROSS", "30", "3")
	across.Allocations = append(across.Allocations, f.record(secondBill, "unused", "20", "2").Allocations[0])
	across.TotalAmount = decimal.NewFromInt(50)
	created, err := f.register("projection", across, f.record(firstBill, "PROJECTION-PARTIAL", "10", "1"))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewFinanceInvoiceRepo(d)
	seen := map[uuid.UUID]bool{}
	for page := 1; page <= 2; page++ {
		result, err := repo.ListScoped(ctx, []uuid.UUID{f.organizationID}, biz.FinanceInvoiceFilter{BillID: &firstBill.ID, Page: page, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 2 || result.MatchedSummary.Count != 2 || len(result.Items) != 1 || len(result.Summary.AmountsByBaseCurrency) != 1 || !result.Summary.AmountsByBaseCurrency[0].ReceivableBaseAmount.Equal(decimal.NewFromInt(60)) {
			t.Fatalf("多账单关联不能放大分页或完整汇总: %+v", result)
		}
		invoice := result.Items[0]
		if seen[invoice.ID] {
			t.Fatal("跨页不得重复发票")
		}
		seen[invoice.ID] = true
		if len(invoice.Links) != invoice.BillCount {
			t.Fatalf("账单内登记列表须返回真实关联: %+v", invoice)
		}
		for _, link := range invoice.Links {
			if !link.Active || len(link.Fees) != 1 || !link.Fees[0].Amount.Equal(link.Amount) || !link.Fees[0].TaxAmount.Equal(link.TaxAmount) {
				t.Fatalf("列表不能把有效票显示成已释放或丢失逐费追溯: %+v", link)
			}
		}
	}
	if !seen[created[0].ID] || !seen[created[1].ID] {
		t.Fatal("必须完整返回两张独立票据")
	}
	plain, err := repo.ListScoped(ctx, []uuid.UUID{f.organizationID}, biz.FinanceInvoiceFilter{Page: 1, PageSize: 20})
	if err != nil || len(plain.Items) != 2 || len(plain.Items[0].Links) == 0 {
		t.Fatalf("普通台账导出也须有账单关联: %+v %v", plain, err)
	}
	for _, invoice := range plain.Items {
		for _, link := range invoice.Links {
			if len(link.Fees) != 0 {
				t.Fatal("普通台账无需加载整份逐费明细")
			}
		}
	}
}

func TestFinanceInvoiceCreationBillKeywordPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f, other := newRegistrationFixture(t, d), newRegistrationFixture(t, d)
	first := f.createConfirmedBill("keyword-first")
	f.createConfirmedBill("keyword-second")
	other.createConfirmedBill("keyword-other-company")
	d.db.Partner.UpdateOneID(f.partnerID).SetCode("INVOICE-QDYH").SetLegalName("青岛远航").SetNormalizedName("青岛远航").SaveX(ctx)
	d.db.PartnerProfile.Create().SetPartnerID(f.partnerID).SetNameEn("InvoiceOcean").SaveX(ctx)
	d.db.PartnerAlias.Create().SetPartnerID(f.partnerID).SetAliasName("登记别名").SetNormalizedAliasName("登记别名").SaveX(ctx)
	d.db.PartnerAlias.Create().SetPartnerID(f.partnerID).SetAliasName("登记别名二").SetNormalizedAliasName("登记别名二").SaveX(ctx)
	d.db.Partner.UpdateOneID(other.partnerID).SetLegalName("青岛远航").SetNormalizedName("青岛远航").SaveX(ctx)
	repo := NewFinanceInvoiceRepo(d)
	for _, keyword := range []string{"INVOICE-QDYH", "青岛远航", "qingdaoyuanhang", "qdyh", "InvoiceOcean", "登记别名", "dengjibieming", "djbm", "SE" + f.suffix} {
		seen := map[uuid.UUID]bool{}
		for page := 1; page <= 2; page++ {
			result, err := repo.ListCreationBills(ctx, f.organizationID, biz.FinanceInvoiceCreationBillFilter{Keyword: keyword, Page: page, PageSize: 1})
			if err != nil || result.Total != 2 || len(result.Items) != 1 || result.Items[0].OrganizationID != f.organizationID || seen[result.Items[0].ID] {
				t.Fatalf("候选%s须在分页前完整检索且不串公司/别名倍增: %+v %v", keyword, result, err)
			}
			seen[result.Items[0].ID] = true
		}
	}
	result, err := repo.ListCreationBills(ctx, f.organizationID, biz.FinanceInvoiceCreationBillFilter{Keyword: first.BillNo, Page: 1, PageSize: 20})
	if err != nil || result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != first.ID {
		t.Fatalf("仍须按账单号精确定位候选: %+v %v", result, err)
	}
}

// 首次读到发票头后、加载账单关联之前，在另一连接提交真实修订。
// 当前详情/列表仍须保留旧头和旧分配；下一请求再完整采纳新版本。
func TestFinanceInvoiceRevisionReadSnapshotPostgres(t *testing.T) {
	for _, view := range []string{"详情", "账单列表"} {
		t.Run(view, func(t *testing.T) {
			d, cleanup := getIntegrationData(t)
			t.Cleanup(cleanup)
			ctx := context.Background()
			f := newRegistrationFixture(t, d)
			bill := f.createConfirmedBill("snapshot")
			created, err := f.register("snapshot-original", f.record(bill, "SNAPSHOT-OLD", "40", "4"))
			if err != nil {
				t.Fatal(err)
			}
			original := created[0]
			bill = d.db.FinanceBill.GetX(ctx, bill.ID)
			replacement := f.record(bill, "SNAPSHOT-NEW", "70", "7")
			input := biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "snapshot-revision", Reason: "读取期间修订", BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: bill.ID, ExpectedVersion: bill.Version}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: original.ID, ExpectedVersion: original.Version, Registration: &replacement}}}
			var revised atomic.Bool
			d.db.FinanceInvoiceBill.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(queryCtx context.Context, query ent.Query) (ent.Value, error) {
					if revised.CompareAndSwap(false, true) {
						if _, err := f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, input); err != nil {
							return nil, err
						}
					}
					return next.Query(queryCtx, query)
				})
			}))
			repo := NewFinanceInvoiceRepo(d)
			read := func() (*biz.FinanceInvoice, error) {
				if view == "详情" {
					return repo.GetScoped(ctx, []uuid.UUID{f.organizationID}, original.ID)
				}
				result, err := repo.ListScoped(ctx, []uuid.UUID{f.organizationID}, biz.FinanceInvoiceFilter{BillID: &bill.ID, Page: 1, PageSize: 20})
				if err != nil {
					return nil, err
				}
				if result.Total != 1 || len(result.Items) != 1 || !result.Summary.AmountsByBaseCurrency[0].ReceivableBaseAmount.Equal(result.Items[0].TotalAmount) {
					t.Fatalf("列表行与完整合计须采用同一修订快照: %+v", result)
				}
				return result.Items[0], nil
			}
			before, err := read()
			if err != nil {
				t.Fatal(err)
			}
			if !revised.Load() || before.Version != 1 || !before.TotalAmount.Equal(decimal.NewFromInt(40)) || len(before.Links) != 1 || !before.Links[0].Active || !before.Links[0].Amount.Equal(before.TotalAmount) {
				t.Fatalf("当前响应不能混用旧发票头和新分配: %+v", before)
			}
			if view == "详情" && (len(before.Lines) != 1 || !before.Lines[0].TotalAmount.Equal(before.TotalAmount) || len(before.Revisions) != 0) {
				t.Fatalf("当前行和历史也须沿同一旧版本: %+v", before)
			}
			after, err := read()
			if err != nil {
				t.Fatal(err)
			}
			activeAmount := decimal.Zero
			for _, link := range after.Links {
				if link.Active {
					activeAmount = activeAmount.Add(link.Amount)
				}
			}
			if after.Version != 2 || !after.TotalAmount.Equal(decimal.NewFromInt(70)) || !activeAmount.Equal(after.TotalAmount) || len(after.Links) != 2 {
				t.Fatalf("下一次请求应完整采纳修订及保留历史: %+v", after)
			}
			if view == "详情" {
				var beforeSnapshot, afterSnapshot map[string]any
				if len(after.Revisions) != 1 {
					t.Fatal("修订后须有一份完整前后业务快照")
				}
				if err := json.Unmarshal(after.Revisions[0].Before, &beforeSnapshot); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(after.Revisions[0].After, &afterSnapshot); err != nil {
					t.Fatal(err)
				}
				if beforeSnapshot["baseCurrency"] != "CNY" || beforeSnapshot["exchangeRate"] != "1.00000000" || original.ExchangeRateSource == nil || beforeSnapshot["exchangeRateSource"] != *original.ExchangeRateSource || beforeSnapshot["exchangeRateDate"] != financeInvoiceIntegrationDate || beforeSnapshot["baseCurrencyAmount"] != "40.00000000" || afterSnapshot["baseCurrencyAmount"] != "70.00000000" {
					t.Fatal("修订必须保留被覆盖前后的汇率、来源、日期及折本币金额事实")
				}
				latestBill := d.db.FinanceBill.GetX(ctx, bill.ID)
				removed, err := f.uc.ReviseRegistrations(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: "snapshot-remove", Reason: "移除错误票据", BillVersions: []biz.FinanceInvoiceBillVersion{{BillID: bill.ID, ExpectedVersion: latestBill.Version}}, Entries: []biz.FinanceInvoiceRevisionEntry{{ID: after.ID, ExpectedVersion: after.Version, Remove: true}}})
				if err != nil {
					t.Fatal(err)
				}
				var removalSnapshot map[string]any
				for _, revision := range removed[0].Revisions {
					if revision.RevisionNumber == 3 {
						if err := json.Unmarshal(revision.After, &removalSnapshot); err != nil {
							t.Fatal(err)
						}
					}
				}
				bills, ok := removalSnapshot["bills"].([]any)
				if !ok || len(bills) != 0 || removalSnapshot["status"] != "CANCELLED" || removalSnapshot["baseCurrencyAmount"] != "70.00000000" {
					t.Fatal("移除后历史仍保留经济事实，空分配须编码为数组，不能成为损坏快照")
				}
			}
		})
	}
}
