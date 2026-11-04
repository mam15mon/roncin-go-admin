package data

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
)

func disableCommissionRuleTestMembership(t *testing.T, f *commissionRuleAssignmentFixture, employee uuid.UUID) {
	t.Helper()
	row, err := f.data.db.Membership.Query().Where(membershipent.UserIDEQ(employee), membershipent.OrganizationIDEQ(f.orgID)).Only(f.ctx())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.admin.UpdateUserMembership(f.ctx(), &biz.AdminUserMembership{ID: row.ID, UserID: employee, Enabled: false}, nil, &biz.AuditEvent{Action: "test.membership.disable", Result: "success"}); err != nil {
		t.Fatalf("无当前资格后应允许停用成员: %v", err)
	}
}

func TestCommissionRuleHistoricalMetadataInactiveEmployeesPostgres(t *testing.T) {
	for _, inactive := range []string{"成员停用", "账号停用"} {
		t.Run(inactive, func(t *testing.T) {
			f := newCommissionRuleAssignmentFixture(t)
			today := f.today
			// 模拟三天前创建、两天前正常到期，当前真实成员停用入口也会认可已结束资格。
			f.today = f.dayOffset(-3)
			from, to := f.today, f.dayOffset(1)
			plan := f.createRule("已到期历史方案", biz.CommissionRoleSales, &from, &to, true, []uuid.UUID{f.employees[0]})
			if inactive == "成员停用" {
				disableCommissionRuleTestMembership(t, f, f.employees[0])
			} else if err := f.data.db.User.UpdateOneID(f.employees[0]).SetEnabled(false).Exec(f.ctx()); err != nil {
				t.Fatal(err)
			}
			in := commissionRuleUpdateInput(plan, today)
			in.Name, in.Note = "历史方案说明维护", stringPtr("历史员工状态不影响名称和备注")
			updated, err := f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in)
			if err != nil {
				t.Fatalf("停用历史员工不得阻止纯元数据修改: %v", err)
			}
			if updated.Version != plan.Version+1 || updated.Name != in.Name || optionalStringValue(updated.Note) != *in.Note || optionalStringValue(updated.EffectiveFrom) != from || optionalStringValue(updated.EffectiveTo) != to {
				t.Fatalf("只应更新名称、备注与版本: %+v", updated)
			}
			if _, err := f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in); !errors.Is(err, biz.ErrCommissionRuleConflict) {
				t.Fatalf("历史员工状态不应掩盖版本冲突: %v", err)
			}
			bad := commissionRuleUpdateInput(updated, today)
			bad.EffectiveTo = nil
			if _, err := f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, bad); !errors.Is(err, biz.ErrCommissionRuleEmployeeInvalid) {
				t.Fatalf("扩大资格仍须要求有效员工: %v", err)
			}
			if got := f.getRule(plan.ID); got.Version != updated.Version || optionalStringValue(got.EffectiveTo) != to {
				t.Fatalf("失败扩展不得写入: %+v", got)
			}
		})
	}
}

func TestCommissionRuleInactiveEmployeeMetadataCannotEnablePostgres(t *testing.T) {
	f := newCommissionRuleAssignmentFixture(t)
	from := f.dayOffset(1)
	plan := f.createRule("未启用未来方案", biz.CommissionRoleSales, &from, nil, false, []uuid.UUID{f.employees[0]})
	disableCommissionRuleTestMembership(t, f, f.employees[0])
	in := commissionRuleUpdateInput(plan, f.today)
	in.Name = "草稿备注维护"
	updated, err := f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in)
	if err != nil {
		t.Fatalf("停用成员不阻断草稿纯元数据维护: %v", err)
	}
	bad := commissionRuleUpdateInput(updated, f.today)
	bad.Enabled = true
	if _, err := f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, bad); !errors.Is(err, biz.ErrCommissionRuleEmployeeInvalid) {
		t.Fatalf("启用方案仍必须校验员工有效性: %v", err)
	}
	if got := f.getRule(plan.ID); got.Enabled || got.Version != updated.Version {
		t.Fatalf("失败启用不得改变状态或版本: %+v", got)
	}
}
