package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerSettlementRuleRepoStub struct {
	created *PartnerSettlementRule
	updated *PartnerSettlementRule
}

func (s *partnerSettlementRuleRepoStub) List(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType) ([]*PartnerSettlementRule, error) {
	return nil, nil
}

func (s *partnerSettlementRuleRepoStub) Create(_ context.Context, _, _ uuid.UUID, _ PartnerRoleType, input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	s.created = input
	input.ID = uuid.New()
	return input, nil
}

func (s *partnerSettlementRuleRepoStub) Update(_ context.Context, _, _ uuid.UUID, _ PartnerRoleType, id uuid.UUID, input *PartnerSettlementRule) (*PartnerSettlementRule, error) {
	s.updated = input
	input.ID = id
	return input, nil
}

func TestPartnerSettlementRuleNormalizesAndAudits(t *testing.T) {
	repo := &partnerSettlementRuleRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewPartnerSettlementRuleUsecase(repo, audit)
	base := PartnerSettlementBillDate
	day := 15

	created, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &PartnerSettlementRule{
		StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementMonthly,
		SettlementDay: &day, SettlementBase: &base, SettlementCurrency: " cny ", IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.SettlementCurrency != "CNY" || created.SettlementDay == nil || *created.SettlementDay != 15 || len(audit.events) != 1 || audit.events[0].Action != "partner.settlement_rule.create" {
		t.Fatalf("created rule = %#v, audit = %#v", created, audit.events)
	}
}

func TestPartnerSettlementRuleRejectsInconsistentSchedule(t *testing.T) {
	usecase := NewPartnerSettlementRuleUsecase(&partnerSettlementRuleRepoStub{}, &auditRepoStub{})
	day := 1
	base := PartnerSettlementBillDate
	cases := []PartnerSettlementRule{
		{StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementByTicket, SettlementDay: &day, SettlementCurrency: "CNY"},
		{StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementMonthly, SettlementCurrency: "CNY"},
		{StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementMonthly, SettlementDay: &day, SettlementBase: &base, SettlementCurrency: "CNY"},
	}
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &cases[0]); err != ErrPartnerSettlementRuleInvalidArgument {
		t.Fatalf("by-ticket schedule error = %v", err)
	}
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &cases[1]); err != ErrPartnerSettlementRuleInvalidArgument {
		t.Fatalf("missing base error = %v", err)
	}
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &cases[2]); err != nil {
		t.Fatalf("valid monthly schedule error = %v", err)
	}
}

var _ PartnerSettlementRuleRepo = (*partnerSettlementRuleRepoStub)(nil)
