package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerContractNotFound        = errors.NotFound("PARTNER_CONTRACT_NOT_FOUND", "合同不存在")
	ErrPartnerContractNoExists        = errors.Conflict("PARTNER_CONTRACT_NO_EXISTS", "合同编号已存在")
	ErrPartnerContractInvalidArgument = errors.BadRequest("PARTNER_CONTRACT_INVALID_ARGUMENT", "合同字段不合法")
	ErrPartnerContractStatusConflict  = errors.Conflict("PARTNER_CONTRACT_STATUS_CONFLICT", "合同状态不允许该变更")
)

type PartnerContractStatus string

const (
	PartnerContractPending    PartnerContractStatus = "pending"
	PartnerContractActive     PartnerContractStatus = "active"
	PartnerContractExpired    PartnerContractStatus = "expired"
	PartnerContractTerminated PartnerContractStatus = "terminated"
)

func (s PartnerContractStatus) Valid() bool {
	return s == PartnerContractPending || s == PartnerContractActive || s == PartnerContractExpired || s == PartnerContractTerminated
}

type PartnerContract struct {
	ID                uuid.UUID
	PartnerID         uuid.UUID
	ContractNo        string
	Name              string
	Status            PartnerContractStatus
	StartDate         time.Time
	EndDate           time.Time
	PaymentTerms      string
	DisputeResolution string
	OtherNotes        string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type PartnerContractRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID, *PartnerContractStatus) ([]*PartnerContract, error)
	Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*PartnerContract, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *PartnerContract) (*PartnerContract, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, PartnerContractStatus, *PartnerContract) (*PartnerContract, error)
}

type PartnerContractUsecase struct {
	repo  PartnerContractRepo
	audit AuditRepo
}

func NewPartnerContractUsecase(repo PartnerContractRepo, audit AuditRepo) *PartnerContractUsecase {
	return &PartnerContractUsecase{repo: repo, audit: audit}
}

func (uc *PartnerContractUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID, status *PartnerContractStatus) ([]*PartnerContract, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerContractInvalidArgument
	}
	if status != nil && !status.Valid() {
		return nil, ErrPartnerContractInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID, status)
}

func (uc *PartnerContractUsecase) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, input *PartnerContract) (*PartnerContract, error) {
	normalized, err := normalizePartnerContract(input, true, nil)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, partnerID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.contract.create", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"contract.id": created.ID.String(), "partner.id": partnerID.String()}}); err != nil {
		return nil, fmt.Errorf("write partner contract create audit: %w", err)
	}
	return created, nil
}

func (uc *PartnerContractUsecase) Update(ctx context.Context, organizationID, actorID, partnerID, id uuid.UUID, input *PartnerContract) (*PartnerContract, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerContractNotFound
	}
	existing, err := uc.repo.Get(ctx, organizationID, partnerID, id)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizePartnerContract(input, false, &existing.Status)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, partnerID, id, existing.Status, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.contract.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"contract.id": updated.ID.String(), "partner.id": partnerID.String()}}); err != nil {
		return nil, fmt.Errorf("write partner contract update audit: %w", err)
	}
	return updated, nil
}

func normalizePartnerContract(input *PartnerContract, creating bool, previous *PartnerContractStatus) (*PartnerContract, error) {
	if input == nil {
		return nil, ErrPartnerContractInvalidArgument
	}
	output := *input
	if creating {
		output.ContractNo = strings.TrimSpace(output.ContractNo)
	}
	output.Name = strings.TrimSpace(output.Name)
	output.PaymentTerms = strings.TrimSpace(output.PaymentTerms)
	output.DisputeResolution = strings.TrimSpace(output.DisputeResolution)
	output.OtherNotes = strings.TrimSpace(output.OtherNotes)
	if (creating && (output.ContractNo == "" || utf8.RuneCountInString(output.ContractNo) > 100)) || output.Name == "" || !output.Status.Valid() || output.StartDate.IsZero() || output.EndDate.IsZero() || !output.StartDate.Before(output.EndDate) || utf8.RuneCountInString(output.Name) > 200 || utf8.RuneCountInString(output.PaymentTerms) > 2000 || utf8.RuneCountInString(output.DisputeResolution) > 2000 || utf8.RuneCountInString(output.OtherNotes) > 2000 {
		return nil, ErrPartnerContractInvalidArgument
	}
	if creating && output.Status != PartnerContractPending && output.Status != PartnerContractActive {
		return nil, ErrPartnerContractStatusConflict
	}
	if previous != nil && !validContractTransition(*previous, output.Status) {
		return nil, ErrPartnerContractStatusConflict
	}
	return &output, nil
}

func validContractTransition(from, to PartnerContractStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case PartnerContractPending:
		return to == PartnerContractActive || to == PartnerContractTerminated
	case PartnerContractActive:
		return to == PartnerContractExpired || to == PartnerContractTerminated
	default:
		return false
	}
}
