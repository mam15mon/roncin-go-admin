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
	Page       int
	PageSize   int
}

type ReferenceDataRepo interface {
	ListCurrencies(context.Context) ([]*Currency, error)
	SearchCurrencies(context.Context, SelectorListOptions) (*PagedList[*Currency], error)
	ListAdministrativeRegions(context.Context, AdministrativeRegionQuery) (*PagedList[*AdministrativeRegion], error)
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

func (uc *ReferenceDataUsecase) SearchCurrencies(ctx context.Context, options SelectorListOptions) (*PagedList[*Currency], error) {
	options.Keyword = strings.TrimSpace(options.Keyword)
	if !ValidListPagination(options.Page, options.PageSize) || len([]rune(options.Keyword)) > 100 {
		return nil, ErrReferenceDataInvalidArgument
	}
	return uc.repo.SearchCurrencies(ctx, options)
}

func (uc *ReferenceDataUsecase) ListAdministrativeRegions(ctx context.Context, query AdministrativeRegionQuery) (*PagedList[*AdministrativeRegion], error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Level < 0 || query.Level > 3 || !ValidListPagination(query.Page, query.PageSize) {
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
