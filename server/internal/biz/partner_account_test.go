package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerAccountRepoStub struct {
	created *PartnerAccount
	updated *PartnerAccount
	audit   *AuditEvent
}

func (s *partnerAccountRepoStub) List(context.Context, uuid.UUID, uuid.UUID, *bool) ([]*PartnerAccount, error) {
	return nil, nil
}

func (s *partnerAccountRepoStub) Create(_ context.Context, _, _ uuid.UUID, input *PartnerAccount, audit *AuditEvent) (*PartnerAccount, error) {
	s.created = input
	input.ID = uuid.New()
	audit.Details["account.id"] = input.ID.String()
	s.audit = audit
	return input, nil
}

func (s *partnerAccountRepoStub) Update(_ context.Context, _, _, id uuid.UUID, input *PartnerAccount, audit *AuditEvent) (*PartnerAccount, error) {
	s.updated = input
	input.ID = id
	s.audit = audit
	return input, nil
}

func TestPartnerAccountCreateNormalizesAndAudits(t *testing.T) {
	repo := &partnerAccountRepoStub{}
	usecase := NewPartnerAccountUsecase(repo)
	organizationID := uuid.New()
	actorID := uuid.New()
	partnerID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, partnerID, &PartnerAccount{
		Currency: " cny ", BankName: " 中国银行 ", BankAccount: " 62220000 ",
		SwiftCode: " bocccnbj ", Status: PartnerAccountActive, IsDefault: true, Remark: "  月结  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Currency != "CNY" || created.BankName != "中国银行" || created.BankAccount != "62220000" || created.SwiftCode != "BOCCCNBJ" || created.Remark != "月结" {
		t.Fatalf("normalized account = %#v", created)
	}
	if repo.audit == nil || repo.audit.Action != "partner.account.create" || repo.audit.Details["partner.id"] != partnerID.String() {
		t.Fatalf("audit event = %#v", repo.audit)
	}
}

func TestPartnerAccountRejectsInactiveDefault(t *testing.T) {
	usecase := NewPartnerAccountUsecase(&partnerAccountRepoStub{})
	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), &PartnerAccount{
		Currency: "CNY", Status: PartnerAccountInactive, IsDefault: true,
	})
	if err != ErrPartnerAccountInvalidArgument {
		t.Fatalf("Create() error = %v, want ErrPartnerAccountInvalidArgument", err)
	}
}

var _ PartnerAccountRepo = (*partnerAccountRepoStub)(nil)
