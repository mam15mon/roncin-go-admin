package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerShippingPresets(ctx context.Context, request *v1.ListPartnerShippingPresetsRequest) (*v1.ListPartnerShippingPresetsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	options := biz.PartnerShippingPresetListOptions{}
	if request.PresetType != nil {
		options.PresetType = partnerShippingPresetTypeFromAPI(request.GetPresetType())
	}
	if request.Enabled != nil {
		enabled := request.GetEnabled()
		options.Enabled = &enabled
	}
	items, err := s.shippingPresetUsecase.List(ctx, principal.Organization.ID, partnerID, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerShippingPreset, 0, len(items))
	for _, item := range items {
		data = append(data, partnerShippingPresetToAPI(item))
	}
	return &v1.ListPartnerShippingPresetsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerShippingPreset(ctx context.Context, request *v1.CreatePartnerShippingPresetRequest) (*v1.CreatePartnerShippingPresetResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	created, err := s.shippingPresetUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, partnerShippingPresetFromAPI(request.GetPreset()))
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerShippingPresetResponse{Success: true, Code: 0, Message: "OK", Data: partnerShippingPresetToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerShippingPreset(ctx context.Context, request *v1.UpdatePartnerShippingPresetRequest) (*v1.UpdatePartnerShippingPresetResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	if partnerErr != nil || idErr != nil {
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	updated, err := s.shippingPresetUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, partnerShippingPresetFromAPI(request.GetPreset()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerShippingPresetResponse{Success: true, Code: 0, Message: "OK", Data: partnerShippingPresetToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func partnerShippingPresetTypeFromAPI(value v1.PartnerShippingPresetType) biz.PartnerShippingPresetType {
	switch value {
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_SHIPPER:
		return biz.PartnerShippingPresetShipper
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_CONSIGNEE:
		return biz.PartnerShippingPresetConsignee
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_NOTIFY_PARTY:
		return biz.PartnerShippingPresetNotifyParty
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_ENGLISH_CARGO_NAME:
		return biz.PartnerShippingPresetEnglishCargoName
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_HS_CODE:
		return biz.PartnerShippingPresetHSCode
	case v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_MARKS:
		return biz.PartnerShippingPresetMarks
	default:
		return ""
	}
}

func partnerShippingPresetTypeToAPI(value biz.PartnerShippingPresetType) v1.PartnerShippingPresetType {
	switch value {
	case biz.PartnerShippingPresetShipper:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_SHIPPER
	case biz.PartnerShippingPresetConsignee:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_CONSIGNEE
	case biz.PartnerShippingPresetNotifyParty:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_NOTIFY_PARTY
	case biz.PartnerShippingPresetEnglishCargoName:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_ENGLISH_CARGO_NAME
	case biz.PartnerShippingPresetHSCode:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_HS_CODE
	case biz.PartnerShippingPresetMarks:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_MARKS
	default:
		return v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_UNSPECIFIED
	}
}

func partnerShippingPresetFromAPI(value *v1.PartnerShippingPresetInput) *biz.PartnerShippingPreset {
	if value == nil {
		return nil
	}
	result := &biz.PartnerShippingPreset{
		PresetType: partnerShippingPresetTypeFromAPI(value.GetPresetType()), Title: value.GetTitle(),
		IsDefault: value.GetIsDefault(), SortOrder: int(value.GetSortOrder()), Remark: value.GetRemark(), Enabled: value.GetEnabled(),
	}
	if party := value.GetParty(); party != nil {
		result.Party = &biz.PartnerShippingPartyPayload{
			CompanyName: party.GetCompanyName(), Address: party.GetAddress(), ContactName: party.GetContactName(),
			Phone: party.GetPhone(), Email: party.GetEmail(), CountryCode: party.GetCountryCode(), TaxIdentifier: party.GetTaxIdentifier(),
		}
	}
	if textPayload := value.GetText(); textPayload != nil {
		result.Text = &biz.PartnerShippingTextPayload{Content: textPayload.GetContent(), Code: textPayload.GetCode()}
	}
	return result
}

func partnerShippingPresetToAPI(value *biz.PartnerShippingPreset) *v1.PartnerShippingPreset {
	result := &v1.PartnerShippingPreset{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), PresetType: partnerShippingPresetTypeToAPI(value.PresetType),
		Title: value.Title, IsDefault: value.IsDefault, SortOrder: int32(value.SortOrder), Remark: value.Remark, Enabled: value.Enabled,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339),
	}
	if value.Party != nil {
		result.Payload = &v1.PartnerShippingPreset_Party{Party: &v1.PartnerShippingPartyPayload{
			CompanyName: value.Party.CompanyName, Address: value.Party.Address, ContactName: value.Party.ContactName,
			Phone: value.Party.Phone, Email: value.Party.Email, CountryCode: value.Party.CountryCode, TaxIdentifier: value.Party.TaxIdentifier,
		}}
	}
	if value.Text != nil {
		result.Payload = &v1.PartnerShippingPreset_Text{Text: &v1.PartnerShippingTextPayload{Content: value.Text.Content, Code: value.Text.Code}}
	}
	return result
}
