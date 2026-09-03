package biz

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestParseAndValidateWeightAndVolume(t *testing.T) {
	// 正常解析
	w, err := ParseAndValidateWeight("100.123")
	if err != nil || w.String() != "100.123" {
		t.Fatalf("unexpected weight result: %v, err: %v", w, err)
	}
	v, err := ParseAndValidateVolume("2.123456")
	if err != nil || v.String() != "2.123456" {
		t.Fatalf("unexpected volume result: %v, err: %v", v, err)
	}

	// 精度超限
	if _, err := ParseAndValidateWeight("100.1234"); err == nil {
		t.Fatal("expected error for 4 decimal places weight")
	}
	if _, err := ParseAndValidateVolume("2.1234567"); err == nil {
		t.Fatal("expected error for 7 decimal places volume")
	}

	// 负数与零
	if _, err := ParseAndValidateWeight("0"); err == nil {
		t.Fatal("expected error for zero weight")
	}
	if _, err := ParseAndValidateVolume("-1.0"); err == nil {
		t.Fatal("expected error for negative volume")
	}

	// 空字符串
	if _, err := ParseAndValidateWeight(""); err == nil {
		t.Fatal("expected error for empty weight")
	}
}

func TestValidateDraftAllocations_PrefersSpecificHouseBillExceededError(t *testing.T) {
	cargoID := uuid.New()
	hbID := uuid.New()
	err := ValidateDraftAllocations(
		[]*OrderCargoItem{{ID: cargoID, CargoName: "货物A", PackageCount: 80, GrossWeightKg: 80, VolumeCbm: 80}},
		nil,
		[]*SeaHouseBill{{ID: hbID, HouseNo: "HBL001"}},
		[]*SeaCargoAllocationInput{{CargoItemID: cargoID, HouseBillID: hbID, PackageCount: 100, GrossWeightKg: decimal.NewFromInt(80), VolumeCbm: decimal.NewFromInt(80)}},
		"LCL",
	)
	if err == nil || !strings.Contains(err.Error(), "HBL001") || !strings.Contains(err.Error(), "100") || !strings.Contains(err.Error(), "80") {
		t.Fatalf("应返回可定位的 HBL 超量错误，实际：%v", err)
	}
	metadata := errorsMetadata(t, err)
	if metadata["object_type"] != "house_bill" || metadata["house_bill_id"] != hbID.String() || metadata["excess_value"] != "20" {
		t.Fatalf("HBL 超量 metadata 不完整：%v", metadata)
	}
}

func TestValidateDraftAllocations_InvalidReferencesAreLocatable(t *testing.T) {
	cargoID := uuid.New()
	hbID := uuid.New()
	containerID := uuid.New()
	validCargo := []*OrderCargoItem{{ID: cargoID, CargoName: "货物A", PackageCount: 1, GrossWeightKg: 1, VolumeCbm: 1}}
	validHB := []*SeaHouseBill{{ID: hbID, HouseNo: "HBL001"}}
	validContainer := []*OrderContainer{{ID: containerID, ContainerNo: "MSCU001", PackageCount: 1, GrossWeightKg: 1, VolumeCbm: 1}}

	tests := []struct {
		name       string
		input      *SeaCargoAllocationInput
		cargo      []*OrderCargoItem
		hbs        []*SeaHouseBill
		containers []*OrderContainer
		objectType string
		objectID   uuid.UUID
	}{
		{"货物", &SeaCargoAllocationInput{CargoItemID: uuid.New(), HouseBillID: hbID, PackageCount: 1, GrossWeightKg: decimal.NewFromInt(1), VolumeCbm: decimal.NewFromInt(1)}, validCargo, validHB, nil, "cargo_item", uuid.Nil},
		{"分单", &SeaCargoAllocationInput{CargoItemID: cargoID, HouseBillID: uuid.New(), PackageCount: 1, GrossWeightKg: decimal.NewFromInt(1), VolumeCbm: decimal.NewFromInt(1)}, validCargo, validHB, nil, "house_bill", uuid.Nil},
		{"实际箱", &SeaCargoAllocationInput{CargoItemID: cargoID, HouseBillID: hbID, ContainerID: func() *uuid.UUID { id := uuid.New(); return &id }(), PackageCount: 1, GrossWeightKg: decimal.NewFromInt(1), VolumeCbm: decimal.NewFromInt(1)}, validCargo, validHB, validContainer, "container", uuid.Nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.objectType {
			case "cargo_item":
				tt.objectID = tt.input.CargoItemID
			case "house_bill":
				tt.objectID = tt.input.HouseBillID
			case "container":
				tt.objectID = *tt.input.ContainerID
			}
			err := ValidateDraftAllocations(tt.cargo, tt.containers, tt.hbs, []*SeaCargoAllocationInput{tt.input}, "FCL")
			metadata := errorsMetadata(t, err)
			if metadata["object_type"] != tt.objectType || metadata["object_id"] != tt.objectID.String() || metadata[tt.objectType+"_id"] != tt.objectID.String() {
				t.Fatalf("非法引用 metadata 不完整：%v", metadata)
			}
		})
	}
}

func errorsMetadata(t *testing.T, err error) map[string]string {
	t.Helper()
	if err == nil {
		t.Fatal("期望错误，实际为 nil")
	}
	kratosErr, ok := err.(interface{ GetMetadata() map[string]string })
	if !ok {
		t.Fatalf("错误不含 metadata：%v", err)
	}
	return kratosErr.GetMetadata()
}

func TestValidateDraftAllocations_Exceeded(t *testing.T) {
	cargoID := uuid.New()
	hbID := uuid.New()
	cntrID := uuid.New()

	cargoItems := []*OrderCargoItem{
		{
			ID:            cargoID,
			CargoName:     "测试货物1",
			PackageCount:  10,
			GrossWeightKg: 100.0,
			VolumeCbm:     5.0,
		},
	}
	containers := []*OrderContainer{
		{
			ID:            cntrID,
			ContainerNo:   "MSKU1234567",
			PackageCount:  10,
			GrossWeightKg: 100.0,
			VolumeCbm:     5.0,
		},
	}
	houseBills := []*SeaHouseBill{
		{
			ID:      hbID,
			HouseNo: "HBL001",
		},
	}

	// 货物件数超分
	err := ValidateDraftAllocations(cargoItems, containers, houseBills, []*SeaCargoAllocationInput{
		{
			CargoItemID:   cargoID,
			HouseBillID:   hbID,
			ContainerID:   &cntrID,
			PackageCount:  11,
			GrossWeightKg: decimal.RequireFromString("100"),
			VolumeCbm:     decimal.RequireFromString("5"),
		},
	}, "FCL")
	if err == nil {
		t.Fatal("expected error for exceeded package count")
	}
	kErr, ok := err.(interface{ GetMetadata() map[string]string })
	if !ok || kErr.GetMetadata()["dimension"] != "package_count" {
		t.Fatalf("expected metadata dimension=package_count, got: %v", err)
	}
	if kErr.GetMetadata()["excess_value"] != "1" {
		t.Fatalf("expected excess_value=1, got: %v", kErr.GetMetadata()["excess_value"])
	}

	// 货物毛重超分
	err = ValidateDraftAllocations(cargoItems, containers, houseBills, []*SeaCargoAllocationInput{
		{
			CargoItemID:   cargoID,
			HouseBillID:   hbID,
			ContainerID:   &cntrID,
			PackageCount:  10,
			GrossWeightKg: decimal.RequireFromString("100.001"),
			VolumeCbm:     decimal.RequireFromString("5"),
		},
	}, "FCL")
	if err == nil {
		t.Fatal("expected error for exceeded gross weight")
	}
	kErr, ok = err.(interface{ GetMetadata() map[string]string })
	if !ok || kErr.GetMetadata()["dimension"] != "gross_weight_kg" {
		t.Fatalf("expected metadata dimension=gross_weight_kg, got: %v", err)
	}

	// 货物未超分合法保存草稿
	err = ValidateDraftAllocations(cargoItems, containers, houseBills, []*SeaCargoAllocationInput{
		{
			CargoItemID:   cargoID,
			HouseBillID:   hbID,
			ContainerID:   &cntrID,
			PackageCount:  5,
			GrossWeightKg: decimal.RequireFromString("50"),
			VolumeCbm:     decimal.RequireFromString("2.5"),
		},
	}, "FCL")
	if err != nil {
		t.Fatalf("expected draft to be valid, got: %v", err)
	}
}

func TestValidateConfirmedAllocations_Completeness(t *testing.T) {
	cargoID := uuid.New()
	hb1 := uuid.New()
	hb2 := uuid.New()
	cntrID := uuid.New()

	cargoItems := []*OrderCargoItem{
		{
			ID:            cargoID,
			CargoName:     "测试货物1",
			PackageCount:  10,
			GrossWeightKg: 100.0,
			VolumeCbm:     5.0,
		},
	}
	containers := []*OrderContainer{
		{
			ID:            cntrID,
			ContainerNo:   "MSKU1234567",
			PackageCount:  10,
			GrossWeightKg: 100.0,
			VolumeCbm:     5.0,
		},
	}
	houseBills := []*SeaHouseBill{
		{
			ID:      hb1,
			HouseNo: "HBL001",
		},
		{
			ID:      hb2,
			HouseNo: "HBL002",
		},
	}

	// 仅分配了部分件重尺，确认应被拒绝
	err := ValidateConfirmedAllocations(cargoItems, containers, houseBills, []*SeaCargoAllocationInput{
		{
			CargoItemID:   cargoID,
			HouseBillID:   hb1,
			ContainerID:   &cntrID,
			PackageCount:  5,
			GrossWeightKg: decimal.RequireFromString("50"),
			VolumeCbm:     decimal.RequireFromString("2.5"),
		},
	}, "FCL")
	if err == nil {
		t.Fatal("expected incomplete error when cargo not fully allocated")
	}

	// 货物分完但 hb2 没有任何分配，确认应被拒绝
	err = ValidateConfirmedAllocations(cargoItems, containers, houseBills, []*SeaCargoAllocationInput{
		{
			CargoItemID:   cargoID,
			HouseBillID:   hb1,
			ContainerID:   &cntrID,
			PackageCount:  10,
			GrossWeightKg: decimal.RequireFromString("100"),
			VolumeCbm:     decimal.RequireFromString("5"),
		},
	}, "FCL")
	if err == nil {
		t.Fatal("expected incomplete error when hb2 has no allocations")
	}

	// 两张分单完整分配，一箱多 HBL 守恒满足
	err = ValidateConfirmedAllocations(cargoItems, containers, houseBills, []*SeaCargoAllocationInput{
		{
			CargoItemID:   cargoID,
			HouseBillID:   hb1,
			ContainerID:   &cntrID,
			PackageCount:  6,
			GrossWeightKg: decimal.RequireFromString("60"),
			VolumeCbm:     decimal.RequireFromString("3"),
		},
		{
			CargoItemID:   cargoID,
			HouseBillID:   hb2,
			ContainerID:   &cntrID,
			PackageCount:  4,
			GrossWeightKg: decimal.RequireFromString("40"),
			VolumeCbm:     decimal.RequireFromString("2"),
		},
	}, "FCL")
	if err != nil {
		t.Fatalf("expected confirmed validation to succeed, got: %v", err)
	}
}
