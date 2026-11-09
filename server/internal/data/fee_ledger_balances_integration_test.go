package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	netting "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
)

func TestFeeLedgerBalancesPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFeeLedgerPostgresFixture(t, d)
	ctx := context.Background()
	repo := NewSettlementRepo(d)
	orgs := []uuid.UUID{f.organizationID}
	firstID := f.createLedgerFee("balance-first", "300")

	check := func(scope string, item *biz.FeeLedgerItem, verified, remaining, netted string) {
		t.Helper()
		if item == nil || item.VerifiedAmount.String() != verified || item.UnverifiedAmount.String() != remaining || item.NettedAmount.String() != netted {
			t.Fatalf("%s 余额错误: %+v，期望核销=%s 余额=%s 对冲=%s", scope, item, verified, remaining, netted)
		}
	}
	list, err := repo.ListFeeLedger(ctx, orgs, biz.FeeLedgerFilter{Page: 1, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	check("未建账", list.Items[0], "0", "300", "0")

	secondID := f.createLedgerFee("balance-second", "900")
	bill := f.createConfirmedBill("balance", "1200", financeBillIntegrationDate)
	f.linkBillFee("balance-first", bill, firstID, "300")
	f.linkBillFee("balance-second", bill, secondID, "900")
	n, nAllocation := f.createNetting("balance", netting.StatusCONFIRMED, bill, "120", true)
	cash := f.createConfirmedCashflow("balance", "300", financeBillIntegrationDate)
	v := f.createActiveVerification("balance", "300", func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
		return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, bill, "300")}
	})

	for page := 1; page <= 2; page++ {
		list, err := repo.ListFeeLedger(ctx, orgs, biz.FeeLedgerFilter{Page: page, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		if list.Total != 2 || len(list.Items) != 1 {
			t.Fatalf("分页结果错误: %+v", list)
		}
		if list.Items[0].Fee.ID == firstID {
			check("第一页费用", list.Items[0], "75", "195", "30")
		} else {
			check("第二页费用", list.Items[0], "225", "585", "90")
		}
	}
	detail, err := repo.GetFeeLedgerOrderDetail(ctx, orgs, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	check("详情第一笔", findFeeLedgerItem(detail.Items, firstID), "75", "195", "30")
	check("详情第二笔", findFeeLedgerItem(detail.Items, secondID), "225", "585", "90")

	d.db.FinanceVerification.UpdateOneID(v.ID).SetStatus(financeverification.StatusREVERSED).SetReversedAt(time.Now()).SetReversedBy(f.actorID).SetReversalReason("验证反核销余额").SaveX(ctx)
	d.db.FinanceVerificationAllocation.Update().Where(financeverificationallocation.VerificationIDEQ(v.ID)).SetActive(false).SaveX(ctx)
	detail, err = repo.GetFeeLedgerOrderDetail(ctx, orgs, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	check("反核销保留对冲", findFeeLedgerItem(detail.Items, firstID), "0", "270", "30")
	d.db.FinanceNetting.UpdateOneID(n.ID).SetStatus(netting.StatusREVERSED).SetReversedAt(time.Now()).SaveX(ctx)
	d.db.FinanceNettingAllocation.UpdateOneID(nAllocation.ID).SetActive(false).SaveX(ctx)
	detail, err = repo.GetFeeLedgerOrderDetail(ctx, orgs, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	check("反对冲恢复余额", findFeeLedgerItem(detail.Items, firstID), "0", "300", "0")

	cash = f.createConfirmedCashflow("balance-full", "1200", financeBillIntegrationDate)
	f.createActiveVerification("balance-full", "1200", func(id uuid.UUID) []*ent.FinanceVerificationAllocationCreate {
		return []*ent.FinanceVerificationAllocationCreate{f.verificationAllocationCreate(id, cash, bill, "1200")}
	})
	detail, err = repo.GetFeeLedgerOrderDetail(ctx, orgs, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	check("全核销第一笔", findFeeLedgerItem(detail.Items, firstID), "300", "0", "0")
	check("全核销第二笔", findFeeLedgerItem(detail.Items, secondID), "900", "0", "0")
	other, err := repo.ListFeeLedger(ctx, []uuid.UUID{uuid.New()}, biz.FeeLedgerFilter{Page: 1, PageSize: 10})
	if err != nil || other.Total != 0 {
		t.Fatalf("不可读取其他公司费用: %+v %v", other, err)
	}
}
