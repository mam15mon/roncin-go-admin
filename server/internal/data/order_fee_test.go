package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestFeeSettingAppliesToOrderContext(t *testing.T) {
	abnormalCaseID := uuid.Must(uuid.NewV7())
	applicability := &orderFeeApplicability{
		abnormalCaseIDs: map[uuid.UUID]struct{}{abnormalCaseID: {}},
	}
	tests := []struct {
		name    string
		setting *ent.FeeSetting
		want    bool
	}{
		{name: "无异常类型挂载", setting: &ent.FeeSetting{}, want: true},
		{name: "异常类型匹配", setting: &ent.FeeSetting{AbnormalCaseID: &abnormalCaseID}, want: true},
		{name: "异常类型不匹配", setting: &ent.FeeSetting{AbnormalCaseID: uuidPointerForOrderFeeTest()}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := feeSettingApplies(test.setting, applicability); got != test.want {
				t.Fatalf("费用设置适用性结果为 %v，期望 %v", got, test.want)
			}
		})
	}
}

func uuidPointerForOrderFeeTest() *uuid.UUID {
	value := uuid.Must(uuid.NewV7())
	return &value
}
