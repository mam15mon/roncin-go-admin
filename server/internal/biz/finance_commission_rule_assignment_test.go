package biz

import (
	"errors"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TestCommissionClosedIntervalsOverlap 覆盖方案员工分配实际有效区间的闭区间
// 判定：相邻、相交、包含、开放终点与分离区间。
func TestCommissionClosedIntervalsOverlap(t *testing.T) {
	cases := []struct {
		name                   string
		fromA, toA, fromB, toB string
		expect                 bool
	}{
		{name: "相邻闭区间在共用端点相交", fromA: "2026-01-01", toA: "2026-01-10", fromB: "2026-01-10", toB: "2026-01-20", expect: true},
		{name: "部分重叠", fromA: "2026-01-01", toA: "2026-01-15", fromB: "2026-01-10", toB: "2026-01-20", expect: true},
		{name: "包含", fromA: "2026-01-01", toA: "2026-12-31", fromB: "2026-03-01", toB: "2026-03-31", expect: true},
		{name: "分离", fromA: "2026-01-01", toA: "2026-01-10", fromB: "2026-01-11", toB: "2026-01-20", expect: false},
		{name: "A 开放终点与 B 相交", fromA: "2026-01-01", toA: "", fromB: "2026-05-01", toB: "2026-05-31", expect: true},
		{name: "B 开放终点与 A 分离", fromA: "2026-01-01", toA: "2026-01-10", fromB: "2026-02-01", toB: "", expect: false},
		{name: "双开放终点相交", fromA: "2026-01-01", toA: "", fromB: "2025-01-01", toB: "", expect: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CommissionClosedIntervalsOverlap(tc.fromA, tc.toA, tc.fromB, tc.toB); got != tc.expect {
				t.Fatalf("CommissionClosedIntervalsOverlap(%q,%q,%q,%q) = %v，期望 %v", tc.fromA, tc.toA, tc.fromB, tc.toB, got, tc.expect)
			}
		})
	}
}

func TestFinanceDateBefore(t *testing.T) {
	cases := []struct {
		input, expect string
	}{
		{input: "2026-09-18", expect: "2026-09-17"},
		{input: "2026-09-01", expect: "2026-08-31"},
		{input: "2026-01-01", expect: "2025-12-31"},
		{input: "2028-03-01", expect: "2028-02-29"},
	}
	for _, tc := range cases {
		if got := FinanceDateBefore(tc.input); got != tc.expect {
			t.Fatalf("FinanceDateBefore(%q) = %q，期望 %q", tc.input, got, tc.expect)
		}
	}
}

func TestFinanceBusinessDateUsesShanghaiOffset(t *testing.T) {
	// UTC 2026-09-18 17:00 已是上海时间 2026-09-19 凌晨 1 点。
	utcTime := time.Date(2026, 9, 18, 17, 0, 0, 0, time.UTC)
	if got := FinanceBusinessDate(utcTime); got != "2026-09-19" {
		t.Fatalf("FinanceBusinessDate = %q，期望上海时区日期 2026-09-19", got)
	}
	// UTC 2026-09-18 08:00 对应上海时间 16 点，仍是同一天。
	if got := FinanceBusinessDate(time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)); got != "2026-09-18" {
		t.Fatalf("FinanceBusinessDate = %q，期望 2026-09-18", got)
	}
}

func TestValidCommissionRuleEmployeeChange(t *testing.T) {
	ruleID := uuid.New()
	employeeID := uuid.New()

	valid, err := validCommissionRuleEmployeeChange(CommissionRuleEmployeeChange{
		RuleID: ruleID, EmployeeIDs: []uuid.UUID{employeeID, employeeID},
		ExpectedVersion: 3, Today: "2026-09-18", ChangeEffectiveDate: "2026-09-20",
	})
	if err != nil {
		t.Fatalf("合法名单变更输入被拒绝: %v", err)
	}
	if len(valid.EmployeeIDs) != 1 || valid.EmployeeIDs[0] != employeeID {
		t.Fatalf("员工集合应去重排序: %v", valid.EmployeeIDs)
	}

	cases := []struct {
		name   string
		mutate func(*CommissionRuleEmployeeChange)
	}{
		{name: "缺少方案", mutate: func(in *CommissionRuleEmployeeChange) { in.RuleID = uuid.Nil }},
		{name: "缺少版本", mutate: func(in *CommissionRuleEmployeeChange) { in.ExpectedVersion = 0 }},
		{name: "缺少员工", mutate: func(in *CommissionRuleEmployeeChange) { in.EmployeeIDs = nil }},
		{name: "零值员工", mutate: func(in *CommissionRuleEmployeeChange) { in.EmployeeIDs = []uuid.UUID{uuid.Nil} }},
		{name: "员工超上限", mutate: func(in *CommissionRuleEmployeeChange) {
			in.EmployeeIDs = make([]uuid.UUID, 201)
			for i := range in.EmployeeIDs {
				in.EmployeeIDs[i] = uuid.New()
			}
		}},
		{name: "变更日期非法", mutate: func(in *CommissionRuleEmployeeChange) { in.ChangeEffectiveDate = "2026-02-30" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := CommissionRuleEmployeeChange{RuleID: ruleID, EmployeeIDs: []uuid.UUID{employeeID}, ExpectedVersion: 1, Today: "2026-09-18"}
			tc.mutate(&in)
			if _, err := validCommissionRuleEmployeeChange(in); err == nil {
				t.Fatalf("非法名单变更输入应被拒绝")
			}
		})
	}
}

func TestValidCommissionRuleEmployeeChangeDefaultsToday(t *testing.T) {
	valid, err := validCommissionRuleEmployeeChange(CommissionRuleEmployeeChange{
		RuleID: uuid.New(), EmployeeIDs: []uuid.UUID{uuid.New()}, ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("缺省业务日期应自动补齐: %v", err)
	}
	if _, parseErr := time.Parse("2006-01-02", valid.Today); parseErr != nil {
		t.Fatalf("补齐的业务日期非法: %v", parseErr)
	}
}

func TestNormalizeCopyCommissionRuleInput(t *testing.T) {
	sourceID := uuid.New()
	employeeID := uuid.New()
	base := CopyCommissionRuleInput{
		SourceRuleID: sourceID, Name: "销售提成-新方案", PersonnelRole: CommissionRoleSales,
		CalculationBasis: CommissionBasisRealizedProfit, RatePercent: decimal.RequireFromString("8.5"),
		EffectiveFrom: "2026-10-01", EmployeeIDs: []uuid.UUID{employeeID}, Today: "2026-09-18",
	}
	normalized, err := normalizeCopyCommissionRuleInput(base)
	if err != nil {
		t.Fatalf("合法复制输入被拒绝: %v", err)
	}
	if len(normalized.EmployeeIDs) != 1 || normalized.EffectiveFrom != "2026-10-01" {
		t.Fatalf("复制输入归一化结果不符: %#v", normalized)
	}

	cases := []struct {
		name   string
		mutate func(*CopyCommissionRuleInput)
	}{
		{name: "缺少源方案", mutate: func(in *CopyCommissionRuleInput) { in.SourceRuleID = uuid.Nil }},
		{name: "缺少员工", mutate: func(in *CopyCommissionRuleInput) { in.EmployeeIDs = nil }},
		{name: "起始日早于业务日期", mutate: func(in *CopyCommissionRuleInput) { in.EffectiveFrom = "2026-09-17" }},
		{name: "起始日非法", mutate: func(in *CopyCommissionRuleInput) { in.EffectiveFrom = "20261301" }},
		{name: "比例为零", mutate: func(in *CopyCommissionRuleInput) { in.RatePercent = decimal.NewFromInt(0) }},
		{name: "终止日早于起始日", mutate: func(in *CopyCommissionRuleInput) { to := "2026-09-30"; in.EffectiveTo = &to }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			if _, err := normalizeCopyCommissionRuleInput(in); !kratoserrors.IsBadRequest(err) {
				t.Fatalf("非法复制输入应返回 400，实际 error = %v", err)
			}
		})
	}
}

func TestCreateRuleRequiresEmployeesWhenEnabled(t *testing.T) {
	usecase := NewCommissionUsecase(nil, nil, nil)
	org, actor := uuid.New(), uuid.New()
	created, err := usecase.CreateRule(t.Context(), org, actor, CreateCommissionRuleInput{
		Name: "启用缺员工", PersonnelRole: CommissionRoleSales, CalculationBasis: CommissionBasisRealizedProfit,
		RatePercent: decimal.NewFromInt(5), Enabled: true,
		EffectiveFrom: stringPtr("2030-01-01"), Today: "2026-09-18",
	})
	if !errors.Is(err, ErrCommissionRuleEnableNoEmployee) {
		t.Fatalf("启用方案缺员工应返回 ENABLE_NO_EMPLOYEE，实际 error = %v", err)
	}
	if created != nil {
		t.Fatalf("失败创建不应返回方案")
	}
}
