package data

import (
	"context"
	"fmt"
	"sort"
	"time"
	"unicode/utf8"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	bill "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	cash "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	nettingallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	ver "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	alloc "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/shopspring/decimal"
)

type verificationRepo struct{ data *Data }

type verificationSummaryRow struct {
	Direction    string `json:"direction"`
	BaseCurrency string `json:"base_currency"`
	BaseAmount   string `json:"base_amount"`
}

func NewVerificationRepo(d *Data) biz.VerificationRepo { return &verificationRepo{d} }
func (r *verificationRepo) withAll(q *ent.FinanceVerificationQuery) *ent.FinanceVerificationQuery {
	return q.WithAllocations(func(x *ent.FinanceVerificationAllocationQuery) { x.Order(alloc.ByCreatedAt()) })
}
func (r *verificationRepo) List(ctx context.Context, org uuid.UUID, f biz.VerificationFilter) (*biz.VerificationListResult, error) {
	return r.ListScoped(ctx, []uuid.UUID{org}, f)
}
func (r *verificationRepo) ListScoped(ctx context.Context, organizationIDs []uuid.UUID, f biz.VerificationFilter) (*biz.VerificationListResult, error) {
	p := []predicate.FinanceVerification{ver.OrganizationIDIn(organizationIDs...)}
	if f.Keyword != "" {
		p = append(p, ver.Or(ver.VerificationNoContainsFold(f.Keyword), ver.SettlementPartyNameContainsFold(f.Keyword), ver.HasAllocationsWith(alloc.Or(alloc.BillNoContainsFold(f.Keyword), alloc.CashflowNoContainsFold(f.Keyword)))))
	}
	if f.Status != "" {
		p = append(p, ver.StatusEQ(ver.Status(f.Status)))
	}
	if f.Direction != "" {
		p = append(p, ver.DirectionEQ(ver.Direction(f.Direction)))
	}
	client, clientErr := r.data.client(ctx)
	if clientErr != nil {
		return nil, clientErr
	}
	q := client.FinanceVerification.Query().WithOrganization().Where(p...)
	n, e := q.Clone().Count(ctx)
	if e != nil {
		return nil, e
	}
	summaryPredicates := append([]predicate.FinanceVerification{}, p...)
	summaryPredicates = append(summaryPredicates, ver.StatusEQ(ver.StatusACTIVE))
	summaryRows := make([]verificationSummaryRow, 0)
	if e := client.FinanceVerification.Query().Where(summaryPredicates...).
		GroupBy(ver.FieldDirection, ver.FieldBaseCurrency).
		Aggregate(ent.As(ent.Sum(ver.FieldBaseAmount), "base_amount")).
		Scan(ctx, &summaryRows); e != nil {
		return nil, e
	}
	summary := biz.VerificationSummary{}
	amountsByBaseCurrency := make(map[string]*biz.FinanceBaseCurrencyAmount, len(summaryRows))
	for _, row := range summaryRows {
		value, parseErr := decimalOf(row.BaseAmount)
		if parseErr != nil {
			return nil, parseErr
		}
		bucket := financeBaseCurrencyAmountFor(amountsByBaseCurrency, row.BaseCurrency)
		if row.Direction == string(ver.DirectionRECEIVABLE) {
			bucket.ReceivableBaseAmount = bucket.ReceivableBaseAmount.Add(value)
		} else {
			bucket.PayableBaseAmount = bucket.PayableBaseAmount.Add(value)
		}
	}
	summary.AmountsByBaseCurrency = financeBaseCurrencyAmountItems(amountsByBaseCurrency)
	xs, e := r.withAll(q).Order(ver.ByVerificationDate(entsql.OrderDesc()), ver.ByCreatedAt(entsql.OrderDesc())).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).All(ctx)
	if e != nil {
		return nil, e
	}
	out := &biz.VerificationListResult{Items: make([]*biz.FinanceVerification, 0, len(xs)), Total: int64(n), Summary: summary}
	for _, x := range xs {
		v, e := verificationToBiz(x)
		if e != nil {
			return nil, e
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}
func (r *verificationRepo) Get(ctx context.Context, org, id uuid.UUID) (*biz.FinanceVerification, error) {
	return r.GetScoped(ctx, []uuid.UUID{org}, id)
}
func (r *verificationRepo) GetScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceVerification, error) {
	client, e := r.data.client(ctx)
	if e != nil {
		return nil, e
	}
	x, e := r.withAll(client.FinanceVerification.Query().WithOrganization()).Where(ver.IDEQ(id), ver.OrganizationIDIn(organizationIDs...)).Only(ctx)
	if e != nil {
		return nil, mapEntError(e, biz.ErrVerificationNotFound, nil)
	}
	return verificationToBiz(x)
}
func (r *verificationRepo) GetByKey(ctx context.Context, org uuid.UUID, key string) (*biz.FinanceVerification, error) {
	client, e := r.data.client(ctx)
	if e != nil {
		return nil, e
	}
	x, e := r.withAll(client.FinanceVerification.Query().WithOrganization()).Where(ver.OrganizationIDEQ(org), ver.IdempotencyKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return verificationToBiz(x)
}
func (r *verificationRepo) LoadCashflowContext(ctx context.Context, org, id uuid.UUID) (*biz.FinanceCashflow, error) {
	return r.LoadCashflowContextScoped(ctx, []uuid.UUID{org}, id)
}
func (r *verificationRepo) LoadCashflowContextScoped(ctx context.Context, organizationIDs []uuid.UUID, id uuid.UUID) (*biz.FinanceCashflow, error) {
	client, e := r.data.client(ctx)
	if e != nil {
		return nil, e
	}
	x, e := client.FinanceCashflow.Query().WithOrganization().Where(cash.IDEQ(id), cash.OrganizationIDIn(organizationIDs...)).Only(ctx)
	if e != nil {
		return nil, mapEntError(e, biz.ErrFinanceCashflowNotFound, nil)
	}
	return cashflowToBiz(x)
}

// ListCreationCandidates 复用账单和资金流水仓储的组织谓词与有效核销余额计算。
// 未结清/未核销过滤下推到数据库（OnlyUnsettled / OnlyUnverified），保证发生在 LIMIT 之前，
// 避免前列记录全部已结清时未结清候选被截断；Go 层保留防御性余额检查。
// 候选只服务于工作台体验；创建事务仍会重新读取并加锁校验所有来源。
func (r *verificationRepo) ListCreationCandidates(ctx context.Context, organizationID uuid.UUID, filter biz.VerificationCreationCandidateFilter) (*biz.VerificationCreationCandidates, error) {
	cashflowResult, err := (&financeCashflowRepo{data: r.data}).ListScoped(ctx, []uuid.UUID{organizationID}, biz.FinanceCashflowFilter{
		Page:              1,
		PageSize:          biz.MaxListPageSize,
		Direction:         filter.Direction,
		Status:            biz.FinanceCashflowConfirmed,
		SettlementPartyID: &filter.SettlementPartyID,
		Currency:          filter.Currency,
		OnlyUnverified:    true,
	})
	if err != nil {
		return nil, err
	}
	billResult, err := (&financeBillRepo{data: r.data}).List(ctx, []uuid.UUID{organizationID}, biz.FinanceBillFilter{
		Page:              1,
		PageSize:          biz.MaxListPageSize,
		Direction:         filter.Direction,
		Status:            biz.FinanceBillConfirmed,
		SettlementPartyID: &filter.SettlementPartyID,
		Currency:          filter.Currency,
		OnlyUnsettled:     true,
	})
	if err != nil {
		return nil, err
	}
	result := &biz.VerificationCreationCandidates{
		Cashflows: make([]*biz.FinanceCashflow, 0, len(cashflowResult.Items)),
		Bills:     make([]*biz.FinanceBill, 0, len(billResult.Items)),
	}
	for _, cashflow := range cashflowResult.Items {
		if cashflow.UnverifiedAmount.IsPositive() {
			result.Cashflows = append(result.Cashflows, cashflow)
		}
	}
	for _, bill := range billResult.Items {
		if bill.UnverifiedAmount.IsPositive() {
			result.Bills = append(result.Bills, bill)
		}
	}
	return result, nil
}
func (r *verificationRepo) Create(ctx context.Context, org, actor uuid.UUID, v *biz.FinanceVerification, audit *biz.AuditEvent) (*biz.FinanceVerification, error) {
	var e error
	e = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		cashSet, billSet := make(map[uuid.UUID]struct{}), make(map[uuid.UUID]struct{})
		for _, a := range v.Allocations {
			cashSet[a.CashflowID] = struct{}{}
			billSet[a.BillID] = struct{}{}
		}
		cashIDs := sortedFinanceUUIDs(cashSet)
		billIDs := sortedFinanceUUIDs(billSet)
		cashs, e := tx.FinanceCashflow.Query().Where(cash.IDIn(cashIDs...), cash.OrganizationIDEQ(org)).Order(cash.ByID()).ForUpdate().All(ctx)
		if e != nil {
			return e
		}
		bills, e := tx.FinanceBill.Query().Where(bill.IDIn(billIDs...), bill.OrganizationIDEQ(org)).Order(bill.ByID()).ForUpdate().All(ctx)
		if e != nil {
			return e
		}
		cm := map[uuid.UUID]*ent.FinanceCashflow{}
		for _, x := range cashs {
			cm[x.ID] = x
		}
		bm := map[uuid.UUID]*ent.FinanceBill{}
		for _, x := range bills {
			bm[x.ID] = x
		}
		usedCash, usedBill := map[uuid.UUID]decimal.Decimal{}, map[uuid.UUID]decimal.Decimal{}
		v.BaseAmount, v.BillBaseAmount, v.CashflowBaseAmount, v.ExchangeGainLoss = decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
		existing, e := tx.FinanceVerificationAllocation.Query().Where(alloc.ActiveEQ(true), alloc.Or(alloc.CashflowIDIn(cashIDs...), alloc.BillIDIn(billIDs...))).All(ctx)
		if e != nil {
			return e
		}
		for _, x := range existing {
			z, parseErr := decimalOf(x.Amount)
			if parseErr != nil {
				return parseErr
			}
			usedCash[x.CashflowID] = usedCash[x.CashflowID].Add(z)
			usedBill[x.BillID] = usedBill[x.BillID].Add(z)
		}
		// 有效对冲分摊同样占用账单余额；普通资金核销只能处理抵销后的剩余余额。
		nettingExisting, e := tx.FinanceNettingAllocation.Query().Where(nettingallocent.BillIDIn(billIDs...), nettingallocent.ActiveEQ(true)).All(ctx)
		if e != nil {
			return e
		}
		for _, x := range nettingExisting {
			z, parseErr := decimalOf(x.Amount)
			if parseErr != nil {
				return parseErr
			}
			usedBill[x.BillID] = usedBill[x.BillID].Add(z)
		}
		for i, x := range v.Allocations {
			c, b := cm[x.CashflowID], bm[x.BillID]
			if c == nil || b == nil || c.Status != cash.StatusCONFIRMED || b.Status != bill.StatusCONFIRMED {
				return biz.ErrVerificationBalance
			}
			if c.Direction != cash.Direction(b.Direction) || c.SettlementPartyID != b.SettlementPartyID || c.Currency != b.Currency || c.BaseCurrency != b.BaseCurrency || c.BaseCurrency != v.BaseCurrency || biz.OrderFeeDirection(c.Direction) != v.Direction || c.Currency != v.Currency {
				return biz.ErrVerificationMismatch
			}
			if i == 0 {
				v.Direction = biz.OrderFeeDirection(c.Direction)
				v.SettlementPartyID = c.SettlementPartyID
				v.SettlementPartyName = c.SettlementPartyName
				v.Currency = c.Currency
			} else if v.Direction != biz.OrderFeeDirection(c.Direction) || v.SettlementPartyID != c.SettlementPartyID || v.Currency != c.Currency {
				return biz.ErrVerificationMismatch
			}
			ca, parseErr := decimalOf(c.Amount)
			if parseErr != nil {
				return parseErr
			}
			ba, parseErr := decimalOf(b.TotalAmount)
			if parseErr != nil {
				return parseErr
			}
			cashBaseTotal, parseErr := decimalOf(c.BaseAmount)
			if parseErr != nil {
				return parseErr
			}
			billBaseTotal, parseErr := decimalOf(b.BaseCurrencyAmount)
			if parseErr != nil {
				return parseErr
			}
			usedCash[c.ID] = usedCash[c.ID].Add(x.Amount)
			usedBill[b.ID] = usedBill[b.ID].Add(x.Amount)
			if usedCash[c.ID].GreaterThan(ca) || usedBill[b.ID].GreaterThan(ba) {
				return biz.ErrVerificationBalance
			}
			x.CashflowNo = c.FlowNo
			x.BillNo = b.BillNo
			x.BillBaseAmount, x.CashflowBaseAmount, x.WriteOffBaseAmount, x.ExchangeGainLoss, e = biz.CalculateVerificationAllocationAmounts(v.Direction, x.Amount, ba, billBaseTotal, ca, cashBaseTotal, v.ExchangeRate)
			if e != nil {
				return e
			}
			v.BillBaseAmount = v.BillBaseAmount.Add(x.BillBaseAmount)
			v.CashflowBaseAmount = v.CashflowBaseAmount.Add(x.CashflowBaseAmount)
			v.BaseAmount = v.BaseAmount.Add(x.WriteOffBaseAmount)
			v.ExchangeGainLoss = v.ExchangeGainLoss.Add(x.ExchangeGainLoss)
		}
		v.BaseAmount = v.BaseAmount.RoundBank(8)
		v.BillBaseAmount = v.BillBaseAmount.RoundBank(8)
		v.CashflowBaseAmount = v.CashflowBaseAmount.RoundBank(8)
		v.ExchangeGainLoss = v.ExchangeGainLoss.RoundBank(8)
		now := time.Now().UTC()
		rule, sequence, e := allocateNumberInTx(ctx, tx, org, biz.DocumentTypeWriteOff, now)
		if e != nil {
			return e
		}
		v.VerificationNo, e = biz.FormatAllocatedNumber(now, rule, sequence, "")
		if e != nil {
			return e
		}
		_, e = tx.FinanceVerification.Create().SetID(v.ID).SetOrganizationID(org).SetVerificationNo(v.VerificationNo).SetIdempotencyKey(v.IdempotencyKey).SetStatus(ver.StatusACTIVE).SetDirection(ver.Direction(v.Direction)).SetSettlementPartyID(v.SettlementPartyID).SetSettlementPartyName(v.SettlementPartyName).SetCurrency(v.Currency).SetAmount(v.Amount.StringFixed(8)).SetBaseCurrency(v.BaseCurrency).SetExchangeRate(v.ExchangeRate.StringFixed(8)).SetExchangeRateSource(ver.ExchangeRateSource(v.ExchangeRateSource)).SetExchangeRateDate(v.ExchangeRateDate).SetNillableExchangeRateSettingID(v.ExchangeRateSettingID).SetBaseAmount(v.BaseAmount.StringFixed(8)).SetBillBaseAmount(v.BillBaseAmount.StringFixed(8)).SetCashflowBaseAmount(v.CashflowBaseAmount.StringFixed(8)).SetExchangeGainLoss(v.ExchangeGainLoss.StringFixed(8)).SetVerificationDate(v.VerificationDate).SetNillableNote(v.Note).SetVersion(1).Save(ctx)
		if e != nil {
			return mapEntConstraint(e, "financeverification_org_idempotency", biz.ErrVerificationIdempotency)
		}
		builders := make([]*ent.FinanceVerificationAllocationCreate, 0, len(v.Allocations))
		for _, x := range v.Allocations {
			builders = append(builders, tx.FinanceVerificationAllocation.Create().SetID(x.ID).SetVerificationID(v.ID).SetCashflowID(x.CashflowID).SetBillID(x.BillID).SetCashflowNo(x.CashflowNo).SetBillNo(x.BillNo).SetAmount(x.Amount.StringFixed(8)).SetBillBaseAmount(x.BillBaseAmount.StringFixed(8)).SetCashflowBaseAmount(x.CashflowBaseAmount.StringFixed(8)).SetWriteOffBaseAmount(x.WriteOffBaseAmount.StringFixed(8)).SetExchangeGainLoss(x.ExchangeGainLoss.StringFixed(8)).SetActive(true))
		}
		if _, e = tx.FinanceVerificationAllocation.CreateBulk(builders...).Save(ctx); e != nil {
			return mapEntConstraint(e, "verification_allocation_pair_unique", biz.ErrVerificationInvalid)
		}
		if e = writeAudit(ctx, tx.AuditLog, audit); e != nil {
			return e
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	if _, transactional := transactionFromContext(ctx); transactional {
		return v, nil
	}
	return r.Get(ctx, org, v.ID)
}

func sortedFinanceUUIDs(values map[uuid.UUID]struct{}) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func (r *verificationRepo) Reverse(ctx context.Context, org, id, actor uuid.UUID, version uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceVerification, error) {
	e := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		x, queryErr := tx.FinanceVerification.Query().Where(ver.IDEQ(id), ver.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrVerificationNotFound, nil)
		}
		if x.Version != version || x.Status != ver.StatusACTIVE {
			return biz.ErrVerificationTransition
		}
		if reconcileErr := reconcileCommissionsForVerificationReversal(ctx, tx, org, id, actor, reason); reconcileErr != nil {
			return reconcileErr
		}
		if _, updateErr := tx.FinanceVerificationAllocation.Update().Where(alloc.VerificationIDEQ(id), alloc.ActiveEQ(true)).SetActive(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		now := time.Now()
		if _, updateErr := tx.FinanceVerification.UpdateOneID(id).SetStatus(ver.StatusREVERSED).SetReversedAt(now).SetReversedBy(actor).SetReversalReason(reason).SetVersion(version + 1).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if e != nil {
		return nil, e
	}
	return r.Get(ctx, org, id)
}

// reconcileCommissionsForVerificationReversal 在同一事务内撤销未支付提成，或对已支付提成形成待追回冲减。
func reconcileCommissionsForVerificationReversal(ctx context.Context, tx *ent.Tx, org, verificationID, actor uuid.UUID, reason string) error {
	commissions, err := tx.FinanceCommission.Query().Where(
		commission.OrganizationIDEQ(org), commission.VerificationIDEQ(verificationID), commission.StatusNEQ(commission.StatusCANCELLED),
	).Order(commission.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	cancellationReason := limitedFinanceReason("核销撤销自动取消：" + reason)
	recoveryReason := limitedFinanceReason("核销撤销自动冲减：" + reason)
	for _, parent := range commissions {
		lines, queryErr := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(parent.ID)).Order(commissionline.ByID()).ForUpdate().All(ctx)
		if queryErr != nil {
			return queryErr
		}
		adjustments, queryErr := tx.FinanceCommissionAdjustment.Query().Where(
			adjustment.CommissionIDEQ(parent.ID), adjustment.StatusNEQ(adjustment.StatusCANCELLED),
		).Order(adjustment.ByCreatedAt(), adjustment.ByID()).ForUpdate().All(ctx)
		if queryErr != nil {
			return queryErr
		}
		baseAmount, parseErr := decimalOf(parent.CommissionAmount)
		if parseErr != nil {
			return parseErr
		}
		domainLines := make([]biz.CommissionReversalLine, 0, len(lines))
		lineByOrder := make(map[uuid.UUID]*ent.FinanceCommissionLine, len(lines))
		for _, line := range lines {
			lineAmount, amountErr := decimalOf(line.CommissionAmount)
			if amountErr != nil {
				return amountErr
			}
			domainLines = append(domainLines, biz.CommissionReversalLine{OrderID: line.OrderID, Amount: lineAmount})
			lineByOrder[line.OrderID] = line
		}
		domainAdjustments := make([]*biz.FinanceCommissionAdjustment, 0, len(adjustments))
		for _, item := range adjustments {
			converted, convertErr := commissionAdjustmentToBiz(item)
			if convertErr != nil {
				return convertErr
			}
			domainAdjustments = append(domainAdjustments, converted)
		}
		plan, planErr := biz.PlanCommissionReversal(biz.CommissionStatus(parent.Status), baseAmount, domainLines, domainAdjustments)
		if planErr != nil {
			return planErr
		}
		if len(plan.CancelAdjustmentIDs) > 0 {
			if _, queryErr = tx.FinanceCommissionAdjustment.Update().Where(
				adjustment.IDIn(plan.CancelAdjustmentIDs...),
			).SetStatus(adjustment.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(cancellationReason).AddVersion(1).Save(ctx); queryErr != nil {
				return queryErr
			}
		}
		if plan.CancelCommission {
			if _, queryErr = tx.FinanceCommission.UpdateOne(parent).SetStatus(commission.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(cancellationReason).AddVersion(1).Save(ctx); queryErr != nil {
				return queryErr
			}
			continue
		}
		if len(plan.Recoveries) == 0 {
			continue
		}
		sequence := parent.AdjustmentSequence
		for _, recovery := range plan.Recoveries {
			line := lineByOrder[recovery.OrderID]
			if line == nil {
				return biz.ErrCommissionSource
			}
			sequence++
			idempotencyKey := fmt.Sprintf("vr:%s:%s:%s", verificationID, parent.ID, line.OrderID)
			_, createErr := tx.FinanceCommissionAdjustment.Create().
				SetID(uuid.Must(uuid.NewV7())).SetOrganizationID(org).SetCommissionID(parent.ID).SetOrderID(line.OrderID).
				SetAdjustmentNo(fmt.Sprintf("%s-ADJ%03d", parent.CommissionNo, sequence)).SetIdempotencyKey(idempotencyKey).
				SetCommissionNo(parent.CommissionNo).SetOrderNo(line.OrderNo).SetEmployeeID(parent.EmployeeID).SetEmployeeName(parent.EmployeeName).
				SetSourceType(adjustment.SourceTypeVERIFICATION_REVERSAL).SetSourceVerificationID(verificationID).
				SetDirection(adjustment.DirectionDECREASE).SetStatus(adjustment.StatusCONFIRMED).SetBaseCurrency(parent.BaseCurrency).
				SetAmount(recovery.Amount.StringFixed(8)).SetReason(recoveryReason).SetVersion(1).SetConfirmedAt(now).SetConfirmedBy(actor).Save(ctx)
			if createErr != nil {
				return createErr
			}
		}
		if _, queryErr = tx.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).AddVersion(1).Save(ctx); queryErr != nil {
			return queryErr
		}
	}
	return nil
}

func limitedFinanceReason(value string) string {
	if utf8.RuneCountInString(value) <= 500 {
		return value
	}
	runes := []rune(value)
	return string(runes[:500])
}
func verificationToBiz(x *ent.FinanceVerification) (*biz.FinanceVerification, error) {
	amount, e := decimalOf(x.Amount)
	if e != nil {
		return nil, e
	}
	exchangeRate, e := decimalOf(x.ExchangeRate)
	if e != nil {
		return nil, e
	}
	baseAmount, e := decimalOf(x.BaseAmount)
	if e != nil {
		return nil, e
	}
	billBaseAmount, e := decimalOf(x.BillBaseAmount)
	if e != nil {
		return nil, e
	}
	cashflowBaseAmount, e := decimalOf(x.CashflowBaseAmount)
	if e != nil {
		return nil, e
	}
	exchangeGainLoss, e := decimalOf(x.ExchangeGainLoss)
	if e != nil {
		return nil, e
	}
	v := &biz.FinanceVerification{ID: x.ID, OrganizationID: x.OrganizationID, VerificationNo: x.VerificationNo, IdempotencyKey: x.IdempotencyKey, Status: biz.VerificationStatus(x.Status), Direction: biz.OrderFeeDirection(x.Direction), SettlementPartyID: x.SettlementPartyID, SettlementPartyName: x.SettlementPartyName, Currency: x.Currency, Amount: amount, BaseCurrency: x.BaseCurrency, ExchangeRate: exchangeRate, ExchangeRateSource: string(x.ExchangeRateSource), ExchangeRateDate: x.ExchangeRateDate, ExchangeRateSettingID: x.ExchangeRateSettingID, BaseAmount: baseAmount, BillBaseAmount: billBaseAmount, CashflowBaseAmount: cashflowBaseAmount, ExchangeGainLoss: exchangeGainLoss, VerificationDate: x.VerificationDate, Note: x.Note, Version: x.Version, ReversedAt: x.ReversedAt, ReversedBy: x.ReversedBy, ReversalReason: x.ReversalReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, Allocations: make([]*biz.VerificationAllocation, 0, len(x.Edges.Allocations))}
	if organization, edgeErr := x.Edges.OrganizationOrErr(); edgeErr == nil {
		v.OrganizationName = organization.Name
	}
	for _, a := range x.Edges.Allocations {
		z, e := decimalOf(a.Amount)
		if e != nil {
			return nil, e
		}
		billBase, e := decimalOf(a.BillBaseAmount)
		if e != nil {
			return nil, e
		}
		cashBase, e := decimalOf(a.CashflowBaseAmount)
		if e != nil {
			return nil, e
		}
		writeOffBase, e := decimalOf(a.WriteOffBaseAmount)
		if e != nil {
			return nil, e
		}
		gainLoss, e := decimalOf(a.ExchangeGainLoss)
		if e != nil {
			return nil, e
		}
		v.Allocations = append(v.Allocations, &biz.VerificationAllocation{ID: a.ID, VerificationID: a.VerificationID, CashflowID: a.CashflowID, BillID: a.BillID, CashflowNo: a.CashflowNo, BillNo: a.BillNo, Amount: z, BillBaseAmount: billBase, CashflowBaseAmount: cashBase, WriteOffBaseAmount: writeOffBase, ExchangeGainLoss: gainLoss, Active: a.Active})
	}
	return v, nil
}

var _ biz.VerificationRepo = (*verificationRepo)(nil)
