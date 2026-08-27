package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrFeeLedgerPreferenceInvalidArgument = errors.BadRequest("FEE_LEDGER_PREFERENCE_INVALID_ARGUMENT", "费用明细表头设置不合法")
	ErrFeeLedgerPreferenceConflict        = errors.Conflict("FEE_LEDGER_PREFERENCE_CONFLICT", "费用明细表头设置已被更新，请刷新后重试")
)

const (
	FeeLedgerSortAscending  = "ASC"
	FeeLedgerSortDescending = "DESC"
)

var (
	feeLedgerFieldKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	feeLedgerColorPattern    = regexp.MustCompile(`^#[0-9A-F]{6}$`)
)

type FeeLedgerColumnPreference struct {
	FieldKey string
	Visible  bool
}

type FeeLedgerRowColors struct {
	Unbilled                    string
	UnverifiedUninvoiced        string
	InvoicedUnverified          string
	VerifiedUninvoiced          string
	InvoicedPartiallyVerified   string
	PartiallyVerifiedUninvoiced string
	Completed                   string
}

type FeeLedgerPreference struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Columns        []FeeLedgerColumnPreference
	PageSize       int
	SortField      string
	SortDirection  string
	RowColors      FeeLedgerRowColors
	Version        uint64
	Customized     bool
	UpdatedAt      time.Time
}

type FeeLedgerPreferenceRepo interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (*FeeLedgerPreference, error)
	Save(context.Context, *FeeLedgerPreference) (*FeeLedgerPreference, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, uint64) error
}

type FeeLedgerPreferenceUsecase struct {
	repo FeeLedgerPreferenceRepo
}

func NewFeeLedgerPreferenceUsecase(repo FeeLedgerPreferenceRepo) *FeeLedgerPreferenceUsecase {
	return &FeeLedgerPreferenceUsecase{repo: repo}
}

func (uc *FeeLedgerPreferenceUsecase) Get(ctx context.Context, organizationID, userID uuid.UUID) (*FeeLedgerPreference, error) {
	if organizationID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrFeeLedgerPreferenceInvalidArgument
	}
	preference, err := uc.repo.Get(ctx, organizationID, userID)
	if err != nil {
		return nil, err
	}
	if preference != nil {
		return preference, nil
	}
	return &FeeLedgerPreference{
		OrganizationID: organizationID,
		UserID:         userID,
		PageSize:       40,
		RowColors:      defaultFeeLedgerRowColors(),
		Customized:     false,
	}, nil
}

func (uc *FeeLedgerPreferenceUsecase) Save(ctx context.Context, organizationID, userID uuid.UUID, input *FeeLedgerPreference) (*FeeLedgerPreference, error) {
	normalized, err := normalizeFeeLedgerPreference(organizationID, userID, input)
	if err != nil {
		return nil, err
	}
	return uc.repo.Save(ctx, normalized)
}

func (uc *FeeLedgerPreferenceUsecase) Reset(ctx context.Context, organizationID, userID uuid.UUID, version uint64) (*FeeLedgerPreference, error) {
	if organizationID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrFeeLedgerPreferenceInvalidArgument
	}
	if err := uc.repo.Delete(ctx, organizationID, userID, version); err != nil {
		return nil, err
	}
	return &FeeLedgerPreference{
		OrganizationID: organizationID,
		UserID:         userID,
		PageSize:       40,
		RowColors:      defaultFeeLedgerRowColors(),
		Customized:     false,
	}, nil
}

func defaultFeeLedgerRowColors() FeeLedgerRowColors {
	return FeeLedgerRowColors{
		Unbilled:                    "#FFF7E6",
		UnverifiedUninvoiced:        "#FFFBE6",
		InvoicedUnverified:          "#E6F4FF",
		VerifiedUninvoiced:          "#F9F0FF",
		InvoicedPartiallyVerified:   "#E6FFFB",
		PartiallyVerifiedUninvoiced: "#FFF0F6",
		Completed:                   "#F6FFED",
	}
}

func normalizeFeeLedgerPreference(organizationID, userID uuid.UUID, input *FeeLedgerPreference) (*FeeLedgerPreference, error) {
	if organizationID == uuid.Nil || userID == uuid.Nil || input == nil || len(input.Columns) == 0 || len(input.Columns) > 200 {
		return nil, ErrFeeLedgerPreferenceInvalidArgument
	}
	if input.PageSize != 40 && input.PageSize != 60 && input.PageSize != 100 {
		return nil, ErrFeeLedgerPreferenceInvalidArgument
	}

	result := &FeeLedgerPreference{
		OrganizationID: organizationID,
		UserID:         userID,
		Columns:        make([]FeeLedgerColumnPreference, 0, len(input.Columns)),
		PageSize:       input.PageSize,
		SortField:      strings.TrimSpace(input.SortField),
		SortDirection:  strings.ToUpper(strings.TrimSpace(input.SortDirection)),
		RowColors: FeeLedgerRowColors{
			Unbilled:                    strings.ToUpper(strings.TrimSpace(input.RowColors.Unbilled)),
			UnverifiedUninvoiced:        strings.ToUpper(strings.TrimSpace(input.RowColors.UnverifiedUninvoiced)),
			InvoicedUnverified:          strings.ToUpper(strings.TrimSpace(input.RowColors.InvoicedUnverified)),
			VerifiedUninvoiced:          strings.ToUpper(strings.TrimSpace(input.RowColors.VerifiedUninvoiced)),
			InvoicedPartiallyVerified:   strings.ToUpper(strings.TrimSpace(input.RowColors.InvoicedPartiallyVerified)),
			PartiallyVerifiedUninvoiced: strings.ToUpper(strings.TrimSpace(input.RowColors.PartiallyVerifiedUninvoiced)),
			Completed:                   strings.ToUpper(strings.TrimSpace(input.RowColors.Completed)),
		},
		Version:    input.Version,
		Customized: true,
	}

	seen := make(map[string]struct{}, len(input.Columns))
	hasVisibleColumn := false
	for _, column := range input.Columns {
		key := strings.TrimSpace(column.FieldKey)
		if !feeLedgerFieldKeyPattern.MatchString(key) || len(key) > 64 {
			return nil, ErrFeeLedgerPreferenceInvalidArgument
		}
		if _, exists := seen[key]; exists {
			return nil, ErrFeeLedgerPreferenceInvalidArgument
		}
		seen[key] = struct{}{}
		hasVisibleColumn = hasVisibleColumn || column.Visible
		result.Columns = append(result.Columns, FeeLedgerColumnPreference{FieldKey: key, Visible: column.Visible})
	}
	if !hasVisibleColumn {
		return nil, ErrFeeLedgerPreferenceInvalidArgument
	}

	if result.SortField == "" {
		if result.SortDirection != "" {
			return nil, ErrFeeLedgerPreferenceInvalidArgument
		}
	} else {
		if _, exists := seen[result.SortField]; !exists || (result.SortDirection != FeeLedgerSortAscending && result.SortDirection != FeeLedgerSortDescending) {
			return nil, ErrFeeLedgerPreferenceInvalidArgument
		}
	}
	for _, color := range []string{
		result.RowColors.Unbilled,
		result.RowColors.UnverifiedUninvoiced,
		result.RowColors.InvoicedUnverified,
		result.RowColors.VerifiedUninvoiced,
		result.RowColors.InvoicedPartiallyVerified,
		result.RowColors.PartiallyVerifiedUninvoiced,
		result.RowColors.Completed,
	} {
		if !feeLedgerColorPattern.MatchString(color) {
			return nil, ErrFeeLedgerPreferenceInvalidArgument
		}
	}
	return result, nil
}
