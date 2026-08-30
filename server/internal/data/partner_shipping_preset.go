package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourcepartner"
	partyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceparty"
	textent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceshippingtext"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

type partnerShippingPresetRepo struct{ data *Data }

func NewPartnerShippingPresetRepo(data *Data) biz.PartnerShippingPresetRepo {
	return &partnerShippingPresetRepo{data: data}
}

func (r *partnerShippingPresetRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, options biz.PartnerShippingPresetListOptions) ([]*biz.PartnerShippingPreset, error) {
	query := r.data.db.EnterpriseResource.Query().Where(
		resourceent.OrganizationIDEQ(organizationID),
		resourceent.HasPartnerLinksWith(linkent.PartnerIDEQ(partnerID)),
	).WithParty().WithShippingText().WithPartnerLinks(func(query *ent.EnterpriseResourcePartnerQuery) {
		query.Where(linkent.PartnerIDEQ(partnerID))
	})
	if options.PresetType != "" {
		query.Where(resourceent.ResourceTypeEQ(resourceent.ResourceType(options.PresetType)))
	} else {
		query.Where(resourceent.ResourceTypeIn(shippingPresetResourceTypes()...))
	}
	if options.Enabled != nil {
		query.Where(resourceent.EnabledEQ(*options.Enabled))
	}
	items, err := query.Order(resourceent.ByResourceType(), resourceent.BySortOrder(), resourceent.ByShortName()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerShippingPreset, 0, len(items))
	for _, item := range items {
		result = append(result, enterpriseResourceToShippingPreset(item, partnerID))
	}
	return result, nil
}

func (r *partnerShippingPresetRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, input *biz.PartnerShippingPreset, audit *biz.AuditEvent) (*biz.PartnerShippingPreset, error) {
	var resource *ent.EnterpriseResource
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if partnerErr := ensurePartnerInOrganization(ctx, tx, organizationID, partnerID); partnerErr != nil {
			return partnerErr
		}
		if input.IsDefault {
			if defaultErr := lockAndClearResourceDefault(ctx, tx, partnerID, input.PresetType, uuid.Nil); defaultErr != nil {
				return defaultErr
			}
		}
		actorID := uuid.Nil
		if audit.UserID != nil {
			actorID = *audit.UserID
		}
		var saveErr error
		resource, saveErr = tx.EnterpriseResource.Create().SetOrganizationID(organizationID).SetResourceType(resourceent.ResourceType(input.PresetType)).SetShortName(input.Title).SetEnabled(input.Enabled).SetSortOrder(input.SortOrder).SetNillableCreatedBy(nonNilUUID(actorID)).SetNillableUpdatedBy(nonNilUUID(actorID)).Save(ctx)
		if saveErr != nil {
			return saveErr
		}
		if detailErr := createShippingPresetDetail(ctx, tx, resource.ID, organizationID, input); detailErr != nil {
			return detailErr
		}
		link, linkErr := tx.EnterpriseResourcePartner.Create().SetResourceID(resource.ID).SetPartnerID(partnerID).SetResourceType(linkent.ResourceType(input.PresetType)).SetIsDefault(input.IsDefault).Save(ctx)
		if linkErr != nil {
			return linkErr
		}
		resource.Edges.PartnerLinks = []*ent.EnterpriseResourcePartner{link}
		audit.Details["preset.id"] = resource.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.getShippingPreset(ctx, organizationID, partnerID, resource.ID)
}

func (r *partnerShippingPresetRepo) Update(ctx context.Context, organizationID, partnerID, id uuid.UUID, input *biz.PartnerShippingPreset, audit *biz.AuditEvent) (*biz.PartnerShippingPreset, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, queryErr := tx.EnterpriseResource.Query().Where(resourceent.IDEQ(id), resourceent.OrganizationIDEQ(organizationID), resourceent.HasPartnerLinksWith(linkent.PartnerIDEQ(partnerID))).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrPartnerShippingPresetNotFound, nil)
		}
		if biz.PartnerShippingPresetType(existing.ResourceType) != input.PresetType {
			return biz.ErrPartnerShippingPresetInvalidArgument
		}
		if input.IsDefault {
			if defaultErr := lockAndClearResourceDefault(ctx, tx, partnerID, input.PresetType, id); defaultErr != nil {
				return defaultErr
			}
		}
		update := existing.Update().SetShortName(input.Title).SetEnabled(input.Enabled).SetSortOrder(input.SortOrder)
		if audit.UserID != nil {
			update.SetUpdatedBy(*audit.UserID)
		}
		if _, saveErr := update.Save(ctx); saveErr != nil {
			return saveErr
		}
		if detailErr := updateShippingPresetDetail(ctx, tx, id, input); detailErr != nil {
			return detailErr
		}
		if _, linkErr := tx.EnterpriseResourcePartner.Update().Where(linkent.ResourceIDEQ(id), linkent.PartnerIDEQ(partnerID)).SetIsDefault(input.IsDefault && input.Enabled).Save(ctx); linkErr != nil {
			return linkErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.getShippingPreset(ctx, organizationID, partnerID, id)
}

func (r *partnerShippingPresetRepo) getShippingPreset(ctx context.Context, organizationID, partnerID, id uuid.UUID) (*biz.PartnerShippingPreset, error) {
	item, err := r.data.db.EnterpriseResource.Query().Where(resourceent.IDEQ(id), resourceent.OrganizationIDEQ(organizationID), resourceent.HasPartnerLinksWith(linkent.PartnerIDEQ(partnerID))).WithParty().WithShippingText().WithPartnerLinks(func(query *ent.EnterpriseResourcePartnerQuery) { query.Where(linkent.PartnerIDEQ(partnerID)) }).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrPartnerShippingPresetNotFound, nil)
	}
	return enterpriseResourceToShippingPreset(item, partnerID), nil
}

func ensurePartnerInOrganization(ctx context.Context, tx *ent.Tx, organizationID, partnerID uuid.UUID) error {
	exists, err := tx.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrPartnerNotFound
	}
	return nil
}

func lockAndClearResourceDefault(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, presetType biz.PartnerShippingPresetType, exceptResourceID uuid.UUID) error {
	if _, err := tx.Partner.Query().Where(partnerent.IDEQ(partnerID)).ForUpdate().Only(ctx); err != nil {
		return err
	}
	query := tx.EnterpriseResourcePartner.Update().Where(linkent.PartnerIDEQ(partnerID), linkent.ResourceTypeEQ(linkent.ResourceType(presetType)), linkent.IsDefaultEQ(true))
	if exceptResourceID != uuid.Nil {
		query.Where(linkent.ResourceIDNEQ(exceptResourceID))
	}
	_, err := query.SetIsDefault(false).Save(ctx)
	return err
}

func createShippingPresetDetail(ctx context.Context, tx *ent.Tx, id, organizationID uuid.UUID, input *biz.PartnerShippingPreset) error {
	if input.PresetType.Party() {
		party := input.Party
		_, err := tx.EnterpriseResourceParty.Create().SetResourceID(id).SetOrganizationID(organizationID).SetResourceType(partyent.ResourceType(input.PresetType)).SetCompanyName(party.CompanyName).SetNillableAddress(optionalString(party.Address)).SetNillableContactName(optionalString(party.ContactName)).SetNillableContactPhone(optionalString(party.Phone)).SetNillableEmail(optionalString(party.Email)).SetCountryCode(party.CountryCode).SetNillableTaxIdentifier(optionalString(party.TaxIdentifier)).SetNillableRemark(optionalString(input.Remark)).Save(ctx)
		return err
	}
	_, err := tx.EnterpriseResourceShippingText.Create().SetResourceID(id).SetNillableContent(optionalString(input.Text.Content)).SetNillableCode(optionalString(input.Text.Code)).SetNillableRemark(optionalString(input.Remark)).Save(ctx)
	return err
}

func updateShippingPresetDetail(ctx context.Context, tx *ent.Tx, id uuid.UUID, input *biz.PartnerShippingPreset) error {
	if input.PresetType.Party() {
		party := input.Party
		builder := tx.EnterpriseResourceParty.Update().Where(partyent.ResourceIDEQ(id)).SetCompanyName(party.CompanyName).SetCountryCode(party.CountryCode).ClearAddress().ClearContactName().ClearContactPhone().ClearEmail().ClearTaxIdentifier().ClearRemark()
		builder.SetNillableAddress(optionalString(party.Address)).SetNillableContactName(optionalString(party.ContactName)).SetNillableContactPhone(optionalString(party.Phone)).SetNillableEmail(optionalString(party.Email)).SetNillableTaxIdentifier(optionalString(party.TaxIdentifier)).SetNillableRemark(optionalString(input.Remark))
		_, err := builder.Save(ctx)
		return err
	}
	builder := tx.EnterpriseResourceShippingText.Update().Where(textent.ResourceIDEQ(id)).ClearContent().ClearCode().ClearRemark()
	builder.SetNillableContent(optionalString(input.Text.Content)).SetNillableCode(optionalString(input.Text.Code)).SetNillableRemark(optionalString(input.Remark))
	_, err := builder.Save(ctx)
	return err
}

func enterpriseResourceToShippingPreset(item *ent.EnterpriseResource, partnerID uuid.UUID) *biz.PartnerShippingPreset {
	result := &biz.PartnerShippingPreset{ID: item.ID, PartnerID: partnerID, PresetType: biz.PartnerShippingPresetType(item.ResourceType), Title: item.ShortName, SortOrder: item.SortOrder, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, link := range item.Edges.PartnerLinks {
		if link.PartnerID == partnerID {
			result.IsDefault = link.IsDefault
			break
		}
	}
	if result.PresetType.Party() && item.Edges.Party != nil {
		party := item.Edges.Party
		result.Party = &biz.PartnerShippingPartyPayload{CompanyName: party.CompanyName, Address: stringValue(party.Address), ContactName: stringValue(party.ContactName), Phone: stringValue(party.ContactPhone), Email: stringValue(party.Email), CountryCode: party.CountryCode, TaxIdentifier: stringValue(party.TaxIdentifier)}
		result.Remark = stringValue(party.Remark)
	} else if item.Edges.ShippingText != nil {
		text := item.Edges.ShippingText
		result.Text = &biz.PartnerShippingTextPayload{Content: stringValue(text.Content), Code: stringValue(text.Code)}
		result.Remark = stringValue(text.Remark)
	}
	return result
}

func shippingPresetResourceTypes() []resourceent.ResourceType {
	return []resourceent.ResourceType{resourceent.ResourceTypeSHIPPER, resourceent.ResourceTypeCONSIGNEE, resourceent.ResourceTypeNOTIFY_PARTY, resourceent.ResourceTypeENGLISH_CARGO_NAME, resourceent.ResourceTypeHS_CODE, resourceent.ResourceTypeMARKS}
}

func nonNilUUID(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
