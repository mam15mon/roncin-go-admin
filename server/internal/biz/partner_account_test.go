package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerAccountRepoStub struct {
	created *PartnerAccount
	updated *PartnerAccount
}

func (s *partnerAccountRepoStub) List(context.Context, uuid.UUID, uuid.UUID, *bool) ([]*PartnerAccount, error) {
	return nil, nil
}

func (s *partnerAccountRepoStub) Create(_ context.Context, _, _ uuid.UUID, input *PartnerAccount) (*PartnerAccount, error) {
	s.created = input
	input.ID = uuid.New()
	return input, nil
}

func (s *partnerAccountRepoStub) Update(_ context.Context, _, _, id uuid.UUID, input *PartnerAccount) (*PartnerAccount, error) {
	s.updated = input
	input.ID = id
	return input, nil
}

func TestPartnerAccountCreateNormalizesAndAudits(t *testing.T) {
	repo := &partnerAccountRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewPartnerAccountUsecase(repo, audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	partnerID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, partnerID, &PartnerAccount{
		Currency: " cny ", InvoiceTitle: " 上海安可物流 ", UnifiedSocialCreditCode: "91310000ma1fl7a21q",
		SwiftCode: " bocccnbj ", Status: PartnerAccountActive, IsDefault: true, Remark: "  月结  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Currency != "CNY" || created.InvoiceTitle != "上海安可物流" || created.UnifiedSocialCreditCode != "91310000MA1FL7A21Q" || created.SwiftCode != "BOCCCNBJ" || created.Remark != "月结" {
		t.Fatalf("normalized account = %#v", created)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "partner.account.create" || audit.events[0].Details["partner.id"] != partnerID.String() {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestPartnerAccountRejectsInactiveDefault(t *testing.T) {
	usecase := NewPartnerAccountUsecase(&partnerAccountRepoStub{}, &auditRepoStub{})
	_, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), &PartnerAccount{
		Currency: "CNY", InvoiceTitle: "上海安可物流", Status: PartnerAccountInactive, IsDefault: true,
	})
	if err != ErrPartnerAccountInvalidArgument {
		t.Fatalf("Create() error = %v, want ErrPartnerAccountInvalidArgument", err)
	}
}

var _ PartnerAccountRepo = (*partnerAccountRepoStub)(nil)
