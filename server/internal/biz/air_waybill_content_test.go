package biz

import (
	"strings"
	"testing"
)

func TestAirWaybillDecimalTotalsAndDeclarationSemantics(t *testing.T) {
	input := &AirWaybillContent{Currency: "USD", WeightPayment: "PPD", OtherPayment: "COLL", CarriageValue: &AirDeclaredValue{Kind: "NVD"}, CustomsValue: &AirDeclaredValue{Kind: "AMOUNT", Amount: "0"}, InsuranceValue: &AirDeclaredValue{Kind: "XXX"}, RateLines: []*AirRateLine{{Pieces: "0", GrossWeightKg: "0", ChargeableWeightKg: "10.125", Rate: "0.12345678", Amount: "0.1"}, {Amount: "0.2"}}, OtherCharges: []*AirOtherCharge{{Name: "代理费", Currency: "USD", Amount: "1.11", Payment: "COLL", DueTo: "AGENT"}, {Name: "承运费", Currency: "USD", Amount: "2.22", Payment: "COLL", DueTo: "CARRIER"}}, Prepaid: &AirChargeSummary{ValuationCharge: "0.07", Tax: "0.03", Total: "999"}, Collect: &AirChargeSummary{Tax: "0.04"}}
	got, err := NormalizeAirWaybillContent(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prepaid.WeightCharge != "0.3" || got.Prepaid.Total != "0.4" || got.Collect.OtherDueAgent != "1.11" || got.Collect.OtherDueCarrier != "2.22" || got.Collect.Total != "3.37" {
		t.Fatalf("十进制合计错误：%+v %+v", got.Prepaid, got.Collect)
	}
	if got.CustomsValue.Amount != "0" || got.CarriageValue.Kind != "NVD" || got.InsuranceValue.Kind != "XXX" {
		t.Fatal("零声明和特殊声明被混用")
	}
	if input.Prepaid.Total != "999" {
		t.Fatal("规范化修改了调用方对象")
	}
	input.AsArranged = true
	got, err = NormalizeAirWaybillContent(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prepaid.WeightCharge != "" || got.Prepaid.Total != "" || got.Collect.Total != "3.37" {
		t.Fatalf("AS ARRANGED 被冒充数字：%+v", got)
	}
}

func TestAirWaybillContentLimitsAndExplicitEmpty(t *testing.T) {
	input := &AirWaybillContent{ShipperText: strings.Repeat("中", 5000), Marks: strings.Repeat("唛", 10000), GoodsDescription: strings.Repeat("货", 10000), Remarks: "  原样\n换行  ", IssueDate: "2026-11-16", FlightDate: "2026-11-17"}
	got, err := NormalizeAirWaybillContent(input)
	if err != nil || got.Remarks != input.Remarks {
		t.Fatalf("边界或正文保留失败：%v", err)
	}
	hash := AirWaybillContentHash(got)
	if len(hash) != 64 {
		t.Fatal("缺少内容哈希")
	}
	input.ShipperText += "中"
	if _, err = NormalizeAirWaybillContent(input); err == nil {
		t.Fatal("超长当事人放行")
	}
	input.ShipperText = ""
	input.Marks += "唛"
	if _, err = NormalizeAirWaybillContent(input); err == nil {
		t.Fatal("超长唛头放行")
	}
	if v, e := NormalizeAirWaybillContent(nil); v != nil || e != nil {
		t.Fatal("省略内容被变成清空")
	}
	if v, e := NormalizeAirWaybillContent(&AirWaybillContent{}); v == nil || e != nil {
		t.Fatal("显式清空丢失存在性")
	}
	if AirWaybillContentHash(&AirWaybillContent{}) == hash {
		t.Fatal("清空未改变哈希")
	}
}

func TestAirWaybillRejectsInvalidValuesWithoutGuessing(t *testing.T) {
	cases := []*AirWaybillContent{
		{CarriageValue: &AirDeclaredValue{Kind: "NVD", Amount: "0"}},
		{CustomsValue: &AirDeclaredValue{Kind: "NVD"}},
		{InsuranceValue: &AirDeclaredValue{Kind: "AMOUNT", Amount: "XXX"}},
		{IssueDate: "2026-02-30"}, {IssueDate: "2026-11-16T00:00:00Z"}, {Currency: "US"},
		{CurrencyConversionRate: "NaN"}, {WeightPayment: "PP"}, {AsArranged: true, Currency: "USD"},
		{Currency: "USD", WeightPayment: "PPD", RateLines: []*AirRateLine{{Amount: "1,000"}}},
		{Currency: "USD", OtherCharges: []*AirOtherCharge{{Currency: "CNY", Amount: "1", Payment: "PPD", DueTo: "AGENT"}}},
	}
	for i, input := range cases {
		if _, err := NormalizeAirWaybillContent(input); err == nil {
			t.Errorf("非法场景%d放行", i)
		}
	}
}
