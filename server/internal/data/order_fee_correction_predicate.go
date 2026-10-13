package data

import (
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// currentOrderFeePredicate 与biz.IsCurrentFee同口径，经济汇总只消费当前正常行。
func currentOrderFeePredicate() predicate.OrderFee {
	return feeent.And(feeent.EconomicEffectEQ(feeent.EconomicEffectINCREASE), feeent.SupersededByRequestIDIsNil())
}

// accountingOrderFeePredicate 与biz.RequiresAccounting同口径，反向待建账行不能被隐藏。
func accountingOrderFeePredicate() predicate.OrderFee {
	return feeent.And(feeent.AccountingRequiredEQ(true), feeent.SupersededByRequestIDIsNil())
}
