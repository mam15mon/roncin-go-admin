package data

import (
	"context"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestFinanceCustomSettingCompanyIsolation(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor := client.User.Create().SetDisplayName("策略测试操作人").SetEnabled(true).SaveX(ctx)
	a := client.Organization.Create().SetCode("POLICY-A").SetName("公司甲").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	b := client.Organization.Create().SetCode("POLICY-B").SetName("公司乙").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	uc := biz.NewFinanceCustomSettingUsecase(NewFinanceCustomSettingRepo(data))
	policy, err := uc.UpdateBilledFeeEditPolicy(ctx, a.ID, actor.ID, &biz.BilledFeeEditPolicy{Enabled: true, EditableFields: []biz.BilledFeeEditableField{biz.BilledFeeFieldFeeName}}, 0)
	if err != nil || !policy.Enabled || policy.OrganizationID != a.ID {
		t.Fatalf("保存甲公司策略: %#v %v", policy, err)
	}
	if _, err = uc.UpdateCreditLimitControlPolicy(ctx, a.ID, actor.ID, false, policy.Version); err != nil {
		t.Fatal(err)
	}
	other, err := uc.GetBilledFeeEditPolicy(ctx, b.ID)
	if err != nil || other.Enabled || other.Version != 0 || other.OrganizationID != b.ID {
		t.Fatalf("乙公司应保持禁止编辑默认值: %#v %v", other, err)
	}
	credit, err := uc.GetCreditLimitControlPolicy(ctx, b.ID)
	if err != nil || !credit.AllowSelectionWhenCreditExceeded {
		t.Fatalf("乙公司应保持仅提醒默认值: %#v %v", credit, err)
	}
	active, err := uc.IsCreditLimitInterventionActive(ctx, a.ID)
	if err != nil || !active {
		t.Fatalf("甲公司应独立拦截: %v %v", active, err)
	}
	if _, err = uc.UpdateCreditLimitControlPolicy(ctx, a.ID, actor.ID, true, policy.Version); err != biz.ErrFinanceCustomSettingConflict {
		t.Fatalf("过期版本应拒绝: %v", err)
	}
	if _, err = uc.UpdateCreditLimitControlPolicy(ctx, b.ID, actor.ID, true, 0); err != nil {
		t.Fatal(err)
	}
	unchanged, err := uc.GetBilledFeeEditPolicy(ctx, a.ID)
	if err != nil || !unchanged.Enabled || len(unchanged.EditableFields) != 1 {
		t.Fatalf("乙公司更新不能改变甲公司: %#v %v", unchanged, err)
	}
}
