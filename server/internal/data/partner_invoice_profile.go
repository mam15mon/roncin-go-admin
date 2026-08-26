package data

import (
	"context"

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

func (r *partnerInvoiceProfileRepo) Get(ctx context.Context, organizationID, partnerID uuid.UUID) (*biz.PartnerInvoiceProfile, error) {
	item, err := r.data.db.PartnerInvoiceProfile.Query().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(partnerID)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrPartnerInvoiceProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func (r *partnerInvoiceProfileRepo) Save(ctx context.Context, organizationID uuid.UUID, profile *biz.PartnerInvoiceProfile, expectedVersion uint64, audit *biz.AuditEvent) (*biz.PartnerInvoiceProfile, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	rollback := func(value error) (*biz.PartnerInvoiceProfile, error) { _ = tx.Rollback(); return nil, value }
	partnerExists, err := tx.Partner.Query().Where(partnerent.IDEQ(profile.PartnerID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return rollback(err)
	}
	if !partnerExists {
		return rollback(biz.ErrPartnerNotFound)
	}
	item, err := tx.PartnerInvoiceProfile.Query().Where(profileent.OrganizationIDEQ(organizationID), profileent.PartnerIDEQ(profile.PartnerID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		if expectedVersion != 0 {
			return rollback(biz.ErrPartnerInvoiceProfileVersionConflict)
		}
		item, err = tx.PartnerInvoiceProfile.Create().SetID(profile.ID).SetOrganizationID(organizationID).SetPartnerID(profile.PartnerID).SetInvoiceTitle(profile.InvoiceTitle).SetTaxpayerIdentificationNo(profile.TaxpayerIdentificationNo).SetRegisteredAddress(profile.RegisteredAddress).SetRegisteredPhone(profile.RegisteredPhone).SetBankName(profile.BankName).SetBankAccount(profile.BankAccount).SetDefaultInvoiceType(profileent.DefaultInvoiceType(profile.DefaultInvoiceType)).SetVersion(1).Save(ctx)
	} else if err == nil {
		if expectedVersion == 0 || item.Version != expectedVersion {
			return rollback(biz.ErrPartnerInvoiceProfileVersionConflict)
		}
		item, err = item.Update().SetInvoiceTitle(profile.InvoiceTitle).SetTaxpayerIdentificationNo(profile.TaxpayerIdentificationNo).SetRegisteredAddress(profile.RegisteredAddress).SetRegisteredPhone(profile.RegisteredPhone).SetBankName(profile.BankName).SetBankAccount(profile.BankAccount).SetDefaultInvoiceType(profileent.DefaultInvoiceType(profile.DefaultInvoiceType)).SetVersion(item.Version + 1).Save(ctx)
	}
	if err != nil {
		return rollback(err)
	}
	if err = writeAudit(ctx, tx.AuditLog, audit); err != nil {
		return rollback(err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return partnerInvoiceProfileToBiz(item), nil
}

func partnerInvoiceProfileToBiz(item *ent.PartnerInvoiceProfile) *biz.PartnerInvoiceProfile {
	return &biz.PartnerInvoiceProfile{ID: item.ID, OrganizationID: item.OrganizationID, PartnerID: item.PartnerID, InvoiceTitle: item.InvoiceTitle, TaxpayerIdentificationNo: item.TaxpayerIdentificationNo, RegisteredAddress: item.RegisteredAddress, RegisteredPhone: item.RegisteredPhone, BankName: item.BankName, BankAccount: item.BankAccount, DefaultInvoiceType: biz.FinanceInvoiceType(item.DefaultInvoiceType), Version: item.Version}
}

var _ biz.PartnerInvoiceProfileRepo = (*partnerInvoiceProfileRepo)(nil)
