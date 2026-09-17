package data

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	financeverificationallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverificationallocation"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// autoLockPostgresFixture 结清自动锁定集成测试夹具：独立组织 + SE 订单 +
// 共享 MBL 组 + 应收账单/核销事实。
type autoLockPostgresFixture struct {
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
	billID            uuid.UUID
	cashflowID        uuid.UUID
	actorUsdAccountID uuid.UUID
	orders            map[string]*ent.Order
	repo              biz.AutoOrderLockRepo
}

func newAutoLockPostgresFixture(t *testing.T, data *Data) *autoLockPostgresFixture {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:10]
	organization, err := data.db.Organization.Create().
		SetCode("ALOCK-" + suffix).
		SetName("自动锁定集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
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
	fixture := &autoLockPostgresFixture{
		t: t, data: data, ctx: ctx,
		organizationID: organization.ID, partnerID: partner.ID,
		actorID: uuid.New(), triggerUserID: triggerUser.ID,
		suffix: suffix,
		orders: map[string]*ent.Order{},
		repo:   NewAutoOrderLockRepo(data),
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

	receivableBill := data.db.FinanceBill.Create().
		SetOrganizationID(organization.ID).
		SetBillNo("BILL-AL-" + suffix).
		SetIdempotencyKey("bill-alock-" + suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(partner.ID).
		SetSettlementPartyName(partner.LegalName).
		SetCurrency("USD").
		SetBaseCurrency("CNY").
		SetExchangeRate("7.20000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("720.00000000").
		SetFeeCount(1).
		SetBillDate(financeBillIntegrationDate).
		SetVersion(1)
	bill, err := withTestFinanceBillSettlementAccountSnapshot(receivableBill, fixture.usdAccountOrCreate(), "USD").Save(ctx)
	if err != nil {
		t.Fatalf("创建应收账单: %v", err)
	}
	fixture.billID = bill.ID
	return fixture
}

// usdAccountOrCreate 首次调用时创建测试结算账户并缓存账户 ID。
func (f *autoLockPostgresFixture) usdAccountOrCreate() uuid.UUID {
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

// createGroupOrder 创建 SE 成员订单、活动 Link 与 HBL，并挂一张已确认应收费用。
func (f *autoLockPostgresFixture) createGroupOrder(orderNo string) *ent.Order {
	f.t.Helper()
	order, err := f.data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(f.organizationID).
		SetOrderNo(orderNo).
		SetCustomerID(f.partnerID).
		SetBusinessType(orderent.BusinessTypeSE).
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
	f.orders[orderNo] = order
	return order
}

// createReceivableFact 为订单挂已建账的应收费用账单行（账单总额 100 的一部分）。
func (f *autoLockPostgresFixture) createReceivableFact(order *ent.Order, amount string) {
	f.t.Helper()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fee-" + order.OrderNo + "-" + uuid.NewString()[:8]).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusBILLED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice(amount).
		SetTotalAmount(amount).
		SetNetAmount(amount).
		SetTaxAmount("0.00000000").
		SetCurrency("USD").
		SetExchangeRate("7.20000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("720.00000000").
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建订单 %s 应收费用: %v", order.OrderNo, err)
	}
	if _, err := f.data.db.FinanceBillLine.Create().
		SetBillID(f.billID).
		SetOrderFeeID(fee.ID).
		SetOrderID(order.ID).
		SetOrderNo(order.OrderNo).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetQuantity("1.0000").
		SetUnitPrice(amount).
		SetTotalAmount(amount).
		SetNetAmount(amount).
		SetTaxAmount("0.00000000").
		SetCurrency("USD").
		SetExchangeRate("7.20000000").
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("720.00000000").
		Save(f.ctx); err != nil {
		f.t.Fatalf("创建订单 %s 账单行: %v", order.OrderNo, err)
	}
}

// settle 通过有效核销分摊为账单结清指定金额。
func (f *autoLockPostgresFixture) settle(amount string) uuid.UUID {
	f.t.Helper()
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
		SetCurrency("USD").
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
		SetCashflowID(f.cashflowOrCreate()).
		SetBillID(f.billID).
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

func (f *autoLockPostgresFixture) cashflowOrCreate() uuid.UUID {
	if f.cashflowID != uuid.Nil {
		return f.cashflowID
	}
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(f.organizationID).
		SetFlowNo("FLOW-AL-" + f.suffix).
		SetIdempotencyKey("cashflow-alock-" + f.suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("自动锁定测试客户").
		SetCurrency("USD").
		SetAmount("100.00000000").
		SetExchangeRate("7.25000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("725.00000000").
		SetTransactionDate(financeBillIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(f.ctx)
	if err != nil {
		f.t.Fatalf("创建资金流水: %v", err)
	}
	f.cashflowID = cashflow.ID
	return cashflow.ID
}

// triggerVerification 发出一次应收核销触发的自动锁定检查。
func (f *autoLockPostgresFixture) triggerVerification(verificationID uuid.UUID) error {
	return f.repo.RunAutoSettlementLockCheck(f.ctx, biz.AutoOrderLockTrigger{
		Type:           biz.AutoLockTriggerVerification,
		ResourceID:     verificationID,
		OrganizationID: f.organizationID,
		TriggeredBy:    f.triggerUserID,
	})
}

func (f *autoLockPostgresFixture) triggerFee(triggerType biz.AutoLockTriggerSource, order *ent.Order) error {
	return f.repo.RunAutoSettlementLockCheck(f.ctx, biz.AutoOrderLockTrigger{
		Type:           triggerType,
		ResourceID:     uuid.Must(uuid.NewV7()),
		OrganizationID: f.organizationID,
		TriggeredBy:    f.triggerUserID,
		OrderID:        order.ID,
	})
}

func (f *autoLockPostgresFixture) reload(orderID uuid.UUID) *ent.Order {
	f.t.Helper()
	order, err := f.data.db.Order.Get(f.ctx, orderID)
	if err != nil {
		f.t.Fatalf("重读订单: %v", err)
	}
	return order
}

func (f *autoLockPostgresFixture) lockRecords(orderID uuid.UUID) []*ent.OrderLockRecord {
	f.t.Helper()
	records, err := f.data.db.OrderLockRecord.Query().
		Where(orderlockrecordent.OrderIDEQ(orderID)).
		All(f.ctx)
	if err != nil {
		f.t.Fatalf("查询锁定记录: %v", err)
	}
	return records
}

func (f *autoLockPostgresFixture) auditReason(order *ent.Order) string {
	f.t.Helper()
	events, err := f.data.db.AuditLog.Query().
		Where(
			auditlogent.OrganizationIDEQ(f.organizationID),
			auditlogent.ActionEQ("order.auto_lock.check"),
			auditlogent.ResourceIDEQ(order.ID.String()),
		).
		Order(auditlogent.ByCreatedAt()).
		All(f.ctx)
	if err != nil {
		f.t.Fatalf("查询自动锁定审计: %v", err)
	}
	if len(events) == 0 {
		return ""
	}
	var details map[string]string
	if err := json.Unmarshal(events[len(events)-1].Details, &details); err != nil {
		f.t.Fatalf("解析自动锁定审计详情: %v", err)
	}
	return details["reason_code"]
}

func (f *autoLockPostgresFixture) cleanup() {
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

func TestAutoOrderLock_SettlementTriggerPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		source = "postgresql://roncin:roncin_local_dev@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable"
		t.Setenv("RONCIN_INTEGRATION_DATABASE_SOURCE", source)
	}
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("应收核销创建生效后系统自动锁定结清订单", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-A")
		fixture.createReceivableFact(order, "100.00000000")
		verificationID := fixture.settle("100.00000000")

		if err := fixture.triggerVerification(verificationID); err != nil {
			t.Fatalf("自动锁定检查失败: %v", err)
		}
		locked := fixture.reload(order.ID)
		if locked.LockedAt == nil {
			t.Fatal("结清订单未被自动锁定")
		}
		if locked.LockSource == nil || *locked.LockSource != orderent.LockSourceAUTO_SETTLEMENT {
			t.Fatalf("订单锁定来源 = %v，期望 AUTO_SETTLEMENT", locked.LockSource)
		}
		if locked.LockedBy != nil {
			t.Fatalf("自动锁定不得归属实际锁定人，locked_by = %s", *locked.LockedBy)
		}
		if locked.AutoLockTriggerType == nil || *locked.AutoLockTriggerType != orderent.AutoLockTriggerTypeVERIFICATION ||
			locked.AutoLockTriggerResourceID == nil || *locked.AutoLockTriggerResourceID != verificationID ||
			locked.AutoLockTriggeredBy == nil || *locked.AutoLockTriggeredBy != fixture.triggerUserID {
			t.Fatalf("订单自动触发审计字段不完整: %+v", locked)
		}
		if locked.LockGeneration != 1 || locked.Version != 2 {
			t.Fatalf("锁代次/版本 = %d/%d，期望 1/2", locked.LockGeneration, locked.Version)
		}

		records := fixture.lockRecords(order.ID)
		if len(records) != 1 {
			t.Fatalf("锁定记录数量 = %d，期望 1", len(records))
		}
		record := records[0]
		if record.LockSource != orderlockrecordent.LockSourceAUTO_SETTLEMENT || record.LockedBy != nil {
			t.Fatalf("锁定记录来源/锁定人 = %s/%v", record.LockSource, record.LockedBy)
		}
		if record.TriggerType == nil || *record.TriggerType != orderlockrecordent.TriggerTypeVERIFICATION ||
			record.TriggerResourceID == nil || *record.TriggerResourceID != verificationID ||
			record.TriggeredBy == nil || *record.TriggeredBy != fixture.triggerUserID {
			t.Fatalf("锁定记录触发审计字段不完整: %+v", record)
		}
		if record.MasterBillID == nil || *record.MasterBillID != fixture.mblID || record.MasterBillVersionID == nil {
			t.Fatalf("SE 自动锁定缺少 MBL 版本快照: %+v", record)
		}
		// 自动锁定新建的 MBL 不可变版本 created_by 必须为空。
		mblVersion, err := fixture.data.db.SeaMasterBillVersion.Get(fixture.ctx, *record.MasterBillVersionID)
		if err != nil {
			t.Fatalf("查询 MBL 版本: %v", err)
		}
		if mblVersion.CreatedBy != nil {
			t.Fatalf("自动锁定新建版本不得归责版本创建人: created_by = %s", *mblVersion.CreatedBy)
		}

		snapshots, err := fixture.data.db.OrderLockRecord.Query().
			Where(orderlockrecordent.IDEQ(record.ID)).
			QueryHouseBillSnapshots().
			All(fixture.ctx)
		if err != nil || len(snapshots) != 1 {
			t.Fatalf("HBL 快照数量 = %d (err=%v)，期望 1", len(snapshots), err)
		}

		// 锁状态投影返回自动来源与触发审计。
		stateRepo := NewOrderLockRepo(fixture.data, &conf.Security{})
		state, err := stateRepo.GetOrderLockState(fixture.ctx, fixture.organizationID, order.ID, nil)
		if err != nil {
			t.Fatalf("读取锁状态: %v", err)
		}
		if state.LockSource == nil || *state.LockSource != biz.LockSourceAutoSettlement {
			t.Fatalf("锁状态来源 = %v", state.LockSource)
		}
		if state.CurrentLockRecord == nil || state.CurrentLockRecord.TriggerType == nil ||
			*state.CurrentLockRecord.TriggerType != string(orderlockrecordent.TriggerTypeVERIFICATION) {
			t.Fatalf("锁状态未返回触发审计: %+v", state.CurrentLockRecord)
		}

		if reason := fixture.auditReason(order); reason != biz.AutoLockReasonLocked {
			t.Fatalf("审计原因码 = %q，期望 %s", reason, biz.AutoLockReasonLocked)
		}

		// 重复触发幂等收敛：不追加锁代次与锁定记录。
		if err := fixture.triggerVerification(verificationID); err != nil {
			t.Fatalf("重复触发失败: %v", err)
		}
		relocked := fixture.reload(order.ID)
		if relocked.LockGeneration != 1 || len(fixture.lockRecords(order.ID)) != 1 {
			t.Fatalf("重复触发追加了锁定事实: generation=%d records=%d", relocked.LockGeneration, len(fixture.lockRecords(order.ID)))
		}
	})

	t.Run("纯成本订单不因费用状态流转进入自动锁定", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-P")
		if err := fixture.triggerFee(biz.AutoLockTriggerFeeConfirm, order); err != nil {
			t.Fatalf("费用触发检查失败: %v", err)
		}
		if reloaded := fixture.reload(order.ID); reloaded.LockedAt != nil {
			t.Fatal("纯成本订单被错误锁定")
		}
		if reason := fixture.auditReason(order); reason != "" {
			t.Fatalf("无有效结清事实的订单不应写检查审计，得到 %q", reason)
		}
	})

	t.Run("费用草稿与未建账应收阻止自动锁定且费用确认后重试锁定", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-D")
		fixture.createReceivableFact(order, "100.00000000")
		verificationID := fixture.settle("100.00000000")

		// 应付方向费用草稿存在时阻止自动锁定。
		draftFee, err := fixture.data.db.OrderFee.Create().
			SetOrderID(order.ID).
			SetIdempotencyKey("fee-draft-" + fixture.suffix).
			SetDirection(orderfeeent.DirectionPAYABLE).
			SetStatus(orderfeeent.StatusDRAFT).
			SetFeeCode("TRUCKING").
			SetFeeName("拖车费").
			SetSettlementPartyID(fixture.partnerID).
			SetBillingUnit("票").
			SetQuantity("1.0000").
			SetUnitPrice("50.0000").
			SetTotalAmount("50.00000000").
			SetNetAmount("50.00000000").
			SetTaxAmount("0.00000000").
			SetCurrency("USD").
			SetExchangeRate("7.20000000").
			SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate(financeBillIntegrationDate).
			SetBaseCurrency("CNY").
			SetBaseCurrencyAmount("360.00000000").
			SetExpenseDate(financeBillIntegrationDate).
			SetVersion(1).
			Save(fixture.ctx)
		if err != nil {
			t.Fatalf("创建应付费用草稿: %v", err)
		}
		if err := fixture.triggerVerification(verificationID); err != nil {
			t.Fatalf("自动锁定检查失败: %v", err)
		}
		if reloaded := fixture.reload(order.ID); reloaded.LockedAt != nil {
			t.Fatal("存在费用草稿时订单被错误锁定")
		}
		if reason := fixture.auditReason(order); reason != biz.AutoLockReasonDraftFee {
			t.Fatalf("审计原因码 = %q，期望 %s", reason, biz.AutoLockReasonDraftFee)
		}

		// 费用草稿确认生效后，FEE_CONFIRM 事件重试并完成自动锁定。
		feeRepo := NewOrderFeeRepo(fixture.data)
		if _, err := feeRepo.Transition(fixture.ctx, fixture.organizationID, order.ID, draftFee.ID, fixture.triggerUserID, 1,
			biz.OrderFeeDraft, biz.OrderFeeConfirmed, nil, &biz.AuditEvent{Action: "order.fee.confirm", Result: "success", Details: map[string]string{}}); err != nil {
			t.Fatalf("确认费用草稿: %v", err)
		}
		if err := fixture.triggerFee(biz.AutoLockTriggerFeeConfirm, order); err != nil {
			t.Fatalf("费用确认触发检查失败: %v", err)
		}
		if reloaded := fixture.reload(order.ID); reloaded.LockedAt == nil {
			t.Fatal("费用草稿确认后订单未被自动锁定")
		}
	})

	t.Run("部分结清订单保持未锁直至全额结清", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-B")
		fixture.createReceivableFact(order, "100.00000000")
		firstVerification := fixture.settle("40.00000000")

		if err := fixture.triggerVerification(firstVerification); err != nil {
			t.Fatalf("自动锁定检查失败: %v", err)
		}
		if reloaded := fixture.reload(order.ID); reloaded.LockedAt != nil {
			t.Fatal("部分结清订单被错误锁定")
		}
		if reason := fixture.auditReason(order); reason != biz.AutoLockReasonUnsettledReceivable {
			t.Fatalf("审计原因码 = %q，期望 %s", reason, biz.AutoLockReasonUnsettledReceivable)
		}

		secondVerification := fixture.settle("60.00000000")
		if err := fixture.triggerVerification(secondVerification); err != nil {
			t.Fatalf("补足结清后触发失败: %v", err)
		}
		if reloaded := fixture.reload(order.ID); reloaded.LockedAt == nil {
			t.Fatal("全额结清后订单未被自动锁定")
		}
	})

	t.Run("核销用例创建生效后自动触发锁定", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-E2E")
		fixture.createReceivableFact(order, "100.00000000")
		if _, err := fixture.data.db.NumberRule.Create().
			SetOrganizationID(fixture.organizationID).
			SetDocumentType(numberruleent.DocumentTypeWriteOff).
			SetPrefix("WO-E2E-").
			SetDateFormat(numberruleent.DateFormatNone).
			SetSequenceLength(4).
			SetResetPolicy(numberruleent.ResetPolicyNever).
			SetEnabled(true).
			Save(fixture.ctx); err != nil {
			t.Fatalf("创建核销编号规则: %v", err)
		}
		usecase := biz.NewVerificationUsecase(
			NewVerificationRepo(fixture.data),
			biz.NewExchangeRateUsecase(NewExchangeRateRepo(fixture.data), nil),
			fixture.data,
			biz.NewAutoOrderLockUsecase(fixture.repo),
			nil,
		)
		created, err := usecase.Create(fixture.ctx, fixture.organizationID, fixture.triggerUserID, biz.CreateVerificationInput{
			Allocations:      []*biz.VerificationAllocation{{CashflowID: fixture.cashflowOrCreate(), BillID: fixture.billID, Amount: decimal.RequireFromString("100")}},
			VerificationDate: financeBillIntegrationDate,
			IdempotencyKey:   "ver-e2e-" + fixture.suffix,
		})
		if err != nil {
			t.Fatalf("创建核销: %v", err)
		}
		if created.Status != biz.VerificationActive {
			t.Fatalf("核销状态 = %s", created.Status)
		}
		// 核销事务成功提交后自动触发检查，订单无需任何显式锁单调用即被系统锁定。
		locked := fixture.reload(order.ID)
		if locked.LockedAt == nil || locked.LockSource == nil || *locked.LockSource != orderent.LockSourceAUTO_SETTLEMENT {
			t.Fatalf("核销生效后订单未被自动锁定: %+v", locked)
		}
	})

	t.Run("已反转核销不计入结清事实", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-R")
		fixture.createReceivableFact(order, "100.00000000")
		verificationID := fixture.settle("100.00000000")
		now := time.Now().UTC()
		if _, err := fixture.data.db.FinanceVerification.UpdateOneID(verificationID).
			SetStatus(financeverificationent.StatusREVERSED).
			SetReversedAt(now).
			SetReversedBy(fixture.triggerUserID).
			SetReversalReason("集成测试反转").
			SetVersion(2).
			Save(fixture.ctx); err != nil {
			t.Fatalf("反转核销单: %v", err)
		}
		if _, err := fixture.data.db.FinanceVerificationAllocation.Update().
			Where(financeverificationallocationent.VerificationIDEQ(verificationID)).
			SetActive(false).
			Save(fixture.ctx); err != nil {
			t.Fatalf("失效核销分摊: %v", err)
		}
		if err := fixture.triggerVerification(verificationID); err != nil {
			t.Fatalf("自动锁定检查失败: %v", err)
		}
		if reloaded := fixture.reload(order.ID); reloaded.LockedAt != nil {
			t.Fatal("仅有已反转核销的订单被错误锁定")
		}
	})
}

func TestAutoOrderLock_SharedMBLGroupPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		source = "postgresql://roncin:roncin_local_dev@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable"
		t.Setenv("RONCIN_INTEGRATION_DATABASE_SOURCE", source)
	}
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("任一成员未结清时整组不锁定且合格后整组原子锁定", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		orderA := fixture.createGroupOrder("SE-" + fixture.suffix + "-GA")
		orderB := fixture.createGroupOrder("SE-" + fixture.suffix + "-GB")
		fixture.createReceivableFact(orderA, "100.00000000")
		fixture.createReceivableFact(orderB, "100.00000000")
		// 共享账单仅部分结清：按订单行推导两个成员各自仍有未结清应收。
		firstVerification := fixture.settle("40.00000000")

		// 任一成员未结清：整组不锁。
		if err := fixture.triggerVerification(firstVerification); err != nil {
			t.Fatalf("组级检查失败: %v", err)
		}
		if reloaded := fixture.reload(orderA.ID); reloaded.LockedAt != nil {
			t.Fatal("组内成员未全部结清时 A 被抢先锁定")
		}
		if reloaded := fixture.reload(orderB.ID); reloaded.LockedAt != nil {
			t.Fatal("组内成员未全部结清时 B 被抢先锁定")
		}
		if reason := fixture.auditReason(orderA); reason != biz.AutoLockReasonMemberNotQualified {
			t.Fatalf("组级审计原因码 = %q，期望 %s", reason, biz.AutoLockReasonMemberNotQualified)
		}

		// 补足结清（第二次核销事件）：整组原子锁定并复用同一份 MBL/运输执行版本。
		secondVerification := fixture.settle("60.00000000")
		if err := fixture.triggerVerification(secondVerification); err != nil {
			t.Fatalf("组级检查失败: %v", err)
		}
		lockedA, lockedB := fixture.reload(orderA.ID), fixture.reload(orderB.ID)
		if lockedA.LockedAt == nil || lockedB.LockedAt == nil {
			t.Fatalf("整组锁定不完整: A=%v B=%v", lockedA.LockedAt != nil, lockedB.LockedAt != nil)
		}
		recordA, recordB := fixture.lockRecords(orderA.ID), fixture.lockRecords(orderB.ID)
		if len(recordA) != 1 || len(recordB) != 1 {
			t.Fatalf("成员锁定记录数量异常: A=%d B=%d", len(recordA), len(recordB))
		}
		if recordA[0].MasterBillVersionID == nil || *recordA[0].MasterBillVersionID != *recordB[0].MasterBillVersionID {
			t.Fatal("组级锁定未复用同一份 MBL 版本")
		}
		if recordA[0].TransportExecutionVersionID == nil || *recordA[0].TransportExecutionVersionID != *recordB[0].TransportExecutionVersionID {
			t.Fatal("组级锁定未复用同一份运输执行版本")
		}
		if recordA[0].Generation != 1 || recordB[0].Generation != 1 {
			t.Fatalf("组级锁定代次异常: A=%d B=%d", recordA[0].Generation, recordB[0].Generation)
		}
		// 各成员保留自己的 HBL 快照。
		for _, record := range []*ent.OrderLockRecord{recordA[0], recordB[0]} {
			snaps, err := fixture.data.db.OrderLockRecord.Query().
				Where(orderlockrecordent.IDEQ(record.ID)).
				QueryHouseBillSnapshots().
				All(fixture.ctx)
			if err != nil || len(snaps) != 1 {
				t.Fatalf("成员 HBL 快照数量 = %d (err=%v)，期望 1", len(snaps), err)
			}
		}
	})

	t.Run("已人工锁定的成员视为完成且剩余成员整组锁定", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		orderA := fixture.createGroupOrder("SE-" + fixture.suffix + "-MA")
		orderB := fixture.createGroupOrder("SE-" + fixture.suffix + "-MB")
		fixture.createReceivableFact(orderA, "100.00000000")
		fixture.createReceivableFact(orderB, "100.00000000")
		fixture.settle("100.00000000")
		fixture.settle("100.00000000")

		// 手动锁定 A（有 lock grant 的用户）。
		lockRepo := NewOrderLockRepo(fixture.data, &conf.Security{})
		locker := &biz.Principal{
			UserID:           fixture.triggerUserID,
			Organization:     biz.Organization{ID: fixture.organizationID},
			IsBootstrapAdmin: true,
		}
		if _, err := lockRepo.LockOrder(fixture.ctx, locker, orderA.ID, 1, "manual-"+fixture.suffix, nil); err != nil {
			t.Fatalf("手动锁定 A: %v", err)
		}
		verificationID := fixture.settle("100.00000000")
		if err := fixture.triggerVerification(verificationID); err != nil {
			t.Fatalf("组级检查失败: %v", err)
		}
		relockedA, lockedB := fixture.reload(orderA.ID), fixture.reload(orderB.ID)
		if relockedA.LockGeneration != 1 || len(fixture.lockRecords(orderA.ID)) != 1 {
			t.Fatalf("手动锁定的成员被重复处理: generation=%d records=%d", relockedA.LockGeneration, len(fixture.lockRecords(orderA.ID)))
		}
		if lockedB.LockedAt == nil {
			t.Fatal("剩余合格成员未被自动锁定")
		}
		manualRecord := fixture.lockRecords(orderA.ID)[0]
		if manualRecord.LockSource != orderlockrecordent.LockSourceMANUAL || manualRecord.LockedBy == nil {
			t.Fatalf("手动锁定记录来源异常: %s/%v", manualRecord.LockSource, manualRecord.LockedBy)
		}
		if recordB := fixture.lockRecords(orderB.ID)[0]; recordB.LockSource != orderlockrecordent.LockSourceAUTO_SETTLEMENT {
			t.Fatalf("自动锁定记录来源异常: %s", recordB.LockSource)
		}
	})
}

func TestAutoOrderLock_ReversalLinearizationPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		source = "postgresql://roncin:roncin_local_dev@127.0.0.1:5432/roncin_go_admin_integration?sslmode=disable"
		t.Setenv("RONCIN_INTEGRATION_DATABASE_SOURCE", source)
	}
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("自动锁定后反核销保持锁定且订单锁不阻止反转", func(t *testing.T) {
		fixture := newAutoLockPostgresFixture(t, data)
		order := fixture.createGroupOrder("SE-" + fixture.suffix + "-V")
		fixture.createReceivableFact(order, "100.00000000")
		verificationID := fixture.settle("100.00000000")
		if err := fixture.triggerVerification(verificationID); err != nil {
			t.Fatalf("自动锁定检查失败: %v", err)
		}
		before := fixture.reload(order.ID)
		originalRecord := fixture.lockRecords(order.ID)[0]

		verificationRepo := NewVerificationRepo(fixture.data)
		if _, err := verificationRepo.Reverse(fixture.ctx, fixture.organizationID, verificationID, fixture.triggerUserID, 1,
			"集成测试反核销", &biz.AuditEvent{Action: "finance.verification.reverse", Result: "success"}); err != nil {
			t.Fatalf("反核销被订单锁阻止: %v", err)
		}
		after := fixture.reload(order.ID)
		if after.LockedAt == nil || after.LockGeneration != before.LockGeneration || after.Version != before.Version {
			t.Fatalf("反核销后订单锁事实被改变: %+v -> %+v", before, after)
		}
		records := fixture.lockRecords(order.ID)
		if len(records) != 1 || records[0].ID != originalRecord.ID || records[0].UnlockedAt != nil {
			t.Fatalf("反核销后锁定记录被修改: %+v", records)
		}
	})
}
