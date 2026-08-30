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
	var item *ent.PartnerInvoiceProfile
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockInvoiceProfilePartner(ctx, tx, organizationID, profile.PartnerID); lockErr != nil {
			return lockErr
		}
		count, countErr := tx.PartnerInvoiceProfile.Query().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID)).Count(ctx)
		if countErr != nil {
			return countErr
		}
		profile.IsDefault = profile.IsDefault || count == 0
		if profile.IsDefault {
			if _, updateErr := tx.PartnerInvoiceProfile.Update().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID), profileent.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); updateErr != nil {
				return updateErr
			}
		}
		var saveErr error
		item, saveErr = tx.PartnerInvoiceProfile.Create().
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
		if saveErr != nil {
			return mapEntConstraint(saveErr, "partner_invoice_profile_title_key", biz.ErrPartnerInvoiceProfileTitleExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func (r *partnerInvoiceProfileRepo) Update(ctx context.Context, organizationID uuid.UUID, profile *biz.PartnerInvoiceProfile, expectedVersion uint64, audit *biz.AuditEvent) (*biz.PartnerInvoiceProfile, error) {
	var item *ent.PartnerInvoiceProfile
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockInvoiceProfilePartner(ctx, tx, organizationID, profile.PartnerID); lockErr != nil {
			return lockErr
		}
		current, queryErr := tx.PartnerInvoiceProfile.Query().Where(profileent.IDEQ(profile.ID), profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrPartnerInvoiceProfileNotFound, nil)
		}
		if current.Version != expectedVersion {
			return biz.ErrPartnerInvoiceProfileVersionConflict
		}
		if current.IsDefault && !profile.IsDefault {
			otherEnabled, countErr := tx.PartnerInvoiceProfile.Query().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID), profileent.IDNEQ(profile.ID), profileent.EnabledEQ(true)).Exist(ctx)
			if countErr != nil {
				return countErr
			}
			if profile.Enabled || otherEnabled {
				return biz.ErrPartnerInvoiceProfileDefaultRequired
			}
		}
		if profile.IsDefault {
			if _, updateErr := tx.PartnerInvoiceProfile.Update().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID), profileent.IDNEQ(profile.ID), profileent.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); updateErr != nil {
				return updateErr
			}
		}
		var saveErr error
		item, saveErr = current.Update().
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
		if saveErr != nil {
			return mapEntConstraint(saveErr, "partner_invoice_profile_title_key", biz.ErrPartnerInvoiceProfileTitleExists)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func lockInvoiceProfilePartner(ctx context.Context, tx *ent.Tx, organizationID, partnerID uuid.UUID) error {
	_, err := tx.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).ForUpdate().Only(ctx)
	return mapEntError(err, biz.ErrPartnerNotFound, nil)
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
