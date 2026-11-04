package biz

import (
	"reflect"
	"strings"
	"testing"
)

func TestSeaHouseBillPrintDetailsValidation(t *testing.T) {
	text := "  上海港\nSHANGHAI  "
	validDate := "2026-11-04"
	zero := int32(0)
	input := &SeaHouseBillPrintDetails{PortOfLoadingText: &text, OriginalCount: &zero, ShippedOnBoardDate: &validDate}
	got, err := ValidateSeaHouseBillPrintDetails(input)
	if err != nil || got == nil || got.PortOfLoadingText == nil || *got.PortOfLoadingText != "上海港\nSHANGHAI" || got.OriginalCount == nil || *got.OriginalCount != 0 {
		t.Fatalf("有效印字校验失败: %#v %v", got, err)
	}
	if *input.PortOfLoadingText != text {
		t.Fatal("校验改写了输入")
	}
	if got, err := ValidateSeaHouseBillPrintDetails(nil); got != nil || err != nil {
		t.Fatal("未填写印字应保持空值")
	}
	cases := []struct {
		name  string
		value SeaHouseBillPrintDetails
	}{
		{name: "非法日历日期", value: SeaHouseBillPrintDetails{IssueDate: printText("2026-02-30")}},
		{name: "非标准日期格式", value: SeaHouseBillPrintDetails{ShippedOnBoardDate: printText("2026-1-04")}},
		{name: "负份数", value: SeaHouseBillPrintDetails{OriginalCount: printCount(-1)}},
		{name: "过大份数", value: SeaHouseBillPrintDetails{OriginalCount: printCount(100)}},
		{name: "控制字符", value: SeaHouseBillPrintDetails{BookingNoText: printText("B\x00L")}},
		{name: "短字段超长", value: SeaHouseBillPrintDetails{IssuePlaceText: printText(strings.Repeat("中", 501))}},
		{name: "联系信息超长", value: SeaHouseBillPrintDetails{HeaderContactText: printText(strings.Repeat("中", 3001))}},
		{name: "箱号明细超长", value: SeaHouseBillPrintDetails{ContainersText: printText(strings.Repeat("中", 12001))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ValidateSeaHouseBillPrintDetails(&tc.value); err == nil {
				t.Fatal("非法印字未被拒绝")
			}
		})
	}
}

func TestSeaHouseBillInputRetainsPrintDetails(t *testing.T) {
	details := &SeaHouseBillPrintDetails{IssueDate: printText("2026-11-04"), OriginalCount: printCount(3), ContainersText: printText("ABCD1234567 / SEAL-01")}
	input, err := ValidateSeaHouseBillInput(&SeaHouseBillInput{HouseNo: "HBL-001", IssuerSource: SeaHouseBillIssuerSourceSelfOrganization, PrintDetails: details})
	if err != nil || !reflect.DeepEqual(input.PrintDetails, details) {
		t.Fatalf("HBL 校验遗漏印字: %#v %v", input, err)
	}
}

func printText(value string) *string { return &value }
func printCount(value int32) *int32  { return &value }
