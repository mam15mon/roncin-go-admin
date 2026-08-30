package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	profileent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerinvoiceprofile"
)

type partnerInvoiceProfileRepo struct{ data *Data }

func NewPartnerInvoiceProfileRepo(data *Data) biz.PartnerInvoiceProfileRepo {
	return &partnerInvoiceProfileRepo{data: data}
}

func (r *partnerInvoiceProfileRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID) ([]*biz.PartnerInvoiceProfile, error) {
	items, err := r.data.db.PartnerInvoiceProfile.Query().
		Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(partnerID)).
		Order(profileent.ByIsDefault(sql.OrderDesc()), profileent.ByEnabled(sql.OrderDesc()), profileent.ByCreatedAt()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.PartnerInvoiceProfile, 0, len(items))
	for _, item := range items {
		out = append(out, partnerInvoiceProfileToBiz(item))
	}
	return out, nil
}

func (r *partnerInvoiceProfileRepo) Create(ctx context.Context, organizationID uuid.UUID, profile *biz.PartnerInvoiceProfile, audit *biz.AuditEvent) (*biz.PartnerInvoiceProfile, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.PartnerInvoiceProfile, error) { _ = tx.Rollback(); return nil, value }
	if err = lockInvoiceProfilePartner(ctx, tx, organizationID, profile.PartnerID); err != nil {
		return rollback(err)
	}
	count, err := tx.PartnerInvoiceProfile.Query().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID)).Count(ctx)
	if err != nil {
		return rollback(err)
	}
	profile.IsDefault = profile.IsDefault || count == 0
	if profile.IsDefault {
		if _, err = tx.PartnerInvoiceProfile.Update().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID), profileent.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
			return rollback(err)
		}
	}
	item, err := tx.PartnerInvoiceProfile.Create().
		SetID(profile.ID).
		SetOrganizationID(organizationID).
		SetPartnerID(profile.PartnerID).
		SetInvoiceTitle(profile.InvoiceTitle).
		SetTaxpayerIdentificationNo(profile.TaxpayerIdentificationNo).
		SetRegisteredAddress(profile.RegisteredAddress).
		SetRegisteredPhone(profile.RegisteredPhone).
		SetBankName(profile.BankName).
		SetBankAccount(profile.BankAccount).
		SetDefaultInvoiceType(profileent.DefaultInvoiceType(profile.DefaultInvoiceType)).
		SetIsDefault(profile.IsDefault).
		SetEnabled(true).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		return rollback(mapEntConstraint(err, "partner_invoice_profile_title_key", biz.ErrPartnerInvoiceProfileTitleExists))
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func (r *partnerInvoiceProfileRepo) Update(ctx context.Context, organizationID uuid.UUID, profile *biz.PartnerInvoiceProfile, expectedVersion uint64, audit *biz.AuditEvent) (*biz.PartnerInvoiceProfile, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.PartnerInvoiceProfile, error) { _ = tx.Rollback(); return nil, value }
	if err = lockInvoiceProfilePartner(ctx, tx, organizationID, profile.PartnerID); err != nil {
		return rollback(err)
	}
	current, err := tx.PartnerInvoiceProfile.Query().Where(profileent.IDEQ(profile.ID), profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return rollback(biz.ErrPartnerInvoiceProfileNotFound)
	}
	if err != nil {
		return rollback(err)
	}
	if current.Version != expectedVersion {
		return rollback(biz.ErrPartnerInvoiceProfileVersionConflict)
	}
	if current.IsDefault && !profile.IsDefault {
		otherEnabled, countErr := tx.PartnerInvoiceProfile.Query().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID), profileent.IDNEQ(profile.ID), profileent.EnabledEQ(true)).Exist(ctx)
		if countErr != nil {
			return rollback(countErr)
		}
		if profile.Enabled || otherEnabled {
			return rollback(biz.ErrPartnerInvoiceProfileDefaultRequired)
		}
	}
	if profile.IsDefault {
		if _, err = tx.PartnerInvoiceProfile.Update().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID), profileent.IDNEQ(profile.ID), profileent.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
			return rollback(err)
		}
	}
	item, err := current.Update().
		SetInvoiceTitle(profile.InvoiceTitle).
		SetTaxpayerIdentificationNo(profile.TaxpayerIdentificationNo).
		SetRegisteredAddress(profile.RegisteredAddress).
		SetRegisteredPhone(profile.RegisteredPhone).
		SetBankName(profile.BankName).
		SetBankAccount(profile.BankAccount).
		SetDefaultInvoiceType(profileent.DefaultInvoiceType(profile.DefaultInvoiceType)).
		SetIsDefault(profile.IsDefault).
		SetEnabled(profile.Enabled).
		SetVersion(current.Version + 1).
		Save(ctx)
	if err != nil {
		return rollback(mapEntConstraint(err, "partner_invoice_profile_title_key", biz.ErrPartnerInvoiceProfileTitleExists))
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func lockInvoiceProfilePartner(ctx context.Context, tx *ent.Tx, organizationID, partnerID uuid.UUID) error {
	_, err := tx.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return biz.ErrPartnerNotFound
	}
	return err
}

func partnerInvoiceProfileToBiz(item *ent.PartnerInvoiceProfile) *biz.PartnerInvoiceProfile {
	return &biz.PartnerInvoiceProfile{
		ID: item.ID, OrganizationID: item.OrganizationID, PartnerID: item.PartnerID,
		InvoiceTitle: item.InvoiceTitle, TaxpayerIdentificationNo: item.TaxpayerIdentificationNo,
		RegisteredAddress: item.RegisteredAddress, RegisteredPhone: item.RegisteredPhone,
		BankName: item.BankName, BankAccount: item.BankAccount, DefaultInvoiceType: biz.FinanceInvoiceType(item.DefaultInvoiceType),
		IsDefault: item.IsDefault, Enabled: item.Enabled, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

var _ biz.PartnerInvoiceProfileRepo = (*partnerInvoiceProfileRepo)(nil)
