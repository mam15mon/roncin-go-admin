package data

import (
	"context"
	"sort"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financeinvoicelineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceline"
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
	p := []predicate.FinanceInvoice{financeinvoiceent.OrganizationIDIn(organizationIDs...)}
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
	q := client.FinanceInvoice.Query().WithOrganization().Where(p...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
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
	result := &biz.FinanceInvoiceListResult{Items: make([]*biz.FinanceInvoice, 0, len(items)), Total: int64(total), Summary: summary}
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
		lq.Order(financeinvoicebillent.ByCreatedAt(), financeinvoicebillent.ByID())
	}).WithLines(func(lq *ent.FinanceInvoiceLineQuery) { lq.Order(financeinvoicelineent.ByLineNo()) })
}
func (r *financeInvoiceRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceInvoice, error) {
	return r.GetScoped(ctx, []uuid.UUID{org}, id)
}

func (r *financeInvoiceRepo) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceInvoice, error) {
	client, clientErr := r.data.client(ctx)
	if clientErr != nil {
		return nil, clientErr
	}
	item, err := r.queryWithLinks(client.FinanceInvoice.Query().WithOrganization()).Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDIn(organizationIDs...)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceInvoiceNotFound, nil)
	}
	return financeInvoiceToBiz(item)
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
	items, err := client.FinanceBill.Query().Where(financebillent.IDIn(ids...), financebillent.OrganizationIDIn(organizationIDs...)).WithLines().All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.FinanceBill, 0, len(items))
	for _, x := range items {
		b, e := financeBillToBiz(x)
		if e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *financeInvoiceRepo) ListCreationBills(ctx context.Context, organizationID uuid.UUID, filter biz.FinanceInvoiceCreationBillFilter) (*biz.FinanceInvoiceCreationBillListResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	predicates := []predicate.FinanceBill{
		financebillent.OrganizationIDEQ(organizationID),
		financebillent.StatusEQ(financebillent.StatusCONFIRMED),
		financebillent.Not(financebillent.HasInvoiceLinksWith(financeinvoicebillent.ActiveEQ(true))),
	}
	if filter.Keyword != "" {
		predicates = append(predicates, financebillent.Or(
			financebillent.BillNoContainsFold(filter.Keyword),
			financebillent.SettlementPartyNameContainsFold(filter.Keyword),
		))
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
	items, err := query.WithOrganization().WithLines().Order(
		financebillent.ByBillDate(entsql.OrderDesc()),
		financebillent.ByCreatedAt(entsql.OrderDesc()),
		financebillent.ByID(entsql.OrderDesc()),
	).Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceInvoiceCreationBillListResult{Items: make([]*biz.FinanceBill, 0, len(items)), Total: int64(total)}
	for _, item := range items {
		converted, convertErr := financeBillToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Items = append(result.Items, converted)
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
		financebillent.Not(financebillent.HasInvoiceLinksWith(financeinvoicebillent.ActiveEQ(true))),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFinanceInvoiceBillInvalid, nil)
	}
	items, err := client.PartnerInvoiceProfile.Query().Where(
		profileent.OrganizationIDEQ(bill.OrganizationID),
		profileent.PartnerIDEQ(bill.SettlementPartyID),
		profileent.EnabledEQ(true),
	).Order(profileent.ByIsDefault(entsql.OrderDesc()), profileent.ByCreatedAt(), profileent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceInvoiceProfilesForBill{
		OrganizationID:    bill.OrganizationID,
		SettlementPartyID: bill.SettlementPartyID,
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
	item, err := client.PartnerInvoiceProfile.Query().Where(profileent.IDEQ(profileID), profileent.OrganizationIDEQ(org), profileent.PartnerIDEQ(partnerID), profileent.EnabledEQ(true)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrPartnerInvoiceProfileNotFound, nil)
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func (r *financeInvoiceRepo) Create(ctx context.Context, invoice *biz.FinanceInvoice, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if invoice.InvoiceProfileID == nil {
			return biz.ErrFinanceInvoiceProfileRequired
		}
		profile, queryErr := tx.PartnerInvoiceProfile.Query().Where(profileent.IDEQ(*invoice.InvoiceProfileID), profileent.OrganizationIDEQ(invoice.OrganizationID), profileent.PartnerIDEQ(invoice.SettlementPartyID), profileent.EnabledEQ(true)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrFinanceInvoiceProfileRequired, nil)
		}
		if profile.InvoiceTitle != invoice.InvoiceTitle || profile.TaxpayerIdentificationNo != invoice.TaxpayerIdentificationNo || profile.RegisteredAddress != invoice.RegisteredAddress || profile.RegisteredPhone != invoice.RegisteredPhone || profile.BankName != invoice.BankName || profile.BankAccount != invoice.BankAccount {
			return biz.ErrFinanceInvoiceProfileRequired
		}
		ids := make([]uuid.UUID, 0, len(invoice.Links))
		for _, link := range invoice.Links {
			ids = append(ids, link.BillID)
		}
		bills, queryErr := tx.FinanceBill.Query().Where(financebillent.IDIn(ids...), financebillent.OrganizationIDEQ(invoice.OrganizationID)).ForUpdate().All(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(bills) != len(ids) {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		for _, bill := range bills {
			if bill.Status != financebillent.StatusCONFIRMED || string(bill.Direction) != string(invoice.Direction) || bill.SettlementPartyID != invoice.SettlementPartyID || bill.Currency != invoice.Currency {
				return biz.ErrFinanceInvoiceBillInvalid
			}
		}
		active, queryErr := tx.FinanceInvoiceBill.Query().Where(financeinvoicebillent.BillIDIn(ids...), financeinvoicebillent.ActiveEQ(true)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if active {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		if _, createErr := tx.FinanceInvoice.Create().SetID(invoice.ID).SetOrganizationID(invoice.OrganizationID).SetRecordNo(invoice.RecordNo).SetIdempotencyKey(invoice.IdempotencyKey).SetDirection(financeinvoiceent.Direction(invoice.Direction)).SetStatus(financeinvoiceent.StatusDRAFT).SetInvoiceType(financeinvoiceent.InvoiceType(invoice.InvoiceType)).SetNillableInvoiceProfileID(invoice.InvoiceProfileID).SetSettlementPartyID(invoice.SettlementPartyID).SetSettlementPartyName(invoice.SettlementPartyName).SetInvoiceTitle(invoice.InvoiceTitle).SetTaxpayerIdentificationNo(invoice.TaxpayerIdentificationNo).SetRegisteredAddress(invoice.RegisteredAddress).SetRegisteredPhone(invoice.RegisteredPhone).SetBankName(invoice.BankName).SetBankAccount(invoice.BankAccount).SetCurrency(invoice.Currency).SetBaseCurrency(invoice.BaseCurrency).SetTotalAmount(invoice.TotalAmount.StringFixed(8)).SetNetAmount(invoice.NetAmount.StringFixed(8)).SetTaxAmount(invoice.TaxAmount.StringFixed(8)).SetBillCount(invoice.BillCount).SetNillableNote(invoice.Note).SetVersion(1).Save(ctx); createErr != nil {
			return createErr
		}
		builders := make([]*ent.FinanceInvoiceBillCreate, 0, len(invoice.Links))
		for _, link := range invoice.Links {
			builders = append(builders, tx.FinanceInvoiceBill.Create().SetID(link.ID).SetInvoiceID(invoice.ID).SetBillID(link.BillID).SetBillNo(link.BillNo).SetAmount(link.Amount.StringFixed(8)).SetTaxAmount(link.TaxAmount.StringFixed(8)).SetActive(true))
		}
		if _, createErr := tx.FinanceInvoiceBill.CreateBulk(builders...).Save(ctx); createErr != nil {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		lineBuilders := make([]*ent.FinanceInvoiceLineCreate, 0, len(invoice.Lines))
		for _, line := range invoice.Lines {
			lineBuilders = append(lineBuilders, tx.FinanceInvoiceLine.Create().SetID(line.ID).SetInvoiceID(invoice.ID).SetLineNo(line.LineNo).SetItemCode(line.ItemCode).SetItemName(line.ItemName).SetTaxRate(line.TaxRate.StringFixed(4)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetTotalAmount(line.TotalAmount.StringFixed(8)).SetCurrency(line.Currency).SetSourceLineCount(line.SourceLineCount))
		}
		if _, createErr := tx.FinanceInvoiceLine.CreateBulk(lineBuilders...).Save(ctx); createErr != nil {
			return createErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, invoice.OrganizationID, invoice.ID)
}

func (r *financeInvoiceRepo) Issue(ctx context.Context, org, id, actor uuid.UUID, version uint64, issue biz.FinanceInvoiceIssueInput, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.FinanceInvoice.Query().Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrFinanceInvoiceNotFound, nil)
		}
		if item.Version != version {
			return biz.ErrFinanceInvoiceVersionConflict
		}
		if item.Status != financeinvoiceent.StatusDRAFT {
			return biz.ErrFinanceInvoiceInvalidTransition
		}
		duplicate, queryErr := tx.FinanceInvoice.Query().Where(financeinvoiceent.OrganizationIDEQ(org), financeinvoiceent.IDNEQ(id), financeinvoiceent.TaxInvoiceNoEQ(issue.TaxInvoiceNo)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if duplicate {
			return biz.ErrFinanceInvoiceTaxNoExists
		}
		now := time.Now()
		if _, updateErr := tx.FinanceInvoice.UpdateOneID(id).SetStatus(financeinvoiceent.StatusISSUED).SetTaxInvoiceNo(issue.TaxInvoiceNo).SetInvoiceDate(issue.InvoiceDate).SetExchangeRate(issue.ExchangeRate.StringFixed(8)).SetExchangeRateSource(financeinvoiceent.ExchangeRateSource(issue.ExchangeRateSource)).SetExchangeRateDate(issue.ExchangeRateDate).SetNillableExchangeRateSettingID(issue.ExchangeRateSettingID).SetBaseCurrencyAmount(issue.BaseCurrencyAmount.StringFixed(8)).SetIssuedAt(now).SetIssuedBy(actor).SetVersion(item.Version + 1).Save(ctx); updateErr != nil {
			return mapEntConstraint(updateErr, "financeinvoice_org_tax_invoice_no", biz.ErrFinanceInvoiceTaxNoExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}

// lockAndReleaseInvoiceActiveBills 释放发票活动账单关联并推进账单版本，供 Cancel/RedFlush 共用。
// 发票状态判断不在此辅助内：调用方必须在 Invoice FOR UPDATE 并完成权威状态校验后调用。
// 锁序固定为 Invoice → 按 ID 排序的活动 FinanceInvoiceBill → 按 UUID 升序的 FinanceBill FOR UPDATE；
// 释放活动关联后每张受影响账单 version 恰好递增一次，任一步失败由外层事务整体回滚。
func lockAndReleaseInvoiceActiveBills(ctx context.Context, tx *ent.Tx, org, invoiceID uuid.UUID) error {
	links, queryErr := tx.FinanceInvoiceBill.Query().Where(financeinvoicebillent.InvoiceIDEQ(invoiceID), financeinvoicebillent.ActiveEQ(true)).Order(financeinvoicebillent.ByID()).ForUpdate().All(ctx)
	if queryErr != nil {
		return queryErr
	}
	if len(links) == 0 {
		return biz.ErrFinanceInvoiceInvalidTransition
	}
	billIDs := make([]uuid.UUID, 0, len(links))
	seen := make(map[uuid.UUID]struct{}, len(links))
	for _, link := range links {
		if _, ok := seen[link.BillID]; ok {
			continue
		}
		seen[link.BillID] = struct{}{}
		billIDs = append(billIDs, link.BillID)
	}
	sort.Slice(billIDs, func(i, j int) bool { return billIDs[i].String() < billIDs[j].String() })
	bills, queryErr := tx.FinanceBill.Query().Where(financebillent.IDIn(billIDs...)).Order(financebillent.ByID()).ForUpdate().All(ctx)
	if queryErr != nil {
		return queryErr
	}
	if len(bills) != len(billIDs) {
		return biz.ErrFinanceInvoiceBillInvalid
	}
	for _, bill := range bills {
		if bill.OrganizationID != org || bill.Status != financebillent.StatusCONFIRMED {
			return biz.ErrFinanceInvoiceBillInvalid
		}
	}
	released, updateErr := tx.FinanceInvoiceBill.Update().Where(financeinvoicebillent.InvoiceIDEQ(invoiceID), financeinvoicebillent.ActiveEQ(true)).SetActive(false).Save(ctx)
	if updateErr != nil {
		return updateErr
	}
	if released != len(links) {
		return biz.ErrFinanceInvoiceBillInvalid
	}
	// 释放关联后推进账单版本，使携带旧版本号的并发账单操作以 409 冲突失败。
	affected, updateErr := tx.FinanceBill.Update().Where(financebillent.IDIn(billIDs...)).AddVersion(1).Save(ctx)
	if updateErr != nil {
		return updateErr
	}
	if affected != len(billIDs) {
		return biz.ErrFinanceInvoiceBillInvalid
	}
	return nil
}

func (r *financeInvoiceRepo) Cancel(ctx context.Context, org, id, actor uuid.UUID, version uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.FinanceInvoice.Query().Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrFinanceInvoiceNotFound, nil)
		}
		if item.Version != version {
			return biz.ErrFinanceInvoiceVersionConflict
		}
		// CANCELLED 与 RED_FLUSHED 均为终态；只有 DRAFT（取消草稿）与 ISSUED（作废已开票发票）允许取消。
		if item.Status != financeinvoiceent.StatusDRAFT && item.Status != financeinvoiceent.StatusISSUED {
			return biz.ErrFinanceInvoiceInvalidTransition
		}
		if releaseErr := lockAndReleaseInvoiceActiveBills(ctx, tx, org, id); releaseErr != nil {
			return releaseErr
		}
		now := time.Now()
		if _, updateErr := tx.FinanceInvoice.UpdateOneID(id).SetStatus(financeinvoiceent.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason).SetVersion(item.Version + 1).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}

func (r *financeInvoiceRepo) RedFlush(ctx context.Context, org, id, actor uuid.UUID, version uint64, redInvoiceNo, redInvoiceDate, reason string, audit *biz.AuditEvent) (*biz.FinanceInvoice, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, queryErr := tx.FinanceInvoice.Query().Where(financeinvoiceent.IDEQ(id), financeinvoiceent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrFinanceInvoiceNotFound, nil)
		}
		if item.Version != version {
			return biz.ErrFinanceInvoiceVersionConflict
		}
		// 红冲只允许 ISSUED；DRAFT 未开票、CANCELLED 与 RED_FLUSHED 为终态。
		if item.Status != financeinvoiceent.StatusISSUED {
			return biz.ErrFinanceInvoiceInvalidTransition
		}
		duplicate, queryErr := tx.FinanceInvoice.Query().Where(financeinvoiceent.OrganizationIDEQ(org), financeinvoiceent.RedInvoiceNoEQ(redInvoiceNo)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if duplicate {
			return biz.ErrFinanceInvoiceRedNoExists
		}
		if releaseErr := lockAndReleaseInvoiceActiveBills(ctx, tx, org, id); releaseErr != nil {
			return releaseErr
		}
		now := time.Now()
		if _, updateErr := tx.FinanceInvoice.UpdateOneID(id).SetStatus(financeinvoiceent.StatusRED_FLUSHED).SetRedInvoiceNo(redInvoiceNo).SetRedInvoiceDate(redInvoiceDate).SetRedFlushedAt(now).SetRedFlushedBy(actor).SetRedFlushReason(reason).SetVersion(item.Version + 1).Save(ctx); updateErr != nil {
			return mapEntConstraint(updateErr, "financeinvoice_org_red_invoice_no", biz.ErrFinanceInvoiceRedNoExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
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
	out := &biz.FinanceInvoice{ID: x.ID, OrganizationID: x.OrganizationID, InvoiceProfileID: x.InvoiceProfileID, RecordNo: x.RecordNo, IdempotencyKey: x.IdempotencyKey, Direction: biz.OrderFeeDirection(x.Direction), Status: biz.FinanceInvoiceStatus(x.Status), InvoiceType: biz.FinanceInvoiceType(x.InvoiceType), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, InvoiceTitle: financeStringValue(x.InvoiceTitle), TaxpayerIdentificationNo: financeStringValue(x.TaxpayerIdentificationNo), RegisteredAddress: financeStringValue(x.RegisteredAddress), RegisteredPhone: financeStringValue(x.RegisteredPhone), BankName: financeStringValue(x.BankName), BankAccount: financeStringValue(x.BankAccount), Currency: x.Currency, BaseCurrency: x.BaseCurrency, ExchangeRate: exchangeRate, ExchangeRateSource: exchangeRateSource, ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingID: x.ExchangeRateSettingID, BaseCurrencyAmount: baseCurrencyAmount, TotalAmount: total, NetAmount: net, TaxAmount: tax, BillCount: x.BillCount, TaxInvoiceNo: x.TaxInvoiceNo, InvoiceDate: x.InvoiceDate, Note: x.Note, Version: x.Version, IssuedAt: x.IssuedAt, IssuedBy: x.IssuedBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy, CancellationReason: x.CancellationReason, RedInvoiceNo: x.RedInvoiceNo, RedInvoiceDate: x.RedInvoiceDate, RedFlushedAt: x.RedFlushedAt, RedFlushedBy: x.RedFlushedBy, RedFlushReason: x.RedFlushReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Links: make([]*biz.FinanceInvoiceBill, 0, len(x.Edges.BillLinks)), Lines: make([]*biz.FinanceInvoiceLine, 0, len(x.Edges.Lines))}
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
		out.Links = append(out.Links, &biz.FinanceInvoiceBill{ID: l.ID, InvoiceID: l.InvoiceID, BillID: l.BillID, BillNo: l.BillNo, Amount: amount, TaxAmount: tax, Active: l.Active})
	}
	for _, line := range x.Edges.Lines {
		taxRate, parseErr := decimalOf(line.TaxRate)
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
	return out, nil
}

func financeStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

var _ biz.FinanceInvoiceRepo = (*financeInvoiceRepo)(nil)
