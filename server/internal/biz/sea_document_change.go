package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

type SeaDocumentType string

const (
	SeaDocumentTypeMasterBill SeaDocumentType = "MASTER_BILL"
	SeaDocumentTypeHouseBill  SeaDocumentType = "HOUSE_BILL"
)

func (t SeaDocumentType) Valid() bool {
	return t == SeaDocumentTypeMasterBill || t == SeaDocumentTypeHouseBill
}

type SeaDocumentEventType string

const (
	SeaDocumentEventTypeAmendment  SeaDocumentEventType = "AMENDMENT"
	SeaDocumentEventTypeVoid       SeaDocumentEventType = "VOID"
	SeaDocumentEventTypeModeChange SeaDocumentEventType = "MODE_CHANGE"
)

var (
	ErrSeaDocumentAmendmentEmpty     = errors.BadRequest("SEA_DOCUMENT_AMENDMENT_EMPTY", "改单内容与当前不可变版本没有差异")
	ErrSeaDocumentChangeBlocked      = errors.Conflict("SEA_DOCUMENT_CHANGE_BLOCKED", "单证存在不可自动调整的下游事实，当前操作已阻断")
	ErrSeaDocumentVoided             = errors.Conflict("SEA_DOCUMENT_VOIDED", "单证已作废，不能再次修改")
	ErrSeaDocumentVersionNotFound    = errors.NotFound("SEA_DOCUMENT_VERSION_NOT_FOUND", "单证不可变版本不存在")
	ErrSeaDocumentModeChangeConflict = errors.Conflict("SEA_DOCUMENT_MODE_CHANGE_CONFLICT", "单证模式或版本已变化，请刷新后重试")
)

// SeaExternalConfirmation 是承运方/船代对一次正式变更的不可变确认事实。
type SeaExternalConfirmation struct {
	ConfirmedByParty           string
	ConfirmedAt                time.Time
	ConfirmationNote           string
	ConfirmationAttachmentID   *uuid.UUID
	ConfirmationAttachmentName string
}

type SeaDocumentVersion struct {
	ID                   uuid.UUID
	DocumentType         SeaDocumentType
	DocumentID           uuid.UUID
	OrderID              uuid.UUID
	MasterBillID         uuid.UUID
	VersionNo            uint64
	SourceEntityVersion  uint64
	DocumentNo           string
	NormalizedDocumentNo string
	Status               string
	Source               string
	Reason               *string
	ShippingLineID       *uuid.UUID
	ShippingLineName     string
	IssuerPartnerID      *uuid.UUID
	IssuerOrganizationID *uuid.UUID
	IssuerSource         SeaHouseBillIssuerSource
	TransportExecutionID *uuid.UUID
	VesselName           *string
	VoyageNo             *string
	ETD                  *time.Time
	ETA                  *time.Time
	Note                 *string
	Content              *SeaBillContent
	CreatedBy            *uuid.UUID
	CreatedAt            time.Time
	Confirmation         *SeaExternalConfirmation
}

type SeaDocumentFieldDifference struct {
	Field       string
	Label       string
	BeforeValue string
	AfterValue  string
}

type SeaDocumentDownstreamImpact struct {
	FactType        string
	ReferenceID     string
	ReferenceNo     string
	Message         string
	BlocksExecution bool
}

type SeaDocumentEvent struct {
	ID                uuid.UUID
	EventType         SeaDocumentEventType
	DocumentType      SeaDocumentType
	DocumentID        *uuid.UUID
	DocumentNo        *string
	PreviousVersionID *uuid.UUID
	ResultVersionID   *uuid.UUID
	Reason            string
	ImpactSummary     *string
	PreviousMode      *SeaDocumentStructure
	TargetMode        *SeaDocumentStructure
	Confirmation      *SeaExternalConfirmation
	CreatedBy         *uuid.UUID
	CreatedAt         time.Time
}

type SeaDocumentAmendmentInput struct {
	MasterBillContent *SeaBillContent
	HouseBill         *SeaHouseBillInput
}

type SeaDocumentAmendmentCommand struct {
	OrderID                  uuid.UUID
	DocumentType             SeaDocumentType
	DocumentID               uuid.UUID
	ExpectedOrderVersion     uint64
	ExpectedDocumentVersion  uint64
	ExpectedCurrentVersionID uuid.UUID
	Reason                   string
	IdempotencyKey           string
	Input                    *SeaDocumentAmendmentInput
	Confirmation             *SeaExternalConfirmation
}

type SeaDocumentVoidCommand struct {
	OrderID                  uuid.UUID
	DocumentType             SeaDocumentType
	DocumentID               uuid.UUID
	ExpectedOrderVersion     uint64
	ExpectedDocumentVersion  uint64
	ExpectedCurrentVersionID uuid.UUID
	Reason                   string
	IdempotencyKey           string
	Confirmation             *SeaExternalConfirmation
}

type SeaDocumentModeChangeCommand struct {
	OrderID                  uuid.UUID
	ExpectedOrderVersion     uint64
	ExpectedLinkVersion      uint64
	ExpectedHouseBillVersion *uint64
	ExpectedCurrentVersionID *uuid.UUID
	TargetMode               SeaDocumentStructure
	NewHouseBill             *SeaHouseBillInput
	Reason                   string
	Confirmation             *SeaExternalConfirmation
	IdempotencyKey           string
}

type SeaDocumentChangePreview struct {
	BaseVersion *SeaDocumentVersion
	Differences []*SeaDocumentFieldDifference
	Impacts     []*SeaDocumentDownstreamImpact
	Executable  bool
}

type SeaDocumentChangeRepo interface {
	ListMasterBillVersions(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*SeaDocumentVersion, int, error)
	ListHouseBillVersions(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int, int) ([]*SeaDocumentVersion, int, error)
	GetDocumentVersion(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, SeaDocumentType) (*SeaDocumentVersion, error)
	ListDocumentEvents(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*SeaDocumentEvent, int, error)
	PreviewAmendment(context.Context, uuid.UUID, *SeaDocumentAmendmentCommand) (*SeaDocumentChangePreview, error)
	ExecuteAmendment(context.Context, uuid.UUID, uuid.UUID, *SeaDocumentAmendmentCommand, *AuditEvent) (*SeaDocumentVersion, error)
	PreviewVoid(context.Context, uuid.UUID, *SeaDocumentVoidCommand) (*SeaDocumentChangePreview, error)
	ExecuteVoid(context.Context, uuid.UUID, uuid.UUID, *SeaDocumentVoidCommand, *AuditEvent) (*SeaDocumentEvent, error)
	PreviewModeChange(context.Context, uuid.UUID, *SeaDocumentModeChangeCommand) (*SeaDocumentChangePreview, error)
	ExecuteModeChange(context.Context, uuid.UUID, uuid.UUID, *SeaDocumentModeChangeCommand, *AuditEvent) error
}

type SeaDocumentChangeUsecase struct{ repo SeaDocumentChangeRepo }

func NewSeaDocumentChangeUsecase(repo SeaDocumentChangeRepo) *SeaDocumentChangeUsecase {
	return &SeaDocumentChangeUsecase{repo: repo}
}

func (uc *SeaDocumentChangeUsecase) ListMasterBillVersions(ctx context.Context, orgID, orderID uuid.UUID, page, pageSize int) ([]*SeaDocumentVersion, int, error) {
	if orgID == uuid.Nil || orderID == uuid.Nil || !ValidListPagination(page, pageSize) {
		return nil, 0, ErrSeaDocumentInvalidArgument
	}
	return uc.repo.ListMasterBillVersions(ctx, orgID, orderID, page, pageSize)
}

func (uc *SeaDocumentChangeUsecase) ListHouseBillVersions(ctx context.Context, orgID, orderID, houseBillID uuid.UUID, page, pageSize int) ([]*SeaDocumentVersion, int, error) {
	if orgID == uuid.Nil || orderID == uuid.Nil || houseBillID == uuid.Nil || !ValidListPagination(page, pageSize) {
		return nil, 0, ErrSeaDocumentInvalidArgument
	}
	return uc.repo.ListHouseBillVersions(ctx, orgID, orderID, houseBillID, page, pageSize)
}

func (uc *SeaDocumentChangeUsecase) GetDocumentVersion(ctx context.Context, orgID, orderID, versionID uuid.UUID, documentType SeaDocumentType) (*SeaDocumentVersion, error) {
	if orgID == uuid.Nil || orderID == uuid.Nil || versionID == uuid.Nil || !documentType.Valid() {
		return nil, ErrSeaDocumentInvalidArgument
	}
	return uc.repo.GetDocumentVersion(ctx, orgID, orderID, versionID, documentType)
}

func (uc *SeaDocumentChangeUsecase) ListDocumentEvents(ctx context.Context, orgID, orderID uuid.UUID, page, pageSize int) ([]*SeaDocumentEvent, int, error) {
	if orgID == uuid.Nil || orderID == uuid.Nil || !ValidListPagination(page, pageSize) {
		return nil, 0, ErrSeaDocumentInvalidArgument
	}
	return uc.repo.ListDocumentEvents(ctx, orgID, orderID, page, pageSize)
}

func normalizeRequiredChangeText(value string, max int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > max || containsControl(value) {
		return "", ErrSeaDocumentInvalidArgument
	}
	return value, nil
}

func ValidateSeaExternalConfirmation(input *SeaExternalConfirmation) (*SeaExternalConfirmation, error) {
	if input == nil || input.ConfirmedAt.IsZero() {
		return nil, ErrSeaDocumentInvalidArgument
	}
	party, err := normalizeRequiredChangeText(input.ConfirmedByParty, 128)
	if err != nil {
		return nil, err
	}
	note, err := normalizeRequiredChangeText(input.ConfirmationNote, 500)
	if err != nil {
		return nil, err
	}
	out := *input
	out.ConfirmedByParty = party
	out.ConfirmationNote = note
	if out.ConfirmationAttachmentID != nil && *out.ConfirmationAttachmentID == uuid.Nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	return &out, nil
}

func validateAmendmentCommand(input *SeaDocumentAmendmentCommand, execute bool) (*SeaDocumentAmendmentCommand, error) {
	if input == nil || input.OrderID == uuid.Nil || input.DocumentID == uuid.Nil || !input.DocumentType.Valid() || input.ExpectedOrderVersion == 0 || input.ExpectedDocumentVersion == 0 || input.ExpectedCurrentVersionID == uuid.Nil || input.Input == nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	reason, err := normalizeRequiredChangeText(input.Reason, 500)
	if err != nil {
		return nil, err
	}
	key := ""
	if execute {
		key, err = normalizeRequiredChangeText(input.IdempotencyKey, 128)
		if err != nil {
			return nil, err
		}
	}
	out := *input
	out.Reason, out.IdempotencyKey = reason, key
	if execute {
		out.Confirmation, err = ValidateSeaExternalConfirmation(input.Confirmation)
		if err != nil {
			return nil, err
		}
	}
	switch input.DocumentType {
	case SeaDocumentTypeMasterBill:
		if input.Input.MasterBillContent == nil || input.Input.HouseBill != nil {
			return nil, ErrSeaDocumentInvalidArgument
		}
		content, err := ValidateSeaBillContent(input.Input.MasterBillContent)
		if err != nil {
			return nil, err
		}
		out.Input = &SeaDocumentAmendmentInput{MasterBillContent: content}
	case SeaDocumentTypeHouseBill:
		if input.Input.HouseBill == nil || input.Input.MasterBillContent != nil {
			return nil, ErrSeaDocumentInvalidArgument
		}
		hb, err := ValidateSeaHouseBillInput(input.Input.HouseBill)
		if err != nil {
			return nil, err
		}
		out.Input = &SeaDocumentAmendmentInput{HouseBill: hb}
	}
	return &out, nil
}

func validateVoidCommand(input *SeaDocumentVoidCommand, execute bool) (*SeaDocumentVoidCommand, error) {
	if input == nil || input.OrderID == uuid.Nil || input.DocumentID == uuid.Nil || !input.DocumentType.Valid() || input.ExpectedOrderVersion == 0 || input.ExpectedDocumentVersion == 0 || input.ExpectedCurrentVersionID == uuid.Nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	reason, err := normalizeRequiredChangeText(input.Reason, 500)
	if err != nil {
		return nil, err
	}
	key := ""
	if execute {
		key, err = normalizeRequiredChangeText(input.IdempotencyKey, 128)
		if err != nil {
			return nil, err
		}
	}
	out := *input
	out.Reason, out.IdempotencyKey = reason, key
	if execute {
		out.Confirmation, err = ValidateSeaExternalConfirmation(input.Confirmation)
		if err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func validateModeChangeCommand(input *SeaDocumentModeChangeCommand, execute bool) (*SeaDocumentModeChangeCommand, error) {
	if input == nil || input.OrderID == uuid.Nil || (input.TargetMode != SeaDocumentStructureHouse && input.TargetMode != SeaDocumentStructureDirect) {
		return nil, ErrSeaDocumentInvalidArgument
	}
	reason, err := normalizeRequiredChangeText(input.Reason, 500)
	if err != nil {
		return nil, err
	}
	key := ""
	if execute {
		key, err = normalizeRequiredChangeText(input.IdempotencyKey, 128)
		if err != nil {
			return nil, err
		}
	}
	out := *input
	out.Reason, out.IdempotencyKey = reason, key
	if input.TargetMode == SeaDocumentStructureHouse {
		if input.NewHouseBill == nil {
			return nil, ErrSeaDocumentInvalidArgument
		}
		out.NewHouseBill, err = ValidateSeaHouseBillInput(input.NewHouseBill)
		if err != nil {
			return nil, err
		}
	} else if input.NewHouseBill != nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	if execute {
		if input.ExpectedOrderVersion == 0 || input.ExpectedLinkVersion == 0 {
			return nil, ErrSeaDocumentInvalidArgument
		}
		out.Confirmation, err = ValidateSeaExternalConfirmation(input.Confirmation)
		if err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (uc *SeaDocumentChangeUsecase) PreviewAmendment(ctx context.Context, orgID uuid.UUID, input *SeaDocumentAmendmentCommand) (*SeaDocumentChangePreview, error) {
	validated, err := validateAmendmentCommand(input, false)
	if err != nil {
		return nil, err
	}
	return uc.repo.PreviewAmendment(ctx, orgID, validated)
}

func (uc *SeaDocumentChangeUsecase) ExecuteAmendment(ctx context.Context, orgID, actorID uuid.UUID, input *SeaDocumentAmendmentCommand, audit *AuditEvent) (*SeaDocumentVersion, error) {
	validated, err := validateAmendmentCommand(input, true)
	if err != nil {
		return nil, err
	}
	if orgID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.ExecuteAmendment(ctx, orgID, actorID, validated, audit)
}

func (uc *SeaDocumentChangeUsecase) PreviewVoid(ctx context.Context, orgID uuid.UUID, input *SeaDocumentVoidCommand) (*SeaDocumentChangePreview, error) {
	validated, err := validateVoidCommand(input, false)
	if err != nil {
		return nil, err
	}
	return uc.repo.PreviewVoid(ctx, orgID, validated)
}

func (uc *SeaDocumentChangeUsecase) ExecuteVoid(ctx context.Context, orgID, actorID uuid.UUID, input *SeaDocumentVoidCommand, audit *AuditEvent) (*SeaDocumentEvent, error) {
	validated, err := validateVoidCommand(input, true)
	if err != nil {
		return nil, err
	}
	if orgID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.ExecuteVoid(ctx, orgID, actorID, validated, audit)
}

func (uc *SeaDocumentChangeUsecase) PreviewModeChange(ctx context.Context, orgID uuid.UUID, input *SeaDocumentModeChangeCommand) (*SeaDocumentChangePreview, error) {
	validated, err := validateModeChangeCommand(input, false)
	if err != nil {
		return nil, err
	}
	return uc.repo.PreviewModeChange(ctx, orgID, validated)
}

func (uc *SeaDocumentChangeUsecase) ExecuteModeChange(ctx context.Context, orgID, actorID uuid.UUID, input *SeaDocumentModeChangeCommand, audit *AuditEvent) error {
	validated, err := validateModeChangeCommand(input, true)
	if err != nil {
		return err
	}
	if orgID == uuid.Nil || actorID == uuid.Nil {
		return ErrSeaDocumentInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return err
	}
	return uc.repo.ExecuteModeChange(ctx, orgID, actorID, validated, audit)
}
