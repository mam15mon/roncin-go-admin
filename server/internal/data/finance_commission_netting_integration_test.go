package data

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financecashflowent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecashflow"
	rule "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	verification "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeverification"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	attribution "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

// commissionNettingPostgresFixture 构造纯对冲结清的提成来源夹具：
// 两张应收账单（300/500，各挂一张订单）与一张应付账单（800）确认后全额对冲，
// 对冲金额 800 恰好覆盖双方可用余额，分摊结果与推导金额完全确定。
type commissionNettingPostgresFixture struct {
	t                 *testing.T
	data              *Data
	organizationID    uuid.UUID
	customerID        uuid.UUID
	employeeID        uuid.UUID
	actorID           uuid.UUID
	orderIDs          []uuid.UUID
	receivableFeeIDs  []uuid.UUID
	receivableBillIDs []uuid.UUID
	payableBillID     uuid.UUID
	nettingID         uuid.UUID
	ruleID            uuid.UUID
	suffix            string
}

var commissionNettingBillAmounts = [2]string{"300.00000000", "500.00000000"}

func newCommissionNettingPostgresFixture(t *testing.T) *commissionNettingPostgresFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]

	org, err := data.db.Organization.Create().
		SetCode("COMM-NT-" + suffix).
		SetName("对冲提成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	fixture := &commissionNettingPostgresFixture{t: t, data: data, organizationID: org.ID, suffix: suffix}

	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-NT-" + suffix).
		SetLegalName("对冲提成客户-" + suffix).
		SetNormalizedName("对冲提成客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试客户: %v", err)
	}
	fixture.customerID = customer.ID

	employee, err := data.db.User.Create().
		SetUsername("comm_nt_" + suffix).
		SetDisplayName("对冲提成业务员-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试员工: %v", err)
	}
	fixture.employeeID = employee.ID
	if _, err = data.db.Membership.Create().SetOrganizationID(org.ID).SetUserID(employee.ID).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试员工成员资格: %v", err)
	}

	actor, err := data.db.User.Create().
		SetUsername("comm_nt_actor_" + suffix).
		SetDisplayName("对冲提成操作员-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试操作员: %v", err)
	}
	fixture.actorID = actor.ID
	if _, err = data.db.Membership.Create().SetOrganizationID(org.ID).SetUserID(actor.ID).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试操作员成员资格: %v", err)
	}

	// 两张订单：O1 应收 300/应付 100，O2 应收 500/应付 200。
	orderPayables := [2]string{"100.00000000", "200.00000000"}
	for index, receivable := range commissionNettingBillAmounts {
		orderNo := "NT-SE" + suffix + "-" + strconv.Itoa(index)
		order, orderErr := data.db.Order.Create().
			SetIdempotencyKey(uuid.NewString()).
			SetOrganizationID(org.ID).
			SetOrderNo(orderNo).
			SetCustomerID(customer.ID).
			SetBusinessType(orderent.BusinessTypeSE).
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetOrderDate(financeCommissionIntegrationDate).
			Save(ctx)
		if orderErr != nil {
			t.Fatalf("创建测试订单 %d: %v", index, orderErr)
		}
		fixture.orderIDs = append(fixture.orderIDs, order.ID)
		if _, attrErr := data.db.OrderCommissionAttribution.Create().
			SetOrganizationID(org.ID).
			SetOrderID(order.ID).
			SetCustomerID(customer.ID).
			SetSourceAssignmentID(uuid.New()).
			SetEmployeeID(employee.ID).
			SetEmployeeName(employee.DisplayName).
			SetPersonnelRole(attribution.PersonnelRoleSALES).
			SetAttributedAt(time.Now()).
			Save(ctx); attrErr != nil {
			t.Fatalf("创建测试提成归属 %d: %v", index, attrErr)
		}
		for _, feeSpec := range []struct {
			direction fee.Direction
			amount    string
		}{{fee.DirectionRECEIVABLE, receivable}, {fee.DirectionPAYABLE, orderPayables[index]}} {
			createdFee, feeErr := data.db.OrderFee.Create().
				SetOrderID(order.ID).
				SetIdempotencyKey("nt-fee-" + suffix + "-" + strconv.Itoa(index) + "-" + string(feeSpec.direction)).
				SetDirection(feeSpec.direction).
				SetStatus(fee.StatusCONFIRMED).
				SetFeeCode("OCEAN_FREIGHT").
				SetFeeName("海运费").
				SetSettlementPartyID(customer.ID).
				SetBillingUnit("票").
				SetQuantity("1.0000").
				SetUnitPrice(strings.TrimSuffix(strings.TrimSuffix(feeSpec.amount, "0000"), ".")).
				SetTotalAmount(feeSpec.amount).
				SetNetAmount(feeSpec.amount).
				SetTaxAmount("0.00000000").
				SetCurrency("CNY").
				SetExchangeRate("1.00000000").
				SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).
				SetExchangeRateDate(financeCommissionIntegrationDate).
				SetBaseCurrency("CNY").
				SetBaseCurrencyAmount(feeSpec.amount).
				SetExpenseDate(financeCommissionIntegrationDate).
				SetVersion(1).
				Save(ctx)
			if feeErr != nil {
				t.Fatalf("创建测试费用 %d/%s: %v", index, feeSpec.direction, feeErr)
			}
			if feeSpec.direction == fee.DirectionRECEIVABLE {
				fixture.receivableFeeIDs = append(fixture.receivableFeeIDs, createdFee.ID)
			}
		}
	}

	// 应收账单 B1(300→O1)、B2(500→O2) 与应付账单 BP(800)。
	for index, amount := range commissionNettingBillAmounts {
		billCreate := data.db.FinanceBill.Create().
			SetOrganizationID(org.ID).
			SetBillNo("NT-BILL-R-" + suffix + "-" + strconv.Itoa(index)).
			SetIdempotencyKey("nt-bill-r-" + suffix + "-" + strconv.Itoa(index)).
			SetDirection(financebillent.DirectionRECEIVABLE).
			SetStatus(financebillent.StatusCONFIRMED).
			SetSettlementPartyID(customer.ID).
			SetSettlementPartyName(customer.LegalName).
			SetCurrency("CNY").
			SetBaseCurrency("CNY").
			SetExchangeRate("1.00000000").
			SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate(financeCommissionIntegrationDate).
			SetTotalAmount(amount).
			SetNetAmount(amount).
			SetTaxAmount("0.00000000").
			SetBaseCurrencyAmount(amount).
			SetFeeCount(1).
			SetBillDate(financeCommissionIntegrationDate).
			SetVersion(1)
		billItem, billErr := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
		if billErr != nil {
			t.Fatalf("创建测试应收账单 %d: %v", index, billErr)
		}
		fixture.receivableBillIDs = append(fixture.receivableBillIDs, billItem.ID)
		if _, lineErr := data.db.FinanceBillLine.Create().
			SetBillID(billItem.ID).
			SetOrderID(fixture.orderIDs[index]).
			SetOrderFeeID(fixture.receivableFeeIDs[index]).
			SetOrderNo("NT-SE" + suffix + "-" + strconv.Itoa(index)).
			SetFeeCode("OCEAN_FREIGHT").
			SetFeeName("海运费").
			SetQuantity("1.0000").
			SetUnitPrice(strings.TrimSuffix(strings.TrimSuffix(amount, "0000"), ".")).
			SetTotalAmount(amount).
			SetNetAmount(amount).
			SetTaxAmount("0.00000000").
			SetCurrency("CNY").
			SetExchangeRate("1.00000000").
			SetBaseCurrencyAmount(amount).
			SetBaseCurrency("CNY").
			SetActive(true).
			Save(ctx); lineErr != nil {
			t.Fatalf("创建测试应收账单明细 %d: %v", index, lineErr)
		}
	}
	payableCreate := data.db.FinanceBill.Create().
		SetOrganizationID(org.ID).
		SetBillNo("NT-BILL-P-" + suffix).
		SetIdempotencyKey("nt-bill-p-" + suffix).
		SetDirection(financebillent.DirectionPAYABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(customer.ID).
		SetSettlementPartyName(customer.LegalName).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetTotalAmount("800.00000000").
		SetNetAmount("800.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("800.00000000").
		SetFeeCount(1).
		SetBillDate(financeCommissionIntegrationDate).
		SetVersion(1)
	payableBill, err := withTestFinanceBillSettlementAccountSnapshot(payableCreate, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		t.Fatalf("创建测试应付账单: %v", err)
	}
	fixture.payableBillID = payableBill.ID

	if _, err = data.db.NumberRule.Create().SetOrganizationID(org.ID).SetDocumentType(numberruleent.DocumentTypeNetting).SetPrefix("NT-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试对冲编号规则: %v", err)
	}
	if _, err = data.db.NumberRule.Create().SetOrganizationID(org.ID).SetDocumentType(numberruleent.DocumentTypeCommission).SetPrefix("TC-").SetDateFormat(numberruleent.DateFormatNone).SetSequenceLength(4).SetResetPolicy(numberruleent.ResetPolicyNever).SetEnabled(true).Save(ctx); err != nil {
		t.Fatalf("创建测试提成编号规则: %v", err)
	}

	ruleItem, err := data.db.FinanceCommissionRule.Create().
		SetOrganizationID(org.ID).
		SetName("销售提成规则-" + suffix).
		SetPersonnelRole(rule.PersonnelRoleSALES).
		SetCalculationBasis(rule.CalculationBasisREALIZED_PROFIT).
		SetRatePercent("10.0000").
		SetEnabled(true).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试提成规则: %v", err)
	}
	fixture.ruleID = ruleItem.ID
	// 方案员工分配：覆盖夹具归属日期的未取消有效段，供计提自动解析唯一命中。
	if _, err = data.db.FinanceCommissionRuleAssignment.Create().
		SetOrganizationID(org.ID).
		SetRuleID(ruleItem.ID).
		SetEmployeeID(fixture.employeeID).
		SetEffectiveFrom("2026-01-01").
		SetCreatedBy(fixture.actorID).
		Save(ctx); err != nil {
		t.Fatalf("创建测试方案员工分配: %v", err)
	}

	// 走真实对冲创建/确认链路形成 CONFIRMED 对冲单：金额 800 双方全额抵销。
	nettingItem, err := fixture.newNettingUsecase().Create(ctx, org.ID, fixture.actorID, biz.CreateFinanceNettingInput{
		Bills: []biz.FinanceNettingBillVersion{
			{BillID: fixture.receivableBillIDs[0], ExpectedVersion: 1},
			{BillID: fixture.receivableBillIDs[1], ExpectedVersion: 1},
			{BillID: fixture.payableBillID, ExpectedVersion: 1},
		},
		IdempotencyKey: "nt-fixture-" + suffix,
	})
	if err != nil {
		t.Fatalf("创建测试对冲单: %v", err)
	}
	fixture.nettingID = nettingItem.ID
	confirmed, err := fixture.newNettingUsecase().Confirm(ctx, []uuid.UUID{org.ID}, fixture.actorID, nettingItem.ID, nettingItem.Version)
	if err != nil {
		t.Fatalf("确认测试对冲单: %v", err)
	}
	if confirmed.Status != biz.FinanceNettingConfirmed {
		t.Fatalf("测试对冲单状态 = %s", confirmed.Status)
	}
	return fixture
}

func (f *commissionNettingPostgresFixture) newNettingUsecase() *biz.FinanceNettingUsecase {
	return biz.NewFinanceNettingUsecase(NewFinanceNettingRepo(f.data), f.data, nil, nil)
}

func (f *commissionNettingPostgresFixture) newUsecase() *biz.CommissionUsecase {
	return biz.NewCommissionUsecase(
		NewCommissionRepo(f.data),
		biz.NewOrderConfigUsecase(NewOrderConfigRepo(f.data)),
		f.data,
	)
}

func (f *commissionNettingPostgresFixture) input(key string) biz.CreateCommissionInput {
	return biz.CreateCommissionInput{
		NettingID:      f.nettingID,
		EmployeeID:     f.employeeID,
		PersonnelRole:  biz.CommissionRoleSales,
		IdempotencyKey: "nt-commission-" + key + "-" + f.suffix,
	}
}

// requireFeeMutationLock 断言订单费用写入拦截的财务锁口径（净额 > 0 才锁）。
func (f *commissionNettingPostgresFixture) requireFeeMutationLock(orderID uuid.UUID, wantLocked bool) {
	f.t.Helper()
	err := f.data.WithTx(context.Background(), func(tx *ent.Tx) error {
		return lockOrderForFeeMutation(context.Background(), tx, f.organizationID, orderID)
	})
	if wantLocked {
		if kratoserrors.FromError(err).Reason != biz.ErrOrderFeeFinanceLocked.Reason {
			f.t.Fatalf("费用写入拦截应返回 ORDER_FEE_FINANCE_LOCKED，实际: %v", err)
		}
		return
	}
	if err != nil {
		f.t.Fatalf("净提成归零后费用写入应放行，实际: %v", err)
	}
}

func TestCommissionNettingSourcePostgres(t *testing.T) {
	t.Run("纯对冲结清走候选预览创建并断言金额", func(t *testing.T) {
		fixture := newCommissionNettingPostgresFixture(t)
		ctx := context.Background()
		usecase := fixture.newUsecase()

		candidates, err := usecase.ListNettingCandidates(ctx, fixture.organizationID, biz.CommissionNettingCandidateFilter{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("查询对冲提成候选失败: %v", err)
		}
		found := false
		for _, item := range candidates.Items {
			if item.ID == fixture.nettingID {
				found = true
			}
		}
		if !found || candidates.Total == 0 {
			t.Fatalf("已确认对冲单未进入提成候选: total=%d items=%d", candidates.Total, len(candidates.Items))
		}

		preview, err := usecase.Preview(ctx, fixture.organizationID, uuid.Nil, fixture.nettingID, fixture.employeeID, biz.CommissionRoleSales)
		if err != nil {
			t.Fatalf("对冲来源预览失败: %v", err)
		}
		if preview.VerificationID != uuid.Nil || preview.VerificationNo != "" || preview.NettingID != fixture.nettingID || preview.NettingNo == "" {
			t.Fatalf("预览来源字段不符: verification=%s/%s netting=%s/%s", preview.VerificationID, preview.VerificationNo, preview.NettingID, preview.NettingNo)
		}
		if preview.RealizedRevenue.StringFixed(8) != "800.00000000" || preview.AllocatedCost.StringFixed(8) != "300.00000000" ||
			preview.RealizedProfit.StringFixed(8) != "500.00000000" || preview.CommissionAmount.StringFixed(8) != "50.00000000" {
			t.Fatalf("对冲来源预览金额不符: rev=%s cost=%s profit=%s amount=%s",
				preview.RealizedRevenue.StringFixed(8), preview.AllocatedCost.StringFixed(8), preview.RealizedProfit.StringFixed(8), preview.CommissionAmount.StringFixed(8))
		}
		if len(preview.Lines) != 2 {
			t.Fatalf("预览明细行数 = %d，期望 2", len(preview.Lines))
		}
		lineAmounts := map[uuid.UUID]string{}
		for _, line := range preview.Lines {
			lineAmounts[line.OrderID] = line.CommissionAmount.StringFixed(8)
		}
		if lineAmounts[fixture.orderIDs[0]] != "20.00000000" || lineAmounts[fixture.orderIDs[1]] != "30.00000000" {
			t.Fatalf("分摊比例 × 账单行本位币的明细金额不符: %s / %s", lineAmounts[fixture.orderIDs[0]], lineAmounts[fixture.orderIDs[1]])
		}
		if preview.CNY == nil || preview.CNY.ExchangeRate.StringFixed(8) != "1.00000000" || preview.CNY.CommissionAmount.StringFixed(8) != "50.00000000" {
			t.Fatalf("预览 CNY 快照不符: %#v", preview.CNY)
		}
		if preview.SourceFingerprint == "" {
			t.Fatal("预览缺少幂等指纹")
		}

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("create"))
		if err != nil {
			t.Fatalf("对冲来源创建提成失败: %v", err)
		}
		if created.Status != biz.CommissionDraft || created.CommissionAmount.StringFixed(8) != "50.00000000" {
			t.Fatalf("对冲来源提成结果不符: status=%s amount=%s", created.Status, created.CommissionAmount.StringFixed(8))
		}
		if created.VerificationID != uuid.Nil || created.NettingID != fixture.nettingID || created.NettingNo == "" {
			t.Fatalf("对冲来源提成来源字段不符: %#v", created)
		}
		nettingItem, err := fixture.newNettingUsecase().Get(ctx, []uuid.UUID{fixture.organizationID}, fixture.nettingID)
		if err != nil {
			t.Fatalf("重读对冲单失败: %v", err)
		}
		if nettingItem.ConfirmedAt == nil || created.CommissionDate != nettingItem.ConfirmedAt.UTC().Format("2006-01-02") {
			t.Fatalf("对冲来源归属日期应为确认日: commissionDate=%s confirmedAt=%v", created.CommissionDate, nettingItem.ConfirmedAt)
		}
		// 确认提成在锁内重算指纹一致，来源段参与陈旧检测。
		confirmedCommission, err := usecase.Confirm(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version)
		if err != nil {
			t.Fatalf("对冲来源确认提成失败: %v", err)
		}
		if confirmedCommission.Status != biz.CommissionConfirmed {
			t.Fatalf("对冲来源提成确认状态 = %s", confirmedCommission.Status)
		}
	})

	t.Run("同来源同员工同角色重复计提被阻止且核销来源互不冲突", func(t *testing.T) {
		fixture := newCommissionNettingPostgresFixture(t)
		ctx := context.Background()
		usecase := fixture.newUsecase()

		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("dedup")); err != nil {
			t.Fatalf("创建首笔对冲来源提成失败: %v", err)
		}
		duplicate := fixture.input("dedup-again")
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, duplicate); !errors.Is(err, biz.ErrCommissionDuplicate) {
			t.Fatalf("重复计提错误 = %v，期望 %v", err, biz.ErrCommissionDuplicate)
		}

		// 幂等重放与取消后重新计提。
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("dedup"))
		if err != nil {
			t.Fatalf("幂等重放失败: %v", err)
		}
		if _, err := usecase.Cancel(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, "测试取消后重新计提"); err != nil {
			t.Fatalf("取消提成失败: %v", err)
		}
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("recreate")); err != nil {
			t.Fatalf("取消后重新计提失败: %v", err)
		}

		// 同员工同角色、不同来源（核销）互不冲突。
		verificationID := fixture.createVerificationSource(ctx)
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateCommissionInput{
			VerificationID: verificationID,
			EmployeeID:     fixture.employeeID,
			PersonnelRole:  biz.CommissionRoleSales,
			IdempotencyKey: "nt-commission-verify-" + fixture.suffix,
		}); err != nil {
			t.Fatalf("核销来源与对冲来源互不冲突失败: %v", err)
		}
	})

	t.Run("对冲反转未支付提成自动取消", func(t *testing.T) {
		fixture := newCommissionNettingPostgresFixture(t)
		ctx := context.Background()
		usecase := fixture.newUsecase()

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("unpaid"))
		if err != nil {
			t.Fatalf("创建对冲来源提成失败: %v", err)
		}
		nettingUsecase := fixture.newNettingUsecase()
		current, err := nettingUsecase.Get(ctx, []uuid.UUID{fixture.organizationID}, fixture.nettingID)
		if err != nil {
			t.Fatalf("重读对冲单失败: %v", err)
		}
		if _, err := nettingUsecase.Reverse(ctx, []uuid.UUID{fixture.organizationID}, fixture.actorID, fixture.nettingID, current.Version, "测试对冲反转取消提成"); err != nil {
			t.Fatalf("反转对冲单失败: %v", err)
		}
		reloaded, err := usecase.Get(ctx, fixture.organizationID, created.ID)
		if err != nil {
			t.Fatalf("重读提成失败: %v", err)
		}
		if reloaded.Status != biz.CommissionCancelled {
			t.Fatalf("对冲反转后提成状态 = %s，期望 CANCELLED", reloaded.Status)
		}
		if reloaded.CancellationReason == nil || !strings.Contains(*reloaded.CancellationReason, "对冲反转自动取消") {
			t.Fatalf("对冲反转取消原因不符: %v", reloaded.CancellationReason)
		}
	})

	t.Run("对冲反转已支付提成生成确认冲减且费用锁释放", func(t *testing.T) {
		fixture := newCommissionNettingPostgresFixture(t)
		ctx := context.Background()
		usecase := fixture.newUsecase()

		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("paid"))
		if err != nil {
			t.Fatalf("创建对冲来源提成失败: %v", err)
		}
		confirmedCommission, err := usecase.Confirm(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version)
		if err != nil {
			t.Fatalf("确认提成失败: %v", err)
		}
		paid, err := usecase.MarkPaid(ctx, fixture.organizationID, fixture.actorID, confirmedCommission.ID, confirmedCommission.Version)
		if err != nil {
			t.Fatalf("发放提成失败: %v", err)
		}
		if paid.Status != biz.CommissionPaid {
			t.Fatalf("发放后状态 = %s", paid.Status)
		}
		// 已支付提成形成正净额，费用写入被财务锁拦截。
		fixture.requireFeeMutationLock(fixture.orderIDs[0], true)

		nettingUsecase := fixture.newNettingUsecase()
		current, err := nettingUsecase.Get(ctx, []uuid.UUID{fixture.organizationID}, fixture.nettingID)
		if err != nil {
			t.Fatalf("重读对冲单失败: %v", err)
		}
		reversed, err := nettingUsecase.Reverse(ctx, []uuid.UUID{fixture.organizationID}, fixture.actorID, fixture.nettingID, current.Version, "测试对冲反转冲减提成")
		if err != nil {
			t.Fatalf("反转对冲单失败: %v", err)
		}
		if reversed.Status != biz.FinanceNettingReversed {
			t.Fatalf("对冲反转状态 = %s", reversed.Status)
		}

		reloaded, err := usecase.Get(ctx, fixture.organizationID, created.ID)
		if err != nil {
			t.Fatalf("重读提成失败: %v", err)
		}
		if reloaded.Status != biz.CommissionPaid {
			t.Fatalf("PAID 父单状态不应被改写: %s", reloaded.Status)
		}
		if len(reloaded.Adjustments) != 2 {
			t.Fatalf("冲减调整数 = %d，期望 2（逐订单冲减）", len(reloaded.Adjustments))
		}
		totalDecrease := decimal.Zero
		for _, item := range reloaded.Adjustments {
			if item.SourceType != biz.CommissionAdjustmentSourceNettingReversal || item.Direction != biz.CommissionAdjustmentDecrease || item.Status != biz.CommissionConfirmed {
				t.Fatalf("冲减调整来源或状态不符: %#v", item)
			}
			if !strings.HasPrefix(item.IdempotencyKey, "nt:"+fixture.nettingID.String()) {
				t.Fatalf("冲减调整幂等键前缀不符: %s", item.IdempotencyKey)
			}
			totalDecrease = totalDecrease.Add(item.Amount)
		}
		if totalDecrease.StringFixed(8) != "50.00000000" || reloaded.AdjustmentAmount.StringFixed(8) != "-50.00000000" ||
			reloaded.EffectiveCommissionAmount.StringFixed(8) != "0.00000000" {
			t.Fatalf("全额冲减金额不符: decrease=%s adjustment=%s effective=%s", totalDecrease.StringFixed(8), reloaded.AdjustmentAmount.StringFixed(8), reloaded.EffectiveCommissionAmount.StringFixed(8))
		}
		// 全额冲减后净额归零，费用财务锁释放。
		fixture.requireFeeMutationLock(fixture.orderIDs[0], false)

		// 反转后的对冲单不再进入提成候选。
		candidates, err := usecase.ListNettingCandidates(ctx, fixture.organizationID, biz.CommissionNettingCandidateFilter{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("重查对冲候选失败: %v", err)
		}
		for _, item := range candidates.Items {
			if item.ID == fixture.nettingID {
				t.Fatal("已反转对冲单不应再进入提成候选")
			}
		}
	})

	t.Run("不同对冲单的指纹包含来源段", func(t *testing.T) {
		fixture := newCommissionNettingPostgresFixture(t)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		nettingUsecase := fixture.newNettingUsecase()

		first, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.input("fp-1"))
		if err != nil {
			t.Fatalf("创建首笔对冲提成失败: %v", err)
		}
		if _, err := usecase.Cancel(ctx, fixture.organizationID, fixture.actorID, first.ID, first.Version, "测试指纹来源段"); err != nil {
			t.Fatalf("取消首笔提成失败: %v", err)
		}
		current, err := nettingUsecase.Get(ctx, []uuid.UUID{fixture.organizationID}, fixture.nettingID)
		if err != nil {
			t.Fatalf("重读对冲单失败: %v", err)
		}
		if _, err := nettingUsecase.Reverse(ctx, []uuid.UUID{fixture.organizationID}, fixture.actorID, fixture.nettingID, current.Version, "测试指纹来源段"); err != nil {
			t.Fatalf("反转首个对冲单失败: %v", err)
		}

		// 余额恢复后用同批账单重新对冲，账单/费用/订单事实与首单一致。
		secondNetting, err := nettingUsecase.Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateFinanceNettingInput{
			Bills: []biz.FinanceNettingBillVersion{
				{BillID: fixture.receivableBillIDs[0], ExpectedVersion: 1},
				{BillID: fixture.receivableBillIDs[1], ExpectedVersion: 1},
				{BillID: fixture.payableBillID, ExpectedVersion: 1},
			},
			IdempotencyKey: "nt-fp-2-" + fixture.suffix,
		})
		if err != nil {
			t.Fatalf("重新创建对冲单失败: %v", err)
		}
		if _, err := nettingUsecase.Confirm(ctx, []uuid.UUID{fixture.organizationID}, fixture.actorID, secondNetting.ID, secondNetting.Version); err != nil {
			t.Fatalf("确认第二个对冲单失败: %v", err)
		}
		secondInput := fixture.input("fp-2")
		secondInput.NettingID = secondNetting.ID
		second, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, secondInput)
		if err != nil {
			t.Fatalf("创建第二笔对冲提成失败: %v", err)
		}
		if second.SourceFingerprint == "" || second.SourceFingerprint == first.SourceFingerprint {
			t.Fatalf("不同对冲来源的指纹应包含来源段: first=%s second=%s", first.SourceFingerprint, second.SourceFingerprint)
		}
		if _, err := usecase.Confirm(ctx, fixture.organizationID, fixture.actorID, second.ID, second.Version); err != nil {
			t.Fatalf("第二笔对冲提成确认失败: %v", err)
		}
	})
}

// createVerificationSource 在同一夹具内补一条核销来源（新费用 + 新账单 + 流水 +
// 核销分摊），用于验证核销与对冲来源互不冲突。
func (f *commissionNettingPostgresFixture) createVerificationSource(ctx context.Context) uuid.UUID {
	f.t.Helper()
	receivableFee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderIDs[0]).
		SetIdempotencyKey("nt-verify-fee-" + f.suffix).
		SetDirection(fee.DirectionRECEIVABLE).
		SetStatus(fee.StatusCONFIRMED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(f.customerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("200.0000").
		SetTotalAmount("200.00000000").
		SetNetAmount("200.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(fee.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("200.00000000").
		SetExpenseDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建核销来源费用: %v", err)
	}
	billCreate := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo("NT-VBILL-" + f.suffix).
		SetIdempotencyKey("nt-vbill-" + f.suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(f.customerID).
		SetSettlementPartyName("对冲提成客户-" + f.suffix).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetTotalAmount("200.00000000").
		SetNetAmount("200.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("200.00000000").
		SetFeeCount(1).
		SetBillDate(financeCommissionIntegrationDate).
		SetVersion(1)
	billItem, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		f.t.Fatalf("创建核销来源账单: %v", err)
	}
	if _, err = f.data.db.FinanceBillLine.Create().
		SetBillID(billItem.ID).
		SetOrderID(f.orderIDs[0]).
		SetOrderFeeID(receivableFee.ID).
		SetOrderNo("NT-SE" + f.suffix + "-0").
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetQuantity("1.0000").
		SetUnitPrice("200.0000").
		SetTotalAmount("200.00000000").
		SetNetAmount("200.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrencyAmount("200.00000000").
		SetBaseCurrency("CNY").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建核销来源账单明细: %v", err)
	}
	cashflow, err := f.data.db.FinanceCashflow.Create().
		SetOrganizationID(f.organizationID).
		SetFlowNo("NT-FLOW-" + f.suffix).
		SetIdempotencyKey("nt-flow-" + f.suffix).
		SetDirection(financecashflowent.DirectionRECEIVABLE).
		SetStatus(financecashflowent.StatusCONFIRMED).
		SetSettlementPartyID(f.customerID).
		SetSettlementPartyName("对冲提成客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount("200.00000000").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financecashflowent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeCommissionIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseAmount("200.00000000").
		SetTransactionDate(financeCommissionIntegrationDate).
		SetOurAccount("测试账户").
		SetPaymentMethod("BANK_TRANSFER").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建核销来源资金流水: %v", err)
	}
	verificationItem, err := f.data.db.FinanceVerification.Create().
		SetOrganizationID(f.organizationID).
		SetVerificationNo("NT-VR-" + f.suffix).
		SetIdempotencyKey("nt-verification-" + f.suffix).
		SetDirection(verification.DirectionRECEIVABLE).
		SetStatus(verification.StatusACTIVE).
		SetSettlementPartyID(f.customerID).
		SetSettlementPartyName("对冲提成客户-" + f.suffix).
		SetCurrency("CNY").
		SetAmount("200.00000000").
		SetBaseCurrency("CNY").
		SetBaseAmount("200.00000000").
		SetBillBaseAmount("200.00000000").
		SetCashflowBaseAmount("200.00000000").
		SetExchangeGainLoss("0.00000000").
		SetVerificationDate(financeCommissionIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建核销来源核销单: %v", err)
	}
	if _, err = f.data.db.FinanceVerificationAllocation.Create().
		SetVerificationID(verificationItem.ID).
		SetCashflowID(cashflow.ID).
		SetBillID(billItem.ID).
		SetCashflowNo(cashflow.FlowNo).
		SetBillNo(billItem.BillNo).
		SetAmount("200.00000000").
		SetBillBaseAmount("200.00000000").
		SetCashflowBaseAmount("200.00000000").
		SetExchangeGainLoss("0.00000000").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建核销来源核销分摊: %v", err)
	}
	return verificationItem.ID
}
