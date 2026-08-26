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
	ErrPartnerAccountNotFound        = errors.NotFound("PARTNER_ACCOUNT_NOT_FOUND", "结算账户不存在")
	ErrPartnerAccountInvalidArgument = errors.BadRequest("PARTNER_ACCOUNT_INVALID_ARGUMENT", "结算账户字段不合法")
	ErrPartnerAccountDefaultConflict = errors.Conflict("PARTNER_ACCOUNT_DEFAULT_CONFLICT", "角色只能有一个默认结算账户")
)

type PartnerAccountStatus string

const (
	PartnerAccountActive   PartnerAccountStatus = "active"
	PartnerAccountInactive PartnerAccountStatus = "inactive"
)

func (s PartnerAccountStatus) Valid() bool {
	return s == PartnerAccountActive || s == PartnerAccountInactive
}

type PartnerAccount struct {
	ID            uuid.UUID
	PartnerRoleID uuid.UUID
	AccountType   string
	Currency      string
	BankName      string
	BankAccount   string
	SwiftCode     string
	IsDefault     bool
	Status        PartnerAccountStatus
	Remark        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type PartnerAccountRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID, *bool) ([]*PartnerAccount, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *PartnerAccount) (*PartnerAccount, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *PartnerAccount) (*PartnerAccount, error)
}

type PartnerAccountUsecase struct {
	repo  PartnerAccountRepo
	audit AuditRepo
}

func NewPartnerAccountUsecase(repo PartnerAccountRepo, audit AuditRepo) *PartnerAccountUsecase {
	return &PartnerAccountUsecase{repo: repo, audit: audit}
}

func (uc *PartnerAccountUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID, enabled *bool) ([]*PartnerAccount, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerAccountInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID, enabled)
}

func (uc *PartnerAccountUsecase) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, input *PartnerAccount) (*PartnerAccount, error) {
	normalized, err := normalizePartnerAccount(input)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, partnerID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.account.create", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"account.id": created.ID.String(), "partner.id": partnerID.String()}}); err != nil {
		return nil, fmt.Errorf("write partner account create audit: %w", err)
	}
	return created, nil
}

func (uc *PartnerAccountUsecase) Update(ctx context.Context, organizationID, actorID, partnerID, id uuid.UUID, input *PartnerAccount) (*PartnerAccount, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerAccountNotFound
	}
	normalized, err := normalizePartnerAccount(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, partnerID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.account.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"account.id": updated.ID.String(), "partner.id": partnerID.String()}}); err != nil {
		return nil, fmt.Errorf("write partner account update audit: %w", err)
	}
	return updated, nil
}

func normalizePartnerAccount(input *PartnerAccount) (*PartnerAccount, error) {
	if input == nil {
		return nil, ErrPartnerAccountInvalidArgument
	}
	output := *input
	output.Currency = strings.ToUpper(strings.TrimSpace(output.Currency))
	output.BankName = strings.TrimSpace(output.BankName)
	output.BankAccount = strings.TrimSpace(output.BankAccount)
	output.SwiftCode = strings.ToUpper(strings.TrimSpace(output.SwiftCode))
	output.Remark = strings.TrimSpace(output.Remark)
	if len(output.Currency) != 3 || !output.Status.Valid() || utf8.RuneCountInString(output.BankName) > 200 || utf8.RuneCountInString(output.BankAccount) > 100 || utf8.RuneCountInString(output.SwiftCode) > 32 || utf8.RuneCountInString(output.Remark) > 500 {
		return nil, ErrPartnerAccountInvalidArgument
	}
	if output.IsDefault && output.Status != PartnerAccountActive {
		return nil, ErrPartnerAccountInvalidArgument
	}
	return &output, nil
}
