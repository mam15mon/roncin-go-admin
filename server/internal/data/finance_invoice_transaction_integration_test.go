package data

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financeinvoiceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoice"
	financeinvoicebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financeinvoicebill"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
)

const financeInvoiceIntegrationDate = "2026-09-10"

// financeInvoicePostgresFixture 直接用 Ent 夹具构造已确认账单与开票记录，
// 验证发票终态命令的状态机、锁序和账单版本推进，不依赖建账用例。
type financeInvoicePostgresFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	partnerID      uuid.UUID
	orderID        uuid.UUID
	actorID        uuid.UUID
	suffix         string
}

func newFinanceInvoicePostgresFixture(t *testing.T, data *Data) *financeInvoicePostgresFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("INV-TX-" + suffix).
		SetName("发票事务集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	partner, err := data.db.Partner.Create().
		SetOrganizationID(organization.ID).
		SetCode("CUSTOMER-" + suffix).
		SetLegalName("发票事务测试客户-" + suffix).
		SetNormalizedName("发票事务测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}
	order, err := data.db.Order.Create().
		SetIdempotencyKey(uuid.NewString()).
		SetOrganizationID(organization.ID).
		SetOrderNo("SE" + suffix).
		SetCustomerID(partner.ID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试订单: %v", err)
	}
	actor, err := data.db.User.Create().
		SetDisplayName("发票事务集成测试用户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试操作用户: %v", err)
	}
	return &financeInvoicePostgresFixture{
		t: t, data: data, organizationID: organization.ID, partnerID: partner.ID,
		orderID: order.ID, actorID: actor.ID, suffix: suffix,
	}
}

// createConfirmedBill 创建带活动明细与 BILLED 费用的已确认账单（version 1），
// 使其同时满足发票关联与账单取消路径的前置事实。
func (f *financeInvoicePostgresFixture) createConfirmedBill(key string) *ent.FinanceBill {
	f.t.Helper()
	ctx := context.Background()
	fee, err := f.data.db.OrderFee.Create().
		SetOrderID(f.orderID).
		SetIdempotencyKey("inv-fee-" + key + "-" + f.suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusBILLED).
		SetFeeCode("INV-FEE-" + key).
		SetFeeName("发票事务测试费用").
		SetSettlementPartyID(f.partnerID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeInvoiceIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("100.00000000").
		SetExpenseDate(financeInvoiceIntegrationDate).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试费用: %v", err)
	}
	billCreate := f.data.db.FinanceBill.Create().
		SetOrganizationID(f.organizationID).
		SetBillNo("INV-BILL-" + key + "-" + f.suffix).
		SetIdempotencyKey("inv-bill-" + key + "-" + f.suffix).
		SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(financebillent.StatusCONFIRMED).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("发票事务测试客户-" + f.suffix).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).
		SetExchangeRateDate(financeInvoiceIntegrationDate).
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetBaseCurrencyAmount("100.00000000").
		SetFeeCount(1).
		SetBillDate(financeInvoiceIntegrationDate)
	bill, err := withTestFinanceBillSettlementAccountSnapshot(billCreate, uuid.New(), "CNY").Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试账单: %v", err)
	}
	if _, err = f.data.db.FinanceBillLine.Create().
		SetBillID(bill.ID).
		SetOrderID(f.orderID).
		SetOrderFeeID(fee.ID).
		SetOrderNo("SE" + f.suffix).
		SetFeeCode(fee.FeeCode).
		SetFeeName(fee.FeeName).
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("CNY").
		SetExchangeRate("1.00000000").
		SetBaseCurrencyAmount("100.00000000").
		SetBaseCurrency("CNY").
		SetActive(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试账单明细: %v", err)
	}
	return bill
}

// createInvoiceWithStatus 创建指向 bills 的发票并建立活动关联，便于直接验证终态命令。
// 各状态的数据库一致性约束由夹具按真实事实补齐：ISSUED/RED_FLUSHED 带开票与汇率快照，
// CANCELLED 带取消事实，RED_FLUSHED 额外带红冲事实。
func (f *financeInvoicePostgresFixture) createInvoiceWithStatus(key string, status financeinvoiceent.Status, bills ...*ent.FinanceBill) *ent.FinanceInvoice {
	f.t.Helper()
	ctx := context.Background()
	now := time.Now()
	create := f.data.db.FinanceInvoice.Create().
		SetOrganizationID(f.organizationID).
		SetRecordNo("INV-" + key + "-" + f.suffix).
		SetIdempotencyKey("inv-" + key + "-" + f.suffix).
		SetDirection(financeinvoiceent.DirectionRECEIVABLE).
		SetStatus(status).
		SetInvoiceType(financeinvoiceent.InvoiceTypeNORMAL).
		SetSettlementPartyID(f.partnerID).
		SetSettlementPartyName("发票事务测试客户-" + f.suffix).
		SetCurrency("CNY").
		SetBaseCurrency("CNY").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetBillCount(len(bills)).
		SetVersion(1)
	switch status {
	case financeinvoiceent.StatusISSUED, financeinvoiceent.StatusRED_FLUSHED:
		create = create.
			SetTaxInvoiceNo("TAX-" + key + "-" + f.suffix).
			SetInvoiceDate(financeInvoiceIntegrationDate).
			SetIssuedAt(now).
			SetIssuedBy(f.actorID).
			SetExchangeRate("1.00000000").
			SetExchangeRateSource(financeinvoiceent.ExchangeRateSourceSYSTEM).
			SetExchangeRateDate(financeInvoiceIntegrationDate).
			SetBaseCurrencyAmount("100.00000000")
	}
	if status == financeinvoiceent.StatusCANCELLED {
		create = create.
			SetCancelledAt(now).
			SetCancelledBy(f.actorID).
			SetCancellationReason("夹具预置取消事实")
	}
	if status == financeinvoiceent.StatusRED_FLUSHED {
		create = create.
			SetRedInvoiceNo("RED-" + key + "-" + f.suffix).
			SetRedInvoiceDate(financeInvoiceIntegrationDate).
			SetRedFlushedAt(now).
			SetRedFlushedBy(f.actorID).
			SetRedFlushReason("夹具预置红冲事实")
	}
	invoice, err := create.Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试发票: %v", err)
	}
	for _, bill := range bills {
		if _, err = f.data.db.FinanceInvoiceBill.Create().
			SetInvoiceID(invoice.ID).
			SetBillID(bill.ID).
			SetBillNo(bill.BillNo).
			SetAmount("100.00000000").
			SetTaxAmount("0.00000000").
			SetActive(true).
			Save(ctx); err != nil {
			f.t.Fatalf("创建测试发票关联: %v", err)
		}
	}
	return invoice
}

func financeInvoiceTestAudit(action string, org, actor, id uuid.UUID) *biz.AuditEvent {
	return &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: action, Result: "success", ResourceType: "finance_invoice", ResourceID: id.String()}
}

func (f *financeInvoicePostgresFixture) requireActiveLinks(invoiceID uuid.UUID, wantActive bool) {
	f.t.Helper()
	ctx := context.Background()
	active, err := f.data.db.FinanceInvoiceBill.Query().
		Where(financeinvoicebillent.InvoiceIDEQ(invoiceID), financeinvoicebillent.ActiveEQ(true)).
		Count(ctx)
	if err != nil {
		f.t.Fatalf("查询发票活动关联失败: %v", err)
	}
	if wantActive && active == 0 {
		f.t.Fatal("发票应仍存在活动关联")
	}
	if !wantActive && active != 0 {
		f.t.Fatalf("发票活动关联未全部释放，剩余 %d 条", active)
	}
}

func (f *financeInvoicePostgresFixture) requireBillVersion(bills []*ent.FinanceBill, wantVersion uint64) {
	f.t.Helper()
	ctx := context.Background()
	for _, bill := range bills {
		current, err := f.data.db.FinanceBill.Get(ctx, bill.ID)
		if err != nil {
			f.t.Fatalf("读取账单 %s 失败: %v", bill.BillNo, err)
		}
		if current.Version != wantVersion {
			f.t.Fatalf("账单 %s 版本 = %d，期望 %d", bill.BillNo, current.Version, wantVersion)
		}
	}
}

// TestFinanceInvoiceTerminalStateMachinePostgres 验证 INV-01 状态矩阵：
// Cancel 只接受 DRAFT/ISSUED，RedFlush 只接受 ISSUED，两个终态不可重复执行或互转。
func TestFinanceInvoiceTerminalStateMachinePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	cancelMatrix := []struct {
		name   string
		status financeinvoiceent.Status
		allow  bool
	}{
		{name: "DRAFT 允许取消", status: financeinvoiceent.StatusDRAFT, allow: true},
		{name: "ISSUED 允许作废", status: financeinvoiceent.StatusISSUED, allow: true},
		{name: "CANCELLED 拒绝重复取消", status: financeinvoiceent.StatusCANCELLED, allow: false},
		{name: "RED_FLUSHED 拒绝取消", status: financeinvoiceent.StatusRED_FLUSHED, allow: false},
	}
	for _, item := range cancelMatrix {
		t.Run("Cancel/"+item.name, func(t *testing.T) {
			fixture := newFinanceInvoicePostgresFixture(t, data)
			repo := NewFinanceInvoiceRepo(data)
			bill := fixture.createConfirmedBill("cancel-" + strings.ToLower(item.status.String()))
			invoice := fixture.createInvoiceWithStatus("cancel-"+strings.ToLower(item.status.String()), item.status, bill)
			result, err := repo.Cancel(context.Background(), fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version, "取消原因", financeInvoiceTestAudit("finance.invoice.cancel", fixture.organizationID, fixture.actorID, invoice.ID))
			if item.allow {
				if err != nil {
					t.Fatalf("取消 %s 发票失败: %v", item.status, err)
				}
				if result.Status != biz.FinanceInvoiceCancelled || result.Version != 2 || result.CancelledAt == nil {
					t.Fatalf("取消后发票状态不正确: status=%s version=%d", result.Status, result.Version)
				}
				fixture.requireActiveLinks(invoice.ID, false)
			} else {
				if !errors.Is(err, biz.ErrFinanceInvoiceInvalidTransition) {
					t.Fatalf("取消 %s 发票错误 = %v，期望 %v", item.status, err, biz.ErrFinanceInvoiceInvalidTransition)
				}
				fixture.requireActiveLinks(invoice.ID, true)
			}
		})
	}

	redFlushMatrix := []struct {
		name   string
		status financeinvoiceent.Status
		allow  bool
	}{
		{name: "DRAFT 拒绝红冲", status: financeinvoiceent.StatusDRAFT, allow: false},
		{name: "ISSUED 允许红冲", status: financeinvoiceent.StatusISSUED, allow: true},
		{name: "CANCELLED 拒绝红冲", status: financeinvoiceent.StatusCANCELLED, allow: false},
		{name: "RED_FLUSHED 拒绝重复红冲", status: financeinvoiceent.StatusRED_FLUSHED, allow: false},
	}
	for _, item := range redFlushMatrix {
		t.Run("RedFlush/"+item.name, func(t *testing.T) {
			fixture := newFinanceInvoicePostgresFixture(t, data)
			repo := NewFinanceInvoiceRepo(data)
			key := "red-" + strings.ToLower(item.status.String())
			bill := fixture.createConfirmedBill(key)
			invoice := fixture.createInvoiceWithStatus(key, item.status, bill)
			result, err := repo.RedFlush(context.Background(), fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version,
				"RED-"+fixture.suffix, financeInvoiceIntegrationDate, "红冲原因",
				financeInvoiceTestAudit("finance.invoice.red_flush", fixture.organizationID, fixture.actorID, invoice.ID))
			if item.allow {
				if err != nil {
					t.Fatalf("红冲 %s 发票失败: %v", item.status, err)
				}
				if result.Status != biz.FinanceInvoiceRedFlushed || result.Version != 2 || result.RedFlushedAt == nil {
					t.Fatalf("红冲后发票状态不正确: status=%s version=%d", result.Status, result.Version)
				}
				fixture.requireActiveLinks(invoice.ID, false)
			} else {
				if !errors.Is(err, biz.ErrFinanceInvoiceInvalidTransition) {
					t.Fatalf("红冲 %s 发票错误 = %v，期望 %v", item.status, err, biz.ErrFinanceInvoiceInvalidTransition)
				}
				fixture.requireActiveLinks(invoice.ID, true)
			}
		})
	}
}

// TestFinanceInvoiceTerminalReleasePostgres 验证 INV-02 释放与版本推进：
// 释放活动关联后每张账单 version 恰好 +1，携带旧版本的账单操作以冲突失败，新版本可继续取消。
func TestFinanceInvoiceTerminalReleasePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("作废释放多张账单并各推进一次版本", func(t *testing.T) {
		fixture := newFinanceInvoicePostgresFixture(t, data)
		bills := []*ent.FinanceBill{
			fixture.createConfirmedBill("release-a"),
			fixture.createConfirmedBill("release-b"),
		}
		invoice := fixture.createInvoiceWithStatus("release", financeinvoiceent.StatusISSUED, bills...)
		repo := NewFinanceInvoiceRepo(data)
		if _, err := repo.Cancel(context.Background(), fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version, "作废原因", financeInvoiceTestAudit("finance.invoice.cancel", fixture.organizationID, fixture.actorID, invoice.ID)); err != nil {
			t.Fatalf("作废发票失败: %v", err)
		}
		fixture.requireActiveLinks(invoice.ID, false)
		fixture.requireBillVersion(bills, 2)
	})

	t.Run("红冲释放账单并推进版本", func(t *testing.T) {
		fixture := newFinanceInvoicePostgresFixture(t, data)
		bill := fixture.createConfirmedBill("red-release")
		invoice := fixture.createInvoiceWithStatus("red-release", financeinvoiceent.StatusISSUED, bill)
		repo := NewFinanceInvoiceRepo(data)
		if _, err := repo.RedFlush(context.Background(), fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version,
			"RED-"+fixture.suffix, financeInvoiceIntegrationDate, "红冲原因",
			financeInvoiceTestAudit("finance.invoice.red_flush", fixture.organizationID, fixture.actorID, invoice.ID)); err != nil {
			t.Fatalf("红冲发票失败: %v", err)
		}
		fixture.requireActiveLinks(invoice.ID, false)
		fixture.requireBillVersion([]*ent.FinanceBill{bill}, 2)
	})

	t.Run("旧账单版本取消失败，新版本取消成功", func(t *testing.T) {
		fixture := newFinanceInvoicePostgresFixture(t, data)
		bill := fixture.createConfirmedBill("bill-version")
		invoice := fixture.createInvoiceWithStatus("bill-version", financeinvoiceent.StatusDRAFT, bill)
		invoiceRepo := NewFinanceInvoiceRepo(data)
		billRepo := NewFinanceBillRepo(data)
		ctx := context.Background()
		if _, err := invoiceRepo.Cancel(ctx, fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version, "取消原因", financeInvoiceTestAudit("finance.invoice.cancel", fixture.organizationID, fixture.actorID, invoice.ID)); err != nil {
			t.Fatalf("取消发票失败: %v", err)
		}
		if _, err := billRepo.Cancel(ctx, []uuid.UUID{fixture.organizationID}, bill.ID, fixture.actorID, bill.Version, "旧版本取消", financeInvoiceTestAudit("finance.bill.cancel", fixture.organizationID, fixture.actorID, bill.ID)); !errors.Is(err, biz.ErrFinanceBillVersionConflict) {
			t.Fatalf("旧版本取消账单错误 = %v，期望 %v", err, biz.ErrFinanceBillVersionConflict)
		}
		cancelled, err := billRepo.Cancel(ctx, []uuid.UUID{fixture.organizationID}, bill.ID, fixture.actorID, bill.Version+1, "释放后取消", financeInvoiceTestAudit("finance.bill.cancel", fixture.organizationID, fixture.actorID, bill.ID))
		if err != nil {
			t.Fatalf("释放后取消账单失败: %v", err)
		}
		if cancelled.Status != biz.FinanceBillCancelled {
			t.Fatalf("释放后账单状态 = %s，期望 CANCELLED", cancelled.Status)
		}
	})
}

// TestFinanceInvoiceTerminalRollbackPostgres 验证事务原子性：
// 审计写入失败时发票状态、活动关联与账单版本全部回滚，无部分释放。
func TestFinanceInvoiceTerminalRollbackPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	for _, command := range []struct {
		name   string
		status financeinvoiceent.Status
		run    func(ctx context.Context, f *financeInvoicePostgresFixture, repo biz.FinanceInvoiceRepo, invoice *ent.FinanceInvoice, audit *biz.AuditEvent) error
	}{
		{
			name:   "Cancel",
			status: financeinvoiceent.StatusISSUED,
			run: func(ctx context.Context, f *financeInvoicePostgresFixture, repo biz.FinanceInvoiceRepo, invoice *ent.FinanceInvoice, audit *biz.AuditEvent) error {
				_, err := repo.Cancel(ctx, f.organizationID, invoice.ID, f.actorID, invoice.Version, "作废原因", audit)
				return err
			},
		},
		{
			name:   "RedFlush",
			status: financeinvoiceent.StatusISSUED,
			run: func(ctx context.Context, f *financeInvoicePostgresFixture, repo biz.FinanceInvoiceRepo, invoice *ent.FinanceInvoice, audit *biz.AuditEvent) error {
				_, err := repo.RedFlush(ctx, f.organizationID, invoice.ID, f.actorID, invoice.Version, "RED-"+f.suffix, financeInvoiceIntegrationDate, "红冲原因", audit)
				return err
			},
		},
	} {
		t.Run(command.name+"/审计失败整体回滚", func(t *testing.T) {
			fixture := newFinanceInvoicePostgresFixture(t, data)
			bills := []*ent.FinanceBill{
				fixture.createConfirmedBill("rollback-a"),
				fixture.createConfirmedBill("rollback-b"),
			}
			invoice := fixture.createInvoiceWithStatus("rollback", command.status, bills...)
			repo := NewFinanceInvoiceRepo(data)
			invalidAudit := financeInvoiceTestAudit("finance.invoice."+strings.ToLower(command.name), fixture.organizationID, fixture.actorID, invoice.ID)
			invalidAudit.Result = "invalid"
			if err := command.run(context.Background(), fixture, repo, invoice, invalidAudit); err == nil {
				t.Fatal("审计结果非法时命令未失败")
			}
			ctx := context.Background()
			current, getErr := fixture.data.db.FinanceInvoice.Get(ctx, invoice.ID)
			if getErr != nil || current.Status != command.status || current.Version != 1 {
				t.Fatalf("回滚后发票状态 = %s version = %d（期望 %s/1），error=%v", current.Status, current.Version, command.status, getErr)
			}
			fixture.requireActiveLinks(invoice.ID, true)
			fixture.requireBillVersion(bills, 1)
			auditCount, auditErr := fixture.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(fixture.organizationID)).Count(ctx)
			if auditErr != nil || auditCount != 0 {
				t.Fatalf("回滚后审计数 = %d，期望 0，error=%v", auditCount, auditErr)
			}
		})
	}
}

// runInvoiceCommandsConcurrently 并发执行命令并按下标收集结果（完成顺序无关），用于验证锁序与唯一胜者。
func runInvoiceCommandsConcurrently(commands ...func(ctx context.Context) error) []error {
	start := make(chan struct{})
	results := make([]error, len(commands))
	var wg sync.WaitGroup
	for index, command := range commands {
		index, command := index, command
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			results[index] = command(ctx)
		}()
	}
	close(start)
	wg.Wait()
	return results
}

// TestFinanceInvoiceTerminalConcurrencyPostgres 验证并发场景：
// 同一发票并发双命令只有一个成功且账单版本只推进一次；
// 发票作废与账单取消并发时按固定锁序执行，无死锁、无部分提交。
func TestFinanceInvoiceTerminalConcurrencyPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("并发取消与红冲同一发票只有一个成功", func(t *testing.T) {
		fixture := newFinanceInvoicePostgresFixture(t, data)
		bills := []*ent.FinanceBill{
			fixture.createConfirmedBill("race-a"),
			fixture.createConfirmedBill("race-b"),
		}
		invoice := fixture.createInvoiceWithStatus("race", financeinvoiceent.StatusISSUED, bills...)
		repo := NewFinanceInvoiceRepo(data)
		errs := runInvoiceCommandsConcurrently(
			func(ctx context.Context) error {
				_, err := repo.Cancel(ctx, fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version, "作废原因", financeInvoiceTestAudit("finance.invoice.cancel", fixture.organizationID, fixture.actorID, invoice.ID))
				return err
			},
			func(ctx context.Context) error {
				_, err := repo.RedFlush(ctx, fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version, "RED-"+fixture.suffix, financeInvoiceIntegrationDate, "红冲原因", financeInvoiceTestAudit("finance.invoice.red_flush", fixture.organizationID, fixture.actorID, invoice.ID))
				return err
			},
		)
		successes, conflicts := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				successes++
			case errors.Is(err, biz.ErrFinanceInvoiceVersionConflict):
				conflicts++
			default:
				t.Fatalf("并发命令返回非预期错误: %v", err)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("并发双命令 success=%d conflict=%d，期望各 1", successes, conflicts)
		}
		ctx := context.Background()
		current, err := fixture.data.db.FinanceInvoice.Get(ctx, invoice.ID)
		if err != nil || current.Version != 2 || (current.Status != financeinvoiceent.StatusCANCELLED && current.Status != financeinvoiceent.StatusRED_FLUSHED) {
			t.Fatalf("并发后发票状态不正确: status=%s version=%d，error=%v", current.Status, current.Version, err)
		}
		fixture.requireActiveLinks(invoice.ID, false)
		fixture.requireBillVersion(bills, 2)
	})

	t.Run("并发发票作废与账单取消无死锁且不部分提交", func(t *testing.T) {
		fixture := newFinanceInvoicePostgresFixture(t, data)
		bill := fixture.createConfirmedBill("lock-order")
		invoice := fixture.createInvoiceWithStatus("lock-order", financeinvoiceent.StatusISSUED, bill)
		invoiceRepo := NewFinanceInvoiceRepo(data)
		billRepo := NewFinanceBillRepo(data)
		errs := runInvoiceCommandsConcurrently(
			func(ctx context.Context) error {
				_, err := invoiceRepo.Cancel(ctx, fixture.organizationID, invoice.ID, fixture.actorID, invoice.Version, "作废原因", financeInvoiceTestAudit("finance.invoice.cancel", fixture.organizationID, fixture.actorID, invoice.ID))
				return err
			},
			func(ctx context.Context) error {
				_, err := billRepo.Cancel(ctx, []uuid.UUID{fixture.organizationID}, bill.ID, fixture.actorID, bill.Version, "并发取消账单", financeInvoiceTestAudit("finance.bill.cancel", fixture.organizationID, fixture.actorID, bill.ID))
				return err
			},
		)
		invoiceErr, billErr := errs[0], errs[1]
		if invoiceErr != nil {
			t.Fatalf("并发中发票作废失败（活动关联未被账单侧锁等待阻塞）: %v", invoiceErr)
		}
		if billErr == nil {
			t.Fatal("账单在发票活动关联未释放时不应取消成功")
		}
		if !errors.Is(billErr, biz.ErrFinanceBillVersionConflict) && !errors.Is(billErr, biz.ErrFinanceBillInvalidTransition) {
			t.Fatalf("账单取消错误 = %v，期望版本冲突或状态冲突", billErr)
		}
		ctx := context.Background()
		currentBill, err := fixture.data.db.FinanceBill.Get(ctx, bill.ID)
		if err != nil || currentBill.Status != financebillent.StatusCONFIRMED || currentBill.Version != 2 {
			t.Fatalf("并发后账单状态 = %s version = %d，期望 CONFIRMED/2，error=%v", currentBill.Status, currentBill.Version, err)
		}
		currentInvoice, err := fixture.data.db.FinanceInvoice.Get(ctx, invoice.ID)
		if err != nil || currentInvoice.Status != financeinvoiceent.StatusCANCELLED {
			t.Fatalf("并发后发票状态 = %s，期望 CANCELLED，error=%v", currentInvoice.Status, err)
		}
		fixture.requireActiveLinks(invoice.ID, false)
	})
}
