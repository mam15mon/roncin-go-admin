package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SeaSharedContainerStatus string

const (
	SeaSharedContainerStatusDraft     SeaSharedContainerStatus = "DRAFT"
	SeaSharedContainerStatusConfirmed SeaSharedContainerStatus = "CONFIRMED"
)

var (
	ErrSeaSharedContainerInvalidArgument  = errors.BadRequest("SEA_SHARED_CONTAINER_INVALID_ARGUMENT", "共享箱参数不合法")
	ErrSeaSharedContainerNotFound         = errors.NotFound("SEA_SHARED_CONTAINER_NOT_FOUND", "共享箱不存在")
	ErrSeaSharedContainerExists           = errors.Conflict("SEA_SHARED_CONTAINER_EXISTS", "该运输执行下已存在相同箱号")
	ErrSeaSharedContainerConflict         = errors.Conflict("SEA_SHARED_CONTAINER_CONFLICT", "共享箱已被更新，请刷新后重试")
	ErrSeaSharedContainerStatusConflict   = errors.Conflict("SEA_SHARED_CONTAINER_STATUS_CONFLICT", "共享箱当前状态不允许该操作")
	ErrSeaSharedContainerInvalidReference = errors.BadRequest("SEA_SHARED_CONTAINER_INVALID_REFERENCE", "共享箱引用的订单、分单或货物不合法")
	ErrSeaSharedContainerExceeded         = errors.BadRequest("SEA_SHARED_CONTAINER_EXCEEDED", "共享箱或来源货物的分配数量已超出")
	ErrSeaSharedContainerIncomplete       = errors.BadRequest("SEA_SHARED_CONTAINER_INCOMPLETE", "共享箱或来源货物尚未严格分配完整")
)

type SeaSharedContainer struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	TransportExecutionID uuid.UUID
	ContainerNo          string
	ContainerSpecID      uuid.UUID
	ContainerSpecName    string
	SealNo               *string
	PackageCount         int32
	GrossWeightKg        decimal.Decimal
	VolumeCbm            decimal.Decimal
	Status               SeaSharedContainerStatus
	ConfirmedAt          *time.Time
	ConfirmedBy          *uuid.UUID
	ConfirmedByName      string
	Note                 *string
	Version              uint64
	Allocations          []*SeaSharedContainerAllocation
	Progress             *SeaSharedContainerProgress
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type SeaSharedContainerAllocation struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	SharedContainerID uuid.UUID
	OrderID           uuid.UUID
	OrderNo           string
	HouseBillID       uuid.UUID
	HouseNo           string
	CargoItemID       uuid.UUID
	CargoName         string
	PackageCount      int32
	GrossWeightKg     decimal.Decimal
	VolumeCbm         decimal.Decimal
	Version           uint64
	OrderVersion      uint64
	LinkVersion       uint64
	HouseBillVersion  uint64
	CargoItemVersion  uint64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SeaSharedContainerProgress struct {
	AllocatedPackageCount  int32
	AllocatedGrossWeightKg decimal.Decimal
	AllocatedVolumeCbm     decimal.Decimal
	RemainingPackageCount  int32
	RemainingGrossWeightKg decimal.Decimal
	RemainingVolumeCbm     decimal.Decimal
	ContainerBalanced      bool
	CargoBalanced          bool
}

type SeaSharedContainerCandidateOrder struct {
	OrderID          uuid.UUID
	OrderNo          string
	HouseBillID      uuid.UUID
	HouseNo          string
	OrderVersion     uint64
	LinkVersion      uint64
	HouseBillVersion uint64
	CargoItems       []*SeaSharedContainerCandidateCargoItem
}

type SeaSharedContainerCandidateCargoItem struct {
	ID            uuid.UUID
	CargoName     string
	PackageCount  int32
	GrossWeightKg decimal.Decimal
	VolumeCbm     decimal.Decimal
	Version       uint64
}

type SeaSharedContainerAllocationInput struct {
	OrderID                  uuid.UUID
	HouseBillID              uuid.UUID
	CargoItemID              uuid.UUID
	PackageCount             int32
	GrossWeightKg            decimal.Decimal
	VolumeCbm                decimal.Decimal
	ExpectedOrderVersion     uint64
	ExpectedLinkVersion      uint64
	ExpectedHouseBillVersion uint64
	ExpectedCargoItemVersion uint64
}

type SeaSharedQuantity struct {
	PackageCount  int32
	GrossWeightKg decimal.Decimal
	VolumeCbm     decimal.Decimal
}

func (q SeaSharedQuantity) Add(other SeaSharedQuantity) SeaSharedQuantity {
	return SeaSharedQuantity{
		PackageCount:  q.PackageCount + other.PackageCount,
		GrossWeightKg: q.GrossWeightKg.Add(other.GrossWeightKg),
		VolumeCbm:     q.VolumeCbm.Add(other.VolumeCbm),
	}
}

func (q SeaSharedQuantity) Equal(other SeaSharedQuantity) bool {
	return q.PackageCount == other.PackageCount && q.GrossWeightKg.Equal(other.GrossWeightKg) && q.VolumeCbm.Equal(other.VolumeCbm)
}

func (q SeaSharedQuantity) Exceeds(other SeaSharedQuantity) bool {
	return q.PackageCount > other.PackageCount || q.GrossWeightKg.GreaterThan(other.GrossWeightKg) || q.VolumeCbm.GreaterThan(other.VolumeCbm)
}

func ParseSharedWeight(value string) (decimal.Decimal, error) {
	return parsePositiveScaledDecimal(value, 3, "999999999999999.999")
}

func ParseSharedVolume(value string) (decimal.Decimal, error) {
	return parsePositiveScaledDecimal(value, 6, "999999999999.999999")
}

func parsePositiveScaledDecimal(value string, scale int32, maximum string) (decimal.Decimal, error) {
	if value == "" || value != strings.TrimSpace(value) {
		return decimal.Zero, ErrSeaSharedContainerInvalidArgument
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil || !parsed.IsPositive() || -parsed.Exponent() > scale {
		return decimal.Zero, ErrSeaSharedContainerInvalidArgument
	}
	max, _ := decimal.NewFromString(maximum)
	if parsed.GreaterThan(max) {
		return decimal.Zero, ErrSeaSharedContainerInvalidArgument
	}
	return parsed, nil
}

func NormalizeSeaSharedContainer(input *SeaSharedContainer) (*SeaSharedContainer, error) {
	if input == nil || input.TransportExecutionID == uuid.Nil || input.ContainerSpecID == uuid.Nil || input.PackageCount <= 0 || !input.GrossWeightKg.IsPositive() || !input.VolumeCbm.IsPositive() {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	containerNo := strings.TrimSpace(input.ContainerNo)
	if containerNo == "" || utf8.RuneCountInString(containerNo) > 64 || -input.GrossWeightKg.Exponent() > 3 || -input.VolumeCbm.Exponent() > 6 {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	result := *input
	result.ContainerNo = containerNo
	result.SealNo = normalizeOptionalSharedText(input.SealNo, 64)
	result.Note = normalizeOptionalSharedText(input.Note, 500)
	if input.SealNo != nil && result.SealNo == nil && strings.TrimSpace(*input.SealNo) != "" {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	if input.Note != nil && result.Note == nil && strings.TrimSpace(*input.Note) != "" {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	return &result, nil
}

func normalizeOptionalSharedText(value *string, max int) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" || utf8.RuneCountInString(normalized) > max {
		return nil
	}
	return &normalized
}

func NormalizeSeaSharedAllocations(inputs []*SeaSharedContainerAllocationInput) ([]*SeaSharedContainerAllocationInput, error) {
	result := make([]*SeaSharedContainerAllocationInput, 0, len(inputs))
	seenCargo := make(map[uuid.UUID]struct{}, len(inputs))
	for _, input := range inputs {
		if input == nil || input.OrderID == uuid.Nil || input.HouseBillID == uuid.Nil || input.CargoItemID == uuid.Nil || input.PackageCount <= 0 || !input.GrossWeightKg.IsPositive() || !input.VolumeCbm.IsPositive() || -input.GrossWeightKg.Exponent() > 3 || -input.VolumeCbm.Exponent() > 6 || input.ExpectedOrderVersion == 0 || input.ExpectedLinkVersion == 0 || input.ExpectedHouseBillVersion == 0 || input.ExpectedCargoItemVersion == 0 {
			return nil, ErrSeaSharedContainerInvalidArgument
		}
		if _, exists := seenCargo[input.CargoItemID]; exists {
			return nil, ErrSeaSharedContainerInvalidArgument
		}
		seenCargo[input.CargoItemID] = struct{}{}
		copy := *input
		result = append(result, &copy)
	}
	return result, nil
}

func ValidateSeaSharedConservation(container SeaSharedQuantity, current []SeaSharedContainerAllocationInput, cargoBaselines, cargoAllocated map[uuid.UUID]SeaSharedQuantity, strict bool) error {
	allocated := SeaSharedQuantity{}
	for _, input := range current {
		quantity := SeaSharedQuantity{PackageCount: input.PackageCount, GrossWeightKg: input.GrossWeightKg, VolumeCbm: input.VolumeCbm}
		allocated = allocated.Add(quantity)
		baseline, ok := cargoBaselines[input.CargoItemID]
		if !ok || cargoAllocated[input.CargoItemID].Exceeds(baseline) {
			return ErrSeaSharedContainerExceeded
		}
	}
	if allocated.Exceeds(container) {
		return ErrSeaSharedContainerExceeded
	}
	if !strict {
		return nil
	}
	if !allocated.Equal(container) {
		return ErrSeaSharedContainerIncomplete
	}
	for cargoID, baseline := range cargoBaselines {
		if !cargoAllocated[cargoID].Equal(baseline) {
			return ErrSeaSharedContainerIncomplete
		}
	}
	return nil
}

func CalculateSeaSharedContainerProgress(container SeaSharedQuantity, allocations []*SeaSharedContainerAllocation, cargoBalanced bool) *SeaSharedContainerProgress {
	allocated := SeaSharedQuantity{}
	for _, allocation := range allocations {
		allocated = allocated.Add(SeaSharedQuantity{PackageCount: allocation.PackageCount, GrossWeightKg: allocation.GrossWeightKg, VolumeCbm: allocation.VolumeCbm})
	}
	return &SeaSharedContainerProgress{
		AllocatedPackageCount:  allocated.PackageCount,
		AllocatedGrossWeightKg: allocated.GrossWeightKg,
		AllocatedVolumeCbm:     allocated.VolumeCbm,
		RemainingPackageCount:  container.PackageCount - allocated.PackageCount,
		RemainingGrossWeightKg: container.GrossWeightKg.Sub(allocated.GrossWeightKg),
		RemainingVolumeCbm:     container.VolumeCbm.Sub(allocated.VolumeCbm),
		ContainerBalanced:      allocated.Equal(container),
		CargoBalanced:          cargoBalanced,
	}
}

type SeaSharedContainerRepo interface {
	List(ctx context.Context, organizationID, anchorOrderID, transportExecutionID uuid.UUID, keyword string, page, pageSize int) ([]*SeaSharedContainer, int, error)
	Get(ctx context.Context, organizationID, anchorOrderID, id uuid.UUID) (*SeaSharedContainer, error)
	ListCandidates(ctx context.Context, organizationID, anchorOrderID, transportExecutionID uuid.UUID, keyword string, page, pageSize int) ([]*SeaSharedContainerCandidateOrder, int, error)
	Create(ctx context.Context, organizationID, actorID, anchorOrderID uuid.UUID, input *SeaSharedContainer, audit *AuditEvent) (*SeaSharedContainer, error)
	Update(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, input *SeaSharedContainer, audit *AuditEvent) (*SeaSharedContainer, error)
	Delete(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, audit *AuditEvent) error
	SaveDraft(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, allocations []*SeaSharedContainerAllocationInput, audit *AuditEvent) (*SeaSharedContainer, error)
	Confirm(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, allocations []*SeaSharedContainerAllocationInput, audit *AuditEvent) (*SeaSharedContainer, error)
	Withdraw(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, audit *AuditEvent) (*SeaSharedContainer, error)
}

type SeaSharedContainerUsecase struct{ repo SeaSharedContainerRepo }

func NewSeaSharedContainerUsecase(repo SeaSharedContainerRepo) *SeaSharedContainerUsecase {
	return &SeaSharedContainerUsecase{repo: repo}
}

func (uc *SeaSharedContainerUsecase) List(ctx context.Context, organizationID, anchorOrderID, executionID uuid.UUID, keyword string, page, pageSize int) ([]*SeaSharedContainer, int, error) {
	if organizationID == uuid.Nil || anchorOrderID == uuid.Nil || executionID == uuid.Nil || !ValidListPagination(page, pageSize) || utf8.RuneCountInString(keyword) > 100 {
		return nil, 0, ErrSeaSharedContainerInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, anchorOrderID, executionID, strings.TrimSpace(keyword), page, pageSize)
}

func (uc *SeaSharedContainerUsecase) Get(ctx context.Context, organizationID, anchorOrderID, id uuid.UUID) (*SeaSharedContainer, error) {
	if organizationID == uuid.Nil || anchorOrderID == uuid.Nil || id == uuid.Nil {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	return uc.repo.Get(ctx, organizationID, anchorOrderID, id)
}

func (uc *SeaSharedContainerUsecase) ListCandidates(ctx context.Context, organizationID, anchorOrderID, executionID uuid.UUID, keyword string, page, pageSize int) ([]*SeaSharedContainerCandidateOrder, int, error) {
	if organizationID == uuid.Nil || anchorOrderID == uuid.Nil || executionID == uuid.Nil || !ValidListPagination(page, pageSize) || utf8.RuneCountInString(keyword) > 100 {
		return nil, 0, ErrSeaSharedContainerInvalidArgument
	}
	return uc.repo.ListCandidates(ctx, organizationID, anchorOrderID, executionID, strings.TrimSpace(keyword), page, pageSize)
}

func (uc *SeaSharedContainerUsecase) Create(ctx context.Context, organizationID, actorID, anchorOrderID uuid.UUID, input *SeaSharedContainer) (*SeaSharedContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || anchorOrderID == uuid.Nil {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	normalized, err := NormalizeSeaSharedContainer(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = uuid.Must(uuid.NewV7())
	normalized.OrganizationID = organizationID
	normalized.Status = SeaSharedContainerStatusDraft
	normalized.Version = 1
	return uc.repo.Create(ctx, organizationID, actorID, anchorOrderID, normalized, sharedContainerAudit(organizationID, actorID, "sea_shared_container.create", normalized.ID))
}

func (uc *SeaSharedContainerUsecase) Update(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, input *SeaSharedContainer) (*SeaSharedContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || anchorOrderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	normalized, err := NormalizeSeaSharedContainer(input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Update(ctx, organizationID, actorID, anchorOrderID, id, expectedVersion, normalized, sharedContainerAudit(organizationID, actorID, "sea_shared_container.update", id))
}

func (uc *SeaSharedContainerUsecase) Delete(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64) error {
	if organizationID == uuid.Nil || actorID == uuid.Nil || anchorOrderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return ErrSeaSharedContainerInvalidArgument
	}
	return uc.repo.Delete(ctx, organizationID, actorID, anchorOrderID, id, expectedVersion, sharedContainerAudit(organizationID, actorID, "sea_shared_container.delete", id))
}

func (uc *SeaSharedContainerUsecase) SaveDraft(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, allocations []*SeaSharedContainerAllocationInput) (*SeaSharedContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || anchorOrderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	normalized, err := NormalizeSeaSharedAllocations(allocations)
	if err != nil {
		return nil, err
	}
	return uc.repo.SaveDraft(ctx, organizationID, actorID, anchorOrderID, id, expectedVersion, normalized, sharedContainerAudit(organizationID, actorID, "sea_shared_container.save_draft", id))
}

// Confirm 确认共享箱分配。allocations 非 nil 时在同一事务内按该输入保存并严格守恒确认，
// 避免客户端两步请求出现“草稿已保存、确认失败”的部分成功；allocations 为 nil 时按当前分配确认。
func (uc *SeaSharedContainerUsecase) Confirm(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64, allocations []*SeaSharedContainerAllocationInput) (*SeaSharedContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || anchorOrderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	if allocations != nil {
		normalized, err := NormalizeSeaSharedAllocations(allocations)
		if err != nil {
			return nil, err
		}
		allocations = normalized
	}
	return uc.repo.Confirm(ctx, organizationID, actorID, anchorOrderID, id, expectedVersion, allocations, sharedContainerAudit(organizationID, actorID, "sea_shared_container.confirm", id))
}

func (uc *SeaSharedContainerUsecase) Withdraw(ctx context.Context, organizationID, actorID, anchorOrderID, id uuid.UUID, expectedVersion uint64) (*SeaSharedContainer, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || anchorOrderID == uuid.Nil || id == uuid.Nil || expectedVersion == 0 {
		return nil, ErrSeaSharedContainerInvalidArgument
	}
	return uc.repo.Withdraw(ctx, organizationID, actorID, anchorOrderID, id, expectedVersion, sharedContainerAudit(organizationID, actorID, "sea_shared_container.withdraw", id))
}

func sharedContainerAudit(organizationID, actorID uuid.UUID, action string, id uuid.UUID) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", Details: map[string]string{"shared_container.id": id.String()}}
}
