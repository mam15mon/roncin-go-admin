package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// settlementOrderLockFixture 结清后保持人工审核状态的集成测试夹具：独立组织 + SE 订单 +
// 共享 MBL 组 + 应收账单/核销事实。
type settlementOrderLockFixture struct {
	t                 *testing.T
	data              *Data
	ctx               context.Context
	organizationID    uuid.UUID
	partnerID         uuid.UUID
	actorID           uuid.UUID
	triggerUserID     uuid.UUID
	suffix            string
	mblID             uuid.UUID
	executionID       uuid.UUID
	cashflowID        uuid.UUID
	orderBills        map[uuid.UUID]uuid.UUID
	actorUsdAccountID uuid.UUID
}

func newSettlementOrderLockFixture(t *testing.T, data *Data) *settlementOrderLockFixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:10]
	// 公司级根组织 + 本币：核销用例在事务内经汇率上下文解析公司本币，系统节点
	// 组织无法解析本币（与 newFeeSupplementFixture 修正同源）。
	organization, err := data.db.Organization.Create().
		SetCode("ALOCK-" + suffix).
		SetName("自动锁定集成测试组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	partner, err := data.db.Partner.Create().
		SetOrganizationID(organization.ID).
		SetCode("CUST-" + suffix).
		SetLegalName("自动锁定测试客户-" + suffix).
		SetNormalizedName("自动锁定测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	// 触发人：无任何角色与锁权限，自动锁定不得复用其权限或归责其锁定。
	triggerUser, err := data.db.User.Create().
		SetDisplayName("核销操作员-" + suffix).
		SetEnabled(true).
		SetIsBootstrapAdmin(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建触发用户: %v", err)
	}
	fixture := &settlementOrderLockFixture{
		t: t, data: data, ctx: ctx,
		organizationID: organization.ID, partnerID: partner.ID,
		actorID: uuid.New(), triggerUserID: triggerUser.ID,
		suffix:     suffix,
		orderBills: map[uuid.UUID]uuid.UUID{},
	}
	t.Cleanup(fixture.cleanup)

	carrier, err := data.db.ShippingLine.Create().
		SetScacCode(scacLettersFromUUID()).
		SetNameZh("自动锁定船公司-" + suffix).
		SetNameEn("AutoLock-" + suffix).
		SetCountryCode("DK").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建船公司: %v", err)
	}
	etd := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)
	exec, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(organization.ID).
		SetShippingLineID(carrier.ID).
		SetVesselName("AUTO LOCK VESSEL").
		SetVoyageNo("2609W").
		SetEtd(etd).
		SetEta(etd.Add(7 * 24 * time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建运输执行: %v", err)
	}
	fixture.executionID = exec.ID
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(organization.ID).
		SetMasterNo("MBL-" + suffix).
		SetNormalizedMasterNo("MBL-" + suffix).
		SetShippingLineID(carrier.ID).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建 MBL: %v", err)
	}
	fixture.mblID = mbl.ID
	return fixture
}

// usdAccountOrCreate 首次调用时创建测试结算账户并缓存账户 ID。
func (f *settlementOrderLockFixture) usdAccountOrCreate() uuid.UUID {
	if f.actorUsdAccountID != uuid.Nil {
		return f.actorUsdAccountID
	}
	account, err := f.data.db.PartnerAccount.Create().
		SetPartnerID(f.partnerID).
		SetName("自动锁定测试结算账户-" + f.suffix).
		SetAccountHolder(f.partnerID.String()).
		SetBankName("集成测试银行").
		SetAccountNo("AL-" + f.suffix).
		SetCurrency("USD").
		SetUsage(partneraccountent.UsageBOTH).
		SetEnabled(true).
		SetIsDefaultReceivable(true).
		SetIsDefaultPayable(true).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建测试结算账户: %v", err)
	}
	f.actorUsdAccountID = account.ID
	return account.ID
}

// createSEOrder 创建挂在共享 MBL 上的 SE 成员订单（活动 Link + HBL）。
func (f *settlementOrderLockFixture) createSEOrder(orderNo string) *ent.Order {
	f.t.Helper()
	order := f.createPlainOrder(orderNo, orderent.BusinessTypeSE)
	if _, err := f.data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(f.organizationID).
		SetOrderID(order.ID).
		SetMasterBillID(f.mblID).
		SetTransportExecutionID(f.executionID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(f.ctx); err != nil {
		f.t.Fatalf("关联订单 %s 到 MBL: %v", orderNo, err)
	}
	if _, err := f.data.db.SeaHouseBill.Create().
		SetOrganizationID(f.organizationID).
		SetOrderID(order.ID).
		SetMasterBillID(f.mblID).
		SetHouseNo("HBL-" + orderNo).
		SetNormalizedHouseNo("HBL-" + orderNo).
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(f.organizationID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(f.ctx); err != nil {
		f.t.Fatalf("创建订单 %s 的 HBL: %v", orderNo, err)
	}
	return order
}

// createPlainOrder 创建不关联海运单证的订单（非 SE 业务类型）。
func (f *settlementOrderLockFixture) createPlainOrder(orderNo string, businessType orderent.BusinessType) *ent.Order {
	f.t.Helper()
	order, err := f.data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(f.organizationID).
		SetOrderNo(orderNo).
		SetCustomerID(f.partnerID).
		SetBusinessType(businessType).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建订单 %s: %v", orderNo, err)
	}
	return order
}

// createReceivableBill 为订单集合创建一张已确认应收账单：账单总额等于全部
// 账单行合计，每张订单挂一条等额账单行。返回账单 ID。
func (f *settlementOrderLockFixture) createReceivableBill(orders []*ent.Order, lineAmount string) uuid.UUID {
	f.t.Helper()
	amounts := make([]string, len(orders))
	for i := range amounts {
		amounts[i] = lineAmount
	}
	return f.createReceivableBillAmounts(orders, amounts, "USD")
}

// createReceivableBillAmounts 支持不同订单行金额与币种，整单金额始终等于行合计。
func (f *settlementOrderLockFixture) createReceivableBillAmounts(orders []*ent.Order, amounts []string, currency string) uuid.UUID {
	f.t.Helper()
	if len(orders) != len(amounts) {
		f.t.Fatal("订单与费用金额数量不一致")
	}
	total := decimal.Zero
	for _, amount := range amounts {
		total = total.Add(decimal.RequireFromString(amount))
	}
	accountID := f.usdAccountOrCreate()
	if currency != "USD" {
		account, err := f.data.db.PartnerAccount.Create().
			SetPartnerID(f.partnerID).
			SetName("自动锁定测试结算账户-" + currency + "-" + f.suffix).
			SetAccountHolder(f.partnerID.String()).
			SetBankName("集成测试银行").
			SetAccountNo("AL-" + currency + "-" + uuid.NewString()[:8]).
			SetCurrency(currency).
			SetUsage(partneraccountent.UsageBOTH).
			SetEnabled(true).
			Save(f.ctx)
		if err != nil {
			f.t.Fatalf("创建 %s 结算账户: %v", currency, err)
		}
		accountID = account.ID
	}
	billNo := "BILL-AL-" + uuid.NewString()[:8]
	billCreate := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo(billNo).
		SetIdempotencyKey("bill-" + billNo).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("自动锁定测试客户").
		SetCurrency(currency).
		SetBaseCurrency("CNY").
		SetExchangeRate("7.20000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount(total.StringFixed(8)).
		SetNetAmount(total.StringFixed(8)).
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount(total.Mul(decimal.RequireFromString("7.2")).StringFixed(8)).
		SetFeeCount(len(orders)).
		SetBillDate(financeBillIntegrationDate).
		SetVersion(1)
	bill, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, accountID, currency).Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建应收账单 %s: %v", billNo, err)
	}
	for i, order := range orders {
		lineAmount := amounts[i]
		fee, feeErr := f.data.db.OrderFee.Create().
			SetOrderID(order.ID).
			SetIdempotencyKey("fee-" + order.OrderNo + "-" + uuid.NewString()[:8]).
			SetDirection(orderfeeent.DirectionRECEIVABLE).
			SetFeeCode("OCEAN_FREIGHT").
			SetFeeName("海运费").
			SetSettlementPartyID(f.partnerID).
			SetBillingUnit("票").
			SetQuantity("1.0000").
			SetUnitPrice(lineAmount).
			SetTotalAmount(lineAmount).
			SetNetAmount(lineAmount).
			SetTaxAmount("0.00000000").
			SetCurrency(currency).
			SetExchangeRate("7.20000000").
			SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate(financeBillIntegrationDate).
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount(decimal.RequireFromString(lineAmount).Mul(decimal.RequireFromString("7.2")).StringFixed(8)).
			SetExpenseDate(financeBillIntegrationDate).
			SetVersion(1).
			Save(f.ctx)
		if feeErr != nil {
			f.t.Fatalf("创建订单 %s 应收费用: %v", order.OrderNo, feeErr)
		}
		if _, lineErr := f.data.db.FinanceBillLine.Create().
			SetBillID(bill.ID).
			SetOrderFeeID(fee.ID).
			SetOrderID(order.ID).
			SetOrderNo(order.OrderNo).
			SetFeeCode("OCEAN_FREIGHT").
			SetFeeName("海运费").
			SetQuantity("1.0000").
			SetUnitPrice(lineAmount).
			SetTotalAmount(lineAmount).
			SetNetAmount(lineAmount).
			SetTaxAmount("0.00000000").
			SetCurrency(currency).
			SetExchangeRate("7.20000000").
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount(decimal.RequireFromString(lineAmount).Mul(decimal.RequireFromString("7.2")).StringFixed(8)).
			Save(f.ctx); lineErr != nil {
			f.t.Fatalf("创建订单 %s 账单行: %v", order.OrderNo, lineErr)
		}
		f.orderBills[order.ID] = bill.ID
	}
	return bill.ID
}

// settleOnBill 通过有效核销分摊为指定账单结清指定金额。
func (f *settlementOrderLockFixture) settleOnBill(billID uuid.UUID, amount string) uuid.UUID {
	f.t.Helper()
	bill, err := f.data.db.FinanceBill.Get(f.ctx, billID)
	if err != nil {
		f.t.Fatalf("查询核销账单: %v", err)
	}
	verificationID := uuid.Must(uuid.NewV7())
	if _, err := f.data.db.FinanceVerification.Create().
		SetID(verificationID).
		SetOrganizationID(f.organizationID).
		SetVerificationNo("WO-" + f.suffix + "-" + verificationID.String()).
		SetIdempotencyKey("ver-alock-" + verificationID.String()).
		SetStatus(financeverificationent.StatusACTIVE).
		SetDirection(financeverificationent.DirectionRECEIVABLE).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("自动锁定测试客户").
		SetCurrency(bill.Currency).
		SetAmount(amount).
		SetBaseCurrency("CNY").
		SetBaseAmount("720.00000000").
		SetBillBaseAmount("720.00000000").
		SetCashflowBaseAmount("725.00000000").
		SetExchangeGainLoss("5.00000000").
		SetVerificationDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(f.ctx); err != nil {
		f.t.Fatalf("创建核销单: %v", err)
	}
	if _, err := f.data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationID).
		SetCashflowID(f.cashflowForCurrency(bill.Currency)).
		SetBillID(billID).
		SetCashflowNo("FLOW-AL").
		SetBillNo("BILL-AL-" + f.suffix).
		SetAmount(amount).
		SetBillBaseAmount("720.00000000").
		SetCashflowBaseAmount("725.00000000").
		SetExchangeGainLoss("5.00000000").
		SetActive(true).
		Save(f.ctx); err != nil {
		f.t.Fatalf("创建核销分摊: %v", err)
	}
	return verificationID
}

// settle 通过订单专属账单结清指定金额并返回核销单 ID。
func (f *settlementOrderLockFixture) settle(order *ent.Order, amount string) uuid.UUID {
	f.t.Helper()
	billID, ok := f.orderBills[order.ID]
	if !ok {
		f.t.Fatalf("订单 %s 尚未创建应收账单", order.OrderNo)
	}
	return f.settleOnBill(billID, amount)
}

func (f *settlementOrderLockFixture) cashflowOrCreate() uuid.UUID {
	return f.cashflowForCurrency("USD")
}

func (f *settlementOrderLockFixture) cashflowForCurrency(currency string) uuid.UUID {
	if currency == "USD" && f.cashflowID != uuid.Nil {
		return f.cashflowID
	}
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(f.organizationID).
		SetFlowNo("FLOW-AL-" + uuid.NewString()).
		SetIdempotencyKey("cashflow-alock-" + uuid.NewString()).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("自动锁定测试客户").
		SetCurrency(currency).
		SetAmount("100000.00000000").
		SetExchangeRate("7.25000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("725000.00000000").
		SetTransactionDate(financeBillIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建资金流水: %v", err)
	}
	if currency == "USD" {
		f.cashflowID = cashflow.ID
	}
	return cashflow.ID
}

func (f *settlementOrderLockFixture) reload(orderID uuid.UUID) *ent.Order {
	f.t.Helper()
	order, err := f.data.db.Order.Get(f.ctx, orderID)
	if err != nil {
		f.t.Fatalf("重读订单: %v", err)
	}
	return order
}

func (f *settlementOrderLockFixture) lockRecords(orderID uuid.UUID) []*ent.OrderLockRecord {
	f.t.Helper()
	records, err := f.data.db.OrderLockRecord.Query().
		Where(orderlockrecordent.OrderIDEQ(orderID)).
		All(f.ctx)
	if err != nil {
		f.t.Fatalf("查询锁定记录: %v", err)
	}
	return records
}

func (f *settlementOrderLockFixture) cleanup() {
	orgStatements := []string{
		`DELETE FROM audit_logs WHERE organization_id = $1`,
		`DELETE FROM order_lock_house_bill_snapshots WHERE organization_id = $1`,
		`DELETE FROM order_lock_records WHERE organization_id = $1`,
		`UPDATE sea_master_bills SET current_version_id = NULL WHERE organization_id = $1`,
		`UPDATE sea_house_bills SET current_version_id = NULL WHERE organization_id = $1`,
		`DELETE FROM sea_house_bill_versions WHERE organization_id = $1`,
		`DELETE FROM sea_master_bill_versions WHERE organization_id = $1`,
		`DELETE FROM sea_house_bills WHERE organization_id = $1`,
		`DELETE FROM sea_master_bill_order_links WHERE organization_id = $1`,
		`DELETE FROM sea_master_bills WHERE organization_id = $1`,
		`DELETE FROM sea_transport_execution_versions WHERE organization_id = $1`,
		`DELETE FROM sea_transport_executions WHERE organization_id = $1`,
		`DELETE FROM finance_verification_allocations WHERE verification_id IN (SELECT id FROM finance_verifications WHERE organization_id = $1)`,
		`DELETE FROM finance_verifications WHERE organization_id = $1`,
		`DELETE FROM finance_cashflows WHERE organization_id = $1`,
		`DELETE FROM finance_netting_allocations WHERE netting_id IN (SELECT id FROM finance_nettings WHERE organization_id = $1)`,
		`DELETE FROM finance_nettings WHERE organization_id = $1`,
		`DELETE FROM finance_bill_lines WHERE bill_id IN (SELECT id FROM finance_bills WHERE organization_id = $1)`,
		`DELETE FROM finance_bills WHERE organization_id = $1`,
		`DELETE FROM number_sequences WHERE rule_id IN (SELECT id FROM number_rules WHERE organization_id = $1)`,
		`DELETE FROM number_rules WHERE organization_id = $1`,
		`DELETE FROM order_fees WHERE order_id IN (SELECT id FROM orders WHERE organization_id = $1)`,
		`DELETE FROM orders WHERE organization_id = $1`,
	}
	for _, stmt := range orgStatements {
		if _, err := f.data.sqlDB.ExecContext(f.ctx, stmt, f.organizationID); err != nil {
			f.t.Errorf("清理自动锁定测试夹具失败: %v", err)
		}
	}
	for _, stmt := range []string{
		`DELETE FROM partner_accounts WHERE partner_id IN (SELECT id FROM partners WHERE organization_id = $1)`,
		`DELETE FROM partners WHERE organization_id = $1`,
	} {
		if _, err := f.data.sqlDB.ExecContext(f.ctx, stmt, f.organizationID); err != nil {
			f.t.Errorf("清理自动锁定测试夹具失败: %v", err)
		}
	}
	if _, err := f.data.sqlDB.ExecContext(f.ctx, `DELETE FROM shipping_lines WHERE name_en = $1`, "AutoLock-"+f.suffix); err != nil {
		f.t.Errorf("清理自动锁定测试夹具失败: %v", err)
	}
	if _, err := f.data.sqlDB.ExecContext(f.ctx, `DELETE FROM users WHERE id = $1`, f.triggerUserID); err != nil {
		f.t.Errorf("清理自动锁定测试夹具失败: %v", err)
	}
	if _, err := f.data.sqlDB.ExecContext(f.ctx, `DELETE FROM organizations WHERE id = $1`, f.organizationID); err != nil {
		f.t.Errorf("清理自动锁定测试夹具失败: %v", err)
	}
}

// scacLettersFromUUID 从 UUID 十六进制字符确定性派生 4 个大写字母的 SCAC 码。
func scacLettersFromUUID() string {
	value := uuid.NewString()
	out := make([]rune, 0, 4)
	for _, c := range value {
		if len(out) == 4 {
			break
		}
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			var digit int
			if c >= 'a' {
				digit = int(c-'a') + 10
			} else {
				digit = int(c - '0')
			}
			out = append(out, rune('A'+digit))
		}
	}
	return string(out)
}

// 通过真实核销用例完成整张单据结清，验证结清不会伪造人工审核或锁单事实。
func TestSettlementDoesNotReviewOrLockOrdersPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	for _, shared := range []bool{false, true} {
		t.Run(map[bool]string{false: "跨订单账单", true: "共享主单成员"}[shared], func(t *testing.T) {
			f := newSettlementOrderLockFixture(t, data)
			create := func(suffix string) *ent.Order {
				if shared {
					return f.createSEOrder("SE-" + f.suffix + suffix)
				}
				return f.createPlainOrder("AI-"+f.suffix+suffix, orderent.BusinessTypeAI)
			}
			a, b := create("A"), create("B")
			billID := f.createReceivableBill([]*ent.Order{a, b}, "100")
			cashID := f.cashflowOrCreate()
			if _, err := data.db.NumberRule.Create().SetOrganizationID(f.organizationID).SetDocumentType(numberruleent.DocumentTypeWriteOff).SetPrefix("WO-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(f.ctx); err != nil {
				t.Fatal(err)
			}
			uc := biz.NewVerificationUsecase(NewVerificationRepo(data), biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), nil), data, nil)
			verification, err := uc.Create(f.ctx, f.organizationID, f.triggerUserID, biz.CreateVerificationInput{Allocations: []*biz.VerificationAllocation{{CashflowID: cashID, BillID: billID, Amount: decimal.NewFromInt(200)}}, VerificationDate: financeBillIntegrationDate, IdempotencyKey: "settlement-no-lock-" + f.suffix})
			if err != nil || verification == nil {
				t.Fatalf("完整核销失败：%#v %v", verification, err)
			}
			for _, before := range []*ent.Order{a, b} {
				current := f.reload(before.ID)
				if current.LockedAt != nil || current.LockedBy != nil || current.LockGeneration != before.LockGeneration || current.Version != before.Version || current.ApprovedFeeReviewEventID != nil || len(f.lockRecords(before.ID)) != 0 {
					t.Fatalf("结清改变了订单人工审核事实：%#v", current)
				}
			}
		})
	}
}
