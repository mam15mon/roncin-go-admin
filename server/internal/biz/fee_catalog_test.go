package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNormalizeFeeSettingAllowsGeneralFee(t *testing.T) {
	input := validFeeSettingForTest()

	value, err := normalizeFeeSetting(input)
	if err != nil {
		t.Fatalf("通用费用设置应允许服务类型和异常情况为空: %v", err)
	}
	if value.ServiceTypeID != nil || value.AbnormalCaseID != nil {
		t.Fatalf("可空关联不应被补值: service_type_id=%v abnormal_case_id=%v", value.ServiceTypeID, value.AbnormalCaseID)
	}
}

func TestNormalizeFeeSettingRejectsInvalidCode(t *testing.T) {
	input := validFeeSettingForTest()
	input.FeeCode = "ocean-fee"

	if _, err := normalizeFeeSetting(input); err != ErrFeeCatalogInvalidArgument {
		t.Fatalf("非大写字母、数字和下划线格式的费用代码应被拒绝，实际错误为 %v", err)
	}
}

func TestNormalizeFeeSettingRejectsInvalidTaxRate(t *testing.T) {
	tests := []struct {
		name string
		rate string
	}{
		{name: "超过上限", rate: "100.01"},
		{name: "负数", rate: "-0.01"},
		{name: "超过两位小数", rate: "6.001"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validFeeSettingForTest()
			input.TaxRate = decimal.RequireFromString(test.rate)
			if _, err := normalizeFeeSetting(input); err != ErrFeeCatalogInvalidArgument {
				t.Fatalf("税率 %s 应被拒绝，实际错误为 %v", test.rate, err)
			}
		})
	}
}

func TestNormalizeFeeSettingRequiresCatalogReferences(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*FeeSetting)
	}{
		{name: "缺少计费单位", modify: func(value *FeeSetting) { value.BillingUnitID = uuid.Nil }},
		{name: "缺少应税劳务", modify: func(value *FeeSetting) { value.TaxableServiceID = uuid.Nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validFeeSettingForTest()
			test.modify(input)
			if _, err := normalizeFeeSetting(input); err != ErrFeeCatalogInvalidArgument {
				t.Fatalf("%s应被拒绝，实际错误为 %v", test.name, err)
			}
		})
	}
}

func validFeeSettingForTest() *FeeSetting {
	return &FeeSetting{
		FeeCode:          "OCEAN_FREIGHT",
		NameZH:           "海运费",
		DefaultCurrency:  "CNY",
		BillingUnitID:    uuid.Must(uuid.NewV7()),
		TaxRate:          decimal.RequireFromString("6.00"),
		TaxableServiceID: uuid.Must(uuid.NewV7()),
	}
}
