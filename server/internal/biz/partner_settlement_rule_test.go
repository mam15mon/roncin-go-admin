package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type partnerSettlementRuleRepoStub struct {
	created *PartnerSettlementRule
	updated *PartnerSettlementRule
	audit   *AuditEvent
}

func (s *partnerSettlementRuleRepoStub) List(context.Context, uuid.UUID, uuid.UUID, PartnerRoleType) ([]*PartnerSettlementRule, error) {
	return nil, nil
}

func (s *partnerSettlementRuleRepoStub) Create(_ context.Context, _, _ uuid.UUID, _ PartnerRoleType, input *PartnerSettlementRule, audit *AuditEvent) (*PartnerSettlementRule, error) {
	s.created = input
	input.ID = uuid.New()
	s.audit = audit
	return input, nil
}

func (s *partnerSettlementRuleRepoStub) Update(_ context.Context, _, _ uuid.UUID, _ PartnerRoleType, id uuid.UUID, input *PartnerSettlementRule, audit *AuditEvent) (*PartnerSettlementRule, error) {
	s.updated = input
	input.ID = id
	s.audit = audit
	return input, nil
}

func TestPartnerSettlementRuleNormalizesAndAudits(t *testing.T) {
	repo := &partnerSettlementRuleRepoStub{}
	usecase := NewPartnerSettlementRuleUsecase(repo)
	base := PartnerSettlementBillDate
	day := 15

	created, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &PartnerSettlementRule{
		StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementMonthly,
		SettlementDay: &day, SettlementBase: &base, SettlementCurrency: " cny ", IsActive: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.SettlementCurrency != "CNY" || created.SettlementDay == nil || *created.SettlementDay != 15 || repo.audit == nil || repo.audit.Action != "partner.settlement_rule.create" {
		t.Fatalf("created rule = %#v, audit = %#v", created, repo.audit)
	}
}

func TestPartnerSettlementRuleRejectsInconsistentSchedule(t *testing.T) {
	usecase := NewPartnerSettlementRuleUsecase(&partnerSettlementRuleRepoStub{})
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

func TestPartnerSettlementRuleValidatesCreditLimitCurrencyPair(t *testing.T) {
	usecase := NewPartnerSettlementRuleUsecase(&partnerSettlementRuleRepoStub{})
	creditLimit := int64(100000)
	creditCurrency := " usd "
	input := PartnerSettlementRule{
		StatementMode: PartnerStatementSingle, SettlementMethod: PartnerSettlementByTicket,
		SettlementCurrency: "cny", CreditLimitMinor: &creditLimit, CreditCurrency: &creditCurrency,
	}
	created, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.CreditCurrency == nil || *created.CreditCurrency != "USD" || created.CreditLimitMinor == nil || *created.CreditLimitMinor != creditLimit {
		t.Fatalf("normalized credit rule = %#v", created)
	}

	missingCurrency := input
	missingCurrency.CreditCurrency = nil
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &missingCurrency); err != ErrPartnerSettlementRuleInvalidArgument {
		t.Fatalf("missing credit currency error = %v", err)
	}

	negativeLimit := int64(-1)
	invalidLimit := input
	invalidLimit.CreditLimitMinor = &negativeLimit
	if _, err := usecase.Create(context.Background(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &invalidLimit); err != ErrPartnerSettlementRuleInvalidArgument {
		t.Fatalf("negative credit limit error = %v", err)
	}
}

var _ PartnerSettlementRuleRepo = (*partnerSettlementRuleRepoStub)(nil)
