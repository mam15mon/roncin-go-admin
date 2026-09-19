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

// billUnsettledPredicate 是「未结清」的 SQL 侧定义：总额 > 有效核销 + 有效对冲（原值比较，不钳负）。
// 展示侧的等价定义在 enrichVerificationAmounts（UnverifiedAmount 负值钳零），两处口径须同步维护。
func billUnsettledPredicate() predicate.FinanceBill {
	return func(selector *entsql.Selector) {
		billID := selector.C(financebillent.FieldID)
		totalAmount := selector.C(financebillent.FieldTotalAmount)
		selector.Where(entsql.P(func(builder *entsql.Builder) {
			builder.WriteString("(")
			builder.Ident(totalAmount)
			builder.WriteString(" > (COALESCE((SELECT SUM(fva.amount) FROM finance_verification_allocations AS fva WHERE fva.bill_id = ")
			builder.Ident(billID)
			builder.WriteString(" AND fva.active = TRUE), 0) + COALESCE((SELECT SUM(fna.amount) FROM finance_netting_allocations AS fna WHERE fna.bill_id = ")
			builder.Ident(billID)
			builder.WriteString(" AND fna.active = TRUE), 0)))")
		}))
	}
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
//   - 额度与默认账期来自客户角色的激活结算规则（多个激活规则取最大值，不收窄商业弹性）；
//   - 未核销应收总额复用 09-10 修复后的未结清口径（billUnsettledPredicate + 总额 - 有效核销 - 有效对冲，
//     折本币、负值钳零），与账单列表汇总的 unverified_base_amount 保持一致。
func partnerCreditSummaries(ctx context.Context, client *ent.Client, organizationID uuid.UUID, partnerIDs []uuid.UUID) (map[uuid.UUID]*biz.PartnerCreditSummary, error) {
	if organizationID == uuid.Nil || len(partnerIDs) == 0 {
		return map[uuid.UUID]*biz.PartnerCreditSummary{}, nil
	}
	ruleItems, err := client.PartnerSettlementRule.Query().
		Where(
			partnersettlementruleent.IsActiveEQ(true),
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
	for _, item := range ruleItems {
		partnerID := item.Edges.PartnerRole.PartnerID
		summary := summaries[partnerID]
		if item.PaymentTermsDays != nil && (summary.DefaultPaymentTermsDays == nil || *item.PaymentTermsDays > *summary.DefaultPaymentTermsDays) {
			value := *item.PaymentTermsDays
			summary.DefaultPaymentTermsDays = &value
		}
		if item.CreditLimitMinor != nil && *item.CreditLimitMinor > 0 {
			limit := decimal.NewFromInt(*item.CreditLimitMinor).Div(decimal.NewFromInt(100))
			if summary.CreditLimitBase == nil || limit.GreaterThan(*summary.CreditLimitBase) {
				summary.CreditLimitBase = &limit
				summary.CreditCurrency = item.CreditCurrency
			}
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
		billUnsettledPredicate(),
	)
}
