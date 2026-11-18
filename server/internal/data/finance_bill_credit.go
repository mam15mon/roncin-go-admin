package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	partnersettlementruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// 账单进度的 SQL 侧定义。展示侧等价定义见 enrichFinanceBillProjection 与
// biz.ResolveFinanceBillProgress，三处口径须同步维护：
//   - 结算净额 = 有效核销分摊 + 有效对冲分摊（带符号，分摊 active=TRUE；草稿对冲分摊未激活不计入）；
//   - 开票额 = 有效发票关联（active=TRUE 且发票为 DRAFT/ISSUED）的登记额；需开票额取账单总额；
//   - 余额 = 总额 − 结算净额，不钳零。

// writeBillSettledSum 写出账单带符号结算净额子表达式。
func writeBillSettledSum(builder *entsql.Builder, billID string) {
	builder.WriteString("(COALESCE((SELECT SUM(fva.amount) FROM finance_verification_allocations AS fva WHERE fva.bill_id = ")
	builder.Ident(billID)
	builder.WriteString(" AND fva.active = TRUE), 0) + COALESCE((SELECT SUM(fna.amount) FROM finance_netting_allocations AS fna WHERE fna.bill_id = ")
	builder.Ident(billID)
	builder.WriteString(" AND fna.active = TRUE), 0))")
}

// writeBillInvoicedSum 写出账单有效发票登记额子表达式。
func writeBillInvoicedSum(builder *entsql.Builder, billID string) {
	builder.WriteString("COALESCE((SELECT SUM(fib.amount) FROM finance_invoice_bills AS fib JOIN finance_invoices AS fi ON fi.id = fib.invoice_id WHERE fib.bill_id = ")
	builder.Ident(billID)
	builder.WriteString(" AND fib.active = TRUE AND fi.status IN ('DRAFT', 'ISSUED')), 0)")
}

func writeBillInvoiceExempt(builder *entsql.Builder, billID string) {
	// 与保护事实谓词共用同一有效性：撤销保留历史，但不再计入当前免票决定。
	builder.WriteString("EXISTS (SELECT 1 FROM finance_invoice_exemptions AS fie WHERE fie.bill_id = ")
	builder.Ident(billID)
	builder.WriteString(" AND fie.revoked_at IS NULL)")
}

// billAmountPredicate 以账单总额、结算净额、开票额组装谓词；write 负责写出完整布尔表达式。
func billAmountPredicate(write func(builder *entsql.Builder, total, billID string)) predicate.FinanceBill {
	return func(selector *entsql.Selector) {
		billID := selector.C(financebillent.FieldID)
		totalAmount := selector.C(financebillent.FieldTotalAmount)
		selector.Where(entsql.P(func(builder *entsql.Builder) {
			builder.WriteString("(")
			write(builder, totalAmount, billID)
			builder.WriteString(")")
		}))
	}
}

// billUnsettledPredicate 是「未结清」的 SQL 侧定义：总额 <> 结算净额（带符号比较）。
// 零额账单判已结清；负数更正账单在带符号净额等于总额时结清。
func billUnsettledPredicate() predicate.FinanceBill {
	return billAmountPredicate(func(builder *entsql.Builder, total, billID string) {
		builder.Ident(total)
		builder.WriteString(" <> ")
		writeBillSettledSum(builder, billID)
	})
}

// billPositiveOutstandingPredicate 要求余额为正（总额 > 结算净额），用于逾期与对冲候选等
// 只处理待收付正数余额的场景；与 biz.CalculateOverdueDays 的正数余额条件同源。
func billPositiveOutstandingPredicate() predicate.FinanceBill {
	return billAmountPredicate(func(builder *entsql.Builder, total, billID string) {
		builder.Ident(total)
		builder.WriteString(" > ")
		writeBillSettledSum(builder, billID)
	})
}

// billSettlementProgressPredicate 与 biz.ResolveFinanceBillProgress 的结算进度同源。
func billSettlementProgressPredicate(progress biz.FinanceBillSettlementProgress) predicate.FinanceBill {
	return billAmountPredicate(func(builder *entsql.Builder, total, billID string) {
		switch progress {
		case biz.FinanceBillSettlementSettled:
			builder.Ident(total)
			builder.WriteString(" = 0 OR ")
			builder.Ident(total)
			builder.WriteString(" = ")
			writeBillSettledSum(builder, billID)
		case biz.FinanceBillSettlementUnsettled:
			builder.Ident(total)
			builder.WriteString(" <> 0 AND ")
			writeBillSettledSum(builder, billID)
			builder.WriteString(" = 0")
		case biz.FinanceBillSettlementPartial:
			builder.Ident(total)
			builder.WriteString(" <> 0 AND ")
			writeBillSettledSum(builder, billID)
			builder.WriteString(" <> 0 AND ")
			builder.Ident(total)
			builder.WriteString(" <> ")
			writeBillSettledSum(builder, billID)
		default:
			builder.WriteString("FALSE")
		}
	})
}

// billInvoiceProgressPredicate 与 biz.ResolveFinanceBillProgress 的开票进度同源。
// 有效免票与零额共同组成 NOT_REQUIRED；其余分支显式排除有效免票。
func billInvoiceProgressPredicate(progress biz.FinanceBillInvoiceProgress) predicate.FinanceBill {
	return billAmountPredicate(func(builder *entsql.Builder, total, billID string) {
		if progress != biz.FinanceBillInvoiceNotRequired {
			builder.WriteString("NOT ")
			writeBillInvoiceExempt(builder, billID)
			builder.WriteString(" AND ")
		}
		switch progress {
		case biz.FinanceBillInvoiceNotRequired:
			builder.Ident(total)
			builder.WriteString(" = 0 OR ")
			writeBillInvoiceExempt(builder, billID)
		case biz.FinanceBillInvoiceNone:
			builder.Ident(total)
			builder.WriteString(" <> 0 AND ")
			writeBillInvoicedSum(builder, billID)
			builder.WriteString(" = 0")
		case biz.FinanceBillInvoiceFull:
			builder.Ident(total)
			builder.WriteString(" <> 0 AND ")
			writeBillInvoicedSum(builder, billID)
			builder.WriteString(" = ")
			builder.Ident(total)
		case biz.FinanceBillInvoicePartial:
			builder.Ident(total)
			builder.WriteString(" <> 0 AND ")
			writeBillInvoicedSum(builder, billID)
			builder.WriteString(" <> 0 AND ")
			writeBillInvoicedSum(builder, billID)
			builder.WriteString(" <> ")
			builder.Ident(total)
		default:
			builder.WriteString("FALSE")
		}
	})
}

// billCompletedPredicate 与 biz.ResolveFinanceBillProgress 的完成判定同源：
// 已确认，且零额账单，或结算净额与开票额均等于总额。
func billCompletedPredicate(completed bool) predicate.FinanceBill {
	done := financebillent.And(
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		billAmountPredicate(func(builder *entsql.Builder, total, billID string) {
			builder.Ident(total)
			builder.WriteString(" = 0 OR (")
			builder.Ident(total)
			builder.WriteString(" = ")
			writeBillSettledSum(builder, billID)
			builder.WriteString(" AND (")
			writeBillInvoiceExempt(builder, billID)
			builder.WriteString(" OR ")
			builder.Ident(total)
			builder.WriteString(" = ")
			writeBillInvoicedSum(builder, billID)
			builder.WriteString("))")
		}),
	)
	if completed {
		return done
	}
	return financebillent.Not(done)
}

func (r *financeBillRepo) GetPartnerUnsettledReceivableBaseAmount(ctx context.Context, organizationID, partnerID uuid.UUID) (decimal.Decimal, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	summaries, err := partnerCreditSummaries(ctx, client, organizationID, []uuid.UUID{partnerID})
	if err != nil {
		return decimal.Zero, err
	}
	summary := summaries[partnerID]
	if summary == nil {
		return decimal.Zero, nil
	}
	return summary.UnsettledReceivableBase, nil
}

func (r *financeBillRepo) GetPartnerCreditSummaries(ctx context.Context, organizationID uuid.UUID, partnerIDs []uuid.UUID) (map[uuid.UUID]*biz.PartnerCreditSummary, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return partnerCreditSummaries(ctx, client, organizationID, partnerIDs)
}

// unsettledReceivablePartyRow 是按结算单位聚合的应收折本币毛额行。
type unsettledReceivablePartyRow struct {
	SettlementPartyID uuid.UUID `json:"settlement_party_id"`
	BaseAmount        string    `json:"base_amount"`
}

// partnerCreditSummaries 汇总客户信用额度判定数据：
//   - 额度与默认账期来自客户角色唯一的收款约定；
//   - 未核销应收总额复用未结清口径（billUnsettledPredicate，带符号：总额 - 有效核销 - 有效对冲，
//     负数更正账单的负余额参与抵减），折本币后汇总；汇总结果为负时按零计（信用占用不为负）。
func partnerCreditSummaries(ctx context.Context, client *ent.Client, organizationID uuid.UUID, partnerIDs []uuid.UUID) (map[uuid.UUID]*biz.PartnerCreditSummary, error) {
	if organizationID == uuid.Nil || len(partnerIDs) == 0 {
		return map[uuid.UUID]*biz.PartnerCreditSummary{}, nil
	}
	ruleItems, err := client.PartnerSettlementRule.Query().
		Where(
			partnersettlementruleent.HasPartnerRoleWith(
				partnerroleent.RoleTypeEQ(partnerroleent.RoleTypeCustomer),
				partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID), partnerent.IDIn(partnerIDs...)),
			),
		).
		WithPartnerRole(func(query *ent.PartnerRoleQuery) {
			query.Select(partnerroleent.FieldPartnerID)
		}).
		Order(partnersettlementruleent.ByCreatedAt(), partnersettlementruleent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	summaries := make(map[uuid.UUID]*biz.PartnerCreditSummary, len(partnerIDs))
	for _, partnerID := range partnerIDs {
		summaries[partnerID] = &biz.PartnerCreditSummary{PartnerID: partnerID, UnsettledReceivableBase: decimal.Zero}
	}
	baseCurrency := ""
	for _, item := range ruleItems {
		if item.CreditLimitMinor != nil && *item.CreditLimitMinor > 0 {
			if baseCurrency == "" {
				baseCurrency, err = partnerCreditBaseCurrency(ctx, client, organizationID)
				if err != nil {
					return nil, err
				}
			}
			if item.CreditCurrency == nil || *item.CreditCurrency != baseCurrency {
				return nil, biz.ErrPartnerCreditCurrencyMismatch
			}
		}
		partnerID := item.Edges.PartnerRole.PartnerID
		summary := summaries[partnerID]
		summary.DefaultPaymentTermsDays = item.PaymentTermsDays
		if item.CreditLimitMinor != nil && *item.CreditLimitMinor > 0 {
			limit := decimal.NewFromInt(*item.CreditLimitMinor).Div(decimal.NewFromInt(100))
			summary.CreditLimitBase = &limit
			summary.CreditCurrency = item.CreditCurrency
		}
	}
	unsettledPredicates := []predicate.FinanceBill{
		financebillent.OrganizationIDEQ(organizationID),
		financebillent.SettlementPartyIDIn(partnerIDs...),
		financebillent.DirectionEQ(financebillent.DirectionRECEIVABLE),
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		billUnsettledPredicate(),
	}
	grossRows := make([]unsettledReceivablePartyRow, 0)
	if err := client.FinanceBill.Query().Where(unsettledPredicates...).
		GroupBy(financebillent.FieldSettlementPartyID).
		Aggregate(ent.As(ent.Sum(financebillent.FieldBaseCurrencyAmount), "base_amount")).
		Scan(ctx, &grossRows); err != nil {
		return nil, err
	}
	for _, row := range grossRows {
		amount, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		summaries[row.SettlementPartyID].UnsettledReceivableBase = amount
	}
	verificationAllocations, err := client.FinanceVerificationAllocation.Query().
		Where(verificationallocationent.ActiveEQ(true), verificationallocationent.HasBillWith(unsettledPredicates...)).
		WithBill(func(query *ent.FinanceBillQuery) {
			query.Select(financebillent.FieldID, financebillent.FieldSettlementPartyID)
		}).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range verificationAllocations {
		amount, parseErr := decimalOf(allocation.BillBaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bill, edgeErr := allocation.Edges.BillOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if summary := summaries[bill.SettlementPartyID]; summary != nil {
			summary.UnsettledReceivableBase = summary.UnsettledReceivableBase.Sub(amount)
		}
	}
	nettingAllocations, err := client.FinanceNettingAllocation.Query().
		Where(financenettingallocationent.ActiveEQ(true), financenettingallocationent.HasBillWith(unsettledPredicates...)).
		WithBill(func(query *ent.FinanceBillQuery) {
			query.Select(financebillent.FieldID, financebillent.FieldSettlementPartyID)
		}).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range nettingAllocations {
		amount, parseErr := decimalOf(allocation.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bill, edgeErr := allocation.Edges.BillOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if summary := summaries[bill.SettlementPartyID]; summary != nil {
			summary.UnsettledReceivableBase = summary.UnsettledReceivableBase.Sub(amount)
		}
	}
	for _, summary := range summaries {
		summary.UnsettledReceivableBase = summary.UnsettledReceivableBase.Round(8)
		if summary.UnsettledReceivableBase.IsNegative() {
			summary.UnsettledReceivableBase = decimal.Zero
		}
	}
	return summaries, nil
}

func billOverduePredicate(businessDate string) predicate.FinanceBill {
	return financebillent.And(
		financebillent.DirectionEQ(financebillent.DirectionRECEIVABLE),
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		financebillent.DueDateNotNil(),
		financebillent.DueDateNEQ(""),
		financebillent.DueDateLT(businessDate),
		billPositiveOutstandingPredicate(),
	)
}
