package biz

import (
	"context"
	"testing"
)

type referenceDataRepoStub struct {
	query           AdministrativeRegionQuery
	currencyOptions SelectorListOptions
}

func (stub *referenceDataRepoStub) SearchCurrencies(_ context.Context, options SelectorListOptions) (*PagedList[*Currency], error) {
	stub.currencyOptions = options
	return &PagedList[*Currency]{Page: options.Page, PageSize: options.PageSize}, nil
}

func (stub *referenceDataRepoStub) ListCurrencies(context.Context) ([]*Currency, error) {
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

var _ ReferenceDataRepo = (*referenceDataRepoStub)(nil)
