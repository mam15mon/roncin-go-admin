package data

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	assignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
)

// 提成方案员工分配集成测试：真实 PostgreSQL 上验证固定锁序（Membership → Rule）、
// 跨方案实际区间唯一、今天/未来名单变更、已生效方案参数锁定、【复制为新方案】
// 衔接、旧规则只读与成员停用阻断，以及并发把同一员工身份加入重叠方案时仅一个
// 成功。方案或名单写入口的锁互斥行为无法用 sqlmock 覆盖，必须以集成环境验证。

type commissionRuleAssignmentFixture struct {
	t       *testing.T
	data    *Data
	usecase *biz.CommissionUsecase
	admin   biz.AdminRepo
	orgID   uuid.UUID
	actorID uuid.UUID
	today   string
	// employees[0..2] 为当前组织有效成员；inactive 为成员关系停用的用户；
	// outsider 为没有成员关系的用户。
	employees []uuid.UUID
	inactive  uuid.UUID
	outsider  uuid.UUID
}

func newCommissionRuleAssignmentFixture(t *testing.T) *commissionRuleAssignmentFixture {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	org, err := data.db.Organization.Create().
		SetCode("CRA-" + suffix).
		SetName("方案分配测试组织-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	fixture := &commissionRuleAssignmentFixture{
		t: t, data: data, today: biz.FinanceBusinessDate(time.Now()),
		orgID: org.ID, employees: make([]uuid.UUID, 0, 3),
		admin: NewAdminRepo(data),
	}
	fixture.usecase = biz.NewCommissionUsecase(NewCommissionRepo(data), nil, nil)

	actor, err := data.db.User.Create().
		SetUsername("cra_actor_" + suffix).
		SetDisplayName("方案分配操作员-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建操作员: %v", err)
	}
	fixture.actorID = actor.ID
	fixture.mustCreateMembership(actor.ID, org.ID)

	for i := 0; i < 3; i++ {
		employee, err := data.db.User.Create().
			SetUsername("cra_emp" + string(rune('0'+i)) + "_" + suffix).
			SetDisplayName("方案分配员工-" + string(rune('0'+i)) + suffix).
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建员工 %d: %v", i, err)
		}
		fixture.employees = append(fixture.employees, employee.ID)
		fixture.mustCreateMembership(employee.ID, org.ID)
	}

	inactive, err := data.db.User.Create().
		SetUsername("cra_inactive_" + suffix).
		SetDisplayName("停用成员-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建停用成员用户: %v", err)
	}
	fixture.inactive = inactive.ID
	fixture.mustCreateMembership(inactive.ID, org.ID)
	if _, err = data.db.Membership.Update().Where(membershipent.UserIDEQ(inactive.ID)).SetEnabled(false).Save(ctx); err != nil {
		t.Fatalf("停用成员关系: %v", err)
	}

	// 员工在第二个组织保留成员关系：成员停用测试只针对本组织成员关系，
	// 不会先触发「在职用户必须保留至少一个有效组织」的既有门禁。
	secondOrg, err := data.db.Organization.Create().
		SetCode("CRA2-" + suffix).
		SetName("方案分配备援组织-" + suffix).
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建备援组织: %v", err)
	}
	for _, employeeID := range fixture.employees {
		fixture.mustCreateMembership(employeeID, secondOrg.ID)
	}

	outsider, err := data.db.User.Create().
		SetUsername("cra_outsider_" + suffix).
		SetDisplayName("组织外用户-" + suffix).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组织外用户: %v", err)
	}
	fixture.outsider = outsider.ID
	return fixture
}

func (f *commissionRuleAssignmentFixture) mustCreateMembership(userID, orgID uuid.UUID) {
	t := f.t
	t.Helper()
	if _, err := f.data.db.Membership.Create().SetUserID(userID).SetOrganizationID(orgID).SetEnabled(true).Save(context.Background()); err != nil {
		t.Fatalf("创建成员关系: %v", err)
	}
}

func (f *commissionRuleAssignmentFixture) ctx() context.Context {
	return context.Background()
}

// createRule 经用例创建方案的便捷封装，断言成功并返回刷新后的方案。
func (f *commissionRuleAssignmentFixture) createRule(name string, role biz.CommissionPersonnelRole, from, to *string, enabled bool, employees []uuid.UUID) *biz.FinanceCommissionRule {
	t := f.t
	t.Helper()
	created, err := f.usecase.CreateRule(f.ctx(), f.orgID, f.actorID, biz.CreateCommissionRuleInput{
		Name: name, PersonnelRole: role, CalculationBasis: biz.CommissionBasisRealizedProfit,
		RatePercent: decimal.RequireFromString("5"), EffectiveFrom: from, EffectiveTo: to,
		Enabled: enabled, EmployeeIDs: employees, Today: f.today,
	})
	if err != nil {
		t.Fatalf("创建方案 %s 失败: %v", name, err)
	}
	return created
}

func (f *commissionRuleAssignmentFixture) dayOffset(days int) string {
	t := f.t
	t.Helper()
	parsed, err := time.Parse("2006-01-02", f.today)
	if err != nil {
		t.Fatalf("解析业务日期: %v", err)
	}
	return parsed.AddDate(0, 0, days).Format("2006-01-02")
}

func (f *commissionRuleAssignmentFixture) assignEmployees(ruleID uuid.UUID, employees []uuid.UUID, version uint64, changeDate string) (*biz.FinanceCommissionRule, error) {
	return f.usecase.AssignRuleEmployees(f.ctx(), f.orgID, f.actorID, biz.CommissionRuleEmployeeChange{
		RuleID: ruleID, EmployeeIDs: employees, ExpectedVersion: version,
		ChangeEffectiveDate: changeDate, Today: f.today,
	})
}

func (f *commissionRuleAssignmentFixture) removeEmployees(ruleID uuid.UUID, employees []uuid.UUID, version uint64, changeDate string) (*biz.FinanceCommissionRule, error) {
	return f.usecase.RemoveRuleEmployees(f.ctx(), f.orgID, f.actorID, biz.CommissionRuleEmployeeChange{
		RuleID: ruleID, EmployeeIDs: employees, ExpectedVersion: version,
		ChangeEffectiveDate: changeDate, Today: f.today,
	})
}

func (f *commissionRuleAssignmentFixture) getRule(ruleID uuid.UUID) *biz.FinanceCommissionRule {
	t := f.t
	t.Helper()
	item, err := f.usecase.GetRuleScoped(f.ctx(), []uuid.UUID{f.orgID}, ruleID)
	if err != nil {
		t.Fatalf("读取方案: %v", err)
	}
	return item
}

// assignmentSegment 在方案中查找指定员工的未取消分配段。
func assignmentSegment(rule *biz.FinanceCommissionRule, employeeID uuid.UUID) *biz.FinanceCommissionRuleAssignment {
	for _, item := range rule.Assignments {
		if item.EmployeeID == employeeID {
			return item
		}
	}
	return nil
}

func assertSameError(t *testing.T, err error, target error, scenario string) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("%s：期望错误 %v，实际 %v", scenario, target, err)
	}
}

func TestCommissionRuleCreateAndAssignPostgres(t *testing.T) {
	fixture := newCommissionRuleAssignmentFixture(t)
	e1, e2 := fixture.employees[0], fixture.employees[1]

	t.Run("创建启用方案写入初始分配", func(t *testing.T) {
		from := fixture.dayOffset(1)
		to := fixture.dayOffset(365)
		rule := fixture.createRule("销售方案A", biz.CommissionRoleSales, &from, &to, true, []uuid.UUID{e1, e2})
		if !rule.Enabled || rule.LegacyReadOnly {
			t.Fatalf("新方案应为启用且非历史只读: %#v", rule)
		}
		if len(rule.Assignments) != 2 {
			t.Fatalf("应写入 2 条初始分配，实际 %d", len(rule.Assignments))
		}
		for _, item := range rule.Assignments {
			if item.EffectiveFrom != from || item.EffectiveTo == nil || *item.EffectiveTo != to {
				t.Fatalf("初始分配区间应跟随方案区间: %#v", item)
			}
			if item.CreatedBy != fixture.actorID {
				t.Fatalf("初始分配创建人应为操作者: %#v", item)
			}
		}
	})

	t.Run("启用方案缺员工与非法成员被拒绝", func(t *testing.T) {
		from := fixture.dayOffset(1)
		_, err := fixture.usecase.CreateRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.CreateCommissionRuleInput{
			Name: "缺员工方案", PersonnelRole: biz.CommissionRoleSales, CalculationBasis: biz.CommissionBasisRealizedProfit,
			RatePercent: decimal.RequireFromString("5"), EffectiveFrom: &from, Enabled: true, Today: fixture.today,
		})
		assertSameError(t, err, biz.ErrCommissionRuleEnableNoEmployee, "启用方案缺员工")

		_, err = fixture.usecase.CreateRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.CreateCommissionRuleInput{
			Name: "非法成员方案", PersonnelRole: biz.CommissionRoleSales, CalculationBasis: biz.CommissionBasisRealizedProfit,
			RatePercent: decimal.RequireFromString("5"), EffectiveFrom: &from, Enabled: true,
			EmployeeIDs: []uuid.UUID{fixture.outsider}, Today: fixture.today,
		})
		assertSameError(t, err, biz.ErrCommissionRuleEmployeeInvalid, "组织外用户不能加入方案")

		_, err = fixture.usecase.CreateRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.CreateCommissionRuleInput{
			Name: "停用成员方案", PersonnelRole: biz.CommissionRoleSales, CalculationBasis: biz.CommissionBasisRealizedProfit,
			RatePercent: decimal.RequireFromString("5"), EffectiveFrom: &from, Enabled: true,
			EmployeeIDs: []uuid.UUID{fixture.inactive}, Today: fixture.today,
		})
		assertSameError(t, err, biz.ErrCommissionRuleEmployeeInvalid, "停用成员不能加入方案")
	})

	t.Run("未生效方案直接调整名单缺省从方案起始日", func(t *testing.T) {
		from := fixture.dayOffset(1)
		rule := fixture.createRule("客服方案B", biz.CommissionRoleCustomerService, &from, nil, true, []uuid.UUID{e2})
		updated, err := fixture.assignEmployees(rule.ID, []uuid.UUID{fixture.employees[2]}, rule.Version, "")
		if err != nil {
			t.Fatalf("未生效方案直接加人应放行: %v", err)
		}
		segment := assignmentSegment(updated, fixture.employees[2])
		if segment == nil || segment.EffectiveFrom != from {
			t.Fatalf("新增分配应从方案起始日开始: %#v", segment)
		}
	})
}

func TestCommissionRuleIntervalUniquenessPostgres(t *testing.T) {
	fixture := newCommissionRuleAssignmentFixture(t)
	e1 := fixture.employees[0]

	t.Run("跨方案重叠与同起点重复被拒绝", func(t *testing.T) {
		from := fixture.dayOffset(1)
		to := fixture.dayOffset(10)
		planA := fixture.createRule("销售唯一A", biz.CommissionRoleSales, &from, &to, true, []uuid.UUID{e1})

		overlapFrom := fixture.dayOffset(5)
		overlapTo := fixture.dayOffset(20)
		_, err := fixture.usecase.CreateRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.CreateCommissionRuleInput{
			Name: "销售唯一B", PersonnelRole: biz.CommissionRoleSales, CalculationBasis: biz.CommissionBasisRealizedProfit,
			RatePercent: decimal.RequireFromString("5"), EffectiveFrom: &overlapFrom, EffectiveTo: &overlapTo,
			Enabled: true, EmployeeIDs: []uuid.UUID{e1}, Today: fixture.today,
		})
		assertSameError(t, err, biz.ErrCommissionRuleIntervalOverlap, "区间重叠的第二方案")

		// 交界日相邻视为连续：第二天从 A 终止日的次日开始。
		adjacentFrom := fixture.dayOffset(11)
		adjacentTo := fixture.dayOffset(20)
		fixture.createRule("销售唯一C", biz.CommissionRoleSales, &adjacentFrom, &adjacentTo, true, []uuid.UUID{e1})

		// 同方案同员工重复登记同一起点：锁内区间校验先命中重叠。
		refreshed := fixture.getRule(planA.ID)
		_, err = fixture.assignEmployees(planA.ID, []uuid.UUID{e1}, refreshed.Version, "")
		assertSameError(t, err, biz.ErrCommissionRuleIntervalOverlap, "同方案同起点重复登记")
	})

	t.Run("不同身份方案互不冲突", func(t *testing.T) {
		from := fixture.dayOffset(1)
		fixture.createRule("操作方案D", biz.CommissionRoleOperator, &from, nil, true, []uuid.UUID{e1})
		fixture.createRule("客服方案E", biz.CommissionRoleCustomerService, &from, nil, true, []uuid.UUID{e1})
	})
}

func TestCommissionRuleEffectiveChangePostgres(t *testing.T) {
	fixture := newCommissionRuleAssignmentFixture(t)
	e1, e2 := fixture.employees[0], fixture.employees[1]

	// 已到达生效日的启用方案：起始日为当天。
	today := fixture.today
	plan := fixture.createRule("销售生效方案", biz.CommissionRoleSales, &today, nil, true, []uuid.UUID{e1})

	t.Run("已生效方案新增成员必须当天或未来", func(t *testing.T) {
		_, err := fixture.assignEmployees(plan.ID, []uuid.UUID{e2}, plan.Version, "")
		assertSameError(t, err, biz.ErrCommissionRuleRetroactive, "缺省变更日")

		_, err = fixture.assignEmployees(plan.ID, []uuid.UUID{e2}, plan.Version, fixture.dayOffset(-1))
		assertSameError(t, err, biz.ErrCommissionRuleRetroactive, "过去变更日")

		updated, err := fixture.assignEmployees(plan.ID, []uuid.UUID{e2}, plan.Version, fixture.dayOffset(2))
		if err != nil {
			t.Fatalf("未来变更日应放行: %v", err)
		}
		if updated.Version != plan.Version+1 {
			t.Fatalf("名单变更应递增版本: %d -> %d", plan.Version, updated.Version)
		}
		segment := assignmentSegment(updated, e2)
		if segment == nil || segment.EffectiveFrom != fixture.dayOffset(2) {
			t.Fatalf("新增分配应从变更日开始: %#v", segment)
		}
	})

	t.Run("移除分配以变更日前一日结束并可重新加入", func(t *testing.T) {
		leaveDate := fixture.dayOffset(6)
		updated, err := fixture.removeEmployees(plan.ID, []uuid.UUID{e2}, plan.Version+1, leaveDate)
		if err != nil {
			t.Fatalf("移除分配失败: %v", err)
		}
		segment := assignmentSegment(updated, e2)
		if segment == nil || segment.EffectiveTo == nil || *segment.EffectiveTo != fixture.dayOffset(5) {
			t.Fatalf("移除分配应以变更日前一日结束: %#v", segment)
		}
		if segment.TerminatedAt == nil || segment.TerminatedBy == nil || segment.CancelledAt != nil {
			t.Fatalf("终止分配应记录终止审计而非取消: %#v", segment)
		}

		rejoined, err := fixture.assignEmployees(plan.ID, []uuid.UUID{e2}, plan.Version+2, fixture.dayOffset(6))
		if err != nil {
			t.Fatalf("终止后重新加入应放行: %v", err)
		}
		segments := 0
		for _, item := range rejoined.Assignments {
			if item.EmployeeID == e2 {
				segments++
			}
		}
		if segments != 2 {
			t.Fatalf("重新加入应新增一段分配，现有 %d 段", segments)
		}
	})

	t.Run("变更日不晚于起始日的分配标记取消", func(t *testing.T) {
		cancelDate := fixture.dayOffset(1)
		if _, err := fixture.removeEmployees(plan.ID, []uuid.UUID{e2}, plan.Version+3, cancelDate); err != nil {
			t.Fatalf("取消未生效分配失败: %v", err)
		}
		// 取消段保留审计但不再进入名单投影，直接查询表断言。
		cancelled, err := fixture.data.db.FinanceCommissionRuleAssignment.Query().
			Where(
				assignmentent.RuleIDEQ(plan.ID),
				assignmentent.EmployeeIDEQ(e2),
				assignmentent.CancelledAtNotNil(),
			).All(fixture.ctx())
		if err != nil {
			t.Fatalf("查询取消分配: %v", err)
		}
		if len(cancelled) == 0 {
			t.Fatalf("应存在被取消的未生效分配段")
		}
		for _, item := range cancelled {
			if item.CancelledBy == nil || item.EffectiveTo != nil {
				t.Fatalf("未生效分配应以取消标记退出且不携带终止日: %#v", item)
			}
			if item.EffectiveFrom < cancelDate {
				t.Fatalf("取消段起始日应不晚于变更日: %#v", item)
			}
		}
	})

	t.Run("已生效方案锁定计算参数", func(t *testing.T) {
		locked := fixture.getRule(plan.ID)
		input := biz.UpdateCommissionRuleInput{
			ID: plan.ID, ExpectedVersion: locked.Version,
			CreateCommissionRuleInput: biz.CreateCommissionRuleInput{
				Name: locked.Name, PersonnelRole: locked.PersonnelRole, CalculationBasis: locked.CalculationBasis,
				RatePercent: decimal.RequireFromString("8"), EffectiveFrom: locked.EffectiveFrom,
				Enabled: true, Today: fixture.today,
			},
		}
		_, err := fixture.usecase.UpdateRule(fixture.ctx(), fixture.orgID, fixture.actorID, input)
		assertSameError(t, err, biz.ErrCommissionRuleLockedParams, "改比例")

		input.CreateCommissionRuleInput.RatePercent = locked.RatePercent
		input.CreateCommissionRuleInput.Enabled = false
		_, err = fixture.usecase.UpdateRule(fixture.ctx(), fixture.orgID, fixture.actorID, input)
		assertSameError(t, err, biz.ErrCommissionRuleLockedParams, "停用已生效方案")

		input.CreateCommissionRuleInput.Enabled = true
		input.CreateCommissionRuleInput.Name = "销售生效方案-改名"
		updated, err := fixture.usecase.UpdateRule(fixture.ctx(), fixture.orgID, fixture.actorID, input)
		if err != nil {
			t.Fatalf("改名应放行: %v", err)
		}
		if updated.Name != "销售生效方案-改名" || updated.Version != locked.Version+1 {
			t.Fatalf("名称更新应生效并递增版本: %#v", updated)
		}

		input.CreateCommissionRuleInput.Name = updated.Name
		input.ExpectedVersion = updated.Version
		yesterday := fixture.dayOffset(-1)
		input.CreateCommissionRuleInput.EffectiveTo = &yesterday
		// 起止倒置的区间先在输入归一化被拒（400），不必等到锁内回溯校验。
		_, err = fixture.usecase.UpdateRule(fixture.ctx(), fixture.orgID, fixture.actorID, input)
		assertSameError(t, err, biz.ErrCommissionRuleInvalid, "终止日早于起始日")

		futureEnd := fixture.dayOffset(30)
		input.CreateCommissionRuleInput.EffectiveTo = &futureEnd
		if _, err = fixture.usecase.UpdateRule(fixture.ctx(), fixture.orgID, fixture.actorID, input); err != nil {
			t.Fatalf("不早于当天的终止日应放行: %v", err)
		}
	})
}

func TestCommissionRuleLegacyAndCopyPostgres(t *testing.T) {
	fixture := newCommissionRuleAssignmentFixture(t)
	e1, e2 := fixture.employees[0], fixture.employees[1]

	t.Run("迁移旧规则只读且仅可复制", func(t *testing.T) {
		suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
		legacy, err := fixture.data.db.FinanceCommissionRule.Create().
			SetOrganizationID(fixture.orgID).SetName("迁移旧规则-" + suffix).
			SetPersonnelRole("SALES").SetCalculationBasis("REALIZED_PROFIT").SetRatePercent("5").
			SetEnabled(false).SetLegacyReadonly(true).Save(fixture.ctx())
		if err != nil {
			t.Fatalf("创建迁移旧规则: %v", err)
		}
		_, err = fixture.usecase.UpdateRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.UpdateCommissionRuleInput{
			ID: legacy.ID, ExpectedVersion: 1,
			CreateCommissionRuleInput: biz.CreateCommissionRuleInput{
				Name: "旧规则改名", PersonnelRole: biz.CommissionRoleSales, CalculationBasis: biz.CommissionBasisRealizedProfit,
				RatePercent: decimal.RequireFromString("5"), Enabled: true,
				EffectiveFrom: stringPtr(fixture.dayOffset(1)), Today: fixture.today,
			},
		})
		assertSameError(t, err, biz.ErrCommissionRuleLegacyReadOnly, "更新旧规则")

		_, err = fixture.assignEmployees(legacy.ID, []uuid.UUID{e1}, 1, fixture.dayOffset(1))
		assertSameError(t, err, biz.ErrCommissionRuleLegacyReadOnly, "旧规则补挂员工")

		copied, err := fixture.usecase.CopyRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.CopyCommissionRuleInput{
			SourceRuleID: legacy.ID, Name: "旧规则复制新方案", PersonnelRole: biz.CommissionRoleSales,
			CalculationBasis: biz.CommissionBasisRealizedProfit, RatePercent: decimal.RequireFromString("6"),
			EffectiveFrom: fixture.dayOffset(1), EmployeeIDs: []uuid.UUID{e1, e2}, Today: fixture.today,
		})
		if err != nil {
			t.Fatalf("复制旧规则应放行: %v", err)
		}
		if !copied.Enabled || copied.LegacyReadOnly || len(copied.Assignments) != 2 {
			t.Fatalf("复制出的新方案应有真实员工分配: %#v", copied)
		}
	})

	t.Run("复制已生效方案并衔接旧方案终止日", func(t *testing.T) {
		// 前一子任务的复制方案已占用 e1 的销售区间，这里改用未分配员工。
		e3 := fixture.employees[2]
		today := fixture.today
		source := fixture.createRule("销售衔接源方案", biz.CommissionRoleSales, &today, nil, true, []uuid.UUID{e3})
		newFrom := fixture.dayOffset(3)
		copied, err := fixture.usecase.CopyRule(fixture.ctx(), fixture.orgID, fixture.actorID, biz.CopyCommissionRuleInput{
			SourceRuleID: source.ID, Name: "销售衔接新方案", PersonnelRole: biz.CommissionRoleSales,
			CalculationBasis: biz.CommissionBasisRealizedProfit, RatePercent: decimal.RequireFromString("7"),
			EffectiveFrom: newFrom, EmployeeIDs: []uuid.UUID{e3}, Today: fixture.today,
		})
		if err != nil {
			t.Fatalf("复制已生效方案失败: %v", err)
		}
		refreshed := fixture.getRule(source.ID)
		if refreshed.EffectiveTo == nil || *refreshed.EffectiveTo != fixture.dayOffset(2) {
			t.Fatalf("旧方案终止日应衔接为新方案生效日前一日: %#v", refreshed)
		}
		if refreshed.Version != source.Version+1 {
			t.Fatalf("旧方案衔接应递增版本: %#v", refreshed)
		}
		segment := assignmentSegment(copied, e3)
		if segment == nil || segment.EffectiveFrom != newFrom {
			t.Fatalf("新方案初始分配应从新起始日开始: %#v", segment)
		}
		// 衔接后同员工同身份区间唯一：对重叠区间再加人会命中锁内重叠校验。
		_, err = fixture.assignEmployees(copied.ID, []uuid.UUID{e3}, copied.Version, newFrom)
		assertSameError(t, err, biz.ErrCommissionRuleIntervalOverlap, "新方案重叠区间重复登记")
	})
}

func TestCommissionRuleMemberDisableGuardPostgres(t *testing.T) {
	fixture := newCommissionRuleAssignmentFixture(t)
	e1, e2 := fixture.employees[0], fixture.employees[1]

	today := fixture.today
	plan := fixture.createRule("停用阻断方案", biz.CommissionRoleSales, &today, nil, true, []uuid.UUID{e1, e2})

	disableMembership := func(t *testing.T, userID uuid.UUID) error {
		t.Helper()
		row, err := fixture.data.db.Membership.Query().
			Where(membershipent.UserIDEQ(userID), membershipent.OrganizationIDEQ(fixture.orgID)).
			Only(fixture.ctx())
		if err != nil {
			t.Fatalf("查询成员关系: %v", err)
		}
		_, err = fixture.admin.UpdateUserMembership(fixture.ctx(), &biz.AdminUserMembership{
			ID: row.ID, UserID: userID, Enabled: false,
		}, nil, &biz.AuditEvent{Action: "test.membership.disable", Result: "success"})
		return err
	}

	t.Run("仍有当前分配时停用成员被阻断", func(t *testing.T) {
		err := disableMembership(t, e1)
		if err == nil {
			t.Fatalf("仍有当前分配时停用成员应被拒绝")
		}
		if !strings.Contains(err.Error(), "停用阻断方案") {
			t.Fatalf("阻断错误应列出需先处理的方案名，实际: %v", err)
		}
		// 分配未被自动删除或截断。
		refreshed := fixture.getRule(plan.ID)
		if assignmentSegment(refreshed, e1) == nil {
			t.Fatalf("停用被拒后历史分配必须保留")
		}
	})

	t.Run("以未来离开日期终止分配后可停用", func(t *testing.T) {
		leaveDate := fixture.dayOffset(7)
		updated, err := fixture.removeEmployees(plan.ID, []uuid.UUID{e1}, plan.Version, leaveDate)
		if err != nil {
			t.Fatalf("以离开日期终止分配失败: %v", err)
		}
		segment := assignmentSegment(updated, e1)
		if segment == nil || segment.EffectiveTo == nil || *segment.EffectiveTo != fixture.dayOffset(6) {
			t.Fatalf("离开日期应以变更日前一日结束分配: %#v", segment)
		}
		if err := disableMembership(t, e1); err != nil {
			t.Fatalf("终止分配后停用成员应放行: %v", err)
		}
		// 停用成员后历史分配仍在，方案名单不再包含其未来资格。
		refreshed := fixture.getRule(plan.ID)
		if assignmentSegment(refreshed, e1) == nil {
			t.Fatalf("停用成员不得删除历史分配")
		}
	})

	t.Run("未来分配同样阻断停用", func(t *testing.T) {
		// 先以未来日期终止 e2 的当前开放段，再新增一段更晚开始的未来分配。
		refreshed := fixture.getRule(plan.ID)
		terminated, err := fixture.removeEmployees(plan.ID, []uuid.UUID{e2}, refreshed.Version, fixture.dayOffset(3))
		if err != nil {
			t.Fatalf("终止当前分配失败: %v", err)
		}
		if segment := assignmentSegment(terminated, e2); segment == nil || segment.TerminatedAt == nil {
			t.Fatalf("当前分配应被终止: %#v", segment)
		}
		afterTerminate := fixture.getRule(plan.ID)
		updated, err := fixture.assignEmployees(plan.ID, []uuid.UUID{e2}, afterTerminate.Version, fixture.dayOffset(5))
		if err != nil {
			t.Fatalf("新增未来分配失败: %v", err)
		}
		if err := disableMembership(t, e2); err == nil {
			t.Fatalf("存在未来分配时停用成员应被拒绝")
		}
		if assignmentSegment(updated, e2) == nil {
			t.Fatalf("分配必须保留")
		}
	})
}

// TestCommissionRuleConcurrentOverlapPostgres 验证固定锁序并发一致性：两个并发
// 事务把同一员工同一身份加入两个重叠方案时，成员行锁串行化使后到事务在锁内
// 读到先到事务已提交的分配，仅一个成功、另一个稳定拒绝。
func TestCommissionRuleConcurrentOverlapPostgres(t *testing.T) {
	fixture := newCommissionRuleAssignmentFixture(t)
	e1, e2, e3 := fixture.employees[0], fixture.employees[1], fixture.employees[2]

	from := fixture.dayOffset(1)
	// 两个重叠方案分别以其他员工创建，避免创建期先触发区间冲突。
	planA := fixture.createRule("并发方案A", biz.CommissionRoleSales, &from, nil, true, []uuid.UUID{e2})
	planB := fixture.createRule("并发方案B", biz.CommissionRoleSales, &from, nil, true, []uuid.UUID{e3})

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	for i, target := range []*biz.FinanceCommissionRule{planA, planB} {
		go func(idx int, plan *biz.FinanceCommissionRule) {
			defer wg.Done()
			_, err := fixture.usecase.AssignRuleEmployees(context.Background(), fixture.orgID, fixture.actorID, biz.CommissionRuleEmployeeChange{
				RuleID: plan.ID, EmployeeIDs: []uuid.UUID{e1}, ExpectedVersion: plan.Version,
				ChangeEffectiveDate: from, Today: fixture.today,
			})
			errs[idx] = err
		}(i, target)
	}
	wg.Wait()

	successes := 0
	for _, err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, biz.ErrCommissionRuleIntervalOverlap):
		default:
			t.Fatalf("并发结果出现意外错误: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("并发加入重叠方案应恰好一个成功，实际 %d 个（errs=%v）", successes, errs)
	}

	// 最终状态：同一员工同身份只能在一个方案中命中该区间。
	ruleA, ruleB := fixture.getRule(planA.ID), fixture.getRule(planB.ID)
	count := 0
	for _, rule := range []*biz.FinanceCommissionRule{ruleA, ruleB} {
		if segment := assignmentSegment(rule, e1); segment != nil {
			count++
			if segment.EffectiveFrom != from {
				t.Fatalf("成功分配应从变更日开始: %#v", segment)
			}
		}
	}
	if count != 1 {
		t.Fatalf("员工应只属于一个重叠方案，实际命中 %d 个", count)
	}
}
