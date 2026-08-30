package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type partnerContractRepoStub struct {
	existing       *PartnerContract
	created        *PartnerContract
	updated        *PartnerContract
	expectedStatus PartnerContractStatus
	audit          *AuditEvent
}

func (s *partnerContractRepoStub) List(context.Context, uuid.UUID, uuid.UUID, *PartnerContractStatus) ([]*PartnerContract, error) {
	return nil, nil
}

func (s *partnerContractRepoStub) Get(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*PartnerContract, error) {
	if s.existing == nil {
		return nil, ErrPartnerContractNotFound
	}
	return s.existing, nil
}

func (s *partnerContractRepoStub) Create(_ context.Context, _, partnerID uuid.UUID, input *PartnerContract, audit *AuditEvent) (*PartnerContract, error) {
	s.created = input
	input.ID = uuid.New()
	input.PartnerID = partnerID
	s.audit = audit
	return input, nil
}

func (s *partnerContractRepoStub) Update(_ context.Context, _, partnerID, id uuid.UUID, expectedStatus PartnerContractStatus, input *PartnerContract, audit *AuditEvent) (*PartnerContract, error) {
	s.updated = input
	s.expectedStatus = expectedStatus
	input.ID = id
	input.PartnerID = partnerID
	input.ContractNo = s.existing.ContractNo
	s.audit = audit
	return input, nil
}

func TestPartnerContractRejectsInvalidDateAndTransition(t *testing.T) {
	start := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	usecase := NewPartnerContractUsecase(&partnerContractRepoStub{})
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), &PartnerContract{
		ContractNo: "HT-001", Name: "年度合同", Status: PartnerContractPending, StartDate: start, EndDate: start,
	}); err != ErrPartnerContractInvalidArgument {
		t.Fatalf("reversed date error = %v, want ErrPartnerContractInvalidArgument", err)
	}

	repo := &partnerContractRepoStub{existing: &PartnerContract{Status: PartnerContractExpired, ContractNo: "HT-001"}}
	usecase = NewPartnerContractUsecase(repo)
	if _, err := usecase.Update(context.Background(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), &PartnerContract{
		Name: "年度合同", Status: PartnerContractActive, StartDate: start, EndDate: start.AddDate(1, 0, 0),
	}); err != ErrPartnerContractStatusConflict {
		t.Fatalf("terminal transition error = %v, want ErrPartnerContractStatusConflict", err)
	}
}

func TestPartnerContractAllowedStatuses(t *testing.T) {
	tests := []struct {
		status PartnerContractStatus
		want   []PartnerContractStatus
	}{
		{status: PartnerContractPending, want: []PartnerContractStatus{PartnerContractPending, PartnerContractActive, PartnerContractTerminated}},
		{status: PartnerContractActive, want: []PartnerContractStatus{PartnerContractActive, PartnerContractExpired, PartnerContractTerminated}},
		{status: PartnerContractExpired, want: []PartnerContractStatus{PartnerContractExpired}},
		{status: PartnerContractTerminated, want: []PartnerContractStatus{PartnerContractTerminated}},
	}
	for _, tt := range tests {
		statuses := (&PartnerContract{Status: tt.status}).AllowedStatuses()
		if len(statuses) != len(tt.want) {
			t.Fatalf("状态 %q 允许状态 = %v，期望 %v", tt.status, statuses, tt.want)
		}
		for index := range tt.want {
			if statuses[index] != tt.want[index] {
				t.Fatalf("状态 %q 允许状态 = %v，期望 %v", tt.status, statuses, tt.want)
			}
		}
	}
}

func TestPartnerContractUpdateUsesExpectedStatusAndAudits(t *testing.T) {
	start := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	contractID := uuid.New()
	partnerID := uuid.New()
	repo := &partnerContractRepoStub{existing: &PartnerContract{ID: contractID, PartnerID: partnerID, ContractNo: "HT-001", Status: PartnerContractPending}}
	usecase := NewPartnerContractUsecase(repo)

	updated, err := usecase.Update(context.Background(), uuid.New(), uuid.New(), partnerID, contractID, &PartnerContract{
		Name: " 年度合同 ", Status: PartnerContractActive, StartDate: start, EndDate: start.AddDate(1, 0, 0),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.expectedStatus != PartnerContractPending || updated.ContractNo != "HT-001" || updated.Name != "年度合同" {
		t.Fatalf("updated contract = %#v, expected status = %q", updated, repo.expectedStatus)
	}
	if repo.audit == nil || repo.audit.Action != "partner.contract.update" || repo.audit.Details["contract.id"] != contractID.String() {
		t.Fatalf("audit event = %#v", repo.audit)
	}
}

var _ PartnerContractRepo = (*partnerContractRepoStub)(nil)
