package biz

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrMasterDataInvalidArgument = errors.BadRequest("MASTER_DATA_INVALID_ARGUMENT", "主数据字段不合法")
	ErrMasterDataNotFound        = errors.NotFound("MASTER_DATA_NOT_FOUND", "主数据不存在")
	ErrMasterDataCodeExists      = errors.Conflict("MASTER_DATA_CODE_EXISTS", "主数据编码已存在")
	ErrMasterDataInvalidKind     = errors.BadRequest("MASTER_DATA_INVALID_KIND", "主数据类型不合法")
)

type MasterDataKind string

const (
	MasterDataKindCurrency      MasterDataKind = "currency"
	MasterDataKindCountry       MasterDataKind = "country"
	MasterDataKindRegion        MasterDataKind = "region"
	MasterDataKindPort          MasterDataKind = "port"
	MasterDataKindAirport       MasterDataKind = "airport"
	MasterDataKindCarrier       MasterDataKind = "carrier"
	MasterDataKindContainerSpec MasterDataKind = "container_spec"
	MasterDataKindServiceType   MasterDataKind = "service_type"
	MasterDataKindCargoCategory MasterDataKind = "cargo_category"
	MasterDataKindAbnormalCase  MasterDataKind = "abnormal_case"
)

func (kind MasterDataKind) Valid() bool {
	switch kind {
	case MasterDataKindCurrency, MasterDataKindCountry, MasterDataKindRegion, MasterDataKindPort, MasterDataKindAirport, MasterDataKindCarrier, MasterDataKindContainerSpec, MasterDataKindServiceType, MasterDataKindCargoCategory, MasterDataKindAbnormalCase:
		return true
	default:
		return false
	}
}

type MasterDataItem struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Kind           MasterDataKind
	Code           string
	Name           string
	NameEN         *string
	ParentCode     *string
	TransportMode  *string
	TEUFactor      *string
	Source         string
	SortOrder      int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MasterDataListOptions struct {
	Page     int
	PageSize int
	Kind     MasterDataKind
	Keyword  string
	Enabled  *bool
}

type MasterDataList struct {
	Items    []*MasterDataItem
	Total    int
	Page     int
	PageSize int
}

type MasterDataImportMode string

const (
	MasterDataImportModeCreateOnly MasterDataImportMode = "create_only"
	MasterDataImportModeUpsert     MasterDataImportMode = "upsert"
)

func (mode MasterDataImportMode) Valid() bool {
	return mode == MasterDataImportModeCreateOnly || mode == MasterDataImportModeUpsert
}

type MasterDataImportInput struct {
	Kind   MasterDataKind
	Source string
	Mode   MasterDataImportMode
	Items  []*MasterDataItem
}

type MasterDataImportResult struct {
	Items   []*MasterDataItem
	Created int
	Updated int
}

type MasterDataRepo interface {
	List(context.Context, uuid.UUID, MasterDataListOptions) (*MasterDataList, error)
	ListEnabled(context.Context, uuid.UUID) ([]*MasterDataItem, error)
	Create(context.Context, uuid.UUID, *MasterDataItem) (*MasterDataItem, error)
	Update(context.Context, uuid.UUID, uuid.UUID, *MasterDataItem) (*MasterDataItem, error)
	Import(context.Context, uuid.UUID, MasterDataImportMode, []*MasterDataItem) (*MasterDataImportResult, error)
}

type MasterDataUsecase struct {
	repo  MasterDataRepo
	audit AuditRepo
}

func NewMasterDataUsecase(repo MasterDataRepo, audit AuditRepo) *MasterDataUsecase {
	return &MasterDataUsecase{repo: repo, audit: audit}
}

func (uc *MasterDataUsecase) List(ctx context.Context, organizationID uuid.UUID, options MasterDataListOptions) (*MasterDataList, error) {
	if organizationID == uuid.Nil || options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 {
		return nil, ErrMasterDataInvalidArgument
	}
	if options.Kind != "" && !options.Kind.Valid() {
		return nil, ErrMasterDataInvalidKind
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *MasterDataUsecase) ListOptions(ctx context.Context, organizationID uuid.UUID) ([]*MasterDataItem, error) {
	if organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	return uc.repo.ListEnabled(ctx, organizationID)
}

func (uc *MasterDataUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *MasterDataItem) (*MasterDataItem, error) {
	if organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	normalized, err := normalizeMasterDataItem(input, true)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "master_data.create", Result: "success", Details: map[string]string{"master_data.id": created.ID.String(), "master_data.kind": string(created.Kind), "master_data.code": created.Code}}); err != nil {
		return nil, fmt.Errorf("write master data create audit: %w", err)
	}
	return created, nil
}

func (uc *MasterDataUsecase) Import(ctx context.Context, organizationID, actorID uuid.UUID, input MasterDataImportInput) (*MasterDataImportResult, error) {
	input.Source = strings.TrimSpace(input.Source)
	if organizationID == uuid.Nil || !input.Kind.Valid() || !input.Mode.Valid() || input.Source == "" || utf8.RuneCountInString(input.Source) > 100 || len(input.Items) == 0 || len(input.Items) > 500 {
		return nil, ErrMasterDataInvalidArgument
	}
	items := make([]*MasterDataItem, 0, len(input.Items))
	seen := make(map[string]struct{}, len(input.Items))
	for _, item := range input.Items {
		if item == nil {
			return nil, ErrMasterDataInvalidArgument
		}
		copyItem := *item
		copyItem.Kind = input.Kind
		copyItem.Source = input.Source
		normalized, err := normalizeMasterDataItem(&copyItem, true)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized.Code]; exists {
			return nil, ErrMasterDataInvalidArgument
		}
		seen[normalized.Code] = struct{}{}
		items = append(items, normalized)
	}
	result, err := uc.repo.Import(ctx, organizationID, input.Mode, items)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "master_data.import", Result: "success", Details: map[string]string{"master_data.kind": string(input.Kind), "source": input.Source, "mode": string(input.Mode), "created": fmt.Sprintf("%d", result.Created), "updated": fmt.Sprintf("%d", result.Updated)}}); err != nil {
		return nil, fmt.Errorf("write master data import audit: %w", err)
	}
	return result, nil
}

func (uc *MasterDataUsecase) Update(ctx context.Context, organizationID, actorID, id uuid.UUID, input *MasterDataItem) (*MasterDataItem, error) {
	if id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	normalized, err := normalizeMasterDataItem(input, false)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "master_data.update", Result: "success", Details: map[string]string{"master_data.id": updated.ID.String(), "master_data.kind": string(updated.Kind), "master_data.code": updated.Code}}); err != nil {
		return nil, fmt.Errorf("write master data update audit: %w", err)
	}
	return updated, nil
}

func normalizeMasterDataItem(input *MasterDataItem, creating bool) (*MasterDataItem, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	output.Source = strings.TrimSpace(output.Source)
	if output.Source == "" {
		output.Source = "manual"
	}
	output.NameEN = normalizedOptionalString(output.NameEN)
	output.ParentCode = normalizedUpperOptionalString(output.ParentCode)
	output.TransportMode = normalizedUpperOptionalString(output.TransportMode)
	output.TEUFactor = normalizedOptionalString(output.TEUFactor)
	if !output.Kind.Valid() || output.Name == "" || utf8.RuneCountInString(output.Name) > 200 || output.SortOrder < 0 || utf8.RuneCountInString(output.Source) > 100 {
		return nil, ErrMasterDataInvalidArgument
	}
	if creating && (output.Code == "" || utf8.RuneCountInString(output.Code) > 64) {
		return nil, ErrMasterDataInvalidArgument
	}
	if optionalStringTooLong(output.NameEN, 200) || optionalStringTooLong(output.ParentCode, 64) || optionalStringTooLong(output.TransportMode, 32) || optionalStringTooLong(output.TEUFactor, 32) {
		return nil, ErrMasterDataInvalidArgument
	}
	if output.ParentCode != nil && output.Kind != MasterDataKindRegion && output.Kind != MasterDataKindPort && output.Kind != MasterDataKindAirport {
		return nil, ErrMasterDataInvalidArgument
	}
	if output.TransportMode != nil && output.Kind != MasterDataKindCarrier {
		return nil, ErrMasterDataInvalidArgument
	}
	if output.TEUFactor != nil {
		if output.Kind != MasterDataKindContainerSpec {
			return nil, ErrMasterDataInvalidArgument
		}
		value, ok := new(big.Rat).SetString(*output.TEUFactor)
		if !ok || value.Sign() <= 0 {
			return nil, ErrMasterDataInvalidArgument
		}
	}
	return &output, nil
}

func optionalStringTooLong(value *string, maximum int) bool {
	return value != nil && utf8.RuneCountInString(*value) > maximum
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizedUpperOptionalString(value *string) *string {
	normalized := normalizedOptionalString(value)
	if normalized == nil {
		return nil
	}
	upper := strings.ToUpper(*normalized)
	return &upper
}

var _ MasterDataRepo = (MasterDataRepo)(nil)
