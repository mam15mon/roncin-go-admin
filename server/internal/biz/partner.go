package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerNotFound        = errors.NotFound(v1.ErrorReason_PARTNER_NOT_FOUND.String(), "往来单位不存在")
	ErrPartnerCodeExists      = errors.Conflict(v1.ErrorReason_PARTNER_CODE_EXISTS.String(), "往来单位编码已存在")
	ErrPartnerInvalidType     = errors.BadRequest(v1.ErrorReason_PARTNER_INVALID_TYPE.String(), "往来单位类型不合法")
	ErrPartnerInvalidArgument = errors.BadRequest(v1.ErrorReason_PARTNER_INVALID_ARGUMENT.String(), "往来单位字段不合法")
)

type PartnerType string

const (
	PartnerTypeCustomer PartnerType = "customer"
	PartnerTypeSupplier PartnerType = "supplier"
	PartnerTypeBoth     PartnerType = "both"
)

func (t PartnerType) Valid() bool {
	return t == PartnerTypeCustomer || t == PartnerTypeSupplier || t == PartnerTypeBoth
}

type Partner struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Name           string
	Type           PartnerType
	ContactName    string
	Phone          string
	Email          string
	Address        string
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PartnerListOptions struct {
	Page     int
	PageSize int
	Keyword  string
	Type     PartnerType
	Enabled  *bool
}

type PartnerList struct {
	Items    []*Partner
	Total    int
	Page     int
	PageSize int
}

type PartnerRepo interface {
	List(context.Context, uuid.UUID, PartnerListOptions) (*PartnerList, error)
	Create(context.Context, uuid.UUID, *Partner) (*Partner, error)
	Update(context.Context, uuid.UUID, uuid.UUID, *Partner) (*Partner, error)
}

type PartnerUsecase struct {
	repo  PartnerRepo
	audit AuditRepo
}

func NewPartnerUsecase(repo PartnerRepo, audit AuditRepo) *PartnerUsecase {
	return &PartnerUsecase{repo: repo, audit: audit}
}

func (uc *PartnerUsecase) List(ctx context.Context, organizationID uuid.UUID, options PartnerListOptions) (*PartnerList, error) {
	if options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 {
		return nil, ErrPartnerInvalidArgument
	}
	if options.Type != "" && !options.Type.Valid() {
		return nil, ErrPartnerInvalidType
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *PartnerUsecase) Create(ctx context.Context, organizationID, userID uuid.UUID, input *Partner) (*Partner, error) {
	normalized, err := normalizePartner(input)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.create",
		Result:         "success",
		Details:        map[string]string{"partner.id": created.ID.String(), "partner.code": created.Code},
	}); err != nil {
		return nil, fmt.Errorf("write partner create audit: %w", err)
	}
	return created, nil
}

func (uc *PartnerUsecase) Update(ctx context.Context, organizationID, userID, id uuid.UUID, input *Partner) (*Partner, error) {
	normalized, err := normalizePartner(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{
		OrganizationID: &organizationID,
		UserID:         &userID,
		Action:         "partner.update",
		Result:         "success",
		Details:        map[string]string{"partner.id": updated.ID.String(), "partner.code": updated.Code},
	}); err != nil {
		return nil, fmt.Errorf("write partner update audit: %w", err)
	}
	return updated, nil
}

func normalizePartner(input *Partner) (*Partner, error) {
	if input == nil {
		return nil, ErrPartnerInvalidArgument
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	output.Type = PartnerType(strings.ToLower(strings.TrimSpace(string(output.Type))))
	output.ContactName = strings.TrimSpace(output.ContactName)
	output.Phone = strings.TrimSpace(output.Phone)
	output.Email = strings.TrimSpace(output.Email)
	output.Address = strings.TrimSpace(output.Address)
	if output.Code == "" || output.Name == "" {
		return nil, ErrPartnerInvalidArgument
	}
	if !output.Type.Valid() {
		return nil, ErrPartnerInvalidType
	}
	return &output, nil
}
