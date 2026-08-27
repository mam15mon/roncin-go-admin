package data

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financefeeledgerpreference"
)

type feeLedgerPreferenceRepo struct {
	data *Data
}

type feeLedgerColumnPreferencePO struct {
	FieldKey string `json:"fieldKey"`
	Visible  bool   `json:"visible"`
}

type feeLedgerRowColorsPO struct {
	Unbilled             string `json:"unbilled"`
	UnverifiedUninvoiced string `json:"unverifiedUninvoiced"`
	InvoicedUnverified   string `json:"invoicedUnverified"`
	VerifiedUninvoiced   string `json:"verifiedUninvoiced"`
	Completed            string `json:"completed"`
}

func NewFeeLedgerPreferenceRepo(data *Data) biz.FeeLedgerPreferenceRepo {
	return &feeLedgerPreferenceRepo{data: data}
}

func (repo *feeLedgerPreferenceRepo) Get(ctx context.Context, organizationID, userID uuid.UUID) (*biz.FeeLedgerPreference, error) {
	entity, err := repo.data.db.FinanceFeeLedgerPreference.Query().Where(
		financefeeledgerpreference.OrganizationIDEQ(organizationID),
		financefeeledgerpreference.UserIDEQ(userID),
	).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return feeLedgerPreferenceToBiz(entity)
}

func (repo *feeLedgerPreferenceRepo) Save(ctx context.Context, value *biz.FeeLedgerPreference) (*biz.FeeLedgerPreference, error) {
	columns, colors, err := encodeFeeLedgerPreference(value)
	if err != nil {
		return nil, err
	}
	existing, err := repo.data.db.FinanceFeeLedgerPreference.Query().Where(
		financefeeledgerpreference.OrganizationIDEQ(value.OrganizationID),
		financefeeledgerpreference.UserIDEQ(value.UserID),
	).Only(ctx)
	if ent.IsNotFound(err) {
		if value.Version != 0 {
			return nil, biz.ErrFeeLedgerPreferenceConflict
		}
		builder := repo.data.db.FinanceFeeLedgerPreference.Create().
			SetOrganizationID(value.OrganizationID).
			SetUserID(value.UserID).
			SetColumns(columns).
			SetPageSize(value.PageSize).
			SetRowColors(colors).
			SetVersion(1)
		if value.SortField != "" {
			builder.SetSortField(value.SortField).SetSortDirection(financefeeledgerpreference.SortDirection(value.SortDirection))
		}
		created, createErr := builder.Save(ctx)
		if ent.IsConstraintError(createErr) {
			return nil, biz.ErrFeeLedgerPreferenceConflict
		}
		if createErr != nil {
			return nil, createErr
		}
		return feeLedgerPreferenceToBiz(created)
	}
	if err != nil {
		return nil, err
	}
	if value.Version == 0 || value.Version != existing.Version {
		return nil, biz.ErrFeeLedgerPreferenceConflict
	}

	builder := repo.data.db.FinanceFeeLedgerPreference.UpdateOneID(existing.ID).
		Where(financefeeledgerpreference.VersionEQ(value.Version)).
		SetColumns(columns).
		SetPageSize(value.PageSize).
		SetRowColors(colors).
		SetVersion(value.Version + 1)
	if value.SortField == "" {
		builder.ClearSortField().ClearSortDirection()
	} else {
		builder.SetSortField(value.SortField).SetSortDirection(financefeeledgerpreference.SortDirection(value.SortDirection))
	}
	updated, err := builder.Save(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrFeeLedgerPreferenceConflict
	}
	if err != nil {
		return nil, err
	}
	return feeLedgerPreferenceToBiz(updated)
}

func (repo *feeLedgerPreferenceRepo) Delete(ctx context.Context, organizationID, userID uuid.UUID, version uint64) error {
	existing, err := repo.data.db.FinanceFeeLedgerPreference.Query().Where(
		financefeeledgerpreference.OrganizationIDEQ(organizationID),
		financefeeledgerpreference.UserIDEQ(userID),
	).Only(ctx)
	if ent.IsNotFound(err) {
		if version == 0 {
			return nil
		}
		return biz.ErrFeeLedgerPreferenceConflict
	}
	if err != nil {
		return err
	}
	if version == 0 || version != existing.Version {
		return biz.ErrFeeLedgerPreferenceConflict
	}
	affected, err := repo.data.db.FinanceFeeLedgerPreference.Delete().Where(
		financefeeledgerpreference.IDEQ(existing.ID),
		financefeeledgerpreference.VersionEQ(version),
	).Exec(ctx)
	if err != nil {
		return err
	}
	if affected != 1 {
		return biz.ErrFeeLedgerPreferenceConflict
	}
	return nil
}

func encodeFeeLedgerPreference(value *biz.FeeLedgerPreference) (json.RawMessage, json.RawMessage, error) {
	columns := make([]feeLedgerColumnPreferencePO, 0, len(value.Columns))
	for _, column := range value.Columns {
		columns = append(columns, feeLedgerColumnPreferencePO{FieldKey: column.FieldKey, Visible: column.Visible})
	}
	columnsJSON, err := json.Marshal(columns)
	if err != nil {
		return nil, nil, err
	}
	colorsJSON, err := json.Marshal(feeLedgerRowColorsPO{
		Unbilled:             value.RowColors.Unbilled,
		UnverifiedUninvoiced: value.RowColors.UnverifiedUninvoiced,
		InvoicedUnverified:   value.RowColors.InvoicedUnverified,
		VerifiedUninvoiced:   value.RowColors.VerifiedUninvoiced,
		Completed:            value.RowColors.Completed,
	})
	if err != nil {
		return nil, nil, err
	}
	return columnsJSON, colorsJSON, nil
}

func feeLedgerPreferenceToBiz(entity *ent.FinanceFeeLedgerPreference) (*biz.FeeLedgerPreference, error) {
	var columns []feeLedgerColumnPreferencePO
	if err := json.Unmarshal(entity.Columns, &columns); err != nil {
		return nil, err
	}
	var colors feeLedgerRowColorsPO
	if err := json.Unmarshal(entity.RowColors, &colors); err != nil {
		return nil, err
	}
	value := &biz.FeeLedgerPreference{
		OrganizationID: entity.OrganizationID,
		UserID:         entity.UserID,
		Columns:        make([]biz.FeeLedgerColumnPreference, 0, len(columns)),
		PageSize:       entity.PageSize,
		RowColors: biz.FeeLedgerRowColors{
			Unbilled:             colors.Unbilled,
			UnverifiedUninvoiced: colors.UnverifiedUninvoiced,
			InvoicedUnverified:   colors.InvoicedUnverified,
			VerifiedUninvoiced:   colors.VerifiedUninvoiced,
			Completed:            colors.Completed,
		},
		Version:    entity.Version,
		Customized: true,
		UpdatedAt:  entity.UpdatedAt,
	}
	for _, column := range columns {
		value.Columns = append(value.Columns, biz.FeeLedgerColumnPreference{FieldKey: column.FieldKey, Visible: column.Visible})
	}
	if entity.SortField != nil {
		value.SortField = *entity.SortField
	}
	if entity.SortDirection != nil {
		value.SortDirection = string(*entity.SortDirection)
	}
	return value, nil
}

var _ biz.FeeLedgerPreferenceRepo = (*feeLedgerPreferenceRepo)(nil)
