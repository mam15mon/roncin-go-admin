package biz

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var ErrReferenceDataInvalidArgument = errors.BadRequest("REFERENCE_DATA_INVALID_ARGUMENT", "基础字典查询参数不合法")

var administrativeRegionCodePattern = regexp.MustCompile(`^\d{12}$`)

type Currency struct {
	ID        uuid.UUID
	Code      string
	Name      string
	Symbol    string
	MinorUnit int
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdministrativeRegion struct {
	ID            uuid.UUID
	Code          string
	Name          string
	Level         int
	ParentCode    *string
	RegionType    *string
	Source        string
	SourceVersion *string
	Enabled       bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AdministrativeRegionQuery struct {
	Level      int
	ParentCode *string
	Keyword    string
}

type ReferenceDataRepo interface {
	ListCurrencies(context.Context) ([]*Currency, error)
	ListAdministrativeRegions(context.Context, AdministrativeRegionQuery) ([]*AdministrativeRegion, error)
}

type ReferenceDataUsecase struct {
	repo ReferenceDataRepo
}

func NewReferenceDataUsecase(repo ReferenceDataRepo) *ReferenceDataUsecase {
	return &ReferenceDataUsecase{repo: repo}
}

func (uc *ReferenceDataUsecase) ListCurrencies(ctx context.Context) ([]*Currency, error) {
	return uc.repo.ListCurrencies(ctx)
}

func (uc *ReferenceDataUsecase) ListAdministrativeRegions(ctx context.Context, query AdministrativeRegionQuery) ([]*AdministrativeRegion, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Level < 0 || query.Level > 3 {
		return nil, ErrReferenceDataInvalidArgument
	}
	if query.Level <= 1 {
		if query.ParentCode != nil {
			return nil, ErrReferenceDataInvalidArgument
		}
	} else {
		if query.ParentCode == nil {
			return nil, ErrReferenceDataInvalidArgument
		}
		parentCode := strings.TrimSpace(*query.ParentCode)
		if !administrativeRegionCodePattern.MatchString(parentCode) {
			return nil, ErrReferenceDataInvalidArgument
		}
		query.ParentCode = &parentCode
	}
	if len([]rune(query.Keyword)) > 100 {
		return nil, ErrReferenceDataInvalidArgument
	}
	return uc.repo.ListAdministrativeRegions(ctx, query)
}
