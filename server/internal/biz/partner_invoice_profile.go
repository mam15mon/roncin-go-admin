package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerInvoiceProfileNotFound        = errors.NotFound("PARTNER_INVOICE_PROFILE_NOT_FOUND", "开票抬头不存在")
	ErrPartnerInvoiceProfileInvalidArgument = errors.BadRequest("PARTNER_INVOICE_PROFILE_INVALID_ARGUMENT", "开票抬头字段不合法")
	ErrPartnerInvoiceProfileVersionConflict = errors.Conflict("PARTNER_INVOICE_PROFILE_VERSION_CONFLICT", "开票抬头已被其他操作人修改，请刷新后重试")
	ErrPartnerInvoiceProfileTitleExists     = errors.Conflict("PARTNER_INVOICE_PROFILE_TITLE_EXISTS", "该客户已存在同名开票抬头")
)

type PartnerInvoiceProfile struct {
	ID, OrganizationID, PartnerID          uuid.UUID
	InvoiceTitle, TaxpayerIdentificationNo string
	RegisteredAddress, RegisteredPhone     string
	BankName, BankAccount                  string
	DefaultInvoiceType                     FinanceInvoiceType
	IsDefault, Enabled                     bool
	Version                                uint64
	CreatedAt, UpdatedAt                   time.Time
}

type CreatePartnerInvoiceProfileInput struct {
	PartnerID                              uuid.UUID
	InvoiceTitle, TaxpayerIdentificationNo string
	RegisteredAddress, RegisteredPhone     string
	BankName, BankAccount                  string
	DefaultInvoiceType                     FinanceInvoiceType
	IsDefault                              bool
}

type UpdatePartnerInvoiceProfileInput struct {
	PartnerID, ID                          uuid.UUID
	InvoiceTitle, TaxpayerIdentificationNo string
	RegisteredAddress, RegisteredPhone     string
	BankName, BankAccount                  string
	DefaultInvoiceType                     FinanceInvoiceType
	IsDefault, Enabled                     bool
	ExpectedVersion                        uint64
}

type PartnerInvoiceProfileRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID) ([]*PartnerInvoiceProfile, error)
	Create(context.Context, uuid.UUID, *PartnerInvoiceProfile, *AuditEvent) (*PartnerInvoiceProfile, error)
	Update(context.Context, uuid.UUID, *PartnerInvoiceProfile, uint64, *AuditEvent) (*PartnerInvoiceProfile, error)
}

type PartnerInvoiceProfileUsecase struct{ repo PartnerInvoiceProfileRepo }

func NewPartnerInvoiceProfileUsecase(repo PartnerInvoiceProfileRepo) *PartnerInvoiceProfileUsecase {
	return &PartnerInvoiceProfileUsecase{repo: repo}
}

func (uc *PartnerInvoiceProfileUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID) ([]*PartnerInvoiceProfile, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerInvoiceProfileInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID)
}

func (uc *PartnerInvoiceProfileUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input CreatePartnerInvoiceProfileInput) (*PartnerInvoiceProfile, error) {
	profile := &PartnerInvoiceProfile{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, PartnerID: input.PartnerID,
		InvoiceTitle: strings.TrimSpace(input.InvoiceTitle), TaxpayerIdentificationNo: strings.TrimSpace(input.TaxpayerIdentificationNo),
		RegisteredAddress: strings.TrimSpace(input.RegisteredAddress), RegisteredPhone: strings.TrimSpace(input.RegisteredPhone),
		BankName: strings.TrimSpace(input.BankName), BankAccount: strings.TrimSpace(input.BankAccount),
		DefaultInvoiceType: input.DefaultInvoiceType, IsDefault: input.IsDefault, Enabled: true,
	}
	if organizationID == uuid.Nil || actorID == uuid.Nil || !validPartnerInvoiceProfile(profile) {
		return nil, ErrPartnerInvoiceProfileInvalidArgument
	}
	audit := partnerInvoiceProfileAudit(organizationID, actorID, profile, "partner.invoice_profile.create")
	return uc.repo.Create(ctx, organizationID, profile, audit)
}

func (uc *PartnerInvoiceProfileUsecase) Update(ctx context.Context, organizationID, actorID uuid.UUID, input UpdatePartnerInvoiceProfileInput) (*PartnerInvoiceProfile, error) {
	profile := &PartnerInvoiceProfile{
		ID: input.ID, OrganizationID: organizationID, PartnerID: input.PartnerID,
		InvoiceTitle: strings.TrimSpace(input.InvoiceTitle), TaxpayerIdentificationNo: strings.TrimSpace(input.TaxpayerIdentificationNo),
		RegisteredAddress: strings.TrimSpace(input.RegisteredAddress), RegisteredPhone: strings.TrimSpace(input.RegisteredPhone),
		BankName: strings.TrimSpace(input.BankName), BankAccount: strings.TrimSpace(input.BankAccount),
		DefaultInvoiceType: input.DefaultInvoiceType, IsDefault: input.IsDefault, Enabled: input.Enabled,
	}
	if organizationID == uuid.Nil || actorID == uuid.Nil || profile.ID == uuid.Nil || input.ExpectedVersion == 0 || !validPartnerInvoiceProfile(profile) || (profile.IsDefault && !profile.Enabled) {
		return nil, ErrPartnerInvoiceProfileInvalidArgument
	}
	audit := partnerInvoiceProfileAudit(organizationID, actorID, profile, "partner.invoice_profile.update")
	return uc.repo.Update(ctx, organizationID, profile, input.ExpectedVersion, audit)
}

func validPartnerInvoiceProfile(profile *PartnerInvoiceProfile) bool {
	return profile != nil && profile.PartnerID != uuid.Nil && profile.InvoiceTitle != "" && utf8.RuneCountInString(profile.InvoiceTitle) <= 200 && profile.TaxpayerIdentificationNo != "" && utf8.RuneCountInString(profile.TaxpayerIdentificationNo) <= 64 && utf8.RuneCountInString(profile.RegisteredAddress) <= 500 && utf8.RuneCountInString(profile.RegisteredPhone) <= 50 && utf8.RuneCountInString(profile.BankName) <= 200 && utf8.RuneCountInString(profile.BankAccount) <= 100 && (profile.DefaultInvoiceType == FinanceInvoiceNormal || profile.DefaultInvoiceType == FinanceInvoiceSpecial) && (profile.DefaultInvoiceType != FinanceInvoiceSpecial || (profile.RegisteredAddress != "" && profile.RegisteredPhone != "" && profile.BankName != "" && profile.BankAccount != ""))
}

func partnerInvoiceProfileAudit(organizationID, actorID uuid.UUID, profile *PartnerInvoiceProfile, action string) *AuditEvent {
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", ResourceType: "partner_invoice_profile", ResourceID: profile.ID.String(), Details: map[string]string{"partner.id": profile.PartnerID.String(), "invoice_profile.id": profile.ID.String()}}
}
