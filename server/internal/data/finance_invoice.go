package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
	financeinvoicelineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceline"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	profileent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerinvoiceprofile"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type financeInvoiceRepo struct{ data *Data }

type financeInvoiceSummaryRow struct {
	Direction    string `json:"direction"`
	BaseCurrency string `json:"base_currency"`
	BaseAmount   string `json:"base_amount"`
}

func NewFinanceInvoiceRepo(data *Data) biz.FinanceInvoiceRepo { return &financeInvoiceRepo{data: data} }

func (r *financeInvoiceRepo) List(ctx context.Context, org uuid.UUID, filter biz.FinanceInvoiceFilter) (*biz.FinanceInvoiceListResult, error) {
	return r.ListScoped(ctx, []uuid.UUID{org}, filter)
}

func (r *financeInvoiceRepo) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, filter biz.FinanceInvoiceFilter) (*biz.FinanceInvoiceListResult, error) {
	var result *biz.FinanceInvoiceListResult
	err := r.data.withReadSnapshot(ctx, func(readCtx context.Context) error {
		var err error
		result, err = r.listScopedSnapshot(readCtx, organizationIDs, filter)
		return err
	})
	return result, err
}

func (r *financeInvoiceRepo) listScopedSnapshot(ctx context.Context, organizationIDs []uuid.UUID, filter biz.FinanceInvoiceFilter) (*biz.FinanceInvoiceListResult, error) {
	p := []predicate.FinanceInvoice{financeinvoiceent.OrganizationIDIn(organizationIDs...)}
	if filter.BillID != nil {
		p = append(p, financeinvoiceent.HasBillLinksWith(financeinvoicebillent.BillIDEQ(*filter.BillID)))
	}
	if filter.Keyword != "" {
		p = append(p, financeinvoiceent.Or(financeinvoiceent.RecordNoContainsFold(filter.Keyword), financeinvoiceent.SettlementPartyNameContainsFold(filter.Keyword), financeinvoiceent.TaxInvoiceNoContainsFold(filter.Keyword), financeinvoiceent.HasBillLinksWith(financeinvoicebillent.BillNoContainsFold(filter.Keyword))))
	}
	if filter.Direction != "" {
		p = append(p, financeinvoiceent.DirectionEQ(financeinvoiceent.Direction(filter.Direction)))
	}
	if filter.Status != "" {
		p = append(p, financeinvoiceent.StatusEQ(financeinvoiceent.Status(filter.Status)))
	}
	client, clientErr := r.data.client(ctx)
	if clientErr != nil {
		return nil, clientErr
	}
	q := client.FinanceInvoice.Query().WithOrganization().Where(p...).WithBillLinks(func(lq *ent.FinanceInvoiceBillQuery) {
		lq.Order(financeinvoicebillent.ByCreatedAt(), financeinvoicebillent.ByID())
		// 账单内登记列表需要逐费用追溯；普通台账只加载用于展示与导出的账单关联。
		if filter.BillID != nil {
			lq.WithFeeAllocations()
		}
	})
	matchedSummary, matchedErr := loadFinanceLedgerMatchedSummary(ctx, q.Clone().GroupBy(financeinvoiceent.FieldDirection, financeinvoiceent.FieldCurrency, financeinvoiceent.FieldBaseCurrency).Aggregate(financeLedgerMatchedAggregates(financeinvoiceent.FieldTotalAmount, financeinvoiceent.FieldBaseCurrencyAmount)...).Scan)
	if matchedErr != nil {
		return nil, matchedErr
	}

	total := int(matchedSummary.Count)
	summaryPredicates := append([]predicate.FinanceInvoice{}, p...)
	summaryPredicates = append(summaryPredicates, financeinvoiceent.StatusEQ(financeinvoiceent.StatusISSUED))
	summaryQuery := client.FinanceInvoice.Query().WithOrganization().Where(summaryPredicates...)
	summaryRows := make([]financeInvoiceSummaryRow, 0)
	if err := summaryQuery.Clone().Where(financeinvoiceent.BaseCurrencyAmountNotNil()).
		GroupBy(financeinvoiceent.FieldDirection, financeinvoiceent.FieldBaseCurrency).
		Aggregate(ent.As(ent.Sum(financeinvoiceent.FieldBaseCurrencyAmount), "base_amount")).
		Scan(ctx, &summaryRows); err != nil {
		return nil, err
	}
	summary := biz.FinanceInvoiceSummary{}
	amountsByBaseCurrency := make(map[string]*biz.FinanceBaseCurrencyAmount, len(summaryRows))
	for _, row := range summaryRows {
		value, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, row.BaseCurrency)
		if row.Direction == string(financeinvoiceent.DirectionRECEIVABLE) {
			bucket.ReceivableBaseAmount = bucket.ReceivableBaseAmount.Add(value)
		} else {
			bucket.PayableBaseAmount = bucket.PayableBaseAmount.Add(value)
		}
	}
	summary.AmountsByBaseCurrency = financeBaseCurrencyAmountItems(amountsByBaseCurrency)
	issuedCount, err := summaryQuery.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	summary.IssuedCount = int64(issuedCount)
	items, err := q.Order(financeinvoiceent.ByCreatedAt(entsql.OrderDesc()), financeinvoiceent.ByID(entsql.OrderDesc())).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceInvoiceListResult{MatchedSummary: matchedSummary, Items: make([]*biz.FinanceInvoice, 0, len(items)), Total: int64(total), Summary: summary}
	for _, item := range items {
		converted, e := financeInvoiceToBiz(item)
		if e != nil {
			return nil, e
		}
		result.Items = append(result.Items, converted)
	}
	return result, nil
}

func (r *financeInvoiceRepo) queryWithLinks(q *ent.FinanceInvoiceQuery) *ent.FinanceInvoiceQuery {
	return q.WithBillLinks(func(lq *ent.FinanceInvoiceBillQuery) {
		lq.WithFeeAllocations().Order(financeinvoicebillent.ByCreatedAt(), financeinvoicebillent.ByID())
	}).WithLines(func(lq *ent.FinanceInvoiceLineQuery) { lq.Order(financeinvoicelineent.ByLineNo()) }).WithRevisions()
}
func (r *financeInvoiceRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceInvoice, error) {
	return r.GetScoped(ctx, []uuid.UUID{org}, id)
}

func (r *financeInvoiceRepo) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceInvoice, error) {
	var result *biz.FinanceInvoice
	// 发票头、当前行、分配及修订历史经多次 SELECT 读取，必须沿同一版本快照。
	err := r.data.withReadSnapshot(ctx, func(readCtx context.Context) error {
		client, err := r.data.client(readCtx)
		if err != nil {
			return err
		}
		item, err := r.queryWithLinks(client.FinanceInvoice.Query().WithOrganization()).Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDIn(organizationIDs...)).Only(readCtx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceInvoiceNotFound, nil)
		}
		result, err = financeInvoiceToBiz(item)
		if err != nil {
			return err
		}
		return r.enrichInvoiceRevisionNames(readCtx, client, result)
	})
	return result, err
}
func (r *financeInvoiceRepo) GetByIdempotencyKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceInvoice, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := r.queryWithLinks(client.FinanceInvoice.Query().WithOrganization()).Where(financeinvoiceent.OrganizationIDEQ(org), financeinvoiceent.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return financeInvoiceToBiz(item)
}
func (r *financeInvoiceRepo) LoadBills(ctx context.Context, org uuid.UUID, ids []uuid.UUID) ([]*biz.FinanceBill, error) {
	return r.LoadBillsScoped(ctx, []uuid.UUID{org}, ids)
}
func (r *financeInvoiceRepo) LoadBillsScoped(ctx context.Context, organizationIDs []uuid.UUID, ids []uuid.UUID) ([]*biz.FinanceBill, error) {
	client, clientErr := r.data.client(ctx)
	if clientErr != nil {
		return nil, clientErr
	}
	items, err := client.FinanceBill.Query().Where(financebillent.IDIn(ids...), financebillent.OrganizationIDIn(organizationIDs...)).WithLines(func(q *ent.FinanceBillLineQuery) { q.WithOrderFee() }).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.FinanceBill, 0, len(items))
	for _, x := range items {
		b, e := invoiceableFinanceBill(x)
		if e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *financeInvoiceRepo) ListCreationBills(ctx context.Context, organizationID uuid.UUID, filter biz.FinanceInvoiceCreationBillFilter) (*biz.FinanceInvoiceCreationBillListResult, error) {
	var result *biz.FinanceInvoiceCreationBillListResult
	err := r.data.withReadSnapshot(ctx, func(readCtx context.Context) error {
		var err error
		result, err = r.listCreationBillsSnapshot(readCtx, organizationID, filter)
		return err
	})
	return result, err
}

func (r *financeInvoiceRepo) listCreationBillsSnapshot(ctx context.Context, organizationID uuid.UUID, filter biz.FinanceInvoiceCreationBillFilter) (*biz.FinanceInvoiceCreationBillListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceBill{
		financebillent.OrganizationIDEQ(organizationID),
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		financebillent.PurposeEQ(financebillent.PurposeNORMAL),
		financebillent.Not(financebillent.HasInvoiceExemptionsWith(financeinvoiceexemption.RevokedAtIsNil())),
		financebillent.HasLinesWith(financebilllineent.ActiveEQ(true), financebilllineent.HasOrderFeeWith(orderfeeent.EconomicEffectEQ(orderfeeent.EconomicEffectINCREASE), orderfeeent.SupersededByRequestIDIsNil())),
		financeInvoiceBillHasRemainingPredicate(),
	}
	if filter.Keyword != "" {
		predicates = append(predicates, financebillent.Or(
			financebillent.BillNoContainsFold(filter.Keyword),
			financebillent.SettlementPartyNameContainsFold(filter.Keyword),
			financebillent.HasSettlementPartyWith(financePartyKeywordPredicate(filter.Keyword)),
			financebillent.HasLinesWith(financebilllineent.OrderNoContainsFold(filter.Keyword)),
		))
	}
	if filter.BillID != nil {
		predicates = append(predicates, financebillent.IDEQ(*filter.BillID))
	}
	if filter.Direction != "" {
		predicates = append(predicates, financebillent.DirectionEQ(financebillent.Direction(filter.Direction)))
	}
	if filter.SettlementPartyID != nil {
		predicates = append(predicates, financebillent.SettlementPartyIDEQ(*filter.SettlementPartyID))
	}
	if filter.Currency != "" {
		predicates = append(predicates, financebillent.CurrencyEQ(filter.Currency))
	}
	query := client.FinanceBill.Query().Where(predicates...)
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.WithOrganization().WithLines(func(q *ent.FinanceBillLineQuery) { q.WithOrderFee() }).Order(
		financebillent.ByBillDate(entsql.OrderDesc()),
		financebillent.ByCreatedAt(entsql.OrderDesc()),
		financebillent.ByID(entsql.OrderDesc()),
	).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceInvoiceCreationBillListResult{Items: make([]*biz.FinanceBill, 0, len(items)), Total: int64(total)}
	for _, item := range items {
		converted, convertErr := invoiceableFinanceBill(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
	}
	if err := enrichInvoiceableBalances(ctx, client, result.Items, nil); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *financeInvoiceRepo) ListProfilesForBill(ctx context.Context, organizationIDs []uuid.UUID, billID uuid.UUID) (*biz.FinanceInvoiceProfilesForBill, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	bill, err := client.FinanceBill.Query().Where(
		financebillent.IDEQ(billID),
		financebillent.OrganizationIDIn(organizationIDs...),
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		financebillent.PurposeEQ(financebillent.PurposeNORMAL),
		financebillent.Not(financebillent.HasInvoiceExemptionsWith(financeinvoiceexemption.RevokedAtIsNil())),
		financebillent.HasLinesWith(financebilllineent.ActiveEQ(true), financebilllineent.HasOrderFeeWith(orderfeeent.EconomicEffectEQ(orderfeeent.EconomicEffectINCREASE), orderfeeent.SupersededByRequestIDIsNil())),
		financeInvoiceBillHasRemainingPredicate(),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceInvoiceBillInvalid, nil)
	}
	return r.ListProfilesForParty(ctx, []uuid.UUID{bill.OrganizationID}, bill.SettlementPartyID)
}
func (r *financeInvoiceRepo) ListProfilesForParty(ctx context.Context, organizationIDs []uuid.UUID, partyID uuid.UUID) (*biz.FinanceInvoiceProfilesForBill, error) {
	if len(organizationIDs) != 1 || organizationIDs[0] == uuid.Nil || partyID == uuid.Nil {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if exists, err := client.Partner.Query().Where(partnerent.IDEQ(partyID), partnerent.OrganizationIDEQ(organizationIDs[0])).Exist(ctx); err != nil {
		return nil, err
	} else if !exists {
		return nil, biz.ErrFinanceInvoiceBillInvalid
	}
	items, err := client.PartnerInvoiceProfile.Query().Where(
		profileent.OrganizationIDEQ(organizationIDs[0]),
		profileent.PartnerIDEQ(partyID),
		profileent.EnabledEQ(true),
	).Order(profileent.ByIsDefault(entsql.OrderDesc()), profileent.ByCreatedAt(), profileent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceInvoiceProfilesForBill{
		OrganizationID:    organizationIDs[0],
		SettlementPartyID: partyID,
		Items:             make([]*biz.PartnerInvoiceProfile, 0, len(items)),
	}
	for _, item := range items {
		result.Items = append(result.Items, partnerInvoiceProfileToBiz(item))
	}
	return result, nil
}

func (r *financeInvoiceRepo) LoadInvoiceProfile(ctx context.Context, org, partnerID, profileID uuid.UUID) (*biz.PartnerInvoiceProfile, error) {
	client, clientErr := r.data.client(ctx)
	if clientErr != nil {
		return nil, clientErr
	}
	query := client.PartnerInvoiceProfile.Query().Where(profileent.IDEQ(profileID), profileent.OrganizationIDEQ(org), profileent.PartnerIDEQ(partnerID), profileent.EnabledEQ(true))
	if _, transactional := transactionFromContext(ctx); transactional {
		query.ForShare()
	}
	item, err := query.Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrPartnerInvoiceProfileNotFound, nil)
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func financeInvoiceToBiz(x *ent.FinanceInvoice) (*biz.FinanceInvoice, error) {
	total, e := decimalOf(x.TotalAmount)
	if e != nil {
		return nil, e
	}
	tax, e := decimalOf(x.TaxAmount)
	if e != nil {
		return nil, e
	}
	net, e := decimalOf(x.NetAmount)
	if e != nil {
		return nil, e
	}
	var exchangeRate, baseCurrencyAmount *decimal.Decimal
	if x.ExchangeRate != nil {
		value, parseErr := decimalOf(*x.ExchangeRate)
		if parseErr != nil {
			return nil, parseErr
		}
		exchangeRate = &value
	}
	if x.BaseCurrencyAmount != nil {
		value, parseErr := decimalOf(*x.BaseCurrencyAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		baseCurrencyAmount = &value
	}
	var exchangeRateSource *string
	if x.ExchangeRateSource != nil {
		value := string(*x.ExchangeRateSource)
		exchangeRateSource = &value
	}
	out := &biz.FinanceInvoice{ID: x.ID, OrganizationID: x.OrganizationID, InvoiceProfileID: x.InvoiceProfileID, RecordNo: x.RecordNo, IdempotencyKey: x.IdempotencyKey, Direction: biz.OrderFeeDirection(x.Direction), Status: biz.FinanceInvoiceStatus(x.Status), InvoiceType: biz.FinanceInvoiceType(x.InvoiceType), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, InvoiceTitle: financeStringValue(x.InvoiceTitle), TaxpayerIdentificationNo: financeStringValue(x.TaxpayerIdentificationNo), RegisteredAddress: financeStringValue(x.RegisteredAddress), RegisteredPhone: financeStringValue(x.RegisteredPhone), BankName: financeStringValue(x.BankName), BankAccount: financeStringValue(x.BankAccount), Currency: x.Currency, BaseCurrency: x.BaseCurrency, ExchangeRate: exchangeRate, ExchangeRateSource: exchangeRateSource, ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingID: x.ExchangeRateSettingID, BaseCurrencyAmount: baseCurrencyAmount, TotalAmount: total, NetAmount: net, TaxAmount: tax, BillCount: x.BillCount, TaxInvoiceNo: x.TaxInvoiceNo, InvoiceDate: x.InvoiceDate, Note: x.Note, Version: x.Version, RevisionNumber: x.RevisionNumber, IssuedAt: x.IssuedAt, IssuedBy: x.IssuedBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy, CancellationReason: x.CancellationReason, RedInvoiceNo: x.RedInvoiceNo, RedInvoiceDate: x.RedInvoiceDate, RedFlushedAt: x.RedFlushedAt, RedFlushedBy: x.RedFlushedBy, RedFlushReason: x.RedFlushReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Links: make([]*biz.FinanceInvoiceBill, 0, len(x.Edges.BillLinks)), Lines: make([]*biz.FinanceInvoiceLine, 0, len(x.Edges.Lines))}
	if organization, edgeErr := x.Edges.OrganizationOrErr(); edgeErr == nil {
		out.OrganizationName = organization.Name
	}
	for _, l := range x.Edges.BillLinks {
		amount, e := decimalOf(l.Amount)
		if e != nil {
			return nil, e
		}
		tax, e := decimalOf(l.TaxAmount)
		if e != nil {
			return nil, e
		}
		link := &biz.FinanceInvoiceBill{SourceFeeIDs: l.SourceFeeIds, ID: l.ID, InvoiceID: l.InvoiceID, BillID: l.BillID, BillNo: l.BillNo, Amount: amount, TaxAmount: tax, Active: l.Active}
		for _, a := range l.Edges.FeeAllocations {
			amount, err := decimalOf(a.Amount)
			if err != nil {
				return nil, err
			}
			net, err := decimalOf(a.NetAmount)
			if err != nil {
				return nil, err
			}
			tax, err := decimalOf(a.TaxAmount)
			if err != nil {
				return nil, err
			}
			rate, err := invoiceOptionalDecimal(a.SourceTaxRate)
			if err != nil {
				return nil, err
			}
			link.Fees = append(link.Fees, &biz.FinanceInvoiceFeeAllocation{ID: a.ID, InvoiceID: a.InvoiceID, InvoiceBillID: a.InvoiceBillID, BillID: l.BillID, BillLineID: a.BillLineID, OrderFeeID: a.OrderFeeID, FeeCode: a.FeeCode, FeeName: a.FeeName, Currency: a.Currency, SourceTaxRate: rate, Amount: amount, NetAmount: net, TaxAmount: tax})
		}
		out.Links = append(out.Links, link)
	}
	for _, line := range x.Edges.Lines {
		if line.RevisionNumber != x.RevisionNumber {
			continue
		}
		taxRate, parseErr := invoiceOptionalDecimal(line.TaxRate)
		if parseErr != nil {
			return nil, parseErr
		}
		netAmount, parseErr := decimalOf(line.NetAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		taxAmount, parseErr := decimalOf(line.TaxAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		totalAmount, parseErr := decimalOf(line.TotalAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		out.Lines = append(out.Lines, &biz.FinanceInvoiceLine{ID: line.ID, InvoiceID: line.InvoiceID, LineNo: line.LineNo, ItemCode: line.ItemCode, ItemName: line.ItemName, TaxRate: taxRate, NetAmount: netAmount, TaxAmount: taxAmount, TotalAmount: totalAmount, Currency: line.Currency, SourceLineCount: line.SourceLineCount})
	}
	for _, revision := range x.Edges.Revisions {
		out.Revisions = append(out.Revisions, &biz.FinanceInvoiceRevision{ID: revision.ID, InvoiceID: revision.InvoiceID, ActorID: revision.ActorID, Reason: revision.Reason, RevisionNumber: revision.RevisionNumber, Before: revision.BeforeSnapshot, After: revision.AfterSnapshot, CreatedAt: revision.CreatedAt})
	}
	return out, nil
}

func financeStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ biz.FinanceInvoiceRepo = (*financeInvoiceRepo)(nil)

func invoiceOptionalDecimal(value *string) (*decimal.Decimal, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := decimalOf(*value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
