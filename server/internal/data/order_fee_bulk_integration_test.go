package data

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	enterpriseresourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeeenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
)

// newOrderFeeBulkUsecase 组装带信用控制与真实汇率解析的订单费用用例，
// 批量维护路径与生产装配一致。
func newOrderFeeBulkUsecase(data *Data) *biz.OrderFeeUsecase {
	return biz.NewOrderFeeUsecase(
		NewOrderFeeRepo(data),
		biz.NewExchangeRateUsecase(NewExchangeRateRepo(data), nil),
		nil,
		biz.NewPartnerCreditUsecase(NewFinanceBillRepo(data), biz.NewFinanceCustomSettingUsecase(NewFinanceCustomSettingRepo(data))),
		nil,
		nil,
	)
}

// createBulkFee 在指定订单下直接创建一条最小可用费用行（结算单位挂夹具往来单位）。
func createBulkFee(t *testing.T, data *Data, orderID, settlementPartyID uuid.UUID, key, currency, rate, baseAmount string, source orderfeeent.ExchangeRateSource) uuid.UUID {
	t.Helper()
	fee, err := data.db.OrderFee.Create().
		SetOrderID(orderID).
		SetIdempotencyKey("bulk-" + key + "-" + uuid.NewString()[:8]).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusUNBILLED).
		SetFeeCode("OCEAN_FREIGHT").
		SetFeeName("海运费").
		SetSettlementPartyID(settlementPartyID).
		SetBillingUnit("票").
		SetQuantity("1.0000").
		SetUnitPrice("100.0000").
		SetTotalAmount("100.00000000").
		SetNetAmount("100.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency(currency).
		SetExchangeRate(rate).
		SetExchangeRateSource(source).
		SetExchangeRateDate(financeBillIntegrationDate).
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount(baseAmount).
		SetExpenseDate(financeBillIntegrationDate).
		SetVersion(1).
		Save(context.Background())
	if err != nil {
		t.Fatalf("创建批量测试费用 %s 失败: %v", key, err)
	}
	return fee.ID
}

// TestOrderFeeBulkMaintenancePostgres 验证订单费用批量维护的真实事务行为：
// 批量改结算单位不动金额汇率、批量改费用时间逐行重解析系统汇率并保留手工
// 汇率、批量删除级联清理标签；任一行版本冲突、状态不符、账单占用或越订单
// 时整批零写入回滚；审计逐行落库。隔离 Schema 执行全量迁移链冷启动。
func TestOrderFeeBulkMaintenancePostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	data, cleanupData := getIntegrationData(t)
	t.Cleanup(cleanupData)
	fixture := newFinanceBillPostgresFixture(t, data)

	orderFeeRepo := NewOrderFeeRepo(data)
	usecase := newOrderFeeBulkUsecase(data)
	billUsecase := fixture.newUsecase(NewFinanceBillRepo(data))
	actorUser, userErr := data.db.User.Create().
		SetDisplayName("费用批量集成测试用户-" + fixture.suffix).
		Save(context.Background())
	if userErr != nil {
		t.Fatalf("创建测试操作用户: %v", userErr)
	}
	actor := actorUser.ID
	organizationID, orderID := fixture.organizationID, fixture.orderID

	// 目标结算单位：同组织启用往来单位。
	newParty, partyErr := data.db.Partner.Create().
		SetOrganizationID(organizationID).
		SetCode("BULK-PARTY-" + fixture.suffix).
		SetLegalName("批量维护目标结算单位-" + fixture.suffix).
		SetNormalizedName("批量维护目标结算单位-" + fixture.suffix).
		Save(context.Background())
	if partyErr != nil {
		t.Fatalf("创建目标结算单位失败: %v", partyErr)
	}

	targetsOf := func(feeIDs ...uuid.UUID) []biz.OrderFeeBulkTarget {
		targets := make([]biz.OrderFeeBulkTarget, 0, len(feeIDs))
		for _, id := range feeIDs {
			targets = append(targets, biz.OrderFeeBulkTarget{FeeID: id, ExpectedVersion: 1})
		}
		return targets
	}

	t.Run("批量改结算单位同步法定名称且不动金额汇率", func(t *testing.T) {
		// 夹具费用为本币（汇率 1），另建一条美元费用覆盖外币行。
		first := fixture.createUnbilledFee("bulk-party-1")
		second := createBulkFee(t, data, orderID, fixture.partnerID, "bulk-party-2", "USD", "7.00000000", "700.00000000", orderfeeent.ExchangeRateSourceSYSTEM)
		snapshots := map[uuid.UUID][2]string{
			first:  {"1.00000000", "100.00000000"},
			second: {"7.00000000", "700.00000000"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := usecase.BulkUpdate(ctx, organizationID, actor, orderID, targetsOf(first, second), &newParty.ID, nil); err != nil {
			t.Fatalf("批量改结算单位失败: %v", err)
		}
		for feeID, snapshot := range snapshots {
			fee, err := data.db.OrderFee.Query().Where(orderfeeent.IDEQ(feeID)).WithSettlementParty().Only(context.Background())
			if err != nil {
				t.Fatalf("读取费用 %s: %v", feeID, err)
			}
			if fee.SettlementPartyID != newParty.ID || fee.Version != 2 {
				t.Fatalf("费用 %s 结算单位/版本未更新: party=%s version=%d", feeID, fee.SettlementPartyID, fee.Version)
			}
			party, edgeErr := fee.Edges.SettlementPartyOrErr()
			if edgeErr != nil || party.LegalName != newParty.LegalName {
				t.Fatalf("费用 %s 应经结算单位关联取到新法定名称: %v", feeID, edgeErr)
			}
			if fee.ExchangeRate != snapshot[0] || fee.BaseCurrencyAmount != snapshot[1] || fee.TotalAmount != "100.00000000" {
				t.Fatalf("批量改结算单位不得改动金额与汇率: rate=%s base=%s", fee.ExchangeRate, fee.BaseCurrencyAmount)
			}
		}
		audits, auditErr := data.db.AuditLog.Query().
			Where(auditlogent.ActionEQ("order.fee.update"), auditlogent.OrganizationIDEQ(organizationID)).
			All(context.Background())
		if auditErr != nil {
			t.Fatalf("读取批量修改审计: %v", auditErr)
		}
		bulkAudits := 0
		for _, event := range audits {
			if strings.Contains(string(event.Details), "fee.bulk") && strings.Contains(string(event.Details), newParty.ID.String()) {
				bulkAudits++
			}
		}
		if bulkAudits != 2 {
			t.Fatalf("批量改结算单位应逐行写入审计，实际 %d 条", bulkAudits)
		}
	})

	t.Run("批量改费用时间重解析系统汇率并保留手工汇率", func(t *testing.T) {
		systemFee := createBulkFee(t, data, orderID, fixture.partnerID, "bulk-date-system", "USD", "7.00000000", "700.00000000", orderfeeent.ExchangeRateSourceSYSTEM)
		manualFee := createBulkFee(t, data, orderID, fixture.partnerID, "bulk-date-manual", "USD", "6.50000000", "650.00000000", orderfeeent.ExchangeRateSourceMANUAL)
		setting, settingErr := data.db.ExchangeRateSetting.Create().
			SetOrganizationID(organizationID).
			SetFromCurrency("USD").
			SetToCurrency("CNY").
			SetEffectiveFrom(time.Date(2026, 9, 1, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())).
			SetRate("7.25000000").
			SetIsActive(true).
			Save(context.Background())
		if settingErr != nil {
			t.Fatalf("创建新周期汇率失败: %v", settingErr)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		newDate := "2026-09-15 10:30"
		if err := usecase.BulkUpdate(ctx, organizationID, actor, orderID, targetsOf(systemFee, manualFee), nil, &newDate); err != nil {
			t.Fatalf("批量改费用时间失败: %v", err)
		}
		system, err := data.db.OrderFee.Get(context.Background(), systemFee)
		if err != nil {
			t.Fatalf("读取系统汇率费用: %v", err)
		}
		if system.ExpenseDate != newDate {
			t.Fatalf("系统汇率行费用时间 = %s，期望 %s", system.ExpenseDate, newDate)
		}
		if system.ExchangeRate != "7.25000000" || system.ExchangeRateSource == orderfeeent.ExchangeRateSourceMANUAL || system.ExchangeRateSettingID == nil || *system.ExchangeRateSettingID != setting.ID {
			t.Fatalf("系统汇率行应按新日期重解析: rate=%s source=%s setting=%s", system.ExchangeRate, system.ExchangeRateSource, system.ExchangeRateSettingID)
		}
		if system.ExchangeRateDate != "2026-09-15" {
			t.Fatalf("汇率日期应取新日期的日期部分，实际 %s", system.ExchangeRateDate)
		}
		if system.BaseCurrencyAmount != "725.00000000" {
			t.Fatalf("折本币应=总额 100×新汇率 7.25，实际 %s", system.BaseCurrencyAmount)
		}
		if system.TotalAmount != "100.00000000" || system.Quantity != "1.0000" || system.UnitPrice != "100.0000" {
			t.Fatalf("批量改费用时间不得改动金额、数量与单价: %+v", system)
		}
		manual, err := data.db.OrderFee.Get(context.Background(), manualFee)
		if err != nil {
			t.Fatalf("读取手工汇率费用: %v", err)
		}
		if manual.ExchangeRate != "6.50000000" || manual.ExchangeRateSource != orderfeeent.ExchangeRateSourceMANUAL || manual.BaseCurrencyAmount != "650.00000000" {
			t.Fatalf("手工汇率应保持原值与来源: rate=%s source=%s base=%s", manual.ExchangeRate, manual.ExchangeRateSource, manual.BaseCurrencyAmount)
		}
		if manual.ExpenseDate != newDate || manual.ExchangeRateDate != "2026-09-15" {
			t.Fatalf("手工汇率行费用时间与汇率日期应随新日期更新: expense=%s rate_date=%s", manual.ExpenseDate, manual.ExchangeRateDate)
		}
	})

	t.Run("批量删除未建账费用并级联清理标签", func(t *testing.T) {
		first := fixture.createUnbilledFee("bulk-del-1")
		second := fixture.createUnbilledFee("bulk-del-2")
		resource, resourceErr := data.db.EnterpriseResource.Create().
			SetOrganizationID(organizationID).
			SetResourceType(enterpriseresourceent.ResourceTypeTAG).
			SetShortName("批量删除级联标签-" + fixture.suffix).
			Save(context.Background())
		if resourceErr != nil {
			t.Fatalf("创建标签资源: %v", resourceErr)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = data.db.EnterpriseResource.DeleteOneID(resource.ID).Exec(ctx)
		})
		if _, err := data.db.OrderFeeEnterpriseTag.Create().
			SetOrganizationID(organizationID).
			SetOrderFeeID(first).
			SetTagResourceID(resource.ID).
			Save(context.Background()); err != nil {
			t.Fatalf("创建费用标签关联: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := usecase.BulkRemove(ctx, organizationID, actor, orderID, targetsOf(first, second), "批量清理"); err != nil {
			t.Fatalf("批量删除失败: %v", err)
		}
		count, err := data.db.OrderFee.Query().Where(orderfeeent.IDIn(first, second)).Count(context.Background())
		if err != nil || count != 0 {
			t.Fatalf("批量删除后费用仍存在: count=%d err=%v", count, err)
		}
		links, err := data.db.OrderFeeEnterpriseTag.Query().Where(orderfeeenterprisetagent.OrderFeeIDEQ(first)).Count(context.Background())
		if err != nil || links != 0 {
			t.Fatalf("费用标签关联应级联清理: count=%d err=%v", links, err)
		}
		audits, auditErr := data.db.AuditLog.Query().
			Where(auditlogent.ActionEQ("order.fee.delete"), auditlogent.OrganizationIDEQ(organizationID)).
			All(context.Background())
		if auditErr != nil {
			t.Fatalf("读取批量删除审计: %v", auditErr)
		}
		bulkAudits := 0
		for _, event := range audits {
			if strings.Contains(string(event.Details), "fee.bulk") && strings.Contains(string(event.Details), "批量清理") {
				bulkAudits++
			}
		}
		if bulkAudits != 2 {
			t.Fatalf("批量删除应逐行写入审计，实际 %d 条", bulkAudits)
		}
	})

	t.Run("任一行版本冲突整批零写入", func(t *testing.T) {
		first := fixture.createUnbilledFee("bulk-conflict-1")
		second := fixture.createUnbilledFee("bulk-conflict-2")
		targets := targetsOf(first, second)
		targets[1].ExpectedVersion = 99

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := usecase.BulkUpdate(ctx, organizationID, actor, orderID, targets, &newParty.ID, nil)
		if !errors.Is(err, biz.ErrOrderFeeVersionConflict) {
			t.Fatalf("版本冲突错误 = %v，期望 ErrOrderFeeVersionConflict", err)
		}
		if !strings.Contains(err.Error(), second.String()) {
			t.Fatalf("版本冲突错误应携带费用定位: %v", err)
		}
		for _, feeID := range []uuid.UUID{first, second} {
			fee, queryErr := data.db.OrderFee.Get(context.Background(), feeID)
			if queryErr != nil {
				t.Fatalf("读取费用 %s: %v", feeID, queryErr)
			}
			if fee.Version != 1 || fee.SettlementPartyID == newParty.ID {
				t.Fatalf("整批回滚后费用 %s 仍被改动: version=%d", feeID, fee.Version)
			}
		}
	})

	t.Run("已进账单行混入修改整批拒绝", func(t *testing.T) {
		billed := fixture.createUnbilledFee("bulk-billed")
		free := fixture.createUnbilledFee("bulk-billed-free")
		bill, err := billUsecase.Create(context.Background(), organizationID, actor, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{billed}, BillDate: financeBillIntegrationDate,
			IdempotencyKey: "bill-bulk-update-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("建立账单失败: %v", err)
		}
		// 建账会推进费用版本与状态，按事务后事实组装目标。
		billedFee, queryErr := data.db.OrderFee.Get(context.Background(), billed)
		if queryErr != nil {
			t.Fatalf("读取已建账费用: %v", queryErr)
		}
		targets := []biz.OrderFeeBulkTarget{{FeeID: billed, ExpectedVersion: billedFee.Version}, {FeeID: free, ExpectedVersion: 1}}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		updateErr := usecase.BulkUpdate(ctx, organizationID, actor, orderID, targets, &newParty.ID, nil)
		if !errors.Is(updateErr, biz.ErrBilledFeeFieldForbidden) {
			t.Fatalf("已建账行修改错误 = %v，期望 ErrBilledFeeFieldForbidden", updateErr)
		}
		if !strings.Contains(updateErr.Error(), billed.String()) {
			t.Fatalf("已建账行错误应携带费用定位: %v", updateErr)
		}
		freeFee, queryErr := data.db.OrderFee.Get(context.Background(), free)
		if queryErr != nil || freeFee.Version != 1 || freeFee.SettlementPartyID == newParty.ID {
			t.Fatalf("混入已建账行时其他费用不得被改动: version=%d err=%v", freeFee.Version, queryErr)
		}

		// 已建账行删除同样整批拒绝，错误携带账单号便于界面定位。
		removeErr := usecase.BulkRemove(ctx, organizationID, actor, orderID, targets, "")
		if !errors.Is(removeErr, biz.ErrOrderFeeBillOccupied) {
			t.Fatalf("占用行删除错误 = %v，期望 ErrOrderFeeBillOccupied", removeErr)
		}
		if !strings.Contains(removeErr.Error(), billed.String()) || !strings.Contains(removeErr.Error(), bill.BillNo) {
			t.Fatalf("占用错误应携带费用与账单号: %v", removeErr)
		}
		exists, existErr := data.db.OrderFee.Query().Where(orderfeeent.IDIn(billed, free)).Count(context.Background())
		if existErr != nil || exists != 2 {
			t.Fatalf("整批回滚后费用应全部保留: count=%d err=%v", exists, existErr)
		}
	})

	t.Run("越订单费用目标整批拒绝", func(t *testing.T) {
		otherOrder, orderErr := data.db.Order.Create().
			SetIdempotencyKey(uuid.NewString()).
			SetOrganizationID(organizationID).
			SetOrderNo("SE-BULK-" + fixture.suffix).
			SetCustomerID(fixture.partnerID).
			SetBusinessType(orderent.BusinessTypeSE).
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			Save(context.Background())
		if orderErr != nil {
			t.Fatalf("创建越订单目标订单: %v", orderErr)
		}
		otherFee := createBulkFee(t, data, otherOrder.ID, fixture.partnerID, "bulk-cross-order", "CNY", "1.00000000", "100.00000000", orderfeeent.ExchangeRateSourceSYSTEM)
		ownFee := fixture.createUnbilledFee("bulk-cross-own")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// 用例层按订单归属预检拒绝；直接调用仓储同样在行锁内复核归属。
		err := usecase.BulkUpdate(ctx, organizationID, actor, orderID, []biz.OrderFeeBulkTarget{{FeeID: otherFee, ExpectedVersion: 1}, {FeeID: ownFee, ExpectedVersion: 1}}, &newParty.ID, nil)
		if !errors.Is(err, biz.ErrOrderFeeNotFound) || !strings.Contains(err.Error(), otherFee.String()) {
			t.Fatalf("越订单目标应携带定位拒绝，实际 %v", err)
		}
		repoErr := orderFeeRepo.BulkUpdate(ctx, organizationID, orderID, &biz.OrderFeeBulkUpdateInput{
			Targets:           []biz.OrderFeeBulkTarget{{FeeID: otherFee, ExpectedVersion: 1}},
			SettlementPartyID: &newParty.ID,
		}, map[uuid.UUID]*biz.AuditEvent{})
		if !errors.Is(repoErr, biz.ErrOrderFeeNotFound) {
			t.Fatalf("仓储层越订单目标应拒绝，实际 %v", repoErr)
		}
		otherFeeRow, queryErr := data.db.OrderFee.Get(context.Background(), otherFee)
		if queryErr != nil || otherFeeRow.Version != 1 {
			t.Fatalf("越订单费用不应被改动: version=%d err=%v", otherFeeRow.Version, queryErr)
		}
	})

	t.Run("已作废行混入批量维护整批拒绝", func(t *testing.T) {
		cancelled := fixture.createUnbilledFee("bulk-cancelled")
		free := fixture.createUnbilledFee("bulk-cancelled-free")
		if _, err := data.db.OrderFee.UpdateOneID(cancelled).
			SetStatus(orderfeeent.StatusCANCELLED).
			SetCancelledAt(time.Now()).
			SetCancelledBy(actor).
			SetCancellationReason("批量维护测试作废").
			Save(context.Background()); err != nil {
			t.Fatalf("置为已作废费用失败: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		updateErr := usecase.BulkUpdate(ctx, organizationID, actor, orderID, targetsOf(cancelled, free), &newParty.ID, nil)
		if !errors.Is(updateErr, biz.ErrOrderFeeInvalidTransition) || !strings.Contains(updateErr.Error(), cancelled.String()) {
			t.Fatalf("已作废行修改应携带定位拒绝，实际 %v", updateErr)
		}
		removeErr := usecase.BulkRemove(ctx, organizationID, actor, orderID, targetsOf(cancelled, free), "")
		if !errors.Is(removeErr, biz.ErrOrderFeeInvalidTransition) {
			t.Fatalf("已作废行删除应整批拒绝，实际 %v", removeErr)
		}
		freeFee, queryErr := data.db.OrderFee.Get(context.Background(), free)
		if queryErr != nil || freeFee.Version != 1 {
			t.Fatalf("混入已作废行时其他费用不得被改动: version=%d err=%v", freeFee.Version, queryErr)
		}
	})
}
