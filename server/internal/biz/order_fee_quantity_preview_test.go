package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestFeeQuantityPreviewRules(t *testing.T) {
	tests := []struct {
		name, code, source, weight, volume, want, reason string
		containers, cargo                                int
		container, matched, shared, integer              bool
	}{
		{name: "体积", code: "CBM", source: "ACTUAL", weight: "2500", volume: "8", cargo: 1, want: "8"},
		{name: "毛重", code: "KG", source: "ACTUAL", weight: "2500", volume: "8", cargo: 1, want: "2500"},
		{name: "吨", code: "TON", source: "ACTUAL", weight: "2500", volume: "8", cargo: 1, want: "2.5"},
		{name: "RT体积大", code: "RT", source: "ACTUAL", weight: "2500", volume: "8", cargo: 1, want: "8"},
		{name: "RT重量大", code: "RT", source: "ACTUAL", weight: "9500", volume: "8", cargo: 1, want: "9.5"},
		{name: "计划3箱", code: "40HQ", source: "PLANNED", containers: 3, container: true, matched: true, want: "3"},
		{name: "实际2箱", code: "40HQ", source: "ACTUAL", containers: 2, container: true, matched: true, want: "2"},
		{name: "自然箱", code: "CONT", source: "ACTUAL", containers: 4, want: "4"},
		{name: "无别名回退", code: "40HC", source: "ACTUAL", containers: 2, container: true, reason: "无法精确"},
		{name: "共享箱拒绝整箱数", code: "CONT", source: "ACTUAL", containers: 2, shared: true, reason: "共享箱"},
		{name: "共享箱不影响计划", code: "CONT", source: "PLANNED", containers: 3, shared: true, want: "3"},
		{name: "无货物", code: "CBM", source: "ACTUAL", reason: "尚未登记"},
		{name: "无箱量", code: "CONT", source: "ACTUAL", reason: "没有可用"},
		{name: "不支持单位", code: "PLT", source: "ACTUAL", reason: "暂不支持"},
		{name: "货量不使用计划", code: "KG", source: "PLANNED", weight: "2500", cargo: 1, reason: "仅支持"},
		{name: "体积超精度不舍入", code: "CBM", source: "ACTUAL", volume: "8.123456", cargo: 1, reason: "四位小数"},
		{name: "吨超精度不截断", code: "TON", source: "ACTUAL", weight: "2500.001", cargo: 1, reason: "四位小数"},
		{name: "数值等价尾零允许", code: "CBM", source: "ACTUAL", volume: "8.120000", cargo: 1, want: "8.12"},
		{name: "超上限", code: "KG", source: "ACTUAL", weight: "10000000000", cargo: 1, reason: "上限"},
		{name: "遵循管理员整数规则", code: "TON", source: "ACTUAL", weight: "2500", cargo: 1, integer: true, reason: "要求整数"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parse := func(v string) decimal.Decimal {
				if v == "" {
					return decimal.Zero
				}
				return decimal.RequireFromString(v)
			}
			data := &FeeQuantitySource{UnitCode: tt.code, UnitEnabled: true, ContainerUnit: tt.container, ContainerSpecMatched: tt.matched, HasSharedContainers: tt.shared, ContainerQuantity: tt.containers, CargoCount: tt.cargo, GrossWeightKG: parse(tt.weight), VolumeCBM: parse(tt.volume), QuantityMustBeInteger: tt.integer}
			got := buildFeeQuantityPreview(data, tt.source)
			if tt.want != "" {
				if !got.Available || got.Quantity != tt.want || got.Formula == "" || len(got.Inputs) == 0 {
					t.Fatalf("预览=%+v，期望%s及计算依据", got, tt.want)
				}
			} else if got.Available || got.Quantity != "" || !strings.Contains(got.Reason, tt.reason) {
				t.Fatalf("预览=%+v，期望拒绝：%s", got, tt.reason)
			}
		})
	}
	for _, data := range []*FeeQuantitySource{nil, {UnitCode: "KG", UnitEnabled: false}} {
		if got := buildFeeQuantityPreview(data, "ACTUAL"); got.Available || !strings.Contains(got.Reason, "停用") {
			t.Fatalf("无效单位：%+v", got)
		}
	}
}

type feeQuantityRepoStub struct {
	OrderFeeRepo
	org, order, unit uuid.UUID
	source           string
	calls            int
}

func (r *feeQuantityRepoStub) QuantitySource(_ context.Context, org, order, unit uuid.UUID, source string) (*FeeQuantitySource, error) {
	r.org, r.order, r.unit, r.source = org, order, unit, source
	r.calls++
	return &FeeQuantitySource{UnitCode: "CONT", UnitEnabled: true, ContainerQuantity: 2}, nil
}
func TestPreviewFeeQuantityBoundaryAndScope(t *testing.T) {
	repo := &feeQuantityRepoStub{}
	uc := &OrderFeeUsecase{repo: repo}
	org, order, unit := uuid.New(), uuid.New(), uuid.New()
	for _, source := range []string{"", "actual", "OTHER"} {
		if _, err := uc.PreviewFeeQuantity(context.Background(), org, order, unit, source); err != ErrOrderFeeInvalidArgument {
			t.Fatalf("无效来源=%q：%v", source, err)
		}
	}
	if _, err := uc.PreviewFeeQuantity(context.Background(), uuid.Nil, order, unit, "ACTUAL"); err != ErrOrderFeeInvalidArgument {
		t.Fatal(err)
	}
	if repo.calls != 0 {
		t.Fatal("无效请求不可查询来源")
	}
	got, err := uc.PreviewFeeQuantity(context.Background(), org, order, unit, "ACTUAL")
	if err != nil || !got.Available || repo.org != org || repo.order != order || repo.unit != unit || repo.source != "ACTUAL" {
		t.Fatalf("组织/订单/单位来源未保留：%+v %v", repo, err)
	}
}
