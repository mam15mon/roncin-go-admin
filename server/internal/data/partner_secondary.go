package data

import (
	"context"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	partnercontractent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontract"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"

	"github.com/google/uuid"
)

type partnerAccountRepo struct{ data *Data }

func NewPartnerAccountRepo(data *Data) biz.PartnerAccountRepo { return &partnerAccountRepo{data: data} }

func (r *partnerAccountRepo) role(ctx context.Context, organizationID, partnerID uuid.UUID) (*ent.PartnerRole, error) {
	role, err := r.data.db.PartnerRole.Query().Where(
		partnerroleent.PartnerIDEQ(partnerID),
		partnerroleent.RoleTypeEQ(partnerroleent.RoleTypeCustomer),
		partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerAccountInvalidArgument
		}
		return nil, err
	}
	return role, nil
}

func (r *partnerAccountRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, enabled *bool) ([]*biz.PartnerAccount, error) {
	role, err := r.role(ctx, organizationID, partnerID)
	if err != nil {
		return nil, err
	}
	query := r.data.db.PartnerAccount.Query().Where(partneraccountent.PartnerRoleIDEQ(role.ID))
	if enabled != nil {
		status := partneraccountent.StatusInactive
		if *enabled {
			status = partneraccountent.StatusActive
		}
		query.Where(partneraccountent.StatusEQ(status))
	}
	items, err := query.Order(partneraccountent.ByIsDefault(), partneraccountent.ByCurrency()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerAccount, 0, len(items))
	for _, item := range items {
		result = append(result, partnerAccountToBiz(item))
	}
	return result, nil
}

func (r *partnerAccountRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, input *biz.PartnerAccount, audit *biz.AuditEvent) (*biz.PartnerAccount, error) {
	role, err := r.role(ctx, organizationID, partnerID)
	if err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if exists, err := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	} else if !exists {
		_ = tx.Rollback()
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	if input.IsDefault {
		if _, err := tx.PartnerAccount.Update().Where(partneraccountent.PartnerRoleIDEQ(role.ID), partneraccountent.AccountTypeEQ(partneraccountent.AccountTypeCustomerSettlement), partneraccountent.IsDefaultEQ(true)).SetIsDefault(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	created := tx.PartnerAccount.Create().
		SetPartnerRoleID(role.ID).
		SetAccountType(partneraccountent.AccountTypeCustomerSettlement).
		SetCurrency(input.Currency).
		SetBankName(input.BankName).
		SetBankAccount(input.BankAccount).
		SetSwiftCode(input.SwiftCode).
		SetIsDefault(input.IsDefault).
		SetStatus(partneraccountent.Status(input.Status)).
		SetRemark(input.Remark)
	item, err := created.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapPartnerSecondaryConstraint(err)
	}
	audit.Details["account.id"] = item.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	item, err = r.data.db.PartnerAccount.Get(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return partnerAccountToBiz(item), nil
}

func (r *partnerAccountRepo) Update(ctx context.Context, organizationID, partnerID, id uuid.UUID, input *biz.PartnerAccount, audit *biz.AuditEvent) (*biz.PartnerAccount, error) {
	role, err := r.role(ctx, organizationID, partnerID)
	if err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if exists, err := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	} else if !exists {
		_ = tx.Rollback()
		return nil, biz.ErrPartnerAccountInvalidArgument
	}
	existing, err := tx.PartnerAccount.Query().Where(partneraccountent.IDEQ(id), partneraccountent.PartnerRoleIDEQ(role.ID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerAccountNotFound
		}
		return nil, err
	}
	if input.IsDefault {
		if _, err := tx.PartnerAccount.Update().Where(partneraccountent.PartnerRoleIDEQ(role.ID), partneraccountent.AccountTypeEQ(partneraccountent.AccountTypeCustomerSettlement), partneraccountent.IsDefaultEQ(true), partneraccountent.IDNEQ(id)).SetIsDefault(false).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	update := existing.Update().SetCurrency(input.Currency).SetBankName(input.BankName).SetBankAccount(input.BankAccount).SetSwiftCode(input.SwiftCode).SetIsDefault(input.IsDefault).SetStatus(partneraccountent.Status(input.Status)).SetRemark(input.Remark)
	updated, err := update.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapPartnerSecondaryConstraint(err)
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return partnerAccountToBiz(updated), nil
}

type partnerContractRepo struct{ data *Data }

func NewPartnerContractRepo(data *Data) biz.PartnerContractRepo {
	return &partnerContractRepo{data: data}
}

func (r *partnerContractRepo) partner(ctx context.Context, organizationID, partnerID uuid.UUID) (*ent.Partner, error) {
	item, err := r.data.db.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerContractInvalidArgument
		}
		return nil, err
	}
	return item, nil
}

func (r *partnerContractRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, status *biz.PartnerContractStatus) ([]*biz.PartnerContract, error) {
	if _, err := r.partner(ctx, organizationID, partnerID); err != nil {
		return nil, err
	}
	query := r.data.db.PartnerContract.Query().Where(partnercontractent.PartnerIDEQ(partnerID))
	if status != nil {
		query.Where(partnercontractent.StatusEQ(partnercontractent.Status(*status)))
	}
	items, err := query.Order(partnercontractent.ByStartDate()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerContract, 0, len(items))
	for _, item := range items {
		result = append(result, partnerContractToBiz(item))
	}
	return result, nil
}

func (r *partnerContractRepo) Get(ctx context.Context, organizationID, partnerID, id uuid.UUID) (*biz.PartnerContract, error) {
	item, err := r.data.db.PartnerContract.Query().Where(
		partnercontractent.IDEQ(id),
		partnercontractent.PartnerIDEQ(partnerID),
		partnercontractent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerContractNotFound
		}
		return nil, err
	}
	return partnerContractToBiz(item), nil
}

func (r *partnerContractRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, input *biz.PartnerContract, audit *biz.AuditEvent) (*biz.PartnerContract, error) {
	if _, err := r.partner(ctx, organizationID, partnerID); err != nil {
		return nil, err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := tx.PartnerContract.Create().SetPartnerID(partnerID).SetContractNo(input.ContractNo).SetName(input.Name).SetStatus(partnercontractent.Status(input.Status)).SetStartDate(input.StartDate).SetEndDate(input.EndDate).SetPaymentTerms(input.PaymentTerms).SetDisputeResolution(input.DisputeResolution).SetOtherNotes(input.OtherNotes).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapPartnerSecondaryConstraint(err)
	}
	audit.Details["contract.id"] = created.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return partnerContractToBiz(created), nil
}

func (r *partnerContractRepo) Update(ctx context.Context, organizationID, partnerID, id uuid.UUID, expectedStatus biz.PartnerContractStatus, input *biz.PartnerContract, audit *biz.AuditEvent) (*biz.PartnerContract, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := tx.PartnerContract.Update().Where(
		partnercontractent.IDEQ(id),
		partnercontractent.PartnerIDEQ(partnerID),
		partnercontractent.StatusEQ(partnercontractent.Status(expectedStatus)),
		partnercontractent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	).
		SetName(input.Name).
		SetStatus(partnercontractent.Status(input.Status)).
		SetStartDate(input.StartDate).
		SetEndDate(input.EndDate).
		SetPaymentTerms(input.PaymentTerms).
		SetDisputeResolution(input.DisputeResolution).
		SetOtherNotes(input.OtherNotes).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapPartnerSecondaryConstraint(err)
	}
	if updated == 0 {
		_ = tx.Rollback()
		return nil, biz.ErrPartnerContractStatusConflict
	}
	item, err := tx.PartnerContract.Get(ctx, id)
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
	return partnerContractToBiz(item), nil
}

func mapPartnerSecondaryConstraint(err error) error {
	if !ent.IsConstraintError(err) {
		return err
	}
	if strings.Contains(err.Error(), "partner_contract_no_key") {
		return biz.ErrPartnerContractNoExists
	}
	if strings.Contains(err.Error(), "partner_account_default_key") {
		return biz.ErrPartnerAccountDefaultConflict
	}
	return err
}

func partnerAccountToBiz(item *ent.PartnerAccount) *biz.PartnerAccount {
	return &biz.PartnerAccount{ID: item.ID, PartnerRoleID: item.PartnerRoleID, AccountType: string(item.AccountType), Currency: item.Currency, BankName: item.BankName, BankAccount: item.BankAccount, SwiftCode: item.SwiftCode, IsDefault: item.IsDefault, Status: biz.PartnerAccountStatus(item.Status), Remark: item.Remark, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func partnerContractToBiz(item *ent.PartnerContract) *biz.PartnerContract {
	return &biz.PartnerContract{ID: item.ID, PartnerID: item.PartnerID, ContractNo: item.ContractNo, Name: item.Name, Status: biz.PartnerContractStatus(item.Status), StartDate: item.StartDate, EndDate: item.EndDate, PaymentTerms: item.PaymentTerms, DisputeResolution: item.DisputeResolution, OtherNotes: item.OtherNotes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
