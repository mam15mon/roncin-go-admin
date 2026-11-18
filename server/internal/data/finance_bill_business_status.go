package data

import (
	entsql "entgo.io/ent/dialect/sql"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// billVerifiedAmountPredicate 只读取实际核销；对冲抵销不冒充实际收付款。
func billVerifiedAmountPredicate(nonzero bool) predicate.FinanceBill {
	return billAmountPredicate(func(builder *entsql.Builder, _, billID string) {
		builder.WriteString("COALESCE((SELECT SUM(fva.amount) FROM finance_verification_allocations AS fva WHERE fva.bill_id = ")
		builder.Ident(billID)
		builder.WriteString(" AND fva.active = TRUE), 0)")
		if nonzero {
			builder.WriteString(" <> 0")
		} else {
			builder.WriteString(" = 0")
		}
	})
}

// billBusinessStatusPredicate 与 biz.ResolveFinanceBillBusinessStatus 的优先级同源。
// 多选在调用处以 OR 组合，再加入列表统一 predicates，确保分页和完整聚合同范围。
func billBusinessStatusPredicate(status biz.FinanceBillBusinessStatus) predicate.FinanceBill {
	confirmed := financebillent.StatusEQ(financebillent.StatusCONFIRMED)
	invoiceReady := financebillent.Or(
		billInvoiceProgressPredicate(biz.FinanceBillInvoiceFull),
		billInvoiceProgressPredicate(biz.FinanceBillInvoiceNotRequired),
	)
	switch status {
	case biz.FinanceBillBusinessUninvoiced:
		return financebillent.And(confirmed, billInvoiceProgressPredicate(biz.FinanceBillInvoiceNone))
	case biz.FinanceBillBusinessPartiallyInvoiced:
		return financebillent.And(confirmed, billInvoiceProgressPredicate(biz.FinanceBillInvoicePartial))
	case biz.FinanceBillBusinessUnverified:
		return financebillent.And(confirmed, invoiceReady, financebillent.Or(
			billSettlementProgressPredicate(biz.FinanceBillSettlementUnsettled),
			financebillent.And(billSettlementProgressPredicate(biz.FinanceBillSettlementPartial), billVerifiedAmountPredicate(false)),
		))
	case biz.FinanceBillBusinessPartiallyVerified:
		return financebillent.And(confirmed, invoiceReady,
			billSettlementProgressPredicate(biz.FinanceBillSettlementPartial), billVerifiedAmountPredicate(true))
	case biz.FinanceBillBusinessCompleted:
		return billCompletedPredicate(true)
	default:
		return billAmountPredicate(func(builder *entsql.Builder, _, _ string) { builder.WriteString("FALSE") })
	}
}

// billVerificationReadyPredicate 是核销候选的开票前置，与 biz.CanVerifyFinanceBillInvoiceProgress 同源。
func billVerificationReadyPredicate() predicate.FinanceBill {
	return financebillent.Or(
		billInvoiceProgressPredicate(biz.FinanceBillInvoicePartial),
		billInvoiceProgressPredicate(biz.FinanceBillInvoiceFull),
		billInvoiceProgressPredicate(biz.FinanceBillInvoiceNotRequired),
	)
}
