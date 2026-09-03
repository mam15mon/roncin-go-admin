package biz

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SeaCargoAllocationStatus string

const (
	SeaCargoAllocationStatusDraft     SeaCargoAllocationStatus = "DRAFT"
	SeaCargoAllocationStatusConfirmed SeaCargoAllocationStatus = "CONFIRMED"
)

func (s SeaCargoAllocationStatus) Valid() bool {
	return s == SeaCargoAllocationStatusDraft || s == SeaCargoAllocationStatusConfirmed
}

type SeaCargoAllocationAction string

const (
	SeaCargoAllocationActionSaveDraft              SeaCargoAllocationAction = "SAVE_DRAFT"
	SeaCargoAllocationActionConfirm                SeaCargoAllocationAction = "CONFIRM"
	SeaCargoAllocationActionWithdraw               SeaCargoAllocationAction = "WITHDRAW"
	SeaCargoAllocationActionApplyHouseBillSummary  SeaCargoAllocationAction = "APPLY_HOUSE_BILL_SUMMARY"
	SeaCargoAllocationActionApplyMasterBillSummary SeaCargoAllocationAction = "APPLY_MASTER_BILL_SUMMARY"
)

var (
	ErrSeaCargoAllocationInvalidArgument  = errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "箱货分配参数不合法")
	ErrSeaCargoAllocationInvalidReference = errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_REFERENCE", "箱货分配引用实体不合法")
	ErrSeaCargoAllocationExceeded         = errors.BadRequest("SEA_CARGO_ALLOCATION_EXCEEDED", "箱货分配超出允许总量")
	ErrSeaCargoAllocationIncomplete       = errors.BadRequest("SEA_CARGO_ALLOCATION_INCOMPLETE", "箱货分配未完整分配")
	ErrSeaCargoAllocationConflict         = errors.Conflict("SEA_CARGO_ALLOCATION_CONFLICT", "箱货分配聚合已被更新，请刷新后重试")
	ErrSeaCargoAllocationStatusConflict   = errors.Conflict("SEA_CARGO_ALLOCATION_STATUS_CONFLICT", "当前单证结构或分配状态不允许该操作")
	ErrOrderContainerShipmentType         = errors.BadRequest("ORDER_CONTAINER_INVALID_ARGUMENT", "仅整箱(FCL)业务允许维护集装箱")
	ErrOrderCargoItemNotFound             = errors.NotFound("ORDER_CARGO_ITEM_NOT_FOUND", "订单货物明细不存在")
	ErrOrderCargoItemInvalidArgument      = errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", "订单货物明细参数不合法")
	ErrOrderCargoItemConflict             = errors.Conflict("ORDER_STATUS_CONFLICT", "货物明细已被更新，请刷新后重试")
	ErrOrderContainerNotFound             = errors.NotFound("ORDER_CONTAINER_NOT_FOUND", "订单集装箱不存在")
	ErrOrderContainerInvalidArgument      = errors.BadRequest("ORDER_CONTAINER_INVALID_ARGUMENT", "订单集装箱参数不合法")
	ErrOrderContainerExists               = errors.Conflict("ORDER_CONTAINER_EXISTS", "该箱号已存在于当前订单")
	ErrOrderContainerSpecInvalid          = errors.BadRequest("ORDER_CONTAINER_SPEC_INVALID", "集装箱规格不存在或已被禁用")
	ErrOrderContainerConflict             = errors.Conflict("ORDER_STATUS_CONFLICT", "集装箱已被更新，请刷新后重试")
)

func IsSeaCargoAllocationIncomplete(err error) bool {
	if se := errors.FromError(err); se != nil {
		return se.Reason == "SEA_CARGO_ALLOCATION_INCOMPLETE"
	}
	return false
}

func IsSeaCargoAllocationExceeded(err error) bool {
	if se := errors.FromError(err); se != nil {
		return se.Reason == "SEA_CARGO_ALLOCATION_EXCEEDED"
	}
	return false
}

func IsSeaCargoAllocationConflict(err error) bool {
	if se := errors.FromError(err); se != nil {
		return se.Reason == "SEA_CARGO_ALLOCATION_CONFLICT"
	}
	return false
}

func IsSeaCargoAllocationStatusConflict(err error) bool {
	if se := errors.FromError(err); se != nil {
		return se.Reason == "SEA_CARGO_ALLOCATION_STATUS_CONFLICT"
	}
	return false
}

type SeaCargoAllocation struct {
	ID                    uuid.UUID
	OrganizationID        uuid.UUID
	OrderID               uuid.UUID
	MasterBillOrderLinkID uuid.UUID
	CargoItemID           uuid.UUID
	HouseBillID           uuid.UUID
	ContainerID           *uuid.UUID
	PackageCount          int32
	GrossWeightKg         decimal.Decimal
	VolumeCbm             decimal.Decimal
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type SeaCargoAllocationInput struct {
	ID            *uuid.UUID
	CargoItemID   uuid.UUID
	HouseBillID   uuid.UUID
	ContainerID   *uuid.UUID
	PackageCount  int32
	GrossWeightKg decimal.Decimal
	VolumeCbm     decimal.Decimal
}

type SeaCargoAllocationCargoItemSummary struct {
	CargoItemID            uuid.UUID
	CargoName              string
	BaselinePackageCount   int32
	AllocatedPackageCount  int32
	RemainingPackageCount  int32
	BaselineGrossWeightKg  decimal.Decimal
	AllocatedGrossWeightKg decimal.Decimal
	RemainingGrossWeightKg decimal.Decimal
	BaselineVolumeCbm      decimal.Decimal
	AllocatedVolumeCbm     decimal.Decimal
	RemainingVolumeCbm     decimal.Decimal
	Status                 string // "IN_PROGRESS", "COMPLETED", "EXCEEDED"
}

type SeaCargoAllocationContainerSummary struct {
	ContainerID            uuid.UUID
	ContainerNo            string
	BaselinePackageCount   int32
	AllocatedPackageCount  int32
	RemainingPackageCount  int32
	BaselineGrossWeightKg  decimal.Decimal
	AllocatedGrossWeightKg decimal.Decimal
	RemainingGrossWeightKg decimal.Decimal
	BaselineVolumeCbm      decimal.Decimal
	AllocatedVolumeCbm     decimal.Decimal
	RemainingVolumeCbm     decimal.Decimal
	Status                 string // "IN_PROGRESS", "COMPLETED", "EXCEEDED"
}

type SeaCargoAllocationHouseBillSummary struct {
	HouseBillID                 uuid.UUID
	HouseNo                     string
	AllocatedPackageCount       int32
	AllocatedGrossWeightKg      decimal.Decimal
	AllocatedVolumeCbm          decimal.Decimal
	OrderRemainingPackageCount  int32
	OrderRemainingGrossWeightKg decimal.Decimal
	OrderRemainingVolumeCbm     decimal.Decimal
	DisplayPackageCount         *int32
	DisplayGrossWeightKg        *decimal.Decimal
	DisplayVolumeCbm            *decimal.Decimal
	DiffPackageCount            int32
	DiffGrossWeightKg           decimal.Decimal
	DiffVolumeCbm               decimal.Decimal
	DisplayMatches              bool
}

type SeaCargoAllocationProgress struct {
	CargoSummaries              []*SeaCargoAllocationCargoItemSummary
	ContainerSummaries          []*SeaCargoAllocationContainerSummary
	HouseBillSummaries          []*SeaCargoAllocationHouseBillSummary
	OrderRemainingPackageCount  int32
	OrderRemainingGrossWeightKg decimal.Decimal
	OrderRemainingVolumeCbm     decimal.Decimal
}

type SeaCargoAllocationAggregate struct {
	OrderID           uuid.UUID
	DocumentStructure SeaDocumentStructure
	ShipmentType      string
	AllocationStatus  SeaCargoAllocationStatus
	AllocationVersion uint64
	ConfirmedAt       *time.Time
	ConfirmedBy       *uuid.UUID
	ConfirmedByName   string
	CargoItems        []*OrderCargoItem
	Containers        []*OrderContainer
	HouseBills        []*SeaHouseBill
	Allocations       []*SeaCargoAllocation
	Progress          *SeaCargoAllocationProgress
	AllowedActions    []SeaCargoAllocationAction
}

func DimensionLabel(dimension string) string {
	switch dimension {
	case "package_count":
		return "件数"
	case "gross_weight_kg":
		return "毛重"
	case "volume_cbm":
		return "体积"
	default:
		return dimension
	}
}

func FormatAllocationExceededMessage(objectType, objectLabel, dimension, allocatedVal, baselineVal, excessVal string) string {
	dimName := DimensionLabel(dimension)
	switch objectType {
	case "house_bill":
		return fmt.Sprintf("%s 的%s已分配 %s，超过操作票可分配总量 %s，超出 %s，请调整", objectLabel, dimName, allocatedVal, baselineVal, excessVal)
	case "cargo_item":
		return fmt.Sprintf("%s 的%s已分配 %s，超过货物可分配总量 %s，超出 %s，请调整", objectLabel, dimName, allocatedVal, baselineVal, excessVal)
	case "container":
		return fmt.Sprintf("箱号 %s 的%s已分配 %s，超过实际箱可分配总量 %s，超出 %s，请调整", objectLabel, dimName, allocatedVal, baselineVal, excessVal)
	default:
		return fmt.Sprintf("%s 的%s已分配 %s，超过基准 %s，超出 %s，请调整", objectLabel, dimName, allocatedVal, baselineVal, excessVal)
	}
}

func FormatAllocationIncompleteMessage(objectType, objectLabel, dimension, allocatedVal, baselineVal, remainingVal string) string {
	dimName := DimensionLabel(dimension)
	switch objectType {
	case "cargo_item":
		return fmt.Sprintf("%s 的%s尚未分完：总量 %s，已分配 %s，剩余 %s，请调整", objectLabel, dimName, baselineVal, allocatedVal, remainingVal)
	case "container":
		return fmt.Sprintf("箱号 %s 的%s尚未分完：总量 %s，已分配 %s，剩余 %s，请调整", objectLabel, dimName, baselineVal, allocatedVal, remainingVal)
	default:
		return fmt.Sprintf("%s 的%s尚未分完：总量 %s，已分配 %s，剩余 %s，请调整", objectLabel, dimName, baselineVal, allocatedVal, remainingVal)
	}
}

func NewErrAllocationExceeded(
	objectType, objectID, objectLabel, dimension, baselineVal, allocatedVal, excessVal string,
	cargoItemID, houseBillID, containerID *uuid.UUID,
) *errors.Error {
	msg := FormatAllocationExceededMessage(objectType, objectLabel, dimension, allocatedVal, baselineVal, excessVal)
	md := map[string]string{
		"object_type":     objectType,
		"object_id":       objectID,
		"object_label":    objectLabel,
		"dimension":       dimension,
		"baseline_value":  baselineVal,
		"allocated_value": allocatedVal,
		"excess_value":    excessVal,
	}
	if cargoItemID != nil {
		md["cargo_item_id"] = cargoItemID.String()
	}
	if houseBillID != nil {
		md["house_bill_id"] = houseBillID.String()
	}
	if containerID != nil {
		md["container_id"] = containerID.String()
	}
	return errors.BadRequest("SEA_CARGO_ALLOCATION_EXCEEDED", msg).WithMetadata(md)
}

func NewErrAllocationIncomplete(
	objectType, objectID, objectLabel, dimension, baselineVal, allocatedVal, remainingVal string,
	cargoItemID, houseBillID, containerID *uuid.UUID,
	customMsg ...string,
) *errors.Error {
	var msg string
	if len(customMsg) > 0 && customMsg[0] != "" {
		msg = customMsg[0]
	} else {
		msg = FormatAllocationIncompleteMessage(objectType, objectLabel, dimension, allocatedVal, baselineVal, remainingVal)
	}
	md := map[string]string{
		"object_type":     objectType,
		"object_id":       objectID,
		"object_label":    objectLabel,
		"dimension":       dimension,
		"baseline_value":  baselineVal,
		"allocated_value": allocatedVal,
		"remaining_value": remainingVal,
	}
	if cargoItemID != nil {
		md["cargo_item_id"] = cargoItemID.String()
	}
	if houseBillID != nil {
		md["house_bill_id"] = houseBillID.String()
	}
	if containerID != nil {
		md["container_id"] = containerID.String()
	}
	return errors.BadRequest("SEA_CARGO_ALLOCATION_INCOMPLETE", msg).WithMetadata(md)
}

func NewErrAllocationInvalidReference(objectType string, objectID uuid.UUID) *errors.Error {
	labels := map[string]string{
		"cargo_item": "货物明细",
		"house_bill": "分单",
		"container":  "实际箱",
	}
	label := labels[objectType]
	if label == "" {
		label = "引用实体"
	}
	metadata := map[string]string{
		"object_type":  objectType,
		"object_id":    objectID.String(),
		"object_label": label,
	}
	metadata[objectType+"_id"] = objectID.String()
	return errors.BadRequest(
		"SEA_CARGO_ALLOCATION_INVALID_REFERENCE",
		fmt.Sprintf("%s %s 不属于当前操作票或当前主单关系，请刷新后重试", label, objectID),
	).WithMetadata(metadata)
}

func ParseAndValidateWeight(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "毛重不能为空")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "毛重格式不合法")
	}
	if !d.GreaterThan(decimal.Zero) {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "毛重必须为正数")
	}
	if d.Exponent() < -3 {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "毛重最多支持 3 位小数")
	}
	if d.GreaterThanOrEqual(decimal.RequireFromString("1000000000000000")) {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "毛重数值超出上限")
	}
	return d, nil
}

func ParseAndValidateVolume(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "体积不能为空")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "体积格式不合法")
	}
	if !d.GreaterThan(decimal.Zero) {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "体积必须为正数")
	}
	if d.Exponent() < -6 {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "体积最多支持 6 位小数")
	}
	if d.GreaterThanOrEqual(decimal.RequireFromString("1000000000000")) {
		return decimal.Zero, errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "体积数值超出上限")
	}
	return d, nil
}

func ValidateFloatWeight(f float64, fieldName string) (decimal.Decimal, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 {
		return decimal.Zero, errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", fieldName+"必须为正有限数")
	}
	d := decimal.NewFromFloat(f)
	if d.Exponent() < -3 {
		return decimal.Zero, errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", fieldName+"最多支持 3 位小数")
	}
	if d.GreaterThanOrEqual(decimal.RequireFromString("1000000000000000")) {
		return decimal.Zero, errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", fieldName+"数值超出上限")
	}
	return d, nil
}

func ValidateFloatVolume(f float64, fieldName string) (decimal.Decimal, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) || f <= 0 {
		return decimal.Zero, errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", fieldName+"必须为正有限数")
	}
	d := decimal.NewFromFloat(f)
	if d.Exponent() < -6 {
		return decimal.Zero, errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", fieldName+"最多支持 6 位小数")
	}
	if d.GreaterThanOrEqual(decimal.RequireFromString("1000000000000")) {
		return decimal.Zero, errors.BadRequest("ORDER_CARGO_ITEM_INVALID_ARGUMENT", fieldName+"数值超出上限")
	}
	return d, nil
}

func CalculateAllocationProgress(
	cargoItems []*OrderCargoItem,
	containers []*OrderContainer,
	houseBills []*SeaHouseBill,
	allocations []*SeaCargoAllocation,
	shipmentType string,
) *SeaCargoAllocationProgress {
	cargoSummaries := make([]*SeaCargoAllocationCargoItemSummary, 0, len(cargoItems))
	var orderRemPkg int32
	orderRemWeight := decimal.Zero
	orderRemVolume := decimal.Zero

	for _, item := range cargoItems {
		itemBaseWeight := decimal.NewFromFloat(item.GrossWeightKg)
		itemBaseVol := decimal.NewFromFloat(item.VolumeCbm)

		var allocPkg int32
		allocWeight := decimal.Zero
		allocVol := decimal.Zero

		for _, a := range allocations {
			if a.CargoItemID == item.ID {
				allocPkg += a.PackageCount
				allocWeight = allocWeight.Add(a.GrossWeightKg)
				allocVol = allocVol.Add(a.VolumeCbm)
			}
		}

		remPkg := int32(item.PackageCount) - allocPkg
		remWeight := itemBaseWeight.Sub(allocWeight)
		remVol := itemBaseVol.Sub(allocVol)

		status := "IN_PROGRESS"
		if remPkg < 0 || remWeight.IsNegative() || remVol.IsNegative() {
			status = "EXCEEDED"
		} else if remPkg == 0 && remWeight.IsZero() && remVol.IsZero() {
			status = "COMPLETED"
		}

		if remPkg > 0 {
			orderRemPkg += remPkg
		}
		if remWeight.GreaterThan(decimal.Zero) {
			orderRemWeight = orderRemWeight.Add(remWeight)
		}
		if remVol.GreaterThan(decimal.Zero) {
			orderRemVolume = orderRemVolume.Add(remVol)
		}

		cargoSummaries = append(cargoSummaries, &SeaCargoAllocationCargoItemSummary{
			CargoItemID:            item.ID,
			CargoName:              item.CargoName,
			BaselinePackageCount:   int32(item.PackageCount),
			AllocatedPackageCount:  allocPkg,
			RemainingPackageCount:  remPkg,
			BaselineGrossWeightKg:  itemBaseWeight,
			AllocatedGrossWeightKg: allocWeight,
			RemainingGrossWeightKg: remWeight,
			BaselineVolumeCbm:      itemBaseVol,
			AllocatedVolumeCbm:     allocVol,
			RemainingVolumeCbm:     remVol,
			Status:                 status,
		})
	}

	containerSummaries := make([]*SeaCargoAllocationContainerSummary, 0, len(containers))
	for _, c := range containers {
		cBaseWeight := decimal.NewFromFloat(c.GrossWeightKg)
		cBaseVol := decimal.NewFromFloat(c.VolumeCbm)

		var allocPkg int32
		allocWeight := decimal.Zero
		allocVol := decimal.Zero

		for _, a := range allocations {
			if a.ContainerID != nil && *a.ContainerID == c.ID {
				allocPkg += a.PackageCount
				allocWeight = allocWeight.Add(a.GrossWeightKg)
				allocVol = allocVol.Add(a.VolumeCbm)
			}
		}

		remPkg := c.PackageCount - allocPkg
		remWeight := cBaseWeight.Sub(allocWeight)
		remVol := cBaseVol.Sub(allocVol)

		status := "IN_PROGRESS"
		if remPkg < 0 || remWeight.IsNegative() || remVol.IsNegative() {
			status = "EXCEEDED"
		} else if remPkg == 0 && remWeight.IsZero() && remVol.IsZero() {
			status = "COMPLETED"
		}

		containerSummaries = append(containerSummaries, &SeaCargoAllocationContainerSummary{
			ContainerID:            c.ID,
			ContainerNo:            c.ContainerNo,
			BaselinePackageCount:   c.PackageCount,
			AllocatedPackageCount:  allocPkg,
			RemainingPackageCount:  remPkg,
			BaselineGrossWeightKg:  cBaseWeight,
			AllocatedGrossWeightKg: allocWeight,
			RemainingGrossWeightKg: remWeight,
			BaselineVolumeCbm:      cBaseVol,
			AllocatedVolumeCbm:     allocVol,
			RemainingVolumeCbm:     remVol,
			Status:                 status,
		})
	}

	houseBillSummaries := make([]*SeaCargoAllocationHouseBillSummary, 0, len(houseBills))
	for _, hb := range houseBills {
		var allocPkg int32
		allocWeight := decimal.Zero
		allocVol := decimal.Zero

		for _, a := range allocations {
			if a.HouseBillID == hb.ID {
				allocPkg += a.PackageCount
				allocWeight = allocWeight.Add(a.GrossWeightKg)
				allocVol = allocVol.Add(a.VolumeCbm)
			}
		}

		var dispPkg *int32
		var dispWeight *decimal.Decimal
		var dispVol *decimal.Decimal
		var diffPkg int32 = allocPkg
		diffWeight := allocWeight
		diffVol := allocVol
		displayMatches := false

		if hb.Content != nil {
			if hb.Content.PackageCount != nil {
				dispPkg = hb.Content.PackageCount
				diffPkg = allocPkg - *dispPkg
			}
			if hb.Content.GrossWeightKg != nil {
				w := decimal.NewFromFloat(*hb.Content.GrossWeightKg)
				dispWeight = &w
				diffWeight = allocWeight.Sub(w)
			}
			if hb.Content.VolumeCbm != nil {
				v := decimal.NewFromFloat(*hb.Content.VolumeCbm)
				dispVol = &v
				diffVol = allocVol.Sub(v)
			}
			if dispPkg != nil && dispWeight != nil && dispVol != nil {
				if diffPkg == 0 && diffWeight.IsZero() && diffVol.IsZero() {
					displayMatches = true
				}
			}
		}

		houseBillSummaries = append(houseBillSummaries, &SeaCargoAllocationHouseBillSummary{
			HouseBillID:                 hb.ID,
			HouseNo:                     hb.HouseNo,
			AllocatedPackageCount:       allocPkg,
			AllocatedGrossWeightKg:      allocWeight,
			AllocatedVolumeCbm:          allocVol,
			OrderRemainingPackageCount:  orderRemPkg,
			OrderRemainingGrossWeightKg: orderRemWeight,
			OrderRemainingVolumeCbm:     orderRemVolume,
			DisplayPackageCount:         dispPkg,
			DisplayGrossWeightKg:        dispWeight,
			DisplayVolumeCbm:            dispVol,
			DiffPackageCount:            diffPkg,
			DiffGrossWeightKg:           diffWeight,
			DiffVolumeCbm:               diffVol,
			DisplayMatches:              displayMatches,
		})
	}

	return &SeaCargoAllocationProgress{
		CargoSummaries:              cargoSummaries,
		ContainerSummaries:          containerSummaries,
		HouseBillSummaries:          houseBillSummaries,
		OrderRemainingPackageCount:  orderRemPkg,
		OrderRemainingGrossWeightKg: orderRemWeight,
		OrderRemainingVolumeCbm:     orderRemVolume,
	}
}

func ValidateDraftAllocations(
	cargoItems []*OrderCargoItem,
	containers []*OrderContainer,
	houseBills []*SeaHouseBill,
	allocations []*SeaCargoAllocationInput,
	shipmentType string,
) error {
	cargoMap := make(map[uuid.UUID]*OrderCargoItem, len(cargoItems))
	var totalOrderPkg int32
	totalOrderWeight := decimal.Zero
	totalOrderVolume := decimal.Zero
	for _, ci := range cargoItems {
		cargoMap[ci.ID] = ci
		totalOrderPkg += int32(ci.PackageCount)
		totalOrderWeight = totalOrderWeight.Add(decimal.NewFromFloat(ci.GrossWeightKg))
		totalOrderVolume = totalOrderVolume.Add(decimal.NewFromFloat(ci.VolumeCbm))
	}

	hbMap := make(map[uuid.UUID]*SeaHouseBill, len(houseBills))
	for _, hb := range houseBills {
		hbMap[hb.ID] = hb
	}

	cntrMap := make(map[uuid.UUID]*OrderContainer, len(containers))
	for _, c := range containers {
		cntrMap[c.ID] = c
	}

	// 唯一性键集合防重
	seenKeys := make(map[string]bool, len(allocations))

	// 累计计算
	cargoAllocPkg := make(map[uuid.UUID]int32)
	cargoAllocWeight := make(map[uuid.UUID]decimal.Decimal)
	cargoAllocVol := make(map[uuid.UUID]decimal.Decimal)

	cntrAllocPkg := make(map[uuid.UUID]int32)
	cntrAllocWeight := make(map[uuid.UUID]decimal.Decimal)
	cntrAllocVol := make(map[uuid.UUID]decimal.Decimal)

	hbAllocPkg := make(map[uuid.UUID]int32)
	hbAllocWeight := make(map[uuid.UUID]decimal.Decimal)
	hbAllocVol := make(map[uuid.UUID]decimal.Decimal)

	for _, a := range allocations {
		if a.PackageCount <= 0 {
			return errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "分配件数必须为正整数")
		}
		if !a.GrossWeightKg.GreaterThan(decimal.Zero) || a.GrossWeightKg.Exponent() < -3 {
			return errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "分配毛重必须为正数且最多 3 位小数")
		}
		if !a.VolumeCbm.GreaterThan(decimal.Zero) || a.VolumeCbm.Exponent() < -6 {
			return errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "分配体积必须为正数且最多 6 位小数")
		}

		ci, ok := cargoMap[a.CargoItemID]
		if !ok {
			return NewErrAllocationInvalidReference("cargo_item", a.CargoItemID)
		}
		hb, ok := hbMap[a.HouseBillID]
		if !ok {
			return NewErrAllocationInvalidReference("house_bill", a.HouseBillID)
		}

		var cntr *OrderContainer
		if a.ContainerID != nil {
			if shipmentType != "FCL" {
				return errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "非整箱业务禁止关联集装箱")
			}
			cntr, ok = cntrMap[*a.ContainerID]
			if !ok {
				return NewErrAllocationInvalidReference("container", *a.ContainerID)
			}
		}

		var dedupKey string
		if a.ContainerID == nil {
			dedupKey = fmt.Sprintf("no_cntr:%s:%s", a.CargoItemID, a.HouseBillID)
		} else {
			dedupKey = fmt.Sprintf("with_cntr:%s:%s:%s", a.CargoItemID, a.HouseBillID, *a.ContainerID)
		}
		if seenKeys[dedupKey] {
			return errors.BadRequest("SEA_CARGO_ALLOCATION_INVALID_ARGUMENT", "存在重复的分配明细行")
		}
		seenKeys[dedupKey] = true

		cargoAllocPkg[a.CargoItemID] += a.PackageCount
		cargoAllocWeight[a.CargoItemID] = cargoAllocWeight[a.CargoItemID].Add(a.GrossWeightKg)
		cargoAllocVol[a.CargoItemID] = cargoAllocVol[a.CargoItemID].Add(a.VolumeCbm)

		if a.ContainerID != nil {
			cntrAllocPkg[*a.ContainerID] += a.PackageCount
			cntrAllocWeight[*a.ContainerID] = cntrAllocWeight[*a.ContainerID].Add(a.GrossWeightKg)
			cntrAllocVol[*a.ContainerID] = cntrAllocVol[*a.ContainerID].Add(a.VolumeCbm)
		}

		hbAllocPkg[a.HouseBillID] += a.PackageCount
		hbAllocWeight[a.HouseBillID] = hbAllocWeight[a.HouseBillID].Add(a.GrossWeightKg)
		hbAllocVol[a.HouseBillID] = hbAllocVol[a.HouseBillID].Add(a.VolumeCbm)

		_ = ci
		_ = hb
		_ = cntr
	}

	// 1. 先检查单张 HBL 是否超出操作票总量，确保用户直接看到具体分单错误。
	for _, hb := range houseBills {
		allocPkg := hbAllocPkg[hb.ID]
		allocWeight := hbAllocWeight[hb.ID]
		allocVol := hbAllocVol[hb.ID]

		if allocPkg > totalOrderPkg {
			excess := allocPkg - totalOrderPkg
			return NewErrAllocationExceeded(
				"house_bill", hb.ID.String(), hb.HouseNo, "package_count",
				fmt.Sprintf("%d", totalOrderPkg), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", excess),
				nil, &hb.ID, nil,
			)
		}
		if allocWeight.GreaterThan(totalOrderWeight) {
			excess := allocWeight.Sub(totalOrderWeight)
			return NewErrAllocationExceeded(
				"house_bill", hb.ID.String(), hb.HouseNo, "gross_weight_kg",
				totalOrderWeight.StringFixed(3), allocWeight.StringFixed(3), excess.StringFixed(3),
				nil, &hb.ID, nil,
			)
		}
		if allocVol.GreaterThan(totalOrderVolume) {
			excess := allocVol.Sub(totalOrderVolume)
			return NewErrAllocationExceeded(
				"house_bill", hb.ID.String(), hb.HouseNo, "volume_cbm",
				totalOrderVolume.StringFixed(6), allocVol.StringFixed(6), excess.StringFixed(6),
				nil, &hb.ID, nil,
			)
		}
	}

	// 2. 检查货物行超分
	for _, ci := range cargoItems {
		baseWeight := decimal.NewFromFloat(ci.GrossWeightKg)
		baseVol := decimal.NewFromFloat(ci.VolumeCbm)

		allocPkg := cargoAllocPkg[ci.ID]
		allocWeight := cargoAllocWeight[ci.ID]
		allocVol := cargoAllocVol[ci.ID]

		if allocPkg > int32(ci.PackageCount) {
			excess := allocPkg - int32(ci.PackageCount)
			return NewErrAllocationExceeded(
				"cargo_item", ci.ID.String(), ci.CargoName, "package_count",
				fmt.Sprintf("%d", ci.PackageCount), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", excess),
				&ci.ID, nil, nil,
			)
		}
		if allocWeight.GreaterThan(baseWeight) {
			excess := allocWeight.Sub(baseWeight)
			return NewErrAllocationExceeded(
				"cargo_item", ci.ID.String(), ci.CargoName, "gross_weight_kg",
				baseWeight.StringFixed(3), allocWeight.StringFixed(3), excess.StringFixed(3),
				&ci.ID, nil, nil,
			)
		}
		if allocVol.GreaterThan(baseVol) {
			excess := allocVol.Sub(baseVol)
			return NewErrAllocationExceeded(
				"cargo_item", ci.ID.String(), ci.CargoName, "volume_cbm",
				baseVol.StringFixed(6), allocVol.StringFixed(6), excess.StringFixed(6),
				&ci.ID, nil, nil,
			)
		}
	}

	// 3. 检查实际箱超分
	for _, c := range containers {
		baseWeight := decimal.NewFromFloat(c.GrossWeightKg)
		baseVol := decimal.NewFromFloat(c.VolumeCbm)

		allocPkg := cntrAllocPkg[c.ID]
		allocWeight := cntrAllocWeight[c.ID]
		allocVol := cntrAllocVol[c.ID]

		if allocPkg > c.PackageCount {
			excess := allocPkg - c.PackageCount
			return NewErrAllocationExceeded(
				"container", c.ID.String(), c.ContainerNo, "package_count",
				fmt.Sprintf("%d", c.PackageCount), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", excess),
				nil, nil, &c.ID,
			)
		}
		if allocWeight.GreaterThan(baseWeight) {
			excess := allocWeight.Sub(baseWeight)
			return NewErrAllocationExceeded(
				"container", c.ID.String(), c.ContainerNo, "gross_weight_kg",
				baseWeight.StringFixed(3), allocWeight.StringFixed(3), excess.StringFixed(3),
				nil, nil, &c.ID,
			)
		}
		if allocVol.GreaterThan(baseVol) {
			excess := allocVol.Sub(baseVol)
			return NewErrAllocationExceeded(
				"container", c.ID.String(), c.ContainerNo, "volume_cbm",
				baseVol.StringFixed(6), allocVol.StringFixed(6), excess.StringFixed(6),
				nil, nil, &c.ID,
			)
		}
	}

	return nil
}

func ValidateConfirmedAllocations(
	cargoItems []*OrderCargoItem,
	containers []*OrderContainer,
	houseBills []*SeaHouseBill,
	allocations []*SeaCargoAllocationInput,
	shipmentType string,
) error {
	// 先做完整的草稿级超分和引用合法性校验
	if err := ValidateDraftAllocations(cargoItems, containers, houseBills, allocations, shipmentType); err != nil {
		return err
	}

	if len(cargoItems) == 0 {
		return errors.BadRequest("SEA_CARGO_ALLOCATION_INCOMPLETE", "操作票无货物明细，无法确认分配")
	}
	if len(houseBills) == 0 {
		return errors.BadRequest("SEA_CARGO_ALLOCATION_INCOMPLETE", "当前操作票无有效分单，无法确认分配")
	}

	// 1. 每条订单货物在件数、毛重、体积三个维度完整分配
	cargoAllocPkg := make(map[uuid.UUID]int32)
	cargoAllocWeight := make(map[uuid.UUID]decimal.Decimal)
	cargoAllocVol := make(map[uuid.UUID]decimal.Decimal)
	hbAllocCount := make(map[uuid.UUID]int)
	cntrAllocPkg := make(map[uuid.UUID]int32)
	cntrAllocWeight := make(map[uuid.UUID]decimal.Decimal)
	cntrAllocVol := make(map[uuid.UUID]decimal.Decimal)
	cntrAllocCount := make(map[uuid.UUID]int)

	for _, a := range allocations {
		cargoAllocPkg[a.CargoItemID] += a.PackageCount
		cargoAllocWeight[a.CargoItemID] = cargoAllocWeight[a.CargoItemID].Add(a.GrossWeightKg)
		cargoAllocVol[a.CargoItemID] = cargoAllocVol[a.CargoItemID].Add(a.VolumeCbm)
		hbAllocCount[a.HouseBillID]++

		if a.ContainerID != nil {
			cntrAllocPkg[*a.ContainerID] += a.PackageCount
			cntrAllocWeight[*a.ContainerID] = cntrAllocWeight[*a.ContainerID].Add(a.GrossWeightKg)
			cntrAllocVol[*a.ContainerID] = cntrAllocVol[*a.ContainerID].Add(a.VolumeCbm)
			cntrAllocCount[*a.ContainerID]++
		}
	}

	for _, ci := range cargoItems {
		baseWeight := decimal.NewFromFloat(ci.GrossWeightKg)
		baseVol := decimal.NewFromFloat(ci.VolumeCbm)

		allocPkg := cargoAllocPkg[ci.ID]
		allocWeight := cargoAllocWeight[ci.ID]
		allocVol := cargoAllocVol[ci.ID]

		if allocPkg < int32(ci.PackageCount) {
			rem := int32(ci.PackageCount) - allocPkg
			return NewErrAllocationIncomplete(
				"cargo_item", ci.ID.String(), ci.CargoName, "package_count",
				fmt.Sprintf("%d", ci.PackageCount), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", rem),
				&ci.ID, nil, nil,
			)
		}
		if allocWeight.LessThan(baseWeight) {
			rem := baseWeight.Sub(allocWeight)
			return NewErrAllocationIncomplete(
				"cargo_item", ci.ID.String(), ci.CargoName, "gross_weight_kg",
				baseWeight.StringFixed(3), allocWeight.StringFixed(3), rem.StringFixed(3),
				&ci.ID, nil, nil,
			)
		}
		if allocVol.LessThan(baseVol) {
			rem := baseVol.Sub(allocVol)
			return NewErrAllocationIncomplete(
				"cargo_item", ci.ID.String(), ci.CargoName, "volume_cbm",
				baseVol.StringFixed(6), allocVol.StringFixed(6), rem.StringFixed(6),
				&ci.ID, nil, nil,
			)
		}
	}

	// 2. 每张当前 HBL 至少取得一条真实分配
	for _, hb := range houseBills {
		if hbAllocCount[hb.ID] == 0 {
			msg := fmt.Sprintf("分单 %s 尚未分配任何货物，请调整", hb.HouseNo)
			return NewErrAllocationIncomplete(
				"house_bill", hb.ID.String(), hb.HouseNo, "package_count",
				"1", "0", "1",
				nil, &hb.ID, nil, msg,
			)
		}
	}

	// 3. FCL 业务必须真实落箱且每只箱完整守恒
	if shipmentType == "FCL" {
		if len(containers) == 0 {
			return errors.BadRequest("SEA_CARGO_ALLOCATION_INCOMPLETE", "FCL 业务尚未录入实际箱，无法确认分配").WithMetadata(map[string]string{
				"object_type": "shipment", "object_label": "当前 FCL 操作票", "dimension": "container", "baseline_value": "至少 1", "allocated_value": "0", "remaining_value": "至少 1",
			})
		}
		for _, a := range allocations {
			if a.ContainerID == nil {
				return errors.BadRequest("SEA_CARGO_ALLOCATION_INCOMPLETE", fmt.Sprintf("货物 %s 在分单 %s 下尚未选择实际箱，请调整", a.CargoItemID, a.HouseBillID)).WithMetadata(map[string]string{
					"object_type": "allocation", "object_label": "未落箱分配行", "dimension": "container_id", "cargo_item_id": a.CargoItemID.String(), "house_bill_id": a.HouseBillID.String(),
				})
			}
		}
		for _, c := range containers {
			if cntrAllocCount[c.ID] == 0 {
				msg := fmt.Sprintf("实际箱 %s 尚未分配任何货物，请调整", c.ContainerNo)
				return NewErrAllocationIncomplete(
					"container", c.ID.String(), c.ContainerNo, "package_count",
					fmt.Sprintf("%d", c.PackageCount), "0", fmt.Sprintf("%d", c.PackageCount),
					nil, nil, &c.ID, msg,
				)
			}
			baseWeight := decimal.NewFromFloat(c.GrossWeightKg)
			baseVol := decimal.NewFromFloat(c.VolumeCbm)

			allocPkg := cntrAllocPkg[c.ID]
			allocWeight := cntrAllocWeight[c.ID]
			allocVol := cntrAllocVol[c.ID]

			if allocPkg < c.PackageCount {
				rem := c.PackageCount - allocPkg
				return NewErrAllocationIncomplete(
					"container", c.ID.String(), c.ContainerNo, "package_count",
					fmt.Sprintf("%d", c.PackageCount), fmt.Sprintf("%d", allocPkg), fmt.Sprintf("%d", rem),
					nil, nil, &c.ID,
				)
			}
			if allocWeight.LessThan(baseWeight) {
				rem := baseWeight.Sub(allocWeight)
				return NewErrAllocationIncomplete(
					"container", c.ID.String(), c.ContainerNo, "gross_weight_kg",
					baseWeight.StringFixed(3), allocWeight.StringFixed(3), rem.StringFixed(3),
					nil, nil, &c.ID,
				)
			}
			if allocVol.LessThan(baseVol) {
				rem := baseVol.Sub(allocVol)
				return NewErrAllocationIncomplete(
					"container", c.ID.String(), c.ContainerNo, "volume_cbm",
					baseVol.StringFixed(6), allocVol.StringFixed(6), rem.StringFixed(6),
					nil, nil, &c.ID,
				)
			}
		}
	}

	return nil
}

type SeaCargoAllocationRepo interface {
	GetSeaCargoAllocation(ctx context.Context, orgID, orderID uuid.UUID) (*SeaCargoAllocationAggregate, error)
	SaveDraft(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedAllocationVersion uint64, allocations []*SeaCargoAllocationInput, audit *AuditEvent) (*SeaCargoAllocationAggregate, error)
	Confirm(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedAllocationVersion uint64, audit *AuditEvent) (*SeaCargoAllocationAggregate, error)
	Withdraw(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedAllocationVersion uint64, audit *AuditEvent) (*SeaCargoAllocationAggregate, error)
	ApplyHouseBillSummary(ctx context.Context, orgID, actorID, orderID, houseBillID uuid.UUID, expectedAllocationVersion, expectedHouseBillVersion uint64, audit *AuditEvent) (*SeaHouseBill, error)
	ApplyMasterBillSummary(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedMblVersion uint64, audit *AuditEvent) (*SeaMasterBillDetail, error)
}

type SeaCargoAllocationUsecase struct {
	repo SeaCargoAllocationRepo
}

func NewSeaCargoAllocationUsecase(repo SeaCargoAllocationRepo) *SeaCargoAllocationUsecase {
	return &SeaCargoAllocationUsecase{repo: repo}
}

func (uc *SeaCargoAllocationUsecase) GetSeaCargoAllocation(ctx context.Context, orgID, orderID uuid.UUID) (*SeaCargoAllocationAggregate, error) {
	if orgID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaCargoAllocationInvalidArgument
	}
	return uc.repo.GetSeaCargoAllocation(ctx, orgID, orderID)
}

func (uc *SeaCargoAllocationUsecase) SaveDraft(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedAllocationVersion uint64, allocations []*SeaCargoAllocationInput, audit *AuditEvent) (*SeaCargoAllocationAggregate, error) {
	if orgID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || expectedAllocationVersion == 0 {
		return nil, ErrSeaCargoAllocationInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.SaveDraft(ctx, orgID, actorID, orderID, expectedAllocationVersion, allocations, audit)
}

func (uc *SeaCargoAllocationUsecase) Confirm(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedAllocationVersion uint64, audit *AuditEvent) (*SeaCargoAllocationAggregate, error) {
	if orgID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || expectedAllocationVersion == 0 {
		return nil, ErrSeaCargoAllocationInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.Confirm(ctx, orgID, actorID, orderID, expectedAllocationVersion, audit)
}

func (uc *SeaCargoAllocationUsecase) Withdraw(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedAllocationVersion uint64, audit *AuditEvent) (*SeaCargoAllocationAggregate, error) {
	if orgID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || expectedAllocationVersion == 0 {
		return nil, ErrSeaCargoAllocationInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.Withdraw(ctx, orgID, actorID, orderID, expectedAllocationVersion, audit)
}

func (uc *SeaCargoAllocationUsecase) ApplyHouseBillSummary(ctx context.Context, orgID, actorID, orderID, houseBillID uuid.UUID, expectedAllocationVersion, expectedHouseBillVersion uint64, audit *AuditEvent) (*SeaHouseBill, error) {
	if orgID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || houseBillID == uuid.Nil || expectedAllocationVersion == 0 || expectedHouseBillVersion == 0 {
		return nil, ErrSeaCargoAllocationInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.ApplyHouseBillSummary(ctx, orgID, actorID, orderID, houseBillID, expectedAllocationVersion, expectedHouseBillVersion, audit)
}

func (uc *SeaCargoAllocationUsecase) ApplyMasterBillSummary(ctx context.Context, orgID, actorID, orderID uuid.UUID, expectedMblVersion uint64, audit *AuditEvent) (*SeaMasterBillDetail, error) {
	if orgID == uuid.Nil || actorID == uuid.Nil || orderID == uuid.Nil || expectedMblVersion == 0 {
		return nil, ErrSeaCargoAllocationInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.ApplyMasterBillSummary(ctx, orgID, actorID, orderID, expectedMblVersion, audit)
}
