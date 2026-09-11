package biz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestParseSharedDecimalRejectsInvalidScaleAndNonPositive(t *testing.T) {
	if got, err := ParseSharedWeight("100.125"); err != nil || !got.Equal(decimal.RequireFromString("100.125")) {
		t.Fatalf("合法毛重解析失败: got=%s err=%v", got, err)
	}
	if _, err := ParseSharedWeight("100.1251"); err == nil {
		t.Fatal("毛重超过三位小数应被拒绝")
	}
	if _, err := ParseSharedVolume("0"); err == nil {
		t.Fatal("零体积应被拒绝")
	}
	if _, err := ParseSharedVolume(" 1.000000"); err == nil {
		t.Fatal("带首尾空白的精确数值应被拒绝")
	}
}

func TestValidateSeaSharedConservationDraftRejectsContainerAndCargoExceeded(t *testing.T) {
	cargoID := uuid.New()
	baseline := map[uuid.UUID]SeaSharedQuantity{
		cargoID: {PackageCount: 10, GrossWeightKg: decimal.RequireFromString("100.000"), VolumeCbm: decimal.RequireFromString("5.000000")},
	}
	input := SeaSharedContainerAllocationInput{CargoItemID: cargoID, PackageCount: 11, GrossWeightKg: decimal.RequireFromString("100.000"), VolumeCbm: decimal.RequireFromString("5.000000")}
	allocated := map[uuid.UUID]SeaSharedQuantity{cargoID: {PackageCount: 11, GrossWeightKg: input.GrossWeightKg, VolumeCbm: input.VolumeCbm}}
	err := ValidateSeaSharedConservation(SeaSharedQuantity{PackageCount: 20, GrossWeightKg: decimal.RequireFromString("200"), VolumeCbm: decimal.RequireFromString("10")}, []SeaSharedContainerAllocationInput{input}, baseline, allocated, false)
	if err != ErrSeaSharedContainerExceeded {
		t.Fatalf("货物超分 error=%v, want ErrSeaSharedContainerExceeded", err)
	}

	input.PackageCount = 10
	allocated[cargoID] = SeaSharedQuantity{PackageCount: 10, GrossWeightKg: input.GrossWeightKg, VolumeCbm: input.VolumeCbm}
	err = ValidateSeaSharedConservation(SeaSharedQuantity{PackageCount: 9, GrossWeightKg: decimal.RequireFromString("200"), VolumeCbm: decimal.RequireFromString("10")}, []SeaSharedContainerAllocationInput{input}, baseline, allocated, false)
	if err != ErrSeaSharedContainerExceeded {
		t.Fatalf("共享箱超分 error=%v, want ErrSeaSharedContainerExceeded", err)
	}
}

func TestValidateSeaSharedConservationConfirmRequiresExactBalance(t *testing.T) {
	cargoID := uuid.New()
	baseline := SeaSharedQuantity{PackageCount: 10, GrossWeightKg: decimal.RequireFromString("100.000"), VolumeCbm: decimal.RequireFromString("5.000000")}
	input := SeaSharedContainerAllocationInput{CargoItemID: cargoID, PackageCount: 9, GrossWeightKg: decimal.RequireFromString("90.000"), VolumeCbm: decimal.RequireFromString("4.500000")}
	allocated := map[uuid.UUID]SeaSharedQuantity{cargoID: {PackageCount: 9, GrossWeightKg: input.GrossWeightKg, VolumeCbm: input.VolumeCbm}}
	if err := ValidateSeaSharedConservation(baseline, []SeaSharedContainerAllocationInput{input}, map[uuid.UUID]SeaSharedQuantity{cargoID: baseline}, allocated, true); err != ErrSeaSharedContainerIncomplete {
		t.Fatalf("未严格守恒 error=%v, want ErrSeaSharedContainerIncomplete", err)
	}

	input.PackageCount = baseline.PackageCount
	input.GrossWeightKg = baseline.GrossWeightKg
	input.VolumeCbm = baseline.VolumeCbm
	allocated[cargoID] = baseline
	if err := ValidateSeaSharedConservation(baseline, []SeaSharedContainerAllocationInput{input}, map[uuid.UUID]SeaSharedQuantity{cargoID: baseline}, allocated, true); err != nil {
		t.Fatalf("严格守恒应通过: %v", err)
	}
}

func TestNormalizeSeaSharedAllocationsRejectsDuplicateCargo(t *testing.T) {
	cargoID := uuid.New()
	base := &SeaSharedContainerAllocationInput{OrderID: uuid.New(), HouseBillID: uuid.New(), CargoItemID: cargoID, PackageCount: 1, GrossWeightKg: decimal.RequireFromString("1"), VolumeCbm: decimal.RequireFromString("1")}
	copy := *base
	if _, err := NormalizeSeaSharedAllocations([]*SeaSharedContainerAllocationInput{base, &copy}); err != ErrSeaSharedContainerInvalidArgument {
		t.Fatalf("重复货物行 error=%v, want ErrSeaSharedContainerInvalidArgument", err)
	}
}
