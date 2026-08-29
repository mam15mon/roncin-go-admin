package data

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	shippingpresetent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnershippingpreset"
)

type partnerShippingPresetRepo struct{ data *Data }

func NewPartnerShippingPresetRepo(data *Data) biz.PartnerShippingPresetRepo {
	return &partnerShippingPresetRepo{data: data}
}

func (r *partnerShippingPresetRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, options biz.PartnerShippingPresetListOptions) ([]*biz.PartnerShippingPreset, error) {
	query := r.data.db.PartnerShippingPreset.Query().Where(
		shippingpresetent.PartnerIDEQ(partnerID),
		shippingpresetent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	)
	if options.PresetType != "" {
		query.Where(shippingpresetent.PresetTypeEQ(shippingpresetent.PresetType(options.PresetType)))
	}
	if options.Enabled != nil {
		query.Where(shippingpresetent.EnabledEQ(*options.Enabled))
	}
	items, err := query.Order(
		shippingpresetent.ByPresetType(),
		shippingpresetent.ByIsDefault(entsql.OrderDesc()),
		shippingpresetent.BySortOrder(),
		shippingpresetent.ByTitle(),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerShippingPreset, 0, len(items))
	for _, item := range items {
		result = append(result, partnerShippingPresetToBiz(item))
	}
	return result, nil
}

func (r *partnerShippingPresetRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, input *biz.PartnerShippingPreset, audit *biz.AuditEvent) (*biz.PartnerShippingPreset, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if err := ensurePartnerInOrganization(ctx, tx, organizationID, partnerID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if input.IsDefault {
		if err := clearPartnerShippingPresetDefault(ctx, tx, partnerID, input.PresetType, uuid.Nil); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	created, err := setPartnerShippingPresetCreate(tx.PartnerShippingPreset.Create().SetPartnerID(partnerID), input).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.Details["preset.id"] = created.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return partnerShippingPresetToBiz(created), nil
}

func (r *partnerShippingPresetRepo) Update(ctx context.Context, organizationID, partnerID, id uuid.UUID, input *biz.PartnerShippingPreset, audit *biz.AuditEvent) (*biz.PartnerShippingPreset, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.PartnerShippingPreset.Query().Where(
		shippingpresetent.IDEQ(id),
		shippingpresetent.PartnerIDEQ(partnerID),
		shippingpresetent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerShippingPresetNotFound
		}
		return nil, err
	}
	if biz.PartnerShippingPresetType(existing.PresetType) != input.PresetType {
		_ = tx.Rollback()
		return nil, biz.ErrPartnerShippingPresetInvalidArgument
	}
	if input.IsDefault {
		if err := clearPartnerShippingPresetDefault(ctx, tx, partnerID, input.PresetType, id); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	updated, err := setPartnerShippingPresetUpdate(existing.Update(), input).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return partnerShippingPresetToBiz(updated), nil
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

func clearPartnerShippingPresetDefault(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, presetType biz.PartnerShippingPresetType, exceptID uuid.UUID) error {
	query := tx.PartnerShippingPreset.Update().Where(
		shippingpresetent.PartnerIDEQ(partnerID),
		shippingpresetent.PresetTypeEQ(shippingpresetent.PresetType(presetType)),
		shippingpresetent.IsDefaultEQ(true),
	)
	if exceptID != uuid.Nil {
		query.Where(shippingpresetent.IDNEQ(exceptID))
	}
	_, err := query.SetIsDefault(false).Save(ctx)
	return err
}

func setPartnerShippingPresetCreate(builder *ent.PartnerShippingPresetCreate, input *biz.PartnerShippingPreset) *ent.PartnerShippingPresetCreate {
	builder.SetPresetType(shippingpresetent.PresetType(input.PresetType)).SetTitle(input.Title).
		SetIsDefault(input.IsDefault).SetSortOrder(input.SortOrder).SetRemark(input.Remark).SetEnabled(input.Enabled)
	if input.Party != nil {
		builder.SetCompanyName(input.Party.CompanyName).SetAddress(input.Party.Address).SetContactName(input.Party.ContactName).
			SetPhone(input.Party.Phone).SetEmail(input.Party.Email).SetCountryCode(input.Party.CountryCode).SetTaxIdentifier(input.Party.TaxIdentifier)
	}
	if input.Text != nil {
		builder.SetContent(input.Text.Content).SetCode(input.Text.Code)
	}
	return builder
}

func setPartnerShippingPresetUpdate(builder *ent.PartnerShippingPresetUpdateOne, input *biz.PartnerShippingPreset) *ent.PartnerShippingPresetUpdateOne {
	builder.SetTitle(input.Title).SetIsDefault(input.IsDefault).SetSortOrder(input.SortOrder).SetRemark(input.Remark).SetEnabled(input.Enabled).
		ClearCompanyName().ClearAddress().ClearContactName().ClearPhone().ClearEmail().ClearCountryCode().ClearTaxIdentifier().ClearContent().ClearCode()
	if input.Party != nil {
		builder.SetCompanyName(input.Party.CompanyName).SetAddress(input.Party.Address).SetContactName(input.Party.ContactName).
			SetPhone(input.Party.Phone).SetEmail(input.Party.Email).SetCountryCode(input.Party.CountryCode).SetTaxIdentifier(input.Party.TaxIdentifier)
	}
	if input.Text != nil {
		builder.SetContent(input.Text.Content).SetCode(input.Text.Code)
	}
	return builder
}

func partnerShippingPresetToBiz(item *ent.PartnerShippingPreset) *biz.PartnerShippingPreset {
	result := &biz.PartnerShippingPreset{
		ID: item.ID, PartnerID: item.PartnerID, PresetType: biz.PartnerShippingPresetType(item.PresetType), Title: item.Title,
		IsDefault: item.IsDefault, SortOrder: item.SortOrder, Remark: item.Remark, Enabled: item.Enabled,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if result.PresetType.Party() {
		result.Party = &biz.PartnerShippingPartyPayload{
			CompanyName: stringValue(item.CompanyName), Address: stringValue(item.Address), ContactName: stringValue(item.ContactName),
			Phone: stringValue(item.Phone), Email: stringValue(item.Email), CountryCode: stringValue(item.CountryCode), TaxIdentifier: stringValue(item.TaxIdentifier),
		}
	} else {
		result.Text = &biz.PartnerShippingTextPayload{Content: stringValue(item.Content), Code: stringValue(item.Code)}
	}
	return result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
