package biz

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerInvoiceProfileNotFound        = errors.NotFound("PARTNER_INVOICE_PROFILE_NOT_FOUND", "往来单位尚未配置开票资料")
	ErrPartnerInvoiceProfileInvalidArgument = errors.BadRequest("PARTNER_INVOICE_PROFILE_INVALID_ARGUMENT", "开票资料字段不合法")
	ErrPartnerInvoiceProfileVersionConflict = errors.Conflict("PARTNER_INVOICE_PROFILE_VERSION_CONFLICT", "开票资料已被其他操作人修改，请刷新后重试")
)

type PartnerInvoiceProfile struct {
	ID, OrganizationID, PartnerID          uuid.UUID
	InvoiceTitle, TaxpayerIdentificationNo string
	RegisteredAddress, RegisteredPhone     string
	BankName, BankAccount                  string
	DefaultInvoiceType                     FinanceInvoiceType
	Version                                uint64
}

type SavePartnerInvoiceProfileInput struct {
	PartnerID                              uuid.UUID
	InvoiceTitle, TaxpayerIdentificationNo string
	RegisteredAddress, RegisteredPhone     string
	BankName, BankAccount                  string
	DefaultInvoiceType                     FinanceInvoiceType
	ExpectedVersion                        uint64
}

type PartnerInvoiceProfileRepo interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*PartnerInvoiceProfile, error)
	Save(context.Context, uuid.UUID, *PartnerInvoiceProfile, uint64, *AuditEvent) (*PartnerInvoiceProfile, error)
}

type PartnerInvoiceProfileUsecase struct{ repo PartnerInvoiceProfileRepo }

func NewPartnerInvoiceProfileUsecase(repo PartnerInvoiceProfileRepo) *PartnerInvoiceProfileUsecase {
	return &PartnerInvoiceProfileUsecase{repo: repo}
}

func (uc *PartnerInvoiceProfileUsecase) Get(ctx context.Context, organizationID, partnerID uuid.UUID) (*PartnerInvoiceProfile, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil {
		return nil, ErrPartnerInvoiceProfileInvalidArgument
	}
	return uc.repo.Get(ctx, organizationID, partnerID)
}

func (uc *PartnerInvoiceProfileUsecase) Save(ctx context.Context, organizationID, actorID uuid.UUID, input SavePartnerInvoiceProfileInput) (*PartnerInvoiceProfile, error) {
	profile := &PartnerInvoiceProfile{
		ID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, PartnerID: input.PartnerID,
		InvoiceTitle: strings.TrimSpace(input.InvoiceTitle), TaxpayerIdentificationNo: strings.TrimSpace(input.TaxpayerIdentificationNo),
		RegisteredAddress: strings.TrimSpace(input.RegisteredAddress), RegisteredPhone: strings.TrimSpace(input.RegisteredPhone),
		BankName: strings.TrimSpace(input.BankName), BankAccount: strings.TrimSpace(input.BankAccount), DefaultInvoiceType: input.DefaultInvoiceType,
	}
	if organizationID == uuid.Nil || actorID == uuid.Nil || profile.PartnerID == uuid.Nil || profile.InvoiceTitle == "" || utf8.RuneCountInString(profile.InvoiceTitle) > 200 || profile.TaxpayerIdentificationNo == "" || utf8.RuneCountInString(profile.TaxpayerIdentificationNo) > 64 || utf8.RuneCountInString(profile.RegisteredAddress) > 500 || utf8.RuneCountInString(profile.RegisteredPhone) > 50 || utf8.RuneCountInString(profile.BankName) > 200 || utf8.RuneCountInString(profile.BankAccount) > 100 || (profile.DefaultInvoiceType != FinanceInvoiceNormal && profile.DefaultInvoiceType != FinanceInvoiceSpecial) {
		return nil, ErrPartnerInvoiceProfileInvalidArgument
	}
	if profile.DefaultInvoiceType == FinanceInvoiceSpecial && (profile.RegisteredAddress == "" || profile.RegisteredPhone == "" || profile.BankName == "" || profile.BankAccount == "") {
		return nil, ErrPartnerInvoiceProfileInvalidArgument
	}
	audit := &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.invoice_profile.save", Result: "success", ResourceType: "partner_invoice_profile", ResourceID: profile.PartnerID.String(), Details: map[string]string{"partner.id": profile.PartnerID.String()}}
	return uc.repo.Save(ctx, organizationID, profile, input.ExpectedVersion, audit)
}
