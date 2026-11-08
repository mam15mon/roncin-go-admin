package data

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	assignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionruleassignment"
	"github.com/shopspring/decimal"
)

func commissionRuleUpdateInput(plan *biz.FinanceCommissionRule, today string) biz.UpdateCommissionRuleInput {
	return biz.UpdateCommissionRuleInput{ID: plan.ID, ExpectedVersion: plan.Version, CreateCommissionRuleInput: biz.CreateCommissionRuleInput{Name: plan.Name, PersonnelRole: plan.PersonnelRole, CalculationBasis: plan.CalculationBasis, RatePercent: plan.RatePercent, EffectiveFrom: plan.EffectiveFrom, EffectiveTo: plan.EffectiveTo, Note: plan.Note, Enabled: plan.Enabled, Today: today}}
}

func TestCommissionRuleHistoricalMetadataUpdatePostgres(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "生效次日", true: "已正常到期"}[expired], func(t *testing.T) {
			f := newCommissionRuleAssignmentFixture(t)
			start := f.today
			var end *string
			if expired {
				end = stringPtr(f.dayOffset(1))
			}
			plan := f.createRule("历史方案", biz.CommissionRoleSales, &start, end, true, []uuid.UUID{f.employees[0]})
			in := commissionRuleUpdateInput(plan, f.dayOffset(2))
			in.Name = "历史方案新名称"
			in.Note = stringPtr("仅补充说明")
			updated, err := f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in)
			if err != nil {
				t.Fatalf("非核心修改应放行: %v", err)
			}
			if updated.Name != in.Name || updated.Note == nil || *updated.Note != *in.Note || updated.Version != plan.Version+1 {
				t.Fatalf("元数据修改错误: %+v", updated)
			}
			if _, err = f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in); !errors.Is(err, biz.ErrCommissionRuleConflict) {
				t.Fatalf("旧版本应拒绝: %v", err)
			}
			for _, field := range []string{"比例", "起始日", "身份", "口径", "停用", "追溯终止日"} {
				bad := commissionRuleUpdateInput(updated, f.dayOffset(2))
				switch field {
				case "比例":
					bad.RatePercent = decimal.NewFromInt(20)
				case "起始日":
					bad.EffectiveFrom = stringPtr(f.dayOffset(1))
				case "身份":
					bad.PersonnelRole = biz.CommissionRoleOperator
				case "口径":
					bad.CalculationBasis = biz.CommissionBasisRealizedRevenue
				case "停用":
					bad.Enabled = false
				case "追溯终止日":
					bad.EffectiveTo = &start
				}
				_, err = f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, bad)
				want := biz.ErrCommissionRuleLockedParams
				if field == "追溯终止日" {
					want = biz.ErrCommissionRuleRetroactive
				}
				if !errors.Is(err, want) {
					t.Fatalf("%s 应拒绝 %v，得到 %v", field, want, err)
				}
			}
			if got := f.getRule(plan.ID); got.Version != updated.Version || got.Name != updated.Name {
				t.Fatalf("失败更新不得写入: %+v", got)
			}
		})
	}
}

func TestCommissionRuleUpdateActualIntervalsPostgres(t *testing.T) {
	f := newCommissionRuleAssignmentFixture(t)
	from := f.dayOffset(1)
	source := f.createRule("待复制方案", biz.CommissionRoleSales, &from, nil, true, []uuid.UUID{f.employees[0]})
	nextFrom := f.dayOffset(10)
	_, err := f.usecase.CopyRule(f.ctx(), f.orgID, f.actorID, biz.CopyCommissionRuleInput{SourceRuleID: source.ID, Name: "后继方案", PersonnelRole: source.PersonnelRole, CalculationBasis: source.CalculationBasis, RatePercent: decimal.NewFromInt(6), EffectiveFrom: nextFrom, EmployeeIDs: []uuid.UUID{f.employees[0]}, Today: f.today})
	if err != nil {
		t.Fatal(err)
	}
	source = f.getRule(source.ID)
	in := commissionRuleUpdateInput(source, f.today)
	in.EffectiveTo = stringPtr(f.dayOffset(12))
	if _, err = f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in); !errors.Is(err, biz.ErrCommissionRuleIntervalOverlap) {
		t.Fatalf("延长源方案不得重叠: %v", err)
	}
	if got := f.getRule(source.ID); got.Version != source.Version || *got.EffectiveTo != f.dayOffset(9) {
		t.Fatalf("失败修改不得写入: %+v", got)
	}
	in.EffectiveTo = stringPtr(f.dayOffset(8))
	if _, err = f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, in); err != nil {
		t.Fatalf("相邻不重叠区间应允许: %v", err)
	}

	// 两个方案区间和分配区间分别相交，但实际资格交集互不相交。
	// A 方案[1,5]∩分配[1,10]=[1,5]；B 方案[4,10]∩分配[6,10]=[6,10]。
	aFrom, aTo := f.dayOffset(1), f.dayOffset(5)
	a := f.createRule("交集A", biz.CommissionRoleOperator, &aFrom, &aTo, true, []uuid.UUID{f.employees[1]})
	if _, err = f.data.db.FinanceCommissionRuleAssignment.Update().Where(assignmentent.RuleIDEQ(a.ID)).SetEffectiveTo(f.dayOffset(10)).Save(f.ctx()); err != nil {
		t.Fatal(err)
	}
	bFrom, bTo := f.dayOffset(4), f.dayOffset(10)
	b := f.createRule("交集B", biz.CommissionRoleOperator, &bFrom, &bTo, true, []uuid.UUID{f.employees[2]})
	if _, err = f.assignEmployees(b.ID, []uuid.UUID{f.employees[1]}, b.Version, f.dayOffset(6)); err != nil {
		t.Fatalf("实际资格不相交应允许: %v", err)
	}
	b = f.getRule(b.ID)
	update := commissionRuleUpdateInput(b, f.today)
	update.Name = "交集B元数据"
	if _, err = f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, update); err != nil {
		t.Fatalf("按每段实际区间复核，不得将员工错误拓宽到整个方案: %v", err)
	}
	// 未生效方案改身份同样必须按新身份复核，不能沿用旧身份查询。
	f.createRule("同员工销售资格", biz.CommissionRoleSales, &aFrom, &bTo, true, []uuid.UUID{f.employees[1]})
	roleUpdate := commissionRuleUpdateInput(a, f.today)
	roleUpdate.PersonnelRole = biz.CommissionRoleSales
	if _, err = f.usecase.UpdateRule(f.ctx(), f.orgID, f.actorID, roleUpdate); !errors.Is(err, biz.ErrCommissionRuleIntervalOverlap) {
		t.Fatalf("变更后的新身份发生重叠必须拒绝: %v", err)
	}
}

func TestCommissionRuleUpdateAndAssignConcurrentPostgres(t *testing.T) {
	f := newCommissionRuleAssignmentFixture(t)
	from, to := f.dayOffset(1), f.dayOffset(5)
	a := f.createRule("并发编辑A", biz.CommissionRoleSales, &from, &to, true, []uuid.UUID{f.employees[0]})
	bFrom, bTo := f.dayOffset(6), f.dayOffset(10)
	b := f.createRule("并发分配B", biz.CommissionRoleSales, &bFrom, &bTo, true, []uuid.UUID{f.employees[1]})
	// 分配段原本有限；先补齐开放分配夹具，扩大方案日期会真实扩大资格。
	if _, err := f.data.db.FinanceCommissionRuleAssignment.Update().Where(assignmentent.RuleIDEQ(a.ID)).ClearEffectiveTo().Save(f.ctx()); err != nil {
		t.Fatal(err)
	}
	in := commissionRuleUpdateInput(a, f.today)
	in.EffectiveTo = &bTo
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := f.usecase.UpdateRule(context.Background(), f.orgID, f.actorID, in)
		results <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := f.assignEmployees(b.ID, []uuid.UUID{f.employees[0]}, b.Version, bFrom)
		results <- err
	}()
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, biz.ErrCommissionRuleIntervalOverlap) {
			t.Fatalf("并发应成功或重叠拒绝: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("两个竞争写入应恰好一个成功，实际%d", success)
	}
	a = f.getRule(a.ID)
	b = f.getRule(b.ID)
	aWindow, _ := biz.WorkbenchActualAssignmentWindow(a.Enabled, *a.EffectiveFrom, optionalStringValue(a.EffectiveTo), assignmentSegment(a, f.employees[0]).EffectiveFrom, optionalStringValue(assignmentSegment(a, f.employees[0]).EffectiveTo))
	if segment := assignmentSegment(b, f.employees[0]); segment != nil {
		bWindow, _ := biz.WorkbenchActualAssignmentWindow(b.Enabled, *b.EffectiveFrom, optionalStringValue(b.EffectiveTo), segment.EffectiveFrom, optionalStringValue(segment.EffectiveTo))
		if biz.CommissionClosedIntervalsOverlap(aWindow.EffectiveFrom, aWindow.EffectiveTo, bWindow.EffectiveFrom, bWindow.EffectiveTo) {
			t.Fatal("并发完成后不得留下重叠资格")
		}
	}
}
