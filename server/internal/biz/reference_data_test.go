package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type referenceDataRepoStub struct {
	query           AdministrativeRegionQuery
	currencyOptions SelectorListOptions
}

func (stub *referenceDataRepoStub) SearchCurrencies(_ context.Context, options SelectorListOptions) (*PagedList[*Currency], error) {
	stub.currencyOptions = options
	return &PagedList[*Currency]{Page: options.Page, PageSize: options.PageSize}, nil
}

func (stub *referenceDataRepoStub) ListCurrencies(_ context.Context, _ uuid.UUID, _ bool) ([]*Currency, error) {
	return nil, nil
}

func (stub *referenceDataRepoStub) SetCurrencyEnabled(_ context.Context, _ uuid.UUID, _ string, _ bool) (*Currency, error) {
	return nil, nil
}

func (stub *referenceDataRepoStub) ListAdministrativeRegions(_ context.Context, query AdministrativeRegionQuery) (*PagedList[*AdministrativeRegion], error) {
	stub.query = query
	return &PagedList[*AdministrativeRegion]{Page: query.Page, PageSize: query.PageSize}, nil
}

func TestReferenceDataAdministrativeRegionQuery(t *testing.T) {
	repo := &referenceDataRepoStub{}
	usecase := NewReferenceDataUsecase(repo)
	parentCode := " 310000000000 "
	if _, err := usecase.ListAdministrativeRegions(context.Background(), AdministrativeRegionQuery{
		Level: 2, ParentCode: &parentCode, Keyword: " 上海 ", Page: 1, PageSize: MaxListPageSize,
	}); err != nil {
		t.Fatalf("ListAdministrativeRegions() error = %v", err)
	}
	if repo.query.ParentCode == nil || *repo.query.ParentCode != "310000000000" || repo.query.Keyword != "上海" {
		t.Fatalf("normalized query = %#v", repo.query)
	}
}

func TestReferenceDataCurrencySearch(t *testing.T) {
	repo := &referenceDataRepoStub{}
	usecase := NewReferenceDataUsecase(repo)
	if _, err := usecase.SearchCurrencies(context.Background(), SelectorListOptions{Keyword: " 人民币 ", Page: 1, PageSize: MaxListPageSize}); err != nil {
		t.Fatalf("SearchCurrencies() error = %v", err)
	}
	if repo.currencyOptions.Keyword != "人民币" || repo.currencyOptions.PageSize != MaxListPageSize {
		t.Fatalf("normalized currency options = %#v", repo.currencyOptions)
	}
	if _, err := usecase.SearchCurrencies(context.Background(), SelectorListOptions{Page: 1, PageSize: MaxListPageSize + 1}); err != ErrReferenceDataInvalidArgument {
		t.Fatalf("SearchCurrencies() boundary error = %v, want ErrReferenceDataInvalidArgument", err)
	}
}

func TestReferenceDataAdministrativeRegionQueryWithoutLevel(t *testing.T) {
	repo := &referenceDataRepoStub{}
	usecase := NewReferenceDataUsecase(repo)
	if _, err := usecase.ListAdministrativeRegions(context.Background(), AdministrativeRegionQuery{Page: 1, PageSize: MaxListPageSize}); err != nil {
		t.Fatalf("ListAdministrativeRegions() error = %v", err)
	}
	if repo.query.Level != 0 || repo.query.ParentCode != nil {
		t.Fatalf("query = %#v", repo.query)
	}
}

func TestReferenceDataRejectsInvalidAdministrativeRegionQuery(t *testing.T) {
	usecase := NewReferenceDataUsecase(&referenceDataRepoStub{})
	parentCode := "310000000000"
	invalidQueries := []AdministrativeRegionQuery{
		{Level: -1, Page: 1, PageSize: 20},
		{Level: 0, ParentCode: &parentCode, Page: 1, PageSize: 20},
		{Level: 1, ParentCode: &parentCode, Page: 1, PageSize: 20},
		{Level: 2, Page: 1, PageSize: 20},
		{Level: 3, ParentCode: stringPointer("310000"), Page: 1, PageSize: 20},
		{Level: 4, Page: 1, PageSize: 20},
		{Page: 1, PageSize: MaxListPageSize + 1},
	}
	for index, query := range invalidQueries {
		if _, err := usecase.ListAdministrativeRegions(context.Background(), query); err != ErrReferenceDataInvalidArgument {
			t.Fatalf("invalid query %d error = %v, want ErrReferenceDataInvalidArgument", index, err)
		}
	}
}

func stringPointer(value string) *string { return &value }

func TestReferenceDataSetCurrencyEnabledValidation(t *testing.T) {
	repo := &referenceDataRepoStub{}
	usecase := NewReferenceDataUsecase(repo)
	if _, err := usecase.SetCurrencyEnabled(context.Background(), uuid.New(), "   ", true); err != ErrReferenceDataInvalidArgument {
		t.Fatalf("SetCurrencyEnabled() empty code error = %v, want ErrReferenceDataInvalidArgument", err)
	}
}

func TestReferenceDataAdministrativeRegionFullListQuery(t *testing.T) {
	repo := &referenceDataRepoStub{}
	usecase := NewReferenceDataUsecase(repo)
	// 维护页整表加载：0/0 透传仓储层，不得被缺省规则改写
	if _, err := usecase.ListAdministrativeRegions(context.Background(), AdministrativeRegionQuery{}); err != nil {
		t.Fatalf("full-list query error = %v", err)
	}
	if repo.query.Page != 0 || repo.query.PageSize != 0 {
		t.Fatalf("full-list query = %#v, want zero Page/PageSize", repo.query)
	}
	if _, err := usecase.ListAdministrativeRegions(context.Background(), AdministrativeRegionQuery{Page: 1}); err != ErrReferenceDataInvalidArgument {
		t.Fatalf("page-only query error = %v, want ErrReferenceDataInvalidArgument", err)
	}
	if _, err := usecase.ListAdministrativeRegions(context.Background(), AdministrativeRegionQuery{PageSize: MaxListPageSize}); err != ErrReferenceDataInvalidArgument {
		t.Fatalf("pageSize-only query error = %v, want ErrReferenceDataInvalidArgument", err)
	}
}

var _ ReferenceDataRepo = (*referenceDataRepoStub)(nil)
