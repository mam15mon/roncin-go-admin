package biz

import (
	"context"
	"fmt"
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
	currency, amount, days := " cny ", int64(100000), 30
	created, err := usecase.Create(t.Context(), uuid.New(), uuid.New(), uuid.New(), PartnerRoleCustomer, &PartnerSettlementRule{CreditLimitMinor: &amount, CreditCurrency: &currency, PaymentTermsDays: &days})
	if err != nil || created.CreditCurrency == nil || *created.CreditCurrency != "CNY" || created.PaymentTermsDays == nil || *created.PaymentTermsDays != days || repo.audit == nil || repo.audit.Action != "partner.settlement_rule.create" {
		t.Fatalf("规范化或创建审计错误: %#v, %#v, %v", created, repo.audit, err)
	}
	updated, err := usecase.Update(t.Context(), uuid.New(), uuid.New(), uuid.New(), created.ID, PartnerRoleCustomer, &PartnerSettlementRule{})
	if err != nil || updated.PaymentTermsDays != nil || updated.CreditLimitMinor != nil || updated.CreditCurrency != nil || repo.audit.Action != "partner.settlement_rule.update" {
		t.Fatalf("未约定应保留空值并写更新审计: %#v %v", updated, err)
	}
}

func TestPartnerSettlementRuleRejectsNonCustomerBeforeRepository(t *testing.T) {
	for _, role := range []PartnerRoleType{PartnerRoleSupplier, PartnerRoleForeignAgent} {
		t.Run(string(role), func(t *testing.T) {
			repo := &partnerSettlementRuleRepoStub{}
			usecase := NewPartnerSettlementRuleUsecase(repo)
			if _, err := usecase.Create(t.Context(), uuid.New(), uuid.New(), uuid.New(), role, &PartnerSettlementRule{}); err != ErrPartnerSettlementRuleInvalidArgument {
				t.Fatalf("非客户创建必须拒绝: %v", err)
			}
			if _, err := usecase.Update(t.Context(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), role, &PartnerSettlementRule{}); err != ErrPartnerSettlementRuleInvalidArgument {
				t.Fatalf("非客户更新必须拒绝: %v", err)
			}
			if repo.created != nil || repo.updated != nil || repo.audit != nil {
				t.Fatal("非客户不得调用仓储或写审计")
			}
		})
	}
}

func TestPartnerSettlementRuleValidatesCreditLimitCurrencyPair(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input PartnerSettlementRule
		valid bool
	}{
		{"未设置", PartnerSettlementRule{}, true},
		{"零额度本币", PartnerSettlementRule{CreditLimitMinor: new(int64(0)), CreditCurrency: new("CNY")}, true},
		{"正额度本币", PartnerSettlementRule{CreditLimitMinor: new(int64(100)), CreditCurrency: new("USD")}, true},
		{"额度缺币种", PartnerSettlementRule{CreditLimitMinor: new(int64(100))}, false},
		{"币种缺额度", PartnerSettlementRule{CreditCurrency: new("CNY")}, false},
		{"币种长度非法", PartnerSettlementRule{CreditLimitMinor: new(int64(100)), CreditCurrency: new("CN")}, false},
		{"负额度", PartnerSettlementRule{CreditLimitMinor: new(int64(-1)), CreditCurrency: new("CNY")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := normalizePartnerSettlementRule(&tc.input)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestPartnerSettlementRuleValidatesPaymentTermsDays(t *testing.T) {
	for _, days := range []int{-1, 0, 30, 3650, 3651} {
		t.Run(fmt.Sprint(days), func(t *testing.T) {
			got, err := normalizePartnerSettlementRule(&PartnerSettlementRule{PaymentTermsDays: &days})
			valid := days >= 0 && days <= 3650
			if (err == nil) != valid {
				t.Fatalf("账期%d合法性错误: %v", days, err)
			}
			if valid && (got.PaymentTermsDays == nil || *got.PaymentTermsDays != days) {
				t.Fatalf("账期存在性或取值丢失: %#v", got)
			}
		})
	}
	if got, err := normalizePartnerSettlementRule(&PartnerSettlementRule{}); err != nil || got.PaymentTermsDays != nil {
		t.Fatalf("未约定不得变为现结: %#v %v", got, err)
	}
}

var _ PartnerSettlementRuleRepo = (*partnerSettlementRuleRepoStub)(nil)
