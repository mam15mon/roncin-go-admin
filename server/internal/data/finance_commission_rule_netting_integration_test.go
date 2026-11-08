package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// 从业务创建方案建立员工分配，覆盖真实候选、预览、草稿与确认，避免夹具直接写规则掩盖缺口。
func TestCommissionRuleToNettingConfirmationPostgres(t *testing.T) {
	f := newCommissionNettingPostgresFixtureWithRule(t, false)
	ctx := context.Background()
	// 客商编码选填，合法空值必须能进入提成快照。
	if _, err := f.data.db.Partner.UpdateOneID(f.customerID).ClearCode().Save(ctx); err != nil {
		t.Fatal(err)
	}
	uc := f.newUsecase()
	filter := biz.CommissionCandidateFilter{NettingID: f.nettingID, Page: 1, PageSize: 20}
	candidates, err := uc.ListCandidates(ctx, f.organizationID, filter)
	if err != nil || candidates.Total != 0 {
		t.Fatalf("无方案应无员工候选: %+v %v", candidates, err)
	}
	source, err := f.newNettingUsecase().Get(ctx, []uuid.UUID{f.organizationID}, f.nettingID)
	if err != nil || source.ConfirmedAt == nil {
		t.Fatalf("读取确认来源: %v", err)
	}
	date := source.ConfirmedAt.UTC().Format("2006-01-02")
	// Today 显式使用来源业务日期，避免测试执行时间跨本地午夜影响夹具；生产继续使用统一业务日。
	rule, err := uc.CreateRule(ctx, f.organizationID, f.actorID, biz.CreateCommissionRuleInput{
		Name: "对冲验收方案-" + f.suffix, PersonnelRole: biz.CommissionRoleSales,
		CalculationBasis: biz.CommissionBasisRealizedProfit, RatePercent: decimal.NewFromInt(10),
		EffectiveFrom: &date, EmployeeIDs: []uuid.UUID{f.employeeID}, Enabled: true, Today: date,
	})
	if err != nil {
		t.Fatalf("业务创建方案: %v", err)
	}
	if len(rule.Assignments) != 1 || rule.Assignments[0].EmployeeID != f.employeeID {
		t.Fatalf("方案未建立员工分配: %+v", rule)
	}
	candidates, err = uc.ListCandidates(ctx, f.organizationID, filter)
	if err != nil || candidates.Total != 1 || candidates.Items[0].RuleID != rule.ID {
		t.Fatalf("方案建立后候选不符: %+v %v", candidates, err)
	}
	preview, err := uc.Preview(ctx, f.organizationID, uuid.Nil, f.nettingID, f.employeeID, biz.CommissionRoleSales)
	if err != nil {
		t.Fatal(err)
	}
	if preview.RealizedRevenue.StringFixed(8) != "800.00000000" || preview.AllocatedCost.StringFixed(8) != "300.00000000" || preview.RealizedProfit.StringFixed(8) != "500.00000000" || preview.CommissionAmount.StringFixed(8) != "50.00000000" {
		t.Fatalf("预览金额错误: %+v", preview)
	}
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.input("business-rule"))
	if err != nil || created.Status != biz.CommissionDraft || !created.CommissionAmount.Equal(preview.CommissionAmount) {
		t.Fatalf("生成草稿不符: %+v %v", created, err)
	}
	confirmed, err := uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil || confirmed.Status != biz.CommissionConfirmed || confirmed.Version != created.Version+1 {
		t.Fatalf("确认草稿不符: %+v %v", confirmed, err)
	}
	for _, line := range confirmed.Lines {
		if line.CustomerCode != "" {
			t.Fatalf("空客商编码应如实保留: %q", line.CustomerCode)
		}
	}
	if confirmed.ConfirmedAt == nil || confirmed.ConfirmedAt.After(time.Now()) {
		t.Fatal("确认时间未保存")
	}
	if _, err = uc.Create(ctx, f.organizationID, f.actorID, f.input("duplicate-rule")); err == nil {
		t.Fatal("同来源同员工重复生成应拒绝")
	}
	if _, err = uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version); err == nil {
		t.Fatal("旧版本重复确认应拒绝")
	}
}
