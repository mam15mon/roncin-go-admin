package biz

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPartnerShippingPresetNotFound        = errors.NotFound("PARTNER_SHIPPING_PRESET_NOT_FOUND", "常用单证预设不存在")
	ErrPartnerShippingPresetInvalidArgument = errors.BadRequest("PARTNER_SHIPPING_PRESET_INVALID_ARGUMENT", "常用单证预设字段不合法")
)

var partnerHSCodePattern = regexp.MustCompile(`^[0-9]{6,10}$`)

type PartnerShippingPresetType string

const (
	PartnerShippingPresetShipper          PartnerShippingPresetType = "SHIPPER"
	PartnerShippingPresetConsignee        PartnerShippingPresetType = "CONSIGNEE"
	PartnerShippingPresetNotifyParty      PartnerShippingPresetType = "NOTIFY_PARTY"
	PartnerShippingPresetEnglishCargoName PartnerShippingPresetType = "ENGLISH_CARGO_NAME"
	PartnerShippingPresetHSCode           PartnerShippingPresetType = "HS_CODE"
	PartnerShippingPresetMarks            PartnerShippingPresetType = "MARKS"
)

func (presetType PartnerShippingPresetType) Valid() bool {
	switch presetType {
	case PartnerShippingPresetShipper, PartnerShippingPresetConsignee, PartnerShippingPresetNotifyParty, PartnerShippingPresetEnglishCargoName, PartnerShippingPresetHSCode, PartnerShippingPresetMarks:
		return true
	default:
		return false
	}
}

func (presetType PartnerShippingPresetType) Party() bool {
	return presetType == PartnerShippingPresetShipper || presetType == PartnerShippingPresetConsignee || presetType == PartnerShippingPresetNotifyParty
}

type PartnerShippingPartyPayload struct {
	CompanyName   string
	Address       string
	ContactName   string
	Phone         string
	Email         string
	CountryCode   string
	TaxIdentifier string
}

type PartnerShippingTextPayload struct {
	Content string
	Code    string
}

type PartnerShippingPreset struct {
	ID         uuid.UUID
	PartnerID  uuid.UUID
	PresetType PartnerShippingPresetType
	Title      string
	Party      *PartnerShippingPartyPayload
	Text       *PartnerShippingTextPayload
	IsDefault  bool
	SortOrder  int
	Remark     string
	Enabled    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PartnerShippingPresetListOptions struct {
	PresetType PartnerShippingPresetType
	Enabled    *bool
}

type PartnerShippingPresetRepo interface {
	List(context.Context, uuid.UUID, uuid.UUID, PartnerShippingPresetListOptions) ([]*PartnerShippingPreset, error)
	Create(context.Context, uuid.UUID, uuid.UUID, *PartnerShippingPreset) (*PartnerShippingPreset, error)
	Update(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, *PartnerShippingPreset) (*PartnerShippingPreset, error)
}

type PartnerShippingPresetUsecase struct {
	repo  PartnerShippingPresetRepo
	audit AuditRepo
}

func NewPartnerShippingPresetUsecase(repo PartnerShippingPresetRepo, audit AuditRepo) *PartnerShippingPresetUsecase {
	return &PartnerShippingPresetUsecase{repo: repo, audit: audit}
}

func (uc *PartnerShippingPresetUsecase) List(ctx context.Context, organizationID, partnerID uuid.UUID, options PartnerShippingPresetListOptions) ([]*PartnerShippingPreset, error) {
	if organizationID == uuid.Nil || partnerID == uuid.Nil || (options.PresetType != "" && !options.PresetType.Valid()) {
		return nil, ErrPartnerShippingPresetInvalidArgument
	}
	return uc.repo.List(ctx, organizationID, partnerID, options)
}

func (uc *PartnerShippingPresetUsecase) Create(ctx context.Context, organizationID, actorID, partnerID uuid.UUID, input *PartnerShippingPreset) (*PartnerShippingPreset, error) {
	normalized, err := normalizePartnerShippingPreset(input)
	if err != nil {
		return nil, err
	}
	created, err := uc.repo.Create(ctx, organizationID, partnerID, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.shipping_preset.create", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"partner.id": partnerID.String(), "preset.id": created.ID.String(), "preset.type": string(created.PresetType)}}); err != nil {
		return nil, fmt.Errorf("write partner shipping preset create audit: %w", err)
	}
	return created, nil
}

func (uc *PartnerShippingPresetUsecase) Update(ctx context.Context, organizationID, actorID, partnerID, id uuid.UUID, input *PartnerShippingPreset) (*PartnerShippingPreset, error) {
	if id == uuid.Nil {
		return nil, ErrPartnerShippingPresetNotFound
	}
	normalized, err := normalizePartnerShippingPreset(input)
	if err != nil {
		return nil, err
	}
	updated, err := uc.repo.Update(ctx, organizationID, partnerID, id, normalized)
	if err != nil {
		return nil, err
	}
	if err := uc.audit.WriteAudit(ctx, &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "partner.shipping_preset.update", ResourceType: "partner", ResourceID: partnerID.String(), Result: "success", Details: map[string]string{"partner.id": partnerID.String(), "preset.id": updated.ID.String(), "preset.type": string(updated.PresetType)}}); err != nil {
		return nil, fmt.Errorf("write partner shipping preset update audit: %w", err)
	}
	return updated, nil
}

func normalizePartnerShippingPreset(input *PartnerShippingPreset) (*PartnerShippingPreset, error) {
	if input == nil || !input.PresetType.Valid() || input.SortOrder < 0 {
		return nil, ErrPartnerShippingPresetInvalidArgument
	}
	output := *input
	output.Title = strings.TrimSpace(output.Title)
	output.Remark = strings.TrimSpace(output.Remark)
	if output.Title == "" || utf8.RuneCountInString(output.Title) > 100 || utf8.RuneCountInString(output.Remark) > 500 {
		return nil, ErrPartnerShippingPresetInvalidArgument
	}
	if output.PresetType.Party() {
		if output.Party == nil || output.Text != nil {
			return nil, ErrPartnerShippingPresetInvalidArgument
		}
		party := *output.Party
		party.CompanyName = strings.TrimSpace(party.CompanyName)
		party.Address = strings.TrimSpace(party.Address)
		party.ContactName = strings.TrimSpace(party.ContactName)
		party.Phone = strings.TrimSpace(party.Phone)
		party.Email = strings.TrimSpace(party.Email)
		party.CountryCode = strings.ToUpper(strings.TrimSpace(party.CountryCode))
		party.TaxIdentifier = strings.TrimSpace(party.TaxIdentifier)
		if party.CompanyName == "" || len(party.CountryCode) != 2 || utf8.RuneCountInString(party.CompanyName) > 200 || utf8.RuneCountInString(party.Address) > 500 || utf8.RuneCountInString(party.ContactName) > 100 || utf8.RuneCountInString(party.Phone) > 64 || utf8.RuneCountInString(party.Email) > 254 || utf8.RuneCountInString(party.TaxIdentifier) > 64 {
			return nil, ErrPartnerShippingPresetInvalidArgument
		}
		if party.Email != "" {
			parsed, err := mail.ParseAddress(party.Email)
			if err != nil || parsed.Address != party.Email {
				return nil, ErrPartnerShippingPresetInvalidArgument
			}
		}
		output.Party = &party
		return &output, nil
	}
	if output.Text == nil || output.Party != nil {
		return nil, ErrPartnerShippingPresetInvalidArgument
	}
	textPayload := *output.Text
	textPayload.Content = strings.TrimSpace(textPayload.Content)
	textPayload.Code = strings.TrimSpace(textPayload.Code)
	if utf8.RuneCountInString(textPayload.Content) > 4000 || utf8.RuneCountInString(textPayload.Code) > 64 {
		return nil, ErrPartnerShippingPresetInvalidArgument
	}
	if output.PresetType == PartnerShippingPresetHSCode {
		if !partnerHSCodePattern.MatchString(textPayload.Code) {
			return nil, ErrPartnerShippingPresetInvalidArgument
		}
	} else if textPayload.Content == "" {
		return nil, ErrPartnerShippingPresetInvalidArgument
	}
	output.Text = &textPayload
	return &output, nil
}
