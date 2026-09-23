package data

import (
	"context"
	"sort"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	billtaglink "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillenterprisetag"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	partneraliasent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

type financeBillRepo struct{ data *Data }

type financeBillSummaryRow struct {
	Direction    string `json:"direction"`
	BaseCurrency string `json:"base_currency"`
	BaseAmount   string `json:"base_amount"`
}

func NewFinanceBillRepo(data *Data) biz.FinanceBillRepo { return &financeBillRepo{data: data} }

func (r *financeBillRepo) List(ctx context.Context, organizationIDs []uuid.UUID, filter biz.FinanceBillFilter) (*biz.FinanceBillListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceBill{financeBillOrganizationScopePredicate(organizationIDs)}
	if filter.Keyword != "" {
		predicates = append(predicates, financebillent.Or(
			financebillent.BillNoContainsFold(filter.Keyword),
			financebillent.SettlementPartyNameContainsFold(filter.Keyword),
			financebillent.HasLinesWith(financebilllineent.OrderNoContainsFold(filter.Keyword)),
		))
	}
	if filter.Direction != "" {
		predicates = append(predicates, financebillent.DirectionEQ(financebillent.Direction(filter.Direction)))
	}
	if filter.Status != "" {
		predicates = append(predicates, financebillent.StatusEQ(financebillent.Status(filter.Status)))
	}
	if filter.SettlementPartyID != nil {
		predicates = append(predicates, financebillent.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.Currency != "" {
		predicates = append(predicates, financebillent.CurrencyEQ(filter.Currency))
	}
	if filter.BillDateFrom != "" {
		predicates = append(predicates, financebillent.BillDateGTE(filter.BillDateFrom))
	}
	if filter.BillDateTo != "" {
		predicates = append(predicates, financebillent.BillDateLTE(filter.BillDateTo))
	}
	if filter.DueDateFrom != "" {
		predicates = append(predicates, financebillent.DueDateGTE(filter.DueDateFrom))
	}
	if filter.DueDateTo != "" {
		predicates = append(predicates, financebillent.DueDateLTE(filter.DueDateTo))
	}
	if filter.OnlyUnsettled {
		if filter.Status == "" {
			predicates = append(predicates, financebillent.StatusEQ(financebillent.StatusCONFIRMED))
		}
		predicates = append(predicates, billUnsettledPredicate())
	}
	currentBusinessDate := time.Now().In(biz.ExchangeRateBusinessLocation()).Format("2006-01-02")
	if filter.OnlyOverdue {
		predicates = append(predicates, billOverduePredicate(currentBusinessDate))
	}
	if len(filter.TagIDs) > 0 {
		predicates = append(predicates, financebillent.HasEnterpriseTagLinksWith(billtaglink.TagResourceIDIn(filter.TagIDs...)))
	}
	query := client.FinanceBill.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	summaryPredicates := append([]predicate.FinanceBill{}, predicates...)
	summaryPredicates = append(summaryPredicates, financebillent.StatusEQ(financebillent.StatusCONFIRMED))
	summaryRows := make([]financeBillSummaryRow, 0)
	if err := client.FinanceBill.Query().Where(summaryPredicates...).
		GroupBy(financebillent.FieldDirection, financebillent.FieldBaseCurrency).
		Aggregate(ent.As(ent.Sum(financebillent.FieldBaseCurrencyAmount), "base_amount")).
		Scan(ctx, &summaryRows); err != nil {
		return nil, err
	}
	summary := biz.FinanceBillSummary{}
	amountsByBaseCurrency := make(map[string]*biz.FinanceBaseCurrencyAmount, len(summaryRows))
	for _, row := range summaryRows {
		value, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, row.BaseCurrency)
		if row.Direction == string(financebillent.DirectionRECEIVABLE) {
			bucket.ReceivableBaseAmount = bucket.ReceivableBaseAmount.Add(value)
		} else {
			bucket.PayableBaseAmount = bucket.PayableBaseAmount.Add(value)
		}
	}
	for _, bucket := range amountsByBaseCurrency {
		bucket.UnverifiedBaseAmount = bucket.ReceivableBaseAmount.Add(bucket.PayableBaseAmount)
	}
	allocations, err := client.FinanceVerificationAllocation.Query().
		Where(verificationallocationent.ActiveEQ(true), verificationallocationent.HasBillWith(summaryPredicates...)).
		WithBill(func(query *ent.FinanceBillQuery) {
			query.Select(financebillent.FieldID, financebillent.FieldBaseCurrency)
		}).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, allocation := range allocations {
		amount, parseErr := decimalOf(allocation.BillBaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bill, edgeErr := allocation.Edges.BillOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, bill.BaseCurrency)
		bucket.UnverifiedBaseAmount = bucket.UnverifiedBaseAmount.Sub(amount)
	}
	nettingAllocations, err := client.FinanceNettingAllocation.Query().
		Where(financenettingallocationent.ActiveEQ(true), financenettingallocationent.HasBillWith(summaryPredicates...)).
		WithBill(func(query *ent.FinanceBillQuery) {
			query.Select(financebillent.FieldID, financebillent.FieldBaseCurrency)
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
		bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, bill.BaseCurrency)
		bucket.UnverifiedBaseAmount = bucket.UnverifiedBaseAmount.Sub(amount)
	}
	overduePredicates := append([]predicate.FinanceBill{}, summaryPredicates...)
	overduePredicates = append(overduePredicates,
		financebillent.DirectionEQ(financebillent.DirectionRECEIVABLE),
		financebillent.DueDateNotNil(),
		financebillent.DueDateNEQ(""),
		financebillent.DueDateLT(currentBusinessDate),
		billUnsettledPredicate(),
	)
	var overdueSummaryRows []financeBillSummaryRow
	if err := client.FinanceBill.Query().Where(overduePredicates...).
		GroupBy(financebillent.FieldBaseCurrency).
		Aggregate(ent.As(ent.Sum(financebillent.FieldBaseCurrencyAmount), "base_amount")).
		Scan(ctx, &overdueSummaryRows); err != nil {
		return nil, err
	}
	for _, row := range overdueSummaryRows {
		value, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, row.BaseCurrency)
		bucket.OverdueReceivableBaseAmount = bucket.OverdueReceivableBaseAmount.Add(value)
	}
	if len(overdueSummaryRows) > 0 {
		overdueAllocations, err := client.FinanceVerificationAllocation.Query().
			Where(verificationallocationent.ActiveEQ(true), verificationallocationent.HasBillWith(overduePredicates...)).
			WithBill(func(query *ent.FinanceBillQuery) {
				query.Select(financebillent.FieldID, financebillent.FieldBaseCurrency)
			}).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, allocation := range overdueAllocations {
			amount, parseErr := decimalOf(allocation.BillBaseAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			bill, edgeErr := allocation.Edges.BillOrErr()
			if edgeErr != nil {
				return nil, edgeErr
			}
			bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, bill.BaseCurrency)
			bucket.OverdueReceivableBaseAmount = bucket.OverdueReceivableBaseAmount.Sub(amount)
		}

		overdueNettings, err := client.FinanceNettingAllocation.Query().
			Where(financenettingallocationent.ActiveEQ(true), financenettingallocationent.HasBillWith(overduePredicates...)).
			WithBill(func(query *ent.FinanceBillQuery) {
				query.Select(financebillent.FieldID, financebillent.FieldBaseCurrency)
			}).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, allocation := range overdueNettings {
			amount, parseErr := decimalOf(allocation.BaseCurrencyAmount)
			if parseErr != nil {
				return nil, parseErr
			}
			bill, edgeErr := allocation.Edges.BillOrErr()
			if edgeErr != nil {
				return nil, edgeErr
			}
			bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, bill.BaseCurrency)
			bucket.OverdueReceivableBaseAmount = bucket.OverdueReceivableBaseAmount.Sub(amount)
		}
	}
	for _, bucket := range amountsByBaseCurrency {
		if bucket.OverdueReceivableBaseAmount.IsNegative() {
			bucket.OverdueReceivableBaseAmount = decimal.Zero
		} else {
			bucket.OverdueReceivableBaseAmount = bucket.OverdueReceivableBaseAmount.Round(8)
		}
	}
	summary.AmountsByBaseCurrency = financeBaseCurrencyAmountItems(amountsByBaseCurrency)

	items, err := query.WithBatch().WithOrganization().Order(financebillent.ByBillDate(entsql.OrderDesc()), financebillent.ByCreatedAt(entsql.OrderDesc()), financebillent.ByID(entsql.OrderDesc())).
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceBillListResult{Items: make([]*biz.FinanceBill, 0, len(items)), Total: int64(total), Summary: summary}
	for _, item := range items {
		converted, convertErr := financeBillToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	if err := r.enrichVerificationAmounts(ctx, result.Items, currentBusinessDate); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *financeBillRepo) Get(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceBill, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.FinanceBill.Query().WithOrganization().Where(financebillent.IDEQ(id), financeBillOrganizationScopePredicate(organizationIDs))
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForUpdate()
	}
	item, err := r.financeBillQueryWithLines(query).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceBillNotFound, nil)
	}
	converted, err := financeBillToBiz(item)
	if err != nil {
		return nil, err
	}
	if err = r.enrichVerificationAmounts(ctx, []*biz.FinanceBill{converted}, time.Now().In(biz.ExchangeRateBusinessLocation()).Format("2006-01-02")); err != nil {
		return nil, err
	}
	return converted, nil
}

func (r *financeBillRepo) enrichVerificationAmounts(ctx context.Context, bills []*biz.FinanceBill, currentBusinessDate string) error {
	if len(bills) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(bills))
	byID := make(map[uuid.UUID]*biz.FinanceBill, len(bills))
	for _, bill := range bills {
		ids = append(ids, bill.ID)
		byID[bill.ID] = bill
		bill.VerifiedAmount = decimal.Zero
		bill.NettedAmount = decimal.Zero
		bill.UnverifiedAmount = bill.TotalAmount
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	allocations, err := client.FinanceVerificationAllocation.Query().Where(verificationallocationent.BillIDIn(ids...), verificationallocationent.ActiveEQ(true)).All(ctx)
	if err != nil {
		return err
	}
	for _, allocation := range allocations {
		amount, err := decimalOf(allocation.Amount)
		if err != nil {
			return err
		}
		bill := byID[allocation.BillID]
		bill.VerifiedAmount = bill.VerifiedAmount.Add(amount)
	}
	nettedSums, err := loadFinanceBillNettedAmounts(ctx, client, ids)
	if err != nil {
		return err
	}
	// 未核销余额 = 总额 - 有效核销 - 有效对冲；普通资金核销只处理抵销后的剩余余额。
	// 展示侧负值钳零；SQL 侧未结清谓词（billUnsettledPredicate）以原值比较，两处口径须同步维护。
	// currentBusinessDate 由调用方传入：列表场景与筛选谓词共用同一业务日，避免跨上海午夜的不一致。
	for _, bill := range bills {
		bill.VerifiedAmount = bill.VerifiedAmount.Round(8)
		bill.NettedAmount = nettedSums[bill.ID].Round(8)
		bill.UnverifiedAmount = bill.TotalAmount.Sub(bill.VerifiedAmount).Sub(bill.NettedAmount).Round(8)
		if bill.UnverifiedAmount.IsNegative() {
			bill.UnverifiedAmount = decimal.Zero
		}
		bill.OverdueDays = biz.CalculateOverdueDays(bill.Direction, bill.Status, bill.UnverifiedAmount, bill.DueDate, currentBusinessDate)
	}
	return nil
}

func (r *financeBillRepo) GetByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*biz.FinanceBill, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := r.financeBillQueryWithLines(client.FinanceBill.Query().WithOrganization()).
		Where(financebillent.OrganizationIDEQ(organizationID), financebillent.IdempotencyKeyEQ(idempotencyKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return financeBillToBiz(item)
}

func financeBillOrganizationScopePredicate(organizationIDs []uuid.UUID) predicate.FinanceBill {
	return financebillent.OrganizationIDIn(organizationIDs...)
}

func (r *financeBillRepo) financeBillQueryWithLines(query *ent.FinanceBillQuery) *ent.FinanceBillQuery {
	return query.WithBatch().WithLines(func(lineQuery *ent.FinanceBillLineQuery) {
		lineQuery.WithOrder().Order(financebilllineent.ByCreatedAt(), financebilllineent.ByID())
	})
}

func (r *financeBillRepo) LoadBillableFees(ctx context.Context, organizationID uuid.UUID, feeIDs []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	return r.LoadBillableFeesScoped(ctx, []uuid.UUID{organizationID}, feeIDs)
}

func (r *financeBillRepo) LoadBillableFeesScoped(ctx context.Context, organizationIDs []uuid.UUID, feeIDs []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.OrderFee.Query().
		Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDIn(organizationIDs...))).
		WithSettlementParty().WithOrder().Order(orderfeeent.ByID())
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForUpdate()
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.FinanceBillableFee, 0, len(items))
	for _, item := range items {
		fee, convertErr := orderFeeToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		businessOrder, edgeErr := item.Edges.OrderOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		party, _ := item.Edges.SettlementPartyOrErr()
		isCasual := false
		if party != nil {
			isCasual = party.IsCasual
		}
		result = append(result, &biz.FinanceBillableFee{Fee: fee, OrganizationID: businessOrder.OrganizationID, OrderNo: businessOrder.OrderNo, BusinessType: string(businessOrder.BusinessType), SettlementPartyIsCasual: isCasual})
	}
	return result, nil
}

func (r *financeBillRepo) ValidateBillCurrencies(ctx context.Context, currencies []string) error {
	if len(currencies) == 0 {
		return biz.ErrExchangeRateCurrencyInvalid
	}
	unique := make(map[string]struct{}, len(currencies))
	for _, currency := range currencies {
		unique[currency] = struct{}{}
	}
	codes := make([]string, 0, len(unique))
	for currency := range unique {
		codes = append(codes, currency)
	}
	sort.Strings(codes)
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	query := client.Currency.Query().Where(currencyent.CodeIn(codes...), currencyent.EnabledEQ(true)).Order(currencyent.ByCode())
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	items, err := query.All(ctx)
	if err != nil {
		return err
	}
	if len(items) != len(codes) {
		return biz.ErrExchangeRateCurrencyInvalid
	}
	return nil
}

func (r *financeBillRepo) HydrateBillSettlementAccounts(ctx context.Context, bills []*biz.FinanceBill) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	accountIDs := make([]uuid.UUID, 0, len(bills))
	seen := make(map[uuid.UUID]struct{}, len(bills))
	for _, bill := range bills {
		if bill == nil || bill.SettlementAccountID == uuid.Nil || bill.OrganizationID == uuid.Nil || bill.SettlementPartyID == uuid.Nil || len(bill.Currency) != 3 {
			return biz.ErrFinanceBillSettlementAccountInvalid
		}
		if _, exists := seen[bill.SettlementAccountID]; !exists {
			seen[bill.SettlementAccountID] = struct{}{}
			accountIDs = append(accountIDs, bill.SettlementAccountID)
		}
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i].String() < accountIDs[j].String() })
	query := client.PartnerAccount.Query().Where(partneraccountent.IDIn(accountIDs...)).WithPartner().Order(partneraccountent.ByID())
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	accounts, err := query.All(ctx)
	if err != nil {
		return mapEntError(err, nil, biz.ErrFinanceBillSettlementAccountInvalid)
	}
	accountsByID := make(map[uuid.UUID]*ent.PartnerAccount, len(accounts))
	for _, account := range accounts {
		accountsByID[account.ID] = account
	}
	for _, bill := range bills {
		account := accountsByID[bill.SettlementAccountID]
		if account == nil || account.PartnerID != bill.SettlementPartyID || account.Currency != bill.Currency || !account.Enabled {
			return biz.ErrFinanceBillSettlementAccountInvalid
		}
		partner, edgeErr := account.Edges.PartnerOrErr()
		if edgeErr != nil || partner.OrganizationID != bill.OrganizationID {
			return biz.ErrFinanceBillSettlementAccountInvalid
		}
		expectedUsage := partneraccountent.UsageRECEIVABLE
		if bill.Direction == biz.OrderFeePayable {
			expectedUsage = partneraccountent.UsagePAYABLE
		}
		if account.Usage != expectedUsage && account.Usage != partneraccountent.UsageBOTH {
			return biz.ErrFinanceBillSettlementAccountInvalid
		}
		bill.SettlementAccountName = account.Name
		bill.SettlementAccountHolder = account.AccountHolder
		bill.SettlementBankName = account.BankName
		bill.SettlementBankAccount = account.AccountNo
		bill.SettlementAccountCurrency = account.Currency
		bill.SettlementSwiftCode = account.SwiftCode
	}
	return nil
}

func (r *financeBillRepo) ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter biz.FinanceBillCreationCandidateFilter) (*biz.FinanceBillCreationCandidateResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.OrderFee{orderfeeent.StatusEQ(orderfeeent.StatusUNBILLED), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID)), orderfeeent.Not(orderfeeent.HasFinanceBillLinesWith(financebilllineent.ActiveEQ(true)))}
	if filter.Keyword != "" {
		predicates = append(predicates, orderfeeent.Or(orderfeeent.FeeCodeContainsFold(filter.Keyword), orderfeeent.FeeNameContainsFold(filter.Keyword), orderfeeent.HasOrderWith(orderent.OrderNoContainsFold(filter.Keyword)), orderfeeent.HasSettlementPartyWith(partnerent.Or(partnerent.CodeContainsFold(filter.Keyword), partnerent.LegalNameContainsFold(filter.Keyword), partnerent.SearchKeywordsContainsFold(filter.Keyword), partnerent.HasAliasesWith(partneraliasent.Or(partneraliasent.AliasNameContainsFold(filter.Keyword), partneraliasent.SearchKeywordsContainsFold(filter.Keyword)))))))
	}
	if filter.Direction != "" {
		predicates = append(predicates, orderfeeent.DirectionEQ(orderfeeent.Direction(filter.Direction)))
	}
	query := client.OrderFee.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.WithSettlementParty().WithOrder().Order(orderfeeent.ByExpenseDate(entsql.OrderDesc()), orderfeeent.ByID()).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceBillCreationCandidateResult{Items: make([]*biz.FinanceBillableFee, 0, len(items)), Total: int64(total)}
	for _, item := range items {
		fee, e := orderFeeToBiz(item)
		if e != nil {
			return nil, e
		}
		order, e := item.Edges.OrderOrErr()
		if e != nil {
			return nil, e
		}
		party, _ := item.Edges.SettlementPartyOrErr()
		isCasual := false
		if party != nil {
			isCasual = party.IsCasual
		}
		result.Items = append(result.Items, &biz.FinanceBillableFee{Fee: fee, OrganizationID: order.OrganizationID, OrderNo: order.OrderNo, BusinessType: string(order.BusinessType), SettlementPartyIsCasual: isCasual})
	}
	return result, nil
}

var _ biz.FinanceBillRepo = (*financeBillRepo)(nil)
