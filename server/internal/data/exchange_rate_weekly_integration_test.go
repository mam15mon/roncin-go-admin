package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// weekEndPointerForTest 返回锚定周一的自然周终点字符串指针（周日 23:59:59）。
func weekEndPointerForTest(weekFrom time.Time) *string {
	value := weekEndForTest(weekFrom).Format(time.RFC3339)
	return &value
}

// weekEndForTest 返回锚定周一的自然周终点（周日 23:59:59，Asia/Shanghai）。
func weekEndForTest(weekFrom time.Time) time.Time {
	return weekFrom.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
}

// createWeeklyRate 经仓储幂等 Upsert 写入一条周汇率行（organizationID 为 nil 时写
// NULL 基线行），source 决定费用快照区分 WEEKLY 与 BOC_SYNC。
func createWeeklyRate(t *testing.T, data *Data, organizationID *uuid.UUID, source, fromCurrency string, weekFrom time.Time, ar, ap string) *biz.ExchangeRateSetting {
	t.Helper()
	arRate := decimal.RequireFromString(ar)
	apRate := decimal.RequireFromString(ap)
	saved, err := NewExchangeRateRepo(data).UpsertWeeklyBatch(context.Background(), source, []*biz.ExchangeRateSetting{{
		ID:             uuid.Must(uuid.NewV7()),
		OrganizationID: organizationID,
		FromCurrency:   fromCurrency, ToCurrency: "CNY",
		EffectiveFrom: weekFrom.Format(time.RFC3339),
		EffectiveTo:   weekEndPointerForTest(weekFrom),
		ARRate:        &arRate, APRate: &apRate,
		Rate: arRate.Add(apRate).Div(decimal.NewFromInt(2)).RoundBank(8),
	}}, &biz.AuditEvent{Action: "test.weekly.upsert", Result: "success"})
	if err != nil {
		t.Fatalf("写入周汇率行 %s: %v", fromCurrency, err)
	}
	return saved[0]
}

func TestExchangeRateWeeklyDisasterChainPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	repo := NewExchangeRateRepo(data)

	// 当周（2026-09-14 ~ 2026-09-20）与上一周（2026-09-07 ~ 2026-09-13）。
	currentWeek := time.Date(2026, 9, 14, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())
	lastWeek := currentWeek.AddDate(0, 0, -7)
	// 基线兜底行：更早的周（NULL 基线，总部维护）。
	baselineWeek := time.Date(2026, 8, 31, 0, 0, 0, 0, biz.ExchangeRateBusinessLocation())

	weeklyDate := "2026-09-16" // 当周三
	// 一级：本组织当周行（BOC_SYNC 来源，ar/ap 双轨）。
	current := createWeeklyRate(t, data, &branchID, biz.ExchangeRateSettingSourceBOCSync, "USD", currentWeek, "6.7500", "6.7000")
	// 二级：上一周组织行（手工维护，ar/ap 双轨）。
	inherited := createWeeklyRate(t, data, &branchID, biz.ExchangeRateSettingSourceManual, "JPY", lastWeek, "0.0510", "0.0490")
	_ = inherited
	// 三级：NULL 基线行（当周，总部维护，只有 rate 基准价）。
	if _, err := data.db.ExchangeRateSetting.Create().
		SetFromCurrency("EUR").SetToCurrency("CNY").
		SetEffectiveFrom(baselineWeek).SetEffectiveTo(weekEndForTest(baselineWeek)).
		SetRate("7.85000000").SetIsActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建 EUR 基线行: %v", err)
	}
	// 三级兜底的第二腿（KRW→CNY 基线行）。
	if _, err := data.db.ExchangeRateSetting.Create().
		SetFromCurrency("KRW").SetToCurrency("CNY").
		SetEffectiveFrom(baselineWeek).SetEffectiveTo(weekEndForTest(baselineWeek)).
		SetRate("0.00520000").SetIsActive(true).
		Save(ctx); err != nil {
		t.Fatalf("创建 KRW 基线行: %v", err)
	}
	_ = hqID

	t.Run("一级命中本组织当周行且按应收方向取 ar_rate", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", weeklyDate)
		if err != nil {
			t.Fatalf("当周应收解析失败: %v", err)
		}
		if resolved.Rate.StringFixed(4) != "6.7500" || resolved.Source != biz.ExchangeRateSourceBOCSync {
			t.Fatalf("应收应命中当周 ar_rate/BOC_SYNC，实际 %s/%s", resolved.Rate, resolved.Source)
		}
		if resolved.SettingID == nil || *resolved.SettingID != current.ID {
			t.Fatalf("快照应携带命中行 ID: %v", resolved.SettingID)
		}
	})
	t.Run("一级命中本组织当周行且按应付方向取 ap_rate", func(t *testing.T) {
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeePayable, "USD", "CNY", "CNY", weeklyDate)
		if err != nil {
			t.Fatalf("当周应付解析失败: %v", err)
		}
		if resolved.Rate.StringFixed(4) != "6.7000" || resolved.Source != biz.ExchangeRateSourceBOCSync {
			t.Fatalf("应付应命中当周 ap_rate/BOC_SYNC，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})
	t.Run("二级回溯最近历史周标记 INHERITED_LAST_WEEK", func(t *testing.T) {
		// JPY 只配了上周行；当周三开单自动继承（绝不阻断保存）。
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "JPY", "CNY", "CNY", weeklyDate)
		if err != nil {
			t.Fatalf("跨周继承解析失败: %v", err)
		}
		if resolved.Rate.StringFixed(4) != "0.0510" || resolved.Source != biz.ExchangeRateSourceInheritedLastWeek {
			t.Fatalf("跨周继承应取最近历史周 ar_rate/INHERITED_LAST_WEEK，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})
	t.Run("三级无组织行时直连 NULL 基线行并回落 rate 列", func(t *testing.T) {
		// 基线行只有 rate 基准价（无 ar/ap），应收/应付均回落基准价，来源 SYSTEM。
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "EUR", "CNY", "CNY", weeklyDate)
		if err != nil {
			t.Fatalf("基线直连解析失败: %v", err)
		}
		if resolved.Rate.StringFixed(8) != "7.85000000" || resolved.Source != biz.ExchangeRateSourceSystem {
			t.Fatalf("基线直连应为 7.85000000/SYSTEM，实际 %s/%s", resolved.Rate, resolved.Source)
		}
	})
	t.Run("四级经 NULL 基线 pivot 交叉套算标记 DERIVED", func(t *testing.T) {
		// KRW→EUR 无直连行（组织行与基线行均无该货币对），经 KRW、EUR→CNY 基线腿套算。
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "KRW", "EUR", "CNY", weeklyDate)
		if err != nil {
			t.Fatalf("基线交叉套算失败: %v", err)
		}
		expected := decimal.RequireFromString("0.00520000").Div(decimal.RequireFromString("7.85000000")).RoundBank(8)
		if !resolved.Rate.Equal(expected) || resolved.Source != biz.ExchangeRateSourceDerived {
			t.Fatalf("四级套算应为 %s/DERIVED，实际 %s/%s", expected, resolved.Rate, resolved.Source)
		}
	})
	t.Run("全链缺失时 fail-closed", func(t *testing.T) {
		// GBP 无任何行：直接缺失返回业务错误（由现场手工覆盖兜底）。
		if _, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "GBP", "CNY", "CNY", weeklyDate); err == nil {
			t.Fatal("GBP 全链缺失应返回业务错误")
		}
	})
	t.Run("同周幂等 Upsert 覆盖更新且不产生新行", func(t *testing.T) {
		updated := createWeeklyRate(t, data, &branchID, biz.ExchangeRateSettingSourceBOCSync, "USD", currentWeek, "6.8800", "6.8200")
		if updated.ID != current.ID {
			t.Fatalf("同周重复发布应覆盖同一行: old=%s new=%s", current.ID, updated.ID)
		}
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", weeklyDate)
		if err != nil || resolved.Rate.StringFixed(4) != "6.8800" {
			t.Fatalf("覆盖更新后应收应取新值，实际 %s err=%v", resolved.Rate, err)
		}
	})
	t.Run("跨周预设下周行不影响当周解析", func(t *testing.T) {
		nextWeek := currentWeek.AddDate(0, 0, 7)
		createWeeklyRate(t, data, &branchID, biz.ExchangeRateSettingSourceManual, "USD", nextWeek, "7.0500", "6.9800")
		resolved, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", weeklyDate)
		if err != nil || resolved.Rate.StringFixed(4) != "6.8800" {
			t.Fatalf("预设下周不应影响当周解析: %s err=%v", resolved.Rate, err)
		}
		future, err := repo.ResolveRate(ctx, branchID, biz.OrderFeeReceivable, "USD", "CNY", "CNY", "2026-09-23")
		if err != nil || future.Rate.StringFixed(4) != "7.0500" || future.Source != biz.ExchangeRateSourceWeekly {
			t.Fatalf("下周应命中预设行 WEEKLY: %s/%s err=%v", future.Rate, future.Source, err)
		}
	})
	t.Run("List 支持分页、总数统计与生效周倒序", func(t *testing.T) {
		items, total, err := repo.List(ctx, branchID, biz.ExchangeRateListOptions{Page: 1, PageSize: 2})
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if total < 3 {
			t.Fatalf("应至少有 3 条汇率行，实际 %d", total)
		}
		if len(items) != 2 {
			t.Fatalf("pageSize=2 时应返回 2 条，实际 %d", len(items))
		}
		// 验证倒序：第一条生效周 >= 第二条生效周
		if items[0].EffectiveFrom < items[1].EffectiveFrom {
			t.Fatalf("生效周应倒序排列，第一条 %s，第二条 %s", items[0].EffectiveFrom, items[1].EffectiveFrom)
		}

		// 验证币种过滤
		usdItems, usdTotal, err := repo.List(ctx, branchID, biz.ExchangeRateListOptions{Page: 1, PageSize: 10, FromCurrency: "USD"})
		if err != nil {
			t.Fatalf("按 USD 过滤失败: %v", err)
		}
		if usdTotal == 0 || len(usdItems) == 0 {
			t.Fatalf("应查出 USD 汇率行")
		}
		for _, item := range usdItems {
			if item.FromCurrency != "USD" {
				t.Fatalf("过滤结果币种应为 USD，实际 %s", item.FromCurrency)
			}
		}
	})
}
