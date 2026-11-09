package data

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

// 首轮费用及账单关系已加载后，由另一连接完成真实整账取消。
// 请求应保持同一读取快照；下一请求才采纳取消结果，不把正常并发操作误报为余额损坏。
func TestFeeLedgerCancellationReadSnapshotPostgres(t *testing.T) {
	for _, view := range []string{"列表", "订单详情"} {
		t.Run(view, func(t *testing.T) {
			d, cleanup := getIntegrationData(t)
			t.Cleanup(cleanup)
			f := newFeeLedgerPostgresFixture(t, d)
			feeID := f.createLedgerFee("snapshot", "300")
			bill := f.createConfirmedBill("snapshot", "300", financeBillIntegrationDate)
			f.linkBillFee("snapshot", bill, feeID, "300")
			orgs := []uuid.UUID{f.organizationID}
			repo := NewSettlementRepo(d)
			var cancelled atomic.Bool
			d.db.OrderFee.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
				return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
					value, err := next.Query(ctx, query)
					if err != nil {
						return value, err
					}
					fees, ok := value.([]*ent.OrderFee)
					if !ok {
						return value, nil
					}
					for _, fee := range fees {
						if fee.ID != feeID || len(fee.Edges.FinanceBillLines) == 0 || !cancelled.CompareAndSwap(false, true) {
							continue
						}
						_, err = NewFinanceBillRepo(d).Cancel(context.Background(), orgs, bill.ID, f.actorID, bill.Version, "验证读取期间取消账单", &biz.AuditEvent{Action: "finance.bill.cancel", Result: "success"})
						if err != nil {
							return nil, err
						}
					}
					return value, nil
				})
			}))
			read := func() (*biz.FeeLedgerItem, error) {
				if view == "列表" {
					result, err := repo.ListFeeLedger(context.Background(), orgs, biz.FeeLedgerFilter{Page: 1, PageSize: 10})
					if err != nil {
						return nil, err
					}
					return findFeeLedgerItem(result.Items, feeID), nil
				}
				result, err := repo.GetFeeLedgerOrderDetail(context.Background(), orgs, f.orderID)
				if err != nil {
					return nil, err
				}
				return findFeeLedgerItem(result.Items, feeID), nil
			}
			item, err := read()
			if err != nil {
				t.Fatalf("读取期间正常取消账单不应使%s失败: %v", view, err)
			}
			if !cancelled.Load() {
				t.Fatal("未触发首轮关联读取后的取消窗口")
			}
			if item == nil || item.BillNo != bill.BillNo || item.FinancialProgress != biz.FeeLedgerUnverifiedUninvoiced || item.UnverifiedAmount.String() != "300" || !item.VerifiedAmount.IsZero() || !item.NettedAmount.IsZero() {
				t.Fatalf("当前请求应保留取消前一致快照: %+v", item)
			}
			item, err = read()
			if err != nil {
				t.Fatal(err)
			}
			if item == nil || item.BillNo != "" || item.FinancialProgress != biz.FeeLedgerUnbilled || item.UnverifiedAmount.String() != "300" {
				t.Fatalf("下一请求应采纳已取消后的未建账状态: %+v", item)
			}
		})
	}
}
