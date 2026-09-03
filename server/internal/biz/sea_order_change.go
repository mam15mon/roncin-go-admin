package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	ResultRoleOriginal = "ORIGINAL"
	ResultRoleCreated  = "CREATED"

	SplitTargetTypeCurrent   = "CURRENT"
	SplitTargetTypeCandidate = "CANDIDATE"
	SplitTargetTypeNew       = "NEW"

	ResponsibilityTypeCarrier      = "CARRIER"
	ResponsibilityTypeCustomer     = "CUSTOMER"
	ResponsibilityTypeCustoms      = "CUSTOMS"
	ResponsibilityTypeOwnCompany   = "OWN_COMPANY"
	ResponsibilityTypeForceMajeure = "FORCE_MAJEURE"
	ResponsibilityTypeOther        = "OTHER"

	EventTypeSplit        = "SPLIT"
	EventTypeReassignment = "REASSIGNMENT"
)

var (
	ErrSeaOrderSplitInvalidArgument            = errors.BadRequest("SEA_ORDER_SPLIT_INVALID_ARGUMENT", "拆票参数不合法")
	ErrSeaOrderSplitBlocked                    = errors.BadRequest("SEA_ORDER_SPLIT_BLOCKED", "当前操作票状态或财务事实不允许拆票")
	ErrSeaOrderSplitConservationFailed         = errors.BadRequest("SEA_ORDER_SPLIT_CONSERVATION_FAILED", "拆票数量守恒校验失败")
	ErrSeaOrderSplitEntityCrossesResults       = errors.BadRequest("SEA_ORDER_SPLIT_ENTITY_CROSSES_RESULTS", "分单或集装箱不可跨结果票分配")
	ErrSeaOrderSplitVersionConflict            = errors.Conflict("SEA_ORDER_SPLIT_VERSION_CONFLICT", "数据已被更新，请刷新后重试")
	ErrSeaOrderSplitIdempotencyConflict        = errors.Conflict("SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT", "相同幂等键请求指纹冲突")
	ErrSeaOrderReassignmentInvalidArgument     = errors.BadRequest("SEA_ORDER_REASSIGNMENT_INVALID_ARGUMENT", "改配参数不合法")
	ErrSeaOrderReassignmentBlocked             = errors.BadRequest("SEA_ORDER_REASSIGNMENT_BLOCKED", "当前操作票状态或财务事实不允许改配")
	ErrSeaOrderReassignmentTargetConflict      = errors.Conflict("SEA_ORDER_REASSIGNMENT_TARGET_CONFLICT", "目标 MBL 冲突或不可用")
	ErrSeaOrderReassignmentVersionConflict     = errors.Conflict("SEA_ORDER_REASSIGNMENT_VERSION_CONFLICT", "数据已被更新，请刷新后重试")
	ErrSeaOrderReassignmentIdempotencyConflict = errors.Conflict("SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT", "相同幂等键请求指纹冲突")
	ErrSeaTransportExecutionNotFound           = errors.NotFound("SEA_TRANSPORT_EXECUTION_NOT_FOUND", "海运运输执行记录不存在")
	ErrSeaCargoAllocationNotFound              = errors.NotFound("SEA_CARGO_ALLOCATION_NOT_FOUND", "海运货物分配记录不存在")
)

type SeaOrderChangeActions struct {
	CanSplit               bool
	CanReassign            bool
	SplitBlockedReasons    []string
	ReassignBlockedReasons []string
}

type SeaOrderSplitContext struct {
	OrderID                        uuid.UUID
	OrderNo                        string
	BusinessType                   string
	ShipmentType                   string
	FlowStatus                     string
	OrderVersion                   uint64
	CustomerReferenceNo            string
	InternalReferenceNo            string
	BookingNotes                   string
	AllocationNotes                string
	OperationNotes                 string
	CurrentMasterBill              *SeaMasterBillSummary
	CurrentLinkID                  uuid.UUID
	CurrentLinkVersion             uint64
	DocumentStructure              string
	CargoAllocationStatus          string
	CargoAllocationVersion         uint64
	HouseBills                     []*SeaOrderSplitHouseBillItem
	CargoItems                     []*SeaOrderSplitCargoItem
	Containers                     []*SeaOrderSplitContainerItem
	Allocations                    []*SeaOrderSplitAllocationItem
	DraftFees                      []*SeaOrderSplitDraftFeeItem
	Attachments                    []*SeaOrderSplitAttachmentItem
	ContainerPlans                 []*SeaOrderSplitContainerPlanItem
	AttachmentReferenceFingerprint string
}

type SeaOrderSplitHouseBillItem struct {
	ID      uuid.UUID
	HouseNo string
	Status  string
	Version uint64
}

type SeaOrderSplitCargoItem struct {
	ID            uuid.UUID
	CargoName     string
	PackageCount  int32
	GrossWeightKg decimal.Decimal
	VolumeCbm     decimal.Decimal
	Version       uint64
}

type SeaOrderSplitContainerItem struct {
	ID                uuid.UUID
	ContainerNo       string
	ContainerSpecID   uuid.UUID
	ContainerSpecName string
	PackageCount      int32
	GrossWeightKg     decimal.Decimal
	VolumeCbm         decimal.Decimal
	Version           uint64
}

type SeaOrderSplitAllocationItem struct {
	ID            uuid.UUID
	CargoItemID   uuid.UUID
	HouseBillID   uuid.UUID
	ContainerID   *uuid.UUID
	PackageCount  int32
	GrossWeightKg decimal.Decimal
	VolumeCbm     decimal.Decimal
}

type SeaOrderSplitDraftFeeItem struct {
	ID                  uuid.UUID
	FeeCode             string
	FeeName             string
	Direction           string
	SettlementPartyID   uuid.UUID
	SettlementPartyName string
	Currency            string
	TotalAmount         decimal.Decimal
	BaseCurrency        string
	BaseCurrencyAmount  decimal.Decimal
	Version             uint64
}

type SeaOrderSplitAttachmentItem struct {
	ID       uuid.UUID
	AssetID  uuid.UUID
	FileName string
	MIMEType string
	FileSize int64
	DocType  string
}

type SeaOrderSplitContainerPlanItem struct {
	ContainerSpecID   uuid.UUID
	ContainerSpecName string
	Quantity          int32
}

type SeaOrderSplitTargetInput struct {
	ClientTargetKey     string
	TargetType          string // CURRENT | CANDIDATE | NEW
	CandidateID         *uuid.UUID
	CandidateVersion    *uint64
	MasterNo            string
	IssuerPartnerID     *uuid.UUID
	CarrierID           *uuid.UUID
	VesselName          string
	VoyageNo            string
	ETD                 string
	ETA                 string
	OriginLocationID    *uuid.UUID
	DischargeLocationID *uuid.UUID
	TransitLocationID   *uuid.UUID
	CandidateTEID       *uuid.UUID
	CandidateTEVersion  *uint64
}

type SeaOrderSplitResultInput struct {
	ClientResultKey        string
	ResultRole             string // ORIGINAL | CREATED
	ClientTargetKey        string
	HouseBillIDs           []uuid.UUID
	DraftFeeIDs            []uuid.UUID
	AttachmentReferenceIDs []uuid.UUID
	InternalReferenceNo    *string
	BookingNotes           *string
	AllocationNotes        *string
	OperationNotes         *string
}

type SeaOrderSplitExpectedVersions struct {
	OrderVersion                   uint64
	LinkVersion                    uint64
	AllocationVersion              uint64
	HouseBillVersions              map[uuid.UUID]uint64
	CargoItemVersions              map[uuid.UUID]uint64
	ContainerVersions              map[uuid.UUID]uint64
	FeeVersions                    map[uuid.UUID]uint64
	CandidateMBLVersions           map[uuid.UUID]uint64
	AttachmentReferenceFingerprint string
	CandidateTEVersions            map[uuid.UUID]uint64
}

type SeaOrderSplitInput struct {
	OrderID            uuid.UUID
	IdempotencyKey     string
	RequestFingerprint string
	Note               *string
	Targets            []*SeaOrderSplitTargetInput
	Results            []*SeaOrderSplitResultInput
	ExpectedVersions   *SeaOrderSplitExpectedVersions
}

type SeaOrderSplitPreview struct {
	IsValid            bool
	ConservationPassed bool
	ValidationErrors   []*SeaOrderSplitValidationError
	Baseline           SeaOrderSplitQuantitySummary
	Allocated          SeaOrderSplitQuantitySummary
	Remaining          SeaOrderSplitQuantitySummary
	Results            []*SeaOrderSplitPreviewResultItem
}

type SeaOrderSplitValidationError struct {
	Reason          string
	Message         string
	Field           string
	ClientResultKey string
	HouseBillID     string
	ContainerID     string
	CargoItemID     string
	FeeID           string
	BaselineValue   string
	AllocatedValue  string
	DiffValue       string
}

type SeaOrderSplitQuantitySummary struct {
	PackageCount   int32
	GrossWeightKg  decimal.Decimal
	VolumeCbm      decimal.Decimal
	ContainerCount int32
	HouseBillCount int32
	FeeCount       int32
}

type SeaOrderSplitPreviewResultItem struct {
	ClientResultKey     string
	ResultRole          string
	ClientTargetKey     string
	PackageCount        int32
	GrossWeightKg       decimal.Decimal
	VolumeCbm           decimal.Decimal
	ContainerCount      int32
	HouseBillCount      int32
	FeeCount            int32
	AttachmentCount     int32
	ContainerPlans      []*SeaOrderSplitContainerPlanItem
	InternalReferenceNo *string
	BookingNotes        string
	AllocationNotes     string
	OperationNotes      string
}

type SeaOrderSplitEvent struct {
	ID                   uuid.UUID
	CreatedAt            time.Time
	OrganizationID       uuid.UUID
	SourceOrderID        uuid.UUID
	SourceOrderNo        string
	IdempotencyKey       string
	RequestFingerprint   string
	Note                 *string
	SourceOrderVersion   uint64
	SourceLinkID         uuid.UUID
	SourceLinkVersion    uint64
	SourceAllocationVer  uint64
	BeforeSnapshot       []byte
	ConservationSnapshot []byte
	CreatedBy            *uuid.UUID
	Results              []*SeaOrderSplitResult
	ReassignmentEventIDs []uuid.UUID
}

type SeaOrderSplitResult struct {
	ID                  uuid.UUID
	CreatedAt           time.Time
	SplitEventID        uuid.UUID
	OrganizationID      uuid.UUID
	OrderID             uuid.UUID
	OrderNo             string
	ResultRole          string
	Sequence            int
	ClientResultKey     string
	InitialMasterBillID uuid.UUID
	FinalMasterBillID   uuid.UUID
	ResultSnapshot      []byte
}

type SeaOrderReassignmentTargetInput struct {
	TargetType          string // CANDIDATE | NEW
	CandidateID         *uuid.UUID
	CandidateVersion    *uint64
	MasterNo            string
	IssuerPartnerID     *uuid.UUID
	CarrierID           *uuid.UUID
	VesselName          string
	VoyageNo            string
	ETD                 string
	ETA                 string
	OriginLocationID    *uuid.UUID
	DischargeLocationID *uuid.UUID
	TransitLocationID   *uuid.UUID
	CandidateTEID       *uuid.UUID
	CandidateTEVersion  *uint64
}

type SeaOrderReassignmentInput struct {
	OrderID                     uuid.UUID
	IdempotencyKey              string
	RequestFingerprint          string
	Target                      *SeaOrderReassignmentTargetInput
	Reason                      string
	ResponsibilityType          string
	ResponsiblePartnerID        *uuid.UUID
	ExpectedOrderVersion        uint64
	ExpectedLinkVersion         uint64
	ExpectedCandidateMBLVersion *uint64
	ExpectedCandidateTEVersion  *uint64
}

type SeaOrderReassignmentPreview struct {
	IsValid            bool
	Errors             []string
	CurrentMasterBill  *SeaMasterBillSummary
	TargetMasterBill   *SeaMasterBillSummary
	TargetMemberCount  int32
	Differences        []*VoyageDifference
	OrderVersion       uint64
	CurrentLinkVersion uint64
}

type VoyageDifference struct {
	FieldName    string
	Label        string
	CurrentValue string
	TargetValue  string
	IsDifferent  bool
}

type SeaOrderReassignmentEvent struct {
	ID                           uuid.UUID
	CreatedAt                    time.Time
	OrganizationID               uuid.UUID
	OrderID                      uuid.UUID
	OrderNo                      string
	SplitEventID                 *uuid.UUID
	SplitResultID                *uuid.UUID
	IdempotencyKey               string
	RequestFingerprint           string
	PreviousMasterBillID         uuid.UUID
	TargetMasterBillID           uuid.UUID
	PreviousTransportExecutionID uuid.UUID
	TargetTransportExecutionID   uuid.UUID
	PreviousLinkID               uuid.UUID
	TargetLinkID                 uuid.UUID
	PreviousLinkVersion          uint64
	TargetLinkVersion            uint64
	Reason                       string
	ResponsibilityType           string
	ResponsiblePartnerID         *uuid.UUID
	ResponsiblePartnerName       *string
	BeforeSnapshot               []byte
	AfterSnapshot                []byte
	CreatedBy                    *uuid.UUID
}

type SeaOrderChangeEventSummary struct {
	ID                  uuid.UUID
	EventType           string // SPLIT | REASSIGNMENT
	CreatedAt           time.Time
	OperatorID          *uuid.UUID
	OperatorName        string
	NoteOrReason        string
	SplitSummary        *SeaOrderSplitEventSummary
	ReassignmentSummary *SeaOrderReassignmentEventSummary
}

type SeaOrderSplitEventSummary struct {
	SourceOrderID uuid.UUID
	SourceOrderNo string
	ResultCount   int32
	Results       []*SeaOrderSplitResultSummaryItem
}

type SeaOrderSplitResultSummaryItem struct {
	ResultRole    string
	OrderID       uuid.UUID
	OrderNo       string
	FinalMasterNo string
	PackageCount  int32
	GrossWeightKg decimal.Decimal
	VolumeCbm     decimal.Decimal
}

type SeaOrderReassignmentEventSummary struct {
	OrderID                uuid.UUID
	OrderNo                string
	PreviousMasterNo       string
	TargetMasterNo         string
	ResponsibilityType     string
	ResponsiblePartnerName string
	Reason                 string
}

type SeaOrderChangeEventDetail struct {
	ID                       uuid.UUID
	EventType                string
	CreatedAt                time.Time
	OperatorID               *uuid.UUID
	OperatorName             string
	NoteOrReason             string
	BeforeSnapshotJSON       string
	AfterSnapshotJSON        string
	ConservationSnapshotJSON string
	SplitSummary             *SeaOrderSplitEventSummary
	ReassignmentSummary      *SeaOrderReassignmentEventSummary
}

type SeaOrderChangeRepo interface {
	GetChangeActions(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderChangeActions, error)
	GetSplitContext(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderSplitContext, error)
	PreviewSplit(ctx context.Context, organizationID uuid.UUID, input *SeaOrderSplitInput) (*SeaOrderSplitPreview, error)
	ExecuteSplit(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderSplitInput, audit *AuditEvent) (*SeaOrderSplitEvent, error)
	GetSplitEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*SeaOrderSplitEvent, error)
	GetSplitEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*SeaOrderSplitEvent, error)
	PreviewReassignment(ctx context.Context, organizationID uuid.UUID, input *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error)
	ExecuteReassignment(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderReassignmentInput, audit *AuditEvent) (*SeaOrderReassignmentEvent, error)
	GetReassignmentEventByIdempotencyKey(ctx context.Context, organizationID uuid.UUID, idempotencyKey string) (*SeaOrderReassignmentEvent, error)
	GetReassignmentEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID) (*SeaOrderReassignmentEvent, error)
	ListChangeEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*SeaOrderChangeEventSummary, int32, error)
	GetChangeEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*SeaOrderChangeEventDetail, error)
}

type SeaOrderChangeUsecase struct {
	repo       SeaOrderChangeRepo
	transactor Transactor
}

func NewSeaOrderChangeUsecase(repo SeaOrderChangeRepo, transactor Transactor) *SeaOrderChangeUsecase {
	return &SeaOrderChangeUsecase{
		repo:       repo,
		transactor: transactor,
	}
}

func (uc *SeaOrderChangeUsecase) GetChangeActions(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderChangeActions, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	return uc.repo.GetChangeActions(ctx, organizationID, orderID)
}

func (uc *SeaOrderChangeUsecase) GetSplitContext(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderSplitContext, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	return uc.repo.GetSplitContext(ctx, organizationID, orderID)
}

func parseOptionalTime(s string) *time.Time {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	return &t
}

func validateSplitTargetsAndResults(targets []*SeaOrderSplitTargetInput, results []*SeaOrderSplitResultInput, expected *SeaOrderSplitExpectedVersions) error {
	if len(targets) == 0 {
		return ErrSeaOrderSplitInvalidArgument
	}

	seenTargetKeys := make(map[string]*SeaOrderSplitTargetInput, len(targets))
	for _, target := range targets {
		if target == nil {
			return ErrSeaOrderSplitInvalidArgument
		}
		targetKey := target.ClientTargetKey
		if strings.TrimSpace(targetKey) == "" {
			return ErrSeaOrderSplitInvalidArgument
		}
		if _, exists := seenTargetKeys[targetKey]; exists {
			return ErrSeaOrderSplitInvalidArgument
		}
		seenTargetKeys[targetKey] = target

		switch target.TargetType {
		case SplitTargetTypeCurrent:
			if target.CandidateID != nil ||
				target.CandidateVersion != nil ||
				target.CandidateTEID != nil ||
				target.CandidateTEVersion != nil ||
				target.MasterNo != "" ||
				target.IssuerPartnerID != nil ||
				target.CarrierID != nil ||
				target.VesselName != "" ||
				target.VoyageNo != "" ||
				target.ETD != "" ||
				target.ETA != "" ||
				target.OriginLocationID != nil ||
				target.DischargeLocationID != nil ||
				target.TransitLocationID != nil {
				return ErrSeaOrderSplitInvalidArgument
			}

		case SplitTargetTypeCandidate:
			if target.CandidateID == nil || *target.CandidateID == uuid.Nil ||
				target.CandidateVersion == nil || *target.CandidateVersion == 0 ||
				target.CandidateTEID == nil || *target.CandidateTEID == uuid.Nil ||
				target.CandidateTEVersion == nil || *target.CandidateTEVersion == 0 ||
				target.IssuerPartnerID == nil || *target.IssuerPartnerID == uuid.Nil {
				return ErrSeaOrderSplitInvalidArgument
			}
			if expected != nil {
				if expected.CandidateMBLVersions == nil ||
					expected.CandidateMBLVersions[*target.CandidateID] != *target.CandidateVersion ||
					expected.CandidateTEVersions == nil ||
					expected.CandidateTEVersions[*target.CandidateTEID] != *target.CandidateTEVersion {
					return ErrSeaOrderSplitInvalidArgument
				}
			}

		case SplitTargetTypeNew:
			if target.CandidateID != nil ||
				target.CandidateVersion != nil ||
				target.CandidateTEID != nil ||
				target.CandidateTEVersion != nil {
				return ErrSeaOrderSplitInvalidArgument
			}
			if _, err := ValidateAndNormalizeSeaMasterNo(target.MasterNo); err != nil {
				return err
			}
			if target.IssuerPartnerID == nil || *target.IssuerPartnerID == uuid.Nil {
				return ErrSeaOrderSplitInvalidArgument
			}
			if strings.TrimSpace(target.VesselName) == "" || strings.TrimSpace(target.VoyageNo) == "" {
				return ErrSeaOrderSplitInvalidArgument
			}
			if strings.TrimSpace(target.ETD) != "" && parseOptionalTime(target.ETD) == nil {
				return ErrSeaMasterBillInvalidArgument
			}
			if strings.TrimSpace(target.ETA) != "" && parseOptionalTime(target.ETA) == nil {
				return ErrSeaMasterBillInvalidArgument
			}

		default:
			return ErrSeaOrderSplitInvalidArgument
		}
	}

	if len(results) == 0 {
		return ErrSeaOrderSplitInvalidArgument
	}
	for _, res := range results {
		if res == nil {
			return ErrSeaOrderSplitInvalidArgument
		}
		if _, exists := seenTargetKeys[res.ClientTargetKey]; !exists {
			return ErrSeaOrderSplitInvalidArgument
		}
	}

	return nil
}

func validateReassignmentCandidateTarget(
	target *SeaOrderReassignmentTargetInput,
	expectedMBLVersion *uint64,
	expectedTEVersion *uint64,
) error {
	if target == nil {
		return ErrSeaOrderReassignmentInvalidArgument
	}
	switch target.TargetType {
	case SplitTargetTypeCandidate:
		if target.CandidateID == nil || *target.CandidateID == uuid.Nil ||
			target.CandidateVersion == nil || *target.CandidateVersion == 0 ||
			target.CandidateTEID == nil || *target.CandidateTEID == uuid.Nil ||
			target.CandidateTEVersion == nil || *target.CandidateTEVersion == 0 ||
			target.IssuerPartnerID == nil || *target.IssuerPartnerID == uuid.Nil {
			return ErrSeaOrderReassignmentInvalidArgument
		}
		if expectedMBLVersion != nil && (*expectedMBLVersion == 0 || *expectedMBLVersion != *target.CandidateVersion) {
			return ErrSeaOrderReassignmentInvalidArgument
		}
		if expectedTEVersion != nil && (*expectedTEVersion == 0 || *expectedTEVersion != *target.CandidateTEVersion) {
			return ErrSeaOrderReassignmentInvalidArgument
		}
		if (expectedMBLVersion == nil) != (expectedTEVersion == nil) {
			return ErrSeaOrderReassignmentInvalidArgument
		}
		return nil

	case SplitTargetTypeNew:
		if target.CandidateID != nil ||
			target.CandidateVersion != nil ||
			target.CandidateTEID != nil ||
			target.CandidateTEVersion != nil {
			return ErrSeaOrderReassignmentInvalidArgument
		}
		if expectedMBLVersion != nil || expectedTEVersion != nil {
			return ErrSeaOrderReassignmentInvalidArgument
		}
		return nil

	default:
		return ErrSeaOrderReassignmentInvalidArgument
	}
}

func (uc *SeaOrderChangeUsecase) PreviewSplit(ctx context.Context, organizationID uuid.UUID, input *SeaOrderSplitInput) (*SeaOrderSplitPreview, error) {
	if organizationID == uuid.Nil || input == nil || input.OrderID == uuid.Nil {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	if err := validateSplitTargetsAndResults(input.Targets, input.Results, input.ExpectedVersions); err != nil {
		return nil, err
	}
	return uc.repo.PreviewSplit(ctx, organizationID, input)
}

func (uc *SeaOrderChangeUsecase) ExecuteSplit(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderSplitInput) (*SeaOrderSplitEvent, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || input == nil || input.OrderID == uuid.Nil {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" || strings.TrimSpace(input.RequestFingerprint) == "" {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	if input.ExpectedVersions == nil ||
		input.ExpectedVersions.OrderVersion == 0 ||
		input.ExpectedVersions.LinkVersion == 0 ||
		input.ExpectedVersions.AllocationVersion == 0 {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	if len(input.Results) < 2 {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	if err := validateSplitTargetsAndResults(input.Targets, input.Results, input.ExpectedVersions); err != nil {
		return nil, err
	}
	if input.Note != nil {
		trimmed := strings.TrimSpace(*input.Note)
		if utf8.RuneCountInString(trimmed) > 500 {
			return nil, ErrSeaOrderSplitInvalidArgument
		}
		input.Note = &trimmed
	}

	// 1. 先查幂等键并核对指纹（先于依赖当前业务状态的 preview，确保幂等重试不受业务状态变更影响）
	existing, err := uc.repo.GetSplitEventByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.RequestFingerprint != input.RequestFingerprint {
			return nil, ErrSeaOrderSplitIdempotencyConflict
		}
		return existing, nil
	}

	// 2. 业务层前置校验：守恒与门禁判定
	preview, err := uc.PreviewSplit(ctx, organizationID, input)
	if err != nil {
		return nil, err
	}
	if !preview.IsValid || !preview.ConservationPassed {
		if len(preview.ValidationErrors) > 0 {
			firstErr := preview.ValidationErrors[0]
			if firstErr.Reason == "CONTAINER_CROSSES_RESULTS" {
				return nil, MetadataError(ErrSeaOrderSplitEntityCrossesResults, map[string]string{
					"reason":  firstErr.Reason,
					"message": firstErr.Message,
				})
			}
			if firstErr.Reason == "QUANTITY_CONSERVATION_FAILED" {
				return nil, MetadataError(ErrSeaOrderSplitConservationFailed, map[string]string{
					"reason":  firstErr.Reason,
					"message": firstErr.Message,
				})
			}
			return nil, MetadataError(ErrSeaOrderSplitInvalidArgument, map[string]string{
				"reason":  firstErr.Reason,
				"message": firstErr.Message,
			})
		}
		return nil, ErrSeaOrderSplitConservationFailed
	}

	auditResult := "success"
	if isInjectedAuditFailure(ctx) {
		auditResult = "INJECTED_AUDIT_FAILURE"
	}

	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.sea.split",
		Result:         auditResult,
		Details: map[string]string{
			"order.id":        input.OrderID.String(),
			"idempotency_key": input.IdempotencyKey,
		},
	}

	var splitEventID uuid.UUID
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 3. 事务内二次幂等防并发
		existingInTx, transErr := uc.repo.GetSplitEventByIdempotencyKey(txCtx, organizationID, input.IdempotencyKey)
		if transErr != nil {
			return transErr
		}
		if existingInTx != nil {
			if existingInTx.RequestFingerprint != input.RequestFingerprint {
				return ErrSeaOrderSplitIdempotencyConflict
			}
			splitEventID = existingInTx.ID
			return nil
		}

		// 4. 仓储层执行拆票原子写入
		saved, transErr := uc.repo.ExecuteSplit(txCtx, organizationID, actorID, input, audit)
		if transErr != nil {
			return transErr
		}
		splitEventID = saved.ID
		return nil
	})

	if err != nil {
		// 5. 并发竞态检测：只对明确的幂等唯一冲突做幂等重读，禁止覆盖真实业务错误
		if errors.Is(err, ErrSeaOrderSplitIdempotencyConflict) {
			existingAfter, lookupErr := uc.repo.GetSplitEventByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if existingAfter != nil {
				if existingAfter.RequestFingerprint == input.RequestFingerprint {
					return existingAfter, nil
				}
				return nil, ErrSeaOrderSplitIdempotencyConflict
			}
		}
		return nil, err
	}

	// 6. 事务提交后，必须用普通 ctx 从仓储重读最终响应，不能返回 txCtx 内拼装对象
	return uc.repo.GetSplitEvent(ctx, organizationID, input.OrderID, splitEventID)
}

func (uc *SeaOrderChangeUsecase) PreviewReassignment(ctx context.Context, organizationID uuid.UUID, input *SeaOrderReassignmentInput) (*SeaOrderReassignmentPreview, error) {
	if organizationID == uuid.Nil || input == nil || input.OrderID == uuid.Nil || input.Target == nil {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}
	if err := validateReassignmentCandidateTarget(input.Target, nil, nil); err != nil {
		return nil, err
	}
	return uc.repo.PreviewReassignment(ctx, organizationID, input)
}

func (uc *SeaOrderChangeUsecase) ExecuteReassignment(ctx context.Context, organizationID, actorID uuid.UUID, input *SeaOrderReassignmentInput) (*SeaOrderReassignmentEvent, error) {
	if organizationID == uuid.Nil || actorID == uuid.Nil || input == nil || input.OrderID == uuid.Nil || input.Target == nil {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" || strings.TrimSpace(input.RequestFingerprint) == "" {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}
	if input.ExpectedOrderVersion == 0 || input.ExpectedLinkVersion == 0 {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}
	if input.Target.TargetType == SplitTargetTypeCandidate &&
		(input.ExpectedCandidateMBLVersion == nil || input.ExpectedCandidateTEVersion == nil) {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}
	if err := validateReassignmentCandidateTarget(
		input.Target,
		input.ExpectedCandidateMBLVersion,
		input.ExpectedCandidateTEVersion,
	); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(input.Reason)
	if utf8.RuneCountInString(reason) < 1 || utf8.RuneCountInString(reason) > 500 {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}
	input.Reason = reason
	if !isValidResponsibilityType(input.ResponsibilityType) {
		return nil, ErrSeaOrderReassignmentInvalidArgument
	}

	// 1. 先查幂等键并核对指纹（先于依赖当前业务状态的 preview）
	existing, err := uc.repo.GetReassignmentEventByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.RequestFingerprint != input.RequestFingerprint {
			return nil, ErrSeaOrderReassignmentIdempotencyConflict
		}
		return existing, nil
	}

	// 2. 业务层前置校验：改配可行性与航程差异比对
	preview, err := uc.PreviewReassignment(ctx, organizationID, input)
	if err != nil {
		return nil, err
	}
	if !preview.IsValid {
		if len(preview.Errors) > 0 {
			return nil, MetadataError(ErrSeaOrderReassignmentBlocked, map[string]string{
				"error": preview.Errors[0],
			})
		}
		return nil, ErrSeaOrderReassignmentBlocked
	}

	auditResult := "success"
	if isInjectedAuditFailure(ctx) {
		auditResult = "INJECTED_AUDIT_FAILURE"
	}

	audit := &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &actorID,
		Action:         "order.sea.reassign",
		Result:         auditResult,
		Details: map[string]string{
			"order.id":        input.OrderID.String(),
			"idempotency_key": input.IdempotencyKey,
			"reason":          reason,
			"responsibility":  input.ResponsibilityType,
		},
	}

	var reassignmentEventID uuid.UUID
	err = uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 3. 事务内二次幂等防并发
		existingInTx, transErr := uc.repo.GetReassignmentEventByIdempotencyKey(txCtx, organizationID, input.IdempotencyKey)
		if transErr != nil {
			return transErr
		}
		if existingInTx != nil {
			if existingInTx.RequestFingerprint != input.RequestFingerprint {
				return ErrSeaOrderReassignmentIdempotencyConflict
			}
			reassignmentEventID = existingInTx.ID
			return nil
		}

		// 4. 仓储层执行改配原子写入
		saved, transErr := uc.repo.ExecuteReassignment(txCtx, organizationID, actorID, input, audit)
		if transErr != nil {
			return transErr
		}
		reassignmentEventID = saved.ID
		return nil
	})

	if err != nil {
		// 5. 并发竞态检测：只对明确的幂等唯一冲突重查历史事件
		if errors.Is(err, ErrSeaOrderReassignmentIdempotencyConflict) {
			existingAfter, lookupErr := uc.repo.GetReassignmentEventByIdempotencyKey(ctx, organizationID, input.IdempotencyKey)
			if lookupErr != nil {
				return nil, lookupErr
			}
			if existingAfter != nil {
				if existingAfter.RequestFingerprint == input.RequestFingerprint {
					return existingAfter, nil
				}
				return nil, ErrSeaOrderReassignmentIdempotencyConflict
			}
		}
		return nil, err
	}

	// 6. 事务提交后，必须用普通 ctx 从仓储重读最终响应，不能返回 txCtx 内拼装对象
	return uc.repo.GetReassignmentEvent(ctx, organizationID, input.OrderID, reassignmentEventID)
}

func (uc *SeaOrderChangeUsecase) ListChangeEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*SeaOrderChangeEventSummary, int32, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil || !ValidListPagination(int(page), int(pageSize)) {
		return nil, 0, ErrSeaOrderSplitInvalidArgument
	}
	return uc.repo.ListChangeEvents(ctx, organizationID, orderID, page, pageSize)
}

func (uc *SeaOrderChangeUsecase) GetChangeEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*SeaOrderChangeEventDetail, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil || eventID == uuid.Nil {
		return nil, ErrSeaOrderSplitInvalidArgument
	}
	return uc.repo.GetChangeEvent(ctx, organizationID, orderID, eventID, eventType)
}

func isValidResponsibilityType(t string) bool {
	switch t {
	case ResponsibilityTypeCarrier, ResponsibilityTypeCustomer, ResponsibilityTypeCustoms, ResponsibilityTypeOwnCompany, ResponsibilityTypeForceMajeure, ResponsibilityTypeOther:
		return true
	default:
		return false
	}
}

// ComputeFingerprint 计算请求内容的稳定 SHA256 指纹
func ComputeFingerprint(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// FormatDecimal3 格式化 3 位小数
func FormatDecimal3(d decimal.Decimal) string {
	return d.StringFixed(3)
}

// FormatDecimal6 格式化 6 位小数
func FormatDecimal6(d decimal.Decimal) string {
	return d.StringFixed(6)
}

// FormatDecimal8 格式化 8 位小数
func FormatDecimal8(d decimal.Decimal) string {
	return d.StringFixed(8)
}

// MetadataError 返回携带结构化元数据的业务错误
func MetadataError(base *errors.Error, metadata map[string]string) *errors.Error {
	return base.WithMetadata(metadata)
}

type injectAuditFailureKey struct{}

// WithInjectedAuditFailure 用于集成测试中注入审计日志保存失败
func WithInjectedAuditFailure(ctx context.Context) context.Context {
	return context.WithValue(ctx, injectAuditFailureKey{}, true)
}

func isInjectedAuditFailure(ctx context.Context) bool {
	v, _ := ctx.Value(injectAuditFailureKey{}).(bool)
	return v
}

// ComputeAttachmentFingerprint 计算附件引用集合的确定性指纹
func ComputeAttachmentFingerprint(attachments []*SeaOrderSplitAttachmentItem) string {
	if len(attachments) == 0 {
		return "empty"
	}
	items := make([]string, len(attachments))
	for i, a := range attachments {
		items[i] = fmt.Sprintf("%s:%s:%s", a.ID.String(), a.AssetID.String(), a.DocType)
	}
	sort.Strings(items)
	combined := strings.Join(items, ";")
	h := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(h[:])
}
