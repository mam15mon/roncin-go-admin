package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type masterDataRepoStub struct {
	created *MasterDataItem
	updated *MasterDataItem
}

func (s *masterDataRepoStub) List(_ context.Context, _ uuid.UUID, options MasterDataListOptions) (*MasterDataList, error) {
	return &MasterDataList{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *masterDataRepoStub) ListEnabled(context.Context, uuid.UUID) ([]*MasterDataItem, error) {
	return nil, nil
}

func (s *masterDataRepoStub) Create(_ context.Context, organizationID uuid.UUID, input *MasterDataItem) (*MasterDataItem, error) {
	s.created = input
	input.OrganizationID = organizationID
	input.ID = uuid.New()
	return input, nil
}

func (s *masterDataRepoStub) Update(_ context.Context, organizationID, id uuid.UUID, input *MasterDataItem) (*MasterDataItem, error) {
	s.updated = input
	input.OrganizationID = organizationID
	input.ID = id
	input.Code = "40HC"
	return input, nil
}

func TestMasterDataCreateNormalizesAndAudits(t *testing.T) {
	repo := &masterDataRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewMasterDataUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{Kind: MasterDataKindCurrency, Code: " cny ", Name: " 人民币 ", NameEN: stringPtr(" Renminbi "), Source: " ", SortOrder: 10})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "CNY" || created.Name != "人民币" || created.NameEN == nil || *created.NameEN != "Renminbi" || created.Source != "manual" {
		t.Fatalf("normalized item = %#v", created)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "master_data.create" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestMasterDataRejectsInvalidTEUFactor(t *testing.T) {
	usecase := NewMasterDataUsecase(&masterDataRepoStub{}, &auditRepoStub{})
	organizationID := uuid.New()
	actorID := uuid.New()

	if _, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{Kind: MasterDataKindCurrency, Code: "CNY", Name: "人民币", TEUFactor: stringPtr("1")}); err != ErrMasterDataInvalidArgument {
		t.Fatalf("currency TEU factor error = %v, want ErrMasterDataInvalidArgument", err)
	}
	if _, err := usecase.Create(context.Background(), organizationID, actorID, &MasterDataItem{Kind: MasterDataKindContainerSpec, Code: "40HC", Name: "40尺高柜", TEUFactor: stringPtr("not-number")}); err != ErrMasterDataInvalidArgument {
		t.Fatalf("invalid TEU factor error = %v, want ErrMasterDataInvalidArgument", err)
	}
}

func TestMasterDataListRejectsInvalidPage(t *testing.T) {
	usecase := NewMasterDataUsecase(&masterDataRepoStub{}, &auditRepoStub{})
	if _, err := usecase.List(context.Background(), uuid.New(), MasterDataListOptions{Page: 0, PageSize: 20}); err != ErrMasterDataInvalidArgument {
		t.Fatalf("List() error = %v, want ErrMasterDataInvalidArgument", err)
	}
}

var _ MasterDataRepo = (*masterDataRepoStub)(nil)
