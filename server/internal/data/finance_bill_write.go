package data

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	financenettingallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	verificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
)

func (r *financeBillRepo) Create(ctx context.Context, bill *biz.FinanceBill, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		feeIDs := make([]uuid.UUID, 0, len(bill.Lines))
		expected := make(map[uuid.UUID]*biz.FinanceBillLine, len(bill.Lines))
		for _, line := range bill.Lines {
			feeIDs = append(feeIDs, line.OrderFeeID)
			expected[line.OrderFeeID] = line
		}
		if err := lockBillFeeOrders(ctx, tx.Client(), []uuid.UUID{bill.OrganizationID}, feeIDs, true); err != nil {
			return err
		}
		fees, err := tx.OrderFee.Query().Where(orderfeeent.IDIn(feeIDs...), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(bill.OrganizationID))).Order(orderfeeent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(fees) != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		for _, fee := range fees {
			line := expected[fee.ID]
			if line == nil || fee.EconomicEffect != orderfeeent.EconomicEffectINCREASE || entFeeSettlementDirection(fee) != bill.Direction || entFeeBillPurpose(fee) != bill.Purpose || !fee.AccountingRequired || fee.SupersededByRequestID != nil || fee.SettlementPartyID != bill.SettlementPartyID || fee.Currency != bill.Currency || fee.BaseCurrency != bill.BaseCurrency || fee.TotalAmount != line.TotalAmount.StringFixed(8) || fee.NetAmount != line.NetAmount.StringFixed(8) || fee.TaxAmount != line.TaxAmount.StringFixed(8) || fee.BaseCurrencyAmount != line.BaseCurrencyAmount.StringFixed(8) || !financeDecimalStringEqual(fee.TaxRate, line.TaxRate, 4) {
				return biz.ErrFinanceBillFeeInvalid
			}
		}
		// 关联资格在插入账单行前锁内复核（费用行已 ForUpdate）：存在有效账单关联
		// （活动账单行且所属账单未取消，含草稿账单）即视为已建账。
		occupied, err := tx.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDIn(feeIDs...), effectiveBillLinePredicate()).Exist(ctx)
		if err != nil {
			return err
		}
		if occupied {
			return biz.ErrFinanceBillFeeInvalid
		}
		if err = hydrateFinanceBillSettlementAccount(ctx, tx, bill); err != nil {
			return err
		}
		now := time.Now().UTC()
		billRule, billSequence, err := allocateNumberInTx(ctx, tx, bill.OrganizationID, biz.DocumentTypeBill, now)
		if err != nil {
			return err
		}
		bill.BillNo, err = biz.FormatAllocatedNumber(now, billRule, billSequence, "")
		if err != nil {
			return err
		}
		_, err = tx.FinanceBill.Create().
			SetID(bill.ID).SetOrganizationID(bill.OrganizationID).SetBillNo(bill.BillNo).SetIdempotencyKey(bill.IdempotencyKey).
			SetPurpose(financebillent.Purpose(bill.Purpose)).SetDirection(financebillent.Direction(bill.Direction)).SetStatus(financebillent.StatusDRAFT).
			SetNillableBatchID(bill.BatchID).
			SetSettlementPartyID(bill.SettlementPartyID).SetSettlementPartyName(bill.SettlementPartyName).
			SetSettlementAccountID(bill.SettlementAccountID).SetSettlementAccountName(bill.SettlementAccountName).SetSettlementAccountHolder(bill.SettlementAccountHolder).SetSettlementBankName(bill.SettlementBankName).SetSettlementBankAccount(bill.SettlementBankAccount).SetSettlementAccountCurrency(bill.SettlementAccountCurrency).SetSettlementSwiftCode(bill.SettlementSwiftCode).SetNillableEstimatedInvoiceCurrency(bill.EstimatedInvoiceCurrency).SetNillableEstimatedInvoiceRate(financeDecimalString(bill.EstimatedInvoiceRate, 8)).SetNillableEstimatedInvoiceAmount(financeDecimalString(bill.EstimatedInvoiceAmount, 8)).
			SetCurrency(bill.Currency).SetBaseCurrency(bill.BaseCurrency).SetExchangeRate(bill.ExchangeRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(bill.ExchangeRateSource)).SetExchangeRateDate(bill.ExchangeRateDate).SetNillableExchangeRateSettingID(bill.ExchangeRateSettingID).
			SetTotalAmount(bill.TotalAmount.StringFixed(8)).SetNetAmount(bill.NetAmount.StringFixed(8)).SetTaxAmount(bill.TaxAmount.StringFixed(8)).SetBaseCurrencyAmount(bill.BaseCurrencyAmount.StringFixed(8)).
			SetFeeCount(bill.FeeCount).SetBillDate(bill.BillDate).SetNillableStatementTitle(bill.StatementTitle).SetNillablePaymentTermsDays(bill.PaymentTermsDays).SetNillableDueDate(bill.DueDate).SetNillableNote(bill.Note).SetVersion(1).Save(ctx)
		if err != nil {
			return mapEntConstraint(err, "idempotency", biz.ErrFinanceBillIdempotencyConflict)
		}
		builders := make([]*ent.FinanceBillLineCreate, 0, len(bill.Lines))
		for _, line := range bill.Lines {
			builders = append(builders, tx.FinanceBillLine.Create().
				SetID(line.ID).SetBillID(bill.ID).SetOrderFeeID(line.OrderFeeID).SetOrderID(line.OrderID).
				SetOrderNo(line.OrderNo).SetFeeCode(line.FeeCode).SetFeeName(line.FeeName).SetQuantity(line.Quantity.StringFixed(4)).SetUnitPrice(line.UnitPrice.StringFixed(4)).
				SetTotalAmount(line.TotalAmount.StringFixed(8)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetNillableTaxRate(financeDecimalString(line.TaxRate, 4)).SetCurrency(line.Currency).
				SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrency(line.BaseCurrency).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8)).
				SetActive(true))
		}
		if _, err = tx.FinanceBillLine.CreateBulk(builders...).Save(ctx); err != nil {
			return mapEntError(err, nil, biz.ErrFinanceBillFeeInvalid)
		}
		// 费用行已在锁内通过关联资格复核，直接递增版本；受影响行数校验保留。
		affected, err := tx.OrderFee.Update().Where(orderfeeent.IDIn(feeIDs...)).AddVersion(1).Save(ctx)
		if err != nil {
			return err
		}
		if affected != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	if _, transactional := transactionFromContext(ctx); transactional {
		return bill, nil
	}
	return r.Get(ctx, []uuid.UUID{bill.OrganizationID}, bill.ID)
}

func (r *financeBillRepo) Update(ctx context.Context, organizationIDs []uuid.UUID, input biz.UpdateFinanceBillInput, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if err := lockExistingBillOrders(ctx, tx.Client(), organizationIDs, []uuid.UUID{input.ID}); err != nil {
			return err
		}
		item, err := tx.FinanceBill.Query().Where(financebillent.IDEQ(input.ID), financeBillOrganizationScopePredicate(organizationIDs)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		protected, err := tx.OrderFee.Query().Where(orderfeeent.SupersededByRequestIDNotNil(), orderfeeent.HasFinanceBillLinesWith(financebilllineent.BillIDEQ(input.ID), financebilllineent.ActiveEQ(true))).Exist(ctx)
		if err != nil {
			return err
		}
		if protected {
			return biz.ErrFeeCorrectionProtected
		}
		if item.Version != input.ExpectedVersion {
			return biz.ErrFinanceBillVersionConflict
		}
		if item.Status != financebillent.StatusDRAFT {
			return biz.ErrFinanceBillInvalidTransition
		}
		candidate := &biz.FinanceBill{OrganizationID: item.OrganizationID, SettlementPartyID: item.SettlementPartyID, Direction: biz.OrderFeeDirection(item.Direction), Currency: item.Currency, SettlementAccountID: input.SettlementAccountID}
		if err = hydrateFinanceBillSettlementAccount(ctx, tx, candidate); err != nil {
			return err
		}
		update := tx.FinanceBill.UpdateOneID(input.ID).SetBillDate(input.BillDate).SetSettlementAccountID(input.SettlementAccountID).SetSettlementAccountName(candidate.SettlementAccountName).SetSettlementAccountHolder(candidate.SettlementAccountHolder).SetSettlementBankName(candidate.SettlementBankName).SetSettlementBankAccount(candidate.SettlementBankAccount).SetSettlementAccountCurrency(candidate.SettlementAccountCurrency).SetSettlementSwiftCode(candidate.SettlementSwiftCode).SetExchangeRate(input.ExchangeRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(input.ExchangeRateSource)).SetExchangeRateDate(input.ExchangeRateDate).SetTotalAmount(input.TotalAmount.StringFixed(8)).SetNetAmount(input.NetAmount.StringFixed(8)).SetTaxAmount(input.TaxAmount.StringFixed(8)).SetBaseCurrencyAmount(input.BaseCurrencyAmount.StringFixed(8)).SetVersion(item.Version + 1)
		if input.ExchangeRateSettingID == nil {
			update.ClearExchangeRateSettingID()
		} else {
			update.SetExchangeRateSettingID(*input.ExchangeRateSettingID)
		}
		if input.DueDate == nil {
			update.ClearDueDate()
		} else {
			update.SetDueDate(*input.DueDate)
		}
		if input.Note == nil {
			update.ClearNote()
		} else {
			update.SetNote(*input.Note)
		}
		if input.StatementTitle == nil {
			update.ClearStatementTitle()
		} else {
			update.SetStatementTitle(*input.StatementTitle)
		}
		if input.PaymentTermsDays == nil {
			update.ClearPaymentTermsDays()
		} else {
			update.SetPaymentTermsDays(*input.PaymentTermsDays)
		}
		if input.EstimatedInvoiceCurrency == nil || input.EstimatedInvoiceRate == nil || input.EstimatedInvoiceAmount == nil {
			update.ClearEstimatedInvoiceCurrency().ClearEstimatedInvoiceRate().ClearEstimatedInvoiceAmount()
		} else {
			update.SetEstimatedInvoiceCurrency(*input.EstimatedInvoiceCurrency).SetEstimatedInvoiceRate(input.EstimatedInvoiceRate.StringFixed(8)).SetEstimatedInvoiceAmount(input.EstimatedInvoiceAmount.StringFixed(8))
		}
		if _, err = update.Save(ctx); err != nil {
			return err
		}
		lockedLines, err := tx.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(input.ID), financebilllineent.ActiveEQ(true)).Order(financebilllineent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(lockedLines) != len(input.Lines) {
			return biz.ErrFinanceBillBatchMismatch
		}
		byID := make(map[uuid.UUID]*biz.FinanceBillLine, len(input.Lines))
		for _, line := range input.Lines {
			byID[line.ID] = line
		}
		for _, stored := range lockedLines {
			line := byID[stored.ID]
			if line == nil || line.OrderFeeID != stored.OrderFeeID {
				return biz.ErrFinanceBillBatchMismatch
			}
			lineUpdate := tx.FinanceBillLine.UpdateOneID(stored.ID).
				SetUnitPrice(line.UnitPrice.StringFixed(4)).SetTotalAmount(line.TotalAmount.StringFixed(8)).SetNetAmount(line.NetAmount.StringFixed(8)).SetTaxAmount(line.TaxAmount.StringFixed(8)).SetCurrency(line.Currency).
				SetExchangeRate(line.ExchangeRate.StringFixed(8)).SetBaseCurrencyAmount(line.BaseCurrencyAmount.StringFixed(8))
			if _, err = lineUpdate.Save(ctx); err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationIDs, input.ID)
}

func (r *financeBillRepo) Confirm(ctx context.Context, organizationIDs []uuid.UUID, id, actorID uuid.UUID, expectedVersion uint64, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if err := lockExistingBillOrders(ctx, tx.Client(), organizationIDs, []uuid.UUID{id}); err != nil {
			return err
		}
		item, err := tx.FinanceBill.Query().Where(financebillent.IDEQ(id), financeBillOrganizationScopePredicate(organizationIDs)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrFinanceBillVersionConflict
		}
		if item.Status != financebillent.StatusDRAFT {
			return biz.ErrFinanceBillInvalidTransition
		}
		now := time.Now()
		if _, err = tx.FinanceBill.UpdateOneID(id).SetStatus(financebillent.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actorID).SetVersion(item.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationIDs, id)
}

func (r *financeBillRepo) Cancel(ctx context.Context, organizationIDs []uuid.UUID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *biz.AuditEvent) (*biz.FinanceBill, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if err := lockExistingBillOrders(ctx, tx.Client(), organizationIDs, []uuid.UUID{id}); err != nil {
			return err
		}
		item, err := tx.FinanceBill.Query().Where(financebillent.IDEQ(id), financeBillOrganizationScopePredicate(organizationIDs)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrFinanceBillNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrFinanceBillVersionConflict
		}
		if item.Status != financebillent.StatusDRAFT && item.Status != financebillent.StatusCONFIRMED {
			return biz.ErrFinanceBillInvalidTransition
		}
		if err := billCancellationDependencies(ctx, tx.Client(), item); err != nil {
			return err
		}
		lines, err := tx.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(id), financebilllineent.ActiveEQ(true)).Order(financebilllineent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			return biz.ErrFinanceBillInvalidTransition
		}
		feeIDs := make([]uuid.UUID, 0, len(lines))
		for _, line := range lines {
			feeIDs = append(feeIDs, line.OrderFeeID)
		}
		sort.Slice(feeIDs, func(i, j int) bool { return feeIDs[i].String() < feeIDs[j].String() })
		fees, err := tx.OrderFee.Query().Where(orderfeeent.IDIn(feeIDs...)).Order(orderfeeent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(fees) != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		for _, fee := range fees {
			if fee.SupersededByRequestID != nil {
				return biz.ErrFeeCorrectionProtected
			}
		}
		// 取消以账单行为准：先停用本账单的有效行，再依据「费用仍被其他有效关联
		// 占用则拒绝」的口径复核；费用行已锁定，版本递增不再依赖状态条件。
		affected, err := tx.OrderFee.Update().Where(orderfeeent.IDIn(feeIDs...)).AddVersion(1).Save(ctx)
		if err != nil {
			return err
		}
		if affected != len(feeIDs) {
			return biz.ErrFinanceBillFeeInvalid
		}
		if _, err = tx.FinanceBillLine.Update().Where(financebilllineent.BillIDEQ(id), financebilllineent.ActiveEQ(true)).SetActive(false).Save(ctx); err != nil {
			return err
		}
		now := time.Now()
		if _, err = tx.FinanceBill.UpdateOneID(id).SetStatus(financebillent.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actorID).SetCancellationReason(reason).SetVersion(item.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationIDs, id)
}

// hydrateFinanceBillSettlementAccount 在账单写事务内重读账户并固化快照，不能信任请求端传来的快照。
func hydrateFinanceBillSettlementAccount(ctx context.Context, tx *ent.Tx, bill *biz.FinanceBill) error {
	return hydrateFinanceBillSettlementAccounts(ctx, tx, []*biz.FinanceBill{bill})
}

// hydrateFinanceBillSettlementAccounts 先按账户主键固定顺序取得共享锁，再逐账单校验账户事实。
// 这样批量建账不会按前端分组顺序与账户默认值切换形成反序锁等待。
func hydrateFinanceBillSettlementAccounts(ctx context.Context, tx *ent.Tx, bills []*biz.FinanceBill) error {
	accountIDs := make([]uuid.UUID, 0, len(bills))
	seen := make(map[uuid.UUID]struct{}, len(bills))
	for _, bill := range bills {
		if bill == nil || bill.SettlementAccountID == uuid.Nil || bill.OrganizationID == uuid.Nil || bill.SettlementPartyID == uuid.Nil || (bill.Direction != biz.OrderFeeReceivable && bill.Direction != biz.OrderFeePayable) || len(bill.Currency) != 3 {
			return biz.ErrFinanceBillSettlementAccountInvalid
		}
		if _, exists := seen[bill.SettlementAccountID]; !exists {
			seen[bill.SettlementAccountID] = struct{}{}
			accountIDs = append(accountIDs, bill.SettlementAccountID)
		}
	}
	sort.Slice(accountIDs, func(i, j int) bool { return accountIDs[i].String() < accountIDs[j].String() })
	accounts, err := tx.PartnerAccount.Query().Where(partneraccountent.IDIn(accountIDs...)).WithPartner().Order(partneraccountent.ByID()).ForShare().All(ctx)
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
		partner, partnerErr := account.Edges.PartnerOrErr()
		if partnerErr != nil || partner.OrganizationID != bill.OrganizationID {
			return biz.ErrFinanceBillSettlementAccountInvalid
		}
		usage := partneraccountent.UsageRECEIVABLE
		if bill.Direction == biz.OrderFeePayable {
			usage = partneraccountent.UsagePAYABLE
		}
		if account.Usage != usage && account.Usage != partneraccountent.UsageBOTH {
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

func entFeeSettlementDirection(fee *ent.OrderFee) biz.OrderFeeDirection {
	direction := biz.OrderFeeDirection(fee.Direction)
	if fee.EconomicEffect != orderfeeent.EconomicEffectDECREASE {
		return direction
	}
	if direction == biz.OrderFeeReceivable {
		return biz.OrderFeePayable
	}
	return biz.OrderFeeReceivable
}
func entFeeBillPurpose(fee *ent.OrderFee) biz.FinanceBillPurpose {
	return biz.FinanceBillNormal
}

// billCancellationDependencies 返回真实依赖与回退动作；整张账单释放占用后费用才回到未建账。
func billCancellationDependencies(ctx context.Context, client *ent.Client, bill *ent.FinanceBill) error {
	instructions := []string{}
	details := []string{}
	permissions := []string{}
	invoices, err := client.FinanceInvoiceBill.Query().Where(financeinvoicebillent.BillIDEQ(bill.ID), financeinvoicebillent.ActiveEQ(true)).WithInvoice().Order(financeinvoicebillent.ByID()).All(ctx)
	if err != nil {
		return err
	}
	for _, link := range invoices {
		head := link.Edges.Invoice
		action := "取消草稿发票"
		permission := access.FinanceInvoiceUpdate
		if string(head.Status) != "DRAFT" {
			action = "红冲已开具发票"
			permission = access.FinanceInvoiceUpdate
		}
		instructions = append(instructions, action+" "+head.RecordNo)
		details = append(details, "invoice:"+head.ID.String())
		permissions = append(permissions, permission)
	}
	verifications, err := client.FinanceVerificationAllocation.Query().Where(verificationallocationent.BillIDEQ(bill.ID), verificationallocationent.ActiveEQ(true)).WithVerification().Order(verificationallocationent.ByID()).All(ctx)
	if err != nil {
		return err
	}
	for _, link := range verifications {
		head := link.Edges.Verification
		instructions = append(instructions, "反核销 "+head.VerificationNo)
		details = append(details, "verification:"+head.ID.String())
		permissions = append(permissions, access.FinanceVerificationReverse)
	}
	nettings, err := client.FinanceNettingAllocation.Query().Where(financenettingallocationent.BillIDEQ(bill.ID), financenettingallocationent.Or(financenettingallocationent.ActiveEQ(true), financenettingallocationent.HasNettingWith(financenettingent.StatusEQ(financenettingent.StatusDRAFT)))).WithNetting().Order(financenettingallocationent.ByID()).All(ctx)
	if err != nil {
		return err
	}
	for _, link := range nettings {
		head := link.Edges.Netting
		action := "反对冲"
		permission := access.FinanceNettingReverse
		if head.Status == financenettingent.StatusDRAFT {
			action = "取消草稿对冲"
			permission = access.FinanceNettingReverse
		}
		instructions = append(instructions, action+" "+head.NettingNo)
		details = append(details, "netting:"+head.ID.String())
		permissions = append(permissions, permission)
	}
	if len(instructions) == 0 {
		return nil
	}
	return errors.Conflict("FINANCE_BILL_CANCELLATION_BLOCKED", "请先按对应权限完成："+strings.Join(instructions, "；")+"；再整张取消账单 "+bill.BillNo+"（将释放该账单全部费用）").WithMetadata(map[string]string{"bill_id": bill.ID.String(), "bill_no": bill.BillNo, "dependencies": strings.Join(details, ","), "required_permissions": strings.Join(permissions, ","), "rollback_scope": "WHOLE_BILL"})
}
