package data

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	adjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	"github.com/shopspring/decimal"
)

// 税率更改权限的真实 PostgreSQL 验收：费用设置目录税率为 6%，显式税率须有权限且不被目录覆盖。
func taxRateTestFee(f *feeSupplementPostgresFixture, amount int64, taxRate string) *biz.OrderFee {
	input := correctionTestInput(f, "ADD", uuid.NewString(), nil, amount).After
	input.IdempotencyKey = uuid.NewString()
	input.TaxRateOverride = nil
	if taxRate != "" {
		rate := decimal.RequireFromString(taxRate)
		input.TaxRateOverride = &rate
	}
	return input
}

func TestOrderFeeTaxRatePermissionPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	uc := normalFeeUsecaseForRollback(f)
	o := f.createOrder("TAXRATE-" + f.suffix)

	// 无权且税率与目录不同：拒绝，且不产生费用。
	if _, err := uc.Add(f.ctx, f.organizationID, f.approverID, o.ID, taxRateTestFee(f, 106, "13"), false, false); err != biz.ErrOrderFeeTaxRateForbidden {
		t.Fatalf("无权更改税率应拒绝: %v", err)
	}
	if fees, err := NewOrderFeeRepo(f.data).List(f.ctx, f.organizationID, o.ID); err != nil || len(fees) != 0 {
		t.Fatalf("拒绝后不得落库: %d %v", len(fees), err)
	}
	// 无权但显式税率等于目录税率：不构成更改。
	if _, err := uc.Add(f.ctx, f.organizationID, f.approverID, o.ID, taxRateTestFee(f, 106, "6"), false, false); err != nil {
		t.Fatalf("税率与目录一致不应要求权限: %v", err)
	}
	// 有权新增含税 106、税率 0：税率保持 0，不被目录 6% 覆盖，未税等于含税。
	zero, err := uc.Add(f.ctx, f.organizationID, f.approverID, o.ID, taxRateTestFee(f, 106, "0"), false, true)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, zero.ID)
	if err != nil || stored.TaxRate == nil || !stored.TaxRate.IsZero() || !stored.NetAmount.Equal(decimal.NewFromInt(106)) || !stored.TaxAmount.IsZero() {
		t.Fatalf("显式税率 0 应落库且金额按 0 税率计算: %+v %v", stored, err)
	}
	// 目录税率 6%：106 含税 → 未税 100，本币未税字段与统一折算入口一致。
	six, err := uc.Add(f.ctx, f.organizationID, f.approverID, o.ID, taxRateTestFee(f, 106, ""), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if !six.NetAmount.Equal(decimal.NewFromInt(100)) || !six.BaseCurrencyNetAmount().Equal(biz.CommissionNetBase(six.BaseCurrencyAmount, six.NetAmount, six.TotalAmount)) || !six.BaseCurrencyNetAmount().Equal(decimal.NewFromInt(100)) {
		t.Fatalf("含税 106 / 未税 100 本币未税应为 100: %+v net=%s", six, six.BaseCurrencyNetAmount())
	}

	// 修改：无权改税率被拒；未提交税率时保持已存税率；有权修改后金额重算。
	update := *zero
	rate := decimal.RequireFromString("13")
	update.TaxRateOverride = &rate
	if _, err = uc.Update(f.ctx, f.organizationID, f.approverID, o.ID, zero.ID, &update, false, false); err != biz.ErrOrderFeeTaxRateForbidden {
		t.Fatalf("无权修改税率应拒绝: %v", err)
	}
	keep := *stored
	keep.Note = new(string)
	kept, err := uc.Update(f.ctx, f.organizationID, f.approverID, o.ID, zero.ID, &keep, false, false)
	if err != nil || kept.TaxRate == nil || !kept.TaxRate.IsZero() {
		t.Fatalf("未显式提交税率时不得被目录税率覆盖: %+v %v", kept, err)
	}
	update = *kept
	update.TaxRateOverride = &rate
	changed, err := uc.Update(f.ctx, f.organizationID, f.approverID, o.ID, zero.ID, &update, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if changed.TaxRate == nil || !changed.TaxRate.Equal(rate) || !changed.NetAmount.Equal(decimal.RequireFromString("93.80530973")) {
		t.Fatalf("税率 13%% 后未税应为 93.80530973: %+v", changed)
	}
	// 审计记录前后值与操作者。
	logs, err := f.data.db.AuditLog.Query().All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range logs {
		details := map[string]string{}
		if row.Action != "order.fee.update" || json.Unmarshal(row.Details, &details) != nil {
			continue
		}
		if details["fee.tax_rate_before"] == "0.00" && details["fee.tax_rate_after"] == "13.00" && details["fee.id"] == zero.ID.String() {
			found = row.UserID != nil && *row.UserID == f.approverID
		}
	}
	if !found {
		t.Fatal("税率变化缺少含前后值与操作者的审计")
	}

	// 已建账：即使有权也拒绝带税率变化的修改。
	createOccupiedTestFeeBill(t, f, o, six)
	billed := *six
	billed.TaxRateOverride = &rate
	if _, err = uc.Update(f.ctx, f.organizationID, f.approverID, o.ID, six.ID, &billed, false, true); err != biz.ErrOrderFeeBillOccupied {
		t.Fatalf("已建账费用带税率变化应拒绝: %v", err)
	}
}

func TestOrderFeeTaxRateReconcilesCommissionPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	o := f.createOrder("TAXCOMM-" + f.suffix)
	f.approveInitialFees(o)
	parent := f.createConfirmedCommissionWithSnapshot(o, "60", "10", "1000", "400")
	cuc := biz.NewCommissionUsecase(NewCommissionRepo(f.data), nil, f.data)
	if _, err := markCommissionPaidForTest(f.ctx, cuc, f.organizationID, f.approverID, parent.ID, parent.Version); err != nil {
		t.Fatal(err)
	}
	unlockFeeTestOrder(t, f, o)
	assertNet := func(want string) {
		t.Helper()
		rows, err := f.data.db.FinanceCommissionAdjustment.Query().Where(adjustmentent.CommissionIDEQ(parent.ID), adjustmentent.StatusIn(adjustmentent.StatusCONFIRMED, adjustmentent.StatusPAID)).All(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		net := decimal.RequireFromString("60")
		for _, row := range rows {
			amount := decimal.RequireFromString(row.Amount)
			if row.Direction == adjustmentent.DirectionDECREASE {
				net = net.Sub(amount)
			} else {
				net = net.Add(amount)
			}
		}
		if !net.Equal(decimal.RequireFromString(want)) {
			t.Fatalf("提成净额=%s，预期%s", net, want)
		}
	}
	uc := normalFeeUsecaseForRollback(f)
	// 含税 100、税率 0：未税 100，成本冲减 10 → 净额 50。
	fee, err := uc.Add(f.ctx, f.organizationID, f.approverID, o.ID, taxRateTestFee(f, 100, "0"), false, true)
	if err != nil {
		t.Fatal(err)
	}
	assertNet("50")
	// 改为税率 6%：未税 94.33962264，冲减 9.43396226。
	six := decimal.NewFromInt(6)
	update := *fee
	update.TaxRateOverride = &six
	if _, err = uc.Update(f.ctx, f.organizationID, f.approverID, o.ID, fee.ID, &update, false, true); err != nil {
		t.Fatal(err)
	}
	assertNet("50.56603774")
}
