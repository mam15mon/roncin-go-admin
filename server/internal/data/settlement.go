package data

import (
	"context"
	"sort"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeeenterprisetag "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneralias "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type settlementRepo struct {
	data *Data
}

func financeLockedOrderPredicate() predicate.Order {
	return order.HasFinanceCommissionLinesWith(
		financecommissionline.HasCommissionWith(
			financecommission.StatusIn(financecommission.StatusCONFIRMED, financecommission.StatusPAID),
		),
	)
}

// feeLedgerFinancialProgressPredicate 将费用的账单、开票和「有效结清金额」组合状态下推到数据库。
// 这里使用固定表名的相关子查询，是因为该组合状态并非单一 Ent 字段，无法用普通字段谓词表达。
// 有效结清金额 = 有效核销分摊（allocation active 且核销单 ACTIVE）+ 有效对冲分摊（allocation active 且对冲单 CONFIRMED），
// 与行投影函数 applyFeeLedgerBillProjection 的预加载口径必须一致。
func feeLedgerFinancialProgressPredicate(progress biz.FeeLedgerFinancialProgress) predicate.OrderFee {
	return func(selector *entsql.Selector) {
		feeID := selector.C(orderfee.FieldID)
		selector.Where(entsql.P(func(builder *entsql.Builder) {
			if progress == biz.FeeLedgerUnbilled {
				builder.WriteString("NOT EXISTS (SELECT 1 FROM finance_bill_lines AS fbl JOIN finance_bills AS fb ON fb.id = fbl.bill_id WHERE fbl.order_fee_id = ")
				builder.Ident(feeID)
				builder.WriteString(" AND fbl.active = TRUE AND fb.status <> ").Arg(financebill.StatusCANCELLED)
				builder.WriteString(")")
				return
			}

			builder.WriteString("EXISTS (SELECT 1 FROM finance_bill_lines AS fbl JOIN finance_bills AS fb ON fb.id = fbl.bill_id WHERE fbl.order_fee_id = ")
			builder.Ident(feeID)
			builder.WriteString(" AND fbl.active = TRUE AND fb.status <> ").Arg(financebill.StatusCANCELLED)
			builder.WriteString(" AND ")
			if progress == biz.FeeLedgerInvoicedUnverified || progress == biz.FeeLedgerInvoicedPartiallyVerified || progress == biz.FeeLedgerCompleted {
				builder.WriteString("EXISTS")
			} else {
				builder.WriteString("NOT EXISTS")
			}
			builder.WriteString(" (SELECT 1 FROM finance_invoice_bills AS fib JOIN finance_invoices AS fi ON fi.id = fib.invoice_id WHERE fib.bill_id = fb.id AND fib.active = TRUE AND fi.status = ").Arg(financeinvoice.StatusISSUED)
			builder.WriteString(") AND ")

			writeSettledAmount := func() {
				builder.WriteString("(COALESCE((SELECT SUM(fva.amount) FROM finance_verification_allocations AS fva JOIN finance_verifications AS fv ON fv.id = fva.verification_id WHERE fva.bill_id = fb.id AND fva.active = TRUE AND fv.status = ").Arg(financeverification.StatusACTIVE)
				builder.WriteString("), 0) + COALESCE((SELECT SUM(fna.amount) FROM finance_netting_allocations AS fna JOIN finance_nettings AS fnt ON fnt.id = fna.netting_id WHERE fna.bill_id = fb.id AND fna.active = TRUE AND fnt.status = ").Arg(financenetting.StatusCONFIRMED)
				builder.WriteString("), 0))")
			}
			switch progress {
			case biz.FeeLedgerUnverifiedUninvoiced, biz.FeeLedgerInvoicedUnverified:
				writeSettledAmount()
				builder.WriteString(" <= 0")
			case biz.FeeLedgerPartiallyVerifiedUninvoiced, biz.FeeLedgerInvoicedPartiallyVerified:
				writeSettledAmount()
				builder.WriteString(" > 0 AND ")
				writeSettledAmount()
				builder.WriteString(" < fb.total_amount")
			case biz.FeeLedgerVerifiedUninvoiced, biz.FeeLedgerCompleted:
				writeSettledAmount()
				builder.WriteString(" >= fb.total_amount")
			}
			builder.WriteString(")")
		}))
	}
}

type feeLedgerSummaryRow struct {
	Direction    string `json:"direction"`
	BaseCurrency string `json:"base_currency"`
	ActiveCount  int64  `json:"active_count"`
	BaseAmount   string `json:"base_amount"`
}

// feeLedgerBillLineQuery 是费用台账活动账单关系的统一加载条件：活动行、非取消账单、
// 有效开票链接、有效核销分摊与有效对冲分摊。列表与订单详情必须共用同一加载条件，
// 保证 bill_no 与 financial_progress 不产生两套口径。
func feeLedgerBillLineQuery(lineQuery *ent.FinanceBillLineQuery) {
	lineQuery.
		Where(
			financebillline.ActiveEQ(true),
			financebillline.HasBillWith(financebill.StatusNEQ(financebill.StatusCANCELLED)),
		).
		WithBill(func(billQuery *ent.FinanceBillQuery) {
			billQuery.
				WithInvoiceLinks(func(linkQuery *ent.FinanceInvoiceBillQuery) {
					linkQuery.Where(
						financeinvoicebill.ActiveEQ(true),
						financeinvoicebill.HasInvoiceWith(financeinvoice.StatusEQ(financeinvoice.StatusISSUED)),
					)
				}).
				WithVerificationAllocations(func(allocationQuery *ent.FinanceVerificationAllocationQuery) {
					allocationQuery.Where(
						financeverificationallocation.ActiveEQ(true),
						financeverificationallocation.HasVerificationWith(financeverification.StatusEQ(financeverification.StatusACTIVE)),
					)
				}).
				WithNettingAllocations(func(allocationQuery *ent.FinanceNettingAllocationQuery) {
					allocationQuery.Where(
						financenettingallocation.ActiveEQ(true),
						financenettingallocation.HasNettingWith(financenetting.StatusEQ(financenetting.StatusCONFIRMED)),
					)
				})
		})
}

// applyFeeLedgerBillProjection 把费用已经按 feeLedgerBillLineQuery 装载的活动账单关系
// 投影为账单号与财务进度。有效结清金额 = 有效核销分摊 + 有效对冲分摊，两类一对多分摊
// 分别预加载后在此聚合，禁止直接 join 两表聚合造成金额倍增；列表与订单详情共用本函数。
func applyFeeLedgerBillProjection(ledgerItem *biz.FeeLedgerItem, feeStatus biz.OrderFeeStatus, billLines []*ent.FinanceBillLine) error {
	if feeStatus == biz.OrderFeeCancelled {
		ledgerItem.FinancialProgress = ""
	}
	if len(billLines) == 0 {
		return nil
	}
	bill, billErr := billLines[0].Edges.BillOrErr()
	if billErr != nil {
		return billErr
	}
	billAmount, parseErr := decimalOf(bill.TotalAmount)
	if parseErr != nil {
		return parseErr
	}
	invoiceLinks, linkErr := bill.Edges.InvoiceLinksOrErr()
	if linkErr != nil {
		return linkErr
	}
	settledAmount := decimal.Zero
	verificationAllocations, verificationErr := bill.Edges.VerificationAllocationsOrErr()
	if verificationErr != nil {
		return verificationErr
	}
	for _, allocation := range verificationAllocations {
		amount, amountErr := decimalOf(allocation.Amount)
		if amountErr != nil {
			return amountErr
		}
		settledAmount = settledAmount.Add(amount)
	}
	nettingAllocations, nettingErr := bill.Edges.NettingAllocationsOrErr()
	if nettingErr != nil {
		return nettingErr
	}
	for _, allocation := range nettingAllocations {
		amount, amountErr := decimalOf(allocation.Amount)
		if amountErr != nil {
			return amountErr
		}
		settledAmount = settledAmount.Add(amount)
	}
	ledgerItem.BillNo = bill.BillNo
	ledgerItem.FinancialProgress = biz.ResolveFeeLedgerFinancialProgress(true, len(invoiceLinks) > 0, billAmount, settledAmount)
	return nil
}

// feeLedgerOrderQuery 装载台账列表与订单详情共用的订单关系。提成线加载条件与
// financeLockedOrderPredicate 的 SQL 口径一致（存在 CONFIRMED/PAID 提成线即财务锁定）。
func feeLedgerOrderQuery(query *ent.OrderQuery) *ent.OrderQuery {
	return query.
		WithOrganization().
		WithCustomer().
		WithFinanceCommissionLines(func(lineQuery *ent.FinanceCommissionLineQuery) {
			lineQuery.Where(
				financecommissionline.HasCommissionWith(
					financecommission.StatusIn(financecommission.StatusCONFIRMED, financecommission.StatusPAID),
				),
			)
		})
}

func NewSettlementRepo(data *Data) biz.SettlementRepo {
	return &settlementRepo{data: data}
}

func (r *settlementRepo) ListFinanceOrganizations(ctx context.Context, organizationIDs []uuid.UUID, keyword string) ([]*biz.FinanceOrganizationOption, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.Organization{organizationent.IDIn(organizationIDs...), organizationent.EnabledEQ(true)}
	if keyword != "" {
		predicates = append(predicates, organizationent.Or(organizationent.NameContainsFold(keyword), organizationent.CodeContainsFold(keyword), organizationent.SearchKeywordsContainsFold(keyword)))
	}
	items, err := client.Organization.Query().Where(predicates...).Order(organizationent.ByCode(), organizationent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.FinanceOrganizationOption, 0, len(items))
	for _, item := range items {
		baseCurrency := ""
		if item.BaseCurrency != nil {
			baseCurrency = *item.BaseCurrency
		}
		result = append(result, &biz.FinanceOrganizationOption{ID: item.ID, Code: item.Code, Name: item.Name, BaseCurrency: baseCurrency})
	}
	return result, nil
}

func (r *settlementRepo) ListFinanceSettlementParties(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*biz.FinanceSettlementPartyOption, int64, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}
	query := client.Partner.Query().Where(partner.OrganizationIDEQ(organizationID), partner.EnabledEQ(true))
	if keyword != "" {
		query.Where(partner.Or(
			partner.CodeContainsFold(keyword),
			partner.LegalNameContainsFold(keyword),
			partner.SearchKeywordsContainsFold(keyword),
			partner.HasAliasesWith(partneralias.Or(partneralias.AliasNameContainsFold(keyword), partneralias.SearchKeywordsContainsFold(keyword))),
		))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.Order(partner.ByLegalName(), partner.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.FinanceSettlementPartyOption, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.FinanceSettlementPartyOption{ID: item.ID.String(), Code: item.Code, Name: item.LegalName})
	}
	return result, int64(total), nil
}

func (r *settlementRepo) ResolveFeeLedgerOrganization(ctx context.Context, organizationIDs, feeIDs []uuid.UUID) (uuid.UUID, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	items, err := client.OrderFee.Query().Where(orderfee.IDIn(feeIDs...), orderfee.HasOrderWith(order.OrganizationIDIn(organizationIDs...))).WithOrder().All(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if len(items) != len(feeIDs) {
		return uuid.Nil, biz.ErrFinanceLedgerInvalidArgument
	}
	organizationID := items[0].Edges.Order.OrganizationID
	for _, item := range items[1:] {
		if item.Edges.Order.OrganizationID != organizationID {
			return uuid.Nil, biz.ErrFinanceLedgerInvalidArgument
		}
	}
	return organizationID, nil
}

func (r *settlementRepo) GetFeeLedgerOrderDetail(ctx context.Context, organizationIDs []uuid.UUID, orderID uuid.UUID) (*biz.FeeLedgerOrderDetail, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := feeLedgerOrderQuery(client.Order.Query().
		Where(order.IDEQ(orderID), order.OrganizationIDIn(organizationIDs...))).
		WithFees(func(query *ent.OrderFeeQuery) {
			query.WithSettlementParty().
				WithFinanceBillLines(feeLedgerBillLineQuery).
				Order(orderfee.ByExpenseDate(entsql.OrderDesc()), orderfee.ByID())
		}).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceLedgerInvalidArgument, nil)
	}
	organization, err := item.Edges.OrganizationOrErr()
	if err != nil {
		return nil, err
	}
	customer, err := item.Edges.CustomerOrErr()
	if err != nil {
		return nil, err
	}
	financeLockLines, err := item.Edges.FinanceCommissionLinesOrErr()
	if err != nil {
		return nil, err
	}
	detail := &biz.FeeLedgerOrderDetail{OrderID: item.ID.String(), OrderNo: item.OrderNo, Business: string(item.BusinessType), CustomerName: customer.LegalName, OrganizationID: item.OrganizationID, OrganizationName: organization.Name}
	buckets := make(map[string]*biz.FeeLedgerBaseCurrencyAmount)
	for _, feeEntity := range item.Edges.Fees {
		fee, convertErr := orderFeeToBiz(feeEntity)
		if convertErr != nil {
			return nil, convertErr
		}
		ledgerItem := &biz.FeeLedgerItem{Fee: fee, OrganizationID: item.OrganizationID, OrganizationName: organization.Name, OrderNo: item.OrderNo, Business: string(item.BusinessType), CustomerID: customer.ID, CustomerName: customer.LegalName, FinancialProgress: biz.FeeLedgerUnbilled, FinanceLocked: len(financeLockLines) > 0}
		billLines, edgeErr := feeEntity.Edges.FinanceBillLinesOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if projectionErr := applyFeeLedgerBillProjection(ledgerItem, fee.Status, billLines); projectionErr != nil {
			return nil, projectionErr
		}
		detail.Items = append(detail.Items, ledgerItem)
		bucket := buckets[fee.BaseCurrency]
		if bucket == nil {
			bucket = &biz.FeeLedgerBaseCurrencyAmount{BaseCurrency: fee.BaseCurrency}
			buckets[fee.BaseCurrency] = bucket
		}
		if fee.Direction == biz.OrderFeeReceivable {
			bucket.ReceivableBaseAmount = bucket.ReceivableBaseAmount.Add(fee.BaseCurrencyAmount)
		} else {
			bucket.PayableBaseAmount = bucket.PayableBaseAmount.Add(fee.BaseCurrencyAmount)
		}
	}
	for _, bucket := range buckets {
		bucket.ProfitBaseAmount = bucket.ReceivableBaseAmount.Sub(bucket.PayableBaseAmount)
		detail.AmountsByBaseCurrency = append(detail.AmountsByBaseCurrency, *bucket)
	}
	sort.Slice(detail.AmountsByBaseCurrency, func(i, j int) bool {
		return detail.AmountsByBaseCurrency[i].BaseCurrency < detail.AmountsByBaseCurrency[j].BaseCurrency
	})
	return detail, nil
}

func (r *settlementRepo) ListFeeLedger(ctx context.Context, organizationIDs []uuid.UUID, filter biz.FeeLedgerFilter) (*biz.FeeLedgerResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.OrderFee{orderfee.HasOrderWith(order.OrganizationIDIn(organizationIDs...))}
	if filter.Keyword != "" {
		predicates = append(predicates, orderfee.Or(
			orderfee.FeeCodeContainsFold(filter.Keyword),
			orderfee.FeeNameContainsFold(filter.Keyword),
			orderfee.HasOrderWith(order.OrderNoContainsFold(filter.Keyword)),
			orderfee.HasSettlementPartyWith(partner.LegalNameContainsFold(filter.Keyword)),
			orderfee.HasOrderWith(order.HasCustomerWith(partner.LegalNameContainsFold(filter.Keyword))),
			orderfee.HasFinanceBillLinesWith(
				financebillline.ActiveEQ(true),
				financebillline.HasBillWith(
					financebill.StatusNEQ(financebill.StatusCANCELLED),
					financebill.BillNoContainsFold(filter.Keyword),
				),
			),
		))
	}
	if filter.BusinessType != "" {
		predicates = append(predicates, orderfee.HasOrderWith(order.BusinessTypeEQ(order.BusinessType(filter.BusinessType))))
	}
	if filter.Direction != "" {
		predicates = append(predicates, orderfee.DirectionEQ(orderfee.Direction(filter.Direction)))
	}
	if filter.Status != "" {
		predicates = append(predicates, orderfee.StatusEQ(orderfee.Status(filter.Status)))
	}
	if filter.SettlementPartyID != nil {
		predicates = append(predicates, orderfee.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.CustomerID != nil {
		predicates = append(predicates, orderfee.HasOrderWith(order.CustomerIDEQ(*filter.CustomerID)))
	}
	if filter.Currency != "" {
		predicates = append(predicates, orderfee.CurrencyEQ(filter.Currency))
	}
	if filter.BillNo != "" {
		predicates = append(predicates, orderfee.HasFinanceBillLinesWith(
			financebillline.ActiveEQ(true),
			financebillline.HasBillWith(
				financebill.StatusNEQ(financebill.StatusCANCELLED),
				financebill.BillNoContainsFold(filter.BillNo),
			),
		))
	}
	if filter.ExpenseDateFrom != "" {
		predicates = append(predicates, orderfee.ExpenseDateGTE(filter.ExpenseDateFrom))
	}
	if filter.ExpenseDateTo != "" {
		predicates = append(predicates, orderfee.ExpenseDateLTE(filter.ExpenseDateTo))
	}
	if filter.FinanceLocked != nil {
		lockedOrder := financeLockedOrderPredicate()
		if !*filter.FinanceLocked {
			lockedOrder = order.Not(lockedOrder)
		}
		predicates = append(predicates, orderfee.HasOrderWith(lockedOrder))
	}
	if len(filter.TagIDs) > 0 {
		predicates = append(predicates, orderfee.HasEnterpriseTagLinksWith(orderfeeenterprisetag.TagResourceIDIn(filter.TagIDs...)))
	}
	if filter.FinancialProgress != "" {
		predicates = append(predicates,
			orderfee.StatusNEQ(orderfee.StatusCANCELLED),
			feeLedgerFinancialProgressPredicate(filter.FinancialProgress),
		)
	}

	baseQuery := client.OrderFee.Query().Where(predicates...)
	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	summaryRows := make([]feeLedgerSummaryRow, 0)
	if err := baseQuery.Clone().
		Where(orderfee.StatusNEQ(orderfee.StatusCANCELLED)).
		GroupBy(orderfee.FieldDirection, orderfee.FieldBaseCurrency).
		Aggregate(
			ent.As(ent.Count(), "active_count"),
			ent.As(ent.Sum(orderfee.FieldBaseCurrencyAmount), "base_amount"),
		).
		Scan(ctx, &summaryRows); err != nil {
		return nil, err
	}
	summary := biz.FeeLedgerSummary{}
	buckets := make(map[string]*biz.FeeLedgerBaseCurrencyAmount, len(summaryRows))
	for _, row := range summaryRows {
		amount, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		summary.ActiveCount += row.ActiveCount
		bucket := buckets[row.BaseCurrency]
		if bucket == nil {
			bucket = &biz.FeeLedgerBaseCurrencyAmount{BaseCurrency: row.BaseCurrency}
			buckets[row.BaseCurrency] = bucket
		}
		if row.Direction == string(orderfee.DirectionRECEIVABLE) {
			bucket.ReceivableBaseAmount = bucket.ReceivableBaseAmount.Add(amount)
		} else {
			bucket.PayableBaseAmount = bucket.PayableBaseAmount.Add(amount)
		}
	}
	for _, bucket := range buckets {
		bucket.ProfitBaseAmount = bucket.ReceivableBaseAmount.Sub(bucket.PayableBaseAmount)
		summary.AmountsByBaseCurrency = append(summary.AmountsByBaseCurrency, *bucket)
	}
	sort.Slice(summary.AmountsByBaseCurrency, func(i, j int) bool {
		return summary.AmountsByBaseCurrency[i].BaseCurrency < summary.AmountsByBaseCurrency[j].BaseCurrency
	})

	items, err := baseQuery.Clone().
		WithSettlementParty().
		WithOrder(func(query *ent.OrderQuery) { feeLedgerOrderQuery(query) }).
		WithFinanceBillLines(feeLedgerBillLineQuery).
		Order(orderfee.ByExpenseDate(entsql.OrderDesc()), orderfee.ByCreatedAt(entsql.OrderDesc()), orderfee.ByID(entsql.OrderDesc())).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	resultItems := make([]*biz.FeeLedgerItem, 0, len(items))
	for _, item := range items {
		fee, convertErr := orderFeeToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		businessOrder, edgeErr := item.Edges.OrderOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		customer, edgeErr := businessOrder.Edges.CustomerOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		ledgerItem := &biz.FeeLedgerItem{
			Fee:               fee,
			OrganizationID:    businessOrder.OrganizationID,
			OrderNo:           businessOrder.OrderNo,
			Business:          string(businessOrder.BusinessType),
			CustomerID:        customer.ID,
			CustomerName:      customer.LegalName,
			FinancialProgress: biz.FeeLedgerUnbilled,
		}
		organization, organizationErr := businessOrder.Edges.OrganizationOrErr()
		if organizationErr != nil {
			return nil, organizationErr
		}
		ledgerItem.OrganizationName = organization.Name
		financeLockLines, edgeErr := businessOrder.Edges.FinanceCommissionLinesOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		ledgerItem.FinanceLocked = len(financeLockLines) > 0
		billLines, edgeErr := item.Edges.FinanceBillLinesOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if projectionErr := applyFeeLedgerBillProjection(ledgerItem, fee.Status, billLines); projectionErr != nil {
			return nil, projectionErr
		}
		resultItems = append(resultItems, ledgerItem)
	}
	result := &biz.FeeLedgerResult{
		Items:   resultItems,
		Total:   int64(total),
		Summary: summary,
	}
	return result, nil
}

var _ biz.SettlementRepo = (*settlementRepo)(nil)
