package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSeaSharedContainerMappingPreservesDecimalStrings(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	item := &biz.SeaSharedContainer{
		ID:                   uuid.New(),
		OrganizationID:       uuid.New(),
		TransportExecutionID: uuid.New(),
		ContainerNo:          "MSCU1234567",
		ContainerSpecID:      uuid.New(),
		ContainerSpecName:    "40HQ",
		PackageCount:         12,
		GrossWeightKg:        decimal.RequireFromString("100.125"),
		VolumeCbm:            decimal.RequireFromString("2.123456"),
		Status:               biz.SeaSharedContainerStatusDraft,
		Version:              3,
		CreatedAt:            now,
		UpdatedAt:            now,
		Progress: &biz.SeaSharedContainerProgress{
			AllocatedGrossWeightKg: decimal.RequireFromString("50.100"),
			AllocatedVolumeCbm:     decimal.RequireFromString("1.100000"),
			RemainingGrossWeightKg: decimal.RequireFromString("50.025"),
			RemainingVolumeCbm:     decimal.RequireFromString("1.023456"),
		},
	}
	api := seaSharedContainerToAPI(item)
	if api.GetGrossWeightKg() != "100.125" || api.GetVolumeCbm() != "2.123456" {
		t.Fatalf("精确件重尺映射错误: weight=%q volume=%q", api.GetGrossWeightKg(), api.GetVolumeCbm())
	}
	if api.GetStatus() != v1.SeaSharedContainerStatus_SEA_SHARED_CONTAINER_STATUS_DRAFT {
		t.Fatalf("状态映射错误: %v", api.GetStatus())
	}
}

func TestSeaSharedContainerInputRejectsInvalidUUIDAndScale(t *testing.T) {
	valid := &v1.SeaSharedContainerInput{
		TransportExecutionId: uuid.NewString(),
		ContainerNo:          "MSCU1234567",
		ContainerSpecId:      uuid.NewString(),
		PackageCount:         1,
		GrossWeightKg:        "1.001",
		VolumeCbm:            "1.000001",
	}
	if _, err := seaSharedContainerInputFromAPI(valid); err != nil {
		t.Fatalf("合法输入转换失败: %v", err)
	}
	invalidUUID := &v1.SeaSharedContainerInput{
		TransportExecutionId: "invalid",
		ContainerNo:          valid.ContainerNo,
		ContainerSpecId:      valid.ContainerSpecId,
		PackageCount:         valid.PackageCount,
		GrossWeightKg:        valid.GrossWeightKg,
		VolumeCbm:            valid.VolumeCbm,
	}
	if _, err := seaSharedContainerInputFromAPI(invalidUUID); err != biz.ErrSeaSharedContainerInvalidArgument {
		t.Fatalf("非法 UUID error=%v", err)
	}
	invalidScale := &v1.SeaSharedContainerInput{
		TransportExecutionId: valid.TransportExecutionId,
		ContainerNo:          valid.ContainerNo,
		ContainerSpecId:      valid.ContainerSpecId,
		PackageCount:         valid.PackageCount,
		GrossWeightKg:        "1.0001",
		VolumeCbm:            valid.VolumeCbm,
	}
	if _, err := seaSharedContainerInputFromAPI(invalidScale); err != biz.ErrSeaSharedContainerInvalidArgument {
		t.Fatalf("超精度毛重 error=%v", err)
	}
}
