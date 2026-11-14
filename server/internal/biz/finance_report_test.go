package biz

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFinanceReportFilterPaginationAndDates(t *testing.T) {
	org := uuid.New()
	for _, size := range []int{1, 200} {
		if _, err := normalizeFinanceReportFilter(org, FinanceReportFilter{Page: 1, PageSize: size, DateFrom: "2026-02-01", DateTo: "2026-02-28"}, false); err != nil {
			t.Fatalf("合法分页%d: %v", size, err)
		}
	}
	for _, f := range []FinanceReportFilter{{Page: 0, PageSize: 20}, {Page: 1, PageSize: 0}, {Page: 1, PageSize: 201}, {Page: 1, PageSize: 20, DateFrom: "2026-03-01", DateTo: "2026-02-28"}, {Page: 1, PageSize: 20, DateFrom: "2026-02-29"}, {Page: 1, PageSize: 20, Currency: "CN"}, {Page: 1, PageSize: 20, Bucket: "UNKNOWN"}} {
		if _, err := normalizeFinanceReportFilter(org, f, false); err == nil {
			t.Fatalf("应拒绝非法筛选: %#v", f)
		}
	}
	for _, f := range []FinanceReportFilter{{DateFrom: "2026-01-01"}, {DateTo: "2026-12-31"}, {}} {
		if _, err := normalizeFinanceReportFilter(org, f, true); err != nil {
			t.Fatalf("独立导出可省略分页且允许单边日期: %v", err)
		}
	}
}
func TestFinanceReportAgingBucket(t *testing.T) {
	// UTC时刻对应中国业务日次日，分桶必须按业务自然日。
	now := time.Date(2026, 11, 13, 17, 0, 0, 0, time.UTC)
	today := time.Date(2026, 11, 14, 0, 0, 0, 0, ExchangeRateBusinessLocation())
	for _, c := range []struct {
		days int
		want string
	}{{-1, "NOT_DUE"}, {0, "NOT_DUE"}, {1, "DAYS_1_30"}, {30, "DAYS_1_30"}, {31, "DAYS_31_60"}, {60, "DAYS_31_60"}, {61, "DAYS_61_90"}, {90, "DAYS_61_90"}, {91, "OVER_90"}} {
		due := today.AddDate(0, 0, -c.days).Format("2006-01-02")
		bucket, days, err := FinanceReportAgingBucket(&due, now)
		if err != nil || bucket != c.want || days == nil {
			t.Fatalf("差%d天应%s，实际%s/%v/%v", c.days, c.want, bucket, days, err)
		}
	}
	bucket, days, err := FinanceReportAgingBucket(nil, now)
	if bucket != "UNDATED" || days != nil || err != nil {
		t.Fatalf("缺日期应保留存在性: %s %v %v", bucket, days, err)
	}
	invalid := "bad-date"
	if _, _, err := FinanceReportAgingBucket(&invalid, now); err == nil {
		t.Fatal("非法非空日期不能吞成未设置")
	}
}
