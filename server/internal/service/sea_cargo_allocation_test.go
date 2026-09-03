package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSeaCargoAllocationServiceMappings(t *testing.T) {
	orderID := uuid.New()
	cargoItemID := uuid.New()
	containerID := uuid.New()
	houseBillID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	weight := decimal.RequireFromString("1234.567")
	vol := decimal.RequireFromString("12.345678")

	agg := &biz.SeaCargoAllocationAggregate{
		OrderID:           orderID,
		DocumentStructure: biz.SeaDocumentStructureHouse,
		ShipmentType:      "FCL",
		AllocationStatus:  biz.SeaCargoAllocationStatusConfirmed,
		AllocationVersion: 5,
		ConfirmedAt:       &now,
		ConfirmedBy:       &userID,
		ConfirmedByName:   "张操作",
		CargoItems: []*biz.OrderCargoItem{
			{
				ID:            cargoItemID,
				OrderID:       orderID,
				CargoName:     "汽车配件",
				PackageCount:  100,
				GrossWeightKg: 1234.567,
				VolumeCbm:     12.345678,
				Version:       2,
			},
		},
		Containers: []*biz.OrderContainer{
			{
				ID:              containerID,
				OrderID:         orderID,
				ContainerNo:     "MSCU1234567",
				ContainerSpecID: uuid.New(),
				PackageCount:    100,
				GrossWeightKg:   1234.567,
				VolumeCbm:       12.345678,
				Version:         1,
			},
		},
		HouseBills: []*biz.SeaHouseBill{
			{
				ID:                houseBillID,
				OrderID:           orderID,
				HouseNo:           "HBL999",
				NormalizedHouseNo: "HBL999",
				Version:           1,
			},
		},
		Allocations: []*biz.SeaCargoAllocation{
			{
				ID:            uuid.New(),
				CargoItemID:   cargoItemID,
				HouseBillID:   houseBillID,
				ContainerID:   &containerID,
				PackageCount:  100,
				GrossWeightKg: weight,
				VolumeCbm:     vol,
			},
		},
		Progress: &biz.SeaCargoAllocationProgress{
			CargoSummaries: []*biz.SeaCargoAllocationCargoItemSummary{
				{
					CargoItemID:            cargoItemID,
					CargoName:              "汽车配件",
					BaselinePackageCount:   100,
					AllocatedPackageCount:  100,
					RemainingPackageCount:  0,
					BaselineGrossWeightKg:  weight,
					AllocatedGrossWeightKg: weight,
					RemainingGrossWeightKg: decimal.Zero,
					BaselineVolumeCbm:      vol,
					AllocatedVolumeCbm:     vol,
					RemainingVolumeCbm:     decimal.Zero,
					Status:                 "COMPLETED",
				},
			},
			OrderRemainingPackageCount:  0,
			OrderRemainingGrossWeightKg: decimal.Zero,
			OrderRemainingVolumeCbm:     decimal.Zero,
		},
		AllowedActions: []biz.SeaCargoAllocationAction{
			biz.SeaCargoAllocationActionWithdraw,
			biz.SeaCargoAllocationActionApplyHouseBillSummary,
		},
	}

	api := seaCargoAllocationAggregateToAPI(agg)
	if api == nil {
		t.Fatal("expected non-nil api aggregate")
	}
	if api.OrderId != orderID.String() {
		t.Fatalf("order_id mismatch: %s", api.OrderId)
	}
	if api.DocumentStructure != v1.SeaDocumentStructure_SEA_DOCUMENT_STRUCTURE_HOUSE {
		t.Fatalf("document structure mismatch: %v", api.DocumentStructure)
	}
	if api.AllocationStatus != v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_CONFIRMED {
		t.Fatalf("allocation status mismatch: %v", api.AllocationStatus)
	}
	if api.AllocationVersion != 5 {
		t.Fatalf("allocation version mismatch: %d", api.AllocationVersion)
	}
	if api.ConfirmedByName == nil || *api.ConfirmedByName != "张操作" {
		t.Fatalf("confirmed by name mismatch: %v", api.ConfirmedByName)
	}
	if len(api.CargoItems) != 1 || api.CargoItems[0].CargoName != "汽车配件" {
		t.Fatalf("cargo items mismatch: %#v", api.CargoItems)
	}
	if len(api.Containers) != 1 || api.Containers[0].ContainerNo != "MSCU1234567" {
		t.Fatalf("containers mismatch: %#v", api.Containers)
	}
	if len(api.Allocations) != 1 || api.Allocations[0].GrossWeightKg != "1234.567" || api.Allocations[0].VolumeCbm != "12.345678" {
		t.Fatalf("allocations mismatch: %#v", api.Allocations)
	}
	if len(api.AllowedActions) != 2 || api.AllowedActions[0] != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_WITHDRAW {
		t.Fatalf("allowed actions mismatch: %#v", api.AllowedActions)
	}
}

func TestSeaCargoAllocationEnums(t *testing.T) {
	if got := seaCargoAllocationStatusToAPI(biz.SeaCargoAllocationStatusDraft); got != v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_DRAFT {
		t.Fatalf("unexpected draft status: %v", got)
	}
	if got := seaCargoAllocationStatusToAPI(biz.SeaCargoAllocationStatusConfirmed); got != v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_CONFIRMED {
		t.Fatalf("unexpected confirmed status: %v", got)
	}
	if got := seaCargoAllocationStatusToAPI("UNKNOWN"); got != v1.SeaCargoAllocationStatus_SEA_CARGO_ALLOCATION_STATUS_UNSPECIFIED {
		t.Fatalf("unexpected unknown status: %v", got)
	}

	if got := seaCargoAllocationActionToAPI(biz.SeaCargoAllocationActionSaveDraft); got != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_SAVE_DRAFT {
		t.Fatalf("unexpected save draft action: %v", got)
	}
	if got := seaCargoAllocationActionToAPI(biz.SeaCargoAllocationActionConfirm); got != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_CONFIRM {
		t.Fatalf("unexpected confirm action: %v", got)
	}
	if got := seaCargoAllocationActionToAPI(biz.SeaCargoAllocationActionWithdraw); got != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_WITHDRAW {
		t.Fatalf("unexpected withdraw action: %v", got)
	}
	if got := seaCargoAllocationActionToAPI(biz.SeaCargoAllocationActionApplyHouseBillSummary); got != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_APPLY_HOUSE_BILL_SUMMARY {
		t.Fatalf("unexpected apply house bill action: %v", got)
	}
	if got := seaCargoAllocationActionToAPI(biz.SeaCargoAllocationActionApplyMasterBillSummary); got != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_APPLY_MASTER_BILL_SUMMARY {
		t.Fatalf("unexpected apply master bill action: %v", got)
	}
	if got := seaCargoAllocationActionToAPI("OTHER"); got != v1.SeaCargoAllocationAction_SEA_CARGO_ALLOCATION_ACTION_UNSPECIFIED {
		t.Fatalf("unexpected other action: %v", got)
	}
}
