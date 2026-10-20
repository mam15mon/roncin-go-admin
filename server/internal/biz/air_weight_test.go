package biz

import (
	"github.com/google/uuid"
	"testing"
)

func TestAirWeightReferenceCalculation(t *testing.T) {
	for _, tc := range []struct {
		name, g, v, d    string
		n, den           int32
		want, volumetric string
	}{
		{"默认取大", "100", "1.2", "6000", 1, 1, "200", "200"},
		{"五千除数", "100", "1.2", "5000", 1, 1, "240", "240"},
		{"三分之一分泡", "100", "0.96", "6000", 1, 3, "120", "160"},
		{"实重大于体积重", "200", "0.96", "6000", 1, 3, "200", "160"},
		{"仅最终舍入", "100", "1", "6000", 1, 1, "166.667", "166.666667"},
		{"主单独立取大", "400", "1.8", "6000", 1, 1, "400", "300"},
		{"分单甲", "100", "1.2", "6000", 1, 1, "200", "200"},
		{"分单乙", "300", "0.6", "6000", 1, 1, "300", "100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, e := BuildAirWeightPreview(&AirWeightInput{Scope: "ORDER", TargetID: uuid.New(), VolumetricDivisor: tc.d, BubbleShareNumerator: tc.n, BubbleShareDenominator: tc.den}, &AirWeightFacts{GrossWeightKG: tc.g, VolumeCBM: tc.v, HasCargo: true}, nil)
			if e != nil || !p.Available || p.SuggestedChargeableWeightKG != tc.want || p.VolumetricWeightKG != tc.volumetric {
				t.Fatalf("参考计算错误: %+v %v", p, e)
			}
		})
	}
}
func TestAirWeightInvalidAndStale(t *testing.T) {
	input := &AirWeightInput{Scope: "ORDER", TargetID: uuid.New()}
	for _, bad := range []*AirWeightInput{
		{Scope: "ORDER", TargetID: input.TargetID, VolumetricDivisor: "0"},
		{Scope: "ORDER", TargetID: input.TargetID, BubbleShareNumerator: 2, BubbleShareDenominator: 1},
		{Scope: "ORDER", TargetID: input.TargetID, BubbleShareNumerator: 1, BubbleShareDenominator: 0},
		{Scope: "ORDER", TargetID: input.TargetID, VolumetricDivisor: "10000/3"},
	} {
		if _, e := NormalizeAirWeightInput(bad); e == nil {
			t.Fatal("非法规则被接受")
		}
	}
	p, e := BuildAirWeightPreview(input, &AirWeightFacts{GrossWeightKG: "0", VolumeCBM: "0"}, nil)
	if e != nil || p.Available || p.Reason == "" {
		t.Fatalf("缺数据不可伪造参考: %+v %v", p, e)
	}
	facts := &AirWeightFacts{GrossWeightKG: "100", VolumeCBM: "1.2", HasCargo: true, Identity: "cargo-v1"}
	p, e = BuildAirWeightPreview(input, facts, nil)
	if e != nil {
		t.Fatal(e)
	}
	c := &AirWeightConfirmation{InputFingerprint: p.InputFingerprint, ConfirmedWeightKG: "125"}
	facts.Identity = "cargo-v2"
	p, e = BuildAirWeightPreview(input, facts, c)
	if e != nil || !p.ConfirmationStale || p.CurrentConfirmation.ConfirmedWeightKG != "125" {
		t.Fatalf("变更须保留旧确认并标记过期: %+v %v", p, e)
	}
}
func TestFeeQuantitySelectionModes(t *testing.T) {
	id := uuid.New()
	for _, s := range []*FeeQuantitySelection{
		{Mode: "DIRECT"},
		{Mode: "MANUAL", ConfirmationID: id, ConfirmationVersion: 1},
		{Mode: "MANUAL_ADJUSTED", ConfirmationID: id, ConfirmationVersion: 1},
	} {
		if _, err := NormalizeFeeQuantitySelection(s); err == nil {
			t.Fatal("非法来源模式被接受")
		}
	}
	if _, err := NormalizeFeeQuantitySelection(&FeeQuantitySelection{Mode: "MANUAL_ADJUSTED", ConfirmationID: id, ConfirmationVersion: 1, Reason: "客户确认调整"}); err != nil {
		t.Fatal(err)
	}
}

func TestBilledFeeQuantitySourceRequiresQuantityPermission(t *testing.T) {
	current := billedOrderFeeForTest()
	requested := *current
	requested.QuantitySource = &FeeQuantitySelection{Mode: "DIRECT", ConfirmationID: uuid.New(), ConfirmationVersion: 1}
	if err := ValidateBilledFeeUpdate(current, &requested, &BilledFeeEditPolicy{Enabled: true}); err != ErrBilledFeeFieldForbidden {
		t.Fatalf("来源变化属于数量变更: %v", err)
	}
	if err := ValidateBilledFeeUpdate(current, &requested, &BilledFeeEditPolicy{Enabled: true, EditableFields: []BilledFeeEditableField{BilledFeeFieldQuantity}}); err != nil {
		t.Fatal(err)
	}
}
