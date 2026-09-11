package data

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraccountent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneraccount"
	partnercontractent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontract"

	"github.com/google/uuid"
)

type partnerAccountRepo struct{ data *Data }

func NewPartnerAccountRepo(data *Data) biz.PartnerAccountRepo { return &partnerAccountRepo{data: data} }

func (r *partnerAccountRepo) partner(ctx context.Context, organizationID, partnerID uuid.UUID) (*ent.Partner, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	partner, err := client.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrPartnerAccountInvalidArgument, nil)
	}
	return partner, nil
}

func (r *partnerAccountRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, filter biz.PartnerAccountFilter) ([]*biz.PartnerAccount, error) {
	partner, err := r.partner(ctx, organizationID, partnerID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(partner.ID))
	if filter.Enabled != nil {
		query.Where(partneraccountent.EnabledEQ(*filter.Enabled))
	}
	if filter.Usage != "" {
		query.Where(partneraccountent.UsageEQ(partneraccountent.Usage(filter.Usage)))
	}
	if filter.Currency != "" {
		query.Where(partneraccountent.CurrencyEQ(filter.Currency))
	}
	items, err := query.Order(partneraccountent.ByIsDefaultReceivable(), partneraccountent.ByIsDefaultPayable(), partneraccountent.ByCurrency()).All(ctx)
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
	partner, err := r.partner(ctx, organizationID, partnerID)
	if err != nil {
		return nil, err
	}
	var item *ent.PartnerAccount
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		exists, queryErr := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrPartnerAccountInvalidArgument
		}
		if err := r.clearPartnerAccountDefaults(ctx, tx, partner.ID, input.Currency, uuid.Nil, input.IsDefaultReceivable, input.IsDefaultPayable); err != nil {
			return err
		}
		created := tx.PartnerAccount.Create().
			SetPartnerID(partner.ID).
			SetName(input.Name).
			SetAccountHolder(input.AccountHolder).
			SetCurrency(input.Currency).
			SetBankName(input.BankName).
			SetAccountNo(input.AccountNo).
			SetSwiftCode(input.SwiftCode).
			SetUsage(partneraccountent.Usage(input.Usage)).
			SetIsDefaultReceivable(input.IsDefaultReceivable).
			SetIsDefaultPayable(input.IsDefaultPayable).
			SetEnabled(input.Enabled).
			SetRemark(input.Remark)
		var createErr error
		item, createErr = created.Save(ctx)
		if createErr != nil {
			return mapEntConstraints(createErr,
				entConstraintMapping{name: "partner_account_default_receivable_key", domainErr: biz.ErrPartnerAccountDefaultConflict},
				entConstraintMapping{name: "partner_account_default_payable_key", domainErr: biz.ErrPartnerAccountDefaultConflict},
			)
		}
		audit.Details["account.id"] = item.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err = client.PartnerAccount.Get(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return partnerAccountToBiz(item), nil
}

func (r *partnerAccountRepo) Update(ctx context.Context, organizationID, partnerID, id uuid.UUID, input *biz.PartnerAccount, audit *biz.AuditEvent) (*biz.PartnerAccount, error) {
	partner, err := r.partner(ctx, organizationID, partnerID)
	if err != nil {
		return nil, err
	}
	var updated *ent.PartnerAccount
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		exists, queryErr := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
		if queryErr != nil {
			return queryErr
		}
		if !exists {
			return biz.ErrPartnerAccountInvalidArgument
		}
		lockedAccounts, queryErr := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(partner.ID)).Order(partneraccountent.ByID()).ForUpdate().All(ctx)
		if queryErr != nil {
			return queryErr
		}
		var existing *ent.PartnerAccount
		for _, item := range lockedAccounts {
			if item.ID == id {
				existing = item
				break
			}
		}
		if existing == nil {
			return biz.ErrPartnerAccountNotFound
		}
		if err := r.clearPartnerAccountDefaults(ctx, tx, partner.ID, input.Currency, id, input.IsDefaultReceivable, input.IsDefaultPayable); err != nil {
			return err
		}
		var updateErr error
		updated, updateErr = existing.Update().SetName(input.Name).SetAccountHolder(input.AccountHolder).SetCurrency(input.Currency).SetBankName(input.BankName).SetAccountNo(input.AccountNo).SetSwiftCode(input.SwiftCode).SetUsage(partneraccountent.Usage(input.Usage)).SetIsDefaultReceivable(input.IsDefaultReceivable).SetIsDefaultPayable(input.IsDefaultPayable).SetEnabled(input.Enabled).SetRemark(input.Remark).Save(ctx)
		if updateErr != nil {
			return mapEntConstraints(updateErr,
				entConstraintMapping{name: "partner_account_default_receivable_key", domainErr: biz.ErrPartnerAccountDefaultConflict},
				entConstraintMapping{name: "partner_account_default_payable_key", domainErr: biz.ErrPartnerAccountDefaultConflict},
			)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return partnerAccountToBiz(updated), nil
}

func (r *partnerAccountRepo) clearPartnerAccountDefaults(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, currency string, excludedID uuid.UUID, receivable, payable bool) error {
	items, err := tx.PartnerAccount.Query().Where(partneraccountent.PartnerIDEQ(partnerID), partneraccountent.CurrencyEQ(currency)).Order(partneraccountent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if item.ID != excludedID {
			ids = append(ids, item.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	update := tx.PartnerAccount.Update().Where(partneraccountent.IDIn(ids...))
	if receivable {
		update.SetIsDefaultReceivable(false)
	}
	if payable {
		update.SetIsDefaultPayable(false)
	}
	if !receivable && !payable {
		return nil
	}
	_, err = update.Save(ctx)
	return err
}

type partnerContractRepo struct{ data *Data }

func NewPartnerContractRepo(data *Data) biz.PartnerContractRepo {
	return &partnerContractRepo{data: data}
}

func (r *partnerContractRepo) partner(ctx context.Context, organizationID, partnerID uuid.UUID) (*ent.Partner, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrPartnerContractInvalidArgument, nil)
	}
	return item, nil
}

func (r *partnerContractRepo) List(ctx context.Context, organizationID, partnerID uuid.UUID, status *biz.PartnerContractStatus) ([]*biz.PartnerContract, error) {
	if _, err := r.partner(ctx, organizationID, partnerID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.PartnerContract.Query().Where(partnercontractent.PartnerIDEQ(partnerID))
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
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.PartnerContract.Query().Where(
		partnercontractent.IDEQ(id),
		partnercontractent.PartnerIDEQ(partnerID),
		partnercontractent.HasPartnerWith(partnerent.OrganizationIDEQ(organizationID)),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrPartnerContractNotFound, nil)
	}
	return partnerContractToBiz(item), nil
}

func (r *partnerContractRepo) Create(ctx context.Context, organizationID, partnerID uuid.UUID, input *biz.PartnerContract, audit *biz.AuditEvent) (*biz.PartnerContract, error) {
	if _, err := r.partner(ctx, organizationID, partnerID); err != nil {
		return nil, err
	}
	var created *ent.PartnerContract
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var createErr error
		created, createErr = tx.PartnerContract.Create().SetPartnerID(partnerID).SetContractNo(input.ContractNo).SetName(input.Name).SetStatus(partnercontractent.Status(input.Status)).SetStartDate(input.StartDate).SetEndDate(input.EndDate).SetPaymentTerms(input.PaymentTerms).SetDisputeResolution(input.DisputeResolution).SetOtherNotes(input.OtherNotes).Save(ctx)
		if createErr != nil {
			return mapEntConstraint(createErr, "partner_contract_no_key", biz.ErrPartnerContractNoExists)
		}
		audit.Details["contract.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return partnerContractToBiz(created), nil
}

func (r *partnerContractRepo) Update(ctx context.Context, organizationID, partnerID, id uuid.UUID, expectedStatus biz.PartnerContractStatus, input *biz.PartnerContract, audit *biz.AuditEvent) (*biz.PartnerContract, error) {
	var item *ent.PartnerContract
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		updated, updateErr := tx.PartnerContract.Update().Where(
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
		if updateErr != nil {
			return mapEntConstraint(updateErr, "partner_contract_no_key", biz.ErrPartnerContractNoExists)
		}
		if updated == 0 {
			return biz.ErrPartnerContractStatusConflict
		}
		var getErr error
		item, getErr = tx.PartnerContract.Get(ctx, id)
		if getErr != nil {
			return getErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return partnerContractToBiz(item), nil
}

func partnerAccountToBiz(item *ent.PartnerAccount) *biz.PartnerAccount {
	return &biz.PartnerAccount{ID: item.ID, PartnerID: item.PartnerID, Name: item.Name, AccountHolder: item.AccountHolder, Currency: item.Currency, BankName: item.BankName, AccountNo: item.AccountNo, SwiftCode: item.SwiftCode, Usage: biz.PartnerAccountUsage(item.Usage), IsDefaultReceivable: item.IsDefaultReceivable, IsDefaultPayable: item.IsDefaultPayable, Enabled: item.Enabled, Remark: item.Remark, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func partnerContractToBiz(item *ent.PartnerContract) *biz.PartnerContract {
	return &biz.PartnerContract{ID: item.ID, PartnerID: item.PartnerID, ContractNo: item.ContractNo, Name: item.Name, Status: biz.PartnerContractStatus(item.Status), StartDate: item.StartDate, EndDate: item.EndDate, PaymentTerms: item.PaymentTerms, DisputeResolution: item.DisputeResolution, OtherNotes: item.OtherNotes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
