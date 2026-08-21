package biz

import (
	"context"
	"testing"
)

type referenceDataRepoStub struct {
	query AdministrativeRegionQuery
}

func (stub *referenceDataRepoStub) ListCurrencies(context.Context) ([]*Currency, error) {
	return nil, nil
}

func (stub *referenceDataRepoStub) ListAdministrativeRegions(_ context.Context, query AdministrativeRegionQuery) ([]*AdministrativeRegion, error) {
	stub.query = query
	return nil, nil
}

func TestReferenceDataAdministrativeRegionQuery(t *testing.T) {
	repo := &referenceDataRepoStub{}
	usecase := NewReferenceDataUsecase(repo)
	parentCode := " 310000000000 "
	if _, err := usecase.ListAdministrativeRegions(context.Background(), AdministrativeRegionQuery{
		Level: 2, ParentCode: &parentCode, Keyword: " 上海 ",
	}); err != nil {
		t.Fatalf("ListAdministrativeRegions() error = %v", err)
	}
	if repo.query.ParentCode == nil || *repo.query.ParentCode != "310000000000" || repo.query.Keyword != "上海" {
		t.Fatalf("normalized query = %#v", repo.query)
	}
}

func TestReferenceDataRejectsInvalidAdministrativeRegionQuery(t *testing.T) {
	usecase := NewReferenceDataUsecase(&referenceDataRepoStub{})
	parentCode := "310000000000"
	invalidQueries := []AdministrativeRegionQuery{
		{Level: 0},
		{Level: 1, ParentCode: &parentCode},
		{Level: 2},
		{Level: 3, ParentCode: stringPointer("310000")},
	}
	for index, query := range invalidQueries {
		if _, err := usecase.ListAdministrativeRegions(context.Background(), query); err != ErrReferenceDataInvalidArgument {
			t.Fatalf("invalid query %d error = %v, want ErrReferenceDataInvalidArgument", index, err)
		}
	}
}

func stringPointer(value string) *string { return &value }

var _ ReferenceDataRepo = (*referenceDataRepoStub)(nil)
