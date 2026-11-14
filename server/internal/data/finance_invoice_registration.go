package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	invoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	commandent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicecommand"
	exemptionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoiceexemption"
	allocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicefeeallocation"
	nettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	verificationallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

func (r *financeInvoiceRepo) BeginRegistrationCommand(ctx context.Context, org, actor uuid.UUID, kind, key, fingerprint string) (*biz.FinanceInvoiceCommand, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	// 同键并发由唯一索引等待首事务结束，再锁定持久命令；失败事务不会留下半条命令。
	err = client.FinanceInvoiceCommand.Create().SetOrganizationID(org).SetActorID(actor).SetKind(commandent.Kind(kind)).SetIdempotencyKey(key).SetRequestFingerprint(fingerprint).
		OnConflict(entsql.ConflictColumns(commandent.FieldOrganizationID, commandent.FieldIdempotencyKey)).DoNothing().Exec(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	item, err := client.FinanceInvoiceCommand.Query().Where(commandent.OrganizationIDEQ(org), commandent.IdempotencyKeyEQ(key)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, err
	}
	if item.RequestFingerprint != fingerprint || string(item.Kind) != kind {
		return nil, biz.ErrFinanceInvoiceIdempotencyConflict
	}
	return &biz.FinanceInvoiceCommand{ID: item.ID, OrganizationID: org, Kind: kind, IdempotencyKey: key, RequestFingerprint: fingerprint, InvoiceIDs: item.InvoiceIds, Completed: item.Completed}, nil
}
func (r *financeInvoiceRepo) CompleteRegistrationCommand(ctx context.Context, id uuid.UUID, ids []uuid.UUID) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	return client.FinanceInvoiceCommand.UpdateOneID(id).SetInvoiceIds(ids).SetCompleted(true).Exec(ctx)
}

func activeInvoiceAllocationPredicate(exclude []uuid.UUID) predicate.FinanceInvoiceFeeAllocation {
	conditions := []predicate.FinanceInvoiceBill{linkent.ActiveEQ(true), linkent.HasInvoiceWith(invoiceent.StatusEQ(invoiceent.StatusISSUED))}
	if len(exclude) > 0 {
		conditions = append(conditions, linkent.InvoiceIDNotIn(exclude...))
	}
	return allocationent.HasInvoiceBillWith(conditions...)
}

// enrichInvoiceableBalances 只累计活动分配事实，不把部分登记当作已覆盖整条费用。
func enrichInvoiceableBalances(ctx context.Context, client *ent.Client, bills []*biz.FinanceBill, exclude []uuid.UUID) error {
	lineIDs := []uuid.UUID{}
	for _, b := range bills {
		for _, l := range b.Lines {
			if !l.Active {
				continue
			}
			lineIDs = append(lineIDs, l.ID)
		}
	}
	amounts := map[uuid.UUID]decimal.Decimal{}
	if len(lineIDs) > 0 {
		rows, err := client.FinanceInvoiceFeeAllocation.Query().Where(allocationent.BillLineIDIn(lineIDs...), activeInvoiceAllocationPredicate(exclude)).All(ctx)
		if err != nil {
			return err
		}
		for _, a := range rows {
			v, err := decimalOf(a.Amount)
			if err != nil {
				return err
			}
			amounts[a.BillLineID] = amounts[a.BillLineID].Add(v)
		}
	}
	for _, b := range bills {
		total, net, tax := decimal.Zero, decimal.Zero, decimal.Zero
		for _, l := range b.Lines {
			if !l.Active {
				zero := decimal.Zero
				l.InvoicedAmount = &zero
				l.InvoiceableAmount = &zero
				continue
			}
			used := amounts[l.ID]
			available := l.TotalAmount.Sub(used)
			if available.IsNegative() {
				return biz.ErrFinanceInvoiceAllocationExceeded
			}
			l.InvoicedAmount = &used
			l.InvoiceableAmount = &available
			total = total.Add(available)
			net = net.Add(l.NetAmount)
			tax = tax.Add(l.TaxAmount)
		}
		b.InvoiceableAmount = &total
		b.InvoiceableNetAmount = &net
		b.InvoiceableTaxAmount = &tax
	}
	return nil
}

func (r *financeInvoiceRepo) LockRegistrationBills(ctx context.Context, scope []uuid.UUID, versions []biz.FinanceInvoiceBillVersion, exclude []uuid.UUID, requireUnsettledFree bool) ([]*biz.FinanceBill, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(versions))
	expected := map[uuid.UUID]uint64{}
	for _, v := range versions {
		ids = append(ids, v.BillID)
		expected[v.BillID] = v.ExpectedVersion
	}
	if len(exclude) > 0 {
		oldLinks, err := client.FinanceInvoiceBill.Query().Where(linkent.InvoiceIDIn(exclude...), linkent.ActiveEQ(true), linkent.HasInvoiceWith(invoiceent.OrganizationIDIn(scope...))).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, link := range oldLinks {
			if _, ok := expected[link.BillID]; !ok {
				return nil, biz.ErrFinanceInvoiceBillInvalid
			}
		}
	}
	if err := lockExistingBillOrders(ctx, client, scope, ids); err != nil {
		return nil, err
	}
	items, err := client.FinanceBill.Query().Where(billent.IDIn(ids...), billent.OrganizationIDIn(scope...)).WithLines(func(q *ent.FinanceBillLineQuery) { q.WithOrderFee() }).Order(billent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, biz.ErrFinanceInvoiceBillInvalid
	}
	for _, b := range items {
		if b.Version != expected[b.ID] {
			return nil, biz.ErrFinanceBillVersionConflict
		}
		if b.Status != billent.StatusCONFIRMED || b.Purpose != billent.PurposeNORMAL {
			return nil, biz.ErrFinanceInvoiceBillInvalid
		}
	}
	if exempt, err := client.FinanceInvoiceExemption.Query().Where(exemptionent.BillIDIn(ids...), exemptionent.RevokedAtIsNil()).Exist(ctx); err != nil {
		return nil, err
	} else if exempt {
		return nil, biz.ErrFinanceInvoiceExemptionConflict
	}
	if requireUnsettledFree {
		if err := assertInvoiceBillsHaveNoSettlement(ctx, client, ids); err != nil {
			return nil, err
		}
	}
	if err := checkCorrectionInvoiceAncestors(ctx, client, items); err != nil {
		return nil, err
	}
	out := make([]*biz.FinanceBill, 0, len(items))
	for _, raw := range items {
		b, err := invoiceableFinanceBill(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := enrichInvoiceableBalances(ctx, client, out, exclude); err != nil {
		return nil, err
	}
	return out, nil
}

func assertInvoiceBillsHaveNoSettlement(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) error {
	exists, err := client.FinanceBill.Query().Where(billent.IDIn(billIDs...), billent.Or(
		billent.HasVerificationAllocationsWith(verificationallocent.ActiveEQ(true), verificationallocent.HasVerificationWith(verificationent.StatusEQ(verificationent.StatusACTIVE))),
		billent.HasNettingAllocationsWith(nettingallocent.ActiveEQ(true), nettingallocent.HasNettingWith(nettingent.StatusEQ(nettingent.StatusCONFIRMED))),
	)).Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return biz.ErrFinanceInvoiceSettlementDependency
	}
	return nil
}

func (r *financeInvoiceRepo) LoadRevisionInvoices(ctx context.Context, scope []uuid.UUID, entries []biz.FinanceInvoiceRevisionEntry) ([]*biz.FinanceInvoice, error) {
	ids := []uuid.UUID{}
	byID := map[uuid.UUID]uint64{}
	for _, e := range entries {
		if e.ID != uuid.Nil {
			ids = append(ids, e.ID)
			byID[e.ID] = e.ExpectedVersion
		}
	}
	if len(ids) == 0 {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := r.queryWithLinks(client.FinanceInvoice.Query()).Where(invoiceent.IDIn(ids...), invoiceent.OrganizationIDIn(scope...)).Order(invoiceent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, biz.ErrFinanceInvoiceNotFound
	}
	out := make([]*biz.FinanceInvoice, 0, len(items))
	for _, item := range items {
		if item.Status != invoiceent.StatusISSUED {
			return nil, biz.ErrFinanceInvoiceInvalidTransition
		}
		if item.Version != byID[item.ID] {
			return nil, biz.ErrFinanceInvoiceVersionConflict
		}
		v, err := financeInvoiceToBiz(item)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// 快照只保存可审阅业务字段，不保存整份请求/响应或幂等与版本等内部控制数据。
func invoiceRevisionSnapshot(invoice *biz.FinanceInvoice) json.RawMessage {
	if invoice == nil {
		return json.RawMessage(`null`)
	}
	type feeSnapshot struct {
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		Amount        string  `json:"amount"`
		TaxAmount     string  `json:"taxAmount"`
		NetAmount     string  `json:"netAmount"`
		SourceTaxRate *string `json:"sourceTaxRate,omitempty"`
	}
	type billSnapshot struct {
		BillNo    string        `json:"billNo"`
		Amount    string        `json:"amount"`
		TaxAmount string        `json:"taxAmount"`
		Fees      []feeSnapshot `json:"fees"`
	}
	snapshot := struct {
		Number       *string        `json:"number"`
		Date         *string        `json:"date"`
		InvoiceType  string         `json:"invoiceType"`
		Status       string         `json:"status"`
		InvoiceTitle string         `json:"invoiceTitle"`
		Currency     string         `json:"currency"`
		BaseCurrency string         `json:"baseCurrency"`
		ExchangeRate *string        `json:"exchangeRate"`
		RateSource   *string        `json:"exchangeRateSource"`
		RateDate     *string        `json:"exchangeRateDate"`
		BaseAmount   *string        `json:"baseCurrencyAmount"`
		TotalAmount  string         `json:"totalAmount"`
		NetAmount    string         `json:"netAmount"`
		TaxAmount    string         `json:"taxAmount"`
		Note         *string        `json:"note"`
		Bills        []billSnapshot `json:"bills"`
	}{Number: invoice.TaxInvoiceNo, Date: invoice.InvoiceDate, InvoiceType: string(invoice.InvoiceType), Status: string(invoice.Status), InvoiceTitle: invoice.InvoiceTitle, Currency: invoice.Currency, BaseCurrency: invoice.BaseCurrency, ExchangeRate: financeDecimalString(invoice.ExchangeRate, 8), RateSource: invoice.ExchangeRateSource, RateDate: invoice.ExchangeRateDate, BaseAmount: financeDecimalString(invoice.BaseCurrencyAmount, 8), TotalAmount: invoice.TotalAmount.StringFixed(8), NetAmount: invoice.NetAmount.StringFixed(8), TaxAmount: invoice.TaxAmount.StringFixed(8), Note: invoice.Note, Bills: []billSnapshot{}}
	for _, link := range invoice.Links {
		if !link.Active {
			continue
		}
		b := billSnapshot{BillNo: link.BillNo, Amount: link.Amount.StringFixed(8), TaxAmount: link.TaxAmount.StringFixed(8)}
		for _, f := range link.Fees {
			b.Fees = append(b.Fees, feeSnapshot{Code: f.FeeCode, Name: f.FeeName, Amount: f.Amount.StringFixed(8), TaxAmount: f.TaxAmount.StringFixed(8), NetAmount: f.NetAmount.StringFixed(8), SourceTaxRate: financeDecimalString(f.SourceTaxRate, 4)})
		}
		snapshot.Bills = append(snapshot.Bills, b)
	}
	value, _ := json.Marshal(snapshot)
	return value
}

func (r *financeInvoiceRepo) SaveRegistrations(ctx context.Context, org, actor uuid.UUID, invoices, old []*biz.FinanceInvoice, reason string, commandID uuid.UUID) ([]uuid.UUID, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	oldByID := map[uuid.UUID]*biz.FinanceInvoice{}
	billIDs := map[uuid.UUID]bool{}
	ids := make([]uuid.UUID, 0, len(invoices)+len(old))
	now := time.Now()
	if len(old) > 0 {
		oldIDs := []uuid.UUID{}
		for _, v := range old {
			oldIDs = append(oldIDs, v.ID)
			oldByID[v.ID] = v
			for _, l := range v.Links {
				if l.Active {
					billIDs[l.BillID] = true
				}
			}
		}
		locked, err := client.FinanceInvoice.Query().Where(invoiceent.IDIn(oldIDs...), invoiceent.OrganizationIDEQ(org)).Order(invoiceent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range locked {
			if item.Version != oldByID[item.ID].Version {
				return nil, biz.ErrFinanceInvoiceVersionConflict
			}
			if item.Status != invoiceent.StatusISSUED {
				return nil, biz.ErrFinanceInvoiceInvalidTransition
			}
		}
		if err := assertInvoiceBillsHaveNoSettlement(ctx, client, mapUUIDs(billIDs)); err != nil {
			return nil, err
		}

		if _, err := client.FinanceInvoiceBill.Update().Where(linkent.InvoiceIDIn(oldIDs...), linkent.ActiveEQ(true)).SetActive(false).Save(ctx); err != nil {
			return nil, err
		}
	}
	for index, inv := range invoices {
		inv.IdempotencyKey = commandID.String() + ":" + decimal.NewFromInt(int64(index)).String()
		for _, l := range inv.Links {
			l.InvoiceID = inv.ID
			billIDs[l.BillID] = true
			for _, a := range l.Fees {
				a.InvoiceID = inv.ID
			}
		}
		existing := oldByID[inv.ID]
		revision := uint64(1)
		if existing != nil {
			if existing.OrganizationID != inv.OrganizationID || existing.Direction != inv.Direction || existing.SettlementPartyID != inv.SettlementPartyID || existing.Currency != inv.Currency || existing.BaseCurrency != inv.BaseCurrency {
				return nil, biz.ErrFinanceInvoiceBillMismatch
			}
			revision = existing.RevisionNumber + 1
			inv.RecordNo = existing.RecordNo
			inv.IdempotencyKey = existing.IdempotencyKey
			inv.Version = existing.Version + 1
			update := client.FinanceInvoice.UpdateOneID(inv.ID).SetInvoiceType(invoiceent.InvoiceType(inv.InvoiceType)).
				SetTotalAmount(inv.TotalAmount.StringFixed(8)).SetNetAmount(inv.NetAmount.StringFixed(8)).SetTaxAmount(inv.TaxAmount.StringFixed(8)).SetBillCount(inv.BillCount).
				SetTaxInvoiceNo(*inv.TaxInvoiceNo).SetInvoiceDate(*inv.InvoiceDate).SetExchangeRate(inv.ExchangeRate.StringFixed(8)).SetExchangeRateSource(invoiceent.ExchangeRateSource(*inv.ExchangeRateSource)).SetExchangeRateDate(*inv.ExchangeRateDate).SetNillableExchangeRateSettingID(inv.ExchangeRateSettingID).SetBaseCurrencyAmount(inv.BaseCurrencyAmount.StringFixed(8)).SetVersion(inv.Version).SetRevisionNumber(revision)
			if inv.Note == nil {
				update.ClearNote()
			} else {
				update.SetNote(*inv.Note)
			}
			if err := update.Exec(ctx); err != nil {
				return nil, mapEntConstraint(err, "financeinvoice_org_tax_invoice_no", biz.ErrFinanceInvoiceTaxNoExists)
			}
		} else {
			_, err := client.FinanceInvoice.Create().SetID(inv.ID).SetOrganizationID(org).SetRecordNo(inv.RecordNo).SetIdempotencyKey(inv.IdempotencyKey).SetDirection(invoiceent.Direction(inv.Direction)).SetStatus(invoiceent.StatusISSUED).
				SetInvoiceType(invoiceent.InvoiceType(inv.InvoiceType)).SetNillableInvoiceProfileID(inv.InvoiceProfileID).SetSettlementPartyID(inv.SettlementPartyID).SetSettlementPartyName(inv.SettlementPartyName).
				SetInvoiceTitle(inv.InvoiceTitle).SetTaxpayerIdentificationNo(inv.TaxpayerIdentificationNo).SetRegisteredAddress(inv.RegisteredAddress).SetRegisteredPhone(inv.RegisteredPhone).SetBankName(inv.BankName).SetBankAccount(inv.BankAccount).
				SetCurrency(inv.Currency).SetBaseCurrency(inv.BaseCurrency).SetTotalAmount(inv.TotalAmount.StringFixed(8)).SetNetAmount(inv.NetAmount.StringFixed(8)).SetTaxAmount(inv.TaxAmount.StringFixed(8)).SetBillCount(inv.BillCount).
				SetTaxInvoiceNo(*inv.TaxInvoiceNo).SetInvoiceDate(*inv.InvoiceDate).SetExchangeRate(inv.ExchangeRate.StringFixed(8)).SetExchangeRateSource(invoiceent.ExchangeRateSource(*inv.ExchangeRateSource)).SetExchangeRateDate(*inv.ExchangeRateDate).SetNillableExchangeRateSettingID(inv.ExchangeRateSettingID).SetBaseCurrencyAmount(inv.BaseCurrencyAmount.StringFixed(8)).SetIssuedAt(now).SetIssuedBy(actor).SetNillableNote(inv.Note).SetVersion(1).SetRevisionNumber(1).Save(ctx)
			if err != nil {
				return nil, mapEntConstraint(err, "financeinvoice_org_tax_invoice_no", biz.ErrFinanceInvoiceTaxNoExists)
			}
		}
		inv.RevisionNumber = revision
		for _, link := range inv.Links {
			if _, err := client.FinanceInvoiceBill.Create().SetID(link.ID).SetInvoiceID(inv.ID).SetBillID(link.BillID).SetBillNo(link.BillNo).SetSourceFeeIds(link.SourceFeeIDs).SetAmount(link.Amount.StringFixed(8)).SetTaxAmount(link.TaxAmount.StringFixed(8)).SetActive(true).Save(ctx); err != nil {
				return nil, err
			}
			for _, a := range link.Fees {
				if _, err := client.FinanceInvoiceFeeAllocation.Create().SetID(a.ID).SetInvoiceID(inv.ID).SetInvoiceBillID(link.ID).SetBillLineID(a.BillLineID).SetOrderFeeID(a.OrderFeeID).SetFeeCode(a.FeeCode).SetFeeName(a.FeeName).SetCurrency(a.Currency).SetNillableSourceTaxRate(financeDecimalString(a.SourceTaxRate, 4)).SetAmount(a.Amount.StringFixed(8)).SetNetAmount(a.NetAmount.StringFixed(8)).SetTaxAmount(a.TaxAmount.StringFixed(8)).Save(ctx); err != nil {
					return nil, err
				}
			}
		}
		for _, line := range inv.Lines {
			if _, err := client.FinanceInvoiceLine.Create().SetID(line.ID).SetInvoiceID(inv.ID).SetRevisionNumber(revision).SetLineNo(line.LineNo).SetItemCode(line.ItemCode).SetItemName(line.ItemName).SetCurrency(line.Currency).SetNillableTaxRate(financeDecimalString(line.TaxRate, 4)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetTotalAmount(line.TotalAmount.StringFixed(8)).SetSourceLineCount(line.SourceLineCount).Save(ctx); err != nil {
				return nil, err
			}
		}
		if reason != "" {
			if err := saveInvoiceRevision(ctx, client, inv.ID, actor, commandID, revision, reason, existing, inv); err != nil {
				return nil, err
			}
		}
		audit := &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "finance.invoice.register", Result: "success", ResourceType: "finance_invoice", ResourceID: inv.ID.String(), Details: map[string]string{"finance_invoice.id": inv.ID.String(), "command_id": commandID.String()}}
		if reason != "" {
			audit.Action = "finance.invoice.revise"
			audit.Details["reason"] = reason
		}
		if err := writeAudit(ctx, client.AuditLog, audit); err != nil {
			return nil, err
		}
		ids = append(ids, inv.ID)
		delete(oldByID, inv.ID)
	}
	for _, oldID := range sortedMapInvoiceIDs(oldByID) {
		previous := oldByID[oldID]
		revision := previous.RevisionNumber + 1
		// 移除登记也是留痕的显式作废，不删除原凭证或分配。
		if err := client.FinanceInvoice.UpdateOneID(oldID).SetStatus(invoiceent.StatusCANCELLED).SetNillableTaxInvoiceNo(previous.TaxInvoiceNo).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason).SetVersion(previous.Version + 1).SetRevisionNumber(revision).Exec(ctx); err != nil {
			return nil, err
		}
		after := *previous
		after.Status = biz.FinanceInvoiceCancelled
		after.Version++
		after.RevisionNumber = revision
		after.Links = nil
		if err := saveInvoiceRevision(ctx, client, oldID, actor, commandID, revision, reason, previous, &after); err != nil {
			return nil, err
		}
		ids = append(ids, oldID)
		if err := writeAudit(ctx, client.AuditLog, &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "finance.invoice.revise_remove", Result: "success", ResourceType: "finance_invoice", ResourceID: oldID.String(), Details: map[string]string{"reason": reason, "command_id": commandID.String()}}); err != nil {
			return nil, err
		}
	}
	if len(billIDs) > 0 {
		if _, err := client.FinanceBill.Update().Where(billent.IDIn(mapUUIDs(billIDs)...)).AddVersion(1).Save(ctx); err != nil {
			return nil, err
		}
	}
	return ids, nil
}
func sortedMapInvoiceIDs(values map[uuid.UUID]*biz.FinanceInvoice) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}
func mapUUIDs(values map[uuid.UUID]bool) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}
func saveInvoiceRevision(ctx context.Context, client *ent.Client, id, actor, command uuid.UUID, revision uint64, reason string, before, after *biz.FinanceInvoice) error {
	_, err := client.FinanceInvoiceRevision.Create().SetInvoiceID(id).SetActorID(actor).SetCommandID(command).SetRevisionNumber(revision).SetReason(reason).SetBeforeSnapshot(invoiceRevisionSnapshot(before)).SetAfterSnapshot(invoiceRevisionSnapshot(after)).Save(ctx)
	return err
}

func (r *financeInvoiceRepo) GetRevisionImpact(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (*biz.FinanceInvoiceRevisionImpact, error) {
	var result *biz.FinanceInvoiceRevisionImpact
	err := r.data.withReadSnapshot(ctx, func(readCtx context.Context) error {
		invoice, err := r.GetScoped(readCtx, scope, id)
		if err != nil {
			return err
		}
		billIDs := map[uuid.UUID]bool{}
		for _, link := range invoice.Links {
			if link.Active {
				billIDs[link.BillID] = true
			}
		}
		client, err := r.data.client(readCtx)
		if err != nil {
			return err
		}
		items, err := client.FinanceBill.Query().Where(billent.IDIn(mapUUIDs(billIDs)...), billent.OrganizationIDIn(scope...)).WithLines(func(q *ent.FinanceBillLineQuery) { q.WithOrderFee() }).Order(billent.ByID()).All(readCtx)
		if err != nil {
			return err
		}
		if len(items) != len(billIDs) {
			return biz.ErrFinanceInvoiceBillInvalid
		}
		result = &biz.FinanceInvoiceRevisionImpact{Invoice: invoice, CanRevise: invoice.Status == biz.FinanceInvoiceIssued}
		for _, raw := range items {
			b, err := invoiceableFinanceBill(raw)
			if err != nil {
				return err
			}
			result.Bills = append(result.Bills, b)
		}
		if err := enrichInvoiceableBalances(readCtx, client, result.Bills, []uuid.UUID{id}); err != nil {
			return err
		}
		dependencies, err := loadInvoiceSettlementDependencies(readCtx, client, mapUUIDs(billIDs))
		if err != nil {
			return err
		}
		result.Dependencies = dependencies
		result.CanRevise = result.CanRevise && len(dependencies) == 0
		return nil
	})
	return result, err
}
func loadInvoiceSettlementDependencies(ctx context.Context, client *ent.Client, billIDs []uuid.UUID) ([]*biz.FinanceInvoiceSettlementDependency, error) {
	out := []*biz.FinanceInvoiceSettlementDependency{}
	verifications, err := client.FinanceVerificationAllocation.Query().Where(verificationallocent.BillIDIn(billIDs...), verificationallocent.ActiveEQ(true), verificationallocent.HasVerificationWith(verificationent.StatusEQ(verificationent.StatusACTIVE))).WithVerification().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range verifications {
		v, err := decimalOf(a.Amount)
		if err != nil {
			return nil, err
		}
		parent, err := a.Edges.VerificationOrErr()
		if err != nil {
			return nil, err
		}
		out = append(out, &biz.FinanceInvoiceSettlementDependency{ID: parent.ID, BillID: a.BillID, Kind: "VERIFICATION", DocumentNo: parent.VerificationNo, BillNo: a.BillNo, Currency: parent.Currency, Amount: v})
	}
	nettings, err := client.FinanceNettingAllocation.Query().Where(nettingallocent.BillIDIn(billIDs...), nettingallocent.ActiveEQ(true), nettingallocent.HasNettingWith(nettingent.StatusEQ(nettingent.StatusCONFIRMED))).WithNetting().All(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range nettings {
		v, err := decimalOf(a.Amount)
		if err != nil {
			return nil, err
		}
		parent, err := a.Edges.NettingOrErr()
		if err != nil {
			return nil, err
		}
		out = append(out, &biz.FinanceInvoiceSettlementDependency{ID: parent.ID, BillID: a.BillID, Kind: "NETTING", DocumentNo: parent.NettingNo, BillNo: a.BillNo, Currency: parent.Currency, Amount: v})
	}
	return out, nil
}
