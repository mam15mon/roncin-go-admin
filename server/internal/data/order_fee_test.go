package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestFeeSettingAppliesToOrderContext(t *testing.T) {
	serviceTypeID := uuid.Must(uuid.NewV7())
	abnormalCaseID := uuid.Must(uuid.NewV7())
	applicability := &orderFeeApplicability{
		serviceTypeIDs:  map[uuid.UUID]struct{}{serviceTypeID: {}},
		abnormalCaseIDs: map[uuid.UUID]struct{}{abnormalCaseID: {}},
	}
	tests := []struct {
		name    string
		setting *ent.FeeSetting
		want    bool
	}{
		{name: "通用费用", setting: &ent.FeeSetting{}, want: true},
		{name: "服务类型匹配", setting: &ent.FeeSetting{ServiceTypeID: &serviceTypeID}, want: true},
		{name: "异常情况匹配", setting: &ent.FeeSetting{AbnormalCaseID: &abnormalCaseID}, want: true},
		{name: "两个条件均匹配", setting: &ent.FeeSetting{ServiceTypeID: &serviceTypeID, AbnormalCaseID: &abnormalCaseID}, want: true},
		{name: "服务类型不匹配", setting: &ent.FeeSetting{ServiceTypeID: uuidPointerForOrderFeeTest()}, want: false},
		{name: "异常情况不匹配", setting: &ent.FeeSetting{AbnormalCaseID: uuidPointerForOrderFeeTest()}, want: false},
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
