package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	ruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
)

func TestPartnerPaymentTermsContractPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newCreditPartnerFixture(t, d)
	partnerID, roleID := f.newCreditPartner("单条约定")
	actorID := uuid.New()
	repo := NewPartnerSettlementRuleRepo(d)
	uc := biz.NewPartnerSettlementRuleUsecase(repo)
	amount, currency := int64(50000000), "CNY"
	created, err := uc.Create(t.Context(), f.organizationID, actorID, partnerID, biz.PartnerRoleCustomer, &biz.PartnerSettlementRule{PaymentTermsDays: new(30), CreditLimitMinor: &amount, CreditCurrency: &currency})
	if err != nil {
		t.Fatal(err)
	}
	beforeAudits, err := d.db.AuditLog.Query().Count(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Create(t.Context(), f.organizationID, actorID, partnerID, biz.PartnerRoleCustomer, &biz.PartnerSettlementRule{PaymentTermsDays: new(99)}); !errors.Is(err, biz.ErrPartnerSettlementRuleExists) {
		t.Fatalf("重复新建须映射明确领域错误: %v", err)
	}
	var constraint *ent.ConstraintError
	if _, err := d.db.PartnerSettlementRule.Create().SetPartnerRoleID(roleID).Save(t.Context()); !errors.As(err, &constraint) {
		t.Fatalf("绕过用例仍须由数据库唯一索引拒绝: %v", err)
	}
	if count, err := d.db.AuditLog.Query().Count(t.Context()); err != nil || count != beforeAudits {
		t.Fatalf("拒绝重复创建不能写成功审计: %d %v", count, err)
	}
	for _, days := range []*int{nil, new(0), new(3650), new(30)} {
		updated, err := uc.Update(t.Context(), f.organizationID, actorID, partnerID, created.ID, biz.PartnerRoleCustomer, &biz.PartnerSettlementRule{PaymentTermsDays: days})
		if err != nil {
			t.Fatal(err)
		}
		stored, err := d.db.PartnerSettlementRule.Get(t.Context(), created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if (updated.PaymentTermsDays == nil) != (days == nil) || (days != nil && *updated.PaymentTermsDays != *days) || (stored.PaymentTermsDays == nil) != (days == nil) || (days != nil && *stored.PaymentTermsDays != *days) {
			t.Fatalf("账期NULL/0/N存在性必须端到端保留: updated=%+v stored=%+v want=%v", updated, stored, days)
		}
		if stored.CreditLimitMinor != nil || stored.CreditCurrency != nil {
			t.Fatalf("空额度同时清空额度与币种: %+v", stored)
		}
		summary, err := f.repo.GetPartnerCreditSummaries(t.Context(), f.organizationID, []uuid.UUID{partnerID})
		if err != nil {
			t.Fatal(err)
		}
		got := summary[partnerID]
		if (got.DefaultPaymentTermsDays == nil) != (days == nil) || (days != nil && *got.DefaultPaymentTermsDays != *days) {
			t.Fatalf("信用汇总必须读取同一条约定账期: %+v", got)
		}
	}
	if count, err := d.db.PartnerSettlementRule.Query().Where(ruleent.PartnerRoleIDEQ(roleID)).Count(t.Context()); err != nil || count != 1 {
		t.Fatalf("更新保持一条约定: %d %v", count, err)
	}
}

func TestPartnerPaymentTermsRejectNonCustomerAtomicallyPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newCreditPartnerFixture(t, d)
	repo := NewPartnerRepo(d)
	ruleRepo := NewPartnerSettlementRuleRepo(d)
	ruleUC := biz.NewPartnerSettlementRuleUsecase(ruleRepo)
	aggregateUC := biz.NewPartnerUsecase(repo)
	access := biz.PartnerSettlementRuleWriteAccess{Create: true, Update: true}
	for _, roleType := range []biz.PartnerRoleType{biz.PartnerRoleSupplier, biz.PartnerRoleForeignAgent} {
		t.Run(string(roleType), func(t *testing.T) {
			partnerID, roleID := f.newCreditPartner("拒绝非客户" + string(roleType))
			if err := d.db.PartnerRole.UpdateOneID(roleID).SetRoleType(roleent.RoleType(roleType)).Exec(t.Context()); err != nil {
				t.Fatal(err)
			}
			actorID := uuid.New()
			beforePartner := d.db.Partner.GetX(t.Context(), partnerID)
			beforeAudits := d.db.AuditLog.Query().CountX(t.Context())
			beforeRules := d.db.PartnerSettlementRule.Query().CountX(t.Context())
			input := &biz.PartnerSettlementRule{PaymentTermsDays: new(30)}
			audit := func() *biz.AuditEvent {
				return &biz.AuditEvent{OrganizationID: &f.organizationID, Action: "partner.settlement_rule.test", Result: "success", Details: map[string]string{}}
			}
			for _, call := range []func() error{
				func() error {
					_, err := ruleUC.Create(t.Context(), f.organizationID, actorID, partnerID, roleType, input)
					return err
				},
				func() error {
					_, err := ruleUC.Update(t.Context(), f.organizationID, actorID, partnerID, uuid.New(), roleType, input)
					return err
				},
				func() error {
					_, err := ruleRepo.Create(t.Context(), f.organizationID, partnerID, roleType, input, audit())
					return err
				},
				func() error {
					_, err := ruleRepo.Update(t.Context(), f.organizationID, partnerID, roleType, uuid.New(), input, audit())
					return err
				},
			} {
				if err := call(); !errors.Is(err, biz.ErrPartnerSettlementRuleInvalidArgument) {
					t.Fatalf("非客户独立写必须参数错误: %v", err)
				}
			}
			invalid := &biz.Partner{Code: "REJECT-" + string(roleType), LegalName: "非法内嵌约定" + string(roleType), NormalizedName: "非法内嵌约定" + string(roleType), Enabled: true, Roles: []*biz.PartnerRole{{Type: roleType, Enabled: true, SettlementRule: input}}}
			for _, call := range []func() error{
				func() error {
					_, err := aggregateUC.Create(t.Context(), f.organizationID, actorID, invalid, access)
					return err
				},
				func() error {
					_, err := aggregateUC.Update(t.Context(), f.organizationID, actorID, partnerID, invalid, access)
					return err
				},
				func() error {
					_, err := aggregateUC.Import(t.Context(), f.organizationID, actorID, biz.PartnerImportInput{Source: "payment-terms-test", Mode: biz.PartnerImportUpsert, Items: []*biz.Partner{invalid}}, access)
					return err
				},
				func() error {
					_, err := repo.Create(t.Context(), f.organizationID, invalid, access, audit())
					return err
				},
				func() error {
					_, err := repo.Update(t.Context(), f.organizationID, partnerID, invalid, access, audit())
					return err
				},
				func() error {
					_, err := repo.Import(t.Context(), f.organizationID, biz.PartnerImportUpsert, []*biz.Partner{invalid}, access, audit())
					return err
				},
			} {
				if err := call(); !errors.Is(err, biz.ErrPartnerSettlementRuleInvalidArgument) {
					t.Fatalf("非客户内嵌写必须参数错误: %v", err)
				}
			}
			afterPartner := d.db.Partner.GetX(t.Context(), partnerID)
			if afterPartner.LegalName != beforePartner.LegalName || !afterPartner.UpdatedAt.Equal(beforePartner.UpdatedAt) || d.db.PartnerSettlementRule.Query().CountX(t.Context()) != beforeRules || d.db.AuditLog.Query().CountX(t.Context()) != beforeAudits {
				t.Fatal("非客户拒绝不得写档案、约定或成功审计")
			}
			if count := d.db.Partner.Query().Where(partnerent.CodeEQ(invalid.Code)).CountX(t.Context()); count != 0 {
				t.Fatalf("拒绝创建及导入不得留下新档案: %d", count)
			}
		})
	}
}

func TestPartnerPaymentTermsBillPreviewMatchesCurrentAgreementPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newFinanceBillBatchPostgresFixture(t, d)
	role, err := d.db.PartnerRole.Create().SetPartnerID(f.partnerID).SetRoleType(roleent.RoleTypeCustomer).SetEnabled(true).Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := d.db.PartnerSettlementRule.Delete().Where(ruleent.PartnerRoleIDEQ(role.ID)).Exec(context.Background()); err != nil {
			t.Error(err)
		}
		if err := d.db.PartnerRole.DeleteOneID(role.ID).Exec(context.Background()); err != nil {
			t.Error(err)
		}
	})
	ruleUC := biz.NewPartnerSettlementRuleUsecase(NewPartnerSettlementRuleRepo(d))
	created, err := ruleUC.Create(t.Context(), f.organizationID, f.actorID, f.partnerID, biz.PartnerRoleCustomer, &biz.PartnerSettlementRule{PaymentTermsDays: new(30)})
	if err != nil {
		t.Fatal(err)
	}
	feeID := f.createBatchUnbilledFee("payment-terms")
	billUC := f.newUsecase(NewFinanceBillRepo(d))
	preview := func(want *int) {
		t.Helper()
		got, err := billUC.PreviewBatch(t.Context(), f.organizationID, biz.PreviewFinanceBillBatchInput{FeeIDs: []uuid.UUID{feeID}, GroupingPolicy: biz.FinanceBillGroupingPolicy{Mode: "NORMAL"}})
		if err != nil || len(got.Groups) != 1 {
			t.Fatalf("账单预览失败: %+v %v", got, err)
		}
		days := got.Groups[0].DefaultPaymentTermsDays
		if (days == nil) != (want == nil) || (want != nil && *days != *want) {
			t.Fatalf("预览必须读取当前客户约定: got=%v want=%v", days, want)
		}
	}
	preview(new(30))
	for _, days := range []*int{nil, new(0), new(45)} {
		if _, err := ruleUC.Update(t.Context(), f.organizationID, f.actorID, f.partnerID, created.ID, biz.PartnerRoleCustomer, &biz.PartnerSettlementRule{PaymentTermsDays: days}); err != nil {
			t.Fatal(err)
		}
		preview(days)
	}
	if err := d.db.Partner.UpdateOneID(f.partnerID).SetIsCasual(true).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	preview(new(0))
}
