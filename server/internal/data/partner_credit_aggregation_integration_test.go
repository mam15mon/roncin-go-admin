package data

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	financeverificationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	partnersettlementruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
	"github.com/shopspring/decimal"
)

// creditPartnerFixture 组装客户信用聚合集成测试的组织、往来户与角色规则。
type creditPartnerFixture struct {
	t              *testing.T
	data           *Data
	repo           *financeBillRepo
	organizationID uuid.UUID
	suffix         string
}

func newCreditPartnerFixture(t *testing.T, data *Data) *creditPartnerFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("CREDIT-" + suffix).
		SetName("信用聚合集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	return &creditPartnerFixture{t: t, data: data, repo: NewFinanceBillRepo(data).(*financeBillRepo), organizationID: organization.ID, suffix: suffix}
}

// newCreditPartner 创建带客户角色的往来户，返回 (partnerID, roleID)。
func (f *creditPartnerFixture) newCreditPartner(name string) (uuid.UUID, uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	partner, err := f.data.db.Partner.Create().
		SetOrganizationID(f.organizationID).
		SetCode(name + "-" + f.suffix).
		SetLegalName(name + "-" + f.suffix).
		SetNormalizedName(name + "-" + f.suffix).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试往来单位 %s: %v", name, err)
	}
	role, err := f.data.db.PartnerRole.Create().
		SetPartnerID(partner.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试客户角色 %s: %v", name, err)
	}
	return partner.ID, role.ID
}

func (f *creditPartnerFixture) newSettlementRule(roleID uuid.UUID, method partnersettlementruleent.SettlementMethod, creditLimitMinor *int64, creditCurrency *string, paymentTermsDays *int, active bool) {
	f.t.Helper()
	ctx := context.Background()
	builder := f.data.db.PartnerSettlementRule.Create().
		SetPartnerRoleID(roleID).
		SetStatementMode(partnersettlementruleent.StatementModeSingle).
		SetSettlementMethod(method).
		SetSettlementCurrency("CNY").
		SetIsActive(active)
	if method == partnersettlementruleent.SettlementMethodMonthly {
		day := 15
		builder = builder.SetSettlementDay(day).SetSettlementBase(partnersettlementruleent.SettlementBaseBillDate)
	}
	if creditLimitMinor != nil {
		builder = builder.SetCreditLimitMinor(*creditLimitMinor).SetCreditCurrency(*creditCurrency)
	}
	if paymentTermsDays != nil {
		builder = builder.SetPaymentTermsDays(*paymentTermsDays)
	}
	if _, err := builder.Save(ctx); err != nil {
		f.t.Fatalf("创建测试结算规则: %v", err)
	}
}

// newConfirmedBill 创建指定方向与状态的账单（金额为折本币总额）。
func (f *creditPartnerFixture) newConfirmedBill(key string, partyID uuid.UUID, partyName string, direction financebillent.Direction, status financebillent.Status, amount string) *ent.FinanceBill {
	f.t.Helper()
	ctx := context.Background()
	bill := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo("BILL-" + key + "-" + f.suffix).
		SetIdempotencyKey("bill-credit-" + key + "-" + f.suffix).
		SetDirection(direction).
		SetStatus(status).
		SetSettlementPartyID(partyID).
		SetSettlementPartyName(partyName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount(amount).
		SetNetAmount(amount).
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount(amount).
		SetFeeCount(1).
		SetBillDate(financeBillIntegrationDate).
		SetVersion(1)
	saved, err := withTestFinanceBillSettlementAccountSnapshot(bill, partyID, "CNY").Save(ctx)
	if err != nil {
		f.t.Fatalf("保存测试账单 %s: %v", key, err)
	}
	return saved
}

// settleBill 通过真实核销 / 核销分摊记录核销账单金额，验证未结清口径排除已结清账单。
func (f *creditPartnerFixture) settleBill(key string, bill *ent.FinanceBill, amount string) {
	f.t.Helper()
	ctx := context.Background()
	amount8 := decimal.RequireFromString(amount).StringFixed(8)
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(f.organizationID).
		SetFlowNo("FLOW-" + key + "-" + f.suffix).
		SetIdempotencyKey("cashflow-credit-" + key + "-" + f.suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(bill.SettlementPartyID).
		SetSettlementPartyName(bill.SettlementPartyName).
		SetCurrency("CNY").
		SetAmount(amount8).
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount(amount8).
		SetTransactionDate(financeBillIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建信用测试资金流水 %s: %v", key, err)
	}
	verification, err := f.data.db.FinanceVerification.Create().
		SetOrganizationID(f.organizationID).
		SetVerificationNo("WO-" + key + "-" + f.suffix).
		SetIdempotencyKey("verification-credit-" + key + "-" + f.suffix).
		SetStatus(financeverificationent.StatusACTIVE).
		SetDirection(financeverificationent.DirectionRECEIVABLE).
		SetSettlementPartyID(bill.SettlementPartyID).
		SetSettlementPartyName(bill.SettlementPartyName).
		SetCurrency("CNY").
		SetAmount(amount8).
		SetBaseCurrency("CNY").
		SetBaseAmount(amount8).
		SetBillBaseAmount(amount8).
		SetCashflowBaseAmount(amount8).
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建信用测试核销单 %s: %v", key, err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verification.ID).
		SetCashflowID(cashflow.ID).
		SetBillID(bill.ID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo(bill.BillNo).
		SetAmount(amount8).
		SetBillBaseAmount(amount8).
		SetCashflowBaseAmount(amount8).
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建信用测试核销分摊 %s: %v", key, err)
	}
}

func TestPartnerCreditAggregationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	fixture := newCreditPartnerFixture(t, data)
	ctx := context.Background()

	// 客户 A：两条激活规则（取最大额度 2000 与最大账期 45）+ 一条停用规则（不参与）。
	partnerA, roleA := fixture.newCreditPartner("信用客户A")
	creditA := int64(200000)
	creditASmall := int64(100000)
	creditCurrency := "CNY"
	termsA := 45
	termsASmall := 30
	hugeLimit := int64(99999999)
	fixture.newSettlementRule(roleA, partnersettlementruleent.SettlementMethodMonthly, &creditASmall, &creditCurrency, &termsASmall, true)
	fixture.newSettlementRule(roleA, partnersettlementruleent.SettlementMethodByTicket, &creditA, &creditCurrency, &termsA, true)
	fixture.newSettlementRule(roleA, partnersettlementruleent.SettlementMethodWeekly, &hugeLimit, &creditCurrency, nil, false)

	// 已结清 800 + 部分核销 500（余额 400）参与聚合；草稿与应付不参与。
	settledBill := fixture.newConfirmedBill("settled-a", partnerA, "信用客户A", financebillent.DirectionRECEIVABLE, financebillent.StatusCONFIRMED, "800.00000000")
	fixture.settleBill("settled-a", settledBill, "800.00000000")
	partialBill := fixture.newConfirmedBill("partial-a", partnerA, "信用客户A", financebillent.DirectionRECEIVABLE, financebillent.StatusCONFIRMED, "500.00000000")
	fixture.settleBill("partial-a", partialBill, "100.00000000")
	fixture.newConfirmedBill("draft-a", partnerA, "信用客户A", financebillent.DirectionRECEIVABLE, financebillent.StatusDRAFT, "5000.00000000")
	fixture.newConfirmedBill("payable-a", partnerA, "信用客户A", financebillent.DirectionPAYABLE, financebillent.StatusCONFIRMED, "300.00000000")

	// 客户 B：额度 100、余额 150，判定超额。
	partnerB, roleB := fixture.newCreditPartner("信用客户B")
	creditB := int64(10000)
	fixture.newSettlementRule(roleB, partnersettlementruleent.SettlementMethodByTicket, &creditB, &creditCurrency, nil, true)
	fixture.newConfirmedBill("bill-b", partnerB, "信用客户B", financebillent.DirectionRECEIVABLE, financebillent.StatusCONFIRMED, "150.00000000")

	// 客户 C：无客户角色规则，不参与超额判定。
	partnerC, _ := fixture.newCreditPartner("信用客户C")
	fixture.newConfirmedBill("bill-c", partnerC, "信用客户C", financebillent.DirectionRECEIVABLE, financebillent.StatusCONFIRMED, "500.00000000")

	amountA, err := fixture.repo.GetPartnerUnsettledReceivableBaseAmount(ctx, fixture.organizationID, partnerA)
	if err != nil {
		t.Fatalf("读取客户 A 未核销应收总额: %v", err)
	}
	if !amountA.Equal(decimal.RequireFromString("400")) {
		t.Fatalf("客户 A 未核销应收总额应为 400（500-100，已结清/草稿/应付不参与），实际 %s", amountA)
	}

	summaries, err := fixture.repo.GetPartnerCreditSummaries(ctx, fixture.organizationID, []uuid.UUID{partnerA, partnerB, partnerC})
	if err != nil {
		t.Fatalf("批量读取客户信用摘要: %v", err)
	}
	summaryA := summaries[partnerA]
	if summaryA == nil {
		t.Fatalf("客户 A 应有信用摘要")
	}
	if summaryA.CreditLimitBase == nil || !summaryA.CreditLimitBase.Equal(decimal.NewFromInt(2000)) {
		t.Fatalf("客户 A 应取激活规则的最大额度 2000，实际 %v", summaryA.CreditLimitBase)
	}
	if summaryA.DefaultPaymentTermsDays == nil || *summaryA.DefaultPaymentTermsDays != termsA {
		t.Fatalf("客户 A 应取激活规则的最大默认账期 45，实际 %v", summaryA.DefaultPaymentTermsDays)
	}
	if summaryA.CreditCurrency == nil || *summaryA.CreditCurrency != "CNY" {
		t.Fatalf("客户 A 信用币种应为 CNY，实际 %v", summaryA.CreditCurrency)
	}
	if !summaryA.UnsettledReceivableBase.Equal(decimal.RequireFromString("400")) {
		t.Fatalf("客户 A 摘要余额应为 400，实际 %s", summaryA.UnsettledReceivableBase)
	}
	if biz.PartnerCreditExceeded(summaryA) {
		t.Fatalf("客户 A 余额 400 未超额度 2000，不应判定超额")
	}

	summaryB := summaries[partnerB]
	if summaryB == nil || summaryB.CreditLimitBase == nil || !summaryB.CreditLimitBase.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("客户 B 额度应为 100，实际 %+v", summaryB)
	}
	if !biz.PartnerCreditExceeded(summaryB) {
		t.Fatalf("客户 B 余额 150 超过额度 100，应判定超额")
	}

	summaryC := summaries[partnerC]
	if summaryC == nil {
		t.Fatalf("客户 C 无规则也应返回余额摘要")
	}
	if summaryC.CreditLimitBase != nil {
		t.Fatalf("客户 C 未配置额度，CreditLimitBase 应为空")
	}
	if biz.PartnerCreditExceeded(summaryC) {
		t.Fatalf("客户 C 未配置额度，不应判定超额")
	}

	amountC, err := fixture.repo.GetPartnerUnsettledReceivableBaseAmount(ctx, fixture.organizationID, partnerC)
	if err != nil {
		t.Fatalf("读取客户 C 未核销应收总额: %v", err)
	}
	if !amountC.Equal(decimal.RequireFromString("500")) {
		t.Fatalf("无规则客户的余额查询仍应返回真实余额 500，实际 %s", amountC)
	}
}
