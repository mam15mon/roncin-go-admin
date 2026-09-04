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
	SeaDocumentEventTypeAmendment SeaDocumentEventType = "AMENDMENT"
	SeaDocumentEventTypeVoid      SeaDocumentEventType = "VOID"
	SeaDocumentEventTypeSwitch    SeaDocumentEventType = "SWITCH"
)

var (
	ErrSeaDocumentAmendmentEmpty           = errors.BadRequest("SEA_DOCUMENT_AMENDMENT_EMPTY", "改单内容与当前不可变版本没有差异")
	ErrSeaDocumentChangeBlocked            = errors.Conflict("SEA_DOCUMENT_CHANGE_BLOCKED", "单证存在不可自动调整的下游事实，当前操作已阻断")
	ErrSeaDocumentVoided                   = errors.Conflict("SEA_DOCUMENT_VOIDED", "单证已作废，不能再次修改")
	ErrSeaHouseBillSwitchConflict          = errors.Conflict("SEA_HOUSE_BILL_SWITCH_CONFLICT", "HBL 已被替代或版本已变化，请刷新后重试")
	ErrSeaHouseBillSwitchDownstreamBlocked = errors.Conflict("SEA_HOUSE_BILL_SWITCH_DOWNSTREAM_BLOCKED", "HBL 存在不可自动调整的下游事实，不能执行 Switch B/L")
	ErrSeaDocumentVersionNotFound          = errors.NotFound("SEA_DOCUMENT_VERSION_NOT_FOUND", "单证不可变版本不存在")
)

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
	OldHouseBillID    *uuid.UUID
	OldHouseNo        *string
	NewHouseBillID    *uuid.UUID
	NewHouseNo        *string
	ChainID           *uuid.UUID
	Sequence          *int
	Reason            string
	ImpactSummary     *string
	SurrenderInfo     *string
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
}

type SeaHouseBillSwitchCommand struct {
	OrderID                  uuid.UUID
	OldHouseBillID           uuid.UUID
	ExpectedOrderVersion     uint64
	ExpectedHouseBillVersion uint64
	ExpectedCurrentVersionID uuid.UUID
	Reason                   string
	SurrenderInfo            *string
	IdempotencyKey           string
	NewHouseBill             *SeaHouseBillInput
}

type SeaDocumentChangePreview struct {
	BaseVersion *SeaDocumentVersion
	Differences []*SeaDocumentFieldDifference
	Impacts     []*SeaDocumentDownstreamImpact
	Executable  bool
}

type SeaHouseBillSwitchResult struct {
	Event        *SeaDocumentEvent
	NewHouseBill *SeaHouseBill
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
	PreviewSwitch(context.Context, uuid.UUID, *SeaHouseBillSwitchCommand) (*SeaDocumentChangePreview, error)
	ExecuteSwitch(context.Context, uuid.UUID, uuid.UUID, *SeaHouseBillSwitchCommand, *AuditEvent) (*SeaHouseBillSwitchResult, error)
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
	return &out, nil
}

func validateSwitchCommand(input *SeaHouseBillSwitchCommand, execute bool) (*SeaHouseBillSwitchCommand, error) {
	if input == nil || input.OrderID == uuid.Nil || input.OldHouseBillID == uuid.Nil || input.ExpectedOrderVersion == 0 || input.ExpectedHouseBillVersion == 0 || input.ExpectedCurrentVersionID == uuid.Nil || input.NewHouseBill == nil {
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
	hb, err := ValidateSeaHouseBillInput(input.NewHouseBill)
	if err != nil {
		return nil, err
	}
	out := *input
	out.Reason, out.IdempotencyKey, out.NewHouseBill = reason, key, hb
	if input.SurrenderInfo != nil {
		v := strings.TrimSpace(*input.SurrenderInfo)
		if utf8.RuneCountInString(v) > 500 || containsControl(v) {
			return nil, ErrSeaDocumentInvalidArgument
		}
		if v == "" {
			out.SurrenderInfo = nil
		} else {
			out.SurrenderInfo = &v
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

func (uc *SeaDocumentChangeUsecase) PreviewSwitch(ctx context.Context, orgID uuid.UUID, input *SeaHouseBillSwitchCommand) (*SeaDocumentChangePreview, error) {
	validated, err := validateSwitchCommand(input, false)
	if err != nil {
		return nil, err
	}
	return uc.repo.PreviewSwitch(ctx, orgID, validated)
}

func (uc *SeaDocumentChangeUsecase) ExecuteSwitch(ctx context.Context, orgID, actorID uuid.UUID, input *SeaHouseBillSwitchCommand, audit *AuditEvent) (*SeaHouseBillSwitchResult, error) {
	validated, err := validateSwitchCommand(input, true)
	if err != nil {
		return nil, err
	}
	if orgID == uuid.Nil || actorID == uuid.Nil {
		return nil, ErrSeaDocumentInvalidArgument
	}
	if err := validateAuditEvent(audit, orgID, actorID); err != nil {
		return nil, err
	}
	return uc.repo.ExecuteSwitch(ctx, orgID, actorID, validated, audit)
}
