package biz

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// orderFeeBulkRateRepoStub 逐行记录批量改时间场景的汇率解析调用。
type orderFeeBulkRateRepoStub struct {
	ExchangeRateRepo
	rate     decimal.Decimal
	missing  bool
	requests []string
}

func (r *orderFeeBulkRateRepoStub) ResolveContext(context.Context, uuid.UUID) (*ExchangeRateContext, error) {
	return &ExchangeRateContext{OwnerOrganizationID: uuid.Must(uuid.NewV7()), BaseCurrency: "CNY"}, nil
}

func (r *orderFeeBulkRateRepoStub) ResolveRate(_ context.Context, _ uuid.UUID, _ OrderFeeDirection, from, _, rateDate string) (ResolvedRate, error) {
	r.requests = append(r.requests, from+"@"+rateDate)
	if r.missing {
		return ResolvedRate{}, ErrExchangeRateMissing
	}
	return ResolvedRate{Rate: r.rate, Source: ExchangeRateSourceSystem}, nil
}

// orderFeeBulkRepoStub 捕获批量用例提交给仓储的计划与逐行审计。
type orderFeeBulkRepoStub struct {
	OrderFeeRepo
	fees         []*OrderFee
	bulkUpdate   *OrderFeeBulkUpdateInput
	updateAudits map[uuid.UUID]*AuditEvent
	removeTarget []OrderFeeBulkTarget
	removeAudits map[uuid.UUID]*AuditEvent
}

func (r *orderFeeBulkRepoStub) List(context.Context, uuid.UUID, uuid.UUID) ([]*OrderFee, error) {
	return r.fees, nil
}

func (r *orderFeeBulkRepoStub) BulkUpdate(_ context.Context, _, _ uuid.UUID, input *OrderFeeBulkUpdateInput, audits map[uuid.UUID]*AuditEvent) error {
	r.bulkUpdate = input
	r.updateAudits = audits
	return nil
}

func (r *orderFeeBulkRepoStub) BulkRemove(_ context.Context, _, _ uuid.UUID, targets []OrderFeeBulkTarget, audits map[uuid.UUID]*AuditEvent) error {
	r.removeTarget = targets
	r.removeAudits = audits
	return nil
}

func bulkOrderFeeForTest(id uuid.UUID, direction OrderFeeDirection, currency, source string) *OrderFee {
	fee := validOrderFeeForTest()
	fee.ID = id
	fee.Direction = direction
	fee.Currency = currency
	fee.Version = 3
	fee.FeeCode = "OCEAN_FREIGHT"
	fee.FeeName = "海运费"
	fee.SettlementPartyName = "原结算单位"
	fee.TotalAmount = decimal.RequireFromString("100")
	fee.ExchangeRateSource = source
	fee.ExchangeRate = decimal.RequireFromString("7.00000000")
	fee.ExchangeRateDate = "2026-08-30"
	fee.BaseCurrency = "CNY"
	fee.BaseCurrencyAmount = decimal.RequireFromString("700.00000000")
	return fee
}

func bulkTargetsForTest(fees ...*OrderFee) []OrderFeeBulkTarget {
	targets := make([]OrderFeeBulkTarget, 0, len(fees))
	for _, fee := range fees {
		targets = append(targets, OrderFeeBulkTarget{FeeID: fee.ID, ExpectedVersion: fee.Version})
	}
	return targets
}

func newOrderFeeBulkUsecase(repo OrderFeeRepo, rateRepo ExchangeRateRepo) *OrderFeeUsecase {
	return NewOrderFeeUsecase(repo, NewExchangeRateUsecase(rateRepo, nil), nil, newReminderModeCreditControl(), nil)
}

// TestBulkOrderFeeErrorKeepsSentinelMatch 断言批量错误保持单条哨兵错误的
// reason/HTTP 语义，errors.Is 仍可命中。
func TestBulkOrderFeeErrorKeepsSentinelMatch(t *testing.T) {
	feeID := uuid.Must(uuid.NewV7())
	err := BulkOrderFeeError(ErrOrderFeeVersionConflict, feeID, "已被其他操作人修改，请刷新后重试")
	if !errors.Is(err, ErrOrderFeeVersionConflict) {
		t.Fatalf("批量版本冲突错误应命中哨兵错误，实际 %v", err)
	}
	if !strings.Contains(err.Error(), feeID.String()) {
		t.Fatalf("批量错误应携带费用定位: %v", err)
	}
}

// TestOrderFeeBulkUpdateSettlementPartySubmitsPlanAndAudits 断言批量改结算单位
// 只提交结算单位目标值（不触发汇率解析），逐行审计记录 from→to 与批量标识。
func TestOrderFeeBulkUpdateSettlementPartySubmitsPlanAndAudits(t *testing.T) {
	receivable := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", ExchangeRateSourceSystem)
	payable := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "CNY", ExchangeRateSourceSystem)
	repo := &orderFeeBulkRepoStub{fees: []*OrderFee{receivable, payable}}
	rateRepo := &orderFeeBulkRateRepoStub{rate: decimal.RequireFromString("7.2")}
	usecase := newOrderFeeBulkUsecase(repo, rateRepo)
	newParty := uuid.Must(uuid.NewV7())

	if err := usecase.BulkUpdate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), bulkTargetsForTest(receivable, payable), &newParty); err != nil {
		t.Fatalf("批量改结算单位失败: %v", err)
	}
	if repo.bulkUpdate == nil {
		t.Fatal("仓储未收到批量修改计划")
	}
	if repo.bulkUpdate.SettlementPartyID == nil || *repo.bulkUpdate.SettlementPartyID != newParty {
		t.Fatalf("批量计划应携带目标结算单位: %+v", repo.bulkUpdate.SettlementPartyID)
	}
	if len(rateRepo.requests) != 0 {
		t.Fatalf("批量改结算单位不应解析汇率，实际 %v", rateRepo.requests)
	}
	if len(repo.updateAudits) != 2 {
		t.Fatalf("逐行审计缺失: %d", len(repo.updateAudits))
	}
	audit := repo.updateAudits[receivable.ID]
	if audit.Action != "order.fee.update" || audit.Details["fee.bulk"] != "true" {
		t.Fatalf("批量修改审计事件不正确: %+v", audit)
	}
	if audit.Details["fee.settlement_party.from"] != receivable.SettlementPartyID.String() || audit.Details["fee.settlement_party.to"] != newParty.String() {
		t.Fatalf("结算单位 from→to 缺失: %+v", audit.Details)
	}
	if _, exists := audit.Details["fee.expense_date.to"]; exists {
		t.Fatalf("改结算单位审计不应记录费用时间变化: %+v", audit.Details)
	}
}

func TestOrderFeeBulkUpdateRejectsInvalidRequests(t *testing.T) {
	fee := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", ExchangeRateSourceSystem)
	party := uuid.Must(uuid.NewV7())
	organizationID, actorID, orderID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	tests := []struct {
		name    string
		targets []OrderFeeBulkTarget
		party   *uuid.UUID
	}{
		{name: "空目标", targets: nil, party: &party},
		{name: "重复费用", targets: []OrderFeeBulkTarget{{FeeID: fee.ID, ExpectedVersion: 1}, {FeeID: fee.ID, ExpectedVersion: 1}}, party: &party},
		{name: "零版本", targets: []OrderFeeBulkTarget{{FeeID: fee.ID}}, party: &party},
		{name: "缺失目标值", targets: bulkTargetsForTest(fee)},
		{name: "空结算单位", targets: bulkTargetsForTest(fee), party: &uuid.Nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &orderFeeBulkRepoStub{fees: []*OrderFee{fee}}
			usecase := newOrderFeeBulkUsecase(repo, &orderFeeBulkRateRepoStub{rate: decimal.NewFromInt(1)})
			if err := usecase.BulkUpdate(context.Background(), organizationID, actorID, orderID, test.targets, test.party); err != ErrOrderFeeInvalidArgument {
				t.Fatalf("%s应被结构层拒绝，实际错误为 %v", test.name, err)
			}
			if repo.bulkUpdate != nil {
				t.Fatalf("%s不应触达仓储", test.name)
			}
		})
	}
}

// TestOrderFeeBulkUpdateRejectsUnknownFee 断言目标费用不属于该订单时整批拒绝
// 且错误携带费用定位。
func TestOrderFeeBulkUpdateRejectsUnknownFee(t *testing.T) {
	known := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", ExchangeRateSourceSystem)
	unknown := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", ExchangeRateSourceSystem)
	repo := &orderFeeBulkRepoStub{fees: []*OrderFee{known}}
	usecase := newOrderFeeBulkUsecase(repo, &orderFeeBulkRateRepoStub{rate: decimal.NewFromInt(1)})
	party := uuid.Must(uuid.NewV7())

	err := usecase.BulkUpdate(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), bulkTargetsForTest(known, unknown), &party)
	if !errors.Is(err, ErrOrderFeeNotFound) {
		t.Fatalf("越订单/未知费用应返回 NotFound，实际 %v", err)
	}
	if !strings.Contains(err.Error(), unknown.ID.String()) {
		t.Fatalf("错误应携带费用定位: %v", err)
	}
	if repo.bulkUpdate != nil {
		t.Fatal("未知费用不应触达仓储")
	}
}

// TestOrderFeeBulkRemoveBuildsAuditsAndForwardsTargets 断言批量删除转发目标并
// 逐行写审计（原因与批量标识），删除超长原因被结构层拒绝。
func TestOrderFeeBulkRemoveBuildsAuditsAndForwardsTargets(t *testing.T) {
	first := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeeReceivable, "USD", ExchangeRateSourceSystem)
	second := bulkOrderFeeForTest(uuid.Must(uuid.NewV7()), OrderFeePayable, "CNY", ExchangeRateSourceSystem)
	repo := &orderFeeBulkRepoStub{fees: []*OrderFee{first, second}}
	usecase := newOrderFeeBulkUsecase(repo, &orderFeeBulkRateRepoStub{})

	if err := usecase.BulkRemove(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), bulkTargetsForTest(first, second), "批量清理测试费用"); err != nil {
		t.Fatalf("批量删除失败: %v", err)
	}
	if len(repo.removeTarget) != 2 {
		t.Fatalf("仓储应收到全部目标: %d", len(repo.removeTarget))
	}
	if len(repo.removeAudits) != 2 {
		t.Fatalf("逐行删除审计缺失: %d", len(repo.removeAudits))
	}
	audit := repo.removeAudits[first.ID]
	if audit.Action != "order.fee.delete" || audit.Details["reason"] != "批量清理测试费用" || audit.Details["fee.bulk"] != "true" {
		t.Fatalf("批量删除审计不正确: %+v", audit)
	}

	longReason := strings.Repeat("长", 501)
	if err := usecase.BulkRemove(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), bulkTargetsForTest(first), longReason); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("超长删除原因应被拒绝，实际错误为 %v", err)
	}
	if err := usecase.BulkRemove(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), nil, ""); err != ErrOrderFeeInvalidArgument {
		t.Fatalf("空目标应被拒绝，实际错误为 %v", err)
	}
}
