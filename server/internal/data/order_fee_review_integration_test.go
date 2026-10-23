package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"testing"
)

func reviewFixture(t *testing.T) (*feeSupplementPostgresFixture, *biz.OrderFeeReviewUsecase, *ent.Order, *ent.OrderFee) {
	f := newFeeSupplementFixture(t)
	permissions := []*ent.Permission{}
	for _, key := range []string{"system.finance.fee.review", "business.order.si.fee.read", "business.order.si.fee.create", "business.order.si.fee.update"} {
		p, err := f.data.db.Permission.Query().Where(permissionent.KeyEQ(key)).Only(f.ctx)
		if ent.IsNotFound(err) {
			p, err = f.data.db.Permission.Create().SetKey(key).SetName(key).SetGroup("测试").Save(f.ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		permissions = append(permissions, p)
	}
	if _, err := f.data.db.Role.UpdateOneID(f.lockRoleID).AddPermissions(permissions...).Save(f.ctx); err != nil {
		t.Fatal(err)
	}
	o := f.createOrder("REVIEW-" + f.suffix)
	fee, err := f.data.db.OrderFee.Create().SetOrderID(o.ID).SetIdempotencyKey("review-fee").SetDirection(feeent.DirectionRECEIVABLE).SetFeeCode("REVIEW").SetFeeName("审核测试收入").SetSettlementPartyID(f.customerID).SetBillingUnit("票").SetQuantity("1").SetUnitPrice("100").SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-10-12").SetBaseCurrency("CNY").SetBaseCurrencyAmount("100").SetExpenseDate("2026-10-12").Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	return f, biz.NewOrderFeeReviewUsecase(NewOrderFeeReviewRepo(f.data)), o, fee
}
