package data

import (
	"context"
	"sort"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"

	"github.com/google/uuid"
)

func (r *adminRepo) ListOrganizations(ctx context.Context) ([]*biz.AdminOrganization, error) {
	items, err := r.data.db.Organization.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	result := make([]*biz.AdminOrganization, 0, len(items))
	for _, item := range items {
		converted, convertErr := organizationToBizWithCurrency(item, items)
		if convertErr != nil {
			return nil, convertErr
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *adminRepo) GetOrganization(ctx context.Context, id uuid.UUID) (*biz.AdminOrganization, error) {
	item, err := r.data.db.Organization.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminOrganizationNotFound
		}
		return nil, err
	}
	return r.organizationToBiz(ctx, item)
}

func (r *adminRepo) CreateOrganization(ctx context.Context, input *biz.AdminOrganization, audit *biz.AuditEvent) (*biz.AdminOrganization, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	create := tx.Organization.Create().SetCode(input.Code).SetName(input.Name).SetKind(organization.Kind(input.Kind))
	if input.Kind == biz.OrganizationKindCompany || input.Kind == biz.OrganizationKindHeadquarters {
		if err := validateOrganizationCurrency(ctx, tx.Currency, input.BaseCurrency); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		create.SetBaseCurrency(input.BaseCurrency)
	}
	if input.ParentID != nil {
		create.SetParentID(*input.ParentID)
	}
	created, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return nil, biz.ErrAdminOrganizationCodeExists
		}
		return nil, err
	}
	if err := CreateDefaultNumberRules(ctx, tx, created.ID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := CreateDefaultOrderOptions(ctx, tx, created.ID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := CreateDefaultCountries(ctx, tx, created.ID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.Details["value"] = created.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result := organizationToBiz(created)
	if result.BaseCurrency == "" {
		result.BaseCurrency = input.BaseCurrency
	}
	return result, nil
}

func (r *adminRepo) UpdateOrganization(ctx context.Context, organizationID uuid.UUID, input *biz.AdminOrganization, audit *biz.AuditEvent) (*biz.AdminOrganization, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	update := tx.Organization.UpdateOneID(input.ID).Where(organization.Or(organization.IDEQ(organizationID), organization.ParentIDEQ(organizationID))).SetName(input.Name).SetEnabled(input.Enabled)
	if input.Kind == biz.OrganizationKindHeadquarters || input.Kind == biz.OrganizationKindCompany {
		if err := validateOrganizationCurrency(ctx, tx.Currency, input.BaseCurrency); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		update.SetBaseCurrency(input.BaseCurrency)
	}
	updated, err := update.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminOrganizationNotFound
		}
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.organizationToBiz(ctx, updated)
}
func organizationToBiz(item *ent.Organization) *biz.AdminOrganization {
	result := &biz.AdminOrganization{ID: item.ID, Code: item.Code, Name: item.Name, Kind: biz.OrganizationKind(item.Kind), ParentID: item.ParentID, Enabled: item.Enabled}
	if item.BaseCurrency != nil {
		result.BaseCurrency = *item.BaseCurrency
	}
	return result
}

func organizationToBizWithCurrency(item *ent.Organization, items []*ent.Organization) (*biz.AdminOrganization, error) {
	result := organizationToBiz(item)
	if result.BaseCurrency != "" {
		return result, nil
	}
	byID := make(map[uuid.UUID]*ent.Organization, len(items))
	for _, candidate := range items {
		byID[candidate.ID] = candidate
	}
	current := item
	for current.ParentID != nil {
		parent, ok := byID[*current.ParentID]
		if !ok {
			return nil, biz.ErrAdminOrganizationCurrency
		}
		if parent.BaseCurrency != nil {
			result.BaseCurrency = *parent.BaseCurrency
			return result, nil
		}
		current = parent
	}
	return nil, biz.ErrAdminOrganizationCurrency
}

func (r *adminRepo) organizationToBiz(ctx context.Context, item *ent.Organization) (*biz.AdminOrganization, error) {
	result := organizationToBiz(item)
	current := item
	for result.BaseCurrency == "" && current.ParentID != nil {
		parent, err := r.data.db.Organization.Get(ctx, *current.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.BaseCurrency != nil {
			result.BaseCurrency = *parent.BaseCurrency
		}
		current = parent
	}
	if result.BaseCurrency == "" {
		return nil, biz.ErrAdminOrganizationCurrency
	}
	return result, nil
}

type currencyQuery interface {
	Query() *ent.CurrencyQuery
}

func validateOrganizationCurrency(ctx context.Context, client currencyQuery, code string) error {
	exists, err := client.Query().Where(currencyent.CodeEQ(code), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrAdminOrganizationCurrency
	}
	return nil
}
